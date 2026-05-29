// generated from spec: zypper-declarative.spec.md sha256:58e1636e2de82ab81a5cd3f81d6b3c9ac6a8976e18f9abb2bbd2b2aba56fe4d4
// tests by: claude-opus-4-8
//
// Black-box test suite for the zypper-declarative CLI binary.
//
// The interface under test is the CLI binary declared in the spec's DEPLOYMENT
// section. Tests invoke it via exec.Command and assert on stdout, stderr, and
// exit code. Tests never import or call the implementation's internal packages.
//
// Per the cli-tool template BINARY-LOCATION constraint (project-root), the
// binary lives at "../../zypper-declarative" relative to this directory.
// TestMain builds it from "../../cmd/zypper-declarative".
package independent_tests

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// binaryRelPath is the canonical path the binary lives at, per the cli-tool
// template's BINARY-LOCATION: project-root constraint, expressed relative to
// this test directory (independent_tests/<llm-name>/).
const binaryRelPath = "../../zypper-declarative"

// binaryAbsPath is resolved in TestMain.
var binaryAbsPath string

func TestMain(m *testing.M) {
	// Build the binary at the canonical location from the canonical source path.
	// The translator must place the entry point at cmd/zypper-declarative/main.go.
	abs, err := filepath.Abs(binaryRelPath)
	if err != nil {
		panic(err)
	}
	binaryAbsPath = abs

	build := exec.Command("go", "build", "-o", abs, "./cmd/zypper-declarative")
	build.Dir = mustProjectRoot()
	build.Env = append(os.Environ(), "CGO_ENABLED=0")
	var bo bytes.Buffer
	build.Stdout = &bo
	build.Stderr = &bo
	if err := build.Run(); err != nil {
		// Could not build; report and fail all tests by exiting non-zero.
		_, _ = os.Stderr.WriteString("failed to build binary under test:\n" + bo.String() + "\n")
		os.Exit(2)
	}
	os.Exit(m.Run())
}

// mustProjectRoot returns the project root (two directories up from this test
// directory), where go.mod and the built binary live.
func mustProjectRoot() string {
	root, err := filepath.Abs("../..")
	if err != nil {
		panic(err)
	}
	return root
}

// runResult captures the observable result of one binary invocation.
type runResult struct {
	stdout   string
	stderr   string
	exitCode int
}

// run executes the binary under test with the given args and returns the result.
func run(t *testing.T, args ...string) runResult {
	t.Helper()
	cmd := exec.Command(binaryAbsPath, args...)
	var so, se bytes.Buffer
	cmd.Stdout = &so
	cmd.Stderr = &se
	err := cmd.Run()
	code := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		} else {
			t.Fatalf("failed to run binary %q args %v: %v", binaryAbsPath, args, err)
		}
	}
	return runResult{stdout: so.String(), stderr: se.String(), exitCode: code}
}

// writeTemp writes content to a temp file with the given suffix and returns the path.
func writeTemp(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatalf("writing temp file %s: %v", p, err)
	}
	return p
}

// ---------------------------------------------------------------------------
// Test fixtures: structurally complete manifests in the shared schema.
// ---------------------------------------------------------------------------

// validManifestJSON is a structurally complete, schema-valid desired manifest
// (Machinery format_version 1, ScopeWrapper idiom, underscore_style fields).
const validManifestJSON = `{
  "meta": {
    "format_version": 1,
    "generator": "test",
    "created_at": "2026-05-29T08:30:00Z",
    "desired_sha256": ""
  },
  "repositories": {
    "_attributes": { "repository_system": "zypp" },
    "_elements": [
      {
        "alias": "sl-micro-6.2-pinned",
        "name": "SL Micro 6.2 (pinned)",
        "url": "https://internal.example/obs/SLMicro/standard",
        "type": "rpm-md",
        "enabled": true,
        "gpgcheck": true,
        "autorefresh": false,
        "priority": 99
      }
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
    "_attributes": null,
    "_elements": [
      {
        "name": "/etc/foo.conf",
        "type": "file",
        "mode": "0644",
        "user": "root",
        "group": "root",
        "sha256": "0000000000000000000000000000000000000000000000000000000000000000",
        "content_ref": "files/etc/foo.conf",
        "package_name": ""
      }
    ]
  }
}`

