// generated from spec: zypper-declarative.spec.md sha256:f2cc80627e483a48bb8411d297711bc5f6c6e74c28dbf0dafc8fe7bd8817251e
// tests by: claude-opus-4-8
//
// Diff and verify tests in offline two-file mode (no live system, no applied
// record). These exercise compute-intent-diff and compute-drift through the
// `diff` and `verify` verbs. Covers EXAMPLES: diff_prints_plan,
// intent_diff_yields_deletion, diff_offline_two_files,
// verify_offline_manifest_and_state, verify_offline_no_applied_record_ok,
// verify_against_external_state_dump, drift_type_transition_is_modified,
// drift_ignores_unmanaged_packaged_file.
#include "test_harness.hpp"
#include "test_fixtures.hpp"

using namespace zdtest;

// A baseline manifest (used as applied/reference) declaring two /etc files,
// plus packages and services so the structure is complete.
static std::string baseline_two_files() {
    return R"JSON({
  "meta": { "format_version": 1, "generator": "t", "created_at": "x", "desired_sha256": "abc" },
  "packages": { "_attributes": { "package_system": "rpm" },
    "_elements": [ { "name": "vim", "version": "9.0", "release": "1", "arch": "x86_64" } ] },
  "services": { "_attributes": { "init_system": "systemd" },
    "_elements": [ { "name": "sshd.service", "state": "enabled" } ] },
  "config_files": { "_attributes": null, "_elements": [
    { "name": "/etc/foo.conf", "type": "file", "mode": "0644", "user": "root",
      "group": "root", "sha256":
      "1111111111111111111111111111111111111111111111111111111111111111",
      "target": "", "content_ref": "", "package_name": "" },
    { "name": "/etc/bar.conf", "type": "file", "mode": "0644", "user": "root",
      "group": "root", "sha256":
      "2222222222222222222222222222222222222222222222222222222222222222",
      "target": "", "content_ref": "", "package_name": "" }
  ] }
})JSON";
}

// EXAMPLE: diff_prints_plan / intent_diff_yields_deletion.
// Desired adds nginx (relative to the applied record) and drops /etc/bar.conf.
// Per compute-intent-diff STEP 3, files_delete = (declared_old - declared_new)
// is computed from the APPLIED RECORD, not from a captured state file. So the
// applied baseline (two files) is provided via applied-root pointing at a
// synthetic generation root carrying usr/lib/zypper-declarative/applied.json;
// the drift portion reads the (empty) state file. The plan lists install nginx
// and delete /etc/bar.conf. Exit 0; system unmodified.
TEST(test_diff_prints_install_and_delete) {
    // desired declares vim+nginx + only /etc/foo.conf
    std::string desired = R"JSON({
  "meta": { "format_version": 1, "generator": "t", "created_at": "x", "desired_sha256": "" },
  "packages": { "_attributes": { "package_system": "rpm" },
    "_elements": [ { "name": "vim", "version": "9.0", "release": "1", "arch": "x86_64" },
                   { "name": "nginx", "version": "", "release": "", "arch": "" } ] },
  "config_files": { "_attributes": null, "_elements": [
    { "name": "/etc/foo.conf", "type": "file", "mode": "0644", "user": "root",
      "group": "root", "sha256":
      "1111111111111111111111111111111111111111111111111111111111111111",
      "target": "", "content_ref": "", "package_name": "" }
  ] }
})JSON";
    std::string mpath = write_temp_file("desired.json", desired);
    // applied record (the old declaration) declares vim + the two /etc files.
    fs::path approot = make_tmpdir("zd-applied");
    fs::path ledger_dir = approot / "usr" / "lib" / "zypper-declarative";
    std::error_code ec; fs::create_directories(ledger_dir, ec);
    {
        std::ofstream f(ledger_dir / "applied.json");
        f << baseline_two_files();
    }
    // an empty captured state for the drift portion (offline, no live read)
    std::string state = R"JSON({
  "meta": { "format_version": 1, "generator": "t", "created_at": "x", "desired_sha256": "" }
})JSON";
    std::string spath = write_temp_file("state.json", state);
    auto r = run({"diff", "manifest-path=" + mpath,
                  "applied-root=" + approot.string(), "state-path=" + spath});
    expect_eq_int(r.code, 0, "diff plan exit");
    expect_contains(r.out, "nginx", "diff lists nginx to install");
    expect_contains(r.out, "/etc/bar.conf", "diff lists /etc/bar.conf to delete");
}

