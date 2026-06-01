# zypper-declarative: translation decisions hints (Go)

Decisions hints (PCD "When the Specification Changes"), read by the translator
during guided regeneration: a clean rebuild from the spec that honours the
worth-keeping architectural decisions. Not a spec artifact; does not affect
`pcd-lint`; disposable. Read the spec and the cli-tool template first, then this.

Tags: `[spec]` decided by the spec (authoritative); `[pcd]` a PCD/environment
constraint; `[recommended]` a sound default; `[extract]` read from existing code
if present. The translator reads the spec, the template, and this file, and does
NOT read old code, so prior bugs are not carried over. Fix the Go module path by
hand after generation (it is a known translator gap).

---

## Module and toolchain

- `[spec]` `go.mod` module line is exactly `github.com/mge1512/zypper-declarative`.
  Verify and fix this post-generation before any push (systematic translator gap).
- `[extract]` Go version floor: the version on the SLES 16.1 / OBS build host; pin
  it in `go.mod`, do not invent one.
- `[pcd]` No root at build time: module downloads run as the current user with
  `GOPATH`/`GOCACHE` under home; vendor with `go mod vendor`; install no system
  packages. Read-only package-DB queries (`rpm -q`/`-qf`/`-qa`/`-ql`/`--queryformat`)
  ARE available to the build/test as an ordinary user and should be used to verify
  the config_files behaviour during translation rather than deferring it.

## Source layout

- `[recommended]` One internal package per concern so each spec behaviour has an
  obvious home:
  ```
  cmd/zypper-declarative/main.go   thin entry: build args, call internal/cli
  internal/cli/                    dispatch, key=value parsing, global contract
  internal/manifest/               data model; json+yaml (de)serialise;
                                   resolve-format; canonical-model hashing
  internal/state/                  describe-actual-state: the single live reader
  internal/diff/                   compute-intent-diff, compute-drift (pure)
  internal/converge/               converge-packages, -files, -units
  internal/txn/                    acquire-transaction-context + bindings
  internal/record/                 load/write applied record
  internal/meta/                   embedded spec SHA256 and version
  ```
- `[spec]` `describe-actual-state` is the ONLY code that reads live system state;
  every verb obtains actual state through it (or a supplied dump). No other package
  reads the rpmdb, repos.d, systemd, or `/etc` directly. Keep it in `internal/state`.
- `[spec]` `compute-drift` performs no I/O; keep `internal/diff` free of filesystem,
  rpmdb, and process calls (pure comparison of two in-memory `Manifest` documents).
- `[spec]` `resolve-format` is the single authority for choosing a serialisation;
  put it in `internal/manifest` and route every read and write through it.

## Argument parsing and the global contract

- `[spec]` Options are `key=value`, parsed by the tool; bare words are verbs;
  options precede bare-word arguments; environment-variable control is forbidden.
- `[spec]` All CONFIG knobs are also `key=value` options (`manifest-format`,
  `repo-lock`, `content-store`, `keep-list`, `signature-verification`, `keyring`,
  `activation-policy`, `applied-root`); a command-line option overrides the preset.
- `[spec]` Global behaviour:
  - bare invocation (no verb) prints usage to stdout, exits 0 (never runs a verb);
  - `version` and `help` are bare-word commands (the canonical form): `version`
    prints program name, version, and embedded spec hash to stdout, exits 0; `help`
    prints usage to stdout, exits 0;
  - `--version`, `--help`, `-h` are tolerated aliases for those two only; no option
    uses POSIX `--flag` style;
  - unknown verb / option / value, or a missing required value: usage to stderr,
    exit 2.

## Error and exit-code convention

- `[spec]` Internal behaviours return `error` (or a typed `Diagnostic`) to their
  caller; only `internal/cli`/`main` map to an exit code.
- `[spec]` Diagnostics carry `severity`, `domain`
  (`packages|repositories|files|units|manifest|transaction|invocation`), and
  `message`; one per line on stderr. Normal output (summaries, diff plan, status
  report, describe document) goes to stdout.
- `[spec]` Exit codes: 0 success (converged, no-op, matches declaration, or
  describe emitted); 1 logical failure (convergence failed and discarded; verify
  drift; invalid/unsafe-YAML/unverified manifest; state collection failed);
  2 invocation error (bad args; unknown format value; manifest unreadable;
  insufficient privilege; transaction unavailable; output path unwritable;
  malformed state dump).

