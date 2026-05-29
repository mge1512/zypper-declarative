// generated from spec: zypper-declarative.spec.md sha256:f8ff76ecbc4bbc69a49e2e32b2924da3a64df1ad46196e05ce8c137b684429b2
// tests by: claude-opus-4-8
//
// Black-box test harness for the zypper-declarative CLI binary.
// Tests invoke the built binary through the DEPLOYMENT interface (a CLI binary
// run with bare-word verbs and key=value options) using os/exec, and assert on
// stdout, stderr, and exit code. No internal package of the implementation is
// imported; the binary is the unit under test.

package independent_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// binaryName is the canonical binary the translator builds at the project root.
const binaryName = "zypper-declarative"

// binaryPath is the canonical location per the cli-tool template's
// BINARY-LOCATION constraint, expressed relative to this test directory:
// two directories up from independent_tests/<llm-name>/ to the project root.
const binaryPath = "../../" + binaryName

// TestMain builds the binary at the canonical project-root location before
// running any test, so the suite is self-contained. The translator must place
// the entry point at cmd/zypper-declarative/main.go for this build to succeed.
// The build runs from the project root (two directories up), where go.mod lives,
// and writes the binary there — which is exactly "../../<binary>" relative to
// this test directory, per the BINARY-LOCATION constraint.
func TestMain(m *testing.M) {
	build := exec.Command("go", "build", "-o", binaryName, "./cmd/zypper-declarative")
	build.Dir = "../.." // project root, where go.mod lives
	build.Env = append(os.Environ(), "CGO_ENABLED=0")
	var out bytes.Buffer
	build.Stdout = &out
	build.Stderr = &out
	if err := build.Run(); err != nil {
		// If the build fails the tests cannot run meaningfully; surface it.
		os.Stderr.WriteString("failed to build " + binaryName + ": " + err.Error() + "\n")
		os.Stderr.Write(out.Bytes())
		os.Exit(1)
	}
	os.Exit(m.Run())
}

// runResult holds the observable result of a binary invocation.
type runResult struct {
	stdout   string
	stderr   string
	exitCode int
}

// run invokes the binary with the given arguments and returns the result.
func run(t *testing.T, args ...string) runResult {
	t.Helper()
	abs, err := filepath.Abs(binaryPath)
	if err != nil {
		t.Fatalf("resolving binary path: %v", err)
	}
	cmd := exec.Command(abs, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	code := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		} else {
			t.Fatalf("running binary %v: %v", args, err)
		}
	}
	return runResult{
		stdout:   stdout.String(),
		stderr:   stderr.String(),
		exitCode: code,
	}
}

// writeTemp writes content to a temp file with the given suffix and returns its path.
func writeTemp(t *testing.T, suffix, content string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "fixture"+suffix)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("writing temp fixture: %v", err)
	}
	return p
}

// mkdirAll creates a directory tree, failing the test on error.
func mkdirAll(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("creating directory %s: %v", dir, err)
	}
}

// writeFile writes content to path, failing the test on error.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing file %s: %v", path, err)
	}
}

// readFile reads a file's bytes, failing the test on error.
func readFile(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading file %s: %v", path, err)
	}
	return b
}
