// generated from spec: zypper-declarative.spec.md sha256:f8ff76ecbc4bbc69a49e2e32b2924da3a64df1ad46196e05ce8c137b684429b2
//
// resolve-format: the single authority for choosing a manifest serialisation.
// Every read that parses a manifest and every write that serialises one resolves
// its format here, so input and output behave symmetrically.

package manifest

import (
	"path/filepath"
	"strings"
)

// ErrUnknownFormat reports an explicit but unrecognised format value.
type ErrUnknownFormat struct {
	Value string
}

func (e *ErrUnknownFormat) Error() string {
	return "unknown format value: " + e.Value
}

// ParseFormat parses an explicit format= option value. The empty string means
// "no explicit format". An unrecognised non-empty value returns ErrUnknownFormat.
func ParseFormat(value string) (Format, bool, error) {
	if value == "" {
		return "", false, nil
	}
	switch strings.ToLower(value) {
	case "json":
		return FormatJSON, true, nil
	case "yaml":
		return FormatYAML, true, nil
	default:
		return "", false, &ErrUnknownFormat{Value: value}
	}
}

// ResolveFormat implements BEHAVIOR/INTERNAL: resolve-format.
//
//	explicit  the format= option (or "" for none), and whether it was given
//	path      the operative file path (or "" for stdin/stdout)
//	def       the manifest-format CONFIG default
//
// Precedence: explicit wins; else a recognised file extension; else the default.
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
