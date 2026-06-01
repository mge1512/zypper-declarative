% ZYPPER-DECLARATIVE(1) zypper-declarative 0.6.2
% Matthias G. Eckermann
% 2026-06-01

# NAME

zypper-declarative - reconciling declarative converger for SUSE systems,
surfaced as a zypper subcommand

# SYNOPSIS

**zypper declarative** *verb* [*key*=*value* ...]

**zypper-declarative** *verb* [*key*=*value* ...]

# DESCRIPTION

**zypper-declarative** converges a system to a desired manifest inside a single
snapshot transaction, recording what was applied. The manifest is the declarable
subset of the SUSE Machinery system description (packages, repositories,
services, config_files) using the ScopeWrapper idiom and underscore_style field
names. Its canonical serialisation is JSON (Machinery format_version 1); YAML is
an opt-in serialisation of the identical data model.

The tool reads live system state itself through a single internal reader, so no
separate collector program is required. Read-only verbs never modify the system.

# VERBS

**apply**
: Converge the system to the desired manifest inside a snapshot transaction.
  Idempotent. Requires privilege.

**diff**
: Print what **apply** would change. Makes no modification and opens no
  transaction.

**verify**
: Check whether the actual state equals a reference declaration, modulo the
  keep-list. Exit 0 on match, 1 on drift.

**status**
: Print the current declarative state: applied manifest hash, generation, and a
  one-line drift summary.

**describe**
: Read the actual state of the declarable scopes and emit it as a Manifest in
  the resolved serialisation (JSON by default, YAML on request).

# GLOBAL COMMANDS

**version**
: Print the program name, version, and embedded specification hash, then exit 0.
  The flag alias **--version** is tolerated.

**help**
: Print usage and exit 0. The flag aliases **--help** and **-h** are tolerated.

A bare invocation with no verb prints usage to stdout and exits 0 (discovery; it
never runs a default verb).

# OPTIONS

Options use *key*=*value* form and may appear before or after the verb. POSIX
**--flag** style is not used for options.

**mode**=auto|external|internal
: Transaction binding. Default auto.

**manifest-path**=*path*
: Desired manifest (apply, diff) or reference manifest (verify).

**state-path**=*path*
: Captured actual state for verify and diff (offline; default reads live).

**format**=json|yaml
: Serialisation for this invocation's manifest I/O.

**manifest-format**=json|yaml
: Default serialisation when no explicit format and no recognised extension.
  Default json.

**root**=*path*
: Root to describe. Default "/".

**out**=*path*
: describe output file. Default stdout.

**on-unreadable**=error|warn
: describe policy on an unreadable scope source. Default error.

**scope**=etc|full
: describe/verify read scope. Default etc; full additionally audits /usr and
  /boot (expensive, opt-in).

**applied-root**=*path*
: Generation root for the applied record. Default "/".

Additional CONFIG knobs accepted as options: **repo-lock**, **content-store**,
**keep-list**, **signature-verification**, **keyring**, **activation-policy**.

# EXIT STATUS

**0**
: Success: convergence complete or no-op, system matches declaration, or
  describe produced output.

**1**
: Logical failure: convergence failed and was discarded; verify found drift;
  manifest invalid, unsafe-YAML, or unverified; state collection failed.

**2**
: Invocation error: bad arguments; unknown format value; manifest unreadable;
  insufficient privilege; transaction mechanism unavailable; output path
  unwritable; malformed state dump.

# INSTALLATION

Distributed via the openSUSE Build Service (build.opensuse.org). Install with the
platform package manager:

    zypper install zypper-declarative
    apt install zypper-declarative
    dnf install zypper-declarative

curl-based installation is not provided.

# SEE ALSO

**zypper**(8), **transactional-update**(8), **snapper**(8)
