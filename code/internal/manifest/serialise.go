// generated from spec: zypper-declarative.spec.md sha256:b2d0de88fbed1163678e59e931c741b9d999b71f902f6eb01db8790bb813d057
//
// Serialisation of the shared data model: canonical JSON (the interoperability
// contract, Machinery format_version 1) and an opt-in YAML serialisation under
// a safe profile. Also the format-independent canonical-model identity hash.
package manifest

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"

	yaml "gopkg.in/yaml.v3"
)

// ParseJSON parses canonical JSON bytes into the data model. Unknown fields
// are tolerated for forward compatibility with full Machinery dumps (which may
// carry extension fields the converger ignores), but observational scopes are
// retained where present and ignored by intent/convergence per the spec.
func ParseJSON(data []byte) (*Manifest, error) {
	var m Manifest
	dec := json.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(&m); err != nil {
		return nil, err
	}
	return &m, nil
}

// ParseYAML parses YAML bytes under the safe profile required by the spec and
// the decisions hints: a non-code-executing loader (no arbitrary/executable
// tags), bounded/disabled alias expansion, a single document only, and explicit
// typing per the schema (achieved by routing through JSON typing). A document
// requiring any disabled feature is rejected with an error.
func ParseYAML(data []byte) (*Manifest, error) {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	var root yaml.Node
	if err := dec.Decode(&root); err != nil {
		return nil, err
	}
	// Single-document only: a second successful decode means a multi-document
	// stream, which the safe profile rejects.
	var extra yaml.Node
	if err := dec.Decode(&extra); err == nil {
		return nil, fmt.Errorf("unsafe YAML: multi-document streams are not permitted")
	}
	if err := checkSafeYAML(&root, 0); err != nil {
		return nil, err
	}
	// Convert to a JSON-typed value, then decode through encoding/json so that
	// typing is explicit (JSON typing) rather than YAML implicit coercion.
	jv, err := yamlNodeToJSONValue(&root)
	if err != nil {
		return nil, err
	}
	buf, err := json.Marshal(jv)
	if err != nil {
		return nil, err
	}
	return ParseJSON(buf)
}

const maxYAMLNodes = 100000

// checkSafeYAML walks the node tree rejecting executable/arbitrary tags and
// aliases (bounded alias expansion is achieved by disabling aliases entirely).
func checkSafeYAML(n *yaml.Node, depth int) error {
	if n == nil {
		return nil
	}
	if depth > 1000 {
		return fmt.Errorf("unsafe YAML: document nesting too deep")
	}
	if n.Kind == yaml.AliasNode {
		return fmt.Errorf("unsafe YAML: anchors/aliases are not permitted")
	}
	// Reject any non-standard explicit tag. Only the core YAML tags and the
	// implicit (empty) tag are allowed; explicit application tags such as
	// !!python/object or any !custom tag are rejected.
	if n.Tag != "" && !isAllowedYAMLTag(n.Tag) {
		return fmt.Errorf("unsafe YAML: tag %q is not permitted", n.Tag)
	}
	for _, c := range n.Content {
		if err := checkSafeYAML(c, depth+1); err != nil {
			return err
		}
	}
	return nil
}

func isAllowedYAMLTag(tag string) bool {
	switch tag {
	case "",
		"!!map", "!!seq", "!!str", "!!int", "!!float",
		"!!bool", "!!null", "!!merge":
		return true
	}
	return false
}

// yamlNodeToJSONValue converts a safe YAML node tree to a generic value
// (map[string]interface{}, []interface{}, string/float64/bool/nil) using the
// node's resolved scalar type, so the subsequent JSON decode applies explicit
// JSON typing.
func yamlNodeToJSONValue(n *yaml.Node) (interface{}, error) {
	switch n.Kind {
	case yaml.DocumentNode:
		if len(n.Content) == 0 {
			return nil, nil
		}
		return yamlNodeToJSONValue(n.Content[0])
	case yaml.MappingNode:
		out := map[string]interface{}{}
		for i := 0; i+1 < len(n.Content); i += 2 {
			key := n.Content[i].Value
			val, err := yamlNodeToJSONValue(n.Content[i+1])
			if err != nil {
				return nil, err
			}
			out[key] = val
		}
		return out, nil
	case yaml.SequenceNode:
		out := make([]interface{}, 0, len(n.Content))
		for _, c := range n.Content {
			v, err := yamlNodeToJSONValue(c)
			if err != nil {
				return nil, err
			}
			out = append(out, v)
		}
		return out, nil
	case yaml.ScalarNode:
		return scalarToJSON(n)
	default:
		return nil, fmt.Errorf("unsafe YAML: unsupported node kind")
	}
}

