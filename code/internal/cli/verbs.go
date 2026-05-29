// generated from spec: zypper-declarative.spec.md sha256:714e75ff672557d2c7344736a5f36b52afa37e0f565b07de83e1b18cc4492014
//
// The five CLI verbs: apply, diff, verify, status, describe. Each orchestrates
// the internal behaviours and maps their returned errors to exit codes per the
// spec ExitCode type and the cli-tool template.
package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mge1512/zypper-declarative/internal/converge"
	"github.com/mge1512/zypper-declarative/internal/diag"
	"github.com/mge1512/zypper-declarative/internal/diff"
	"github.com/mge1512/zypper-declarative/internal/manifest"
	"github.com/mge1512/zypper-declarative/internal/record"
	"github.com/mge1512/zypper-declarative/internal/state"
	"github.com/mge1512/zypper-declarative/internal/txn"
)

// emit writes a Diagnostic to stderr and returns the matching exit code.
func (a *App) emit(d *diag.Diagnostic) int {
	fmt.Fprintln(a.Stderr, d.Error())
	switch d.Domain {
	case diag.DomainInvocation, diag.DomainTransaction:
		return ExitInvocation
	default:
		return ExitLogical
	}
}

// cmdApply implements BEHAVIOR: apply STEPS 1–11.
func (a *App) cmdApply(cfg Config) int {
	// STEP 1 — load desired manifest.
	res, d := manifest.Load(cfg.ManifestPath, manifest.LoadOptions{
		ExplicitFormat:  cfg.Format,
		DefaultFormat:   cfg.ManifestFormat,
		VerifySignature: cfg.SignatureVerification && cfg.Keyring != "",
		Keyring:         cfg.Keyring,
	})
	if d != nil {
		return a.emit(d)
	}
	desired := res.Manifest

	// STEP 2 — load applied record (absence => empty).
	applied, _, d := record.Load(cfg.AppliedRoot)
	if d != nil {
		return a.emit(d)
	}

	// STEP 3 — compute intent diff.
	intent := diff.ComputeIntentDiff(&desired, &applied)

	keepList, keepSet := a.loadKeepList(cfg.KeepListPath)

	// STEP 4 — if intent empty, check drift; if also empty, nothing to do.
	if intent.Empty() {
		actual, derr := a.actualState(cfg, "/", keepList)
		if derr != nil {
			if dd, ok := derr.(*diag.Diagnostic); ok {
				return a.emit(dd)
			}
			return a.emit(diag.New(diag.DomainFiles, "%v", derr))
		}
		drift := diff.ComputeDrift(&actual, &applied, keepSet)
		if drift.Empty() {
			fmt.Fprintln(a.Stdout, "nothing to do")
			return ExitOK
		}
	}

	// STEP 5 — acquire transaction context.
	acq := txn.NewAcquirer(a.Runner)
	ctx, d := acq.Acquire(cfg.TransactionMode)
	if d != nil {
		return a.emit(d) // transaction domain -> exit 2
	}

	conv := &converge.Converger{
		Runner:       a.Runner,
		ContentStore: cfg.ContentStore,
		RepoLock:     cfg.RepoLock,
		KeepList:     keepSet,
	}

	// STEP 6 — repositories + packages.
	resolved, d := conv.Packages(ctx, intent)
	if d != nil {
		return a.emit(d) // discard implied; packages domain -> exit 1
	}

	// STEP 7 — files.
	if d := conv.Files(ctx, intent); d != nil {
		return a.emit(d)
	}

	// STEP 8 — units.
	if d := conv.Units(ctx, intent); d != nil {
		return a.emit(d)
	}

	// STEP 9 — write applied record (resolved packages) into the context.
	if d := record.Write(ctx.Root, &desired, res.DesiredSHA256, resolved); d != nil {
		return a.emit(d)
	}

	// STEP 10 — verify the converged tree (post-converge drift check).
	newApplied, _, d := record.Load(ctx.Root)
	if d != nil {
		return a.emit(d)
	}
	actual, derr := a.actualState(cfg, ctx.Root, keepList)
	if derr != nil {
		if dd, ok := derr.(*diag.Diagnostic); ok {
			return a.emit(dd)
		}
		return a.emit(diag.New(diag.DomainFiles, "%v", derr))
	}
	postDrift := diff.ComputeDrift(&actual, &newApplied, keepSet)
	if !postDrift.Empty() {
		return a.emit(diag.New(diag.DomainFiles, "post-converge verification found drift (%d item(s))", postDrift.Count()))
	}

	// STEP 11 — seal and activate (best-effort, delegated when not opened here).
	if ctx.OpenedHere {
		a.seal(ctx, cfg)
	}
	a.printIntentSummary(intent)
	return ExitOK
}

