<!-- generated from spec: zypper-declarative.spec.md sha256:aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3 -->

# zypper-declarative

Declarative convergence of the declarable subset of a SUSE Machinery system
description (packages, repositories, services, config_files) to a desired
manifest, inside a single snapshot transaction, recording what was applied.

Module identity: `github.com/mge1512/zypper-declarative` (from spec META
`Module:` field).

This is the **C++17** implementation. It links the SUSE system libraries
**dynamically** (no static binary, no vendoring): libzypp (packages, rpmdb
query, per-file baseline), libsnapper (snapshots), jsoncpp (JSON), yaml-cpp
(opt-in YAML), and libcrypto (SHA256). Building per service pack via OBS links
each SP's own sonames.

## Installation (OBS)

Distributed via [build.opensuse.org](https://build.opensuse.org). Install with
your distribution's package manager — **never** via a curl-piped script.

```sh
# openSUSE / SLE (zypper)
zypper install zypper-declarative

# dnf-based systems
dnf install zypper-declarative

# Debian/Ubuntu-derived systems
apt install zypper-declarative
```

The package installs the binary as a **zypper subcommand** at
`/usr/lib/zypper/commands/zypper-declarative`, so it is invoked as
`zypper declarative <verb>` and also directly as `zypper-declarative <verb>`.

## Usage

```sh
zypper declarative apply
zypper declarative apply mode=external
zypper declarative apply manifest-path=/var/lib/zypper-declarative/desired.json
zypper declarative diff
zypper declarative verify
zypper declarative verify state-path=/tmp/state.json
zypper declarative status
zypper declarative describe                       # bootstrap a manifest (JSON, stdout)
zypper declarative describe format=yaml > desired.yaml
zypper declarative describe root=/mnt out=/tmp/state.json
zypper declarative describe scope=full out=/tmp/full-state.json
zypper declarative verify scope=full              # declaration + /usr,/boot audit
zypper declarative diff manifest-path=baseline.json state-path=after.json   # offline
zypper declarative init out=/var/lib/zypper-declarative/manifest.json
```

Equivalent direct form: `zypper-declarative <verb> [key=value ...]`.

### Verbs

| Verb | Purpose |
|---|---|
| `apply` | Converge the system to the desired manifest in a snapshot. Idempotent. |
| `diff` | Dry run: print what `apply` would change. No transaction. |
| `verify` | Check the actual state against a reference declaration. |
| `status` | Print the current declarative state. |
| `describe` | Read the actual state and emit it as a manifest. |
| `init` | Adopt the current state as the managed baseline. |
| `version` | Print version and embedded spec hash. |
| `help` | Print usage. |

### Options (key=value, any position)

`mode`, `manifest-path`, `state-path`, `format`, `root`, `out`,
`on-unreadable`, `scope`, `content-store`, `keep-list`, `repo-lock`,
`applied-root`, `manifest-format`, `signature-verification`, `keyring`,
`activation-policy`. POSIX `--flag` options are not used; behaviour is never
controlled via environment variables.

### Exit codes

| Code | Meaning |
|---|---|
| 0 | success (converged, no-op, matches declaration, or describe emitted) |
| 1 | logical failure (convergence failed/discarded; verify drift; invalid/unsafe/unverified manifest) |
| 2 | invocation error (bad arguments; unknown format; unreadable manifest; insufficient privilege; transaction unavailable; unwritable output; malformed dump) |

## Building from source

Requires the SUSE C/C++ toolchain and the devel packages:

- SLE 15 SP7: `gcc15-c++`, `libzypp-devel`, `libsnapper-devel`, `jsoncpp-devel`,
  `yaml-cpp-devel`, `libopenssl-3-devel`, `cmake`, `pandoc`. Build with
  `g++-15`.
- SLE 16.0: the default toolchain (GCC 15) suffices.

```sh
make build      # configure + cmake build, copies the binary to the project root
make test       # build, then run both independent black-box test suites
make man        # render the man page via pandoc
make dist       # produce the release source tarball
make install    # install the binary and man page under DESTDIR
```

## Snapshot library compatibility

libsnapper's API differs across service packs: soname 5 (0.8.x) exposes only
the one-argument snapshot-creation call; soname 7 (0.12.x) and 8 (0.13.x) added
a trailing `snapper::Plugins::Report&` parameter. The build detects the soname
major from `snapper/Version.h` and compiles the correct call (defining
`ZD_SNAPPER_REPORT_PARAM` for soname >= 7). Per-SP OBS builds therefore link
the right snapper and compile the right call.

## Transaction binding

The binding between the tool and the snapshot transaction is abstract
(`mode=auto|external|internal`). Under `external`, a separate mechanism
(`transactional-update run ...`) opens the transaction. Under `internal`, the
zypper-merged transactional machinery opens and commits it. `auto` detects
which applies. The convergence behaviour is identical either way.

## License

GPL-2.0-or-later. See `LICENSE`.
