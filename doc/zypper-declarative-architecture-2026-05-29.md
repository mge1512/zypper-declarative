# Declarative SL Micro via `zypper declarative`: Architecture and Design

**Status:** Working architecture document (POC scoping)
**Date:** 2026-05-29
**Author:** Matthias G. Eckermann
**Substrate:** SL Micro 6.2 / SLES 16.1
**Companion artifact:** `zypper-declarative.spec.md` (PCD specification, cli-tool deployment, v0.4.0)
**Manifest format:** SUSE Machinery system description (`format_version` 1), declarable subset; JSON canonical, YAML optional

---

## 1. Purpose and scope

This document captures the architecture for turning SL Micro into a reconciling
declarative operating system without adopting Kubernetes, Rancher, or NixOS. The
goal is a system whose running state is a function of an authored, signed
declaration, where reapplying the declaration converges the system to it and
removes what the declaration no longer asserts.

The deliverable that makes this real is a tool, surfaced as `zypper declarative`,
that reads a declared manifest, computes the difference against the current
system, applies the difference inside a snapshot transaction, and records what it
did so the next run can reason about deletions. The tool is specified separately
in `zypper-declarative.spec.md`. This document is the rationale and the system
context around that spec.

In scope: the converger design, the manifest model and its shared Machinery
format, the JSON and YAML serialisations, the built-in `describe` state reader,
the `/etc` handling, the package/file/unit convergence domains, the transaction
abstraction, and the delivery paths.

Out of scope for the POC, deliberately deferred: secret material in the
declaration (WiFi PSKs, TLS keys), kernel command line as declared state under
the UKI / `sdbootutil` direction, and the central fleet-management product
integration beyond identifying the seam.

---

## 2. What "declarative" means here, and what it does not

A precise definition matters, because the term carries borrowed mechanisms that
are not part of the definition.

The minimal definition of a declarative OS needs only two things: state described
as data, and an engine that converges the system to that data idempotently,
including removing what the data no longer asserts. Nothing about how packages
are stored on disk is part of this.

Two properties are commonly bundled in from NixOS and should be set aside:

1. **Version coexistence** (`foo-1.2` and `foo-1.3` both present in a content
   addressed store) is how Nix implements cheap switchable generations. It is not
   part of declarativeness. SL Micro already provides many switchable generations
   through btrfs snapshots with exactly one version of each RPM present at a time.
2. **Bit-for-bit reproducibility by construction** is a property of Nix's total
   input pinning. RPM reaches reproducibility differently (see Section 11), but
   the absence of intrinsic pinning is not an absence of declarativeness.

A useful ladder:

| Level | Property | Where SUSE sits |
|---|---|---|
| 1 | Immutable + transactional (read-only `/usr`, snapshots, transactional apply) | SL Micro today |
| 2 | Image-based, first-boot declared | SL Micro + Elemental; SLES 16 + /usr-merge |
| 3 | Reconciling-declarative: authored spec + converge-and-remove-drift | this design's target |

There is no meaningful level above 3 for this purpose. NixOS sits at level 3 as
well; it merely reaches reproducibility intrinsically and uses coexistence for its
rollback. Neither is a higher tier of declarative.

The genuinely missing piece on the SLES 16.1 / SL Micro 6.2 substrate is the
converger: an authored state description plus an engine that reapplies it and
removes what is no longer declared. The state description is JSON. The engine is
`zypper declarative`.

---

## 3. The substrate already present

The design leans on mechanisms that ship today, which is why the novel code is
small and the rigor is in the discipline rather than the line count.

- **Read-only root, writable `/etc`.** Since transactional-update 5.0.0, `/etc`
  is a nested btrfs subvolume of the root filesystem, not an overlay. Older
  overlay-based systems are migrated automatically. SL Micro 6.2 is past that
  line.
- **Snapshots as generations.** Every change happens in a new btrfs snapshot;
  the running system is unaffected until reboot. Rollback is booting a prior
  snapshot. This provides the multi-generation property for free.
- **The transaction mechanism.** On SL Micro today this is `transactional-update`.
  In SLES 16.1 the transactional machinery is reported to be merged into zypper
  itself. This design treats the transaction boundary as an abstraction with two
  bindings and does not commit to either (Section 8).
