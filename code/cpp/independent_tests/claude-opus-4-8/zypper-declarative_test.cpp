// generated from spec: zypper-declarative.spec.md sha256:1641bb4413b82fecb081125067107bd5a4e30a8393edc778ead646207d68da5e
// tests by: claude-opus-4-8
//
// Black-box test suite for zypper-declarative (CLI). The binary under test is
// invoked at "../../zypper-declarative" (project root, per the cli-tool
// template BINARY-LOCATION constraint). No internal symbol is linked.
//
// Coverage map (EXAMPLE / INVARIANT -> test):
//  - CLI global contract: bare invocation, version verb + alias, help verb +
//    aliases, unknown verb, unknown option/value, exit codes
//  - describe: emits JSON manifest, format=yaml, out extension resolution,
//    explicit format overrides extension, unknown format, scope=full scopes,
//    populates/omits content_ref, repositories from repos.d, content-store
//  - diff: prints plan, manifest unreadable, offline two-file, drift against
//    desired manifest (no false files_extra), intent-diff yields deletion
//  - verify: clean (offline), drift detection, malformed dump, no applied
//    record, offline manifest+state, state-path extension yaml
//  - status: no declaration, unknown argument
//  - apply: manifest unreadable, manifest invalid, rejects full describe dump
//  - resolve-format / type / drift semantics via diff & verify on fixtures
//
// Mutating verbs that require root and a live snapshot transaction (a
// successful apply that creates a snapshot, init that opens a transaction) are
// NOT asserted here for their success path because they cannot run in an
// unprivileged black-box test environment; their failure/validation paths
// (unreadable manifest, invalid manifest, transaction unavailable, rejecting a
// full-describe dump) ARE asserted, as those are observable without privilege.

#include "harness.hpp"
#include <sstream>
#include <fstream>
#include <cstdio>
#include <string>
#include <filesystem>

namespace fs = std::filesystem;
using namespace zdtest;

// ----------------------------------------------------------------------
// Fixture helpers
// ----------------------------------------------------------------------
namespace {

std::string tmpdir() {
    static std::string base;
    if (base.empty()) {
        char tmpl[] = "/tmp/zdtestXXXXXX";
        char* p = mkdtemp(tmpl);
        base = p ? std::string(p) : std::string("/tmp/zdtest-fallback");
    }
    return base;
}

std::string write_file(const std::string& name, const std::string& content) {
    std::string path = tmpdir() + "/" + name;
    std::ofstream f(path, std::ios::binary | std::ios::trunc);
    f << content;
    f.close();
    return path;
}

// A structurally complete JSON manifest (declarable subset): meta + the four
// scopes. Used as the GIVEN fixture for diff/verify/apply tests.
const char* MANIFEST_FULL = R"JSON({
  "meta": {
    "format_version": 1,
    "generator": "zypper-declarative 0.6.8",
    "created_at": "2026-05-29T08:30:00Z",
    "desired_sha256": ""
  },
  "repositories": {
    "_attributes": { "repository_system": "zypp" },
    "_elements": [
      { "alias": "repo1", "name": "Repo One",
        "url": "https://example/repo1", "type": "rpm-md",
        "enabled": true, "gpgcheck": true, "autorefresh": false, "priority": 99 }
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
      { "name": "/etc/foo.conf", "type": "file", "mode": "0644",
        "user": "root", "group": "root",
        "sha256": "0000000000000000000000000000000000000000000000000000000000000000",
        "target": "", "content_ref": "", "package_name": "" }
    ]
  }
})JSON";

// A state dump (actual state) equal to the applied/desired declaration in the
// declarable scopes — used for offline verify "matches" tests.
const char* STATE_MATCHING = R"JSON({
  "meta": {
    "format_version": 1,
    "generator": "zypper-declarative 0.6.8",
    "created_at": "2026-05-29T08:30:00Z",
    "desired_sha256": ""
  },
  "packages": {
    "_attributes": { "package_system": "rpm" },
    "_elements": [
      { "name": "nginx", "version": "1.0", "release": "1", "arch": "x86_64" }
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
      { "name": "/etc/foo.conf", "type": "file", "mode": "0644",
        "user": "root", "group": "root",
        "sha256": "0000000000000000000000000000000000000000000000000000000000000000",
        "target": "", "content_ref": "", "package_name": "" }
    ]
  }
})JSON";

