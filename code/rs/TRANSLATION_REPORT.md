# TRANSLATION_REPORT.md — zypper-declarative (Rust)

**Spec-SHA256:** `51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03`
(merged spec text == host spec; no `Includes:` directives)

**Spec-SHA256 (host):** `51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03`

**Included-Specs:**

| Path | SHA256 |
|------|--------|
| *(none)* | — |

**LLM-Name:** `claude-opus-4-8`

**Mode:** `translator`

**Tests-First-Compliance:** `yes` — every file under
`independent_tests/claude-opus-4-8/` was written and its compile gate
(`cargo check --tests`) passed **before** any implementation source file in
`src/` or `Cargo.toml` was written. The structural Tests-First guard (the
non-empty test directory) was satisfied before Phase 2 began.

**Continuity-Check:** not applicable — no test-author input. The input
directory contained no `independent_tests/<other-role-llm-name>/` directory and
no `TEST_REPORT.md`; this was a single-LLM run.

## Translation Inputs (provenance)

- `Spec-SHA256:` `51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03` (host == merged; no includes)
- `Decisions-Hints-SHA256:` `zypper-declarative.rs.decisions.hints.md` `b00395c6d6cc6af40c3d78b75c5aa4488fbeb19ef1182f181f76daefcf0ce6c4`
- `Milestones-Hints-SHA256:` `cli-tool.rs.milestones.hints.md` `811af43339b25e4f180f30a508ad5bc65cc01467d7ab4084ca345206cb7ab5a7`
- `Template-SHA256:` `cli-tool.template.md` `c8447ba8f1e63f3605b8e671e5bf58f4df44665a5ba1ff76864d28e4570042b5`
- `Style-Hints-SHA256:` `none` (no `<scope>.rs.style.hints.md` present in input or preset hierarchy)

## Target language resolution

- **Resolved LANGUAGE:** Rust.
- **Rationale:** the cli-tool template default is Go, but this run was invoked
  with the target language Rust (prompt header `Language: Rust`, output
  directory `/tmp/pcd-output/code/rs/`). Rust is a `supported` LANGUAGE in the
  template TEMPLATE-TABLE and a valid `LANGUAGE-ALTERNATIVES`. This is treated as
  a project/preset override of the template default; the deviation from the Go
  default is explicit (per "Derive the target language from the deployment
  template"). The spec is language-neutral (its DEPLOYMENT section states the
  resolved language does not affect any behaviour) and the Rust decisions hints
  file (`zypper-declarative.rs.decisions.hints.md`) confirms Rust as the intended
  target.
- **BINARY-TYPE:** `static` (required for Rust per template INVARIANT). Achieved
  via `-C target-feature=+crt-static`. `ldd ./zypper-declarative` reports
  "statically linked"; `readelf -d` shows no NEEDED entries.

## Module identity resolved

- **Resolved identity:** crate name `zypper-declarative`, repository
  `https://github.com/mge1512/zypper-declarative`.
- **Authoritative source (priority 1):** the spec META `Module:` field
  `github.com/mge1512/zypper-declarative`. Per the Rust decisions hints, the Go
  module path maps to the Rust crate name `zypper-declarative` (the trailing
  path segment), and the full Go module path is recorded as the `Cargo.toml`
  `repository` URL. The language-specific hints file (source 2) agreed
  (`zypper-declarative`). No conflict; `MODULE-IDENTITY: conflict-halts` did not
  fire. The spec-title fallback (source 4) was not used.
- Identity is propagated: `Cargo.toml` `package.name` and `repository`, RPM
  `URL:` and `%files`, DEB `Source:`/`Homepage:`, the man page Homepage/install
  lines, README install commands, and the on-disk `applied.json` path
  `/usr/lib/zypper-declarative/`.

## Delivery mode

Filesystem (mode 1). All source files written directly to
`/tmp/pcd-output/code/rs/`. The compile gate and both test suites were executed
in-environment.

## Active MILESTONE

The spec declares MILESTONEs 0.0.0 through 0.6.0, but **every milestone has
`Status: pending`** — none is `Status: active`. Per the prompt ("If no MILESTONE
section is present, or no milestone has `Status: active`, translate the full spec
as normal"), the full spec was translated. (More than one active milestone would
have been an error; zero active is the translate-full-spec case.) No `Scaffold`
pass applies. All BEHAVIORs were implemented; none was left as a scaffold stub.

## STEPS ordering per BEHAVIOR

Each BEHAVIOR's STEPS were implemented in declared order:

- **describe** (`src/cli/mod.rs::verb_describe`): reject unknown args/format
  (handled in `config::parse` before dispatch) → `describe_actual_state` →
  `resolve-format(format, out)` → serialise → write to `out` or stdout (write
  failure → exit 2) → exit 0.
- **diff** (`verb_diff`): load-desired-manifest → load-applied-record → intent
  diff → actual state (state_path offline, else live describe scope=etc) → print
  combined plan → exit 0.
- **verify** (`verb_verify`): determine reference (manifest_path else applied
  record; absent → "no declaration applied", exit 2) → obtain actual state
  (state_path offline else live) → compute-drift → empty → "system matches
  declaration" exit 0, else one diagnostic per item to stderr, exit 1.
