# TRANSLATION_REPORT.md — zypper-declarative

- **Spec-SHA256:** `f2cc80627e483a48bb8411d297711bc5f6c6e74c28dbf0dafc8fe7bd8817251e`
  (merged spec text; host has no `Includes:` directives, so this equals the host hash)
- **Spec-SHA256 (host):** `f2cc80627e483a48bb8411d297711bc5f6c6e74c28dbf0dafc8fe7bd8817251e`
- **Included-Specs:** (none)

  | Path | SHA256 |
  |------|--------|
  | — | — |

- **LLM-Name:** `claude-opus-4-8`
- **Mode:** `translator`
- **Tests-First-Compliance:** `yes` — every file under
  `independent_tests/claude-opus-4-8/` was written and the Tests-First structural
  guard (a non-empty test directory) was satisfied before any implementation
  source file in `cmd/` or `internal/` was written.
- **Continuity-Check:** not applicable — no test-author input. The input
  directory contained no `independent_tests/<other-role-llm-name>/` directory and
  no `TEST_REPORT.md`. This is a single-LLM run (a fully supported invocation).
- **Deployment-Template:** `cli-tool.template.md v0.3.29`
- **Delivery mode:** Filesystem. Source files written directly to
  `/tmp/pcd-output/`. No remote push performed.

## Language resolution

- **Resolved language:** Go. This is the cli-tool template default
  (`LANGUAGE | Go | default`). No preset file was present and the spec does not
  declare LANGUAGE in META (per the template's POSTCONDITION, a cli-tool spec may
  not declare LANGUAGE in META). No deviation from the template default.
- **BINARY-TYPE:** `static` (template default; required for Go).
  `CGO_ENABLED=0` is set in the Makefile `build:` target, the RPM `%build`, and
  `debian/rules`.

## Module identity resolution

- **Resolved module identity:** `github.com/mge1512/zypper-declarative`.
- **Authoritative source:** source (1), the spec META `Module:` field
  (`Module: github.com/mge1512/zypper-declarative`). The language-specific
  decisions hints file (`zypper-declarative.go.decisions.hints.md`) independently
  confirms the same value, so sources (1) and (2) agree. No conflict; no halt.
  The spec-title fallback (source 4) was **not** used.

## Active MILESTONE

