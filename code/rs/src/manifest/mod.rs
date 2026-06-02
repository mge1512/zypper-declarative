// generated from spec: zypper-declarative.spec.md
// sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
//
// Manifest data model: the declarable subset of the SUSE Machinery system
// description (packages, repositories, services, config_files), plus the
// observational scopes that describe scope=full produces. The ScopeWrapper
// idiom (_attributes object + _elements array) and underscore_style field
// names match Machinery. JSON is canonical; YAML is an opt-in serialisation
// of the identical model.

pub mod format;
pub mod hash;
pub mod load;
pub mod yaml;

use serde::{Deserialize, Serialize};
use serde_json::{Map, Value};

/// A scope-level attributes map. ALWAYS serialised as a JSON object (empty `{}`
/// when the scope has no attributes), NEVER null (Machinery consistency).
pub type Attributes = Map<String, Value>;

/// ScopeWrapper<T> := { _attributes: object, _elements: []T }
#[derive(Serialize, Deserialize, Debug, Clone, PartialEq)]
pub struct ScopeWrapper<T> {
    #[serde(rename = "_attributes", default)]
    pub attributes: Attributes,
    #[serde(rename = "_elements", default = "Vec::new")]
    pub elements: Vec<T>,
}

impl<T> Default for ScopeWrapper<T> {
    fn default() -> Self {
        ScopeWrapper {
            attributes: Map::new(),
            elements: Vec::new(),
        }
    }
}

impl<T> ScopeWrapper<T> {
    pub fn with_attr(key: &str, value: &str) -> Self {
        let mut attrs = Map::new();
        attrs.insert(key.to_string(), Value::String(value.to_string()));
        ScopeWrapper {
            attributes: attrs,
            elements: Vec::new(),
        }
    }
}

#[derive(Serialize, Deserialize, Debug, Clone, PartialEq, Default)]
pub struct ManifestMeta {
    pub format_version: i64,
    #[serde(default)]
    pub generator: String,
    #[serde(default)]
    pub created_at: String,
    #[serde(default)]
    pub desired_sha256: String,
}

#[derive(Serialize, Deserialize, Debug, Clone, PartialEq, Default)]
pub struct PackageRecord {
    pub name: String,
    #[serde(default)]
    pub version: String,
    #[serde(default)]
    pub release: String,
    #[serde(default)]
    pub arch: String,
}

pub type PackagesScope = ScopeWrapper<PackageRecord>;

#[derive(Serialize, Deserialize, Debug, Clone, PartialEq, Default)]
pub struct RepositoryRecord {
    pub alias: String,
    #[serde(default)]
    pub name: String,
    pub url: String,
    #[serde(default)]
    pub r#type: String,
    #[serde(default)]
    pub enabled: bool,
    #[serde(default)]
    pub gpgcheck: bool,
    #[serde(default)]
    pub autorefresh: bool,
    #[serde(default)]
    pub priority: i64,
}

pub type RepositoriesScope = ScopeWrapper<RepositoryRecord>;

#[derive(Serialize, Deserialize, Debug, Clone, PartialEq, Default)]
pub struct ServiceRecord {
    pub name: String,
    pub state: String, // enabled | disabled | masked
}

pub type ServicesScope = ScopeWrapper<ServiceRecord>;

/// ManagedFileRecord — a declarable /etc file/link/dir record.
/// Optional fields (status/changes) carry the rpm verdict for changed records.
#[derive(Serialize, Deserialize, Debug, Clone, PartialEq, Default)]
pub struct ManagedFileRecord {
    pub name: String,
    pub r#type: String, // file | link | dir
    #[serde(default)]
    pub mode: String,
    #[serde(default)]
    pub user: String,
    #[serde(default)]
    pub group: String,
    #[serde(default)]
    pub sha256: String,
    #[serde(default)]
    pub target: String,
    #[serde(default)]
    pub content_ref: String,
    #[serde(default)]
    pub package_name: String,
    // Verdict fields for changed-from-package records (omitted when empty).
    #[serde(default, skip_serializing_if = "String::is_empty")]
    pub status: String,
    #[serde(default, skip_serializing_if = "Vec::is_empty")]
    pub changes: Vec<String>,
}