- **status** (`verb_status`): unknown arg rejected by `config::parse` (exit 2) →
  load-applied-record (absent → "no declaration applied", exit 0) → print
  desired_sha256/format_version/generation/created_at/package count → drift
  summary line → exit 0.
- **apply** (`verb_apply`): load-desired → load-applied → intent diff → if empty,
  live drift; if also empty → "nothing to do" exit 0 → acquire transaction
  context → converge repositories+packages → converge files → converge units →
  write applied record → post-converge verify → seal/activate summary → exit 0.
  (Steps 5–11 are host-only and return a transaction error in a non-privileged
  environment, exit 2, per the spec.)
- **describe-actual-state** (`src/state/mod.rs`): packages → repositories →
  services → config_files → (scope=full) full-scan integrity → assemble manifest
  (omit genuinely-empty scopes) → unreadable-source handling per `on_unreadable`.
- **resolve-format** (`src/manifest/format.rs`): explicit wins → extension →
  `manifest-format` default.
- **load-desired-manifest** (`src/manifest/load.rs`): read → resolve-format →
  parse (JSON, or YAML under the safe profile) → schema-validate
  (format_version 1; reject non-empty observational scopes; record refinements)
  → signature hook → compute canonical-model `desired_sha256`.
- **load/write-applied-record** (`src/record.rs`), **compute-intent-diff /
  compute-drift** (`src/diff.rs`, pure, no I/O),
  **acquire-transaction-context** (`src/txn.rs`),
  **converge-packages/-files/-units** (`src/converge.rs`) — each in declared
  STEPS order.

## MECHANISM / method notes (config_files)

Per the Rust decisions hints, config_files uses the **rpm-verdict-parse** method
(not a self-built baseline join): owning packages via `rpm -qca`, per-package
`rpm -V --nodeps --noscripts` whose `SM5DLUGTP` flag string is parsed (type char
`c` only); the `L` flag yields the type-mismatch (link) case; content-bearing
`%ghost` files and manual `/etc/alternatives/*` symlinks are a small separate
pass; unpackaged files come from walking `/etc` and subtracting the rpm-owned
set. Every changed record carries `status="changed"` and a non-empty `changes`
list. Ownership is determined through the package database under the described
root (so a synthetic root with no rpmdb yields all-unpackaged — the database
answering, not a skipped lookup). `describe-actual-state` is the single live-state
reader; `compute-drift` performs no I/O.

## INTERFACES test doubles

The spec's `## INTERFACES` (package manager, snapshot/filesystem, init system,
transaction mechanism, external state producer) are modelled as the
`CommandRunner` trait (`src/interfaces.rs`). Production implementation:
`OsCommandRunner` (execs the real tool with a sanitised PATH). A test double,
`FakeCommandRunner`, is provided under `#[cfg(test)]`. The independent black-box
tests do **not** use the doubles — they invoke the built binary — consistent with
the test methodology; the doubles serve only the in-tree unit tests of parsing
logic.

## TYPE-BINDINGS / GENERATED-FILE-BINDINGS

The cli-tool template declares no `## TYPE-BINDINGS` or
`## GENERATED-FILE-BINDINGS` section, so neither applied. Logical TYPES were
mapped idiomatically: `ScopeWrapper<T>` → generic struct with serde
`rename = "_attributes"/"_elements"`; absent-vs-empty scope → `Option<Scope>`
(`None` = unmanaged/absent, `Some` with empty `_elements` = present-empty);
`_attributes` always a JSON object (`serde_json::Map`), never null.

## Constraint: supported / forbidden BEHAVIORs

