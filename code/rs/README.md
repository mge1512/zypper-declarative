<!-- generated from spec: zypper-declarative.spec.md sha256:18253550a5c3d3f1818f0380811cb5dbc98874828693e49fc9cd5cbc923303dd -->
# zypper-declarative

A declarative, reconciling converger for SUSE systems, surfaced as a
`zypper` subcommand and also invokable directly. It converges a system
toward a desired manifest inside a single snapshot transaction, recording
what was applied, and is idempotent: a second run against an unchanged
manifest and an undrifted system makes no changes.

- **Spec version:** 0.6.4
- **Spec SHA256:** `18253550a5c3d3f1818f0380811cb5dbc98874828693e49fc9cd5cbc923303dd`
- **License:** GPL-2.0-or-later
- **Language:** Rust (edition 2021), single static binary

## Data model

The desired manifest, the applied record, `describe` output, and any supplied
state dump all share one data model: the **declarable subset** of the SUSE
Machinery system description — the `packages`, `repositories`, `services`, and
`config_files` scopes, using the `ScopeWrapper` (`_attributes`/`_elements`)
idiom and `underscore_style` field names. The canonical serialisation is JSON
(Machinery `format_version` 1); YAML is an opt-in serialisation of the same
model, selected by `format=` or by file extension. The applied record is always
canonical JSON.

## Verbs

| Verb | Purpose |
|------|---------|
| `apply` | Converge the system inside a snapshot transaction (privileged). |
| `diff` | Dry run: print what `apply` would change (read-only). |
| `verify` | Check actual state against a reference, modulo the keep-list. |
| `status` | Print the applied state, generation, and a drift summary. |
| `describe` | Emit the actual declarable state as a manifest. |
| `version` | Print name, version, and embedded spec hash. |
| `help` | Print usage. |

## Usage

```
zypper declarative describe > desired.json          # bootstrap a manifest
zypper declarative diff manifest-path=desired.json  # dry run
zypper declarative apply                            # converge (privileged)
zypper declarative verify                           # declaration check (/etc)
zypper declarative verify scope=full                # + /usr,/boot integrity
zypper declarative status                           # current state
zypper declarative describe format=yaml > desired.yaml
zypper declarative verify manifest-path=baseline.json state-path=after.json
```

Equivalent direct form: `zypper-declarative <verb> [key=value ...]`.

### Options (key=value; precede or follow the verb)

```
mode=auto|external|internal       transaction binding (default auto)
manifest-path=<path>              desired/reference manifest
format=json|yaml                  serialisation for this invocation's I/O
state-path=<path>                 captured actual state (verify, diff)
root=<path>                       root to describe (default /)
out=<path>                        describe output file (default stdout)
on-unreadable=error|warn          describe: fail (default) or omit+warn
scope=etc|full                    describe/verify read scope (etc default)
```

Other CONFIG knobs accepted as key=value: `manifest-format`, `repo-lock`,
`content-store`, `keep-list`, `signature-verification`, `keyring`,
`activation-policy`, `applied-root`. Behaviour is never controlled via
environment variables. POSIX `--flag` style is not used for options;
`--version`, `--help`, and `-h` are tolerated aliases.

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | Success: converged, no-op, system matches declaration, or describe emitted. |
| 1 | Logical failure: convergence failed/discarded; verify drift; invalid/unsafe-YAML/unverified manifest; state collection failed. |
| 2 | Invocation error: bad arguments; unknown format value; manifest unreadable; insufficient privilege; transaction unavailable; output unwritable; malformed state dump. |

## Installation

Distributed via the [openSUSE Build Service](https://build.opensuse.org).
Install with your platform package manager:

```
# openSUSE / SLE
zypper install zypper-declarative

# Debian / Ubuntu
apt install zypper-declarative

# Fedora / RHEL
dnf install zypper-declarative
```

curl-based installation is not supported (supply-chain security requirement).

## Build from source

```
make build      # static release binary at ./zypper-declarative
make test       # build (dynamic) + run the independent test suite
make man        # render the man page via pandoc
make install    # install binary, man page, and zypper subcommand symlink
```

The static binary is produced with `RUSTFLAGS='-C target-feature=+crt-static'`
and an explicit `--target x86_64-unknown-linux-gnu`, so the resulting binary is
statically linked against glibc with no runtime library dependency for its own
logic.

## Target platforms

SL Micro 6.2 and SLES 16.1. Linux only.

## Notes

- `apply` requires privilege to modify the system and to open or operate within
  a snapshot transaction. The read-only verbs (`diff`, `verify`, `status`,
  `describe`) require only read access.
- The full integrity scan (`scope=full`) is expensive and opt-in; it is never
  engaged by default.
- The tool performs no direct network I/O of its own; all package retrieval is
  delegated to the package manager against a declared, pinned, signed repository.
