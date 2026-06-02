// generated from spec: zypper-declarative.spec.md
// sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
//
// BEHAVIOR/INTERNAL: converge-packages, converge-files, converge-units. These
// apply the intent diff inside the transaction context by driving zypper /
// systemctl and writing /etc, and return the resolved packages scope (the lock).
// Convergence is host-only (privileged, real transaction); the functions return
// domain-tagged diagnostics so the verb layer maps them to exit codes.

use crate::diff::{Diff, SYNCPOINT};
use crate::error::{Diagnostic, Domain};
use crate::interfaces::CommandRunner;
use crate::manifest::hash::sha256_bytes;
use crate::manifest::PackagesScope;
use crate::state::packages;
use crate::txn::TransactionContext;
use std::collections::HashSet;
use std::path::Path;

/// converge-packages: configure repos, install/remove packages, return the lock.
pub fn converge_packages(
    runner: &dyn CommandRunner,
    ctx: &TransactionContext,
    diff: &Diff,
) -> Result<PackagesScope, Diagnostic> {
    // 1. ensure repositories configured (delegated to zypper).
    for repo in &diff.repos_set {
        let res = runner.run(
            "zypper",
            &[
                "--root",
                &ctx.root,
                "ar",
                "--no-gpgcheck",
                &repo.url,
                &repo.alias,
            ],
        );
        if res.spawn_failed {
            return Err(Diagnostic::error(
                Domain::Repositories,
                format!("zypper unavailable: {}", res.stderr.trim()),
            ));
        }
        // re-adding an existing repo is not fatal; only a spawn failure is.
    }

    // 2. install
    if !diff.packages_install.is_empty() {
        let mut args = vec![
            "--root".to_string(),
            ctx.root.clone(),
            "in".to_string(),
            "-y".to_string(),
        ];
        for p in &diff.packages_install {
            args.push(p.name.clone());
        }
        let argref: Vec<&str> = args.iter().map(|s| s.as_str()).collect();
        let res = runner.run("zypper", &argref);
        if res.spawn_failed || !res.success {
            return Err(Diagnostic::error(
                Domain::Packages,
                format!("package install failed: {}", res.stderr.trim()),
            ));
        }
    }

    // 3. remove
    if !diff.packages_remove.is_empty() {
        let mut args = vec![
            "--root".to_string(),
            ctx.root.clone(),
            "rm".to_string(),
            "-y".to_string(),
        ];
        for p in &diff.packages_remove {
            args.push(p.name.clone());
        }
        let argref: Vec<&str> = args.iter().map(|s| s.as_str()).collect();
        let res = runner.run("zypper", &argref);
        if res.spawn_failed || !res.success {
            return Err(Diagnostic::error(
                Domain::Packages,
                format!("package remove failed: {}", res.stderr.trim()),
            ));
        }
    }

    // 4. query the rpmdb under ctx.root for the resolved set (the lock).
    match packages::read_packages(runner, &ctx.root) {
        packages::PackagesResult::Records(recs) => {
            let mut scope = PackagesScope::with_attr("package_system", "rpm");
            scope.elements = recs;
            Ok(scope)
        }
        packages::PackagesResult::Unreadable(src) => Err(Diagnostic::error(
            Domain::Packages,
            format!("could not read resolved package set: {}", src),
        )),
    }
}

/// converge-files: write declared regular files and delete dropped ones.
/// (Symlink convergence and type-transition handling are reserved for a later
/// version, per the spec's converge-files note.)
pub fn converge_files(
    ctx: &TransactionContext,
    diff: &Diff,
    keep_list: &HashSet<String>,
    content_store: Option<&str>,
    rpm_owned: &dyn Fn(&str) -> bool,
) -> Result<(), Diagnostic> {
    // 1. writes (regular files only in this version)
    for rec in &diff.files_write {
        if rec.r#type != "file" {
            continue; // symlink/dir convergence reserved for a later version
        }
        let content = resolve_content(content_store, rec)?;
        let dest = join_root(&ctx.root, &rec.name);
        if let Some(parent) = dest.parent() {
            std::fs::create_dir_all(parent).map_err(|e| {
                Diagnostic::error(
                    Domain::Files,
                    format!("cannot create {}: {}", parent.display(), e),
                )
            })?;
        }
        std::fs::write(&dest, &content).map_err(|e| {
            Diagnostic::error(
                Domain::Files,
                format!("cannot write {}: {}", dest.display(), e),
            )
        })?;
        apply_mode(&dest, &rec.mode)?;
        // verify written content hashes to declared sha256
        if !rec.sha256.is_empty() {
            let got = sha256_bytes(&content);
            if got != rec.sha256 {
                return Err(Diagnostic::error(
                    Domain::Files,
                    format!(
                        "written {} hashes to {}, expected {}",
                        rec.name, got, rec.sha256
                    ),
                ));
            }
        }
    }

    // 2. deletes (skip RPM-owned, keep-listed, syncpoint)
    for path in &diff.files_delete {
        if path == SYNCPOINT || keep_list.contains(path) || rpm_owned(path) {
            continue;
        }
        let dest = join_root(&ctx.root, path);
        if dest.exists() {
            std::fs::remove_file(&dest).map_err(|e| {
                Diagnostic::error(
                    Domain::Files,
                    format!("cannot delete {}: {}", dest.display(), e),
                )
            })?;
        }
    }
    Ok(())
}

fn resolve_content(
    content_store: Option<&str>,
    rec: &crate::manifest::ManagedFileRecord,
) -> Result<Vec<u8>, Diagnostic> {
    if rec.content_ref.is_empty() {
        // No content reference: an empty body is written (a declared file may
        // assert presence with empty content).
        return Ok(Vec::new());
    }
    let store = content_store.ok_or_else(|| {
        Diagnostic::error(
            Domain::Files,
            format!(
                "{} has content_ref {} but no content-store is set",
                rec.name, rec.content_ref
            ),
        )
    })?;
    // content_ref form: "sha256/<digest>"
    let rel = rec.content_ref.trim_start_matches('/');
    let blob = Path::new(store).join(rel);
    std::fs::read(&blob).map_err(|e| {
        Diagnostic::error(
            Domain::Files,
            format!("cannot read content {}: {}", blob.display(), e),
        )
    })
}

fn apply_mode(dest: &Path, mode: &str) -> Result<(), Diagnostic> {
    if mode.is_empty() {
        return Ok(());
    }
    use std::os::unix::fs::PermissionsExt;
    let parsed = u32::from_str_radix(mode, 8)
        .map_err(|_| Diagnostic::error(Domain::Files, format!("invalid mode {:?}", mode)))?;
    std::fs::set_permissions(dest, std::fs::Permissions::from_mode(parsed)).map_err(|e| {
        Diagnostic::error(
            Domain::Files,
            format!("cannot chmod {}: {}", dest.display(), e),
        )
    })
}

fn join_root(root: &str, logical: &str) -> std::path::PathBuf {
    Path::new(root).join(logical.trim_start_matches('/'))
}

/// converge-units: apply declared enablement OFFLINE against ctx.root.
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
                    Domain::Services,
                    format!("unknown declared state {:?} for {}", other, u.name),
                ))
            }
        };
        let res = runner.run("systemctl", &["--root", &ctx.root, verb, &u.name]);
        if res.spawn_failed || !res.success {
            return Err(Diagnostic::error(
                Domain::Services,
                format!(
                    "offline {} of {} failed: {}",
                    verb,
                    u.name,
                    res.stderr.trim()
                ),
            ));
        }
    }
    Ok(())
}
