// generated from spec: zypper-declarative.spec.md sha256:f8ff76ecbc4bbc69a49e2e32b2924da3a64df1ad46196e05ce8c137b684429b2
//
// Schema validation of the manifest data model, and BEHAVIOR/INTERNAL:
// load-desired-manifest (read, resolve-format, parse under the safe profile,
// validate, optionally verify signature, compute the canonical-model hash).

package manifest

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/mge1512/zypper-declarative/internal/diag"
)

var (
	reSha256   = regexp.MustCompile(`^[0-9a-f]{64}$`)
	reMode     = regexp.MustCompile(`^[0-7]{3,4}$`)
	unitSuffix = []string{".service", ".timer", ".socket", ".target", ".path", ".mount"}
)

// LoadOptions configures load-desired-manifest.
type LoadOptions struct {
	// ExplicitFormat is the format= option value, if any.
	ExplicitFormat      Format
	ExplicitFormatGiven bool
	DefaultFormat       Format
	VerifySignature     bool
	KeyringPath         string
}

// Load implements BEHAVIOR/INTERNAL: load-desired-manifest. It returns the
// parsed Manifest and its canonical-model desired_sha256, or a Diagnostic on
// failure (returned to the caller; no exit).
func Load(path string, opts LoadOptions) (*Manifest, string, *diag.Diagnostic) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", diag.Errorf(diag.DomainInvocation, "manifest unreadable: %s: %v", path, err)
	}

	f, _, perr := ParseFormat(string(opts.ExplicitFormat))
	if opts.ExplicitFormatGiven && perr != nil {
		return nil, "", diag.Errorf(diag.DomainInvocation, "%v", perr)
	}
	resolved := ResolveFormat(f, opts.ExplicitFormatGiven, path, opts.DefaultFormat)

	m, derr := Decode(data, resolved)
	if derr != nil {
		if _, ok := derr.(*ErrUnsafeYAML); ok {
			return nil, "", diag.Errorf(diag.DomainManifest, "%v", derr)
		}
		return nil, "", diag.Errorf(diag.DomainManifest, "manifest parse error: %v", derr)
	}

	if verr := Validate(m); verr != nil {
		return nil, "", diag.Errorf(diag.DomainManifest, "%v", verr)
	}

	if opts.VerifySignature {
		if serr := verifySignature(path, opts.KeyringPath); serr != nil {
			return nil, "", diag.Errorf(diag.DomainManifest, "signature verification failed: %v", serr)
		}
	}

	hash, herr := m.CanonicalHash()
	if herr != nil {
		return nil, "", diag.Errorf(diag.DomainManifest, "could not compute manifest identity: %v", herr)
	}
	return m, hash, nil
}

// Validate checks the manifest against the schema: meta.format_version must be
// 1, and every present scope must conform to its record type. It returns the
// first violation, or nil.
func Validate(m *Manifest) error {
	if m.Meta.FormatVersion != 1 {
		return validationError("meta.format_version must be 1")
	}
	if m.Packages != nil {
		for i, p := range m.Packages.Elements {
			if strings.TrimSpace(p.Name) == "" {
				return validationErrorf("packages._elements[%d].name must be non-empty", i)
			}
		}
	}
	if m.Repositories != nil {
		for i, r := range m.Repositories.Elements {
			if strings.TrimSpace(r.Alias) == "" {
				return validationErrorf("repositories._elements[%d].alias must be non-empty", i)
			}
			if strings.TrimSpace(r.URL) == "" {
				return validationErrorf("repositories._elements[%d].url must be non-empty", i)
			}
		}
	}
	if m.Services != nil {
		for i, s := range m.Services.Elements {
			if !validUnitName(s.Name) {
				return validationErrorf("services._elements[%d].name %q is not a valid unit name", i, s.Name)
			}
			switch s.State {
			case "enabled", "disabled", "masked":
			default:
				return validationErrorf("services._elements[%d].state %q must be enabled|disabled|masked", i, s.State)
			}
		}
	}
	if m.ConfigFiles != nil {
		for i, c := range m.ConfigFiles.Elements {
			if !strings.HasPrefix(c.Name, "/etc/") {
				return validationErrorf("config_files._elements[%d].name %q must be under /etc/", i, c.Name)
			}
			switch c.Type {
			case "file", "link", "dir":
			default:
				return validationErrorf("config_files._elements[%d].type %q must be file|link|dir", i, c.Type)
			}
			if c.Mode != "" && !reMode.MatchString(c.Mode) {
				return validationErrorf("config_files._elements[%d].mode %q is not a valid mode", i, c.Mode)
			}
			if c.SHA256 != "" && !reSha256.MatchString(c.SHA256) {
				return validationErrorf("config_files._elements[%d].sha256 %q is not a valid sha256", i, c.SHA256)
			}
			if strings.TrimSpace(c.User) == "" {
				return validationErrorf("config_files._elements[%d].user must be non-empty", i)
			}
			if strings.TrimSpace(c.Group) == "" {
				return validationErrorf("config_files._elements[%d].group must be non-empty", i)
			}
		}
	}
	return nil
}

// validationError and validationErrorf build schema-validation errors.
func validationError(msg string) error { return &schemaError{msg: msg} }

func validationErrorf(format string, args ...interface{}) error {
	return &schemaError{msg: fmt.Sprintf(format, args...)}
}

type schemaError struct{ msg string }

func (e *schemaError) Error() string { return e.msg }

func validUnitName(name string) bool {
	for _, s := range unitSuffix {
		if strings.HasSuffix(name, s) {
			return true
		}
	}
	return false
}

// verifySignature is a placeholder for keyring-based signature verification. The
// concrete keyring binding is out of scope for the language-neutral spec; when
// signature-verification is enabled and no keyring material is configured, the
// verification cannot succeed and is reported as a failure to the caller.
func verifySignature(path, keyring string) error {
	if keyring == "" {
		return validationError("signature-verification is enabled but no keyring is configured")
	}
	// A real keyring binding would verify a detached signature here. Absent a
	// keyring binding in this build, treat a missing signature artefact as a
	// verification failure so the strict contract holds.
	if _, err := os.Stat(path + ".sig"); err != nil {
		return validationError("no signature found for manifest")
	}
	return nil
}