- **Config layering.** NetworkManager and systemd both read from a `/usr/lib`
  (immutable, image-shipped) layer and an `/etc` (writable, user-or-admin) layer,
  with the writable layer taking precedence. This is the model for separating
  declared instance state from user-created instance state.
- **Signed supply chain.** OBS produces signed RPMs with provenance. The
  declaration and the tool both ride this chain rather than `curl`.

Two building blocks already inside transactional-update are directly reusable:
`create_dirs_from_rpmdb` (reconstructs directories from the rpmdb) and the
`/etc` sync-state logic (`sync-etc-state`, the three-way merge over the `/etc`
subvolume and its syncpoint).

---

## 4. The converger model: two diffs, not one

The single most important design insight is that convergence requires two
distinct comparisons, and the filesystem only gives one of them cheaply.

**Drift diff (actual vs declared).** The live system versus the declaration in the
applied record. This catches a rogue edit an admin made in a crisis. The tool
reads the live system into a Machinery document, the same format as the manifest,
via the built-in `describe` reader (Section 5.5), and compares it to the applied
record; the comparison itself is pure. For `/etc` the read uses the
subvolume-versus-syncpoint comparison (Section 6), so it stays cheap on btrfs.

**Intent diff (desired vs desired).** The new manifest versus the previously
applied manifest. This is the comparison that yields deletions, and the
filesystem cannot provide it, because nothing on disk has changed when a line is
merely dropped from the manifest.

Worked example: manifest v1 declares `/etc/bar.conf`; v2 drops that line. Nothing
on disk drifted, so a snapshot diff against the clean state is empty. The
filesystem will never tell you to delete `bar.conf`. Only comparing v2's declared
set to v1's declared set reveals it.

This is also why a snapshot branch-and-write does not suffice: a btrfs snapshot is
a copy-on-write clone that contains every file of its source, and writing only
adds or overwrites. Removal is never implied by branch-and-write. The previous
manifest must be retained and diffed against the new one.

The deletion rule that falls out is safe by construction. The deletion-candidate
set is exactly `(declared_old - declared_new)`. RPM-owned `/etc` files and
machine-identity files (machine-id, SSH host keys, the systemd random seed) were
never in the declared set, so they are never candidates. The tool only ever
deletes a file it previously declared. Walking the four file cases confirms it:
present in both with identical content is a no-op; present in both with differing
content is an overwrite (this is drift reversion); present in new only is a
create; present in old only is a delete. Nothing else is ever touched.

After a converge, the live `/etc` should equal the new manifest modulo the
keep-list. That equality is the post-condition and a filesystem-level assurance
that the system equals the declaration.

---

## 5. The manifest and its format

### 5.1 Desired spec versus applied record

Two different files, easy to conflate:

- **Desired spec (input).** What we want now. Authored by humans, signed,
  delivered to the host (Section 10). Transient input to a converge run.
- **Applied record (output, the ledger).** What the tool actually applied on the
  last successful converge, including the resolved packages scope (the lock).
  Written by the tool, immutable in the running system, and the source of truth
  for the next intent diff.

The intent diff compares the new desired spec against the applied record from the
current generation. After a successful apply, the new applied record is the
desired spec just applied, with its packages scope resolved to the lock, baked
into the new snapshot.

### 5.2 Location: `/usr/lib/zypper-declarative/`

The applied record lives at `/usr/lib/zypper-declarative/applied.json`, a single
document in the shared format whose packages scope is the resolved lock. The
reasoning pins the location almost uniquely:

- `/usr` is part of the root subvolume in the SL Micro layout (unlike `/etc`,
  `/var`, `/home`, which are separate subvolumes). Every root snapshot captures
  it, so a rollback to generation N restores N's applied record automatically.
  The previous manifest for the next intent diff is therefore always whatever sits
  in the currently-running `/usr/lib/zypper-declarative/applied.json`. The
  per-generation ledger property is free.
- `/usr` is read-only in the running system and writable during the transaction,
  which is exactly when the tool writes it. The commit seals it.
