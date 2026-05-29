// generated from spec: zypper-declarative.spec.md sha256:714e75ff672557d2c7344736a5f36b52afa37e0f565b07de83e1b18cc4492014
//
// CLI argument parsing and dispatch. The entry-point binary calls Run; this
// package owns the key=value parsing, verb dispatch, and the usage/version
// text. Control via environment variables is forbidden (CONFIG-ENV-VARS).
package zypperdeclarative

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// usageText returns the usage string. The "usage:" prefix is asserted by the
// help/argument-error tests and the milestone acceptance criteria.
func usageText() string {
	return `usage: zypper-declarative <verb> [key=value ...]

verbs:
  apply      converge the system to the desired manifest
  diff       print what apply would change (dry run)
  verify     check the actual state against the applied record
  status     print the current declarative state
  describe   read the actual state and emit it as a manifest

options (key=value, precede bare-word arguments):
  mode=auto|external|internal   transaction binding; default auto
  manifest-path=<path>          desired manifest; default from CONFIG
  format=json|yaml              manifest input / describe output format
  state-path=<path>             state dump as actual-state source for verify
  root=<path>                   root to describe; default "/"
  out=<path>                    describe output file; default stdout
  applied-root=<path>           generation root for the applied record; default "/"
  keep-list=<path>              allowlist of persistent-but-undeclared paths
  content-store=<path>          base path for resolving content_ref values
  repo-lock=<repo>              fallback pinned repository
  signature-verification=on|off enable/disable manifest signature checking
  keyring=<path>                keyring path when signature verification is on
  activation-policy=reboot|soft-reboot|none   activation of a sealed snapshot

global:
  --version    print version and embedded spec hash, then exit 0
  --help       print this usage and exit 0
`
}

// versionText returns the version line including the spec hash.
func versionText() string {
	return fmt.Sprintf("zypper-declarative %s spec:%s", Version, SpecSHA256)
}

// Run is the CLI entry point. It parses args (excluding the program name),
// dispatches to a verb, and returns the process exit code. stdout/stderr are
// injected for testability.
func Run(args []string, stdout, stderr io.Writer) int {
	// Global flags first.
	for _, a := range args {
		switch a {
		case "--version", "version":
			fmt.Fprintln(stdout, versionText())
			return ExitOK
		case "--help", "help", "-h":
			fmt.Fprint(stdout, usageText())
			return ExitOK
		}
	}

	if len(args) == 0 {
		fmt.Fprintln(stderr, "Error: [invocation] no verb given")
		fmt.Fprint(stderr, usageText())
		return ExitInvocation
	}

	verb := args[0]
	rest := args[1:]

	cfg := defaultConfig()
	var extraArgs []string
	if d := parseOptions(&cfg, rest, &extraArgs); d != nil {
		fmt.Fprintf(stderr, "%s: [%s] %s\n", d.Severity, d.Domain, d.Message)
		fmt.Fprint(stderr, usageText())
		return ExitInvocation
	}

	v := &Verbs{
		Providers: NewProductionProviders(),
		Stdout:    stdout,
		Stderr:    stderr,
	}

	switch verb {
	case "apply":
		if len(extraArgs) > 0 {
			fmt.Fprintf(stderr, "Error: [invocation] unrecognised argument: %s\n", strings.Join(extraArgs, " "))
			fmt.Fprint(stderr, usageText())
			return ExitInvocation
		}
		return v.Apply(&cfg)
	case "diff":
		if len(extraArgs) > 0 {
			fmt.Fprintf(stderr, "Error: [invocation] unrecognised argument: %s\n", strings.Join(extraArgs, " "))
			fmt.Fprint(stderr, usageText())
			return ExitInvocation
		}
		return v.Diff(&cfg)
	case "verify":
		if len(extraArgs) > 0 {
			fmt.Fprintf(stderr, "Error: [invocation] unrecognised argument: %s\n", strings.Join(extraArgs, " "))
			fmt.Fprint(stderr, usageText())
			return ExitInvocation
		}
		return v.Verify(&cfg)
	case "status":
		return v.Status(&cfg, extraArgs)
	case "describe":
		return v.Describe(&cfg, extraArgs)
	default:
		fmt.Fprintf(stderr, "Error: [invocation] unknown verb: %s\n", verb)
		fmt.Fprint(stderr, usageText())
		return ExitInvocation
	}
}

// parseOptions parses key=value options into cfg. Unrecognised tokens are
// collected into extraArgs for the verb to reject (status/describe reject any;
// apply/diff/verify reject any). A key=value token with an unknown key is an
// extra argument.
func parseOptions(cfg *Config, tokens []string, extraArgs *[]string) *Diagnostic {
	for _, tok := range tokens {
		if !strings.Contains(tok, "=") {
			*extraArgs = append(*extraArgs, tok)
			continue
		}
		kv := strings.SplitN(tok, "=", 2)
		key, val := kv[0], kv[1]
		switch key {
		case "mode":
			switch TransactionMode(val) {
			case ModeAuto, ModeExternal, ModeInternal:
				cfg.TransactionMode = TransactionMode(val)
			default:
				return newError(DomainInvocation, "invalid mode value: "+val)
			}
		case "manifest-path":
			cfg.ManifestPath = val
		case "manifest-format":
			cfg.ManifestFormat = ManifestFormat(val)
		case "format":
			cfg.Format = ManifestFormat(val)
			cfg.FormatSet = true
		case "state-path":
			cfg.StatePath = val
		case "root":
			cfg.Root = val
		case "out":
			cfg.Out = val
		case "applied-root":
			cfg.AppliedRoot = val
		case "keep-list":
			cfg.KeepListPath = val
		case "content-store":
			cfg.ContentStore = val
		case "repo-lock":
			cfg.RepoLock = val
		case "signature-verification":
			switch val {
			case "on":
				cfg.SignatureVerification = true
			case "off":
				cfg.SignatureVerification = false
			default:
				return newError(DomainInvocation, "invalid signature-verification value: "+val)
			}
		case "keyring":
			cfg.KeyringPath = val
		case "activation-policy":
			switch val {
			case "reboot", "soft-reboot", "none":
				cfg.ActivationPolicy = val
			default:
				return newError(DomainInvocation, "invalid activation-policy value: "+val)
			}
		default:
			*extraArgs = append(*extraArgs, tok)
		}
	}
	return nil
}

// exitForDomain maps a diagnostic domain to the verb-level exit code per the
// spec's ERRORS and ExitCode definitions.
func exitForDomain(domain string) int {
	switch domain {
	case DomainInvocation, DomainTransaction:
		return ExitInvocation
	default:
		return ExitLogical
	}
}

// writeFileStrict writes data to path, failing (rather than creating
// directories) when the parent is not writable, so an unwritable output path
// is reported as an invocation error.
func writeFileStrict(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(data)
	return err
}
