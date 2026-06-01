<!-- generated from spec: zypper-declarative.spec.md sha256:27aee8e374eb3507189bad0b78339109d0116e6a13b55ae4df2ba9a18e769fc4 -->

# zypper-declarative

Declarative system convergence for zypper-managed systems.

`zypper-declarative` converges a SUSE system to a **desired manifest** — the
declarable subset of the SUSE Machinery system description (`packages`,
`repositories`, `services`, `config_files`) — inside a single snapshot
transaction, recording what was applied. It is surfaced as the
`zypper declarative` subcommand and is also invocable directly.

The manifest's canonical serialisation is JSON (Machinery `format_version` 1).
YAML is an opt-in alternative serialisation of the identical data model.

- **Module:** `github.com/mge1512/zypper-declarative`
- **Spec:** `zypper-declarative.spec.md` (sha256 `27aee8e374eb3507189bad0b78339109d0116e6a13b55ae4df2ba9a18e769fc4`)
- **License:** GPL-2.0-or-later

## Installation

`zypper-declarative` is distributed as an OBS package via
[build.opensuse.org](https://build.opensuse.org). Install it with your
distribution's package manager:

```sh
# openSUSE / SLES / SL Micro
sudo zypper install zypper-declarative

# Fedora / RHEL (dnf)
sudo dnf install zypper-declarative

# Debian / Ubuntu (apt)
sudo apt install zypper-declarative
```

There is no `curl`-based installation: software is delivered through signed
distribution packages only.

## Usage

```sh
zypper declarative apply                       # converge to the desired manifest
zypper declarative apply mode=external
zypper declarative diff                        # dry run: what apply would change
zypper declarative verify                      # does the system still match?
zypper declarative verify scope=full           # + /usr,/boot integrity audit
zypper declarative status                      # current declarative state
zypper declarative describe                    # emit actual state as a manifest
zypper declarative describe > desired.json     # bootstrap a manifest (JSON)
zypper declarative describe format=yaml > desired.yaml
zypper declarative diff   manifest-path=baseline.json state-path=after.json   # offline
zypper declarative verify manifest-path=baseline.json state-path=after.json   # offline
```

The equivalent direct form is `zypper-declarative <verb> [key=value ...]`.

### Options (key=value; precede any bare-word argument)

| Option | Values | Meaning |
|--------|--------|---------|
| `mode` | `auto`\|`external`\|`internal` | transaction binding (default `auto`) |
| `manifest-path` | path | desired manifest (apply, diff); reference for verify |
| `format` | `json`\|`yaml` | serialisation for this invocation's manifest I/O |
| `state-path` | path | captured actual state for verify and diff (offline) |
| `root` | path | root to describe (default `/`) |
| `out` | path | describe output file (default stdout) |
| `on-unreadable` | `error`\|`warn` | describe: fail (default) or omit+warn |
| `scope` | `etc`\|`full` | describe/verify read scope (default `etc`) |

The CONFIG knobs `manifest-format`, `repo-lock`, `content-store`, `keep-list`,
`signature-verification`, `keyring`, `activation-policy`, and `applied-root` are
also accepted as `key=value` options. Behaviour is never controlled via
environment variables; POSIX `--flag` style is not used for options
(`--version`, `--help`, `-h` are tolerated aliases for the global commands
only).

### Exit codes

| Code | Meaning |
|------|---------|
| `0` | success: converged, no-op, system matches declaration, or describe emitted |
| `1` | logical failure: convergence discarded; verify drift; invalid/unsafe/unverified manifest; state collection failed |
| `2` | invocation error: bad arguments; unknown format value; manifest unreadable; insufficient privilege; transaction unavailable; output path unwritable; malformed state dump |

## Building from source

```sh
make build      # CGO_ENABLED=0 static binary at ./zypper-declarative
make test       # run the test suite (go test)
make man        # generate the man page via pandoc
make install    # install binary, zypper subcommand, and man page
```

The binary is a single static executable (`CGO_ENABLED=0`) with no runtime
dependencies of its own beyond the system package manager, snapshot tooling, and
init system it drives.

## Architecture

| Package | Responsibility |
|---------|----------------|
| `cmd/zypper-declarative` | thin entry point: dispatch only |
| `internal/cli` | argument parsing, global contract, verb handlers |
| `internal/manifest` | the data model, JSON/YAML serialisation, `resolve-format`, canonical-model hashing, diagnostics |
| `internal/state` | `describe-actual-state` — the single live-state reader |
| `internal/diff` | `compute-intent-diff`, `compute-drift` (pure, no I/O) |
| `internal/converge` | `converge-packages`, `-files`, `-units` |
| `internal/txn` | `acquire-transaction-context` and bindings |
| `internal/record` | `load-applied-record`, `write-applied-record` |
| `internal/meta` | embedded spec SHA256 and version |

`describe-actual-state` is the only code that reads live system state; every
verb obtains actual state through it (or a supplied dump). `compute-drift`
performs no I/O — it is a pure comparison of two in-memory `Manifest` documents.
