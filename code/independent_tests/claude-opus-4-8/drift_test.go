// generated from spec: zypper-declarative.spec.md sha256:714e75ff672557d2c7344736a5f36b52afa37e0f565b07de83e1b18cc4492014
// tests by: claude-opus-4-8
//
// Tests for BEHAVIOR/INTERNAL: compute-drift, exercised through verify with
// a supplied state dump. Covers the files_extra rule (only unpackaged,
// undeclared /etc files), package_divergent, and the absent-declared-file
// matching rule.
package independent_tests

import (
	"testing"
)

// appliedDeclaresFoo declares /etc/foo.conf and no other scopes.
func appliedDeclaresFoo() string {
	return `{
  "meta": { "format_version": 1, "generator": "g", "created_at": "t", "desired_sha256": "4444444444444444444444444444444444444444444444444444444444444444" },
  "packages": { "_attributes": { "package_system": "rpm" }, "_elements": [] },
  "config_files": {
    "_attributes": null,
    "_elements": [
      { "name": "/etc/foo.conf", "type": "file", "mode": "0644", "user": "root", "group": "root", "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "content_ref": "", "package_name": "" }
    ]
  }
}`
}

// EXAMPLE: drift_ignores_unmanaged_packaged_file — actual state has a changed
// but package-owned /etc file that the reference does not declare
// (package_name non-empty). It must NOT appear in files_extra, so verify is
// clean (exit 0) when the declared file matches.
func TestDriftIgnoresUnmanagedPackagedFile(t *testing.T) {
	applied := writeAppliedRecord(t, appliedDeclaresFoo())
	dump := `{
  "meta": { "format_version": 1, "generator": "g", "created_at": "t", "desired_sha256": "" },
  "packages": { "_attributes": { "package_system": "rpm" }, "_elements": [] },
  "config_files": {
    "_attributes": null,
    "_elements": [
      { "name": "/etc/foo.conf", "type": "file", "mode": "0644", "user": "root", "group": "root", "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "content_ref": "", "package_name": "" },
      { "name": "/etc/httpd/conf.d/extra.conf", "type": "file", "mode": "0644", "user": "root", "group": "root", "sha256": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "content_ref": "", "package_name": "apache2" }
    ]
  }
}`
	sp := writeTempFile(t, "state.json", dump)
	r := run(t, "verify", "state-path="+sp, "applied-root="+applied)
	mustExit(t, r.exitCode, 0, "drift ignores packaged unmanaged file")
	mustContain(t, r.stdout, "system matches declaration", "packaged unmanaged file not extra")
}

// compute-drift: an unpackaged, undeclared /etc file IS files_extra ->
// verify reports drift, exit 1.
func TestDriftReportsUnpackagedExtraFile(t *testing.T) {
	applied := writeAppliedRecord(t, appliedDeclaresFoo())
	dump := `{
  "meta": { "format_version": 1, "generator": "g", "created_at": "t", "desired_sha256": "" },
  "packages": { "_attributes": { "package_system": "rpm" }, "_elements": [] },
  "config_files": {
    "_attributes": null,
    "_elements": [
      { "name": "/etc/foo.conf", "type": "file", "mode": "0644", "user": "root", "group": "root", "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "content_ref": "", "package_name": "" },
      { "name": "/etc/orphan.conf", "type": "file", "mode": "0644", "user": "root", "group": "root", "sha256": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "content_ref": "", "package_name": "" }
    ]
  }
}`
	sp := writeTempFile(t, "state.json", dump)
	r := run(t, "verify", "state-path="+sp, "applied-root="+applied)
	mustExit(t, r.exitCode, 1, "drift reports unpackaged extra file")
	mustContain(t, r.stderr, "/etc/orphan.conf", "extra file named in diagnostic")
}