// equivalent YAML serialisation of the same manifest data model.
const validManifestYAML = `meta:
  format_version: 1
  generator: "test"
  created_at: "2026-05-29T08:30:00Z"
  desired_sha256: ""
repositories:
  _attributes:
    repository_system: "zypp"
  _elements:
    - alias: "sl-micro-6.2-pinned"
      name: "SL Micro 6.2 (pinned)"
      url: "https://internal.example/obs/SLMicro/standard"
      type: "rpm-md"
      enabled: true
      gpgcheck: true
      autorefresh: false
      priority: 99
packages:
  _attributes:
    package_system: "rpm"
  _elements:
    - name: "nginx"
      version: ""
      release: ""
      arch: ""
services:
  _attributes:
    init_system: "systemd"
  _elements:
    - name: "nginx.service"
      state: "enabled"
config_files:
  _attributes: null
  _elements:
    - name: "/etc/foo.conf"
      type: "file"
      mode: "0644"
      user: "root"
      group: "root"
      sha256: "0000000000000000000000000000000000000000000000000000000000000000"
      content_ref: "files/etc/foo.conf"
      package_name: ""
`

// manifestFormatVersion2JSON is a structurally complete manifest that is invalid
// only because meta.format_version is 2 (the schema requires 1).
const manifestFormatVersion2JSON = `{
  "meta": {
    "format_version": 2,
    "generator": "test",
    "created_at": "2026-05-29T08:30:00Z",
    "desired_sha256": ""
  },
  "packages": {
    "_attributes": { "package_system": "rpm" },
    "_elements": [ { "name": "nginx", "version": "", "release": "", "arch": "" } ]
  }
}`

// ===========================================================================
// Top-level CLI contract (DEPLOYMENT section, v0.5.0 changelog item 1)
// ===========================================================================

// EXAMPLE: bare_invocation_shows_help
func TestBareInvocationShowsHelp(t *testing.T) {
	r := run(t)
	if r.exitCode != 0 {
		t.Fatalf("bare invocation: want exit 0, got %d (stderr=%q)", r.exitCode, r.stderr)
	}
	if !strings.Contains(strings.ToLower(r.stdout), "usage") {
		t.Errorf("bare invocation: stdout should contain usage, got %q", r.stdout)
	}
}

// --help prints usage to stdout and exits 0.
func TestHelpFlagStdoutExitZero(t *testing.T) {
	r := run(t, "--help")
	if r.exitCode != 0 {
		t.Fatalf("--help: want exit 0, got %d", r.exitCode)
	}
	if !strings.Contains(strings.ToLower(r.stdout), "usage") {
		t.Errorf("--help: stdout should contain usage, got %q", r.stdout)
	}
}

// -h prints usage to stdout and exits 0.
func TestHelpShortFlagStdoutExitZero(t *testing.T) {
	r := run(t, "-h")
	if r.exitCode != 0 {
		t.Fatalf("-h: want exit 0, got %d", r.exitCode)
	}
	if !strings.Contains(strings.ToLower(r.stdout), "usage") {
		t.Errorf("-h: stdout should contain usage, got %q", r.stdout)
	}
}

// --version prints program name, version, and embedded spec hash to stdout, exit 0.
func TestVersionFlag(t *testing.T) {
	r := run(t, "--version")
	if r.exitCode != 0 {
		t.Fatalf("--version: want exit 0, got %d (stderr=%q)", r.exitCode, r.stderr)
	}
	if !strings.HasPrefix(r.stdout, "zypper-declarative ") {
		t.Errorf("--version: stdout should start with %q, got %q", "zypper-declarative ", r.stdout)
	}
	if !strings.Contains(r.stdout, "spec:") {
		t.Errorf("--version: stdout should contain embedded spec hash (spec:...), got %q", r.stdout)
	}
	if !strings.Contains(r.stdout, "58e1636e2de82ab81a5cd3f81d6b3c9ac6a8976e18f9abb2bbd2b2aba56fe4d4") {
		t.Errorf("--version: stdout should contain the spec sha256, got %q", r.stdout)
	}
}

// EXAMPLE: unknown_verb_rejected
func TestUnknownVerbRejected(t *testing.T) {
	r := run(t, "frobnicate")
	if r.exitCode != 2 {
		t.Fatalf("unknown verb: want exit 2, got %d (stdout=%q)", r.exitCode, r.stdout)
	}
	if !strings.Contains(strings.ToLower(r.stderr), "usage") {
		t.Errorf("unknown verb: stderr should contain usage, got %q", r.stderr)
	}
}

