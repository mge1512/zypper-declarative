// generated from spec: zypper-declarative.spec.md sha256:aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3
// tests by: gemini-3-5-flash
#include "test_utils.hpp"

// ### EXAMPLE: verify_clean
TEST_CASE(test_verify_clean) {
    TempDir t;
    auto manifest_path = t.path / "desired.json";
    auto state_path = t.path / "actual.json";

    std::string mf = R"({
        "meta": { "format_version": 1, "generator": "zypper-declarative 0.6.9", "created_at": "2026-06-03T00:00:00Z", "desired_sha256": "" }
    })";
    write_file(manifest_path, mf);
    write_file(state_path, mf);

    auto res = run_command({"../../zypper-declarative", "verify", "manifest-path=" + manifest_path.string(), "state-path=" + state_path.string()});
    ASSERT_EQ(res.exit_code, 0);
    ASSERT_CONTAINS(res.stdout_data, "system matches declaration");
}

// ### EXAMPLE: verify_against_external_state_dump
TEST_CASE(test_verify_against_external_state_dump) {
    TempDir t;
    auto manifest_path = t.path / "desired.json";
    auto state_path = t.path / "actual.json";

    // reference has nginx service enabled
    std::string mf = R"({
        "meta": { "format_version": 1, "generator": "zypper-declarative 0.6.9", "created_at": "2026-06-03T00:00:00Z", "desired_sha256": "" },
        "services": {
            "_attributes": { "init_system": "systemd" },
            "_elements": [
                { "name": "nginx.service", "state": "enabled" }
            ]
        }
    })";

    // state dump has nginx service disabled (diverges)
    std::string sf = R"({
        "meta": { "format_version": 1, "generator": "zypper-declarative 0.6.9", "created_at": "2026-06-03T00:00:00Z", "desired_sha256": "" },
        "services": {
            "_attributes": { "init_system": "systemd" },
            "_elements": [
                { "name": "nginx.service", "state": "disabled" }
            ]
        }
    })";

    write_file(manifest_path, mf);
    write_file(state_path, sf);

    auto res = run_command({"../../zypper-declarative", "verify", "manifest-path=" + manifest_path.string(), "state-path=" + state_path.string()});
    ASSERT_EQ(res.exit_code, 1);
    ASSERT_CONTAINS(res.stderr_data, "domain=services");
    ASSERT_CONTAINS(res.stderr_data, "nginx.service");
}

// ### EXAMPLE: verify_malformed_state_dump
TEST_CASE(test_verify_malformed_state_dump) {
    TempDir t;
    auto manifest_path = t.path / "desired.json";
    auto state_path = t.path / "malformed.json";

    std::string mf = R"({
        "meta": { "format_version": 1, "generator": "zypper-declarative 0.6.9", "created_at": "2026-06-03T00:00:00Z", "desired_sha256": "" }
    })";
    write_file(manifest_path, mf);
    write_file(state_path, "not a valid json document }");

    auto res = run_command({"../../zypper-declarative", "verify", "manifest-path=" + manifest_path.string(), "state-path=" + state_path.string()});
    ASSERT_EQ(res.exit_code, 2);
    ASSERT_CONTAINS(res.stderr_data, "domain=invocation");
}

// ### EXAMPLE: verify_detects_drift
TEST_CASE(test_verify_detects_drift) {
    TempDir t;
    auto manifest_path = t.path / "desired.json";
    auto state_path = t.path / "actual.json";

    // reference declares /etc/foo.conf with hash AAA
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

    // state has /etc/foo.conf with hash BBB
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
                    "sha256": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
                    "target": "",
                    "content_ref": "",
                    "package_name": ""
                }
            ]
        }
    })";

    write_file(manifest_path, mf);
    write_file(state_path, sf);

    auto res = run_command({"../../zypper-declarative", "verify", "manifest-path=" + manifest_path.string(), "state-path=" + state_path.string()});
    ASSERT_EQ(res.exit_code, 1);
    ASSERT_CONTAINS(res.stderr_data, "domain=files");
    ASSERT_CONTAINS(res.stderr_data, "/etc/foo.conf");
}

