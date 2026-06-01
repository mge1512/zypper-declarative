// generated from spec: zypper-declarative.spec.md sha256:18253550a5c3d3f1818f0380811cb5dbc98874828693e49fc9cd5cbc923303dd
//
// Canonical-model hashing. `desired_sha256` is the SHA256 of a canonical
// serialisation of the PARSED data model (format-independent), not of the raw
// input bytes — so the same intent expressed in JSON or YAML yields the same
// hash and idempotence holds across a format switch.
//
// The canonical form: only the declarable scopes participate, `meta.created_at`
// and `meta.desired_sha256` are excluded (they are per-run/derived), object
// keys are emitted in sorted order, separators are compact, and `_elements` are
// sorted by their identity key (packages by name+arch, repositories by alias,
// services by name, config_files by path). The applied.json on disk may be
// pretty-printed; the HASH is over this compact canonical form.

use super::Manifest;
use sha2::{Digest, Sha256};

/// Compute the canonical-model SHA256 hash of a parsed manifest.
pub fn desired_sha256(manifest: &Manifest) -> String {
    let canonical = canonical_value(manifest);
    let bytes = to_canonical_bytes(&canonical);
    let mut hasher = Sha256::new();
    hasher.update(&bytes);
    let digest = hasher.finalize();
    hex_lower(&digest)
}

/// Build a serde_json::Value capturing the canonical data model: the declarable
/// scopes only, with `_elements` sorted by identity key, and a meta block that
/// excludes created_at and desired_sha256.
fn canonical_value(manifest: &Manifest) -> serde_json::Value {
    use serde_json::{json, Map, Value};

    let mut root = Map::new();

    // meta: only format_version and generator are part of the canonical
    // identity. created_at and desired_sha256 are excluded.
    root.insert(
        "meta".to_string(),
        json!({
            "format_version": manifest.meta.format_version,
            "generator": manifest.meta.generator,
        }),
    );

    if let Some(scope) = &manifest.packages {
        let mut elems: Vec<&super::PackageRecord> = scope.elements.iter().collect();
        elems.sort_by(|a, b| (&a.name, &a.arch).cmp(&(&b.name, &b.arch)));
        let arr: Vec<Value> = elems
            .iter()
            .map(|r| {
                json!({
                    "name": r.name,
                    "version": r.version,
                    "release": r.release,
                    "arch": r.arch,
                })
            })
            .collect();
        root.insert("packages".to_string(), scope_value(&scope.attributes, arr));
    }

    if let Some(scope) = &manifest.repositories {
        let mut elems: Vec<&super::RepositoryRecord> = scope.elements.iter().collect();
        elems.sort_by(|a, b| a.alias.cmp(&b.alias));
        let arr: Vec<Value> = elems
            .iter()
            .map(|r| {
                json!({
                    "alias": r.alias,
                    "name": r.name,
                    "url": r.url,
                    "type": r.repo_type,
                    "enabled": r.enabled,
                    "gpgcheck": r.gpgcheck,
                    "autorefresh": r.autorefresh,
                    "priority": r.priority,
                })
            })
            .collect();
        root.insert("repositories".to_string(), scope_value(&scope.attributes, arr));
    }

    if let Some(scope) = &manifest.services {
        let mut elems: Vec<&super::ServiceRecord> = scope.elements.iter().collect();
        elems.sort_by(|a, b| a.name.cmp(&b.name));
        let arr: Vec<Value> = elems
            .iter()
            .map(|r| {
                json!({
                    "name": r.name,
                    "state": r.state,
                })
            })
            .collect();
        root.insert("services".to_string(), scope_value(&scope.attributes, arr));
    }

    if let Some(scope) = &manifest.config_files {
        let mut elems: Vec<&super::ManagedFileRecord> = scope.elements.iter().collect();
        elems.sort_by(|a, b| a.name.cmp(&b.name));
        let arr: Vec<Value> = elems
            .iter()
            .map(|r| {
                // content_ref and package_name are not part of file IDENTITY for
                // the canonical hash: identity is name + type + sha256 (file) or
                // target (link) + mode/user/group. content_ref is an apply-time
                // supply detail; package_name is observational ownership.
                json!({
                    "name": r.name,
                    "type": r.file_type,
                    "mode": r.mode,
                    "user": r.user,
                    "group": r.group,
                    "sha256": r.sha256,
                    "target": r.target,
                })
            })
            .collect();
        root.insert("config_files".to_string(), scope_value(&scope.attributes, arr));
    }

    Value::Object(root)
}

