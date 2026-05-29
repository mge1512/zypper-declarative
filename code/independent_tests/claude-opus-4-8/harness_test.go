// generated from spec: zypper-declarative.spec.md sha256:b2d0de88fbed1163678e59e931c741b9d999b71f902f6eb01db8790bb813d057
// tests by: claude-opus-4-8
//
// Black-box test harness for the zypper-declarative CLI.
//
// Tests invoke the built binary through the DEPLOYMENT interface using
// os/exec and assert only on stdout, stderr, and exit code. No internal
// package of the implementation is imported. The binary under test is
// located at ../../zypper-declarative relative to this directory, per the
// cli-tool deployment template BINARY-LOCATION: project-root constraint.
package zypperdeclarative_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// binaryPath is the canonical location of the binary under test, relative
// to this test directory (independent_tests/<llm-name>/). Per the cli-tool
// template, this is ../../<binary-name>.
const binaryPath = "../../zypper-declarative"

// binarySource is the entry-point source path the translator must provide,
// relative to the project root. The translator's continuity check verifies
// this path exists in its planned layout.
const binarySource = "./cmd/zypper-declarative"

// TestMain builds the binary at the canonical location before running the
// suite, so the tests can run even if the translator has not built it yet.
func TestMain(m *testing.M) {
	if err := buildBinary(); err != nil {
		// If the build fails, surface it but still allow the run so that
		// individual tests report a clear failure rather than a silent skip.
		os.Stderr.WriteString("TestMain: build failed: " + err.Error() + "\n")
	}
	os.Exit(m.Run())
}

func buildBinary() error {
	cmd := exec.Command("go", "build", "-o", binaryPath, binarySource)
	cmd.Dir = "."
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return &buildError{msg: stderr.String(), err: err}
	}
	return nil
}

type buildError struct {
	msg string
	err error
}

func (b *buildError) Error() string {
	return b.err.Error() + ": " + b.msg
}

// runResult captures the externally-observable outcome of one invocation.
type runResult struct {
	stdout   string
	stderr   string
	exitCode int
}

// run invokes the binary under test with the given arguments and captures
// its streams and exit code.
func run(t *testing.T, args ...string) runResult {
	t.Helper()
	return runWithStdin(t, "", args...)
}

func runWithStdin(t *testing.T, stdin string, args ...string) runResult {
	t.Helper()
	r, _ := runWithTimeout(t, 0, stdin, args...)
	return r
}

// runWithTimeout invokes the binary and, if timeout > 0, kills it and reports
// timedOut=true should it exceed the deadline. A zero timeout waits
// indefinitely. This lets tests that depend on a live read of "/" (whose cost
// is host-dependent) skip rather than hang on a large or rpm-heavy host.
func runWithTimeout(t *testing.T, timeout time.Duration, stdin string, args ...string) (runResult, bool) {
	t.Helper()
	abs, err := filepath.Abs(binaryPath)
	if err != nil {
		t.Fatalf("resolving binary path: %v", err)
	}
	if _, statErr := os.Stat(abs); statErr != nil {
		t.Fatalf("binary under test not found at %s: %v", abs, statErr)
	}
	cmd := exec.Command(abs, args...)
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if startErr := cmd.Start(); startErr != nil {
		t.Fatalf("starting binary: %v", startErr)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	var waitErr error
	if timeout > 0 {
		select {
		case waitErr = <-done:
		case <-time.After(timeout):
			_ = cmd.Process.Kill()
			<-done
			return runResult{}, true
		}
	} else {
		waitErr = <-done
	}

	code := 0
	if waitErr != nil {
		if exitErr, ok := waitErr.(*exec.ExitError); ok {
			code = exitErr.ExitCode()
		} else {
			t.Fatalf("invoking binary: %v", waitErr)
		}
	}
	return runResult{
		stdout:   stdout.String(),
		stderr:   stderr.String(),
		exitCode: code,
	}, false
}

// writeTemp writes content to a temp file with the given name suffix and
// returns its absolute path. The file is cleaned up at test end.
func writeTemp(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("writing temp file %s: %v", p, err)
	}
	return p
}

func assertExit(t *testing.T, r runResult, want int) {
	t.Helper()
	if r.exitCode != want {
		t.Errorf("exit code = %d, want %d\nstdout:\n%s\nstderr:\n%s",
			r.exitCode, want, r.stdout, r.stderr)
	}
}

func assertStdoutContains(t *testing.T, r runResult, sub string) {
	t.Helper()
	if !strings.Contains(r.stdout, sub) {
		t.Errorf("stdout does not contain %q\nstdout:\n%s\nstderr:\n%s",
			sub, r.stdout, r.stderr)
	}
}

func assertStderrContains(t *testing.T, r runResult, sub string) {
	t.Helper()
	if !strings.Contains(r.stderr, sub) {
		t.Errorf("stderr does not contain %q\nstdout:\n%s\nstderr:\n%s",
			sub, r.stdout, r.stderr)
	}
}