// An unknown option exits 2 with usage to stderr.
func TestUnknownOptionRejected(t *testing.T) {
	r := run(t, "status", "--frobnicate")
	if r.exitCode != 2 {
		t.Fatalf("unknown option: want exit 2, got %d (stdout=%q stderr=%q)", r.exitCode, r.stdout, r.stderr)
	}
	if !strings.Contains(strings.ToLower(r.stderr), "usage") {
		t.Errorf("unknown option: stderr should contain usage, got %q", r.stderr)
	}
}

// EXAMPLE: status_unknown_argument
func TestStatusUnknownArgument(t *testing.T) {
	r := run(t, "status", "--frobnicate")
	if r.exitCode != 2 {
		t.Fatalf("status unknown arg: want exit 2, got %d", r.exitCode)
	}
	if !strings.Contains(strings.ToLower(r.stderr), "usage") {
		t.Errorf("status unknown arg: stderr should contain usage, got %q", r.stderr)
	}
}

// ===========================================================================
// describe verb
// ===========================================================================

// EXAMPLE: describe_unknown_format
func TestDescribeUnknownFormat(t *testing.T) {
	r := run(t, "describe", "format=toml")
	if r.exitCode != 2 {
		t.Fatalf("describe format=toml: want exit 2, got %d (stdout=%q)", r.exitCode, r.stdout)
	}
	if !strings.Contains(strings.ToLower(r.stderr), "usage") {
		t.Errorf("describe format=toml: stderr should contain usage, got %q", r.stderr)
	}
}

// EXAMPLE: describe_output_unwritable
func TestDescribeOutputUnwritable(t *testing.T) {
	// A path under a non-existent / unwritable directory is not writable.
	r := run(t, "describe", "out=/nonexistent-dir-zd/state.json", "on-unreadable=warn")
	if r.exitCode != 2 {
		t.Fatalf("describe unwritable out: want exit 2, got %d (stdout=%q stderr=%q)", r.exitCode, r.stdout, r.stderr)
	}
	if !strings.Contains(strings.ToLower(r.stderr), "invocation") &&
		!strings.Contains(strings.ToLower(r.stderr), "write") &&
		!strings.Contains(strings.ToLower(r.stderr), "unwritable") &&
		!strings.Contains(strings.ToLower(r.stderr), "no such") {
		t.Errorf("describe unwritable out: stderr should report an invocation/write error, got %q", r.stderr)
	}
}

