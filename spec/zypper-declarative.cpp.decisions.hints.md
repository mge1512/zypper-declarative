# zypper-declarative: translation decisions hints (C++)

This is the C++ instance of `zypper-declarative.<lang>.decisions.hints.md`, the
decisions hints file from PCD section 13, read by a translator during guided
regeneration. It is NOT a specification artifact (it does not affect `pcd-lint`)
and it is disposable. Read the spec, the cli-tool template, and
`cli-tool.cpp.milestones.hints.md` before writing code.

It is the C++ sibling of the Go and Rust decisions files: the same architecture
and the same "do NOT carry over" list, retargeted to C++ and to the SUSE C/C++
libraries. The specification is language-neutral; this file holds the
C++-specific decisions, including the concrete library bindings, which are the
hard part for C++.

## Provenance tags

- `[spec]` decided by `zypper-declarative.spec.md`; authoritative here.
- `[pcd]` a documented PCD finding or environment constraint; authoritative.
- `[verified]` confirmed present on SLE 15 SP7 and SLE 16.0 from the SCC package
  lists; the basis for a library choice.
- `[recommended]` a sound C++ default for this tool; apply unless an existing C++
  implementation does something equally good.
- `[extract]` read from an existing C++ implementation if one exists; a slot.
- `[changed-N]` spec version N changed this; follow the new spec.

## Settings locked for this implementation

- **Standard:** C++17.
- **Build system:** CMake.
- **Linking:** DYNAMIC against the distribution's supported shared libraries. No
  static binary, no vendoring, no pinning. Build per service pack via OBS so each
  package links that SP's own sonames.
- **System integration:** link the SUSE libraries directly (libzypp, libsnapper);
  do NOT exec `zypper` or `snapper`, and do NOT call `rpm` or link librpm.
- **Compiler:** g++-15 (`gcc15-c++`) on SLE 15 SP7; default GCC 15 on SLE 16.0.

---

## Verified library bindings (the core of this file)

All four libraries below are present in a FULLY SUPPORTED module on both SLE 15
SP7 and SLE 16.0 (not SUSE Package Hub), confirmed from the SCC package lists.
Two of them carry a soname/version difference between the service packs that the
build and the code must respect; OBS per-SP builds handle the linking, and the
code must not assume one version's API.

| Concern | Library | devel package | SLE 15 SP7 | SLE 16.0 | Note |
|---|---|---|---|---|---|
| packages, rpmdb query, file ownership, baseline digests | **libzypp** | `libzypp-devel` | 17.37.x (Basesystem Module) | 17.37-17.38 (SLES core) | major 17 on both; minor API moves |
| snapshots / generations | **libsnapper** | `libsnapper-devel` | `libsnapper5` 0.8.16 (Basesystem Module) | `libsnapper7` 0.12.1 (SLES core) | **soname 5 vs 7: binary-incompatible, build per-SP** |
| JSON | **jsoncpp** | `jsoncpp-devel` | 1.8.4 (Development Tools Module) | 1.9.6 (SLES core) | stable API across these versions |
| YAML | **yaml-cpp** | `libyaml-cpp` | `libyaml-cpp0_6` 0.6.3 (Basesystem Module) | `libyaml-cpp0_8` 0.8.0 (SLES core) | **0.6 vs 0.8: API differs, compile against both** |

### `[verified]` Packages: libzypp ONLY (not librpm, not exec rpm)

- Link **libzypp** for every package operation the spec requires: querying the
  installed set (name, version, release, arch), determining a file's owning
  package, and obtaining a packaged file's recorded baseline digest/metadata for
  the changed-vs-pristine comparison. libzypp's rpm target provides all of this.
- **Do NOT link librpm and do NOT add `rpm-devel`.** libzypp already sits on top
  of librpm; reaching past it would be a redundant dependency and a second,
  lower-level trust path into the rpmdb. The spec needs nothing librpm offers that
  libzypp does not already expose. (Recorded here explicitly because it is an easy
  wrong turn: the answer is libzypp, full stop.)
