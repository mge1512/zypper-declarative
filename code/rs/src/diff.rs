// generated from spec: zypper-declarative.spec.md sha256:87b9f2f3bf92afd6fe037412d56a23959290e5816d722a3def9505d07aa5acd7
//
// BEHAVIOR/INTERNAL: compute-intent-diff and compute-drift. Both are PURE: they
// perform no filesystem, rpmdb, or process I/O. They compare in-memory Manifest
// values only.

use crate::types::*;
use std::collections::HashSet;

/// compute-intent-diff: changes from the previously applied declaration to the
/// desired declaration, scope by scope. A scope absent in desired produces no
/// change for that scope; a present scope is reconciled to exactly its elements.
pub fn compute_intent_diff(desired: &Manifest, applied: &AppliedRecord) -> Diff {
    let mut diff = Diff::default();

    // Step 1: packages.
    if let Some(dp) = desired.packages.as_ref() {
        diff.packages_install = dp.elements.clone();
        let desired_names: HashSet<&str> = dp.elements.iter().map(|p| p.name.as_str()).collect();
        diff.packages_remove = applied
            .packages_elems()
            .iter()
            .filter(|p| !desired_names.contains(p.name.as_str()))
            .cloned()
            .collect();
    }

    // Step 2: repositories.
    if let Some(dr) = desired.repositories.as_ref() {
        diff.repos_set = dr.elements.clone();
    }

    // Step 3: config_files.
    if let Some(dc) = desired.config_files.as_ref() {
        diff.files_write = dc.elements.clone();
        let desired_paths: HashSet<&str> = dc.elements.iter().map(|f| f.name.as_str()).collect();
        // files_delete = (declared_old - declared_new) within config_files.
        diff.files_delete = applied
            .config_files_elems()
            .iter()
            .filter(|f| !desired_paths.contains(f.name.as_str()))
            .map(|f| f.name.clone())
            .collect();
    }

    // Step 4: services.
    if let Some(ds) = desired.services.as_ref() {
        diff.units_change = ds
            .elements
            .iter()
            .filter(|u| {
                match applied.services_elems().iter().find(|a| a.name == u.name) {
                    Some(applied_unit) => applied_unit.state != u.state,
                    None => true, // present in desired, absent in applied -> change
                }
            })
            .cloned()
            .collect();
    }

    diff
}

/// compute-drift: compares an actual-state Manifest against a declaration, scope
/// by scope on identity fields, and reports divergence. keep_list paths and
/// /etc/etc.syncpoint never appear in files_extra.
pub fn compute_drift(
    actual: &Manifest,
    reference: &AppliedRecord,
    keep_list: &HashSet<String>,
) -> DriftReport {
    let mut report = DriftReport::default();

    // Step 1: files_modified.
    for e in reference.config_files_elems() {
        if let Some(a) = actual
            .config_files_elems()
            .iter()
            .find(|a| a.name == e.name)
        {
            let modified = if a.r#type != e.r#type {
                // a type transition (type is part of identity)
                true
            } else if a.r#type == "file" {
                a.sha256 != e.sha256
            } else if a.r#type == "link" {
                a.target != e.target
            } else {
                false
            };
            if modified {
                report.files_modified.push(e.name.clone());
            }
        }
        // A declared entry absent from actual is treated as matching (not missing).
    }

    // Step 2: files_extra. Unpackaged, undeclared /etc files not keep-listed and
    // not /etc/etc.syncpoint.
    let declared: HashSet<&str> = reference
        .config_files_elems()
        .iter()
        .map(|f| f.name.as_str())
        .collect();
    for a in actual.config_files_elems() {
        if declared.contains(a.name.as_str()) {
            continue;
        }
        if !a.package_name.is_empty() {
            continue; // package-owned, not "extra"
        }
        if is_excluded(&a.name, keep_list) {
            continue;
        }
        report.files_extra.push(a.name.clone());
    }

    // Step 3: units_divergent.
    for u in reference.services_elems() {
        if let Some(a) = actual.services_elems().iter().find(|a| a.name == u.name) {
            if a.state != u.state {
                report.units_divergent.push(u.clone());
            }
        }
    }

    // Step 4: packages_divergent. Identity-field comparison; any package present
    // in one but not the other.
    let actual_pkgs: HashSet<PkgIdentity> = actual
        .packages_elems()
        .iter()
        .map(PkgIdentity::of)
        .collect();
    let ref_pkgs: HashSet<PkgIdentity> = reference
        .packages_elems()
        .iter()
        .map(PkgIdentity::of)
        .collect();
    for p in reference.packages_elems() {
        if !actual_pkgs.contains(&PkgIdentity::of(p)) {
            report.packages_divergent.push(p.clone());
        }
    }
    for p in actual.packages_elems() {
        if !ref_pkgs.contains(&PkgIdentity::of(p)) {
            report.packages_divergent.push(p.clone());
        }
    }

    // Step 5: integrity categories (full scan). Presence is itself drift.
    if let Some(cm) = actual.changed_managed_files.as_ref() {
        for e in &cm.elements {
            report.managed_files_modified.push(e.name.clone());
        }
    }
    if let Some(um) = actual.unmanaged_files.as_ref() {
        for e in &um.elements {
            if !is_excluded(&e.name, keep_list) {
                report.unmanaged_files_present.push(e.name.clone());
            }
        }
    }

    report
}

