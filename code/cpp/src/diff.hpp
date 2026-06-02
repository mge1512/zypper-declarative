// generated from spec: zypper-declarative.spec.md sha256:1641bb4413b82fecb081125067107bd5a4e30a8393edc778ead646207d68da5e
//
// Pure comparison behaviours: compute-intent-diff (desired vs applied) and
// compute-drift (actual vs reference). Neither performs any I/O.
#ifndef ZD_DIFF_HPP
#define ZD_DIFF_HPP

#include <set>
#include <string>

#include "types.hpp"

namespace zd {

// A keep-list: paths that compute-drift and converge-files must never report
// or delete. Empty by default.
using KeepList = std::set<std::string>;

// compute-intent-diff: changes from applied (old) to desired (new), scope by
// scope. A scope absent in desired yields no change for that scope.
Diff compute_intent_diff(const Manifest& desired, const Manifest& applied);

// compute-drift: actual vs reference (a desired manifest OR an applied record;
// same schema). Type is part of a config file's identity.
DriftReport compute_drift(const Manifest& actual, const Manifest& reference,
                          const KeepList& keep_list);

}  // namespace zd

#endif  // ZD_DIFF_HPP
