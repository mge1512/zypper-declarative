// generated from spec: zypper-declarative.spec.md sha256:f2cc80627e483a48bb8411d297711bc5f6c6e74c28dbf0dafc8fe7bd8817251e
//
// commands.hpp -- the five CLI verbs (apply, diff, verify, status, describe).
// Each returns an ExitCode (0/1/2) and writes diagnostics to stderr, output to
// stdout, per the deployment template stream rules. The verb layer is the only
// place that maps results to a process exit code.
#ifndef ZD_COMMANDS_HPP
#define ZD_COMMANDS_HPP

#include "config.hpp"
#include "command_runner.hpp"
#include "describe.hpp"

namespace zd {

// Each verb takes the resolved Config and a CommandRunner / SystemReader seam.
// reader may be nullptr (synthetic-root / no live integration available).
int cmd_apply(const Config& cfg, const CommandRunner& runner, const SystemReader* reader);
int cmd_diff(const Config& cfg, const CommandRunner& runner, const SystemReader* reader);
int cmd_verify(const Config& cfg, const CommandRunner& runner, const SystemReader* reader);
int cmd_status(const Config& cfg, const CommandRunner& runner, const SystemReader* reader);
int cmd_describe(const Config& cfg, const CommandRunner& runner, const SystemReader* reader);

} // namespace zd

#endif // ZD_COMMANDS_HPP
