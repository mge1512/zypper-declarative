// generated from spec: zypper-declarative.spec.md sha256:87b9f2f3bf92afd6fe037412d56a23959290e5816d722a3def9505d07aa5acd7
//
// CONFIG and the parsed invocation. All knobs are surfaced via key=value options
// (or preset files; presets are not loaded in this build's default path but the
// option keys mirror the CONFIG names). Control via environment variables is
// forbidden. Options are key=value; bare words are verbs. Options may appear in
// any position (including after the verb).

use crate::error::{Diagnostic, Domain};
use crate::types::{ManifestFormat, OnUnreadable, ScanScope, TransactionMode};

/// The recognised verbs (bare words backed by a BEHAVIOR) plus the global
/// commands handled by the dispatcher.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Verb {
    Apply,
    Diff,
    Verify,
    Status,
    Describe,
    Version,
    Help,
}

impl Verb {
    pub fn parse(s: &str) -> Option<Verb> {
        match s {
            "apply" => Some(Verb::Apply),
            "diff" => Some(Verb::Diff),
            "verify" => Some(Verb::Verify),
            "status" => Some(Verb::Status),
            "describe" => Some(Verb::Describe),
            "version" => Some(Verb::Version),
            "help" => Some(Verb::Help),
            _ => None,
        }
    }
}

/// The parsed invocation: the verb (or None for bare invocation) and the resolved
/// options. Unknown options/values produce an invocation error during parsing.
#[derive(Debug, Clone)]
pub struct Invocation {
    pub verb: Option<Verb>,
    pub opts: Options,
}

/// All option values, with CONFIG defaults applied.
#[derive(Debug, Clone)]
pub struct Options {
    pub mode: TransactionMode,
    pub manifest_path: Option<String>,
    pub manifest_format: ManifestFormat, // the resolve-format default
    pub format: Option<ManifestFormat>,  // explicit format= for this invocation
    pub state_path: Option<String>,
    pub root: String,
    pub out: Option<String>,
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

impl Default for Options {
    fn default() -> Self {
        Options {
            mode: TransactionMode::Auto,
            manifest_path: None,
            manifest_format: ManifestFormat::Json,
            format: None,
            state_path: None,
            root: "/".to_string(),
            out: None,
            on_unreadable: OnUnreadable::Error,
            scope: ScanScope::Etc,
            repo_lock: None,
            content_store: None,
            keep_list: None,
            signature_verification: true,
            keyring: None,
            activation_policy: "none".to_string(),
            applied_root: "/".to_string(),
        }
    }
}

/// Parse the argument vector (excluding argv[0]). Returns an Invocation or an
/// invocation-domain Diagnostic (the CLI layer maps that to exit 2). The flag
/// aliases --version/--help/-h are accepted only for the two global commands.
pub fn parse_args(args: &[String]) -> Result<Invocation, Diagnostic> {
    let mut verb: Option<Verb> = None;
    let mut opts = Options::default();
    let mut saw_bareword = false;

    for arg in args {
        // Tolerated flag aliases for the global commands (only).
        match arg.as_str() {
            "--version" => {
                set_global_verb(&mut verb, Verb::Version)?;
                continue;
            }
            "--help" | "-h" => {
                set_global_verb(&mut verb, Verb::Help)?;
                continue;
            }
            _ => {}
        }

        if let Some((key, value)) = arg.split_once('=') {
            // key=value option. Reject empty value (missing required value).
            apply_option(&mut opts, key, value)?;
        } else if arg.starts_with('-') {
            // Any other POSIX-style flag is not a recognised option.
            return Err(invocation_usage(&format!("unknown option '{}'", arg)));
        } else {
            // bare word -> a verb.
            match Verb::parse(arg) {
                Some(v) => {
                    if verb.is_some() && saw_bareword {
                        return Err(invocation_usage(&format!("unexpected argument '{}'", arg)));
                    }
                    verb = Some(v);
                    saw_bareword = true;
                }
                None => {
                    return Err(invocation_usage(&format!("unknown verb '{}'", arg)));
                }
            }
        }
    }

    Ok(Invocation { verb, opts })
}

fn set_global_verb(verb: &mut Option<Verb>, v: Verb) -> Result<(), Diagnostic> {
    *verb = Some(v);
    Ok(())
}

fn apply_option(opts: &mut Options, key: &str, value: &str) -> Result<(), Diagnostic> {
    // A present key with an empty value is a missing required value.
    if value.is_empty() {
        return Err(invocation_usage(&format!(
            "missing value for option '{}='",
            key
        )));
    }
    match key {
        "mode" | "transaction-mode" => {
            opts.mode = TransactionMode::parse(value).ok_or_else(|| {
                invocation_usage(&format!("unknown value '{}' for {}", value, key))
            })?;
        }
        "manifest-path" => opts.manifest_path = Some(value.to_string()),
        "manifest-format" => {
            opts.manifest_format = ManifestFormat::parse(value).ok_or_else(|| {
                invocation_usage(&format!("unknown value '{}' for manifest-format", value))
            })?;
        }
        "format" => {
            opts.format = Some(ManifestFormat::parse(value).ok_or_else(|| {
                invocation_usage(&format!("unknown value '{}' for format", value))
            })?);
        }
        "state-path" => opts.state_path = Some(value.to_string()),
        "root" => opts.root = value.to_string(),
        "out" => opts.out = Some(value.to_string()),
        "on-unreadable" => {
            opts.on_unreadable = OnUnreadable::parse(value).ok_or_else(|| {
                invocation_usage(&format!("unknown value '{}' for on-unreadable", value))
            })?;
        }
        "scope" => {
            opts.scope = ScanScope::parse(value)
                .ok_or_else(|| invocation_usage(&format!("unknown value '{}' for scope", value)))?;
        }
        "repo-lock" => opts.repo_lock = Some(value.to_string()),
        "content-store" => opts.content_store = Some(value.to_string()),
        "keep-list" => opts.keep_list = Some(value.to_string()),
        "signature-verification" => {
            opts.signature_verification = match value {
                "on" => true,
                "off" => false,
                _ => {
                    return Err(invocation_usage(&format!(
                        "unknown value '{}' for signature-verification",
                        value
                    )))
                }
            };
        }
        "keyring" => opts.keyring = Some(value.to_string()),
        "activation-policy" => match value {
            "reboot" | "soft-reboot" | "none" => opts.activation_policy = value.to_string(),
            _ => {
                return Err(invocation_usage(&format!(
                    "unknown value '{}' for activation-policy",
                    value
                )))
            }
        },
        "applied-root" => opts.applied_root = value.to_string(),
        other => {
            return Err(invocation_usage(&format!("unknown option '{}='", other)));
        }
    }
    Ok(())
}

fn invocation_usage(message: &str) -> Diagnostic {
    Diagnostic::error(Domain::Invocation, message.to_string())
}

/// Load the keep-list file into a set of absolute paths (one per line). Missing
/// file yields an empty set (the keep-list is optional).
pub fn load_keep_list(opts: &Options) -> std::collections::HashSet<String> {
    let mut set = std::collections::HashSet::new();
    if let Some(path) = opts.keep_list.as_ref() {
        if let Ok(content) = std::fs::read_to_string(path) {
            for line in content.lines() {
                let t = line.trim();
                if !t.is_empty() && !t.starts_with('#') {
                    set.insert(t.to_string());
                }
            }
        }
    }
    set
}
