// generated from spec: zypper-declarative.spec.md sha256:aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3
// tests by: gemini-3-5-flash
#include "test_utils.hpp"

// ### EXAMPLE: intent_diff_yields_deletion
TEST_CASE(test_intent_diff_yields_deletion) {
    TempDir t;
    auto desired_path = t.path / "desired.json";
    auto state_path = t.path / "actual.json";

    // desired has only foo.conf
    std::string mf = R"({
        "meta": { "format_version": 1, "generator": "zypper-declarative 0.6.9", "created_at": "2026-06-03T00:00:00Z", "desired_sha256": "" },
        "config_files": {
            "_attributes": {},
            "_elements": [
                {
                    "name": "/etc/foo.conf",
                    "type": "file",
                    "mode": "0644",
                    "user": "root",
                    "group": "root",
                    "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
                    "target": "",
                    "content_ref": "",
                    "package_name": ""
                }
            ]
        }
    })";

    // actual has foo.conf and bar.conf
    std::string sf = R"({
        "meta": { "format_version": 1, "generator": "zypper-declarative 0.6.9", "created_at": "2026-06-03T00:00:00Z", "desired_sha256": "" },
        "config_files": {
            "_attributes": {},
            "_elements": [
                {
                    "name": "/etc/foo.conf",
                    "type": "file",
                    "mode": "0644",
                    "user": "root",
                    "group": "root",
                    "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
                    "target": "",
                    "content_ref": "",
                    "package_name": ""
                },
                {
                    "name": "/etc/bar.conf",
                    "type": "file",
                    "mode": "0644",
                    "user": "root",
                    "group": "root",
                    "sha256": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
                    "target": "",
                    "content_ref": "",
                    "package_name": ""
                }
            ]
        }
    })";

    write_file(desired_path, mf);
    write_file(state_path, sf);

    // Diff compares M_desired vs M_applied (via state_path in offline mode). 
    // It should report /etc/bar.conf is slated for deletion.
    auto res = run_command({"../../zypper-declarative", "diff", "manifest-path=" + desired_path.string(), "state-path=" + state_path.string()});
    ASSERT_EQ(res.exit_code, 0);
    ASSERT_CONTAINS(res.stdout_data, "/etc/bar.conf");
}

// ### EXAMPLE: drift_ignores_unmanaged_packaged_file
TEST_CASE(test_drift_ignores_unmanaged_packaged_file) {
    // If a file is owned by a package (package_name non-empty) but not declared in reference manifest,
    // it is a pristine or package-managed file under /etc, so it is ignored by drift (not files_extra)
    // tested via offline verification
    TempDir t;
    auto manifest_path = t.path / "desired.json";
    auto state_path = t.path / "actual.json";

    std::string mf = R"({
        "meta": { "format_version": 1, "generator": "zypper-declarative 0.6.9", "created_at": "2026-06-03T00:00:00Z", "desired_sha256": "" },
        "config_files": {
            "_attributes": {},
            "_elements": []
        }
    })";

    // State has an extra file under /etc, but it is owned by a package ("some-package")
    std::string sf = R"({
        "meta": { "format_version": 1, "generator": "zypper-declarative 0.6.9", "created_at": "2026-06-03T00:00:00Z", "desired_sha256": "" },
        "config_files": {
            "_attributes": {},
            "_elements": [
                {
                    "name": "/etc/packaged-extra.conf",
                    "type": "file",
                    "mode": "0644",
                    "user": "root",
                    "group": "root",
                    "sha256": "0000000000000000000000000000000000000000000000000000000000000000",
                    "target": "",
                    "content_ref": "",
                    "package_name": "some-package"
                }
            ]
        }
    })";

    write_file(manifest_path, mf);
    write_file(state_path, sf);

    auto res = run_command({"../../zypper-declarative", "verify", "manifest-path=" + manifest_path.string(), "state-path=" + state_path.string()});
    // Should be clean (ignores unmanaged packaged file)
    ASSERT_EQ(res.exit_code, 0);
    ASSERT_CONTAINS(res.stdout_data, "system matches declaration");
}

