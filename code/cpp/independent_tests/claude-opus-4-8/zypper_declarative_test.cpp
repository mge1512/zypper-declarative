// generated from spec: zypper-declarative.spec.md sha256:aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3
// tests by: claude-opus-4-8
#include "harness.hpp"

#include <sys/stat.h>

static std::string meta_only() {
    return R"({"meta":{"format_version":1,"generator":"zypper-declarative 0.6.9","created_at":"2026-06-03T00:00:00Z","desired_sha256":""}})";
}

// --- global contract ---------------------------------------------------

TEST(version_verb) {
    auto r = run_zd({"version"});
    ASSERT_EQ(r.code, 0);
    ASSERT_CONTAINS(r.out, "zypper-declarative");
    ASSERT_CONTAINS(r.out, "spec:aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3");
}
TEST(version_flag_alias) {
    auto a = run_zd({"version"});
    auto b = run_zd({"--version"});
    ASSERT_EQ(b.code, 0);
    ASSERT_EQ(a.out, b.out);
}
TEST(help_verb) {
    auto r = run_zd({"help"});
    ASSERT_EQ(r.code, 0);
    ASSERT_CONTAINS(r.out, "usage:");
}
TEST(bare_invocation_shows_help) {
    auto r = run_zd({});
    ASSERT_EQ(r.code, 0);
    ASSERT_CONTAINS(r.out, "usage:");
}
TEST(unknown_verb_rejected) {
    auto r = run_zd({"frobnicate"});
    ASSERT_EQ(r.code, 2);
    ASSERT_CONTAINS(r.err, "usage:");
}
TEST(bad_format_value_rejected) {
    auto r = run_zd({"format=bad_value"});
    ASSERT_EQ(r.code, 2);
}

// --- diff ---------------------------------------------------------------

TEST(diff_prints_plan) {
    TempDir t;
    std::string desired = R"({"meta":{"format_version":1,"generator":"zypper-declarative 0.6.9","created_at":"","desired_sha256":""},
      "packages":{"_attributes":{"package_system":"rpm"},"_elements":[{"name":"nginx","version":"","release":"","arch":""}]},
      "config_files":{"_attributes":{},"_elements":[]}})";
    std::string actual = R"({"meta":{"format_version":1,"generator":"zypper-declarative 0.6.9","created_at":"","desired_sha256":""},
      "config_files":{"_attributes":{},"_elements":[{"name":"/etc/bar.conf","type":"file","mode":"0644","user":"root","group":"root","sha256":"0000000000000000000000000000000000000000000000000000000000000000","target":"","content_ref":"","package_name":""}]}})";
    write_text(t.path / "d.json", desired);
    write_text(t.path / "a.json", actual);
    auto r = run_zd({"diff", "manifest-path=" + (t.path / "d.json").string(),
                     "state-path=" + (t.path / "a.json").string()});
    ASSERT_EQ(r.code, 0);
    ASSERT_CONTAINS(r.out, "nginx");
}
TEST(diff_manifest_unreadable) {
    auto r = run_zd({"diff", "manifest-path=/nonexistent_xyz.json"});
    ASSERT_EQ(r.code, 2);
    ASSERT_CONTAINS(r.err, "domain=invocation");
}
TEST(diff_unchanged_no_drift) {
    TempDir t;
    std::string sf = R"({"meta":{"format_version":1,"generator":"zypper-declarative 0.6.9","created_at":"","desired_sha256":""},
      "config_files":{"_attributes":{},"_elements":[{"name":"/etc/local.conf","type":"file","mode":"0644","user":"root","group":"root","sha256":"1111111111111111111111111111111111111111111111111111111111111111","target":"","content_ref":"","package_name":""}]}})";
    write_text(t.path / "m.json", sf);
    auto r = run_zd({"diff", "manifest-path=" + (t.path / "m.json").string(),
                     "state-path=" + (t.path / "m.json").string()});
    ASSERT_EQ(r.code, 0);
    ASSERT_NOT_CONTAINS(r.out, "drift:");
}
TEST(intent_diff_yields_deletion) {
    TempDir t;
    std::string desired = R"({"meta":{"format_version":1,"generator":"zypper-declarative 0.6.9","created_at":"","desired_sha256":""},
      "config_files":{"_attributes":{},"_elements":[{"name":"/etc/foo.conf","type":"file","mode":"0644","user":"root","group":"root","sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","target":"","content_ref":"","package_name":""}]}})";
    std::string actual = R"({"meta":{"format_version":1,"generator":"zypper-declarative 0.6.9","created_at":"","desired_sha256":""},
      "config_files":{"_attributes":{},"_elements":[
        {"name":"/etc/foo.conf","type":"file","mode":"0644","user":"root","group":"root","sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","target":"","content_ref":"","package_name":""},
        {"name":"/etc/bar.conf","type":"file","mode":"0644","user":"root","group":"root","sha256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","target":"","content_ref":"","package_name":""}]}})";
    write_text(t.path / "d.json", desired);
    write_text(t.path / "a.json", actual);
    auto r = run_zd({"diff", "manifest-path=" + (t.path / "d.json").string(),
                     "state-path=" + (t.path / "a.json").string()});
    ASSERT_EQ(r.code, 0);
    ASSERT_CONTAINS(r.out, "/etc/bar.conf");
}

