// generated from spec: zypper-declarative.spec.md sha256:18253550a5c3d3f1818f0380811cb5dbc98874828693e49fc9cd5cbc923303dd
//
// BEHAVIOR/INTERNAL: resolve-format — the single authority for choosing a
// manifest serialisation. Every read that parses a manifest and every write
// that serialises one resolves its format here, so input and output behave
// symmetrically and the rule cannot drift between call sites.

use super::ManifestFormat;

/// resolve-format(explicit, path):
/// 1. If `explicit` is given, return it (an explicit format= always wins).
/// 2. Else if `path` is given and its extension is recognised, return json for
///    `.json` and yaml for `.yaml`/`.yml`.
/// 3. Else (no explicit, no path or unrecognised extension) return the
///    `manifest-format` CONFIG default.
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

/// Map a recognised file extension to a format, or None.
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
    fn explicit_wins_over_extension() {
        let f = resolve_format(Some(ManifestFormat::Json), Some("x.yaml"), ManifestFormat::Yaml);
        assert_eq!(f, ManifestFormat::Json);
    }

    #[test]
    fn extension_decides_when_no_explicit() {
        assert_eq!(
            resolve_format(None, Some("x.yaml"), ManifestFormat::Json),
            ManifestFormat::Yaml
        );
        assert_eq!(
            resolve_format(None, Some("x.yml"), ManifestFormat::Json),
            ManifestFormat::Yaml
        );
        assert_eq!(
            resolve_format(None, Some("x.json"), ManifestFormat::Yaml),
            ManifestFormat::Json
        );
    }

    #[test]
    fn default_when_no_path_or_unrecognised() {
        assert_eq!(
            resolve_format(None, None, ManifestFormat::Json),
            ManifestFormat::Json
        );
        assert_eq!(
            resolve_format(None, Some("x.toml"), ManifestFormat::Yaml),
            ManifestFormat::Yaml
        );
    }
}
