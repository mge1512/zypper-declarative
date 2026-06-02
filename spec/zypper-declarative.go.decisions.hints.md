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

## config_files: let rpm decide (verdict-parse, NOT a self-built baseline)

config_files is the changed-from-package and unpackaged `/etc` files. The spec
defines the RESULT (the reproducibility emission test); this is the METHOD for Go,
and the method matters: do NOT build a `path -> recorded-baseline` map and compare
it yourself. That join failed repeatedly. Instead let `rpm -V` do the comparison
and parse its verdict (this is how the sister tool `sitar` does it, and it works).
The tool runs as root, so `rpm -V` can read everything.

- `[spec]` CHANGED config files (the main case): get the config-file owning
  packages with `rpm -qca --queryformat '%{NAME}\n'`, dedupe into a package set
  (drop blank lines and lines starting with `(`, `error:`, or `warning:`), and for
  each package run `rpm -V --nodeps --noscript <pkg>`. `rpm -V` exits non-zero when
  it finds differences: that is NORMAL, parse the output regardless of exit code;
  treat it as a package error ONLY when stdout is empty AND stderr is non-empty.
  Each verify line is `<9 flag chars><space><type><space><path>`; keep only lines
  whose type char is `c`. The 9 flags are `S M 5 D L U G T P` (size, mode, md5,
  device, link, user, group, time, caps); `.` or `?` means unchanged. A line
  beginning `missing` means the file is deleted. Emit the on-disk type: a changed
  REGULAR file is type "file" with its real sha256; an `L` flag (link differs) on a
  path the package shipped as a file is the TYPE-MISMATCH case (e.g.
  `/etc/pam.d/common-auth`, verify shows `....L....  c ...`) and is emitted as type
  "link" with the verbatim on-disk target. `package_name` is the BARE name. Every
  changed record MUST carry `status` = "changed" and a non-empty `changes` list
  built from the flags actually set (`S`->size, `M`->mode, `5`->md5, `D`->device,
  `L`->link_path, `U`->user, `G`->group, `T`->time, `P`->caps; a `missing` line ->
  `changes` includes "deleted"); build 12 decided emission from the flags correctly
  but left `changes` and `status` null, do not repeat that. This is the whole
  changed-files mechanism: no digest map, no algorithm handling, no per-path join.
- `[spec]` GHOST REGULAR FILES are a SEPARATE, REQUIRED pass, do NOT skip it. `rpm
  -V` does not report `%ghost` files at all, so they never appear in the verdict
  parse and must be found by their own enumeration. (The Rust sibling omitted this
  pass and dropped all 32 content-bearing ghosts, the `common-*-pc` PAM files,
  `/etc/machine-id`, `/etc/ld.so.cache`, `/etc/hostname`, `/etc/hosts`,
  `/etc/crypto-policies/*`, ...; do not repeat that.) The pass: enumerate
  ghost-flagged paths under `/etc` with `rpm -qf --queryformat '[%{FILENAMES}
  %{FILEFLAGS}\n]' <path>` (or scan installed packages' file lists for the ghost bit,
  FILEFLAGS bit 64), and for each ghost REGULAR FILE with real on-disk content
  (exists and is non-empty) EMIT type "file" with its real sha256, `package_name`
  set, `status` "changed", and `changes` listing the ghost-content reason (e.g.
  `/etc/pam.d/common-auth-pc`). An empty ghost file is suppressed. Tiny pass over the
  few ghost paths, NOT a walk of all `/etc`, but mandatory: the changed-files result
  is the UNION of the `rpm -V` verdict parse and this ghost pass.
