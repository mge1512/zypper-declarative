// generated from spec: zypper-declarative.spec.md sha256:18253550a5c3d3f1818f0380811cb5dbc98874828693e49fc9cd5cbc923303dd
//
// Build-time identity. The spec SHA256 is embedded as a compile-time constant
// so that `version` output, the source headers, and the audit trail all carry
// the cryptographic identity of the specification this binary was produced from.

/// Program name (also the binary name and the crate name).
pub const PROGRAM: &str = "zypper-declarative";

/// Program version. Tracks the spec META Version field (0.6.4).
pub const VERSION: &str = "0.6.4";

/// SHA256 of the specification this binary was produced from. There are no
/// `Includes:` directives in the host spec, so the merged hash equals the host
/// hash.
pub const SPEC_SHA256: &str =
    "18253550a5c3d3f1818f0380811cb5dbc98874828693e49fc9cd5cbc923303dd";

/// The generator string embedded in every emitted manifest's meta.generator.
/// It carries the program name AND version so independent implementations of
/// the same spec version emit the same string (spec INVARIANT).
pub fn generator() -> String {
    format!("{} {}", PROGRAM, VERSION)
}
