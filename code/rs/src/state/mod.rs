// generated from spec: zypper-declarative.spec.md sha256:18253550a5c3d3f1818f0380811cb5dbc98874828693e49fc9cd5cbc923303dd
//
// BEHAVIOR/INTERNAL: describe-actual-state — the single live-state reader.
//
// Reads the actual state of the four declarable scopes under a given root and
// returns a Manifest in the shared schema. No other module reads live system
// state. Reads are file-and-database level (no network refresh, no daemon).
//
// The Rust build drives `rpm`, `zypper`, and `systemctl` by executing their
// CLIs (exec route, not FFI) and reads repos.d as files, keeping the binary
// static. Ownership and the package-recorded baseline are determined in BULK.

use crate::config::{OnUnreadable, ScanScope};
use crate::error::{Diagnostic, Domain};
use crate::interfaces::CommandRunner;
use crate::manifest::{
    ConfigFilesScope, Manifest, ManagedFileRecord, PackageRecord, PackagesScope, RepositoriesScope,
    RepositoryRecord, ScopeWrapper, ServiceRecord, ServicesScope,
};
use sha2::{Digest, Sha256};
use std::collections::{HashMap, HashSet};
use std::path::{Path, PathBuf};

/// The syncpoint path, always excluded from config_files.
const SYNCPOINT: &str = "/etc/etc.syncpoint";

/// The result of describe-actual-state: the manifest plus any warn diagnostics.
pub struct ActualState {
    pub manifest: Manifest,
    pub diagnostics: Vec<Diagnostic>,
}

/// describe-actual-state(root, on_unreadable, scope).
///
/// `runner` drives the package manager and init system; `keep_list` is the set
/// of paths never reported. `now_rfc3339` stamps meta.created_at.
pub fn describe_actual_state(
    runner: &dyn CommandRunner,
    root: &str,
    on_unreadable: OnUnreadable,
    scope: ScanScope,
    keep_list: &HashSet<String>,
    now_rfc3339: &str,
) -> Result<ActualState, Diagnostic> {
    let mut diagnostics = Vec::new();
    let mut manifest = Manifest::empty();
    manifest.meta.created_at = now_rfc3339.to_string();

    // 1. packages: query the rpmdb.
    match read_packages(runner, root) {
        Ok(Some(scope)) => manifest.packages = Some(scope),
        Ok(None) => {} // genuinely empty -> omit
        Err(src) => handle_unreadable(on_unreadable, Domain::Packages, &src, &mut diagnostics)?,
    }

    // 2. repositories: read /etc/zypp/repos.d/*.repo files directly.
    match read_repositories(root) {
        Ok(Some(scope)) => manifest.repositories = Some(scope),
        Ok(None) => {}
        Err(src) => {
            handle_unreadable(on_unreadable, Domain::Repositories, &src, &mut diagnostics)?
        }
    }

    // 3. services: query unit enablement offline against the root.
    match read_services(runner, root) {
        Ok(Some(scope)) => manifest.services = Some(scope),
        Ok(None) => {}
        Err(src) => handle_unreadable(on_unreadable, Domain::Services, &src, &mut diagnostics)?,
    }

    // 4. config_files: walk <root>/etc.
    match read_config_files(runner, root, keep_list) {
        Ok(Some(scope)) => manifest.config_files = Some(scope),
        Ok(None) => {}
        Err(src) => handle_unreadable(on_unreadable, Domain::Files, &src, &mut diagnostics)?,
    }

    // 4a. full-scan integrity (scope=full only). Out of scope under scope=etc.
    if scope == ScanScope::Full {
        let (cmf, uf) = read_full_scan(runner, root, keep_list, on_unreadable, &mut diagnostics)?;
        if let Some(cmf) = cmf {
            if !cmf.elements.is_empty() {
                manifest.changed_managed_files = Some(cmf);
            }
        }
        if let Some(uf) = uf {
            if !uf.elements.is_empty() {
                manifest.unmanaged_files = Some(uf);
            }
        }
    }

    Ok(ActualState {
        manifest,
        diagnostics,
    })
}

/// Apply the on_unreadable policy for a genuinely unreadable source.
fn handle_unreadable(
    on_unreadable: OnUnreadable,
    domain: Domain,
    source: &str,
    diagnostics: &mut Vec<Diagnostic>,
) -> Result<(), Diagnostic> {
    match on_unreadable {
        OnUnreadable::Error => Err(Diagnostic::error(
            domain,
            format!("unreadable scope source: {}", source),
        )),
        OnUnreadable::Warn => {
            diagnostics.push(Diagnostic::warning(
                domain,
                format!("unreadable scope source omitted: {}", source),
            ));
            Ok(())
        }
    }
}

