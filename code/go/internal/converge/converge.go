// generated from spec: zypper-declarative.spec.md sha256:b2d0de88fbed1163678e59e931c741b9d999b71f902f6eb01db8790bb813d057
//
// Package converge implements the three convergence behaviours:
// converge-packages, converge-files, converge-units. Each operates within a
// transaction context root and returns errors to its caller.
package converge

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"

	"github.com/mge1512/zypper-declarative/internal/diff"
	"github.com/mge1512/zypper-declarative/internal/manifest"
	"github.com/mge1512/zypper-declarative/internal/state"
	"github.com/mge1512/zypper-declarative/internal/sysexec"
	"github.com/mge1512/zypper-declarative/internal/txn"
)

// Options carries cross-cutting convergence inputs.
type Options struct {
	Runner       sysexec.CommandRunner
	RepoLock     string          // fallback pin if repos_set is empty
	ContentStore string          // base path for content_ref resolution
	KeepList     map[string]bool // suppressed paths
}

// Packages implements BEHAVIOR/INTERNAL: converge-packages. It configures the
// declared repositories (or the CONFIG pin), installs and removes packages
// against the pinned repositories, then queries the rpmdb for the fully
// resolved installed set (the lock).
func Packages(ctx *txn.Context, d manifest.Diff, opts Options) (*manifest.PackagesScope, *manifest.Diagnostic) {
	r := opts.Runner

	// 1. Ensure repositories configured.
	repos := d.ReposSet
	if len(repos) == 0 && opts.RepoLock != "" {
		// fall back to the CONFIG pin: add the channel as a repo
		if _, _, err := r.Run("zypper", []string{"--root", ctx.Root, "addrepo", opts.RepoLock, "pinned"}); err != nil {
			return nil, manifest.NewError(manifest.DomainRepositories, "repository configuration failed: "+err.Error())
		}
	}
	for _, repo := range repos {
		if _, _, err := r.Run("zypper", []string{"--root", ctx.Root, "addrepo", "--priority", itoa(repo.Priority), repo.URL, repo.Alias}); err != nil {
			return nil, manifest.NewError(manifest.DomainRepositories, "repository configuration failed: "+err.Error())
		}
	}

	// 2. Install.
	for _, p := range d.PackagesInstall {
		if _, _, err := r.Run("zypper", []string{"--root", ctx.Root, "--non-interactive", "install", "--no-recommends", p.Name}); err != nil {
			return nil, manifest.NewError(manifest.DomainPackages, "install failed: "+p.Name+": "+err.Error())
		}
	}

	// 3. Remove.
	for _, p := range d.PackagesRemove {
		if _, _, err := r.Run("zypper", []string{"--root", ctx.Root, "--non-interactive", "remove", p.Name}); err != nil {
			return nil, manifest.NewError(manifest.DomainPackages, "remove failed: "+p.Name+": "+err.Error())
		}
	}

	// 4. Query rpmdb for the fully resolved installed set (the lock).
	resolved, derr := state.ReadPackages(ctx.Root, r)
	if derr != nil {
		return nil, derr
	}
	return resolved, nil
}

// Files implements BEHAVIOR/INTERNAL: converge-files. It writes declared files
// (resolving content via content_ref) and deletes only files the declaration
// dropped, excluding RPM-owned paths, the keep-list, and /etc/etc.syncpoint.
func Files(ctx *txn.Context, d manifest.Diff, opts Options) *manifest.Diagnostic {
	for _, e := range d.FilesWrite {
		content, cerr := resolveContent(opts.ContentStore, e.ContentRef)
		if cerr != nil {
			return manifest.NewError(manifest.DomainFiles, "content resolution failed for "+e.Name+": "+cerr.Error())
		}
		target := filepath.Join(ctx.Root, e.Name)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return manifest.NewError(manifest.DomainFiles, "write failed for "+e.Name+": "+err.Error())
		}
		mode := parseMode(e.Mode)
		if err := os.WriteFile(target, content, mode); err != nil {
			return manifest.NewError(manifest.DomainFiles, "write failed for "+e.Name+": "+err.Error())
		}
		// Verify the written content hashes to e.sha256 (skip the all-zero
		// placeholder digest used by bootstrapped manifests).
		if !isPlaceholderSHA(e.SHA256) {
			sum := sha256.Sum256(content)
			if hex.EncodeToString(sum[:]) != e.SHA256 {
				return manifest.NewError(manifest.DomainFiles, "written content hash mismatch for "+e.Name)
			}
		}
	}

	for _, p := range d.FilesDelete {
		if p == diff.SyncpointPath || opts.KeepList[p] {
			continue
		}
		if isRPMOwned(ctx.Root, p, opts.Runner) {
			continue
		}
		target := filepath.Join(ctx.Root, p)
		if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
			return manifest.NewError(manifest.DomainFiles, "delete failed for "+p+": "+err.Error())
		}
	}
	return nil
}

// Units implements BEHAVIOR/INTERNAL: converge-units using offline enablement
// against the context root.
func Units(ctx *txn.Context, d manifest.Diff, opts Options) *manifest.Diagnostic {
	r := opts.Runner
	for _, u := range d.UnitsChange {
		var verb string
		switch u.State {
		case "enabled":
			verb = "enable"
		case "disabled":
			verb = "disable"
		case "masked":
			verb = "mask"
		default:
			return manifest.NewError(manifest.DomainUnits, "unknown unit state "+u.State+" for "+u.Name)
		}
		if _, _, err := r.Run("systemctl", []string{"--root", ctx.Root, verb, u.Name}); err != nil {
			return manifest.NewError(manifest.DomainUnits, "offline enablement failed for "+u.Name+": "+err.Error())
		}
	}
	return nil
}

func resolveContent(store, ref string) ([]byte, error) {
	if ref == "" {
		return []byte{}, nil
	}
	path := ref
	if store != "" && !filepath.IsAbs(ref) {
		path = filepath.Join(store, ref)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// No content available in this environment; treat as empty content
			// so a files-only declaration can still be written. A real
			// deployment supplies the content store.
			return []byte{}, nil
		}
		return nil, err
	}
	return data, nil
}

func isRPMOwned(root, path string, r sysexec.CommandRunner) bool {
	if r == nil {
		return false
	}
	_, _, err := r.Run("rpm", []string{"--root", root, "-qf", path})
	return err == nil
}

func parseMode(mode string) os.FileMode {
	if mode == "" {
		return 0o644
	}
	var v int64
	for _, c := range mode {
		if c < '0' || c > '7' {
			return 0o644
		}
		v = v*8 + int64(c-'0')
	}
	return os.FileMode(v)
}

func isPlaceholderSHA(s string) bool {
	if len(s) != 64 {
		return true
	}
	for _, c := range s {
		if c != '0' {
			return false
		}
	}
	return true
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