- `[spec]` GHOST SYMLINKS (the `/etc/alternatives/*` case): "has content" is NOT the
  test (every symlink has a target); the test is whether the on-disk target equals
  the target a fresh install would establish. For alternatives that is the auto/best
  provider, query it with `update-alternatives --query <name>` (or read
  `/var/lib/alternatives/<name>`), which reports both the current link and the
  auto/best choice. If the on-disk target EQUALS the auto/best target -> SUPPRESS
  (pristine; this drops the bulk of `/etc/alternatives/*`); if it DIFFERS (a manual
  `update-alternatives --set`) -> EMIT as type "link" with the verbatim target. If
  the alternatives DB cannot be consulted for a given ghost symlink, treat it under
  `on_unreadable`, do NOT blanket-emit or blanket-suppress.
- `[spec]` UNPACKAGED files: a path under `/etc` that no package owns is emitted as
  unpackaged. Find these by walking `/etc` and subtracting the rpm-owned path set
  (the file lists of installed packages); do not mark a file unpackaged just because
  a lookup was skipped.
- `[spec]` Exclusions: drop the keep-list and `/etc/etc.syncpoint`. Stay bounded to
  `/etc`.
- `[spec]` CONTENT STORE: by default describe is read-only and every `content_ref`
  is "". When the `content-store` option gives a base path, for each EMITTED
  regular-file record write its bytes to `<content-store>/sha256/<digest>`
  (idempotent: skip if that digest blob already exists, dedup by content) and set the
  record's `content_ref` to `sha256/<digest>` (the same digest as the record's
  `sha256`). Symlinks/dirs keep `content_ref` "". A regular file emitted but
  unreadable follows `on_unreadable` (error, or under warn emit with `content_ref`
  "" plus a diagnostic), never silent. The manifest references content, never inlines
  it.
- `[spec]` Required self-checks (black-box, run as root in the test step), each MUST
  actually run and fail the build if unmet: run `describe` and assert (1a)
  `/etc/pam.d/common-auth` present as type "link"; (1b) `/etc/pam.d/common-auth-pc`
  present as type "file" with a non-empty sha256 (the CONTENT-BEARING GHOST, this
  binds the separate ghost pass); (1c) at least one other content-bearing ghost,
  e.g. `/etc/machine-id`, present as a type-file record; (2) a known-pristine file
  (an `/etc/ImageMagick-7-SUSE/*.xml`) ABSENT; (3) every emitted record that has a
  `package_name` and is NOT an unpackaged file carries `status` == "changed" and a
  non-empty `changes` list (binds the field build 12 dropped). Because `rpm -V`
  reports only changes, pristine files never appear and the over-emission class
  cannot recur; (1b)/(1c) ensure the ghost pass is not silently missing.

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
  In Go, find `unmanaged_files` by walking and subtracting the rpm-owned path set;
  find `changed_managed_files` the same verdict-parse way as config_files but with
  `rpm -Va --nodeps --noscript` (verify all), keeping the NON-config lines (type
  char is not `c`) and parsing the same flag string. Do not build a digest baseline
  map. Scope keys are underscore_style.

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

- 2026-06-02: Tracks spec v0.6.6. Split ghost handling into ghost FILES (emit if
  on-disk content) and ghost SYMLINKS (the `/etc/alternatives/*` case: suppress when
  the link equals the alternatives auto/best target, emit a manually-set link),
  querying `update-alternatives --query`. Added content-store population: when
  `content-store` is set, describe writes emitted regular-file bytes content-addressed
  by sha256 and sets `content_ref`; read-only otherwise.
- 2026-06-01: Switched config_files (and changed_managed_files) from a self-built
  recorded-baseline map to `rpm -V`/`rpm -Va` verdict-parsing, the method the sister
  tool sitar uses and which converges; the self-built join failed repeatedly in Go.
  rpm does the comparison; the code parses the `SM5DLUGTP` flag string (type char
  `c` for config). Type-mismatch comes free from the `L` flag; the one case rpm -V
  does not cover, content-bearing `%ghost` files, is a tiny separate pass over
  ghost-flagged `/etc` paths. The tool runs as root, so rpm -V reads everything.
  Spec unchanged (it defines the result, not the method).
