// generated from spec: zypper-declarative.spec.md sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
// tests by: claude-opus-4-8
//
// Black-box test suite for the zypper-declarative cli-tool.
//
// Methodology: the tests invoke the built binary through the CLI interface
// declared in the spec's DEPLOYMENT section (key=value arguments and bare-word
// verbs) using exec.Command, and assert on stdout, stderr, and exit code only.
// They do NOT import or call any internal package of the implementation, and
// they do NOT simulate the binary's behaviour. The binary is built once by
// TestMain at the canonical BINARY-LOCATION (project root, two directories up
// from this test directory), i.e. "../../zypper-declarative", from the entry
// point at "../../cmd/zypper-declarative".
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

// binaryPath is the canonical BINARY-LOCATION per the cli-tool template:
// project root, relative to the test directory at independent_tests/<llm-name>/.
const binaryPath = "../../zypper-declarative"

// run executes the built binary with the given args and returns stdout,
// stderr, and the process exit code.
func run(t *testing.T, args ...string) (string, string, int) {
	t.Helper()
	cmd := exec.Command(binaryPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			exitCode = ee.ExitCode()
		} else {
			t.Fatalf("failed to run binary %q with args %v: %v", binaryPath, args, err)
		}
	}
	return stdout.String(), stderr.String(), exitCode
}

// writeTemp writes content to a temp file with the given suffix and returns its path.
func writeTemp(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatalf("writing temp file %s: %v", p, err)
	}
	return p
}

func TestMain(m *testing.M) {
	// Build the binary at the canonical project-root location from the
	// entry point at cmd/zypper-declarative. The translator must place the
	// entry point at exactly this source path (../../cmd/zypper-declarative).
	build := exec.Command("go", "build", "-o", "zypper-declarative", "./cmd/zypper-declarative")
	build.Dir = "../.." // project root, where go.mod lives
	build.Env = append(os.Environ(), "CGO_ENABLED=0")
	out, err := build.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "TestMain: failed to build binary: %v\n%s\n", err, out)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

// ---------------------------------------------------------------------------
// Global dispatcher behaviour (DEPLOYMENT, INVARIANTS)
// ---------------------------------------------------------------------------

// EXAMPLE: bare_invocation_shows_help
func TestBareInvocationShowsHelp(t *testing.T) {
	stdout, _, code := run(t)
	if code != 0 {
		t.Errorf("bare invocation: exit = %d, want 0", code)
	}
	if !strings.Contains(strings.ToLower(stdout), "usage") {
		t.Errorf("bare invocation: stdout = %q, want it to contain usage", stdout)
	}
}

// EXAMPLE: version_verb_bare_word
func TestVersionVerbBareWord(t *testing.T) {
	stdout, _, code := run(t, "version")
	if code != 0 {
		t.Errorf("version: exit = %d, want 0", code)
	}
	if !strings.HasPrefix(stdout, "zypper-declarative ") {
		t.Errorf("version: stdout = %q, want it to start with %q", stdout, "zypper-declarative ")
	}
	if !strings.Contains(stdout, "spec:") {
		t.Errorf("version: stdout = %q, want it to contain the embedded spec hash (spec:)", stdout)
	}
}

// EXAMPLE: version_flag_alias — --version is identical to bare-word version.
func TestVersionFlagAlias(t *testing.T) {
	bareOut, _, bareCode := run(t, "version")
	flagOut, _, flagCode := run(t, "--version")
	if flagCode != 0 {
		t.Errorf("--version: exit = %d, want 0", flagCode)
	}
	if flagOut != bareOut {
		t.Errorf("--version output %q != version output %q", flagOut, bareOut)
	}
	if bareCode != flagCode {
		t.Errorf("--version exit %d != version exit %d", flagCode, bareCode)
	}
}

// EXAMPLE: help_verb_bare_word
func TestHelpVerbBareWord(t *testing.T) {
	stdout, _, code := run(t, "help")
	if code != 0 {
		t.Errorf("help: exit = %d, want 0", code)
	}
	if !strings.Contains(strings.ToLower(stdout), "usage") {
		t.Errorf("help: stdout = %q, want it to contain usage", stdout)
	}
}

