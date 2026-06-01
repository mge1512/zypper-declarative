// generated from spec: zypper-declarative.spec.md sha256:18253550a5c3d3f1818f0380811cb5dbc98874828693e49fc9cd5cbc923303dd
//
// Diagnostics, severities, domains, and the verb-layer exit-code mapping.
//
// Internal behaviours return `Diagnostic`s (or `Result<_, Diagnostic>`) to
// their caller rather than exiting; exit-code mapping lives only in the verb
// layer (the `cli` module). This mirrors the spec note: "exit-code mapping
// lives only in the verbs".

use std::fmt;

/// Diagnostic severity (spec TYPES: Severity := Error | Warning).
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

/// Diagnostic domain (spec TYPES: Diagnostic.domain). One of:
/// packages | repositories | services | files | manifest | transaction |
/// invocation. (`units` is the spec's domain name for service-state drift; it
/// is surfaced via [`Domain::Units`].)
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum Domain {
    Packages,
    Repositories,
    Services,
    Units,
    Files,
    Manifest,
    Transaction,
    Invocation,
}

impl Domain {
    pub fn as_str(self) -> &'static str {
        match self {
            Domain::Packages => "packages",
            Domain::Repositories => "repositories",
            Domain::Services => "services",
            Domain::Units => "units",
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

/// A diagnostic carrying a severity, a domain, and a human-readable message.
/// `Diagnostic` is also the internal-behaviour error type.
#[derive(Debug, Clone, PartialEq, Eq)]
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

    /// Render the diagnostic as a single stderr line, including the domain so
    /// tests (and operators) can identify the affected scope.
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

/// The process exit code (spec TYPES: ExitCode := 0 | 1 | 2).
#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ExitCode {
    /// 0 = success: convergence complete or no-op, system matches declaration,
    /// or describe produced output.
    Ok = 0,
    /// 1 = logical failure: convergence failed and discarded; verify found
    /// drift; manifest invalid or unverified; state collection failed.
    Logical = 1,
    /// 2 = invocation error: bad arguments; manifest unreadable; insufficient
    /// privilege; transaction mechanism unavailable; output path unwritable.
    Invocation = 2,
}

impl ExitCode {
    pub fn code(self) -> i32 {
        self as i32
    }
}

/// The verb-layer mapping from a behaviour error to an exit code.
///
/// Per the spec, a read/format failure on the manifest is an invocation error
/// (exit 2), while schema, unsafe-YAML, and signature failures are logical
/// failures (exit 1). The `Domain::Invocation` carries the read/format/argument
/// errors and maps to 2; all other domains map to 1 by default. The caller may
/// already know an exit code (e.g. transaction unavailable is exit 2 even
/// though its domain is `transaction`); those callers select the code directly.
pub fn default_exit_for_domain(domain: Domain) -> ExitCode {
    match domain {
        Domain::Invocation => ExitCode::Invocation,
        // packages, repositories, services, units, files, manifest are logical
        // failures by default.
        _ => ExitCode::Logical,
    }
}
