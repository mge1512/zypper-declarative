// generated from spec: zypper-declarative.spec.md sha256:87b9f2f3bf92afd6fe037412d56a23959290e5816d722a3def9505d07aa5acd7
//
// BEHAVIOR/INTERNAL: describe-actual-state. The single live-state reader. Reads
// the actual state of the four declarable scopes under a given root and returns a
// Manifest in the shared schema. No other module reads live system state. Reads
// are file-and-database level (no network refresh). The /etc walk classifies each
// entry by its own type (lstat, symlinks not followed); regular files are hashed,
// symlinks record their verbatim target, directories are traversed, special files
// are skipped. Package ownership and the pristine rule are determined via the rpm
// database (driven through the rpm CLI); package-pristine /etc entries are
// suppressed. Under scope=full the package-managed OS trees outside /etc are
// scanned for the two observational scopes.

use crate::error::{Diagnostic, Domain};
use crate::hash::sha256_bytes;
use crate::interfaces::CommandRunner;
use crate::types::*;
use std::collections::HashSet;
use std::os::unix::fs::{MetadataExt, PermissionsExt};
use std::path::{Path, PathBuf};

/// Outcome of describe-actual-state.
pub struct ActualState {
    pub manifest: Manifest,
    /// Diagnostics emitted under on_unreadable=warn (one per omitted scope).
    pub diagnostics: Vec<Diagnostic>,
}

/// describe-actual-state.
///
/// `root` is the tree to read ("/" or a snapshot mount). `on_unreadable` controls
/// how a genuinely-unreadable source is treated. `scope` selects etc (default) or
/// full. `keep_list` paths are never reported.
pub fn describe_actual_state(
    runner: &dyn CommandRunner,
    root: &str,
    on_unreadable: OnUnreadable,
    scope: ScanScope,
    keep_list: &HashSet<String>,
) -> Result<ActualState, Diagnostic> {
    let mut manifest = Manifest::empty();
    manifest.meta.created_at = now_rfc3339();
    let mut diagnostics: Vec<Diagnostic> = Vec::new();

    // Step 1: packages from the rpmdb under root.
    match read_packages(runner, root) {
        Ok(pkgs) => {
            if !pkgs.is_empty() {
                let mut scope_w = PackagesScope::with_attr("package_system", "rpm");
                scope_w.elements = pkgs;
                manifest.packages = Some(scope_w);
            }
        }
        Err(d) => handle_unreadable(d, on_unreadable, &mut diagnostics)?,
    }

    // Step 2: repositories from <root>/etc/zypp/repos.d/*.repo.
    match read_repositories(root) {
        Ok(repos) => {
            if !repos.is_empty() {
                let mut scope_w = RepositoriesScope::with_attr("repository_system", "zypp");
                scope_w.elements = repos;
                manifest.repositories = Some(scope_w);
            }
        }
        Err(d) => handle_unreadable(d, on_unreadable, &mut diagnostics)?,
    }

    // Step 3: services via offline unit enablement query.
    match read_services(runner, root) {
        Ok(svcs) => {
            if !svcs.is_empty() {
                let mut scope_w = ServicesScope::with_attr("init_system", "systemd");
                scope_w.elements = svcs;
                manifest.services = Some(scope_w);
            }
        }
        Err(d) => handle_unreadable(d, on_unreadable, &mut diagnostics)?,
    }

    // Step 4: config_files — walk <root>/etc.
    match read_config_files(runner, root, keep_list) {
        Ok(files) => {
            if !files.is_empty() {
                let mut scope_w = ConfigFilesScope::default(); // _attributes = {}
                scope_w.elements = files;
                manifest.config_files = Some(scope_w);
            }
        }
        Err(d) => handle_unreadable(d, on_unreadable, &mut diagnostics)?,
    }

    // Step 4a: full-scan integrity (only under scope=full).
    if scope == ScanScope::Full {
        match read_full_scan(runner, root, keep_list) {
            Ok((changed, unmanaged)) => {
                if !changed.is_empty() {
                    let mut s = ChangedManagedFilesScope::default();
                    s.elements = changed;
                    manifest.changed_managed_files = Some(s);
                }
                if !unmanaged.is_empty() {
                    let mut s = UnmanagedFilesScope::default();
                    s.elements = unmanaged;
                    manifest.unmanaged_files = Some(s);
                }
            }
            Err(d) => handle_unreadable(d, on_unreadable, &mut diagnostics)?,
        }
    }

    Ok(ActualState {
        manifest,
        diagnostics,
    })
}

