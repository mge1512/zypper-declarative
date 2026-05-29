// generated from spec: zypper-declarative.spec.md sha256:b2d0de88fbed1163678e59e931c741b9d999b71f902f6eb01db8790bb813d057
//
// The five CLI verbs: apply, diff, verify, status, describe. Each orchestrates
// the internal behaviours and maps their returned diagnostics to exit codes.
package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mge1512/zypper-declarative/internal/converge"
	"github.com/mge1512/zypper-declarative/internal/diff"
	"github.com/mge1512/zypper-declarative/internal/manifest"
	"github.com/mge1512/zypper-declarative/internal/record"
	"github.com/mge1512/zypper-declarative/internal/state"
	"github.com/mge1512/zypper-declarative/internal/sysexec"
	"github.com/mge1512/zypper-declarative/internal/txn"
)

func runner() sysexec.CommandRunner { return &sysexec.OSCommandRunner{} }

func loadKeepList(path string) map[string]bool {
	out := map[string]bool{}
	if path == "" {
		return out
	}
	f, err := os.Open(path)
	if err != nil {
		return out
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			out[line] = true
		}
	}
	return out
}

func loadOpts(cfg Config) manifest.LoadOptions {
	return manifest.LoadOptions{
		ExplicitFormat:        cfg.ExplicitFormat,
		DefaultFormat:         cfg.ManifestFormat,
		SignatureVerification: cfg.SignatureVerification,
		Keyring:               cfg.Keyring,
	}
}

// emitDiagnostics writes each diagnostic to stderr, one per line.
func emitDiagnostics(stderr io.Writer, ds ...*manifest.Diagnostic) {
	for _, d := range ds {
		if d != nil {
			fmt.Fprintln(stderr, d.Error())
		}
	}
}

// runDiff implements BEHAVIOR: diff.
func runDiff(cfg Config, stdout, stderr io.Writer) int {
	lr, d := manifest.LoadDesiredManifest(cfg.ManifestPath, loadOpts(cfg))
	if d != nil {
		emitDiagnostics(stderr, d)
		return exitForLoad(d)
	}
	applied, ad := record.LoadAppliedRecord(cfg.AppliedRoot)
	if ad != nil {
		emitDiagnostics(stderr, ad)
		return 1
	}
	intent := diff.ComputeIntentDiff(lr.Manifest, applied.Record)

	keep := loadKeepList(cfg.KeepList)
	actual, sd := state.Describe(state.Options{
		Root: "/", OnUnreadable: state.OnUnreadableError, Scope: state.ScopeEtc,
		Runner: runner(), KeepList: keep,
	})
	if sd != nil {
		emitDiagnostics(stderr, sd)
		// state collection failure is a logical failure.
		return 1
	}
	drift := diff.ComputeDrift(actual.Manifest, applied.Record, keep)

	printPlan(stdout, intent, drift)
	return 0
}

func printPlan(w io.Writer, d manifest.Diff, dr manifest.DriftReport) {
	fmt.Fprintln(w, "packages to install:")
	for _, p := range d.PackagesInstall {
		fmt.Fprintf(w, "  %s\n", p.Name)
	}
	fmt.Fprintln(w, "packages to remove:")
	for _, p := range d.PackagesRemove {
		fmt.Fprintf(w, "  %s\n", p.Name)
	}
	fmt.Fprintln(w, "repositories to set:")
	for _, r := range d.ReposSet {
		fmt.Fprintf(w, "  %s\n", r.Alias)
	}
	fmt.Fprintln(w, "files to write:")
	for _, f := range d.FilesWrite {
		fmt.Fprintf(w, "  %s\n", f.Name)
	}
	fmt.Fprintln(w, "files to delete:")
	for _, f := range d.FilesDelete {
		fmt.Fprintf(w, "  %s\n", f)
	}
	fmt.Fprintln(w, "units to change:")
	for _, u := range d.UnitsChange {
		fmt.Fprintf(w, "  %s -> %s\n", u.Name, u.State)
	}
	fmt.Fprintln(w, "current drift:")
	for _, p := range dr.FilesModified {
		fmt.Fprintf(w, "  modified: %s\n", p)
	}
	for _, p := range dr.FilesExtra {
		fmt.Fprintf(w, "  extra: %s\n", p)
	}
	for _, u := range dr.UnitsDivergent {
		fmt.Fprintf(w, "  unit-divergent: %s\n", u.Name)
	}
	for _, p := range dr.PackagesDivergent {
		fmt.Fprintf(w, "  package-divergent: %s\n", p.Name)
	}
}

