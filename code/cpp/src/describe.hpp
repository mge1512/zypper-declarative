// generated from spec: zypper-declarative.spec.md sha256:aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3
#pragma once

#include <set>
#include <string>
#include <vector>

#include "types.hpp"
#include "diagnostic.hpp"
#include "command_runner.hpp"

namespace zd {

// BEHAVIOR/INTERNAL: describe-actual-state. The single live-state reader.
// Reads the four declarable scopes under `root` and returns a Manifest.
// Under scope=full it also produces the changed_managed_files and
// unmanaged_files observational scopes. content_store, when set, captures the
// bytes of every emitted regular-file record. Unreadable sources are treated
// per on_unreadable.
struct DescribeResult {
    Manifest manifest;
    std::vector<Diagnostic> diagnostics;  // warnings under warn
    std::optional<Diagnostic> error;      // first unreadable source under error
    bool ok() const { return !error.has_value(); }
};

DescribeResult describe_actual_state(const std::string& root,
                                     OnUnreadable on_unreadable,
                                     ScanScope scope,
                                     const std::optional<std::string>& content_store,
                                     const std::set<std::string>& keep_list,
                                     const std::string& generator,
                                     const CommandRunner& runner);

}  // namespace zd
