# TRANSLATION_REPORT.md — zypper-declarative (C++)

## Provenance and identity

- **Spec-SHA256:** `aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3`
  (merged — the host spec declares no `Includes:`, so the merged hash equals the
  host hash).
- **Spec-SHA256 (host):** `aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3`
- **Included-Specs:** none.

  | Path | SHA256 |
  |------|--------|
  | *(none)* | — |

- **Decisions-Hints-SHA256:** `zypper-declarative.cpp.decisions.hints.md` `5becc2206f60b2e9658bbc896606f3c02d6b0f10d8434c69906492838c1e1035`
- **Milestones-Hints-SHA256:** `cli-tool.cpp.milestones.hints.md` `9acc4a6ab9fbcda39161aeb5cfd250715976598437d82eabb34a7d951e4a782e`
- **Template-SHA256:** `cli-tool.template.md` `c8447ba8f1e63f3605b8e671e5bf58f4df44665a5ba1ff76864d28e4570042b5`
- **Upgrade-Guidance-SHA256:** `UPGRADE-0.6.8-to-0.6.9.md` `186f8ea108f5581ef85ad1c5a74df66ee938c611e31fa492f1adda5fbd562931`
- **Style-Hints-SHA256:** `none` (no `<scope>.cpp.style.hints.md` present in the input or preset hierarchy)
- **LLM-Name:** `claude-opus-4-8`
- **Mode:** `translator`
- **Deployment template:** `cli-tool.template.md v0.3.29`

## Regeneration context

This run is a **guided regeneration** of a pre-existing C++ translation. The
caller stated the existing code was produced from an **older spec version
(v0.6.8)** and provided `UPGRADE-0.6.8-to-0.6.9.md` as the authoritative,
exact delta to apply. Per that guidance, the v0.6.8 implementation is correct
*except* for the one shipped bug described in Change 1; the code was **updated in
place**, not regenerated from scratch. The pre-existing output embedded the old
spec hash `1641bb44…` and version `0.6.8`.

Per the template's **Resume logic**, existing non-empty deliverables were
inspected and edited surgically. The actions taken match the four changes in the
upgrade document:

1. **Change 1 — BUG FIX: symlink mechanism classification (`src/actual_state.cpp`).**
   The v0.6.8 build routed every *ghost* symlink through the `update-alternatives`
   query path. On a real host this produced `error: files: cannot query
   alternatives for /etc/motd.d/cockpit` under the default `on-unreadable=error`
   and ~24 spurious `alternatives unreadable` warnings under `warn`
   (crypto-policies back-ends, motd.d, issue.d links). The fix introduces a
   mechanism classifier in front of the alternatives branch:
   - `build_alt_db(root)` scans `<root>/var/lib/alternatives/*` once, collecting
     every master and slave **link** path into a `link_to_name` map. Directory
     absence is *not* an unreadable failure; a genuine read failure on the
     directory sets `readable=false`.
   - `is_alternatives_link(path)` returns true **iff** the path is under
     `/etc/alternatives/` **or** appears in that map. Only these are resolved
     against the alternatives DB.
   - The symlink branch of `judge_entry` now: (a) for a **non-alternatives**
     symlink, judges by the normal target rule — unpackaged → emit, recorded type
     mismatch → emit, on-disk target ≠ recorded target → emit, else suppress
     (pristine) — and **never** calls `update-alternatives`; (b) for an
     **alternatives** symlink, resolves auto/best and suppresses iff the on-disk
     target equals it, emits on a manual `--set`, and emits conservatively when
     auto/best is indeterminable (a slave). `on_unreadable` applies to an
     alternatives symlink **only** when `/var/lib/alternatives` genuinely cannot
     be read.
   Verified live on the build host (unprivileged, `on-unreadable=warn`): **zero**
   `alternatives unreadable` / `cannot query alternatives` diagnostics; the
   pristine crypto-policies back-end links are suppressed (absent from
   `config_files`); under the **default error mode** describe no longer aborts on
   `/etc/motd.d/cockpit` (it now stops only on a genuinely root-only file,
   `/etc/libaudit.conf`, which is the correct error-mode behaviour and exactly
   why `init` forces warn — see Change 2).
2. **Change 2 — `init` forces `on_unreadable=warn` (`src/commands.cpp`).**
   `cmd_init`'s step-1 `describe-actual-state` call now sets
   `OnUnreadable::Warn` unconditionally, overriding the default `error` and any
   command-line value, so onboarding a real machine does not abort on a protected
   root-only file or an indeterminable source. `init` is the only verb that
   overrides the knob; `describe`/`diff`/`verify`/`apply` keep `error` as default.
   Verified live: `init out=… mode=external` got past the protected
   `/etc/libaudit.conf` read without aborting and stopped only at transaction
   acquisition (exit 2, transaction domain) — expected unprivileged.
3. **Change 3 — `packages_divergent` empty-field wildcard (CLARIFICATION).** The
   spec now makes normative the behaviour the C++ code already implemented (an
   empty identity field in a reference package element is a wildcard). Verified the
   code still matches; **no change required** (see Specification ambiguities).
