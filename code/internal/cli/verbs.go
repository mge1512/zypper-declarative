// generated from spec: zypper-declarative.spec.md sha256:58e1636e2de82ab81a5cd3f81d6b3c9ac6a8976e18f9abb2bbd2b2aba56fe4d4
//
// Verb implementations: apply, diff, verify, status, describe. Each orchestrates
// the internal behaviours and maps results to exit codes and output streams.
package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mge1512/zypper-declarative/internal/config"
	"github.com/mge1512/zypper-declarative/internal/converge"
	"github.com/mge1512/zypper-declarative/internal/diag"
	"github.com/mge1512/zypper-declarative/internal/diff"
	"github.com/mge1512/zypper-declarative/internal/manifest"
	"github.com/mge1512/zypper-declarative/internal/record"
	"github.com/mge1512/zypper-declarative/internal/state"
	"github.com/mge1512/zypper-declarative/internal/system"
	"github.com/mge1512/zypper-declarative/internal/txn"
)

// keepListSet loads the keep-list file (one path per line) into a set. A missing
// file yields an empty set (the keep-list is optional).
func keepListSet(path string) map[string]bool {
	set := map[string]bool{}
	if path == "" {
		return set
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return set
	}
	for _, line := range splitLines(string(data)) {
		if line != "" {
			set[line] = true
		}
	}
	return set
}

func splitLines(s string) []string {
	var out []string
	cur := ""
	for _, r := range s {
		if r == '\n' || r == '\r' {
			out = append(out, trimSpace(cur))
			cur = ""
			continue
		}
		cur += string(r)
	}
	out = append(out, trimSpace(cur))
	return out
}

func trimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}

func (a *App) loadOpts(cfg config.Config) manifest.LoadOptions {
	return manifest.LoadOptions{
		ExplicitFormat:        cfg.ExplicitFormat,
		ExplicitFormatGiven:   cfg.ExplicitFormatGiven,
		DefaultFormat:         cfg.ManifestFormat,
		SignatureVerification: cfg.SignatureVerification,
		Keyring:               cfg.Keyring,
		Verifier:              manifest.NoopVerifier{},
	}
}

func (a *App) reader(cfg config.Config) *state.Reader {
	return &state.Reader{
		Runner:   &system.OSCommandRunner{},
		KeepList: keepListSet(cfg.KeepList),
	}
}

// ---------------------------------------------------------------------------
// diff
// ---------------------------------------------------------------------------

func (a *App) runDiff(cfg config.Config) int {
	m, _, derr := manifest.LoadDesiredManifest(cfg.ManifestPath, a.loadOpts(cfg))
	if derr != nil {
		a.emit(derr)
		return exitFor(derr)
	}
	applied, _, lerr := record.Load(cfg.AppliedRoot)
	if lerr != nil {
		a.emit(lerr)
		return exitFor(lerr)
	}
	id := diff.ComputeIntentDiff(m, applied)

	// Obtain actual state on "/" with on_unreadable=error (internal caller),
	// except a CLI on-unreadable=warn override is honoured to allow unprivileged
	// dry runs.
	res, serr := a.reader(cfg).Describe("/", stateOnUnreadable(cfg))
	if serr != nil {
		a.emit(serr)
		return exitFor(serr)
	}
	drift := diff.ComputeDrift(res.Manifest, applied, diff.DriftOptions{KeepList: keepListSet(cfg.KeepList)})

	a.printPlan(id, drift)
	a.emit(res.Diagnostics...)
	return ExitOK
}

