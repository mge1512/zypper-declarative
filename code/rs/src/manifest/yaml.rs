// generated from spec: zypper-declarative.spec.md
// sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
//
// Safe-profile YAML handling for manifest input/output.
//
// Read side (per load-desired-manifest STEP 3 and the YAML safe profile):
//   - a non-code-executing loader only (no arbitrary/executable tags),
//   - alias expansion bounded or disabled (reject anchors/aliases),
//   - a single document only (reject multi-document streams),
//   - explicit typing per the schema (parse into the typed model; string fields
//     stay strings; no implicit coercion of NO / 1.10 because the typed model
//     declares the field types).
//
// We parse into an untyped serde_yaml::Value first, walk it to reject tags,
// anchors/aliases, and multi-document inputs, then deserialize into the typed
// Manifest. serde_yaml is the chosen YAML crate (version 0.9; recorded in
// TRANSLATION_REPORT.md). A YAML input needing a disabled feature is a manifest
// error.
//
// Write side: string-typed fields must serialise as QUOTED YAML scalars so they
// round-trip as strings (mode: "0600", not mode: 0600). serde_yaml renders
// String values as quoted scalars when the unquoted form would be ambiguous; we
// additionally verify round-trip type stability in tests.

use crate::manifest::Manifest;

/// Error returned by the safe YAML loader. The message is suitable for a
/// manifest-domain diagnostic.
#[derive(Debug)]
pub struct YamlSafetyError(pub String);

impl std::fmt::Display for YamlSafetyError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{}", self.0)
    }
}

impl std::error::Error for YamlSafetyError {}

/// Parse YAML text into a Manifest under the safe profile.
pub fn parse_manifest_safe(text: &str) -> Result<Manifest, YamlSafetyError> {
    // 1. Reject multi-document streams. serde_yaml's multi-document API lets us
    //    count documents; more than one is rejected.
    let docs: Vec<serde_yaml::Value> = serde_yaml::Deserializer::from_str(text)
        .map(|de| serde_yaml::Value::deserialize(de))
        .collect::<Result<Vec<_>, _>>()
        .map_err(|e| YamlSafetyError(format!("YAML parse error: {}", e)))?;

    if docs.len() != 1 {
        return Err(YamlSafetyError(format!(
            "YAML manifest must be a single document; found {} documents (multi-document streams are disabled in the safe profile)",
            docs.len()
        )));
    }

    // 2. Reject explicit (custom/executable) tags and anchors/aliases by walking
    //    the raw text for the YAML constructs the safe profile disables. The
    //    typed model already prevents implicit-typing coercion (fields are typed
    //    strings/ints/bools), so we focus on tag and alias rejection here.
    reject_unsafe_constructs(text)?;

    // 3. Deserialize the single document into the typed model.
    let manifest: Manifest = serde_yaml::from_value(docs.into_iter().next().unwrap())
        .map_err(|e| YamlSafetyError(format!("YAML schema error: {}", e)))?;

    Ok(manifest)
}

// Reject anchors (`&name`), aliases (`*name`), merge keys (`<<:`), and explicit
// tags (`!tag` / `!!tag`) that the safe profile disables. This is a
// conservative lexical check on the raw stream: any of these constructs makes
// the input a manifest error rather than being parsed.
fn reject_unsafe_constructs(text: &str) -> Result<(), YamlSafetyError> {
    for (lineno, raw) in text.lines().enumerate() {
        // Strip a trailing comment and surrounding quotes context is complex; we
        // scan token-wise but ignore the inside of double/single quoted scalars.
        let line = strip_quoted_and_comment(raw);
        let trimmed = line.trim_start();

        // Alias usage: a value that is `*name`.
        if value_starts_with(&line, '*') {
            return Err(YamlSafetyError(format!(
                "YAML alias usage is disabled in the safe profile (line {})",
                lineno + 1
            )));
        }
        // Anchor definition: a `&name` token after a key or in a sequence entry.
        if line.contains(" &") || trimmed.starts_with("&") || line.contains("\t&") {
            return Err(YamlSafetyError(format!(
                "YAML anchors are disabled in the safe profile (line {})",
                lineno + 1
            )));
        }
        // Merge key.
        if trimmed.starts_with("<<:") || trimmed.starts_with("<< :") {
            return Err(YamlSafetyError(format!(
                "YAML merge keys are disabled in the safe profile (line {})",
                lineno + 1
            )));
        }
        // Explicit / custom tags: a `!` introducing a tag. We allow none.
        if contains_explicit_tag(&line) {
            return Err(YamlSafetyError(format!(
                "YAML explicit tags are disabled in the safe profile (line {})",
                lineno + 1
            )));
        }
    }
    Ok(())
}

