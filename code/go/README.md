<!-- generated from spec: zypper-declarative.spec.md sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03 -->

# zypper-declarative

Declarative convergence of SUSE system state via Machinery manifests.

`zypper-declarative` converges a SUSE system to a **desired manifest** and reports
on its state. The manifest is the *declarable subset* of the SUSE Machinery
system description — **packages**, **repositories**, **services**, and
**config_files** — using the `ScopeWrapper` (`_attributes`/`_elements`) idiom and
`underscore_style` field names. Its canonical serialisation is JSON (Machinery
`format_version` 1); YAML is an opt-in serialisation of the identical data model.

The tool reads live system state itself (no separate collector required), and is
surfaced as a **zypper subcommand** (`/usr/lib/zypper/commands/zypper-declarative`)
as well as being invokable directly.

## Installation

Distributed as an OBS package via [build.opensuse.org](https://build.opensuse.org).
There is no curl-based installer (supply-chain security requirement).

```sh
# openSUSE / SLES / SL Micro
sudo zypper install zypper-declarative

# Debian / Ubuntu (from the OBS-built repository)
sudo apt install zypper-declarative

# Fedora / RHEL (from the OBS-built repository)
sudo dnf install zypper-declarative
```

## Usage

```sh
zypper declarative <verb> [key=value ...]
# equivalent direct form:
zypper-declarative <verb> [key=value ...]
```

### Verbs

| Verb       | Purpose                                                                 |
|------------|-------------------------------------------------------------------------|
| `apply`    | Converge the system to the desired manifest in a snapshot transaction.  |
| `diff`     | Dry run: print what `apply` would change. No modification.              |
| `verify`   | Check the actual state against a reference declaration (modulo keep-list). |
| `status`   | Print the applied manifest, generation, package count, drift summary.   |
| `describe` | Read the actual state and emit it as a Manifest (JSON or YAML). Read-only. |

### Global commands

- `version` (alias `--version`): prints program name, version, and the embedded
  spec hash, then exits 0.
- `help` (aliases `--help`, `-h`): prints usage, then exits 0.
- Bare invocation (no verb) prints usage and exits 0.

### Options (`key=value`)

Options use `key=value` pairs and precede any bare-word argument. POSIX `--flag`
style is **not** used for options (only the three global aliases above are
tolerated).

| Option | Values | Meaning |
|--------|--------|---------|
| `mode` | `auto` \| `external` \| `internal` | Transaction binding (default `auto`). |
| `manifest-path` | path | Desired manifest (apply, diff); reference for verify. |
| `state-path` | path | Captured actual state for verify/diff (offline). |
| `format` | `json` \| `yaml` | Serialisation for this invocation's manifest I/O. |
| `root` | path | Root to describe (default `/`). |
| `out` | path | Describe output file (default stdout). |
| `on-unreadable` | `error` \| `warn` | Describe: fail (default) or omit+warn. |
| `scope` | `etc` \| `full` | Describe/verify read scope (default `etc`). |

The CONFIG knobs `manifest-format`, `repo-lock`, `content-store`, `keep-list`,
`signature-verification`, `keyring`, `activation-policy`, and `applied-root` are
also accepted as `key=value` options. A command-line option overrides the
corresponding preset value. Behaviour is **never** controlled via environment
variables.

### Examples

```sh
zypper declarative describe > desired.json              # bootstrap a manifest (JSON)
zypper declarative describe format=yaml > desired.yaml
zypper declarative describe scope=full out=/tmp/full.json  # include /usr and /boot
zypper declarative diff manifest-path=desired.json
zypper declarative verify                               # declaration check (/etc)
zypper declarative verify scope=full                    # + /usr,/boot integrity audit
zypper declarative diff  manifest-path=baseline.json state-path=after.json   # offline
zypper declarative verify manifest-path=baseline.json state-path=after.json  # offline
zypper declarative apply manifest-path=/var/lib/zypper-declarative/desired.json
```

## Exit codes

| Code | Meaning |
|------|---------|
| `0`  | Success: converged, no-op, system matches declaration, or describe emitted. |
| `1`  | Logical failure: convergence failed and discarded; verify drift; invalid/unsafe-YAML/unverified manifest; state collection failed. |
| `2`  | Invocation error: bad arguments; unknown format value; manifest unreadable; insufficient privilege; transaction mechanism unavailable; output path unwritable; malformed state dump. |

Diagnostics (errors and warnings) are written to **stderr**, one per line; normal
output (summaries, the diff plan, the status report, the describe document) goes
to **stdout**.

## Building from source

```sh
make build      # CGO_ENABLED=0 static binary at the project root
make test       # build, then run the black-box test suite
make man        # render the man page via pandoc
make install    # install binary, zypper subcommand, and man page
```

The binary is a single static executable (`CGO_ENABLED=0`); it has no runtime
dependencies of its own beyond the system package manager, snapshot tooling, and
init system it drives.

## License

GPL-2.0-or-later. See [`LICENSE`](LICENSE).
