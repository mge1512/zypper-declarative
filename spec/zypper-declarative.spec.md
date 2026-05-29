# zypper-declarative

## META
Deployment:  cli-tool
Version:     0.5.0
Spec-Schema: 0.4.0
Author:      Matthias G. Eckermann <pcd@mailbox.org>
License:     GPL-2.0-or-later
Verification: none
Safety-Level: QM
Module:      github.com/mge1512/zypper-declarative

---

## TYPES

The desired manifest, the applied record, the output of `describe`, and any
externally supplied state dump all share one data model: the **declarable
subset** of the SUSE Machinery system description, the scopes a converger can act
on (`packages`, `repositories`, `services`, `config_files`), using the
`ScopeWrapper` idiom and `underscore_style` field names. The tool reads the live
system into this same subset itself (see `describe-actual-state`), so no separate
collector program is required; an external producer of the same model (for
example sitar) is interchangeable but optional. Observational scopes a full dump
may carry (cpu, pci, dmi, processes, storage, network, and the rest) are ignored:
comparison is always on the declarable identity fields.

The manifest is this typed data model, not a byte format. Its canonical
serialisation is JSON (Machinery `format_version` 1). YAML is an opt-in
alternative serialisation of the identical data model (`ManifestFormat`), for
environments that author OS state in YAML (for example a ZARF-centric, air-gapped
workflow); it is selected by file extension or an explicit `format=` option (see
`load-desired-manifest` and CONFIG). YAML trades away Machinery and sitar
compatibility on the output side only: a YAML-capable reader still ingests a JSON
dump for `verify` (JSON is valid YAML 1.2), but `describe` output written as YAML
is not Machinery. The applied record is always canonical JSON regardless of input
format, so the on-disk ledger stays Machinery-readable. Manifest identity
(`desired_sha256`) is the hash of a canonical serialisation of the parsed data
model, not of the raw input bytes, so the same intent expressed in JSON or YAML
yields the same hash and idempotence holds across a format switch.

```
AbsolutePath := string where starts_with("/")

SemanticVersion := string where matches "^[0-9]+\.[0-9]+\.[0-9]+$"

Sha256 := string where matches "^[0-9a-f]{64}$"

Mode := string where matches "^[0-7]{3,4}$"

UnitName := string where (ends_with(".service") OR ends_with(".timer")
  OR ends_with(".socket") OR ends_with(".target") OR ends_with(".path")
  OR ends_with(".mount"))

// -----------------------------------------------------------------------
// Shared scope idiom (Machinery / sitar convention)
// -----------------------------------------------------------------------

ScopeWrapper<T> := {
  _attributes: object | null,   // scope-level metadata
  _elements:   []T              // the records in the scope
}

ManifestMeta := {
  format_version: integer,   // always 1 (Machinery JSON format)
  generator:      string,    // e.g. "zypper-declarative 0.5.0"
  created_at:     string,    // RFC3339, informational only, not compared
  desired_sha256: Sha256     // canonical-model hash of the desired manifest
                             // (hash of the canonical JSON serialisation of the
                             // parsed data model, format-independent); set in the
                             // applied record, "" elsewhere
}

// -----------------------------------------------------------------------
// Declarable scopes (the subset a converger acts on)
// -----------------------------------------------------------------------

PackageRecord := {
  name:    string where non-empty,
  version: string,   // "" in a desired manifest = newest from the pinned repo
  release: string,   // "" unless pinning an exact build
  arch:    string    // "" = native / any
}
// Machinery PackageRecord (identity subset). A desired package may carry name
// only. In the applied record and in describe output every PackageRecord has
// version, release, and arch populated; the fully-resolved packages scope IS the
// lock (the former NEVRA set), expressed in the shared format. A full dump may
// carry extension fields (vendor, checksum, size, ...) that the converger
// ignores; comparison is on the identity fields only.

PackagesScope := ScopeWrapper<PackageRecord>
  where _attributes.package_system = "rpm"

RepositoryRecord := {
  alias:       string where non-empty,
  name:        string,
  url:         string where non-empty,
  type:        string,    // e.g. "rpm-md"
  enabled:     bool,
  gpgcheck:    bool,
  autorefresh: bool,
  priority:    integer
}
// Machinery zypp repository record. Declares the pinned repositories the
// packages scope resolves against, in-band in the manifest.

RepositoriesScope := ScopeWrapper<RepositoryRecord>
  where _attributes.repository_system = "zypp"

ServiceRecord := {
  name:  UnitName,
  state: one_of("enabled" | "disabled" | "masked")
}
// Machinery service record, declarable states only. A full dump may also report
// "static" and a legacy_sysv field; both are observational and ignored.

ServicesScope := ScopeWrapper<ServiceRecord>
  where _attributes.init_system = "systemd"

ManagedFileRecord := {
  name:         AbsolutePath where starts_with("/etc/"),  // file path (Machinery: name)
  type:         one_of("file" | "link" | "dir"),
  mode:         Mode,
  user:         string where non-empty,
  group:        string where non-empty,
  sha256:       Sha256,    // content digest; comparison key for drift
  content_ref:  string,    // desired: how content is supplied at apply time;
                           // "" in describe output (actual state carries no source ref)
  package_name: string     // owning package; "" if unpackaged. Machinery field.
                           // Drives the files_extra rule: only unpackaged,
                           // undeclared /etc files count as extra.
}
// Aligned with the Machinery changed_config_files record (name, type, mode,
// user, group, package_name) and extended, as sitar extends Machinery, with
// sha256 and a content reference, because a DESIRED file needs its content,
// which an observational dump does not carry inline. v1 confines declared files
// to /etc.

ConfigFilesScope := ScopeWrapper<ManagedFileRecord>

Manifest := {
  meta:         ManifestMeta,
  packages:     PackagesScope,       // optional; absent = scope unmanaged
  repositories: RepositoriesScope,   // optional; absent = scope unmanaged
  services:     ServicesScope,       // optional; absent = scope unmanaged
  config_files: ConfigFilesScope     // optional; absent = scope unmanaged
}  where meta.format_version = 1 AND well-formed per the manifest JSON schema
// A scope ABSENT from the document means the converger makes no assertion about
// it (unmanaged). A scope PRESENT with empty _elements means the converger
// asserts the scope should be exactly empty, removing what was previously
// declared in that scope. The same Manifest shape is produced by describe (as
// the actual state) and consumed by apply/diff/verify (as the desired state).

AppliedRecord := Manifest
  where (every PackageRecord in packages._elements has non-empty version,
         release, and arch) AND meta.desired_sha256 != ""
// The applied record is a Manifest with the packages scope fully resolved (the
// lock) and the source manifest's hash recorded in meta. Stored inside the
// generation it describes; restored on rollback.

// -----------------------------------------------------------------------
// Transaction binding (deliberately deferred)
// -----------------------------------------------------------------------

TransactionMode := one_of("auto" | "external" | "internal")
// auto     = detect whether already running inside a snapshot transaction
// external = a separate mechanism opened it (e.g. transactional-update run ...)
// internal = open and commit it through the zypper-merged machinery (SLES 16.1)

TransactionContext := {
  mode:        TransactionMode,
  root:        AbsolutePath,   // mount point of the new snapshot's root tree
  opened_here: bool            // true if this tool opened it; false if external
}

// -----------------------------------------------------------------------
// Diff and drift
// -----------------------------------------------------------------------

Diff := {
  packages_install: []PackageRecord,
  packages_remove:  []PackageRecord,   // resolved records (name + version + release + arch)
  repos_set:        []RepositoryRecord,
  files_write:      []ManagedFileRecord,
  files_delete:     []AbsolutePath,
  units_change:     []ServiceRecord
}
// The intent diff: desired_new versus applied_old, computed scope by scope.

DriftReport := {
  files_modified:     []AbsolutePath,   // declared files whose actual content != declared
  files_extra:        []AbsolutePath,   // unpackaged /etc files not declared, not keep-listed
  units_divergent:    []ServiceRecord,
  packages_divergent: []PackageRecord
}
// The drift diff: actual versus declared. Empty == actual equals the declaration
// (modulo the keep-list).

Severity := Error | Warning

Diagnostic := {
  severity: Severity,
  domain:   string,   // packages | repositories | files | units | manifest | transaction | invocation
  message:  string
}

ExitCode := 0 | 1 | 2
// 0 = success: convergence complete (or no-op), system matches declaration, or
//     describe produced output
// 1 = logical failure: convergence failed and discarded; verify found drift;
//     manifest invalid or unverified; state collection failed
// 2 = invocation error: bad arguments; manifest unreadable; insufficient
//     privilege; transaction mechanism unavailable; output path unwritable

ManifestFormat := json | yaml
// Serialisation of the manifest data model. json is canonical and the default.
// yaml is an opt-in input and output format for environments that author OS
// state in YAML (e.g. a ZARF-centric workflow). The data model is identical in
// both; only the byte serialisation differs. yaml interchange is one-directional:
// a yaml-capable parser still ingests json (json is valid yaml 1.2), but describe
// output written as yaml is not Machinery and strict Machinery or sitar consumers
// will not read it. The yaml safety profile is defined in load-desired-manifest.
// Format selection for every read and write goes through resolve-format
// (explicit option, else operative file extension, else the manifest-format
// default), so input and output behave symmetrically.
```

Example desired manifest in the canonical JSON serialisation (illustration;
fenced and excluded from structural parsing):

