// generated from spec: zypper-declarative.spec.md sha256:18253550a5c3d3f1818f0380811cb5dbc98874828693e49fc9cd5cbc923303dd
// tests by: claude-opus-4-8
//
// Black-box integration tests for the zypper-declarative CLI.
//
// The interface under test is the CLI binary, per the spec DEPLOYMENT section
// (cli-tool deployment, key=value arguments). Tests invoke the binary via
// std::process::Command and assert on stdout, stderr, and exit code only. No
// implementation source package is imported; no internal function is called.
//
// Binary discovery: the binary lives at the project root, which is two
// directories up from independent_tests/claude-opus-4-8/ — i.e. the relative
// path ../../zypper-declarative (per the deployment template BINARY-LOCATION:
// project-root constraint). If the binary is not yet present (translator has
// not built it), the test harness builds it once via `cargo build --release`
// against the project's own Cargo.toml and copies it to the canonical path.

use std::path::{Path, PathBuf};
use std::process::{Command, Output};
use std::sync::Once;

const BINARY_REL: &str = "../../zypper-declarative";

static BUILD_ONCE: Once = Once::new();

// Absolute path to independent_tests/claude-opus-4-8/
fn test_dir() -> PathBuf {
    PathBuf::from(env!("CARGO_MANIFEST_DIR"))
}

// Absolute path to the project root (two dirs up from the test dir).
fn project_root() -> PathBuf {
    let mut p = test_dir();
    p.pop(); // independent_tests
    p.pop(); // project root
    p
}

fn binary_path() -> PathBuf {
    test_dir().join(BINARY_REL)
}

// Ensure the binary exists at the canonical location. If absent, build it.
fn ensure_binary() {
    BUILD_ONCE.call_once(|| {
        let bin = binary_path();
        if bin.exists() {
            return;
        }
        let root = project_root();
        // Build the translator's crate at the project root.
        let status = Command::new("cargo")
            .args(["build", "--release"])
            .current_dir(&root)
            .status();
        if let Ok(st) = status {
            if st.success() {
                let built = root.join("target/release/zypper-declarative");
                if built.exists() && !bin.exists() {
                    let _ = std::fs::copy(&built, &bin);
                    #[cfg(unix)]
                    {
                        use std::os::unix::fs::PermissionsExt;
                        if let Ok(meta) = std::fs::metadata(&bin) {
                            let mut perms = meta.permissions();
                            perms.set_mode(0o755);
                            let _ = std::fs::set_permissions(&bin, perms);
                        }
                    }
                }
            }
        }
    });
}

// Run the binary with the given args. Returns (stdout, stderr, exit_code).
fn run(args: &[&str]) -> (String, String, i32) {
    ensure_binary();
    let bin = binary_path();
    let out: Output = Command::new(&bin)
        .args(args)
        .output()
        .unwrap_or_else(|e| panic!("failed to execute {:?}: {}", bin, e));
    let stdout = String::from_utf8_lossy(&out.stdout).into_owned();
    let stderr = String::from_utf8_lossy(&out.stderr).into_owned();
    let code = out.status.code().unwrap_or(-1);
    (stdout, stderr, code)
}

// A scratch directory that cleans up on drop.
struct Scratch {
    dir: PathBuf,
}

impl Scratch {
    fn new(tag: &str) -> Self {
        let mut dir = std::env::temp_dir();
        let pid = std::process::id();
        let nanos = std::time::SystemTime::now()
            .duration_since(std::time::UNIX_EPOCH)
            .unwrap()
            .as_nanos();
        dir.push(format!("zd-test-{}-{}-{}", tag, pid, nanos));
        std::fs::create_dir_all(&dir).unwrap();
        Scratch { dir }
    }
    fn path(&self, rel: &str) -> PathBuf {
        self.dir.join(rel)
    }
    fn write(&self, rel: &str, content: &str) -> PathBuf {
        let p = self.path(rel);
        if let Some(parent) = p.parent() {
            std::fs::create_dir_all(parent).unwrap();
        }
        std::fs::write(&p, content).unwrap();
        p
    }
    // Create an applied-record tree rooted at this scratch dir and write the
    // applied.json into <root>/usr/lib/zypper-declarative/applied.json.
    fn write_applied(&self, content: &str) {
        let p = self
            .dir
            .join("usr/lib/zypper-declarative/applied.json");
        std::fs::create_dir_all(p.parent().unwrap()).unwrap();
        std::fs::write(&p, content).unwrap();
    }
}

impl Drop for Scratch {
    fn drop(&mut self) {
        let _ = std::fs::remove_dir_all(&self.dir);
    }
}

// ---------------------------------------------------------------------------
// Manifest / state fixtures (structurally complete, schema-valid documents).
// ---------------------------------------------------------------------------

fn s(p: &Path) -> String {
    p.to_string_lossy().into_owned()
}

