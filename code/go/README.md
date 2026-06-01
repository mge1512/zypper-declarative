<!-- generated from spec: zypper-declarative.spec.md sha256:f2cc80627e483a48bb8411d297711bc5f6c6e74c28dbf0dafc8fe7bd8817251e -->

# zypper-declarative

A reconciling declarative converger for SUSE systems, surfaced as a `zypper`
subcommand and also directly invocable. It converges a system to a desired
manifest inside a single snapshot transaction and records what was applied.

The manifest is the **declarable subset of the SUSE Machinery system
description** — `packages`, `repositories`, `services`, `config_files` — using
the `ScopeWrapper` (`_attributes`/`_elements`) idiom and `underscore_style`
field names. Its canonical serialisation is JSON (Machinery `format_version`
1); YAML is an opt-in serialisation of the identical data model.

This binary embeds the SHA256 of the specification it was generated from; see
`zypper-declarative version`.

## Installation

Distributed via the [openSUSE Build Service](https://build.opensuse.org). Install
with your platform package manager:

```
# openSUSE / SLE
zypper install zypper-declarative

# Debian / Ubuntu
apt install zypper-declarative

# Fedora / RHEL
dnf install zypper-declarative
```

curl-based installation is intentionally **not** provided (supply-chain
security requirement).

## Usage

```
zypper declarative <verb> [key=value ...]
zypper-declarative <verb> [key=value ...]
```

### Verbs

| Verb | Purpose |
|------|---------|
| `apply` | Converge the system to the desired manifest inside a snapshot transaction (privileged, idempotent). |
| `diff` | Print what `apply` would change. Makes no modification, opens no transaction. |
| `verify` | Check the actual state against a reference declaration, modulo the keep-list. |
| `status` | Print the applied manifest hash, generation, and a one-line drift summary. |
| `describe` | Emit the actual state as a Manifest (JSON by default, YAML on request). |

### Global commands

| Command | Purpose |
|---------|---------|
| `version` (alias `--version`) | Print program name, version, and embedded spec hash; exit 0. |
| `help` (aliases `--help`, `-h`) | Print usage; exit 0. |

A bare invocation (no verb) prints usage to stdout and exits 0. It is a
discovery action and never runs a default verb.

### Options

Options use `key=value` form and may appear before or after the verb. POSIX
`--flag` style is **not** used for options.

| Option | Values | Meaning |
|--------|--------|---------|
| `mode` | `auto`\|`external`\|`internal` | Transaction binding (default `auto`). |
| `manifest-path` | path | Desired manifest (apply, diff); reference manifest (verify). |
| `state-path` | path | Captured actual state for verify/diff (offline; default reads live). |
| `format` | `json`\|`yaml` | Serialisation for this invocation's manifest I/O. |
| `manifest-format` | `json`\|`yaml` | Default serialisation (default `json`). |
| `root` | path | Root to describe (default `/`). |
| `out` | path | describe output file (default stdout). |
| `on-unreadable` | `error`\|`warn` | describe policy on an unreadable scope source (default `error`). |
| `scope` | `etc`\|`full` | describe/verify read scope (default `etc`; `full` audits `/usr` and `/boot`). |
| `applied-root` | path | Generation root for the applied record (default `/`). |

Additional CONFIG knobs accepted as options: `repo-lock`, `content-store`,
`keep-list`, `signature-verification`, `keyring`, `activation-policy`.

### Examples

```
zypper declarative apply
zypper declarative apply manifest-path=/var/lib/zypper-declarative/desired.json
zypper declarative diff
zypper declarative describe > desired.json            # bootstrap a manifest
zypper declarative describe format=yaml > desired.yaml
zypper declarative verify
zypper declarative verify scope=full                  # + /usr,/boot integrity audit
zypper declarative verify manifest-path=baseline.json state-path=after.json   # offline
```

## Exit codes

| Code | Meaning |
|------|---------|
| `0` | Success: converged or no-op, system matches declaration, or describe produced output. |
| `1` | Logical failure: convergence failed and discarded; verify drift; invalid/unsafe-YAML/unverified manifest; state collection failed. |
| `2` | Invocation error: bad arguments; unknown format value; manifest unreadable; insufficient privilege; transaction mechanism unavailable; output path unwritable; malformed state dump. |

Diagnostics (errors and warnings) are written to **stderr**, one per line.
Normal output (summaries, the diff plan, the status report, the describe
document) is written to **stdout**.

## Building from source

```
make build      # static binary at ./zypper-declarative (CGO_ENABLED=0)
make test       # run the test suite
make man        # generate the troff man page via pandoc
make install    # install binary, zypper subcommand, and man page
```

## License

GPL-2.0-or-later. See `LICENSE`.