4. **Change 4 — long-running examples (HARNESS note).** The `describe scope=full`
   example is O(installed packages) and takes minutes on this host. The black-box
   harness imposes **no** per-test timeout (concurrent `poll()`-based drain of
   stdout/stderr, no watchdog), so a still-running long example is never killed.
   The full suite (including `scope=full`) was run to completion and the long test
   passed. No code change; confirmed the harness does not fail long cases.

Then the spec hash and version were re-embedded everywhere: `1641bb44…` →
`aafbb315…` in every source/header/`CMakeLists.txt`/`meta.hpp.in`/test/packaging
file, and `VERSION` was bumped `0.6.8` → `0.6.9`. The binary now reports
`zypper-declarative 0.6.9 spec:aafbb315…`. New RPM/DEB changelog entries were
added for 0.6.9; the man-page version line was bumped. Test fixture `generator`
strings (informational, never compared) were updated to `0.6.9`.

No other behaviour was modified. The rest of the implementation
(`manifest.cpp`, `diff.cpp`, `transaction.cpp`, `cli.cpp`, `types.hpp`, the
repositories/services/file-baseline readers, the content store, etc.) was already
aligned with the spec and was untouched beyond the hash re-embedding.

## Language and toolchain resolution

- **Target language:** C++ (C++17). Resolved from the invocation
  (`Target language: C++`) and the cli-tool template's `LANGUAGE` row, which
  lists C++ as a `supported` alternative to the Go default. The deviation from
  the template default (Go) is explicit and caller-directed.
- **Build system:** CMake (per the C++ source-layout row and the milestones hints).
- **Compiler:** built and gated with `g++-15` (GCC 15.2). On the build host
  (SLE 15 SP7 class) the default `g++` is GCC 7, which lacks `<filesystem>`;
  per the milestones hints the side-by-side `g++-15` is selected
  (`make build CXX=g++-15`). On SLE 16.0 the default GCC 15 suffices.
- **Linking:** DYNAMIC against the distribution's shared libraries (no static
  binary, no vendoring), per the C++ decisions hints — the deliberate difference
  from the Go/Rust siblings. `BINARY-TYPE=dynamic` is valid because
  `LANGUAGE ∈ {C, C++, C#}` (template INVARIANT).

## Module identity resolution

`MODULE-IDENTITY: host-specified` applies. The spec META declares
`Module: github.com/mge1512/zypper-declarative` (authoritative source 1). For a
C++ CMake artefact the concrete module identity is the **project / binary name**,
which is derived consistently as `zypper-declarative` (the spec title, matching
the trailing path component of the META `Module:` value and the existing
`go.mod`/`Cargo.toml` identity of the sibling translations). No conflict among
sources; the spec-title fallback was **not** needed. The identity propagates to:
the CMake `project()` name, the binary name, the RPM `Name:`/`Source0:`/`URL:`,
the DEB `Source:`/`Homepage:`, the man page, and the README install commands.

## Dependency versions (verified on the build host)

Discovered via `pkg_check_modules(... IMPORTED_TARGET ...)` (pkg-config), per the
decisions hints (the CMake configs are Meson-fragile across service packs; the
`.pc` files are stable). libsnapper ships no `.pc`, so it is located with
`find_path`/`find_library` (soname differs per SP: `libsnapper5` here).

| Library | Discovery | Version on build host | devel package |
|---------|-----------|-----------------------|---------------|
| libzypp | pkg-config `libzypp` | 17.37.18 | `libzypp-devel` |
| jsoncpp | pkg-config `jsoncpp` | 1.8.4 | `jsoncpp-devel` |
| yaml-cpp | pkg-config `yaml-cpp` | 0.6.3 (`libyaml-cpp0_6`) | `libyaml-cpp` devel |
| libcrypto | pkg-config `libcrypto` | 3.2.3 (OpenSSL 3) | `libopenssl-3-devel` |
| libsnapper | `find_library` | `libsnapper.so.5` (0.8.x) | `libsnapper-devel` |

All versions are the verified ones from the decisions hints; none were
fabricated. The code restricts itself to the API surface stable across the two
service-pack versions of each library (yaml-cpp 0.6↔0.8, libsnapper 5↔7).

## Delivery mode

Filesystem (mode 1). All artefacts were written to disk under
`/tmp/pcd-output/code/cpp/`. Single-LLM run: the `independent_tests/claude-opus-4-8/`
directory is the translator's own suite (the `llm-name` from `ROLE.md`); there is
no separate test-author directory.

## Tests-First-Compliance

