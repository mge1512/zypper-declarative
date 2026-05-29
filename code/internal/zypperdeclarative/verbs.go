// generated from spec: zypper-declarative.spec.md sha256:714e75ff672557d2c7344736a5f36b52afa37e0f565b07de83e1b18cc4492014
//
// The five CLI verbs (apply, diff, verify, status, describe). Each returns an
// ExitCode and writes diagnostics to stderr / output to stdout via the
// provided io.Writers. Exit-code mapping lives only in the verbs.
package zypperdeclarative

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Verbs holds the runtime context for executing a verb.
type Verbs struct {
	Providers *Providers
	Stdout    io.Writer
	Stderr    io.Writer
}

func (v *Verbs) emit(d *Diagnostic) {
	fmt.Fprintf(v.Stderr, "%s: [%s] %s\n", d.Severity, d.Domain, d.Message)
}

// Apply converges the system to the desired manifest. BEHAVIOR: apply.
func (v *Verbs) Apply(cfg *Config) int {
	keep := LoadKeepList(cfg.KeepListPath)

	// Step 1: load desired.
	desired, desiredSHA, diag := LoadDesiredManifest(cfg, cfg.ManifestPath)
	if diag != nil {
		v.emit(diag)
		return exitForDomain(diag.Domain)
	}

	// Step 2: load applied.
	applied, _, diag := LoadAppliedRecord(cfg.AppliedRoot)
	if diag != nil {
		v.emit(diag)
		return exitForDomain(diag.Domain)
	}

	// Step 3: intent diff.
	diff := ComputeIntentDiff(desired, applied)

	// Step 4: if empty, compute drift; if also empty, nothing to do.
	if diff.IsEmpty() {
		actual, d := DescribeActualState(v.Providers, cfg.Root, keep)
		if d != nil {
			v.emit(d)
			return ExitLogical
		}
		drift := ComputeDrift(actual, applied, keep)
		if drift.IsEmpty() {
			fmt.Fprintln(v.Stdout, "nothing to do")
			return ExitOK
		}
	}

	// Step 5: acquire transaction.
	ctx, diag := AcquireTransactionContext(v.Providers, cfg.TransactionMode)
	if diag != nil {
		v.emit(diag)
		return ExitInvocation // transaction mechanism unavailable -> exit 2
	}

	// Step 6: repositories + packages.
	resolved, diag := ConvergePackages(v.Providers, ctx, diff, cfg.RepoLock)
	if diag != nil {
		v.emit(diag)
		return ExitLogical
	}

	// Step 7: files.
	if diag := ConvergeFiles(v.Providers, ctx, diff, cfg, keep); diag != nil {
		v.emit(diag)
		return ExitLogical
	}

	// Step 8: units.
	if diag := ConvergeUnits(v.Providers, ctx, diff); diag != nil {
		v.emit(diag)
		return ExitLogical
	}

	// Step 9: write applied record.
	if diag := WriteAppliedRecord(v.Providers, ctx, desired, desiredSHA, resolved); diag != nil {
		v.emit(diag)
		return ExitLogical
	}

	// Step 10: post-converge verification.
	postActual, d := DescribeActualState(v.Providers, ctx.Root, keep)
	if d != nil {
		v.emit(d)
		return ExitLogical
	}
	newApplied := &Manifest{
		Meta:         ManifestMeta{FormatVersion: 1, DesiredSHA256: desiredSHA},
		Packages:     resolved,
		Repositories: desired.Repositories,
		Services:     desired.Services,
		ConfigFiles:  desired.ConfigFiles,
	}
	postDrift := ComputeDrift(postActual, newApplied, keep)
	if !postDrift.IsEmpty() {
		v.emit(newError(DomainFiles, "post-converge verification found drift"))
		return ExitLogical
	}

	// Step 11: seal and activate.
	if ctx.OpenedHere {
		if d := v.Providers.Snapshot.Seal(ctx.Root, cfg.ActivationPolicy); d != nil {
			v.emit(d)
			return ExitLogical
		}
	}
	fmt.Fprintf(v.Stdout, "applied %s\n", desiredSHA)
	return ExitOK
}

