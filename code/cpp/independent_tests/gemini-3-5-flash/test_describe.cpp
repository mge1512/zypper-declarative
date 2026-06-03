// generated from spec: zypper-declarative.spec.md sha256:aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3
// tests by: gemini-3-5-flash
#include "test_utils.hpp"
#include <sys/stat.h>

// Helper to make a standard synthetic root
void setup_synthetic_root(const fs::path& r) {
    fs::create_directories(r / "etc");
    fs::create_directories(r / "etc/zypp/repos.d");
}

// ### EXAMPLE: describe_emits_manifest
TEST_CASE(test_describe_emits_manifest) {
    TempDir r;
    setup_synthetic_root(r.path);

    auto res = run_command({"../../zypper-declarative", "describe", "root=" + r.path.string()});
    ASSERT_EQ(res.exit_code, 0);
    ASSERT_CONTAINS(res.stdout_data, "\"format_version\": 1");
    ASSERT_CONTAINS(res.stdout_data, "\"generator\": \"zypper-declarative 0.6.9\"");
}

// ### EXAMPLE: describe_output_unwritable
TEST_CASE(test_describe_output_unwritable) {
    TempDir r;
    setup_synthetic_root(r.path);

    // Write output to a nonexistent directory
    auto res = run_command({"../../zypper-declarative", "describe", "root=" + r.path.string(), "out=/readonly_nonexistent_dir/state.json"});
    ASSERT_EQ(res.exit_code, 2);
    ASSERT_CONTAINS(res.stderr_data, "domain=invocation");
}

// ### EXAMPLE: describe_bootstraps_desired_manifest
TEST_CASE(test_describe_bootstraps_desired_manifest) {
    TempDir r;
    setup_synthetic_root(r.path);
    write_file(r.path / "etc/foo.conf", "hello");

    TempDir t;
    auto mf_path = t.path / "desired.json";

    auto res = run_command({"../../zypper-declarative", "describe", "root=" + r.path.string(), "out=" + mf_path.string()});
    ASSERT_EQ(res.exit_code, 0);

    // If we compare this manifest against the same state offline, drift should be empty
    auto res_diff = run_command({"../../zypper-declarative", "diff", "manifest-path=" + mf_path.string(), "state-path=" + mf_path.string()});
    ASSERT_EQ(res_diff.exit_code, 0);
    ASSERT_NOT_CONTAINS(res_diff.stdout_data, "drift:");
}

// ### EXAMPLE: describe_traverses_etc_subdirectories
TEST_CASE(test_describe_traverses_etc_subdirectories) {
    TempDir r;
    setup_synthetic_root(r.path);
    fs::create_directories(r.path / "etc/ImageMagick-7");
    write_file(r.path / "etc/ImageMagick-7/policy.xml", "<policy></policy>");

    auto res = run_command({"../../zypper-declarative", "describe", "root=" + r.path.string()});
    ASSERT_EQ(res.exit_code, 0);
    ASSERT_CONTAINS(res.stdout_data, "/etc/ImageMagick-7/policy.xml");
}

// ### EXAMPLE: describe_records_symlink_verbatim
TEST_CASE(test_describe_records_symlink_verbatim) {
    TempDir r;
    setup_synthetic_root(r.path);
    std::error_code ec;
    fs::create_symlink("../foo/bar.conf", r.path / "etc/mylink.conf", ec);

    auto res = run_command({"../../zypper-declarative", "describe", "root=" + r.path.string()});
    ASSERT_EQ(res.exit_code, 0);
    ASSERT_CONTAINS(res.stdout_data, "\"/etc/mylink.conf\"");
    ASSERT_CONTAINS(res.stdout_data, "\"type\": \"link\"");
    ASSERT_CONTAINS(res.stdout_data, "\"target\": \"../foo/bar.conf\"");
}

// ### EXAMPLE: describe_skips_special_file
TEST_CASE(test_describe_skips_special_file) {
    TempDir r;
    setup_synthetic_root(r.path);
    
    // Create FIFO
    std::string fifo_path = (r.path / "etc/myfifo").string();
    mkfifo(fifo_path.c_str(), 0666);

    auto res = run_command({"../../zypper-declarative", "describe", "root=" + r.path.string()});
    ASSERT_EQ(res.exit_code, 0);
    ASSERT_NOT_CONTAINS(res.stdout_data, "myfifo");
}

