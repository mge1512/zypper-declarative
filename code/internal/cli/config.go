// generated from spec: zypper-declarative.spec.md sha256:f8ff76ecbc4bbc69a49e2e32b2924da3a64df1ad46196e05ce8c137b684429b2
//
// Argument and configuration parsing. Options are key=value pairs (POSIX --flag
// style is not used except the tolerated version/help aliases handled by the
// dispatcher); bare words are verbs. Control via environment variables is
// forbidden. All CONFIG knobs are also accepted as key=value options.

package cli

import (
	"os"
	"strings"

	"github.com/mge1512/zypper-declarative/internal/manifest"
	"github.com/mge1512/zypper-declarative/internal/state"
	"github.com/mge1512/zypper-declarative/internal/txn"
)

// Config is the resolved invocation configuration after applying defaults and
// the parsed key=value options.
type Config struct {
	Mode                  txn.Mode
	ManifestPath          string
	ManifestFormat        manifest.Format
	ExplicitFormat        manifest.Format
	ExplicitFormatGiven   bool
	StatePath             string
	Root                  string
	Out                   string
	OnUnreadable          state.OnUnreadable
	RepoLock              string
	ContentStore          string
	KeepListPath          string
	SignatureVerification bool
	KeyringPath           string
	ActivationPolicy      string
	AppliedRoot           string
}

// defaultConfig returns the CONFIG defaults.
func defaultConfig() Config {
	return Config{
		Mode:                  txn.ModeAuto,
		ManifestPath:          defaultManifestPath,
		ManifestFormat:        manifest.FormatJSON,
		OnUnreadable:          state.OnUnreadableError,
		SignatureVerification: true,
		ActivationPolicy:      "reboot",
		AppliedRoot:           "/",
		Root:                  "/",
	}
}

// defaultManifestPath is the fixed staging path supplied by the delivery layer.
const defaultManifestPath = "/var/lib/zypper-declarative/desired.json"

// parseError reports a bad option, value, or missing required value.
type parseError struct{ msg string }

func (e *parseError) Error() string { return e.msg }

func parseErr(msg string) *parseError { return &parseError{msg: msg} }

// parseOptions consumes the leading key=value options from args and returns the
// resolved Config and the remaining (bare-word) arguments. A POSIX --flag (other
// than the tolerated aliases the dispatcher already handled) or an unknown
// key/value is a parse error.
func parseOptions(args []string) (Config, []string, *parseError) {
	cfg := defaultConfig()
	var rest []string
	parsingOptions := true

	for _, arg := range args {
		if parsingOptions && strings.Contains(arg, "=") && !strings.HasPrefix(arg, "-") {
			key, val := splitKV(arg)
			if err := applyOption(&cfg, key, val); err != nil {
				return cfg, nil, err
			}
			continue
		}
		// First non-option token ends option parsing; the spec states options
		// precede bare-word arguments.
		parsingOptions = false
		if strings.HasPrefix(arg, "-") {
			return cfg, nil, parseErr("unknown option " + arg + " (options are key=value; POSIX --flags are not used)")
		}
		rest = append(rest, arg)
	}
	return cfg, rest, nil
}

func splitKV(arg string) (string, string) {
	i := strings.IndexByte(arg, '=')
	return arg[:i], arg[i+1:]
}

func applyOption(cfg *Config, key, val string) *parseError {
	switch key {
	case "mode", "transaction-mode":
		switch val {
		case "auto", "external", "internal":
			cfg.Mode = txn.Mode(val)
		default:
			return parseErr("unknown value for " + key + ": " + val)
		}
	case "manifest-path":
		cfg.ManifestPath = val
	case "format":
		f, given, err := manifest.ParseFormat(val)
		if err != nil {
			return parseErr(err.Error())
		}
		cfg.ExplicitFormat = f
		cfg.ExplicitFormatGiven = given
	case "manifest-format":
		switch val {
		case "json":
			cfg.ManifestFormat = manifest.FormatJSON
		case "yaml":
			cfg.ManifestFormat = manifest.FormatYAML
		default:
			return parseErr("unknown value for manifest-format: " + val)
		}
	case "state-path":
		cfg.StatePath = val
	case "root":
		cfg.Root = val
	case "out":
		cfg.Out = val
	case "on-unreadable":
		switch val {
		case "error", "warn":
			cfg.OnUnreadable = state.OnUnreadable(val)
		default:
			return parseErr("unknown value for on-unreadable: " + val)
		}
	case "repo-lock":
		cfg.RepoLock = val
	case "content-store":
		cfg.ContentStore = val
	case "keep-list":
		cfg.KeepListPath = val
	case "signature-verification":
		switch val {
		case "on":
			cfg.SignatureVerification = true
		case "off":
			cfg.SignatureVerification = false
		default:
			return parseErr("unknown value for signature-verification: " + val)
		}
	case "keyring":
		cfg.KeyringPath = val
	case "activation-policy":
		switch val {
		case "reboot", "soft-reboot", "none":
			cfg.ActivationPolicy = val
		default:
			return parseErr("unknown value for activation-policy: " + val)
		}
	case "applied-root":
		cfg.AppliedRoot = val
	default:
		return parseErr("unknown option: " + key)
	}
	return nil
}

// loadKeepList reads the keep-list allowlist (one path per line). A missing
// keep-list path yields an empty allowlist (the keep-list is optional).
func loadKeepList(path string) map[string]bool {
	set := map[string]bool{}
	if path == "" {
		return set
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return set
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		set[line] = true
	}
	return set
}
