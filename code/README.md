<!-- generated from spec: zypper-declarative.spec.md sha256:714e75ff672557d2c7344736a5f36b52afa37e0f565b07de83e1b18cc4492014 -->

# zypper-declarative

A declarative, reconciling converger for SUSE systems, surfaced as a
`zypper` subcommand and also invokable directly. It converges a system to a
*desired manifest* expressed in the declarable subset of the SUSE Machinery
system-description format — the `packages`, `repositories`, `services`, and
`config_files` scopes — inside a single snapshot transaction, recording what
was applied so that a rollback restores the matching declaration.

The canonical serialisation is **JSON** (Machinery `format_version` 1).
**YAML** is an opt-in serialisation of the identical data model, for
environments that author OS state in YAML.

## Installation

Distributed via the openSUSE Build Service (OBS). No `curl`-based install.

openSUSE / SLE (zypper):

```
zypper install zypper-declarative
```

Debian / Ubuntu derivatives (apt):

```
apt install zypper-declarative
```

Fedora family (dnf):

```
dnf install zypper-declarative
```

## Usage

```
zypper declarative <verb> [key=value ...]
zypper-declarative <verb> [key=value ...]
```

### Verbs

| Verb       | Effect                                                                 |
|------------|------------------------------------------------------------------------|
| `apply`    | Converge the system to the desired manifest inside a snapshot transaction. |
| `diff`     | Dry run: print what `apply` would change. Makes no modification.       |
| `verify`   | Check that the actual state equals the applied record (modulo keep-list). |
| `status`   | Print the applied manifest hash, generation, package count, drift summary. |
| `describe` | Read the actual state and emit it as a manifest (JSON default, YAML on request). |

### Examples

```
zypper declarative apply
zypper declarative apply mode=external
zypper declarative apply manifest-path=/var/lib/zypper-declarative/desired.json
zypper declarative diff
zypper declarative verify
zypper declarative verify state-path=/tmp/state.json
zypper declarative status
zypper declarative describe > desired.json
zypper declarative describe format=yaml > desired.yaml
zypper declarative describe root=/mnt out=/tmp/state.json
```

### Options (key=value)

Options use `key=value` syntax and precede bare-word arguments. Behaviour is
**not** controlled through environment variables.

| Option | Meaning | Default |
|--------|---------|---------|
| `mode=auto|external|internal` | transaction binding | `auto` |
| `manifest-path=<path>` | desired manifest | from config |
| `format=json|yaml` | input / describe output format | `json` (else by extension) |
| `state-path=<path>` | state dump as actual-state source for `verify` | live |
| `root=<path>` | root to describe / actual-state root | `/` |
| `out=<path>` | describe output file | stdout |
| `applied-root=<path>` | generation root for the applied record | `/` |
| `keep-list=<path>` | allowlist of persistent-but-undeclared paths | none |
| `content-store=<path>` | base path for resolving `content_ref` values | none |
| `repo-lock=<repo>` | fallback pinned repository | none |
| `signature-verification=on|off` | manifest signature checking | `on` |
| `keyring=<path>` | keyring path when verification is on | none |
| `activation-policy=reboot|soft-reboot|none` | activation of a sealed snapshot | `reboot` |
| `--version` | print version + spec hash, exit 0 | — |
| `--help` | print usage, exit 0 | — |

## Exit codes

| Code | Meaning |
|------|---------|
| `0` | Success: converged, no-op, system matches declaration, or describe emitted. |
| `1` | Logical failure: convergence failed and discarded; verify drift; invalid/unsafe-YAML/unverified manifest; state collection failed. |
| `2` | Invocation error: bad arguments; unknown format value; manifest unreadable; insufficient privilege; transaction mechanism unavailable; output path unwritable; malformed state dump. |

Diagnostics (errors and warnings) are written to **stderr**, one per line;
summaries, the diff plan, the status report, and the describe document are
written to **stdout**.

## How it works

- **Two diffs.** The *intent diff* (`compute-intent-diff`) compares the desired
  manifest against the applied record, scope by scope, and is the only source
  of deletions; it reads no filesystem. The *drift diff* (`compute-drift`)
  compares the live actual state against the declaration and is a pure
  comparison of two Manifest documents.
- **One live-state reader.** `describe` and every verb that needs actual state
  obtain it through a single reader (`describe-actual-state`), or from a
  supplied dump in the same schema.
- **Idempotence.** A second `apply` of an unchanged manifest against an
  undrifted system computes an empty intent diff and empty drift and exits 0
  without creating a new generation.
- **Safety.** `/etc/etc.syncpoint`, RPM-owned paths, and keep-listed paths are
  never written to or deleted. YAML is parsed under a safe profile (no
  code-executing tags, no anchors/aliases, single document, explicit typing).

## Platform

Linux only. Targets SL Micro 6.2 and SLES 16.1.

## License

GPL-2.0-or-later. See [LICENSE](LICENSE).
