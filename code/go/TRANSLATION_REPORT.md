# TRANSLATION_REPORT.md — zypper-declarative (Go)

## Header

- **Spec-SHA256:** `51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03` (merged = host; no `Includes:`)
- **Spec-SHA256 (host):** `51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03`
- **Included-Specs:**

  | Path | SHA256 |
  |------|--------|
  | _(none)_ | _(none)_ |

  The host spec declares `Spec-Schema: 0.4.0` but contains **no** `Includes:`
  directives in its META, so the merged-spec hash equals the host hash and the
  inclusions table is empty (the v0.3.x-compatible case).

- **LLM-Name:** `claude-opus-4-8`
- **Mode:** `translator`
- **Tests-First-Compliance:** `yes` — every file in
  `independent_tests/claude-opus-4-8/` was written and present on disk before any
  implementation source file. The Tests-First structural guard (step 3) was
  checked: the test directory existed and contained a `*_test.go` file before
  Phase 2 began.
- **Continuity-Check:** not applicable — no test-author input. The input directory
  contained no `independent_tests/<other-role-llm-name>/` and no `TEST_REPORT.md`,
  so this is a single-LLM run.

### Translation Inputs (provenance)

- `Spec-SHA256:` `51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03`
- `Decisions-Hints-SHA256:` `zypper-declarative.go.decisions.hints.md` `c31330c25e7b90ad520fe6103a255e1b0dd55e620a1c18bb7d3cdf97aae4efa5`
- `Milestones-Hints-SHA256:` `cli-tool.go.milestones.hints.md` `c210c2f1404a72fcdf9bfeea353ea3c68f4ef70dabae39293887b32d75f89f52`
- `Template-SHA256:` `cli-tool.template.md` `c8447ba8f1e63f3605b8e671e5bf58f4df44665a5ba1ff76864d28e4570042b5`
- `Style-Hints-SHA256:` `none` (no `<scope>.<language>.style.hints.md` present in input or preset hierarchy)
- `Library-Hints-SHA256:` `none`

---

## Target language and preset

- **Resolved LANGUAGE:** Go — the cli-tool template default. No preset files were
  present (`/usr/share/pcd/presets/`, `/etc/pcd/presets/`, `~/.config/pcd/presets/`,
  `<project-dir>/.pcd/`), and the spec does not (and may not) declare LANGUAGE in
  META. No override applied; the default was used.
- **BINARY-TYPE:** `static` (the only valid value for Go; `CGO_ENABLED=0` set in
  every build path — Makefile, RPM `%build`, `debian/rules`).

## Module identity resolved

- **Resolved module identity:** `github.com/mge1512/zypper-declarative`.
- **Authoritative source:** **source 1** — the spec META `Module:` field
  (`Module:      github.com/mge1512/zypper-declarative`). The Go decisions hints
  file also names the same value (`[spec]` go.mod module line is exactly
  `github.com/mge1512/zypper-declarative`), so sources 1 and 2 **agree**; no
  conflict, no spec-title fallback. The identity propagates to `go.mod`, all
  internal import paths, `debian/rules` (`DH_GOPKG`), the RPM/DEB `URL`/`Homepage`,
  and the README.

## Delivery mode

Filesystem (mode 1). All files written under `/tmp/pcd-output/code/go/`.
Dependencies vendored with `go mod vendor` (no system packages installed; module
downloads ran as the current user with `GOPATH`/`GOCACHE` under `$HOME`).

## Active MILESTONE

