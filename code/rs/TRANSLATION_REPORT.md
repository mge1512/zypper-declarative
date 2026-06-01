# TRANSLATION_REPORT.md — zypper-declarative (Rust)

- **Spec-SHA256:** `18253550a5c3d3f1818f0380811cb5dbc98874828693e49fc9cd5cbc923303dd` (merged spec; the host spec declares no `Includes:`, so the merged hash equals the host hash)
- **Spec-SHA256 (host):** `18253550a5c3d3f1818f0380811cb5dbc98874828693e49fc9cd5cbc923303dd`
- **Included-Specs:**

  | Path | SHA256 |
  |------|--------|
  | *(none)* | — |

- **LLM-Name:** `claude-opus-4-8`
- **Mode:** `translator`
- **Deployment-Template:** `cli-tool.template.md v0.3.29`
- **Target language:** Rust (edition 2021). Resolved from the prompt's explicit target (`Language: RUST`) and confirmed by the `*.rs.decisions.hints.md` / `cli-tool.rs.milestones.hints.md` presence. The cli-tool template default is Go; Rust is a `supported` LANGUAGE alternative selected here by the invocation. No project preset was present; the deviation from the template default is the explicit translator-run target.
- **Tests-First-Compliance:** `yes`. The independent test suite under `independent_tests/claude-opus-4-8/` (the `Cargo.toml` and `tests/cli.rs`) was written and compiled (`cargo check --tests` → exit 0) **before** any non-test source file. The Tests-First structural guard (step 3 of the translator flow) was satisfied: the directory contained a compiling test file before Phase 2 began.
- **Continuity-Check:** not applicable — no test-author input. There is no `independent_tests/<other-role-llm-name>/` directory or `TEST_REPORT.md` in the input directory; this is a single-LLM run.

## Spec composition

The host spec META declares `Spec-Schema: 0.4.0` and **no** `Includes:`
directives. The merge described in the prompt's Spec Composition section is
therefore trivial: the merged spec equals the host spec. The merged-spec hash
embedded in all artefacts equals the host-spec hash. The `0.4.0` forward-
compatibility requirement is honoured: the resolver handled `Includes:` (an
empty set) rather than silently ignoring the directive.

## Module identity resolution

The spec META declares `Module: github.com/mge1512/zypper-declarative`
(authoritative source 1). Per the Rust decisions hints, this Go-style module
path is mapped to the Rust crate name `zypper-declarative`, with the URL
recorded as `Cargo.toml` `repository`. No conflict: source 1 is the sole
identity source consulted; the hints file agrees (it pins the same crate name).
The spec-title fallback (source 4) was **not** used. Identity propagates to
`Cargo.toml` (`package.name`, `repository`), the `[[bin]]`/`[lib]` names, the
RPM `Name:`/`URL:`, the DEB `Source:`/`Homepage:`, the man page, and the README.

## Active milestone