// runVerify implements BEHAVIOR: verify.
func runVerify(cfg Config, stdout, stderr io.Writer) int {
	applied, ad := record.LoadAppliedRecord(cfg.AppliedRoot)
	if ad != nil {
		emitDiagnostics(stderr, ad)
		return 1
	}
	if !applied.Present {
		fmt.Fprintf(stderr, "%s: no declaration applied\n", manifest.DomainInvocation)
		return 2
	}

	keep := loadKeepList(cfg.KeepList)
	var actual *manifest.Manifest
	if cfg.StatePath != "" {
		m, d := manifest.LoadStateDump(cfg.StatePath, cfg.ExplicitFormat, cfg.ManifestFormat)
		if d != nil {
			emitDiagnostics(stderr, d)
			return 2
		}
		actual = m
	} else {
		sc := state.ScopeEtc
		if cfg.Scope == "full" {
			sc = state.ScopeFull
		}
		res, sd := state.Describe(state.Options{
			Root: "/", OnUnreadable: state.OnUnreadableError, Scope: sc,
			Runner: runner(), KeepList: keep,
		})
		if sd != nil {
			emitDiagnostics(stderr, sd)
			return 1
		}
		actual = res.Manifest
	}

	dr := diff.ComputeDrift(actual, applied.Record, keep)
	if dr.Empty() {
		fmt.Fprintln(stdout, "system matches declaration")
		return 0
	}
	emitDriftDiagnostics(stderr, dr)
	return 1
}

func emitDriftDiagnostics(stderr io.Writer, dr manifest.DriftReport) {
	for _, p := range dr.FilesModified {
		fmt.Fprintln(stderr, (&manifest.Diagnostic{Severity: manifest.SeverityError, Domain: manifest.DomainFiles, Message: "drift: " + p}).Error())
	}
	for _, p := range dr.FilesExtra {
		fmt.Fprintln(stderr, (&manifest.Diagnostic{Severity: manifest.SeverityError, Domain: manifest.DomainFiles, Message: "extra: " + p}).Error())
	}
	for _, u := range dr.UnitsDivergent {
		fmt.Fprintln(stderr, (&manifest.Diagnostic{Severity: manifest.SeverityError, Domain: manifest.DomainUnits, Message: "drift: " + u.Name}).Error())
	}
	for _, p := range dr.PackagesDivergent {
		fmt.Fprintln(stderr, (&manifest.Diagnostic{Severity: manifest.SeverityError, Domain: manifest.DomainPackages, Message: "drift: " + p.Name}).Error())
	}
	for _, p := range dr.ManagedFilesModified {
		fmt.Fprintln(stderr, (&manifest.Diagnostic{Severity: manifest.SeverityError, Domain: manifest.DomainFiles, Message: "managed-file-modified: " + p}).Error())
	}
	for _, p := range dr.UnmanagedFilesPresent {
		fmt.Fprintln(stderr, (&manifest.Diagnostic{Severity: manifest.SeverityError, Domain: manifest.DomainFiles, Message: "unmanaged-file-present: " + p}).Error())
	}
}

// runStatus implements BEHAVIOR: status.
func runStatus(cfg Config, stdout, stderr io.Writer) int {
	applied, ad := record.LoadAppliedRecord(cfg.AppliedRoot)
	if ad != nil {
		emitDiagnostics(stderr, ad)
		return 1
	}
	if !applied.Present {
		fmt.Fprintln(stdout, "no declaration applied")
		return 0
	}
	rec := applied.Record
	pkgCount := 0
	if rec.Packages != nil {
		pkgCount = len(rec.Packages.Elements)
	}
	fmt.Fprintf(stdout, "desired_sha256: %s\n", rec.Meta.DesiredSHA256)
	fmt.Fprintf(stdout, "format_version: %d\n", rec.Meta.FormatVersion)
	fmt.Fprintf(stdout, "generation: %s\n", cfg.AppliedRoot)
	fmt.Fprintf(stdout, "created_at: %s\n", rec.Meta.CreatedAt)
	fmt.Fprintf(stdout, "packages: %d resolved\n", pkgCount)

	keep := loadKeepList(cfg.KeepList)
	res, sd := state.Describe(state.Options{
		Root: "/", OnUnreadable: state.OnUnreadableError, Scope: state.ScopeEtc,
		Runner: runner(), KeepList: keep,
	})
	if sd != nil {
		// status is read-only and exits 0 on valid invocation; surface the
		// drift line as unknown rather than failing on a live-read limitation.
		fmt.Fprintln(stdout, "drift: unknown (actual state unavailable)")
		return 0
	}
	dr := diff.ComputeDrift(res.Manifest, rec, keep)
	if dr.Empty() {
		fmt.Fprintln(stdout, "drift: clean")
	} else {
		fmt.Fprintf(stdout, "drift: %d drift item(s)\n", driftCount(dr))
	}
	return 0
}

func driftCount(dr manifest.DriftReport) int {
	return len(dr.FilesModified) + len(dr.FilesExtra) + len(dr.UnitsDivergent) +
		len(dr.PackagesDivergent) + len(dr.ManagedFilesModified) + len(dr.UnmanagedFilesPresent)
}