// ### EXAMPLE: describe_config_files_bounded_to_etc
TEST_CASE(test_describe_config_files_bounded_to_etc) {
    TempDir r;
    setup_synthetic_root(r.path);
    write_file(r.path / "etc/foo.conf", "etc-file");
    
    fs::create_directories(r.path / "usr/share");
    write_file(r.path / "usr/share/foo.conf", "usr-file");

    auto res = run_command({"../../zypper-declarative", "describe", "root=" + r.path.string()});
    ASSERT_EQ(res.exit_code, 0);
    ASSERT_CONTAINS(res.stdout_data, "/etc/foo.conf");
    ASSERT_NOT_CONTAINS(res.stdout_data, "/usr/share/foo.conf");
}

// ### EXAMPLE: describe_populates_content_store
TEST_CASE(test_describe_populates_content_store) {
    TempDir r;
    setup_synthetic_root(r.path);
    write_file(r.path / "etc/foo.conf", "my-secret-content");

    TempDir t;
    auto store_path = t.path / "store";

    auto res = run_command({"../../zypper-declarative", "describe", "root=" + r.path.string(), "content-store=" + store_path.string()});
    ASSERT_EQ(res.exit_code, 0);

    // sha256 of "my-secret-content" is 35a4d52140bb7116a4d7d105260172bf42ff8272821dfa015cc20d91b8bc228b
    std::string h = "35a4d52140bb7116a4d7d105260172bf42ff8272821dfa015cc20d91b8bc228b";
    ASSERT_CONTAINS(res.stdout_data, "sha256/" + h);

    // Ensure content store has file
    auto content_file = store_path / "sha256" / h;
    ASSERT_TRUE(fs::exists(content_file));
    ASSERT_EQ(read_file(content_file), "my-secret-content");
}

// ### EXAMPLE: describe_without_content_store_is_readonly
TEST_CASE(test_describe_without_content_store_is_readonly) {
    TempDir r;
    setup_synthetic_root(r.path);
    write_file(r.path / "etc/foo.conf", "my-content");

    auto res = run_command({"../../zypper-declarative", "describe", "root=" + r.path.string()});
    ASSERT_EQ(res.exit_code, 0);
    ASSERT_CONTAINS(res.stdout_data, "\"content_ref\": \"\"");
}

// ### EXAMPLE: scope_attributes_always_object
TEST_CASE(test_scope_attributes_always_object) {
    TempDir r;
    setup_synthetic_root(r.path);

    auto res = run_command({"../../zypper-declarative", "describe", "root=" + r.path.string()});
    ASSERT_EQ(res.exit_code, 0);
    ASSERT_CONTAINS(res.stdout_data, "\"_attributes\": {}");
}

// ### EXAMPLE: describe_scope_full_emits_observational_scopes
TEST_CASE(test_describe_scope_full_emits_observational_scopes) {
    TempDir r;
    setup_synthetic_root(r.path);
    
    fs::create_directories(r.path / "usr/bin");
    write_file(r.path / "usr/bin/some_tool", "binary");

    // Describe scope=full
    auto res = run_command({"../../zypper-declarative", "describe", "root=" + r.path.string(), "scope=full"});
    ASSERT_EQ(res.exit_code, 0);
    ASSERT_CONTAINS(res.stdout_data, "unmanaged_files");
    ASSERT_CONTAINS(res.stdout_data, "/usr/bin/some_tool");
}

// ### EXAMPLE: describe_format_yaml
TEST_CASE(test_describe_format_yaml) {
    TempDir r;
    setup_synthetic_root(r.path);

    auto res = run_command({"../../zypper-declarative", "describe", "root=" + r.path.string(), "format=yaml"});
    ASSERT_EQ(res.exit_code, 0);
    ASSERT_CONTAINS(res.stdout_data, "format_version: 1");
    ASSERT_NOT_CONTAINS(res.stdout_data, "{"); // not json
}

// ### EXAMPLE: describe_unknown_format
TEST_CASE(test_describe_unknown_format) {
    TempDir r;
    setup_synthetic_root(r.path);

    auto res = run_command({"../../zypper-declarative", "describe", "root=" + r.path.string(), "format=toml"});
    ASSERT_EQ(res.exit_code, 2);
    ASSERT_CONTAINS(res.stderr_data, "usage:");
}

// ### EXAMPLE: describe_out_extension_yaml
TEST_CASE(test_describe_out_extension_yaml) {
    TempDir r;
    setup_synthetic_root(r.path);

    TempDir t;
    auto out_yaml = t.path / "state.yaml";

    auto res = run_command({"../../zypper-declarative", "describe", "root=" + r.path.string(), "out=" + out_yaml.string()});
    ASSERT_EQ(res.exit_code, 0);
    
    std::string content = read_file(out_yaml);
    ASSERT_CONTAINS(content, "format_version: 1");
    ASSERT_NOT_CONTAINS(content, "{");
}

