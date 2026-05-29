# TEST_REPORT

## Spec Hashes
- **Spec-SHA256 (merged):** `714e75ff672557d2c7344736a5f36b52afa37e0f565b07de83e1b18cc4492014`
- **Spec-SHA256 (host):** `714e75ff672557d2c7344736a5f36b52afa37e0f565b07de83e1b18cc4492014`
- **Included-Specs:** (none)

| Path | SHA256 |
|------|--------|


## Metadata
- **LLM-Name:** `mistral-large-2512`
- **Mode:** `test-author`
- **Deployment-Template:** `cli-tool.template.md v0.3.29`
- **Preset-Resolution:** (none)
- **Hints-Files-Read:** (none)
- **Test-Compile-Gate:** `pass`
- **Binary-Discovery-Path:** `../../zypper-declarative`
- **Source-Path (if built by tests):** `../../cmd/zypper-declarative/main.go`


## Target Language
- Resolved: `Go` (default from `cli-tool` template)


## Tests Produced
| Test Function | Coverage |
|---------------|----------|
| `TestApply_Success` | `apply` with no prior state (exit 0, no-op or convergence)
| `TestApply_InvalidManifest` | `apply` with invalid manifest (exit 1, manifest error)
| `TestDiff_Success` | `diff` with desired manifest (exit 0, intent diff)
| `TestVerify_NoAppliedRecord` | `verify` with no applied record (exit 2, no declaration)
| `TestStatus_NoAppliedRecord` | `status` with no applied record (exit 0, no declaration)
| `TestDescribe_Success` | `describe` with default args (exit 0, valid JSON)
| `TestDescribe_YAML` | `describe` with YAML output (exit 0, valid YAML)


## Specification Ambiguities
- None encountered.


## Notes
- Tests assume the binary is built at `../../zypper-declarative` (per `BINARY-LOCATION: project-root`).
- Test fixtures are **structurally complete** (valid `Manifest` JSON with all required scopes).
- The `TestMain` helper builds the binary from `../../cmd/zypper-declarative/main.go` if it doesn't exist.