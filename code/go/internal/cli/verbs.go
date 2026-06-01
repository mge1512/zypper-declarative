// generated from spec: zypper-declarative.spec.md sha256:27aee8e374eb3507189bad0b78339109d0116e6a13b55ae4df2ba9a18e769fc4
//
// Verb handlers: diff, verify, status. Each parses options, orchestrates the
// internal behaviours, and maps results to an exit code.
package cli

import (
	"fmt"
	"io"
	"time"

	"github.com/mge1512/zypper-declarative/internal/diff"
	"github.com/mge1512/zypper-declarative/internal/manifest"
	"github.com/mge1512/zypper-declarative/internal/record"
)

func nowRFC3339() string { return time.Now().UTC().Format(time.RFC3339) }

// runDiff implements BEHAVIOR: diff (dry run).
func runDiff(args []string, stdout, stderr io.Writer) int {
	cfg := defaultConfig()
	if _, err := parseArgs(&cfg, args); err != nil {
		printUsage(stderr)
		return ExitInvocation
	}

	// 1. load desired manifest
	desired, _, code := loadDesiredManifest(cfg, cfg.ManifestPath, stderr)
	if code != ExitOK {
		return code
	}
	// 2. load applied record
	applied, _, ad := record.Load(cfg.AppliedRoot)
	if ad != nil {
		emitDiag(stderr, *ad)
		return ExitError
	}
	// 3. intent diff
	d := diff.ComputeIntentDiff(desired, applied)
	// 4. actual state for drift
	keep := readKeepList(cfg)
	var actual *manifest.Manifest
	if cfg.StatePath != "" {
		var c int
		actual, c = loadStateDump(cfg, cfg.StatePath, stderr)
		if c != ExitOK {
			return c
		}
	} else {
		var c int
		actual, c = readActualState(cfg, "error", "etc", keep, nowRFC3339(), stderr)
		if c != ExitOK {
			return c
		}
	}
	report := diff.ComputeDrift(actual, applied, diff.KeepList(keep))
	// 5. print plan
	printPlan(stdout, d, report)
	return ExitOK
}

// printPlan writes the combined intent diff and drift report to stdout.
func printPlan(w io.Writer, d manifest.Diff, r manifest.DriftReport) {
	fmt.Fprintln(w, "packages to install:")
	for _, p := range d.PackagesInstall {
		fmt.Fprintf(w, "  %s\n", p.Name)
	}
	fmt.Fprintln(w, "packages to remove:")
	for _, p := range d.PackagesRemove {
		fmt.Fprintf(w, "  %s\n", p.Name)
	}
	fmt.Fprintln(w, "repositories to set:")
	for _, repo := range d.ReposSet {
		fmt.Fprintf(w, "  %s\n", repo.Alias)
	}
	fmt.Fprintln(w, "files to write:")
	for _, f := range d.FilesWrite {
		fmt.Fprintf(w, "  %s\n", f.Name)
	}
	fmt.Fprintln(w, "files to delete:")
	for _, p := range d.FilesDelete {
		fmt.Fprintf(w, "  %s\n", p)
	}
	fmt.Fprintln(w, "units to change:")
	for _, u := range d.UnitsChange {
		fmt.Fprintf(w, "  %s -> %s\n", u.Name, u.State)
	}
	fmt.Fprintln(w, "drift:")
	for _, p := range r.FilesModified {
		fmt.Fprintf(w, "  modified: %s\n", p)
	}
	for _, p := range r.FilesExtra {
		fmt.Fprintf(w, "  extra: %s\n", p)
	}
	for _, u := range r.UnitsDivergent {
		fmt.Fprintf(w, "  unit divergent: %s\n", u.Name)
	}
	for _, p := range r.PackagesDivergent {
		fmt.Fprintf(w, "  package divergent: %s\n", p.Name)
	}
	for _, p := range r.ManagedFilesModified {
		fmt.Fprintf(w, "  managed file modified: %s\n", p)
	}
	for _, p := range r.UnmanagedFilesPresent {
		fmt.Fprintf(w, "  unmanaged file present: %s\n", p)
	}
}