// EXAMPLE: describe_emits_manifest — describe on / under warn must succeed and
// emit a JSON document with meta.format_version = 1. Running unprivileged with
// on-unreadable=warn lets unreadable scopes be omitted rather than failing.
func TestDescribeEmitsJSONDocument(t *testing.T) {
	r := run(t, "describe", "on-unreadable=warn")
	if r.exitCode != 0 {
		t.Fatalf("describe (warn): want exit 0, got %d (stderr=%q)", r.exitCode, r.stderr)
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal([]byte(r.stdout), &doc); err != nil {
		t.Fatalf("describe (warn): stdout is not valid JSON: %v\nstdout=%q", err, r.stdout)
	}
	meta, ok := doc["meta"]
	if !ok {
		t.Fatalf("describe (warn): output has no meta section: %q", r.stdout)
	}
	var m struct {
		FormatVersion int `json:"format_version"`
	}
	if err := json.Unmarshal(meta, &m); err != nil {
		t.Fatalf("describe (warn): meta not parseable: %v", err)
	}
	if m.FormatVersion != 1 {
		t.Errorf("describe (warn): meta.format_version want 1, got %d", m.FormatVersion)
	}
}

// EXAMPLE: describe_out_extension_yaml — out=...yaml writes a YAML document
// (resolve-format selects yaml from the extension when no format= is given).
func TestDescribeOutExtensionYAML(t *testing.T) {
	out := filepath.Join(t.TempDir(), "state.yaml")
	r := run(t, "describe", "out="+out, "on-unreadable=warn")
	if r.exitCode != 0 {
		t.Fatalf("describe out=...yaml: want exit 0, got %d (stderr=%q)", r.exitCode, r.stderr)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("describe out=...yaml: output file not written: %v", err)
	}
	// A YAML document must not begin with '{' (which would indicate JSON).
	trimmed := strings.TrimLeft(string(data), " \t\r\n")
	if strings.HasPrefix(trimmed, "{") {
		t.Errorf("describe out=...yaml: expected YAML, but file begins with '{' (JSON): %q", trimmed[:min(40, len(trimmed))])
	}
}

// EXAMPLE: describe_out_extension_json — out=...json writes JSON.
func TestDescribeOutExtensionJSON(t *testing.T) {
	out := filepath.Join(t.TempDir(), "state.json")
	r := run(t, "describe", "out="+out, "on-unreadable=warn")
	if r.exitCode != 0 {
		t.Fatalf("describe out=...json: want exit 0, got %d (stderr=%q)", r.exitCode, r.stderr)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("describe out=...json: output file not written: %v", err)
	}
	if json.Unmarshal(data, &map[string]json.RawMessage{}) != nil {
		t.Errorf("describe out=...json: file is not valid JSON: %q", string(data))
	}
}

// EXAMPLE: describe_format_overrides_extension — format=json with out=...yaml
// writes JSON because the explicit option wins over the extension.
func TestDescribeFormatOverridesExtension(t *testing.T) {
	out := filepath.Join(t.TempDir(), "state.yaml")
	r := run(t, "describe", "format=json", "out="+out, "on-unreadable=warn")
	if r.exitCode != 0 {
		t.Fatalf("describe format=json out=...yaml: want exit 0, got %d (stderr=%q)", r.exitCode, r.stderr)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("describe format=json out=...yaml: output file not written: %v", err)
	}
	if json.Unmarshal(data, &map[string]json.RawMessage{}) != nil {
		t.Errorf("describe format=json out=...yaml: file should be JSON, got %q", string(data))
	}
}

// EXAMPLE: describe_format_yaml — describe format=yaml writes a YAML document to
// stdout (same data model as JSON, not Machinery-compatible).
func TestDescribeFormatYAMLStdout(t *testing.T) {
	r := run(t, "describe", "format=yaml", "on-unreadable=warn")
	if r.exitCode != 0 {
		t.Fatalf("describe format=yaml: want exit 0, got %d (stderr=%q)", r.exitCode, r.stderr)
	}
	trimmed := strings.TrimLeft(r.stdout, " \t\r\n")
	if strings.HasPrefix(trimmed, "{") {
		t.Errorf("describe format=yaml: expected YAML on stdout, got JSON-looking output: %q", trimmed[:min(40, len(trimmed))])
	}
}

// ===========================================================================
// diff verb
// ===========================================================================

// EXAMPLE: diff_manifest_unreadable
func TestDiffManifestUnreadable(t *testing.T) {
	r := run(t, "diff", "manifest-path=/nonexistent-zd-manifest.json")
	if r.exitCode != 2 {
		t.Fatalf("diff unreadable manifest: want exit 2, got %d (stdout=%q stderr=%q)", r.exitCode, r.stdout, r.stderr)
	}
	if !strings.Contains(strings.ToLower(r.stderr), "invocation") {
		t.Errorf("diff unreadable manifest: stderr should carry domain=invocation, got %q", r.stderr)
	}
}

// EXAMPLE: apply_manifest_unreadable (verifiable without privilege because the
// read error precedes any transaction or privilege requirement).
func TestApplyManifestUnreadable(t *testing.T) {
	r := run(t, "apply", "manifest-path=/nonexistent-zd-manifest.json")
	if r.exitCode != 2 {
		t.Fatalf("apply unreadable manifest: want exit 2, got %d (stdout=%q stderr=%q)", r.exitCode, r.stdout, r.stderr)
	}
	if !strings.Contains(strings.ToLower(r.stderr), "invocation") {
		t.Errorf("apply unreadable manifest: stderr should carry domain=invocation, got %q", r.stderr)
	}
}

// EXAMPLE: apply_manifest_invalid — meta.format_version=2 fails schema validation.
// The manifest is loaded and validated before any transaction is opened, so this
// path is verifiable without privilege: exit 1, domain=manifest.
func TestApplyManifestInvalid(t *testing.T) {
	p := writeTemp(t, "bad.json", manifestFormatVersion2JSON)
	r := run(t, "apply", "manifest-path="+p)
	if r.exitCode != 1 {
		t.Fatalf("apply invalid manifest: want exit 1, got %d (stdout=%q stderr=%q)", r.exitCode, r.stdout, r.stderr)
	}
	if !strings.Contains(strings.ToLower(r.stderr), "manifest") {
		t.Errorf("apply invalid manifest: stderr should carry domain=manifest, got %q", r.stderr)
	}
}

// EXAMPLE: diff_prints_plan — with no applied record, every desired element is an
// addition. The plan must list the package to install and the files to write.
func TestDiffPrintsPlan(t *testing.T) {
	p := writeTemp(t, "desired.json", validManifestJSON)
	r := run(t, "diff", "manifest-path="+p, "on-unreadable=warn")
	if r.exitCode != 0 {
		t.Fatalf("diff plan: want exit 0, got %d (stderr=%q)", r.exitCode, r.stderr)
	}
	if !strings.Contains(r.stdout, "nginx") {
		t.Errorf("diff plan: stdout should mention nginx (packages to install), got %q", r.stdout)
	}
}

// EXAMPLE: yaml_manifest_accepted — a YAML manifest is parsed and a plan computed,
// exit 0.
func TestDiffYAMLManifestAccepted(t *testing.T) {
	p := writeTemp(t, "desired.yaml", validManifestYAML)
	r := run(t, "diff", "manifest-path="+p, "on-unreadable=warn")
	if r.exitCode != 0 {
		t.Fatalf("diff yaml manifest: want exit 0, got %d (stdout=%q stderr=%q)", r.exitCode, r.stdout, r.stderr)
	}
	if !strings.Contains(r.stdout, "nginx") {
		t.Errorf("diff yaml manifest: plan should mention nginx, got %q", r.stdout)
	}
}

// ===========================================================================
// verify verb
// ===========================================================================

// EXAMPLE: verify_no_applied_record — with no applied record present, verify
// emits "no declaration applied" with domain=invocation and exits 2.
//
// applied-root is pointed at an empty directory so no applied.json exists.
func TestVerifyNoAppliedRecord(t *testing.T) {
	emptyRoot := t.TempDir()
	r := run(t, "verify", "applied-root="+emptyRoot, "on-unreadable=warn")
	if r.exitCode != 2 {
		t.Fatalf("verify no applied record: want exit 2, got %d (stdout=%q stderr=%q)", r.exitCode, r.stdout, r.stderr)
	}
	if !strings.Contains(r.stderr, "no declaration applied") {
		t.Errorf("verify no applied record: stderr should contain 'no declaration applied', got %q", r.stderr)
	}
}

// EXAMPLE: verify_malformed_state_dump — a malformed state dump yields exit 2,
// domain=invocation. An applied record must exist for verify to proceed past
// step 1, so we lay one down under applied-root.
func TestVerifyMalformedStateDump(t *testing.T) {
	root := t.TempDir()
	layAppliedRecord(t, root, validAppliedRecordJSON)
	bad := writeTemp(t, "broken.json", "{ this is not valid json")
	r := run(t, "verify", "applied-root="+root, "state-path="+bad)
	if r.exitCode != 2 {
		t.Fatalf("verify malformed dump: want exit 2, got %d (stdout=%q stderr=%q)", r.exitCode, r.stdout, r.stderr)
	}
	if !strings.Contains(strings.ToLower(r.stderr), "invocation") {
		t.Errorf("verify malformed dump: stderr should carry domain=invocation, got %q", r.stderr)
	}
}

// EXAMPLE: verify_clean — the supplied state dump equals the applied record, so
// verify reports a match and exits 0.
func TestVerifyCleanWithStateDump(t *testing.T) {
	root := t.TempDir()
	layAppliedRecord(t, root, validAppliedRecordJSON)
	state := writeTemp(t, "state.json", actualStateMatchingJSON)
	r := run(t, "verify", "applied-root="+root, "state-path="+state)
	if r.exitCode != 0 {
		t.Fatalf("verify clean: want exit 0, got %d (stdout=%q stderr=%q)", r.exitCode, r.stdout, r.stderr)
	}
	if !strings.Contains(r.stdout, "system matches declaration") {
		t.Errorf("verify clean: stdout should contain 'system matches declaration', got %q", r.stdout)
	}
}

// EXAMPLE: verify_against_external_state_dump — the dump diverges from the applied
// record in one declared service state, so verify reports a units diagnostic and
// exits 1.
func TestVerifyAgainstExternalStateDumpDrift(t *testing.T) {
	root := t.TempDir()
	layAppliedRecord(t, root, validAppliedRecordJSON)
	state := writeTemp(t, "state.json", actualStateServiceDriftJSON)
	r := run(t, "verify", "applied-root="+root, "state-path="+state)
	if r.exitCode != 1 {
		t.Fatalf("verify service drift: want exit 1, got %d (stdout=%q stderr=%q)", r.exitCode, r.stdout, r.stderr)
	}
	if !strings.Contains(strings.ToLower(r.stderr), "units") {
		t.Errorf("verify service drift: stderr should carry domain=units, got %q", r.stderr)
	}
}

// EXAMPLE: verify_detects_drift — a declared file's actual content differs (sha256
// differs in the supplied dump), so verify reports a files diagnostic naming the
// file and exits 1.
func TestVerifyDetectsFileDrift(t *testing.T) {
	root := t.TempDir()
	layAppliedRecord(t, root, validAppliedRecordJSON)
	state := writeTemp(t, "state.json", actualStateFileDriftJSON)
	r := run(t, "verify", "applied-root="+root, "state-path="+state)
	if r.exitCode != 1 {
		t.Fatalf("verify file drift: want exit 1, got %d (stdout=%q stderr=%q)", r.exitCode, r.stdout, r.stderr)
	}
	if !strings.Contains(strings.ToLower(r.stderr), "files") {
		t.Errorf("verify file drift: stderr should carry domain=files, got %q", r.stderr)
	}
	if !strings.Contains(r.stderr, "/etc/foo.conf") {
		t.Errorf("verify file drift: stderr should name /etc/foo.conf, got %q", r.stderr)
	}
}

// EXAMPLE: verify_state_path_extension_yaml — a YAML state dump matching the
// applied record is accepted by extension (resolve-format), reporting a match.
func TestVerifyStatePathExtensionYAML(t *testing.T) {
	root := t.TempDir()
	layAppliedRecord(t, root, validAppliedRecordJSON)
	state := writeTemp(t, "state.yaml", actualStateMatchingYAML)
	r := run(t, "verify", "applied-root="+root, "state-path="+state)
	if r.exitCode != 0 {
		t.Fatalf("verify yaml state dump: want exit 0, got %d (stdout=%q stderr=%q)", r.exitCode, r.stdout, r.stderr)
	}
	if !strings.Contains(r.stdout, "system matches declaration") {
		t.Errorf("verify yaml state dump: stdout should contain match line, got %q", r.stdout)
	}
}

// ===========================================================================
// status verb
// ===========================================================================

// EXAMPLE: status_no_declaration — no applied record: print "no declaration
// applied" and exit 0.
func TestStatusNoDeclaration(t *testing.T) {
	emptyRoot := t.TempDir()
	r := run(t, "status", "applied-root="+emptyRoot, "on-unreadable=warn")
	if r.exitCode != 0 {
		t.Fatalf("status no declaration: want exit 0, got %d (stderr=%q)", r.exitCode, r.stderr)
	}
	if !strings.Contains(r.stdout, "no declaration applied") {
		t.Errorf("status no declaration: stdout should contain 'no declaration applied', got %q", r.stdout)
	}
}

// EXAMPLE: status_reports_generation — an applied record exists; status prints the
// desired_sha256, the resolved package count, and a single drift-summary line.
func TestStatusReportsGeneration(t *testing.T) {
	root := t.TempDir()
	layAppliedRecord(t, root, validAppliedRecordJSON)
	r := run(t, "status", "applied-root="+root, "on-unreadable=warn")
	if r.exitCode != 0 {
		t.Fatalf("status reports generation: want exit 0, got %d (stderr=%q)", r.exitCode, r.stderr)
	}
	// desired_sha256 of the applied record is known: it appears verbatim.
	if !strings.Contains(r.stdout, appliedRecordDesiredSHA) {
		t.Errorf("status reports generation: stdout should contain the desired_sha256 %q, got %q",
			appliedRecordDesiredSHA, r.stdout)
	}
}

// ===========================================================================
// Idempotence / round-trip (INVARIANTS)
// ===========================================================================

// EXAMPLE: yaml_format_identity_stable — the same manifest expressed in JSON and
// YAML must yield the same intent diff (the format-independent, manifest-derived
// part of the plan), demonstrating that desired identity does not depend on the
// serialisation. The live-drift section of the plan reflects the running system
// at the moment of each invocation and is therefore not format-determined; the
// spec's invariant is about manifest identity, not about live drift being equal
// between two separate process runs. We compare the intent-diff portion (the
// plan up to the "current drift:" marker), which is what the EXAMPLE constrains.
func TestJSONAndYAMLProduceSameIntentDiff(t *testing.T) {
	pj := writeTemp(t, "desired.json", validManifestJSON)
	py := writeTemp(t, "desired.yaml", validManifestYAML)
	rj := run(t, "diff", "manifest-path="+pj, "on-unreadable=warn")
	ry := run(t, "diff", "manifest-path="+py, "on-unreadable=warn")
	if rj.exitCode != 0 || ry.exitCode != 0 {
		t.Fatalf("diff json/yaml: want both exit 0, got %d and %d", rj.exitCode, ry.exitCode)
	}
	intentJSON := intentPortion(rj.stdout)
	intentYAML := intentPortion(ry.stdout)
	if intentJSON != intentYAML {
		t.Errorf("diff json vs yaml: intent diffs differ.\nJSON:\n%s\nYAML:\n%s", intentJSON, intentYAML)
	}
}

// intentPortion returns the part of a diff plan that is determined by the
// manifest (everything before the live-drift section).
func intentPortion(plan string) string {
	if i := strings.Index(plan, "current drift:"); i >= 0 {
		return plan[:i]
	}
	return plan
}

// EXAMPLE: describe_bootstraps_desired_manifest — describe output is accepted
// unchanged by load-desired-manifest as a starting desired manifest. We capture
// describe output, feed it back to diff against the same (no applied record)
// state, and require exit 0.
func TestDescribeOutputAcceptedAsDesiredManifest(t *testing.T) {
	out := filepath.Join(t.TempDir(), "desired.json")
	rd := run(t, "describe", "out="+out, "on-unreadable=warn")
	if rd.exitCode != 0 {
		t.Fatalf("describe for bootstrap: want exit 0, got %d (stderr=%q)", rd.exitCode, rd.stderr)
	}
	rdiff := run(t, "diff", "manifest-path="+out, "on-unreadable=warn")
	if rdiff.exitCode != 0 {
		t.Fatalf("diff against bootstrapped manifest: want exit 0, got %d (stdout=%q stderr=%q)",
			rdiff.exitCode, rdiff.stdout, rdiff.stderr)
	}
}

// ===========================================================================
// YAML safe profile (INVARIANT: unsafe YAML rejected)
// ===========================================================================

// EXAMPLE: yaml_unsafe_rejected — a YAML manifest using a multi-document stream
// (a disabled safe-profile feature) is rejected with a manifest error; the verb
// (here diff, which does not need privilege) exits 1.
func TestYAMLUnsafeMultiDocRejected(t *testing.T) {
	// Multi-document stream: the safe profile permits a single document only.
	multi := validManifestYAML + "\n---\n" + validManifestYAML
	p := writeTemp(t, "evil.yaml", multi)
	r := run(t, "diff", "manifest-path="+p, "on-unreadable=warn")
	if r.exitCode != 1 {
		t.Fatalf("unsafe yaml (multi-doc): want exit 1, got %d (stdout=%q stderr=%q)", r.exitCode, r.stdout, r.stderr)
	}
	if !strings.Contains(strings.ToLower(r.stderr), "manifest") {
		t.Errorf("unsafe yaml (multi-doc): stderr should carry domain=manifest, got %q", r.stderr)
	}
}

// ===========================================================================
// Read-only invariant: diff/verify/status/describe never modify input files.
// ===========================================================================

// INVARIANT: the tool must not modify its input files (FILE-MODIFICATION).
func TestDiffDoesNotModifyManifest(t *testing.T) {
	p := writeTemp(t, "desired.json", validManifestJSON)
	before, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	_ = run(t, "diff", "manifest-path="+p, "on-unreadable=warn")
	after, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Errorf("diff modified its input manifest; before=%q after=%q", before, after)
	}
}

