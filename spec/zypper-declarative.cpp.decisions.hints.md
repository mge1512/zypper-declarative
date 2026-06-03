# zypper-declarative: translation decisions hints (C++)

C++ instance of `zypper-declarative.<lang>.decisions.hints.md` (PCD "When the
Specification Changes"), read by the translator during guided regeneration. Not a
spec artifact; does not affect `pcd-lint`; disposable. The C++ sibling of the Go
and Rust decisions files: same architecture, retargeted to C++ and the SUSE C/C++
libraries (the concrete bindings are the hard part for C++). Read the spec, the
cli-tool template, and `cli-tool.cpp.milestones.hints.md` first.

Tags: `[spec]` decided by the spec; `[pcd]` a PCD/environment constraint;
`[verified]` confirmed present on SLE 15 SP7 and SLE 16.0 from the SCC package
lists; `[recommended]` a sound C++ default; `[extract]` read from existing C++
code if present.

## Settings locked for this implementation

- Standard: C++17. Build system: CMake.
- Linking: DYNAMIC against the distribution's supported shared libraries. No static
  binary, no vendoring, no pinning. Build per service pack via OBS so each package
  links that SP's own sonames.
- System integration: link the SUSE libraries directly (libzypp, libsnapper); do
  NOT exec `zypper`/`snapper`, and do NOT call `rpm` or link librpm. (This is the
  deliberate difference from Go and Rust, which exec these tools to stay FFI-free
  and static; the C++ build linking the native libraries is what makes it a
  compelling "real SUSE C++ tool" and preserves three-way independence.)
- Compiler: g++-15 (`gcc15-c++`) on SLE 15 SP7; default GCC 15 on SLE 16.0.

---

## Verified library bindings (the core of this file)

All four are present in a FULLY SUPPORTED module on both SLE 15 SP7 and SLE 16.0
(not Package Hub), from the SCC lists. Two carry a soname/version difference
between SPs that the build and code must respect; OBS per-SP builds handle linking,
and the code must not assume one version's API.

| Concern | Library | devel package | SLE 15 SP7 | SLE 16.0 | Note |
|---|---|---|---|---|---|
| packages, rpmdb query, file ownership, baseline digests | libzypp | `libzypp-devel` | 17.37.x (Basesystem) | 17.37-17.38 (SLES core) | major 17 on both; minor API moves |
| snapshots / generations | libsnapper | `libsnapper-devel` | runtime lib `libsnapper5` 0.8.16 (15 SP7) | `libsnapper7` 0.12.x (16.x), `libsnapper8` 0.13.x (TW) | soname 5/7/8 binary-incompatible AND the API differs (Plugins::Report added in 0.12, one-arg removed in 0.13); build per-target, guard the snapshot call (see Snapshots section). devel pkg is `libsnapper-devel` on all |
| JSON | jsoncpp | `jsoncpp-devel` | 1.8.4 (Dev Tools) | 1.9.6 (SLES core) | stable API across these versions |
| YAML | yaml-cpp | `yaml-cpp-devel` | 0.6.3 (Basesystem; runtime lib `libyaml-cpp0_6`) | 0.8.0 (SLES core; runtime lib `libyaml-cpp0_8`) | 0.6 vs 0.8: API differs, compile against both; devel pkg is `yaml-cpp-devel` on BOTH, NOT `libyaml-cpp-devel` |

### Packages and per-file baseline: libzypp ONLY, via `RpmHeader::tag_fileinfos()`

- `[verified]` Link libzypp for every package operation: the installed set (name,
  version, release, arch), a file's owning package, AND the per-file recorded
  baseline for the pristine/ghost/type-mismatch determination.
- `[pcd]` These libzypp queries are READS of the rpmdb: run them at TRANSLATION
  TIME against the build host's real database, do NOT defer them to on-target.
  Build 01 deferred the package enumeration and `tag_fileinfos()` ownership, emitted
  an empty `packages` scope and 2221 all-unpackaged `/etc` files, and then
  rationalised the empty output as "a real absence of ownership". On a host with a
  populated rpmdb that is wrong: an empty result from a query you did not run is a
  SKIPPED lookup, not an absence. Only mutating operations (snapshot/apply) defer.
- `[verified from libzypp source]` The per-file baseline comes from
  `zypp::target::rpm::RpmHeader::tag_fileinfos()`, returning
  `std::list<zypp::target::rpm::FileInfo>`. `RpmHeader.cc` builds each `FileInfo`
  from the rpm header tags: `RPMTAG_FILEMD5S` (recorded digest), `RPMTAG_FILELINKTOS`
  (recorded symlink target), `RPMTAG_FILEMODES`/`FILEUSERNAME`/`FILEGROUPNAME`
  (recorded mode/owner/group), and `RPMTAG_FILEFLAGS`, which carries the GHOST bit
  (`RPMFILE_GHOST = 64`) and config bit (`RPMFILE_CONFIG = 1`). So libzypp exposes
  everything the v0.6.5 rule needs, ghost marker included, with no subprocess.
- Obtain the owning installed package's `RpmHeader` (via libzypp's rpm target /
  database query) and read its `tag_fileinfos()` once per relevant package: a single
  in-process pass, which naturally satisfies the bulk-lookup requirement (no per-file
  work).