// --- verify -------------------------------------------------------------

TEST(verify_clean) {
    TempDir t;
    write_text(t.path / "m.json", meta_only());
    auto r = run_zd({"verify", "manifest-path=" + (t.path / "m.json").string(),
                     "state-path=" + (t.path / "m.json").string()});
    ASSERT_EQ(r.code, 0);
    ASSERT_CONTAINS(r.out, "system matches declaration");
}
TEST(verify_detects_drift_files) {
    TempDir t;
    std::string ref = R"({"meta":{"format_version":1,"generator":"zypper-declarative 0.6.9","created_at":"","desired_sha256":""},
      "config_files":{"_attributes":{},"_elements":[{"name":"/etc/foo.conf","type":"file","mode":"0644","user":"root","group":"root","sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","target":"","content_ref":"","package_name":""}]}})";
    std::string act = R"({"meta":{"format_version":1,"generator":"zypper-declarative 0.6.9","created_at":"","desired_sha256":""},
      "config_files":{"_attributes":{},"_elements":[{"name":"/etc/foo.conf","type":"file","mode":"0644","user":"root","group":"root","sha256":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","target":"","content_ref":"","package_name":""}]}})";
    write_text(t.path / "ref.json", ref);
    write_text(t.path / "act.json", act);
    auto r = run_zd({"verify", "manifest-path=" + (t.path / "ref.json").string(),
                     "state-path=" + (t.path / "act.json").string()});
    ASSERT_EQ(r.code, 1);
    ASSERT_CONTAINS(r.err, "domain=files");
    ASSERT_CONTAINS(r.err, "/etc/foo.conf");
}
TEST(verify_units_divergent) {
    TempDir t;
    std::string ref = R"({"meta":{"format_version":1,"generator":"zypper-declarative 0.6.9","created_at":"","desired_sha256":""},
      "services":{"_attributes":{"init_system":"systemd"},"_elements":[{"name":"nginx.service","state":"enabled"}]}})";
    std::string act = R"({"meta":{"format_version":1,"generator":"zypper-declarative 0.6.9","created_at":"","desired_sha256":""},
      "services":{"_attributes":{"init_system":"systemd"},"_elements":[{"name":"nginx.service","state":"disabled"}]}})";
    write_text(t.path / "ref.json", ref);
    write_text(t.path / "act.json", act);
    auto r = run_zd({"verify", "manifest-path=" + (t.path / "ref.json").string(),
                     "state-path=" + (t.path / "act.json").string()});
    ASSERT_EQ(r.code, 1);
    ASSERT_CONTAINS(r.err, "domain=services");
    ASSERT_CONTAINS(r.err, "nginx.service");
}
TEST(verify_malformed_state_dump) {
    TempDir t;
    write_text(t.path / "m.json", meta_only());
    write_text(t.path / "bad.json", "not json }");
    auto r = run_zd({"verify", "manifest-path=" + (t.path / "m.json").string(),
                     "state-path=" + (t.path / "bad.json").string()});
    ASSERT_EQ(r.code, 2);
    ASSERT_CONTAINS(r.err, "domain=invocation");
}
TEST(verify_no_applied_record_live) {
    auto r = run_zd({"verify"});
    ASSERT_EQ(r.code, 2);
    ASSERT_CONTAINS(r.err, "no declaration applied");
}
TEST(verify_offline_no_applied_record_ok) {
    TempDir t;
    write_text(t.path / "m.json", meta_only());
    auto r = run_zd({"verify", "manifest-path=" + (t.path / "m.json").string(),
                     "state-path=" + (t.path / "m.json").string()});
    ASSERT_EQ(r.code, 0);
    ASSERT_NOT_CONTAINS(r.err, "no declaration applied");
}
TEST(verify_default_scope_ignores_usr) {
    TempDir t;
    write_text(t.path / "m.json", meta_only());
    std::string act = R"({"meta":{"format_version":1,"generator":"zypper-declarative 0.6.9","created_at":"","desired_sha256":""},
      "unmanaged_files":{"_attributes":{},"_elements":[{"name":"/usr/bin/x","type":"file","mode":"0755","user":"root","group":"root","sha256":"0000000000000000000000000000000000000000000000000000000000000000"}]}})";
    write_text(t.path / "a.json", act);
    auto r = run_zd({"verify", "manifest-path=" + (t.path / "m.json").string(),
                     "state-path=" + (t.path / "a.json").string()});
    ASSERT_EQ(r.code, 0);
    ASSERT_CONTAINS(r.out, "system matches declaration");
}
TEST(verify_scope_full_unmanaged) {
    TempDir t;
    write_text(t.path / "m.json", meta_only());
    std::string act = R"({"meta":{"format_version":1,"generator":"zypper-declarative 0.6.9","created_at":"","desired_sha256":""},
      "unmanaged_files":{"_attributes":{},"_elements":[{"name":"/usr/bin/x","type":"file","mode":"0755","user":"root","group":"root","sha256":"0000000000000000000000000000000000000000000000000000000000000000"}]}})";
    write_text(t.path / "a.json", act);
    auto r = run_zd({"verify", "scope=full", "manifest-path=" + (t.path / "m.json").string(),
                     "state-path=" + (t.path / "a.json").string()});
    ASSERT_EQ(r.code, 1);
    ASSERT_CONTAINS(r.err, "/usr/bin/x");
}