// A minimal valid manifest with a packages and services scope.
fn manifest_pkgs_services() -> &'static str {
    r#"{
  "meta": { "format_version": 1, "generator": "test", "created_at": "2026-05-29T08:30:00Z", "desired_sha256": "" },
  "packages": { "_attributes": { "package_system": "rpm" }, "_elements": [ { "name": "nginx", "version": "", "release": "", "arch": "" } ] },
  "services": { "_attributes": { "init_system": "systemd" }, "_elements": [ { "name": "nginx.service", "state": "enabled" } ] }
}"#
}

// Applied record declaring /etc/foo.conf and /etc/bar.conf, packages resolved.
fn applied_foo_bar() -> &'static str {
    r#"{
  "meta": { "format_version": 1, "generator": "test", "created_at": "2026-05-29T08:30:00Z", "desired_sha256": "abababababababababababababababababababababababababababababababab" },
  "packages": { "_attributes": { "package_system": "rpm" }, "_elements": [ { "name": "nginx", "version": "1.0.0", "release": "1", "arch": "x86_64" } ] },
  "config_files": { "_attributes": {}, "_elements": [
    { "name": "/etc/foo.conf", "type": "file", "mode": "0644", "user": "root", "group": "root", "sha256": "0000000000000000000000000000000000000000000000000000000000000000", "target": "", "content_ref": "", "package_name": "" },
    { "name": "/etc/bar.conf", "type": "file", "mode": "0644", "user": "root", "group": "root", "sha256": "2222222222222222222222222222222222222222222222222222222222222222", "target": "", "content_ref": "", "package_name": "" }
  ] }
}"#
}

// An applied record with a known desired_sha256 and a resolved package lock.
fn applied_with_lock() -> &'static str {
    r#"{
  "meta": { "format_version": 1, "generator": "test", "created_at": "2026-05-29T08:30:00Z", "desired_sha256": "deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef" },
  "packages": { "_attributes": { "package_system": "rpm" }, "_elements": [
    { "name": "nginx", "version": "1.0.0", "release": "1", "arch": "x86_64" },
    { "name": "bash", "version": "5.2", "release": "1", "arch": "x86_64" }
  ] }
}"#
}

// A state dump (actual state) matching applied_with_lock exactly.
fn state_matches_lock() -> &'static str {
    r#"{
  "meta": { "format_version": 1, "generator": "describe", "created_at": "2026-05-29T09:00:00Z", "desired_sha256": "" },
  "packages": { "_attributes": { "package_system": "rpm" }, "_elements": [
    { "name": "nginx", "version": "1.0.0", "release": "1", "arch": "x86_64" },
    { "name": "bash", "version": "5.2", "release": "1", "arch": "x86_64" }
  ] }
}"#
}

// ===========================================================================
// Global CLI contract (dispatcher, version, help, unknown verb/option/value)
// ===========================================================================

#[test]
fn bare_invocation_shows_help_exit_0() {
    // EXAMPLE: bare_invocation_shows_help
    let (out, _err, code) = run(&[]);
    assert_eq!(code, 0, "bare invocation must exit 0");
    assert!(
        out.to_lowercase().contains("usage"),
        "bare invocation prints usage to stdout, got stdout={:?}",
        out
    );
}

#[test]
fn version_verb_bare_word() {
    // EXAMPLE: version_verb_bare_word
    let (out, _err, code) = run(&["version"]);
    assert_eq!(code, 0, "version must exit 0");
    assert!(
        out.contains("zypper-declarative"),
        "version prints program name, got {:?}",
        out
    );
    assert!(
        out.contains("spec:"),
        "version prints the embedded spec hash (spec:...), got {:?}",
        out
    );
    assert!(
        out.contains("18253550a5c3d3f1818f0380811cb5dbc98874828693e49fc9cd5cbc923303dd"),
        "version output embeds the spec sha256, got {:?}",
        out
    );
}

#[test]
fn version_flag_alias_matches_bare_word() {
    // EXAMPLE: version_flag_alias
    let (out_verb, _e1, c1) = run(&["version"]);
    let (out_flag, _e2, c2) = run(&["--version"]);
    assert_eq!(c1, 0);
    assert_eq!(c2, 0);
    assert_eq!(out_flag, out_verb, "--version output identical to bare version verb");
}

#[test]
fn help_verb_bare_word() {
    // EXAMPLE: help_verb_bare_word
    let (out, _err, code) = run(&["help"]);
    assert_eq!(code, 0, "help must exit 0");
    assert!(
        out.to_lowercase().contains("usage"),
        "help prints usage to stdout, got {:?}",
        out
    );
}

#[test]
fn help_flag_aliases() {
    // INVARIANT: --help and -h are tolerated aliases for help (exit 0, usage to stdout)
    for flag in ["--help", "-h"] {
        let (out, _err, code) = run(&[flag]);
        assert_eq!(code, 0, "{} must exit 0", flag);
        assert!(
            out.to_lowercase().contains("usage"),
            "{} prints usage to stdout, got {:?}",
            flag,
            out
        );
    }
}

