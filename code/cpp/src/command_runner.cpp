// generated from spec: zypper-declarative.spec.md sha256:1641bb4413b82fecb081125067107bd5a4e30a8393edc778ead646207d68da5e
#include "command_runner.hpp"

#include <unistd.h>
#include <sys/wait.h>
#include <fcntl.h>

#include <array>
#include <cstring>

namespace zd {

CommandResult OSCommandRunner::run(const std::string& cmd,
                                   const std::vector<std::string>& args) const {
    CommandResult result;
    int outpipe[2];
    int errpipe[2];
    if (pipe(outpipe) != 0 || pipe(errpipe) != 0) {
        result.spawn_failed = true;
        result.code = -1;
        result.err = "pipe failed";
        return result;
    }
    pid_t pid = fork();
    if (pid < 0) {
        result.spawn_failed = true;
        result.code = -1;
        result.err = "fork failed";
        return result;
    }
    if (pid == 0) {
        // child: redirect stdout/stderr to pipes, close stdin to /dev/null
        dup2(outpipe[1], STDOUT_FILENO);
        dup2(errpipe[1], STDERR_FILENO);
        close(outpipe[0]); close(outpipe[1]);
        close(errpipe[0]); close(errpipe[1]);
        int devnull = open("/dev/null", O_RDONLY);
        if (devnull >= 0) { dup2(devnull, STDIN_FILENO); close(devnull); }

        // Fixed PATH in the child only (no env-var control of behaviour).
        setenv("PATH", "/sbin:/bin:/usr/bin:/usr/sbin", 1);

        std::vector<char*> argv;
        argv.push_back(const_cast<char*>(cmd.c_str()));
        for (const auto& a : args) argv.push_back(const_cast<char*>(a.c_str()));
        argv.push_back(nullptr);
        execvp(cmd.c_str(), argv.data());
        _exit(127);  // exec failed
    }

    // parent
    close(outpipe[1]);
    close(errpipe[1]);
    std::array<char, 4096> buf;
    ssize_t n;
    while ((n = read(outpipe[0], buf.data(), buf.size())) > 0)
        result.out.append(buf.data(), static_cast<size_t>(n));
    while ((n = read(errpipe[0], buf.data(), buf.size())) > 0)
        result.err.append(buf.data(), static_cast<size_t>(n));
    close(outpipe[0]);
    close(errpipe[0]);

    int status = 0;
    waitpid(pid, &status, 0);
    if (WIFEXITED(status)) {
        result.code = WEXITSTATUS(status);
        if (result.code == 127) result.spawn_failed = true;
    } else {
        result.code = -1;
        result.spawn_failed = true;
    }
    return result;
}

}  // namespace zd
