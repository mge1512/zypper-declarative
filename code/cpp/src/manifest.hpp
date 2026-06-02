// generated from spec: zypper-declarative.spec.md sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
//
// Manifest serialisation and loading. Covers:
//   resolve-format          single authority for serialisation choice
//   serialise               Manifest -> JSON or YAML (canonical, ScopeWrapper)
//   canonical_json          deterministic JSON used for desired_sha256
//   load-desired-manifest   read+validate+hash a desired manifest
//   load-applied-record     read the applied record of a generation
#ifndef ZD_MANIFEST_HPP
#define ZD_MANIFEST_HPP

#include <optional>
#include <string>

#include "config.hpp"
#include "types.hpp"

namespace zd {

// resolve-format: explicit format= wins; else operative file extension; else
// the manifest-format CONFIG default.
ManifestFormat resolve_format(const std::optional<ManifestFormat>& explicit_fmt,
                              const std::optional<std::string>& path,
                              ManifestFormat config_default);

// Serialise a Manifest in the resolved format. A nullopt scope is OMITTED
// entirely; a present scope is written with _attributes (object, never null)
// and _elements. JSON is pretty (2-space). YAML quotes string scalars.
std::string serialise(const Manifest& m, ManifestFormat fmt);

// Canonical JSON of the parsed data model used for the format-independent
// desired_sha256: keys sorted, compact, _elements sorted by identity key,
// meta.created_at and meta.desired_sha256 excluded from the hashed form.
std::string canonical_json(const Manifest& m);

// Compute desired_sha256 = SHA256(canonical_json(m)).
std::string desired_sha256(const Manifest& m);

struct LoadResult {
    bool ok = false;
    Manifest manifest;
    std::string desired_sha256;
    Diagnostic error;
};

// load-desired-manifest: read, resolve format, parse (JSON or safe-profile
// YAML), schema-validate, reject non-empty observational scopes, optionally
// verify signature, compute desired_sha256.
LoadResult load_desired_manifest(const std::string& manifest_path,
                                 const std::optional<ManifestFormat>& fmt,
                                 const Config& cfg);

// Parse a captured actual-state dump (Manifest) for diff/verify state-path.
// Schema-validated; observational scopes are tolerated here (a dump may carry
// them). Returns a manifest with present scopes as found.
LoadResult load_state_dump(const std::string& state_path,
                           const std::optional<ManifestFormat>& fmt,
                           const Config& cfg);

struct AppliedLoad {
    bool ok = false;       // false only on a present-but-corrupt record
    AppliedRecord record;  // all scopes empty if absent
    bool present = false;
    Diagnostic error;
};

// load-applied-record: read <root>/usr/lib/zypper-declarative/applied.json.
AppliedLoad load_applied_record(const std::string& root);

}  // namespace zd

#endif  // ZD_MANIFEST_HPP
