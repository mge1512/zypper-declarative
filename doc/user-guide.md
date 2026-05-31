# zypper-declarative User Guide

This guide is for operators and architects who want to put a SUSE SL Micro 6.2
host under declarative management with `zypper-declarative`: how to get the tool
and a first manifest onto a freshly provisioned machine, how to author a baseline
from a running reference system, and how the pieces fit together. It is a
companion to, not a substitute for, the specification (`zypper-declarative.spec.md`,
the authoritative description of every command) and the architecture document
(`zypper-declarative-architecture.md`, the design rationale).

Status: this is pre-1.0 software, generated from the specification. The read-only
commands (`describe`, `diff`, `verify`, `status`) are the mature path; the
converging command (`apply`) is milestone-gated. Do not point it at a production
host yet.

## 1. The goal

A declarative operating system, in the sense this project uses, needs two things:
the desired state described as data, and an engine that converges the system to
that data and removes what the data no longer asserts. SL Micro already provides
the hard parts (a read-only `/usr`, transactional updates, btrfs snapshots as
switchable generations). What was missing is the converger. `zypper-declarative`
is that converger, surfaced as `zypper declarative`.

You write the host's intended state as a signed manifest (packages, repositories,
services, and `/etc` files). The tool reads the actual state, computes the
difference, applies it inside a snapshot transaction, and records what it applied
so the next run can reason about deletions as well as additions. Re-running it is
safe: an unchanged manifest against an undrifted system does nothing.

This guide threads two tasks:

- Initiating a host: getting the tool and the first manifest onto a fresh SL Micro
  install, then handing off to ongoing reconciliation (Section 2).
- Designing a baseline: deriving a manifest from a reference host and validating
  that it reapplies cleanly (Section 3).

The mechanics of how provisioning and reconciliation actually work are in the
appendix (Section 5); read it if you want the why.

## 2. Initiating an SL Micro host

SL Micro is provisioned on first boot by Combustion and Ignition. The short
version: Ignition is declarative JSON for users, files, and disks; Combustion runs
a script once on first boot and can do arbitrary work, including installing
packages through `transactional-update`. Because the bootstrap has to install the
`zypper-declarative` RPM, Combustion is the right tool (Ignition alone cannot
install a package). The appendix explains the division in more detail.

The pattern is the same for virtual and physical machines; only how the
configuration is delivered to the machine differs.

### 2.1 The Combustion script

Combustion looks for a script with a `# combustion: network` directive (when it
needs the network during provisioning) and runs it in two phases. The script
below, in its first-boot phase, configures a pinned repository, installs the tool
and a first manifest, and enables a systemd unit that runs the converge on the
next boot. Adjust the repository URL, the GPG key, and the manifest source to your
environment.

```bash
#!/bin/bash
# combustion: network
set -euo pipefail

# --- phase: configuration (runs in the initrd, before the real system boots) ---

# 1. Trust and pin the signed repository that carries zypper-declarative and,
#    optionally, the baseline manifest package. Use your own OBS project.
rpm --import https://repo.example.com/keys/zypper-declarative.asc
zypper --gpg-auto-import-keys addrepo --refresh \
  https://repo.example.com/SLMicro:6.2:pinned/standard zd-pinned

# 2. Install the tool into the new system using the transactional path.
#    Do not use curl to fetch a binary; install the signed package.
transactional-update --non-interactive pkg install zypper-declarative

# 3. Place the first desired manifest where the converge unit will read it.
#    Option A: ship it as a signed RPM and install it (preferred, auditable):
#      transactional-update --non-interactive pkg install zypper-declarative-baseline
#    Option B: write it directly from a here-document or fetch it from a verified
#    source. Below is the direct form for a minimal baseline.
mkdir -p /etc/zypper-declarative
cat > /etc/zypper-declarative/desired.json <<'MANIFEST'
{
  "meta": { "format_version": 1, "generator": "baseline", "desired_sha256": "" },
  "repositories": {
    "_attributes": { "repository_system": "zypp" },
    "_elements": [
      { "alias": "zd-pinned", "name": "SL Micro 6.2 (pinned)",
        "url": "https://repo.example.com/SLMicro:6.2:pinned/standard",
        "type": "rpm-md", "enabled": true, "gpgcheck": true,
        "autorefresh": false, "priority": 99 }
    ]
  },
  "packages": {
    "_attributes": { "package_system": "rpm" },
    "_elements": [
      { "name": "vim-small", "version": "", "release": "", "arch": "" }
    ]
  },
  "services": {
    "_attributes": { "init_system": "systemd" },
    "_elements": [
      { "name": "sshd.service", "state": "enabled" }
    ]
  }
}
MANIFEST

# 4. Install a one-shot unit that converges on the first real boot, after the
#    network is up, then does not run again unless re-enabled or timed.
cat > /etc/systemd/system/zypper-declarative-firstboot.service <<'UNIT'
[Unit]
Description=Initial declarative converge
After=network-online.target
Wants=network-online.target
ConditionPathExists=/etc/zypper-declarative/desired.json

[Service]
Type=oneshot
ExecStart=/usr/bin/zypper-declarative apply manifest-path=/etc/zypper-declarative/desired.json
# A successful apply schedules its own activation per activation-policy.

[Install]
WantedBy=multi-user.target
UNIT

systemctl enable zypper-declarative-firstboot.service

# --- end of configuration phase ---
```

