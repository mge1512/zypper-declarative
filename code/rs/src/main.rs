// generated from spec: zypper-declarative.spec.md
// sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
//
// Entry point: CLI dispatch only. Installs SIGTERM/SIGINT handlers for a clean
// exit (no partial output), parses argv, and calls into the implementation
// (zypper_declarative::cli::run). No behaviour is implemented here.

use zypper_declarative::cli;

fn install_signal_handlers() {
    // Clean exit on SIGTERM and SIGINT: an interrupted run terminates the
    // process without leaving partial output. Because apply performs all
    // mutation inside a snapshot transaction that is only sealed/activated as
    // the final step, an interrupt before that step leaves the transaction
    // unsealed (discarded), so the running system is unchanged. We install
    // handlers that exit the process cleanly with the conventional 128+signo
    // code via libc, after flushing stdio is implicitly handled by exit.
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

extern "C" fn handle_signal(signo: libc::c_int) {
    // Exit cleanly. Use _exit to avoid running non-async-signal-safe atexit
    // handlers; no partial output is emitted because verbs write their full
    // output in one pass at the end.
    unsafe {
        libc::_exit(128 + signo);
    }
}

fn main() {
    install_signal_handlers();
    let args: Vec<String> = std::env::args().skip(1).collect();
    let code = cli::run(&args);
    std::process::exit(code);
}
