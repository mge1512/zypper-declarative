# zypper-declarative: translation decisions hints (Rust)

This is the Rust instance of `zypper-declarative.<lang>.decisions.hints.md`, the
decisions hints file from PCD section 13. It is read by a translator during
guided regeneration: a clean rebuild from the specification that honours the
worth-keeping architectural decisions. It is NOT a specification artifact (it does
not affect `pcd-lint`), and it is disposable.

It is the Rust sibling of the Go decisions file: the same architecture, the same
"do NOT carry over" list, retargeted to Rust idiom. The specification itself is
language-neutral; this file is where the Rust-specific decisions live. Read the
spec, the cli-tool template, and `cli-tool.rs.milestones.hints.md` before writing
code.

## Provenance tags

- `[spec]` decided by `zypper-declarative.spec.md`; authoritative here.
- `[pcd]` a documented PCD finding or environment constraint; authoritative.
- `[recommended]` a sound Rust default; apply unless the existing code does
  something equally good.
- `[extract]` read from an existing Rust implementation if one exists; a slot.
- `[changed-N]` the spec version N changed this; follow the new spec.

## How KIT should consume this

Guided regeneration: the translator reads the spec, the template, this file, and
`cli-tool.rs.milestones.hints.md`, and does NOT read any prior buggy code. Use the
"normal" translator prompt (input -> output), not the incremental-from-old-code
prompt. After generation, verify the crate name and the embedded spec hash by
hand.

---

## Decisions to preserve

### Crate and toolchain

- `[spec]` The spec META `Module:` is `github.com/mge1512/zypper-declarative`. For
  Rust this is not a module path; map it to a crate name `zypper-declarative` and
  record the upstream as that URL in `Cargo.toml` `repository`. Do not invent a
  different name.
- `[pcd]` No root at build time. Vendor dependencies with `cargo vendor` and build
  offline (`cargo build --offline` against the vendor directory); do not fetch at
  build time on the target. Use a `GOPATH`-equivalent under the home directory
  only insofar as Cargo's `CARGO_HOME` is set in the user's home.
- `[recommended]` Edition 2021, a recent stable toolchain. Pin the toolchain in
  `rust-toolchain.toml` if reproducibility across build hosts matters (it does for
  EUCC).
- `[pcd]` Dynamic vs static: the spec's deployment template asks for a static
  binary by default; Rust achieves this with `target-feature=+crt-static` against
  glibc (see the Rust milestones hints). Unlike the C++ build (which links
  distro shared libraries), the Rust build has no system library dependency for
  its logic, so a static binary is the natural, low-cost default here.

### Source layout

- `[recommended]` Mirror the spec's behaviour grouping as modules, one concern per
  file. Reconcile with any existing layout and prefer it where at least as clear:

  ```
  src/
    main.rs              entry: build args, dispatch to cli
    cli/mod.rs           dispatch, key=value parsing, global contract
    manifest/mod.rs      data-model types (serde); format; hash
    manifest/format.rs   resolve-format (the single authority)
    manifest/hash.rs     canonical-model hashing
    state/mod.rs         describe-actual-state: the single live reader
    diff/mod.rs          compute-intent-diff, compute-drift (pure)
    converge/mod.rs      converge-packages, -files, -units
    txn/mod.rs           acquire-transaction-context + bindings
    record/mod.rs        load/write applied record
    meta.rs              embedded spec SHA256 and version
  ```

- `[spec]` `describe-actual-state` is the single live-state reader; no other module
  reads the rpmdb, repos.d, systemd, or `/etc` directly. Keep it in `state`.
- `[spec]` `compute-drift` performs no I/O; it compares two in-memory `Manifest`
  values. Keep `diff` free of filesystem/process calls.
- `[spec]` `resolve-format` is the single authority for serialisation choice; route
  every read and write through it. No per-call-site format logic.

### Argument parsing and the global contract

- `[spec]` Options are `key=value`, parsed by the tool; bare words are verbs;
  options precede bare-word arguments BUT must be accepted in any position (the Go
  build had a bug where options after the verb were rejected; do not reproduce
  it). Environment-variable control is forbidden.
- `[spec]` All CONFIG knobs are also accepted as `key=value` options
  (`manifest-format`, `repo-lock`, `content-store`, `keep-list`,
  `signature-verification`, `keyring`, `activation-policy`, `applied-root`,
  `on-unreadable`, `scope`), command-line overriding preset.
- `[spec]` `[changed-0.5.1]` `version` and `help` are bare-word global commands and
  exit 0; `--version`, `--help`, `-h` are tolerated aliases only. No option uses
  POSIX `--flag`. Bare invocation prints usage to stdout and exits 0 (discovery,
  never converges). Unknown verb/option/value or missing value -> usage to stderr,
  exit 2. The cli-tool milestones M0 gate exercises the bare words.