// compute-drift: a declared file absent from the actual scope is treated as
// matching (equals the declared default) -> verify clean.
func TestDriftDeclaredFileAbsentTreatedAsMatching(t *testing.T) {
	applied := writeAppliedRecord(t, appliedDeclaresFoo())
	dump := `{
  "meta": { "format_version": 1, "generator": "g", "created_at": "t", "desired_sha256": "" },
  "packages": { "_attributes": { "package_system": "rpm" }, "_elements": [] },
  "config_files": { "_attributes": null, "_elements": [] }
}`
	sp := writeTempFile(t, "state.json", dump)
	r := run(t, "verify", "state-path="+sp, "applied-root="+applied)
	mustExit(t, r.exitCode, 0, "declared file absent treated as matching")
	mustContain(t, r.stdout, "system matches declaration", "absent declared file clean")
}

// compute-drift: packages divergent when reference has a package the actual
// state does not -> verify reports drift, exit 1, domain=packages.
func TestDriftPackagesDivergent(t *testing.T) {
	record := `{
  "meta": { "format_version": 1, "generator": "g", "created_at": "t", "desired_sha256": "5555555555555555555555555555555555555555555555555555555555555555" },
  "packages": {
    "_attributes": { "package_system": "rpm" },
    "_elements": [ { "name": "nginx", "version": "1.25.3", "release": "1.1", "arch": "x86_64" } ]
  }
}`
	applied := writeAppliedRecord(t, record)
	dump := `{
  "meta": { "format_version": 1, "generator": "g", "created_at": "t", "desired_sha256": "" },
  "packages": { "_attributes": { "package_system": "rpm" }, "_elements": [] },
  "config_files": { "_attributes": null, "_elements": [] }
}`
	sp := writeTempFile(t, "state.json", dump)
	r := run(t, "verify", "state-path="+sp, "applied-root="+applied)
	mustExit(t, r.exitCode, 1, "packages divergent")
	mustContain(t, r.stderr, "packages", "packages divergent domain=packages")
}

// keep-list: a keep-listed unpackaged undeclared /etc file must NOT appear in
// files_extra. verify with a keep-list including that path is clean.
func TestDriftKeepListedFileNotExtra(t *testing.T) {
	applied := writeAppliedRecord(t, appliedDeclaresFoo())
	keepList := writeTempFile(t, "keep.list", "/etc/machine-id\n/etc/ssh/ssh_host_rsa_key\n")
	dump := `{
  "meta": { "format_version": 1, "generator": "g", "created_at": "t", "desired_sha256": "" },
  "packages": { "_attributes": { "package_system": "rpm" }, "_elements": [] },
  "config_files": {
    "_attributes": null,
    "_elements": [
      { "name": "/etc/foo.conf", "type": "file", "mode": "0644", "user": "root", "group": "root", "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "content_ref": "", "package_name": "" },
      { "name": "/etc/machine-id", "type": "file", "mode": "0644", "user": "root", "group": "root", "sha256": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "content_ref": "", "package_name": "" }
    ]
  }
}`
	sp := writeTempFile(t, "state.json", dump)
	r := run(t, "verify", "state-path="+sp, "applied-root="+applied, "keep-list="+keepList)
	mustExit(t, r.exitCode, 0, "keep-listed file not extra")
	mustContain(t, r.stdout, "system matches declaration", "keep-listed file clean")
}

// /etc/etc.syncpoint must never appear in files_extra.
func TestDriftSyncpointNeverExtra(t *testing.T) {
	applied := writeAppliedRecord(t, appliedDeclaresFoo())
	dump := `{
  "meta": { "format_version": 1, "generator": "g", "created_at": "t", "desired_sha256": "" },
  "packages": { "_attributes": { "package_system": "rpm" }, "_elements": [] },
  "config_files": {
    "_attributes": null,
    "_elements": [
      { "name": "/etc/foo.conf", "type": "file", "mode": "0644", "user": "root", "group": "root", "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "content_ref": "", "package_name": "" },
      { "name": "/etc/etc.syncpoint", "type": "file", "mode": "0644", "user": "root", "group": "root", "sha256": "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", "content_ref": "", "package_name": "" }
    ]
  }
}`
	sp := writeTempFile(t, "state.json", dump)
	r := run(t, "verify", "state-path="+sp, "applied-root="+applied)
	mustExit(t, r.exitCode, 0, "syncpoint never extra")
	mustContain(t, r.stdout, "system matches declaration", "syncpoint clean")
}