No `## MILESTONE:` section has `Status: active` — every milestone in the spec is
`Status: pending`. Per the prompt ("If no MILESTONE section is present, or no
milestone has `Status: active`, translate the full spec as normal"), the **full
spec** was translated: all five CLI verbs (`apply`, `diff`, `verify`, `status`,
`describe`) and all eleven `BEHAVIOR/INTERNAL` behaviours
(`load-desired-manifest`, `load-applied-record`, `compute-intent-diff`,
`compute-drift`, `describe-actual-state`, `resolve-format`,
`acquire-transaction-context`, `converge-packages`, `converge-files`,
`converge-units`, `write-applied-record`). No BEHAVIOR is `Constraint:
forbidden`; all are `required` and implemented unconditionally. No BEHAVIOR is
`Constraint: supported`. No "not yet scheduled" BEHAVIORs.

## Delivery mode

Filesystem (mode 1). All files written directly to
`/tmp/pcd-output/code/rs/`. The compile gate and both test suites were executed
in-environment. Dual-LLM was not in scope (no test-author output present).

## Source partitioning (SOURCE-PARTITIONING: modular, one-entry-one-implementation)

The entry point `src/main.rs` contains only CLI dispatch (argument collection,
signal-handling note, calling into the verb layer). All behaviour lives in a
separate module tree under `src/`, mirroring the spec's behaviour grouping
(per the Rust decisions hints layout):

| Module | Responsibility |
|--------|----------------|
| `src/main.rs` | entry point; dispatch only |
| `src/cli/mod.rs` | verb layer: key=value parsing, the global contract, the five verbs, exit-code mapping |
| `src/manifest/mod.rs` | the shared data model (serde types, `ScopeWrapper`, absent-vs-empty as `Option<Scope>`) |
| `src/manifest/format.rs` | `resolve-format` — the single serialisation authority |
| `src/manifest/hash.rs` | canonical-model `desired_sha256` |
| `src/manifest/serialize.rs` | JSON/YAML read/write + the safe YAML profile |
| `src/load.rs` | `load-desired-manifest` (parse, schema-validate, signature, hash) |
| `src/record/mod.rs` | `load-applied-record`, `write-applied-record`, state-dump loading |
| `src/diff/mod.rs` | `compute-intent-diff`, `compute-drift` (pure, no I/O) |
| `src/state/mod.rs` | `describe-actual-state` — the single live-state reader |
| `src/txn/mod.rs` | `acquire-transaction-context` + bindings |
| `src/converge/mod.rs` | `converge-packages`, `converge-files`, `converge-units` |
| `src/interfaces.rs` | the `CommandRunner` trait, the production `OsCommandRunner`, and a test double |
| `src/config.rs` | CONFIG knobs and the resolved `Config` |
| `src/error.rs` | `Diagnostic`, `Severity`, `Domain`, `ExitCode`, exit mapping |
| `src/clock.rs` | dependency-free RFC3339 wall-clock formatter |
| `src/meta.rs` | embedded spec hash, version, generator string |

The entry point implements no behaviour. The partitioning is by behavioural
domain, satisfying both the `modular` and `one-entry-one-implementation`
constraints.

## STEPS ordering per BEHAVIOR

Each BEHAVIOR's STEPS were implemented in the order written:

- **apply** (`src/cli/mod.rs::verb_apply`): load desired (1) → load applied (2)
  → intent diff (3) → if empty, read actual + drift; nothing-to-do without a
  transaction (4) → acquire transaction (5) → converge packages after repos (6)
  → converge files (7) → converge units (8) → write applied record (9) →
  post-converge verify against the new applied record (10) → seal/activate +
  summary (11). The signature class maps: read/unknown-format → exit 2;
  schema/unsafe-YAML/signature → exit 1; transaction unavailable → exit 2;
  convergence failure → exit 1 (transaction discarded).
- **diff** (`verb_diff`): load desired (1) → load applied (2) → intent diff (3)
  → actual state (state-path offline, else live) (4) → print combined plan,
  exit 0 (5).
- **verify** (`verb_verify`): determine reference (manifest-path else applied
  record; "no declaration applied" → exit 2 when neither) (1) → actual state
  (state-path offline, else live) (2) → drift (3) → empty → exit 0 with
  "system matches declaration", else one diagnostic per item → exit 1 (4).
- **status** (`verb_status`): reject unrecognised argument → exit 2 (1) → load
  applied record; absent → "no declaration applied", exit 0 (2) → print hash,
  format_version, generation, created_at, package count (3) → live drift
  summary line (4).
- **describe** (`verb_describe`): reject unrecognised arg / unknown format →
  exit 2 (1) → obtain actual state, on_unreadable+scope (2) → resolve output
  format via `resolve-format(format, out)` (3) → serialise (4) → write to
  out/stdout; unwritable → exit 2 (5).
- **describe-actual-state** (`src/state/mod.rs`): packages from rpmdb (1) →
  repositories from `/etc/zypp/repos.d/*.repo` (2) → services via offline
  `systemctl --root` enablement (3) → config_files `/etc` walk with bulk
  ownership + pristine suppression (4) → full-scan integrity only under
  scope=full (4a) → assemble Manifest, omitting genuinely-empty scopes (5) →
  unreadable-source handling per on_unreadable (6).
- The internal behaviours **resolve-format**, **load-desired-manifest**,
  **load-applied-record**, **compute-intent-diff**, **compute-drift**,
  **acquire-transaction-context**, **converge-packages**, **converge-files**,
  **converge-units**, **write-applied-record** each follow their STEPS in their
  respective modules.

`MECHANISM:` annotations: the spec uses none in the STEPS lists; none to apply.

## INTERFACES test doubles

The spec's `## INTERFACES` section lists abstract external dependencies (package
manager, snapshot/filesystem, init system, transaction mechanism, optional
external state producer) rather than named in-code interfaces with declared test
doubles. The in-code seam is the `CommandRunner` trait (`src/interfaces.rs`)
with the production `OsCommandRunner` and an in-tree `FakeCommandRunner` double.
The **independent** test suite uses neither: it is black-box and invokes the
built binary via `std::process::Command` (per the test methodology), so it does
not import production code or the double. No `<INTERFACE_PLACEHOLDER>` markers
were required (this is a `cli-tool`, not a library template).

## TYPE-BINDINGS / GENERATED-FILE-BINDINGS

The cli-tool template declares no `## TYPE-BINDINGS` and no
`## GENERATED-FILE-BINDINGS` section, so neither applies. Spec logical TYPES were
mapped to Rust types directly: `ScopeWrapper<T>` → generic serde struct;
absent-vs-empty scope → `Option<ScopeWrapper<T>>`; `Sha256`/`Mode`/`UnitName`
refinements → validated `String` fields; `ExitCode`/`Severity`/`TransactionMode`
/`ManifestFormat`/`ScanScope`/`OnUnreadable` → Rust enums.

## COMPONENT → filename mapping

The spec's DEPLOYMENT does not use a DELIVERABLES `COMPONENT:` table; the single
binary maps to `zypper-declarative` (project-root binary) per the cli-tool
template Naming Convention.

## Constraint handling

Every BEHAVIOR is `Constraint: required` and was implemented. No `supported` or
`forbidden` BEHAVIOR exists, so no conditional code generation or omission was
needed. Template `forbidden` rows were honoured: no environment-variable control
(env is never read for behaviour), no network calls of the tool's own (package
retrieval is delegated to the package manager — documented deviation, see
below), no input-file modification, no curl install method.

## Documented template deviations

