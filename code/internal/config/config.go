// generated from spec: zypper-declarative.spec.md sha256:58e1636e2de82ab81a5cd3f81d6b3c9ac6a8976e18f9abb2bbd2b2aba56fe4d4
//
// Package config holds the resolved CONFIG knobs. All knobs are surfaced via
// key=value arguments (and, in a full deployment, preset files). Control via
// environment variables is forbidden. A command-line option overrides the
// corresponding preset value.
package config

import "github.com/mge1512/zypper-declarative/internal/manifest"

// OnUnreadable controls how an unreadable scope source is treated.
type OnUnreadable string

const (
	OnUnreadableError OnUnreadable = "error"
	OnUnreadableWarn  OnUnreadable = "warn"
)

// Config is the resolved set of CONFIG knobs for one invocation.
type Config struct {
	TransactionMode       string            // auto | external | internal
	ManifestPath          string            // desired manifest path
	ManifestFormat        manifest.Format   // fallback serialisation (json|yaml)
	ManifestFormatGiven   bool              // whether an explicit format= was supplied
	ExplicitFormat        manifest.Format   // the explicit format= value, if given
	ExplicitFormatGiven   bool              // whether format= was supplied for this invocation
	OnUnreadable          OnUnreadable      // describe / describe-actual-state policy
	RepoLock              string            // fallback pinned repo when no repositories scope
	ContentStore          string            // base path for content_ref resolution
	KeepList              string            // allowlist path
	SignatureVerification bool              // verify manifest signatures
	Keyring               string            // keyring path when verification on
	ActivationPolicy      string            // reboot | soft-reboot | none
	AppliedRoot           string            // generation root for load-applied-record
	StatePath             string            // verify: state dump path
	Root                  string            // describe: root to describe
	Out                   string            // describe: output file
	Extra                 map[string]string // any other accepted CONFIG knobs
}

// Defaults returns the CONFIG default set per the spec CONFIG section.
func Defaults() Config {
	return Config{
		TransactionMode:       "auto",
		ManifestPath:          "/var/lib/zypper-declarative/desired.json",
		ManifestFormat:        manifest.FormatJSON,
		OnUnreadable:          OnUnreadableError,
		SignatureVerification: true,
		ActivationPolicy:      "reboot",
		AppliedRoot:           "/",
		Root:                  "/",
		Extra:                 map[string]string{},
	}
}
