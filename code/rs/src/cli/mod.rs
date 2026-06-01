// generated from spec: zypper-declarative.spec.md sha256:18253550a5c3d3f1818f0380811cb5dbc98874828693e49fc9cd5cbc923303dd
//
// The verb layer: dispatch, key=value parsing, the global contract, and the
// five CLI verbs (apply, diff, verify, status, describe). Exit-code mapping
// lives ONLY here; internal behaviours return Diagnostics to this layer.
//
// CLI contract (spec DEPLOYMENT + INVARIANTS):
// - options are key=value, accepted in ANY position (also after the verb);
// - bare words are verbs; `version` and `help` are canonical global commands
//   (exit 0), with `--version`, `--help`, `-h` as tolerated aliases;
// - bare invocation prints usage to stdout and exits 0 (discovery, never
//   converges);
// - unknown verb/option/value or a missing value -> usage to stderr, exit 2;
// - no option uses POSIX --flag style.

use crate::clock::now_rfc3339;
use crate::config::{Config, OnUnreadable, ScanScope};
use crate::converge;
use crate::diff as diffmod;
use crate::error::{Diagnostic, Domain, ExitCode};
use crate::interfaces::{CommandRunner, OsCommandRunner};
use crate::load::load_desired_manifest;
use crate::manifest::format::resolve_format;
use crate::manifest::serialize::serialise_manifest;
use crate::manifest::{AppliedRecord, Manifest, ManifestFormat, TransactionMode};
use crate::record;
use crate::state::describe_actual_state;
use crate::txn::acquire_transaction_context;
use crate::meta;
use std::collections::HashSet;
use std::io::Write;
use std::path::Path;

/// The parsed invocation: a verb (or a global command) plus options and any bare
/// trailing argument tokens.
struct Parsed {
    verb: Option<String>,
    options: Vec<(String, String)>,
    bare_extra: Vec<String>,
}

/// Top-level entry: parse, dispatch, return an exit code.
pub fn run(args: &[String]) -> i32 {
    let runner = OsCommandRunner;
    dispatch(&runner, args)
}

/// Dispatch is split out so unit tests can pass a scripted runner. Stdout/stderr
/// are the real process streams.
pub fn dispatch(runner: &dyn CommandRunner, args: &[String]) -> i32 {
    // Global flag aliases are recognised regardless of position.
    if args.iter().any(|a| a == "--version") {
        print_version();
        return ExitCode::Ok.code();
    }
    if args.iter().any(|a| a == "--help" || a == "-h") {
        print_usage_stdout();
        return ExitCode::Ok.code();
    }

    // Bare invocation: usage to stdout, exit 0.
    if args.is_empty() {
        print_usage_stdout();
        return ExitCode::Ok.code();
    }

    let parsed = match parse_args(args) {
        Ok(p) => p,
        Err(diag) => {
            eprintln!("{}", diag.message);
            print_usage_stderr();
            return ExitCode::Invocation.code();
        }
    };

    let verb = match &parsed.verb {
        Some(v) => v.clone(),
        None => {
            // Only options, no verb: treat as discovery -> usage to stdout, 0.
            print_usage_stdout();
            return ExitCode::Ok.code();
        }
    };

    match verb.as_str() {
        "version" => {
            print_version();
            ExitCode::Ok.code()
        }
        "help" => {
            print_usage_stdout();
            ExitCode::Ok.code()
        }
        "apply" => verb_apply(runner, &parsed),
        "diff" => verb_diff(runner, &parsed),
        "verify" => verb_verify(runner, &parsed),
        "status" => verb_status(runner, &parsed),
        "describe" => verb_describe(runner, &parsed),
        other => {
            eprintln!("error: invocation: unknown verb: {}", other);
            print_usage_stderr();
            ExitCode::Invocation.code()
        }
    }
}

// ---------------------------------------------------------------------------
// argument parsing
// ---------------------------------------------------------------------------

/// Recognised key=value option keys.
fn known_option(key: &str) -> bool {
    matches!(
        key,
        "mode"
            | "manifest-path"
            | "format"
            | "state-path"
            | "root"
            | "out"
            | "on-unreadable"
            | "scope"
            | "transaction-mode"
            | "manifest-format"
            | "repo-lock"
            | "content-store"
            | "keep-list"
            | "signature-verification"
            | "keyring"
            | "activation-policy"
            | "applied-root"
    )
}

