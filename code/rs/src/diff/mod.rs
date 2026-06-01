// generated from spec: zypper-declarative.spec.md sha256:18253550a5c3d3f1818f0380811cb5dbc98874828693e49fc9cd5cbc923303dd
//
// BEHAVIOR/INTERNAL: compute-intent-diff and compute-drift.
//
// Both are PURE comparisons: they perform NO filesystem, rpmdb, or process I/O.
// The actual state is produced beforehand (by describe-actual-state or supplied
// as a dump). This module therefore imports nothing from `state` or `converge`.

use crate::manifest::{
    AppliedRecord, Diff, DriftReport, Manifest, ManagedFileRecord, PackageRecord, ServiceRecord,
};
use std::collections::HashSet;

/// The path the converge/drift layers must never write or delete, and which is
/// never reported as extra (spec INVARIANT).
const SYNCPOINT: &str = "/etc/etc.syncpoint";

/// compute-intent-diff: the changes from applied (old) to desired (new), scope
/// by scope. A scope absent in `desired` produces no change for that scope.
pub fn compute_intent_diff(desired: &Manifest, applied: &AppliedRecord) -> Diff {
    let mut diff = Diff::default();

    // 1. packages
    if let Some(d_pkgs) = &desired.packages {
        diff.packages_install = d_pkgs.elements.clone();
        let desired_names: HashSet<&str> =
            d_pkgs.elements.iter().map(|p| p.name.as_str()).collect();
        if let Some(a_pkgs) = &applied.packages {
            diff.packages_remove = a_pkgs
                .elements
                .iter()
                .filter(|p| !desired_names.contains(p.name.as_str()))
                .cloned()
                .collect();
        }
    }

    // 2. repositories
    if let Some(d_repos) = &desired.repositories {
        diff.repos_set = d_repos.elements.clone();
    }

    // 3. config_files: files_write := desired elements; files_delete :=
    //    (declared_old - declared_new) within the config_files scope.
    if let Some(d_files) = &desired.config_files {
        diff.files_write = d_files.elements.clone();
        let desired_paths: HashSet<&str> =
            d_files.elements.iter().map(|f| f.name.as_str()).collect();
        if let Some(a_files) = &applied.config_files {
            diff.files_delete = a_files
                .elements
                .iter()
                .filter(|f| !desired_paths.contains(f.name.as_str()))
                .map(|f| f.name.clone())
                .collect();
        }
    }

    // 4. services: units_change := desired services whose declared state differs
    //    from applied (including services present in desired but absent in
    //    applied).
    if let Some(d_svcs) = &desired.services {
        let applied_states: std::collections::HashMap<&str, &str> = applied
            .services
            .as_ref()
            .map(|s| {
                s.elements
                    .iter()
                    .map(|r| (r.name.as_str(), r.state.as_str()))
                    .collect()
            })
            .unwrap_or_default();
        diff.units_change = d_svcs
            .elements
            .iter()
            .filter(|r| applied_states.get(r.name.as_str()) != Some(&r.state.as_str()))
            .cloned()
            .collect();
    }

    diff
}

/// compute-drift: compares an actual-state Manifest against a declaration, scope
/// by scope on identity fields, and reports divergence. Performs no I/O.
///
/// `keep_list` is the set of paths that must never appear in files_extra,
/// managed_files_modified, or unmanaged_files_present (the keep-list escape
/// hatch). The syncpoint is always excluded from files_extra.
pub fn compute_drift(
    actual: &Manifest,
    reference: &AppliedRecord,
    keep_list: &HashSet<String>,
) -> DriftReport {
    let mut report = DriftReport::default();

    let actual_files: Vec<&ManagedFileRecord> = actual
        .config_files
        .as_ref()
        .map(|s| s.elements.iter().collect())
        .unwrap_or_default();
    let ref_files: Vec<&ManagedFileRecord> = reference
        .config_files
        .as_ref()
        .map(|s| s.elements.iter().collect())
        .unwrap_or_default();

    let actual_by_name: std::collections::HashMap<&str, &ManagedFileRecord> =
        actual_files.iter().map(|f| (f.name.as_str(), *f)).collect();
    let ref_names: HashSet<&str> = ref_files.iter().map(|f| f.name.as_str()).collect();

    // 1. files_modified: for each declared file, compare against actual.
    for e in &ref_files {
        if let Some(a) = actual_by_name.get(e.name.as_str()) {
            let modified = if a.file_type != e.file_type {
                // Type is part of identity: a type transition is modified.
                true
            } else if e.file_type == "file" {
                a.sha256 != e.sha256
            } else if e.file_type == "link" {
                a.target != e.target
            } else {
                false
            };
            if modified {
                report.files_modified.push(e.name.clone());
            }
        }
        // A declared entry absent from actual is treated as matching.
    }

    // 2. files_extra: actual files not declared, unpackaged, not keep-listed,
    //    not the syncpoint.
    for a in &actual_files {
        if ref_names.contains(a.name.as_str()) {
            continue;
        }
        if !a.package_name.is_empty() {
            continue; // package-owned undeclared files are not "extra".
        }
        if a.name == SYNCPOINT || keep_list.contains(&a.name) {
            continue;
        }
        report.files_extra.push(a.name.clone());
    }

    // 3. units_divergent: for each declared service, if actual reports a
    //    different state, add it.
    let actual_unit_state: std::collections::HashMap<&str, &str> = actual
        .services
        .as_ref()
        .map(|s| {
            s.elements
                .iter()
                .map(|r| (r.name.as_str(), r.state.as_str()))
                .collect()
        })
        .unwrap_or_default();
    if let Some(ref_svcs) = &reference.services {
        for u in &ref_svcs.elements {
            match actual_unit_state.get(u.name.as_str()) {
                Some(state) if *state != u.state.as_str() => {
                    report.units_divergent.push(ServiceRecord {
                        name: u.name.clone(),
                        state: u.state.clone(),
                    });
                }
                _ => {}
            }
        }
    }

    // 4. packages_divergent: compare identity fields; add any package present
    //    in one but not the other.
    let ref_pkgs: HashSet<PackageKey> = reference
        .packages
        .as_ref()
        .map(|s| s.elements.iter().map(PackageKey::of).collect())
        .unwrap_or_default();
    let act_pkgs: HashSet<PackageKey> = actual
        .packages
        .as_ref()
        .map(|s| s.elements.iter().map(PackageKey::of).collect())
        .unwrap_or_default();
    if reference.packages.is_some() || actual.packages.is_some() {
        // Records in reference not in actual.
        if let Some(rp) = &reference.packages {
            for p in &rp.elements {
                if !act_pkgs.contains(&PackageKey::of(p)) {
                    report.packages_divergent.push(p.clone());
                }
            }
        }
        // Records in actual not in reference.
        if let Some(ap) = &actual.packages {
            for p in &ap.elements {
                if !ref_pkgs.contains(&PackageKey::of(p)) {
                    report.packages_divergent.push(p.clone());
                }
            }
        }
    }

    // 5. Integrity categories (full scan): the observational scopes carry their
    //    own baseline (presence is itself drift). Honour the keep-list.
    if let Some(cmf) = &actual.changed_managed_files {
        for r in &cmf.elements {
            if !keep_list.contains(&r.name) {
                report.managed_files_modified.push(r.name.clone());
            }
        }
    }
    if let Some(uf) = &actual.unmanaged_files {
        for r in &uf.elements {
            if !keep_list.contains(&r.name) {
                report.unmanaged_files_present.push(r.name.clone());
            }
        }
    }

    report
}