```json
{
  "meta": {
    "format_version": 1,
    "generator": "zypper-declarative 0.5.0",
    "created_at": "2026-05-29T08:30:00Z",
    "desired_sha256": ""
  },
  "repositories": {
    "_attributes": { "repository_system": "zypp" },
    "_elements": [
      {
        "alias": "sl-micro-6.2-pinned",
        "name": "SL Micro 6.2 (pinned)",
        "url": "https://internal.example/obs/SLMicro:6.2:pinned/standard",
        "type": "rpm-md",
        "enabled": true,
        "gpgcheck": true,
        "autorefresh": false,
        "priority": 99
      }
    ]
  },
  "packages": {
    "_attributes": { "package_system": "rpm" },
    "_elements": [
      { "name": "nginx", "version": "", "release": "", "arch": "" }
    ]
  },
  "services": {
    "_attributes": { "init_system": "systemd" },
    "_elements": [
      { "name": "nginx.service", "state": "enabled" }
    ]
  },
  "config_files": {
    "_attributes": null,
    "_elements": [
      {
        "name": "/etc/nginx/nginx.conf",
        "type": "file",
        "mode": "0644",
        "user": "root",
        "group": "root",
        "sha256": "0000000000000000000000000000000000000000000000000000000000000000",
        "content_ref": "files/etc/nginx/nginx.conf",
        "package_name": ""
      }
    ]
  }
}
```

`describe` produces the same shape as the actual state: `content_ref` empty,
`package_name` populated from rpm, every `packages` record fully resolved. The
applied record is this shape with `meta.desired_sha256` set.

Note: Multiple BEHAVIOR and BEHAVIOR/INTERNAL sections are permitted. The five
BEHAVIOR sections are the CLI verbs (`apply`, `diff`, `verify`, `status`,
`describe`). The BEHAVIOR/INTERNAL sections describe the logic the verbs
orchestrate; they return errors to their caller rather than exiting, so exit-code
mapping lives only in the verbs. All sections share the TYPES, INVARIANTS, and
EXAMPLES of this specification.

---

## BEHAVIOR: apply
Constraint: required

The primary operation. Converges the system to the desired manifest inside a
single snapshot transaction, recording what was applied. Idempotent: a second
run against an unchanged manifest and an undrifted system makes no changes.

INPUTS:
```
manifest_path: AbsolutePath    // the desired manifest JSON (default from CONFIG)
mode:          TransactionMode  // default "auto" from CONFIG
```

OUTPUTS:
```
exit:        ExitCode
diagnostics: []Diagnostic   // to stderr
summary:     string          // to stdout
```

PRECONDITIONS:
- The process runs with privilege sufficient to modify the system.
- The transaction mechanism selected by `mode` is available.

STEPS:
1. Load the desired manifest via `load-desired-manifest`, which also yields its
   canonical-model `desired_sha256`. On read or unknown-format failure exit 2; on
   schema, unsafe-YAML, or signature failure exit 1.
2. Load the applied record of the current generation via `load-applied-record`.
   Absence is not failure: treat every scope as empty (first-ever apply).
3. Compute the intent diff via `compute-intent-diff` from desired and applied,
   scope by scope. A scope absent in desired yields no change for that scope.
4. If the intent diff is empty, obtain the actual state via
   `describe-actual-state` on "/" and compute drift via `compute-drift`. If drift
   is also empty, emit "nothing to do" to stdout and exit 0 without opening a
   transaction.
5. Acquire the transaction context via `acquire-transaction-context` for `mode`.
   On failure exit 2.
6. Apply the repositories scope (if managed) so the package step resolves against
   the declared pins, then converge packages via `converge-packages` inside the
   context; capture the resolved packages scope. On failure discard and exit 1.
7. Converge files via `converge-files`: write `files_write`, delete
   `files_delete`, excluding RPM-owned paths, the keep-list, and
   `/etc/etc.syncpoint`. On failure discard and exit 1.
8. Converge units via `converge-units` using offline enablement against the
   context root. On failure discard and exit 1.
9. Write the applied record via `write-applied-record` into the context's
   `/usr/lib/zypper-declarative/applied.json`, with the packages scope fully
   resolved, and stamp the snapshot's snapper userdata with `desired_sha256`.
   On failure discard and exit 1.
10. Verify the converged tree: obtain its actual state via `describe-actual-state`
    on the context root and run `compute-drift` against the new applied record.
    If the drift report is non-empty (excluding the keep-list), discard and
    exit 1.
11. Seal and activate: if `opened_here`, seal the snapshot read-only, mark it the
    default boot target, and schedule activation per the activation policy in
    CONFIG; if not `opened_here`, leave sealing and activation to the external
    mechanism. Emit the change summary to stdout and exit 0.

POSTCONDITIONS:
- On exit 0, either the system was already converged (no snapshot created), or a
  new sealed snapshot exists whose managed scopes equal the desired manifest, and
  whose `applied.json` records that manifest's `desired_sha256` and the resolved
  packages scope.
- On any non-zero exit, no new snapshot is left as the default boot target; the
  running system is unchanged.
- Re-running `apply` with the same manifest against the resulting system computes
  an empty intent diff and an empty drift report (idempotence).

ERRORS:
- manifest unreadable -> exit 2, domain=invocation
- manifest invalid or signature unverified -> exit 1, domain=manifest
- transaction mechanism unavailable -> exit 2, domain=transaction
- package convergence failed -> exit 1, domain=packages (transaction discarded)
- file convergence failed -> exit 1, domain=files (transaction discarded)
- unit convergence failed -> exit 1, domain=units (transaction discarded)
- post-converge verification found drift -> exit 1, domain=files (discarded)

---

## BEHAVIOR: diff
Constraint: required

Dry run. Computes and prints what `apply` would change, making no modification
to the system and opening no transaction.

INPUTS:
```
manifest_path: AbsolutePath
```

OUTPUTS:
```
exit: ExitCode
plan: string   // to stdout: the intent diff and the drift report
```

PRECONDITIONS:
- The desired manifest is readable.

STEPS:
1. Load the desired manifest via `load-desired-manifest`. On read or
   unknown-format failure exit 2; on schema, unsafe-YAML, or signature failure
   exit 1.
2. Load the applied record via `load-applied-record`; absence yields all scopes
   empty.
3. Compute the intent diff via `compute-intent-diff`.
4. Obtain the actual state via `describe-actual-state` on "/" and compute the
   drift report via `compute-drift`.
5. Print the combined plan (packages to install and remove, repositories to set,
   files to write and delete, units to change, and current drift) to stdout.
   Exit 0.

POSTCONDITIONS:
- No file, package, repository, or unit on the system is modified.
- No transaction is opened.
- Exit is 0 whenever the plan was computed, whether or not differences exist.

ERRORS:
- manifest unreadable -> exit 2, domain=invocation
- manifest invalid or signature unverified -> exit 1, domain=manifest

---

## BEHAVIOR: verify
Constraint: required

Checks whether the actual state equals the declaration recorded in the current
generation, modulo the keep-list. The post-condition assertion of the converge
loop, usable on a timer for drift detection. The actual state is read live by
default, or supplied as a state dump in the shared schema (for offline or remote
verification), since `describe` output and any compatible dump are the same
format.

INPUTS:
```
state_path: AbsolutePath          // optional state dump in the shared schema;
                                  // default = read live via describe-actual-state
format:     ManifestFormat | none // optional; resolved via resolve-format against state_path
```

OUTPUTS:
```
exit:        ExitCode
diagnostics: []Diagnostic   // to stderr: each drift item
summary:     string          // to stdout
```

PRECONDITIONS:
- An applied record exists for the current generation.

STEPS:
1. Load the applied record via `load-applied-record`. If none exists, emit
   "no declaration applied" to stderr and exit 2 (nothing to verify against).
2. Obtain the actual state. If `state_path` is given, resolve its format via
   `resolve-format(format, state_path)` (explicit option, else the dump's file
   extension, else the default), load it under that serialisation, and
   schema-validate it as a Manifest; a YAML dump is parsed under the same safe
   profile as a desired manifest. Otherwise obtain the actual state via
   `describe-actual-state` on "/" with `on_unreadable=error`. On a malformed dump
   exit 2.
3. Compute the drift report via `compute-drift` from the actual state and the
   applied record.
4. If the drift report is empty (excluding the keep-list), emit "system matches
   declaration" to stdout and exit 0. Otherwise emit one diagnostic per drift
   item to stderr and exit 1.

POSTCONDITIONS:
- The system is not modified.
- Exit 0 if and only if the actual state equals the applied record modulo the
  keep-list.

ERRORS:
- no applied record present -> exit 2, domain=invocation
- supplied state dump malformed -> exit 2, domain=invocation
- drift detected -> exit 1, domain=files or units or packages

---

## BEHAVIOR: status
Constraint: required

Prints the current declarative state: which manifest is applied, the generation,
and a one-line drift summary. Read-only and fast.

INPUTS:
```
(none beyond CONFIG)
```

OUTPUTS:
```
exit:   ExitCode
report: string   // to stdout
```

PRECONDITIONS:
- (none)

STEPS:
1. Reject any unrecognised argument: emit usage to stderr and exit 2.
2. Load the applied record via `load-applied-record`. If none exists, print
   "no declaration applied" and exit 0.