fn parse_args(args: &[String]) -> Result<Parsed, Diagnostic> {
    let mut verb: Option<String> = None;
    let mut options = Vec::new();
    let mut bare_extra = Vec::new();

    for arg in args {
        if arg == "--version" || arg == "--help" || arg == "-h" {
            // Handled at dispatch; ignore here.
            continue;
        }
        if let Some((k, v)) = arg.split_once('=') {
            if !known_option(k) {
                return Err(Diagnostic::error(
                    Domain::Invocation,
                    format!("error: invocation: unknown option: {}", k),
                ));
            }
            if v.is_empty() && requires_value(k) {
                return Err(Diagnostic::error(
                    Domain::Invocation,
                    format!("error: invocation: option {} requires a value", k),
                ));
            }
            // Validate enumerated option VALUES at parse time so an unknown
            // value is an invocation error regardless of whether a verb follows
            // (e.g. a bare `format=bad_value` must exit 2).
            validate_option_value(k, v)?;
            options.push((k.to_string(), v.to_string()));
        } else if arg.starts_with("--") || arg.starts_with('-') {
            // A POSIX-style flag that is not a tolerated alias is unknown.
            return Err(Diagnostic::error(
                Domain::Invocation,
                format!("error: invocation: unknown argument: {}", arg),
            ));
        } else if verb.is_none() {
            verb = Some(arg.clone());
        } else {
            bare_extra.push(arg.clone());
        }
    }

    Ok(Parsed {
        verb,
        options,
        bare_extra,
    })
}

fn requires_value(key: &str) -> bool {
    // Every key=value option carries a value; format= with an empty value is an
    // invocation error (it is not a valid format value).
    matches!(
        key,
        "manifest-path" | "state-path" | "root" | "out" | "keyring" | "keep-list"
            | "content-store" | "repo-lock" | "applied-root" | "format"
    )
}

/// Validate enumerated option VALUES. An unknown value of an enumerated option
/// is an invocation error; free-form options (paths, etc.) accept any string.
fn validate_option_value(key: &str, value: &str) -> Result<(), Diagnostic> {
    let ok = match key {
        "mode" | "transaction-mode" => TransactionMode::parse(value).is_some(),
        "format" | "manifest-format" => ManifestFormat::parse(value).is_some(),
        "on-unreadable" => OnUnreadable::parse(value).is_some(),
        "scope" => ScanScope::parse(value).is_some(),
        "signature-verification" => matches!(value, "on" | "off"),
        "activation-policy" => matches!(value, "reboot" | "soft-reboot" | "none"),
        _ => true,
    };
    if ok {
        Ok(())
    } else {
        Err(Diagnostic::error(
            Domain::Invocation,
            format!("error: invocation: unknown {} value: {}", key, value),
        ))
    }
}


/// Build a Config from the parsed options. Returns an invocation diagnostic on
/// an unknown option VALUE (e.g. format=toml, mode=bogus).
fn build_config(parsed: &Parsed) -> Result<(Config, ParsedOpts), Diagnostic> {
    let mut config = Config::default();
    let mut opts = ParsedOpts::default();

    for (k, v) in &parsed.options {
        match k.as_str() {
            "mode" | "transaction-mode" => {
                config.transaction_mode = TransactionMode::parse(v).ok_or_else(|| {
                    Diagnostic::error(
                        Domain::Invocation,
                        format!("error: invocation: unknown {} value: {}", k, v),
                    )
                })?;
            }
            "manifest-path" => opts.manifest_path = Some(v.clone()),
            "state-path" => opts.state_path = Some(v.clone()),
            "root" => opts.root = Some(v.clone()),
            "out" => opts.out = Some(v.clone()),
            "format" => {
                opts.format = Some(ManifestFormat::parse(v).ok_or_else(|| {
                    Diagnostic::error(
                        Domain::Invocation,
                        format!("error: invocation: unknown format value: {}", v),
                    )
                })?);
            }
            "manifest-format" => {
                config.manifest_format = ManifestFormat::parse(v).ok_or_else(|| {
                    Diagnostic::error(
                        Domain::Invocation,
                        format!("error: invocation: unknown manifest-format value: {}", v),
                    )
                })?;
            }
            "on-unreadable" => {
                opts.on_unreadable_set = true;
                config.on_unreadable = OnUnreadable::parse(v).ok_or_else(|| {
                    Diagnostic::error(
                        Domain::Invocation,
                        format!("error: invocation: unknown on-unreadable value: {}", v),
                    )
                })?;
            }
            "scope" => {
                opts.scope_set = true;
                config.scope = ScanScope::parse(v).ok_or_else(|| {
                    Diagnostic::error(
                        Domain::Invocation,
                        format!("error: invocation: unknown scope value: {}", v),
                    )
                })?;
            }
            "repo-lock" => config.repo_lock = Some(v.clone()),
            "content-store" => config.content_store = Some(v.clone()),
            "keep-list" => config.keep_list = Some(v.clone()),
            "signature-verification" => match v.as_str() {
                "on" => config.signature_verification = true,
                "off" => config.signature_verification = false,
                other => {
                    return Err(Diagnostic::error(
                        Domain::Invocation,
                        format!(
                            "error: invocation: unknown signature-verification value: {}",
                            other
                        ),
                    ))
                }
            },
            "keyring" => config.keyring = Some(v.clone()),
            "activation-policy" => config.activation_policy = v.clone(),
            "applied-root" => config.applied_root = v.clone(),
            _ => {}
        }
    }
    Ok((config, opts))
}