fn scope_value(
    attributes: &std::collections::BTreeMap<String, serde_json::Value>,
    elements: Vec<serde_json::Value>,
) -> serde_json::Value {
    use serde_json::{Map, Value};
    let mut m = Map::new();
    let mut attrs = Map::new();
    for (k, v) in attributes.iter() {
        attrs.insert(k.clone(), v.clone());
    }
    m.insert("_attributes".to_string(), Value::Object(attrs));
    m.insert("_elements".to_string(), Value::Array(elements));
    Value::Object(m)
}

/// Serialise a serde_json::Value with sorted object keys and compact
/// separators, deterministically. BTreeMap-based maps in serde_json with the
/// `preserve_order` feature off already iterate keys in sorted order; to be
/// robust we serialise via a recursive canonicaliser.
fn to_canonical_bytes(value: &serde_json::Value) -> Vec<u8> {
    let mut out = String::new();
    write_canonical(value, &mut out);
    out.into_bytes()
}

fn write_canonical(value: &serde_json::Value, out: &mut String) {
    use serde_json::Value;
    match value {
        Value::Null => out.push_str("null"),
        Value::Bool(b) => out.push_str(if *b { "true" } else { "false" }),
        Value::Number(n) => out.push_str(&n.to_string()),
        Value::String(s) => {
            out.push_str(&serde_json::to_string(s).unwrap_or_else(|_| "\"\"".to_string()))
        }
        Value::Array(arr) => {
            out.push('[');
            for (i, v) in arr.iter().enumerate() {
                if i > 0 {
                    out.push(',');
                }
                write_canonical(v, out);
            }
            out.push(']');
        }
        Value::Object(map) => {
            // Sort keys for determinism.
            let mut keys: Vec<&String> = map.keys().collect();
            keys.sort();
            out.push('{');
            for (i, k) in keys.iter().enumerate() {
                if i > 0 {
                    out.push(',');
                }
                out.push_str(&serde_json::to_string(k).unwrap_or_else(|_| "\"\"".to_string()));
                out.push(':');
                write_canonical(&map[*k], out);
            }
            out.push('}');
        }
    }
}

fn hex_lower(bytes: &[u8]) -> String {
    let mut s = String::with_capacity(bytes.len() * 2);
    for b in bytes {
        s.push_str(&format!("{:02x}", b));
    }
    s
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::manifest::*;

    fn pkg(name: &str, v: &str) -> PackageRecord {
        PackageRecord {
            name: name.to_string(),
            version: v.to_string(),
            release: "1".to_string(),
            arch: "x86_64".to_string(),
        }
    }

    #[test]
    fn order_independent_for_elements() {
        let mut a = Manifest::empty();
        a.packages = Some(ScopeWrapper {
            attributes: Default::default(),
            elements: vec![pkg("bash", "5.2"), pkg("nginx", "1.0")],
        });
        let mut b = Manifest::empty();
        b.packages = Some(ScopeWrapper {
            attributes: Default::default(),
            elements: vec![pkg("nginx", "1.0"), pkg("bash", "5.2")],
        });
        assert_eq!(desired_sha256(&a), desired_sha256(&b));
    }

    #[test]
    fn created_at_excluded_from_hash() {
        let mut a = Manifest::empty();
        a.meta.created_at = "2020-01-01T00:00:00Z".to_string();
        let mut b = Manifest::empty();
        b.meta.created_at = "2026-06-01T00:00:00Z".to_string();
        assert_eq!(desired_sha256(&a), desired_sha256(&b));
    }

    #[test]
    fn hash_is_64_hex() {
        let h = desired_sha256(&Manifest::empty());
        assert_eq!(h.len(), 64);
        assert!(h.chars().all(|c| c.is_ascii_hexdigit()));
    }
}
