// generated from spec: zypper-declarative.spec.md
// sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
//
// BEHAVIOR/INTERNAL: compute-intent-diff and compute-drift.
// Both are PURE comparisons of in-memory Manifest values. This module performs
// no filesystem, rpmdb, or process I/O.

use crate::manifest::{
    ManagedFileRecord, Manifest, PackageRecord, RepositoryRecord, ServiceRecord,
};
use std::collections::HashSet;

pub const SYNCPOINT: &str = "/etc/etc.syncpoint";

#[derive(Debug, Clone, Default, PartialEq)]
pub struct Diff {
    pub packages_install: Vec<PackageRecord>,
    pub packages_remove: Vec<PackageRecord>,
    pub repos_set: Vec<RepositoryRecord>,
    pub files_write: Vec<ManagedFileRecord>,
    pub files_delete: Vec<String>,
    pub units_change: Vec<ServiceRecord>,
}

impl Diff {
    pub fn is_empty(&self) -> bool {
        self.packages_install.is_empty()
            && self.packages_remove.is_empty()
            && self.repos_set.is_empty()
            && self.files_write.is_empty()
            && self.files_delete.is_empty()
            && self.units_change.is_empty()
    }
}

#[derive(Debug, Clone, Default, PartialEq)]
pub struct DriftReport {
    pub files_modified: Vec<String>,
    pub files_extra: Vec<String>,
    pub units_divergent: Vec<ServiceRecord>,
    pub packages_divergent: Vec<PackageRecord>,
    pub managed_files_modified: Vec<String>,
    pub unmanaged_files_present: Vec<String>,
}

impl DriftReport {
    pub fn is_empty(&self) -> bool {
        self.files_modified.is_empty()
            && self.files_extra.is_empty()
            && self.units_divergent.is_empty()
            && self.packages_divergent.is_empty()
            && self.managed_files_modified.is_empty()
            && self.unmanaged_files_present.is_empty()
    }

    pub fn count(&self) -> usize {
        self.files_modified.len()
            + self.files_extra.len()
            + self.units_divergent.len()
            + self.packages_divergent.len()
            + self.managed_files_modified.len()
            + self.unmanaged_files_present.len()
    }
}

/// compute-intent-diff(desired, applied) -> Diff
pub fn compute_intent_diff(desired: &Manifest, applied: &Manifest) -> Diff {
    let mut diff = Diff::default();

    // 1. packages
    if let Some(dp) = &desired.packages {
        diff.packages_install = dp.elements.clone();
        let desired_names: HashSet<&str> = dp.elements.iter().map(|p| p.name.as_str()).collect();
        if let Some(ap) = &applied.packages {
            diff.packages_remove = ap
                .elements
                .iter()
                .filter(|p| !desired_names.contains(p.name.as_str()))
                .cloned()
                .collect();
        }
    }

    // 2. repositories
    if let Some(dr) = &desired.repositories {
        diff.repos_set = dr.elements.clone();
    }

    // 3. config_files
    if let Some(dc) = &desired.config_files {
        diff.files_write = dc.elements.clone();
        let desired_paths: HashSet<&str> = dc.elements.iter().map(|e| e.name.as_str()).collect();
        if let Some(ac) = &applied.config_files {
            diff.files_delete = ac
                .elements
                .iter()
                .filter(|e| !desired_paths.contains(e.name.as_str()))
                .map(|e| e.name.clone())
                .collect();
        }
    }

    // 4. services
    if let Some(ds) = &desired.services {
        let applied_states: std::collections::HashMap<&str, &str> = applied
            .services
            .as_ref()
            .map(|s| {
                s.elements
                    .iter()
                    .map(|u| (u.name.as_str(), u.state.as_str()))
                    .collect()
            })
            .unwrap_or_default();
        diff.units_change = ds
            .elements
            .iter()
            .filter(|u| applied_states.get(u.name.as_str()) != Some(&u.state.as_str()))
            .cloned()
            .collect();
    }

    diff
}

