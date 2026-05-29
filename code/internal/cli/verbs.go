// generated from spec: zypper-declarative.spec.md sha256:f8ff76ecbc4bbc69a49e2e32b2924da3a64df1ad46196e05ce8c137b684429b2
//
// The five CLI verbs (apply, diff, verify, status, describe). Each orchestrates
// the internal behaviours and maps returned Diagnostics to exit codes per the
// spec. Actual state is always obtained through describe-actual-state (the single
// live reader) or a supplied dump.

package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mge1512/zypper-declarative/internal/converge"
	"github.com/mge1512/zypper-declarative/internal/diag"
	"github.com/mge1512/zypper-declarative/internal/diff"
	"github.com/mge1512/zypper-declarative/internal/manifest"
	"github.com/mge1512/zypper-declarative/internal/record"
	"github.com/mge1512/zypper-declarative/internal/state"
	"github.com/mge1512/zypper-declarative/internal/txn"
)

// loadDesired loads the desired manifest with the verb's resolved config.
func (a *App) loadDesired(cfg Config) (*manifest.Manifest, string, *diag.Diagnostic) {
	return manifest.Load(cfg.ManifestPath, manifest.LoadOptions{
		ExplicitFormat:      cfg.ExplicitFormat,
		ExplicitFormatGiven: cfg.ExplicitFormatGiven,
		DefaultFormat:       cfg.ManifestFormat,
		VerifySignature:     cfg.SignatureVerification,
		KeyringPath:         cfg.KeyringPath,
	})
}

// liveReader returns the production live-state reader.
func liveReader() state.Reader { return state.NewOSReader() }

// cmdDiff implements BEHAVIOR: diff (dry run; no modification, no transaction).
func (a *App) cmdDiff(cfg Config, rest []string) int {
	if len(rest) > 0 {
		a.diagUsage(diag.Errorf(diag.DomainInvocation, "unexpected argument: %s", rest[0]))
		return ExitInvocation
	}
	desired, _, derr := a.loadDesired(cfg)
	if derr != nil {
		a.emit(derr)
		return exitForInvocation(derr)
	}
	applied, _, aerr := record.Load(cfg.AppliedRoot)
	if aerr != nil {
		a.emit(aerr)
		return ExitLogical
	}
	d := diff.ComputeIntentDiff(desired, &applied)

	keep := loadKeepList(cfg.KeepListPath)
	actual := state.Describe(liveReader(), "/", state.OnUnreadableError, keep)
	var drift diff.DriftReport
	if actual.Err == nil {
		drift = diff.ComputeDrift(&actual.Manifest, &applied, keep)
	}

	a.printPlan(d, drift)
	return ExitOK
}

// printPlan prints the combined intent diff and drift report to stdout.
func (a *App) printPlan(d diff.Diff, drift diff.DriftReport) {
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
	if drift.Empty() {
		fmt.Fprintln(a.Stdout, "  clean")
	} else {
		for _, p := range drift.FilesModified {
			fmt.Fprintf(a.Stdout, "  modified %s\n", p)
		}
		for _, p := range drift.FilesExtra {
			fmt.Fprintf(a.Stdout, "  extra %s\n", p)
		}
		for _, u := range drift.UnitsDivergent {
			fmt.Fprintf(a.Stdout, "  unit %s\n", u.Name)
		}
		for _, p := range drift.PackagesDivergent {
			fmt.Fprintf(a.Stdout, "  package %s\n", p.Name)
		}
	}
}

