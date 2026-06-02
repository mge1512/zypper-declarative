// tests by: claude-opus-4-8
// generated from spec: zypper-declarative.spec.md sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
//
// Shared helpers for the black-box test suite. Included via `include!` into
// each integration test file so all tests use the same binary discovery path.
//
// Binary discovery: per the cli-tool deployment template BINARY-LOCATION
// constraint (project-root), the binary lives at ../../zypper-declarative
// relative to this test directory (independent_tests/<llm-name>/).

use std::path::PathBuf;
use std::process::{Command, Output};

/// The canonical path to the binary under test, per the template's
/// BINARY-LOCATION: project-root constraint, expressed relative to the
/// test directory `independent_tests/<llm-name>/`.
pub fn binary_path() -> PathBuf {
    // CARGO_MANIFEST_DIR is independent_tests/<llm-name>/ for this test crate.
    let manifest_dir = env!("CARGO_MANIFEST_DIR");
    let mut p = PathBuf::from(manifest_dir);
    p.push("..");
    p.push("..");
    p.push("zypper-declarative");
    p
}

/// Run the binary with the given args. Returns the raw Output.
pub fn run(args: &[&str]) -> Output {
    Command::new(binary_path())
        .args(args)
        .output()
        .unwrap_or_else(|e| panic!("failed to execute {:?}: {}", binary_path(), e))
}

pub struct RunResult {
    pub code: i32,
    pub stdout: String,
    pub stderr: String,
}

pub fn run_str(args: &[&str]) -> RunResult {
    let out = run(args);
    RunResult {
        code: out.status.code().unwrap_or(-1),
        stdout: String::from_utf8_lossy(&out.stdout).into_owned(),
        stderr: String::from_utf8_lossy(&out.stderr).into_owned(),
    }
}

/// A unique temporary directory for a test, created under the OS temp dir.
pub fn temp_dir(tag: &str) -> PathBuf {
    let mut p = std::env::temp_dir();
    let pid = std::process::id();
    let nanos = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .unwrap()
        .as_nanos();
    p.push(format!("zd-test-{}-{}-{}", tag, pid, nanos));
    std::fs::create_dir_all(&p).unwrap();
    p
}
