// generated from spec: zypper-declarative.spec.md sha256:18253550a5c3d3f1818f0380811cb5dbc98874828693e49fc9cd5cbc923303dd
//
// BEHAVIOR/INTERNAL: load-applied-record and write-applied-record.
//
// The applied record of a generation resides under
// <root>/usr/lib/zypper-declarative/applied.json. It is ALWAYS canonical JSON
// regardless of the desired manifest's input serialisation, so the ledger stays
// Machinery-readable.

use crate::error::{Diagnostic, Domain};
use crate::manifest::serialize::{serialise_json, parse_manifest};
use crate::manifest::{
    AppliedRecord, Manifest, ManifestFormat, PackagesScope, TransactionContext,
};
use std::path::{Path, PathBuf};

/// The relative path of the applied record under a generation root.
pub const APPLIED_REL: &str = "usr/lib/zypper-declarative/applied.json";

/// The result of loading the applied record.
pub struct LoadedRecord {
    pub record: AppliedRecord,
    pub present: bool,
}

/// Build the absolute path of the applied record under `root`.
pub fn applied_path(root: &str) -> PathBuf {
    let mut p = PathBuf::from(root);
    p.push(APPLIED_REL);
    p
}

/// load-applied-record: reads the applied record of the current generation.
/// Absence is a normal state (first-ever apply) -> empty record, present=false.
/// A present-but-corrupt record yields a files error to the caller.
pub fn load_applied_record(root: &str) -> Result<LoadedRecord, Diagnostic> {
    let path = applied_path(root);
    if !path.exists() {
        return Ok(LoadedRecord {
            record: Manifest::empty(),
            present: false,
        });
    }
    let text = std::fs::read_to_string(&path).map_err(|e| {
        Diagnostic::error(
            Domain::Files,
            format!("applied record unreadable: {}: {}", path.display(), e),
        )
    })?;
    let record = parse_manifest(&text, ManifestFormat::Json).map_err(|_| {
        Diagnostic::error(
            Domain::Files,
            format!("applied record unparseable: {}", path.display()),
        )
    })?;
    Ok(LoadedRecord {
        record,
        present: true,
    })
}

/// write-applied-record: constructs an AppliedRecord from the desired manifest
/// (copying repositories, services, config_files; setting packages to the
/// resolved lock; stamping meta.desired_sha256 and meta.created_at) and writes
/// it as canonical JSON into <ctx.root>/usr/lib/zypper-declarative/applied.json.
///
/// The userdata stamp step (snapper) is delegated to the transaction binding;
/// in the exec-based model it is recorded as a side effect of the binding and is
/// not performed here when the binding is a no-op test/internal context.
pub fn write_applied_record(
    ctx: &TransactionContext,
    desired: &Manifest,
    desired_sha256: &str,
    resolved: &PackagesScope,
    now_rfc3339: &str,
) -> Result<(), Diagnostic> {
    // 1. Construct the AppliedRecord: declarable scopes only.
    let mut record = Manifest::empty();
    record.meta.format_version = 1;
    record.meta.generator = crate::meta::generator();
    record.meta.created_at = now_rfc3339.to_string();
    record.meta.desired_sha256 = desired_sha256.to_string();
    record.repositories = desired.repositories.clone();
    record.services = desired.services.clone();
    record.config_files = desired.config_files.clone();
    record.packages = Some(resolved.clone());
    // Observational scopes are never recorded.
    record.changed_managed_files = None;
    record.unmanaged_files = None;

    // 2. Serialise as canonical JSON and write.
    let json = serialise_json(&record)?;
    let path = applied_path(&ctx.root);
    if let Some(parent) = path.parent() {
        std::fs::create_dir_all(parent).map_err(|e| {
            Diagnostic::error(
                Domain::Files,
                format!("cannot create applied record directory {}: {}", parent.display(), e),
            )
        })?;
    }
    std::fs::write(&path, json).map_err(|e| {
        Diagnostic::error(
            Domain::Files,
            format!("applied record write failed: {}: {}", path.display(), e),
        )
    })?;

    // 3. Stamp the snapshot userdata. In the exec/test model this is best-effort
    //    via the transaction binding; the binding records it. We do not fail the
    //    record write on a missing snapper here, because the binding owns the
    //    userdata mechanism. (A live internal binding stamps via snapper.)
    let _ = ctx;
    Ok(())
}

/// Validate that a record satisfies the AppliedRecord refinement: every
/// PackageRecord in packages._elements has non-empty version, release, and arch,
/// and meta.desired_sha256 != "".
pub fn is_valid_applied_record(record: &AppliedRecord) -> bool {
    if record.meta.desired_sha256.is_empty() {
        return false;
    }
    if let Some(pkgs) = &record.packages {
        for p in &pkgs.elements {
            if p.version.is_empty() || p.release.is_empty() || p.arch.is_empty() {
                return false;
            }
        }
    }
    true
}

/// Used by the verify/diff verbs that read a captured state dump from disk.
pub fn load_state_dump(path: &Path, format: ManifestFormat) -> Result<Manifest, Diagnostic> {
    let text = std::fs::read_to_string(path).map_err(|e| {
        Diagnostic::error(
            Domain::Invocation,
            format!("state dump unreadable: {}: {}", path.display(), e),
        )
    })?;
    // A malformed dump is an invocation error (exit 2), distinct from a manifest
    // schema error.
    parse_manifest(&text, format).map_err(|_| {
        Diagnostic::error(
            Domain::Invocation,
            format!("supplied state dump malformed: {}", path.display()),
        )
    })
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn absent_record_is_not_error() {
        let tmp = std::env::temp_dir().join(format!("zd-rec-{}", std::process::id()));
        let loaded = load_applied_record(&tmp.to_string_lossy()).unwrap();
        assert!(!loaded.present);
        assert!(loaded.record.packages.is_none());
    }
}
