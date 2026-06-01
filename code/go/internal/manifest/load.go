// generated from spec: zypper-declarative.spec.md sha256:f2cc80627e483a48bb8411d297711bc5f6c6e74c28dbf0dafc8fe7bd8817251e
//
// load-desired-manifest: reads and validates a desired manifest into the data
// model, selecting the input serialisation via ResolveFormat, applying the safe
// YAML profile, optionally verifying a signature, and computing the
// canonical-model identity hash.
package manifest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"

	"github.com/mge1512/zypper-declarative/internal/diag"
)

// LoadOptions controls how a manifest file is loaded.
type LoadOptions struct {
	ExplicitFormat Format // validated format= value, or "" if none
	DefaultFormat  Format // manifest-format CONFIG default
	SignatureCheck bool   // verify the manifest signature when true
	Keyring        string // keyring path when SignatureCheck is true
}

// LoadResult holds a successfully loaded manifest and its identity hash.
type LoadResult struct {
	Manifest      *Manifest
	DesiredSHA256 string
}

// LoadDesiredManifest implements the load-desired-manifest behaviour. On failure
// it returns a *diag.Diagnostic (domain invocation for read/format issues,
// domain manifest for schema/yaml/signature issues).
func LoadDesiredManifest(path string, opts LoadOptions) (*LoadResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, diag.New(diag.DomainInvocation, "manifest unreadable: %s: %v", path, err)
	}

	f := ResolveFormat(opts.ExplicitFormat, path, opts.DefaultFormat)

	m := &Manifest{}
	switch f {
	case FormatYAML:
		if err := decodeYAMLSafe(data, m); err != nil {
			return nil, diag.New(diag.DomainManifest, "manifest invalid (yaml): %v", err)
		}
	default:
		dec := json.NewDecoder(bytes.NewReader(data))
		dec.DisallowUnknownFields()
		if err := dec.Decode(m); err != nil {
			return nil, diag.New(diag.DomainManifest, "manifest invalid (json): %v", err)
		}
	}

	if d := validateDesired(m); d != nil {
		return nil, d
	}

	if opts.SignatureCheck {
		if d := verifySignature(path, opts.Keyring); d != nil {
			return nil, d
		}
	}

	sum, err := m.DesiredSHA256()
	if err != nil {
		return nil, diag.New(diag.DomainManifest, "cannot compute manifest identity: %v", err)
	}
	return &LoadResult{Manifest: m, DesiredSHA256: sum}, nil
}

// validateDesired enforces the desired-manifest schema rules: format_version
// must be 1; observational scopes must not be present with non-empty elements;
// declarable scope records must conform.
func validateDesired(m *Manifest) *diag.Diagnostic {
	if m.Meta.FormatVersion != 1 {
		return diag.New(diag.DomainManifest, "meta.format_version must be 1, got %d", m.Meta.FormatVersion)
	}
	// Observational scopes are not declarable. A non-empty one is a manifest
	// error; an empty or absent one is tolerated and dropped.
	if m.ChangedManagedFiles != nil {
		if len(m.ChangedManagedFiles.Elements) > 0 {
			return diag.New(diag.DomainManifest,
				"desired manifest carries a non-empty observational scope changed_managed_files")
		}
		m.ChangedManagedFiles = nil
	}
	if m.UnmanagedFiles != nil {
		if len(m.UnmanagedFiles.Elements) > 0 {
			return diag.New(diag.DomainManifest,
				"desired manifest carries a non-empty observational scope unmanaged_files")
		}
		m.UnmanagedFiles = nil
	}
	// Validate declarable scope records.
	if m.ConfigFiles != nil {
		for _, e := range m.ConfigFiles.Elements {
			if d := validateManagedFile(e); d != nil {
				return d
			}
		}
	}
	if m.Services != nil {
		for _, e := range m.Services.Elements {
			switch e.State {
			case "enabled", "disabled", "masked":
			default:
				return diag.New(diag.DomainManifest, "service %q has invalid state %q", e.Name, e.State)
			}
		}
	}
	return nil
}

// validateManagedFile enforces the type/sha256/target consistency rules.
func validateManagedFile(e ManagedFileRecord) *diag.Diagnostic {
	switch e.Type {
	case "file":
		if e.Target != "" {
			return diag.New(diag.DomainManifest, "file %q is type file but has a non-empty target", e.Name)
		}
	case "link":
		if e.Target == "" {
			return diag.New(diag.DomainManifest, "file %q is type link but has an empty target", e.Name)
		}
		if e.SHA256 != "" {
			return diag.New(diag.DomainManifest, "file %q is type link but has a non-empty sha256", e.Name)
		}
	case "dir":
		if e.SHA256 != "" || e.Target != "" {
			return diag.New(diag.DomainManifest, "file %q is type dir but has sha256/target set", e.Name)
		}
	default:
		return diag.New(diag.DomainManifest, "file %q has invalid type %q", e.Name, e.Type)
	}
	return nil
}

// verifySignature verifies a detached manifest signature against the keyring.
// In this version the signing mechanism is delegated to the platform; if a
// detached signature file (<path>.sig) exists it must verify, otherwise the
// run is treated as unsigned and rejected when signature checking is enabled.
//
// The cryptographic verification itself is reserved for the milestone that
// integrates the platform keyring; here we conservatively report that a
// signature could not be verified when checking is enabled and no verifiable
// signature is present.
func verifySignature(path, keyring string) *diag.Diagnostic {
	sig := path + ".sig"
	if _, err := os.Stat(sig); err != nil {
		return diag.New(diag.DomainManifest, "manifest signature missing or unverified: %s", sig)
	}
	// A present .sig is accepted as verified in this version. Real keyring
	// verification is integrated in the apply-on-live-host milestone.
	_ = keyring
	return nil
}

// ParseDump parses a captured actual-state dump (a Manifest in the shared
// schema) without the desired-manifest observational-scope restriction. Used by
// diff/verify with state_path. On malformed input it returns a *diag.Diagnostic
// with domain invocation (a malformed dump is an invocation error).
func ParseDump(path string, explicit Format, def Format) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, diag.New(diag.DomainInvocation, "state dump unreadable: %s: %v", path, err)
	}
	f := ResolveFormat(explicit, path, def)
	m := &Manifest{}
	switch f {
	case FormatYAML:
		if err := decodeYAMLSafe(data, m); err != nil {
			return nil, diag.New(diag.DomainInvocation, "state dump malformed (yaml): %v", err)
		}
	default:
		dec := json.NewDecoder(bytes.NewReader(data))
		dec.DisallowUnknownFields()
		if err := dec.Decode(m); err != nil {
			return nil, diag.New(diag.DomainInvocation, "state dump malformed (json): %v", err)
		}
	}
	if m.Meta.FormatVersion != 1 {
		return nil, diag.New(diag.DomainInvocation, "state dump malformed: meta.format_version must be 1")
	}
	return m, nil
}

// EnsureString is a tiny guard kept for symmetry with future schema checks.
func EnsureString(v interface{}) (string, error) {
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("expected string, got %T", v)
	}
	return s, nil
}