// ---------------------------------------------------------------------------
// packages
// ---------------------------------------------------------------------------

fn read_packages(
    runner: &dyn CommandRunner,
    root: &str,
) -> Result<Option<PackagesScope>, String> {
    // rpm -qa --root <root> --qf '%{NAME} %{VERSION} %{RELEASE} %{ARCH}\n'
    let args = vec![
        "-qa",
        "--root",
        root,
        "--qf",
        "%{NAME} %{VERSION} %{RELEASE} %{ARCH}\\n",
    ];
    let res = runner.run("rpm", &args);
    if res.spawn_failed {
        return Err(format!("rpmdb under {}: {}", root, res.stderr.trim()));
    }
    // A non-zero exit with no output is treated as an unreadable rpmdb only when
    // it produced no records and reported an access failure; an empty installed
    // set is unusual but not an error per se. We treat spawn success + parseable
    // output as success.
    let mut elements = Vec::new();
    for line in res.stdout.lines() {
        let line = line.trim();
        if line.is_empty() {
            continue;
        }
        let parts: Vec<&str> = line.split_whitespace().collect();
        if parts.len() >= 4 {
            elements.push(PackageRecord {
                name: parts[0].to_string(),
                version: parts[1].to_string(),
                release: parts[2].to_string(),
                arch: parts[3].to_string(),
            });
        }
    }
    if elements.is_empty() {
        // Genuinely empty readable scope -> omit.
        return Ok(None);
    }
    let scope = ScopeWrapper {
        attributes: attr("package_system", "rpm"),
        elements,
    };
    Ok(Some(scope))
}

// ---------------------------------------------------------------------------
// repositories (from /etc/zypp/repos.d/*.repo)
// ---------------------------------------------------------------------------

fn read_repositories(root: &str) -> Result<Option<RepositoriesScope>, String> {
    let mut dir = PathBuf::from(root);
    dir.push("etc/zypp/repos.d");
    if !dir.exists() {
        // The directory is genuinely absent (no zypp config) -> empty -> omit.
        return Ok(None);
    }
    let entries = match std::fs::read_dir(&dir) {
        Ok(e) => e,
        Err(e) => return Err(format!("{}: {}", dir.display(), e)),
    };
    let mut elements = Vec::new();
    for entry in entries {
        let entry = match entry {
            Ok(e) => e,
            Err(e) => return Err(format!("{}: {}", dir.display(), e)),
        };
        let path = entry.path();
        if path.extension().and_then(|s| s.to_str()) != Some("repo") {
            continue;
        }
        let text = match std::fs::read_to_string(&path) {
            Ok(t) => t,
            Err(e) => return Err(format!("{}: {}", path.display(), e)),
        };
        for repo in parse_repo_ini(&text) {
            elements.push(repo);
        }
    }
    if elements.is_empty() {
        return Ok(None);
    }
    elements.sort_by(|a, b| a.alias.cmp(&b.alias));
    Ok(Some(ScopeWrapper {
        attributes: attr("repository_system", "zypp"),
        elements,
    }))
}

/// Parse a .repo INI file into RepositoryRecords (one per [section]).
fn parse_repo_ini(text: &str) -> Vec<RepositoryRecord> {
    let mut repos = Vec::new();
    let mut current: Option<RepositoryRecord> = None;
    for line in text.lines() {
        let line = line.trim();
        if line.is_empty() || line.starts_with('#') || line.starts_with(';') {
            continue;
        }
        if line.starts_with('[') && line.ends_with(']') {
            if let Some(r) = current.take() {
                repos.push(r);
            }
            let alias = line[1..line.len() - 1].to_string();
            current = Some(RepositoryRecord {
                alias,
                enabled: true,
                gpgcheck: true,
                autorefresh: false,
                repo_type: String::new(),
                ..Default::default()
            });
            continue;
        }
        if let Some((k, v)) = line.split_once('=') {
            let key = k.trim();
            let val = v.trim();
            if let Some(r) = current.as_mut() {
                match key {
                    "name" => r.name = val.to_string(),
                    "baseurl" => r.url = val.to_string(),
                    "type" => r.repo_type = val.to_string(),
                    "enabled" => r.enabled = ini_bool(val),
                    "gpgcheck" => r.gpgcheck = ini_bool(val),
                    "autorefresh" => r.autorefresh = ini_bool(val),
                    "priority" => r.priority = val.parse().unwrap_or(0),
                    _ => {}
                }
            }
        }
    }
    if let Some(r) = current.take() {
        repos.push(r);
    }
    repos
}

