#![allow(dead_code)]
// tests by: claude-opus-4-8
// generated from spec: zypper-declarative.spec.md sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
//
// Required config_files self-checks (black-box). These bind the spec's
// reproducibility emission test on a REAL host package database; per the Rust
// decisions hints they "MUST actually run and fail the build if unmet" when the
// test step runs as root against the live system. They are the assertions that
// catch a missing or mis-targeted ghost pass: build 04 (Go) ran the `rpm -V`
// verdict parse but omitted the separate %ghost pass and silently dropped every
// content-bearing ghost; assertions (1b)/(1c) below bind that pass so the same
// regression cannot recur in Rust.
//
// Because `rpm -V` reports only differences (pristine files never appear), the
// over-emission class cannot recur either; assertion (2) confirms a known
// pristine file is absent.
//
// These run `describe` against the live root ("/") and therefore require root
// (to read the rpmdb and all of /etc) AND a SUSE host with the relevant
// packages installed. When those preconditions are absent the test SKIPS rather
// than fails spuriously: it is a binding self-check on the host where the
// translator's test step runs as root, and an explicit, logged no-op elsewhere.

include!("common.rs");

use serde_json::Value;

fn is_root() -> bool {
    // Effective uid 0. Avoids a libc dependency by reading /proc.
    // `id -u` is the simplest portable probe through the same exec boundary.
    match std::process::Command::new("id").arg("-u").output() {
        Ok(o) => String::from_utf8_lossy(&o.stdout).trim() == "0",
        Err(_) => false,
    }
}

fn host_has(path: &str) -> bool {
    std::path::Path::new(path).exists()
}

fn live_describe_config_files() -> Vec<Value> {
    // Read the live declarable config_files actual state. on-unreadable=error is
    // the default; as root every /etc source is readable, so the run exits 0.
    let r = run_str(&["describe", "scope=etc"]);
    assert_eq!(
        r.code, 0,
        "describe scope=etc on the live root must exit 0 as root; stderr={}",
        r.stderr
    );
    let doc: Value = serde_json::from_str(&r.stdout)
        .unwrap_or_else(|e| panic!("describe stdout was not valid JSON: {}\n{}", e, r.stdout));
    doc.get("config_files")
        .and_then(|s| s.get("_elements"))
        .and_then(|e| e.as_array())
        .cloned()
        .unwrap_or_default()
}

fn find<'a>(elems: &'a [Value], name: &str) -> Option<&'a Value> {
    elems
        .iter()
        .find(|e| e.get("name").and_then(|n| n.as_str()) == Some(name))
}

fn str_field<'a>(rec: &'a Value, key: &str) -> &'a str {
    rec.get(key).and_then(|v| v.as_str()).unwrap_or("")
}

// (1a) /etc/pam.d/common-auth present as type "link" — the TYPE-MISMATCH case
//      (pam ships a regular %config(noreplace) file; pam-config replaced it with
//      a symlink to common-auth-pc).
#[test]
fn selfcheck_common_auth_is_type_link() {
    if !is_root() {
        eprintln!("skip selfcheck_common_auth_is_type_link: not root");
        return;
    }
    if !host_has("/etc/pam.d/common-auth") {
        eprintln!("skip: /etc/pam.d/common-auth not present on this host");
        return;
    }
    let elems = live_describe_config_files();
    let rec = find(&elems, "/etc/pam.d/common-auth")
        .expect("(1a) /etc/pam.d/common-auth must be present (type-mismatch case)");
    assert_eq!(
        str_field(rec, "type"),
        "link",
        "(1a) /etc/pam.d/common-auth must be emitted as type \"link\" with its verbatim target"
    );
    assert!(
        !str_field(rec, "target").is_empty(),
        "(1a) the type-link record must carry the verbatim on-disk target"
    );
}