`no` — this is a guided **regeneration / in-place upgrade**. The translator test
suite at `independent_tests/claude-opus-4-8/` pre-existed from the prior run; the
pre-existing tests were not authored fresh before the implementation in this run
(the implementation also pre-existed). The pre-existing suite was retained, its
spec-hash/version updated, reviewed for v0.6.9 alignment, and re-run, and **three
new v0.6.9 tests** were added for the Change 1 symlink classification (these new
tests *were* written before/alongside the Change 1 code edit and exercise the new
behaviour). Per the prompt, examples whose pre-existing tests pass are recorded at
**Medium** confidence rather than High, because fresh tests-first ordering was not
re-established for the bulk of the suite in this upgrade. The structural guard (a
non-empty `independent_tests/<llm-name>/` before any implementation source is
written) is satisfied: the directory existed and contained the test files
throughout.

## Continuity-Check

Not applicable — no test-author input. The single test directory present is the
translator's own (`claude-opus-4-8`), not a distinct `<other-role-llm-name>`.

## STEPS ordering per BEHAVIOR

Implemented in spec order; verified unchanged in this regeneration:

- **describe** (`cmd_describe`): reject unknown arg/format → describe-actual-state
  → resolve-format → serialise → write/stdout. STEPS 1–5.
- **diff** (`cmd_diff`): load desired → load applied (intent only) →
  compute-intent-diff → actual state (state-path or live with `on_unreadable`,
  scope=etc) → compute-drift **against the desired manifest** → print plan, exit 0.
  STEPS 1–5.
- **verify** (`cmd_verify`): determine reference (manifest-path else applied
  record; exit 2 if neither) → actual state (state-path or live with
  `on_unreadable`/scope) → compute-drift → exit 0/1 with per-item diagnostics.
- **status** (`cmd_status`): reject unknown arg → load applied (else "no
  declaration applied", exit 0) → print fields → drift summary line.
- **apply** (`cmd_apply`): load desired → load applied → intent diff → if empty,
  drift check (live, `on_unreadable`); if empty "nothing to do" exit 0 (no txn)
  → acquire-transaction-context → converge packages/files/units →
  write-applied-record → summary. STEPS 1–11 (post-converge verify/seal are
  on-target).
- **init** (`cmd_init`): describe-actual-state on "/" (incl. content-store) →
  acquire-transaction-context → write-applied-record (converge NOTHING) → write
  adopted manifest to `out` → summary.
- INTERNAL behaviours (`resolve-format`, `load-desired-manifest`,
  `load-applied-record`, `compute-intent-diff`, `compute-drift`,
  `describe-actual-state`, `acquire-transaction-context`, `converge-*`,
  `write-applied-record`) each return errors to their caller; only the verb layer
  maps to an exit code.

## INTERFACES test doubles

The spec's INTERFACES section describes external integration systems
(libzypp/zypper, btrfs/snapper, systemd, the transaction mechanism, an optional
external state producer) rather than an in-tree INTERFACES seam with named
production/test-double implementations. The code expresses the one genuine
in-process seam — the **CommandRunner** — as an abstract base with a production
`OSCommandRunner` (fork/execvp, separate stdout/stderr capture, fixed child
`PATH`, non-zero exit returned as data not thrown) and a `FakeCommandRunner`
pattern (per the milestones hints) for unit-level doubling. The black-box test
suite uses neither double: it invokes the built binary as a subprocess and
asserts only on stdout/stderr/exit code.

## TYPE-BINDINGS / GENERATED-FILE-BINDINGS

The cli-tool template declares no `## TYPE-BINDINGS` or
`## GENERATED-FILE-BINDINGS` section, so neither applies. The data-model types
follow the spec TYPES and the C++ decisions hints (`ScopeWrapper<T>` with
`attributes`/`elements`; absent-vs-present scopes via `std::optional<Scope>`).

## Constraint: supported / forbidden BEHAVIORs

Every BEHAVIOR and BEHAVIOR/INTERNAL in the spec carries `Constraint: required`;
all are implemented. No `supported` or `forbidden` behaviour exists. The
`converge-files` symlink/type-transition convergence is explicitly **reserved**
by the spec for a later version and is correspondingly not implemented (the
record still classifies and compares these types in describe/drift).

## COMPONENT → filename mapping

The spec has no DELIVERABLES `COMPONENT:` entries. Deliverable filenames derive
from the template DELIVERABLES table and the per-language source layout:

| Logical unit | Files |
|---|---|
| entry-point (CLI dispatch only) | `src/main.cpp`, `src/cli.{hpp,cpp}` |
| data model | `src/types.hpp`, `src/diagnostic.hpp` |
| command runner (seam) | `src/command_runner.{hpp,cpp}` |
| manifest model + serialisation + load/resolve | `src/manifest.{hpp,cpp}` |
| intent diff + drift (pure) | `src/diff.{hpp,cpp}` |
| live-state reader (single reader) | `src/actual_state.{hpp,cpp}` |
| transaction + convergence | `src/transaction.{hpp,cpp}` |
| verb implementations | `src/commands.{hpp,cpp}` |
| build manifest | `CMakeLists.txt`, `Makefile` |
| RPM / DEB | `zypper-declarative.spec`, `debian/*` |
| docs / man / license | `README.md`, `zypper-declarative.1.md`+`.1`, `LICENSE` |

`SOURCE-PARTITIONING: modular` and `one-entry-one-implementation` are satisfied:
`main.cpp`/`cli.cpp` only dispatch; behaviour lives in the per-domain TUs.

## SOURCE-PARTITIONING / single-reader compliance

`describe-actual-state` (`actual_state.cpp`) is the only TU that links libzypp's
rpmdb, reads `/etc/zypp/repos.d`, queries unit state, or walks `/etc`/full-scan
trees. `compute-drift` (`diff.cpp`) performs no I/O. `resolve-format`
(`manifest.cpp`) is the single serialisation authority. This matches the spec's
`[implementation]` invariants.

## Parsing approach

- **JSON:** jsoncpp. `_attributes` always emitted as an object (empty `{}` when
  no attributes, never `null`); absent (`std::nullopt`) scopes are omitted
  entirely rather than written as `null`/`{}`.
- **YAML:** yaml-cpp under a safe profile — single document only, non-default
  tags rejected, alias/anchor expansion bounded, explicit `as<std::string>()`
  typing (no implicit coercion of `NO`/`1.10`). A YAML input requiring a disabled
  feature returns a manifest error.
- **`desired_sha256`:** SHA256 (libcrypto) of a canonical JSON serialisation of
  the parsed data model — keys sorted, compact separators, `_elements` sorted by
  identity key — so JSON and YAML expressions of the same manifest hash equally.
- **CLI:** `key=value` options parsed in any position; bare-word verbs;
  `--version`/`--help`/`-h` tolerated aliases only; no POSIX `--flag` options.

## Signal handling

`SIGTERM` and `SIGINT` are handled for a clean exit; the read-only verbs need no
cleanup, and `apply`'s transaction is discarded on interruption (the snapshot is
never sealed as the default boot target unless the full converge+verify succeeds).
The mutating snapshot/seal path is exercised only on a privileged target; the
unprivileged build-time gate covers the read-only and validation paths.

