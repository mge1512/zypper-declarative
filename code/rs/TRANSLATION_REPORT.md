# TRANSLATION_REPORT.md — zypper-declarative (Rust)

- **Spec-SHA256:** `87b9f2f3bf92afd6fe037412d56a23959290e5816d722a3def9505d07aa5acd7`
  — SHA256 of the merged spec text. The host spec declares no `Includes:`
  directives, so the merged hash equals the host hash and the Included-Specs
  table below is empty.
- **Spec-SHA256 (host):** `87b9f2f3bf92afd6fe037412d56a23959290e5816d722a3def9505d07aa5acd7`
- **Included-Specs:**

  | Path | SHA256 |
  |------|--------|
  | _(none)_ | — |

- **LLM-Name:** `claude-opus-4-8` (from `ROLE.md`)
- **Mode:** `translator`
- **Deployment-Template:** `cli-tool.template.md v0.3.29`
- **Spec-Schema:** `0.4.0` (host META). The merge described in the prompt's Spec
  Composition section was implemented (trivially — no includes); `Includes:`
  directives were not silently ignored because there are none.
- **Tests-First-Compliance:** `yes`. All test files under
  `independent_tests/claude-opus-4-8/` were written and passed their syntax
  check (`cargo check --tests`) **before** any implementation source file was
  written. The structural Tests-First guard (prompt step 3) passed: the test
  directory contained five non-empty test files before Phase 2 began.
- **Continuity-Check:** not applicable — no test-author input. The input
  directory contained no `independent_tests/<other-role-llm-name>/` directory
  and no `TEST_REPORT.md`, so this is a single-LLM run.

## Target language resolution

Resolved **LANGUAGE = Rust**. The cli-tool template's default language is Go;
the invocation (`prompt.md`: `Language: RUST`) selects Rust, which the template
lists as a `supported` LANGUAGE and a valid `LANGUAGE-ALTERNATIVES` entry,
selectable via preset/project override. No preset files were present; the
override is the explicit invocation parameter. `BINARY-TYPE` resolves to
`static` (the only valid value for Rust per the template), achieved with
`-C target-feature=+crt-static` against glibc.

## Module identity resolution

`MODULE-IDENTITY: host-specified` applies. Authoritative source **(1) spec META
`Module:`** is present: `Module: github.com/mge1512/zypper-declarative`. Per the
Rust decisions hints (`[spec]`), this Go-style module path is mapped to a Rust
crate **name** `zypper-declarative`, with the upstream URL recorded in
`Cargo.toml` `repository = "github.com/mge1512/zypper-declarative"`. Source (2)
the Rust decisions hints file agrees (`crate name zypper-declarative`). Sources
1 and 2 agree; no conflict, no fallback used. The crate name matches the spec
title (`# zypper-declarative`).

## Delivery mode

Filesystem (mode 1): all source files written directly to
`/tmp/pcd-output/code/rs/`. The compile gate and both test suites were executed
in-environment. Dependencies were resolved with `cargo fetch` and vendored with
`cargo vendor vendor/` for offline OBS builds; `cargo build --release --offline`
succeeds against the vendored tree.

## Source partitioning (SOURCE-PARTITIONING: modular, one-entry-one-implementation)

The implementation is partitioned into one entry-point module and many
implementation modules (by behavioural domain, per the `by-behaviour-domain`
supported row and the Rust decisions hints layout):

| File | Responsibility |
|------|----------------|
| `src/main.rs` | Entry point: **CLI dispatch only** — argv collection, signal handlers, exit-code propagation. Implements no behaviour. |
| `src/cli.rs` | Verb layer: dispatch, orchestration of the five verbs, domain→exit-code mapping, usage text. |
| `src/config.rs` | CONFIG + invocation parsing (key=value, bare verbs, flag aliases). |
| `src/types.rs` | The shared data model (serde), Diff, DriftReport, enums. |
| `src/format.rs` | `resolve-format` (single authority). |
| `src/manifest.rs` | `load-desired-manifest`, JSON/YAML parse + safe profile, schema validation, serialise. |
| `src/hash.rs` | Canonical-model hashing (`desired_sha256`) and file content SHA256. |
| `src/diff.rs` | `compute-intent-diff`, `compute-drift` (pure, no I/O). |
| `src/state.rs` | `describe-actual-state` (the single live reader). |
| `src/record.rs` | `load-applied-record`, `write-applied-record`. |
| `src/txn.rs` | `acquire-transaction-context`. |
| `src/converge.rs` | `converge-packages`, `converge-files`, `converge-units`. |
| `src/interfaces.rs` | INTERFACES: `CommandRunner`/`Filesystem` traits + production and test-double implementations. |
| `src/error.rs` | `Diagnostic`, `Severity`, `Domain`, `ExitCode`. |
| `src/meta.rs` | Embedded spec hash, version, generator. |
| `src/lib.rs` | Library root (re-exports modules for in-crate unit tests). |

A single-file implementation is **not** present.

## STEPS ordering applied per BEHAVIOR

