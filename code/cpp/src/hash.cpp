// generated from spec: zypper-declarative.spec.md sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03

#include "hash.hpp"

#include <array>
#include <cstdio>
#include <openssl/evp.h>

namespace zd {

static std::string to_hex(const unsigned char* digest, unsigned int len) {
    static const char* hexchars = "0123456789abcdef";
    std::string out;
    out.reserve(len * 2);
    for (unsigned int i = 0; i < len; ++i) {
        out.push_back(hexchars[(digest[i] >> 4) & 0xF]);
        out.push_back(hexchars[digest[i] & 0xF]);
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
    FILE* f = std::fopen(path.c_str(), "rb");
    if (!f) return std::nullopt;
    EVP_MD_CTX* ctx = EVP_MD_CTX_new();
    EVP_DigestInit_ex(ctx, EVP_sha256(), nullptr);
    std::array<unsigned char, 65536> buf;
    size_t n;
    while ((n = std::fread(buf.data(), 1, buf.size(), f)) > 0)
        EVP_DigestUpdate(ctx, buf.data(), n);
    bool read_err = std::ferror(f) != 0;
    std::fclose(f);
    unsigned char digest[EVP_MAX_MD_SIZE];
    unsigned int len = 0;
    EVP_DigestFinal_ex(ctx, digest, &len);
    EVP_MD_CTX_free(ctx);
    if (read_err) return std::nullopt;
    return to_hex(digest, len);
}

std::string md5_file_hex(const std::string& path) {
    FILE* f = std::fopen(path.c_str(), "rb");
    if (!f) return "";
    EVP_MD_CTX* ctx = EVP_MD_CTX_new();
    EVP_DigestInit_ex(ctx, EVP_md5(), nullptr);
    std::array<unsigned char, 65536> buf;
    size_t n;
    while ((n = std::fread(buf.data(), 1, buf.size(), f)) > 0)
        EVP_DigestUpdate(ctx, buf.data(), n);
    bool read_err = std::ferror(f) != 0;
    std::fclose(f);
    unsigned char digest[EVP_MAX_MD_SIZE];
    unsigned int len = 0;
    EVP_DigestFinal_ex(ctx, digest, &len);
    EVP_MD_CTX_free(ctx);
    if (read_err) return "";
    return to_hex(digest, len);
}

}  // namespace zd
