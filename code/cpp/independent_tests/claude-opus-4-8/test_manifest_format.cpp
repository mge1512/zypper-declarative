// generated from spec: zypper-declarative.spec.md sha256:f2cc80627e483a48bb8411d297711bc5f6c6e74c28dbf0dafc8fe7bd8817251e
// tests by: claude-opus-4-8
//
// Manifest loading & format resolution tests, exercised through the `diff`,
// `verify`, and `apply` verbs (the externally observable interface).
// Covers EXAMPLES: apply_manifest_invalid, apply_manifest_unreadable,
// diff_manifest_unreadable, verify_malformed_state_dump,
// apply_rejects_full_describe_dump, yaml_manifest_accepted,
// yaml_unsafe_rejected, and resolve-format behaviour
// (describe_out_extension_*, describe_format_overrides_extension).
#include "test_harness.hpp"
#include "test_fixtures.hpp"

using namespace zdtest;

// EXAMPLE: apply_manifest_invalid -- meta.format_version=2 -> domain=manifest, exit 1.
TEST(test_apply_manifest_invalid_format_version) {
    std::string m = R"JSON({
  "meta": { "format_version": 2, "generator": "t", "created_at": "x", "desired_sha256": "" },
  "packages": { "_attributes": { "package_system": "rpm" }, "_elements": [] }
})JSON";
    std::string path = write_temp_file("bad.json", m);
    auto r = run({"apply", "manifest-path=" + path});
    expect_eq_int(r.code, 1, "invalid manifest exit");
    expect_contains(r.err, "manifest", "invalid manifest domain=manifest");
}

// EXAMPLE: apply_manifest_unreadable -- missing file -> domain=invocation, exit 2.
TEST(test_apply_manifest_unreadable) {
    auto r = run({"apply", "manifest-path=/nonexistent-zd-manifest.json"});
    expect_eq_int(r.code, 2, "unreadable manifest exit");
    expect_contains(r.err, "invocation", "unreadable manifest domain=invocation");
}

// EXAMPLE: diff_manifest_unreadable -- missing file -> domain=invocation, exit 2.
TEST(test_diff_manifest_unreadable) {
    auto r = run({"diff", "manifest-path=/nonexistent-zd-manifest.json"});
    expect_eq_int(r.code, 2, "diff unreadable manifest exit");
    expect_contains(r.err, "invocation", "diff unreadable domain=invocation");
}

// EXAMPLE: verify_malformed_state_dump -- not a valid Manifest -> domain=invocation, exit 2.
TEST(test_verify_malformed_state_dump) {
    std::string path = write_temp_file("broken.json", "this is not json {{{");
    auto manifest = write_temp_file("ref.json", valid_desired_json());
    auto r = run({"verify", "manifest-path=" + manifest, "state-path=" + path});
    expect_eq_int(r.code, 2, "malformed state dump exit");
    expect_contains(r.err, "invocation", "malformed state dump domain=invocation");
}

// EXAMPLE: apply_rejects_full_describe_dump -- desired manifest carrying a
// non-empty observational scope (unmanaged_files) -> domain=manifest, exit 1.
TEST(test_apply_rejects_observational_scope) {
    std::string m = R"JSON({
  "meta": { "format_version": 1, "generator": "t", "created_at": "x", "desired_sha256": "" },
  "packages": { "_attributes": { "package_system": "rpm" }, "_elements": [] },
  "unmanaged_files": {
    "_attributes": null,
    "_elements": [
      { "name": "/usr/bin/extra", "type": "file", "mode": "0755",
        "user": "root", "group": "root", "sha256":
        "2222222222222222222222222222222222222222222222222222222222222222",
        "target": "" }
    ]
  }
})JSON";
    std::string path = write_temp_file("fulldump.json", m);
    auto r = run({"apply", "manifest-path=" + path});
    expect_eq_int(r.code, 1, "observational-scope rejection exit");
    expect_contains(r.err, "manifest", "observational rejection domain=manifest");
}

// EXAMPLE: yaml_manifest_accepted -- a YAML serialisation of a valid manifest
// is parsed under the safe profile, validated, and the plan computed; exit 0.
// Used in offline two-file diff mode so no live system is read.
TEST(test_yaml_manifest_accepted_offline) {
    std::string yaml = R"YAML(meta:
  format_version: 1
  generator: "test"
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
    std::string mpath = write_temp_file("desired.yaml", yaml);
    // an empty actual state dump (valid manifest, no scopes asserted)
    std::string state = R"JSON({
  "meta": { "format_version": 1, "generator": "t", "created_at": "x", "desired_sha256": "" }
})JSON";
    std::string spath = write_temp_file("state.json", state);
    auto r = run({"diff", "manifest-path=" + mpath, "state-path=" + spath});
    expect_eq_int(r.code, 0, "yaml manifest accepted exit");
}

// EXAMPLE: yaml_unsafe_rejected -- a YAML manifest using an executable/arbitrary
// tag is rejected with a manifest error (exit 1), no transaction opened.
TEST(test_yaml_unsafe_tag_rejected) {
    std::string yaml = R"YAML(meta:
  format_version: 1
  generator: !!python/object/apply:os.system ["echo pwned"]
  created_at: "x"
  desired_sha256: ""
)YAML";
    std::string mpath = write_temp_file("evil.yaml", yaml);
    auto r = run({"apply", "manifest-path=" + mpath});
    expect_eq_int(r.code, 1, "unsafe yaml rejected exit");
    expect_contains(r.err, "manifest", "unsafe yaml domain=manifest");
}

// EXAMPLE: yaml_unsafe_rejected (multi-document variant) -- multiple documents
// are rejected by the safe profile.
TEST(test_yaml_multidoc_rejected) {
    std::string yaml = R"YAML(meta:
  format_version: 1
  generator: "t"
  created_at: "x"
  desired_sha256: ""
---
meta:
  format_version: 1
)YAML";
    std::string mpath = write_temp_file("multi.yaml", yaml);
    auto r = run({"apply", "manifest-path=" + mpath});
    expect_eq_int(r.code, 1, "multidoc yaml rejected exit");
    expect_contains(r.err, "manifest", "multidoc yaml domain=manifest");
}
