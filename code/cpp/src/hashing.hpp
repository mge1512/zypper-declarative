// generated from spec: zypper-declarative.spec.md sha256:aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3
#pragma once

#include <string>
#include <optional>

namespace zd {

// SHA256 of a byte string, lowercase hex (64 chars).
std::string sha256_hex(const std::string& data);

// SHA256 of a file's contents. nullopt on read failure.
std::optional<std::string> sha256_file(const std::string& path);

}  // namespace zd
