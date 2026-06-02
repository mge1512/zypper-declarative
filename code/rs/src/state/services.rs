// generated from spec: zypper-declarative.spec.md
// sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
//
// Services actual state: query unit enablement under <root> OFFLINE via
// `systemctl --root <root>`, normalising each unit's state to enabled,
// disabled, or masked. Purely-static units are omitted (not declarable).

use crate::interfaces::CommandRunner;
use crate::manifest::ServiceRecord;

pub enum ServicesResult {
    Records(Vec<ServiceRecord>),
    Unreadable(String),
}

/// Read unit enablement under `root`.
pub fn read_services(runner: &dyn CommandRunner, root: &str) -> ServicesResult {
    let mut args: Vec<String> = Vec::new();
    if root != "/" {
        args.push("--root".to_string());
        args.push(root.to_string());
    }
    args.push("list-unit-files".to_string());
    args.push("--no-legend".to_string());
    args.push("--no-pager".to_string());
    args.push("--plain".to_string());
    let argref: Vec<&str> = args.iter().map(|s| s.as_str()).collect();
    let res = runner.run("systemctl", &argref);
    if res.spawn_failed {
        return ServicesResult::Unreadable(format!(
            "unit enablement under {} (systemctl: {})",
            root,
            res.stderr.trim()
        ));
    }
    if !res.success && res.stdout.trim().is_empty() && !res.stderr.trim().is_empty() {
        return ServicesResult::Unreadable(format!(
            "unit enablement under {}: {}",
            root,
            res.stderr.trim()
        ));
    }
    ServicesResult::Records(parse_unit_files(&res.stdout))
}

/// Parse `systemctl list-unit-files` output, keeping only declarable states.
pub fn parse_unit_files(stdout: &str) -> Vec<ServiceRecord> {
    let mut recs = Vec::new();
    for line in stdout.lines() {
        let line = line.trim();
        if line.is_empty() {
            continue;
        }
        let mut it = line.split_whitespace();
        let name = match it.next() {
            Some(n) => n,
            None => continue,
        };
        let raw_state = match it.next() {
            Some(s) => s,
            None => continue,
        };
        // Only declarable unit names (per UnitName refinement).
        if !is_declarable_unit(name) {
            continue;
        }
        let state = match raw_state {
            "enabled" | "enabled-runtime" => "enabled",
            "disabled" => "disabled",
            "masked" | "masked-runtime" => "masked",
            // static, indirect, generated, transient, alias, linked, etc. are
            // not declarable enablement states -> omit.
            _ => continue,
        };
        recs.push(ServiceRecord {
            name: name.to_string(),
            state: state.to_string(),
        });
    }
    recs
}

fn is_declarable_unit(name: &str) -> bool {
    name.ends_with(".service")
        || name.ends_with(".timer")
        || name.ends_with(".socket")
        || name.ends_with(".target")
        || name.ends_with(".path")
        || name.ends_with(".mount")
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn keeps_only_declarable_states() {
        let out = "nginx.service enabled\nsshd.service disabled\nfoo.service static\nbar.service masked\nbaz.timer enabled\n";
        let recs = parse_unit_files(out);
        let names: Vec<&str> = recs.iter().map(|r| r.name.as_str()).collect();
        assert!(names.contains(&"nginx.service"));
        assert!(names.contains(&"sshd.service"));
        assert!(names.contains(&"bar.service"));
        assert!(names.contains(&"baz.timer"));
        assert!(!names.contains(&"foo.service"), "static unit omitted");
        assert_eq!(
            recs.iter()
                .find(|r| r.name == "nginx.service")
                .unwrap()
                .state,
            "enabled"
        );
    }
}