- It is FHS-correct: `/usr/lib/<subsystem>/` is the home for program-internal
  data, mirroring `/usr/lib/systemd` and `/usr/lib/os-release`.

Why not the alternatives: `/etc` is the surface the tool itself diffs and deletes
within, so placing the record there is self-referential and the record is meant to
be immutable, not mutable config. `/var` is shared across snapshots and would not
roll back, leaving the record describing a generation the system is no longer on.

The snapshot's snapper userdata carries the SHA256 of the applied record as an
index and integrity check (`snapper modify --userdata`). The document lives in
`/usr/lib`; the pointer lives in userdata.

### 5.3 What the record contains

A lean inventory expressed as the four declarable scopes, not raw file contents:
the resolved packages scope (the lock), the repositories scope (the pinned
sources), the services scope (declared unit enablement), and the config_files
scope (declared `/etc` files, each by path, metadata, and content hash). File
content is referenced, not inlined. The record holds what is needed to compute the
next intent diff and to verify the system, nothing more. Kernel cmdline and sysctl
are reserved for a later version (Section 12) and are not yet part of the record.

The packages scope is the lock. Each entry is a fully-qualified package identity,
the **NEVRA**: Name, Epoch, Version, Release, Architecture, for example
`bash-0:5.2.15-150600.10.1.x86_64`. This is the RPM equivalent of a resolved
lockfile. The desired spec may name a bare package or a pattern (ergonomic); the
tool resolves and records the exact build it installed (reproducible).
Reproducibility lives in the lock, not the spec. The lock is not a separate
sidecar format: it is the packages scope of the same document with version,
release, and architecture filled in.

### 5.4 The format: the Machinery system description

The desired manifest, the applied record, the output of `describe`, and any
externally supplied state dump are one data model: the declarable subset of the
SUSE Machinery system description, the scopes a converger can act on (`packages`,
`repositories`, `services`, `config_files`). Each scope is a wrapper of the form
`{ _attributes, _elements }` with `underscore_style` field names, the convention
Machinery established. The canonical serialisation of that model is JSON
(`format_version` 1).

Choosing this format is deliberate and has lineage. Machinery was SUSE's own
system-description-and-comparison tool: it could describe a running system to a
structured document, compare two such documents, and drive a system toward one.
That is exactly the describe, compare, converge loop a declarative engine needs.
Reusing the format means the actual state and the desired state are the same kind
of object, so comparison is native rather than a translation exercise, and it
keeps the design inside SUSE's own provenance rather than importing a foreign
schema. A scope absent from a manifest is unmanaged (no assertion); a scope
present but empty asserts that scope should be empty. That absent-versus-empty
distinction is the contract for whether a domain is reconciled at all.

### 5.5 Reading the actual state: the built-in `describe`

The converger reads the live system itself, into the same Machinery format, so no
separate collector program is required. A single internal reader,
`describe-actual-state`, is the only code that touches live system state: the
rpmdb for packages, the zypp configuration for repositories, `systemctl` for
services, and the changed-or-unpackaged `/etc` files for config_files. Every verb
that needs actual state goes through this one reader. Drift is then a pure
comparison of two documents (Section 4): the actual one from the reader, the
reference one from the applied record.

The same reader is exposed as a verb, `describe`, which prints the actual state as
a Machinery document. This earns three things at no extra cost: bootstrap a
manifest from a running reference host (`zypper declarative describe > desired.json`
as a starting point to edit down), capture a state dump for offline or remote
verification (describe on host A, `verify state-path=...` on host B), and audit.

This also answers a design question directly. If a separate collector such as
sitar does not exist, nothing is lost. sitar is a system collector in this very
format, so it is interchangeable with `describe` output and may feed `verify` via
`state-path`, but it is optional. Only the declarable scopes are read; the
observational scopes a full sitar dump carries (cpu, pci, dmi, processes, storage,
network, and the rest) are not needed by a converger and are ignored when present.

### 5.6 Serialisation: JSON canonical, YAML optional

The manifest is a typed data model, and a serialisation renders it to bytes.
Keeping that distinction sharp is what lets a second format be added as a switch
rather than a fork: the data model and every behaviour that operates on it stay
identical, and only the thin parse-and-serialise edge changes.

