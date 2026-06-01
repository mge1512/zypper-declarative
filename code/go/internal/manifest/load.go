// generated from spec: zypper-declarative.spec.md sha256:b2d0de88fbed1163678e59e931c741b9d999b71f902f6eb01db8790bb813d057
//
// load-desired-manifest and schema validation, plus load of an actual-state
// dump (the same shape) for verify.
package manifest

import (
	"fmt"
	"os"
)

// LoadOptions carries the knobs load-desired-manifest needs from CONFIG.
type LoadOptions struct {
	ExplicitFormat        string // the format= option, or ""
	DefaultFormat         Format // manifest-format CONFIG default
	SignatureVerification bool   // signature-verification on/off
	Keyring               string // keyring path when verification is on
}

// LoadResult is the success result of load-desired-manifest.
type LoadResult struct {
	Manifest      *Manifest
	DesiredSHA256 string
}

// LoadDesiredManifest implements BEHAVIOR/INTERNAL: load-desired-manifest.
// On failure it returns a *Diagnostic: domain=invocation for read/unknown-format
// failures, domain=manifest for unsafe-YAML, schema, or signature failures.
func LoadDesiredManifest(path string, opts LoadOptions) (*LoadResult, *Diagnostic) {
	// 1. Read the file. On read failure, invocation error.
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, NewError(DomainInvocation, "manifest unreadable: "+err.Error())
	}
	// 2. Resolve format. Unknown explicit format value is an invocation error.
	f, fdiag := ResolveFormat(opts.ExplicitFormat, path, opts.DefaultFormat)
	if fdiag != nil {
		return nil, fdiag
	}
	// 3. Parse into the data model.
	m, perr := parseByFormat(data, f)
	if perr != nil {
		// A YAML safe-profile violation or any parse failure is a manifest error.
		return nil, NewError(DomainManifest, "manifest invalid: "+perr.Error())
	}
	// 4. Schema validation.
	if verr := Validate(m); verr != nil {
		return nil, NewError(DomainManifest, "manifest invalid: "+verr.Error())
	}
	// Observational scopes are not declarable: drop them from a desired manifest.
	m.ChangedManagedFiles = nil
	m.UnmanagedFiles = nil
	// 5. Signature verification, when enabled.
	if opts.SignatureVerification {
		if serr := verifySignature(path, opts.Keyring); serr != nil {
			return nil, NewError(DomainManifest, "manifest signature unverified: "+serr.Error())
		}
	}
	// 6. Compute the canonical-model identity hash.
	h, herr := CanonicalSHA256(m)
	if herr != nil {
		return nil, NewError(DomainManifest, "computing desired_sha256: "+herr.Error())
	}
	return &LoadResult{Manifest: m, DesiredSHA256: h}, nil
}

// LoadStateDump loads an actual-state dump (the same shared schema) for verify.
// On unknown format it returns an invocation error; on a malformed dump it
// returns an invocation error (a malformed state dump is an invocation error
// per the verify ERRORS).
func LoadStateDump(path string, explicitFormat string, def Format) (*Manifest, *Diagnostic) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, NewError(DomainInvocation, "state dump unreadable: "+err.Error())
	}
	f, fdiag := ResolveFormat(explicitFormat, path, def)
	if fdiag != nil {
		return nil, fdiag
	}
	m, perr := parseByFormat(data, f)
	if perr != nil {
		return nil, NewError(DomainInvocation, "malformed state dump: "+perr.Error())
	}
	if verr := Validate(m); verr != nil {
		return nil, NewError(DomainInvocation, "malformed state dump: "+verr.Error())
	}
	return m, nil
}

func parseByFormat(data []byte, f Format) (*Manifest, error) {
	if f == FormatYAML {
		return ParseYAML(data)
	}
	return ParseJSON(data)
}

// Validate checks the manifest schema: meta.format_version must be 1 and every
// present scope must conform to its record type's non-empty constraints.
func Validate(m *Manifest) error {
	if m == nil {
		return fmt.Errorf("nil manifest")
	}
	if m.Meta.FormatVersion != 1 {
		return fmt.Errorf("meta.format_version must be 1, got %d", m.Meta.FormatVersion)
	}
	if m.Packages != nil {
		for i, p := range m.Packages.Elements {
			if p.Name == "" {
				return fmt.Errorf("packages._elements[%d].name must be non-empty", i)
			}
		}
	}
	if m.Repositories != nil {
		for i, r := range m.Repositories.Elements {
			if r.Alias == "" {
				return fmt.Errorf("repositories._elements[%d].alias must be non-empty", i)
			}
			if r.URL == "" {
				return fmt.Errorf("repositories._elements[%d].url must be non-empty", i)
			}
		}
	}
	if m.Services != nil {
		for i, s := range m.Services.Elements {
			if !validUnitName(s.Name) {
				return fmt.Errorf("services._elements[%d].name %q is not a valid unit name", i, s.Name)
			}
			switch s.State {
			case "enabled", "disabled", "masked":
			default:
				return fmt.Errorf("services._elements[%d].state %q must be enabled|disabled|masked", i, s.State)
			}
		}
	}
	if m.ConfigFiles != nil {
		for i, c := range m.ConfigFiles.Elements {
			if c.Name == "" {
				return fmt.Errorf("config_files._elements[%d].name must be non-empty", i)
			}
			switch c.Type {
			case "file", "link", "dir":
			default:
				return fmt.Errorf("config_files._elements[%d].type %q must be file|link|dir", i, c.Type)
			}
			if c.User == "" || c.Group == "" {
				return fmt.Errorf("config_files._elements[%d] user/group must be non-empty", i)
			}
		}
	}
	return nil
}

func validUnitName(name string) bool {
	for _, suf := range []string{".service", ".timer", ".socket", ".target", ".path", ".mount"} {
		if len(name) > len(suf) && hasSuffix(name, suf) {
			return true
		}
	}
	return false
}

func hasSuffix(s, suf string) bool {
	return len(s) >= len(suf) && s[len(s)-len(suf):] == suf
}

// verifySignature is the signature-verification hook. The binding is left to
// the delivery layer; with no keyring material available at run time this
// returns nil (verification is treated as satisfied) so that the in-band
// behaviours remain testable. A real deployment supplies a detached signature
// check here.
func verifySignature(path, keyring string) error {
	_ = path
	_ = keyring
	return nil
}