- **apply** (`cli::verb_apply`): implements STEPS 1–11 in order — load manifest →
  load applied record → compute intent diff → empty-diff drift check / "nothing
  to do" → acquire transaction → converge packages (repositories applied within)
  → converge files → converge units → write applied record → post-converge
  verification → seal/activate + summary. On any error, the verb returns before
  sealing, so no new snapshot becomes the default boot target.
- **diff** (`verb_diff`): STEPS 1–5 — load manifest → load applied → intent diff
  → obtain actual state (offline `state-path` or live) → print combined plan,
  exit 0.
- **verify** (`verify_inner`): STEPS 1–4 — determine reference (manifest-path or
  applied record; "no declaration applied" exit 2 when neither) → obtain actual
  state → compute drift → clean exit 0 / per-item diagnostics exit 1.
- **status** (`verb_status`): STEPS 1–4 — reject unknown args (parser) → load
  applied record / "no declaration applied" exit 0 → print summary → drift line.
- **describe** (`verb_describe`): STEPS 1–5 — reject unknown arg/format (parser)
  → obtain actual state with `on_unreadable`+`scope` → resolve output format via
  `resolve-format(out)` → serialise → write to `out` or stdout, exit 0.
- **describe-actual-state** (`state::describe_actual_state`): STEPS 1–6 —
  packages (rpmdb) → repositories (`repos.d`) → services (offline enablement) →
  config_files (`/etc` walk with lstat classification, ownership/pristine via
  rpm) → 4a full-scan integrity under `scope=full` → assemble (omit empty
  scopes) → unreadable-source handling (error/warn).
- **resolve-format**, **load-desired-manifest**, **load-applied-record**,
  **compute-intent-diff**, **compute-drift**, **acquire-transaction-context**,
  **converge-packages/-files/-units**, **write-applied-record**: each implements
  its STEPS in declared order; see the module list above.

## MECHANISM / object-model specifics implemented exactly

- `/etc` walk and `scope=full` walk classify each entry by its own type with
  `std::fs::symlink_metadata` (lstat): regular files are hashed (sha2), symlinks
  record their **verbatim** target via `read_link` (no `canonicalize`, no
  normalisation), directories are traversed but not emitted, special files are
  skipped. A directory/symlink/special file is never an unreadable-source error.
- `compute-drift` treats type as part of a config file's identity (type
  transition → modified, regardless of content); symlinks compared by target,
  files by sha256.
- `files_extra` contains only unpackaged, undeclared `/etc` files;
  `/etc/etc.syncpoint` and keep-listed paths are excluded.
- Package ownership and the pristine rule are determined via the rpm database
  (`rpm -qf`, `rpm -V`); package-pristine `/etc` entries are suppressed. A
  package verifier exiting non-zero to report differences is the normal
  changed-file result, **not** an unreadable source. Ownership is never
  defaulted to unpackaged because a lookup was *skipped*; see the documented
  conservative choices below.
- Every scope's `_attributes` serialises as a JSON object (empty `{}`), never
  `null` (`BTreeMap` + `ScopeWrapper`). `meta.generator` always carries the
  version (`zypper-declarative 0.6.3`).
- `desired_sha256` is the SHA256 of a canonical serialisation of the parsed data
  model: keys sorted, compact separators, `_elements` identity-sorted (packages
  by name+arch, repositories by alias, services by name, config_files by path),
  with `created_at`/`generator`/`desired_sha256` neutralised. JSON and YAML of
  the same model hash identically (unit-tested).
- YAML safe profile: a single document only (interior `---`/`...` rejected),
  explicit tags rejected (`!`/`!!` tokens), single typed deserialise rejecting
  implicit coercion; a YAML input needing a disabled feature is a manifest error.
- Applied record is always canonical JSON regardless of input format.

## INTERFACES test doubles produced

Per the INTERFACES section, production implementations and test doubles were
produced (in `src/interfaces.rs`):

- `CommandRunner` (trait) → `OSCommandRunner` (production, `std::process::Command`)
  and `FakeCommandRunner` (test double, canned responses).
- `Filesystem` (trait) → `OSFilesystem` (production, `std::fs`) and
  `FakeFilesystem` (test double, in-memory map).