func scalarToJSON(n *yaml.Node) (interface{}, error) {
	// Decode the scalar through yaml's resolver into a typed value. Because we
	// then re-encode via JSON, the values land in the JSON type system.
	var v interface{}
	if err := n.Decode(&v); err != nil {
		return nil, err
	}
	return v, nil
}

// MarshalJSON serialises a Manifest as pretty-printed JSON (Machinery
// format_version 1). Used for on-disk and stdout JSON output.
func MarshalJSON(m *Manifest) ([]byte, error) {
	return json.MarshalIndent(m, "", "  ")
}

// MarshalYAML serialises a Manifest as YAML representing the same data model.
func MarshalYAML(m *Manifest) ([]byte, error) {
	// Round-trip through JSON to obtain a generic value, then emit YAML, so the
	// YAML mirrors the JSON key names (underscore_style) exactly.
	jb, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	var generic interface{}
	if err := json.Unmarshal(jb, &generic); err != nil {
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

// Serialise renders a Manifest in the resolved format.
func Serialise(m *Manifest, f Format) ([]byte, error) {
	if f == FormatYAML {
		return MarshalYAML(m)
	}
	return MarshalJSON(m)
}

// CanonicalSHA256 computes desired_sha256: the SHA256 of a canonical JSON
// serialisation of the parsed data model. It is format-independent (same
// intent in JSON or YAML yields the same hash) and stable: object keys are
// sorted by encoding/json, scope _elements are sorted by their identity key,
// and meta fields that are informational only (generator, created_at,
// desired_sha256 itself) are excluded from the hash.
func CanonicalSHA256(m *Manifest) (string, error) {
	c := canonicalModel(m)
	buf, err := json.Marshal(c) // encoding/json sorts map keys; compact output
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(buf)
	return hex.EncodeToString(sum[:]), nil
}

// canonicalModel builds a map with sorted scope elements and only the
// hash-relevant meta field (format_version), so two equivalent manifests in
// different formats hash identically.
func canonicalModel(m *Manifest) map[string]interface{} {
	out := map[string]interface{}{
		"meta": map[string]interface{}{
			"format_version": m.Meta.FormatVersion,
		},
	}
	if m.Packages != nil {
		els := append([]PackageRecord(nil), m.Packages.Elements...)
		sort.Slice(els, func(i, j int) bool {
			if els[i].Name != els[j].Name {
				return els[i].Name < els[j].Name
			}
			return els[i].Arch < els[j].Arch
		})
		out["packages"] = scopeToCanonical(m.Packages.Attributes, els)
	}
	if m.Repositories != nil {
		els := append([]RepositoryRecord(nil), m.Repositories.Elements...)
		sort.Slice(els, func(i, j int) bool { return els[i].Alias < els[j].Alias })
		out["repositories"] = scopeToCanonical(m.Repositories.Attributes, els)
	}
	if m.Services != nil {
		els := append([]ServiceRecord(nil), m.Services.Elements...)
		sort.Slice(els, func(i, j int) bool { return els[i].Name < els[j].Name })
		out["services"] = scopeToCanonical(m.Services.Attributes, els)
	}
	if m.ConfigFiles != nil {
		els := append([]ManagedFileRecord(nil), m.ConfigFiles.Elements...)
		sort.Slice(els, func(i, j int) bool { return els[i].Name < els[j].Name })
		out["config_files"] = scopeToCanonical(m.ConfigFiles.Attributes, els)
	}
	return out
}

func scopeToCanonical(attrs map[string]interface{}, elements interface{}) map[string]interface{} {
	if attrs == nil {
		attrs = map[string]interface{}{}
	}
	// Round-trip elements through JSON so the canonical form uses the JSON key
	// names and a stable representation.
	jb, _ := json.Marshal(elements)
	var generic interface{}
	_ = json.Unmarshal(jb, &generic)
	return map[string]interface{}{
		"_attributes": attrs,
		"_elements":   generic,
	}
}