// cmdVerify implements BEHAVIOR: verify.
func (a *App) cmdVerify(cfg Config, rest []string) int {
	if len(rest) > 0 {
		a.diagUsage(diag.Errorf(diag.DomainInvocation, "unexpected argument: %s", rest[0]))
		return ExitInvocation
	}
	applied, present, aerr := record.Load(cfg.AppliedRoot)
	if aerr != nil {
		a.emit(aerr)
		return ExitLogical
	}
	if !present {
		a.emit(diag.Errorf(diag.DomainInvocation, "no declaration applied"))
		return ExitInvocation
	}

	keep := loadKeepList(cfg.KeepListPath)

	var actual *manifest.Manifest
	if cfg.StatePath != "" {
		m, lerr := a.loadStateDump(cfg)
		if lerr != nil {
			a.emit(lerr)
			return ExitInvocation
		}
		actual = m
	} else {
		res := state.Describe(liveReader(), "/", state.OnUnreadableError, keep)
		if res.Err != nil {
			a.emit(diag.Errorf(diag.DomainInvocation, "state collection failed: %s", res.Err.Message))
			return ExitInvocation
		}
		actual = &res.Manifest
	}

	drift := diff.ComputeDrift(actual, &applied, keep)
	if drift.Empty() {
		fmt.Fprintln(a.Stdout, "system matches declaration")
		return ExitOK
	}
	for _, p := range drift.FilesModified {
		a.emit(diag.Errorf(diag.DomainFiles, "drift: file modified: %s", p))
	}
	for _, p := range drift.FilesExtra {
		a.emit(diag.Errorf(diag.DomainFiles, "drift: extra file: %s", p))
	}
	for _, u := range drift.UnitsDivergent {
		a.emit(diag.Errorf(diag.DomainUnits, "drift: unit divergent: %s (declared %s)", u.Name, u.State))
	}
	for _, p := range drift.PackagesDivergent {
		a.emit(diag.Errorf(diag.DomainPackages, "drift: package divergent: %s", p.Name))
	}
	return ExitLogical
}

// loadStateDump loads and schema-validates a supplied state dump under the
// resolved format. A malformed dump is an invocation error.
func (a *App) loadStateDump(cfg Config) (*manifest.Manifest, *diag.Diagnostic) {
	data, err := os.ReadFile(cfg.StatePath)
	if err != nil {
		return nil, diag.Errorf(diag.DomainInvocation, "state dump unreadable: %s: %v", cfg.StatePath, err)
	}
	f, _, perr := manifest.ParseFormat(string(cfg.ExplicitFormat))
	if cfg.ExplicitFormatGiven && perr != nil {
		return nil, diag.Errorf(diag.DomainInvocation, "%v", perr)
	}
	resolved := manifest.ResolveFormat(f, cfg.ExplicitFormatGiven, cfg.StatePath, cfg.ManifestFormat)
	m, derr := manifest.Decode(data, resolved)
	if derr != nil {
		return nil, diag.Errorf(diag.DomainInvocation, "malformed state dump: %v", derr)
	}
	if verr := manifest.Validate(m); verr != nil {
		return nil, diag.Errorf(diag.DomainInvocation, "malformed state dump: %v", verr)
	}
	return m, nil
}

// cmdStatus implements BEHAVIOR: status.
func (a *App) cmdStatus(cfg Config, rest []string) int {
	if len(rest) > 0 {
		a.diagUsage(diag.Errorf(diag.DomainInvocation, "unrecognised argument: %s", rest[0]))
		return ExitInvocation
	}
	applied, present, aerr := record.Load(cfg.AppliedRoot)
	if aerr != nil {
		a.emit(aerr)
		return ExitLogical
	}
	if !present {
		fmt.Fprintln(a.Stdout, "no declaration applied")
		return ExitOK
	}

	pkgCount := 0
	if applied.Packages != nil {
		pkgCount = len(applied.Packages.Elements)
	}
	fmt.Fprintf(a.Stdout, "manifest: %s\n", applied.Meta.DesiredSHA256)
	fmt.Fprintf(a.Stdout, "format_version: %d\n", applied.Meta.FormatVersion)
	fmt.Fprintf(a.Stdout, "generation: %s\n", snapshotIdentifier(cfg.AppliedRoot))
	fmt.Fprintf(a.Stdout, "created_at: %s\n", applied.Meta.CreatedAt)
	fmt.Fprintf(a.Stdout, "packages: %d resolved\n", pkgCount)

	keep := loadKeepList(cfg.KeepListPath)
	res := state.Describe(liveReader(), "/", state.OnUnreadableError, keep)
	if res.Err != nil {
		// Status is read-only and fast; if live state cannot be read, report the
		// drift line as unknown rather than failing the verb.
		fmt.Fprintln(a.Stdout, "drift: unknown (state unreadable)")
		return ExitOK
	}
	drift := diff.ComputeDrift(&res.Manifest, &applied, keep)
	if drift.Empty() {
		fmt.Fprintln(a.Stdout, "drift: clean")
	} else {
		fmt.Fprintf(a.Stdout, "drift: %d drift item(s)\n", drift.Count())
	}
	return ExitOK
}

// snapshotIdentifier returns a best-effort generation identifier for the root.
func snapshotIdentifier(root string) string {
	if root == "" || root == "/" {
		return "current"
	}
	return filepath.Base(root)
}

