// generated from spec: zypper-declarative.spec.md sha256:f2cc80627e483a48bb8411d297711bc5f6c6e74c28dbf0dafc8fe7bd8817251e
//
// hashing.hpp -- SHA256 over a byte buffer, lowercase hex. Implemented with
// the system crypto library (OpenSSL libcrypto), which is already part of the
// libzypp dependency closure. Used for desired_sha256 (over the canonical model
// serialisation) and for file content digests in describe-actual-state.
#ifndef ZD_HASHING_HPP
#define ZD_HASHING_HPP

#include <string>

namespace zd {
std::string sha256_hex(const std::string& data);
// Hash a regular file's contents; returns "" with ok=false on read failure.
std::string sha256_file(const std::string& path, bool& ok);
} // namespace zd

#endif // ZD_HASHING_HPP
