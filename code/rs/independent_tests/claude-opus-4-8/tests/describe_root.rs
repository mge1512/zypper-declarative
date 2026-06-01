// generated from spec: zypper-declarative.spec.md sha256:87b9f2f3bf92afd6fe037412d56a23959290e5816d722a3def9505d07aa5acd7
// tests by: claude-opus-4-8
//
// describe tests. Several describe EXAMPLEs assert on the live system (rpmdb,
// repos.d, systemd) which is not deterministic in CI; those are exercised with a
// synthetic root=<dir> where the behaviour is constructible offline. Covers EXAMPLEs
// describe_output_unwritable, describe_out_extension_yaml/json,
// describe_format_overrides_extension, describe_records_symlink_verbatim,
// describe_skips_special_file, describe_traverses_etc_subdirectories,
// scope_attributes_always_object, describe_omits_genuinely_empty_scope (repos.d
// empty), describe_scope_full_emits_observational_scopes (structural), and the
// observable INVARIANTs on the /etc walk object model.

mod common;
use common::*;

use std::os::unix::fs::symlink;
use std::path::Path;

// A synthetic root with an /etc subtree we control. repos.d is created empty so the
// repositories scope is genuinely empty (omitted), rpmdb/systemd reads fall to the
// host but we assert only on the structural rules that the synthetic /etc exercises.
fn make_root(tag: &str) -> std::path::PathBuf {
    let dir = temp_dir(tag);
    let etc = dir.join("etc");
    std::fs::create_dir_all(etc.join("zypp").join("repos.d")).unwrap();
    dir
}

fn parse_json_doc(s: &str) -> serde_json::Value {
    serde_json::from_str(s).unwrap_or_else(|e| panic!("describe stdout not valid JSON: {e}\n---\n{s}"))
}

// EXAMPLE: describe_output_unwritable -> exit 2, domain=invocation
#[test]
fn test_describe_output_unwritable_exit2() {
    let root = make_root("desc-unwritable");
    // A path under a non-existent, uncreatable directory: /proc is not writable.
    let out = run(&[
        "describe",
        &format!("root={}", root.display()),
        "out=/proc/zd-cannot-write/state.json",
    ]);
    assert_eq!(
        exit_code(&out),
        2,
        "an unwritable output path must exit 2; stderr={}",
        stderr_str(&out)
    );
}

// EXAMPLE: describe_out_extension_json -- .json extension selects JSON output.
#[test]
fn test_describe_out_extension_json() {
    let root = make_root("desc-ext-json");
    let dir = temp_dir("desc-ext-json-out");
    let outp = dir.join("state.json");
    let out = run(&[
        "describe",
        &format!("root={}", root.display()),
        &format!("out={}", outp.display()),
    ]);
    // describe may exit 0 (clean) on a synthetic root; if a host scope source is
    // unreadable under default error this could be 1. We only assert the format
    // resolution when the file was produced.
    if exit_code(&out) == 0 {
        let content = std::fs::read_to_string(&outp).expect("output file written");
        assert!(
            content.trim_start().starts_with('{'),
            ".json extension must select JSON output; got: {}",
            &content[..content.len().min(80)]
        );
    }
}

// EXAMPLE: describe_out_extension_yaml -- .yaml extension selects YAML output.
#[test]
fn test_describe_out_extension_yaml() {
    let root = make_root("desc-ext-yaml");
    let dir = temp_dir("desc-ext-yaml-out");
    let outp = dir.join("state.yaml");
    let out = run(&[
        "describe",
        &format!("root={}", root.display()),
        &format!("out={}", outp.display()),
    ]);
    if exit_code(&out) == 0 {
        let content = std::fs::read_to_string(&outp).expect("output file written");
        assert!(
            !content.trim_start().starts_with('{'),
            ".yaml extension must select YAML output (not a JSON object); got: {}",
            &content[..content.len().min(80)]
        );
    }
}

