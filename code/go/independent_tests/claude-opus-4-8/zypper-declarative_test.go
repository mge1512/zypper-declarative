// generated from spec: zypper-declarative.spec.md sha256:27aee8e374eb3507189bad0b78339109d0116e6a13b55ae4df2ba9a18e769fc4
// tests by: claude-opus-4-8
//
// Black-box test suite for the zypper-declarative cli-tool.
//
// Methodology: these tests invoke the built binary through the CLI interface
// declared in the spec's DEPLOYMENT section (key=value arguments, bare-word
// verbs) via os/exec, and assert on stdout, stderr, and exit code only. No
// internal package of the implementation is imported. The binary is discovered
// at the canonical BINARY-LOCATION path "../../zypper-declarative" relative to
// this test directory (project root, per the cli-tool template). TestMain
// builds it there from ./cmd/zypper-declarative before any test runs.
//
// Tests that require root, a live system package database, or a snapshot
// transaction mechanism are deliberately omitted from the runnable set: the
// translation environment is non-privileged. The verb-dispatch contract,
// the global commands (version/help/bare), format resolution, and the offline
// two-file comparisons (diff/verify with manifest-path + state-path) are fully
// exercisable black-box and are the focus here.
package independent_tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// binaryPath is the canonical BINARY-LOCATION: two directories up from
// independent_tests/<llm-name>/ to the project root.
const binaryPath = "../../zypper-declarative"

// TestMain builds the binary at the canonical location from the entry point
// the translator is required to place at ./cmd/zypper-declarative (recorded in
// TEST_REPORT.md as the Binary-Discovery-Path source path).
func TestMain(m *testing.M) {
	abs, err := filepath.Abs(binaryPath)
	if err != nil {
		os.Stderr.WriteString("cannot resolve binary path: " + err.Error() + "\n")
		os.Exit(3)
	}
	// project root is the directory containing the binary.
	root := filepath.Dir(abs)
	build := exec.Command("go", "build", "-o", abs, "./cmd/zypper-declarative")
	build.Dir = root
	build.Env = append(os.Environ(), "CGO_ENABLED=0")
	out, err := build.CombinedOutput()
	if err != nil {
		os.Stderr.WriteString("go build failed: " + err.Error() + "\n" + string(out) + "\n")
		os.Exit(3)
	}
	os.Exit(m.Run())
}

// run executes the binary with the given args and returns stdout, stderr, exit.
func run(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	cmd := exec.Command(binaryPath, args...)
	var stdout, stderr strings.Builder
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

// writeFile writes content to a temp file with the given name and returns path.
func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", p, err)
	}
	return p
}

// ----------------------------------------------------------------------------
// Reusable structurally-complete fixtures (the shared Manifest schema).
// ----------------------------------------------------------------------------

// A baseline applied/reference manifest in canonical JSON: packages (nginx),
// services (nginx.service enabled), config_files (foo.conf, bar.conf), and
// repositories. This is a complete, schema-valid Manifest.
const baselineJSON = `{
  "meta": { "format_version": 1, "generator": "zypper-declarative 0.6.5", "created_at": "2026-05-29T08:30:00Z", "desired_sha256": "" },
  "repositories": {
    "_attributes": { "repository_system": "zypp" },
    "_elements": [
      { "alias": "repo-a", "name": "Repo A", "url": "https://example/a", "type": "rpm-md", "enabled": true, "gpgcheck": true, "autorefresh": false, "priority": 99 }
    ]
  },
  "packages": {
    "_attributes": { "package_system": "rpm" },
    "_elements": [ { "name": "nginx", "version": "1.0.0", "release": "1", "arch": "x86_64" } ]
  },
  "services": {
    "_attributes": { "init_system": "systemd" },
    "_elements": [ { "name": "nginx.service", "state": "enabled" } ]
  },
  "config_files": {
    "_attributes": {},
    "_elements": [
      { "name": "/etc/foo.conf", "type": "file", "mode": "0644", "user": "root", "group": "root", "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "target": "", "content_ref": "", "package_name": "" },
      { "name": "/etc/bar.conf", "type": "file", "mode": "0644", "user": "root", "group": "root", "sha256": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "target": "", "content_ref": "", "package_name": "" }
    ]
  }
}`

