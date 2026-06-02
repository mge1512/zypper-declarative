// generated from spec: zypper-declarative.spec.md sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
//
// Package record reads and writes the applied record of a generation.
package record

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/mge1512/zypper-declarative/internal/manifest"
	"github.com/mge1512/zypper-declarative/internal/meta"
)

// Diagnostic mirrors the spec Diagnostic type for returning domain-tagged errors.
type Diagnostic struct {
	Severity string
	Domain   string
	Message  string
}

func (d *Diagnostic) Error() string { return d.Message }

// AppliedPath is the on-disk location of the applied record under a root.
func AppliedPath(root string) string {
	return filepath.Join(root, "usr", "lib", "zypper-declarative", "applied.json")
}

// LoadResult is the output of LoadApplied.
type LoadResult struct {
	Record  *manifest.Manifest
	Present bool
}

// emptyRecord returns an AppliedRecord with all scopes empty (first-ever apply).
func emptyRecord() *manifest.Manifest {
	return &manifest.Manifest{Meta: manifest.ManifestMeta{FormatVersion: 1, Generator: meta.Generator()}}
}

// LoadApplied implements BEHAVIOR/INTERNAL: load-applied-record.
//
//	Absence yields present=false and an all-empty record (never an error).
//	A present-but-corrupt record yields a files error to the caller.
func LoadApplied(root string) (*LoadResult, error) {
	p := AppliedPath(root)
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return &LoadResult{Record: emptyRecord(), Present: false}, nil
		}
		return nil, &Diagnostic{Severity: "Error", Domain: "files", Message: "applied record unreadable: " + err.Error()}
	}
	var m manifest.Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, &Diagnostic{Severity: "Error", Domain: "files", Message: "applied record unparseable: " + err.Error()}
	}
	return &LoadResult{Record: &m, Present: true}, nil
}

// WriteApplied implements BEHAVIOR/INTERNAL: write-applied-record. It constructs
// an AppliedRecord from the desired manifest plus the resolved packages lock,
// serialises it as canonical JSON, and writes it under the context root.
func WriteApplied(root string, desired *manifest.Manifest, desiredSHA256 string, resolved *manifest.ScopeWrapper[manifest.PackageRecord]) error {
	rec := &manifest.Manifest{
		Meta: manifest.ManifestMeta{
			FormatVersion: 1,
			Generator:     meta.Generator(),
			CreatedAt:     time.Now().UTC().Format(time.RFC3339),
			DesiredSHA256: desiredSHA256,
		},
		Repositories: desired.Repositories,
		Services:     desired.Services,
		ConfigFiles:  desired.ConfigFiles,
		Packages:     resolved,
	}
	// Observational scopes are never recorded.
	rec.ChangedManagedFiles = nil
	rec.UnmanagedFiles = nil

	out, err := rec.MarshalJSONIndent()
	if err != nil {
		return &Diagnostic{Severity: "Error", Domain: "files", Message: "could not serialise applied record: " + err.Error()}
	}
	p := AppliedPath(root)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return &Diagnostic{Severity: "Error", Domain: "files", Message: "could not create applied record directory: " + err.Error()}
	}
	if err := os.WriteFile(p, append(out, '\n'), 0o644); err != nil {
		return &Diagnostic{Severity: "Error", Domain: "files", Message: "applied record write failed: " + err.Error()}
	}
	// Stamp snapper userdata. Not available in this build; left as a documented
	// no-op so the write path completes on a non-snapper root.
	return nil
}
