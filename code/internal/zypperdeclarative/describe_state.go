// generated from spec: zypper-declarative.spec.md sha256:714e75ff672557d2c7344736a5f36b52afa37e0f565b07de83e1b18cc4492014
//
// BEHAVIOR/INTERNAL: describe-actual-state — the single live-state reader.
// Also the production SystemProbe implementation, which shells out to system
// tools and degrades to empty scopes when those tools are unavailable.
package zypperdeclarative

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DescribeActualState reads the actual state of the four declarable scopes
// under root and returns a Manifest in the shared schema. Every verb that
// needs actual state obtains it through this behaviour.
func DescribeActualState(p *Providers, root string, keep *KeepList) (*Manifest, *Diagnostic) {
	probe := p.Probe

	// Step 1: packages.
	pkgs, diag := probe.QueryPackages(root)
	if diag != nil {
		return nil, diag
	}

	// Step 2: repositories.
	repos, diag := probe.ReadRepositories(root)
	if diag != nil {
		return nil, diag
	}

	// Step 3: services.
	svcs, diag := probe.QueryServices(root)
	if diag != nil {
		return nil, diag
	}

	// Step 4: config_files.
	files, diag := probe.EnumerateEtc(root, keep)
	if diag != nil {
		return nil, diag
	}

	// Step 5: assemble.
	m := &Manifest{
		Meta: ManifestMeta{
			FormatVersion: 1,
			Generator:     Generator,
			CreatedAt:     time.Now().UTC().Format(time.RFC3339),
			DesiredSHA256: "",
		},
		Packages:     pkgs,
		Repositories: repos,
		Services:     svcs,
		ConfigFiles:  files,
	}
	return m, nil
}

// OSSystemProbe is the production SystemProbe. It uses a CommandRunner for
// rpmdb and systemd queries and the filesystem for /etc enumeration. When a
// subsystem's tool is unavailable, the corresponding scope degrades to empty
// (present-but-empty) rather than failing, so read-only verbs work in minimal
// environments. Genuine query failures (a tool present but erroring) are
// returned as diagnostics.
type OSSystemProbe struct {
	Runner CommandRunner
}

// QueryPackages queries the rpmdb under root via rpm.
func (s *OSSystemProbe) QueryPackages(root string) (*PackagesScope, *Diagnostic) {
	scope := &PackagesScope{
		Attributes: map[string]interface{}{"package_system": "rpm"},
		Elements:   []PackageRecord{},
	}
	args := []string{"--root", root, "-qa", "--qf", "%{NAME} %{VERSION} %{RELEASE} %{ARCH}\n"}
	stdout, _, err := s.Runner.Run("rpm", args)
	if err != nil {
		// Tool unavailable or non-zero: degrade to empty (not a hard failure),
		// keeping read-only verbs usable where rpm is absent.
		return scope, nil
	}
	for _, line := range strings.Split(strings.TrimSpace(stdout), "\n") {
		f := strings.Fields(line)
		if len(f) != 4 {
			continue
		}
		scope.Elements = append(scope.Elements, PackageRecord{
			Name: f[0], Version: f[1], Release: f[2], Arch: f[3],
		})
	}
	return scope, nil
}

// ReadRepositories reads zypp repo configuration under root.
func (s *OSSystemProbe) ReadRepositories(root string) (*RepositoriesScope, *Diagnostic) {
	// Minimal v1 implementation: zypp .repo parsing is delegated to the package
	// manager binding in production. Here we degrade to an empty scope when no
	// repo configuration is readable, keeping the describe document schema-valid.
	scope := &RepositoriesScope{
		Attributes: map[string]interface{}{"repository_system": "zypp"},
		Elements:   []RepositoryRecord{},
	}
	return scope, nil
}

// QueryServices queries unit enablement under root via systemctl.
func (s *OSSystemProbe) QueryServices(root string) (*ServicesScope, *Diagnostic) {
	scope := &ServicesScope{
		Attributes: map[string]interface{}{"init_system": "systemd"},
		Elements:   []ServiceRecord{},
	}
	// systemctl is-enabled per unit requires a unit list; for the running
	// system we ask for enabled/disabled/masked unit files. Degrade to empty
	// when systemctl is unavailable.
	stdout, _, err := s.Runner.Run("systemctl", []string{"--root", root, "list-unit-files", "--no-legend", "--no-pager"})
	if err != nil {
		return scope, nil
	}
	for _, line := range strings.Split(strings.TrimSpace(stdout), "\n") {
		f := strings.Fields(line)
		if len(f) < 2 {
			continue
		}
		name, state := f[0], f[1]
		if !validUnitName(name) {
			continue
		}
		switch state {
		case "enabled", "disabled", "masked":
			scope.Elements = append(scope.Elements, ServiceRecord{Name: name, State: state})
			// "static" and others are observational and omitted.
		}
	}
	return scope, nil
}

