# zypper-declarative: translation decisions hints (Go)

This is the decisions hints file from PCD section 13 ("When the Specification
Changes"). It is read by a translator during **guided regeneration**: a clean
rebuild from the specification that nonetheless honours the worth-keeping
architectural decisions of the prior implementation. It is NOT a specification
artifact: it does not affect `pcd-lint`, and it is disposable.

## Naming and language

This file is the Go instance, named `zypper-declarative.go.decisions.hints.md`.
The generic name is `zypper-declarative.<lang>.decisions.hints.md`; the qualifier
is the implementation language and the file is discarded when the language
changes. If the language is later chosen to be Rust or C++, rename the qualifier
(`.rust.`, `.cpp.`) and re-read the language-specific sections below, which are
written for Go with substitution notes. The specification itself stays
language-neutral; this file is where the language-specific decisions live.

## Provenance of the entries below

Normally this file is produced by the translator (via `assess_change_impact` or
`prompts/change-impact.md`) by reading the existing code. This copy is
hand-authored as a starting point, so each entry is tagged:

- `[spec]` decided by `zypper-declarative.spec.md` v0.5.0; authoritative here.
- `[pcd]` a documented PCD finding or environment constraint; authoritative here.
- `[recommended]` a sound default to apply unless the existing code already does
  something equally good; reconcile against `/tmp/pcd-original/code/`.
- `[extract]` must be read from the existing code; left as a slot to fill, since
  the existing implementation is not visible to the author of this file.
- `[changed-0.5.0]` the v0.5.0 spec changed this; do NOT carry the old code's
  behaviour over, follow the new spec.

## How KIT should consume this (important)

This file supports **guided regeneration**, which is the middle of the PCD
three-state model. The translator must read the spec, the template, and this
file, and must NOT read the old code (so the prior bugs cannot be carried over).
That means:

- Use your "normal" translator prompt (input `/tmp/pcd-input/`, output
  `/tmp/pcd-output/`), NOT the existing-code prompt that mounts
  `/tmp/pcd-original/code/`. The existing-code prompt is for incremental update,
  which is the mode we deliberately did not choose for v0.5.0.
- Place into `/tmp/pcd-input/` alongside the v0.5.0 spec and the cli-tool
  template: this file, `ROLE.md` declaring translator mode, and `prompt.md`.
- Ensure the translator flow in `prompt.md` reads `*.decisions.hints.md` from the
  input directory. If `prompt.md` only supports clean-from-scratch, either add a
  step that reads this file, or accept clean-from-scratch (in which case this file
  is human onboarding, not a translator input; PCD says it is ignored on clean
  full regeneration).
- After generation, fix the Go module path by hand (see Module path below); the
  PCD record notes this is a systematic translator gap.

---

## Decisions to preserve

### Module and toolchain

- `[spec]` Go module path is `github.com/mge1512/zypper-declarative` (META
  `Module`). Set the `go.mod` `module` line to exactly this.
- `[pcd]` Module path is a systematic translator gap and cannot be inferred from
  the spec by the translator reliably. Verify and fix it post-generation before
  any push.
- `[extract]` Go language version floor for `go.mod`. Use what the existing code
  used, or, if starting clean, the Go available on the SLES 16.1 / OBS build host;
  pin it in `go.mod`. Do not invent a version.
- `[pcd]` No root at build time. Module downloads run as the current user with
  `GOPATH`/`GOCACHE` under the home directory. Vendor dependencies with
  `go mod vendor`. Do not install system packages.

### Source layout

- `[recommended]` Group code to mirror the spec's behaviour grouping, one internal
  package per concern, so each spec behaviour maps to an obvious home. Reconcile
  with the layout already in `/tmp/pcd-original/code/` and prefer the existing one
  where it is at least as clear:

  ```
  cmd/zypper-declarative/main.go     thin entry: build args, call internal/cli
  internal/cli/                      dispatch, key=value parsing, global contract
  internal/manifest/                 data-model types; json+yaml (de)serialise;
                                     resolve-format; canonical-model hashing
  internal/state/                    describe-actual-state: the single live reader
  internal/diff/                     compute-intent-diff, compute-drift (pure)
  internal/converge/                 converge-packages, -files, -units
  internal/txn/                      acquire-transaction-context + bindings
  internal/record/                   load/write applied record
  internal/meta/                     embedded spec SHA256 and version
  ```

- `[spec]` `describe-actual-state` is the single live-state reader: it is the only
  code that reads live system state. `describe`, `diff`, `verify`, `status`, and
  the post-converge check in `apply` all obtain actual state through it (or a
  supplied dump). Keep it in one package (`internal/state`) and do not let any
  other package read the rpmdb, repos.d, systemd, or `/etc` directly.
- `[spec]` `compute-drift` performs no I/O; it compares two in-memory `Manifest`
  documents. Keep `internal/diff` free of filesystem, rpmdb, and process calls.
- `[changed-0.5.0]` `resolve-format` is a new shared behaviour and the single
  authority for choosing a serialisation. Put it in `internal/manifest` and route
  every read and write through it. Remove any per-call-site inline format logic
  the old code had in the manifest loader.

### Argument parsing and the global contract

- `[spec]` Options are `key=value`, parsed by the tool itself; bare words are
  verbs. Options precede bare-word arguments. Control via environment variables is
  forbidden.
- `[spec]` All CONFIG knobs are also accepted as `key=value` options
  (`manifest-format`, `repo-lock`, `content-store`, `keep-list`,
  `signature-verification`, `keyring`, `activation-policy`, `applied-root`), with
  a command-line option overriding the corresponding preset value. The prior help
  text already exposed these; keep them.
- `[changed-0.5.0]` `[changed-0.5.1]` Global behaviour, do NOT reuse the old "no
  verb -> exit 2", and do NOT make the flags the only form:
  - bare invocation (no verb) prints usage to stdout and exits 0 (discovery, never
    runs a default verb, never converges);
  - `version` and `help` are bare-word global commands (the canonical form, per the
    cli-tool template: bare-words supported, POSIX `--flag` forbidden for new
    options): `version` prints program name, version, and embedded spec hash to
    stdout and exits 0; `help` prints usage to stdout and exits 0;
  - `--version`, `--help`, and `-h` are tolerated aliases for those two commands
    only; no option uses POSIX `--flag` style (options are key=value only);
  - unknown verb, unknown option, unknown value, or missing required value prints
    usage to stderr, exits 2.
  - The cli-tool milestones-hints M0 gate exercises the bare words
    (`<binary> version`, `<binary> help` must exit 0); make sure both pass, not
    just the flag forms.

### Error and exit-code convention

- `[spec]` Internal behaviours return errors to their caller; they do not exit.
  Exit-code mapping lives only in the verb layer (`internal/cli`). In Go terms:
  internal packages return `error` (or a typed `Diagnostic`), and only
  `internal/cli`/`main` translate that to an exit code.
- `[spec]` Diagnostics carry `severity`, `domain`
  (`packages|repositories|files|units|manifest|transaction|invocation`), and
  `message`. Write them to stderr, one per line. Normal output (summaries, the
  diff plan, the status report, the describe document) goes to stdout.
- `[spec]` Exit codes: 0 success (converged, no-op, matches declaration, or
  describe emitted); 1 logical failure (convergence failed and discarded; verify
  drift; invalid, unsafe-YAML, or unverified manifest; state collection failed);
  2 invocation error (bad args; unknown format value; manifest unreadable;
  insufficient privilege; transaction mechanism unavailable; output path
  unwritable; malformed state dump).
- `[extract]` The concrete Go error type or sentinel-error pattern the existing
  code used for carrying `domain`. Preserve it if it was clean.

### Manifest data model and serialisation

- `[spec]` The manifest is a typed data model (the declarable Machinery subset:
  packages, repositories, services, config_files, with the `ScopeWrapper`
  `{_attributes, _elements}` idiom and underscore_style field names). JSON and
  YAML are serialisations of that model. Keep the Go structs as the single model
  and treat json/yaml strictly as edges.
- `[spec]` Canonical serialisation is JSON (`format_version` 1). The applied
  record is ALWAYS written as canonical JSON regardless of the desired manifest's
  input format.
- `[spec]` `resolve-format` precedence: explicit `format=` option, else the
  operative file extension (`.json` -> json; `.yaml`/`.yml` -> yaml), else the
  `manifest-format` default. The operative path is `manifest-path` on load,
  `state-path` on verify, and `out` on describe. stdin/stdout with no explicit
  format use the default.
- `[changed-0.5.0]` `describe` output format follows `resolve-format(out)`. Do NOT
  hardcode JSON output. `describe out=...yaml` must write YAML; `out=...json` must
  write JSON; an explicit `format=` overrides the extension.
- `[spec]` Manifest identity `desired_sha256` is the SHA256 of a canonical
  serialisation of the parsed data model, format-independent, so JSON and YAML of
  the same manifest hash identically.
- `[recommended]` Define "canonical" concretely for the hash and apply it
  consistently: object keys sorted, compact separators, UTF-8, scope `_elements`
  sorted by their identity key (packages by name+arch, repositories by alias,
  services by name, config_files by path). The on-disk `applied.json` may be
  pretty-printed for readability, but the HASH is taken over the canonical compact
  form. Sorting `_elements` before both serialising and hashing also makes
  describe output deterministic and diffable.
- `[spec]` YAML safe profile (only when YAML is enabled): a non-code-executing
  loader, no arbitrary or executable tags, bounded or disabled anchor/alias
  expansion, single-document streams only, explicit typing per the schema (no
  implicit YAML coercion such as `NO` -> false or `1.10` -> float). A YAML input
  needing any disabled feature is rejected with a manifest error.
- `[recommended]` In Go, the translator selects a YAML approach that satisfies the
  safe profile and records the exact library and configuration in
  `TRANSLATION_REPORT.md`. One robust route is to convert YAML to JSON and decode
  with `encoding/json` using `DisallowUnknownFields`, which rejects YAML-only
  constructs and uses JSON typing; whichever route is chosen, it must demonstrably
  meet every safe-profile constraint above. Do not pick a loader that executes
  tags or expands aliases without a bound.

### Reading actual state and the empty-scope rule

- `[changed-0.5.0]` Repositories actual state is read from the on-disk
  `<root>/etc/zypp/repos.d/*.repo` files (INI sections: alias, name, baseurl
  mapped to `RepositoryRecord.url`, type, enabled, gpgcheck, autorefresh,
  priority). Do NOT read repositories via a network refresh or a privileged cache.
  This is what fixed the empty-`repositories` bug: those files are world-readable
  in the normal case.
- `[changed-0.5.0]` A scope source that cannot be read is NEVER represented as an
  empty `_elements`. Under `on-unreadable=error` (default) return an error naming
  the source; under `on-unreadable=warn` omit the affected scope and emit a
  diagnostic. A genuinely-empty readable scope is OMITTED from describe output, so
  a bootstrapped manifest leaves it unmanaged rather than asserting deletion.
- `[spec]` The `describe` verb passes `on_unreadable` through from its option;
  every other caller (apply, diff, status, verify reading live state) passes
  `on_unreadable=error`.
- `[spec]` `[changed-0.5.2]` `[changed-0.6.3]` `config_files` actual state is the
  changed-from-package and unpackaged `/etc` files, excluding package-pristine
  files, the keep-list, and `/etc/etc.syncpoint`. `package_name` is populated from
  rpm; `content_ref` is empty in actual state. Bound the work to `/etc`: do not
  read, hash, or verify anything outside `/etc`, and do not run a whole-system
  verification such as `rpm -Va` (it is the cause of the slow describe). Treat a
  verifier's non-zero exit (it returns non-zero when it finds changed files) as the
  normal changed-file result, not an unreadable source.
  CRITICAL (the v0.6.2 Go build got this wrong, exposed by diffing against the
  C++/libzypp build): you MUST actually determine each `/etc` entry's owning
  package and its package-recorded baseline (digest, mode, owner, group), and
  SUPPRESS package-pristine entries, emitting only unpackaged or changed-from-package
  ones. The v0.6.2 build mislabelled ~1700 package-owned files as unpackaged
  (empty `package_name`) and over-emitted them, because the ownership/digest lookup
  was not actually performed. Do the lookup: for each enumerated `/etc` path, query
  rpm for the owning package and the recorded file digest/mode/owner (e.g. via the
  rpm file database; `rpm -qf` for ownership and the recorded digest for the
  baseline), compare, and suppress when all of digest/target+mode+owner+group match.
  Never default a path to unpackaged because the lookup was skipped. The C++ build
  (libzypp) is the behavioural oracle for this scope.
- `[spec]` `[changed-0.6.4]` Pristine rule refinements: (1) judge each `/etc` path
  INDEPENDENTLY against its own owning package; never collapse a symlink with the
  file it points to (suppressing a pristine symlink must NOT suppress its target
  file; the target is a separate path judged against its own owner, often a
  different package, e.g. `/etc/pam.d/common-auth` owned by `pam` vs
  `common-auth-pc` owned by `pam-config`); never dereference a symlink to judge it.
  (2) A symlink is pristine when its TARGET matches the package-recorded target
  (do not compare a symlink's mode); a regular file when digest+mode+owner+group
  match. An owned distro symlink with the package's target (the `/etc/X11/xim.d/*/
  40-ibus` links) is suppressed. (3) `package_name` is the BARE name
  (`openssh-server`), never the NEVRA; `rpm -qf` prints the full NEVRA, so reduce
  it to the name. (4) Do the lookups in BULK: one `rpm -qf` over all enumerated
  `/etc` paths (it accepts many path arguments and prints owners line-by-line) and
  a bulk verification of the owning packages, NOT `rpm -qf`/`rpm -V` per file. This
  is the performance fix; the result is unchanged and the work stays bounded to
  `/etc`.
- `[spec]` `[changed-0.6.2]` Filesystem object model. The `/etc` walk (and the
  scope=full walk over `/usr`/`/boot`) must recurse into directories and classify
  each entry by its own type using lstat (do NOT follow symlinks, do NOT os.ReadFile
  a path before classifying). In Go: `filepath.WalkDir` or a manual stack with
  `os.Lstat`; for each entry, `d.Type()&fs.ModeSymlink` -> read target with
  `os.Readlink` (store verbatim, do not `filepath.Abs`/`Clean` it), regular file
  -> hash, `IsDir` -> descend (emit nothing), anything else (device/fifo/socket)
  -> skip. The original crash was calling a file read on a directory; classify
  first. A directory, symlink, or special file is never an unreadable-source
  error. Records carry a `target` field (verbatim symlink target, "" for
  non-links) with the type/sha256/target consistency from TYPES. In compute-drift,
  type is part of identity: differing type -> modified; same type compare sha256
  (file) or target (link). Hardlinks: treat per path by content+type, do not
  attempt to detect or preserve hardlink identity.
- `[spec]` `[changed-0.7.0-reserved]` converge-files does NOT yet create/update/
  remove symlinks or handle type transitions; that is reserved for the apply
  milestone. When implemented: a declared type "link" is converged by its target;
  a declared-vs-actual type mismatch at a path is a HARD ERROR that aborts the
  transaction (no silent destructive rewrite). Do not silently delete a directory
  tree to write a file or vice versa.
- `[spec]` `[changed-0.6.0]` Full-scan integrity (`scope=full`, on `describe` and
  `verify` only; default `etc` scans nothing outside `/etc`). Under `full`, scan the
  package-managed trees outside `/etc` (`/usr`, the usr-merge roots `/bin` `/sbin`
  `/lib` `/lib64`, and `/boot`; exclude `/opt` and the virtual, runtime, and
  mutable-data trees; do not cross into unlisted mounts; honour the keep-list) and
  emit two observational scopes: `changed_managed_files` (packaged files outside
  `/etc` differing from the package baseline, with a `changes` list) and
  `unmanaged_files` (files no package owns). These are observational: do NOT feed
  them into `compute-intent-diff` or convergence, and never write them to the
  applied record; `compute-drift` surfaces them under `scope=full` as
  `managed_files_modified` and `unmanaged_files_present`. The scan is expensive
  (stat + hash the trees, verify packaged files); it is the part deliberately kept
  out of the default. In Go, find additions by walking the trees and subtracting
  the rpmdb-owned path set, and find modifications by comparing packaged file
  digests to the rpmdb baseline (or `rpm -V` scoped to those trees); do not shell
  out to a whole-system `rpm -Va`. Scope keys are underscore_style
  (`changed_managed_files`, `unmanaged_files`), matching Machinery's JSON keys.

### Integration with the system (Go-specific)

- `[recommended]` Drive `zypper`, `snapper`, `systemctl`, and `rpm` by executing
  their command-line interfaces (`os/exec`) and parsing their output, rather than
  binding libzypp via cgo. This keeps `CGO_ENABLED=0` and yields the single static
  binary the spec calls for, and matches "no runtime deps of its own beyond the
  tools it drives". Repositories are read as files directly (no exec needed).
  `[extract]` Confirm what the existing code did; if it used cgo/libzypp, that is a
  decision to revisit against the static-binary goal, not to preserve by default.
- `[spec]` The transaction binding is abstract: `acquire-transaction-context`
  resolves `auto|external|internal` and returns a context with a writable `root`
  and `opened_here`. The convergence code path is identical regardless of binding.
  Keep the binding isolated in `internal/txn` so the rest of the code is unaware
  of which mechanism opened the snapshot.
- `[spec]` Unit enablement under a root uses offline enablement (e.g.
  `systemctl --root <ctx.root> ...` semantics) for `converge-units`, and a query
  for `describe-actual-state`; do not rely on first-boot preset evaluation.

### Spec-hash embedding and provenance

- `[spec]` Embed the SHA256 of `zypper-declarative.spec.md` in every produced
  artifact: source headers, the `--version` output, `TRANSLATION_REPORT.md`
  (`Spec-SHA256:`), the RPM spec comment, the DEB control `X-PCD-Spec-SHA256:`,
  the Containerfile label, and the Makefile variable.
- `[recommended]` In Go, keep the hash and version in `internal/meta`, injected at
  build via `-ldflags -X` or as a generated file consumed by `go:embed`. `--version`
  prints `zypper-declarative <version> spec:<sha256>`.

### Build and packaging

- `[spec]` Deliverable is a single static binary, no runtime deps of its own,
  surfaced as a zypper subcommand (an executable in `/usr/lib/zypper/commands`)
  and invocable directly. Final container stage is `FROM scratch`.
- `[spec]` Installation is via an OBS package on build.opensuse.org. No
  curl-based installation.
- `[spec]` Signal handling: clean exit on SIGTERM and SIGINT; an interrupted
  `apply` discards the transaction and leaves no new snapshot as the default boot
  target. Document the approach in `TRANSLATION_REPORT.md`.
- `[extract]` The existing `Makefile`/packaging targets and any OBS `.spec`
  scaffolding worth keeping.

### Testing boundary (aligns with your test-author methodology)

- `[pcd]` Tests are black-box: they invoke the built binary through the DEPLOYMENT
  interface using `os/exec` (`exec.Command`) and assert on stdout, stderr, and
  exit code. Tests must NOT call internal Go functions or simulate the binary's
  behaviour through wrapper code. The new v0.5.0 examples (bare invocation,
  unknown verb, `describe out=...yaml`, `verify state-path=...yaml`, unreadable
  and genuinely-empty repositories) are black-box assertions of exactly this kind.

---

## Do NOT carry these over from the existing code (v0.5.0 through v0.6.4 changes)

1. `describe` writing JSON regardless of the `out` extension. Output now follows
   `resolve-format`.
2. `repositories` read via a method that returns empty for an unprivileged or
   uncached run, and any code path that emits an empty scope on a read failure.
   Read repos.d files; never emit empty for unreadable; omit genuine-empty.
3. Bare `zypper declarative` exiting 2. It now prints usage to stdout and exits 0.
4. Any inline, per-call-site format selection in the manifest loader. It now goes
   through the shared `resolve-format`.
5. (v0.5.1) `version` and `help` as bare words returning "unknown verb" (exit 2)
   while only `--version` / `--help` work. The bare words are now the canonical
   global commands and exit 0; the flags are tolerated aliases.
6. (v0.5.2) Treating a package-verification command's non-zero exit (it returns
   non-zero when it finds changed files, i.e. the normal case) as an unreadable
   source. Consume the verifier output; only a genuine access or I/O failure is
   unreadable.
7. (v0.5.2) Running a whole-system package verification (for example `rpm -Va`)
   for config_files. Bound the work to `/etc`: enumerate `/etc`, and consult
   package metadata only for the `/etc` files found (for example query the
   packaged `/etc` file digests from the rpmdb and compare, or verify only the
   packages that own an `/etc` file). Do not read, hash, or verify files outside
   `/etc`. This is both correctness and the performance fix (the slow full-system
   verification scales with the installed base; the bounded read scales with the
   size of `/etc`).
8. (v0.6.0) The integrity scan (`scope=full`) is opt-in and on `describe`/`verify`
   only; do not engage it by default or on `status`/`diff`/`apply`. Its two scopes
   are observational; never feed them to convergence or write them to the applied
   record.
9. (v0.6.1) `verify` requiring an applied record even when a reference
   `manifest-path` is supplied, and `verify`/`diff` always reading the live system.
   When `manifest-path` (reference) and/or `state-path` (actual) are given, route
   `compute-drift` over the files: `verify manifest-path=X state-path=Y` and
   `diff manifest-path=X state-path=Y` must be pure two-file comparisons that read
   neither the live system nor any applied record. `compute-drift` is already pure;
   this is verb-layer routing.
10. (v0.6.1) `apply` (via `load-desired-manifest`) silently accepting a manifest
    that carries a non-empty observational scope. Reject it with a manifest error
    so a raw `describe scope=full` dump cannot be applied as a baseline; an empty
    or absent observational scope is tolerated and dropped.
11. (v0.6.3) mislabelling package-owned `/etc` files as unpackaged and over-emitting
    package-pristine files (the actual v0.6.2 Go bug). Do real rpm ownership/digest
    lookup and suppress pristine entries; emit only unpackaged or changed-from-package.
12. (v0.6.3) serialising a scope's `_attributes` as `null`; it is always a JSON
    object, empty `{}`. Quote string-typed YAML scalars (`mode: "0600"`).
13. (v0.6.3) anything that makes describe output diverge from the C++ and Rust
    builds on the same host (after excluding `meta.created_at`); the three-way
    diff is a consistency oracle. `meta.generator` must be `zypper-declarative
    <version>` so it matches across implementations.
14. (v0.6.4) collapsing a symlink with its target file (judge each `/etc` path
    independently against its own owning package; suppressing a pristine link must
    not suppress its target).
15. (v0.6.4) comparing a symlink's mode for pristine-ness (a link is pristine iff
    its target matches the package); over-comparing files is fine, over-comparing
    links wrongly emits them.
16. (v0.6.4) putting the full NEVRA in `package_name` (bare name only); and doing
    `rpm -qf`/`rpm -V` per file instead of in bulk (batch the queries).

## Slots to fill from /tmp/pcd-original/code/

- `[extract]` Actual package names and file split, if they differ from the
  recommended layout and are at least as clear.
- `[extract]` The Go version floor and any direct dependencies already vendored.
- `[extract]` The error-carrying type or pattern used for `domain`.
- `[extract]` The YAML library already chosen, if any, and whether it meets the
  safe profile (replace it if it does not).
- `[extract]` Existing Makefile/OBS packaging targets, container build, and the
  spec-hash injection mechanism.
- `[extract]` Whether system integration was exec-based or cgo/libzypp, and the
  decision to keep or revisit it against the static-binary goal.

---

## Changelog

- 2026-06-01: Updated to spec v0.6.4. Added the pristine-rule refinements from the
  three-way comparison: independent per-path judgement (no symlink/target
  collapse), symlink pristine-by-target-only, bare `package_name` (not NEVRA), and
  bulk `rpm -qf`/verification instead of per-file (the performance fix). Items
  14-16 added to the do-not-carry list.
- 2026-06-01: Updated to spec v0.6.3 after diffing the Go and C++ describe output
  on milos. Headline: the Go build's config_files reader mislabelled ~1700
  package-owned `/etc` files as unpackaged and over-emitted package-pristine files,
  because the ownership/digest lookup was not actually performed; the entry and the
  do-not-carry list now require real rpm ownership/digest determination and
  suppression of pristine entries (the C++/libzypp build is the oracle). Also added:
  `_attributes` `{}`-not-null and YAML string quoting, `meta.generator` carrying the
  version, and the three-way cross-implementation diff as a consistency oracle.
- 2026-05-29: Initial Go decisions hints for the v0.5.0 guided regeneration.
  Captures the spec-determined architecture (single live-state reader,
  pure compute-drift, shared resolve-format, abstract transaction binding,
  applied-record-always-JSON, canonical-model hashing), the Go-specific defaults
  (exec-based integration for a static CGO_ENABLED=0 binary, internal package
  layout, spec-hash in internal/meta), the v0.5.0 behaviours that must not be
  inherited from the prior buggy code, and the slots to extract from the existing
  implementation.