JSON is the canonical serialisation and the default. YAML is an opt-in
serialisation of the same model, added because some customers author OS state in
YAML, in particular those standardised on ZARF for deploying the application
above the OS. ZARF and this converger are complementary layers, ZARF deploys the
workload (typically into Kubernetes) and zypper-declarative converges the host
beneath it, and a YAML-centric, air-gapped ZARF workflow reasonably wants the OS
manifest in the same idiom. The air-gapped angle reinforces the supply-chain
stance rather than conflicting with it: no curl, pinned and signed sources,
offline first.

Where Machinery compatibility ends is precise and one-directional. JSON is valid
YAML 1.2, so a YAML-capable reader still ingests a Machinery or sitar JSON dump
for `verify`; what YAML loses is only the output side, `describe` written as YAML
is not Machinery and strict Machinery consumers will not read it. That is the only
thing actually given up, and it is the part YAML-requesting customers have
accepted. Two design choices contain the blast radius. The applied record stays
canonical JSON regardless of the input format, so the on-disk ledger and its
snapper-userdata hash remain Machinery-readable even on an all-YAML host. And
manifest identity is the hash of a canonical serialisation of the parsed data
model, not of the raw bytes, so the same intent in JSON or YAML yields the same
`desired_sha256` and idempotence survives a format switch.

YAML also widens the parsing attack surface in a way JSON does not, which matters
under an EAL4+/EUCC posture, so the spec constrains the parser rather than leaving
it to the implementation: a non-code-executing loader only, bounded or disabled
anchor and alias expansion (the alias-expansion denial of service), single-
document streams, and explicit typing per the schema instead of YAML implicit
typing (so `NO` does not become false and a version like `1.10` does not become a
float). These are stated as constraints, not as a named library, so they bind Go,
Rust, or C++ equally.

---

## 6. `/etc` handling

`/etc` is a nested btrfs subvolume, which has one consequence that bites if
missed: **btrfs snapshots are not recursive.** When the root subvolume is
snapshotted, the nested `/etc` appears as an empty directory in that snapshot;
`/etc` has its own snapshot lineage. This is exactly why transactional-update has
a dedicated `50-etc` snapper plugin.

Consequence for the converger: a snapper diff on the root config is blind to
`/etc`. To diff `/etc` you compare the `/etc` subvolume against a reference, which
is the built-in `etc.syncpoint` (a copy-on-write reference snapshot of `/etc`
taken at generation-creation time, tagged with a `transactional-update.comparewith`
marker). The drift diff for `/etc` is a subvolume comparison; the intent diff is
the manifest comparison and is filesystem-agnostic.

One concrete trap: because the syncpoint is a nested subvolume inside `/etc`, a
naive file walker sees a directory at `/etc/etc.syncpoint`. The converger must
hard-exclude that path from the file walk, the diff, and the deletion pass. Treat
it like RPM-owned paths: SUSE-owned infrastructure, never touched.

---

## 7. Convergence domains

Convergence is not one operation. It is a small set of independent domains, each
with its own differ and applier; the actual side of every domain is read by the
single `describe` reader (Section 5.5). The tool is a thin orchestrator over them,
run in a fixed phase order.

- **Packages.** Desired is the packages scope; actual is read from the rpmdb by
  the single `describe` reader (Section 5.5); apply is zypper install/remove
  against the declared pinned repositories. Package removal deletes files in
  `/usr` via zypper, not via the file pass. The tool records the resolved packages
  scope as the lock. The tool must never reverse-engineer "package X is installed"
  from file diffs; the rpmdb is the oracle.
- **Files (`/etc`).** Desired is the declared file set; actual is `/etc`; apply is
  write declared files and delete `(declared_old - declared_new)`, excluding
  RPM-owned paths, the keep-list, and `etc.syncpoint`.
- **Units.** Desired is which units are enabled or masked; actual is
  `systemctl is-enabled`; apply is offline enablement with `systemctl --root`
  pointed at the new snapshot mount. Offline enablement sidesteps the
  systemd-preset-only-evaluated-on-first-boot trap, which is the clean answer
  rather than the firstboot/machine-id re-trigger hack.

