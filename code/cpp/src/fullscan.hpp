// generated from spec: zypper-declarative.spec.md sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
//
// Out-of-/etc integrity scan (scope=full): walks /usr, the usr-merge roots,
// and /boot under a root, classifying each entry by its own type without
// following symlinks, and emits two observational scopes:
//   changed_managed_files  packaged entries differing from the package baseline
//   unmanaged_files        entries no installed package owns
// Excludes /etc, /opt, and the virtual/runtime/mutable-data trees, honours the
// keep-list, and does not cross into other filesystem mounts.
#ifndef ZD_FULLSCAN_HPP
#define ZD_FULLSCAN_HPP

#include <set>
#include <string>
#include <vector>

#include "types.hpp"

namespace zd {

bool full_scan(const std::string& root, bool on_unreadable_error,
               const std::set<std::string>& keep_list,
               ScopeWrapper<ManagedBaselineRecord>& changed,
               ScopeWrapper<UnmanagedFileRecord>& unmanaged,
               std::vector<Diagnostic>& diags, std::string& err);

}  // namespace zd

#endif  // ZD_FULLSCAN_HPP