#[derive(Default)]
struct ParsedOpts {
    manifest_path: Option<String>,
    state_path: Option<String>,
    root: Option<String>,
    out: Option<String>,
    format: Option<ManifestFormat>,
    on_unreadable_set: bool,
    scope_set: bool,
}

/// Load the keep-list file into a set of paths (one per non-empty line).
fn load_keep_list(config: &Config) -> HashSet<String> {
    let mut set = HashSet::new();
    if let Some(path) = &config.keep_list {
        if let Ok(text) = std::fs::read_to_string(path) {
            for line in text.lines() {
                let p = line.trim();
                if !p.is_empty() && !p.starts_with('#') {
                    set.insert(p.to_string());
                }
            }
        }
    }
    set
}

/// Emit each diagnostic to stderr, one per line.
fn emit_diagnostics(diags: &[Diagnostic]) {
    let stderr = std::io::stderr();
    let mut lock = stderr.lock();
    for d in diags {
        let _ = writeln!(lock, "{}", d.line());
    }
}

fn emit_error(d: &Diagnostic) {
    eprintln!("{}", d.line());
}

// ---------------------------------------------------------------------------
// verb: status
// ---------------------------------------------------------------------------

fn verb_status(runner: &dyn CommandRunner, parsed: &Parsed) -> i32 {
    // 1. Reject any unrecognised argument (a bare extra word) -> usage, exit 2.
    if !parsed.bare_extra.is_empty() {
        eprintln!("error: invocation: unrecognised argument(s) to status");
        print_usage_stderr();
        return ExitCode::Invocation.code();
    }
    let (config, opts) = match build_config(parsed) {
        Ok(c) => c,
        Err(d) => {
            emit_error(&d);
            print_usage_stderr();
            return ExitCode::Invocation.code();
        }
    };
    // scope is not accepted on status.
    if opts.scope_set {
        eprintln!("error: invocation: scope is not accepted on status");
        print_usage_stderr();
        return ExitCode::Invocation.code();
    }

    // 2. Load the applied record.
    let loaded = match record::load_applied_record(&config.applied_root) {
        Ok(l) => l,
        Err(d) => {
            emit_error(&d);
            return ExitCode::Logical.code();
        }
    };
    if !loaded.present {
        println!("no declaration applied");
        return ExitCode::Ok.code();
    }
    let rec = &loaded.record;

    // 3. Print the record summary.
    let pkg_count = rec.packages.as_ref().map(|s| s.elements.len()).unwrap_or(0);
    println!("desired_sha256: {}", rec.meta.desired_sha256);
    println!("format_version: {}", rec.meta.format_version);
    println!("generation: {}", current_generation_id(runner, &config.applied_root));
    println!("created_at: {}", rec.meta.created_at);
    println!("packages: {} resolved", pkg_count);

    // 4. Drift summary (live read, scope=etc).
    let keep = load_keep_list(&config);
    let now = now_rfc3339();
    match describe_actual_state(runner, "/", OnUnreadable::Error, ScanScope::Etc, &keep, &now) {
        Ok(actual) => {
            let report = diffmod::compute_drift(&actual.manifest, rec, &keep);
            if report.is_empty() {
                println!("drift: clean");
            } else {
                println!("drift: {} drift item(s)", report.item_count());
            }
        }
        Err(_d) => {
            // A live read failure during status drift summary is non-fatal to the
            // report's primary purpose; report unknown drift.
            println!("drift: unknown (actual state unavailable)");
        }
    }
    ExitCode::Ok.code()
}

