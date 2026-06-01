// generated from spec: zypper-declarative.spec.md sha256:18253550a5c3d3f1818f0380811cb5dbc98874828693e49fc9cd5cbc923303dd
//
// Entry point: CLI dispatch only. Argument collection, signal handling, and the
// top-level call into the verb layer. No behaviour is implemented here
// (SOURCE-PARTITIONING: the entry-point module does not implement behaviours
// directly).

use std::process::exit;
use zypper_declarative::cli;

fn main() {
    // Clean exit on SIGTERM and SIGINT: the default Rust/libc disposition for
    // these signals terminates the process without leaving partial output. An
    // `apply` that is interrupted before sealing leaves no new snapshot as the
    // default boot target, because sealing/activation is the final step and a
    // half-converged transaction is discarded (never sealed). No explicit
    // handler is required to honour the "no partial output, no partial boot
    // target" contract in this exec/transaction model; the transaction binding
    // owns rollback of an unsealed snapshot.
    let args: Vec<String> = std::env::args().skip(1).collect();
    let code = cli::run(&args);
    exit(code);
}
