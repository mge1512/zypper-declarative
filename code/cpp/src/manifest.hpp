// generated from spec: zypper-declarative.spec.md sha256:f2cc80627e483a48bb8411d297711bc5f6c6e74c28dbf0dafc8fe7bd8817251e
//
// manifest.hpp -- serialisation/parsing of the shared data model and the
// internal behaviours that read manifests: resolve-format, load-desired-manifest,
// load-applied-record, plus the canonical-model hash. underscore_style keys.
#ifndef ZD_MANIFEST_HPP
#define ZD_MANIFEST_HPP

#include "types.hpp"
#include "config.hpp"
#include <string>
#include <optional>
#include <variant>

namespace zd {

// Result of a load: either a Manifest (+ desired_sha256) or a Diagnostic.
struct LoadResult {
    bool ok = false;
    Manifest manifest;
    Sha256 desired_sha256;
    Diagnostic error;
};

// BEHAVIOR/INTERNAL: resolve-format
// explicit wins; else recognised path extension; else the manifest-format default.
ManifestFormat resolve_format(const std::optional<ManifestFormat>& explicit_fmt,
                              const std::optional<std::string>& path,
                              ManifestFormat default_fmt);

// Serialise a Manifest in the resolved format.
//   json: canonical Machinery JSON (format_version 1, ScopeWrapper idiom).
//   pretty controls indentation of JSON output; the canonical hash always uses
//   compact form regardless of `pretty`.
std::string serialize_manifest(const Manifest& m, ManifestFormat fmt, bool pretty);

// The canonical compact JSON serialisation used as the basis for desired_sha256:
// keys sorted, _elements sorted by identity key, compact separators.
std::string canonical_json(const Manifest& m);

// Compute desired_sha256 = SHA256(canonical_json(parsed model)).
Sha256 canonical_hash(const Manifest& m);

// Parse a document (already read into memory) of the given format into a
// Manifest, applying the schema validation and (for yaml) the safe profile.
// On failure sets ok=false and fills error.
LoadResult parse_manifest(const std::string& text, ManifestFormat fmt,
                          bool is_desired);

// BEHAVIOR/INTERNAL: load-desired-manifest
LoadResult load_desired_manifest(const std::string& manifest_path,
                                 const std::optional<ManifestFormat>& explicit_fmt,
                                 const Config& cfg);

// Load a captured actual-state dump (for diff/verify state-path). Same schema
// but observational scopes are tolerated (and used). Malformed -> invocation.
LoadResult load_state_dump(const std::string& state_path,
                           const std::optional<ManifestFormat>& explicit_fmt,
                           const Config& cfg);

// BEHAVIOR/INTERNAL: load-applied-record
struct AppliedResult {
    bool ok = true;           // false only on a present-but-corrupt record
    bool present = false;
    AppliedRecord record;     // all scopes empty if not present
    Diagnostic error;
};
AppliedResult load_applied_record(const std::string& root);

// Write a manifest to a path (or stdout if path is nullopt). Returns false on
// write failure (caller maps to exit 2).
bool write_manifest(const Manifest& m, ManifestFormat fmt,
                    const std::optional<std::string>& out_path);

} // namespace zd

#endif // ZD_MANIFEST_HPP
