// generated from spec: zypper-declarative.spec.md sha256:58e1636e2de82ab81a5cd3f81d6b3c9ac6a8976e18f9abb2bbd2b2aba56fe4d4
//
// Package cli is the verb layer: argument parsing (key=value), the global CLI
// contract, verb dispatch, and exit-code mapping. Only this layer maps a
// Diagnostic's domain to an exit code; internal behaviours return errors.
package cli

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/mge1512/zypper-declarative/internal/config"
	"github.com/mge1512/zypper-declarative/internal/diag"
	"github.com/mge1512/zypper-declarative/internal/manifest"
	"github.com/mge1512/zypper-declarative/internal/meta"
)

// Exit codes per the spec / cli-tool template.
const (
	ExitOK         = 0
	ExitError      = 1
	ExitInvocation = 2
)

// App carries the IO streams so the verb layer is testable.
type App struct {
	Stdout io.Writer
	Stderr io.Writer
}

// New returns an App wired to the process std streams.
func New() *App {
	return &App{Stdout: os.Stdout, Stderr: os.Stderr}
}

// Run is the entry point invoked by main. args excludes the program name. It
// returns the process exit code.
func (a *App) Run(args []string) int {
	a.installSignalHandler()

	// Global flags first.
	for _, arg := range args {
		switch arg {
		case "--version":
			fmt.Fprintf(a.Stdout, "%s %s spec:%s\n", meta.Name, meta.Version, meta.SpecSHA256)
			return ExitOK
		case "--help", "-h":
			a.printUsage(a.Stdout)
			return ExitOK
		}
	}

	// Bare invocation (no verb): print usage to stdout, exit 0 (discovery).
	if len(args) == 0 {
		a.printUsage(a.Stdout)
		return ExitOK
	}

	verb := args[0]
	rest := args[1:]

	switch verb {
	case "apply", "diff", "verify", "status", "describe":
		// handled below
	default:
		fmt.Fprintf(a.Stderr, "%s: unknown verb %q\n", meta.Name, verb)
		a.printUsage(a.Stderr)
		return ExitInvocation
	}

	cfg, optErr := a.parseOptions(rest, verb)
	if optErr != nil {
		fmt.Fprintln(a.Stderr, optErr.Line())
		a.printUsage(a.Stderr)
		return ExitInvocation
	}

	switch verb {
	case "apply":
		return a.runApply(cfg)
	case "diff":
		return a.runDiff(cfg)
	case "verify":
		return a.runVerify(cfg)
	case "status":
		return a.runStatus(cfg)
	case "describe":
		return a.runDescribe(cfg)
	}
	return ExitInvocation
}

// parseOptions parses key=value options (and rejects unknown options / values /
// bare words) into a Config. Options precede any bare-word argument; for the
// verbs here there are no further bare words, so a bare word is an error.
func (a *App) parseOptions(args []string, verb string) (config.Config, *diag.Diagnostic) {
	cfg := config.Defaults()
	for _, arg := range args {
		eq := strings.IndexByte(arg, '=')
		if eq < 0 {
			return cfg, diag.Errorf(diag.DomainInvocation, "unknown argument %q (options are key=value)", arg)
		}
		key := arg[:eq]
		val := arg[eq+1:]
		if d := applyOption(&cfg, key, val); d != nil {
			return cfg, d
		}
	}
	return cfg, nil
}

