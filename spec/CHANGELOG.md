# Changelog: zypper-declarative.spec.md

Version history for the zypper-declarative specification. This file lives
beside the spec but is NOT part of it: it is not read by the translator and
is not covered by the embedded spec SHA256. Detailed rationale for a change
(including any build failure that motivated it) belongs here and in the git
commit messages, not in the normative spec text.


- 2026-06-01: Version 0.6.5. Pinned the config_files emission test as an explicit
  REPRODUCIBILITY criterion and resolved the `%ghost` and type-mismatch cases that
  a three-way Go/C++/Rust comparison left ambiguous (the `/etc/pam.d/common-auth`
  and `common-auth-pc` pair). The rule: emit an `/etc` path exactly when a fresh
  install of its owning package (or no owning package) would NOT reproduce its
  current on-disk state, equivalently, when it would have to be carried in a
  tarball to reconstruct this machine's `/etc` on freshly-installed packages. This
  subsumes the earlier pristine rule and settles the hard cases. (1) TYPE MISMATCH
  is emitted: where the package records a regular file but the disk holds a symlink
  (pam ships `common-auth` as a 462-byte `%config(noreplace)` file, but pam-config
  has replaced it with a symlink to `common-auth-pc`), the symlink is emitted,
  because a fresh install would lay down the file, not the link. (2) GHOST WITH
  CONTENT is emitted: a `%ghost` path the package ships no content for, but which
  holds real content on disk (pam-config ships `common-auth-pc` as a 0-byte
  `%ghost`, but on disk it holds the real 462-byte PAM configuration), is emitted
  with its actual content and digest, because a fresh install reproduces no content
  for it. (3) EMPTY-GHOST-MATCHING-EMPTY is suppressed: a ghost that is 0 bytes on
  disk and 0 bytes in the recorded baseline carries nothing to reproduce. The
  determination now explicitly requires the package-recorded file flags, in
  particular the ghost marker, alongside the digest, link target, mode, owner, and
  group; an implementation must obtain the ghost marker (libzypp exposes it via the
  rpm header file info; an exec-based implementation must query the file flags) and
  must not ignore it. The independent-per-path and symlink-target rules from v0.6.4
  are unchanged and reinforced by the worked pam example. Added invariants and
  examples for the reproducibility criterion, type-mismatch emission,
  ghost-with-content emission, and empty-ghost suppression. Implementation note:
  an implementation that emitted the type-mismatch symlink but dropped the
  content-bearing ghost target (or vice versa) is incomplete under this version;
  both are emitted, judged independently against their own owning packages.
- 2026-06-01: Version 0.6.4. Refined the config_files pristine rule after a
  three-way Go/C++/Rust describe comparison on a live host exposed disagreements,
  and added a performance property. (1) Each `/etc` path is now judged
  INDEPENDENTLY against its own owning package, and a symlink is never collapsed
  with the file it points to: suppressing a pristine symlink must not suppress its
  target file, the target is evaluated on its own against its own owner (often a
  different package), and the symlink is never dereferenced. The worked case is
  `/etc/pam.d/common-auth` (symlink owned by `pam`) versus
  `/etc/pam.d/common-auth-pc` (file owned by `pam-config`); implementations had
  variously collapsed these. (2) Pristine is now type-specific: a symlink is
  pristine when its TARGET matches the package-recorded target (its mode is not
  compared, symlink permissions not being meaningful or tracked), while a regular
  file still requires digest, mode, owner, and group to match. An owned distro
  symlink with the package's target (such as the many `/etc/X11/xim.d/*/40-ibus`
  links) is therefore suppressed. (3) `package_name` is pinned to the bare package
  NAME (`openssh-server`), never the full name-version-release-arch identifier;
  one build had emitted the full NEVRA. (4) Ownership and baseline determination
  must be performed in BULK (a bounded number of queries or one database pass), not
  once per path; this is a performance requirement, not a change to which entries
  are emitted, and it addresses a real per-file-subprocess slowdown observed in an
  exec-based implementation while leaving the result identical and the work bounded
  to `/etc`. Added invariants and examples for the independent judgement, the
  symlink-target rule, the bare name, and pristine distro symlinks.
