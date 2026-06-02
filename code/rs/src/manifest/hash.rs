// generated from spec: zypper-declarative.spec.md
// sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
//
// Canonical-model hashing: desired_sha256 is the SHA256 of a canonical
// serialisation of the PARSED data model, not of the raw input bytes, so the
// same intent expressed in JSON or YAML yields the same hash (idempotence holds
// across a format switch).
//
// Canonical form (concrete definition per the Rust decisions hints):
//   - keys sorted (recursively),
//   - compact separators (no whitespace), UTF-8,
//   - _elements sorted by identity key (packages by name+arch, repositories by
//     alias, services by name, config_files by path),
//   - meta.created_at is excluded (a per-run timestamp), and meta.desired_sha256
//     is excluded (it is computed FROM this hash and is "" in a desired manifest).

use crate::manifest::Manifest;
use serde_json::{Map, Value};
use sha2::{Digest, Sha256};

/// Compute the canonical-model SHA256 of a parsed manifest.
pub fn desired_sha256(manifest: &Manifest) -> String {
    let mut v = serde_json::to_value(manifest).unwrap_or(Value::Null);
    strip_volatile(&mut v);
    sort_elements(&mut v);
    let canonical = canonical_string(&v);
    let mut hasher = Sha256::new();
    hasher.update(canonical.as_bytes());
    let digest = hasher.finalize();
    hex(&digest)
}

/// Hex-encode bytes.
pub fn hex(bytes: &[u8]) -> String {
    let mut s = String::with_capacity(bytes.len() * 2);
    for b in bytes {
        s.push_str(&format!("{:02x}", b));
    }
    s
}

/// SHA256 of arbitrary bytes (used for file content digests).
pub fn sha256_bytes(bytes: &[u8]) -> String {
    let mut hasher = Sha256::new();
    hasher.update(bytes);
    hex(&hasher.finalize())
}

// Remove fields that must not affect identity: meta.created_at and
// meta.desired_sha256.
fn strip_volatile(v: &mut Value) {
    if let Some(meta) = v.get_mut("meta").and_then(|m| m.as_object_mut()) {
        meta.remove("created_at");
        meta.remove("desired_sha256");
    }
}

// Sort each scope's _elements by its identity key.
fn sort_elements(v: &mut Value) {
    let scopes: [(&str, fn(&Value) -> String); 4] = [
        ("packages", pkg_key),
        ("repositories", repo_key),
        ("services", svc_key),
        ("config_files", path_key),
    ];
    for (scope, keyfn) in scopes {
        if let Some(elems) = v
            .get_mut(scope)
            .and_then(|s| s.get_mut("_elements"))
            .and_then(|e| e.as_array_mut())
        {
            elems.sort_by(|a, b| keyfn(a).cmp(&keyfn(b)));
        }
    }
}

fn pkg_key(e: &Value) -> String {
    let name = e.get("name").and_then(|x| x.as_str()).unwrap_or("");
    let arch = e.get("arch").and_then(|x| x.as_str()).unwrap_or("");
    format!("{}\u{0}{}", name, arch)
}
fn repo_key(e: &Value) -> String {
    e.get("alias")
        .and_then(|x| x.as_str())
        .unwrap_or("")
        .to_string()
}
fn svc_key(e: &Value) -> String {
    e.get("name")
        .and_then(|x| x.as_str())
        .unwrap_or("")
        .to_string()
}
fn path_key(e: &Value) -> String {
    e.get("name")
        .and_then(|x| x.as_str())
        .unwrap_or("")
        .to_string()
}

// Produce a canonical, compact JSON string with object keys sorted recursively.
fn canonical_string(v: &Value) -> String {
    let mut out = String::new();
    write_canonical(v, &mut out);
    out
}

fn write_canonical(v: &Value, out: &mut String) {
    match v {
        Value::Object(map) => {
            // sort keys
            let mut keys: Vec<&String> = map.keys().collect();
            keys.sort();
            out.push('{');
            for (i, k) in keys.iter().enumerate() {
                if i > 0 {
                    out.push(',');
                }
                write_json_string(k, out);
                out.push(':');
                write_canonical(&map[*k], out);
            }
            out.push('}');
        }
        Value::Array(arr) => {
            out.push('[');
            for (i, e) in arr.iter().enumerate() {
                if i > 0 {
                    out.push(',');
                }
                write_canonical(e, out);
            }
            out.push(']');
        }
        Value::String(s) => write_json_string(s, out),
        Value::Null => out.push_str("null"),
        Value::Bool(b) => out.push_str(if *b { "true" } else { "false" }),
        Value::Number(n) => out.push_str(&n.to_string()),
    }
}

fn write_json_string(s: &str, out: &mut String) {
    // Mirror serde_json string escaping for canonical stability.
    let mut tmp = Map::new();
    tmp.insert(String::new(), Value::String(s.to_string()));
    let rendered = serde_json::to_string(&Value::String(s.to_string())).unwrap();
    let _ = tmp;
    out.push_str(&rendered);
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::manifest::*;

    fn sample(order_swapped: bool) -> Manifest {
        let mut m = Manifest::new_actual("2026-01-01T00:00:00Z".into());
        let mut pkgs = PackagesScope::with_attr("package_system", "rpm");
        let a = PackageRecord {
            name: "alpha".into(),
            ..Default::default()
        };
        let b = PackageRecord {
            name: "beta".into(),
            ..Default::default()
        };
        if order_swapped {
            pkgs.elements = vec![b, a];
        } else {
            pkgs.elements = vec![a, b];
        }
        m.packages = Some(pkgs);
        m
    }

    #[test]
    fn hash_is_order_independent_for_elements() {
        let h1 = desired_sha256(&sample(false));
        let h2 = desired_sha256(&sample(true));
        assert_eq!(
            h1, h2,
            "element ordering must not change the canonical hash"
        );
    }

    #[test]
    fn hash_ignores_created_at() {
        let mut m1 = sample(false);
        let mut m2 = sample(false);
        m1.meta.created_at = "2026-01-01T00:00:00Z".into();
        m2.meta.created_at = "2030-12-31T23:59:59Z".into();
        assert_eq!(desired_sha256(&m1), desired_sha256(&m2));
    }

    #[test]
    fn hash_is_64_hex_chars() {
        let h = desired_sha256(&sample(false));
        assert_eq!(h.len(), 64);
        assert!(h.chars().all(|c| c.is_ascii_hexdigit()));
    }
}
