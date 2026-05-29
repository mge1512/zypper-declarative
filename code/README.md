<!-- generated from spec: zypper-declarative.spec.md sha256:f8ff76ecbc4bbc69a49e2e32b2924da3a64df1ad46196e05ce8c137b684429b2 -->

# zypper-declarative

A declarative, reconciling converger for SUSE systems (SL Micro 6.2, SLES
16.1), surfaced as the zypper subcommand `zypper declarative` and also
invocable directly as `zypper-declarative`.

It converges a system to a *desired manifest* — the declarable subset of the
SUSE Machinery system-description format (packages, repositories, services,
config_files) — inside a single snapshot transaction, recording what was
applied. The canonical serialisation is JSON (`format_version` 1); YAML is an
opt-in serialisation of the identical data model.

## Verbs

| Verb | Purpose |
|------|---------|
| `apply` | Converge the system to the desired manifest in one snapshot transaction. Idempotent. |
| `diff` | Dry run: print what `apply` would change. No modification, no transaction. |
| `verify` | Check actual state against the applied declaration (live or from a state dump). |
| `status` | Print the applied manifest hash, generation, package count, and a drift summary. |
| `describe` | Emit the actual state of the declarable scopes as a manifest (JSON or YAML). |

Global commands: `version`, `help` (tolerated flag aliases: `--version`,
`--help`, `-h`). A bare invocation prints usage and exits 0.

## Usage

```
zypper declarative apply
zypper declarative apply mode=external
zypper declarative apply manifest-path=/var/lib/zypper-declarative/desired.json
zypper declarative diff
zypper declarative verify
zypper declarative verify state-path=/tmp/state.json
zypper declarative status
zypper declarative describe > desired.json            # bootstrap a manifest
zypper declarative describe format=yaml > desired.yaml
zypper declarative describe root=/mnt out=/tmp/state.json
```

Options are `key=value` pairs and precede any bare-word argument. POSIX
`--flag` style is not used (except the tolerated `version`/`help` aliases).
Behaviour is never controlled by environment variables; use `key=value`
options or systemd-style preset files.

### Options

| Option | Meaning |
|--------|---------|
| `mode=auto\|external\|internal` | Transaction binding; default auto. |
| `manifest-path=<path>` | Desired manifest; default from CONFIG. |
| `format=json\|yaml` | Serialisation for this invocation's manifest I/O. |
| `manifest-format=json\|yaml` | Fallback serialisation; default json. |
| `state-path=<path>` | State dump as the actual-state source for `verify`. |
| `root=<path>` | Root to describe; default "/". |
| `out=<path>` | `describe` output file; default stdout. |
| `on-unreadable=error\|warn` | `describe`: fail (default) or omit+warn on an unreadable source. |
| `repo-lock`, `content-store`, `keep-list`, `signature-verification`, `keyring`, `activation-policy`, `applied-root` | Additional CONFIG knobs. |

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | Success (converged, no-op, matches declaration, or describe emitted). |
| 1 | Logical failure (convergence failed and discarded; verify drift; invalid/unsafe-YAML/unverified manifest; state collection failed). |
| 2 | Invocation error (bad arguments; unknown format value; manifest unreadable; insufficient privilege; transaction mechanism unavailable; output path unwritable; malformed state dump). |

## Installation

Distributed via the openSUSE Build Service (OBS). Install with your
distribution's package manager:

```
# openSUSE / SLE
zypper install zypper-declarative

# Debian / Ubuntu
apt install zypper-declarative

# Fedora / RHEL
dnf install zypper-declarative
```

curl-based installation is not supported (supply-chain security requirement).

## Building from source

```
make build      # static binary at ./zypper-declarative (CGO_ENABLED=0)
make test       # run the independent test suite
make man        # build the man page via pandoc
make install    # install binary, zypper subcommand symlink, and man page
```

The build uses vendored dependencies (`go build -mod=vendor`). No network
access is required at build time.

## License

GPL-2.0-or-later. See `LICENSE`.