All seven `## MILESTONE:` sections carry `Status: pending`; **none is active**.
Per the universal prompt ("If no MILESTONE section is present, or no milestone has
`Status: active`, translate the full spec as normal"), the **full spec** was
translated. All BEHAVIORs were implemented, not a milestone subset. The M0
acceptance criteria were nonetheless used as a smoke gate and all pass (see Compile
gate). No BEHAVIOR is "not yet scheduled": every BEHAVIOR in the spec has an
implementation.

## STEPS ordering per BEHAVIOR

- **apply** — STEPS 1–11 implemented in order in `internal/cli/verbs.go::runApply`:
  load desired (→ exit 2 on read, exit 1 on schema/unsafe), load applied,
  compute-intent-diff, empty-diff → live describe + compute-drift → "nothing to
  do"/exit 0, acquire-transaction-context, then converge packages/files/units,
  write-applied-record, post-converge verify, seal/activate. Convergence and
  transaction sealing require a live snapshot mechanism and privilege; in an
  environment with no transaction mechanism, `acquire-transaction-context` (and
  thus apply) surfaces a `domain=transaction` error rather than fabricating a
  snapshot (see Ambiguities/Deviations).
- **diff** — STEPS 1–5 in `runDiff`: load desired, load applied, compute-intent-diff,
  obtain actual (supplied `state-path` offline, else live `scope=etc`; malformed
  dump → exit 2), compute-drift, print plan, exit 0.
- **verify** — STEPS 1–4 in `runVerify`: determine reference (`manifest-path` else
  applied record; none → "no declaration applied" exit 2), obtain actual
  (`state-path` offline else live with `scope`), compute-drift, report (exit 0 with
  "system matches declaration" else one diagnostic per item, exit 1).
- **status** — STEPS 1–4 in `runStatus`: reject unrecognised argument (exit 2 in
  the option parser; `scope` is rejected on status), load applied (none → "no
  declaration applied" exit 0), print hash/format_version/generation/created_at/
  package count, live drift summary line ("clean"/"N drift item(s)").
- **describe** — STEPS 1–5 in `runDescribe`: reject unknown argument/format (exit 2),
  describe-actual-state with `on_unreadable`/`scope`, resolve-format(out),
  serialise (JSON/YAML), write to `out` or stdout (unwritable → exit 2), exit 0.
- **describe-actual-state** — STEPS 1–6 in `internal/state`: packages (rpm -qa),
  repositories (read `<root>/etc/zypp/repos.d/*.repo` directly), services
  (systemctl list-unit-files, declarable states only), config_files (rpm -V
  verdict-parse + ghost pass + unpackaged walk + content store), full-scan
  integrity under `scope=full`, assemble manifest omitting genuinely-empty scopes,
  unreadable-source handling per `on_unreadable`.
- **resolve-format** — STEPS 1–3 in `internal/manifest/format.go`: explicit wins,
  else extension (`.json`/`.yaml`/`.yml`), else default.
- **load-desired-manifest** — STEPS 1–6 in `internal/manifest/load.go`: read,
  resolve-format, parse (safe YAML profile), schema-validate + reject non-empty
  observational scopes, signature verification (when enabled), canonical hash.
- **load-applied-record / compute-intent-diff / compute-drift /
  acquire-transaction-context / converge-packages / converge-files /
  converge-units / write-applied-record** — each implemented STEP-by-STEP in
  `internal/record`, `internal/diff`, `internal/txn`, and `internal/converge`.

## INTERFACES test doubles produced

The spec's INTERFACES section lists external systems (package manager, snapshot/
filesystem, init system, transaction mechanism, optional external state producer).
The implementation abstracts command execution behind the `CommandRunner`
interface (`internal/syscmd`), with the production `OSCommandRunner` and a
`FakeCommandRunner` test double. The transaction binding is abstracted behind the
`EnvProbe` interface (`internal/txn`) with the production `liveProbe`
(`internal/cli`). The independent tests are **black-box** (they drive the built
binary via `exec.Command` and never import these packages), per the test
methodology; the test doubles are provided for in-tree unit testing as the
INTERFACES requirement asks ("production and all test doubles").

## TYPE-BINDINGS / GENERATED-FILE-BINDINGS

The cli-tool template contains no `## TYPE-BINDINGS` or
`## GENERATED-FILE-BINDINGS` section. Not applicable. The spec's logical types map
to Go structs in `internal/manifest/types.go` with explicit `json:` tags using the
spec's `underscore_style` keys; `ScopeWrapper<T>` maps to the generic
`ScopeWrapper[T any]` (Go 1.21+) with `_attributes`/`_elements` tags.

## BEHAVIOR Constraint handling

Every BEHAVIOR and BEHAVIOR/INTERNAL in the spec carries `Constraint: required`.
There were **no** `supported` or `forbidden` BEHAVIORs, so all were implemented
unconditionally and none was suppressed.

## COMPONENT → filename mapping

The spec has no DELIVERABLES `COMPONENT:` entries. Filenames follow the template's
per-language source layout: entry point `cmd/zypper-declarative/main.go`,
implementation under `internal/<concern>/`, manifest `go.mod`. The component name
`<n>` = `zypper-declarative` (spec title, lowercase-hyphenated) drives
`zypper-declarative.spec`, `zypper-declarative.1.md`, etc.

## Parsing approach

- **Argument parsing:** hand-written `key=value` parser in `internal/cli/cli.go`.
  Bare-word verbs (`apply`/`diff`/`verify`/`status`/`describe`) and the global
  commands `version`/`help` are dispatched first; options are `key=value` only,
  validated against a fixed key set, with value validation for `format`,
  `on-unreadable`, `scope`, `mode`, `signature-verification`. POSIX `--flag` style
  is used only for the tolerated global aliases `--version`/`--help`/`-h`. An
  unknown verb/option/value or a non-`key=value` token → usage to stderr, exit 2.
- **Manifest parsing:** one typed data model (`internal/manifest`). JSON via
  `encoding/json`. YAML via `gopkg.in/yaml.v3` under a **safe profile**: the
  document is parsed into a `yaml.Node` tree, which is walked to reject any
  non-core (executable/arbitrary) tag, any anchor, and any alias; multi-document
  streams are rejected by attempting a second decode; the safe node tree is then
  converted to a generic value and re-marshalled to JSON for strict decoding. A
  YAML input requiring any disabled feature returns a `domain=manifest` error
  rather than being parsed. `desired_sha256` is the SHA256 of a canonical,
  recursively key-sorted, element-sorted JSON serialisation of the parsed model
  with meta neutralised, so JSON and YAML expressions of the same manifest hash
  equal.

## Signal handling approach

`internal/cli/cli.go::installSignalHandlers` registers a handler for `SIGTERM` and
`SIGINT` via `os/signal` + `syscall`; on receipt it exits cleanly with code 0
(`ExitOK`) and emits no partial output. The read-only verbs hold no transaction;
for `apply`, an interrupt before the transaction is sealed leaves no new snapshot
as the default boot target (the transaction binding seals only on success, and an
unsealed snapshot is never marked default). Documented per the DEPLOYMENT
Signal-handling requirement.

## Dependency versions

- `gopkg.in/yaml.v3 v3.0.1` — the YAML library for the opt-in YAML serialisation.
  No language-specific hints file pinned a YAML library version, so a current
  stable release (`v3.0.1`) was selected and is **flagged here for manual version
  verification** per the spec DEPENDENCIES section. It is driven only under the
  safe profile described above. `go.sum` was produced by the Go resolver
  (`go mod tidy`); dependencies are vendored under `vendor/`.
- libzypp / snapper-btrfs / systemd bindings: the implementation drives `zypper`,
  `snapper`, `systemctl`, `rpm`, and `update-alternatives` via `os/exec` (per the
  Go decisions hints, to keep `CGO_ENABLED=0` and a single static binary) rather
  than linking native libraries, so **no native binding versions are required**.
  These tools are runtime dependencies of the host, not build/link dependencies.

## Template constraints compliance

| Constraint | Required | Status |
|---|---|---|
| LANGUAGE | Go (default) | Go ✔ |
| BINARY-TYPE | static (Go) | `CGO_ENABLED=0`, `file` reports "statically linked" ✔ |
| SOURCE-PARTITIONING modular | yes | entry point + 9 internal packages ✔ |
| SOURCE-PARTITIONING one-entry-one-implementation | yes | `cmd/.../main.go` is CLI dispatch only, calls `internal/cli` ✔ |
| MODULE-IDENTITY host-specified | yes | from spec META `Module:` (source 1) ✔ |
| MODULE-IDENTITY propagated | yes | go.mod, imports, debian/rules, RPM/DEB URL ✔ |
| PUBLIC-API-SURFACE recorded-in-report | yes | see `## Public API Surface` below ✔ |
| BINARY-COUNT 1 | yes | one binary `zypper-declarative` ✔ |
| BINARY-LOCATION project-root | yes | `make build` → `./zypper-declarative`; tests invoke `../../zypper-declarative` ✔ |
| RUNTIME-DEPS none | yes | single static binary; drives host tools via exec at run time ✔ |
| CLI-ARG-STYLE key=value | yes | key=value options; bare-word verbs ✔ |
| EXIT-CODE-OK/ERROR/INVOCATION 0/1/2 | yes | mapped in `internal/cli` ✔ |
| STREAM-DIAGNOSTICS stderr | yes | diagnostics on stderr ✔ |
| STREAM-OUTPUT stdout | yes | summaries/plan/report/document on stdout ✔ |
| SIGNAL-HANDLING SIGTERM/SIGINT | yes | clean exit 0, no partial output ✔ |
| OUTPUT-FORMAT RPM (required) | yes | `zypper-declarative.spec` ✔ |
| OUTPUT-FORMAT DEB (required) | yes | `debian/control`,`changelog`,`rules`,`copyright` ✔ |
| OUTPUT-FORMAT OCI/PKG/binary (supported) | only if active | not active in any resolved preset → not produced (see Deviations) |
| INSTALL-METHOD OBS | yes | README documents OBS; no curl ✔ |
| PLATFORM Linux | yes | Linux only ✔ |
| CONFIG-ENV-VARS forbidden | yes | no env-var control; key=value/preset only ✔ |
| NETWORK-CALLS forbidden | yes (with documented deviation) | no direct network I/O; package fetch delegated to zypper (see Deviations) |
| FILE-MODIFICATION input-files forbidden | yes | input manifest never modified ✔ |
| IDEMPOTENT true | yes | sorted output + canonical hash; second apply computes empty diff/drift ✔ |
| spec-hash embedded | yes | source headers, version output, RPM comment, DEB control field, Makefile var ✔ |

## Deliverables produced

| Deliverable | File(s) | Status |
|---|---|---|
| source | `cmd/zypper-declarative/main.go`, `internal/{meta,manifest,record,diff,txn,converge,syscmd,state,cli}/*.go`, `go.mod`, `go.sum` | ✔ |
| build | `Makefile` (build/test/install/clean/man; `test` is executable) | ✔ |
| docs | `README.md` (OBS install, usage, options, exit codes; no curl) | ✔ |
| man | `zypper-declarative.1.md` (+ `zypper-declarative.1` generated by `make man`/pandoc) | ✔ |
| license | `LICENSE` (SPDX `GPL-2.0-or-later` + authoritative URL) | ✔ |
| RPM | `zypper-declarative.spec` | ✔ |
| DEB | `debian/control`, `debian/changelog`, `debian/rules`, `debian/copyright` (DEP-5) | ✔ |
| auxiliary | `translation_report/translation-workflow.pikchr` | ✔ |
| report | `TRANSLATION_REPORT.md` | ✔ (this file) |
| spec-hash | embedded in all artefacts | ✔ |

The man page troff (`zypper-declarative.1`) and the binary are **build outputs**,
not source artefacts, and were verified to generate/build then removed from the
tree (they are produced by `make man` / `make build`). OCI `Containerfile`, PKG
`<n>.pkgbuild`, and raw `binary` are `supported` OUTPUT-FORMATs not active in any
resolved preset (no preset files present, macOS not declared), so per the
template's DELIVERABLES rule they were **not** produced.

## Compile gate result

Phase 6 executed in full (environment has the Go 1.26 toolchain):

- **Step 1 — dependency resolution:** `go mod tidy` produced `go.sum`;
  `go mod vendor` populated `vendor/`. Pass.
- **Step 2 — compilation:** `go build ./...` → exit 0; `go vet ./...` → exit 0;
  `gofmt -l .` (excluding vendor) → empty (clean). `make build` produced a
  statically linked binary (`file` reports "statically linked"). Pass.
- **Step 3 — translator test run:** `make test` → `ok ... 32 tests, all PASS`.
- **M0 acceptance smoke:** all four acceptance criteria pass (`version` prefix,
  `help` usage, `--version` alias, `format=bad_value` exit 2), plus the
  version-output spec-hash check.

## Test results — translator suite (`independent_tests/claude-opus-4-8/`)

All 32 tests PASS, no skips, no failures (run without any live external service):

| Test | Result |
|---|---|
| TestBareInvocationShowsHelp | PASS |
| TestVersionVerbBareWord | PASS |
| TestVersionFlagAlias | PASS |
| TestHelpVerbBareWord | PASS |
| TestHelpFlagAliases | PASS |
| TestUnknownVerbRejected | PASS |
| TestDescribeUnknownFormat | PASS |
| TestUnknownFormatValueIsInvocationError | PASS |
| TestStatusUnknownArgument | PASS |
| TestApplyManifestUnreadable | PASS |
| TestApplyManifestInvalid | PASS |
| TestApplyRejectsFullDescribeDump | PASS |
| TestApplyYamlUnsafeRejected | PASS |
| TestDiffManifestUnreadable | PASS |
| TestDiffOfflineTwoFiles | PASS |
| TestDiffPrintsPlanInstall | PASS |
| TestDiffMalformedStateDump | PASS |
| TestDiffYamlManifestAccepted | PASS |
| TestVerifyOfflineMatches | PASS |
| TestVerifyOfflineUnitDrift | PASS |
| TestVerifyOfflineFileDrift | PASS |
| TestVerifyTypeTransitionIsModified | PASS |
| TestVerifyMalformedStateDump | PASS |
| TestVerifyStatePathExtensionYaml | PASS |
| TestVerifyNoAppliedRecord | PASS |
| TestStatusNoDeclaration | PASS |
| TestDescribeOutExtensionYaml | PASS |
| TestDescribeOutExtensionJson | PASS |
| TestDescribeFormatOverridesExtension | PASS |
| TestDescribeOutputUnwritable | PASS |
| TestScopeRejectedOnStatus | PASS |
| TestScopeAttributesNeverNull | PASS |

## Test results — test-author suite

Not present (single-LLM run). No `independent_tests/<other-role-llm-name>/` in the
input directory.

## Test Refinements

| Test | Result before | Action | Rationale |
|---|---|---|---|
| _(all)_ | passed | none | Every test passed on its first run against the implementation; no test was edited and no implementation fix was required post-run. |

## Per-EXAMPLE confidence

Confidence is **High** only where a named test in
`independent_tests/claude-opus-4-8/` passes without a live external service
(Tests-First-Compliance is `yes`). EXAMPLEs that inherently require a live
privileged system (real rpmdb verdicts, snapshot transactions, systemd offline
enablement, `/usr`+`/boot` integrity scan) cannot be black-box-verified during
translation on a non-root build host and are **Medium** (logic implemented and
reviewed; not exercised end-to-end here).

| EXAMPLE | Confidence | Verification method | Unverified claims |
|---|---|---|---|
| bare_invocation_shows_help | High | TestBareInvocationShowsHelp | — |
| version_verb_bare_word | High | TestVersionVerbBareWord | — |
| version_flag_alias | High | TestVersionFlagAlias | — |
| help_verb_bare_word | High | TestHelpVerbBareWord | — |
| unknown_verb_rejected | High | TestUnknownVerbRejected | — |
| describe_unknown_format | High | TestDescribeUnknownFormat | — |
| status_unknown_argument | High | TestStatusUnknownArgument | — |
| apply_manifest_unreadable | High | TestApplyManifestUnreadable | — |
| apply_manifest_invalid | High | TestApplyManifestInvalid | — |
| apply_rejects_full_describe_dump | High | TestApplyRejectsFullDescribeDump | — |
| yaml_unsafe_rejected | High | TestApplyYamlUnsafeRejected | — |
| diff_manifest_unreadable | High | TestDiffManifestUnreadable | — |
| diff_offline_two_files | High | TestDiffOfflineTwoFiles | — |
| diff_prints_plan | High | TestDiffPrintsPlanInstall | full live-read path of diff is Medium (no state-path) |
| yaml_manifest_accepted | High | TestDiffYamlManifestAccepted | — |
| verify_offline_manifest_and_state | High | TestVerifyOfflineMatches | — |
| verify_offline_no_applied_record_ok | High | TestVerifyOfflineMatches / TestVerifyNoAppliedRecord | — |
| verify_against_external_state_dump | High | TestVerifyOfflineUnitDrift | — |
| verify_detects_drift | High | TestVerifyOfflineFileDrift | live-read variant is Medium |
| drift_type_transition_is_modified | High | TestVerifyTypeTransitionIsModified | — |
| verify_malformed_state_dump | High | TestVerifyMalformedStateDump | — |
| verify_state_path_extension_yaml | High | TestVerifyStatePathExtensionYaml | — |
| verify_no_applied_record | High | TestVerifyNoAppliedRecord | — |
| status_no_declaration | High | TestStatusNoDeclaration | — |
| status_reports_generation | Medium | code review (requires an applied.json + live drift read) | live drift line untested here |
| status_unknown_argument | High | TestStatusUnknownArgument | — |
| describe_out_extension_yaml | High | TestDescribeOutExtensionYaml | — |
| describe_out_extension_json | High | TestDescribeOutExtensionJson | — |
| describe_format_overrides_extension | High | TestDescribeFormatOverridesExtension | — |
| describe_output_unwritable | High | TestDescribeOutputUnwritable | — |
| scope_attributes_always_object | High | TestScopeAttributesNeverNull + manual describe of a synthetic root | — |
| describe_traverses_etc_subdirectories | Medium | manual describe of a synthetic root (subdir + symlink + file emitted correctly) | not a CI test; no rpm ownership in synthetic root |
| describe_records_symlink_verbatim | Medium | manual describe of a synthetic root (target stored verbatim, sha256 "") | not a CI test |
| describe_skips_special_file | Medium | code review (`!info.Mode().IsRegular()` skip) | not exercised |
| intent_diff_yields_deletion | Medium | exercised indirectly via diff plan (files to delete) on offline files; logic reviewed | no dedicated CI assertion on the exact delete-set |
| idempotent_second_apply / apply_no_op_when_converged | Medium | code review (empty intent diff + empty drift → "nothing to do") | requires live converged system |
| describe_emits_manifest, *_suppresses_package_pristine_*, *_ghost_*, *_alternative_*, *_repositories_from_reposd, *_unreadable_scope_*, *_omits_genuinely_empty_scope, content-store, scope=full examples | Medium | code review against the Go decisions hints (rpm -V verdict-parse + ghost pass + unpackaged walk + content store + full scan) | require a real rpmdb / privileged root / systemd; not black-box-verifiable on a non-root build host |
| lock_is_fully_resolved_packages_scope | Medium | code review (converge-packages queries rpmdb for the resolved set) | requires live zypper/rpm |
| apply_* convergence/transaction examples | Medium | code review | require a live snapshot transaction mechanism and privilege |

## Specification ambiguities encountered

1. **rpm -V "missing" line column layout.** The decisions hints describe a
   `missing` prefix for deleted files but the exact column order of the type char
   varies across rpm versions. The parser handles both a 2-field and 3-field
   `missing` line and extracts the trailing path; documented here as a minor
   ambiguity resolved conservatively.
2. **packages_divergent identity when the reference is a desired-style manifest.**
   The spec compares "identity fields", but a desired package may carry name only
   while an actual record is fully resolved. `compute-drift` reduces a
   version-less reference package's key to name+arch so a resolved actual record
   still matches by name (otherwise every package would appear divergent). This is
   the most conservative reading consistent with the lock semantics.
3. **YAML "bounded alias expansion".** The spec permits bounded *or* disabled
   alias expansion. The implementation **disables** aliases entirely (the stricter,
   safer option), which satisfies the constraint.

## Rules that could not be implemented exactly as written, and why

- **apply convergence / transaction sealing (STEPS 5–11).** These require a live,
  privileged snapshot transaction mechanism (transactional-update or the
  zypper-merged machinery) that does not exist in the translation environment. The
  convergence logic (`internal/converge`) and the transaction binding
  (`internal/txn`) are implemented and the `apply` verb walks STEPS 1–11 in order;
  with no mechanism present, `acquire-transaction-context` returns a
  `domain=transaction` error and apply exits with an invocation/transaction error
  rather than fabricating a snapshot or silently exiting 0. This is the
  conservative interpretation and matches the spec POSTCONDITION that on any
  non-zero exit the running system is unchanged. End-to-end apply is the subject of
  the spec's later (`apply on a live host`) milestones.
- **describe-actual-state live reads** (rpmdb, repos.d on `/`, systemd, ghost/
  alternatives queries) require a real SUSE system and, for protected content,
  root. The reader is implemented per the Go decisions hints (rpm -V verdict-parse
  + separate ghost-content pass + unpackaged-walk subtraction + content store +
  full scan via rpm -Va) and was smoke-tested against a synthetic root
  (subdirectory traversal, verbatim symlink target, file hashing, `_attributes`
  always an object, content-store blob writing + dedup). The privileged
  config_files self-checks named in the hints (common-auth link, content-bearing
  ghosts, pristine suppression) are root-only and are left for human/CI
  verification on a real host.

## Template deviations (carried from the spec DEPLOYMENT section)

- **NETWORK-CALLS (template: forbidden).** The tool performs no direct network I/O
  of its own; all package retrieval is delegated to the package manager against a
  declared, pinned, signed repository. The supply-chain intent (no curl-style
  fetching) is honoured. Documented as a deviation because the delegated package
  operation reaches the network through `zypper`.
- **Privilege.** Unlike a typical read-only cli-tool, `apply` requires privilege;
  the read-only verbs (`diff`/`verify`/`status`/`describe`) require only read
  access. Carried from the spec.
- **OCI/PKG/binary OUTPUT-FORMATs.** `supported`, not active in any resolved preset
  (no preset present; macOS not declared), so not produced. This is the template's
  prescribed behaviour for inactive `supported` formats, recorded here for
  completeness.

---

## Public API Surface

The names and signatures of symbols exported by the implementation packages,
grouped by module. The next translation of this spec at Version 0.6.6 must
preserve every entry below (additions permitted; removals/renames require a spec
Version increment).

### internal/meta
- `const Version = "0.6.6"`
- `const ProgramName = "zypper-declarative"`
- `const SpecSHA256 = "51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03"`
- `func Generator() string`

### internal/manifest
- `type Format string`
- `const FormatJSON Format = "json"`
- `const FormatYAML Format = "yaml"`
- `var ErrUnknownFormat error`
- `func ParseFormat(s string) (Format, error)`
- `func ResolveFormat(explicit *Format, path string, def Format) Format`
- `type ScopeWrapper[T any] struct { Attributes map[string]interface{}; Elements []T }`
- `func NewScope[T any](attrs map[string]interface{}) ScopeWrapper[T]`
- `type ManifestMeta struct { FormatVersion int; Generator string; CreatedAt string; DesiredSHA256 string }`
- `type PackageRecord struct { Name, Version, Release, Arch string }`
- `type RepositoryRecord struct { Alias, Name, URL, Type string; Enabled, GPGCheck, Autorefresh bool; Priority int }`
- `type ServiceRecord struct { Name, State string }`
- `type ManagedFileRecord struct { Name, Type, Mode, User, Group, SHA256, Target, ContentRef, PackageName, Status string; Changes []string }`
- `type ManagedBaselineRecord struct { Name, Type, Mode, User, Group, SHA256, Target, PackageName string; Changes []string }`
- `type UnmanagedFileRecord struct { Name, Type, Mode, User, Group, SHA256, Target string }`
- `type Manifest struct { Meta ManifestMeta; Packages *ScopeWrapper[PackageRecord]; Repositories *ScopeWrapper[RepositoryRecord]; Services *ScopeWrapper[ServiceRecord]; ConfigFiles *ScopeWrapper[ManagedFileRecord]; ChangedManagedFiles *ScopeWrapper[ManagedBaselineRecord]; UnmanagedFiles *ScopeWrapper[UnmanagedFileRecord] }`
- `type ParseError struct { Domain, Message string }`
- `func (e *ParseError) Error() string`
- `func Parse(data []byte, f Format) (*Manifest, error)`
- `func (m *Manifest) Validate(rejectObservational bool) error`
- `func (m *Manifest) MarshalJSONIndent() ([]byte, error)`
- `func (m *Manifest) MarshalYAML() ([]byte, error)`
- `func (m *Manifest) Serialise(f Format) ([]byte, error)`
- `func (m *Manifest) CanonicalSHA256() string`
- `type LoadOptions struct { Explicit *Format; Default Format; SigVerify bool; Keyring string; RejectObs bool }`
- `type LoadResult struct { Manifest *Manifest; DesiredSHA256 string }`
- `func Load(path string, opts LoadOptions) (*LoadResult, error)`
- `func LoadStateDump(path string, explicit *Format, def Format) (*Manifest, error)`

### internal/record
- `type Diagnostic struct { Severity, Domain, Message string }`
- `func (d *Diagnostic) Error() string`
- `func AppliedPath(root string) string`
- `type LoadResult struct { Record *manifest.Manifest; Present bool }`
- `func LoadApplied(root string) (*LoadResult, error)`
- `func WriteApplied(root string, desired *manifest.Manifest, desiredSHA256 string, resolved *manifest.ScopeWrapper[manifest.PackageRecord]) error`

### internal/diff
- `type Diff struct { PackagesInstall, PackagesRemove []manifest.PackageRecord; ReposSet []manifest.RepositoryRecord; FilesWrite []manifest.ManagedFileRecord; FilesDelete []string; UnitsChange []manifest.ServiceRecord }`
- `func (d *Diff) Empty() bool`
- `type DriftReport struct { FilesModified, FilesExtra []string; UnitsDivergent []manifest.ServiceRecord; PackagesDivergent []manifest.PackageRecord; ManagedFilesModified, UnmanagedFilesPresent []string }`
- `func (r *DriftReport) Empty() bool`
- `func ComputeIntentDiff(desired, applied *manifest.Manifest) *Diff`
- `func ComputeDrift(actual, reference *manifest.Manifest, keepList map[string]bool) *DriftReport`

### internal/txn
- `type Mode string`
- `const ModeAuto, ModeExternal, ModeInternal Mode`
- `type Context struct { Mode Mode; Root string; OpenedHere bool }`
- `type Diagnostic struct { Severity, Domain, Message string }`
- `func (d *Diagnostic) Error() string`
- `type EnvProbe interface { InsideTransaction() bool; ExternalRoot() (string, bool); OpenInternal() (string, error) }`
- `func Acquire(mode Mode, probe EnvProbe) (*Context, error)`

### internal/converge
- `type Diagnostic struct { Severity, Domain, Message string }`
- `func (d *Diagnostic) Error() string`
- `func Packages(ctx *txn.Context, d *diff.Diff, runner syscmd.CommandRunner, repoLock string) (*manifest.ScopeWrapper[manifest.PackageRecord], error)`
- `func Files(ctx *txn.Context, d *diff.Diff, contentStore string, ownedByRPM func(path string) bool, keepList map[string]bool) error`
- `func Units(ctx *txn.Context, d *diff.Diff, runner syscmd.CommandRunner) error`

### internal/syscmd
- `type CommandRunner interface { Run(cmd string, args []string) (string, string, error) }`
- `type OSCommandRunner struct{}`
- `func (r *OSCommandRunner) Run(cmd string, args []string) (string, string, error)`
- `type FakeCommandRunner struct { Responses map[string]FakeResponse }`
- `type FakeResponse struct { Stdout, Stderr string; Err error }`
- `func (f *FakeCommandRunner) Run(cmd string, args []string) (string, string, error)`

### internal/state
- `type OnUnreadable string`
- `const OnUnreadableError, OnUnreadableWarn OnUnreadable`
- `type Scope string`
- `const ScopeEtc, ScopeFull Scope`
- `type Diagnostic struct { Severity, Domain, Message string }`
- `func (d *Diagnostic) Error() string`
- `type Options struct { Root string; OnUnreadable OnUnreadable; Scope Scope; ContentStore string; KeepList map[string]bool; Runner syscmd.CommandRunner }`
- `type Result struct { Manifest *manifest.Manifest; Diagnostics []Diagnostic }`
- `func Read(opts Options) (*Result, error)`

### internal/cli
- `const ExitOK = 0; ExitError = 1; ExitInvocation = 2`
- `type App struct { Stdout io.Writer; Stderr io.Writer }`
- `func (a *App) Run(args []string) int`
- `type Config struct { ... }` (resolved key=value options; internal-use, exported for testability)