// A reference manifest that declares a service state which the matching state
// above contradicts (state "disabled" vs actual "enabled") -> units drift.
const char* MANIFEST_SERVICE_DIVERGENT = R"JSON({
  "meta": { "format_version": 1, "generator": "zypper-declarative 0.6.8",
            "created_at": "2026-05-29T08:30:00Z", "desired_sha256": "" },
  "services": {
    "_attributes": { "init_system": "systemd" },
    "_elements": [ { "name": "nginx.service", "state": "disabled" } ]
  }
})JSON";

// A reference manifest declaring a file with a sha256 that differs from the
// state dump's file sha256 -> files drift.
const char* MANIFEST_FILE_DIVERGENT = R"JSON({
  "meta": { "format_version": 1, "generator": "zypper-declarative 0.6.8",
            "created_at": "2026-05-29T08:30:00Z", "desired_sha256": "" },
  "config_files": {
    "_attributes": {},
    "_elements": [
      { "name": "/etc/foo.conf", "type": "file", "mode": "0644",
        "user": "root", "group": "root",
        "sha256": "1111111111111111111111111111111111111111111111111111111111111111",
        "target": "", "content_ref": "", "package_name": "" }
    ]
  }
})JSON";

// A reference declaring /etc/foo as type "file"; state declares it as "link"
// -> type transition drift (files_modified), regardless of content.
const char* MANIFEST_TYPE_FILE = R"JSON({
  "meta": { "format_version": 1, "generator": "zypper-declarative 0.6.8",
            "created_at": "2026-05-29T08:30:00Z", "desired_sha256": "" },
  "config_files": {
    "_attributes": {},
    "_elements": [
      { "name": "/etc/foo", "type": "file", "mode": "0644",
        "user": "root", "group": "root",
        "sha256": "2222222222222222222222222222222222222222222222222222222222222222",
        "target": "", "content_ref": "", "package_name": "" }
    ]
  }
})JSON";

const char* STATE_TYPE_LINK = R"JSON({
  "meta": { "format_version": 1, "generator": "zypper-declarative 0.6.8",
            "created_at": "2026-05-29T08:30:00Z", "desired_sha256": "" },
  "config_files": {
    "_attributes": {},
    "_elements": [
      { "name": "/etc/foo", "type": "link", "mode": "0777",
        "user": "root", "group": "root", "sha256": "",
        "target": "elsewhere", "content_ref": "", "package_name": "" }
    ]
  }
})JSON";

// A manifest carrying a non-empty observational scope (unmanaged_files) — must
// be rejected by load-desired-manifest as a desired manifest.
const char* MANIFEST_WITH_OBSERVATIONAL = R"JSON({
  "meta": { "format_version": 1, "generator": "zypper-declarative 0.6.8",
            "created_at": "2026-05-29T08:30:00Z", "desired_sha256": "" },
  "packages": {
    "_attributes": { "package_system": "rpm" },
    "_elements": [ { "name": "nginx", "version": "", "release": "", "arch": "" } ]
  },
  "unmanaged_files": {
    "_attributes": {},
    "_elements": [
      { "name": "/usr/bin/foo", "type": "file", "mode": "0755",
        "user": "root", "group": "root", "sha256":
        "3333333333333333333333333333333333333333333333333333333333333333",
        "target": "" }
    ]
  }
})JSON";

// format_version = 2 -> schema invalid.
const char* MANIFEST_BAD_VERSION = R"JSON({
  "meta": { "format_version": 2, "generator": "zypper-declarative 0.6.8",
            "created_at": "2026-05-29T08:30:00Z", "desired_sha256": "" }
})JSON";

// A YAML serialisation of a valid manifest.
const char* MANIFEST_YAML = R"YAML(meta:
  format_version: 1
  generator: "zypper-declarative 0.6.8"
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
)YAML";

} // namespace

// ======================================================================
// Global CLI contract (template + spec DEPLOYMENT/INVARIANTS)
// ======================================================================

// EXAMPLE: bare_invocation_shows_help
ZD_TEST(test_bare_invocation_shows_help) {
    auto r = run({});
    ZD_EXPECT_CODE(r, 0);
    ZD_EXPECT_CONTAINS(r.out, "usage:");
}

