// generated from spec: zypper-declarative.spec.md sha256:aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3
#pragma once

#include <string>

#include "types.hpp"
#include "diagnostic.hpp"

namespace zd {

// BEHAVIOR/INTERNAL: acquire-transaction-context. Resolves auto|external|
// internal and yields a context with a writable root, or a transaction error.
Result<TransactionContext> acquire_transaction_context(TransactionMode mode);

// Stamp the snapshot's snapper userdata with manifest=<desired_sha256>.
VoidResult stamp_snapshot_userdata(const TransactionContext& ctx,
                                   const std::string& desired_sha256);

// Seal a snapshot read-only and mark it the default boot target (if opened
// here). A no-op for an externally-opened transaction.
VoidResult seal_and_activate(const TransactionContext& ctx,
                             const std::string& activation_policy);

}  // namespace zd
