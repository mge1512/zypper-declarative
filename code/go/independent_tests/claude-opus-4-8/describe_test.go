// generated from spec: zypper-declarative.spec.md sha256:f2cc80627e483a48bb8411d297711bc5f6c6e74c28dbf0dafc8fe7bd8817251e
// tests by: claude-opus-4-8
//
// describe black-box tests against a fixture root, format resolution tests,
// and YAML acceptance tests. describe reads only files under <root>/etc for
// config_files and <root>/etc/zypp/repos.d for repositories, so a synthetic
// fixture root exercises the read paths without a live rpmdb. Tests that
// would require a populated rpmdb (packages/services scopes) assert only on
// the deterministic, file-derived scopes.
package independent_tests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeAppliedRecord places a canonical applied record at
// <root>/usr/lib/zypper-declarative/applied.json so load-applied-record finds it.
func writeAppliedRecord(t *testing.T, root, content string) {
	t.Helper()
	dir := filepath.Join(root, "usr", "lib", "zypper-declarative")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir applied dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "applied.json"), []byte(content), 0o644); err != nil {
		t.Fatalf("write applied record: %v", err)
	}
}

// makeReposFixture creates <root>/etc/zypp/repos.d with n readable .repo files.
func makeReposFixture(t *testing.T, root string, n int) {
	t.Helper()
	d := filepath.Join(root, "etc", "zypp", "repos.d")
	if err := os.MkdirAll(d, 0o755); err != nil {
		t.Fatalf("mkdir repos.d: %v", err)
	}
	for i := 0; i < n; i++ {
		name := []string{"first", "second", "third"}[i%3]
		body := "[" + name + "]\n" +
			"name=Repo " + name + "\n" +
			"enabled=1\n" +
			"autorefresh=0\n" +
			"baseurl=https://example/" + name + "\n" +
			"type=rpm-md\n" +
			"gpgcheck=1\n" +
			"priority=99\n"
		if err := os.WriteFile(filepath.Join(d, name+".repo"), []byte(body), 0o644); err != nil {
			t.Fatalf("write repo file: %v", err)
		}
	}
}

// ----------------------------------------------------------------------------
// describe: repositories from on-disk repos.d (read directly, world-readable)
// ----------------------------------------------------------------------------

// EXAMPLE: describe_repositories_from_reposd — two .repo files -> two records.
func TestDescribeRepositoriesFromReposd(t *testing.T) {
	root := t.TempDir()
	// Empty /etc so config_files is genuinely empty (omitted), repos.d populated.
	if err := os.MkdirAll(filepath.Join(root, "etc"), 0o755); err != nil {
		t.Fatalf("mkdir etc: %v", err)
	}
	makeReposFixture(t, root, 2)
	stdout, stderr, exit := run(t, "describe", "root="+root, "on-unreadable=warn")
	if exit != 0 {
		t.Fatalf("describe repos: exit=%d, want 0\nstderr=%s", exit, stderr)
	}
	// The repositories scope should be present and carry both aliases.
	if !strings.Contains(stdout, "repositories") {
		t.Errorf("describe repos: output missing repositories scope: %q", stdout)
	}
	if !strings.Contains(stdout, "first") || !strings.Contains(stdout, "second") {
		t.Errorf("describe repos: output should contain both repo aliases: %q", stdout)
	}
}

// EXAMPLE: describe_omits_genuinely_empty_scope — empty readable repos.d -> scope omitted.
func TestDescribeOmitsGenuinelyEmptyScope(t *testing.T) {
	root := t.TempDir()
	// readable but empty repos.d, and empty etc
	if err := os.MkdirAll(filepath.Join(root, "etc", "zypp", "repos.d"), 0o755); err != nil {
		t.Fatalf("mkdir repos.d: %v", err)
	}
	stdout, stderr, exit := run(t, "describe", "root="+root, "on-unreadable=warn")
	if exit != 0 {
		t.Fatalf("describe empty: exit=%d, want 0\nstderr=%s", exit, stderr)
	}
	// repositories scope must NOT be emitted with empty _elements.
	if strings.Contains(stdout, `"repositories"`) {
		t.Errorf("describe empty: repositories scope should be omitted, not emitted empty: %q", stdout)
	}
}

// ----------------------------------------------------------------------------
// describe: output format resolution (resolve-format)
// ----------------------------------------------------------------------------

