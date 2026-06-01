# TRANSLATION_REPORT.md

**Spec-SHA256:** `f2cc80627e483a48bb8411d297711bc5f6c6e74c28dbf0dafc8fe7bd8817251e`
(merged spec text — host has no `Includes:`, so this equals the host hash)

**Spec-SHA256 (host):** `f2cc80627e483a48bb8411d297711bc5f6c6e74c28dbf0dafc8fe7bd8817251e`

**Included-Specs:** *(none)*

| Path | SHA256 |
|------|--------|
| *(none)* | — |

**LLM-Name:** `claude-opus-4-8`

**Mode:** `translator`

**Tests-First-Compliance:** `yes`. Every file under
`independent_tests/claude-opus-4-8/` (test harness, fixtures, and the five test
source files) was written and passed its `-fsyntax-only` gate **before** any
implementation source file in `src/` was written. The structural Tests-First
guard (translator flow step 3) was satisfied: the test directory existed and
contained test files before Phase 2 began.

**Continuity-Check:** not applicable — no test-author input. The input directory
contained no `independent_tests/<other-role-llm-name>/` directory and no
`TEST_REPORT.md`; this is a single-LLM run (a fully supported invocation).

**Deployment template:** `cli-tool.template.md` v0.3.29.

---

## Spec Composition (v0.4.0)

The host spec declares `Spec-Schema: 0.4.0` and **no** `Includes:` directives.
The merge described in the prompt resolves trivially: the merged spec equals the
host spec, the merged hash equals the host hash, and the Included-Specs table is
empty. This is the v0.3.x-compatible case. The `Spec-SHA256` embedded in every
artefact is the host hash above, computed with `sha256sum` before any output was
written.

## Active MILESTONE