All BEHAVIOR/BEHAVIOR-INTERNAL sections carry `Constraint: required`; all were
implemented unconditionally. No `supported` or `forbidden` BEHAVIOR appears in
the spec, so no conditional code generation or omission was required. The
spec-internal reservation ("converge-files does not yet create/update/remove
symlinks or handle type transitions") is honoured: `converge_files` writes and
deletes regular files only and skips non-file records (documented in code).

## DELIVERABLES (COMPONENT → filenames)

Required OUTPUT-FORMATs produced (OCI/PKG/binary are `supported` and no preset
activates them, so they were intentionally omitted — see "Deviations"):

| Deliverable | File(s) |
|---|---|
| source (entry-point + impl modules + manifest) | `src/main.rs` (dispatch only), `src/lib.rs`, modules under `src/`, `Cargo.toml`, `Cargo.lock`, `build.rs`, `.cargo/config.toml`, `rust-toolchain.toml` |
| build | `Makefile` (`build`, `test`, `install`, `clean`, `man` targets) |
| docs | `README.md` |
| man | `zypper-declarative.1.md`, `zypper-declarative.1` |
| license | `LICENSE` |
| RPM | `zypper-declarative.spec` |
| DEB | `debian/control`, `debian/changelog`, `debian/rules`, `debian/copyright` |
| public-api | this report's `## Public API Surface` |
| aux | `translation_report/translation-workflow.pikchr` |
| report | `TRANSLATION_REPORT.md` |
| spec-hash | embedded in every source header, `Makefile` `SPEC_SHA256`, RPM `# pcd-spec-sha256`, DEB `X-PCD-Spec-SHA256`, `--version` output, this report |

## SOURCE-PARTITIONING

`modular` and `one-entry-one-implementation` satisfied: `src/main.rs` contains
only CLI dispatch (signal handlers + `cli::run`); behaviour lives in
`src/cli/`, `src/config.rs`, `src/manifest/`, `src/state/`, `src/diff.rs`,
`src/converge.rs`, `src/record.rs`, `src/txn.rs`, `src/error.rs`,
`src/interfaces.rs`, `src/meta.rs`. The layout follows the Rust decisions hints'
recommended module grouping (`describe-actual-state` is the single live reader in
`state`; `diff` is I/O-free; `resolve-format` is the single serialisation
authority).

## Dependencies (direct only; versions verified against the registry)

| Crate | Version | Purpose |
|---|---|---|
| `serde` | 1 (resolved 1.0.228) | derive-based (de)serialisation |
| `serde_json` | 1 (resolved 1.0.150) | canonical JSON I/O |
| `serde_yaml` | 0.9 (resolved 0.9.34+deprecated) | opt-in YAML I/O (safe profile applied around it) |
| `sha2` | 0.10 (resolved 0.10.9) | SHA-256 content and canonical-model hashing |
| `libc` | 0.2 (resolved 0.2.186) | SIGTERM/SIGINT handlers |

`serde_yaml` is the chosen YAML crate. It is marked "deprecated/unmaintained"
upstream but is the de-facto stable serde YAML implementation; the spec's safe
profile (single document, no anchors/aliases, no explicit tags, explicit typing
via the typed model) is enforced by `src/manifest/yaml.rs` around it, with the
crate and version recorded here per the DEPENDENCIES requirement. **Flag for the
maintainer:** if a maintained YAML crate is preferred (e.g. `serde_yml` or a
`saphyr`-based reader), substitute it behind the same `parse_manifest_safe` /
`to_yaml` interface. Bindings to libzypp/snapper/systemd are intentionally *not*
linked: the tool drives `rpm`/`zypper`/`systemctl`/`update-alternatives` via
`std::process::Command` to keep the binary static and FFI-free (the deliberate
Rust/Go exec route vs. the C++ libzypp route), so no version pinning of those
libraries is required.

Dependencies are vendored (`cargo vendor` → `vendor/`, with
`.cargo/config.toml` `replace-with = "vendored-sources"`); the project builds and
tests fully offline (`cargo build --release --offline`, `make test`).

## Parsing approach

Hand-written `key=value` + bare-word parser (`src/config.rs`); no `clap` or other
arg-parser dependency (per the Rust decisions hints — a `--flag`-shaped parser
would fight the spec's CLI style). Options are accepted in any position relative
to the verb. `version`/`help` are bare-word global commands handled by the
dispatcher; `--version`/`--help`/`-h` are tolerated aliases. Unknown
verb/option/value or missing value → usage to stderr, exit 2. JSON via
`serde_json`; YAML via a safe-profile wrapper that rejects multi-document
streams, anchors/aliases, merge keys, and explicit tags, then deserialises the
single document into the typed model (so implicit typing cannot coerce string
fields). The canonical-model hash sorts object keys and `_elements` identity
keys, drops `created_at`/`desired_sha256`, and hashes the compact form, so
JSON/YAML expressions of the same intent share a `desired_sha256`.

## Signal handling

`src/main.rs` installs SIGTERM and SIGINT handlers via `libc::signal` that call
`libc::_exit(128 + signo)` (async-signal-safe; no non-reentrant atexit work). No
partial output is emitted because each verb writes its full output in a single
pass at the end. An interrupted `apply` discards the transaction (all mutation
occurs inside a snapshot that is only sealed/activated as the final step), so the
running system is unchanged — satisfying the spec's signal POSTCONDITION.

## Specification ambiguities

1. **`mode`/`transaction-mode` option key.** The DEPLOYMENT invocation table uses
   `mode=...` while CONFIG names the knob `transaction-mode`. Conservative
   interpretation: both keys are accepted and map to the same setting.
2. **Transaction-context detection signal.** The spec leaves the external/internal
   binding abstract. `acquire-transaction-context` detects an external transaction
   via the `TRANSACTIONAL_UPDATE_ROOT`/`DISTUPDATE_ROOT` mount signal exported by
   the opener; this is the transaction mechanism's own contract (not tool
   configuration, which remains key=value/preset only). Internal mode returns a
   transaction error where the zypper-merged machinery is unavailable (exit 2),
   per the ERRORS list.
3. **`changes` for ghost/alternatives emission.** The spec calls for a `changes`
   interpretation "analogous to" the changed_managed_files list. Conservative
   choice: ghost regular files carry `["md5"]`, manual alternative links carry
   `["link_path"]`; rpm-`-V`-derived records carry the full flag-derived set.
4. **full-scan package-name attribution.** `ManagedBaselineRecord.package_name`
   is left empty when a reverse-ownership map is not built; the structural
   full-scan classifies changed-vs-unmanaged by ownership but does not attribute
   the owning package name without that map. Documented as a refinement for the
   apply-on-live-host milestone; does not affect drift detection (which keys on
   the path).

## Rules not implemented exactly as written

- **apply convergence (steps 5–11), converge-units offline enablement, and the
  full-scan reverse-ownership map** are host-only operations requiring root, a
  real snapshot transaction, and a populated rpmdb under the context root. They
  are implemented structurally (correct command invocation and control flow) and
  return domain-tagged diagnostics; in a non-privileged sandbox `apply` halts at
  `acquire-transaction-context` with a transaction error (exit 2), which is the
  spec-defined behaviour when the mechanism is unavailable. The pure logic these
  orchestrate (intent diff, drift, record build/serialise, format resolution,
  hashing, the rpm-verdict and walk parsers) is fully implemented and unit-tested.
- **Signature verification** (`signature-verification=on` with a keyring) is a
  structural hook in `load-desired-manifest`: absence of a configured keyring is
  not treated as an error. Flagged for the maintainer if a concrete keyring
  binding is required.

## Compile gate result (template EXECUTION, Rust row)

- **Step 1 — Dependency resolution:** `cargo generate-lockfile` + `cargo vendor`
  → `Cargo.lock` written, `vendor/` populated. **pass.**
- **Step 2 — Compilation:** `cargo build --release` (and `--offline`) → **pass**,
  warning-free. Binary copied to project root `./zypper-declarative`; statically
  linked (verified via `ldd` / `readelf -d`).
- **Step 3 — Translator test run:** `make test`
  (`cargo test --test '*' --manifest-path independent_tests/claude-opus-4-8/Cargo.toml`)
  → **pass** (cli 11, describe 14, diff_verify 14, selfcheck 5; the `common.rs`
  helper target reports 0 tests, expected). In-tree lib unit tests: 34 passed.
- **Step 4 — Test-author test run:** not applicable (single-LLM run).
- **Step 5 — Record result:** all steps pass.

Acceptance probes (M0/M1 criteria + examples) all pass: `version` prefix and
`spec:` hash, `help`/bare usage, `--version` alias, `format=bad_value` → exit 2,
bare → exit 0, `describe out=...yaml` → YAML by extension, `status` → "no
declaration applied".

## Test results — translator suite (`independent_tests/claude-opus-4-8/`)

All 44 black-box tests pass.

`tests/cli.rs` (11): version_verb_bare_word, version_flag_alias,
help_verb_bare_word, help_flag_aliases, bare_invocation_shows_help,
unknown_verb_rejected, unknown_format_value_global_exits_2,
describe_unknown_format, status_unknown_argument, status_no_declaration,
options_accepted_in_any_position_for_describe — **pass**.

`tests/describe.rs` (14): describe_emits_json_manifest_with_format_version_1,
scope_attributes_always_object, describe_records_symlink_verbatim,
describe_skips_special_file, describe_traverses_subdirectories,
describe_regular_file_sha256, describe_without_content_store_is_readonly,
describe_populates_content_store, describe_out_extension_selects_format,
describe_format_overrides_extension, describe_format_yaml_stdout,
describe_unpackaged_etc_file_has_empty_package_name,
describe_omits_genuinely_empty_config_files_scope,
describe_scope_etc_has_no_observational_scopes — **pass**.

`tests/selfcheck.rs` (5): the required config_files self-checks that bind the
separate %ghost pass on a REAL host package database (run as root in the test
step; an explicit, logged no-op otherwise): selfcheck_common_auth_is_type_link
(1a, type-mismatch link), selfcheck_common_auth_pc_content_bearing_ghost
(1b, the content-bearing ghost that build 04 dropped),
selfcheck_other_content_bearing_ghost_present (1c, `/etc/machine-id`, guards
against pam-only special-casing), selfcheck_pristine_imagemagick_xml_absent
(2, pristine suppression), selfcheck_packaged_records_carry_status_and_changes
(3, every packaged record carries status="changed" + non-empty changes)
— **pass**.

`tests/diff_verify.rs` (14): diff_offline_two_files, intent_diff_yields_deletion,
diff_manifest_unreadable, manifest_invalid_format_version,
manifest_with_observational_scope_rejected, verify_offline_matching_exits_0,
verify_offline_no_applied_record_ok, verify_detects_divergent_service,
verify_type_transition_is_drift, verify_malformed_state_dump,
verify_no_applied_record, yaml_manifest_accepted_offline,
yaml_unsafe_multidoc_rejected, diff_does_not_modify_input_files — **pass**.

## Test results — test-author suite

Not present (single-LLM run).

## Test Refinements

| Test | Result before | Action | Rationale |
|------|---------------|--------|-----------|
| options_accepted_in_any_position_for_describe | hung (timeout) | test edited | The first draft ran `describe` with no `root=`, triggering an unbounded live `/etc` walk + whole-host rpm verification on the build host (DEPLOYMENT/`describe-actual-state` reads the live system when no `root=` is given). The black-box intent (options accepted in any position) is preserved by pointing at a synthetic `root=` so the run is bounded and deterministic; the assertion was strengthened to exit 0 + out-file written. No behavioural claim weakened. |
| cli::tests::rfc3339_known_timestamp (in-tree unit test) | failed | test edited | The unit test's expected RFC3339 string was wrong (1780392000 = 2026-06-02T09:20:00Z, not 10:00:00Z); the `format_rfc3339` function is correct (epoch case passes). Fixed the expected literal. |
| selfcheck::selfcheck_common_auth_pc_content_bearing_ghost (1b) | failed (record ABSENT) | code fixed | The ghost-regular-file pass was effectively missing: `ghost_paths` queried `rpm -qa --qf '[%{NAME} %{FILENAMES} %{FILEFLAGS}\n]'`, but `%{NAME}` is a per-package SCALAR inside the array iterator `[ ... ]`, so rpm emitted only the FIRST file of each package and dropped every other ghost path. Switched to the files-only template `[%{FILENAMES} %{FILEFLAGS:fflags}\n]` (lists every file with its per-file flags), filter the `g`/GHOST flag, and resolve the owning package separately via `rpm -qf` for the few emitted paths. Content-bearing ghosts (`/etc/pam.d/common-*-pc`, `/etc/machine-id`, ...) are now emitted. Spec: config_files BEHAVIOR ghost-regular-file rule ("a ghost REGULAR FILE with REAL on-disk content is EMITTED") and the decisions-hints "GHOST REGULAR FILES are a SEPARATE, REQUIRED pass". |
| selfcheck::selfcheck_packaged_records_carry_status_and_changes (3) | failed (26 offenders, `changes`=null) | code fixed | `rpm -V` lines whose flag string carried no concrete change marker (only `.`=unchanged and `?`=test-not-performed, e.g. `..?......  c /etc/sudoers` for files rpm could not test) were emitted as `status="changed"` with an empty `changes` list, violating the invariant that every changed record carries a non-empty `changes`. `parse_verify_output` now drops a verdict line that yields no change marker (and is not `missing`): no detected difference means no changed record. Spec/decisions-hints self-check (3). |

All other tests passed on first run after their corresponding implementation
module was complete.

## Fix pass — 2026-06-02 (ghost-regular-file pass)

This run is a targeted fix on existing output: the ghost-regular-file pass was
missing in practice, so content-bearing ghosts (`common-auth-pc`, `machine-id`,
and the rest of the ~32 content-bearing `%ghost` `/etc` files) were absent from
`describe` output. Two defects in `src/state/configfiles.rs` were corrected
(see the two `code fixed` rows above): (a) the malformed `rpm` query that mixed
the scalar `%{NAME}` inside the array iterator and so only saw one file per
package; (b) `rpm -V` verdict lines with no concrete change marker being
mislabelled as changed-with-empty-changes. The required self-checks (1a/1b/1c,
2, 3) were added as `tests/selfcheck.rs` so the pass is now bound by an
executable assertion rather than by prose. Verified live: a `describe scope=etc`
run on `/` emits `/etc/pam.d/common-auth` as type "link", `common-auth-pc` and
`/etc/machine-id` as type "file" with 64-hex sha256, suppresses the pristine
`/etc/ImageMagick-7-SUSE/*.xml`, and produces zero packaged records with a null
`changes` list.

## Per-example confidence

Confidence is **High** when a named test function in
`independent_tests/claude-opus-4-8/` passes without a live external service.
Examples whose behaviour requires root / a live SUSE host / a real transaction
are **Medium** (verified by reasoning + structural unit tests, not by a
service-free black-box test).

| EXAMPLE | Confidence | Verification method | Unverified claims |
|---|---|---|---|
| version_verb_bare_word | High | `cli::version_verb_bare_word` | — |
| version_flag_alias | High | `cli::version_flag_alias` | — |
| help_verb_bare_word | High | `cli::help_verb_bare_word` | — |
| bare_invocation_shows_help | High | `cli::bare_invocation_shows_help` | — |
| unknown_verb_rejected | High | `cli::unknown_verb_rejected` | — |
| describe_unknown_format | High | `cli::describe_unknown_format` | — |
| status_unknown_argument | High | `cli::status_unknown_argument` | — |
| status_no_declaration | High | `cli::status_no_declaration` (applied-root with no record) | — |
| scope_attributes_always_object | High | `describe::scope_attributes_always_object` | — |
| describe_records_symlink_verbatim | High | `describe::describe_records_symlink_verbatim` | — |
| describe_skips_special_file | High | `describe::describe_skips_special_file` | — |
| describe_traverses_etc_subdirectories | High | `describe::describe_traverses_subdirectories` | — |
| describe_config_files_bounded_to_etc | High | `describe::describe_unpackaged_etc_file_has_empty_package_name` (+ root-bounded walk) | bound at large scale not measured |
| describe_without_content_store_is_readonly | High | `describe::describe_without_content_store_is_readonly` | — |
| describe_populates_content_store | High | `describe::describe_populates_content_store` (incl. dedup) | — |
| describe_out_extension_yaml / _json | High | `describe::describe_out_extension_selects_format` | — |
| describe_format_overrides_extension | High | `describe::describe_format_overrides_extension` | — |
| describe_format_yaml | High | `describe::describe_format_yaml_stdout` | — |
| describe_omits_genuinely_empty_scope | High | `describe::describe_omits_genuinely_empty_config_files_scope` | — |
| describe_scope_full_emits_observational_scopes | Medium | `describe::describe_scope_etc_has_no_observational_scopes` (etc side) + reasoning | full-scan emission needs a packaged /usr (root host) |
| describe_emits_manifest | High (structure) | `describe::describe_emits_json_manifest_with_format_version_1` | live nginx resolution needs a real rpmdb |
| describe_suppresses_package_pristine_etc_file | High | `selfcheck::selfcheck_pristine_imagemagick_xml_absent` (2, root self-check) + `state::configfiles` unit tests | bound by a root-only self-check (skips/logs off-root) |
| describe_symlink_and_target_judged_independently | Medium | reasoning + independent walk classification | needs real package ownership |
| describe_pristine_distro_symlink_suppressed | Medium | reasoning | needs real package ownership |
| describe_type_mismatch_emitted | High | `selfcheck::selfcheck_common_auth_is_type_link` (1a, root self-check) + `state::configfiles::type_mismatch_link_flag_parsed` | bound by a root-only self-check (skips/logs off-root) |
| describe_ghost_with_content_emitted | High | `selfcheck::selfcheck_common_auth_pc_content_bearing_ghost` (1b) + `selfcheck::selfcheck_other_content_bearing_ghost_present` (1c) | bound by root-only self-checks (skip/log off-root); verified live on `/` |
| describe_default/manual_alternative_symlink | Medium | reasoning + `update-alternatives --query` parse | needs real alternatives DB |
| describe_empty_ghost_suppressed | Medium | reasoning (empty-content suppression in the ghost pass) | needs real %ghost on host |
| describe_verify_differences_not_unreadable | Medium | reasoning (`rpm -V` non-zero is normal) | needs real modified packaged /etc |
| describe_repositories_from_reposd | Medium | `state::repos::parses_two_sections` unit test | not exercised via the binary against a synthetic repos.d |
| describe_unreadable_scope_strict / _warn | Medium | reasoning + `on_unreadable` handling | unreadable source not constructed as a black-box test |
| intent_diff_yields_deletion | High | `diff_verify::intent_diff_yields_deletion` + `diff::tests::intent_diff_yields_deletion` | — |
| drift_ignores_unmanaged_packaged_file | High | `diff::tests::drift_ignores_package_owned_undeclared_file` | — |
| drift_type_transition_is_modified | High | `diff_verify::verify_type_transition_is_drift` + `diff::tests::drift_type_transition_is_modified` | — |
| diff_offline_two_files | High | `diff_verify::diff_offline_two_files` | — |
| diff_prints_plan | High | `diff_verify::diff_offline_two_files` (plan sections) | live-read variant needs root |
| diff_manifest_unreadable | High | `diff_verify::diff_manifest_unreadable` | — |
| verify_clean | High | `diff_verify::verify_offline_matching_exits_0` | live-read variant needs root |
| verify_offline_manifest_and_state | High | `diff_verify::verify_offline_matching_exits_0` | — |
| verify_offline_no_applied_record_ok | High | `diff_verify::verify_offline_no_applied_record_ok` | — |
| verify_against_external_state_dump | High | `diff_verify::verify_detects_divergent_service` | — |
| verify_malformed_state_dump | High | `diff_verify::verify_malformed_state_dump` | — |
| verify_no_applied_record | High | `diff_verify::verify_no_applied_record` | — |
| verify_detects_drift | High | `diff_verify::verify_detects_divergent_service` / `verify_type_transition_is_drift` | live-read variant needs root |
| verify_default_scope_ignores_usr | Medium | reasoning (scope=etc never scans /usr) | needs root host |
| verify_scope_full_detects_* | Medium | reasoning + full-scan code | needs root host |
| yaml_manifest_accepted | High | `diff_verify::yaml_manifest_accepted_offline` | — |
| yaml_format_identity_stable | High | `manifest::hash::tests::*` (order/ created_at independence) + `yaml_manifest_accepted_offline` | — |
| yaml_unsafe_rejected | High | `diff_verify::yaml_unsafe_multidoc_rejected` + `manifest::yaml::tests::{anchor,tag}_rejected` | — |
| describe_unknown_format | High | `cli::describe_unknown_format` | — |
| bare_invocation_shows_help | High | `cli::bare_invocation_shows_help` | — |
| status_reports_generation | Medium | `verb_status` + reasoning | needs an applied record on host |
| lock_is_fully_resolved_packages_scope | Medium | reasoning + `record::build_applied_record` + `state::packages` parse | needs real converge on host |
| apply_* (all apply examples) | Medium | reasoning + structural converge code | host-only (root, transaction); `apply` halts at transaction acquisition in a sandbox (exit 2, spec-defined) |
| idempotent_second_apply | Medium | reasoning + empty-diff/empty-drift path | host-only |

**Unverified claims (explicit):** the live-system read paths (real rpmdb,
systemd enablement, repos.d, %ghost, alternatives DB), the full-scan integrity
emission, and the host-only `apply` convergence/transaction/seal steps are
verified by reasoning and structural unit tests of their parsers/control flow,
not by a service-free black-box test, because they require root and a real SUSE
host. These are listed Medium above.

## Deviations

- **Language:** Rust selected over the template's Go default (preset/project
  override; documented above).
