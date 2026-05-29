// generated from spec: zypper-declarative.spec.md sha256:f8ff76ecbc4bbc69a49e2e32b2924da3a64df1ad46196e05ce8c137b684429b2
//
// Package meta holds the build-time identity of the binary: its version (from
// the spec META Version field) and the SHA256 of the specification it was
// translated from. Both are embedded so that `version` output is
// cryptographically tied to its source of truth.
package meta

const (
	// ProgramName is the canonical binary name.
	ProgramName = "zypper-declarative"

	// Version is the spec META Version of zypper-declarative.spec.md.
	Version = "0.5.1"

	// SpecSHA256 is the SHA256 of zypper-declarative.spec.md as translated.
	SpecSHA256 = "f8ff76ecbc4bbc69a49e2e32b2924da3a64df1ad46196e05ce8c137b684429b2"
)

// Generator returns the generator string embedded in produced manifests, e.g.
// "zypper-declarative 0.5.1".
func Generator() string {
	return ProgramName + " " + Version
}

// VersionLine returns the single-line version string printed by the `version`
// verb and the `--version` alias: program name, version, and the embedded spec
// hash.
func VersionLine() string {
	return ProgramName + " " + Version + " spec:" + SpecSHA256
}
