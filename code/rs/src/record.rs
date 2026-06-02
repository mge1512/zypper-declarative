// generated from spec: zypper-declarative.spec.md
// sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
//
// BEHAVIOR/INTERNAL: load-applied-record and write-applied-record.
// The applied record is always canonical JSON regardless of the desired
// manifest's input serialisation, so the on-disk ledger stays Machinery-readable.

use crate::error::{Diagnostic, Domain};
use crate::manifest::{Manifest, PackagesScope};
use std::path::{Path, PathBuf};

/// Relative path of the applied record within a generation root.
pub const APPLIED_REL: &str = "usr/lib/zypper-declarative/applied.json";

pub struct AppliedLoad {
    pub record: Manifest,
    pub present: bool,
}

/// Resolve `<root>/usr/lib/zypper-declarative/applied.json`.
pub fn applied_path(root: &str) -> PathBuf {
    Path::new(root).join(APPLIED_REL)
}

/// load-applied-record: read the applied record of the current generation.
/// Absence yields an all-empty record with present=false (not an error).
pub fn load_applied_record(root: &str) -> Result<AppliedLoad, Diagnostic> {
    let path = applied_path(root);
    if !path.exists() {
        return Ok(AppliedLoad {
            record: empty_applied(),
            present: false,
        });
    }
    let text = std::fs::read_to_string(&path).map_err(|e| {
        Diagnostic::error(
            Domain::Files,
            format!("applied record {} unreadable: {}", path.display(), e),
        )
    })?;
    let record: Manifest = serde_json::from_str(&text).map_err(|e| {
        Diagnostic::error(
            Domain::Files,
            format!("applied record {} is unparseable: {}", path.display(), e),
        )
    })?;
    Ok(AppliedLoad {
        record,
        present: true,
    })
}

/// An empty applied record: format_version 1, all declarable scopes empty/absent.
pub fn empty_applied() -> Manifest {
    Manifest {
        meta: crate::manifest::ManifestMeta {
            format_version: 1,
            generator: crate::meta::generator(),
            created_at: String::new(),
            desired_sha256: String::new(),
        },
        ..Default::default()
    }
}

/// Construct the applied record from a desired manifest plus the resolved
/// packages lock and the desired_sha256 (write-applied-record STEP 1).
pub fn build_applied_record(
    desired: &Manifest,
    desired_sha256: &str,
    resolved: PackagesScope,
    created_at: String,
) -> Manifest {
    Manifest {
        meta: crate::manifest::ManifestMeta {
            format_version: 1,
            generator: crate::meta::generator(),
            created_at,
            desired_sha256: desired_sha256.to_string(),
        },
        packages: Some(resolved),
        repositories: desired.repositories.clone(),
        services: desired.services.clone(),
        config_files: desired.config_files.clone(),
        // Observational scopes are never recorded.
        changed_managed_files: None,
        unmanaged_files: None,
    }
}

/// Serialise the applied record as canonical JSON and write it into the
/// transaction context root (write-applied-record STEP 2).
pub fn write_applied_record(root: &str, record: &Manifest) -> Result<(), Diagnostic> {
    let path = applied_path(root);
    if let Some(parent) = path.parent() {
        std::fs::create_dir_all(parent).map_err(|e| {
            Diagnostic::error(
                Domain::Files,
                format!("cannot create {}: {}", parent.display(), e),
            )
        })?;
    }
    let json = serde_json::to_string_pretty(record).map_err(|e| {
        Diagnostic::error(
            Domain::Files,
            format!("cannot serialise applied record: {}", e),
        )
    })?;
    std::fs::write(&path, json).map_err(|e| {
        Diagnostic::error(
            Domain::Files,
            format!("cannot write {}: {}", path.display(), e),
        )
    })?;
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn absent_record_is_present_false_not_error() {
        let dir = std::env::temp_dir().join(format!("zd-rec-{}", std::process::id()));
        std::fs::create_dir_all(&dir).unwrap();
        let load = load_applied_record(dir.to_str().unwrap()).unwrap();
        assert!(!load.present);
        assert_eq!(load.record.meta.format_version, 1);
    }
}