Notes:

- The converge runs on the first real boot, not inside the initrd, because `apply`
  opens a snapshot transaction and wants the normal system context.
- Signature verification is on by default. In production, install the manifest as
  a signed RPM (Option A) or fetch it from a verified source; the inline
  here-document is convenient for a lab.
- For ongoing reconciliation rather than a one-shot, replace the service with a
  `.timer` that runs `apply` periodically; that is what turns first-boot
  provisioning into drift-correcting reconciliation. See the architecture
  document's delivery section.

### 2.2 Delivering the configuration to a virtual machine

For a VM, the ignition/combustion configuration is presented to the guest by the
hypervisor or by an attached volume:

- Build a small FAT or ISO volume whose layout contains the Combustion script at
  `combustion/script` (and any Ignition config at `ignition/config.ign`), labelled
  so SL Micro's first-boot tooling finds it.
- Attach that volume to the VM, or, on platforms that support it, pass the config
  through the firmware channel (for example qemu `fw_cfg`) or the platform's
  instance-userdata mechanism.
- Boot the SL Micro image. Combustion runs the script once; subsequent boots skip
  it.

### 2.3 Delivering the configuration to a physical machine

Physical deployments are increasingly the priority, and the same script applies.
The delivery options are:

- A labelled USB stick or second partition carrying the `combustion/` (and
  optional `ignition/`) directory, inserted at install time. This is the simplest
  bare-metal route and needs no infrastructure.
- Network provisioning: serve the configuration over HTTP from your provisioning
  server and point the installer at it with the appropriate boot parameter, so a
  PXE or installer-driven rollout fetches the same Combustion script.
- For fleets, combine the network route with the signed-RPM manifest (Option A
  above), so each machine verifies its own provenance from your pinned repository,
  which suits the supply-chain posture (no unsigned downloads, no curl).

In all three cases the machine ends the install with the tool present, a first
manifest in place, and the converge unit enabled, ready to reconcile.

### 2.4 Confirm the first converge

After the first real boot:

```bash
zypper declarative status        # shows the applied manifest and a drift summary
zypper declarative verify        # exits 0 if the system matches what was applied
```

A clean `verify` means the host equals its declaration. From here the host is
under declarative management.

## 3. Designing a baseline

The most common architect task is to design a baseline for a class of machines:
install SL Micro, shape it by hand, then capture that shape as a manifest you can
apply elsewhere. The key idea, which the tool enforces, is that a describe dump is
a census of everything that is true, while a baseline is a statement of intent.
Turning one into the other is an editing step, and that editing is the act of
authoring the baseline.

### 3.1 Capture, edit, validate

1. Install SL Micro and configure it by hand to taste.

2. Capture the current state:

   ```bash
   zypper declarative describe out=baseline.json
   ```

   Use the default scope (just `/etc` plus packages, repositories, services) for a
   baseline. The `scope=full` mode is for auditing `/usr` and `/boot` integrity
   (Section 4), not for authoring intent.

3. Edit `baseline.json` into a statement of intent. This is the important step:

   - Remove runtime and machine-generated files that you do not actually want to
     pin (lock files, caches, seeds, anything under `/etc` that is state rather
     than configuration).
   - Decide your package policy. A fresh describe pins every package to its exact
     version-release-arch. If you want "these packages from this pinned repo" and
     are content to let the repository define versions, clear the version fields;
     if you want a frozen build set, keep them.
   - Keep the `repositories` scope honest: it is what the package step resolves
     against.

4. Validate that the edited baseline reapplies cleanly. You can do this without a
   live target using the offline comparison: capture a second host's state (or the
   same host after a change) and compare it against your baseline as two files.

   ```bash
   # what would apply do, comparing the baseline against a captured state,
   # without reading the live system or needing a prior apply
   zypper declarative diff manifest-path=baseline.json state-path=after.json

   # does a captured state satisfy the baseline? (offline, no applied record)
   zypper declarative verify manifest-path=baseline.json state-path=after.json
   ```

5. The real reapply test, on a fresh SL Micro install, is `apply` followed by
   `verify`:

   ```bash
   zypper declarative apply manifest-path=baseline.json
   zypper declarative verify        # should be clean
   zypper declarative apply manifest-path=baseline.json   # should be a no-op
   ```

   A clean `verify` and a no-op second `apply` together demonstrate the baseline is
   convergent and idempotent.

