// generated from spec: zypper-declarative.spec.md sha256:aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3
//
// CLI dispatch: key=value argument parsing, bare-word verb handling, the global
// version/help commands and their tolerated flag aliases, and the usage text.
#ifndef ZD_CLI_HPP
#define ZD_CLI_HPP

#include <string>
#include <vector>

#include "command_runner.hpp"

namespace zd {

// Parse args (argv[1..]) and dispatch to a verb. Returns the process exit code.
int run_cli(const std::vector<std::string>& args, const CommandRunner& runner);

// The usage text printed by help / bare invocation.
std::string usage_text();

// The version line printed by version / --version.
std::string version_text();

}  // namespace zd

#endif  // ZD_CLI_HPP