The independent black-box test suite does **not** use the production
implementations or the test doubles directly — it invokes the built binary as a
subprocess (the spec's CLI interface). The test doubles are available for
in-crate unit testing of orchestration logic.

## TYPE-BINDINGS / GENERATED-FILE-BINDINGS

The cli-tool template declares no `## TYPE-BINDINGS` or
`## GENERATED-FILE-BINDINGS` section, so neither applies. Logical types were
mapped to idiomatic Rust types following the Rust decisions/milestones hints:
`ScopeWrapper<T>` → generic serde struct with `_attributes`/`_elements`;
declarable scopes → `Option<Scope>` to preserve absent-vs-empty.

## Constraint: supported / forbidden BEHAVIORs

All BEHAVIORs in the spec are `Constraint: required`; all were implemented.
No BEHAVIOR carries `supported` or `forbidden`. The template's `OUTPUT-FORMAT`
constraints were honoured: `RPM` and `DEB` (required) were produced; `OCI`,
`PKG`, `binary` (supported) were **not** produced because no resolved preset
activates them (single-LLM run, no preset files present). The template's
`forbidden` rows (`INSTALL-METHOD: curl`, `CONFIG-ENV-VARS`, `NETWORK-CALLS`,
`FILE-MODIFICATION: input-files`) were respected — see Template-constraints
compliance and Documented deviations below.

## COMPONENT → filename mapping (DELIVERABLES)

The spec has no `## DELIVERABLES` COMPONENT entries; deliverable filenames derive
from the template's DELIVERABLES table with `<n>` = `zypper-declarative`:
`Cargo.toml`, `src/main.rs` + `src/*.rs`, `Makefile`, `README.md`, `LICENSE`,
`zypper-declarative.1.md` + `zypper-declarative.1`, `zypper-declarative.spec`,
`debian/{control,changelog,rules,copyright}`, and this `TRANSLATION_REPORT.md`.
Dependencies were vendored under `vendor/` and `Cargo.lock` was produced by the
resolver (compile-gate output).

## Active MILESTONE

No MILESTONE has `Status: active` — all six (`0.0.0`–`0.6.0`) are
`Status: pending`. Per the prompt ("If no MILESTONE has Status: active, translate
the full spec as normal"), the **full spec** was translated (all 16 BEHAVIORs).
The Rust scaffold/milestones hints (`cli-tool.rs.milestones.hints.md`) and the
decisions hints (`zypper-declarative.rs.decisions.hints.md`) were read and
applied. The M0 and 0.1.0 acceptance criteria all pass (verified
post-build): `version`/`help` bare words exit 0, `--version` alias, bare
invocation exits 0 with usage, `format=bad_value` exits 2, `describe out=*.yaml`
emits YAML by extension, `status` reports "no declaration applied".

## Dependencies (DEPENDENCIES section)

Direct dependencies, resolved by the Cargo resolver (versions in `Cargo.lock`):

| Crate | Version | Role | Notes |
|-------|---------|------|-------|
| `serde` | 1 (1.0.228) | data model derive | from registry |
| `serde_json` | 1 (1.0.150) | canonical JSON | from registry |
| `serde_yaml` | 0.9 (0.9.34+deprecated) | YAML opt-in (Value-walk safe profile) | **flagged**: the crate is marked deprecated upstream. It meets the safe-profile requirement (no executable tags; we additionally reject multi-document and explicit tags and parse via a single typed deserialise). A maintainer should verify the YAML library choice/version before release; alternatives such as a maintained YAML crate may be substituted without behaviour change. |
| `sha2` | 0.11 (0.11.0) | SHA256 (content + canonical hash) | from registry |
| `libc` | 0.2 (0.2.186) | SIGTERM/SIGINT handler registration | from registry |

Bindings to libzypp, snapper/btrfs, and systemd are **not** linked: per the Rust
decisions hints, the tool drives `zypper`, `snapper`, `systemctl`, and `rpm` via
their command-line interfaces (`std::process::Command`) and reads `repos.d` as
files, keeping the binary static with no SUSE C/C++ library FFI. No version
strings for those bindings are therefore required.

## Parsing approach

Hand-written key=value + bare-verb parser (`src/config.rs`), no `clap` (per the
decisions hints — a `--flag`-shaped parser would fight the spec's CLI style).
Options are `key=value` and accepted in **any position** (including after the
verb); bare words are verbs; `--version`/`--help`/`-h` are tolerated aliases for
the two global commands only. Unknown verb/option/value or a missing value →
usage to stderr, exit 2. JSON parsing via serde_json; YAML via serde_yaml under
the safe profile (text pre-checks for multi-document and explicit tags, then a
single typed deserialise).

## Signal-handling approach

`src/main.rs` installs `SIGTERM` and `SIGINT` handlers via `libc::signal` that
exit cleanly (code 130) with no partial output: stdout is line-buffered, and a
long-running `apply` seals the snapshot only at its final step, so an interrupt
before that leaves no new default boot target and the running system unchanged.

## Compile gate result (template EXECUTION)

Executed in-environment, resolved language Rust:

| Step | Command | Result |
|------|---------|--------|
| 1 Dependency resolution | `cargo fetch`; `cargo vendor vendor` | pass — `Cargo.lock` + `vendor/` produced |
| 2 Compilation (debug) | `cargo build` | pass — clean, no warnings |
| 2 Compilation (release, static) | `cargo build --release` / `--offline` | pass — clean; binary statically linked (`ldd` → "statically linked") |
| 3 Translator test run | `make test` (`cargo test --lib` + `cargo test --test '*' --manifest-path independent_tests/claude-opus-4-8/Cargo.toml`) | pass — 19 lib + 37 black-box tests |
| 4 Test-author test run | — | not applicable (single-LLM run) |

Static linking note: `-C target-feature=+crt-static` cannot be applied globally
(it breaks proc-macro compilation of `serde_derive`). The working configuration
sets a **default build target** in `.cargo/config.toml` so host build-deps
(proc-macros) build without the target rustflags while the binary links
statically. A plain `cargo build --release` then produces the static binary at
`target/x86_64-unknown-linux-gnu/release/zypper-declarative`, which the Makefile
copies to the project root (BINARY-LOCATION: project-root). The black-box tests
invoke `../../zypper-declarative`.

## Public API Surface

The names and signatures below form the public API surface of the implementation
modules. The next translation of this spec at Version 0.6.3 must preserve them
(it may add, not remove/rename without a Version increment).

### module `meta`
- `pub const PROGRAM_NAME: &str`
- `pub const VERSION: &str`
- `pub const SPEC_SHA256: &str`
- `pub fn generator() -> String`
- `pub fn version_line() -> String`

### module `error`
- `pub enum Severity { Error, Warning }`
- `pub enum Domain { Packages, Repositories, Units, Files, Manifest, Transaction, Invocation }`
- `pub struct Diagnostic { severity: Severity, domain: Domain, message: String }`
- `pub fn Diagnostic::error(domain: Domain, message: impl Into<String>) -> Diagnostic`
- `pub fn Diagnostic::warning(domain: Domain, message: impl Into<String>) -> Diagnostic`
- `pub fn Diagnostic::render(&self) -> String`
- `pub enum ExitCode { Ok = 0, Logical = 1, Invocation = 2 }`
- `pub fn ExitCode::code(self) -> i32`
- `pub type DResult<T> = Result<T, Diagnostic>`

### module `types`
- `pub type Attributes = BTreeMap<String, serde_json::Value>`
- `pub struct ScopeWrapper<T> { attributes: Attributes, elements: Vec<T> }`
- `pub fn ScopeWrapper::<T>::with_attr(key: &str, value: &str) -> Self`
- `pub fn ScopeWrapper::<T>::is_empty(&self) -> bool`
- `pub struct ManifestMeta { format_version: i64, generator: String, created_at: String, desired_sha256: String }`
- `pub struct PackageRecord { name, version, release, arch: String }`
- `pub type PackagesScope = ScopeWrapper<PackageRecord>`
- `pub struct RepositoryRecord { alias, name, url, type: String, enabled, gpgcheck, autorefresh: bool, priority: i64 }`
- `pub type RepositoriesScope = ScopeWrapper<RepositoryRecord>`
- `pub struct ServiceRecord { name: String, state: String }`
- `pub type ServicesScope = ScopeWrapper<ServiceRecord>`
- `pub struct ManagedFileRecord { name, type, mode, user, group, sha256, target, content_ref, package_name: String }`
- `pub type ConfigFilesScope = ScopeWrapper<ManagedFileRecord>`
- `pub struct ManagedBaselineRecord { name, type, mode, user, group, sha256, target, package_name: String, changes: Vec<String> }`
- `pub type ChangedManagedFilesScope = ScopeWrapper<ManagedBaselineRecord>`
- `pub struct UnmanagedFileRecord { name, type, mode, user, group, sha256, target: String }`
- `pub type UnmanagedFilesScope = ScopeWrapper<UnmanagedFileRecord>`
- `pub struct Manifest { meta, packages?, repositories?, services?, config_files?, changed_managed_files?, unmanaged_files? }`
- `pub fn Manifest::empty() -> Self`
- `pub fn Manifest::{packages_elems, repositories_elems, services_elems, config_files_elems}(&self) -> &[...]`
- `pub type AppliedRecord = Manifest`
- `pub struct Diff { packages_install, packages_remove, repos_set, files_write, files_delete, units_change }`
- `pub fn Diff::is_empty(&self) -> bool`
- `pub struct DriftReport { files_modified, files_extra, units_divergent, packages_divergent, managed_files_modified, unmanaged_files_present }`
- `pub fn DriftReport::is_empty(&self) -> bool`
- `pub enum ManifestFormat { Json, Yaml }` + `pub fn parse(&str) -> Option<ManifestFormat>`
- `pub enum TransactionMode { Auto, External, Internal }` + `pub fn parse(&str) -> Option<TransactionMode>`
- `pub struct TransactionContext { mode: TransactionMode, opened_here: bool }`
- `pub enum ScanScope { Etc, Full }` + `pub fn parse(&str) -> Option<ScanScope>`
- `pub enum OnUnreadable { Error, Warn }` + `pub fn parse(&str) -> Option<OnUnreadable>`

### module `format`
- `pub fn resolve_format(explicit: Option<ManifestFormat>, path: Option<&str>, default: ManifestFormat) -> ManifestFormat`

### module `hash`
- `pub fn canonical_sha256(manifest: &Manifest) -> String`
- `pub fn sha256_bytes(data: &[u8]) -> String`

### module `manifest`
- `pub struct LoadedManifest { manifest: Manifest, desired_sha256: String }`
- `pub fn load_desired_manifest(manifest_path: &str, explicit_format: Option<ManifestFormat>, default_format: ManifestFormat) -> Result<LoadedManifest, Diagnostic>`
- `pub fn parse_manifest(bytes: &[u8], fmt: ManifestFormat) -> Result<Manifest, Diagnostic>`
- `pub fn parse_state_dump(bytes: &[u8], fmt: ManifestFormat) -> Result<Manifest, Diagnostic>`
- `pub fn validate_manifest(manifest: &Manifest) -> Result<(), Diagnostic>`
- `pub fn serialise_manifest(manifest: &Manifest, fmt: ManifestFormat) -> Result<String, Diagnostic>`
- `pub fn serialise_applied_record(record: &Manifest) -> Result<String, Diagnostic>`

### module `diff`
- `pub fn compute_intent_diff(desired: &Manifest, applied: &AppliedRecord) -> Diff`
- `pub fn compute_drift(actual: &Manifest, reference: &AppliedRecord, keep_list: &HashSet<String>) -> DriftReport`

### module `state`
- `pub struct ActualState { manifest: Manifest, diagnostics: Vec<Diagnostic> }`
- `pub fn describe_actual_state(runner: &dyn CommandRunner, root: &str, on_unreadable: OnUnreadable, scope: ScanScope, keep_list: &HashSet<String>) -> Result<ActualState, Diagnostic>`

### module `record`
- `pub struct LoadedApplied { record: AppliedRecord, present: bool }`
- `pub fn load_applied_record(root: &str) -> Result<LoadedApplied, Diagnostic>`
- `pub fn write_applied_record(ctx_root: &str, desired: &Manifest, desired_sha256: &str, resolved: &PackagesScope) -> Result<(), Diagnostic>`

### module `txn`
- `pub struct AcquiredContext { ctx: TransactionContext, root: String }`
- `pub fn acquire_transaction_context(runner: &dyn CommandRunner, mode: TransactionMode) -> Result<AcquiredContext, Diagnostic>`

### module `converge`
- `pub fn converge_packages(runner: &dyn CommandRunner, ctx_root: &str, diff: &Diff, repo_lock: Option<&str>) -> Result<PackagesScope, Diagnostic>`
- `pub fn converge_files(runner: &dyn CommandRunner, ctx_root: &str, diff: &Diff, content_store: Option<&str>, keep_list: &HashSet<String>) -> Result<(), Diagnostic>`
- `pub fn converge_units(runner: &dyn CommandRunner, ctx_root: &str, diff: &Diff) -> Result<(), Diagnostic>`

### module `interfaces`
- `pub trait CommandRunner: Send + Sync { fn run(&self, cmd: &str, args: &[&str]) -> Result<(String, String), CommandError>; }`
- `pub struct CommandError { command, message: String, status: Option<i32>, stdout, stderr: String }`
- `pub struct OSCommandRunner` (impl CommandRunner)
- `pub trait Filesystem: Send + Sync { read_file, exists, list_repo_files }`
- `pub struct OSFilesystem` (impl Filesystem)
- `pub struct FakeCommandRunner` (test double, impl CommandRunner) + `new`, `with`
- `pub struct FakeFilesystem` (test double, impl Filesystem) + `new`

### module `config`
- `pub enum Verb { Apply, Diff, Verify, Status, Describe, Version, Help }` + `pub fn parse(&str) -> Option<Verb>`
- `pub struct Invocation { verb: Option<Verb>, opts: Options }`
- `pub struct Options { ...all CONFIG knobs... }` (impl Default)
- `pub fn parse_args(args: &[String]) -> Result<Invocation, Diagnostic>`
- `pub fn load_keep_list(opts: &Options) -> HashSet<String>`

### module `cli`
- `pub fn run(args: &[String]) -> i32`

## Template-constraints compliance

| Constraint | Status | Evidence |
|------------|--------|----------|
| LANGUAGE = Rust (supported alternative) | ✓ | `Cargo.toml`, this report |
| BINARY-TYPE = static | ✓ | `.cargo/config.toml` crt-static; `ldd` → statically linked |
| SOURCE-PARTITIONING modular + one-entry-one-implementation | ✓ | `src/main.rs` dispatch only; 14 impl modules |
| MODULE-IDENTITY host-specified + propagated | ✓ | crate `zypper-declarative`, repository URL in Cargo.toml, RPM URL, DEB Homepage, README install |
| PUBLIC-API-SURFACE recorded | ✓ | `## Public API Surface` above |
| BINARY-COUNT 1 | ✓ | one `[[bin]]` |
| BINARY-LOCATION project-root | ✓ | Makefile copies binary to `./zypper-declarative`; tests use `../../zypper-declarative` |
| RUNTIME-DEPS none / static | ✓ | static binary; drives system tools via exec, no linked runtime deps |
| CLI-ARG-STYLE key=value (+ bare-words) | ✓ | `src/config.rs`; no POSIX flags for options |
| EXIT-CODE-OK/ERROR/INVOCATION 0/1/2 | ✓ | `error::ExitCode`, `cli::exit_for_domain` |
| STREAM-DIAGNOSTICS stderr / STREAM-OUTPUT stdout | ✓ | `cli::emit_diagnostic` → stderr; results → stdout |
| SIGNAL-HANDLING SIGTERM/SIGINT | ✓ | `src/main.rs` handlers |
| OUTPUT-FORMAT RPM, DEB (required) | ✓ | `zypper-declarative.spec`, `debian/` |
| OUTPUT-FORMAT OCI/PKG/binary (supported) | n/a | not active in any preset → not produced |
| INSTALL-METHOD OBS; curl forbidden | ✓ | README/RPM/DEB document OBS only; no curl |
| PLATFORM Linux | ✓ | spec is Linux-only; RPM `ExclusiveArch: x86_64` |
| CONFIG-ENV-VARS forbidden | ✓ | no behaviour controlled via env vars (transaction *detection* probes a marker file, not an option) |
| NETWORK-CALLS forbidden | deviation | documented below |
| FILE-MODIFICATION input-files forbidden | ✓ | inputs (manifest, state dump) are read-only |
| IDEMPOTENT | ✓ | apply no-op on unchanged manifest + undrifted system; format-independent `desired_sha256` |
| spec-hash embedded everywhere | ✓ | see Spec-hash embedding below |

## Documented deviations (per the spec's own Template-deviations section)

- **NETWORK-CALLS:** the tool performs no direct network I/O of its own; all
  package retrieval is delegated to the package manager against a declared,
  pinned, signed repository. The supply-chain intent (no curl-style fetching) is
  fully honoured. Recorded as a deviation because the delegated package
  operation reaches the network through the package manager. (This matches the
  spec's DEPLOYMENT "Template deviations" note.)
- **Privilege:** `apply` requires privilege; the read-only verbs require only
  read access. The cli-tool template assumes a read-only tool by default; this
  is the spec-acknowledged exception.

## Spec-hash embedding

`87b9f2f3bf92afd6fe037412d56a23959290e5816d722a3def9505d07aa5acd7` is embedded
in: every `src/*.rs` header comment; `src/meta.rs` `SPEC_SHA256` (and the binary
`version`/`--version` output `spec:<hash>`); `Makefile` `SPEC_SHA256`;
`zypper-declarative.spec` `# pcd-spec-sha256:`; `debian/control`
`X-PCD-Spec-SHA256:`; `README.md`; the man-page markdown comment; and this
report's `Spec-SHA256:` field. (No Containerfile — OCI not produced.)

## Specification ambiguities encountered

1. **Diagnostic domain string for service/unit drift.** The Rust decisions hints
   list the domain set as `{..., services, ...}`, but the **spec's authoritative
   `Diagnostic` TYPE (line 283)** lists `... | units | ...` and every EXAMPLE
   uses `domain=units`. The spec is authoritative (the hints "cannot override
   spec invariants"), so the implementation emits `units`. Documented in the
   Test Refinements table below.
2. **`rpm -qf` under a non-"/" root.** `rpm --root=<dir> -qf <path>` interprets
   the path against the *host* filesystem, not under `--root`, so per-path
   ownership lookups against an arbitrary root are unreliable. Conservative
   interpretation: when a root carries **no rpm database** (`usr/lib/sysimage/rpm`
   and `var/lib/rpm` both absent), there is no package system, so every file is
   *genuinely* unpackaged — a definitive negative answer, not a skipped lookup,
   consistent with the spec's "never default to unpackaged because the lookup was
   skipped" rule. When an rpmdb *is* present but a query genuinely fails (database
   present but unreadable), the file is conservatively suppressed (owned+pristine)
   rather than over-emitted as unpackaged. Same probe gates the `services` scope
   (`unit_state_present`) so a synthetic root yields a genuinely-empty (omitted)
   scope rather than a spurious unreadable error.
3. **`meta.created_at` format.** The spec calls it RFC3339, informational, and
   not compared. To avoid an unverified third-party date crate, a stable
   RFC3339-shaped placeholder is emitted; it is excluded from the canonical hash
   and never compared, so this does not affect any observable behaviour.
4. **Transaction binding on a non-transactional host.** `acquire-transaction-context`
   probes `transactional-update`/`snapper`; on a non-transactional host
   `internal` mode fails with a transaction error (exit 2 via the verb layer),
   matching `apply_transaction_unavailable`. Full live convergence
   (`converge-*`, snapshot sealing, snapper userdata stamping) is implemented as
   exec-driven calls but is not exercisable offline; see confidence table.

## Rules that could not be implemented exactly as written, and why

- **Live convergence and sealing** (apply STEPS 6–11 on a real transactional
  host: opening a snapshot, offline package/unit convergence, sealing read-only,
  marking the default boot target, snapper userdata stamp) are implemented as
  command-driven operations but cannot be executed in this environment (no
  snapper/transactional-update/live rpmdb mutation). They are verified by code
  review and by the offline-decomposable parts (intent diff, drift, manifest
  load/validate, applied-record write/read round-trip). Marked Medium/Low in the
  confidence table.

## Test results — translator suite (`independent_tests/claude-opus-4-8/`)

Black-box (subprocess) suite, all passing:

- `cli_contract.rs` (10): version bare-word + spec hash, `--version` alias
  identical, help bare-word, `--help`/`-h` aliases, bare invocation exit 0,
  unknown verb exit 2, status unknown arg exit 2, describe unknown format exit 2,
  `format=bad_value` exit 2, option-after-verb accepted.
- `diff_verify_offline.rs` (12): offline two-file diff plan, intent-diff deletion
  label, diff manifest-unreadable exit 2, apply manifest-unreadable exit 2, apply
  invalid format_version exit 1 domain=manifest, apply rejects non-empty
  observational scope exit 1, verify no-reference exit 2 "no declaration
  applied", verify malformed state dump exit 2, verify offline matching exit 0,
  verify offline service divergence exit 1 domain=units, drift type-transition
  exit 1 domain=files, drift ignores changed package-owned undeclared file exit 0.
- `yaml_format_status.rs` (5): status no-declaration exit 0, YAML manifest
  accepted offline, YAML unsafe multi-document rejected exit 1, JSON/YAML
  identity offline both clean, verify state-path `.yaml` extension matching exit 0.
- `describe_root.rs` (10): output-unwritable exit 2, out `.json`/`.yaml` extension
  selection, `format=json` overrides `.yaml` extension, symlink recorded
  verbatim (type=link, verbatim target, sha256 ""), special file (fifo) skipped,
  subdirectory traversal (no "is a directory" error), `_attributes` always object
  never null, meta.format_version=1, default scope omits observational scopes.

In addition, **19 in-crate unit tests** (`cargo test --lib`) cover resolve-format,
canonical hashing, manifest parse/validate, the YAML safe profile, intent-diff,
drift, repo-INI parsing, unit-file state normalisation, logical path stripping,
and rpm -V change detection. All pass.

## Test results — test-author suite

Not applicable — single-LLM run; no test-author suite present in the input.

## Test Refinements

| Test | Result before | Action | Rationale |
|------|---------------|--------|-----------|
| `test_describe_output_unwritable_exit2` | failed | test edited | The synthetic root made the live unit-enablement read fail under default `on-unreadable=error`, so describe exited 1 (a read error) before reaching the write step. Added `on-unreadable=warn` to the fixture (and made synthetic-root scope reads genuinely-empty in the implementation) so the write step is reached; per DESCRIBE STEPS 2→5 the unwritable-output exit-2 path is only reachable after a successful read. Asserts exit 2 on the unwritable `out=`. |
| `test_describe_records_symlink_verbatim` | failed | code fixed | The symlink was suppressed because the rpm ownership query against the synthetic root failed and the conservative fallback marked it owned+pristine. Fixed the implementation to treat a root with no rpm database as "no package owns anything" (definitive unpackaged), per the spec's pristine-rule wording (never default to unpackaged because the lookup was *skipped* — here it is a definitive negative). The unpackaged symlink is now emitted with type=link and verbatim target. |
| `test_verify_offline_service_divergence_units_exit1` | failed | code fixed | The diagnostic domain string was `services` (following the decisions hints) but the spec's authoritative Diagnostic TYPE and EXAMPLEs use `units`. Changed `Domain::Services` → `Domain::Units` rendering as `units`. The spec overrides the advisory hint. |
| (all other tests) | passed | none | — |

## Per-example confidence

Confidence: **High** = Tests-First `yes` and a named test passes without a live
external service. **Medium** = passes but needs live services / not fully
covered offline. **Low** = reasoning/code-review only.

| EXAMPLE | Confidence | Verification method | Unverified claims |
|---------|------------|---------------------|-------------------|
| bare_invocation_shows_help | High | `test_bare_invocation_usage_to_stdout_exit0` | — |
| version_verb_bare_word | High | `test_version_bare_word_prints_name_version_and_spec_hash` | — |
| version_flag_alias | High | `test_version_flag_alias_identical_to_bare_word` | — |
| help_verb_bare_word | High | `test_help_bare_word_prints_usage_to_stdout_exit0` | — |
| unknown_verb_rejected | High | `test_unknown_verb_usage_to_stderr_exit2` | — |
| status_unknown_argument | High | `test_status_unknown_argument_exit2` | — |
| describe_unknown_format | High | `test_describe_unknown_format_exit2` | — |
| status_no_declaration | High | `test_status_no_declaration_exit0` | — |
| diff_manifest_unreadable | High | `test_diff_manifest_unreadable_exit2` | — |
| apply_manifest_unreadable | High | `test_apply_manifest_unreadable_exit2` | — |
| apply_manifest_invalid | High | `test_apply_manifest_invalid_format_version_exit1` | — |
| apply_rejects_full_describe_dump | High | `test_apply_rejects_nonempty_observational_scope_exit1` | — |
| diff_prints_plan / diff_offline_two_files | High | `test_diff_offline_two_files_lists_plan_exit0` | — |
| intent_diff_yields_deletion | High | `test_intent_diff_yields_deletion_lists_file_to_delete`; unit `diff::tests::intent_diff_yields_deletion` | — |
| verify_no_applied_record | High | `test_verify_no_reference_no_applied_record_exit2` | — |
| verify_malformed_state_dump | High | `test_verify_malformed_state_dump_exit2` | — |
| verify_offline_manifest_and_state / verify_offline_no_applied_record_ok | High | `test_verify_offline_matching_exit0_no_applied_record` | — |
| verify_against_external_state_dump | High | `test_verify_offline_service_divergence_units_exit1` | — |
| verify_detects_drift | High | `test_drift_type_transition_is_modified_files_exit1` (files-domain drift, exit 1) | live `/etc` edit path is the same drift code, untested live |
| drift_type_transition_is_modified | High | `test_drift_type_transition_is_modified_files_exit1`; unit `diff::tests::drift_type_transition_is_modified` | — |
| drift_ignores_unmanaged_packaged_file | High | `test_drift_ignores_changed_package_owned_undeclared_file_exit0`; unit `diff::tests::drift_ignores_unmanaged_packaged_file` | — |
| describe_records_symlink_verbatim | High | `test_describe_records_symlink_verbatim` (synthetic root) | — |
| describe_skips_special_file | High | `test_describe_skips_special_file` (fifo under synthetic root) | — |
| describe_traverses_etc_subdirectories | High | `test_describe_traverses_etc_subdirectories_no_isdir_error` | — |
| scope_attributes_always_object | High | `test_scope_attributes_always_object_never_null` | — |
| describe_emits_manifest | Medium | `test_describe_emits_manifest_format_version_1` (format_version verified) | full nginx/rpm resolution on a live host untested |
| describe_output_unwritable | High | `test_describe_output_unwritable_exit2` | — |
| describe_out_extension_yaml / _json | High | `test_describe_out_extension_yaml` / `_json` | — |
| describe_format_overrides_extension | High | `test_describe_format_overrides_extension` | — |
| describe_scope_full_emits_observational_scopes | Medium | `test_describe_default_scope_omits_observational_scopes` (negative half verified) | the positive half (full scan emits the scopes) needs a live `/usr` with packaged+unpackaged files; code-reviewed |
| describe_bootstraps_desired_manifest | Medium | offline diff round-trips a describe-shaped manifest (`test_diff_offline_two_files`) | live describe → diff round-trip untested |
| yaml_manifest_accepted | High | `test_yaml_manifest_accepted_offline_diff_exit0`; unit safe-profile tests | — |
| yaml_format_identity_stable | High | unit `hash::tests::json_and_yaml_models_hash_identically`; `test_yaml_json_identity_offline_both_clean` | — |
| yaml_unsafe_rejected | High | `test_yaml_unsafe_multidoc_rejected_exit1`; unit `manifest::tests::{yaml_multidoc_rejected, yaml_tag_rejected}` | — |
| describe_format_yaml | High | `test_describe_out_extension_yaml` (YAML emission) | — |
| verify_state_path_extension_yaml | High | `test_verify_state_path_yaml_extension_matches_exit0` | — |
| describe_repositories_from_reposd | Medium | unit `state::tests::parse_repo_ini_one_section` | live repos.d read untested at the binary level |
| describe_unreadable_scope_strict / _warn | Medium | code-reviewed; warn path exercised indirectly by synthetic-root describe tests | a genuinely-unreadable repos.d under strict mode untested at the binary level (requires a permission-denied fixture) |
| describe_omits_genuinely_empty_scope | Medium | code-reviewed; synthetic-root describe omits empty scopes | — |
| describe_suppresses_package_pristine_etc_file | Medium | code-reviewed (rpm -qf/-V driven) | requires a live rpmdb with a pristine + a changed file |
| describe_actual_state_omits_pristine | Medium | code-reviewed | live rpmdb needed |
| describe_config_files_bounded_to_etc | Medium | code-reviewed (`read_config_files` walks only `<root>/etc`) | scale assertion needs a many-package host |
| describe_verify_differences_not_unreadable | Medium | code-reviewed (`rpm -V` non-zero treated as changed, not unreadable) | live rpmdb needed |
| diff_offline_two_files | High | `test_diff_offline_two_files_lists_plan_exit0` | — |
| verify_default_scope_ignores_usr | Medium | code-reviewed (default scope=etc never scans /usr) | live /usr needed |
| verify_scope_full_detects_unmanaged_addition | Low | code-reviewed | needs live /usr full scan |
| verify_scope_full_detects_modified_package_file | Low | code-reviewed | needs live /usr full scan |
| describe_scope_full_boot_generated_files_unmanaged | Low | code-reviewed | needs live /boot full scan |
| apply_no_op_when_converged / idempotent_second_apply | Medium | offline intent-diff + drift emptiness verified by units; "nothing to do" path code-reviewed | needs a live applied record + undrifted system |
| apply_writes_and_deletes_etc_file | Low | code-reviewed (`converge_files`) | needs a live transaction |
| apply_absent_scope_unmanaged | Medium | unit `diff::tests::absent_scope_no_change`; code-reviewed apply path | live apply untested |
| apply_transaction_unavailable | Medium | code-reviewed (`acquire_transaction_context` external mode error → exit 2) | live external-mode probe untested |
| apply_package_failure_rolls_back | Low | code-reviewed (`converge_packages` error → discard, exit 1) | needs a live transaction |
| lock_is_fully_resolved_packages_scope | Low | code-reviewed (`converge_packages` queries rpmdb for the lock) | needs a live transaction |

## Resume logic

The output directory was empty at start; all deliverables were produced fresh.
No prior translation manifest was present.
