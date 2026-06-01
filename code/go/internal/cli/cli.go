// generated from spec: zypper-declarative.spec.md sha256:27aee8e374eb3507189bad0b78339109d0116e6a13b55ae4df2ba9a18e769fc4
//
// Package cli is the dispatch layer: key=value argument parsing, the global
// command contract (version/help/bare invocation and tolerated flag aliases),
// and the five verb handlers that orchestrate the internal behaviours and map
// their results to exit codes. The entry-point (cmd/zypper-declarative) does
// nothing but call Run.
package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/mge1512/zypper-declarative/internal/manifest"
)

// Exit codes (spec ExitCode).
const (
	ExitOK         = 0
	ExitError      = 1
	ExitInvocation = 2
)

// Config holds the resolved CONFIG knobs and per-invocation options. All knobs
// are surfaced as key=value options; environment-variable control is forbidden.
type Config struct {
	TransactionMode       string
	ManifestPath          string
	ManifestFormat        string
	OnUnreadable          string
	Scope                 string
	RepoLock              string
	ContentStore          string
	KeepList              string
	SignatureVerification string
	Keyring               string
	ActivationPolicy      string
	AppliedRoot           string

	// per-invocation
	Format    string
	StatePath string
	Root      string
	Out       string
}

func defaultConfig() Config {
	return Config{
		TransactionMode:       "auto",
		ManifestPath:          "/var/lib/zypper-declarative/desired.json",
		ManifestFormat:        "json",
		OnUnreadable:          "error",
		Scope:                 "etc",
		SignatureVerification: "off", // verification mechanism is host-specific; default off in this build
		ActivationPolicy:      "reboot",
		AppliedRoot:           "/",
		Root:                  "/",
	}
}

// knownOptionKeys is the set of accepted key=value option keys.
var knownOptionKeys = map[string]bool{
	"mode": true, "transaction-mode": true, "manifest-path": true,
	"manifest-format": true, "format": true, "state-path": true,
	"root": true, "out": true, "on-unreadable": true, "scope": true,
	"repo-lock": true, "content-store": true, "keep-list": true,
	"signature-verification": true, "keyring": true,
	"activation-policy": true, "applied-root": true,
}

// usageErr is an invocation error carrying a message printed to stderr.
type usageErr struct{ msg string }

func (e usageErr) Error() string { return e.msg }

// parseArgs parses key=value options (which must precede any bare-word args) and
// the trailing bare words into a Config and an option set. A POSIX --flag style
// option, an unknown key, an unknown value, or a missing value is an invocation
// error (the global flag aliases are handled by the dispatcher before this).
func parseArgs(cfg *Config, args []string) (seen map[string]bool, err error) {
	seen = map[string]bool{}
	for _, a := range args {
		if !strings.Contains(a, "=") {
			return nil, usageErr{"unexpected argument: " + a}
		}
		if strings.HasPrefix(a, "-") {
			return nil, usageErr{"POSIX flag style is not supported for options: " + a}
		}
		eq := strings.IndexByte(a, '=')
		key := a[:eq]
		val := a[eq+1:]
		if !knownOptionKeys[key] {
			return nil, usageErr{"unknown option: " + key}
		}
		if err := applyOption(cfg, key, val); err != nil {
			return nil, err
		}
		seen[key] = true
	}
	return seen, nil
}

func applyOption(cfg *Config, key, val string) error {
	switch key {
	case "mode", "transaction-mode":
		switch val {
		case "auto", "external", "internal":
			cfg.TransactionMode = val
		default:
			return usageErr{"unknown value for " + key + ": " + val}
		}
	case "manifest-path":
		cfg.ManifestPath = val
	case "manifest-format":
		if val != "json" && val != "yaml" {
			return usageErr{"unknown value for manifest-format: " + val}
		}
		cfg.ManifestFormat = val
	case "format":
		if val != "json" && val != "yaml" {
			return usageErr{"unknown format value: " + val}
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
			return usageErr{"unknown value for on-unreadable: " + val}
		}
		cfg.OnUnreadable = val
	case "scope":
		if val != "etc" && val != "full" {
			return usageErr{"unknown value for scope: " + val}
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
			return usageErr{"unknown value for signature-verification: " + val}
		}
		cfg.SignatureVerification = val
	case "keyring":
		cfg.Keyring = val
	case "activation-policy":
		switch val {
		case "reboot", "soft-reboot", "none":
			cfg.ActivationPolicy = val
		default:
			return usageErr{"unknown value for activation-policy: " + val}
		}
	case "applied-root":
		cfg.AppliedRoot = val
	default:
		return usageErr{"unknown option: " + key}
	}
	return nil
}

// manifestFormatDefault returns the resolved manifest.Format default.
func (c Config) manifestFormatDefault() manifest.Format {
	if c.ManifestFormat == "yaml" {
		return manifest.FormatYAML
	}
	return manifest.FormatJSON
}

// printUsage writes the usage text to w.
func printUsage(w io.Writer) {
	fmt.Fprint(w, usageText)
}

const usageText = `usage: zypper-declarative <verb> [key=value ...]

Verbs:
  apply      converge the system to the desired manifest in a transaction
  diff       dry run: print what apply would change (no modification)
  verify     check actual state against a reference declaration
  status     print the current declarative state
  describe   read actual state and emit it as a manifest document

Global:
  version    print program name, version, and embedded spec hash
  help       print this usage

Key=value options (precede any bare-word argument):
  mode=auto|external|internal       transaction binding; default auto
  manifest-path=<path>              desired manifest (apply, diff); reference for verify
  format=json|yaml                  serialisation for this invocation's manifest I/O
  state-path=<path>                 captured actual state for verify and diff (offline)
  root=<path>                       root to describe; default "/"
  out=<path>                        describe output file; default stdout
  on-unreadable=error|warn          describe: fail (default) or omit+warn
  scope=etc|full                    describe/verify read scope; etc default, full audits /usr,/boot
  manifest-format, repo-lock, content-store, keep-list,
  signature-verification, keyring, activation-policy, applied-root

Exit codes: 0 success, 1 logical failure, 2 invocation error.
`
