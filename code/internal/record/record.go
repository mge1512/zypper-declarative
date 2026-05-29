// generated from spec: zypper-declarative.spec.md sha256:f8ff76ecbc4bbc69a49e2e32b2924da3a64df1ad46196e05ce8c137b684429b2
//
// Package record implements load-applied-record and write-applied-record. The
// applied record travels with the generation under
// <root>/usr/lib/zypper-declarative/applied.json and is always canonical JSON.
package record

import (
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/mge1512/zypper-declarative/internal/diag"
	"github.com/mge1512/zypper-declarative/internal/manifest"
	"github.com/mge1512/zypper-declarative/internal/meta"
)

// RelPath is the applied-record path relative to a generation root.
const RelPath = "usr/lib/zypper-declarative/applied.json"

// Path returns the applied-record path under the given root.
func Path(root string) string {
	return filepath.Join(root, RelPath)
}

// Load implements BEHAVIOR/INTERNAL: load-applied-record. Absence is a normal
// state (first-ever apply) reported as an empty record with present=false, not
// an error. A present-but-corrupt record yields a files error to the caller.
func Load(root string) (manifest.AppliedRecord, bool, *diag.Diagnostic) {
	p := Path(root)
	data, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return manifest.Empty(), false, nil
		}
		return manifest.Empty(), false, diag.Errorf(diag.DomainFiles, "applied record unreadable: %s: %v", p, err)
	}
	rec, derr := manifest.Decode(data, manifest.FormatJSON)
	if derr != nil {
		return manifest.Empty(), false, diag.Errorf(diag.DomainFiles, "applied record unparseable: %s: %v", p, derr)
	}
	return *rec, true, nil
}

// Write implements BEHAVIOR/INTERNAL: write-applied-record. It constructs the
// AppliedRecord (desired's repositories, services, config_files; resolved
// packages scope; meta.desired_sha256 set; meta.created_at now;
// meta.format_version 1), serialises it as canonical JSON, and writes it under
// the context root. The snapper userdata stamp is delegated to the caller via
// the returned record; the file write is the in-tree ledger.
func Write(root string, desired *manifest.Manifest, desiredSHA256 string, resolved *manifest.PackagesScope) (manifest.AppliedRecord, *diag.Diagnostic) {
	rec := manifest.AppliedRecord{
		Meta: manifest.Meta{
			FormatVersion: 1,
			Generator:     meta.Generator(),
			CreatedAt:     time.Now().UTC().Format(time.RFC3339),
			DesiredSHA256: desiredSHA256,
		},
		Packages:     resolved,
		Repositories: desired.Repositories,
		Services:     desired.Services,
		ConfigFiles:  desired.ConfigFiles,
	}

	data, err := rec.MarshalCanonicalJSON()
	if err != nil {
		return rec, diag.Errorf(diag.DomainFiles, "could not serialise applied record: %v", err)
	}
	p := Path(root)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return rec, diag.Errorf(diag.DomainFiles, "could not create applied-record directory: %v", err)
	}
	if err := os.WriteFile(p, data, 0o644); err != nil {
		return rec, diag.Errorf(diag.DomainFiles, "could not write applied record: %v", err)
	}
	return rec, nil
}
