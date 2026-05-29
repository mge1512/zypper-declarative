// generated from spec: zypper-declarative.spec.md sha256:f8ff76ecbc4bbc69a49e2e32b2924da3a64df1ad46196e05ce8c137b684429b2
//
// Command execution boundary. OSCommandRunner runs external tools (rpm,
// systemctl) with a sanitised PATH. Repositories are read as files directly and
// need no command runner.

package state

import (
	"bytes"
	"os"
	"os/exec"
	"time"
)

// CommandRunner runs an external command and returns its stdout, stderr, and error.
type CommandRunner interface {
	Run(cmd string, args []string, dir string) (stdout string, stderr string, err error)
}

// OSCommandRunner is the production CommandRunner. Per the cli-tool Go hints it
// is implemented in full (never a stub): a stub returning ("","",nil) would make
// every command-dependent scope silently empty.
type OSCommandRunner struct{}

// Run executes cmd with args, with PATH constrained to standard system paths.
func (r *OSCommandRunner) Run(cmd string, args []string, dir string) (string, string, error) {
	oldPath := os.Getenv("PATH")
	_ = os.Setenv("PATH", "/sbin:/bin:/usr/bin:/usr/sbin")
	defer func() { _ = os.Setenv("PATH", oldPath) }()

	c := exec.Command(cmd, args...)
	if dir != "" {
		c.Dir = dir
	}
	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr
	err := c.Run()
	return stdout.String(), stderr.String(), err
}

func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}
