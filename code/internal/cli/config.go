// generated from spec: zypper-declarative.spec.md sha256:714e75ff672557d2c7344736a5f36b52afa37e0f565b07de83e1b18cc4492014
//
// Package cli holds the verb dispatch, key=value argument parsing, the global
// invocation contract, and the exit-code mapping. Exit-code mapping lives only
// here; internal behaviours return errors to their caller.
package cli

import (
	"github.com/mge1512/zypper-declarative/internal/manifest"
	"github.com/mge1512/zypper-declarative/internal/txn"
)

// Config carries the resolved CONFIG knobs. All knobs are surfaced via
// key=value options (CONFIG: control via environment variables is forbidden).
type Config struct {
	TransactionMode       txn.Mode
	ManifestPath          string
	ManifestFormat        manifest.Format
	RepoLock              string
	ContentStore          string
	KeepListPath          string
	SignatureVerification bool
	Keyring               string
	ActivationPolicy      string

	// AppliedRoot is the generation root the applied record is read from /
	// written to ("/" for the running system). Surfaced as applied-root so
	// read-only verbs are testable against an arbitrary root.
	AppliedRoot string

	// Per-invocation options (not persistent CONFIG):
	Format    string // explicit format= option ("" if unset)
	StatePath string // verify state-path
	Root      string // describe root (default "/")
	Out       string // describe out (default stdout)
	Mode      string // explicit mode= option ("" if unset)
}

// defaultConfig returns the template/CONFIG defaults.
func defaultConfig() Config {
	return Config{
		TransactionMode:       txn.ModeAuto,
		ManifestPath:          "/var/lib/zypper-declarative/desired.json",
		ManifestFormat:        manifest.FormatJSON,
		SignatureVerification: true,
		ActivationPolicy:      "reboot",
		AppliedRoot:           "/",
		Root:                  "/",
	}
}
