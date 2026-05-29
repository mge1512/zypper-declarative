// generated from spec: zypper-declarative.spec.md sha256:714e75ff672557d2c7344736a5f36b52afa37e0f565b07de83e1b18cc4492014
// tests by: claude-opus-4-8
//
// Invocation, version, help, and argument-handling tests. These paths
// require no privilege and no live system tooling.
package independent_tests

import (
	"strings"
	"testing"
)

func TestVersionPrintsNameAndSpecHash(t *testing.T) {
	r := run(t, "--version")
	mustExit(t, r.exitCode, 0, "--version")
	// MILESTONE 0.0.0 acceptance: "^zypper-declarative "
	if !strings.HasPrefix(strings.TrimSpace(r.stdout), "zypper-declarative ") &&
		!strings.HasPrefix(strings.TrimSpace(r.stderr), "zypper-declarative ") {
		t.Errorf("--version: expected output beginning %q, got stdout=%q stderr=%q",
			"zypper-declarative ", r.stdout, r.stderr)
	}
	combined := r.stdout + r.stderr
	mustContain(t, combined, "spec:714e75ff672557d2c7344736a5f36b52afa37e0f565b07de83e1b18cc4492014",
		"--version spec hash")
}

func TestHelpPrintsUsage(t *testing.T) {
	r := run(t, "--help")
	mustExit(t, r.exitCode, 0, "--help")
	combined := r.stdout + r.stderr
	mustContain(t, strings.ToLower(combined), "usage:", "--help usage")
}

// EXAMPLE: status_unknown_argument
func TestStatusUnknownArgument(t *testing.T) {
	r := run(t, "status", "--frobnicate")
	mustExit(t, r.exitCode, 2, "status --frobnicate")
	mustContain(t, strings.ToLower(r.stderr), "usage", "status unknown arg usage on stderr")
}

// EXAMPLE: describe_unknown_format
func TestDescribeUnknownFormat(t *testing.T) {
	skipNonLinux(t)
	r := run(t, "describe", "format=toml")
	mustExit(t, r.exitCode, 2, "describe format=toml")
	mustContain(t, strings.ToLower(r.stderr), "usage", "describe unknown format usage on stderr")
}

func TestUnknownVerbIsInvocationError(t *testing.T) {
	r := run(t, "frobnicate")
	mustExit(t, r.exitCode, 2, "unknown verb")
}

func TestNoVerbIsInvocationError(t *testing.T) {
	r := run(t)
	mustExit(t, r.exitCode, 2, "no verb")
}
