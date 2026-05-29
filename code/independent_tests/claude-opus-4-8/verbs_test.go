// generated from spec: zypper-declarative.spec.md sha256:f8ff76ecbc4bbc69a49e2e32b2924da3a64df1ad46196e05ce8c137b684429b2
// tests by: claude-opus-4-8
//
// Tests for the verbs that read state but do not require system-modifying
// privilege along the asserted paths: status (no declaration), verify
// (no applied record, malformed dump, external dump drift), diff/apply
// (manifest unreadable, manifest invalid). Privileged convergence paths
// (real transactions, real rpmdb writes) are not asserted here because they
// require root and a live transactional system; only the spec's observable
// invocation/manifest/no-op behaviours that a normal user can reach are tested.

package independent_test

import (
	"path/filepath"
	"strings"
	"testing"
)

// validManifestJSON is a structurally-complete, schema-valid desired manifest
// in the canonical JSON serialisation, taken from the spec's illustration.
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

// invalidFormatVersionJSON is otherwise complete but declares format_version 2,
// which the schema forbids (must be 1).
const invalidFormatVersionJSON = `{
  "meta": {
    "format_version": 2,
    "generator": "test",
    "created_at": "2026-05-29T08:30:00Z",
    "desired_sha256": ""
  },
  "packages": { "_attributes": { "package_system": "rpm" }, "_elements": [] }
}`

// EXAMPLE: apply_manifest_unreadable
// manifest-path points at a file that does not exist -> domain=invocation, exit 2.
func TestApplyManifestUnreadable(t *testing.T) {
	r := run(t, "apply", "manifest-path=/nonexistent-zd-manifest.json")
	if r.exitCode != 2 {
		t.Fatalf("apply manifest unreadable: exit = %d, want 2; stdout=%q stderr=%q", r.exitCode, r.stdout, r.stderr)
	}
	if !strings.Contains(r.stderr, "invocation") {
		t.Errorf("apply manifest unreadable: stderr %q does not carry domain=invocation", r.stderr)
	}
}

// EXAMPLE: diff_manifest_unreadable
// manifest-path points at an unreadable file -> domain=invocation, exit 2.
func TestDiffManifestUnreadable(t *testing.T) {
	r := run(t, "diff", "manifest-path=/nonexistent-zd-manifest.json")
	if r.exitCode != 2 {
		t.Fatalf("diff manifest unreadable: exit = %d, want 2; stdout=%q stderr=%q", r.exitCode, r.stdout, r.stderr)
	}
	if !strings.Contains(r.stderr, "invocation") {
		t.Errorf("diff manifest unreadable: stderr %q does not carry domain=invocation", r.stderr)
	}
}

// EXAMPLE: apply_manifest_invalid
// meta.format_version = 2 -> domain=manifest, exit 1, no transaction opened.
func TestApplyManifestInvalid(t *testing.T) {
	p := writeTemp(t, ".json", invalidFormatVersionJSON)
	r := run(t, "apply", "manifest-path="+p)
	if r.exitCode != 1 {
		t.Fatalf("apply invalid manifest: exit = %d, want 1; stdout=%q stderr=%q", r.exitCode, r.stdout, r.stderr)
	}
	if !strings.Contains(r.stderr, "manifest") {
		t.Errorf("apply invalid manifest: stderr %q does not carry domain=manifest", r.stderr)
	}
}

// diff against an invalid manifest is a manifest error (exit 1).
func TestDiffManifestInvalid(t *testing.T) {
	p := writeTemp(t, ".json", invalidFormatVersionJSON)
	r := run(t, "diff", "manifest-path="+p)
	if r.exitCode != 1 {
		t.Fatalf("diff invalid manifest: exit = %d, want 1; stdout=%q stderr=%q", r.exitCode, r.stdout, r.stderr)
	}
	if !strings.Contains(r.stderr, "manifest") {
		t.Errorf("diff invalid manifest: stderr %q does not carry domain=manifest", r.stderr)
	}
}

// EXAMPLE: status_no_declaration
// With no applied record present (applied-root pointed at an empty tree),
// status prints "no declaration applied" and exits 0.
func TestStatusNoDeclaration(t *testing.T) {
	root := t.TempDir()
	r := run(t, "status", "applied-root="+root)
	if r.exitCode != 0 {
		t.Fatalf("status no declaration: exit = %d, want 0; stdout=%q stderr=%q", r.exitCode, r.stdout, r.stderr)
	}
	if !strings.Contains(r.stdout, "no declaration applied") {
		t.Errorf("status no declaration: stdout %q does not contain 'no declaration applied'", r.stdout)
	}
}

