// generated from spec: zypper-declarative.spec.md sha256:27aee8e374eb3507189bad0b78339109d0116e6a13b55ae4df2ba9a18e769fc4
// tests by: claude-opus-4-8
//
// Black-box describe tests. describe reads actual state under a given root.
// To remain non-privileged and deterministic, these tests point describe at a
// synthetic root= directory that the test populates (etc/zypp/repos.d, etc.),
// so the repositories scope and the resolve-format / output-path behaviour can
// be asserted without touching the live system or requiring rpm/root. Cases
// that genuinely require the rpmdb (config_files emission against package
// baselines) are documented in TEST_REPORT.md as deferred to a privileged
// environment and are not asserted here.
package independent_tests

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// makeRoot builds a synthetic root with a readable, empty-but-present
// /etc/zypp/repos.d and an /etc directory, then returns the root path.
func makeRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, d := range []string{"etc/zypp/repos.d"} {
		if err := os.MkdirAll(filepath.Join(root, d), 0755); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}
	return root
}

// addRepo writes a .repo INI file under <root>/etc/zypp/repos.d.
func addRepo(t *testing.T, root, name, content string) {
	t.Helper()
	p := filepath.Join(root, "etc", "zypp", "repos.d", name)
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatalf("write repo %s: %v", name, err)
	}
}

const repoA = `[repo-a]
name=Repo A
baseurl=https://example/a
type=rpm-md
enabled=1
gpgcheck=1
autorefresh=0
priority=99
`

const repoB = `[repo-b]
name=Repo B
baseurl=https://example/b
type=rpm-md
enabled=1
gpgcheck=1
autorefresh=1
priority=50
`

// EXAMPLE: describe_repositories_from_reposd — two readable .repo files yield a
// repositories scope with two RepositoryRecord entries.
func TestDescribeRepositoriesFromReposd(t *testing.T) {
	root := makeRoot(t)
	addRepo(t, root, "a.repo", repoA)
	addRepo(t, root, "b.repo", repoB)
	stdout, stderr, exit := run(t, "describe", "root="+root, "on-unreadable=warn")
	if exit != 0 {
		t.Fatalf("describe exit = %d, want 0 (stderr=%q)", exit, stderr)
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
		t.Fatalf("describe output not valid JSON: %v\n%s", err, stdout)
	}
	repos, ok := doc["repositories"]
	if !ok {
		t.Fatalf("describe output missing repositories scope: %s", stdout)
	}
	var scope struct {
		Attributes map[string]interface{}   `json:"_attributes"`
		Elements   []map[string]interface{} `json:"_elements"`
	}
	if err := json.Unmarshal(repos, &scope); err != nil {
		t.Fatalf("repositories scope not a ScopeWrapper: %v", err)
	}
	if len(scope.Elements) != 2 {
		t.Errorf("repositories _elements = %d, want 2: %s", len(scope.Elements), stdout)
	}
	if scope.Attributes["repository_system"] != "zypp" {
		t.Errorf("repositories _attributes.repository_system = %v, want zypp", scope.Attributes["repository_system"])
	}
}

// EXAMPLE: describe_emits_manifest — meta.format_version = 1, and the document
// is a valid JSON Manifest. (We assert the structural envelope; package/config
// scope contents against the live system require privilege and are deferred.)
func TestDescribeEmitsManifestEnvelope(t *testing.T) {
	root := makeRoot(t)
	addRepo(t, root, "a.repo", repoA)
	stdout, stderr, exit := run(t, "describe", "root="+root, "on-unreadable=warn")
	if exit != 0 {
		t.Fatalf("describe exit = %d, want 0 (stderr=%q)", exit, stderr)
	}
	var doc struct {
		Meta struct {
			FormatVersion int    `json:"format_version"`
			Generator     string `json:"generator"`
		} `json:"meta"`
	}
	if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
		t.Fatalf("describe output not valid JSON: %v\n%s", err, stdout)
	}
	if doc.Meta.FormatVersion != 1 {
		t.Errorf("meta.format_version = %d, want 1", doc.Meta.FormatVersion)
	}
	if !strings.HasPrefix(doc.Meta.Generator, "zypper-declarative ") {
		t.Errorf("meta.generator = %q, want prefix 'zypper-declarative '", doc.Meta.Generator)
	}
}

// EXAMPLE: scope_attributes_always_object — every present scope's _attributes is
// a JSON object, never null. We check the repositories scope here.
func TestDescribeScopeAttributesAlwaysObject(t *testing.T) {
	root := makeRoot(t)
	addRepo(t, root, "a.repo", repoA)
	stdout, _, exit := run(t, "describe", "root="+root, "on-unreadable=warn")
	if exit != 0 {
		t.Fatalf("describe exit = %d, want 0", exit)
	}
	// Ensure no scope serialises _attributes as null.
	if strings.Contains(stdout, `"_attributes":null`) || strings.Contains(stdout, `"_attributes": null`) {
		t.Errorf("describe output contains _attributes: null (must be an object): %s", stdout)
	}
}

