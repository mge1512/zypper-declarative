% ZYPPER-DECLARATIVE(1) zypper-declarative 0.6.6 | User Commands
% Matthias G. Eckermann
% June 2026

# NAME

zypper-declarative - declarative convergence of SUSE system state via Machinery manifests

# SYNOPSIS

**zypper declarative** *verb* [*key*=*value* ...]

**zypper-declarative** *verb* [*key*=*value* ...]

# DESCRIPTION

**zypper-declarative** converges a SUSE system to a desired manifest and reports
on its state. The manifest is the declarable subset of the SUSE Machinery system
description: **packages**, **repositories**, **services**, and **config_files**.
Its canonical serialisation is JSON (Machinery *format_version* 1); YAML is an
opt-in serialisation of the identical data model.

The tool reads live system state itself, so no separate collector program is
required. It is surfaced as a **zypper** subcommand (an executable in
*/usr/lib/zypper/commands*) and may be invoked directly.

# VERBS

**apply**
: Converge the system to the desired manifest inside a single snapshot
  transaction, recording what was applied. Idempotent.

**diff**
: Dry run. Compute and print what **apply** would change, modifying nothing and
  opening no transaction.

**verify**
: Check whether the actual state equals a reference declaration (the applied
  record by default, or a supplied manifest), modulo the keep-list.

**status**
: Print the current declarative state: applied manifest hash, generation,
  resolved package count, and a one-line drift summary.

**describe**
: Read the actual state of the declarable scopes and emit it as a Manifest in the
  resolved serialisation (JSON by default, YAML on request). Read-only.

# GLOBAL COMMANDS

**version**, **--version**
: Print the program name, version, and embedded spec hash; exit 0.

**help**, **--help**, **-h**
: Print usage; exit 0.

Bare invocation (no verb) prints usage and exits 0.

# OPTIONS

Options are *key*=*value* pairs and precede any bare-word argument. POSIX
**--flag** style is not used for options.

**mode**=auto|external|internal
: Transaction binding. Default *auto*.

**manifest-path**=*path*
: Desired manifest (apply, diff); reference manifest for verify (offline).

**state-path**=*path*
: Captured actual state for verify and diff (offline; default reads live).

**format**=json|yaml
: Serialisation for this invocation's manifest I/O. When omitted, the operative
  file extension decides, else **manifest-format**.

**root**=*path*
: Root to describe. Default */*.

**out**=*path*
: Describe output file. Default stdout.

**on-unreadable**=error|warn
: Describe: fail (default) or omit the scope and warn on an unreadable source.

**scope**=etc|full
: Describe/verify read scope. *etc* (default) inspects only */etc*; *full* also
  audits */usr* and */boot* (expensive, opt-in).

The CONFIG knobs **manifest-format**, **repo-lock**, **content-store**,
**keep-list**, **signature-verification**, **keyring**, **activation-policy**,
and **applied-root** are also accepted as *key*=*value* options.

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
: The applied record of the current generation.

# INSTALLATION

Installed via an OBS package from build.opensuse.org (zypper, apt, or dnf).
curl-based installation is not provided.

# SEE ALSO

**zypper**(8), **snapper**(8), **transactional-update**(8)
