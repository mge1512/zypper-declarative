// generated from spec: zypper-declarative.spec.md sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
#include <iostream>

#include "describe.hpp"

#include <algorithm>
#include <cstring>
#include <ctime>
#include <filesystem>
#include <fstream>
#include <grp.h>
#include <map>
#include <pwd.h>
#include <sstream>
#include <sys/stat.h>
#include <unistd.h>

#include <zypp/target/rpm/RpmHeader.h>
#include <zypp/target/rpm/librpmDb.h>

#include "hash.hpp"
#include "fullscan.hpp"
#include "meta.hpp"

namespace fs = std::filesystem;

namespace zd {

namespace {

std::string now_rfc3339() {
    std::time_t t = std::time(nullptr);
    std::tm tm;
    gmtime_r(&t, &tm);
    char buf[32];
    std::strftime(buf, sizeof(buf), "%Y-%m-%dT%H:%M:%SZ", &tm);
    return std::string(buf);
}

std::string octal_mode(mode_t m) {
    char buf[8];
    std::snprintf(buf, sizeof(buf), "%04o", m & 07777);
    return std::string(buf);
}

std::string user_name(uid_t uid) {
    struct passwd pw;
    struct passwd* res = nullptr;
    char buf[4096];
    if (getpwuid_r(uid, &pw, buf, sizeof(buf), &res) == 0 && res)
        return pw.pw_name;
    return std::to_string(uid);
}
std::string group_name(gid_t gid) {
    struct group gr;
    struct group* res = nullptr;
    char buf[4096];
    if (getgrgid_r(gid, &gr, buf, sizeof(buf), &res) == 0 && res)
        return gr.gr_name;
    return std::to_string(gid);
}

// The "logical" path of a filesystem object below the described root, i.e. with
// `root` stripped so the record carries an absolute /etc path even when reading
// a mounted snapshot. e.g. root="/mnt", full="/mnt/etc/foo" -> "/etc/foo".
std::string logical_path(const std::string& root, const fs::path& full) {
    std::string r = root;
    while (r.size() > 1 && r.back() == '/') r.pop_back();
    std::string f = full.string();
    if (r != "/" && f.rfind(r, 0) == 0) {
        std::string rest = f.substr(r.size());
        if (rest.empty()) return "/";
        return rest;
    }
    return f;
}

// Cache of per-package baseline FileInfo lists keyed by package name.
struct PackageBaseline {
    // path -> FileInfo
    std::map<std::string, zypp::target::rpm::FileInfo> by_path;
};

}  // namespace

// --------------------------------------------------------------------------
// Packages scope: enumerate the installed set from the rpmdb under root.
// --------------------------------------------------------------------------
static bool read_packages(const std::string& root,
                          ScopeWrapper<PackageRecord>& out, std::string& err) {
    try {
        zypp::target::rpm::librpmDb::db_const_iterator it{zypp::Pathname(root)};
        if (!it.hasDB()) {
            err = "rpmdb under " + root + " is not present";
            return false;
        }
        for (it.findAll(); *it; ++it) {
            const auto& h = *it;
            if (!h) continue;
            PackageRecord p;
            p.name = h->tag_name();
            zypp::Edition e = h->tag_edition();
            p.version = e.version();
            p.release = e.release();
            p.arch = h->tag_arch().asString();
            if (p.name == "gpg-pubkey") continue;  // pseudo package
            out.elements.push_back(p);
        }
        std::sort(out.elements.begin(), out.elements.end(),
                  [](const PackageRecord& a, const PackageRecord& b) {
                      if (a.name != b.name) return a.name < b.name;
                      return a.arch < b.arch;
                  });
        out.attributes["package_system"] = "rpm";
        return true;
    } catch (const std::exception& e) {
        err = std::string("rpmdb read failed: ") + e.what();
        return false;
    }
}

// --------------------------------------------------------------------------
// Repositories scope: parse <root>/etc/zypp/repos.d/*.repo (INI).
// --------------------------------------------------------------------------
static std::string trim(const std::string& s) {
    size_t a = s.find_first_not_of(" \t\r\n");
    size_t b = s.find_last_not_of(" \t\r\n");
    if (a == std::string::npos) return "";
    return s.substr(a, b - a + 1);
}
static bool to_bool(const std::string& v) {
    std::string s = v;
    std::transform(s.begin(), s.end(), s.begin(), ::tolower);
    return s == "1" || s == "true" || s == "yes" || s == "on";
}

static bool read_repositories(const std::string& root,
                              ScopeWrapper<RepositoryRecord>& out,
                              std::string& err) {
    std::string r = root;
    while (r.size() > 1 && r.back() == '/') r.pop_back();
    fs::path dir = fs::path(r == "/" ? "" : r) / "etc/zypp/repos.d";
    if (r == "/") dir = fs::path("/etc/zypp/repos.d");
    std::error_code ec;
    if (!fs::exists(dir, ec)) {
        // genuinely absent directory: treat as unreadable source.
        err = "repository directory not present: " + dir.string();
        return false;
    }
    fs::directory_iterator dit(dir, ec);
    if (ec) {
        err = "cannot list repository directory: " + dir.string();
        return false;
    }
    for (const auto& entry : dit) {
        if (entry.path().extension() != ".repo") continue;
        std::ifstream f(entry.path());
        if (!f.good()) {
            err = "cannot read repo file: " + entry.path().string();
            return false;
        }
        std::string line;
        RepositoryRecord rec;
        bool in_section = false;
        auto flush = [&]() {
            if (in_section && !rec.alias.empty()) out.elements.push_back(rec);
            rec = RepositoryRecord{};
        };
        while (std::getline(f, line)) {
            std::string t = trim(line);
            if (t.empty() || t[0] == '#' || t[0] == ';') continue;
            if (t.front() == '[' && t.back() == ']') {
                flush();
                rec.alias = t.substr(1, t.size() - 2);
                in_section = true;
                continue;
            }
            auto eq = t.find('=');
            if (eq == std::string::npos) continue;
            std::string key = trim(t.substr(0, eq));
            std::string val = trim(t.substr(eq + 1));
            if (key == "name") rec.name = val;
            else if (key == "baseurl" || key == "url") rec.url = val;
            else if (key == "type") rec.type = val;
            else if (key == "enabled") rec.enabled = to_bool(val);
            else if (key == "gpgcheck") rec.gpgcheck = to_bool(val);
            else if (key == "autorefresh") rec.autorefresh = to_bool(val);
            else if (key == "priority") {
                try { rec.priority = std::stol(val); } catch (...) {}
            }
        }
        flush();
    }
    std::sort(out.elements.begin(), out.elements.end(),
              [](const RepositoryRecord& a, const RepositoryRecord& b) {
                  return a.alias < b.alias;
              });
    out.attributes["repository_system"] = "zypp";
    return true;
}

// --------------------------------------------------------------------------
// Services scope: unit enablement via systemctl --root <root> (offline).
// --------------------------------------------------------------------------
static bool read_services(const std::string& root,
                          ScopeWrapper<ServiceRecord>& out,
                          const CommandRunner& runner, std::string& err) {
    CommandResult res =
        runner.run("systemctl", {"--root", root, "list-unit-files", "--no-legend",
                                 "--no-pager", "--type=service,timer,socket,"
                                 "target,path,mount"});
    if (res.code != 0 && res.out.empty()) {
        err = "cannot read unit enablement under " + root;
        return false;
    }
    std::istringstream ss(res.out);
    std::string line;
    while (std::getline(ss, line)) {
        std::istringstream ls(line);
        std::string name, state;
        ls >> name >> state;
        if (name.empty() || state.empty()) continue;
        // Only declarable states; omit static/generated etc.
        std::string norm;
        if (state == "enabled" || state == "enabled-runtime") norm = "enabled";
        else if (state == "disabled") norm = "disabled";
        else if (state == "masked" || state == "masked-runtime") norm = "masked";
        else continue;  // static, indirect, generated, transient -> omit
        ServiceRecord s;
        s.name = name;
        s.state = norm;
        out.elements.push_back(s);
    }
    std::sort(out.elements.begin(), out.elements.end(),
              [](const ServiceRecord& a, const ServiceRecord& b) {
                  return a.name < b.name;
              });
    out.attributes["init_system"] = "systemd";
    return true;
}

// --------------------------------------------------------------------------
// Per-file baseline via libzypp tag_fileinfos for the owning package.
// --------------------------------------------------------------------------
static const PackageBaseline& baseline_for(
    const std::string& root, const std::string& pkg,
    std::map<std::string, PackageBaseline>& cache) {
    auto it = cache.find(pkg);
    if (it != cache.end()) return it->second;
    PackageBaseline pb;
    try {
        zypp::target::rpm::librpmDb::db_const_iterator dit{
            zypp::Pathname(root)};
        if (dit.findPackage(pkg) && *dit) {
            for (const auto& fi : (*dit)->tag_fileinfos())
                pb.by_path[fi.filename.asString()] = fi;
        }
    } catch (...) {
        // leave empty; treated as no baseline
    }
    auto res = cache.emplace(pkg, std::move(pb));
    return res.first->second;
}

// Parse <root>/var/lib/alternatives/<name> and return the auto/best master
// target. In auto mode the best target is the provider with the highest
// priority. File layout (alternatives(8)): status line; master link name;
// slave-name/slave-link pairs until a blank line; then provider blocks of
// <master target> + <priority> + one slave target per slave, blank-separated.
static std::string alternatives_best_from_file(const std::string& root,
                                               const std::string& name) {
    std::string r = root;
    while (r.size() > 1 && r.back() == '/') r.pop_back();
    fs::path adm = (r == "/")
                       ? fs::path("/var/lib/alternatives") / name
                       : fs::path(r + "/var/lib/alternatives") / name;
    std::ifstream f(adm);
    if (!f.good()) return "";
    std::vector<std::string> lines;
    std::string l;
    while (std::getline(f, l)) lines.push_back(l);
    if (lines.size() < 2) return "";
    size_t i = 0;
    ++i;  // status line
    ++i;  // master link name
    // slave-name + slave-link pairs until a blank line
    while (i < lines.size() && !lines[i].empty()) {
        ++i;                          // slave name
        if (i < lines.size()) ++i;    // slave link
    }
    if (i < lines.size() && lines[i].empty()) ++i;  // skip blank separator
    std::string best_target;
    long best_prio = -1;
    while (i < lines.size()) {
        if (lines[i].empty()) { ++i; continue; }
        std::string provider = lines[i];
        if (i + 1 >= lines.size()) break;
        long prio = -1;
        try { prio = std::stol(lines[i + 1]); }
        catch (...) { ++i; continue; }
        if (prio > best_prio) { best_prio = prio; best_target = provider; }
        i += 2;
        while (i < lines.size() && !lines[i].empty()) ++i;  // slave targets
    }
    return best_target;
}

// Query the alternatives auto/best target for a given /etc/alternatives name.
static std::string alternatives_best(const std::string& root,
                                     const std::string& name,
                                     const CommandRunner& runner) {
    std::string from_file = alternatives_best_from_file(root, name);
    if (!from_file.empty()) return from_file;
    CommandResult r = runner.run("update-alternatives", {"--query", name});
    if (r.code != 0) return "";
    std::istringstream ss(r.out);
    std::string line, best, value;
    while (std::getline(ss, line)) {
        if (line.rfind("Best:", 0) == 0) best = trim(line.substr(5));
        else if (line.rfind("Value:", 0) == 0) value = trim(line.substr(6));
    }
    if (!best.empty()) return best;
    return value;
}

// --------------------------------------------------------------------------
// config_files: walk <root>/etc applying the reproducibility emission rule.
// --------------------------------------------------------------------------
struct WalkErr {
    bool err = false;
    std::string source;
};

static bool emit_decision(const std::string& root, const fs::path& full,
                          const std::string& lpath, char fstype,
                          const std::string& linktarget,
                          const std::string& disk_sha256, mode_t fmode,
                          const std::string& owner_pkg,
                          std::map<std::string, PackageBaseline>& cache,
                          const CommandRunner& runner) {
    // fstype: 'f' regular, 'l' symlink
    if (owner_pkg.empty()) {
        // unpackaged -> always emit (cannot be reproduced by a fresh install)
        return true;
    }
    const PackageBaseline& pb = baseline_for(root, owner_pkg, cache);
    auto bit = pb.by_path.find(lpath);
    if (bit == pb.by_path.end()) {
        // owned by a package but no per-file baseline found: be conservative
        // and emit (cannot prove reproducibility).
        return true;
    }
    const zypp::target::rpm::FileInfo& fi = bit->second;
    // A ghost path records no usable baseline (a %ghost): its recorded
    // link_target is empty even for a symlink, so the type-mismatch test below
    // must NOT run for a ghost. Handle ghost first.
    if (fi.ghost) {
        if (fstype == 'l') {
            // Ghost symlink: compare on-disk target to the reproducible target.
            // For /etc/alternatives/*, that is the auto/best provider.
            std::string name = fs::path(lpath).filename().string();
            std::string best;
            if (lpath.rfind("/etc/alternatives/", 0) == 0)
                best = alternatives_best(root, name, runner);
            if (best.empty()) return true;  // cannot determine -> emit
            return linktarget != best;       // differs -> emit, equals -> suppress
        }
        // Ghost regular file: emit iff it has real on-disk content.
        std::error_code ec;
        auto sz = fs::file_size(full, ec);
        if (ec) return true;
        return sz > 0;  // empty ghost matching empty baseline -> suppress
    }

    bool recorded_is_link = !fi.link_target.empty();
    // Type mismatch (non-ghost): recorded type differs from on-disk type ->
    // emit. A non-ghost regular file records a digest (non-empty md5sum) and an
    // empty link_target; a non-ghost symlink records a non-empty link_target.
    bool recorded_is_file = !fi.md5sum.empty();
    if (fstype == 'l' && recorded_is_file) return true;   // disk link, pkg file
    if (fstype == 'f' && recorded_is_link) return true;   // disk file, pkg link

    // Non-ghost pristine test.
    if (fstype == 'l') {
        // Symlink: pristine iff target matches the recorded target. The
        // package records the target verbatim too, but rpm and the filesystem
        // may differ only in a redundant leading "./" segment (e.g. recorded
        // "./../ibus" vs on-disk "../ibus"); collapse that for the pristine
        // DECISION only — the emitted record still stores the verbatim target.
        auto norm = [](std::string t) {
            while (t.rfind("./", 0) == 0 && t.size() > 2) t.erase(0, 2);
            return t;
        };
        return norm(linktarget) != norm(fi.link_target.asString());
    }
    // Regular file: pristine iff content digest AND mode match the recorded
    // baseline. The recorded digest (FileInfo.md5sum) carries whatever digest
    // algorithm the package used; modern rpm records SHA256 (64 hex chars),
    // older packages MD5 (32) or SHA1 (40). Compare against the matching
    // on-disk digest by length.
    bool mode_match = ((fi.mode & 07777) == (fmode & 07777));
    if (fi.md5sum.empty()) return true;  // no baseline -> not provably pristine
    std::string disk_digest;
    if (fi.md5sum.size() == 64) {
        disk_digest = disk_sha256;  // SHA256, already computed for the record
    } else if (fi.md5sum.size() == 32) {
        disk_digest = md5_file_hex(full.string());
    } else {
        // SHA1 or other legacy algorithm we do not compute here: fall back to
        // the SHA256 the record carries; if it differs, emit.
        disk_digest = disk_sha256;
    }
    if (disk_digest.empty()) return true;
    bool digest_match = (disk_digest == fi.md5sum);
    return !(digest_match && mode_match);
}

static void store_content(const std::string& content_store,
                          const std::string& digest, const fs::path& full,
                          std::vector<Diagnostic>& diags,
                          bool on_unreadable_error, WalkErr& werr) {
    if (content_store.empty() || digest.empty()) return;
    fs::path blob = fs::path(content_store) / "sha256" / digest;
    std::error_code ec;
    if (fs::exists(blob, ec)) return;  // dedup: do not rewrite
    fs::create_directories(blob.parent_path(), ec);
    std::ifstream in(full, std::ios::binary);
    if (!in.good()) {
        if (on_unreadable_error) {
            werr.err = true;
            werr.source = full.string();
        } else {
            diags.push_back({Severity::Warning, "files",
                             "content unreadable: " + full.string()});
        }
        return;
    }
    std::ofstream out(blob, std::ios::binary | std::ios::trunc);
    out << in.rdbuf();
}

static bool walk_etc(const std::string& root, ScopeWrapper<ManagedFileRecord>& out,
                     bool on_unreadable_error,
                     const std::set<std::string>& keep_list,
                     const std::string& content_store,
                     const CommandRunner& runner,
                     std::vector<Diagnostic>& diags, WalkErr& werr) {
    std::string r = root;
    while (r.size() > 1 && r.back() == '/') r.pop_back();
    fs::path etc = (r == "/") ? fs::path("/etc") : fs::path(r) / "etc";
    std::error_code ec;
    if (!fs::exists(etc, ec)) {
        // No /etc at all: treat as a genuinely empty readable scope (omit).
        return true;
    }
    std::map<std::string, PackageBaseline> cache;

    fs::recursive_directory_iterator it(
        etc, fs::directory_options::skip_permission_denied, ec);
    if (ec) {
        if (on_unreadable_error) { werr.err = true; werr.source = etc.string(); }
        else diags.push_back({Severity::Warning, "files",
                              "cannot list " + etc.string()});
        return !on_unreadable_error;
    }
    fs::recursive_directory_iterator end;
    for (; it != end; it.increment(ec)) {
        if (ec) {
            if (on_unreadable_error) {
                werr.err = true;
                werr.source = "directory listing under /etc";
                return false;
            }
            diags.push_back({Severity::Warning, "files",
                             "skipped unreadable entry under /etc"});
            ec.clear();
            continue;
        }
        const fs::path& full = it->path();
        std::string lpath = logical_path(root, full);
        if (lpath == "/etc/etc.syncpoint" || keep_list.count(lpath)) {
            continue;
        }
        // lstat-equivalent: classify by the entry's OWN type, not following
        // links.
        std::error_code sec;
        fs::file_status st = fs::symlink_status(full, sec);
        if (sec) {
            // genuine stat failure on a required entry
            if (on_unreadable_error) {
                werr.err = true;
                werr.source = full.string();
                return false;
            }
            diags.push_back({Severity::Warning, "files",
                             "cannot stat " + full.string()});
            continue;
        }
        fs::file_type ft = st.type();
        if (ft == fs::file_type::directory) {
            continue;  // traversed by the recursive iterator; not emitted
        }
        if (ft != fs::file_type::regular && ft != fs::file_type::symlink) {
            continue;  // special file: skip, do not read/hash/emit/error
        }

        // Gather metadata via lstat for mode/uid/gid.
        struct stat sb;
        if (lstat(full.c_str(), &sb) != 0) {
            if (on_unreadable_error) { werr.err = true; werr.source = full.string(); return false; }
            diags.push_back({Severity::Warning, "files", "cannot lstat " + full.string()});
            continue;
        }
        std::string owner_pkg;
        try {
            zypp::target::rpm::librpmDb::db_const_iterator dit{
                zypp::Pathname(root)};
            if (dit.findByFile(lpath) && *dit) owner_pkg = (*dit)->tag_name();
        } catch (...) {
            owner_pkg.clear();
        }

        ManagedFileRecord rec;
        rec.name = lpath;
        rec.mode = octal_mode(sb.st_mode);
        rec.user = user_name(sb.st_uid);
        rec.group = group_name(sb.st_gid);
        rec.package_name = owner_pkg;

        if (ft == fs::file_type::symlink) {
            rec.type = "link";
            std::error_code lec;
            fs::path tgt = fs::read_symlink(full, lec);  // verbatim
            if (lec) {
                if (on_unreadable_error) { werr.err = true; werr.source = full.string(); return false; }
                diags.push_back({Severity::Warning, "files", "cannot read symlink " + full.string()});
                continue;
            }
            rec.target = tgt.string();
            rec.sha256 = "";
            rec.content_ref = "";
            if (emit_decision(root, full, lpath, 'l', rec.target, "", sb.st_mode,
                              owner_pkg, cache, runner))
                out.elements.push_back(rec);
        } else {  // regular file
            rec.type = "file";
            rec.target = "";
            auto digest = sha256_file(full.string());
            if (!digest) {
                if (on_unreadable_error) { werr.err = true; werr.source = full.string(); return false; }
                diags.push_back({Severity::Warning, "files", "content unreadable: " + full.string()});
                continue;
            }
            rec.sha256 = *digest;
            if (emit_decision(root, full, lpath, 'f', "", rec.sha256, sb.st_mode,
                              owner_pkg, cache, runner)) {
                if (!content_store.empty()) {
                    store_content(content_store, rec.sha256, full, diags,
                                  on_unreadable_error, werr);
                    if (werr.err) return false;
                    rec.content_ref = "sha256/" + rec.sha256;
                }
                out.elements.push_back(rec);
            }
        }
    }
    std::sort(out.elements.begin(), out.elements.end(),
              [](const ManagedFileRecord& a, const ManagedFileRecord& b) {
                  return a.name < b.name;
              });
    // config_files scope has no scope-level attributes.
    return true;
}

DescribeResult describe_actual_state(const std::string& root,
                                     bool on_unreadable_error, ScanScope scope,
                                     const std::set<std::string>& keep_list,
                                     const std::string& content_store,
                                     const CommandRunner& runner) {
    DescribeResult res;
    res.manifest.meta.format_version = 1;
    res.manifest.meta.generator =
        std::string(kProgramName) + " " + kVersion;
    res.manifest.meta.created_at = now_rfc3339();
    res.manifest.meta.desired_sha256 = "";

    auto fail = [&](const std::string& domain, const std::string& src) {
        res.ok = false;
        res.error = {Severity::Error, domain, "unreadable source: " + src};
    };

    // packages
    {
        ScopeWrapper<PackageRecord> sc;
        std::string err;
        if (!read_packages(root, sc, err)) {
            if (on_unreadable_error) { fail("packages", err); return res; }
            res.diagnostics.push_back({Severity::Warning, "packages", err});
        } else if (!sc.elements.empty()) {
            res.manifest.packages = std::move(sc);
        }
        // genuinely empty readable scope -> omit
    }
    // repositories
    {
        ScopeWrapper<RepositoryRecord> sc;
        std::string err;
        if (!read_repositories(root, sc, err)) {
            if (on_unreadable_error) { fail("repositories", err); return res; }
            res.diagnostics.push_back({Severity::Warning, "repositories", err});
        } else if (!sc.elements.empty()) {
            res.manifest.repositories = std::move(sc);
        }
    }
    // services
    {
        ScopeWrapper<ServiceRecord> sc;
        std::string err;
        if (!read_services(root, sc, runner, err)) {
            if (on_unreadable_error) { fail("units", err); return res; }
            res.diagnostics.push_back({Severity::Warning, "units", err});
        } else if (!sc.elements.empty()) {
            res.manifest.services = std::move(sc);
        }
    }
    // config_files (/etc walk)
    {
        ScopeWrapper<ManagedFileRecord> sc;
        WalkErr werr;
        if (!walk_etc(root, sc, on_unreadable_error, keep_list, content_store,
                      runner, res.diagnostics, werr)) {
            if (werr.err) { fail("files", werr.source); return res; }
        }
        if (!sc.elements.empty()) res.manifest.config_files = std::move(sc);
    }

    // full-scan integrity scopes are produced only under scope=full. Walking
    // /usr and /boot for integrity is the expensive, opt-in path; it is
    // populated here when requested. (It mirrors the /etc walk for the
    // out-of-/etc trees.)
    if (scope == ScanScope::Full) {
        // The full-scan trees and exclusions are defined by the spec. A
        // complete /usr+/boot integrity scan against the package baseline is
        // performed by the same emission machinery; populated lazily. When no
        // changed/unmanaged entries are found the scopes are omitted.
        // (Implementation note: see TRANSLATION_REPORT.md "scope=full".)
        ScopeWrapper<ManagedBaselineRecord> changed;
        ScopeWrapper<UnmanagedFileRecord> unmanaged;
        std::string ferr;
        if (!full_scan(root, on_unreadable_error, keep_list, changed, unmanaged,
                       res.diagnostics, ferr)) {
            if (on_unreadable_error) { fail("files", ferr); return res; }
        }
        if (!changed.elements.empty())
            res.manifest.changed_managed_files = std::move(changed);
        if (!unmanaged.elements.empty())
            res.manifest.unmanaged_files = std::move(unmanaged);
    }

    res.ok = true;
    return res;
}

}  // namespace zd
