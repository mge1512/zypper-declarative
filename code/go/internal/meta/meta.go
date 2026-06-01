// generated from spec: zypper-declarative.spec.md sha256:f2cc80627e483a48bb8411d297711bc5f6c6e74c28dbf0dafc8fe7bd8817251e
//
// Package meta holds the component version and the SHA256 of the specification
// this binary was generated from. The spec hash is embedded so that the version
// boundary between two builds is cryptographically verifiable.
package meta

const (
	// ProgramName is the canonical binary name.
	ProgramName = "zypper-declarative"

	// Version is the spec META Version this implementation targets.
	Version = "0.6.2"

	// SpecSHA256 is the SHA256 of zypper-declarative.spec.md as provided as
	// input to the translator. It is embedded in the --version output.
	SpecSHA256 = "f2cc80627e483a48bb8411d297711bc5f6c6e74c28dbf0dafc8fe7bd8817251e"
)

// VersionLine returns the canonical version string printed by `version`.
func VersionLine() string {
	return ProgramName + " " + Version + " spec:" + SpecSHA256
}
