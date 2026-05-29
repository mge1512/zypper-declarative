// generated from spec: zypper-declarative.spec.md sha256:714e75ff672557d2c7344736a5f36b52afa37e0f565b07de83e1b18cc4492014
//
// Serialisation, parsing, schema validation, and the canonical-model identity
// hash for the manifest data model.
package manifest

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Parse decodes raw bytes in the given format into a Manifest. For YAML the
// safe profile is enforced (load-desired-manifest STEP 3): a non-code-executing
// loader, no executable/arbitrary tags, single document only, bounded alias
// expansion, and explicit typing (achieved by routing YAML through JSON typing
// with unknown-field rejection). The unsafe bool reports whether a YAML input
// required a disabled feature (so the caller maps it to a manifest error rather
// than an invocation error).
func Parse(raw []byte, format Format) (m Manifest, unsafe bool, err error) {
	switch format {
	case FormatJSON:
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&m); err != nil {
			return Manifest{}, false, fmt.Errorf("invalid JSON manifest: %w", err)
		}
		// Reject trailing content (multiple JSON documents).
		if dec.More() {
			return Manifest{}, false, fmt.Errorf("invalid JSON manifest: trailing data after document")
		}
		return m, false, nil
	case FormatYAML:
		return parseYAMLSafe(raw)
	default:
		return Manifest{}, false, fmt.Errorf("unknown format value %q", format)
	}
}

// parseYAMLSafe enforces the YAML safe profile and then decodes via JSON typing.
func parseYAMLSafe(raw []byte) (Manifest, bool, error) {
	// Single-document streams only: reject multi-document YAML.
	dec := yaml.NewDecoder(bytes.NewReader(raw))
	var node yaml.Node
	if err := dec.Decode(&node); err != nil {
		return Manifest{}, true, fmt.Errorf("unsafe or malformed YAML: %w", err)
	}
	var probe yaml.Node
	if err := dec.Decode(&probe); err == nil {
		// A second document decoded successfully: multi-document stream.
		return Manifest{}, true, fmt.Errorf("YAML multi-document streams are not permitted (safe profile)")
	}

	// No executable/arbitrary tags, no anchors/aliases (bounded == disabled here).
	if err := rejectUnsafeNodes(&node, 0); err != nil {
		return Manifest{}, true, err
	}

	// Explicit typing per the schema: marshal the safe node to JSON-equivalent
	// generic structure, then decode with JSON typing and unknown-field
	// rejection. This avoids YAML implicit coercions (NO->false, 1.10->float)
	// influencing the typed model and rejects YAML-only constructs.
	var generic interface{}
	if err := node.Decode(&generic); err != nil {
		return Manifest{}, true, fmt.Errorf("unsafe or malformed YAML: %w", err)
	}
	jsonBytes, err := json.Marshal(generic)
	if err != nil {
		return Manifest{}, true, fmt.Errorf("YAML not representable under safe profile: %w", err)
	}
	jdec := json.NewDecoder(bytes.NewReader(jsonBytes))
	jdec.DisallowUnknownFields()
	var m Manifest
	if err := jdec.Decode(&m); err != nil {
		return Manifest{}, false, fmt.Errorf("invalid manifest: %w", err)
	}
	return m, false, nil
}

// rejectUnsafeNodes walks a YAML node tree and rejects executable/arbitrary
// tags and any anchor or alias use (unbounded alias expansion defence). depth
// bounds recursion against pathological documents.
func rejectUnsafeNodes(n *yaml.Node, depth int) error {
	if depth > 64 {
		return fmt.Errorf("YAML nesting too deep (safe profile bound exceeded)")
	}
	if n == nil {
		return nil
	}
	if n.Kind == yaml.AliasNode || n.Anchor != "" {
		return fmt.Errorf("YAML anchors/aliases are not permitted (safe profile)")
	}
	// Allow only the standard core/JSON-schema tags and the implicit map/seq.
	switch n.Tag {
	case "", "!!str", "!!int", "!!float", "!!bool", "!!null", "!!map", "!!seq",
		"!!merge", "!!timestamp":
		// acceptable; "!!merge" is rejected indirectly via anchors above.
	default:
		if strings.HasPrefix(n.Tag, "!!") || strings.HasPrefix(n.Tag, "!") {
			return fmt.Errorf("YAML tag %q is not permitted (safe profile)", n.Tag)
		}
	}
	for _, c := range n.Content {
		if err := rejectUnsafeNodes(c, depth+1); err != nil {
			return err
		}
	}
	return nil
}

