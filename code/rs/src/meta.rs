// generated from spec: zypper-declarative.spec.md sha256:87b9f2f3bf92afd6fe037412d56a23959290e5816d722a3def9505d07aa5acd7
//
// Embedded build identity: the program name, the spec Version, and the SHA256 of
// the specification this binary was generated from. The spec hash is computed once
// (sha256sum zypper-declarative.spec.md) and embedded here so the version boundary
// is cryptographically verifiable per the spec-hash-embedding invariant.

/// Program name, matching the spec title (lowercase-hyphenated) and the binary name.
pub const PROGRAM_NAME: &str = "zypper-declarative";

/// Spec Version (META: Version: 0.6.3). Used in meta.generator and --version output.
pub const VERSION: &str = "0.6.3";

/// SHA256 of zypper-declarative.spec.md as provided as input.
pub const SPEC_SHA256: &str = "87b9f2f3bf92afd6fe037412d56a23959290e5816d722a3def9505d07aa5acd7";

/// The generator string embedded in every emitted manifest's meta.generator.
/// Always carries the version so independent implementations of the same spec
/// version emit the same generator string.
pub fn generator() -> String {
    format!("{} {}", PROGRAM_NAME, VERSION)
}

/// The version line printed by `version` / `--version`. Includes the spec hash.
pub fn version_line() -> String {
    format!("{} {} spec:{}", PROGRAM_NAME, VERSION, SPEC_SHA256)
}
