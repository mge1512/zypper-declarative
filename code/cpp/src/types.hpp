// generated from spec: zypper-declarative.spec.md sha256:f2cc80627e483a48bb8411d297711bc5f6c6e74c28dbf0dafc8fe7bd8817251e
//
// types.hpp -- the shared data model: the declarable subset of the SUSE
// Machinery system description (packages, repositories, services,
// config_files), plus the observational scopes, the Diff, the DriftReport,
// Diagnostic, and the enums. underscore_style field names are produced by the
// serialiser (serialize.cpp), not by these C++ member names.
#ifndef ZD_TYPES_HPP
#define ZD_TYPES_HPP

#include <string>
#include <vector>
#include <map>
#include <optional>
#include <cstdint>

namespace zd {

// ---- Refinement-typed string aliases (validated where parsed) --------------
using AbsolutePath = std::string;
using Sha256 = std::string;

// ---- Shared scope idiom (Machinery / sitar convention) ---------------------
template <class T>
struct ScopeWrapper {
    // "_attributes": object | null. We model the known string attributes
    // (package_system, repository_system, init_system). A genuine JSON null is
    // tracked by has_attributes=false for config_files which carries null.
    std::map<std::string, std::string> attributes;
    bool has_attributes = true;   // false -> emit "_attributes": null
    std::vector<T> elements;       // "_elements"
};

struct ManifestMeta {
    int format_version = 1;
    std::string generator;
    std::string created_at;
    Sha256 desired_sha256;   // "" except in an applied record
};

// ---- Declarable scopes -----------------------------------------------------
struct PackageRecord {
    std::string name;     // non-empty
    std::string version;  // "" = newest from pinned repo
    std::string release;  // "" unless pinning an exact build
    std::string arch;     // "" = native / any
};

struct RepositoryRecord {
    std::string alias;     // non-empty
    std::string name;
    std::string url;       // non-empty
    std::string type;      // e.g. "rpm-md"
    bool enabled = true;
    bool gpgcheck = true;
    bool autorefresh = false;
    int priority = 99;
};

struct ServiceRecord {
    std::string name;   // UnitName
    std::string state;  // "enabled" | "disabled" | "masked"
};

struct ManagedFileRecord {
    AbsolutePath name;       // starts_with("/etc/")
    std::string type;        // "file" | "link" | "dir"
    std::string mode;        // Mode "^[0-7]{3,4}$"
    std::string user;        // non-empty
    std::string group;       // non-empty
    Sha256 sha256;           // for type=file; "" otherwise
    std::string target;      // for type=link (verbatim); "" otherwise
    std::string content_ref; // for a DESIRED type=file; "" in describe output
    std::string package_name;// owning package; "" if unpackaged
};

// ---- Observational scopes (full-scan integrity, not declarable) ------------
struct ManagedBaselineRecord {
    AbsolutePath name;       // path OUTSIDE /etc
    std::string type;        // "file" | "link" | "dir"
    std::string mode;
    std::string user;
    std::string group;
    Sha256 sha256;           // for type=file; "" otherwise
    std::string target;      // for type=link; "" otherwise
    std::string package_name;// owning package (always set)
    std::vector<std::string> changes; // what differs from package baseline
};

struct UnmanagedFileRecord {
    AbsolutePath name;       // path OUTSIDE /etc, unowned
    std::string type;        // "file" | "link" | "dir"
    std::string mode;
    std::string user;
    std::string group;
    Sha256 sha256;           // for type=file; "" otherwise
    std::string target;      // for type=link; "" otherwise
};

using PackagesScope = ScopeWrapper<PackageRecord>;
using RepositoriesScope = ScopeWrapper<RepositoryRecord>;
using ServicesScope = ScopeWrapper<ServiceRecord>;
using ConfigFilesScope = ScopeWrapper<ManagedFileRecord>;
using ChangedManagedFilesScope = ScopeWrapper<ManagedBaselineRecord>;
using UnmanagedFilesScope = ScopeWrapper<UnmanagedFileRecord>;

// ---- Manifest --------------------------------------------------------------
// std::optional distinguishes ABSENT (unmanaged) from PRESENT-but-empty
// (reconcile-to-empty). nullopt = absent.
struct Manifest {
    ManifestMeta meta;
    std::optional<PackagesScope> packages;
    std::optional<RepositoriesScope> repositories;
    std::optional<ServicesScope> services;
    std::optional<ConfigFilesScope> config_files;
    // Observational, never declarable. Present only in describe/verify under
    // scope=full; ignored by diff and convergence; never in a desired manifest
    // or applied record.
    std::optional<ChangedManagedFilesScope> changed_managed_files;
    std::optional<UnmanagedFilesScope> unmanaged_files;
};

// AppliedRecord is a Manifest with packages fully resolved and desired_sha256 set.
using AppliedRecord = Manifest;

// ---- Transaction binding ---------------------------------------------------
enum class TransactionMode { Auto, External, Internal };

struct TransactionContext {
    TransactionMode mode = TransactionMode::Auto;
    AbsolutePath root;     // mount point of the new snapshot's root tree
    bool opened_here = false;
};

// ---- Diff and drift --------------------------------------------------------
struct Diff {
    std::vector<PackageRecord> packages_install;
    std::vector<PackageRecord> packages_remove;
    std::vector<RepositoryRecord> repos_set;
    std::vector<ManagedFileRecord> files_write;
    std::vector<AbsolutePath> files_delete;
    std::vector<ServiceRecord> units_change;

    bool empty() const {
        return packages_install.empty() && packages_remove.empty() &&
               repos_set.empty() && files_write.empty() &&
               files_delete.empty() && units_change.empty();
    }
};

struct DriftReport {
    std::vector<AbsolutePath> files_modified;
    std::vector<AbsolutePath> files_extra;
    std::vector<ServiceRecord> units_divergent;
    std::vector<PackageRecord> packages_divergent;
    // full-scan integrity categories
    std::vector<AbsolutePath> managed_files_modified;
    std::vector<AbsolutePath> unmanaged_files_present;

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

// ---- Diagnostics -----------------------------------------------------------
enum class Severity { Error, Warning };

struct Diagnostic {
    Severity severity = Severity::Error;
    std::string domain;   // packages|repositories|services|files|manifest|transaction|invocation|units
    std::string message;
};

// ExitCode: 0 success, 1 logical failure, 2 invocation error.

enum class ManifestFormat { Json, Yaml };

enum class ScanScope { Etc, Full };
enum class OnUnreadable { Error, Warn };

} // namespace zd

#endif // ZD_TYPES_HPP
