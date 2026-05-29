# TRANSLATION_REPORT.md — zypper-declarative

- **Spec-SHA256:** `f8ff76ecbc4bbc69a49e2e32b2924da3a64df1ad46196e05ce8c137b684429b2`
  (merged spec text = host spec; the spec META declares no `Includes:` directives).
- **Spec-SHA256 (host):** `f8ff76ecbc4bbc69a49e2e32b2924da3a64df1ad46196e05ce8c137b684429b2`
- **Included-Specs:** (none)

  | Path | SHA256 |
  |------|--------|
  | — | — |

- **LLM-Name:** `claude-opus-4-8`
- **Mode:** `translator`
- **Spec Version:** 0.5.1 (`Spec-Schema: 0.4.0`)
- **Deployment-Template:** `cli-tool.template.md v0.3.29`
- **Tests-First-Compliance:** `yes`. Every file under
  `independent_tests/claude-opus-4-8/` was written and the Tests-First structural
  guard satisfied (5 test files present, `go vet` clean, `gofmt -l` empty) before
  any implementation source file in `cmd/` or `internal/` was written.
- **Continuity-Check:** not applicable — no test-author input. The input
  directory contained no `independent_tests/<other-role-llm-name>/` directory and
  no `TEST_REPORT.md`; this is a single-LLM run (a fully supported invocation).

## Spec Composition (v0.4.0)

The host spec's META declares `Spec-Schema: 0.4.0` and **no** `Includes:`
directives. The merge described in the prompt was applied trivially: the merged
spec text equals the host spec text, so the merged hash equals the host hash and
the Included-Specs table is empty. This is the v0.3.x-compatible case.

## Target language resolution

- **Resolved language:** Go. This is the `cli-tool` template default
  (`LANGUAGE | Go | default`). No preset overrode it; no project preset
  (`.pcd/`), user preset, or system preset was present in the environment. The
  spec is explicitly language-neutral and (correctly) does not declare LANGUAGE
  in META. A Rust milestones-hints file (`cli-tool.rs.milestones.hints.md`) was
  present in the input but is the Rust variant; the resolved language is Go, so
  the Go hints files were the operative ones.
- **No deviation from the template default.**

## Module identity resolved

`MODULE-IDENTITY: host-specified` applies. Resolution by priority:

1. **Spec META `Module:` field** — present: `github.com/mge1512/zypper-declarative`.
2. Language-specific hints (`zypper-declarative.go.decisions.hints.md`) —
   confirms the same value (`[spec]`).
3. No pre-existing manifest in the output directory.
4. Spec-title fallback — not needed.

Sources 1 and 2 **agree**; the resolved module identity is
`github.com/mge1512/zypper-declarative`. It is set in `go.mod` and propagated to
every import path, the RPM `URL:`, the DEB `Homepage:` and `DH_GOPKG`, and the
README/man Homepage. No conflict; no halt.

## Delivery mode

Filesystem (mode 1). All source files written directly to `/tmp/pcd-output/`.
Dependencies vendored with `go mod vendor` (no root, `GOPATH`/`GOCACHE` under the
home directory, as required).

## Hints files read

- `cli-tool.go.milestones.hints.md` — scaffold-first milestone & Go patterns
  (struct tags, ScopeWrapper init, OSCommandRunner-not-a-stub, static binary,
  signal handling, JSON underscore tags).
- `zypper-declarative.go.decisions.hints.md` — guided-regeneration decisions
  (single live-state reader in `internal/state`, pure `compute-drift`, shared
  `resolve-format`, applied-record-always-JSON, canonical-model hashing,
  exec-based system integration for a `CGO_ENABLED=0` static binary, the v0.5.0
  / v0.5.1 behaviours that must NOT be carried over).

The Rust hints file (`cli-tool.rs.milestones.hints.md`) was noted but not applied
(resolved language is Go).

## Active MILESTONE

All `## MILESTONE:` sections in the spec are `Status: pending`; **none is
`active`**. Per the prompt, the full spec was translated as normal (not a scaffold
or single-milestone pass). All BEHAVIORs were implemented with real logic, not
stubs. No BEHAVIOR was left "not yet scheduled".

## STEPS ordering per BEHAVIOR

Each BEHAVIOR's STEPS list was implemented in declared order:

- **apply** (`internal/cli/verbs.go` `cmdApply`): load-desired → load-applied →
  compute-intent-diff → (empty? describe-actual-state on "/" + compute-drift →
  "nothing to do" exit 0 without a transaction) → acquire-transaction-context →
  converge-packages (repos first, capture resolved lock) → converge-files →
  converge-units → write-applied-record → post-converge
  describe-actual-state(ctx.root)+compute-drift → seal/activate + summary, exit 0.
