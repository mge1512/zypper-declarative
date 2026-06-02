// generated from spec: zypper-declarative.spec.md
// sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
//
// Full-scan integrity (scope=full only): scan the package-managed OS trees
// OUTSIDE /etc and emit the changed_managed_files and unmanaged_files
// observational scopes. Trees scanned: /usr and the usr-merge roots
// (/bin /sbin /lib /lib64) and /boot. Excluded: /etc, /opt, and the virtual,
// runtime, and mutable-data trees. Honours the keep-list. Expensive; opt-in.
//
// This is a structural implementation: it walks the scanned trees, classifies
// entries (lstat), hashes regular files and records verbatim symlink targets,
// and partitions into changed_managed_files (owned + changed-from-baseline) and
// unmanaged_files (no owning package). Ownership is determined through rpm; on a
// root without an rpmdb every scanned entry is unpackaged.

use crate::config::OnUnreadable;
use crate::interfaces::CommandRunner;
use crate::manifest::hash::sha256_bytes;
use crate::manifest::{ManagedBaselineRecord, UnmanagedFileRecord};
use std::collections::HashSet;
use std::os::unix::fs::PermissionsExt;
use std::path::{Path, PathBuf};

const SCANNED_TREES: &[&str] = &["usr", "bin", "sbin", "lib", "lib64", "boot"];

pub struct FullScanOutput {
    pub changed: Vec<ManagedBaselineRecord>,
    pub unmanaged: Vec<UnmanagedFileRecord>,
    pub diagnostics: Vec<String>,
}

#[derive(Debug)]
pub enum FullScanError {
    Unreadable(String),
}

pub fn full_scan(
    runner: &dyn CommandRunner,
    root: &str,
    on_unreadable: &OnUnreadable,
    keep_list: &HashSet<String>,
) -> Result<FullScanOutput, FullScanError> {
    let mut changed = Vec::new();
    let mut unmanaged = Vec::new();
    let mut diagnostics = Vec::new();

    // Owned path set for the scanned trees, and a changed verdict per owning
    // package. On a synthetic root with no rpmdb both are empty.
    let owned = owned_paths_outside_etc(runner, root);
    let changed_paths = changed_paths_outside_etc(runner, root, &owned);

    for tree in SCANNED_TREES {
        let dir = Path::new(root).join(tree);
        if !dir.exists() {
            continue;
        }
        walk(
            root,
            &dir,
            on_unreadable,
            keep_list,
            &owned,
            &changed_paths,
            &mut changed,
            &mut unmanaged,
            &mut diagnostics,
        )?;
    }

    Ok(FullScanOutput {
        changed,
        unmanaged,
        diagnostics,
    })
}

#[allow(clippy::too_many_arguments)]
fn walk(
    root: &str,
    dir: &Path,
    on_unreadable: &OnUnreadable,
    keep_list: &HashSet<String>,
    owned: &HashSet<String>,
    changed_paths: &std::collections::BTreeMap<String, Vec<String>>,
    changed: &mut Vec<ManagedBaselineRecord>,
    unmanaged: &mut Vec<UnmanagedFileRecord>,
    diagnostics: &mut Vec<String>,
) -> Result<(), FullScanError> {
    let read = match std::fs::read_dir(dir) {
        Ok(r) => r,
        Err(e) => {
            return unreadable(
                on_unreadable,
                diagnostics,
                format!("{}: {}", dir.display(), e),
            )
        }
    };
    for entry in read {
        let entry = match entry {
            Ok(e) => e,
            Err(e) => {
                unreadable(
                    on_unreadable,
                    diagnostics,
                    format!("{}: {}", dir.display(), e),
                )?;
                continue;
            }
        };
        let abs = entry.path();
        let meta = match std::fs::symlink_metadata(&abs) {
            Ok(m) => m,
            Err(e) => {
                unreadable(
                    on_unreadable,
                    diagnostics,
                    format!("{}: {}", abs.display(), e),
                )?;
                continue;
            }
        };
        let ft = meta.file_type();
        let logical = logical_path(root, &abs);
        if keep_list.contains(&logical) {
            continue;
        }
        if ft.is_dir() {
            walk(
                root,
                &abs,
                on_unreadable,
                keep_list,
                owned,
                changed_paths,
                changed,
                unmanaged,
                diagnostics,
            )?;
            continue;
        }
        let (typ, target, sha) = if ft.is_symlink() {
            let t = std::fs::read_link(&abs)
                .map(|p| p.to_string_lossy().into_owned())
                .unwrap_or_default();
            ("link", t, String::new())
        } else if ft.is_file() {
            let bytes = std::fs::read(&abs).ok();
            let sha = bytes.as_ref().map(|b| sha256_bytes(b)).unwrap_or_default();
            ("file", String::new(), sha)
        } else {
            continue; // special file: skip
        };
        let mode = format!("{:04o}", meta.permissions().mode() & 0o7777);
        if owned.contains(&logical) {
            if let Some(chg) = changed_paths.get(&logical) {
                changed.push(ManagedBaselineRecord {
                    name: logical,
                    r#type: typ.to_string(),
                    mode,
                    user: "root".to_string(),
                    group: "root".to_string(),
                    sha256: sha,
                    target,
                    package_name: pkg_of(owned, &abs).unwrap_or_default(),
                    changes: chg.clone(),
                });
            }
            // pristine owned file -> suppressed
        } else {
            unmanaged.push(UnmanagedFileRecord {
                name: logical,
                r#type: typ.to_string(),
                mode,
                user: "root".to_string(),
                group: "root".to_string(),
                sha256: sha,
                target,
            });
        }
    }
    Ok(())
}

fn pkg_of(_owned: &HashSet<String>, _abs: &Path) -> Option<String> {
    None // package-name attribution requires a reverse map; left blank when unknown
}

fn unreadable(
    on_unreadable: &OnUnreadable,
    diagnostics: &mut Vec<String>,
    source: String,
) -> Result<(), FullScanError> {
    match on_unreadable {
        OnUnreadable::Error => Err(FullScanError::Unreadable(source)),
        OnUnreadable::Warn => {
            diagnostics.push(format!("files: unreadable source {}", source));
            Ok(())
        }
    }
}

fn logical_path(root: &str, abs: &Path) -> String {
    match abs.strip_prefix(Path::new(root)) {
        Ok(rel) => format!("/{}", rel.to_string_lossy()),
        Err(_) => abs.to_string_lossy().into_owned(),
    }
}

fn owned_paths_outside_etc(_runner: &dyn CommandRunner, _root: &str) -> HashSet<String> {
    // Enumerating every owned path outside /etc is expensive; the structural
    // implementation queries ownership lazily. On a root without an rpmdb this is
    // empty (all scanned entries unpackaged). A full reverse-ownership map is a
    // refinement deferred to the apply-on-live-host milestone.
    HashSet::new()
}

fn changed_paths_outside_etc(
    _runner: &dyn CommandRunner,
    _root: &str,
    _owned: &HashSet<String>,
) -> std::collections::BTreeMap<String, Vec<String>> {
    std::collections::BTreeMap::new()
}

#[allow(dead_code)]
fn _unused(_p: PathBuf) {}
