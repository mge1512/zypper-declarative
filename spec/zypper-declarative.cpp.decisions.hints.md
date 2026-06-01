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
  what libzypp gives you. (But do not reach for boost GRATUITOUSLY either: use
  `std::filesystem` for the `/etc` walk, not boost.filesystem, and jsoncpp for
  JSON, not boost.json. boost is a transitive dependency, not a toolkit to add
  surface from.)
- `[changed-0.6.4]` Pristine refinements (the C++ build was mostly correct here,
  these keep it correct and converge it with Go/Rust): judge each `/etc` path
  INDEPENDENTLY against its own owning package, and never collapse a symlink with
  the file it points to (`/etc/pam.d/common-auth` owned by `pam` and
  `common-auth-pc` owned by `pam-config` are separate judgements; suppressing the
  pristine link must not suppress the target, and the target is judged against its
  own owner). A symlink is pristine iff its TARGET matches the package-recorded
  target (do NOT compare a symlink's mode); the first C++ build over-emitted some
  pristine distro symlinks (`/etc/X11/xim.d/*/40-ibus`), which this rule suppresses.
  `package_name` is the BARE name (`openssh-server`), which libzypp gives directly
  (do not append version/arch). Determine ownership and baseline for the whole
  enumerated `/etc` set in a single libzypp pass (in-process; this naturally
  satisfies the spec's bulk-lookup requirement, no per-file work).
- `[recommended]` Forward, use libzypp MORE rather than adding breadth: enumerate
  repositories through libzypp's repo API against `<root>`'s zypp configuration
  rather than hand-parsing `/etc/zypp/repos.d/*.repo`, so the tool reads repos the
  way zypp itself does (one interpretation, one trust boundary, chroot-safe because
  it reads files under the given root). This replaces a bespoke INI parser.
- `[verified]` `[recommended]` SHA256 via the platform crypto library: link
  **libcrypto** (OpenSSL 3), which is already in libzypp's dependency closure,
  rather than vendoring a hash routine. OpenSSL 3 devel is present in a fully
  supported core module on BOTH service packs: `libopenssl-3-devel` 3.2.3 on SLE 15
  SP7 (Basesystem Module) and 3.5.0 on SLE 16.0. Build against OpenSSL 3 on both;
  the only difference is a minor version (3.2.3 vs 3.5.0), same API, so unlike
  yaml-cpp and libsnapper there is NO major-version/soname split to code around for
  this dependency. (SLE 15 SP7 also still carries OpenSSL 1.1 in the Certifications
  and Dev Tools modules, but there is no reason to target it; use OpenSSL 3.)
- `[recommended]` `[reserved-0.7.0]` libsolv (verified: `libsolv-devel` on both
  SPs, and already a libzypp transitive dependency) is the right tool for
  dependency RESOLUTION when the `apply`/transaction path is built, computing the
  package transaction against the rpmdb under `<root>`. It is chroot-safe and not
  needed for the read/describe path; adopt it at the apply milestone, not now. Do
  not add it for describe.
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

### `[verified]` Services: offline `systemctl --root`, NOT libsystemd/sd-bus

- The `services` scope is unit ENABLEMENT state (enabled / disabled / masked). The
  first C++ build deferred this and emitted an empty (hence omitted) services
  scope, while the Go build read 214 services; that divergence is a bug to close.
  The services reader is mandatory in `describe-actual-state`, not a deferral.
- Read enablement OFFLINE under the context root, via
  `systemctl --root <root> is-enabled <unit>` and
  `systemctl --root <root> list-unit-files`, through the `OSCommandRunner`. Do NOT
  use libsystemd / sd-bus for this scope. Reason: sd-bus talks to the running
  system's PID 1 and cannot answer "what is the enablement state under THIS other
  root", but the tool's whole model is rooted operations (describe `root=/mnt`,
  convergence into a mounted snapshot), so the D-Bus API is the wrong tool here.
  The file/CLI route is also what the Go build does correctly.
- `systemd-devel` is present on SLE 16 but was NOT confirmed on SLE 15 SP7 in the
  package lists, a second reason not to take a build-time libsystemd dependency
  for this. (libsystemd/sd-bus would only earn its place for a future LIVE
  runtime-state feature, active/failed/substate, which the spec does not ask for
  and which is not declarable.)
- Purely-static units are omitted (not declarable). Normalise state to exactly
  enabled / disabled / masked.

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

## Do NOT carry these over from an existing C++ implementation (spec v0.5.0-v0.6.4)

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
15. (v0.6.3) emitting an empty/omitted `services` scope or deferring the services
    reader; it is mandatory, read offline via `systemctl --root` (see above). A
    deferred-empty scope silently drops declarable state.
16. (v0.6.3) emitting more config_files than the package-pristine rule allows;
    suppress package-pristine `/etc` entries (owned and matching the
    package-recorded digest, target, mode, owner, group via libzypp), emit only
    unpackaged or changed-from-package. Do not over-emit pristine files.
17. (v0.6.3) serialising a scope's `_attributes` as `null` (or YAML `~`); it is
    always a JSON object, empty `{}` when there are no attributes. Quote
    string-typed scalars in YAML (`mode`, `sha256`, `target`) so `mode: "0600"`,
    not `mode: 0600`.
18. (v0.6.3) dropping the version from `meta.generator`; emit
    `zypper-declarative <version>` so the string matches other implementations of
    the same spec version.
19. (v0.6.4) collapsing a symlink with its target file, or comparing a symlink's
    mode for pristine-ness (target-match only); the first build over-emitted
    pristine distro symlinks. Judge each path independently against its own owner.
20. (v0.6.4) appending version/arch to `package_name` (bare name only; libzypp
    gives the name directly). Adding boost surface gratuitously (use std::filesystem
    and jsoncpp); hand-parsing repos.d instead of using libzypp's repo API;
    vendoring a SHA256 routine instead of linking libcrypto.

## Slots to fill from an existing C++ implementation (if any)

- `[extract]` actual source tree and CMake structure, the diagnostic/error type,
  the SHA256 routine chosen, how the YAML safe profile is enforced with yaml-cpp,
  the CMake discovery mechanism for each library and the detected versions, the
  OBS spec and per-SP handling, and the spec-hash injection mechanism.

---

## Changelog

- 2026-06-01: Updated to spec v0.6.4. Added the pristine refinements (independent
  per-path judgement, symlink-pristine-by-target-only - the first build over-emitted
  pristine distro symlinks, bare `package_name`, single libzypp pass satisfying the
  bulk-lookup rule) and forward libzypp-depth guidance: enumerate repos via libzypp
  rather than hand-parsing repos.d, link libcrypto for SHA256 (OpenSSL 3 on both
  SPs, `libopenssl-3-devel` 3.2.3 on 15 SP7 / 3.5.0 on 16.0, no major-version
  split), and adopt libsolv for dependency resolution at the v0.7.0 apply
  milestone (verified on both SPs, already transitive). Also: do not add boost
  surface gratuitously. Items 19-20 added to the do-not-carry list.
- 2026-06-01: Updated to spec v0.6.3, after a Go-vs-C++ describe comparison on a
  live host. Added the verified services binding (offline `systemctl --root` via
  OSCommandRunner, NOT libsystemd/sd-bus, with the rooted-operation rationale),
  since the first C++ build deferred services to an empty/omitted scope while Go
  read 214. Added the v0.6.3 do-not-carry items: services mandatory; suppress
  package-pristine `/etc` entries via libzypp (the C++ build was already correct
  here, Go was not, this keeps it correct); `_attributes` always `{}` not null and
  YAML string scalars quoted; and `meta.generator` carries the version.
- 2026-06-01: Initial C++ decisions hints at spec v0.6.2. Records the verified
  SLE 15 SP7 / SLE 16.0 library bindings (libzypp for all package/rpmdb work, NOT
  librpm and NOT exec rpm; libsnapper with the soname-5-vs-7 per-SP caveat;
  jsoncpp for JSON; yaml-cpp with the 0.6-vs-0.8 caveat), the locked settings
  (C++17, CMake, dynamic linking, g++-15 on SLE 15 / default GCC 15 on SLE 16,
  libraries-not-exec), and the same architecture and "do NOT carry over" list as
  the Go and Rust decisions files expressed in C++ and these libraries' terms.
