// generated from spec: zypper-declarative.spec.md
// sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
//
// BEHAVIOR/INTERNAL: resolve-format — the single authority for choosing a
// manifest serialisation. Every read that parses a manifest and every write
// that serialises one resolves its format here, so input and output behave
// symmetrically and the rule cannot drift between call sites.

use std::path::Path;

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ManifestFormat {
    Json,
    Yaml,
}

impl ManifestFormat {
    /// Parse an explicit `format=` value. Returns None for an unknown value;
    /// the caller maps that to an invocation error.
    pub fn parse(s: &str) -> Option<ManifestFormat> {
        match s {
            "json" => Some(ManifestFormat::Json),
            "yaml" => Some(ManifestFormat::Yaml),
            _ => None,
        }
    }
}

/// resolve-format(explicit, path) -> ManifestFormat
///
/// STEPS:
/// 1. If `explicit` is given, return it (an explicit format= always wins).
/// 2. Else if `path` is given and its extension is recognised, return json for
///    `.json` and yaml for `.yaml`/`.yml`.
/// 3. Else return the `manifest-format` CONFIG default.
pub fn resolve_format(
    explicit: Option<ManifestFormat>,
    path: Option<&str>,
    default: ManifestFormat,
) -> ManifestFormat {
    if let Some(f) = explicit {
        return f;
    }
    if let Some(p) = path {
        if let Some(ext) = Path::new(p).extension().and_then(|e| e.to_str()) {
            match ext.to_ascii_lowercase().as_str() {
                "json" => return ManifestFormat::Json,
                "yaml" | "yml" => return ManifestFormat::Yaml,
                _ => {}
            }
        }
    }
    default
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn explicit_wins_over_extension() {
        let f = resolve_format(
            Some(ManifestFormat::Json),
            Some("/tmp/x.yaml"),
            ManifestFormat::Yaml,
        );
        assert_eq!(f, ManifestFormat::Json);
    }

    #[test]
    fn extension_decides_when_no_explicit() {
        assert_eq!(
            resolve_format(None, Some("/tmp/x.yaml"), ManifestFormat::Json),
            ManifestFormat::Yaml
        );
        assert_eq!(
            resolve_format(None, Some("/tmp/x.yml"), ManifestFormat::Json),
            ManifestFormat::Yaml
        );
        assert_eq!(
            resolve_format(None, Some("/tmp/x.json"), ManifestFormat::Yaml),
            ManifestFormat::Json
        );
    }

    #[test]
    fn default_when_no_path_or_unknown_ext() {
        assert_eq!(
            resolve_format(None, None, ManifestFormat::Json),
            ManifestFormat::Json
        );
        assert_eq!(
            resolve_format(None, Some("/tmp/x.toml"), ManifestFormat::Json),
            ManifestFormat::Json
        );
    }
}
