// generated from spec: zypper-declarative.spec.md sha256:87b9f2f3bf92afd6fe037412d56a23959290e5816d722a3def9505d07aa5acd7
// tests by: claude-opus-4-8
//
// Top-level CLI contract: version, help, bare invocation, unknown verb/option/
// value, and the tolerated flag aliases. Covers EXAMPLEs bare_invocation_shows_help,
// version_verb_bare_word, version_flag_alias, help_verb_bare_word,
// unknown_verb_rejected, status_unknown_argument, describe_unknown_format, and the
// observable INVARIANT on version/help bare words and POSIX-flag avoidance.

mod common;
use common::*;

const SPEC_HASH: &str = "87b9f2f3bf92afd6fe037412d56a23959290e5816d722a3def9505d07aa5acd7";

// EXAMPLE: version_verb_bare_word
#[test]
fn test_version_bare_word_prints_name_version_and_spec_hash() {
    let out = run(&["version"]);
    assert_eq!(exit_code(&out), 0, "version must exit 0; stderr={}", stderr_str(&out));
    let s = stdout_str(&out);
    assert!(
        s.contains("zypper-declarative"),
        "version output must contain the program name; got: {s}"
    );
    assert!(
        s.contains("spec:"),
        "version output must embed the spec hash marker 'spec:'; got: {s}"
    );
    assert!(
        s.contains(SPEC_HASH),
        "version output must embed the actual spec SHA256; got: {s}"
    );
}

// EXAMPLE: version_flag_alias  (--version identical to bare-word version)
#[test]
fn test_version_flag_alias_identical_to_bare_word() {
    let bare = run(&["version"]);
    let flag = run(&["--version"]);
    assert_eq!(exit_code(&bare), 0);
    assert_eq!(exit_code(&flag), 0);
    assert_eq!(
        stdout_str(&bare),
        stdout_str(&flag),
        "--version stdout must be identical to bare-word version stdout"
    );
}

// EXAMPLE: help_verb_bare_word
#[test]
fn test_help_bare_word_prints_usage_to_stdout_exit0() {
    let out = run(&["help"]);
    assert_eq!(exit_code(&out), 0, "help must exit 0; stderr={}", stderr_str(&out));
    let s = stdout_str(&out);
    assert!(
        s.to_lowercase().contains("usage:"),
        "help must print usage to stdout; got: {s}"
    );
}

// INVARIANT: --help and -h are tolerated aliases for help, exit 0, usage to stdout.
#[test]
fn test_help_flag_aliases_exit0_usage_stdout() {
    for alias in ["--help", "-h"] {
        let out = run(&[alias]);
        assert_eq!(exit_code(&out), 0, "{alias} must exit 0; stderr={}", stderr_str(&out));
        assert!(
            stdout_str(&out).to_lowercase().contains("usage:"),
            "{alias} must print usage to stdout"
        );
    }
}

// EXAMPLE: bare_invocation_shows_help  (no verb -> usage to stdout, exit 0)
#[test]
fn test_bare_invocation_usage_to_stdout_exit0() {
    let out = run(&[]);
    assert_eq!(
        exit_code(&out),
        0,
        "bare invocation is a discovery action and must exit 0; stderr={}",
        stderr_str(&out)
    );
    assert!(
        stdout_str(&out).to_lowercase().contains("usage:"),
        "bare invocation must print usage to stdout"
    );
}

// EXAMPLE: unknown_verb_rejected  (usage to stderr, exit 2)
#[test]
fn test_unknown_verb_usage_to_stderr_exit2() {
    let out = run(&["frobnicate"]);
    assert_eq!(exit_code(&out), 2, "unknown verb must exit 2");
    assert!(
        stderr_str(&out).to_lowercase().contains("usage")
            || !stderr_str(&out).is_empty(),
        "unknown verb must print usage/diagnostic to stderr"
    );
}

// EXAMPLE: status_unknown_argument  (status rejects unrecognised arg -> exit 2)
#[test]
fn test_status_unknown_argument_exit2() {
    let out = run(&["status", "--frobnicate"]);
    assert_eq!(exit_code(&out), 2, "status with unknown arg must exit 2");
    assert!(
        !stderr_str(&out).is_empty(),
        "status unknown argument must write a diagnostic/usage to stderr"
    );
}

// EXAMPLE: describe_unknown_format  (unknown format value -> usage to stderr, exit 2)
#[test]
fn test_describe_unknown_format_exit2() {
    let out = run(&["describe", "format=toml"]);
    assert_eq!(exit_code(&out), 2, "unknown format value must exit 2");
    assert!(
        !stderr_str(&out).is_empty(),
        "unknown format value must write a diagnostic to stderr"
    );
}

// M0 acceptance gate: a bad value for a known option is an invocation error.
#[test]
fn test_bad_format_value_exits_2() {
    let out = run(&["format=bad_value"]);
    assert_eq!(exit_code(&out), 2, "format=bad_value must exit 2 (invocation error)");
}

// INVARIANT: options accepted in any position (key=value after the verb too).
// describe is read-only and accepts format=; format after the verb must parse.
#[test]
fn test_option_after_verb_is_accepted() {
    // An explicit json format after the describe verb must be accepted as an
    // option, not rejected as an unknown bare word. We route output to a temp
    // file under a writable directory so we exercise only argument parsing.
    let dir = temp_dir("optpos");
    let out_path = dir.join("state.json");
    let out = run(&[
        "describe",
        &format!("out={}", out_path.display()),
        "format=json",
    ]);
    // The describe verb may succeed (exit 0) or fail to read a scope (exit 1)
    // depending on the host, but it must NOT be an invocation error (exit 2)
    // caused by rejecting the option after the verb.
    assert_ne!(
        exit_code(&out),
        2,
        "key=value option after the verb must be accepted, not rejected as exit 2; stderr={}",
        stderr_str(&out)
    );
}
