// generated from spec: zypper-declarative.spec.md sha256:58e1636e2de82ab81a5cd3f81d6b3c9ac6a8976e18f9abb2bbd2b2aba56fe4d4
//
// Manifest schema validation. Validates a parsed Manifest against the manifest
// schema: meta.format_version must be 1, and every present scope must conform to
// its ScopeWrapper record type with the required refinement predicates.
package manifest

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	reSha256 = regexp.MustCompile(`^[0-9a-f]{64}$`)
	reMode   = regexp.MustCompile(`^[0-7]{3,4}$`)
)

var unitSuffixes = []string{".service", ".timer", ".socket", ".target", ".path", ".mount"}

// ValidationError is a schema violation naming the first problem found.
type ValidationError struct{ Msg string }

func (e *ValidationError) Error() string { return e.Msg }

func verr(format string, args ...interface{}) *ValidationError {
	return &ValidationError{Msg: fmt.Sprintf(format, args...)}
}

// Validate checks a Manifest against the schema and returns the first violation,
// or nil if valid.
func Validate(m Manifest) error {
	if m.Meta.FormatVersion != 1 {
		return verr("meta.format_version must be 1, got %d", m.Meta.FormatVersion)
	}
	if m.Packages != nil {
		for i, p := range m.Packages.Elements {
			if strings.TrimSpace(p.Name) == "" {
				return verr("packages._elements[%d].name must be non-empty", i)
			}
		}
	}
	if m.Repositories != nil {
		for i, r := range m.Repositories.Elements {
			if strings.TrimSpace(r.Alias) == "" {
				return verr("repositories._elements[%d].alias must be non-empty", i)
			}
			if strings.TrimSpace(r.URL) == "" {
				return verr("repositories._elements[%d].url must be non-empty", i)
			}
		}
	}
	if m.Services != nil {
		for i, s := range m.Services.Elements {
			if !isUnitName(s.Name) {
				return verr("services._elements[%d].name %q is not a valid unit name", i, s.Name)
			}
			switch s.State {
			case "enabled", "disabled", "masked":
			default:
				return verr("services._elements[%d].state %q must be one of enabled|disabled|masked", i, s.State)
			}
		}
	}
	if m.ConfigFiles != nil {
		for i, f := range m.ConfigFiles.Elements {
			if !strings.HasPrefix(f.Name, "/etc/") {
				return verr("config_files._elements[%d].name %q must be an absolute path under /etc/", i, f.Name)
			}
			switch f.Type {
			case "file", "link", "dir":
			default:
				return verr("config_files._elements[%d].type %q must be one of file|link|dir", i, f.Type)
			}
			if !reMode.MatchString(f.Mode) {
				return verr("config_files._elements[%d].mode %q is not a valid mode", i, f.Mode)
			}
			if strings.TrimSpace(f.User) == "" {
				return verr("config_files._elements[%d].user must be non-empty", i)
			}
			if strings.TrimSpace(f.Group) == "" {
				return verr("config_files._elements[%d].group must be non-empty", i)
			}
			if !reSha256.MatchString(f.SHA256) {
				return verr("config_files._elements[%d].sha256 %q is not a valid sha256", i, f.SHA256)
			}
		}
	}
	return nil
}

func isUnitName(s string) bool {
	for _, suf := range unitSuffixes {
		if strings.HasSuffix(s, suf) {
			return true
		}
	}
	return false
}