fn current_generation_id(runner: &dyn CommandRunner, root: &str) -> String {
    let _ = (runner, root);
    // The current snapshot/generation identifier. In the exec model this is the
    // active snapper snapshot number; absent snapper it is the root marker.
    let res = runner.run("snapper", &["--print-number", "list", "--columns", "number"]);
    if res.spawn_failed || res.code != 0 {
        return root.to_string();
    }
    res.stdout
        .lines()
        .last()
        .map(|s| s.trim().to_string())
        .filter(|s| !s.is_empty())
        .unwrap_or_else(|| root.to_string())
}

// ---------------------------------------------------------------------------
// verb: describe
// ---------------------------------------------------------------------------

fn verb_describe(runner: &dyn CommandRunner, parsed: &Parsed) -> i32 {
    // 1. Reject unrecognised arguments / unknown format value -> usage, exit 2.
    if !parsed.bare_extra.is_empty() {
        eprintln!("error: invocation: unrecognised argument(s) to describe");
        print_usage_stderr();
        return ExitCode::Invocation.code();
    }
    let (config, opts) = match build_config(parsed) {
        Ok(c) => c,
        Err(d) => {
            emit_error(&d);
            print_usage_stderr();
            return ExitCode::Invocation.code();
        }
    };

    let root = opts.root.clone().unwrap_or_else(|| "/".to_string());
    let keep = load_keep_list(&config);
    let now = now_rfc3339();

    // 2. Obtain the actual state.
    let actual = match describe_actual_state(
        runner,
        &root,
        config.on_unreadable,
        config.scope,
        &keep,
        &now,
    ) {
        Ok(a) => a,
        Err(d) => {
            // Unreadable source under on_unreadable=error -> exit 1.
            emit_error(&d);
            return ExitCode::Logical.code();
        }
    };
    // Warn diagnostics (omitted scopes) go to stderr.
    emit_diagnostics(&actual.diagnostics);

    // 3. Resolve the output format via resolve-format(format, out).
    let format = resolve_format(
        opts.format,
        opts.out.as_deref(),
        config.manifest_format,
    );

    // 4. Serialise.
    let doc = match serialise_manifest(&actual.manifest, format) {
        Ok(s) => s,
        Err(d) => {
            emit_error(&d);
            return ExitCode::Logical.code();
        }
    };

    // 5. Write to out if given, else stdout. On write failure exit 2.
    if let Some(out) = &opts.out {
        if let Err(e) = std::fs::write(out, doc.as_bytes()) {
            emit_error(&Diagnostic::error(
                Domain::Invocation,
                format!("output path unwritable: {}: {}", out, e),
            ));
            return ExitCode::Invocation.code();
        }
    } else {
        print!("{}", doc);
    }
    ExitCode::Ok.code()
}

// ---------------------------------------------------------------------------
// verb: diff
// ---------------------------------------------------------------------------