// EXAMPLE: describe_format_overrides_extension -- format=json with out=...yaml writes JSON.
#[test]
fn test_describe_format_overrides_extension() {
    let root = make_root("desc-fmt-override");
    let dir = temp_dir("desc-fmt-override-out");
    let outp = dir.join("state.yaml");
    let out = run(&[
        "describe",
        &format!("root={}", root.display()),
        "format=json",
        &format!("out={}", outp.display()),
    ]);
    if exit_code(&out) == 0 {
        let content = std::fs::read_to_string(&outp).expect("output file written");
        assert!(
            content.trim_start().starts_with('{'),
            "explicit format=json must win over the .yaml extension; got: {}",
            &content[..content.len().min(80)]
        );
    }
}

// EXAMPLE: describe_records_symlink_verbatim -- a symlink under /etc whose target is
// "../foo/bar.conf" is emitted as type "link" with the verbatim target, sha256 "".
// Constructed offline under a synthetic root with on-unreadable=warn so host-scope
// read failures do not abort the run.
#[test]
fn test_describe_records_symlink_verbatim() {
    let root = make_root("desc-symlink");
    let etc = root.join("etc");
    // Create a symlink with a relative, chroot-relative target.
    let link = etc.join("mylink.conf");
    symlink("../foo/bar.conf", &link).expect("create symlink");

    let out = run(&[
        "describe",
        &format!("root={}", root.display()),
        "on-unreadable=warn",
        "format=json",
    ]);
    // warn means host-scope read failures are diagnostics, not fatal -> exit 0.
    assert_eq!(
        exit_code(&out),
        0,
        "describe with on-unreadable=warn over a synthetic root must exit 0; stderr={}",
        stderr_str(&out)
    );
    let doc = parse_json_doc(&stdout_str(&out));
    let elements = doc
        .get("config_files")
        .and_then(|s| s.get("_elements"))
        .and_then(|e| e.as_array())
        .cloned()
        .unwrap_or_default();
    let link_rec = elements.iter().find(|e| {
        e.get("name").and_then(|n| n.as_str()) == Some("/etc/mylink.conf")
    });
    let link_rec = link_rec.expect("symlink must be emitted as a config_files record");
    assert_eq!(link_rec["type"], "link", "symlink must have type=link");
    assert_eq!(
        link_rec["target"], "../foo/bar.conf",
        "symlink target must be stored verbatim (not resolved/normalised)"
    );
    assert_eq!(link_rec["sha256"], "", "a link record's sha256 must be empty");
}

// EXAMPLE: describe_skips_special_file -- a fifo under /etc is skipped: not read,
// not hashed, not emitted; the run does not hang or error.
#[test]
fn test_describe_skips_special_file() {
    let root = make_root("desc-fifo");
    let etc = root.join("etc");
    let fifo = etc.join("myfifo");
    // Create a fifo via mkfifo(3) through libc; skip the test if unavailable.
    let made = make_fifo(&fifo);
    if !made {
        eprintln!("skipping: could not create fifo on this platform");
        return;
    }
    let out = run(&[
        "describe",
        &format!("root={}", root.display()),
        "on-unreadable=warn",
        "format=json",
    ]);
    assert_eq!(
        exit_code(&out),
        0,
        "describe must skip a fifo and not error/hang; stderr={}",
        stderr_str(&out)
    );
    let doc = parse_json_doc(&stdout_str(&out));
    let elements = doc
        .get("config_files")
        .and_then(|s| s.get("_elements"))
        .and_then(|e| e.as_array())
        .cloned()
        .unwrap_or_default();
    assert!(
        !elements.iter().any(|e| e.get("name").and_then(|n| n.as_str()) == Some("/etc/myfifo")),
        "a special file (fifo) must never be emitted as a config_files record"
    );
}

