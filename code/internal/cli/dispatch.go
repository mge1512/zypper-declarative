// generated from spec: zypper-declarative.spec.md sha256:f8ff76ecbc4bbc69a49e2e32b2924da3a64df1ad46196e05ce8c137b684429b2
//
// The verb dispatcher and the top-level CLI contract: bare invocation, the
// bare-word global commands version and help (with tolerated --version/--help/-h
// aliases), unknown-verb/option/value handling, and exit-code mapping. Internal
// behaviours return Diagnostics; only this layer maps a Diagnostic's domain to
// an exit code.

package cli

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/mge1512/zypper-declarative/internal/diag"
	"github.com/mge1512/zypper-declarative/internal/meta"
)

// Exit codes per the spec and the cli-tool template.
const (
	ExitOK         = 0
	ExitLogical    = 1
	ExitInvocation = 2
)

// App carries the I/O streams so the dispatcher is testable and the binary is a
// thin wrapper.
type App struct {
	Stdout io.Writer
	Stderr io.Writer
}

// New returns an App wired to the process streams.
func New() *App {
	return &App{Stdout: os.Stdout, Stderr: os.Stderr}
}

// Run dispatches args (excluding the program name) and returns the process exit
// code.
func (a *App) Run(args []string) int {
	a.installSignalHandler()

	if len(args) == 0 {
		// Bare invocation: usage to stdout, exit 0 (discovery, never converges).
		a.printUsage(a.Stdout)
		return ExitOK
	}

	// Tolerated flag aliases for the global commands, accepted anywhere as the
	// first token.
	switch args[0] {
	case "version", "--version":
		fmt.Fprintln(a.Stdout, meta.VersionLine())
		return ExitOK
	case "help", "--help", "-h":
		a.printUsage(a.Stdout)
		return ExitOK
	}

	verb := args[0]
	rest := args[1:]

	switch verb {
	case "apply":
		return a.run(a.cmdApply, rest)
	case "diff":
		return a.run(a.cmdDiff, rest)
	case "verify":
		return a.run(a.cmdVerify, rest)
	case "status":
		return a.run(a.cmdStatus, rest)
	case "describe":
		return a.run(a.cmdDescribe, rest)
	default:
		a.diagUsage(diag.Errorf(diag.DomainInvocation, "unknown verb: %s", verb))
		return ExitInvocation
	}
}

// verbFunc is a verb implementation operating on a parsed Config and the bare
// arguments that follow the options. It returns an exit code.
type verbFunc func(cfg Config, rest []string) int

// run parses the leading options, then invokes the verb. A parse error prints
// usage to stderr and exits 2.
func (a *App) run(fn verbFunc, args []string) int {
	cfg, rest, perr := parseOptions(args)
	if perr != nil {
		a.diagUsage(diag.Errorf(diag.DomainInvocation, "%s", perr.Error()))
		return ExitInvocation
	}
	return fn(cfg, rest)
}

// emit writes a diagnostic to stderr, one per line.
func (a *App) emit(d *diag.Diagnostic) {
	fmt.Fprintln(a.Stderr, d.Line())
}

// diagUsage emits a diagnostic and then usage to stderr (invocation errors).
func (a *App) diagUsage(d *diag.Diagnostic) {
	a.emit(d)
	a.printUsage(a.Stderr)
}

// exitFor maps a diagnostic domain to an exit code: invocation/transaction
// errors are invocation (2) where the spec says so; logical failures are 1.
// Callers pass the intended class explicitly via the verb logic; this helper is
// used where a domain alone determines the class.
func exitForInvocation(d *diag.Diagnostic) int {
	switch d.Domain {
	case diag.DomainInvocation, diag.DomainTransaction:
		return ExitInvocation
	default:
		return ExitLogical
	}
}

func (a *App) installSignalHandler() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sigChan
		// Clean exit; no partial output. An interrupted apply has not yet sealed
		// or activated any snapshot, so leaving here makes no new generation the
		// default boot target.
		os.Exit(ExitOK)
	}()
}

// printUsage writes the usage text.
func (a *App) printUsage(w io.Writer) {
	fmt.Fprint(w, usageText)
}

const usageText = `usage: zypper-declarative <verb> [key=value ...]

Verbs:
  apply       Converge the system to the desired manifest in one snapshot transaction.
  diff        Dry run: print what apply would change. No modification.
  verify      Check the actual state against the applied declaration.
  status      Print the current declarative state and a drift summary.
  describe    Emit the actual state of the declarable scopes as a manifest.

Global commands:
  version     Print program name, version, and the embedded spec hash.
  help        Print this usage.
  (Tolerated aliases: --version, --help, -h.)

Key=value options (precede any bare-word argument):
  mode=auto|external|internal       Transaction binding; default auto.
  manifest-path=<path>              Desired manifest; default from CONFIG.
  format=json|yaml                  Serialisation for this invocation's manifest I/O.
  manifest-format=json|yaml         Fallback serialisation; default json.
  state-path=<path>                 State dump as actual-state source for verify.
  root=<path>                       Root to describe; default "/".
  out=<path>                        describe output file; default stdout.
  on-unreadable=error|warn          describe: fail (default) or omit+warn on an unreadable source.
  repo-lock=<repo>                  Fallback pin when no repositories scope is declared.
  content-store=<path>              Base path for resolving config_files content_ref.
  keep-list=<path>                  Allowlist of persistent-but-undeclared paths.
  signature-verification=on|off     Verify the manifest signature; default on.
  keyring=<path>                    Keyring path for signature verification.
  activation-policy=reboot|soft-reboot|none
  applied-root=<path>               Generation root for the applied record; default "/".

Exit codes: 0 success; 1 logical failure; 2 invocation error.
`
