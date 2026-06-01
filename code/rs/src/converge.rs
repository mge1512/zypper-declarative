// generated from spec: zypper-declarative.spec.md sha256:87b9f2f3bf92afd6fe037412d56a23959290e5816d722a3def9505d07aa5acd7
//
// BEHAVIOR/INTERNAL: converge-packages, converge-files, converge-units. These
// apply the intent diff inside a transaction context. Package retrieval is
// delegated to the package manager against the declared pinned repositories; the
// tool performs no direct network fetch of its own.

use crate::error::{Diagnostic, Domain};
use crate::hash::sha256_bytes;
use crate::interfaces::CommandRunner;
use crate::types::*;
use std::collections::HashSet;
use std::path::PathBuf;

/// converge-packages. Ensures repositories, installs and removes packages within
/// the context root, then queries the rpmdb for the resolved installed set (the
/// lock). The returned scope is the rpmdb-reported set, never inferred from file
/// diffs, with all identity fields populated.
pub fn converge_packages(
    runner: &dyn CommandRunner,
    ctx_root: &str,
    diff: &Diff,
    repo_lock: Option<&str>,
) -> Result<PackagesScope, Diagnostic> {
    // Step 1: ensure repositories (declared, or the CONFIG pin if repos_set empty).
    if diff.repos_set.is_empty() {
        if let Some(lock) = repo_lock {
            ensure_repo_pin(runner, ctx_root, lock)?;
        }
    } else {
        for repo in &diff.repos_set {
            ensure_repo(runner, ctx_root, repo)?;
        }
    }

    // Step 2: install.
    if !diff.packages_install.is_empty() {
        let mut args: Vec<String> = vec![
            format!("--root={}", ctx_root),
            "--non-interactive".into(),
            "install".into(),
            "--no-recommends".into(),
        ];
        for p in &diff.packages_install {
            args.push(p.name.clone());
        }
        let argrefs: Vec<&str> = args.iter().map(|s| s.as_str()).collect();
        runner.run("zypper", &argrefs).map_err(|e| {
            Diagnostic::error(
                Domain::Packages,
                format!("package install failed: {}", e.message),
            )
        })?;
    }

    // Step 3: remove.
    if !diff.packages_remove.is_empty() {
        let mut args: Vec<String> = vec![
            format!("--root={}", ctx_root),
            "--non-interactive".into(),
            "remove".into(),
        ];
        for p in &diff.packages_remove {
            args.push(p.name.clone());
        }
        let argrefs: Vec<&str> = args.iter().map(|s| s.as_str()).collect();
        runner.run("zypper", &argrefs).map_err(|e| {
            Diagnostic::error(
                Domain::Packages,
                format!("package remove failed: {}", e.message),
            )
        })?;
    }

    // Step 4: query the rpmdb for the full installed set (the lock).
    let root_arg = format!("--root={}", ctx_root);
    let qargs: Vec<&str> = if ctx_root == "/" {
        vec!["-qa", "--qf", "%{NAME}\t%{VERSION}\t%{RELEASE}\t%{ARCH}\n"]
    } else {
        vec![
            &root_arg,
            "-qa",
            "--qf",
            "%{NAME}\t%{VERSION}\t%{RELEASE}\t%{ARCH}\n",
        ]
    };
    let (stdout, _) = runner.run("rpm", &qargs).map_err(|e| {
        Diagnostic::error(
            Domain::Packages,
            format!("rpmdb query failed: {}", e.message),
        )
    })?;
    let mut scope = PackagesScope::with_attr("package_system", "rpm");
    for line in stdout.lines() {
        let cols: Vec<&str> = line.split('\t').collect();
        if cols.len() >= 4 && !cols[0].is_empty() {
            scope.elements.push(PackageRecord {
                name: cols[0].into(),
                version: cols[1].into(),
                release: cols[2].into(),
                arch: cols[3].into(),
            });
        }
    }
    scope.elements.sort_by(|a, b| {
        (a.name.as_str(), a.arch.as_str()).cmp(&(b.name.as_str(), b.arch.as_str()))
    });
    Ok(scope)
}

fn ensure_repo(
    runner: &dyn CommandRunner,
    ctx_root: &str,
    repo: &RepositoryRecord,
) -> Result<(), Diagnostic> {
    let root_arg = format!("--root={}", ctx_root);
    let args = vec![
        root_arg.as_str(),
        "--non-interactive",
        "addrepo",
        "--check",
        repo.url.as_str(),
        repo.alias.as_str(),
    ];
    runner.run("zypper", &args).map(|_| ()).map_err(|e| {
        Diagnostic::error(
            Domain::Repositories,
            format!(
                "repository configuration failed: {}: {}",
                repo.alias, e.message
            ),
        )
    })
}

fn ensure_repo_pin(
    runner: &dyn CommandRunner,
    ctx_root: &str,
    pin: &str,
) -> Result<(), Diagnostic> {
    let root_arg = format!("--root={}", ctx_root);
    let args = vec![
        root_arg.as_str(),
        "--non-interactive",
        "addrepo",
        "--check",
        pin,
        "config-pin",
    ];
    runner.run("zypper", &args).map(|_| ()).map_err(|e| {
        Diagnostic::error(
            Domain::Repositories,
            format!("repository pin configuration failed: {}", e.message),
        )
    })
}

