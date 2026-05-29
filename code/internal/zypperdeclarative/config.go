// generated from spec: zypper-declarative.spec.md sha256:714e75ff672557d2c7344736a5f36b52afa37e0f565b07de83e1b18cc4492014
//
// CONFIG resolution from key=value arguments. Per the cli-tool template,
// control via environment variables is forbidden; all knobs are key=value
// arguments (or preset files, not implemented in v1). This file holds the
// effective configuration and its defaults.
package zypperdeclarative

// Config holds the resolved CONFIG knobs and per-invocation options.
type Config struct {
	// CONFIG knobs (spec CONFIG section).
	TransactionMode       TransactionMode
	ManifestPath          string
	ManifestFormat        ManifestFormat
	RepoLock              string
	ContentStore          string
	KeepListPath          string
	SignatureVerification bool
	KeyringPath           string
	ActivationPolicy      string

	// Per-invocation options (spec DEPLOYMENT key=value table).
	Format    ManifestFormat // explicit format override for load/describe; "" = unset
	FormatSet bool
	StatePath string
	Root      string // describe root / actual-state root; default "/"
	Out       string // describe output file; default stdout

	// AppliedRoot is the generation root from which load-applied-record reads
	// (spec load-applied-record takes a root input). Surfaced as a key=value
	// option for offline operation and testability; default "/".
	AppliedRoot string
}

// defaultConfig returns the CONFIG defaults from the spec CONFIG section.
func defaultConfig() Config {
	return Config{
		TransactionMode:       ModeAuto,
		ManifestPath:          "/var/lib/zypper-declarative/desired.json",
		ManifestFormat:        FormatJSON,
		SignatureVerification: true,
		ActivationPolicy:      "reboot",
		Root:                  "/",
		AppliedRoot:           "/",
	}
}
