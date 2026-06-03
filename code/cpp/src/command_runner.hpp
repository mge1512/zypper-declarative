// generated from spec: zypper-declarative.spec.md sha256:aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3
#pragma once

#include <string>
#include <vector>
#include <map>

namespace zd {

struct CommandResult {
    std::string out;
    std::string err;
    int code = 0;
};

// Abstract command runner seam (INTERFACES: a runtime command surface).
class CommandRunner {
public:
    virtual ~CommandRunner() = default;
    virtual CommandResult run(const std::string& cmd,
                              const std::vector<std::string>& args) const = 0;
};

// Production runner: fork/execvp with separate stdout/stderr pipes, fixed PATH
// in the child, no shell. A non-zero exit is returned in `code`, never thrown
// (some tools report "differences found" with a non-zero exit that the caller
// must interpret as data, not failure).
class OSCommandRunner : public CommandRunner {
public:
    CommandResult run(const std::string& cmd,
                      const std::vector<std::string>& args) const override;
};

// Test double for unit tests; not used by the production code paths.
class FakeCommandRunner : public CommandRunner {
public:
    std::map<std::string, CommandResult> responses;
    CommandResult run(const std::string& cmd,
                      const std::vector<std::string>&) const override;
};

}  // namespace zd
