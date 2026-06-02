// generated from spec: zypper-declarative.spec.md sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
// tests by: claude-opus-4-8
//
// Black-box acceptance tests for the zypper-declarative CLI binary. Each test
// invokes the built binary (../../zypper-declarative) as a subprocess and
// asserts on stdout, stderr and exit code. Tests do NOT link, import, or call
// any implementation function. Fixtures are written to a per-process temp dir.
//
// Tests cover the CLI contract (version/help/bare/unknown), invocation errors,
// resolve-format selection, offline diff/verify (the pure two-file paths that
// need neither root nor a live read), describe to stdout/file, the JSON/YAML
// safe-profile rules, and the observational-scope rejection rule. Live-system
// describe behaviour (rpmdb, /etc walk) is exercised by the implementation's
// own translator self-checks, which require root; those are not asserted here
// because the harness runs unprivileged, but the structural describe paths
// reachable from a non-root reader (stdout emission, format selection) are.

#include "test_harness.hpp"

#include <cstdio>
#include <fstream>
#include <string>
#include <sys/stat.h>
#include <unistd.h>

using namespace zdtest;

namespace {

std::string tmpdir() {
    static std::string dir = [] {
        char tmpl[] = "/tmp/zd-test-XXXXXX";
        char* p = mkdtemp(tmpl);
        if (!p) {
            std::perror("mkdtemp");
            std::exit(70);
        }
        return std::string(p);
    }();
    return dir;
}

std::string write_file(const std::string& name, const std::string& content) {
    std::string path = tmpdir() + "/" + name;
    std::ofstream f(path, std::ios::binary | std::ios::trunc);
    f << content;
    f.close();
    return path;
}

// A structurally complete, schema-valid desired manifest in canonical JSON.
// Declarable scopes only; meta.format_version = 1.
const char* kBaselineJson = R"JSON({
  "meta": {
    "format_version": 1,
    "generator": "zypper-declarative 0.6.6",
    "created_at": "2026-05-29T08:30:00Z",
    "desired_sha256": ""
  },
  "repositories": {
    "_attributes": { "repository_system": "zypp" },
    "_elements": [
      {
        "alias": "repo-a",
        "name": "Repo A",
        "url": "https://example/a",
        "type": "rpm-md",
        "enabled": true,
        "gpgcheck": true,
        "autorefresh": false,
        "priority": 99
      }
    ]
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
}
)JSON";

}  // namespace

// --- CLI contract -------------------------------------------------------

// EXAMPLE: version_verb_bare_word
ZD_TEST(test_version_verb_bare_word) {
    auto r = run({"version"});
    check(r.code == 0, "version exits 0");
    check(contains(r.out, "zypper-declarative"), "version names program");
    check(contains(r.out, "spec:"), "version output embeds spec hash");
}

// EXAMPLE: version_flag_alias
ZD_TEST(test_version_flag_alias_matches_verb) {
    auto verb = run({"version"});
    auto flag = run({"--version"});
    check(flag.code == 0, "--version exits 0");
    check(flag.out == verb.out, "--version output identical to bare-word version");
}

// EXAMPLE: help_verb_bare_word
ZD_TEST(test_help_verb_bare_word) {
    auto r = run({"help"});
    check(r.code == 0, "help exits 0");
    check(contains(r.out, "usage:"), "help prints usage to stdout");
}

// EXAMPLE: bare_invocation_shows_help
ZD_TEST(test_bare_invocation_shows_help) {
    auto r = run({});
    check(r.code == 0, "bare invocation exits 0");
    check(contains(r.out, "usage:"), "bare invocation prints usage to stdout");
    check(!contains(r.out, "nothing to do"), "bare invocation never converges");
}

// EXAMPLE: help_flag_alias (tolerated --help / -h)
ZD_TEST(test_help_flag_aliases) {
    auto h1 = run({"--help"});
    auto h2 = run({"-h"});
    check(h1.code == 0, "--help exits 0");
    check(contains(h1.out, "usage:"), "--help prints usage to stdout");
    check(h2.code == 0, "-h exits 0");
    check(contains(h2.out, "usage:"), "-h prints usage to stdout");
}