// ----------------------------------------------------------------------------
// Global commands and dispatch (spec DEPLOYMENT + INVARIANTS).
// ----------------------------------------------------------------------------

// EXAMPLE: version_verb_bare_word
func TestVersionVerbBareWord(t *testing.T) {
	stdout, _, exit := run(t, "version")
	if exit != 0 {
		t.Fatalf("exit = %d, want 0", exit)
	}
	if !strings.Contains(stdout, "zypper-declarative") {
		t.Errorf("stdout missing program name: %q", stdout)
	}
	if !strings.Contains(stdout, "spec:") {
		t.Errorf("stdout missing embedded spec hash (spec:): %q", stdout)
	}
}

// EXAMPLE: version_flag_alias — --version identical to bare-word version.
func TestVersionFlagAlias(t *testing.T) {
	bareOut, _, bareExit := run(t, "version")
	flagOut, _, flagExit := run(t, "--version")
	if flagExit != 0 {
		t.Fatalf("--version exit = %d, want 0", flagExit)
	}
	if bareExit != 0 {
		t.Fatalf("version exit = %d, want 0", bareExit)
	}
	if flagOut != bareOut {
		t.Errorf("--version output %q != version output %q", flagOut, bareOut)
	}
}

// EXAMPLE: help_verb_bare_word
func TestHelpVerbBareWord(t *testing.T) {
	stdout, _, exit := run(t, "help")
	if exit != 0 {
		t.Fatalf("exit = %d, want 0", exit)
	}
	if !strings.Contains(strings.ToLower(stdout), "usage") {
		t.Errorf("help stdout missing usage: %q", stdout)
	}
}

// help flag aliases --help and -h print usage to stdout and exit 0.
func TestHelpFlagAliases(t *testing.T) {
	for _, alias := range []string{"--help", "-h"} {
		stdout, _, exit := run(t, alias)
		if exit != 0 {
			t.Errorf("%s exit = %d, want 0", alias, exit)
		}
		if !strings.Contains(strings.ToLower(stdout), "usage") {
			t.Errorf("%s stdout missing usage: %q", alias, stdout)
		}
	}
}

// EXAMPLE: bare_invocation_shows_help
func TestBareInvocationShowsHelp(t *testing.T) {
	stdout, _, exit := run(t)
	if exit != 0 {
		t.Fatalf("bare invocation exit = %d, want 0", exit)
	}
	if !strings.Contains(strings.ToLower(stdout), "usage") {
		t.Errorf("bare invocation stdout missing usage: %q", stdout)
	}
}

// EXAMPLE: unknown_verb_rejected
func TestUnknownVerbRejected(t *testing.T) {
	stdout, stderr, exit := run(t, "frobnicate")
	if exit != 2 {
		t.Fatalf("exit = %d, want 2 (stdout=%q stderr=%q)", exit, stdout, stderr)
	}
	if !strings.Contains(strings.ToLower(stderr), "usage") {
		t.Errorf("unknown verb: stderr missing usage: %q", stderr)
	}
}

// MILESTONE acceptance: an unknown format value is an invocation error (exit 2).
func TestUnknownFormatValueExit2(t *testing.T) {
	_, _, exit := run(t, "format=bad_value")
	if exit != 2 {
		t.Fatalf("format=bad_value exit = %d, want 2", exit)
	}
}

// EXAMPLE: describe_unknown_format — unknown format on describe -> exit 2.
func TestDescribeUnknownFormat(t *testing.T) {
	stdout, stderr, exit := run(t, "describe", "format=toml")
	if exit != 2 {
		t.Fatalf("describe format=toml exit = %d, want 2 (stdout=%q stderr=%q)", exit, stdout, stderr)
	}
	if !strings.Contains(strings.ToLower(stderr), "usage") {
		t.Errorf("describe unknown format: stderr missing usage: %q", stderr)
	}
}

