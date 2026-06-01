// generated from spec: zypper-declarative.spec.md sha256:f2cc80627e483a48bb8411d297711bc5f6c6e74c28dbf0dafc8fe7bd8817251e
//
// The five behaviour verbs: apply, diff, verify, status, describe. Each
// orchestrates the internal behaviours in the spec's STEPS order and maps a
// returned Diagnostic to an exit code.
package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mge1512/zypper-declarative/internal/converge"
	"github.com/mge1512/zypper-declarative/internal/diag"
	"github.com/mge1512/zypper-declarative/internal/diff"
	"github.com/mge1512/zypper-declarative/internal/manifest"
	"github.com/mge1512/zypper-declarative/internal/record"
	"github.com/mge1512/zypper-declarative/internal/state"
	"github.com/mge1512/zypper-declarative/internal/txn"
)

// loadKeepList reads the keep-list file into a path set (empty if unset).
func (c *Config) loadKeepList() map[string]bool {
	out := map[string]bool{}
	if c.KeepList == "" {
		return out
	}
	data, err := os.ReadFile(c.KeepList)
	if err != nil {
		return out
	}
	for _, line := range strings.Split(string(data), "\n") {
		p := strings.TrimSpace(line)
		if p != "" && !strings.HasPrefix(p, "#") {
			out[p] = true
		}
	}
	return out
}

func (c *Config) loadOptions() manifest.LoadOptions {
	return manifest.LoadOptions{
		ExplicitFormat: c.explicitFormat(),
		DefaultFormat:  c.defaultFormat(),
		SignatureCheck: c.SignatureVerify == "on",
		Keyring:        c.Keyring,
	}
}

// ---- apply ----

func runApply(cfg *Config, rest []string, io IO) int {
	if rejectExtra(io, "apply", rest) {
		return ExitInvocation
	}
	// 1. Load desired manifest.
	res, err := manifest.LoadDesiredManifest(cfg.ManifestPath, cfg.loadOptions())
	if err != nil {
		d := err.(*diag.Diagnostic)
		emitDiagnostics(io.Stderr, []*diag.Diagnostic{d})
		return exitForDomain(d)
	}
	desired := res.Manifest
	desiredSHA := res.DesiredSHA256

	// 2. Load applied record (absence -> empty).
	ar, err := record.LoadAppliedRecord(cfg.AppliedRoot)
	if err != nil {
		d := err.(*diag.Diagnostic)
		emitDiagnostics(io.Stderr, []*diag.Diagnostic{d})
		return exitForDomain(d)
	}

	// 3. Compute intent diff.
	intent := diff.ComputeIntentDiff(desired, ar.Record)
	keep := cfg.loadKeepList()

	// 4. If intent empty, check drift; if also empty, "nothing to do".
	if intent.Empty() {
		sres, derr := state.DescribeActualState("/", state.Options{
			OnUnreadable: state.OnUnreadableError, Scope: state.ScopeEtc, KeepList: keep,
		})
		if derr != nil {
			d := derr.(*diag.Diagnostic)
			emitDiagnostics(io.Stderr, []*diag.Diagnostic{d})
			return exitForDomain(d)
		}
		drift := diff.ComputeDrift(sres.Manifest, ar.Record, keep)
		if drift.Empty() {
			fmt.Fprintln(io.Stdout, "nothing to do")
			return ExitOK
		}
	}

	// 5. Acquire transaction context.
	mode, _ := txn.ParseMode(cfg.Mode)
	ctx, td := txn.Acquire(mode)
	if td != nil {
		emitDiagnostics(io.Stderr, []*diag.Diagnostic{td})
		return exitForDomain(td)
	}

	copts := converge.Options{
		ContentStore: cfg.ContentStore, KeepList: keep, RepoLock: cfg.RepoLock,
	}

	// 6. Converge packages (repositories applied within).
	resolved, pd := converge.ConvergePackages(ctx, intent, copts)
	if pd != nil {
		emitDiagnostics(io.Stderr, []*diag.Diagnostic{pd})
		return ExitError // transaction discarded
	}

	// 7. Converge files.
	if fd := converge.ConvergeFiles(ctx, intent, copts); fd != nil {
		emitDiagnostics(io.Stderr, []*diag.Diagnostic{fd})
		return ExitError
	}

	// 8. Converge units.
	if ud := converge.ConvergeUnits(ctx, intent, copts); ud != nil {
		emitDiagnostics(io.Stderr, []*diag.Diagnostic{ud})
		return ExitError
	}

	// 9. Write applied record.
	if wd := record.WriteAppliedRecord(ctx.Root, desired, desiredSHA, resolved); wd != nil {
		d := wd.(*diag.Diagnostic)
		emitDiagnostics(io.Stderr, []*diag.Diagnostic{d})
		return ExitError
	}

	// 10. Post-converge verification against the new applied record.
	newAR, _ := record.LoadAppliedRecord(ctx.Root)
	vres, verr := state.DescribeActualState(ctx.Root, state.Options{
		OnUnreadable: state.OnUnreadableError, Scope: state.ScopeEtc, KeepList: keep,
	})
	if verr != nil {
		d := verr.(*diag.Diagnostic)
		emitDiagnostics(io.Stderr, []*diag.Diagnostic{d})
		return ExitError
	}
	postDrift := diff.ComputeDrift(vres.Manifest, newAR.Record, keep)
	if !postDrift.Empty() {
		emitDiagnostics(io.Stderr, []*diag.Diagnostic{
			diag.New(diag.DomainFiles, "post-converge verification found drift"),
		})
		return ExitError
	}

	// 11. Seal and activate (delegated); emit summary.
	fmt.Fprintf(io.Stdout, "applied: %d package(s) resolved, %d file(s) written, %d unit(s) changed\n",
		len(resolved.Elements), len(intent.FilesWrite), len(intent.UnitsChange))
	return ExitOK
}

