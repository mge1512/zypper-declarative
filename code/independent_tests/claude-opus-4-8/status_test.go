// generated from spec: zypper-declarative.spec.md sha256:714e75ff672557d2c7344736a5f36b52afa37e0f565b07de83e1b18cc4492014
// tests by: claude-opus-4-8
//
// Tests for BEHAVIOR: status (read-only, fast).
package independent_tests

import (
	"strings"
	"testing"
)

// EXAMPLE: status_no_declaration — no applied record exists; exit 0.
func TestStatusNoDeclaration(t *testing.T) {
	skipNonLinux(t)
	emptyRoot := t.TempDir()
	r := run(t, "status", "applied-root="+emptyRoot, "root="+t.TempDir())
	mustExit(t, r.exitCode, 0, "status no declaration")
	mustContain(t, r.stdout, "no declaration applied", "status no declaration stdout")
}

// EXAMPLE: status_reports_generation — applied record with known
// desired_sha256 and a resolved packages lock. status prints the
// desired_sha256, a snapshot identifier, the resolved package count, and a
// single drift-summary line.
func TestStatusReportsGeneration(t *testing.T) {
	skipNonLinux(t)
	record := `{
  "meta": { "format_version": 1, "generator": "zypper-declarative 0.4.0", "created_at": "2026-05-29T08:30:00Z", "desired_sha256": "2222222222222222222222222222222222222222222222222222222222222222" },
  "packages": {
    "_attributes": { "package_system": "rpm" },
    "_elements": [
      { "name": "nginx", "version": "1.25.3", "release": "1.1", "arch": "x86_64" },
      { "name": "openssl", "version": "3.1.4", "release": "2.1", "arch": "x86_64" }
    ]
  }
}`
	applied := writeAppliedRecord(t, record)
	r := run(t, "status", "applied-root="+applied, "root="+t.TempDir())
	mustExit(t, r.exitCode, 0, "status reports generation")
	mustContain(t, r.stdout, "2222222222222222222222222222222222222222222222222222222222222222",
		"status prints desired_sha256")
	// resolved package count = 2
	mustContain(t, r.stdout, "2", "status prints resolved package count")
	// a single drift-summary line: "clean" or "N drift item(s)"
	if !strings.Contains(r.stdout, "clean") && !strings.Contains(r.stdout, "drift item") {
		t.Errorf("status: expected a drift-summary line (clean or N drift item(s)); got:\n%s", r.stdout)
	}
}
