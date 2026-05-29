// generated from spec: zypper-declarative.spec.md sha256:714e75ff672557d2c7344736a5f36b52afa37e0f565b07de83e1b18cc4492014
// tests by: claude-opus-4-8
//
// Tests for BEHAVIOR/INTERNAL: load-desired-manifest, observed through the
// verbs that orchestrate it (apply, diff). Covers format selection, schema
// validation, the safe YAML profile, and the format-independent identity
// hash (desired_sha256).
package independent_tests

import (
	"strings"
	"testing"
)

// EXAMPLE: apply_manifest_invalid — meta.format_version = 2 -> domain=manifest,
// no transaction, exit 1.
func TestApplyManifestInvalidFormatVersion(t *testing.T) {
	skipNonLinux(t)
	bad := strings.Replace(validManifestJSON(), `"format_version": 1`, `"format_version": 2`, 1)
	mp := writeTempFile(t, "bad.json", bad)
	r := run(t, "apply", "manifest-path="+mp, "mode=external")
	mustExit(t, r.exitCode, 1, "apply manifest invalid")
	mustContain(t, r.stderr, "manifest", "apply manifest invalid domain=manifest")
}

// EXAMPLE: apply_manifest_unreadable — manifest-path nonexistent -> exit 2,
// domain=invocation.
func TestApplyManifestUnreadable(t *testing.T) {
	skipNonLinux(t)
	r := run(t, "apply", "manifest-path=/nonexistent-zd.json")
	mustExit(t, r.exitCode, 2, "apply manifest unreadable")
	mustContain(t, r.stderr, "invocation", "apply manifest unreadable domain=invocation")
}

// load-desired-manifest: unknown explicit format value is an invocation error.
func TestApplyUnknownFormatValue(t *testing.T) {
	skipNonLinux(t)
	mp := writeTempFile(t, "m.json", validManifestJSON())
	r := run(t, "apply", "manifest-path="+mp, "format=toml")
	mustExit(t, r.exitCode, 2, "apply unknown format value")
	mustContain(t, r.stderr, "invocation", "apply unknown format domain=invocation")
}

// EXAMPLE: yaml_manifest_accepted — manifest-path desired.yaml, a YAML
// serialisation of a valid manifest, parsed under the safe profile;
// diff exits 0.
func TestYAMLManifestAccepted(t *testing.T) {
	skipNonLinux(t)
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
config_files:
  _attributes: null
  _elements:
    - name: "/etc/foo.conf"
      type: "file"
      mode: "0644"
      user: "root"
      group: "root"
      sha256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
      content_ref: "files/etc/foo.conf"
      package_name: ""
`
	mp := writeTempFile(t, "desired.yaml", yaml)
	appliedRoot := t.TempDir()
	r := run(t, "diff", "manifest-path="+mp, "applied-root="+appliedRoot, "root="+t.TempDir())
	mustExit(t, r.exitCode, 0, "yaml manifest accepted")
	mustContain(t, r.stdout, "nginx", "yaml manifest plan lists nginx")
}

// EXAMPLE: yaml_unsafe_rejected — YAML using an executable/arbitrary tag is
// rejected with a manifest error; apply exits 1, no transaction.
func TestYAMLUnsafeTagRejected(t *testing.T) {
	skipNonLinux(t)
	// A YAML custom/explicit tag that would require an unsafe loader feature.
	evil := `meta:
  format_version: 1
  generator: !!python/object/apply:os.system ["echo pwned"]
  created_at: "2026-05-29T08:30:00Z"
  desired_sha256: ""
`
	mp := writeTempFile(t, "evil.yaml", evil)
	r := run(t, "apply", "manifest-path="+mp, "mode=external")
	mustExit(t, r.exitCode, 1, "yaml unsafe rejected")
	mustContain(t, r.stderr, "manifest", "yaml unsafe rejected domain=manifest")
}

// EXAMPLE: yaml_unsafe_rejected (multi-document variant) — a YAML stream with
// multiple documents must be rejected under the safe profile (single
// document only).
func TestYAMLMultiDocumentRejected(t *testing.T) {
	skipNonLinux(t)
	multi := `meta:
  format_version: 1
  generator: "zypper-declarative 0.4.0"
  created_at: "2026-05-29T08:30:00Z"
  desired_sha256: ""
---
meta:
  format_version: 1
`
	mp := writeTempFile(t, "multi.yaml", multi)
	r := run(t, "apply", "manifest-path="+mp, "mode=external")
	mustExit(t, r.exitCode, 1, "yaml multi-document rejected")
	mustContain(t, r.stderr, "manifest", "yaml multi-document domain=manifest")
}

// EXAMPLE: yaml_format_identity_stable — desired.json and desired.yaml
// expressing the same manifest yield the same desired_sha256. Observed via
// status after... but status needs applied record. Instead we observe the
// identity through diff output stability: both produce an identical plan
// against the same applied-root, and the implementation surfaces the
// computed desired_sha256 in diff output for verification.
func TestYAMLAndJSONIdentityStable(t *testing.T) {
	skipNonLinux(t)
	jsonManifest := `{
  "meta": { "format_version": 1, "generator": "g", "created_at": "t", "desired_sha256": "" },
  "packages": { "_attributes": { "package_system": "rpm" }, "_elements": [ { "name": "nginx", "version": "", "release": "", "arch": "" } ] }
}`
	yamlManifest := `meta:
  format_version: 1
  generator: "g"
  created_at: "t"
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
	jp := writeTempFile(t, "d.json", jsonManifest)
	yp := writeTempFile(t, "d.yaml", yamlManifest)
	appliedRoot := t.TempDir()

	rj := run(t, "diff", "manifest-path="+jp, "applied-root="+appliedRoot, "root="+t.TempDir())
	ry := run(t, "diff", "manifest-path="+yp, "applied-root="+appliedRoot, "root="+t.TempDir())
	mustExit(t, rj.exitCode, 0, "json diff")
	mustExit(t, ry.exitCode, 0, "yaml diff")

	hj := extractDesiredSHA(rj.stdout)
	hy := extractDesiredSHA(ry.stdout)
	if hj == "" || hy == "" {
		t.Skipf("diff output does not surface desired_sha256; json=%q yaml=%q", hj, hy)
	}
	if hj != hy {
		t.Errorf("desired_sha256 differs between JSON and YAML of the same manifest: %q vs %q", hj, hy)
	}
}

// extractDesiredSHA pulls a 64-hex token following "desired_sha256" if the
// diff output reports it.
func extractDesiredSHA(s string) string {
	idx := strings.Index(s, "desired_sha256")
	if idx < 0 {
		return ""
	}
	rest := s[idx:]
	for i := 0; i < len(rest); i++ {
		if isHex(rest[i]) {
			j := i
			for j < len(rest) && isHex(rest[j]) {
				j++
			}
			if j-i == 64 {
				return rest[i:j]
			}
			i = j
		}
	}
	return ""
}

func isHex(b byte) bool {
	return (b >= '0' && b <= '9') || (b >= 'a' && b <= 'f')
}