// ---- diff ----

func runDiff(cfg *Config, rest []string, io IO) int {
	if rejectExtra(io, "diff", rest) {
		return ExitInvocation
	}
	// 1. Load desired manifest.
	res, err := manifest.LoadDesiredManifest(cfg.ManifestPath, cfg.loadOptions())
	if err != nil {
		d := err.(*diag.Diagnostic)
		emitDiagnostics(io.Stderr, []*diag.Diagnostic{d})
		return exitForDomain(d)
	}
	desired := res.Manifest

	// 2. Load applied record (absence -> empty).
	ar, err := record.LoadAppliedRecord(cfg.AppliedRoot)
	if err != nil {
		d := err.(*diag.Diagnostic)
		emitDiagnostics(io.Stderr, []*diag.Diagnostic{d})
		return exitForDomain(d)
	}

	// 3. Compute intent diff.
	intent := diff.ComputeIntentDiff(desired, ar.Record)
	keep := cfg.loadKeepList()

	// 4. Obtain actual state for drift: supplied dump or live read.
	var actual *manifest.Manifest
	if cfg.StatePath != "" {
		actual, err = manifest.ParseDump(cfg.StatePath, cfg.explicitFormat(), cfg.defaultFormat())
		if err != nil {
			d := err.(*diag.Diagnostic)
			emitDiagnostics(io.Stderr, []*diag.Diagnostic{d})
			return exitForDomain(d)
		}
	} else {
		sres, derr := state.DescribeActualState("/", state.Options{
			OnUnreadable: state.OnUnreadableError, Scope: state.ScopeEtc, KeepList: keep,
		})
		if derr != nil {
			d := derr.(*diag.Diagnostic)
			emitDiagnostics(io.Stderr, []*diag.Diagnostic{d})
			return exitForDomain(d)
		}
		actual = sres.Manifest
	}
	drift := diff.ComputeDrift(actual, ar.Record, keep)

	// 5. Print the combined plan.
	printPlan(io.Stdout, intent, drift)
	return ExitOK
}

