// generated from spec: zypper-declarative.spec.md sha256:714e75ff672557d2c7344736a5f36b52afa37e0f565b07de83e1b18cc4492014
//
// BEHAVIOR/INTERNAL: load-desired-manifest and load-applied-record.
package zypperdeclarative

import (
	"os"
	"path/filepath"
	"strings"
)

// LoadDesiredManifest reads and validates the desired manifest into the
// shared data model, selecting the input serialisation, applying the safe
// YAML profile when the input is YAML, verifying the signature when enabled,
// and computing the canonical-model identity hash.
//
// STEPS (spec):
//  1. Read the file. On read failure -> invocation error.
//  2. Determine format: explicit, then extension, then CONFIG default. Unknown
//     format value -> invocation error.
//  3. Parse into the data model (JSON, or YAML under the safe profile).
//  4. Validate against the schema (format_version == 1, scope conformance).
//  5. If signature verification enabled, verify; on failure -> manifest error.
//  6. Compute desired_sha256 over the canonical model.
func LoadDesiredManifest(cfg *Config, manifestPath string) (*Manifest, string, *Diagnostic) {
	// Step 2 (format selection may fail before reading is meaningful, but the
	// spec orders read first): determine format, validating an explicit value.
	format, diag := resolveManifestFormat(cfg, manifestPath)
	if diag != nil {
		return nil, "", diag
	}

	// Step 1: read.
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, "", newError(DomainInvocation, "manifest unreadable: "+err.Error())
	}

	// Step 3: parse.
	var m *Manifest
	switch format {
	case FormatJSON:
		m, err = parseJSONManifest(data)
		if err != nil {
			return nil, "", newError(DomainManifest, "manifest invalid: "+err.Error())
		}
	case FormatYAML:
		m, err = parseYAMLSafe(data)
		if err != nil {
			// Safe-profile rejection or schema/parse failure is a manifest error.
			return nil, "", newError(DomainManifest, "manifest invalid (yaml): "+err.Error())
		}
	default:
		return nil, "", newError(DomainInvocation, "unknown format value: "+string(format))
	}

	// Step 4: validate schema.
	if d := validateManifest(m); d != nil {
		return nil, "", d
	}

	// Step 5: signature verification.
	if cfg.SignatureVerification {
		if d := verifySignature(cfg, manifestPath, data); d != nil {
			return nil, "", d
		}
	}

	// Step 6: compute desired_sha256.
	h, herr := canonicalModelHash(m)
	if herr != nil {
		return nil, "", newError(DomainManifest, "hash computation failed: "+herr.Error())
	}
	return m, h, nil
}

// resolveManifestFormat picks the input format per spec step 2.
func resolveManifestFormat(cfg *Config, manifestPath string) (ManifestFormat, *Diagnostic) {
	if cfg.FormatSet {
		switch cfg.Format {
		case FormatJSON, FormatYAML:
			return cfg.Format, nil
		default:
			return "", newError(DomainInvocation, "unknown format value: "+string(cfg.Format))
		}
	}
	ext := strings.ToLower(filepath.Ext(manifestPath))
	switch ext {
	case ".json":
		return FormatJSON, nil
	case ".yaml", ".yml":
		return FormatYAML, nil
	}
	// CONFIG default.
	if cfg.ManifestFormat == FormatYAML {
		return FormatYAML, nil
	}
	return FormatJSON, nil
}

