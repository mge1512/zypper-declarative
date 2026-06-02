#![allow(dead_code)]
// tests by: claude-opus-4-8
// generated from spec: zypper-declarative.spec.md sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
//
// describe behaviour tests (black-box), driven against a synthetic root so the
// results are deterministic and do not depend on the host's package database.
//
// Under a synthetic root, files placed in <root>/etc are unpackaged (no owning
// package on the host owns a path under the temp tree), so describe must emit
// them as type-classified records with package_name "". This lets us assert
// the filesystem object model (type classification, verbatim symlink targets,
// special-file skip, directory traversal, sha256 for regular files) and the
// serialisation rules (format resolution, _attributes always an object,
// genuinely-empty-scope omission, content store) without root or a real rpmdb.

include!("common.rs");

use serde_json::Value;

fn write_file(root: &std::path::Path, rel: &str, content: &[u8]) {
    let p = root.join(rel);
    std::fs::create_dir_all(p.parent().unwrap()).unwrap();
    std::fs::write(&p, content).unwrap();
}

fn sha256_hex(bytes: &[u8]) -> String {
    // Minimal SHA-256 so the test does not depend on the implementation's hash.
    // Pure-Rust reference implementation.
    let mut h: [u32; 8] = [
        0x6a09e667, 0xbb67ae85, 0x3c6ef372, 0xa54ff53a, 0x510e527f, 0x9b05688c, 0x1f83d9ab,
        0x5be0cd19,
    ];
    const K: [u32; 64] = [
        0x428a2f98, 0x71374491, 0xb5c0fbcf, 0xe9b5dba5, 0x3956c25b, 0x59f111f1, 0x923f82a4,
        0xab1c5ed5, 0xd807aa98, 0x12835b01, 0x243185be, 0x550c7dc3, 0x72be5d74, 0x80deb1fe,
        0x9bdc06a7, 0xc19bf174, 0xe49b69c1, 0xefbe4786, 0x0fc19dc6, 0x240ca1cc, 0x2de92c6f,
        0x4a7484aa, 0x5cb0a9dc, 0x76f988da, 0x983e5152, 0xa831c66d, 0xb00327c8, 0xbf597fc7,
        0xc6e00bf3, 0xd5a79147, 0x06ca6351, 0x14292967, 0x27b70a85, 0x2e1b2138, 0x4d2c6dfc,
        0x53380d13, 0x650a7354, 0x766a0abb, 0x81c2c92e, 0x92722c85, 0xa2bfe8a1, 0xa81a664b,
        0xc24b8b70, 0xc76c51a3, 0xd192e819, 0xd6990624, 0xf40e3585, 0x106aa070, 0x19a4c116,
        0x1e376c08, 0x2748774c, 0x34b0bcb5, 0x391c0cb3, 0x4ed8aa4a, 0x5b9cca4f, 0x682e6ff3,
        0x748f82ee, 0x78a5636f, 0x84c87814, 0x8cc70208, 0x90befffa, 0xa4506ceb, 0xbef9a3f7,
        0xc67178f2,
    ];
    let mut msg = bytes.to_vec();
    let bitlen = (bytes.len() as u64) * 8;
    msg.push(0x80);
    while msg.len() % 64 != 56 {
        msg.push(0);
    }
    msg.extend_from_slice(&bitlen.to_be_bytes());
    for chunk in msg.chunks(64) {
        let mut w = [0u32; 64];
        for i in 0..16 {
            w[i] = u32::from_be_bytes([
                chunk[i * 4],
                chunk[i * 4 + 1],
                chunk[i * 4 + 2],
                chunk[i * 4 + 3],
            ]);
        }
        for i in 16..64 {
            let s0 = w[i - 15].rotate_right(7) ^ w[i - 15].rotate_right(18) ^ (w[i - 15] >> 3);
            let s1 = w[i - 2].rotate_right(17) ^ w[i - 2].rotate_right(19) ^ (w[i - 2] >> 10);
            w[i] = w[i - 16]
                .wrapping_add(s0)
                .wrapping_add(w[i - 7])
                .wrapping_add(s1);
        }
        let mut a = h;
        for i in 0..64 {
            let s1 = a[4].rotate_right(6) ^ a[4].rotate_right(11) ^ a[4].rotate_right(25);
            let ch = (a[4] & a[5]) ^ ((!a[4]) & a[6]);
            let t1 = a[7]
                .wrapping_add(s1)
                .wrapping_add(ch)
                .wrapping_add(K[i])
                .wrapping_add(w[i]);
            let s0 = a[0].rotate_right(2) ^ a[0].rotate_right(13) ^ a[0].rotate_right(22);
            let maj = (a[0] & a[1]) ^ (a[0] & a[2]) ^ (a[1] & a[2]);
            let t2 = s0.wrapping_add(maj);
            a[7] = a[6];
            a[6] = a[5];
            a[5] = a[4];
            a[4] = a[3].wrapping_add(t1);
            a[3] = a[2];
            a[2] = a[1];
            a[1] = a[0];
            a[0] = t1.wrapping_add(t2);
        }
        for i in 0..8 {
            h[i] = h[i].wrapping_add(a[i]);
        }
    }
    let mut out = String::new();
    for v in h {
        out.push_str(&format!("{:08x}", v));
    }
    out
}

