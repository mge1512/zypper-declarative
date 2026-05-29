// generated from spec: zypper-declarative.spec.md sha256:714e75ff672557d2c7344736a5f36b52afa37e0f565b07de83e1b18cc4492014
//
// Package record reads and writes the applied record of a generation. Absence
// is a normal state (first-ever apply), reported as an empty record, never an
// error. The applied record always serialises as canonical JSON.
package record

import (
	"os"
	"path/filepath"
	"time"

	"github.com/mge1512/zypper-declarative/internal/diag"
	"github.com/mge1512/zypper-declarative/internal/manifest"
	"github.com/mge1512/zypper-declarative/internal/meta"
)

// RelPath is the applied record location relative to a generation root.
const RelPath = "usr/lib/zypper-declarative/applied.json"

// Load reads the applied record under root (load-applied-record STEPS 1–4).
// present is false with an all-empty record when none exists. A present but
// unparseable record returns a files error.
func Load(root string) (rec manifest.AppliedRecord, present bool, d *diag.Diagnostic) {
	path := filepath.Join(rootOrSlash(root), RelPath)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return emptyRecord(), false, nil
		}
		return manifest.AppliedRecord{}, false, diag.New(diag.DomainFiles, "applied record unreadable: %v", err)
	}
	m, _, perr := manifest.Parse(data, manifest.FormatJSON)
	if perr != nil {
		return manifest.AppliedRecord{}, false, diag.New(diag.DomainFiles, "applied record corrupt: %v", perr)
	}
	return m, true, nil
}

// Write constructs and writes the applied record into the generation root
// (write-applied-record STEPS 1–3). The packages scope is set to resolved; the
// record is serialised as canonical JSON regardless of input format. The
// snapper userdata stamp is delegated by the caller (converge layer).
func Write(root string, desired *manifest.Manifest, desiredSHA256 string, resolved *manifest.PackagesScope) *diag.Diagnostic {
	out := manifest.AppliedRecord{
		Meta: manifest.ManifestMeta{
			FormatVersion: 1,
			Generator:     meta.Generator,
			CreatedAt:     time.Now().UTC().Format(time.RFC3339),
			DesiredSHA256: desiredSHA256,
		},
		Packages:     resolved,
		Repositories: desired.Repositories,
		Services:     desired.Services,
		ConfigFiles:  desired.ConfigFiles,
	}
	if out.Packages == nil {
		out.Packages = manifest.EmptyPackages()
	}

	path := filepath.Join(rootOrSlash(root), RelPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return diag.New(diag.DomainFiles, "applied record directory: %v", err)
	}
	data, err := manifest.MarshalJSON(&out)
	if err != nil {
		return diag.New(diag.DomainFiles, "applied record encode: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return diag.New(diag.DomainFiles, "applied record write: %v", err)
	}
	return nil
}

func emptyRecord() manifest.AppliedRecord {
	return manifest.AppliedRecord{
		Meta:         manifest.ManifestMeta{FormatVersion: 1, Generator: meta.Generator},
		Packages:     manifest.EmptyPackages(),
		Repositories: manifest.EmptyRepositories(),
		Services:     manifest.EmptyServices(),
		ConfigFiles:  manifest.EmptyConfigFiles(),
	}
}

func rootOrSlash(root string) string {
	if root == "" {
		return "/"
	}
	return root
}