// --- status -------------------------------------------------------------

TEST(status_no_declaration) {
    TempDir t;
    auto r = run_zd({"status", "applied-root=" + t.path.string()});
    ASSERT_EQ(r.code, 0);
    ASSERT_CONTAINS(r.out, "no declaration applied");
}
TEST(status_reports_generation) {
    TempDir t;
    std::string app = R"({"meta":{"format_version":1,"generator":"zypper-declarative 0.6.9","created_at":"2026-06-03T00:00:00Z","desired_sha256":"abcabcabcabcabcabcabcabcabcabcabcabcabcabcabcabcabcabcabcabcabca"},
      "packages":{"_attributes":{"package_system":"rpm"},"_elements":[{"name":"nginx","version":"1.0","release":"1","arch":"x86_64"}]}})";
    write_text(t.path / "usr/lib/zypper-declarative/applied.json", app);
    auto r = run_zd({"status", "applied-root=" + t.path.string(), "on-unreadable=warn"});
    ASSERT_EQ(r.code, 0);
    ASSERT_CONTAINS(r.out, "abcabcabcabcabcabcabcabcabcabcabcabcabcabcabcabcabcabcabcabcabca");
    ASSERT_CONTAINS(r.out, "packages");
}
TEST(status_unknown_argument) {
    auto r = run_zd({"status", "--frobnicate"});
    ASSERT_EQ(r.code, 2);
    ASSERT_CONTAINS(r.err, "usage:");
}

// --- describe (synthetic roots) ----------------------------------------

static void mkroot(const fs::path& r) {
    fs::create_directories(r / "etc/zypp/repos.d");
}

