// generated from spec: zypper-declarative.spec.md sha256:b2d0de88fbed1163678e59e931c741b9d999b71f902f6eb01db8790bb813d057
//
// Package sysexec provides the single external-command runner interface used
// by the convergence and live-state code paths, plus its production
// implementation and an in-memory test double.
package sysexec

import (
	"bytes"
	"os"
	"os/exec"
)

// CommandRunner executes an external command and returns its stdout, stderr,
// and error. It is the seam between the tool and the systems it drives
// (zypper, rpm, systemctl, snapper). Production code uses OSCommandRunner; the
// declared test double is FakeCommandRunner.
type CommandRunner interface {
	Run(cmd string, args []string) (stdout string, stderr string, err error)
}

// OSCommandRunner runs commands against the real operating system.
type OSCommandRunner struct{}

// Run executes cmd with args using a sanitised PATH and captures its output.
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

// Call records a single command invocation for the test double.
type Call struct {
	Cmd  string
	Args []string
}

// Response is a scripted result keyed by command name in the test double.
type Response struct {
	Stdout string
	Stderr string
	Err    error
}

// FakeCommandRunner is the declared test double. It records calls and replies
// from a scripted map keyed by command name. Independent tests that need to
// exercise convergence in-process use this; production never does.
type FakeCommandRunner struct {
	Responses map[string]Response
	Calls     []Call
}

// NewFakeCommandRunner constructs an empty fake.
func NewFakeCommandRunner() *FakeCommandRunner {
	return &FakeCommandRunner{Responses: map[string]Response{}}
}

// Run records the call and returns the scripted response for cmd, or empty
// output and no error if none is scripted.
func (f *FakeCommandRunner) Run(cmd string, args []string) (string, string, error) {
	f.Calls = append(f.Calls, Call{Cmd: cmd, Args: args})
	if resp, ok := f.Responses[cmd]; ok {
		return resp.Stdout, resp.Stderr, resp.Err
	}
	return "", "", nil
}
