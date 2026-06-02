// generated from spec: zypper-declarative.spec.md sha256:1641bb4413b82fecb081125067107bd5a4e30a8393edc778ead646207d68da5e
#include "actual_state.hpp"

#include <openssl/evp.h>

#include <zypp/target/rpm/RpmHeader.h>
#include <zypp/target/rpm/librpmDb.h>

#include <sys/stat.h>
#include <pwd.h>
#include <grp.h>

#include <algorithm>
#include <cstdint>
#include <cstdio>
#include <filesystem>
#include <fstream>
#include <map>
#include <sstream>
#include <system_error>
#include <unordered_map>

#include "manifest.hpp"

namespace fs = std::filesystem;

namespace zd {

namespace {

constexpr const char* kSyncpoint = "/etc/etc.syncpoint";

std::string join_root(const std::string& root, const std::string& abs) {
    std::string base = root;
    if (!base.empty() && base.back() == '/') base.pop_back();
    if (base == "" ) base = "";
    std::string p = abs;
    if (p.empty() || p[0] != '/') p = "/" + p;
    return base + p;
}

std::string mode_octal(mode_t m) {
    char buf[8];
    std::snprintf(buf, sizeof(buf), "0%03o", m & 07777);
    return std::string(buf);
}

std::string user_name(uid_t uid) {
    struct passwd* pw = getpwuid(uid);
    if (pw && pw->pw_name) return pw->pw_name;
    return std::to_string(uid);
}
std::string group_name(gid_t gid) {
    struct group* gr = getgrgid(gid);
    if (gr && gr->gr_name) return gr->gr_name;
    return std::to_string(gid);
}

std::string hash_file(const std::string& path, bool& readable) {
    readable = false;
    std::ifstream f(path, std::ios::binary);
    if (!f) return "";
    EVP_MD_CTX* ctx = EVP_MD_CTX_new();
    EVP_DigestInit_ex(ctx, EVP_sha256(), nullptr);
    char buf[65536];
    while (f) {
        f.read(buf, sizeof(buf));
        std::streamsize n = f.gcount();
        if (n > 0) EVP_DigestUpdate(ctx, buf, static_cast<size_t>(n));
    }
    if (f.bad()) { EVP_MD_CTX_free(ctx); return ""; }
    unsigned char digest[EVP_MAX_MD_SIZE];
    unsigned int len = 0;
    EVP_DigestFinal_ex(ctx, digest, &len);
    EVP_MD_CTX_free(ctx);
    static const char* hexd = "0123456789abcdef";
    std::string out;
    out.reserve(len * 2);
    for (unsigned int i = 0; i < len; ++i) {
        out.push_back(hexd[digest[i] >> 4]);
        out.push_back(hexd[digest[i] & 0xf]);
    }
    readable = true;
    return out;
}

// ----------------------------------------------------------------------
// libzypp rpmdb: package set and per-file baseline (bulk, one pass)
// ----------------------------------------------------------------------
struct FileBaseline {
    std::string package_name;
    std::string md5sum;       // recorded digest (legacy MD5 baseline)
    std::string link_target;  // recorded symlink target
    mode_t mode = 0;
    bool ghost = false;
    bool present = false;
};

struct RpmData {
    bool readable = false;
    std::vector<PackageRecord> packages;
    // path (as recorded by rpm, absolute, no root prefix) -> baseline
    std::unordered_map<std::string, FileBaseline> file_baseline;
};

// Read the rpmdb under root: the installed package set, and a path->baseline
// map built from each header's tag_fileinfos() in a single pass (bulk).
RpmData read_rpmdb(const std::string& root) {
    using zypp::target::rpm::RpmHeader;
    using zypp::target::rpm::librpmDb;
    using zypp::target::rpm::FileInfo;
    RpmData data;
    try {
        zypp::Pathname rootp(root);
        librpmDb::db_const_iterator it(rootp);
        if (!it.hasDB()) {
            data.readable = false;
            return data;
        }
        data.readable = true;
        for (it.findAll(); *it; ++it) {
            RpmHeader::constPtr h = *it;
            if (!h) continue;
            if (h->isSrc()) continue;
            std::string name = h->tag_name();
            if (name == "gpg-pubkey") continue;
            PackageRecord pr;
            pr.name = name;
            pr.version = h->tag_version();
            pr.release = h->tag_release();
            pr.arch = h->tag_arch().asString();
            data.packages.push_back(pr);

            for (const FileInfo& fi : h->tag_fileinfos()) {
                FileBaseline b;
                b.package_name = name;
                b.md5sum = fi.md5sum;
                b.link_target = fi.link_target.asString();
                b.mode = fi.mode;
                b.ghost = fi.ghost;
                b.present = true;
                data.file_baseline[fi.filename.asString()] = b;
            }
        }
        std::sort(data.packages.begin(), data.packages.end(),
                  [](const PackageRecord& a, const PackageRecord& b) {
                      if (a.name != b.name) return a.name < b.name;
                      return a.arch < b.arch;
                  });
    } catch (const std::exception&) {
        data.readable = false;
    } catch (...) {
        data.readable = false;
    }
    return data;
}

// libzypp records file digests historically as MD5; for the pristine
// comparison we compute the on-disk MD5 only to compare against that recorded
// baseline. The EMITTED record always carries SHA256 (computed separately).
std::string md5_file(const std::string& path, bool& readable) {
    readable = false;
    std::ifstream f(path, std::ios::binary);
    if (!f) return "";
    EVP_MD_CTX* ctx = EVP_MD_CTX_new();
    EVP_DigestInit_ex(ctx, EVP_md5(), nullptr);
    char buf[65536];
    while (f) {
        f.read(buf, sizeof(buf));
        std::streamsize n = f.gcount();
        if (n > 0) EVP_DigestUpdate(ctx, buf, static_cast<size_t>(n));
    }
    if (f.bad()) { EVP_MD_CTX_free(ctx); return ""; }
    unsigned char digest[EVP_MAX_MD_SIZE];
    unsigned int len = 0;
    EVP_DigestFinal_ex(ctx, digest, &len);
    EVP_MD_CTX_free(ctx);
    static const char* hexd = "0123456789abcdef";
    std::string out;
    for (unsigned int i = 0; i < len; ++i) {
        out.push_back(hexd[digest[i] >> 4]);
        out.push_back(hexd[digest[i] & 0xf]);
    }
    readable = true;
    return out;
}

// ----------------------------------------------------------------------
// Repositories: parse <root>/etc/zypp/repos.d/*.repo (INI)
// ----------------------------------------------------------------------
std::string strip(const std::string& s) {
    size_t a = s.find_first_not_of(" \t\r\n");
    if (a == std::string::npos) return "";
    size_t b = s.find_last_not_of(" \t\r\n");
    return s.substr(a, b - a + 1);
}
bool truthy(const std::string& v) {
    std::string s = v; std::transform(s.begin(), s.end(), s.begin(), ::tolower);
    return s == "1" || s == "true" || s == "yes" || s == "on";
}

// Returns false on unreadable directory (genuine I/O failure).
bool read_repositories(const std::string& root, std::vector<RepositoryRecord>& out,
                       bool& dir_exists) {
    std::string dir = join_root(root, "/etc/zypp/repos.d");
    std::error_code ec;
    dir_exists = fs::exists(dir, ec);
    if (!dir_exists) {
        // Genuinely absent directory => readable but empty.
        return true;
    }
    if (!fs::is_directory(dir, ec)) return true;
    std::vector<fs::directory_entry> entries;
    {
        fs::directory_iterator dit(dir, ec);
        if (ec) return false;  // cannot list directory => unreadable
        for (const auto& e : dit) entries.push_back(e);
    }
    for (const auto& e : entries) {
        if (e.path().extension() != ".repo") continue;
        std::ifstream f(e.path());
        if (!f) return false;  // a present .repo we cannot read => unreadable
        RepositoryRecord cur;
        bool in_section = false;
        std::string line;
        auto flush = [&]() {
            if (in_section && !cur.alias.empty()) out.push_back(cur);
        };
        while (std::getline(f, line)) {
            std::string t = strip(line);
            if (t.empty() || t[0] == '#' || t[0] == ';') continue;
            if (t.front() == '[' && t.back() == ']') {
                flush();
                cur = RepositoryRecord{};
                cur.alias = t.substr(1, t.size() - 2);
                cur.priority = 99;
                in_section = true;
                continue;
            }
            auto eq = t.find('=');
            if (eq == std::string::npos) continue;
            std::string k = strip(t.substr(0, eq));
            std::string v = strip(t.substr(eq + 1));
            if (k == "name") cur.name = v;
            else if (k == "baseurl") cur.url = v;
            else if (k == "type") cur.type = v;
            else if (k == "enabled") cur.enabled = truthy(v);
            else if (k == "gpgcheck") cur.gpgcheck = truthy(v);
            else if (k == "autorefresh") cur.autorefresh = truthy(v);
            else if (k == "priority") { try { cur.priority = std::stol(v); } catch (...) {} }
        }
        flush();
    }
    return true;
}

// ----------------------------------------------------------------------
// Services: systemctl --root <root> list-unit-files (offline enablement)
// ----------------------------------------------------------------------
bool read_services(const std::string& root, const CommandRunner& runner,
                   std::vector<ServiceRecord>& out, bool& source_ok) {
    source_ok = false;
    CommandResult r = runner.run(
        "systemctl",
        {"--root", root, "list-unit-files", "--no-legend", "--no-pager", "--full"});
    if (r.spawn_failed) return false;  // systemctl unavailable => unreadable
    // exit code may be non-zero on benign conditions; parse what we got.
    source_ok = true;
    std::istringstream is(r.out);
    std::string line;
    while (std::getline(is, line)) {
        std::istringstream ls(line);
        std::string unit, state;
        ls >> unit >> state;
        if (unit.empty() || state.empty()) continue;
        // Only declarable unit types and declarable states.
        auto ends = [&](const char* suf) {
            std::string s(suf);
            return unit.size() >= s.size() &&
                   unit.compare(unit.size() - s.size(), s.size(), s) == 0;
        };
        if (!(ends(".service") || ends(".timer") || ends(".socket") ||
              ends(".target") || ends(".path") || ends(".mount")))
            continue;
        std::string norm;
        if (state == "enabled" || state == "enabled-runtime") norm = "enabled";
        else if (state == "disabled") norm = "disabled";
        else if (state == "masked" || state == "masked-runtime") norm = "masked";
        else continue;  // static, generated, etc. are not declarable
        out.push_back(ServiceRecord{unit, norm});
    }
    std::sort(out.begin(), out.end(),
              [](const ServiceRecord& a, const ServiceRecord& b) {
                  return a.name < b.name;
              });
    return true;
}

// ----------------------------------------------------------------------
// Alternatives query for ghost symlinks (auto/best target)
// ----------------------------------------------------------------------
std::optional<std::string> alternatives_best(const std::string& name,
                                             const CommandRunner& runner) {
    CommandResult r = runner.run("update-alternatives", {"--query", name});
    if (r.spawn_failed) return std::nullopt;
    // Parse "Best: <path>" or "Value: <path>" line.
    std::istringstream is(r.out);
    std::string line, best, value;
    while (std::getline(is, line)) {
        if (line.rfind("Best:", 0) == 0) best = strip(line.substr(5));
        else if (line.rfind("Value:", 0) == 0) value = strip(line.substr(6));
    }
    if (!best.empty()) return best;
    if (!value.empty()) return value;
    return std::nullopt;
}

// ----------------------------------------------------------------------
// config_files walk over <root>/etc
// ----------------------------------------------------------------------
struct WalkContext {
    const RpmData* rpm;
    const CommandRunner* runner;
    const DescribeOptions* opts;
    std::vector<ManagedFileRecord>* out;
    std::vector<Diagnostic>* diags;
    bool unreadable_error = false;
    Diagnostic first_error;
};

// Decide whether an /etc entry should be emitted (reproducibility criterion).
// Returns true to EMIT, and fills the record (except content_ref).
bool judge_entry(const std::string& abs_path, const std::string& disk_path,
                 WalkContext& ctx, ManagedFileRecord& rec) {
    struct stat st;
    if (lstat(disk_path.c_str(), &st) != 0) {
        // genuine read failure on a required path
        if (ctx.opts->on_unreadable == OnUnreadable::Error) {
            ctx.unreadable_error = true;
            ctx.first_error = err("files", "cannot stat " + abs_path);
        } else {
            ctx.diags->push_back(warn("files", "omitted unreadable " + abs_path));
        }
        return false;
    }
    rec.name = abs_path;
    rec.mode = mode_octal(st.st_mode);
    rec.user = user_name(st.st_uid);
    rec.group = group_name(st.st_gid);

    const FileBaseline* base = nullptr;
    auto bit = ctx.rpm->file_baseline.find(abs_path);
    if (bit != ctx.rpm->file_baseline.end()) base = &bit->second;
    rec.package_name = base ? base->package_name : "";

    if (S_ISLNK(st.st_mode)) {
        rec.type = "link";
        std::error_code ec;
        fs::path tgt = fs::read_symlink(disk_path, ec);
        if (ec) {
            if (ctx.opts->on_unreadable == OnUnreadable::Error) {
                ctx.unreadable_error = true;
                ctx.first_error = err("files", "cannot read symlink " + abs_path);
            } else {
                ctx.diags->push_back(warn("files", "omitted unreadable " + abs_path));
            }
            return false;
        }
        rec.target = tgt.string();  // verbatim, not resolved
        rec.sha256 = "";
        rec.content_ref = "";
        if (!base) return true;  // unpackaged symlink -> emit
        if (base->mode && !S_ISLNK(base->mode)) {
            // recorded type is not a symlink -> type mismatch -> emit
            return true;
        }
        if (base->ghost) {
            // ghost symlink: compare against alternatives auto/best target
            std::string altname = fs::path(abs_path).filename().string();
            auto best = alternatives_best(altname, *ctx.runner);
            if (!best) {
                // cannot consult: treat under on_unreadable
                if (ctx.opts->on_unreadable == OnUnreadable::Error) {
                    ctx.unreadable_error = true;
                    ctx.first_error =
                        err("files", "cannot query alternatives for " + abs_path);
                    return false;
                }
                ctx.diags->push_back(
                    warn("files", "alternatives unreadable for " + abs_path));
                return false;
            }
            return rec.target != *best;  // differs from auto/best -> emit
        }
        // non-ghost symlink: pristine iff target matches recorded target
        return rec.target != base->link_target;
    }

    if (S_ISREG(st.st_mode)) {
        rec.type = "file";
        rec.target = "";
        bool readable = false;
        rec.sha256 = hash_file(disk_path, readable);
        if (!readable) {
            if (ctx.opts->on_unreadable == OnUnreadable::Error) {
                ctx.unreadable_error = true;
                ctx.first_error = err("files", "cannot read file " + abs_path);
            } else {
                ctx.diags->push_back(warn("files", "omitted unreadable " + abs_path));
            }
            return false;
        }
        if (!base) return true;  // unpackaged file -> emit
        if (base->mode && S_ISLNK(base->mode)) {
            return true;  // recorded as symlink, on disk a file -> type mismatch
        }
        if (base->ghost) {
            // ghost regular file: emit iff it has real on-disk content; a 0-byte
            // ghost matching an empty baseline is suppressed.
            return st.st_size > 0;
        }
        // non-ghost regular file: pristine iff content digest + mode match the
        // recorded baseline (libzypp records an MD5 digest).
        if (base->md5sum.empty()) return true;  // no usable baseline -> emit
        bool md5ok = false;
        std::string disk_md5 = md5_file(disk_path, md5ok);
        if (!md5ok) {
            if (ctx.opts->on_unreadable == OnUnreadable::Error) {
                ctx.unreadable_error = true;
                ctx.first_error = err("files", "cannot read file " + abs_path);
            } else {
                ctx.diags->push_back(warn("files", "omitted unreadable " + abs_path));
            }
            return false;
        }
        if (disk_md5 != base->md5sum) return true;  // content changed -> emit
        if ((st.st_mode & 07777) != (base->mode & 07777)) return true;  // mode change
        return false;  // pristine -> suppress
    }

    // directory: caller traverses, does not emit; special: skip.
    return false;
}

// Populate content store for an emitted regular-file record.
void capture_content(const std::string& store, const std::string& disk_path,
                     ManagedFileRecord& rec, WalkContext& ctx) {
    if (rec.type != "file") return;  // only regular files have content
    std::string blobdir = store;
    if (!blobdir.empty() && blobdir.back() == '/') blobdir.pop_back();
    blobdir += "/sha256";
    std::error_code ec;
    fs::create_directories(blobdir, ec);
    std::string blob = blobdir + "/" + rec.sha256;
    if (!fs::exists(blob, ec)) {
        std::ifstream in(disk_path, std::ios::binary);
        if (!in) {
            if (ctx.opts->on_unreadable == OnUnreadable::Error) {
                ctx.unreadable_error = true;
                ctx.first_error = err("files", "cannot read content of " + rec.name);
            } else {
                rec.content_ref = "";
                ctx.diags->push_back(
                    warn("files", "content unreadable for " + rec.name));
            }
            return;
        }
        std::ofstream o(blob, std::ios::binary | std::ios::trunc);
        o << in.rdbuf();
    }
    rec.content_ref = "sha256/" + rec.sha256;
}

void walk_etc(WalkContext& ctx) {
    std::string etc = join_root(ctx.opts->root, "/etc");
    std::error_code ec;
    if (!fs::exists(etc, ec)) return;
    // Iterative DFS classifying each entry by its own type (no symlink follow).
    std::vector<fs::path> stack;
    stack.push_back(etc);
    std::string rootbase = ctx.opts->root;
    if (!rootbase.empty() && rootbase.back() == '/') rootbase.pop_back();

    while (!stack.empty()) {
        fs::path dir = stack.back();
        stack.pop_back();
        std::error_code dec;
        fs::directory_iterator dit(dir, fs::directory_options::none, dec);
        if (dec) {
            // cannot list a directory => unreadable source
            std::string abs = dir.string();
            if (abs.rfind(rootbase, 0) == 0 && !rootbase.empty())
                abs = abs.substr(rootbase.size());
            if (ctx.opts->on_unreadable == OnUnreadable::Error) {
                ctx.unreadable_error = true;
                ctx.first_error = err("files", "cannot list directory " + abs);
                return;
            }
            ctx.diags->push_back(warn("files", "omitted unreadable directory " + abs));
            continue;
        }
        for (const auto& entry : dit) {
            if (ctx.unreadable_error) return;
            const fs::path& p = entry.path();
            std::string disk = p.string();
            std::string abs = disk;
            if (!rootbase.empty() && abs.rfind(rootbase, 0) == 0)
                abs = abs.substr(rootbase.size());
            if (abs.empty() || abs[0] != '/') abs = "/" + abs;

            struct stat st;
            if (lstat(disk.c_str(), &st) != 0) {
                if (ctx.opts->on_unreadable == OnUnreadable::Error) {
                    ctx.unreadable_error = true;
                    ctx.first_error = err("files", "cannot stat " + abs);
                    return;
                }
                ctx.diags->push_back(warn("files", "omitted unreadable " + abs));
                continue;
            }
            if (S_ISDIR(st.st_mode)) {
                stack.push_back(p);  // traverse, do not emit
                continue;
            }
            if (!S_ISREG(st.st_mode) && !S_ISLNK(st.st_mode)) {
                continue;  // special file: skip
            }
            // skip keep-list and syncpoint
            if (abs == kSyncpoint) continue;
            if (ctx.opts->keep_list.find(abs) != ctx.opts->keep_list.end()) continue;

            ManagedFileRecord rec;
            if (judge_entry(abs, disk, ctx, rec)) {
                if (ctx.opts->content_store && rec.type == "file")
                    capture_content(*ctx.opts->content_store, disk, rec, ctx);
                ctx.out->push_back(rec);
            }
            if (ctx.unreadable_error) return;
        }
    }
    std::sort(ctx.out->begin(), ctx.out->end(),
              [](const ManagedFileRecord& a, const ManagedFileRecord& b) {
                  return a.name < b.name;
              });
}

// ----------------------------------------------------------------------
// full-scan integrity (scope=full): /usr, usr-merge roots, /boot
// ----------------------------------------------------------------------
void walk_full(const DescribeOptions& opts, const RpmData& rpm,
               std::vector<ManagedBaselineRecord>& changed,
               std::vector<UnmanagedFileRecord>& unmanaged) {
    std::vector<std::string> trees = {"/usr", "/bin", "/sbin", "/lib", "/lib64", "/boot"};
    std::string rootbase = opts.root;
    if (!rootbase.empty() && rootbase.back() == '/') rootbase.pop_back();
    for (const auto& tree : trees) {
        std::string start = join_root(opts.root, tree);
        std::error_code ec;
        if (!fs::exists(start, ec)) continue;
        std::vector<fs::path> stack{start};
        while (!stack.empty()) {
            fs::path dir = stack.back();
            stack.pop_back();
            std::error_code dec;
            fs::directory_iterator dit(dir, fs::directory_options::none, dec);
            if (dec) continue;
            for (const auto& entry : dit) {
                const fs::path& p = entry.path();
                std::string disk = p.string();
                std::string abs = disk;
                if (!rootbase.empty() && abs.rfind(rootbase, 0) == 0)
                    abs = abs.substr(rootbase.size());
                if (abs.empty() || abs[0] != '/') abs = "/" + abs;
                if (opts.keep_list.find(abs) != opts.keep_list.end()) continue;
                struct stat st;
                if (lstat(disk.c_str(), &st) != 0) continue;
                if (S_ISDIR(st.st_mode)) { stack.push_back(p); continue; }
                if (!S_ISREG(st.st_mode) && !S_ISLNK(st.st_mode)) continue;

                auto bit = rpm.file_baseline.find(abs);
                bool owned = (bit != rpm.file_baseline.end());
                if (S_ISLNK(st.st_mode)) {
                    std::error_code lec;
                    std::string tgt = fs::read_symlink(disk, lec).string();
                    if (lec) continue;
                    if (owned) {
                        const FileBaseline& b = bit->second;
                        if (!b.ghost && b.link_target != tgt) {
                            ManagedBaselineRecord r;
                            r.name = abs; r.type = "link";
                            r.mode = mode_octal(st.st_mode);
                            r.user = user_name(st.st_uid);
                            r.group = group_name(st.st_gid);
                            r.target = tgt; r.package_name = b.package_name;
                            r.changes = {"target"};
                            changed.push_back(r);
                        }
                    } else {
                        UnmanagedFileRecord r;
                        r.name = abs; r.type = "link";
                        r.mode = mode_octal(st.st_mode);
                        r.user = user_name(st.st_uid);
                        r.group = group_name(st.st_gid);
                        r.target = tgt;
                        unmanaged.push_back(r);
                    }
                } else {  // regular file
                    bool ok = false;
                    std::string sha = hash_file(disk, ok);
                    if (!ok) continue;
                    if (owned) {
                        const FileBaseline& b = bit->second;
                        bool md5ok = false;
                        std::string md5 = md5_file(disk, md5ok);
                        if (!b.ghost && md5ok && !b.md5sum.empty() && md5 != b.md5sum) {
                            ManagedBaselineRecord r;
                            r.name = abs; r.type = "file";
                            r.mode = mode_octal(st.st_mode);
                            r.user = user_name(st.st_uid);
                            r.group = group_name(st.st_gid);
                            r.sha256 = sha; r.package_name = b.package_name;
                            r.changes = {"sha256"};
                            changed.push_back(r);
                        }
                    } else {
                        UnmanagedFileRecord r;
                        r.name = abs; r.type = "file";
                        r.mode = mode_octal(st.st_mode);
                        r.user = user_name(st.st_uid);
                        r.group = group_name(st.st_gid);
                        r.sha256 = sha;
                        unmanaged.push_back(r);
                    }
                }
            }
        }
    }
    std::sort(changed.begin(), changed.end(),
              [](const ManagedBaselineRecord& a, const ManagedBaselineRecord& b) {
                  return a.name < b.name;
              });
    std::sort(unmanaged.begin(), unmanaged.end(),
              [](const UnmanagedFileRecord& a, const UnmanagedFileRecord& b) {
                  return a.name < b.name;
              });
}

}  // namespace

Result<DescribeResult> describe_actual_state(const DescribeOptions& opts,
                                             const CommandRunner& runner) {
    DescribeResult out;
    Manifest& m = out.manifest;
    m.meta.format_version = 1;

    // STEP 1+ : packages and per-file baseline (one rpmdb pass).
    RpmData rpm = read_rpmdb(opts.root);
    if (!rpm.readable) {
        if (opts.on_unreadable == OnUnreadable::Error)
            return err("packages", "cannot read the rpm database under " + opts.root);
        out.diagnostics.push_back(warn("packages", "rpmdb unreadable; scope omitted"));
    } else if (!rpm.packages.empty()) {
        PackagesScope ps;
        ps.attributes["package_system"] = "rpm";
        ps.elements = rpm.packages;
        m.packages = ps;
    }

    // STEP 2: repositories
    {
        std::vector<RepositoryRecord> repos;
        bool dir_exists = false;
        bool ok = read_repositories(opts.root, repos, dir_exists);
        if (!ok) {
            if (opts.on_unreadable == OnUnreadable::Error)
                return err("repositories", "cannot read " +
                           join_root(opts.root, "/etc/zypp/repos.d"));
            out.diagnostics.push_back(
                warn("repositories", "repos.d unreadable; scope omitted"));
        } else if (!repos.empty()) {
            std::sort(repos.begin(), repos.end(),
                      [](const RepositoryRecord& a, const RepositoryRecord& b) {
                          return a.alias < b.alias;
                      });
            RepositoriesScope rs;
            rs.attributes["repository_system"] = "zypp";
            rs.elements = repos;
            m.repositories = rs;
        }
        // genuinely empty readable scope => omitted (no scope set)
    }

    // STEP 3: services
    {
        std::vector<ServiceRecord> svcs;
        bool source_ok = false;
        bool ok = read_services(opts.root, runner, svcs, source_ok);
        if (!ok || !source_ok) {
            if (opts.on_unreadable == OnUnreadable::Error)
                return err("services", "cannot read unit enablement under " + opts.root);
            out.diagnostics.push_back(
                warn("services", "unit enablement unreadable; scope omitted"));
        } else if (!svcs.empty()) {
            ServicesScope ss;
            ss.attributes["init_system"] = "systemd";
            ss.elements = svcs;
            m.services = ss;
        }
    }

    // STEP 4: config_files (/etc walk)
    {
        std::vector<ManagedFileRecord> files;
        WalkContext ctx;
        ctx.rpm = &rpm;
        ctx.runner = &runner;
        ctx.opts = &opts;
        ctx.out = &files;
        ctx.diags = &out.diagnostics;
        walk_etc(ctx);
        if (ctx.unreadable_error) return ctx.first_error;
        if (!files.empty()) {
            ConfigFilesScope cs;  // _attributes is {} (empty object)
            cs.elements = files;
            m.config_files = cs;
        }
    }

    // STEP 4a: full-scan integrity
    if (opts.scope == ScanScope::Full) {
        std::vector<ManagedBaselineRecord> changed;
        std::vector<UnmanagedFileRecord> unmanaged;
        walk_full(opts, rpm, changed, unmanaged);
        if (!changed.empty()) {
            ChangedManagedFilesScope sc;
            sc.elements = changed;
            m.changed_managed_files = sc;
        }
        if (!unmanaged.empty()) {
            UnmanagedFilesScope su;
            su.elements = unmanaged;
            m.unmanaged_files = su;
        }
    }

    return out;
}

}  // namespace zd
