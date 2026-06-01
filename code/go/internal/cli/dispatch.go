// generated from spec: zypper-declarative.spec.md sha256:b2d0de88fbed1163678e59e931c741b9d999b71f902f6eb01db8790bb813d057
//
// Dispatch: the top-level CLI contract. Bare invocation and help print usage to
// stdout and exit 0; version prints program name, version, and the embedded
// spec hash and exits 0; an unknown verb, option, value, or missing value
// prints usage to stderr and exits 2. Exit-code mapping lives only here.
package cli

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/mge1512/zypper-declarative/internal/manifest"
	"github.com/mge1512/zypper-declarative/internal/meta"
)

// Run is the single entry point. It parses argv (excluding the program name),
// dispatches to a verb, and returns the process exit code. stdout and stderr
// are injected for testability; production passes os.Stdout/os.Stderr.
func Run(args []string, stdout, stderr io.Writer) int {
	installSignalHandlers()

	// Tolerated flag aliases for the two global commands.
	if len(args) == 1 {
		switch args[0] {
		case "--version":
			printVersion(stdout)
			return 0
		case "--help", "-h":
			printUsage(stdout)
			return 0
		}
	}

	p := parseArgs(args)

	// Bare invocation (no verb): discovery. Usage to stdout, exit 0.
	if len(p.bareWords) == 0 {
		// A bare invocation that nonetheless carried a bad option value is
		// still an invocation error.
		cfg := defaultConfig()
		if d := applyOptions(&cfg, p.options); d != nil {
			fmt.Fprintln(stderr, d.Error())
			printUsage(stderr)
			return 2
		}
		printUsage(stdout)
		return 0
	}

	verb := p.bareWords[0]
	extraBare := p.bareWords[1:]

	switch verb {
	case "version":
		printVersion(stdout)
		return 0
	case "help":
		printUsage(stdout)
		return 0
	case "apply", "diff", "verify", "status", "describe":
		// fallthrough to verb handling below
	default:
		fmt.Fprintf(stderr, "%s: unknown verb: %s\n", meta.Program, verb)
		printUsage(stderr)
		return 2
	}

	// Resolve config from options.
	cfg := defaultConfig()
	if d := applyOptions(&cfg, p.options); d != nil {
		fmt.Fprintln(stderr, d.Error())
		printUsage(stderr)
		return 2
	}

	// Reject stray bare words after the verb (e.g. status --frobnicate, where
	// --frobnicate is not a key=value option so it lands as a bare word).
	if len(extraBare) > 0 {
		fmt.Fprintf(stderr, "%s: unrecognised argument: %s\n", meta.Program, strings.Join(extraBare, " "))
		printUsage(stderr)
		return 2
	}

	// scope is accepted only on describe and verify.
	if _, set := p.options["scope"]; set && verb != "describe" && verb != "verify" {
		fmt.Fprintf(stderr, "%s: scope is accepted only on describe and verify\n", meta.Program)
		printUsage(stderr)
		return 2
	}

	switch verb {
	case "apply":
		return runApply(cfg, stdout, stderr)
	case "diff":
		return runDiff(cfg, stdout, stderr)
	case "verify":
		return runVerify(cfg, stdout, stderr)
	case "status":
		return runStatus(cfg, stdout, stderr)
	case "describe":
		return runDescribe(cfg, stdout, stderr)
	}
	return 2
}

func printVersion(w io.Writer) {
	fmt.Fprintf(w, "%s %s spec:%s\n", meta.Program, meta.Version, meta.SpecSHA256)
}

func printUsage(w io.Writer) {
	fmt.Fprint(w, usageText)
}

const usageText = `usage: zypper-declarative <verb> [key=value ...]

Verbs:
  apply       converge the system to the desired manifest in a snapshot
  diff        dry run: print what apply would change
  verify      check the actual state against the applied declaration
  status      print the applied declaration and a drift summary
  describe    emit the actual state as a manifest (json default, yaml optional)
  version     print program name, version, and embedded spec hash
  help        print this usage

Options (key=value; precede any bare-word argument):
  mode=auto|external|internal        transaction binding (default auto)
  manifest-path=<path>               desired manifest (default from CONFIG)
  manifest-format=json|yaml          default serialisation (default json)
  format=json|yaml                   serialisation for this invocation's I/O
  state-path=<path>                  state dump as actual-state source (verify)
  root=<path>                        root to describe (default /)
  out=<path>                         describe output file (default stdout)
  on-unreadable=error|warn           describe: fail (default) or omit+warn
  scope=etc|full                     describe/verify read scope (default etc)
  repo-lock=<channel>                fallback pinned repo
  content-store=<path>               base path for content_ref resolution
  keep-list=<path>                   allowlist of persistent undeclared paths
  signature-verification=on|off      manifest signature checking (default on)
  keyring=<path>                     keyring for signature verification
  activation-policy=reboot|soft-reboot|none
  applied-root=<path>                generation root for the applied record

Tolerated aliases: --version, --help, -h. No option uses POSIX --flag style.
`

// installSignalHandlers ensures a clean exit on SIGTERM and SIGINT, leaving no
// partial output. An interrupted apply discards its transaction (the apply
// path holds no committed snapshot until its final seal step).
func installSignalHandlers() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-ch
		os.Exit(0)
	}()
}

// exitFor maps a diagnostic to its exit code for the verb layer: invocation
// and transaction-unavailable are exit 2; all other error domains are exit 1.
func exitForLoad(d *manifest.Diagnostic) int {
	if d.Domain == manifest.DomainInvocation {
		return 2
	}
	return 1
}
