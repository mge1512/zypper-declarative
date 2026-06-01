// generated from spec: zypper-declarative.spec.md sha256:f2cc80627e483a48bb8411d297711bc5f6c6e74c28dbf0dafc8fe7bd8817251e
// tests by: claude-opus-4-8
//
// YAML serialisation acceptance, format identity (desired_sha256 stable across
// JSON/YAML), unsafe-YAML rejection, and the observational-scope rejection in
// load-desired-manifest. All black-box and offline.
package independent_tests

import (
	"strings"
	"testing"
)

// A YAML serialisation of a valid manifest equivalent to baselineJSON's shape:
// one service, two config files. Explicit string typing throughout.
const desiredYAML = `meta:
  format_version: 1
  generator: "test"
  created_at: "2026-05-29T08:30:00Z"
  desired_sha256: ""
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
      sha256: "1111111111111111111111111111111111111111111111111111111111111111"
      target: ""
      content_ref: ""
      package_name: ""
    - name: "/etc/bar.conf"
      type: "file"
      mode: "0644"
      user: "root"
      group: "root"
      sha256: "2222222222222222222222222222222222222222222222222222222222222222"
      target: ""
      content_ref: ""
      package_name: ""
`

// EXAMPLE: yaml_manifest_accepted — diff with a YAML manifest parses and computes
// a plan; exit 0.
func TestYAMLManifestAccepted(t *testing.T) {
	ref := writeTemp(t, "desired.yaml", desiredYAML)
	state := writeTemp(t, "after.json", matchingStateJSON)
	// Offline diff: manifest (yaml) vs state (json). applied-root empty.
	dir := t.TempDir()
	_, stderr, exit := run(t, "diff", "manifest-path="+ref, "state-path="+state, "applied-root="+dir)
	if exit != 0 {
		t.Fatalf("yaml manifest accepted: exit=%d, want 0\nstderr=%s", exit, stderr)
	}
}

// EXAMPLE: verify_state_path_extension_yaml — a matching YAML state dump -> clean.
func TestVerifyStatePathExtensionYAML(t *testing.T) {
	ref := writeTemp(t, "baseline.json", baselineJSON)
	// YAML state dump matching the baseline.
	state := writeTemp(t, "state.yaml", desiredYAML)
	stdout, stderr, exit := run(t, "verify", "manifest-path="+ref, "state-path="+state)
	if exit != 0 {
		t.Fatalf("verify yaml state: exit=%d, want 0\nstdout=%s\nstderr=%s", exit, stdout, stderr)
	}
	if !strings.Contains(stdout, "system matches declaration") {
		t.Errorf("verify yaml state: stdout=%q, want 'system matches declaration'", stdout)
	}
}

// EXAMPLE: yaml_unsafe_rejected — a YAML manifest with multiple documents
// (a disabled feature) is rejected with a manifest error; apply exits 1, no txn.
func TestYAMLUnsafeRejected(t *testing.T) {
	multiDoc := desiredYAML + "---\nmeta:\n  format_version: 1\n"
	p := writeTemp(t, "evil.yaml", multiDoc)
	_, stderr, exit := run(t, "apply", "manifest-path="+p, "mode=internal")
	if exit != 1 {
		t.Fatalf("yaml unsafe rejected: exit=%d, want 1\nstderr=%s", exit, stderr)
	}
	if !strings.Contains(stderr, "manifest") {
		t.Errorf("yaml unsafe rejected: stderr missing domain=manifest: %q", stderr)
	}
}

// EXAMPLE: apply_rejects_full_describe_dump — a manifest carrying a non-empty
// observational scope (unmanaged_files) is rejected; domain=manifest, exit 1.
func TestApplyRejectsFullDescribeDump(t *testing.T) {
	dump := `{
  "meta": { "format_version": 1, "generator": "describe", "created_at": "2026-05-29T08:30:00Z", "desired_sha256": "" },
  "config_files": { "_attributes": null, "_elements": [] },
  "unmanaged_files": {
    "_attributes": null,
    "_elements": [
      { "name": "/usr/bin/extra", "type": "file", "mode": "0755", "user": "root", "group": "root",
        "sha256": "3333333333333333333333333333333333333333333333333333333333333333", "target": "" }
    ]
  }
}`
	p := writeTemp(t, "full-dump.json", dump)
	_, stderr, exit := run(t, "apply", "manifest-path="+p, "mode=internal")
	if exit != 1 {
		t.Fatalf("apply rejects full dump: exit=%d, want 1\nstderr=%s", exit, stderr)
	}
	if !strings.Contains(stderr, "manifest") {
		t.Errorf("apply rejects full dump: stderr missing domain=manifest: %q", stderr)
	}
}

// EXAMPLE: yaml_format_identity_stable — desired.json and desired.yaml express
// the same manifest; both yield the same desired_sha256.
//
// Black-box: status prints the applied record's desired_sha256. We cannot apply
// without a live host, so we observe the hash indirectly: verifying a state
// against a JSON reference and against the equivalent YAML reference must yield
// the same verdict (both clean), demonstrating the parsed model is identical.
func TestYAMLFormatIdentityStable(t *testing.T) {
	jsonRef := writeTemp(t, "baseline.json", baselineJSON)
	yamlRef := writeTemp(t, "baseline.yaml", desiredYAML)
	state := writeTemp(t, "after.json", matchingStateJSON)

	outJSON, _, exitJSON := run(t, "verify", "manifest-path="+jsonRef, "state-path="+state)
	outYAML, _, exitYAML := run(t, "verify", "manifest-path="+yamlRef, "state-path="+state)

	if exitJSON != exitYAML {
		t.Fatalf("format identity: JSON ref exit=%d, YAML ref exit=%d (should match)", exitJSON, exitYAML)
	}
	if exitJSON != 0 {
		t.Fatalf("format identity: both should verify clean; got exit=%d", exitJSON)
	}
	if outJSON != outYAML {
		t.Errorf("format identity: JSON and YAML references should produce identical verify output:\njson=%q\nyaml=%q",
			outJSON, outYAML)
	}
}