All seven `## MILESTONE:` sections (0.0.0 scaffold, 0.1.0–0.6.0) carry
`Status: pending`. **No milestone has `Status: active`.** Per the prompt
("If no MILESTONE section is present, or no milestone has `Status: active`,
translate the full spec as normal"), this run translates the **full spec**: all
five verbs (`apply`, `diff`, `verify`, `status`, `describe`) and all eleven
`BEHAVIOR/INTERNAL` sections are implemented (subject to the live-host
deferrals documented below, which the spec and the C++ decisions hints
explicitly scope as on-target work). No milestone gating (`Included` /
`Deferred` BEHAVIORs) applies because none is active; the M0 and 0.1.0
acceptance criteria were nonetheless used as additional offline acceptance
checks and all pass (see Compile gate).

## Language resolution

- Target language: **C++** (C++17), as requested at invocation and as the
  language of the supplied hints files
  (`cli-tool.cpp.milestones.hints.md`, `zypper-declarative.cpp.decisions.hints.md`).
- The cli-tool template's default LANGUAGE is **Go**; C++ is a `supported`
  alternative (TEMPLATE-TABLE `LANGUAGE | C++ | supported` and
  `LANGUAGE-ALTERNATIVES | C++ | supported`). The deviation from the default is
  intentional and is selected by the run's stated target plus the C++ hints
  files in the preset hierarchy. This is recorded here per the "Universal
  principles" requirement to state language deviations explicitly.
- **BINARY-TYPE: dynamic** is used (permitted for C/C++/C# only). This is the
  deliberate C++ design: the tool links the distribution's supported shared
  libraries (libzypp, jsoncpp, yaml-cpp, libsnapper, OpenSSL libcrypto) rather
  than producing a static binary, per the decisions hints' no-vendoring /
  per-SP-OBS posture. PRECONDITION "If BINARY-TYPE is dynamic, LANGUAGE must be
  one of C, C++, C#" is satisfied.

## Module identity resolved

- Resolved identity: **`github.com/mge1512/zypper-declarative`**.
- Authoritative source: **source 1** — the spec META `Module:` field
  (`Module: github.com/mge1512/zypper-declarative`). No conflict; sources 2–4
  were not consulted. The spec-title fallback was **not** used.
- The identity is propagated (MODULE-IDENTITY: propagated) to: `CMakeLists.txt`
  (comment + project name), the RPM `.spec` (`URL:` and `Source:`), `debian/control`
  (`Homepage:` + `Source:`), `debian/copyright` (`Source:`), and `README.md`.

## Delivery mode

**Filesystem** (mode 1). All source and deliverable files were written directly
to `/tmp/pcd-output/`. The build, test, and install flows were executed in the
environment (SLES 15 SP7, GCC 15.2 via `g++-15`, CMake 4.3.2, dependencies
present). This is a single-LLM run.

## Source partitioning (SOURCE-PARTITIONING: modular, one-entry-one-implementation)

The implementation is partitioned into separate translation units, one
`{.hpp,.cpp}` pair per concern, following the C++ milestones-hints tree. The
entry-point (`src/main.cpp`) contains **only** signal wiring and a call into
`zd::run`; it implements no behaviour.

| Module | Responsibility |
|--------|----------------|
| `src/main.cpp` | Entry point: SIGTERM/SIGINT handlers, dispatch into `zd::run`. |
| `src/cli.{hpp,cpp}` | Dispatch, key=value parsing, global commands (version/help), usage, option/value validation. |
| `src/types.hpp` | The shared data model: scopes, Manifest, Diff, DriftReport, Diagnostic, enums. |
| `src/config.hpp` | Resolved CONFIG knobs and the option model. |
| `src/command_runner.{hpp,cpp}` | `CommandRunner` seam: `OSCommandRunner` (fork/execvp, separate streams, fixed PATH) + `FakeCommandRunner` test double. |
| `src/hashing.{hpp,cpp}` | SHA256 over a buffer / a file, via OpenSSL libcrypto. |
| `src/manifest.{hpp,cpp}` | `resolve-format`, JSON/YAML serialise+parse, schema validation, YAML safe profile, canonical-model hash, `load-desired-manifest`, `load-applied-record`, `load_state_dump`, `write_manifest`. |
| `src/diff.{hpp,cpp}` | `compute-intent-diff`, `compute-drift` (pure, no I/O), keep-list. |
| `src/describe.{hpp,cpp}` | `describe-actual-state` (the single live-state reader), the `SystemReader` seam. |
| `src/system_reader.{hpp,cpp}` | `ZyppSystemReader` (production `SystemReader` backed by libzypp). |
| `src/transaction.{hpp,cpp}` | `acquire-transaction-context`, `converge-packages/files/units`, `write-applied-record`. |
| `src/commands.{hpp,cpp}` | The five verbs; the only layer that maps results to a process exit code. |

The `by-behaviour-domain` `supported` partitioning is applied (parsing,
diff/drift, describe, transaction, and dispatch are separate modules).

## STEPS ordering per BEHAVIOR

Each verb and internal behaviour implements its spec STEPS in the written order:

- **apply** (`cmd_apply`): load desired → load applied → intent diff → (empty?
  describe+drift→"nothing to do") → acquire txn → repos+converge packages →
  converge files → converge units → write applied record → post-converge verify
  → seal/summary. Steps 1–11 in order.
- **diff** (`cmd_diff`): load desired → load applied → intent diff → obtain
  actual (state-path offline or describe) → print plan. Steps 1–5.
- **verify** (`cmd_verify`): determine reference (manifest-path else applied
  record; missing→exit 2) → obtain actual (scope, state-path) → compute drift →
  match→exit 0 else per-item diagnostics→exit 1. Steps 1–4.
- **status** (`cmd_status`): reject unknown args (done in `cli.cpp`) → load
  applied (none→message, exit 0) → print sha256/format_version/generation/
  created_at/package count → drift summary line. Steps 1–4.
- **describe** (`cmd_describe`): reject unknown args/format (in `cli.cpp`) →
  describe-actual-state with on_unreadable+scope → resolve-format(out) →
  serialise → write to out/stdout (write failure→exit 2). Steps 1–5.
- **describe-actual-state**: packages → repositories (from `<root>/etc/zypp/
  repos.d/*.repo`) → services → config_files (`/etc` walk) → 4a full-scan
  (scope=full) → assemble → unreadable-source handling. Steps 1–6.
- **resolve-format**: explicit wins → recognised extension → CONFIG default.
- **compute-intent-diff**, **compute-drift**, **converge-***,
  **write-applied-record**, **acquire-transaction-context**,
  **load-desired-manifest**, **load-applied-record** all follow their STEPS in
  order; see the per-function comments in the source.

MECHANISM: there are no explicit `MECHANISM:` annotations in this spec.

## Constraint: field on BEHAVIOR headers

Every BEHAVIOR and BEHAVIOR/INTERNAL section in the spec is `Constraint:
required`; all are implemented unconditionally. No `supported` or `forbidden`
behaviours appear, so no behaviour was skipped for a constraint reason.

## INTERFACES test doubles

The spec's `## INTERFACES` declares abstract external systems (package manager,
snapshot/filesystem, init system, transaction mechanism, external state
producer). These are expressed as seams:

- `CommandRunner` (abstract) with production `OSCommandRunner` and the test
  double `FakeCommandRunner`.
- `SystemReader` (abstract) with production `ZyppSystemReader`; a test double can
  stand in, and the offline path passes `nullptr` (synthetic-root mode), which
  exercises the filesystem walk without a live system. The external state
  producer interface is realised by `load_state_dump`, accepting any shared-schema
  dump via `state-path` (verified by the offline diff/verify tests).

The black-box `independent_tests/` suite uses **neither** the production nor the
double internals directly; it invokes the built binary as a subprocess, as the
deployment interface (CLI) mandates.

## TYPE-BINDINGS / GENERATED-FILE-BINDINGS

The cli-tool template has **no** `## TYPE-BINDINGS` and **no**
`## GENERATED-FILE-BINDINGS` section. Not applicable.

## COMPONENT → filename mapping

The spec's DEPLOYMENT does not use `COMPONENT:` entries; deliverable filenames
are taken from the template DELIVERABLES table (see "Deliverables produced"
below) with `<n>` = `zypper-declarative` (the spec title, lowercase,
hyphen-separated).

## Deliverables produced (per template DELIVERABLES, OUTPUT-FORMAT)

| OUTPUT-FORMAT | Constraint | Produced | Files |
|---|---|---|---|
| source | required | yes | `src/*.{hpp,cpp}` (entry-point + 11 impl modules), `CMakeLists.txt` (manifest) |
| public-api | required | yes | `## Public API Surface` below |
| build | required | yes | `Makefile` (executable `build`/`test`/`install`/`clean`/`man` targets) |
| docs | required | yes | `README.md` (OBS install for zypper/apt/dnf; usage; options; exit codes; no curl) |
| man | required | yes | `zypper-declarative.1.md`, `zypper-declarative.1` (pandoc) |
| license | required | yes | `LICENSE` (SPDX `GPL-2.0-or-later` + authoritative URL, no full text) |
| RPM | required | yes | `zypper-declarative.spec` |
| DEB | required | yes | `debian/control`, `debian/changelog`, `debian/rules`, `debian/copyright` (DEP-5) |
| OCI | supported | **no** | not active in resolved preset (no preset declares OCI); `Containerfile` not produced |
| PKG | supported | **no** | macOS not declared; not produced |
| binary | supported | **no** | no descriptor required; the raw binary is a build output, not committed |
| report | required | yes | this file |
| spec-hash | required | yes | embedded in all artefacts (see below) |

Auxiliary: `translation_report/translation-workflow.pikchr` (Phase 4).

The build directory (`build/`), the root binary (`zypper-declarative`), and the
compiled test runner are build **outputs** and were removed before finalising;
they are not committed (per "No unsolicited deliverables"). Empty `include/` and
`testdata/` scratch directories were removed for the same reason.

### Spec-hash embedding

`f2cc8062…7251e` is embedded in: every source/test file header comment;
`Makefile` `SPEC_SHA256`; `CMakeLists.txt` `ZD_SPEC_SHA256` (→ configured
`meta.hpp` → binary `version` output `spec:<hash>`); RPM `.spec`
`# pcd-spec-sha256:`; `debian/control` `X-PCD-Spec-SHA256:`; `debian/rules`
comment; `README.md`; and this report's header. (No `Containerfile` is
produced, so the `LABEL pcd.spec.sha256` consumer is not applicable.)

