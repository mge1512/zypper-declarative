// generated from spec: zypper-declarative.spec.md sha256:714e75ff672557d2c7344736a5f36b52afa37e0f565b07de83e1b18cc4492014
//
// Format resolution: the single authority for choosing a serialisation. Used
// by load-desired-manifest (manifest-path), verify (state-path), and describe
// (out). Precedence: explicit format=, else operative file extension, else the
// manifest-format CONFIG default.
package manifest

import (
	"fmt"
	"strings"
)

// ResolveFormat applies the format-selection precedence (load-desired-manifest
// STEP 2 / decisions hints resolve-format): explicit value first, then the
// operative path extension, then the configured default.
//
// explicit is the value of the format= option ("" when unset). path is the
// operative path (manifest-path, state-path, or out); "" for stdin/stdout.
// def is the manifest-format CONFIG default.
//
// An explicit format that is neither json nor yaml is an error (unknown format
// value). The returned error is non-nil only for an unknown explicit value.
func ResolveFormat(explicit, path string, def Format) (Format, error) {
	if explicit != "" {
		switch Format(strings.ToLower(explicit)) {
		case FormatJSON:
			return FormatJSON, nil
		case FormatYAML:
			return FormatYAML, nil
		default:
			return "", fmt.Errorf("unknown format value %q", explicit)
		}
	}
	if path != "" {
		lower := strings.ToLower(path)
		switch {
		case strings.HasSuffix(lower, ".json"):
			return FormatJSON, nil
		case strings.HasSuffix(lower, ".yaml"), strings.HasSuffix(lower, ".yml"):
			return FormatYAML, nil
		}
	}
	if def == "" {
		return FormatJSON, nil
	}
	return def, nil
}