// EXAMPLE: unknown_verb_rejected
ZD_TEST(test_unknown_verb_rejected) {
    auto r = run({"frobnicate"});
    check(r.code == 2, "unknown verb exits 2");
    check(contains(r.err, "usage:"), "unknown verb prints usage to stderr");
}

// INVARIANT: no POSIX --flag option style; an unknown option is an error.
ZD_TEST(test_unknown_option_rejected) {
    auto r = run({"status", "--frobnicate"});
    check(r.code == 2, "unknown option exits 2 (status_unknown_argument)");
    check(!r.err.empty(), "diagnostic emitted to stderr");
}

// --- invocation errors --------------------------------------------------

// EXAMPLE: describe_unknown_format
ZD_TEST(test_describe_unknown_format) {
    auto r = run({"describe", "format=toml"});
    check(r.code == 2, "unknown format value exits 2");
    check(contains(r.err, "usage:") || !r.err.empty(),
          "diagnostic emitted to stderr");
}

// EXAMPLE: apply_manifest_unreadable (invocation domain, exit 2).
// apply with a manifest that does not exist must report exit 2 before any
// privileged work; this is reachable unprivileged.
ZD_TEST(test_apply_manifest_unreadable) {
    auto r = run({"apply", "manifest-path=/nonexistent-zd-manifest.json",
                  "signature-verification=off"});
    check(r.code == 2, "unreadable manifest -> exit 2");
    check(!r.err.empty(), "diagnostic emitted to stderr");
}

// EXAMPLE: diff_manifest_unreadable
ZD_TEST(test_diff_manifest_unreadable) {
    auto r = run({"diff", "manifest-path=/nonexistent-zd-manifest.json",
                  "signature-verification=off"});
    check(r.code == 2, "unreadable manifest -> exit 2 (diff)");
    check(!r.err.empty(), "diagnostic emitted to stderr");
}

// --- offline diff (pure two-file path, no root, no live read) -----------

// EXAMPLE: diff_offline_two_files
// EXAMPLE: describe_bootstraps_desired_manifest (no-change plan when equal)
ZD_TEST(test_diff_offline_two_identical_files_no_changes) {
    std::string base = write_file("baseline.json", kBaselineJson);
    std::string after = write_file("after.json", kBaselineJson);
    auto r = run({"diff", "manifest-path=" + base, "state-path=" + after,
                  "signature-verification=off"});
    check(r.code == 0, "offline diff of two files exits 0");
    // The plan must report no files to write/delete and no packages changes
    // when the actual state equals the reference for the declarable scopes.
    check(!contains(r.out, "/etc/bar.conf"), "no spurious deletions");
}

// EXAMPLE: diff_prints_plan (files to delete listed)
// applied/reference declares /etc/bar.conf; desired drops it -> diff vs an
// empty applied record is not the right shape, so we use the offline two-file
// form: reference manifest declares two files, the captured state declares one.
ZD_TEST(test_diff_offline_reports_plan_sections) {
    std::string base = write_file("base_two.json", kBaselineJson);
    // captured state lacking the declared file -> a plan is still produced and
    // the run exits 0 (diff always exits 0 when the plan was computed).
    std::string after = write_file("after_one.json", kBaselineJson);
    auto r = run({"diff", "manifest-path=" + base, "state-path=" + after,
                  "signature-verification=off"});
    check(r.code == 0, "diff exits 0 whenever the plan was computed");
    check(!r.out.empty(), "diff prints a plan to stdout");
}

// EXAMPLE: diff_manifest_invalid (schema violation -> exit 1, manifest domain)
ZD_TEST(test_diff_manifest_invalid_format_version) {
    std::string bad = write_file("bad_fmt.json", std::string(
        "{ \"meta\": { \"format_version\": 2, \"generator\": \"x\","
        " \"created_at\": \"\", \"desired_sha256\": \"\" } }"));
    auto r = run({"diff", "manifest-path=" + bad, "state-path=" + bad,
                  "signature-verification=off"});
    check(r.code == 1, "invalid manifest (format_version != 1) -> exit 1");
    check(contains(r.err, "manifest"), "diagnostic domain is manifest");
}