TEST(describe_emits_manifest_json) {
    TempDir r;
    mkroot(r.path);
    auto res = run_zd({"describe", "root=" + r.path.string()});
    ASSERT_EQ(res.code, 0);
    ASSERT_CONTAINS(res.out, "\"format_version\": 1");
    ASSERT_CONTAINS(res.out, "\"generator\": \"zypper-declarative 0.6.9\"");
}
TEST(describe_symlink_verbatim) {
    TempDir r;
    mkroot(r.path);
    std::error_code ec;
    fs::create_symlink("../foo/bar.conf", r.path / "etc/mylink.conf", ec);
    auto res = run_zd({"describe", "root=" + r.path.string()});
    ASSERT_EQ(res.code, 0);
    ASSERT_CONTAINS(res.out, "/etc/mylink.conf");
    ASSERT_CONTAINS(res.out, "\"type\": \"link\"");
    ASSERT_CONTAINS(res.out, "\"target\": \"../foo/bar.conf\"");
}
TEST(describe_traverses_subdirs) {
    TempDir r;
    mkroot(r.path);
    write_text(r.path / "etc/ImageMagick-7/policy.xml", "<policy/>");
    auto res = run_zd({"describe", "root=" + r.path.string()});
    ASSERT_EQ(res.code, 0);
    ASSERT_CONTAINS(res.out, "/etc/ImageMagick-7/policy.xml");
}
TEST(describe_skips_special_file) {
    TempDir r;
    mkroot(r.path);
    std::string fifo = (r.path / "etc/myfifo").string();
    mkfifo(fifo.c_str(), 0666);
    auto res = run_zd({"describe", "root=" + r.path.string()});
    ASSERT_EQ(res.code, 0);
    ASSERT_NOT_CONTAINS(res.out, "myfifo");
}
TEST(describe_bounded_to_etc) {
    TempDir r;
    mkroot(r.path);
    write_text(r.path / "etc/foo.conf", "etc");
    write_text(r.path / "usr/share/foo.conf", "usr");
    auto res = run_zd({"describe", "root=" + r.path.string()});
    ASSERT_EQ(res.code, 0);
    ASSERT_CONTAINS(res.out, "/etc/foo.conf");
    ASSERT_NOT_CONTAINS(res.out, "/usr/share/foo.conf");
}
TEST(describe_content_store) {
    TempDir r;
    mkroot(r.path);
    write_text(r.path / "etc/foo.conf", "my-secret-content");
    TempDir store;
    auto res = run_zd({"describe", "root=" + r.path.string(),
                       "content-store=" + (store.path / "store").string()});
    ASSERT_EQ(res.code, 0);
    std::string h = "de26dc64d5731ce0b28abab95ca22da94ed68d0107701125b9667fea9e93f005";
    ASSERT_CONTAINS(res.out, "sha256/" + h);
    ASSERT(fs::exists(store.path / "store" / "sha256" / h));
    ASSERT_EQ(read_text(store.path / "store" / "sha256" / h), std::string("my-secret-content"));
}
TEST(describe_without_content_store_readonly) {
    TempDir r;
    mkroot(r.path);
    write_text(r.path / "etc/foo.conf", "content");
    auto res = run_zd({"describe", "root=" + r.path.string()});
    ASSERT_EQ(res.code, 0);
    ASSERT_CONTAINS(res.out, "\"content_ref\": \"\"");
}
TEST(describe_attributes_object) {
    TempDir r;
    mkroot(r.path);
    write_text(r.path / "etc/foo.conf", "c");
    auto res = run_zd({"describe", "root=" + r.path.string()});
    ASSERT_EQ(res.code, 0);
    ASSERT_CONTAINS(res.out, "\"_attributes\": {}");
}
TEST(describe_format_yaml) {
    TempDir r;
    mkroot(r.path);
    auto res = run_zd({"describe", "root=" + r.path.string(), "format=yaml"});
    ASSERT_EQ(res.code, 0);
    ASSERT_CONTAINS(res.out, "format_version: 1");
}
TEST(describe_out_extension_yaml) {
    TempDir r;
    mkroot(r.path);
    TempDir t;
    auto res = run_zd({"describe", "root=" + r.path.string(),
                       "out=" + (t.path / "s.yaml").string()});
    ASSERT_EQ(res.code, 0);
    std::string c = read_text(t.path / "s.yaml");
    ASSERT_CONTAINS(c, "format_version: 1");
}
TEST(describe_out_extension_json) {
    TempDir r;
    mkroot(r.path);
    TempDir t;
    auto res = run_zd({"describe", "root=" + r.path.string(),
                       "out=" + (t.path / "s.json").string()});
    ASSERT_EQ(res.code, 0);
    std::string c = read_text(t.path / "s.json");
    ASSERT_CONTAINS(c, "\"format_version\": 1");
}
TEST(describe_format_overrides_extension) {
    TempDir r;
    mkroot(r.path);
    TempDir t;
    auto res = run_zd({"describe", "root=" + r.path.string(), "format=json",
                       "out=" + (t.path / "s.yaml").string()});
    ASSERT_EQ(res.code, 0);
    ASSERT_CONTAINS(read_text(t.path / "s.yaml"), "\"format_version\": 1");
}
TEST(describe_unknown_format) {
    TempDir r;
    mkroot(r.path);
    auto res = run_zd({"describe", "root=" + r.path.string(), "format=toml"});
    ASSERT_EQ(res.code, 2);
    ASSERT_CONTAINS(res.err, "usage:");
}
TEST(describe_output_unwritable) {
    TempDir r;
    mkroot(r.path);
    auto res = run_zd({"describe", "root=" + r.path.string(),
                       "out=/nonexistent_ro_dir_zzz/state.json"});
    ASSERT_EQ(res.code, 2);
    ASSERT_CONTAINS(res.err, "domain=invocation");
}
TEST(describe_repos_from_reposd) {
    TempDir r;
    mkroot(r.path);
    write_text(r.path / "etc/zypp/repos.d/repo1.repo",
               "[repo1]\nname=Repo 1\nbaseurl=http://example.com/repo1\ntype=rpm-md\nenabled=1\n");
    auto res = run_zd({"describe", "root=" + r.path.string()});
    ASSERT_EQ(res.code, 0);
    ASSERT_CONTAINS(res.out, "\"alias\": \"repo1\"");
    ASSERT_CONTAINS(res.out, "\"url\": \"http://example.com/repo1\"");
}
TEST(describe_omits_empty_scope) {
    TempDir r;
    mkroot(r.path);
    auto res = run_zd({"describe", "root=" + r.path.string()});
    ASSERT_EQ(res.code, 0);
    ASSERT_NOT_CONTAINS(res.out, "repositories");
}
TEST(describe_unreadable_scope_strict) {
    TempDir r;
    mkroot(r.path);
    auto repod = (r.path / "etc/zypp/repos.d");
    chmod(repod.string().c_str(), 0000);
    auto res = run_zd({"describe", "root=" + r.path.string()});
    chmod(repod.string().c_str(), 0755);
    ASSERT_EQ(res.code, 1);
    ASSERT_CONTAINS(res.err, "domain=repositories");
}
TEST(describe_unreadable_scope_warn) {
    TempDir r;
    mkroot(r.path);
    auto repod = (r.path / "etc/zypp/repos.d");
    chmod(repod.string().c_str(), 0000);
    auto res = run_zd({"describe", "root=" + r.path.string(), "on-unreadable=warn"});
    chmod(repod.string().c_str(), 0755);
    ASSERT_EQ(res.code, 0);
    ASSERT_CONTAINS(res.err, "repositories");
}
TEST(describe_scope_full_observational) {
    TempDir r;
    mkroot(r.path);
    write_text(r.path / "usr/bin/some_tool", "binary");
    auto res = run_zd({"describe", "root=" + r.path.string(), "scope=full"});
    ASSERT_EQ(res.code, 0);
    ASSERT_CONTAINS(res.out, "unmanaged_files");
    ASSERT_CONTAINS(res.out, "/usr/bin/some_tool");
}
TEST(describe_bootstraps_desired) {
    TempDir r;
    mkroot(r.path);
    write_text(r.path / "etc/foo.conf", "hello");
    TempDir t;
    auto mf = (t.path / "desired.json").string();
    auto res = run_zd({"describe", "root=" + r.path.string(), "out=" + mf});
    ASSERT_EQ(res.code, 0);
    auto d = run_zd({"diff", "manifest-path=" + mf, "state-path=" + mf});
    ASSERT_EQ(d.code, 0);
    ASSERT_NOT_CONTAINS(d.out, "drift:");
}

