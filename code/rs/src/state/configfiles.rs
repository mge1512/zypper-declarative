// generated from spec: zypper-declarative.spec.md
// sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
//
// config_files actual state: the changed-from-package and unpackaged /etc files
// and symlinks. The result is the spec's reproducibility emission test; the
// METHOD here (per the Rust decisions hints) is rpm-verdict-parsing rather than
// a self-built baseline join:
//
//   - CHANGED: enumerate config-file owning packages and run `rpm -V` per
//     package, parsing the SM5DLUGTP flag string (type char `c`). A changed
//     regular file is emitted as type "file" with its real sha256; an `L` flag
//     on a package-shipped file is the type-mismatch case, emitted as type
//     "link" with the verbatim on-disk target. Every changed record carries
//     status="changed" and a non-empty `changes` list built from the flags.
//   - GHOST regular files (the case rpm -V skips): a ghost-flagged /etc path
//     with real on-disk content is emitted as type "file" with its sha256.
//   - GHOST symlinks (/etc/alternatives/*): emitted only when the on-disk
//     target differs from the alternatives auto/best target.
//   - UNPACKAGED: an /etc path no package owns is emitted; found by walking
//     /etc and subtracting the rpm-owned path set.
//
// The walk classifies each entry by its own type WITHOUT following symlinks
// (lstat). Regular files are hashed; symlinks store their verbatim target;
// directories are traversed but not emitted; special files are skipped.
// Exclusions: keep-list and /etc/etc.syncpoint.
//
// Content store: when set, each EMITTED regular-file record's bytes are written
// to <content-store>/sha256/<digest> (idempotent dedup) and content_ref is set
// to sha256/<digest>. Read-only otherwise.

use crate::config::OnUnreadable;
use crate::interfaces::CommandRunner;
use crate::manifest::hash::sha256_bytes;
use crate::manifest::ManagedFileRecord;
use std::collections::{BTreeMap, HashSet};
use std::os::unix::fs::PermissionsExt;
use std::path::{Path, PathBuf};

pub const SYNCPOINT: &str = "/etc/etc.syncpoint";

#[derive(Debug, Clone)]
#[allow(dead_code)]
enum DiskKind {
    File,
    Link,
    Dir,
    Special,
}

#[derive(Debug, Clone)]
#[allow(dead_code)]
struct DiskEntry {
    /// Logical path as it appears in the manifest (root-relative, starting "/etc").
    logical: String,
    /// Absolute on-disk path under the described root.
    abs: PathBuf,
    kind: DiskKind,
    mode: String,
    user: String,
    group: String,
    target: String,
    sha256: String,
    content: Option<Vec<u8>>, // raw bytes for regular files (for content store)
}

pub struct ConfigFilesOutput {
    pub records: Vec<ManagedFileRecord>,
    pub diagnostics: Vec<String>,
}

#[derive(Debug)]
pub enum ConfigFilesError {
    Unreadable(String),
}