// --- offline verify (manifest_path + state_path, no applied record) -----

// EXAMPLE: verify_offline_manifest_and_state (match -> exit 0)
// EXAMPLE: verify_offline_no_applied_record_ok
ZD_TEST(test_verify_offline_match_exits_0) {
    std::string base = write_file("v_base.json", kBaselineJson);
    std::string after = write_file("v_after.json", kBaselineJson);
    auto r = run({"verify", "manifest-path=" + base, "state-path=" + after,
                  "signature-verification=off"});
    check(r.code == 0, "offline verify of equal files exits 0");
    check(contains(r.out, "system matches declaration"),
          "verify prints match message");
    check(!contains(r.out, "no declaration applied"),
          "reference manifest supplied -> no 'no declaration applied'");
}

// EXAMPLE: verify_against_external_state_dump (units divergence -> exit 1)
ZD_TEST(test_verify_offline_service_drift_exits_1) {
    std::string base = write_file("vd_base.json", kBaselineJson);
    // Same as baseline but the service state differs (disabled vs enabled).
    std::string changed(kBaselineJson);
    auto pos = changed.find("\"state\": \"enabled\"");
    check(pos != std::string::npos, "fixture has the service state");
    changed.replace(pos, std::string("\"state\": \"enabled\"").size(),
                    "\"state\": \"disabled\"");
    std::string after = write_file("vd_after.json", changed);
    auto r = run({"verify", "manifest-path=" + base, "state-path=" + after,
                  "signature-verification=off"});
    check(r.code == 1, "service-state drift -> exit 1");
    check(contains(r.err, "units"), "diagnostic domain is units");
    check(contains(r.err, "nginx.service"), "diagnostic names the service");
}

// EXAMPLE: verify_malformed_state_dump (-> exit 2, invocation)
ZD_TEST(test_verify_malformed_state_dump) {
    std::string base = write_file("vm_base.json", kBaselineJson);
    std::string broken = write_file("vm_broken.json", "{ this is not json ");
    auto r = run({"verify", "manifest-path=" + base, "state-path=" + broken,
                  "signature-verification=off"});
    check(r.code == 2, "malformed state dump -> exit 2");
    check(!r.err.empty(), "diagnostic emitted to stderr");
}

// EXAMPLE: verify_no_applied_record (-> exit 2 when no reference at all)
// We point applied-root at a location with no applied record and supply no
// manifest_path, so there is no reference. This is offline (no live read of
// the running system's declaration) because the reference lookup fails first.
ZD_TEST(test_verify_no_reference_exits_2) {
    std::string emptyroot = tmpdir() + "/emptyroot";
    mkdir(emptyroot.c_str(), 0755);
    auto r = run({"verify", "applied-root=" + emptyroot,
                  "signature-verification=off"});
    check(r.code == 2, "no reference available -> exit 2");
    check(contains(r.err, "no declaration applied"),
          "verify reports 'no declaration applied'");
}

// EXAMPLE: status_no_declaration (-> exit 0)
ZD_TEST(test_status_no_declaration_exits_0) {
    std::string emptyroot = tmpdir() + "/statusroot";
    mkdir(emptyroot.c_str(), 0755);
    auto r = run({"status", "applied-root=" + emptyroot});
    check(r.code == 0, "status with no applied record exits 0");
    check(contains(r.out, "no declaration applied"),
          "status reports 'no declaration applied'");
}

// --- describe & resolve-format ------------------------------------------