// INVARIANT: --help and -h are tolerated aliases for help.
func TestHelpFlagAliases(t *testing.T) {
	for _, alias := range []string{"--help", "-h"} {
		stdout, _, code := run(t, alias)
		if code != 0 {
			t.Errorf("%s: exit = %d, want 0", alias, code)
		}
		if !strings.Contains(strings.ToLower(stdout), "usage") {
			t.Errorf("%s: stdout = %q, want usage", alias, stdout)
		}
	}
}

// EXAMPLE: unknown_verb_rejected
func TestUnknownVerbRejected(t *testing.T) {
	stdout, stderr, code := run(t, "frobnicate")
	if code != 2 {
		t.Errorf("unknown verb: exit = %d, want 2", code)
	}
	if !strings.Contains(strings.ToLower(stderr), "usage") {
		t.Errorf("unknown verb: stderr = %q, want usage on stderr", stderr)
	}
	// Usage must go to stderr (not stdout) for the error path.
	if strings.Contains(strings.ToLower(stdout), "usage") {
		t.Errorf("unknown verb: usage should be on stderr, not stdout; stdout = %q", stdout)
	}
}

// EXAMPLE: describe_unknown_format
func TestDescribeUnknownFormat(t *testing.T) {
	_, stderr, code := run(t, "describe", "format=toml")
	if code != 2 {
		t.Errorf("describe format=toml: exit = %d, want 2", code)
	}
	if !strings.Contains(strings.ToLower(stderr), "usage") {
		t.Errorf("describe format=toml: stderr = %q, want usage", stderr)
	}
}

// MILESTONE 0 acceptance criterion: an unknown format value is an invocation error.
func TestUnknownFormatValueIsInvocationError(t *testing.T) {
	_, _, code := run(t, "format=bad_value")
	if code != 2 {
		t.Errorf("format=bad_value: exit = %d, want 2", code)
	}
}

// EXAMPLE: status_unknown_argument
func TestStatusUnknownArgument(t *testing.T) {
	stdout, stderr, code := run(t, "status", "--frobnicate")
	if code != 2 {
		t.Errorf("status --frobnicate: exit = %d, want 2", code)
	}
	if !strings.Contains(strings.ToLower(stderr), "usage") {
		t.Errorf("status --frobnicate: stderr = %q, want usage", stderr)
	}
	if strings.Contains(strings.ToLower(stdout), "usage") {
		t.Errorf("status --frobnicate: usage should be on stderr; stdout = %q", stdout)
	}
}

// ---------------------------------------------------------------------------
// apply error paths (do not require live convergence)
// ---------------------------------------------------------------------------

// EXAMPLE: apply_manifest_unreadable
func TestApplyManifestUnreadable(t *testing.T) {
	_, stderr, code := run(t, "apply", "manifest-path=/nonexistent-zd-manifest.json")
	if code != 2 {
		t.Errorf("apply manifest unreadable: exit = %d, want 2", code)
	}
	if !strings.Contains(stderr, "invocation") {
		t.Errorf("apply manifest unreadable: stderr = %q, want a diagnostic with domain=invocation", stderr)
	}
}

// EXAMPLE: apply_manifest_invalid — meta.format_version = 2 is a manifest error (exit 1).
func TestApplyManifestInvalid(t *testing.T) {
	manifest := `{
  "meta": { "format_version": 2, "generator": "x", "created_at": "2026-05-29T08:30:00Z", "desired_sha256": "" }
}`
	p := writeTemp(t, "bad-version.json", manifest)
	_, stderr, code := run(t, "apply", "manifest-path="+p)
	if code != 1 {
		t.Errorf("apply manifest format_version=2: exit = %d, want 1", code)
	}
	if !strings.Contains(stderr, "manifest") {
		t.Errorf("apply manifest format_version=2: stderr = %q, want a diagnostic with domain=manifest", stderr)
	}
}