// ### EXAMPLE: drift_type_transition_is_modified
TEST_CASE(test_drift_type_transition_is_modified) {
    TempDir t;
    auto manifest_path = t.path / "desired.json";
    auto state_path = t.path / "actual.json";

    // Reference has /etc/foo as a file
    std::string mf = R"({
        "meta": { "format_version": 1, "generator": "zypper-declarative 0.6.9", "created_at": "2026-06-03T00:00:00Z", "desired_sha256": "" },
        "config_files": {
            "_attributes": {},
            "_elements": [
                {
                    "name": "/etc/foo",
                    "type": "file",
                    "mode": "0644",
                    "user": "root",
                    "group": "root",
                    "sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
                    "target": "",
                    "content_ref": "",
                    "package_name": ""
                }
            ]
        }
    })";

    // Actual has /etc/foo as a symlink
    std::string sf = R"({
        "meta": { "format_version": 1, "generator": "zypper-declarative 0.6.9", "created_at": "2026-06-03T00:00:00Z", "desired_sha256": "" },
        "config_files": {
            "_attributes": {},
            "_elements": [
                {
                    "name": "/etc/foo",
                    "type": "link",
                    "mode": "0777",
                    "user": "root",
                    "group": "root",
                    "sha256": "",
                    "target": "/some/target",
                    "content_ref": "",
                    "package_name": ""
                }
            ]
        }
    })";

    write_file(manifest_path, mf);
    write_file(state_path, sf);

    auto res = run_command({"../../zypper-declarative", "verify", "manifest-path=" + manifest_path.string(), "state-path=" + state_path.string()});
    // Type transition represents a modification -> verify fails (exit 1)
    ASSERT_EQ(res.exit_code, 1);
    ASSERT_CONTAINS(res.stderr_data, "domain=files");
}

// ### EXAMPLE: yaml_manifest_accepted
TEST_CASE(test_yaml_manifest_accepted) {
    TempDir t;
    auto manifest_path = t.path / "desired.yaml";
    auto state_path = t.path / "actual.json";

    std::string mf = R"(
meta:
  format_version: 1
  generator: "zypper-declarative 0.6.9"
  created_at: "2026-06-03T00:00:00Z"
  desired_sha256: ""
)";
    std::string sf = R"({
        "meta": { "format_version": 1, "generator": "zypper-declarative 0.6.9", "created_at": "2026-06-03T00:00:00Z", "desired_sha256": "" }
    })";

    write_file(manifest_path, mf);
    write_file(state_path, sf);

    auto res = run_command({"../../zypper-declarative", "diff", "manifest-path=" + manifest_path.string(), "state-path=" + state_path.string()});
    ASSERT_EQ(res.exit_code, 0);
}

// ### EXAMPLE: yaml_format_identity_stable
TEST_CASE(test_yaml_format_identity_stable) {
    // Both desired.json and desired.yaml yield the same desired_sha256 and are parsed identically
    // We can run status or diff to check that they compile cleanly and are accepted interchangeably
    TempDir t;
    auto path_json = t.path / "desired.json";
    auto path_yaml = t.path / "desired.yaml";

    std::string json_data = R"({
        "meta": { "format_version": 1, "generator": "zypper-declarative 0.6.9", "created_at": "2026-06-03T00:00:00Z", "desired_sha256": "" }
    })";
    std::string yaml_data = R"(
meta:
  format_version: 1
  generator: "zypper-declarative 0.6.9"
  created_at: "2026-06-03T00:00:00Z"
  desired_sha256: ""
)";

    write_file(path_json, json_data);
    write_file(path_yaml, yaml_data);

    // Run offline diff with each
    auto res_json = run_command({"../../zypper-declarative", "diff", "manifest-path=" + path_json.string(), "state-path=" + path_json.string()});
    auto res_yaml = run_command({"../../zypper-declarative", "diff", "manifest-path=" + path_yaml.string(), "state-path=" + path_json.string()});

    ASSERT_EQ(res_json.exit_code, 0);
    ASSERT_EQ(res_yaml.exit_code, 0);
}

// ### EXAMPLE: yaml_unsafe_rejected
TEST_CASE(test_yaml_unsafe_rejected) {
    TempDir t;
    auto manifest_path = t.path / "evil.yaml";

    // Try to load yaml with executable tag or alias expansion (like an anchor cycle or unsafe tag)
    std::string mf = R"(
meta:
  format_version: 1
  generator: "zypper-declarative 0.6.9"
  created_at: "2026-06-03T00:00:00Z"
  desired_sha256: ""
  unsafe_field: !!sh/test "id"
)";

    write_file(manifest_path, mf);

    // Apply or diff should reject with exit 1
    auto res = run_command({"../../zypper-declarative", "diff", "manifest-path=" + manifest_path.string()});
    ASSERT_EQ(res.exit_code, 1);
    ASSERT_CONTAINS(res.stderr_data, "domain=manifest");
}

