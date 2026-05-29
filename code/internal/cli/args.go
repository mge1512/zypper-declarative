// generated from spec: zypper-declarative.spec.md sha256:b2d0de88fbed1163678e59e931c741b9d999b71f902f6eb01db8790bb813d057
//
// Argument parsing and configuration resolution for the CLI. Options are
// key=value (parsed by the tool itself); bare words are verbs and must precede
// no further parsing logic here (the dispatcher handles ordering). Control via
// environment variables is forbidden.
package cli

import (
	"strings"

	"github.com/mge1512/zypper-declarative/internal/manifest"
)

// Config holds the resolved knobs for one invocation. Every CONFIG knob is
// also accepted as a key=value option; a command-line option overrides the
// corresponding preset value.
type Config struct {
	Mode                  string // transaction-mode: auto|external|internal
	ManifestPath          string
	ManifestFormat        manifest.Format // json|yaml default
	ExplicitFormat        string          // the format= option, "" if unset
	StatePath             string
	Root                  string
	Out                   string
	OnUnreadable          string // error|warn
	Scope                 string // etc|full
	RepoLock              string
	ContentStore          string
	KeepList              string
	SignatureVerification bool
	Keyring               string
	ActivationPolicy      string
	AppliedRoot           string
}

// defaultConfig returns the CONFIG defaults.
func defaultConfig() Config {
	return Config{
		Mode:                  "auto",
		ManifestPath:          "/var/lib/zypper-declarative/desired.json",
		ManifestFormat:        manifest.FormatJSON,
		OnUnreadable:          "error",
		Scope:                 "etc",
		SignatureVerification: true,
		ActivationPolicy:      "reboot",
		AppliedRoot:           "/",
		Root:                  "/",
	}
}

// knownOptions is the set of accepted key=value option keys.
var knownOptions = map[string]bool{
	"mode": true, "transaction-mode": true, "manifest-path": true,
	"manifest-format": true, "format": true, "state-path": true,
	"root": true, "out": true, "on-unreadable": true, "scope": true,
	"repo-lock": true, "content-store": true, "keep-list": true,
	"signature-verification": true, "keyring": true,
	"activation-policy": true, "applied-root": true,
}

// parsed is the outcome of splitting argv into options and bare words.
type parsed struct {
	options   map[string]string
	bareWords []string
}

// parseArgs splits argv into key=value options and bare words. An option is a
// token of the form key=value; anything else is a bare word. A malformed
// option (unknown key) is reported by validate, not here.
func parseArgs(args []string) parsed {
	p := parsed{options: map[string]string{}}
	for _, a := range args {
		if i := strings.IndexByte(a, '='); i > 0 && !strings.HasPrefix(a, "-") {
			p.options[a[:i]] = a[i+1:]
			continue
		}
		p.bareWords = append(p.bareWords, a)
	}
	return p
}

// applyOptions folds parsed options onto a Config, returning an invocation
// diagnostic on an unknown key or an unknown value.
func applyOptions(cfg *Config, opts map[string]string) *manifest.Diagnostic {
	for k, v := range opts {
		if !knownOptions[k] {
			return manifest.NewError(manifest.DomainInvocation, "unknown option: "+k)
		}
		switch k {
		case "mode", "transaction-mode":
			switch v {
			case "auto", "external", "internal":
				cfg.Mode = v
			default:
				return manifest.NewError(manifest.DomainInvocation, "unknown value for "+k+": "+v)
			}
		case "manifest-path":
			cfg.ManifestPath = v
		case "manifest-format":
			switch v {
			case "json":
				cfg.ManifestFormat = manifest.FormatJSON
			case "yaml":
				cfg.ManifestFormat = manifest.FormatYAML
			default:
				return manifest.NewError(manifest.DomainInvocation, "unknown value for manifest-format: "+v)
			}
		case "format":
			switch v {
			case "json", "yaml":
				cfg.ExplicitFormat = v
			default:
				return manifest.NewError(manifest.DomainInvocation, "unknown format value: "+v)
			}
		case "state-path":
			cfg.StatePath = v
		case "root":
			cfg.Root = v
		case "out":
			cfg.Out = v
		case "on-unreadable":
			switch v {
			case "error", "warn":
				cfg.OnUnreadable = v
			default:
				return manifest.NewError(manifest.DomainInvocation, "unknown value for on-unreadable: "+v)
			}
		case "scope":
			switch v {
			case "etc", "full":
				cfg.Scope = v
			default:
				return manifest.NewError(manifest.DomainInvocation, "unknown value for scope: "+v)
			}
		case "repo-lock":
			cfg.RepoLock = v
		case "content-store":
			cfg.ContentStore = v
		case "keep-list":
			cfg.KeepList = v
		case "signature-verification":
			switch v {
			case "on":
				cfg.SignatureVerification = true
			case "off":
				cfg.SignatureVerification = false
			default:
				return manifest.NewError(manifest.DomainInvocation, "unknown value for signature-verification: "+v)
			}
		case "keyring":
			cfg.Keyring = v
		case "activation-policy":
			switch v {
			case "reboot", "soft-reboot", "none":
				cfg.ActivationPolicy = v
			default:
				return manifest.NewError(manifest.DomainInvocation, "unknown value for activation-policy: "+v)
			}
		case "applied-root":
			cfg.AppliedRoot = v
		}
	}
	return nil
}