// ### EXAMPLE: verify_no_applied_record
TEST_CASE(test_verify_no_applied_record) {
    // If we run without manifest-path or state-path and there is no applied record, verify should exit 2 with domain=invocation
    // Since we are unprivileged and there won't be an applied record in /usr/lib/zypper-declarative/applied.json, this should fail unprivileged.
    // Let's run it live unprivileged!
    auto res = run_command({"../../zypper-declarative", "verify"});
    ASSERT_EQ(res.exit_code, 2);
    ASSERT_CONTAINS(res.stderr_data, "domain=invocation");
    ASSERT_CONTAINS(res.stderr_data, "no declaration applied");
}

// ### EXAMPLE: verify_default_scope_ignores_usr
TEST_CASE(test_verify_default_scope_ignores_usr) {
    TempDir t;
    auto manifest_path = t.path / "desired.json";
    auto state_path = t.path / "actual.json";

    std::string mf = R"({
        "meta": { "format_version": 1, "generator": "zypper-declarative 0.6.9", "created_at": "2026-06-03T00:00:00Z", "desired_sha256": "" }
    })";

    // state has unmanaged_files under /usr/bin but default verify ignores it
    std::string sf = R"({
        "meta": { "format_version": 1, "generator": "zypper-declarative 0.6.9", "created_at": "2026-06-03T00:00:00Z", "desired_sha256": "" },
        "unmanaged_files": {
            "_attributes": {},
            "_elements": [
                {
                    "name": "/usr/bin/some_tool",
                    "type": "file",
                    "mode": "0755",
                    "user": "root",
                    "group": "root",
                    "sha256": "0000000000000000000000000000000000000000000000000000000000000000"
                }
            ]
        }
    })";

    write_file(manifest_path, mf);
    write_file(state_path, sf);

    auto res = run_command({"../../zypper-declarative", "verify", "manifest-path=" + manifest_path.string(), "state-path=" + state_path.string()});
    // Defaults to etc scope which ignores unmanaged_files outside /etc
    ASSERT_EQ(res.exit_code, 0);
    ASSERT_CONTAINS(res.stdout_data, "system matches declaration");
}

// ### EXAMPLE: verify_scope_full_detects_unmanaged_addition
TEST_CASE(test_verify_scope_full_detects_unmanaged_addition) {
    TempDir t;
    auto manifest_path = t.path / "desired.json";
    auto state_path = t.path / "actual.json";

    std::string mf = R"({
        "meta": { "format_version": 1, "generator": "zypper-declarative 0.6.9", "created_at": "2026-06-03T00:00:00Z", "desired_sha256": "" }
    })";

    std::string sf = R"({
        "meta": { "format_version": 1, "generator": "zypper-declarative 0.6.9", "created_at": "2026-06-03T00:00:00Z", "desired_sha256": "" },
        "unmanaged_files": {
            "_attributes": {},
            "_elements": [
                {
                    "name": "/usr/bin/some_tool",
                    "type": "file",
                    "mode": "0755",
                    "user": "root",
                    "group": "root",
                    "sha256": "0000000000000000000000000000000000000000000000000000000000000000"
                }
            ]
        }
    })";

    write_file(manifest_path, mf);
    write_file(state_path, sf);

    auto res = run_command({"../../zypper-declarative", "verify", "scope=full", "manifest-path=" + manifest_path.string(), "state-path=" + state_path.string()});
    ASSERT_EQ(res.exit_code, 1);
    ASSERT_CONTAINS(res.stderr_data, "/usr/bin/some_tool");
}