// EXAMPLE: diff_offline_two_files -- plan computed purely from two files; exit 0.
TEST(test_diff_offline_two_files_exit0) {
    std::string mpath = write_temp_file("baseline.json", baseline_two_files());
    std::string spath = write_temp_file("after.json", baseline_two_files());
    auto r = run({"diff", "manifest-path=" + mpath, "state-path=" + spath});
    expect_eq_int(r.code, 0, "offline two-file diff exit");
}

// EXAMPLE: verify_offline_manifest_and_state + verify_offline_no_applied_record_ok.
// Matching state vs reference -> "system matches declaration", exit 0, and no
// "no declaration applied" message (a reference manifest was supplied).
TEST(test_verify_offline_match_exit0) {
    std::string mpath = write_temp_file("baseline.json", baseline_two_files());
    std::string spath = write_temp_file("after.json", baseline_two_files());
    auto r = run({"verify", "manifest-path=" + mpath, "state-path=" + spath});
    expect_eq_int(r.code, 0, "offline verify match exit");
    expect_contains(r.out, "system matches declaration", "verify match message");
    expect_not_contains(r.err, "no declaration applied",
                        "verify with manifest must not require applied record");
}

// EXAMPLE: verify_against_external_state_dump -- the dump diverges in one
// declared service state -> domain=units diagnostic, exit 1.
TEST(test_verify_offline_service_drift) {
    // state reports sshd disabled while reference declares it enabled
    std::string state = R"JSON({
  "meta": { "format_version": 1, "generator": "t", "created_at": "x", "desired_sha256": "" },
  "services": { "_attributes": { "init_system": "systemd" },
    "_elements": [ { "name": "sshd.service", "state": "disabled" } ] },
  "config_files": { "_attributes": null, "_elements": [
    { "name": "/etc/foo.conf", "type": "file", "mode": "0644", "user": "root",
      "group": "root", "sha256":
      "1111111111111111111111111111111111111111111111111111111111111111",
      "target": "", "content_ref": "", "package_name": "" },
    { "name": "/etc/bar.conf", "type": "file", "mode": "0644", "user": "root",
      "group": "root", "sha256":
      "2222222222222222222222222222222222222222222222222222222222222222",
      "target": "", "content_ref": "", "package_name": "" }
  ] }
})JSON";
    std::string mpath = write_temp_file("baseline.json", baseline_two_files());
    std::string spath = write_temp_file("after.json", state);
    auto r = run({"verify", "manifest-path=" + mpath, "state-path=" + spath});
    expect_eq_int(r.code, 1, "service drift exit");
    expect_contains(r.err, "units", "service drift domain=units");
    expect_contains(r.err, "sshd.service", "diagnostic names the divergent service");
}

// EXAMPLE: verify_detects_drift -- a declared file content (sha256) differs.
TEST(test_verify_offline_file_drift) {
    std::string state = R"JSON({
  "meta": { "format_version": 1, "generator": "t", "created_at": "x", "desired_sha256": "" },
  "services": { "_attributes": { "init_system": "systemd" },
    "_elements": [ { "name": "sshd.service", "state": "enabled" } ] },
  "config_files": { "_attributes": null, "_elements": [
    { "name": "/etc/foo.conf", "type": "file", "mode": "0644", "user": "root",
      "group": "root", "sha256":
      "9999999999999999999999999999999999999999999999999999999999999999",
      "target": "", "content_ref": "", "package_name": "" },
    { "name": "/etc/bar.conf", "type": "file", "mode": "0644", "user": "root",
      "group": "root", "sha256":
      "2222222222222222222222222222222222222222222222222222222222222222",
      "target": "", "content_ref": "", "package_name": "" }
  ] }
})JSON";
    std::string mpath = write_temp_file("baseline.json", baseline_two_files());
    std::string spath = write_temp_file("after.json", state);
    auto r = run({"verify", "manifest-path=" + mpath, "state-path=" + spath});
    expect_eq_int(r.code, 1, "file drift exit");
    expect_contains(r.err, "files", "file drift domain=files");
    expect_contains(r.err, "/etc/foo.conf", "diagnostic names the drifted file");
}