## Parsing approach

- **JSON:** jsoncpp (`Json::CharReaderBuilder` / `StreamWriterBuilder`). Found
  at 1.8.4 on this host (SLE 15 SP7 Development Tools Module); API stable across
  1.8–1.9.
- **YAML:** yaml-cpp (`find_package(yaml-cpp)`; 0.6.3 / `libyaml-cpp.so.0.6` on
  this host). Usage is restricted to the 0.6-stable surface (`LoadAll`, node
  walking, `YAML::Emitter`), avoiding 0.7+-only entry points so the same source
  compiles against 0.8 on SLE 16.
- **Canonical model hash:** `canonical_json` emits compact JSON with keys sorted
  and `_elements` sorted by identity (packages by name+arch, repositories by
  alias, services by name, config_files by path), and clears the
  non-identity meta fields (`generator`, `created_at`, `desired_sha256`). The
  SHA256 of this byte string is `desired_sha256`, so the **same intent expressed
  in JSON or YAML yields the same hash** (verified manually: a JSON and a YAML
  manifest with differing `generator`/`created_at` parse to the same model).
- **YAML safe profile** (each constraint and how it is enforced):
  - *non-code-executing loader / no arbitrary or executable tags:* every node is
    checked against an allow-list of core-schema tags (`yaml_tag_is_safe`); any
    non-default/application tag (e.g. `!!python/object/apply:…`) → manifest error.
  - *bounded/disabled alias expansion:* yaml-cpp resolves anchors/aliases into
    the node tree at load; the recursive `yaml_to_json` walk over the resolved
    tree terminates on the (finite) materialised structure, and an alias bomb is
    bounded by yaml-cpp's own depth handling. (Documented limitation: an explicit
    expansion **cap** is not separately imposed beyond yaml-cpp's behaviour; a
    hard numeric cap is a candidate hardening for the live-host milestone.)
  - *single document only:* `LoadAll(...).size() != 1` → manifest error (rejects
    multi-document streams).
  - *explicit typing per schema:* YAML scalars are carried as strings and typed
    explicitly by the schema reader (`as_int`/`as_bool` with explicit parsing),
    not by YAML implicit typing, so `NO`/`1.10`-style coercion does not occur.

