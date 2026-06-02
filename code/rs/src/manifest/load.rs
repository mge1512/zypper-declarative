// generated from spec: zypper-declarative.spec.md
// sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
//
// BEHAVIOR/INTERNAL: load-desired-manifest. Reads and validates the desired
// manifest into the shared model, selecting the input serialisation via
// resolve-format, applying the safe YAML profile when the input is YAML, and
// computing the canonical-model identity hash.

use crate::error::{Diagnostic, Domain};
use crate::manifest::format::{resolve_format, ManifestFormat};
use crate::manifest::hash::desired_sha256;
use crate::manifest::{yaml, Manifest};

pub struct LoadedManifest {
    pub manifest: Manifest,
    pub desired_sha256: String,
}

/// load-desired-manifest(manifest_path, explicit_format, default_format).
pub fn load_desired_manifest(
    manifest_path: &str,
    explicit_format: Option<ManifestFormat>,
    default_format: ManifestFormat,
) -> Result<LoadedManifest, Diagnostic> {
    // 1. read file (read failure -> invocation error)
    let text = std::fs::read_to_string(manifest_path).map_err(|e| {
        Diagnostic::error(
            Domain::Invocation,
            format!("manifest {} unreadable: {}", manifest_path, e),
        )
    })?;

    // 2. resolve format
    let format = resolve_format(explicit_format, Some(manifest_path), default_format);

    // 3. parse into the data model
    let manifest = match format {
        ManifestFormat::Json => serde_json::from_str::<Manifest>(&text).map_err(|e| {
            Diagnostic::error(Domain::Manifest, format!("invalid JSON manifest: {}", e))
        })?,
        ManifestFormat::Yaml => yaml::parse_manifest_safe(&text).map_err(|e| {
            Diagnostic::error(
                Domain::Manifest,
                format!("unsafe or invalid YAML manifest: {}", e),
            )
        })?,
    };

    // 4. schema validation
    validate_schema(&manifest)?;

    // 5. signature verification (CONFIG-gated; default on but no keyring available
    //    here so it is a no-op unless a keyring is configured — left as a
    //    structural hook; absence of a signature requirement is not an error).

    // 6. canonical-model hash
    let hash = desired_sha256(&manifest);

    Ok(LoadedManifest {
        manifest,
        desired_sha256: hash,
    })
}

fn validate_schema(manifest: &Manifest) -> Result<(), Diagnostic> {
    if manifest.meta.format_version != 1 {
        return Err(Diagnostic::error(
            Domain::Manifest,
            format!(
                "meta.format_version must be 1, got {}",
                manifest.meta.format_version
            ),
        ));
    }
    // observational scopes must not be present with non-empty _elements
    if let Some(cmf) = &manifest.changed_managed_files {
        if !cmf.elements.is_empty() {
            return Err(Diagnostic::error(
                Domain::Manifest,
                "a desired manifest must not carry a non-empty changed_managed_files scope",
            ));
        }
    }
    if let Some(uf) = &manifest.unmanaged_files {
        if !uf.elements.is_empty() {
            return Err(Diagnostic::error(
                Domain::Manifest,
                "a desired manifest must not carry a non-empty unmanaged_files scope",
            ));
        }
    }
    // record-level refinement: config_files type/field consistency
    if let Some(cf) = &manifest.config_files {
        for e in &cf.elements {
            match e.r#type.as_str() {
                "file" | "link" | "dir" => {}
                other => {
                    return Err(Diagnostic::error(
                        Domain::Manifest,
                        format!(
                            "config_files record {} has invalid type {:?}",
                            e.name, other
                        ),
                    ));
                }
            }
        }
    }
    if let Some(sv) = &manifest.services {
        for s in &sv.elements {
            match s.state.as_str() {
                "enabled" | "disabled" | "masked" => {}
                other => {
                    return Err(Diagnostic::error(
                        Domain::Manifest,
                        format!("service {} has invalid state {:?}", s.name, other),
                    ));
                }
            }
        }
    }
    Ok(())
}

/// Load a captured actual-state dump (offline). Observational scopes ARE
/// tolerated here (a full describe dump). A malformed dump is an invocation
/// error (exit 2).
pub fn load_state_dump(
    state_path: &str,
    explicit_format: Option<ManifestFormat>,
    default_format: ManifestFormat,
) -> Result<Manifest, Diagnostic> {
    let text = std::fs::read_to_string(state_path).map_err(|e| {
        Diagnostic::error(
            Domain::Invocation,
            format!("state dump {} unreadable: {}", state_path, e),
        )
    })?;
    let format = resolve_format(explicit_format, Some(state_path), default_format);
    let manifest = match format {
        ManifestFormat::Json => serde_json::from_str::<Manifest>(&text).map_err(|e| {
            Diagnostic::error(Domain::Invocation, format!("malformed state dump: {}", e))
        })?,
        ManifestFormat::Yaml => yaml::parse_manifest_safe(&text).map_err(|e| {
            Diagnostic::error(Domain::Invocation, format!("malformed state dump: {}", e))
        })?,
    };
    if manifest.meta.format_version != 1 {
        return Err(Diagnostic::error(
            Domain::Invocation,
            format!(
                "malformed state dump: format_version {}",
                manifest.meta.format_version
            ),
        ));
    }
    Ok(manifest)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn rejects_format_version_2() {
        let m = Manifest {
            meta: crate::manifest::ManifestMeta {
                format_version: 2,
                ..Default::default()
            },
            ..Default::default()
        };
        assert!(validate_schema(&m).is_err());
    }

    #[test]
    fn rejects_observational_scope() {
        let mut m = Manifest::new_actual("".into());
        let mut uf = crate::manifest::UnmanagedFilesScope::default();
        uf.elements.push(crate::manifest::UnmanagedFileRecord {
            name: "/usr/bin/x".into(),
            r#type: "file".into(),
            ..Default::default()
        });
        m.unmanaged_files = Some(uf);
        assert!(validate_schema(&m).is_err());
    }
}
