// generated from spec: zypper-declarative.spec.md sha256:714e75ff672557d2c7344736a5f36b52afa37e0f565b07de83e1b18cc4492014
// tests by: claude-opus-4-8
//
// Tests for BEHAVIOR: describe. describe-actual-state degrades to empty
// scopes when system tooling is unavailable; the emitted document remains a
// schema-valid Manifest in the declarable subset. We test the structural
// shape and format selection here. Tests requiring a populated live system
// (a specific installed package, a real changed file) are documented in
// TEST_REPORT.md as needing root/live tooling and are not asserted here.
package independent_tests

import (
	"encoding/json"
	"strings"
	"testing"
)

// EXAMPLE: describe_emits_manifest (structural portion)
// stdout is a JSON document with meta.format_version = 1 and a packages
// scope carrying package_system = "rpm". (The populated-nginx assertion
// requires live tooling; see TEST_REPORT.md.)
func TestDescribeEmitsSchemaValidManifest(t *testing.T) {
	skipNonLinux(t)
	// Describe an empty root so the result is deterministic.
	root := t.TempDir()
	r := run(t, "describe", "root="+root)
	mustExit(t, r.exitCode, 0, "describe emits manifest")

	var doc map[string]json.RawMessage
	if err := json.Unmarshal([]byte(r.stdout), &doc); err != nil {
		t.Fatalf("describe output is not valid JSON: %v\noutput:\n%s", err, r.stdout)
	}
	meta, ok := doc["meta"]
	if !ok {
		t.Fatalf("describe output has no meta:\n%s", r.stdout)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(meta, &m); err != nil {
		t.Fatalf("meta not an object: %v", err)
	}
	if fv, ok := m["format_version"].(float64); !ok || int(fv) != 1 {
		t.Errorf("describe meta.format_version: expected 1, got %v", m["format_version"])
	}
	// MILESTONE 0.1.0 acceptance: describe output contains "package_system"
	mustContain(t, r.stdout, "package_system", "describe contains package_system attribute")
}

// EXAMPLE: describe_format_yaml — stdout is a YAML document representing the
// same data model; it is not Machinery-compatible JSON.
func TestDescribeFormatYAML(t *testing.T) {
	skipNonLinux(t)
	root := t.TempDir()
	r := run(t, "describe", "format=yaml", "root="+root)
	mustExit(t, r.exitCode, 0, "describe format=yaml")
	// A YAML document is not parseable as a JSON object at top level.
	var asJSON map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(r.stdout)), &asJSON); err == nil {
		t.Errorf("describe format=yaml produced JSON, not YAML:\n%s", r.stdout)
	}
	// YAML rendering of underscore_style keys still mentions the key names.
	mustContain(t, r.stdout, "format_version", "yaml describe mentions format_version")
}

// EXAMPLE: describe_output_unwritable — out path not writable -> exit 2.
func TestDescribeOutputUnwritable(t *testing.T) {
	skipNonLinux(t)
	root := t.TempDir()
	// A path under a non-existent, non-creatable parent.
	r := run(t, "describe", "root="+root, "out=/proc/zd-cannot-write/state.json")
	mustExit(t, r.exitCode, 2, "describe output unwritable")
	mustContain(t, r.stderr, "invocation", "describe output unwritable domain=invocation")
}

// EXAMPLE: describe_bootstraps_desired_manifest (round-trip portion)
// describe output is accepted unchanged by load-desired-manifest: piping
// describe into a file and running diff against it must not error on the
// manifest being unreadable or invalid (exit 0, not 1 or 2).
func TestDescribeOutputAcceptedAsDesiredManifest(t *testing.T) {
	skipNonLinux(t)
	root := t.TempDir()
	out := writeTempFile(t, "desired.json", "")
	r := run(t, "describe", "root="+root, "out="+out)
	mustExit(t, r.exitCode, 0, "describe to file")

	// Now feed it back to diff against an empty applied-root.
	appliedRoot := t.TempDir()
	r2 := run(t, "diff", "manifest-path="+out, "applied-root="+appliedRoot, "root="+t.TempDir())
	if r2.exitCode == 1 || r2.exitCode == 2 {
		t.Errorf("describe output rejected by load-desired-manifest: exit=%d stderr=%s",
			r2.exitCode, r2.stderr)
	}
}