### 3.2 Why a raw describe dump is not a baseline

The tool deliberately refuses to apply a manifest that still carries the
observational `changed_managed_files` or `unmanaged_files` scopes (the output of
`scope=full`). Those describe findings about `/usr` and `/boot` are not things
`apply` can act on, so a raw full dump used as a manifest would be misleading;
`apply` rejects it and tells you to edit it into intent first. A plain
`describe` (default scope) does not carry those scopes and is the right starting
point.

## 4. Checking system integrity

Declaration drift ("does the system match my manifest") and system integrity
("has anything in the package-managed tree been tampered with") are different
questions. By default the tool answers only the first, and reads only `/etc`,
which keeps it fast.

When `/usr` is not guaranteed immutable, you can ask the integrity question
explicitly with `scope=full`, which additionally scans the package-managed trees
outside `/etc` (`/usr`, the merged-usr roots, and `/boot`):

```bash
# capture an integrity-inclusive description
zypper declarative describe scope=full out=full-state.json

# audit: changed packaged files and out-of-band additions are reported as drift
zypper declarative verify scope=full
```

This is expensive and opt-in; it never runs by default, including on a mutable
`/usr`. `verify scope=full` exits non-zero if it finds a changed packaged file or
an unpackaged addition outside `/etc` (boot-chain artifacts such as the generated
initramfs are genuinely unpackaged and will show up unless you keep-list them).

On a host where `/usr` is read-only and verity-protected, this scan is largely
redundant: integrity is guaranteed by construction rather than detected. The scan
is the fallback for hosts that are neither transactional nor verity-protected.

## 5. Appendix: how it works

This appendix sketches the mechanics behind the workflows above. The architecture
document is the fuller treatment.

### 5.1 Why Combustion, and how it differs from Ignition

Both run only on first boot, off the same ignition/combustion configuration source:

- Ignition applies a declarative JSON document: it can create users, write files,
  and partition disks. It cannot install an RPM.
- Combustion runs an arbitrary script in two phases (a configuration phase in the
  initrd, then the system boot). Because it is a script, it can call
  `transactional-update pkg install`, which is exactly what bootstrapping
  `zypper-declarative` needs.

So the bootstrap uses Combustion to install the tool and the first manifest, and
may use Ignition alongside it for the purely declarative file and user bits. After
first boot, neither runs again; ongoing change is the job of `zypper declarative`.

### 5.2 The two-diff model

Convergence needs two comparisons, and the filesystem gives only one of them
cheaply:

- The intent diff compares the new manifest against the previously applied
  manifest (the applied record). This is the only thing that yields deletions:
  when you drop a line from the manifest, nothing on disk changes, so only
  comparing manifests reveals that something should be removed.
- The drift diff compares the actual system against the declaration, catching a
  change made out of band.

The deletion rule that falls out is safe by construction: the only files removed
are those the tool previously declared and no longer declares. Machine-identity
and package-owned files are never candidates.

### 5.3 The manifest format, and where state is read

Manifests use the SUSE Machinery system-description format (the declarable subset:
packages, repositories, services, config_files), with JSON as the canonical
serialisation and YAML as an opt-in alternative for YAML-centric (for example
ZARF) workflows. The tool reads the live system into the same format itself, so a
captured `describe` dump and a desired manifest are the same kind of object, which
is what makes the offline comparisons in Section 3 possible. Reading is bounded to
`/etc` by default (and the rpmdb, the on-disk repository files, and unit state);
the `/usr` and `/boot` scan happens only under `scope=full`.

### 5.4 The applied record and rollback

A successful `apply` writes an applied record under `/usr/lib/zypper-declarative/`
inside the new snapshot. Because it lives under `/usr` within the generation, a
rollback to a previous snapshot restores that generation's record automatically,
so the tool always knows what was last applied to the running system. This record,
not the desired manifest, is what `verify` and the next intent diff compare
against by default.

### 5.5 The transaction boundary

`apply` does its work inside a btrfs snapshot and only activates it on reboot (or a
soft-reboot when no kernel change is involved). The boundary to the transaction is
abstract: on SL Micro a separate mechanism (`transactional-update`) opens the
snapshot, while on SLES 16.1 the transactional machinery merged into zypper can
open it. The convergence behaviour is identical either way, which is why the same
tool and the same manifests work across both.

---

## Changelog

- 2026-05-29: Initial user guide. Covers the goal, initiating an SL Micro host via
  a Combustion script (virtual and physical delivery), designing and validating a
  baseline (including the offline two-file comparison and why a raw describe dump
  is not a baseline), checking integrity with scope=full, and an appendix on the
  mechanics (Combustion vs Ignition, the two-diff model, the manifest format, the
  applied record, and the transaction boundary). Aligns with
  `zypper-declarative.spec.md` v0.6.1.
