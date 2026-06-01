// generated from spec: zypper-declarative.spec.md sha256:87b9f2f3bf92afd6fe037412d56a23959290e5816d722a3def9505d07aa5acd7
//
// The shared data model: the declarable subset of the SUSE Machinery system
// description, using the ScopeWrapper idiom and underscore_style field names.
// JSON is the canonical serialisation (format_version 1); YAML is an opt-in
// serialisation of the identical model. A declarable scope is modelled as
// Option<Scope>: None = absent (unmanaged), Some with empty _elements =
// present-empty (reconcile to empty). The two are never collapsed.

use serde::{Deserialize, Serialize};
use std::collections::BTreeMap;

/// Scope-level attributes are ALWAYS a JSON object (empty {} when none), never null.
/// We use a BTreeMap so serialisation is deterministic (sorted keys) for hashing
/// and cross-implementation consistency.
pub type Attributes = BTreeMap<String, serde_json::Value>;

/// ScopeWrapper<T>: { _attributes: object, _elements: []T }.
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
            attributes: Attributes::new(),
            elements: Vec::new(),
        }
    }
}

impl<T> ScopeWrapper<T> {
    pub fn with_attr(key: &str, value: &str) -> Self {
        let mut attrs = Attributes::new();
        attrs.insert(
            key.to_string(),
            serde_json::Value::String(value.to_string()),
        );
        ScopeWrapper {
            attributes: attrs,
            elements: Vec::new(),
        }
    }

    pub fn is_empty(&self) -> bool {
        self.elements.is_empty()
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
    pub state: String, // one_of("enabled" | "disabled" | "masked")
}

pub type ServicesScope = ScopeWrapper<ServiceRecord>;

#[derive(Serialize, Deserialize, Debug, Clone, PartialEq, Default)]
pub struct ManagedFileRecord {
    pub name: String,   // AbsolutePath under /etc/
    pub r#type: String, // one_of("file" | "link" | "dir")
    #[serde(default)]
    pub mode: String,
    #[serde(default)]
    pub user: String,
    #[serde(default)]
    pub group: String,
    #[serde(default)]
    pub sha256: String, // for type=file; "" otherwise
    #[serde(default)]
    pub target: String, // verbatim symlink target for type=link; "" otherwise
    #[serde(default)]
    pub content_ref: String, // desired type=file only; "" in describe output
    #[serde(default)]
    pub package_name: String, // owning package; "" if unpackaged
}

pub type ConfigFilesScope = ScopeWrapper<ManagedFileRecord>;

#[derive(Serialize, Deserialize, Debug, Clone, PartialEq, Default)]
pub struct ManagedBaselineRecord {
    pub name: String, // AbsolutePath OUTSIDE /etc
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
    pub package_name: String, // always set here
    #[serde(default)]
    pub changes: Vec<String>,
}

pub type ChangedManagedFilesScope = ScopeWrapper<ManagedBaselineRecord>;

#[derive(Serialize, Deserialize, Debug, Clone, PartialEq, Default)]
pub struct UnmanagedFileRecord {
    pub name: String, // AbsolutePath OUTSIDE /etc that no package owns
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

/// The Manifest. Declarable scopes are Option to preserve absent-vs-empty.
/// Observational scopes are Option and never appear in a desired manifest or an
/// applied record.
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

/// An applied record is a Manifest with packages fully resolved and
/// meta.desired_sha256 set. The type alias documents intent.
pub type AppliedRecord = Manifest;

impl Manifest {
    /// An empty manifest (all scopes absent) with format_version = 1.
    pub fn empty() -> Self {
        Manifest {
            meta: ManifestMeta {
                format_version: 1,
                generator: crate::meta::generator(),
                created_at: String::new(),
                desired_sha256: String::new(),
            },
            packages: None,
            repositories: None,
            services: None,
            config_files: None,
            changed_managed_files: None,
            unmanaged_files: None,
        }
    }