The spec declares MILESTONEs `0.0.0`–`0.6.0`, but **every** milestone has
`Status: pending`; none is `Status: active`. Per the prompt ("If no MILESTONE
section is present, or no milestone has `Status: active`, translate the full spec
as normal"), the full spec was translated. All five CLI verbs and all eleven
BEHAVIOR/INTERNAL behaviours were implemented. The M0 scaffold acceptance
criteria are nonetheless all satisfied by the full build (see Compile gate).

## Delivery phases (per template EXECUTION)

1. **Phase 1 — Tests first:** `independent_tests/claude-opus-4-8/` with four test
   files (32 black-box tests), written before any implementation source.
2. **Phase 2 — Core implementation:** `cmd/zypper-declarative/main.go` (entry
   point, CLI dispatch wiring only) plus `internal/` packages (behaviour), and
   `go.mod` (direct deps only).
3. **Phase 3 — Build and packaging:** `Makefile`, `zypper-declarative.spec`,
   `debian/{control,changelog,rules,copyright}`, `LICENSE`,
   `zypper-declarative.1.md` + `zypper-declarative.1`.
4. **Phase 4 — Auxiliary:** `translation_report/translation-workflow.pikchr`.
5. **Phase 5 — Documentation:** `README.md`.
6. **Phase 6 — Compile gate:** executed (see below).
7. **Phase 7 — Report:** this file, written last.

The `zypper-declarative` binary itself is a **build output** (not a named
deliverable) and is therefore not committed; `make build` and the tests'
`TestMain` build it at the project root (the `BINARY-LOCATION: project-root`
contract — `../../zypper-declarative` from the test directory).

## Source partitioning (SOURCE-PARTITIONING: modular, one-entry-one-implementation)

The entry-point module (`cmd/zypper-declarative/main.go`) contains only signal
setup and a call into `internal/cli`. Behaviour is partitioned by domain, one
internal package per concern, mirroring the spec's behaviour grouping and the
decisions hints layout:

| Package | Behaviours |
|---------|-----------|
| `internal/cli` | Verb dispatch, key=value parsing, the global CLI contract, exit-code mapping, and the five verbs (apply, diff, verify, status, describe). |
| `internal/manifest` | Data model TYPES; JSON/YAML edges; `resolve-format`; `load-desired-manifest`; canonical-model hashing; dump parsing. |
| `internal/state` | `describe-actual-state` — the single live-state reader (rpmdb, repos.d, systemd, /etc walk, full scan). |
| `internal/diff` | `compute-intent-diff`, `compute-drift` (pure, no I/O). |
| `internal/converge` | `converge-packages`, `converge-files`, `converge-units`. |
| `internal/txn` | `acquire-transaction-context` + the auto/external/internal binding. |
| `internal/record` | `load-applied-record`, `write-applied-record`. |
| `internal/diag` | The `Diagnostic` value (severity, domain, message). |
| `internal/meta` | Embedded spec SHA256 and version. |

## STEPS ordering per BEHAVIOR

Each verb implements its spec STEPS in declared order:

- **apply** (`internal/cli/verbs.go runApply`): (1) load desired → (2) load
  applied → (3) intent diff → (4) if empty, describe-actual-state + compute-drift
  and "nothing to do" → (5) acquire-transaction-context → (6) converge-packages
  → (7) converge-files → (8) converge-units → (9) write-applied-record → (10)
  post-converge describe + compute-drift → (11) summary + exit 0. Each failing
  step discards (returns without committing) and exits 1, except read/format and
  transaction-availability failures which exit 2.
- **diff** (`runDiff`): (1) load desired → (2) load applied → (3) intent diff →
  (4) actual state (supplied dump or live) → compute-drift → (5) print plan,
  exit 0.
- **verify** (`runVerify`): (1) determine reference (manifest-path else applied
  record; absent applied record → exit 2) → (2) actual state (state-path else
  live with scope) → (3) compute-drift → (4) empty → "system matches
  declaration"/0, else one diagnostic per item to stderr/1.
- **status** (`runStatus`): (1) reject unrecognised argument → (2) load applied
  (none → "no declaration applied"/0) → (3) print hash/format_version/generation/
  created_at/package count → (4) drift summary line.
- **describe** (`runDescribe`): (1) reject unknown arg/format → (2)
  describe-actual-state with on_unreadable/scope → (3) resolve-format(out) → (4)
  serialise → (5) write to out or stdout.

Internal behaviours return `*diag.Diagnostic` to their caller; exit-code mapping
lives **only** in `internal/cli`, as the spec requires.

## INTERFACES test doubles

The spec's `## INTERFACES` section describes abstract external systems (package
manager, snapshot/filesystem, init system, transaction mechanism, optional
external state producer), not a set of named INTERFACE production/double types
to implement. The one programmatic seam introduced is `state.CommandRunner` (with
the production `state.OSCommandRunner`), which lets external-command-driven code
(rpm, systemctl, zypper) be exercised without a live host. No production
implementation is used by the independent tests — they are black-box and invoke
the built binary only, so no test double is needed in the test layer.

## TYPE-BINDINGS / GENERATED-FILE-BINDINGS

The cli-tool template contains no `## TYPE-BINDINGS` and no
`## GENERATED-FILE-BINDINGS` section, so neither was applied. Logical TYPES were
mapped to idiomatic Go structs with explicit `json:` tags using underscore_style
keys (per the Go hints file). The `ScopeWrapper<T>` idiom is realised as a
concrete struct per scope (`{_attributes, _elements}` with `json` tags). Optional
declarable scopes are pointers so an absent scope (nil) is distinguished from a
present-but-empty scope.

## Constraint: supported / forbidden BEHAVIORs

All BEHAVIOR and BEHAVIOR/INTERNAL sections in the spec carry
`Constraint: required`; all were implemented unconditionally. No BEHAVIOR is
`supported` or `forbidden`, so no behaviour was conditionally activated or
suppressed.

## COMPONENT → filename mapping

The spec has no DELIVERABLES section with COMPONENT entries; deliverable
filenames were taken from the cli-tool template's DELIVERABLES table and the
per-language source-layout table (Go: `cmd/<n>/main.go` entry point,
`internal/<n>/`-style packages, `go.mod`).

## Output formats produced

