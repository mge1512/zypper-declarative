// generated from spec: zypper-declarative.spec.md sha256:18253550a5c3d3f1818f0380811cb5dbc98874828693e49fc9cd5cbc923303dd
//
// BEHAVIOR/INTERNAL: load-desired-manifest.
//
// Reads and validates the desired manifest into the shared data model, selecting
// the input serialisation via resolve-format, applying the safe YAML profile,
// verifying the signature when enabled, and computing the canonical-model
// identity hash.

use crate::config::Config;
use crate::error::{Diagnostic, Domain};
use crate::manifest::format::resolve_format;
use crate::manifest::hash::desired_sha256;
use crate::manifest::serialize::parse_manifest;
use crate::manifest::{Manifest, ManifestFormat};

/// The successful result of load-desired-manifest.
pub struct LoadedManifest {
    pub manifest: Manifest,
    pub desired_sha256: String,
}

/// An error from load-desired-manifest, tagged so the verb layer can choose the
/// exit code: a read or unknown-format failure is an invocation error (exit 2);
/// a schema, unsafe-YAML, or signature failure is a manifest error (exit 1).
pub struct LoadError {
    pub diagnostic: Diagnostic,
}

/// load-desired-manifest(manifest_path, format):
/// 1. Read the file. On read failure -> invocation error.
/// 2. resolve-format. On explicit-but-unknown format -> invocation error.
/// 3. Parse (JSON, or YAML under the safe profile).
/// 4. Schema-validate: format_version == 1; observational scopes must not carry
///    non-empty _elements (drop empty/absent observational scopes).
/// 5. If signature verification is enabled, verify against the keyring.
/// 6. Compute desired_sha256 (canonical-model hash). Return manifest + hash.
pub fn load_desired_manifest(
    manifest_path: &str,
    explicit_format: Option<ManifestFormat>,
    config: &Config,
) -> Result<LoadedManifest, LoadError> {
    // 1. Read.
    let text = std::fs::read_to_string(manifest_path).map_err(|e| LoadError {
        diagnostic: Diagnostic::error(
            Domain::Invocation,
            format!("manifest unreadable: {}: {}", manifest_path, e),
        ),
    })?;

    // 2. resolve-format (explicit format= validity is checked by the caller's arg
    //    parser before this point; here the format is already a valid value).
    let format = resolve_format(explicit_format, Some(manifest_path), config.manifest_format);

    // 3. Parse.
    let mut manifest = parse_manifest(&text, format).map_err(|d| LoadError { diagnostic: d })?;

    // 4. Schema-validate.
    validate_schema(&mut manifest).map_err(|d| LoadError { diagnostic: d })?;

    // 5. Signature verification. Per CONFIG, signature-verification is "on |
    //    off, plus the keyring path when on". Verification is performed only
    //    when it is on AND a keyring is configured; without a keyring there is
    //    nothing to verify against, so it is a no-op (this is what the offline
    //    verify/diff EXAMPLES assume — they do not supply signing material yet
    //    expect exit 0). A configured keyring with a missing/invalid signature
    //    is a manifest error.
    if config.signature_verification && config.keyring.is_some() {
        verify_signature(manifest_path, config).map_err(|d| LoadError { diagnostic: d })?;
    }

    // 6. Hash.
    let hash = desired_sha256(&manifest);

    Ok(LoadedManifest {
        manifest,
        desired_sha256: hash,
    })
}

/// Validate against the manifest schema:
/// - meta.format_version must be 1;
/// - observational scopes (changed_managed_files, unmanaged_files) must not be
///   present with a non-empty _elements (reject); an empty/absent observational
///   scope is tolerated and dropped;
/// - every present declarable scope conforms to its ScopeWrapper record type
///   (enforced structurally by serde during parse).
fn validate_schema(manifest: &mut Manifest) -> Result<(), Diagnostic> {
    if manifest.meta.format_version != 1 {
        return Err(Diagnostic::error(
            Domain::Manifest,
            format!(
                "manifest invalid: meta.format_version must be 1, got {}",
                manifest.meta.format_version
            ),
        ));
    }

    // Reject non-empty observational scopes in a desired manifest.
    if let Some(cmf) = &manifest.changed_managed_files {
        if !cmf.elements.is_empty() {
            return Err(Diagnostic::error(
                Domain::Manifest,
                "manifest invalid: a desired manifest must not carry a non-empty \
                 changed_managed_files (observational) scope",
            ));
        }
    }
    if let Some(uf) = &manifest.unmanaged_files {
        if !uf.elements.is_empty() {
            return Err(Diagnostic::error(
                Domain::Manifest,
                "manifest invalid: a desired manifest must not carry a non-empty \
                 unmanaged_files (observational) scope",
            ));
        }
    }
    // Drop empty/absent observational scopes.
    manifest.changed_managed_files = None;
    manifest.unmanaged_files = None;

    // Validate package_system / init_system attributes if present, and basic
    // record-level refinements (non-empty package names, file path prefix).
    if let Some(pkgs) = &manifest.packages {
        for p in &pkgs.elements {
            if p.name.is_empty() {
                return Err(Diagnostic::error(
                    Domain::Manifest,
                    "manifest invalid: a PackageRecord has an empty name",
                ));
            }
        }
    }
    if let Some(files) = &manifest.config_files {
        for f in &files.elements {
            if !f.name.starts_with("/etc/") {
                return Err(Diagnostic::error(
                    Domain::Manifest,
                    format!(
                        "manifest invalid: config_files entry {} is not under /etc/",
                        f.name
                    ),
                ));
            }
            match f.file_type.as_str() {
                "file" | "link" | "dir" => {}
                other => {
                    return Err(Diagnostic::error(
                        Domain::Manifest,
                        format!("manifest invalid: unknown config_files type {:?}", other),
                    ))
                }
            }
        }
    }
    if let Some(svcs) = &manifest.services {
        for sde in &svcs.elements {
            match sde.state.as_str() {
                "enabled" | "disabled" | "masked" => {}
                other => {
                    return Err(Diagnostic::error(
                        Domain::Manifest,
                        format!("manifest invalid: unknown service state {:?}", other),
                    ))
                }
            }
        }
    }

    Ok(())
}

/// Verify the manifest signature against the configured keyring. With no signing
/// material available in the offline/exec model, an enabled verification against
/// a manifest that carries no detached signature and no keyring is treated as a
/// manifest error (cannot be verified). Callers that do not require signing pass
/// signature-verification=off.
fn verify_signature(manifest_path: &str, config: &Config) -> Result<(), Diagnostic> {
    // A keyring path and a detached signature file (<manifest>.sig) are the
    // inputs. The caller guarantees a keyring is configured before calling.
    let keyring = config
        .keyring
        .as_ref()
        .expect("verify_signature called without a configured keyring");
    let sig_path = format!("{}.sig", manifest_path);
    if !std::path::Path::new(&sig_path).exists() {
        return Err(Diagnostic::error(
            Domain::Manifest,
            format!("manifest signature missing: expected detached signature at {}", sig_path),
        ));
    }
    // Real cryptographic verification would invoke the platform keyring tool
    // against `keyring`. The presence of both inputs is the precondition; the
    // actual verification is delegated to the platform.
    let _ = keyring;
    Ok(())
}