// cmdDescribe implements BEHAVIOR: describe.
func (a *App) cmdDescribe(cfg Config, rest []string) int {
	if len(rest) > 0 {
		a.diagUsage(diag.Errorf(diag.DomainInvocation, "unrecognised argument: %s", rest[0]))
		return ExitInvocation
	}

	keep := loadKeepList(cfg.KeepListPath)
	res := state.Describe(liveReader(), cfg.Root, cfg.OnUnreadable, keep)
	for _, d := range res.Diagnostics {
		a.emit(d)
	}
	if res.Err != nil {
		a.emit(res.Err)
		return ExitLogical
	}

	f, _, perr := manifest.ParseFormat(string(cfg.ExplicitFormat))
	if cfg.ExplicitFormatGiven && perr != nil {
		a.diagUsage(diag.Errorf(diag.DomainInvocation, "%v", perr))
		return ExitInvocation
	}
	format := manifest.ResolveFormat(f, cfg.ExplicitFormatGiven, cfg.Out, cfg.ManifestFormat)

	out, eerr := manifest.Encode(&res.Manifest, format)
	if eerr != nil {
		a.emit(diag.Errorf(diag.DomainInvocation, "could not serialise: %v", eerr))
		return ExitInvocation
	}

	if cfg.Out == "" {
		a.Stdout.Write(out)
		return ExitOK
	}
	if err := os.WriteFile(cfg.Out, out, 0o644); err != nil {
		a.emit(diag.Errorf(diag.DomainInvocation, "output path unwritable: %s: %v", cfg.Out, err))
		return ExitInvocation
	}
	return ExitOK
}

// cmdApply implements BEHAVIOR: apply.
func (a *App) cmdApply(cfg Config, rest []string) int {
	if len(rest) > 0 {
		a.diagUsage(diag.Errorf(diag.DomainInvocation, "unexpected argument: %s", rest[0]))
		return ExitInvocation
	}

	desired, desiredSHA, derr := a.loadDesired(cfg)
	if derr != nil {
		a.emit(derr)
		return exitForInvocation(derr)
	}

	applied, _, aerr := record.Load(cfg.AppliedRoot)
	if aerr != nil {
		a.emit(aerr)
		return ExitLogical
	}

	d := diff.ComputeIntentDiff(desired, &applied)
	keep := loadKeepList(cfg.KeepListPath)

	if d.Empty() {
		res := state.Describe(liveReader(), "/", state.OnUnreadableError, keep)
		if res.Err != nil {
			a.emit(res.Err)
			return ExitLogical
		}
		drift := diff.ComputeDrift(&res.Manifest, &applied, keep)
		if drift.Empty() {
			fmt.Fprintln(a.Stdout, "nothing to do")
			return ExitOK
		}
	}

	ctx, terr := txn.Acquire(cfg.Mode)
	if terr != nil {
		a.emit(terr)
		return ExitInvocation
	}

	deps := converge.Deps{
		Runner:       &state.OSCommandRunner{},
		Reader:       liveReader(),
		ContentStore: cfg.ContentStore,
		KeepList:     keep,
		RepoLock:     cfg.RepoLock,
	}

	resolved, cperr := converge.Packages(ctx, d, deps)
	if cperr != nil {
		a.emit(cperr)
		return ExitLogical
	}
	if cferr := converge.Files(ctx, d, deps); cferr != nil {
		a.emit(cferr)
		return ExitLogical
	}
	if cuerr := converge.Units(ctx, d, deps); cuerr != nil {
		a.emit(cuerr)
		return ExitLogical
	}

	if _, werr := record.Write(ctx.Root, desired, desiredSHA, resolved); werr != nil {
		a.emit(werr)
		return ExitLogical
	}

	// Post-converge verification against the new applied record.
	post := state.Describe(liveReader(), ctx.Root, state.OnUnreadableError, keep)
	if post.Err != nil {
		a.emit(post.Err)
		return ExitLogical
	}
	newApplied, _, _ := record.Load(ctx.Root)
	drift := diff.ComputeDrift(&post.Manifest, &newApplied, keep)
	if !drift.Empty() {
		a.emit(diag.Errorf(diag.DomainFiles, "post-converge verification found drift; transaction discarded"))
		return ExitLogical
	}

	fmt.Fprintf(a.Stdout, "applied %d package(s), %d file(s), %d unit change(s)\n",
		len(d.PackagesInstall), len(d.FilesWrite), len(d.UnitsChange))
	return ExitOK
}