3. Print the applied manifest `desired_sha256`, the record `format_version`, the
   current snapshot/generation identifier, the `created_at`, and the count of
   resolved packages in the lock.
4. Obtain the actual state via `describe-actual-state` on "/", compute a drift
   summary via `compute-drift`, and print a single line ("clean" or "N drift
   item(s)"). Exit 0.

POSTCONDITIONS:
- The system is not modified.
- Exit is 0 whenever invocation is valid, including when no declaration is
  applied.

ERRORS:
- unrecognised argument -> exit 2, domain=invocation

---

## BEHAVIOR: describe
Constraint: required

Reads the actual state of the declarable scopes and emits it in the selected
serialisation (canonical JSON by default, YAML on request). This internalises the
system-description capability so no separate collector program is required. Three
uses: bootstrap a manifest from a running system (`describe > desired.json`),
capture a state dump for offline or remote `verify`, and audit. Read-only.

INPUTS:
```
root:          AbsolutePath          // root to describe; default "/"
out:           AbsolutePath          // output file; default = stdout
format:        ManifestFormat | none // optional; resolved via resolve-format against out
on_unreadable: one_of("error" | "warn")  // default "error" from CONFIG
```

OUTPUTS:
```
exit:     ExitCode
document: string   // Manifest of the actual state, in the resolved format, to out or stdout
```

PRECONDITIONS:
- (none beyond read access to the described root)

STEPS:
1. Reject any unrecognised argument or an unknown `format` value: emit usage to
   stderr and exit 2.
2. Obtain the actual state via `describe-actual-state` on `root` with
   `on_unreadable`. If a scope source is unreadable and `on_unreadable` is error,
   emit a diagnostic naming the source to stderr and exit 1. If warn, the
   unreadable scopes are omitted and a diagnostic is emitted to stderr for each,
   and processing continues.
3. Resolve the output format via `resolve-format(format, out)`: an explicit
   `format=` wins, else the `out` file extension decides (`.json` -> json,
   `.yaml`/`.yml` -> yaml), else the `manifest-format` default (stdout has no
   extension, so it uses the default unless `format=` is given).
4. Serialise the resulting Manifest in the resolved format: canonical JSON
   (Machinery `format_version` 1, ScopeWrapper idiom, underscore_style), or, when
   yaml, the same data model rendered as YAML. JSON output remains
   Machinery-compatible; YAML output does not.
5. Write it to `out` if given, else to stdout. On write failure exit 2. Exit 0.

POSTCONDITIONS:
- The system is not modified.
- The emitted document is a schema-valid Manifest (the declarable subset) in the
  resolved format, and is accepted unchanged by `load-desired-manifest` as a
  starting desired manifest.
- The output format follows `resolve-format`: when no `format=` is given, the
  `out` extension determines it, so `out=...yaml` yields YAML and `out=...json`
  yields JSON.
- No scope is emitted with empty `_elements` because its source was unreadable;
  under error an unreadable source fails the run, under warn it is omitted with a
  diagnostic.

ERRORS:
- unrecognised argument or unknown format value -> exit 2, domain=invocation
- unreadable scope source under on_unreadable=error -> exit 1, domain=packages or
  repositories or units or files
- output path unwritable -> exit 2, domain=invocation

---

## BEHAVIOR/INTERNAL: describe-actual-state
Constraint: required

The single live-state reader. Reads the actual state of the four declarable
scopes under a given root and returns a Manifest in the shared schema. Every
verb that needs actual state obtains it through this behaviour (or through a
supplied dump in the same format); no other code reads live system state.
Reads are file-and-database level (no network refresh, no daemon, no privileged
cache), so a normal user can read what is world-readable and the result is
deterministic. The `describe` verb passes `on_unreadable` through from its option;
every other caller (`apply`, `diff`, `status`, `verify` reading live state) passes
`on_unreadable=error`, because convergence and verification require complete
state.

INPUTS:
```
root:          AbsolutePath              // tree to read; "/" or a snapshot mount
on_unreadable: one_of("error" | "warn")  // how to treat a source that cannot be read
```

OUTPUTS:
```
manifest:    Manifest        // the actual state (declarable scopes)
diagnostics: []Diagnostic    // under warn: one per omitted unreadable scope
error:       Diagnostic      // under error: on the first unreadable source
```

STEPS:
1. packages: query the rpmdb under `root`; build a PackagesScope
   (`_attributes.package_system = "rpm"`) with every record's name, version,
   release, and arch populated. If the rpmdb cannot be read, treat per
   `on_unreadable` (step 6).
2. repositories: read the on-disk zypp repository configuration under `root`,
   namely the `.repo` files in `<root>/etc/zypp/repos.d/` (INI sections: alias,
   name, baseurl mapped to RepositoryRecord.url, type, enabled, gpgcheck,
   autorefresh, priority); build a RepositoriesScope
   (`_attributes.repository_system = "zypp"`). This reads declared repository
   files directly, which are world-readable in the normal case, rather than
   refreshing metadata or reading a privileged cache. If `repos.d` cannot be read,
   treat per `on_unreadable` (step 6).
3. services: query unit enablement under `root`; build a ServicesScope
   (`_attributes.init_system = "systemd"`) with each unit's state normalised to
   enabled, disabled, or masked. Purely-static units are omitted (not
   declarable). If enablement cannot be read, treat per `on_unreadable` (step 6).
4. config_files: enumerate `/etc` under `root`; for each file determine its
   owning package. Build a ManagedFileRecord with actual sha256, mode, user,
   group, type, and package_name for every file that is either changed from its
   package default or unpackaged. Skip files that are package-pristine, the
   keep-list, and `/etc/etc.syncpoint`. content_ref is "". If a file that must be
   inspected cannot be read (for example a root-only file under a non-root run),
   treat per `on_unreadable` (step 6).
5. Assemble the Manifest with meta.format_version = 1, generator, created_at, and
   empty desired_sha256. Omit any scope whose readable content is genuinely empty
   (so a bootstrapped manifest leaves that scope unmanaged rather than asserting
   deletion). Return it, with any warn diagnostics.
6. Unreadable-source handling: a scope or item whose source cannot be read is
   never represented as an empty scope. Under `on_unreadable=error`, return an
   error naming the unreadable source (the caller fails the run). Under
   `on_unreadable=warn`, omit the affected scope (or the affected items), append a
   diagnostic naming the source, and continue.

POSTCONDITIONS:
- The described root is not modified.
- The returned Manifest is schema-valid and contains only the four declarable
  scopes.
- config_files contains exactly the changed-from-package and unpackaged /etc
  files it could read (minus keep-list and syncpoint); package-pristine files are
  absent.
- No scope is emitted with empty `_elements` because its source was unreadable; an
  unreadable source is an error (strict) or an omission with a diagnostic (warn).
- A scope that is readable and genuinely empty is omitted, not emitted empty.

ERRORS:
- under on_unreadable=error, the first unreadable source (rpmdb, repos.d, unit
  enablement, or a required /etc file) -> packages, repositories, units, or files
  error returned to caller

---

## BEHAVIOR/INTERNAL: resolve-format
Constraint: required

The single authority for choosing a manifest serialisation. Every read that parses
a manifest and every write that serialises one resolves its format here, so input
and output behave symmetrically and the rule cannot drift between call sites.

INPUTS:
```
explicit: ManifestFormat | none   // the format= option, or none if not given
path:     AbsolutePath | none     // the operative file path, or none for stdin/stdout
```

OUTPUTS:
```
format: ManifestFormat
```

STEPS:
1. If `explicit` is given, return it (an explicit `format=` always wins).
2. Else if `path` is given and its file extension is recognised, return json for
   `.json` and yaml for `.yaml` or `.yml`.
3. Else (no explicit format, and no path or an unrecognised extension) return the
   `manifest-format` CONFIG default.

POSTCONDITIONS:
- An explicit format option always wins over the extension and the default.
- The file extension is consulted only when no explicit format is given.
- stdin or stdout (no path), and any unrecognised extension, fall back to the
  CONFIG default.
- The same resolution applies whether the path is an input (manifest-path,
  state-path) or an output (describe out).

---

## BEHAVIOR/INTERNAL: load-desired-manifest
Constraint: required

Reads and validates the desired manifest into the shared data model, selecting
the input serialisation via `resolve-format`, applying a safe YAML profile when
the input is YAML, and verifying the signature when signature verification is
enabled. Also computes the manifest's canonical-model identity hash.

INPUTS:
```
manifest_path: AbsolutePath
format:        ManifestFormat | none   // optional; passed to resolve-format
```

OUTPUTS:
```
manifest:       Manifest      // on success
desired_sha256: Sha256        // canonical-model hash of the parsed manifest
error:          Diagnostic    // on failure (returned to caller, no exit)
```

STEPS:
1. Read the file at `manifest_path`. On read failure return an invocation error.
2. Resolve the input format via `resolve-format(format, manifest_path)`. On an
   explicit but unknown format value return an invocation error.
3. Parse into the data model. For json, parse JSON. For yaml, parse under a safe
   profile: a non-code-executing loader only (no arbitrary or executable tags),
   alias expansion bounded or disabled (reject unbounded anchor/alias expansion),
   a single document only (reject multi-document streams), and explicit typing per
   the schema rather than YAML implicit typing (so values such as `NO` or a
   version like `1.10` are not coerced). A YAML input that requires any disabled
   feature returns a manifest error rather than being parsed.
4. Validate against the manifest schema: `meta.format_version` must be 1, and
   every present scope must conform to its ScopeWrapper record type. On violation
   return a manifest error naming the first violation.
5. If signature verification is enabled in CONFIG, verify the manifest's signature
   against the configured keyring. On failure return a manifest error.
6. Compute desired_sha256 as the SHA256 of the canonical JSON serialisation of the
   parsed data model (format-independent). Return the Manifest and desired_sha256.

POSTCONDITIONS:
- A returned Manifest is schema-valid and, when verification is enabled,
  signature-verified.
- desired_sha256 depends only on the parsed data model, so JSON and YAML
  expressions of the same manifest yield the same value.
- The input file is never modified.

ERRORS:
- file unreadable -> invocation error returned to caller
- unknown format value -> invocation error returned to caller
- YAML requires a disabled (unsafe) feature -> manifest error returned to caller
- schema violation -> manifest error returned to caller
- signature invalid -> manifest error returned to caller

---

## BEHAVIOR/INTERNAL: load-applied-record
Constraint: required

Reads the applied record of the current generation. Absence is a normal state
(first-ever apply) and is reported as an empty record, not an error.

INPUTS:
```
root: AbsolutePath   // generation root to read from; "/" for the running system
```

OUTPUTS:
```
record:  AppliedRecord   // empty (all scopes empty) if none present
present: bool
```

STEPS:
1. Resolve the path `<root>/usr/lib/zypper-declarative/applied.json`.
2. If absent, return a record with all scopes empty and present=false.
3. Read and parse it against the shared schema. On parse failure return a files
   error to the caller.
4. Return the record with present=true.

POSTCONDITIONS:
- An absent record yields present=false and an all-empty record, never an error.
- A present-but-corrupt record yields an error to the caller.

ERRORS:
- record present but unparseable -> files error returned to caller

---

## BEHAVIOR/INTERNAL: compute-intent-diff
Constraint: required

Computes the changes from the previously applied declaration to the desired
declaration, scope by scope. This is the comparison that yields deletions; it
does not consult the filesystem. A scope absent in the desired manifest produces
no change for that scope (unmanaged); a present scope is reconciled to exactly
its elements.

INPUTS:
```
desired: Manifest
applied: AppliedRecord
```

OUTPUTS:
```
diff: Diff
```

STEPS:
1. If desired.packages is present: packages_install := desired.packages._elements;
   packages_remove := the records in applied.packages._elements whose name is
   absent from desired.packages._elements. If desired.packages is absent, both
   lists are empty.
2. If desired.repositories is present: repos_set := desired.repositories._elements.
   Else empty.
3. If desired.config_files is present: files_write := desired.config_files._elements;
   files_delete := { e.name for e in applied.config_files._elements if e.name not
   in { d.name for d in desired.config_files._elements } }. This is
   `(declared_old - declared_new)` within the config_files scope. If absent, both
   are empty.
4. If desired.services is present: units_change := the desired.services._elements
   whose declared state differs from applied.services._elements (including
   services present in desired but absent in applied). Else empty.
5. Return the Diff.

POSTCONDITIONS:
- files_delete contains only paths the tool previously declared and no longer
  declares; it never contains a path absent from applied.config_files.
- A scope absent in desired contributes nothing to the diff.
- The filesystem is not read.

---

## BEHAVIOR/INTERNAL: compute-drift
Constraint: required

Compares an actual-state Manifest against a declaration, scope by scope on
identity fields, and reports divergence. Performs no I/O: the actual state is
produced beforehand by `describe-actual-state` (live system) or supplied as a
dump in the same schema. Used by `verify`, by `status`, and as the post-converge
check in `apply`.

INPUTS:
```
actual:    Manifest        // actual state (from describe-actual-state or a dump)
reference: AppliedRecord    // the declaration to compare against
```

OUTPUTS:
```
report: DriftReport
```

STEPS:
1. files_modified: for each ManagedFileRecord e in reference.config_files._elements,
   if actual.config_files._elements contains a record with name e.name whose
   sha256 differs from e.sha256, add e.name. A declared file absent from
   actual.config_files is treated as matching the declaration (it equals the
   package or declared default and so was not reported as changed).
2. files_extra: for each record a in actual.config_files._elements whose name is
   not in reference.config_files._elements, whose package_name is "" (unpackaged),
   and which is not keep-listed and not `/etc/etc.syncpoint`, add a.name. Changed
   but package-owned files that are undeclared are not "extra"; they are package
   managed (the keep-list is the escape hatch if they must be ignored entirely).
3. units_divergent: for each ServiceRecord u in reference.services._elements, if
   actual.services._elements reports a different state for u.name, add u.
4. packages_divergent: compare reference.packages._elements (identity fields)
   against actual.packages._elements; add any package present in one but not the
   other.
5. Return the DriftReport.

POSTCONDITIONS:
- Performs no filesystem, rpmdb, or process I/O; it is a pure comparison of two
  Manifest documents.
- `/etc/etc.syncpoint` and keep-listed paths never appear in files_extra; only
  unpackaged, undeclared /etc files do.
- A declared file absent from the actual scope is treated as matching, not as
  missing.
- An empty report means actual equals reference modulo the keep-list.

---

## BEHAVIOR/INTERNAL: acquire-transaction-context
Constraint: required

Resolves the transaction binding (deliberately deferred between an external
mechanism and the zypper-internal mechanism) and yields a context the
convergence domains operate within.

INPUTS:
```
mode: TransactionMode
```

OUTPUTS:
```
ctx:   TransactionContext   // on success
error: Diagnostic           // on failure (returned to caller)
```

STEPS:
1. If mode=auto, detect whether the process already runs inside a fresh snapshot
   transaction. If so resolve to external; otherwise resolve to internal.
2. If external: assert a writable new-generation root is present; set
   opened_here=false and root to that mount point. If no such root is present,
   return a transaction error (the caller must be invoked inside a transaction).
3. If internal: open a new snapshot transaction through the zypper-merged
   transactional machinery; set opened_here=true and root to the new mount point.
   On failure return a transaction error.
4. Return the TransactionContext.

POSTCONDITIONS:
- A returned context has a writable root distinct from the running system root.
- opened_here is true if and only if this tool opened the transaction.
- The same convergence behaviour applies regardless of which binding was
  resolved.

ERRORS:
- external mode but not running inside a transaction -> transaction error
- internal mode but transaction could not be opened -> transaction error

---

## BEHAVIOR/INTERNAL: converge-packages
Constraint: required

Applies the package portion of the intent diff inside the transaction context by
delegating to the package manager, resolving against the repositories declared in
the manifest (falling back to the CONFIG pin), and reports the resolved scope.

INPUTS:
```
ctx:  TransactionContext
diff: Diff
```

OUTPUTS:
```
resolved: PackagesScope   // fully populated installed set after convergence (the lock)
error:    Diagnostic      // on failure (returned to caller)
```

STEPS:
1. Within ctx.root, ensure the repositories in diff.repos_set are configured (or
   the CONFIG pin if repos_set is empty). On failure return a repositories error.
2. Within ctx.root, install diff.packages_install against the configured pinned
   repositories. On failure return a packages error.
3. Within ctx.root, remove diff.packages_remove. On failure return a packages
   error.
4. Query the rpmdb under ctx.root for the full installed set and return it as a
   PackagesScope with every record's name, version, release, and arch populated.

POSTCONDITIONS:
- The returned scope is the rpmdb-reported installed set, never inferred from
  file diffs, with all identity fields populated.
- Package retrieval occurs only against the configured, pinned repositories.

ERRORS:
- repository configuration failed -> repositories error returned to caller
- install failed -> packages error returned to caller
- remove failed -> packages error returned to caller

---

## BEHAVIOR/INTERNAL: converge-files
Constraint: required

Applies the file portion of the intent diff to `<ctx.root>/etc`, writing declared
files (resolving their content via content_ref) and deleting only files the
declaration dropped.

INPUTS:
```
ctx:  TransactionContext
diff: Diff
```

OUTPUTS:
```
error: Diagnostic   // on failure (returned to caller)
```

STEPS:
1. For each ManagedFileRecord e in diff.files_write: resolve content via
   e.content_ref, write `<ctx.root>/<e.name>` with that content and e.mode,
   e.user, e.group; verify the written content hashes to e.sha256. On failure
   return a files error.
2. For each path p in diff.files_delete: if p is RPM-owned, keep-listed, or equals
   `/etc/etc.syncpoint`, skip it; otherwise delete `<ctx.root>/<p>`. On failure
   return a files error.
3. Return success.

POSTCONDITIONS:
- Only files in diff.files_write are created or overwritten, and each matches its
  declared sha256.
- Only files in diff.files_delete that are not excluded are removed.
- `/etc/etc.syncpoint`, RPM-owned paths, and keep-listed paths are never deleted.

ERRORS:
- content resolution or write failed -> files error returned to caller
- written content hash mismatch -> files error returned to caller
- delete failed -> files error returned to caller

---

## BEHAVIOR/INTERNAL: converge-units
Constraint: required

Applies the unit portion of the intent diff using offline enablement against the
transaction context root, which avoids the systemd preset being evaluated only on
first boot.

INPUTS:
```
ctx:  TransactionContext
diff: Diff
```

OUTPUTS:
```
error: Diagnostic   // on failure (returned to caller)
```

STEPS:
1. For each ServiceRecord u in diff.units_change: apply the declared state
   (enabled, disabled, or masked) offline against ctx.root. On failure return a
   units error.
2. Return success.

POSTCONDITIONS:
- Unit enablement is written into ctx.root, not the running system.
- The result does not depend on first-boot preset evaluation.

ERRORS:
- offline enablement failed -> units error returned to caller

---

## BEHAVIOR/INTERNAL: write-applied-record
Constraint: required

Writes the applied record into the transaction context so that it travels with
the generation and is restored automatically on rollback.

INPUTS:
```
ctx:            TransactionContext
desired:        Manifest
desired_sha256: Sha256
resolved:       PackagesScope   // the lock from converge-packages
```

OUTPUTS:
```
error: Diagnostic   // on failure (returned to caller)
```

STEPS:
1. Construct an AppliedRecord: copy desired's repositories, services, and
   config_files scopes; set the packages scope to `resolved`; set
   meta.desired_sha256 to desired_sha256, meta.created_at to now, and
   meta.format_version to 1.
2. Serialise it as canonical JSON in the shared format, regardless of the desired
   manifest's input serialisation (the ledger is always JSON so it stays
   Machinery-readable), and write to
   `<ctx.root>/usr/lib/zypper-declarative/applied.json`. On failure return a
   files error.
