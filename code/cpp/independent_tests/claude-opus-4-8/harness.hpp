// generated from spec: zypper-declarative.spec.md sha256:aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3
// tests by: claude-opus-4-8
#pragma once

#include <string>
#include <vector>
#include <functional>
#include <filesystem>
#include <fstream>
#include <sstream>

namespace fs = std::filesystem;

// Result of invoking the binary under test (black-box).
struct CmdResult {
    std::string out;
    std::string err;
    int code = -1;
};

// Invoke the binary under test at the canonical path "../../zypper-declarative"
// with an argv vector (no shell). Captures stdout/stderr per-invocation in
// per-call temp files, hermetic and shell-free.
CmdResult run_zd(const std::vector<std::string>& args);

// RAII temp directory; removed on destruction.
struct TempDir {
    fs::path path;
    TempDir();
    ~TempDir();
};

void write_text(const fs::path& p, const std::string& content);
std::string read_text(const fs::path& p);

// Minimal test registry/harness; exit non-zero on any failure.
struct TestCase {
    std::string name;
    std::function<void()> fn;
};
std::vector<TestCase>& registry();
struct Reg { Reg(const std::string& n, std::function<void()> f); };

#define TEST(name)                                            \
    static void name();                                       \
    static Reg reg_##name(#name, name);                       \
    static void name()

// Assertions throw on failure (caught by the runner).
void zd_assert(bool cond, const std::string& msg, const char* file, int line);
void zd_skip(const std::string& reason);

#define ASSERT(cond) zd_assert((cond), "ASSERT failed: " #cond, __FILE__, __LINE__)
#define ASSERT_EQ(a, b)                                                       \
    do {                                                                      \
        auto _a = (a); auto _b = (b);                                         \
        std::ostringstream _ss;                                               \
        _ss << "ASSERT_EQ failed: " #a " == " #b " (got '" << _a             \
            << "' vs '" << _b << "')";                                        \
        zd_assert(_a == _b, _ss.str(), __FILE__, __LINE__);                   \
    } while (0)
#define ASSERT_CONTAINS(s, sub)                                               \
    zd_assert((s).find(sub) != std::string::npos,                             \
              std::string("ASSERT_CONTAINS failed: missing '") + (sub) +      \
                  "' in: " + (s), __FILE__, __LINE__)
#define ASSERT_NOT_CONTAINS(s, sub)                                           \
    zd_assert((s).find(sub) == std::string::npos,                             \
              std::string("ASSERT_NOT_CONTAINS failed: found '") + (sub) +    \
                  "'", __FILE__, __LINE__)
#define SKIP(reason) zd_skip(reason)