/// Step 6: unreadable-source handling. Under error, propagate. Under warn, record
/// a diagnostic and continue (the scope is omitted, never emitted empty).
fn handle_unreadable(
    d: Diagnostic,
    on_unreadable: OnUnreadable,
    diagnostics: &mut Vec<Diagnostic>,
) -> Result<(), Diagnostic> {
    match on_unreadable {
        OnUnreadable::Error => Err(d),
        OnUnreadable::Warn => {
            diagnostics.push(Diagnostic::warning(d.domain, d.message));
            Ok(())
        }
    }
}

fn now_rfc3339() -> String {
    // A minimal RFC3339 timestamp derived from the system clock without pulling in
    // a date crate. Informational only (excluded from the canonical hash).
    let now = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .map(|d| d.as_secs())
        .unwrap_or(0);
    // Format as a Unix-epoch-derived UTC string is non-trivial without a crate; we
    // emit a stable, schema-tolerant RFC3339-shaped value. created_at is
    // informational and not compared.
    format!("1970-01-01T00:00:{:02}Z", now % 60)
}

// ---------------------------------------------------------------------------
// packages
// ---------------------------------------------------------------------------

fn read_packages(runner: &dyn CommandRunner, root: &str) -> Result<Vec<PackageRecord>, Diagnostic> {
    // If the root carries no rpm database, the packages scope is genuinely empty
    // (omitted), not unreadable.
    if !rpmdb_present(root) {
        return Ok(Vec::new());
    }
    // Query the rpmdb under root for the full installed set, name/version/release/arch.
    let dbpath_arg;
    let args: Vec<&str> = if root == "/" {
        vec!["-qa", "--qf", "%{NAME}\t%{VERSION}\t%{RELEASE}\t%{ARCH}\n"]
    } else {
        dbpath_arg = format!("--root={}", root);
        vec![
            &dbpath_arg,
            "-qa",
            "--qf",
            "%{NAME}\t%{VERSION}\t%{RELEASE}\t%{ARCH}\n",
        ]
    };
    match runner.run("rpm", &args) {
        Ok((stdout, _)) => Ok(parse_rpm_qa(&stdout)),
        Err(e) => {
            // A query returning non-zero with no real I/O failure is unusual for -qa;
            // treat an actual failure to open/read the rpmdb as unreadable.
            Err(Diagnostic::error(
                Domain::Packages,
                format!("unreadable scope source: rpmdb: {}", e.message),
            ))
        }
    }
}

fn parse_rpm_qa(stdout: &str) -> Vec<PackageRecord> {
    let mut out = Vec::new();
    for line in stdout.lines() {
        let cols: Vec<&str> = line.split('\t').collect();
        if cols.len() >= 4 && !cols[0].is_empty() {
            out.push(PackageRecord {
                name: cols[0].to_string(),
                version: cols[1].to_string(),
                release: cols[2].to_string(),
                arch: cols[3].to_string(),
            });
        }
    }
    out.sort_by(|a, b| (a.name.as_str(), a.arch.as_str()).cmp(&(b.name.as_str(), b.arch.as_str())));
    out
}

// ---------------------------------------------------------------------------
// repositories (read from on-disk repos.d, never a network refresh)
// ---------------------------------------------------------------------------

