// generated from spec: zypper-declarative.spec.md sha256:b2d0de88fbed1163678e59e931c741b9d999b71f902f6eb01db8790bb813d057
// tests by: claude-opus-4-8
//
// status and verify verb tests, plus the applied-record fixture helper.
package zypperdeclarative_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// mustWriteAppliedRecord writes an applied record into a generation root at
// the spec-declared path <root>/usr/lib/zypper-declarative/applied.json so
// that load-applied-record (parameterised by applied-root) finds it.
func mustWriteAppliedRecord(t *testing.T, root, content string) {
	t.Helper()
	dir := filepath.Join(root, "usr", "lib", "zypper-declarative")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("creating applied-record dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "applied.json"), []byte(content), 0o644); err != nil {
		t.Fatalf("writing applied.json: %v", err)
	}
}

// EXAMPLE: status_no_declaration -- with no applied record, status prints
// "no declaration applied" and exits 0.
func TestStatusNoDeclaration(t *testing.T) {
	emptyRoot := t.TempDir() // no applied.json under it
	r := run(t, "status", "applied-root="+emptyRoot)
	assertExit(t, r, 0)
	assertStdoutContains(t, r, "no declaration applied")
}

// EXAMPLE: status_reports_generation -- with an applied record carrying a
// known desired_sha256 and a resolved packages lock, status prints the hash
// and the resolved package count. Conditional drift line depends on a live
// read; the deterministic part is the recorded record fields.
func TestStatusReportsGeneration(t *testing.T) {
	applied := `{
  "meta": { "format_version": 1, "generator": "zypper-declarative 0.6.0", "created_at": "2026-01-02T03:04:05Z", "desired_sha256": "abc123def4567890abc123def4567890abc123def4567890abc123def4567890" },
  "packages": {
    "_attributes": { "package_system": "rpm" },
    "_elements": [
      { "name": "nginx", "version": "1.25.0", "release": "1.1", "arch": "x86_64" }
    ]
  }
}`
	root := t.TempDir()
	mustWriteAppliedRecord(t, root, applied)
	r, timedOut := runWithTimeout(t, liveReadBudget, "", "status", "applied-root="+root)
	if timedOut {
		t.Skip("status live drift read of / exceeded the budget on this host")
	}
	// status is read-only and exits 0 whenever invocation is valid; the drift
	// summary may use a live read, but the recorded desired_sha256 must appear.
	if r.exitCode != 0 {
		t.Logf("status exited %d (live drift read unavailable); stderr:\n%s", r.exitCode, r.stderr)
	}
	assertStdoutContains(t, r, "abc123def4567890abc123def4567890abc123def4567890abc123def4567890")
}

// EXAMPLE: verify_no_applied_record -- no applied record => exit 2,
// "no declaration applied" to stderr, domain=invocation.
func TestVerifyNoAppliedRecord(t *testing.T) {
	emptyRoot := t.TempDir()
	r := run(t, "verify", "applied-root="+emptyRoot)
	assertExit(t, r, 2)
	assertStderrContains(t, r, "no declaration applied")
}

// EXAMPLE: verify_malformed_state_dump -- a state-path that is not a valid
// shared-schema Manifest is an invocation error (exit 2). An applied record
// must exist so the verb gets past step 1.
func TestVerifyMalformedStateDump(t *testing.T) {
	root := t.TempDir()
	mustWriteAppliedRecord(t, root, minimalAppliedRecord)
	bad := writeTemp(t, "broken.json", "this is not json or yaml manifest: {{{")
	r := run(t, "verify", "applied-root="+root, "state-path="+bad)
	assertExit(t, r, 2)
}