- 2026-06-01: Version 0.6.3. Clarifications driven by a cross-implementation
  comparison: running `describe` from the Go and the C++ builds on the same host
  exposed three divergences, two of which were spec ambiguities. (1) The
  package-pristine rule is now pinned: an `/etc` entry is emitted only if it is
  unpackaged or changed-from-package, and a package-pristine entry (owned, and
  matching the package-recorded content digest, target, mode, owner, and group) is
  suppressed; ownership is determined through the native package database (libzypp
  on SUSE) and must not be defaulted to unpackaged because a lookup was skipped.
  This resolves a divergence where one build correctly suppressed pristine files
  via libzypp while another mislabelled package-owned files as unpackaged and
  over-emitted them. (2) Every scope's `_attributes` is now required to serialise
  as a JSON object (empty `{}`, never `null`), for Machinery consistency; the
  ScopeWrapper type and the JSON example were corrected. (3) `meta.generator` is
  now normative as program-name-and-version, so independent implementations of the
  same spec version emit the same generator string (one build had dropped the
  version). Also stated explicitly: digests are SHA256, with md5/sha1 not used
  except for reading a legacy recorded digest during a comparison; and the
  config_files record is documented as a declarable Machinery superset
  (Machinery-consistent envelope and type semantics, extended with the
  convergence fields and SHA256). The missing services scope in one build was a
  pure implementation gap, not a spec change, and is addressed in the
  language-specific decisions hints. Added invariants and examples for the pristine
  rule and the attributes-object rule.