3. Stamp the snapshot's snapper userdata with the key/value
   `manifest=<desired_sha256>`. On failure return a files error.

POSTCONDITIONS:
- The record is inside the generation it describes, so a rollback to that
  generation restores the matching record.
- The record validates as an AppliedRecord (packages fully resolved,
  desired_sha256 set).
- The snapper userdata of the generation carries the desired_sha256 as an index.

ERRORS:
- record write failed -> files error returned to caller
- userdata stamp failed -> files error returned to caller

---

## INTERFACES

The tool integrates with these external systems. Each is an abstract dependency;
the binding is resolved at build or run time.

- **Package manager (libzypp / zypper).** Provides repository configuration,
  package install, remove, and rpmdb query within a context root, against the
  declared pinned repositories. All package retrieval is delegated here; the tool
  performs no direct network fetch.
- **Snapshot and filesystem (btrfs / snapper).** Provides the snapshot
  transaction, the writable `/etc` subvolume, the `etc.syncpoint` reference, and
  snapshot userdata. `/etc` is a nested subvolume with its own snapshot lineage;
  drift comparison of `/etc` is over the actual-state Manifest, which
  `describe-actual-state` produces from the `/etc` subvolume content.
- **Init system (systemd).** Provides unit enablement query (for
  `describe-actual-state`) and offline enablement against a context root (for
  `converge-units`), sidestepping first-boot preset evaluation.