- Do NOT link librpm or add `rpm-devel` (libzypp's `tag_fileinfos()`/`FileInfo` is
  confirmed to expose the flags/digest/linkto; there is no remaining reason to touch
  librpm). Do NOT exec `zypper`/`rpm`.
- `[changed-0.6.5]` `[changed-0.6.6]` Reproducibility rule via the `FileInfo` flags
  (emit a path exactly when a fresh install would not reproduce its on-disk state),
  each path judged INDEPENDENTLY against its own owning package, never collapsing a
  symlink with its target:
  - ghost (FILEFLAGS ghost bit) REGULAR FILE with on-disk content -> EMIT (a fresh
    install ships no content; e.g. `/etc/pam.d/common-auth-pc`);
  - ghost regular file on-disk empty AND recorded baseline empty -> SUPPRESS;
  - SYMLINK: "has content" is NOT the test (every symlink has a target). Build
    02 emitted all ~287 `/etc/alternatives/*` by treating a ghost symlink's target as
    "content"; do NOT do that. CLASSIFY THE SYMLINK'S MECHANISM BEFORE JUDGING: a
    symlink is an ALTERNATIVES symlink IFF it is under `/etc/alternatives/` OR appears
    as a master or slave in a `/var/lib/alternatives/<name>` admin file. ONLY those
    are resolved against the alternatives DB. Any OTHER symlink
    (`/etc/crypto-policies/back-ends/*.config` which point into
    `/usr/share/crypto-policies/<policy>/...`, `/etc/motd.d/*`, `/etc/issue.d/*`, any
    package-owned symlink) is a NON-ALTERNATIVES symlink: judge it by the normal
    target rule (on-disk target == the target a fresh install of its owning package
    establishes -> SUPPRESS, else EMIT as type "link" with the verbatim target), and
    NEVER call `update-alternatives` for it. A non-alternatives symlink with no
    alternatives entry is NOT an `on_unreadable` condition. THIS IS A REAL BUG THAT
    SHIPPED: the v0.6.8 build queried `update-alternatives` for crypto-policies
    back-ends and motd.d/issue.d links, producing 24 spurious "alternatives
    unreadable" warnings and aborting describe under the default error mode on
    `/etc/motd.d/cockpit`. On a default-policy system the crypto-policies back-end
    links point where the package set them, so once classified correctly they are
    SUPPRESSED (pristine) and emit no diagnostic. For an ALTERNATIVES symlink: the
    reproducible target is the auto/best provider; libzypp does not know it, so query
    the alternatives DB (`update-alternatives --query <name>` via OSCommandRunner, or
    read `/var/lib/alternatives/<name>`). Target EQUALS auto/best -> SUPPRESS; DIFFERS
    (a manual `--set`) -> EMIT as type "link" with the verbatim target. A SLAVE
    alternative whose auto/best is indeterminable -> EMIT conservatively (resolving it
    via the master admin file to suppress defaults is a permitted refinement, not
    required). `on_unreadable` applies to an alternatives symlink ONLY when
    `/var/lib/alternatives` genuinely cannot be read (rare), never for "this is not an
    alternative".
  - on-disk type differs from the recorded type (recorded a regular file, disk has a
    symlink) -> EMIT as the on-disk type (e.g. `/etc/pam.d/common-auth`);
  - otherwise non-ghost: pristine iff digest+mode+owner+group (file) or recorded
    linkto (symlink) match -> SUPPRESS, else EMIT.
  Build 01 suppressed BOTH pam paths (no ghost/type-mismatch handling) and build 02
  over-emitted alternatives (ghost-symlink-as-content); under v0.6.6 emit the
  type-mismatch symlink and the content-bearing ghost FILE, suppress default
  alternatives, emit manually-set ones, all judged independently.
- `[changed-0.6.4]` Pristine refinements (the C++ build was mostly correct; these
  converge it with Go/Rust): a symlink is pristine iff its TARGET matches the
  recorded target (do NOT compare a symlink's mode; the first build over-emitted
  pristine distro symlinks `/etc/X11/xim.d/*/40-ibus`, which this suppresses);
  `package_name` is the BARE name (`openssh-server`), which libzypp gives directly
  (do not append version/arch).