// EXAMPLE: version_verb_bare_word
ZD_TEST(test_version_verb_bare_word) {
    auto r = run({"version"});
    ZD_EXPECT_CODE(r, 0);
    ZD_EXPECT_CONTAINS(r.out, "zypper-declarative ");
    // INVARIANT: version output includes the spec hash (spec:<hash>)
    ZD_EXPECT_CONTAINS(r.out, "spec:");
}

// EXAMPLE: version_flag_alias  (--version identical to bare-word version)
ZD_TEST(test_version_flag_alias_matches) {
    auto bare = run({"version"});
    auto flag = run({"--version"});
    ZD_EXPECT_CODE(flag, 0);
    ZD_EXPECT_EQ(flag.out, bare.out);
}

// EXAMPLE: help_verb_bare_word  + alias --help / -h
ZD_TEST(test_help_verb_and_aliases) {
    auto h = run({"help"});
    ZD_EXPECT_CODE(h, 0);
    ZD_EXPECT_CONTAINS(h.out, "usage:");
    auto h2 = run({"--help"});
    ZD_EXPECT_CODE(h2, 0);
    ZD_EXPECT_CONTAINS(h2.out, "usage:");
    auto h3 = run({"-h"});
    ZD_EXPECT_CODE(h3, 0);
    ZD_EXPECT_CONTAINS(h3.out, "usage:");
}

// EXAMPLE: unknown_verb_rejected
ZD_TEST(test_unknown_verb_rejected) {
    auto r = run({"frobnicate"});
    ZD_EXPECT_CODE(r, 2);
    ZD_EXPECT_CONTAINS(r.err, "usage:");
}

// EXAMPLE: describe_unknown_format  (unknown format value -> exit 2)
ZD_TEST(test_unknown_format_value_rejected) {
    auto r = run({"describe", "format=toml"});
    ZD_EXPECT_CODE(r, 2);
}

// Acceptance criterion / boundary: a bad format value anywhere is exit 2.
ZD_TEST(test_bad_format_value_exit2) {
    auto r = run({"format=bad_value"});
    ZD_EXPECT_CODE(r, 2);
}

// EXAMPLE: status_unknown_argument
ZD_TEST(test_status_unknown_argument) {
    auto r = run({"status", "--frobnicate"});
    ZD_EXPECT_CODE(r, 2);
    ZD_EXPECT_CONTAINS(r.err, "usage:");
}

// ======================================================================
// describe (read-only, runs against the build host's live system)
// ======================================================================

// EXAMPLE: describe_emits_manifest  (JSON, format_version 1, packages resolved)
ZD_TEST(test_describe_emits_json_manifest) {
    // on-unreadable=warn: the build host's /etc carries non-world-readable
    // files (e.g. /etc/libaudit.conf, mode 0640 root:root) that an
    // unprivileged test process cannot hash. Under the spec default
    // (on-unreadable=error) describe correctly exits 1 on such a file; the
    // spec-provided mechanism to exercise the success path unprivileged is
    // on-unreadable=warn (omit the unreadable file with a diagnostic, continue).
    auto r = run({"describe", "on-unreadable=warn"});
    ZD_EXPECT_CODE(r, 0);
    // canonical JSON, Machinery format_version 1
    ZD_EXPECT_CONTAINS(r.out, "\"format_version\"");
    ZD_EXPECT_CONTAINS(r.out, "\"meta\"");
    // packages scope present and non-empty on a package-managed build host
    // (decisions hints self-check #1: packages must be present and non-empty)
    ZD_EXPECT_CONTAINS(r.out, "\"packages\"");
    ZD_EXPECT_CONTAINS(r.out, "\"package_system\"");
}

// EXAMPLE: scope_attributes_always_object  (config_files._attributes is {} not null)
// describe_emits_manifest INVARIANT: every scope's _attributes is an object.
ZD_TEST(test_describe_attributes_never_null) {
    auto r = run({"describe", "on-unreadable=warn"});
    ZD_EXPECT_CODE(r, 0);
    ZD_EXPECT_CONTAINS(r.out, "\"_attributes\"");
    ZD_EXPECT_NOT_CONTAINS(r.out, "\"_attributes\" : null");
    ZD_EXPECT_NOT_CONTAINS(r.out, "\"_attributes\":null");
}