// ===========================================================================
// Test fixtures for applied record / actual state dumps.
// ===========================================================================

// appliedRecordDesiredSHA is the desired_sha256 stamped into validAppliedRecordJSON.
// (An arbitrary but fixed valid Sha256; the test asserts status echoes it.)
const appliedRecordDesiredSHA = "1111111111111111111111111111111111111111111111111111111111111111"

// validAppliedRecordJSON is a structurally complete applied record: a Manifest
// with the packages scope fully resolved (version/release/arch populated) and
// meta.desired_sha256 set.
const validAppliedRecordJSON = `{
  "meta": {
    "format_version": 1,
    "generator": "test",
    "created_at": "2026-05-29T08:30:00Z",
    "desired_sha256": "1111111111111111111111111111111111111111111111111111111111111111"
  },
  "packages": {
    "_attributes": { "package_system": "rpm" },
    "_elements": [
      { "name": "nginx", "version": "1.25.3", "release": "1.1", "arch": "x86_64" }
    ]
  },
  "services": {
    "_attributes": { "init_system": "systemd" },
    "_elements": [
      { "name": "nginx.service", "state": "enabled" }
    ]
  },
  "config_files": {
    "_attributes": null,
    "_elements": [
      {
        "name": "/etc/foo.conf",
        "type": "file",
        "mode": "0644",
        "user": "root",
        "group": "root",
        "sha256": "aaaa000000000000000000000000000000000000000000000000000000000000",
        "content_ref": "",
        "package_name": ""
      }
    ]
  }
}`