/// Read config_files actual state under `root`.
pub fn read_config_files(
    runner: &dyn CommandRunner,
    root: &str,
    on_unreadable: &OnUnreadable,
    keep_list: &HashSet<String>,
    content_store: Option<&str>,
) -> Result<ConfigFilesOutput, ConfigFilesError> {
    let etc_root = Path::new(root).join("etc");
    let mut diagnostics = Vec::new();

    // 1. Walk <root>/etc, classifying each entry. Build the on-disk entry map by
    //    logical path.
    let mut entries: BTreeMap<String, DiskEntry> = BTreeMap::new();
    if etc_root.exists() {
        walk_etc(
            root,
            &etc_root,
            on_unreadable,
            &mut entries,
            &mut diagnostics,
        )?;
    }

    // 2. Owned-path set and per-package verdicts from rpm. For a root without an
    //    rpmdb (e.g. a synthetic test root) rpm reports zero packages, so the
    //    owned set is empty and every walked entry is unpackaged. This is the
    //    package database answering, not a skipped lookup.
    let owned = owned_etc_paths(runner, root);
    let changed = changed_verdicts(runner, root, &owned);

    // 3. Decide emission per the reproducibility criterion.
    let mut records: Vec<ManagedFileRecord> = Vec::new();
    let mut emitted_paths: HashSet<String> = HashSet::new();

    // 3a. Changed-from-package records (rpm -V verdict). Only for paths that are
    //     owned and present on disk under /etc.
    for (path, verdict) in &changed {
        if excluded(path, keep_list) {
            continue;
        }
        if let Some(entry) = entries.get(path) {
            let mut rec = record_from_entry(entry, verdict.package.clone());
            rec.status = "changed".to_string();
            rec.changes = verdict.changes.clone();
            // Type-mismatch: rpm 'L' flag means a shipped file is now a link on
            // disk; the walk already captured the on-disk type/target.
            emit(
                &mut records,
                &mut emitted_paths,
                rec,
                entry,
                content_store,
                on_unreadable,
                &mut diagnostics,
            )?;
        }
    }

    // 3b. Unpackaged records: walked /etc entries that no package owns.
    for (path, entry) in &entries {
        if excluded(path, keep_list) {
            continue;
        }
        if owned.contains(path) {
            continue; // owned -> handled by the changed verdict (pristine suppressed)
        }
        if emitted_paths.contains(path) {
            continue;
        }
        match entry.kind {
            DiskKind::File | DiskKind::Link => {
                let rec = record_from_entry(entry, String::new());
                emit(
                    &mut records,
                    &mut emitted_paths,
                    rec,
                    entry,
                    content_store,
                    on_unreadable,
                    &mut diagnostics,
                )?;
            }
            DiskKind::Dir | DiskKind::Special => {}
        }
    }

    // 3c. Ghost regular files with content, and manual alternative symlinks, are
    //     handled by ghost_records (a small targeted pass).
    let ghosts = ghost_records(runner, root, &entries, keep_list, &emitted_paths);
    for entry_path in ghosts {
        if let Some(entry) = entries.get(&entry_path.logical) {
            let mut rec = record_from_entry(entry, entry_path.package.clone());
            rec.status = "changed".to_string();
            rec.changes = entry_path.changes.clone();
            emit(
                &mut records,
                &mut emitted_paths,
                rec,
                entry,
                content_store,
                on_unreadable,
                &mut diagnostics,
            )?;
        }
    }

    records.sort_by(|a, b| a.name.cmp(&b.name));
    Ok(ConfigFilesOutput {
        records,
        diagnostics,
    })
}

fn excluded(path: &str, keep_list: &HashSet<String>) -> bool {
    path == SYNCPOINT || keep_list.contains(path)
}

fn record_from_entry(entry: &DiskEntry, package_name: String) -> ManagedFileRecord {
    match entry.kind {
        DiskKind::Link => ManagedFileRecord {
            name: entry.logical.clone(),
            r#type: "link".to_string(),
            mode: entry.mode.clone(),
            user: entry.user.clone(),
            group: entry.group.clone(),
            sha256: String::new(),
            target: entry.target.clone(),
            content_ref: String::new(),
            package_name,
            status: String::new(),
            changes: Vec::new(),
        },
        _ => ManagedFileRecord {
            name: entry.logical.clone(),
            r#type: "file".to_string(),
            mode: entry.mode.clone(),
            user: entry.user.clone(),
            group: entry.group.clone(),
            sha256: entry.sha256.clone(),
            target: String::new(),
            content_ref: String::new(),
            package_name,
            status: String::new(),
            changes: Vec::new(),
        },
    }
}

#[allow(clippy::too_many_arguments)]
fn emit(
    records: &mut Vec<ManagedFileRecord>,
    emitted: &mut HashSet<String>,
    mut rec: ManagedFileRecord,
    entry: &DiskEntry,
    content_store: Option<&str>,
    on_unreadable: &OnUnreadable,
    diagnostics: &mut Vec<String>,
) -> Result<(), ConfigFilesError> {
    if emitted.contains(&rec.name) {
        return Ok(());
    }
    // Content store population: only for emitted regular files.
    if rec.r#type == "file" {
        if let Some(store) = content_store {
            match &entry.content {
                Some(bytes) => {
                    let digest = if rec.sha256.is_empty() {
                        sha256_bytes(bytes)
                    } else {
                        rec.sha256.clone()
                    };
                    write_blob(store, &digest, bytes).map_err(|e| {
                        ConfigFilesError::Unreadable(format!(
                            "content store write for {}: {}",
                            rec.name, e
                        ))
                    })?;
                    rec.content_ref = format!("sha256/{}", digest);
                }
                None => {
                    // content unreadable -> follow on_unreadable
                    match on_unreadable {
                        OnUnreadable::Error => {
                            return Err(ConfigFilesError::Unreadable(format!(
                                "content of {} unreadable",
                                rec.name
                            )));
                        }
                        OnUnreadable::Warn => {
                            diagnostics.push(format!(
                                "files: content of {} unreadable; emitted with empty content_ref",
                                rec.name
                            ));
                        }
                    }
                }
            }
        }
    }
    emitted.insert(rec.name.clone());
    records.push(rec);
    Ok(())
}

