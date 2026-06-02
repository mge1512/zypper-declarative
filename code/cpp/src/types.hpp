// generated from spec: zypper-declarative.spec.md sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
//
// Data model for the declarable subset of the SUSE Machinery system
// description: packages, repositories, services, config_files, plus the two
// observational scopes (changed_managed_files, unmanaged_files). The
// ScopeWrapper idiom carries _attributes (always a JSON object) and _elements.
#ifndef ZD_TYPES_HPP
#define ZD_TYPES_HPP

#include <cstdint>
#include <map>
#include <optional>
#include <string>
#include <vector>

namespace zd {

enum class Severity { Error, Warning };

struct Diagnostic {
    Severity severity = Severity::Error;
    std::string domain;   // packages|repositories|services|files|units|
                          // manifest|transaction|invocation
    std::string message;
};

// ScopeWrapper<T>: _attributes is ALWAYS a JSON object (empty {} when none,
// never null). _elements holds the records of the scope.
template <class T>
struct ScopeWrapper {
    std::map<std::string, std::string> attributes;  // "_attributes"
    std::vector<T> elements;                          // "_elements"
};

struct PackageRecord {
    std::string name;
    std::string version;   // "" = newest from pinned repo in a desired manifest
    std::string release;   // "" unless pinning an exact build
    std::string arch;      // "" = native / any
};

struct RepositoryRecord {
    std::string alias;
    std::string name;
    std::string url;
    std::string type;          // e.g. "rpm-md"
    bool enabled = false;
    bool gpgcheck = false;
    bool autorefresh = false;
    long priority = 0;
};

struct ServiceRecord {
    std::string name;    // UnitName
    std::string state;   // enabled|disabled|masked
};

struct ManagedFileRecord {
    std::string name;          // AbsolutePath under /etc
    std::string type;          // file|link|dir
    std::string mode;          // Mode
    std::string user;
    std::string group;
    std::string sha256;        // for type=file; "" otherwise
    std::string target;        // for type=link (verbatim); "" otherwise
    std::string content_ref;   // "sha256/<digest>" or ""
    std::string package_name;  // bare owning package name; "" if unpackaged
};

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
    std::string name;          // path OUTSIDE /etc, unpackaged
    std::string type;
    std::string mode;
    std::string user;
    std::string group;
    std::string sha256;
    std::string target;
};

struct ManifestMeta {
    int format_version = 1;
    std::string generator;
    std::string created_at;
    std::string desired_sha256;  // "" except in an applied record
};

// A declarable scope ABSENT means unmanaged (std::nullopt). A PRESENT scope
// with empty _elements means reconcile-to-empty. The two are never collapsed.
struct Manifest {
    ManifestMeta meta;
    std::optional<ScopeWrapper<PackageRecord>> packages;
    std::optional<ScopeWrapper<RepositoryRecord>> repositories;
    std::optional<ScopeWrapper<ServiceRecord>> services;
    std::optional<ScopeWrapper<ManagedFileRecord>> config_files;
    // Observational, never declarable.
    std::optional<ScopeWrapper<ManagedBaselineRecord>> changed_managed_files;
    std::optional<ScopeWrapper<UnmanagedFileRecord>> unmanaged_files;
};

using AppliedRecord = Manifest;

struct Diff {
    std::vector<PackageRecord> packages_install;
    std::vector<PackageRecord> packages_remove;
    std::vector<RepositoryRecord> repos_set;
    std::vector<ManagedFileRecord> files_write;
    std::vector<std::string> files_delete;        // AbsolutePath
    std::vector<ServiceRecord> units_change;
};

struct DriftReport {
    std::vector<std::string> files_modified;
    std::vector<std::string> files_extra;
    std::vector<ServiceRecord> units_divergent;
    std::vector<PackageRecord> packages_divergent;
    std::vector<std::string> managed_files_modified;
    std::vector<std::string> unmanaged_files_present;

    bool empty() const {
        return files_modified.empty() && files_extra.empty() &&
               units_divergent.empty() && packages_divergent.empty() &&
               managed_files_modified.empty() &&
               unmanaged_files_present.empty();
    }
};

enum class ManifestFormat { Json, Yaml };
enum class TransactionMode { Auto, External, Internal };
enum class ScanScope { Etc, Full };

struct TransactionContext {
    TransactionMode mode = TransactionMode::Auto;
    std::string root;       // AbsolutePath; mount point of new snapshot root
    bool opened_here = false;
};

}  // namespace zd

#endif  // ZD_TYPES_HPP
