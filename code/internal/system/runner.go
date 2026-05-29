// generated from spec: zypper-declarative.spec.md sha256:58e1636e2de82ab81a5cd3f81d6b3c9ac6a8976e18f9abb2bbd2b2aba56fe4d4
//
// Package system provides the abstract command runner used to drive zypper,
// snapper, systemctl, and rpm by executing their command-line interfaces. This
// keeps CGO_ENABLED=0 and yields a single static binary. The CommandRunner
// interface is an INTERFACES declaration: the production OSCommandRunner runs
// real commands; FakeCommandRunner is a declared test double for unit tests.
package system

import (
	"bytes"
	"os"
	"os/exec"
)

// CommandRunner runs an external command and returns its stdout, stderr, and
// error. All system integration that needs to execute a command goes through
// this interface.
type CommandRunner interface {
	Run(cmd string, args []string) (stdout string, stderr string, err error)
}

// OSCommandRunner is the production CommandRunner. It executes commands with a
// hardened PATH.
type OSCommandRunner struct{}

// Run executes cmd with args using a fixed system PATH.
func (r *OSCommandRunner) Run(cmd string, args []string) (string, string, error) {
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

// FakeCommandRunner is a declared test double for CommandRunner. Independent
// tests must use only declared test doubles, never the production implementation.
// Responses maps a key ("cmd arg0 arg1 ...") to a canned result.
type FakeCommandRunner struct {
	Responses map[string]FakeResult
	Calls     []string
}

// FakeResult is the canned result for one fake command.
type FakeResult struct {
	Stdout string
	Stderr string
	Err    error
}

// Run records the call and returns the canned result for the matching key, or an
// empty success if none is configured.
func (f *FakeCommandRunner) Run(cmd string, args []string) (string, string, error) {
	key := cmd
	for _, a := range args {
		key += " " + a
	}
	f.Calls = append(f.Calls, key)
	if f.Responses != nil {
		if res, ok := f.Responses[key]; ok {
			return res.Stdout, res.Stderr, res.Err
		}
	}
	return "", "", nil
}