- libzypp pulls in libsolv, curl, and boost transitively; that is fine for a
  dynamically linked tool, do not re-implement what libzypp gives you. But do not
  reach for boost GRATUITOUSLY: use `std::filesystem` for the `/etc` walk (not
  boost.filesystem) and jsoncpp for JSON (not boost.json).
- `[recommended]` Use libzypp MORE rather than adding breadth: enumerate
  repositories through libzypp's repo API against `<root>`'s zypp configuration
  rather than hand-parsing `/etc/zypp/repos.d/*.repo` (one interpretation, one trust
  boundary, chroot-safe). This replaces a bespoke INI parser.
- `[verified]` `[recommended]` SHA256 via libcrypto (OpenSSL 3), already in
  libzypp's dependency closure, rather than vendoring a hash routine.
  `libopenssl-3-devel` is in a fully supported core module on both SPs: 3.2.3 on
  15 SP7, 3.5.0 on 16.0, same API, NO major-version/soname split to code around
  (unlike yaml-cpp and libsnapper). 15 SP7 also carries OpenSSL 1.1 in the
  Certifications/Dev Tools modules, but do not target it.
- `[recommended]` `[reserved-0.7.0]` libsolv (`libsolv-devel` on both SPs, already a
  libzypp transitive dep) is the right tool for dependency RESOLUTION when the
  `apply`/transaction path is built (computing the transaction against the rpmdb
  under `<root>`, chroot-safe). Adopt it at the apply milestone, not for describe.
- CMake: discover libzypp via `pkg_check_modules(ZYPP REQUIRED IMPORTED_TARGET
  libzypp)` and link `PkgConfig::ZYPP`. Prefer pkg-config over `find_package(Zypp)`
  for the same reason as jsoncpp/yaml-cpp (per-SP CMake-config fragility; `libzypp.pc`
  is the stable cross-SP discovery). Only fall back to a small Find module under
  `cmake/` if no `.pc` is present. Record the mechanism and detected version in
  `TRANSLATION_REPORT.md` and verify on the host.

### Snapshots: libsnapper, soname 5 vs 7 vs 8, and the Plugins::Report API break

- Link libsnapper (`libsnapper-devel`) for snapshot/generation operations the
  transaction binding needs (creating, listing, querying btrfs snapshots, and
  reading/writing snapshot userdata where the applied-record ledger rides along).
- The soname AND the API differ across the targets, this is a hard
  compile-time divergence, not just a soname bump:
  - `libsnapper5` 0.8.16 (15 SP7): only `createSingleSnapshotOfDefault(const SCD&)`;
    the `Plugins` namespace and `Plugins::Report` type DO NOT EXIST.
  - `libsnapper7` 0.12.x (16.0/16.1): BOTH `createSingleSnapshotOfDefault(const SCD&)`
    (marked `SN_DEPRECATED`) AND `createSingleSnapshotOfDefault(const SCD&,
    Plugins::Report&)` exist.
  - `libsnapper8` 0.13.1 (Tumbleweed): the one-arg form is REMOVED; only
    `createSingleSnapshotOfDefault(const SCD&, Plugins::Report&)` exists.
  In snapper 0.12 every mutating call (createSingleSnapshot, createPreSnapshot,
  modifySnapshot, deleteSnapshot, ...) gained a trailing `Plugins::Report&`; the same
  conditioning applies to any of them the binding uses.
- REQUIRED: the snapshot-creation call MUST be guarded so it compiles on all three.
  A single unconditional call cannot work (the `Plugins::Report` type is absent on
  0.8.16, and the one-arg overload is absent on 0.13.1). Detect snapper >= 0.12 and
  branch:
  - on >= 0.12: default-construct a `snapper::Plugins::Report` and call the two-arg
    form `createSingleSnapshotOfDefault(scd, report)` (this is the non-deprecated path
    and avoids the `SN_DEPRECATED` warning on 16.x, the demo target).
  - on < 0.12: call the one-arg form `createSingleSnapshotOfDefault(scd)`.
- MECHANISM: do the version detection in CMake (it already has the snapper version
  from `pkg_check_modules`, captured as the snapper module version), NOT a guessed
  snapper header macro. When the detected version is >= 0.12, define a project macro,
  e.g. `target_compile_definitions(zypper-declarative PRIVATE ZD_SNAPPER_REPORT_PARAM)`;
  the code does `#if defined(ZD_SNAPPER_REPORT_PARAM)`. This keeps the version logic
  in the build (where the version is known) and out of a fragile header probe. Record
  the detected snapper version and which branch was compiled in `TRANSLATION_REPORT.md`.
- If some other operation has no stable libsnapper API across the versions, THAT
  operation (only) may fall back to `OSCommandRunner` invoking `snapper`, noted
  explicitly; the default remains the library.

