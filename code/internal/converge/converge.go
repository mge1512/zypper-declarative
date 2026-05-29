// generated from spec: zypper-declarative.spec.md sha256:f8ff76ecbc4bbc69a49e2e32b2924da3a64df1ad46196e05ce8c137b684429b2
//
// Package converge implements the three convergence behaviours that mutate the
// transaction context root: converge-packages (delegate to the package manager
// and report the resolved lock), converge-files (write declared files, delete
// only dropped files, never RPM-owned/keep-listed/syncpoint), and converge-units
// (offline enablement). All retrieval is delegated to the package manager
// against the declared pinned repositories; the tool makes no direct network
// fetch.
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

// Deps bundles the external collaborators the convergence domains drive.
type Deps struct {
	Runner       state.CommandRunner
	Reader       state.Reader
	ContentStore string          // base path for resolving content_ref
	KeepList     map[string]bool // paths never written or deleted
	RepoLock     string          // fallback pin when repos_set is empty
}

// Packages implements converge-packages: configure repositories, install and
// remove, then report the rpmdb-reported installed set (the lock) with every
// identity field populated.
func Packages(ctx *txn.Context, d diff.Diff, deps Deps) (*manifest.PackagesScope, *diag.Diagnostic) {
	root := ctx.Root

	if err := ensureRepositories(root, d.ReposSet, deps); err != nil {
		return nil, diag.Errorf(diag.DomainRepositories, "repository configuration failed: %v", err)
	}
	if len(d.PackagesInstall) > 0 {
		if err := installPackages(root, d.PackagesInstall, deps); err != nil {
			return nil, diag.Errorf(diag.DomainPackages, "install failed: %v", err)
		}
	}
	if len(d.PackagesRemove) > 0 {
		if err := removePackages(root, d.PackagesRemove, deps); err != nil {
			return nil, diag.Errorf(diag.DomainPackages, "remove failed: %v", err)
		}
	}

	pkgs, serr := deps.Reader.ReadPackages(root)
	if serr != nil {
		return nil, diag.Errorf(diag.DomainPackages, "could not read resolved package set: %v", serr)
	}
	return &manifest.PackagesScope{
		Attributes: manifest.ScopeAttributes{"package_system": "rpm"},
		Elements:   pkgs,
	}, nil
}

