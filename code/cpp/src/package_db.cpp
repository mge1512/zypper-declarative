// generated from spec: zypper-declarative.spec.md sha256:aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3
#include "package_db.hpp"

#include <sys/stat.h>
#include <grp.h>
#include <pwd.h>

#include <sstream>

#include <zypp/target/rpm/RpmDb.h>
#include <zypp/target/rpm/RpmHeader.h>
#include <zypp/PathInfo.h>

namespace zypprpm = zypp::target::rpm;

namespace zd {

static std::string mode_to_octal(mode_t m) {
    std::ostringstream ss;
    // permission bits only, four-digit octal with leading zero
    ss << '0';
    ss << ((m >> 6) & 7);
    ss << ((m >> 3) & 7);
    ss << (m & 7);
    return ss.str();
}

static std::string uid_name(uid_t uid) {
    struct passwd* pw = ::getpwuid(uid);
    if (pw && pw->pw_name) return pw->pw_name;
    return std::to_string(uid);
}
static std::string gid_name(gid_t gid) {
    struct group* gr = ::getgrgid(gid);
    if (gr && gr->gr_name) return gr->gr_name;
    return std::to_string(gid);
}

struct PackageDb::Impl {
    zypprpm::RpmDb db;
    bool opened = false;
};

PackageDb::PackageDb(const std::string& root) {
    impl_ = new Impl();
    try {
        impl_->db.initDatabase(zypp::Pathname(root));
        impl_->opened = true;
        available_ = true;
    } catch (...) {
        // No rpmdb under this root (synthetic root, or unreadable). Treat as an
        // empty installed set; every path is unowned. This is the correct
        // answer for a root with no package database.
        impl_->opened = false;
        available_ = false;
    }
}

PackageDb::~PackageDb() {
    delete impl_;
}

std::vector<PackageRecord> PackageDb::installed_packages() const {
    std::vector<PackageRecord> out;
    if (!available_) return out;
    try {
        for (auto it = impl_->db.dbConstIterator(); *it; ++it) {
            zypprpm::RpmHeader::constPtr h = *it;
            if (!h) continue;
            PackageRecord r;
            r.name = h->tag_name();
            zypp::Edition ed = h->tag_edition();
            r.version = ed.version();
            r.release = ed.release();
            r.arch = h->tag_arch().asString();
            if (!r.name.empty()) out.push_back(r);
        }
    } catch (...) {
        // partial enumeration failure: return what we have
    }
    return out;
}

FileBaseline PackageDb::file_baseline(const std::string& abs_path) const {
    FileBaseline fb;
    if (!available_) return fb;

    std::string owner;
    try {
        owner = impl_->db.whoOwnsFile(abs_path);
    } catch (...) {
        owner = "";
    }
    if (owner.empty()) return fb;  // unpackaged

    fb.found = true;
    fb.package_name = owner;

    // Read the owning package header and find the FileInfo for this path.
    try {
        zypprpm::RpmHeader::constPtr h;
        impl_->db.getData(owner, h);
        if (h) {
            for (const auto& fi : h->tag_fileinfos()) {
                if (fi.filename.asString() == abs_path) {
                    fb.recorded_md5 = fi.md5sum;
                    {
                        // libzypp/rpm records a symlink target that may carry a
                        // leading "./" normalisation artifact (e.g. "./../ibus"
                        // for an on-disk "../ibus"); strip it so the pristine
                        // comparison against the verbatim on-disk target holds.
                        std::string lt = fi.link_target.asString();
                        if (lt.rfind("./", 0) == 0 && lt.size() > 2 && lt[2] != '/')
                            lt = lt.substr(2);
                        fb.recorded_target = lt;
                    }
                    fb.recorded_mode = mode_to_octal(fi.mode);
                    fb.recorded_user = uid_name(fi.uid);
                    fb.recorded_group = gid_name(fi.gid);
                    fb.ghost = fi.ghost;
                    fb.is_link = S_ISLNK(fi.mode);
                    fb.is_dir = S_ISDIR(fi.mode);
                    break;
                }
            }
        }
    } catch (...) {
        // ownership known, baseline detail unavailable
    }

    return fb;
}

}  // namespace zd
