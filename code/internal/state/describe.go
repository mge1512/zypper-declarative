// generated from spec: zypper-declarative.spec.md sha256:f8ff76ecbc4bbc69a49e2e32b2924da3a64df1ad46196e05ce8c137b684429b2
//
// Package state implements BEHAVIOR/INTERNAL: describe-actual-state, the single
// live-state reader. It is the only code that reads live system state. Every
// verb that needs actual state obtains it here (or through a supplied dump in
// the same format). Reads are file-and-database level (no network refresh, no
// daemon, no privileged cache).
package state

import (
	"sort"

	"github.com/mge1512/zypper-declarative/internal/diag"
	"github.com/mge1512/zypper-declarative/internal/manifest"
	"github.com/mge1512/zypper-declarative/internal/meta"
)

// OnUnreadable selects how a source that cannot be read is treated.
type OnUnreadable string

const (
	OnUnreadableError OnUnreadable = "error"
	OnUnreadableWarn  OnUnreadable = "warn"
)

// Result is the output of Describe: the actual-state Manifest, any warn
// diagnostics, and an error diagnostic on the first unreadable source under the
// strict policy.
type Result struct {
	Manifest    manifest.Manifest
	Diagnostics []*diag.Diagnostic
	Err         *diag.Diagnostic
}

// Reader reads the individual scopes from a root. Production wires this to the
// OS; tests of the implementation may substitute a double, but the independent
// (black-box) suite drives the built binary instead.
type Reader interface {
	// ReadPackages returns the rpmdb-reported installed set, fully resolved, or
	// (nil, sourceErr) if the source cannot be read.
	ReadPackages(root string) ([]manifest.PackageRecord, *SourceError)
	// ReadRepositories returns the on-disk zypp repository configuration.
	ReadRepositories(root string) ([]manifest.RepositoryRecord, *SourceError)
	// ReadServices returns declarable unit enablement states.
	ReadServices(root string) ([]manifest.ServiceRecord, *SourceError)
	// ReadConfigFiles returns changed-from-package and unpackaged /etc files.
	ReadConfigFiles(root string, keepList map[string]bool) ([]manifest.ManagedFileRecord, *SourceError)
}

// SourceError names an unreadable scope source and its domain.
type SourceError struct {
	Domain  diag.Domain
	Source  string
	Wrapped error
}

func (e *SourceError) Error() string {
	if e.Wrapped != nil {
		return e.Source + ": " + e.Wrapped.Error()
	}
	return e.Source
}

// Describe implements describe-actual-state. The Manifest carries
// meta.format_version 1, generator, created_at, and empty desired_sha256. A
// scope whose readable content is genuinely empty is omitted (left unmanaged); a
// scope whose source is unreadable is never represented empty — under error it
// fails the run, under warn it is omitted with a diagnostic.
func Describe(r Reader, root string, on OnUnreadable, keepList map[string]bool) Result {
	res := Result{
		Manifest: manifest.Manifest{
			Meta: manifest.Meta{
				FormatVersion: 1,
				Generator:     meta.Generator(),
				CreatedAt:     nowRFC3339(),
				DesiredSHA256: "",
			},
		},
	}

	// packages
	if pkgs, serr := r.ReadPackages(root); serr != nil {
		if res.handleUnreadable(on, serr) {
			return res
		}
	} else if len(pkgs) > 0 {
		sort.SliceStable(pkgs, func(i, j int) bool { return pkgs[i].Name < pkgs[j].Name })
		res.Manifest.Packages = &manifest.PackagesScope{
			Attributes: manifest.ScopeAttributes{"package_system": "rpm"},
			Elements:   pkgs,
		}
	}

	// repositories
	if repos, serr := r.ReadRepositories(root); serr != nil {
		if res.handleUnreadable(on, serr) {
			return res
		}
	} else if len(repos) > 0 {
		sort.SliceStable(repos, func(i, j int) bool { return repos[i].Alias < repos[j].Alias })
		res.Manifest.Repositories = &manifest.RepositoriesScope{
			Attributes: manifest.ScopeAttributes{"repository_system": "zypp"},
			Elements:   repos,
		}
	}

	// services
	if svcs, serr := r.ReadServices(root); serr != nil {
		if res.handleUnreadable(on, serr) {
			return res
		}
	} else if len(svcs) > 0 {
		sort.SliceStable(svcs, func(i, j int) bool { return svcs[i].Name < svcs[j].Name })
		res.Manifest.Services = &manifest.ServicesScope{
			Attributes: manifest.ScopeAttributes{"init_system": "systemd"},
			Elements:   svcs,
		}
	}

	// config_files
	if files, serr := r.ReadConfigFiles(root, keepList); serr != nil {
		if res.handleUnreadable(on, serr) {
			return res
		}
	} else if len(files) > 0 {
		sort.SliceStable(files, func(i, j int) bool { return files[i].Name < files[j].Name })
		res.Manifest.ConfigFiles = &manifest.ConfigFilesScope{
			Attributes: nil,
			Elements:   files,
		}
	}

	return res
}

// handleUnreadable records an unreadable source per the policy. It returns true
// if the caller should stop (strict error), false to continue (warn).
func (res *Result) handleUnreadable(on OnUnreadable, serr *SourceError) bool {
	if on == OnUnreadableWarn {
		res.Diagnostics = append(res.Diagnostics,
			diag.Warnf(serr.Domain, "scope source omitted (unreadable): %s", serr.Error()))
		return false
	}
	res.Err = diag.Errorf(serr.Domain, "unreadable scope source: %s", serr.Error())
	return true
}