### JSON: jsoncpp

- Use jsoncpp (`jsoncpp-devel`) for canonical JSON; present and supported on both
  SPs (1.8.4 on 15 SP7 via Dev Tools; 1.9.6 on 16.0 in SLES core), API stable.
- Rejected: boost.json ships only on 16.0 (fails "supported on both"); rapidjson and
  similar are Package Hub only; header-only libraries (nlohmann-json) would be
  vendored, against the dynamic-linking/no-vendoring decision.
- CMake: discover jsoncpp via `pkg_check_modules(JSONCPP REQUIRED IMPORTED_TARGET
  jsoncpp)` and link `PkgConfig::JSONCPP`. Do NOT use `find_package(jsoncpp)`: on
  SLE 16 the jsoncpp `jsoncppConfig.cmake` is generated by Meson (its header says
  "input file was jsoncppConfig.cmake.meson.in") and Meson's `@PACKAGE_INIT@`
  imitation omits the `include(CMakePackageConfigHelpers)` that real CMake-generated
  configs carry, so the file's final `check_required_components(jsoncpp)` call fails
  with "Unknown CMake command", breaking the build. pkg-config sidesteps the broken
  CMake config entirely: `jsoncpp.pc` is present on both SPs (1.8.4 on 15 SP7, 1.9.6
  on 16.0) and `pkg-config --libs jsoncpp` yields `-ljsoncpp`. Prefer pkg-config for
  this dependency on both SPs (uniform, and immune to the per-SP CMake-glue drift).

### YAML: yaml-cpp, 0.6 vs 0.8

- Use yaml-cpp for the opt-in YAML serialisation; present on both but at different
  API levels (runtime lib `libyaml-cpp0_6` 0.6.3 on 15 SP7 vs `libyaml-cpp0_8` 0.8.0
  on 16.0). The devel package to BuildRequire is `yaml-cpp-devel` on BOTH SLE 15 and
  16, NOT `libyaml-cpp-devel`; the `lib...0_6`/`0_8` names are the RUNTIME shared-lib
  packages, not the devel package. Compile against both: restrict usage to API stable
  across 0.6 through 0.8, avoid 0.7+-only entry points. DISCOVER via
  `pkg_check_modules(YAMLCPP REQUIRED IMPORTED_TARGET yaml-cpp)` (pkg-config module is
  `yaml-cpp`, no `lib`) and link `PkgConfig::YAMLCPP`, NOT
  `find_package(yaml-cpp)`: yaml-cpp's CMake config has the same per-SP fragility as
  jsoncpp (the SLE 16 build can ship a config that assumes a helper preamble it does
  not include), and pkg-config (`yaml-cpp.pc`, present on both) is immune. If a
  version branch is needed, read it from `YAMLCPP_VERSION` (set by
  `pkg_check_modules`) rather than a CMake-config variable; record the version in
  `TRANSLATION_REPORT.md`.
- Rejected: rapidyaml is Package Hub only. The C library libyaml (`libyaml-devel`,
  on both) is a fallback if yaml-cpp's cross-version surface proves too narrow, but
  prefer yaml-cpp; note any switch.

### Services: offline `systemctl --root`, NOT libsystemd/sd-bus

- The `services` scope is unit ENABLEMENT (enabled/disabled/masked) and is MANDATORY
  in `describe-actual-state`, not a deferral. The first C++ build deferred it to an
  empty (hence omitted) scope while Go read 214; that is a bug to close. A
  deferred-empty scope silently drops declarable state.
- Read enablement OFFLINE under the context root via `systemctl --root <root>
  is-enabled <unit>` and `systemctl --root <root> list-unit-files`, through
  `OSCommandRunner`. Do NOT use libsystemd/sd-bus: it talks to the running system's
  PID 1 and cannot answer enablement under THIS other root, but the tool's whole
  model is rooted operations (describe `root=/mnt`, convergence into a mounted
  snapshot). `systemd-devel` was also not confirmed on 15 SP7, a second reason to
  avoid the build-time dependency. (libsystemd would only earn its place for a future
  LIVE runtime-state feature, which the spec does not ask for.) Purely-static units
  are omitted; normalise to enabled/disabled/masked.

---

## Architecture decisions (same as Go/Rust, in C++ terms)

### Source layout and the single-reader rule

- `[recommended]` One `.{hpp,cpp}` pair per concern, mirroring the spec's behaviour
  grouping (see the C++ milestones hints for the tree); reconcile with any existing
  layout and prefer it where at least as clear.
- `[spec]` `describe-actual-state` is the single live-state reader: the only TU that
  talks to libzypp's rpmdb, reads `/etc/zypp/repos.d`, reads unit state, or walks
  `/etc`. Keep it isolated so the libzypp/libsnapper linkage is concentrated there
  and in the txn module.
