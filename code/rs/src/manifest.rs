// generated from spec: zypper-declarative.spec.md sha256:87b9f2f3bf92afd6fe037412d56a23959290e5816d722a3def9505d07aa5acd7
//
// BEHAVIOR/INTERNAL: load-desired-manifest, plus manifest serialisation (JSON
// canonical and YAML opt-in) and schema validation. Reads and validates the
// desired manifest into the shared data model, selecting the input serialisation
// via resolve-format, applying a safe YAML profile when the input is YAML, and
// computing the manifest's canonical-model identity hash.

use crate::error::{Diagnostic, Domain};
use crate::format::resolve_format;
use crate::hash::canonical_sha256;
use crate::types::{Manifest, ManifestFormat};

/// Result of load-desired-manifest: the parsed manifest and its canonical hash.
pub struct LoadedManifest {
    pub manifest: Manifest,
    pub desired_sha256: String,
}

/// load-desired-manifest.
///
/// STEPS:
/// 1. Read the file. On read failure return an invocation error.
/// 2. Resolve the input format via resolve-format. (Unknown explicit format is
///    handled at the CLI layer before this is called.)
/// 3. Parse into the data model. JSON via serde_json; YAML under a safe profile.
/// 4. Validate: meta.format_version == 1; observational scopes must not be present
///    with non-empty _elements.
/// 5. (signature verification is a CONFIG hook; off by default in this build.)
/// 6. Compute desired_sha256 as the canonical-model hash.
pub fn load_desired_manifest(
    manifest_path: &str,
    explicit_format: Option<ManifestFormat>,
    default_format: ManifestFormat,
) -> Result<LoadedManifest, Diagnostic> {
    // Step 1: read.
    let bytes = std::fs::read(manifest_path).map_err(|e| {
        Diagnostic::error(
            Domain::Invocation,
            format!("manifest unreadable: {}: {}", manifest_path, e),
        )
    })?;

    // Step 2: resolve format from the operative path.
    let fmt = resolve_format(explicit_format, Some(manifest_path), default_format);

    // Step 3: parse.
    let manifest = parse_manifest(&bytes, fmt)?;

    // Step 4: validate schema.
    validate_manifest(&manifest)?;

    // Step 6: canonical-model hash.
    let desired_sha256 = canonical_sha256(&manifest);

    Ok(LoadedManifest {
        manifest,
        desired_sha256,
    })
}

/// Parse manifest bytes into the model under the resolved format.
pub fn parse_manifest(bytes: &[u8], fmt: ManifestFormat) -> Result<Manifest, Diagnostic> {
    match fmt {
        ManifestFormat::Json => serde_json::from_slice::<Manifest>(bytes).map_err(|e| {
            Diagnostic::error(
                Domain::Manifest,
                format!("manifest schema violation: {}", e),
            )
        }),
        ManifestFormat::Yaml => parse_yaml_safe(bytes),
    }
}

/// Parse a state dump (an actual-state Manifest) under the resolved format. The
/// difference from a desired manifest is that observational scopes are tolerated;
/// a malformed document is an invocation error (the caller decides exit 2).
pub fn parse_state_dump(bytes: &[u8], fmt: ManifestFormat) -> Result<Manifest, Diagnostic> {
    let result = match fmt {
        ManifestFormat::Json => serde_json::from_slice::<Manifest>(bytes).map_err(|e| {
            Diagnostic::error(Domain::Invocation, format!("malformed state dump: {}", e))
        }),
        ManifestFormat::Yaml => parse_yaml_safe(bytes)
            .map_err(|_| Diagnostic::error(Domain::Invocation, "malformed state dump".to_string())),
    };
    let m = result?;
    if m.meta.format_version != 1 {
        return Err(Diagnostic::error(
            Domain::Invocation,
            format!(
                "malformed state dump: format_version {} != 1",
                m.meta.format_version
            ),
        ));
    }
    Ok(m)
}