Phase order for a single converge: packages, then files, then units, then write
the applied record, then commit and activate. Install a package before enabling
its unit; write a config before the service that reads it; remove a unit before
removing its package.

---

## 8. The transaction abstraction (decision left open)

The transaction boundary is deliberately abstract. Two bindings exist and the
design commits to neither:

- **External binding.** The tool is invoked inside a transaction created by a
  separate mechanism, for example `transactional-update run zypper declarative apply`.
  In this mode the tool detects it is already inside a fresh snapshot and operates
  on that filesystem without opening its own.
- **Internal binding.** On SLES 16.1, where the transactional machinery is merged
  into zypper, `zypper declarative apply` opens and commits the transaction
  through the merged path.

The selection is a configuration knob (`transaction-mode = auto | external | internal`),
defaulting to auto-detect. The converger's behavior is identical inside the
snapshot regardless of who opened it: converge the domains, write the record,
verify, hand back to the mechanism to seal and schedule activation.

This is the answer to "leave the transactional-update versus zypper decision open
for now": the spec defines the interface and defers the binding, so the same
`zypper-declarative` implementation works whether transactional-update is a
separate tool or a zypper subsystem.

Activation is a reboot, or a soft-reboot (SL Micro 6.1+, userland-only restart)
when no kernel or bootloader change is involved, with rebootmgr honoring
maintenance windows.

---

## 9. The `zypper declarative` seam

zypper has a git-style subcommand dispatch built in. An executable in
`/usr/lib/zypper/commands` (or named `zypper-declarative` anywhere on `$PATH`) is
invoked as `zypper declarative`. No patching of zypper is required, and the
extension may be written in any language. This is the seam regardless of whether
the transaction is external or internal, so the user-facing surface is stable:
`zypper declarative apply`, `diff`, `verify`, `status`, `describe`.

Two constraints to design around: zypper's global options are not forwarded to the
subcommand, and subcommands cannot run inside the interactive zypper shell. The
tool parses its own arguments and has its own verb set, so neither limits it.

If the tool later becomes a true built-in zypper command rather than a dispatched
subcommand, the user-facing surface does not change; only the dispatch path does.

---

## 10. Delivery and GitOps

The tool's job ends at "given a manifest on disk, converge the system." How the
manifest arrives is a separate concern, with one weak option and three good ones.

SSH or scp works for a lab but is the weakest: no signature verification, no
version record, no automatic trigger. The three real channels, in increasing
alignment with the supply-chain stance:

1. **Signed pull (the GitOps default).** A host timer fetches a signed artifact
   over HTTPS, verifies the signature, stages the desired spec to a writable path
   (`/var/lib/zypper-declarative/desired.json`, no snapshot needed to write), and
   runs `zypper declarative apply`. Source of truth stays in git; the host is a
   pull agent. Best fit for disconnected and sovereignty-oriented devices, because
   each host verifies its own provenance.
2. **Signed package (supply-chain-native).** CI builds the manifest into a signed
   RPM in OBS. The host's pinned repo carries it; the RPM signature is verified by
   the trust chain already present; the installed manifest version is recorded in
   the rpmdb and is therefore auditable; a manifest rollback is a package rollback.
   Fold delivery and converge into one transaction: install the new manifest
   package and invoke the converger against it in a single snapshot.
3. **Central push (fleet management).** SUSE Multi-Linux Manager (the current name
   for what was SUSE Manager, Salt-based) pushes the manifest and runs the
   converger as a Salt state via the transactional executor, scheduling the
   activating reboot through rebootmgr. Best fit for regulated fleets that want a
   single audit point and maintenance windows.

A practical starting point for any channel: run `zypper declarative describe` on a
hand-configured reference host and edit the result down into the base layer,
rather than authoring the first manifest from scratch. The dump can be emitted as
JSON or, for a YAML-centric workflow, as YAML via `format=yaml` (Section 5.6).

The GitOps loop, host-level and non-Kubernetes: a git repo is the source of
truth, structured for composition (base layer, role layers, thin per-host
overrides); commits and tags are signed; CI validates against the Machinery JSON
schema, resolves the packages lock against the pinned repo, and builds the signed
artifact; the host reconciles on a timer or under Multi-Linux Manager control; the
timer cadence is what turns provisioning into reconciliation and reverts drift;
status is reported back. These compose: humans author signed JSON in git, OBS CI
builds a signed manifest, hosts consume it by pull or by push, and no path uses
raw SSH or curl.

