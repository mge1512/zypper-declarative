<!-- generated from spec: zypper-declarative.spec.md sha256:87b9f2f3bf92afd6fe037412d56a23959290e5816d722a3def9505d07aa5acd7 -->

# zypper-declarative

A declarative, reconciling converger for SUSE systems, surfaced as a
`zypper declarative` subcommand and invocable directly. It converges a system to
a desired manifest — the declarable subset of the SUSE Machinery system
description (**packages**, **repositories**, **services**, and `/etc`
**config_files**) — inside a single snapshot transaction, recording what was
applied.

The tool performs **no direct network I/O of its own**; all package retrieval is
delegated to the package manager against declared, pinned, signed repositories.
Read-only verbs (`diff`, `verify`, `status`, `describe`) require only read access;
`apply` requires privilege and the selected transaction mechanism.

- Target: SL Micro 6.2 and SLES 16.1, Linux only.
- Single static binary, no runtime dependencies of its own.
- Spec: `zypper-declarative.spec.md`, Version 0.6.3,
  sha256 `87b9f2f3bf92afd6fe037412d56a23959290e5816d722a3def9505d07aa5acd7`.

## Installation (via OBS)

Distributed via [build.opensuse.org](https://build.opensuse.org). Install with
your distribution's package manager. **There is no curl-based installer** (a
supply-chain security requirement).

```sh
# openSUSE / SLE (zypper)
sudo zypper install zypper-declarative

# Fedora / RHEL family (dnf), where packaged
sudo dnf install zypper-declarative

# Debian / Ubuntu (apt), where packaged
sudo apt install zypper-declarative
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
zypper declarative describe scope=full out=/tmp/full-state.json
zypper declarative verify scope=full                  # /usr,/boot integrity audit
zypper declarative diff manifest-path=baseline.json state-path=after.json   # offline
zypper declarative verify manifest-path=baseline.json state-path=after.json # offline
```

Equivalent direct form: `zypper-declarative <verb> [key=value ...]`.

### Verbs

| Verb       | Purpose                                                                |
|------------|------------------------------------------------------------------------|
| `apply`    | Converge the system to the desired manifest (idempotent; needs privilege). |
| `diff`     | Dry run: print what `apply` would change. No modification, no transaction. |
| `verify`   | Check the actual state against a reference declaration, modulo the keep-list. |
| `status`   | Print the applied manifest, generation, and a one-line drift summary.  |
| `describe` | Read the actual state of the declarable scopes and emit it as a Manifest. |
| `version`  | Print program name, version, and embedded spec hash. Exit 0.           |
| `help`     | Print usage. Exit 0.                                                    |

### Options (key=value)

Options are `key=value` pairs and may appear **in any position**, including after
the verb. POSIX `--flag` style is not used for options; `--version`, `--help`, and
`-h` are tolerated aliases for the `version` and `help` global commands only.

| Option | Values | Default | Notes |
|--------|--------|---------|-------|
| `mode` | `auto`/`external`/`internal` | `auto` | Transaction binding. |
| `manifest-path` | path | staging path | Desired (apply/diff) or reference (verify) manifest. |
| `format` | `json`/`yaml` | — | Serialisation for this invocation; else extension, else `manifest-format`. |
| `manifest-format` | `json`/`yaml` | `json` | Fallback serialisation (resolve-format default). |
| `state-path` | path | — | Captured actual state for `verify`/`diff` (offline). |
| `root` | path | `/` | Root to describe. |
| `out` | path | stdout | Describe output file. |
| `on-unreadable` | `error`/`warn` | `error` | Describe unreadable-source policy. |
| `scope` | `etc`/`full` | `etc` | Describe/verify read scope (`full` = +/usr,/boot, expensive). |
| `repo-lock` | repo | — | Fallback pinned repository. |
| `content-store` | path | — | Base for `content_ref` resolution. |
| `keep-list` | path | — | Allowlist of persistent undeclared paths. |
| `signature-verification` | `on`/`off` | `on` | Manifest signature verification. |
| `keyring` | path | — | Signature keyring. |
| `activation-policy` | `reboot`/`soft-reboot`/`none` | `none` | Apply activation policy. |
| `applied-root` | path | `/` | Generation root for the applied record. |

Behaviour must not be controlled via environment variables; use `key=value`
options or preset files.

### Exit codes

| Code | Meaning |
|------|---------|
| `0`  | Success: converged, no-op, system matches declaration, or describe emitted. |
| `1`  | Logical failure: convergence failed/discarded; verify drift; invalid, unsafe-YAML, or unverified manifest; state collection failed. |
| `2`  | Invocation error: bad arguments; unknown format value; manifest unreadable; insufficient privilege; transaction unavailable; output path unwritable; malformed state dump. |

Diagnostics (errors and warnings) are written to stderr, one per line; normal
output (summaries, the diff plan, the status report, the describe document) is
written to stdout.

## Manifest format

The manifest is a typed data model — the declarable subset of the SUSE Machinery
system description. JSON is canonical (`format_version` 1, Machinery-compatible);
YAML is an opt-in serialisation of the identical model (not Machinery). A JSON
dump is still accepted as YAML input for `verify`. The applied record is always
canonical JSON. YAML is parsed under a safe profile: no code-executing or
arbitrary tags, bounded/disabled anchor-alias expansion, a single document, and
explicit typing per the schema.

Manifest identity (`desired_sha256`) is the hash of a canonical serialisation of
the parsed data model, so the same intent in JSON or YAML yields the same hash and
idempotence holds across a format switch.

## Building from source

```sh
make build      # cargo build --release -> static binary copied to ./zypper-declarative
make test       # run the library unit tests and the black-box integration suite
make man        # generate the troff man page via pandoc
make install    # install the binary, man page, and zypper subcommand surface
make clean
```

Dependencies are vendored under `vendor/` for offline (OBS) builds
(`cargo build --release --offline`). The binary is statically linked against
glibc (configured in `.cargo/config.toml`).

## License

GPL-2.0-or-later. See [`LICENSE`](LICENSE).
