// generated from spec: zypper-declarative.spec.md sha256:aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3
//
// describe-actual-state: the single live-state reader. Reads the four
// declarable scopes under a root and returns a Manifest in the shared schema.
// Under scope=full it additionally produces the two observational scopes. This
// is the only translation unit that talks to libzypp's rpmdb, reads
// /etc/zypp/repos.d, reads unit enablement, or walks /etc.
#ifndef ZD_ACTUAL_STATE_HPP
#define ZD_ACTUAL_STATE_HPP

#include <optional>
#include <string>
#include <vector>

#include "command_runner.hpp"
#include "diagnostic.hpp"
#include "diff.hpp"
#include "types.hpp"

namespace zd {

enum class OnUnreadable { Error, Warn };
enum class ScanScope { Etc, Full };

struct DescribeOptions {
    std::string root = "/";
    OnUnreadable on_unreadable = OnUnreadable::Error;
    ScanScope scope = ScanScope::Etc;
    std::optional<std::string> content_store;  // base path; if set, capture content
    KeepList keep_list;
};

struct DescribeResult {
    Manifest manifest;
    std::vector<Diagnostic> diagnostics;  // under warn: one per omitted scope
};

// describe-actual-state. On an unreadable source under OnUnreadable::Error,
// returns a Diagnostic (caller fails). Under warn, omits the scope, appends a
// diagnostic, and continues.
Result<DescribeResult> describe_actual_state(const DescribeOptions& opts,
                                             const CommandRunner& runner);

}  // namespace zd

#endif  // ZD_ACTUAL_STATE_HPP
