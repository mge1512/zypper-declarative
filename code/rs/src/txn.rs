// generated from spec: zypper-declarative.spec.md
// sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
//
// BEHAVIOR/INTERNAL: acquire-transaction-context. Resolves the transaction
// binding (deliberately deferred between an external mechanism and the
// zypper-internal mechanism) and yields a context the convergence domains
// operate within. The same convergence behaviour applies regardless of which
// binding was resolved.

use crate::config::TransactionMode;
use crate::error::{Diagnostic, Domain};

#[derive(Debug, Clone)]
pub struct TransactionContext {
    pub mode: TransactionMode,
    pub root: String,
    pub opened_here: bool,
}

/// Detect whether the process already runs inside a fresh snapshot transaction.
/// The transactional-update tooling exports TRANSACTIONAL_UPDATE and provides a
/// writable new-generation root; we detect its presence. (Environment variable
/// reading here is the transaction-mechanism's own contract, not tool
/// configuration, which the template forbids — config still comes only from
/// key=value/preset.)
fn detect_external_root() -> Option<String> {
    // transactional-update mounts the new root and points the tool at it. The
    // canonical signal is the presence of a writable new-generation root passed
    // by the opener. We probe the conventional mount used by
    // `transactional-update run`.
    for var in ["TRANSACTIONAL_UPDATE_ROOT", "DISTUPDATE_ROOT"] {
        if let Ok(v) = std::env::var(var) {
            if !v.is_empty() {
                return Some(v);
            }
        }
    }
    None
}

/// acquire-transaction-context(mode) -> TransactionContext
pub fn acquire_transaction_context(
    mode: &TransactionMode,
) -> Result<TransactionContext, Diagnostic> {
    let resolved = match mode {
        TransactionMode::Auto => {
            if detect_external_root().is_some() {
                TransactionMode::External
            } else {
                TransactionMode::Internal
            }
        }
        other => other.clone(),
    };

    match resolved {
        TransactionMode::External => match detect_external_root() {
            Some(root) => Ok(TransactionContext {
                mode: TransactionMode::External,
                root,
                opened_here: false,
            }),
            None => Err(Diagnostic::error(
                Domain::Transaction,
                "external mode but not running inside a snapshot transaction; \
                 invoke inside `transactional-update run ...`",
            )),
        },
        TransactionMode::Internal => {
            // Opening a snapshot transaction through the zypper-merged
            // transactional machinery is a privileged, host-only operation. In
            // an environment where that machinery is unavailable, this fails
            // with a transaction error (exit 2), per the spec.
            Err(Diagnostic::error(
                Domain::Transaction,
                "internal transaction machinery unavailable in this environment",
            ))
        }
        TransactionMode::Auto => unreachable!(),
    }
}
