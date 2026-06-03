% ZYPPER-DECLARATIVE(1) zypper-declarative 0.6.9 | User Commands
%
% June 2026

<!-- generated from spec: zypper-declarative.spec.md sha256:aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3 -->

# NAME

zypper-declarative - declarative convergence of SUSE system state in a snapshot transaction

# SYNOPSIS

**zypper declarative** *VERB* \[*key=value* ...]

**zypper-declarative** *VERB* \[*key=value* ...]

# DESCRIPTION

**zypper-declarative** converges the declarable subset of a SUSE Machinery
system description (packages, repositories, services, config_files) to a
desired manifest inside a single snapshot transaction, recording what was
applied. It is surfaced as a **zypper** subcommand (an executable in
*/usr/lib/zypper/commands*) and is also invokable directly.

The manifest is a typed data model whose canonical serialisation is JSON
(Machinery `format_version` 1). YAML is an opt-in serialisation of the same
model, selected by `format=` or by file extension. The applied record is
always canonical JSON.

# VERBS

**apply**
: Converge the system to the desired manifest inside a snapshot transaction.
  Idempotent: a second run against an unchanged manifest and an undrifted
  system makes no changes and prints `nothing to do`.

**diff**
: Dry run. Compute and print what **apply** would change. Opens no transaction
  and modifies nothing. Drift is computed against the desired manifest.

**verify**
: Check whether the actual state equals a reference declaration, modulo the
  keep-list. Exits 0 on a match, 1 on drift.

**status**
: Print the current declarative state: applied manifest hash, generation,
  resolved package count, and a one-line drift summary.

**describe**
: Read the actual state of the declarable scopes and emit it as a manifest
  (JSON by default, YAML on request). Read-only.

**init**
: Adopt the current state as the managed baseline: read the system, open a
  snapshot, write the applied record, and converge nothing. Forces
  `on-unreadable=warn`.

**version**
: Print the program name, version, and embedded spec hash. Exit 0.

**help**
: Print usage. Exit 0.

# OPTIONS

Options are *key=value* pairs and may appear in any position. POSIX `--flag`
style is not used (except the tolerated aliases `--version`, `--help`, `-h`).

**mode=**auto|external|internal
: Transaction binding. Default auto.

**manifest-path=**\<path>
: Desired manifest (apply, diff); reference manifest for verify.

**state-path=**\<path>
: Captured actual state for verify and diff (offline; default reads the live
  system).

**format=**json|yaml
: Serialisation for this invocation's manifest I/O. When omitted, the file
  extension decides, else the `manifest-format` default.

**root=**\<path>
: Root to describe. Default `/`.

**out=**\<path>
: Describe/init output file. Default stdout.

**on-unreadable=**error|warn
: How the actual-state reader treats an unreadable scope source. Default error.

**scope=**etc|full
: Describe/verify read scope. `etc` (default) inspects only `/etc`; `full` also
  audits `/usr` and `/boot` (expensive, opt-in). Accepted only on describe and
  verify.

**content-store=**\<path>, **keep-list=**\<path>, **repo-lock=**\<value>, **applied-root=**\<path>, **manifest-format=**json|yaml, **signature-verification=**on|off, **keyring=**\<path>, **activation-policy=**reboot|soft-reboot|none
: Additional CONFIG knobs, also accepted as key=value options.

# EXIT STATUS

**0**
: Success: convergence complete or no-op, system matches declaration, or
  describe produced output.

**1**
: Logical failure: convergence failed and discarded; verify found drift;
  manifest invalid, unsafe-YAML, or unverified; state collection failed.

**2**
: Invocation error: bad arguments; unknown format value; manifest unreadable;
  insufficient privilege; transaction mechanism unavailable; output path
  unwritable; malformed state dump.

# FILES

*/usr/lib/zypper-declarative/applied.json*
: The applied record of the current generation.

*/etc/zypp/repos.d/*.repo*
: The on-disk zypp repository configuration read by **describe**.

# SEE ALSO

**zypper**(8), **snapper**(8), **transactional-update**(8)

# AUTHOR

Matthias G. Eckermann \<pcd@mailbox.org>