fn parse_json(s: &str) -> Value {
    serde_json::from_str(s).unwrap_or_else(|e| panic!("describe stdout was not valid JSON: {}\n{}", e, s))
}

fn config_files_elements(doc: &Value) -> Vec<Value> {
    doc.get("config_files")
        .and_then(|s| s.get("_elements"))
        .and_then(|e| e.as_array())
        .cloned()
        .unwrap_or_default()
}

fn find_record<'a>(elems: &'a [Value], name: &str) -> Option<&'a Value> {
    elems.iter().find(|e| e.get("name").and_then(|n| n.as_str()) == Some(name))
}

// EXAMPLE: describe_emits_manifest (structural subset) + format_version=1
#[test]
fn describe_emits_json_manifest_with_format_version_1() {
    let root = temp_dir("emit");
    write_file(&root, "etc/myapp/app.conf", b"hello world\n");
    let root_arg = format!("root={}", root.display());
    let r = run_str(&["describe", &root_arg]);
    assert_eq!(r.code, 0, "describe must exit 0; stderr={}", r.stderr);
    let doc = parse_json(&r.stdout);
    assert_eq!(
        doc.pointer("/meta/format_version").and_then(|v| v.as_i64()),
        Some(1),
        "meta.format_version must be 1"
    );
    let gen = doc.pointer("/meta/generator").and_then(|v| v.as_str()).unwrap_or("");
    assert!(
        gen.starts_with("zypper-declarative "),
        "meta.generator must be 'zypper-declarative <version>'; got {:?}",
        gen
    );
}

// EXAMPLE: scope_attributes_always_object
#[test]
fn scope_attributes_always_object() {
    let root = temp_dir("attrs");
    write_file(&root, "etc/myapp/app.conf", b"data\n");
    let root_arg = format!("root={}", root.display());
    let r = run_str(&["describe", &root_arg]);
    assert_eq!(r.code, 0, "stderr={}", r.stderr);
    let doc = parse_json(&r.stdout);
    // For every scope present, _attributes must be a JSON object, never null.
    for key in ["packages", "repositories", "services", "config_files"] {
        if let Some(scope) = doc.get(key) {
            let attrs = scope.get("_attributes");
            assert!(
                attrs.map(|a| a.is_object()).unwrap_or(false),
                "{}._attributes must be a JSON object, got {:?}",
                key,
                attrs
            );
        }
    }
}

