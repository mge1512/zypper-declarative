// generated from spec: zypper-declarative.spec.md sha256:87b9f2f3bf92afd6fe037412d56a23959290e5816d722a3def9505d07aa5acd7
//
// INTERFACES. The tool integrates with external systems (the package manager,
// the init system, the filesystem, the transaction mechanism) as abstract
// dependencies. We drive zypper/systemctl/rpm by executing their command-line
// interfaces (std::process::Command) and read repos.d as files; this keeps the
// binary free of FFI to the SUSE C/C++ libraries and lets it stay static.
//
// Each interface is a trait with a production implementation and a test double.
// Independent (black-box) tests use only the built binary; these test doubles are
// for in-crate unit tests of the orchestration logic.

use std::collections::HashMap;

/// Runs an external command, returning (stdout, stderr) on success.
pub trait CommandRunner: Send + Sync {
    fn run(&self, cmd: &str, args: &[&str]) -> Result<(String, String), CommandError>;
}

#[derive(Debug, Clone)]
pub struct CommandError {
    pub command: String,
    pub message: String,
    /// The process exit status code, if it exited normally. A non-zero status is
    /// not necessarily an error for callers that interpret "differences found"
    /// (e.g. rpm -V); the caller inspects this rather than treating any non-zero
    /// as failure.
    pub status: Option<i32>,
    pub stdout: String,
    pub stderr: String,
}

impl std::fmt::Display for CommandError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        write!(f, "{} failed: {}", self.command, self.message)
    }
}

impl std::error::Error for CommandError {}

/// Production CommandRunner: executes via std::process::Command with a fixed PATH.
pub struct OSCommandRunner;

impl CommandRunner for OSCommandRunner {
    fn run(&self, cmd: &str, args: &[&str]) -> Result<(String, String), CommandError> {
        use std::process::Command;
        let output = Command::new(cmd)
            .args(args)
            .env("PATH", "/sbin:/bin:/usr/bin:/usr/sbin")
            .output()
            .map_err(|e| CommandError {
                command: cmd.to_string(),
                message: e.to_string(),
                status: None,
                stdout: String::new(),
                stderr: String::new(),
            })?;
        let stdout = String::from_utf8_lossy(&output.stdout).into_owned();
        let stderr = String::from_utf8_lossy(&output.stderr).into_owned();
        if output.status.success() {
            Ok((stdout, stderr))
        } else {
            Err(CommandError {
                command: cmd.to_string(),
                message: format!("exit status {:?}", output.status.code()),
                status: output.status.code(),
                stdout,
                stderr,
            })
        }
    }
}

/// Filesystem read interface. Reads only; the tool never modifies its inputs.
pub trait Filesystem: Send + Sync {
    fn read_file(&self, path: &str) -> Result<Vec<u8>, std::io::Error>;
    fn exists(&self, path: &str) -> bool;
    /// List the .repo files under a repos.d directory (full paths).
    fn list_repo_files(&self, dir: &str) -> Result<Vec<String>, std::io::Error>;
}

/// Production Filesystem: std::fs.
pub struct OSFilesystem;

impl Filesystem for OSFilesystem {
    fn read_file(&self, path: &str) -> Result<Vec<u8>, std::io::Error> {
        std::fs::read(path)
    }
    fn exists(&self, path: &str) -> bool {
        std::path::Path::new(path).exists()
    }
    fn list_repo_files(&self, dir: &str) -> Result<Vec<String>, std::io::Error> {
        let mut out = Vec::new();
        for entry in std::fs::read_dir(dir)? {
            let entry = entry?;
            let p = entry.path();
            if p.extension().and_then(|e| e.to_str()) == Some("repo") {
                out.push(p.to_string_lossy().into_owned());
            }
        }
        out.sort();
        Ok(out)
    }
}

// ---------------------------------------------------------------------------
// Test doubles (used by in-crate unit tests only; never by the black-box suite).
// ---------------------------------------------------------------------------

/// A test double for CommandRunner with canned responses keyed by command name.
pub struct FakeCommandRunner {
    pub responses: HashMap<String, Result<(String, String), CommandError>>,
}

impl FakeCommandRunner {
    pub fn new() -> Self {
        FakeCommandRunner {
            responses: HashMap::new(),
        }
    }
    pub fn with(mut self, cmd: &str, stdout: &str, stderr: &str) -> Self {
        self.responses.insert(
            cmd.to_string(),
            Ok((stdout.to_string(), stderr.to_string())),
        );
        self
    }
}

impl Default for FakeCommandRunner {
    fn default() -> Self {
        Self::new()
    }
}

impl CommandRunner for FakeCommandRunner {
    fn run(&self, cmd: &str, _args: &[&str]) -> Result<(String, String), CommandError> {
        match self.responses.get(cmd) {
            Some(Ok(v)) => Ok(v.clone()),
            Some(Err(e)) => Err(e.clone()),
            None => Ok((String::new(), String::new())),
        }
    }
}

/// A test double for Filesystem backed by an in-memory map.
pub struct FakeFilesystem {
    pub files: HashMap<String, Vec<u8>>,
    pub repo_files: HashMap<String, Vec<String>>,
}

impl FakeFilesystem {
    pub fn new() -> Self {
        FakeFilesystem {
            files: HashMap::new(),
            repo_files: HashMap::new(),
        }
    }
}

impl Default for FakeFilesystem {
    fn default() -> Self {
        Self::new()
    }
}

impl Filesystem for FakeFilesystem {
    fn read_file(&self, path: &str) -> Result<Vec<u8>, std::io::Error> {
        self.files
            .get(path)
            .cloned()
            .ok_or_else(|| std::io::Error::new(std::io::ErrorKind::NotFound, path.to_string()))
    }
    fn exists(&self, path: &str) -> bool {
        self.files.contains_key(path)
    }
    fn list_repo_files(&self, dir: &str) -> Result<Vec<String>, std::io::Error> {
        self.repo_files
            .get(dir)
            .cloned()
            .ok_or_else(|| std::io::Error::new(std::io::ErrorKind::NotFound, dir.to_string()))
    }
}
