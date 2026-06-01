% ZYPPER-DECLARATIVE(1) zypper-declarative 0.6.3 | User Commands
% Matthias G. Eckermann
% June 2026

<!-- generated from spec: zypper-declarative.spec.md sha256:87b9f2f3bf92afd6fe037412d56a23959290e5816d722a3def9505d07aa5acd7 -->

# NAME

zypper-declarative - declarative, reconciling converger for SUSE systems

# SYNOPSIS

**zypper declarative** *verb* [*key=value* ...]

**zypper-declarative** *verb* [*key=value* ...]

# DESCRIPTION

**zypper-declarative** converges a SUSE system to a desired manifest inside a
single snapshot transaction, recording what was applied. The manifest is the
declarable subset of the SUSE Machinery system description: **packages**,
**repositories**, **services**, and **config_files** (files under /etc). Its
canonical serialisation is JSON (Machinery format_version 1); YAML is an opt-in
serialisation of the identical data model.

The tool is surfaced as a *zypper declarative* subcommand and is also invocable
directly. It performs no direct network I/O of its own; all package retrieval is
delegated to the package manager against declared, pinned, signed repositories.

# VERBS

**apply**
: Converge the system to the desired manifest inside a snapshot transaction.
  Idempotent. Requires privilege.

**diff**
: Dry run. Print what **apply** would change. Opens no transaction and modifies
  nothing.

**verify**
: Check whether the actual state equals a reference declaration (the applied
  record by default, or a supplied manifest), modulo the keep-list.

**status**
: Print the current declarative state and a one-line drift summary.

**describe**
: Read the actual state of the declarable scopes and emit it as a Manifest.

**version**
: Print the program name, version, and embedded spec hash. Exit 0.

**help**
: Print usage. Exit 0.

# OPTIONS

Options are *key=value* pairs and may appear in any position, including after the
verb. POSIX `--flag` style is not used for options; `--version`, `--help`, and
`-h` are tolerated aliases for the **version** and **help** global commands only.

**mode=auto|external|internal**
: Transaction binding. Default *auto*.

**manifest-path=**\<path\>
: The desired manifest (apply, diff) or the reference manifest (verify).

**format=json|yaml**
: Serialisation for this invocation's manifest I/O. When omitted, the operative
  file extension decides, else **manifest-format**.

**manifest-format=json|yaml**
: Fallback serialisation used by resolve-format. Default *json*.

**state-path=**\<path\>
: Captured actual state for **verify** and **diff** (offline; default reads the
  live system).

**root=**\<path\>
: Root to describe. Default "/".

**out=**\<path\>
: Describe output file. Default stdout.

**on-unreadable=error|warn**
: How **describe** treats an unreadable scope source. Default *error*.

**scope=etc|full**
: Read scope for **describe** and **verify**. *etc* (default) inspects only /etc;
  *full* also audits /usr and /boot (expensive, opt-in).

Additional CONFIG knobs accepted as options: **repo-lock**, **content-store**,
**keep-list**, **signature-verification**, **keyring**, **activation-policy**,
**applied-root**.

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
: The applied record of a generation, stored within the snapshot it describes.

*/etc/etc.syncpoint*
: Never written or deleted by the converger.

# SEE ALSO

**zypper**(8), **snapper**(8), **transactional-update**(8), **systemctl**(1)