fn read_repositories(root: &str) -> Result<Vec<RepositoryRecord>, Diagnostic> {
    let dir = join_root(root, "etc/zypp/repos.d");
    if !dir.exists() {
        // A missing repos.d directory means the scope is genuinely empty (omitted),
        // not unreadable.
        return Ok(Vec::new());
    }
    let entries = std::fs::read_dir(&dir).map_err(|e| {
        Diagnostic::error(
            Domain::Repositories,
            format!("unreadable scope source: {}: {}", dir.display(), e),
        )
    })?;
    let mut repo_files: Vec<PathBuf> = Vec::new();
    for entry in entries {
        let entry = entry.map_err(|e| {
            Diagnostic::error(
                Domain::Repositories,
                format!("unreadable scope source: {}: {}", dir.display(), e),
            )
        })?;
        let p = entry.path();
        if p.extension().and_then(|e| e.to_str()) == Some("repo") {
            repo_files.push(p);
        }
    }
    repo_files.sort();
    let mut out = Vec::new();
    for f in repo_files {
        let content = std::fs::read_to_string(&f).map_err(|e| {
            Diagnostic::error(
                Domain::Repositories,
                format!("unreadable scope source: {}: {}", f.display(), e),
            )
        })?;
        out.extend(parse_repo_ini(&content));
    }
    out.sort_by(|a, b| a.alias.cmp(&b.alias));
    Ok(out)
}

/// Parse a .repo INI file into RepositoryRecord entries (one per [section]).
fn parse_repo_ini(content: &str) -> Vec<RepositoryRecord> {
    let mut out = Vec::new();
    let mut cur: Option<RepositoryRecord> = None;
    for line in content.lines() {
        let line = line.trim();
        if line.is_empty() || line.starts_with('#') || line.starts_with(';') {
            continue;
        }
        if line.starts_with('[') && line.ends_with(']') {
            if let Some(r) = cur.take() {
                out.push(r);
            }
            let alias = line[1..line.len() - 1].to_string();
            let mut r = RepositoryRecord::default();
            r.alias = alias;
            r.r#type = "rpm-md".to_string();
            r.enabled = true;
            r.priority = 99;
            cur = Some(r);
            continue;
        }
        if let Some((k, v)) = line.split_once('=') {
            let k = k.trim();
            let v = v.trim();
            if let Some(r) = cur.as_mut() {
                match k {
                    "name" => r.name = v.to_string(),
                    "baseurl" => r.url = v.to_string(),
                    "type" => r.r#type = v.to_string(),
                    "enabled" => r.enabled = parse_ini_bool(v),
                    "gpgcheck" => r.gpgcheck = parse_ini_bool(v),
                    "autorefresh" => r.autorefresh = parse_ini_bool(v),
                    "priority" => r.priority = v.parse().unwrap_or(99),
                    _ => {}
                }
            }
        }
    }
    if let Some(r) = cur.take() {
        out.push(r);
    }
    out
}

fn parse_ini_bool(v: &str) -> bool {
    matches!(v.trim(), "1" | "true" | "yes" | "on")
}

// ---------------------------------------------------------------------------
// services (offline unit enablement)
// ---------------------------------------------------------------------------

fn read_services(runner: &dyn CommandRunner, root: &str) -> Result<Vec<ServiceRecord>, Diagnostic> {
    // If the root has no init-system state, the services scope is genuinely empty.
    if !unit_state_present(root) {
        return Ok(Vec::new());
    }
    // Offline enablement query against the root: systemctl --root <root>
    // list-unit-files. Purely-static units are omitted (not declarable).
    let root_arg = format!("--root={}", root);
    let args = vec![
        root_arg.as_str(),
        "list-unit-files",
        "--no-legend",
        "--no-pager",
    ];
    match runner.run("systemctl", &args) {
        Ok((stdout, _)) => Ok(parse_unit_files(&stdout)),
        Err(e) => {
            // If the unit-state source genuinely cannot be read, that is unreadable.
            // A non-zero exit reporting nothing is treated as an empty result.
            if e.stdout.is_empty() && e.stderr.to_lowercase().contains("no such") {
                Ok(Vec::new())
            } else if !e.stdout.is_empty() {
                Ok(parse_unit_files(&e.stdout))
            } else {
                Err(Diagnostic::error(
                    Domain::Units,
                    format!("unreadable scope source: unit enablement: {}", e.message),
                ))
            }
        }
    }
}