// EXAMPLE: verify_no_applied_record
// No applied record exists -> "no declaration applied" to stderr, domain=invocation, exit 2.
func TestVerifyNoAppliedRecord(t *testing.T) {
	root := t.TempDir()
	r := run(t, "verify", "applied-root="+root)
	if r.exitCode != 2 {
		t.Fatalf("verify no applied record: exit = %d, want 2; stdout=%q stderr=%q", r.exitCode, r.stdout, r.stderr)
	}
	if !strings.Contains(r.stderr, "no declaration applied") {
		t.Errorf("verify no applied record: stderr %q does not contain 'no declaration applied'", r.stderr)
	}
	if !strings.Contains(r.stderr, "invocation") {
		t.Errorf("verify no applied record: stderr %q does not carry domain=invocation", r.stderr)
	}
}

// EXAMPLE: verify_malformed_state_dump
// state-path points at a file that is not a valid shared-schema Manifest
// -> domain=invocation, exit 2. An applied record must exist for verify to
// reach the state-dump load step, so we plant one under applied-root.
func TestVerifyMalformedStateDump(t *testing.T) {
	root := newAppliedRoot(t, validAppliedRecordJSON)
	broken := writeTemp(t, ".json", "{ this is not valid json :::")
	r := run(t, "verify", "applied-root="+root, "state-path="+broken)
	if r.exitCode != 2 {
		t.Fatalf("verify malformed dump: exit = %d, want 2; stdout=%q stderr=%q", r.exitCode, r.stdout, r.stderr)
	}
	if !strings.Contains(r.stderr, "invocation") {
		t.Errorf("verify malformed dump: stderr %q does not carry domain=invocation", r.stderr)
	}
}

// EXAMPLE: verify_against_external_state_dump
// A state dump that diverges from the applied record in one declared service
// state -> a diagnostic with domain=units naming the divergent service, exit 1.
func TestVerifyExternalStateDumpDrift(t *testing.T) {
	root := newAppliedRoot(t, validAppliedRecordJSON)
	// Dump that matches the applied packages but reports the service disabled
	// where the applied record declares it enabled.
	dump := `{
  "meta": { "format_version": 1, "generator": "dump", "created_at": "2026-05-29T08:30:00Z", "desired_sha256": "" },
  "packages": { "_attributes": { "package_system": "rpm" },
    "_elements": [ { "name": "nginx", "version": "1.24.0", "release": "1.1", "arch": "x86_64" } ] },
  "services": { "_attributes": { "init_system": "systemd" },
    "_elements": [ { "name": "nginx.service", "state": "disabled" } ] }
}`
	dumpPath := writeTemp(t, ".json", dump)
	r := run(t, "verify", "applied-root="+root, "state-path="+dumpPath)
	if r.exitCode != 1 {
		t.Fatalf("verify external drift: exit = %d, want 1; stdout=%q stderr=%q", r.exitCode, r.stdout, r.stderr)
	}
	if !strings.Contains(r.stderr, "units") {
		t.Errorf("verify external drift: stderr %q does not carry domain=units", r.stderr)
	}
	if !strings.Contains(r.stderr, "nginx.service") {
		t.Errorf("verify external drift: stderr %q does not name the divergent service nginx.service", r.stderr)
	}
}

// EXAMPLE: verify_clean (variant using a state dump rather than live read).
// A state dump that equals the applied record -> "system matches declaration", exit 0.
func TestVerifyCleanWithMatchingDump(t *testing.T) {
	root := newAppliedRoot(t, validAppliedRecordJSON)
	dump := `{
  "meta": { "format_version": 1, "generator": "dump", "created_at": "2026-05-29T08:30:00Z", "desired_sha256": "" },
  "packages": { "_attributes": { "package_system": "rpm" },
    "_elements": [ { "name": "nginx", "version": "1.24.0", "release": "1.1", "arch": "x86_64" } ] },
  "services": { "_attributes": { "init_system": "systemd" },
    "_elements": [ { "name": "nginx.service", "state": "enabled" } ] }
}`
	dumpPath := writeTemp(t, ".json", dump)
	r := run(t, "verify", "applied-root="+root, "state-path="+dumpPath)
	if r.exitCode != 0 {
		t.Fatalf("verify clean dump: exit = %d, want 0; stdout=%q stderr=%q", r.exitCode, r.stdout, r.stderr)
	}
	if !strings.Contains(r.stdout, "system matches declaration") {
		t.Errorf("verify clean dump: stdout %q does not contain 'system matches declaration'", r.stdout)
	}
}

