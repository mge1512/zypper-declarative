// generated from spec: zypper-declarative.spec.md sha256:f2cc80627e483a48bb8411d297711bc5f6c6e74c28dbf0dafc8fe7bd8817251e
//
// Serialisation edges: JSON and YAML encode/decode of the data model, the
// resolve-format authority, and the canonical-model identity hash.
package manifest

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// Format is the manifest serialisation.
type Format string

const (
	FormatJSON Format = "json"
	FormatYAML Format = "yaml"
)

// ErrUnknownFormat is returned by ParseFormat for an unrecognised value.
var ErrUnknownFormat = errors.New("unknown format value")

// ParseFormat validates an explicit format= value.
func ParseFormat(s string) (Format, error) {
	switch s {
	case "json":
		return FormatJSON, nil
	case "yaml":
		return FormatYAML, nil
	default:
		return "", fmt.Errorf("%w: %q", ErrUnknownFormat, s)
	}
}

// ResolveFormat is the single authority for choosing a serialisation.
//
//  1. If explicit is non-empty, return it (an explicit format= always wins).
//  2. Else if path has a recognised extension, return json for .json and yaml
//     for .yaml/.yml.
//  3. Else return the manifest-format CONFIG default (configDefault).
//
// explicit must already be a validated Format ("" means none given).
func ResolveFormat(explicit Format, path string, configDefault Format) Format {
	if explicit != "" {
		return explicit
	}
	if path != "" {
		lower := strings.ToLower(path)
		switch {
		case strings.HasSuffix(lower, ".json"):
			return FormatJSON
		case strings.HasSuffix(lower, ".yaml"), strings.HasSuffix(lower, ".yml"):
			return FormatYAML
		}
	}
	if configDefault == "" {
		return FormatJSON
	}
	return configDefault
}

// MarshalJSONPretty serialises the manifest as indented JSON for on-disk and
// stdout readability. It remains Machinery-compatible.
func (m *Manifest) MarshalJSONPretty() ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(m); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Serialise renders the manifest in the resolved format.
func (m *Manifest) Serialise(f Format) ([]byte, error) {
	switch f {
	case FormatYAML:
		return marshalYAML(m)
	default:
		return m.MarshalJSONPretty()
	}
}

// DesiredSHA256 is the SHA256 of the canonical JSON serialisation of the parsed
// data model: object keys sorted, compact separators, scope _elements sorted by
// identity key. It is format-independent, so the same intent expressed in JSON
// or YAML yields the same value.
func (m *Manifest) DesiredSHA256() (string, error) {
	canon := m.canonical()
	b, err := canonicalJSON(canon)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

// canonical produces a deep copy with _elements sorted by identity and the meta
// fields that do not participate in identity (generator, created_at,
// desired_sha256) zeroed, so the hash depends only on the declarable intent.
func (m *Manifest) canonical() *Manifest {
	c := &Manifest{
		Meta: ManifestMeta{FormatVersion: m.Meta.FormatVersion},
	}
	if m.Packages != nil {
		els := append([]PackageRecord(nil), m.Packages.Elements...)
		sort.Slice(els, func(i, j int) bool {
			if els[i].Name != els[j].Name {
				return els[i].Name < els[j].Name
			}
			return els[i].Arch < els[j].Arch
		})
		c.Packages = &PackagesScope{Attributes: m.Packages.Attributes, Elements: els}
	}
	if m.Repositories != nil {
		els := append([]RepositoryRecord(nil), m.Repositories.Elements...)
		sort.Slice(els, func(i, j int) bool { return els[i].Alias < els[j].Alias })
		c.Repositories = &RepositoriesScope{Attributes: m.Repositories.Attributes, Elements: els}
	}
	if m.Services != nil {
		els := append([]ServiceRecord(nil), m.Services.Elements...)
		sort.Slice(els, func(i, j int) bool { return els[i].Name < els[j].Name })
		c.Services = &ServicesScope{Attributes: m.Services.Attributes, Elements: els}
	}
	if m.ConfigFiles != nil {
		els := append([]ManagedFileRecord(nil), m.ConfigFiles.Elements...)
		sort.Slice(els, func(i, j int) bool { return els[i].Name < els[j].Name })
		c.ConfigFiles = &ConfigFilesScope{Attributes: m.ConfigFiles.Attributes, Elements: els}
	}
	return c
}

// canonicalJSON marshals v with map keys sorted (encoding/json already sorts map
// keys) and struct fields in declaration order, compact. encoding/json is
// deterministic for our struct/map shapes.
func canonicalJSON(v interface{}) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return b, nil
}