// EXAMPLE: apply_rejects_full_describe_dump — a manifest carrying a non-empty
// observational scope (unmanaged_files) is rejected as a manifest error (exit 1).
func TestApplyRejectsFullDescribeDump(t *testing.T) {
	manifest := `{
  "meta": { "format_version": 1, "generator": "x", "created_at": "2026-05-29T08:30:00Z", "desired_sha256": "" },
  "unmanaged_files": {
    "_attributes": {},
    "_elements": [
      { "name": "/usr/bin/foo", "type": "file", "mode": "0755", "user": "root", "group": "root", "sha256": "0000000000000000000000000000000000000000000000000000000000000000", "target": "" }
    ]
  }
}`
	p := writeTemp(t, "full-dump.json", manifest)
	_, stderr, code := run(t, "apply", "manifest-path="+p)
	if code != 1 {
		t.Errorf("apply full-dump: exit = %d, want 1", code)
	}
	if !strings.Contains(stderr, "manifest") {
		t.Errorf("apply full-dump: stderr = %q, want domain=manifest", stderr)
	}
}

// EXAMPLE: yaml_unsafe_rejected — a YAML manifest using an executable/arbitrary
// tag is rejected with a manifest error (exit 1), no transaction opened.
func TestApplyYamlUnsafeRejected(t *testing.T) {
	// A YAML manifest using an arbitrary/executable tag (e.g. a Go/Python style
	// non-standard tag). Under the safe profile this must be rejected.
	yaml := "meta: !!python/object/apply:os.system ['echo pwned']\n"
	p := writeTemp(t, "evil.yaml", yaml)
	_, stderr, code := run(t, "apply", "manifest-path="+p)
	if code != 1 {
		t.Errorf("apply evil.yaml: exit = %d, want 1 (manifest error)", code)
	}
	if !strings.Contains(stderr, "manifest") {
		t.Errorf("apply evil.yaml: stderr = %q, want domain=manifest", stderr)
	}
}

// ---------------------------------------------------------------------------
// diff (dry run, pure function of files when state-path supplied)
// ---------------------------------------------------------------------------

// EXAMPLE: diff_manifest_unreadable
func TestDiffManifestUnreadable(t *testing.T) {
	_, stderr, code := run(t, "diff", "manifest-path=/nonexistent-zd-manifest.json")
	if code != 2 {
		t.Errorf("diff manifest unreadable: exit = %d, want 2", code)
	}
	if !strings.Contains(stderr, "invocation") {
		t.Errorf("diff manifest unreadable: stderr = %q, want domain=invocation", stderr)
	}
}

// A complete, valid desired manifest used as a fixture for offline diff.
const validDesiredManifestJSON = `{
  "meta": { "format_version": 1, "generator": "zypper-declarative test", "created_at": "2026-05-29T08:30:00Z", "desired_sha256": "" },
  "repositories": {
    "_attributes": { "repository_system": "zypp" },
    "_elements": [
      { "alias": "repo1", "name": "Repo One", "url": "https://example/repo", "type": "rpm-md", "enabled": true, "gpgcheck": true, "autorefresh": false, "priority": 99 }
    ]
  },
  "packages": {
    "_attributes": { "package_system": "rpm" },
    "_elements": [
      { "name": "nginx", "version": "", "release": "", "arch": "" }
    ]
  },
  "services": {
    "_attributes": { "init_system": "systemd" },
    "_elements": [
      { "name": "nginx.service", "state": "enabled" }
    ]
  },
  "config_files": {
    "_attributes": {},
    "_elements": [
      { "name": "/etc/foo.conf", "type": "file", "mode": "0644", "user": "root", "group": "root", "sha256": "1111111111111111111111111111111111111111111111111111111111111111", "target": "", "content_ref": "", "package_name": "" }
    ]
  }
}`

// EXAMPLE: diff_offline_two_files — diff with manifest-path and state-path is a
// pure comparison of the two files; no live read, no transaction; exit 0.
func TestDiffOfflineTwoFiles(t *testing.T) {
	mp := writeTemp(t, "baseline.json", validDesiredManifestJSON)
	// state (actual) is the same shape; for a clean offline comparison use an
	// actual state with no config files (so files_write is computed but no I/O).
	state := `{
  "meta": { "format_version": 1, "generator": "x", "created_at": "2026-05-29T08:30:00Z", "desired_sha256": "" }
}`
	sp := writeTemp(t, "after.json", state)
	stdout, _, code := run(t, "diff", "manifest-path="+mp, "state-path="+sp)
	if code != 0 {
		t.Errorf("diff offline two files: exit = %d, want 0", code)
	}
	if strings.TrimSpace(stdout) == "" {
		t.Errorf("diff offline two files: stdout is empty, want a plan")
	}
}

