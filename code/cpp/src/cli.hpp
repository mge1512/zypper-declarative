// generated from spec: zypper-declarative.spec.md sha256:aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3
#pragma once

#include <ostream>
#include <string>
#include <vector>

namespace zd {

// Top-level CLI dispatch. argv excludes the program name. Returns the process
// exit code. version/help/bare invocation are handled here; verbs delegate to
// the implementation modules.
int dispatch(const std::vector<std::string>& argv, std::ostream& out, std::ostream& err);

void print_usage(std::ostream& os);
void print_version(std::ostream& os);

}  // namespace zd
