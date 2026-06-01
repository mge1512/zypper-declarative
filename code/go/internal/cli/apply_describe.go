// generated from spec: zypper-declarative.spec.md sha256:27aee8e374eb3507189bad0b78339109d0116e6a13b55ae4df2ba9a18e769fc4
//
// Verb handlers: describe and apply.
package cli

import (
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/mge1512/zypper-declarative/internal/converge"
	"github.com/mge1512/zypper-declarative/internal/diff"
	"github.com/mge1512/zypper-declarative/internal/manifest"
	"github.com/mge1512/zypper-declarative/internal/record"
	"github.com/mge1512/zypper-declarative/internal/state"
	"github.com/mge1512/zypper-declarative/internal/txn"
)

// runDescribe implements BEHAVIOR: describe.
func runDescribe(args []string, stdout, stderr io.Writer) int {
	cfg := defaultConfig()
	if _, err := parseArgs(&cfg, args); err != nil {
		printUsage(stderr)
		return ExitInvocation
	}

	// 1. resolve output format early so an unknown format value is exit 2.
	//    (parseArgs already rejects an unknown format= value; here we resolve
	//    against out= per resolve-format.)
	outFormat, ferr := manifest.ResolveFormat(cfg.Format, cfg.Out, cfg.manifestFormatDefault())
	if ferr != nil {
		printUsage(stderr)
		return ExitInvocation
	}

	// 2. obtain actual state
	keep := readKeepList(cfg)
	actual, c := readActualState(cfg, cfg.OnUnreadable, cfg.Scope, keep, time.Now().UTC().Format(time.RFC3339), stderr)
	if c != ExitOK {
		return c
	}

	// 3+4. serialise in the resolved format
	data, merr := manifest.Marshal(actual, outFormat)
	if merr != nil {
		emitDiag(stderr, manifest.NewError(manifest.DomainInvocation, "cannot serialise document: "+merr.Error()))
		return ExitInvocation
	}

	// 5. write to out or stdout
	if cfg.Out != "" {
		if mkerr := os.MkdirAll(filepath.Dir(cfg.Out), 0755); mkerr != nil {
			emitDiag(stderr, manifest.NewError(manifest.DomainInvocation, "output path unwritable: "+mkerr.Error()))
			return ExitInvocation
		}
		if werr := os.WriteFile(cfg.Out, data, 0644); werr != nil {
			emitDiag(stderr, manifest.NewError(manifest.DomainInvocation, "output path unwritable: "+werr.Error()))
			return ExitInvocation
		}
		return ExitOK
	}
	if _, werr := stdout.Write(data); werr != nil {
		emitDiag(stderr, manifest.NewError(manifest.DomainInvocation, "cannot write to stdout: "+werr.Error()))
		return ExitInvocation
	}
	return ExitOK
}

// runApply implements BEHAVIOR: apply.
func runApply(args []string, stdout, stderr io.Writer) int {
	cfg := defaultConfig()
	if _, err := parseArgs(&cfg, args); err != nil {
		printUsage(stderr)
		return ExitInvocation
	}

	// 1. load desired manifest
	desired, desiredSHA, code := loadDesiredManifest(cfg, cfg.ManifestPath, stderr)
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
	keep := readKeepList(cfg)

	// 4. if intent diff empty, check drift; if also empty, nothing to do.
	if d.Empty() {
		actual, c := readActualState(cfg, "error", "etc", keep, nowRFC3339(), stderr)
		if c != ExitOK {
			return c
		}
		report := diff.ComputeDrift(actual, applied, diff.KeepList(keep))
		if report.Empty() {
			io.WriteString(stdout, "nothing to do\n")
			return ExitOK
		}
	}

	// 5. acquire transaction context
	acq := &txn.DefaultAcquirer{}
	ctx, td := acq.Acquire(txn.Mode(cfg.TransactionMode))
	if td != nil {
		emitDiag(stderr, *td)
		return ExitInvocation
	}

	conv := &converge.Converger{Runner: &state.OSCommandRunner{}, ContentStore: cfg.ContentStore, KeepList: keep}

	// 6. repositories + packages
	resolved, pd := conv.Packages(ctx, d)
	if pd != nil {
		emitDiag(stderr, *pd)
		return ExitError
	}
	// 7. files
	if fd := conv.Files(ctx, d); fd != nil {
		emitDiag(stderr, *fd)
		return ExitError
	}
	// 8. units
	if ud := conv.Units(ctx, d); ud != nil {
		emitDiag(stderr, *ud)
		return ExitError
	}
	// 9. write applied record
	if wd := record.Write(ctx.Root, desired, desiredSHA, resolved, nowRFC3339()); wd != nil {
		emitDiag(stderr, *wd)
		return ExitError
	}
	// 10. post-converge verification
	verifyCfg := cfg
	verifyCfg.Root = ctx.Root
	actual, c := readActualState(verifyCfg, "error", "etc", keep, nowRFC3339(), stderr)
	if c != ExitOK {
		return c
	}
	newApplied, _, _ := record.Load(ctx.Root)
	report := diff.ComputeDrift(actual, newApplied, diff.KeepList(keep))
	if !report.Empty() {
		emitDriftDiagnostics(stderr, report)
		return ExitError
	}

	// 11. seal/activate is handled by the external/internal binding; emit summary.
	io.WriteString(stdout, "applied: "+desiredSHA+"\n")
	return ExitOK
}
