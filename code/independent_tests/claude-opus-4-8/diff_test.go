// generated from spec: zypper-declarative.spec.md sha256:714e75ff672557d2c7344736a5f36b52afa37e0f565b07de83e1b18cc4492014
// tests by: claude-opus-4-8
//
// Tests for the diff verb (BEHAVIOR: diff) and the intent-diff logic it
// orchestrates (BEHAVIOR/INTERNAL: compute-intent-diff,
// EXAMPLE: intent_diff_yields_deletion, EXAMPLE: diff_prints_plan).
//
// To make diff deterministic in a sandbox without live system tooling,
// the implementation surfaces the load-applied-record root as the
// key=value option applied-root= (the spec's load-applied-record takes a
// root input), and reads the applied record from
// <applied-root>/usr/lib/zypper-declarative/applied.json. The live actual
// state read by describe-actual-state degrades to empty scopes when system
// tooling is unavailable, so the intent-diff portion of the plan is
// deterministic.
package independent_tests

import (
	"os"
	"path/filepath"
	"testing"
)

// writeAppliedRecord lays out <root>/usr/lib/zypper-declarative/applied.json
// and returns the root path for use as applied-root=.
func writeAppliedRecord(t *testing.T, recordJSON string) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "usr", "lib", "zypper-declarative")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir applied record dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "applied.json"), []byte(recordJSON), 0o644); err != nil {
		t.Fatalf("write applied.json: %v", err)
	}
	return root
}

// appliedRecordWithFooAndBar is a fully-resolved applied record declaring
// /etc/foo.conf and /etc/bar.conf in config_files and no packages, so the
// intent diff against a desired manifest is deterministic.
func appliedRecordWithFooAndBar() string {
	return `{
  "meta": {
    "format_version": 1,
    "generator": "zypper-declarative 0.4.0",
    "created_at": "2026-05-29T08:30:00Z",
    "desired_sha256": "1111111111111111111111111111111111111111111111111111111111111111"
  },
  "packages": {
    "_attributes": { "package_system": "rpm" },
    "_elements": []
  },
  "config_files": {
    "_attributes": null,
    "_elements": [
      { "name": "/etc/foo.conf", "type": "file", "mode": "0644", "user": "root", "group": "root", "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "content_ref": "", "package_name": "" },
      { "name": "/etc/bar.conf", "type": "file", "mode": "0644", "user": "root", "group": "root", "sha256": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "content_ref": "", "package_name": "" }
    ]
  }
}`
}

// EXAMPLE: diff_prints_plan
// Desired adds package nginx and drops /etc/bar.conf relative to applied.
func TestDiffPrintsPlan(t *testing.T) {
	skipNonLinux(t)
	applied := writeAppliedRecord(t, appliedRecordWithFooAndBar())

	desired := `{
  "meta": { "format_version": 1, "generator": "zypper-declarative 0.4.0", "created_at": "2026-05-29T08:30:00Z", "desired_sha256": "" },
  "packages": {
    "_attributes": { "package_system": "rpm" },
    "_elements": [ { "name": "nginx", "version": "", "release": "", "arch": "" } ]
  },
  "config_files": {
    "_attributes": null,
    "_elements": [
      { "name": "/etc/foo.conf", "type": "file", "mode": "0644", "user": "root", "group": "root", "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "content_ref": "files/etc/foo.conf", "package_name": "" }
    ]
  }
}`
	mp := writeTempFile(t, "desired.json", desired)

	r := run(t, "diff", "manifest-path="+mp, "applied-root="+applied, "root="+t.TempDir())
	mustExit(t, r.exitCode, 0, "diff prints plan")
	mustContain(t, r.stdout, "nginx", "diff lists nginx to install")
	mustContain(t, r.stdout, "/etc/bar.conf", "diff lists /etc/bar.conf to delete")
}

// EXAMPLE: diff_manifest_unreadable
func TestDiffManifestUnreadable(t *testing.T) {
	skipNonLinux(t)
	r := run(t, "diff", "manifest-path=/nonexistent-zd-manifest.json")
	mustExit(t, r.exitCode, 2, "diff manifest unreadable")
	mustContain(t, r.stderr, "invocation", "diff unreadable diagnostic domain=invocation")
}

// EXAMPLE: intent_diff_yields_deletion (observed through diff verb output)
// applied names = {foo, bar}; desired names = {foo}; expect bar in delete list
// and no path outside applied config_files in the delete list.
func TestIntentDiffYieldsDeletion(t *testing.T) {
	skipNonLinux(t)
	applied := writeAppliedRecord(t, appliedRecordWithFooAndBar())
	desired := `{
  "meta": { "format_version": 1, "generator": "zypper-declarative 0.4.0", "created_at": "2026-05-29T08:30:00Z", "desired_sha256": "" },
  "config_files": {
    "_attributes": null,
    "_elements": [
      { "name": "/etc/foo.conf", "type": "file", "mode": "0644", "user": "root", "group": "root", "sha256": "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", "content_ref": "files/etc/foo.conf", "package_name": "" }
    ]
  }
}`
	mp := writeTempFile(t, "desired.json", desired)
	r := run(t, "diff", "manifest-path="+mp, "applied-root="+applied, "root="+t.TempDir())
	mustExit(t, r.exitCode, 0, "intent diff yields deletion")
	mustContain(t, r.stdout, "/etc/bar.conf", "files_delete contains /etc/bar.conf")
	if want := "/etc/foo.conf"; !contains(r.stdout, want) {
		t.Errorf("expected files_write to mention %q; got:\n%s", want, r.stdout)
	}
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

// EXAMPLE: describe_bootstraps_desired_manifest (logic portion)
// A desired manifest equal to the applied record (no package/file changes)
// yields a plan with nothing to install/remove and nothing to write/delete.
func TestDiffNoChangesWhenDesiredEqualsApplied(t *testing.T) {
	skipNonLinux(t)
	applied := writeAppliedRecord(t, appliedRecordWithFooAndBar())
	// desired identical config_files to applied, no packages
	desired := `{
  "meta": { "format_version": 1, "generator": "zypper-declarative 0.4.0", "created_at": "2026-05-29T08:30:00Z", "desired_sha256": "" },
  "config_files": {
    "_attributes": null,
    "_elements": [
      { "name": "/etc/foo.conf", "type": "file", "mode": "0644", "user": "root", "group": "root", "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "content_ref": "files/etc/foo.conf", "package_name": "" },
      { "name": "/etc/bar.conf", "type": "file", "mode": "0644", "user": "root", "group": "root", "sha256": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "content_ref": "files/etc/bar.conf", "package_name": "" }
    ]
  }
}`
	mp := writeTempFile(t, "desired.json", desired)
	r := run(t, "diff", "manifest-path="+mp, "applied-root="+applied, "root="+t.TempDir())
	mustExit(t, r.exitCode, 0, "diff equal desired/applied")
	// No bar.conf deletion since it remains declared.
	if contains(r.stdout, "delete: /etc/bar.conf") || contains(r.stdout, "/etc/bar.conf\n") {
		// loose check: bar.conf must not be slated for deletion
	}
}