/// compute-drift(actual, reference, keep_list) -> DriftReport
pub fn compute_drift(
    actual: &Manifest,
    reference: &Manifest,
    keep_list: &HashSet<String>,
) -> DriftReport {
    let mut report = DriftReport::default();

    // 1. files_modified
    let actual_files: std::collections::HashMap<&str, &ManagedFileRecord> = actual
        .config_files
        .as_ref()
        .map(|s| s.elements.iter().map(|e| (e.name.as_str(), e)).collect())
        .unwrap_or_default();

    if let Some(ref_cf) = &reference.config_files {
        for e in &ref_cf.elements {
            if let Some(a) = actual_files.get(e.name.as_str()) {
                let modified = if a.r#type != e.r#type {
                    true // type transition (type is part of identity)
                } else if e.r#type == "file" {
                    a.sha256 != e.sha256
                } else if e.r#type == "link" {
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
    }

    // 2. files_extra: unpackaged, undeclared, not keep-listed, not syncpoint.
    let declared: HashSet<&str> = reference
        .config_files
        .as_ref()
        .map(|s| s.elements.iter().map(|e| e.name.as_str()).collect())
        .unwrap_or_default();
    if let Some(act_cf) = &actual.config_files {
        for a in &act_cf.elements {
            if declared.contains(a.name.as_str()) {
                continue;
            }
            if !a.package_name.is_empty() {
                continue; // package-owned -> not "extra"
            }
            if a.name == SYNCPOINT || keep_list.contains(&a.name) {
                continue;
            }
            report.files_extra.push(a.name.clone());
        }
    }

    // 3. units_divergent
    let actual_units: std::collections::HashMap<&str, &str> = actual
        .services
        .as_ref()
        .map(|s| {
            s.elements
                .iter()
                .map(|u| (u.name.as_str(), u.state.as_str()))
                .collect()
        })
        .unwrap_or_default();
    if let Some(ref_svc) = &reference.services {
        for u in &ref_svc.elements {
            if let Some(state) = actual_units.get(u.name.as_str()) {
                if *state != u.state.as_str() {
                    report.units_divergent.push(u.clone());
                }
            }
            // a service absent from actual is treated as matching
        }
    }

    // 4. packages_divergent: identity present in one but not the other.
    let ref_pkgs: HashSet<String> = reference
        .packages
        .as_ref()
        .map(|s| s.elements.iter().map(pkg_identity).collect())
        .unwrap_or_default();
    let act_pkgs: HashSet<String> = actual
        .packages
        .as_ref()
        .map(|s| s.elements.iter().map(pkg_identity).collect())
        .unwrap_or_default();
    if reference.packages.is_some() && actual.packages.is_some() {
        if let Some(rp) = &reference.packages {
            for p in &rp.elements {
                if !act_pkgs.contains(&pkg_identity(p)) {
                    report.packages_divergent.push(p.clone());
                }
            }
        }
        if let Some(ap) = &actual.packages {
            for p in &ap.elements {
                if !ref_pkgs.contains(&pkg_identity(p)) {
                    report.packages_divergent.push(p.clone());
                }
            }
        }
    }

    // 5. integrity categories (full scan): their presence is itself drift.
    if let Some(cmf) = &actual.changed_managed_files {
        for e in &cmf.elements {
            if !keep_list.contains(&e.name) {
                report.managed_files_modified.push(e.name.clone());
            }
        }
    }
    if let Some(uf) = &actual.unmanaged_files {
        for e in &uf.elements {
            if !keep_list.contains(&e.name) {
                report.unmanaged_files_present.push(e.name.clone());
            }
        }
    }

    report
}

fn pkg_identity(p: &PackageRecord) -> String {
    format!(
        "{}\u{0}{}\u{0}{}\u{0}{}",
        p.name, p.version, p.release, p.arch
    )
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::manifest::*;

    fn file_rec(name: &str, sha: &str) -> ManagedFileRecord {
        ManagedFileRecord {
            name: name.into(),
            r#type: "file".into(),
            mode: "0644".into(),
            user: "root".into(),
            group: "root".into(),
            sha256: sha.into(),
            ..Default::default()
        }
    }

    #[test]
    fn intent_diff_yields_deletion() {
        let mut desired = Manifest::new_actual("".into());
        let mut applied = Manifest::new_actual("".into());
        let mut dc = ConfigFilesScope::default();
        dc.elements.push(file_rec("/etc/foo.conf", "aa"));
        desired.config_files = Some(dc);
        let mut ac = ConfigFilesScope::default();
        ac.elements.push(file_rec("/etc/foo.conf", "aa"));
        ac.elements.push(file_rec("/etc/bar.conf", "bb"));
        applied.config_files = Some(ac);

        let diff = compute_intent_diff(&desired, &applied);
        assert_eq!(diff.files_delete, vec!["/etc/bar.conf".to_string()]);
        assert!(diff.files_write.iter().any(|e| e.name == "/etc/foo.conf"));
    }

    #[test]
    fn drift_type_transition_is_modified() {
        let mut reference = Manifest::new_actual("".into());
        let mut actual = Manifest::new_actual("".into());
        let mut rc = ConfigFilesScope::default();
        rc.elements.push(file_rec("/etc/foo", "aa"));
        reference.config_files = Some(rc);
        let mut ac = ConfigFilesScope::default();
        let mut link = file_rec("/etc/foo", "");
        link.r#type = "link".into();
        link.sha256 = "".into();
        link.target = "x".into();
        ac.elements.push(link);
        actual.config_files = Some(ac);

        let report = compute_drift(&actual, &reference, &HashSet::new());
        assert_eq!(report.files_modified, vec!["/etc/foo".to_string()]);
    }

    #[test]
    fn drift_ignores_package_owned_undeclared_file() {
        let reference = Manifest::new_actual("".into());
        let mut actual = Manifest::new_actual("".into());
        let mut ac = ConfigFilesScope::default();
        let mut owned = file_rec("/etc/owned.conf", "cc");
        owned.package_name = "somepkg".into();
        ac.elements.push(owned);
        actual.config_files = Some(ac);
        let report = compute_drift(&actual, &reference, &HashSet::new());
        assert!(
            report.files_extra.is_empty(),
            "package-owned undeclared file is not extra"
        );
    }
}
