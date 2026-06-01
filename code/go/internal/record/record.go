// generated from spec: zypper-declarative.spec.md sha256:27aee8e374eb3507189bad0b78339109d0116e6a13b55ae4df2ba9a18e769fc4
//
// Package record implements load-applied-record and write-applied-record. The
// applied record of a generation lives at
// <root>/usr/lib/zypper-declarative/applied.json and is always canonical JSON.
package record

import (
	"os"
	"path/filepath"

	"github.com/mge1512/zypper-declarative/internal/manifest"
	"github.com/mge1512/zypper-declarative/internal/meta"
)

// AppliedRelPath is the applied record path relative to a generation root.
const AppliedRelPath = "usr/lib/zypper-declarative/applied.json"

// Load implements BEHAVIOR/INTERNAL: load-applied-record. Absence is normal and
// yields an all-empty record with present=false; a present-but-corrupt record is
// an error.
func Load(root string) (rec *manifest.Manifest, present bool, err *manifest.Diagnostic) {
	path := filepath.Join(root, AppliedRelPath)
	data, rerr := os.ReadFile(path)
	if rerr != nil {
		if os.IsNotExist(rerr) {
			return &manifest.Manifest{Meta: manifest.ManifestMeta{FormatVersion: 1}}, false, nil
		}
		d := manifest.NewError(manifest.DomainFiles, "applied record unreadable: "+rerr.Error())
		return nil, false, &d
	}
	m, perr := manifest.Parse(data, manifest.ParseOptions{Format: manifest.FormatJSON, AllowObservational: true})
	if perr != nil {
		d := manifest.NewError(manifest.DomainFiles, "applied record unparseable: "+perr.Error())
		return nil, false, &d
	}
	return m, true, nil
}

// Write implements BEHAVIOR/INTERNAL: write-applied-record. It constructs an
// AppliedRecord from the desired manifest with the packages scope set to the
// resolved lock and meta.desired_sha256 set, then writes it as canonical JSON.
func Write(root string, desired *manifest.Manifest, desiredSHA256 string, resolved *manifest.PackagesScope, createdAt string) *manifest.Diagnostic {
	rec := &manifest.Manifest{
		Meta: manifest.ManifestMeta{
			FormatVersion: 1,
			Generator:     meta.Generator(),
			CreatedAt:     createdAt,
			DesiredSHA256: desiredSHA256,
		},
		Repositories: desired.Repositories,
		Services:     desired.Services,
		ConfigFiles:  desired.ConfigFiles,
		Packages:     resolved,
	}
	data, merr := manifest.MarshalJSON(rec)
	if merr != nil {
		d := manifest.NewError(manifest.DomainFiles, "cannot serialise applied record: "+merr.Error())
		return &d
	}
	dir := filepath.Join(root, filepath.Dir(AppliedRelPath))
	if mkerr := os.MkdirAll(dir, 0755); mkerr != nil {
		d := manifest.NewError(manifest.DomainFiles, "cannot create applied record directory: "+mkerr.Error())
		return &d
	}
	if werr := os.WriteFile(filepath.Join(root, AppliedRelPath), data, 0644); werr != nil {
		d := manifest.NewError(manifest.DomainFiles, "cannot write applied record: "+werr.Error())
		return &d
	}
	return nil
}
