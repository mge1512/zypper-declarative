// generated from spec: zypper-declarative.spec.md sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03

#include "fullscan.hpp"

#include <algorithm>
#include <filesystem>
#include <grp.h>
#include <map>
#include <pwd.h>
#include <sys/stat.h>

#include <zypp/target/rpm/RpmHeader.h>
#include <zypp/target/rpm/librpmDb.h>

#include "hash.hpp"

namespace fs = std::filesystem;

namespace zd {
namespace {

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
std::string logical_path(const std::string& root, const fs::path& full) {
    std::string r = root;
    while (r.size() > 1 && r.back() == '/') r.pop_back();
    std::string f = full.string();
    if (r != "/" && f.rfind(r, 0) == 0) {
        std::string rest = f.substr(r.size());
        return rest.empty() ? "/" : rest;
    }
    return f;
}

}  // namespace

bool full_scan(const std::string& root, bool on_unreadable_error,
               const std::set<std::string>& keep_list,
               ScopeWrapper<ManagedBaselineRecord>& changed,
               ScopeWrapper<UnmanagedFileRecord>& unmanaged,
               std::vector<Diagnostic>& diags, std::string& err) {
    std::string r = root;
    while (r.size() > 1 && r.back() == '/') r.pop_back();
    const std::vector<std::string> trees = {"/usr", "/bin",  "/sbin", "/lib",
                                            "/lib64", "/boot"};

    std::map<std::string, std::map<std::string, zypp::target::rpm::FileInfo>>
        baseline_cache;
    auto baseline_for = [&](const std::string& pkg)
        -> const std::map<std::string, zypp::target::rpm::FileInfo>& {
        auto it = baseline_cache.find(pkg);
        if (it != baseline_cache.end()) return it->second;
        std::map<std::string, zypp::target::rpm::FileInfo> m;
        try {
            zypp::target::rpm::librpmDb::db_const_iterator dit{
                zypp::Pathname(root)};
            if (dit.findPackage(pkg) && *dit)
                for (const auto& fi : (*dit)->tag_fileinfos())
                    m[fi.filename.asString()] = fi;
        } catch (...) {
        }
        return baseline_cache.emplace(pkg, std::move(m)).first->second;
    };

    for (const auto& tree : trees) {
        fs::path base = (r == "/") ? fs::path(tree) : fs::path(r + tree);
        std::error_code ec;
        if (!fs::exists(base, ec)) continue;
        // Do not cross into other mounts: record this tree's device.
        struct stat tb;
        dev_t tree_dev = 0;
        if (lstat(base.c_str(), &tb) == 0) tree_dev = tb.st_dev;

        fs::recursive_directory_iterator it(
            base, fs::directory_options::skip_permission_denied, ec);
        if (ec) {
            if (on_unreadable_error) { err = base.string(); return false; }
            diags.push_back({Severity::Warning, "files",
                             "cannot list " + base.string()});
            continue;
        }
        fs::recursive_directory_iterator end;
        for (; it != end; it.increment(ec)) {
            if (ec) {
                if (on_unreadable_error) { err = "listing under " + tree; return false; }
                ec.clear();
                continue;
            }
            const fs::path& full = it->path();
            std::string lpath = logical_path(root, full);
            if (keep_list.count(lpath)) {
                it.disable_recursion_pending();
                continue;
            }
            struct stat sb;
            if (lstat(full.c_str(), &sb) != 0) {
                if (on_unreadable_error) { err = full.string(); return false; }
                continue;
            }
            // separate-mount guard
            if (tree_dev != 0 && sb.st_dev != tree_dev &&
                S_ISDIR(sb.st_mode)) {
                it.disable_recursion_pending();
                continue;
            }
            if (S_ISDIR(sb.st_mode)) continue;  // traversed, not emitted
            bool is_link = S_ISLNK(sb.st_mode);
            bool is_reg = S_ISREG(sb.st_mode);
            if (!is_link && !is_reg) continue;  // special file: skip

            std::string owner_pkg;
            try {
                zypp::target::rpm::librpmDb::db_const_iterator dit{
                    zypp::Pathname(root)};
                if (dit.findByFile(lpath) && *dit)
                    owner_pkg = (*dit)->tag_name();
            } catch (...) {
            }

            std::string disk_target, disk_sha;
            if (is_link) {
                std::error_code lec;
                disk_target = fs::read_symlink(full, lec).string();
                if (lec) continue;
            } else {
                auto d = sha256_file(full.string());
                if (!d) {
                    if (on_unreadable_error) { err = full.string(); return false; }
                    diags.push_back({Severity::Warning, "files",
                                     "content unreadable: " + full.string()});
                    continue;
                }
                disk_sha = *d;
            }

            if (owner_pkg.empty()) {
                UnmanagedFileRecord u;
                u.name = lpath;
                u.type = is_link ? "link" : "file";
                u.mode = octal_mode(sb.st_mode);
                u.user = user_name(sb.st_uid);
                u.group = group_name(sb.st_gid);
                u.sha256 = disk_sha;
                u.target = disk_target;
                unmanaged.elements.push_back(std::move(u));
                continue;
            }

            const auto& bl = baseline_for(owner_pkg);
            auto bit = bl.find(lpath);
            std::vector<std::string> chg;
            bool emit = false;
            if (bit == bl.end()) {
                emit = true;  // owned but no baseline -> conservatively changed
                chg.push_back("baseline");
            } else {
                const auto& fi = bit->second;
                bool recorded_link = !fi.link_target.empty();
                if (is_link != recorded_link) { emit = true; chg.push_back("type"); }
                else if (is_link) {
                    if (disk_target != fi.link_target.asString()) {
                        emit = true; chg.push_back("target");
                    }
                } else {  // regular
                    if (!fi.md5sum.empty()) {
                        std::string dm;
                        if (fi.md5sum.size() == 32)
                            dm = md5_file_hex(full.string());
                        else
                            dm = disk_sha;  // SHA256 (or fallback) comparison
                        if (!dm.empty() && dm != fi.md5sum) {
                            emit = true; chg.push_back("sha256");
                        }
                    }
                    if ((fi.mode & 07777) != (sb.st_mode & 07777)) {
                        emit = true; chg.push_back("mode");
                    }
                }
            }
            if (emit) {
                ManagedBaselineRecord m;
                m.name = lpath;
                m.type = is_link ? "link" : "file";
                m.mode = octal_mode(sb.st_mode);
                m.user = user_name(sb.st_uid);
                m.group = group_name(sb.st_gid);
                m.sha256 = disk_sha;
                m.target = disk_target;
                m.package_name = owner_pkg;
                m.changes = chg;
                changed.elements.push_back(std::move(m));
            }
        }
    }
    std::sort(changed.elements.begin(), changed.elements.end(),
              [](const ManagedBaselineRecord& a, const ManagedBaselineRecord& b) {
                  return a.name < b.name;
              });
    std::sort(unmanaged.elements.begin(), unmanaged.elements.end(),
              [](const UnmanagedFileRecord& a, const UnmanagedFileRecord& b) {
                  return a.name < b.name;
              });
    return true;
}

}  // namespace zd
