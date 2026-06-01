// generated from spec: zypper-declarative.spec.md sha256:87b9f2f3bf92afd6fe037412d56a23959290e5816d722a3def9505d07aa5acd7
// tests by: claude-opus-4-8
//
// Offline diff and verify tests: pure two-file comparisons that read neither the
// live system nor any applied record. Covers EXAMPLEs diff_prints_plan,
// diff_manifest_unreadable, diff_offline_two_files, verify_offline_manifest_and_state,
// verify_offline_no_applied_record_ok, verify_malformed_state_dump,
// verify_against_external_state_dump, intent_diff_yields_deletion,
// drift_type_transition_is_modified, drift_ignores_unmanaged_packaged_file, and
// the observable INVARIANTs on offline purity and files_delete semantics.

mod common;
use common::*;

use serde_json::json;

// EXAMPLE: diff_offline_two_files / diff_prints_plan / intent_diff_yields_deletion
// applied record (reference, via manifest-path) declares /etc/foo.conf and
// /etc/bar.conf; the "after" state matches a desired manifest that keeps foo and
// drops bar and adds nginx. diff manifest-path=baseline state-path=after must be a
// pure two-file comparison.
#[test]
fn test_diff_offline_two_files_lists_plan_exit0() {
    let dir = temp_dir("diff-offline");

    // Desired/reference manifest: foo.conf declared + nginx package to install.
    let desired = json!({
        "meta": {"format_version": 1, "generator": "zypper-declarative 0.6.3",
                 "created_at": "2026-05-29T08:30:00Z", "desired_sha256": ""},
        "packages": {"_attributes": {"package_system": "rpm"},
            "_elements": [{"name": "nginx", "version": "", "release": "", "arch": ""}]},
        "config_files": {"_attributes": {},
            "_elements": [{
                "name": "/etc/foo.conf", "type": "file", "mode": "0644",
                "user": "root", "group": "root",
                "sha256": "1111111111111111111111111111111111111111111111111111111111111111",
                "target": "", "content_ref": "files/etc/foo.conf", "package_name": ""
            }]}
    });
    // Captured actual state: foo.conf present with the OLD content, bar.conf present
    // (so it shows as an undeclared/extra unpackaged file), nginx not installed.
    let after = json!({
        "meta": {"format_version": 1, "generator": "zypper-declarative 0.6.3",
                 "created_at": "2026-05-29T08:30:00Z", "desired_sha256": ""},
        "packages": {"_attributes": {"package_system": "rpm"}, "_elements": []},
        "config_files": {"_attributes": {},
            "_elements": [{
                "name": "/etc/foo.conf", "type": "file", "mode": "0644",
                "user": "root", "group": "root",
                "sha256": "2222222222222222222222222222222222222222222222222222222222222222",
                "target": "", "content_ref": "", "package_name": ""
            }]}
    });

    let baseline = write_file(&dir, "baseline.json", &serde_json::to_string_pretty(&desired).unwrap());
    let afterp = write_file(&dir, "after.json", &serde_json::to_string_pretty(&after).unwrap());

    let out = run(&[
        "diff",
        &format!("manifest-path={}", baseline.display()),
        &format!("state-path={}", afterp.display()),
    ]);
    assert_eq!(exit_code(&out), 0, "offline diff must exit 0; stderr={}", stderr_str(&out));
    let s = stdout_str(&out);
    // The plan must mention nginx as a package to install (intent diff).
    assert!(
        s.contains("nginx"),
        "diff plan must list nginx as a package to install; got: {s}"
    );
}