- **Transaction mechanism.** Either an external opener
  (`transactional-update run ...`) or the zypper-internal transactional machinery
  (SLES 16.1). The choice is the `TransactionMode` and is resolved by
  `acquire-transaction-context`. This specification does not commit to either
  binding.
- **External state producer (optional, interchangeable).** Because actual state
  is the shared Machinery data model, any external producer of that format (for
  example sitar) may supply a dump to `verify` via `state-path`. This is optional:
  `describe-actual-state` provides the same capability internally, so no separate
  collector program is required. If such a tool is present, its output and
  `describe` output are interchangeable. Even when desired manifests are authored
  in YAML, a Machinery JSON dump is still accepted as the actual-state input.

---

## CONFIG

All knobs are surfaced via key=value arguments or preset files (systemd-style
layering). Control via environment variables is forbidden.

- `transaction-mode` = auto | external | internal. Default auto.
- `manifest-path` = path to the desired manifest. Default a fixed staging path
  supplied by the delivery layer.
- `manifest-format` = json | yaml. Default json. The fallback serialisation used
  by `resolve-format` when neither an explicit `format=` option nor a recognised
  file extension determines it (for example stdin or stdout). The applied record
  is canonical JSON irrespective of this knob.
- `on-unreadable` = error | warn. Default error. How `describe` (and the
  `describe-actual-state` reader) treats a scope source it cannot read: error
  fails the run naming the source; warn omits the affected scope, emits a
  diagnostic, and continues. A source that cannot be read is never represented as
  an empty scope. Internal callers (apply, diff, status, verify) always use error.
- `repo-lock` = fallback pinned repository or channel used only when the manifest
  declares no `repositories` scope. The primary, declarative source of the pin is
  the manifest's repositories scope.
- `content-store` = base path against which ManagedFileRecord `content_ref`
  values are resolved at apply time.
- `keep-list` = path to the allowlist of persistent-but-undeclared paths that
  `describe-actual-state`, `compute-drift`, and `converge-files` must never
  report or delete (machine-id, SSH host keys, the systemd random seed). Used by
  the hard-reset path; harmless for incremental convergence.
- `signature-verification` = on | off, plus the keyring path when on. Default on.
- `activation-policy` = reboot | soft-reboot | none. How `apply` schedules
  activation of a freshly sealed snapshot.
- `applied-root` = generation root from which `load-applied-record` reads the
  applied record. Default "/". Set it to a mounted snapshot to inspect that
  generation's record.

Reserved for a later Version, explicitly out of scope for v1: secret material in
the declaration (resolved at apply time from an external store, never stored in
the manifest), and the additional Machinery scopes (kernel params, sysctl, users,
groups) as declarable domains.

---

## DEPENDENCIES

Build and translation dependencies. Direct dependencies only; the language's
resolver fills in the transitive set.

- Milestone hints: `cli-tool.go.milestones.hints.md` when the resolved language
  is Go (the template default). Equivalent milestone hints apply if a preset
  selects Rust or C++. This specification is language-neutral; nothing in it
  assumes a particular implementation language.
- Serialisation: the canonical manifest and the applied record use the SUSE
  Machinery system-description format in JSON (`format_version` 1); this is the
  interoperability contract, and an external Machinery-format producer (e.g.
  sitar) is optional and interchangeable, not required. YAML is an opt-in input
  and output serialisation of the same data model.
- YAML parsing (only when YAML is enabled): a YAML library is a direct dependency.
  It must be driven under a safe profile, stated here as a constraint rather than
  a named library so the spec stays language-neutral: a non-code-executing loader
  only (no arbitrary or executable tags), bounded or disabled anchor/alias
  expansion (defends against alias-expansion denial of service), single-document
  streams only, and explicit typing per the schema rather than YAML implicit
  typing. If a language-specific hints file pins the YAML library version, use it;
  otherwise flag it in `TRANSLATION_REPORT.md` for manual version verification.
- Bindings to libzypp, snapper/btrfs, and systemd require verified version
  strings. If a language-specific hints file for these bindings is present, use
  the versions it provides; if absent, flag each binding in
  `TRANSLATION_REPORT.md` as requiring manual version verification before build.

---

## PRECONDITIONS

- The tool is invoked as `zypper declarative <verb>` (dispatched subcommand) or
  as `zypper-declarative <verb>` directly.
- For `apply`, the process has privilege sufficient to modify the system and the
  selected transaction mechanism is available.
- The desired manifest, when required by the verb, is readable, schema-valid, and
  (when verification is enabled) signed.

---

## POSTCONDITIONS

- `apply` leaves the system either unchanged (on any failure, or when already
  converged) or advanced to exactly one new sealed generation equal to the
  declaration in its managed scopes.
- `diff`, `verify`, `status`, and `describe` never modify the system.
- After a successful `apply`, `verify` against the resulting generation exits 0.
- `describe` output is accepted unchanged by `load-desired-manifest`.
- Every produced artifact embeds the SHA256 of this specification.

---

## INVARIANTS

- [observable] `diff`, `verify`, `status`, and `describe` make no modification to
  packages, repositories, files, or units, and open no transaction.
- [observable] `apply` is idempotent: a second run against an unchanged manifest
  and an undrifted system computes an empty intent diff and empty drift, and exits
  0 without creating a new generation.
- [observable] Within the config_files scope, the only files `apply` deletes are
  those in `(declared_old - declared_new)`; a path never previously declared is
  never deleted.
- [observable] A scope absent from the desired manifest is left unmanaged: the
  converger makes no change to it; a present-but-empty scope is reconciled to
  empty.
- [observable] `/etc/etc.syncpoint`, RPM-owned paths, and keep-listed paths are
  never written to or deleted by `converge-files`.
- [observable] In `compute-drift`, files_extra contains only unpackaged,
  undeclared /etc files; package-owned files, keep-listed paths, and
  `/etc/etc.syncpoint` never appear there.
- [observable] On any non-zero exit of `apply`, no new snapshot is left as the
  default boot target.
- [observable] The applied record of a generation is restored together with that
  generation on rollback, because it resides under `/usr` within the snapshot.
- [observable] `verify` exits 0 if and only if the actual state equals the applied
  record modulo the keep-list.
- [observable] The resolved package lock is the rpmdb-reported installed set, not
  a set inferred from file differences, and is recorded as a fully populated
  packages scope.
- [observable] Package retrieval occurs only against the declared pinned
  repositories; the tool performs no direct network fetch of its own.
- [observable] `describe` output is a schema-valid Manifest in the declarable
  subset, in the requested serialisation (json or yaml), and is accepted unchanged
  as a desired manifest.
- [observable] The applied record is serialised as canonical JSON regardless of
  the desired manifest's input serialisation.
