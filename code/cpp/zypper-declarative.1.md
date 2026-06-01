% ZYPPER-DECLARATIVE(1) zypper-declarative 0.6.2
% Matthias G. Eckermann
% June 2026

# NAME

zypper-declarative - declarative reconciling converger surfaced as a zypper subcommand

# SYNOPSIS

**zypper declarative** *verb* [*key=value* ...]

**zypper-declarative** *verb* [*key=value* ...]

# DESCRIPTION

**zypper-declarative** converges a SUSE system to a desired manifest inside a
single snapshot transaction, recording what was applied. The manifest is the
declarable subset of the SUSE Machinery system description: **packages**,
**repositories**, **services**, and **config_files**. Its canonical
serialisation is JSON (Machinery `format_version` 1); YAML is an opt-in
alternative serialisation of the identical data model.

Options use **key=value** syntax (POSIX `--flag` style is not used for options).
Bare-word global commands **version** and **help** are accepted, with
`--version`, `--help`, and `-h` as tolerated aliases.

# VERBS

**apply**
: Converge the system to the desired manifest inside a snapshot transaction.
Idempotent.

**diff**
: Dry run. Print what **apply** would change, making no modification and opening
no transaction.

**verify**
: Check whether the actual state equals a reference declaration, modulo the
keep-list.

**status**
: Print the current declarative state: which manifest is applied, the
generation, and a one-line drift summary.

**describe**
: Read the actual state of the declarable scopes and emit it in the selected
serialisation (JSON by default, YAML on request).

# OPTIONS

**mode=auto|external|internal**
: Transaction binding. Default **auto**.

**manifest-path=**\<path\>
: Desired manifest (apply, diff); reference manifest for verify.

**format=json|yaml**
: Serialisation for this invocation's manifest I/O. When omitted, the operative
file extension decides, else **manifest-format**.

**state-path=**\<path\>
: Captured actual state for verify and diff (offline; default reads the live
system).

**root=**\<path\>
: Root to describe. Default **/**.

**out=**\<path\>
: describe output file. Default stdout.

**on-unreadable=error|warn**
: describe handling of an unreadable scope source. Default **error**.

**scope=etc|full**
: describe/verify read scope. **etc** (default) inspects only /etc; **full** also
audits /usr and /boot (expensive, opt-in).

Additional CONFIG knobs accepted as key=value options: **manifest-format**,
**repo-lock**, **content-store**, **keep-list**, **signature-verification**,
**keyring**, **activation-policy**, **applied-root**.

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

**/usr/lib/zypper-declarative/applied.json**
: The applied record of a generation (stored inside the snapshot).

# SEE ALSO

zypper(8), snapper(8), transactional-update(8)
