// generated from spec: zypper-declarative.spec.md sha256:87b9f2f3bf92afd6fe037412d56a23959290e5816d722a3def9505d07aa5acd7
//
// The verb layer. Dispatches the parsed invocation to the BEHAVIOR verbs (apply,
// diff, verify, status, describe) and the global commands (version, help, bare
// invocation). Exit-code mapping lives ONLY here: internal behaviours return
// Diagnostics carrying their domain, and this layer maps them to exit codes.
// Diagnostics go to stderr (one per line); normal output goes to stdout.

use crate::config::{load_keep_list, parse_args, Invocation, Options, Verb};
use crate::converge::{converge_files, converge_packages, converge_units};
use crate::diff::{compute_drift, compute_intent_diff};
use crate::error::{Diagnostic, Domain, ExitCode};
use crate::format::resolve_format;
use crate::interfaces::{CommandRunner, OSCommandRunner};
use crate::manifest::{load_desired_manifest, parse_state_dump, serialise_manifest};
use crate::record::{load_applied_record, write_applied_record};
use crate::state::describe_actual_state;
use crate::txn::acquire_transaction_context;
use crate::types::*;
use std::collections::HashSet;
use std::io::Write;

/// Entry from main: parse argv, dispatch, return the process exit code.
pub fn run(args: &[String]) -> i32 {
    let invocation = match parse_args(args) {
        Ok(inv) => inv,
        Err(d) => {
            // Invocation error: usage to stderr, exit 2.
            emit_diagnostic(&d);
            eprintln!("{}", usage_text());
            return ExitCode::Invocation.code();
        }
    };

    let runner = OSCommandRunner;
    dispatch(&invocation, &runner)
}

fn dispatch(inv: &Invocation, runner: &dyn CommandRunner) -> i32 {
    match inv.verb {
        None => {
            // Bare invocation: usage to stdout, exit 0 (discovery).
            println!("{}", usage_text());
            ExitCode::Ok.code()
        }
        Some(Verb::Version) => {
            println!("{}", crate::meta::version_line());
            ExitCode::Ok.code()
        }
        Some(Verb::Help) => {
            println!("{}", usage_text());
            ExitCode::Ok.code()
        }
        Some(Verb::Apply) => map_result(verb_apply(&inv.opts, runner)),
        Some(Verb::Diff) => map_result(verb_diff(&inv.opts, runner)),
        Some(Verb::Verify) => verb_verify(&inv.opts, runner),
        Some(Verb::Status) => map_result(verb_status(&inv.opts, runner)),
        Some(Verb::Describe) => map_result(verb_describe(&inv.opts, runner)),
    }
}

/// Map a verb result to an exit code: Ok -> 0; Err(domain) -> 1 or 2 per domain.
fn map_result(r: Result<(), Diagnostic>) -> i32 {
    match r {
        Ok(()) => ExitCode::Ok.code(),
        Err(d) => {
            emit_diagnostic(&d);
            exit_for_domain(d.domain)
        }
    }
}

/// Domain -> exit code. Invocation/transaction read/format failures are exit 2;
/// logical failures (manifest invalid, convergence failed, drift) are exit 1.
fn exit_for_domain(domain: Domain) -> i32 {
    match domain {
        Domain::Invocation => ExitCode::Invocation.code(),
        Domain::Transaction => ExitCode::Invocation.code(),
        _ => ExitCode::Logical.code(),
    }
}

fn emit_diagnostic(d: &Diagnostic) {
    let _ = writeln!(std::io::stderr(), "{}", d.render());
}

fn default_format(opts: &Options) -> ManifestFormat {
    opts.manifest_format
}

// ---------------------------------------------------------------------------
// apply
// ---------------------------------------------------------------------------

