// generated from spec: zypper-declarative.spec.md sha256:18253550a5c3d3f1818f0380811cb5dbc98874828693e49fc9cd5cbc923303dd
//
// BEHAVIOR/INTERNAL: acquire-transaction-context.
//
// Resolves the deliberately-deferred transaction binding (external mechanism vs
// the zypper-internal mechanism) and yields a context the convergence domains
// operate within. The convergence behaviour is identical regardless of which
// binding resolves.

use crate::error::{Diagnostic, Domain};
use crate::interfaces::CommandRunner;
use crate::manifest::{TransactionContext, TransactionMode};

/// acquire-transaction-context(mode):
/// 1. auto: detect whether already inside a fresh snapshot transaction; if so
///    resolve to external, else internal.
/// 2. external: assert a writable new-generation root is present; opened_here =
///    false. If absent -> transaction error (caller must be inside a
///    transaction). This is an INVOCATION-class failure (exit 2).
/// 3. internal: open a new snapshot transaction; opened_here = true. On failure
///    -> transaction error (exit 2 — mechanism unavailable).
pub fn acquire_transaction_context(
    runner: &dyn CommandRunner,
    mode: TransactionMode,
) -> Result<TransactionContext, Diagnostic> {
    let resolved = match mode {
        TransactionMode::Auto => {
            if running_inside_transaction() {
                TransactionMode::External
            } else {
                TransactionMode::Internal
            }
        }
        m => m,
    };

    match resolved {
        TransactionMode::External => {
            // A writable new-generation root must already be mounted. In the
            // exec/offline model we detect it via the TRANSACTIONAL_UPDATE root
            // marker; absent it, the caller is not inside a transaction.
            match transaction_root_from_environmentless_markers() {
                Some(root) => Ok(TransactionContext {
                    mode: TransactionMode::External,
                    root,
                    opened_here: false,
                }),
                None => Err(Diagnostic::error(
                    Domain::Transaction,
                    "not running inside a snapshot transaction (mode=external requires \
                     the tool to be invoked inside a transaction)",
                )),
            }
        }
        TransactionMode::Internal => open_internal_transaction(runner),
        TransactionMode::Auto => unreachable!("auto already resolved"),
    }
}

/// Detect whether the process already runs inside a fresh snapshot transaction.
/// In the exec model this is determined by the presence of a transactional
/// new-root marker file, not an environment variable (env control is forbidden).
fn running_inside_transaction() -> bool {
    transaction_root_from_environmentless_markers().is_some()
}

/// Resolve the new-generation root from non-environment markers. A
/// transactional-update run sets up a new root and records it in a well-known
/// marker file; absent it, there is no active external transaction.
fn transaction_root_from_environmentless_markers() -> Option<String> {
    // transactional-update records the new root at this conventional path while a
    // run is active. (No environment variable is consulted.)
    let marker = "/run/transactional-update/new-root";
    match std::fs::read_to_string(marker) {
        Ok(s) => {
            let p = s.trim().to_string();
            if p.is_empty() {
                None
            } else {
                Some(p)
            }
        }
        Err(_) => None,
    }
}

/// Open a new snapshot transaction through the zypper-merged transactional
/// machinery. In the exec model this drives `transactional-update` / `snapper`;
/// failure to open is a transaction error (exit 2).
fn open_internal_transaction(runner: &dyn CommandRunner) -> Result<TransactionContext, Diagnostic> {
    // snapper create --print-number prints the new snapshot number; its mount
    // point is then the new-generation root. If the mechanism is unavailable
    // (snapper cannot be spawned), this is a transaction error.
    let res = runner.run("snapper", &["create", "--type", "single", "--print-number"]);
    if res.spawn_failed {
        return Err(Diagnostic::error(
            Domain::Transaction,
            "transaction mechanism unavailable: cannot open a snapshot transaction \
             (snapper not available)",
        ));
    }
    if res.code != 0 {
        return Err(Diagnostic::error(
            Domain::Transaction,
            format!(
                "transaction mechanism unavailable: snapshot creation failed: {}",
                res.stderr.trim()
            ),
        ));
    }
    let num = res.stdout.trim();
    let root = format!("/.snapshots/{}/snapshot", num);
    Ok(TransactionContext {
        mode: TransactionMode::Internal,
        root,
        opened_here: true,
    })
}