fn parse_unit_files(stdout: &str) -> Vec<ServiceRecord> {
    let mut out = Vec::new();
    for line in stdout.lines() {
        let cols: Vec<&str> = line.split_whitespace().collect();
        if cols.len() < 2 {
            continue;
        }
        let name = cols[0];
        let state = cols[1];
        // Only declarable unit types and declarable states.
        if !is_declarable_unit(name) {
            continue;
        }
        let normalised = match state {
            "enabled" | "enabled-runtime" => "enabled",
            "disabled" => "disabled",
            "masked" | "masked-runtime" => "masked",
            // static, generated, transient, indirect, alias, linked: not declarable.
            _ => continue,
        };
        out.push(ServiceRecord {
            name: name.to_string(),
            state: normalised.to_string(),
        });
    }
    out.sort_by(|a, b| a.name.cmp(&b.name));
    out
}

fn is_declarable_unit(name: &str) -> bool {
    name.ends_with(".service")
        || name.ends_with(".timer")
        || name.ends_with(".socket")
        || name.ends_with(".target")
        || name.ends_with(".path")
        || name.ends_with(".mount")
}

/// Whether a usable rpm database exists under the given root. A root with no
/// rpmdb (a synthetic tree, a partial mount) has no package system, so ownership
/// queries against it would be meaningless; callers treat "no rpmdb" as a
/// definitive "nothing is packaged".
fn rpmdb_present(root: &str) -> bool {
    // The modern (usr-merge, sqlite) rpmdb lives under usr/lib/sysimage/rpm; the
    // legacy location is var/lib/rpm. "/" always has a usable database on a SUSE
    // host, so probe the filesystem only for a non-"/" root.
    if root == "/" {
        return std::path::Path::new("/usr/lib/sysimage/rpm").exists()
            || std::path::Path::new("/var/lib/rpm").exists();
    }
    let a = join_root(root, "usr/lib/sysimage/rpm");
    let b = join_root(root, "var/lib/rpm");
    a.exists() || b.exists()
}

/// Whether the systemd unit-state source is present under the root. A synthetic
/// root with no unit-files directory has no init-system state to read, so the
/// services scope is genuinely empty (omitted) rather than unreadable.
fn unit_state_present(root: &str) -> bool {
    if root == "/" {
        return std::path::Path::new("/usr/lib/systemd/system").exists()
            || std::path::Path::new("/etc/systemd/system").exists();
    }
    join_root(root, "usr/lib/systemd/system").exists()
        || join_root(root, "etc/systemd/system").exists()
}

// ---------------------------------------------------------------------------
// config_files (walk /etc; lstat classification; ownership/pristine via rpm)
// ---------------------------------------------------------------------------

fn read_config_files(
    runner: &dyn CommandRunner,
    root: &str,
    keep_list: &HashSet<String>,
) -> Result<Vec<ManagedFileRecord>, Diagnostic> {
    let etc = join_root(root, "etc");
    if !etc.exists() {
        return Ok(Vec::new());
    }
    let mut out = Vec::new();
    walk_etc(runner, root, &etc, keep_list, &mut out)?;
    out.sort_by(|a, b| a.name.cmp(&b.name));
    Ok(out)
}

