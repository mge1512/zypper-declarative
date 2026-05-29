// generated from spec: zypper-declarative.spec.md sha256:714e75ff672557d2c7344736a5f36b52afa37e0f565b07de83e1b18cc4492014
// tests by: claude-opus-4-8
//
// Tests for BEHAVIOR: verify. verify reads the applied record from the
// generation root (surfaced as applied-root=) and the actual state from a
// supplied dump (state-path=) or live. The supplied-dump path needs no
// privilege and is the primary cross-check of compute-drift.
package independent_tests

import (
	"strings"
	"testing"
)

// appliedRecordOneServiceOneFile declares one service and one file, fully
// resolved, so drift comparison against a dump is deterministic.
func appliedRecordOneServiceOneFile() string {
	return `{
  "meta": { "format_version": 1, "generator": "zypper-declarative 0.4.0", "created_at": "2026-05-29T08:30:00Z", "desired_sha256": "2222222222222222222222222222222222222222222222222222222222222222" },
  "packages": { "_attributes": { "package_system": "rpm" }, "_elements": [] },
  "services": {
    "_attributes": { "init_system": "systemd" },
    "_elements": [ { "name": "nginx.service", "state": "enabled" } ]
  },
  "config_files": {
    "_attributes": null,
    "_elements": [
      { "name": "/etc/foo.conf", "type": "file", "mode": "0644", "user": "root", "group": "root", "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "content_ref": "", "package_name": "" }
    ]
  }
}`
}

// EXAMPLE: verify_clean — the live system equals the applied record.
// Tested deterministically by supplying a state dump that matches the
// applied record exactly.
func TestVerifyClean(t *testing.T) {
	applied := writeAppliedRecord(t, appliedRecordOneServiceOneFile())
	dump := `{
  "meta": { "format_version": 1, "generator": "zypper-declarative 0.4.0", "created_at": "2026-05-29T09:00:00Z", "desired_sha256": "" },
  "packages": { "_attributes": { "package_system": "rpm" }, "_elements": [] },
  "services": {
    "_attributes": { "init_system": "systemd" },
    "_elements": [ { "name": "nginx.service", "state": "enabled" } ]
  },
  "config_files": {
    "_attributes": null,
    "_elements": [
      { "name": "/etc/foo.conf", "type": "file", "mode": "0644", "user": "root", "group": "root", "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "content_ref": "", "package_name": "" }
    ]
  }
}`
	sp := writeTempFile(t, "state.json", dump)
	r := run(t, "verify", "state-path="+sp, "applied-root="+applied)
	mustExit(t, r.exitCode, 0, "verify clean")
	mustContain(t, r.stdout, "system matches declaration", "verify clean stdout")
}

// EXAMPLE: verify_against_external_state_dump — dump diverges in one service
// state; expect domain=units diagnostic naming the service and exit 1.
func TestVerifyAgainstExternalStateDumpUnitDrift(t *testing.T) {
	applied := writeAppliedRecord(t, appliedRecordOneServiceOneFile())
	dump := `{
  "meta": { "format_version": 1, "generator": "zypper-declarative 0.4.0", "created_at": "2026-05-29T09:00:00Z", "desired_sha256": "" },
  "packages": { "_attributes": { "package_system": "rpm" }, "_elements": [] },
  "services": {
    "_attributes": { "init_system": "systemd" },
    "_elements": [ { "name": "nginx.service", "state": "disabled" } ]
  },
  "config_files": {
    "_attributes": null,
    "_elements": [
      { "name": "/etc/foo.conf", "type": "file", "mode": "0644", "user": "root", "group": "root", "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "content_ref": "", "package_name": "" }
    ]
  }
}`
	sp := writeTempFile(t, "state.json", dump)
	r := run(t, "verify", "state-path="+sp, "applied-root="+applied)
	mustExit(t, r.exitCode, 1, "verify unit drift")
	mustContain(t, r.stderr, "units", "verify unit drift domain=units")
	mustContain(t, r.stderr, "nginx.service", "verify unit drift names service")
}

// EXAMPLE: verify_detects_drift — declared file /etc/foo.conf edited; expect
// a diagnostic naming /etc/foo.conf with domain=files and exit 1.
func TestVerifyDetectsFileDrift(t *testing.T) {
	applied := writeAppliedRecord(t, appliedRecordOneServiceOneFile())
	dump := `{
  "meta": { "format_version": 1, "generator": "zypper-declarative 0.4.0", "created_at": "2026-05-29T09:00:00Z", "desired_sha256": "" },
  "packages": { "_attributes": { "package_system": "rpm" }, "_elements": [] },
  "services": {
    "_attributes": { "init_system": "systemd" },
    "_elements": [ { "name": "nginx.service", "state": "enabled" } ]
  },
  "config_files": {
    "_attributes": null,
    "_elements": [
      { "name": "/etc/foo.conf", "type": "file", "mode": "0644", "user": "root", "group": "root", "sha256": "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd", "content_ref": "", "package_name": "" }
    ]
  }
}`
	sp := writeTempFile(t, "state.json", dump)
	r := run(t, "verify", "state-path="+sp, "applied-root="+applied)
	mustExit(t, r.exitCode, 1, "verify file drift")
	mustContain(t, r.stderr, "/etc/foo.conf", "verify file drift names path")
	mustContain(t, r.stderr, "files", "verify file drift domain=files")
}

// EXAMPLE: verify_malformed_state_dump — state-path is not a valid Manifest.
func TestVerifyMalformedStateDump(t *testing.T) {
	applied := writeAppliedRecord(t, appliedRecordOneServiceOneFile())
	sp := writeTempFile(t, "broken.json", "{ this is not valid json ")
	r := run(t, "verify", "state-path="+sp, "applied-root="+applied)
	mustExit(t, r.exitCode, 2, "verify malformed dump")
	mustContain(t, r.stderr, "invocation", "verify malformed dump domain=invocation")
}

// EXAMPLE: verify_no_applied_record — no applied record present.
func TestVerifyNoAppliedRecord(t *testing.T) {
	emptyRoot := t.TempDir() // no applied.json under it
	r := run(t, "verify", "applied-root="+emptyRoot)
	mustExit(t, r.exitCode, 2, "verify no applied record")
	mustContain(t, strings.ToLower(r.stderr), "no declaration applied", "verify no applied record stderr")
	mustContain(t, r.stderr, "invocation", "verify no applied record domain=invocation")
}