fn verb_diff(runner: &dyn CommandRunner, parsed: &Parsed) -> i32 {
    if !parsed.bare_extra.is_empty() {
        eprintln!("error: invocation: unrecognised argument(s) to diff");
        print_usage_stderr();
        return ExitCode::Invocation.code();
    }
    let (config, opts) = match build_config(parsed) {
        Ok(c) => c,
        Err(d) => {
            emit_error(&d);
            print_usage_stderr();
            return ExitCode::Invocation.code();
        }
    };
    if opts.scope_set {
        eprintln!("error: invocation: scope is not accepted on diff");
        print_usage_stderr();
        return ExitCode::Invocation.code();
    }

    let manifest_path = match opts.manifest_path.clone().or(config.manifest_path.clone()) {
        Some(p) => p,
        None => {
            emit_error(&Diagnostic::error(
                Domain::Invocation,
                "manifest unreadable: no manifest-path given and no default configured",
            ));
            return ExitCode::Invocation.code();
        }
    };

    // 1. Load the desired manifest.
    let loaded = match load_desired_manifest(&manifest_path, opts.format, &config) {
        Ok(l) => l,
        Err(e) => return exit_for_load_error(&e.diagnostic),
    };

    // 2. Load the applied record (absence -> all scopes empty).
    let applied = match record::load_applied_record(&config.applied_root) {
        Ok(l) => l.record,
        Err(d) => {
            emit_error(&d);
            return ExitCode::Logical.code();
        }
    };

    // 3. Compute the intent diff.
    let intent = diffmod::compute_intent_diff(&loaded.manifest, &applied);

    // 4. Obtain actual state for the drift portion.
    let keep = load_keep_list(&config);
    let actual = match obtain_actual_for_drift(runner, &opts, &config, &keep) {
        Ok(a) => a,
        Err(d) => return exit_for_actual_error(&d),
    };
    let drift = diffmod::compute_drift(&actual, &applied, &keep);

    // 5. Print the combined plan to stdout. Exit 0.
    print_plan(&intent, &drift);
    ExitCode::Ok.code()
}

fn print_plan(intent: &crate::manifest::Diff, drift: &crate::manifest::DriftReport) {
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
    for p in &intent.files_delete {
        println!("  {}", p);
    }
    println!("units to change:");
    for u in &intent.units_change {
        println!("  {} -> {}", u.name, u.state);
    }
    println!("drift:");
    for p in &drift.files_modified {
        println!("  modified: {}", p);
    }
    for p in &drift.files_extra {
        println!("  extra: {}", p);
    }
    for u in &drift.units_divergent {
        println!("  unit: {} -> {}", u.name, u.state);
    }
    for p in &drift.packages_divergent {
        println!("  package: {}", p.name);
    }
    for p in &drift.managed_files_modified {
        println!("  managed-modified: {}", p);
    }
    for p in &drift.unmanaged_files_present {
        println!("  unmanaged: {}", p);
    }
}

// ---------------------------------------------------------------------------
// verb: verify
// ---------------------------------------------------------------------------

fn verb_verify(runner: &dyn CommandRunner, parsed: &Parsed) -> i32 {
    if !parsed.bare_extra.is_empty() {
        eprintln!("error: invocation: unrecognised argument(s) to verify");
        print_usage_stderr();
        return ExitCode::Invocation.code();
    }
    let (config, opts) = match build_config(parsed) {
        Ok(c) => c,
        Err(d) => {
            emit_error(&d);
            print_usage_stderr();
            return ExitCode::Invocation.code();
        }
    };

    // 1. Determine the reference.
    let reference: AppliedRecord = if let Some(mpath) = &opts.manifest_path {
        match load_desired_manifest(mpath, opts.format, &config) {
            Ok(l) => l.manifest,
            Err(e) => return exit_for_load_error(&e.diagnostic),
        }
    } else {
        let loaded = match record::load_applied_record(&config.applied_root) {
            Ok(l) => l,
            Err(d) => {
                emit_error(&d);
                return ExitCode::Logical.code();
            }
        };
        if !loaded.present {
            emit_error(&Diagnostic::error(
                Domain::Invocation,
                "no declaration applied",
            ));
            return ExitCode::Invocation.code();
        }
        loaded.record
    };

    // 2. Obtain the actual state (state-path offline, else live).
    let keep = load_keep_list(&config);
    let actual = match obtain_actual_for_drift(runner, &opts, &config, &keep) {
        Ok(a) => a,
        Err(d) => return exit_for_actual_error(&d),
    };

    // 3. Compute the drift report.
    let drift = diffmod::compute_drift(&actual, &reference, &keep);

    // 4. Empty -> exit 0; else one diagnostic per item -> exit 1.
    if drift.is_empty() {
        println!("system matches declaration");
        return ExitCode::Ok.code();
    }
    let mut diags = Vec::new();
    for p in &drift.files_modified {
        diags.push(Diagnostic::error(Domain::Files, format!("drift: modified file {}", p)));
    }
    for p in &drift.files_extra {
        diags.push(Diagnostic::error(Domain::Files, format!("drift: extra file {}", p)));
    }
    for u in &drift.units_divergent {
        diags.push(Diagnostic::error(
            Domain::Units,
            format!("drift: unit {} should be {}", u.name, u.state),
        ));
    }
    for p in &drift.packages_divergent {
        diags.push(Diagnostic::error(
            Domain::Packages,
            format!("drift: package {} divergent", p.name),
        ));
    }
    for p in &drift.managed_files_modified {
        diags.push(Diagnostic::error(
            Domain::Files,
            format!("drift: managed file modified outside /etc: {}", p),
        ));
    }
    for p in &drift.unmanaged_files_present {
        diags.push(Diagnostic::error(
            Domain::Files,
            format!("drift: unmanaged file present outside /etc: {}", p),
        ));
    }
    emit_diagnostics(&diags);
    ExitCode::Logical.code()
}