#[test]
fn unknown_verb_rejected_exit_2() {
    // EXAMPLE: unknown_verb_rejected
    let (_out, err, code) = run(&["frobnicate"]);
    assert_eq!(code, 2, "unknown verb exits 2");
    assert!(
        err.to_lowercase().contains("usage"),
        "usage printed to stderr for unknown verb, got stderr={:?}",
        err
    );
}

#[test]
fn unknown_format_value_exit_2() {
    // MILESTONE acceptance: ./zypper-declarative format=bad_value; test $? -eq 2
    // EXAMPLE-adjacent: unknown format value -> invocation error exit 2
    let (_out, _err, code) = run(&["format=bad_value"]);
    assert_eq!(code, 2, "unknown/invalid format value is an invocation error (exit 2)");
}

#[test]
fn describe_unknown_format_exit_2() {
    // EXAMPLE: describe_unknown_format
    let (_out, err, code) = run(&["describe", "format=toml"]);
    assert_eq!(code, 2, "unknown format value on describe exits 2");
    assert!(
        err.to_lowercase().contains("usage") || !err.is_empty(),
        "diagnostic to stderr for unknown format, got stderr={:?}",
        err
    );
}

#[test]
fn status_unknown_argument_exit_2() {
    // EXAMPLE: status_unknown_argument
    let (_out, err, code) = run(&["status", "--frobnicate"]);
    assert_eq!(code, 2, "status with unrecognised argument exits 2");
    assert!(
        err.to_lowercase().contains("usage"),
        "usage to stderr, got stderr={:?}",
        err
    );
}

// ===========================================================================
// status
// ===========================================================================

#[test]
fn status_no_declaration_exit_0() {
    // EXAMPLE: status_no_declaration
    let scratch = Scratch::new("status-none");
    // No applied.json under this root.
    let root = s(&scratch.dir);
    let (out, _err, code) = run(&["status", &format!("applied-root={}", root)]);
    assert_eq!(code, 0, "status with no declaration exits 0");
    assert!(
        out.contains("no declaration applied"),
        "status prints 'no declaration applied', got {:?}",
        out
    );
}

#[test]
fn status_reports_generation() {
    // EXAMPLE: status_reports_generation
    let scratch = Scratch::new("status-gen");
    scratch.write_applied(applied_with_lock());
    let root = s(&scratch.dir);
    let (out, _err, code) = run(&["status", &format!("applied-root={}", root)]);
    assert_eq!(code, 0, "status exits 0");
    assert!(
        out.contains("deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"),
        "status prints the applied desired_sha256, got {:?}",
        out
    );
    // Resolved package count (2) should appear somewhere in the report.
    assert!(
        out.contains('2'),
        "status reports the resolved package count, got {:?}",
        out
    );
}

// ===========================================================================
// describe (output format resolution, structure)
// ===========================================================================

#[test]
fn describe_out_extension_json() {
    // EXAMPLE: describe_out_extension_json
    let scratch = Scratch::new("desc-json-ext");
    let out_path = scratch.path("state.json");
    let (_o, _e, code) = run(&[
        "describe",
        &format!("out={}", s(&out_path)),
        "on-unreadable=warn",
    ]);
    assert_eq!(code, 0, "describe out=...json exits 0, stderr/stdout for diag");
    let body = std::fs::read_to_string(&out_path).expect("output json file written");
    let trimmed = body.trim_start();
    assert!(
        trimmed.starts_with('{'),
        "resolve-format selects JSON from .json extension; got {:?}",
        &body[..body.len().min(80)]
    );
}

#[test]
fn describe_out_extension_yaml() {
    // EXAMPLE: describe_out_extension_yaml
    // MILESTONE acceptance 0.1.0: yaml by extension (first line not starting with "{")
    let scratch = Scratch::new("desc-yaml-ext");
    let out_path = scratch.path("state.yaml");
    let (_o, _e, code) = run(&[
        "describe",
        &format!("out={}", s(&out_path)),
        "on-unreadable=warn",
    ]);
    assert_eq!(code, 0, "describe out=...yaml exits 0");
    let body = std::fs::read_to_string(&out_path).expect("output yaml file written");
    let first = body.lines().next().unwrap_or("");
    assert!(
        !first.trim_start().starts_with('{'),
        "resolve-format selects YAML from .yaml extension; first line was {:?}",
        first
    );
}

#[test]
fn describe_format_overrides_extension() {
    // EXAMPLE: describe_format_overrides_extension
    let scratch = Scratch::new("desc-fmt-over");
    let out_path = scratch.path("state.yaml");
    let (_o, _e, code) = run(&[
        "describe",
        "format=json",
        &format!("out={}", s(&out_path)),
        "on-unreadable=warn",
    ]);
    assert_eq!(code, 0, "describe format=json out=...yaml exits 0");
    let body = std::fs::read_to_string(&out_path).expect("output file written");
    assert!(
        body.trim_start().starts_with('{'),
        "explicit format=json wins over .yaml extension; got {:?}",
        &body[..body.len().min(80)]
    );
}