## Manifest data model and serialisation

- `[spec]` The manifest is one typed data model (the declarable Machinery subset:
  packages, repositories, services, config_files; `ScopeWrapper`
  `{_attributes, _elements}`; underscore_style field names). JSON and YAML are
  serialisations of it; keep the Go structs as the single model, json/yaml as edges.
  Every struct in JSON output needs explicit `json:` tags (underscore_style keys).
- `[spec]` Canonical serialisation is JSON (`format_version` 1). The applied record
  is ALWAYS canonical JSON regardless of the desired manifest's input format.
- `[spec]` Each scope's `_attributes` serialises as a JSON object, empty `{}` when
  it has no attributes, NEVER `null`. In YAML, quote string-typed scalars
  (`mode: "0600"`, `sha256`, `target`).
- `[spec]` `resolve-format` precedence: explicit `format=`, else the operative file
  extension (`.json`->json, `.yaml`/`.yml`->yaml), else the `manifest-format`
  default. Operative path is `manifest-path` on load, `state-path` on verify, `out`
  on describe; stdin/stdout with no explicit format use the default. `describe`
  output follows `resolve-format(out)`; do not hardcode JSON.
- `[spec]` `desired_sha256` is the SHA256 of a canonical serialisation of the
  parsed model, format-independent (JSON and YAML of the same manifest hash equal).
  `[recommended]` Define canonical concretely: keys sorted, compact separators,
  UTF-8, `_elements` sorted by identity key (packages by name+arch, repositories by
  alias, services by name, config_files by path). On-disk `applied.json` may be
  pretty-printed, but the hash is over the canonical compact form. Sorting
  `_elements` also makes describe output deterministic and diffable.
- `[spec]` `meta.generator` is `zypper-declarative <version>`, so independent
  implementations of the same spec version emit the same string.
- `[spec]` YAML safe profile (only when YAML enabled): non-code-executing loader,
  no arbitrary/executable tags, bounded or disabled alias expansion, single
  document, explicit typing per schema (no implicit coercion such as `NO`->false,
  `1.10`->float). A YAML input needing any disabled feature is a manifest error.
  `[recommended]` One robust Go route: convert YAML to JSON and decode with
  `encoding/json` `DisallowUnknownFields`; whichever route, it must meet every
  safe-profile constraint; record the library and config in `TRANSLATION_REPORT.md`.

## Reading actual state and the empty-scope rule

- `[spec]` Repositories actual state is read from `<root>/etc/zypp/repos.d/*.repo`
  (INI: alias, name, baseurl->`RepositoryRecord.url`, type, enabled, gpgcheck,
  autorefresh, priority). Not via a network refresh or privileged cache (those
  files are world-readable).
- `[spec]` A scope source that cannot be read is NEVER an empty `_elements`. Under
  `on-unreadable=error` (default) return an error naming the source; under
  `on-unreadable=warn` omit the scope and emit a diagnostic. A genuinely-empty
  readable scope is OMITTED from describe (so a bootstrapped manifest leaves it
  unmanaged). `describe` passes its `on_unreadable` option through; every other
  caller passes `on_unreadable=error`.

## config_files: ownership and the pristine/reproducibility rule

This scope is the changed-from-package and unpackaged `/etc` files, excluding
package-pristine files, the keep-list, and `/etc/etc.syncpoint`. `content_ref` is
empty in actual state. This is the highest-risk behaviour; verify it during
translation using read-only rpm (available to the build), and ship the two
self-checks below as tests.

- `[spec]` Bound the work to `/etc`: enumerate `/etc` and consult package metadata
  only for those paths. Do not read, hash, or verify anything outside `/etc`, and
  do not run a whole-system verification (`rpm -Va`). A package verifier exiting
  non-zero (it does so when it finds changed files) is the normal result, not an
  unreadable source.
- `[spec]` Determine each path's owning package and its package-recorded baseline
  (digest, link target, mode, owner, group, and the file FLAGS including the GHOST
  bit). Never default a path to unpackaged because a lookup was skipped.
