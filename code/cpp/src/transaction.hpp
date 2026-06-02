// generated from spec: zypper-declarative.spec.md sha256:1641bb4413b82fecb081125067107bd5a4e30a8393edc778ead646207d68da5e
//
// Transaction binding and convergence domains. acquire-transaction-context
// resolves auto|external|internal and yields a context with a writable root,
// layered on libsnapper. The convergence path (packages/files/units) is
// identical regardless of binding. write-applied-record writes the ledger.
#ifndef ZD_TRANSACTION_HPP
#define ZD_TRANSACTION_HPP

#include <optional>
#include <string>

#include "command_runner.hpp"
#include "diagnostic.hpp"
#include "types.hpp"

namespace zd {

// acquire-transaction-context. Under external, requires a writable
// new-generation root to be present; under internal, opens a snapshot
// transaction through the snapshot machinery.
Result<TransactionContext> acquire_transaction_context(TransactionMode mode,
                                                       const CommandRunner& runner);

// converge-packages: returns the fully-resolved installed set (the lock).
Result<PackagesScope> converge_packages(const TransactionContext& ctx,
                                        const Diff& diff,
                                        const CommandRunner& runner);

// converge-files: write files_write, delete files_delete (regular files only in
// v1), excluding RPM-owned paths, the keep-list, and /etc/etc.syncpoint.
std::optional<Diagnostic> converge_files(const TransactionContext& ctx,
                                         const Diff& diff,
                                         const std::optional<std::string>& content_store);

// converge-units: apply declared enablement offline against ctx.root.
std::optional<Diagnostic> converge_units(const TransactionContext& ctx,
                                         const Diff& diff,
                                         const CommandRunner& runner);

// write-applied-record: construct an AppliedRecord from desired + resolved
// packages, serialise as canonical JSON, write into the context, stamp userdata.
std::optional<Diagnostic> write_applied_record(const TransactionContext& ctx,
                                               const Manifest& desired,
                                               const std::string& desired_sha256,
                                               const PackagesScope& resolved);

}  // namespace zd

#endif  // ZD_TRANSACTION_HPP
