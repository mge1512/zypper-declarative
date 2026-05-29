% ZYPPER-DECLARATIVE(1) zypper-declarative 0.5.0 | User Commands
% Matthias G. Eckermann
% May 2026

# NAME

zypper-declarative - declarative, reconciling converger surfaced as a zypper subcommand

# SYNOPSIS

**zypper declarative** *verb* [*key=value* ...]

**zypper-declarative** *verb* [*key=value* ...]

# DESCRIPTION

**zypper-declarative** converges a SUSE system to a desired manifest inside a
single snapshot transaction, recording what was applied. The manifest is the
declarable subset of the SUSE Machinery system description: **packages**,
**repositories**, **services**, and **config_files**, using the
`{_attributes, _elements}` ScopeWrapper idiom and `underscore_style` field
names. Its canonical serialisation is JSON (Machinery `format_version` 1); YAML
is an opt-in serialisation of the identical data model.

The tool reads the live system into this same model itself (see **describe**),
so no separate collector program is required.

# VERBS

**apply**
: Converge the system to the desired manifest inside a single snapshot
transaction, recording what was applied. Idempotent. Requires privilege and an
available transaction mechanism.

**diff**
: Dry run. Compute and print what **apply** would change, making no
modification and opening no transaction.

**verify**
: Check whether the actual state equals the declaration recorded in the current
generation, modulo the keep-list. Exit 0 if and only if it matches.

**status**
: Print the current declarative state: the applied manifest hash, the
generation, the resolved package count, and a one-line drift summary.

**describe**
: Read the actual state of the declarable scopes and emit it in the selected
serialisation (JSON by default, YAML on request). Read-only.

# OPTIONS

Options are *key=value* pairs and precede any bare-word argument. Control via
environment variables is not supported.

**mode=auto|external|internal**
: Transaction binding. Default **auto**.

**manifest-path=**\<path>
: Desired manifest. Default from CONFIG.

**format=json|yaml**
: Serialisation for this invocation's manifest I/O. When omitted, the operative
file extension decides, else the **manifest-format** default.

**state-path=**\<path>
: State dump used as the actual-state source for **verify**.

**root=**\<path>
: Root to describe. Default "/".

**out=**\<path>
: **describe** output file. Default stdout.

**on-unreadable=error|warn**
: **describe**: fail (default) or omit-and-warn on an unreadable scope source.
A source that cannot be read is never emitted as an empty scope.

Additional CONFIG knobs accepted as options: **manifest-format**, **repo-lock**,
**content-store**, **keep-list**, **signature-verification**, **keyring**,
**activation-policy**, **applied-root**.

# GLOBAL

**--version**
: Print the program name, version, and embedded spec hash; exit 0.

**--help**, **-h**
: Print usage; exit 0.

A bare invocation prints usage to stdout and exits 0 (discovery; it never runs a
default verb).

# EXIT STATUS

**0**
: Success (converged, no-op, system matches declaration, or describe emitted).

**1**
: Logical failure (convergence failed and discarded; verify drift; invalid,
unsafe-YAML, or unverified manifest; state collection failed).

**2**
: Invocation error (bad arguments; unknown format value; manifest unreadable;
insufficient privilege; transaction mechanism unavailable; output path
unwritable; malformed state dump).

# FILES

*/usr/lib/zypper-declarative/applied.json*
: The applied record of a generation, stored within the snapshot it describes.

# SEE ALSO

**zypper**(8), **snapper**(8), **transactional-update**(8), **systemctl**(1)