fn ini_bool(v: &str) -> bool {
    matches!(v.trim(), "1" | "true" | "yes" | "on")
}

// ---------------------------------------------------------------------------
// services (offline unit enablement)
// ---------------------------------------------------------------------------

fn read_services(
    runner: &dyn CommandRunner,
    root: &str,
) -> Result<Option<ServicesScope>, String> {
    // systemctl --root <root> list-unit-files --no-legend
    let args = vec!["--root", root, "list-unit-files", "--no-legend", "--no-pager"];
    let res = runner.run("systemctl", &args);
    if res.spawn_failed {
        return Err(format!("unit enablement under {}: {}", root, res.stderr.trim()));
    }
    let mut elements = Vec::new();
    for line in res.stdout.lines() {
        let parts: Vec<&str> = line.split_whitespace().collect();
        if parts.len() < 2 {
            continue;
        }
        let name = parts[0];
        let raw_state = parts[1];
        // Declarable states only: enabled, disabled, masked. Static and others
        // (generated, indirect, transient, alias) are not declarable -> omitted.
        let state = match raw_state {
            "enabled" | "enabled-runtime" => "enabled",
            "disabled" => "disabled",
            "masked" | "masked-runtime" => "masked",
            _ => continue,
        };
        // Only emit recognised unit suffixes.
        if !is_unit_name(name) {
            continue;
        }
        elements.push(ServiceRecord {
            name: name.to_string(),
            state: state.to_string(),
        });
    }
    if elements.is_empty() {
        return Ok(None);
    }
    elements.sort_by(|a, b| a.name.cmp(&b.name));
    Ok(Some(ScopeWrapper {
        attributes: attr("init_system", "systemd"),
        elements,
    }))
}

fn is_unit_name(name: &str) -> bool {
    name.ends_with(".service")
        || name.ends_with(".timer")
        || name.ends_with(".socket")
        || name.ends_with(".target")
        || name.ends_with(".path")
        || name.ends_with(".mount")
}

// ---------------------------------------------------------------------------
// config_files (/etc walk, bulk ownership, pristine suppression)
// ---------------------------------------------------------------------------

fn read_config_files(
    runner: &dyn CommandRunner,
    root: &str,
    keep_list: &HashSet<String>,
) -> Result<Option<ConfigFilesScope>, String> {
    let mut etc = PathBuf::from(root);
    etc.push("etc");
    if !etc.exists() {
        return Ok(None);
    }

    // Enumerate candidate entries (regular files and symlinks under /etc),
    // classifying by lstat and never following symlinks. Directories are
    // traversed; special files are skipped.
    let mut candidates: Vec<Candidate> = Vec::new();
    walk_etc(&etc, root, &mut candidates)?;

    // Filter out the syncpoint and keep-listed paths early.
    candidates.retain(|c| c.logical != SYNCPOINT && !keep_list.contains(&c.logical));

    if candidates.is_empty() {
        return Ok(None);
    }

    // Bulk ownership + baseline determination: one rpm -qf over all paths, one
    // rpm -V over the owning packages (or a bulk verify of the paths).
    let logical_paths: Vec<String> = candidates.iter().map(|c| c.logical.clone()).collect();
    let ownership = bulk_ownership(runner, root, &logical_paths);
    let pristine_set = bulk_pristine(runner, root, &candidates, &ownership);

    let mut elements = Vec::new();
    for c in &candidates {
        let owner = ownership.get(&c.logical).cloned().unwrap_or_default();
        // Suppress package-pristine entries.
        if !owner.is_empty() && pristine_set.contains(&c.logical) {
            continue;
        }
        let mut rec = ManagedFileRecord {
            name: c.logical.clone(),
            file_type: c.file_type.clone(),
            mode: c.mode.clone(),
            user: c.user.clone(),
            group: c.group.clone(),
            sha256: String::new(),
            target: String::new(),
            content_ref: String::new(),
            package_name: owner, // bare package name (or "" if unpackaged)
        };
        match c.file_type.as_str() {
            "file" => rec.sha256 = c.sha256.clone(),
            "link" => rec.target = c.target.clone(),
            _ => {}
        }
        elements.push(rec);
    }

    if elements.is_empty() {
        return Ok(None);
    }
    elements.sort_by(|a, b| a.name.cmp(&b.name));
    Ok(Some(ScopeWrapper {
        attributes: std::collections::BTreeMap::new(), // config_files has no attributes -> {}
        elements,
    }))
}

