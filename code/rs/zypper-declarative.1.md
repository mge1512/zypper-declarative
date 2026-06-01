<!-- generated from spec: zypper-declarative.spec.md sha256:18253550a5c3d3f1818f0380811cb5dbc98874828693e49fc9cd5cbc923303dd -->
% ZYPPER-DECLARATIVE(1) zypper-declarative 0.6.4 | User Commands
%
% 2026-06-01

# NAME

zypper-declarative - declarative reconciling converger for SUSE systems

# SYNOPSIS

**zypper declarative** *verb* [*key*=*value* ...]

**zypper-declarative** *verb* [*key*=*value* ...]

# DESCRIPTION

**zypper-declarative** converges a SUSE system toward a desired manifest
inside a single snapshot transaction, recording what was applied. It is
surfaced as a zypper subcommand (`zypper declarative`) and is also invokable
directly. The desired state is the declarable subset of the SUSE Machinery
system description: the **packages**, **repositories**, **services**, and
**config_files** scopes. The same shared data model is produced by `describe`
(the actual state), stored as the applied record, and consumed by
`apply`/`diff`/`verify` (the desired state).

Options are *key*=*value* pairs and may precede or follow the verb. POSIX
`--flag` style is not used for options; `--version`, `--help`, and `-h` are
tolerated aliases for the `version` and `help` global commands. Behaviour is
never controlled via environment variables.

# VERBS

**apply**
: Converge the system to the desired manifest inside a snapshot transaction
  (privileged). Idempotent: a second run against an unchanged manifest and an
  undrifted system makes no changes.

**diff**
: Dry run. Compute and print what `apply` would change, modifying nothing and
  opening no transaction.

**verify**
: Check whether the actual state equals a reference declaration (the applied
  record by default, or a supplied `manifest-path`), modulo the keep-list.

**status**
: Print the current declarative state: applied manifest hash, generation, and a
  one-line drift summary. Read-only and fast.

**describe**
: Read the actual declarable state and emit it as a manifest (JSON by default,
  YAML on request). Bootstraps a manifest from a running system.

**version**
: Print the program name, version, and embedded specification hash.

**help**
: Print usage.

# OPTIONS

**mode**=auto|external|internal
: Transaction binding. Default *auto*.

**manifest-path**=*path*
: Desired manifest (apply, diff); reference manifest for verify (offline).

**format**=json|yaml
: Serialisation for this invocation's manifest I/O. When omitted, the operative
  file extension decides, else **manifest-format**.

**state-path**=*path*
: Captured actual state for verify and diff (offline; default reads live).

**root**=*path*
: Root to describe. Default `/`.

**out**=*path*
: describe output file. Default stdout.

**on-unreadable**=error|warn
: describe: fail (default) or omit-and-warn on an unreadable scope source. A
  source that cannot be read is never emitted as an empty scope.

**scope**=etc|full
: describe/verify read scope. *etc* (default) inspects only `/etc`; *full* also
  audits `/usr` and `/boot` (expensive, opt-in).

Other CONFIG knobs accepted as *key*=*value*: **manifest-format**, **repo-lock**,
**content-store**, **keep-list**, **signature-verification**, **keyring**,
**activation-policy**, **applied-root**.

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

# EXAMPLES

Bootstrap a manifest from the running system:

    zypper declarative describe > desired.json

Dry-run the plan:

    zypper declarative diff manifest-path=desired.json

Verify a captured state against a reference, fully offline:

    zypper declarative verify manifest-path=baseline.json state-path=after.json

# INSTALLATION

Distributed via the openSUSE Build Service. Install with the platform package
manager (`zypper install zypper-declarative`, `apt install zypper-declarative`,
or `dnf install zypper-declarative`). curl-based installation is not supported.

# SEE ALSO

zypper(8), snapper(8), transactional-update(8), systemctl(1)

# AUTHOR

Matthias G. Eckermann <pcd@mailbox.org>