// EXAMPLE: describe_records_symlink_verbatim
#[test]
fn describe_records_symlink_verbatim() {
    let root = temp_dir("symlink");
    std::fs::create_dir_all(root.join("etc/foo")).unwrap();
    let link = root.join("etc/foo/link.conf");
    std::os::unix::fs::symlink("../bar/baz.conf", &link).unwrap();
    let root_arg = format!("root={}", root.display());
    let r = run_str(&["describe", &root_arg]);
    assert_eq!(r.code, 0, "stderr={}", r.stderr);
    let doc = parse_json(&r.stdout);
    let elems = config_files_elements(&doc);
    let rec = find_record(&elems, "/etc/foo/link.conf")
        .expect("symlink must be emitted as a config_files record");
    assert_eq!(rec.get("type").and_then(|v| v.as_str()), Some("link"));
    assert_eq!(
        rec.get("target").and_then(|v| v.as_str()),
        Some("../bar/baz.conf"),
        "symlink target must be stored verbatim"
    );
    assert_eq!(
        rec.get("sha256").and_then(|v| v.as_str()),
        Some(""),
        "a link record carries sha256 \"\""
    );
}

// EXAMPLE: describe_skips_special_file
#[test]
fn describe_skips_special_file() {
    let root = temp_dir("fifo");
    std::fs::create_dir_all(root.join("etc")).unwrap();
    let fifo = root.join("etc/myfifo");
    let cpath = std::ffi::CString::new(fifo.to_str().unwrap()).unwrap();
    let rc = unsafe { libc_mkfifo(cpath.as_ptr(), 0o644) };
    if rc != 0 {
        // If we cannot create a FIFO in this environment, skip the assertion
        // body but do not fail spuriously.
        eprintln!("skip: could not create fifo");
        return;
    }
    // also a normal file so the scope is non-empty
    write_file(&root, "etc/real.conf", b"x\n");
    let root_arg = format!("root={}", root.display());
    let r = run_str(&["describe", &root_arg]);
    assert_eq!(r.code, 0, "run must not error on a special file; stderr={}", r.stderr);
    let doc = parse_json(&r.stdout);
    let elems = config_files_elements(&doc);
    assert!(
        find_record(&elems, "/etc/myfifo").is_none(),
        "special file (fifo) must be skipped, not emitted"
    );
}

extern "C" {
    #[link_name = "mkfifo"]
    fn libc_mkfifo(path: *const std::os::raw::c_char, mode: u32) -> std::os::raw::c_int;
}

// EXAMPLE: describe_traverses_etc_subdirectories
#[test]
fn describe_traverses_subdirectories() {
    let root = temp_dir("subdir");
    write_file(&root, "etc/ImageMagick-7/policy.xml", b"<policymap/>\n");
    let root_arg = format!("root={}", root.display());
    let r = run_str(&["describe", &root_arg]);
    assert_eq!(r.code, 0, "no 'is a directory' error; stderr={}", r.stderr);
    let doc = parse_json(&r.stdout);
    let elems = config_files_elements(&doc);
    let rec = find_record(&elems, "/etc/ImageMagick-7/policy.xml")
        .expect("file inside subdirectory must be emitted");
    assert_eq!(rec.get("type").and_then(|v| v.as_str()), Some("file"));
}

// Regular file record carries a real sha256 of its content.
#[test]
fn describe_regular_file_sha256() {
    let root = temp_dir("sha");
    let content = b"some configuration body\n";
    write_file(&root, "etc/app.conf", content);
    let root_arg = format!("root={}", root.display());
    let r = run_str(&["describe", &root_arg]);
    assert_eq!(r.code, 0, "stderr={}", r.stderr);
    let doc = parse_json(&r.stdout);
    let elems = config_files_elements(&doc);
    let rec = find_record(&elems, "/etc/app.conf").expect("regular file must be emitted");
    assert_eq!(rec.get("type").and_then(|v| v.as_str()), Some("file"));
    assert_eq!(
        rec.get("sha256").and_then(|v| v.as_str()),
        Some(sha256_hex(content).as_str()),
        "sha256 must be the SHA-256 of the file content"
    );
    assert_eq!(
        rec.get("target").and_then(|v| v.as_str()),
        Some(""),
        "a file record carries target \"\""
    );
}

