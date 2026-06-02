// generated from spec: zypper-declarative.spec.md sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
//
// Package meta carries the program version and the embedded spec SHA256.
package meta

// Version is the program version, matching the spec META Version field.
// The same spec version yields the same generator string across implementations.
const Version = "0.6.6"

// ProgramName is the canonical program name used in version and generator strings.
const ProgramName = "zypper-declarative"

// SpecSHA256 is the SHA256 of the specification this binary was generated from.
// It is embedded per the spec-hash invariant and surfaced in the version output.
const SpecSHA256 = "51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03"

// Generator returns the meta.generator string: program name and version.
func Generator() string {
	return ProgramName + " " + Version
}
