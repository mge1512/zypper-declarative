// generated from spec: zypper-declarative.spec.md
// sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
//
// Packages actual state: query the rpmdb under <root> and build a PackagesScope
// with every record's name, version, release, and arch populated.

use crate::interfaces::CommandRunner;
use crate::manifest::PackageRecord;

pub enum PackagesResult {
    Records(Vec<PackageRecord>),
    Unreadable(String),
}

/// Query the installed package set under `root`. For root "/" no --root flag is
/// passed; for a non-/ root, rpm's --root is used.
pub fn read_packages(runner: &dyn CommandRunner, root: &str) -> PackagesResult {
    let fmt = "%{NAME}\\t%{VERSION}\\t%{RELEASE}\\t%{ARCH}\\n";
    let mut args: Vec<String> = Vec::new();
    if root != "/" {
        args.push("--root".to_string());
        args.push(root.to_string());
    }
    args.push("-qa".to_string());
    args.push("--queryformat".to_string());
    args.push(fmt.to_string());
    let argref: Vec<&str> = args.iter().map(|s| s.as_str()).collect();
    let res = runner.run("rpm", &argref);
    if res.spawn_failed {
        return PackagesResult::Unreadable(format!(
            "rpmdb under {} (rpm: {})",
            root,
            res.stderr.trim()
        ));
    }
    // rpm -qa returns 0 normally; a genuine rpmdb open failure yields empty
    // stdout with a non-empty stderr.
    if !res.success && res.stdout.trim().is_empty() && !res.stderr.trim().is_empty() {
        return PackagesResult::Unreadable(format!("rpmdb under {}: {}", root, res.stderr.trim()));
    }
    PackagesResult::Records(parse_packages(&res.stdout))
}

pub fn parse_packages(stdout: &str) -> Vec<PackageRecord> {
    let mut recs = Vec::new();
    for line in stdout.lines() {
        let line = line.trim_end();
        if line.is_empty() {
            continue;
        }
        let parts: Vec<&str> = line.split('\t').collect();
        if parts.len() >= 4 {
            recs.push(PackageRecord {
                name: parts[0].to_string(),
                version: parts[1].to_string(),
                release: parts[2].to_string(),
                arch: parts[3].to_string(),
            });
        }
    }
    recs
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn parses_nevra_lines() {
        let out = "nginx\t1.25.0\t1.1\tx86_64\nbash\t5.2\t3.2\tx86_64\n";
        let recs = parse_packages(out);
        assert_eq!(recs.len(), 2);
        assert_eq!(recs[0].name, "nginx");
        assert_eq!(recs[0].version, "1.25.0");
        assert_eq!(recs[0].arch, "x86_64");
    }
}