// cmdDiff implements BEHAVIOR: diff STEPS 1–5 (dry run, no transaction).
func (a *App) cmdDiff(cfg Config) int {
	res, d := manifest.Load(cfg.ManifestPath, manifest.LoadOptions{
		ExplicitFormat:  cfg.Format,
		DefaultFormat:   cfg.ManifestFormat,
		VerifySignature: cfg.SignatureVerification && cfg.Keyring != "",
		Keyring:         cfg.Keyring,
	})
	if d != nil {
		return a.emit(d)
	}
	desired := res.Manifest

	applied, _, d := record.Load(cfg.AppliedRoot)
	if d != nil {
		return a.emit(d)
	}

	intent := diff.ComputeIntentDiff(&desired, &applied)
	keepList, keepSet := a.loadKeepList(cfg.KeepListPath)

	actual, derr := a.actualState(cfg, "/", keepList)
	if derr != nil {
		// diff plan still printed against an empty actual on a soft read failure;
		// but a hard reader error is surfaced.
		if dd, ok := derr.(*diag.Diagnostic); ok {
			return a.emit(dd)
		}
	}
	drift := diff.ComputeDrift(&actual, &applied, keepSet)

	a.printPlan(intent, drift)
	return ExitOK
}

// cmdVerify implements BEHAVIOR: verify STEPS 1–4.
func (a *App) cmdVerify(cfg Config) int {
	applied, present, d := record.Load(cfg.AppliedRoot)
	if d != nil {
		return a.emit(d)
	}
	if !present {
		fmt.Fprintln(a.Stderr, diag.New(diag.DomainInvocation, "no declaration applied").Error())
		return ExitInvocation
	}

	keepList, keepSet := a.loadKeepList(cfg.KeepListPath)

	var actual manifest.Manifest
	if cfg.StatePath != "" {
		m, dd := manifest.LoadDump(cfg.StatePath, cfg.Format, cfg.ManifestFormat)
		if dd != nil {
			return a.emit(dd) // invocation -> exit 2
		}
		actual = m
	} else {
		m, dd := state.Describe("/", a.Runner, keepList)
		if dd != nil {
			return a.emit(dd)
		}
		actual = m
	}

	drift := diff.ComputeDrift(&actual, &applied, keepSet)
	if drift.Empty() {
		fmt.Fprintln(a.Stdout, "system matches declaration")
		return ExitOK
	}
	a.emitDrift(drift)
	return ExitLogical
}

