# zypper-declarative

A reconciling, declarative converger for SL Micro and SLES in immutable mode, surfaced as
the `zypper declarative` subcommand. You author the desired state of a host as a
signed manifest; the tool reads the actual state, computes the difference, applies
it inside a snapshot transaction, and records what it did so the next run can also
remove what the manifest no longer declares.

It does this without Kubernetes, without a new package store, and without leaving
the SUSE substrate: btrfs snapshots are the generations, the rpmdb is the package
oracle, `/etc` is reconciled as the writable layer, and the description format is
the SUSE Machinery system description.

| | |
|---|---|
| Substrate | SL Micro 6.x, SLES 16.x (immutable and transactional, btrfs + snapper); also builds on SLES 15 SP7 |
| Surface | `zypper declarative <verb>` (or `zypper-declarative <verb>`) |
| Manifest | Machinery system description, declarable subset; JSON canonical, YAML optional |
| Method | Post-Coding Development (spec-driven, generated implementation) |
| Implementations | Three independent generations from one specification: C++, Go, Rust |
| License | GPL-2.0-or-later |
| Status | pre-1.0; specification mature through v0.6.9; see Status below |

## Status

This is early work and not production software. The specification is the mature
artifact; the implementation is generated from it, milestone by milestone (see
Roadmap), in three independent languages (C++, Go, Rust) from the one
specification, their agreement is the correctness check. The read-only and
onboarding foundation (`describe`, `init`, `diff`, `verify`, `status`, `version`,
and `help`, manifest loading, the live-state reader, and the applied-record
ledger) is verified. The convergence side of `apply` (the package, file, and unit
convergers acting inside a live snapshot) is milestone-gated and exercised on a
transactional target; do not point it at a host you care about yet.

Every build embeds the SHA256 of the specification it was generated from, so you
can always tell what a binary corresponds to:

```bash
zypper declarative version
# zypper-declarative 0.6.9 spec:aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3
```

## What it does

A declarative operating system, in the minimal sense this project uses, needs two
things: the state described as data, and an engine that converges the system to
that data idempotently, including removing what the data no longer asserts.
SL Micro already provides the hard parts (an immutable root, transactional
updates, and btrfs snapshots as switchable generations); what was missing was the
converger. That is what this tool is.

The central design point is that convergence needs two different comparisons, and
the filesystem only gives one of them cheaply:

- the intent diff compares the new manifest against the previously applied
  manifest, and is the only thing that yields deletions (nothing on disk changes
  when a line is simply dropped from the manifest);
- the drift diff compares the actual system against the declaration, and catches a
  change made out of band.

The deletion rule that falls out is safe by construction: the only files removed
are those the tool previously declared and no longer declares. Machine identity
and package-owned files are never candidates.

## How it is built: Post-Coding Development

This repository follows the Post-Coding Development paradigm (PCD, "Piccadilly").
The source of truth is the specification, `zypper-declarative.spec.md`. The
implementations are generated from that specification by a translator; humans do
not hand-edit the generated code. When behaviour is wrong, the fix goes into the
specification (or the language-specific decisions-hints that guide regeneration)
and the code is regenerated, never the other way round. The same specification is
generated independently into three languages (C++, Go, Rust); the three agreeing
on the same host is the project's correctness story, and a divergence between them
has repeatedly located a genuine specification gap.

That has two practical consequences for anyone reading this repository:

- the specification, not the code, is the document to read to understand what the
  tool does and why;
- contributions are proposed against the specification (see Contributing), and the
  generated implementations are build artifacts.

The repository carries these companion documents:

- `zypper-declarative.spec.md` - the PCD specification (the source of truth).
- `zypper-declarative-architecture.md` - the rationale and system context: the
  declarative ladder, the substrate, the two-diff model, the manifest format, the
  delivery paths, and the reproducibility stance.
- `zypper-declarative.{cpp,go,rs}.decisions.hints.md` - the per-language
  implementation decisions used to guide regeneration (disposable, not spec
  artifacts).

## The manifest