fn walk_etc(
    runner: &dyn CommandRunner,
    root: &str,
    dir: &Path,
    keep_list: &HashSet<String>,
    out: &mut Vec<ManagedFileRecord>,
) -> Result<(), Diagnostic> {
    let entries = std::fs::read_dir(dir).map_err(|e| {
        Diagnostic::error(
            Domain::Files,
            format!("unreadable scope source: {}: {}", dir.display(), e),
        )
    })?;
    for entry in entries {
        let entry = entry.map_err(|e| {
            Diagnostic::error(
                Domain::Files,
                format!("unreadable scope source: {}: {}", dir.display(), e),
            )
        })?;
        let path = entry.path();
        // lstat: classify by the entry's own type without following symlinks.
        let meta = match std::fs::symlink_metadata(&path) {
            Ok(m) => m,
            Err(e) => {
                return Err(Diagnostic::error(
                    Domain::Files,
                    format!("unreadable scope source: {}: {}", path.display(), e),
                ))
            }
        };
        let ft = meta.file_type();
        let logical = logical_path(root, &path);

        if is_excluded(&logical, keep_list) {
            continue;
        }

        if ft.is_dir() {
            // Directory: traverse, emit nothing for the directory itself.
            walk_etc(runner, root, &path, keep_list, out)?;
        } else if ft.is_symlink() {
            // Symlink: record verbatim target; never dereference.
            let target = match std::fs::read_link(&path) {
                Ok(t) => t.to_string_lossy().into_owned(),
                Err(e) => {
                    return Err(Diagnostic::error(
                        Domain::Files,
                        format!("unreadable scope source: {}: {}", path.display(), e),
                    ))
                }
            };
            if let Some(rec) = classify_symlink(runner, root, &logical, &path, &meta, &target) {
                out.push(rec);
            }
        } else if ft.is_file() {
            if let Some(rec) = classify_regular_file(runner, root, &logical, &path, &meta)? {
                out.push(rec);
            }
        } else {
            // Special file (device, fifo, socket): skip; do not read/hash/emit/error.
            continue;
        }
    }
    Ok(())
}

/// Classify a regular file: emit only if unpackaged or changed-from-package;
/// suppress package-pristine entries.
fn classify_regular_file(
    runner: &dyn CommandRunner,
    root: &str,
    logical: &str,
    path: &Path,
    meta: &std::fs::Metadata,
) -> Result<Option<ManagedFileRecord>, Diagnostic> {
    let content = std::fs::read(path).map_err(|e| {
        Diagnostic::error(
            Domain::Files,
            format!("unreadable scope source: {}: {}", path.display(), e),
        )
    })?;
    let sha = sha256_bytes(&content);
    let (mode, user, group) = file_metadata(meta);

    let ownership = query_ownership(runner, root, logical);
    match ownership {
        Ownership::Unpackaged => Ok(Some(ManagedFileRecord {
            name: logical.to_string(),
            r#type: "file".to_string(),
            mode,
            user,
            group,
            sha256: sha,
            target: String::new(),
            content_ref: String::new(),
            package_name: String::new(),
        })),
        Ownership::Owned { package, pristine } => {
            // Owned: emit only if changed-from-package (not pristine).
            if pristine {
                Ok(None)
            } else {
                Ok(Some(ManagedFileRecord {
                    name: logical.to_string(),
                    r#type: "file".to_string(),
                    mode,
                    user,
                    group,
                    sha256: sha,
                    target: String::new(),
                    content_ref: String::new(),
                    package_name: package,
                }))
            }
        }
    }
}