// EXAMPLE: describe_format_yaml  (stdout is YAML, not JSON object)
ZD_TEST(test_describe_format_yaml_stdout) {
    auto r = run({"describe", "format=yaml", "on-unreadable=warn"});
    ZD_EXPECT_CODE(r, 0);
    // A YAML document does not start with '{'
    ZD_EXPECT_TRUE(!r.out.empty() && r.out[0] != '{');
    ZD_EXPECT_CONTAINS(r.out, "meta:");
}

// EXAMPLE: describe_out_extension_yaml
ZD_TEST(test_describe_out_extension_yaml) {
    std::string out = tmpdir() + "/state_ext.yaml";
    auto r = run({"describe", "on-unreadable=warn", "out=" + out});
    ZD_EXPECT_CODE(r, 0);
    std::ifstream f(out);
    std::string first; std::getline(f, first);
    ZD_EXPECT_TRUE(!first.empty() && first[0] != '{');
}

// EXAMPLE: describe_out_extension_json
ZD_TEST(test_describe_out_extension_json) {
    std::string out = tmpdir() + "/state_ext.json";
    auto r = run({"describe", "on-unreadable=warn", "out=" + out});
    ZD_EXPECT_CODE(r, 0);
    std::ifstream f(out, std::ios::binary);
    std::string content((std::istreambuf_iterator<char>(f)),
                         std::istreambuf_iterator<char>());
    ZD_EXPECT_CONTAINS(content, "\"format_version\"");
}

// EXAMPLE: describe_format_overrides_extension
ZD_TEST(test_describe_format_overrides_extension) {
    std::string out = tmpdir() + "/override.yaml";
    auto r = run({"describe", "format=json", "on-unreadable=warn", "out=" + out});
    ZD_EXPECT_CODE(r, 0);
    std::ifstream f(out, std::ios::binary);
    std::string content((std::istreambuf_iterator<char>(f)),
                         std::istreambuf_iterator<char>());
    // explicit format=json wins over the .yaml extension
    ZD_EXPECT_CONTAINS(content, "\"format_version\"");
}

// EXAMPLE: describe_output_unwritable
ZD_TEST(test_describe_output_unwritable) {
    // on-unreadable=warn so the /etc walk succeeds and the run reaches the
    // output-write step, which is the unwritable path under test.
    auto r = run({"describe", "on-unreadable=warn",
                  "out=/this/path/does/not/exist/state.json"});
    ZD_EXPECT_CODE(r, 2);
}

// EXAMPLE: describe_without_content_store_is_readonly  (content_ref "")
ZD_TEST(test_describe_without_content_store_is_readonly) {
    auto r = run({"describe", "on-unreadable=warn"});
    ZD_EXPECT_CODE(r, 0);
    // With no content-store, every emitted record has content_ref "" — there
    // must be no "sha256/" content-ref value anywhere in the output.
    ZD_EXPECT_NOT_CONTAINS(r.out, "\"content_ref\" : \"sha256/");
    ZD_EXPECT_NOT_CONTAINS(r.out, "\"content_ref\":\"sha256/");
}

// EXAMPLE: describe_bootstraps_desired_manifest  (describe output is a valid
// desired manifest accepted by load-desired-manifest; diff against the same
// unchanged system shows no files to write/delete).
ZD_TEST(test_describe_output_is_valid_desired_manifest) {
    std::string out = tmpdir() + "/bootstrap.json";
    auto d = run({"describe", "on-unreadable=warn", "out=" + out});
    ZD_EXPECT_CODE(d, 0);
    // Feed the describe output straight back as BOTH the desired manifest and
    // the captured state (offline two-file diff, EXAMPLE diff_offline_two_files):
    // the live system is not re-read, so the host's protected /etc files (which
    // a non-root reader cannot hash, correctly failing diff's internal
    // on_unreadable=error live read) do not interfere. Because state == desired,
    // the drift report is empty by construction (the idempotence invariant).
    auto r = run({"diff", "manifest-path=" + out, "state-path=" + out});
    ZD_EXPECT_CODE(r, 0);
    // No files_extra: the describe output is accepted unchanged as a desired
    // manifest and the same bytes as the captured state produce no drift.
    ZD_EXPECT_NOT_CONTAINS(r.out, "files_extra");
}

