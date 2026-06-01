// generated from spec: zypper-declarative.spec.md sha256:27aee8e374eb3507189bad0b78339109d0116e6a13b55ae4df2ba9a18e769fc4
//
// Shared helpers for the verb handlers: loading the desired manifest, loading a
// state dump, reading the keep-list, obtaining actual state through the single
// live reader, and emitting diagnostics.
package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mge1512/zypper-declarative/internal/manifest"
	"github.com/mge1512/zypper-declarative/internal/state"
)

// emitDiag writes one diagnostic per line to stderr.
func emitDiag(stderr io.Writer, d manifest.Diagnostic) {
	fmt.Fprintf(stderr, "%s: %s: %s\n", d.Severity, d.Domain, d.Message)
}

// loadDesiredManifest reads and validates the desired manifest. It returns the
// manifest, its desired_sha256, and an exit code (0 on success). On failure it
// emits the diagnostic and returns the mapped exit code.
func loadDesiredManifest(cfg Config, path string, stderr io.Writer) (*manifest.Manifest, string, int) {
	data, rerr := os.ReadFile(path)
	if rerr != nil {
		emitDiag(stderr, manifest.NewError(manifest.DomainInvocation, "manifest unreadable: "+rerr.Error()))
		return nil, "", ExitInvocation
	}
	f, ferr := manifest.ResolveFormat(cfg.Format, path, cfg.manifestFormatDefault())
	if ferr != nil {
		emitDiag(stderr, manifest.NewError(manifest.DomainInvocation, ferr.Error()))
		return nil, "", ExitInvocation
	}
	m, perr := manifest.Parse(data, manifest.ParseOptions{Format: f, AllowObservational: false})
	if perr != nil {
		if pe, ok := perr.(*manifest.ParseError); ok {
			if pe.Kind == manifest.ErrInvocation {
				emitDiag(stderr, manifest.NewError(manifest.DomainInvocation, pe.Message))
				return nil, "", ExitInvocation
			}
			emitDiag(stderr, manifest.NewError(manifest.DomainManifest, pe.Message))
			return nil, "", ExitError
		}
		emitDiag(stderr, manifest.NewError(manifest.DomainManifest, perr.Error()))
		return nil, "", ExitError
	}
	// signature verification, when enabled, is host-specific; default off.
	if cfg.SignatureVerification == "on" {
		emitDiag(stderr, manifest.NewError(manifest.DomainManifest, "signature verification enabled but no keyring binding available in this build"))
		return nil, "", ExitError
	}
	sum, herr := manifest.DesiredSHA256(m)
	if herr != nil {
		emitDiag(stderr, manifest.NewError(manifest.DomainManifest, "cannot hash manifest: "+herr.Error()))
		return nil, "", ExitError
	}
	return m, sum, ExitOK
}

// loadStateDump reads and schema-validates a captured actual-state dump (offline,
// no live read). Observational scopes are tolerated. Returns the manifest and an
// exit code; a malformed dump is an invocation error (exit 2).
func loadStateDump(cfg Config, path string, stderr io.Writer) (*manifest.Manifest, int) {
	data, rerr := os.ReadFile(path)
	if rerr != nil {
		emitDiag(stderr, manifest.NewError(manifest.DomainInvocation, "state dump unreadable: "+rerr.Error()))
		return nil, ExitInvocation
	}
	f, ferr := manifest.ResolveFormat(cfg.Format, path, cfg.manifestFormatDefault())
	if ferr != nil {
		emitDiag(stderr, manifest.NewError(manifest.DomainInvocation, ferr.Error()))
		return nil, ExitInvocation
	}
	m, perr := manifest.Parse(data, manifest.ParseOptions{Format: f, AllowObservational: true})
	if perr != nil {
		// A malformed state dump is an invocation error regardless of kind.
		emitDiag(stderr, manifest.NewError(manifest.DomainInvocation, "malformed state dump: "+perr.Error()))
		return nil, ExitInvocation
	}
	return m, ExitOK
}

// readKeepList loads the keep-list allowlist (one path per line) if configured.
func readKeepList(cfg Config) map[string]bool {
	keep := map[string]bool{}
	if cfg.KeepList == "" {
		return keep
	}
	f, err := os.Open(cfg.KeepList)
	if err != nil {
		return keep
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		keep[line] = true
	}
	return keep
}

// readActualState obtains actual state through the single live reader. Internal
// callers pass on_unreadable=error and scope=etc; describe/verify pass their own.
func readActualState(cfg Config, onUnreadable, scope string, keep map[string]bool, createdAt string, stderr io.Writer) (*manifest.Manifest, int) {
	reader := state.NewReader()
	m, diags, errDiag := reader.Describe(state.Options{
		Root:         cfg.Root,
		OnUnreadable: state.OnUnreadable(onUnreadable),
		Scope:        state.ScanScope(scope),
		KeepList:     keep,
		CreatedAt:    createdAt,
	})
	for _, d := range diags {
		emitDiag(stderr, d)
	}
	if errDiag != nil {
		emitDiag(stderr, *errDiag)
		return nil, ExitError
	}
	return m, ExitOK
}