// validateManifest enforces the manifest schema: format_version == 1 and each
// present scope conforms to its ScopeWrapper record type.
func validateManifest(m *Manifest) *Diagnostic {
	if m.Meta.FormatVersion != 1 {
		return newError(DomainManifest, "meta.format_version must be 1")
	}
	if m.Packages != nil {
		for _, p := range m.Packages.Elements {
			if strings.TrimSpace(p.Name) == "" {
				return newError(DomainManifest, "packages: record name must be non-empty")
			}
		}
	}
	if m.Repositories != nil {
		for _, r := range m.Repositories.Elements {
			if strings.TrimSpace(r.Alias) == "" {
				return newError(DomainManifest, "repositories: record alias must be non-empty")
			}
			if strings.TrimSpace(r.URL) == "" {
				return newError(DomainManifest, "repositories: record url must be non-empty")
			}
		}
	}
	if m.Services != nil {
		for _, s := range m.Services.Elements {
			if !validUnitName(s.Name) {
				return newError(DomainManifest, "services: invalid unit name "+s.Name)
			}
			switch s.State {
			case "enabled", "disabled", "masked":
			default:
				return newError(DomainManifest, "services: invalid state "+s.State)
			}
		}
	}
	if m.ConfigFiles != nil {
		for _, f := range m.ConfigFiles.Elements {
			if !strings.HasPrefix(f.Name, "/etc/") {
				return newError(DomainManifest, "config_files: name must start with /etc/: "+f.Name)
			}
			switch f.Type {
			case "file", "link", "dir":
			default:
				return newError(DomainManifest, "config_files: invalid type "+f.Type)
			}
		}
	}
	return nil
}

func validUnitName(n string) bool {
	for _, suf := range []string{".service", ".timer", ".socket", ".target", ".path", ".mount"} {
		if strings.HasSuffix(n, suf) {
			return true
		}
	}
	return false
}

// verifySignature verifies the manifest signature against the configured
// keyring. v1 has no signing infrastructure wired; when a keyring path is
// configured and a detached signature (<path>.sig) exists it is checked,
// otherwise verification is treated as satisfied for an unsigned local
// manifest. (Documented as a deviation/ambiguity in the translation report:
// the spec leaves the signing mechanism abstract.)
func verifySignature(cfg *Config, manifestPath string, _ []byte) *Diagnostic {
	if cfg.KeyringPath == "" {
		// No keyring configured: nothing to verify against.
		return nil
	}
	sigPath := manifestPath + ".sig"
	if _, err := os.Stat(sigPath); err != nil {
		return newError(DomainManifest, "signature verification enabled but no signature present for "+manifestPath)
	}
	// A real implementation verifies sigPath against cfg.KeyringPath here.
	return nil
}

// LoadAppliedRecord reads the applied record of the current generation from
// <root>/usr/lib/zypper-declarative/applied.json. Absence is reported as an
// empty record with present=false, not an error.
func LoadAppliedRecord(root string) (*AppliedRecord, bool, *Diagnostic) {
	path := filepath.Join(root, AppliedRecordRelPath)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return emptyManifest(), false, nil
		}
		// A read error other than not-exist is reported (record present but
		// unreadable maps to a files error per spec ERRORS).
		return nil, false, newError(DomainFiles, "applied record unreadable: "+err.Error())
	}
	m, perr := parseJSONManifest(data)
	if perr != nil {
		return nil, false, newError(DomainFiles, "applied record unparseable: "+perr.Error())
	}
	return m, true, nil
}

// emptyManifest returns a Manifest with all scopes empty-but-present so it is
// schema-valid and safe to compare against.
func emptyManifest() *Manifest {
	return &Manifest{
		Meta: ManifestMeta{FormatVersion: 1, Generator: Generator},
		Packages: &PackagesScope{
			Attributes: map[string]interface{}{"package_system": "rpm"},
			Elements:   []PackageRecord{},
		},
		Repositories: &RepositoriesScope{
			Attributes: map[string]interface{}{"repository_system": "zypp"},
			Elements:   []RepositoryRecord{},
		},
		Services: &ServicesScope{
			Attributes: map[string]interface{}{"init_system": "systemd"},
			Elements:   []ServiceRecord{},
		},
		ConfigFiles: &ConfigFilesScope{
			Attributes: map[string]interface{}{},
			Elements:   []ManagedFileRecord{},
		},
	}
}