// Diff prints what apply would change. BEHAVIOR: diff.
func (v *Verbs) Diff(cfg *Config) int {
	keep := LoadKeepList(cfg.KeepListPath)

	desired, desiredSHA, diag := LoadDesiredManifest(cfg, cfg.ManifestPath)
	if diag != nil {
		v.emit(diag)
		return exitForDomain(diag.Domain)
	}
	applied, _, diag := LoadAppliedRecord(cfg.AppliedRoot)
	if diag != nil {
		v.emit(diag)
		return exitForDomain(diag.Domain)
	}
	diff := ComputeIntentDiff(desired, applied)
	actual, d := DescribeActualState(v.Providers, cfg.Root, keep)
	if d != nil {
		v.emit(d)
		return ExitLogical
	}
	drift := ComputeDrift(actual, applied, keep)
	v.printPlan(desiredSHA, diff, drift)
	return ExitOK
}

func (v *Verbs) printPlan(desiredSHA string, diff *Diff, drift *DriftReport) {
	out := v.Stdout
	fmt.Fprintf(out, "desired_sha256: %s\n", desiredSHA)
	fmt.Fprintln(out, "packages to install:")
	for _, p := range diff.PackagesInstall {
		fmt.Fprintf(out, "  + %s\n", p.Name)
	}
	fmt.Fprintln(out, "packages to remove:")
	for _, p := range diff.PackagesRemove {
		fmt.Fprintf(out, "  - %s %s-%s.%s\n", p.Name, p.Version, p.Release, p.Arch)
	}
	fmt.Fprintln(out, "repositories to set:")
	for _, r := range diff.ReposSet {
		fmt.Fprintf(out, "  = %s %s\n", r.Alias, r.URL)
	}
	fmt.Fprintln(out, "files to write:")
	for _, f := range diff.FilesWrite {
		fmt.Fprintf(out, "  > %s\n", f.Name)
	}
	fmt.Fprintln(out, "files to delete:")
	for _, p := range diff.FilesDelete {
		fmt.Fprintf(out, "  x %s\n", p)
	}
	fmt.Fprintln(out, "units to change:")
	for _, u := range diff.UnitsChange {
		fmt.Fprintf(out, "  ~ %s -> %s\n", u.Name, u.State)
	}
	fmt.Fprintln(out, "drift:")
	for _, p := range drift.FilesModified {
		fmt.Fprintf(out, "  modified %s\n", p)
	}
	for _, p := range drift.FilesExtra {
		fmt.Fprintf(out, "  extra %s\n", p)
	}
	for _, u := range drift.UnitsDivergent {
		fmt.Fprintf(out, "  unit %s\n", u.Name)
	}
	for _, p := range drift.PackagesDivergent {
		fmt.Fprintf(out, "  package %s\n", p.Name)
	}
}

// Verify checks the actual state against the applied record. BEHAVIOR: verify.
func (v *Verbs) Verify(cfg *Config) int {
	keep := LoadKeepList(cfg.KeepListPath)

	applied, present, diag := LoadAppliedRecord(cfg.AppliedRoot)
	if diag != nil {
		v.emit(diag)
		return exitForDomain(diag.Domain)
	}
	if !present {
		v.emit(newError(DomainInvocation, "no declaration applied"))
		return ExitInvocation
	}

	var actual *Manifest
	if cfg.StatePath != "" {
		m, d := loadStateDump(cfg.StatePath)
		if d != nil {
			v.emit(d)
			return ExitInvocation
		}
		actual = m
	} else {
		m, d := DescribeActualState(v.Providers, cfg.Root, keep)
		if d != nil {
			v.emit(d)
			return ExitLogical
		}
		actual = m
	}

	drift := ComputeDrift(actual, applied, keep)
	if drift.IsEmpty() {
		fmt.Fprintln(v.Stdout, "system matches declaration")
		return ExitOK
	}
	for _, p := range drift.FilesModified {
		v.emit(newError(DomainFiles, "drift: file modified "+p))
	}
	for _, p := range drift.FilesExtra {
		v.emit(newError(DomainFiles, "drift: extra file "+p))
	}
	for _, u := range drift.UnitsDivergent {
		v.emit(newError(DomainUnits, "drift: unit "+u.Name+" diverges (declared "+u.State+")"))
	}
	for _, p := range drift.PackagesDivergent {
		v.emit(newError(DomainPackages, "drift: package "+p.Name+" diverges"))
	}
	return ExitLogical
}

// loadStateDump loads and schema-validates a supplied state dump as a Manifest.
func loadStateDump(path string) (*Manifest, *Diagnostic) {
	cfg := defaultConfig()
	cfg.SignatureVerification = false // a dump is actual-state, never signed
	// A JSON dump is valid YAML 1.2; choose by extension, defaulting to JSON.
	m, _, diag := LoadDesiredManifest(&cfg, path)
	if diag != nil {
		// Any read/parse/schema failure of a supplied dump is an invocation error.
		return nil, newError(DomainInvocation, "malformed state dump: "+diag.Message)
	}
	return m, nil
}

