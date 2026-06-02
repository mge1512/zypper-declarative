// generated from spec: zypper-declarative.spec.md sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
//
// Manifest parsing, schema validation, serialisation, and canonical-model hashing.
package manifest

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	yaml "gopkg.in/yaml.v3"
)

// ParseError is a schema/parse failure with a domain so the caller can map it to
// the correct exit code and diagnostic.
type ParseError struct {
	Domain  string // "invocation" (malformed/unparseable) or "manifest" (schema/safety)
	Message string
}

func (e *ParseError) Error() string { return e.Message }

// Allowed YAML tags under the safe profile: only the YAML core schema tags plus
// the implicit empty tag the parser assigns before resolution.
var safeYAMLTags = map[string]bool{
	"":                        true,
	"!!str":                   true,
	"!!int":                   true,
	"!!float":                 true,
	"!!bool":                  true,
	"!!null":                  true,
	"!!map":                   true,
	"!!seq":                   true,
	"tag:yaml.org,2002:str":   true,
	"tag:yaml.org,2002:int":   true,
	"tag:yaml.org,2002:float": true,
	"tag:yaml.org,2002:bool":  true,
	"tag:yaml.org,2002:null":  true,
	"tag:yaml.org,2002:map":   true,
	"tag:yaml.org,2002:seq":   true,
}

// enforceSafeYAML walks a yaml.Node tree and rejects any feature outside the safe
// profile: non-core (executable/arbitrary) tags, and anchors/aliases (bounded or
// disabled alias expansion -> we disable them entirely).
func enforceSafeYAML(n *yaml.Node) error {
	if n == nil {
		return nil
	}
	if n.Alias != nil || n.Kind == yaml.AliasNode {
		return &ParseError{Domain: "manifest", Message: "manifest error: YAML alias expansion is not permitted under the safe profile"}
	}
	if n.Anchor != "" {
		return &ParseError{Domain: "manifest", Message: "manifest error: YAML anchors are not permitted under the safe profile"}
	}
	switch n.Kind {
	case yaml.ScalarNode, yaml.MappingNode, yaml.SequenceNode:
		if n.Tag != "" && !safeYAMLTags[n.Tag] {
			return &ParseError{Domain: "manifest", Message: fmt.Sprintf("manifest error: unsafe or unsupported YAML tag %q rejected under the safe profile", n.Tag)}
		}
	}
	for _, c := range n.Content {
		if err := enforceSafeYAML(c); err != nil {
			return err
		}
	}
	return nil
}

// decodeYAMLSafe parses a single-document YAML stream under the safe profile and
// returns the equivalent JSON bytes for downstream strict JSON decoding.
func decodeYAMLSafe(data []byte) ([]byte, error) {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	var doc yaml.Node
	if err := dec.Decode(&doc); err != nil {
		return nil, &ParseError{Domain: "invocation", Message: "malformed YAML document: " + err.Error()}
	}
	// Reject multi-document streams.
	var extra yaml.Node
	if err := dec.Decode(&extra); err == nil {
		return nil, &ParseError{Domain: "manifest", Message: "manifest error: multi-document YAML streams are not permitted under the safe profile"}
	} else if !errors.Is(err, errEOF) && err.Error() != "EOF" {
		// any non-EOF here is a genuine parse error on the second document
		return nil, &ParseError{Domain: "manifest", Message: "manifest error: multi-document YAML streams are not permitted under the safe profile"}
	}
	if err := enforceSafeYAML(&doc); err != nil {
		return nil, err
	}
	// Convert the safe node tree into a generic value, then to JSON.
	var v interface{}
	if err := doc.Decode(&v); err != nil {
		return nil, &ParseError{Domain: "invocation", Message: "malformed YAML document: " + err.Error()}
	}
	jb, err := json.Marshal(v)
	if err != nil {
		return nil, &ParseError{Domain: "invocation", Message: "could not normalise YAML document: " + err.Error()}
	}
	return jb, nil
}

// errEOF is the yaml decoder's end-of-stream sentinel surfaced via errors.Is.
var errEOF = errors.New("EOF")

// Parse parses raw bytes in the given format into a Manifest, without schema
// validation. Use Load for the validated read path.
func Parse(data []byte, f Format) (*Manifest, error) {
	jb := data
	if f == FormatYAML {
		converted, err := decodeYAMLSafe(data)
		if err != nil {
			return nil, err
		}
		jb = converted
	}
	dec := json.NewDecoder(bytes.NewReader(jb))
	var m Manifest
	if err := dec.Decode(&m); err != nil {
		return nil, &ParseError{Domain: "invocation", Message: "malformed manifest document: " + err.Error()}
	}
	return &m, nil
}