// ### EXAMPLE: verify_scope_full_detects_modified_package_file
TEST_CASE(test_verify_scope_full_detects_modified_package_file) {
    TempDir t;
    auto manifest_path = t.path / "desired.json";
    auto state_path = t.path / "actual.json";

    std::string mf = R"({
        "meta": { "format_version": 1, "generator": "zypper-declarative 0.6.9", "created_at": "2026-06-03T00:00:00Z", "desired_sha256": "" }
    })";

    std::string sf = R"({
        "meta": { "format_version": 1, "generator": "zypper-declarative 0.6.9", "created_at": "2026-06-03T00:00:00Z", "desired_sha256": "" },
        "changed_managed_files": {
            "_attributes": {},
            "_elements": [
                {
                    "name": "/usr/share/something/file",
                    "type": "file",
                    "mode": "0644",
                    "user": "root",
                    "group": "root",
                    "sha256": "0000000000000000000000000000000000000000000000000000000000000001",
                    "package_name": "some-package",
                    "changes": ["sha256"]
                }
            ]
        }
    })";

    write_file(manifest_path, mf);
    write_file(state_path, sf);

    auto res = run_command({"../../zypper-declarative", "verify", "scope=full", "manifest-path=" + manifest_path.string(), "state-path=" + state_path.string()});
    ASSERT_EQ(res.exit_code, 1);
    ASSERT_CONTAINS(res.stderr_data, "/usr/share/something/file");
}

// ### EXAMPLE: verify_offline_manifest_and_state
TEST_CASE(test_verify_offline_manifest_and_state) {
    TempDir t;
    auto manifest_path = t.path / "desired.json";
    auto state_path = t.path / "actual.json";

    std::string mf = R"({
        "meta": { "format_version": 1, "generator": "zypper-declarative 0.6.9", "created_at": "2026-06-03T00:00:00Z", "desired_sha256": "" }
    })";
    write_file(manifest_path, mf);
    write_file(state_path, mf);

    auto res = run_command({"../../zypper-declarative", "verify", "manifest-path=" + manifest_path.string(), "state-path=" + state_path.string()});
    ASSERT_EQ(res.exit_code, 0);
}

// ### EXAMPLE: verify_offline_no_applied_record_ok
TEST_CASE(test_verify_offline_no_applied_record_ok) {
    TempDir t;
    auto manifest_path = t.path / "desired.json";
    auto state_path = t.path / "actual.json";

    std::string mf = R"({
        "meta": { "format_version": 1, "generator": "zypper-declarative 0.6.9", "created_at": "2026-06-03T00:00:00Z", "desired_sha256": "" }
    })";
    write_file(manifest_path, mf);
    write_file(state_path, mf);

    auto res = run_command({"../../zypper-declarative", "verify", "manifest-path=" + manifest_path.string(), "state-path=" + state_path.string()});
    ASSERT_EQ(res.exit_code, 0);
    // Ensure we did not complain about missing applied record since we passed explicit files
    ASSERT_NOT_CONTAINS(res.stderr_data, "no declaration applied");
}

// ### EXAMPLE: verify_state_path_extension_yaml
TEST_CASE(test_verify_state_path_extension_yaml) {
    TempDir t;
    auto manifest_path = t.path / "desired.json";
    auto state_path = t.path / "actual.yaml";

    std::string mf = R"({
        "meta": { "format_version": 1, "generator": "zypper-declarative 0.6.9", "created_at": "2026-06-03T00:00:00Z", "desired_sha256": "" }
    })";
    // State in YAML
    std::string sf = R"(
meta:
  format_version: 1
  generator: "zypper-declarative 0.6.9"
  created_at: "2026-06-03T00:00:00Z"
  desired_sha256: ""
)";

    write_file(manifest_path, mf);
    write_file(state_path, sf);

    auto res = run_command({"../../zypper-declarative", "verify", "manifest-path=" + manifest_path.string(), "state-path=" + state_path.string()});
    ASSERT_EQ(res.exit_code, 0);
}