const SYNCPOINT: &str = "/etc/etc.syncpoint";

fn is_excluded(path: &str, keep_list: &HashSet<String>) -> bool {
    path == SYNCPOINT || keep_list.contains(path)
}

#[derive(Hash, PartialEq, Eq)]
struct PkgIdentity {
    name: String,
    version: String,
    release: String,
    arch: String,
}

impl PkgIdentity {
    fn of(p: &PackageRecord) -> Self {
        PkgIdentity {
            name: p.name.clone(),
            version: p.version.clone(),
            release: p.release.clone(),
            arch: p.arch.clone(),
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn cf(name: &str, ty: &str, sha: &str, target: &str, pkg: &str) -> ManagedFileRecord {
        ManagedFileRecord {
            name: name.into(),
            r#type: ty.into(),
            mode: "0644".into(),
            user: "root".into(),
            group: "root".into(),
            sha256: sha.into(),
            target: target.into(),
            content_ref: String::new(),
            package_name: pkg.into(),
        }
    }

    #[test]
    fn intent_diff_yields_deletion() {
        let mut applied = Manifest::empty();
        applied.config_files = Some(ScopeWrapper {
            attributes: Default::default(),
            elements: vec![
                cf("/etc/foo.conf", "file", "a", "", ""),
                cf("/etc/bar.conf", "file", "b", "", ""),
            ],
        });
        let mut desired = Manifest::empty();
        desired.config_files = Some(ScopeWrapper {
            attributes: Default::default(),
            elements: vec![cf("/etc/foo.conf", "file", "a", "", "")],
        });
        let d = compute_intent_diff(&desired, &applied);
        assert_eq!(d.files_delete, vec!["/etc/bar.conf".to_string()]);
        assert_eq!(d.files_write.len(), 1);
    }

    #[test]
    fn absent_scope_no_change() {
        let applied = Manifest::empty();
        let desired = Manifest::empty(); // config_files absent
        let d = compute_intent_diff(&desired, &applied);
        assert!(d.files_delete.is_empty() && d.files_write.is_empty());
    }

    #[test]
    fn drift_type_transition_is_modified() {
        let mut reference = Manifest::empty();
        reference.config_files = Some(ScopeWrapper {
            attributes: Default::default(),
            elements: vec![cf("/etc/foo", "file", "x", "", "")],
        });
        let mut actual = Manifest::empty();
        actual.config_files = Some(ScopeWrapper {
            attributes: Default::default(),
            elements: vec![cf("/etc/foo", "link", "", "../bar", "")],
        });
        let r = compute_drift(&actual, &reference, &HashSet::new());
        assert_eq!(r.files_modified, vec!["/etc/foo".to_string()]);
    }

    #[test]
    fn drift_ignores_unmanaged_packaged_file() {
        let reference = Manifest::empty();
        let mut actual = Manifest::empty();
        actual.config_files = Some(ScopeWrapper {
            attributes: Default::default(),
            elements: vec![cf("/etc/owned.conf", "file", "z", "", "some-package")],
        });
        let r = compute_drift(&actual, &reference, &HashSet::new());
        assert!(r.files_extra.is_empty(), "package-owned file is not extra");
    }

    #[test]
    fn syncpoint_never_extra() {
        let reference = Manifest::empty();
        let mut actual = Manifest::empty();
        actual.config_files = Some(ScopeWrapper {
            attributes: Default::default(),
            elements: vec![cf("/etc/etc.syncpoint", "file", "z", "", "")],
        });
        let r = compute_drift(&actual, &reference, &HashSet::new());
        assert!(r.files_extra.is_empty());
    }
}
