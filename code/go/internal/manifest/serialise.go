// generated from spec: zypper-declarative.spec.md sha256:27aee8e374eb3507189bad0b78339109d0116e6a13b55ae4df2ba9a18e769fc4
//
// resolve-format and (de)serialisation of the Manifest model in JSON and YAML.
// resolve-format is the single authority for choosing a serialisation; every
// read and write routes through ResolveFormat.
package manifest

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	yaml "gopkg.in/yaml.v3"
)

// ResolveFormat implements BEHAVIOR/INTERNAL: resolve-format.
//
// explicit: the format= option text (or "" if not given).
// path:     the operative file path (or "" for stdin/stdout).
// def:      the manifest-format CONFIG default.
//
// An explicit format always wins; else a recognised extension decides; else the
// default. An explicit but unknown format value is an error.
func ResolveFormat(explicit, path string, def Format) (Format, error) {
	if explicit != "" {
		switch explicit {
		case "json":
			return FormatJSON, nil
		case "yaml":
			return FormatYAML, nil
		default:
			return "", fmt.Errorf("unknown format value %q", explicit)
		}
	}
	if path != "" {
		switch strings.ToLower(filepath.Ext(path)) {
		case ".json":
			return FormatJSON, nil
		case ".yaml", ".yml":
			return FormatYAML, nil
		}
	}
	return def, nil
}

// MarshalJSON serialises the manifest as canonical (pretty) JSON suitable for
// describe output and the on-disk applied record. Scopes are sorted by identity
// for determinism (see SortScopes).
func MarshalJSON(m *Manifest) ([]byte, error) {
	c := m.cloneSorted()
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(c); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// MarshalYAML serialises the manifest as YAML representing the identical data
// model. YAML output is not Machinery-compatible.
func MarshalYAML(m *Manifest) ([]byte, error) {
	c := m.cloneSorted()
	// Route through JSON tags by converting to a generic map first so the
	// underscore_style keys (which carry json tags) are honoured uniformly.
	jb, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}
	var generic interface{}
	dec := json.NewDecoder(bytes.NewReader(jb))
	dec.UseNumber()
	if err := dec.Decode(&generic); err != nil {
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

// Marshal serialises the manifest in the resolved format.
func Marshal(m *Manifest, f Format) ([]byte, error) {
	switch f {
	case FormatYAML:
		return MarshalYAML(m)
	default:
		return MarshalJSON(m)
	}
}

// CanonicalBytes returns the canonical compact JSON serialisation of the parsed
// data model used for the identity hash: keys sorted (encoding/json sorts map
// keys; struct field order is canonical), compact separators, scopes sorted by
// identity, and meta.created_at and meta.desired_sha256 excluded (informational
// / set elsewhere) so the hash depends only on the declared intent.
func CanonicalBytes(m *Manifest) ([]byte, error) {
	c := m.cloneSorted()
	// Neutralise non-identity meta fields so JSON and YAML of the same manifest
	// (and re-serialisations with a different created_at) hash equal.
	c.Meta.CreatedAt = ""
	c.Meta.DesiredSHA256 = ""
	c.Meta.Generator = ""
	// Convert to a generic structure and re-encode with sorted keys + compact
	// separators for a stable canonical form.
	jb, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}
	var generic interface{}
	dec := json.NewDecoder(bytes.NewReader(jb))
	dec.UseNumber()
	if err := dec.Decode(&generic); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(generic); err != nil { // encoding/json sorts object keys
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}