The manifest is a typed data model: the declarable subset of the Machinery system
description, the scopes a converger can act on. Each scope is a wrapper of the
form `{ _attributes, _elements }` with underscore_style field names, the Machinery
convention. Reusing that format means the desired state, the actual state read by
`describe`, and the recorded applied state are all the same kind of object, so
comparison is native rather than a translation exercise.

| Scope | Declares |
|---|---|
| `packages` | RPM packages (name; resolved to name-version-release-arch on apply) |
| `repositories` | pinned zypp repositories the package step resolves against |
| `services` | systemd unit enablement (enabled, disabled, masked) |
| `config_files` | declared files under `/etc`, by path, metadata, and content hash |

A scope that is absent from a manifest is left unmanaged; a scope that is present
but empty asserts that the scope should be empty, which removes what was
previously declared there. That absent-versus-empty distinction is the contract
for whether a domain is reconciled at all.

The canonical serialisation is JSON. YAML is an opt-in alternative for
environments that author OS state in YAML (for example a ZARF-centric, air-gapped
workflow); it is selected by the `format=` option or by file extension. JSON
output stays Machinery-compatible; YAML output does not. The applied record is
always written as canonical JSON, and a manifest's identity hash is computed over
the parsed data model, so the same intent in JSON or YAML is recognised as
identical.

A minimal desired manifest in JSON:

```json
{
  "meta": { "format_version": 1, "generator": "zypper-declarative", "desired_sha256": "" },
  "repositories": {
    "_attributes": { "repository_system": "zypp" },
    "_elements": [
      { "alias": "sl-micro-6.2-pinned", "name": "SL Micro 6.2 (pinned)",
        "url": "https://internal.example/obs/SLMicro:6.2:pinned/standard",
        "type": "rpm-md", "enabled": true, "gpgcheck": true,
        "autorefresh": false, "priority": 99 }
    ]
  },
  "packages": {
    "_attributes": { "package_system": "rpm" },
    "_elements": [ { "name": "nginx", "version": "", "release": "", "arch": "" } ]
  },
  "services": {
    "_attributes": { "init_system": "systemd" },
    "_elements": [ { "name": "nginx.service", "state": "enabled" } ]
  }
}
```

The same manifest in YAML:

```yaml
meta:
  format_version: 1
  generator: zypper-declarative
  desired_sha256: ""
repositories:
  _attributes:
    repository_system: zypp
  _elements:
    - alias: sl-micro-6.2-pinned
      name: SL Micro 6.2 (pinned)
      url: https://internal.example/obs/SLMicro:6.2:pinned/standard
      type: rpm-md
      enabled: true
      gpgcheck: true
      autorefresh: false
      priority: 99
packages:
  _attributes:
    package_system: rpm
  _elements:
    - { name: nginx, version: "", release: "", arch: "" }
services:
  _attributes:
    init_system: systemd
  _elements:
    - { name: nginx.service, state: enabled }
```

## Usage

The interface is bare-word verbs plus `key=value` options (no POSIX `--flag`
options; `--version`, `--help`, and `-h` are tolerated aliases of the `version`
and `help` commands).

```bash
# discover
zypper declarative help
zypper declarative version

# read the live system as a manifest (JSON to stdout, or YAML, or a file)
zypper declarative describe
zypper declarative describe format=yaml out=/tmp/host.yaml

# onboard this machine in one command: describe it, open a snapshot, adopt the
# current state as the managed baseline (the applied record), converge nothing,
# and write the manifest for you to edit
zypper declarative init out=/etc/zypper-declarative/desired.json

# see what apply would change, without changing anything (drift is computed
# against the desired manifest, so an unchanged machine shows none)
zypper declarative diff manifest-path=/etc/zypper-declarative/desired.json

# check whether the system still matches the recorded declaration
zypper declarative verify

# show the current declarative state
zypper declarative status

# converge the system to the manifest, inside a snapshot transaction
zypper declarative apply manifest-path=/etc/zypper-declarative/desired.json
```

