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
| Substrate | SL Micro 6.2++, SLES 16 (immutable and transactional, btrfs + snapper) |
| Surface | `zypper declarative <verb>` (or `zypper-declarative <verb>`) |
| Manifest | Machinery system description, declarable subset; JSON canonical, YAML optional |
| Method | Post-Coding Development (spec-driven, generated implementation) |
| License | GPL-2.0-or-later |
| Status | pre-1.0, specification-complete through v0.5.1; see Status below |

## Status

This is early work and not production software. The specification is the mature
artifact; the implementation is generated from it milestone by milestone (see
Roadmap). The read-only foundation (the `describe`, `status`, `version`, and
`help` commands, manifest loading, and the live-state reader) is the current
focus. The convergence commands (`apply` and the package, file, and unit
convergers) are milestone-gated and not all complete. Do not point it at a host
you care about yet.

Every build embeds the SHA256 of the specification it was generated from, so you
can always tell what a binary corresponds to:

```bash
zypper declarative version
# zypper-declarative 0.5.1 spec:<sha256-of-spec>
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
The source of truth is the specification, `zypper-declarative.spec.md`. The Go
implementation is generated from that specification by a translator; humans do not
hand-edit the generated code. When behaviour is wrong, the fix goes into the
specification and the code is regenerated, never the other way round.

That has two practical consequences for anyone reading this repository:

- the specification, not the code, is the document to read to understand what the
  tool does and why;
- contributions are proposed against the specification (see Contributing), and the
  generated implementation is a build artifact.

The repository carries three companion documents:

- `zypper-declarative.spec.md` - the PCD specification (the source of truth).
- `zypper-declarative-architecture.md` - the rationale and system context: the
  declarative ladder, the substrate, the two-diff model, the manifest format, the
  delivery paths, and the reproducibility stance.
- `zypper-declarative.go.decisions.hints.md` - the language-specific
  implementation decisions used to guide regeneration (disposable, not a spec
  artifact).

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

# see what apply would change, without changing anything
zypper declarative diff manifest-path=/etc/zypper-declarative/desired.json

# check whether the system still matches the recorded declaration
zypper declarative verify

# show the current declarative state
zypper declarative status

# converge the system to the manifest, inside a snapshot transaction
zypper declarative apply manifest-path=/etc/zypper-declarative/desired.json
```

The read-only verbs (`diff`, `verify`, `status`, `describe`) never modify the
system. `apply` is the only privileged verb and is idempotent: re-running it
against an unchanged manifest and an undrifted system makes no changes and creates
no new snapshot.

A convenient way to start a manifest is to describe a hand-configured reference
host and edit the result down:

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
no `curl | sh` install path, by design.

To build from source (Go, static binary):

```bash
git clone https://github.com/mge1512/zypper-declarative
cd zypper-declarative
CGO_ENABLED=0 go build -o zypper-declarative .
```

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

- 0.0.0 compilable skeleton (all commands present, no logic)
- 0.1.0 read-only foundation: `status`, `describe`, the live-state reader, manifest loading
- 0.2.0 `diff` and the intent and drift comparisons
- 0.3.0 `verify`
- 0.4.0 `apply` with file convergence and the applied-record ledger
- 0.5.0 package convergence
- 0.6.0 unit convergence and idempotent full converge

Deliberately deferred: secret material in the manifest, the kernel command line as
declared state, and the hard-reset keep-list.

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