// runDescribe implements BEHAVIOR: describe.
func runDescribe(cfg Config, stdout, stderr io.Writer) int {
	// Resolve output format via resolve-format(format, out).
	f, fd := manifest.ResolveFormat(cfg.ExplicitFormat, cfg.Out, cfg.ManifestFormat)
	if fd != nil {
		emitDiagnostics(stderr, fd)
		printUsage(stderr)
		return 2
	}

	sc := state.ScopeEtc
	if cfg.Scope == "full" {
		sc = state.ScopeFull
	}
	onUnreadable := state.OnUnreadableError
	if cfg.OnUnreadable == "warn" {
		onUnreadable = state.OnUnreadableWarn
	}
	keep := loadKeepList(cfg.KeepList)
	res, sd := state.Describe(state.Options{
		Root: cfg.Root, OnUnreadable: onUnreadable, Scope: sc,
		Runner: runner(), KeepList: keep,
	})
	if sd != nil {
		emitDiagnostics(stderr, sd)
		return 1
	}
	emitDiagnostics(stderr, res.Diagnostics...)

	doc, serr := manifest.Serialise(res.Manifest, f)
	if serr != nil {
		fmt.Fprintf(stderr, "%s: serialising describe output: %v\n", manifest.DomainInvocation, serr)
		return 2
	}
	if cfg.Out != "" {
		if err := os.WriteFile(cfg.Out, append(doc, '\n'), 0o644); err != nil {
			fmt.Fprintf(stderr, "%s: output path unwritable: %v\n", manifest.DomainInvocation, err)
			return 2
		}
		return 0
	}
	stdout.Write(doc)
	if len(doc) == 0 || doc[len(doc)-1] != '\n' {
		fmt.Fprintln(stdout)
	}
	return 0
}

// runApply implements BEHAVIOR: apply.
func runApply(cfg Config, stdout, stderr io.Writer) int {
	// 1. Load desired manifest.
	lr, d := manifest.LoadDesiredManifest(cfg.ManifestPath, loadOpts(cfg))
	if d != nil {
		emitDiagnostics(stderr, d)
		return exitForLoad(d)
	}
	// 2. Load applied record.
	applied, ad := record.LoadAppliedRecord(cfg.AppliedRoot)
	if ad != nil {
		emitDiagnostics(stderr, ad)
		return 1
	}
	// 3. Intent diff.
	intent := diff.ComputeIntentDiff(lr.Manifest, applied.Record)

	keep := loadKeepList(cfg.KeepList)

	// 4. If intent empty, check drift; if also empty, nothing to do.
	if intent.Empty() {
		res, sd := state.Describe(state.Options{
			Root: "/", OnUnreadable: state.OnUnreadableError, Scope: state.ScopeEtc,
			Runner: runner(), KeepList: keep,
		})
		if sd != nil {
			emitDiagnostics(stderr, sd)
			return 1
		}
		dr := diff.ComputeDrift(res.Manifest, applied.Record, keep)
		if dr.Empty() {
			fmt.Fprintln(stdout, "nothing to do")
			return 0
		}
	}

	// 5. Acquire transaction context.
	ctx, td := txn.Acquire(txn.Mode(cfg.Mode), txn.EnvBinding{})
	if td != nil {
		emitDiagnostics(stderr, td)
		return 2 // transaction mechanism unavailable -> exit 2
	}

	copts := converge.Options{
		Runner: runner(), RepoLock: cfg.RepoLock,
		ContentStore: cfg.ContentStore, KeepList: keep,
	}

	// 6. Converge packages (after repositories).
	resolved, pd := converge.Packages(ctx, intent, copts)
	if pd != nil {
		emitDiagnostics(stderr, pd)
		return 1 // transaction discarded (opened_here snapshot is never sealed)
	}
	// 7. Converge files.
	if fd := converge.Files(ctx, intent, copts); fd != nil {
		emitDiagnostics(stderr, fd)
		return 1
	}
	// 8. Converge units.
	if ud := converge.Units(ctx, intent, copts); ud != nil {
		emitDiagnostics(stderr, ud)
		return 1
	}
	// 9. Write applied record.
	if wd := record.WriteAppliedRecord(record.WriteOptions{
		Root: ctx.Root, Desired: lr.Manifest, DesiredSHA256: lr.DesiredSHA256, Resolved: resolved,
	}); wd != nil {
		emitDiagnostics(stderr, wd)
		return 1
	}
	// 10. Post-converge verification against the new applied record.
	newApplied, _ := record.LoadAppliedRecord(ctx.Root)
	res, sd := state.Describe(state.Options{
		Root: ctx.Root, OnUnreadable: state.OnUnreadableError, Scope: state.ScopeEtc,
		Runner: runner(), KeepList: keep,
	})
	if sd != nil {
		emitDiagnostics(stderr, sd)
		return 1
	}
	if newApplied != nil {
		dr := diff.ComputeDrift(res.Manifest, newApplied.Record, keep)
		if !dr.Empty() {
			emitDriftDiagnostics(stderr, dr)
			return 1
		}
	}
	// 11. Seal and activate (delegated to the binding when opened_here).
	fmt.Fprintln(stdout, "converged: new generation sealed")
	printPlan(stdout, intent, manifest.DriftReport{})
	return 0
}
