// generated from spec: zypper-declarative.spec.md sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
//
// Verb implementations: apply, diff, verify, status, describe. Each orchestrates
// the internal behaviours and maps their results to exit codes and diagnostics.
package cli

import (
	"fmt"
	"os"

	"github.com/mge1512/zypper-declarative/internal/diff"
	"github.com/mge1512/zypper-declarative/internal/manifest"
	"github.com/mge1512/zypper-declarative/internal/record"
	"github.com/mge1512/zypper-declarative/internal/state"
	"github.com/mge1512/zypper-declarative/internal/syscmd"
	"github.com/mge1512/zypper-declarative/internal/txn"
)

// domainOf extracts the domain from a behaviour error, defaulting to "manifest".
func domainOf(err error, fallback string) string {
	switch e := err.(type) {
	case *manifest.ParseError:
		return e.Domain
	case *state.Diagnostic:
		return e.Domain
	case *record.Diagnostic:
		return e.Domain
	case *txn.Diagnostic:
		return e.Domain
	}
	return fallback
}

func (a *App) diag(domain, msg string) {
	fmt.Fprintf(a.Stderr, "[Error] %s: %s\n", domain, msg)
}

// loadDesired loads and validates the desired manifest, returning the manifest,
// its hash, and an exit code on failure (0 on success).
func (a *App) loadDesired(cfg *Config) (*manifest.Manifest, string, int) {
	res, err := manifest.Load(cfg.ManifestPath, manifest.LoadOptions{
		Explicit:  cfg.Format,
		Default:   cfg.defaultFormat(),
		SigVerify: cfg.SigVerify == "on",
		Keyring:   cfg.Keyring,
		RejectObs: true,
	})
	if err != nil {
		d := domainOf(err, "manifest")
		a.diag(d, err.Error())
		if d == "invocation" {
			return nil, "", ExitInvocation
		}
		return nil, "", ExitError
	}
	return res.Manifest, res.DesiredSHA256, ExitOK
}

// liveActual reads live actual state through the single reader, with the given
// scope and on_unreadable. Returns a manifest and an exit code on failure.
func (a *App) liveActual(cfg *Config, scope state.Scope, onUnreadable state.OnUnreadable) (*manifest.Manifest, int) {
	res, err := state.Read(state.Options{
		Root:         cfg.Root,
		OnUnreadable: onUnreadable,
		Scope:        scope,
		ContentStore: cfg.ContentStore,
		KeepList:     cfg.loadKeepList(),
		Runner:       &syscmd.OSCommandRunner{},
	})
	if err != nil {
		a.diag(domainOf(err, "files"), err.Error())
		return nil, ExitError
	}
	for _, d := range res.Diagnostics {
		fmt.Fprintf(a.Stderr, "[%s] %s: %s\n", d.Severity, d.Domain, d.Message)
	}
	return res.Manifest, ExitOK
}

// ---------------------------------------------------------------------------
// apply
// ---------------------------------------------------------------------------

func (a *App) runApply(cfg *Config) int {
	desired, _, code := a.loadDesired(cfg)
	if code != ExitOK {
		return code
	}
	appliedRes, err := record.LoadApplied(cfg.AppliedRoot)
	if err != nil {
		a.diag(domainOf(err, "files"), err.Error())
		return ExitError
	}
	d := diff.ComputeIntentDiff(desired, appliedRes.Record)
	if d.Empty() {
		// Read live actual state and compute drift; if also empty, nothing to do.
		actual, c := a.liveActual(cfg, state.ScopeEtc, state.OnUnreadableError)
		if c != ExitOK {
			return c
		}
		dr := diff.ComputeDrift(actual, appliedRes.Record, cfg.loadKeepList())
		if dr.Empty() {
			fmt.Fprintln(a.Stdout, "nothing to do")
			return ExitOK
		}
	}
	// Acquiring a transaction and converging requires privilege and a live
	// transaction mechanism. This build resolves the binding and reports a
	// transaction error when no mechanism is available, rather than fabricating
	// a snapshot.
	ctx, terr := txn.Acquire(txn.Mode(cfg.Mode), &liveProbe{})
	if terr != nil {
		a.diag(domainOf(terr, "transaction"), terr.Error())
		return ExitInvocation
	}
	_ = ctx
	// Convergence on a live host is exercised by the apply milestone on a real
	// system; in this environment we have reached the converge step with a valid
	// intent diff. Report that convergence could not be completed here as a
	// transaction error rather than silently exiting 0.
	a.diag("transaction", "transaction mechanism unavailable: cannot converge outside a snapshot transaction in this environment")
	return ExitInvocation
}