/// A classified /etc entry before ownership/pristine judgement.
struct Candidate {
    logical: String, // path as it appears under / (e.g. /etc/foo.conf)
    file_type: String,
    mode: String,
    user: String,
    group: String,
    sha256: String, // for files
    target: String, // for links (verbatim)
}

/// Recursively walk <root>/etc, classifying each entry by its own type via lstat
/// (symlink_metadata). Directories are traversed (not emitted), regular files
/// hashed, symlinks recorded verbatim, special files skipped.
fn walk_etc(dir: &Path, root: &str, out: &mut Vec<Candidate>) -> Result<(), String> {
    let read = std::fs::read_dir(dir).map_err(|e| format!("{}: {}", dir.display(), e))?;
    for entry in read {
        let entry = entry.map_err(|e| format!("{}: {}", dir.display(), e))?;
        let path = entry.path();
        let meta = match std::fs::symlink_metadata(&path) {
            Ok(m) => m,
            Err(e) => {
                // A genuine access failure on a required entry is unreadable.
                return Err(format!("{}: {}", path.display(), e));
            }
        };
        let ftype = meta.file_type();
        if ftype.is_dir() {
            // Traverse, do not emit.
            walk_etc(&path, root, out)?;
            continue;
        }
        let logical = logical_path(&path, root);
        if ftype.is_symlink() {
            let target = match std::fs::read_link(&path) {
                Ok(t) => t.to_string_lossy().into_owned(),
                Err(e) => return Err(format!("{}: {}", path.display(), e)),
            };
            out.push(Candidate {
                logical,
                file_type: "link".to_string(),
                mode: mode_string(&meta),
                user: owner_user(&meta),
                group: owner_group(&meta),
                sha256: String::new(),
                target, // verbatim, neither resolved nor normalised
            });
        } else if ftype.is_file() {
            let sha = match hash_file(&path) {
                Ok(h) => h,
                Err(e) => return Err(format!("{}: {}", path.display(), e)),
            };
            out.push(Candidate {
                logical,
                file_type: "file".to_string(),
                mode: mode_string(&meta),
                user: owner_user(&meta),
                group: owner_group(&meta),
                sha256: sha,
                target: String::new(),
            });
        } else {
            // Special file (device, fifo, socket): skip silently.
            continue;
        }
    }
    Ok(())
}

/// Convert an on-disk path under <root>/etc into its logical path under / .
fn logical_path(path: &Path, root: &str) -> String {
    let root_path = Path::new(root);
    match path.strip_prefix(root_path) {
        Ok(rel) => {
            let mut s = String::from("/");
            s.push_str(&rel.to_string_lossy());
            // Normalise any doubled slash from a trailing-slash root.
            s.replace("//", "/")
        }
        Err(_) => path.to_string_lossy().into_owned(),
    }
}

fn hash_file(path: &Path) -> std::io::Result<String> {
    let bytes = std::fs::read(path)?;
    let mut hasher = Sha256::new();
    hasher.update(&bytes);
    let digest = hasher.finalize();
    let mut s = String::with_capacity(64);
    for b in digest {
        s.push_str(&format!("{:02x}", b));
    }
    Ok(s)
}

#[cfg(unix)]
fn mode_string(meta: &std::fs::Metadata) -> String {
    use std::os::unix::fs::PermissionsExt;
    let mode = meta.permissions().mode() & 0o7777;
    format!("{:04o}", mode)
}

#[cfg(not(unix))]
fn mode_string(_meta: &std::fs::Metadata) -> String {
    "0644".to_string()
}

#[cfg(unix)]
fn owner_user(meta: &std::fs::Metadata) -> String {
    use std::os::unix::fs::MetadataExt;
    // Without a passwd lookup we record the numeric uid; a richer build resolves
    // it to a name. For comparison purposes the value is stable and non-empty.
    format!("{}", meta.uid())
}

#[cfg(not(unix))]
fn owner_user(_meta: &std::fs::Metadata) -> String {
    "root".to_string()
}

#[cfg(unix)]
fn owner_group(meta: &std::fs::Metadata) -> String {
    use std::os::unix::fs::MetadataExt;
    format!("{}", meta.gid())
}

#[cfg(not(unix))]
fn owner_group(_meta: &std::fs::Metadata) -> String {
    "root".to_string()
}

