// generated from spec: zypper-declarative.spec.md
// sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
//
// Abstract external-system interfaces (## INTERFACES): the package manager, the
// init system, and the alternatives database are driven via a CommandRunner
// trait so the production path execs the real tools and tests can substitute a
// double. Independent (black-box) tests do NOT use these doubles — they invoke
// the built binary — but the doubles let the in-tree unit tests cover the
// parsing logic without a live system.

use std::process::Command;

/// Runs an external command and returns (stdout, stderr, success).
/// Unlike a bare error, this surfaces stdout even when the command exits
/// non-zero, because rpm -V reports differences with a non-zero exit and that
/// is a NORMAL successful outcome to parse, not a failure.
pub trait CommandRunner: Send + Sync {
    fn run(&self, cmd: &str, args: &[&str]) -> CommandResult;
}

#[derive(Debug, Clone, Default)]
pub struct CommandResult {
    pub stdout: String,
    pub stderr: String,
    pub success: bool,
    /// True if the command could not be spawned at all (e.g. binary missing).
    pub spawn_failed: bool,
}

/// Production CommandRunner: execs the real tool with a sanitised PATH.
pub struct OsCommandRunner;

impl CommandRunner for OsCommandRunner {
    fn run(&self, cmd: &str, args: &[&str]) -> CommandResult {
        match Command::new(cmd)
            .args(args)
            .env("PATH", "/sbin:/bin:/usr/bin:/usr/sbin")
            .output()
        {
            Ok(output) => CommandResult {
                stdout: String::from_utf8_lossy(&output.stdout).into_owned(),
                stderr: String::from_utf8_lossy(&output.stderr).into_owned(),
                success: output.status.success(),
                spawn_failed: false,
            },
            Err(e) => CommandResult {
                stdout: String::new(),
                stderr: e.to_string(),
                success: false,
                spawn_failed: true,
            },
        }
    }
}

#[cfg(test)]
pub struct FakeCommandRunner {
    pub responses: std::collections::HashMap<String, CommandResult>,
}

#[cfg(test)]
impl CommandRunner for FakeCommandRunner {
    fn run(&self, cmd: &str, _args: &[&str]) -> CommandResult {
        self.responses.get(cmd).cloned().unwrap_or_default()
    }
}