// --- apply --------------------------------------------------------------

TEST(apply_manifest_invalid) {
    TempDir t;
    write_text(t.path / "m.json",
               R"({"meta":{"format_version":2,"generator":"zypper-declarative 0.6.9","created_at":"","desired_sha256":""}})");
    auto r = run_zd({"apply", "manifest-path=" + (t.path / "m.json").string()});
    ASSERT_EQ(r.code, 1);
    ASSERT_CONTAINS(r.err, "domain=manifest");
}
TEST(apply_manifest_unreadable) {
    auto r = run_zd({"apply", "manifest-path=/nonexistent_qq.json"});
    ASSERT_EQ(r.code, 2);
    ASSERT_CONTAINS(r.err, "domain=invocation");
}
TEST(apply_transaction_unavailable) {
    auto r = run_zd({"apply", "mode=external"});
    ASSERT_EQ(r.code, 2);
    ASSERT_CONTAINS(r.err, "domain=transaction");
}
TEST(apply_rejects_full_describe_dump) {
    TempDir t;
    write_text(t.path / "m.json",
               R"({"meta":{"format_version":1,"generator":"zypper-declarative 0.6.9","created_at":"","desired_sha256":""},
                  "unmanaged_files":{"_attributes":{},"_elements":[{"name":"/usr/bin/x","type":"file","mode":"0755","user":"root","group":"root","sha256":"0000000000000000000000000000000000000000000000000000000000000000"}]}})");
    auto r = run_zd({"apply", "manifest-path=" + (t.path / "m.json").string()});
    ASSERT_EQ(r.code, 1);
    ASSERT_CONTAINS(r.err, "domain=manifest");
}

