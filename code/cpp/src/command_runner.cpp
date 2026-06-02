// generated from spec: zypper-declarative.spec.md sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03

#include "command_runner.hpp"

#include <array>
#include <cstdlib>
#include <unistd.h>
#include <sys/wait.h>

namespace zd {

CommandResult OSCommandRunner::run(const std::string& cmd,
                                   const std::vector<std::string>& args) const {
    int outpipe[2];
    int errpipe[2];
    if (pipe(outpipe) != 0 || pipe(errpipe) != 0) {
        return CommandResult{"", "pipe() failed", 127};
    }
    pid_t pid = fork();
    if (pid < 0) {
        return CommandResult{"", "fork() failed", 127};
    }
    if (pid == 0) {
        // child
        dup2(outpipe[1], STDOUT_FILENO);
        dup2(errpipe[1], STDERR_FILENO);
        close(outpipe[0]);
        close(outpipe[1]);
        close(errpipe[0]);
        close(errpipe[1]);
        // Fixed PATH in the child only (no environment-variable control of the
        // parent's behaviour).
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
    CommandResult r;
    std::array<char, 4096> buf;
    ssize_t n;
    while ((n = read(outpipe[0], buf.data(), buf.size())) > 0)
        r.out.append(buf.data(), static_cast<size_t>(n));
    while ((n = read(errpipe[0], buf.data(), buf.size())) > 0)
        r.err.append(buf.data(), static_cast<size_t>(n));
    close(outpipe[0]);
    close(errpipe[0]);
    int status = 0;
    waitpid(pid, &status, 0);
    r.code = WIFEXITED(status) ? WEXITSTATUS(status) : 127;
    return r;
}

}  // namespace zd