Required OUTPUT-FORMATs were all produced: `source`, `public-api`, `build`
(Makefile), `docs` (README.md), `man` (`.1.md` + `.1`), `license` (LICENSE),
`RPM` (`zypper-declarative.spec`), `DEB` (`debian/`), `report`, and `spec-hash`
(embedded). The `supported` formats `OCI`, `PKG`, and `binary` were **not**
produced because no resolved preset activates them (no preset files present;
PLATFORM resolves to Linux only, so PKG is not required).

## Parsing approach

- **CLI:** hand-written `key=value` parser (`internal/cli`). A token containing
  `=` is always an option (wherever it appears relative to the verb — the spec's
  invocation examples place options after the verb); any other token is a bare
  word. Unknown option keys, unknown option values, and unrecognised verb
  positionals produce a `domain=invocation` diagnostic to stderr and exit 2.
  `version`/`help` are bare-word global commands; `--version`/`--help`/`-h` are
  tolerated aliases. Bare invocation prints usage to stdout and exits 0.
- **JSON:** `encoding/json` with `DisallowUnknownFields` on load of a desired
  manifest and a state dump, so schema-foreign keys are rejected.
- **YAML (safe profile):** `gopkg.in/yaml.v3` (a non-code-executing loader). The
  loader decodes a single document into a `yaml.Node`, then the implementation
  (`internal/manifest/yaml.go`):
  - rejects a **second document** (single-document streams only);
  - rejects any **anchor** or **alias** node (defends against alias-expansion
    denial of service — aliases are disabled, not merely bounded);
  - rejects any explicit **non-core/custom tag** (executable or arbitrary tags);
  - converts the safe node to a generic value, marshals it to JSON, and
    re-decodes that JSON with `DisallowUnknownFields`, so **JSON typing** governs
    (values such as `NO` or `1.10` are not YAML-coerced — explicit typing per the
    schema).
  This route demonstrably meets every safe-profile constraint in the spec and the
  decisions hints. A JSON dump is still accepted as YAML input (JSON is valid
  YAML 1.2).

## Signal handling

`main()` installs a `signal.Notify` handler for `SIGTERM` and `SIGINT` and exits
cleanly (exit 0) on receipt, before any partial output. An interrupted `apply`
discards the transaction by simply not committing/sealing it: convergence
mutates only the transaction-context root, and sealing/activation occurs only at
the final step, so an interrupt leaves no new snapshot as the default boot
target. (Live snapshot opening/sealing is delegated to the platform transaction
mechanism and is exercised on a live host.)

## Dependency versions

- `gopkg.in/yaml.v3 v3.0.1` — the only direct third-party dependency, resolved
  via `go mod tidy` and vendored with `go mod vendor`. This version is the
  current stable release and was available in the local module cache; it is not
  fabricated. The `go.sum` lock file was produced by the resolver.
- The spec's DEPENDENCIES section notes that bindings to libzypp, snapper/btrfs,
  and systemd require verified version strings *if a binding library is used*.
  Per the decisions hints (`exec`-based integration to keep `CGO_ENABLED=0`),
  **no** cgo binding to libzypp/snapper/systemd is taken; these systems are
  driven through their command-line interfaces (`rpm`, `zypper`, `systemctl`) and
  repos.d files are read directly. There is therefore no binding dependency
  version to verify. This is flagged here as the deliberate integration choice.

## Compile gate (Phase 6)

Executed in `/tmp/pcd-output` with `GOPATH`/`GOCACHE` under `$HOME` and no root.

| Step | Command | Result |
|------|---------|--------|
| 1 — dependency resolution | `go mod tidy`; `go mod vendor` | pass (go.sum + vendor/ written) |
| 2 — compilation | `go build ./...` | pass |
| static analysis | `go vet ./...` | pass |
| formatting | `gofmt -l internal cmd independent_tests` | pass (no files listed) |
| binary build | `CGO_ENABLED=0 go build -o zypper-declarative ./cmd/zypper-declarative` | pass (`file` reports "statically linked") |
| 3 — translator test run | `go test ./independent_tests/claude-opus-4-8/...` | pass (32/32) |
| 4 — test-author test run | n/a (single-LLM) | not applicable |

M0 scaffold acceptance criteria (all pass):

