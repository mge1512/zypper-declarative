// generated from spec: zypper-declarative.spec.md sha256:27aee8e374eb3507189bad0b78339109d0116e6a13b55ae4df2ba9a18e769fc4
//
// Package converge applies the intent diff inside a transaction context:
// converge-packages (delegating to the package manager and reporting the
// resolved lock), converge-files (writing/deleting regular /etc files), and
// converge-units (offline enablement against the context root). External tools
// are driven through a CommandRunner so the binding stays a single seam.
package converge

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mge1512/zypper-declarative/internal/manifest"
	"github.com/mge1512/zypper-declarative/internal/state"
	"github.com/mge1512/zypper-declarative/internal/txn"
)

// Converger applies convergence by driving external tools through Runner.
type Converger struct {
	Runner state.CommandRunner
	// ContentStore is the base path against which ManagedFileRecord content_ref
	// values are resolved at apply time (CONFIG: content-store).
	ContentStore string
	// KeepList paths converge-files must never delete.
	KeepList map[string]bool
}

const syncpoint = "/etc/etc.syncpoint"

// Packages implements BEHAVIOR/INTERNAL: converge-packages. It configures the
// declared repositories (or the CONFIG pin), installs and removes, then queries
// the rpmdb under ctx.root for the fully-resolved installed set (the lock).
func (c *Converger) Packages(ctx *txn.Context, diff manifest.Diff) (*manifest.PackagesScope, *manifest.Diagnostic) {
	// 1. ensure repositories
	for _, repo := range diff.ReposSet {
		if _, stderr, err := c.run("zypper", c.zypperRootArgs(ctx.Root, "--non-interactive", "ar", "-f", repo.URL, repo.Alias)...); err != nil {
			// "already exists" is not a failure; otherwise report a repositories error.
			if !strings.Contains(strings.ToLower(stderr), "already") {
				d := manifest.NewError(manifest.DomainRepositories, "repository configuration failed: "+repo.Alias+": "+strings.TrimSpace(stderr))
				return nil, &d
			}
		}
	}
	// 2. install
	if len(diff.PackagesInstall) > 0 {
		args := c.zypperRootArgs(ctx.Root, "--non-interactive", "install", "--no-recommends")
		for _, p := range diff.PackagesInstall {
			args = append(args, p.Name)
		}
		if _, stderr, err := c.run("zypper", args...); err != nil {
			d := manifest.NewError(manifest.DomainPackages, "package install failed: "+strings.TrimSpace(stderr))
			return nil, &d
		}
	}
	// 3. remove
	if len(diff.PackagesRemove) > 0 {
		args := c.zypperRootArgs(ctx.Root, "--non-interactive", "remove")
		for _, p := range diff.PackagesRemove {
			args = append(args, p.Name)
		}
		if _, stderr, err := c.run("zypper", args...); err != nil {
			d := manifest.NewError(manifest.DomainPackages, "package remove failed: "+strings.TrimSpace(stderr))
			return nil, &d
		}
	}
	// 4. query the resolved installed set (the lock) from the rpmdb under ctx.root.
	resolved, d := c.queryInstalled(ctx.Root)
	if d != nil {
		return nil, d
	}
	return resolved, nil
}

// queryInstalled reads the installed set via rpm and returns it as a fully
// populated PackagesScope.
func (c *Converger) queryInstalled(root string) (*manifest.PackagesScope, *manifest.Diagnostic) {
	args := rpmRootArgs(root, "-qa", "--queryformat", "%{NAME} %{VERSION} %{RELEASE} %{ARCH}\n")
	stdout, stderr, err := c.run("rpm", args...)
	if err != nil && strings.TrimSpace(stdout) == "" {
		d := manifest.NewError(manifest.DomainPackages, "cannot read resolved package set: "+strings.TrimSpace(stderr))
		return nil, &d
	}
	scope := &manifest.PackagesScope{
		Attributes: map[string]interface{}{"package_system": "rpm"},
		Elements:   []manifest.PackageRecord{},
	}
	for _, line := range strings.Split(stdout, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		scope.Elements = append(scope.Elements, manifest.PackageRecord{
			Name: fields[0], Version: fields[1], Release: fields[2], Arch: fields[3],
		})
	}
	return scope, nil
}

