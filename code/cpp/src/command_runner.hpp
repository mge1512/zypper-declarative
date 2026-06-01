// generated from spec: zypper-declarative.spec.md sha256:f2cc80627e483a48bb8411d297711bc5f6c6e74c28dbf0dafc8fe7bd8817251e
//
// command_runner.hpp -- the CommandRunner seam (abstract base + production
// OSCommandRunner + FakeCommandRunner test double). Some operations have no
// library API and are done by executing a command; a non-zero exit is returned
// in `code`, not thrown, because some tools report "differences found" with a
// non-zero exit the caller must interpret as data, not failure.
#ifndef ZD_COMMAND_RUNNER_HPP
#define ZD_COMMAND_RUNNER_HPP

#include <string>
#include <vector>
#include <map>

namespace zd {

struct CommandResult {
    std::string out;
    std::string err;
    int code = 0;
};

class CommandRunner {
public:
    virtual ~CommandRunner() = default;
    virtual CommandResult run(const std::string& cmd,
                              const std::vector<std::string>& args) const = 0;
};

// Production runner: fork/execvp with separate stdout/stderr pipes and a fixed
// PATH in the child. Never goes through a shell.
class OSCommandRunner : public CommandRunner {
public:
    CommandResult run(const std::string& cmd,
                      const std::vector<std::string>& args) const override;
};

// Test double: returns canned responses keyed by command name.
class FakeCommandRunner : public CommandRunner {
public:
    std::map<std::string, CommandResult> responses;
    CommandResult run(const std::string& cmd,
                      const std::vector<std::string>&) const override {
        auto it = responses.find(cmd);
        return it != responses.end() ? it->second : CommandResult{"", "", 0};
    }
};

} // namespace zd

#endif // ZD_COMMAND_RUNNER_HPP
