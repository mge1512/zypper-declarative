// generated from spec: zypper-declarative.spec.md sha256:f2cc80627e483a48bb8411d297711bc5f6c6e74c28dbf0dafc8fe7bd8817251e
// tests by: claude-opus-4-8
//
// describe and status tests. describe runs against a synthetic root (root=...)
// so the /etc walk, symlink-verbatim, special-file skip, empty-scope omission,
// and resolve-format output behaviour are exercised without a live system and
// without root privilege. status (no declaration applied) is also covered.
//
// Covers EXAMPLES: describe_emits_manifest (shape), describe_output_unwritable,
// describe_traverses_etc_subdirectories, describe_records_symlink_verbatim,
// describe_skips_special_file, describe_omits_genuinely_empty_scope,
// describe_out_extension_yaml, describe_out_extension_json,
// describe_format_overrides_extension, describe_format_yaml,
// status_no_declaration.
#include "test_harness.hpp"
#include "test_fixtures.hpp"
#include <sys/stat.h>
#include <sys/types.h>

using namespace zdtest;

// Build a synthetic root with /etc and /usr/lib/... structure.
// Returns the root path. The caller may pass extra builders.
static fs::path build_root(const std::string& tag) {
    fs::path root = make_tmpdir("zd-root-" + tag);
    std::error_code ec;
    fs::create_directories(root / "etc", ec);
    // a readable repos.d so the repositories scope is not an unreadable source
    fs::create_directories(root / "etc" / "zypp" / "repos.d", ec);
    return root;
}

// EXAMPLE: describe_omits_genuinely_empty_scope -- a readable but empty
// repos.d yields no repositories scope (omitted, not emitted empty). With an
// otherwise-empty synthetic /etc the describe output must not contain an empty
// "_elements": [] for repositories. Exit 0.
TEST(test_describe_omits_empty_repositories_scope) {
    fs::path root = build_root("empty-repos");
    auto r = run({"describe", "root=" + root.string()});
    expect_eq_int(r.code, 0, "describe empty repos exit");
    // The repositories scope must not be emitted with an empty element list.
    expect_not_contains(r.out, "repository_system",
                        "genuinely empty repositories scope must be omitted");
}

// EXAMPLE: describe_records_symlink_verbatim -- a changed/unpackaged symlink in
// /etc whose target is "../foo/bar.conf" is emitted as type "link" with that
// verbatim target and sha256 "". The symlink is not dereferenced.
TEST(test_describe_symlink_verbatim) {
    fs::path root = build_root("symlink");
    std::error_code ec;
    fs::create_symlink("../foo/bar.conf", root / "etc" / "mylink", ec);
    check(!ec, "create symlink fixture");
    auto r = run({"describe", "root=" + root.string()});
    expect_eq_int(r.code, 0, "describe symlink exit");
    expect_contains(r.out, "../foo/bar.conf",
                    "symlink target stored verbatim");
    expect_contains(r.out, "\"link\"", "symlink emitted as type link");
}

// EXAMPLE: describe_traverses_etc_subdirectories -- the walk descends into a
// subdirectory rather than reading it as a file; a changed file inside is
// emitted as a type "file" record; no "is a directory" error; run does not abort.
TEST(test_describe_traverses_subdirectory) {
    fs::path root = build_root("subdir");
    std::error_code ec;
    fs::create_directories(root / "etc" / "ImageMagick-7", ec);
    {
        std::ofstream f(root / "etc" / "ImageMagick-7" / "policy.xml");
        f << "<policymap></policymap>\n";
    }
    auto r = run({"describe", "root=" + root.string()});
    expect_eq_int(r.code, 0, "describe subdir traversal exit (no is-a-directory abort)");
    expect_not_contains(r.err, "is a directory",
                        "directory must be traversed, not read as a file");
    expect_contains(r.out, "policy.xml", "changed file inside subdir is emitted");
}

// EXAMPLE: describe_skips_special_file -- a fifo in /etc is skipped: not read,
// not hashed, not emitted; the run does not hang or error.
TEST(test_describe_skips_special_file) {
    fs::path root = build_root("fifo");
    fs::path fifo = root / "etc" / "myfifo";
    int rc = ::mkfifo(fifo.c_str(), 0644);
    check(rc == 0, "create fifo fixture");
    auto r = run({"describe", "root=" + root.string()});
    expect_eq_int(r.code, 0, "describe special-file skip exit");
    expect_not_contains(r.out, "myfifo", "special file must not be emitted");
}