// Files implements BEHAVIOR/INTERNAL: converge-files. In this version it writes
// and deletes REGULAR files; symlink convergence and type-transition handling
// are reserved for a later milestone (see decisions hints [reserved-0.7.0]).
func (c *Converger) Files(ctx *txn.Context, diff manifest.Diff) *manifest.Diagnostic {
	// 1. write declared files
	for _, e := range diff.FilesWrite {
		if e.Type != "file" {
			// symlink/dir convergence deferred; skip non-file records.
			continue
		}
		content, rerr := c.resolveContent(e)
		if rerr != nil {
			d := manifest.NewError(manifest.DomainFiles, "content resolution failed for "+e.Name+": "+rerr.Error())
			return &d
		}
		dst := filepath.Join(ctx.Root, e.Name)
		if mkerr := os.MkdirAll(filepath.Dir(dst), 0755); mkerr != nil {
			d := manifest.NewError(manifest.DomainFiles, "cannot create directory for "+e.Name+": "+mkerr.Error())
			return &d
		}
		mode := parseMode(e.Mode)
		if werr := os.WriteFile(dst, content, mode); werr != nil {
			d := manifest.NewError(manifest.DomainFiles, "cannot write "+e.Name+": "+werr.Error())
			return &d
		}
		// verify the written content hashes to e.sha256
		if e.SHA256 != "" {
			sum := sha256.Sum256(content)
			if hex.EncodeToString(sum[:]) != e.SHA256 {
				d := manifest.NewError(manifest.DomainFiles, "written content hash mismatch for "+e.Name)
				return &d
			}
		}
	}
	// 2. delete dropped files (excluding RPM-owned, keep-listed, syncpoint)
	for _, p := range diff.FilesDelete {
		if p == syncpoint || c.KeepList[p] || c.rpmOwned(ctx.Root, p) {
			continue
		}
		dst := filepath.Join(ctx.Root, p)
		if rerr := os.Remove(dst); rerr != nil && !os.IsNotExist(rerr) {
			d := manifest.NewError(manifest.DomainFiles, "cannot delete "+p+": "+rerr.Error())
			return &d
		}
	}
	return nil
}

// Units implements BEHAVIOR/INTERNAL: converge-units via offline enablement
// against ctx.root, avoiding first-boot preset evaluation.
func (c *Converger) Units(ctx *txn.Context, diff manifest.Diff) *manifest.Diagnostic {
	for _, u := range diff.UnitsChange {
		var verb string
		switch u.State {
		case "enabled":
			verb = "enable"
		case "disabled":
			verb = "disable"
		case "masked":
			verb = "mask"
		default:
			d := manifest.NewError(manifest.DomainUnits, "unknown unit state for "+u.Name+": "+u.State)
			return &d
		}
		args := []string{"--root", ctx.Root, verb, u.Name}
		if _, stderr, err := c.run("systemctl", args...); err != nil {
			d := manifest.NewError(manifest.DomainUnits, "offline enablement failed for "+u.Name+": "+strings.TrimSpace(stderr))
			return &d
		}
	}
	return nil
}

func (c *Converger) resolveContent(e manifest.ManagedFileRecord) ([]byte, error) {
	if e.ContentRef == "" {
		return nil, fmt.Errorf("no content_ref for %s", e.Name)
	}
	path := e.ContentRef
	if !filepath.IsAbs(path) && c.ContentStore != "" {
		path = filepath.Join(c.ContentStore, e.ContentRef)
	}
	return os.ReadFile(path)
}

func (c *Converger) rpmOwned(root, path string) bool {
	out, _, _ := c.run("rpm", rpmRootArgs(root, "-qf", path)...)
	o := strings.TrimSpace(out)
	return o != "" && !strings.Contains(o, "not owned") && !strings.HasPrefix(o, "error")
}

func (c *Converger) run(cmd string, args ...string) (string, string, error) {
	return c.Runner.Run(cmd, args)
}

func (c *Converger) zypperRootArgs(root string, rest ...string) []string {
	if root != "" && root != "/" {
		return append([]string{"--root", root}, rest...)
	}
	return rest
}

func rpmRootArgs(root string, rest ...string) []string {
	if root != "" && root != "/" {
		return append([]string{"--root", root}, rest...)
	}
	return rest
}

func parseMode(m string) os.FileMode {
	if m == "" {
		return 0644
	}
	var v uint64
	for _, ch := range m {
		if ch < '0' || ch > '7' {
			return 0644
		}
		v = v*8 + uint64(ch-'0')
	}
	return os.FileMode(v) & os.ModePerm
}
