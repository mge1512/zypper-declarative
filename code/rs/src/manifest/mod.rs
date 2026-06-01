// generated from spec: zypper-declarative.spec.md sha256:18253550a5c3d3f1818f0380811cb5dbc98874828693e49fc9cd5cbc923303dd
//
// The shared data model: the declarable subset of the SUSE Machinery system
// description (packages, repositories, services, config_files), the ScopeWrapper
// idiom, and underscore_style field names. The desired manifest, the applied
// record, describe output, and any supplied state dump all share this model.
//
// Absent vs empty scopes are semantic: a declarable scope modelled as
// `Option<Scope>` is `None` when absent (unmanaged) and `Some` with empty
// `_elements` when present-but-empty (reconcile to empty). The two are never
// collapsed.

pub mod format;
pub mod hash;
pub mod serialize;

use serde::{Deserialize, Serialize};
use std::collections::BTreeMap;

/// The shared scope idiom (Machinery / sitar convention). `_attributes` is
/// ALWAYS a JSON object — empty `{}` when the scope has no attributes, never
/// `null` (Machinery consistency invariant).
#[derive(Serialize, Deserialize, Debug, Clone, PartialEq)]
pub struct ScopeWrapper<T> {
    #[serde(rename = "_attributes", default)]
    pub attributes: BTreeMap<String, serde_json::Value>,
    #[serde(rename = "_elements", default = "Vec::new")]
    pub elements: Vec<T>,
}

impl<T> Default for ScopeWrapper<T> {
    fn default() -> Self {
        ScopeWrapper {
            attributes: BTreeMap::new(),
            elements: Vec::new(),
        }
    }
}

impl<T> ScopeWrapper<T> {
    pub fn new() -> Self {
        Self::default()
    }

    pub fn with_attribute(mut self, key: &str, value: serde_json::Value) -> Self {
        self.attributes.insert(key.to_string(), value);
        self
    }
}

/// Manifest meta block (Machinery JSON format).
#[derive(Serialize, Deserialize, Debug, Clone, PartialEq)]
pub struct ManifestMeta {
    /// Always 1 (Machinery JSON format_version).
    pub format_version: i64,
    /// Program name and version, e.g. "zypper-declarative 0.6.4".
    pub generator: String,
    /// RFC3339, informational only, not compared and not hashed.
    pub created_at: String,
    /// Canonical-model hash of the desired manifest; set in the applied record,
    /// "" elsewhere.
    pub desired_sha256: String,
}

impl Default for ManifestMeta {
    fn default() -> Self {
        ManifestMeta {
            format_version: 1,
            generator: crate::meta::generator(),
            created_at: String::new(),
            desired_sha256: String::new(),
        }
    }
}

/// Machinery PackageRecord (identity subset). A desired package may carry name
/// only; in the applied record and describe output every record is fully
/// resolved (version, release, arch).
#[derive(Serialize, Deserialize, Debug, Clone, PartialEq, Eq, Default)]
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