func (a *App) printPlan(id diff.Diff, drift diff.DriftReport) {
	w := a.Stdout
	fmt.Fprintln(w, "packages to install:")
	for _, p := range id.PackagesInstall {
		fmt.Fprintf(w, "  %s\n", p.Name)
	}
	fmt.Fprintln(w, "packages to remove:")
	for _, p := range id.PackagesRemove {
		fmt.Fprintf(w, "  %s\n", p.Name)
	}
	fmt.Fprintln(w, "repositories to set:")
	for _, r := range id.ReposSet {
		fmt.Fprintf(w, "  %s\n", r.Alias)
	}
	fmt.Fprintln(w, "files to write:")
	for _, f := range id.FilesWrite {
		fmt.Fprintf(w, "  %s\n", f.Name)
	}
	fmt.Fprintln(w, "files to delete:")
	for _, p := range id.FilesDelete {
		fmt.Fprintf(w, "  %s\n", p)
	}
	fmt.Fprintln(w, "units to change:")
	for _, u := range id.UnitsChange {
		fmt.Fprintf(w, "  %s -> %s\n", u.Name, u.State)
	}
	fmt.Fprintln(w, "current drift:")
	if drift.Empty() {
		fmt.Fprintln(w, "  clean")
	} else {
		for _, p := range drift.FilesModified {
			fmt.Fprintf(w, "  modified file: %s\n", p)
		}
		for _, p := range drift.FilesExtra {
			fmt.Fprintf(w, "  extra file: %s\n", p)
		}
		for _, u := range drift.UnitsDivergent {
			fmt.Fprintf(w, "  divergent unit: %s\n", u.Name)
		}
		for _, p := range drift.PackagesDivergent {
			fmt.Fprintf(w, "  divergent package: %s\n", p.Name)
		}
	}
}

// stateOnUnreadable maps the CLI config to the state reader policy. Internal
// callers default to error; the CLI on-unreadable=warn override is permitted so
// read-only verbs can run unprivileged.
func stateOnUnreadable(cfg config.Config) state.OnUnreadable {
	if cfg.OnUnreadable == config.OnUnreadableWarn {
		return state.OnUnreadableWarn
	}
	return state.OnUnreadableError
}

// ---------------------------------------------------------------------------
// verify
// ---------------------------------------------------------------------------

func (a *App) runVerify(cfg config.Config) int {
	applied, present, lerr := record.Load(cfg.AppliedRoot)
	if lerr != nil {
		a.emit(lerr)
		return exitFor(lerr)
	}
	if !present {
		a.emit(diag.Errorf(diag.DomainInvocation, "no declaration applied"))
		return ExitInvocation
	}

	var actual manifest.Manifest
	if cfg.StatePath != "" {
		// Load and schema-validate the supplied dump under its resolved format.
		format := manifest.ResolveFormat(cfg.ExplicitFormat, cfg.ExplicitFormatGiven, cfg.StatePath, cfg.ManifestFormat)
		data, rerr := os.ReadFile(cfg.StatePath)
		if rerr != nil {
			a.emit(diag.Errorf(diag.DomainInvocation, "state dump unreadable: %v", rerr))
			return ExitInvocation
		}
		m, perr := manifest.Parse(data, format)
		if perr != nil {
			a.emit(diag.Errorf(diag.DomainInvocation, "state dump malformed: %v", perr))
			return ExitInvocation
		}
		if verr := manifest.Validate(m); verr != nil {
			a.emit(diag.Errorf(diag.DomainInvocation, "state dump malformed: %v", verr))
			return ExitInvocation
		}
		actual = m
	} else {
		res, serr := a.reader(cfg).Describe("/", stateOnUnreadable(cfg))
		if serr != nil {
			a.emit(serr)
			return exitFor(serr)
		}
		actual = res.Manifest
	}

	drift := diff.ComputeDrift(actual, applied, diff.DriftOptions{KeepList: keepListSet(cfg.KeepList)})
	if drift.Empty() {
		fmt.Fprintln(a.Stdout, "system matches declaration")
		return ExitOK
	}
	a.emitDrift(drift)
	return ExitError
}