fn verb_apply(opts: &Options, runner: &dyn CommandRunner) -> Result<(), Diagnostic> {
    let keep_list = load_keep_list(opts);

    // Step 1: load the desired manifest.
    let manifest_path = opts
        .manifest_path
        .clone()
        .unwrap_or_else(|| "/var/lib/zypper-declarative/desired.json".to_string());
    let loaded = load_desired_manifest(&manifest_path, opts.format, default_format(opts))?;
    let desired = loaded.manifest;
    let desired_sha = loaded.desired_sha256;

    // Step 2: load the applied record.
    let applied = load_applied_record(&opts.applied_root)?.record;

    // Step 3: intent diff.
    let intent = compute_intent_diff(&desired, &applied);

    // Step 4: if intent empty, check drift on "/"; if also empty, nothing to do.
    if intent.is_empty() {
        let actual =
            describe_actual_state(runner, "/", OnUnreadable::Error, ScanScope::Etc, &keep_list)?;
        let drift = compute_drift(&actual.manifest, &applied, &keep_list);
        if drift.is_empty() {
            println!("nothing to do");
            return Ok(());
        }
    }

    // Step 5: acquire the transaction context.
    let acquired = acquire_transaction_context(runner, opts.mode)?;
    let ctx_root = acquired.root.clone();

    // Step 6: converge packages (repositories applied within converge-packages).
    let resolved = converge_packages(runner, &ctx_root, &intent, opts.repo_lock.as_deref())?;

    // Step 7: converge files.
    converge_files(
        runner,
        &ctx_root,
        &intent,
        opts.content_store.as_deref(),
        &keep_list,
    )?;

    // Step 8: converge units.
    converge_units(runner, &ctx_root, &intent)?;

    // Step 9: write the applied record.
    write_applied_record(&ctx_root, &desired, &desired_sha, &resolved)?;

    // Step 10: post-converge verification.
    let new_applied = load_applied_record(&ctx_root)?.record;
    let actual = describe_actual_state(
        runner,
        &ctx_root,
        OnUnreadable::Error,
        ScanScope::Etc,
        &keep_list,
    )?;
    let drift = compute_drift(&actual.manifest, &new_applied, &keep_list);
    if !drift.is_empty() {
        return Err(Diagnostic::error(
            Domain::Files,
            "post-converge verification found drift; transaction discarded".to_string(),
        ));
    }

    // Step 11: seal and activate (delegated; emit summary).
    println!(
        "converged: {} package(s), {} file(s) written, {} unit(s) changed",
        resolved.elements.len(),
        intent.files_write.len(),
        intent.units_change.len()
    );
    Ok(())
}

// ---------------------------------------------------------------------------
// diff
// ---------------------------------------------------------------------------

fn verb_diff(opts: &Options, runner: &dyn CommandRunner) -> Result<(), Diagnostic> {
    let keep_list = load_keep_list(opts);

    // Step 1: load the desired manifest.
    let manifest_path = opts
        .manifest_path
        .clone()
        .unwrap_or_else(|| "/var/lib/zypper-declarative/desired.json".to_string());
    let loaded = load_desired_manifest(&manifest_path, opts.format, default_format(opts))?;
    let desired = loaded.manifest;

    // Step 2: load the applied record.
    let applied = load_applied_record(&opts.applied_root)?.record;

    // Step 3: intent diff.
    let intent = compute_intent_diff(&desired, &applied);

    // Step 4: obtain actual state for the drift portion.
    let actual = obtain_actual_state(opts, runner, &keep_list)?;
    let drift = compute_drift(&actual, &applied, &keep_list);

    // Step 5: print the combined plan, exit 0.
    print_plan(&intent, &drift);
    Ok(())
}

fn print_plan(intent: &Diff, drift: &DriftReport) {
    println!("packages to install:");
    for p in &intent.packages_install {
        println!("  {}", p.name);
    }
    println!("packages to remove:");
    for p in &intent.packages_remove {
        println!("  {}", p.name);
    }
    println!("repositories to set:");
    for r in &intent.repos_set {
        println!("  {}", r.alias);
    }
    println!("files to write:");
    for f in &intent.files_write {
        println!("  {}", f.name);
    }
    println!("files to delete:");
    for f in &intent.files_delete {
        println!("  {}", f);
    }
    println!("units to change:");
    for u in &intent.units_change {
        println!("  {} -> {}", u.name, u.state);
    }
    println!("drift:");
    for f in &drift.files_modified {
        println!("  modified: {}", f);
    }
    for f in &drift.files_extra {
        println!("  extra: {}", f);
    }
    for u in &drift.units_divergent {
        println!("  unit divergent: {}", u.name);
    }
    for p in &drift.packages_divergent {
        println!("  package divergent: {}", p.name);
    }
    for f in &drift.managed_files_modified {
        println!("  managed modified: {}", f);
    }
    for f in &drift.unmanaged_files_present {
        println!("  unmanaged present: {}", f);
    }
}

// ---------------------------------------------------------------------------
// verify
// ---------------------------------------------------------------------------

