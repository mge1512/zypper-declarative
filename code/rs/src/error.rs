// generated from spec: zypper-declarative.spec.md sha256:87b9f2f3bf92afd6fe037412d56a23959290e5816d722a3def9505d07aa5acd7
//
// Diagnostics and the internal error type. Internal behaviours return errors to
// their caller; exit-code mapping lives only in the verb layer (cli.rs). The
// Diagnostic carries a domain so the verb layer maps domain -> exit code without
// string matching.

use std::fmt;

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Severity {
    Error,
    Warning,
}

impl Severity {
    pub fn as_str(&self) -> &'static str {
        match self {
            Severity::Error => "error",
            Severity::Warning => "warning",
        }
    }
}

/// Diagnostic domain. Per the spec's Diagnostic TYPE the domain string set is:
/// packages | repositories | files | units | manifest | transaction | invocation.
/// (The unit/service drift domain is spelled `units` in the spec; the enum
/// variant is named Units to match.)
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Domain {
    Packages,
    Repositories,
    Units,
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
            Domain::Units => "units",
            Domain::Files => "files",
            Domain::Manifest => "manifest",
            Domain::Transaction => "transaction",
            Domain::Invocation => "invocation",
        }
    }
}

/// A single diagnostic: a severity, a domain, and a human-readable message.
#[derive(Debug, Clone)]
pub struct Diagnostic {
    pub severity: Severity,
    pub domain: Domain,
    pub message: String,
}

impl Diagnostic {
    pub fn error(domain: Domain, message: impl Into<String>) -> Self {
        Diagnostic {
            severity: Severity::Error,
            domain,
            message: message.into(),
        }
    }

    pub fn warning(domain: Domain, message: impl Into<String>) -> Self {
        Diagnostic {
            severity: Severity::Warning,
            domain,
            message: message.into(),
        }
    }

    /// Render one diagnostic line for stderr: "<severity> [<domain>] <message>".
    pub fn render(&self) -> String {
        format!(
            "{} [{}] {}",
            self.severity.as_str(),
            self.domain.as_str(),
            self.message
        )
    }
}

impl fmt::Display for Diagnostic {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "{}", self.render())
    }
}

impl std::error::Error for Diagnostic {}

/// The exit codes the spec defines.
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ExitCode {
    /// 0 = success
    Ok = 0,
    /// 1 = logical failure
    Logical = 1,
    /// 2 = invocation error
    Invocation = 2,
}

impl ExitCode {
    pub fn code(self) -> i32 {
        self as i32
    }
}

/// The internal Result alias used throughout the implementation; errors are
/// Diagnostics carrying their domain.
pub type DResult<T> = Result<T, Diagnostic>;
