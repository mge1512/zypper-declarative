// generated from spec: zypper-declarative.spec.md sha256:58e1636e2de82ab81a5cd3f81d6b3c9ac6a8976e18f9abb2bbd2b2aba56fe4d4
//
// Serialisation of the Manifest data model and the canonical-model identity
// hash. JSON is canonical and Machinery-compatible. YAML is an opt-in
// serialisation of the same model, parsed under a safe profile.
package manifest

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"gopkg.in/yaml.v3"
)

// ErrUnsafeYAML signals a YAML input that requires a disabled (unsafe) safe-
// profile feature: a code-executing or arbitrary tag, unbounded alias
// expansion, or a multi-document stream.
var ErrUnsafeYAML = errors.New("YAML input requires a disabled (unsafe) feature")

// Parse parses raw bytes into a Manifest under the given format. For YAML the
// safe profile is enforced: a single document only, no executable/arbitrary
// custom tags, and explicit typing via a JSON re-decode (rejecting unknown
// fields). The schema is NOT validated here; call Validate for that.
func Parse(data []byte, format Format) (Manifest, error) {
	switch format {
	case FormatJSON:
		return parseJSON(data)
	case FormatYAML:
		return parseYAML(data)
	default:
		return Manifest{}, fmt.Errorf("unsupported format: %s", format)
	}
}

func parseJSON(data []byte) (Manifest, error) {
	var m Manifest
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&m); err != nil {
		return Manifest{}, err
	}
	// Reject trailing content (a single document only).
	if dec.More() {
		return Manifest{}, errors.New("trailing content after JSON document")
	}
	return m, nil
}

// parseYAML enforces the safe profile, then converts the YAML node tree to JSON
// and decodes it with encoding/json (DisallowUnknownFields). This gives JSON
// typing (no YAML implicit coercion such as NO->false or 1.10->float) and
// rejects YAML-only constructs.
func parseYAML(data []byte) (Manifest, error) {
	dec := yaml.NewDecoder(bytes.NewReader(data))

	var root yaml.Node
	if err := dec.Decode(&root); err != nil {
		// An empty document is not a valid manifest.
		return Manifest{}, err
	}

	// Single-document only: a second successful decode means a multi-document
	// stream, which the safe profile forbids.
	var second yaml.Node
	if err := dec.Decode(&second); err == nil {
		return Manifest{}, ErrUnsafeYAML
	}

	// Reject custom (non-core) tags anywhere in the tree: executable/arbitrary
	// tags are not permitted. Core YAML tags (map/seq/str/int/bool/null/float)
	// and untagged nodes are allowed.
	if err := rejectUnsafeTags(&root); err != nil {
		return Manifest{}, err
	}

	// Re-encode the node tree to JSON to obtain explicit (JSON) typing, then
	// decode strictly. We marshal via an intermediate value tree built from the
	// node, preserving strings as strings.
	jsonBytes, err := yamlNodeToJSON(&root)
	if err != nil {
		return Manifest{}, err
	}
	return parseJSON(jsonBytes)
}

// rejectUnsafeTags walks the YAML node tree and rejects any explicit non-core
// tag, which is how executable/arbitrary tags appear, and aliases (anchors are
// resolved by the decoder, but an alias node indicates expansion we do not
// permit under the bounded/disabled policy).
func rejectUnsafeTags(n *yaml.Node) error {
	if n == nil {
		return nil
	}
	if n.Kind == yaml.AliasNode {
		// Aliases are disabled under the safe profile.
		return ErrUnsafeYAML
	}
	switch n.Tag {
	case "",
		"!!map", "!!seq", "!!str", "!!int", "!!bool", "!!null", "!!float",
		"!!merge":
		// core tags are fine
	default:
		// Any other explicit tag (e.g. !!python/object, !!binary, custom !foo)
		// is rejected.
		return ErrUnsafeYAML
	}
	for _, c := range n.Content {
		if err := rejectUnsafeTags(c); err != nil {
			return err
		}
	}
	return nil
}