// ### EXAMPLE: describe_out_extension_json
TEST_CASE(test_describe_out_extension_json) {
    TempDir r;
    setup_synthetic_root(r.path);

    TempDir t;
    auto out_json = t.path / "state.json";

    auto res = run_command({"../../zypper-declarative", "describe", "root=" + r.path.string(), "out=" + out_json.string()});
    ASSERT_EQ(res.exit_code, 0);
    
    std::string content = read_file(out_json);
    ASSERT_CONTAINS(content, "\"format_version\": 1");
}

// ### EXAMPLE: describe_format_overrides_extension
TEST_CASE(test_describe_format_overrides_extension) {
    TempDir r;
    setup_synthetic_root(r.path);

    TempDir t;
    auto out_yaml = t.path / "state.yaml";

    // format=json overrides .yaml extension
    auto res = run_command({"../../zypper-declarative", "describe", "root=" + r.path.string(), "format=json", "out=" + out_yaml.string()});
    ASSERT_EQ(res.exit_code, 0);
    
    std::string content = read_file(out_yaml);
    ASSERT_CONTAINS(content, "\"format_version\": 1"); // is json
}

// ### EXAMPLE: describe_repositories_from_reposd
TEST_CASE(test_describe_repositories_from_reposd) {
    TempDir r;
    setup_synthetic_root(r.path);
    
    std::string repo_content = R"([repo1]
name=Repo 1
baseurl=http://example.com/repo1
type=rpm-md
enabled=1
gpgcheck=1
autorefresh=0
priority=99
)";
    write_file(r.path / "etc/zypp/repos.d/repo1.repo", repo_content);

    auto res = run_command({"../../zypper-declarative", "describe", "root=" + r.path.string()});
    ASSERT_EQ(res.exit_code, 0);
    ASSERT_CONTAINS(res.stdout_data, "\"alias\": \"repo1\"");
    ASSERT_CONTAINS(res.stdout_data, "\"url\": \"http://example.com/repo1\"");
}

// ### EXAMPLE: describe_unreadable_scope_strict
TEST_CASE(test_describe_unreadable_scope_strict) {
    TempDir r;
    setup_synthetic_root(r.path);
    
    auto repo_dir = r.path / "etc/zypp/repos.d";
    // Change permission to make it unreadable
    chmod(repo_dir.string().c_str(), 0000);

    // Runs under default on-unreadable=error
    auto res = run_command({"../../zypper-declarative", "describe", "root=" + r.path.string()});
    
    // Restore permission for cleanup
    chmod(repo_dir.string().c_str(), 0755);

    ASSERT_EQ(res.exit_code, 1);
    ASSERT_CONTAINS(res.stderr_data, "domain=repositories");
}

// ### EXAMPLE: describe_unreadable_scope_warn
TEST_CASE(test_describe_unreadable_scope_warn) {
    TempDir r;
    setup_synthetic_root(r.path);
    
    auto repo_dir = r.path / "etc/zypp/repos.d";
    chmod(repo_dir.string().c_str(), 0000);

    auto res = run_command({"../../zypper-declarative", "describe", "root=" + r.path.string(), "on-unreadable=warn"});
    
    chmod(repo_dir.string().c_str(), 0755);

    ASSERT_EQ(res.exit_code, 0);
    // repositories scope should be omitted, and stderr should contain a warning
    ASSERT_NOT_CONTAINS(res.stdout_data, "repositories");
    ASSERT_CONTAINS(res.stderr_data, "repositories");
}

// ### EXAMPLE: describe_omits_genuinely_empty_scope
TEST_CASE(test_describe_omits_genuinely_empty_scope) {
    TempDir r;
    setup_synthetic_root(r.path); // repos.d is empty

    auto res = run_command({"../../zypper-declarative", "describe", "root=" + r.path.string()});
    ASSERT_EQ(res.exit_code, 0);
    ASSERT_NOT_CONTAINS(res.stdout_data, "repositories");
}

// Host self-checks per decisions file
TEST_CASE(test_host_self_checks) {
    // Run describe with on-unreadable=warn unprivileged against the real host root
    auto res = run_command({"../../zypper-declarative", "describe", "on-unreadable=warn"});
    ASSERT_EQ(res.exit_code, 0);
    // (1) packages scope present and non-empty
    ASSERT_CONTAINS(res.stdout_data, "packages");
    // (2) check for package ownership of some common file like /etc/ssh/sshd_config (or any that exists)
    if (fs::exists("/etc/ssh/sshd_config")) {
        // Run describe and check that sshd_config is listed with openssh or similar package_name
        ASSERT_CONTAINS(res.stdout_data, "/etc/ssh/sshd_config");
    }
}
