% ZYPPER-DECLARATIVE(1) zypper-declarative 0.4.0 | User Commands
%
% 2026-05-29

<!-- generated from spec: zypper-declarative.spec.md sha256:714e75ff672557d2c7344736a5f36b52afa37e0f565b07de83e1b18cc4492014 -->

# NAME

zypper-declarative - reconciling converger for declarative SUSE system state

# SYNOPSIS

**zypper declarative** *verb* [*key=value* ...]

**zypper-declarative** *verb* [*key=value* ...]

# DESCRIPTION

**zypper-declarative** converges a SUSE system to a desired manifest inside a
single snapshot transaction and records what was applied. The desired manifest
is the declarable subset of the SUSE Machinery system description (packages,
repositories, services, config_files), serialised as canonical JSON
(format_version 1) or, optionally, YAML.

It is surfaced as a **zypper** subcommand (an executable in
*/usr/lib/zypper/commands*) and is also invokable directly.

# VERBS

**apply**
: Converge the system to the desired manifest in a snapshot transaction.
  Idempotent.

**diff**
: Dry run. Compute and print what **apply** would change. Makes no
  modification and opens no transaction.

**verify**
: Check whether the actual state equals the applied declaration, modulo the
  keep-list.

**status**
: Print the current declarative state and a one-line drift summary.

**describe**
: Read the actual state of the declarable scopes and emit it as JSON (default)
  or YAML.

# OPTIONS

Options use *key=value* syntax and must precede any bare-word argument.
Behaviour is never controlled via environment variables.

**mode=auto|external|internal**
: Transaction binding. Default *auto*.

**manifest-path=**\<path>
: Desired manifest. Default from CONFIG.

**manifest-format=json|yaml**
: Default input serialisation. Default *json*.

**format=json|yaml**
: Explicit input format (load) and **describe** output format.

**state-path=**\<path>
: State dump used as the actual-state source for **verify**.

**root=**\<path>
: Root to **describe**. Default */*.

**out=**\<path>
: **describe** output file. Default standard output.

**repo-lock=**\<repo>
: Fallback pinned repository when the manifest declares none.

**content-store=**\<path>
: Base path against which content_ref values are resolved.

**keep-list=**\<path>
: Allowlist of persistent-but-undeclared paths never reported or deleted.

**signature-verification=on|off**
: Manifest signature check. Default *on*.

**keyring=**\<path>
: Keyring path used when verification is on.

**activation-policy=reboot|soft-reboot|none**
: How **apply** schedules activation of a freshly sealed snapshot.

**applied-root=**\<path>
: Generation root the applied record is read from or written to.

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

*/etc/etc.syncpoint*
: Never written to or deleted.

# INSTALLATION

Installed from an OBS package via build.opensuse.org (zypper, apt, or dnf
depending on distribution). curl-based installation is not supported.

# SEE ALSO

**zypper**(8), **transactional-update**(8), **snapper**(8), **systemctl**(1)