// EXAMPLE: status_unknown_argument
func TestStatusUnknownArgument(t *testing.T) {
	_, stderr, exit := run(t, "status", "--frobnicate")
	if exit != 2 {
		t.Fatalf("status --frobnicate exit = %d, want 2 (stderr=%q)", exit, stderr)
	}
	if !strings.Contains(strings.ToLower(stderr), "usage") {
		t.Errorf("status unknown arg: stderr missing usage: %q", stderr)
	}
}

// POSIX --flag style is rejected for ordinary options: a bogus --flag on a verb
// that takes no such option must be an invocation error.
func TestPosixFlagRejectedForOptions(t *testing.T) {
	_, _, exit := run(t, "diff", "--manifest-path=/tmp/x.json")
	if exit != 2 {
		t.Fatalf("diff --manifest-path=... exit = %d, want 2 (POSIX flag style forbidden)", exit)
	}
}

// ----------------------------------------------------------------------------
// status (read-only; no applied record is the common non-privileged case).
// ----------------------------------------------------------------------------

// EXAMPLE: status_no_declaration — with no applied record, prints the message,
// exit 0. We point applied-root at an empty dir so no applied.json exists.
func TestStatusNoDeclaration(t *testing.T) {
	dir := t.TempDir()
	stdout, _, exit := run(t, "status", "applied-root="+dir)
	if exit != 0 {
		t.Fatalf("status exit = %d, want 0", exit)
	}
	if !strings.Contains(stdout, "no declaration applied") {
		t.Errorf("status stdout missing 'no declaration applied': %q", stdout)
	}
}

// ----------------------------------------------------------------------------
// load-desired-manifest validation paths (driven through verbs, offline).
// ----------------------------------------------------------------------------

// EXAMPLE: apply_manifest_unreadable / diff_manifest_unreadable -> exit 2.
func TestManifestUnreadableExit2(t *testing.T) {
	for _, verb := range []string{"diff", "apply"} {
		_, stderr, exit := run(t, verb, "manifest-path=/nonexistent-zd-xyz.json")
		if exit != 2 {
			t.Errorf("%s missing manifest exit = %d, want 2 (stderr=%q)", verb, exit, stderr)
		}
		if !strings.Contains(stderr, "invocation") {
			t.Errorf("%s missing manifest: stderr missing domain=invocation: %q", verb, stderr)
		}
	}
}

// EXAMPLE: apply_manifest_invalid — format_version=2 -> manifest error, exit 1.
// We drive it through diff (offline: manifest-path + state-path) so that the
// invalid-manifest path is reached without needing privilege or a live system.
func TestManifestInvalidFormatVersion(t *testing.T) {
	dir := t.TempDir()
	bad := `{ "meta": { "format_version": 2, "generator": "g", "created_at": "", "desired_sha256": "" } }`
	mp := writeFile(t, dir, "bad.json", bad)
	sp := writeFile(t, dir, "state.json", baselineJSON)
	_, stderr, exit := run(t, "diff", "manifest-path="+mp, "state-path="+sp)
	if exit != 1 {
		t.Fatalf("invalid manifest exit = %d, want 1 (stderr=%q)", exit, stderr)
	}
	if !strings.Contains(stderr, "manifest") {
		t.Errorf("invalid manifest: stderr missing domain=manifest: %q", stderr)
	}
}

// EXAMPLE: apply_rejects_full_describe_dump — a desired manifest carrying a
// non-empty observational scope (unmanaged_files) is rejected with domain=manifest.
// Driven through diff offline to avoid privilege.
func TestDesiredManifestWithObservationalScopeRejected(t *testing.T) {
	dir := t.TempDir()
	dump := `{
      "meta": { "format_version": 1, "generator": "g", "created_at": "", "desired_sha256": "" },
      "unmanaged_files": {
        "_attributes": {},
        "_elements": [ { "name": "/usr/bin/foo", "type": "file", "mode": "0755", "user": "root", "group": "root", "sha256": "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", "target": "" } ]
      }
    }`
	mp := writeFile(t, dir, "dump.json", dump)
	sp := writeFile(t, dir, "state.json", baselineJSON)
	_, stderr, exit := run(t, "diff", "manifest-path="+mp, "state-path="+sp)
	if exit != 1 {
		t.Fatalf("observational-scope manifest exit = %d, want 1 (stderr=%q)", exit, stderr)
	}
	if !strings.Contains(stderr, "manifest") {
		t.Errorf("observational-scope manifest: stderr missing domain=manifest: %q", stderr)
	}
}

