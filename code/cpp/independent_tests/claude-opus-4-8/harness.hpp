// generated from spec: zypper-declarative.spec.md sha256:aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3
// tests by: claude-opus-4-8
//
// Minimal black-box test harness for the zypper-declarative CLI binary.
//
// Methodology (per PCD prompt, test methodology for cli-tool deployments):
// these tests are BLACK-BOX. They invoke the built binary as a subprocess via
// fork/execvp, capture stdout/stderr/exit-code, and assert on the observable
// result. They do NOT link, import, or call any internal implementation
// function. The binary under test lives at the project root, i.e. two
// directories up from this test directory: "../../zypper-declarative"
// (per the cli-tool template BINARY-LOCATION: project-root constraint).
#ifndef ZD_TEST_HARNESS_HPP
#define ZD_TEST_HARNESS_HPP

#include <string>
#include <vector>
#include <cstdio>
#include <cstdlib>
#include <cerrno>
#include <iostream>
#include <functional>
#include <unistd.h>
#include <sys/wait.h>
#include <fcntl.h>
#include <poll.h>

namespace zdtest {

// Canonical path to the binary under test, per BINARY-LOCATION: project-root.
inline const char* binary_path() { return "../../zypper-declarative"; }

struct RunResult {
    std::string out;
    std::string err;
    int code = -1;
};

// Run the binary with the given args (argv[0] is set to the binary path
// automatically). stdin is closed. Captures stdout and stderr separately.
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
        int devnull = open("/dev/null", O_RDONLY);
        if (devnull >= 0) { dup2(devnull, STDIN_FILENO); close(devnull); }
        std::vector<char*> argv;
        std::string path = binary_path();
        argv.push_back(const_cast<char*>(path.c_str()));
        for (auto& a : args) argv.push_back(const_cast<char*>(a.c_str()));
        argv.push_back(nullptr);
        execv(path.c_str(), argv.data());
        // exec failed
        _exit(127);
    }
    // parent
    close(outpipe[1]);
    close(errpipe[1]);
    std::string out, err;
    char buf[4096];
    // Drain stdout and stderr CONCURRENTLY via poll(). Reading one stream to
    // EOF before the other deadlocks whenever the child fills the unread
    // stream's pipe buffer (e.g. `describe scope=full` emits a large stdout
    // manifest plus many stderr warnings). Non-blocking + poll avoids that.
    int ofd = outpipe[0], efd = errpipe[0];
    fcntl(ofd, F_SETFL, O_NONBLOCK);
    fcntl(efd, F_SETFL, O_NONBLOCK);
    bool oopen = true, eopen = true;
    while (oopen || eopen) {
        struct pollfd fds[2];
        int nf = 0;
        int oidx = -1, eidx = -1;
        if (oopen) { fds[nf].fd = ofd; fds[nf].events = POLLIN; oidx = nf; ++nf; }
        if (eopen) { fds[nf].fd = efd; fds[nf].events = POLLIN; eidx = nf; ++nf; }
        int pr = poll(fds, nf, -1);
        if (pr < 0) break;
        if (oidx >= 0 && (fds[oidx].revents & (POLLIN | POLLHUP | POLLERR))) {
            ssize_t n = read(ofd, buf, sizeof(buf));
            if (n > 0) out.append(buf, n);
            else if (n == 0) oopen = false;
            else if (errno != EAGAIN && errno != EINTR) oopen = false;
        }
        if (eidx >= 0 && (fds[eidx].revents & (POLLIN | POLLHUP | POLLERR))) {
            ssize_t n = read(efd, buf, sizeof(buf));
            if (n > 0) err.append(buf, n);
            else if (n == 0) eopen = false;
            else if (errno != EAGAIN && errno != EINTR) eopen = false;
        }
    }
    close(outpipe[0]);
    close(errpipe[0]);
    int status = 0;
    waitpid(pid, &status, 0);
    RunResult r;
    r.out = out;
    r.err = err;
    r.code = WIFEXITED(status) ? WEXITSTATUS(status) : -1;
    return r;
}

// ----------------------------------------------------------------------
// Tiny assertion framework
// ----------------------------------------------------------------------
struct TestCase {
    std::string name;
    std::function<void()> fn;
};

inline std::vector<TestCase>& registry() {
    static std::vector<TestCase> r;
    return r;
}

struct Registrar {
    Registrar(const std::string& name, std::function<void()> fn) {
        registry().push_back(TestCase{name, std::move(fn)});
    }
};

inline int g_failures = 0;
inline std::string g_current;

inline void fail(const std::string& msg) {
    ++g_failures;
    std::cerr << "  FAIL [" << g_current << "]: " << msg << "\n";
}

inline bool contains(const std::string& hay, const std::string& needle) {
    return hay.find(needle) != std::string::npos;
}

#define ZD_TEST(name) \
    static void name(); \
    static ::zdtest::Registrar reg_##name(#name, name); \
    static void name()

#define ZD_EXPECT_EQ(a, b) do { auto _a=(a); auto _b=(b); if (!((_a)==(_b))) { \
    std::ostringstream _o; _o << "expected " #a " == " #b " (got " << _a << " vs " << _b << ")"; \
    ::zdtest::fail(_o.str()); } } while(0)

#define ZD_EXPECT_CODE(res, expected) do { if ((res).code != (expected)) { \
    std::ostringstream _o; _o << "exit code: expected " << (expected) << " got " << (res).code \
    << " [stderr: " << (res).err << "]"; ::zdtest::fail(_o.str()); } } while(0)

#define ZD_EXPECT_CONTAINS(s, sub) do { if (!::zdtest::contains((s),(sub))) { \
    std::ostringstream _o; _o << "expected to contain \"" << (sub) << "\" but was: " << (s); \
    ::zdtest::fail(_o.str()); } } while(0)

#define ZD_EXPECT_NOT_CONTAINS(s, sub) do { if (::zdtest::contains((s),(sub))) { \
    std::ostringstream _o; _o << "expected NOT to contain \"" << (sub) << "\" but was: " << (s); \
    ::zdtest::fail(_o.str()); } } while(0)

#define ZD_EXPECT_TRUE(c) do { if (!(c)) { ::zdtest::fail("expected true: " #c); } } while(0)

inline int run_all() {
    int run = 0;
    for (auto& tc : registry()) {
        g_current = tc.name;
        int before = g_failures;
        try {
            tc.fn();
        } catch (const std::exception& e) {
            fail(std::string("exception: ") + e.what());
        }
        ++run;
        if (g_failures == before)
            std::cout << "  ok   " << tc.name << "\n";
    }
    std::cout << "\n" << run << " tests run, " << g_failures << " failures\n";
    return g_failures == 0 ? 0 : 1;
}

} // namespace zdtest

#endif