// actualStateMatchingJSON is an actual-state Manifest that equals the applied
// record on every identity field (same packages, same service state, same file
// sha256), so compute-drift returns empty.
const actualStateMatchingJSON = `{
  "meta": { "format_version": 1, "generator": "test", "created_at": "2026-05-29T08:30:00Z", "desired_sha256": "" },
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
      { "name": "/etc/foo.conf", "type": "file", "mode": "0644", "user": "root", "group": "root",
        "sha256": "aaaa000000000000000000000000000000000000000000000000000000000000",
        "content_ref": "", "package_name": "" }
    ]
  }
}`

// actualStateMatchingYAML is the YAML serialisation of actualStateMatchingJSON.
const actualStateMatchingYAML = `meta:
  format_version: 1
  generator: "test"
  created_at: "2026-05-29T08:30:00Z"
  desired_sha256: ""
packages:
  _attributes:
    package_system: "rpm"
  _elements:
    - name: "nginx"
      version: "1.25.3"
      release: "1.1"
      arch: "x86_64"
services:
  _attributes:
    init_system: "systemd"
  _elements:
    - name: "nginx.service"
      state: "enabled"
config_files:
  _attributes: null
  _elements:
    - name: "/etc/foo.conf"
      type: "file"
      mode: "0644"
      user: "root"
      group: "root"
      sha256: "aaaa000000000000000000000000000000000000000000000000000000000000"
      content_ref: ""
      package_name: ""
`