- `zypper-declarative version` → `zypper-declarative 0.6.2 spec:f2cc...` (exit 0)
- `zypper-declarative help` → usage (exit 0)
- `zypper-declarative --version` → identical to bare-word version (exit 0)
- `zypper-declarative format=bad_value` → exit 2

## Test results — translator suite

All 32 tests in `independent_tests/claude-opus-4-8/` pass.

| Test | Result |
|------|--------|
| TestVersionVerbBareWord | PASS |
| TestVersionFlagAlias | PASS |
| TestHelpVerbBareWord | PASS |
| TestHelpFlagAliases | PASS |
| TestBareInvocationShowsHelp | PASS |
| TestUnknownVerbRejected | PASS |
| TestDescribeUnknownFormat | PASS |
| TestBadFormatValueExits2 | PASS |
| TestStatusUnknownArgument | PASS |
| TestStatusNoDeclaration | PASS |
| TestApplyManifestUnreadable | PASS |
| TestDiffManifestUnreadable | PASS |
| TestApplyManifestInvalid | PASS |
| TestVerifyMalformedStateDump | PASS |
| TestVerifyNoAppliedRecord | PASS |
| TestVerifyOfflineClean | PASS |
| TestVerifyOfflineServiceDrift | PASS |
| TestVerifyOfflineFileDrift | PASS |
| TestDiffOfflineTwoFiles | PASS |
| TestDiffPrintsPlan | PASS |
| TestYAMLManifestAccepted | PASS |
| TestVerifyStatePathExtensionYAML | PASS |
| TestYAMLUnsafeRejected | PASS |
| TestApplyRejectsFullDescribeDump | PASS |
| TestYAMLFormatIdentityStable | PASS |
| TestDescribeRepositoriesFromReposd | PASS |
| TestDescribeOmitsGenuinelyEmptyScope | PASS |
| TestDescribeOutExtensionJSON | PASS |
| TestDescribeOutExtensionYAML | PASS |
| TestDescribeFormatOverridesExtension | PASS |
| TestDescribeOutputUnwritable | PASS |
| TestDescribeTraversesEtcSubdirectories | PASS |

## Test results — test-author suite

Not present (single-LLM run). No test-author cross-check suite was supplied.

## Test Refinements

Two implementation defects were found by the first test run and fixed in the
**implementation** (no test was edited). Both rows are `code fixed`.

| Test | Result before | Action | Rationale |
|------|---------------|--------|-----------|
| TestApplyRejectsFullDescribeDump | failed (exit 2, "unrecognised argument") | code fixed | The key=value parser stopped consuming options once the verb (a bare word) was seen, so options *after* the verb were rejected as positionals. The spec's invocation examples (e.g. `apply manifest-path=…`) place options after the verb; the parser now treats any `key=value` token as an option regardless of position. |
| TestYAMLFormatIdentityStable | failed (exit 2) | code fixed | Same parsing defect: `verify manifest-path=… state-path=…` had its options rejected. Fixed by the same parser change. |
| (all other 30 tests) | passed | none | — |

## Per-example confidence

Confidence definitions per the prompt. **High** requires Tests-First-Compliance
`yes` (satisfied) AND a named passing test that needs no live external service.
Many spec EXAMPLEs require a live host (snapshot transaction, populated rpmdb,
privileged enablement) and so cannot be black-box verified during translation;
those are **Low** (code review / reasoning only) and their unverified claims are
listed explicitly.

