// generated from spec: zypper-declarative.spec.md sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
//
// resolve-format: the single authority for choosing a manifest serialisation.
package manifest

import (
	"errors"
	"path/filepath"
	"strings"
)

// ErrUnknownFormat is returned for an explicit but unrecognised format value.
var ErrUnknownFormat = errors.New("unknown manifest format")

// ParseFormat validates an explicit format= value.
func ParseFormat(s string) (Format, error) {
	switch strings.ToLower(s) {
	case "json":
		return FormatJSON, nil
	case "yaml":
		return FormatYAML, nil
	default:
		return "", ErrUnknownFormat
	}
}

// ResolveFormat implements BEHAVIOR/INTERNAL: resolve-format.
//
//  1. If explicit is non-empty, it wins (already validated by the caller into a Format).
//  2. Else if path has a recognised extension, .json -> json, .yaml/.yml -> yaml.
//  3. Else return the manifest-format default.
func ResolveFormat(explicit *Format, path string, def Format) Format {
	if explicit != nil {
		return *explicit
	}
	if path != "" {
		switch strings.ToLower(filepath.Ext(path)) {
		case ".json":
			return FormatJSON
		case ".yaml", ".yml":
			return FormatYAML
		}
	}
	return def
}