The read-only verbs (`describe`, `diff`, `verify`, `status`) never modify the
system. `init` and `apply` open a snapshot: `init` adopts the current state as the
baseline and converges nothing (it is how you onboard a machine; afterwards `diff`
is clean and `verify` is meaningful), and `apply` is the only converging verb. Both
are idempotent: re-running `apply` against an unchanged manifest and an undrifted
system makes no changes and creates no new snapshot.

The live-reading verbs accept `on-unreadable=error|warn` (default `error`); pass
`warn` to skip a protected, root-only source with a diagnostic instead of aborting.
`init` always reads with `warn` so onboarding never fails on a protected file.

A convenient way to start is `init` (which both adopts and writes the manifest), or
to describe a hand-configured reference host and edit the result down:

```bash
zypper declarative describe > base.json    # or: format=yaml > base.yaml
```

### Transaction binding

The boundary to the snapshot transaction is deliberately abstract, so the same
tool works whether the transactional machinery is a separate tool
(`transactional-update`, SL Micro today) or merged into zypper potentially. The
`mode=auto|external|internal` option selects the binding; `auto` detects it. The
convergence behaviour is identical either way.

## Installation

The intended distribution is a signed RPM built in the Open Build Service
(https://build.opensuse.org), installed from a pinned, signed repository. There is
no `curl | sh` install path, by design. On a transactional host the package is
installed with `transactional-update pkg install` and takes effect after a reboot.

Currently there are test packages at:
https://build.opensuse.org/project/show/home:mge1512:declarative

The tool is generated in three languages from one specification; pick whichever
fits your environment (they are behaviourally equivalent). To build from source:

```bash
git clone https://github.com/mge1512/zypper-declarative
cd zypper-declarative

# C++ (links libzypp and libsnapper directly; the reference build):
cd code/cpp && make build        # produces ./zypper-declarative

# Go (static binary):
cd code/go && make build         # CGO_ENABLED=0 go build

# Rust:
cd code/rs && make build         # cargo build --release
```

Each implementation's `Makefile` also provides `make test`, `make man`, and
`make dist` (the release source tarball). The C++ build discovers its system
libraries (libzypp, libsnapper, jsoncpp, yaml-cpp, libcrypto) via pkg-config.

To expose it as a zypper subcommand, place the binary on `PATH` as
`zypper-declarative`, or install it into `/usr/lib/zypper/commands/`; zypper then
runs it as `zypper declarative`. It is also invokable directly as
`zypper-declarative <verb>`.

## Requirements

- SL Micro or SLES in Transactional mode 
- A btrfs root with snapper, and a transactional update mechanism.
- For `apply`, sufficient privilege to modify the system and open a transaction.
  The read-only verbs need only read access; a complete `describe` of all `/etc`
  wants root.

## Roadmap

The implementation is generated against the milestones declared in the
specification, scaffold first:

- 0.0.0 compilable skeleton (all verbs present, no logic)
- 0.1.0 read-only foundation: `status`, `describe`, the live-state reader, manifest loading
- 0.2.0 `diff` and the intent and drift comparisons
- 0.3.0 `verify`
- 0.4.0 `init` (onboarding: adopt the current state as the baseline, in a snapshot,
  converging nothing) and `apply` with file convergence and the applied-record ledger
- 0.5.0 package convergence
- 0.6.0 unit convergence and idempotent full converge

The read-only and onboarding foundation (through `verify` and `init`) is verified
across all three implementations. The converging side of `apply` is exercised on a
transactional target.

Deliberately deferred: secret material in the manifest, the kernel command line as
declared state, the hard-reset keep-list, NixOS-style version coexistence, and
resolving alternatives slaves to suppress default selections.

## Contributing

Because this is a PCD project, propose changes against the specification, not the
generated code. A good change is a precise edit to `zypper-declarative.spec.md`
(a behaviour, an invariant, an example) that passes `pcd-lint`; the implementation
is then regenerated from it. Bug reports are most useful when they describe the
observed behaviour against the specified behaviour, since that is usually a
specification gap to close rather than a patch to write.

## License

GPL-2.0-or-later. See `LICENSE`.

## Author

Matthias G. Eckermann (https://github.com/mge1512).
