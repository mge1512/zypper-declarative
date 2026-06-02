% ZYPPER-DECLARATIVE(1) zypper-declarative 0.6.6 | User Commands
% Matthias G. Eckermann
% June 2026

# NAME

zypper-declarative - declaratively converge SUSE system state in a snapshot transaction

# SYNOPSIS

**zypper declarative** *verb* [*key=value* ...]

**zypper-declarative** *verb* [*key=value* ...]

# DESCRIPTION

**zypper-declarative** converges a SUSE system to a desired manifest inside a
single snapshot transaction and records what was applied. The managed scopes are
the declarable subset of the SUSE Machinery system description: **packages**,
**repositories**, **services** (unit enablement), and **config_files** (the
changed-from-package and unpackaged files under */etc*).

The manifest is a typed data model. Its canonical serialisation is JSON
(Machinery `format_version` 1); YAML is an opt-in serialisation of the identical
model, selected by `format=` or by file extension. The applied record is always
canonical JSON.

Options are *key=value* pairs and may appear in any position relative to the
bare-word verb. POSIX `--flag` style is not used for options (only the tolerated
`--version`, `--help`, and `-h` aliases for the global commands). Behaviour is
never controlled by environment variables.

# VERBS

**apply**
: Converge the system to the desired manifest inside a snapshot transaction,
  recording what was applied. Idempotent: a second run against an unchanged
  manifest and an undrifted system makes no changes.

**diff**
: Dry run. Print what **apply** would change, making no modification and opening
  no transaction.

**verify**
: Check whether the actual state equals a reference declaration (the applied
  record by default, or a supplied manifest), modulo the keep-list.

**status**
: Print the current declarative state: applied manifest hash, generation, the
  resolved package count, and a one-line drift summary. Read-only.

**describe**
: Read the actual state of the declarable scopes and emit it as a manifest, in
  the resolved serialisation. Read-only.

# GLOBAL COMMANDS

**version** (alias **--version**)
: Print the program name, version, and embedded specification SHA256, then exit 0.

**help** (aliases **--help**, **-h**)
: Print usage to stdout, then exit 0.

A bare invocation with no verb prints usage to stdout and exits 0; it never
converges.

# OPTIONS

**mode=**auto|external|internal
: Transaction binding (default auto).

**manifest-path=**\<path\>
: Desired manifest (apply, diff) or reference manifest (verify).

**state-path=**\<path\>
: Captured actual state for verify and diff (offline; default reads live).

**format=**json|yaml
: Serialisation for this invocation's manifest I/O.

**root=**\<path\>
: Root to describe (default /).

**out=**\<path\>
: Describe output file (default stdout).

**on-unreadable=**error|warn
: How describe treats an unreadable scope source (default error).

**scope=**etc|full
: Read scope for describe and verify; etc (default) inspects only /etc, full also
  audits /usr and /boot (expensive, opt-in).

**content-store=**\<path\>
: Base path for content-addressed file content.

**keep-list=**\<path\>
: Allowlist of persistent-but-undeclared paths never reported or deleted.

**applied-root=**\<path\>
: Generation root from which the applied record is read (default /).

**manifest-format=**json|yaml
: Fallback serialisation default.

# EXIT STATUS

**0**
: Success: convergence complete or no-op, system matches declaration, or describe
  produced output.

**1**
: Logical failure: convergence failed and discarded; verify found drift; manifest
  invalid, unsafe-YAML, or unverified; state collection failed.

**2**
: Invocation error: bad arguments; unknown format value; manifest unreadable;
  insufficient privilege; transaction mechanism unavailable; output path
  unwritable; malformed state dump.

# FILES

*/usr/lib/zypper-declarative/applied.json*
: The applied record of a generation (canonical JSON), stored inside the snapshot
  it describes and restored on rollback.

*/etc/zypp/repos.d/\*.repo*
: The on-disk zypp repository configuration read for the repositories scope.

# SIGNALS

**zypper-declarative** exits cleanly on SIGTERM and SIGINT with no partial
output. An interrupted **apply** discards the transaction; no partially converged
snapshot is left as the default boot target.

# SEE ALSO

**zypper**(8), **transactional-update**(8), **snapper**(8)

# INSTALLATION

Distributed via the openSUSE Build Service (OBS). Install with **zypper**,
**apt**, or **dnf** from the appropriate repository. curl-based installation is
not supported.