// EXAMPLE: drift_type_transition_is_modified -- reference declares /etc/foo as
// "file", actual reports it as "link"; it is modified by type regardless of content.
TEST(test_verify_type_transition_is_modified) {
    std::string ref = R"JSON({
  "meta": { "format_version": 1, "generator": "t", "created_at": "x", "desired_sha256": "" },
  "config_files": { "_attributes": null, "_elements": [
    { "name": "/etc/foo", "type": "file", "mode": "0644", "user": "root",
      "group": "root", "sha256":
      "1111111111111111111111111111111111111111111111111111111111111111",
      "target": "", "content_ref": "", "package_name": "" }
  ] }
})JSON";
    std::string state = R"JSON({
  "meta": { "format_version": 1, "generator": "t", "created_at": "x", "desired_sha256": "" },
  "config_files": { "_attributes": null, "_elements": [
    { "name": "/etc/foo", "type": "link", "mode": "0777", "user": "root",
      "group": "root", "sha256": "", "target": "../elsewhere", "content_ref": "",
      "package_name": "" }
  ] }
})JSON";
    std::string mpath = write_temp_file("ref.json", ref);
    std::string spath = write_temp_file("state.json", state);
    auto r = run({"verify", "manifest-path=" + mpath, "state-path=" + spath});
    expect_eq_int(r.code, 1, "type transition drift exit");
    expect_contains(r.err, "/etc/foo", "type transition names the path");
}

// EXAMPLE: drift_ignores_unmanaged_packaged_file -- a changed but package-owned
// /etc file the reference does not declare must NOT appear in files_extra
// (it is package-managed). With only that file in the actual state and an empty
// reference, verify exits 0 (no extra).
TEST(test_verify_packaged_undeclared_not_extra) {
    std::string ref = R"JSON({
  "meta": { "format_version": 1, "generator": "t", "created_at": "x", "desired_sha256": "" },
  "config_files": { "_attributes": null, "_elements": [] }
})JSON";
    std::string state = R"JSON({
  "meta": { "format_version": 1, "generator": "t", "created_at": "x", "desired_sha256": "" },
  "config_files": { "_attributes": null, "_elements": [
    { "name": "/etc/owned.conf", "type": "file", "mode": "0644", "user": "root",
      "group": "root", "sha256":
      "3333333333333333333333333333333333333333333333333333333333333333",
      "target": "", "content_ref": "", "package_name": "some-pkg" }
  ] }
})JSON";
    std::string mpath = write_temp_file("ref.json", ref);
    std::string spath = write_temp_file("state.json", state);
    auto r = run({"verify", "manifest-path=" + mpath, "state-path=" + spath});
    expect_eq_int(r.code, 0, "packaged undeclared file is not extra -> exit 0");
    expect_not_contains(r.err, "/etc/owned.conf",
                        "package-owned undeclared file must not be reported as extra");
}

// A genuinely unpackaged, undeclared /etc file IS extra -> drift, exit 1.
TEST(test_verify_unpackaged_undeclared_is_extra) {
    std::string ref = R"JSON({
  "meta": { "format_version": 1, "generator": "t", "created_at": "x", "desired_sha256": "" },
  "config_files": { "_attributes": null, "_elements": [] }
})JSON";
    std::string state = R"JSON({
  "meta": { "format_version": 1, "generator": "t", "created_at": "x", "desired_sha256": "" },
  "config_files": { "_attributes": null, "_elements": [
    { "name": "/etc/rogue.conf", "type": "file", "mode": "0644", "user": "root",
      "group": "root", "sha256":
      "4444444444444444444444444444444444444444444444444444444444444444",
      "target": "", "content_ref": "", "package_name": "" }
  ] }
})JSON";
    std::string mpath = write_temp_file("ref.json", ref);
    std::string spath = write_temp_file("state.json", state);
    auto r = run({"verify", "manifest-path=" + mpath, "state-path=" + spath});
    expect_eq_int(r.code, 1, "unpackaged undeclared file is extra -> exit 1");
    expect_contains(r.err, "/etc/rogue.conf", "diagnostic names the extra file");
}
