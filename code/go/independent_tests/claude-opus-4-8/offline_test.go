// generated from spec: zypper-declarative.spec.md sha256:f2cc80627e483a48bb8411d297711bc5f6c6e74c28dbf0dafc8fe7bd8817251e
// tests by: claude-opus-4-8
//
// Offline (two-file) black-box tests: diff and verify with both manifest-path
// and state-path supplied are pure functions of the two files, reading neither
// the live system nor any applied record. These are deterministically testable
// without root or a live host.
package independent_tests

import (
	"strings"
	"testing"
)

// A reference manifest (baseline) declaring two /etc files and one service.
const baselineJSON = `{
  "meta": { "format_version": 1, "generator": "test", "created_at": "2026-05-29T08:30:00Z", "desired_sha256": "" },
  "services": {
    "_attributes": { "init_system": "systemd" },
    "_elements": [ { "name": "nginx.service", "state": "enabled" } ]
  },
  "config_files": {
    "_attributes": null,
    "_elements": [
      { "name": "/etc/foo.conf", "type": "file", "mode": "0644", "user": "root", "group": "root",
        "sha256": "1111111111111111111111111111111111111111111111111111111111111111",
        "target": "", "content_ref": "", "package_name": "" },
      { "name": "/etc/bar.conf", "type": "file", "mode": "0644", "user": "root", "group": "root",
        "sha256": "2222222222222222222222222222222222222222222222222222222222222222",
        "target": "", "content_ref": "", "package_name": "" }
    ]
  }
}`

// An actual-state dump matching the baseline exactly (both files, same sha256,
// same service state). Used for the "clean" offline verify.
const matchingStateJSON = `{
  "meta": { "format_version": 1, "generator": "describe", "created_at": "2026-05-29T09:00:00Z", "desired_sha256": "" },
  "services": {
    "_attributes": { "init_system": "systemd" },
    "_elements": [ { "name": "nginx.service", "state": "enabled" } ]
  },
  "config_files": {
    "_attributes": null,
    "_elements": [
      { "name": "/etc/foo.conf", "type": "file", "mode": "0644", "user": "root", "group": "root",
        "sha256": "1111111111111111111111111111111111111111111111111111111111111111",
        "target": "", "content_ref": "", "package_name": "" },
      { "name": "/etc/bar.conf", "type": "file", "mode": "0644", "user": "root", "group": "root",
        "sha256": "2222222222222222222222222222222222222222222222222222222222222222",
        "target": "", "content_ref": "", "package_name": "" }
    ]
  }
}`

// An actual-state dump where the declared service nginx.service is disabled
// instead of enabled -> a units divergence.
const driftServiceStateJSON = `{
  "meta": { "format_version": 1, "generator": "describe", "created_at": "2026-05-29T09:00:00Z", "desired_sha256": "" },
  "services": {
    "_attributes": { "init_system": "systemd" },
    "_elements": [ { "name": "nginx.service", "state": "disabled" } ]
  },
  "config_files": {
    "_attributes": null,
    "_elements": [
      { "name": "/etc/foo.conf", "type": "file", "mode": "0644", "user": "root", "group": "root",
        "sha256": "1111111111111111111111111111111111111111111111111111111111111111",
        "target": "", "content_ref": "", "package_name": "" },
      { "name": "/etc/bar.conf", "type": "file", "mode": "0644", "user": "root", "group": "root",
        "sha256": "2222222222222222222222222222222222222222222222222222222222222222",
        "target": "", "content_ref": "", "package_name": "" }
    ]
  }
}`

// An actual-state dump where /etc/foo.conf content differs (changed sha256).
const driftFileJSON = `{
  "meta": { "format_version": 1, "generator": "describe", "created_at": "2026-05-29T09:00:00Z", "desired_sha256": "" },
  "services": {
    "_attributes": { "init_system": "systemd" },
    "_elements": [ { "name": "nginx.service", "state": "enabled" } ]
  },
  "config_files": {
    "_attributes": null,
    "_elements": [
      { "name": "/etc/foo.conf", "type": "file", "mode": "0644", "user": "root", "group": "root",
        "sha256": "deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef0",
        "target": "", "content_ref": "", "package_name": "" },
      { "name": "/etc/bar.conf", "type": "file", "mode": "0644", "user": "root", "group": "root",
        "sha256": "2222222222222222222222222222222222222222222222222222222222222222",
        "target": "", "content_ref": "", "package_name": "" }
    ]
  }
}`

// ----------------------------------------------------------------------------
// EXAMPLE: verify_offline_manifest_and_state (+ verify_offline_no_applied_record_ok)
// ----------------------------------------------------------------------------

