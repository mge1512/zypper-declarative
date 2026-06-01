// generated from spec: zypper-declarative.spec.md sha256:18253550a5c3d3f1818f0380811cb5dbc98874828693e49fc9cd5cbc923303dd
//
// BEHAVIOR/INTERNAL: converge-packages, converge-files, converge-units.
//
// Each applies the corresponding portion of the intent diff inside a transaction
// context root and returns errors to the caller (the apply verb maps them to
// exit codes). The same convergence code path runs regardless of whether the
// transaction binding resolved external or internal.

use crate::config::Config;
use crate::error::{Diagnostic, Domain};
use crate::interfaces::CommandRunner;
use crate::manifest::{Diff, PackagesScope, ScopeWrapper, TransactionContext};
use sha2::{Digest, Sha256};
use std::collections::HashSet;
use std::path::{Path, PathBuf};

const SYNCPOINT: &str = "/etc/etc.syncpoint";

/// converge-packages: configure repos, install, remove, then return the rpmdb
/// installed set as the resolved lock.
pub fn converge_packages(
    runner: &dyn CommandRunner,
    ctx: &TransactionContext,
    diff: &Diff,
    config: &Config,
) -> Result<PackagesScope, Diagnostic> {
    // 1. Ensure repositories configured (or the CONFIG pin if repos_set empty).
    if diff.repos_set.is_empty() {
        if let Some(pin) = &config.repo_lock {
            let r = runner.run("zypper", &["--root", &ctx.root, "ar", "-f", pin, "pinned"]);
            if r.spawn_failed {
                return Err(Diagnostic::error(
                    Domain::Repositories,
                    "repository configuration failed: package manager unavailable",
                ));
            }
        }
    } else {
        for repo in &diff.repos_set {
            let r = runner.run(
                "zypper",
                &["--root", &ctx.root, "ar", "-f", &repo.url, &repo.alias],
            );
            if r.spawn_failed {
                return Err(Diagnostic::error(
                    Domain::Repositories,
                    "repository configuration failed: package manager unavailable",
                ));
            }
        }
    }

    // 2. Install desired packages against the configured pinned repositories.
    for p in &diff.packages_install {
        let r = runner.run(
            "zypper",
            &["--root", &ctx.root, "--non-interactive", "in", &p.name],
        );
        if r.spawn_failed || r.code != 0 {
            return Err(Diagnostic::error(
                Domain::Packages,
                format!("package install failed: {}", p.name),
            ));
        }
    }

    // 3. Remove dropped packages.
    for p in &diff.packages_remove {
        let r = runner.run(
            "zypper",
            &["--root", &ctx.root, "--non-interactive", "rm", &p.name],
        );
        if r.spawn_failed || r.code != 0 {
            return Err(Diagnostic::error(
                Domain::Packages,
                format!("package remove failed: {}", p.name),
            ));
        }
    }

    // 4. Query the rpmdb for the full installed set; return as the lock.
    let res = runner.run(
        "rpm",
        &[
            "-qa",
            "--root",
            &ctx.root,
            "--qf",
            "%{NAME} %{VERSION} %{RELEASE} %{ARCH}\\n",
        ],
    );
    if res.spawn_failed {
        return Err(Diagnostic::error(
            Domain::Packages,
            "package convergence failed: cannot query the resolved installed set",
        ));
    }
    let mut elements = Vec::new();
    for line in res.stdout.lines() {
        let parts: Vec<&str> = line.split_whitespace().collect();
        if parts.len() >= 4 {
            elements.push(crate::manifest::PackageRecord {
                name: parts[0].to_string(),
                version: parts[1].to_string(),
                release: parts[2].to_string(),
                arch: parts[3].to_string(),
            });
        }
    }
    let mut attrs = std::collections::BTreeMap::new();
    attrs.insert(
        "package_system".to_string(),
        serde_json::Value::String("rpm".to_string()),
    );
    Ok(ScopeWrapper {
        attributes: attrs,
        elements,
    })
}

