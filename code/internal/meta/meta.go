// generated from spec: zypper-declarative.spec.md sha256:714e75ff672557d2c7344736a5f36b52afa37e0f565b07de83e1b18cc4492014
//
// Package meta carries build-time provenance: the component version and the
// SHA256 of the specification this binary was generated from. The values are
// embedded as constants so that `--version` and the generator metadata can
// report them without external state.
package meta

// Version is the component version, tracking the spec META Version field.
const Version = "0.4.0"

// SpecSHA256 is the SHA256 of zypper-declarative.spec.md as provided to the
// translator. It is embedded in every artifact for cryptographic provenance.
const SpecSHA256 = "714e75ff672557d2c7344736a5f36b52afa37e0f565b07de83e1b18cc4492014"

// Generator is the generator identity stamped into produced manifests.
const Generator = "zypper-declarative " + Version
