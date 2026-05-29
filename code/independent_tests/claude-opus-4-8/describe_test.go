// generated from spec: zypper-declarative.spec.md sha256:f8ff76ecbc4bbc69a49e2e32b2924da3a64df1ad46196e05ce8c137b684429b2
// tests by: claude-opus-4-8
//
// Tests for the describe verb, exercised against a controlled root= tree so the
// asserted paths need no privilege and no live system state. The repositories
// scope is read from <root>/etc/zypp/repos.d/*.repo (world-readable INI files),
// which is the spec's pinned source. on-unreadable=warn is used so the other
// scopes (rpmdb, systemd, /etc) under an empty synthetic root are omitted with
// diagnostics rather than failing the run, isolating the repositories-scope and
// format-resolution assertions.

package independent_test

import (
	"path/filepath"
	"strings"
	"testing"
)

// twoRepoRoot builds a synthetic root containing two readable .repo files under
// etc/zypp/repos.d and returns the root path.
func twoRepoRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	reposd := filepath.Join(root, "etc", "zypp", "repos.d")
	mkdirAll(t, reposd)
	repo1 := "[sl-micro-6.2-pinned]\n" +
		"name=SL Micro 6.2 (pinned)\n" +
		"baseurl=https://internal.example/obs/SLMicro/standard\n" +
		"type=rpm-md\n" +
		"enabled=1\n" +
		"gpgcheck=1\n" +
		"autorefresh=0\n" +
		"priority=99\n"
	repo2 := "[update]\n" +
		"name=Update repo\n" +
		"baseurl=https://internal.example/obs/Update/standard\n" +
		"type=rpm-md\n" +
		"enabled=1\n" +
		"gpgcheck=1\n" +
		"autorefresh=1\n" +
		"priority=100\n"
	writeFile(t, filepath.Join(reposd, "pinned.repo"), repo1)
	writeFile(t, filepath.Join(reposd, "update.repo"), repo2)
	return root
}

// emptyReposRoot builds a synthetic root whose repos.d exists and is readable
// but contains no .repo files.
func emptyReposRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	mkdirAll(t, filepath.Join(root, "etc", "zypp", "repos.d"))
	return root
}

// EXAMPLE: describe_repositories_from_reposd
// Two readable .repo files -> the repositories scope contains two records and
// the scope is not empty; exit 0.
func TestDescribeRepositoriesFromReposd(t *testing.T) {
	root := twoRepoRoot(t)
	r := run(t, "describe", "root="+root, "on-unreadable=warn")
	if r.exitCode != 0 {
		t.Fatalf("describe repos: exit = %d, want 0; stdout=%q stderr=%q", r.exitCode, r.stdout, r.stderr)
	}
	if !strings.Contains(r.stdout, "repositories") {
		t.Fatalf("describe repos: stdout %q does not contain a repositories scope", r.stdout)
	}
	if !strings.Contains(r.stdout, "sl-micro-6.2-pinned") {
		t.Errorf("describe repos: stdout %q does not contain first repo alias", r.stdout)
	}
	if !strings.Contains(r.stdout, "update") {
		t.Errorf("describe repos: stdout %q does not contain second repo alias", r.stdout)
	}
}

// EXAMPLE: describe_emits_manifest (shape assertion on the produced document):
// stdout is a JSON document with meta.format_version = 1.
func TestDescribeEmitsManifestShape(t *testing.T) {
	root := twoRepoRoot(t)
	r := run(t, "describe", "root="+root, "on-unreadable=warn")
	if r.exitCode != 0 {
		t.Fatalf("describe shape: exit = %d, want 0; stderr=%q", r.exitCode, r.stderr)
	}
	if !strings.HasPrefix(strings.TrimSpace(r.stdout), "{") {
		t.Errorf("describe shape: stdout %q is not a JSON document", r.stdout)
	}
	if !strings.Contains(r.stdout, "\"format_version\"") || !strings.Contains(r.stdout, "1") {
		t.Errorf("describe shape: stdout %q does not contain meta.format_version = 1", r.stdout)
	}
}

// EXAMPLE: describe_omits_genuinely_empty_scope
// repos.d is readable but contains no .repo files -> the output omits the
// repositories scope rather than emitting empty _elements; exit 0.
func TestDescribeOmitsGenuinelyEmptyScope(t *testing.T) {
	root := emptyReposRoot(t)
	r := run(t, "describe", "root="+root, "on-unreadable=warn")
	if r.exitCode != 0 {
		t.Fatalf("describe empty repos: exit = %d, want 0; stdout=%q stderr=%q", r.exitCode, r.stdout, r.stderr)
	}
	// The repositories scope must not appear with empty _elements. Since no
	// repos exist and the source is readable, the scope is omitted entirely.
	if strings.Contains(r.stdout, "repository_system") {
		t.Errorf("describe empty repos: stdout %q emitted a repositories scope for a genuinely-empty source", r.stdout)
	}
}

