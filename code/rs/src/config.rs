// generated from spec: zypper-declarative.spec.md
// sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
//
// Configuration knobs and key=value option parsing. All CONFIG knobs are also
// accepted as key=value options; a command-line option overrides preset.
// Options use key=value only (POSIX --flag forbidden, except the tolerated
// version/help aliases handled by the dispatcher). Options may appear in ANY
// position relative to the bare-word verb. Environment-variable control is
// forbidden.

use crate::error::{Diagnostic, Domain};
use crate::manifest::format::ManifestFormat;

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum OnUnreadable {
    Error,
    Warn,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum Scope {
    Etc,
    Full,
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum TransactionMode {
    Auto,
    External,
    Internal,
}

/// Parsed invocation: a bare-word verb plus key=value options. Bare words other
/// than the verb are an error (no positional arguments are defined).
#[derive(Debug, Clone, Default)]
pub struct Invocation {
    pub verb: Option<String>,
    pub mode: Option<TransactionMode>,
    pub manifest_path: Option<String>,
    pub state_path: Option<String>,
    pub root: Option<String>,
    pub out: Option<String>,
    pub on_unreadable: Option<OnUnreadable>,
    pub scope: Option<Scope>,
    pub format: Option<ManifestFormat>,
    // CONFIG knobs accepted as key=value:
    pub manifest_format: Option<ManifestFormat>,
    pub repo_lock: Option<String>,
    pub content_store: Option<String>,
    pub keep_list: Option<String>,
    pub signature_verification: Option<bool>,
    pub keyring: Option<String>,
    pub activation_policy: Option<String>,
    pub applied_root: Option<String>,
    // version/help dispatcher flags
    pub want_version: bool,
    pub want_help: bool,
}

impl Invocation {
    /// resolve manifest-format default (json unless overridden).
    pub fn manifest_format_default(&self) -> ManifestFormat {
        self.manifest_format.unwrap_or(ManifestFormat::Json)
    }

    pub fn on_unreadable_or_error(&self) -> OnUnreadable {
        self.on_unreadable.clone().unwrap_or(OnUnreadable::Error)
    }

    pub fn scope_or_etc(&self) -> Scope {
        self.scope.clone().unwrap_or(Scope::Etc)
    }

    pub fn root_or_slash(&self) -> String {
        self.root.clone().unwrap_or_else(|| "/".to_string())
    }

    pub fn applied_root_or_slash(&self) -> String {
        self.applied_root.clone().unwrap_or_else(|| "/".to_string())
    }

    pub fn transaction_mode_or_auto(&self) -> TransactionMode {
        self.mode.clone().unwrap_or(TransactionMode::Auto)
    }
}

/// The set of bare-word verbs and global commands.
const VERBS: &[&str] = &["apply", "diff", "verify", "status", "describe"];
const GLOBAL: &[&str] = &["version", "help"];

/// Parse argv (excluding program name) into an Invocation. Returns a
/// invocation-domain Diagnostic on any unknown verb/option/value or missing
/// value.
pub fn parse(args: &[String]) -> Result<Invocation, Diagnostic> {
    let mut inv = Invocation::default();
    for arg in args {
        // Tolerated global flag aliases (dispatcher only).
        match arg.as_str() {
            "--version" => {
                inv.want_version = true;
                continue;
            }
            "--help" | "-h" => {
                inv.want_help = true;
                continue;
            }
            _ => {}
        }

        if let Some((key, value)) = arg.split_once('=') {
            apply_kv(&mut inv, key, value)?;
            continue;
        }

        // Bare word: must be a verb or global command.
        if GLOBAL.contains(&arg.as_str()) {
            match arg.as_str() {
                "version" => inv.want_version = true,
                "help" => inv.want_help = true,
                _ => unreachable!(),
            }
            continue;
        }
        if VERBS.contains(&arg.as_str()) {
            if inv.verb.is_some() {
                return Err(Diagnostic::error(
                    Domain::Invocation,
                    format!("multiple verbs given: already had a verb, then '{}'", arg),
                ));
            }
            inv.verb = Some(arg.clone());
            continue;
        }
        return Err(Diagnostic::error(
            Domain::Invocation,
            format!("unknown argument '{}'", arg),
        ));
    }
    Ok(inv)
}

fn parse_format(value: &str) -> Result<ManifestFormat, Diagnostic> {
    ManifestFormat::parse(value).ok_or_else(|| {
        Diagnostic::error(
            Domain::Invocation,
            format!("unknown format value '{}' (expected json or yaml)", value),
        )
    })
}

fn apply_kv(inv: &mut Invocation, key: &str, value: &str) -> Result<(), Diagnostic> {
    match key {
        "mode" | "transaction-mode" => {
            inv.mode = Some(match value {
                "auto" => TransactionMode::Auto,
                "external" => TransactionMode::External,
                "internal" => TransactionMode::Internal,
                _ => {
                    return Err(Diagnostic::error(
                        Domain::Invocation,
                        format!(
                            "unknown mode value '{}' (expected auto, external, or internal)",
                            value
                        ),
                    ))
                }
            });
        }
        "manifest-path" => inv.manifest_path = Some(value.to_string()),
        "state-path" => inv.state_path = Some(value.to_string()),
        "root" => inv.root = Some(value.to_string()),
        "out" => inv.out = Some(value.to_string()),
        "format" => inv.format = Some(parse_format(value)?),
        "manifest-format" => inv.manifest_format = Some(parse_format(value)?),
        "on-unreadable" => {
            inv.on_unreadable = Some(match value {
                "error" => OnUnreadable::Error,
                "warn" => OnUnreadable::Warn,
                _ => {
                    return Err(Diagnostic::error(
                        Domain::Invocation,
                        format!(
                            "unknown on-unreadable value '{}' (expected error or warn)",
                            value
                        ),
                    ))
                }
            });
        }
        "scope" => {
            inv.scope = Some(match value {
                "etc" => Scope::Etc,
                "full" => Scope::Full,
                _ => {
                    return Err(Diagnostic::error(
                        Domain::Invocation,
                        format!("unknown scope value '{}' (expected etc or full)", value),
                    ))
                }
            });
        }
        "repo-lock" => inv.repo_lock = Some(value.to_string()),
        "content-store" => inv.content_store = Some(value.to_string()),
        "keep-list" => inv.keep_list = Some(value.to_string()),
        "signature-verification" => {
            inv.signature_verification = Some(match value {
                "on" => true,
                "off" => false,
                _ => {
                    return Err(Diagnostic::error(
                        Domain::Invocation,
                        format!(
                            "unknown signature-verification value '{}' (expected on or off)",
                            value
                        ),
                    ))
                }
            });
        }
        "keyring" => inv.keyring = Some(value.to_string()),
        "activation-policy" => inv.activation_policy = Some(value.to_string()),
        "applied-root" => inv.applied_root = Some(value.to_string()),
        _ => {
            return Err(Diagnostic::error(
                Domain::Invocation,
                format!("unknown option '{}'", key),
            ))
        }
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    fn args(v: &[&str]) -> Vec<String> {
        v.iter().map(|s| s.to_string()).collect()
    }

    #[test]
    fn parses_verb_and_options_any_position() {
        let inv = parse(&args(&["manifest-path=/m.json", "diff", "format=yaml"])).unwrap();
        assert_eq!(inv.verb.as_deref(), Some("diff"));
        assert_eq!(inv.manifest_path.as_deref(), Some("/m.json"));
        assert_eq!(inv.format, Some(ManifestFormat::Yaml));
    }

    #[test]
    fn unknown_format_value_is_invocation_error() {
        let e = parse(&args(&["describe", "format=toml"])).unwrap_err();
        assert_eq!(e.domain, Domain::Invocation);
    }

    #[test]
    fn unknown_verb_is_error() {
        let e = parse(&args(&["frobnicate"])).unwrap_err();
        assert_eq!(e.domain, Domain::Invocation);
    }

    #[test]
    fn version_and_help_flags() {
        assert!(parse(&args(&["--version"])).unwrap().want_version);
        assert!(parse(&args(&["--help"])).unwrap().want_help);
        assert!(parse(&args(&["-h"])).unwrap().want_help);
        assert!(parse(&args(&["version"])).unwrap().want_version);
        assert!(parse(&args(&["help"])).unwrap().want_help);
    }
}
