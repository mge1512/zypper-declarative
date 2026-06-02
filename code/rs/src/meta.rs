// generated from spec: zypper-declarative.spec.md
// sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
//
// Embedded build identity: the tool version and the SHA256 of the
// specification this implementation was generated from. Injected by build.rs.

/// Program name as it appears in `meta.generator` and version output.
pub const PROGRAM_NAME: &str = "zypper-declarative";

/// Semantic version of the tool (== spec Version).
pub const VERSION: &str = env!("ZD_VERSION");

/// SHA256 of the specification this binary was generated from.
pub const SPEC_SHA256: &str = env!("ZD_SPEC_SHA256");

/// The `meta.generator` string: "zypper-declarative <version>".
/// Independent implementations of the same spec version emit the same value.
pub fn generator() -> String {
    format!("{} {}", PROGRAM_NAME, VERSION)
}

/// The `version` verb / `--version` output line.
pub fn version_line() -> String {
    format!("{} {} spec:{}", PROGRAM_NAME, VERSION, SPEC_SHA256)
}