- **NETWORK-CALLS: forbidden** — the tool performs no direct network I/O; all
  package retrieval is delegated to the package manager against a declared,
  pinned, signed repository (spec DEPLOYMENT "Template deviations"). The
  supply-chain intent (no curl-style fetching) is fully honoured. The Rust build
  drives `zypper`/`rpm`/`snapper`/`systemctl` via `std::process::Command` (the
  exec route, per the decisions hints), keeping the binary free of FFI and
  statically linkable.
- **BINARY-TYPE: static** — achieved via `RUSTFLAGS='-C
  target-feature=+crt-static'` with an explicit `--target
  x86_64-unknown-linux-gnu`. The explicit target is required because, on the
  build toolchain (Rust 1.95.0, stable), a global crt-static rustflag (in
  `.cargo/config.toml` or in `RUSTFLAGS` without an explicit `--target`) breaks
  host proc-macro compilation (`serde_derive`). No `.cargo/config.toml` is
  shipped: the default `cargo build --release` (dynamic) is what the test
  harness and compile gate use, and the static binary is produced by the
  Makefile/RPM/DEB build targets with the explicit-target form. The resulting
  binary reports `statically linked` under `ldd` and has no `INTERP` segment.

## Specification ambiguities encountered

1. **Signature verification default vs. offline examples.** CONFIG defaults
   `signature-verification = on`, "plus the keyring path when on", and
   `load-desired-manifest` STEP 5 verifies "if signature verification is enabled
   in CONFIG". Yet the offline `verify`/`diff` EXAMPLES (`verify_clean`,
   `verify_offline_manifest_and_state`, `yaml_manifest_accepted`,
   `diff_offline_two_files`, etc.) supply no signing material and expect exit 0.
   **Conservative resolution:** verification is performed only when
   signature-verification is on **and** a keyring is configured; without a
   keyring there is nothing to verify against, so verification is a no-op (the
   CONFIG wording "plus the keyring path when on" supports treating a missing
   keyring as "verification not actually active"). A configured keyring with a
   missing/invalid detached signature (`<manifest>.sig`) is a manifest error
   (exit 1 on apply/diff/verify-with-manifest). This honours every EXAMPLE while
   keeping the verification path real when signing material is supplied.
2. **Bulk `rpm -qf` per-path correspondence.** `rpm -qf --qf '%{NAME}' p1 p2 …`
   interleaves owned-name (stdout) and not-owned (stderr) lines in argument
   order, and the correspondence is positional only when every path is owned.
   The implementation takes the positional mapping when the count of name lines
   equals the path count, else falls back to a bounded per-path query (still
   proportional to `/etc`, never a whole-system `rpm -Va`). This preserves the
   bulk-query performance property for the common case while staying correct
   when some `/etc` paths are unpackaged.
3. **`user`/`group` resolution without a passwd lookup.** The spec records
   `user`/`group` as non-empty strings. Without a name-service lookup in the
   exec/static model, the reader records the numeric uid/gid as a stable,
   non-empty value. This is sufficient for the drift comparison (which compares
   the recorded values) and never empty; a richer build may resolve names.
4. **Snapper userdata stamp in `write-applied-record`.** STEP 3 stamps the
   snapshot userdata. In the exec model this is owned by the transaction
   binding; the record write does not fail on a missing snapper, because the
   binding (not the record writer) owns the userdata mechanism. The applied
   record itself is always written as canonical JSON.

## Rules that could not be implemented exactly as written, and why

- **Live-host behaviours** (`apply` end-to-end convergence, full-scan integrity
  over `/usr` and `/boot`, real snapshot transactions, real package
  install/remove) require a privileged SUSE host with snapper/zypper/systemd and
  are not exercisable in this environment. They are implemented to drive the
  correct CLIs and follow the spec STEPS, and are covered by code review and the
  pure-logic unit tests, but cannot be black-box verified here. The full-scan
  walk (`read_full_scan`) is structurally present and returns `(None, None)`
  (omitting both observational scopes) when the named trees are absent, which is
  the correct clean-scan result; the recursive walk logic mirrors the verified
  `/etc` walk. These are recorded with Low/Medium confidence below.

## Parsing approach

Hand-written key=value + bare-word parsing (`src/cli/mod.rs::parse_args`), per
the decisions hints (no `clap`, to avoid a `--flag`-shaped dependency that would
fight the spec's CLI style). Options are accepted in **any** position (before or
after the verb). Enumerated option **values** are validated at parse time so a
bare `format=bad_value` (no verb) is an invocation error (exit 2), matching the
M0 acceptance criterion. `--version`/`--help`/`-h` are tolerated aliases handled
in `dispatch` regardless of position; bare invocation prints usage to stdout and
exits 0. JSON is parsed with `serde_json`; YAML with `serde_yaml 0.9.34` under a
safe profile (single document only, explicit-tag rejection via a `Value` walk,
anchor/alias rejection via a quote-aware source scan, typed deserialisation).

## Signal handling

