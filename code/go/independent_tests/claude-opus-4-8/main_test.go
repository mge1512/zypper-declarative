// generated from spec: zypper-declarative.spec.md sha256:f2cc80627e483a48bb8411d297711bc5f6c6e74c28dbf0dafc8fe7bd8817251e
// tests by: claude-opus-4-8
//
// Black-box test suite for the zypper-declarative CLI binary.
//
// These tests invoke the built binary through the DEPLOYMENT interface
// (a CLI binary invoked with bare-word verbs and key=value options) using
// exec.Command and assert on stdout, stderr, and exit code only. They do
// NOT import or call any internal package of the implementation.
//
// Per the cli-tool template BINARY-LOCATION constraint, the binary lives at
// the project root, which is "../../zypper-declarative" relative to this
// directory. TestMain builds it from ./cmd/zypper-declarative/main.go.
package independent_tests

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// binaryPath is the canonical path to the binary under test, relative to
// this test directory, per the cli-tool template's BINARY-LOCATION:
// project-root constraint (../../<binary-name>).
const binaryPath = "../../zypper-declarative"

// TestMain builds the binary at the canonical project-root location before
// running the suite. The source entry point is expected at
// ../../cmd/zypper-declarative/main.go (the translator must honour this path).
func TestMain(m *testing.M) {
	abs, err := filepath.Abs(binaryPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot resolve binary path: %v\n", err)
		os.Exit(3)
	}
	projectRoot := filepath.Dir(abs)
	build := exec.Command("go", "build", "-o", abs, "./cmd/zypper-declarative")
	build.Dir = projectRoot
	build.Env = append(os.Environ(), "CGO_ENABLED=0")
	out, err := build.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to build binary under test:\n%s\n", out)
		os.Exit(3)
	}
	os.Exit(m.Run())
}

// run executes the binary with the given args, returning stdout, stderr, exit.
func run(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	cmd := exec.Command(binaryPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	exit := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			exit = ee.ExitCode()
		} else {
			t.Fatalf("failed to run binary: %v", err)
		}
	}
	return stdout.String(), stderr.String(), exit
}

// writeTemp writes content to a temp file with the given suffix and returns path.
func writeTemp(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp %s: %v", p, err)
	}
	return p
}

// ---- shared fixtures (structurally complete manifests in the shared schema) ----

// A minimal valid desired manifest (JSON), declaring one package and one repo.
const desiredJSON = `{
  "meta": { "format_version": 1, "generator": "test", "created_at": "2026-05-29T08:30:00Z", "desired_sha256": "" },
  "repositories": {
    "_attributes": { "repository_system": "zypp" },
    "_elements": [
      { "alias": "pinned", "name": "Pinned", "url": "https://example/repo", "type": "rpm-md", "enabled": true, "gpgcheck": true, "autorefresh": false, "priority": 99 }
    ]
  },
  "packages": {
    "_attributes": { "package_system": "rpm" },
    "_elements": [ { "name": "nginx", "version": "", "release": "", "arch": "" } ]
  },
  "services": {
    "_attributes": { "init_system": "systemd" },
    "_elements": [ { "name": "nginx.service", "state": "enabled" } ]
  },
  "config_files": {
    "_attributes": null,
    "_elements": [
      { "name": "/etc/foo.conf", "type": "file", "mode": "0644", "user": "root", "group": "root",
        "sha256": "0000000000000000000000000000000000000000000000000000000000000000",
        "target": "", "content_ref": "files/etc/foo.conf", "package_name": "" }
    ]
  }
}`

// ----------------------------------------------------------------------------
// CLI surface: version, help, bare invocation, unknown verb/option/value
// ----------------------------------------------------------------------------

// EXAMPLE: version_verb_bare_word
func TestVersionVerbBareWord(t *testing.T) {
	stdout, _, exit := run(t, "version")
	if exit != 0 {
		t.Fatalf("version: exit=%d, want 0", exit)
	}
	if !strings.Contains(stdout, "zypper-declarative") {
		t.Errorf("version stdout missing program name: %q", stdout)
	}
	if !strings.Contains(stdout, "spec:") {
		t.Errorf("version stdout missing embedded spec hash (spec:): %q", stdout)
	}
}

