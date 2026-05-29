// generated from spec: zypper-declarative.spec.md sha256:f8ff76ecbc4bbc69a49e2e32b2924da3a64df1ad46196e05ce8c137b684429b2
// tests by: claude-opus-4-8
//
// Tests that combine an applied record with describe/status/diff over a
// controlled root, exercising integration paths that need no privilege.

package independent_test

import (
	"strings"
	"testing"
)

// EXAMPLE: status_reports_generation
// An applied record exists with a known desired_sha256 and a resolved packages
// lock -> status prints the desired_sha256, the resolved package count, and a
// single drift-summary line; exit 0. We point applied-root at the planted record
// (status reads actual state of "/" for the drift summary, but the assertion is
// on the recorded fields and that the verb completes with exit 0).
func TestStatusReportsGeneration(t *testing.T) {
	root := newAppliedRoot(t, validAppliedRecordJSON)
	r := run(t, "status", "applied-root="+root)
	if r.exitCode != 0 {
		t.Fatalf("status reports generation: exit = %d, want 0; stdout=%q stderr=%q", r.exitCode, r.stdout, r.stderr)
	}
	if !strings.Contains(r.stdout, "deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef0") {
		t.Errorf("status reports generation: stdout %q does not contain the recorded desired_sha256", r.stdout)
	}
}

// EXAMPLE: describe_bootstraps_desired_manifest (round-trip, no privilege)
// A document described from a synthetic root, fed back as a desired manifest via
// diff against the SAME synthetic root with an empty applied-root, yields no
// repository changes — proving describe output is accepted unchanged by
// load-desired-manifest. We assert the round-trip completes with exit 0 and the
// described repo alias appears in the plan's repos-to-set (the desired state).
func TestDescribeBootstrapsDesiredManifest(t *testing.T) {
	root := twoRepoRoot(t)
	out := writeTemp(t, ".json", "{}") // placeholder path; describe overwrites it
	dr := run(t, "describe", "root="+root, "on-unreadable=warn", "out="+out)
	if dr.exitCode != 0 {
		t.Fatalf("describe for bootstrap: exit = %d, want 0; stderr=%q", dr.exitCode, dr.stderr)
	}
	// Now load the described document as a desired manifest.
	appliedRoot := t.TempDir() // empty applied-root => first-ever apply
	diffr := run(t, "diff", "applied-root="+appliedRoot, "signature-verification=off", "manifest-path="+out)
	if diffr.exitCode != 0 {
		t.Fatalf("diff over described manifest: exit = %d, want 0; stdout=%q stderr=%q", diffr.exitCode, diffr.stdout, diffr.stderr)
	}
	// The described repositories must round-trip through load-desired-manifest.
	if !strings.Contains(diffr.stdout, "sl-micro-6.2-pinned") {
		t.Errorf("describe bootstrap: diff stdout %q does not reflect the described repository", diffr.stdout)
	}
}