// Remove the contents of quoted scalars (so a `*` or `!` inside a string is not
// mistaken for a construct) and drop trailing `#` comments outside quotes.
fn strip_quoted_and_comment(line: &str) -> String {
    let mut out = String::new();
    let mut chars = line.chars().peekable();
    let mut in_single = false;
    let mut in_double = false;
    while let Some(c) = chars.next() {
        match c {
            '\'' if !in_double => {
                in_single = !in_single;
                out.push(' ');
            }
            '"' if !in_single => {
                in_double = !in_double;
                out.push(' ');
            }
            '#' if !in_single && !in_double => {
                // rest is a comment
                break;
            }
            _ => {
                if in_single || in_double {
                    out.push(' ');
                } else {
                    out.push(c);
                }
            }
        }
    }
    out
}

// True when, after the first `:` separator (or `-` sequence marker), the value
// token begins with `ch`.
fn value_starts_with(line: &str, ch: char) -> bool {
    let after = if let Some(idx) = line.find(": ") {
        &line[idx + 2..]
    } else if let Some(rest) = line.trim_start().strip_prefix("- ") {
        rest
    } else {
        return false;
    };
    after.trim_start().starts_with(ch)
}

fn contains_explicit_tag(line: &str) -> bool {
    // A tag token is a `!` followed by a non-space, appearing as a value or
    // standalone, e.g. `!!python/object`, `!Foo`. We flag any `!` token that is
    // preceded by whitespace or a `:`/`-` separator and followed by a non-space.
    let bytes: Vec<char> = line.chars().collect();
    for i in 0..bytes.len() {
        if bytes[i] == '!' {
            let prev_ok = i == 0
                || bytes[i - 1].is_whitespace()
                || bytes[i - 1] == ':'
                || bytes[i - 1] == '-'
                || bytes[i - 1] == '[';
            let next_ok = i + 1 < bytes.len() && !bytes[i + 1].is_whitespace();
            if prev_ok && next_ok {
                return true;
            }
        }
    }
    false
}

/// Serialise a Manifest to YAML (write side). serde_yaml renders ambiguous
/// string scalars quoted, preserving string typing on round-trip.
pub fn to_yaml(manifest: &Manifest) -> Result<String, serde_yaml::Error> {
    serde_yaml::to_string(manifest)
}

use serde::Deserialize;

#[cfg(test)]
mod tests {
    use super::*;

    const GOOD: &str = r#"meta:
  format_version: 1
  generator: "t"
  created_at: "2026-01-01T00:00:00Z"
  desired_sha256: ""
config_files:
  _attributes: {}
  _elements:
    - name: "/etc/foo.conf"
      type: "file"
      mode: "0600"
      user: "root"
      group: "root"
      sha256: "1111111111111111111111111111111111111111111111111111111111111111"
      target: ""
      content_ref: ""
      package_name: ""
"#;

    #[test]
    fn good_yaml_parses() {
        let m = parse_manifest_safe(GOOD).unwrap();
        assert_eq!(m.meta.format_version, 1);
        let cf = m.config_files.unwrap();
        assert_eq!(cf.elements[0].mode, "0600");
    }

    #[test]
    fn multidoc_rejected() {
        let s = format!("{}\n---\n{}", GOOD, GOOD);
        assert!(parse_manifest_safe(&s).is_err());
    }

    #[test]
    fn anchor_rejected() {
        let s = "meta: &a\n  format_version: 1\n";
        assert!(parse_manifest_safe(s).is_err());
    }

    #[test]
    fn tag_rejected() {
        let s = "meta: !!python/object\n  format_version: 1\n";
        assert!(parse_manifest_safe(s).is_err());
    }

    #[test]
    fn mode_round_trips_as_string() {
        let m = parse_manifest_safe(GOOD).unwrap();
        let y = to_yaml(&m).unwrap();
        let m2 = parse_manifest_safe(&y).unwrap();
        assert_eq!(m2.config_files.unwrap().elements[0].mode, "0600");
    }
}