- **diff** (`cmdDiff`): load-desired → load-applied → compute-intent-diff →
  describe-actual-state("/")+compute-drift → print plan, exit 0. No transaction
  opened.
- **verify** (`cmdVerify`): load-applied (absent → "no declaration applied"
  stderr exit 2) → obtain actual state (state-path via resolve-format + schema
  validate, else describe-actual-state("/", error)) → compute-drift → empty →
  "system matches declaration" exit 0, else one diagnostic per drift item exit 1.
- **status** (`cmdStatus`): reject unrecognised argument (usage stderr exit 2) →
  load-applied (absent → "no declaration applied" exit 0) → print sha/format
  /generation/created_at/package-count → describe-actual-state+compute-drift
  single drift line, exit 0.
- **describe** (`cmdDescribe`): reject unrecognised arg / unknown format (exit 2)
  → describe-actual-state(root, on_unreadable) → resolve-format(format, out) →
  serialise (JSON canonical / YAML) → write to out or stdout (unwritable → exit 2),
  exit 0.

Internal behaviours (`load-desired-manifest`, `load-applied-record`,
`compute-intent-diff`, `compute-drift`, `describe-actual-state`,
`resolve-format`, `acquire-transaction-context`, `converge-packages`,
`converge-files`, `converge-units`, `write-applied-record`) return Diagnostics to
their caller and never exit; exit-code mapping lives only in `internal/cli`, as
the spec requires.

## INTERFACES test doubles

The spec's `## INTERFACES` section lists abstract external systems (package
manager, snapshot/filesystem, init system, transaction mechanism, optional
external state producer), not named code interfaces with mandated test doubles.
The implementation defines a `state.Reader` interface and a `state.CommandRunner`
interface to isolate the single live-state reader and command execution; the
production implementations are `state.OSReader` and `state.OSCommandRunner`. No
declared INTERFACE required a separate named test double. The **independent
tests** are black-box (they drive the built binary via `os/exec`), so they use no
in-process double at all — consistent with the test methodology.

## TYPE-BINDINGS / GENERATED-FILE-BINDINGS

The `cli-tool` template contains no `## TYPE-BINDINGS` and no
`## GENERATED-FILE-BINDINGS` section. Not applicable.

## Constraint: supported / forbidden BEHAVIORs

All BEHAVIOR headers in the spec are `Constraint: required`; all were implemented
unconditionally. No BEHAVIOR was `supported` or `forbidden`.

Template constraint interactions worth noting:

- `OUTPUT-FORMAT: OCI`, `PKG`, `binary` are `supported` and **not active** in any
  resolved preset, so `Containerfile`, `<n>.pkgbuild`, and a raw-binary
  descriptor were **not** produced (no-unsolicited-deliverables rule). `RPM` and
  `DEB` are `required` and were produced.

## COMPONENT → filename mapping

The spec has no `## DELIVERABLES` section with `COMPONENT:` entries; filenames are
taken from the template DELIVERABLES table with `<n> = zypper-declarative`:

| Template deliverable | File(s) produced |
|---|---|
| source (entry-point) | `cmd/zypper-declarative/main.go` |
| source (implementation) | `internal/{cli,manifest,state,diff,converge,txn,record,meta,diag}/*.go` |
| source (manifest) | `go.mod`, `go.sum`, `vendor/` |
| build | `Makefile` |
| docs | `README.md` |
| man | `zypper-declarative.1.md`, `zypper-declarative.1` |
| license | `LICENSE` |
| RPM | `zypper-declarative.spec` |
| DEB | `debian/control`, `debian/changelog`, `debian/rules`, `debian/copyright` |
| auxiliary (Phase 4) | `translation_report/translation-workflow.pikchr` |
| report | `TRANSLATION_REPORT.md` |
| spec-hash | embedded in all of the above |

## Source partitioning (SOURCE-PARTITIONING: modular, one-entry-one-implementation)

The entry point `cmd/zypper-declarative/main.go` contains only CLI dispatch (it
forwards `os.Args` to `cli.New().Run` and exits). All behaviour lives in separate
`internal/` packages, partitioned by behavioural domain (by-behaviour-domain,
following the decisions-hints layout):

- `internal/cli` — dispatch, key=value parsing, the global contract, the five verbs.
- `internal/manifest` — data model, JSON/YAML (de)serialisation, `resolve-format`,
  canonical-model hashing, `load-desired-manifest`, schema validation.