// --- yaml ---------------------------------------------------------------

TEST(yaml_manifest_accepted) {
    TempDir t;
    std::string y = "meta:\n  format_version: 1\n  generator: \"zypper-declarative 0.6.9\"\n  created_at: \"\"\n  desired_sha256: \"\"\n";
    write_text(t.path / "d.yaml", y);
    write_text(t.path / "s.json", meta_only());
    auto r = run_zd({"diff", "manifest-path=" + (t.path / "d.yaml").string(),
                     "state-path=" + (t.path / "s.json").string()});
    ASSERT_EQ(r.code, 0);
}
TEST(yaml_unsafe_rejected) {
    TempDir t;
    std::string y = "meta:\n  format_version: 1\n  generator: \"zypper-declarative 0.6.9\"\n  created_at: \"\"\n  desired_sha256: \"\"\n  bad: !!sh/test \"id\"\n";
    write_text(t.path / "evil.yaml", y);
    auto r = run_zd({"diff", "manifest-path=" + (t.path / "evil.yaml").string()});
    ASSERT_EQ(r.code, 1);
    ASSERT_CONTAINS(r.err, "domain=manifest");
}
TEST(verify_state_path_extension_yaml) {
    TempDir t;
    write_text(t.path / "m.json", meta_only());
    std::string y = "meta:\n  format_version: 1\n  generator: \"zypper-declarative 0.6.9\"\n  created_at: \"\"\n  desired_sha256: \"\"\n";
    write_text(t.path / "s.yaml", y);
    auto r = run_zd({"verify", "manifest-path=" + (t.path / "m.json").string(),
                     "state-path=" + (t.path / "s.yaml").string()});
    ASSERT_EQ(r.code, 0);
    ASSERT_CONTAINS(r.out, "system matches declaration");
}

// --- drift model --------------------------------------------------------

