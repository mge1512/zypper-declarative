// generated from spec: zypper-declarative.spec.md sha256:aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3
//
// The CLI verbs (apply, diff, verify, status, describe, init). Each verb maps
// internal-behaviour Diagnostics to the spec's exit codes (0/1/2). The verbs
// are the only layer that decides the process exit status.
#ifndef ZD_COMMANDS_HPP
#define ZD_COMMANDS_HPP

#include <map>
#include <optional>
#include <string>

#include "command_runner.hpp"

namespace zd {

// Parsed invocation: a verb and the key=value options.
struct Invocation {
    std::string verb;
    std::map<std::string, std::string> options;  // key=value
};

// Each verb runs and returns the process exit code, writing stdout/stderr.
int cmd_apply(const Invocation& inv, const CommandRunner& runner);
int cmd_diff(const Invocation& inv, const CommandRunner& runner);
int cmd_verify(const Invocation& inv, const CommandRunner& runner);
int cmd_status(const Invocation& inv, const CommandRunner& runner);
int cmd_describe(const Invocation& inv, const CommandRunner& runner);
int cmd_init(const Invocation& inv, const CommandRunner& runner);

}  // namespace zd

#endif  // ZD_COMMANDS_HPP
