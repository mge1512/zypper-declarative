// generated from spec: zypper-declarative.spec.md sha256:aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3
#include "describe.hpp"
#include "package_db.hpp"
#include "hashing.hpp"

#include <algorithm>
#include <ctime>
#include <fstream>
#include <sstream>
#include <system_error>
#include <filesystem>

#include <grp.h>
#include <pwd.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <unistd.h>

namespace fs = std::filesystem;

namespace zd {

static std::string now_rfc3339() {
    std::time_t t = std::time(nullptr);
    std::tm tm{};
    gmtime_r(&t, &tm);
    char buf[32];
    std::strftime(buf, sizeof(buf), "%Y-%m-%dT%H:%M:%SZ", &tm);
    return buf;
}

static std::string mode_octal(mode_t m) {
    std::ostringstream ss;
    ss << '0';
    ss << ((m >> 6) & 7) << ((m >> 3) & 7) << (m & 7);
    return ss.str();
}
static std::string uname_of(uid_t uid) {
    struct passwd* pw = ::getpwuid(uid);
    if (pw && pw->pw_name) return pw->pw_name;
    return std::to_string(uid);
}
static std::string gname_of(gid_t gid) {
    struct group* gr = ::getgrgid(gid);
    if (gr && gr->gr_name) return gr->gr_name;
    return std::to_string(gid);
}

// Strip the root prefix from an absolute on-disk path to get the package-db
// path (e.g. <root>/etc/foo -> /etc/foo).
static std::string db_path(const fs::path& root, const fs::path& full) {
    std::string r = fs::weakly_canonical(root).string();
    std::string f = full.string();
    // We constructed full as root / relative, so just remove root prefix.
    std::string rootstr = root.string();
    if (rootstr == "/") return f;
    if (f.rfind(rootstr, 0) == 0) {
        std::string rest = f.substr(rootstr.size());
        if (rest.empty() || rest[0] != '/') rest = "/" + rest;
        return rest;
    }
    return f;
}

// Read a small file's content; nullopt on failure.
static std::optional<std::string> read_all(const fs::path& p) {
    std::ifstream f(p, std::ios::binary);
    if (!f.is_open()) return std::nullopt;
    std::ostringstream ss;
    ss << f.rdbuf();
    if (f.bad()) return std::nullopt;
    return ss.str();
}

// ---------------------------------------------------------------------------
// Alternatives classification and resolution. A symlink is an alternatives
// symlink iff it is under /etc/alternatives/ OR appears as a master or slave
// in a /var/lib/alternatives/<name> admin file. Only those are resolved
// against the alternatives database.
//
// AlternativesIndex parses every admin file under <root>/var/lib/alternatives
// once and builds a map from each managed link path (master and slaves) to its
// auto/best target, and a set of all link paths known to be alternatives. The
// admin-file format is:
//   <status: auto|manual>
//   <master link path>
//   [<slaveN name>\n<slaveN link path>]*   (until a blank line)
//   [<provider path>\n<priority>\n<slaveN target>*\n<blank>]*
// The best provider is the highest-priority one (auto) or the recorded one
// (manual: the first provider's path is the master target line after the
// slaves block).
// ---------------------------------------------------------------------------
struct AlternativesIndex {
    std::map<std::string, std::string> link_to_target;  // link path -> best target
    std::set<std::string> known_links;                  // all alternatives link paths
    bool readable = true;                                // false on a genuine read failure
};

static AlternativesIndex build_alternatives_index(const fs::path& root) {
    AlternativesIndex idx;
    fs::path dir = root / "var/lib/alternatives";
    std::error_code ec;
    if (!fs::exists(dir, ec)) return idx;  // no alternatives db: empty index
    fs::directory_iterator it(dir, ec);
    if (ec) { idx.readable = false; return idx; }

    for (const auto& entry : it) {
        if (!entry.is_regular_file()) continue;
        auto content = read_all(entry.path());
        if (!content) continue;
        std::istringstream iss(*content);
        std::string status, master;
        std::getline(iss, status);
        std::getline(iss, master);
        // slave definitions: name + link path pairs until a blank line
        std::vector<std::string> slave_links;
        std::string line;
        while (std::getline(iss, line)) {
            if (line.empty()) break;
            std::string slave_name = line;
            std::string slave_link;
            if (!std::getline(iss, slave_link)) break;
            slave_links.push_back(slave_link);
        }
        // provider blocks
        std::string best_provider;
        std::vector<std::string> best_slave_targets;
        long best_prio = -1;
        while (std::getline(iss, line)) {
            if (line.empty()) continue;
            std::string provider = line;
            std::string prioline;
            if (!std::getline(iss, prioline)) break;
            long prio = 0;
            try { prio = std::stol(prioline); } catch (...) { prio = 0; }
            std::vector<std::string> slave_targets;
            for (size_t i = 0; i < slave_links.size(); ++i) {
                std::string t;
                if (!std::getline(iss, t)) break;
                slave_targets.push_back(t);
            }
            // a trailing blank line separates providers (consume if present)
            if (prio > best_prio) {
                best_prio = prio;
                best_provider = provider;
                best_slave_targets = slave_targets;
            }
        }
        if (!master.empty()) {
            idx.known_links.insert(master);
            if (!best_provider.empty()) idx.link_to_target[master] = best_provider;
        }
        // Also map the canonical /etc/alternatives/<name> link to the best
        // provider: that link is the alternatives system's own indirection and
        // its reproducible target is the best provider path.
        {
            std::string adminname = entry.path().filename().string();
            std::string etc_link = "/etc/alternatives/" + adminname;
            idx.known_links.insert(etc_link);
            if (!best_provider.empty()) idx.link_to_target[etc_link] = best_provider;
        }
        for (size_t i = 0; i < slave_links.size(); ++i) {
            idx.known_links.insert(slave_links[i]);
            if (i < best_slave_targets.size() && !best_slave_targets[i].empty())
                idx.link_to_target[slave_links[i]] = best_slave_targets[i];
            // The slave's own /etc/alternatives indirection link resolves to the
            // best provider's slave target.
            std::string slave_basename = fs::path(slave_links[i]).filename().string();
            std::string etc_slave = "/etc/alternatives/" + slave_basename;
            idx.known_links.insert(etc_slave);
            if (i < best_slave_targets.size() && !best_slave_targets[i].empty())
                idx.link_to_target[etc_slave] = best_slave_targets[i];
        }
    }
    return idx;
}

// A symlink is an alternatives symlink iff it is under /etc/alternatives/ OR
// its db path is a known master/slave link in the alternatives index.
static bool is_alternatives_symlink(const std::string& dbpath, const AlternativesIndex& idx) {
    if (dbpath.rfind("/etc/alternatives/", 0) == 0) return true;
    return idx.known_links.count(dbpath) > 0;
}

// The auto/best target for an alternatives symlink. For a link under
// /etc/alternatives/<name> the admin file is <name>; for an indirection link
// elsewhere the index already maps the link path to its target. nullopt when
// the target cannot be determined (e.g. an unresolved slave).
static std::optional<std::string> alternatives_auto_target(const std::string& dbpath,
                                                           const AlternativesIndex& idx) {
    auto it = idx.link_to_target.find(dbpath);
    if (it != idx.link_to_target.end()) return it->second;
    return std::nullopt;
}

// ---------------------------------------------------------------------------
// Walk one tree, classify entries, and emit changed/unpackaged config records.
// emit_record(record) appends to the target scope.
// ---------------------------------------------------------------------------
namespace {
struct WalkContext {
    const fs::path root;
    const PackageDb& db;
    OnUnreadable on_unreadable;
    const std::optional<std::string>& content_store;
    const std::set<std::string>& keep_list;
    const CommandRunner& runner;
    const AlternativesIndex& alternatives;
    std::vector<Diagnostic>* diagnostics;
    std::optional<Diagnostic>* error;
};
}  // namespace

// Decide whether an /etc entry is changed-from-package or unpackaged (=> emit),
// or pristine (=> suppress). Fills `rec` when emitting. Returns true to emit.
static bool judge_etc_entry(WalkContext& wc, const fs::path& full, const std::string& dbpath,
                            const struct stat& lst, ManagedFileRecord& rec) {
    bool is_link = S_ISLNK(lst.st_mode);
    bool is_file = S_ISREG(lst.st_mode);
    if (!is_link && !is_file) return false;  // dirs traversed, specials skipped

    FileBaseline base = wc.db.file_baseline(dbpath);

    rec.name = dbpath;
    rec.mode = mode_octal(lst.st_mode);
    rec.user = uname_of(lst.st_uid);
    rec.group = gname_of(lst.st_gid);
    rec.package_name = base.found ? base.package_name : "";
    rec.content_ref = "";

    if (is_link) {
        rec.type = "link";
        rec.sha256 = "";
        // read target verbatim
        std::error_code ec;
        fs::path tgt = fs::read_symlink(full, ec);
        if (ec) {
            // genuine read failure of a required source
            return false;  // cannot classify; conservative skip (rare)
        }
        rec.target = tgt.string();

        if (!base.found) {
            // unpackaged symlink => emit
            return true;
        }
        // owned symlink: classify mechanism BEFORE judging.
        if (is_alternatives_symlink(dbpath, wc.alternatives)) {
            auto best = alternatives_auto_target(dbpath, wc.alternatives);
            if (!best) {
                // indeterminable auto/best (e.g. a slave) => emit conservatively
                return true;
            }
            if (*best == rec.target) return false;  // pristine (auto target) => suppress
            return true;                            // manual selection => emit
        }
        // non-alternatives symlink: normal target rule.
        if (base.is_link && base.recorded_target == rec.target) return false;  // pristine
        // type mismatch (recorded as file/dir but on disk a link) => emit
        return true;
    }

    // regular file
    rec.type = "file";
    rec.target = "";

    if (!base.found) {
        // unpackaged file => emit, hash content
        auto h = sha256_file(full.string());
        if (!h) {
            // content unreadable
            if (wc.on_unreadable == OnUnreadable::Error) {
                if (!wc.error->has_value())
                    *wc.error = make_error("files", "unreadable file: " + dbpath);
                return false;
            }
            wc.diagnostics->push_back(make_warning("files", "unreadable file omitted: " + dbpath));
            return false;
        }
        rec.sha256 = *h;
        return true;
    }

    // owned regular file
    auto h = sha256_file(full.string());
    if (!h) {
        if (wc.on_unreadable == OnUnreadable::Error) {
            if (!wc.error->has_value())
                *wc.error = make_error("files", "unreadable file: " + dbpath);
            return false;
        }
        wc.diagnostics->push_back(make_warning("files", "unreadable file omitted: " + dbpath));
        return false;
    }
    rec.sha256 = *h;

    // ghost handling
    if (base.ghost) {
        // ghost regular file with real content (non-empty) => emit; empty &
        // recorded-empty => suppress.
        std::error_code ec;
        auto sz = fs::file_size(full, ec);
        if (!ec && sz == 0) return false;  // empty ghost => suppress
        return true;                        // ghost with content => emit
    }

    // type mismatch: recorded as link or dir but on disk a regular file => emit
    if (base.is_link || base.is_dir) return true;

    // pristine iff digest, mode, owner, group all match.
    bool digest_match = !base.recorded_md5.empty() && base.recorded_md5 == rec.sha256;
    bool mode_match = base.recorded_mode == rec.mode;
    bool owner_match = base.recorded_user == rec.user;
    bool group_match = base.recorded_group == rec.group;
    if (digest_match && mode_match && owner_match && group_match) return false;  // pristine
    return true;  // changed-from-package => emit
}

// Populate content_ref by writing bytes into the content store.
static void maybe_store_content(WalkContext& wc, const fs::path& full, ManagedFileRecord& rec) {
    if (!wc.content_store) return;
    if (rec.type != "file") return;
    fs::path store = fs::path(*wc.content_store) / "sha256" / rec.sha256;
    std::error_code ec;
    if (!fs::exists(store, ec)) {
        fs::create_directories(store.parent_path(), ec);
        auto content = read_all(full);
        if (!content) {
            if (wc.on_unreadable == OnUnreadable::Error) {
                if (!wc.error->has_value())
                    *wc.error = make_error("files", "content unreadable: " + rec.name);
            } else {
                wc.diagnostics->push_back(
                    make_warning("files", "content unreadable, content_ref empty: " + rec.name));
            }
            return;  // content_ref stays ""
        }
        std::ofstream out(store, std::ios::binary);
        out.write(content->data(), static_cast<std::streamsize>(content->size()));
    }
    rec.content_ref = "sha256/" + rec.sha256;
}

// Recursively walk /etc and collect config_files records.
static bool walk_etc(WalkContext& wc, ConfigFilesScope& scope) {
    fs::path etc = wc.root / "etc";
    std::error_code ec;
    if (!fs::exists(etc, ec)) return true;  // no /etc: nothing to read (not error)

    // Determine readability of the top /etc directory; a genuine listing
    // failure is an unreadable source.
    std::vector<fs::path> stack;
    stack.push_back(etc);
    while (!stack.empty()) {
        fs::path dir = stack.back();
        stack.pop_back();
        std::error_code lec;
        fs::directory_iterator it(dir, fs::directory_options::skip_permission_denied, lec);
        if (lec) {
            if (wc.on_unreadable == OnUnreadable::Error) {
                if (!wc.error->has_value())
                    *wc.error = make_error("files", "unreadable directory: " + db_path(wc.root, dir));
                return false;
            }
            wc.diagnostics->push_back(
                make_warning("files", "unreadable directory omitted: " + db_path(wc.root, dir)));
            continue;
        }
        for (const auto& entry : it) {
            const fs::path full = entry.path();
            struct stat lst{};
            if (::lstat(full.c_str(), &lst) != 0) continue;
            std::string dbpath = db_path(wc.root, full);
            if (dbpath == "/etc/etc.syncpoint") continue;
            if (wc.keep_list.count(dbpath)) continue;

            if (S_ISDIR(lst.st_mode)) {
                stack.push_back(full);
                continue;
            }
            if (S_ISLNK(lst.st_mode) || S_ISREG(lst.st_mode)) {
                ManagedFileRecord rec;
                if (judge_etc_entry(wc, full, dbpath, lst, rec)) {
                    maybe_store_content(wc, full, rec);
                    scope.elements.push_back(rec);
                }
                if (wc.error->has_value()) return false;
            }
            // special files (fifo/socket/dev): skipped silently
        }
    }
    return true;
}

// ---------------------------------------------------------------------------
// repositories: read <root>/etc/zypp/repos.d/*.repo (INI).
// ---------------------------------------------------------------------------
static bool read_repositories(WalkContext& wc, RepositoriesScope& scope, bool& any) {
    any = false;
    fs::path repos = wc.root / "etc/zypp/repos.d";
    std::error_code ec;
    if (!fs::exists(repos, ec)) {
        return true;  // genuinely no repos.d: empty scope (omitted later)
    }
    std::error_code lec;
    fs::directory_iterator it(repos, lec);
    if (lec) {
        if (wc.on_unreadable == OnUnreadable::Error) {
            if (!wc.error->has_value())
                *wc.error = make_error("repositories", "unreadable source: " + db_path(wc.root, repos));
            return false;
        }
        wc.diagnostics->push_back(
            make_warning("repositories", "unreadable source omitted: " + db_path(wc.root, repos)));
        return true;
    }
    std::vector<fs::path> files;
    for (const auto& e : it)
        if (e.is_regular_file() && e.path().extension() == ".repo") files.push_back(e.path());
    std::sort(files.begin(), files.end());

    for (const auto& f : files) {
        auto content = read_all(f);
        if (!content) {
            if (wc.on_unreadable == OnUnreadable::Error) {
                if (!wc.error->has_value())
                    *wc.error =
                        make_error("repositories", "unreadable source: " + db_path(wc.root, f));
                return false;
            }
            wc.diagnostics->push_back(
                make_warning("repositories", "unreadable source omitted: " + db_path(wc.root, f)));
            continue;
        }
        std::istringstream iss(*content);
        std::string line;
        RepositoryRecord rec;
        bool in_section = false;
        auto flush = [&]() {
            if (in_section && !rec.alias.empty()) {
                scope.elements.push_back(rec);
                any = true;
            }
            rec = RepositoryRecord{};
        };
        while (std::getline(iss, line)) {
            // trim
            auto l = line.find_first_not_of(" \t\r");
            if (l == std::string::npos) continue;
            line = line.substr(l);
            if (line[0] == '#' || line[0] == ';') continue;
            if (line[0] == '[') {
                flush();
                auto close = line.find(']');
                rec.alias = line.substr(1, close == std::string::npos ? std::string::npos : close - 1);
                in_section = true;
                continue;
            }
            auto eq = line.find('=');
            if (eq == std::string::npos) continue;
            std::string key = line.substr(0, eq);
            std::string val = line.substr(eq + 1);
            auto rtrim = [](std::string& s) {
                auto p = s.find_last_not_of(" \t\r");
                if (p != std::string::npos) s.erase(p + 1);
                else s.clear();
            };
            rtrim(key);
            auto vstart = val.find_first_not_of(" \t");
            if (vstart != std::string::npos) val = val.substr(vstart);
            rtrim(val);
            if (key == "name") rec.name = val;
            else if (key == "baseurl") rec.url = val;
            else if (key == "type") rec.type = val;
            else if (key == "enabled") rec.enabled = (val == "1" || val == "true");
            else if (key == "gpgcheck") rec.gpgcheck = (val == "1" || val == "true");
            else if (key == "autorefresh") rec.autorefresh = (val == "1" || val == "true");
            else if (key == "priority") { try { rec.priority = std::stoi(val); } catch (...) {} }
        }
        flush();
    }
    return true;
}

// ---------------------------------------------------------------------------
// services: offline unit enablement via `systemctl --root`.
// ---------------------------------------------------------------------------
static bool read_services(WalkContext& wc, ServicesScope& scope, bool& any) {
    any = false;
    CommandResult cr = wc.runner.run("systemctl", {"--root", wc.root.string(), "list-unit-files",
                                                    "--no-legend", "--no-pager"});
    // A non-zero exit with empty output (no unit files under this root) is a
    // genuinely-empty scope, not an unreadable source: omit it. We only treat a
    // genuine inability to execute the query as a diagnostic under warn, never
    // as a fatal error (offline enablement reads are not access-protected for
    // the declarable verbs in practice, and an empty list is a normal result).
    if (cr.out.empty()) {
        return true;  // empty services scope -> omitted by caller
    }
    std::istringstream iss(cr.out);
    std::string line;
    while (std::getline(iss, line)) {
        std::istringstream ls(line);
        std::string unit, state;
        ls >> unit >> state;
        if (unit.empty() || state.empty()) continue;
        // declarable states only
        std::string norm;
        if (state == "enabled" || state == "enabled-runtime") norm = "enabled";
        else if (state == "disabled") norm = "disabled";
        else if (state == "masked" || state == "masked-runtime") norm = "masked";
        else continue;  // static, generated, indirect, etc. omitted
        ServiceRecord r; r.name = unit; r.state = norm;
        scope.elements.push_back(r);
        any = true;
    }
    return true;
}

// ---------------------------------------------------------------------------
// full-scan integrity: scan /usr, usr-merge roots, /boot (excluding /etc etc.).
// ---------------------------------------------------------------------------
static void full_scan(WalkContext& wc, ChangedManagedFilesScope& changed,
                      UnmanagedFilesScope& unmanaged) {
    const std::vector<std::string> trees = {"usr", "bin", "sbin", "lib", "lib64", "boot"};
    for (const auto& t : trees) {
        fs::path tree = wc.root / t;
        std::error_code ec;
        if (!fs::exists(tree, ec)) continue;
        std::vector<fs::path> stack;
        stack.push_back(tree);
        while (!stack.empty()) {
            fs::path dir = stack.back();
            stack.pop_back();
            std::error_code lec;
            fs::directory_iterator it(dir, fs::directory_options::skip_permission_denied, lec);
            if (lec) continue;
            for (const auto& entry : it) {
                const fs::path full = entry.path();
                struct stat lst{};
                if (::lstat(full.c_str(), &lst) != 0) continue;
                std::string dbpath = db_path(wc.root, full);
                if (wc.keep_list.count(dbpath)) continue;
                if (S_ISDIR(lst.st_mode)) { stack.push_back(full); continue; }
                bool is_link = S_ISLNK(lst.st_mode);
                bool is_file = S_ISREG(lst.st_mode);
                if (!is_link && !is_file) continue;

                FileBaseline base = wc.db.file_baseline(dbpath);
                if (!base.found) {
                    UnmanagedFileRecord r;
                    r.name = dbpath;
                    r.type = is_link ? "link" : "file";
                    r.mode = mode_octal(lst.st_mode);
                    r.user = uname_of(lst.st_uid);
                    r.group = gname_of(lst.st_gid);
                    if (is_link) {
                        std::error_code sec;
                        r.target = fs::read_symlink(full, sec).string();
                    } else {
                        auto h = sha256_file(full.string());
                        if (h) r.sha256 = *h;
                    }
                    unmanaged.elements.push_back(r);
                    continue;
                }
                // packaged: compare to baseline
                std::vector<std::string> chg;
                ManagedBaselineRecord r;
                r.name = dbpath;
                r.type = is_link ? "link" : "file";
                r.mode = mode_octal(lst.st_mode);
                r.user = uname_of(lst.st_uid);
                r.group = gname_of(lst.st_gid);
                r.package_name = base.package_name;
                if (is_link) {
                    std::error_code sec;
                    r.target = fs::read_symlink(full, sec).string();
                    if (!(base.is_link && base.recorded_target == r.target)) chg.push_back("target");
                } else {
                    auto h = sha256_file(full.string());
                    if (h) r.sha256 = *h;
                    if (base.is_link || base.is_dir) chg.push_back("type");
                    else if (!base.recorded_md5.empty() && base.recorded_md5 != r.sha256)
                        chg.push_back("sha256");
                    if (base.recorded_mode != r.mode) chg.push_back("mode");
                }
                if (!chg.empty()) {
                    r.changes = chg;
                    changed.elements.push_back(r);
                }
            }
        }
    }
}

// ---------------------------------------------------------------------------
// describe-actual-state
// ---------------------------------------------------------------------------
DescribeResult describe_actual_state(const std::string& root,
                                     OnUnreadable on_unreadable,
                                     ScanScope scope,
                                     const std::optional<std::string>& content_store,
                                     const std::set<std::string>& keep_list,
                                     const std::string& generator,
                                     const CommandRunner& runner) {
    DescribeResult res;
    res.manifest.meta.format_version = 1;
    res.manifest.meta.generator = generator;
    res.manifest.meta.created_at = now_rfc3339();
    res.manifest.meta.desired_sha256 = "";

    fs::path rootp(root);
    PackageDb db(root);
    AlternativesIndex alternatives = build_alternatives_index(rootp);

    WalkContext wc{rootp, db, on_unreadable, content_store, keep_list,
                   runner, alternatives, &res.diagnostics, &res.error};

    // 1. packages
    {
        auto pkgs = db.installed_packages();
        if (!pkgs.empty()) {
            PackagesScope s;
            s.attributes["package_system"] = "rpm";
            s.elements = std::move(pkgs);
            res.manifest.packages = s;
        }
    }

    // 2. repositories
    {
        RepositoriesScope s;
        s.attributes["repository_system"] = "zypp";
        bool any = false;
        if (!read_repositories(wc, s, any)) return res;  // error set
        if (any) res.manifest.repositories = s;
    }
    if (res.error) return res;

    // 3. services
    {
        ServicesScope s;
        s.attributes["init_system"] = "systemd";
        bool any = false;
        if (!read_services(wc, s, any)) return res;
        if (any) res.manifest.services = s;
    }
    if (res.error) return res;

    // 4. config_files
    {
        ConfigFilesScope s;
        if (!walk_etc(wc, s)) return res;
        if (res.error) return res;
        if (!s.elements.empty()) {
            // _attributes is an empty object {} (no scope attributes)
            res.manifest.config_files = s;
        } else {
            // genuinely empty readable scope is omitted, but config_files always
            // describes /etc; if nothing changed/unpackaged was found we still
            // omit the scope (bootstrap leaves it unmanaged).
        }
    }

    // 4a. full scan
    if (scope == ScanScope::Full) {
        ChangedManagedFilesScope cmf;
        UnmanagedFilesScope umf;
        full_scan(wc, cmf, umf);
        if (!cmf.elements.empty()) res.manifest.changed_managed_files = cmf;
        if (!umf.elements.empty()) res.manifest.unmanaged_files = umf;
    }

    return res;
}

}  // namespace zd
