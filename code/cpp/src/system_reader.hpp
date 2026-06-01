// generated from spec: zypper-declarative.spec.md sha256:f2cc80627e483a48bb8411d297711bc5f6c6e74c28dbf0dafc8fe7bd8817251e
//
// system_reader.hpp -- the production SystemReader backed by libzypp's rpmdb
// access (ZyppSystemReader). Package query, file ownership, and baseline digest
// comparison go through libzypp ONLY (never librpm directly, never exec rpm).
#ifndef ZD_SYSTEM_READER_HPP
#define ZD_SYSTEM_READER_HPP

#include "describe.hpp"

namespace zd {

class ZyppSystemReader : public SystemReader {
public:
    PackagesScope query_packages(const std::string& root, bool& ok) const override;
    ServicesScope query_services(const std::string& root, bool& ok) const override;
    std::string owning_package(const std::string& root,
                               const std::string& path) const override;
    bool file_differs_from_baseline(const std::string& root, const std::string& path,
                                    const std::string& actual_sha256) const override;
    bool link_differs_from_baseline(const std::string& root, const std::string& path,
                                    const std::string& actual_target) const override;
};

} // namespace zd

#endif // ZD_SYSTEM_READER_HPP
