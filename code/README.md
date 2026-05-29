<!-- generated from spec: zypper-declarative.spec.md sha256:714e75ff672557d2c7344736a5f36b52afa37e0f565b07de83e1b18cc4492014 -->

# zypper-declarative

A reconciling converger that brings a SUSE system to a **desired manifest**
inside a single snapshot transaction and records exactly what was applied.
Surfaced as a `zypper` subcommand and also invokable directly.

The manifest is the *declarable subset* of the SUSE Machinery system
description — `packages`, `repositories`, `services`, and `config_files` — in
the shared `ScopeWrapper` (`_attributes` / `_elements`) idiom. Its canonical
serialisation is JSON (`format_version` 1). YAML is an opt-in serialisation of
the identical data model.

- **Module:** `github.com/mge1512/zypper-declarative`
- **Spec-SHA256:** `714e75ff672557d2c7344736a5f36b52afa37e0f565b07de83e1b18cc4492014`
- **Language:** Go (single static binary, `CGO_ENABLED=0`)

## Installation

Distributed via the openSUSE Build Service (OBS). curl-based installation is
not supported.

```sh
# openSUSE / SLES / SL Micro
sudo zypper install zypper-declarative

# Debian / Ubuntu (where packaged)
sudo apt install zypper-declarative

# Fedora / RHEL (where packaged)
sudo dnf install zypper-declarative
```

To build from source (requires Go >= 1.21 and pandoc for the man page):

```sh
make build      # produces ./zypper-declarative (static)
make man        # produces ./zypper-declarative.1
make test       # runs the test suite
sudo make install
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
zypper declarative describe                     # bootstrap a manifest (JSON)
zypper declarative describe format=yaml > desired.yaml
zypper declarative describe root=/mnt out=/tmp/state.json
```

Equivalent direct form: `zypper-declarative <verb> [key=value ...]`.

### Verbs

| Verb       | Purpose                                                        |
|------------|----------------------------------------------------------------|
| `apply`    | Converge the system to the manifest in a snapshot transaction. |
| `diff`     | Dry run: print what `apply` would change. No modification.     |
| `verify`   | Check actual state equals the applied declaration.             |
| `status`   | Print the applied state and a one-line drift summary.          |
| `describe` | Emit the actual state of the declarable scopes (JSON or YAML). |

### Options (`key=value`, precede bare-word arguments)

POSIX `--flag` style is not a supported option syntax; options are `key=value`
pairs. (A leading `--` prefix is tolerated and stripped for convenience.)
Behaviour is never controlled via environment variables.

| Option | Meaning |
|--------|---------|
| `mode=auto\|external\|internal` | Transaction binding. Default `auto`. |
| `manifest-path=<path>` | Desired manifest. Default from CONFIG. |
| `manifest-format=json\|yaml` | Default input serialisation. Default `json`. |
| `format=json\|yaml` | Explicit input/`describe`-output format. |
| `state-path=<path>` | State dump as actual-state source for `verify`. |
| `root=<path>` | Root to `describe`. Default `/`. |
| `out=<path>` | `describe` output file. Default stdout. |
| `repo-lock=<repo>` | Fallback pinned repo when the manifest has none. |
| `content-store=<path>` | Base path for `content_ref` resolution. |
| `keep-list=<path>` | Allowlist of persistent-but-undeclared paths. |
| `signature-verification=on\|off` | Manifest signature check. Default `on`. |
| `keyring=<path>` | Keyring path when verification is on. |
| `activation-policy=reboot\|soft-reboot\|none` | Activation scheduling for `apply`. |
| `applied-root=<path>` | Generation root for the applied record. |

### Exit codes

| Code | Meaning |
|------|---------|
| `0`  | Success: converged or no-op, system matches declaration, or describe emitted. |
| `1`  | Logical failure: convergence failed and discarded; verify drift; invalid/unsafe/unverified manifest; state collection failed. |
| `2`  | Invocation error: bad arguments; unknown format; manifest unreadable; insufficient privilege; transaction mechanism unavailable; output path unwritable; malformed state dump. |

Diagnostics are written to stderr (one per line); normal output (summaries,
the diff plan, the status report, the describe document) goes to stdout.

## Design notes

- **Single live-state reader.** `describe-actual-state` is the only code that
  reads live system state; every verb obtains actual state through it or via a
  supplied dump in the same schema.
- **Two diffs.** The *intent diff* (`compute-intent-diff`) yields deletions
  without touching the filesystem; the *drift diff* (`compute-drift`) is a pure
  comparison of two manifests, used by `verify`, `status`, and the
  post-converge check in `apply`.
- **Abstract transaction binding.** `acquire-transaction-context` resolves
  `auto`/`external`/`internal`; the convergence code path is identical
  regardless of which mechanism opened the snapshot.
- **Applied record is always JSON**, stored under `/usr` within the generation
  so it is restored on rollback.

## License

GPL-2.0-or-later. See [LICENSE](LICENSE).
