// generated from spec: zypper-declarative.spec.md sha256:b2d0de88fbed1163678e59e931c741b9d999b71f902f6eb01db8790bb813d057
//
// resolve-format: the single authority for choosing a manifest serialisation.
package manifest

import "strings"

// ResolveFormat implements BEHAVIOR/INTERNAL: resolve-format.
//
//	explicit  the format= option, or "" if not given.
//	path      the operative file path, or "" for stdin/stdout.
//	def       the manifest-format CONFIG default.
//
// Returns the resolved format and an *Diagnostic (domain=invocation) if an
// explicit value is given but is not a recognised format.
func ResolveFormat(explicit string, path string, def Format) (Format, *Diagnostic) {
	// 1. An explicit format= always wins.
	if explicit != "" {
		switch strings.ToLower(explicit) {
		case "json":
			return FormatJSON, nil
		case "yaml":
			return FormatYAML, nil
		default:
			return "", NewError(DomainInvocation, "unknown format value: "+explicit)
		}
	}
	// 2. Else the operative file extension decides, when recognised.
	if path != "" {
		lower := strings.ToLower(path)
		switch {
		case strings.HasSuffix(lower, ".json"):
			return FormatJSON, nil
		case strings.HasSuffix(lower, ".yaml"), strings.HasSuffix(lower, ".yml"):
			return FormatYAML, nil
		}
	}
	// 3. Else the manifest-format default.
	if def == "" {
		def = FormatJSON
	}
	return def, nil
}