// EXAMPLE: describe_out_extension_json -- no format option, out=...json ->
// resolve-format selects json; the file contains a JSON document. Exit 0.
TEST(test_describe_out_extension_json) {
    fs::path root = build_root("ext-json");
    fs::path out = make_tmpdir("zd-out-json") / "state.json";
    auto r = run({"describe", "root=" + root.string(), "out=" + out.string()});
    expect_eq_int(r.code, 0, "describe out=json exit");
    check(fs::exists(out), "describe json output file exists");
    std::ifstream f(out);
    std::string first; std::getline(f, first);
    // JSON object starts with '{' (possibly after whitespace)
    bool brace = first.find('{') != std::string::npos;
    check(brace, "json output begins with a JSON object");
}

// EXAMPLE: describe_out_extension_yaml -- out=...yaml -> yaml selected; the
// file's first line is not a JSON object opener. Exit 0.
TEST(test_describe_out_extension_yaml) {
    fs::path root = build_root("ext-yaml");
    fs::path out = make_tmpdir("zd-out-yaml") / "state.yaml";
    auto r = run({"describe", "root=" + root.string(), "out=" + out.string()});
    expect_eq_int(r.code, 0, "describe out=yaml exit");
    check(fs::exists(out), "describe yaml output file exists");
    std::ifstream f(out);
    std::string first; std::getline(f, first);
    // YAML must not start with a JSON-object brace (the 0.1.0 gate's grep -vq "^{").
    check(first.empty() || first[0] != '{',
          "yaml output first line must not begin with '{'");
}

// EXAMPLE: describe_format_overrides_extension -- format=json out=...yaml ->
// explicit option wins; the .yaml file contains a JSON document. Exit 0.
TEST(test_describe_format_overrides_extension) {
    fs::path root = build_root("override");
    fs::path out = make_tmpdir("zd-out-override") / "state.yaml";
    auto r = run({"describe", "root=" + root.string(),
                  "format=json", "out=" + out.string()});
    expect_eq_int(r.code, 0, "describe format override exit");
    std::ifstream f(out);
    std::string first; std::getline(f, first);
    check(first.find('{') != std::string::npos,
          "explicit format=json wins: .yaml file contains JSON");
}

// EXAMPLE: describe_format_yaml -- format=yaml -> stdout is a YAML document
// (not a JSON object opener). Exit 0.
TEST(test_describe_format_yaml_stdout) {
    fs::path root = build_root("fmt-yaml");
    auto r = run({"describe", "root=" + root.string(), "format=yaml"});
    expect_eq_int(r.code, 0, "describe format=yaml exit");
    // The very first non-space character of stdout must not be a JSON brace.
    size_t pos = r.out.find_first_not_of(" \t\r\n");
    check(pos == std::string::npos || r.out[pos] != '{',
          "format=yaml stdout is not a JSON object");
}

// describe stdout (default format) is JSON with format_version 1.
// EXAMPLE: describe_emits_manifest (the meta shape part, no live packages needed).
TEST(test_describe_stdout_json_shape) {
    fs::path root = build_root("json-shape");
    auto r = run({"describe", "root=" + root.string()});
    expect_eq_int(r.code, 0, "describe default json exit");
    expect_contains(r.out, "format_version", "describe meta.format_version present");
    expect_contains(r.out, "\"meta\"", "describe meta object present");
}

// EXAMPLE: describe_output_unwritable -- output path not writable ->
// domain=invocation, exit 2.
TEST(test_describe_output_unwritable) {
    fs::path root = build_root("unwritable");
    auto r = run({"describe", "root=" + root.string(),
                  "out=/nonexistent-dir-zd/sub/state.json"});
    expect_eq_int(r.code, 2, "describe unwritable out exit");
    expect_contains(r.err, "invocation", "unwritable out domain=invocation");
}

// EXAMPLE: status_no_declaration -- with applied-root pointing at a root that
// has no applied.json, status prints "no declaration applied" and exits 0.
TEST(test_status_no_declaration) {
    fs::path root = make_tmpdir("zd-status-empty");
    auto r = run({"status", "applied-root=" + root.string()});
    expect_eq_int(r.code, 0, "status no declaration exit");
    expect_contains(r.out, "no declaration applied", "status no-declaration message");
}