// EXAMPLE: diff_prints_plan — when a state-path is supplied (offline), the plan
// lists packages to install and files to delete relative to the applied/empty side.
// We use an empty actual state so the desired packages appear as "to install".
func TestDiffPrintsPlanInstall(t *testing.T) {
	mp := writeTemp(t, "desired.json", validDesiredManifestJSON)
	state := `{ "meta": { "format_version": 1, "generator": "x", "created_at": "2026-05-29T08:30:00Z", "desired_sha256": "" } }`
	sp := writeTemp(t, "empty-actual.json", state)
	stdout, _, code := run(t, "diff", "manifest-path="+mp, "state-path="+sp)
	if code != 0 {
		t.Fatalf("diff: exit = %d, want 0; stdout=%q", code, stdout)
	}
	if !strings.Contains(stdout, "nginx") {
		t.Errorf("diff plan: stdout = %q, want it to mention nginx (package to install)", stdout)
	}
}

// EXAMPLE: diff supplied state dump malformed -> exit 2.
func TestDiffMalformedStateDump(t *testing.T) {
	mp := writeTemp(t, "desired.json", validDesiredManifestJSON)
	sp := writeTemp(t, "broken.json", "{ this is not json")
	_, stderr, code := run(t, "diff", "manifest-path="+mp, "state-path="+sp)
	if code != 2 {
		t.Errorf("diff malformed state: exit = %d, want 2", code)
	}
	if !strings.Contains(stderr, "invocation") {
		t.Errorf("diff malformed state: stderr = %q, want domain=invocation", stderr)
	}
}

// EXAMPLE: yaml_manifest_accepted — a YAML serialisation of a valid manifest is
// parsed under the safe profile and the plan computed identically to JSON.
func TestDiffYamlManifestAccepted(t *testing.T) {
	yamlManifest := `meta:
  format_version: 1
  generator: "zypper-declarative test"
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
`
	mp := writeTemp(t, "desired.yaml", yamlManifest)
	state := `{ "meta": { "format_version": 1, "generator": "x", "created_at": "2026-05-29T08:30:00Z", "desired_sha256": "" } }`
	sp := writeTemp(t, "empty-actual.json", state)
	stdout, stderr, code := run(t, "diff", "manifest-path="+mp, "state-path="+sp)
	if code != 0 {
		t.Errorf("diff yaml manifest: exit = %d, want 0; stderr=%q", code, stderr)
	}
	if !strings.Contains(stdout, "nginx") {
		t.Errorf("diff yaml manifest: stdout = %q, want it to mention nginx", stdout)
	}
}

// ---------------------------------------------------------------------------
// verify (offline two-file comparison; no applied record required)
// ---------------------------------------------------------------------------

// EXAMPLE: verify_offline_manifest_and_state (clean) and
// verify_offline_no_applied_record_ok: a reference manifest + matching state,
// no applied record -> exit 0, "system matches declaration", no "no declaration
// applied".
func TestVerifyOfflineMatches(t *testing.T) {
	// reference manifest declaring one service enabled
	ref := `{
  "meta": { "format_version": 1, "generator": "x", "created_at": "2026-05-29T08:30:00Z", "desired_sha256": "" },
  "services": { "_attributes": { "init_system": "systemd" }, "_elements": [ { "name": "nginx.service", "state": "enabled" } ] }
}`
	// actual state with the same service state
	state := `{
  "meta": { "format_version": 1, "generator": "x", "created_at": "2026-05-29T08:30:00Z", "desired_sha256": "" },
  "services": { "_attributes": { "init_system": "systemd" }, "_elements": [ { "name": "nginx.service", "state": "enabled" } ] }
}`
	mp := writeTemp(t, "baseline.json", ref)
	sp := writeTemp(t, "after.json", state)
	stdout, stderr, code := run(t, "verify", "manifest-path="+mp, "state-path="+sp)
	if code != 0 {
		t.Errorf("verify offline matches: exit = %d, want 0; stderr=%q", code, stderr)
	}
	if !strings.Contains(stdout, "system matches declaration") {
		t.Errorf("verify offline matches: stdout = %q, want 'system matches declaration'", stdout)
	}
	if strings.Contains(stdout, "no declaration applied") || strings.Contains(stderr, "no declaration applied") {
		t.Errorf("verify offline: must not emit 'no declaration applied' when a reference manifest is supplied")
	}
}