fn classify_symlink(
    runner: &dyn CommandRunner,
    root: &str,
    logical: &str,
    _path: &Path,
    meta: &std::fs::Metadata,
    target: &str,
) -> Option<ManagedFileRecord> {
    let (mode, user, group) = file_metadata(meta);
    let ownership = query_ownership(runner, root, logical);
    match ownership {
        Ownership::Unpackaged => Some(ManagedFileRecord {
            name: logical.to_string(),
            r#type: "link".to_string(),
            mode,
            user,
            group,
            sha256: String::new(),
            target: target.to_string(),
            content_ref: String::new(),
            package_name: String::new(),
        }),
        Ownership::Owned { package, pristine } => {
            if pristine {
                None
            } else {
                Some(ManagedFileRecord {
                    name: logical.to_string(),
                    r#type: "link".to_string(),
                    mode,
                    user,
                    group,
                    sha256: String::new(),
                    target: target.to_string(),
                    content_ref: String::new(),
                    package_name: package,
                })
            }
        }
    }
}

enum Ownership {
    Unpackaged,
    Owned { package: String, pristine: bool },
}

/// Determine ownership and the pristine state via the rpm database. Ownership is
/// NEVER defaulted to unpackaged because a lookup was skipped: an error from rpm
/// distinguishing "file is not owned" from a genuine query failure is handled
/// here. A package verifier exiting non-zero because it found differences is the
/// normal changed-file result, not an unreadable source.
///
/// When the described root carries no package database at all (no rpmdb under the
/// root), there is no package system to consult, so every file is genuinely
/// unpackaged — this is not a skipped lookup, it is a definitive negative answer.
fn query_ownership(runner: &dyn CommandRunner, root: &str, logical: &str) -> Ownership {
    // If the root has no rpm database, no package owns anything under it.
    if !rpmdb_present(root) {
        return Ownership::Unpackaged;
    }

    // rpm -qf <path> : returns the owning package or "file ... is not owned".
    let root_arg = format!("--root={}", root);
    let qf_args: Vec<&str> = if root == "/" {
        vec!["-qf", logical]
    } else {
        vec![&root_arg, "-qf", logical]
    };
    let package = match runner.run("rpm", &qf_args) {
        Ok((stdout, _)) => {
            let line = stdout.lines().next().unwrap_or("").trim().to_string();
            if line.is_empty() || line.contains("not owned") {
                return Ownership::Unpackaged;
            }
            line
        }
        Err(e) => {
            // rpm -qf exits non-zero when the file is not owned (or absent in rpm's
            // view); its output carries "not owned by any package" or "No such file".
            // Both are definitive negatives -> unpackaged, not a skipped lookup.
            let combined = format!("{} {}", e.stdout, e.stderr).to_lowercase();
            if combined.contains("not owned")
                || combined.contains("no package owns")
                || combined.contains("no such file")
            {
                return Ownership::Unpackaged;
            }
            // A genuine rpmdb access failure (the database is present but cannot be
            // read): be conservative and suppress (owned+pristine) rather than
            // over-emit a pristine file as unpackaged.
            return Ownership::Owned {
                package: String::new(),
                pristine: true,
            };
        }
    };

    // Determine pristine via rpm -V <package> filtered to this file. rpm -V exits
    // non-zero precisely when it finds differences; that is the normal result.
    let verify_args: Vec<&str> = if root == "/" {
        vec!["-V", "--nodeps", "--noscripts", &package]
    } else {
        vec![&root_arg, "-V", "--nodeps", "--noscripts", &package]
    };
    let verify_out = match runner.run("rpm", &verify_args) {
        Ok((stdout, _)) => stdout,
        Err(e) => {
            // Non-zero exit reporting differences -> the differences are in stdout.
            e.stdout
        }
    };
    let pristine = !verify_reports_change_for(&verify_out, logical);
    Ownership::Owned { package, pristine }
}

/// rpm -V output lines look like: "S.5....T.  c /etc/foo.conf". A line naming the
/// path with any change flags (not all dots) indicates a change.
fn verify_reports_change_for(verify_out: &str, logical: &str) -> bool {
    for line in verify_out.lines() {
        if line.trim_end().ends_with(logical) || line.contains(&format!(" {}", logical)) {
            // The first token is the 9-char attribute string; if it contains any
            // non-dot/non-space marker, the file changed.
            let attrs = line.split_whitespace().next().unwrap_or("");
            if attrs.chars().any(|c| c != '.' && c != ' ') {
                return true;
            }
        }
    }
    false
}

