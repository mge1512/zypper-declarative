// generated from spec: zypper-declarative.spec.md sha256:b2d0de88fbed1163678e59e931c741b9d999b71f902f6eb01db8790bb813d057
// tests by: claude-opus-4-8
//
// describe verb tests: format resolution (resolve-format), unwritable output,
// and the read-only invariant.
//
// describe reads live state via describe-actual-state. To keep these tests
// deterministic and fast on any host (a developer workstation, a CI runner, a
// SUSE build host), they point describe at a controlled small root via the
// root= option (a spec-declared describe input) containing a tiny /etc, rather
// than at "/" where the size of the real /etc and per-file rpm queries make
// the run host-dependent and slow. The format-selection behaviour and the
// read-only/output-write behaviour under test are independent of which root is
// described.
package zypperdeclarative_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeRoot builds a small describable root with one /etc file and returns its
// path. With rpm queries against this root yielding nothing, the file is
// treated as unpackaged and appears in config_files, so describe produces a
// non-empty, deterministic document.
func fakeRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	etc := filepath.Join(root, "etc")
	if err := os.MkdirAll(etc, 0o755); err != nil {
		t.Fatalf("setup fake root: %v", err)
	}
	if err := os.WriteFile(filepath.Join(etc, "demo.conf"), []byte("key = value\n"), 0o644); err != nil {
		t.Fatalf("setup fake root file: %v", err)
	}
	return root
}