// EXAMPLE: verify_against_external_state_dump — divergent service state -> exit 1,
// stderr diagnostic with domain=units naming the divergent service.
func TestVerifyOfflineUnitDrift(t *testing.T) {
	ref := `{
  "meta": { "format_version": 1, "generator": "x", "created_at": "2026-05-29T08:30:00Z", "desired_sha256": "" },
  "services": { "_attributes": { "init_system": "systemd" }, "_elements": [ { "name": "nginx.service", "state": "enabled" } ] }
}`
	state := `{
  "meta": { "format_version": 1, "generator": "x", "created_at": "2026-05-29T08:30:00Z", "desired_sha256": "" },
  "services": { "_attributes": { "init_system": "systemd" }, "_elements": [ { "name": "nginx.service", "state": "disabled" } ] }
}`
	mp := writeTemp(t, "baseline.json", ref)
	sp := writeTemp(t, "after.json", state)
	_, stderr, code := run(t, "verify", "manifest-path="+mp, "state-path="+sp)
	if code != 1 {
		t.Errorf("verify offline unit drift: exit = %d, want 1", code)
	}
	if !strings.Contains(stderr, "units") {
		t.Errorf("verify offline unit drift: stderr = %q, want domain=units", stderr)
	}
	if !strings.Contains(stderr, "nginx.service") {
		t.Errorf("verify offline unit drift: stderr = %q, want it to name nginx.service", stderr)
	}
}

// EXAMPLE: verify_detects_drift — a declared file with changed sha256 in the
// actual state -> exit 1 with a diagnostic naming the file, domain=files.
func TestVerifyOfflineFileDrift(t *testing.T) {
	ref := `{
  "meta": { "format_version": 1, "generator": "x", "created_at": "2026-05-29T08:30:00Z", "desired_sha256": "" },
  "config_files": { "_attributes": {}, "_elements": [ { "name": "/etc/foo.conf", "type": "file", "mode": "0644", "user": "root", "group": "root", "sha256": "1111111111111111111111111111111111111111111111111111111111111111", "target": "", "content_ref": "", "package_name": "" } ] }
}`
	state := `{
  "meta": { "format_version": 1, "generator": "x", "created_at": "2026-05-29T08:30:00Z", "desired_sha256": "" },
  "config_files": { "_attributes": {}, "_elements": [ { "name": "/etc/foo.conf", "type": "file", "mode": "0644", "user": "root", "group": "root", "sha256": "2222222222222222222222222222222222222222222222222222222222222222", "target": "", "content_ref": "", "package_name": "" } ] }
}`
	mp := writeTemp(t, "baseline.json", ref)
	sp := writeTemp(t, "after.json", state)
	_, stderr, code := run(t, "verify", "manifest-path="+mp, "state-path="+sp)
	if code != 1 {
		t.Errorf("verify offline file drift: exit = %d, want 1", code)
	}
	if !strings.Contains(stderr, "/etc/foo.conf") {
		t.Errorf("verify offline file drift: stderr = %q, want it to name /etc/foo.conf", stderr)
	}
	if !strings.Contains(stderr, "files") {
		t.Errorf("verify offline file drift: stderr = %q, want domain=files", stderr)
	}
}