// EXAMPLE: describe_without_content_store_is_readonly
#[test]
fn describe_without_content_store_is_readonly() {
    let root = temp_dir("nocs");
    write_file(&root, "etc/app.conf", b"body\n");
    let root_arg = format!("root={}", root.display());
    let r = run_str(&["describe", &root_arg]);
    assert_eq!(r.code, 0, "stderr={}", r.stderr);
    let doc = parse_json(&r.stdout);
    let elems = config_files_elements(&doc);
    let rec = find_record(&elems, "/etc/app.conf").unwrap();
    assert_eq!(
        rec.get("content_ref").and_then(|v| v.as_str()),
        Some(""),
        "without content-store, content_ref must be \"\""
    );
}

// EXAMPLE: describe_populates_content_store
#[test]
fn describe_populates_content_store() {
    let root = temp_dir("cs-root");
    let store = temp_dir("cs-store");
    let content = b"changed sshd config\n";
    write_file(&root, "etc/ssh/sshd_config", content);
    let digest = sha256_hex(content);
    let root_arg = format!("root={}", root.display());
    let cs_arg = format!("content-store={}", store.display());
    let r = run_str(&["describe", &root_arg, &cs_arg]);
    assert_eq!(r.code, 0, "stderr={}", r.stderr);
    let doc = parse_json(&r.stdout);
    let elems = config_files_elements(&doc);
    let rec = find_record(&elems, "/etc/ssh/sshd_config").unwrap();
    assert_eq!(
        rec.get("content_ref").and_then(|v| v.as_str()),
        Some(format!("sha256/{}", digest).as_str()),
        "content_ref must be sha256/<digest>"
    );
    let blob = store.join("sha256").join(&digest);
    assert!(blob.exists(), "blob must be written into the content store at {:?}", blob);
    assert_eq!(
        std::fs::read(&blob).unwrap(),
        content,
        "stored blob bytes must equal the file content"
    );
    // second describe must not error and must keep the same blob (idempotent dedup)
    let r2 = run_str(&["describe", &root_arg, &cs_arg]);
    assert_eq!(r2.code, 0, "second describe must also succeed");
    assert!(blob.exists(), "blob still present after second describe");
}

// EXAMPLE: describe_out_extension_yaml + describe_out_extension_json
#[test]
fn describe_out_extension_selects_format() {
    let root = temp_dir("ext");
    write_file(&root, "etc/app.conf", b"x\n");
    let root_arg = format!("root={}", root.display());

    let d = temp_dir("ext-out");
    let yaml = d.join("state.yaml");
    let json = d.join("state.json");

    let ry = run_str(&["describe", &root_arg, &format!("out={}", yaml.display())]);
    assert_eq!(ry.code, 0, "describe out=.yaml must exit 0; stderr={}", ry.stderr);
    let ytxt = std::fs::read_to_string(&yaml).unwrap();
    assert!(
        !ytxt.trim_start().starts_with('{'),
        "out=.yaml must contain a YAML document, not JSON; got first chars: {:?}",
        &ytxt.chars().take(40).collect::<String>()
    );

    let rj = run_str(&["describe", &root_arg, &format!("out={}", json.display())]);
    assert_eq!(rj.code, 0, "describe out=.json must exit 0; stderr={}", rj.stderr);
    let jtxt = std::fs::read_to_string(&json).unwrap();
    let _: Value = serde_json::from_str(&jtxt).expect("out=.json must contain JSON");
}

