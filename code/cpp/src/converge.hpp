// generated from spec: zypper-declarative.spec.md sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
//
// Convergence domains and the transaction binding. These are the mutating
// paths; acquire-transaction-context resolves auto|external|internal and
// yields a context with a writable root. The convergence path is identical
// regardless of binding. Snapshot creation/sealing via libsnapper and the
// package resolution via libsolv are exercised only on a live target; here the
// behaviours implement their declared steps and return Diagnostics to the verb
// layer (only the verb maps to an exit code).
#ifndef ZD_CONVERGE_HPP
#define ZD_CONVERGE_HPP

#include <string>

#include "command_runner.hpp"
#include "config.hpp"
#include "types.hpp"

namespace zd {

// acquire-transaction-context: resolve the binding for `mode`.
Result<TransactionContext> acquire_transaction_context(
    TransactionMode mode, const CommandRunner& runner);

// converge-packages: configure repos, install/remove, return the resolved lock.
Result<ScopeWrapper<PackageRecord>> converge_packages(
    const TransactionContext& ctx, const Diff& diff, const Config& cfg,
    const CommandRunner& runner);

// converge-files: write files_write (resolving content_ref), delete
// files_delete excluding RPM-owned/keep-listed/etc.syncpoint paths.
Status converge_files(const TransactionContext& ctx, const Diff& diff,
                      const Config& cfg, const CommandRunner& runner);

// converge-units: offline enablement against ctx.root.
Status converge_units(const TransactionContext& ctx, const Diff& diff,
                      const CommandRunner& runner);

// write-applied-record: assemble + write applied.json into ctx.root.
Status write_applied_record(const TransactionContext& ctx,
                            const Manifest& desired,
                            const std::string& desired_sha,
                            const ScopeWrapper<PackageRecord>& resolved,
                            const CommandRunner& runner);

}  // namespace zd

#endif  // ZD_CONVERGE_HPP