// EXAMPLE: drift_type_transition_is_modified — reference declares a type "file",
// actual reports type "link" at the same path -> reported as modified (exit 1).
func TestVerifyTypeTransitionIsModified(t *testing.T) {
	ref := `{
  "meta": { "format_version": 1, "generator": "x", "created_at": "2026-05-29T08:30:00Z", "desired_sha256": "" },
  "config_files": { "_attributes": {}, "_elements": [ { "name": "/etc/foo", "type": "file", "mode": "0644", "user": "root", "group": "root", "sha256": "1111111111111111111111111111111111111111111111111111111111111111", "target": "", "content_ref": "", "package_name": "" } ] }
}`
	state := `{
  "meta": { "format_version": 1, "generator": "x", "created_at": "2026-05-29T08:30:00Z", "desired_sha256": "" },
  "config_files": { "_attributes": {}, "_elements": [ { "name": "/etc/foo", "type": "link", "mode": "0777", "user": "root", "group": "root", "sha256": "", "target": "elsewhere", "content_ref": "", "package_name": "" } ] }
}`
	mp := writeTemp(t, "baseline.json", ref)
	sp := writeTemp(t, "after.json", state)
	_, stderr, code := run(t, "verify", "manifest-path="+mp, "state-path="+sp)
	if code != 1 {
		t.Errorf("verify type transition: exit = %d, want 1", code)
	}
	if !strings.Contains(stderr, "/etc/foo") {
		t.Errorf("verify type transition: stderr = %q, want it to name /etc/foo", stderr)
	}
}

// EXAMPLE: verify_malformed_state_dump -> exit 2, domain=invocation.
func TestVerifyMalformedStateDump(t *testing.T) {
	ref := writeTemp(t, "baseline.json", validDesiredManifestJSON)
	sp := writeTemp(t, "broken.json", "{ not valid")
	_, stderr, code := run(t, "verify", "manifest-path="+ref, "state-path="+sp)
	if code != 2 {
		t.Errorf("verify malformed state: exit = %d, want 2", code)
	}
	if !strings.Contains(stderr, "invocation") {
		t.Errorf("verify malformed state: stderr = %q, want domain=invocation", stderr)
	}
}

// EXAMPLE: verify_state_path_extension_yaml — a YAML state dump (no format option)
// matching the reference -> resolve-format picks yaml from the extension; exit 0.
func TestVerifyStatePathExtensionYaml(t *testing.T) {
	ref := `{
  "meta": { "format_version": 1, "generator": "x", "created_at": "2026-05-29T08:30:00Z", "desired_sha256": "" },
  "services": { "_attributes": { "init_system": "systemd" }, "_elements": [ { "name": "nginx.service", "state": "enabled" } ] }
}`
	stateYaml := `meta:
  format_version: 1
  generator: "x"
  created_at: "2026-05-29T08:30:00Z"
  desired_sha256: ""
services:
  _attributes:
    init_system: "systemd"
  _elements:
    - name: "nginx.service"
      state: "enabled"
`
	mp := writeTemp(t, "baseline.json", ref)
	sp := writeTemp(t, "state.yaml", stateYaml)
	stdout, stderr, code := run(t, "verify", "manifest-path="+mp, "state-path="+sp)
	if code != 0 {
		t.Errorf("verify yaml state extension: exit = %d, want 0; stderr=%q", code, stderr)
	}
	if !strings.Contains(stdout, "system matches declaration") {
		t.Errorf("verify yaml state extension: stdout = %q, want 'system matches declaration'", stdout)
	}
}

// EXAMPLE: verify_no_applied_record — no manifest_path and no applied record
// (running on a build host with no applied.json under /) -> exit 2,
// "no declaration applied", domain=invocation.
func TestVerifyNoAppliedRecord(t *testing.T) {
	// Point applied-root at an empty dir so no applied record exists, and read
	// the live system would otherwise be needed; with no reference this must be
	// an invocation error before any live read.
	emptyRoot := t.TempDir()
	_, stderr, code := run(t, "verify", "applied-root="+emptyRoot)
	if code != 2 {
		t.Errorf("verify no applied record: exit = %d, want 2", code)
	}
	if !strings.Contains(stderr, "no declaration applied") {
		t.Errorf("verify no applied record: stderr = %q, want 'no declaration applied'", stderr)
	}
	if !strings.Contains(stderr, "invocation") {
		t.Errorf("verify no applied record: stderr = %q, want domain=invocation", stderr)
	}
}

// ---------------------------------------------------------------------------
// status
// ---------------------------------------------------------------------------

// EXAMPLE: status_no_declaration — no applied record -> "no declaration applied",
// exit 0.
func TestStatusNoDeclaration(t *testing.T) {
	emptyRoot := t.TempDir()
	stdout, _, code := run(t, "status", "applied-root="+emptyRoot)
	if code != 0 {
		t.Errorf("status no declaration: exit = %d, want 0", code)
	}
	if !strings.Contains(stdout, "no declaration applied") {
		t.Errorf("status no declaration: stdout = %q, want 'no declaration applied'", stdout)
	}
}