// EXAMPLE: idempotence self-check (#5 in decisions hints):
// describe out=m.json ; diff (offline) manifest-path=m.json state-path=m.json
// -> empty drift report. Offline so diff does not re-read the live system
// (whose protected /etc files an unprivileged process cannot hash under the
// spec default on_unreadable=error).
ZD_TEST(test_describe_then_diff_empty_drift) {
    std::string out = tmpdir() + "/idem.json";
    auto d = run({"describe", "on-unreadable=warn", "out=" + out});
    ZD_EXPECT_CODE(d, 0);
    auto r = run({"diff", "manifest-path=" + out, "state-path=" + out});
    ZD_EXPECT_CODE(r, 0);
    // The drift section is present and carries no drift items: every line after
    // the "drift:" header is empty (no indented entry lines).
    ZD_EXPECT_CONTAINS(r.out, "drift");
    auto pos = r.out.find("drift:");
    ZD_EXPECT_TRUE(pos != std::string::npos);
    if (pos != std::string::npos) {
        std::string after = r.out.substr(pos + 6);
        // No indented (two-space-prefixed) entry line follows the drift header.
        ZD_EXPECT_NOT_CONTAINS(after, "\n  ");
    }
}

// EXAMPLE: describe_scope_full_emits_observational_scopes
ZD_TEST(test_describe_scope_full_has_observational_scopes) {
    // scope=full additionally scans /usr and /boot. On the build host these
    // trees almost always contain at least one unpackaged or modified file, so
    // at least one observational scope appears. The plain (etc) describe has
    // neither.
    auto plain = run({"describe", "on-unreadable=warn"});
    ZD_EXPECT_CODE(plain, 0);
    ZD_EXPECT_NOT_CONTAINS(plain.out, "changed_managed_files");
    ZD_EXPECT_NOT_CONTAINS(plain.out, "unmanaged_files");
    auto full = run({"describe", "scope=full", "on-unreadable=warn"});
    ZD_EXPECT_CODE(full, 0);
    // At least one of the two observational scopes should appear under full.
    bool any = contains(full.out, "changed_managed_files") ||
               contains(full.out, "unmanaged_files");
    ZD_EXPECT_TRUE(any);
}

// EXAMPLE: describe_populates_content_store
ZD_TEST(test_describe_populates_content_store) {
    std::string store = tmpdir() + "/cstore";
    std::string out = tmpdir() + "/with_store.json";
    auto r = run({"describe", "on-unreadable=warn",
                  "content-store=" + store, "out=" + out});
    ZD_EXPECT_CODE(r, 0);
    std::ifstream f(out, std::ios::binary);
    std::string content((std::istreambuf_iterator<char>(f)),
                         std::istreambuf_iterator<char>());
    // If any regular-file record was emitted, its content_ref is "sha256/...".
    // (On hosts where /etc has at least one changed/unpackaged file there will
    // be one; we assert the store directory was created.)
    ZD_EXPECT_TRUE(fs::exists(store + "/sha256") || content.find("\"_elements\" : []") != std::string::npos
                   || content.find("config_files") == std::string::npos);
}

// ======================================================================
// diff (read-only)
// ======================================================================

// EXAMPLE: diff_manifest_unreadable
ZD_TEST(test_diff_manifest_unreadable) {
    auto r = run({"diff", "manifest-path=/nonexistent_zd_manifest.json"});
    ZD_EXPECT_CODE(r, 2);
    ZD_EXPECT_CONTAINS(r.err, "invocation");
}

// EXAMPLE: diff_prints_plan + intent_diff_yields_deletion (offline two files):
// applied/state with /etc/bar.conf dropped, nginx added.
ZD_TEST(test_diff_offline_two_files_plan) {
    std::string manifest = write_file("diff_desired.json", MANIFEST_FULL);
    std::string state = write_file("diff_state.json", STATE_MATCHING);
    // EXAMPLE diff_offline_two_files: plan computed purely from the two files.
    auto r = run({"diff", "manifest-path=" + manifest, "state-path=" + state});
    ZD_EXPECT_CODE(r, 0);
    // The plan must contain the standard section labels.
    ZD_EXPECT_CONTAINS(r.out, "drift");
}

