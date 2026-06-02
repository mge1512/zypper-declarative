// generated from spec: zypper-declarative.spec.md
// sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
//
// Repositories actual state: read the on-disk zypp repository configuration
// from <root>/etc/zypp/repos.d/*.repo (INI). No network refresh, no privileged
// cache. A genuinely-empty readable repos.d yields an empty list (the scope is
// omitted by the caller); a repos.d that cannot be listed is an unreadable
// source error reported by the caller.

use crate::manifest::RepositoryRecord;
use std::path::Path;

pub enum ReposResult {
    Records(Vec<RepositoryRecord>),
    Unreadable(String),
}

/// Read all .repo files under <root>/etc/zypp/repos.d.
pub fn read_repositories(root: &str) -> ReposResult {
    let dir = Path::new(root).join("etc/zypp/repos.d");
    if !dir.exists() {
        // A missing repos.d is treated as genuinely-empty: nothing declared.
        return ReposResult::Records(Vec::new());
    }
    let entries = match std::fs::read_dir(&dir) {
        Ok(e) => e,
        Err(_) => return ReposResult::Unreadable(dir.display().to_string()),
    };
    let mut files: Vec<std::path::PathBuf> = Vec::new();
    for entry in entries {
        let entry = match entry {
            Ok(e) => e,
            Err(_) => return ReposResult::Unreadable(dir.display().to_string()),
        };
        let p = entry.path();
        if p.extension().and_then(|e| e.to_str()) == Some("repo") {
            files.push(p);
        }
    }
    files.sort();
    let mut records = Vec::new();
    for f in files {
        let text = match std::fs::read_to_string(&f) {
            Ok(t) => t,
            Err(_) => return ReposResult::Unreadable(f.display().to_string()),
        };
        records.extend(parse_repo_file(&text));
    }
    ReposResult::Records(records)
}

/// Parse a .repo INI file into RepositoryRecords (one per section).
pub fn parse_repo_file(text: &str) -> Vec<RepositoryRecord> {
    let mut out = Vec::new();
    let mut cur: Option<RepositoryRecord> = None;
    let mut have_section = false;

    for raw in text.lines() {
        let line = raw.trim();
        if line.is_empty() || line.starts_with('#') || line.starts_with(';') {
            continue;
        }
        if line.starts_with('[') && line.ends_with(']') {
            if have_section {
                if let Some(r) = cur.take() {
                    out.push(r);
                }
            }
            let alias = line[1..line.len() - 1].to_string();
            cur = Some(RepositoryRecord {
                alias,
                r#type: "rpm-md".to_string(),
                enabled: false,
                gpgcheck: false,
                autorefresh: false,
                priority: 0,
                ..Default::default()
            });
            have_section = true;
            continue;
        }
        if let Some((k, v)) = line.split_once('=') {
            let key = k.trim();
            let val = v.trim();
            if let Some(r) = cur.as_mut() {
                match key {
                    "name" => r.name = val.to_string(),
                    "baseurl" => r.url = val.to_string(),
                    "type" => r.r#type = val.to_string(),
                    "enabled" => r.enabled = parse_bool(val),
                    "gpgcheck" => r.gpgcheck = parse_bool(val),
                    "autorefresh" => r.autorefresh = parse_bool(val),
                    "priority" => r.priority = val.parse().unwrap_or(0),
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

fn parse_bool(v: &str) -> bool {
    matches!(v.trim(), "1" | "true" | "yes" | "on")
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn parses_two_sections() {
        let text = "[repo-a]\nname=Repo A\nbaseurl=https://a/\nenabled=1\ngpgcheck=1\npriority=99\n\n[repo-b]\nname=Repo B\nbaseurl=https://b/\nenabled=0\n";
        let recs = parse_repo_file(text);
        assert_eq!(recs.len(), 2);
        assert_eq!(recs[0].alias, "repo-a");
        assert_eq!(recs[0].url, "https://a/");
        assert!(recs[0].enabled);
        assert_eq!(recs[0].priority, 99);
        assert_eq!(recs[1].alias, "repo-b");
        assert!(!recs[1].enabled);
    }
}