// cmdStatus implements BEHAVIOR: status STEPS 1–4.
func (a *App) cmdStatus(cfg Config, rest []string) int {
	// STEP 1 — reject any unrecognised argument (already parsed; but reject
	// bare words / unknown flags that survived parsing as errors).
	for _, arg := range rest {
		stripped := strings.TrimPrefix(arg, "--")
		if !strings.Contains(stripped, "=") {
			fmt.Fprintf(a.Stderr, "Error [invocation] unrecognised argument %q\n", arg)
			a.printUsage(a.Stderr)
			return ExitInvocation
		}
	}

	applied, present, d := record.Load(cfg.AppliedRoot)
	if d != nil {
		return a.emit(d)
	}
	if !present {
		fmt.Fprintln(a.Stdout, "no declaration applied")
		return ExitOK
	}

	// STEP 3 — print record summary.
	gen := a.generationID(cfg.AppliedRoot)
	pkgCount := 0
	if applied.Packages != nil {
		pkgCount = len(applied.Packages.Elements)
	}
	fmt.Fprintf(a.Stdout, "desired_sha256: %s\n", applied.Meta.DesiredSHA256)
	fmt.Fprintf(a.Stdout, "format_version: %d\n", applied.Meta.FormatVersion)
	fmt.Fprintf(a.Stdout, "generation: %s\n", gen)
	fmt.Fprintf(a.Stdout, "created_at: %s\n", applied.Meta.CreatedAt)
	fmt.Fprintf(a.Stdout, "resolved packages: %d\n", pkgCount)

	// STEP 4 — drift summary line.
	keepList, keepSet := a.loadKeepList(cfg.KeepListPath)
	actual, derr := a.actualState(cfg, "/", keepList)
	if derr != nil {
		fmt.Fprintln(a.Stdout, "drift: unavailable")
		return ExitOK
	}
	drift := diff.ComputeDrift(&actual, &applied, keepSet)
	if drift.Empty() {
		fmt.Fprintln(a.Stdout, "drift: clean")
	} else {
		fmt.Fprintf(a.Stdout, "drift: %d drift item(s)\n", drift.Count())
	}
	return ExitOK
}

// cmdDescribe implements BEHAVIOR: describe STEPS 1–4.
func (a *App) cmdDescribe(cfg Config) int {
	// STEP 1 — format already validated during parse; resolve against out.
	format, ferr := manifest.ResolveFormat(cfg.Format, cfg.Out, cfg.ManifestFormat)
	if ferr != nil {
		fmt.Fprintf(a.Stderr, "Error [invocation] %v\n", ferr)
		a.printUsage(a.Stderr)
		return ExitInvocation
	}

	keepList, _ := a.loadKeepList(cfg.KeepListPath)

	// STEP 2 — obtain actual state.
	m, d := state.Describe(cfg.Root, a.Runner, keepList)
	if d != nil {
		return a.emit(d) // state collection failure -> exit 1
	}

	// STEP 3 — serialise.
	var data []byte
	var err error
	if format == manifest.FormatYAML {
		data, err = manifest.MarshalYAML(&m)
	} else {
		data, err = manifest.MarshalJSON(&m)
	}
	if err != nil {
		fmt.Fprintf(a.Stderr, "Error [invocation] serialisation failed: %v\n", err)
		return ExitInvocation
	}

	// STEP 4 — write to out or stdout.
	if cfg.Out != "" {
		if werr := os.WriteFile(cfg.Out, append(data, '\n'), 0o644); werr != nil {
			fmt.Fprintf(a.Stderr, "Error [invocation] output path unwritable: %v\n", werr)
			return ExitInvocation
		}
		return ExitOK
	}
	a.Stdout.Write(data)
	if len(data) == 0 || data[len(data)-1] != '\n' {
		fmt.Fprintln(a.Stdout)
	}
	return ExitOK
}

// seal seals the snapshot read-only, marks it the default boot target, and
// schedules activation per the activation policy. Best-effort via the runner.
func (a *App) seal(ctx txn.Context, cfg Config) {
	switch cfg.ActivationPolicy {
	case "none":
		// no activation scheduled
	default:
		a.Runner.Run("snapper", "modify", "--read-only", ctx.Root)
	}
}

func (a *App) generationID(root string) string {
	stdout, _, err := a.Runner.Run("snapper", "--machine-readable", "csv", "list", "--columns", "number")
	if err == nil {
		if line := strings.TrimSpace(stdout); line != "" {
			parts := strings.Split(line, "\n")
			return parts[len(parts)-1]
		}
	}
	return filepath.Clean(rootName(root))
}

func rootName(root string) string {
	if root == "" || root == "/" {
		return "current"
	}
	return root
}
