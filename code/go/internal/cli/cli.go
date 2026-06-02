// generated from spec: zypper-declarative.spec.md sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
//
// Package cli is the entry-point dispatch layer: key=value argument parsing, the
// global contract (version/help/bare/unknown), and exit-code mapping. It calls
// into the internal behaviour packages and is the only place exit codes are set.
package cli

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"

	"github.com/mge1512/zypper-declarative/internal/manifest"
	"github.com/mge1512/zypper-declarative/internal/meta"
)

// Exit codes per the spec ExitCode type and the cli-tool template.
const (
	ExitOK         = 0
	ExitError      = 1
	ExitInvocation = 2
)

// App holds the IO streams and parsed configuration for one invocation.
type App struct {
	Stdout io.Writer
	Stderr io.Writer
}

// Config holds the resolved key=value options for one invocation.
type Config struct {
	Verb         string
	ManifestPath string
	StatePath    string
	Root         string
	Out          string
	Format       *manifest.Format
	OnUnreadable string
	Scope        string
	Mode         string

	ManifestFormat string
	RepoLock       string
	ContentStore   string
	KeepListPath   string
	SigVerify      string
	Keyring        string
	ActivationPol  string
	AppliedRoot    string
}

// knownOptionKeys is the set of accepted key=value option keys.
var knownOptionKeys = map[string]bool{
	"manifest-path": true, "state-path": true, "root": true, "out": true,
	"format": true, "on-unreadable": true, "scope": true, "mode": true,
	"manifest-format": true, "repo-lock": true, "content-store": true,
	"keep-list": true, "signature-verification": true, "keyring": true,
	"activation-policy": true, "applied-root": true,
}