// Validate checks a parsed Manifest against the schema (load-desired-manifest
// STEP 4): meta.format_version must be 1 and every present scope must conform
// to its ScopeWrapper record type. Returns the first violation message.
func Validate(m *Manifest) error {
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
			switch s.State {
			case "enabled", "disabled", "masked":
			default:
				return fmt.Errorf("services._elements[%d].state must be enabled|disabled|masked, got %q", i, s.State)
			}
		}
	}
	if m.ConfigFiles != nil {
		for i, f := range m.ConfigFiles.Elements {
			if !strings.HasPrefix(f.Name, "/etc/") {
				return fmt.Errorf("config_files._elements[%d].name must be an absolute /etc path, got %q", i, f.Name)
			}
			switch f.Type {
			case "file", "link", "dir":
			default:
				return fmt.Errorf("config_files._elements[%d].type must be file|link|dir, got %q", i, f.Type)
			}
		}
	}
	return nil
}

// MarshalJSON renders a Manifest as pretty-printed canonical JSON for on-disk
// or stdout use (Machinery-compatible). Scope elements are sorted by identity
// for determinism.
func MarshalJSON(m *Manifest) ([]byte, error) {
	c := canonicalClone(m)
	return json.MarshalIndent(&c, "", "  ")
}

// MarshalYAML renders a Manifest as YAML of the identical data model.
func MarshalYAML(m *Manifest) ([]byte, error) {
	c := canonicalClone(m)
	return yaml.Marshal(&c)
}

// CanonicalHash computes desired_sha256: the SHA256 of the canonical JSON
// serialisation of the parsed data model, format-independent. The meta block is
// excluded so the hash depends only on the declared scopes (the intent), and
// elements are sorted by identity so JSON and YAML expressions of the same
// manifest hash identically.
func CanonicalHash(m *Manifest) string {
	c := canonicalClone(m)
	c.Meta = ManifestMeta{}  // exclude informational/identity meta from the hash
	b, _ := json.Marshal(&c) // compact, sorted keys via struct field order
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// canonicalClone returns a copy of m with each present scope's _elements sorted
// by its identity key, leaving absent scopes absent.
func canonicalClone(m *Manifest) Manifest {
	c := *m
	if m.Packages != nil {
		els := append([]PackageRecord(nil), m.Packages.Elements...)
		sort.Slice(els, func(i, j int) bool {
			if els[i].Name != els[j].Name {
				return els[i].Name < els[j].Name
			}
			return els[i].Arch < els[j].Arch
		})
		cp := *m.Packages
		cp.Elements = els
		c.Packages = &cp
	}
	if m.Repositories != nil {
		els := append([]RepositoryRecord(nil), m.Repositories.Elements...)
		sort.Slice(els, func(i, j int) bool { return els[i].Alias < els[j].Alias })
		cp := *m.Repositories
		cp.Elements = els
		c.Repositories = &cp
	}
	if m.Services != nil {
		els := append([]ServiceRecord(nil), m.Services.Elements...)
		sort.Slice(els, func(i, j int) bool { return els[i].Name < els[j].Name })
		cp := *m.Services
		cp.Elements = els
		c.Services = &cp
	}
	if m.ConfigFiles != nil {
		els := append([]ManagedFileRecord(nil), m.ConfigFiles.Elements...)
		sort.Slice(els, func(i, j int) bool { return els[i].Name < els[j].Name })
		cp := *m.ConfigFiles
		cp.Elements = els
		c.ConfigFiles = &cp
	}
	return c
}
