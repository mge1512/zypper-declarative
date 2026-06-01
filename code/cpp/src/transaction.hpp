// generated from spec: zypper-declarative.spec.md sha256:f2cc80627e483a48bb8411d297711bc5f6c6e74c28dbf0dafc8fe7bd8817251e
//
// transaction.hpp -- acquire-transaction-context and the convergence behaviours
// (converge-packages, converge-files, converge-units, write-applied-record).
// These operate on a live host inside a snapshot transaction (libsnapper /
// transactional-update). Per the milestone hints, real transaction, package,
// and unit work is exercised on-target; the offline-testable parts (the file
// writer, the applied-record writer) are implemented here, and the libsnapper /
// libzypp convergence is layered behind the same interface.
#ifndef ZD_TRANSACTION_HPP
#define ZD_TRANSACTION_HPP

#include "types.hpp"
#include "diff.hpp"
#include "command_runner.hpp"
#include <optional>
#include <string>

namespace zd {

struct TxnResult {
    bool ok = false;
    TransactionContext ctx;
    Diagnostic error;
};

// BEHAVIOR/INTERNAL: acquire-transaction-context
TxnResult acquire_transaction_context(TransactionMode mode, const CommandRunner& runner);

struct ConvergeResult {
    bool ok = false;
    Diagnostic error;
};

// BEHAVIOR/INTERNAL: converge-packages -> resolved packages scope (the lock).
struct PackagesConvergeResult {
    bool ok = false;
    PackagesScope resolved;
    Diagnostic error;
};
PackagesConvergeResult converge_packages(const TransactionContext& ctx, const Diff& diff,
                                         const CommandRunner& runner);

// BEHAVIOR/INTERNAL: converge-files (writes/deletes regular files in <root>/etc).
ConvergeResult converge_files(const TransactionContext& ctx, const Diff& diff,
                              const KeepList& keep, const std::string& content_store,
                              const CommandRunner& runner);

// BEHAVIOR/INTERNAL: converge-units (offline enablement against ctx.root).
ConvergeResult converge_units(const TransactionContext& ctx, const Diff& diff,
                              const CommandRunner& runner);

// BEHAVIOR/INTERNAL: write-applied-record
ConvergeResult write_applied_record(const TransactionContext& ctx, const Manifest& desired,
                                    const Sha256& desired_sha256,
                                    const PackagesScope& resolved,
                                    const CommandRunner& runner);

} // namespace zd

#endif // ZD_TRANSACTION_HPP