- `[spec]` `compute-drift` performs no I/O (a pure comparison of two in-memory
  `Manifest` values); keep it free of libzypp, filesystem, and process calls.
- `[spec]` `resolve-format` is the single authority for serialisation choice; route
  every read and write through it, no per-call-site format logic.

### CLI contract and errors

- `[spec]` Options are `key=value`, parsed by the tool; bare words are verbs; accept
  options in ANY position (do not reject options after the verb).
  Environment-variable control is forbidden (a debug env var is a trace gate only,
  not control). All CONFIG knobs are also `key=value` options; command-line overrides
  preset.
- `[spec]` `version` and `help` are bare-word global commands, exit 0;
  `--version`/`--help`/`-h` are tolerated aliases only; no option uses POSIX `--flag`.
  Bare invocation prints usage to stdout, exits 0 (never converges). Unknown
  verb/option/value or missing value -> usage to stderr, exit 2.
- `[spec]` Internal functions return errors to their caller; only the verb layer maps
  to an exit code. Model `Diagnostic { severity, domain, message }` with `domain` in
  {packages, repositories, services, files, manifest, transaction, invocation}.
  `[recommended]` Use a small `Result`-like type or exceptions caught at the verb
  boundary; do NOT let a libzypp exception escape as an uncaught throw, translate it
  to a `Diagnostic` with the right domain. Exit codes: 0 success; 1 logical failure;
  2 invocation error (full list per spec).

### Manifest model and serialisation

- `[spec]` Declarable Machinery subset (packages, repositories, services,
  config_files), `ScopeWrapper { _attributes, _elements }`, underscore field names;
  emit those exact keys via jsoncpp/yaml-cpp regardless of C++ member names. Each
  scope's `_attributes` is a JSON object, empty `{}` when there are none, NEVER null
  (or YAML `~`). In YAML, quote string-typed scalars so `mode: "0600"`, not
  `mode: 0600`.
- `[spec]` Absent vs empty scope is semantic: model a declarable scope as
  `std::optional<ScopeWrapper<...>>` (`std::nullopt` = absent/unmanaged; a present
  wrapper with empty `elements` = reconcile-to-empty). The JSON writer must OMIT a
  `nullopt` scope entirely, not write `null` or an empty object.
- `[spec]` `resolve-format`: explicit `format=` wins, else operative file extension,
  else `manifest-format` default. describe output follows `resolve-format(out)`; do
  not hardcode JSON. The applied record is ALWAYS canonical JSON regardless of input
  format.
- `[spec]` `desired_sha256` is the SHA256 of a canonical serialisation of the parsed
  model (format-independent). `[recommended]` Define canonical concretely: keys
  sorted, compact separators, UTF-8, `_elements` sorted by identity key (packages by
  name+arch, repositories by alias, services by name, config_files by path). Sort
  containers yourself before writing (jsoncpp does not guarantee key order); its
  `StreamWriterBuilder` can be configured compact. On-disk `applied.json` may be
  pretty; the hash is over the compact form. Use libcrypto for SHA256.
- `[spec]` `meta.generator` is `zypper-declarative <version>`, matching other
  implementations of the same spec version (do not drop the version).
- `[spec]` YAML safe profile: non-executing loader, no arbitrary/executable tags,
  bounded or disabled alias expansion, single document only, explicit typing per
  schema (no implicit coercion such as `NO`->false, `1.10`->float). `[recommended]`
  With yaml-cpp, load into a `YAML::Node` and walk it: reject any node with a
  non-default tag, reject documents beyond the first, read scalars with explicit
  `as<std::string>()` typing, and bound anchor/alias expansion (reject or cap). A
  YAML input needing a disabled feature is a manifest error. Record how each
  constraint is enforced in `TRANSLATION_REPORT.md`.

### Reading actual state, empty-scope rule, filesystem object model