/// A package identity key (name + version + release + arch).
#[derive(Hash, PartialEq, Eq)]
struct PackageKey {
    name: String,
    version: String,
    release: String,
    arch: String,
}

impl PackageKey {
    fn of(p: &PackageRecord) -> PackageKey {
        PackageKey {
            name: p.name.clone(),
            version: p.version.clone(),
            release: p.release.clone(),
            arch: p.arch.clone(),
        }
    }
}

/// Helper retained for documentation: a file record is considered declared if it
/// is in the reference scope.
#[allow(dead_code)]
fn is_declared(name: &str, reference: &AppliedRecord) -> bool {
    reference
        .config_files
        .as_ref()
        .map(|s| s.elements.iter().any(|f| f.name == name))
        .unwrap_or(false)
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::manifest::*;

    fn file_rec(name: &str, sha: &str) -> ManagedFileRecord {
        ManagedFileRecord {
            name: name.to_string(),
            file_type: "file".to_string(),
            mode: "0644".to_string(),
            user: "root".to_string(),
            group: "root".to_string(),
            sha256: sha.to_string(),
            ..Default::default()
        }
    }

    #[test]
    fn intent_diff_yields_deletion() {
        // EXAMPLE: intent_diff_yields_deletion
        let mut applied = Manifest::empty();
        applied.config_files = Some(ScopeWrapper {
            attributes: Default::default(),
            elements: vec![file_rec("/etc/foo.conf", "a"), file_rec("/etc/bar.conf", "b")],
        });
        let mut desired = Manifest::empty();
        desired.config_files = Some(ScopeWrapper {
            attributes: Default::default(),
            elements: vec![file_rec("/etc/foo.conf", "a")],
        });
        let diff = compute_intent_diff(&desired, &applied);
        assert_eq!(diff.files_delete, vec!["/etc/bar.conf".to_string()]);
        assert!(diff.files_write.iter().any(|f| f.name == "/etc/foo.conf"));
    }

    #[test]
    fn drift_type_transition_is_modified() {
        // EXAMPLE: drift_type_transition_is_modified
        let mut reference = Manifest::empty();
        reference.config_files = Some(ScopeWrapper {
            attributes: Default::default(),
            elements: vec![file_rec("/etc/foo", "deadbeef")],
        });
        let mut actual = Manifest::empty();
        let mut link = file_rec("/etc/foo", "");
        link.file_type = "link".to_string();
        link.target = "../else".to_string();
        actual.config_files = Some(ScopeWrapper {
            attributes: Default::default(),
            elements: vec![link],
        });
        let report = compute_drift(&actual, &reference, &HashSet::new());
        assert!(report.files_modified.contains(&"/etc/foo".to_string()));
    }

    #[test]
    fn drift_ignores_unmanaged_packaged_file() {
        // EXAMPLE: drift_ignores_unmanaged_packaged_file
        let reference = Manifest::empty();
        let mut actual = Manifest::empty();
        let mut owned = file_rec("/etc/owned.conf", "x");
        owned.package_name = "somepkg".to_string();
        actual.config_files = Some(ScopeWrapper {
            attributes: Default::default(),
            elements: vec![owned],
        });
        let report = compute_drift(&actual, &reference, &HashSet::new());
        assert!(report.files_extra.is_empty());
    }

    #[test]
    fn syncpoint_never_extra() {
        let reference = Manifest::empty();
        let mut actual = Manifest::empty();
        actual.config_files = Some(ScopeWrapper {
            attributes: Default::default(),
            elements: vec![file_rec("/etc/etc.syncpoint", "x")],
        });
        let report = compute_drift(&actual, &reference, &HashSet::new());
        assert!(report.files_extra.is_empty());
    }
}
