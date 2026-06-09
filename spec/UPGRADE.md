# zypper-declarative - UPGRADE notes for spec 0.6.10

For the translating LLM (KIT). This pass adds two transaction modes, `snapshot`
and `plain`, to the existing `TransactionMode` axis and broadens `auto`. It is a
focused, mostly additive change to three existing behaviours. Do an incremental
translation: re-translate only the affected behaviours and the new
examples/invariants, in each target language, and preserve everything else.

## Change-impact assessment

- recommendation:              incremental
- structural impact:           low
- blast radius:                three existing behaviours (`apply`, `init`,
                               `acquire-transaction-context`), two existing TYPES
                               (`TransactionMode`, `TransactionContext`), CONFIG,
                               plus three new EXAMPLES and four new INVARIANTS. No
                               new BEHAVIOR section, no new file, no new verb.
- scaffold affected:           no
- released milestone affected: the milestone(s) that already contain `apply`,
                               `init`, and `acquire-transaction-context` are in
                               scope only for those behaviours; the rest of each
                               milestone is untouched.
- consistency risk:            low
- reasoning:                   The new modes extend one enum and add branches at a
                               single chokepoint (`acquire-transaction-context`)
                               and at two finalisation steps. The convergence code
                               path is unchanged; `mode` governs only the undo
                               unit and finalisation.

## What changed in the spec

1. META: Version 0.6.9 -> 0.6.10
2. TYPES: `TransactionMode` gains `snapshot` and `plain`. `TransactionContext`
   `root`/`opened_here` comments generalised (root is "/" for the two new modes;
   `opened_here` is false for `plain`).
3. BEHAVIOR/INTERNAL acquire-transaction-context: `auto` now resolves by
   substrate across four outcomes; new bindings for `snapshot` (snapper
   pre-snapshot, root="/") and `plain` (open nothing, root="/"); a new error for
   snapshot-without-snapper; relaxed postcondition (root need not differ from the
   running root).
4. BEHAVIOR apply, BEHAVIOR init: the "Seal and activate" step is now a mode-aware
   "Finalise" step; postconditions generalised, including the explicit statement
   that `plain` is non-atomic.
5. CONFIG: `transaction-mode` value list extended; `activation-policy` noted as
   transactional-only.
6. INVARIANTS: four added (snapshot bracket/live/rollback; plain
   no-undo/non-atomic; auto substrate resolution; mode governs only the undo unit
   not the converged result).
7. EXAMPLES: three added (`apply_mode_plain_in_place`,
   `apply_mode_snapshot_brackets_with_snapper`,
   `apply_auto_resolves_to_snapshot_on_non_transactional`).

## Translate this (incremental scope), in each target language (C++, Go, Rust)

- `acquire-transaction-context`: the four-way `auto` resolution and the `snapshot`
  and `plain` bindings. This is the only place mode becomes action.
- `apply` and `init`: the mode-aware finalisation step (transactional seal + boot
  target; `snapshot` closes the snapper post-snapshot; `plain` does nothing).
- The `TransactionMode` enum/constant set: add `snapshot` and `plain`.
- CONFIG parsing/validation: accept the two new `transaction-mode` values.
- Tests for the three new EXAMPLES and the four new INVARIANTS.

## Preserve this (do not regenerate)

- Every other behaviour and BEHAVIOR/INTERNAL, unchanged: `describe`,
  `describe-actual-state`, `diff`, `verify`, `status`, `resolve-format`,
  `load-*`, `compute-*`, `converge-*`, `write-applied-record`.
- The convergence code path. Do NOT fork `converge-packages`, `converge-files`,
  or `converge-units` by mode; they operate against `context.root` exactly as
  before. For `snapshot` and `plain` that root is "/".
- The existing transactional modes (`external`, `internal`) and their behaviour.
- Append rows to TRANSLATION_REPORT for the new examples rather than rewriting it.

## Required test assertions for the new observable invariants

- snapshot: a pre-snapshot is taken before convergence and a post-snapshot after;
  the result is live on the running root; no boot target is set; the pair is the
  rollback unit. Use a snapper test double; assert the pre/post calls bracket the
  converge calls.
- plain: no snapshot is created and no transaction is opened; `opened_here` is
  false; `context.root` is "/"; changes are made in place.
- auto resolution: on a non-transactional btrfs-with-snapper substrate, `auto`
  resolves to `snapshot` (not internal/external, not plain). Drive substrate
  detection through a test double, not the live host.
- mode-invariance of convergence: the same converge calls are issued regardless
  of mode (only the surrounding undo unit differs).

## Watch-items (avoid drift)

- `plain` is non-atomic by design. The convergence steps still read "on failure
  discard and exit 1"; under `plain` there is no undo unit to discard, so a
  literal implementation correctly exits non-zero with the partial changes left in
  place. Do NOT fabricate a rollback path for `plain` (there is none) and do NOT
  suppress the partial-change reality - the postconditions and an invariant state
  it explicitly.
- `snapshot` requires snapper on the running root; its absence is a transaction
  error (exit 2, domain=transaction), parallel to the existing external/internal
  unavailable errors. Do not silently fall back.
- The snapper bracket and the substrate probe are external-tool interactions
  (snapper, the transactional-update/btrfs probes). Invoke them through the same
  command-runner abstraction the existing modes use. Do NOT add new modules to
  go.mod / Cargo.toml / the C++ build for them, and do NOT fabricate any tool
  versions; they are runtime binaries, not build dependencies.
- CLI stays bare key=value: `mode=plain`, `mode=snapshot`. No `--flag` form.
- `activation-policy` is meaningful only for the transactional modes; `snapshot`
  and `plain` schedule no activation.

## Build note

Re-bake the spec-hash so every artifact embeds the 0.6.10 spec SHA256. No schema
migration is involved (this tool has no database). CHANGELOG.md and VERSION are
updated beside the spec; CHANGELOG.md is not read by the translator and is not
covered by the spec hash.

---

## Changelog

- 2026-06-08: initial UPGRADE notes for the `snapshot` and `plain` transaction
  modes (spec 0.6.10). Incremental scope: `acquire-transaction-context`, `apply`,
  `init`, the `TransactionMode` enum, CONFIG validation, and the new
  examples/invariants. Convergence code path and all other behaviours preserved.