// yamlNodeToJSON converts a YAML node tree into JSON bytes, treating every scalar
// per its core tag so that explicit typing (not YAML implicit coercion) governs.
func yamlNodeToJSON(n *yaml.Node) ([]byte, error) {
	v, err := yamlNodeToValue(n)
	if err != nil {
		return nil, err
	}
	return json.Marshal(v)
}

func yamlNodeToValue(n *yaml.Node) (interface{}, error) {
	switch n.Kind {
	case yaml.DocumentNode:
		if len(n.Content) == 0 {
			return nil, errors.New("empty YAML document")
		}
		return yamlNodeToValue(n.Content[0])
	case yaml.MappingNode:
		obj := make(map[string]interface{}, len(n.Content)/2)
		for i := 0; i+1 < len(n.Content); i += 2 {
			keyNode := n.Content[i]
			valNode := n.Content[i+1]
			key := keyNode.Value
			val, err := yamlNodeToValue(valNode)
			if err != nil {
				return nil, err
			}
			obj[key] = val
		}
		return obj, nil
	case yaml.SequenceNode:
		arr := make([]interface{}, 0, len(n.Content))
		for _, c := range n.Content {
			v, err := yamlNodeToValue(c)
			if err != nil {
				return nil, err
			}
			arr = append(arr, v)
		}
		return arr, nil
	case yaml.ScalarNode:
		return scalarToValue(n)
	case yaml.AliasNode:
		return nil, ErrUnsafeYAML
	default:
		return nil, fmt.Errorf("unsupported YAML node kind: %d", n.Kind)
	}
}

// scalarToValue maps a YAML scalar to a Go value using its core tag, so typing
// is explicit and JSON-shaped rather than YAML-implicit.
func scalarToValue(n *yaml.Node) (interface{}, error) {
	switch n.Tag {
	case "!!str", "":
		// Quoted or plain string: treat as string. (Plain scalars under our
		// schema are always strings in the relevant positions; the JSON decode
		// step then enforces the struct types.)
		if n.Tag == "!!str" {
			return n.Value, nil
		}
		// Untagged plain scalar: resolve common core scalars explicitly.
		switch n.Value {
		case "null", "~", "":
			return nil, nil
		case "true":
			return true, nil
		case "false":
			return false, nil
		}
		// Try integer, else fall back to string (no implicit float/date coercion).
		var i json.Number
		if err := json.Unmarshal([]byte(n.Value), &i); err == nil {
			// Only accept if it is purely numeric.
			if isJSONNumber(n.Value) {
				return i, nil
			}
		}
		return n.Value, nil
	case "!!int":
		var i json.Number
		if err := json.Unmarshal([]byte(n.Value), &i); err != nil {
			return nil, err
		}
		return i, nil
	case "!!float":
		var f json.Number
		if err := json.Unmarshal([]byte(n.Value), &f); err != nil {
			return nil, err
		}
		return f, nil
	case "!!bool":
		switch n.Value {
		case "true":
			return true, nil
		case "false":
			return false, nil
		default:
			return nil, fmt.Errorf("invalid bool: %q", n.Value)
		}
	case "!!null":
		return nil, nil
	default:
		return nil, ErrUnsafeYAML
	}
}

func isJSONNumber(s string) bool {
	if s == "" {
		return false
	}
	var n json.Number
	dec := json.NewDecoder(bytes.NewReader([]byte(s)))
	if err := dec.Decode(&n); err != nil {
		return false
	}
	return !dec.More()
}