func (a *App) emitDrift(drift diff.DriftReport) {
	for _, p := range drift.FilesModified {
		a.emit(diag.Errorf(diag.DomainFiles, "drift: modified file %s", p))
	}
	for _, p := range drift.FilesExtra {
		a.emit(diag.Errorf(diag.DomainFiles, "drift: extra file %s", p))
	}
	for _, u := range drift.UnitsDivergent {
		a.emit(diag.Errorf(diag.DomainUnits, "drift: divergent unit %s (declared %s)", u.Name, u.State))
	}
	for _, p := range drift.PackagesDivergent {
		a.emit(diag.Errorf(diag.DomainPackages, "drift: divergent package %s", p.Name))
	}
}

// ---------------------------------------------------------------------------
// status
// ---------------------------------------------------------------------------

func (a *App) runStatus(cfg config.Config) int {
	applied, present, lerr := record.Load(cfg.AppliedRoot)
	if lerr != nil {
		a.emit(lerr)
		return exitFor(lerr)
	}
	if !present {
		fmt.Fprintln(a.Stdout, "no declaration applied")
		return ExitOK
	}

	pkgCount := 0
	if applied.Packages != nil {
		pkgCount = len(applied.Packages.Elements)
	}
	fmt.Fprintf(a.Stdout, "desired_sha256: %s\n", applied.Meta.DesiredSHA256)
	fmt.Fprintf(a.Stdout, "format_version: %d\n", applied.Meta.FormatVersion)
	fmt.Fprintf(a.Stdout, "generation: %s\n", generationID(cfg.AppliedRoot))
	fmt.Fprintf(a.Stdout, "created_at: %s\n", applied.Meta.CreatedAt)
	fmt.Fprintf(a.Stdout, "packages: %d resolved\n", pkgCount)

	res, serr := a.reader(cfg).Describe("/", stateOnUnreadable(cfg))
	if serr != nil {
		// status remains exit 0 on valid invocation; report drift as unknown.
		a.emit(serr)
		fmt.Fprintln(a.Stdout, "drift: unknown (actual state unreadable)")
		return ExitOK
	}
	drift := diff.ComputeDrift(res.Manifest, applied, diff.DriftOptions{KeepList: keepListSet(cfg.KeepList)})
	if drift.Empty() {
		fmt.Fprintln(a.Stdout, "drift: clean")
	} else {
		fmt.Fprintf(a.Stdout, "drift: %d drift item(s)\n", drift.Count())
	}
	a.emit(res.Diagnostics...)
	return ExitOK
}

func generationID(root string) string {
	if root == "" || root == "/" {
		return "running-system"
	}
	return filepath.Clean(root)
}

// ---------------------------------------------------------------------------
// describe
// ---------------------------------------------------------------------------

func (a *App) runDescribe(cfg config.Config) int {
	res, serr := a.reader(cfg).Describe(cfg.Root, describeOnUnreadable(cfg))
	if serr != nil {
		a.emit(serr)
		return exitFor(serr) // domain packages/repositories/units/files -> exit 1
	}
	a.emit(res.Diagnostics...)

	// Resolve the output format (explicit wins, else out extension, else default).
	format := manifest.ResolveFormat(cfg.ExplicitFormat, cfg.ExplicitFormatGiven, cfg.Out, cfg.ManifestFormat)
	data, merr := manifest.Marshal(res.Manifest, format)
	if merr != nil {
		a.emit(diag.Errorf(diag.DomainInvocation, "serialise failed: %v", merr))
		return ExitInvocation
	}

	if cfg.Out == "" {
		_, _ = a.Stdout.Write(data)
		return ExitOK
	}
	if werr := os.WriteFile(cfg.Out, data, 0644); werr != nil {
		a.emit(diag.Errorf(diag.DomainInvocation, "output path unwritable: %v", werr))
		return ExitInvocation
	}
	return ExitOK
}

func describeOnUnreadable(cfg config.Config) state.OnUnreadable {
	if cfg.OnUnreadable == config.OnUnreadableWarn {
		return state.OnUnreadableWarn
	}
	return state.OnUnreadableError
}

// ---------------------------------------------------------------------------
// apply
// ---------------------------------------------------------------------------

