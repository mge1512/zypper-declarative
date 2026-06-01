// generated from spec: zypper-declarative.spec.md sha256:f2cc80627e483a48bb8411d297711bc5f6c6e74c28dbf0dafc8fe7bd8817251e
//
// hashing.cpp -- SHA256 via OpenSSL libcrypto (EVP API, version-portable).
#include "hashing.hpp"

#include <openssl/evp.h>
#include <fstream>
#include <cstdio>

namespace zd {

static std::string to_hex(const unsigned char* d, unsigned int n) {
    static const char* h = "0123456789abcdef";
    std::string out;
    out.reserve(n * 2);
    for (unsigned int i = 0; i < n; ++i) {
        out.push_back(h[(d[i] >> 4) & 0xf]);
        out.push_back(h[d[i] & 0xf]);
    }
    return out;
}

std::string sha256_hex(const std::string& data) {
    unsigned char md[EVP_MAX_MD_SIZE];
    unsigned int mdlen = 0;
    EVP_MD_CTX* ctx = EVP_MD_CTX_new();
    EVP_DigestInit_ex(ctx, EVP_sha256(), nullptr);
    EVP_DigestUpdate(ctx, data.data(), data.size());
    EVP_DigestFinal_ex(ctx, md, &mdlen);
    EVP_MD_CTX_free(ctx);
    return to_hex(md, mdlen);
}

std::string sha256_file(const std::string& path, bool& ok) {
    std::ifstream f(path, std::ios::binary);
    if (!f) { ok = false; return ""; }
    EVP_MD_CTX* ctx = EVP_MD_CTX_new();
    EVP_DigestInit_ex(ctx, EVP_sha256(), nullptr);
    char buf[65536];
    while (f) {
        f.read(buf, sizeof(buf));
        std::streamsize n = f.gcount();
        if (n > 0) EVP_DigestUpdate(ctx, buf, static_cast<size_t>(n));
    }
    if (f.bad()) { EVP_MD_CTX_free(ctx); ok = false; return ""; }
    unsigned char md[EVP_MAX_MD_SIZE];
    unsigned int mdlen = 0;
    EVP_DigestFinal_ex(ctx, md, &mdlen);
    EVP_MD_CTX_free(ctx);
    ok = true;
    return to_hex(md, mdlen);
}

} // namespace zd