- `[spec]` Repositories actual state from `/etc/zypp/repos.d/*.repo` (INI; or via
  libzypp's repo enumeration); no network refresh, no privileged cache. A scope
  source that cannot be read is NEVER an empty scope: `on-unreadable=error` (default)
  errors naming the source, `warn` omits with a diagnostic; genuinely-empty readable
  scopes are omitted.
- `[spec]` Filesystem object model: walk `/etc` (and the scope=full trees)
  recursively, classifying each entry by its own type WITHOUT following symlinks. In
  C++: `std::filesystem::directory_iterator` with `symlink_status()` (NOT `status()`,
  which follows links), or `lstat(2)`; classify regular file / symlink / directory /
  other. Read a symlink target with `std::filesystem::read_symlink` and store it
  VERBATIM (no `canonical`/`weakly_canonical`/normalisation; keeps chroot-relative
  targets correct). Hash regular files only; descend directories and emit nothing;
  skip special files. A directory, symlink, or special file is NEVER an
  unreadable-source error (the C++ trap is calling `status()`/`file_size`/an ifstream
  on a path before classifying; classify first). Records carry a verbatim `target`
  field ("" for non-links), sha256 only for regular files, per the TYPES rules.
- `[spec]` In `compute-drift`, type is part of identity: differing type -> modified;
  same type compare sha256 (file) or target (link); a declared entry absent from
  actual is treated as matching. Hardlinks: treat per path by content+type, do not
  detect or preserve hardlink identity. `compute-drift`'s `reference` is a `Manifest`
  that may be a DESIRED MANIFEST or an APPLIED RECORD (same schema); the comparison
  is identical, the caller chooses. CRITICAL for `diff`: pass the DESIRED MANIFEST as
  the drift reference, NOT the applied record. On SL Micro most of `/etc` is
  unpackaged, and on SL Micro a packaged-looking path may legitimately report
  `package_name` "" (SL Micro does not package-own much of `/etc`, that is correct,
  not a bug); diffing actual against an absent/empty applied record reported all such
  files as `files_extra`, while the desired manifest already contains those paths so
  drift is empty. The applied record is the reference only for the intent diff and
  apply's post-converge check.
- `[spec]` `compute-drift` packages_divergent: an EMPTY identity field in a REFERENCE
  package element is a WILDCARD matching any actual value. A desired
  `{name: nginx, version: ""}` matches the resolved actual `{name: nginx, version:
  "1.27.4"}` and is NOT divergent; only non-empty reference fields must match. This
  is required so the "newest from repo" case (empty desired version) does not
  manufacture false drift against the fully-resolved installed record. Wildcard only
  on the reference side, not the actual side. (Do not implement packages_divergent as
  a strict symmetric identity set-difference; that would false-flag every
  empty-version desired package.)
- `[spec]` `init` verb (onboarding): one command adopting current state as the
  managed baseline. INCLUDES the describe read (the same libzypp + filesystem
  actual-state path describe uses, with content-store population if `content-store=`
  is set), acquires the transaction context (TAKES A SNAPSHOT, the same
  acquire-transaction-context path apply uses), writes the described state as the
  applied record inside the snapshot (`meta.desired_sha256` = hash of that state),
  CONVERGES NOTHING (no /etc write, no package, no unit), also writes the adopted
  manifest to `out`, then seals. After init, `diff` against the adopted manifest is
  empty and `verify` matches. A never-onboarded machine showing a large intent diff
  is the run-init signal, not a bug. (init reuses describe + acquire-transaction +
  write-applied-record; it is apply minus the convergence steps.) `init` FORCES
  `on_unreadable=warn` for its live read, overriding the default error and any
  `on-unreadable=error` passed in: onboarding a real machine must not abort on a
  protected root-only file or an indeterminable source. `init` is the ONLY verb that
  overrides the knob; describe/diff/verify/apply keep error as default.
- `[spec]` config_files is bounded to `/etc` (never read/hash/verify outside it,
  never a whole-system verification). Difference-reporting (a verifier returning
  non-zero because it found changes) is the normal result, not an unreadable source.
  (For C++ the baseline comes from libzypp `tag_fileinfos()`, not a verifier exec;
  the rule still holds at the model level.)
- `[spec]` CONTENT STORE: by default describe is read-only and every `content_ref`
  is "". When the `content-store` option gives a base path, for each EMITTED
  regular-file record write its bytes to `<content-store>/sha256/<digest>`
  (idempotent: skip if the blob exists, dedup by content) and set `content_ref` to
  `sha256/<digest>` (same digest as the record's `sha256`, computed via libcrypto).
  Symlinks/dirs keep `content_ref` "". A regular file emitted but unreadable follows
  `on_unreadable` (error, or under warn emit with `content_ref` "" plus a
  diagnostic), never silent. The manifest references content, never inlines it.

### Transaction binding

- `[spec]` Abstract: `acquire-transaction-context` resolves auto|external|internal
  and yields a context with a writable root; keep it isolated in the txn module,
  layered on libsnapper. The convergence path is identical regardless of binding.
  Unit enablement under a root uses offline enablement; do not rely on first-boot
  preset.
- `[spec]` `[reserved-0.7.0]` `converge-files` does NOT yet create/update/remove
  symlinks or handle type transitions (reserved for the apply milestone). When
  implemented: a declared type "link" is converged by its target; a declared-vs-
  actual type mismatch at a path is a HARD ERROR that aborts the transaction (no
  silent destructive rewrite, no deleting a directory tree to write a file).

### Spec-hash embedding and packaging

- `[spec]` Embed the spec SHA256 in source headers, `version` output,
  `TRANSLATION_REPORT.md` (`Spec-SHA256:`), the RPM spec, the DEB control
  (`X-PCD-Spec-SHA256:`), the Containerfile label, and the CMake project metadata.
  `[recommended]` Generate a `meta.hpp` (or a configured header via CMake
  `configure_file`) holding version and hash; `version` prints
  `zypper-declarative <version> spec:<sha256>`.
- `[spec]` Surfaced as a zypper subcommand (`/usr/lib/zypper/commands`) and invocable
  directly. OBS package, no curl install. RPM spec `BuildRequires:` on SLE 15:
  `gcc15-c++`, `libzypp-devel`, `libsnapper-devel`, `jsoncpp-devel`, `yaml-cpp-devel`,
  `libopenssl-3-devel`; dynamic runtime requires resolved by the linker. TWO devel
  package names are commonly gotten wrong; both follow the `<name>-devel` convention
  with NO `lib` prefix on SLE 15 AND 16: the jsoncpp devel package is `jsoncpp-devel`
  (NOT `libjsoncpp-devel`), and the yaml-cpp devel package is `yaml-cpp-devel` (NOT
  `libyaml-cpp-devel`). The `lib...` names with soname digits (`libyaml-cpp0_6`,
  `libyaml-cpp0_8`) are the RUNTIME shared-library packages, not the devel packages;
  do NOT put them in `BuildRequires`. (A prior RPM spec used `libjsoncpp-devel` and
  `libyaml-cpp-devel` and failed to find the dependencies.) The VERSION single-source
  file (below) supplies the spec `Version:`. SIGTERM/SIGINT clean exit; an
  interrupted `apply` discards the transaction.
- `[spec]` RPM directory ownership (OBS Factory `50-check-filelist` FAILS the build
  on directories owned by no package): the package installs into
  `/usr/lib/zypper/commands/`, so it MUST ensure `/usr/lib/zypper` and
  `/usr/lib/zypper/commands` are owned. Add `Requires: zypper` (correct regardless,
  it is a zypper subcommand and needs zypper at runtime). THEN: if the `zypper`
  package owns `/usr/lib/zypper/commands` (check on the target with
  `rpm -qf /usr/lib/zypper/commands`), do NOT also `%dir` it (that causes a
  duplicate-ownership rpmlint complaint), the `Requires: zypper` suffices. If zypper
  does NOT own it, the `%files` section must claim both directories with
  `%dir %{_prefix}/lib/zypper` and `%dir %{_prefix}/lib/zypper/commands`. A prior
  build failed `50-check-filelist` with "directories not owned by a package:
  /usr/lib/zypper, /usr/lib/zypper/commands" on both 15 SP7 and 16.1 because neither
  the dirs were owned nor zypper required. (Note: the build itself succeeds and the
  RPM is written; this check runs AFTER and fails the OBS build, so it is easy to
  miss in a local rpmbuild that does not run the OBS post-checks.)
- `[pcd]` Version single-source: a top-level `VERSION` file (one line, e.g. `0.6.7`)
  is the sole version authority. The RPM spec reads it for `Version:` (e.g.
  `Version: %(cat %{_sourcedir}/VERSION)` or injected at tarball build), the
  `make dist` target (below) reads it for the tarball name and subdirectory, and the
  build embeds it (via `configure_file`) so the binary's `version` output, the RPM
  `Version:`, and the tarball `zypper-declarative-X.Y.Z/` are guaranteed identical.
- `[pcd]` The Makefile MUST define a `make dist` target (this is REQUIRED, not
  optional; a prior build omitted it). Conventional name, avoids CMake CPack's
  `package` collision. It produces `zypper-declarative-X.Y.Z.tar.gz` containing a
  single top-level directory `zypper-declarative-X.Y.Z/`, where `X.Y.Z` is read from
  the `VERSION` file, so `rpmbuild`'s default `%setup`/`%autosetup` (which cd's into
  `%{name}-%{version}/`) finds the expected directory. Exclude build artifacts
  (`build/`, VCS dirs) from the tarball. See the milestones hints for the Makefile
  target list and shape; `dist` sits alongside `build`, `test`, `man`, `clean`.

### Testing boundary

- `[pcd]` Black-box tests invoke the built binary and assert on stdout, stderr, and
  exit code; they do NOT link the internals. The v0.5.x/0.6.x examples (bare
  `version`/`help`, `describe out=...yaml`, offline `verify`/`diff`, `/etc`
  directory traversal, symlink-verbatim, special-file skip, type-transition drift)
  are black-box assertions; the symlink and fifo cases are constructible under a
  synthetic root without root privilege, cover them rather than leaving them
  code-review-only.
- `[pcd]` The test harness's command-runner contract (hermetic per-invocation temp
  files or concurrently-drained pipes, never shared `/tmp/out`/`/tmp/err`, never
  `std::system`/shell, run as the build user without sudo) is generic and is
  specified in the cli-tool milestones scaffold; follow it. The project-specific
  part is that this tool's live-reading verbs take `on-unreadable=warn` in tests
  (below) so a protected root-only source never aborts a test.