// EXAMPLE: describe_out_extension_json — out=...json writes a JSON document.
func TestDescribeOutExtensionJSON(t *testing.T) {
	root := makeRoot(t)
	addRepo(t, root, "a.repo", repoA)
	out := filepath.Join(t.TempDir(), "state.json")
	_, stderr, exit := run(t, "describe", "root="+root, "on-unreadable=warn", "out="+out)
	if exit != 0 {
		t.Fatalf("describe out=json exit = %d, want 0 (stderr=%q)", exit, stderr)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read out file: %v", err)
	}
	s := strings.TrimSpace(string(data))
	if !strings.HasPrefix(s, "{") {
		t.Errorf("out=.json did not produce a JSON document (starts %q)", firstLine(s))
	}
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		t.Errorf("out=.json content is not valid JSON: %v", err)
	}
}

// EXAMPLE: describe_out_extension_yaml — out=...yaml writes a YAML document
// (resolve-format selects yaml from the extension). The first line must not be
// a JSON object opener.
func TestDescribeOutExtensionYAML(t *testing.T) {
	root := makeRoot(t)
	addRepo(t, root, "a.repo", repoA)
	out := filepath.Join(t.TempDir(), "state.yaml")
	_, stderr, exit := run(t, "describe", "root="+root, "on-unreadable=warn", "out="+out)
	if exit != 0 {
		t.Fatalf("describe out=yaml exit = %d, want 0 (stderr=%q)", exit, stderr)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read out file: %v", err)
	}
	s := strings.TrimSpace(string(data))
	if strings.HasPrefix(s, "{") {
		t.Errorf("out=.yaml produced a JSON object (first line %q); expected YAML", firstLine(s))
	}
}

// EXAMPLE: describe_format_overrides_extension — format=json out=...yaml writes
// a JSON document (explicit format wins over extension).
func TestDescribeFormatOverridesExtension(t *testing.T) {
	root := makeRoot(t)
	addRepo(t, root, "a.repo", repoA)
	out := filepath.Join(t.TempDir(), "state.yaml")
	_, stderr, exit := run(t, "describe", "root="+root, "on-unreadable=warn", "format=json", "out="+out)
	if exit != 0 {
		t.Fatalf("describe format=json out=yaml exit = %d, want 0 (stderr=%q)", exit, stderr)
	}
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read out file: %v", err)
	}
	s := strings.TrimSpace(string(data))
	if !strings.HasPrefix(s, "{") {
		t.Errorf("format=json must produce JSON even with .yaml extension; got %q", firstLine(s))
	}
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		t.Errorf("format=json content is not valid JSON: %v", err)
	}
}

// EXAMPLE: describe_format_yaml — describe format=yaml to stdout yields a YAML
// document (not a JSON object).
func TestDescribeFormatYAMLStdout(t *testing.T) {
	root := makeRoot(t)
	addRepo(t, root, "a.repo", repoA)
	stdout, stderr, exit := run(t, "describe", "root="+root, "on-unreadable=warn", "format=yaml")
	if exit != 0 {
		t.Fatalf("describe format=yaml exit = %d, want 0 (stderr=%q)", exit, stderr)
	}
	s := strings.TrimSpace(stdout)
	if strings.HasPrefix(s, "{") {
		t.Errorf("describe format=yaml produced a JSON object; expected YAML: %q", firstLine(s))
	}
}

// EXAMPLE: describe_output_unwritable — out points into a non-existent /readonly
// path that cannot be created -> exit 2, domain=invocation.
func TestDescribeOutputUnwritable(t *testing.T) {
	root := makeRoot(t)
	addRepo(t, root, "a.repo", repoA)
	// A path under /proc cannot be written to as a regular file.
	_, stderr, exit := run(t, "describe", "root="+root, "on-unreadable=warn", "out=/proc/zd-cannot-write/state.json")
	if exit != 2 {
		t.Fatalf("describe unwritable out exit = %d, want 2 (stderr=%q)", exit, stderr)
	}
	if !strings.Contains(stderr, "invocation") {
		t.Errorf("describe unwritable out: stderr missing domain=invocation: %q", stderr)
	}
}

// EXAMPLE: describe_omits_genuinely_empty_scope — repos.d readable but empty:
// the repositories scope is omitted (not emitted with empty _elements).
func TestDescribeOmitsGenuinelyEmptyScope(t *testing.T) {
	root := makeRoot(t) // repos.d exists but is empty
	stdout, stderr, exit := run(t, "describe", "root="+root, "on-unreadable=warn")
	if exit != 0 {
		t.Fatalf("describe empty repos.d exit = %d, want 0 (stderr=%q)", exit, stderr)
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
		t.Fatalf("describe output not valid JSON: %v\n%s", err, stdout)
	}
	if _, present := doc["repositories"]; present {
		t.Errorf("repositories scope present though genuinely empty; should be omitted: %s", stdout)
	}
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}
