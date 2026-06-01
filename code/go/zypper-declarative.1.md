% ZYPPER-DECLARATIVE(1) zypper-declarative 0.6.5
% Matthias G. Eckermann
% June 2026

<!-- generated from spec: zypper-declarative.spec.md sha256:27aee8e374eb3507189bad0b78339109d0116e6a13b55ae4df2ba9a18e769fc4 -->

# NAME

zypper-declarative - declarative system convergence for zypper-managed systems

# SYNOPSIS

**zypper declarative** *verb* [*key*=*value* ...]

**zypper-declarative** *verb* [*key*=*value* ...]

# DESCRIPTION

**zypper-declarative** converges a SUSE system to a desired manifest inside a
single snapshot transaction, recording what was applied. The manifest is the
declarable subset of the SUSE Machinery system description: **packages**,
**repositories**, **services**, and **config_files** (files under */etc*). Its
canonical serialisation is JSON (Machinery *format_version* 1); YAML is an
opt-in alternative of the identical data model.

The tool reads the live system into this same data model itself (see
**describe**), so no separate collector program is required.

Options are *key*=*value* pairs and must precede any bare-word argument. POSIX
**--flag** style is not used for options; **--version**, **--help**, and **-h**
are tolerated aliases for the global **version** and **help** commands only.

# VERBS

**apply**
:   Converge the system to the desired manifest inside a transaction, recording
    the applied record. Idempotent: a second run against an unchanged manifest
    and an undrifted system makes no changes and prints "nothing to do".

**diff**
:   Dry run. Print what **apply** would change (the intent diff and the drift
    report) without modifying the system and without opening a transaction.

**verify**
:   Check whether the actual state equals a reference declaration, modulo the
    keep-list. By default the reference is the applied record and the actual
    state is read live; both may instead be supplied as files for an offline
    comparison.

**status**
:   Print the current declarative state: the applied manifest hash, the
    generation, the resolved package count, and a one-line drift summary.

**describe**
:   Read the actual state of the declarable scopes and emit it as a manifest
    document (JSON by default, YAML on request).

**version**
:   Print the program name, version, and embedded spec hash. Exit 0.

**help**
:   Print usage. Exit 0.

# OPTIONS

**mode**=*auto*|*external*|*internal*
:   Transaction binding. Default *auto*.

**manifest-path**=*path*
:   The desired manifest (apply, diff); the reference manifest for verify.

**format**=*json*|*yaml*
:   Serialisation for this invocation's manifest I/O. When omitted, the
    operative file extension decides, else the *manifest-format* default.

**state-path**=*path*
:   A captured actual-state dump for verify and diff (offline; default reads the
    live system).

**root**=*path*
:   Root to describe. Default "/".

**out**=*path*
:   describe output file. Default stdout.

**on-unreadable**=*error*|*warn*
:   describe: fail (default) or omit and warn on an unreadable scope source.

**scope**=*etc*|*full*
:   describe/verify read scope. *etc* (default) inspects only */etc*; *full* also
    audits */usr* and */boot* for integrity drift (expensive, opt-in).

The CONFIG knobs **manifest-format**, **repo-lock**, **content-store**,
**keep-list**, **signature-verification**, **keyring**, **activation-policy**,
and **applied-root** are also accepted as *key*=*value* options. A command-line
option overrides the corresponding preset value. Behaviour is never controlled
via environment variables.

# EXIT STATUS

**0**
:   Success: convergence complete or no-op, system matches declaration, or
    describe produced output.

**1**
:   Logical failure: convergence failed and was discarded; verify found drift;
    manifest invalid, unsafe-YAML, or unverified; state collection failed.

**2**
:   Invocation error: bad arguments; unknown format value; manifest unreadable;
    insufficient privilege; transaction mechanism unavailable; output path
    unwritable; malformed state dump.

# FILES

*/usr/lib/zypper-declarative/applied.json*
:   The applied record of a generation, stored inside the snapshot it describes.

*/etc/zypp/repos.d/*.repo*
:   The on-disk zypp repository configuration read for the repositories scope.

# SEE ALSO

**zypper**(8), **snapper**(8), **transactional-update**(8), **systemctl**(1)