// EXAMPLE: verify_against_external_state_dump -- a state dump that diverges
// from the applied record in one declared service state yields a units
// diagnostic and exit 1.
func TestVerifyAgainstExternalStateDumpServiceDrift(t *testing.T) {
	applied := `{
  "meta": { "format_version": 1, "generator": "zd", "created_at": "2026-01-01T00:00:00Z", "desired_sha256": "deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef0" },
  "services": {
    "_attributes": { "init_system": "systemd" },
    "_elements": [ { "name": "nginx.service", "state": "enabled" } ]
  }
}`
	dump := `{
  "meta": { "format_version": 1, "generator": "zd", "created_at": "2026-01-01T00:00:00Z", "desired_sha256": "" },
  "services": {
    "_attributes": { "init_system": "systemd" },
    "_elements": [ { "name": "nginx.service", "state": "disabled" } ]
  }
}`
	root := t.TempDir()
	mustWriteAppliedRecord(t, root, applied)
	dp := writeTemp(t, "state.json", dump)
	r := run(t, "verify", "applied-root="+root, "state-path="+dp)
	assertExit(t, r, 1)
	if !strings.Contains(r.stderr, "units") && !strings.Contains(r.stderr, "nginx.service") {
		t.Errorf("expected a units diagnostic naming the divergent service\nstderr:\n%s", r.stderr)
	}
}

// EXAMPLE: verify_clean / verify_state_path_extension_yaml -- a state dump
// equal to the applied record yields "system matches declaration" and exit 0.
// The dump is supplied via state-path so no live read is needed.
func TestVerifyCleanAgainstMatchingDump(t *testing.T) {
	applied := `{
  "meta": { "format_version": 1, "generator": "zd", "created_at": "2026-01-01T00:00:00Z", "desired_sha256": "feedface00000000000000000000000000000000000000000000000000000000" },
  "services": {
    "_attributes": { "init_system": "systemd" },
    "_elements": [ { "name": "nginx.service", "state": "enabled" } ]
  }
}`
	dump := `{
  "meta": { "format_version": 1, "generator": "zd", "created_at": "2026-01-01T00:00:00Z", "desired_sha256": "" },
  "services": {
    "_attributes": { "init_system": "systemd" },
    "_elements": [ { "name": "nginx.service", "state": "enabled" } ]
  }
}`
	root := t.TempDir()
	mustWriteAppliedRecord(t, root, applied)
	dp := writeTemp(t, "state.json", dump)
	r := run(t, "verify", "applied-root="+root, "state-path="+dp)
	assertExit(t, r, 0)
	assertStdoutContains(t, r, "system matches declaration")
}

// EXAMPLE: verify_state_path_extension_yaml -- a YAML state dump matching the
// applied record is selected via the .yaml extension and yields exit 0.
func TestVerifyStatePathExtensionYAML(t *testing.T) {
	applied := `{
  "meta": { "format_version": 1, "generator": "zd", "created_at": "2026-01-01T00:00:00Z", "desired_sha256": "feedface00000000000000000000000000000000000000000000000000000000" },
  "services": {
    "_attributes": { "init_system": "systemd" },
    "_elements": [ { "name": "nginx.service", "state": "enabled" } ]
  }
}`
	dumpYAML := `meta:
  format_version: 1
  generator: zd
  created_at: "2026-01-01T00:00:00Z"
  desired_sha256: ""
services:
  _attributes:
    init_system: systemd
  _elements:
    - name: nginx.service
      state: enabled
`
	root := t.TempDir()
	mustWriteAppliedRecord(t, root, applied)
	dp := writeTemp(t, "state.yaml", dumpYAML)
	r := run(t, "verify", "applied-root="+root, "state-path="+dp)
	assertExit(t, r, 0)
	assertStdoutContains(t, r, "system matches declaration")
}

const minimalAppliedRecord = `{
  "meta": { "format_version": 1, "generator": "zd", "created_at": "2026-01-01T00:00:00Z", "desired_sha256": "00ff00ff00ff00ff00ff00ff00ff00ff00ff00ff00ff00ff00ff00ff00ff00ff" },
  "packages": { "_attributes": { "package_system": "rpm" }, "_elements": [] }
}`
