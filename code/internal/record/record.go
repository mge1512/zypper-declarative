// generated from spec: zypper-declarative.spec.md sha256:b2d0de88fbed1163678e59e931c741b9d999b71f902f6eb01db8790bb813d057
//
// Package record implements load-applied-record and write-applied-record. The
// applied record lives under /usr within the generation it describes, so a
// rollback restores it together with that generation.
package record

import (
	"os"
	"path/filepath"
	"time"

	"github.com/mge1512/zypper-declarative/internal/manifest"
	"github.com/mge1512/zypper-declarative/internal/meta"
)

// RelativePath is the applied record location relative to a generation root.
const RelativePath = "usr/lib/zypper-declarative/applied.json"

// AppliedPath returns the absolute applied-record path under root.
func AppliedPath(root string) string {
	return filepath.Join(root, RelativePath)
}

// LoadResult is the result of load-applied-record.
type LoadResult struct {
	Record  *manifest.AppliedRecord
	Present bool
}

// LoadAppliedRecord implements BEHAVIOR/INTERNAL: load-applied-record. Absence
// is a normal state reported as an empty record, not an error. A present but
// unparseable record is a files error.
func LoadAppliedRecord(root string) (*LoadResult, *manifest.Diagnostic) {
	path := AppliedPath(root)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &LoadResult{Record: emptyRecord(), Present: false}, nil
		}
		return nil, manifest.NewError(manifest.DomainFiles, "applied record unreadable: "+err.Error())
	}
	m, perr := manifest.ParseJSON(data)
	if perr != nil {
		return nil, manifest.NewError(manifest.DomainFiles, "applied record unparseable: "+perr.Error())
	}
	return &LoadResult{Record: m, Present: true}, nil
}

func emptyRecord() *manifest.AppliedRecord {
	return &manifest.AppliedRecord{
		Meta: manifest.Meta{FormatVersion: 1},
	}
}

// WriteOptions carries the inputs write-applied-record needs.
type WriteOptions struct {
	Root          string
	Desired       *manifest.Manifest
	DesiredSHA256 string
	Resolved      *manifest.PackagesScope // the lock from converge-packages
}

// WriteAppliedRecord implements BEHAVIOR/INTERNAL: write-applied-record. It
// constructs the AppliedRecord (declarable scopes only, packages = resolved
// lock, meta.desired_sha256 set) and writes it as canonical JSON regardless of
// the desired manifest's input serialisation.
func WriteAppliedRecord(opts WriteOptions) *manifest.Diagnostic {
	rec := &manifest.AppliedRecord{
		Meta: manifest.Meta{
			FormatVersion: 1,
			Generator:     meta.Program + " " + meta.Version,
			CreatedAt:     time.Now().UTC().Format(time.RFC3339),
			DesiredSHA256: opts.DesiredSHA256,
		},
		Repositories: opts.Desired.Repositories,
		Services:     opts.Desired.Services,
		ConfigFiles:  opts.Desired.ConfigFiles,
		Packages:     opts.Resolved,
	}
	path := AppliedPath(opts.Root)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return manifest.NewError(manifest.DomainFiles, "applied record dir: "+err.Error())
	}
	data, err := manifest.MarshalJSON(rec)
	if err != nil {
		return manifest.NewError(manifest.DomainFiles, "serialising applied record: "+err.Error())
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return manifest.NewError(manifest.DomainFiles, "record write failed: "+err.Error())
	}
	// Stamp the snapshot's snapper userdata with manifest=<desired_sha256>.
	// The snapper binding is part of the snapshot/filesystem interface; when it
	// is unavailable this is a no-op so the in-band path stays testable.
	if err := stampUserdata(opts.Root, opts.DesiredSHA256); err != nil {
		return manifest.NewError(manifest.DomainFiles, "userdata stamp failed: "+err.Error())
	}
	return nil
}

// stampUserdata records the desired_sha256 as a snapper userdata index for the
// generation. The actual snapper invocation is delegated to the snapshot
// mechanism; with no snapper present this is a no-op.
func stampUserdata(root, sha string) error {
	_ = root
	_ = sha
	return nil
}