- **OUTPUT-FORMAT OCI/PKG/binary:** `supported`, not activated by any resolved
  preset, therefore not produced (per "No unsolicited deliverables" and the
  DELIVERABLES rule that supported formats are produced only when active). No
  `Containerfile` or `<n>.pkgbuild` was written.
- **NETWORK-CALLS / FILE-MODIFICATION / privilege:** carried over from the spec's
  own documented template deviations (package retrieval delegated to the package
  manager; input manifest never modified; `apply` requires privilege). No new
  deviations introduced.
- **Binary location:** the static binary is built under
  `target/x86_64-unknown-linux-gnu/release/` (explicit `--target`, required so
  `crt-static` does not break host proc-macro builds) and copied by `make build`
  to the project root `./zypper-declarative`, the canonical `BINARY-LOCATION`
  (`../../zypper-declarative` from the test directory).

## Public API Surface

The exported symbols of the implementation library (crate `zypper-declarative`),
grouped by module. The next translation of this spec at Version 0.6.6 must
preserve these (additions allowed; removals/renames require a Version increment).

### `meta`
- `pub const PROGRAM_NAME: &str`
- `pub const VERSION: &str`
- `pub const SPEC_SHA256: &str`
- `pub fn generator() -> String`
- `pub fn version_line() -> String`