// Validate checks the schema rules a desired manifest / shared document must obey
// (BEHAVIOR/INTERNAL: load-desired-manifest, step 4).
//
//	rejectObservational: when true (a desired manifest), a non-empty observational
//	scope is a manifest error; an empty/absent one is tolerated and dropped.
func (m *Manifest) Validate(rejectObservational bool) error {
	if m.Meta.FormatVersion != 1 {
		return &ParseError{Domain: "manifest", Message: fmt.Sprintf("manifest error: meta.format_version must be 1, got %d", m.Meta.FormatVersion)}
	}
	// Validate declarable scope records minimally per their refinement predicates.
	if m.ConfigFiles != nil {
		for _, e := range m.ConfigFiles.Elements {
			if e.Name == "" {
				return &ParseError{Domain: "manifest", Message: "manifest error: config_files record has empty name"}
			}
			switch e.Type {
			case "file", "link", "dir":
			default:
				return &ParseError{Domain: "manifest", Message: fmt.Sprintf("manifest error: config_files %q has invalid type %q", e.Name, e.Type)}
			}
		}
	}
	if m.Services != nil {
		for _, e := range m.Services.Elements {
			switch e.State {
			case "enabled", "disabled", "masked":
			default:
				return &ParseError{Domain: "manifest", Message: fmt.Sprintf("manifest error: service %q has invalid state %q", e.Name, e.State)}
			}
		}
	}
	if rejectObservational {
		if m.ChangedManagedFiles != nil && len(m.ChangedManagedFiles.Elements) > 0 {
			return &ParseError{Domain: "manifest", Message: "manifest error: a desired manifest must not carry a non-empty changed_managed_files scope"}
		}
		if m.UnmanagedFiles != nil && len(m.UnmanagedFiles.Elements) > 0 {
			return &ParseError{Domain: "manifest", Message: "manifest error: a desired manifest must not carry a non-empty unmanaged_files scope"}
		}
		// Drop empty/absent observational scopes.
		m.ChangedManagedFiles = nil
		m.UnmanagedFiles = nil
	}
	return nil
}

// MarshalJSON serialises the manifest as canonical, pretty-printed JSON suitable
// for describe/applied output (Machinery-compatible). Keys are ordered by struct
// definition; elements are sorted by identity for determinism.
func (m *Manifest) MarshalJSONIndent() ([]byte, error) {
	m.sortElements()
	return json.MarshalIndent(m, "", "  ")
}

// MarshalYAML serialises the manifest as YAML representing the same data model
// (not Machinery-compatible).
func (m *Manifest) MarshalYAML() ([]byte, error) {
	m.sortElements()
	// Round-trip through JSON so the field names (json tags) are honoured.
	jb, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	var v interface{}
	if err := json.Unmarshal(jb, &v); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	enc.Close()
	return buf.Bytes(), nil
}

// Serialise renders the manifest in the requested format.
func (m *Manifest) Serialise(f Format) ([]byte, error) {
	switch f {
	case FormatYAML:
		return m.MarshalYAML()
	default:
		b, err := m.MarshalJSONIndent()
		if err != nil {
			return nil, err
		}
		return append(b, '\n'), nil
	}
}

// sortElements sorts every scope's elements by its identity key so output is
// deterministic and the canonical hash is stable.
func (m *Manifest) sortElements() {
	if m.Packages != nil {
		sort.Slice(m.Packages.Elements, func(i, j int) bool {
			a, b := m.Packages.Elements[i], m.Packages.Elements[j]
			if a.Name != b.Name {
				return a.Name < b.Name
			}
			return a.Arch < b.Arch
		})
	}
	if m.Repositories != nil {
		sort.Slice(m.Repositories.Elements, func(i, j int) bool {
			return m.Repositories.Elements[i].Alias < m.Repositories.Elements[j].Alias
		})
	}
	if m.Services != nil {
		sort.Slice(m.Services.Elements, func(i, j int) bool {
			return m.Services.Elements[i].Name < m.Services.Elements[j].Name
		})
	}
	if m.ConfigFiles != nil {
		sort.Slice(m.ConfigFiles.Elements, func(i, j int) bool {
			return m.ConfigFiles.Elements[i].Name < m.ConfigFiles.Elements[j].Name
		})
	}
	if m.ChangedManagedFiles != nil {
		sort.Slice(m.ChangedManagedFiles.Elements, func(i, j int) bool {
			return m.ChangedManagedFiles.Elements[i].Name < m.ChangedManagedFiles.Elements[j].Name
		})
	}
	if m.UnmanagedFiles != nil {
		sort.Slice(m.UnmanagedFiles.Elements, func(i, j int) bool {
			return m.UnmanagedFiles.Elements[i].Name < m.UnmanagedFiles.Elements[j].Name
		})
	}
}

// CanonicalSHA256 computes the canonical-model identity hash (desired_sha256):
// the SHA256 of a canonical JSON serialisation of the parsed data model, with the
// volatile/identity-irrelevant meta fields (generator, created_at, desired_sha256)
// zeroed and elements sorted. The same intent in JSON or YAML yields the same hash.
func (m *Manifest) CanonicalSHA256() string {
	// Work on a shallow copy with neutralised meta so format/timestamp/hash do
	// not perturb identity. Only the declarable model contributes.
	cp := *m
	cp.Meta = ManifestMeta{FormatVersion: 1}
	cp.ChangedManagedFiles = nil
	cp.UnmanagedFiles = nil
	cp.sortElements()
	b, err := canonicalJSON(&cp)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// canonicalJSON marshals a value into compact JSON with object keys sorted
// recursively, so the byte form is stable regardless of struct field order.
func canonicalJSON(v interface{}) ([]byte, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var generic interface{}
	if err := json.Unmarshal(raw, &generic); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := writeCanonical(&buf, generic); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func writeCanonical(buf *bytes.Buffer, v interface{}) error {
	switch t := v.(type) {
	case map[string]interface{}:
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		buf.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				buf.WriteByte(',')
			}
			kb, _ := json.Marshal(k)
			buf.Write(kb)
			buf.WriteByte(':')
			if err := writeCanonical(buf, t[k]); err != nil {
				return err
			}
		}
		buf.WriteByte('}')
	case []interface{}:
		buf.WriteByte('[')
		for i, e := range t {
			if i > 0 {
				buf.WriteByte(',')
			}
			if err := writeCanonical(buf, e); err != nil {
				return err
			}
		}
		buf.WriteByte(']')
	default:
		b, err := json.Marshal(t)
		if err != nil {
			return err
		}
		buf.Write(b)
	}
	return nil
}