## Signal handling approach

`main.cpp` installs `SIGTERM` and `SIGINT` handlers in `main()`. The handler is
async-signal-safe: it sets a `volatile sig_atomic_t` flag and calls `_exit(130)`
for a clean immediate exit with no partial output. The interruptible long
operation (an `apply` discarding its transaction mid-converge) is part of the
live-host converge milestone; in this version no partial snapshot can be left as
the default boot target because the internal/external transaction open is the
gate and it is reported (not silently half-done) when unavailable.

## Compile gate result (template EXECUTION)

Executed in full on this host.

- **Step 1 — dependency resolution:** C/C++ row is "None at this step"
  (dependencies are distro-provided and declared in the packaging artefacts).
  `cmake` configure located libzypp 17.37.18, jsoncpp 1.8.4, yaml-cpp 0.6.3,
  OpenSSL libcrypto 3.2.3. `libsnapper-devel` is **not installed** on this host;
  `CMakeLists.txt` makes it optional at configure time (linked when present so
  the per-SP RPM picks the right soname). **PASS** (with the libsnapper note).
- **Step 2 — compilation:** `make build` (→ `cmake --build build`) succeeds,
  **warning-clean** under `-Wall -Wextra` with `g++-15` (GCC 15.2). The Makefile
  selects `g++-15` automatically on SLE 15 (make pre-defines `CXX=g++`, so an
  `origin`-guarded assignment is used to pick `g++-15` when present). **PASS.**
- **Step 3 — translator test run:** `make test` builds the binary at the project
  root and runs the black-box suite: **38 tests, 0 failures.** **PASS.**
- **Step 4 — test-author run:** not applicable (single-LLM).
- **Step 5:** all steps pass; no further source modification.

`ldd` confirms the dynamic linkage: `libzypp.so.1735`, `libjsoncpp.so.19`,
`libyaml-cpp.so.0.6`, `libcrypto.so.3`. `file` confirms an ELF dynamically
linked executable (BINARY-TYPE: dynamic, as designed).

Additional offline acceptance checks (the M0 and 0.1.0 milestone criteria),
all **PASS**: bare-word `version`/`help`/`--version`; `format=bad_value` → exit
2; bare invocation → usage + exit 0; `version` contains `spec:`; `describe
out=…yaml` writes non-`{`-leading YAML by extension; `status` → "no declaration
applied"; `describe scope=full` emits observational scopes while `scope=etc`
does not.

## Live-host deferrals (documented per the C++ decisions/milestones hints)

The spec and the C++ hints scope real package, snapshot, transaction, and unit
work as on-target verification. The following are implemented behind the spec'd
interfaces but their effecting steps return a correctly-domained `Diagnostic`
(never a silent empty result) and are exercised on a live SUSE host:

- `acquire-transaction-context` **internal** open (needs the live
  transactional-update / zypper-merged stack and privilege) → `transaction`
  error; **external** detection via the `TRANSACTIONAL_UPDATE` marker is
  implemented and testable.
- `converge-packages` (libzypp commit) → `packages` error until on-target.
- `converge-units` (offline systemd enablement under ctx.root) → `units` error
  when units are requested; no-op when none requested.
- `ZyppSystemReader::query_services` returns an empty, **readable** scope (the
  live systemd enablement read is on-target work); it is never reported as an
  unreadable source.
- `ZyppSystemReader` package query, ownership (`findByFile`), and baseline
  comparison use **libzypp only** (`librpmDb::db_const_iterator`, `RpmHeader`);
  no librpm linkage and no `rpm`/`zypper`/`snapper` exec. The per-file baseline
  digest field rpm exposes is `FileInfo.md5sum` (rpm's configured algorithm,
  SHA256 on modern SUSE but not guaranteed for every package); an
  algorithm-aware comparison is flagged for the live-host milestone.
- `converge-files` (regular-file write/delete) **is** implemented; symlink
  convergence and type-transition handling are spec-reserved (`[reserved-0.7.0]`)
  for the apply-on-live-host milestone, as the spec states.

These deferrals do not affect the offline-testable behaviour the test suite
verifies (CLI contract, manifest load/validation/format, intent diff, drift,
describe `/etc` walk semantics, resolve-format).

## Dependency version verification

All four library bindings were resolved on this host and their detected versions
recorded above. They match the `[verified]` SLE 15 SP7 entries in the decisions
hints (libzypp 17.37.x, jsoncpp 1.8.4, yaml-cpp 0.6.3 / soname 0.6). libsnapper
devel was not installed here; on an OBS SLE 15 builder `libsnapper-devel`
(`libsnapper5`, 0.8.16) and on SLE 16 (`libsnapper7`, 0.12.1) apply — the build
links whichever is present, never assuming one soname. No dependency version was
fabricated.