- 2026-06-01: Version 0.6.2. Fixed a `describe` crash surfaced on a live host
  ("files: unreadable scope source: /etc: read /etc/ImageMagick-7: is a
  directory") and, with it, underspecified handling of non-regular-file entries
  across the read and compare paths. The `/etc` walk (and the scope=full walk over
  `/usr` and `/boot`) now recurses into directories and classifies each entry by
  its own type without following symlinks: regular files are hashed (type "file"),
  symlinks record their verbatim target (type "link", never dereferenced, neither
  resolved nor normalised, which also keeps chroot-relative targets correct),
  directories are traversed but not emitted (traverse-only), and special files
  (device, fifo, socket) are skipped. Encountering a directory, symlink, or
  special file is explicitly never an unreadable-source error. Type is now part of
  a config file's identity in `compute-drift`: a path whose on-disk type differs
  from the declared type is modified regardless of content; a symlink is compared
  by target, a regular file by sha256. The file records gained a verbatim `target`
  field with type/sha256/target consistency rules. Hardlinks are treated as single
  files by content and type per path (hardlink identity out of scope for v1). The
  converge-side type semantics (creating, updating, and removing symlinks, and
  treating a declared-versus-actual type transition as a hard error that aborts the
  transaction) are noted as reserved for the milestone that exercises `apply` on a
  live host; this version covers the read and drift side, which is testable
  offline. Added invariants and examples (directory traversal, verbatim symlink
  recording, special-file skip, type-transition drift).
- 2026-05-29: Version 0.6.1. Added offline two-file comparison and a guard against
  applying a raw describe dump, both motivated by the architect baseline-authoring
  workflow. `verify` now accepts `manifest_path` as the reference (used instead of
  the applied record, and not requiring one to exist) and `diff` now accepts
  `state_path` as a captured actual state; with both files supplied, `verify` and
  `diff` are pure comparisons that read neither the live system nor any applied
  record, which serves air-gapped and audit review (capture state on one host,
  compare against an intended manifest on another). `compute-drift` was already
  pure, so this is a routing change at the verb layer. Separately,
  `load-desired-manifest` now rejects a desired manifest that carries a non-empty
  observational scope (changed_managed_files or unmanaged_files), so a raw
  `describe scope=full` dump cannot be mistaken for a baseline and silently
  half-applied; it must first be edited into intent. Added invariants and examples
  for both.
- 2026-05-29: Version 0.6.0. Added an opt-in full-system integrity scan, mirroring
  the old Machinery and sitar behaviour, for the case where `/usr` is not
  guaranteed immutable. A `scope` option (`etc` default, `full`), accepted on
  `describe` and `verify` only, controls it. Under `scope=full`,
  `describe-actual-state` additionally scans the package-managed trees outside
  `/etc` (`/usr`, the usr-merge roots `/bin` `/sbin` `/lib` `/lib64`, and `/boot`;
  `/opt` and the virtual, runtime, and mutable-data trees excluded; keep-list
  honoured) and emits two observational scopes: `changed_managed_files` (packaged
  files changed in place) and `unmanaged_files` (out-of-band additions no package
  owns). These are observational, not declarable: `compute-intent-diff` and
  convergence ignore them, and they never appear in a desired manifest or applied
  record (matching the existing rule that observational scopes are ignored).
  `compute-drift` surfaces them under `scope=full` as two new drift categories
  (`managed_files_modified`, `unmanaged_files_present`), so `verify scope=full` is
  an integrity audit of the package-managed trees against the package baseline, in
  addition to the declaration check. The full scan is expensive and never engaged
  by default, including on a mutable `/usr`; the default `scope=etc` is unchanged
  bounded behaviour. Scope keys use the underscore form (identical to Machinery's
  JSON keys; the hyphenated forms were Machinery's CLI scope identifiers). Added
  the `scope` CONFIG knob, types, invariants, and examples.
- 2026-05-29: Version 0.5.2. Fixed a `describe` defect surfaced by the build:
  `describe` aborted with "files: unreadable scope source: rpm config-file
  verification: exit status 1". The package-verification mechanism returns
  non-zero precisely when it finds changed files, which on any real system is the
  normal case, and the reader misclassified that as an unreadable source.
  "Unreadable" is now defined precisely (a genuine access or I/O failure to read a
  required source), and a verification or query command exiting non-zero to report
  differences, or returning an empty result, is explicitly a normal successful
  outcome, never unreadable. In the same step, the config_files reader is now
  bounded to `/etc`: it inspects only `/etc`, consults package metadata only for
  the `/etc` files it enumerates, and never performs a whole-system package
  verification. This is both correctness (the reader only ever manages `/etc`) and
  the performance fix for the slow full-system verification, since the cost now
  scales with the size of `/etc` rather than the installed package base. Added
  matching invariants and two examples (config_files bounded to `/etc`;
  verification differences are not an unreadable source).
- 2026-05-29: Version 0.5.1. Fixed a CLI-surface defect surfaced by the v0.5.0
  build: `zypper declarative version` returned "unknown verb" (exit 2) while only
  the `--version` flag worked, which contradicts the cli-tool template
  (CLI-ARG-STYLE: bare-words supported, POSIX `--flag` forbidden for new options)
  and its milestones-hints M0 gate (`version` and `help` as bare words must exit
  0). The implementation was a faithful translation of the v0.5.0 spec, which
  listed only the five behavior verbs and provided version/help solely as POSIX
  flags; the fix is in the spec. `version` and `help` are now the canonical
  bare-word global commands (exit 0), with `--version`, `--help`, and `-h` kept as
  tolerated aliases (the spec-hash convention still references `--version`).
  Updated the verb listing, the global-behaviour section, the M0 and 0.1.0
  acceptance criteria to exercise the bare-word forms, an invariant, and added
  examples for the version and help bare words and the version flag alias.
- 2026-05-29: Version 0.5.0. Three fixes surfaced by a first implementation, all
  closed in the spec rather than the code. (1) Defined the top-level CLI contract:
  bare invocation and `--help` print usage to stdout and exit 0 (discovery, not an
  error; never runs a default verb), `--version` exits 0, and an unknown verb,
  option, value, or missing value exits 2 to stderr; documented that all CONFIG
  knobs are also accepted as key=value options. (2) Centralised serialisation
  choice in a new internal behaviour `resolve-format` (explicit `format=` option,
  else the operative file extension, else the `manifest-format` default) and
  routed every manifest read and write through it, so output now honours the `out`
  extension (`describe out=...yaml` writes YAML) symmetrically with input, on
  manifest-path, state-path, and describe out alike; `verify` gained a `format`
  option for the state dump. (3) Pinned the repositories actual-state source to
  the on-disk `/etc/zypp/repos.d` files (readable without elevated privilege, no
  network refresh or privileged cache), and fixed a latent footgun: a scope source
  that cannot be read is never represented as an empty scope. `describe-actual-state`
  now errors on an unreadable source by default, or omits it with a diagnostic
  under `on-unreadable=warn`, and omits genuinely-empty scopes so a bootstrapped
  manifest leaves them unmanaged rather than asserting deletion. Internal callers
  (apply, diff, status, verify) always use the strict reader. Added `on-unreadable`
  and `applied-root` CONFIG knobs.
- 2026-05-29: Version 0.4.0. Added YAML as an opt-in serialisation of the manifest
  alongside the canonical JSON, on a `format=` switch (and `manifest-format` CONFIG
  default, and file-extension inference), for environments that author OS state in
  YAML such as a ZARF-centric workflow. The manifest is now framed explicitly as a
  typed data model with JSON and YAML as serialisations of it; the data model and
  all logic are unchanged. `load-desired-manifest` gained format selection and a
  safe YAML profile (no code-executing tags, bounded aliases, single document,
  explicit typing); `describe` gained a `format=` output option. Manifest identity
  (`desired_sha256`) is now the hash of the canonical serialisation of the parsed
  data model, so JSON and YAML expressions of the same manifest are recognised as
  identical and idempotence holds across a format switch. The applied record stays
  canonical JSON regardless of input format, preserving Machinery readability;
  YAML breaks Machinery and sitar compatibility on the output side only, which is
  accepted for YAML-requesting customers.
- 2026-05-29: Version 0.3.0. Internalised the system-description capability so no
  separate collector program is required. Added a `describe` verb and a single
  internal live-state reader `describe-actual-state` that reads the four
  declarable scopes into the shared Machinery format. Refactored `compute-drift`
  into a pure comparison over two Manifest documents (actual versus reference),
  with the live actual state now produced by `describe-actual-state` or supplied
  as a dump; `verify`, `status`, and `apply` route their actual-state reads
  through this one path. An external Machinery-format producer such as sitar is
  now optional and interchangeable rather than a dependency. Added a `package_name`
  field to ManagedFileRecord (Machinery field) governing the files_extra rule
  (only unpackaged undeclared /etc files count as extra). Reaffirmed
  language-neutrality explicitly.
- 2026-05-29: Version 0.2.0. Adopted the SUSE Machinery system-description JSON
  format (format_version 1) for the desired manifest, the applied record, and the
  actual-state input. Manifest became the declarable subset of that schema using
  the ScopeWrapper idiom (packages, repositories, services, config_files). The
  package lock became a fully resolved packages scope rather than a bespoke NEVRA
  string type. Repository pinning became declarative via an in-band repositories
  scope. Added absent-versus-empty scope semantics, a content_ref for declared
  file content, and a worked JSON manifest example.
- 2026-05-29: Version 0.1.0. Initial specification of `zypper-declarative`, a
  reconciling converger surfaced as a zypper subcommand. Verbs over internal
  convergence behaviours. Two-diff model (intent diff for deletions, drift diff
  for verification). Applied record stored under /usr within the generation. /etc
  treated as a nested btrfs subvolume with `etc.syncpoint` excluded. Transaction
  binding abstracted between an external mechanism and the zypper-internal
  mechanism, with the decision deliberately left open. Secrets, kernel cmdline,
  and sysctl domains reserved for a later Version.

## Version 0.6.10

- 2026-06-08: Added two transaction modes, `snapshot` and `plain`, to the
  `TransactionMode` axis (was `auto | external | internal`, all transactional-
  update flavours) and taught `auto` to resolve by substrate across all five.
  Background: a caller on a throwaway root (an ephemeral build VM whose overlay is
  discarded after the run) has no use for a sealed boot snapshot, and a normal
  btrfs SLES host wants the stock snapper pre/post bracket rather than a
  transactional-update generation; neither was expressible. (1) `snapshot`
  brackets the whole apply in a snapper pre/post pair on the running root: changes
  are live immediately, the pair is the rollback unit (`snapper rollback`), and no
  boot target is set. This is the stock SLES-12+ btrfs zypper behaviour, now a
  first-class mode. (2) `plain` applies in place on the running root with no
  snapshot and no transactional wrapper; it has no undo unit, so a mid-run failure
  leaves the partial changes already made (non-atomic by design), which is
  acceptable for a throwaway root or an offline bake. (3) `auto` now resolves by
  substrate: inside a transactional-update snapshot -> external; on a
  transactional-update host -> internal; on a non-transactional btrfs root with
  snapper -> snapshot; otherwise -> plain. The transactional modes are unchanged,
  and the resolved mode governs only the undo unit and finalisation, never which
  scopes are converged or how (the convergence code path is identical across
  modes). Touched TYPES (TransactionMode, TransactionContext),
  acquire-transaction-context (substrate resolution and the snapshot/plain
  bindings), the apply and init finalisation steps and their postconditions (the
  plain non-atomicity is stated explicitly), CONFIG (transaction-mode values;
  activation-policy noted as transactional-only), and added invariants and
  examples for snapshot, plain, and the auto resolution. This is the spec-side
  counterpart of kvm-manager's dev-run, which invokes
  `zypper declarative apply mode=plain` inside its ephemeral overlay. An offline
  target (a root= input for in-place baking of a mounted tree) is deliberately NOT
  added here; it remains a possible later addition, orthogonal to the mode axis.

## Version 0.6.9

- 2026-06-02: Fixed symlink classification (a real bug that shipped in the v0.6.8
  C++ build) and made `init` force `on-unreadable=warn`. (1) SYMLINK CLASSIFICATION:
  only symlinks that are actually in the alternatives system (under
  `/etc/alternatives/`, or listed as master or slave in a `/var/lib/alternatives/`
  admin file) are resolved against the alternatives database. Every other symlink,
  including `/etc/crypto-policies/back-ends/*.config` (crypto-policies links into
  `/usr/share/crypto-policies/<policy>/`), `/etc/motd.d/*`, and `/etc/issue.d/*`, is
  judged by the normal symlink target rule and is NEVER queried against
  update-alternatives; its lack of an alternatives entry is not an on_unreadable
  condition. The v0.6.8 build queried update-alternatives for these non-alternatives
  symlinks, producing 24 spurious "alternatives unreadable" warnings and aborting
  describe under the default error mode on /etc/motd.d/cockpit. On a default-policy
  system the crypto-policies back-end links point where the package set them and are
  now suppressed as pristine. Slave alternatives whose auto/best is indeterminable
  are emitted conservatively (resolving them via the master admin file is a permitted
  refinement, not required). (2) `init` now FORCES on_unreadable=warn for its live
  read (overriding the default and any error passed in), so onboarding a real machine
  never aborts on a protected root-only file or an indeterminable source; init is the
  only verb that overrides the knob. Added matching invariants and worked examples
  (crypto-policies suppression, init-forces-warn).
- 2026-06-02: Two further v0.6.9 clarifications (spec hash now
  aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3). (a)
  `compute-drift` packages_divergent: an empty identity field in a reference package
  element is a wildcard matching any actual value, so an empty-version "newest from
  repo" desired package does not manufacture false drift (the code already did this;
  now normative). (b) EXAMPLES note: examples driving a real describe or full scan are
  O(installed packages) and can take minutes on large systems; a harness must allow a
  generous timeout and must not kill a still-running long example.

## Version 0.6.8

- 2026-06-02: Exposed the `on-unreadable` knob CONSISTENTLY on every verb that reads
  live state. Previously only `describe` accepted it while `diff`, `verify`, and
  `apply` hard-coded `on_unreadable=error` on the same kind of live read via
  `describe-actual-state`, an inconsistency that surfaced on a real host: the
  `describe`-then-`diff` idempotence self-check could not pass unprivileged because
  the describe half could warn-skip a root-only file (`/etc/libaudit.conf`) but the
  diff half forced error and aborted. Now `describe`, `diff`, `verify`, and `apply`
  all accept `on-unreadable` (default `error`, unchanged safe default) and pass it to
  their internal live read, so an operator or a test reading live state unprivileged
  can pass `on-unreadable=warn`. Added the matching invariant. The default is
  unchanged; the knob is simply available on the three verbs that previously forced
  error.

## Version 0.6.7

- 2026-06-02: Added the `init` verb and fixed `diff` drift to compare against the
  desired manifest. (1) `zypper declarative init` onboards a machine in one command:
  it INCLUDES describe, opens a snapshot, writes the described current state as the
  applied-record baseline, and converges nothing (actual already equals the adopted
  state). After init, diff/verify reference the applied record, diff is clean, apply
  is a no-op until the manifest is edited. init also emits the manifest for the
  operator to edit. (2) `diff` now computes DRIFT against the desired manifest, not
  the applied record (the applied record is used only for the intent diff). This
  fixes a bug exposed on SL Micro, where /etc is largely unpackaged: diff against an
  empty applied record had reported the entire unpackaged /etc as "extra" drift. On a
  freshly-described unedited manifest, drift is now empty whether or not the machine
  is onboarded. `compute-drift`'s reference is clarified to be a desired manifest or
  an applied record (same schema); each caller states which it passes. NOTE: a
  packaged file legitimately showing package_name "" on SL Micro is correct (SL Micro
  does not package-own much of /etc); that was not a bug. Added invariants and
  examples for init onboarding/idempotence and diff-drift-against-manifest.

## Version 0.6.6

- 2026-06-02: Two additions to the config_files emission rule, both consistent with
  the reproducibility criterion. (1) GHOST SYMLINKS (the `/etc/alternatives/*` case):
  a `%ghost` symlink is judged by whether a fresh install would reproduce its TARGET,
  not by whether it "has content" (every symlink has a target). For alternatives, the
  reproducible target is the alternatives system's auto/best provider; a link pointing
  there is pristine and suppressed, a manually-set link (update-alternatives --set) is
  emitted as declarable intent. This resolved a real Go-vs-C++ divergence where C++
  emitted all ~287 default alternatives. (2) CONTENT-STORE POPULATION: when the
  `content-store` option is set, describe writes the bytes of every emitted
  regular-file record into the store content-addressed by SHA256
  (`<content-store>/sha256/<digest>`, deduplicated) and sets `content_ref` to
  `sha256/<digest>`; with no content store, describe stays read-only and `content_ref`
  is "". The manifest references content, it does not inline it; this is the first
  step toward an applicable manifest. Unreadable file content follows `on_unreadable`,
  never silent. Added invariants and examples for both.