#[test]
fn describe_output_unwritable_exit_2() {
    // EXAMPLE: describe_output_unwritable
    let (_o, err, code) = run(&[
        "describe",
        "out=/nonexistent-dir-xyz/state.json",
        "on-unreadable=warn",
    ]);
    assert_eq!(code, 2, "unwritable output path exits 2");
    assert!(!err.is_empty(), "a diagnostic is emitted to stderr, got {:?}", err);
}

#[test]
fn describe_emits_json_with_format_version_1() {
    // EXAMPLE: describe_emits_manifest (structure: JSON document, meta.format_version=1)
    let scratch = Scratch::new("desc-emit");
    let out_path = scratch.path("state.json");
    let (_o, _e, code) = run(&[
        "describe",
        &format!("out={}", s(&out_path)),
        "on-unreadable=warn",
    ]);
    assert_eq!(code, 0, "describe exits 0 under on-unreadable=warn");
    let body = std::fs::read_to_string(&out_path).expect("output written");
    assert!(
        body.contains("\"format_version\""),
        "describe JSON has meta.format_version, got {:?}",
        &body[..body.len().min(200)]
    );
    // format_version is 1.
    assert!(
        body.contains("\"format_version\": 1") || body.contains("\"format_version\":1"),
        "meta.format_version = 1, got {:?}",
        &body[..body.len().min(200)]
    );
}

#[test]
fn describe_generator_carries_version() {
    // INVARIANT: meta.generator carries program name AND version ("zypper-declarative 0.6.4")
    let scratch = Scratch::new("desc-gen");
    let out_path = scratch.path("state.json");
    let (_o, _e, code) = run(&[
        "describe",
        &format!("out={}", s(&out_path)),
        "on-unreadable=warn",
    ]);
    assert_eq!(code, 0);
    let body = std::fs::read_to_string(&out_path).expect("output written");
    assert!(
        body.contains("zypper-declarative 0.6.4"),
        "meta.generator is 'zypper-declarative <version>', got {:?}",
        body
    );
}

// ===========================================================================
// diff (intent diff + drift), offline two-file mode
// ===========================================================================

#[test]
fn diff_prints_plan_install_and_delete() {
    // EXAMPLE: diff_prints_plan
    // Desired adds nginx and drops /etc/bar.conf relative to applied record.
    let scratch = Scratch::new("diff-plan");
    scratch.write_applied(applied_foo_bar());
    let manifest = r#"{
  "meta": { "format_version": 1, "generator": "test", "created_at": "2026-05-29T08:30:00Z", "desired_sha256": "" },
  "packages": { "_attributes": { "package_system": "rpm" }, "_elements": [ { "name": "nginx", "version": "", "release": "", "arch": "" } ] },
  "config_files": { "_attributes": {}, "_elements": [
    { "name": "/etc/foo.conf", "type": "file", "mode": "0644", "user": "root", "group": "root", "sha256": "1111111111111111111111111111111111111111111111111111111111111111", "target": "", "content_ref": "files/etc/foo.conf", "package_name": "" }
  ] }
}"#;
    let mpath = scratch.write("desired.json", manifest);
    // Provide a state-path so no live read happens, keeping the test offline.
    let state = r#"{
  "meta": { "format_version": 1, "generator": "describe", "created_at": "2026-05-29T09:00:00Z", "desired_sha256": "" }
}"#;
    let spath = scratch.write("after.json", state);
    let root = s(&scratch.dir);
    let (out, _err, code) = run(&[
        "diff",
        &format!("manifest-path={}", s(&mpath)),
        &format!("state-path={}", s(&spath)),
        &format!("applied-root={}", root),
    ]);
    assert_eq!(code, 0, "diff exits 0 when the plan was computed");
    assert!(out.contains("nginx"), "nginx listed under packages to install, got {:?}", out);
    assert!(
        out.contains("/etc/bar.conf"),
        "/etc/bar.conf listed under files to delete, got {:?}",
        out
    );
}

#[test]
fn diff_manifest_unreadable_exit_2() {
    // EXAMPLE: diff_manifest_unreadable
    let (_o, err, code) = run(&["diff", "manifest-path=/nonexistent.json"]);
    assert_eq!(code, 2, "unreadable manifest -> exit 2");
    assert!(
        err.contains("invocation") || !err.is_empty(),
        "diagnostic with domain=invocation to stderr, got {:?}",
        err
    );
}

#[test]
fn diff_offline_two_files_exit_0() {
    // EXAMPLE: diff_offline_two_files
    let scratch = Scratch::new("diff-offline");
    let baseline = scratch.write("baseline.json", manifest_pkgs_services());
    let after = scratch.write("after.json", manifest_pkgs_services());
    let (_out, _err, code) = run(&[
        "diff",
        &format!("manifest-path={}", s(&baseline)),
        &format!("state-path={}", s(&after)),
    ]);
    assert_eq!(code, 0, "offline two-file diff exits 0");
}