## Specification ambiguities encountered

1. **`diff`/`verify` intent-diff baseline vs. `state-path`.** The intent diff
   (and thus `files_delete`) is computed from the **applied record** per
   `compute-intent-diff`, while `state-path` supplies only the **drift** actual
   state. Resolved conservatively per the spec (EXAMPLE `diff_prints_plan`:
   "relative to the applied record"); `applied-root` selects the applied-record
   generation. One test was corrected to match (see Test Refinements).
2. **Signature verification.** CONFIG `signature-verification` defaults to `on`,
   but the spec abstracts the keyring/mechanism. With no keyring configured the
   verification step is a documented no-op for offline parsing; real verification
   is a live-host concern. Conservative interpretation: parsing/validation always
   runs; signature failure paths are reachable only with a configured keyring.
3. **`status` "current snapshot/generation identifier".** Without libsnapper on
   this host the generation id is reported from the applied record's
   `created_at` (or `current`); the precise snapper generation id is a live-host
   detail.

## Rules not implemented exactly as written, and why

- The package/unit **convergence effecting steps** and the **snapshot open/seal/
  userdata stamp** are layered behind their spec interfaces but their effecting
  bodies are live-host work (return correctly-domained errors rather than
  pretending success). See "Live-host deferrals". This is consistent with the
  C++ decisions/milestones hints, which scope this as on-target verification and
  forbid faking an empty success.
- Everything else in the spec is implemented as written.

## Test results — translator suite (`independent_tests/claude-opus-4-8/`)

All **38 PASS, 0 fail, 0 skip**:

`test_version_bare_word`, `test_version_flag_alias_matches`,
`test_help_bare_word`, `test_help_flag_aliases`,
`test_bare_invocation_usage_stdout_exit0`, `test_unknown_verb_rejected`,
`test_status_unknown_argument`, `test_describe_unknown_format_rejected`,
`test_global_bad_format_value`, `test_unknown_option_value_mode`,
`test_unknown_option_key`, `test_apply_manifest_invalid_format_version`,
`test_apply_manifest_unreadable`, `test_diff_manifest_unreadable`,
`test_verify_malformed_state_dump`, `test_apply_rejects_observational_scope`,
`test_yaml_manifest_accepted_offline`, `test_yaml_unsafe_tag_rejected`,
`test_yaml_multidoc_rejected`, `test_diff_prints_install_and_delete`,
`test_diff_offline_two_files_exit0`, `test_verify_offline_match_exit0`,
`test_verify_offline_service_drift`, `test_verify_offline_file_drift`,
`test_verify_type_transition_is_modified`,
`test_verify_packaged_undeclared_not_extra`,
`test_verify_unpackaged_undeclared_is_extra`,
`test_describe_omits_empty_repositories_scope`, `test_describe_symlink_verbatim`,
`test_describe_traverses_subdirectory`, `test_describe_skips_special_file`,
`test_describe_out_extension_json`, `test_describe_out_extension_yaml`,
`test_describe_format_overrides_extension`, `test_describe_format_yaml_stdout`,
`test_describe_stdout_json_shape`, `test_describe_output_unwritable`,
`test_status_no_declaration`.

## Test results — test-author suite

Not present (single-LLM run).

## Test Refinements

| Test | Result before | Action | Rationale |
|------|---------------|--------|-----------|
| test_diff_prints_install_and_delete | failed | test edited | The test fed the old baseline via `state-path`, but `compute-intent-diff` STEP 3 computes `files_delete` = `(declared_old − declared_new)` from the **applied record**, not from a captured state dump (EXAMPLE `diff_prints_plan`/`intent_diff_yields_deletion`: "relative to the applied record"). The test now supplies the old declaration via `applied-root` pointing at a synthetic generation carrying `usr/lib/zypper-declarative/applied.json`, and an empty state dump for the drift portion. No assertion weakened; intent unchanged. |
| (all other 37 tests) | passed | none | — |

## Per-example confidence

Confidence: **High** = Tests-First `yes` AND a named test passes without a live
external service. **Medium** = passes but needs a live service / no direct test.
**Low** = reasoning/review only.

