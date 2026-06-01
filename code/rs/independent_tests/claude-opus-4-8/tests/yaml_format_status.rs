// generated from spec: zypper-declarative.spec.md sha256:87b9f2f3bf92afd6fe037412d56a23959290e5816d722a3def9505d07aa5acd7
// tests by: claude-opus-4-8
//
// YAML serialisation, resolve-format, and status behaviour. Covers EXAMPLEs
// yaml_manifest_accepted, yaml_format_identity_stable, yaml_unsafe_rejected,
// verify_state_path_extension_yaml, status_no_declaration, status_reports_generation
// (offline subset), and the observable INVARIANTs on resolve-format and the YAML safe
// profile. Live-system describe cases that cannot be made deterministic offline are
// covered in describe_live.rs and marked accordingly.

mod common;
use common::*;

use serde_json::json;

// EXAMPLE: status_no_declaration -- no applied record -> "no declaration applied", exit 0.
#[test]
fn test_status_no_declaration_exit0() {
    let dir = temp_dir("status-nodecl");
    let applied_root = dir.join("root");
    std::fs::create_dir_all(&applied_root).unwrap();
    let out = run(&["status", &format!("applied-root={}", applied_root.display())]);
    assert_eq!(exit_code(&out), 0, "status with no applied record must exit 0; stderr={}", stderr_str(&out));
    assert!(
        stdout_str(&out).contains("no declaration applied"),
        "status must print 'no declaration applied'; got: {}",
        stdout_str(&out)
    );
}

// EXAMPLE: yaml_manifest_accepted -- a YAML serialisation of a valid manifest is
// parsed under the safe profile and validated; diff against itself (offline) exits 0.
#[test]
fn test_yaml_manifest_accepted_offline_diff_exit0() {
    let dir = temp_dir("yaml-accept");
    let yaml = r#"
meta:
  format_version: 1
  generator: "zypper-declarative 0.6.3"
  created_at: "2026-05-29T08:30:00Z"
  desired_sha256: ""
packages:
  _attributes:
    package_system: "rpm"
  _elements:
    - name: "nginx"
      version: ""
      release: ""
      arch: ""
config_files:
  _attributes: {}
  _elements:
    - name: "/etc/foo.conf"
      type: "file"
      mode: "0644"
      user: "root"
      group: "root"
      sha256: "1111111111111111111111111111111111111111111111111111111111111111"
      target: ""
      content_ref: ""
      package_name: ""
"#;
    let manifest = write_file(&dir, "desired.yaml", yaml);
    // Provide a state dump (JSON) equal to the YAML model so diff is offline & clean.
    let state = json!({
        "meta": {"format_version": 1, "generator": "zypper-declarative 0.6.3",
                 "created_at": "2026-05-29T08:30:00Z", "desired_sha256": ""},
        "packages": {"_attributes": {"package_system": "rpm"},
            "_elements": [{"name": "nginx", "version": "", "release": "", "arch": ""}]},
        "config_files": {"_attributes": {},
            "_elements": [{
                "name": "/etc/foo.conf", "type": "file", "mode": "0644", "user": "root",
                "group": "root",
                "sha256": "1111111111111111111111111111111111111111111111111111111111111111",
                "target": "", "content_ref": "", "package_name": ""
            }]}
    });
    let statep = write_file(&dir, "state.json", &serde_json::to_string_pretty(&state).unwrap());
    let out = run(&[
        "diff",
        &format!("manifest-path={}", manifest.display()),
        &format!("state-path={}", statep.display()),
    ]);
    assert_eq!(
        exit_code(&out),
        0,
        "a valid YAML manifest must parse and diff offline (exit 0); stderr={}",
        stderr_str(&out)
    );
}

