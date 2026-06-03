// generated from spec: zypper-declarative.spec.md sha256:aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3
// tests by: gemini-3-5-flash
#include "test_utils.hpp"

// ### EXAMPLE: status_reports_generation
TEST_CASE(test_status_reports_generation) {
    TempDir t;
    auto app_dir = t.path / "usr/lib/zypper-declarative";
    fs::create_directories(app_dir);
    std::string applied_rec = R"({
        "meta": { 
            "format_version": 1, 
            "generator": "zypper-declarative 0.6.9", 
            "created_at": "2026-06-03T00:00:00Z", 
            "desired_sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" 
        },
        "packages": {
            "_attributes": { "package_system": "rpm" },
            "_elements": [
                { "name": "nginx", "version": "1.25.1", "release": "1.1", "arch": "x86_64" }
            ]
        }
    })";
    write_file(app_dir / "applied.json", applied_rec);

    // Run status with synthetic applied-root. Pass on-unreadable=warn to avoid aborting on protected live-system sources during actual state reading.
    auto res = run_command({"../../zypper-declarative", "status", "applied-root=" + t.path.string(), "on-unreadable=warn"});
    ASSERT_EQ(res.exit_code, 0);
    ASSERT_CONTAINS(res.stdout_data, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa");
    ASSERT_CONTAINS(res.stdout_data, "packages");
}

// ### EXAMPLE: status_no_declaration
TEST_CASE(test_status_no_declaration) {
    TempDir t; // empty applied root (no applied.json)
    auto res = run_command({"../../zypper-declarative", "status", "applied-root=" + t.path.string()});
    ASSERT_EQ(res.exit_code, 0);
    ASSERT_CONTAINS(res.stdout_data, "no declaration applied");
}

// ### EXAMPLE: status_unknown_argument
TEST_CASE(test_status_unknown_argument) {
    auto res = run_command({"../../zypper-declarative", "status", "--frobnicate"});
    ASSERT_EQ(res.exit_code, 2);
    ASSERT_CONTAINS(res.stderr_data, "usage:");
}