| EXAMPLE | Confidence | Verification method | Unverified claims |
|---|---|---|---|
| bare_invocation_shows_help | High | `test_bare_invocation_usage_stdout_exit0` | — |
| version_verb_bare_word | High | `test_version_bare_word` | — |
| version_flag_alias | High | `test_version_flag_alias_matches` | — |
| help_verb_bare_word | High | `test_help_bare_word`, `test_help_flag_aliases` | — |
| unknown_verb_rejected | High | `test_unknown_verb_rejected` | — |
| status_unknown_argument | High | `test_status_unknown_argument` | — |
| describe_unknown_format | High | `test_describe_unknown_format_rejected` | — |
| apply_manifest_invalid | High | `test_apply_manifest_invalid_format_version` | — |
| apply_manifest_unreadable | High | `test_apply_manifest_unreadable` | — |
| diff_manifest_unreadable | High | `test_diff_manifest_unreadable` | — |
| verify_malformed_state_dump | High | `test_verify_malformed_state_dump` | — |
| apply_rejects_full_describe_dump | High | `test_apply_rejects_observational_scope` | — |
| yaml_manifest_accepted | High | `test_yaml_manifest_accepted_offline` | — |
| yaml_unsafe_rejected | High | `test_yaml_unsafe_tag_rejected`, `test_yaml_multidoc_rejected` | alias-expansion cap relies on yaml-cpp default (no separate numeric cap) |
| diff_prints_plan | High | `test_diff_prints_install_and_delete` | — |
| intent_diff_yields_deletion | High | `test_diff_prints_install_and_delete` | — |
| diff_offline_two_files | High | `test_diff_offline_two_files_exit0` | — |
| verify_clean | High | `test_verify_offline_match_exit0` | live-system path is via describe (see verify_default_scope_ignores_usr) |
| verify_offline_manifest_and_state | High | `test_verify_offline_match_exit0` | — |
| verify_offline_no_applied_record_ok | High | `test_verify_offline_match_exit0` (asserts no "no declaration applied") | — |
| verify_against_external_state_dump | High | `test_verify_offline_service_drift` | — |
| verify_detects_drift | High | `test_verify_offline_file_drift` | — |
| verify_no_applied_record | Medium | reviewed in `cmd_verify` | needs live applied-record absence on "/"; covered logically, no dedicated live test |
| drift_type_transition_is_modified | High | `test_verify_type_transition_is_modified` | — |
| drift_ignores_unmanaged_packaged_file | High | `test_verify_packaged_undeclared_not_extra` | — |
| status_reports_generation | Medium | reviewed in `cmd_status` | needs a live applied record + drift read |
| status_no_declaration | High | `test_status_no_declaration` | — |
| describe_emits_manifest | High (shape) / Medium (live pkgs) | `test_describe_stdout_json_shape` | the live nginx/changed-file content needs a live host |
| describe_output_unwritable | High | `test_describe_output_unwritable` | — |
| describe_bootstraps_desired_manifest | Medium | reviewed (describe→diff round-trip) | needs a live unchanged system |
| describe_traverses_etc_subdirectories | High | `test_describe_traverses_subdirectory` | — |
| describe_records_symlink_verbatim | High | `test_describe_symlink_verbatim` | — |
| describe_skips_special_file | High | `test_describe_skips_special_file` | — |
| describe_config_files_bounded_to_etc | Medium | reviewed (`describe_actual_state` walks only `<root>/etc`) | bounded-cost claim not load-tested |
| describe_verify_differences_not_unreadable | Medium | reviewed (non-zero verifier exit is data, not unreadable) | needs a live verifier |
| describe_out_extension_yaml | High | `test_describe_out_extension_yaml` | — |
| describe_out_extension_json | High | `test_describe_out_extension_json` | — |
| describe_format_overrides_extension | High | `test_describe_format_overrides_extension` | — |
| describe_format_yaml | High | `test_describe_format_yaml_stdout` | — |
| describe_omits_genuinely_empty_scope | High | `test_describe_omits_empty_repositories_scope` | — |
| describe_repositories_from_reposd | Medium | reviewed (repos.d INI reader) + indirectly via empty-scope test | a populated-repos.d live test not added |
| describe_unreadable_scope_strict / _warn | Medium | reviewed in `describe_actual_state` unreadable handling | needs an unreadable source (root/permission) to test live |
| describe_scope_full_emits_observational_scopes | High | manual check on a synthetic full root (`/usr/bin/extra`) | not in the committed suite |
| describe_scope_full_boot_generated_files_unmanaged | Medium | reviewed (boot trees scanned, keep-list honoured) | needs a /boot fixture |
| verify_default_scope_ignores_usr | Medium | reviewed (`scope=etc` skips the full scan) | manual synthetic-root check, not committed |
| verify_scope_full_detects_unmanaged_addition | Medium | reviewed; manual synthetic-root check | not committed |
| verify_scope_full_detects_modified_package_file | Low/Medium | reviewed; needs a live packaged-file modification | baseline-digest algorithm caveat |
| lock_is_fully_resolved_packages_scope | Low | reviewed (`write_applied_record` sets resolved scope) | needs live `converge-packages` |
| yaml_format_identity_stable | Medium | manual check (JSON vs YAML same model parse) | hash-equality not asserted by a committed test |
| describe_unknown_format | High | `test_describe_unknown_format_rejected` | — |
| apply_no_op_when_converged | Low | reviewed (`cmd_apply` STEP 4 "nothing to do") | needs a live converged system |
| apply_writes_and_deletes_etc_file | Low | reviewed (`converge_files`) | needs a live transaction |
| apply_absent_scope_unmanaged | Medium | reviewed (`compute_intent_diff` absent-scope semantics; also covered by diff tests) | apply path needs a live transaction |
| apply_transaction_unavailable | Medium | reviewed (`acquire_transaction_context` external→transaction error, exit 2) | live-host external mode |
| apply_package_failure_rolls_back | Low | reviewed (converge failure → discard, exit 1) | needs a live transaction |
| idempotent_second_apply | Low | reviewed (intent diff + drift empty path) | needs a live applied system |

