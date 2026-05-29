// generated from spec: zypper-declarative.spec.md sha256:714e75ff672557d2c7344736a5f36b52afa37e0f565b07de83e1b18cc4492014
// tests by: claude-opus-4-8
//
// Black-box test harness for the zypper-declarative CLI binary.
// Tests invoke the binary via exec.Command per the spec DEPLOYMENT section
// (key=value argument style) and assert on stdout, stderr, and exit code.
// No internal package of the implementation is imported.
package independent_tests

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// binaryName is the canonical binary name from the spec title.
const binaryName = "zypper-declarative"

// binaryPath is the canonical discovery path mandated by the cli-tool
// template's BINARY-LOCATION constraint: two directories up from
// independent_tests/<llm-name>/ to the project root.
func binaryPath() string {
	return filepath.Join("..", "..", binaryName)
}

// TestMain builds the binary at the canonical project-root location from
// the entry point the translator places at cmd/zypper-declarative/main.go.
func TestMain(m *testing.M) {
	bp := binaryPath()
	if _, err := os.Stat(bp); err != nil {
		cmd := exec.Command("go", "build", "-o", bp, "./cmd/zypper-declarative")
		cmd.Dir = filepath.Join("..", "..")
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out
		if berr := cmd.Run(); berr != nil {
			// Cannot build; tests that need the binary will fail clearly.
			os.Stderr.WriteString("build failed: " + berr.Error() + "\n" + out.String() + "\n")
		}
	}
	os.Exit(m.Run())
}

// runResult captures one invocation of the binary.
type runResult struct {
	stdout   string
	stderr   string
	exitCode int
}

// run invokes the binary directly: zypper-declarative <args...>.
func run(t *testing.T, args ...string) runResult {
	t.Helper()
	bp := binaryPath()
	if _, err := os.Stat(bp); err != nil {
		t.Fatalf("binary not found at %s: %v", bp, err)
	}
	cmd := exec.Command(bp, args...)
	var so, se bytes.Buffer
	cmd.Stdout = &so
	cmd.Stderr = &se
	err := cmd.Run()
	code := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		} else {
			t.Fatalf("failed to run %s %v: %v", bp, args, err)
		}
	}
	return runResult{stdout: so.String(), stderr: se.String(), exitCode: code}
}

func skipNonLinux(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skipf("spec targets Linux only; running on %s", runtime.GOOS)
	}
}

func writeTempFile(t *testing.T, name, content string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	return p
}

func mustContain(t *testing.T, haystack, needle, label string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Errorf("%s: expected to contain %q\ngot:\n%s", label, needle, haystack)
	}
}

func mustExit(t *testing.T, got, want int, label string) {
	t.Helper()
	if got != want {
		t.Errorf("%s: expected exit code %d, got %d", label, want, got)
	}
}

// validManifestJSON returns a structurally complete desired manifest in
// canonical JSON, parameterised so individual tests can vary content.
func validManifestJSON() string {
	return `{
  "meta": {
    "format_version": 1,
    "generator": "zypper-declarative 0.4.0",
    "created_at": "2026-05-29T08:30:00Z",
    "desired_sha256": ""
  },
  "repositories": {
    "_attributes": { "repository_system": "zypp" },
    "_elements": [
      {
        "alias": "sl-micro-6.2-pinned",
        "name": "SL Micro 6.2 (pinned)",
        "url": "https://internal.example/obs/SLMicro:6.2:pinned/standard",
        "type": "rpm-md",
        "enabled": true,
        "gpgcheck": true,
        "autorefresh": false,
        "priority": 99
      }
    ]
  },
  "packages": {
    "_attributes": { "package_system": "rpm" },
    "_elements": [
      { "name": "nginx", "version": "", "release": "", "arch": "" }
    ]
  },
  "services": {
    "_attributes": { "init_system": "systemd" },
    "_elements": [
      { "name": "nginx.service", "state": "enabled" }
    ]
  },
  "config_files": {
    "_attributes": null,
    "_elements": [
      {
        "name": "/etc/nginx/nginx.conf",
        "type": "file",
        "mode": "0644",
        "user": "root",
        "group": "root",
        "sha256": "0000000000000000000000000000000000000000000000000000000000000000",
        "content_ref": "files/etc/nginx/nginx.conf",
        "package_name": ""
      }
    ]
  }
}`
}