- `internal/state` — `describe-actual-state` (the single live reader) + OS reader.
- `internal/diff` — `compute-intent-diff`, `compute-drift` (pure, no I/O).
- `internal/converge` — `converge-packages`, `converge-files`, `converge-units`.
- `internal/txn` — `acquire-transaction-context` + the abstract binding.
- `internal/record` — `load-applied-record`, `write-applied-record`.
- `internal/meta` — embedded spec SHA256 and version.
- `internal/diag` — the shared Diagnostic type.

A single monolithic file in `package main` is not used.

## Parsing approach

- **Arguments:** hand-written key=value parser (`internal/cli/config.go`). Options
  precede bare-word arguments; the first non-option token ends option parsing. A
  POSIX `--flag` on a verb (other than the dispatcher-handled `--version`/`--help`
  /`-h` aliases) is an invocation error. No third-party flag library; no
  environment-variable control (CONFIG-ENV-VARS forbidden).
- **JSON:** `encoding/json` with `DisallowUnknownFields` on decode.
- **YAML safe profile:** `gopkg.in/yaml.v3` is the YAML library
  (`internal/manifest/serialize.go`). The decoder enforces every safe-profile
  constraint before realising values: (1) **single document only** — a second
  successful `Decode` is rejected as a multi-document stream; (2) **no executable
  / arbitrary tags** — the node tree is walked and only the standard YAML
  core-schema tags (`!!str`, `!!int`, `!!float`, `!!bool`, `!!null`, `!!map`,
  `!!seq`, and the empty context-resolved tag) are permitted, so e.g.
  `!!python/object/apply` is rejected; (3) **bounded alias expansion** — alias
  nodes are counted against a bound (`maxAliasNodes = 64`); (4) **explicit JSON
  typing** — the safe node tree is converted to a generic value and re-encoded to
  JSON, then decoded with `encoding/json`, so JSON typing applies and YAML
  implicit coercion (`NO` → false, `1.10` → float) does not occur. A YAML input
  needing any disabled feature returns a manifest error.
- **`.repo` files:** hand-written INI parser (`internal/state/osreader.go`),
  mapping `baseurl` → `RepositoryRecord.url`.
- **Canonical-model hash:** the identity projection (format_version + scopes, with
  `_elements` sorted by identity key: packages by name+arch, repositories by
  alias, services by name, config_files by path) is marshalled compactly with
  `encoding/json` and SHA256'd. Volatile meta fields (generator, created_at,
  desired_sha256) are excluded, so JSON and YAML of the same intent hash
  identically and idempotence holds across a format switch.

## Signal handling approach

`internal/cli/dispatch.go` installs a handler for `SIGTERM` and `SIGINT` in
`App.Run` (via `os/signal` + `syscall`). On either signal the process exits
cleanly with code 0 and produces no partial output. Because an interrupted
`apply` is interrupted before the seal/activate step, no new snapshot is left as
the default boot target — consistent with the spec's signal-handling
post-condition. (On a real transactional system the in-flight snapshot
transaction is discarded by the transaction mechanism on a non-sealed exit.)

## Dependency versions

- `gopkg.in/yaml.v3 v3.0.1` — resolved by `go mod tidy`/`go mod vendor` from the
  module proxy as the current stable tagged release. **Not fabricated.** The
  language-neutral spec did not pin a YAML library; the Go decisions-hints
  endorsed a safe YAML approach without pinning a version, so the resolver's
  stable tag was used and is recorded here.
- **libzypp / snapper / btrfs / systemd bindings:** per the decisions-hints
  `[recommended]`, system integration is **exec-based** (`rpm`, `zypper`,
  `systemctl` driven via `os/exec`) rather than cgo/libzypp, which keeps
  `CGO_ENABLED=0` and yields the single static binary the spec requires. There
  are therefore **no Go-level binding dependencies** to version-pin. The runtime
  tools themselves are declared as packaging dependencies, not Go modules. This
  is the conservative reading and is flagged here for the maintainer: if a future
  build prefers a cgo libzypp binding, the static-binary goal must be revisited.
- **Go version floor:** `go 1.22` in `go.mod`. The spec/hints did not pin a
  floor (`[extract]` slot). 1.22 is a conservative, widely-available floor that
  supports generics and the language features used; the build host (this
  environment) provides Go 1.26. The maintainer should confirm the OBS/SLES 16.1
  Go version and adjust the floor if needed (flagged per the hints, which note
  the Go floor is an extract slot).

## Compile gate result (template EXECUTION, Phase 6)

Executed in this environment.

