// generated from spec: zypper-declarative.spec.md
// sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
//
// CLI dispatch and the five verbs (apply, diff, verify, status, describe), plus
// the global commands (version, help) and the usage text. Exit-code mapping
// lives ONLY here: internal behaviours return domain-tagged Diagnostics, and
// this layer maps a terminal diagnostic's domain to an exit code.

pub mod render;
pub mod serialize;

use crate::config::{self, Invocation, OnUnreadable, Scope};
use crate::diff;
use crate::error::{exit_code_for, Diagnostic, Domain, EXIT_INVOCATION, EXIT_LOGICAL, EXIT_OK};
use crate::interfaces::{CommandRunner, OsCommandRunner};
use crate::manifest::load;
use crate::manifest::Manifest;
use crate::record;
use crate::state::{self, DescribeOptions};
use std::collections::HashSet;
use std::io::Write;

/// Entry from main(): parse argv and dispatch. Returns the process exit code.
pub fn run(args: &[String]) -> i32 {
    let inv = match config::parse(args) {
        Ok(inv) => inv,
        Err(diag) => {
            emit_usage_stderr();
            eprintln!("{}", diag.line());
            return EXIT_INVOCATION;
        }
    };

    // Global commands handled by the dispatcher (not behaviours).
    if inv.want_version {
        println!("{}", crate::meta::version_line());
        return EXIT_OK;
    }
    if inv.want_help {
        print!("{}", usage_text());
        return EXIT_OK;
    }

    let runner = OsCommandRunner;

    match inv.verb.as_deref() {
        None => {
            // Bare invocation: print usage to stdout, exit 0 (never converge).
            print!("{}", usage_text());
            EXIT_OK
        }
        Some("describe") => terminate(verb_describe(&inv, &runner)),
        Some("diff") => terminate(verb_diff(&inv, &runner)),
        Some("verify") => terminate(verb_verify(&inv, &runner)),
        Some("status") => terminate(verb_status(&inv, &runner)),
        Some("apply") => terminate(verb_apply(&inv, &runner)),
        Some(other) => {
            emit_usage_stderr();
            eprintln!("error: invocation: unknown verb '{}'", other);
            EXIT_INVOCATION
        }
    }
}

/// A verb's result: a chosen exit code, plus already-printed output. Verbs print
/// their own stdout/stderr and return either Ok(code) or Err(diagnostic).
type VerbResult = Result<i32, Diagnostic>;

fn terminate(r: VerbResult) -> i32 {
    match r {
        Ok(code) => code,
        Err(diag) => {
            eprintln!("{}", diag.line());
            exit_code_for(&diag)
        }
    }
}

fn keep_list_set(inv: &Invocation) -> HashSet<String> {
    let mut set = HashSet::new();
    if let Some(path) = &inv.keep_list {
        if let Ok(text) = std::fs::read_to_string(path) {
            for line in text.lines() {
                let l = line.trim();
                if !l.is_empty() && !l.starts_with('#') {
                    set.insert(l.to_string());
                }
            }
        }
    }
    set
}

fn describe_opts<'a>(
    inv: &Invocation,
    runner: &'a dyn CommandRunner,
    root: String,
    on_unreadable: OnUnreadable,
    scope: Scope,
) -> DescribeOptions<'a> {
    DescribeOptions {
        root,
        on_unreadable,
        scope,
        keep_list: keep_list_set(inv),
        content_store: inv.content_store.clone(),
        created_at: now_rfc3339(),
        runner,
    }
}

// ---- describe -----------------------------------------------------------