// Marshal serialises a Manifest in the given format. JSON output is pretty-
// printed canonical-ish (Machinery format_version 1). YAML output is the same
// data model rendered as YAML (not Machinery-compatible).
func Marshal(m Manifest, format Format) ([]byte, error) {
	switch format {
	case FormatJSON:
		b, err := json.MarshalIndent(m, "", "  ")
		if err != nil {
			return nil, err
		}
		return append(b, '\n'), nil
	case FormatYAML:
		var buf bytes.Buffer
		enc := yaml.NewEncoder(&buf)
		enc.SetIndent(2)
		if err := enc.Encode(m); err != nil {
			return nil, err
		}
		_ = enc.Close()
		return buf.Bytes(), nil
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

// MarshalCanonicalJSON serialises a Manifest as canonical JSON for on-disk
// storage of the applied record (always JSON regardless of input format).
func MarshalCanonicalJSON(m Manifest) ([]byte, error) {
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

// CanonicalSHA256 computes desired_sha256: the SHA256 of a canonical
// serialisation of the parsed data model, format-independent. The canonical form
// has object keys sorted, scope _elements sorted by identity key, and compact
// separators, with meta.created_at and meta.desired_sha256 excluded (they are
// informational / output-only and must not affect identity).
func CanonicalSHA256(m Manifest) string {
	c := canonicalValue(m)
	b := canonicalJSONBytes(c)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// canonicalValue produces a deterministic, sorted Go value tree representing the
// identity of the manifest (excluding informational meta fields).
func canonicalValue(m Manifest) map[string]interface{} {
	out := map[string]interface{}{
		"meta": map[string]interface{}{
			"format_version": m.Meta.FormatVersion,
			"generator":      m.Meta.Generator,
		},
	}

	if m.Packages != nil {
		pkgs := append([]PackageRecord(nil), m.Packages.Elements...)
		sort.Slice(pkgs, func(i, j int) bool {
			if pkgs[i].Name != pkgs[j].Name {
				return pkgs[i].Name < pkgs[j].Name
			}
			return pkgs[i].Arch < pkgs[j].Arch
		})
		els := make([]interface{}, 0, len(pkgs))
		for _, p := range pkgs {
			els = append(els, map[string]interface{}{
				"name": p.Name, "version": p.Version, "release": p.Release, "arch": p.Arch,
			})
		}
		out["packages"] = scopeMap(m.Packages.Attributes, els)
	}
	if m.Repositories != nil {
		rs := append([]RepositoryRecord(nil), m.Repositories.Elements...)
		sort.Slice(rs, func(i, j int) bool { return rs[i].Alias < rs[j].Alias })
		els := make([]interface{}, 0, len(rs))
		for _, r := range rs {
			els = append(els, map[string]interface{}{
				"alias": r.Alias, "name": r.Name, "url": r.URL, "type": r.Type,
				"enabled": r.Enabled, "gpgcheck": r.GPGCheck,
				"autorefresh": r.Autorefresh, "priority": r.Priority,
			})
		}
		out["repositories"] = scopeMap(m.Repositories.Attributes, els)
	}
	if m.Services != nil {
		ss := append([]ServiceRecord(nil), m.Services.Elements...)
		sort.Slice(ss, func(i, j int) bool { return ss[i].Name < ss[j].Name })
		els := make([]interface{}, 0, len(ss))
		for _, s := range ss {
			els = append(els, map[string]interface{}{"name": s.Name, "state": s.State})
		}
		out["services"] = scopeMap(m.Services.Attributes, els)
	}
	if m.ConfigFiles != nil {
		fs := append([]ManagedFileRecord(nil), m.ConfigFiles.Elements...)
		sort.Slice(fs, func(i, j int) bool { return fs[i].Name < fs[j].Name })
		els := make([]interface{}, 0, len(fs))
		for _, f := range fs {
			els = append(els, map[string]interface{}{
				"name": f.Name, "type": f.Type, "mode": f.Mode, "user": f.User,
				"group": f.Group, "sha256": f.SHA256, "content_ref": f.ContentRef,
				"package_name": f.PackageName,
			})
		}
		out["config_files"] = scopeMap(m.ConfigFiles.Attributes, els)
	}
	return out
}

func scopeMap(attrs ScopeAttributes, els []interface{}) map[string]interface{} {
	var a interface{}
	if attrs != nil {
		a = map[string]interface{}(attrs)
	} else {
		a = nil
	}
	return map[string]interface{}{
		"_attributes": a,
		"_elements":   els,
	}
}

// canonicalJSONBytes marshals a value with sorted keys and compact separators.
// encoding/json already sorts map keys, so a compact Marshal is canonical.
func canonicalJSONBytes(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}
