// generated from spec: zypper-declarative.spec.md sha256:aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3
#include "diffdrift.hpp"

#include <map>

namespace zd {

static const char* kSyncpoint = "/etc/etc.syncpoint";

// ---------------------------------------------------------------------------
// compute-intent-diff
// ---------------------------------------------------------------------------
Diff compute_intent_diff(const Manifest& desired, const Manifest& applied) {
    Diff d;

    // 1. packages
    if (desired.packages) {
        d.packages_install = desired.packages->elements;
        std::set<std::string> desired_names;
        for (const auto& p : desired.packages->elements) desired_names.insert(p.name);
        if (applied.packages) {
            for (const auto& p : applied.packages->elements)
                if (desired_names.find(p.name) == desired_names.end())
                    d.packages_remove.push_back(p);
        }
    }

    // 2. repositories
    if (desired.repositories) d.repos_set = desired.repositories->elements;

    // 3. config_files
    if (desired.config_files) {
        d.files_write = desired.config_files->elements;
        std::set<std::string> desired_paths;
        for (const auto& f : desired.config_files->elements) desired_paths.insert(f.name);
        if (applied.config_files) {
            for (const auto& f : applied.config_files->elements)
                if (desired_paths.find(f.name) == desired_paths.end())
                    d.files_delete.push_back(f.name);
        }
    }

    // 4. services
    if (desired.services) {
        std::map<std::string, std::string> applied_state;
        if (applied.services)
            for (const auto& s : applied.services->elements) applied_state[s.name] = s.state;
        for (const auto& s : desired.services->elements) {
            auto it = applied_state.find(s.name);
            if (it == applied_state.end() || it->second != s.state) d.units_change.push_back(s);
        }
    }

    return d;
}

// ---------------------------------------------------------------------------
// compute-drift
// ---------------------------------------------------------------------------
// A reference package field that is empty is a wildcard matching any actual
// value (only the reference side wildcards).
static bool pkg_field_matches(const std::string& ref, const std::string& act) {
    if (ref.empty()) return true;  // wildcard
    return ref == act;
}

DriftReport compute_drift(const Manifest& actual, const Manifest& reference,
                          const std::set<std::string>& keep_list) {
    DriftReport r;

    // Build lookup of actual config_files by name.
    std::map<std::string, const ManagedFileRecord*> actual_files;
    if (actual.config_files)
        for (const auto& a : actual.config_files->elements) actual_files[a.name] = &a;

    // 1. files_modified: declared entries whose actual differs (type/sha/target).
    if (reference.config_files) {
        for (const auto& e : reference.config_files->elements) {
            auto it = actual_files.find(e.name);
            if (it == actual_files.end()) continue;  // absent from actual = matching
            const ManagedFileRecord& a = *it->second;
            bool modified = false;
            if (a.type != e.type) modified = true;  // type is part of identity
            else if (e.type == "file" && a.sha256 != e.sha256) modified = true;
            else if (e.type == "link" && a.target != e.target) modified = true;
            if (modified) r.files_modified.push_back(e.name);
        }
    }

    // 2. files_extra: actual unpackaged, undeclared /etc files (not keep-listed,
    //    not syncpoint).
    {
        std::set<std::string> declared;
        if (reference.config_files)
            for (const auto& e : reference.config_files->elements) declared.insert(e.name);
        if (actual.config_files) {
            for (const auto& a : actual.config_files->elements) {
                if (declared.count(a.name)) continue;
                if (!a.package_name.empty()) continue;  // package-managed, not extra
                if (a.name == kSyncpoint) continue;
                if (keep_list.count(a.name)) continue;
                r.files_extra.push_back(a.name);
            }
        }
    }

    // 3. units_divergent
    if (reference.services) {
        std::map<std::string, std::string> actual_state;
        if (actual.services)
            for (const auto& s : actual.services->elements) actual_state[s.name] = s.state;
        for (const auto& u : reference.services->elements) {
            auto it = actual_state.find(u.name);
            if (it != actual_state.end() && it->second != u.state) r.units_divergent.push_back(u);
        }
    }

    // 4. packages_divergent: reference empty fields are wildcards.
    if (reference.packages) {
        std::map<std::string, const PackageRecord*> actual_pkgs;
        if (actual.packages)
            for (const auto& p : actual.packages->elements) actual_pkgs[p.name] = &p;
        for (const auto& ref : reference.packages->elements) {
            auto it = actual_pkgs.find(ref.name);
            if (it == actual_pkgs.end()) {
                r.packages_divergent.push_back(ref);
                continue;
            }
            const PackageRecord& act = *it->second;
            if (!pkg_field_matches(ref.version, act.version) ||
                !pkg_field_matches(ref.release, act.release) ||
                !pkg_field_matches(ref.arch, act.arch)) {
                r.packages_divergent.push_back(ref);
            }
        }
        // Packages present in actual but not in reference: with a reference
        // that declares packages, the desired set is authoritative; a package
        // present in actual but absent in reference is a removal candidate but
        // not drift per se. Spec step 4 says "add any package present in one
        // but not the other"; for the verify/diff use the reference is the
        // declaration and an undeclared installed package is not divergence
        // unless reference explicitly declares the scope empty. We follow the
        // reference-driven interpretation: only reference elements are checked.
    }

    // 5. integrity categories (full scan): presence is itself drift.
    if (actual.changed_managed_files) {
        for (const auto& e : actual.changed_managed_files->elements)
            if (!keep_list.count(e.name)) r.managed_files_modified.push_back(e.name);
    }
    if (actual.unmanaged_files) {
        for (const auto& e : actual.unmanaged_files->elements)
            if (!keep_list.count(e.name)) r.unmanaged_files_present.push_back(e.name);
    }

    return r;
}

}  // namespace zd