/// Parse YAML under the safe profile:
///   - non-code-executing loader only (serde_yaml never executes tags)
///   - reject multi-document streams (single document only)
///   - reject explicit tags (no arbitrary/executable tags)
///   - explicit typing per the schema (the typed deserialise rejects coerced types)
///
/// serde_yaml is a pure-data loader (no executable tags by construction). We
/// additionally walk the raw text for multi-document markers and parse via the
/// Value API which yields a single document. A YAML input that requires any
/// disabled feature returns a manifest error rather than being parsed.
fn parse_yaml_safe(bytes: &[u8]) -> Result<Manifest, Diagnostic> {
    let text = std::str::from_utf8(bytes).map_err(|_| {
        Diagnostic::error(Domain::Manifest, "manifest is not valid UTF-8".to_string())
    })?;

    // Reject multi-document streams: count document boundaries. A leading `---`
    // is permitted; an interior `---` or `...` document separator is rejected.
    reject_multidocument(text)?;

    // Reject explicit/arbitrary tags (anything of the form `!tag` or `!!python/...`).
    // The schema uses no tags, so any `!`-prefixed tag token is a disabled feature.
    reject_explicit_tags(text)?;

    // Parse into a single untyped Value first (this also fails on multiple docs).
    let value: serde_yaml::Value = serde_yaml::from_str(text).map_err(|e| {
        let msg = e.to_string();
        if msg.contains("deserializing from YAML containing more than one document")
            || msg.contains("more than one document")
        {
            Diagnostic::error(
                Domain::Manifest,
                "YAML multi-document stream rejected".to_string(),
            )
        } else {
            Diagnostic::error(
                Domain::Manifest,
                format!("manifest schema violation: {}", e),
            )
        }
    })?;

    // Deserialize the value into the typed model with explicit typing. serde_yaml
    // deserialises a string field only from a YAML string; a bare 0600 or NO would
    // be a non-string scalar and is rejected by the typed deserialise.
    let manifest: Manifest = serde_yaml::from_value(value).map_err(|e| {
        Diagnostic::error(
            Domain::Manifest,
            format!("manifest schema violation: {}", e),
        )
    })?;
    Ok(manifest)
}

fn reject_multidocument(text: &str) -> Result<(), Diagnostic> {
    let mut doc_markers = 0usize;
    for line in text.lines() {
        let t = line.trim_end();
        if t == "---" {
            doc_markers += 1;
        } else if t == "..." {
            // explicit document end
            doc_markers += 1;
        }
    }
    // A single leading `---` is one marker and is harmless. Two or more document
    // markers indicate a multi-document stream, which the safe profile rejects.
    if doc_markers >= 2 {
        return Err(Diagnostic::error(
            Domain::Manifest,
            "YAML multi-document stream rejected (safe profile permits a single document)"
                .to_string(),
        ));
    }
    Ok(())
}

fn reject_explicit_tags(text: &str) -> Result<(), Diagnostic> {
    // A YAML explicit tag begins with `!` after a key/value/sequence indicator.
    // The schema uses no tags; reject any `!`-introduced tag token (e.g. `!!str`,
    // `!!python/object`, `!Custom`). We scan tokens outside of quoted strings.
    let mut in_single = false;
    let mut in_double = false;
    let bytes = text.as_bytes();
    let mut i = 0;
    while i < bytes.len() {
        let c = bytes[i] as char;
        match c {
            '\'' if !in_double => in_single = !in_single,
            '"' if !in_single => in_double = !in_double,
            '!' if !in_single && !in_double => {
                // A `!` that is not inside a quoted scalar. Treat as a tag indicator
                // unless it is part of plain text following a non-space (rare). We
                // require the `!` to be preceded by start-of-token (space, ':', '-',
                // '[', '{', ',', or line start) to count as a tag.
                let prev = if i == 0 { ' ' } else { bytes[i - 1] as char };
                if prev == ' '
                    || prev == ':'
                    || prev == '-'
                    || prev == '['
                    || prev == '{'
                    || prev == ','
                    || prev == '\n'
                    || prev == '\t'
                {
                    return Err(Diagnostic::error(
                        Domain::Manifest,
                        "YAML explicit tag rejected (safe profile forbids arbitrary/executable tags)"
                            .to_string(),
                    ));
                }
            }
            _ => {}
        }
        i += 1;
    }
    Ok(())
}

