// generated from spec: zypper-declarative.spec.md sha256:714e75ff672557d2c7344736a5f36b52afa37e0f565b07de83e1b18cc4492014
//
// Argument parsing and verb dispatch. Options are key=value pairs parsed by the
// tool itself; bare words are verbs. POSIX --flag style is not a supported
// option syntax (cli-tool template CLI-ARG-STYLE: key=value), but a leading
// "--" prefix on a key=value pair is tolerated and stripped, so "--key=value"
// is parsed identically to "key=value". Bare "--help"/"-h"/"--version" are
// recognised global words.
package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/mge1512/zypper-declarative/internal/manifest"
	"github.com/mge1512/zypper-declarative/internal/sysiface"
	"github.com/mge1512/zypper-declarative/internal/txn"
)

// Exit codes (cli-tool template + spec ExitCode).
const (
	ExitOK         = 0
	ExitLogical    = 1
	ExitInvocation = 2
)

// App wires the CLI to its system dependencies. The runner seam allows tests to
// substitute a fake; production uses OSCommandRunner.
type App struct {
	Stdout io.Writer
	Stderr io.Writer
	Runner sysiface.CommandRunner
}

// Run parses args, dispatches the verb, and returns the process exit code.
// args excludes the program name.
func (a *App) Run(args []string) int {
	// Global words first.
	for _, arg := range args {
		switch arg {
		case "--help", "-h", "help":
			a.printUsage(a.Stdout)
			return ExitOK
		case "--version":
			fmt.Fprintf(a.Stdout, "zypper-declarative %s spec:%s\n", versionString(), specHash())
			return ExitOK
		}
	}

	if len(args) == 0 {
		// Bare invocation: print usage to stderr, exit 2 (invocation error).
		a.printUsage(a.Stderr)
		return ExitInvocation
	}

	verb := args[0]
	rest := args[1:]

	cfg, perr := a.parseOptions(rest)
	if perr != nil {
		fmt.Fprintln(a.Stderr, perr.Error())
		a.printUsage(a.Stderr)
		return ExitInvocation
	}

	switch verb {
	case "apply":
		return a.cmdApply(cfg)
	case "diff":
		return a.cmdDiff(cfg)
	case "verify":
		return a.cmdVerify(cfg)
	case "status":
		return a.cmdStatus(cfg, rest)
	case "describe":
		return a.cmdDescribe(cfg)
	default:
		fmt.Fprintf(a.Stderr, "Error [invocation] unknown verb %q\n", verb)
		a.printUsage(a.Stderr)
		return ExitInvocation
	}
}

// parseOptions parses key=value options into a Config, applying defaults first.
func (a *App) parseOptions(args []string) (Config, error) {
	cfg := defaultConfig()
	for _, raw := range args {
		arg := raw
		// Tolerate a leading "--" so "--key=value" parses as "key=value".
		arg = strings.TrimPrefix(arg, "--")
		k, v, ok := strings.Cut(arg, "=")
		if !ok {
			return Config{}, fmt.Errorf("Error [invocation] expected key=value, got %q", raw)
		}
		switch k {
		case "mode", "transaction-mode":
			switch v {
			case "auto", "external", "internal":
				cfg.TransactionMode = txn.Mode(v)
				cfg.Mode = v
			default:
				return Config{}, fmt.Errorf("Error [invocation] unknown transaction mode %q", v)
			}
		case "manifest-path":
			cfg.ManifestPath = v
		case "manifest-format":
			f, err := manifest.ResolveFormat(v, "", manifest.FormatJSON)
			if err != nil {
				return Config{}, fmt.Errorf("Error [invocation] %v", err)
			}
			cfg.ManifestFormat = f
		case "format":
			if v != "json" && v != "yaml" {
				return Config{}, fmt.Errorf("Error [invocation] unknown format value %q", v)
			}
			cfg.Format = v
		case "state-path":
			cfg.StatePath = v
		case "root":
			cfg.Root = v
		case "out":
			cfg.Out = v
		case "repo-lock":
			cfg.RepoLock = v
		case "content-store":
			cfg.ContentStore = v
		case "keep-list":
			cfg.KeepListPath = v
		case "signature-verification":
			switch v {
			case "on":
				cfg.SignatureVerification = true
			case "off":
				cfg.SignatureVerification = false
			default:
				return Config{}, fmt.Errorf("Error [invocation] signature-verification must be on|off, got %q", v)
			}
		case "keyring":
			cfg.Keyring = v
		case "activation-policy":
			switch v {
			case "reboot", "soft-reboot", "none":
				cfg.ActivationPolicy = v
			default:
				return Config{}, fmt.Errorf("Error [invocation] unknown activation-policy %q", v)
			}
		case "applied-root":
			cfg.AppliedRoot = v
		default:
			return Config{}, fmt.Errorf("Error [invocation] unknown option %q", k)
		}
	}
	return cfg, nil
}

func (a *App) printUsage(w io.Writer) {
	fmt.Fprint(w, `usage: zypper-declarative <verb> [key=value ...]
       zypper declarative <verb> [key=value ...]

Verbs:
  apply       Converge the system to the desired manifest in a snapshot transaction.
  diff        Dry run: print what apply would change. Makes no modification.
  verify      Check whether actual state equals the applied declaration.
  status      Print the current declarative state and a drift summary.
  describe    Emit the actual state of the declarable scopes (JSON or YAML).

Options (key=value, precede bare-word arguments):
  mode=auto|external|internal       Transaction binding; default auto.
  manifest-path=<path>              Desired manifest; default from CONFIG.
  manifest-format=json|yaml         Default input serialisation; default json.
  format=json|yaml                  Explicit input/describe-output format.
  state-path=<path>                 State dump as actual-state source for verify.
  root=<path>                       Root to describe; default "/".
  out=<path>                        Describe output file; default stdout.
  repo-lock=<repo>                  Fallback pinned repo when manifest has none.
  content-store=<path>              Base path for content_ref resolution.
  keep-list=<path>                  Allowlist of persistent-but-undeclared paths.
  signature-verification=on|off     Manifest signature check; default on.
  keyring=<path>                    Keyring path when verification is on.
  activation-policy=reboot|soft-reboot|none
  applied-root=<path>               Generation root for the applied record.

Exit codes: 0 success; 1 logical failure; 2 invocation error.
`)
}
