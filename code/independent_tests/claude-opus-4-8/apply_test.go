// generated from spec: zypper-declarative.spec.md sha256:714e75ff672557d2c7344736a5f36b52afa37e0f565b07de83e1b18cc4492014
// tests by: claude-opus-4-8
//
// Tests for BEHAVIOR: apply. Paths requiring real privilege, real snapshot
// transactions, or live package/unit convergence are documented in
// TEST_REPORT.md as needing root and SUSE transactional tooling; they are
// not asserted in this black-box sandbox. The paths asserted here need no
// privilege:
//   - transaction unavailable (mode=external, not inside a transaction)
//   - no-op when already converged (no transaction opened)
//   - idempotence of the no-op decision
package independent_tests

import (
	"testing"
)

// EXAMPLE: apply_transaction_unavailable — mode=external and not running
// inside a snapshot transaction -> domain=transaction, no modification,
// exit 2.
//
// To reach the transaction-acquisition step the intent diff or drift must be
// non-empty (an empty plan short-circuits to "nothing to do"). We supply a
// desired manifest that adds a config file relative to an empty applied
// record, so the plan is non-empty and apply attempts to acquire the
// (unavailable) external transaction.
func TestApplyTransactionUnavailable(t *testing.T) {
	skipNonLinux(t)
	desired := `{
  "meta": { "format_version": 1, "generator": "zypper-declarative 0.4.0", "created_at": "2026-05-29T08:30:00Z", "desired_sha256": "" },
  "config_files": {
    "_attributes": null,
    "_elements": [
      { "name": "/etc/zd-test.conf", "type": "file", "mode": "0644", "user": "root", "group": "root", "sha256": "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee", "content_ref": "files/etc/zd-test.conf", "package_name": "" }
    ]
  }
}`
	mp := writeTempFile(t, "desired.json", desired)
	appliedRoot := t.TempDir() // empty -> first-ever apply, plan is non-empty
	r := run(t, "apply", "manifest-path="+mp, "mode=external", "applied-root="+appliedRoot)
	mustExit(t, r.exitCode, 2, "apply transaction unavailable")
	mustContain(t, r.stderr, "transaction", "apply transaction unavailable domain=transaction")
}

// EXAMPLE: apply_no_op_when_converged / idempotent_second_apply
// The desired manifest equals the applied record in all managed scopes and
// the system has no drift -> "nothing to do", exit 0, no transaction opened.
//
// Per the spec's compute-intent-diff (STEP 3), files_write is set to ALL
// desired config_files elements unconditionally, so a non-empty config_files
// scope always yields a non-empty intent diff. The empty-intent-diff no-op
// therefore arises when the managed scopes contain no elements to (re)write:
// an applied record and a desired manifest that both declare an empty
// config_files scope (present-but-empty, reconciled to empty) and no other
// managed scope. The intent diff is then empty, and the declared-file-absent
// drift rule keeps drift empty, so apply must emit "nothing to do" without
// opening a transaction.
func TestApplyNoOpWhenConverged(t *testing.T) {
	skipNonLinux(t)
	record := `{
  "meta": { "format_version": 1, "generator": "zypper-declarative 0.4.0", "created_at": "2026-05-29T08:30:00Z", "desired_sha256": "3333333333333333333333333333333333333333333333333333333333333333" },
  "config_files": { "_attributes": null, "_elements": [] }
}`
	applied := writeAppliedRecord(t, record)
	desired := `{
  "meta": { "format_version": 1, "generator": "zypper-declarative 0.4.0", "created_at": "2026-05-29T08:30:00Z", "desired_sha256": "" },
  "config_files": { "_attributes": null, "_elements": [] }
}`
	mp := writeTempFile(t, "desired.json", desired)
	// Describe the actual state from a controlled empty root so the drift
	// check is deterministic and does not probe the host's live /etc.
	actualRoot := t.TempDir()
	r := run(t, "apply", "manifest-path="+mp, "mode=external", "applied-root="+applied, "root="+actualRoot)
	mustExit(t, r.exitCode, 0, "apply no-op when converged")
	mustContain(t, r.stdout, "nothing to do", "apply no-op stdout")
}

// EXAMPLE: idempotent_second_apply (decision portion) — re-running apply with
// the same converged manifest again still computes an empty plan and exits 0
// without a new generation.
func TestApplyIdempotentSecondApply(t *testing.T) {
	skipNonLinux(t)
	record := `{
  "meta": { "format_version": 1, "generator": "zypper-declarative 0.4.0", "created_at": "2026-05-29T08:30:00Z", "desired_sha256": "3333333333333333333333333333333333333333333333333333333333333333" },
  "config_files": { "_attributes": null, "_elements": [] }
}`
	applied := writeAppliedRecord(t, record)
	desired := `{
  "meta": { "format_version": 1, "generator": "zypper-declarative 0.4.0", "created_at": "2026-05-29T08:30:00Z", "desired_sha256": "" },
  "config_files": { "_attributes": null, "_elements": [] }
}`
	mp := writeTempFile(t, "desired.json", desired)
	actualRoot := t.TempDir()
	r1 := run(t, "apply", "manifest-path="+mp, "mode=external", "applied-root="+applied, "root="+actualRoot)
	mustExit(t, r1.exitCode, 0, "apply idempotent first")
	mustContain(t, r1.stdout, "nothing to do", "apply idempotent first stdout")
	r2 := run(t, "apply", "manifest-path="+mp, "mode=external", "applied-root="+applied, "root="+actualRoot)
	mustExit(t, r2.exitCode, 0, "apply idempotent second")
	mustContain(t, r2.stdout, "nothing to do", "apply idempotent second stdout")
}
