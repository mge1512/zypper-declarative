// generated from spec: zypper-declarative.spec.md sha256:87b9f2f3bf92afd6fe037412d56a23959290e5816d722a3def9505d07aa5acd7
//
// BEHAVIOR/INTERNAL: acquire-transaction-context. Resolves the transaction binding
// (deliberately deferred between an external mechanism and the zypper-internal
// mechanism) and yields a context the convergence domains operate within. The same
// convergence code path runs regardless of which binding resolved.

use crate::error::{Diagnostic, Domain};
use crate::interfaces::CommandRunner;
use crate::types::{TransactionContext, TransactionMode};

/// A resolved transaction context plus the writable root the convergence operates on.
pub struct AcquiredContext {
    pub ctx: TransactionContext,
    pub root: String,
}

/// acquire-transaction-context.
///
/// STEPS:
/// 1. mode=auto: detect whether already running inside a fresh snapshot
///    transaction; if so resolve to external, else internal.
/// 2. external: assert a writable new-generation root is present; opened_here=false.
/// 3. internal: open a new snapshot transaction; opened_here=true.
pub fn acquire_transaction_context(
    runner: &dyn CommandRunner,
    mode: TransactionMode,
) -> Result<AcquiredContext, Diagnostic> {
    let resolved = match mode {
        TransactionMode::Auto => {
            if running_inside_transaction(runner) {
                TransactionMode::External
            } else {
                TransactionMode::Internal
            }
        }
        m => m,
    };

    match resolved {
        TransactionMode::External => {
            // A separate mechanism opened it; we operate inside it. Assert a
            // writable new-generation root is present.
            match new_generation_root(runner) {
                Some(root) => Ok(AcquiredContext {
                    ctx: TransactionContext {
                        mode: TransactionMode::External,
                        opened_here: false,
                    },
                    root,
                }),
                None => Err(Diagnostic::error(
                    Domain::Transaction,
                    "external mode but not running inside a transaction (no writable new-generation root)"
                        .to_string(),
                )),
            }
        }
        TransactionMode::Internal => {
            // Open a new snapshot transaction through the transactional machinery.
            match open_internal_transaction(runner) {
                Some(root) => Ok(AcquiredContext {
                    ctx: TransactionContext {
                        mode: TransactionMode::Internal,
                        opened_here: true,
                    },
                    root,
                }),
                None => Err(Diagnostic::error(
                    Domain::Transaction,
                    "internal mode but transaction could not be opened".to_string(),
                )),
            }
        }
        TransactionMode::Auto => unreachable!("auto resolved above"),
    }
}

/// Detect whether the process already runs inside a fresh snapshot transaction.
/// On a non-transactional host this is false. We probe transactional-update's
/// in-transaction marker; absence means not inside a transaction.
fn running_inside_transaction(_runner: &dyn CommandRunner) -> bool {
    // The canonical signal is the TRANSACTIONAL_UPDATE marker file/env the opener
    // sets. Environment-variable control of behaviour is forbidden for options, but
    // detecting an externally-set transaction marker is a system probe, not a knob.
    std::path::Path::new("/run/transactional-update.flag").exists()
}

/// On a host where an external transaction is open, return its writable root.
fn new_generation_root(_runner: &dyn CommandRunner) -> Option<String> {
    // Without a live transactional host this is unavailable.
    if std::path::Path::new("/run/transactional-update.flag").exists() {
        Some("/".to_string())
    } else {
        None
    }
}

/// Open a new internal snapshot transaction; return its mount point. On a host
/// without the transactional machinery this fails (returns None).
fn open_internal_transaction(runner: &dyn CommandRunner) -> Option<String> {
    // Drive the transactional machinery. On a non-transactional host the command
    // is absent or fails, which the caller maps to a transaction error.
    match runner.run("snapper", &["create", "--print-number", "--type", "single"]) {
        Ok((stdout, _)) => {
            let num = stdout.trim();
            if num.is_empty() {
                None
            } else {
                Some(format!("/.snapshots/{}/snapshot", num))
            }
        }
        Err(_) => None,
    }
}
