// generated from spec: zypper-declarative.spec.md
// sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
//
// Diagnostics and the domain -> exit-code mapping. Internal behaviours return a
// Diagnostic to their caller; the verb layer maps the diagnostic's domain to an
// exit code. Exit-code mapping lives ONLY in the verb layer (cli), not here.

use std::fmt;

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Severity {
    Error,
    Warning,
}

impl fmt::Display for Severity {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            Severity::Error => write!(f, "error"),
            Severity::Warning => write!(f, "warning"),
        }
    }
}

/// domain := packages | repositories | services | files | manifest |
///           transaction | invocation
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Domain {
    Packages,
    Repositories,
    Services,
    Files,
    Manifest,
    Transaction,
    Invocation,
}

impl Domain {
    pub fn as_str(&self) -> &'static str {
        match self {
            Domain::Packages => "packages",
            Domain::Repositories => "repositories",
            Domain::Services => "services",
            Domain::Files => "files",
            Domain::Manifest => "manifest",
            Domain::Transaction => "transaction",
            Domain::Invocation => "invocation",
        }
    }
}

impl fmt::Display for Domain {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{}", self.as_str())
    }
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Diagnostic {
    pub severity: Severity,
    pub domain: Domain,
    pub message: String,
}

impl Diagnostic {
    pub fn error(domain: Domain, message: impl Into<String>) -> Diagnostic {
        Diagnostic {
            severity: Severity::Error,
            domain,
            message: message.into(),
        }
    }

    pub fn warning(domain: Domain, message: impl Into<String>) -> Diagnostic {
        Diagnostic {
            severity: Severity::Warning,
            domain,
            message: message.into(),
        }
    }

    /// One diagnostic line written to stderr.
    pub fn line(&self) -> String {
        format!("{}: {}: {}", self.severity, self.domain, self.message)
    }
}

impl fmt::Display for Diagnostic {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{}", self.line())
    }
}

impl std::error::Error for Diagnostic {}

/// Exit codes. The verb layer maps a terminal Diagnostic's domain to one of
/// these (see cli::exit_code_for_domain).
pub const EXIT_OK: i32 = 0;
pub const EXIT_LOGICAL: i32 = 1;
pub const EXIT_INVOCATION: i32 = 2;

/// Map a domain to its exit code, per the spec's ExitCode mapping:
///   invocation/transaction-read/unwritable -> 2
///   manifest/files/units/packages logical failures -> 1
/// The transaction domain is special: an unavailable mechanism is exit 2.
pub fn exit_code_for(diag: &Diagnostic) -> i32 {
    match diag.domain {
        Domain::Invocation => EXIT_INVOCATION,
        Domain::Transaction => EXIT_INVOCATION,
        _ => EXIT_LOGICAL,
    }
}