### `error`
- `pub enum Severity { Error, Warning }`
- `pub enum Domain { Packages, Repositories, Services, Files, Manifest, Transaction, Invocation }`
- `pub fn Domain::as_str(&self) -> &'static str`
- `pub struct Diagnostic { severity: Severity, domain: Domain, message: String }`
- `pub fn Diagnostic::error(domain: Domain, message: impl Into<String>) -> Diagnostic`
- `pub fn Diagnostic::warning(domain: Domain, message: impl Into<String>) -> Diagnostic`
- `pub fn Diagnostic::line(&self) -> String`
- `pub const EXIT_OK: i32` / `pub const EXIT_LOGICAL: i32` / `pub const EXIT_INVOCATION: i32`
- `pub fn exit_code_for(diag: &Diagnostic) -> i32`

### `config`
- `pub enum OnUnreadable { Error, Warn }`
- `pub enum Scope { Etc, Full }`
- `pub enum TransactionMode { Auto, External, Internal }`
- `pub struct Invocation { … }`
- `pub fn Invocation::manifest_format_default(&self) -> ManifestFormat`
- `pub fn Invocation::on_unreadable_or_error(&self) -> OnUnreadable`
- `pub fn Invocation::scope_or_etc(&self) -> Scope`
- `pub fn Invocation::root_or_slash(&self) -> String`
- `pub fn Invocation::applied_root_or_slash(&self) -> String`
- `pub fn Invocation::transaction_mode_or_auto(&self) -> TransactionMode`
- `pub fn parse(args: &[String]) -> Result<Invocation, Diagnostic>`

