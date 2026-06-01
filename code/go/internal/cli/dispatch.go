// generated from spec: zypper-declarative.spec.md sha256:f2cc80627e483a48bb8411d297711bc5f6c6e74c28dbf0dafc8fe7bd8817251e
//
// Dispatch: the top-level CLI contract. Bare invocation, version, and help
// print to stdout and exit 0; unknown verb/option/value print usage to stderr
// and exit 2. Each behaviour verb is dispatched to its implementation.
package cli

import (
	"fmt"

	"github.com/mge1512/zypper-declarative/internal/diag"
)

// Run is the entry point invoked from main(). args excludes the program name.
func Run(args []string, io IO) int {
	// Global flag aliases and bare-word global commands are handled first,
	// since they may appear as the only token.
	if len(args) == 0 {
		// Bare invocation: usage to stdout, exit 0 (discovery, never converges).
		printUsage(io.Stdout)
		return ExitOK
	}

	// The first token is either a verb, a bare-word global command, a tolerated
	// flag alias, or (if it contains '=') an option preceding no verb.
	first := args[0]
	switch first {
	case "version", "--version":
		return runVersion(io)
	case "help", "--help", "-h":
		printUsage(io.Stdout)
		return ExitOK
	}

	// If the first token is an option (key=value) with no verb following, this
	// is a discovery-style invocation with options but no verb. Per the global
	// contract a bad option value is exit 2; a well-formed option with no verb
	// is treated as an invocation error (no verb to act on) — except that a bad
	// format value must be detected (acceptance gate format=bad_value -> 2).
	cfg := defaultConfig()
	bare, d := parseArgs(cfg, args)
	if d != nil {
		emitDiagnostics(io.Stderr, []*diag.Diagnostic{d})
		printUsage(io.Stderr)
		return ExitInvocation
	}
	if len(bare) == 0 {
		// Options only, no verb: invocation error.
		fmt.Fprintln(io.Stderr, "Error [invocation] no verb given")
		printUsage(io.Stderr)
		return ExitInvocation
	}

	verb := bare[0]
	rest := bare[1:]
	switch verb {
	case "apply":
		return runApply(cfg, rest, io)
	case "diff":
		return runDiff(cfg, rest, io)
	case "verify":
		return runVerify(cfg, rest, io)
	case "status":
		return runStatus(cfg, rest, io)
	case "describe":
		return runDescribe(cfg, rest, io)
	case "version":
		return runVersion(io)
	case "help":
		printUsage(io.Stdout)
		return ExitOK
	default:
		fmt.Fprintf(io.Stderr, "Error [invocation] unknown verb %q\n", verb)
		printUsage(io.Stderr)
		return ExitInvocation
	}
}

// rejectExtra rejects any trailing bare-word arguments a verb does not accept.
func rejectExtra(io IO, verb string, rest []string) bool {
	if len(rest) == 0 {
		return false
	}
	fmt.Fprintf(io.Stderr, "Error [invocation] %s: unrecognised argument %q\n", verb, rest[0])
	printUsage(io.Stderr)
	return true
}