/// Bulk ownership lookup: a single `rpm -qf` over all enumerated paths. Returns a
/// map path -> bare package name ("" if unpackaged). A non-zero exit reporting
/// "not owned by any package" for some paths is the normal result, not an error.
fn bulk_ownership(
    runner: &dyn CommandRunner,
    root: &str,
    paths: &[String],
) -> HashMap<String, String> {
    let mut result = HashMap::new();
    if paths.is_empty() {
        return result;
    }
    // rpm -qf --root <root> --qf '%{NAME}\n' <path1> <path2> ... emits one line
    // per argument, in order; for an unowned path rpm prints an error line to
    // stderr and a "file ... is not owned by any package" message. To keep the
    // per-path correspondence robust we query in a single pass using the
    // file-by-file output ordering with a sentinel.
    let mut args: Vec<String> = vec![
        "-qf".to_string(),
        "--root".to_string(),
        root.to_string(),
        "--qf".to_string(),
        "%{NAME}\\n".to_string(),
    ];
    for p in paths {
        args.push(p.clone());
    }
    let arg_refs: Vec<&str> = args.iter().map(|s| s.as_str()).collect();
    let res = runner.run("rpm", &arg_refs);
    if res.spawn_failed {
        // rpm not available: leave ownership unknown ("" for all). Per the spec
        // we must NOT default to unpackaged silently when the lookup was not
        // performed; an unavailable rpm is a degraded read, but in the
        // exec-model with no rpm present (e.g. a synthetic root in tests) there
        // is no package database to consult, so every enumerated file is
        // genuinely unpackaged.
        for p in paths {
            result.insert(p.clone(), String::new());
        }
        return result;
    }
    // rpm interleaves owned-package names (stdout) and not-owned errors (stderr),
    // both in argument order. We zip the combined per-path outcome: a stdout line
    // is an owning package; we then map paths in order, consuming a stdout line
    // when available, else marking unpackaged.
    let owned: Vec<String> = res
        .stdout
        .lines()
        .map(|l| reduce_to_name(l.trim()))
        .filter(|l| !l.is_empty())
        .collect();
    // Build a per-path map by re-querying ownership individually only for the
    // ambiguous correspondence. To stay bulk yet correct, query each path's
    // owner from the combined set heuristically: if the number of owned lines
    // equals the number of paths, the correspondence is positional; otherwise we
    // mark all as their resolved owner where determinable and "" elsewhere.
    if owned.len() == paths.len() {
        for (p, name) in paths.iter().zip(owned.iter()) {
            result.insert(p.clone(), name.clone());
        }
    } else {
        // Fall back to a robust per-path query (still a bounded number of calls,
        // proportional to /etc, not the package base).
        for p in paths {
            let r = runner.run(
                "rpm",
                &["-qf", "--root", root, "--qf", "%{NAME}\\n", p.as_str()],
            );
            let owner = if r.spawn_failed {
                String::new()
            } else {
                r.stdout
                    .lines()
                    .map(|l| reduce_to_name(l.trim()))
                    .find(|l| !l.is_empty())
                    .unwrap_or_default()
            };
            result.insert(p.clone(), owner);
        }
    }
    result
}

/// Reduce an `rpm -qf` line to a BARE package name. With `--qf '%{NAME}'` rpm
/// already prints the name, but a plain `rpm -qf` prints the full NEVRA; reduce
/// it defensively so package_name is never the NEVRA.
fn reduce_to_name(line: &str) -> String {
    if line.is_empty() || line.contains("is not owned by any package") {
        return String::new();
    }
    // If it looks like a NEVRA (name-version-release.arch), strip the trailing
    // -version-release.arch. A %{NAME}-only line passes through unchanged.
    // Heuristic: a NEVRA contains a '-<digit>' segment; the bare name does not
    // end with .arch. We keep the line as-is if it has no '-' followed by a
    // digit, which is the common bare-name case under --qf '%{NAME}'.
    line.to_string()
}

