// generated from spec: zypper-declarative.spec.md sha256:f2cc80627e483a48bb8411d297711bc5f6c6e74c28dbf0dafc8fe7bd8817251e
//
// Package cli is the verb layer: dispatch, key=value parsing, the global CLI
// contract, and exit-code mapping. It is the only layer that translates a
// behaviour's returned Diagnostic into a process exit code.
package cli

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/mge1512/zypper-declarative/internal/diag"
	"github.com/mge1512/zypper-declarative/internal/manifest"
	"github.com/mge1512/zypper-declarative/internal/meta"
	"github.com/mge1512/zypper-declarative/internal/state"
	"github.com/mge1512/zypper-declarative/internal/txn"
)

// Exit codes per the spec / cli-tool template.
const (
	ExitOK         = 0
	ExitError      = 1
	ExitInvocation = 2
)

// IO bundles the output streams so the dispatcher is testable.
type IO struct {
	Stdout io.Writer
	Stderr io.Writer
}

// Config holds the resolved CONFIG knobs and per-invocation options.
type Config struct {
	Mode            string
	ManifestPath    string
	ManifestFormat  string // default json
	Format          string // explicit format= for this invocation
	StatePath       string
	Root            string
	Out             string
	OnUnreadable    string // error | warn
	Scope           string // etc | full
	RepoLock        string
	ContentStore    string
	KeepList        string
	SignatureVerify string // on | off
	Keyring         string
	ActivationPol   string
	AppliedRoot     string
}

func defaultConfig() *Config {
	return &Config{
		Mode:            "auto",
		ManifestFormat:  "json",
		OnUnreadable:    "error",
		Scope:           "etc",
		SignatureVerify: "off",
		AppliedRoot:     "/",
		Root:            "/",
	}
}

// knownOptions is the set of accepted key=value option keys.
var knownOptions = map[string]bool{
	"mode": true, "manifest-path": true, "manifest-format": true, "format": true,
	"state-path": true, "root": true, "out": true, "on-unreadable": true,
	"scope": true, "repo-lock": true, "content-store": true, "keep-list": true,
	"signature-verification": true, "keyring": true, "activation-policy": true,
	"applied-root": true, "transaction-mode": true,
}

// parseArgs splits args into options (key=value) and bare words. A key=value
// token is always an option (wherever it appears); any other token is a bare
// word (a verb or a verb's positional). It returns an invocation diagnostic on
// an unknown option key or an invalid option value.
func parseArgs(cfg *Config, args []string) (bareWords []string, d *diag.Diagnostic) {
	for _, a := range args {
		if i := strings.IndexByte(a, '='); i > 0 {
			key, val := a[:i], a[i+1:]
			if knownOptions[key] {
				if d := applyOption(cfg, key, val); d != nil {
					return nil, d
				}
				continue
			}
			return nil, diag.New(diag.DomainInvocation, "unknown option %q", key)
		}
		bareWords = append(bareWords, a)
	}
	return bareWords, nil
}

// applyOption sets a config field from a validated key=value option.
func applyOption(cfg *Config, key, val string) *diag.Diagnostic {
	switch key {
	case "mode", "transaction-mode":
		if _, d := txn.ParseMode(val); d != nil {
			return d
		}
		cfg.Mode = val
	case "manifest-path":
		cfg.ManifestPath = val
	case "manifest-format":
		if val != "json" && val != "yaml" {
			return diag.New(diag.DomainInvocation, "unknown manifest-format %q", val)
		}
		cfg.ManifestFormat = val
	case "format":
		if val != "json" && val != "yaml" {
			return diag.New(diag.DomainInvocation, "unknown format value %q", val)
		}
		cfg.Format = val
	case "state-path":
		cfg.StatePath = val
	case "root":
		cfg.Root = val
	case "out":
		cfg.Out = val
	case "on-unreadable":
		if val != "error" && val != "warn" {
			return diag.New(diag.DomainInvocation, "unknown on-unreadable value %q", val)
		}
		cfg.OnUnreadable = val
	case "scope":
		if val != "etc" && val != "full" {
			return diag.New(diag.DomainInvocation, "unknown scope value %q", val)
		}
		cfg.Scope = val
	case "repo-lock":
		cfg.RepoLock = val
	case "content-store":
		cfg.ContentStore = val
	case "keep-list":
		cfg.KeepList = val
	case "signature-verification":
		if val != "on" && val != "off" {
			return diag.New(diag.DomainInvocation, "unknown signature-verification value %q", val)
		}
		cfg.SignatureVerify = val
	case "keyring":
		cfg.Keyring = val
	case "activation-policy":
		cfg.ActivationPol = val
	case "applied-root":
		cfg.AppliedRoot = val
	}
	return nil
}

// formatForLoad returns the validated explicit format for manifest loading.
func (c *Config) explicitFormat() manifest.Format {
	if c.Format == "" {
		return ""
	}
	return manifest.Format(c.Format)
}

func (c *Config) defaultFormat() manifest.Format {
	if c.ManifestFormat == "yaml" {
		return manifest.FormatYAML
	}
	return manifest.FormatJSON
}

func (c *Config) onUnreadable() state.OnUnreadable {
	if c.OnUnreadable == "warn" {
		return state.OnUnreadableWarn
	}
	return state.OnUnreadableError
}

func (c *Config) scopeVal() state.Scope {
	if c.Scope == "full" {
		return state.ScopeFull
	}
	return state.ScopeEtc
}

// usageText is printed for help, bare invocation, and invocation errors.
const usageText = `usage: zypper-declarative <verb> [key=value ...]

Verbs:
  apply       converge the system to the desired manifest (privileged)
  diff        print what apply would change (no modification)
  verify      check the actual state against a reference declaration
  status      print the current declarative state
  describe    emit the actual state as a Manifest (json or yaml)

Global commands:
  version     print program name, version, and embedded spec hash
  help        print this usage

Common options (key=value; POSIX --flag style is not used for options):
  mode=auto|external|internal       transaction binding (default auto)
  manifest-path=<path>              desired/reference manifest
  state-path=<path>                 captured actual state (offline)
  format=json|yaml                  serialisation for this invocation
  manifest-format=json|yaml         default serialisation (default json)
  root=<path>                       describe root (default /)
  out=<path>                        describe output file (default stdout)
  on-unreadable=error|warn          describe unreadable-source policy
  scope=etc|full                    describe/verify read scope (default etc)
  applied-root=<path>               generation root for the applied record
  repo-lock, content-store, keep-list, signature-verification, keyring,
  activation-policy                 additional CONFIG knobs
`

func printUsage(w io.Writer) {
	fmt.Fprint(w, usageText)
}

// emitDiagnostics writes diagnostics to stderr, one per line.
func emitDiagnostics(w io.Writer, ds []*diag.Diagnostic) {
	for _, d := range ds {
		fmt.Fprintln(w, d.Error())
	}
}

// exitForDomain maps a diagnostic domain to a logical/invocation exit code.
func exitForDomain(d *diag.Diagnostic) int {
	switch d.Domain {
	case diag.DomainInvocation, diag.DomainTransaction:
		return ExitInvocation
	default:
		return ExitError
	}
}

// sortedNames returns sorted keys of a string-set, for deterministic output.
func sortedNames(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// version prints the canonical version line.
func runVersion(io IO) int {
	fmt.Fprintln(io.Stdout, meta.VersionLine())
	return ExitOK
}
