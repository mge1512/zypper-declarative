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

## config_files: ownership and the pristine/reproducibility rule

This is the highest-risk behaviour; the C++/libzypp build is the behavioural oracle
for it (Go's over-emission was the bug). Verify it during translation using
read-only rpm, and ship the two self-checks below as tests.

- `[spec]` config_files is the changed-from-package and unpackaged `/etc` files,
  excluding package-pristine files, the keep-list, and `/etc/etc.syncpoint`; bounded
  to `/etc` (never read/hash/verify outside it, never a whole-system `rpm -Va`). A
  package verifier returning non-zero because it found changes is the normal result,
  not an unreadable source.
- `[spec]` Determine each path's owning package and recorded baseline (digest, link
  target, mode, owner, group, and file FLAGS including the GHOST bit) via the rpm
  database. Never default a path to unpackaged because a lookup was skipped or
  failed.
- `[spec]` BULK lookup keyed BY PATH, not by row position. `rpm -qf path1 path2 ...`
  does not return one block per input path in order (rpm reorders, deduplicates when
  paths share an owner, drops unowned paths), so a positional zip misaligns owners
  to files. Query the owning packages' file lists, which emit the absolute path on
  every line:
  ```
  rpm -q --queryformat '[%{FILENAMES} %{FILEFLAGS} %{FILEDIGESTS} %{FILELINKTOS} %{FILEMODES} %{FILEUSERNAME} %{FILEGROUPNAME} %{FILEDIGESTALGO}\n]' <pkglist>
  ```
  and build a `path -> attributes` map indexed by that path. (Per-path `rpm -qf` is
  correct but slow, and was the cause of the first Rust build's slowness; batch as
  above.)
- `[spec]` Judge each `/etc` path INDEPENDENTLY against its OWN owning package;
  never collapse a symlink with the file it points to (e.g. `/etc/pam.d/common-auth`
  owned by `pam` vs `common-auth-pc` owned by `pam-config` are separate judgements;
  suppressing the pristine link must not suppress the target); never dereference a
  symlink to judge it. `package_name` is the BARE name (`openssh-server`), never the
  NEVRA `rpm -qf` prints.
- `[spec]` Emission test (reproducibility): emit a path exactly when a fresh install
  of its owning package (or no owning package) would NOT reproduce its on-disk
  state. Concretely:
  - unpackaged -> EMIT;
  - regular file: pristine iff on-disk digest AND mode/owner/group match -> SUPPRESS;
    else EMIT;
  - symlink: pristine iff on-disk target matches the recorded target (mode NOT
    compared) -> SUPPRESS; else EMIT. An owned distro symlink with the package's
    target (the `/etc/X11/xim.d/*/40-ibus` links) is suppressed (the first build
    over-emitted some pristine symlinks);
  - type mismatch (recorded type differs from on-disk type) -> EMIT as the on-disk
    type (`/etc/pam.d/common-auth`: pam ships a regular file, disk has a symlink ->
    emit the link), judged against its own package;
  - ghost (FLAGS has the ghost bit; no shipped content baseline) with real on-disk
    content -> EMIT (`/etc/pam.d/common-auth-pc`, a 0-byte ghost holding the real
    bytes; the v0.6.4 rebuild dropped this, it must be emitted); ghost empty on disk
    with empty baseline -> SUPPRESS. A ghost is never pristine-by-digest.
- `[spec]` Digest comparison is algorithm-aware and normalised: read the recorded
  algorithm (`%{FILEDIGESTALGO}`, 8=SHA256, 1=MD5) and hash the on-disk file with
  the SAME algorithm; compare lowercase, trimmed. An EMPTY recorded digest
  (directories, symlinks, ghosts) is no-baseline, route through the type/ghost rule,
  not a mismatch. The emitted `sha256` is always the real SHA256 of the on-disk file
  regardless of the recorded algorithm.
- `[spec]` A file whose CONTENT cannot be read (a protected file an unprivileged
  reader cannot open) is an `on_unreadable` condition, never silently classified as
  changed-from-package. Distinguish "read the file, digest differs" (emit) from
  "could not read it" (on_unreadable). `rpm -V` itself reads content and trips on
  protected files, so prefer the header-metadata route above for the baseline.
- `[spec]` `created_at` is a real RFC3339 timestamp (a properly converted
  `SystemTime::now()`; the first build emitted `1970-01-01T00:00:36Z`). It is
  informational, excluded from comparison and the hash, but must be correct.
- `[spec]` Two required self-check tests (runnable with read-only rpm during
  translation): (1) ownership resolves a known file to its known package
  (`/etc/ssh/sshd_config` -> `openssh-server`, `/etc/pam.d/common-auth` -> `pam`);
  (2) a known-pristine packaged file (e.g. an `/etc/ImageMagick-7-SUSE/*.xml`) is
  ABSENT from config_files. The first catches a misaligned join; the second catches
  a broken digest comparison.

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

- 2026-06-01: Compressed losslessly from the accreted v0.5.0-v0.6.5 file (same rule
  coverage; post-mortem narration and the per-build changelog diary removed; the
  duplicate do-not-carry list folded into the rules above). Tracks spec v0.6.5:
  reproducibility emission rule (type-mismatch and content-bearing ghost emit,
  empty-ghost suppress), algorithm-aware digest comparison, bulk ownership keyed by
  path, protected-file handling via on_unreadable. Rust-specific preventive lessons
  (path-keyed join, algorithm-aware digest) retained inline, since Rust is exec-based
  and meets both hazards when it batches.
