// generated from spec: zypper-declarative.spec.md sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
//
// compute-intent-diff and compute-drift. Both are pure functions over
// in-memory Manifest values; they perform no filesystem, rpmdb, or process
// I/O. The keep-list (paths never reported or deleted) is passed in.
#ifndef ZD_DIFF_HPP
#define ZD_DIFF_HPP

#include <set>
#include <string>

#include "types.hpp"

namespace zd {

// compute-intent-diff: desired vs applied, scope by scope.
Diff compute_intent_diff(const Manifest& desired, const AppliedRecord& applied);

// compute-drift: actual vs reference, scope by scope on identity fields.
// keep_list paths and /etc/etc.syncpoint never appear in files_extra.
DriftReport compute_drift(const Manifest& actual, const AppliedRecord& reference,
                          const std::set<std::string>& keep_list);

}  // namespace zd

#endif  // ZD_DIFF_HPP