- [observable] `desired_sha256` is the hash of the canonical serialisation of the
  parsed data model; the same manifest expressed in JSON or YAML yields the same
  value, so idempotence holds across a format switch.
- [observable] A YAML manifest requiring a disabled (unsafe) loader feature (an
  executable or arbitrary tag, unbounded alias expansion, or multiple documents)
  is rejected with a manifest error rather than parsed.
- [observable] Every manifest read and write resolves its serialisation through
  `resolve-format`: an explicit `format=` wins, else the operative file extension
  (`.json` / `.yaml` / `.yml`) decides, else the `manifest-format` default; so
  `describe out=...yaml` writes YAML and `out=...json` writes JSON.
- [observable] `describe-actual-state` never emits a scope with empty `_elements`
  because its source could not be read; an unreadable source is an error (strict)
  or an omission with a diagnostic (warn).
- [observable] A genuinely empty actual scope is omitted from `describe` output,
  so a bootstrapped manifest leaves that scope unmanaged rather than asserting
  deletion.
- [observable] Repositories actual state is read from the on-disk zypp
  configuration (`/etc/zypp/repos.d`), not from a network refresh or a privileged
  cache.
- [observable] Drift and diff comparison uses only the declarable identity fields
  of each scope; observational extension fields in a supplied dump are ignored.
- [implementation] The intent diff is computed without reading the filesystem.
- [implementation] `compute-drift` performs no I/O; it compares two Manifest
  documents.
- [implementation] There is exactly one live-state reader, `describe-actual-state`;
  `diff`, `verify`, `status`, `describe`, and `apply` obtain actual state through
  it or through a supplied dump in the same format.
- [implementation] The same convergence code path runs regardless of whether the
  transaction binding resolved to external or internal.
- [observable] Every generated artifact embeds the SHA256 of the spec file it was
  produced from.

---

## EXAMPLES

### EXAMPLE: apply_no_op_when_converged
GIVEN:
  the desired manifest equals the current generation's applied record in all
    managed scopes
  the live system has no drift
  invocation: zypper declarative apply
WHEN:
  apply runs
THEN:
  no transaction is opened
  no new snapshot is created
  stdout contains "nothing to do"
  exit_code = 0

### EXAMPLE: apply_writes_and_deletes_etc_file
GIVEN:
  the applied record config_files scope declares /etc/foo.conf and /etc/bar.conf
  the desired manifest config_files scope declares /etc/foo.conf (changed sha256)
    and drops /etc/bar.conf
  transaction-mode = internal and a transaction can be opened
  invocation: zypper declarative apply
WHEN:
  apply runs
THEN:
  a new snapshot is created
  /etc/foo.conf in the new snapshot has the declared content matching its sha256
  /etc/bar.conf is absent in the new snapshot
  the new applied.json records the desired manifest's desired_sha256
  exit_code = 0

### EXAMPLE: apply_absent_scope_unmanaged
GIVEN:
  the desired manifest contains packages and services scopes but no config_files
    scope
  the live /etc contains admin-written files
  transaction-mode = internal and a transaction can be opened
  invocation: zypper declarative apply
WHEN:
  apply runs
THEN:
  no file under /etc is written or deleted by the file domain
  packages and services are converged as declared
  exit_code = 0

### EXAMPLE: apply_manifest_invalid
GIVEN:
  the desired manifest has meta.format_version = 2
  invocation: zypper declarative apply
WHEN:
  apply runs
THEN:
  stderr contains one diagnostic with domain=manifest
  no transaction is opened
  exit_code = 1

### EXAMPLE: apply_manifest_unreadable
GIVEN:
  manifest-path points at a file that does not exist
  invocation: zypper declarative apply manifest-path=/nonexistent.json
WHEN:
  apply runs
THEN:
  stderr contains one diagnostic with domain=invocation
  exit_code = 2

### EXAMPLE: apply_transaction_unavailable
GIVEN:
  transaction-mode = external
  the process is not running inside a snapshot transaction
  invocation: zypper declarative apply mode=external
WHEN:
  apply runs
THEN:
  stderr contains one diagnostic with domain=transaction
  no modification is made to the system
  exit_code = 2

### EXAMPLE: apply_package_failure_rolls_back
GIVEN:
  the desired packages scope requests a package absent from the declared pinned
    repositories
  transaction-mode = internal and a transaction can be opened
  invocation: zypper declarative apply
WHEN:
  apply runs
THEN:
  the transaction is discarded
  no new snapshot is left as the default boot target
  stderr contains one diagnostic with domain=packages
  exit_code = 1

### EXAMPLE: diff_prints_plan
GIVEN:
  the desired manifest adds package nginx and drops /etc/bar.conf relative to the
    applied record
  invocation: zypper declarative diff
WHEN:
  diff runs
THEN:
  stdout lists nginx under packages to install
  stdout lists /etc/bar.conf under files to delete
  no transaction is opened and the system is unmodified
  exit_code = 0

### EXAMPLE: diff_manifest_unreadable
GIVEN:
  manifest-path points at an unreadable file
  invocation: zypper declarative diff manifest-path=/nonexistent.json
WHEN:
  diff runs
THEN:
  stderr contains one diagnostic with domain=invocation
  exit_code = 2

### EXAMPLE: describe_emits_manifest
GIVEN:
  the live system has nginx installed and one changed file /etc/nginx/nginx.conf
  invocation: zypper declarative describe
WHEN:
  describe runs
THEN:
  stdout is a JSON document with meta.format_version = 1
  the packages scope contains a fully resolved nginx record (version, release, arch)
  the config_files scope contains /etc/nginx/nginx.conf with its actual sha256
  exit_code = 0

### EXAMPLE: describe_output_unwritable
GIVEN:
  invocation: zypper declarative describe out=/readonly/dir/state.json
  the output path is not writable
WHEN:
  describe runs
THEN:
  stderr contains one diagnostic with domain=invocation
  exit_code = 2

### EXAMPLE: describe_bootstraps_desired_manifest
GIVEN:
  a freshly described document captured via: zypper declarative describe > desired.json
  invocation: zypper declarative diff manifest-path=desired.json
WHEN:
  diff runs against the same unchanged system
THEN:
  the plan shows no packages to install or remove and no files to write or delete
  exit_code = 0

### EXAMPLE: verify_clean
GIVEN:
  the live system equals the current generation's applied record
  invocation: zypper declarative verify
WHEN:
  verify runs
THEN:
  stdout contains "system matches declaration"
  exit_code = 0

### EXAMPLE: verify_against_external_state_dump
GIVEN:
  a state dump in the shared schema (from describe on another host, or from an
    external Machinery-format producer) is at /tmp/state.json
  the dump diverges from the applied record in one declared service state
  invocation: zypper declarative verify state-path=/tmp/state.json
WHEN:
  verify runs
THEN:
  stderr contains a diagnostic with domain=units naming the divergent service
  exit_code = 1

### EXAMPLE: verify_malformed_state_dump
GIVEN:
  the file at state-path is not a valid shared-schema Manifest
  invocation: zypper declarative verify state-path=/tmp/broken.json
WHEN:
  verify runs
THEN:
  stderr contains one diagnostic with domain=invocation
  exit_code = 2

### EXAMPLE: verify_detects_drift
GIVEN:
  a declared file /etc/foo.conf has been edited on the live system
  invocation: zypper declarative verify
WHEN:
  verify runs
THEN:
  stderr contains a diagnostic naming /etc/foo.conf with domain=files
  exit_code = 1

### EXAMPLE: verify_no_applied_record
GIVEN:
  no applied record exists for the current generation
  invocation: zypper declarative verify
WHEN:
  verify runs
THEN:
  stderr contains "no declaration applied" with domain=invocation
  exit_code = 2

### EXAMPLE: status_reports_generation
GIVEN:
  an applied record exists with a known desired_sha256 and a resolved packages lock
  invocation: zypper declarative status
WHEN:
  status runs
THEN:
  stdout contains the desired_sha256 and the snapshot identifier
  stdout contains the resolved package count and a single drift-summary line
  exit_code = 0

### EXAMPLE: status_no_declaration
GIVEN:
  no applied record exists
  invocation: zypper declarative status
WHEN:
  status runs
THEN:
  stdout contains "no declaration applied"
  exit_code = 0

### EXAMPLE: status_unknown_argument
GIVEN:
  invocation: zypper declarative status --frobnicate
WHEN:
  status runs
THEN:
  stderr contains usage with domain=invocation
  exit_code = 2

### EXAMPLE: intent_diff_yields_deletion
GIVEN:
  applied record config_files._elements names = { /etc/foo.conf, /etc/bar.conf }
  desired manifest config_files._elements names = { /etc/foo.conf }
WHEN:
  compute-intent-diff runs
THEN:
  files_delete = { /etc/bar.conf }
  files_write contains the /etc/foo.conf record
  no path outside the applied config_files scope appears in files_delete

### EXAMPLE: drift_ignores_unmanaged_packaged_file
GIVEN:
  actual state contains a changed but package-owned /etc file that the reference
    does not declare (package_name non-empty)
  invocation path: compute-drift over actual and reference
WHEN:
  compute-drift runs
THEN:
  the file does not appear in files_extra
  it appears in files_modified only if it is also a declared file with a differing sha256

### EXAMPLE: describe_actual_state_omits_pristine
GIVEN:
  /etc contains a package-pristine file and one changed file
  invocation path: describe-actual-state on "/"
