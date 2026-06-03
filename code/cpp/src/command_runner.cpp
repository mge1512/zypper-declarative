// generated from spec: zypper-declarative.spec.md sha256:aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3
#include "command_runner.hpp"

#include <array>
#include <cstdlib>
#include <cstring>
#include <vector>

#include <poll.h>
#include <sys/types.h>
#include <sys/wait.h>
#include <unistd.h>

namespace zd {

CommandResult OSCommandRunner::run(const std::string& cmd,
                                   const std::vector<std::string>& args) const {
    CommandResult result;

    int out_pipe[2];
    int err_pipe[2];
    if (pipe(out_pipe) != 0 || pipe(err_pipe) != 0) {
        result.code = -1;
        result.err = "failed to create pipes";
        return result;
    }

    pid_t pid = fork();
    if (pid < 0) {
        close(out_pipe[0]); close(out_pipe[1]);
        close(err_pipe[0]); close(err_pipe[1]);
        result.code = -1;
        result.err = "fork failed";
        return result;
    }

    if (pid == 0) {
        // Child.
        dup2(out_pipe[1], STDOUT_FILENO);
        dup2(err_pipe[1], STDERR_FILENO);
        close(out_pipe[0]); close(out_pipe[1]);
        close(err_pipe[0]); close(err_pipe[1]);

        // Fixed, minimal PATH in the child only.
        setenv("PATH", "/sbin:/bin:/usr/bin:/usr/sbin", 1);

        std::vector<char*> argv;
        argv.push_back(const_cast<char*>(cmd.c_str()));
        for (const auto& a : args) argv.push_back(const_cast<char*>(a.c_str()));
        argv.push_back(nullptr);

        execvp(cmd.c_str(), argv.data());
        _exit(127);
    }

    // Parent: drain stdout and stderr concurrently to avoid pipe-buffer deadlock.
    close(out_pipe[1]);
    close(err_pipe[1]);

    struct pollfd fds[2];
    fds[0].fd = out_pipe[0]; fds[0].events = POLLIN;
    fds[1].fd = err_pipe[0]; fds[1].events = POLLIN;
    bool out_open = true, err_open = true;

    char buf[4096];
    while (out_open || err_open) {
        int nfds = 0;
        struct pollfd active[2];
        int idx_out = -1, idx_err = -1;
        if (out_open) { active[nfds] = fds[0]; idx_out = nfds; ++nfds; }
        if (err_open) { active[nfds] = fds[1]; idx_err = nfds; ++nfds; }
        if (nfds == 0) break;

        int pr = poll(active, nfds, -1);
        if (pr < 0) break;

        if (idx_out >= 0 && (active[idx_out].revents & (POLLIN | POLLHUP))) {
            ssize_t n = read(out_pipe[0], buf, sizeof(buf));
            if (n > 0) result.out.append(buf, static_cast<size_t>(n));
            else { close(out_pipe[0]); out_open = false; }
        }
        if (idx_err >= 0 && (active[idx_err].revents & (POLLIN | POLLHUP))) {
            ssize_t n = read(err_pipe[0], buf, sizeof(buf));
            if (n > 0) result.err.append(buf, static_cast<size_t>(n));
            else { close(err_pipe[0]); err_open = false; }
        }
    }
    if (out_open) close(out_pipe[0]);
    if (err_open) close(err_pipe[0]);

    int status = 0;
    waitpid(pid, &status, 0);
    if (WIFEXITED(status)) result.code = WEXITSTATUS(status);
    else if (WIFSIGNALED(status)) result.code = 128 + WTERMSIG(status);
    else result.code = -1;

    return result;
}

CommandResult FakeCommandRunner::run(const std::string& cmd,
                                     const std::vector<std::string>&) const {
    auto it = responses.find(cmd);
    return it != responses.end() ? it->second : CommandResult{"", "", 0};
}

}  // namespace zd
