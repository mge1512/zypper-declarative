// generated from spec: zypper-declarative.spec.md sha256:f2cc80627e483a48bb8411d297711bc5f6c6e74c28dbf0dafc8fe7bd8817251e
//
// describe.cpp -- describe-actual-state. The single live-state reader. Reads
// the four declarable scopes under a root and returns a Manifest in the shared
// schema. The /etc walk classifies each entry by its own type without following
// symlinks (lstat): regular files hashed, symlinks recorded verbatim, dirs
// traversed (not emitted), special files skipped. config_files is bounded to
// /etc. Under scope=full the package-managed trees outside /etc are scanned.
#include "describe.hpp"
#include "hashing.hpp"

#include <sys/stat.h>
#include <unistd.h>
#include <dirent.h>
#include <pwd.h>
#include <grp.h>
#include <filesystem>
#include <fstream>
#include <sstream>
#include <map>
#include <ctime>
#include <algorithm>

namespace fs = std::filesystem;

namespace zd {

static const char* kSyncpoint = "/etc/etc.syncpoint";

static std::string now_rfc3339() {
    std::time_t t = std::time(nullptr);
    std::tm tm{};
    gmtime_r(&t, &tm);
    char buf[32];
    std::strftime(buf, sizeof(buf), "%Y-%m-%dT%H:%M:%SZ", &tm);
    return buf;
}

static std::string mode_octal(mode_t m) {
    char buf[8];
    std::snprintf(buf, sizeof(buf), "%04o", static_cast<unsigned>(m & 07777));
    return buf;
}
static std::string user_name(uid_t uid) {
    struct passwd* pw = getpwuid(uid);
    return pw ? pw->pw_name : std::to_string(uid);
}
static std::string group_name(gid_t gid) {
    struct group* gr = getgrgid(gid);
    return gr ? gr->gr_name : std::to_string(gid);
}

enum class EntryKind { Regular, Symlink, Directory, Special };
static EntryKind classify(const struct stat& st) {
    if (S_ISLNK(st.st_mode)) return EntryKind::Symlink;
    if (S_ISDIR(st.st_mode)) return EntryKind::Directory;
    if (S_ISREG(st.st_mode)) return EntryKind::Regular;
    return EntryKind::Special;
}

// Read a symlink target verbatim (not resolved, not normalised).
static std::string read_link_verbatim(const std::string& path) {
    char buf[4096];
    ssize_t n = ::readlink(path.c_str(), buf, sizeof(buf) - 1);
    if (n < 0) return "";
    buf[n] = '\0';
    return std::string(buf);
}

// The collected outcome of walking a tree.
struct WalkError { bool occurred = false; std::string path; };

// Recursively walk `dir` (an absolute filesystem path). For each entry classify
// by lstat; descend into directories; emit a callback for regular files and
// symlinks. keep-list and syncpoint suppression is the caller's concern via the
// `logical_prefix` it passes to the callback (the path as it should appear in
// the manifest, i.e. starting at /etc...).
template <class FileFn, class LinkFn>
static void walk_tree(const std::string& dir, const std::string& logical_dir,
                      WalkError& werr, FileFn on_file, LinkFn on_link) {
    DIR* d = ::opendir(dir.c_str());
    if (!d) {
        werr.occurred = true; werr.path = dir;
        return;
    }
    std::vector<std::string> names;
    struct dirent* de;
    while ((de = ::readdir(d)) != nullptr) {
        std::string n = de->d_name;
        if (n == "." || n == "..") continue;
        names.push_back(n);
    }
    ::closedir(d);
    std::sort(names.begin(), names.end());
    for (auto& n : names) {
        std::string full = dir + "/" + n;
        std::string logical = logical_dir + "/" + n;
        struct stat st;
        if (::lstat(full.c_str(), &st) != 0) {
            // genuine I/O failure on a required path
            werr.occurred = true; werr.path = full;
            return;
        }
        EntryKind k = classify(st);
        if (k == EntryKind::Directory) {
            walk_tree(full, logical, werr, on_file, on_link);
            if (werr.occurred) return;
        } else if (k == EntryKind::Regular) {
            on_file(full, logical, st);
        } else if (k == EntryKind::Symlink) {
            on_link(full, logical, st);
        } else {
            // special file: skip silently
        }
    }
}

ActualStateResult describe_actual_state(const std::string& root,
                                        OnUnreadable on_unreadable,
                                        ScanScope scope,
                                        const KeepList& keep,
                                        const SystemReader* reader) {
    ActualStateResult res;
    res.manifest.meta.format_version = 1;
    res.manifest.meta.generator = "zypper-declarative";
    res.manifest.meta.created_at = now_rfc3339();
    res.manifest.meta.desired_sha256 = "";

    auto handle_unreadable = [&](const std::string& domain, const std::string& src) -> bool {
        // returns true if the run should abort (error mode)
        if (on_unreadable == OnUnreadable::Error) {
            res.ok = false;
            res.error = Diagnostic{Severity::Error, domain,
                                   "unreadable scope source: " + src};
            return true;
        } else {
            res.diagnostics.push_back(Diagnostic{Severity::Warning, domain,
                                       "unreadable scope source omitted: " + src});
            return false;
        }
    };

    auto is_suppressed = [&](const std::string& logical) {
        return logical == kSyncpoint || keep.find(logical) != keep.end();
    };

    // STEP 1: packages (via reader; without a reader, treat as empty/readable).
    if (reader) {
        bool ok = true;
        PackagesScope ps = reader->query_packages(root, ok);
        if (!ok) { if (handle_unreadable("packages", "rpmdb")) return res; }
        else if (!ps.elements.empty()) {
            ps.attributes["package_system"] = "rpm";
            res.manifest.packages = ps;
        }
    }

    // STEP 2: repositories from <root>/etc/zypp/repos.d/*.repo (INI).
    {
        fs::path reposd = fs::path(root) / "etc" / "zypp" / "repos.d";
        std::error_code ec;
        if (fs::exists(reposd, ec)) {
            DIR* d = ::opendir(reposd.c_str());
            if (!d) {
                if (handle_unreadable("repositories", reposd.string())) return res;
            } else {
                RepositoriesScope rs;
                rs.attributes["repository_system"] = "zypp";
                std::vector<std::string> files;
                struct dirent* de;
                bool read_failure = false;
                while ((de = ::readdir(d)) != nullptr) {
                    std::string n = de->d_name;
                    if (n.size() > 5 && n.substr(n.size() - 5) == ".repo")
                        files.push_back((reposd / n).string());
                }
                ::closedir(d);
                std::sort(files.begin(), files.end());
                for (auto& fp : files) {
                    std::ifstream f(fp);
                    if (!f) { read_failure = true; break; }
                    RepositoryRecord cur; bool in_section = false;
                    std::string line;
                    auto flush = [&]() {
                        if (in_section && !cur.alias.empty()) rs.elements.push_back(cur);
                    };
                    while (std::getline(f, line)) {
                        auto b = line.find_first_not_of(" \t\r\n");
                        if (b == std::string::npos) continue;
                        line = line.substr(b);
                        if (line[0] == '#' || line[0] == ';') continue;
                        if (line[0] == '[') {
                            flush();
                            cur = RepositoryRecord{};
                            in_section = true;
                            auto close = line.find(']');
                            cur.alias = (close != std::string::npos)
                                ? line.substr(1, close - 1) : "";
                            continue;
                        }
                        auto eq = line.find('=');
                        if (eq == std::string::npos) continue;
                        std::string key = line.substr(0, eq);
                        std::string val = line.substr(eq + 1);
                        auto trim = [](std::string& s) {
                            auto a = s.find_first_not_of(" \t\r\n");
                            auto z = s.find_last_not_of(" \t\r\n");
                            if (a == std::string::npos) { s.clear(); return; }
                            s = s.substr(a, z - a + 1);
                        };
                        trim(key); trim(val);
                        if (key == "name") cur.name = val;
                        else if (key == "baseurl") cur.url = val;
                        else if (key == "type") cur.type = val;
                        else if (key == "enabled") cur.enabled = (val == "1" || val == "true");
                        else if (key == "gpgcheck") cur.gpgcheck = (val == "1" || val == "true");
                        else if (key == "autorefresh") cur.autorefresh = (val == "1" || val == "true");
                        else if (key == "priority") { try { cur.priority = std::stoi(val); } catch (...) {} }
                    }
                    flush();
                }
                if (read_failure) {
                    if (handle_unreadable("repositories", reposd.string())) return res;
                } else if (!rs.elements.empty()) {
                    res.manifest.repositories = rs;   // omit if genuinely empty
                }
            }
        }
        // repos.d absent entirely: genuinely-empty -> omit (not an error).
    }

    // STEP 3: services (via reader).
    if (reader) {
        bool ok = true;
        ServicesScope ss = reader->query_services(root, ok);
        if (!ok) { if (handle_unreadable("units", "unit enablement")) return res; }
        else if (!ss.elements.empty()) {
            ss.attributes["init_system"] = "systemd";
            res.manifest.services = ss;
        }
    }

    // STEP 4: config_files -- walk <root>/etc.
    {
        fs::path etc = fs::path(root) / "etc";
        std::error_code ec;
        ConfigFilesScope cfs;
        cfs.has_attributes = false; // config_files _attributes is null
        if (fs::exists(etc, ec)) {
            WalkError werr;
            auto on_file = [&](const std::string& full, const std::string& logical_etc,
                               const struct stat& st) {
                std::string logical = "/etc" + logical_etc; // logical_etc starts with "/..."
                if (is_suppressed(logical)) return;
                bool hok = true;
                std::string digest = sha256_file(full, hok);
                if (!hok) return; // unreadable individual file handled below via werr if dir-level
                std::string pkg = reader ? reader->owning_package(root, logical) : "";
                bool changed = reader
                    ? reader->file_differs_from_baseline(root, logical, digest)
                    : true; // no reader: treat as unpackaged/changed (emit)
                if (!changed && !pkg.empty()) return; // package-pristine -> skip
                ManagedFileRecord r;
                r.name = logical; r.type = "file"; r.mode = mode_octal(st.st_mode);
                r.user = user_name(st.st_uid); r.group = group_name(st.st_gid);
                r.sha256 = digest; r.target = ""; r.content_ref = "";
                r.package_name = pkg;
                cfs.elements.push_back(r);
            };
            auto on_link = [&](const std::string& full, const std::string& logical_etc,
                               const struct stat& st) {
                std::string logical = "/etc" + logical_etc;
                if (is_suppressed(logical)) return;
                std::string target = read_link_verbatim(full);
                std::string pkg = reader ? reader->owning_package(root, logical) : "";
                bool changed = reader
                    ? reader->link_differs_from_baseline(root, logical, target)
                    : true;
                if (!changed && !pkg.empty()) return;
                ManagedFileRecord r;
                r.name = logical; r.type = "link"; r.mode = mode_octal(st.st_mode);
                r.user = user_name(st.st_uid); r.group = group_name(st.st_gid);
                r.sha256 = ""; r.target = target; r.content_ref = "";
                r.package_name = pkg;
                cfs.elements.push_back(r);
            };
            // walk; logical_dir starts empty so logical_etc becomes "/sub/file"
            walk_tree(etc.string(), "", werr, on_file, on_link);
            if (werr.occurred) {
                if (handle_unreadable("files", werr.path)) return res;
            }
            if (!cfs.elements.empty()) res.manifest.config_files = cfs;
        }
    }

    // STEP 4a: full-scan integrity (scope=full).
    if (scope == ScanScope::Full) {
        std::vector<std::string> trees = {"usr", "bin", "sbin", "lib", "lib64", "boot"};
        ChangedManagedFilesScope changed;
        UnmanagedFilesScope unmanaged;
        for (auto& tree : trees) {
            fs::path base = fs::path(root) / tree;
            std::error_code ec;
            if (!fs::exists(base, ec)) continue;
            WalkError werr;
            auto on_file = [&](const std::string& full, const std::string& logical_sub,
                               const struct stat& st) {
                std::string logical = "/" + tree + logical_sub;
                if (keep.find(logical) != keep.end()) return;
                bool hok = true;
                std::string digest = sha256_file(full, hok);
                if (!hok) return;
                std::string pkg = reader ? reader->owning_package(root, logical) : "";
                if (pkg.empty()) {
                    UnmanagedFileRecord u;
                    u.name = logical; u.type = "file"; u.mode = mode_octal(st.st_mode);
                    u.user = user_name(st.st_uid); u.group = group_name(st.st_gid);
                    u.sha256 = digest; u.target = "";
                    unmanaged.elements.push_back(u);
                } else if (reader && reader->file_differs_from_baseline(root, logical, digest)) {
                    ManagedBaselineRecord b;
                    b.name = logical; b.type = "file"; b.mode = mode_octal(st.st_mode);
                    b.user = user_name(st.st_uid); b.group = group_name(st.st_gid);
                    b.sha256 = digest; b.target = ""; b.package_name = pkg;
                    b.changes = {"sha256"};
                    changed.elements.push_back(b);
                }
            };
            auto on_link = [&](const std::string& full, const std::string& logical_sub,
                               const struct stat& st) {
                std::string logical = "/" + tree + logical_sub;
                if (keep.find(logical) != keep.end()) return;
                std::string target = read_link_verbatim(full);
                std::string pkg = reader ? reader->owning_package(root, logical) : "";
                if (pkg.empty()) {
                    UnmanagedFileRecord u;
                    u.name = logical; u.type = "link"; u.mode = mode_octal(st.st_mode);
                    u.user = user_name(st.st_uid); u.group = group_name(st.st_gid);
                    u.sha256 = ""; u.target = target;
                    unmanaged.elements.push_back(u);
                } else if (reader && reader->link_differs_from_baseline(root, logical, target)) {
                    ManagedBaselineRecord b;
                    b.name = logical; b.type = "link"; b.mode = mode_octal(st.st_mode);
                    b.user = user_name(st.st_uid); b.group = group_name(st.st_gid);
                    b.sha256 = ""; b.target = target; b.package_name = pkg;
                    b.changes = {"target"};
                    changed.elements.push_back(b);
                }
            };
            walk_tree(base.string(), "", werr, on_file, on_link);
            if (werr.occurred) { if (handle_unreadable("files", werr.path)) return res; }
        }
        if (!changed.elements.empty()) res.manifest.changed_managed_files = changed;
        if (!unmanaged.elements.empty()) res.manifest.unmanaged_files = unmanaged;
    }

    res.ok = true;
    return res;
}

} // namespace zd