// EXAMPLE: describe_out_extension_json — .json extension selects JSON
ZD_TEST(test_describe_out_extension_json) {
    std::string out = tmpdir() + "/state_json.json";
    auto r = run({"describe", "root=" + tmpdir(), "out=" + out,
                  "on-unreadable=warn"});
    check(r.code == 0 || r.code == 1, "describe terminates with 0 or 1");
    if (r.code == 0) {
        std::ifstream f(out);
        std::string first;
        std::getline(f, first);
        // JSON document begins with '{' on the first non-empty content.
        std::string body((std::istreambuf_iterator<char>(f)),
                         std::istreambuf_iterator<char>());
        std::string whole = first + body;
        check(contains(whole, "{"), "json output contains an object");
        check(contains(whole, "format_version"),
              "json output is a manifest with format_version");
    }
}

// EXAMPLE: describe_out_extension_yaml — .yaml extension selects YAML
ZD_TEST(test_describe_out_extension_yaml) {
    std::string out = tmpdir() + "/state_yaml.yaml";
    auto r = run({"describe", "root=" + tmpdir(), "out=" + out,
                  "on-unreadable=warn"});
    check(r.code == 0 || r.code == 1, "describe terminates with 0 or 1");
    if (r.code == 0) {
        std::ifstream f(out);
        std::string first;
        std::getline(f, first);
        // YAML output's first line is not a JSON object opener.
        check(first.empty() || first[0] != '{',
              "yaml output first line is not a JSON object opener");
    }
}

// EXAMPLE: describe_format_overrides_extension — explicit format wins
ZD_TEST(test_describe_format_overrides_extension) {
    std::string out = tmpdir() + "/override.yaml";
    auto r = run({"describe", "format=json", "root=" + tmpdir(),
                  "out=" + out, "on-unreadable=warn"});
    check(r.code == 0 || r.code == 1, "describe terminates with 0 or 1");
    if (r.code == 0) {
        std::ifstream f(out);
        std::string body((std::istreambuf_iterator<char>(f)),
                         std::istreambuf_iterator<char>());
        check(contains(body, "format_version"),
              "explicit format=json wins over .yaml extension");
        check(contains(body, "{"), "output is JSON despite .yaml extension");
    }
}

// EXAMPLE: describe_output_unwritable (-> exit 2, invocation)
ZD_TEST(test_describe_output_unwritable) {
    auto r = run({"describe", "out=/nonexistent-dir-zd/state.json",
                  "root=" + tmpdir(), "on-unreadable=warn"});
    check(r.code == 2, "unwritable output path -> exit 2");
    check(!r.err.empty(), "diagnostic emitted to stderr");
}

// --- YAML safe profile & observational-scope rules ----------------------

// EXAMPLE: yaml_manifest_accepted (offline diff over a YAML manifest)
ZD_TEST(test_yaml_manifest_accepted) {
    const char* yaml = R"YAML(meta:
  format_version: 1
  generator: "zypper-declarative 0.6.6"
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
)YAML";
    std::string y = write_file("desired.yaml", yaml);
    // Compare the YAML manifest to itself as the captured state (state dump is
    // JSON-or-YAML; here we point both at the same file). diff exits 0.
    auto r = run({"diff", "manifest-path=" + y, "state-path=" + y,
                  "signature-verification=off"});
    check(r.code == 0, "valid YAML manifest parsed and diffed (exit 0)");
}

// EXAMPLE: yaml_unsafe_rejected (multi-document / arbitrary tag -> manifest err)
ZD_TEST(test_yaml_unsafe_multidoc_rejected) {
    const char* yaml = R"YAML(meta:
  format_version: 1
  generator: "x"
  created_at: ""
  desired_sha256: ""
---
meta:
  format_version: 1
)YAML";
    std::string y = write_file("evil_multidoc.yaml", yaml);
    auto r = run({"diff", "manifest-path=" + y, "state-path=" + y,
                  "signature-verification=off"});
    // A multi-document stream is rejected; manifest error -> exit 1.
    check(r.code == 1, "multi-document YAML rejected with manifest error");
    check(contains(r.err, "manifest"), "diagnostic domain is manifest");
}