// EXAMPLE: verify_state_path_extension_yaml
// A YAML state dump matching the applied record, selected by .yaml extension
// (no format option) -> "system matches declaration", exit 0.
func TestVerifyStatePathExtensionYAML(t *testing.T) {
	root := newAppliedRoot(t, validAppliedRecordJSON)
	dump := "" +
		"meta:\n" +
		"  format_version: 1\n" +
		"  generator: dump\n" +
		"  created_at: \"2026-05-29T08:30:00Z\"\n" +
		"  desired_sha256: \"\"\n" +
		"packages:\n" +
		"  _attributes:\n" +
		"    package_system: rpm\n" +
		"  _elements:\n" +
		"    - name: nginx\n" +
		"      version: \"1.24.0\"\n" +
		"      release: \"1.1\"\n" +
		"      arch: x86_64\n" +
		"services:\n" +
		"  _attributes:\n" +
		"    init_system: systemd\n" +
		"  _elements:\n" +
		"    - name: nginx.service\n" +
		"      state: enabled\n"
	dumpPath := writeTemp(t, ".yaml", dump)
	r := run(t, "verify", "applied-root="+root, "state-path="+dumpPath)
	if r.exitCode != 0 {
		t.Fatalf("verify yaml dump by extension: exit = %d, want 0; stdout=%q stderr=%q", r.exitCode, r.stdout, r.stderr)
	}
	if !strings.Contains(r.stdout, "system matches declaration") {
		t.Errorf("verify yaml dump by extension: stdout %q does not contain 'system matches declaration'", r.stdout)
	}
}

// EXAMPLE: diff_prints_plan (read-only path) — exercised against an empty
// applied-root so diff treats all applied scopes as empty and the desired
// manifest yields a plan that lists nginx to install. diff opens no transaction
// and modifies nothing; exit 0.
func TestDiffPrintsPlan(t *testing.T) {
	root := t.TempDir() // no applied.json => all applied scopes empty
	p := writeTemp(t, ".json", validManifestJSON)
	r := run(t, "diff", "applied-root="+root, "signature-verification=off", "manifest-path="+p)
	if r.exitCode != 0 {
		t.Fatalf("diff prints plan: exit = %d, want 0; stdout=%q stderr=%q", r.exitCode, r.stdout, r.stderr)
	}
	if !strings.Contains(r.stdout, "nginx") {
		t.Errorf("diff prints plan: stdout %q does not list nginx to install", r.stdout)
	}
}

// EXAMPLE: yaml_manifest_accepted
// A YAML desired manifest is parsed under the safe profile and validated; diff
// computes a plan and exits 0. Uses an empty applied-root so no privilege is needed.
func TestYAMLManifestAccepted(t *testing.T) {
	root := t.TempDir()
	yamlManifest := "" +
		"meta:\n" +
		"  format_version: 1\n" +
		"  generator: test\n" +
		"  created_at: \"2026-05-29T08:30:00Z\"\n" +
		"  desired_sha256: \"\"\n" +
		"packages:\n" +
		"  _attributes:\n" +
		"    package_system: rpm\n" +
		"  _elements:\n" +
		"    - name: nginx\n" +
		"      version: \"\"\n" +
		"      release: \"\"\n" +
		"      arch: \"\"\n"
	p := writeTemp(t, ".yaml", yamlManifest)
	r := run(t, "diff", "applied-root="+root, "signature-verification=off", "manifest-path="+p)
	if r.exitCode != 0 {
		t.Fatalf("yaml manifest accepted: exit = %d, want 0; stdout=%q stderr=%q", r.exitCode, r.stdout, r.stderr)
	}
	if !strings.Contains(r.stdout, "nginx") {
		t.Errorf("yaml manifest accepted: stdout %q does not list nginx", r.stdout)
	}
}