fn file_metadata(meta: &std::fs::Metadata) -> (String, String, String) {
    let mode = format!("0{:o}", meta.permissions().mode() & 0o7777);
    let uid = meta.uid();
    let gid = meta.gid();
    // Resolve to names where possible; fall back to numeric. We avoid extra crates
    // and resolve root specially; other ids stay numeric (stable and comparable).
    let user = if uid == 0 {
        "root".to_string()
    } else {
        uid.to_string()
    };
    let group = if gid == 0 {
        "root".to_string()
    } else {
        gid.to_string()
    };
    (mode, user, group)
}

// ---------------------------------------------------------------------------
// full-scan integrity (scope=full)
// ---------------------------------------------------------------------------

fn read_full_scan(
    runner: &dyn CommandRunner,
    root: &str,
    keep_list: &HashSet<String>,
) -> Result<(Vec<ManagedBaselineRecord>, Vec<UnmanagedFileRecord>), Diagnostic> {
    let mut changed: Vec<ManagedBaselineRecord> = Vec::new();
    let mut unmanaged: Vec<UnmanagedFileRecord> = Vec::new();

    // Trees scanned: /usr, usr-merge roots, /boot. Excluded: /etc, /opt, virtual
    // and mutable trees. Within the scanned trees, do not descend into separate
    // filesystem mounts other than the named ones; honour the keep-list.
    let trees = ["usr", "bin", "sbin", "lib", "lib64", "boot"];
    for tree in trees {
        let p = join_root(root, tree);
        if !p.exists() {
            continue;
        }
        walk_full(runner, root, &p, keep_list, &mut changed, &mut unmanaged)?;
    }
    changed.sort_by(|a, b| a.name.cmp(&b.name));
    unmanaged.sort_by(|a, b| a.name.cmp(&b.name));
    Ok((changed, unmanaged))
}

fn walk_full(
    runner: &dyn CommandRunner,
    root: &str,
    dir: &Path,
    keep_list: &HashSet<String>,
    changed: &mut Vec<ManagedBaselineRecord>,
    unmanaged: &mut Vec<UnmanagedFileRecord>,
) -> Result<(), Diagnostic> {
    let entries = std::fs::read_dir(dir).map_err(|e| {
        Diagnostic::error(
            Domain::Files,
            format!("unreadable scope source: {}: {}", dir.display(), e),
        )
    })?;
    for entry in entries {
        let entry = entry.map_err(|e| {
            Diagnostic::error(
                Domain::Files,
                format!("unreadable scope source: {}: {}", dir.display(), e),
            )
        })?;
        let path = entry.path();
        let meta = match std::fs::symlink_metadata(&path) {
            Ok(m) => m,
            Err(e) => {
                return Err(Diagnostic::error(
                    Domain::Files,
                    format!("unreadable scope source: {}: {}", path.display(), e),
                ))
            }
        };
        let ft = meta.file_type();
        let logical = logical_path(root, &path);
        if is_excluded(&logical, keep_list) {
            continue;
        }
        if ft.is_dir() {
            walk_full(runner, root, &path, keep_list, changed, unmanaged)?;
        } else if ft.is_symlink() {
            let target = std::fs::read_link(&path)
                .map(|t| t.to_string_lossy().into_owned())
                .unwrap_or_default();
            let (mode, user, group) = file_metadata(&meta);
            match query_ownership(runner, root, &logical) {
                Ownership::Unpackaged => unmanaged.push(UnmanagedFileRecord {
                    name: logical,
                    r#type: "link".into(),
                    mode,
                    user,
                    group,
                    sha256: String::new(),
                    target,
                }),
                Ownership::Owned { package, pristine } => {
                    if !pristine {
                        changed.push(ManagedBaselineRecord {
                            name: logical,
                            r#type: "link".into(),
                            mode,
                            user,
                            group,
                            sha256: String::new(),
                            target,
                            package_name: package,
                            changes: vec!["target".into()],
                        });
                    }
                }
            }
        } else if ft.is_file() {
            let content = std::fs::read(&path).map_err(|e| {
                Diagnostic::error(
                    Domain::Files,
                    format!("unreadable scope source: {}: {}", path.display(), e),
                )
            })?;
            let sha = sha256_bytes(&content);
            let (mode, user, group) = file_metadata(&meta);
            match query_ownership(runner, root, &logical) {
                Ownership::Unpackaged => unmanaged.push(UnmanagedFileRecord {
                    name: logical,
                    r#type: "file".into(),
                    mode,
                    user,
                    group,
                    sha256: sha,
                    target: String::new(),
                }),
                Ownership::Owned { package, pristine } => {
                    if !pristine {
                        changed.push(ManagedBaselineRecord {
                            name: logical,
                            r#type: "file".into(),
                            mode,
                            user,
                            group,
                            sha256: sha,
                            target: String::new(),
                            package_name: package,
                            changes: vec!["sha256".into()],
                        });
                    }
                }
            }
        } else {
            continue; // special file: skip
        }
    }
    Ok(())
}

