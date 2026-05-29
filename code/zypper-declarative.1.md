<!-- generated from spec: zypper-declarative.spec.md sha256:b2d0de88fbed1163678e59e931c741b9d999b71f902f6eb01db8790bb813d057 -->
% ZYPPER-DECLARATIVE(1) zypper-declarative 0.6.0 | User Commands
% Matthias G. Eckermann
% May 2026

# NAME

zypper-declarative - declarative reconciling converger surfaced as a zypper subcommand

# SYNOPSIS

**zypper declarative** *verb* [*key=value* ...]

**zypper-declarative** *verb* [*key=value* ...]

# DESCRIPTION

**zypper-declarative** converges a SUSE system to a desired manifest, the
declarable subset of the SUSE Machinery system description (*packages*,
*repositories*, *services*, *config_files*), inside a single snapshot
transaction, recording what was applied. It is idempotent: a second run against
an unchanged manifest and an undrifted system makes no changes.

The manifest is a typed data model whose canonical serialisation is JSON
(Machinery format_version 1). YAML is an opt-in serialisation of the same model,
selected by **format=** or by file extension.

# VERBS

**apply**
: Converge the system to the desired manifest inside a snapshot transaction.

**diff**
: Dry run. Print what **apply** would change; make no modification and open no transaction.

**verify**
: Check whether the actual state equals the applied declaration, modulo the keep-list.

**status**
: Print the applied manifest identity, generation, and a one-line drift summary.

**describe**
: Read the actual state of the declarable scopes and emit it as a manifest.

**version**
: Print the program name, version, and the embedded specification hash.

**help**
: Print usage.

# OPTIONS

Options are *key=value* pairs and must precede any bare-word argument. POSIX
**--flag** style is not used for options; **--version**, **--help**, and **-h**
are tolerated aliases for the **version** and **help** commands only.

**mode=auto|external|internal**
: Transaction binding. Default *auto*.

**manifest-path=**\<path\>
: Desired manifest. Default from CONFIG.

**manifest-format=json|yaml**
: Default serialisation. Default *json*.

**format=json|yaml**
: Serialisation for this invocation's manifest I/O.

**state-path=**\<path\>
: State dump as the actual-state source for **verify**.

**root=**\<path\>
: Root to **describe**. Default */*.

**out=**\<path\>
: **describe** output file. Default stdout.

**on-unreadable=error|warn**
: **describe**: fail (default) or omit+warn on an unreadable scope source.

**scope=etc|full**
: **describe**/**verify** read scope. *etc* (default) inspects only /etc; *full* also audits /usr and /boot (expensive, opt-in).

**repo-lock=**\<channel\>
: Fallback pinned repository when the manifest declares none.

**content-store=**\<path\>
: Base path against which config-file content_ref values are resolved.

**keep-list=**\<path\>
: Allowlist of persistent-but-undeclared paths never reported or deleted.

**signature-verification=on|off**
: Manifest signature checking. Default *on*.

**keyring=**\<path\>
: Keyring used for signature verification.

**activation-policy=reboot|soft-reboot|none**
: How **apply** schedules activation of a freshly sealed snapshot.

**applied-root=**\<path\>
: Generation root from which the applied record is read. Default */*.

# EXIT STATUS

**0**
: Success: converged, no-op, system matches declaration, or describe emitted.

**1**
: Logical failure: convergence failed and discarded; verify drift; invalid, unsafe-YAML, or unverified manifest; state collection failed.

**2**
: Invocation error: bad arguments; unknown format value; manifest unreadable; insufficient privilege; transaction mechanism unavailable; output path unwritable; malformed state dump.

# FILES

*/usr/lib/zypper-declarative/applied.json*
: The applied record of the current generation, stored within the snapshot.

# EXAMPLES

Converge to the default manifest:

    zypper declarative apply

Bootstrap a manifest from the running system:

    zypper declarative describe > desired.json

Audit /usr and /boot integrity in addition to the declaration:

    zypper declarative verify scope=full

# SEE ALSO

**zypper**(8), **transactional-update**(8), **snapper**(8)