// EXAMPLE: yaml_unsafe_rejected
// A YAML manifest using a code-executing/arbitrary tag is rejected with a
// manifest error (exit 1), no transaction opened.
func TestYAMLUnsafeRejected(t *testing.T) {
	root := t.TempDir()
	// A non-specific Python/object tag is an arbitrary/executable tag under the
	// safe profile and must be rejected.
	unsafe := "" +
		"meta: !!python/object/apply:os.system [\"echo pwned\"]\n"
	p := writeTemp(t, ".yaml", unsafe)
	r := run(t, "apply", "applied-root="+root, "manifest-path="+p)
	if r.exitCode != 1 {
		t.Fatalf("yaml unsafe rejected: exit = %d, want 1; stdout=%q stderr=%q", r.exitCode, r.stdout, r.stderr)
	}
	if !strings.Contains(r.stderr, "manifest") {
		t.Errorf("yaml unsafe rejected: stderr %q does not carry domain=manifest", r.stderr)
	}
}

// EXAMPLE: intent_diff_yields_deletion (observed through diff)
// applied declares /etc/foo.conf and /etc/bar.conf; desired declares only
// /etc/foo.conf -> diff lists /etc/bar.conf under files to delete and does not
// list any path outside the applied config_files scope. Exit 0.
func TestIntentDiffYieldsDeletion(t *testing.T) {
	applied := `{
  "meta": { "format_version": 1, "generator": "applied", "created_at": "2026-05-29T08:30:00Z", "desired_sha256": "abc" },
  "packages": { "_attributes": { "package_system": "rpm" }, "_elements": [] },
  "config_files": { "_attributes": null, "_elements": [
    { "name": "/etc/foo.conf", "type": "file", "mode": "0644", "user": "root", "group": "root",
      "sha256": "1111111111111111111111111111111111111111111111111111111111111111", "content_ref": "", "package_name": "" },
    { "name": "/etc/bar.conf", "type": "file", "mode": "0644", "user": "root", "group": "root",
      "sha256": "2222222222222222222222222222222222222222222222222222222222222222", "content_ref": "", "package_name": "" }
  ] }
}`
	desired := `{
  "meta": { "format_version": 1, "generator": "desired", "created_at": "2026-05-29T08:30:00Z", "desired_sha256": "" },
  "config_files": { "_attributes": null, "_elements": [
    { "name": "/etc/foo.conf", "type": "file", "mode": "0644", "user": "root", "group": "root",
      "sha256": "3333333333333333333333333333333333333333333333333333333333333333", "content_ref": "files/etc/foo.conf", "package_name": "" }
  ] }
}`
	root := newAppliedRoot(t, applied)
	p := writeTemp(t, ".json", desired)
	r := run(t, "diff", "applied-root="+root, "signature-verification=off", "manifest-path="+p)
	if r.exitCode != 0 {
		t.Fatalf("intent diff deletion: exit = %d, want 0; stdout=%q stderr=%q", r.exitCode, r.stdout, r.stderr)
	}
	if !strings.Contains(r.stdout, "/etc/bar.conf") {
		t.Errorf("intent diff deletion: stdout %q does not list /etc/bar.conf for deletion", r.stdout)
	}
}

// newAppliedRoot writes the given applied-record JSON into the canonical
// applied.json location under a fresh temp root, and returns the root path so a
// test can pass applied-root=<root>.
func newAppliedRoot(t *testing.T, recordJSON string) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "usr", "lib", "zypper-declarative")
	mkdirAll(t, dir)
	writeFile(t, filepath.Join(dir, "applied.json"), recordJSON)
	return root
}

// validAppliedRecordJSON is a complete AppliedRecord: packages fully resolved,
// meta.desired_sha256 set.
const validAppliedRecordJSON = `{
  "meta": {
    "format_version": 1,
    "generator": "zypper-declarative test",
    "created_at": "2026-05-29T08:30:00Z",
    "desired_sha256": "deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef0"
  },
  "packages": {
    "_attributes": { "package_system": "rpm" },
    "_elements": [
      { "name": "nginx", "version": "1.24.0", "release": "1.1", "arch": "x86_64" }
    ]
  },
  "services": {
    "_attributes": { "init_system": "systemd" },
    "_elements": [
      { "name": "nginx.service", "state": "enabled" }
    ]
  }
}`