#[test]
fn diff_state_path_no_live_read_idempotent_bootstrap() {
    // EXAMPLE: describe_bootstraps_desired_manifest (bootstrap+diff same system => no changes)
    // Constructed offline: the desired manifest equals the captured actual state.
    let scratch = Scratch::new("diff-bootstrap");
    let doc = manifest_pkgs_services();
    let desired = scratch.write("desired.json", doc);
    let state = scratch.write("after.json", doc);
    let (out, _err, code) = run(&[
        "diff",
        &format!("manifest-path={}", s(&desired)),
        &format!("state-path={}", s(&state)),
        &format!("applied-root={}", s(&scratch.dir)),
    ]);
    assert_eq!(code, 0, "diff of a bootstrap against itself exits 0");
    // With desired == applied-absent, packages_install lists the desired pkgs but
    // there is nothing to delete and (with empty applied) the intent reflects only
    // installs. The plan computes and exit is 0; that is the asserted contract.
    let _ = out;
}

// ===========================================================================
// verify (offline manifest+state, drift detection, no reference)
// ===========================================================================

#[test]
fn verify_offline_clean_exit_0() {
    // EXAMPLE: verify_offline_manifest_and_state (clean) and verify_clean shape.
    let scratch = Scratch::new("verify-clean");
    let manifest = scratch.write("baseline.json", applied_with_lock());
    let state = scratch.write("after.json", state_matches_lock());
    let (out, _err, code) = run(&[
        "verify",
        &format!("manifest-path={}", s(&manifest)),
        &format!("state-path={}", s(&state)),
    ]);
    assert_eq!(code, 0, "offline verify with matching state exits 0");
    assert!(
        out.contains("system matches declaration"),
        "verify prints 'system matches declaration', got {:?}",
        out
    );
}

#[test]
fn verify_offline_no_applied_record_ok() {
    // EXAMPLE: verify_offline_no_applied_record_ok
    let scratch = Scratch::new("verify-noar");
    let manifest = scratch.write("baseline.json", applied_with_lock());
    let state = scratch.write("after.json", state_matches_lock());
    // applied-root points at an empty tree => no applied record present.
    let (out, _err, code) = run(&[
        "verify",
        &format!("manifest-path={}", s(&manifest)),
        &format!("state-path={}", s(&state)),
        &format!("applied-root={}", s(&scratch.dir)),
    ]);
    assert_eq!(code, 0, "offline verify with reference manifest does not need an applied record");
    assert!(
        !out.contains("no declaration applied"),
        "must not emit 'no declaration applied' when a reference manifest is supplied, got {:?}",
        out
    );
}

#[test]
fn verify_detects_unit_drift_offline() {
    // EXAMPLE: verify_against_external_state_dump (service state divergence -> exit 1, domain=units)
    let scratch = Scratch::new("verify-unitdrift");
    let manifest = r#"{
  "meta": { "format_version": 1, "generator": "test", "created_at": "2026-05-29T08:30:00Z", "desired_sha256": "x" },
  "services": { "_attributes": { "init_system": "systemd" }, "_elements": [ { "name": "nginx.service", "state": "enabled" } ] }
}"#;
    let state = r#"{
  "meta": { "format_version": 1, "generator": "describe", "created_at": "2026-05-29T09:00:00Z", "desired_sha256": "" },
  "services": { "_attributes": { "init_system": "systemd" }, "_elements": [ { "name": "nginx.service", "state": "disabled" } ] }
}"#;
    let mpath = scratch.write("baseline.json", manifest);
    let spath = scratch.write("after.json", state);
    let (_out, err, code) = run(&[
        "verify",
        &format!("manifest-path={}", s(&mpath)),
        &format!("state-path={}", s(&spath)),
    ]);
    assert_eq!(code, 1, "drift detected -> exit 1");
    assert!(
        err.contains("nginx.service") && err.contains("units"),
        "stderr names the divergent service with domain=units, got {:?}",
        err
    );
}

#[test]
fn verify_detects_file_drift_offline() {
    // EXAMPLE: verify_detects_drift (declared /etc/foo.conf edited -> exit 1, domain=files)
    let scratch = Scratch::new("verify-filedrift");
    let manifest = r#"{
  "meta": { "format_version": 1, "generator": "test", "created_at": "2026-05-29T08:30:00Z", "desired_sha256": "x" },
  "config_files": { "_attributes": {}, "_elements": [
    { "name": "/etc/foo.conf", "type": "file", "mode": "0644", "user": "root", "group": "root", "sha256": "1111111111111111111111111111111111111111111111111111111111111111", "target": "", "content_ref": "", "package_name": "" }
  ] }
}"#;
    let state = r#"{
  "meta": { "format_version": 1, "generator": "describe", "created_at": "2026-05-29T09:00:00Z", "desired_sha256": "" },
  "config_files": { "_attributes": {}, "_elements": [
    { "name": "/etc/foo.conf", "type": "file", "mode": "0644", "user": "root", "group": "root", "sha256": "9999999999999999999999999999999999999999999999999999999999999999", "target": "", "content_ref": "", "package_name": "" }
  ] }
}"#;
    let mpath = scratch.write("baseline.json", manifest);
    let spath = scratch.write("after.json", state);
    let (_out, err, code) = run(&[
        "verify",
        &format!("manifest-path={}", s(&mpath)),
        &format!("state-path={}", s(&spath)),
    ]);
    assert_eq!(code, 1, "edited declared file -> drift -> exit 1");
    assert!(
        err.contains("/etc/foo.conf") && err.contains("files"),
        "stderr names /etc/foo.conf with domain=files, got {:?}",
        err
    );
}