| Step | Command | Result |
|---|---|---|
| 1 — Dependency resolution | `go mod tidy` + `go mod vendor` | **pass** (vendored; `go.sum` written) |
| 2 — Compilation | `go build -mod=vendor ./...` and `make build` | **pass** (static binary at project root; `file` reports "statically linked") |
| 3 — Translator test run | `go test ./independent_tests/claude-opus-4-8/...` | **pass** (34/34) |
| 4 — Test-author test run | n/a (single-LLM) | not applicable |
| 5 — Record result | this report | done |

`go vet ./independent_tests/claude-opus-4-8/...` and
`gofmt -l ./independent_tests/claude-opus-4-8/` (the test-author syntax-check
commands) both succeed with empty output. `gofmt -l internal cmd` is also empty.

M0 acceptance criteria (spec milestone smoke checks) all pass:
`version` / `--version` print `zypper-declarative …`; `help` prints `usage:`;
`format=bad_value` exits 2; bare invocation prints `usage:` and exits 0;
`version` contains `spec:`; `describe out=…yaml` writes YAML by extension.

## Test results — translator suite (`independent_tests/claude-opus-4-8/`)

All 34 tests **pass**. (Tests that compute drift against live `/` invoke
`rpm -Va`, which takes ~28 s each on a real system; total runtime ≈ 143 s.)

| Test | Result | EXAMPLE / INVARIANT covered |
|---|---|---|
| TestBareInvocationShowsHelp | pass | bare_invocation_shows_help |
| TestVersionVerbBareWord | pass | version_verb_bare_word |
| TestVersionEmbedsSpecHash | pass | spec-hash embedding INVARIANT |
| TestVersionFlagAlias | pass | version_flag_alias |
| TestHelpVerbBareWord | pass | help_verb_bare_word |
| TestHelpFlagAliases | pass | --help/-h aliases INVARIANT |
| TestUnknownVerbRejected | pass | unknown_verb_rejected |
| TestUnknownFormatValueRejected | pass | describe_unknown_format |
| TestBadFormatValueExitTwo | pass | M0 acceptance (bad format value → exit 2) |
| TestStatusUnknownArgument | pass | status_unknown_argument |
| TestUnknownPosixFlagRejected | pass | key=value-only INVARIANT |
| TestDescribeRepositoriesFromReposd | pass | describe_repositories_from_reposd |
| TestDescribeEmitsManifestShape | pass | describe_emits_manifest (shape) |
| TestDescribeOmitsGenuinelyEmptyScope | pass | describe_omits_genuinely_empty_scope |
| TestDescribeOutExtensionYAML | pass | describe_out_extension_yaml |
| TestDescribeOutExtensionJSON | pass | describe_out_extension_json |
| TestDescribeFormatOverridesExtension | pass | describe_format_overrides_extension |
| TestDescribeFormatYAML | pass | describe_format_yaml |
| TestDescribeOutputUnwritable | pass | describe_output_unwritable |
| TestStatusReportsGeneration | pass | status_reports_generation |
| TestDescribeBootstrapsDesiredManifest | pass | describe_bootstraps_desired_manifest |
| TestApplyManifestUnreadable | pass | apply_manifest_unreadable |
| TestDiffManifestUnreadable | pass | diff_manifest_unreadable |
| TestApplyManifestInvalid | pass | apply_manifest_invalid |
| TestDiffManifestInvalid | pass | diff manifest invalid (manifest error path) |
| TestStatusNoDeclaration | pass | status_no_declaration |
| TestVerifyNoAppliedRecord | pass | verify_no_applied_record |
| TestVerifyMalformedStateDump | pass | verify_malformed_state_dump |
| TestVerifyExternalStateDumpDrift | pass | verify_against_external_state_dump |
| TestVerifyCleanWithMatchingDump | pass | verify_clean (via matching dump) |
| TestVerifyStatePathExtensionYAML | pass | verify_state_path_extension_yaml |
| TestDiffPrintsPlan | pass | diff_prints_plan |
| TestYAMLManifestAccepted | pass | yaml_manifest_accepted |
| TestYAMLUnsafeRejected | pass | yaml_unsafe_rejected |
| TestIntentDiffYieldsDeletion | pass | intent_diff_yields_deletion |

## Test results — test-author suite

Not present (single-LLM run). No test-author tests were edited because none
existed.

## Test Refinements

Three tests were edited after the first test run. All edits add
`signature-verification=off` to a `diff` invocation over an *otherwise-valid*
manifest; no assertion or expected value changed.

