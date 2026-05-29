% ZYPPER-DECLARATIVE(1) zypper-declarative 0.5.1 | User Commands
% Matthias G. Eckermann
% May 2026

<!-- generated from spec: zypper-declarative.spec.md sha256:f8ff76ecbc4bbc69a49e2e32b2924da3a64df1ad46196e05ce8c137b684429b2 -->

# NAME

zypper-declarative - declarative, reconciling converger surfaced as a zypper subcommand

# SYNOPSIS

**zypper declarative** *verb* [*key=value* ...]

**zypper-declarative** *verb* [*key=value* ...]

# DESCRIPTION

**zypper-declarative** converges a SUSE system (SL Micro 6.2, SLES 16.1) to a
desired manifest, inside a single snapshot transaction, recording what was
applied. The manifest is the declarable subset of the SUSE Machinery
system-description format (packages, repositories, services, config_files),
serialised as canonical JSON (`format_version` 1) or, optionally, as YAML.

Options are *key=value* pairs and precede any bare-word argument. POSIX
`--flag` style is not used, except as tolerated aliases for the global commands
**version** and **help**. Behaviour is never controlled by environment
variables; use *key=value* options or preset files.

# VERBS

**apply**
: Converge the system to the desired manifest in one snapshot transaction.
  Idempotent: a second run against an unchanged manifest and an undrifted
  system makes no changes.

**diff**
: Dry run. Compute and print what **apply** would change, making no
  modification and opening no transaction.

**verify**
: Check whether the actual state equals the applied declaration, modulo the
  keep-list. Reads live state by default, or a supplied state dump.

**status**
: Print the current declarative state: the applied manifest hash, the
  generation, the resolved package count, and a one-line drift summary.

**describe**
: Read the actual state of the declarable scopes and emit it as a manifest in
  the resolved serialisation (JSON by default, YAML on request).

# GLOBAL COMMANDS

**version**
: Print the program name, version, and embedded spec hash to stdout; exit 0.
  Tolerated alias: `--version`.

**help**
: Print usage to stdout; exit 0. Tolerated aliases: `--help`, `-h`.

A bare invocation with no verb prints usage to stdout and exits 0.

# OPTIONS

**mode=**auto|external|internal
: Transaction binding; default auto.

**manifest-path=**\*path\*
: Desired manifest; default from CONFIG.

**format=**json|yaml
: Serialisation for this invocation's manifest I/O. When omitted, the operative
  file extension decides, else **manifest-format**.

**manifest-format=**json|yaml
: Fallback serialisation; default json.

**state-path=**\*path\*
: State dump used as the actual-state source for **verify**.

**root=**\*path\*
: Root to describe; default "/".

**out=**\*path\*
: **describe** output file; default stdout.

**on-unreadable=**error|warn
: For **describe**: fail (default) or omit+warn on an unreadable source; never
  emit an empty scope.

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
: The applied record of a generation. Always canonical JSON.

*/etc/zypp/repos.d/*.repo*
: The on-disk zypp repository configuration, read for the repositories scope.

# INSTALLATION

Install via the openSUSE Build Service (OBS): `zypper install
zypper-declarative` (or the distribution's `apt`/`dnf` equivalent). curl-based
installation is not supported.

# SEE ALSO

**zypper**(8), **snapper**(8), **transactional-update**(8), **systemctl**(1)
