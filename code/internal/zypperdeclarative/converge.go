// generated from spec: zypper-declarative.spec.md sha256:714e75ff672557d2c7344736a5f36b52afa37e0f565b07de83e1b18cc4492014
//
// BEHAVIOR/INTERNAL: acquire-transaction-context, converge-packages,
// converge-files, converge-units, write-applied-record. Plus the production
// providers for the snapshot, package-manager, and init-system bindings.
package zypperdeclarative

import (
	"os"
	"path/filepath"
	"time"
)

// AcquireTransactionContext resolves the transaction binding and yields a
// context the convergence domains operate within.
func AcquireTransactionContext(p *Providers, mode TransactionMode) (*TransactionContext, *Diagnostic) {
	resolved := mode
	if mode == ModeAuto {
		if inside, _ := p.Snapshot.DetectInTransaction(); inside {
			resolved = ModeExternal
		} else {
			resolved = ModeInternal
		}
	}

	switch resolved {
	case ModeExternal:
		inside, root := p.Snapshot.DetectInTransaction()
		if !inside || root == "" {
			return nil, newError(DomainTransaction,
				"external mode but not running inside a snapshot transaction")
		}
		return &TransactionContext{Mode: ModeExternal, Root: root, OpenedHere: false}, nil
	case ModeInternal:
		root, diag := p.Snapshot.OpenInternal()
		if diag != nil {
			return nil, diag
		}
		return &TransactionContext{Mode: ModeInternal, Root: root, OpenedHere: true}, nil
	default:
		return nil, newError(DomainTransaction, "unknown transaction mode: "+string(resolved))
	}
}

// ConvergePackages applies the package portion of the intent diff inside the
// context and returns the resolved scope (the lock).
func ConvergePackages(p *Providers, ctx *TransactionContext, diff *Diff, repoLock string) (*PackagesScope, *Diagnostic) {
	if d := p.Packages.ConfigureRepositories(ctx.Root, diff.ReposSet, repoLock); d != nil {
		return nil, d
	}
	if d := p.Packages.Install(ctx.Root, diff.PackagesInstall); d != nil {
		return nil, d
	}
	if d := p.Packages.Remove(ctx.Root, diff.PackagesRemove); d != nil {
		return nil, d
	}
	resolved, d := p.Packages.QueryInstalled(ctx.Root)
	if d != nil {
		return nil, d
	}
	return resolved, nil
}

// ConvergeFiles applies the file portion of the intent diff to <ctx.root>/etc.
func ConvergeFiles(p *Providers, ctx *TransactionContext, diff *Diff, cfg *Config, keep *KeepList) *Diagnostic {
	// Step 1: write declared files.
	for _, e := range diff.FilesWrite {
		content, d := resolveContent(cfg, e)
		if d != nil {
			return d
		}
		dest := filepath.Join(ctx.Root, e.Name)
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return newError(DomainFiles, "create parent dir failed: "+err.Error())
		}
		mode := parseMode(e.Mode)
		if err := os.WriteFile(dest, content, mode); err != nil {
			return newError(DomainFiles, "file write failed: "+err.Error())
		}
		sum, herr := fileSHA256(dest)
		if herr != nil {
			return newError(DomainFiles, "hash of written file failed: "+herr.Error())
		}
		if e.SHA256 != "" && sum != e.SHA256 {
			return newError(DomainFiles, "written content hash mismatch for "+e.Name)
		}
	}
	// Step 2: delete dropped files, excluding RPM-owned, keep-listed, syncpoint.
	for _, pth := range diff.FilesDelete {
		if keep.Has(pth) || pth == Syncpoint {
			continue
		}
		if rpmOwned(p, ctx.Root, pth) {
			continue
		}
		dest := filepath.Join(ctx.Root, pth)
		if err := os.Remove(dest); err != nil && !os.IsNotExist(err) {
			return newError(DomainFiles, "delete failed: "+err.Error())
		}
	}
	return nil
}

func rpmOwned(p *Providers, root, name string) bool {
	if probe, ok := p.Probe.(*OSSystemProbe); ok {
		return probe.ownerPackage(root, name) != ""
	}
	return false
}

// resolveContent resolves a ManagedFileRecord's content via its content_ref
// against the configured content store.
func resolveContent(cfg *Config, e ManagedFileRecord) ([]byte, *Diagnostic) {
	if e.ContentRef == "" {
		return nil, newError(DomainFiles, "no content_ref for "+e.Name)
	}
	base := cfg.ContentStore
	path := e.ContentRef
	if base != "" {
		path = filepath.Join(base, e.ContentRef)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, newError(DomainFiles, "content resolution failed for "+e.Name+": "+err.Error())
	}
	return data, nil
}

func parseMode(s string) os.FileMode {
	var v uint32
	for _, c := range s {
		if c < '0' || c > '7' {
			return 0o644
		}
		v = v*8 + uint32(c-'0')
	}
	if v == 0 {
		return 0o644
	}
	return os.FileMode(v)
}

// ConvergeUnits applies the unit portion of the intent diff offline against
// ctx.root.
func ConvergeUnits(p *Providers, ctx *TransactionContext, diff *Diff) *Diagnostic {
	for _, u := range diff.UnitsChange {
		if d := p.Init.SetEnablementOffline(ctx.Root, u); d != nil {
			return d
		}
	}
	return nil
}

// WriteAppliedRecord writes the applied record into the transaction context.
func WriteAppliedRecord(p *Providers, ctx *TransactionContext, desired *Manifest, desiredSHA string, resolved *PackagesScope) *Diagnostic {
	rec := &Manifest{
		Meta: ManifestMeta{
			FormatVersion: 1,
			Generator:     Generator,
			CreatedAt:     time.Now().UTC().Format(time.RFC3339),
			DesiredSHA256: desiredSHA,
		},
		Repositories: desired.Repositories,
		Services:     desired.Services,
		ConfigFiles:  desired.ConfigFiles,
		Packages:     resolved,
	}
	data, err := MarshalCanonicalJSON(rec)
	if err != nil {
		return newError(DomainFiles, "applied record marshal failed: "+err.Error())
	}
	dest := filepath.Join(ctx.Root, AppliedRecordRelPath)
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return newError(DomainFiles, "create applied record dir failed: "+err.Error())
	}
	if err := os.WriteFile(dest, data, 0o644); err != nil {
		return newError(DomainFiles, "applied record write failed: "+err.Error())
	}
	if d := p.Snapshot.StampUserdata(ctx.Root, "manifest", desiredSHA); d != nil {
		return d
	}
	return nil
}
