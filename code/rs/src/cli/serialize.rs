// generated from spec: zypper-declarative.spec.md
// sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
//
// Serialisation of a Manifest in the resolved format. JSON is canonical
// (Machinery format_version 1, ScopeWrapper idiom). YAML is the same data
// model rendered through the safe-profile writer (quoted string scalars).

use crate::error::{Diagnostic, Domain};
use crate::manifest::format::ManifestFormat;
use crate::manifest::{yaml, Manifest};

pub fn serialise(manifest: &Manifest, format: ManifestFormat) -> Result<String, Diagnostic> {
    match format {
        ManifestFormat::Json => serde_json::to_string_pretty(manifest).map_err(|e| {
            Diagnostic::error(Domain::Files, format!("JSON serialisation failed: {}", e))
        }),
        ManifestFormat::Yaml => yaml::to_yaml(manifest).map_err(|e| {
            Diagnostic::error(Domain::Files, format!("YAML serialisation failed: {}", e))
        }),
    }
}