// applyOption applies one key=value option to cfg, validating the key and value.
func applyOption(cfg *config.Config, key, val string) *diag.Diagnostic {
	switch key {
	case "mode", "transaction-mode":
		switch val {
		case "auto", "external", "internal":
			cfg.TransactionMode = val
		default:
			return diag.Errorf(diag.DomainInvocation, "unknown transaction mode %q", val)
		}
	case "manifest-path":
		cfg.ManifestPath = val
	case "format":
		f, given, err := manifest.ParseFormat(val)
		if err != nil {
			return diag.Errorf(diag.DomainInvocation, "unknown format value %q", val)
		}
		cfg.ExplicitFormat = f
		cfg.ExplicitFormatGiven = given
	case "manifest-format":
		f, given, err := manifest.ParseFormat(val)
		if err != nil || !given {
			return diag.Errorf(diag.DomainInvocation, "unknown manifest-format value %q", val)
		}
		cfg.ManifestFormat = f
	case "state-path":
		cfg.StatePath = val
	case "root":
		cfg.Root = val
	case "out":
		cfg.Out = val
	case "on-unreadable":
		switch val {
		case "error":
			cfg.OnUnreadable = config.OnUnreadableError
		case "warn":
			cfg.OnUnreadable = config.OnUnreadableWarn
		default:
			return diag.Errorf(diag.DomainInvocation, "unknown on-unreadable value %q", val)
		}
	case "repo-lock":
		cfg.RepoLock = val
	case "content-store":
		cfg.ContentStore = val
	case "keep-list":
		cfg.KeepList = val
	case "signature-verification":
		switch val {
		case "on":
			cfg.SignatureVerification = true
		case "off":
			cfg.SignatureVerification = false
		default:
			return diag.Errorf(diag.DomainInvocation, "unknown signature-verification value %q", val)
		}
	case "keyring":
		cfg.Keyring = val
	case "activation-policy":
		switch val {
		case "reboot", "soft-reboot", "none":
			cfg.ActivationPolicy = val
		default:
			return diag.Errorf(diag.DomainInvocation, "unknown activation-policy value %q", val)
		}
	case "applied-root":
		cfg.AppliedRoot = val
	default:
		return diag.Errorf(diag.DomainInvocation, "unknown option %q", key)
	}
	return nil
}

// emit writes diagnostics to stderr, one per line.
func (a *App) emit(ds ...*diag.Diagnostic) {
	for _, d := range ds {
		if d != nil {
			fmt.Fprintln(a.Stderr, d.Line())
		}
	}
}

// exitFor maps a diagnostic domain to an exit code.
func exitFor(d *diag.Diagnostic) int {
	if d == nil {
		return ExitOK
	}
	switch d.Domain {
	case diag.DomainInvocation, diag.DomainTransaction:
		return ExitInvocation
	default:
		return ExitError
	}
}

func (a *App) installSignalHandler() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sigChan
		// Clean exit; no partial output. An interrupted apply has not sealed or
		// activated any snapshot, so the running system is unchanged.
		os.Exit(ExitOK)
	}()
}

func (a *App) printUsage(w io.Writer) {
	fmt.Fprintf(w, `usage: %s <verb> [key=value ...]

Verbs:
  apply       converge the system to the desired manifest (privileged)
  diff        print what apply would change (dry run, read-only)
  verify      check whether actual state equals the declaration (read-only)
  status      print the current declarative state (read-only)
  describe    emit the actual state as a manifest (read-only)

Global:
  --version   print program name, version, and spec hash; exit 0
  --help, -h  print this usage; exit 0

Options (key=value, precede any bare-word argument):
  mode=auto|external|internal     transaction binding; default auto
  manifest-path=<path>            desired manifest; default from CONFIG
  format=json|yaml                serialisation for this invocation's manifest I/O
  state-path=<path>               state dump as actual-state source for verify
  root=<path>                     root to describe; default "/"
  out=<path>                      describe output file; default stdout
  on-unreadable=error|warn        describe: fail (default) or omit+warn
  manifest-format=json|yaml       fallback serialisation default
  repo-lock=<repo>                fallback pinned repository
  content-store=<path>            base path for content_ref resolution
  keep-list=<path>                allowlist of persistent-but-undeclared paths
  signature-verification=on|off   manifest signature verification; default on
  keyring=<path>                  keyring path when verification on
  activation-policy=reboot|soft-reboot|none
  applied-root=<path>             generation root for the applied record; default "/"

Exit codes:
  0  success (converged, no-op, matches declaration, or describe emitted)
  1  logical failure (convergence failed; verify drift; invalid manifest)
  2  invocation error (bad arguments; unreadable manifest; unavailable transaction)
`, meta.Name)
}