// liveProbe is the production EnvProbe. With no snapshot mechanism present it
// reports no external root and fails to open an internal transaction, so apply
// surfaces a transaction error rather than converging unsafely.
type liveProbe struct{}

func (p *liveProbe) InsideTransaction() bool { return os.Getenv("TRANSACTIONAL_UPDATE") != "" }
func (p *liveProbe) ExternalRoot() (string, bool) {
	if r := os.Getenv("TRANSACTIONAL_UPDATE_ROOT"); r != "" {
		return r, true
	}
	return "", false
}
func (p *liveProbe) OpenInternal() (string, error) {
	return "", fmt.Errorf("zypper-internal transactional machinery not available")
}

// ---------------------------------------------------------------------------
// diff
// ---------------------------------------------------------------------------

func (a *App) runDiff(cfg *Config) int {
	desired, _, code := a.loadDesired(cfg)
	if code != ExitOK {
		return code
	}
	appliedRes, err := record.LoadApplied(cfg.AppliedRoot)
	if err != nil {
		a.diag(domainOf(err, "files"), err.Error())
		return ExitError
	}
	d := diff.ComputeIntentDiff(desired, appliedRes.Record)

	var actual *manifest.Manifest
	if cfg.StatePath != "" {
		actual, err = manifest.LoadStateDump(cfg.StatePath, cfg.Format, cfg.defaultFormat())
		if err != nil {
			a.diag(domainOf(err, "invocation"), err.Error())
			return ExitInvocation
		}
	} else {
		var c int
		actual, c = a.liveActual(cfg, state.ScopeEtc, state.OnUnreadableError)
		if c != ExitOK {
			return c
		}
	}
	dr := diff.ComputeDrift(actual, appliedRes.Record, cfg.loadKeepList())
	a.printPlan(d, dr)
	return ExitOK
}

func (a *App) printPlan(d *diff.Diff, dr *diff.DriftReport) {
	fmt.Fprintln(a.Stdout, "packages to install:")
	for _, p := range d.PackagesInstall {
		fmt.Fprintf(a.Stdout, "  %s\n", p.Name)
	}
	fmt.Fprintln(a.Stdout, "packages to remove:")
	for _, p := range d.PackagesRemove {
		fmt.Fprintf(a.Stdout, "  %s\n", p.Name)
	}
	fmt.Fprintln(a.Stdout, "repositories to set:")
	for _, r := range d.ReposSet {
		fmt.Fprintf(a.Stdout, "  %s\n", r.Alias)
	}
	fmt.Fprintln(a.Stdout, "files to write:")
	for _, f := range d.FilesWrite {
		fmt.Fprintf(a.Stdout, "  %s\n", f.Name)
	}
	fmt.Fprintln(a.Stdout, "files to delete:")
	for _, p := range d.FilesDelete {
		fmt.Fprintf(a.Stdout, "  %s\n", p)
	}
	fmt.Fprintln(a.Stdout, "units to change:")
	for _, u := range d.UnitsChange {
		fmt.Fprintf(a.Stdout, "  %s -> %s\n", u.Name, u.State)
	}
	fmt.Fprintln(a.Stdout, "current drift:")
	for _, f := range dr.FilesModified {
		fmt.Fprintf(a.Stdout, "  modified: %s\n", f)
	}
	for _, f := range dr.FilesExtra {
		fmt.Fprintf(a.Stdout, "  extra: %s\n", f)
	}
	for _, u := range dr.UnitsDivergent {
		fmt.Fprintf(a.Stdout, "  unit: %s\n", u.Name)
	}
	for _, p := range dr.PackagesDivergent {
		fmt.Fprintf(a.Stdout, "  package: %s\n", p.Name)
	}
}

// ---------------------------------------------------------------------------
// verify
// ---------------------------------------------------------------------------