    /// All declarable scope elements as empty when absent. Convenience accessors
    /// that return an empty slice when the scope is absent.
    pub fn packages_elems(&self) -> &[PackageRecord] {
        self.packages
            .as_ref()
            .map(|s| s.elements.as_slice())
            .unwrap_or(&[])
    }
    pub fn repositories_elems(&self) -> &[RepositoryRecord] {
        self.repositories
            .as_ref()
            .map(|s| s.elements.as_slice())
            .unwrap_or(&[])
    }
    pub fn services_elems(&self) -> &[ServiceRecord] {
        self.services
            .as_ref()
            .map(|s| s.elements.as_slice())
            .unwrap_or(&[])
    }
    pub fn config_files_elems(&self) -> &[ManagedFileRecord] {
        self.config_files
            .as_ref()
            .map(|s| s.elements.as_slice())
            .unwrap_or(&[])
    }
}

/// The intent diff: desired_new versus applied_old.
#[derive(Debug, Clone, Default, PartialEq)]
pub struct Diff {
    pub packages_install: Vec<PackageRecord>,
    pub packages_remove: Vec<PackageRecord>,
    pub repos_set: Vec<RepositoryRecord>,
    pub files_write: Vec<ManagedFileRecord>,
    pub files_delete: Vec<String>,
    pub units_change: Vec<ServiceRecord>,
}

impl Diff {
    pub fn is_empty(&self) -> bool {
        self.packages_install.is_empty()
            && self.packages_remove.is_empty()
            && self.repos_set.is_empty()
            && self.files_write.is_empty()
            && self.files_delete.is_empty()
            && self.units_change.is_empty()
    }
}

/// The drift diff: actual versus declared.
#[derive(Debug, Clone, Default, PartialEq)]
pub struct DriftReport {
    pub files_modified: Vec<String>,
    pub files_extra: Vec<String>,
    pub units_divergent: Vec<ServiceRecord>,
    pub packages_divergent: Vec<PackageRecord>,
    pub managed_files_modified: Vec<String>,
    pub unmanaged_files_present: Vec<String>,
}

impl DriftReport {
    pub fn is_empty(&self) -> bool {
        self.files_modified.is_empty()
            && self.files_extra.is_empty()
            && self.units_divergent.is_empty()
            && self.packages_divergent.is_empty()
            && self.managed_files_modified.is_empty()
            && self.unmanaged_files_present.is_empty()
    }
}

/// Manifest serialisation format.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ManifestFormat {
    Json,
    Yaml,
}

impl ManifestFormat {
    pub fn parse(s: &str) -> Option<ManifestFormat> {
        match s {
            "json" => Some(ManifestFormat::Json),
            "yaml" => Some(ManifestFormat::Yaml),
            _ => None,
        }
    }
}

/// Transaction mode.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum TransactionMode {
    Auto,
    External,
    Internal,
}

impl TransactionMode {
    pub fn parse(s: &str) -> Option<TransactionMode> {
        match s {
            "auto" => Some(TransactionMode::Auto),
            "external" => Some(TransactionMode::External),
            "internal" => Some(TransactionMode::Internal),
            _ => None,
        }
    }
}

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub struct TransactionContext {
    pub mode: TransactionMode,
    pub opened_here: bool,
}

/// Scan scope: etc (default) or full.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ScanScope {
    Etc,
    Full,
}

impl ScanScope {
    pub fn parse(s: &str) -> Option<ScanScope> {
        match s {
            "etc" => Some(ScanScope::Etc),
            "full" => Some(ScanScope::Full),
            _ => None,
        }
    }
}

/// How an unreadable scope source is treated.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum OnUnreadable {
    Error,
    Warn,
}

impl OnUnreadable {
    pub fn parse(s: &str) -> Option<OnUnreadable> {
        match s {
            "error" => Some(OnUnreadable::Error),
            "warn" => Some(OnUnreadable::Warn),
            _ => None,
        }
    }
}
