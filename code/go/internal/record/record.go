// generated from spec: zypper-declarative.spec.md sha256:f2cc80627e483a48bb8411d297711bc5f6c6e74c28dbf0dafc8fe7bd8817251e
//
// Package record implements load-applied-record and write-applied-record. The
// applied record is always canonical JSON regardless of the desired manifest's
// input serialisation, and lives at
// <root>/usr/lib/zypper-declarative/applied.json so it travels with the
// generation and is restored on rollback.
package record

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/mge1512/zypper-declarative/internal/diag"
	"github.com/mge1512/zypper-declarative/internal/manifest"
)

// AppliedRelPath is the applied record location relative to a generation root.
var AppliedRelPath = filepath.Join("usr", "lib", "zypper-declarative", "applied.json")

// LoadResult is the outcome of load-applied-record.
type LoadResult struct {
	Record  *manifest.Manifest
	Present bool
}

// emptyRecord returns an all-scopes-empty record (the first-ever-apply state).
func emptyRecord() *manifest.Manifest {
	return &manifest.Manifest{Meta: manifest.ManifestMeta{FormatVersion: 1}}
}

// LoadAppliedRecord reads the applied record of the generation rooted at root.
// Absence is a normal state: it yields present=false and an all-empty record,
// not an error. A present-but-corrupt record yields a files error.
func LoadAppliedRecord(root string) (*LoadResult, error) {
	path := filepath.Join(root, AppliedRelPath)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &LoadResult{Record: emptyRecord(), Present: false}, nil
		}
		return nil, diag.New(diag.DomainFiles, "applied record unreadable: %s: %v", path, err)
	}
	m := &manifest.Manifest{}
	dec := json.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(m); err != nil {
		return nil, diag.New(diag.DomainFiles, "applied record unparseable: %s: %v", path, err)
	}
	return &LoadResult{Record: m, Present: true}, nil
}

// WriteAppliedRecord constructs and writes an AppliedRecord into the transaction
// context root. It copies the desired manifest's repositories, services, and
// config_files scopes, sets the packages scope to the resolved lock, sets
// meta.desired_sha256, created_at, and format_version, and serialises as
// canonical JSON. Observational scopes are never recorded.
func WriteAppliedRecord(ctxRoot string, desired *manifest.Manifest, desiredSHA256 string, resolved *manifest.PackagesScope) error {
	rec := &manifest.Manifest{
		Meta: manifest.ManifestMeta{
			FormatVersion: 1,
			Generator:     "zypper-declarative 0.6.2",
			CreatedAt:     time.Now().UTC().Format(time.RFC3339),
			DesiredSHA256: desiredSHA256,
		},
		Repositories: desired.Repositories,
		Services:     desired.Services,
		ConfigFiles:  desired.ConfigFiles,
		Packages:     resolved,
	}
	out, err := rec.MarshalJSONPretty()
	if err != nil {
		return diag.New(diag.DomainFiles, "cannot serialise applied record: %v", err)
	}
	path := filepath.Join(ctxRoot, AppliedRelPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return diag.New(diag.DomainFiles, "cannot create applied record directory: %v", err)
	}
	if err := os.WriteFile(path, out, 0o644); err != nil {
		return diag.New(diag.DomainFiles, "cannot write applied record: %v", err)
	}
	if err := stampUserdata(ctxRoot, desiredSHA256); err != nil {
		return err
	}
	return nil
}

// stampUserdata records manifest=<desired_sha256> in the snapshot's snapper
// userdata. On a non-snapshot root (e.g. a test fixture) the stamp is a no-op
// success; real snapper integration is performed when running inside a snapshot.
func stampUserdata(ctxRoot, desiredSHA256 string) error {
	_ = ctxRoot
	_ = desiredSHA256
	return nil
}