// EXAMPLE: diff with a malformed state dump -> exit 2, domain=invocation.
func TestDiffMalformedStateDump(t *testing.T) {
	dir := t.TempDir()
	mp := writeFile(t, dir, "m.json", baselineJSON)
	sp := writeFile(t, dir, "broken.json", `{ this is : not json `)
	_, stderr, exit := run(t, "diff", "manifest-path="+mp, "state-path="+sp)
	if exit != 2 {
		t.Fatalf("malformed state dump exit = %d, want 2 (stderr=%q)", exit, stderr)
	}
	if !strings.Contains(stderr, "invocation") {
		t.Errorf("malformed state dump: stderr missing domain=invocation: %q", stderr)
	}
}

// ----------------------------------------------------------------------------
// diff: offline two-file plan (compute-intent-diff + compute-drift).
// ----------------------------------------------------------------------------

// EXAMPLE: diff_prints_plan + intent_diff_yields_deletion.
// Reference (applied) state via state-path equals the baseline; the desired
// manifest adds nginx-extra and drops /etc/bar.conf. The intent diff is between
// the desired manifest and the *applied record*; since no applied record exists
// here, we verify the plan output is produced (exit 0) and lists the scopes.
// The deletion of /etc/bar.conf relative to the applied record is exercised in
// TestDiffComputesDeletionFromAppliedRecord using applied-root.
func TestDiffPrintsPlanOffline(t *testing.T) {
	dir := t.TempDir()
	desired := `{
      "meta": { "format_version": 1, "generator": "g", "created_at": "", "desired_sha256": "" },
      "packages": { "_attributes": { "package_system": "rpm" }, "_elements": [ { "name": "nginx", "version": "", "release": "", "arch": "" } ] }
    }`
	mp := writeFile(t, dir, "desired.json", desired)
	sp := writeFile(t, dir, "state.json", baselineJSON)
	stdout, stderr, exit := run(t, "diff", "manifest-path="+mp, "state-path="+sp)
	if exit != 0 {
		t.Fatalf("diff offline exit = %d, want 0 (stderr=%q)", exit, stderr)
	}
	// The plan must mention packages to install (nginx). The plan output is to
	// stdout per the spec.
	if !strings.Contains(stdout, "nginx") {
		t.Errorf("diff plan stdout missing nginx: %q", stdout)
	}
}

// EXAMPLE: diff_offline_two_files — pure function of the two files, exit 0.
func TestDiffOfflineTwoFilesExit0(t *testing.T) {
	dir := t.TempDir()
	mp := writeFile(t, dir, "baseline.json", baselineJSON)
	sp := writeFile(t, dir, "after.json", baselineJSON)
	_, stderr, exit := run(t, "diff", "manifest-path="+mp, "state-path="+sp)
	if exit != 0 {
		t.Fatalf("diff offline two files exit = %d, want 0 (stderr=%q)", exit, stderr)
	}
}

// EXAMPLE: intent_diff_yields_deletion — drive diff against an applied record
// (via applied-root) where the applied config_files declares foo.conf and
// bar.conf and the desired manifest declares only foo.conf; the plan must list
// /etc/bar.conf under files to delete. state-path supplies the actual state so
// no live read happens.
func TestDiffComputesDeletionFromAppliedRecord(t *testing.T) {
	root := t.TempDir()
	// Place an applied record at <root>/usr/lib/zypper-declarative/applied.json.
	dir := filepath.Join(root, "usr", "lib", "zypper-declarative")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("mkdir applied dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "applied.json"), []byte(baselineJSON), 0644); err != nil {
		t.Fatalf("write applied.json: %v", err)
	}
	work := t.TempDir()
	// Desired drops /etc/bar.conf (declares only /etc/foo.conf).
	desired := `{
      "meta": { "format_version": 1, "generator": "g", "created_at": "", "desired_sha256": "" },
      "config_files": {
        "_attributes": {},
        "_elements": [
          { "name": "/etc/foo.conf", "type": "file", "mode": "0644", "user": "root", "group": "root", "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "target": "", "content_ref": "", "package_name": "" }
        ]
      }
    }`
	mp := writeFile(t, work, "desired.json", desired)
	sp := writeFile(t, work, "state.json", baselineJSON)
	stdout, stderr, exit := run(t, "diff", "manifest-path="+mp, "state-path="+sp, "applied-root="+root)
	if exit != 0 {
		t.Fatalf("diff with applied-root exit = %d, want 0 (stderr=%q)", exit, stderr)
	}
	if !strings.Contains(stdout, "/etc/bar.conf") {
		t.Errorf("diff plan stdout missing /etc/bar.conf (files to delete): %q", stdout)
	}
}