## Template constraints compliance

| Constraint | Value | Status |
|---|---|---|
| LANGUAGE | C++ (supported alternative) | met (caller-directed deviation from Go default) |
| BINARY-TYPE | dynamic | met (valid for C++; libzypp/libsnapper/etc. linked dynamically) |
| SOURCE-PARTITIONING | modular + one-entry-one-implementation | met |
| MODULE-IDENTITY | host-specified, propagated | met (spec META `Module:`; name `zypper-declarative`) |
| BINARY-COUNT | 1 | met |
| BINARY-LOCATION | project-root (`../../zypper-declarative`) | met |
| CLI-ARG-STYLE | key=value + bare-words | met |
| EXIT codes | 0 / 1 / 2 | met |
| STREAMS | stderr diagnostics, stdout output | met |
| SIGNAL-HANDLING | SIGTERM, SIGINT | met |
| OUTPUT-FORMAT | RPM, DEB (required) | met; OCI/PKG/binary (supported) not active in preset, not produced |
| INSTALL-METHOD | OBS; curl forbidden | met (no curl anywhere) |
| CONFIG-ENV-VARS | forbidden | met (only a debug trace gate, not control) |
| NETWORK-CALLS | forbidden | honored (package retrieval delegated to libzypp; documented spec deviation) |
| FILE-MODIFICATION input-files | forbidden | met (manifest never modified) |
| spec-hash | embedded everywhere | met |

## Compile gate result (Phase 6)

Executed with `g++-15` on the build host.

- **Step 1 — dependency resolution:** N/A for C++ (system libraries resolved at
  build time via pkg-config / find_library). Discovery succeeded: see versions
  above.
- **Step 2 — compilation:** `make build CXX=g++-15` → **pass** (clean build,
  `-Wall -Wextra`, no warnings). `ldd` confirms dynamic linking of
  `libzypp.so.1735`, `libsnapper.so.5`, `libjsoncpp.so.19`, `libyaml-cpp.so.0.6`,
  `libcrypto.so.3`.
- **Step 3 — translator test run:** `make test CXX=g++-15` →
  **39 tests run, 0 failures** (the `scope=full` case runs for several minutes and
  was allowed to complete; no timeout killed it, per Change 4).
- **Step 4 — test-author run:** N/A (single-LLM).
- **M0/M0.1 acceptance gates** re-verified: bare-word `version`/`help`,
  `--version` alias, `format=bad_value → exit 2`, bare invocation `→ exit 0` with
  usage, `version` contains `spec:`, `status → "no declaration applied"`.

## Test results — translator suite (independent_tests/claude-opus-4-8)

All 39 black-box tests pass:

```
test_bare_invocation_shows_help                 ok
test_version_verb_bare_word                     ok
test_version_flag_alias_matches                 ok
test_help_verb_and_aliases                      ok
test_unknown_verb_rejected                      ok
test_unknown_format_value_rejected              ok
test_bad_format_value_exit2                     ok
test_status_unknown_argument                    ok
test_describe_emits_json_manifest               ok
test_describe_attributes_never_null             ok
test_describe_format_yaml_stdout                ok
test_describe_out_extension_yaml                ok
test_describe_out_extension_json                ok
test_describe_format_overrides_extension        ok
test_describe_output_unwritable                 ok
test_describe_without_content_store_is_readonly ok
test_describe_output_is_valid_desired_manifest  ok
test_describe_then_diff_empty_drift             ok
test_describe_scope_full_has_observational_scopes ok
test_describe_populates_content_store           ok
test_diff_manifest_unreadable                   ok
test_diff_offline_two_files_plan                ok
test_diff_offline_exit_zero                     ok
test_diff_yaml_manifest_accepted                ok
test_verify_offline_matches                     ok
test_verify_offline_units_drift                 ok
test_verify_offline_files_drift                 ok
test_verify_type_transition_drift               ok
test_verify_malformed_state_dump                ok
test_verify_offline_no_applied_record_ok        ok
test_verify_state_path_extension_yaml           ok
test_status_no_declaration                      ok
test_apply_manifest_unreadable                  ok
test_apply_manifest_invalid                     ok
test_apply_rejects_full_describe_dump           ok
test_apply_transaction_unavailable_external     ok
test_nonalt_symlink_no_alternatives_query_default_error ok
test_nonalt_symlink_emitted_verbatim            ok
test_alternatives_dir_symlink_classified_as_alt ok
```

## Test results — test-author suite

Not present (single-LLM run).

## Test Refinements

| Test | Result before | Action | Rationale |
|------|---------------|--------|-----------|
| *(all 36 pre-existing)* | passed | none | After the v0.6.9 symlink-classification and `init`-warn fixes plus the spec-hash/version re-embed, every pre-existing test passed unmodified. No assertion changed; only fixture `generator` strings were updated `0.6.8`→`0.6.9` (informational, never compared). |
| test_nonalt_symlink_no_alternatives_query_default_error | n/a (new) | added | New test for Change 1: a non-alternatives symlink (crypto-policies-style, into `/usr/share`) under a synthetic root must not produce any `alternatives` diagnostic. |
| test_nonalt_symlink_emitted_verbatim | n/a (new) | added | New test for Change 1: the unpackaged non-alternatives symlink is emitted as type `link` with its verbatim target, not routed through the alternatives path. |
| test_alternatives_dir_symlink_classified_as_alt | n/a (new) | added | New test for Change 1: a symlink under `/etc/alternatives/` IS classified as an alternatives link and (auto/best indeterminable on the synthetic root) emitted conservatively. |

## Specification ambiguities and deviations

- **`compute-drift` `packages_divergent` (spec step 4) — now NORMATIVE in v0.6.9.**
   The v0.6.9 spec (Change 3) makes explicit what was previously an undocumented
   judgment call inherited from v0.6.7: an **empty identity field in a reference
   package element is a WILDCARD** matching any actual value, so desired
   `{name: nginx, version: ""}` matches the resolved actual
   `{name: nginx, version: "1.27.4", …}` and is not divergent; only non-empty
   reference fields must match, and only the reference side wildcards. The C++ code
   already implemented exactly this (directional comparison, reference-side
   wildcard). Verified the code still matches the now-normative wording; **no
   change required**. This is no longer a spec ambiguity — recorded here for the
   audit trail of the v0.6.8→v0.6.9 upgrade.
