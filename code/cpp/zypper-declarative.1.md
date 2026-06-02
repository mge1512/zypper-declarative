% ZYPPER-DECLARATIVE(1) zypper-declarative 0.6.8
% Matthias G. Eckermann <pcd@mailbox.org>
% June 2026

# NAME

zypper-declarative - declarative convergence of SUSE system state inside a snapshot transaction

# SYNOPSIS

**zypper declarative** *verb* [*key*=*value* ...]

**zypper-declarative** *verb* [*key*=*value* ...]

# DESCRIPTION

**zypper-declarative** converges a SUSE system (SL Micro 6.2, SLES 16.1) to a
declarative manifest describing the declarable subset of the SUSE Machinery
system description: **packages**, **repositories**, **services**, and
**config_files** (files under */etc*). The manifest's canonical serialisation
is JSON (Machinery *format_version* 1); YAML is an opt-in serialisation of the
same data model.

The tool reads the live system into this same model itself (see **describe**),
so no separate collector program is required. Convergence happens inside a
single snapshot transaction; on any failure the transaction is discarded and
the running system is left unchanged.

It is surfaced as a **zypper** subcommand (an executable in
*/usr/lib/zypper/commands*) and is also invokable directly.

Options are **key=value** pairs and may precede or follow the verb. POSIX
**--flag** style is not used for options; the only tolerated flag aliases are
**--version**, **--help**, and **-h**. Behaviour is never controlled via
environment variables.

# VERBS

**apply**
: Converge the system to the desired manifest inside one snapshot transaction,
  recording what was applied. Idempotent: a second run against an unchanged
  manifest and an undrifted system makes no changes.

**diff**
: Dry run. Print what **apply** would change without modifying the system or
  opening a transaction. Drift is computed against the desired manifest.

**verify**
: Check whether the actual state equals a reference declaration (the applied
  record by default, or a supplied manifest), modulo the keep-list.

**status**
: Print the current declarative state: applied manifest hash, generation, and a
  one-line drift summary. Read-only and fast.

**describe**
: Read the actual state of the declarable scopes and emit it in the selected
  serialisation (JSON by default, YAML on request).

**init**
: Onboard a machine: adopt its current state as the managed baseline in one
  command. Reads the live system, opens a snapshot, writes the described state
  as the applied record, and converges nothing.

**version**
: Print the program name, version, and embedded spec hash. Exit 0.

**help**
: Print usage. Exit 0.

# OPTIONS

**mode=auto|external|internal**
: Transaction binding. Default *auto*.

**manifest-path=**\<path>
: Desired manifest (apply, diff); reference manifest for verify.

**format=json|yaml**
: Serialisation for this invocation's manifest I/O. When omitted, the operative
  file extension decides, else the **manifest-format** default.

**state-path=**\<path>
: Captured actual state for verify and diff (offline; default reads live).

**root=**\<path>
: Root to describe. Default */*.

**out=**\<path>
: describe output file. Default stdout.

**on-unreadable=error|warn**
: How a live-state read treats a source it cannot read. *error* (default) fails
  the run naming the source; *warn* omits the affected items with a diagnostic
  and continues. Accepted by **describe**, **diff**, **verify**, and **apply**.

**scope=etc|full**
: Read scope for **describe** and **verify**. *etc* (default) inspects only
  */etc*; *full* additionally audits */usr* and */boot* (expensive, opt-in).

Other CONFIG knobs accepted as key=value options: **manifest-format**,
**repo-lock**, **content-store**, **keep-list**, **signature-verification**,
**keyring**, **activation-policy**, **applied-root**. A command-line option
overrides the corresponding preset value.

# EXIT STATUS

**0**
: Success (converged, no-op, system matches declaration, or describe emitted).

**1**
: Logical failure (convergence failed and discarded; verify found drift;
  manifest invalid, unsafe-YAML, or unverified; state collection failed).

**2**
: Invocation error (bad arguments; unknown format value; manifest unreadable;
  insufficient privilege; transaction mechanism unavailable; output path
  unwritable; malformed state dump).

# FILES

*/usr/lib/zypper-declarative/applied.json*
: The applied record of the current generation, stored inside the snapshot.

*/etc/etc.syncpoint*
: Never written to or deleted by the file converger.

# INSTALLATION

Install from the openSUSE Build Service:

    zypper install zypper-declarative

# SEE ALSO

**zypper**(8), **snapper**(8), **transactional-update**(8)

# COPYRIGHT

Licensed under GPL-2.0-or-later. Spec SHA256:
1641bb4413b82fecb081125067107bd5a4e30a8393edc778ead646207d68da5e