- `[pcd]` No hollow tests (the milestones scaffold states the general rule; this is
  how it applies here). A test for an EXAMPLE asserts that example's THEN, the
  emitted/suppressed records, the drift report content, the exit code, the specific
  diagnostic, NOT merely `exit_code == 0`. Concretely, build the fixture the
  example's GIVEN implies and assert the real outcome:
  - manifest-driven examples (`apply_*`, `diff_*`, `verify_*`, `intent_diff_*`,
    `idempotent_second_apply`): write the desired manifest as a JSON (or YAML) file
    in a temp dir, and for verify/diff also write a captured actual-state dump;
    use the OFFLINE modes the spec provides (`diff manifest-path=... state-path=...`,
    `verify manifest-path=... state-path=...`) so the behaviour is asserted with no
    live system, then assert the plan/drift lines or exit code the example states.
  - describe examples that judge `/etc` content (`describe_records_symlink_verbatim`,
    `describe_skips_special_file`, `describe_traverses_etc_subdirectories`,
    `describe_type_mismatch_emitted`, `describe_ghost_with_content_emitted`,
    `describe_*_alternative_symlink_*`, `describe_crypto_policies_symlinks_not_alternatives`,
    `describe_config_files_bounded_to_etc`): construct a SYNTHETIC root/`/etc` subtree
    in a temp dir with the specific objects (a symlink with a known target, a fifo, a
    nested file, a link whose target differs from auto/best), point describe at it via
    the root option the spec defines, and assert the record is present/absent with the
    expected `type`/`target`/`sha256`. These are constructible without root.
  - the `init`/snapshot/apply-convergence examples that need a real transactional
    root or root privilege the build user lacks: mark SKIPPED (needs a live
    transactional target), do not emit an exit-0 stub. The offline-checkable half of
    init (that it WRITES a manifest and an applied record, idempotent diff afterwards)
    can still be asserted where the snapshot step is not required.
  A suite where the describe/verify/apply examples are bare `exit_code == 0` stubs
  with an "auto-generated generic"/"fallback" comment is a FAILED run; reject it.