`apply` must not leave a partially converged snapshot as the default boot
target. In this exec/transaction model, sealing and activation are the **final**
apply step (STEP 11); a transaction interrupted before that point is never
sealed and is discarded by the transaction binding, so the running system is
unchanged and no partial boot target is left. Rust's default SIGTERM/SIGINT
disposition terminates the process without partial output before any sealing
occurs, satisfying the template's SIGNAL-HANDLING (SIGTERM, SIGINT — clean exit,
no partial output) without an explicit handler. The approach is documented in
`src/main.rs`. (A future live-apply milestone may add an explicit handler that
proactively discards an open internal transaction; not required for the read
verbs or for the pre-seal discard contract.)

## Dependency versions

Direct dependencies (resolved by `cargo fetch`; lock file `Cargo.lock` written):

| Crate | Version | Purpose |
|-------|---------|---------|
| `serde` | 1 (features = ["derive"]) | data-model (de)serialisation |
| `serde_json` | 1 | canonical JSON |
| `serde_yaml` | 0.9 (resolved 0.9.34) | opt-in YAML; driven under the safe profile (Value-walk tag/alias rejection, single-document, typed deserialisation) |
| `sha2` | 0.10 | SHA256 for `desired_sha256` and file content digests |

The Rust milestones hints pin `serde`/`serde_json`; `serde_yaml` and `sha2` were
not pinned by a hints file but are widely-used, stable releases — versions are
resolved by Cargo, not fabricated. `serde_yaml 0.9.34` is the last published
release of that crate (marked deprecated upstream); it meets the safe-profile
requirements as driven here. **Flag for the maintainer:** consider migrating to
a maintained YAML crate (e.g. `serde_yml`) in a future revision; the safe-profile
guard logic (`src/manifest/serialize.rs`) is crate-agnostic and would port
directly. The libzypp/snapper/systemd bindings are **not** linked (exec route),
so no binding version strings are required.

## Compile gate result (template EXECUTION, Phase 6)

| Step | Command | Result |
|------|---------|--------|
| 1 — dependency resolution | `cargo fetch` | **pass** (Cargo.lock written) |
| 2 — compilation (debug) | `cargo build` | **pass**, zero warnings |
| 2b — static release build | `RUSTFLAGS='-C target-feature=+crt-static' cargo build --release --target x86_64-unknown-linux-gnu` | **pass**; `ldd` → `statically linked`, no INTERP segment |
| 3 — translator unit tests | `cargo test --lib` | **pass** — 21 passed, 0 failed |
| 3b — independent black-box suite | `cargo test --test cli` (binary at project root) | **pass** — 41 passed, 0 failed |
| 4 — test-author suite | (none present) | not applicable |

M0 milestone acceptance criteria (verified directly):

- `./zypper-declarative version | grep -q "^zypper-declarative "` → ✔ (prints `zypper-declarative 0.6.4 spec:<hash>`)
- `./zypper-declarative help | grep -q "usage:"` → ✔
- `./zypper-declarative --version | grep -q "^zypper-declarative "` → ✔ (tolerated alias, identical output)
- `./zypper-declarative format=bad_value; test $? -eq 2` → ✔ (exit 2)

## Test results — translator suite (`independent_tests/claude-opus-4-8/tests/cli.rs`)

All 41 black-box tests pass. Coverage map (test → EXAMPLE/INVARIANT):

| Test | Covers | Result |
|------|--------|--------|
| `bare_invocation_shows_help_exit_0` | EXAMPLE bare_invocation_shows_help | pass |
| `version_verb_bare_word` | EXAMPLE version_verb_bare_word + spec-hash embedding | pass |
| `version_flag_alias_matches_bare_word` | EXAMPLE version_flag_alias | pass |
| `help_verb_bare_word` | EXAMPLE help_verb_bare_word | pass |
| `help_flag_aliases` | INVARIANT --help/-h aliases | pass |
| `unknown_verb_rejected_exit_2` | EXAMPLE unknown_verb_rejected | pass |
| `unknown_format_value_exit_2` | M0 format=bad_value | pass |
| `describe_unknown_format_exit_2` | EXAMPLE describe_unknown_format | pass |
| `status_unknown_argument_exit_2` | EXAMPLE status_unknown_argument | pass |
| `status_no_declaration_exit_0` | EXAMPLE status_no_declaration | pass |
| `status_reports_generation` | EXAMPLE status_reports_generation | pass |
| `describe_out_extension_json` | EXAMPLE describe_out_extension_json | pass |
| `describe_out_extension_yaml` | EXAMPLE describe_out_extension_yaml | pass |
| `describe_format_overrides_extension` | EXAMPLE describe_format_overrides_extension | pass |
| `describe_output_unwritable_exit_2` | EXAMPLE describe_output_unwritable | pass |
| `describe_emits_json_with_format_version_1` | EXAMPLE describe_emits_manifest (structure) | pass |
| `describe_generator_carries_version` | INVARIANT meta.generator carries version | pass |
| `describe_attributes_object_never_null` | EXAMPLE scope_attributes_always_object | pass |
| `diff_prints_plan_install_and_delete` | EXAMPLE diff_prints_plan | pass |
| `diff_manifest_unreadable_exit_2` | EXAMPLE diff_manifest_unreadable | pass |
| `diff_offline_two_files_exit_0` | EXAMPLE diff_offline_two_files | pass |
| `diff_state_path_no_live_read_idempotent_bootstrap` | EXAMPLE describe_bootstraps_desired_manifest (offline) | pass |
| `diff_does_not_modify_system_no_transaction` | INVARIANT diff opens no transaction / no input modification | pass |
| `verify_offline_clean_exit_0` | EXAMPLE verify_clean / verify_offline_manifest_and_state | pass |
| `verify_offline_no_applied_record_ok` | EXAMPLE verify_offline_no_applied_record_ok | pass |
| `verify_detects_unit_drift_offline` | EXAMPLE verify_against_external_state_dump | pass |
| `verify_detects_file_drift_offline` | EXAMPLE verify_detects_drift | pass |
| `verify_type_transition_is_modified` | EXAMPLE drift_type_transition_is_modified | pass |
| `verify_no_applied_record_exit_2` | EXAMPLE verify_no_applied_record | pass |
| `verify_malformed_state_dump_exit_2` | EXAMPLE verify_malformed_state_dump | pass |
| `verify_state_path_extension_yaml_clean` | EXAMPLE verify_state_path_extension_yaml | pass |
| `apply_manifest_unreadable_exit_2` | EXAMPLE apply_manifest_unreadable | pass |
| `apply_manifest_invalid_format_version_exit_1` | EXAMPLE apply_manifest_invalid | pass |
| `apply_rejects_full_describe_dump_exit_1` | EXAMPLE apply_rejects_full_describe_dump | pass |
| `diff_with_invalid_manifest_exit_1` | ERROR: invalid manifest → exit 1 | pass |
| `yaml_manifest_accepted_diff_exit_0` | EXAMPLE yaml_manifest_accepted | pass |
| `yaml_unsafe_multidoc_rejected_exit_1` | EXAMPLE yaml_unsafe_rejected | pass |
| `yaml_format_identity_stable_hash` | EXAMPLE yaml_format_identity_stable (indirect) | pass |
| `scope_rejected_on_status` | INVARIANT scope only on describe/verify | pass |
| `scope_rejected_on_apply` | INVARIANT scope only on describe/verify | pass |
| `options_after_verb_accepted` | decisions hint: any-position options | pass |

