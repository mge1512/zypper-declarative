// generated from spec: zypper-declarative.spec.md sha256:f2cc80627e483a48bb8411d297711bc5f6c6e74c28dbf0dafc8fe7bd8817251e
//
// cli.hpp -- CLI dispatch: key=value parsing, the bare-word global commands
// (version, help) and their tolerated flag aliases, usage, and option/value
// validation. The entry point (main.cpp) contains only signal wiring and a call
// into run(); all dispatch logic lives here.
#ifndef ZD_CLI_HPP
#define ZD_CLI_HPP

#include <string>
#include <vector>

namespace zd {

// Run the CLI with argv[1..argc-1]. Returns the process exit code (0/1/2).
int run(const std::vector<std::string>& args);

// Usage text (printed to stdout for help/bare, to stderr for invocation errors).
std::string usage_text();

} // namespace zd

#endif // ZD_CLI_HPP
