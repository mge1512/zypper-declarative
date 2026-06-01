// generated from spec: zypper-declarative.spec.md sha256:f2cc80627e483a48bb8411d297711bc5f6c6e74c28dbf0dafc8fe7bd8817251e
//
// command_runner.cpp -- OSCommandRunner::run, implemented in full (never a stub).
#include "command_runner.hpp"

#include <unistd.h>
#include <sys/wait.h>
#include <cstring>
#include <vector>

namespace zd {

CommandResult OSCommandRunner::run(const std::string& cmd,
                                   const std::vector<std::string>& args) const {
    int outpipe[2];
    int errpipe[2];
    if (pipe(outpipe) != 0 || pipe(errpipe) != 0) {
        return CommandResult{"", "pipe creation failed", 127};
    }
    pid_t pid = fork();
    if (pid < 0) {
        return CommandResult{"", "fork failed", 127};
    }
    if (pid == 0) {
        // child
        dup2(outpipe[1], STDOUT_FILENO);
        dup2(errpipe[1], STDERR_FILENO);
        close(outpipe[0]); close(outpipe[1]);
        close(errpipe[0]); close(errpipe[1]);
        // Fixed, minimal PATH in the child only.
        setenv("PATH", "/sbin:/bin:/usr/bin:/usr/sbin", 1);
        std::vector<char*> argv;
        argv.push_back(const_cast<char*>(cmd.c_str()));
        for (const auto& a : args) argv.push_back(const_cast<char*>(a.c_str()));
        argv.push_back(nullptr);
        execvp(cmd.c_str(), argv.data());
        _exit(127); // exec failed
    }
    // parent
    close(outpipe[1]);
    close(errpipe[1]);
    CommandResult res;
    char buf[4096];
    ssize_t n;
    while ((n = read(outpipe[0], buf, sizeof(buf))) > 0) res.out.append(buf, n);
    while ((n = read(errpipe[0], buf, sizeof(buf))) > 0) res.err.append(buf, n);
    close(outpipe[0]);
    close(errpipe[0]);
    int status = 0;
    waitpid(pid, &status, 0);
    if (WIFEXITED(status)) res.code = WEXITSTATUS(status);
    else res.code = 128;
    return res;
}

} // namespace zd
