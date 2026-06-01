// generated from spec: zypper-declarative.spec.md sha256:27aee8e374eb3507189bad0b78339109d0116e6a13b55ae4df2ba9a18e769fc4
//
// CommandRunner abstracts the external system tools (rpm, systemctl, zypper,
// snapper) the live-state reader and convergence domains drive via os/exec.
// This is the INTERFACES seam: a production OSCommandRunner and a test-double
// FakeCommandRunner. Independent (black-box) tests use the binary itself and
// never this seam; the seam exists for in-tree wiring and is fully implemented
// in production form so command-dependent code is never silently empty.
package state

import (
	"bytes"
	"os"
	"os/exec"
)

// CommandRunner runs an external command and returns stdout, stderr, and error.
type CommandRunner interface {
	Run(cmd string, args []string) (stdout string, stderr string, err error)
}

// OSCommandRunner is the production CommandRunner. It runs the command with a
// sanitised PATH and captures stdout/stderr separately.
type OSCommandRunner struct{}

// Run executes cmd with args under a fixed PATH and returns captured output.
func (r *OSCommandRunner) Run(cmd string, args []string) (string, string, error) {
	oldPath := os.Getenv("PATH")
	_ = os.Setenv("PATH", "/sbin:/bin:/usr/bin:/usr/sbin")
	defer os.Setenv("PATH", oldPath)

	c := exec.Command(cmd, args...)
	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr
	err := c.Run()
	return stdout.String(), stderr.String(), err
}

// FakeResult is a canned response for the FakeCommandRunner.
type FakeResult struct {
	Stdout string
	Stderr string
	Err    error
}

// FakeCommandRunner is the declared test double. It returns canned results keyed
// by the first argument (or the command name), and records invocations. Tests
// in-tree may use it; the independent black-box suite does not.
type FakeCommandRunner struct {
	Results map[string]FakeResult
	Default FakeResult
	Calls   [][]string
}

// Run returns the canned result for the command's first arg, else Default.
func (f *FakeCommandRunner) Run(cmd string, args []string) (string, string, error) {
	f.Calls = append(f.Calls, append([]string{cmd}, args...))
	key := cmd
	if len(args) > 0 {
		key = args[0]
	}
	if f.Results != nil {
		if res, ok := f.Results[key]; ok {
			return res.Stdout, res.Stderr, res.Err
		}
	}
	return f.Default.Stdout, f.Default.Stderr, f.Default.Err
}