fn verb_verify(opts: &Options, runner: &dyn CommandRunner) -> i32 {
    match verify_inner(opts, runner) {
        Ok(VerifyOutcome::Clean) => {
            println!("system matches declaration");
            ExitCode::Ok.code()
        }
        Ok(VerifyOutcome::Drift(diags)) => {
            for d in &diags {
                emit_diagnostic(d);
            }
            ExitCode::Logical.code()
        }
        Err(d) => {
            emit_diagnostic(&d);
            exit_for_domain(d.domain)
        }
    }
}

enum VerifyOutcome {
    Clean,
    Drift(Vec<Diagnostic>),
}

fn verify_inner(opts: &Options, runner: &dyn CommandRunner) -> Result<VerifyOutcome, Diagnostic> {
    let keep_list = load_keep_list(opts);

    // Step 1: determine the reference.
    let reference: AppliedRecord = if let Some(mp) = opts.manifest_path.as_ref() {
        // reference manifest read on read/format failure -> exit 2; schema -> exit 1.
        load_desired_manifest(mp, opts.format, default_format(opts))?.manifest
    } else {
        let loaded = load_applied_record(&opts.applied_root)?;
        if !loaded.present {
            return Err(Diagnostic::error(
                Domain::Invocation,
                "no declaration applied".to_string(),
            ));
        }
        loaded.record
    };

    // Step 2: obtain the actual state.
    let actual = obtain_actual_state(opts, runner, &keep_list)?;

    // Step 3: compute drift.
    let drift = compute_drift(&actual, &reference, &keep_list);

    // Step 4: clean -> exit 0; else one diagnostic per drift item -> exit 1.
    if drift.is_empty() {
        Ok(VerifyOutcome::Clean)
    } else {
        let mut diags = Vec::new();
        for f in &drift.files_modified {
            diags.push(Diagnostic::error(
                Domain::Files,
                format!("drift: modified {}", f),
            ));
        }
        for f in &drift.files_extra {
            diags.push(Diagnostic::error(
                Domain::Files,
                format!("drift: extra {}", f),
            ));
        }
        for u in &drift.units_divergent {
            diags.push(Diagnostic::error(
                Domain::Units,
                format!("drift: unit {} state diverges", u.name),
            ));
        }
        for p in &drift.packages_divergent {
            diags.push(Diagnostic::error(
                Domain::Packages,
                format!("drift: package {} diverges", p.name),
            ));
        }
        for f in &drift.managed_files_modified {
            diags.push(Diagnostic::error(
                Domain::Files,
                format!("drift: managed file modified {}", f),
            ));
        }
        for f in &drift.unmanaged_files_present {
            diags.push(Diagnostic::error(
                Domain::Files,
                format!("drift: unmanaged file present {}", f),
            ));
        }
        Ok(VerifyOutcome::Drift(diags))
    }
}

// ---------------------------------------------------------------------------
// status
// ---------------------------------------------------------------------------

fn verb_status(opts: &Options, runner: &dyn CommandRunner) -> Result<(), Diagnostic> {
    // Step 1: unrecognised arguments are already rejected by the parser (exit 2).
    let keep_list = load_keep_list(opts);

    // Step 2: load the applied record.
    let loaded = load_applied_record(&opts.applied_root)?;
    if !loaded.present {
        println!("no declaration applied");
        return Ok(());
    }
    let record = loaded.record;

    // Step 3: print applied summary.
    let pkg_count = record.packages_elems().len();
    println!("desired_sha256: {}", record.meta.desired_sha256);
    println!("format_version: {}", record.meta.format_version);
    println!("generation: {}", opts.applied_root);
    println!("created_at: {}", record.meta.created_at);
    println!("packages: {}", pkg_count);

    // Step 4: drift summary line.
    let actual =
        describe_actual_state(runner, "/", OnUnreadable::Error, ScanScope::Etc, &keep_list)?;
    let drift = compute_drift(&actual.manifest, &record, &keep_list);
    let count = drift_item_count(&drift);
    if count == 0 {
        println!("clean");
    } else {
        println!("{} drift item(s)", count);
    }
    Ok(())
}

fn drift_item_count(d: &DriftReport) -> usize {
    d.files_modified.len()
        + d.files_extra.len()
        + d.units_divergent.len()
        + d.packages_divergent.len()
        + d.managed_files_modified.len()
        + d.unmanaged_files_present.len()
}

// ---------------------------------------------------------------------------
// describe
// ---------------------------------------------------------------------------