// Run is the dispatcher. It returns the process exit code.
func (a *App) Run(args []string) int {
	if a.Stdout == nil {
		a.Stdout = os.Stdout
	}
	if a.Stderr == nil {
		a.Stderr = os.Stderr
	}

	installSignalHandlers()

	// Global commands and bare invocation, before option parsing.
	if len(args) == 0 {
		a.printUsage(a.Stdout)
		return ExitOK
	}
	switch args[0] {
	case "version", "--version":
		fmt.Fprintf(a.Stdout, "%s %s spec:%s\n", meta.ProgramName, meta.Version, meta.SpecSHA256)
		return ExitOK
	case "help", "--help", "-h":
		a.printUsage(a.Stdout)
		return ExitOK
	}

	verb := args[0]
	switch verb {
	case "apply", "diff", "verify", "status", "describe":
		// recognised verb
	default:
		fmt.Fprintf(a.Stderr, "[Error] invocation: unknown verb %q\n", verb)
		a.printUsage(a.Stderr)
		return ExitInvocation
	}

	cfg, err := a.parseOptions(verb, args[1:])
	if err != nil {
		fmt.Fprintf(a.Stderr, "[Error] invocation: %s\n", err.Error())
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

// parseOptions parses key=value options for a verb. Bare words after options are
// not used by any verb here, so an unrecognised token is an invocation error.
func (a *App) parseOptions(verb string, args []string) (*Config, error) {
	cfg := &Config{
		Verb:           verb,
		Root:           "/",
		OnUnreadable:   "error",
		Scope:          "etc",
		Mode:           "auto",
		ManifestFormat: "json",
		SigVerify:      "off", // default off for the read-only/offline paths in this build
		AppliedRoot:    "/",
	}
	for _, arg := range args {
		eq := strings.IndexByte(arg, '=')
		if eq < 0 {
			return nil, fmt.Errorf("unrecognised argument %q (options are key=value)", arg)
		}
		key := arg[:eq]
		val := arg[eq+1:]
		if !knownOptionKeys[key] {
			return nil, fmt.Errorf("unknown option %q", key)
		}
		switch key {
		case "manifest-path":
			cfg.ManifestPath = val
		case "state-path":
			cfg.StatePath = val
		case "root":
			cfg.Root = val
		case "out":
			cfg.Out = val
		case "format":
			f, ferr := manifest.ParseFormat(val)
			if ferr != nil {
				return nil, fmt.Errorf("unknown format value %q", val)
			}
			cfg.Format = &f
		case "on-unreadable":
			if val != "error" && val != "warn" {
				return nil, fmt.Errorf("unknown on-unreadable value %q", val)
			}
			cfg.OnUnreadable = val
		case "scope":
			if val != "etc" && val != "full" {
				return nil, fmt.Errorf("unknown scope value %q", val)
			}
			if verb != "describe" && verb != "verify" {
				return nil, fmt.Errorf("scope is accepted only on describe and verify")
			}
			cfg.Scope = val
		case "mode":
			if val != "auto" && val != "external" && val != "internal" {
				return nil, fmt.Errorf("unknown mode value %q", val)
			}
			cfg.Mode = val
		case "manifest-format":
			f, ferr := manifest.ParseFormat(val)
			if ferr != nil {
				return nil, fmt.Errorf("unknown manifest-format value %q", val)
			}
			cfg.ManifestFormat = string(f)
		case "repo-lock":
			cfg.RepoLock = val
		case "content-store":
			cfg.ContentStore = val
		case "keep-list":
			cfg.KeepListPath = val
		case "signature-verification":
			if val != "on" && val != "off" {
				return nil, fmt.Errorf("unknown signature-verification value %q", val)
			}
			cfg.SigVerify = val
		case "keyring":
			cfg.Keyring = val
		case "activation-policy":
			cfg.ActivationPol = val
		case "applied-root":
			cfg.AppliedRoot = val
		}
	}
	return cfg, nil
}

func (c *Config) defaultFormat() manifest.Format {
	if c.ManifestFormat == "yaml" {
		return manifest.FormatYAML
	}
	return manifest.FormatJSON
}

// loadKeepList reads the keep-list allowlist (one path per line). Absent -> empty.
func (c *Config) loadKeepList() map[string]bool {
	out := map[string]bool{}
	if c.KeepListPath == "" {
		return out
	}
	data, err := os.ReadFile(c.KeepListPath)
	if err != nil {
		return out
	}
	for _, line := range strings.Split(string(data), "\n") {
		l := strings.TrimSpace(line)
		if l != "" && !strings.HasPrefix(l, "#") {
			out[l] = true
		}
	}
	return out
}

func installSignalHandlers() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sigChan
		os.Exit(ExitOK)
	}()
}

func (a *App) printUsage(w io.Writer) {
	fmt.Fprint(w, usageText)
}

const usageText = `usage: zypper-declarative <verb> [key=value ...]

Verbs:
  apply        Converge the system to the desired manifest in a snapshot transaction.
  diff         Dry run: print what apply would change. No modification.
  verify       Check the actual state against a reference declaration.
  status       Print the current declarative state and a drift summary.
  describe     Read the actual state and emit it as a Manifest (JSON or YAML).

Global commands:
  version      Print program name, version, and embedded spec hash (alias --version).
  help         Print this usage (aliases --help, -h).

Options (key=value; precede any bare-word argument):
  mode=auto|external|internal       transaction binding (default auto)
  manifest-path=<path>              desired/reference manifest
  state-path=<path>                 captured actual state (verify, diff; offline)
  format=json|yaml                  serialisation for this invocation's manifest I/O
  root=<path>                       root to describe (default /)
  out=<path>                        describe output file (default stdout)
  on-unreadable=error|warn          describe: fail (default) or omit+warn
  scope=etc|full                    describe/verify read scope (default etc)
  manifest-format, repo-lock, content-store, keep-list,
  signature-verification, keyring, activation-policy, applied-root

Exit codes: 0 success; 1 logical failure; 2 invocation error.
`

// sortedDiagnosticDomains is a stable order helper (unused placeholder removed by linter if needed).
var _ = sort.Strings