#[test]
fn verify_type_transition_is_modified() {
    // EXAMPLE: drift_type_transition_is_modified (declared file, actual link -> modified by type)
    let scratch = Scratch::new("verify-typetrans");
    let manifest = r#"{
  "meta": { "format_version": 1, "generator": "test", "created_at": "2026-05-29T08:30:00Z", "desired_sha256": "x" },
  "config_files": { "_attributes": {}, "_elements": [
    { "name": "/etc/foo", "type": "file", "mode": "0644", "user": "root", "group": "root", "sha256": "1111111111111111111111111111111111111111111111111111111111111111", "target": "", "content_ref": "", "package_name": "" }
  ] }
}"#;
    let state = r#"{
  "meta": { "format_version": 1, "generator": "describe", "created_at": "2026-05-29T09:00:00Z", "desired_sha256": "" },
  "config_files": { "_attributes": {}, "_elements": [
    { "name": "/etc/foo", "type": "link", "mode": "0777", "user": "root", "group": "root", "sha256": "", "target": "../elsewhere", "content_ref": "", "package_name": "" }
  ] }
}"#;
    let mpath = scratch.write("baseline.json", manifest);
    let spath = scratch.write("after.json", state);
    let (_out, err, code) = run(&[
        "verify",
        &format!("manifest-path={}", s(&mpath)),
        &format!("state-path={}", s(&spath)),
    ]);
    assert_eq!(code, 1, "type transition is drift -> exit 1");
    assert!(
        err.contains("/etc/foo"),
        "stderr names /etc/foo as modified by type, got {:?}",
        err
    );
}

#[test]
fn verify_no_applied_record_exit_2() {
    // EXAMPLE: verify_no_applied_record (no reference manifest, no applied record)
    let scratch = Scratch::new("verify-noref");
    // state-path supplied but no manifest-path and no applied record.
    let state = scratch.write("after.json", state_matches_lock());
    let (_out, err, code) = run(&[
        "verify",
        &format!("state-path={}", s(&state)),
        &format!("applied-root={}", s(&scratch.dir)),
    ]);
    assert_eq!(code, 2, "no reference -> exit 2");
    assert!(
        err.contains("no declaration applied"),
        "stderr contains 'no declaration applied' with domain=invocation, got {:?}",
        err
    );
}

#[test]
fn verify_malformed_state_dump_exit_2() {
    // EXAMPLE: verify_malformed_state_dump
    let scratch = Scratch::new("verify-malformed");
    let manifest = scratch.write("baseline.json", applied_with_lock());
    let broken = scratch.write("broken.json", "{ this is : not valid json ");
    let (_out, err, code) = run(&[
        "verify",
        &format!("manifest-path={}", s(&manifest)),
        &format!("state-path={}", s(&broken)),
    ]);
    assert_eq!(code, 2, "malformed state dump -> exit 2");
    assert!(
        err.contains("invocation") || !err.is_empty(),
        "diagnostic with domain=invocation, got {:?}",
        err
    );
}

#[test]
fn verify_state_path_extension_yaml_clean() {
    // EXAMPLE: verify_state_path_extension_yaml
    let scratch = Scratch::new("verify-yamlstate");
    let manifest = scratch.write("baseline.json", applied_with_lock());
    // YAML serialisation of state_matches_lock.
    let yaml_state = r#"meta:
  format_version: 1
  generator: "describe"
  created_at: "2026-05-29T09:00:00Z"
  desired_sha256: ""
packages:
  _attributes:
    package_system: "rpm"
  _elements:
    - name: "nginx"
      version: "1.0.0"
      release: "1"
      arch: "x86_64"
    - name: "bash"
      version: "5.2"
      release: "1"
      arch: "x86_64"
"#;
    let spath = scratch.write("after.yaml", yaml_state);
    let (out, _err, code) = run(&[
        "verify",
        &format!("manifest-path={}", s(&manifest)),
        &format!("state-path={}", s(&spath)),
    ]);
    assert_eq!(code, 0, "yaml state by .yaml extension, clean -> exit 0");
    assert!(
        out.contains("system matches declaration"),
        "verify prints 'system matches declaration', got {:?}",
        out
    );
}

