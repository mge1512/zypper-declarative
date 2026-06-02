// generated from spec: zypper-declarative.spec.md sha256:aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3
//
// OSCommandRunner: executes an external command via fork/execvp with separate
// stdout/stderr capture and a clean exit status. Used for the few operations
// that have no in-process library API (offline systemctl enablement, the
// alternatives database query). A non-zero exit is returned in `code`, not
// thrown, because some tools report "differences found" with a non-zero exit
// that the caller must interpret as data, not failure.
#ifndef ZD_COMMAND_RUNNER_HPP
#define ZD_COMMAND_RUNNER_HPP

#include <string>
#include <vector>

namespace zd {

struct CommandResult {
    std::string out;
    std::string err;
    int code = 0;
    bool spawn_failed = false;  // true if the binary could not be executed
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

}  // namespace zd

#endif  // ZD_COMMAND_RUNNER_HPP