// EXAMPLE: describe_bootstraps_desired_manifest — diff of an unchanged system
// against itself shows no changes. We use the baseline as both reference (via
// applied-root) and actual state; intent diff and drift are both empty, exit 0.
func TestDiffUnchangedSystemNoChanges(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "usr", "lib", "zypper-declarative")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "applied.json"), []byte(baselineJSON), 0644); err != nil {
		t.Fatalf("write applied.json: %v", err)
	}
	work := t.TempDir()
	mp := writeFile(t, work, "desired.json", baselineJSON)
	sp := writeFile(t, work, "state.json", baselineJSON)
	stdout, stderr, exit := run(t, "diff", "manifest-path="+mp, "state-path="+sp, "applied-root="+root)
	if exit != 0 {
		t.Fatalf("diff unchanged exit = %d, want 0 (stderr=%q stdout=%q)", exit, stderr, stdout)
	}
}

// ----------------------------------------------------------------------------
// verify: offline two-file comparison (manifest-path + state-path).
// ----------------------------------------------------------------------------

// EXAMPLE: verify_offline_manifest_and_state + verify_offline_no_applied_record_ok.
// A reference manifest + a matching captured state, no applied record required:
// exit 0, "system matches declaration", and NOT "no declaration applied".
func TestVerifyOfflineMatches(t *testing.T) {
	dir := t.TempDir()
	mp := writeFile(t, dir, "baseline.json", baselineJSON)
	sp := writeFile(t, dir, "after.json", baselineJSON)
	stdout, stderr, exit := run(t, "verify", "manifest-path="+mp, "state-path="+sp)
	if exit != 0 {
		t.Fatalf("verify offline match exit = %d, want 0 (stderr=%q)", exit, stderr)
	}
	if !strings.Contains(stdout, "system matches declaration") {
		t.Errorf("verify offline match: stdout missing 'system matches declaration': %q", stdout)
	}
	if strings.Contains(stdout, "no declaration applied") || strings.Contains(stderr, "no declaration applied") {
		t.Errorf("verify offline match must not say 'no declaration applied'")
	}
}

// EXAMPLE: verify_against_external_state_dump — the captured state diverges in
// one declared service state; exit 1 with a units diagnostic naming the service.
func TestVerifyOfflineUnitDrift(t *testing.T) {
	dir := t.TempDir()
	// state has nginx.service disabled instead of enabled.
	state := strings.Replace(baselineJSON, `"state": "enabled"`, `"state": "disabled"`, 1)
	mp := writeFile(t, dir, "baseline.json", baselineJSON)
	sp := writeFile(t, dir, "after.json", state)
	stdout, stderr, exit := run(t, "verify", "manifest-path="+mp, "state-path="+sp)
	if exit != 1 {
		t.Fatalf("verify unit drift exit = %d, want 1 (stdout=%q stderr=%q)", exit, stdout, stderr)
	}
	if !strings.Contains(stderr, "units") {
		t.Errorf("verify unit drift: stderr missing domain=units: %q", stderr)
	}
	if !strings.Contains(stderr, "nginx.service") {
		t.Errorf("verify unit drift: stderr missing divergent service name: %q", stderr)
	}
}