// EXAMPLE: describe_format_overrides_extension
#[test]
fn describe_format_overrides_extension() {
    let root = temp_dir("override");
    write_file(&root, "etc/app.conf", b"x\n");
    let root_arg = format!("root={}", root.display());
    let d = temp_dir("override-out");
    let out = d.join("state.yaml");
    let r = run_str(&[
        "describe",
        &root_arg,
        "format=json",
        &format!("out={}", out.display()),
    ]);
    assert_eq!(r.code, 0, "stderr={}", r.stderr);
    let txt = std::fs::read_to_string(&out).unwrap();
    let _: Value = serde_json::from_str(&txt)
        .expect("explicit format=json overrides the .yaml extension -> JSON content");
}

// EXAMPLE: describe_format_yaml (stdout YAML)
#[test]
fn describe_format_yaml_stdout() {
    let root = temp_dir("yaml-stdout");
    write_file(&root, "etc/app.conf", b"x\n");
    let root_arg = format!("root={}", root.display());
    let r = run_str(&["describe", &root_arg, "format=yaml"]);
    assert_eq!(r.code, 0, "stderr={}", r.stderr);
    assert!(
        !r.stdout.trim_start().starts_with('{'),
        "format=yaml stdout must be a YAML document, not JSON; got: {:?}",
        &r.stdout.chars().take(40).collect::<String>()
    );
    // YAML must NOT round-trip mode as an int: a string field like mode must be quoted.
    // We assert there is at least a recognizable YAML mapping key from the model.
    assert!(
        r.stdout.contains("meta") || r.stdout.contains("format_version"),
        "YAML output should render the data model"
    );
}

// EXAMPLE: describe_config_files_bounded_to_etc / unpackaged emission
#[test]
fn describe_unpackaged_etc_file_has_empty_package_name() {
    let root = temp_dir("unpackaged");
    write_file(&root, "etc/local.conf", b"local\n");
    let root_arg = format!("root={}", root.display());
    let r = run_str(&["describe", &root_arg]);
    assert_eq!(r.code, 0, "stderr={}", r.stderr);
    let doc = parse_json(&r.stdout);
    let elems = config_files_elements(&doc);
    let rec = find_record(&elems, "/etc/local.conf")
        .expect("an unpackaged /etc file must be emitted");
    assert_eq!(
        rec.get("package_name").and_then(|v| v.as_str()),
        Some(""),
        "an unpackaged file has package_name \"\""
    );
}

// EXAMPLE: describe_omits_genuinely_empty_scope
// A readable but empty /etc must not produce an empty config_files scope.
#[test]
fn describe_omits_genuinely_empty_config_files_scope() {
    let root = temp_dir("empty-etc");
    std::fs::create_dir_all(root.join("etc")).unwrap(); // empty, readable
    let root_arg = format!("root={}", root.display());
    let r = run_str(&["describe", &root_arg]);
    assert_eq!(r.code, 0, "stderr={}", r.stderr);
    let doc = parse_json(&r.stdout);
    if let Some(scope) = doc.get("config_files") {
        let elems = scope.get("_elements").and_then(|e| e.as_array());
        assert!(
            elems.map(|e| !e.is_empty()).unwrap_or(true),
            "a genuinely-empty config_files scope must be omitted, not emitted with empty _elements"
        );
    }
}

// EXAMPLE: describe_scope_full_emits_observational_scopes (presence under full,
// absence under etc) — asserted structurally against a synthetic root.
#[test]
fn describe_scope_etc_has_no_observational_scopes() {
    let root = temp_dir("scope-etc");
    write_file(&root, "etc/app.conf", b"x\n");
    let root_arg = format!("root={}", root.display());
    let r = run_str(&["describe", &root_arg, "scope=etc"]);
    assert_eq!(r.code, 0, "stderr={}", r.stderr);
    let doc = parse_json(&r.stdout);
    assert!(
        doc.get("changed_managed_files").is_none(),
        "scope=etc must not emit changed_managed_files"
    );
    assert!(
        doc.get("unmanaged_files").is_none(),
        "scope=etc must not emit unmanaged_files"
    );
}
