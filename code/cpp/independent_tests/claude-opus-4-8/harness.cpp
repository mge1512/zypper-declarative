// generated from spec: zypper-declarative.spec.md sha256:aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3
// tests by: claude-opus-4-8
#include "harness.hpp"

#include <atomic>
#include <chrono>
#include <cstdlib>
#include <iostream>
#include <stdexcept>

#include <fcntl.h>
#include <poll.h>
#include <sys/types.h>
#include <sys/wait.h>
#include <unistd.h>

struct SkipException : std::runtime_error {
    explicit SkipException(const std::string& m) : std::runtime_error(m) {}
};
struct FailException : std::runtime_error {
    explicit FailException(const std::string& m) : std::runtime_error(m) {}
};

std::vector<TestCase>& registry() {
    static std::vector<TestCase> r;
    return r;
}
Reg::Reg(const std::string& n, std::function<void()> f) { registry().push_back({n, std::move(f)}); }

void zd_assert(bool cond, const std::string& msg, const char* file, int line) {
    if (!cond) throw FailException(std::string(file) + ":" + std::to_string(line) + " " + msg);
}
void zd_skip(const std::string& reason) { throw SkipException(reason); }

TempDir::TempDir() {
    static std::atomic<unsigned> ctr{0};
    auto now = std::chrono::steady_clock::now().time_since_epoch().count();
    path = fs::temp_directory_path() /
           ("zd_co_" + std::to_string(now) + "_" + std::to_string(ctr++));
    std::error_code ec;
    fs::create_directories(path, ec);
}
TempDir::~TempDir() {
    std::error_code ec;
    fs::remove_all(path, ec);
}

void write_text(const fs::path& p, const std::string& content) {
    std::error_code ec;
    fs::create_directories(p.parent_path(), ec);
    std::ofstream f(p, std::ios::binary);
    f.write(content.data(), static_cast<std::streamsize>(content.size()));
}
std::string read_text(const fs::path& p) {
    std::ifstream f(p, std::ios::binary);
    if (!f.is_open()) return "";
    std::ostringstream ss;
    ss << f.rdbuf();
    return ss.str();
}

CmdResult run_zd(const std::vector<std::string>& args) {
    // Canonical binary path per the cli-tool BINARY-LOCATION constraint:
    // two directories up from independent_tests/<llm-name>/ is the project root.
    const std::string bin = "../../zypper-declarative";

    int out_pipe[2], err_pipe[2];
    if (pipe(out_pipe) || pipe(err_pipe)) return {"", "pipe failed", -1};

    pid_t pid = fork();
    if (pid < 0) return {"", "fork failed", -1};
    if (pid == 0) {
        dup2(out_pipe[1], STDOUT_FILENO);
        dup2(err_pipe[1], STDERR_FILENO);
        close(out_pipe[0]); close(out_pipe[1]);
        close(err_pipe[0]); close(err_pipe[1]);
        std::vector<char*> argv;
        argv.push_back(const_cast<char*>(bin.c_str()));
        for (const auto& a : args) argv.push_back(const_cast<char*>(a.c_str()));
        argv.push_back(nullptr);
        execv(bin.c_str(), argv.data());
        _exit(127);
    }
    close(out_pipe[1]);
    close(err_pipe[1]);

    CmdResult r;
    struct pollfd fds[2];
    fds[0].fd = out_pipe[0]; fds[0].events = POLLIN;
    fds[1].fd = err_pipe[0]; fds[1].events = POLLIN;
    bool o = true, e = true;
    char buf[4096];
    while (o || e) {
        struct pollfd active[2];
        int n = 0, io = -1, ie = -1;
        if (o) { active[n] = fds[0]; io = n; ++n; }
        if (e) { active[n] = fds[1]; ie = n; ++n; }
        if (n == 0) break;
        if (poll(active, n, -1) < 0) break;
        if (io >= 0 && (active[io].revents & (POLLIN | POLLHUP))) {
            ssize_t k = read(out_pipe[0], buf, sizeof(buf));
            if (k > 0) r.out.append(buf, static_cast<size_t>(k));
            else { close(out_pipe[0]); o = false; }
        }
        if (ie >= 0 && (active[ie].revents & (POLLIN | POLLHUP))) {
            ssize_t k = read(err_pipe[0], buf, sizeof(buf));
            if (k > 0) r.err.append(buf, static_cast<size_t>(k));
            else { close(err_pipe[0]); e = false; }
        }
    }
    if (o) close(out_pipe[0]);
    if (e) close(err_pipe[0]);
    int status = 0;
    waitpid(pid, &status, 0);
    if (WIFEXITED(status)) r.code = WEXITSTATUS(status);
    else if (WIFSIGNALED(status)) r.code = 128 + WTERMSIG(status);
    return r;
}

int main() {
    int passed = 0, failed = 0, skipped = 0;
    for (const auto& tc : registry()) {
        try {
            tc.fn();
            std::cout << "[PASS] " << tc.name << "\n";
            ++passed;
        } catch (const SkipException& s) {
            std::cout << "[SKIP] " << tc.name << " (" << s.what() << ")\n";
            ++skipped;
        } catch (const FailException& f) {
            std::cout << "[FAIL] " << tc.name << ": " << f.what() << "\n";
            ++failed;
        } catch (const std::exception& ex) {
            std::cout << "[FAIL] " << tc.name << ": exception " << ex.what() << "\n";
            ++failed;
        }
    }
    std::cout << "----\npassed=" << passed << " failed=" << failed
              << " skipped=" << skipped << "\n";
    return failed > 0 ? 1 : 0;
}