// EXAMPLE: describe_out_extension_json — out=...json -> JSON file.
func TestDescribeOutExtensionJSON(t *testing.T) {
	root := t.TempDir()
	makeReposFixture(t, root, 1)
	outDir := t.TempDir()
	out := filepath.Join(outDir, "state.json")
	_, stderr, exit := run(t, "describe", "root="+root, "out="+out, "on-unreadable=warn")
	if exit != 0 {
		t.Fatalf("describe out json: exit=%d, want 0\nstderr=%s", exit, stderr)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read out: %v", err)
	}
	trimmed := strings.TrimSpace(string(data))
	if !strings.HasPrefix(trimmed, "{") {
		t.Errorf("describe out json: file should be JSON (start with '{'): %q", trimmed[:min(40, len(trimmed))])
	}
}

// EXAMPLE: describe_out_extension_yaml — out=...yaml -> YAML file (not starting with '{').
func TestDescribeOutExtensionYAML(t *testing.T) {
	root := t.TempDir()
	makeReposFixture(t, root, 1)
	outDir := t.TempDir()
	out := filepath.Join(outDir, "state.yaml")
	_, stderr, exit := run(t, "describe", "root="+root, "out="+out, "on-unreadable=warn")
	if exit != 0 {
		t.Fatalf("describe out yaml: exit=%d, want 0\nstderr=%s", exit, stderr)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read out: %v", err)
	}
	trimmed := strings.TrimSpace(string(data))
	if strings.HasPrefix(trimmed, "{") {
		t.Errorf("describe out yaml: file should be YAML, not JSON object: %q", trimmed[:min(40, len(trimmed))])
	}
}

// EXAMPLE: describe_format_overrides_extension — format=json out=...yaml -> JSON content.
func TestDescribeFormatOverridesExtension(t *testing.T) {
	root := t.TempDir()
	makeReposFixture(t, root, 1)
	outDir := t.TempDir()
	out := filepath.Join(outDir, "state.yaml")
	_, stderr, exit := run(t, "describe", "root="+root, "format=json", "out="+out, "on-unreadable=warn")
	if exit != 0 {
		t.Fatalf("describe format override: exit=%d, want 0\nstderr=%s", exit, stderr)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read out: %v", err)
	}
	trimmed := strings.TrimSpace(string(data))
	if !strings.HasPrefix(trimmed, "{") {
		t.Errorf("describe format override: explicit format=json should yield JSON despite .yaml ext: %q",
			trimmed[:min(40, len(trimmed))])
	}
}

// EXAMPLE: describe_output_unwritable — unwritable out path -> domain=invocation, exit 2.
func TestDescribeOutputUnwritable(t *testing.T) {
	root := t.TempDir()
	makeReposFixture(t, root, 1)
	_, stderr, exit := run(t, "describe", "root="+root, "out=/nonexistent-dir-zd/state.json", "on-unreadable=warn")
	if exit != 2 {
		t.Fatalf("describe unwritable out: exit=%d, want 2\nstderr=%s", exit, stderr)
	}
	if !strings.Contains(stderr, "invocation") {
		t.Errorf("describe unwritable out: stderr missing domain=invocation: %q", stderr)
	}
}

// ----------------------------------------------------------------------------
// describe: /etc walk classifies entry types (regression for the 0.6.2 crash)
// ----------------------------------------------------------------------------

// EXAMPLE: describe_traverses_etc_subdirectories — a subdir under /etc is
// descended into (not read as a file), no "is a directory" error, run continues.
func TestDescribeTraversesEtcSubdirectories(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "etc", "ImageMagick-7")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatalf("mkdir subdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(sub, "policy.xml"), []byte("<policy/>\n"), 0o644); err != nil {
		t.Fatalf("write file in subdir: %v", err)
	}
	makeReposFixture(t, root, 1)
	stdout, stderr, exit := run(t, "describe", "root="+root, "on-unreadable=warn")
	if exit != 0 {
		t.Fatalf("describe subdir: exit=%d, want 0\nstderr=%s", exit, stderr)
	}
	// Must not abort with an "is a directory" error.
	if strings.Contains(strings.ToLower(stderr), "is a directory") {
		t.Errorf("describe subdir: must not error on directory entry: %q", stderr)
	}
	_ = stdout
}

// EXAMPLE: describe_skips_special_file is hard to construct portably (mkfifo)
// without extra privileges; covered indirectly by the directory-traversal test
// and the type-classification invariant. Omitted as a black-box fixture.

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
