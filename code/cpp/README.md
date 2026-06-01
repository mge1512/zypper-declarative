<!-- generated from spec: zypper-declarative.spec.md sha256:f2cc80627e483a48bb8411d297711bc5f6c6e74c28dbf0dafc8fe7bd8817251e -->
# zypper-declarative

A declarative reconciling converger for SUSE systems, surfaced as a zypper
subcommand. It converges a system to a desired **manifest** — the declarable
subset of the SUSE Machinery system description (`packages`, `repositories`,
`services`, `config_files`) — inside a single snapshot transaction, recording
what was applied.

Module identity: `github.com/mge1512/zypper-declarative`.
Spec version: 0.6.2. Spec SHA256:
`f2cc80627e483a48bb8411d297711bc5f6c6e74c28dbf0dafc8fe7bd8817251e`.

## Installation (via OBS)

Distributed via the openSUSE Build Service. Install with your system package
manager:

```sh
# openSUSE / SLE (RPM)
sudo zypper install zypper-declarative

# Fedora / RHEL family (RPM)
sudo dnf install zypper-declarative

# Debian / Ubuntu (DEB)
sudo apt install zypper-declarative
```

curl-based installation is **not** supported (supply-chain security
requirement).

## Usage

Invoke as a zypper subcommand or directly:

```sh
zypper declarative <verb> [key=value ...]
zypper-declarative <verb> [key=value ...]
```

Verbs:

| Verb | Purpose |
|------|---------|
| `apply` | Converge the system to the desired manifest (inside a snapshot transaction). |
| `diff` | Dry run: print what `apply` would change. No modification, no transaction. |
| `verify` | Check the actual state against a reference declaration (modulo the keep-list). |
| `status` | Print the current declarative state and a one-line drift summary. |
| `describe` | Emit the actual state of the declarable scopes as a manifest. |

Global commands (bare words, with tolerated flag aliases):

```sh
zypper-declarative version      # prints name, version, and the embedded spec hash
zypper-declarative help         # prints usage
zypper-declarative --version    # tolerated alias for version
zypper-declarative --help / -h  # tolerated aliases for help
```

### Examples

```sh
zypper declarative describe > desired.json          # bootstrap a manifest (JSON)
zypper declarative describe format=yaml > desired.yaml
zypper declarative diff manifest-path=desired.json
zypper declarative verify
zypper declarative verify scope=full                 # /usr + /boot integrity audit
zypper declarative diff manifest-path=baseline.json state-path=after.json  # offline
zypper declarative apply manifest-path=/etc/zypper-declarative/desired.json
```

## Options (key=value; accepted in any position)

`mode`, `manifest-path`, `format`, `state-path`, `root`, `out`,
`on-unreadable`, `scope`, plus the CONFIG knobs `manifest-format`, `repo-lock`,
`content-store`, `keep-list`, `signature-verification`, `keyring`,
`activation-policy`, `applied-root`. POSIX `--flag` style is not used for
options. Behaviour is never controlled via environment variables.

## Exit codes

| Code | Meaning |
|------|---------|
| `0` | Success (converged, no-op, matches declaration, or describe emitted). |
| `1` | Logical failure (convergence failed/discarded; verify drift; invalid/unsafe/unverified manifest; state collection failed). |
| `2` | Invocation error (bad arguments; unknown format value; manifest unreadable; insufficient privilege; transaction mechanism unavailable; output path unwritable; malformed state dump). |

## Building from source

C++17, built with CMake. Dependencies are **dynamically** linked distribution
shared libraries (no static binary, no vendoring): `libzypp`, `jsoncpp`,
`yaml-cpp`, `libsnapper`, and OpenSSL `libcrypto`.

On **SLE 15 SP7**, build with the side-by-side GCC 15 (`gcc15-c++`), since the
default `gcc-c++` is GCC 7 (too old for clean C++17). On **SLE 16** the default
toolchain is already GCC 15.

```sh
make build       # cmake configure + build, binary copied to project root
make test        # build and run the black-box test suite
make man         # generate the man page via pandoc
sudo make install
```

## Platform

Linux only. Target: SL Micro 6.2 and SLES 16.1.