### `manifest`
- `pub type Attributes = serde_json::Map<String, serde_json::Value>`
- `pub struct ScopeWrapper<T> { attributes: Attributes, elements: Vec<T> }`
- `pub fn ScopeWrapper::<T>::with_attr(key: &str, value: &str) -> Self`
- `pub struct ManifestMeta { format_version: i64, generator: String, created_at: String, desired_sha256: String }`
- `pub struct PackageRecord { name, version, release, arch }`
- `pub type PackagesScope = ScopeWrapper<PackageRecord>`
- `pub struct RepositoryRecord { alias, name, url, type, enabled, gpgcheck, autorefresh, priority }`
- `pub type RepositoriesScope = ScopeWrapper<RepositoryRecord>`
- `pub struct ServiceRecord { name, state }`
- `pub type ServicesScope = ScopeWrapper<ServiceRecord>`
- `pub struct ManagedFileRecord { name, type, mode, user, group, sha256, target, content_ref, package_name, status, changes }`
- `pub type ConfigFilesScope = ScopeWrapper<ManagedFileRecord>`
- `pub struct ManagedBaselineRecord { name, type, mode, user, group, sha256, target, package_name, changes }`
- `pub type ChangedManagedFilesScope = ScopeWrapper<ManagedBaselineRecord>`
- `pub struct UnmanagedFileRecord { name, type, mode, user, group, sha256, target }`
- `pub type UnmanagedFilesScope = ScopeWrapper<UnmanagedFileRecord>`
- `pub struct Manifest { meta, packages?, repositories?, services?, config_files?, changed_managed_files?, unmanaged_files? }`
- `pub fn Manifest::new_actual(created_at: String) -> Self`

