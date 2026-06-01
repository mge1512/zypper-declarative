// generated from spec: zypper-declarative.spec.md sha256:f2cc80627e483a48bb8411d297711bc5f6c6e74c28dbf0dafc8fe7bd8817251e
//
// diff.hpp -- compute-intent-diff and compute-drift, both pure comparisons
// (no filesystem, rpmdb, or process I/O), and the keep-list helper.
#ifndef ZD_DIFF_HPP
#define ZD_DIFF_HPP

#include "types.hpp"
#include <set>
#include <string>

namespace zd {

// A keep-list: paths describe/drift/converge must never report or delete.
using KeepList = std::set<std::string>;

// BEHAVIOR/INTERNAL: compute-intent-diff (desired_new versus applied_old).
Diff compute_intent_diff(const Manifest& desired, const AppliedRecord& applied);

// BEHAVIOR/INTERNAL: compute-drift (actual versus declaration), keep-list aware.
DriftReport compute_drift(const Manifest& actual, const AppliedRecord& reference,
                          const KeepList& keep);

// Load a keep-list file (one path per line, '#' comments). Missing file = empty.
KeepList load_keep_list(const std::string& path);

} // namespace zd

#endif // ZD_DIFF_HPP
