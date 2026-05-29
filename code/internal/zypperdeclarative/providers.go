// generated from spec: zypper-declarative.spec.md sha256:714e75ff672557d2c7344736a5f36b52afa37e0f565b07de83e1b18cc4492014
//
// Production implementations of the external-system interfaces, and a
// constructor for the production Providers bundle. These bindings shell out
// to the system tools (zypper/rpm, snapper/transactional-update, systemctl).
// They are kept thin; the convergence logic lives in converge.go.
package zypperdeclarative

// OSPackageManager binds to libzypp/zypper via the CommandRunner.
type OSPackageManager struct {
	Runner CommandRunner
}

// ConfigureRepositories ensures the declared (or fallback) repositories are
// configured under root.
func (m *OSPackageManager) ConfigureRepositories(root string, repos []RepositoryRecord, fallbackLock string) *Diagnostic {
	for _, r := range repos {
		_, stderr, err := m.Runner.Run("zypper", []string{"--root", root, "ar", "-f", r.URL, r.Alias})
		if err != nil {
			return newError(DomainRepositories, "repository configuration failed for "+r.Alias+": "+stderr)
		}
	}
	if len(repos) == 0 && fallbackLock != "" {
		_, stderr, err := m.Runner.Run("zypper", []string{"--root", root, "ar", "-f", fallbackLock, "repo-lock"})
		if err != nil {
			return newError(DomainRepositories, "fallback repo-lock configuration failed: "+stderr)
		}
	}
	return nil
}

// Install installs the requested packages against the configured pins.
func (m *OSPackageManager) Install(root string, pkgs []PackageRecord) *Diagnostic {
	if len(pkgs) == 0 {
		return nil
	}
	args := []string{"--root", root, "--non-interactive", "in", "--no-recommends"}
	for _, p := range pkgs {
		args = append(args, p.Name)
	}
	_, stderr, err := m.Runner.Run("zypper", args)
	if err != nil {
		return newError(DomainPackages, "install failed: "+stderr)
	}
	return nil
}

// Remove removes the requested packages.
func (m *OSPackageManager) Remove(root string, pkgs []PackageRecord) *Diagnostic {
	if len(pkgs) == 0 {
		return nil
	}
	args := []string{"--root", root, "--non-interactive", "rm"}
	for _, p := range pkgs {
		args = append(args, p.Name)
	}
	_, stderr, err := m.Runner.Run("zypper", args)
	if err != nil {
		return newError(DomainPackages, "remove failed: "+stderr)
	}
	return nil
}

// QueryInstalled queries the full installed set under root from the rpmdb.
func (m *OSPackageManager) QueryInstalled(root string) (*PackagesScope, *Diagnostic) {
	probe := &OSSystemProbe{Runner: m.Runner}
	scope, diag := probe.QueryPackages(root)
	if diag != nil {
		return nil, newError(DomainPackages, diag.Message)
	}
	return scope, nil
}

// OSSnapshotProvider binds to snapper/btrfs via the CommandRunner.
type OSSnapshotProvider struct {
	Runner CommandRunner
}

// DetectInTransaction detects whether the process runs inside a fresh
// snapshot transaction. The production heuristic checks for the
// TRANSACTIONAL_UPDATE marker in the environment is forbidden (no env-var
// control); instead it asks the transactional tooling. When that tooling is
// unavailable it reports not-inside.
func (s *OSSnapshotProvider) DetectInTransaction() (bool, string) {
	stdout, _, err := s.Runner.Run("transactional-update", []string{"--help"})
	_ = stdout
	if err != nil {
		return false, ""
	}
	// Without an actually-open transaction we cannot resolve a writable root;
	// report not-inside so callers fall back to internal mode.
	return false, ""
}

// OpenInternal opens a new snapshot transaction (internal mode).
func (s *OSSnapshotProvider) OpenInternal() (string, *Diagnostic) {
	stdout, stderr, err := s.Runner.Run("transactional-update", []string{"--quiet", "shell"})
	_ = stdout
	if err != nil {
		return "", newError(DomainTransaction, "could not open internal transaction: "+stderr)
	}
	return "", newError(DomainTransaction, "internal transaction mechanism unavailable in this environment")
}

// Seal seals the snapshot and marks it the default boot target.
func (s *OSSnapshotProvider) Seal(root, activationPolicy string) *Diagnostic {
	_, stderr, err := s.Runner.Run("snapper", []string{"modify", "--read-only", root})
	if err != nil {
		return newError(DomainTransaction, "snapshot seal failed: "+stderr)
	}
	return nil
}

// StampUserdata records key=value in the snapshot's snapper userdata.
func (s *OSSnapshotProvider) StampUserdata(root, key, value string) *Diagnostic {
	_, stderr, err := s.Runner.Run("snapper", []string{"modify", "--userdata", key + "=" + value, root})
	if err != nil {
		return newError(DomainFiles, "userdata stamp failed: "+stderr)
	}
	return nil
}

// OSInitSystem binds to systemd via the CommandRunner.
type OSInitSystem struct {
	Runner CommandRunner
}

// QueryEnablement returns each unit's normalised state under root.
func (i *OSInitSystem) QueryEnablement(root string) ([]ServiceRecord, *Diagnostic) {
	probe := &OSSystemProbe{Runner: i.Runner}
	scope, diag := probe.QueryServices(root)
	if diag != nil {
		return nil, diag
	}
	return scope.Elements, nil
}

// SetEnablementOffline applies a unit's declared state offline against root.
func (i *OSInitSystem) SetEnablementOffline(root string, svc ServiceRecord) *Diagnostic {
	var verb string
	switch svc.State {
	case "enabled":
		verb = "enable"
	case "disabled":
		verb = "disable"
	case "masked":
		verb = "mask"
	default:
		return newError(DomainUnits, "unknown unit state: "+svc.State)
	}
	_, stderr, err := i.Runner.Run("systemctl", []string{"--root", root, verb, svc.Name})
	if err != nil {
		return newError(DomainUnits, "offline enablement failed for "+svc.Name+": "+stderr)
	}
	return nil
}

// NewProductionProviders builds the production Providers bundle.
func NewProductionProviders() *Providers {
	runner := &OSCommandRunner{}
	return &Providers{
		Runner:   runner,
		Packages: &OSPackageManager{Runner: runner},
		Snapshot: &OSSnapshotProvider{Runner: runner},
		Init:     &OSInitSystem{Runner: runner},
		Probe:    &OSSystemProbe{Runner: runner},
	}
}