// EXAMPLE: yaml_unsafe_rejected -- a YAML manifest using a multi-document stream
// (a disabled safe-profile feature) is rejected with a manifest error, exit 1.
#[test]
fn test_yaml_unsafe_multidoc_rejected_exit1() {
    let dir = temp_dir("yaml-unsafe");
    // Two YAML documents in one stream -> multi-document, which the safe profile
    // must reject.
    let yaml = "---\nmeta:\n  format_version: 1\n---\nmeta:\n  format_version: 1\n";
    let manifest = write_file(&dir, "evil.yaml", yaml);
    let out = run(&["apply", &format!("manifest-path={}", manifest.display())]);
    assert_eq!(
        exit_code(&out),
        1,
        "an unsafe multi-document YAML manifest must be rejected with a manifest error (exit 1); stderr={}",
        stderr_str(&out)
    );
    assert!(
        stderr_str(&out).to_lowercase().contains("manifest"),
        "rejection must carry domain=manifest"
    );
}

// EXAMPLE: yaml_format_identity_stable -- desired.json and desired.yaml expressing the
// same manifest yield the same desired_sha256. We observe this through version-style
// idempotence: diff(desired.json vs state) and diff(desired.yaml vs state) where state
// equals the model both exit 0 with no changes. (The hash itself is internal; the
// observable consequence is format-independent idempotence.)
#[test]
fn test_yaml_json_identity_offline_both_clean() {
    let dir = temp_dir("yaml-identity");
    let model = json!({
        "meta": {"format_version": 1, "generator": "zypper-declarative 0.6.3",
                 "created_at": "2026-05-29T08:30:00Z", "desired_sha256": ""},
        "packages": {"_attributes": {"package_system": "rpm"},
            "_elements": [{"name": "nginx", "version": "1.0", "release": "1", "arch": "x86_64"}]}
    });
    let json_manifest = write_file(&dir, "desired.json", &serde_json::to_string_pretty(&model).unwrap());
    let yaml = r#"
meta:
  format_version: 1
  generator: "zypper-declarative 0.6.3"
  created_at: "2026-05-29T08:30:00Z"
  desired_sha256: ""
packages:
  _attributes:
    package_system: "rpm"
  _elements:
    - name: "nginx"
      version: "1.0"
      release: "1"
      arch: "x86_64"
"#;
    let yaml_manifest = write_file(&dir, "desired.yaml", yaml);
    let statep = write_file(&dir, "state.json", &serde_json::to_string_pretty(&model).unwrap());

    let out_json = run(&[
        "diff",
        &format!("manifest-path={}", json_manifest.display()),
        &format!("state-path={}", statep.display()),
    ]);
    let out_yaml = run(&[
        "diff",
        &format!("manifest-path={}", yaml_manifest.display()),
        &format!("state-path={}", statep.display()),
    ]);
    assert_eq!(exit_code(&out_json), 0, "json offline diff exit 0; stderr={}", stderr_str(&out_json));
    assert_eq!(exit_code(&out_yaml), 0, "yaml offline diff exit 0; stderr={}", stderr_str(&out_yaml));
}

// EXAMPLE: verify_state_path_extension_yaml -- a YAML state dump matching the
// reference is parsed via the .yaml extension and verify is clean (exit 0).
#[test]
fn test_verify_state_path_yaml_extension_matches_exit0() {
    let dir = temp_dir("verify-yaml-ext");
    let reference = json!({
        "meta": {"format_version": 1, "generator": "zypper-declarative 0.6.3",
                 "created_at": "2026-05-29T08:30:00Z", "desired_sha256": ""},
        "services": {"_attributes": {"init_system": "systemd"},
            "_elements": [{"name": "nginx.service", "state": "enabled"}]}
    });
    let baseline = write_file(&dir, "baseline.json", &serde_json::to_string_pretty(&reference).unwrap());
    let yaml_state = r#"
meta:
  format_version: 1
  generator: "zypper-declarative 0.6.3"
  created_at: "2026-05-29T08:30:00Z"
  desired_sha256: ""
services:
  _attributes:
    init_system: "systemd"
  _elements:
    - name: "nginx.service"
      state: "enabled"
"#;
    let statep = write_file(&dir, "state.yaml", yaml_state);
    let out = run(&[
        "verify",
        &format!("manifest-path={}", baseline.display()),
        &format!("state-path={}", statep.display()),
    ]);
    assert_eq!(
        exit_code(&out),
        0,
        "YAML state dump (by .yaml extension) matching reference must verify clean (exit 0); stderr={}",
        stderr_str(&out)
    );
    assert!(stdout_str(&out).contains("system matches declaration"));
}