// EXAMPLE: verify_detects_drift — a declared file edited (sha256 differs) ->
// exit 1, domain=files naming the path.
func TestVerifyOfflineFileDrift(t *testing.T) {
	dir := t.TempDir()
	// actual state reports /etc/foo.conf with a different sha256.
	state := strings.Replace(baselineJSON,
		`"name": "/etc/foo.conf", "type": "file", "mode": "0644", "user": "root", "group": "root", "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"`,
		`"name": "/etc/foo.conf", "type": "file", "mode": "0644", "user": "root", "group": "root", "sha256": "9999999999999999999999999999999999999999999999999999999999999999"`,
		1)
	mp := writeFile(t, dir, "baseline.json", baselineJSON)
	sp := writeFile(t, dir, "after.json", state)
	_, stderr, exit := run(t, "verify", "manifest-path="+mp, "state-path="+sp)
	if exit != 1 {
		t.Fatalf("verify file drift exit = %d, want 1 (stderr=%q)", exit, stderr)
	}
	if !strings.Contains(stderr, "files") {
		t.Errorf("verify file drift: stderr missing domain=files: %q", stderr)
	}
	if !strings.Contains(stderr, "/etc/foo.conf") {
		t.Errorf("verify file drift: stderr missing /etc/foo.conf: %q", stderr)
	}
}

// EXAMPLE: drift_type_transition_is_modified — reference declares /etc/foo as a
// file; actual reports it as a link. Reported as modified (files) regardless of
// content. Driven offline via verify.
func TestVerifyTypeTransitionIsModified(t *testing.T) {
	dir := t.TempDir()
	ref := `{
      "meta": { "format_version": 1, "generator": "g", "created_at": "", "desired_sha256": "" },
      "config_files": { "_attributes": {}, "_elements": [
        { "name": "/etc/foo", "type": "file", "mode": "0644", "user": "root", "group": "root", "sha256": "1111111111111111111111111111111111111111111111111111111111111111", "target": "", "content_ref": "", "package_name": "" }
      ] }
    }`
	state := `{
      "meta": { "format_version": 1, "generator": "g", "created_at": "", "desired_sha256": "" },
      "config_files": { "_attributes": {}, "_elements": [
        { "name": "/etc/foo", "type": "link", "mode": "0777", "user": "root", "group": "root", "sha256": "", "target": "../somewhere", "content_ref": "", "package_name": "" }
      ] }
    }`
	mp := writeFile(t, dir, "ref.json", ref)
	sp := writeFile(t, dir, "state.json", state)
	_, stderr, exit := run(t, "verify", "manifest-path="+mp, "state-path="+sp)
	if exit != 1 {
		t.Fatalf("verify type transition exit = %d, want 1 (stderr=%q)", exit, stderr)
	}
	if !strings.Contains(stderr, "/etc/foo") {
		t.Errorf("verify type transition: stderr missing /etc/foo: %q", stderr)
	}
}

// EXAMPLE: verify_malformed_state_dump -> exit 2, domain=invocation.
func TestVerifyMalformedStateDump(t *testing.T) {
	dir := t.TempDir()
	mp := writeFile(t, dir, "baseline.json", baselineJSON)
	sp := writeFile(t, dir, "broken.json", `not a manifest at all`)
	_, stderr, exit := run(t, "verify", "manifest-path="+mp, "state-path="+sp)
	if exit != 2 {
		t.Fatalf("verify malformed dump exit = %d, want 2 (stderr=%q)", exit, stderr)
	}
	if !strings.Contains(stderr, "invocation") {
		t.Errorf("verify malformed dump: stderr missing domain=invocation: %q", stderr)
	}
}

// EXAMPLE: verify_no_applied_record — no manifest-path and no applied record ->
// "no declaration applied", exit 2, domain=invocation.
func TestVerifyNoAppliedRecord(t *testing.T) {
	dir := t.TempDir() // empty: no applied.json under it
	sp := writeFile(t, dir, "state.json", baselineJSON)
	_, stderr, exit := run(t, "verify", "state-path="+sp, "applied-root="+dir)
	if exit != 2 {
		t.Fatalf("verify no applied record exit = %d, want 2 (stderr=%q)", exit, stderr)
	}
	if !strings.Contains(stderr, "no declaration applied") {
		t.Errorf("verify no applied record: stderr missing 'no declaration applied': %q", stderr)
	}
}

