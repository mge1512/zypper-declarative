// generated from spec: zypper-declarative.spec.md sha256:b2d0de88fbed1163678e59e931c741b9d999b71f902f6eb01db8790bb813d057
// tests by: claude-opus-4-8
//
// CLI surface / global contract tests. These exercise the dispatcher-level
// behaviour declared in the spec DEPLOYMENT section and the [observable]
// INVARIANTs about version/help bare words and POSIX-flag aliases.
package zypperdeclarative_test

import (
	"strings"
	"testing"
)

// EXAMPLE: version_verb_bare_word
func TestVersionVerbBareWord(t *testing.T) {
	r := run(t, "version")
	assertExit(t, r, 0)
	assertStdoutContains(t, r, "zypper-declarative")
	// spec hash must be embedded in version output (DEPLOYMENT, INVARIANT).
	assertStdoutContains(t, r, "spec:")
}

// EXAMPLE: version_flag_alias -- --version is a tolerated alias for the
// bare-word version verb and produces identical stdout.
func TestVersionFlagAlias(t *testing.T) {
	bare := run(t, "version")
	flag := run(t, "--version")
	assertExit(t, flag, 0)
	if flag.stdout != bare.stdout {
		t.Errorf("--version stdout differs from bare version verb\n--version:\n%s\nversion:\n%s",
			flag.stdout, bare.stdout)
	}
}

// INVARIANT: version output begins with the program name line.
func TestVersionStartsWithProgramName(t *testing.T) {
	r := run(t, "version")
	assertExit(t, r, 0)
	if !strings.HasPrefix(strings.TrimLeft(r.stdout, " \t"), "zypper-declarative ") {
		t.Errorf("version stdout does not start with %q\nstdout:\n%s",
			"zypper-declarative ", r.stdout)
	}
}

// EXAMPLE: help_verb_bare_word
func TestHelpVerbBareWord(t *testing.T) {
	r := run(t, "help")
	assertExit(t, r, 0)
	assertStdoutContains(t, r, "usage:")
}

// --help and -h are tolerated aliases for help; usage to stdout, exit 0.
func TestHelpFlagAliases(t *testing.T) {
	for _, alias := range []string{"--help", "-h"} {
		r := run(t, alias)
		assertExit(t, r, 0)
		assertStdoutContains(t, r, "usage:")
	}
}

// EXAMPLE: bare_invocation_shows_help -- no verb prints usage to stdout and
// exits 0 (discovery, never an error, never converges).
func TestBareInvocationShowsHelp(t *testing.T) {
	r := run(t)
	assertExit(t, r, 0)
	assertStdoutContains(t, r, "usage:")
}

// EXAMPLE: unknown_verb_rejected -- usage to stderr, exit 2.
func TestUnknownVerbRejected(t *testing.T) {
	r := run(t, "frobnicate")
	assertExit(t, r, 2)
	assertStderrContains(t, r, "usage:")
}

// Acceptance criterion / EXAMPLE describe_unknown_format applied at the
// dispatcher: an unknown format value is an invocation error, exit 2.
func TestBadFormatValueExitsTwo(t *testing.T) {
	r := run(t, "format=bad_value")
	assertExit(t, r, 2)
}

// EXAMPLE: describe_unknown_format -- unknown format on describe.
func TestDescribeUnknownFormatRejected(t *testing.T) {
	r := run(t, "describe", "format=toml")
	assertExit(t, r, 2)
	assertStderrContains(t, r, "usage:")
}

// An unknown key=value option is an invocation error (exit 2).
func TestUnknownOptionRejected(t *testing.T) {
	r := run(t, "status", "nonsense-option=foo")
	assertExit(t, r, 2)
}

// POSIX --flag style (other than the tolerated version/help aliases) is not a
// valid option; status with an unknown --flag is an invocation error.
// EXAMPLE: status_unknown_argument
func TestStatusUnknownArgument(t *testing.T) {
	r := run(t, "status", "--frobnicate")
	assertExit(t, r, 2)
	assertStderrContains(t, r, "usage:")
}