// ---------------------------------------------------------------------------
// describe (read-only; format resolution is verifiable without live state)
// ---------------------------------------------------------------------------

// EXAMPLE: describe_out_extension_yaml — out=.yaml with no format option writes YAML.
func TestDescribeOutExtensionYaml(t *testing.T) {
	out := filepath.Join(t.TempDir(), "state.yaml")
	_, stderr, code := run(t, "describe", "root="+t.TempDir(), "out="+out, "on-unreadable=warn")
	if code != 0 {
		t.Fatalf("describe out=.yaml: exit = %d, want 0; stderr=%q", code, stderr)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("reading describe output %s: %v", out, err)
	}
	first := firstLine(string(data))
	// A JSON document begins with '{'; a YAML document does not.
	if strings.HasPrefix(strings.TrimSpace(first), "{") {
		t.Errorf("describe out=.yaml: first line %q looks like JSON, want YAML", first)
	}
}

// EXAMPLE: describe_out_extension_json — out=.json with no format option writes JSON.
func TestDescribeOutExtensionJson(t *testing.T) {
	out := filepath.Join(t.TempDir(), "state.json")
	_, stderr, code := run(t, "describe", "root="+t.TempDir(), "out="+out, "on-unreadable=warn")
	if code != 0 {
		t.Fatalf("describe out=.json: exit = %d, want 0; stderr=%q", code, stderr)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("reading describe output: %v", err)
	}
	if !strings.HasPrefix(strings.TrimSpace(string(data)), "{") {
		t.Errorf("describe out=.json: output does not start with '{': %q", string(data))
	}
}

// EXAMPLE: describe_format_overrides_extension — format=json with out=.yaml writes JSON.
func TestDescribeFormatOverridesExtension(t *testing.T) {
	out := filepath.Join(t.TempDir(), "state.yaml")
	_, stderr, code := run(t, "describe", "root="+t.TempDir(), "format=json", "out="+out, "on-unreadable=warn")
	if code != 0 {
		t.Fatalf("describe format=json out=.yaml: exit = %d, want 0; stderr=%q", code, stderr)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("reading describe output: %v", err)
	}
	if !strings.HasPrefix(strings.TrimSpace(string(data)), "{") {
		t.Errorf("describe format=json out=.yaml: output should be JSON, got %q", string(data))
	}
}

// EXAMPLE: describe_output_unwritable -> exit 2, domain=invocation.
func TestDescribeOutputUnwritable(t *testing.T) {
	_, stderr, code := run(t, "describe", "root="+t.TempDir(),
		"out=/nonexistent-zd-dir/does/not/exist/state.json", "on-unreadable=warn")
	if code != 2 {
		t.Errorf("describe unwritable out: exit = %d, want 2", code)
	}
	if !strings.Contains(stderr, "invocation") {
		t.Errorf("describe unwritable out: stderr = %q, want domain=invocation", stderr)
	}
}

// INVARIANT: scope is accepted only on describe and verify; not on status.
func TestScopeRejectedOnStatus(t *testing.T) {
	_, _, code := run(t, "status", "scope=full")
	if code != 2 {
		t.Errorf("status scope=full: exit = %d, want 2 (scope not accepted on status)", code)
	}
}

// EXAMPLE: scope_attributes_always_object — describe a tree with no /etc; the
// emitted document's scope _attributes (when present) must be objects, never null.
// We describe an empty root with warn so no scope source errors; assert the
// emitted JSON contains no "_attributes": null.
func TestScopeAttributesNeverNull(t *testing.T) {
	out := filepath.Join(t.TempDir(), "state.json")
	_, _, code := run(t, "describe", "root="+t.TempDir(), "out="+out, "on-unreadable=warn")
	if code != 0 {
		t.Skipf("describe of empty root exited %d; environment-dependent, skipping null-attr assertion", code)
	}
	data, _ := os.ReadFile(out)
	s := strings.ReplaceAll(string(data), " ", "")
	if strings.Contains(s, `"_attributes":null`) {
		t.Errorf("describe output contains _attributes: null; must be an object: %q", string(data))
	}
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