// Status prints the current declarative state. BEHAVIOR: status.
func (v *Verbs) Status(cfg *Config, extraArgs []string) int {
	// Step 1: reject unrecognised arguments.
	if len(extraArgs) > 0 {
		fmt.Fprintf(v.Stderr, "Error: [%s] unrecognised argument: %s\n", DomainInvocation, strings.Join(extraArgs, " "))
		fmt.Fprintln(v.Stderr, usageText())
		return ExitInvocation
	}

	keep := LoadKeepList(cfg.KeepListPath)
	applied, present, diag := LoadAppliedRecord(cfg.AppliedRoot)
	if diag != nil {
		v.emit(diag)
		return exitForDomain(diag.Domain)
	}
	if !present {
		fmt.Fprintln(v.Stdout, "no declaration applied")
		return ExitOK
	}

	pkgCount := 0
	if applied.Packages != nil {
		pkgCount = len(applied.Packages.Elements)
	}
	fmt.Fprintf(v.Stdout, "desired_sha256: %s\n", applied.Meta.DesiredSHA256)
	fmt.Fprintf(v.Stdout, "format_version: %d\n", applied.Meta.FormatVersion)
	fmt.Fprintf(v.Stdout, "generation: %s\n", generationID(applied))
	fmt.Fprintf(v.Stdout, "created_at: %s\n", applied.Meta.CreatedAt)
	fmt.Fprintf(v.Stdout, "resolved packages: %s\n", strconv.Itoa(pkgCount))

	actual, d := DescribeActualState(v.Providers, cfg.Root, keep)
	if d != nil {
		// status is read-only and must not fail invocation; report clean-unknown.
		fmt.Fprintln(v.Stdout, "drift: unknown (state unavailable)")
		return ExitOK
	}
	drift := ComputeDrift(actual, applied, keep)
	n := len(drift.FilesModified) + len(drift.FilesExtra) + len(drift.UnitsDivergent) + len(drift.PackagesDivergent)
	if n == 0 {
		fmt.Fprintln(v.Stdout, "drift: clean")
	} else {
		fmt.Fprintf(v.Stdout, "drift: %d drift item(s)\n", n)
	}
	return ExitOK
}

// generationID derives a stable generation identifier from the applied record.
func generationID(applied *Manifest) string {
	if applied.Meta.DesiredSHA256 != "" {
		return "manifest=" + applied.Meta.DesiredSHA256[:minInt(12, len(applied.Meta.DesiredSHA256))]
	}
	return "current"
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Describe reads the actual state and emits it in the selected format.
// BEHAVIOR: describe.
func (v *Verbs) Describe(cfg *Config, extraArgs []string) int {
	// Step 1: reject unrecognised argument / unknown format value.
	if len(extraArgs) > 0 {
		fmt.Fprintf(v.Stderr, "Error: [%s] unrecognised argument: %s\n", DomainInvocation, strings.Join(extraArgs, " "))
		fmt.Fprintln(v.Stderr, usageText())
		return ExitInvocation
	}
	format := cfg.ManifestFormat
	if cfg.FormatSet {
		format = cfg.Format
	}
	switch format {
	case FormatJSON, FormatYAML:
	default:
		fmt.Fprintf(v.Stderr, "Error: [%s] unknown format value: %s\n", DomainInvocation, string(format))
		fmt.Fprintln(v.Stderr, usageText())
		return ExitInvocation
	}

	keep := LoadKeepList(cfg.KeepListPath)

	// Step 2: actual state.
	m, d := DescribeActualState(v.Providers, cfg.Root, keep)
	if d != nil {
		v.emit(d)
		return ExitLogical
	}

	// Step 3: serialise.
	var data []byte
	var err error
	if format == FormatYAML {
		data, err = MarshalYAML(m)
	} else {
		data, err = MarshalCanonicalJSON(m)
		data = append(data, '\n')
	}
	if err != nil {
		v.emit(newError(DomainInvocation, "serialisation failed: "+err.Error()))
		return ExitLogical
	}

	// Step 4: write.
	if cfg.Out != "" {
		if werr := writeFileStrict(cfg.Out, data); werr != nil {
			v.emit(newError(DomainInvocation, "output path unwritable: "+werr.Error()))
			return ExitInvocation
		}
	} else {
		v.Stdout.Write(data)
	}
	return ExitOK
}
