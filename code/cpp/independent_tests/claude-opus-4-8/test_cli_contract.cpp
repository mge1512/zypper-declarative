// generated from spec: zypper-declarative.spec.md sha256:f2cc80627e483a48bb8411d297711bc5f6c6e74c28dbf0dafc8fe7bd8817251e
// tests by: claude-opus-4-8
//
// CLI contract tests: bare-word global commands (version, help), bare
// invocation, flag aliases, unknown verb/option/value, exit-code mapping.
// Covers EXAMPLES: bare_invocation_shows_help, version_verb_bare_word,
// version_flag_alias, help_verb_bare_word, unknown_verb_rejected,
// describe_unknown_format, and the M0/0.1.0 acceptance gates. Also the
// [observable] CLI invariants about version/help and POSIX-flag style.
#include "test_harness.hpp"

using namespace zdtest;

// EXAMPLE: version_verb_bare_word
// stdout contains program name, version, and embedded spec hash; exit 0.
TEST(test_version_bare_word) {
    auto r = run({"version"});
    expect_eq_int(r.code, 0, "version exit");
    expect_contains(r.out, "zypper-declarative", "version program name");
    expect_contains(r.out, "spec:", "version spec hash marker");
}

// EXAMPLE: version_flag_alias -- --version identical to bare-word version.
TEST(test_version_flag_alias_matches) {
    auto bare = run({"version"});
    auto flag = run({"--version"});
    expect_eq_int(flag.code, 0, "--version exit");
    check(flag.out == bare.out,
          "--version output must equal bare-word version output");
}

// EXAMPLE: help_verb_bare_word -- usage to stdout, exit 0.
TEST(test_help_bare_word) {
    auto r = run({"help"});
    expect_eq_int(r.code, 0, "help exit");
    expect_contains(r.out, "usage:", "help usage on stdout");
}

// --help and -h are tolerated aliases for help (exit 0, usage to stdout).
TEST(test_help_flag_aliases) {
    auto a = run({"--help"});
    expect_eq_int(a.code, 0, "--help exit");
    expect_contains(a.out, "usage:", "--help usage on stdout");
    auto b = run({"-h"});
    expect_eq_int(b.code, 0, "-h exit");
    expect_contains(b.out, "usage:", "-h usage on stdout");
}

// EXAMPLE: bare_invocation_shows_help -- no verb -> usage to stdout, exit 0.
TEST(test_bare_invocation_usage_stdout_exit0) {
    auto r = run({});
    expect_eq_int(r.code, 0, "bare invocation exit");
    expect_contains(r.out, "usage:", "bare invocation usage on stdout");
}

// EXAMPLE: unknown_verb_rejected -- usage to stderr, exit 2.
TEST(test_unknown_verb_rejected) {
    auto r = run({"frobnicate"});
    expect_eq_int(r.code, 2, "unknown verb exit");
    expect_contains(r.err, "usage", "unknown verb usage on stderr");
}

// EXAMPLE: status_unknown_argument -- unknown option -> usage to stderr, exit 2.
TEST(test_status_unknown_argument) {
    auto r = run({"status", "--frobnicate"});
    expect_eq_int(r.code, 2, "status unknown arg exit");
    expect_contains(r.err, "usage", "status unknown arg usage on stderr");
}

// M0 acceptance gate: format=bad_value -> invocation error, exit 2.
// EXAMPLE: describe_unknown_format (format=toml) -- usage to stderr, exit 2.
TEST(test_describe_unknown_format_rejected) {
    auto r = run({"describe", "format=toml"});
    expect_eq_int(r.code, 2, "describe unknown format exit");
    expect_contains(r.err, "usage", "unknown format usage on stderr");
}

// M0 acceptance gate: a bad global format value is an invocation error (exit 2).
TEST(test_global_bad_format_value) {
    auto r = run({"format=bad_value"});
    expect_eq_int(r.code, 2, "format=bad_value exit");
}

// Unknown option value generally -> exit 2 (e.g. unknown mode).
TEST(test_unknown_option_value_mode) {
    auto r = run({"apply", "mode=sideways"});
    expect_eq_int(r.code, 2, "unknown mode value exit");
}

// Unknown option key -> exit 2.
TEST(test_unknown_option_key) {
    auto r = run({"status", "bogus-key=1"});
    expect_eq_int(r.code, 2, "unknown option key exit");
}
