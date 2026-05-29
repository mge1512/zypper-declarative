<!-- generated from spec: zypper-declarative.spec.md sha256:58e1636e2de82ab81a5cd3f81d6b3c9ac6a8976e18f9abb2bbd2b2aba56fe4d4 -->

# zypper-declarative

A declarative, reconciling converger for SUSE systems, surfaced as a `zypper`
subcommand and also invocable directly. It converges a system to a desired
**manifest** — the declarable subset of the SUSE Machinery system description
(`packages`, `repositories`, `services`, `config_files`) — inside a single
snapshot transaction, recording what was applied so it can be restored on
rollback.

The manifest is a typed data model. Its canonical serialisation is JSON
(Machinery `format_version` 1); YAML is an opt-in serialisation of the identical
model, for environments that author OS state in YAML.

## Installation

Distribution is via the openSUSE Build Service (OBS). curl-based installation is
not supported.

**openSUSE / SLE (zypper):**

```sh
zypper install zypper-declarative
```

**Fedora / RHEL (dnf):**

```sh
dnf install zypper-declarative
```

**Debian / Ubuntu (apt):**

```sh
apt install zypper-declarative
```

## Usage

```sh
zypper declarative apply
zypper declarative apply mode=external
zypper declarative apply manifest-path=/var/lib/zypper-declarative/desired.json
zypper declarative diff
zypper declarative verify
zypper declarative verify state-path=/tmp/state.json
zypper declarative status
zypper declarative describe
zypper declarative describe > desired.json            # bootstrap a JSON manifest
zypper declarative describe format=yaml > desired.yaml
zypper declarative describe root=/mnt out=/tmp/state.json
```

Equivalent direct form: `zypper-declarative <verb> [key=value ...]`.

### Verbs

| Verb | Effect |
|------|--------|
| `apply` | Converge the system to the desired manifest (privileged). Idempotent. |
| `diff` | Print what `apply` would change (dry run, read-only). |
| `verify` | Check whether actual state equals the declaration (read-only). |
| `status` | Print the current declarative state (read-only). |
| `describe` | Emit the actual state as a manifest (read-only). |

### Options (key=value)

Options are `key=value` pairs and precede any bare-word argument. Behaviour is
**not** controlled via environment variables.

| Option | Meaning |
|--------|---------|
| `mode=auto\|external\|internal` | Transaction binding; default `auto`. |
| `manifest-path=<path>` | Desired manifest; default from CONFIG. |
| `format=json\|yaml` | Serialisation for this invocation's manifest I/O. |
| `state-path=<path>` | State dump as actual-state source for `verify`. |
| `root=<path>` | Root to describe; default `/`. |
| `out=<path>` | `describe` output file; default stdout. |
| `on-unreadable=error\|warn` | `describe`: fail (default) or omit-and-warn. |
| `manifest-format=json\|yaml` | Fallback serialisation default. |
| `repo-lock=<repo>` | Fallback pinned repository. |
| `content-store=<path>` | Base path for `content_ref` resolution. |
| `keep-list=<path>` | Allowlist of persistent-but-undeclared paths. |
| `signature-verification=on\|off` | Manifest signature verification; default `on`. |
| `keyring=<path>` | Keyring path when verification is on. |
| `activation-policy=reboot\|soft-reboot\|none` | Activation scheduling for a sealed snapshot. |
| `applied-root=<path>` | Generation root for the applied record; default `/`. |

### Global flags

| Flag | Effect |
|------|--------|
| `--version` | Print program name, version, and embedded spec hash; exit 0. |
| `--help`, `-h` | Print usage; exit 0. |

A bare invocation (`zypper declarative` with no verb) prints usage to stdout and
exits 0 — it is a discovery action and never runs a default verb.

## Exit codes

| Code | Meaning |
|------|---------|
| `0` | Success: converged, no-op, system matches declaration, or describe emitted. |
| `1` | Logical failure: convergence failed and discarded; verify drift; invalid, unsafe-YAML, or unverified manifest; state collection failed. |
| `2` | Invocation error: bad arguments; unknown format value; manifest unreadable; insufficient privilege; transaction mechanism unavailable; output path unwritable; malformed state dump. |

Diagnostics (errors and warnings) are written to **stderr**, one per line.
Normal output (summaries, the diff plan, the status report, the describe
document) is written to **stdout**.

## Building from source

```sh
make build      # CGO_ENABLED=0 go build -o zypper-declarative ./cmd/zypper-declarative
make test       # run the test suite
make man        # render the man page via pandoc
make install    # install the binary and man page
```

The result is a single statically-linked binary with no runtime dependencies of
its own beyond the system package manager, snapshot tooling, and init system it
drives.

## License

GPL-2.0-or-later. See [LICENSE](LICENSE).
