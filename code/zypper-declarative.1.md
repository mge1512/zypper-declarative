% ZYPPER-DECLARATIVE(1) zypper-declarative 0.4.0 | User Commands
% Matthias G. Eckermann
% May 2026

# NAME

zypper-declarative - declarative, reconciling converger surfaced as a zypper subcommand

# SYNOPSIS

**zypper declarative** *verb* [*key=value* ...]

**zypper-declarative** *verb* [*key=value* ...]

# DESCRIPTION

**zypper-declarative** converges a SUSE system to a desired manifest expressed
in the declarable subset of the SUSE Machinery system-description format:
the *packages*, *repositories*, *services*, and *config_files* scopes. The
canonical serialisation is JSON (Machinery format_version 1); YAML is an
opt-in serialisation of the identical data model.

Convergence happens inside a single snapshot transaction. What was applied is
recorded as an *applied record* under `/usr/lib/zypper-declarative/applied.json`
within the generation, so a rollback restores the matching declaration. The
tool is idempotent: a second run against an unchanged manifest and an
undrifted system makes no changes.

# VERBS

**apply**
: Converge the system to the desired manifest inside a snapshot transaction.

**diff**
: Dry run. Print what **apply** would change. Makes no modification.

**verify**
: Check whether the actual state equals the applied record, modulo the
keep-list. Reads live state, or a supplied state dump (**state-path=**).

**status**
: Print the current declarative state: applied manifest hash, generation,
resolved package count, and a one-line drift summary.

**describe**
: Read the actual state of the declarable scopes and emit it as a manifest
(canonical JSON by default, YAML on request).

# OPTIONS

Options use **key=value** syntax and precede any bare-word argument. Control
via environment variables is not supported.

**mode=auto|external|internal**
: Transaction binding. Default *auto*.

**manifest-path=**\<path\>
: Desired manifest path. Default from configuration.

**format=json|yaml**
: Manifest input format (for **apply**/**diff**) and **describe** output
format. Default *json*, otherwise inferred from the file extension.

**state-path=**\<path\>
: A state dump in the shared schema, used as the actual-state source for
**verify**.

**root=**\<path\>
: Root to describe (and the actual-state root for read-only verbs). Default `/`.

**out=**\<path\>
: **describe** output file. Default standard output.

**applied-root=**\<path\>
: Generation root from which the applied record is read. Default `/`.

**keep-list=**\<path\>
: Allowlist of persistent-but-undeclared paths never reported or deleted.

**content-store=**\<path\>
: Base path for resolving config_files *content_ref* values at apply time.

**repo-lock=**\<repo\>
: Fallback pinned repository used only when the manifest declares no
*repositories* scope.

**signature-verification=on|off**, **keyring=**\<path\>
: Enable or disable manifest signature verification. Default *on*.

**activation-policy=reboot|soft-reboot|none**
: How **apply** schedules activation of a freshly sealed snapshot.

**--version**
: Print the version and embedded spec hash, then exit 0.

**--help**
: Print usage, then exit 0.

# EXIT STATUS

**0**
: Success: convergence complete or a no-op, system matches the declaration, or
**describe** produced output.

**1**
: Logical failure: convergence failed and was discarded; **verify** found
drift; manifest invalid, unsafe-YAML, or unverified; state collection failed.

**2**
: Invocation error: bad arguments; unknown format value; manifest unreadable;
insufficient privilege; transaction mechanism unavailable; output path
unwritable; malformed state dump.

# FILES

`/usr/lib/zypper-declarative/applied.json`
: The applied record of a generation.

`/etc/etc.syncpoint`
: Never written to or deleted.

# INSTALLATION

Install from the openSUSE Build Service:

    zypper install zypper-declarative

On Debian/Ubuntu derivatives:

    apt install zypper-declarative

On Fedora-family systems:

    dnf install zypper-declarative

# SEE ALSO

**zypper**(8), **snapper**(8), **transactional-update**(8), **systemctl**(1)