## Test results — test-author suite

Not present (single-LLM run). No test-author cross-check suite was provided in
the input directory.

## Test Refinements

| Test | Result before | Action | Rationale |
|------|---------------|--------|-----------|
| `verify_offline_clean_exit_0` and the other `verify`/`diff` tests using `manifest-path` | failed | code fixed | The first implementation hard-failed `load-desired-manifest` when `signature-verification=on` but no keyring was configured, contradicting the spec EXAMPLES (`verify_clean`, `verify_offline_manifest_and_state`, `yaml_manifest_accepted`, `diff_offline_two_files`) which supply no signing material yet require exit 0. Per CONFIG ("on, plus the keyring path when on"), verification is now a no-op without a keyring; a configured keyring with a missing/invalid signature still fails. The tests were not changed. |
| (all other tests) | passed | none | — |

No test assertion, expected value, or fixture was edited after a run. The single
refinement above was a **code** fix justified by the spec EXAMPLES and CONFIG.

## Per-example confidence

Confidence: **High** = Tests-First `yes` and a named test in
`independent_tests/claude-opus-4-8/` passes without a live external service.
**Medium** = passes but requires a live service path partly untested.
**Low** = no test covers it (reasoning/review only).

| EXAMPLE | Confidence | Verification method | Unverified claims |
|---------|------------|---------------------|-------------------|
| apply_no_op_when_converged | Low | code review (`verb_apply` STEP 4) | requires a live host to confirm no transaction opened |
| apply_writes_and_deletes_etc_file | Low | code review (`converge_files`) | requires a live snapshot transaction |
| apply_absent_scope_unmanaged | Medium | `compute_intent_diff` unit logic + review | live convergence untested |
| apply_manifest_invalid | High | `apply_manifest_invalid_format_version_exit_1` | — |
| apply_manifest_unreadable | High | `apply_manifest_unreadable_exit_2` | — |
| apply_transaction_unavailable | Medium | code review (`acquire_transaction_context` external path → exit 2) | needs a host where no transaction is active to black-box; logic verified by review |
| apply_package_failure_rolls_back | Low | code review (`converge_packages` error → exit 1) | requires a live package manager |
| diff_prints_plan | High | `diff_prints_plan_install_and_delete` | — |
| diff_manifest_unreadable | High | `diff_manifest_unreadable_exit_2` | — |
| describe_emits_manifest | Medium | `describe_emits_json_with_format_version_1`, `describe_generator_carries_version` | live rpm/installed-nginx content not asserted (no live host) |
| describe_output_unwritable | High | `describe_output_unwritable_exit_2` | — |
| describe_bootstraps_desired_manifest | High | `diff_state_path_no_live_read_idempotent_bootstrap` (offline) | — |
| verify_clean | High | `verify_offline_clean_exit_0` | — |
| verify_against_external_state_dump | High | `verify_detects_unit_drift_offline` | — |
| verify_malformed_state_dump | High | `verify_malformed_state_dump_exit_2` | — |
| verify_detects_drift | High | `verify_detects_file_drift_offline` | — |
| verify_no_applied_record | High | `verify_no_applied_record_exit_2` | — |
| status_reports_generation | High | `status_reports_generation` | live drift line uses a live read (drift portion not asserted) |
| status_no_declaration | High | `status_no_declaration_exit_0` | — |
| status_unknown_argument | High | `status_unknown_argument_exit_2` | — |
| intent_diff_yields_deletion | High | unit test `diff::tests::intent_diff_yields_deletion` + `diff_prints_plan_install_and_delete` | — |
| drift_ignores_unmanaged_packaged_file | High | unit test `diff::tests::drift_ignores_unmanaged_packaged_file` | — |
| describe_actual_state_omits_pristine | Low | code review (`bulk_pristine` suppression) | requires a live rpmdb |
| describe_traverses_etc_subdirectories | Medium | `walk_etc` recursion logic + review | live `/etc` traversal not black-box asserted here |
| describe_records_symlink_verbatim | Medium | `walk_etc` `read_link` verbatim + review | offline synthetic-root black-box not included (constructible; see note) |
| describe_skips_special_file | Medium | `walk_etc` special-file skip + review | offline synthetic-root black-box not included |
| drift_type_transition_is_modified | High | `verify_type_transition_is_modified` + unit `diff::tests::drift_type_transition_is_modified` | — |
| describe_config_files_bounded_to_etc | Medium | `read_config_files` bounds to `<root>/etc` + review | live large-host scale not measured |
| describe_suppresses_package_pristine_etc_file | Low | code review (`bulk_ownership`+`bulk_pristine`) | requires a live rpmdb |
| describe_symlink_and_target_judged_independently | Low | code review (independent per-path judgement) | requires a live rpmdb |
| describe_pristine_distro_symlink_suppressed | Low | code review (symlink pristine-by-target) | requires a live rpmdb |
| scope_attributes_always_object | High | `describe_attributes_object_never_null` | — |
| describe_verify_differences_not_unreadable | Medium | `bulk_pristine` treats non-zero verify as normal + review | live verifier not exercised |
| verify_default_scope_ignores_usr | Medium | scope=etc default in `describe_actual_state` + review | live `/usr` not scanned by construction; not black-box asserted on a host |
| verify_scope_full_detects_unmanaged_addition | Low | code review (`read_full_scan` + drift integrity) | full scan returns empty in this env |
| verify_scope_full_detects_modified_package_file | Low | code review | full scan empty in this env |
| describe_scope_full_emits_observational_scopes | Low | code review | full scan empty in this env |
| describe_scope_full_boot_generated_files_unmanaged | Low | code review | full scan empty in this env |
| lock_is_fully_resolved_packages_scope | Medium | `converge_packages` rpmdb query + `is_valid_applied_record` | live package resolution untested |
| yaml_manifest_accepted | High | `yaml_manifest_accepted_diff_exit_0` | — |
| describe_format_yaml | High | `describe_out_extension_yaml` (YAML emission) | live nginx content not asserted |
| yaml_format_identity_stable | High | `yaml_format_identity_stable_hash` + unit `hash::tests` | — |
| yaml_unsafe_rejected | High | `yaml_unsafe_multidoc_rejected_exit_1` + unit `serialize::tests::rejects_yaml_anchor_alias` | — |
| describe_unknown_format | High | `describe_unknown_format_exit_2` | — |
| bare_invocation_shows_help | High | `bare_invocation_shows_help_exit_0` | — |
| version_verb_bare_word | High | `version_verb_bare_word` | — |
| version_flag_alias | High | `version_flag_alias_matches_bare_word` | — |
| help_verb_bare_word | High | `help_verb_bare_word` | — |
| unknown_verb_rejected | High | `unknown_verb_rejected_exit_2` | — |
| describe_out_extension_yaml | High | `describe_out_extension_yaml` | — |
| describe_out_extension_json | High | `describe_out_extension_json` | — |
| describe_format_overrides_extension | High | `describe_format_overrides_extension` | — |
| verify_state_path_extension_yaml | High | `verify_state_path_extension_yaml_clean` | — |
| describe_repositories_from_reposd | Medium | `read_repositories`/`parse_repo_ini` unit test + review | live repos.d not black-box asserted on a host |
| describe_unreadable_scope_strict | Low | code review (`handle_unreadable` error path) | requires an unreadable live source |
| describe_unreadable_scope_warn | Low | code review (`handle_unreadable` warn path) | requires an unreadable live source |
| describe_omits_genuinely_empty_scope | Medium | `read_*` return `None` for empty + review | — |
| diff_offline_two_files | High | `diff_offline_two_files_exit_0` | — |
| verify_offline_manifest_and_state | High | `verify_offline_clean_exit_0` | — |
| verify_offline_no_applied_record_ok | High | `verify_offline_no_applied_record_ok` | — |
| apply_rejects_full_describe_dump | High | `apply_rejects_full_describe_dump_exit_1` | — |
| idempotent_second_apply | Medium | `compute_intent_diff`/`compute_drift` emptiness logic + review | live second-apply untested |

