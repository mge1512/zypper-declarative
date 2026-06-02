// generated from spec: zypper-declarative.spec.md sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
//
// load-desired-manifest: read and validate the desired manifest into the model,
// selecting the input serialisation via resolve-format and computing the
// canonical-model identity hash.
package manifest

import (
	"os"
)

// LoadOptions configures a desired-manifest load.
type LoadOptions struct {
	Explicit  *Format // an explicit format= option, or nil
	Default   Format  // the manifest-format CONFIG default
	SigVerify bool    // signature-verification on/off
	Keyring   string  // keyring path when SigVerify
	RejectObs bool    // reject non-empty observational scopes (a desired manifest)
}

// LoadResult is the output of a successful load.
type LoadResult struct {
	Manifest      *Manifest
	DesiredSHA256 string
}

// Load implements BEHAVIOR/INTERNAL: load-desired-manifest.
//
//  1. Read the file. On read failure -> invocation error.
//  2. Resolve the input format via resolve-format. Unknown explicit format ->
//     invocation error (handled by the caller validating format= up front).
//  3. Parse into the model (safe YAML profile for YAML).
//  4. Validate against the schema; reject non-empty observational scopes when
//     RejectObs.
//  5. Verify the signature when enabled.
//  6. Compute desired_sha256 from the canonical model.
func Load(path string, opts LoadOptions) (*LoadResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, &ParseError{Domain: "invocation", Message: "manifest unreadable: " + err.Error()}
	}
	f := ResolveFormat(opts.Explicit, path, opts.Default)
	m, err := Parse(data, f)
	if err != nil {
		// A malformed document on the desired-manifest path is a manifest error
		// when it is structurally a manifest we could not validate; but an
		// unparseable byte stream is an invocation-class read problem only for a
		// supplied state dump. For load-desired-manifest the spec maps unknown
		// format to invocation (handled by the caller) and schema/safety failures
		// to manifest. A parse failure here is reported as a manifest error so
		// apply/diff/verify map it to exit 1, except a YAML safe-profile rejection
		// which Parse already tags Domain=manifest.
		if pe, ok := err.(*ParseError); ok {
			if pe.Domain == "invocation" {
				// Re-tag a structural JSON/YAML parse failure of a manifest as a
				// manifest error (invalid manifest), per ERRORS of load-desired-manifest.
				return nil, &ParseError{Domain: "manifest", Message: "manifest invalid: " + pe.Message}
			}
			return nil, pe
		}
		return nil, err
	}
	if err := m.Validate(opts.RejectObs); err != nil {
		return nil, err
	}
	if opts.SigVerify {
		if err := verifySignature(path, opts.Keyring); err != nil {
			return nil, err
		}
	}
	sum := m.CanonicalSHA256()
	m.Meta.DesiredSHA256 = sum
	return &LoadResult{Manifest: m, DesiredSHA256: sum}, nil
}

// verifySignature verifies the manifest signature against the keyring.
// Signature verification is not implemented in this version; when enabled it
// returns a manifest error so an unverified manifest is rejected rather than
// silently accepted. Callers default signature-verification off for the
// read-only/offline paths exercised here.
func verifySignature(path, keyring string) error {
	return &ParseError{Domain: "manifest", Message: "manifest error: signature verification is enabled but no keyring/signature could be verified in this build"}
}

// LoadStateDump reads a captured actual-state dump (offline) into the model. A
// malformed dump is an invocation error (exit 2) per diff/verify ERRORS.
func LoadStateDump(path string, explicit *Format, def Format) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, &ParseError{Domain: "invocation", Message: "state dump unreadable: " + err.Error()}
	}
	f := ResolveFormat(explicit, path, def)
	m, err := Parse(data, f)
	if err != nil {
		if pe, ok := err.(*ParseError); ok {
			// Any parse/safety failure on a supplied state dump is a malformed
			// dump -> invocation error.
			return nil, &ParseError{Domain: "invocation", Message: "malformed state dump: " + pe.Message}
		}
		return nil, &ParseError{Domain: "invocation", Message: "malformed state dump"}
	}
	if m.Meta.FormatVersion != 1 {
		return nil, &ParseError{Domain: "invocation", Message: "malformed state dump: meta.format_version must be 1"}
	}
	return m, nil
}