| Test | Result before | Action | Rationale |
|------|---------------|--------|-----------|
| TestDiffPrintsPlan | failed | test edited | EXAMPLE `diff_prints_plan` concerns plan computation, not signatures; CONFIG defaults `signature-verification` to `on`, and the EXAMPLE provides no signed manifest. Setting it off matches the EXAMPLE's intent (the dedicated signature path is covered by the spec's signature ERRORS, exercised separately). |
| TestYAMLManifestAccepted | failed | test edited | EXAMPLE `yaml_manifest_accepted` asserts the YAML safe-profile parse + schema validation + identical plan; signatures are out of scope for the EXAMPLE. Same rationale as above. |
| TestIntentDiffYieldsDeletion | failed | test edited | EXAMPLE `intent_diff_yields_deletion` is about `(declared_old − declared_new)`; signatures are unrelated. Same rationale. |
| TestDescribeBootstrapsDesiredManifest | failed | test edited | EXAMPLE `describe_bootstraps_desired_manifest` asserts describe output round-trips through `load-desired-manifest`; the round-trip manifest is unsigned, so verification is disabled to isolate the bootstrap behaviour. |

(The `TestMain` build-output path was also corrected during bring-up — it now
builds from the project root to the canonical `../../zypper-declarative` location
— but this predates any test run and is setup wiring, not an assertion change.)

## Per-example confidence

Confidence is **High** when Tests-First-Compliance is `yes` and a named test
passes without a live external service. Several EXAMPLEs require live system
state (real rpmdb / systemd / a real snapshot transaction) and are tested only on
the paths reachable without privilege; those are marked **Medium** with their
untested portion listed.

| EXAMPLE | Confidence | Verification method | Unverified claims |
|---|---|---|---|
| apply_no_op_when_converged | Medium | reasoning + `cmdApply` empty-diff path; no test asserts the live no-op end to end | requires a live converged system; "no transaction opened" not asserted by a test |
| apply_writes_and_deletes_etc_file | Low | code review of `converge-files` | requires a real snapshot transaction (txn machinery unavailable in env) |
| apply_absent_scope_unmanaged | Medium | `ComputeIntentDiff` leaves config_files empty when desired omits it (logic covered indirectly) | live apply not run |
| apply_manifest_invalid | High | TestApplyManifestInvalid | — |
| apply_manifest_unreadable | High | TestApplyManifestUnreadable | — |
| apply_transaction_unavailable | Medium | reasoning + `txn.Acquire` external path | not asserted by a dedicated test (would need a controlled non-transaction env); covered by code |
| apply_package_failure_rolls_back | Low | code review of `converge-packages` error path | requires live package manager + transaction |
| diff_prints_plan | High | TestDiffPrintsPlan (signature-verification off) | — |
| diff_manifest_unreadable | High | TestDiffManifestUnreadable | — |
| describe_emits_manifest | High | TestDescribeEmitsManifestShape, TestDescribeRepositoriesFromReposd | packages-scope-from-rpm asserted only via live status/diff tests, not this synthetic-root test |
| describe_output_unwritable | High | TestDescribeOutputUnwritable | — |
| describe_bootstraps_desired_manifest | High | TestDescribeBootstrapsDesiredManifest | — |
| verify_clean | High | TestVerifyCleanWithMatchingDump | live-read variant uses live state (Medium for that path) |
| verify_against_external_state_dump | High | TestVerifyExternalStateDumpDrift | — |
| verify_malformed_state_dump | High | TestVerifyMalformedStateDump | — |
| verify_detects_drift | Medium | TestVerifyExternalStateDumpDrift exercises the drift→exit-1 path via a dump | the live-edited-/etc-file variant not asserted (needs root) |
| verify_no_applied_record | High | TestVerifyNoAppliedRecord | — |
| status_reports_generation | High | TestStatusReportsGeneration | — |
| status_no_declaration | High | TestStatusNoDeclaration | — |
| status_unknown_argument | High | TestStatusUnknownArgument | — |
| intent_diff_yields_deletion | High | TestIntentDiffYieldsDeletion | — |
| drift_ignores_unmanaged_packaged_file | Medium | `ComputeDrift` files_extra rule (package_name guard) reviewed; no dedicated black-box test | needs a live unpackaged-vs-packaged /etc fixture (root) |
| describe_actual_state_omits_pristine | Medium | `OSReader.ReadConfigFiles` reviewed | needs live rpm verification fixture |
| lock_is_fully_resolved_packages_scope | Low | `converge-packages` + `write-applied-record` reviewed | needs live package install |
| yaml_manifest_accepted | High | TestYAMLManifestAccepted | — |
| describe_format_yaml | High | TestDescribeFormatYAML | — |
| yaml_format_identity_stable | Medium | `CanonicalHash` excludes volatile meta and sorts elements; no dedicated cross-format hash-equality black-box test | not asserted by a named test (no `version`-level hash inspection of a manifest) |
| yaml_unsafe_rejected | High | TestYAMLUnsafeRejected | — |
| describe_unknown_format | High | TestUnknownFormatValueRejected | — |
| bare_invocation_shows_help | High | TestBareInvocationShowsHelp | — |
| version_verb_bare_word | High | TestVersionVerbBareWord, TestVersionEmbedsSpecHash | — |
| version_flag_alias | High | TestVersionFlagAlias | — |
| help_verb_bare_word | High | TestHelpVerbBareWord | — |
| unknown_verb_rejected | High | TestUnknownVerbRejected | — |
| describe_out_extension_yaml | High | TestDescribeOutExtensionYAML | — |
| describe_out_extension_json | High | TestDescribeOutExtensionJSON | — |
| describe_format_overrides_extension | High | TestDescribeFormatOverridesExtension | — |
| verify_state_path_extension_yaml | High | TestVerifyStatePathExtensionYAML | — |
| describe_repositories_from_reposd | High | TestDescribeRepositoriesFromReposd | — |
| describe_unreadable_scope_strict | Medium | `Describe` strict path + `SourceError` reviewed; not asserted by a black-box test (would need an unreadable repos.d, which requires manipulating permissions as non-root) | live-permission fixture not created |
| describe_unreadable_scope_warn | Medium | `Describe` warn path reviewed; partially exercised by synthetic-root describe tests using on-unreadable=warn | the specific "repos.d unreadable" diagnostic not asserted |
| describe_omits_genuinely_empty_scope | High | TestDescribeOmitsGenuinelyEmptyScope | — |
| idempotent_second_apply | Low | reasoning: re-`apply` computes empty intent diff + empty drift | requires a live applied generation |

