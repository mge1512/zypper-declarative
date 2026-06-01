// generated from spec: zypper-declarative.spec.md sha256:87b9f2f3bf92afd6fe037412d56a23959290e5816d722a3def9505d07aa5acd7
//
// BEHAVIOR/INTERNAL: resolve-format. The single authority for choosing a manifest
// serialisation. Every read that parses a manifest and every write that
// serialises one resolves its format here, so input and output behave
// symmetrically and the rule cannot drift between call sites.

use crate::types::ManifestFormat;

/// resolve-format(explicit, path) with the manifest-format CONFIG default.
///
/// STEPS:
/// 1. If `explicit` is given, return it (an explicit format= always wins).
/// 2. Else if `path` is given and its extension is recognised, return json for
///    .json and yaml for .yaml/.yml.
/// 3. Else return the manifest-format CONFIG default.
pub fn resolve_format(
    explicit: Option<ManifestFormat>,
    path: Option<&str>,
    default: ManifestFormat,
) -> ManifestFormat {
    if let Some(fmt) = explicit {
        return fmt;
    }
    if let Some(p) = path {
        if let Some(fmt) = format_from_extension(p) {
            return fmt;
        }
    }
    default
}

/// Map a recognised file extension to a format; None if unrecognised or absent.
fn format_from_extension(path: &str) -> Option<ManifestFormat> {
    let lower = path.to_ascii_lowercase();
    if lower.ends_with(".json") {
        Some(ManifestFormat::Json)
    } else if lower.ends_with(".yaml") || lower.ends_with(".yml") {
        Some(ManifestFormat::Yaml)
    } else {
        None
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn explicit_wins_over_extension_and_default() {
        let r = resolve_format(
            Some(ManifestFormat::Json),
            Some("x.yaml"),
            ManifestFormat::Yaml,
        );
        assert_eq!(r, ManifestFormat::Json);
    }

    #[test]
    fn extension_consulted_when_no_explicit() {
        assert_eq!(
            resolve_format(None, Some("/tmp/state.yaml"), ManifestFormat::Json),
            ManifestFormat::Yaml
        );
        assert_eq!(
            resolve_format(None, Some("/tmp/state.json"), ManifestFormat::Yaml),
            ManifestFormat::Json
        );
        assert_eq!(
            resolve_format(None, Some("/tmp/state.yml"), ManifestFormat::Json),
            ManifestFormat::Yaml
        );
    }

    #[test]
    fn default_used_for_no_path_or_unrecognised() {
        assert_eq!(
            resolve_format(None, None, ManifestFormat::Json),
            ManifestFormat::Json
        );
        assert_eq!(
            resolve_format(None, Some("/tmp/state.txt"), ManifestFormat::Yaml),
            ManifestFormat::Yaml
        );
    }
}