Unverified claims are listed explicitly in the rows above and are concentrated
on the live-host `apply`/converge/snapshot paths, consistent with the spec's and
the C++ hints' on-target deferral.

## Public API Surface

The exported surface is the C++ namespace `zd` symbols that other modules (and a
future test-double consumer) depend on. The black-box test suite does not link
these; it invokes the binary. The next translation must preserve these
signatures at spec Version 0.6.2.

### module: types.hpp
```
struct zd::ScopeWrapper<T> { std::map<std::string,std::string> attributes; bool has_attributes; std::vector<T> elements; };
struct zd::ManifestMeta { int format_version; std::string generator; std::string created_at; std::string desired_sha256; };
struct zd::PackageRecord { std::string name, version, release, arch; };
struct zd::RepositoryRecord { std::string alias, name, url, type; bool enabled, gpgcheck, autorefresh; int priority; };
struct zd::ServiceRecord { std::string name, state; };
struct zd::ManagedFileRecord { std::string name, type, mode, user, group, sha256, target, content_ref, package_name; };
struct zd::ManagedBaselineRecord { std::string name,type,mode,user,group,sha256,target,package_name; std::vector<std::string> changes; };
struct zd::UnmanagedFileRecord { std::string name,type,mode,user,group,sha256,target; };
struct zd::Manifest { ManifestMeta meta; std::optional<PackagesScope> packages; std::optional<RepositoriesScope> repositories; std::optional<ServicesScope> services; std::optional<ConfigFilesScope> config_files; std::optional<ChangedManagedFilesScope> changed_managed_files; std::optional<UnmanagedFilesScope> unmanaged_files; };
using zd::AppliedRecord = Manifest;
enum class zd::TransactionMode { Auto, External, Internal };
struct zd::TransactionContext { TransactionMode mode; std::string root; bool opened_here; };
struct zd::Diff { ...; bool empty() const; };
struct zd::DriftReport { ...; bool empty() const; size_t count() const; };
enum class zd::Severity { Error, Warning };
struct zd::Diagnostic { Severity severity; std::string domain; std::string message; };
enum class zd::ManifestFormat { Json, Yaml };
enum class zd::ScanScope { Etc, Full };
enum class zd::OnUnreadable { Error, Warn };
```

### module: config.hpp
```
struct zd::Config { ... };  // resolved CONFIG knobs + per-invocation describe options
struct zd::Option { std::string key, value; };
void zd::debug_log(const std::string& msg);
```

### module: command_runner.hpp
```
struct zd::CommandResult { std::string out, err; int code; };
class zd::CommandRunner { virtual CommandResult run(const std::string&, const std::vector<std::string>&) const = 0; };
class zd::OSCommandRunner : public CommandRunner { CommandResult run(const std::string&, const std::vector<std::string>&) const override; };
class zd::FakeCommandRunner : public CommandRunner { std::map<std::string,CommandResult> responses; ... };
```

### module: hashing.hpp
```
std::string zd::sha256_hex(const std::string& data);
std::string zd::sha256_file(const std::string& path, bool& ok);
```

### module: manifest.hpp
```
struct zd::LoadResult { bool ok; Manifest manifest; std::string desired_sha256; Diagnostic error; };
zd::ManifestFormat zd::resolve_format(const std::optional<ManifestFormat>&, const std::optional<std::string>&, ManifestFormat);
std::string zd::serialize_manifest(const Manifest&, ManifestFormat, bool pretty);
std::string zd::canonical_json(const Manifest&);
std::string zd::canonical_hash(const Manifest&);
zd::LoadResult zd::parse_manifest(const std::string& text, ManifestFormat, bool is_desired);
zd::LoadResult zd::load_desired_manifest(const std::string&, const std::optional<ManifestFormat>&, const Config&);
zd::LoadResult zd::load_state_dump(const std::string&, const std::optional<ManifestFormat>&, const Config&);
struct zd::AppliedResult { bool ok; bool present; AppliedRecord record; Diagnostic error; };
zd::AppliedResult zd::load_applied_record(const std::string& root);
bool zd::write_manifest(const Manifest&, ManifestFormat, const std::optional<std::string>&);
```

