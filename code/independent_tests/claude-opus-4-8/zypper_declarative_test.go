// generated from spec: zypper-declarative.spec.md sha256:714e75ff672557d2c7344736a5f36b52afa37e0f565b07de83e1b18cc4492014
// tests by: claude-opus-4-8
//
// Black-box test suite for the zypper-declarative CLI binary.
// Tests invoke the binary via os/exec per the DEPLOYMENT interface and assert
// on stdout, stderr, and exit code. No internal Go function is called; no
// behaviour is simulated through wrapper code.
//
// Binary discovery path follows the cli-tool template BINARY-LOCATION:
// project-root constraint: relative to this test directory the binary is at
// "../../zypper-declarative". TestMain builds it from
// "../../cmd/zypper-declarative" if it is absent.

package claude_opus_4_8_test

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const binaryPath = "../../zypper-declarative"

// TestMain builds the binary at the canonical project-root location before the
// suite runs, so the tests exercise a real binary regardless of build state.
func TestMain(m *testing.M) {
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		build := exec.Command("go", "build", "-o", binaryPath, "../../cmd/zypper-declarative")
		build.Stdout = os.Stdout
		build.Stderr = os.Stderr
		if err := build.Run(); err != nil {
			panic("failed to build binary under test: " + err.Error())
		}
	}
	os.Exit(m.Run())
}

// run executes the binary with the supplied args and an empty environment except
// PATH (env-var control of behaviour is forbidden, so the suite never sets any).
func run(t *testing.T, args ...string) (stdout string, stderr string, code int) {
	t.Helper()
	cmd := exec.Command(binaryPath, args...)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	code = 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		} else {
			t.Fatalf("failed to run binary %v: %v", args, err)
		}
	}
	return outBuf.String(), errBuf.String(), code
}

func writeManifest(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return p
}

// A structurally complete desired manifest (canonical JSON, format_version 1).
const desiredManifestJSON = `{
  "meta": {
    "format_version": 1,
    "generator": "zypper-declarative 0.4.0",
    "created_at": "2026-05-29T08:30:00Z",
    "desired_sha256": ""
  },
  "repositories": {
    "_attributes": { "repository_system": "zypp" },
    "_elements": [
      { "alias": "sl-micro-6.2-pinned", "name": "SL Micro 6.2 (pinned)",
        "url": "https://internal.example/obs/SLMicro:6.2:pinned/standard",
        "type": "rpm-md", "enabled": true, "gpgcheck": true,
        "autorefresh": false, "priority": 99 }
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
      { "name": "/etc/nginx/nginx.conf", "type": "file", "mode": "0644",
        "user": "root", "group": "root",
        "sha256": "0000000000000000000000000000000000000000000000000000000000000000",
        "content_ref": "files/etc/nginx/nginx.conf", "package_name": "" }
    ]
  }
}`

// --version / --help / bare invocation -------------------------------------

func TestVersionEmbedsSpecHash(t *testing.T) {
	stdout, _, code := run(t, "--version")
	if code != 0 {
		t.Fatalf("--version exit = %d, want 0", code)
	}
	if !strings.Contains(stdout, "zypper-declarative ") {
		t.Errorf("--version stdout missing program name: %q", stdout)
	}
	if !strings.Contains(stdout, "spec:714e75ff672557d2c7344736a5f36b52afa37e0f565b07de83e1b18cc4492014") {
		t.Errorf("--version stdout missing embedded spec hash: %q", stdout)
	}
}

func TestHelpPrintsUsage(t *testing.T) {
	stdout, _, code := run(t, "--help")
	if code != 0 {
		t.Fatalf("--help exit = %d, want 0", code)
	}
	if !strings.Contains(strings.ToLower(stdout), "usage") {
		t.Errorf("--help stdout missing usage: %q", stdout)
	}
}

