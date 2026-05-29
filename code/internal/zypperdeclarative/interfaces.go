// generated from spec: zypper-declarative.spec.md sha256:714e75ff672557d2c7344736a5f36b52afa37e0f565b07de83e1b18cc4492014
//
// INTERFACES: abstract dependencies on external systems (package manager,
// snapshot/filesystem, init system, transaction mechanism). Each is an
// interface with a production implementation and a test double, so the
// behaviours can be exercised without live system tooling.
package zypperdeclarative

import (
	"bytes"
	"os"
	"os/exec"
)

// CommandRunner runs an external command and returns stdout, stderr, error.
type CommandRunner interface {
	Run(cmd string, args []string) (stdout string, stderr string, err error)
}

// OSCommandRunner runs commands against the real OS with a sanitised PATH.
type OSCommandRunner struct{}

// Run executes the command. Implemented in full (never a stub) per the
// scaffold hints, so command-dependent behaviours do not silently no-op.
func (r *OSCommandRunner) Run(cmd string, args []string) (string, string, error) {
	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", "/sbin:/bin:/usr/bin:/usr/sbin")
	defer os.Setenv("PATH", oldPath)

	c := exec.Command(cmd, args...)
	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr
	err := c.Run()
	return stdout.String(), stderr.String(), err
}

// FakeCommandRunner is a test double returning canned results.
type FakeCommandRunner struct {
	Results map[string]struct {
		Stdout string
		Stderr string
		Err    error
	}
}

// Run returns the canned result keyed by command name, or empty success.
func (r *FakeCommandRunner) Run(cmd string, args []string) (string, string, error) {
	if r.Results != nil {
		if res, ok := r.Results[cmd]; ok {
			return res.Stdout, res.Stderr, res.Err
		}
	}
	return "", "", nil
}

// PackageManager abstracts libzypp/zypper operations within a context root.
type PackageManager interface {
	ConfigureRepositories(root string, repos []RepositoryRecord, fallbackLock string) *Diagnostic
	Install(root string, pkgs []PackageRecord) *Diagnostic
	Remove(root string, pkgs []PackageRecord) *Diagnostic
	QueryInstalled(root string) (*PackagesScope, *Diagnostic)
}

// SnapshotProvider abstracts btrfs/snapper operations.
type SnapshotProvider interface {
	// DetectInTransaction reports whether the process already runs inside a
	// fresh snapshot transaction, and the writable new-generation root.
	DetectInTransaction() (inside bool, root string)
	// OpenInternal opens a new snapshot transaction (internal mode).
	OpenInternal() (root string, diag *Diagnostic)
	// Seal seals and marks the snapshot the default boot target.
	Seal(root string, activationPolicy string) *Diagnostic
	// StampUserdata records key=value in the snapshot's snapper userdata.
	StampUserdata(root, key, value string) *Diagnostic
}

// InitSystem abstracts systemd unit operations.
type InitSystem interface {
	// QueryEnablement returns each unit's normalised state under root.
	QueryEnablement(root string) ([]ServiceRecord, *Diagnostic)
	// SetEnablementOffline applies a unit's declared state offline.
	SetEnablementOffline(root string, svc ServiceRecord) *Diagnostic
}

// SystemProbe abstracts the live-state reads that describe-actual-state needs
// for the rpmdb, repositories, and /etc enumeration. The production
// implementation shells out to system tools; when those tools are absent it
// degrades to empty scopes (a system without the queried subsystem reports an
// empty declarable scope, not an error), so read-only verbs remain usable in
// minimal environments.
type SystemProbe interface {
	QueryPackages(root string) (*PackagesScope, *Diagnostic)
	ReadRepositories(root string) (*RepositoriesScope, *Diagnostic)
	QueryServices(root string) (*ServicesScope, *Diagnostic)
	EnumerateEtc(root string, keep *KeepList) (*ConfigFilesScope, *Diagnostic)
}

// Providers bundles the external-system bindings used by the behaviours.
type Providers struct {
	Runner   CommandRunner
	Packages PackageManager
	Snapshot SnapshotProvider
	Init     InitSystem
	Probe    SystemProbe
}
