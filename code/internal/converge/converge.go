// generated from spec: zypper-declarative.spec.md sha256:714e75ff672557d2c7344736a5f36b52afa37e0f565b07de83e1b18cc4492014
//
// Package converge applies the intent diff inside a transaction context:
// converge-packages, converge-files, converge-units. All work is performed
// against the context root, never the running system root.
package converge

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mge1512/zypper-declarative/internal/diag"
	"github.com/mge1512/zypper-declarative/internal/diff"
	"github.com/mge1512/zypper-declarative/internal/manifest"
	"github.com/mge1512/zypper-declarative/internal/sysiface"
	"github.com/mge1512/zypper-declarative/internal/txn"
)

const syncpoint = "/etc/etc.syncpoint"

// Converger applies convergence against a context. ContentStore resolves
// content_ref values; KeepList and RPM-owned paths are protected from deletion.
type Converger struct {
	Runner       sysiface.CommandRunner
	ContentStore string
	RepoLock     string
	KeepList     map[string]bool
}

// Packages applies the package portion of the diff (converge-packages STEPS
// 1–4) and returns the resolved installed set as the lock.
func (c *Converger) Packages(ctx txn.Context, d diff.Diff) (*manifest.PackagesScope, *diag.Diagnostic) {
	root := ctx.Root
	// STEP 1 — configure repositories (or the CONFIG pin).
	repos := d.ReposSet
	if len(repos) == 0 && c.RepoLock != "" {
		if _, _, err := c.Runner.Run("zypper", "--root", root, "addrepo", c.RepoLock, "repo-lock"); err != nil {
			return nil, diag.New(diag.DomainRepositories, "repository configuration failed: %v", err)
		}
	}
	for _, r := range repos {
		if _, _, err := c.Runner.Run("zypper", "--root", root, "addrepo", "--check",
			"--priority", strconv.Itoa(r.Priority), r.URL, r.Alias); err != nil {
			return nil, diag.New(diag.DomainRepositories, "repository configuration failed for %s: %v", r.Alias, err)
		}
	}

	// STEP 2 — install.
	if len(d.PackagesInstall) > 0 {
		args := []string{"--root", root, "--non-interactive", "install", "--no-recommends"}
		for _, p := range d.PackagesInstall {
			args = append(args, p.Name)
		}
		if _, stderr, err := c.Runner.Run("zypper", args...); err != nil {
			return nil, diag.New(diag.DomainPackages, "package install failed: %v (%s)", err, strings.TrimSpace(stderr))
		}
	}

	// STEP 3 — remove.
	if len(d.PackagesRemove) > 0 {
		args := []string{"--root", root, "--non-interactive", "remove"}
		for _, p := range d.PackagesRemove {
			args = append(args, p.Name)
		}
		if _, stderr, err := c.Runner.Run("zypper", args...); err != nil {
			return nil, diag.New(diag.DomainPackages, "package remove failed: %v (%s)", err, strings.TrimSpace(stderr))
		}
	}

	// STEP 4 — query the rpmdb for the full installed set (the lock).
	scope := manifest.EmptyPackages()
	stdout, _, err := c.Runner.Run("rpm", "--root", root, "-qa", "--qf",
		"%{NAME} %{VERSION} %{RELEASE} %{ARCH}\\n")
	if err != nil {
		return nil, diag.New(diag.DomainPackages, "rpmdb query failed after convergence: %v", err)
	}
	sc := bufio.NewScanner(strings.NewReader(stdout))
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		if len(f) < 4 {
			continue
		}
		scope.Elements = append(scope.Elements, manifest.PackageRecord{
			Name: f[0], Version: f[1], Release: f[2], Arch: f[3],
		})
	}
	return scope, nil
}

// Files applies the file portion of the diff (converge-files STEPS 1–3).
func (c *Converger) Files(ctx txn.Context, d diff.Diff) *diag.Diagnostic {
	root := ctx.Root
	// STEP 1 — write declared files, verifying content hash.
	for _, e := range d.FilesWrite {
		content, derr := c.resolveContent(e.ContentRef)
		if derr != nil {
			return diag.New(diag.DomainFiles, "content resolution failed for %s: %v", e.Name, derr)
		}
		target := filepath.Join(root, e.Name)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return diag.New(diag.DomainFiles, "file write failed for %s: %v", e.Name, err)
		}
		mode := parseMode(e.Mode)
		if err := os.WriteFile(target, content, mode); err != nil {
			return diag.New(diag.DomainFiles, "file write failed for %s: %v", e.Name, err)
		}
		if e.SHA256 != "" {
			sum := sha256.Sum256(content)
			if hex.EncodeToString(sum[:]) != e.SHA256 {
				return diag.New(diag.DomainFiles, "written content hash mismatch for %s", e.Name)
			}
		}
	}

	// STEP 2 — delete dropped files, excluding RPM-owned, keep-listed, syncpoint.
	for _, p := range d.FilesDelete {
		if p == syncpoint || c.KeepList[p] {
			continue
		}
		if c.rpmOwned(root, p) {
			continue
		}
		target := filepath.Join(root, p)
		if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
			return diag.New(diag.DomainFiles, "delete failed for %s: %v", p, err)
		}
	}
	return nil
}

// Units applies the unit portion of the diff using offline enablement against
// the context root (converge-units STEPS 1–2).
func (c *Converger) Units(ctx txn.Context, d diff.Diff) *diag.Diagnostic {
	root := ctx.Root
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
			return diag.New(diag.DomainUnits, "unknown declared state %q for %s", u.State, u.Name)
		}
		if _, stderr, err := c.Runner.Run("systemctl", "--root", root, verb, u.Name); err != nil {
			return diag.New(diag.DomainUnits, "offline enablement failed for %s: %v (%s)", u.Name, err, strings.TrimSpace(stderr))
		}
	}
	return nil
}

func (c *Converger) resolveContent(ref string) ([]byte, error) {
	if ref == "" {
		return []byte{}, nil
	}
	base := c.ContentStore
	if base == "" {
		base = "."
	}
	return os.ReadFile(filepath.Join(base, ref))
}

func (c *Converger) rpmOwned(root, p string) bool {
	stdout, _, err := c.Runner.Run("rpm", "--root", root, "-qf", p)
	if err != nil {
		return false
	}
	return !strings.Contains(stdout, "not owned")
}

func parseMode(mode string) os.FileMode {
	if mode == "" {
		return 0o644
	}
	n, err := strconv.ParseUint(mode, 8, 32)
	if err != nil {
		return 0o644
	}
	return os.FileMode(n)
}