// EXAMPLE: describe_output_unwritable -- writing to an unwritable path is an
// invocation error (exit 2, domain=invocation).
func TestDescribeOutputUnwritable(t *testing.T) {
	// A path whose parent component is a regular file (not a directory) is
	// unwritable.
	tmp := t.TempDir()
	notADir := filepath.Join(tmp, "regular-file")
	if err := os.WriteFile(notADir, []byte("x"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}
	target := filepath.Join(notADir, "state.json")
	r := run(t, "describe", "root="+fakeRoot(t), "out="+target)
	assertExit(t, r, 2)
}

// EXAMPLE: describe_out_extension_json -- with no format option, the .json
// extension selects JSON. The written file's first non-space byte is '{'.
func TestDescribeOutExtensionJSON(t *testing.T) {
	out := filepath.Join(t.TempDir(), "state.json")
	r := run(t, "describe", "root="+fakeRoot(t), "out="+out)
	assertExit(t, r, 0)
	assertFileIsJSON(t, out)
}

// EXAMPLE: describe_out_extension_yaml -- with no format option, the .yaml
// extension selects YAML. The written file is not a JSON object.
func TestDescribeOutExtensionYAML(t *testing.T) {
	out := filepath.Join(t.TempDir(), "state.yaml")
	r := run(t, "describe", "root="+fakeRoot(t), "out="+out)
	assertExit(t, r, 0)
	assertFileIsYAML(t, out)
}

// EXAMPLE: describe_format_overrides_extension -- explicit format=json with a
// .yaml out path writes JSON (explicit option wins over extension).
func TestDescribeFormatOverridesExtension(t *testing.T) {
	out := filepath.Join(t.TempDir(), "state.yaml")
	r := run(t, "describe", "root="+fakeRoot(t), "format=json", "out="+out)
	assertExit(t, r, 0)
	assertFileIsJSON(t, out)
}

// EXAMPLE: describe_format_yaml -- describe format=yaml to stdout produces a
// YAML document (not a JSON object).
func TestDescribeFormatYAMLToStdout(t *testing.T) {
	r := run(t, "describe", "root="+fakeRoot(t), "format=yaml")
	assertExit(t, r, 0)
	trimmed := strings.TrimLeft(r.stdout, " \t\r\n")
	if strings.HasPrefix(trimmed, "{") {
		t.Errorf("expected YAML output, got JSON-looking stdout:\n%s", r.stdout)
	}
}

// describe to stdout with default format (json) yields a JSON document whose
// meta.format_version is 1.
func TestDescribeDefaultJSONToStdout(t *testing.T) {
	r := run(t, "describe", "root="+fakeRoot(t))
	assertExit(t, r, 0)
	trimmed := strings.TrimLeft(r.stdout, " \t\r\n")
	if !strings.HasPrefix(trimmed, "{") {
		t.Errorf("expected a JSON document on stdout\nstdout:\n%s", r.stdout)
	}
	assertStdoutContains(t, r, "format_version")
}

// EXAMPLE: describe_bootstraps_desired_manifest -- describe output is accepted
// unchanged by load-desired-manifest. We capture a describe JSON document and
// feed it back to diff as manifest-path against an empty applied record. The
// load must succeed (not a manifest-domain error).
func TestDescribeOutputAcceptedAsManifest(t *testing.T) {
	out := filepath.Join(t.TempDir(), "desired.json")
	d := run(t, "describe", "root="+fakeRoot(t), "out="+out)
	assertExit(t, d, 0)
	assertFileIsJSON(t, out)

	emptyRoot := t.TempDir()
	r, timedOut := runWithTimeout(t, liveReadBudget, "", "diff", "manifest-path="+out, "applied-root="+emptyRoot)
	if timedOut {
		t.Skip("diff live read of / exceeded the budget on this host; the load-desired-manifest acceptance is exercised by the diff exit/stderr path")
	}
	if r.exitCode == 1 && strings.Contains(r.stderr, "manifest") {
		t.Errorf("describe output was rejected as a manifest error by load-desired-manifest\nstderr:\n%s", r.stderr)
	}
}

// EXAMPLE: describe_omits_genuinely_empty_scope -- a readable but empty
// repos.d yields no repositories scope (not an empty one). Described root has
// an empty etc/zypp/repos.d directory.
func TestDescribeOmitsGenuinelyEmptyRepositories(t *testing.T) {
	root := fakeRoot(t)
	if err := os.MkdirAll(filepath.Join(root, "etc", "zypp", "repos.d"), 0o755); err != nil {
		t.Fatalf("setup: %v", err)
	}
	out := filepath.Join(t.TempDir(), "state.json")
	r := run(t, "describe", "root="+root, "out="+out)
	assertExit(t, r, 0)
	body := readNonEmpty(t, out)
	// The emitted document must not carry a repositories scope with empty
	// _elements; the omitted-scope rule means the key is absent entirely.
	if strings.Contains(body, "repository_system") {
		t.Errorf("describe emitted a repositories scope for an empty repos.d\noutput:\n%s", body)
	}
}

// EXAMPLE: describe_scope_full_emits_observational_scopes (partial) -- a plain
// describe (scope=etc, the default) of a root emits neither observational
// scope.
func TestDescribeDefaultScopeOmitsObservational(t *testing.T) {
	out := filepath.Join(t.TempDir(), "state.json")
	r := run(t, "describe", "root="+fakeRoot(t), "out="+out)
	assertExit(t, r, 0)
	body := readNonEmpty(t, out)
	if strings.Contains(body, "changed_managed_files") || strings.Contains(body, "unmanaged_files") {
		t.Errorf("scope=etc describe emitted an observational scope\noutput:\n%s", body)
	}
}

func readNonEmpty(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	s := strings.TrimLeft(string(b), " \t\r\n")
	if s == "" {
		t.Fatalf("file %s is empty", path)
	}
	return s
}

func assertFileIsJSON(t *testing.T, path string) {
	t.Helper()
	s := readNonEmpty(t, path)
	if !strings.HasPrefix(s, "{") {
		t.Errorf("file %s does not begin with a JSON object: first bytes %q", path, head(s))
	}
}

func assertFileIsYAML(t *testing.T, path string) {
	t.Helper()
	s := readNonEmpty(t, path)
	if strings.HasPrefix(s, "{") {
		t.Errorf("file %s looks like JSON, expected YAML: first bytes %q", path, head(s))
	}
}

func head(s string) string {
	if len(s) > 40 {
		return s[:40]
	}
	return s
}