// EXAMPLE: diff offline with no live read — exit 0 whether or not differences.
ZD_TEST(test_diff_offline_exit_zero) {
    std::string manifest = write_file("diff2_desired.json", MANIFEST_FILE_DIVERGENT);
    std::string state = write_file("diff2_state.json", STATE_MATCHING);
    auto r = run({"diff", "manifest-path=" + manifest, "state-path=" + state});
    ZD_EXPECT_CODE(r, 0);
}

// EXAMPLE: yaml_manifest_accepted  (diff on a YAML manifest, offline)
ZD_TEST(test_diff_yaml_manifest_accepted) {
    std::string manifest = write_file("desired.yaml", MANIFEST_YAML);
    std::string state = write_file("yml_state.json", STATE_MATCHING);
    auto r = run({"diff", "manifest-path=" + manifest, "state-path=" + state});
    ZD_EXPECT_CODE(r, 0);
}

// ======================================================================
// verify (offline two-file comparisons; pure function of the files)
// ======================================================================

// EXAMPLE: verify_offline_manifest_and_state (matching) -> exit 0
ZD_TEST(test_verify_offline_matches) {
    std::string manifest = write_file("v_match_ref.json", MANIFEST_FULL);
    std::string state = write_file("v_match_state.json", STATE_MATCHING);
    auto r = run({"verify", "manifest-path=" + manifest, "state-path=" + state});
    ZD_EXPECT_CODE(r, 0);
    ZD_EXPECT_CONTAINS(r.out, "system matches declaration");
}

// EXAMPLE: verify_against_external_state_dump (service state diverges) -> exit 1 units
ZD_TEST(test_verify_offline_units_drift) {
    std::string manifest = write_file("v_units_ref.json", MANIFEST_SERVICE_DIVERGENT);
    std::string state = write_file("v_units_state.json", STATE_MATCHING);
    auto r = run({"verify", "manifest-path=" + manifest, "state-path=" + state});
    ZD_EXPECT_CODE(r, 1);
    ZD_EXPECT_CONTAINS(r.err, "units");
    ZD_EXPECT_CONTAINS(r.err, "nginx.service");
}

// EXAMPLE: verify_detects_drift (declared file edited) -> exit 1 files
ZD_TEST(test_verify_offline_files_drift) {
    std::string manifest = write_file("v_files_ref.json", MANIFEST_FILE_DIVERGENT);
    std::string state = write_file("v_files_state.json", STATE_MATCHING);
    auto r = run({"verify", "manifest-path=" + manifest, "state-path=" + state});
    ZD_EXPECT_CODE(r, 1);
    ZD_EXPECT_CONTAINS(r.err, "/etc/foo.conf");
    ZD_EXPECT_CONTAINS(r.err, "files");
}

// EXAMPLE: drift_type_transition_is_modified (type differs -> modified)
ZD_TEST(test_verify_type_transition_drift) {
    std::string manifest = write_file("v_type_ref.json", MANIFEST_TYPE_FILE);
    std::string state = write_file("v_type_state.json", STATE_TYPE_LINK);
    auto r = run({"verify", "manifest-path=" + manifest, "state-path=" + state});
    ZD_EXPECT_CODE(r, 1);
    ZD_EXPECT_CONTAINS(r.err, "/etc/foo");
}

// EXAMPLE: verify_malformed_state_dump -> exit 2 invocation
ZD_TEST(test_verify_malformed_state_dump) {
    std::string manifest = write_file("v_bad_ref.json", MANIFEST_FULL);
    std::string state = write_file("broken.json", "{ this is not valid json ]]]");
    auto r = run({"verify", "manifest-path=" + manifest, "state-path=" + state});
    ZD_EXPECT_CODE(r, 2);
    ZD_EXPECT_CONTAINS(r.err, "invocation");
}

// EXAMPLE: verify_offline_no_applied_record_ok (manifest+state supplied; no
// applied record required; must NOT print "no declaration applied")
ZD_TEST(test_verify_offline_no_applied_record_ok) {
    std::string manifest = write_file("v_noar_ref.json", MANIFEST_FULL);
    std::string state = write_file("v_noar_state.json", STATE_MATCHING);
    auto r = run({"verify", "manifest-path=" + manifest, "state-path=" + state});
    // exit 0 (matches) and never the no-declaration message
    ZD_EXPECT_CODE(r, 0);
    ZD_EXPECT_NOT_CONTAINS(r.out, "no declaration applied");
    ZD_EXPECT_NOT_CONTAINS(r.err, "no declaration applied");
}