func (a *App) runApply(cfg config.Config) int {
	// 1. load desired manifest.
	desired, desiredSHA, derr := manifest.LoadDesiredManifest(cfg.ManifestPath, a.loadOpts(cfg))
	if derr != nil {
		a.emit(derr)
		return exitFor(derr)
	}

	// 2. load applied record.
	applied, _, lerr := record.Load(cfg.AppliedRoot)
	if lerr != nil {
		a.emit(lerr)
		return exitFor(lerr)
	}

	// 3. compute intent diff.
	id := diff.ComputeIntentDiff(desired, applied)

	keep := keepListSet(cfg.KeepList)

	// 4. if empty intent diff, check drift; if also empty, nothing to do.
	if id.Empty() {
		res, serr := a.reader(cfg).Describe("/", stateOnUnreadable(cfg))
		if serr != nil {
			a.emit(serr)
			return exitFor(serr)
		}
		drift := diff.ComputeDrift(res.Manifest, applied, diff.DriftOptions{KeepList: keep})
		if drift.Empty() {
			fmt.Fprintln(a.Stdout, "nothing to do")
			return ExitOK
		}
	}

	// 5. acquire transaction context.
	acq := a.acquirer(cfg)
	ctx, terr := acq.Acquire(txn.Mode(cfg.TransactionMode))
	if terr != nil {
		a.emit(terr)
		return exitFor(terr) // transaction -> exit 2
	}

	cv := &converge.Converger{
		Runner:       &system.OSCommandRunner{},
		Reader:       a.reader(cfg),
		ContentStore: cfg.ContentStore,
		KeepList:     keep,
		RepoLock:     cfg.RepoLock,
	}

	// 6. converge repositories + packages; capture the resolved lock.
	resolved, perr := cv.Packages(ctx, id)
	if perr != nil {
		a.emit(perr)
		return ExitError // discarded
	}

	// 7. converge files.
	if ferr := cv.Files(ctx, id); ferr != nil {
		a.emit(ferr)
		return ExitError
	}

	// 8. converge units.
	if uerr := cv.Units(ctx, id); uerr != nil {
		a.emit(uerr)
		return ExitError
	}

	// 9. write applied record.
	if werr := record.Write(ctx.Root, desired, desiredSHA, resolved, record.NoopStamper{}); werr != nil {
		a.emit(werr)
		return ExitError
	}

	// 10. post-converge verification.
	postRes, pverr := a.reader(cfg).Describe(ctx.Root, state.OnUnreadableError)
	if pverr != nil {
		a.emit(pverr)
		return ExitError
	}
	newApplied := manifest.Manifest{
		Meta:         manifest.ManifestMeta{FormatVersion: 1, DesiredSHA256: desiredSHA},
		Packages:     resolved,
		Repositories: desired.Repositories,
		Services:     desired.Services,
		ConfigFiles:  desired.ConfigFiles,
	}
	postDrift := diff.ComputeDrift(postRes.Manifest, newApplied, diff.DriftOptions{KeepList: keep})
	if !postDrift.Empty() {
		a.emit(diag.Errorf(diag.DomainFiles, "post-converge verification found drift; transaction discarded"))
		return ExitError
	}

	// 11. seal and activate (delegated/abstract); emit summary.
	fmt.Fprintf(a.Stdout, "converged: %d package(s), %d file(s) written, %d unit(s) changed\n",
		len(id.PackagesInstall), len(id.FilesWrite), len(id.UnitsChange))
	return ExitOK
}

// acquirer returns the production transaction acquirer. The detection and
// open/seal mechanisms are abstract bindings; in this build they report no
// external transaction and no internal machinery, so apply on a host without a
// transaction mechanism fails with a transaction diagnostic (exit 2), which is
// the specified behaviour for an unavailable mechanism.
func (a *App) acquirer(cfg config.Config) txn.Acquirer {
	return &txn.SystemAcquirer{
		Runner:            &system.OSCommandRunner{},
		InsideTransaction: func() bool { return false },
		ExternalRoot:      func() string { return "" },
		OpenInternal:      nil, // no internal machinery wired into this build
	}
}