// runVerify implements BEHAVIOR: verify.
func runVerify(args []string, stdout, stderr io.Writer) int {
	cfg := defaultConfig()
	seen, err := parseArgs(&cfg, args)
	if err != nil {
		printUsage(stderr)
		return ExitInvocation
	}

	keep := readKeepList(cfg)

	// 1. determine reference
	var reference *manifest.Manifest
	if seen["manifest-path"] {
		var code int
		reference, _, code = loadDesiredManifest(cfg, cfg.ManifestPath, stderr)
		if code != ExitOK {
			// verify maps a manifest read/format error to exit 2, else 1.
			if code == ExitInvocation {
				return ExitInvocation
			}
			return ExitError
		}
	} else {
		rec, present, rd := record.Load(cfg.AppliedRoot)
		if rd != nil {
			emitDiag(stderr, *rd)
			return ExitError
		}
		if !present {
			emitDiag(stderr, manifest.NewError(manifest.DomainInvocation, "no declaration applied"))
			return ExitInvocation
		}
		reference = rec
	}

	// 2. actual state
	var actual *manifest.Manifest
	if cfg.StatePath != "" {
		var c int
		actual, c = loadStateDump(cfg, cfg.StatePath, stderr)
		if c != ExitOK {
			return c
		}
	} else {
		var c int
		actual, c = readActualState(cfg, "error", cfg.Scope, keep, nowRFC3339(), stderr)
		if c != ExitOK {
			return c
		}
	}

	// 3. drift
	report := diff.ComputeDrift(actual, reference, diff.KeepList(keep))

	// 4. result
	if report.Empty() {
		fmt.Fprintln(stdout, "system matches declaration")
		return ExitOK
	}
	emitDriftDiagnostics(stderr, report)
	return ExitError
}

// emitDriftDiagnostics writes one diagnostic per drift item to stderr.
func emitDriftDiagnostics(stderr io.Writer, r manifest.DriftReport) {
	for _, p := range r.FilesModified {
		emitDiag(stderr, manifest.NewError(manifest.DomainFiles, "modified: "+p))
	}
	for _, p := range r.FilesExtra {
		emitDiag(stderr, manifest.NewError(manifest.DomainFiles, "extra: "+p))
	}
	for _, u := range r.UnitsDivergent {
		emitDiag(stderr, manifest.NewError(manifest.DomainUnits, "divergent service: "+u.Name))
	}
	for _, p := range r.PackagesDivergent {
		emitDiag(stderr, manifest.NewError(manifest.DomainPackages, "divergent package: "+p.Name))
	}
	for _, p := range r.ManagedFilesModified {
		emitDiag(stderr, manifest.NewError(manifest.DomainFiles, "managed file modified: "+p))
	}
	for _, p := range r.UnmanagedFilesPresent {
		emitDiag(stderr, manifest.NewError(manifest.DomainFiles, "unmanaged file present: "+p))
	}
}

// runStatus implements BEHAVIOR: status.
func runStatus(args []string, stdout, stderr io.Writer) int {
	cfg := defaultConfig()
	// status accepts no options beyond CONFIG; reject any unrecognised argument.
	if _, err := parseArgs(&cfg, args); err != nil {
		printUsage(stderr)
		return ExitInvocation
	}

	rec, present, rd := record.Load(cfg.AppliedRoot)
	if rd != nil {
		emitDiag(stderr, *rd)
		return ExitError
	}
	if !present {
		fmt.Fprintln(stdout, "no declaration applied")
		return ExitOK
	}

	pkgCount := 0
	if rec.Packages != nil {
		pkgCount = len(rec.Packages.Elements)
	}
	fmt.Fprintf(stdout, "desired_sha256: %s\n", rec.Meta.DesiredSHA256)
	fmt.Fprintf(stdout, "format_version: %d\n", rec.Meta.FormatVersion)
	fmt.Fprintf(stdout, "generation: %s\n", cfg.AppliedRoot)
	fmt.Fprintf(stdout, "created_at: %s\n", rec.Meta.CreatedAt)
	fmt.Fprintf(stdout, "packages: %d resolved\n", pkgCount)

	// drift summary line via the live reader.
	keep := readKeepList(cfg)
	actual, c := readActualState(cfg, "error", "etc", keep, nowRFC3339(), stderr)
	if c != ExitOK {
		return c
	}
	report := diff.ComputeDrift(actual, rec, diff.KeepList(keep))
	if report.Empty() {
		fmt.Fprintln(stdout, "clean")
	} else {
		fmt.Fprintf(stdout, "%d drift item(s)\n", report.Count())
	}
	return ExitOK
}
