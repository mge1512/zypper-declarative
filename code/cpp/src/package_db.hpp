// generated from spec: zypper-declarative.spec.md sha256:aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3
#pragma once

#include <map>
#include <optional>
#include <string>
#include <vector>

#include "types.hpp"

namespace zd {

// Per-file baseline a package records for a path.
struct FileBaseline {
    bool found = false;          // the path is owned and recorded by a package
    std::string package_name;    // bare owning package name
    std::string recorded_md5;    // RPMTAG_FILEMD5S (may be md5 or sha256, hex)
    std::string recorded_target; // RPMTAG_FILELINKTOS (symlink target)
    std::string recorded_mode;   // octal string of recorded mode bits
    std::string recorded_user;
    std::string recorded_group;
    bool is_link = false;        // recorded type is a symlink
    bool is_dir = false;         // recorded type is a directory
    bool ghost = false;          // RPMFILE_GHOST
};

// Isolates the libzypp/rpmdb interface (BEHAVIOR: describe-actual-state's
// single reader of package state). Opens the rpmdb under a given root once and
// provides bulk ownership + baseline lookups. On a root with no rpmdb (e.g. a
// synthetic test root) it reports an empty installed set and every path as
// unowned, which is the correct, deterministic answer for that root.
class PackageDb {
public:
    explicit PackageDb(const std::string& root);
    ~PackageDb();

    // True if the rpmdb under `root` could be opened.
    bool available() const { return available_; }

    // The fully resolved installed set (name, version, release, arch).
    std::vector<PackageRecord> installed_packages() const;

    // Owning package and recorded baseline for an absolute path (as it appears
    // in the package db, i.e. without the root prefix). `found` is false when
    // no package owns the path.
    FileBaseline file_baseline(const std::string& abs_path) const;

private:
    struct Impl;
    Impl* impl_;
    bool available_ = false;
};

}  // namespace zd
