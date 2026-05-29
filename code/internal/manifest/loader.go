// generated from spec: zypper-declarative.spec.md sha256:714e75ff672557d2c7344736a5f36b52afa37e0f565b07de83e1b18cc4492014
//
// load-desired-manifest: read, format-select, safe-parse, schema-validate,
// (optionally) signature-verify, and compute the canonical-model identity hash.
package manifest

import (
	"os"

	"github.com/mge1512/zypper-declarative/internal/diag"
)

// LoadOptions controls load-desired-manifest.
type LoadOptions struct {
	// ExplicitFormat is the value of the format= option ("" if unset).
	ExplicitFormat string
	// DefaultFormat is the manifest-format CONFIG default.
	DefaultFormat Format
	// VerifySignature enables signature verification (CONFIG signature-verification).
	VerifySignature bool
	// Keyring is the keyring path used when VerifySignature is true.
	Keyring string
}

// LoadResult is the output of Load.
type LoadResult struct {
	Manifest      Manifest
	DesiredSHA256 string
}

// Load implements load-desired-manifest STEPS 1–6. It returns a Diagnostic on
// failure (never exits): invocation for read/unknown-format failures; manifest
// for unsafe-YAML, schema, or signature failures.
func Load(path string, opts LoadOptions) (LoadResult, *diag.Diagnostic) {
	// STEP 1 — read
	data, err := os.ReadFile(path)
	if err != nil {
		return LoadResult{}, diag.New(diag.DomainInvocation, "manifest unreadable: %v", err)
	}

	// STEP 2 — determine format
	format, ferr := ResolveFormat(opts.ExplicitFormat, path, opts.DefaultFormat)
	if ferr != nil {
		return LoadResult{}, diag.New(diag.DomainInvocation, "%v", ferr)
	}

	// STEP 3 — parse (safe YAML profile when YAML)
	m, _, perr := Parse(data, format)
	if perr != nil {
		return LoadResult{}, diag.New(diag.DomainManifest, "manifest error: %v", perr)
	}

	// STEP 4 — schema validation
	if verr := Validate(&m); verr != nil {
		return LoadResult{}, diag.New(diag.DomainManifest, "manifest error: schema violation: %v", verr)
	}

	// STEP 5 — signature verification (when enabled)
	if opts.VerifySignature {
		if sigErr := verifySignature(path, data, opts.Keyring); sigErr != nil {
			return LoadResult{}, diag.New(diag.DomainManifest, "manifest error: signature verification failed: %v", sigErr)
		}
	}

	// STEP 6 — canonical-model identity hash
	hash := CanonicalHash(&m)
	return LoadResult{Manifest: m, DesiredSHA256: hash}, nil
}

// LoadDump reads and schema-validates a state dump (verify STEP 2). A malformed
// dump returns an invocation error per the verify ERRORS table.
func LoadDump(path string, explicitFormat string, def Format) (Manifest, *diag.Diagnostic) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, diag.New(diag.DomainInvocation, "state dump unreadable: %v", err)
	}
	format, ferr := ResolveFormat(explicitFormat, path, def)
	if ferr != nil {
		return Manifest{}, diag.New(diag.DomainInvocation, "%v", ferr)
	}
	m, _, perr := Parse(data, format)
	if perr != nil {
		return Manifest{}, diag.New(diag.DomainInvocation, "malformed state dump: %v", perr)
	}
	if verr := Validate(&m); verr != nil {
		return Manifest{}, diag.New(diag.DomainInvocation, "malformed state dump: %v", verr)
	}
	return m, nil
}
