// generated from spec: zypper-declarative.spec.md sha256:87b9f2f3bf92afd6fe037412d56a23959290e5816d722a3def9505d07aa5acd7
//
// Entry point: CLI dispatch only. Argument collection, signal handling, and
// top-level exit-code propagation. All behaviour implementation lives in the
// library modules (per SOURCE-PARTITIONING: one-entry-one-implementation).

use std::process::exit;

fn main() {
    install_signal_handlers();

    // argv[0] is the program path; the dispatcher receives the remaining args.
    let args: Vec<String> = std::env::args().skip(1).collect();
    let code = zypper_declarative::cli::run(&args);
    exit(code);
}

/// Clean exit on SIGTERM and SIGINT with no partial output. We install handlers
/// that exit promptly; the long-running verb (apply) discards its transaction by
/// virtue of never having sealed it (sealing is the final step), so an interrupt
/// before step 11 leaves no new default boot target.
fn install_signal_handlers() {
    // SIG_DFL termination for SIGTERM/SIGINT already exits the process without
    // partial output for the read-only verbs. We register explicit handlers that
    // exit cleanly so the behaviour is deterministic and documented.
    unsafe {
        libc::signal(
            libc::SIGTERM,
            handle_signal as *const () as libc::sighandler_t,
        );
        libc::signal(
            libc::SIGINT,
            handle_signal as *const () as libc::sighandler_t,
        );
    }
}

extern "C" fn handle_signal(_sig: i32) {
    // Exit cleanly. No partial output is emitted: stdout writes are line-buffered
    // and any in-progress transaction has not been sealed, so the running system
    // is left unchanged.
    std::process::exit(130);
}
