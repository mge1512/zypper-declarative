// generated from spec: zypper-declarative.spec.md sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
//
// Package converge implements converge-packages, converge-files, and
// converge-units. The convergence code path is identical regardless of which
// transaction binding (external/internal) was resolved.
package converge

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"

	"github.com/mge1512/zypper-declarative/internal/diff"
	"github.com/mge1512/zypper-declarative/internal/manifest"
	"github.com/mge1512/zypper-declarative/internal/syscmd"
	"github.com/mge1512/zypper-declarative/internal/txn"
)

// Diagnostic is a domain-tagged convergence error.
type Diagnostic struct {
	Severity string
	Domain   string
	Message  string
}

func (d *Diagnostic) Error() string { return d.Message }

// Packages implements BEHAVIOR/INTERNAL: converge-packages. It configures the
// declared repositories (or the CONFIG pin), installs and removes packages within
// the context root via the package manager, and returns the rpmdb-reported
// resolved installed set (the lock).
func Packages(ctx *txn.Context, d *diff.Diff, runner syscmd.CommandRunner, repoLock string) (*manifest.ScopeWrapper[manifest.PackageRecord], error) {
	if runner == nil {
		runner = &syscmd.OSCommandRunner{}
	}
	root := ctx.Root

	// 1. Ensure repositories (or the CONFIG pin) are configured.
	if len(d.ReposSet) == 0 && repoLock != "" {
		if _, stderr, err := runner.Run("zypper", []string{"--root", root, "addrepo", "--check", repoLock, "repo-lock"}); err != nil {
			return nil, &Diagnostic{Severity: "Error", Domain: "repositories", Message: "repository configuration failed: " + strings.TrimSpace(stderr)}
		}
	}

	// 2. Install.
	if len(d.PackagesInstall) > 0 {
		args := []string{"--root", root, "--non-interactive", "install", "--no-recommends"}
		for _, p := range d.PackagesInstall {
			args = append(args, p.Name)
		}
		if _, stderr, err := runner.Run("zypper", args); err != nil {
			return nil, &Diagnostic{Severity: "Error", Domain: "packages", Message: "package install failed: " + strings.TrimSpace(stderr)}
		}
	}

	// 3. Remove.
	if len(d.PackagesRemove) > 0 {
		args := []string{"--root", root, "--non-interactive", "remove"}
		for _, p := range d.PackagesRemove {
			args = append(args, p.Name)
		}
		if _, stderr, err := runner.Run("zypper", args); err != nil {
			return nil, &Diagnostic{Severity: "Error", Domain: "packages", Message: "package remove failed: " + strings.TrimSpace(stderr)}
		}
	}

	// 4. Query rpmdb for the full installed set (the lock).
	out, stderr, err := runner.Run("rpm", []string{"--root", root, "-qa", "--queryformat", "%{NAME} %{VERSION} %{RELEASE} %{ARCH}\n"})
	if err != nil && strings.TrimSpace(out) == "" {
		return nil, &Diagnostic{Severity: "Error", Domain: "packages", Message: "rpmdb query after convergence failed: " + strings.TrimSpace(stderr)}
	}
	scope := manifest.NewScope[manifest.PackageRecord](map[string]interface{}{"package_system": "rpm"})
	for _, line := range strings.Split(out, "\n") {
		f := strings.Fields(strings.TrimSpace(line))
		if len(f) < 4 {
			continue
		}
		scope.Elements = append(scope.Elements, manifest.PackageRecord{Name: f[0], Version: f[1], Release: f[2], Arch: f[3]})
	}
	return &scope, nil
}

// Files implements BEHAVIOR/INTERNAL: converge-files. In this version it writes
// and deletes regular files only; symlink convergence and type-transition
// handling are deferred (see the spec's reserved note).
func Files(ctx *txn.Context, d *diff.Diff, contentStore string, ownedByRPM func(path string) bool, keepList map[string]bool) error {
	if keepList == nil {
		keepList = map[string]bool{}
	}
	root := ctx.Root

	// 1. Writes.
	for _, e := range d.FilesWrite {
		if e.Type != "file" {
			continue // symlink/dir convergence is deferred
		}
		content, err := resolveContent(contentStore, e)
		if err != nil {
			return &Diagnostic{Severity: "Error", Domain: "files", Message: "content resolution failed for " + e.Name + ": " + err.Error()}
		}
		dest := filepath.Join(root, e.Name)
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return &Diagnostic{Severity: "Error", Domain: "files", Message: "could not create directory for " + e.Name + ": " + err.Error()}
		}
		mode := parseMode(e.Mode)
		if err := os.WriteFile(dest, content, mode); err != nil {
			return &Diagnostic{Severity: "Error", Domain: "files", Message: "write failed for " + e.Name + ": " + err.Error()}
		}
		// Verify the written content hashes to the declared sha256.
		if e.SHA256 != "" {
			sum := sha256.Sum256(content)
			if hex.EncodeToString(sum[:]) != e.SHA256 {
				return &Diagnostic{Severity: "Error", Domain: "files", Message: "written content hash mismatch for " + e.Name}
			}
		}
	}

	// 2. Deletes (skip RPM-owned, keep-listed, and /etc/etc.syncpoint).
	for _, p := range d.FilesDelete {
		if p == "/etc/etc.syncpoint" || keepList[p] {
			continue
		}
		if ownedByRPM != nil && ownedByRPM(p) {
			continue
		}
		dest := filepath.Join(root, p)
		if err := os.Remove(dest); err != nil && !os.IsNotExist(err) {
			return &Diagnostic{Severity: "Error", Domain: "files", Message: "delete failed for " + p + ": " + err.Error()}
		}
	}
	return nil
}

// resolveContent resolves a record's content via its content_ref against the
// content store. With no content store or no content_ref there is nothing to
// resolve and an empty file is written (the record carries no inline content).
func resolveContent(contentStore string, e manifest.ManagedFileRecord) ([]byte, error) {
	if e.ContentRef == "" || contentStore == "" {
		return []byte{}, nil
	}
	ref := strings.TrimPrefix(e.ContentRef, "sha256/")
	return os.ReadFile(filepath.Join(contentStore, "sha256", ref))
}

func parseMode(m string) os.FileMode {
	if m == "" {
		return 0o644
	}
	var v int64
	for _, c := range m {
		if c < '0' || c > '7' {
			return 0o644
		}
		v = v*8 + int64(c-'0')
	}
	return os.FileMode(v)
}

// Units implements BEHAVIOR/INTERNAL: converge-units using offline enablement
// against the context root.
func Units(ctx *txn.Context, d *diff.Diff, runner syscmd.CommandRunner) error {
	if runner == nil {
		runner = &syscmd.OSCommandRunner{}
	}
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
			continue
		}
		if _, stderr, err := runner.Run("systemctl", []string{"--root", root, verb, u.Name}); err != nil {
			return &Diagnostic{Severity: "Error", Domain: "units", Message: "offline enablement failed for " + u.Name + ": " + strings.TrimSpace(stderr)}
		}
	}
	return nil
}
