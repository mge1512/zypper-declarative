// generated from spec: zypper-declarative.spec.md sha256:58e1636e2de82ab81a5cd3f81d6b3c9ac6a8976e18f9abb2bbd2b2aba56fe4d4
//
// Package record reads and writes the applied record of a generation. The
// applied record lives at <root>/usr/lib/zypper-declarative/applied.json so it
// travels with the snapshot and is restored automatically on rollback.
package record

import (
	"os"
	"path/filepath"
	"time"

	"github.com/mge1512/zypper-declarative/internal/diag"
	"github.com/mge1512/zypper-declarative/internal/manifest"
)

// RelPath is the path of the applied record relative to a generation root.
const RelPath = "usr/lib/zypper-declarative/applied.json"

// Load reads the applied record under root. Absence is a normal state
// (first-ever apply) and is reported as an empty record with present=false, not
// an error. A present-but-corrupt record yields a files error to the caller.
func Load(root string) (manifest.AppliedRecord, bool, *diag.Diagnostic) {
	path := filepath.Join(root, RelPath)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return manifest.Empty(), false, nil
		}
		return manifest.Empty(), false, diag.Errorf(diag.DomainFiles, "applied record unreadable: %v", err)
	}
	rec, perr := manifest.Parse(data, manifest.FormatJSON)
	if perr != nil {
		return manifest.Empty(), false, diag.Errorf(diag.DomainFiles, "applied record unparseable: %v", perr)
	}
	return rec, true, nil
}

// Stamper stamps the snapshot's snapper userdata with a key/value. The
// production binding talks to snapper; this interface keeps it abstract. The
// no-op stamper is used where snapper is unavailable (non-privileged paths).
type Stamper interface {
	Stamp(root, key, value string) error
}

// NoopStamper is a Stamper that does nothing and succeeds.
type NoopStamper struct{}

// Stamp is a no-op.
func (NoopStamper) Stamp(_, _, _ string) error { return nil }

// Write constructs and writes the applied record under root: it copies desired's
// repositories, services, and config_files; sets the packages scope to the
// resolved lock; sets meta.desired_sha256, created_at (now), and format_version
// (1). It is serialised as canonical JSON regardless of the desired manifest's
// input serialisation. Then it stamps the snapper userdata.
func Write(root string, desired manifest.Manifest, desiredSHA256 string, resolved *manifest.PackagesScope, stamper Stamper) *diag.Diagnostic {
	rec := manifest.Manifest{
		Meta: manifest.ManifestMeta{
			FormatVersion: 1,
			Generator:     "zypper-declarative " + version(),
			CreatedAt:     time.Now().UTC().Format(time.RFC3339),
			DesiredSHA256: desiredSHA256,
		},
		Repositories: desired.Repositories,
		Services:     desired.Services,
		ConfigFiles:  desired.ConfigFiles,
		Packages:     resolved,
	}

	data, err := manifest.MarshalCanonicalJSON(rec)
	if err != nil {
		return diag.Errorf(diag.DomainFiles, "applied record serialise failed: %v", err)
	}

	path := filepath.Join(root, RelPath)
	if mkErr := os.MkdirAll(filepath.Dir(path), 0755); mkErr != nil {
		return diag.Errorf(diag.DomainFiles, "applied record dir create failed: %v", mkErr)
	}
	if wErr := os.WriteFile(path, data, 0644); wErr != nil {
		return diag.Errorf(diag.DomainFiles, "applied record write failed: %v", wErr)
	}

	if stamper != nil {
		if sErr := stamper.Stamp(root, "manifest", desiredSHA256); sErr != nil {
			return diag.Errorf(diag.DomainFiles, "userdata stamp failed: %v", sErr)
		}
	}
	return nil
}

// version returns the generator version string; kept local to avoid an import
// cycle with the meta package consumers.
func version() string { return metaVersion }

// metaVersion mirrors internal/meta.Version; it is set via SetVersion to avoid a
// package import cycle if record were imported by meta (it is not, but this keeps
// the dependency direction one-way and explicit).
var metaVersion = "0.5.0"
