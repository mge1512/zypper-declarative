// generated from spec: zypper-declarative.spec.md sha256:aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3
#include "hashing.hpp"

#include <array>
#include <cstdio>
#include <fstream>

#include <openssl/evp.h>

namespace zd {

static std::string to_hex(const unsigned char* d, size_t n) {
    static const char* hx = "0123456789abcdef";
    std::string out;
    out.reserve(n * 2);
    for (size_t i = 0; i < n; ++i) {
        out.push_back(hx[(d[i] >> 4) & 0xF]);
        out.push_back(hx[d[i] & 0xF]);
    }
    return out;
}

std::string sha256_hex(const std::string& data) {
    unsigned char digest[EVP_MAX_MD_SIZE];
    unsigned int len = 0;
    EVP_MD_CTX* ctx = EVP_MD_CTX_new();
    EVP_DigestInit_ex(ctx, EVP_sha256(), nullptr);
    EVP_DigestUpdate(ctx, data.data(), data.size());
    EVP_DigestFinal_ex(ctx, digest, &len);
    EVP_MD_CTX_free(ctx);
    return to_hex(digest, len);
}

std::optional<std::string> sha256_file(const std::string& path) {
    std::ifstream f(path, std::ios::binary);
    if (!f.is_open()) return std::nullopt;
    EVP_MD_CTX* ctx = EVP_MD_CTX_new();
    EVP_DigestInit_ex(ctx, EVP_sha256(), nullptr);
    std::array<char, 65536> buf;
    while (f) {
        f.read(buf.data(), static_cast<std::streamsize>(buf.size()));
        std::streamsize n = f.gcount();
        if (n > 0) EVP_DigestUpdate(ctx, buf.data(), static_cast<size_t>(n));
    }
    if (f.bad()) { EVP_MD_CTX_free(ctx); return std::nullopt; }
    unsigned char digest[EVP_MAX_MD_SIZE];
    unsigned int len = 0;
    EVP_DigestFinal_ex(ctx, digest, &len);
    EVP_MD_CTX_free(ctx);
    return to_hex(digest, len);
}

}  // namespace zd