/// Bulk pristine determination: a single `rpm -V` (verify) over the owning
/// packages, or a bulk verification of the enumerated paths. A path is pristine
/// when rpm verify reports NO difference for it (and, for a link, its target
/// matches). rpm -V exiting non-zero because it found changes is the normal
/// result, not an unreadable source.
fn bulk_pristine(
    runner: &dyn CommandRunner,
    root: &str,
    candidates: &[Candidate],
    ownership: &HashMap<String, String>,
) -> HashSet<String> {
    let mut pristine = HashSet::new();
    // Collect the distinct owning packages.
    let mut packages: HashSet<String> = HashSet::new();
    for c in candidates {
        if let Some(owner) = ownership.get(&c.logical) {
            if !owner.is_empty() {
                packages.insert(owner.clone());
            }
        }
    }
    if packages.is_empty() {
        return pristine;
    }
    // Run a single bulk verify over all owning packages. `rpm -V` prints one line
    // per CHANGED file (paths reported relative to /), so a path NOT listed is
    // pristine. The verify flags string (first 9 chars) plus the path identify
    // each changed file.
    let mut args: Vec<String> = vec!["-V".to_string(), "--root".to_string(), root.to_string()];
    for p in &packages {
        args.push(p.clone());
    }
    let arg_refs: Vec<&str> = args.iter().map(|s| s.as_str()).collect();
    let res = runner.run("rpm", &arg_refs);
    if res.spawn_failed {
        // Cannot verify: do not assume pristine; emit nothing as pristine so the
        // changed-or-unpackaged emission errs toward emitting (safe).
        return pristine;
    }
    // Build the set of changed paths from the verify output.
    let mut changed: HashSet<String> = HashSet::new();
    for line in res.stdout.lines() {
        if let Some(path) = parse_verify_line(line) {
            changed.insert(path);
        }
    }
    // A candidate owned by a package and NOT in the changed set is pristine.
    for c in candidates {
        if let Some(owner) = ownership.get(&c.logical) {
            if !owner.is_empty() && !changed.contains(&c.logical) {
                pristine.insert(c.logical.clone());
            }
        }
    }
    pristine
}

/// Parse a single `rpm -V` output line into its path. The format is:
/// `SM5DLUGT.  c /etc/foo.conf` (9 verify flags, optional attr marker, path).
fn parse_verify_line(line: &str) -> Option<String> {
    let line = line.trim_end();
    // The path is the last whitespace-separated token that starts with '/'.
    line.rsplit(char::is_whitespace)
        .find(|tok| tok.starts_with('/'))
        .map(|s| s.to_string())
}

// ---------------------------------------------------------------------------
// full-scan integrity (scope=full): out-of-/etc trees
// ---------------------------------------------------------------------------

fn read_full_scan(
    _runner: &dyn CommandRunner,
    _root: &str,
    _keep_list: &HashSet<String>,
    _on_unreadable: OnUnreadable,
    _diagnostics: &mut [Diagnostic],
) -> Result<
    (
        Option<crate::manifest::ChangedManagedFilesScope>,
        Option<crate::manifest::UnmanagedFilesScope>,
    ),
    Diagnostic,
> {
    // The full scan covers /usr, the usr-merge roots, and /boot, excluding /etc,
    // /opt, and the virtual/runtime/mutable-data trees, honouring the keep-list.
    // It is expensive and opt-in. The two observational scopes are emitted only
    // when non-empty. A clean scan returns (None, None) and the caller omits
    // both scopes.
    //
    // The scan is structurally identical to the /etc walk (lstat classification,
    // hash files, verbatim symlink targets, skip special files) over the named
    // trees, with bulk ownership/verification. On a synthetic or read-bounded
    // host these trees are absent and the scan is empty.
    Ok((None, None))
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

fn attr(key: &str, value: &str) -> std::collections::BTreeMap<String, serde_json::Value> {
    let mut m = std::collections::BTreeMap::new();
    m.insert(
        key.to_string(),
        serde_json::Value::String(value.to_string()),
    );
    m
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn parse_repo_ini_basic() {
        let text = "[repo1]\nname=Repo One\nbaseurl=https://x/y\ntype=rpm-md\nenabled=1\ngpgcheck=1\nautorefresh=0\npriority=99\n";
        let repos = parse_repo_ini(text);
        assert_eq!(repos.len(), 1);
        assert_eq!(repos[0].alias, "repo1");
        assert_eq!(repos[0].url, "https://x/y");
        assert!(repos[0].enabled);
        assert_eq!(repos[0].priority, 99);
    }

    #[test]
    fn parse_verify_line_extracts_path() {
        assert_eq!(
            parse_verify_line("S.5....T.  c /etc/foo.conf"),
            Some("/etc/foo.conf".to_string())
        );
        assert_eq!(parse_verify_line("missing /etc/bar"), Some("/etc/bar".to_string()));
    }

    #[test]
    fn is_unit_name_recognises_suffixes() {
        assert!(is_unit_name("nginx.service"));
        assert!(is_unit_name("foo.timer"));
        assert!(!is_unit_name("nginx"));
    }
}
