// generated from spec: zypper-declarative.spec.md sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
// tests by: claude-opus-4-8
//
// Minimal black-box test harness for the zypper-declarative CLI binary.
// The harness invokes the built binary as a subprocess (per the cli-tool
// deployment template's BINARY-LOCATION constraint: ../../<binary-name>,
// i.e. two directories up from independent_tests/<llm-name>/ to the project
// root) and asserts on stdout, stderr and exit code. It NEVER links the
// implementation's internals.
#ifndef ZD_TEST_HARNESS_HPP
#define ZD_TEST_HARNESS_HPP

#include <array>
#include <cstdio>
#include <cstdlib>
#include <functional>
#include <iostream>
#include <string>
#include <sys/wait.h>
#include <unistd.h>
#include <vector>

namespace zdtest {

// The canonical path to the binary under test, relative to this test
// directory, per the cli-tool template's BINARY-LOCATION: project-root.
inline const char* binary_path() { return "../../zypper-declarative"; }

struct RunResult {
    std::string out;
    std::string err;
    int code = 0;
};

// Run the binary under test with the given argv (not including the program
// name) and an optional stdin payload. Captures stdout and stderr separately
// and the exit code. Uses fork/exec with pipes; does not go through a shell.
inline RunResult run(const std::vector<std::string>& args,
                     const std::string& stdin_data = std::string()) {
    int outpipe[2];
    int errpipe[2];
    int inpipe[2];
    if (pipe(outpipe) != 0 || pipe(errpipe) != 0 || pipe(inpipe) != 0) {
        std::perror("pipe");
        std::exit(70);
    }
    pid_t pid = fork();
    if (pid < 0) {
        std::perror("fork");
        std::exit(70);
    }
    if (pid == 0) {
        // child
        dup2(inpipe[0], STDIN_FILENO);
        dup2(outpipe[1], STDOUT_FILENO);
        dup2(errpipe[1], STDERR_FILENO);
        close(inpipe[0]);
        close(inpipe[1]);
        close(outpipe[0]);
        close(outpipe[1]);
        close(errpipe[0]);
        close(errpipe[1]);
        std::vector<char*> argv;
        argv.push_back(const_cast<char*>(binary_path()));
        for (const auto& a : args) argv.push_back(const_cast<char*>(a.c_str()));
        argv.push_back(nullptr);
        execv(binary_path(), argv.data());
        // exec failed
        std::perror("execv");
        _exit(127);
    }
    // parent
    close(inpipe[0]);
    close(outpipe[1]);
    close(errpipe[1]);
    if (!stdin_data.empty()) {
        ssize_t w = write(inpipe[1], stdin_data.data(), stdin_data.size());
        (void)w;
    }
    close(inpipe[1]);

    RunResult r;
    auto drain = [](int fd, std::string& dst) {
        std::array<char, 4096> buf;
        ssize_t n;
        while ((n = read(fd, buf.data(), buf.size())) > 0)
            dst.append(buf.data(), static_cast<size_t>(n));
    };
    drain(outpipe[0], r.out);
    drain(errpipe[0], r.err);
    close(outpipe[0]);
    close(errpipe[0]);

    int status = 0;
    waitpid(pid, &status, 0);
    if (WIFEXITED(status))
        r.code = WEXITSTATUS(status);
    else
        r.code = -1;
    return r;
}

inline bool contains(const std::string& haystack, const std::string& needle) {
    return haystack.find(needle) != std::string::npos;
}

// --- tiny test registry -------------------------------------------------
struct Case {
    std::string name;
    std::function<void()> fn;
};

inline std::vector<Case>& registry() {
    static std::vector<Case> r;
    return r;
}

inline int& failure_count() {
    static int f = 0;
    return f;
}

struct Registrar {
    Registrar(const std::string& name, std::function<void()> fn) {
        registry().push_back({name, std::move(fn)});
    }
};

struct AssertionFailure {
    std::string msg;
};

inline void check(bool cond, const std::string& msg) {
    if (!cond) throw AssertionFailure{msg};
}

inline int run_all() {
    int failed = 0;
    int passed = 0;
    for (auto& c : registry()) {
        try {
            c.fn();
            std::cout << "PASS " << c.name << "\n";
            ++passed;
        } catch (const AssertionFailure& f) {
            std::cout << "FAIL " << c.name << ": " << f.msg << "\n";
            ++failed;
        } catch (const std::exception& e) {
            std::cout << "FAIL " << c.name << ": unexpected exception: "
                      << e.what() << "\n";
            ++failed;
        }
    }
    std::cout << "\n" << passed << " passed, " << failed << " failed, "
              << registry().size() << " total\n";
    return failed == 0 ? 0 : 1;
}

}  // namespace zdtest

#define ZD_CONCAT_(a, b) a##b
#define ZD_CONCAT(a, b) ZD_CONCAT_(a, b)
#define ZD_TEST(name)                                                       \
    static void name();                                                     \
    static ::zdtest::Registrar ZD_CONCAT(reg_, name)(#name, name);          \
    static void name()

#endif  // ZD_TEST_HARNESS_HPP