// EXAMPLE: version_flag_alias — output identical to bare-word version verb.
func TestVersionFlagAlias(t *testing.T) {
	bare, _, e1 := run(t, "version")
	flag, _, e2 := run(t, "--version")
	if e1 != 0 || e2 != 0 {
		t.Fatalf("version exits: bare=%d flag=%d, want 0,0", e1, e2)
	}
	if bare != flag {
		t.Errorf("--version output differs from bare version verb:\nbare=%q\nflag=%q", bare, flag)
	}
}

// EXAMPLE: help_verb_bare_word
func TestHelpVerbBareWord(t *testing.T) {
	stdout, _, exit := run(t, "help")
	if exit != 0 {
		t.Fatalf("help: exit=%d, want 0", exit)
	}
	if !strings.Contains(strings.ToLower(stdout), "usage") {
		t.Errorf("help stdout missing usage: %q", stdout)
	}
}

// INVARIANT: --help and -h are tolerated aliases for help, exit 0 to stdout.
func TestHelpFlagAliases(t *testing.T) {
	for _, a := range []string{"--help", "-h"} {
		stdout, _, exit := run(t, a)
		if exit != 0 {
			t.Errorf("%s: exit=%d, want 0", a, exit)
		}
		if !strings.Contains(strings.ToLower(stdout), "usage") {
			t.Errorf("%s: stdout missing usage: %q", a, stdout)
		}
	}
}

// EXAMPLE: bare_invocation_shows_help — usage to stdout, exit 0.
func TestBareInvocationShowsHelp(t *testing.T) {
	stdout, _, exit := run(t)
	if exit != 0 {
		t.Fatalf("bare invocation: exit=%d, want 0", exit)
	}
	if !strings.Contains(strings.ToLower(stdout), "usage") {
		t.Errorf("bare invocation stdout missing usage: %q", stdout)
	}
}

// EXAMPLE: unknown_verb_rejected — usage to stderr, exit 2.
func TestUnknownVerbRejected(t *testing.T) {
	stdout, stderr, exit := run(t, "frobnicate")
	if exit != 2 {
		t.Fatalf("unknown verb: exit=%d, want 2", exit)
	}
	if !strings.Contains(strings.ToLower(stderr), "usage") {
		t.Errorf("unknown verb: stderr missing usage: %q", stderr)
	}
	_ = stdout
}

// EXAMPLE: describe_unknown_format — unknown format value -> usage to stderr, exit 2.
func TestDescribeUnknownFormat(t *testing.T) {
	_, stderr, exit := run(t, "describe", "format=toml")
	if exit != 2 {
		t.Fatalf("describe format=toml: exit=%d, want 2", exit)
	}
	if !strings.Contains(strings.ToLower(stderr), "usage") {
		t.Errorf("describe format=toml: stderr missing usage: %q", stderr)
	}
}

// Acceptance gate (M0): a bad format value is an invocation error (exit 2).
func TestBadFormatValueExits2(t *testing.T) {
	_, _, exit := run(t, "format=bad_value")
	if exit != 2 {
		t.Fatalf("format=bad_value: exit=%d, want 2", exit)
	}
}

// EXAMPLE: status_unknown_argument — unrecognised argument -> usage to stderr, exit 2.
func TestStatusUnknownArgument(t *testing.T) {
	_, stderr, exit := run(t, "status", "--frobnicate")
	if exit != 2 {
		t.Fatalf("status --frobnicate: exit=%d, want 2", exit)
	}
	if !strings.Contains(strings.ToLower(stderr), "usage") {
		t.Errorf("status --frobnicate: stderr missing usage: %q", stderr)
	}
}

// ----------------------------------------------------------------------------
// status (read-only, no declaration applied case is deterministic)
// ----------------------------------------------------------------------------