// ---------------------------------------------------------------------------
// verb: apply
// ---------------------------------------------------------------------------

fn verb_apply(runner: &dyn CommandRunner, parsed: &Parsed) -> i32 {
    if !parsed.bare_extra.is_empty() {
        eprintln!("error: invocation: unrecognised argument(s) to apply");
        print_usage_stderr();
        return ExitCode::Invocation.code();
    }
    let (config, opts) = match build_config(parsed) {
        Ok(c) => c,
        Err(d) => {
            emit_error(&d);
            print_usage_stderr();
            return ExitCode::Invocation.code();
        }
    };
    if opts.scope_set {
        eprintln!("error: invocation: scope is not accepted on apply");
        print_usage_stderr();
        return ExitCode::Invocation.code();
    }

    let manifest_path = match opts.manifest_path.clone().or(config.manifest_path.clone()) {
        Some(p) => p,
        None => {
            emit_error(&Diagnostic::error(
                Domain::Invocation,
                "manifest unreadable: no manifest-path given and no default configured",
            ));
            return ExitCode::Invocation.code();
        }
    };

    // 1. Load the desired manifest.
    let loaded = match load_desired_manifest(&manifest_path, opts.format, &config) {
        Ok(l) => l,
        Err(e) => return exit_for_load_error(&e.diagnostic),
    };

    // 2. Load the applied record (absence -> empty).
    let applied = match record::load_applied_record(&config.applied_root) {
        Ok(l) => l.record,
        Err(d) => {
            emit_error(&d);
            return ExitCode::Logical.code();
        }
    };

    // 3. Compute the intent diff.
    let intent = diffmod::compute_intent_diff(&loaded.manifest, &applied);

    // 4. If empty, read actual and compute drift; if also empty -> nothing to do.
    let keep = load_keep_list(&config);
    let now = now_rfc3339();
    if intent.is_empty() {
        match describe_actual_state(runner, "/", OnUnreadable::Error, ScanScope::Etc, &keep, &now) {
            Ok(actual) => {
                let drift = diffmod::compute_drift(&actual.manifest, &applied, &keep);
                if drift.is_empty() {
                    println!("nothing to do");
                    return ExitCode::Ok.code();
                }
            }
            Err(d) => {
                emit_error(&d);
                return ExitCode::Logical.code();
            }
        }
    }

    // 5. Acquire the transaction context. On failure exit 2.
    let ctx = match acquire_transaction_context(runner, config.transaction_mode) {
        Ok(c) => c,
        Err(d) => {
            emit_error(&d);
            return ExitCode::Invocation.code();
        }
    };

    // 6. Converge packages (after repos). On failure discard and exit 1.
    let resolved = match converge::converge_packages(runner, &ctx, &intent, &config) {
        Ok(r) => r,
        Err(d) => {
            emit_error(&d);
            return ExitCode::Logical.code();
        }
    };

    // 7. Converge files.
    if let Err(d) = converge::converge_files(runner, &ctx, &intent, &config, &keep) {
        emit_error(&d);
        return ExitCode::Logical.code();
    }

    // 8. Converge units.
    if let Err(d) = converge::converge_units(runner, &ctx, &intent) {
        emit_error(&d);
        return ExitCode::Logical.code();
    }

    // 9. Write the applied record.
    if let Err(d) =
        record::write_applied_record(&ctx, &loaded.manifest, &loaded.desired_sha256, &resolved, &now)
    {
        emit_error(&d);
        return ExitCode::Logical.code();
    }

    // 10. Verify the converged tree against the new applied record.
    let new_applied = {
        let mut m = loaded.manifest.clone();
        m.packages = Some(resolved.clone());
        m.meta.desired_sha256 = loaded.desired_sha256.clone();
        m
    };
    match describe_actual_state(runner, &ctx.root, OnUnreadable::Error, ScanScope::Etc, &keep, &now)
    {
        Ok(actual) => {
            let drift = diffmod::compute_drift(&actual.manifest, &new_applied, &keep);
            if !drift.is_empty() {
                emit_error(&Diagnostic::error(
                    Domain::Files,
                    "post-converge verification found drift; transaction discarded",
                ));
                return ExitCode::Logical.code();
            }
        }
        Err(d) => {
            emit_error(&d);
            return ExitCode::Logical.code();
        }
    }

    // 11. Seal and activate (binding-dependent), emit summary, exit 0.
    println!(
        "applied: {} package(s), {} file(s) written, {} unit(s) changed",
        resolved.elements.len(),
        intent.files_write.len(),
        intent.units_change.len()
    );
    ExitCode::Ok.code()
}

