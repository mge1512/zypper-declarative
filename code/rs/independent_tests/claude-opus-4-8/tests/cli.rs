#![allow(dead_code)]
// tests by: claude-opus-4-8
// generated from spec: zypper-declarative.spec.md sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
//
// CLI dispatch and global-contract tests (black-box).
// Covers: version_verb_bare_word, version_flag_alias, help_verb_bare_word,
// bare_invocation_shows_help, unknown_verb_rejected, describe_unknown_format,
// status_no_declaration, status_unknown_argument, and the global
// argument-style invariants.

include!("common.rs");

const SPEC_HASH: &str = "51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03";

// EXAMPLE: version_verb_bare_word
#[test]
fn version_verb_bare_word() {
    let r = run_str(&["version"]);
    assert_eq!(r.code, 0, "version must exit 0; stderr={}", r.stderr);
    assert!(
        r.stdout.starts_with("zypper-declarative "),
        "stdout must start with program name; got: {:?}",
        r.stdout
    );
    assert!(
        r.stdout.contains("spec:"),
        "version output must embed the spec hash marker 'spec:'; got: {:?}",
        r.stdout
    );
    assert!(
        r.stdout.contains(SPEC_HASH),
        "version output must embed the exact spec sha256; got: {:?}",
        r.stdout
    );
}

// EXAMPLE: version_flag_alias  (--version is identical to bare-word version)
#[test]
fn version_flag_alias() {
    let bare = run_str(&["version"]);
    let flag = run_str(&["--version"]);
    assert_eq!(flag.code, 0, "--version must exit 0");
    assert_eq!(
        bare.stdout, flag.stdout,
        "--version stdout must equal bare-word version stdout"
    );
}

// EXAMPLE: help_verb_bare_word
#[test]
fn help_verb_bare_word() {
    let r = run_str(&["help"]);
    assert_eq!(r.code, 0, "help must exit 0; stderr={}", r.stderr);
    assert!(
        r.stdout.contains("usage:"),
        "help must print usage to stdout; got: {:?}",
        r.stdout
    );
}

// help flag aliases --help and -h print usage to stdout, exit 0
#[test]
fn help_flag_aliases() {
    for alias in ["--help", "-h"] {
        let r = run_str(&[alias]);
        assert_eq!(r.code, 0, "{} must exit 0", alias);
        assert!(
            r.stdout.contains("usage:"),
            "{} must print usage to stdout; got: {:?}",
            alias,
            r.stdout
        );
    }
}

// EXAMPLE: bare_invocation_shows_help
#[test]
fn bare_invocation_shows_help() {
    let r = run_str(&[]);
    assert_eq!(r.code, 0, "bare invocation must exit 0; stderr={}", r.stderr);
    assert!(
        r.stdout.contains("usage:"),
        "bare invocation prints usage to stdout; got: {:?}",
        r.stdout
    );
}

// EXAMPLE: unknown_verb_rejected
#[test]
fn unknown_verb_rejected() {
    let r = run_str(&["frobnicate"]);
    assert_eq!(r.code, 2, "unknown verb must exit 2; stdout={}", r.stdout);
    assert!(
        r.stderr.contains("usage:"),
        "unknown verb prints usage to stderr; got: {:?}",
        r.stderr
    );
}

// MILESTONE M0 acceptance: format=bad_value is an invocation error (exit 2).
#[test]
fn unknown_format_value_global_exits_2() {
    let r = run_str(&["format=bad_value"]);
    assert_eq!(
        r.code, 2,
        "an unknown/standalone format value must exit 2; stdout={} stderr={}",
        r.stdout, r.stderr
    );
}

// EXAMPLE: describe_unknown_format
#[test]
fn describe_unknown_format() {
    let r = run_str(&["describe", "format=toml"]);
    assert_eq!(r.code, 2, "unknown format must exit 2; stdout={}", r.stdout);
    assert!(
        r.stderr.contains("usage:") || r.stderr.to_lowercase().contains("invocation")
            || r.stderr.to_lowercase().contains("format"),
        "describe with unknown format prints diagnostic to stderr; got: {:?}",
        r.stderr
    );
}

// EXAMPLE: status_unknown_argument
#[test]
fn status_unknown_argument() {
    let r = run_str(&["status", "--frobnicate"]);
    assert_eq!(
        r.code, 2,
        "status with unrecognised argument must exit 2; stdout={}",
        r.stdout
    );
    assert!(
        !r.stderr.is_empty(),
        "status unknown argument writes a diagnostic to stderr"
    );
}

// EXAMPLE: status_no_declaration  (read against an applied-root with no record)
#[test]
fn status_no_declaration() {
    let empty_root = temp_dir("status-empty-root");
    let root_arg = format!("applied-root={}", empty_root.display());
    let r = run_str(&["status", &root_arg]);
    assert_eq!(
        r.code, 0,
        "status exits 0 even with no declaration applied; stderr={}",
        r.stderr
    );
    assert!(
        r.stdout.contains("no declaration applied"),
        "status with no applied record prints 'no declaration applied'; got: {:?}",
        r.stdout
    );
}

// Global invariant: options may appear in any position (after the verb too).
// We use `version` so the test does not depend on system state — an option
// appearing after a global verb must still be tolerated, not rejected as an
// unknown trailing argument for version.
#[test]
fn options_accepted_in_any_position_for_describe() {
    // describe with out= and other options placed in various positions; we use a
    // synthetic root so the run is bounded and deterministic (no live /etc walk).
    let root = temp_dir("anypos-root");
    std::fs::create_dir_all(root.join("etc")).unwrap();
    std::fs::write(root.join("etc/app.conf"), b"x\n").unwrap();
    let d = temp_dir("anypos");
    let out = d.join("state.json");
    let out_arg = format!("out={}", out.display());
    let root_arg = format!("root={}", root.display());
    // option before AND after — both positions must be accepted.
    let r = run_str(&[&out_arg, "describe", &root_arg, "on-unreadable=warn", "scope=etc"]);
    assert_eq!(
        r.code, 0,
        "describe with key=value options in any position must succeed; stderr={}",
        r.stderr
    );
    assert!(out.exists(), "out file must have been written");
}
