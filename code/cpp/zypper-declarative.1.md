% ZYPPER-DECLARATIVE(1) zypper-declarative 0.6.6 | User Commands
% Matthias G. Eckermann
% June 2026

# NAME

zypper-declarative - declarative system convergence for zypper (Machinery subset)

# SYNOPSIS

**zypper declarative** *verb* [*key*=*value* ...]

**zypper-declarative** *verb* [*key*=*value* ...]

# DESCRIPTION

**zypper-declarative** converges a SUSE system to a desired manifest expressed
in the declarable subset of the SUSE Machinery system description: the
**packages**, **repositories**, **services**, and **config_files** scopes.
Convergence happens inside a single snapshot transaction and records what was
applied; running it twice against an unchanged manifest and an undrifted system
makes no changes (idempotent). The tool also reads the actual state of a system
into the same data model (**describe**), prints a dry-run plan (**diff**), and
checks drift against a reference (**verify**).

The manifest's canonical serialisation is JSON (Machinery `format_version` 1).
YAML is an opt-in input and output serialisation of the identical data model,
selected by an explicit **format=** option or by the file extension
(`.yaml`/`.yml`). The applied record is always canonical JSON.

# VERBS

**apply**
: Converge the system to the desired manifest inside a snapshot transaction.

**diff**
: Print what **apply** would change. Makes no modification and opens no
  transaction.

**verify**
: Check whether the actual state equals a reference declaration (the applied
  record by default, or a manifest given with **manifest-path=**), modulo the
  keep-list.

**status**
: Print the current declarative state: applied manifest hash, generation, and a
  one-line drift summary.

**describe**
: Read the actual state of the declarable scopes and emit it in the resolved
  serialisation.

# GLOBAL COMMANDS

**version** (alias **--version**)
: Print the program name, version, and the embedded spec hash; exit 0.

**help** (aliases **--help**, **-h**)
: Print usage; exit 0.

A bare invocation with no verb prints usage to stdout and exits 0.

# OPTIONS

Options are **key=value** pairs; POSIX `--flag` style is not used for options.

**mode=**auto|external|internal
: Transaction binding (default auto).

**manifest-path=**\<path\>
: Desired manifest (apply, diff); reference manifest for verify.

**format=**json|yaml
: Serialisation for this invocation's manifest I/O. When omitted, the file
  extension decides, else **manifest-format**.

**state-path=**\<path\>
: Captured actual state for verify and diff (offline; default reads live).

**root=**\<path\>
: Root to describe (default "/").

**out=**\<path\>
: describe output file (default stdout).

**on-unreadable=**error|warn
: describe: fail (default) or omit+warn on an unreadable scope source.

**scope=**etc|full
: describe/verify read scope; etc (default) is /etc only, full also audits
  /usr and /boot (expensive, opt-in).

Other CONFIG knobs accepted as options: **manifest-format**, **repo-lock**,
**content-store**, **keep-list**, **signature-verification**, **keyring**,
**activation-policy**, **applied-root**. A command-line option overrides the
corresponding preset value.

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
: The applied record of the current generation.

# SEE ALSO

**zypper**(8), **snapper**(8), **transactional-update**(8)