- `[spec]` BULK lookup, keyed BY PATH not by row position. `rpm -qf path1 path2 ...`
  does not return one block per input path in order (rpm reorders, deduplicates
  when paths share an owner, and drops unowned paths), so a positional zip
  misaligns owners to files. Instead, query the owning packages' file lists, which
  emit the absolute path on every line:
  ```
  rpm -q --queryformat '[%{FILENAMES} %{FILEFLAGS} %{FILEDIGESTS} %{FILELINKTOS} %{FILEMODES} %{FILEUSERNAME} %{FILEGROUPNAME} %{FILEDIGESTALGO}\n]' <pkglist>
  ```
  and build a `path -> {package, flags, digest, algo, linkto, mode, owner, group}`
  map indexed by that path; look up each `/etc` path in the map. Resolve ownership
  with `rpm -qf` first if needed, tying each owner to its queried path, never
  zipping. (Per-path `rpm -qf` is correct but slow; batch as above.)
- `[spec]` Judge each `/etc` path INDEPENDENTLY against its OWN owning package.
  Never collapse a symlink with the file it points to: suppressing a pristine
  symlink must not suppress its target, which is judged separately against its own
  owner (often a different package, e.g. `/etc/pam.d/common-auth` owned by `pam`
  vs `common-auth-pc` owned by `pam-config`). Never dereference a symlink to judge
  it.
- `[spec]` `package_name` is the BARE name (`openssh-server`), never the NEVRA
  (`rpm -qf` prints NEVRA; reduce it).
- `[spec]` Emission test (reproducibility): emit a path exactly when a fresh
  install of its owning package (or no owning package) would NOT reproduce its
  on-disk state. Concretely:
  - unpackaged (no owner) -> EMIT;
  - regular file: pristine iff on-disk digest AND mode/owner/group match the
    recorded baseline -> SUPPRESS; else EMIT;
  - symlink: pristine iff on-disk target matches the recorded target (mode NOT
    compared) -> SUPPRESS; else EMIT. An owned distro symlink with the package's
    target (the `/etc/X11/xim.d/*/40-ibus` links) is suppressed;
  - type mismatch (recorded type differs from on-disk type, e.g. recorded a regular
    file, disk has a symlink) -> EMIT as the on-disk type, judged against its own
    package;
  - ghost (FLAGS has the ghost bit; no shipped content baseline) with real on-disk
    content -> EMIT (a fresh install ships no content; e.g. `/etc/pam.d/common-auth-pc`,
    a 0-byte ghost holding the real bytes); ghost empty on disk with empty baseline
    -> SUPPRESS. A ghost is never pristine-by-digest.
- `[spec]` Digest comparison is algorithm-aware and normalised: read the recorded
  algorithm (`%{FILEDIGESTALGO}`, 8=SHA256, 1=MD5) and hash the on-disk file with
  the SAME algorithm; compare lowercase, trimmed. An EMPTY recorded digest
  (directories, symlinks, ghosts) is no-baseline, route through the type/ghost rule,
  not a mismatch. The emitted `sha256` is always the real SHA256 of the on-disk
  file regardless of the recorded algorithm.
- `[spec]` A file whose CONTENT cannot be read (a protected file an unprivileged
  reader cannot open) is an `on_unreadable` condition, never silently classified as
  changed-from-package. Distinguish "read the file, digest differs" (emit) from
  "could not read the file" (on_unreadable). Note `rpm -V` itself reads content and
  trips on protected files, so prefer the header-metadata route above for the
  baseline.
- `[spec]` Two required self-check tests (runnable with read-only rpm during
  translation): (1) ownership resolves a known file to its known package
  (`/etc/ssh/sshd_config` -> `openssh-server`); (2) a known-pristine packaged file
  (e.g. an `/etc/ImageMagick-7-SUSE/*.xml`) is ABSENT from config_files. The first
  catches a misaligned join; the second catches a broken digest comparison.

## Filesystem object model (the /etc and full-scan walks)

- `[spec]` Recurse into directories and classify each entry by its own type using
  lstat (do NOT follow symlinks, do NOT read a path before classifying). In Go:
  `filepath.WalkDir` or a manual stack with `os.Lstat`; symlink
  (`d.Type()&fs.ModeSymlink`) -> `os.Readlink`, store verbatim (no `Abs`/`Clean`);
  regular file -> hash; directory -> descend, emit nothing; device/fifo/socket ->
  skip. A directory, symlink, or special file is never an unreadable-source error.
  Records carry a `target` field (verbatim symlink target, "" for non-links).
  Hardlinks: treat per path by content+type, do not detect or preserve hardlink
  identity.
- `[spec]` In `compute-drift`, type is part of identity: differing type -> modified;
  same type compares sha256 (file) or target (link).

## Full-scan integrity (scope=full)

