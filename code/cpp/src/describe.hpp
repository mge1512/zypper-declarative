// generated from spec: zypper-declarative.spec.md sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
//
// describe-actual-state: the single live-state reader. Reads the four
// declarable scopes (packages, repositories, services, config_files) under a
// root and returns a Manifest in the shared schema. Optionally adds the two
// observational scopes under scope=full. This is the only TU that talks to
// libzypp's rpmdb, reads /etc/zypp/repos.d, reads unit enablement, or walks
// /etc; keep the library linkage concentrated here.
#ifndef ZD_DESCRIBE_HPP
#define ZD_DESCRIBE_HPP

#include <set>
#include <string>
#include <vector>

#include "command_runner.hpp"
#include "config.hpp"
#include "types.hpp"

namespace zd {

struct DescribeResult {
    bool ok = false;                 // false only under on_unreadable=error
    Manifest manifest;
    std::vector<Diagnostic> diagnostics;  // warn diagnostics
    Diagnostic error;                // set when !ok
};

// describe-actual-state on `root`. on_unreadable_error selects strict vs warn.
DescribeResult describe_actual_state(const std::string& root,
                                     bool on_unreadable_error, ScanScope scope,
                                     const std::set<std::string>& keep_list,
                                     const std::string& content_store,
                                     const CommandRunner& runner);

}  // namespace zd

#endif  // ZD_DESCRIBE_HPP