fn write_blob(store: &str, digest: &str, bytes: &[u8]) -> std::io::Result<()> {
    let dir = Path::new(store).join("sha256");
    std::fs::create_dir_all(&dir)?;
    let blob = dir.join(digest);
    if blob.exists() {
        return Ok(()); // idempotent dedup
    }
    std::fs::write(&blob, bytes)
}

// Recursively walk <root>/etc, classifying each entry by its own type (lstat).
fn walk_etc(
    root: &str,
    dir: &Path,
    on_unreadable: &OnUnreadable,
    out: &mut BTreeMap<String, DiskEntry>,
    diagnostics: &mut Vec<String>,
) -> Result<(), ConfigFilesError> {
    let read = match std::fs::read_dir(dir) {
        Ok(r) => r,
        Err(e) => {
            return unreadable(
                on_unreadable,
                diagnostics,
                format!("{}: {}", dir.display(), e),
            );
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
        if ft.is_dir() {
            // traverse; do not emit a record for the directory itself
            walk_etc(root, &abs, on_unreadable, out, diagnostics)?;
        } else if ft.is_symlink() {
            let target = match std::fs::read_link(&abs) {
                Ok(t) => t.to_string_lossy().into_owned(),
                Err(e) => {
                    unreadable(
                        on_unreadable,
                        diagnostics,
                        format!("{}: {}", abs.display(), e),
                    )?;
                    continue;
                }
            };
            out.insert(
                logical.clone(),
                DiskEntry {
                    logical,
                    abs,
                    kind: DiskKind::Link,
                    mode: mode_string(&meta),
                    user: "root".to_string(),
                    group: "root".to_string(),
                    target,
                    sha256: String::new(),
                    content: None,
                },
            );
        } else if ft.is_file() {
            let bytes = std::fs::read(&abs).ok();
            let sha = bytes.as_ref().map(|b| sha256_bytes(b)).unwrap_or_default();
            out.insert(
                logical.clone(),
                DiskEntry {
                    logical,
                    abs,
                    kind: DiskKind::File,
                    mode: mode_string(&meta),
                    user: "root".to_string(),
                    group: "root".to_string(),
                    target: String::new(),
                    sha256: sha,
                    content: bytes,
                },
            );
        } else {
            // special file (device, fifo, socket): skip, never read/emit/error
            out.insert(
                logical.clone(),
                DiskEntry {
                    logical,
                    abs,
                    kind: DiskKind::Special,
                    mode: String::new(),
                    user: String::new(),
                    group: String::new(),
                    target: String::new(),
                    sha256: String::new(),
                    content: None,
                },
            );
        }
    }
    Ok(())
}

fn unreadable(
    on_unreadable: &OnUnreadable,
    diagnostics: &mut Vec<String>,
    source: String,
) -> Result<(), ConfigFilesError> {
    match on_unreadable {
        OnUnreadable::Error => Err(ConfigFilesError::Unreadable(source)),
        OnUnreadable::Warn => {
            diagnostics.push(format!("files: unreadable source {}", source));
            Ok(())
        }
    }
}

fn logical_path(root: &str, abs: &Path) -> String {
    let root_path = Path::new(root);
    match abs.strip_prefix(root_path) {
        Ok(rel) => {
            let mut s = String::from("/");
            s.push_str(&rel.to_string_lossy());
            s
        }
        Err(_) => abs.to_string_lossy().into_owned(),
    }
}

fn mode_string(meta: &std::fs::Metadata) -> String {
    format!("{:04o}", meta.permissions().mode() & 0o7777)
}

// Owned /etc path set: enumerate config-file owning packages and their file
// lists under `root`. Empty for a root without an rpmdb.
fn owned_etc_paths(runner: &dyn CommandRunner, root: &str) -> HashSet<String> {
    let mut owned = HashSet::new();
    let pkgs = config_owning_packages(runner, root);
    for pkg in &pkgs {
        for path in package_file_list(runner, root, pkg) {
            if path.starts_with("/etc/") || path == "/etc" {
                owned.insert(path);
            }
        }
    }
    owned
}

// `rpm -qca --queryformat '%{NAME}\n'`: packages owning config files.
fn config_owning_packages(runner: &dyn CommandRunner, root: &str) -> Vec<String> {
    let mut args = root_args(root);
    args.push("-qca".to_string());
    args.push("--queryformat".to_string());
    args.push("%{NAME}\\n".to_string());
    let argref: Vec<&str> = args.iter().map(|s| s.as_str()).collect();
    let res = runner.run("rpm", &argref);
    if res.spawn_failed {
        return Vec::new();
    }
    let mut set: Vec<String> = Vec::new();
    let mut seen = HashSet::new();
    for line in res.stdout.lines() {
        let l = line.trim();
        if l.is_empty()
            || l.starts_with('(')
            || l.starts_with("error:")
            || l.starts_with("warning:")
        {
            continue;
        }
        if seen.insert(l.to_string()) {
            set.push(l.to_string());
        }
    }
    set
}

fn package_file_list(runner: &dyn CommandRunner, root: &str, pkg: &str) -> Vec<String> {
    let mut args = root_args(root);
    args.push("-ql".to_string());
    args.push(pkg.to_string());
    let argref: Vec<&str> = args.iter().map(|s| s.as_str()).collect();
    let res = runner.run("rpm", &argref);
    if res.spawn_failed {
        return Vec::new();
    }
    res.stdout
        .lines()
        .map(|l| l.trim().to_string())
        .filter(|l| l.starts_with('/'))
        .collect()
}

fn root_args(root: &str) -> Vec<String> {
    let mut args = Vec::new();
    if root != "/" {
        args.push("--root".to_string());
        args.push(root.to_string());
    }
    args
}

#[derive(Debug, Clone)]
struct Verdict {
    package: String,
    changes: Vec<String>,
}

// Run `rpm -V` per owning package and parse the verdict for /etc config files.
fn changed_verdicts(
    runner: &dyn CommandRunner,
    root: &str,
    owned: &HashSet<String>,
) -> BTreeMap<String, Verdict> {
    let mut out: BTreeMap<String, Verdict> = BTreeMap::new();
    if owned.is_empty() {
        return out;
    }
    let pkgs = config_owning_packages(runner, root);
    for pkg in &pkgs {
        let mut args = root_args(root);
        args.push("-V".to_string());
        args.push("--nodeps".to_string());
        args.push("--noscripts".to_string());
        args.push(pkg.to_string());
        let argref: Vec<&str> = args.iter().map(|s| s.as_str()).collect();
        let res = runner.run("rpm", &argref);
        // Non-zero exit is NORMAL (rpm -V reports differences). Parse regardless.
        // A genuine package error: stdout empty AND stderr non-empty.
        if res.stdout.trim().is_empty() && !res.stderr.trim().is_empty() {
            continue;
        }
        for (path, changes) in parse_verify_output(&res.stdout) {
            if path.starts_with("/etc/") {
                out.insert(
                    path,
                    Verdict {
                        package: pkg.clone(),
                        changes,
                    },
                );
            }
        }
    }
    out
}

/// Parse `rpm -V` output. Each line: `<9 flag chars><space><type><space><path>`.
/// Keep only type char `c` (config). Returns (path, changes) where changes is
/// the human-readable attribute set derived from the flags.
pub fn parse_verify_output(stdout: &str) -> Vec<(String, Vec<String>)> {
    let mut out = Vec::new();
    for line in stdout.lines() {
        if line.trim().is_empty() {
            continue;
        }
        // "missing  c /etc/foo" form (deleted)
        let (flags, typ, path) = match split_verify_line(line) {
            Some(t) => t,
            None => continue,
        };
        if typ != "c" {
            continue;
        }
        let mut changes = changes_from_flags(&flags);
        if flags.contains("missing") {
            changes.push("deleted".to_string());
        }
        // A verdict line whose flag string carries no concrete change marker
        // (only `.` for unchanged and `?` for a test that could not be
        // performed) reports no detected difference: rpm could not confirm a
        // change, so emitting it as `status=changed` with an empty `changes`
        // list would violate the invariant that every changed record carries a
        // non-empty `changes`. Under root (the self-check environment) such
        // untestable lines do not occur for readable files; here they are
        // dropped rather than mislabelled as changed.
        if changes.is_empty() {
            continue;
        }
        out.push((path, changes));
    }
    out
}

fn split_verify_line(line: &str) -> Option<(String, String, String)> {
    let trimmed = line.trim_end();
    // The line may start with "missing" or with 9 attribute chars.
    if let Some(rest) = trimmed.strip_prefix("missing") {
        let rest = rest.trim_start();
        let mut it = rest.splitn(2, char::is_whitespace);
        let typ = it.next()?.to_string();
        let path = it.next()?.trim().to_string();
        return Some(("missing".to_string(), typ, path));
    }
    // Standard: first 9 chars are the attribute flags, then space, type, space, path.
    if trimmed.len() < 11 {
        return None;
    }
    let flags = &trimmed[..9];
    let rest = trimmed[9..].trim_start();
    let mut it = rest.splitn(2, char::is_whitespace);
    let typ = it.next()?.to_string();
    let path = it.next()?.trim().to_string();
    Some((flags.to_string(), typ, path))
}

fn changes_from_flags(flags: &str) -> Vec<String> {
    let mut changes = Vec::new();
    for ch in flags.chars() {
        let label = match ch {
            'S' => "size",
            'M' => "mode",
            '5' => "md5",
            'D' => "device",
            'L' => "link_path",
            'U' => "user",
            'G' => "group",
            'T' => "time",
            'P' => "caps",
            _ => continue,
        };
        changes.push(label.to_string());
    }
    changes
}

#[derive(Debug, Clone)]
struct GhostRecord {
    logical: String,
    package: String,
    changes: Vec<String>,
}

// Ghost handling: ghost regular files with real on-disk content, and manual
// alternative symlinks. A small targeted pass over the few ghost paths, never a
// whole-/etc walk.
fn ghost_records(
    runner: &dyn CommandRunner,
    root: &str,
    entries: &BTreeMap<String, DiskEntry>,
    keep_list: &HashSet<String>,
    emitted: &HashSet<String>,
) -> Vec<GhostRecord> {
    let mut out = Vec::new();
    let ghosts = ghost_paths(runner, root);
    for path in ghosts {
        if excluded(&path, keep_list) || emitted.contains(&path) {
            continue;
        }
        let entry = match entries.get(&path) {
            Some(e) => e,
            None => continue,
        };
        match entry.kind {
            DiskKind::File => {
                // ghost regular file: emit only if it has real (non-empty) content
                let nonempty = entry
                    .content
                    .as_ref()
                    .map(|b| !b.is_empty())
                    .unwrap_or(false);
                if nonempty {
                    out.push(GhostRecord {
                        package: owning_package(runner, root, &path),
                        logical: path,
                        changes: vec!["md5".to_string()],
                    });
                }
            }
            DiskKind::Link => {
                // ghost symlink: emit only if on-disk target differs from the
                // alternatives auto/best target.
                if let Some(name) = path.strip_prefix("/etc/alternatives/") {
                    if let Some(best) = alternatives_best(runner, root, name) {
                        if entry.target != best {
                            out.push(GhostRecord {
                                package: owning_package(runner, root, &path),
                                logical: path,
                                changes: vec!["link_path".to_string()],
                            });
                        }
                    }
                    // if best is unknown, on_unreadable would apply; conservative
                    // here: do not blanket-emit/suppress (skip).
                }
            }
            _ => {}
        }
    }
    out
}

// Enumerate ghost-flagged /etc paths.
//
// `%{NAME}` is a per-package SCALAR; placing it inside the array iterator
// `[ ... ]` makes rpm emit it only for the first array element, so a mixed
// `[%{NAME} %{FILENAMES} %{FILEFLAGS}\n]` template yields exactly one line per
// package (the first file) and silently drops every other ghost path. The
// correct enumeration iterates files ONLY (`[%{FILENAMES} %{FILEFLAGS}\n]`),
// which lists every file of every package with its per-file flags; the owning
// package is resolved separately by `owning_package` for just the paths that
// are actually emitted. The `g` flag (bit 64, GHOST) is what `rpm -V` never
// reports, so this pass is the only source of content-bearing ghosts such as
// `/etc/pam.d/common-auth-pc` and `/etc/machine-id`.
fn ghost_paths(runner: &dyn CommandRunner, root: &str) -> Vec<String> {
    let mut args = root_args(root);
    args.push("-qa".to_string());
    args.push("--queryformat".to_string());
    args.push("[%{FILENAMES} %{FILEFLAGS:fflags}\\n]".to_string());
    let argref: Vec<&str> = args.iter().map(|s| s.as_str()).collect();
    let res = runner.run("rpm", &argref);
    if res.spawn_failed {
        return Vec::new();
    }
    let mut out = Vec::new();
    let mut seen = HashSet::new();
    for line in res.stdout.lines() {
        let parts: Vec<&str> = line.split_whitespace().collect();
        if parts.len() < 2 {
            continue;
        }
        let path = parts[0];
        let fflags = parts[1];
        if path.starts_with("/etc/") && fflags.contains('g') && seen.insert(path.to_string()) {
            out.push(path.to_string());
        }
    }
    out
}

// Resolve the bare owning package name for a single path via `rpm -qf`.
fn owning_package(runner: &dyn CommandRunner, root: &str, path: &str) -> String {
    let mut args = root_args(root);
    args.push("-qf".to_string());
    args.push("--queryformat".to_string());
    args.push("%{NAME}\\n".to_string());
    args.push(path.to_string());
    let argref: Vec<&str> = args.iter().map(|s| s.as_str()).collect();
    let res = runner.run("rpm", &argref);
    if res.spawn_failed {
        return String::new();
    }
    for line in res.stdout.lines() {
        let l = line.trim();
        if l.is_empty()
            || l.starts_with('(')
            || l.starts_with("error:")
            || l.starts_with("warning:")
        {
            continue;
        }
        return l.to_string();
    }
    String::new()
}

// Query the alternatives auto/best target for `name`.
fn alternatives_best(runner: &dyn CommandRunner, _root: &str, name: &str) -> Option<String> {
    let res = runner.run("update-alternatives", &["--query", name]);
    if res.spawn_failed || res.stdout.trim().is_empty() {
        return None;
    }
    // `Best: /usr/bin/foo` line, else `Value: ...` for the current.
    for line in res.stdout.lines() {
        if let Some(v) = line.strip_prefix("Best: ") {
            return Some(v.trim().to_string());
        }
    }
    None
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn parses_changed_config_file() {
        let out = "S.5....T.  c /etc/foo.conf\n";
        let v = parse_verify_output(out);
        assert_eq!(v.len(), 1);
        assert_eq!(v[0].0, "/etc/foo.conf");
        assert!(v[0].1.contains(&"size".to_string()));
        assert!(v[0].1.contains(&"md5".to_string()));
        assert!(v[0].1.contains(&"time".to_string()));
    }

    #[test]
    fn type_mismatch_link_flag_parsed() {
        let out = "....L....  c /etc/pam.d/common-auth\n";
        let v = parse_verify_output(out);
        assert_eq!(v[0].0, "/etc/pam.d/common-auth");
        assert!(v[0].1.contains(&"link_path".to_string()));
    }

    #[test]
    fn non_config_lines_dropped() {
        let out = "S.5....T.  d /usr/share/doc/x\n.M.......    /etc/binfile\n";
        let v = parse_verify_output(out);
        assert!(v.is_empty(), "only type 'c' lines are kept");
    }

    #[test]
    fn missing_means_deleted() {
        let out = "missing     c /etc/gone.conf\n";
        let v = parse_verify_output(out);
        assert_eq!(v[0].0, "/etc/gone.conf");
        assert!(v[0].1.contains(&"deleted".to_string()));
    }
}
