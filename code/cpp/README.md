<!-- generated from spec: zypper-declarative.spec.md sha256:aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3 -->

# zypper-declarative

Declarative convergence of SUSE system state inside a snapshot transaction.

`zypper-declarative` converges a SUSE system (SL Micro 6.2, SLES 16.1) to a
declarative **manifest** describing the declarable subset of the SUSE Machinery
system description:

- **packages** — the resolved RPM set (the lock),
- **repositories** — the pinned zypp repositories the package set resolves against,
- **services** — systemd unit enablement (enabled / disabled / masked),
- **config_files** — files under `/etc` that differ from their package baseline
  or are unpackaged.

The manifest's canonical serialisation is **JSON** (Machinery `format_version`
1). **YAML** is an opt-in serialisation of the identical data model. The tool
reads the live system into this model itself (`describe`), so no separate
collector program is required. Convergence happens inside a single snapshot
transaction; on any failure the transaction is discarded and the running system
is left unchanged.

This C++ implementation links the native SUSE libraries directly
(libzypp, libsnapper, jsoncpp, yaml-cpp, libcrypto) and is **dynamically
linked** against each service pack's own shared libraries (no static binary, no
vendoring). It is the C++ sibling of the Go and Rust implementations of the same
specification.

## Installation

Distributed via the openSUSE Build Service (OBS). No `curl`-based install.

openSUSE / SLE (zypper):

```sh
zypper install zypper-declarative
```

Debian / Ubuntu (apt):

```sh
apt install zypper-declarative
```

Fedora / RHEL (dnf):

```sh
dnf install zypper-declarative
```

It installs both as a standalone binary (`/usr/bin/zypper-declarative`) and as a
**zypper subcommand** (`/usr/lib/zypper/commands/zypper-declarative`), so
`zypper declarative <verb>` and `zypper-declarative <verb>` are equivalent.

## Building from source

C++17, CMake, dynamic linking. Build dependencies: `cmake (>= 3.20)`,
`pkg-config`, `pandoc`, a C++17 compiler, and the devel packages
`libzypp-devel`, `libsnapper-devel`, `jsoncpp-devel`, `libyaml-cpp` devel, and
`libopenssl-3-devel`.

> **Compiler note.** On **SLE 15 SP7** the default `g++` is GCC 7 (too old for
> C++17); install `gcc15-c++` and build with `g++-15`. On **SLE 16.0** the
> default toolchain (GCC 15) suffices.

```sh
# SLE 15 SP7: select g++-15. SLE 16.0: omit CXX.
make build CXX=g++-15
make test                # builds and runs the black-box test suite
sudo make install
```

## Usage

```
zypper declarative <verb> [key=value ...]
zypper-declarative <verb> [key=value ...]
```

Verbs:

| Verb       | Purpose                                                           |
|------------|-------------------------------------------------------------------|
| `apply`    | Converge the system to the manifest inside a snapshot transaction. Idempotent. |
| `diff`     | Dry run: print what `apply` would change. No modification, no transaction. |
| `verify`   | Check the actual state against a reference declaration, modulo the keep-list. |
| `status`   | Print the applied manifest, generation, and a one-line drift summary. |
| `describe` | Read the actual state and emit it as a manifest (JSON or YAML).   |
| `init`     | Onboard a machine: adopt its current state as the managed baseline. |
| `version`  | Print name, version, and embedded spec hash. Exit 0.              |
| `help`     | Print usage. Exit 0.                                              |

Examples:

```sh
zypper declarative describe > desired.json          # bootstrap a manifest (JSON)
zypper declarative describe format=yaml > desired.yaml
zypper declarative describe scope=full out=/tmp/full.json   # include /usr and /boot
zypper declarative diff manifest-path=desired.json
zypper declarative verify
zypper declarative verify scope=full                 # + /usr,/boot integrity audit
zypper declarative diff manifest-path=baseline.json state-path=after.json  # offline
zypper declarative apply manifest-path=/var/lib/zypper-declarative/desired.json
zypper declarative init out=/var/lib/zypper-declarative/manifest.json
```

## Options (key=value)

Options are `key=value` pairs and may appear before or after the verb. POSIX
`--flag` style is **not** used for options; behaviour is never controlled via
environment variables.

| Option | Values | Meaning |
|--------|--------|---------|
| `mode` | `auto`\|`external`\|`internal` | Transaction binding. Default `auto`. |
| `manifest-path` | path | Desired manifest (apply, diff); reference manifest for verify. |
| `format` | `json`\|`yaml` | Serialisation for this invocation's manifest I/O. |
| `state-path` | path | Captured actual state for verify/diff (offline). |
| `root` | path | Root to describe. Default `/`. |
| `out` | path | describe output file. Default stdout. |
| `on-unreadable` | `error`\|`warn` | How a live read treats an unreadable source. Default `error`. Accepted by describe, diff, verify, and apply. |
| `scope` | `etc`\|`full` | Read scope for describe/verify. Default `etc`. `full` audits `/usr` and `/boot` (expensive). |

Additional CONFIG knobs accepted as key=value options: `manifest-format`,
`repo-lock`, `content-store`, `keep-list`, `signature-verification`, `keyring`,
`activation-policy`, `applied-root`. A command-line option overrides the
corresponding preset value.

Tolerated flag aliases (only for the two global commands): `--version`,
`--help`, `-h`.

## Exit codes

| Code | Meaning |
|------|---------|
| `0`  | Success: converged, no-op, system matches declaration, or describe produced output. |
| `1`  | Logical failure: convergence failed and discarded; verify found drift; manifest invalid, unsafe-YAML, or unverified; state collection failed. |
| `2`  | Invocation error: bad arguments; unknown format value; manifest unreadable; insufficient privilege; transaction mechanism unavailable; output path unwritable; malformed state dump. |

Diagnostics (errors and warnings) are written to **stderr**, one per line.
Summaries, the diff plan, the status report, and the describe document are
written to **stdout**.

## License

GPL-2.0-or-later. See [LICENSE](LICENSE).
