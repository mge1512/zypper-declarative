// generated from spec: zypper-declarative.spec.md
// sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
//
// Human-readable rendering of the intent diff plan and the drift report for
// stdout (diff plan, status report, verify diagnostics).

use crate::diff::{Diff, DriftReport};

/// Render the diff plan + drift for `diff` to stdout.
pub fn render_plan(diff: &Diff, drift: &DriftReport) -> String {
    let mut s = String::new();
    s.push_str("packages to install:\n");
    for p in &diff.packages_install {
        s.push_str(&format!("  + {}\n", p.name));
    }
    s.push_str("packages to remove:\n");
    for p in &diff.packages_remove {
        s.push_str(&format!("  - {}\n", p.name));
    }
    s.push_str("repositories to set:\n");
    for r in &diff.repos_set {
        s.push_str(&format!("  = {} ({})\n", r.alias, r.url));
    }
    s.push_str("files to write:\n");
    for f in &diff.files_write {
        s.push_str(&format!("  > {}\n", f.name));
    }
    s.push_str("files to delete:\n");
    for f in &diff.files_delete {
        s.push_str(&format!("  x {}\n", f));
    }
    s.push_str("units to change:\n");
    for u in &diff.units_change {
        s.push_str(&format!("  ~ {} -> {}\n", u.name, u.state));
    }
    s.push_str("drift:\n");
    s.push_str(&render_drift_summary(drift));
    s
}

pub fn render_drift_summary(drift: &DriftReport) -> String {
    if drift.is_empty() {
        return "  clean\n".to_string();
    }
    let mut s = String::new();
    for f in &drift.files_modified {
        s.push_str(&format!("  modified: {}\n", f));
    }
    for f in &drift.files_extra {
        s.push_str(&format!("  extra: {}\n", f));
    }
    for u in &drift.units_divergent {
        s.push_str(&format!("  unit: {} (want {})\n", u.name, u.state));
    }
    for p in &drift.packages_divergent {
        s.push_str(&format!("  package: {}\n", p.name));
    }
    for f in &drift.managed_files_modified {
        s.push_str(&format!("  managed-modified: {}\n", f));
    }
    for f in &drift.unmanaged_files_present {
        s.push_str(&format!("  unmanaged: {}\n", f));
    }
    s
}