- **Do NOT exec `zypper` or `rpm`.** Use the library. (This is the deliberate
  difference from the Go and Rust builds, which exec these tools to stay free of
  C/C++ FFI and remain static binaries. The C++ build is the one that links the
  native SUSE libraries, which is part of what makes it a compelling "real SUSE
  C++ tool" demonstration.)
- libzypp pulls in libsolv, curl, and boost transitively; that is expected and
  fine for a dynamically linked tool. Do not try to avoid boost by re-implementing
  what libzypp gives you.
- CMake: locate via `find_package(Zypp)` if the devel package ships a CMake config
  or pkg-config file; otherwise `pkg_check_modules(ZYPP libzypp)` or a small Find
  module under `cmake/`. Record in `TRANSLATION_REPORT.md` which mechanism was
  used and the detected version, and verify it on the build host.

### `[verified]` Snapshots: libsnapper, soname 5 vs 7

- Link **libsnapper** (`libsnapper-devel`) for snapshot/generation operations
  needed by the transaction binding (creating, listing, and querying btrfs
  snapshots, and reading/writing snapshot userdata where the applied-record ledger
  rides along).
- **The soname differs: `libsnapper5` (0.8.16) on SLE 15 SP7 vs `libsnapper7`
  (0.12.1) on SLE 16.0.** These are binary-incompatible and the 0.8 -> 0.12 jump
  may move the API. The code must compile against whichever is present and must
  not hardcode assumptions tied to one version; the OBS spec builds separately per
  SP, so each RPM links the correct soname. Where the API differs between 0.8 and
  0.12, guard with a configure-time check (CMake feature test) rather than a
  compile-time version macro guess. Record the detected libsnapper version in
  `TRANSLATION_REPORT.md`.
- If a given operation has no stable libsnapper API across both SPs, that specific
  operation is a candidate to fall back to `OSCommandRunner` invoking `snapper`,
  but only that operation, and note it explicitly. The default remains the library.

### `[verified]` JSON: jsoncpp

- Use **jsoncpp** (`jsoncpp-devel`) for the canonical JSON serialisation and
  parsing. Present and supported on both SPs (1.8.4 on 15 SP7 via the Development
  Tools Module; 1.9.6 on 16.0 in SLES core), API stable across that range.
- Rejected alternatives and why: **boost.json** ships only on SLE 16.0
  (`libboost_json1_86_0`), not on 15 SP7, so it fails the "supported on both" rule;
  **rapidyaml/rapidjson**-style and other choices are Package Hub only. Header-only
  libraries (e.g. nlohmann-json) are not used because they would be vendored,
  against the dynamic-linking/no-vendoring decision.
- CMake: `find_package(jsoncpp)` (ships a CMake config) or `pkg_check_modules(
  JSONCPP jsoncpp)`.

### `[verified]` YAML: yaml-cpp, 0.6 vs 0.8

- Use **yaml-cpp** (`libyaml-cpp`) for the opt-in YAML serialisation. Present on
  both SPs but at different API levels: `libyaml-cpp0_6` (0.6.3) on 15 SP7 vs
  `libyaml-cpp0_8` (0.8.0) on 16.0.
- **Compile against both:** restrict usage to API stable across 0.6 through 0.8,
  and avoid 0.7+-only entry points. Locate with `find_package(yaml-cpp)` and, if
  needed, branch on the discovered version in CMake. Record the detected version
  in `TRANSLATION_REPORT.md`.
- Rejected: **rapidyaml** is Package Hub only. The C library **libyaml**
  (`libyaml-devel`, present on both) is an alternative if yaml-cpp's cross-version
  surface proves too narrow, but prefer yaml-cpp for a C++ API; note any switch.

---

## Architecture decisions (same as Go/Rust, in C++ terms)

### Source layout and the single-reader rule

- `[recommended]` Mirror the spec's behaviour grouping, one `.{hpp,cpp}` pair per
  concern (see the C++ milestones hints for the tree). Reconcile with any existing
  layout and prefer it where at least as clear.
- `[spec]` `describe-actual-state` is the single live-state reader: it is the only
  translation unit that talks to libzypp's rpmdb, reads `/etc/zypp/repos.d`, reads
  unit state, or walks `/etc`. No other module reads live state. Keep it isolated
  so the libzypp/libsnapper linkage is concentrated there and in the txn module.
- `[spec]` `compute-drift` performs no I/O: a pure comparison of two in-memory
  `Manifest` values. Keep it free of libzypp, filesystem, and process calls.
- `[spec]` `resolve-format` is the single authority for serialisation choice; route
  every read and write through it. No per-call-site format logic.

### CLI contract and errors

- `[spec]` Options are `key=value`, parsed by the tool; bare words are verbs;
  accept options in any position (do not reject options after the verb).
  Environment-variable control is forbidden (the debug env var is a trace gate
  only, not control).
- `[spec]` All CONFIG knobs are also accepted as `key=value` options. Command-line
  overrides preset.
- `[spec]` `[changed-0.5.1]` `version` and `help` are bare-word global commands and
  exit 0; `--version`/`--help`/`-h` are tolerated aliases only; no option uses
  POSIX `--flag`. Bare invocation prints usage to stdout and exits 0 (discovery,
  never converges). Unknown verb/option/value or missing value -> usage to stderr,
  exit 2.
- `[spec]` Internal functions return errors to their caller; only the verb layer
  maps to a process exit code. Model a `Diagnostic { severity, domain, message }`
  with `domain` in {packages, repositories, services, files, manifest,
  transaction, invocation}. `[recommended]` Represent operation results as either
  a small `Result`-like type or exceptions caught at the verb boundary; do not let
  a libzypp exception escape as an uncaught throw, translate it to a `Diagnostic`
  with the right domain. Exit codes: 0 success; 1 logical failure; 2 invocation
  error (full list per spec).

### Manifest model and serialisation

- `[spec]` Declarable Machinery subset (packages, repositories, services,
  config_files), `ScopeWrapper { _attributes, _elements }`, underscore field
  names. Emit those exact keys via jsoncpp/yaml-cpp regardless of C++ member names.
- `[spec]` Absent vs empty scope is semantic: model a declarable scope as
  `std::optional<ScopeWrapper<...>>`. `std::nullopt` = absent (unmanaged); a present
  wrapper with empty `elements` = reconcile-to-empty. Do not collapse them. (Take
  care that the JSON writer omits a `nullopt` scope entirely rather than writing
  `null` or an empty object.)
- `[spec]` `resolve-format`: explicit `format=` wins, else operative file extension,
  else `manifest-format` default. `[changed-0.5.0]`/`[changed-0.5.2]` describe
  output follows `resolve-format(out)`; do not hardcode JSON.
- `[spec]` The applied record is ALWAYS canonical JSON regardless of input format.
- `[spec]` `desired_sha256` is the SHA256 of a canonical serialisation of the parsed
  data model (format-independent). `[recommended]` Define canonical concretely and
  apply it for the hash: keys sorted, compact separators, UTF-8, `_elements` sorted
  by identity key (packages by name+arch, repositories by alias, services by name,
  config_files by path). jsoncpp's `StreamWriterBuilder` can be configured for
  compact output; sort containers yourself before writing, because jsoncpp does
  not guarantee key order for you in the way the canonical form requires. The
  on-disk `applied.json` may be pretty; the HASH is over the compact canonical
  form. For SHA256, use a small vetted routine or a system crypto library already
  in the dependency set; record the choice.
- `[spec]` YAML safe profile: a non-executing loader, no arbitrary/executable tags,
  bounded or disabled anchor/alias expansion, single document only, explicit typing
  per schema (no implicit coercion such as `NO` -> false or `1.10` -> float).
  `[recommended]` With yaml-cpp, load into a `YAML::Node` and walk it: reject any
  node with a non-default tag, reject documents beyond the first, and read scalar
  values with explicit `as<std::string>()` typing per the schema rather than
  relying on implicit conversions; yaml-cpp's anchor/alias handling must be bounded
  (reject or cap expansion). A YAML input needing a disabled feature is a manifest
  error. Record exactly how each safe-profile constraint is enforced in
  `TRANSLATION_REPORT.md`.

### Reading actual state, empty-scope rule, filesystem object model

- `[spec]` `[changed-0.5.2]` Repositories actual state from `/etc/zypp/repos.d/*.repo`
  (INI). libzypp can enumerate repositories; read the on-disk repo files (no
  network refresh, no privileged cache). A scope source that cannot be read is
  NEVER an empty scope: `on-unreadable=error` (default) errors naming the source,
  `warn` omits with a diagnostic; genuinely-empty readable scopes are omitted.
- `[spec]` `[changed-0.6.2]` Filesystem object model. Walk `/etc` (and the
  scope=full trees) recursively, classifying each entry by its own type WITHOUT
  following symlinks. In C++: `std::filesystem::directory_iterator` with
  `symlink_status()` (NOT `status()`, which follows links), or `lstat(2)` directly;
  classify into regular file / symlink / directory / other. Read a symlink target
  with `std::filesystem::read_symlink` and store it VERBATIM (do NOT
  `canonical`/`weakly_canonical`/normalise it; verbatim also keeps chroot-relative
  targets correct). Hash regular files only; descend directories and emit nothing
  for them (traverse-only); skip special files (anything not file/symlink/dir). A
  directory, symlink, or special file is NEVER an unreadable-source error. (The
  original Go crash was reading a directory as a file; classify first. In C++ the
  equivalent trap is calling `status()`/opening a path before classifying, or using
  `std::filesystem::file_size`/an ifstream on a directory.)
- `[spec]` File records carry a verbatim `target` field (symlink target, "" for
  non-links), sha256 only for regular files, per the TYPES consistency rules.
- `[spec]` In `compute-drift`, type is part of identity: differing type -> modified;
  same type compare sha256 (file) or target (link). A declared entry absent from
  actual is treated as matching.
- `[spec]` Hardlinks: treat per path by content+type; do not detect or preserve
  hardlink identity (out of scope for v1).
- `[spec]` config_files inspection is bounded to `/etc`; never read/hash/verify
  outside it and never run a whole-system verification. Difference-reporting (a
  verifier returning non-zero because it found changes) is the normal result, not
  an unreadable source.

### Transaction binding

- `[spec]` Abstract: `acquire-transaction-context` resolves auto|external|internal
  and yields a context with a writable root. Keep the binding isolated in the txn
  module, layered on libsnapper. The convergence path is identical regardless of
  binding. Unit enablement under a root uses offline enablement; do not rely on
  first-boot preset evaluation.
- `[spec]` `[reserved-0.7.0]` converge-files does NOT yet create/update/remove
  symlinks or handle type transitions; reserved for the apply milestone. When
  implemented: a declared type "link" is converged by its target; a declared-vs-
  actual type mismatch at a path is a HARD ERROR that aborts the transaction (no
  silent destructive rewrite, no deleting a directory tree to write a file).

### Spec-hash embedding and packaging

- `[spec]` Embed the spec SHA256 in: source headers, `version` output,
  `TRANSLATION_REPORT.md` (`Spec-SHA256:`), the RPM spec, the DEB control
  (`X-PCD-Spec-SHA256:`), the Containerfile label, and the CMake project metadata.
  `[recommended]` Generate a `meta.hpp` (or a configured header via CMake
  `configure_file`) holding the version and hash; `version` prints
  `zypper-declarative <version> spec:<sha256>`.
- `[spec]` Surfaced as a zypper subcommand (`/usr/lib/zypper/commands`) and
  invocable directly. OBS package; no curl install. RPM spec: `BuildRequires:
  gcc15-c++` on SLE 15, plus `libzypp-devel`, `libsnapper-devel`, `jsoncpp-devel`,
  and the right `libyaml-cpp` devel for the SP; dynamic runtime requires resolved
  by the linker. SIGTERM/SIGINT clean exit; an interrupted `apply` discards the
  transaction.

### Testing boundary

- `[pcd]` Black-box tests invoke the built binary and assert on stdout, stderr, and
  exit code; they do not link the internals. The v0.5.x/0.6.x examples (bare
  `version`/`help`, `describe out=...yaml`, offline `verify`/`diff`, `/etc`
  directory traversal, symlink-verbatim, special-file skip, type-transition drift)
  are black-box assertions. The symlink and fifo filesystem cases are constructible
  under a synthetic root without root privilege; cover them rather than leaving
  them code-review-only.

---

## Do NOT carry these over from an existing C++ implementation (spec v0.5.0-v0.6.2)

1. describe output ignoring the `out` extension (now follows `resolve-format`).
2. emitting an empty scope for an unreadable source (read repos.d; never empty for
   unreadable; omit genuine-empty).
3. bare `version`/`help` exiting 2 (now canonical, exit 0; flags are aliases); bare
   invocation exiting non-zero (now usage + exit 0).
4. per-call-site format selection (now `resolve-format`).
5. rejecting options that follow the verb; accept key=value in any position.
6. treating a verifier's non-zero "differences found" exit as an unreadable source.
7. a whole-system package verification; bound to `/etc` (scope=full trees only
   under scope=full).
8. scope=full engaged by default, or its observational scopes fed into convergence
   or written to the applied record.
9. `verify` requiring an applied record when a reference manifest is supplied;
   with `manifest-path`/`state-path`, `verify`/`diff` are pure offline comparisons.
10. `apply` accepting a manifest with a non-empty observational scope (reject it).
11. classifying a path by following symlinks or by opening it before an lstat;
    use `symlink_status`/`lstat`, traverse dirs, record symlink targets verbatim,
    skip special files, never read a directory as a file.
12. comparing config files by content hash alone; type is part of identity.
13. linking librpm or exec-ing `rpm`/`zypper`/`snapper`; link libzypp and
    libsnapper as libraries (the one allowed exception is a single libsnapper
    operation with no stable cross-SP API, via OSCommandRunner, explicitly noted).
14. a static binary or vendored dependencies; link the distro shared libraries
    dynamically and build per-SP in OBS.

## Slots to fill from an existing C++ implementation (if any)

- `[extract]` actual source tree and CMake structure, the diagnostic/error type,
  the SHA256 routine chosen, how the YAML safe profile is enforced with yaml-cpp,
  the CMake discovery mechanism for each library and the detected versions, the
  OBS spec and per-SP handling, and the spec-hash injection mechanism.

---

## Changelog

- 2026-06-01: Initial C++ decisions hints at spec v0.6.2. Records the verified
  SLE 15 SP7 / SLE 16.0 library bindings (libzypp for all package/rpmdb work, NOT
  librpm and NOT exec rpm; libsnapper with the soname-5-vs-7 per-SP caveat;
  jsoncpp for JSON; yaml-cpp with the 0.6-vs-0.8 caveat), the locked settings
  (C++17, CMake, dynamic linking, g++-15 on SLE 15 / default GCC 15 on SLE 16,
  libraries-not-exec), and the same architecture and "do NOT carry over" list as
  the Go and Rust decisions files expressed in C++ and these libraries' terms.