// (1b) /etc/pam.d/common-auth-pc present as type "file" with a non-empty sha256
//      — the CONTENT-BEARING GHOST that binds the separate ghost pass. pam-config
//      ships it as a 0-byte %ghost %config; on disk it holds the real ~462-byte
//      PAM configuration, so a fresh install would NOT reproduce it and it must
//      be emitted with that content and digest. Build 04 dropped this.
#[test]
fn selfcheck_common_auth_pc_content_bearing_ghost() {
    if !is_root() {
        eprintln!("skip selfcheck_common_auth_pc_content_bearing_ghost: not root");
        return;
    }
    if !host_has("/etc/pam.d/common-auth-pc") {
        eprintln!("skip: /etc/pam.d/common-auth-pc not present on this host");
        return;
    }
    let elems = live_describe_config_files();
    let rec = find(&elems, "/etc/pam.d/common-auth-pc").expect(
        "(1b) /etc/pam.d/common-auth-pc must be present — the content-bearing ghost; \
         its absence means the separate ghost pass is missing",
    );
    assert_eq!(
        str_field(rec, "type"),
        "file",
        "(1b) common-auth-pc must be a type \"file\" record"
    );
    assert!(
        str_field(rec, "sha256").len() == 64,
        "(1b) common-auth-pc must carry a non-empty (64-hex) content sha256; got {:?}",
        str_field(rec, "sha256")
    );
}

// (1c) at least one OTHER content-bearing ghost is present, guarding against the
//      ghost pass being special-cased to pam only. /etc/machine-id is a systemd
//      %ghost with real on-disk content.
#[test]
fn selfcheck_other_content_bearing_ghost_present() {
    if !is_root() {
        eprintln!("skip selfcheck_other_content_bearing_ghost_present: not root");
        return;
    }
    if !host_has("/etc/machine-id") {
        eprintln!("skip: /etc/machine-id not present on this host");
        return;
    }
    let elems = live_describe_config_files();
    let rec = find(&elems, "/etc/machine-id").expect(
        "(1c) /etc/machine-id must be present as a content-bearing ghost — guards \
         against the ghost pass being special-cased to pam only",
    );
    assert_eq!(
        str_field(rec, "type"),
        "file",
        "(1c) /etc/machine-id must be a type \"file\" record"
    );
    assert!(
        str_field(rec, "sha256").len() == 64,
        "(1c) /etc/machine-id must carry a content sha256"
    );
}

// (2) a known-pristine /etc/ImageMagick-7-SUSE/*.xml is ABSENT (suppressed).
//     Because rpm -V reports only changes, pristine files never appear; this
//     confirms the over-emission class does not recur.
#[test]
fn selfcheck_pristine_imagemagick_xml_absent() {
    if !is_root() {
        eprintln!("skip selfcheck_pristine_imagemagick_xml_absent: not root");
        return;
    }
    let dir = std::path::Path::new("/etc/ImageMagick-7-SUSE");
    if !dir.is_dir() {
        eprintln!("skip: /etc/ImageMagick-7-SUSE not present on this host");
        return;
    }
    let elems = live_describe_config_files();
    let leaked: Vec<&str> = elems
        .iter()
        .filter_map(|e| e.get("name").and_then(|n| n.as_str()))
        .filter(|n| n.starts_with("/etc/ImageMagick-7-SUSE/") && n.ends_with(".xml"))
        .collect();
    assert!(
        leaked.is_empty(),
        "(2) pristine /etc/ImageMagick-7-SUSE/*.xml must be suppressed, but these \
         were emitted: {:?}",
        leaked
    );
}

// (3) every emitted record that carries a package_name (i.e. not an unpackaged
//     file) must carry status == "changed" and a NON-EMPTY changes list. The Go
//     sibling left these null in its first verdict-parse build; this assertion
//     prevents that regression.
#[test]
fn selfcheck_packaged_records_carry_status_and_changes() {
    if !is_root() {
        eprintln!("skip selfcheck_packaged_records_carry_status_and_changes: not root");
        return;
    }
    let elems = live_describe_config_files();
    let mut offenders: Vec<String> = Vec::new();
    for rec in &elems {
        let pkg = str_field(rec, "package_name");
        if pkg.is_empty() {
            continue; // unpackaged: package_name "" is correct, no status/changes required
        }
        let status_ok = str_field(rec, "status") == "changed";
        let changes_nonempty = rec
            .get("changes")
            .and_then(|c| c.as_array())
            .map(|a| !a.is_empty())
            .unwrap_or(false);
        if !status_ok || !changes_nonempty {
            offenders.push(format!(
                "{} (pkg={}, status={:?}, changes={:?})",
                str_field(rec, "name"),
                pkg,
                rec.get("status"),
                rec.get("changes"),
            ));
        }
    }
    assert!(
        offenders.is_empty(),
        "(3) every packaged record must carry status=\"changed\" and a non-empty \
         changes list; offenders: {:#?}",
        offenders
    );
}