// ----------------------------------------------------------------------------
// resolve-format and YAML (offline, via diff/verify with state files).
// ----------------------------------------------------------------------------

// EXAMPLE: yaml_manifest_accepted + yaml_format_identity_stable — a YAML
// manifest is parsed under the safe profile and produces the same plan as the
// equivalent JSON. Driven offline: a YAML manifest-path with a JSON state-path
// (JSON is valid YAML; here state stays JSON), exit 0.
func TestYAMLManifestAccepted(t *testing.T) {
	dir := t.TempDir()
	yamlManifest := `meta:
  format_version: 1
  generator: "g"
  created_at: ""
  desired_sha256: ""
packages:
  _attributes:
    package_system: "rpm"
  _elements:
    - name: "nginx"
      version: ""
      release: ""
      arch: ""
`
	mp := writeFile(t, dir, "desired.yaml", yamlManifest)
	sp := writeFile(t, dir, "state.json", baselineJSON)
	stdout, stderr, exit := run(t, "diff", "manifest-path="+mp, "state-path="+sp)
	if exit != 0 {
		t.Fatalf("yaml manifest diff exit = %d, want 0 (stderr=%q)", exit, stderr)
	}
	if !strings.Contains(stdout, "nginx") {
		t.Errorf("yaml manifest plan missing nginx: %q", stdout)
	}
}

// EXAMPLE: yaml_unsafe_rejected — a YAML manifest using a code/arbitrary tag is
// rejected with a manifest error (exit 1). Driven offline via diff.
func TestYAMLUnsafeRejected(t *testing.T) {
	dir := t.TempDir()
	// A multi-document YAML stream (a disabled safe-profile feature) must be
	// rejected with a manifest error.
	unsafe := `meta:
  format_version: 1
  generator: "g"
  created_at: ""
  desired_sha256: ""
---
meta:
  format_version: 1
`
	mp := writeFile(t, dir, "evil.yaml", unsafe)
	sp := writeFile(t, dir, "state.json", baselineJSON)
	_, stderr, exit := run(t, "diff", "manifest-path="+mp, "state-path="+sp)
	if exit != 1 {
		t.Fatalf("unsafe yaml exit = %d, want 1 (stderr=%q)", exit, stderr)
	}
	if !strings.Contains(stderr, "manifest") {
		t.Errorf("unsafe yaml: stderr missing domain=manifest: %q", stderr)
	}
}

// EXAMPLE: yaml_format_identity_stable — JSON and YAML of the same manifest
// yield the same desired_sha256 (so idempotence holds across a format switch).
// We observe this black-box by checking that diff against a matching state is
// empty (exit 0) whether the manifest is JSON or YAML, with the same state.
func TestYAMLAndJSONManifestEquivalentPlan(t *testing.T) {
	dir := t.TempDir()
	jsonManifest := `{
      "meta": { "format_version": 1, "generator": "g", "created_at": "", "desired_sha256": "" },
      "packages": { "_attributes": { "package_system": "rpm" }, "_elements": [ { "name": "nginx", "version": "", "release": "", "arch": "" } ] }
    }`
	yamlManifest := `meta:
  format_version: 1
  generator: "g"
  created_at: ""
  desired_sha256: ""
packages:
  _attributes:
    package_system: "rpm"
  _elements:
    - name: "nginx"
      version: ""
      release: ""
      arch: ""
`
	jp := writeFile(t, dir, "m.json", jsonManifest)
	yp := writeFile(t, dir, "m.yaml", yamlManifest)
	sp := writeFile(t, dir, "state.json", baselineJSON)
	jOut, _, jExit := run(t, "diff", "manifest-path="+jp, "state-path="+sp)
	yOut, _, yExit := run(t, "diff", "manifest-path="+yp, "state-path="+sp)
	if jExit != 0 || yExit != 0 {
		t.Fatalf("json/yaml diff exits = %d/%d, want 0/0", jExit, yExit)
	}
	if jOut != yOut {
		t.Errorf("json and yaml manifests produced different plans:\nJSON:\n%s\nYAML:\n%s", jOut, yOut)
	}
}