fn verb_describe(opts: &Options, runner: &dyn CommandRunner) -> Result<(), Diagnostic> {
    let keep_list = load_keep_list(opts);

    // Step 2: obtain the actual state.
    let actual = describe_actual_state(
        runner,
        &opts.root,
        opts.on_unreadable,
        opts.scope,
        &keep_list,
    )?;
    for d in &actual.diagnostics {
        emit_diagnostic(d);
    }

    // Step 3: resolve the output format via resolve-format against `out`.
    let fmt = resolve_format(opts.format, opts.out.as_deref(), default_format(opts));

    // Step 4: serialise.
    let document = serialise_manifest(&actual.manifest, fmt)?;

    // Step 5: write to out if given, else stdout. On write failure exit 2.
    match opts.out.as_ref() {
        Some(path) => {
            std::fs::write(path, document.as_bytes()).map_err(|e| {
                Diagnostic::error(
                    Domain::Invocation,
                    format!("output path unwritable: {}: {}", path, e),
                )
            })?;
        }
        None => {
            print!("{}", document);
            if !document.ends_with('\n') {
                println!();
            }
        }
    }
    Ok(())
}

// ---------------------------------------------------------------------------
// shared actual-state acquisition for diff/verify (supports offline state-path)
// ---------------------------------------------------------------------------

fn obtain_actual_state(
    opts: &Options,
    runner: &dyn CommandRunner,
    keep_list: &HashSet<String>,
) -> Result<Manifest, Diagnostic> {
    if let Some(sp) = opts.state_path.as_ref() {
        // Offline: load and schema-validate the dump (no live read).
        let bytes = std::fs::read(sp).map_err(|e| {
            Diagnostic::error(
                Domain::Invocation,
                format!("malformed state dump: {}: {}", sp, e),
            )
        })?;
        let fmt = resolve_format(opts.format, Some(sp), default_format(opts));
        parse_state_dump(&bytes, fmt)
    } else {
        // Live read via the single reader, always scope=etc unless verify scope=full.
        let scope = opts.scope;
        let actual = describe_actual_state(runner, "/", OnUnreadable::Error, scope, keep_list)?;
        Ok(actual.manifest)
    }
}

// ---------------------------------------------------------------------------
// usage
// ---------------------------------------------------------------------------

fn usage_text() -> String {
    let mut s = String::new();
    s.push_str("usage: zypper-declarative <verb> [key=value ...]\n");
    s.push_str("\n");
    s.push_str("verbs:\n");
    s.push_str(
        "  apply      converge the system to the desired manifest in a snapshot transaction\n",
    );
    s.push_str("  diff       dry run: print what apply would change (no modification)\n");
    s.push_str("  verify     check the actual state against a reference declaration\n");
    s.push_str("  status     print the current declarative state and a drift summary\n");
    s.push_str("  describe   read the actual state and emit it as a Manifest\n");
    s.push_str("  version    print program name, version, and embedded spec hash\n");
    s.push_str("  help       print this usage\n");
    s.push_str("\n");
    s.push_str("options (key=value, may appear in any position):\n");
    s.push_str("  mode=auto|external|internal       transaction binding (default auto)\n");
    s.push_str("  manifest-path=<path>              desired/reference manifest\n");
    s.push_str("  format=json|yaml                  serialisation for this invocation\n");
    s.push_str("  manifest-format=json|yaml         fallback serialisation (default json)\n");
    s.push_str("  state-path=<path>                 captured actual state (offline)\n");
    s.push_str("  root=<path>                       root to describe (default /)\n");
    s.push_str("  out=<path>                        describe output file (default stdout)\n");
    s.push_str(
        "  on-unreadable=error|warn          describe unreadable-source policy (default error)\n",
    );
    s.push_str("  scope=etc|full                    describe/verify read scope (default etc)\n");
    s.push_str("  repo-lock=<repo>                  fallback pinned repository\n");
    s.push_str("  content-store=<path>              base for content_ref resolution\n");
    s.push_str("  keep-list=<path>                  allowlist of persistent undeclared paths\n");
    s.push_str(
        "  signature-verification=on|off     manifest signature verification (default on)\n",
    );
    s.push_str("  keyring=<path>                    signature keyring\n");
    s.push_str("  activation-policy=reboot|soft-reboot|none   apply activation policy\n");
    s.push_str(
        "  applied-root=<path>               generation root for the applied record (default /)\n",
    );
    s.push_str("\n");
    s.push_str(&format!("{}\n", crate::meta::version_line()));
    s
}
