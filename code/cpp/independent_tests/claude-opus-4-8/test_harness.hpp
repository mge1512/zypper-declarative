// generated from spec: zypper-declarative.spec.md sha256:f2cc80627e483a48bb8411d297711bc5f6c6e74c28dbf0dafc8fe7bd8817251e
// tests by: claude-opus-4-8
//
// Minimal black-box test harness for the zypper-declarative CLI binary.
// Tests invoke the built binary via fork/execvp (the interface declared in the
// spec's DEPLOYMENT section: a CLI tool invoked with bare-word verbs and
// key=value options). Tests assert on stdout, stderr, and the process exit
// code. They never link or call the implementation's internal functions.
//
// The binary is located at the canonical BINARY-LOCATION the cli-tool template
// mandates: project root, which is "../../zypper-declarative" relative to this
// test directory (independent_tests/<llm-name>/).
#ifndef ZD_TEST_HARNESS_HPP
#define ZD_TEST_HARNESS_HPP

#include <string>
#include <vector>
#include <functional>
#include <iostream>
#include <sstream>
#include <cstdlib>
#include <unistd.h>
#include <sys/wait.h>
#include <fcntl.h>

namespace zdtest {

// Canonical binary path per cli-tool template BINARY-LOCATION: project-root,
// expressed relative to independent_tests/<llm-name>/ -> "../../zypper-declarative".
inline const char* binary_path() { return "../../zypper-declarative"; }

struct RunResult {
    std::string out;
    std::string err;
    int code = -1;
};

// Run the binary with the given args (argv[1..]); capture stdout, stderr, exit.
inline RunResult run(const std::vector<std::string>& args) {
    int outpipe[2];
    int errpipe[2];
    if (pipe(outpipe) != 0 || pipe(errpipe) != 0) {
        return RunResult{"", "pipe failed", -1};
    }
    pid_t pid = fork();
    if (pid < 0) {
        return RunResult{"", "fork failed", -1};
    }
    if (pid == 0) {
        // child
        dup2(outpipe[1], STDOUT_FILENO);
        dup2(errpipe[1], STDERR_FILENO);
        close(outpipe[0]); close(outpipe[1]);
        close(errpipe[0]); close(errpipe[1]);
        std::vector<char*> argv;
        std::string bp = binary_path();
        argv.push_back(const_cast<char*>(bp.c_str()));
        for (auto& a : args) argv.push_back(const_cast<char*>(a.c_str()));
        argv.push_back(nullptr);
        execv(bp.c_str(), argv.data());
        // exec failed
        _exit(127);
    }
    // parent
    close(outpipe[1]);
    close(errpipe[1]);
    RunResult r;
    char buf[4096];
    ssize_t n;
    while ((n = read(outpipe[0], buf, sizeof(buf))) > 0) r.out.append(buf, n);
    while ((n = read(errpipe[0], buf, sizeof(buf))) > 0) r.err.append(buf, n);
    close(outpipe[0]);
    close(errpipe[0]);
    int status = 0;
    waitpid(pid, &status, 0);
    if (WIFEXITED(status)) r.code = WEXITSTATUS(status);
    else r.code = -2;
    return r;
}

// --- assertion + test registry ---------------------------------------------

struct TestCase { std::string name; std::function<void()> fn; };

inline std::vector<TestCase>& registry() {
    static std::vector<TestCase> r;
    return r;
}

inline int& failures() { static int f = 0; return f; }
inline std::string& current_test() { static std::string s; return s; }

struct Registrar {
    Registrar(const std::string& name, std::function<void()> fn) {
        registry().push_back({name, std::move(fn)});
    }
};

#define ZD_CONCAT_(a,b) a##b
#define ZD_CONCAT(a,b) ZD_CONCAT_(a,b)
#define TEST(name) \
    static void name(); \
    static ::zdtest::Registrar ZD_CONCAT(reg_, name)(#name, name); \
    static void name()

inline void fail(const std::string& msg) {
    failures()++;
    std::cerr << "  FAIL [" << current_test() << "]: " << msg << "\n";
}

inline void check(bool cond, const std::string& msg) {
    if (!cond) fail(msg);
}

inline void expect_eq_int(int got, int want, const std::string& what) {
    if (got != want) {
        std::ostringstream os;
        os << what << ": got " << got << " want " << want;
        fail(os.str());
    }
}

inline bool contains(const std::string& hay, const std::string& needle) {
    return hay.find(needle) != std::string::npos;
}

inline void expect_contains(const std::string& hay, const std::string& needle,
                            const std::string& what) {
    if (!contains(hay, needle)) {
        fail(what + ": expected to contain \"" + needle + "\" but got: " + hay);
    }
}

inline void expect_not_contains(const std::string& hay, const std::string& needle,
                                const std::string& what) {
    if (contains(hay, needle)) {
        fail(what + ": expected NOT to contain \"" + needle + "\" but got: " + hay);
    }
}

inline int run_all() {
    int n = 0;
    for (auto& tc : registry()) {
        current_test() = tc.name;
        int before = failures();
        tc.fn();
        ++n;
        if (failures() == before)
            std::cout << "ok   " << tc.name << "\n";
        else
            std::cout << "FAIL " << tc.name << "\n";
    }
    std::cout << "\n" << n << " tests, " << failures() << " failures\n";
    return failures() == 0 ? 0 : 1;
}

} // namespace zdtest

#endif // ZD_TEST_HARNESS_HPP