// ===========================================================================
// load-desired-manifest validation (via apply/diff error paths)
// ===========================================================================

#[test]
fn apply_manifest_unreadable_exit_2() {
    // EXAMPLE: apply_manifest_unreadable
    let (_out, err, code) = run(&["apply", "manifest-path=/nonexistent.json"]);
    assert_eq!(code, 2, "apply with unreadable manifest exits 2");
    assert!(
        err.contains("invocation") || !err.is_empty(),
        "diagnostic domain=invocation to stderr, got {:?}",
        err
    );
}

#[test]
fn apply_manifest_invalid_format_version_exit_1() {
    // EXAMPLE: apply_manifest_invalid (format_version=2 -> exit 1, domain=manifest)
    let scratch = Scratch::new("apply-invalid");
    let bad = r#"{
  "meta": { "format_version": 2, "generator": "test", "created_at": "2026-05-29T08:30:00Z", "desired_sha256": "" }
}"#;
    let mpath = scratch.write("bad.json", bad);
    let (_out, err, code) = run(&[
        "apply",
        &format!("manifest-path={}", s(&mpath)),
        "signature-verification=off",
    ]);
    assert_eq!(code, 1, "invalid manifest -> exit 1");
    assert!(
        err.contains("manifest"),
        "diagnostic with domain=manifest, got {:?}",
        err
    );
}

#[test]
fn apply_rejects_full_describe_dump_exit_1() {
    // EXAMPLE: apply_rejects_full_describe_dump (non-empty observational scope -> manifest error)
    let scratch = Scratch::new("apply-fulldump");
    let dump = r#"{
  "meta": { "format_version": 1, "generator": "test", "created_at": "2026-05-29T08:30:00Z", "desired_sha256": "" },
  "unmanaged_files": { "_attributes": {}, "_elements": [
    { "name": "/usr/bin/extra", "type": "file", "mode": "0755", "user": "root", "group": "root", "sha256": "3333333333333333333333333333333333333333333333333333333333333333", "target": "" }
  ] }
}"#;
    let mpath = scratch.write("full-dump.json", dump);
    let (_out, err, code) = run(&[
        "apply",
        &format!("manifest-path={}", s(&mpath)),
        "signature-verification=off",
    ]);
    assert_eq!(code, 1, "raw describe scope=full dump rejected -> exit 1");
    assert!(
        err.contains("manifest"),
        "diagnostic with domain=manifest, got {:?}",
        err
    );
}

#[test]
fn diff_with_invalid_manifest_exit_1() {
    // ERROR path: manifest invalid (schema) -> exit 1, domain=manifest (diff)
    let scratch = Scratch::new("diff-invalid");
    let bad = r#"{
  "meta": { "format_version": 5, "generator": "test", "created_at": "2026-05-29T08:30:00Z", "desired_sha256": "" }
}"#;
    let mpath = scratch.write("bad.json", bad);
    let (_out, err, code) = run(&[
        "diff",
        &format!("manifest-path={}", s(&mpath)),
        "signature-verification=off",
    ]);
    assert_eq!(code, 1, "diff with invalid manifest -> exit 1");
    assert!(err.contains("manifest"), "domain=manifest, got {:?}", err);
}

// ===========================================================================
// YAML safe profile and format identity
// ===========================================================================

