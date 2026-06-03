// generated from spec: zypper-declarative.spec.md sha256:aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3
#pragma once

#include <ostream>

#include "config.hpp"
#include "command_runner.hpp"

namespace zd {

// The CLI verbs. Each returns the process exit code (0/1/2) and writes
// diagnostics to err and normal output to out. The CommandRunner is the
// production OSCommandRunner in main, injectable for tests.
int verb_apply(const Config& cfg, const CommandRunner& runner, std::ostream& out, std::ostream& err);
int verb_diff(const Config& cfg, const CommandRunner& runner, std::ostream& out, std::ostream& err);
int verb_verify(const Config& cfg, const CommandRunner& runner, std::ostream& out, std::ostream& err);
int verb_status(const Config& cfg, const CommandRunner& runner, std::ostream& out, std::ostream& err);
int verb_describe(const Config& cfg, const CommandRunner& runner, std::ostream& out, std::ostream& err);
int verb_init(const Config& cfg, const CommandRunner& runner, std::ostream& out, std::ostream& err);

}  // namespace zd