### module: diff.hpp
```
using zd::KeepList = std::set<std::string>;
zd::Diff zd::compute_intent_diff(const Manifest& desired, const AppliedRecord& applied);
zd::DriftReport zd::compute_drift(const Manifest& actual, const AppliedRecord& reference, const KeepList&);
zd::KeepList zd::load_keep_list(const std::string& path);
```

### module: describe.hpp
```
struct zd::ActualStateResult { bool ok; Manifest manifest; std::vector<Diagnostic> diagnostics; Diagnostic error; };
class zd::SystemReader { /* query_packages, query_services, owning_package, file_differs_from_baseline, link_differs_from_baseline */ };
zd::ActualStateResult zd::describe_actual_state(const std::string& root, OnUnreadable, ScanScope, const KeepList&, const SystemReader*);
```

### module: system_reader.hpp
```
class zd::ZyppSystemReader : public SystemReader { /* overrides all SystemReader methods */ };
```

### module: transaction.hpp
```
struct zd::TxnResult { bool ok; TransactionContext ctx; Diagnostic error; };
zd::TxnResult zd::acquire_transaction_context(TransactionMode, const CommandRunner&);
struct zd::ConvergeResult { bool ok; Diagnostic error; };
struct zd::PackagesConvergeResult { bool ok; PackagesScope resolved; Diagnostic error; };
zd::PackagesConvergeResult zd::converge_packages(const TransactionContext&, const Diff&, const CommandRunner&);
zd::ConvergeResult zd::converge_files(const TransactionContext&, const Diff&, const KeepList&, const std::string& content_store, const CommandRunner&);
zd::ConvergeResult zd::converge_units(const TransactionContext&, const Diff&, const CommandRunner&);
zd::ConvergeResult zd::write_applied_record(const TransactionContext&, const Manifest&, const std::string& desired_sha256, const PackagesScope&, const CommandRunner&);
```

### module: commands.hpp
```
int zd::cmd_apply(const Config&, const CommandRunner&, const SystemReader*);
int zd::cmd_diff(const Config&, const CommandRunner&, const SystemReader*);
int zd::cmd_verify(const Config&, const CommandRunner&, const SystemReader*);
int zd::cmd_status(const Config&, const CommandRunner&, const SystemReader*);
int zd::cmd_describe(const Config&, const CommandRunner&, const SystemReader*);
```

### module: cli.hpp
```
int zd::run(const std::vector<std::string>& args);
std::string zd::usage_text();
```

## Template constraints compliance

| Constraint | Resolution |
|---|---|
| LANGUAGE | C++ (supported alternative; default Go overridden, documented) |
| BINARY-TYPE | dynamic (permitted for C++; distro shared libs, no static, no vendoring) |
| SOURCE-PARTITIONING modular / one-entry-one-implementation | satisfied (12 modules; entry-point dispatch only) |
| MODULE-IDENTITY host-specified / propagated | `github.com/mge1512/zypper-declarative` from spec META; propagated to all artefacts |
| PUBLIC-API-SURFACE recorded-in-report | this section |
| BINARY-COUNT 1 | one binary `zypper-declarative` |
| BINARY-LOCATION project-root | binary at project root; tests use `../../zypper-declarative` |
| CLI-ARG-STYLE key=value (+ bare-words) | enforced; POSIX `--flag` only as version/help aliases |
| EXIT-CODE-OK/ERROR/INVOCATION 0/1/2 | mapped in the verb layer |
| STREAM-DIAGNOSTICS stderr / STREAM-OUTPUT stdout | diagnostics→stderr, output→stdout |
| SIGNAL-HANDLING SIGTERM/SIGINT | clean exit handlers in main |
| OUTPUT-FORMAT RPM, DEB required | produced; OCI/PKG/binary not active → not produced |
| INSTALL-METHOD OBS (curl forbidden) | README documents OBS install only; no curl |
| PLATFORM Linux | Linux only |
| CONFIG-ENV-VARS forbidden | no behaviour via env vars (only `ZYPPER_DECLARATIVE_DEBUG` trace gate) |
| NETWORK-CALLS forbidden | no direct network I/O; package retrieval delegated to libzypp (documented spec deviation) |
| FILE-MODIFICATION input-files forbidden | input manifests never modified |
| IDEMPOTENT true | apply idempotence per spec (intent diff + drift empty → no-op) |
| spec-hash embedded | embedded in all artefacts |

Written last, after all other deliverables were produced and the compile gate
(build + 38/38 tests + install) passed.
