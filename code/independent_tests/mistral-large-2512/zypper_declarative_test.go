// generated from spec: zypper-declarative.spec.md sha256:714e75ff672557d2c7344736a5f36b52afa37e0f565b07de83e1b18cc4492014
// tests by: mistral-large-2512

package mistral_large_2512

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const (
	binaryPath  = "../../zypper-declarative"
	testdataDir = "testdata"
)

// TestApply_Success tests a successful `apply` with no prior state.
func TestApply_Success(t *testing.T) {
	// Setup: Copy a desired manifest to a temp dir.
	desiredManifest := filepath.Join(testdataDir, "desired_nginx.json")
	tempDir := t.TempDir()
	manifestPath := filepath.Join(tempDir, "manifest.json")
	copyFile(t, desiredManifest, manifestPath)

	// Build the binary if it doesn't exist (for test-author mode).
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		buildBinary(t)
	}

	cmd := exec.Command(binaryPath, "apply", "--manifest-path="+manifestPath)
	stdout, stderr, exitCode := runCommand(t, cmd)

	// Assertions.
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d. stderr: %s", exitCode, stderr)
	}
	if !strings.Contains(stdout, "nothing to do") && !strings.Contains(stdout, "convergence complete") {
		t.Errorf("unexpected stdout: %s", stdout)
	}
}

// TestApply_InvalidManifest tests `apply` with an invalid manifest (schema violation).
func TestApply_InvalidManifest(t *testing.T) {
	invalidManifest := filepath.Join(testdataDir, "invalid_manifest.json")
	cmd := exec.Command(binaryPath, "apply", "--manifest-path="+invalidManifest)
	_, stderr, exitCode := runCommand(t, cmd)

	if exitCode != 1 {
		t.Fatalf("expected exit 1, got %d", exitCode)
	}
	if !strings.Contains(stderr, "manifest error") {
		t.Errorf("expected 'manifest error' in stderr, got: %s", stderr)
	}
}

// TestDiff_Success tests `diff` with a desired manifest and no prior state.
func TestDiff_Success(t *testing.T) {
	desiredManifest := filepath.Join(testdataDir, "desired_nginx.json")
	cmd := exec.Command(binaryPath, "diff", "--manifest-path="+desiredManifest)
	stdout, stderr, exitCode := runCommand(t, cmd)

	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d. stderr: %s", exitCode, stderr)
	}
	if !strings.Contains(stdout, "packages_install") {
		t.Errorf("expected 'packages_install' in stdout, got: %s", stdout)
	}
}

// TestVerify_NoAppliedRecord tests `verify` when no applied record exists.
func TestVerify_NoAppliedRecord(t *testing.T) {
	cmd := exec.Command(binaryPath, "verify")
	_, stderr, exitCode := runCommand(t, cmd)

	if exitCode != 2 {
		t.Fatalf("expected exit 2, got %d", exitCode)
	}
	if !strings.Contains(stderr, "no declaration applied") {
		t.Errorf("expected 'no declaration applied' in stderr, got: %s", stderr)
	}
}

// TestStatus_NoAppliedRecord tests `status` when no applied record exists.
func TestStatus_NoAppliedRecord(t *testing.T) {
	cmd := exec.Command(binaryPath, "status")
	stdout, stderr, exitCode := runCommand(t, cmd)

	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d. stderr: %s", exitCode, stderr)
	}
	if !strings.Contains(stdout, "no declaration applied") {
		t.Errorf("expected 'no declaration applied' in stdout, got: %s", stdout)
	}
}

// TestDescribe_Success tests `describe` with default args.
func TestDescribe_Success(t *testing.T) {
	cmd := exec.Command(binaryPath, "describe")
	stdout, stderr, exitCode := runCommand(t, cmd)

	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d. stderr: %s", exitCode, stderr)
	}

	// Validate the output is a valid Manifest.
	var manifest map[string]interface{}
	if err := json.Unmarshal([]byte(stdout), &manifest); err != nil {
		t.Fatalf("describe output is not valid JSON: %v", err)
	}
	if _, ok := manifest["meta"]; !ok {
		t.Errorf("describe output missing 'meta' field")
	}
}

// TestDescribe_YAML tests `describe` with YAML output.
func TestDescribe_YAML(t *testing.T) {
	outFile := filepath.Join(t.TempDir(), "state.yaml")
	cmd := exec.Command(binaryPath, "describe", "--format=yaml", "--out="+outFile)
	_, stderr, exitCode := runCommand(t, cmd)

	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d. stderr: %s", exitCode, stderr)
	}

	// Validate the output file exists and is non-empty.
	content, err := os.ReadFile(outFile)
	if err != nil || len(content) == 0 {
		t.Fatalf("YAML output file is missing or empty")
	}
}

// Helper: Run a command and return stdout, stderr, and exit code.
func runCommand(t *testing.T, cmd *exec.Cmd) (string, string, int) {
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("failed to run command: %v", err)
		}
	}
	return stdoutBuf.String(), stderrBuf.String(), exitCode
}

// Helper: Copy a file.
func copyFile(t *testing.T, src, dst string) {
	in, err := os.Open(src)
	if err != nil {
		t.Fatal(err)
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		t.Fatal(err)
	}
}

// Helper: Build the binary if it doesn't exist (for test-author mode).
func buildBinary(t *testing.T) {
	cmd := exec.Command("go", "build", "-o", binaryPath, "../../cmd/zypper-declarative/main.go")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to build binary: %v", err)
	}
}