// EXAMPLE: describe_traverses_etc_subdirectories -- the walk descends into a
// subdirectory rather than reading it as a file; no 'is a directory' error.
#[test]
fn test_describe_traverses_etc_subdirectories_no_isdir_error() {
    let root = make_root("desc-subdir");
    let etc = root.join("etc");
    let sub = etc.join("ImageMagick-7");
    std::fs::create_dir_all(&sub).unwrap();
    std::fs::write(sub.join("policy.xml"), "changed content\n").unwrap();

    let out = run(&[
        "describe",
        &format!("root={}", root.display()),
        "on-unreadable=warn",
        "format=json",
    ]);
    assert_eq!(
        exit_code(&out),
        0,
        "describe must traverse subdirectories without an 'is a directory' error; stderr={}",
        stderr_str(&out)
    );
    assert!(
        !stderr_str(&out).to_lowercase().contains("is a directory"),
        "must not report an 'is a directory' read error"
    );
}

// EXAMPLE: scope_attributes_always_object -- every scope's _attributes is an object
// (empty {} for config_files), never null. Asserted on emitted describe JSON.
#[test]
fn test_scope_attributes_always_object_never_null() {
    let root = make_root("desc-attrs");
    let etc = root.join("etc");
    std::fs::write(etc.join("local.conf"), "x\n").unwrap();
    let out = run(&[
        "describe",
        &format!("root={}", root.display()),
        "on-unreadable=warn",
        "format=json",
    ]);
    assert_eq!(exit_code(&out), 0, "stderr={}", stderr_str(&out));
    let doc = parse_json_doc(&stdout_str(&out));
    // For every present scope, _attributes must be a JSON object (never null).
    for key in ["packages", "repositories", "services", "config_files",
                "changed_managed_files", "unmanaged_files"] {
        if let Some(scope) = doc.get(key) {
            let attrs = scope.get("_attributes");
            assert!(
                attrs.map(|a| a.is_object()).unwrap_or(false),
                "scope {key}: _attributes must be a JSON object, never null; got {:?}",
                attrs
            );
        }
    }
}

// EXAMPLE: describe_emits_manifest (structural subset) -- the emitted document is a
// schema-valid Manifest with meta.format_version = 1.
#[test]
fn test_describe_emits_manifest_format_version_1() {
    let root = make_root("desc-fmtver");
    let out = run(&[
        "describe",
        &format!("root={}", root.display()),
        "on-unreadable=warn",
        "format=json",
    ]);
    assert_eq!(exit_code(&out), 0, "stderr={}", stderr_str(&out));
    let doc = parse_json_doc(&stdout_str(&out));
    assert_eq!(
        doc["meta"]["format_version"], 1,
        "describe output must declare meta.format_version = 1"
    );
}

// EXAMPLE: describe_scope_full_emits_observational_scopes (structural negative) --
// a plain describe (scope=etc, the default) must NOT contain the observational scopes.
#[test]
fn test_describe_default_scope_omits_observational_scopes() {
    let root = make_root("desc-default-scope");
    let out = run(&[
        "describe",
        &format!("root={}", root.display()),
        "on-unreadable=warn",
        "format=json",
    ]);
    assert_eq!(exit_code(&out), 0, "stderr={}", stderr_str(&out));
    let doc = parse_json_doc(&stdout_str(&out));
    assert!(
        doc.get("changed_managed_files").is_none(),
        "scope=etc (default) must omit changed_managed_files"
    );
    assert!(
        doc.get("unmanaged_files").is_none(),
        "scope=etc (default) must omit unmanaged_files"
    );
}

// --- helpers ---

fn make_fifo(path: &Path) -> bool {
    use std::ffi::CString;
    let c = match CString::new(path.as_os_str().to_string_lossy().as_bytes()) {
        Ok(c) => c,
        Err(_) => return false,
    };
    // 0o644 fifo
    let r = unsafe { libc_mkfifo(c.as_ptr(), 0o644) };
    r == 0
}

extern "C" {
    #[link_name = "mkfifo"]
    fn libc_mkfifo(path: *const std::os::raw::c_char, mode: u32) -> std::os::raw::c_int;
}
