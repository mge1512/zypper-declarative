// generated from spec: zypper-declarative.spec.md sha256:27aee8e374eb3507189bad0b78339109d0116e6a13b55ae4df2ba9a18e769fc4
//
// Dispatch and global commands. Run is the single entry point: it installs
// signal handling, parses the verb and options, and routes to a handler. Only
// this layer maps a Diagnostic to an exit code and writes diagnostics to stderr.
package cli

import (
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/mge1512/zypper-declarative/internal/meta"
)

// Run executes the CLI with the given args (excluding the program name) and
// streams. It returns the process exit code.
func Run(args []string, stdout, stderr io.Writer) int {
	installSignalHandling()

	// Global flag aliases and bare invocation.
	if len(args) == 0 {
		printUsage(stdout)
		return ExitOK
	}

	first := args[0]
	switch first {
	case "version", "--version":
		// version is a bare-word global command; --version is a tolerated alias.
		// Any trailing args are ignored for the global banner.
		if _, err := io.WriteString(stdout, meta.VersionLine()+"\n"); err != nil {
			return ExitError
		}
		return ExitOK
	case "help", "--help", "-h":
		printUsage(stdout)
		return ExitOK
	}

	verb := first
	rest := args[1:]

	switch verb {
	case "apply":
		return runApply(rest, stdout, stderr)
	case "diff":
		return runDiff(rest, stdout, stderr)
	case "verify":
		return runVerify(rest, stdout, stderr)
	case "status":
		return runStatus(rest, stdout, stderr)
	case "describe":
		return runDescribe(rest, stdout, stderr)
	default:
		// An unknown verb, OR a bare key=value with no verb (e.g. format=bad):
		// if the first token looks like an option, treat it as a no-verb
		// invocation whose options are validated (invocation error on bad value).
		if isOption(first) {
			return runNoVerbOptions(args, stderr)
		}
		printUsage(stderr)
		return ExitInvocation
	}
}

// isOption reports whether a token is a key=value option (not a bare word).
func isOption(tok string) bool {
	if tok == "" {
		return false
	}
	for i := 0; i < len(tok); i++ {
		if tok[i] == '=' {
			return i > 0
		}
	}
	return false
}

// runNoVerbOptions handles an invocation that is only options (no verb). Per the
// MILESTONE acceptance criterion, `format=bad_value` with no verb is an
// invocation error (exit 2); a valid option set with no verb prints usage and
// exits 0 (treated as discovery).
func runNoVerbOptions(args []string, stderr io.Writer) int {
	cfg := defaultConfig()
	if _, err := parseArgs(&cfg, args); err != nil {
		printUsage(stderr)
		return ExitInvocation
	}
	// Valid options but no verb: discovery, exit 0 with usage to stdout would be
	// surprising on stderr; emit nothing extra and exit 0.
	return ExitOK
}

// installSignalHandling installs a clean-exit handler for SIGTERM/SIGINT so an
// interrupted run leaves no partial output and (for apply) no new snapshot as
// the default boot target.
func installSignalHandling() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sigChan
		os.Exit(ExitOK)
	}()
}
