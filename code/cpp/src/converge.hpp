// generated from spec: zypper-declarative.spec.md sha256:aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3
#pragma once

#include <set>
#include <string>

#include "types.hpp"
#include "diagnostic.hpp"
#include "command_runner.hpp"

namespace zd {

// BEHAVIOR/INTERNAL: converge-packages. Applies the package portion of the
// intent diff inside ctx and returns the resolved packages scope (the lock).
Result<PackagesScope> converge_packages(const TransactionContext& ctx, const Diff& diff,
                                        const std::optional<std::string>& repo_lock,
                                        const CommandRunner& runner);

// BEHAVIOR/INTERNAL: converge-files. Writes files_write, deletes files_delete
// (excluding RPM-owned, keep-listed, and /etc/etc.syncpoint paths).
VoidResult converge_files(const TransactionContext& ctx, const Diff& diff,
                          const std::optional<std::string>& content_store,
                          const std::set<std::string>& keep_list);

// BEHAVIOR/INTERNAL: converge-units. Offline unit enablement against ctx.root.
VoidResult converge_units(const TransactionContext& ctx, const Diff& diff,
                          const CommandRunner& runner);

// BEHAVIOR/INTERNAL: write-applied-record. Constructs and serialises the
// applied record (canonical JSON) into ctx.root and stamps userdata.
VoidResult write_applied_record(const TransactionContext& ctx, const Manifest& desired,
                                const std::string& desired_sha256,
                                const PackagesScope& resolved);

}  // namespace zd