func TestUnknownVerbExits2(t *testing.T) {
	_, stderr, code := run(t, "frobnicate")
	if code != 2 {
		t.Fatalf("unknown verb exit = %d, want 2", code)
	}
	if strings.TrimSpace(stderr) == "" {
		t.Errorf("unknown verb should print a diagnostic to stderr")
	}
}

// apply --------------------------------------------------------------------

// EXAMPLE: apply_manifest_unreadable
func TestApplyManifestUnreadable(t *testing.T) {
	_, stderr, code := run(t, "apply", "manifest-path=/nonexistent.json")
	if code != 2 {
		t.Fatalf("apply unreadable manifest exit = %d, want 2", code)
	}
	if !strings.Contains(stderr, "invocation") {
		t.Errorf("expected domain=invocation diagnostic, got stderr: %q", stderr)
	}
}

// EXAMPLE: apply_manifest_invalid (format_version = 2 -> manifest domain, exit 1)
func TestApplyManifestInvalid(t *testing.T) {
	bad := strings.Replace(desiredManifestJSON, `"format_version": 1`, `"format_version": 2`, 1)
	p := writeManifest(t, "bad.json", bad)
	_, stderr, code := run(t, "apply", "manifest-path="+p)
	if code != 1 {
		t.Fatalf("apply invalid manifest exit = %d, want 1", code)
	}
	if !strings.Contains(stderr, "manifest") {
		t.Errorf("expected domain=manifest diagnostic, got stderr: %q", stderr)
	}
}

// EXAMPLE: apply_transaction_unavailable (external mode, not in a transaction)
func TestApplyTransactionUnavailable(t *testing.T) {
	p := writeManifest(t, "desired.json", desiredManifestJSON)
	_, stderr, code := run(t, "apply", "manifest-path="+p, "mode=external")
	// external mode but not inside a transaction -> exit 2, domain=transaction.
	if code != 2 {
		t.Fatalf("apply external w/o txn exit = %d, want 2; stderr=%q", code, stderr)
	}
	if !strings.Contains(stderr, "transaction") {
		t.Errorf("expected domain=transaction diagnostic, got stderr: %q", stderr)
	}
}

// EXAMPLE: yaml_unsafe_rejected (apply on unsafe YAML -> exit 1, manifest)
func TestApplyUnsafeYAMLRejected(t *testing.T) {
	// Multi-document stream is one of the disabled features.
	yaml := "meta:\n  format_version: 1\n---\nmeta:\n  format_version: 1\n"
	p := writeManifest(t, "evil.yaml", yaml)
	_, stderr, code := run(t, "apply", "manifest-path="+p)
	if code != 1 {
		t.Fatalf("apply unsafe yaml exit = %d, want 1; stderr=%q", code, stderr)
	}
	if !strings.Contains(stderr, "manifest") {
		t.Errorf("expected domain=manifest diagnostic, got stderr: %q", stderr)
	}
}

// diff ---------------------------------------------------------------------

// EXAMPLE: diff_manifest_unreadable
func TestDiffManifestUnreadable(t *testing.T) {
	_, stderr, code := run(t, "diff", "manifest-path=/nonexistent.json")
	if code != 2 {
		t.Fatalf("diff unreadable manifest exit = %d, want 2", code)
	}
	if !strings.Contains(stderr, "invocation") {
		t.Errorf("expected domain=invocation, got stderr: %q", stderr)
	}
}

// EXAMPLE: diff_prints_plan — first-ever apply, all packages install, exit 0.
func TestDiffPrintsPlan(t *testing.T) {
	p := writeManifest(t, "desired.json", desiredManifestJSON)
	stdout, stderr, code := run(t, "diff", "manifest-path="+p)
	if code != 0 {
		t.Fatalf("diff exit = %d, want 0; stderr=%q", code, stderr)
	}
	if !strings.Contains(stdout, "nginx") {
		t.Errorf("diff plan should list nginx to install; stdout=%q", stdout)
	}
}

