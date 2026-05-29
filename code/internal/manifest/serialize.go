// generated from spec: zypper-declarative.spec.md sha256:f8ff76ecbc4bbc69a49e2e32b2924da3a64df1ad46196e05ce8c137b684429b2
//
// Serialisation edges: encoding the in-memory model to JSON or YAML, and the
// safe-profile YAML decoder. YAML is decoded by first inspecting the node tree
// for unsafe features (executable/arbitrary tags, unbounded alias expansion,
// multiple documents), then re-encoding the safe tree to JSON and decoding that
// with encoding/json so that JSON typing applies (no YAML implicit coercion such
// as NO -> false or 1.10 -> float).

package manifest

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"gopkg.in/yaml.v3"
)

// maxAliasNodes bounds anchor/alias expansion: a document with more alias nodes
// than this is rejected as an alias-expansion denial-of-service vector.
const maxAliasNodes = 64

// ErrUnsafeYAML reports a YAML input that requires a disabled (unsafe) feature.
type ErrUnsafeYAML struct {
	Reason string
}

func (e *ErrUnsafeYAML) Error() string { return "unsafe YAML: " + e.Reason }

// Encode serialises the manifest in the resolved format.
func Encode(m *Manifest, f Format) ([]byte, error) {
	switch f {
	case FormatJSON:
		return m.MarshalCanonicalJSON()
	case FormatYAML:
		return encodeYAML(m)
	default:
		return nil, &ErrUnknownFormat{Value: string(f)}
	}
}

// encodeYAML renders the same data model as YAML, going through canonical JSON
// first so the YAML mirrors the JSON shape exactly (underscore_style keys, scope
// idiom). The output is deterministic.
func encodeYAML(m *Manifest) ([]byte, error) {
	j, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	var generic interface{}
	if err := json.Unmarshal(j, &generic); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(generic); err != nil {
		return nil, err
	}
	_ = enc.Close()
	return buf.Bytes(), nil
}

// Decode parses bytes in the given format into the data model. JSON is parsed
// directly; YAML is parsed under the safe profile.
func Decode(data []byte, f Format) (*Manifest, error) {
	switch f {
	case FormatJSON:
		return decodeJSON(data)
	case FormatYAML:
		return decodeYAMLSafe(data)
	default:
		return nil, &ErrUnknownFormat{Value: string(f)}
	}
}

func decodeJSON(data []byte) (*Manifest, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var m Manifest
	if err := dec.Decode(&m); err != nil {
		return nil, err
	}
	return &m, nil
}

// decodeYAMLSafe enforces the safe profile, then routes through JSON typing.
func decodeYAMLSafe(data []byte) (*Manifest, error) {
	dec := yaml.NewDecoder(bytes.NewReader(data))

	var doc yaml.Node
	if err := dec.Decode(&doc); err != nil {
		return nil, &ErrUnsafeYAML{Reason: "could not parse YAML document: " + err.Error()}
	}

	// Reject multi-document streams: a second successful decode means >1 doc.
	var extra yaml.Node
	err := dec.Decode(&extra)
	switch {
	case err == nil:
		return nil, &ErrUnsafeYAML{Reason: "multi-document streams are not permitted (single document only)"}
	case errors.Is(err, io.EOF):
		// Exactly one document: the expected case.
	default:
		return nil, &ErrUnsafeYAML{Reason: "trailing content after the first document: " + err.Error()}
	}

	aliasCount := 0
	if err := inspectNode(&doc, &aliasCount); err != nil {
		return nil, err
	}

	// Re-encode the inspected (safe) node to JSON via a generic decode, then
	// decode with encoding/json so JSON typing rules apply.
	var generic interface{}
	if err := doc.Decode(&generic); err != nil {
		return nil, &ErrUnsafeYAML{Reason: "could not realise YAML values: " + err.Error()}
	}
	j, err := json.Marshal(generic)
	if err != nil {
		return nil, &ErrUnsafeYAML{Reason: "could not convert YAML to JSON: " + err.Error()}
	}
	return decodeJSON(j)
}

// inspectNode walks the YAML node tree enforcing the safe profile: only standard
// tags, and a bounded number of alias nodes.
func inspectNode(n *yaml.Node, aliasCount *int) error {
	if n == nil {
		return nil
	}
	if n.Kind == yaml.AliasNode {
		*aliasCount++
		if *aliasCount > maxAliasNodes {
			return &ErrUnsafeYAML{Reason: fmt.Sprintf("alias expansion exceeds bound (%d)", maxAliasNodes)}
		}
	}
	if !tagAllowed(n.Tag) {
		return &ErrUnsafeYAML{Reason: "disallowed tag " + n.Tag + " (no executable or arbitrary tags)"}
	}
	for _, c := range n.Content {
		if err := inspectNode(c, aliasCount); err != nil {
			return err
		}
	}
	return nil
}

// tagAllowed permits only the standard YAML core-schema tags and the empty
// (resolved-by-context) tag. Any custom, executable, or library-specific tag
// (e.g. !!python/object, !!binary, !!merge with arbitrary effect) is rejected.
func tagAllowed(tag string) bool {
	switch tag {
	case "",
		"!!str", "!!int", "!!float", "!!bool", "!!null",
		"!!map", "!!seq":
		return true
	default:
		return false
	}
}
