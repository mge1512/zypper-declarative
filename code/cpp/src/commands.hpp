// generated from spec: zypper-declarative.spec.md sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
//
// The CLI verbs (apply, diff, verify, status, describe). Each returns an exit
// code and writes diagnostics to stderr and normal output to stdout. The verb
// layer is the only place that maps an internal Diagnostic to an exit code.
#ifndef ZD_COMMANDS_HPP
#define ZD_COMMANDS_HPP

#include "command_runner.hpp"
#include "config.hpp"

namespace zd {

int cmd_apply(const Config& cfg, const CommandRunner& runner);
int cmd_diff(const Config& cfg, const CommandRunner& runner);
int cmd_verify(const Config& cfg, const CommandRunner& runner);
int cmd_status(const Config& cfg, const CommandRunner& runner);
int cmd_describe(const Config& cfg, const CommandRunner& runner);

}  // namespace zd

#endif  // ZD_COMMANDS_HPP