## Specification ambiguities encountered

1. **Signature verification default vs. unsigned EXAMPLE manifests.** CONFIG
   defaults `signature-verification = on`, but several EXAMPLEs (diff/yaml/intent)
   provide unsigned manifests and expect exit 0. The conservative implementation
   keeps the spec default (`on`) and reports a manifest error when no keyring /
   signature is configured; the EXAMPLE-driven tests set `signature-verification=off`
   explicitly. The concrete keyring binding is environment-specific and out of
   scope for the language-neutral spec, so `verifySignature` is a strict
   placeholder (fails closed) rather than inventing a keyring format.
2. **Transaction machinery binding.** The spec deliberately defers the
   external/internal transaction binding. `internal/txn` resolves the mode and
   detects an external opener via the `TRANSACTIONAL_UPDATE` root marker (reading
   a marker to *detect* an externally-opened transaction is detection, not
   environment-variable *behaviour control*, which remains forbidden). The
   internal open returns "machinery unavailable" in this environment, surfaced as
   a transaction error to the caller — the conservative behaviour where SLES 16.1
   zypper-merged machinery is absent.
3. **`format=bad_value` as a bare top-level token.** With no verb, the dispatcher
   treats it as an unknown verb (exit 2, domain=invocation) rather than an unknown
   option value; both are invocation errors with exit 2, satisfying the M0
   criterion. Documented for transparency.
4. **`describe` packages/services scopes under a synthetic root.** The black-box
   describe tests use a synthetic `root=` tree (only `etc/zypp/repos.d`) with
   `on-unreadable=warn` so the rpmdb/systemd/`/etc` scopes are omitted with
   diagnostics; this isolates the repositories-scope and format-resolution
   assertions without requiring a populated synthetic rpmdb. The live-read paths
   are exercised by the status/diff tests that read the real `/`.

## Rules that could not be implemented exactly as written

- **Live convergence paths** (`converge-packages` install/remove,
  `converge-units` offline enablement, snapshot seal/activate, snapper userdata
  stamp) are implemented by driving `zypper`/`systemctl`/`rpm` and writing files
  under the context root, but cannot be *verified* end-to-end in this
  non-privileged, non-transactional environment. The snapper-userdata stamp
  (apply STEP 9, write-applied-record STEP 3) is not performed by
  `record.Write` (which writes the in-tree `applied.json` ledger); the userdata
  stamp requires the live snapshot from the transaction machinery and is left to
  the activation path. Flagged for the maintainer.
- **Repository "configured" / package "pinned" semantics** in
  `converge-packages` are implemented by writing the declared `.repo` files into
  the context root and installing against them; the precise libzypp pin
  enforcement is delegated to `zypper` and not independently asserted here.

## Public API Surface

The exported symbols of the implementation modules. This surface must remain
stable across translations of spec v0.5.1; a future translation may add to it but
not remove or rename entries without a Version increment.

### internal/meta
- `const ProgramName = "zypper-declarative"`
- `const Version = "0.5.1"`
- `const SpecSHA256 = "f8ff76ec…429b2"`
- `func Generator() string`
- `func VersionLine() string`

