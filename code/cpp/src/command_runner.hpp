// generated from spec: zypper-declarative.spec.md sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
//
// OSCommandRunner: executes external commands via fork/execvp with separate
// stdout/stderr pipes and a clean PATH. A non-zero exit is returned in the
// result, never thrown, because some tools report differences with a non-zero
// status the caller must interpret as data. Abstract CommandRunner allows a
// test double to be substituted without a live system.
#ifndef ZD_COMMAND_RUNNER_HPP
#define ZD_COMMAND_RUNNER_HPP

#include <map>
#include <string>
#include <vector>

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

class OSCommandRunner : public CommandRunner {
public:
    CommandResult run(const std::string& cmd,
                      const std::vector<std::string>& args) const override;
};

// Test double: returns a canned response keyed by command name.
class FakeCommandRunner : public CommandRunner {
public:
    std::map<std::string, CommandResult> responses;
    CommandResult run(const std::string& cmd,
                      const std::vector<std::string>&) const override {
        auto it = responses.find(cmd);
        return it != responses.end() ? it->second : CommandResult{"", "", 0};
    }
};

}  // namespace zd

#endif  // ZD_COMMAND_RUNNER_HPP