func TestVerifyOfflineClean(t *testing.T) {
	ref := writeTemp(t, "baseline.json", baselineJSON)
	state := writeTemp(t, "after.json", matchingStateJSON)
	dir := t.TempDir() // applied-root with no applied.json: proves reference comes from manifest-path
	stdout, stderr, exit := run(t, "verify", "manifest-path="+ref, "state-path="+state, "applied-root="+dir)
	if exit != 0 {
		t.Fatalf("offline verify clean: exit=%d, want 0\nstdout=%s\nstderr=%s", exit, stdout, stderr)
	}
	if !strings.Contains(stdout, "system matches declaration") {
		t.Errorf("offline verify clean: stdout=%q, want 'system matches declaration'", stdout)
	}
	// verify_offline_no_applied_record_ok: must NOT emit "no declaration applied".
	if strings.Contains(stdout, "no declaration applied") || strings.Contains(stderr, "no declaration applied") {
		t.Errorf("offline verify with manifest-path must not report 'no declaration applied'")
	}
}

// EXAMPLE: verify_against_external_state_dump — service state divergence -> domain=units, exit 1.
func TestVerifyOfflineServiceDrift(t *testing.T) {
	ref := writeTemp(t, "baseline.json", baselineJSON)
	state := writeTemp(t, "after.json", driftServiceStateJSON)
	_, stderr, exit := run(t, "verify", "manifest-path="+ref, "state-path="+state)
	if exit != 1 {
		t.Fatalf("offline verify service drift: exit=%d, want 1\nstderr=%s", exit, stderr)
	}
	if !strings.Contains(stderr, "units") {
		t.Errorf("offline verify service drift: stderr missing domain=units: %q", stderr)
	}
	if !strings.Contains(stderr, "nginx.service") {
		t.Errorf("offline verify service drift: stderr should name divergent service: %q", stderr)
	}
}

// EXAMPLE: verify_detects_drift — declared file edited -> domain=files naming the file, exit 1.
func TestVerifyOfflineFileDrift(t *testing.T) {
	ref := writeTemp(t, "baseline.json", baselineJSON)
	state := writeTemp(t, "after.json", driftFileJSON)
	_, stderr, exit := run(t, "verify", "manifest-path="+ref, "state-path="+state)
	if exit != 1 {
		t.Fatalf("offline verify file drift: exit=%d, want 1\nstderr=%s", exit, stderr)
	}
	if !strings.Contains(stderr, "files") {
		t.Errorf("offline verify file drift: stderr missing domain=files: %q", stderr)
	}
	if !strings.Contains(stderr, "/etc/foo.conf") {
		t.Errorf("offline verify file drift: stderr should name /etc/foo.conf: %q", stderr)
	}
}

// ----------------------------------------------------------------------------
// EXAMPLE: diff_offline_two_files — plan computed from the two files; exit 0.
// EXAMPLE: intent_diff_yields_deletion — desired drops /etc/bar.conf -> files to delete.
// ----------------------------------------------------------------------------

func TestDiffOfflineTwoFiles(t *testing.T) {
	// desired drops /etc/bar.conf and adds package nginx relative to applied/baseline.
	ref := writeTemp(t, "baseline.json", baselineJSON) // applied side via manifest? -> use applied-root
	state := writeTemp(t, "after.json", matchingStateJSON)
	// For diff the manifest-path is the DESIRED; we compare against applied via root.
	// Here we use desiredJSON (declares foo.conf only + nginx) and an applied-root
	// holding baseline as the applied record.
	dir := t.TempDir()
	writeAppliedRecord(t, dir, baselineJSON)
	desired := writeTemp(t, "desired.json", desiredJSON)
	stdout, stderr, exit := run(t, "diff", "manifest-path="+desired, "state-path="+state, "applied-root="+dir)
	if exit != 0 {
		t.Fatalf("diff offline: exit=%d, want 0\nstdout=%s\nstderr=%s", exit, stdout, stderr)
	}
	// nginx is in desired but not in applied baseline -> install.
	if !strings.Contains(stdout, "nginx") {
		t.Errorf("diff offline: stdout should list nginx to install: %q", stdout)
	}
	// /etc/bar.conf declared in applied baseline, dropped by desired -> delete.
	if !strings.Contains(stdout, "/etc/bar.conf") {
		t.Errorf("diff offline: stdout should list /etc/bar.conf to delete: %q", stdout)
	}
	_ = ref
}

// EXAMPLE: diff_prints_plan — adds nginx, drops /etc/bar.conf; lists both; exit 0.
func TestDiffPrintsPlan(t *testing.T) {
	dir := t.TempDir()
	writeAppliedRecord(t, dir, baselineJSON)
	state := writeTemp(t, "after.json", matchingStateJSON)
	desired := writeTemp(t, "desired.json", desiredJSON)
	stdout, _, exit := run(t, "diff", "manifest-path="+desired, "state-path="+state, "applied-root="+dir)
	if exit != 0 {
		t.Fatalf("diff plan: exit=%d, want 0", exit)
	}
	lower := strings.ToLower(stdout)
	if !strings.Contains(lower, "install") {
		t.Errorf("diff plan: stdout should describe packages to install: %q", stdout)
	}
	if !strings.Contains(lower, "delete") {
		t.Errorf("diff plan: stdout should describe files to delete: %q", stdout)
	}
}