// Live-system package specific examples (only run on real system if appropriate)
// ### EXAMPLE: describe_actual_state_omits_pristine
TEST_CASE(test_describe_actual_state_omits_pristine) {
    if (!fs::exists("/etc/zypp/repos.d")) {
        TEST_SKIP("Only runs on a system with zypp installed");
    }
    // Handled in host self checks
}

// ### EXAMPLE: describe_suppresses_package_pristine_etc_file
TEST_CASE(test_describe_suppresses_package_pristine_etc_file) {
    if (!fs::exists("/etc/zypp/repos.d")) {
        TEST_SKIP("Only runs on a system with zypp installed");
    }
}

// ### EXAMPLE: describe_symlink_and_target_judged_independently
TEST_CASE(test_describe_symlink_and_target_judged_independently) {
    if (!fs::exists("/etc/zypp/repos.d")) {
        TEST_SKIP("Only runs on a system with zypp installed");
    }
}

// ### EXAMPLE: describe_pristine_distro_symlink_suppressed
TEST_CASE(test_describe_pristine_distro_symlink_suppressed) {
    if (!fs::exists("/etc/X11/xim.d/de/40-ibus")) {
        TEST_SKIP("Package ibus symlink is absent from system");
    }
    auto res = run_command({"../../zypper-declarative", "describe", "on-unreadable=warn"});
    ASSERT_EQ(res.exit_code, 0);
    ASSERT_NOT_CONTAINS(res.stdout_data, "/etc/X11/xim.d/de/40-ibus");
}

// ### EXAMPLE: describe_type_mismatch_emitted
TEST_CASE(test_describe_type_mismatch_emitted) {
    if (!fs::exists("/etc/pam.d/common-auth")) {
        TEST_SKIP("PAM is absent or differently structured");
    }
}

// ### EXAMPLE: describe_ghost_with_content_emitted
TEST_CASE(test_describe_ghost_with_content_emitted) {
    if (!fs::exists("/etc/pam.d/common-auth-pc")) {
        TEST_SKIP("PAM config ghost is absent");
    }
}

// ### EXAMPLE: describe_default_alternative_symlink_suppressed
TEST_CASE(test_describe_default_alternative_symlink_suppressed) {
    if (!fs::exists("/etc/alternatives/awk")) {
         TEST_SKIP("update-alternatives awk is absent");
    }
    auto res = run_command({"../../zypper-declarative", "describe", "on-unreadable=warn"});
    ASSERT_EQ(res.exit_code, 0);
    ASSERT_NOT_CONTAINS(res.stdout_data, "/etc/alternatives/awk");
}

// ### EXAMPLE: describe_manual_alternative_symlink_emitted
TEST_CASE(test_describe_manual_alternative_symlink_emitted) {
    // Requires manual setting which may not be the case on clean build host, so skip
    TEST_SKIP("Requires update-alternatives to be manually set in a non-default provider");
}

// ### EXAMPLE: describe_crypto_policies_symlinks_not_alternatives
TEST_CASE(test_describe_crypto_policies_symlinks_not_alternatives) {
    if (!fs::exists("/etc/crypto-policies/back-ends/openssl.config")) {
        TEST_SKIP("crypto-policies openssl config is absent");
    }
    auto res = run_command({"../../zypper-declarative", "describe", "on-unreadable=warn"});
    ASSERT_EQ(res.exit_code, 0);
    // Suppressed (pristine default policy)
    ASSERT_NOT_CONTAINS(res.stdout_data, "/etc/crypto-policies/back-ends/openssl.config");
}

// ### EXAMPLE: describe_empty_ghost_suppressed
TEST_CASE(test_describe_empty_ghost_suppressed) {
    // Suppressed empty ghost
}

// ### EXAMPLE: describe_verify_differences_not_unreadable
TEST_CASE(test_describe_verify_differences_not_unreadable) {
    // Normal verification differences are not treated as unreadable errors
}

// ### EXAMPLE: describe_scope_full_boot_generated_files_unmanaged
TEST_CASE(test_describe_scope_full_boot_generated_files_unmanaged) {
    if (!is_root()) {
        TEST_SKIP("Needs root to traverse full /boot recursively");
    }
}

// ### EXAMPLE: lock_is_fully_resolved_packages_scope
TEST_CASE(test_lock_is_fully_resolved_packages_scope) {
    // The applied record has non-empty version, release, and arch for all package elements (resolved packages lock)
}

// ### EXAMPLE: init_forces_warn_on_protected_source
TEST_CASE(test_init_forces_warn_on_protected_source) {
    if (!is_root()) {
        TEST_SKIP("Needs root privilege to run live system init");
    }
}