---

## 11. Reproducibility stance

This design reaches level 3, not Nix-style intrinsic reproducibility, and that is
the honest framing to carry into any POC.

RPM delivers reproducible installs: applying a pinned NEVRA set is deterministic,
files land with identical content and build-time mtimes, and modern packaging has
moved scriptlet logic into declarative `sysusers.d`, `tmpfiles.d`, and file
triggers. The one real condition is pinning the repository state, which OBS and a
versioned channel provide. Two non-authoritative caveats (the rpmdb records
per-package install time, and regenerated caches vary) are noise to a converger
that compares the spec to actual files rather than to rpmdb bytes.

The difference from NixOS is locus, not capability: Nix pins the entire build
closure by construction; RPM reaches the same outcome assembled from OBS
reproducible builds, a pinned repo, and declarative packaging. "Intrinsic by
construction" versus "achievable with discipline," not "reproducible" versus
"broken." For the EUCC and Common Criteria objective of rebuild-and-verify, SUSE
answers through a trusted, reproducible build service with provenance rather than
a decentralized reproducible-by-spec model. Different mechanism, same assurance
objective.

---

## 12. Open decisions and deferred scope

- **Secrets in the declaration.** A WiFi PSK or TLS key must not sit in plaintext
  in a version-controlled JSON. The converger needs a secrets path (references
  resolved at apply time from systemd-creds, an encrypted-values scheme, or a
  fetch from a store). Deferred; the spec marks it out of scope and leaves a
  CONFIG seam.
- **Kernel cmdline as declared state.** Under the SLES 16.1 UKI and `sdbootutil`
  direction, cmdline lives in the UKI or via addons rather than freely edited in
  grub.cfg. "cmdline as declared state" interacts with that move and should be
  pinned down early.
- **Hard-reset path.** Incremental convergence needs no keep-list, because the
  deletion rule subtracts the old declared set. A flatcar-reset-style wipe of
  `/etc` back to declared-only does need an explicit keep-list (machine-id, SSH
  host keys, the systemd random seed), because a from-scratch reset has no
  `declared_old` to subtract. Define the keep-list even if the POC only uses
  incremental.
- **Built-in versus subcommand.** Whether `declarative` ends up a dispatched
  subcommand or a true zypper built-in is left open; the user-facing surface is
  identical either way.

---

## Changelog

- 2026-05-29: Aligned with `zypper-declarative.spec.md` v0.4.0. Added Section 5.6
  on serialisation: the manifest is a typed data model with JSON as the canonical
  serialisation and YAML as an opt-in alternative (for YAML-centric and ZARF
  workflows), the Machinery compatibility boundary is one-directional (JSON input
  still ingests, only YAML output breaks Machinery), the applied record stays
  canonical JSON, manifest identity is the canonical-model hash so it is
  format-independent, and the YAML parser is constrained to a safe profile for the
  EAL4+/EUCC posture. Header and scope updated accordingly.
- 2026-05-29: Aligned with `zypper-declarative.spec.md` v0.3.0. The manifest, the
  applied record, and the actual state are now one JSON schema, the SUSE Machinery
  system-description format (declarable subset, ScopeWrapper idiom); the package
  lock is the resolved packages scope rather than a separate NEVRA list. Added the
  built-in `describe` reader as the single live-state source and the `describe`
  verb (bootstrap, capture, audit), with an external collector such as sitar now
  optional and interchangeable. Updated the drift model to a pure comparison of
  two Machinery documents and added the format and actual-state sections (5.4,
  5.5).
- 2026-05-29: Initial architecture document. Synthesizes the declarative-OS design
  discussion into a reference for the `zypper-declarative` PCD specification.
  Records the two-diff model, the manifest location and ledger semantics, the
  `/etc` nested-subvolume handling, the convergence domains, the transaction
  abstraction with the binding decision left open, the `zypper declarative` seam,
  the three delivery paths, and the level-3 reproducibility stance.
