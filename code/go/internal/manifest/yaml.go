// generated from spec: zypper-declarative.spec.md sha256:f2cc80627e483a48bb8411d297711bc5f6c6e74c28dbf0dafc8fe7bd8817251e
//
// YAML edge: a safe-profile loader. The chosen route is to decode the single
// YAML document into a generic value with a non-code-executing loader
// (gopkg.in/yaml.v3, which never executes tags), reject any disabled feature
// (custom/executable tags, anchors/aliases, multi-document streams), convert the
// generic value to JSON, and decode that JSON strictly into the data model with
// DisallowUnknownFields. This uses JSON typing rather than YAML implicit typing,
// so values such as NO or 1.10 are not coerced.
package manifest

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	yaml "gopkg.in/yaml.v3"
)

// ErrUnsafeYAML indicates the YAML input requires a disabled (unsafe) feature.
var ErrUnsafeYAML = errors.New("yaml requires a disabled (unsafe) feature")

// marshalYAML renders the manifest as YAML by first producing canonical-shaped
// JSON (via the JSON struct tags) and re-encoding as YAML, so the YAML keys
// match the underscore_style schema exactly.
func marshalYAML(m *Manifest) ([]byte, error) {
	j, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	var generic interface{}
	dec := json.NewDecoder(bytes.NewReader(j))
	dec.UseNumber()
	if err := dec.Decode(&generic); err != nil {
		return nil, err
	}
	return yaml.Marshal(generic)
}

// scanUnsafeYAML walks a yaml.Node tree and rejects any disabled feature:
// custom/non-core tags, anchors, and aliases.
func scanUnsafeYAML(n *yaml.Node) error {
	if n == nil {
		return nil
	}
	if n.Anchor != "" {
		return fmt.Errorf("%w: anchor", ErrUnsafeYAML)
	}
	if n.Kind == yaml.AliasNode {
		return fmt.Errorf("%w: alias", ErrUnsafeYAML)
	}
	// Reject explicit non-core tags (executable / arbitrary tags). The core
	// schema tags begin with "tag:yaml.org,2002:"; an explicitly-set custom tag
	// such as "!!python/object" or "!mytag" is rejected.
	switch n.Tag {
	case "", "!!map", "!!seq", "!!str", "!!int", "!!float", "!!bool", "!!null",
		"!!binary", "!!timestamp":
		// core / standard tags are allowed
	default:
		if len(n.Tag) > 0 && n.Tag[0] == '!' {
			return fmt.Errorf("%w: tag %q", ErrUnsafeYAML, n.Tag)
		}
	}
	for _, c := range n.Content {
		if err := scanUnsafeYAML(c); err != nil {
			return err
		}
	}
	return nil
}

// decodeYAMLSafe parses a single YAML document under the safe profile into the
// data model. It rejects multi-document streams, anchors/aliases, and custom
// tags, then converts to JSON and decodes strictly.
func decodeYAMLSafe(data []byte, out *Manifest) error {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	var doc yaml.Node
	if err := dec.Decode(&doc); err != nil {
		if errors.Is(err, io.EOF) {
			return fmt.Errorf("empty yaml document")
		}
		return err
	}
	// Reject a second document (single-document streams only).
	var probe yaml.Node
	if err := dec.Decode(&probe); err == nil {
		return fmt.Errorf("%w: multiple documents", ErrUnsafeYAML)
	} else if !errors.Is(err, io.EOF) {
		return err
	}
	if err := scanUnsafeYAML(&doc); err != nil {
		return err
	}
	// Convert the safe node to a generic value, then to JSON, then strict-decode.
	var generic interface{}
	if err := doc.Decode(&generic); err != nil {
		return err
	}
	jb, err := json.Marshal(generic)
	if err != nil {
		return err
	}
	jd := json.NewDecoder(bytes.NewReader(jb))
	jd.DisallowUnknownFields()
	if err := jd.Decode(out); err != nil {
		return err
	}
	return nil
}