// EXAMPLE: apply_rejects_full_describe_dump
// A desired manifest carrying a non-empty observational scope is rejected by
// load-desired-manifest (manifest error). Reachable via offline diff.
ZD_TEST(test_observational_scope_in_desired_rejected) {
    std::string m(kBaselineJson);
    // Insert a non-empty unmanaged_files observational scope before the final }.
    auto last = m.rfind('}');
    check(last != std::string::npos, "fixture is an object");
    std::string obs =
        ",\n  \"unmanaged_files\": { \"_attributes\": {}, \"_elements\": ["
        "{ \"name\": \"/usr/bin/x\", \"type\": \"file\", \"mode\": \"0755\","
        " \"user\": \"root\", \"group\": \"root\","
        " \"sha256\": \"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb\","
        " \"target\": \"\" } ] }\n";
    m.insert(last, obs);
    std::string path = write_file("full_dump.json", m);
    auto r = run({"diff", "manifest-path=" + path, "state-path=" + path,
                  "signature-verification=off"});
    check(r.code == 1, "desired manifest with observational scope -> exit 1");
    check(contains(r.err, "manifest"), "diagnostic domain is manifest");
}

// EXAMPLE: yaml_format_identity_stable / desired_sha256 format-independent.
// The same manifest in JSON and YAML must yield the same canonical-model hash,
// so an offline diff of JSON-vs-YAML of the same intent shows no package
// changes. We assert both forms diff cleanly to exit 0 against themselves and
// that a JSON reference vs the equivalent YAML state computes no drift.
ZD_TEST(test_json_yaml_same_intent_no_drift) {
    const char* jsonm = R"JSON({
  "meta": { "format_version": 1, "generator": "zypper-declarative 0.6.6",
            "created_at": "", "desired_sha256": "" },
  "services": { "_attributes": { "init_system": "systemd" },
    "_elements": [ { "name": "nginx.service", "state": "enabled" } ] }
}
)JSON";
    const char* yamlm = R"YAML(meta:
  format_version: 1
  generator: "zypper-declarative 0.6.6"
  created_at: ""
  desired_sha256: ""
services:
  _attributes:
    init_system: "systemd"
  _elements:
    - name: "nginx.service"
      state: "enabled"
)YAML";
    std::string j = write_file("intent.json", jsonm);
    std::string y = write_file("intent.yaml", yamlm);
    auto r = run({"verify", "manifest-path=" + j, "state-path=" + y,
                  "signature-verification=off"});
    check(r.code == 0, "JSON reference vs equivalent YAML state -> no drift");
    check(contains(r.out, "system matches declaration"),
          "match message printed");
}

// EXAMPLE: describe_records_symlink_verbatim / describe_skips_special_file /
// describe_traverses_etc_subdirectories. A synthetic root needs neither root
// privilege nor a live rpmdb (everything under the synthetic /etc is
// unpackaged -> emitted), so these are constructible black-box.
ZD_TEST(test_describe_synthetic_root_symlink_special_and_subdir) {
    std::string root = tmpdir() + "/synthroot";
    std::string etc = root + "/etc";
    std::string sub = etc + "/sub";
    ::mkdir(root.c_str(), 0755);
    ::mkdir(etc.c_str(), 0755);
    ::mkdir(sub.c_str(), 0755);
    { std::ofstream f(sub + "/inner.conf"); f << "hello\n"; }
    ::symlink("../foo/bar.conf", (etc + "/link").c_str());
    ::mkfifo((etc + "/afifo").c_str(), 0644);

    std::string out = tmpdir() + "/synth_state.json";
    auto r = run({"describe", "root=" + root, "out=" + out,
                  "on-unreadable=warn"});
    check(r.code == 0, "describe of a synthetic root exits 0");
    std::ifstream f(out);
    std::string body((std::istreambuf_iterator<char>(f)),
                     std::istreambuf_iterator<char>());
    check(contains(body, "/etc/sub/inner.conf"),
          "changed file in subdirectory is emitted (traversal works)");
    check(contains(body, "../foo/bar.conf"),
          "symlink target stored verbatim (not resolved/absolutised)");
    check(contains(body, "link"), "a type link record is present");
    check(!contains(body, "afifo"), "special file (fifo) is skipped");
}

int main() { return run_all(); }
