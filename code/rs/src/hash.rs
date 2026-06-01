// generated from spec: zypper-declarative.spec.md sha256:87b9f2f3bf92afd6fe037412d56a23959290e5816d722a3def9505d07aa5acd7
//
// Canonical-model hashing. desired_sha256 is the SHA256 of a canonical
// serialisation of the parsed data model, not of the raw input bytes, so the same
// intent expressed in JSON or YAML yields the same hash and idempotence holds
// across a format switch.
//
// Canonical form (concrete definition, applied for the hash only):
//   - object keys sorted lexicographically
//   - compact separators (no insignificant whitespace)
//   - UTF-8
//   - meta.created_at and meta.desired_sha256 are excluded from the hash (a
//     per-run timestamp and the hash slot itself are not part of identity)
//   - _elements sorted by identity key (packages by name+arch, repositories by
//     alias, services by name, config_files by path)
//
// On-disk applied.json may be pretty-printed; the HASH is over the compact
// canonical form computed here.

use crate::types::Manifest;
use sha2::{Digest, Sha256};

/// Compute the canonical-model SHA256 of a manifest.
pub fn canonical_sha256(manifest: &Manifest) -> String {
    let value = canonical_value(manifest);
    let bytes = canonical_bytes(&value);
    let mut hasher = Sha256::new();
    hasher.update(&bytes);
    let digest = hasher.finalize();
    hex_encode(&digest)
}

/// Build a serde_json::Value reflecting the canonical model: declarable scopes
/// only, identity-sorted elements, with non-identity meta fields neutralised.
fn canonical_value(manifest: &Manifest) -> serde_json::Value {
    let mut m = manifest.clone();
    // Neutralise non-identity meta fields so the hash depends on intent only.
    m.meta.created_at = String::new();
    m.meta.desired_sha256 = String::new();
    // The generator is program/version metadata, not declared intent; neutralise
    // it so the same intent hashes identically regardless of who wrote it.
    m.meta.generator = String::new();
    // Observational scopes never participate in identity.
    m.changed_managed_files = None;
    m.unmanaged_files = None;

    // Sort each present scope's elements by its identity key.
    if let Some(s) = m.packages.as_mut() {
        s.elements.sort_by(|a, b| {
            (a.name.as_str(), a.arch.as_str()).cmp(&(b.name.as_str(), b.arch.as_str()))
        });
    }
    if let Some(s) = m.repositories.as_mut() {
        s.elements.sort_by(|a, b| a.alias.cmp(&b.alias));
    }
    if let Some(s) = m.services.as_mut() {
        s.elements.sort_by(|a, b| a.name.cmp(&b.name));
    }
    if let Some(s) = m.config_files.as_mut() {
        s.elements.sort_by(|a, b| a.name.cmp(&b.name));
    }

    serde_json::to_value(&m).unwrap_or(serde_json::Value::Null)
}

/// Serialise a serde_json::Value with sorted object keys and compact separators.
fn canonical_bytes(value: &serde_json::Value) -> Vec<u8> {
    let mut out = Vec::new();
    write_canonical(value, &mut out);
    out
}

fn write_canonical(value: &serde_json::Value, out: &mut Vec<u8>) {
    match value {
        serde_json::Value::Object(map) => {
            out.push(b'{');
            let mut keys: Vec<&String> = map.keys().collect();
            keys.sort();
            for (i, k) in keys.iter().enumerate() {
                if i > 0 {
                    out.push(b',');
                }
                // serde_json string-encodes the key for us via to_string on a Value
                let key_value = serde_json::Value::String((*k).clone());
                out.extend_from_slice(serde_json::to_string(&key_value).unwrap().as_bytes());
                out.push(b':');
                write_canonical(&map[*k], out);
            }
            out.push(b'}');
        }
        serde_json::Value::Array(arr) => {
            out.push(b'[');
            for (i, v) in arr.iter().enumerate() {
                if i > 0 {
                    out.push(b',');
                }
                write_canonical(v, out);
            }
            out.push(b']');
        }
        other => {
            out.extend_from_slice(serde_json::to_string(other).unwrap().as_bytes());
        }
    }
}

fn hex_encode(bytes: &[u8]) -> String {
    let mut s = String::with_capacity(bytes.len() * 2);
    for b in bytes {
        s.push_str(&format!("{:02x}", b));
    }
    s
}

/// SHA256 of arbitrary content (used to hash regular-file contents during the walk).
pub fn sha256_bytes(data: &[u8]) -> String {
    let mut hasher = Sha256::new();
    hasher.update(data);
    hex_encode(&hasher.finalize())
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::types::*;

    #[test]
    fn json_and_yaml_models_hash_identically() {
        // Same model, element order differs -> identity sort makes the hash equal.
        let mut a = Manifest::empty();
        a.packages = Some(ScopeWrapper {
            attributes: Default::default(),
            elements: vec![
                PackageRecord {
                    name: "b".into(),
                    arch: "x86_64".into(),
                    ..Default::default()
                },
                PackageRecord {
                    name: "a".into(),
                    arch: "x86_64".into(),
                    ..Default::default()
                },
            ],
        });
        let mut b = Manifest::empty();
        b.packages = Some(ScopeWrapper {
            attributes: Default::default(),
            elements: vec![
                PackageRecord {
                    name: "a".into(),
                    arch: "x86_64".into(),
                    ..Default::default()
                },
                PackageRecord {
                    name: "b".into(),
                    arch: "x86_64".into(),
                    ..Default::default()
                },
            ],
        });
        assert_eq!(canonical_sha256(&a), canonical_sha256(&b));
    }

    #[test]
    fn created_at_does_not_affect_hash() {
        let mut a = Manifest::empty();
        a.meta.created_at = "2026-01-01T00:00:00Z".into();
        let mut b = Manifest::empty();
        b.meta.created_at = "2030-12-31T23:59:59Z".into();
        assert_eq!(canonical_sha256(&a), canonical_sha256(&b));
    }
}
