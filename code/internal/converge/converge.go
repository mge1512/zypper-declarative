// generated from spec: zypper-declarative.spec.md sha256:58e1636e2de82ab81a5cd3f81d6b3c9ac6a8976e18f9abb2bbd2b2aba56fe4d4
//
// Package converge applies the package, file, and unit portions of the intent
// diff inside the transaction context. converge-packages delegates to the
// package manager and reports the resolved scope (the lock); converge-files
// writes/deletes /etc files; converge-units applies offline enablement.
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
	"github.com/mge1512/zypper-declarative/internal/system"
	"github.com/mge1512/zypper-declarative/internal/txn"
)

const syncpoint = "/etc/etc.syncpoint"

// Converger applies convergence domains through a CommandRunner. ContentStore is
// the base path against which ManagedFileRecord.content_ref values are resolved.
type Converger struct {
	Runner       system.CommandRunner
	Reader       *state.Reader
	ContentStore string
	KeepList     map[string]bool
	RepoLock     string
}

// Packages applies the package portion of the intent diff inside ctx and returns
// the fully-resolved installed set (the lock) read from the rpmdb under ctx.Root.
func (c *Converger) Packages(ctx txn.Context, d diff.Diff) (*manifest.PackagesScope, *diag.Diagnostic) {
	// 1. ensure repositories are configured (or the CONFIG pin if repos_set empty).
	if len(d.ReposSet) > 0 {
		for _, r := range d.ReposSet {
			if _, _, err := c.Runner.Run("zypper", []string{"--root", ctx.Root, "ar", "-f", r.URL, r.Alias}); err != nil {
				return nil, diag.Errorf(diag.DomainRepositories, "repository configuration failed for %s: %v", r.Alias, err)
			}
		}
	} else if c.RepoLock != "" {
		if _, _, err := c.Runner.Run("zypper", []string{"--root", ctx.Root, "ar", "-f", c.RepoLock, "repo-lock"}); err != nil {
			return nil, diag.Errorf(diag.DomainRepositories, "repository configuration failed for repo-lock: %v", err)
		}
	}

	// 2. install.
	if len(d.PackagesInstall) > 0 {
		args := []string{"--root", ctx.Root, "--non-interactive", "in", "--no-recommends"}
		for _, p := range d.PackagesInstall {
			args = append(args, p.Name)
		}
		if _, stderr, err := c.Runner.Run("zypper", args); err != nil {
			return nil, diag.Errorf(diag.DomainPackages, "install failed: %v: %s", err, strings.TrimSpace(stderr))
		}
	}

	// 3. remove.
	if len(d.PackagesRemove) > 0 {
		args := []string{"--root", ctx.Root, "--non-interactive", "rm"}
		for _, p := range d.PackagesRemove {
			args = append(args, p.Name)
		}
		if _, stderr, err := c.Runner.Run("zypper", args); err != nil {
			return nil, diag.Errorf(diag.DomainPackages, "remove failed: %v: %s", err, strings.TrimSpace(stderr))
		}
	}

	// 4. query the rpmdb under ctx.Root for the full installed set (the lock).
	res, derr := c.Reader.Describe(ctx.Root, state.OnUnreadableError)
	if derr != nil {
		return nil, diag.Errorf(diag.DomainPackages, "post-install rpmdb query failed: %v", derr)
	}
	if res.Manifest.Packages == nil {
		// Never infer from file diffs; an empty installed set is an empty scope.
		return &manifest.PackagesScope{
			Attributes: manifest.ScopeAttributes{"package_system": "rpm"},
			Elements:   []manifest.PackageRecord{},
		}, nil
	}
	return res.Manifest.Packages, nil
}

// Files applies the file portion of the intent diff to <ctx.Root>/etc.
func (c *Converger) Files(ctx txn.Context, d diff.Diff) *diag.Diagnostic {
	// 1. write declared files.
	for _, e := range d.FilesWrite {
		content, rerr := c.resolveContent(e.ContentRef)
		if rerr != nil {
			return diag.Errorf(diag.DomainFiles, "content resolution failed for %s: %v", e.Name, rerr)
		}
		target := filepath.Join(ctx.Root, e.Name)
		if mkErr := os.MkdirAll(filepath.Dir(target), 0755); mkErr != nil {
			return diag.Errorf(diag.DomainFiles, "create dir for %s failed: %v", e.Name, mkErr)
		}
		mode := parseMode(e.Mode)
		if wErr := os.WriteFile(target, content, mode); wErr != nil {
			return diag.Errorf(diag.DomainFiles, "write %s failed: %v", e.Name, wErr)
		}
		// verify the written content hashes to e.sha256.
		sum := sha256.Sum256(content)
		if got := hex.EncodeToString(sum[:]); got != e.SHA256 {
			return diag.Errorf(diag.DomainFiles, "written content hash mismatch for %s: got %s want %s", e.Name, got, e.SHA256)
		}
	}

	// 2. delete files, excluding RPM-owned, keep-listed, and the syncpoint.
	for _, p := range d.FilesDelete {
		if p == syncpoint || (c.KeepList != nil && c.KeepList[p]) {
			continue
		}
		if c.rpmOwned(ctx.Root, p) {
			continue
		}
		target := filepath.Join(ctx.Root, p)
		if rmErr := os.Remove(target); rmErr != nil && !os.IsNotExist(rmErr) {
			return diag.Errorf(diag.DomainFiles, "delete %s failed: %v", p, rmErr)
		}
	}
	return nil
}

// Units applies the unit portion of the intent diff using offline enablement
// against ctx.Root.
func (c *Converger) Units(ctx txn.Context, d diff.Diff) *diag.Diagnostic {
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
			return diag.Errorf(diag.DomainUnits, "unknown unit state %q for %s", u.State, u.Name)
		}
		if _, stderr, err := c.Runner.Run("systemctl", []string{"--root", ctx.Root, verb, u.Name}); err != nil {
			return diag.Errorf(diag.DomainUnits, "offline %s of %s failed: %v: %s", verb, u.Name, err, strings.TrimSpace(stderr))
		}
	}
	return nil
}

func (c *Converger) resolveContent(ref string) ([]byte, error) {
	if ref == "" {
		return []byte{}, nil
	}
	path := ref
	if c.ContentStore != "" && !filepath.IsAbs(ref) {
		path = filepath.Join(c.ContentStore, ref)
	}
	return os.ReadFile(path)
}

func (c *Converger) rpmOwned(root, path string) bool {
	args := []string{}
	if root != "" && root != "/" {
		args = append(args, "--root", root)
	}
	args = append(args, "-qf", path)
	stdout, _, err := c.Runner.Run("rpm", args)
	if err != nil {
		return false
	}
	line := strings.TrimSpace(stdout)
	return line != "" && !strings.Contains(line, "not owned")
}

func parseMode(s string) os.FileMode {
	if s == "" {
		return 0644
	}
	v, err := strconv.ParseUint(s, 8, 32)
	if err != nil {
		return 0644
	}
	return os.FileMode(v)
}