## Public API Surface

The exported symbols of the implementation library (`zypper_declarative`),
grouped by module. The next translation at Version 0.6.4 must preserve these
(additions permitted; no removals/renames without a Version increment).

### module `meta`
- `pub const PROGRAM: &str`
- `pub const VERSION: &str`
- `pub const SPEC_SHA256: &str`
- `pub fn generator() -> String`

### module `error`
- `pub enum Severity { Error, Warning }`
- `pub enum Domain { Packages, Repositories, Services, Units, Files, Manifest, Transaction, Invocation }`
- `pub fn Domain::as_str(self) -> &'static str`
- `pub struct Diagnostic { severity: Severity, domain: Domain, message: String }`
- `pub fn Diagnostic::error(domain: Domain, message: impl Into<String>) -> Self`
- `pub fn Diagnostic::warning(domain: Domain, message: impl Into<String>) -> Self`
- `pub fn Diagnostic::line(&self) -> String`
- `pub enum ExitCode { Ok = 0, Logical = 1, Invocation = 2 }`
- `pub fn ExitCode::code(self) -> i32`
- `pub fn default_exit_for_domain(domain: Domain) -> ExitCode`

### module `manifest`
- `pub struct ScopeWrapper<T> { attributes: BTreeMap<String, serde_json::Value>, elements: Vec<T> }`
- `pub fn ScopeWrapper::<T>::new() -> Self`
- `pub fn ScopeWrapper::<T>::with_attribute(self, key: &str, value: serde_json::Value) -> Self`
- `pub struct ManifestMeta { format_version: i64, generator: String, created_at: String, desired_sha256: String }`
- `pub struct PackageRecord { name, version, release, arch: String }`
- `pub type PackagesScope = ScopeWrapper<PackageRecord>`
- `pub struct RepositoryRecord { alias, name, url, repo_type: String, enabled, gpgcheck, autorefresh: bool, priority: i64 }`
- `pub type RepositoriesScope = ScopeWrapper<RepositoryRecord>`
- `pub struct ServiceRecord { name: String, state: String }`
- `pub type ServicesScope = ScopeWrapper<ServiceRecord>`
- `pub struct ManagedFileRecord { name, file_type, mode, user, group, sha256, target, content_ref, package_name: String }`
- `pub type ConfigFilesScope = ScopeWrapper<ManagedFileRecord>`
- `pub struct ManagedBaselineRecord { …, changes: Vec<String> }`
- `pub type ChangedManagedFilesScope = ScopeWrapper<ManagedBaselineRecord>`
- `pub struct UnmanagedFileRecord { name, file_type, mode, user, group, sha256, target: String }`
- `pub type UnmanagedFilesScope = ScopeWrapper<UnmanagedFileRecord>`
- `pub struct Manifest { meta, packages, repositories, services, config_files, changed_managed_files, unmanaged_files }`
- `pub fn Manifest::empty() -> Self`
- `pub type AppliedRecord = Manifest`
- `pub enum TransactionMode { Auto, External, Internal }`
- `pub fn TransactionMode::parse(s: &str) -> Option<TransactionMode>`
- `pub struct TransactionContext { mode: TransactionMode, root: String, opened_here: bool }`
- `pub struct Diff { packages_install, packages_remove, repos_set, files_write, files_delete, units_change }`
- `pub fn Diff::is_empty(&self) -> bool`
- `pub struct DriftReport { files_modified, files_extra, units_divergent, packages_divergent, managed_files_modified, unmanaged_files_present }`
- `pub fn DriftReport::is_empty(&self) -> bool`
- `pub fn DriftReport::item_count(&self) -> usize`
- `pub enum ManifestFormat { Json, Yaml }`
- `pub fn ManifestFormat::parse(s: &str) -> Option<ManifestFormat>`

