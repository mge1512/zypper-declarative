// generated from spec: zypper-declarative.spec.md sha256:f8ff76ecbc4bbc69a49e2e32b2924da3a64df1ad46196e05ce8c137b684429b2
// tests by: claude-opus-4-8
//
// Tests for the top-level CLI contract: bare invocation, version, help, and
// invocation-error paths. These are the globally-dispatched commands and the
// argument-validation surface. None require privilege or live system state.

package independent_test

import (
	"strings"
	"testing"
)

// EXAMPLE: bare_invocation_shows_help
// Bare invocation (no verb) prints usage to stdout and exits 0.
func TestBareInvocationShowsHelp(t *testing.T) {
	r := run(t)
	if r.exitCode != 0 {
		t.Fatalf("bare invocation: exit = %d, want 0; stderr=%q", r.exitCode, r.stderr)
	}
	if !strings.Contains(strings.ToLower(r.stdout), "usage") {
		t.Errorf("bare invocation: stdout %q does not contain usage", r.stdout)
	}
}

// EXAMPLE: version_verb_bare_word
// `version` prints program name, version, and the embedded spec hash to stdout, exit 0.
func TestVersionVerbBareWord(t *testing.T) {
	r := run(t, "version")
	if r.exitCode != 0 {
		t.Fatalf("version: exit = %d, want 0; stderr=%q", r.exitCode, r.stderr)
	}
	if !strings.HasPrefix(r.stdout, binaryName+" ") {
		t.Errorf("version: stdout %q does not start with program name %q", r.stdout, binaryName+" ")
	}
	if !strings.Contains(r.stdout, "spec:") {
		t.Errorf("version: stdout %q does not contain the embedded spec hash marker 'spec:'", r.stdout)
	}
}

// EXAMPLE: version_verb embeds the actual spec sha256.
func TestVersionEmbedsSpecHash(t *testing.T) {
	const specHash = "f8ff76ecbc4bbc69a49e2e32b2924da3a64df1ad46196e05ce8c137b684429b2"
	r := run(t, "version")
	if r.exitCode != 0 {
		t.Fatalf("version: exit = %d, want 0", r.exitCode)
	}
	if !strings.Contains(r.stdout, specHash) {
		t.Errorf("version: stdout %q does not embed spec sha256 %q", r.stdout, specHash)
	}
}

// EXAMPLE: version_flag_alias
// `--version` produces output identical to the bare-word version verb, exit 0.
func TestVersionFlagAlias(t *testing.T) {
	bare := run(t, "version")
	flag := run(t, "--version")
	if flag.exitCode != 0 {
		t.Fatalf("--version: exit = %d, want 0; stderr=%q", flag.exitCode, flag.stderr)
	}
	if flag.stdout != bare.stdout {
		t.Errorf("--version stdout %q != version verb stdout %q", flag.stdout, bare.stdout)
	}
}

// EXAMPLE: help_verb_bare_word
// `help` prints usage to stdout and exits 0.
func TestHelpVerbBareWord(t *testing.T) {
	r := run(t, "help")
	if r.exitCode != 0 {
		t.Fatalf("help: exit = %d, want 0; stderr=%q", r.exitCode, r.stderr)
	}
	if !strings.Contains(strings.ToLower(r.stdout), "usage") {
		t.Errorf("help: stdout %q does not contain usage", r.stdout)
	}
}

// INVARIANT: --help and -h are tolerated aliases for help; usage to stdout, exit 0.
func TestHelpFlagAliases(t *testing.T) {
	for _, arg := range []string{"--help", "-h"} {
		r := run(t, arg)
		if r.exitCode != 0 {
			t.Errorf("%s: exit = %d, want 0; stderr=%q", arg, r.exitCode, r.stderr)
		}
		if !strings.Contains(strings.ToLower(r.stdout), "usage") {
			t.Errorf("%s: stdout %q does not contain usage", arg, r.stdout)
		}
	}
}

// EXAMPLE: unknown_verb_rejected
// Unknown verb prints usage to stderr and exits 2.
func TestUnknownVerbRejected(t *testing.T) {
	r := run(t, "frobnicate")
	if r.exitCode != 2 {
		t.Fatalf("unknown verb: exit = %d, want 2; stdout=%q stderr=%q", r.exitCode, r.stdout, r.stderr)
	}
	if !strings.Contains(strings.ToLower(r.stderr), "usage") {
		t.Errorf("unknown verb: stderr %q does not contain usage", r.stderr)
	}
}

// EXAMPLE: describe_unknown_format (acceptance-criterion style: a bad option value
// is an invocation error, exit 2). The spec M0 acceptance criterion exercises
// `format=bad_value` exiting 2.
func TestUnknownFormatValueRejected(t *testing.T) {
	r := run(t, "describe", "format=toml")
	if r.exitCode != 2 {
		t.Fatalf("describe format=toml: exit = %d, want 2; stdout=%q stderr=%q", r.exitCode, r.stdout, r.stderr)
	}
	if !strings.Contains(strings.ToLower(r.stderr), "usage") &&
		!strings.Contains(strings.ToLower(r.stderr), "format") {
		t.Errorf("describe format=toml: stderr %q does not indicate the invocation error", r.stderr)
	}
}

// M0 acceptance criterion: a bad global format value is an invocation error.
func TestBadFormatValueExitTwo(t *testing.T) {
	r := run(t, "format=bad_value")
	if r.exitCode != 2 {
		t.Fatalf("format=bad_value: exit = %d, want 2; stdout=%q stderr=%q", r.exitCode, r.stdout, r.stderr)
	}
}

// EXAMPLE: status_unknown_argument
// Status rejects an unrecognised argument with usage to stderr and exit 2.
func TestStatusUnknownArgument(t *testing.T) {
	r := run(t, "status", "--frobnicate")
	if r.exitCode != 2 {
		t.Fatalf("status --frobnicate: exit = %d, want 2; stdout=%q stderr=%q", r.exitCode, r.stdout, r.stderr)
	}
	if !strings.Contains(strings.ToLower(r.stderr), "usage") {
		t.Errorf("status --frobnicate: stderr %q does not contain usage", r.stderr)
	}
}

// INVARIANT: no option uses POSIX --flag style except the tolerated version/help
// aliases. An unknown POSIX-style flag on a verb is an invocation error.
func TestUnknownPosixFlagRejected(t *testing.T) {
	r := run(t, "diff", "--strict")
	if r.exitCode != 2 {
		t.Fatalf("diff --strict: exit = %d, want 2; stdout=%q stderr=%q", r.exitCode, r.stdout, r.stderr)
	}
}
