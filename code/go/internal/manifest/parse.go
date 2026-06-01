// generated from spec: zypper-declarative.spec.md sha256:27aee8e374eb3507189bad0b78339109d0116e6a13b55ae4df2ba9a18e769fc4
//
// Parsing and validation of the Manifest model from JSON and YAML, plus
// BEHAVIOR/INTERNAL: load-desired-manifest. YAML is parsed under a safe profile:
// a non-code-executing loader (gopkg.in/yaml.v3 does not execute tags), a single
// document only (multi-document streams rejected), bounded alias handling, and
// explicit typing per the schema (we route YAML through a typed decode so values
// are not implicitly coerced). A YAML input requiring any disabled feature is a
// manifest error.
package manifest

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"

	yaml "gopkg.in/yaml.v3"
)

// ErrKind categorises a parse/validation failure so callers can map it to an
// exit code and a Diagnostic domain.
type ErrKind int

const (
	// ErrInvocation: file unreadable / unknown format value / malformed dump.
	ErrInvocation ErrKind = iota
	// ErrManifest: schema violation / unsafe YAML / observational scope present
	// in a desired manifest / signature failure.
	ErrManifest
)

// ParseError carries a kind and message.
type ParseError struct {
	Kind    ErrKind
	Message string
}

func (e *ParseError) Error() string { return e.Message }

func invocationErr(format string, a ...interface{}) *ParseError {
	return &ParseError{Kind: ErrInvocation, Message: fmt.Sprintf(format, a...)}
}

func manifestErr(format string, a ...interface{}) *ParseError {
	return &ParseError{Kind: ErrManifest, Message: fmt.Sprintf(format, a...)}
}

// ParseOptions controls how a manifest document is parsed and validated.
type ParseOptions struct {
	// Format is the resolved serialisation.
	Format Format
	// AllowObservational permits non-empty observational scopes. true for an
	// actual-state dump (verify/diff state-path); false for a desired manifest.
	AllowObservational bool
}

// Parse decodes raw bytes into a Manifest under the given options and validates
// it against the schema.
func Parse(data []byte, opts ParseOptions) (*Manifest, error) {
	var jsonBytes []byte
	switch opts.Format {
	case FormatYAML:
		jb, err := yamlToJSONSafe(data)
		if err != nil {
			return nil, err
		}
		jsonBytes = jb
	default:
		jsonBytes = data
	}

	m, err := decodeStrictJSON(jsonBytes)
	if err != nil {
		return nil, err
	}
	if err := validate(m, opts.AllowObservational); err != nil {
		return nil, err
	}
	return m, nil
}

// decodeStrictJSON decodes JSON into a Manifest rejecting unknown fields, so a
// structurally-wrong document is a clear error rather than a silently-dropped
// field. A decode failure is a malformed/invocation error.
func decodeStrictJSON(data []byte) (*Manifest, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var m Manifest
	if err := dec.Decode(&m); err != nil {
		return nil, invocationErr("malformed manifest: %v", err)
	}
	// Reject trailing content (must be a single JSON value).
	if dec.More() {
		return nil, invocationErr("malformed manifest: trailing data after JSON value")
	}
	return &m, nil
}

// yamlToJSONSafe converts a YAML document to JSON under the safe profile and
// returns a manifest error if any disabled feature is required.
func yamlToJSONSafe(data []byte) ([]byte, error) {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	var doc interface{}
	if err := dec.Decode(&doc); err != nil {
		if err == io.EOF {
			return nil, manifestErr("manifest invalid: empty YAML document")
		}
		return nil, manifestErr("manifest invalid: unsafe or malformed YAML: %v", err)
	}
	// Single-document streams only: a second document is a disabled feature.
	var extra interface{}
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			return nil, manifestErr("manifest invalid: multi-document YAML streams are not permitted (safe profile)")
		}
		return nil, manifestErr("manifest invalid: unsafe or malformed YAML: %v", err)
	}
	// yaml.v3 surfaces map keys as map[string]interface{} for string keys; reject
	// non-string keys (a typing the schema never uses) for explicit typing.
	norm, err := normaliseYAML(doc)
	if err != nil {
		return nil, err
	}
	jb, err := json.Marshal(norm)
	if err != nil {
		return nil, manifestErr("manifest invalid: cannot normalise YAML: %v", err)
	}
	return jb, nil
}

// normaliseYAML converts yaml.v3's decoded structure into JSON-compatible types
// and rejects non-string mapping keys.
func normaliseYAML(v interface{}) (interface{}, error) {
	switch t := v.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(t))
		for k, val := range t {
			nv, err := normaliseYAML(val)
			if err != nil {
				return nil, err
			}
			out[k] = nv
		}
		return out, nil
	case map[interface{}]interface{}:
		out := make(map[string]interface{}, len(t))
		for k, val := range t {
			ks, ok := k.(string)
			if !ok {
				return nil, manifestErr("manifest invalid: non-string mapping key in YAML")
			}
			nv, err := normaliseYAML(val)
			if err != nil {
				return nil, err
			}
			out[ks] = nv
		}
		return out, nil
	case []interface{}:
		out := make([]interface{}, len(t))
		for i, val := range t {
			nv, err := normaliseYAML(val)
			if err != nil {
				return nil, err
			}
			out[i] = nv
		}
		return out, nil
	default:
		return v, nil
	}
}

// validate checks the schema: meta.format_version must be 1; the observational
// scopes must not be present non-empty in a desired manifest. Other structural
// conformance is enforced by the typed decode.
func validate(m *Manifest, allowObservational bool) error {
	if m.Meta.FormatVersion != 1 {
		return manifestErr("manifest invalid: meta.format_version = %d, must be 1", m.Meta.FormatVersion)
	}
	if !allowObservational {
		if m.ChangedManagedFiles != nil && len(m.ChangedManagedFiles.Elements) > 0 {
			return manifestErr("manifest invalid: desired manifest carries a non-empty changed_managed_files (observational) scope")
		}
		if m.UnmanagedFiles != nil && len(m.UnmanagedFiles.Elements) > 0 {
			return manifestErr("manifest invalid: desired manifest carries a non-empty unmanaged_files (observational) scope")
		}
	}
	// Drop empty/absent observational scopes from a desired manifest (tolerated).
	if !allowObservational {
		m.ChangedManagedFiles = nil
		m.UnmanagedFiles = nil
	}
	return nil
}

// DesiredSHA256 computes the canonical-model identity hash of a manifest.
func DesiredSHA256(m *Manifest) (string, error) {
	b, err := CanonicalBytes(m)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}