// printPlan writes the intent diff and the drift report to w.
func printPlan(w io.Writer, intent *diff.Diff, drift *diff.DriftReport) {
	fmt.Fprintln(w, "packages to install:")
	for _, p := range intent.PackagesInstall {
		fmt.Fprintf(w, "  + %s\n", p.Name)
	}
	fmt.Fprintln(w, "packages to remove:")
	for _, p := range intent.PackagesRemove {
		fmt.Fprintf(w, "  - %s\n", p.Name)
	}
	fmt.Fprintln(w, "repositories to set:")
	for _, r := range intent.ReposSet {
		fmt.Fprintf(w, "  = %s\n", r.Alias)
	}
	fmt.Fprintln(w, "files to write:")
	for _, e := range intent.FilesWrite {
		fmt.Fprintf(w, "  > %s\n", e.Name)
	}
	fmt.Fprintln(w, "files to delete:")
	for _, p := range intent.FilesDelete {
		fmt.Fprintf(w, "  x %s\n", p)
	}
	fmt.Fprintln(w, "units to change:")
	for _, u := range intent.UnitsChange {
		fmt.Fprintf(w, "  ! %s -> %s\n", u.Name, u.State)
	}
	fmt.Fprintln(w, "current drift:")
	for _, p := range drift.FilesModified {
		fmt.Fprintf(w, "  ~ %s (modified)\n", p)
	}
	for _, p := range drift.FilesExtra {
		fmt.Fprintf(w, "  ? %s (extra)\n", p)
	}
	for _, u := range drift.UnitsDivergent {
		fmt.Fprintf(w, "  ! %s (divergent)\n", u.Name)
	}
}

// emitDrift writes one diagnostic per drift item to stderr.
func emitDrift(w io.Writer, r *diff.DriftReport) {
	for _, p := range r.FilesModified {
		fmt.Fprintln(w, diag.New(diag.DomainFiles, "drift: %s modified", p).Error())
	}
	for _, p := range r.FilesExtra {
		fmt.Fprintln(w, diag.New(diag.DomainFiles, "drift: %s extra (unpackaged, undeclared)", p).Error())
	}
	for _, u := range r.UnitsDivergent {
		fmt.Fprintln(w, diag.New(diag.DomainServices, "drift: %s state divergent (declared %s)", u.Name, u.State).Error())
	}
	for _, p := range r.PackagesDivergent {
		fmt.Fprintln(w, diag.New(diag.DomainPackages, "drift: %s package divergent", p.Name).Error())
	}
	for _, p := range r.ManagedFilesModified {
		fmt.Fprintln(w, diag.New(diag.DomainFiles, "drift: %s managed file modified", p).Error())
	}
	for _, p := range r.UnmanagedFilesPresent {
		fmt.Fprintln(w, diag.New(diag.DomainFiles, "drift: %s unmanaged file present", p).Error())
	}
}

// ---- status ----

func runStatus(cfg *Config, rest []string, io IO) int {
	// 1. Reject any unrecognised argument.
	if rejectExtra(io, "status", rest) {
		return ExitInvocation
	}
	// 2. Load applied record.
	ar, err := record.LoadAppliedRecord(cfg.AppliedRoot)
	if err != nil {
		d := err.(*diag.Diagnostic)
		emitDiagnostics(io.Stderr, []*diag.Diagnostic{d})
		return exitForDomain(d)
	}
	if !ar.Present {
		fmt.Fprintln(io.Stdout, "no declaration applied")
		return ExitOK
	}
	// 3. Print record summary.
	rec := ar.Record
	pkgCount := 0
	if rec.Packages != nil {
		pkgCount = len(rec.Packages.Elements)
	}
	fmt.Fprintf(io.Stdout, "desired_sha256: %s\n", rec.Meta.DesiredSHA256)
	fmt.Fprintf(io.Stdout, "format_version: %d\n", rec.Meta.FormatVersion)
	fmt.Fprintf(io.Stdout, "generation: %s\n", cfg.AppliedRoot)
	fmt.Fprintf(io.Stdout, "created_at: %s\n", rec.Meta.CreatedAt)
	fmt.Fprintf(io.Stdout, "packages: %d resolved\n", pkgCount)

	// 4. Drift summary.
	keep := cfg.loadKeepList()
	sres, derr := state.DescribeActualState("/", state.Options{
		OnUnreadable: state.OnUnreadableError, Scope: state.ScopeEtc, KeepList: keep,
	})
	if derr != nil {
		d := derr.(*diag.Diagnostic)
		emitDiagnostics(io.Stderr, []*diag.Diagnostic{d})
		return exitForDomain(d)
	}
	drift := diff.ComputeDrift(sres.Manifest, rec, keep)
	n := driftCount(drift)
	if n == 0 {
		fmt.Fprintln(io.Stdout, "clean")
	} else {
		fmt.Fprintf(io.Stdout, "%d drift item(s)\n", n)
	}
	return ExitOK
}

// ---- verify ----

