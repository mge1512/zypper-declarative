// generated from spec: zypper-declarative.spec.md sha256:aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3
#pragma once

#include <set>
#include <string>

#include "types.hpp"

namespace zd {

// BEHAVIOR/INTERNAL: compute-intent-diff. Desired vs applied, scope by scope.
// No filesystem access.
Diff compute_intent_diff(const Manifest& desired, const Manifest& applied);

// BEHAVIOR/INTERNAL: compute-drift. actual vs reference (a Manifest that may be
// a desired manifest or an applied record). Pure comparison, no I/O. The
// keep-list paths are excluded from files_extra.
DriftReport compute_drift(const Manifest& actual, const Manifest& reference,
                          const std::set<std::string>& keep_list);

}  // namespace zd
