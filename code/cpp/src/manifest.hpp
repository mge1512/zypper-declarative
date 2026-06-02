// generated from spec: zypper-declarative.spec.md sha256:aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3
//
// Manifest serialisation and parsing. Single authority for serialisation
// choice (resolve-format), JSON/YAML read/write, the canonical-model hash
// (desired_sha256), and load-desired-manifest / load-applied-record.
#ifndef ZD_MANIFEST_HPP
#define ZD_MANIFEST_HPP

#include <optional>
#include <string>

#include "diagnostic.hpp"
#include "types.hpp"

namespace zd {

// resolve-format: explicit format= wins, else operative file extension
// (.json -> json, .yaml/.yml -> yaml), else the manifest-format default.
ManifestFormat resolve_format(const std::optional<ManifestFormat>& explicit_fmt,
                              const std::optional<std::string>& path,
                              ManifestFormat default_fmt);

// Parse a recognised format string ("json"/"yaml"); nullopt if unknown.
std::optional<ManifestFormat> parse_format(const std::string& s);

// Serialise a Manifest to canonical JSON (Machinery format_version 1).
// pretty=true produces indented output for human/file consumption; pretty=false
// produces the compact canonical form used for hashing.
std::string serialise_json(const Manifest& m, bool pretty);

// Serialise a Manifest to YAML (same data model; not Machinery-compatible).
std::string serialise_yaml(const Manifest& m);

// Canonical-model hash: SHA256 of the canonical (compact, sorted) JSON
// serialisation of the parsed data model. Format-independent.
std::string canonical_sha256(const Manifest& m);

// Hex SHA256 of arbitrary bytes (libcrypto).
std::string sha256_hex(const std::string& bytes);

// Parse a manifest document from text in the given format into the data model.
// schema_validate=true enforces the desired-manifest rules (format_version==1,
// observational scopes must be empty/absent). Returns a Diagnostic on failure.
struct LoadedManifest {
    Manifest manifest;
    std::string desired_sha256;  // canonical-model hash
};

// load-desired-manifest: read file at path, resolve format, parse (YAML under a
// safe profile), schema-validate, compute desired_sha256.
Result<LoadedManifest> load_desired_manifest(
    const std::string& manifest_path,
    const std::optional<ManifestFormat>& explicit_fmt,
    ManifestFormat default_fmt);

// Load a captured actual-state dump (a Manifest) from a file. Observational
// scopes are tolerated (this is an actual-state dump, not a desired manifest).
Result<Manifest> load_state_dump(const std::string& state_path,
                                 const std::optional<ManifestFormat>& explicit_fmt,
                                 ManifestFormat default_fmt);

// load-applied-record: read <root>/usr/lib/zypper-declarative/applied.json.
struct AppliedLoad {
    Manifest record;  // all scopes empty if absent
    bool present = false;
};
Result<AppliedLoad> load_applied_record(const std::string& root);

}  // namespace zd

#endif  // ZD_MANIFEST_HPP
