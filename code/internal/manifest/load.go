// generated from spec: zypper-declarative.spec.md sha256:58e1636e2de82ab81a5cd3f81d6b3c9ac6a8976e18f9abb2bbd2b2aba56fe4d4
//
// load-desired-manifest: reads and validates the desired manifest into the
// shared data model, selecting the input serialisation via resolve-format,
// applying a safe YAML profile when the input is YAML, verifying the signature
// when enabled, and computing the manifest's canonical-model identity hash.
package manifest

import (
	"os"

	"github.com/mge1512/zypper-declarative/internal/diag"
)

// SignatureVerifier verifies a manifest's detached or embedded signature. The
// production binding is resolved at deploy time; this interface keeps the
// dependency abstract and testable. The default no-op verifier accepts.
type SignatureVerifier interface {
	Verify(data []byte, keyring string) error
}

// LoadOptions carry the resolve-format inputs and the signature policy.
type LoadOptions struct {
	ExplicitFormat        Format
	ExplicitFormatGiven   bool
	DefaultFormat         Format
	SignatureVerification bool
	Keyring               string
	Verifier              SignatureVerifier
}

// LoadDesiredManifest reads, format-resolves, parses (safe-profile for YAML),
// schema-validates, signature-verifies (if enabled), and hashes a desired
// manifest. It returns the Manifest and its canonical-model desired_sha256, or a
// Diagnostic to the caller (it does not exit).
func LoadDesiredManifest(path string, opts LoadOptions) (Manifest, string, *diag.Diagnostic) {
	// Step 1: read the file.
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, "", diag.Errorf(diag.DomainInvocation, "manifest unreadable: %v", err)
	}

	// Step 2: resolve the input format.
	format := ResolveFormat(opts.ExplicitFormat, opts.ExplicitFormatGiven, path, opts.DefaultFormat)

	// Step 3: parse into the data model (safe profile for YAML).
	m, perr := Parse(data, format)
	if perr != nil {
		if perr == ErrUnsafeYAML {
			return Manifest{}, "", diag.Errorf(diag.DomainManifest, "manifest invalid: %v", perr)
		}
		return Manifest{}, "", diag.Errorf(diag.DomainManifest, "manifest invalid: %v", perr)
	}

	// Step 4: schema validation.
	if verr := Validate(m); verr != nil {
		return Manifest{}, "", diag.Errorf(diag.DomainManifest, "manifest invalid: %v", verr)
	}

	// Step 5: signature verification, if enabled.
	if opts.SignatureVerification && opts.Verifier != nil {
		if serr := opts.Verifier.Verify(data, opts.Keyring); serr != nil {
			return Manifest{}, "", diag.Errorf(diag.DomainManifest, "manifest signature invalid: %v", serr)
		}
	}

	// Step 6: compute desired_sha256 (canonical-model hash, format-independent).
	hash := CanonicalSHA256(m)
	return m, hash, nil
}

// NoopVerifier is a SignatureVerifier that accepts everything. It is the default
// binding when no keyring/signature mechanism is configured for the run; signing
// is a deployment-layer concern and not exercisable in a non-privileged test.
type NoopVerifier struct{}

// Verify always succeeds.
func (NoopVerifier) Verify(_ []byte, _ string) error { return nil }