// ---------------------------------------------------------------------------
// shared actual-state acquisition for diff/verify
// ---------------------------------------------------------------------------

/// Obtain the actual state: from a supplied state-path (offline) if given, else
/// live via describe-actual-state on "/". Returns an invocation-tagged error for
/// a malformed dump, or the live-read error otherwise.
fn obtain_actual_for_drift(
    runner: &dyn CommandRunner,
    opts: &ParsedOpts,
    config: &Config,
    keep: &HashSet<String>,
) -> Result<Manifest, Diagnostic> {
    if let Some(sp) = &opts.state_path {
        let fmt = resolve_format(opts.format, Some(sp), config.manifest_format);
        let m = record::load_state_dump(Path::new(sp), fmt)?;
        Ok(m)
    } else {
        let now = now_rfc3339();
        let actual =
            describe_actual_state(runner, "/", config.on_unreadable, config.scope, keep, &now)?;
        // Warn diagnostics surface to stderr.
        emit_diagnostics(&actual.diagnostics);
        Ok(actual.manifest)
    }
}

/// Map a load-desired-manifest diagnostic to an exit code: invocation -> 2,
/// else (manifest schema/unsafe-YAML/signature) -> 1.
fn exit_for_load_error(d: &Diagnostic) -> i32 {
    emit_error(d);
    match d.domain {
        Domain::Invocation => ExitCode::Invocation.code(),
        _ => ExitCode::Logical.code(),
    }
}

/// Map an actual-state acquisition error: a malformed dump (domain=invocation)
/// is exit 2; a live-read error is exit 1.
fn exit_for_actual_error(d: &Diagnostic) -> i32 {
    emit_error(d);
    match d.domain {
        Domain::Invocation => ExitCode::Invocation.code(),
        _ => ExitCode::Logical.code(),
    }
}

// ---------------------------------------------------------------------------
// version / usage
// ---------------------------------------------------------------------------

fn print_version() {
    println!(
        "{} {} spec:{}",
        meta::PROGRAM,
        meta::VERSION,
        meta::SPEC_SHA256
    );
}

const USAGE: &str = "\
usage: zypper-declarative <verb> [key=value ...]
       zypper declarative <verb> [key=value ...]

verbs:
  apply       converge the system to the desired manifest (privileged)
  diff        dry run: print what apply would change (read-only)
  verify      check actual state against a reference (read-only)
  status      print the current declarative state (read-only)
  describe    emit the actual declarable state as a manifest (read-only)
  version     print program name, version, and spec hash
  help        print this usage

options (key=value; precede or follow the verb):
  mode=auto|external|internal       transaction binding (default auto)
  manifest-path=<path>              desired/reference manifest
  format=json|yaml                  serialisation for this invocation's I/O
  state-path=<path>                 captured actual state (verify, diff)
  root=<path>                       root to describe (default /)
  out=<path>                        describe output file (default stdout)
  on-unreadable=error|warn          describe: fail (default) or omit+warn
  scope=etc|full                    describe/verify read scope (etc default)
  manifest-format, repo-lock, content-store, keep-list,
  signature-verification, keyring, activation-policy, applied-root

exit codes: 0 success; 1 logical failure; 2 invocation error.
";

fn print_usage_stdout() {
    print!("{}", USAGE);
}

fn print_usage_stderr() {
    eprint!("{}", USAGE);
}