func runVerify(cfg *Config, rest []string, io IO) int {
	if rejectExtra(io, "verify", rest) {
		return ExitInvocation
	}
	keep := cfg.loadKeepList()

	// 1. Determine the reference.
	var reference *manifest.Manifest
	if cfg.ManifestPath != "" {
		res, err := manifest.LoadDesiredManifest(cfg.ManifestPath, cfg.loadOptions())
		if err != nil {
			d := err.(*diag.Diagnostic)
			emitDiagnostics(io.Stderr, []*diag.Diagnostic{d})
			return exitForDomain(d)
		}
		reference = res.Manifest
	} else {
		ar, err := record.LoadAppliedRecord(cfg.AppliedRoot)
		if err != nil {
			d := err.(*diag.Diagnostic)
			emitDiagnostics(io.Stderr, []*diag.Diagnostic{d})
			return exitForDomain(d)
		}
		if !ar.Present {
			emitDiagnostics(io.Stderr, []*diag.Diagnostic{
				diag.New(diag.DomainInvocation, "no declaration applied"),
			})
			return ExitInvocation
		}
		reference = ar.Record
	}

	// 2. Obtain the actual state.
	var actual *manifest.Manifest
	if cfg.StatePath != "" {
		m, err := manifest.ParseDump(cfg.StatePath, cfg.explicitFormat(), cfg.defaultFormat())
		if err != nil {
			d := err.(*diag.Diagnostic)
			emitDiagnostics(io.Stderr, []*diag.Diagnostic{d})
			return exitForDomain(d)
		}
		actual = m
	} else {
		sres, derr := state.DescribeActualState("/", state.Options{
			OnUnreadable: state.OnUnreadableError, Scope: cfg.scopeVal(), KeepList: keep,
		})
		if derr != nil {
			d := derr.(*diag.Diagnostic)
			emitDiagnostics(io.Stderr, []*diag.Diagnostic{d})
			return exitForDomain(d)
		}
		actual = sres.Manifest
	}

	// 3. Compute drift.
	drift := diff.ComputeDrift(actual, reference, keep)

	// 4. Report.
	if drift.Empty() {
		fmt.Fprintln(io.Stdout, "system matches declaration")
		return ExitOK
	}
	emitDrift(io.Stderr, drift)
	return ExitError
}

// ---- describe ----

func runDescribe(cfg *Config, rest []string, io IO) int {
	// 1. Reject unrecognised argument or unknown format (format validated in parse).
	if rejectExtra(io, "describe", rest) {
		return ExitInvocation
	}
	keep := cfg.loadKeepList()

	// 2. Obtain actual state.
	sres, derr := state.DescribeActualState(cfg.Root, state.Options{
		OnUnreadable: cfg.onUnreadable(), Scope: cfg.scopeVal(), KeepList: keep,
	})
	if derr != nil {
		d := derr.(*diag.Diagnostic)
		emitDiagnostics(io.Stderr, []*diag.Diagnostic{d})
		return exitForDomain(d) // unreadable source under error -> exit 1 (files/etc domain)
	}
	// Emit any warn diagnostics.
	emitDiagnostics(io.Stderr, sres.Diagnostics)

	// 3. Resolve output format.
	f := manifest.ResolveFormat(cfg.explicitFormat(), cfg.Out, cfg.defaultFormat())

	// 4. Serialise.
	out, serr := sres.Manifest.Serialise(f)
	if serr != nil {
		emitDiagnostics(io.Stderr, []*diag.Diagnostic{
			diag.New(diag.DomainInvocation, "cannot serialise manifest: %v", serr),
		})
		return ExitInvocation
	}

	// 5. Write to out or stdout.
	if cfg.Out != "" {
		if werr := os.WriteFile(cfg.Out, out, 0o644); werr != nil {
			emitDiagnostics(io.Stderr, []*diag.Diagnostic{
				diag.New(diag.DomainInvocation, "output path unwritable: %s: %v", cfg.Out, werr),
			})
			return ExitInvocation
		}
		return ExitOK
	}
	io.Stdout.Write(out)
	return ExitOK
}

// driftCount counts the total drift items in a report.
func driftCount(r *diff.DriftReport) int {
	return len(r.FilesModified) + len(r.FilesExtra) + len(r.UnitsDivergent) +
		len(r.PackagesDivergent) + len(r.ManagedFilesModified) + len(r.UnmanagedFilesPresent)
}
