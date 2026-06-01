// generated from spec: zypper-declarative.spec.md sha256:87b9f2f3bf92afd6fe037412d56a23959290e5816d722a3def9505d07aa5acd7
// tests by: claude-opus-4-8
//
// Shared black-box test harness helpers. The binary under test is located at
// the canonical BINARY-LOCATION the cli-tool template mandates: project root,
// i.e. ../../zypper-declarative relative to independent_tests/<llm-name>/.

#![allow(dead_code)]

use std::path::{Path, PathBuf};
use std::process::{Command, Output};
use std::sync::Once;

/// Binary name as declared by the spec title (lowercase-hyphenated).
pub const BINARY_NAME: &str = "zypper-declarative";

static BUILD_ONCE: Once = Once::new();

/// Absolute path to the project root (two directories up from this test crate).
/// independent_tests/<llm-name>/  ->  ../../  is the project root.
pub fn project_root() -> PathBuf {
    // CARGO_MANIFEST_DIR points at independent_tests/<llm-name>/
    let manifest_dir = PathBuf::from(env!("CARGO_MANIFEST_DIR"));
    manifest_dir
        .parent() // independent_tests/
        .and_then(|p| p.parent()) // project root
        .map(|p| p.to_path_buf())
        .expect("project root resolvable from test manifest dir")
}

/// Canonical path to the binary under test: ../../zypper-declarative
pub fn binary_path() -> PathBuf {
    project_root().join(BINARY_NAME)
}

/// Build the binary once at the canonical project-root location if it is not
/// already present. The translator places the entry point at src/main.rs and
/// builds the release binary; we copy it to the project root if needed.
pub fn ensure_binary() {
    BUILD_ONCE.call_once(|| {
        let bin = binary_path();
        if bin.exists() {
            return;
        }
        // Attempt to build via the project Makefile's build target, which the
        // translator wires to place the binary at the project root.
        let root = project_root();
        let status = Command::new("make")
            .arg("build")
            .current_dir(&root)
            .status();
        if let Ok(s) = status {
            if s.success() && bin.exists() {
                return;
            }
        }
        // Fallback: cargo build --release then copy the artifact to the root.
        let cargo_status = Command::new("cargo")
            .args(["build", "--release"])
            .current_dir(&root)
            .status();
        if let Ok(s) = cargo_status {
            if s.success() {
                // The artifact may be at target/release/ or, when a default build
                // target is configured, target/<triple>/release/.
                for rel in [
                    "target/release",
                    "target/x86_64-unknown-linux-gnu/release",
                ] {
                    let artifact = root.join(rel).join(BINARY_NAME);
                    if artifact.exists() {
                        let _ = std::fs::copy(&artifact, &bin);
                        break;
                    }
                }
            }
        }
    });
}

/// Run the binary with the given args (each already split into a key=value or
/// bare word token). Returns the captured Output.
pub fn run(args: &[&str]) -> Output {
    ensure_binary();
    Command::new(binary_path())
        .args(args)
        .output()
        .expect("failed to spawn zypper-declarative binary")
}

/// Convenience: exit code as i32 (panics if killed by signal).
pub fn exit_code(out: &Output) -> i32 {
    out.status.code().expect("process exited via signal, not code")
}

pub fn stdout_str(out: &Output) -> String {
    String::from_utf8_lossy(&out.stdout).into_owned()
}

pub fn stderr_str(out: &Output) -> String {
    String::from_utf8_lossy(&out.stderr).into_owned()
}

/// Create a unique temporary directory under the system temp dir.
pub fn temp_dir(tag: &str) -> PathBuf {
    let base = std::env::temp_dir();
    let pid = std::process::id();
    let nanos = std::time::SystemTime::now()
        .duration_since(std::time::UNIX_EPOCH)
        .unwrap()
        .as_nanos();
    let dir = base.join(format!("zd-test-{}-{}-{}", tag, pid, nanos));
    std::fs::create_dir_all(&dir).expect("create temp dir");
    dir
}

pub fn write_file(dir: &Path, name: &str, content: &str) -> PathBuf {
    let p = dir.join(name);
    if let Some(parent) = p.parent() {
        std::fs::create_dir_all(parent).expect("create parent dir");
    }
    std::fs::write(&p, content).expect("write fixture file");
    p
}

/// A structurally-complete desired manifest in canonical JSON. Callers may
/// mutate the returned serde_json::Value for negative-path fixtures.
pub fn complete_desired_manifest_json() -> serde_json::Value {
    serde_json::json!({
        "meta": {
            "format_version": 1,
            "generator": "zypper-declarative 0.6.3",
            "created_at": "2026-05-29T08:30:00Z",
            "desired_sha256": ""
        },
        "repositories": {
            "_attributes": { "repository_system": "zypp" },
            "_elements": [
                {
                    "alias": "sl-micro-6.2-pinned",
                    "name": "SL Micro 6.2 (pinned)",
                    "url": "https://internal.example/obs/SLMicro:6.2:pinned/standard",
                    "type": "rpm-md",
                    "enabled": true,
                    "gpgcheck": true,
                    "autorefresh": false,
                    "priority": 99
                }
            ]
        },
        "packages": {
            "_attributes": { "package_system": "rpm" },
            "_elements": [
                { "name": "nginx", "version": "", "release": "", "arch": "" }
            ]
        },
        "services": {
            "_attributes": { "init_system": "systemd" },
            "_elements": [
                { "name": "nginx.service", "state": "enabled" }
            ]
        },
        "config_files": {
            "_attributes": {},
            "_elements": [
                {
                    "name": "/etc/foo.conf",
                    "type": "file",
                    "mode": "0644",
                    "user": "root",
                    "group": "root",
                    "sha256": "0000000000000000000000000000000000000000000000000000000000000000",
                    "target": "",
                    "content_ref": "files/etc/foo.conf",
                    "package_name": ""
                }
            ]
        }
    })
}
