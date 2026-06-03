// generated from spec: zypper-declarative.spec.md sha256:aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3
// tests by: gemini-3-5-flash
#include "test_utils.hpp"
#include <atomic>
#include <chrono>
#include <fcntl.h>
#include <unistd.h>
#include <sys/stat.h>

std::vector<TestInfo> g_test_results;
std::vector<TestCase> g_test_cases;

TempDir::TempDir() {
    auto temp_base = fs::temp_directory_path();
    static std::atomic<size_t> counter{0};
    auto now = std::chrono::system_clock::now().time_since_epoch().count();
    auto unique_name = "zd_test_" + std::to_string(now) + "_" + std::to_string(counter++);
    path = temp_base / unique_name;
    std::error_code ec;
    fs::create_directories(path, ec);
}

TempDir::~TempDir() {
    std::error_code ec;
    fs::remove_all(path, ec);
}

bool is_root() {
    return ::geteuid() == 0;
}

void write_file(const fs::path& p, const std::string& content) {
    fs::create_directories(p.parent_path());
    std::ofstream f(p, std::ios::binary);
    if (f.is_open()) {
        f.write(content.data(), content.size());
    }
}

std::string read_file(const fs::path& p) {
    std::ifstream f(p, std::ios::binary);
    if (!f.is_open()) return "";
    std::ostringstream ss;
    ss << f.rdbuf();
    return ss.str();
}

void register_test_result(const std::string& name, const std::string& file, int line, bool failed, bool skipped, const std::string& message) {
    g_test_results.push_back({name, file, line, failed, skipped, message});
}

CommandResult run_command(const std::vector<std::string>& args) {
    if (args.empty()) {
        return {"", "Empty args", -1};
    }

    std::vector<char*> argv;
    for (const auto& arg : args) {
        argv.push_back(const_cast<char*>(arg.c_str()));
    }
    argv.push_back(nullptr);

    // Create unique temp files for stdout and stderr to avoid deadlocks or shell injection
    char stdout_tpl[] = "/tmp/zd_stdout_XXXXXX";
    char stderr_tpl[] = "/tmp/zd_stderr_XXXXXX";
    int stdout_fd = mkstemp(stdout_tpl);
    int stderr_fd = mkstemp(stderr_tpl);

    if (stdout_fd < 0 || stderr_fd < 0) {
        if (stdout_fd >= 0) { close(stdout_fd); unlink(stdout_tpl); }
        if (stderr_fd >= 0) { close(stderr_fd); unlink(stderr_tpl); }
        return {"", "Failed to create temp files", -1};
    }

    pid_t pid = fork();
    if (pid == 0) {
        // Child process
        dup2(stdout_fd, STDOUT_FILENO);
        dup2(stderr_fd, STDERR_FILENO);
        close(stdout_fd);
        close(stderr_fd);

        // Reset signal handlers inside the child to default
        signal(SIGTERM, SIG_DFL);
        signal(SIGINT, SIG_DFL);

        execvp(argv[0], argv.data());
        // If execvp fails, exit with 127
        _exit(127);
    } else if (pid > 0) {
        // Parent process
        close(stdout_fd);
        close(stderr_fd);

        int status = 0;
        waitpid(pid, &status, 0);

        std::string stdout_data = read_file(stdout_tpl);
        std::string stderr_data = read_file(stderr_tpl);

        unlink(stdout_tpl);
        unlink(stderr_tpl);

        int exit_code = -1;
        if (WIFEXITED(status)) {
            exit_code = WEXITSTATUS(status);
        } else if (WIFSIGNALED(status)) {
            exit_code = 128 + WTERMSIG(status);
        }

        return {stdout_data, stderr_data, exit_code};
    } else {
        close(stdout_fd);
        close(stderr_fd);
        unlink(stdout_tpl);
        unlink(stderr_tpl);
        return {"", "Failed to fork", -1};
    }
}
