// generated from spec: zypper-declarative.spec.md sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
//
// SHA256 helpers over libcrypto (OpenSSL 3): hash a byte string and hash a
// file's content. Used for desired_sha256 (canonical-model hash) and for
// config-file content digests.
#ifndef ZD_HASH_HPP
#define ZD_HASH_HPP

#include <optional>
#include <string>

namespace zd {

// Lowercase hex SHA256 of the given bytes.
std::string sha256_hex(const std::string& data);

// Lowercase hex SHA256 of the file at path; std::nullopt if it cannot be read.
std::optional<std::string> sha256_file(const std::string& path);

// Lowercase hex MD5 of the file at path; empty string if it cannot be read.
// Used only to compare against a package-recorded legacy md5 baseline.
std::string md5_file_hex(const std::string& path);

}  // namespace zd

#endif  // ZD_HASH_HPP