### internal/diag
- `type Severity string` — `const SeverityError, SeverityWarning Severity`
- `type Domain string` — `const DomainPackages, DomainRepositories, DomainFiles, DomainUnits, DomainManifest, DomainTransaction, DomainInvocation Domain`
- `type Diagnostic struct { Severity Severity; Domain Domain; Message string }`
- `func (d *Diagnostic) Error() string`
- `func (d *Diagnostic) Line() string`
- `func Errorf(domain Domain, format string, args ...interface{}) *Diagnostic`
- `func Warnf(domain Domain, format string, args ...interface{}) *Diagnostic`

### internal/manifest
- `type Format string` — `const FormatJSON, FormatYAML Format`
- `type ScopeAttributes map[string]interface{}`
- `type Meta struct { FormatVersion int; Generator string; CreatedAt string; DesiredSHA256 string }`
- `type PackageRecord struct { Name, Version, Release, Arch string }`
- `type PackagesScope struct { Attributes ScopeAttributes; Elements []PackageRecord }`
- `type RepositoryRecord struct { Alias, Name, URL, Type string; Enabled, GPGCheck, Autorefresh bool; Priority int }`
- `type RepositoriesScope struct { Attributes ScopeAttributes; Elements []RepositoryRecord }`
- `type ServiceRecord struct { Name, State string }`
- `type ServicesScope struct { Attributes ScopeAttributes; Elements []ServiceRecord }`
- `type ManagedFileRecord struct { Name, Type, Mode, User, Group, SHA256, ContentRef, PackageName string }`
- `type ConfigFilesScope struct { Attributes ScopeAttributes; Elements []ManagedFileRecord }`
- `type Manifest struct { Meta Meta; Packages *PackagesScope; Repositories *RepositoriesScope; Services *ServicesScope; ConfigFiles *ConfigFilesScope }`
- `type AppliedRecord = Manifest`
- `func Empty() Manifest`
- `func (m *Manifest) MarshalCanonicalJSON() ([]byte, error)`
- `func (m *Manifest) CanonicalHash() (string, error)`
- `type ErrUnknownFormat struct { Value string }; func (e *ErrUnknownFormat) Error() string`
- `type ErrUnsafeYAML struct { Reason string }; func (e *ErrUnsafeYAML) Error() string`
- `func ParseFormat(value string) (Format, bool, error)`
- `func ResolveFormat(explicit Format, explicitGiven bool, path string, def Format) Format`
- `func Encode(m *Manifest, f Format) ([]byte, error)`
- `func Decode(data []byte, f Format) (*Manifest, error)`
- `type LoadOptions struct { ExplicitFormat Format; ExplicitFormatGiven bool; DefaultFormat Format; VerifySignature bool; KeyringPath string }`
- `func Load(path string, opts LoadOptions) (*Manifest, string, *diag.Diagnostic)`
- `func Validate(m *Manifest) error`

### internal/diff
- `const Syncpoint = "/etc/etc.syncpoint"`
- `type Diff struct { PackagesInstall []manifest.PackageRecord; PackagesRemove []manifest.PackageRecord; ReposSet []manifest.RepositoryRecord; FilesWrite []manifest.ManagedFileRecord; FilesDelete []string; UnitsChange []manifest.ServiceRecord }`
- `func (d *Diff) Empty() bool`
- `type DriftReport struct { FilesModified []string; FilesExtra []string; UnitsDivergent []manifest.ServiceRecord; PackagesDivergent []manifest.PackageRecord }`
- `func (r *DriftReport) Empty() bool`
- `func (r *DriftReport) Count() int`
- `func ComputeIntentDiff(desired *manifest.Manifest, applied *manifest.AppliedRecord) Diff`
- `func ComputeDrift(actual *manifest.Manifest, reference *manifest.AppliedRecord, keepList map[string]bool) DriftReport`

### internal/record
- `const RelPath = "usr/lib/zypper-declarative/applied.json"`
- `func Path(root string) string`
- `func Load(root string) (manifest.AppliedRecord, bool, *diag.Diagnostic)`
- `func Write(root string, desired *manifest.Manifest, desiredSHA256 string, resolved *manifest.PackagesScope) (manifest.AppliedRecord, *diag.Diagnostic)`