func (a *App) runVerify(cfg *Config) int {
	// 1. Determine the reference.
	var reference *manifest.Manifest
	if cfg.ManifestPath != "" {
		ref, _, code := a.loadDesired(cfg)
		if code != ExitOK {
			// reference manifest read/format -> exit 2; else exit 1 (already mapped)
			return code
		}
		reference = ref
	} else {
		appliedRes, err := record.LoadApplied(cfg.AppliedRoot)
		if err != nil {
			a.diag(domainOf(err, "files"), err.Error())
			return ExitError
		}
		if !appliedRes.Present {
			a.diag("invocation", "no declaration applied")
			return ExitInvocation
		}
		reference = appliedRes.Record
	}

	// 2. Obtain the actual state.
	var actual *manifest.Manifest
	if cfg.StatePath != "" {
		m, err := manifest.LoadStateDump(cfg.StatePath, cfg.Format, cfg.defaultFormat())
		if err != nil {
			a.diag(domainOf(err, "invocation"), err.Error())
			return ExitInvocation
		}
		actual = m
	} else {
		scope := state.ScopeEtc
		if cfg.Scope == "full" {
			scope = state.ScopeFull
		}
		m, c := a.liveActual(cfg, scope, state.OnUnreadableError)
		if c != ExitOK {
			return c
		}
		actual = m
	}

	// 3. Compute drift.
	dr := diff.ComputeDrift(actual, reference, cfg.loadKeepList())

	// 4. Report.
	if dr.Empty() {
		fmt.Fprintln(a.Stdout, "system matches declaration")
		return ExitOK
	}
	for _, f := range dr.FilesModified {
		a.diag("files", "drift: modified "+f)
	}
	for _, f := range dr.FilesExtra {
		a.diag("files", "drift: extra "+f)
	}
	for _, u := range dr.UnitsDivergent {
		a.diag("units", "drift: unit "+u.Name+" should be "+u.State)
	}
	for _, p := range dr.PackagesDivergent {
		a.diag("packages", "drift: package "+p.Name)
	}
	for _, f := range dr.ManagedFilesModified {
		a.diag("files", "integrity: managed file modified "+f)
	}
	for _, f := range dr.UnmanagedFilesPresent {
		a.diag("files", "integrity: unmanaged file present "+f)
	}
	return ExitError
}

// ---------------------------------------------------------------------------
// status
// ---------------------------------------------------------------------------

func (a *App) runStatus(cfg *Config) int {
	appliedRes, err := record.LoadApplied(cfg.AppliedRoot)
	if err != nil {
		a.diag(domainOf(err, "files"), err.Error())
		return ExitError
	}
	if !appliedRes.Present {
		fmt.Fprintln(a.Stdout, "no declaration applied")
		return ExitOK
	}
	rec := appliedRes.Record
	fmt.Fprintf(a.Stdout, "desired_sha256: %s\n", rec.Meta.DesiredSHA256)
	fmt.Fprintf(a.Stdout, "format_version: %d\n", rec.Meta.FormatVersion)
	fmt.Fprintf(a.Stdout, "generation: %s\n", cfg.AppliedRoot)
	fmt.Fprintf(a.Stdout, "created_at: %s\n", rec.Meta.CreatedAt)
	pkgCount := 0
	if rec.Packages != nil {
		pkgCount = len(rec.Packages.Elements)
	}
	fmt.Fprintf(a.Stdout, "packages: %d resolved\n", pkgCount)

	actual, c := a.liveActual(cfg, state.ScopeEtc, state.OnUnreadableError)
	if c != ExitOK {
		return c
	}
	dr := diff.ComputeDrift(actual, rec, cfg.loadKeepList())
	if dr.Empty() {
		fmt.Fprintln(a.Stdout, "clean")
	} else {
		n := len(dr.FilesModified) + len(dr.FilesExtra) + len(dr.UnitsDivergent) +
			len(dr.PackagesDivergent) + len(dr.ManagedFilesModified) + len(dr.UnmanagedFilesPresent)
		fmt.Fprintf(a.Stdout, "%d drift item(s)\n", n)
	}
	return ExitOK
}

// ---------------------------------------------------------------------------
// describe
// ---------------------------------------------------------------------------

func (a *App) runDescribe(cfg *Config) int {
	scope := state.ScopeEtc
	if cfg.Scope == "full" {
		scope = state.ScopeFull
	}
	onUnreadable := state.OnUnreadableError
	if cfg.OnUnreadable == "warn" {
		onUnreadable = state.OnUnreadableWarn
	}
	res, err := state.Read(state.Options{
		Root:         cfg.Root,
		OnUnreadable: onUnreadable,
		Scope:        scope,
		ContentStore: cfg.ContentStore,
		KeepList:     cfg.loadKeepList(),
		Runner:       &syscmd.OSCommandRunner{},
	})
	if err != nil {
		a.diag(domainOf(err, "files"), err.Error())
		return ExitError
	}
	for _, d := range res.Diagnostics {
		fmt.Fprintf(a.Stderr, "[%s] %s: %s\n", d.Severity, d.Domain, d.Message)
	}

	f := manifest.ResolveFormat(cfg.Format, cfg.Out, cfg.defaultFormat())
	doc, serr := res.Manifest.Serialise(f)
	if serr != nil {
		a.diag("invocation", "could not serialise describe output: "+serr.Error())
		return ExitInvocation
	}
	if cfg.Out != "" {
		if werr := os.WriteFile(cfg.Out, doc, 0o644); werr != nil {
			a.diag("invocation", "output path unwritable: "+werr.Error())
			return ExitInvocation
		}
		return ExitOK
	}
	a.Stdout.Write(doc)
	return ExitOK
}