| EXAMPLE | Confidence | Verification method | Unverified claims |
|---------|-----------|---------------------|-------------------|
| version_verb_bare_word | High | TestVersionVerbBareWord | — |
| version_flag_alias | High | TestVersionFlagAlias | — |
| help_verb_bare_word | High | TestHelpVerbBareWord | — |
| bare_invocation_shows_help | High | TestBareInvocationShowsHelp | — |
| unknown_verb_rejected | High | TestUnknownVerbRejected | — |
| describe_unknown_format | High | TestDescribeUnknownFormat | — |
| status_unknown_argument | High | TestStatusUnknownArgument | — |
| status_no_declaration | High | TestStatusNoDeclaration | — |
| apply_manifest_unreadable | High | TestApplyManifestUnreadable | — |
| diff_manifest_unreadable | High | TestDiffManifestUnreadable | — |
| apply_manifest_invalid | High | TestApplyManifestInvalid | — |
| verify_malformed_state_dump | High | TestVerifyMalformedStateDump | — |
| verify_no_applied_record | High | TestVerifyNoAppliedRecord | — |
| verify_offline_manifest_and_state | High | TestVerifyOfflineClean | — |
| verify_offline_no_applied_record_ok | High | TestVerifyOfflineClean | — |
| verify_against_external_state_dump | High | TestVerifyOfflineServiceDrift | — |
| verify_detects_drift | High | TestVerifyOfflineFileDrift | — |
| diff_offline_two_files | High | TestDiffOfflineTwoFiles | — |
| diff_prints_plan | High | TestDiffPrintsPlan, TestDiffOfflineTwoFiles | — |
| intent_diff_yields_deletion | High | TestDiffOfflineTwoFiles (files-to-delete) | — |
| yaml_manifest_accepted | High | TestYAMLManifestAccepted | — |
| verify_state_path_extension_yaml | High | TestVerifyStatePathExtensionYAML | — |
| yaml_unsafe_rejected | High | TestYAMLUnsafeRejected (multi-doc) | other unsafe features (custom tags, unbounded aliases) verified by code review only |
| yaml_format_identity_stable | High | TestYAMLFormatIdentityStable | desired_sha256 byte-equality verified indirectly (identical verify verdict); direct hash equality is code-review |
| apply_rejects_full_describe_dump | High | TestApplyRejectsFullDescribeDump | — |
| describe_repositories_from_reposd | High | TestDescribeRepositoriesFromReposd | — |
| describe_omits_genuinely_empty_scope | High | TestDescribeOmitsGenuinelyEmptyScope | — |
| describe_out_extension_json | High | TestDescribeOutExtensionJSON | — |
| describe_out_extension_yaml | High | TestDescribeOutExtensionYAML | — |
| describe_format_overrides_extension | High | TestDescribeFormatOverridesExtension | — |
| describe_output_unwritable | High | TestDescribeOutputUnwritable | — |
| describe_traverses_etc_subdirectories | High | TestDescribeTraversesEtcSubdirectories | — |
| describe_records_symlink_verbatim | Low | code review (`internal/state` Readlink, verbatim target) | not black-box tested (symlink fixture under a synthetic root depends on owner classification; recorded only on changed/unpackaged entries) |
| describe_skips_special_file | Low | code review (lstat classification skips non-regular/non-symlink) | mkfifo fixture not constructed in tests |
| drift_type_transition_is_modified | Low | code review (`compute-drift` type-as-identity) | not covered by a named test; offline drift tests cover sha256 and unit drift only |
| drift_ignores_unmanaged_packaged_file | Low | code review (`files_extra` requires package_name=="") | not covered by a named test |
| describe_actual_state_omits_pristine | Low | code review (only changed/unpackaged /etc entries emitted; pristine skipped via owner+digest comparison reserved for live host) | requires a populated rpmdb; not verifiable offline |
| describe_config_files_bounded_to_etc | Low | code review (walk rooted at `<root>/etc`; no read outside /etc) | requires a large installed base; not verifiable offline |
| describe_verify_differences_not_unreadable | Low | code review (verifier non-zero treated as changed-file result, not unreadable) | requires a live rpm/verification command |
| verify_default_scope_ignores_usr | Low | code review (scope=etc never scans outside /etc) | requires a live host |
| verify_scope_full_detects_unmanaged_addition | Low | code review (`readFullScan` emits unmanaged_files; compute-drift surfaces it) | requires a live host / populated rpmdb |
| verify_scope_full_detects_modified_package_file | Low | code review | managed_files_modified baseline comparison reserved for live host (rpm -V); see "Rules not implemented exactly" |
| describe_scope_full_emits_observational_scopes | Low | code review (scope=full assembles the two observational scopes) | requires a live host |
| describe_scope_full_boot_generated_files_unmanaged | Low | code review (/boot scanned under full; keep-list honoured) | requires a live host |
| lock_is_fully_resolved_packages_scope | Low | code review (`converge-packages` queries rpmdb for the lock; `write-applied-record` stores it) | requires a live host / zypper / rpmdb |
| apply_no_op_when_converged | Low | code review (apply step 4 "nothing to do") | requires a live host |
| apply_writes_and_deletes_etc_file | Low | code review (`converge-files`) | requires a live snapshot transaction |
| apply_absent_scope_unmanaged | Low | code review (absent scope → no diff entries) | requires a live host |
| apply_transaction_unavailable | Low | code review (`acquire-transaction-context` external/internal failure → domain=transaction, exit 2) | requires a live transaction mechanism |
| apply_package_failure_rolls_back | Low | code review (converge-packages failure → exit 1, no commit) | requires a live host |
| describe_emits_manifest | Low | code review (describe assembles a format_version 1 Manifest) | populated rpmdb/services not present under a synthetic root |
| describe_bootstraps_desired_manifest | Low | code review (describe output is a valid desired manifest) | requires a live host |
| verify_clean | Low | code review (live verify path mirrors the offline path, which is tested) | requires a live host; offline equivalent is High (TestVerifyOfflineClean) |
| status_reports_generation | Low | code review (status prints hash/generation/count) | requires an applied record + live drift read |
| idempotent_second_apply | Low | code review (empty intent + empty drift → no new generation) | requires a live host |