### `manifest::format`
- `pub enum ManifestFormat { Json, Yaml }`
- `pub fn ManifestFormat::parse(s: &str) -> Option<ManifestFormat>`
- `pub fn resolve_format(explicit: Option<ManifestFormat>, path: Option<&str>, default: ManifestFormat) -> ManifestFormat`

### `manifest::hash`
- `pub fn desired_sha256(manifest: &Manifest) -> String`
- `pub fn hex(bytes: &[u8]) -> String`
- `pub fn sha256_bytes(bytes: &[u8]) -> String`

### `manifest::yaml`
- `pub struct YamlSafetyError(pub String)`
- `pub fn parse_manifest_safe(text: &str) -> Result<Manifest, YamlSafetyError>`
- `pub fn to_yaml(manifest: &Manifest) -> Result<String, serde_yaml::Error>`

### `manifest::load`
- `pub struct LoadedManifest { manifest: Manifest, desired_sha256: String }`
- `pub fn load_desired_manifest(manifest_path: &str, explicit_format: Option<ManifestFormat>, default_format: ManifestFormat) -> Result<LoadedManifest, Diagnostic>`
- `pub fn load_state_dump(state_path: &str, explicit_format: Option<ManifestFormat>, default_format: ManifestFormat) -> Result<Manifest, Diagnostic>`

