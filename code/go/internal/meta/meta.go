// generated from spec: zypper-declarative.spec.md sha256:27aee8e374eb3507189bad0b78339109d0116e6a13b55ae4df2ba9a18e769fc4
//
// Package meta carries the program identity: name, version, and the embedded
// SHA256 of the specification this binary was generated from. The spec hash is
// surfaced in `version`/`--version` output per the spec's provenance invariant.
package meta

const (
	// ProgramName is the canonical program name (and zypper subcommand verb).
	ProgramName = "zypper-declarative"

	// Version is the spec Version (META) the binary implements.
	Version = "0.6.5"

	// SpecSHA256 is the SHA256 of zypper-declarative.spec.md as provided as
	// input. Embedded here per the spec-hash invariant.
	SpecSHA256 = "27aee8e374eb3507189bad0b78339109d0116e6a13b55ae4df2ba9a18e769fc4"
)

// Generator returns the meta.generator string for emitted manifests, e.g.
// "zypper-declarative 0.6.5". Independent implementations of the same spec
// version emit the same string.
func Generator() string {
	return ProgramName + " " + Version
}

// VersionLine returns the single-line version banner printed by the version
// verb and its --version alias, e.g.
// "zypper-declarative 0.6.5 spec:<sha256>".
func VersionLine() string {
	return ProgramName + " " + Version + " spec:" + SpecSHA256
}