/// Validate a desired manifest against the schema.
///
/// - meta.format_version must be 1.
/// - The observational scopes (changed_managed_files, unmanaged_files) are not
///   declarable: if either is present with a non-empty _elements, return a
///   manifest error. An empty or absent observational scope is tolerated and
///   dropped.
pub fn validate_manifest(manifest: &Manifest) -> Result<(), Diagnostic> {
    if manifest.meta.format_version != 1 {
        return Err(Diagnostic::error(
            Domain::Manifest,
            format!(
                "manifest invalid: meta.format_version {} != 1",
                manifest.meta.format_version
            ),
        ));
    }
    if let Some(s) = manifest.changed_managed_files.as_ref() {
        if !s.elements.is_empty() {
            return Err(Diagnostic::error(
                Domain::Manifest,
                "desired manifest carries a non-empty changed_managed_files scope (observational scopes are not declarable)"
                    .to_string(),
            ));
        }
    }
    if let Some(s) = manifest.unmanaged_files.as_ref() {
        if !s.elements.is_empty() {
            return Err(Diagnostic::error(
                Domain::Manifest,
                "desired manifest carries a non-empty unmanaged_files scope (observational scopes are not declarable)"
                    .to_string(),
            ));
        }
    }
    Ok(())
}

/// Serialise a manifest to the resolved format. JSON is pretty Machinery JSON;
/// YAML renders the same data model. String-typed fields serialise as quoted YAML
/// scalars (serde_yaml quotes strings that would otherwise be ambiguous; our
/// fields are typed String so they never emit as bare ints/octals).
pub fn serialise_manifest(manifest: &Manifest, fmt: ManifestFormat) -> Result<String, Diagnostic> {
    match fmt {
        ManifestFormat::Json => serde_json::to_string_pretty(manifest)
            .map_err(|e| Diagnostic::error(Domain::Files, format!("serialise: {}", e))),
        ManifestFormat::Yaml => serde_yaml::to_string(manifest)
            .map_err(|e| Diagnostic::error(Domain::Files, format!("serialise: {}", e))),
    }
}

/// Serialise an applied record as canonical JSON (the ledger is always JSON).
pub fn serialise_applied_record(record: &Manifest) -> Result<String, Diagnostic> {
    serde_json::to_string_pretty(record)
        .map_err(|e| Diagnostic::error(Domain::Files, format!("serialise applied record: {}", e)))
}

#[cfg(test)]
mod tests {
    use super::*;

    fn minimal_json() -> Vec<u8> {
        br#"{"meta":{"format_version":1,"generator":"zypper-declarative 0.6.3","created_at":"","desired_sha256":""}}"#.to_vec()
    }

    #[test]
    fn json_parses_and_validates() {
        let m = parse_manifest(&minimal_json(), ManifestFormat::Json).unwrap();
        validate_manifest(&m).unwrap();
        assert_eq!(m.meta.format_version, 1);
    }

    #[test]
    fn format_version_2_rejected() {
        let bytes = br#"{"meta":{"format_version":2}}"#.to_vec();
        let m = parse_manifest(&bytes, ManifestFormat::Json).unwrap();
        assert!(validate_manifest(&m).is_err());
    }

    #[test]
    fn yaml_multidoc_rejected() {
        let y = b"---\nmeta:\n  format_version: 1\n---\nmeta:\n  format_version: 1\n";
        assert!(parse_yaml_safe(y).is_err());
    }

    #[test]
    fn yaml_tag_rejected() {
        let y = b"meta:\n  format_version: 1\n  generator: !!python/object/apply:os.system []\n";
        assert!(parse_yaml_safe(y).is_err());
    }

    #[test]
    fn observational_scope_nonempty_rejected() {
        let bytes = br#"{"meta":{"format_version":1},"unmanaged_files":{"_attributes":{},"_elements":[{"name":"/usr/bin/x","type":"file"}]}}"#.to_vec();
        let m = parse_manifest(&bytes, ManifestFormat::Json).unwrap();
        assert!(validate_manifest(&m).is_err());
    }
}
