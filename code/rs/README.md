<!-- generated from spec: zypper-declarative.spec.md sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03 -->

# zypper-declarative

Declarative convergence of SUSE system state — packages, repositories, services,
and `/etc` config files — to a desired manifest inside a single snapshot
transaction, recording what was applied.

`zypper-declarative` internalises the SUSE Machinery system-description
capability (`describe`), computes intent diffs and drift, and verifies the
converged tree. It is surfaced as the `zypper declarative` subcommand and is also
invocable directly. It builds to a single static binary with no runtime
dependencies of its own beyond the system package manager, snapshot tooling, and
init system it drives.

- **Spec:** `zypper-declarative.spec.md`
- **Spec-SHA256:** `51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03`
- **Version:** 0.6.6
- **License:** GPL-2.0-or-later
- **Module identity:** `zypper-declarative` (Rust crate name; repository
  `https://github.com/mge1512/zypper-declarative`)

## Installation

Distributed via the [openSUSE Build Service (OBS)](https://build.opensuse.org).
Install from the appropriate repository with your platform's package manager:

```sh
# openSUSE / SLES / SL Micro
sudo zypper install zypper-declarative

# Debian / Ubuntu
sudo apt install zypper-declarative

# Fedora / RHEL
sudo dnf install zypper-declarative
```

curl-based installation is **not** supported (supply-chain security
requirement).

## Building from source

The build is offline and vendored (no network fetch at build time):

```sh
make build      # builds the static release binary, copies it to ./zypper-declarative
make test       # runs the independent black-box test suite
make man        # renders the man page via pandoc
sudo make install
```

The binary is statically linked (`-C target-feature=+crt-static`) and placed at
the project root by `make build`.

## Usage

```
zypper declarative <verb> [key=value ...]
zypper-declarative <verb> [key=value ...]
```

### Verbs

| Verb       | Purpose                                                                 |
|------------|-------------------------------------------------------------------------|
| `apply`    | Converge the system to the desired manifest in a snapshot transaction.  |
| `diff`     | Dry run: print what `apply` would change. No modification.              |
| `verify`   | Check the actual state against a reference declaration (modulo keep-list). |
| `status`   | Print the current declarative state and a one-line drift summary.       |
| `describe` | Read the actual state and emit it as a manifest (JSON or YAML).         |

### Global commands

- `version` (alias `--version`) — print program name, version, and embedded spec hash.
- `help` (aliases `--help`, `-h`) — print usage.

A bare invocation (no verb) prints usage and exits 0; it never converges.

### Options (key=value, any position)

| Option | Meaning |
|--------|---------|
| `mode=auto\|external\|internal` | Transaction binding (default `auto`). |
| `manifest-path=<path>` | Desired manifest (apply, diff) / reference manifest (verify). |
| `state-path=<path>` | Captured actual state for verify/diff (offline). |
| `format=json\|yaml` | Serialisation for this invocation's manifest I/O. |
| `root=<path>` | Root to describe (default `/`). |
| `out=<path>` | Describe output file (default stdout). |
| `on-unreadable=error\|warn` | How describe treats an unreadable source (default `error`). |
| `scope=etc\|full` | Read scope for describe/verify; `etc` (default) or `full`. |
| `content-store=<path>` | Content-addressed file content store base path. |
| `keep-list=<path>` | Allowlist of persistent-but-undeclared paths. |
| `applied-root=<path>` | Generation root for the applied record (default `/`). |
| `manifest-format=json\|yaml` | Fallback serialisation default. |

Options use `key=value` only. POSIX `--flag` style is not used for options
(only the tolerated `--version`/`--help`/`-h` aliases). Behaviour is never
controlled by environment variables.

### Examples

```sh
# Bootstrap a desired manifest from a running system:
zypper declarative describe > desired.json
zypper declarative describe format=yaml > desired.yaml

# Dry run and verify:
zypper declarative diff manifest-path=/etc/zypper-declarative/desired.json
zypper declarative verify
zypper declarative verify scope=full          # /etc + /usr,/boot integrity audit

# Offline two-file comparisons (no live read, no applied record required):
zypper declarative diff   manifest-path=baseline.json state-path=after.json
zypper declarative verify manifest-path=baseline.json state-path=after.json

# Apply:
zypper declarative apply
zypper declarative apply mode=external
```

## Exit codes

| Code | Meaning |
|------|---------|
| `0`  | Success: converged, no-op, system matches declaration, or describe emitted. |
| `1`  | Logical failure: convergence failed/discarded; verify drift; invalid/unsafe-YAML/unverified manifest; state collection failed. |
| `2`  | Invocation error: bad arguments; unknown format value; manifest unreadable; insufficient privilege; transaction mechanism unavailable; output path unwritable; malformed state dump. |

Diagnostics (errors and warnings) are written to stderr, one per line; normal
output (summaries, the diff plan, the status report, the describe document) to
stdout.

## Manifest format

The manifest is a typed data model: the declarable subset of the SUSE Machinery
system description. Its canonical serialisation is JSON (`format_version` 1, the
`ScopeWrapper` `_attributes`/`_elements` idiom, `underscore_style` keys). YAML is
an opt-in serialisation of the same model, parsed under a safe profile (no
code-executing tags, no anchors/aliases, single document, explicit typing). JSON
output is Machinery-compatible; YAML output is not. The applied record is always
canonical JSON. The manifest identity (`desired_sha256`) is the hash of a
canonical serialisation of the parsed model, so the same intent in JSON or YAML
yields the same hash.

## Signal handling

`zypper-declarative` exits cleanly on `SIGTERM` and `SIGINT` with no partial
output. An interrupted `apply` discards the transaction; no partially converged
snapshot is left as the default boot target.
