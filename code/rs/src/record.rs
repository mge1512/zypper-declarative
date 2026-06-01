// generated from spec: zypper-declarative.spec.md sha256:87b9f2f3bf92afd6fe037412d56a23959290e5816d722a3def9505d07aa5acd7
//
// BEHAVIOR/INTERNAL: load-applied-record and write-applied-record. The applied
// record is stored under /usr within the generation it describes; it is always
// canonical JSON regardless of the desired manifest's input serialisation.

use crate::error::{Diagnostic, Domain};
use crate::manifest::serialise_applied_record;
use crate::types::*;
use std::path::PathBuf;

const APPLIED_REL: &str = "usr/lib/zypper-declarative/applied.json";

/// Result of load-applied-record.
pub struct LoadedApplied {
    pub record: AppliedRecord,
    pub present: bool,
}

/// load-applied-record. Absence is a normal state (first-ever apply) reported as
/// an empty record, not an error. A present-but-corrupt record is a files error.
pub fn load_applied_record(root: &str) -> Result<LoadedApplied, Diagnostic> {
    let path = applied_path(root);
    if !path.exists() {
        return Ok(LoadedApplied {
            record: Manifest::empty(),
            present: false,
        });
    }
    let bytes = std::fs::read(&path).map_err(|e| {
        Diagnostic::error(
            Domain::Files,
            format!("applied record unreadable: {}: {}", path.display(), e),
        )
    })?;
    let record: AppliedRecord = serde_json::from_slice(&bytes).map_err(|e| {
        Diagnostic::error(
            Domain::Files,
            format!("applied record unparseable: {}: {}", path.display(), e),
        )
    })?;
    Ok(LoadedApplied {
        record,
        present: true,
    })
}

/// write-applied-record. Constructs an AppliedRecord from the desired manifest with
/// the packages scope set to the resolved lock, serialises it as canonical JSON,
/// and writes it under the context root. (Snapper userdata stamping is delegated
/// to the transaction layer on a live host; here we write the ledger file.)
pub fn write_applied_record(
    ctx_root: &str,
    desired: &Manifest,
    desired_sha256: &str,
    resolved: &PackagesScope,
) -> Result<(), Diagnostic> {
    let mut record = AppliedRecord::empty();
    record.meta.format_version = 1;
    record.meta.generator = crate::meta::generator();
    record.meta.created_at = now_stamp();
    record.meta.desired_sha256 = desired_sha256.to_string();
    // Copy declarable scopes from desired; observational scopes are never recorded.
    record.repositories = desired.repositories.clone();
    record.services = desired.services.clone();
    record.config_files = desired.config_files.clone();
    // packages scope = resolved lock.
    record.packages = Some(resolved.clone());

    let path = applied_path(ctx_root);
    if let Some(parent) = path.parent() {
        std::fs::create_dir_all(parent).map_err(|e| {
            Diagnostic::error(
                Domain::Files,
                format!("applied record write failed: {}: {}", parent.display(), e),
            )
        })?;
    }
    let json = serialise_applied_record(&record)?;
    std::fs::write(&path, json).map_err(|e| {
        Diagnostic::error(
            Domain::Files,
            format!("applied record write failed: {}: {}", path.display(), e),
        )
    })?;
    Ok(())
}

fn applied_path(root: &str) -> PathBuf {
    let mut p = PathBuf::from(root);
    p.push(APPLIED_REL);
    p
}

fn now_stamp() -> String {
    let now = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map(|d| d.as_secs())
        .unwrap_or(0);
    format!("1970-01-01T00:00:{:02}Z", now % 60)
}
