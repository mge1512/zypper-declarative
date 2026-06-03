// generated from spec: zypper-declarative.spec.md sha256:aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3
// tests by: gemini-3-5-flash
#pragma once

#include <string>
#include <vector>
#include <filesystem>
#include <iostream>
#include <fstream>
#include <sstream>
#include <atomic>
#include <chrono>
#include <map>
#include <algorithm>
#include <sys/types.h>
#include <sys/wait.h>
#include <unistd.h>
#include <signal.h>

namespace fs = std::filesystem;

struct CommandResult {
    std::string stdout_data;
    std::string stderr_data;
    int exit_code;
};

class TempDir {
public:
    fs::path path;
    TempDir();
    ~TempDir();
};

CommandResult run_command(const std::vector<std::string>& args);
bool is_root();
void write_file(const fs::path& p, const std::string& content);
std::string read_file(const fs::path& p);

// Test registration and reporting structures
struct TestInfo {
    std::string name;
    std::string file;
    int line;
    bool failed;
    bool skipped;
    std::string message;
};

extern std::vector<TestInfo> g_test_results;

void register_test_result(const std::string& name, const std::string& file, int line, bool failed, bool skipped, const std::string& message);

typedef void (*TestCaseFn)();
struct TestCase {
    std::string name;
    TestCaseFn fn;
};
extern std::vector<TestCase> g_test_cases;

// Static registry helper class
class TestRegistry {
public:
    TestRegistry(const std::string& name, TestCaseFn fn) {
        g_test_cases.push_back({name, fn});
    }
};

#define TEST_CASE(name) \
    void name(); \
    static TestRegistry register_##name(#name, name); \
    void name()

// Assertion macros
#define ASSERT_TRUE(cond) \
    do { \
        if (!(cond)) { \
            register_test_result(__func__, __FILE__, __LINE__, true, false, "Assertion failed: " #cond); \
            return; \
        } \
    } while(0)

#define ASSERT_FALSE(cond) \
    do { \
        if (cond) { \
            register_test_result(__func__, __FILE__, __LINE__, true, false, "Assertion failed: !(" #cond ")"); \
            return; \
        } \
    } while(0)

#define ASSERT_EQ(val1, val2) \
    do { \
        if ((val1) != (val2)) { \
            std::ostringstream ss; \
            ss << "Assertion failed: " #val1 " == " #val2 " (actual: '" << (val1) << "', expected: '" << (val2) << "')"; \
            register_test_result(__func__, __FILE__, __LINE__, true, false, ss.str()); \
            return; \
        } \
    } while(0)

#define ASSERT_NE(val1, val2) \
    do { \
        if ((val1) == (val2)) { \
            std::ostringstream ss; \
            ss << "Assertion failed: " #val1 " != " #val2 " (value: '" << (val1) << "')"; \
            register_test_result(__func__, __FILE__, __LINE__, true, false, ss.str()); \
            return; \
        } \
    } while(0)

#define ASSERT_CONTAINS(str, substr) \
    do { \
        if ((str).find(substr) == std::string::npos) { \
            register_test_result(__func__, __FILE__, __LINE__, true, false, "Assertion failed: " #str " contains " #substr " (actual: '" + (str) + "')"); \
            return; \
        } \
    } while(0)

#define ASSERT_NOT_CONTAINS(str, substr) \
    do { \
        if ((str).find(substr) != std::string::npos) { \
            register_test_result(__func__, __FILE__, __LINE__, true, false, "Assertion failed: " #str " does not contain " #substr); \
            return; \
        } \
    } while(0)

#define TEST_SKIP(reason) \
    do { \
        register_test_result(__func__, __FILE__, __LINE__, false, true, reason); \
        return; \
    } while(0)