WHEN:
  describe-actual-state runs
THEN:
  the config_files scope contains the changed file with package_name set
  the package-pristine file is absent from the config_files scope

### EXAMPLE: lock_is_fully_resolved_packages_scope
GIVEN:
  the desired packages scope contains { name: "nginx" } with empty version
  the pinned repository provides nginx at a specific version-release-arch
WHEN:
  converge-packages runs and write-applied-record stores the result
THEN:
  the applied record packages._elements entry for nginx has non-empty version,
    release, and arch
  the applied record validates as an AppliedRecord

### EXAMPLE: yaml_manifest_accepted
GIVEN:
  manifest-path points at desired.yaml, a YAML serialisation of a valid manifest
  invocation: zypper declarative diff manifest-path=desired.yaml
WHEN:
  diff runs
THEN:
  the YAML is parsed under the safe profile and validated against the schema
  the plan is computed identically to the equivalent JSON manifest
  exit_code = 0

### EXAMPLE: describe_format_yaml
GIVEN:
  the live system has nginx installed
  invocation: zypper declarative describe format=yaml
WHEN:
  describe runs
THEN:
  stdout is a YAML document representing the same data model as the JSON output
  the document is not Machinery-compatible
  exit_code = 0

### EXAMPLE: yaml_format_identity_stable
GIVEN:
  desired.json and desired.yaml express the same manifest in different formats
WHEN:
  load-desired-manifest parses each
THEN:
  both yield the same desired_sha256 (the canonical-model hash)
  after applying one, applying the other computes an empty intent diff (idempotent)

### EXAMPLE: yaml_unsafe_rejected
GIVEN:
  a YAML manifest uses an executable or arbitrary tag, or unbounded alias expansion
  invocation: zypper declarative apply manifest-path=evil.yaml
WHEN:
  apply runs
THEN:
  load-desired-manifest rejects it with a manifest error
  no transaction is opened
  exit_code = 1

### EXAMPLE: describe_unknown_format
GIVEN:
  invocation: zypper declarative describe format=toml
WHEN:
  describe runs
THEN:
  stderr contains usage with domain=invocation
  exit_code = 2

### EXAMPLE: bare_invocation_shows_help
GIVEN:
  invocation: zypper declarative          (no verb)
WHEN:
  the tool is invoked
THEN:
  usage is printed to stdout
  exit_code = 0

### EXAMPLE: unknown_verb_rejected
GIVEN:
  invocation: zypper declarative frobnicate
WHEN:
  the tool is invoked
THEN:
  usage is printed to stderr with domain=invocation
  exit_code = 2

### EXAMPLE: describe_out_extension_yaml
GIVEN:
  no format option is given
  invocation: zypper declarative describe out=/tmp/state.yaml
WHEN:
  describe runs
THEN:
  resolve-format selects yaml from the .yaml extension
  /tmp/state.yaml contains a YAML document
  exit_code = 0

### EXAMPLE: describe_out_extension_json
GIVEN:
  no format option is given
  invocation: zypper declarative describe out=/tmp/state.json
WHEN:
  describe runs
THEN:
  resolve-format selects json from the .json extension
  /tmp/state.json contains a JSON document
  exit_code = 0

### EXAMPLE: describe_format_overrides_extension
GIVEN:
  invocation: zypper declarative describe format=json out=/tmp/state.yaml
WHEN:
  describe runs
THEN:
  resolve-format returns json because the explicit option wins over the extension
  /tmp/state.yaml contains a JSON document
  exit_code = 0

### EXAMPLE: verify_state_path_extension_yaml
GIVEN:
  a YAML state dump is at /tmp/state.yaml and no format option is given
  the dump matches the applied record
  invocation: zypper declarative verify state-path=/tmp/state.yaml
WHEN:
  verify runs
THEN:
  resolve-format selects yaml from the .yaml extension and the dump is parsed
  stdout contains "system matches declaration"
  exit_code = 0

### EXAMPLE: describe_repositories_from_reposd
GIVEN:
  /etc/zypp/repos.d contains two readable .repo files
  invocation: zypper declarative describe
WHEN:
  describe runs
THEN:
  the repositories scope _elements contains two RepositoryRecord entries
  the scope is not empty
  exit_code = 0

### EXAMPLE: describe_unreadable_scope_strict
GIVEN:
  /etc/zypp/repos.d cannot be read by the current user
  on-unreadable defaults to error
  invocation: zypper declarative describe
WHEN:
  describe runs
THEN:
  stderr contains a diagnostic naming the unreadable source with domain=repositories
  no document with an empty repositories scope is emitted
  exit_code = 1

### EXAMPLE: describe_unreadable_scope_warn
GIVEN:
  /etc/zypp/repos.d cannot be read by the current user
  invocation: zypper declarative describe on-unreadable=warn
WHEN:
  describe runs
THEN:
  the repositories scope is omitted from the output, not emitted empty
  stderr contains a diagnostic naming the unreadable source
  exit_code = 0

### EXAMPLE: describe_omits_genuinely_empty_scope
GIVEN:
  /etc/zypp/repos.d is readable and contains no .repo files
  invocation: zypper declarative describe
WHEN:
  describe runs
THEN:
  the output omits the repositories scope rather than emitting empty _elements
  a manifest bootstrapped from this output leaves repositories unmanaged
  exit_code = 0

### EXAMPLE: idempotent_second_apply
GIVEN:
  apply has just succeeded for a manifest M
  the system has not drifted
  invocation: zypper declarative apply (same manifest M)
WHEN:
  apply runs again
THEN:
  the intent diff is empty and drift is empty
  no new snapshot is created
  exit_code = 0

---

## DEPLOYMENT

Runtime: command-line tool, single static binary, no runtime dependencies of its
own beyond the system package manager, snapshot tooling, and init system it
drives. Surfaced as a zypper subcommand via the zypper subcommand mechanism (an
executable in `/usr/lib/zypper/commands`), and also invokable directly.

This specification is language-neutral. The resolved implementation language (Go
by default per the cli-tool template, Rust or C++ via preset) does not affect any
behaviour, type, interface, or exit-code defined here. LANGUAGE is never declared
in META; the template resolves it.

Invocation:
```
zypper declarative apply
zypper declarative apply mode=external
zypper declarative apply manifest-path=/var/lib/zypper-declarative/desired.json
zypper declarative diff
zypper declarative verify
zypper declarative verify state-path=/tmp/state.json
zypper declarative status
zypper declarative describe
zypper declarative describe > desired.json          # bootstrap a manifest (JSON)
zypper declarative describe format=yaml > desired.yaml
zypper declarative describe root=/mnt out=/tmp/state.json
zypper declarative apply manifest-path=/etc/zypper-declarative/desired.yaml
```
Equivalent direct form: `zypper-declarative <verb> [key=value ...]`.

Key=value options (precede any bare-word argument):
```
mode=auto|external|internal       transaction binding; default auto
manifest-path=<path>              desired manifest; default from CONFIG
format=json|yaml                  serialisation for this invocation's manifest I/O
                                  (manifest-path on load, state-path on verify,
                                  out on describe); when omitted, the operative
                                  file extension decides, else manifest-format
state-path=<path>                 state dump as actual-state source for verify
root=<path>                       root to describe; default "/"
out=<path>                        describe output file; default stdout
on-unreadable=error|warn          describe: fail (default) or omit+warn on an
                                  unreadable scope source; never emit empty
```
All CONFIG knobs are also accepted as key=value options (the same key as in
CONFIG): `manifest-format`, `repo-lock`, `content-store`, `keep-list`,
`signature-verification`, `keyring`, `activation-policy`, `applied-root`. A
command-line option overrides the corresponding preset value.

Verbs (bare words): `apply`, `diff`, `verify`, `status`, `describe`.

Global behaviour (no verb, or a global flag):
- `--version` prints the program name, version, and embedded spec hash to stdout,
  then exits 0.
- `--help` and `-h` print usage to stdout, then exit 0.
- Bare invocation (`zypper declarative` with no verb) prints usage to stdout and
  exits 0; it is treated as a discovery action, not an error, and it never runs a
  default verb (in particular it never converges). When dispatched as a zypper
  subcommand, exit 0 avoids zypper reporting a non-zero subcommand status for a
  plain discovery call.
- An unknown verb, an unknown option, an unknown option value, or a missing
  required value prints usage to stderr and exits 2.

Output streams:
- stderr: diagnostics (errors and warnings), one per line
- stdout: summaries, the diff plan, the status report, the describe document

Exit codes:
- 0 success (converged, no-op, system matches declaration, or describe emitted)
- 1 logical failure (convergence failed and discarded; verify drift; invalid,
  unsafe-YAML, or unverified manifest; state collection failed)
- 2 invocation error (bad arguments; unknown format value; manifest unreadable;
  insufficient privilege; transaction mechanism unavailable; output path
  unwritable; malformed state dump)

Manifest format:
- The manifest is a typed data model: the declarable subset of the SUSE Machinery
  system description (packages, repositories, services, config_files). Its
  canonical serialisation is JSON (`format_version` 1). YAML is an opt-in
  serialisation of the same model, selected by `format=` or by file extension
  (`.yaml` / `.yml`), with `manifest-format` as the CONFIG default. JSON output is
  Machinery-compatible; YAML output is not. A JSON dump is still accepted as YAML
  input for `verify` (JSON is valid YAML 1.2). The applied record is always
  canonical JSON. `describe` reads the live system into this model, so no separate
  collector is required; a full external dump supplied to `verify` has its
  observational scopes ignored. YAML is parsed under a safe profile: no
  code-executing tags, bounded aliases, a single document, explicit typing.

