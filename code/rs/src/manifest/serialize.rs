// generated from spec: zypper-declarative.spec.md sha256:18253550a5c3d3f1818f0380811cb5dbc98874828693e49fc9cd5cbc923303dd
//
// Serialisation of the Manifest data model: JSON (canonical) and YAML (opt-in),
// plus the safe YAML profile. This module is the mechanism behind
// load-desired-manifest's parse step and describe's serialise step. The choice
// of format is always made by resolve-format; this module performs the read or
// write under the format chosen there.

use crate::error::{Diagnostic, Domain};
use crate::manifest::{Manifest, ManifestFormat};

/// Parse a manifest from bytes under the given format.
///
/// For YAML, the safe profile is enforced BEFORE typed deserialisation:
/// - single document only (reject multi-document streams),
/// - no arbitrary/executable tags (reject `!!python/...`, custom `!tag`),
/// - bounded/disabled anchor-alias expansion (reject anchors/aliases),
/// - explicit typing per the schema (the typed structs apply explicit typing).
/// A YAML input that requires any disabled feature returns a manifest error.
pub fn parse_manifest(text: &str, format: ManifestFormat) -> Result<Manifest, Diagnostic> {
    match format {
        ManifestFormat::Json => parse_json(text),
        ManifestFormat::Yaml => parse_yaml_safe(text),
    }
}

fn parse_json(text: &str) -> Result<Manifest, Diagnostic> {
    serde_json::from_str::<Manifest>(text).map_err(|e| {
        Diagnostic::error(Domain::Manifest, format!("manifest parse error: {}", e))
    })
}

fn parse_yaml_safe(text: &str) -> Result<Manifest, Diagnostic> {
    // 1. Reject multi-document streams. serde_yaml's Deserializer iterator
    //    yields one item per document; more than one document is a disabled
    //    feature.
    let mut docs = serde_yaml::Deserializer::from_str(text);
    let first = match docs.next() {
        Some(d) => d,
        None => {
            return Err(Diagnostic::error(
                Domain::Manifest,
                "unsafe or invalid YAML: empty document stream",
            ))
        }
    };
    if docs.next().is_some() {
        return Err(Diagnostic::error(
            Domain::Manifest,
            "unsafe YAML: multi-document stream rejected (single document only)",
        ));
    }

    // 2. Deserialise the single document into an untyped Value, then walk it to
    //    reject arbitrary/executable tags and anchor/alias expansion.
    let value: serde_yaml::Value = serde::Deserialize::deserialize(first).map_err(|e| {
        Diagnostic::error(Domain::Manifest, format!("YAML parse error: {}", e))
    })?;
    reject_unsafe_yaml(&value)?;

    // 3. Reject anchors/aliases by detecting them at the source level. serde_yaml
    //    expands aliases during parse, so a textual guard catches the unbounded
    //    alias-expansion class. A literal '*' alias reference or a non-merge '&'
    //    anchor in the document body is a disabled feature.
    if yaml_uses_anchors_or_aliases(text) {
        return Err(Diagnostic::error(
            Domain::Manifest,
            "unsafe YAML: anchors/aliases are not permitted (bounded expansion disabled)",
        ));
    }

    // 4. Deserialise into the typed model. Explicit typing is enforced by the
    //    typed structs; a value such as a bare `NO` or `1.10` in a string field
    //    deserialises as the string text under serde_yaml's typed path.
    serde_yaml::from_value::<Manifest>(value).map_err(|e| {
        Diagnostic::error(Domain::Manifest, format!("manifest schema error: {}", e))
    })
}

/// Walk a YAML Value and reject any node carrying a non-standard tag. serde_yaml
/// represents `Value::Tagged` for explicit tags (e.g. `!!python/object`, `!Foo`).
fn reject_unsafe_yaml(value: &serde_yaml::Value) -> Result<(), Diagnostic> {
    use serde_yaml::Value;
    match value {
        Value::Tagged(_) => Err(Diagnostic::error(
            Domain::Manifest,
            "unsafe YAML: explicit/executable tags are not permitted",
        )),
        Value::Sequence(seq) => {
            for v in seq {
                reject_unsafe_yaml(v)?;
            }
            Ok(())
        }
        Value::Mapping(map) => {
            for (k, v) in map {
                reject_unsafe_yaml(k)?;
                reject_unsafe_yaml(v)?;
            }
            Ok(())
        }
        _ => Ok(()),
    }
}

