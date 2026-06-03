# UPGRADE: zypper-declarative spec v0.6.8 -> v0.6.9 (for KIT)

Purpose: this file states the EXACT delta between spec v0.6.8 and v0.6.9 so you do
NOT need to diff the spec against the existing code to find what changed. Apply the
two changes below to the existing implementation, re-embed the new spec hash, and
re-verify. Do NOT regenerate from scratch; the v0.6.8 implementation is correct
except for the one shipped bug described in change (1).

- OLD spec: v0.6.8, SHA256 1641bb4413b82fecb081125067107bd5a4e30a8393edc778ead646207d68da5e
- NEW spec: v0.6.9, SHA256 aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3

There are TWO behavioral changes (1 and 2) plus one spec clarification the code
already satisfies (3) and one harness/doc note (4). Everything else is unchanged.

## Change 1 (BUG FIX): symlink classification - only true alternatives use the alternatives DB

Symptom in the v0.6.8 build (observed live): `describe` on a real system emitted

    error: files: cannot query alternatives for /etc/motd.d/cockpit

under the default `on-unreadable=error`, and under `on-unreadable=warn` produced ~24
spurious warnings of the form `alternatives unreadable for <path>`, including all
`/etc/crypto-policies/back-ends/*.config`, `/etc/motd.d/cockpit`, and
`/etc/issue.d/cockpit.issue`.

Root cause: the implementation routed NON-alternatives symlinks through the
`update-alternatives` query path. Crypto-policies back-end configs, motd.d and
issue.d links are NOT alternatives; querying `update-alternatives` for them returns
nothing, which the code then treated as an `on_unreadable` condition.

Required behavior (v0.6.9): classify a symlink's MECHANISM before judging it.

- A symlink is an ALTERNATIVES symlink IF AND ONLY IF it is located under
  `/etc/alternatives/` OR it appears as a master or slave in some
  `/var/lib/alternatives/<name>` admin file. ONLY these are resolved against the
  alternatives database (`update-alternatives --query <name>` or reading
  `/var/lib/alternatives/<name>`).
- Every OTHER symlink is a NON-ALTERNATIVES symlink. Examples that MUST take this
  path: `/etc/crypto-policies/back-ends/*.config` (these point into
  `/usr/share/crypto-policies/<policy>/...`), `/etc/motd.d/*`, `/etc/issue.d/*`, and
  any other package-owned symlink. For these:
  - judge by the NORMAL symlink target rule: on-disk target equals the target a fresh
    install of the owning package would establish -> SUPPRESS (pristine); differs ->
    EMIT as a type "link" record with the verbatim on-disk target.
  - NEVER call `update-alternatives` for them.
  - the absence of an alternatives entry is NOT an `on_unreadable` condition.
  - on a default-policy system, the crypto-policies back-end links point where the
    package set them, so they are PRISTINE and SUPPRESSED (they must NOT appear in
    config_files and must NOT emit any diagnostic).
- For an ALTERNATIVES symlink (unchanged from before): reproducible target is the
  alternatives auto/best provider; on-disk EQUALS auto/best -> SUPPRESS, DIFFERS ->
  EMIT with verbatim target. A SLAVE alternative (no own admin file; auto/best lives
  in the master's admin file, e.g. the `.gz` man-page links) whose auto/best cannot
  be determined is EMITTED conservatively. Resolving slaves via the master admin file
  to suppress the default ones is a permitted refinement, not required.
- `on_unreadable` applies to an alternatives symlink ONLY when the alternatives
  database genuinely cannot be read (a real IO/permission failure on
  `/var/lib/alternatives`), which is rare. It must NEVER be used for the routine case
  of "this symlink is not an alternative".

Where this lives in the code (C++ build): the symlink-handling branch of the
config_files / actual-state collection, the same area that in build 02 over-emitted
`/etc/alternatives/*`. Add the mechanism classification in front of the existing
alternatives-resolution branch; send only true alternatives into it; send everything
else to the normal target-match branch.

Expected result after the fix: on the live host, `describe` under the DEFAULT
(`on-unreadable=error`) completes without aborting on `/etc/motd.d/cockpit`, and the
24 `alternatives unreadable` warnings are gone (the crypto-policies/motd/issue links
are suppressed as pristine, the true `/etc/alternatives/*` slaves are emitted
conservatively rather than warned).

## Change 2: `init` forces on_unreadable=warn

`init` (the onboarding verb) MUST force `on_unreadable=warn` for its internal live
read, overriding both the default (`error`) and any `on-unreadable=error` passed on
the command line. Rationale: onboarding a real machine must not abort on a protected
root-only file (e.g. `/etc/libaudit.conf`) or an indeterminable source.

`init` is the ONLY verb that overrides the knob. `describe`, `diff`, `verify`, and
`apply` keep `error` as their default and honor whatever `on-unreadable` value is
passed (this was already the case in v0.6.8; do not change it).

Where this lives: `init`'s step 1 (the `describe-actual-state` call). Set the
on_unreadable argument to warn unconditionally there.

## Change 3 (CLARIFICATION, code already matches): packages_divergent empty-field wildcard

`compute-drift` step 4 (packages_divergent) now states explicitly that an EMPTY
identity field in a REFERENCE package element is a WILDCARD matching any actual
value: a desired `{name: nginx, version: ""}` matches the resolved actual
`{name: nginx, version: "1.27.4", ...}` and is NOT divergent; only non-empty
reference fields must match, and only the reference side wildcards. The existing C++
code ALREADY does this (it was an undocumented judgment call inherited from v0.6.7,
noted in the prior TRANSLATION_REPORT under "Specification ambiguities"). No code
change is required; this change only makes the behavior normative in the spec. Verify
the code still matches and keep it.

## Change 4 (DOC/HARNESS note, no behavior change): long-running examples

The EXAMPLES section now notes that examples driving a real `describe` or full scan
are O(installed packages) and can take minutes on a system with thousands of
packages (notably scope=full describe and the describe-then-diff idempotence case).
The test harness MUST allow a generous per-example timeout (or run long cases in the
background and wait) and MUST NOT kill or fail a still-running long example. This
matches what already had to be done at build time (the scope=full test was run in the
background). Ensure the harness does not impose a short timeout that kills the
scope=full test.

## After applying the changes

1. Re-embed the new spec hash everywhere it appears (source headers, `version`
   output, RPM spec comment, etc.): replace
   1641bb4413b82fecb081125067107bd5a4e30a8393edc778ead646207d68da5e with
   aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3, and the version
   string 0.6.8 with 0.6.9 (the VERSION file already says 0.6.9).
2. Rebuild. `version` must print `zypper-declarative 0.6.9 spec:aafbb315...`.
3. Re-run the test suite. All previously-passing tests must still pass. The
   self-checks run unprivileged with `on-unreadable=warn` and MUST NOT use sudo (an
   interactive sudo prompt hangs the build).
4. Manual confirmation on the host (as root): `zypper declarative describe` under the
   DEFAULT mode (no on-unreadable flag) must complete with exit 0 and emit no
   `alternatives unreadable` diagnostic for crypto-policies/motd/issue paths.
