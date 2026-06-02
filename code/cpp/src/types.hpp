// generated from spec: zypper-declarative.spec.md sha256:aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3
//
// Data model for the declarable subset of the SUSE Machinery system
// description (TYPES section of the spec). The on-the-wire field names are the
// Machinery underscore_style keys; the C++ member names mirror them. Scopes
// use the ScopeWrapper { _attributes, _elements } idiom. An ABSENT declarable
// scope is std::nullopt (unmanaged); a PRESENT-but-empty scope has empty
// _elements (reconcile-to-empty). The two are never collapsed.
#ifndef ZD_TYPES_HPP
#define ZD_TYPES_HPP

#include <map>
#include <optional>
#include <string>
#include <vector>

namespace zd {

// ScopeWrapper<T>: _attributes is ALWAYS a JSON object (empty {} when no
// attributes, never null); _elements holds the scope records.
template <class T>
struct ScopeWrapper {
    std::map<std::string, std::string> attributes;  // "_attributes"
    std::vector<T> elements;                          // "_elements"
};

// ManifestMeta
struct ManifestMeta {
    int format_version = 1;
    std::string generator;       // "zypper-declarative <version>"
    std::string created_at;      // RFC3339, informational
    std::string desired_sha256;  // canonical-model hash; "" except in applied record
};

// PackageRecord (identity subset)
struct PackageRecord {
    std::string name;
    std::string version;  // "" = newest from pinned repo in a desired manifest
    std::string release;  // "" unless pinning an exact build
    std::string arch;     // "" = native/any
};

// RepositoryRecord (zypp repository)
struct RepositoryRecord {
    std::string alias;
    std::string name;
    std::string url;
    std::string type;          // e.g. "rpm-md"
    bool enabled = true;
    bool gpgcheck = true;
    bool autorefresh = false;
    long priority = 99;
};

// ServiceRecord (declarable states only)
struct ServiceRecord {
    std::string name;   // UnitName
    std::string state;  // enabled | disabled | masked
};

// ManagedFileRecord (declarable superset of Machinery changed-config-files;
// confined to /etc in v1).
struct ManagedFileRecord {
    std::string name;          // AbsolutePath under /etc
    std::string type;          // file | link | dir
    std::string mode;          // octal string
    std::string user;
    std::string group;
    std::string sha256;        // for type=file; "" otherwise
    std::string target;        // for type=link: verbatim target; "" otherwise
    std::string content_ref;   // "sha256/<digest>" when content store in use; "" otherwise
    std::string package_name;  // bare owning package name; "" if unpackaged
};

// ManagedBaselineRecord (observational: changed packaged file outside /etc)
struct ManagedBaselineRecord {
    std::string name;          // path OUTSIDE /etc
    std::string type;          // file | link | dir
    std::string mode;
    std::string user;
    std::string group;
    std::string sha256;        // for type=file; "" otherwise
    std::string target;        // for type=link; "" otherwise
    std::string package_name;  // owning package (always set)
    std::vector<std::string> changes;  // what differs from baseline
};

// UnmanagedFileRecord (observational: file outside /etc no package owns)
struct UnmanagedFileRecord {
    std::string name;
    std::string type;
    std::string mode;
    std::string user;
    std::string group;
    std::string sha256;
    std::string target;
};

using PackagesScope            = ScopeWrapper<PackageRecord>;
using RepositoriesScope        = ScopeWrapper<RepositoryRecord>;
using ServicesScope            = ScopeWrapper<ServiceRecord>;
using ConfigFilesScope         = ScopeWrapper<ManagedFileRecord>;
using ChangedManagedFilesScope = ScopeWrapper<ManagedBaselineRecord>;
using UnmanagedFilesScope      = ScopeWrapper<UnmanagedFileRecord>;

// Manifest: meta + the four declarable scopes (optional = absent/unmanaged) +
// the two observational scopes (present only in describe/verify scope=full).
struct Manifest {
    ManifestMeta meta;
    std::optional<PackagesScope> packages;
    std::optional<RepositoriesScope> repositories;
    std::optional<ServicesScope> services;
    std::optional<ConfigFilesScope> config_files;
    std::optional<ChangedManagedFilesScope> changed_managed_files;  // observational
    std::optional<UnmanagedFilesScope> unmanaged_files;             // observational
};

// Diff: the intent diff (desired_new vs applied_old).
struct Diff {
    std::vector<PackageRecord> packages_install;
    std::vector<PackageRecord> packages_remove;
    std::vector<RepositoryRecord> repos_set;
    std::vector<ManagedFileRecord> files_write;
    std::vector<std::string> files_delete;       // AbsolutePath
    std::vector<ServiceRecord> units_change;
};

// DriftReport: actual vs declared.
struct DriftReport {
    std::vector<std::string> files_modified;
    std::vector<std::string> files_extra;
    std::vector<ServiceRecord> units_divergent;
    std::vector<PackageRecord> packages_divergent;
    std::vector<std::string> managed_files_modified;    // full-scan only
    std::vector<std::string> unmanaged_files_present;   // full-scan only

    bool empty() const {
        return files_modified.empty() && files_extra.empty() &&
               units_divergent.empty() && packages_divergent.empty() &&
               managed_files_modified.empty() && unmanaged_files_present.empty();
    }
    size_t count() const {
        return files_modified.size() + files_extra.size() +
               units_divergent.size() + packages_divergent.size() +
               managed_files_modified.size() + unmanaged_files_present.size();
    }
};

// TransactionMode / TransactionContext
enum class TransactionMode { Auto, External, Internal };

struct TransactionContext {
    TransactionMode mode = TransactionMode::Auto;
    std::string root;          // mount point of the new snapshot's root tree
    bool opened_here = false;  // true if this tool opened the transaction
};

// ManifestFormat: serialisation of the data model.
enum class ManifestFormat { Json, Yaml };

}  // namespace zd

#endif  // ZD_TYPES_HPP
