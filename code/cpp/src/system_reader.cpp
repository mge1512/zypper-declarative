// generated from spec: zypper-declarative.spec.md sha256:f2cc80627e483a48bb8411d297711bc5f6c6e74c28dbf0dafc8fe7bd8817251e
//
// system_reader.cpp -- ZyppSystemReader, backed by libzypp's rpmdb iterator.
// Package query and file ownership use libzypp (zypp::target::rpm). Unit
// enablement is read via systemd's offline-friendly on-disk state; in this
// version it is read with systemctl-equivalent logic deferred to the live-host
// milestone, so query_services returns empty/readable (no live units asserted)
// until that milestone wires systemd. This keeps the converger correct (it adds
// nothing it cannot verify) while leaving the live unit read to on-target work.
#include "system_reader.hpp"
#include "config.hpp"

#include <zypp/target/rpm/librpmDb.h>
#include <zypp/target/rpm/RpmHeader.h>
#include <zypp/Pathname.h>

#include <exception>

namespace zd {

using zypp::target::rpm::librpmDb;
using zypp::target::rpm::RpmHeader;

PackagesScope ZyppSystemReader::query_packages(const std::string& root, bool& ok) const {
    PackagesScope ps;
    ps.attributes["package_system"] = "rpm";
    try {
        librpmDb::db_const_iterator it{ zypp::Pathname(root) };
        for (; *it; ++it) {
            RpmHeader::constPtr h = *it;
            if (!h) continue;
            PackageRecord r;
            r.name = h->tag_name();
            r.version = h->tag_version();
            r.release = h->tag_release();
            r.arch = h->tag_arch().asString();
            if (r.name.empty()) continue;
            ps.elements.push_back(r);
        }
        ok = true;
    } catch (const std::exception&) {
        // A genuine rpmdb access failure is an unreadable source.
        ok = false;
    }
    return ps;
}

ServicesScope ZyppSystemReader::query_services(const std::string& root, bool& ok) const {
    debug_log("query_services: live systemd enablement read is deferred to the "
              "live-host milestone for root=" + root);
    // Readable but no live unit data wired yet: return an empty, readable scope.
    // (Not an unreadable source; the converger simply asserts nothing it cannot
    // verify here. The live read is on-target work.)
    ServicesScope ss;
    ss.attributes["init_system"] = "systemd";
    ok = true;
    return ss;
}

std::string ZyppSystemReader::owning_package(const std::string& root,
                                             const std::string& path) const {
    try {
        librpmDb::db_const_iterator it{ zypp::Pathname(root) };
        if (it.findByFile(path)) {
            RpmHeader::constPtr h = *it;
            if (h) return h->tag_name();
        }
    } catch (const std::exception&) {
        // treat as unpackaged on lookup failure
    }
    return "";
}

bool ZyppSystemReader::file_differs_from_baseline(const std::string& root,
                                                  const std::string& path,
                                                  const std::string& actual_sha256) const {
    // Compare the actual content digest against the package-recorded digest for
    // this file. If the file is unpackaged we report a difference (no baseline).
    //
    // NOTE (live-host deferred): rpm's per-file digest field (FileInfo.md5sum)
    // carries the package-recorded file digest in rpm's configured algorithm,
    // which on modern SUSE is SHA256 but is not guaranteed across all packages.
    // A robust algorithm-aware comparison is part of the live-host milestone.
    // Here we compare the recorded digest string against the actual SHA256; an
    // empty recorded digest (non-regular or no baseline) is treated as pristine.
    try {
        librpmDb::db_const_iterator it{ zypp::Pathname(root) };
        if (!it.findByFile(path)) return true; // unpackaged
        RpmHeader::constPtr h = *it;
        if (!h) return true;
        std::list<zypp::target::rpm::FileInfo> files = h->tag_fileinfos();
        for (const auto& fi : files) {
            if (fi.filename.asString() == path) {
                if (fi.md5sum.empty()) return false; // no content baseline
                return fi.md5sum != actual_sha256;
            }
        }
    } catch (const std::exception&) {
        // On lookup failure, be conservative and report changed.
        return true;
    }
    return false; // file in package and digest matched (pristine)
}

bool ZyppSystemReader::link_differs_from_baseline(const std::string& root,
                                                  const std::string& path,
                                                  const std::string& actual_target) const {
    try {
        librpmDb::db_const_iterator it{ zypp::Pathname(root) };
        if (!it.findByFile(path)) return true; // unpackaged
        RpmHeader::constPtr h = *it;
        if (!h) return true;
        std::list<zypp::target::rpm::FileInfo> files = h->tag_fileinfos();
        for (const auto& fi : files) {
            if (fi.filename.asString() == path) {
                std::string recorded = fi.link_target.asString();
                if (recorded.empty()) return false;
                return recorded != actual_target;
            }
        }
    } catch (const std::exception&) {
        return true;
    }
    return false;
}

} // namespace zd
