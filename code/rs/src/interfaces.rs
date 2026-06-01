// generated from spec: zypper-declarative.spec.md sha256:18253550a5c3d3f1818f0380811cb5dbc98874828693e49fc9cd5cbc923303dd
//
// INTERFACES: the abstract dependencies on external systems (package manager,
// init system, transaction mechanism). Modelled as Rust traits so the production
// implementation (driving the CLIs via std::process::Command) and test doubles
// share a surface. Independent black-box tests do not use these — they invoke
// the built binary — but the internal layering keeps `describe-actual-state` the
// single live-state reader.

use std::collections::HashMap;
use std::process::Command;

/// Runs an external command and returns (stdout, stderr, exit_code).
///
/// A non-zero exit is NOT an error at this layer: a package verifier reporting
/// differences commonly exits non-zero, and that is the normal successful
/// outcome. Callers decide whether a non-zero status is meaningful.
pub trait CommandRunner: Send + Sync {
    fn run(&self, cmd: &str, args: &[&str]) -> CommandResult;
}

/// The outcome of running an external command.
pub struct CommandResult {
    pub stdout: String,
    pub stderr: String,
    pub code: i32,
    /// True only if the command could not be spawned at all (a genuine
    /// access/exec failure), which IS an unreadable-source condition.
    pub spawn_failed: bool,
}

/// The production CommandRunner: drives the real CLIs with a fixed PATH.
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
                code: output.status.code().unwrap_or(-1),
                spawn_failed: false,
            },
            Err(e) => CommandResult {
                stdout: String::new(),
                stderr: e.to_string(),
                code: -1,
                spawn_failed: true,
            },
        }
    }
}

/// A scripted CommandRunner for tests (in-tree unit tests only; the independent
/// black-box suite does not use it).
#[allow(dead_code)]
pub struct FakeCommandRunner {
    pub responses: HashMap<String, CommandResult>,
}

#[allow(dead_code)]
impl CommandRunner for FakeCommandRunner {
    fn run(&self, cmd: &str, _args: &[&str]) -> CommandResult {
        match self.responses.get(cmd) {
            Some(r) => CommandResult {
                stdout: r.stdout.clone(),
                stderr: r.stderr.clone(),
                code: r.code,
                spawn_failed: r.spawn_failed,
            },
            None => CommandResult {
                stdout: String::new(),
                stderr: String::new(),
                code: 0,
                spawn_failed: false,
            },
        }
    }
}