- `[spec]` Required self-checks (black-box, against the build host's real rpmdb). Run
  with the binary invoked as the build user, NOT via `sudo` and NOT assuming root:
  run `describe` with `on-unreadable=warn` so a protected root-only file (e.g.
  `/etc/libaudit.conf`) is skipped with a diagnostic instead of aborting describe;
  never call `sudo` in a test step (an interactive password prompt hangs the build,
  this happened twice). The asserted paths are world-readable, so the check is
  satisfiable unprivileged. Run `describe` and assert (1) the `packages` scope
  is present and NON-EMPTY (build 01 omitted it); (2) ownership is attached where rpm
  reports an owner, pick a path that `rpm -qf` resolves on the build host and assert
  the record's `package_name` matches (on a package-managed host
  `/etc/ssh/sshd_config` -> `openssh-server`; do NOT hardcode an owner for a path
  that `rpm -qf` reports as unowned, on SL Micro much of `/etc` is legitimately
  unpackaged and an empty `package_name` there is correct); (3) a known-pristine
  `/etc/ImageMagick-7-SUSE/*.xml` is ABSENT when that package is installed; (4) the
  pam pair on a host that packages it, `common-auth`/`common-auth-pc` with the right
  types. These bind the libzypp read: if `tag_fileinfos()` returned nothing, (1) and
  (2) fail and the build cannot pass by rationalising empty output. (5) IDEMPOTENCE:
  `describe out=/tmp/m.json on-unreadable=warn` then `diff manifest-path=/tmp/m.json
  on-unreadable=warn` MUST produce an EMPTY drift report (no `files_extra`), and
  `init out=/tmp/m.json` then the same `diff` likewise; this binds the drift-reference
  fix (drift compares the desired manifest, not the applied record) and is the check
  that the SL Micro bug would have failed. Pass `on-unreadable=warn` to BOTH the
  describe and the diff (the knob is exposed on `diff`/`verify`/`apply` as well as
  `describe`, default error) so neither half aborts on a protected root-only file
  unprivileged. (Go and Rust carry the equivalent checks; this is how the three stay
  convergent.)

## Spec tracking

- Tracks spec zypper-declarative.spec.md v0.6.9 (hash aafbb315...). History is in git and in the spec CHANGELOG.md, not here.