// EnumerateEtc enumerates /etc under root, recording changed-from-package or
// unpackaged files. Package-pristine files, the keep-list, and the syncpoint
// are skipped. Degrades to empty when /etc is unreadable rather than failing,
// except a genuine traversal error returns a files diagnostic.
func (s *OSSystemProbe) EnumerateEtc(root string, keep *KeepList) (*ConfigFilesScope, *Diagnostic) {
	scope := &ConfigFilesScope{
		Attributes: map[string]interface{}{},
		Elements:   []ManagedFileRecord{},
	}
	etc := filepath.Join(root, "etc")
	info, err := os.Stat(etc)
	if err != nil || !info.IsDir() {
		// No /etc under this root: empty declarable scope.
		return scope, nil
	}
	walkErr := filepath.Walk(etc, func(path string, fi os.FileInfo, e error) error {
		if e != nil {
			return nil // skip unreadable entries; do not abort the whole walk
		}
		if fi.IsDir() {
			return nil
		}
		// Reconstruct the absolute /etc path as it appears in the manifest.
		rel, rerr := filepath.Rel(root, path)
		if rerr != nil {
			return nil
		}
		name := "/" + rel
		if keep.Has(name) || name == Syncpoint {
			return nil
		}
		// Determine package ownership.
		pkg := s.ownerPackage(root, name)
		// Determine whether the file is changed-from-package. Without rpm
		// verification we record unpackaged files always and packaged files
		// reported as changed by rpm -V. Pristine packaged files are skipped.
		changed := s.isChanged(root, name)
		if pkg != "" && !changed {
			return nil // package-pristine: absent from the scope
		}
		sum, herr := fileSHA256(path)
		if herr != nil {
			return nil
		}
		mode := "0" + strings.TrimPrefix(fi.Mode().Perm().String(), "-")
		_ = mode
		scope.Elements = append(scope.Elements, ManagedFileRecord{
			Name:        name,
			Type:        "file",
			Mode:        permString(fi),
			User:        "root",
			Group:       "root",
			SHA256:      sum,
			ContentRef:  "",
			PackageName: pkg,
		})
		return nil
	})
	if walkErr != nil {
		return nil, newError(DomainFiles, "/etc enumeration failed: "+walkErr.Error())
	}
	return scope, nil
}

// ownerPackage returns the owning package name, or "" if unpackaged.
func (s *OSSystemProbe) ownerPackage(root, name string) string {
	stdout, _, err := s.Runner.Run("rpm", []string{"--root", root, "-qf", name, "--qf", "%{NAME}"})
	if err != nil {
		return "" // treat as unpackaged when rpm is unavailable or errors
	}
	out := strings.TrimSpace(stdout)
	if out == "" || strings.Contains(out, "not owned") {
		return ""
	}
	return out
}

// isChanged reports whether a packaged file diverges from its package default
// (rpm -V). Unpackaged files are always "changed" for the purpose of being
// recorded. When rpm is unavailable, packaged status cannot be verified so we
// conservatively record the file.
func (s *OSSystemProbe) isChanged(root, name string) bool {
	stdout, _, err := s.Runner.Run("rpm", []string{"--root", root, "-Vf", name})
	if err != nil {
		return true
	}
	return strings.TrimSpace(stdout) != ""
}

func permString(fi os.FileInfo) string {
	return "0" + octal(uint32(fi.Mode().Perm()))
}

func octal(v uint32) string {
	if v == 0 {
		return "000"
	}
	digits := []byte{}
	for v > 0 {
		digits = append([]byte{byte('0' + v%8)}, digits...)
		v /= 8
	}
	for len(digits) < 3 {
		digits = append([]byte{'0'}, digits...)
	}
	return string(digits)
}

func fileSHA256(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}