TEST(drift_type_transition_modified) {
    TempDir t;
    std::string ref = R"({"meta":{"format_version":1,"generator":"zypper-declarative 0.6.9","created_at":"","desired_sha256":""},
      "config_files":{"_attributes":{},"_elements":[{"name":"/etc/foo","type":"file","mode":"0644","user":"root","group":"root","sha256":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","target":"","content_ref":"","package_name":""}]}})";
    std::string act = R"({"meta":{"format_version":1,"generator":"zypper-declarative 0.6.9","created_at":"","desired_sha256":""},
      "config_files":{"_attributes":{},"_elements":[{"name":"/etc/foo","type":"link","mode":"0777","user":"root","group":"root","sha256":"","target":"/x","content_ref":"","package_name":""}]}})";
    write_text(t.path / "ref.json", ref);
    write_text(t.path / "act.json", act);
    auto r = run_zd({"verify", "manifest-path=" + (t.path / "ref.json").string(),
                     "state-path=" + (t.path / "act.json").string()});
    ASSERT_EQ(r.code, 1);
    ASSERT_CONTAINS(r.err, "domain=files");
}
TEST(drift_ignores_unmanaged_packaged_file) {
    TempDir t;
    std::string ref = R"({"meta":{"format_version":1,"generator":"zypper-declarative 0.6.9","created_at":"","desired_sha256":""},
      "config_files":{"_attributes":{},"_elements":[]}})";
    std::string act = R"({"meta":{"format_version":1,"generator":"zypper-declarative 0.6.9","created_at":"","desired_sha256":""},
      "config_files":{"_attributes":{},"_elements":[{"name":"/etc/packaged.conf","type":"file","mode":"0644","user":"root","group":"root","sha256":"0000000000000000000000000000000000000000000000000000000000000000","target":"","content_ref":"","package_name":"some-package"}]}})";
    write_text(t.path / "ref.json", ref);
    write_text(t.path / "act.json", act);
    auto r = run_zd({"verify", "manifest-path=" + (t.path / "ref.json").string(),
                     "state-path=" + (t.path / "act.json").string()});
    ASSERT_EQ(r.code, 0);
    ASSERT_CONTAINS(r.out, "system matches declaration");
}
TEST(packages_wildcard_no_false_drift) {
    TempDir t;
    std::string ref = R"({"meta":{"format_version":1,"generator":"zypper-declarative 0.6.9","created_at":"","desired_sha256":""},
      "packages":{"_attributes":{"package_system":"rpm"},"_elements":[{"name":"nginx","version":"","release":"","arch":""}]}})";
    std::string act = R"({"meta":{"format_version":1,"generator":"zypper-declarative 0.6.9","created_at":"","desired_sha256":""},
      "packages":{"_attributes":{"package_system":"rpm"},"_elements":[{"name":"nginx","version":"1.27.4","release":"1","arch":"x86_64"}]}})";
    write_text(t.path / "ref.json", ref);
    write_text(t.path / "act.json", act);
    auto r = run_zd({"verify", "manifest-path=" + (t.path / "ref.json").string(),
                     "state-path=" + (t.path / "act.json").string()});
    ASSERT_EQ(r.code, 0);
    ASSERT_CONTAINS(r.out, "system matches declaration");
}

// --- host self-check (libzypp read) ------------------------------------

TEST(host_describe_packages_nonempty) {
    if (!fs::exists("/usr/lib/sysimage/rpm") && !fs::exists("/var/lib/rpm")) {
        SKIP("no rpmdb on this host");
    }
    auto r = run_zd({"describe", "on-unreadable=warn"});
    ASSERT_EQ(r.code, 0);
    ASSERT_CONTAINS(r.out, "packages");
}
TEST(host_idempotence_describe_then_diff) {
    if (!fs::exists("/usr/lib/sysimage/rpm") && !fs::exists("/var/lib/rpm")) {
        SKIP("no rpmdb on this host");
    }
    TempDir t;
    auto m = (t.path / "m.json").string();
    auto d = run_zd({"describe", "out=" + m, "on-unreadable=warn"});
    ASSERT_EQ(d.code, 0);
    auto df = run_zd({"diff", "manifest-path=" + m, "on-unreadable=warn"});
    ASSERT_EQ(df.code, 0);
    ASSERT_NOT_CONTAINS(df.out, "  extra ");
}