/// Detect anchors (`&name`) or aliases (`*name`) in the YAML source. This is a
/// conservative textual scan that ignores characters inside quoted scalars.
fn yaml_uses_anchors_or_aliases(text: &str) -> bool {
    for line in text.lines() {
        let trimmed = line.trim_start();
        if trimmed.starts_with('#') {
            continue;
        }
        if scan_line_for_anchor_or_alias(trimmed) {
            return true;
        }
    }
    false
}

fn scan_line_for_anchor_or_alias(line: &str) -> bool {
    let bytes = line.as_bytes();
    let mut in_single = false;
    let mut in_double = false;
    let mut i = 0;
    while i < bytes.len() {
        let c = bytes[i] as char;
        match c {
            '\'' if !in_double => in_single = !in_single,
            '"' if !in_single => in_double = !in_double,
            '#' if !in_single && !in_double => break, // rest is a comment
            '&' | '*' if !in_single && !in_double => {
                // An anchor/alias marker is followed by an identifier char and
                // typically preceded by whitespace or a structural char.
                let prev_ok = i == 0
                    || matches!(bytes[i - 1] as char, ' ' | '\t' | ':' | '-' | '[' | '{' | ',');
                let next = bytes.get(i + 1).map(|b| *b as char);
                let next_ok = matches!(next, Some(ch) if ch.is_ascii_alphanumeric() || ch == '_');
                if prev_ok && next_ok {
                    return true;
                }
            }
            _ => {}
        }
        i += 1;
    }
    false
}

/// Serialise a manifest in the given format. JSON is canonical Machinery
/// (pretty-printed for human readability, `_attributes` always an object).
/// YAML is the same data model rendered as YAML (string-typed fields stay
/// quoted scalars so they round-trip correctly).
pub fn serialise_manifest(manifest: &Manifest, format: ManifestFormat) -> Result<String, Diagnostic> {
    match format {
        ManifestFormat::Json => serde_json::to_string_pretty(manifest)
            .map(|mut s| {
                s.push('\n');
                s
            })
            .map_err(|e| {
                Diagnostic::error(Domain::Files, format!("JSON serialise error: {}", e))
            }),
        ManifestFormat::Yaml => serde_yaml::to_string(manifest)
            .map_err(|e| Diagnostic::error(Domain::Files, format!("YAML serialise error: {}", e))),
    }
}

/// Serialise a manifest as canonical JSON (used for the applied record, which is
/// always JSON regardless of the desired manifest's input format).
pub fn serialise_json(manifest: &Manifest) -> Result<String, Diagnostic> {
    serde_json::to_string_pretty(manifest)
        .map(|mut s| {
            s.push('\n');
            s
        })
        .map_err(|e| Diagnostic::error(Domain::Files, format!("JSON serialise error: {}", e)))
}

#[cfg(test)]
mod tests {
    use super::*;

    const VALID_JSON: &str = r#"{
      "meta": { "format_version": 1, "generator": "t", "created_at": "x", "desired_sha256": "" },
      "packages": { "_attributes": { "package_system": "rpm" }, "_elements": [ { "name": "nginx", "version": "", "release": "", "arch": "" } ] }
    }"#;

    #[test]
    fn parses_valid_json() {
        let m = parse_manifest(VALID_JSON, ManifestFormat::Json).unwrap();
        assert!(m.packages.is_some());
        assert_eq!(m.packages.unwrap().elements[0].name, "nginx");
    }

    #[test]
    fn rejects_multidoc_yaml() {
        let y = "meta:\n  format_version: 1\n  generator: t\n  created_at: x\n  desired_sha256: \"\"\n---\nmeta:\n  format_version: 1\n";
        let e = parse_manifest(y, ManifestFormat::Yaml).unwrap_err();
        assert_eq!(e.domain, Domain::Manifest);
    }

    #[test]
    fn rejects_yaml_anchor_alias() {
        let y = "meta: &m\n  format_version: 1\n  generator: t\n  created_at: x\n  desired_sha256: \"\"\nother: *m\n";
        let e = parse_manifest(y, ManifestFormat::Yaml).unwrap_err();
        assert_eq!(e.domain, Domain::Manifest);
    }

    #[test]
    fn accepts_plain_yaml() {
        let y = "meta:\n  format_version: 1\n  generator: \"t\"\n  created_at: \"x\"\n  desired_sha256: \"\"\npackages:\n  _attributes:\n    package_system: \"rpm\"\n  _elements:\n    - name: \"nginx\"\n      version: \"\"\n      release: \"\"\n      arch: \"\"\n";
        let m = parse_manifest(y, ManifestFormat::Yaml).unwrap();
        assert_eq!(m.packages.unwrap().elements[0].name, "nginx");
    }
}