- `[recommended]` Do NOT pull in `clap` or another arg parser; the grammar is
  key=value plus bare verbs, which is a few lines of hand-written parsing and
  avoids a `--flag`-shaped dependency that would fight the spec's CLI style.

### Errors and exit codes

- `[spec]` Internal functions return errors to their caller; exit-code mapping
  lives only in the verb layer (`cli`). Model a `Diagnostic { severity, domain,
  message }` with `domain` in {packages, repositories, services, files, manifest,
  transaction, invocation}. Diagnostics to stderr, one per line; normal output to
  stdout.
- `[recommended]` Use a concrete error enum (or `thiserror`) carrying the domain,
  rather than `Box<dyn Error>`, so the verb layer can map domain -> exit code
  without string matching. Exit codes: 0 success; 1 logical failure (convergence
  failed/discarded, verify drift, invalid/unsafe-YAML/unverified manifest, state
  collection failed); 2 invocation error (bad args, unknown format value,
  unreadable manifest, insufficient privilege, transaction unavailable,
  unwritable output, malformed state dump).

### Manifest data model and serialisation

- `[spec]` The manifest is the declarable Machinery subset (packages, repositories,
  services, config_files), `ScopeWrapper { _attributes, _elements }`, underscore
  field names. JSON canonical, YAML opt-in, same data model. Model scopes with
  serde structs; `#[serde(rename = "_attributes")]` / `"_elements"`.
- `[spec]` Absent vs empty scope is semantic. In Rust, model a declarable scope as
  `Option<Scope>`: `None` = absent (unmanaged), `Some` with empty `_elements` =
  present-empty (reconcile to empty). Do not collapse the two.
- `[spec]` `resolve-format`: explicit `format=` wins, else operative file extension
  (`.json` -> json; `.yaml`/`.yml` -> yaml), else `manifest-format` default.
  Operative path = manifest-path on load, state-path on verify, out on describe.
  `[changed-0.5.0]`/`[changed-0.5.2]` describe output follows
  `resolve-format(out)`; do not hardcode JSON.
- `[spec]` The applied record is ALWAYS canonical JSON regardless of input format.
- `[spec]` `desired_sha256` is the SHA256 of a canonical serialisation of the
  parsed data model (format-independent). `[recommended]` Define canonical
  concretely and apply it for the hash: object keys sorted, compact separators,
  UTF-8, `_elements` sorted by identity key (packages by name+arch, repositories
  by alias, services by name, config_files by path). On-disk `applied.json` may be
  pretty; the HASH is over the compact canonical form. Use the `sha2` crate.
- `[spec]` YAML safe profile: a non-executing loader, no arbitrary/executable tags,
  bounded or disabled anchor/alias expansion, single document only, explicit typing
  per schema (no implicit coercion, e.g. `NO`/`1.10`). `[recommended]` Parse YAML
  into an untyped value first (e.g. `serde_yaml::Value`), walk it to reject tags,
  aliases, and multi-document, then deserialize into the typed structs; record the
  exact crate and version in `TRANSLATION_REPORT.md`. A YAML input needing a
  disabled feature is a manifest error.

### Reading actual state, empty-scope rule, filesystem object model

- `[spec]` `[changed-0.5.2]` Repositories actual state from `/etc/zypp/repos.d/*.repo`
  (INI). No network refresh, no privileged cache. A scope source that cannot be
  read is NEVER an empty scope: `on-unreadable=error` (default) errors naming the
  source, `warn` omits with a diagnostic; genuinely-empty readable scopes are
  omitted so a bootstrap leaves them unmanaged.
- `[spec]` `[changed-0.6.2]` Filesystem object model. Walk `/etc` (and the
  scope=full trees) recursively, classifying each entry by its own type WITHOUT
  following symlinks. In Rust: `std::fs::symlink_metadata` (lstat) and
  `FileType::is_symlink`/`is_dir`/`is_file`; read a symlink target with
  `std::fs::read_link` and store it VERBATIM (no `canonicalize`, no normalisation,
  which also keeps chroot-relative targets correct); hash regular files only
  (`sha2`); descend directories and emit nothing for them (traverse-only); skip
  special files (anything not file/symlink/dir). A directory, symlink, or special
  file is NEVER an unreadable-source error. The original Go crash was reading a
  directory as a file; classify first.
- `[spec]` File records carry a verbatim `target` field (symlink target, "" for
  non-links), sha256 only for regular files, with the TYPES consistency rules.
- `[spec]` In `compute-drift`, type is part of identity: differing type -> modified;
  same type compare sha256 (file) or target (link). A declared entry absent from
  actual is treated as matching.
- `[spec]` Hardlinks: treat per path by content+type; do not detect or preserve
  hardlink identity (out of scope for v1).