/// converge-files. Writes declared files (resolving content via content_ref) and
/// deletes only files the declaration dropped, excluding RPM-owned paths, the
/// keep-list, and /etc/etc.syncpoint. v1 writes/deletes regular files; symlink
/// convergence and type-transition handling are deferred (per the spec).
pub fn converge_files(
    runner: &dyn CommandRunner,
    ctx_root: &str,
    diff: &Diff,
    content_store: Option<&str>,
    keep_list: &HashSet<String>,
) -> Result<(), Diagnostic> {
    // Step 1: write.
    for e in &diff.files_write {
        if e.r#type != "file" {
            // v1 converges regular files only; non-file convergence is deferred.
            continue;
        }
        let content = resolve_content(&e.content_ref, content_store)?;
        let actual_sha = sha256_bytes(&content);
        let dest = join_root(ctx_root, &e.name);
        if let Some(parent) = dest.parent() {
            std::fs::create_dir_all(parent).map_err(|err| {
                Diagnostic::error(
                    Domain::Files,
                    format!("file write failed: {}: {}", parent.display(), err),
                )
            })?;
        }
        std::fs::write(&dest, &content).map_err(|err| {
            Diagnostic::error(
                Domain::Files,
                format!("file write failed: {}: {}", dest.display(), err),
            )
        })?;
        apply_mode(&dest, &e.mode)?;
        // verify the written content hashes to e.sha256 (when a digest is declared).
        if !e.sha256.is_empty() && actual_sha != e.sha256 {
            return Err(Diagnostic::error(
                Domain::Files,
                format!(
                    "written content hash mismatch for {}: got {} expected {}",
                    e.name, actual_sha, e.sha256
                ),
            ));
        }
    }

    // Step 2: delete.
    for p in &diff.files_delete {
        if p == SYNCPOINT || keep_list.contains(p) {
            continue;
        }
        if is_rpm_owned(runner, ctx_root, p) {
            continue;
        }
        let dest = join_root(ctx_root, p);
        if dest.exists() {
            std::fs::remove_file(&dest).map_err(|err| {
                Diagnostic::error(
                    Domain::Files,
                    format!("delete failed: {}: {}", dest.display(), err),
                )
            })?;
        }
    }

    Ok(())
}

fn resolve_content(content_ref: &str, content_store: Option<&str>) -> Result<Vec<u8>, Diagnostic> {
    if content_ref.is_empty() {
        return Ok(Vec::new());
    }
    let base = content_store.unwrap_or(".");
    let mut p = PathBuf::from(base);
    p.push(content_ref);
    std::fs::read(&p).map_err(|e| {
        Diagnostic::error(
            Domain::Files,
            format!("content resolution failed: {}: {}", p.display(), e),
        )
    })
}

fn apply_mode(path: &std::path::Path, mode: &str) -> Result<(), Diagnostic> {
    use std::os::unix::fs::PermissionsExt;
    if mode.is_empty() {
        return Ok(());
    }
    let bits = u32::from_str_radix(mode.trim_start_matches('0'), 8)
        .or_else(|_| u32::from_str_radix(mode, 8))
        .unwrap_or(0o644);
    let perms = std::fs::Permissions::from_mode(bits);
    std::fs::set_permissions(path, perms).map_err(|e| {
        Diagnostic::error(
            Domain::Files,
            format!("set mode failed: {}: {}", path.display(), e),
        )
    })
}

fn is_rpm_owned(runner: &dyn CommandRunner, ctx_root: &str, logical: &str) -> bool {
    let root_arg = format!("--root={}", ctx_root);
    let args: Vec<&str> = if ctx_root == "/" {
        vec!["-qf", logical]
    } else {
        vec![&root_arg, "-qf", logical]
    };
    match runner.run("rpm", &args) {
        Ok((stdout, _)) => {
            let line = stdout.lines().next().unwrap_or("").trim();
            !line.is_empty() && !line.contains("not owned")
        }
        Err(e) => {
            let combined = format!("{} {}", e.stdout, e.stderr).to_lowercase();
            !(combined.contains("not owned") || combined.contains("no package owns"))
                && e.status.is_none()
        }
    }
}

/// converge-units. Applies the declared state (enabled, disabled, masked) offline
/// against ctx.root for each ServiceRecord in units_change.
pub fn converge_units(
    runner: &dyn CommandRunner,
    ctx_root: &str,
    diff: &Diff,
) -> Result<(), Diagnostic> {
    let root_arg = format!("--root={}", ctx_root);
    for u in &diff.units_change {
        let verb = match u.state.as_str() {
            "enabled" => "enable",
            "disabled" => "disable",
            "masked" => "mask",
            other => {
                return Err(Diagnostic::error(
                    Domain::Units,
                    format!("unknown declared unit state '{}' for {}", other, u.name),
                ))
            }
        };
        let args = vec![root_arg.as_str(), verb, u.name.as_str()];
        runner.run("systemctl", &args).map(|_| ()).map_err(|e| {
            Diagnostic::error(
                Domain::Units,
                format!("offline enablement failed for {}: {}", u.name, e.message),
            )
        })?;
    }
    Ok(())
}

const SYNCPOINT: &str = "/etc/etc.syncpoint";

fn join_root(root: &str, logical: &str) -> PathBuf {
    let mut p = PathBuf::from(root);
    // logical begins with '/', so strip it before joining.
    p.push(logical.trim_start_matches('/'));
    p
}
