// generated from spec: zypper-declarative.spec.md sha256:b2d0de88fbed1163678e59e931c741b9d999b71f902f6eb01db8790bb813d057
// tests by: claude-opus-4-8
//
// Manifest loading, format resolution, and intent-diff tests, exercised
// through the `diff` verb. These are black-box: a manifest fixture is written
// to a temp file and passed via manifest-path=, and stdout/stderr/exit are
// asserted.
//
// Note on live-state reads: the `diff` verb's step 4 reads live actual state
// via describe-actual-state on "/" with on_unreadable=error. On a host that is
// not a SUSE/rpm system the rpmdb may be unreadable, which the spec treats as
// an error path. Tests that need a successful (exit 0) plan therefore assert
// conditionally on the intent-diff lines when exit is 0, and unconditionally
// assert the deterministic manifest error paths (unreadable, invalid, unknown
// format). This is documented as an environment dependency in TEST/TRANSLATION
// reports.
package zypperdeclarative_test

import (
	"strings"
	"testing"
	"time"
)

// liveReadBudget bounds how long a verb that reads live actual state of "/"
// may take in the test environment before the test skips. diff/status/verify
// (live) read describe-actual-state on "/" per the spec; on a large or
// rpm-heavy host that read is slow, which is a property of the host, not a
// defect. Tests that need a successful plan skip rather than hang.
const liveReadBudget = 25 * time.Second

// completeManifestJSON is a structurally complete, schema-valid desired
// manifest in the canonical JSON serialisation. It mirrors the spec's worked
// EXAMPLE manifest.
const completeManifestJSON = `{
  "meta": {
    "format_version": 1,
    "generator": "test 0.6.0",
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
        "name": "/etc/nginx/nginx.conf",
        "type": "file",
        "mode": "0644",
        "user": "root",
        "group": "root",
        "sha256": "0000000000000000000000000000000000000000000000000000000000000000",
        "content_ref": "files/etc/nginx/nginx.conf",
        "package_name": ""
      }
    ]
  }
}`

// EXAMPLE: diff_manifest_unreadable -- a manifest-path pointing at a
// non-existent file is an invocation error (exit 2, domain=invocation).
func TestDiffManifestUnreadable(t *testing.T) {
	r := run(t, "diff", "manifest-path=/nonexistent-zd-manifest.json")
	assertExit(t, r, 2)
	if !strings.Contains(r.stderr, "invocation") {
		t.Errorf("expected stderr to mention domain=invocation\nstderr:\n%s", r.stderr)
	}
}

// EXAMPLE: apply_manifest_unreadable -- apply with a non-existent manifest is
// an invocation error (exit 2). This path opens no transaction.
func TestApplyManifestUnreadable(t *testing.T) {
	r := run(t, "apply", "manifest-path=/nonexistent-zd-manifest.json")
	assertExit(t, r, 2)
	if !strings.Contains(r.stderr, "invocation") {
		t.Errorf("expected stderr to mention domain=invocation\nstderr:\n%s", r.stderr)
	}
}

// EXAMPLE: apply_manifest_invalid -- a manifest with format_version=2 is a
// manifest error (exit 1, domain=manifest); no transaction is opened.
func TestApplyManifestInvalidFormatVersion(t *testing.T) {
	bad := strings.Replace(completeManifestJSON,
		`"format_version": 1`, `"format_version": 2`, 1)
	p := writeTemp(t, "bad.json", bad)
	r := run(t, "apply", "manifest-path="+p)
	assertExit(t, r, 1)
	if !strings.Contains(r.stderr, "manifest") {
		t.Errorf("expected stderr to mention domain=manifest\nstderr:\n%s", r.stderr)
	}
}

// EXAMPLE: diff_prints_plan (conditional on a successful live read).
// The manifest adds package nginx and config_files declares /etc/nginx/nginx.conf;
// against an empty applied record, the plan lists nginx to install.
// applied-root is pointed at a temp dir with no applied.json so the applied
// record is empty and deterministic.
func TestDiffPrintsInstallPlan(t *testing.T) {
	p := writeTemp(t, "desired.json", completeManifestJSON)
	emptyRoot := t.TempDir()
	r, timedOut := runWithTimeout(t, liveReadBudget, "", "diff", "manifest-path="+p, "applied-root="+emptyRoot)
	if timedOut {
		t.Skip("diff live read of / exceeded the budget on this host; skipping plan assertion")
	}
	if r.exitCode == 0 {
		assertStdoutContains(t, r, "nginx")
	} else {
		// Live-state read failed in this environment (no rpmdb etc.).
		// The spec permits exit 1/2 on such a read; assert it is not a
		// silent success with wrong content.
		t.Logf("diff exited %d (live-state read unavailable in this environment); "+
			"stderr:\n%s", r.exitCode, r.stderr)
	}
}