## Rules that could not be implemented exactly as written, and why

These are reserved by the spec itself for the live-host milestone and are
implemented as conservative, no-op-safe placeholders that never silently corrupt
state:

- **converge-files symlink/dir convergence and type-transition handling:** the
  spec explicitly defers symlink creation/update/removal and type-transition
  handling ("Reserved for a later version"). `converge-files` writes/deletes
  regular files only and skips non-`file` write records. The read/drift side
  (classification, verbatim symlink target, type-as-identity in `compute-drift`)
  is fully implemented as the spec requires.
- **changed_managed_files (full scan) baseline comparison:** detecting *modified*
  packaged files outside /etc requires comparing on-disk digests to the
  rpm-recorded baseline (`rpm -V` scoped to the trees) on a live host. The full
  scan implements **unmanaged_files** (additions, by subtracting the rpmdb-owned
  path set) fully; the `changed_managed_files` scope is assembled but its element
  population is reserved for the live milestone (it remains empty under a
  synthetic root). This is documented rather than fabricated.
- **acquire-transaction-context internal open / seal / activation / snapper
  userdata stamp:** opening and sealing a real btrfs/snapper snapshot and writing
  snapper userdata are platform operations performed only inside a real snapshot
  transaction. `internal/txn` resolves auto/external/internal and detects a
  surrounding transaction; the internal-open path reports the mechanism
  unavailable (domain=transaction, exit 2) when no writable new-generation root
  is present, so the running system is never mutated outside a transaction.
- **signature verification:** `load-desired-manifest` honours
  `signature-verification`; when on, a missing/absent detached signature is a
  manifest error. Cryptographic verification against the platform keyring is
  integrated in the live-apply milestone (a present `<path>.sig` is accepted in
  this version). Documented here so the maintainer wires the keyring before
  enabling signature checking in production.

## Specification ambiguities encountered

- **`status` drift summary on a host with no applied record but live drift:** the
  spec routes `status` to "no declaration applied" / exit 0 when no record
  exists, so the drift line is only printed when a record is present. Implemented
  accordingly.
- **`applied-root` for the post-converge verification in `apply`:** the spec says
  step 10 reads the *context root*. Implemented as reading `ctx.Root` (not
  `applied-root`), since the verification is of the converged tree, not the prior
  generation.
- **package-owner determination for the bounded /etc reader:** the spec requires
  consulting package metadata only for the enumerated /etc entries and emitting
  only changed-or-unpackaged files. Under a synthetic root with no rpmdb, no
  entry has an owner and digests cannot be baseline-compared, so describe emits
  all readable /etc entries (the conservative interpretation: nothing is wrongly
  suppressed). On a live host the rpmdb owner/digest comparison applies. This is
  the most conservative interpretation that keeps the read deterministic and
  bounded to /etc.

## Public API Surface

The exported symbols of each implementation module. The next translation of this
spec at Version 0.6.2 must preserve these (additions allowed; no removals or
incompatible renames without a spec Version increment).

### Module `internal/meta`
```
const ProgramName string = "zypper-declarative"
const Version string = "0.6.2"
const SpecSHA256 string
func VersionLine() string
```