/// converge-files: write files_write (resolving content via content_ref),
/// delete files_delete excluding RPM-owned, keep-listed, and the syncpoint.
/// In this version converge-files writes and deletes REGULAR FILES only; symlink
/// convergence and type-transition handling are deferred per the spec.
pub fn converge_files(
    runner: &dyn CommandRunner,
    ctx: &TransactionContext,
    diff: &Diff,
    config: &Config,
    keep_list: &HashSet<String>,
) -> Result<(), Diagnostic> {
    // 1. Write declared regular files.
    let content_store = config.content_store.clone().unwrap_or_default();
    for e in &diff.files_write {
        if e.file_type != "file" {
            // Symlink/dir convergence is deferred to a later version.
            continue;
        }
        let content = resolve_content(&content_store, &e.content_ref)?;
        // Verify content hashes to the declared sha256 before writing.
        if !e.sha256.is_empty() {
            let actual = hash_bytes(content.as_bytes());
            if actual != e.sha256 {
                return Err(Diagnostic::error(
                    Domain::Files,
                    format!("written content hash mismatch for {}", e.name),
                ));
            }
        }
        let dest = join_root(&ctx.root, &e.name);
        if let Some(parent) = dest.parent() {
            std::fs::create_dir_all(parent).map_err(|err| {
                Diagnostic::error(
                    Domain::Files,
                    format!("file write failed (mkdir {}): {}", parent.display(), err),
                )
            })?;
        }
        std::fs::write(&dest, content.as_bytes()).map_err(|err| {
            Diagnostic::error(
                Domain::Files,
                format!("file write failed {}: {}", dest.display(), err),
            )
        })?;
        apply_mode(&dest, &e.mode);
    }

    // 2. Delete dropped files, excluding RPM-owned, keep-listed, syncpoint.
    for p in &diff.files_delete {
        if p == SYNCPOINT || keep_list.contains(p) {
            continue;
        }
        if is_rpm_owned(runner, &ctx.root, p) {
            continue;
        }
        let dest = join_root(&ctx.root, p);
        if dest.exists() {
            std::fs::remove_file(&dest).map_err(|err| {
                Diagnostic::error(
                    Domain::Files,
                    format!("file delete failed {}: {}", dest.display(), err),
                )
            })?;
        }
    }
    Ok(())
}

/// converge-units: apply declared unit states offline against the context root.
pub fn converge_units(
    runner: &dyn CommandRunner,
    ctx: &TransactionContext,
    diff: &Diff,
) -> Result<(), Diagnostic> {
    for u in &diff.units_change {
        let verb = match u.state.as_str() {
            "enabled" => "enable",
            "disabled" => "disable",
            "masked" => "mask",
            other => {
                return Err(Diagnostic::error(
                    Domain::Units,
                    format!("offline enablement failed: unknown unit state {:?}", other),
                ))
            }
        };
        let r = runner.run("systemctl", &["--root", &ctx.root, verb, &u.name]);
        if r.spawn_failed || r.code != 0 {
            return Err(Diagnostic::error(
                Domain::Units,
                format!("offline enablement failed: {} {}", verb, u.name),
            ));
        }
    }
    Ok(())
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

fn resolve_content(content_store: &str, content_ref: &str) -> Result<String, Diagnostic> {
    if content_ref.is_empty() {
        return Ok(String::new());
    }
    let mut path = PathBuf::from(content_store);
    path.push(content_ref);
    std::fs::read_to_string(&path).map_err(|e| {
        Diagnostic::error(
            Domain::Files,
            format!("content resolution failed for {}: {}", path.display(), e),
        )
    })
}

fn join_root(root: &str, logical: &str) -> PathBuf {
    let mut p = PathBuf::from(root);
    // logical begins with '/'; strip it so it joins under root.
    p.push(logical.trim_start_matches('/'));
    p
}

fn hash_bytes(bytes: &[u8]) -> String {
    let mut hasher = Sha256::new();
    hasher.update(bytes);
    let digest = hasher.finalize();
    let mut s = String::with_capacity(64);
    for b in digest {
        s.push_str(&format!("{:02x}", b));
    }
    s
}

#[cfg(unix)]
fn apply_mode(path: &Path, mode: &str) {
    use std::os::unix::fs::PermissionsExt;
    if let Ok(parsed) = u32::from_str_radix(mode.trim_start_matches('0'), 8).or_else(|_| {
        if mode.is_empty() {
            Ok(0o644)
        } else {
            u32::from_str_radix(mode, 8)
        }
    }) {
        let _ = std::fs::set_permissions(path, std::fs::Permissions::from_mode(parsed));
    }
}

#[cfg(not(unix))]
fn apply_mode(_path: &Path, _mode: &str) {}

fn is_rpm_owned(runner: &dyn CommandRunner, root: &str, logical: &str) -> bool {
    let r = runner.run("rpm", &["-qf", "--root", root, logical]);
    if r.spawn_failed {
        return false;
    }
    r.code == 0 && !r.stdout.contains("is not owned by any package")
}
