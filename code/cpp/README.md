# zypper-declarative

<!-- generated from spec: zypper-declarative.spec.md sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03 -->

Declarative system convergence for SUSE systems. `zypper-declarative` converges
a system to a desired manifest expressed in the **declarable subset** of the
SUSE Machinery system description — the `packages`, `repositories`, `services`,
and `config_files` scopes — inside a single snapshot transaction, records what
was applied, and is idempotent. It also reads the actual state into the same
data model (`describe`), prints a dry-run plan (`diff`), and checks drift
(`verify`).

It is surfaced as the `zypper declarative` subcommand and is also invokable
directly as `zypper-declarative`.

## Manifest format

The manifest is a typed data model. Its canonical serialisation is JSON
(Machinery `format_version` 1, the `_attributes`/`_elements` `ScopeWrapper`
idiom, `underscore_style` field names). YAML is an opt-in serialisation of the
identical model, selected by an explicit `format=` option or the file extension
(`.yaml` / `.yml`); the `manifest-format` default applies otherwise. JSON output
is Machinery-compatible; YAML output is not. The applied record is always
canonical JSON. Manifest identity (`desired_sha256`) is the hash of a canonical
serialisation of the parsed model, so the same intent in JSON or YAML yields the
same hash.

## Installation

Distributed via the openSUSE Build Service (OBS). curl-based installation is not
supported.

```sh
# openSUSE / SLE (zypper)
sudo zypper install zypper-declarative

# Fedora / RHEL family (dnf)
sudo dnf install zypper-declarative

# Debian / Ubuntu family (apt)
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
zypper declarative describe > desired.json              # bootstrap a manifest
zypper declarative describe format=yaml > desired.yaml
zypper declarative describe root=/mnt out=/tmp/state.json
zypper declarative describe scope=full out=/tmp/full.json   # include /usr,/boot
zypper declarative diff manifest-path=baseline.json state-path=after.json   # offline
zypper declarative verify manifest-path=baseline.json state-path=after.json # offline
```

Equivalent direct form: `zypper-declarative <verb> [key=value ...]`.

### Verbs

| Verb | Purpose |
|------|---------|
| `apply` | converge the system to the desired manifest |
| `diff` | print what `apply` would change (dry run) |
| `verify` | check the actual state against a reference, modulo the keep-list |
| `status` | print the current declarative state and a drift summary |
| `describe` | emit the actual state as a manifest |

`version` and `help` are bare-word global commands (with tolerated `--version`,
`--help`, `-h` aliases). A bare invocation prints usage and exits 0.

### Key=value options

Options are `key=value` pairs (POSIX `--flag` style is not used for options):
`mode`, `manifest-path`, `format`, `state-path`, `root`, `out`,
`on-unreadable`, `scope`, plus the CONFIG knobs `manifest-format`, `repo-lock`,
`content-store`, `keep-list`, `signature-verification`, `keyring`,
`activation-policy`, `applied-root`. A command-line option overrides the
corresponding preset value.

## Exit codes

| Code | Meaning |
|------|---------|
| `0` | success: converged, no-op, system matches declaration, or describe emitted |
| `1` | logical failure: convergence discarded; verify drift; invalid/unsafe/unverified manifest; state collection failed |
| `2` | invocation error: bad arguments; unknown format; manifest unreadable; insufficient privilege; transaction unavailable; output unwritable; malformed state dump |

Diagnostics go to stderr; normal output (summaries, plan, status, the describe
document) goes to stdout.

## Building from source

C++17, built with CMake against the distribution's shared libraries
(`libzypp`, `libsnapper`, `jsoncpp`, `yaml-cpp`, `libcrypto`) — dynamic
linking, no vendoring. On SLE 15 SP7 use the side-by-side GCC 15 (`gcc15-c++`,
`CXX=g++-15`); on SLE 16.0 the default toolchain is GCC 15.

```sh
make build      # configure + compile, copies the binary to the project root
make test       # build, then run the black-box test suite
make man        # render the man page (requires pandoc)
make install    # install binary, zypper subcommand, and man page
make clean
```

## Documentation

See the man page `zypper-declarative.1` (`man zypper-declarative`).