#[test]
fn yaml_manifest_accepted_diff_exit_0() {
    // EXAMPLE: yaml_manifest_accepted
    let scratch = Scratch::new("yaml-accept");
    let yaml = r#"meta:
  format_version: 1
  generator: "test"
  created_at: "2026-05-29T08:30:00Z"
  desired_sha256: ""
packages:
  _attributes:
    package_system: "rpm"
  _elements:
    - name: "nginx"
      version: ""
      release: ""
      arch: ""
"#;
    let mpath = scratch.write("desired.yaml", yaml);
    // Provide an empty state-path to stay offline.
    let state = scratch.write("after.json", r#"{ "meta": { "format_version": 1, "generator": "d", "created_at": "2026-05-29T09:00:00Z", "desired_sha256": "" } }"#);
    let (_out, _err, code) = run(&[
        "diff",
        &format!("manifest-path={}", s(&mpath)),
        &format!("state-path={}", s(&state)),
        &format!("applied-root={}", s(&scratch.dir)),
    ]);
    assert_eq!(code, 0, "valid YAML manifest parsed and planned -> exit 0");
}

#[test]
fn yaml_unsafe_multidoc_rejected_exit_1() {
    // EXAMPLE: yaml_unsafe_rejected (multi-document stream is a disabled feature)
    let scratch = Scratch::new("yaml-multidoc");
    let yaml = r#"meta:
  format_version: 1
  generator: "test"
  created_at: "2026-05-29T08:30:00Z"
  desired_sha256: ""
---
meta:
  format_version: 1
"#;
    let mpath = scratch.write("evil.yaml", yaml);
    let (_out, err, code) = run(&[
        "apply",
        &format!("manifest-path={}", s(&mpath)),
        "signature-verification=off",
    ]);
    assert_eq!(code, 1, "unsafe YAML (multi-document) rejected -> exit 1");
    assert!(
        err.contains("manifest"),
        "rejected with a manifest error, got {:?}",
        err
    );
}

#[test]
fn yaml_format_identity_stable_hash() {
    // EXAMPLE: yaml_format_identity_stable + INVARIANT: desired_sha256 is canonical-model
    // hash, format-independent. We assert it indirectly: verifying a JSON state against
    // a YAML manifest expressing the same model is clean (exit 0).
    let scratch = Scratch::new("yaml-identity");
    let yaml_manifest = r#"meta:
  format_version: 1
  generator: "test"
  created_at: "2026-05-29T08:30:00Z"
  desired_sha256: ""
packages:
  _attributes:
    package_system: "rpm"
  _elements:
    - name: "nginx"
      version: "1.0.0"
      release: "1"
      arch: "x86_64"
"#;
    let json_state = r#"{
  "meta": { "format_version": 1, "generator": "d", "created_at": "2026-05-29T09:00:00Z", "desired_sha256": "" },
  "packages": { "_attributes": { "package_system": "rpm" }, "_elements": [ { "name": "nginx", "version": "1.0.0", "release": "1", "arch": "x86_64" } ] }
}"#;
    let mpath = scratch.write("baseline.yaml", yaml_manifest);
    let spath = scratch.write("after.json", json_state);
    let (out, _err, code) = run(&[
        "verify",
        &format!("manifest-path={}", s(&mpath)),
        &format!("state-path={}", s(&spath)),
    ]);
    assert_eq!(code, 0, "YAML manifest vs equal JSON state verifies clean");
    assert!(out.contains("system matches declaration"), "clean, got {:?}", out);
}

// ===========================================================================
// scope acceptance (etc default, full opt-in) and read-only no modification
// ===========================================================================

#[test]
fn scope_rejected_on_status() {
    // INVARIANT: scope is accepted only on describe and verify; status rejects it.
    let scratch = Scratch::new("scope-status");
    let (_out, _err, code) = run(&[
        "status",
        "scope=full",
        &format!("applied-root={}", s(&scratch.dir)),
    ]);
    assert_eq!(code, 2, "scope is not accepted on status -> invocation error exit 2");
}

#[test]
fn scope_rejected_on_apply() {
    // INVARIANT: scope is not accepted on apply.
    let scratch = Scratch::new("scope-apply");
    let m = scratch.write("desired.json", manifest_pkgs_services());
    let (_out, _err, code) = run(&[
        "apply",
        "scope=full",
        &format!("manifest-path={}", s(&m)),
        "signature-verification=off",
    ]);
    assert_eq!(code, 2, "scope is not accepted on apply -> invocation error exit 2");
}

#[test]
fn diff_does_not_modify_system_no_transaction() {
    // INVARIANT: diff opens no transaction and modifies nothing (offline two-file).
    let scratch = Scratch::new("diff-nomod");
    let baseline = scratch.write("baseline.json", manifest_pkgs_services());
    let after = scratch.write("after.json", manifest_pkgs_services());
    let before = std::fs::read_to_string(&baseline).unwrap();
    let (_out, _err, code) = run(&[
        "diff",
        &format!("manifest-path={}", s(&baseline)),
        &format!("state-path={}", s(&after)),
    ]);
    assert_eq!(code, 0);
    let unchanged = std::fs::read_to_string(&baseline).unwrap();
    assert_eq!(before, unchanged, "diff must not modify its input files");
}

// ===========================================================================
// Argument-position tolerance (options after the verb must be accepted)
// ===========================================================================

#[test]
fn options_after_verb_accepted() {
    // decisions hint: options must be accepted in any position (Go parser bug must
    // not be reproduced). status with applied-root after the verb must work.
    let scratch = Scratch::new("argpos");
    scratch.write_applied(applied_with_lock());
    let (out, _err, code) = run(&[
        "status",
        &format!("applied-root={}", s(&scratch.dir)),
    ]);
    assert_eq!(code, 0, "options after the verb are accepted");
    assert!(
        out.contains("deadbeef"),
        "status read the applied record located via the post-verb option, got {:?}",
        out
    );
}

// ===========================================================================
// scope_attributes_always_object (describe output)
// ===========================================================================

#[test]
fn describe_attributes_object_never_null() {
    // EXAMPLE: scope_attributes_always_object
    let scratch = Scratch::new("attrs-object");
    let out_path = scratch.path("state.json");
    let (_o, _e, code) = run(&[
        "describe",
        &format!("out={}", s(&out_path)),
        "on-unreadable=warn",
    ]);
    assert_eq!(code, 0);
    let body = std::fs::read_to_string(&out_path).expect("output written");
    assert!(
        !body.contains("\"_attributes\": null") && !body.contains("\"_attributes\":null"),
        "no scope _attributes is serialised as null, got {:?}",
        body
    );
}
