// generated from spec: zypper-declarative.spec.md sha256:714e75ff672557d2c7344736a5f36b52afa37e0f565b07de83e1b18cc4492014
//
// Serialisation of the manifest data model: canonical JSON (Machinery
// format_version 1), YAML under a safe profile, and the format-independent
// canonical-model identity hash (desired_sha256).
package zypperdeclarative

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"gopkg.in/yaml.v3"
)

// MarshalCanonicalJSON renders a Manifest as canonical JSON: stable key
// order (Go's encoding/json sorts map keys and preserves struct field order),
// two-space indentation, no HTML escaping. The same data model always
// produces the same bytes.
func MarshalCanonicalJSON(m *Manifest) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(m); err != nil {
		return nil, err
	}
	// Drop the trailing newline encoder adds.
	b := buf.Bytes()
	if len(b) > 0 && b[len(b)-1] == '\n' {
		b = b[:len(b)-1]
	}
	return b, nil
}

// MarshalYAML renders a Manifest as YAML representing the identical data
// model. Not Machinery-compatible.
func MarshalYAML(m *Manifest) ([]byte, error) {
	return yaml.Marshal(m)
}

// canonicalModelHash computes desired_sha256: the SHA256 of the canonical
// JSON serialisation of the parsed data model with meta.desired_sha256 and
// the informational meta.created_at zeroed, so it depends only on intent and
// is format-independent.
func canonicalModelHash(m *Manifest) (string, error) {
	clone := *m
	clone.Meta.DesiredSHA256 = ""
	clone.Meta.CreatedAt = "" // informational only, not compared
	b, err := MarshalCanonicalJSON(&clone)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

// parseJSONManifest parses canonical JSON into the data model. Unknown fields
// are tolerated (a full dump may carry observational scopes/extension fields
// that the converger ignores).
func parseJSONManifest(data []byte) (*Manifest, error) {
	var m Manifest
	dec := json.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(&m); err != nil {
		return nil, err
	}
	return &m, nil
}

// parseYAMLSafe parses YAML under the safe profile required by the spec:
//   - non-code-executing loader only (no arbitrary/executable tags),
//   - bounded/disabled alias expansion (reject unbounded anchor/alias use),
//   - single document only (reject multi-document streams),
//   - explicit typing per the schema (struct-typed decode, not loose maps).
//
// A YAML input requiring any disabled feature returns a non-nil error.
func parseYAMLSafe(data []byte) (*Manifest, error) {
	// Reject unsafe explicit tags before decoding. The struct decode below
	// will not execute tags, but we additionally reject any custom/explicit
	// tag (other than the standard core schema tags) to honour the "no
	// arbitrary or executable tags" requirement explicitly.
	if err := rejectUnsafeYAML(data); err != nil {
		return nil, err
	}

	dec := yaml.NewDecoder(bytes.NewReader(data))
	var m Manifest
	if err := dec.Decode(&m); err != nil {
		return nil, fmt.Errorf("yaml decode: %w", err)
	}
	// Reject multi-document streams: a second successful decode means more
	// than one document is present.
	var extra interface{}
	if err := dec.Decode(&extra); err == nil {
		return nil, errors.New("yaml safe profile: multi-document streams are not permitted")
	}
	return &m, nil
}

// rejectUnsafeYAML walks the YAML node tree and rejects any non-core explicit
// tag (executable/arbitrary tags such as !!python/... or custom !Foo tags)
// and unbounded alias use. Anchors/aliases are bounded by limiting the alias
// count; any alias at all in a manifest is unexpected, so we reject aliases
// to defend against alias-expansion denial of service.
func rejectUnsafeYAML(data []byte) error {
	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		// A genuine parse error is surfaced by the typed decode later; here we
		// only police tags/aliases. Returning nil lets the typed decode report.
		return nil
	}
	return walkYAML(&root)
}

func walkYAML(n *yaml.Node) error {
	if n == nil {
		return nil
	}
	if n.Kind == yaml.AliasNode {
		return errors.New("yaml safe profile: anchors/aliases are not permitted")
	}
	// Permit only the standard core-schema tags and untagged nodes.
	if t := n.Tag; t != "" && !isCoreYAMLTag(t) {
		return fmt.Errorf("yaml safe profile: tag %q is not permitted (no arbitrary or executable tags)", t)
	}
	for _, c := range n.Content {
		if err := walkYAML(c); err != nil {
			return err
		}
	}
	return nil
}

func isCoreYAMLTag(tag string) bool {
	switch tag {
	case "!!str", "!!int", "!!float", "!!bool", "!!null",
		"!!map", "!!seq", "!!merge", "!!timestamp", "!!binary":
		return true
	default:
		return false
	}
}