### Module `internal/diag`
```
type Severity string
const SeverityError, SeverityWarning Severity
type Domain string
const DomainPackages, DomainRepositories, DomainServices, DomainFiles, DomainManifest, DomainTransaction, DomainInvocation Domain
type Diagnostic struct { Severity Severity; Domain Domain; Message string }
func (d *Diagnostic) Error() string
func New(domain Domain, format string, args ...interface{}) *Diagnostic
func Warn(domain Domain, format string, args ...interface{}) *Diagnostic
```

### Module `internal/manifest`
```
const PackageSystemRPM, RepositorySystemZypp, InitSystemSystemd string
type PackageRecord struct { Name, Version, Release, Arch string }
type PackagesScope struct { Attributes map[string]interface{}; Elements []PackageRecord }
type RepositoryRecord struct { Alias, Name, URL, Type string; Enabled, GPGCheck, Autorefresh bool; Priority int }
type RepositoriesScope struct { Attributes map[string]interface{}; Elements []RepositoryRecord }
type ServiceRecord struct { Name, State string }
type ServicesScope struct { Attributes map[string]interface{}; Elements []ServiceRecord }
type ManagedFileRecord struct { Name, Type, Mode, User, Group, SHA256, Target, ContentRef, PackageName string }
type ConfigFilesScope struct { Attributes map[string]interface{}; Elements []ManagedFileRecord }
type ManagedBaselineRecord struct { Name, Type, Mode, User, Group, SHA256, Target, PackageName string; Changes []string }
type ChangedManagedFilesScope struct { Attributes map[string]interface{}; Elements []ManagedBaselineRecord }
type UnmanagedFileRecord struct { Name, Type, Mode, User, Group, SHA256, Target string }
type UnmanagedFilesScope struct { Attributes map[string]interface{}; Elements []UnmanagedFileRecord }
type ManifestMeta struct { FormatVersion int; Generator, CreatedAt, DesiredSHA256 string }
type Manifest struct { Meta ManifestMeta; Packages *PackagesScope; Repositories *RepositoriesScope; Services *ServicesScope; ConfigFiles *ConfigFilesScope; ChangedManagedFiles *ChangedManagedFilesScope; UnmanagedFiles *UnmanagedFilesScope }
type Format string
const FormatJSON, FormatYAML Format
var ErrUnknownFormat, ErrUnsafeYAML error
func ParseFormat(s string) (Format, error)
func ResolveFormat(explicit Format, path string, configDefault Format) Format
func (m *Manifest) MarshalJSONPretty() ([]byte, error)
func (m *Manifest) Serialise(f Format) ([]byte, error)
func (m *Manifest) DesiredSHA256() (string, error)
type LoadOptions struct { ExplicitFormat, DefaultFormat Format; SignatureCheck bool; Keyring string }
type LoadResult struct { Manifest *Manifest; DesiredSHA256 string }
func LoadDesiredManifest(path string, opts LoadOptions) (*LoadResult, error)
func ParseDump(path string, explicit Format, def Format) (*Manifest, error)
func EnsureString(v interface{}) (string, error)
```

### Module `internal/diff`
```
const SyncpointPath string = "/etc/etc.syncpoint"
type Diff struct { PackagesInstall, PackagesRemove []manifest.PackageRecord; ReposSet []manifest.RepositoryRecord; FilesWrite []manifest.ManagedFileRecord; FilesDelete []string; UnitsChange []manifest.ServiceRecord }
func (d *Diff) Empty() bool
type DriftReport struct { FilesModified, FilesExtra []string; UnitsDivergent []manifest.ServiceRecord; PackagesDivergent []manifest.PackageRecord; ManagedFilesModified, UnmanagedFilesPresent []string }
func (r *DriftReport) Empty() bool
func ComputeIntentDiff(desired *manifest.Manifest, applied *manifest.Manifest) *Diff
func ComputeDrift(actual *manifest.Manifest, reference *manifest.Manifest, keepList map[string]bool) *DriftReport
```

### Module `internal/state`
```
type OnUnreadable string
const OnUnreadableError, OnUnreadableWarn OnUnreadable
type Scope string
const ScopeEtc, ScopeFull Scope
type CommandRunner interface { Run(cmd string, args ...string) (string, string, error) }
type OSCommandRunner struct{}
func (r *OSCommandRunner) Run(cmd string, args ...string) (string, string, error)
type Options struct { OnUnreadable OnUnreadable; Scope Scope; KeepList map[string]bool; Runner CommandRunner }
type Result struct { Manifest *manifest.Manifest; Diagnostics []*diag.Diagnostic }
func DescribeActualState(root string, opts Options) (*Result, error)
```