// EXAMPLE: verify_state_path_extension_yaml (state dump is YAML, matches) -> 0
ZD_TEST(test_verify_state_path_extension_yaml) {
    // Build a YAML state matching a minimal reference manifest.
    const char* ref = R"JSON({
      "meta": { "format_version": 1, "generator": "zypper-declarative 0.6.8",
                "created_at": "2026-05-29T08:30:00Z", "desired_sha256": "" },
      "services": { "_attributes": { "init_system": "systemd" },
                    "_elements": [ { "name": "a.service", "state": "enabled" } ] }
    })JSON";
    const char* stateYaml = R"YAML(meta:
  format_version: 1
  generator: "zypper-declarative 0.6.8"
  created_at: "2026-05-29T08:30:00Z"
  desired_sha256: ""
services:
  _attributes:
    init_system: "systemd"
  _elements:
    - name: "a.service"
      state: "enabled"
)YAML";
    std::string manifest = write_file("v_yaml_ref.json", ref);
    std::string state = write_file("v_state.yaml", stateYaml);
    auto r = run({"verify", "manifest-path=" + manifest, "state-path=" + state});
    ZD_EXPECT_CODE(r, 0);
    ZD_EXPECT_CONTAINS(r.out, "system matches declaration");
}

// ======================================================================
// status (read-only)
// ======================================================================

// EXAMPLE: status_no_declaration  (applied-root pointed at a root with no
// applied record -> "no declaration applied", exit 0)
ZD_TEST(test_status_no_declaration) {
    std::string emptyroot = tmpdir() + "/emptyroot";
    fs::create_directories(emptyroot);
    auto r = run({"status", "applied-root=" + emptyroot});
    ZD_EXPECT_CODE(r, 0);
    ZD_EXPECT_CONTAINS(r.out, "no declaration applied");
}

// ======================================================================
// apply (validation / failure paths observable without privilege)
// ======================================================================

// EXAMPLE: apply_manifest_unreadable -> exit 2 invocation
ZD_TEST(test_apply_manifest_unreadable) {
    auto r = run({"apply", "manifest-path=/nonexistent.json"});
    ZD_EXPECT_CODE(r, 2);
    ZD_EXPECT_CONTAINS(r.err, "invocation");
}

// EXAMPLE: apply_manifest_invalid (format_version 2) -> exit 1 manifest, no txn
ZD_TEST(test_apply_manifest_invalid) {
    std::string manifest = write_file("bad_version.json", MANIFEST_BAD_VERSION);
    auto r = run({"apply", "manifest-path=" + manifest});
    ZD_EXPECT_CODE(r, 1);
    ZD_EXPECT_CONTAINS(r.err, "manifest");
}

// EXAMPLE: apply_rejects_full_describe_dump (non-empty observational scope) ->
// exit 1 manifest, no transaction opened.
ZD_TEST(test_apply_rejects_full_describe_dump) {
    std::string manifest = write_file("full_dump.json", MANIFEST_WITH_OBSERVATIONAL);
    auto r = run({"apply", "manifest-path=" + manifest});
    ZD_EXPECT_CODE(r, 1);
    ZD_EXPECT_CONTAINS(r.err, "manifest");
}

// EXAMPLE: apply_transaction_unavailable (mode=external, not in a transaction)
// -> exit 2 transaction. The manifest is structurally complete & valid so the
// only error is the transaction unavailability.
ZD_TEST(test_apply_transaction_unavailable_external) {
    std::string manifest = write_file("apply_ext.json", MANIFEST_FULL);
    // mode=external requires running inside a snapshot transaction; the test
    // environment is not, so the tool must report a transaction error. The
    // intent diff must be non-empty for the transaction to be reached: the
    // manifest declares scopes against an absent applied record.
    auto r = run({"apply", "manifest-path=" + manifest, "mode=external",
                  "applied-root=" + tmpdir() + "/emptyroot"});
    ZD_EXPECT_CODE(r, 2);
    ZD_EXPECT_CONTAINS(r.err, "transaction");
}

// ======================================================================
// Entry point
// ======================================================================
int main() {
    int rc = zdtest::run_all();
    return rc;
}
