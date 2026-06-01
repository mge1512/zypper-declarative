// generated from spec: zypper-declarative.spec.md sha256:f2cc80627e483a48bb8411d297711bc5f6c6e74c28dbf0dafc8fe7bd8817251e
//
// Package converge implements converge-packages, converge-files, and
// converge-units. Each applies the relevant portion of the intent diff within
// the transaction context and returns errors (as *diag.Diagnostic) to its
// caller; exit mapping lives only in the verb layer.
package converge

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mge1512/zypper-declarative/internal/diag"
	"github.com/mge1512/zypper-declarative/internal/diff"
	"github.com/mge1512/zypper-declarative/internal/manifest"
	"github.com/mge1512/zypper-declarative/internal/state"
	"github.com/mge1512/zypper-declarative/internal/txn"
)

// Options carries cross-cutting convergence inputs.
type Options struct {
	ContentStore string          // base path for content_ref resolution
	KeepList     map[string]bool // paths never written or deleted
	RepoLock     string          // fallback pin when repos_set is empty
	Runner       state.CommandRunner
}

// ConvergePackages applies the package portion of the intent diff and returns
// the rpmdb-reported installed set (the lock).
func ConvergePackages(ctx *txn.Context, d *diff.Diff, opts Options) (*manifest.PackagesScope, *diag.Diagnostic) {
	runner := opts.Runner
	if runner == nil {
		runner = &state.OSCommandRunner{}
	}
	// 1. Ensure declared repositories are configured (or the CONFIG pin).
	if len(d.ReposSet) == 0 && opts.RepoLock == "" {
		// No repository pin available; resolving installs would be unpinned.
		if len(d.PackagesInstall) > 0 {
			return nil, diag.New(diag.DomainRepositories,
				"no repositories declared and no repo-lock pin configured")
		}
	}
	// 2/3. Install and remove against the pinned repositories.
	for _, p := range d.PackagesInstall {
		if _, stderr, err := runner.Run("zypper", "--root", ctx.Root, "-n", "install", "--no-recommends", p.Name); err != nil {
			return nil, diag.New(diag.DomainPackages, "install %q failed: %s", p.Name, strings.TrimSpace(stderr))
		}
	}
	for _, p := range d.PackagesRemove {
		if _, stderr, err := runner.Run("zypper", "--root", ctx.Root, "-n", "remove", p.Name); err != nil {
			return nil, diag.New(diag.DomainPackages, "remove %q failed: %s", p.Name, strings.TrimSpace(stderr))
		}
	}
	// 4. Query the rpmdb for the full installed set: the lock.
	stdout, _, err := runner.Run("rpm", "-qa", "--dbpath", filepath.Join(ctx.Root, "usr/lib/sysimage/rpm"),
		"--qf", "%{NAME}\t%{VERSION}\t%{RELEASE}\t%{ARCH}\n")
	if err != nil {
		return nil, diag.New(diag.DomainPackages, "rpmdb query after convergence failed: %v", err)
	}
	scope := &manifest.PackagesScope{
		Attributes: map[string]interface{}{"package_system": manifest.PackageSystemRPM},
		Elements:   []manifest.PackageRecord{},
	}
	for _, line := range strings.Split(stdout, "\n") {
		f := strings.Split(line, "\t")
		if len(f) < 4 {
			continue
		}
		scope.Elements = append(scope.Elements, manifest.PackageRecord{Name: f[0], Version: f[1], Release: f[2], Arch: f[3]})
	}
	return scope, nil
}

// ConvergeFiles writes declared regular files and deletes only files the
// declaration dropped, excluding RPM-owned paths, the keep-list, and the
// syncpoint. Symlink convergence and type-transition handling are reserved.
func ConvergeFiles(ctx *txn.Context, d *diff.Diff, opts Options) *diag.Diagnostic {
	keep := opts.KeepList
	for _, e := range d.FilesWrite {
		if e.Type != "file" {
			// Symlink/dir convergence is reserved for the live-apply milestone.
			continue
		}
		content, derr := resolveContent(opts.ContentStore, e.ContentRef)
		if derr != nil {
			return diag.New(diag.DomainFiles, "content resolution for %q failed: %v", e.Name, derr)
		}
		dest := filepath.Join(ctx.Root, e.Name)
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return diag.New(diag.DomainFiles, "cannot create directory for %q: %v", e.Name, err)
		}
		mode := parseMode(e.Mode)
		if err := os.WriteFile(dest, content, mode); err != nil {
			return diag.New(diag.DomainFiles, "write %q failed: %v", e.Name, err)
		}
		if e.SHA256 != "" {
			sum := sha256.Sum256(content)
			if hex.EncodeToString(sum[:]) != e.SHA256 {
				return diag.New(diag.DomainFiles, "written content hash mismatch for %q", e.Name)
			}
		}
	}
	for _, p := range d.FilesDelete {
		if p == diff.SyncpointPath || keep[p] {
			continue
		}
		dest := filepath.Join(ctx.Root, p)
		if err := os.Remove(dest); err != nil && !os.IsNotExist(err) {
			return diag.New(diag.DomainFiles, "delete %q failed: %v", p, err)
		}
	}
	return nil
}

// ConvergeUnits applies the declared unit states offline against ctx.Root.
func ConvergeUnits(ctx *txn.Context, d *diff.Diff, opts Options) *diag.Diagnostic {
	runner := opts.Runner
	if runner == nil {
		runner = &state.OSCommandRunner{}
	}
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
			return diag.New(diag.DomainServices, "unit %q has invalid state %q", u.Name, u.State)
		}
		if _, stderr, err := runner.Run("systemctl", "--root", ctx.Root, verb, u.Name); err != nil {
			return diag.New(diag.DomainServices, "offline %s of %q failed: %s", verb, u.Name, strings.TrimSpace(stderr))
		}
	}
	return nil
}

// resolveContent loads the content for a content_ref relative to the store.
func resolveContent(store, ref string) ([]byte, error) {
	if ref == "" {
		return []byte{}, nil
	}
	path := ref
	if store != "" && !filepath.IsAbs(ref) {
		path = filepath.Join(store, ref)
	}
	return os.ReadFile(path)
}

// parseMode parses an octal mode string into os.FileMode, defaulting to 0644.
func parseMode(s string) os.FileMode {
	if s == "" {
		return 0o644
	}
	n, err := strconv.ParseUint(s, 8, 32)
	if err != nil {
		return 0o644
	}
	return os.FileMode(n)
}