Installation:
- OBS package, distributed via build.opensuse.org. No curl-based installation.
- Target: SL Micro 6.2 and SLES 16.1.

Platform: Linux only.

Transaction binding (decision left open):
- The binding between this tool and the snapshot transaction is abstracted. Under
  `mode=external` a separate mechanism such as `transactional-update run` opens
  the transaction and this tool operates inside it. Under `mode=internal` the
  zypper-merged transactional machinery (SLES 16.1) opens and commits it.
  `mode=auto` detects which applies. The convergence behaviour is identical
  either way. This specification does not commit to whether the transactional
  machinery is a separate tool or part of zypper.

Template deviations (documented per cli-tool template conventions):
- NETWORK-CALLS: the cli-tool template forbids runtime network calls. This tool
  performs no direct network I/O of its own; all package retrieval is delegated
  to the package manager against a declared, pinned, signed repository. The
  supply-chain intent of the constraint (no curl-style fetching) is fully
  honored. Documented as a deviation because the delegated package operation does
  reach the network through the package manager.
- FILE-MODIFICATION input-files: the tool modifies system state (its purpose) but
  never modifies its input, the desired manifest. The constraint as written
  (do not modify input files) holds.
- Privilege: unlike a typical read-only cli-tool, `apply` requires privilege to
  modify the system and to open or operate within a snapshot transaction. The
  read-only verbs (`diff`, `verify`, `status`, `describe`) require only read
  access.

Signal handling: clean exit on SIGTERM and SIGINT. `apply` must not leave a
partially converged snapshot as the default boot target; an interrupted converge
discards the transaction. Translators document the signal-handling approach in
the translation report.

Spec hash: the SHA256 of this specification is embedded in every produced
artifact (source headers, TRANSLATION_REPORT.md `Spec-SHA256:`, binary `--version`
output, RPM spec comment, DEB control `X-PCD-Spec-SHA256:`, Containerfile label,
Makefile variable).

---

## MILESTONE: 0.0.0
Status: pending
Scaffold: true
Hints-file: cli-tool.go.milestones.hints.md

Included BEHAVIORs:
  apply, diff, verify, status, describe, load-desired-manifest,
  load-applied-record, compute-intent-diff, compute-drift, describe-actual-state,
  resolve-format, acquire-transaction-context, converge-packages, converge-files,
  converge-units, write-applied-record

Acceptance criteria:
  ./zypper-declarative --version | grep -q "^zypper-declarative "
  ./zypper-declarative --help | grep -q "usage:"

## MILESTONE: 0.1.0
Status: pending

Included BEHAVIORs:
  status, describe, load-applied-record, load-desired-manifest,
  describe-actual-state, resolve-format

Deferred BEHAVIORs:
  apply, diff, verify, compute-intent-diff, compute-drift,
  acquire-transaction-context, converge-packages, converge-files,
  converge-units, write-applied-record

Acceptance criteria:
  ./zypper-declarative | grep -q "usage:"                        # bare invocation
  test $( ./zypper-declarative >/dev/null 2>&1; echo $? ) -eq 0  # bare exits 0
  ./zypper-declarative describe out=/tmp/d.yaml && head -1 /tmp/d.yaml | grep -vq "^{"  # yaml by extension
  ./zypper-declarative status | grep -q "no declaration applied"

## MILESTONE: 0.2.0
Status: pending

Included BEHAVIORs:
  diff, compute-intent-diff, compute-drift

Deferred BEHAVIORs:
  apply, verify, acquire-transaction-context, converge-packages,
  converge-files, converge-units, write-applied-record

Acceptance criteria:
  ./zypper-declarative diff manifest-path=testdata/desired.json | grep -q "files to delete"

## MILESTONE: 0.3.0
Status: pending

Included BEHAVIORs:
  verify

Deferred BEHAVIORs:
  apply, acquire-transaction-context, converge-packages, converge-files,
  converge-units, write-applied-record

Acceptance criteria:
  ./zypper-declarative verify; test $? -eq 1   # exits 1 on injected drift

## MILESTONE: 0.4.0
Status: pending

Included BEHAVIORs:
  apply, acquire-transaction-context, converge-files, write-applied-record

Deferred BEHAVIORs:
  converge-packages, converge-units

Acceptance criteria:
  ./zypper-declarative apply manifest-path=testdata/files-only.json; test $? -eq 0

## MILESTONE: 0.5.0
Status: pending

Included BEHAVIORs:
  converge-packages

Deferred BEHAVIORs:
  converge-units

Acceptance criteria:
  ./zypper-declarative status | grep -q "packages"   # resolved packages lock recorded

## MILESTONE: 0.6.0
Status: pending

Included BEHAVIORs:
  converge-units

Deferred BEHAVIORs:

Acceptance criteria:
  ./zypper-declarative apply manifest-path=testdata/full.json; test $? -eq 0
  ./zypper-declarative apply manifest-path=testdata/full.json | grep -q "nothing to do"

---

## Changelog

- 2026-05-29: Version 0.5.0. Three fixes surfaced by a first implementation, all
  closed in the spec rather than the code. (1) Defined the top-level CLI contract:
  bare invocation and `--help` print usage to stdout and exit 0 (discovery, not an
  error; never runs a default verb), `--version` exits 0, and an unknown verb,
  option, value, or missing value exits 2 to stderr; documented that all CONFIG
  knobs are also accepted as key=value options. (2) Centralised serialisation
  choice in a new internal behaviour `resolve-format` (explicit `format=` option,
  else the operative file extension, else the `manifest-format` default) and
  routed every manifest read and write through it, so output now honours the `out`
  extension (`describe out=...yaml` writes YAML) symmetrically with input, on
  manifest-path, state-path, and describe out alike; `verify` gained a `format`
  option for the state dump. (3) Pinned the repositories actual-state source to
  the on-disk `/etc/zypp/repos.d` files (readable without elevated privilege, no
  network refresh or privileged cache), and fixed a latent footgun: a scope source
  that cannot be read is never represented as an empty scope. `describe-actual-state`
  now errors on an unreadable source by default, or omits it with a diagnostic
  under `on-unreadable=warn`, and omits genuinely-empty scopes so a bootstrapped
  manifest leaves them unmanaged rather than asserting deletion. Internal callers
  (apply, diff, status, verify) always use the strict reader. Added `on-unreadable`
  and `applied-root` CONFIG knobs.
- 2026-05-29: Version 0.4.0. Added YAML as an opt-in serialisation of the manifest
  alongside the canonical JSON, on a `format=` switch (and `manifest-format` CONFIG
  default, and file-extension inference), for environments that author OS state in
  YAML such as a ZARF-centric workflow. The manifest is now framed explicitly as a
  typed data model with JSON and YAML as serialisations of it; the data model and
  all logic are unchanged. `load-desired-manifest` gained format selection and a
  safe YAML profile (no code-executing tags, bounded aliases, single document,
  explicit typing); `describe` gained a `format=` output option. Manifest identity
  (`desired_sha256`) is now the hash of the canonical serialisation of the parsed
  data model, so JSON and YAML expressions of the same manifest are recognised as
  identical and idempotence holds across a format switch. The applied record stays
  canonical JSON regardless of input format, preserving Machinery readability;
  YAML breaks Machinery and sitar compatibility on the output side only, which is
  accepted for YAML-requesting customers.
- 2026-05-29: Version 0.3.0. Internalised the system-description capability so no
  separate collector program is required. Added a `describe` verb and a single
  internal live-state reader `describe-actual-state` that reads the four
  declarable scopes into the shared Machinery format. Refactored `compute-drift`
  into a pure comparison over two Manifest documents (actual versus reference),
  with the live actual state now produced by `describe-actual-state` or supplied
  as a dump; `verify`, `status`, and `apply` route their actual-state reads
  through this one path. An external Machinery-format producer such as sitar is
  now optional and interchangeable rather than a dependency. Added a `package_name`
  field to ManagedFileRecord (Machinery field) governing the files_extra rule
  (only unpackaged undeclared /etc files count as extra). Reaffirmed
  language-neutrality explicitly.
- 2026-05-29: Version 0.2.0. Adopted the SUSE Machinery system-description JSON
  format (format_version 1) for the desired manifest, the applied record, and the
  actual-state input. Manifest became the declarable subset of that schema using
  the ScopeWrapper idiom (packages, repositories, services, config_files). The
  package lock became a fully resolved packages scope rather than a bespoke NEVRA
  string type. Repository pinning became declarative via an in-band repositories
  scope. Added absent-versus-empty scope semantics, a content_ref for declared
  file content, and a worked JSON manifest example.
- 2026-05-29: Version 0.1.0. Initial specification of `zypper-declarative`, a
  reconciling converger surfaced as a zypper subcommand. Verbs over internal
  convergence behaviours. Two-diff model (intent diff for deletions, drift diff
  for verification). Applied record stored under /usr within the generation. /etc
  treated as a nested btrfs subvolume with `etc.syncpoint` excluded. Transaction
  binding abstracted between an external mechanism and the zypper-internal
  mechanism, with the decision deliberately left open. Secrets, kernel cmdline,
  and sysctl domains reserved for a later Version.