- `[spec]` config_files inspection is bounded to `/etc`; never read/hash/verify
  outside it and never run a whole-system verification.
- `[spec]` Difference-reporting is not failure: a package verifier returning
  non-zero because it found changed files is the normal result, not an unreadable
  source.

### Integration with the system (Rust-specific)

- `[recommended]` Drive `zypper`, `snapper`, `systemctl`, and `rpm` by executing
  their command-line interfaces (`std::process::Command`) and parsing their
  output; read repos.d as files. This keeps the Rust binary free of FFI to the
  SUSE C/C++ libraries and lets it stay a static binary. (This is the deliberate
  difference from the C++ build, which links libzypp/libsnapper directly; Rust and
  Go both take the exec route.) `[extract]` If an existing Rust build chose FFI to
  libzypp via bindgen, that is a decision to revisit against the static-binary
  goal, not to preserve by default.
- `[spec]` The transaction binding is abstract (`acquire-transaction-context`
  resolves auto|external|internal); keep it isolated in `txn`. Unit enablement
  under a root uses offline enablement; do not rely on first-boot preset.

### Spec-hash embedding and packaging

- `[spec]` Embed the SHA256 of the spec in: source headers, `--version`/`version`
  output, `TRANSLATION_REPORT.md` (`Spec-SHA256:`), the RPM spec, the DEB control
  (`X-PCD-Spec-SHA256:`), the Containerfile label, and the Makefile/Cargo metadata.
  `[recommended]` Keep version and hash in `meta.rs`, injected at build via a
  `build.rs` that writes a generated constant, or via `env!` of a build-time var.
  `version` prints `zypper-declarative <version> spec:<sha256>`.
- `[spec]` Deliverable surfaced as a zypper subcommand (`/usr/lib/zypper/commands`)
  and invocable directly. OBS package, no curl install. SIGTERM/SIGINT clean exit;
  an interrupted `apply` discards the transaction.

### Testing boundary

- `[pcd]` Black-box tests invoke the built binary via `std::process::Command` and
  assert on stdout, stderr, and exit code; they do NOT call internal functions.
  The v0.5.x/0.6.x examples (bare `version`/`help`, `describe out=...yaml`, offline
  `verify`/`diff`, the `/etc` directory-traversal, symlink-verbatim, special-file
  skip, type-transition drift) are black-box assertions of exactly this kind. Two
  of the filesystem cases (a symlink and a fifo under a synthetic root) are
  constructible offline; cover them rather than leaving them code-review-only.

---

## Do NOT carry these over from the existing code (spec v0.5.0 through v0.6.2)

1. describe output ignoring the `out` extension (now follows `resolve-format`).
2. repositories read in a way that returns empty on read failure, or any path that
   emits an empty scope for an unreadable source (read repos.d; never empty for
   unreadable; omit genuine-empty).
3. bare `version`/`help` returning exit 2 (now the canonical commands, exit 0;
   flags are aliases); bare invocation exiting non-zero (now usage + exit 0).
4. per-call-site format selection (now `resolve-format`).
5. options after the verb being rejected (the Go parser bug); accept key=value in
   any position.
6. treating a package verifier's non-zero "differences found" exit as an unreadable
   source.
7. a whole-system `rpm -Va`; bound to `/etc` (and scope=full trees only under
   scope=full).
8. scope=full engaged by default, or its observational scopes fed into convergence
   or written to the applied record.
9. `verify` requiring an applied record when a reference manifest is supplied, and
   `verify`/`diff` always reading the live system; with `manifest-path`/`state-path`
   they are pure offline comparisons.
10. `apply` accepting a manifest carrying a non-empty observational scope (reject
    it; a raw `describe scope=full` dump is not a baseline).
11. reading a directory (or symlink, or special file) as a regular file, or erroring
    on encountering one; classify by lstat first, traverse dirs, record symlink
    targets verbatim, skip special files.
12. comparing config files by content hash alone; type is part of identity.

## Slots to fill from an existing Rust implementation (if any)

- `[extract]` actual module layout, error enum design, the YAML crate chosen and
  whether it meets the safe profile, the Makefile/OBS packaging, the spec-hash
  injection mechanism, and whether integration was exec-based or FFI.

---

## Changelog

- 2026-06-01: Initial Rust decisions hints, translated from the Go decisions file
  at spec v0.6.2. Same architecture and the same twelve "do NOT carry over" items,
  retargeted to Rust idiom (serde model with Option-typed scopes for absent-vs-empty,
  hand-written key=value parsing, concrete error enum, sha2 canonical hashing,
  serde_yaml Value-walk for the safe profile, std::fs symlink_metadata/read_link
  for the filesystem object model, exec-based system integration and a static
  binary). Library choices are self-contained Cargo crates, so this file has no
  dependency on SLE packaging.
