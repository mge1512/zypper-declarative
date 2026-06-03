// generated from spec: zypper-declarative.spec.md sha256:aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3
#pragma once

#include <string>
#include <optional>

#include "types.hpp"
#include "diagnostic.hpp"

namespace zd {

// BEHAVIOR/INTERNAL: resolve-format. Explicit format= wins, else the operative
// file extension (.json / .yaml / .yml), else the manifest-format CONFIG
// default.
ManifestFormat resolve_format(std::optional<ManifestFormat> explicit_fmt,
                              const std::optional<std::string>& path,
                              ManifestFormat config_default);

// Serialise a Manifest. JSON is pretty-printed (Machinery-compatible);
// canonical (compact, sorted) is used only for the hash. YAML mirrors the
// data model. A nullopt scope is omitted entirely (never null/empty object).
std::string serialize_manifest(const Manifest& m, ManifestFormat fmt);

// Canonical JSON serialisation of the parsed data model: keys sorted, compact,
// elements sorted by identity key. Used as the desired_sha256 preimage so the
// same intent in JSON or YAML yields the same hash.
std::string canonical_json(const Manifest& m);

// Compute desired_sha256 = SHA256 of canonical_json(m) (format-independent).
std::string canonical_hash(const Manifest& m);

// BEHAVIOR/INTERNAL: load-desired-manifest. Reads, resolves format, parses
// (safe YAML profile when YAML), schema-validates (format_version == 1,
// scopes well-formed, observational scopes must be empty/absent), and computes
// desired_sha256. Returns the Manifest plus its hash, or a Diagnostic.
struct LoadedManifest {
    Manifest manifest;
    std::string desired_sha256;
};
Result<LoadedManifest> load_desired_manifest(const std::string& manifest_path,
                                             std::optional<ManifestFormat> explicit_fmt,
                                             ManifestFormat config_default,
                                             bool require_signature);

// Parse a captured actual-state dump as a Manifest (offline, no live read).
// Observational scopes are tolerated (a full describe dump may carry them).
// Malformed input => invocation error.
Result<Manifest> load_state_dump(const std::string& state_path,
                                 std::optional<ManifestFormat> explicit_fmt,
                                 ManifestFormat config_default);

// BEHAVIOR/INTERNAL: load-applied-record.
struct AppliedLoad {
    Manifest record;
    bool present = false;
};
Result<AppliedLoad> load_applied_record(const std::string& root);

}  // namespace zd