pub type ConfigFilesScope = ScopeWrapper<ManagedFileRecord>;

#[derive(Serialize, Deserialize, Debug, Clone, PartialEq, Default)]
pub struct ManagedBaselineRecord {
    pub name: String,
    pub r#type: String,
    #[serde(default)]
    pub mode: String,
    #[serde(default)]
    pub user: String,
    #[serde(default)]
    pub group: String,
    #[serde(default)]
    pub sha256: String,
    #[serde(default)]
    pub target: String,
    #[serde(default)]
    pub package_name: String,
    #[serde(default)]
    pub changes: Vec<String>,
}

pub type ChangedManagedFilesScope = ScopeWrapper<ManagedBaselineRecord>;

#[derive(Serialize, Deserialize, Debug, Clone, PartialEq, Default)]
pub struct UnmanagedFileRecord {
    pub name: String,
    pub r#type: String,
    #[serde(default)]
    pub mode: String,
    #[serde(default)]
    pub user: String,
    #[serde(default)]
    pub group: String,
    #[serde(default)]
    pub sha256: String,
    #[serde(default)]
    pub target: String,
}

pub type UnmanagedFilesScope = ScopeWrapper<UnmanagedFileRecord>;

/// Manifest := { meta, packages?, repositories?, services?, config_files?,
///               changed_managed_files?, unmanaged_files? }
/// A declarable scope ABSENT means unmanaged (None). A scope PRESENT with empty
/// _elements asserts the scope should be exactly empty. The two are NOT
/// collapsed: `Option<Scope>`.
#[derive(Serialize, Deserialize, Debug, Clone, PartialEq, Default)]
pub struct Manifest {
    pub meta: ManifestMeta,
    #[serde(skip_serializing_if = "Option::is_none", default)]
    pub packages: Option<PackagesScope>,
    #[serde(skip_serializing_if = "Option::is_none", default)]
    pub repositories: Option<RepositoriesScope>,
    #[serde(skip_serializing_if = "Option::is_none", default)]
    pub services: Option<ServicesScope>,
    #[serde(skip_serializing_if = "Option::is_none", default)]
    pub config_files: Option<ConfigFilesScope>,
    #[serde(skip_serializing_if = "Option::is_none", default)]
    pub changed_managed_files: Option<ChangedManagedFilesScope>,
    #[serde(skip_serializing_if = "Option::is_none", default)]
    pub unmanaged_files: Option<UnmanagedFilesScope>,
}

impl Manifest {
    /// A fresh empty manifest with format_version 1 and the standard generator.
    pub fn new_actual(created_at: String) -> Self {
        Manifest {
            meta: ManifestMeta {
                format_version: 1,
                generator: crate::meta::generator(),
                created_at,
                desired_sha256: String::new(),
            },
            ..Default::default()
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn empty_scope_serialises_attributes_as_object() {
        let s: PackagesScope = ScopeWrapper::default();
        let v = serde_json::to_value(&s).unwrap();
        assert!(v.get("_attributes").unwrap().is_object());
        assert!(v.get("_elements").unwrap().as_array().unwrap().is_empty());
    }

    #[test]
    fn absent_scope_omitted_from_json() {
        let m = Manifest::new_actual("2026-01-01T00:00:00Z".into());
        let v = serde_json::to_value(&m).unwrap();
        assert!(v.get("config_files").is_none());
        assert!(v.get("packages").is_none());
    }

    #[test]
    fn present_empty_scope_preserved() {
        let mut m = Manifest::new_actual("2026-01-01T00:00:00Z".into());
        m.config_files = Some(ScopeWrapper::default());
        let v = serde_json::to_value(&m).unwrap();
        let cf = v.get("config_files").unwrap();
        assert!(cf.get("_elements").unwrap().as_array().unwrap().is_empty());
    }
}