// actualStateServiceDriftJSON diverges from the applied record only in the
// nginx.service state (disabled vs enabled).
const actualStateServiceDriftJSON = `{
  "meta": { "format_version": 1, "generator": "test", "created_at": "2026-05-29T08:30:00Z", "desired_sha256": "" },
  "packages": {
    "_attributes": { "package_system": "rpm" },
    "_elements": [ { "name": "nginx", "version": "1.25.3", "release": "1.1", "arch": "x86_64" } ]
  },
  "services": {
    "_attributes": { "init_system": "systemd" },
    "_elements": [ { "name": "nginx.service", "state": "disabled" } ]
  },
  "config_files": {
    "_attributes": null,
    "_elements": [
      { "name": "/etc/foo.conf", "type": "file", "mode": "0644", "user": "root", "group": "root",
        "sha256": "aaaa000000000000000000000000000000000000000000000000000000000000",
        "content_ref": "", "package_name": "" }
    ]
  }
}`

// actualStateFileDriftJSON diverges from the applied record only in the
// /etc/foo.conf sha256 (content changed).
const actualStateFileDriftJSON = `{
  "meta": { "format_version": 1, "generator": "test", "created_at": "2026-05-29T08:30:00Z", "desired_sha256": "" },
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
      { "name": "/etc/foo.conf", "type": "file", "mode": "0644", "user": "root", "group": "root",
        "sha256": "bbbb000000000000000000000000000000000000000000000000000000000000",
        "content_ref": "", "package_name": "" }
    ]
  }
}`

// layAppliedRecord writes an applied record under <root>/usr/lib/zypper-declarative/applied.json.
func layAppliedRecord(t *testing.T, root, content string) {
	t.Helper()
	dir := filepath.Join(root, "usr", "lib", "zypper-declarative")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("creating applied-record dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "applied.json"), []byte(content), 0644); err != nil {
		t.Fatalf("writing applied record: %v", err)
	}
}