fn verb_describe(inv: &Invocation, runner: &dyn CommandRunner) -> VerbResult {
    let opts = describe_opts(
        inv,
        runner,
        inv.root_or_slash(),
        inv.on_unreadable_or_error(),
        inv.scope_or_etc(),
    );
    let result = state::describe_actual_state(&opts)?;
    for d in &result.diagnostics {
        eprintln!("{}", d.line());
    }
    // resolve output format against `out`
    let format = crate::manifest::format::resolve_format(
        inv.format,
        inv.out.as_deref(),
        inv.manifest_format_default(),
    );
    let document = serialize::serialise(&result.manifest, format)?;
    match &inv.out {
        Some(path) => {
            std::fs::write(path, document).map_err(|e| {
                Diagnostic::error(
                    Domain::Invocation,
                    format!("output path {} unwritable: {}", path, e),
                )
            })?;
        }
        None => {
            let mut stdout = std::io::stdout();
            stdout.write_all(document.as_bytes()).map_err(|e| {
                Diagnostic::error(Domain::Invocation, format!("stdout unwritable: {}", e))
            })?;
            if !document.ends_with('\n') {
                let _ = stdout.write_all(b"\n");
            }
        }
    }
    Ok(EXIT_OK)
}

// ---- diff ---------------------------------------------------------------

fn verb_diff(inv: &Invocation, runner: &dyn CommandRunner) -> VerbResult {
    let manifest_path = inv
        .manifest_path
        .clone()
        .ok_or_else(|| Diagnostic::error(Domain::Invocation, "diff requires manifest-path"))?;
    let loaded =
        load::load_desired_manifest(&manifest_path, inv.format, inv.manifest_format_default())?;
    let applied = record::load_applied_record(&inv.applied_root_or_slash())?;
    let intent = diff::compute_intent_diff(&loaded.manifest, &applied.record);

    // actual state: from state_path (offline) or live describe(scope=etc)
    let actual = actual_state_for(inv, runner)?;
    let keep = keep_list_set(inv);
    let drift = diff::compute_drift(&actual, &applied.record, &keep);

    print!("{}", render::render_plan(&intent, &drift));
    Ok(EXIT_OK)
}

// ---- verify -------------------------------------------------------------

fn verb_verify(inv: &Invocation, runner: &dyn CommandRunner) -> VerbResult {
    // 1. determine the reference
    let reference: Manifest = if let Some(mp) = &inv.manifest_path {
        load::load_desired_manifest(mp, inv.format, inv.manifest_format_default())?.manifest
    } else {
        let applied = record::load_applied_record(&inv.applied_root_or_slash())?;
        if !applied.present {
            return Err(Diagnostic::error(
                Domain::Invocation,
                "no declaration applied",
            ));
        }
        applied.record
    };

    // 2. obtain actual state
    let actual = actual_state_for(inv, runner)?;

    // 3. compute drift
    let keep = keep_list_set(inv);
    let drift = diff::compute_drift(&actual, &reference, &keep);

    // 4. report
    if drift.is_empty() {
        println!("system matches declaration");
        Ok(EXIT_OK)
    } else {
        for f in &drift.files_modified {
            eprintln!(
                "{}",
                Diagnostic::error(Domain::Files, format!("modified: {}", f)).line()
            );
        }
        for f in &drift.files_extra {
            eprintln!(
                "{}",
                Diagnostic::error(Domain::Files, format!("extra: {}", f)).line()
            );
        }
        for u in &drift.units_divergent {
            eprintln!(
                "{}",
                Diagnostic::error(Domain::Services, format!("divergent unit: {}", u.name)).line()
            );
        }
        for p in &drift.packages_divergent {
            eprintln!(
                "{}",
                Diagnostic::error(Domain::Packages, format!("divergent package: {}", p.name))
                    .line()
            );
        }
        for f in &drift.managed_files_modified {
            eprintln!(
                "{}",
                Diagnostic::error(Domain::Files, format!("managed-modified: {}", f)).line()
            );
        }
        for f in &drift.unmanaged_files_present {
            eprintln!(
                "{}",
                Diagnostic::error(Domain::Files, format!("unmanaged: {}", f)).line()
            );
        }
        Ok(EXIT_LOGICAL)
    }
}

// ---- status -------------------------------------------------------------