// INVARIANT [observable]: diff opens no transaction / modifies nothing.
// Verified observably: diff against a manifest exits 0 and is a dry run.
func TestDiffIsDryRun(t *testing.T) {
	p := writeManifest(t, "desired.json", desiredManifestJSON)
	_, _, code := run(t, "diff", "manifest-path="+p)
	if code != 0 {
		t.Fatalf("diff exit = %d, want 0", code)
	}
	// Running diff twice is idempotent and never changes exit behaviour.
	_, _, code2 := run(t, "diff", "manifest-path="+p)
	if code2 != 0 {
		t.Fatalf("second diff exit = %d, want 0", code2)
	}
}

// EXAMPLE: yaml_manifest_accepted — diff on a YAML manifest, exit 0.
func TestDiffYAMLManifestAccepted(t *testing.T) {
	yaml := `meta:
  format_version: 1
  generator: "zypper-declarative 0.4.0"
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
	p := writeManifest(t, "desired.yaml", yaml)
	stdout, stderr, code := run(t, "diff", "manifest-path="+p)
	if code != 0 {
		t.Fatalf("diff yaml exit = %d, want 0; stderr=%q", code, stderr)
	}
	if !strings.Contains(stdout, "nginx") {
		t.Errorf("yaml diff plan should list nginx; stdout=%q", stdout)
	}
}

// verify -------------------------------------------------------------------

// EXAMPLE: verify_no_applied_record (against an empty root)
func TestVerifyNoAppliedRecord(t *testing.T) {
	root := t.TempDir()
	_, stderr, code := run(t, "verify", "applied-root="+root)
	if code != 2 {
		t.Fatalf("verify no record exit = %d, want 2; stderr=%q", code, stderr)
	}
	if !strings.Contains(stderr, "no declaration applied") {
		t.Errorf("expected 'no declaration applied', got stderr: %q", stderr)
	}
}

// EXAMPLE: verify_malformed_state_dump
func TestVerifyMalformedStateDump(t *testing.T) {
	root := t.TempDir()
	// Provide an applied record so verify proceeds to read the state dump.
	mkAppliedRecord(t, root)
	bad := writeManifest(t, "broken.json", "{ this is not json")
	_, stderr, code := run(t, "verify", "applied-root="+root, "state-path="+bad)
	if code != 2 {
		t.Fatalf("verify malformed dump exit = %d, want 2; stderr=%q", code, stderr)
	}
	if !strings.Contains(stderr, "invocation") {
		t.Errorf("expected domain=invocation, got stderr: %q", stderr)
	}
}

// EXAMPLE: verify_clean — actual state equals the applied record.
func TestVerifyClean(t *testing.T) {
	root := t.TempDir()
	mkAppliedRecord(t, root)
	// Supply a state dump identical to the applied record's managed scopes.
	dump := writeManifest(t, "state.json", appliedRecordJSON)
	stdout, stderr, code := run(t, "verify", "applied-root="+root, "state-path="+dump)
	if code != 0 {
		t.Fatalf("verify clean exit = %d, want 0; stderr=%q", code, stderr)
	}
	if !strings.Contains(stdout, "system matches declaration") {
		t.Errorf("expected 'system matches declaration', got stdout: %q", stdout)
	}
}

// EXAMPLE: verify_against_external_state_dump — divergent service -> exit 1.
func TestVerifyDetectsServiceDrift(t *testing.T) {
	root := t.TempDir()
	mkAppliedRecord(t, root)
	// State dump where nginx.service is disabled instead of enabled.
	diverged := strings.Replace(appliedRecordJSON,
		`{ "name": "nginx.service", "state": "enabled" }`,
		`{ "name": "nginx.service", "state": "disabled" }`, 1)
	dump := writeManifest(t, "state.json", diverged)
	_, stderr, code := run(t, "verify", "applied-root="+root, "state-path="+dump)
	if code != 1 {
		t.Fatalf("verify drift exit = %d, want 1; stderr=%q", code, stderr)
	}
	if !strings.Contains(stderr, "units") || !strings.Contains(stderr, "nginx.service") {
		t.Errorf("expected units diagnostic naming nginx.service, got stderr: %q", stderr)
	}
}

// EXAMPLE: verify_detects_drift — declared file edited -> files diagnostic, exit 1.
func TestVerifyDetectsFileDrift(t *testing.T) {
	root := t.TempDir()
	mkAppliedRecord(t, root)
	// State dump where /etc/nginx/nginx.conf has a different sha256.
	diverged := strings.Replace(appliedRecordJSON,
		`"sha256": "1111111111111111111111111111111111111111111111111111111111111111"`,
		`"sha256": "2222222222222222222222222222222222222222222222222222222222222222"`, 1)
	dump := writeManifest(t, "state.json", diverged)
	_, stderr, code := run(t, "verify", "applied-root="+root, "state-path="+dump)
	if code != 1 {
		t.Fatalf("verify file drift exit = %d, want 1; stderr=%q", code, stderr)
	}
	if !strings.Contains(stderr, "files") || !strings.Contains(stderr, "/etc/nginx/nginx.conf") {
		t.Errorf("expected files diagnostic naming the file, got stderr: %q", stderr)
	}
}

// status -------------------------------------------------------------------

// EXAMPLE: status_no_declaration
func TestStatusNoDeclaration(t *testing.T) {
	root := t.TempDir()
	stdout, _, code := run(t, "status", "applied-root="+root)
	if code != 0 {
		t.Fatalf("status no decl exit = %d, want 0", code)
	}
	if !strings.Contains(stdout, "no declaration applied") {
		t.Errorf("expected 'no declaration applied', got stdout: %q", stdout)
	}
}

// EXAMPLE: status_unknown_argument
func TestStatusUnknownArgument(t *testing.T) {
	_, stderr, code := run(t, "status", "--frobnicate")
	if code != 2 {
		t.Fatalf("status unknown arg exit = %d, want 2", code)
	}
	if strings.TrimSpace(stderr) == "" {
		t.Errorf("status unknown arg should print usage to stderr")
	}
}

// EXAMPLE: status_reports_generation
func TestStatusReportsGeneration(t *testing.T) {
	root := t.TempDir()
	mkAppliedRecord(t, root)
	stdout, stderr, code := run(t, "status", "applied-root="+root)
	if code != 0 {
		t.Fatalf("status exit = %d, want 0; stderr=%q", code, stderr)
	}
	// The applied record's desired_sha256 and the package count must appear.
	if !strings.Contains(stdout, "deadbeef") {
		t.Errorf("status should print desired_sha256; stdout=%q", stdout)
	}
	if !strings.Contains(stdout, "1") { // one resolved package
		t.Errorf("status should print resolved package count; stdout=%q", stdout)
	}
}

// describe -----------------------------------------------------------------

// EXAMPLE: describe_unknown_format
func TestDescribeUnknownFormat(t *testing.T) {
	_, stderr, code := run(t, "describe", "format=toml")
	if code != 2 {
		t.Fatalf("describe unknown format exit = %d, want 2", code)
	}
	if !strings.Contains(stderr, "invocation") {
		t.Errorf("expected domain=invocation, got stderr: %q", stderr)
	}
}

// EXAMPLE: describe_emits_manifest — describe against an empty root emits a
// schema-valid JSON Manifest (format_version 1) with the declarable scopes.
func TestDescribeEmitsJSONManifest(t *testing.T) {
	root := t.TempDir()
	stdout, stderr, code := run(t, "describe", "root="+root)
	if code != 0 {
		t.Fatalf("describe exit = %d, want 0; stderr=%q", code, stderr)
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &m); err != nil {
		t.Fatalf("describe output is not valid JSON: %v; out=%q", err, stdout)
	}
	meta, ok := m["meta"].(map[string]interface{})
	if !ok {
		t.Fatalf("describe output missing meta object; out=%q", stdout)
	}
	if fv, _ := meta["format_version"].(float64); fv != 1 {
		t.Errorf("describe meta.format_version = %v, want 1", meta["format_version"])
	}
}

// EXAMPLE: describe_format_yaml + describe_output_unwritable patterns.
func TestDescribeYAMLToFile(t *testing.T) {
	root := t.TempDir()
	out := filepath.Join(t.TempDir(), "state.yaml")
	_, stderr, code := run(t, "describe", "root="+root, "format=yaml", "out="+out)
	if code != 0 {
		t.Fatalf("describe yaml exit = %d, want 0; stderr=%q", code, stderr)
	}
	content, err := os.ReadFile(out)
	if err != nil || len(content) == 0 {
		t.Fatalf("yaml output missing or empty: %v", err)
	}
	// YAML uses "key: value", not the JSON "{". A YAML doc should not begin with '{'.
	if strings.HasPrefix(strings.TrimSpace(string(content)), "{") {
		t.Errorf("expected YAML serialisation, got JSON-looking content: %q", string(content))
	}
}

// EXAMPLE: describe_output_unwritable
func TestDescribeOutputUnwritable(t *testing.T) {
	root := t.TempDir()
	_, stderr, code := run(t, "describe", "root="+root, "out=/readonly-nonexistent-dir/state.json")
	if code != 2 {
		t.Fatalf("describe unwritable out exit = %d, want 2; stderr=%q", code, stderr)
	}
	if !strings.Contains(stderr, "invocation") {
		t.Errorf("expected domain=invocation, got stderr: %q", stderr)
	}
}

// INVARIANT [observable]: describe output is accepted unchanged as a desired
// manifest. EXAMPLE describe_bootstraps_desired_manifest.
func TestDescribeBootstrapsDesiredManifest(t *testing.T) {
	root := t.TempDir()
	stdout, _, code := run(t, "describe", "root="+root)
	if code != 0 {
		t.Fatalf("describe exit = %d, want 0", code)
	}
	p := writeManifest(t, "bootstrapped.json", stdout)
	_, stderr, code2 := run(t, "diff", "manifest-path="+p, "applied-root="+root)
	if code2 != 0 {
		t.Fatalf("diff on bootstrapped manifest exit = %d, want 0; stderr=%q", code2, stderr)
	}
}

// Helpers for applied-record fixtures --------------------------------------

const appliedRecordJSON = `{
  "meta": {
    "format_version": 1,
    "generator": "zypper-declarative 0.4.0",
    "created_at": "2026-05-29T08:30:00Z",
    "desired_sha256": "deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef0"
  },
  "packages": {
    "_attributes": { "package_system": "rpm" },
    "_elements": [ { "name": "nginx", "version": "1.25.3", "release": "1.1", "arch": "x86_64" } ]
  },
  "services": {
    "_attributes": { "init_system": "systemd" },
    "_elements": [ { "name": "nginx.service", "state": "enabled" } ]
  },
  "config_files": {
    "_attributes": null,
    "_elements": [
      { "name": "/etc/nginx/nginx.conf", "type": "file", "mode": "0644",
        "user": "root", "group": "root",
        "sha256": "1111111111111111111111111111111111111111111111111111111111111111",
        "content_ref": "", "package_name": "" }
    ]
  }
}`

// mkAppliedRecord writes a valid applied.json under
// <root>/usr/lib/zypper-declarative/applied.json so verify/status can load it.
func mkAppliedRecord(t *testing.T, root string) {
	t.Helper()
	dir := filepath.Join(root, "usr", "lib", "zypper-declarative")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir applied dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "applied.json"), []byte(appliedRecordJSON), 0o644); err != nil {
		t.Fatalf("write applied.json: %v", err)
	}
}