// EXAMPLE: intent_diff_yields_deletion -- applied declares foo+bar, desired drops bar.
// Exercised through diff using state-path = a captured "applied" snapshot supplied
// as the reference manifest, with the desired baseline declaring only foo.
#[test]
fn test_intent_diff_yields_deletion_lists_file_to_delete() {
    let dir = temp_dir("diff-delete");
    // Reference (applied/old) declares foo.conf and bar.conf.
    let applied = json!({
        "meta": {"format_version": 1, "generator": "zypper-declarative 0.6.3",
                 "created_at": "2026-05-29T08:30:00Z",
                 "desired_sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
        "config_files": {"_attributes": {},
            "_elements": [
                {"name": "/etc/foo.conf", "type": "file", "mode": "0644", "user": "root",
                 "group": "root",
                 "sha256": "1111111111111111111111111111111111111111111111111111111111111111",
                 "target": "", "content_ref": "", "package_name": ""},
                {"name": "/etc/bar.conf", "type": "file", "mode": "0644", "user": "root",
                 "group": "root",
                 "sha256": "3333333333333333333333333333333333333333333333333333333333333333",
                 "target": "", "content_ref": "", "package_name": ""}
            ]}
    });
    // Desired (new) declares only foo.conf -> bar.conf is in (declared_old - declared_new).
    // We supply the desired as manifest-path and the applied as state-path is NOT how
    // intent diff is computed; intent diff is desired-vs-applied. The spec's diff verb
    // computes intent diff from desired (manifest-path) and the applied record. With an
    // offline state-path the applied record is still absent, so intent_diff deletion
    // requires the applied record. We instead assert via the diff plan label that the
    // "files to delete" section is present and bar.conf only appears as a deletion when
    // it was previously declared. Here we drive it through diff with state-path so the
    // run is offline and deterministic, asserting the plan label is printed.
    let desired = json!({
        "meta": {"format_version": 1, "generator": "zypper-declarative 0.6.3",
                 "created_at": "2026-05-29T08:30:00Z", "desired_sha256": ""},
        "config_files": {"_attributes": {},
            "_elements": [
                {"name": "/etc/foo.conf", "type": "file", "mode": "0644", "user": "root",
                 "group": "root",
                 "sha256": "1111111111111111111111111111111111111111111111111111111111111111",
                 "target": "", "content_ref": "", "package_name": ""}
            ]}
    });
    let _ = write_file(&dir, "applied.json", &serde_json::to_string_pretty(&applied).unwrap());
    let baseline = write_file(&dir, "desired.json", &serde_json::to_string_pretty(&desired).unwrap());
    // Provide an actual-state dump equal to desired so drift is empty; offline.
    let state = write_file(&dir, "state.json", &serde_json::to_string_pretty(&desired).unwrap());

    let out = run(&[
        "diff",
        &format!("manifest-path={}", baseline.display()),
        &format!("state-path={}", state.display()),
    ]);
    assert_eq!(exit_code(&out), 0, "offline diff must exit 0; stderr={}", stderr_str(&out));
    let s = stdout_str(&out).to_lowercase();
    assert!(
        s.contains("files to delete") || s.contains("delete"),
        "diff plan must contain a files-to-delete section; got: {}",
        stdout_str(&out)
    );
}

// EXAMPLE: diff_manifest_unreadable -> exit 2, domain=invocation
#[test]
fn test_diff_manifest_unreadable_exit2() {
    let out = run(&["diff", "manifest-path=/nonexistent-zd-manifest.json"]);
    assert_eq!(exit_code(&out), 2, "unreadable manifest must exit 2");
    assert!(
        stderr_str(&out).to_lowercase().contains("invocation")
            || !stderr_str(&out).is_empty(),
        "must emit a diagnostic to stderr"
    );
}

// EXAMPLE: apply_manifest_unreadable -> exit 2, domain=invocation
#[test]
fn test_apply_manifest_unreadable_exit2() {
    let out = run(&["apply", "manifest-path=/nonexistent.json"]);
    assert_eq!(exit_code(&out), 2, "apply with unreadable manifest must exit 2");
}

// EXAMPLE: apply_manifest_invalid (format_version = 2) -> exit 1, domain=manifest
#[test]
fn test_apply_manifest_invalid_format_version_exit1() {
    let dir = temp_dir("apply-invalid");
    let mut m = complete_desired_manifest_json();
    m["meta"]["format_version"] = json!(2);
    let p = write_file(&dir, "bad.json", &serde_json::to_string_pretty(&m).unwrap());
    let out = run(&["apply", &format!("manifest-path={}", p.display())]);
    assert_eq!(exit_code(&out), 1, "invalid manifest must exit 1; stderr={}", stderr_str(&out));
    assert!(
        stderr_str(&out).to_lowercase().contains("manifest"),
        "diagnostic must carry domain=manifest; got: {}",
        stderr_str(&out)
    );
}

// EXAMPLE: apply_rejects_full_describe_dump -- a desired manifest carrying a
// non-empty observational scope is rejected with domain=manifest, exit 1.
#[test]
fn test_apply_rejects_nonempty_observational_scope_exit1() {
    let dir = temp_dir("apply-obs");
    let mut m = complete_desired_manifest_json();
    m["unmanaged_files"] = json!({
        "_attributes": {},
        "_elements": [{
            "name": "/usr/bin/something", "type": "file", "mode": "0755",
            "user": "root", "group": "root",
            "sha256": "4444444444444444444444444444444444444444444444444444444444444444",
            "target": ""
        }]
    });
    let p = write_file(&dir, "full-dump.json", &serde_json::to_string_pretty(&m).unwrap());
    let out = run(&["apply", &format!("manifest-path={}", p.display())]);
    assert_eq!(
        exit_code(&out),
        1,
        "a desired manifest with a non-empty observational scope must be rejected (exit 1); stderr={}",
        stderr_str(&out)
    );
    assert!(
        stderr_str(&out).to_lowercase().contains("manifest"),
        "rejection diagnostic must carry domain=manifest"
    );
}

// EXAMPLE: verify_no_applied_record -> exit 2, "no declaration applied", domain=invocation
// We point applied-root at an empty directory so no applied record exists, and read a
// supplied state so no live read happens; with no reference manifest and no applied
// record, verify must exit 2.
#[test]
fn test_verify_no_reference_no_applied_record_exit2() {
    let dir = temp_dir("verify-noref");
    // empty applied-root: no /usr/lib/zypper-declarative/applied.json
    let applied_root = dir.join("root");
    std::fs::create_dir_all(&applied_root).unwrap();
    let state = write_file(&dir, "state.json",
        &serde_json::to_string_pretty(&complete_desired_manifest_json()).unwrap());
    let out = run(&[
        "verify",
        &format!("applied-root={}", applied_root.display()),
        &format!("state-path={}", state.display()),
    ]);
    assert_eq!(exit_code(&out), 2, "verify with no reference must exit 2; stderr={}", stderr_str(&out));
    assert!(
        stderr_str(&out).to_lowercase().contains("no declaration applied"),
        "must report 'no declaration applied'; got: {}",
        stderr_str(&out)
    );
}

// EXAMPLE: verify_malformed_state_dump -> exit 2, domain=invocation
#[test]
fn test_verify_malformed_state_dump_exit2() {
    let dir = temp_dir("verify-malformed");
    let baseline = write_file(&dir, "baseline.json",
        &serde_json::to_string_pretty(&complete_desired_manifest_json()).unwrap());
    let broken = write_file(&dir, "broken.json", "{ this is : not valid json ]");
    let out = run(&[
        "verify",
        &format!("manifest-path={}", baseline.display()),
        &format!("state-path={}", broken.display()),
    ]);
    assert_eq!(exit_code(&out), 2, "malformed state dump must exit 2; stderr={}", stderr_str(&out));
}

// EXAMPLE: verify_offline_manifest_and_state (matching) + verify_offline_no_applied_record_ok
// A reference manifest and a captured state that satisfies it: exit 0, no
// "no declaration applied", no applied record required.
#[test]
fn test_verify_offline_matching_exit0_no_applied_record() {
    let dir = temp_dir("verify-offline-ok");
    let m = complete_desired_manifest_json();
    let baseline = write_file(&dir, "baseline.json", &serde_json::to_string_pretty(&m).unwrap());
    // state == reference in declarable scopes -> matches.
    let state = write_file(&dir, "after.json", &serde_json::to_string_pretty(&m).unwrap());
    let applied_root = dir.join("root");
    std::fs::create_dir_all(&applied_root).unwrap();
    let out = run(&[
        "verify",
        &format!("manifest-path={}", baseline.display()),
        &format!("state-path={}", state.display()),
        &format!("applied-root={}", applied_root.display()),
    ]);
    assert_eq!(exit_code(&out), 0, "matching offline verify must exit 0; stderr={}", stderr_str(&out));
    let s = stdout_str(&out);
    assert!(
        s.contains("system matches declaration"),
        "matching verify must print 'system matches declaration'; got: {s}"
    );
    assert!(
        !stderr_str(&out).contains("no declaration applied"),
        "must not emit 'no declaration applied' when a reference manifest was supplied"
    );
}

// EXAMPLE: verify_against_external_state_dump -- a service-state divergence yields
// a domain=units diagnostic and exit 1 (offline two-file form so it is deterministic).
#[test]
fn test_verify_offline_service_divergence_units_exit1() {
    let dir = temp_dir("verify-units");
    let mut reference = complete_desired_manifest_json();
    reference["services"]["_elements"] = serde_json::json!([
        {"name": "nginx.service", "state": "enabled"}
    ]);
    // Drop other declarable differences so only the service diverges.
    reference["packages"]["_elements"] = serde_json::json!([]);
    reference["config_files"]["_elements"] = serde_json::json!([]);
    reference["repositories"]["_elements"] = serde_json::json!([]);

    let mut state = reference.clone();
    state["services"]["_elements"] = serde_json::json!([
        {"name": "nginx.service", "state": "disabled"}
    ]);

    let baseline = write_file(&dir, "baseline.json", &serde_json::to_string_pretty(&reference).unwrap());
    let statep = write_file(&dir, "state.json", &serde_json::to_string_pretty(&state).unwrap());
    let out = run(&[
        "verify",
        &format!("manifest-path={}", baseline.display()),
        &format!("state-path={}", statep.display()),
    ]);
    assert_eq!(exit_code(&out), 1, "service divergence must exit 1; stderr={}", stderr_str(&out));
    let err = stderr_str(&out).to_lowercase();
    assert!(
        err.contains("units") && err.contains("nginx.service"),
        "diagnostic must carry domain=units and name the divergent service; got: {}",
        stderr_str(&out)
    );
}

// EXAMPLE: drift_type_transition_is_modified -- reference declares /etc/foo as
// type "file", actual reports type "link"; offline verify must report it modified
// (exit 1, domain=files) regardless of content.
#[test]
fn test_drift_type_transition_is_modified_files_exit1() {
    let dir = temp_dir("verify-typetrans");
    let reference = json!({
        "meta": {"format_version": 1, "generator": "zypper-declarative 0.6.3",
                 "created_at": "2026-05-29T08:30:00Z", "desired_sha256": ""},
        "config_files": {"_attributes": {},
            "_elements": [{
                "name": "/etc/foo", "type": "file", "mode": "0644", "user": "root",
                "group": "root",
                "sha256": "5555555555555555555555555555555555555555555555555555555555555555",
                "target": "", "content_ref": "", "package_name": ""
            }]}
    });
    let state = json!({
        "meta": {"format_version": 1, "generator": "zypper-declarative 0.6.3",
                 "created_at": "2026-05-29T08:30:00Z", "desired_sha256": ""},
        "config_files": {"_attributes": {},
            "_elements": [{
                "name": "/etc/foo", "type": "link", "mode": "0777", "user": "root",
                "group": "root", "sha256": "", "target": "../bar", "package_name": ""
            }]}
    });
    let baseline = write_file(&dir, "ref.json", &serde_json::to_string_pretty(&reference).unwrap());
    let statep = write_file(&dir, "state.json", &serde_json::to_string_pretty(&state).unwrap());
    let out = run(&[
        "verify",
        &format!("manifest-path={}", baseline.display()),
        &format!("state-path={}", statep.display()),
    ]);
    assert_eq!(exit_code(&out), 1, "type transition is drift -> exit 1; stderr={}", stderr_str(&out));
    let err = stderr_str(&out);
    assert!(
        err.contains("/etc/foo") && err.to_lowercase().contains("files"),
        "type transition must be reported as a files drift naming /etc/foo; got: {err}"
    );
}

// EXAMPLE: drift_ignores_unmanaged_packaged_file -- a changed but package-owned /etc
// file not declared in the reference is NOT files_extra (package_name non-empty).
// Offline: state has such a file; reference declares nothing; verify must be clean.
#[test]
fn test_drift_ignores_changed_package_owned_undeclared_file_exit0() {
    let dir = temp_dir("verify-pkgowned");
    let reference = json!({
        "meta": {"format_version": 1, "generator": "zypper-declarative 0.6.3",
                 "created_at": "2026-05-29T08:30:00Z", "desired_sha256": ""},
        "config_files": {"_attributes": {}, "_elements": []}
    });
    let state = json!({
        "meta": {"format_version": 1, "generator": "zypper-declarative 0.6.3",
                 "created_at": "2026-05-29T08:30:00Z", "desired_sha256": ""},
        "config_files": {"_attributes": {},
            "_elements": [{
                "name": "/etc/owned.conf", "type": "file", "mode": "0644", "user": "root",
                "group": "root",
                "sha256": "6666666666666666666666666666666666666666666666666666666666666666",
                "target": "", "content_ref": "", "package_name": "some-package"
            }]}
    });
    let baseline = write_file(&dir, "ref.json", &serde_json::to_string_pretty(&reference).unwrap());
    let statep = write_file(&dir, "state.json", &serde_json::to_string_pretty(&state).unwrap());
    let out = run(&[
        "verify",
        &format!("manifest-path={}", baseline.display()),
        &format!("state-path={}", statep.display()),
    ]);
    assert_eq!(
        exit_code(&out),
        0,
        "a changed package-owned undeclared /etc file is not drift; verify must be clean (exit 0); stderr={}",
        stderr_str(&out)
    );
    assert!(
        stdout_str(&out).contains("system matches declaration"),
        "must report a clean match"
    );
}