fn verb_status(inv: &Invocation, runner: &dyn CommandRunner) -> VerbResult {
    // status takes no options beyond CONFIG; an unrecognised argument was
    // already rejected by config::parse. (applied-root is a CONFIG knob.)
    let applied = record::load_applied_record(&inv.applied_root_or_slash())?;
    if !applied.present {
        println!("no declaration applied");
        return Ok(EXIT_OK);
    }
    let rec = &applied.record;
    println!("desired_sha256: {}", rec.meta.desired_sha256);
    println!("format_version: {}", rec.meta.format_version);
    println!("generation: {}", inv.applied_root_or_slash());
    println!("created_at: {}", rec.meta.created_at);
    let pkg_count = rec.packages.as_ref().map(|p| p.elements.len()).unwrap_or(0);
    println!("packages: {}", pkg_count);

    // drift summary
    let opts = describe_opts(
        inv,
        runner,
        "/".to_string(),
        OnUnreadable::Error,
        Scope::Etc,
    );
    match state::describe_actual_state(&opts) {
        Ok(result) => {
            let keep = keep_list_set(inv);
            let drift = diff::compute_drift(&result.manifest, rec, &keep);
            if drift.is_empty() {
                println!("clean");
            } else {
                println!("{} drift item(s)", drift.count());
            }
        }
        Err(_) => {
            println!("drift: unavailable");
        }
    }
    Ok(EXIT_OK)
}

// ---- apply --------------------------------------------------------------

fn verb_apply(inv: &Invocation, runner: &dyn CommandRunner) -> VerbResult {
    let manifest_path = inv
        .manifest_path
        .clone()
        .ok_or_else(|| Diagnostic::error(Domain::Invocation, "apply requires manifest-path"))?;
    // 1. load desired
    let loaded =
        load::load_desired_manifest(&manifest_path, inv.format, inv.manifest_format_default())?;
    // 2. load applied
    let applied = record::load_applied_record(&inv.applied_root_or_slash())?;
    // 3. intent diff
    let intent = diff::compute_intent_diff(&loaded.manifest, &applied.record);
    // 4. if empty, check live drift; if also empty -> nothing to do, exit 0
    let keep = keep_list_set(inv);
    if intent.is_empty() {
        let opts = describe_opts(
            inv,
            runner,
            "/".to_string(),
            OnUnreadable::Error,
            Scope::Etc,
        );
        let live = state::describe_actual_state(&opts)?;
        let drift = diff::compute_drift(&live.manifest, &applied.record, &keep);
        if drift.is_empty() {
            println!("nothing to do");
            return Ok(EXIT_OK);
        }
    }
    // 5. acquire transaction context (host-only; fails in non-privileged env)
    let ctx = crate::txn::acquire_transaction_context(&inv.transaction_mode_or_auto())?;
    // 6. converge packages (repos first)
    let resolved = crate::converge::converge_packages(runner, &ctx, &intent)?;
    // 7. converge files
    let rpm_owned = |_p: &str| false;
    crate::converge::converge_files(
        &ctx,
        &intent,
        &keep,
        inv.content_store.as_deref(),
        &rpm_owned,
    )?;
    // 8. converge units
    crate::converge::converge_units(runner, &ctx, &intent)?;
    // 9. write applied record
    let record_manifest = record::build_applied_record(
        &loaded.manifest,
        &loaded.desired_sha256,
        resolved,
        now_rfc3339(),
    );
    record::write_applied_record(&ctx.root, &record_manifest)?;
    // 10. post-converge verification
    let opts = describe_opts(
        inv,
        runner,
        ctx.root.clone(),
        OnUnreadable::Error,
        Scope::Etc,
    );
    let after = state::describe_actual_state(&opts)?;
    let drift = diff::compute_drift(&after.manifest, &record_manifest, &keep);
    if !drift.is_empty() {
        return Err(Diagnostic::error(
            Domain::Files,
            "post-converge verification found drift; transaction discarded",
        ));
    }
    // 11. seal/activate (host-only); emit summary
    println!("applied {} change(s)", intent_count(&intent));
    Ok(EXIT_OK)
}

fn intent_count(d: &diff::Diff) -> usize {
    d.packages_install.len()
        + d.packages_remove.len()
        + d.repos_set.len()
        + d.files_write.len()
        + d.files_delete.len()
        + d.units_change.len()
}

// ---- shared helpers -----------------------------------------------------

