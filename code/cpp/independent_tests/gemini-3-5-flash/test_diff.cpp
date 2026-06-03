// generated from spec: zypper-declarative.spec.md sha256:aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3
// tests by: gemini-3-5-flash
#include "test_utils.hpp"

// ### EXAMPLE: diff_prints_plan
TEST_CASE(test_diff_prints_plan) {
    TempDir t;
    auto desired_path = t.path / "desired.json";
    auto state_path = t.path / "actual.json";

    std::string mf = R"({
        "meta": { "format_version": 1, "generator": "zypper-declarative 0.6.9", "created_at": "2026-06-03T00:00:00Z", "desired_sha256": "" },
        "packages": {
            "_attributes": { "package_system": "rpm" },
            "_elements": [
                { "name": "nginx", "version": "", "release": "", "arch": "" }
            ]
        },
        "config_files": {
            "_attributes": {},
            "_elements": []
        }
    })";

    // actual state has bar.conf but nginx is not installed
    std::string sf = R"({
        "meta": { "format_version": 1, "generator": "zypper-declarative 0.6.9", "created_at": "2026-06-03T00:00:00Z", "desired_sha256": "" },
        "packages": {
            "_attributes": { "package_system": "rpm" },
            "_elements": []
        },
        "config_files": {
            "_attributes": {},
            "_elements": [
                {
                    "name": "/etc/bar.conf",
                    "type": "file",
                    "mode": "0644",
                    "user": "root",
                    "group": "root",
                    "sha256": "0000000000000000000000000000000000000000000000000000000000000000",
                    "target": "",
                    "content_ref": "",
                    "package_name": ""
                }
            ]
        }
    })";

    write_file(desired_path, mf);
    write_file(state_path, sf);

    // Note: Diff offline compares desired vs actual.
    auto res = run_command({"../../zypper-declarative", "diff", "manifest-path=" + desired_path.string(), "state-path=" + state_path.string()});
    ASSERT_EQ(res.exit_code, 0);
    ASSERT_CONTAINS(res.stdout_data, "nginx");
    ASSERT_CONTAINS(res.stdout_data, "/etc/bar.conf");
}

// ### EXAMPLE: diff_manifest_unreadable
TEST_CASE(test_diff_manifest_unreadable) {
    auto res = run_command({"../../zypper-declarative", "diff", "manifest-path=/nonexistent.json"});
    ASSERT_EQ(res.exit_code, 2);
    ASSERT_CONTAINS(res.stderr_data, "domain=invocation");
}

// ### EXAMPLE: diff_unchanged_machine_no_drift
TEST_CASE(test_diff_unchanged_machine_no_drift) {
    TempDir t;
    auto manifest_path = t.path / "manifest.json";
    auto state_path = t.path / "actual.json";

    // Construct a state dump reflecting a fresh machine
    std::string sf = R"({
        "meta": { "format_version": 1, "generator": "zypper-declarative 0.6.9", "created_at": "2026-06-03T00:00:00Z", "desired_sha256": "" },
        "packages": {
            "_attributes": { "package_system": "rpm" },
            "_elements": []
        },
        "config_files": {
            "_attributes": {},
            "_elements": [
                {
                    "name": "/etc/local.conf",
                    "type": "file",
                    "mode": "0644",
                    "user": "root",
                    "group": "root",
                    "sha256": "0000000000000000000000000000000000000000000000000000000000000000",
                    "target": "",
                    "content_ref": "",
                    "package_name": ""
                }
            ]
        }
    })";

    write_file(manifest_path, sf);
    write_file(state_path, sf);

    auto res = run_command({"../../zypper-declarative", "diff", "manifest-path=" + manifest_path.string(), "state-path=" + state_path.string()});
    ASSERT_EQ(res.exit_code, 0);
    // Drift report should be empty because actual matches desired
    ASSERT_NOT_CONTAINS(res.stdout_data, "drift:");
}

// ### EXAMPLE: diff_offline_two_files
TEST_CASE(test_diff_offline_two_files) {
    TempDir t;
    auto baseline = t.path / "baseline.json";
    auto after = t.path / "after.json";

    std::string bf = R"({
        "meta": { "format_version": 1, "generator": "zypper-declarative 0.6.9", "created_at": "2026-06-03T00:00:00Z", "desired_sha256": "" }
    })";

    write_file(baseline, bf);
    write_file(after, bf);

    auto res = run_command({"../../zypper-declarative", "diff", "manifest-path=" + baseline.string(), "state-path=" + after.string()});
    ASSERT_EQ(res.exit_code, 0);
}