### module `manifest::format`
- `pub fn resolve_format(explicit: Option<ManifestFormat>, path: Option<&str>, default: ManifestFormat) -> ManifestFormat`

### module `manifest::hash`
- `pub fn desired_sha256(manifest: &Manifest) -> String`

### module `manifest::serialize`
- `pub fn parse_manifest(text: &str, format: ManifestFormat) -> Result<Manifest, Diagnostic>`
- `pub fn serialise_manifest(manifest: &Manifest, format: ManifestFormat) -> Result<String, Diagnostic>`
- `pub fn serialise_json(manifest: &Manifest) -> Result<String, Diagnostic>`

### module `config`
- `pub enum OnUnreadable { Error, Warn }`
- `pub fn OnUnreadable::parse(s: &str) -> Option<OnUnreadable>`
- `pub enum ScanScope { Etc, Full }`
- `pub fn ScanScope::parse(s: &str) -> Option<ScanScope>`
- `pub struct Config { transaction_mode, manifest_path, manifest_format, on_unreadable, scope, repo_lock, content_store, keep_list, signature_verification, keyring, activation_policy, applied_root }`

### module `load`
- `pub struct LoadedManifest { manifest: Manifest, desired_sha256: String }`
- `pub struct LoadError { diagnostic: Diagnostic }`
- `pub fn load_desired_manifest(manifest_path: &str, explicit_format: Option<ManifestFormat>, config: &Config) -> Result<LoadedManifest, LoadError>`