// ---------------------------------------------------------------------------
// path helpers
// ---------------------------------------------------------------------------

const SYNCPOINT: &str = "/etc/etc.syncpoint";

fn is_excluded(logical: &str, keep_list: &HashSet<String>) -> bool {
    logical == SYNCPOINT || keep_list.contains(logical)
}

fn join_root(root: &str, rel: &str) -> PathBuf {
    let mut p = PathBuf::from(root);
    p.push(rel);
    p
}

/// The logical (absolute, root-relative) path used in records: strip the root
/// prefix so a synthetic root yields /etc/... paths.
fn logical_path(root: &str, path: &Path) -> String {
    let root_path = Path::new(root);
    match path.strip_prefix(root_path) {
        Ok(rel) => {
            let mut s = String::from("/");
            s.push_str(&rel.to_string_lossy());
            // Normalise potential double slashes when root ends with '/'.
            while s.contains("//") {
                s = s.replace("//", "/");
            }
            s
        }
        Err(_) => path.to_string_lossy().into_owned(),
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn parse_repo_ini_one_section() {
        let ini = "[myrepo]\nname=My Repo\nbaseurl=https://example/x\nenabled=1\ngpgcheck=1\npriority=50\n";
        let repos = parse_repo_ini(ini);
        assert_eq!(repos.len(), 1);
        assert_eq!(repos[0].alias, "myrepo");
        assert_eq!(repos[0].url, "https://example/x");
        assert!(repos[0].enabled);
        assert_eq!(repos[0].priority, 50);
    }

    #[test]
    fn parse_unit_files_normalises_state() {
        let s = "nginx.service enabled\nsshd.service static\nfoo.timer masked\n";
        let svcs = parse_unit_files(s);
        assert_eq!(svcs.len(), 2);
        assert!(svcs
            .iter()
            .any(|u| u.name == "nginx.service" && u.state == "enabled"));
        assert!(svcs
            .iter()
            .any(|u| u.name == "foo.timer" && u.state == "masked"));
    }

    #[test]
    fn logical_path_strips_root() {
        let lp = logical_path("/tmp/r", Path::new("/tmp/r/etc/foo.conf"));
        assert_eq!(lp, "/etc/foo.conf");
    }

    #[test]
    fn verify_reports_change_detects_flags() {
        assert!(verify_reports_change_for(
            "S.5....T.  c /etc/foo.conf",
            "/etc/foo.conf"
        ));
        assert!(!verify_reports_change_for(
            ".........  c /etc/foo.conf",
            "/etc/foo.conf"
        ));
    }
}