/// Obtain actual state from a supplied dump (offline) or a live describe(etc).
fn actual_state_for(inv: &Invocation, runner: &dyn CommandRunner) -> Result<Manifest, Diagnostic> {
    if let Some(sp) = &inv.state_path {
        load::load_state_dump(sp, inv.format, inv.manifest_format_default())
    } else {
        let scope = match inv.verb.as_deref() {
            // verify may carry scope=full; diff/status/apply always etc
            Some("verify") => inv.scope_or_etc(),
            _ => Scope::Etc,
        };
        let opts = describe_opts(inv, runner, inv.root_or_slash(), OnUnreadable::Error, scope);
        Ok(state::describe_actual_state(&opts)?.manifest)
    }
}

pub fn now_rfc3339() -> String {
    let now = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .unwrap_or_default();
    let secs = now.as_secs();
    format_rfc3339(secs)
}

// Convert a Unix timestamp (UTC) into an RFC3339 string. Pure arithmetic; no
// external dependency.
fn format_rfc3339(secs: u64) -> String {
    let days = secs / 86400;
    let rem = secs % 86400;
    let hour = rem / 3600;
    let min = (rem % 3600) / 60;
    let sec = rem % 60;
    let (y, m, d) = civil_from_days(days as i64);
    format!(
        "{:04}-{:02}-{:02}T{:02}:{:02}:{:02}Z",
        y, m, d, hour, min, sec
    )
}

// Howard Hinnant's days->civil algorithm.
fn civil_from_days(z: i64) -> (i64, u32, u32) {
    let z = z + 719468;
    let era = if z >= 0 { z } else { z - 146096 } / 146097;
    let doe = z - era * 146097;
    let yoe = (doe - doe / 1460 + doe / 36524 - doe / 146096) / 365;
    let y = yoe + era * 400;
    let doy = doe - (365 * yoe + yoe / 4 - yoe / 100);
    let mp = (5 * doy + 2) / 153;
    let d = (doy - (153 * mp + 2) / 5 + 1) as u32;
    let m = (if mp < 10 { mp + 3 } else { mp - 9 }) as u32;
    let y = if m <= 2 { y + 1 } else { y };
    (y, m, d)
}

pub fn emit_usage_stderr() {
    eprint!("{}", usage_text());
}

pub fn usage_text() -> String {
    format!(
        "usage: {prog} <verb> [key=value ...]\n\
         \n\
         verbs:\n\
         \x20 apply      converge the system to the desired manifest in a snapshot transaction\n\
         \x20 diff       print what apply would change (no modification)\n\
         \x20 verify     check the actual state against a reference declaration\n\
         \x20 status     print the current declarative state\n\
         \x20 describe   read the actual state and emit it as a manifest\n\
         \n\
         global commands:\n\
         \x20 version    print program name, version, and embedded spec hash (alias: --version)\n\
         \x20 help       print this usage (aliases: --help, -h)\n\
         \n\
         options (key=value; any position):\n\
         \x20 mode=auto|external|internal       transaction binding (default auto)\n\
         \x20 manifest-path=<path>              desired/reference manifest\n\
         \x20 state-path=<path>                 captured actual state (offline)\n\
         \x20 format=json|yaml                  serialisation for this invocation's I/O\n\
         \x20 root=<path>                       root to describe (default /)\n\
         \x20 out=<path>                        describe output file (default stdout)\n\
         \x20 on-unreadable=error|warn          describe: fail or omit+warn (default error)\n\
         \x20 scope=etc|full                    describe/verify read scope (default etc)\n\
         \x20 content-store=<path>              file content store base path\n\
         \x20 keep-list=<path>                  allowlist of persistent-but-undeclared paths\n\
         \x20 applied-root=<path>               generation root for the applied record (default /)\n\
         \x20 manifest-format=json|yaml         fallback serialisation default\n",
        prog = crate::meta::PROGRAM_NAME
    )
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn rfc3339_epoch_is_1970() {
        assert_eq!(format_rfc3339(0), "1970-01-01T00:00:00Z");
    }

    #[test]
    fn rfc3339_known_timestamp() {
        // 2026-06-02T09:20:00Z = 1780392000
        assert_eq!(format_rfc3339(1780392000), "2026-06-02T09:20:00Z");
    }

    #[test]
    fn usage_contains_keyword() {
        assert!(usage_text().contains("usage:"));
    }
}