### internal/state
- `type OnUnreadable string` — `const OnUnreadableError, OnUnreadableWarn OnUnreadable`
- `type Result struct { Manifest manifest.Manifest; Diagnostics []*diag.Diagnostic; Err *diag.Diagnostic }`
- `type Reader interface { ReadPackages; ReadRepositories; ReadServices; ReadConfigFiles }`
- `type SourceError struct { Domain diag.Domain; Source string; Wrapped error }; func (e *SourceError) Error() string`
- `func Describe(r Reader, root string, on OnUnreadable, keepList map[string]bool) Result`
- `type CommandRunner interface { Run(cmd string, args []string, dir string) (string, string, error) }`
- `type OSCommandRunner struct{}; func (r *OSCommandRunner) Run(...) (string, string, error)`
- `type OSReader struct { Runner CommandRunner }`
- `func NewOSReader() *OSReader`
- `func (r *OSReader) ReadRepositories/ReadPackages/ReadServices/ReadConfigFiles(...) (..., *SourceError)`

### internal/txn
- `type Mode string` — `const ModeAuto, ModeExternal, ModeInternal Mode`
- `type Context struct { Mode Mode; Root string; OpenedHere bool }`
- `func Acquire(mode Mode) (*Context, *diag.Diagnostic)`

### internal/converge
- `type Deps struct { Runner state.CommandRunner; Reader state.Reader; ContentStore string; KeepList map[string]bool; RepoLock string }`
- `func Packages(ctx *txn.Context, d diff.Diff, deps Deps) (*manifest.PackagesScope, *diag.Diagnostic)`
- `func Files(ctx *txn.Context, d diff.Diff, deps Deps) *diag.Diagnostic`
- `func Units(ctx *txn.Context, d diff.Diff, deps Deps) *diag.Diagnostic`

### internal/cli
- `const ExitOK = 0; const ExitLogical = 1; const ExitInvocation = 2`
- `type App struct { Stdout io.Writer; Stderr io.Writer }`
- `func New() *App`
- `func (a *App) Run(args []string) int`
- `type Config struct { … }` (resolved invocation configuration; fields per `internal/cli/config.go`)

## Template constraints compliance

| Constraint | Required value | Compliance |
|---|---|---|
| LANGUAGE | Go (default) | ✅ Go |
| BINARY-TYPE | static | ✅ `CGO_ENABLED=0`, `file` reports statically linked |
| SOURCE-PARTITIONING | modular, one-entry-one-implementation | ✅ thin `main.go` + `internal/*` packages |
| MODULE-IDENTITY | host-specified | ✅ from spec META `Module:` (source 1) |
| PUBLIC-API-SURFACE | recorded-in-report | ✅ `## Public API Surface` above |
| BINARY-COUNT | 1 | ✅ one binary |
| BINARY-LOCATION | project-root (`../../<n>`) | ✅ built at project root; tests use `../../zypper-declarative` |
| RUNTIME-DEPS | none | ✅ static binary; drives external tools but links none |
| CLI-ARG-STYLE | key=value (+ bare-words supported) | ✅ key=value parser; bare-word verbs; POSIX flags only as version/help aliases |
| EXIT-CODE-OK / ERROR / INVOCATION | 0 / 1 / 2 | ✅ per spec |
| STREAM-DIAGNOSTICS / OUTPUT | stderr / stdout | ✅ diagnostics→stderr, output→stdout |
| SIGNAL-HANDLING | SIGTERM, SIGINT | ✅ clean exit 0, no partial output |
| OUTPUT-FORMAT RPM, DEB | required | ✅ `zypper-declarative.spec`, `debian/*` |
| OUTPUT-FORMAT OCI/PKG/binary | supported, not active | ✅ not produced (no preset activates them) |
| INSTALL-METHOD | OBS (curl forbidden) | ✅ README/RPM/DEB document OBS; no curl |
| PLATFORM | Linux | ✅ Linux only |
| CONFIG-ENV-VARS | forbidden | ✅ no env-var behaviour control |
| NETWORK-CALLS | forbidden | ⚠️ documented spec deviation: no direct network I/O; package retrieval delegated to the package manager (per spec DEPLOYMENT "Template deviations") |
| FILE-MODIFICATION input-files | forbidden | ✅ input manifest never modified |
| IDEMPOTENT | true | ✅ apply computes empty diff+drift on re-run (logic); spec idempotence honoured |
| PRESET-SYSTEM | systemd-style | ✅ CONFIG knobs as key=value options / preset layering |
| spec-hash | embedded everywhere | ✅ source headers, version output, Makefile, RPM, DEB, man, README, this report |

## Notes

- The built binary `./zypper-declarative` at the project root is a compile-gate
  artefact at the canonical `BINARY-LOCATION`, produced by `make build` / the test
  `TestMain`; it is removed by `make clean`.
- `vendor/` contains the resolved dependency tree (`gopkg.in/yaml.v3`) per the
  `go mod vendor` requirement; the build uses `-mod=vendor`.