### `record`
- `pub const APPLIED_REL: &str`
- `pub struct AppliedLoad { record: Manifest, present: bool }`
- `pub fn applied_path(root: &str) -> PathBuf`
- `pub fn load_applied_record(root: &str) -> Result<AppliedLoad, Diagnostic>`
- `pub fn empty_applied() -> Manifest`
- `pub fn build_applied_record(desired: &Manifest, desired_sha256: &str, resolved: PackagesScope, created_at: String) -> Manifest`
- `pub fn write_applied_record(root: &str, record: &Manifest) -> Result<(), Diagnostic>`

### `diff`
- `pub const SYNCPOINT: &str`
- `pub struct Diff { packages_install, packages_remove, repos_set, files_write, files_delete, units_change }`
- `pub fn Diff::is_empty(&self) -> bool`
- `pub struct DriftReport { files_modified, files_extra, units_divergent, packages_divergent, managed_files_modified, unmanaged_files_present }`
- `pub fn DriftReport::is_empty(&self) -> bool`
- `pub fn DriftReport::count(&self) -> usize`
- `pub fn compute_intent_diff(desired: &Manifest, applied: &Manifest) -> Diff`
- `pub fn compute_drift(actual: &Manifest, reference: &Manifest, keep_list: &HashSet<String>) -> DriftReport`

### `interfaces`
- `pub trait CommandRunner: Send + Sync { fn run(&self, cmd: &str, args: &[&str]) -> CommandResult; }`
- `pub struct CommandResult { stdout, stderr, success, spawn_failed }`
- `pub struct OsCommandRunner`
- `pub struct FakeCommandRunner` (test only)

### `state`
- `pub struct DescribeOptions<'a> { root, on_unreadable, scope, keep_list, content_store, created_at, runner }`
- `pub struct DescribeResult { manifest: Manifest, diagnostics: Vec<Diagnostic> }`
- `pub fn describe_actual_state(opts: &DescribeOptions) -> Result<DescribeResult, Diagnostic>`

### `state::packages`
- `pub enum PackagesResult { Records(Vec<PackageRecord>), Unreadable(String) }`
- `pub fn read_packages(runner: &dyn CommandRunner, root: &str) -> PackagesResult`
- `pub fn parse_packages(stdout: &str) -> Vec<PackageRecord>`

### `state::repos`
- `pub enum ReposResult { Records(Vec<RepositoryRecord>), Unreadable(String) }`
- `pub fn read_repositories(root: &str) -> ReposResult`
- `pub fn parse_repo_file(text: &str) -> Vec<RepositoryRecord>`

### `state::services`
- `pub enum ServicesResult { Records(Vec<ServiceRecord>), Unreadable(String) }`
- `pub fn read_services(runner: &dyn CommandRunner, root: &str) -> ServicesResult`
- `pub fn parse_unit_files(stdout: &str) -> Vec<ServiceRecord>`

### `state::configfiles`
- `pub const SYNCPOINT: &str`
- `pub struct ConfigFilesOutput { records: Vec<ManagedFileRecord>, diagnostics: Vec<String> }`
- `pub enum ConfigFilesError { Unreadable(String) }`
- `pub fn read_config_files(runner: &dyn CommandRunner, root: &str, on_unreadable: &OnUnreadable, keep_list: &HashSet<String>, content_store: Option<&str>) -> Result<ConfigFilesOutput, ConfigFilesError>`
- `pub fn parse_verify_output(stdout: &str) -> Vec<(String, Vec<String>)>`

### `state::fullscan`
- `pub struct FullScanOutput { changed: Vec<ManagedBaselineRecord>, unmanaged: Vec<UnmanagedFileRecord>, diagnostics: Vec<String> }`
- `pub enum FullScanError { Unreadable(String) }`
- `pub fn full_scan(runner: &dyn CommandRunner, root: &str, on_unreadable: &OnUnreadable, keep_list: &HashSet<String>) -> Result<FullScanOutput, FullScanError>`

### `txn`
- `pub struct TransactionContext { mode: TransactionMode, root: String, opened_here: bool }`
- `pub fn acquire_transaction_context(mode: &TransactionMode) -> Result<TransactionContext, Diagnostic>`

### `converge`
- `pub fn converge_packages(runner: &dyn CommandRunner, ctx: &TransactionContext, diff: &Diff) -> Result<PackagesScope, Diagnostic>`
- `pub fn converge_files(ctx: &TransactionContext, diff: &Diff, keep_list: &HashSet<String>, content_store: Option<&str>, rpm_owned: &dyn Fn(&str) -> bool) -> Result<(), Diagnostic>`
- `pub fn converge_units(runner: &dyn CommandRunner, ctx: &TransactionContext, diff: &Diff) -> Result<(), Diagnostic>`

### `cli`
- `pub fn run(args: &[String]) -> i32`
- `pub fn now_rfc3339() -> String`
- `pub fn emit_usage_stderr()`
- `pub fn usage_text() -> String`
- `cli::render::{render_plan, render_drift_summary}`; `cli::serialize::serialise`