/// Machinery zypp repository record.
#[derive(Serialize, Deserialize, Debug, Clone, PartialEq, Eq, Default)]
pub struct RepositoryRecord {
    pub alias: String,
    #[serde(default)]
    pub name: String,
    pub url: String,
    #[serde(default, rename = "type")]
    pub repo_type: String,
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

/// Machinery service record, declarable states only.
#[derive(Serialize, Deserialize, Debug, Clone, PartialEq, Eq, Default)]
pub struct ServiceRecord {
    pub name: String,
    /// one_of("enabled" | "disabled" | "masked").
    pub state: String,
}

pub type ServicesScope = ScopeWrapper<ServiceRecord>;

/// A declarable SUPERSET of the Machinery changed-config-files record (with a
/// content digest for files, a verbatim target for symlinks, and a content_ref
/// for supplying content at apply time).
#[derive(Serialize, Deserialize, Debug, Clone, PartialEq, Eq, Default)]
pub struct ManagedFileRecord {
    /// File path under /etc (Machinery: name).
    pub name: String,
    /// one_of("file" | "link" | "dir").
    #[serde(rename = "type")]
    pub file_type: String,
    pub mode: String,
    pub user: String,
    pub group: String,
    /// For type=file: a Sha256 content digest; "" otherwise.
    #[serde(default)]
    pub sha256: String,
    /// For type=link: the verbatim symlink target; "" otherwise.
    #[serde(default)]
    pub target: String,
    /// For a DESIRED type=file: how content is supplied at apply time; ""
    /// in describe output and for non-file types.
    #[serde(default)]
    pub content_ref: String,
    /// Owning package (bare name); "" if unpackaged.
    #[serde(default)]
    pub package_name: String,
}

pub type ConfigFilesScope = ScopeWrapper<ManagedFileRecord>;

/// Machinery changed_managed_files record (a packaged file outside /etc whose
/// state differs from the package baseline). Observational, scope=full only.
#[derive(Serialize, Deserialize, Debug, Clone, PartialEq, Eq, Default)]
pub struct ManagedBaselineRecord {
    pub name: String,
    #[serde(rename = "type")]
    pub file_type: String,
    pub mode: String,
    pub user: String,
    pub group: String,
    #[serde(default)]
    pub sha256: String,
    #[serde(default)]
    pub target: String,
    pub package_name: String,
    #[serde(default)]
    pub changes: Vec<String>,
}

pub type ChangedManagedFilesScope = ScopeWrapper<ManagedBaselineRecord>;

/// Machinery unmanaged_files record (a file no package owns, outside /etc).
/// Observational, scope=full only.
#[derive(Serialize, Deserialize, Debug, Clone, PartialEq, Eq, Default)]
pub struct UnmanagedFileRecord {
    pub name: String,
    #[serde(rename = "type")]
    pub file_type: String,
    pub mode: String,
    pub user: String,
    pub group: String,
    #[serde(default)]
    pub sha256: String,
    #[serde(default)]
    pub target: String,
}

pub type UnmanagedFilesScope = ScopeWrapper<UnmanagedFileRecord>;

/// The Manifest: the shared shape produced by describe (actual state) and
/// consumed by apply/diff/verify (desired state, declarable scopes only).
///
/// A declarable scope ABSENT (`None`) means the converger makes no assertion;
/// a scope PRESENT with empty `_elements` asserts the scope should be exactly
/// empty. The observational scopes carry no such meaning and never appear in a
/// desired manifest or an applied record.
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
    /// Observational, never declarable. Present only under scope=full.
    #[serde(skip_serializing_if = "Option::is_none", default)]
    pub changed_managed_files: Option<ChangedManagedFilesScope>,
    #[serde(skip_serializing_if = "Option::is_none", default)]
    pub unmanaged_files: Option<UnmanagedFilesScope>,
}

impl Manifest {
    /// An all-empty manifest with a well-formed meta block. Used as the
    /// "empty applied record" (every scope absent) for a first-ever apply.
    pub fn empty() -> Self {
        Manifest {
            meta: ManifestMeta::default(),
            ..Default::default()
        }
    }
}

/// The applied record is a Manifest with the packages scope fully resolved (the
/// lock) and the source manifest's hash recorded in meta. We model it as a
/// `Manifest` newtype-equivalent for clarity at call sites.
pub type AppliedRecord = Manifest;

/// The transaction mode (deliberately deferred binding).
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

/// The transaction context the convergence domains operate within.
#[derive(Debug, Clone)]
pub struct TransactionContext {
    pub mode: TransactionMode,
    pub root: String,
    pub opened_here: bool,
}

/// The intent diff: desired_new versus applied_old, computed scope by scope.
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

    pub fn item_count(&self) -> usize {
        self.files_modified.len()
            + self.files_extra.len()
            + self.units_divergent.len()
            + self.packages_divergent.len()
            + self.managed_files_modified.len()
            + self.unmanaged_files_present.len()
    }
}

/// Serialisation of the manifest data model.
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
