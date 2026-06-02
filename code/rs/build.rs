// build.rs — injects the spec SHA256 and tool version as compile-time env vars
// so meta.rs can embed them without a placeholder.
//
// generated from spec: zypper-declarative.spec.md
// sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03

fn main() {
    // The spec SHA256 is a fixed, computed value for this spec revision. It is
    // recorded here (not read from a file at build time) so the built binary
    // carries it even when the spec file is not present in the build sandbox.
    let spec_sha256 = "51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03";
    println!("cargo:rustc-env=ZD_SPEC_SHA256={}", spec_sha256);
    let version = std::env::var("CARGO_PKG_VERSION").unwrap_or_else(|_| "0.0.0".to_string());
    println!("cargo:rustc-env=ZD_VERSION={}", version);
    println!("cargo:rerun-if-changed=build.rs");
}
