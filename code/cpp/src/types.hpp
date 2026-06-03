// generated from spec: zypper-declarative.spec.md sha256:aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3
#pragma once

#include <string>
#include <vector>
#include <map>
#include <optional>

namespace zd {

// ---------------------------------------------------------------------------
// Shared scope idiom (Machinery / sitar convention): _attributes + _elements.
// _attributes is ALWAYS serialised as a JSON object, empty {} when no
// attributes, never null.
// ---------------------------------------------------------------------------
template <class T>
struct ScopeWrapper {
    std::map<std::string, std::string> attributes;  // "_attributes"
    std::vector<T> elements;                         // "_elements"
};

struct ManifestMeta {
    int format_version = 1;     // always 1 (Machinery JSON format)
    std::string generator;      // program name and version
    std::string created_at;     // RFC3339, informational only
    std::string desired_sha256; // canonical-model hash; "" except in applied record
};

// Declarable scopes -------------------------------------------------------

struct PackageRecord {
    std::string name;     // non-empty
    std::string version;  // "" in desired = newest from pinned repo
    std::string release;  // "" unless pinning an exact build
    std::string arch;     // "" = native / any
};

struct RepositoryRecord {
    std::string alias;        // non-empty
    std::string name;
    std::string url;          // non-empty
    std::string type;         // e.g. "rpm-md"
    bool enabled = true;
    bool gpgcheck = true;
    bool autorefresh = false;
    int priority = 99;
};

struct ServiceRecord {
    std::string name;   // UnitName
    std::string state;  // enabled | disabled | masked
};

struct ManagedFileRecord {
    std::string name;          // AbsolutePath under /etc
    std::string type;          // file | link | dir
    std::string mode;          // octal mode string
    std::string user;
    std::string group;
    std::string sha256;        // for type=file: a Sha256 content digest; "" otherwise
    std::string target;        // for type=link: verbatim symlink target; "" otherwise
    std::string content_ref;   // "sha256/<digest>" when content store in use; "" otherwise
    std::string package_name;  // owning package bare name; "" if unpackaged
};

// Observational scopes (full-scan integrity, never declarable) -------------

struct ManagedBaselineRecord {
    std::string name;          // path OUTSIDE /etc
    std::string type;
    std::string mode;
    std::string user;
    std::string group;
    std::string sha256;
    std::string target;
    std::string package_name;  // always set
    std::vector<std::string> changes;
};

struct UnmanagedFileRecord {
    std::string name;
    std::string type;
    std::string mode;
    std::string user;
    std::string group;
    std::string sha256;
    std::string target;
};

using PackagesScope     = ScopeWrapper<PackageRecord>;
using RepositoriesScope = ScopeWrapper<RepositoryRecord>;
using ServicesScope     = ScopeWrapper<ServiceRecord>;
using ConfigFilesScope  = ScopeWrapper<ManagedFileRecord>;
using ChangedManagedFilesScope = ScopeWrapper<ManagedBaselineRecord>;
using UnmanagedFilesScope      = ScopeWrapper<UnmanagedFileRecord>;

struct Manifest {
    ManifestMeta meta;
    // Declarable scopes; nullopt = absent (unmanaged), present+empty = reconcile-to-empty.
    std::optional<PackagesScope>     packages;
    std::optional<RepositoriesScope> repositories;
    std::optional<ServicesScope>     services;
    std::optional<ConfigFilesScope>  config_files;
    // Observational, only present under scope=full describe/verify actual state.
    std::optional<ChangedManagedFilesScope> changed_managed_files;
    std::optional<UnmanagedFilesScope>      unmanaged_files;
};

// Transaction binding ------------------------------------------------------

enum class TransactionMode { Auto, External, Internal };

struct TransactionContext {
    TransactionMode mode = TransactionMode::Auto;
    std::string root;        // mount point of the new snapshot's root tree
    bool opened_here = false;
    std::string snapshot_id; // identifier of the snapshot (if any)
};

// Diff and drift -----------------------------------------------------------

struct Diff {
    std::vector<PackageRecord>    packages_install;
    std::vector<PackageRecord>    packages_remove;
    std::vector<RepositoryRecord> repos_set;
    std::vector<ManagedFileRecord> files_write;
    std::vector<std::string>      files_delete;
    std::vector<ServiceRecord>    units_change;
};

struct DriftReport {
    std::vector<std::string>   files_modified;
    std::vector<std::string>   files_extra;
    std::vector<ServiceRecord> units_divergent;
    std::vector<PackageRecord> packages_divergent;
    std::vector<std::string>   managed_files_modified;
    std::vector<std::string>   unmanaged_files_present;

    bool empty() const {
        return files_modified.empty() && files_extra.empty() &&
               units_divergent.empty() && packages_divergent.empty() &&
               managed_files_modified.empty() && unmanaged_files_present.empty();
    }
};

enum class ScanScope { Etc, Full };
enum class ManifestFormat { Json, Yaml };
enum class OnUnreadable { Error, Warn };

}  // namespace zd