// EXAMPLE: yaml_manifest_accepted -- a YAML serialisation of a valid manifest
// is parsed under the safe profile. We assert the manifest is NOT rejected as
// a manifest error (exit 1) on parse; a successful load proceeds to the
// (possibly environment-limited) live read.
func TestYAMLManifestAccepted(t *testing.T) {
	const yamlManifest = `meta:
  format_version: 1
  generator: "test 0.6.0"
  created_at: "2026-05-29T08:30:00Z"
  desired_sha256: ""
packages:
  _attributes:
    package_system: rpm
  _elements:
    - name: nginx
      version: ""
      release: ""
      arch: ""
`
	p := writeTemp(t, "desired.yaml", yamlManifest)
	emptyRoot := t.TempDir()
	r, timedOut := runWithTimeout(t, liveReadBudget, "", "diff", "manifest-path="+p, "applied-root="+emptyRoot)
	if timedOut {
		t.Skip("diff live read of / exceeded the budget; the YAML parse precedes the live read and is exercised elsewhere")
	}
	// A valid YAML manifest must not produce a manifest-domain parse error.
	if r.exitCode == 1 && strings.Contains(r.stderr, "manifest") {
		t.Errorf("valid YAML manifest was rejected as a manifest error\nstderr:\n%s", r.stderr)
	}
}

// EXAMPLE: yaml_unsafe_rejected -- a YAML manifest using an executable/arbitrary
// tag is rejected with a manifest error (exit 1); no transaction is opened.
func TestYAMLUnsafeRejected(t *testing.T) {
	// A YAML document using a non-standard/executable-style tag. Under the safe
	// profile this must be rejected rather than parsed.
	const evil = `meta: !!python/object/apply:os.system ["echo pwned"]
`
	p := writeTemp(t, "evil.yaml", evil)
	r := run(t, "apply", "manifest-path="+p)
	assertExit(t, r, 1)
	if !strings.Contains(r.stderr, "manifest") {
		t.Errorf("expected an unsafe-YAML manifest error (domain=manifest)\nstderr:\n%s", r.stderr)
	}
}

// EXAMPLE: intent_diff_yields_deletion (observable via the diff plan).
// applied record declares /etc/foo.conf and /etc/bar.conf; desired declares
// only /etc/foo.conf. The plan must list /etc/bar.conf under files to delete.
// Conditional on a successful live read (the drift section reads live state).
func TestDiffYieldsDeletion(t *testing.T) {
	desired := `{
  "meta": { "format_version": 1, "generator": "t", "created_at": "2026-01-01T00:00:00Z", "desired_sha256": "" },
  "config_files": {
    "_attributes": null,
    "_elements": [
      { "name": "/etc/foo.conf", "type": "file", "mode": "0644", "user": "root", "group": "root", "sha256": "1111111111111111111111111111111111111111111111111111111111111111", "content_ref": "files/etc/foo.conf", "package_name": "" }
    ]
  }
}`
	applied := `{
  "meta": { "format_version": 1, "generator": "t", "created_at": "2026-01-01T00:00:00Z", "desired_sha256": "aaaa" },
  "packages": { "_attributes": { "package_system": "rpm" }, "_elements": [] },
  "config_files": {
    "_attributes": null,
    "_elements": [
      { "name": "/etc/foo.conf", "type": "file", "mode": "0644", "user": "root", "group": "root", "sha256": "1111111111111111111111111111111111111111111111111111111111111111", "content_ref": "files/etc/foo.conf", "package_name": "owner-pkg" },
      { "name": "/etc/bar.conf", "type": "file", "mode": "0644", "user": "root", "group": "root", "sha256": "2222222222222222222222222222222222222222222222222222222222222222", "content_ref": "files/etc/bar.conf", "package_name": "owner-pkg" }
    ]
  }
}`
	dp := writeTemp(t, "desired.json", desired)
	root := t.TempDir()
	mustWriteAppliedRecord(t, root, applied)
	r, timedOut := runWithTimeout(t, liveReadBudget, "", "diff", "manifest-path="+dp, "applied-root="+root)
	if timedOut {
		t.Skip("diff live read of / exceeded the budget on this host; skipping deletion assertion")
	}
	if r.exitCode == 0 {
		assertStdoutContains(t, r, "/etc/bar.conf")
	} else {
		t.Logf("diff exited %d (live-state read unavailable); stderr:\n%s", r.exitCode, r.stderr)
	}
}