- **NETWORK-CALLS deviation (from the spec's own DEPLOYMENT).** The tool makes no
  direct network I/O; package retrieval is delegated to libzypp against pinned,
  signed repositories. The supply-chain intent of the constraint is honored;
  documented per the spec's template-deviation note.
- **Privileged / mutating paths** (real snapshot creation, the full apply/seal
  transaction) are not exercised at translation time (they require root and a live
  btrfs/snapper transaction). Their validation/failure paths (unreadable manifest,
  invalid manifest, observational-scope rejection, transaction unavailable) ARE
  asserted unprivileged. This follows the decisions/milestones hints' explicit
  instruction never to invoke `sudo` in a build step.

## Rules not implemented exactly as written

- `converge-files` symlink convergence and apply-time type-transition handling
  are **deferred by the spec itself** to a later version; the implementation
  writes/deletes regular files only, consistent with the spec's BEHAVIOR note.

## Active MILESTONE

All `## MILESTONE:` sections (0.0.0 through 0.6.0) have `Status: pending`; none is
`active`. Per the prompt, the **full spec was translated** (no scaffold-only or
single-milestone restriction applied). The milestone acceptance criteria were
nonetheless used as additional gates and all pass (see Compile gate).

## Public API Surface

The exported (header-declared) symbols of the implementation modules. Stable
across translations of spec v0.6.9. The upgrade from v0.6.8 added **no** new
exported symbol and removed/renamed none: the symlink-classification helpers
(`build_alt_db`, `is_alternatives_link`, `AltDb`) and the unchanged
`alternatives_best` signature change are file-local (anonymous-namespace) symbols
in `actual_state.cpp`, not part of the public header surface. The surface below is
identical to the v0.6.8 surface.

### module: types (`src/types.hpp`)
- `template <class T> struct ScopeWrapper { std::map<std::string,std::string> attributes; std::vector<T> elements; }`
- `struct ManifestMeta { int format_version; std::string generator; std::string created_at; std::string desired_sha256; }`
- `struct PackageRecord { std::string name, version, release, arch; }`
- `struct RepositoryRecord { std::string alias, name, url, type; bool enabled, gpgcheck, autorefresh; long priority; }`
- `struct ServiceRecord { std::string name, state; }`
- `struct ManagedFileRecord { std::string name, type, mode, user, group, sha256, target, content_ref, package_name; }`
- `struct ManagedBaselineRecord { std::string name, type, mode, user, group, sha256, target, package_name; std::vector<std::string> changes; }`
- `struct UnmanagedFileRecord { std::string name, type, mode, user, group, sha256, target; }`
- `using PackagesScope / RepositoriesScope / ServicesScope / ConfigFilesScope / ChangedManagedFilesScope / UnmanagedFilesScope`
- `struct Manifest { ManifestMeta meta; std::optional<…> packages, repositories, services, config_files, changed_managed_files, unmanaged_files; }`
- `struct Diff { … }`, `struct DriftReport { … bool empty() const; size_t count() const; }`
- `enum class TransactionMode { Auto, External, Internal }`
- `struct TransactionContext { TransactionMode mode; std::string root; bool opened_here; }`
- `enum class ManifestFormat { Json, Yaml }`

### module: diagnostic (`src/diagnostic.hpp`)
- `struct Diagnostic { Severity severity; std::string domain, message; std::string format() const; }`
- `template <class T> class Result { … bool ok() const; const T& value() const; const Diagnostic& error() const; }`
- `Diagnostic err(std::string domain, std::string message)`
- `Diagnostic warn(std::string domain, std::string message)`

### module: command_runner (`src/command_runner.hpp`)
- `struct CommandResult { std::string out; std::string err; int code; bool spawn_failed; }`
- `class CommandRunner { virtual CommandResult run(const std::string& cmd, const std::vector<std::string>& args) const = 0; }`
- `class OSCommandRunner : public CommandRunner { CommandResult run(...) const override; }`

### module: manifest (`src/manifest.hpp`)
- `ManifestFormat resolve_format(const std::optional<ManifestFormat>& explicit_fmt, const std::optional<std::string>& path, ManifestFormat default_fmt)`
- `std::optional<ManifestFormat> parse_format(const std::string& s)`
- `std::string serialise_json(const Manifest& m, bool pretty)`
- `std::string serialise_yaml(const Manifest& m)`
- `std::string canonical_sha256(const Manifest& m)`
- `std::string sha256_hex(const std::string& bytes)`
- `Result<LoadedManifest> load_desired_manifest(const std::string& manifest_path, const std::optional<ManifestFormat>& explicit_fmt, ManifestFormat default_fmt)`
- `Result<Manifest> load_state_dump(const std::string& state_path, const std::optional<ManifestFormat>& explicit_fmt, ManifestFormat default_fmt)`
- `Result<AppliedLoad> load_applied_record(const std::string& root)`
- `struct LoadedManifest { Manifest manifest; std::string desired_sha256; }`
- `struct AppliedLoad { Manifest record; bool present; }`

### module: diff (`src/diff.hpp`)
- `Diff compute_intent_diff(const Manifest& desired, const Manifest& applied)`
- `DriftReport compute_drift(const Manifest& actual, const Manifest& reference, const KeepList& keep_list)`
- `using KeepList = std::set<std::string>`

### module: actual_state (`src/actual_state.hpp`)
- `enum class OnUnreadable { Error, Warn }`, `enum class ScanScope { Etc, Full }`
- `struct DescribeOptions { std::string root; OnUnreadable on_unreadable; ScanScope scope; std::optional<std::string> content_store; KeepList keep_list; }`
- `struct DescribeResult { Manifest manifest; std::vector<Diagnostic> diagnostics; }`
- `Result<DescribeResult> describe_actual_state(const DescribeOptions& opts, const CommandRunner& runner)`

### module: transaction (`src/transaction.hpp`)
- `Result<TransactionContext> acquire_transaction_context(TransactionMode mode, const CommandRunner& runner)`
- `Result<PackagesScope> converge_packages(const TransactionContext& ctx, const Diff& diff, const CommandRunner& runner)`
- `std::optional<Diagnostic> converge_files(const TransactionContext& ctx, const Diff& diff, const std::optional<std::string>& content_store)`
- `std::optional<Diagnostic> converge_units(const TransactionContext& ctx, const Diff& diff, const CommandRunner& runner)`
- `std::optional<Diagnostic> write_applied_record(const TransactionContext& ctx, const Manifest& desired, const std::string& desired_sha256, const PackagesScope& resolved)`

### module: commands (`src/commands.hpp`)
- `struct Invocation { std::string verb; std::map<std::string,std::string> options; }`
- `int cmd_apply(const Invocation&, const CommandRunner&)`
- `int cmd_diff(const Invocation&, const CommandRunner&)`
- `int cmd_verify(const Invocation&, const CommandRunner&)`
- `int cmd_status(const Invocation&, const CommandRunner&)`
- `int cmd_describe(const Invocation&, const CommandRunner&)`
- `int cmd_init(const Invocation&, const CommandRunner&)`

### module: cli (`src/cli.hpp`)
- `int run_cli(const std::vector<std::string>& args, const CommandRunner& runner)`
- `std::string usage_text()`
- `std::string version_text()`

### module: meta (`src/meta.hpp`, configured)
- `constexpr const char* kProgramName`, `kVersion`, `kSpecSha256`

## Per-EXAMPLE confidence

Confidence is **Medium** for every tested example because Tests-First-Compliance
is `no` for this regeneration (see above), per the prompt's demotion rule.
Examples with no unprivileged black-box test are **Low** (code-review only).

| EXAMPLE | Confidence | Verification method | Unverified claims |
|---|---|---|---|
| apply_no_op_when_converged | Low | reasoning; needs live state | no-op path needs live converged system |
| apply_writes_and_deletes_etc_file | Low | code review | requires root + live snapshot |
| apply_absent_scope_unmanaged | Low | code review | requires root |
| apply_manifest_invalid | Medium | `test_apply_manifest_invalid` | — |
| apply_manifest_unreadable | Medium | `test_apply_manifest_unreadable` | — |
| apply_transaction_unavailable | Medium | `test_apply_transaction_unavailable_external` | — |
| apply_package_failure_rolls_back | Low | code review | requires root + repos |
| diff_prints_plan | Medium | `test_diff_offline_two_files_plan` | — |
| diff_manifest_unreadable | Medium | `test_diff_manifest_unreadable` | — |
| diff_unchanged_machine_no_drift | Medium | `test_describe_then_diff_empty_drift` (offline form) | live-read form needs the host |
| init_onboards_machine | Low | code review | requires root + live snapshot |
| init_then_apply_is_noop | Low | code review | requires root |
| describe_emits_manifest | Medium | `test_describe_emits_json_manifest` (live, on-unreadable=warn) | — |
| describe_output_unwritable | Medium | `test_describe_output_unwritable` | — |
| describe_bootstraps_desired_manifest | Medium | `test_describe_output_is_valid_desired_manifest` | — |
| verify_clean | Low | code review (live) | needs an applied record + live match |
| verify_against_external_state_dump | Medium | `test_verify_offline_units_drift` | — |
| verify_malformed_state_dump | Medium | `test_verify_malformed_state_dump` | — |
| verify_detects_drift | Medium | `test_verify_offline_files_drift` | — |
| verify_no_applied_record | Medium | covered by offline-reference + status tests | live no-record exit 2 path code-reviewed |
| status_reports_generation | Low | code review | needs a real applied record |
| status_no_declaration | Medium | `test_status_no_declaration` | — |
| status_unknown_argument | Medium | `test_status_unknown_argument` | — |
| intent_diff_yields_deletion | Medium | `test_diff_offline_two_files_plan` (drives compute-intent-diff) | — |
| drift_ignores_unmanaged_packaged_file | Medium | exercised via verify drift fixtures | direct package_name-non-empty fixture code-reviewed |
| describe_actual_state_omits_pristine | Medium | `test_describe_emits_json_manifest` + libzypp baseline | exact pristine suppression on a known path code-reviewed |
| describe_traverses_etc_subdirectories | Medium | covered by live describe (walks /etc) | — |
| describe_records_symlink_verbatim | Medium | `test_nonalt_symlink_emitted_verbatim` (synthetic root, verbatim target preserved) | — |
| describe_skips_special_file | Low | code review | needs a constructed fifo/socket |
| drift_type_transition_is_modified | Medium | `test_verify_type_transition_drift` | — |
| describe_config_files_bounded_to_etc | Medium | live describe; scope=etc never scans /usr | — |
| describe_suppresses_package_pristine_etc_file | Medium | live describe + libzypp | exact P/Q/local example code-reviewed |
| describe_symlink_and_target_judged_independently | Low | code review | needs constructed independent pair |
| describe_pristine_distro_symlink_suppressed | Low | code review | needs the specific distro link |
| describe_type_mismatch_emitted | Low | code review | needs the pam fixture |
| describe_ghost_with_content_emitted | Low | code review | needs the pam-config ghost fixture |
| describe_default_alternative_symlink_suppressed | Low | code review | needs /etc/alternatives + populated alt DB; classification path covered by `test_alternatives_dir_symlink_classified_as_alt` |
| describe_manual_alternative_symlink_emitted | Low | code review | needs update-alternatives --set |
| describe_nonalternatives_symlink_not_routed_through_alt_db | Medium | `test_nonalt_symlink_no_alternatives_query_default_error` + `test_nonalt_symlink_emitted_verbatim` (v0.6.9 Change 1) | live crypto-policies/motd suppression vs emission verified manually on host |
| init_forces_on_unreadable_warn | Medium | live host check (`init out=… mode=external` did not abort on `/etc/libaudit.conf`; stopped at txn) | full onboarding success path needs root |
| describe_populates_content_store | Medium | `test_describe_populates_content_store` | — |
| describe_without_content_store_is_readonly | Medium | `test_describe_without_content_store_is_readonly` | — |
| describe_empty_ghost_suppressed | Low | code review | needs an empty ghost fixture |
| scope_attributes_always_object | Medium | `test_describe_attributes_never_null` | — |
| describe_verify_differences_not_unreadable | Medium | live describe under modified /etc (on-unreadable=warn) | — |
| verify_default_scope_ignores_usr | Low | code review | needs live applied record |
| verify_scope_full_detects_unmanaged_addition | Low | code review | needs live full scan + drift |
| verify_scope_full_detects_modified_package_file | Low | code review | needs live full scan |
| describe_scope_full_emits_observational_scopes | Medium | `test_describe_scope_full_has_observational_scopes` | — |
| describe_scope_full_boot_generated_files_unmanaged | Low | code review | host-specific /boot content |
| lock_is_fully_resolved_packages_scope | Low | code review | requires root convergence |
| yaml_manifest_accepted | Medium | `test_diff_yaml_manifest_accepted` | — |
| describe_format_yaml | Medium | `test_describe_format_yaml_stdout` | — |
| yaml_format_identity_stable | Low | code review (canonical_sha256 is format-independent) | identity-across-formats not directly asserted |
| yaml_unsafe_rejected | Low | code review | unsafe-YAML fixture not in suite |
| describe_unknown_format | Medium | `test_unknown_format_value_rejected` | — |
| bare_invocation_shows_help | Medium | `test_bare_invocation_shows_help` | — |
| version_verb_bare_word | Medium | `test_version_verb_bare_word` | — |
| version_flag_alias | Medium | `test_version_flag_alias_matches` | — |
| help_verb_bare_word | Medium | `test_help_verb_and_aliases` | — |
| unknown_verb_rejected | Medium | `test_unknown_verb_rejected` | — |
| describe_out_extension_yaml | Medium | `test_describe_out_extension_yaml` | — |
| describe_out_extension_json | Medium | `test_describe_out_extension_json` | — |
| describe_format_overrides_extension | Medium | `test_describe_format_overrides_extension` | — |
| verify_state_path_extension_yaml | Medium | `test_verify_state_path_extension_yaml` | — |
| describe_repositories_from_reposd | Low | code review | host repos.d-dependent |
| describe_unreadable_scope_strict | Low | code review | needs an unreadable repos.d |
| describe_unreadable_scope_warn | Medium | exercised by `on-unreadable=warn` describe tests | repos.d-specific omission code-reviewed |
| describe_omits_genuinely_empty_scope | Low | code review | needs an empty readable repos.d |
| diff_offline_two_files | Medium | `test_diff_offline_two_files_plan` | — |
| verify_offline_manifest_and_state | Medium | `test_verify_offline_matches` | — |
| verify_offline_no_applied_record_ok | Medium | `test_verify_offline_no_applied_record_ok` | — |
| apply_rejects_full_describe_dump | Medium | `test_apply_rejects_full_describe_dump` | — |
| idempotent_second_apply | Low | code review | requires root convergence |

## Files produced / updated this run

- **Updated (v0.6.9 behaviour + spec-hash + version):**
  - `src/actual_state.cpp` — Change 1: `AltDb`/`build_alt_db`/`is_alternatives_link`
    mechanism classifier; rewritten symlink branch of `judge_entry`;
    `alternatives_best` now also takes `root` and gated on non-zero exit.
  - `src/commands.cpp` — Change 2: `cmd_init` forces `OnUnreadable::Warn`.
  - Spec-hash re-embed across all `src/*.{hpp,cpp}`, `src/meta.hpp.in`,
    `CMakeLists.txt`, `Makefile`, the man page, RPM/DEB packaging, README, and the
    pikchr diagram; `VERSION` bumped `0.6.8`→`0.6.9`.
  - `independent_tests/claude-opus-4-8/{zypper-declarative_test.cpp,harness.hpp}`
    — spec-hash re-embed, fixture `generator` strings → `0.6.9`, and three new
    v0.6.9 black-box tests for the symlink classification.
  - `debian/changelog`, `zypper-declarative.spec` — new 0.6.9 changelog entries.
- **Unchanged in behaviour** (hash re-embed only): `manifest.cpp`, `diff.cpp`,
  `transaction.cpp`, `cli.cpp`, `main.cpp`, `command_runner.cpp`, and the headers.
- **This report** rewritten last, after the compile gate and live verification.
