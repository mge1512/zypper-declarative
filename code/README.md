<!-- generated from spec: zypper-declarative.spec.md sha256:b2d0de88fbed1163678e59e931c741b9d999b71f902f6eb01db8790bb813d057 -->

# zypper-declarative

A declarative, reconciling converger surfaced as a `zypper` subcommand. It
converges a SUSE system to a desired manifest — the *declarable subset* of the
SUSE Machinery system description (`packages`, `repositories`, `services`,
`config_files`) — inside a single snapshot transaction, recording what was
applied. It is idempotent: a second run against an unchanged manifest and an
undrifted system makes no changes.

The manifest is a typed data model. Its canonical serialisation is JSON
(Machinery `format_version` 1); YAML is an opt-in serialisation of the same
model, selected by `format=` or by file extension.

## Verbs

| Verb | Purpose |
|------|---------|
| `apply` | Converge the system to the desired manifest in a snapshot transaction. |
| `diff` | Dry run: print what `apply` would change. No modification, no transaction. |
| `verify` | Check the actual state against the applied declaration (drift detection). |
| `status` | Print the applied declaration and a one-line drift summary. |
| `describe` | Emit the actual state as a manifest (JSON by default, YAML on request). |
| `version` | Print program name, version, and the embedded spec hash. |
| `help` | Print usage. |

## Usage

```
zypper declarative <verb> [key=value ...]
# or the equivalent direct form:
zypper-declarative <verb> [key=value ...]
```

Options are `key=value` pairs and must precede any bare-word argument. POSIX
`--flag` style is not used for options; `--version`, `--help`, and `-h` are
tolerated aliases for the `version` and `help` global commands only. Behaviour
is never controlled via environment variables.

Examples:

```
zypper declarative apply
zypper declarative apply mode=external
zypper declarative diff manifest-path=/etc/zypper-declarative/desired.json
zypper declarative verify
zypper declarative verify state-path=/tmp/state.json
zypper declarative verify scope=full          # /etc + /usr,/boot integrity audit
zypper declarative status
zypper declarative describe > desired.json     # bootstrap a manifest
zypper declarative describe format=yaml > desired.yaml
zypper declarative describe scope=full out=/tmp/full-state.json
```

## Options

| Option | Meaning |
|--------|---------|
| `mode=auto\|external\|internal` | Transaction binding (default `auto`). |
| `manifest-path=<path>` | Desired manifest (default from CONFIG). |
| `manifest-format=json\|yaml` | Default serialisation (default `json`). |
| `format=json\|yaml` | Serialisation for this invocation's manifest I/O. |
| `state-path=<path>` | State dump as actual-state source for `verify`. |
| `root=<path>` | Root to `describe` (default `/`). |
| `out=<path>` | `describe` output file (default stdout). |
| `on-unreadable=error\|warn` | `describe`: fail (default) or omit+warn on an unreadable scope source. |
| `scope=etc\|full` | `describe`/`verify` read scope; `etc` (default) is `/etc` only, `full` also audits `/usr` and `/boot` (expensive, opt-in). |
| `repo-lock=<channel>` | Fallback pinned repository when the manifest declares none. |
| `content-store=<path>` | Base path against which `content_ref` values are resolved. |
| `keep-list=<path>` | Allowlist of persistent-but-undeclared paths never reported or deleted. |
| `signature-verification=on\|off` | Manifest signature checking (default `on`). |
| `keyring=<path>` | Keyring for signature verification. |
| `activation-policy=reboot\|soft-reboot\|none` | How `apply` schedules activation. |
| `applied-root=<path>` | Generation root for the applied record (default `/`). |

## Exit codes

| Code | Meaning |
|------|---------|
| `0` | Success: converged, no-op, system matches declaration, or `describe` emitted. |
| `1` | Logical failure: convergence failed and discarded; `verify` drift; invalid, unsafe-YAML, or unverified manifest; state collection failed. |
| `2` | Invocation error: bad arguments; unknown format value; manifest unreadable; insufficient privilege; transaction mechanism unavailable; output path unwritable; malformed state dump. |

Diagnostics (errors and warnings) are written to stderr, one per line. Normal
output (summaries, the diff plan, the status report, the `describe` document) is
written to stdout.

## Installation

Distributed via the openSUSE Build Service (OBS). `curl`-based installation is
not provided.

openSUSE / SLES / SL Micro (RPM):

```
sudo zypper install zypper-declarative
```

Fedora / RHEL family (DNF), once published to a compatible repository:

```
sudo dnf install zypper-declarative
```

Debian / Ubuntu (APT), once published to a compatible repository:

```
sudo apt install zypper-declarative
```

Target platforms: SL Micro 6.2 and SLES 16.1. Linux only.

## Building from source

```
make build      # static binary at ./zypper-declarative (CGO_ENABLED=0)
make test       # run the black-box test suite
make man        # render the man page via pandoc
make install    # install the binary, zypper subcommand symlink, and man page
```

The binary is a single static executable with no runtime dependencies of its
own beyond the system package manager, snapshot tooling, and init system it
drives.

## License

GPL-2.0-or-later. See `LICENSE`.