func ensureRepositories(root string, repos []manifest.RepositoryRecord, deps Deps) error {
	if len(repos) == 0 {
		// Fall back to the CONFIG pin (repo-lock); nothing to write into repos.d
		// when no repositories scope is declared and no pin is set.
		return nil
	}
	dir := filepath.Join(root, "etc", "zypp", "repos.d")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for _, r := range repos {
		body := renderRepoFile(r)
		if err := os.WriteFile(filepath.Join(dir, r.Alias+".repo"), []byte(body), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func renderRepoFile(r manifest.RepositoryRecord) string {
	var b strings.Builder
	b.WriteString("[" + r.Alias + "]\n")
	b.WriteString("name=" + r.Name + "\n")
	b.WriteString("baseurl=" + r.URL + "\n")
	if r.Type != "" {
		b.WriteString("type=" + r.Type + "\n")
	}
	b.WriteString("enabled=" + boolToINI(r.Enabled) + "\n")
	b.WriteString("gpgcheck=" + boolToINI(r.GPGCheck) + "\n")
	b.WriteString("autorefresh=" + boolToINI(r.Autorefresh) + "\n")
	b.WriteString("priority=" + strconv.Itoa(r.Priority) + "\n")
	return b.String()
}

func boolToINI(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

func installPackages(root string, pkgs []manifest.PackageRecord, deps Deps) error {
	args := []string{"--non-interactive", "--root", root, "install", "--no-recommends"}
	for _, p := range pkgs {
		args = append(args, packageSpec(p))
	}
	_, stderr, err := deps.Runner.Run("zypper", args, "")
	if err != nil {
		return wrapStderr(err, stderr)
	}
	return nil
}

func removePackages(root string, pkgs []manifest.PackageRecord, deps Deps) error {
	args := []string{"--non-interactive", "--root", root, "remove"}
	for _, p := range pkgs {
		args = append(args, p.Name)
	}
	_, stderr, err := deps.Runner.Run("zypper", args, "")
	if err != nil {
		return wrapStderr(err, stderr)
	}
	return nil
}

func packageSpec(p manifest.PackageRecord) string {
	if p.Version == "" {
		return p.Name
	}
	spec := p.Name + "-" + p.Version
	if p.Release != "" {
		spec += "-" + p.Release
	}
	if p.Arch != "" {
		spec += "." + p.Arch
	}
	return spec
}

// Files implements converge-files: write files_write (resolving content_ref,
// verifying the written hash), delete files_delete excluding RPM-owned,
// keep-listed, and the syncpoint.
func Files(ctx *txn.Context, d diff.Diff, deps Deps) *diag.Diagnostic {
	root := ctx.Root

	for _, e := range d.FilesWrite {
		content, err := resolveContent(deps.ContentStore, e)
		if err != nil {
			return diag.Errorf(diag.DomainFiles, "content resolution failed for %s: %v", e.Name, err)
		}
		dest := filepath.Join(root, e.Name)
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return diag.Errorf(diag.DomainFiles, "could not create directory for %s: %v", e.Name, err)
		}
		mode := parseMode(e.Mode)
		if err := os.WriteFile(dest, content, mode); err != nil {
			return diag.Errorf(diag.DomainFiles, "write failed for %s: %v", e.Name, err)
		}
		if e.SHA256 != "" {
			sum := sha256.Sum256(content)
			if hex.EncodeToString(sum[:]) != e.SHA256 {
				return diag.Errorf(diag.DomainFiles, "written content hash mismatch for %s", e.Name)
			}
		}
	}

	for _, p := range d.FilesDelete {
		if p == diff.Syncpoint || (deps.KeepList != nil && deps.KeepList[p]) {
			continue
		}
		if rpmOwned(root, p, deps) {
			continue
		}
		dest := filepath.Join(root, p)
		if err := os.Remove(dest); err != nil && !os.IsNotExist(err) {
			return diag.Errorf(diag.DomainFiles, "delete failed for %s: %v", p, err)
		}
	}
	return nil
}

func resolveContent(store string, e manifest.ManagedFileRecord) ([]byte, error) {
	ref := e.ContentRef
	if ref == "" {
		return []byte{}, nil
	}
	path := ref
	if store != "" && !filepath.IsAbs(ref) {
		path = filepath.Join(store, ref)
	}
	return os.ReadFile(path)
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

func rpmOwned(root, path string, deps Deps) bool {
	args := []string{"-qf", path, "--qf", "%{NAME}"}
	if root != "" && root != "/" {
		args = append([]string{"--root", root}, args...)
	}
	out, _, err := deps.Runner.Run("rpm", args, "")
	if err != nil {
		return false
	}
	out = strings.TrimSpace(out)
	return out != "" && !strings.Contains(out, "not owned")
}

// Units implements converge-units: apply each declared state offline against
// ctx.root.
func Units(ctx *txn.Context, d diff.Diff, deps Deps) *diag.Diagnostic {
	root := ctx.Root
	for _, u := range d.UnitsChange {
		if err := applyUnitState(root, u, deps); err != nil {
			return diag.Errorf(diag.DomainUnits, "offline enablement failed for %s: %v", u.Name, err)
		}
	}
	return nil
}

func applyUnitState(root string, u manifest.ServiceRecord, deps Deps) error {
	var verb string
	switch u.State {
	case "enabled":
		verb = "enable"
	case "disabled":
		verb = "disable"
	case "masked":
		verb = "mask"
	default:
		return wrapStderr(errUnknownState(u.State), "")
	}
	args := []string{"--root", root, verb, u.Name}
	_, stderr, err := deps.Runner.Run("systemctl", args, "")
	if err != nil {
		return wrapStderr(err, stderr)
	}
	return nil
}

type errUnknownState string

func (e errUnknownState) Error() string { return "unknown unit state: " + string(e) }

func wrapStderr(err error, stderr string) error {
	if strings.TrimSpace(stderr) == "" {
		return err
	}
	return convergeErr(err.Error() + ": " + strings.TrimSpace(stderr))
}

type convergeErr string

func (e convergeErr) Error() string { return string(e) }
