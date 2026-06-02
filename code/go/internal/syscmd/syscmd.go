// generated from spec: zypper-declarative.spec.md sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
//
// Package syscmd provides the command-runner interface used to drive external
// tools (rpm, zypper, snapper, systemctl, update-alternatives) plus the
// production OSCommandRunner and a test double, per the spec INTERFACES section.
package syscmd

import (
	"bytes"
	"os"
	"os/exec"
)

// CommandRunner abstracts execution of an external command. Implementations
// return stdout, stderr, and an error (the process exit error, if any).
type CommandRunner interface {
	Run(cmd string, args []string) (stdout string, stderr string, err error)
}

// OSCommandRunner is the production CommandRunner: it executes the command with a
// fixed safe PATH and captures stdout/stderr.
type OSCommandRunner struct{}

// Run executes cmd with args, returning captured stdout, stderr, and the run error.
func (r *OSCommandRunner) Run(cmd string, args []string) (string, string, error) {
	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", "/sbin:/bin:/usr/bin:/usr/sbin")
	defer os.Setenv("PATH", oldPath)

	c := exec.Command(cmd, args...)
	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr
	err := c.Run()
	return stdout.String(), stderr.String(), err
}

// FakeCommandRunner is a test double returning canned responses keyed by command
// name. Independent tests use only declared test doubles, never the production
// runner. (Black-box independent tests drive the binary directly; this double is
// available for in-tree unit testing per the INTERFACES requirement.)
type FakeCommandRunner struct {
	Responses map[string]FakeResponse
}

// FakeResponse is a canned command result.
type FakeResponse struct {
	Stdout string
	Stderr string
	Err    error
}

// Run returns the canned response for cmd, or empty output and no error.
func (f *FakeCommandRunner) Run(cmd string, args []string) (string, string, error) {
	if f.Responses != nil {
		if r, ok := f.Responses[cmd]; ok {
			return r.Stdout, r.Stderr, r.Err
		}
	}
	return "", "", nil
}