### module `record`
- `pub const APPLIED_REL: &str`
- `pub struct LoadedRecord { record: AppliedRecord, present: bool }`
- `pub fn applied_path(root: &str) -> PathBuf`
- `pub fn load_applied_record(root: &str) -> Result<LoadedRecord, Diagnostic>`
- `pub fn write_applied_record(ctx: &TransactionContext, desired: &Manifest, desired_sha256: &str, resolved: &PackagesScope, now_rfc3339: &str) -> Result<(), Diagnostic>`
- `pub fn is_valid_applied_record(record: &AppliedRecord) -> bool`
- `pub fn load_state_dump(path: &Path, format: ManifestFormat) -> Result<Manifest, Diagnostic>`

### module `diff`
- `pub fn compute_intent_diff(desired: &Manifest, applied: &AppliedRecord) -> Diff`
- `pub fn compute_drift(actual: &Manifest, reference: &AppliedRecord, keep_list: &HashSet<String>) -> DriftReport`

### module `state`
- `pub struct ActualState { manifest: Manifest, diagnostics: Vec<Diagnostic> }`
- `pub fn describe_actual_state(runner: &dyn CommandRunner, root: &str, on_unreadable: OnUnreadable, scope: ScanScope, keep_list: &HashSet<String>, now_rfc3339: &str) -> Result<ActualState, Diagnostic>`

### module `txn`
- `pub fn acquire_transaction_context(runner: &dyn CommandRunner, mode: TransactionMode) -> Result<TransactionContext, Diagnostic>`

### module `converge`
- `pub fn converge_packages(runner: &dyn CommandRunner, ctx: &TransactionContext, diff: &Diff, config: &Config) -> Result<PackagesScope, Diagnostic>`
- `pub fn converge_files(runner: &dyn CommandRunner, ctx: &TransactionContext, diff: &Diff, config: &Config, keep_list: &HashSet<String>) -> Result<(), Diagnostic>`
- `pub fn converge_units(runner: &dyn CommandRunner, ctx: &TransactionContext, diff: &Diff) -> Result<(), Diagnostic>`

### module `interfaces`
- `pub trait CommandRunner: Send + Sync { fn run(&self, cmd: &str, args: &[&str]) -> CommandResult }`
- `pub struct CommandResult { stdout: String, stderr: String, code: i32, spawn_failed: bool }`
- `pub struct OsCommandRunner` (impl `CommandRunner`)
- `pub struct FakeCommandRunner { responses: HashMap<String, CommandResult> }` (impl `CommandRunner`; in-tree test double)

### module `clock`
- `pub fn now_rfc3339() -> String`

### module `cli`
- `pub fn run(args: &[String]) -> i32`
- `pub fn dispatch(runner: &dyn CommandRunner, args: &[String]) -> i32`

## Template constraints compliance

| Constraint | Required | Compliance |
|------------|----------|------------|
| LANGUAGE | Go default; Rust supported | Rust (explicit run target); deviation documented above |
| BINARY-TYPE static | Go/Rust must be static | static via crt-static + explicit target; `ldd` → statically linked |
| SOURCE-PARTITIONING modular / one-entry-one-implementation | yes | entry point dispatch-only; 17 implementation modules |
| MODULE-IDENTITY host-specified / propagated / conflict-halts | yes | crate `zypper-declarative` from spec META `Module:`; propagated; no conflict |
| BINARY-COUNT 1 | yes | single `[[bin]]` |
| BINARY-LOCATION project-root (`../../<bin>`) | yes | binary at project root; tests invoke `../../zypper-declarative` |
| RUNTIME-DEPS none | yes | static binary; system tools driven via exec (delegated, documented) |
| CLI-ARG-STYLE key=value / bare-words | yes | hand-written parser; bare verbs; no POSIX `--flag` options (only tolerated version/help aliases) |
| EXIT-CODE-OK 0 / ERROR 1 / INVOCATION 2 | yes | `ExitCode` enum; mapping in verb layer |
| STREAM-DIAGNOSTICS stderr / STREAM-OUTPUT stdout | yes | diagnostics to stderr one per line; output to stdout |
| SIGNAL-HANDLING SIGTERM/SIGINT | yes | clean exit; no partial output / no partial boot target (documented) |
| OUTPUT-FORMAT RPM/DEB required | yes | `zypper-declarative.spec`, `debian/*` produced |
| OUTPUT-FORMAT OCI/PKG/binary supported | not active | not produced (no preset activation) |
| INSTALL-METHOD OBS / curl forbidden | yes | README + packaging document OBS; no curl |
| PLATFORM Linux | yes | Linux only |
| CONFIG-ENV-VARS forbidden | yes | environment is never read for behaviour |
| NETWORK-CALLS forbidden | yes (with documented delegation) | no direct network I/O; package retrieval delegated |
| FILE-MODIFICATION input-files forbidden | yes | inputs never modified (verified by `diff_does_not_modify_system_no_transaction`) |
| IDEMPOTENT true | yes | apply idempotence by design; pure diff/drift |
| PRESET-SYSTEM systemd-style | n/a in this run | preset layering not exercised; key=value options honoured |
| spec-hash embedded | yes | embedded in all source headers, `version` output, Makefile (`SPEC_SHA256`), RPM (`# pcd-spec-sha256:`), DEB (`X-PCD-Spec-SHA256:`), this report |

## Resume logic

The output directory contained only the pre-created empty `code/rs/` tree at
start; no prior deliverable existed, so all files were produced fresh. No file
was skipped as already-complete.
