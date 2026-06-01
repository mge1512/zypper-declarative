// generated from spec: zypper-declarative.spec.md sha256:18253550a5c3d3f1818f0380811cb5dbc98874828693e49fc9cd5cbc923303dd
//
// CONFIG: all knobs are surfaced via key=value arguments (preset-file layering
// would be merged below these defaults in a full deployment; environment-variable
// control is forbidden by the spec and the cli-tool template, and is not read
// here). A command-line key=value option overrides the corresponding default.

use crate::manifest::{ManifestFormat, TransactionMode};

/// How an unreadable scope source is treated by describe-actual-state.
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

/// The actual-state read scope.
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

/// The resolved configuration for one invocation.
#[derive(Debug, Clone)]
pub struct Config {
    pub transaction_mode: TransactionMode,
    pub manifest_path: Option<String>,
    pub manifest_format: ManifestFormat,
    pub on_unreadable: OnUnreadable,
    pub scope: ScanScope,
    pub repo_lock: Option<String>,
    pub content_store: Option<String>,
    pub keep_list: Option<String>,
    pub signature_verification: bool,
    pub keyring: Option<String>,
    pub activation_policy: String,
    pub applied_root: String,
}

impl Default for Config {
    fn default() -> Self {
        Config {
            transaction_mode: TransactionMode::Auto,
            manifest_path: None,
            manifest_format: ManifestFormat::Json,
            on_unreadable: OnUnreadable::Error,
            scope: ScanScope::Etc,
            repo_lock: None,
            content_store: None,
            keep_list: None,
            signature_verification: true,
            keyring: None,
            activation_policy: "reboot".to_string(),
            applied_root: "/".to_string(),
        }
    }
}
