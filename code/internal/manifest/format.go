// generated from spec: zypper-declarative.spec.md sha256:58e1636e2de82ab81a5cd3f81d6b3c9ac6a8976e18f9abb2bbd2b2aba56fe4d4
//
// resolve-format: the single authority for choosing a manifest serialisation.
// Every read that parses a manifest and every write that serialises one resolves
// its format here, so input and output behave symmetrically.
package manifest

import (
	"path/filepath"
	"strings"
)

// Format is the serialisation of the manifest data model.
type Format string

const (
	FormatJSON Format = "json"
	FormatYAML Format = "yaml"
)

// ErrUnknownFormat is returned for an explicit format value that is not
// recognised.
type ErrUnknownFormat struct{ Value string }

func (e *ErrUnknownFormat) Error() string {
	return "unknown format value: " + e.Value
}

// ParseFormat validates an explicit format string. The empty string means "no
// explicit format". An unrecognised non-empty value is an error.
func ParseFormat(s string) (Format, bool, error) {
	if s == "" {
		return "", false, nil
	}
	switch strings.ToLower(s) {
	case "json":
		return FormatJSON, true, nil
	case "yaml":
		return FormatYAML, true, nil
	default:
		return "", false, &ErrUnknownFormat{Value: s}
	}
}

// ResolveFormat chooses a serialisation per the spec resolve-format behaviour:
//  1. an explicit format always wins;
//  2. else the operative file extension decides (.json -> json, .yaml/.yml -> yaml);
//  3. else the manifest-format CONFIG default.
//
// explicitGiven indicates whether an explicit format= was supplied; explicit is
// the resolved explicit value. path is the operative file path ("" for
// stdin/stdout). def is the manifest-format default.
func ResolveFormat(explicit Format, explicitGiven bool, path string, def Format) Format {
	if explicitGiven {
		return explicit
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
