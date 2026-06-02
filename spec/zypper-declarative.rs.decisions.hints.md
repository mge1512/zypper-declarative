# zypper-declarative: translation decisions hints (Rust)

Rust instance of `zypper-declarative.<lang>.decisions.hints.md` (PCD "When the
Specification Changes"), read by the translator during guided regeneration: a
clean rebuild from the spec that honours the worth-keeping architectural
decisions. Not a spec artifact; does not affect `pcd-lint`; disposable. The Rust
sibling of the Go decisions file: same architecture, retargeted to Rust idiom.
Read the spec, the cli-tool template, and `cli-tool.rs.milestones.hints.md` first.

Tags: `[spec]` decided by the spec; `[pcd]` a PCD/environment constraint;
`[recommended]` a sound Rust default; `[extract]` read from existing Rust code if
present. The translator does NOT read prior buggy code. After generation, verify
the crate name and embedded spec hash by hand.

---

## Crate and toolchain

- `[spec]` Spec META `Module:` is `github.com/mge1512/zypper-declarative`; for Rust
  map it to crate name `zypper-declarative` and record that URL as `Cargo.toml`
  `repository`. Do not invent another name.
- `[pcd]` No root at build time: vendor with `cargo vendor` and build offline
  (`cargo build --offline`); `CARGO_HOME` under the user's home; read-only
  package-DB queries (`rpm -q`/`-qf`/`-qa`/`-ql`/`--queryformat`) ARE available to
  build/test as an ordinary user and should be used to verify config_files during
  translation rather than deferring it.
- `[recommended]` Edition 2021, recent stable toolchain; pin in
  `rust-toolchain.toml` (reproducibility matters for EUCC).
- `[pcd]` Static binary via `target-feature=+crt-static` (see milestones hints).
  Unlike C++, the Rust logic has no system-library dependency, so static is the
  natural low-cost default.

## Source layout

- `[recommended]` One module per concern, mirroring the spec's behaviour grouping;
  reconcile with any existing layout and prefer it where at least as clear:
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
- `[spec]` `describe-actual-state` is the ONLY live-state reader; no other module
  reads the rpmdb, repos.d, systemd, or `/etc` directly. Keep it in `state`.
- `[spec]` `compute-drift` performs no I/O (compares two in-memory `Manifest`
  values); keep `diff` free of filesystem/process calls.
- `[spec]` `resolve-format` is the single authority for serialisation choice; route
  every read and write through it, no per-call-site format logic.

## Argument parsing and the global contract

- `[spec]` Options are `key=value`, parsed by the tool; bare words are verbs;
  options must be accepted in ANY position (the Go build wrongly rejected options
  after the verb; do not reproduce). Environment-variable control is forbidden.
- `[spec]` All CONFIG knobs are also `key=value` options (`manifest-format`,
  `repo-lock`, `content-store`, `keep-list`, `signature-verification`, `keyring`,
  `activation-policy`, `applied-root`, `on-unreadable`, `scope`); command-line
  overrides preset.
- `[spec]` `version` and `help` are bare-word global commands, exit 0; `--version`,
  `--help`, `-h` are tolerated aliases only; no option uses POSIX `--flag`. Bare
  invocation prints usage to stdout, exits 0 (never converges). Unknown
  verb/option/value or missing value -> usage to stderr, exit 2.
- `[recommended]` Do NOT pull in `clap` or another arg parser; the grammar is
  key=value plus bare verbs (a few lines of hand-written parsing) and a `--flag`-
  shaped dependency would fight the spec's CLI style.

## Errors and exit codes

- `[spec]` Internal functions return errors to their caller; exit-code mapping lives
  only in the verb layer (`cli`). Model `Diagnostic { severity, domain, message }`
  with `domain` in {packages, repositories, services, files, manifest, transaction,
  invocation}; diagnostics to stderr one per line, normal output to stdout.
- `[recommended]` Use a concrete error enum (or `thiserror`) carrying the domain,
  not `Box<dyn Error>`, so the verb layer maps domain -> exit code without string
  matching. Exit codes: 0 success; 1 logical failure (convergence failed/discarded,
  verify drift, invalid/unsafe-YAML/unverified manifest, state collection failed);
  2 invocation error (bad args, unknown format value, unreadable manifest,
  insufficient privilege, transaction unavailable, unwritable output, malformed
  state dump).

## Manifest data model and serialisation

- `[spec]` The manifest is the declarable Machinery subset (packages, repositories,
  services, config_files), `ScopeWrapper { _attributes, _elements }`, underscore
  field names; JSON canonical, YAML opt-in, same model. Use serde structs with
  `#[serde(rename = "_attributes")]` / `"_elements"`.
- `[spec]` Absent vs empty scope is semantic: model a declarable scope as
  `Option<Scope>` (`None` = absent/unmanaged; `Some` with empty `_elements` =
  present-empty, reconcile to empty). Do not collapse the two.
- `[spec]` `resolve-format`: explicit `format=` wins, else operative file extension
  (`.json`->json, `.yaml`/`.yml`->yaml), else `manifest-format` default. Operative
  path = manifest-path on load, state-path on verify, out on describe. describe
  output follows `resolve-format(out)`; do not hardcode JSON.
- `[spec]` The applied record is ALWAYS canonical JSON regardless of input format.
- `[spec]` `desired_sha256` is the SHA256 of a canonical serialisation of the parsed
  model (format-independent). `[recommended]` Define canonical concretely: keys
  sorted, compact separators, UTF-8, `_elements` sorted by identity key (packages
  by name+arch, repositories by alias, services by name, config_files by path).
  On-disk `applied.json` may be pretty; the hash is over the compact form. Use the
  `sha2` crate.
- `[spec]` `meta.generator` is `zypper-declarative <version>`, matching other
  implementations of the same spec version (do not drop the version).
- `[spec]` YAML safe profile: non-executing loader, no arbitrary/executable tags,
  bounded or disabled alias expansion, single document only, explicit typing per
  schema (no implicit coercion, e.g. `NO`/`1.10`). `[recommended]` Parse YAML into
  an untyped value first (e.g. `serde_yaml::Value`), walk it to reject tags,
  aliases, and multi-document, then deserialize into typed structs; record crate
  and version in `TRANSLATION_REPORT.md`. A YAML input needing a disabled feature
  is a manifest error.
- `[spec]` On the WRITE side, string-typed fields must serialise as QUOTED YAML
  scalars: `mode: "0600"`, `sha256: "..."`, `target: "..."`, not `mode: 0600`
  (which round-trips as int/octal). Verify by round-tripping a written YAML file
  back through the reader and checking types are preserved.

## Reading actual state, empty-scope rule, filesystem object model

- `[spec]` Repositories actual state from `/etc/zypp/repos.d/*.repo` (INI: alias,
  name, baseurl->url, type, enabled, gpgcheck, autorefresh, priority); no network
  refresh, no privileged cache. A scope source that cannot be read is NEVER an empty
  scope: `on-unreadable=error` (default) errors naming the source, `warn` omits with
  a diagnostic; genuinely-empty readable scopes are omitted so a bootstrap leaves
  them unmanaged.
- `[spec]` Filesystem object model: walk `/etc` (and the scope=full trees)
  recursively, classifying each entry by its own type WITHOUT following symlinks.
  In Rust: `std::fs::symlink_metadata` (lstat) and `FileType::is_symlink`/
  `is_dir`/`is_file`; read a symlink target with `std::fs::read_link` and store it
  VERBATIM (no `canonicalize`, no normalisation; keeps chroot-relative targets
  correct); hash regular files only (`sha2`); descend directories and emit nothing;
  skip special files (anything not file/symlink/dir). A directory, symlink, or
  special file is NEVER an unreadable-source error (the original Go crash was
  reading a directory as a file; classify first). Records carry a verbatim `target`
  field ("" for non-links), sha256 only for regular files, per the TYPES rules.
- `[spec]` In `compute-drift`, type is part of identity: differing type -> modified;
  same type compare sha256 (file) or target (link); a declared entry absent from
  actual is treated as matching. Hardlinks: treat per path by content+type, do not
  detect or preserve hardlink identity.

## config_files: let rpm decide (verdict-parse, NOT a self-built baseline)

config_files is the changed-from-package and unpackaged `/etc` files. The spec
defines the RESULT (the reproducibility emission test); this is the METHOD for Rust.
Do NOT build a `path -> recorded-baseline` map and compare it yourself: let `rpm -V`
do the comparison and parse its verdict. This is how the sister tool `sitar` does
it (working Go and Rust), and it converges where the self-built join did not. The
tool runs as root, so `rpm -V` reads everything. (C++ takes the libzypp route
instead; Rust and Go take this CLI route. All three must reach the same result, the
spec's examples are the shared target and the three-way diff is the check.)

- `[spec]` CHANGED config files (main case): get config-file owning packages with
  `rpm -qca --queryformat '%{NAME}\n'`, dedupe into a set (drop blank lines and
  lines starting with `(`, `error:`, `warning:`), and for each run
  `rpm -V --nodeps --noscript <pkg>`. Non-zero exit is NORMAL (parse regardless);
  it is a package error only when stdout is empty AND stderr is non-empty. Each line
  is `<9 flag chars><space><type><space><path>`; keep only type char `c`. Flags are
  `S M 5 D L U G T P`; `.`/`?` means unchanged; a leading `missing` means deleted.
  Emit the on-disk type: a changed regular file is type "file" with real sha256; an
  `L` flag on a package-shipped file is the TYPE-MISMATCH case (e.g.
  `/etc/pam.d/common-auth`, verify shows `....L....  c ...`), emitted as type "link"
  with the verbatim on-disk target. `package_name` is the BARE name. Every changed
  record MUST carry `status` = "changed" and a non-empty `changes` list built from
  the flags set (`S`->size, `M`->mode, `5`->md5, `D`->device, `L`->link_path,
  `U`->user, `G`->group, `T`->time, `P`->caps; a `missing` line -> includes
  "deleted"); the Go sibling left these null in its first verdict-parse build, do
  not repeat that. No digest map,
  no algorithm handling, no per-path join.
- `[spec]` GHOST REGULAR FILES (the one case `rpm -V` skips): enumerate
  ghost-flagged `/etc` paths (`rpm -qf --queryformat '[%{FILENAMES} %{FILEFLAGS}\n]'`
  or scan owning packages' file lists for FILEFLAGS bit 64). For each ghost REGULAR
  FILE with real on-disk content, EMIT type "file" with real sha256 (e.g.
  `/etc/pam.d/common-auth-pc`); an empty ghost file is suppressed. Small pass over
  the few ghost paths, not a walk of all `/etc`.
- `[spec]` GHOST SYMLINKS (the `/etc/alternatives/*` case): "has content" is NOT the
  test (every symlink has a target); the test is whether the on-disk target equals
  the target a fresh install would establish, for alternatives the auto/best
  provider. Query `update-alternatives --query <name>` (or read
  `/var/lib/alternatives/<name>`): target EQUALS auto/best -> SUPPRESS (drops most of
  `/etc/alternatives/*`); DIFFERS (a manual `--set`) -> EMIT as type "link" with the
  verbatim target. If the alternatives DB cannot be consulted, treat under
  `on_unreadable`, do NOT blanket-emit or blanket-suppress.
- `[spec]` UNPACKAGED files: an `/etc` path no package owns is emitted; find these
  by walking `/etc` and subtracting the rpm-owned path set. Do not mark a file
  unpackaged because a lookup was skipped.
- `[spec]` Exclusions: drop the keep-list and `/etc/etc.syncpoint`; stay bounded to
  `/etc`.
- `[spec]` CONTENT STORE: by default describe is read-only and every `content_ref`
  is "". When `content-store` gives a base path, for each EMITTED regular-file record
  write its bytes to `<content-store>/sha256/<digest>` (idempotent, dedup by content)
  and set `content_ref` to `sha256/<digest>` (same digest as the record's `sha256`).
  Symlinks/dirs keep `content_ref` "". A regular file emitted but unreadable follows
  `on_unreadable` (error, or under warn emit with `content_ref` "" plus a
  diagnostic), never silent. Reference content, never inline it.
- `[spec]` `created_at` is a real RFC3339 timestamp (a properly converted
  `SystemTime::now()`; the first build emitted `1970-01-01T00:00:36Z`),
  informational and excluded from comparison and the hash, but correct.
- `[spec]` Required self-checks (black-box, run as root in the test step): run
  `describe` and assert (1) the pam pair, `common-auth` as type "link",
  `common-auth-pc` as type "file" with a sha256; (2) a known-pristine
  `/etc/ImageMagick-7-SUSE/*.xml` ABSENT; (3) every emitted record with a
  `package_name` that is not an unpackaged file carries `status` == "changed" and a
  non-empty `changes` list. Because `rpm -V` reports only changes, pristine files
  never appear and the over-emission class cannot recur.

## Integration with the system (Rust-specific)

- `[recommended]` Drive `zypper`, `snapper`, `systemctl`, and `rpm` via
  `std::process::Command` and parse their output; read repos.d as files. This keeps
  the binary free of FFI to the SUSE C/C++ libraries and lets it stay static. (The
  deliberate difference from the C++ build, which links libzypp/libsnapper; Rust and
  Go both take the exec route, which preserves the three-way independence.)
  `[extract]` If an existing Rust build chose FFI via bindgen, revisit against the
  static-binary goal rather than preserving it by default.
- `[spec]` The transaction binding is abstract (`acquire-transaction-context`
  resolves auto|external|internal); keep it isolated in `txn`. Unit enablement under
  a root uses offline enablement; do not rely on first-boot preset.
- `[spec]` The `services` scope is MANDATORY: unit ENABLEMENT (enabled/disabled/
  masked), read OFFLINE via `systemctl --root <root>` (the C++ build deferred it to
  an empty scope while Go read ~214; do not regress). Do NOT use a D-Bus/sd-bus API:
  it cannot answer enablement under a non-running root, which the rooted model
  requires. Purely-static units are omitted (not declarable).
- `[spec]` `[reserved-0.7.0]` `converge-files` does NOT yet create/update/remove
  symlinks or handle type transitions (reserved for the apply milestone). When
  implemented: a declared type "link" is converged by its target; a declared-vs-
  actual type mismatch at a path is a HARD ERROR that aborts the transaction (no
  silent destructive rewrite).

## Cross-implementation consistency (the three-way oracle)

- `[spec]` The three implementations are a continuous consistency check: Rust
  `describe` output on a host should match Go and C++ after canonicalisation; any
  divergence is a bug in one of them, arbitrated by the spec. For the comparison to
  be clean, scopes, field presence, type classification, scalar typing, and ordering
  must agree: sort `_elements` by identity key, emit `_attributes` as `{}` never
  null, and exclude `meta.created_at` (a per-run timestamp) from the comparison.
  This is the diff that caught Go's pristine mislabelling, which neither build's own
  tests caught.

## Spec-hash embedding and packaging

- `[spec]` Embed the spec SHA256 in source headers, `--version`/`version` output,
  `TRANSLATION_REPORT.md` (`Spec-SHA256:`), the RPM spec, the DEB control
  (`X-PCD-Spec-SHA256:`), the Containerfile label, and the Makefile/Cargo metadata.
  `[recommended]` Keep version and hash in `meta.rs`, injected via a `build.rs` that
  writes a generated constant (or `env!` of a build-time var); `version` prints
  `zypper-declarative <version> spec:<sha256>`.
- `[spec]` Surfaced as a zypper subcommand (`/usr/lib/zypper/commands`) and invocable
  directly; OBS package, no curl install; SIGTERM/SIGINT clean exit; an interrupted
  `apply` discards the transaction.

## Testing boundary

- `[pcd]` Black-box tests invoke the built binary via `std::process::Command` and
  assert on stdout, stderr, and exit code; they do NOT call internal functions. The
  config_files self-checks above and the v0.5.x/0.6.x examples (bare `version`/
  `help`, `describe out=...yaml`, offline `verify`/`diff`, `/etc` directory
  traversal, symlink-verbatim, special-file skip, type-transition drift) are
  black-box assertions; the symlink and fifo cases are constructible under a
  synthetic root, cover them rather than leaving them code-review-only.

## Changelog

- 2026-06-02: Tracks spec v0.6.6. Split ghost handling into ghost FILES (emit if
  on-disk content) and ghost SYMLINKS (`/etc/alternatives/*`: suppress when the link
  equals the alternatives auto/best target via `update-alternatives --query`, emit a
  manually-set link). Added content-store population: when `content-store` is set,
  describe writes emitted regular-file bytes content-addressed by sha256 and sets
  `content_ref`; read-only otherwise.
- 2026-06-01: Switched config_files from a self-built recorded-baseline map to
  `rpm -V` verdict-parsing, the method the sister tool sitar uses (working Go and
  Rust) and which converges; the self-built join failed repeatedly in the Go
  sibling. rpm does the comparison; the code parses the `SM5DLUGTP` flag string
  (type char `c`). Type-mismatch comes free from the `L` flag; content-bearing
  `%ghost` files are a tiny separate pass. The tool runs as root, so rpm -V reads
  everything. C++ keeps the libzypp route; Rust and Go share this CLI route; the
  spec (result, not method) is unchanged and the three-way diff remains the check.
