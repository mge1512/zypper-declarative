// generated from spec: zypper-declarative.spec.md sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
//
// CLI dispatch: key=value option parsing, the bare-word global commands
// (version, help), bare invocation, and routing to the verb handlers. POSIX
// --flag style is not used for options; --version/--help/-h are tolerated
// aliases only.
#ifndef ZD_CLI_HPP
#define ZD_CLI_HPP

#include <vector>
#include <string>

#include "command_runner.hpp"

namespace zd {

// Entry point for dispatch. argv excludes the program name. Returns the
// process exit code.
int dispatch(const std::vector<std::string>& argv, const CommandRunner& runner);

}  // namespace zd

#endif  // ZD_CLI_HPP