// EXAMPLE: describe_out_extension_yaml
// No format option; out=...yaml -> resolve-format selects yaml; the file holds a
// YAML document (not starting with '{'); exit 0.
func TestDescribeOutExtensionYAML(t *testing.T) {
	root := twoRepoRoot(t)
	out := filepath.Join(t.TempDir(), "state.yaml")
	r := run(t, "describe", "root="+root, "on-unreadable=warn", "out="+out)
	if r.exitCode != 0 {
		t.Fatalf("describe out yaml: exit = %d, want 0; stdout=%q stderr=%q", r.exitCode, r.stdout, r.stderr)
	}
	content := readFileTrim(t, out)
	if strings.HasPrefix(content, "{") {
		t.Errorf("describe out yaml: %s holds a JSON document, want YAML: %q", out, content)
	}
	if !strings.Contains(content, "sl-micro-6.2-pinned") {
		t.Errorf("describe out yaml: %s does not contain the repo alias: %q", out, content)
	}
}

// EXAMPLE: describe_out_extension_json
// No format option; out=...json -> resolve-format selects json; the file holds a
// JSON document (starts with '{'); exit 0.
func TestDescribeOutExtensionJSON(t *testing.T) {
	root := twoRepoRoot(t)
	out := filepath.Join(t.TempDir(), "state.json")
	r := run(t, "describe", "root="+root, "on-unreadable=warn", "out="+out)
	if r.exitCode != 0 {
		t.Fatalf("describe out json: exit = %d, want 0; stdout=%q stderr=%q", r.exitCode, r.stdout, r.stderr)
	}
	content := readFileTrim(t, out)
	if !strings.HasPrefix(content, "{") {
		t.Errorf("describe out json: %s does not hold a JSON document: %q", out, content)
	}
}

// EXAMPLE: describe_format_overrides_extension
// format=json out=...yaml -> resolve-format returns json (explicit wins); the
// file holds a JSON document; exit 0.
func TestDescribeFormatOverridesExtension(t *testing.T) {
	root := twoRepoRoot(t)
	out := filepath.Join(t.TempDir(), "state.yaml")
	r := run(t, "describe", "root="+root, "on-unreadable=warn", "format=json", "out="+out)
	if r.exitCode != 0 {
		t.Fatalf("describe format overrides: exit = %d, want 0; stdout=%q stderr=%q", r.exitCode, r.stdout, r.stderr)
	}
	content := readFileTrim(t, out)
	if !strings.HasPrefix(content, "{") {
		t.Errorf("describe format overrides: %s does not hold a JSON document despite format=json: %q", out, content)
	}
}

// EXAMPLE: describe_format_yaml
// format=yaml -> stdout is a YAML document (not starting with '{'); exit 0.
func TestDescribeFormatYAML(t *testing.T) {
	root := twoRepoRoot(t)
	r := run(t, "describe", "root="+root, "on-unreadable=warn", "format=yaml")
	if r.exitCode != 0 {
		t.Fatalf("describe format=yaml: exit = %d, want 0; stdout=%q stderr=%q", r.exitCode, r.stdout, r.stderr)
	}
	if strings.HasPrefix(strings.TrimSpace(r.stdout), "{") {
		t.Errorf("describe format=yaml: stdout %q is a JSON document, want YAML", r.stdout)
	}
}

// EXAMPLE: describe_output_unwritable
// out points into a non-existent/unwritable directory -> domain=invocation, exit 2.
func TestDescribeOutputUnwritable(t *testing.T) {
	root := twoRepoRoot(t)
	r := run(t, "describe", "root="+root, "on-unreadable=warn", "out=/nonexistent-zd-dir/state.json")
	if r.exitCode != 2 {
		t.Fatalf("describe unwritable out: exit = %d, want 2; stdout=%q stderr=%q", r.exitCode, r.stdout, r.stderr)
	}
	if !strings.Contains(r.stderr, "invocation") {
		t.Errorf("describe unwritable out: stderr %q does not carry domain=invocation", r.stderr)
	}
}

// readFileTrim reads a file and returns its trimmed content, failing on error.
func readFileTrim(t *testing.T, path string) string {
	t.Helper()
	b := readFile(t, path)
	return strings.TrimSpace(string(b))
}
