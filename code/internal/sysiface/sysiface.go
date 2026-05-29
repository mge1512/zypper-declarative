// generated from spec: zypper-declarative.spec.md sha256:714e75ff672557d2c7344736a5f36b52afa37e0f565b07de83e1b18cc4492014
//
// Package sysiface declares the abstract dependencies from the spec INTERFACES
// section (package manager, snapshot/filesystem, init system, transaction
// mechanism) as Go interfaces, with a production exec-based implementation and
// independent test-double implementations. Independent tests use only the
// declared test doubles, never the production implementation.
package sysiface

import (
	"bytes"
	"os"
	"os/exec"
)

// CommandRunner runs an external command within a sanitised PATH and returns
// its stdout, stderr, and error. It is the seam through which the production
// implementation drives zypper, snapper, systemctl, and rpm.
type CommandRunner interface {
	Run(cmd string, args ...string) (stdout string, stderr string, err error)
}

// OSCommandRunner is the production CommandRunner. It is implemented in full
// (never a stub) so command-dependent modules behave correctly.
type OSCommandRunner struct{}

// Run executes cmd with a fixed system PATH so behaviour does not depend on the
// caller's environment.
func (r *OSCommandRunner) Run(cmd string, args ...string) (string, string, error) {
	oldPath := os.Getenv("PATH")
	_ = os.Setenv("PATH", "/sbin:/bin:/usr/bin:/usr/sbin")
	defer func() { _ = os.Setenv("PATH", oldPath) }()

	c := exec.Command(cmd, args...)
	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr
	err := c.Run()
	return stdout.String(), stderr.String(), err
}

// FakeCommandRunner is a test-double CommandRunner. Responses maps a command
// key to a canned result; Calls records every invocation in order.
type FakeCommandRunner struct {
	Responses map[string]FakeResult
	Calls     []string
}

// FakeResult is a canned command result.
type FakeResult struct {
	Stdout string
	Stderr string
	Err    error
}

// Run records the call and returns the canned result for cmd, or empty success.
func (r *FakeCommandRunner) Run(cmd string, args ...string) (string, string, error) {
	key := cmd
	for _, a := range args {
		key += " " + a
	}
	r.Calls = append(r.Calls, key)
	if r.Responses != nil {
		if res, ok := r.Responses[cmd]; ok {
			return res.Stdout, res.Stderr, res.Err
		}
		if res, ok := r.Responses[key]; ok {
			return res.Stdout, res.Stderr, res.Err
		}
	}
	return "", "", nil
}