### Module `internal/txn`
```
type Mode string
const ModeAuto, ModeExternal, ModeInternal Mode
func ParseMode(s string) (Mode, *diag.Diagnostic)
type Context struct { Mode Mode; Root string; OpenedHere bool }
func Acquire(mode Mode) (*Context, *diag.Diagnostic)
```

### Module `internal/converge`
```
type Options struct { ContentStore string; KeepList map[string]bool; RepoLock string; Runner state.CommandRunner }
func ConvergePackages(ctx *txn.Context, d *diff.Diff, opts Options) (*manifest.PackagesScope, *diag.Diagnostic)
func ConvergeFiles(ctx *txn.Context, d *diff.Diff, opts Options) *diag.Diagnostic
func ConvergeUnits(ctx *txn.Context, d *diff.Diff, opts Options) *diag.Diagnostic
```

### Module `internal/record`
```
var AppliedRelPath string
type LoadResult struct { Record *manifest.Manifest; Present bool }
func LoadAppliedRecord(root string) (*LoadResult, error)
func WriteAppliedRecord(ctxRoot string, desired *manifest.Manifest, desiredSHA256 string, resolved *manifest.PackagesScope) error
```

### Module `internal/cli`
```
const ExitOK, ExitError, ExitInvocation int
type IO struct { Stdout, Stderr io.Writer }
type Config struct { /* resolved CONFIG knobs and per-invocation options */ }
func Run(args []string, io IO) int
```

## Template constraints compliance

| Constraint | Required value | Status |
|------------|----------------|--------|
| LANGUAGE | Go (default) | Go ✓ |
| BINARY-TYPE | static | static (CGO_ENABLED=0) ✓ |
| SOURCE-PARTITIONING | modular, one-entry-one-implementation | entry-point `cmd/` + 9 internal packages ✓ |
| MODULE-IDENTITY | host-specified | spec META `Module:` ✓ |
| PUBLIC-API-SURFACE | recorded-in-report | this report ✓ |
| BINARY-COUNT | 1 | one binary ✓ |
| BINARY-LOCATION | project-root (`../../<n>`) | built at root; tests use `../../zypper-declarative` ✓ |
| RUNTIME-DEPS | none | static binary; drives system tools via CLI ✓ |
| CLI-ARG-STYLE | key=value (+ bare-words) | key=value parser; bare-word verbs ✓ |
| EXIT-CODE-OK / ERROR / INVOCATION | 0 / 1 / 2 | implemented ✓ |
| STREAM-DIAGNOSTICS / OUTPUT | stderr / stdout | implemented ✓ |
| SIGNAL-HANDLING | SIGTERM, SIGINT | clean exit ✓ |
| OUTPUT-FORMAT | RPM, DEB (required) | both produced ✓ |
| INSTALL-METHOD | OBS (curl forbidden) | README documents OBS only ✓ |
| PLATFORM | Linux | ExclusiveOS linux; no macOS/Windows ✓ |
| CONFIG-ENV-VARS | forbidden | no behaviour knob via env var ✓ (see note) |
| NETWORK-CALLS | forbidden | no direct network I/O; package fetch delegated (documented deviation) |
| FILE-MODIFICATION input-files | forbidden | inputs (manifests/dumps) never modified ✓ |
| IDEMPOTENT | true | apply no-ops on empty intent + empty drift ✓ |
| spec-hash | embedded | embedded in all artefacts ✓ |

**Documented deviations (per spec DEPLOYMENT):** NETWORK-CALLS — the tool makes
no direct network I/O of its own; package retrieval is delegated to the package
manager against a declared, pinned, signed repository, honouring the
supply-chain intent. Privilege — `apply` requires privilege (unlike a typical
read-only cli-tool); read-only verbs need only read access. CONFIG-ENV-VARS — no
behaviour is controlled via environment variables; the single env read
(`TRANSACTIONAL_UPDATE_NEWROOT` in `internal/txn`) is *detection of a surrounding
transaction*, not a behaviour knob, and is consistent with the spec's
`mode=external` description.