// EXAMPLE: status_no_declaration — no applied record -> "no declaration applied", exit 0.
func TestStatusNoDeclaration(t *testing.T) {
	dir := t.TempDir() // empty root: no applied.json under it
	stdout, _, exit := run(t, "status", "applied-root="+dir)
	if exit != 0 {
		t.Fatalf("status no decl: exit=%d, want 0", exit)
	}
	if !strings.Contains(stdout, "no declaration applied") {
		t.Errorf("status no decl: stdout=%q, want 'no declaration applied'", stdout)
	}
}

// ----------------------------------------------------------------------------
// manifest read errors (apply, diff) — deterministic, no privilege needed
// ----------------------------------------------------------------------------

// EXAMPLE: apply_manifest_unreadable — nonexistent manifest -> domain=invocation, exit 2.
func TestApplyManifestUnreadable(t *testing.T) {
	_, stderr, exit := run(t, "apply", "manifest-path=/nonexistent-zd-manifest.json")
	if exit != 2 {
		t.Fatalf("apply unreadable: exit=%d, want 2", exit)
	}
	if !strings.Contains(stderr, "invocation") {
		t.Errorf("apply unreadable: stderr missing domain=invocation: %q", stderr)
	}
}

// EXAMPLE: diff_manifest_unreadable — nonexistent manifest -> domain=invocation, exit 2.
func TestDiffManifestUnreadable(t *testing.T) {
	_, stderr, exit := run(t, "diff", "manifest-path=/nonexistent-zd-manifest.json")
	if exit != 2 {
		t.Fatalf("diff unreadable: exit=%d, want 2", exit)
	}
	if !strings.Contains(stderr, "invocation") {
		t.Errorf("diff unreadable: stderr missing domain=invocation: %q", stderr)
	}
}

// EXAMPLE: apply_manifest_invalid — format_version=2 -> domain=manifest, exit 1, no txn.
func TestApplyManifestInvalid(t *testing.T) {
	bad := strings.Replace(desiredJSON, `"format_version": 1`, `"format_version": 2`, 1)
	p := writeTemp(t, "bad.json", bad)
	_, stderr, exit := run(t, "apply", "manifest-path="+p, "mode=internal")
	if exit != 1 {
		t.Fatalf("apply invalid manifest: exit=%d, want 1\nstderr=%s", exit, stderr)
	}
	if !strings.Contains(stderr, "manifest") {
		t.Errorf("apply invalid manifest: stderr missing domain=manifest: %q", stderr)
	}
}

// EXAMPLE: verify_malformed_state_dump — malformed state -> domain=invocation, exit 2.
func TestVerifyMalformedStateDump(t *testing.T) {
	broken := writeTemp(t, "broken.json", "{ this is not valid json")
	ref := writeTemp(t, "ref.json", desiredJSON)
	_, stderr, exit := run(t, "verify", "manifest-path="+ref, "state-path="+broken)
	if exit != 2 {
		t.Fatalf("verify malformed dump: exit=%d, want 2\nstderr=%s", exit, stderr)
	}
	if !strings.Contains(stderr, "invocation") {
		t.Errorf("verify malformed dump: stderr missing domain=invocation: %q", stderr)
	}
}

// EXAMPLE: verify_no_applied_record — no manifest_path and no applied record -> exit 2.
func TestVerifyNoAppliedRecord(t *testing.T) {
	dir := t.TempDir()
	_, stderr, exit := run(t, "verify", "applied-root="+dir)
	if exit != 2 {
		t.Fatalf("verify no applied record: exit=%d, want 2\nstderr=%s", exit, stderr)
	}
	if !strings.Contains(stderr, "no declaration applied") {
		t.Errorf("verify no applied record: stderr=%q, want 'no declaration applied'", stderr)
	}
	if !strings.Contains(stderr, "invocation") {
		t.Errorf("verify no applied record: stderr missing domain=invocation: %q", stderr)
	}
}