- `[spec]` On `describe` and `verify` only; default `etc` scans nothing outside
  `/etc`. Under `full`, scan the package-managed trees (`/usr`, the usr-merge roots
  `/bin` `/sbin` `/lib` `/lib64`, `/boot`; exclude `/opt` and the virtual/runtime/
  mutable-data trees; do not cross unlisted mounts; honour the keep-list) and emit
  two observational scopes: `changed_managed_files` (packaged files outside `/etc`
  differing from baseline, with a `changes` list) and `unmanaged_files` (files no
  package owns). Observational: do NOT feed them to `compute-intent-diff` or
  convergence, and never write them to the applied record; `compute-drift` surfaces
  them under `scope=full` as `managed_files_modified` and `unmanaged_files_present`.
  In Go, find additions by walking and subtracting the rpmdb-owned path set, and
  modifications by comparing packaged digests to the baseline; do not run
  whole-system `rpm -Va`. Scope keys are underscore_style.

## Integration with the system (Go-specific)

- `[recommended]` Drive `zypper`, `snapper`, `systemctl`, and `rpm` via `os/exec`
  and parse their output, rather than binding libzypp via cgo; this keeps
  `CGO_ENABLED=0` and the single static binary. Repositories are read as files (no
  exec). `[extract]` If existing code used cgo/libzypp, revisit against the
  static-binary goal rather than preserving it by default.
- `[spec]` The transaction binding is abstract: `acquire-transaction-context`
  resolves `auto|external|internal` and returns a context with a writable `root`
  and `opened_here`; the convergence path is identical regardless. Keep the binding
  isolated in `internal/txn`.
- `[spec]` Unit enablement under a root uses offline enablement
  (`systemctl --root <ctx.root> ...`) for `converge-units`, and a query for
  `describe-actual-state`; do not rely on first-boot preset evaluation.
- `[spec]` `[reserved-0.7.0]` `converge-files` does NOT yet create/update/remove
  symlinks or handle type transitions (reserved for the apply milestone). When
  implemented: a declared type "link" is converged by its target; a declared-vs-
  actual type mismatch at a path is a HARD ERROR that aborts the transaction (no
  silent destructive rewrite).

## Spec-hash embedding and provenance

- `[spec]` Embed the SHA256 of `zypper-declarative.spec.md` in every produced
  artifact: source headers, `--version` output, `TRANSLATION_REPORT.md`
  (`Spec-SHA256:`), the RPM spec comment, the DEB control `X-PCD-Spec-SHA256:`, the
  Containerfile label, the Makefile variable. `[recommended]` Keep hash and version
  in `internal/meta`, injected via `-ldflags -X` or `go:embed`. `--version` prints
  `zypper-declarative <version> spec:<sha256>`. `created_at` is a real RFC3339
  timestamp (e.g. `time.Now()`), informational and excluded from the hash/comparison
  but present and correct.

## Build and packaging

- `[spec]` Single static binary (`CGO_ENABLED=0`; final container stage
  `FROM scratch`), no runtime deps of its own, surfaced as a zypper subcommand (an
  executable in `/usr/lib/zypper/commands`) and invocable directly.
- `[spec]` Installed via an OBS package on build.opensuse.org; no curl-based
  installation.
- `[spec]` Signal handling: clean exit on SIGTERM/SIGINT; an interrupted `apply`
  discards the transaction and leaves no new snapshot as the default boot target.
  Document the approach in `TRANSLATION_REPORT.md`.

## Testing boundary

- `[pcd]` Tests are black-box: invoke the built binary through the DEPLOYMENT
  interface via `os/exec` and assert on stdout, stderr, and exit code. Tests must
  NOT call internal Go functions or simulate the binary. The config_files
  self-checks above and the v0.5.0 examples (bare invocation, unknown verb,
  `describe out=...yaml`, `verify state-path=...yaml`, unreadable and genuinely-empty
  repositories) are black-box assertions of exactly this kind.

## Changelog

- 2026-06-01: Compressed losslessly from the accreted v0.5.0-v0.6.5 file (same rule
  coverage; post-mortem narration and the per-build changelog diary removed; the
  duplicate do-not-carry list folded into the rules above). Tracks spec v0.6.5:
  reproducibility emission rule (type-mismatch and content-bearing ghost emit,
  empty-ghost suppress), algorithm-aware digest comparison, bulk ownership keyed by
  path, protected-file handling via on_unreadable.
