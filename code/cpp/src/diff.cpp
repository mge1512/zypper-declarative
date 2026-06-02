// generated from spec: zypper-declarative.spec.md sha256:aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3
#include "diff.hpp"

#include <algorithm>
#include <map>

namespace zd {

namespace {
constexpr const char* kSyncpoint = "/etc/etc.syncpoint";

template <class T>
const std::vector<T>& scope_els(const std::optional<ScopeWrapper<T>>& s) {
    static const std::vector<T> empty;
    return s ? s->elements : empty;
}
}  // namespace

// ----------------------------------------------------------------------
// compute-intent-diff
// ----------------------------------------------------------------------
Diff compute_intent_diff(const Manifest& desired, const Manifest& applied) {
    Diff d;

    // STEP 1: packages
    if (desired.packages) {
        d.packages_install = desired.packages->elements;
        std::set<std::string> desired_names;
        for (const auto& p : desired.packages->elements) desired_names.insert(p.name);
        for (const auto& p : scope_els(applied.packages))
            if (desired_names.find(p.name) == desired_names.end())
                d.packages_remove.push_back(p);
    }

    // STEP 2: repositories
    if (desired.repositories)
        d.repos_set = desired.repositories->elements;

    // STEP 3: config_files
    if (desired.config_files) {
        d.files_write = desired.config_files->elements;
        std::set<std::string> desired_paths;
        for (const auto& f : desired.config_files->elements)
            desired_paths.insert(f.name);
        for (const auto& f : scope_els(applied.config_files))
            if (desired_paths.find(f.name) == desired_paths.end())
                d.files_delete.push_back(f.name);  // (declared_old - declared_new)
    }

    // STEP 4: services
    if (desired.services) {
        std::map<std::string, std::string> applied_states;
        for (const auto& s : scope_els(applied.services))
            applied_states[s.name] = s.state;
        for (const auto& s : desired.services->elements) {
            auto it = applied_states.find(s.name);
            if (it == applied_states.end() || it->second != s.state)
                d.units_change.push_back(s);
        }
    }

    return d;
}

// ----------------------------------------------------------------------
// compute-drift
// ----------------------------------------------------------------------
DriftReport compute_drift(const Manifest& actual, const Manifest& reference,
                          const KeepList& keep_list) {
    DriftReport rep;

    auto keep_listed = [&](const std::string& p) {
        return keep_list.find(p) != keep_list.end();
    };

    // STEP 1: files_modified — for each reference config file, compare against
    // the actual record of the same name. Type is part of identity.
    std::map<std::string, const ManagedFileRecord*> actual_files;
    for (const auto& a : scope_els(actual.config_files))
        actual_files[a.name] = &a;
    for (const auto& e : scope_els(reference.config_files)) {
        auto it = actual_files.find(e.name);
        if (it == actual_files.end()) continue;  // absent in actual = matching
        const ManagedFileRecord& a = *it->second;
        bool modified = false;
        if (a.type != e.type) {
            modified = true;  // type transition
        } else if (e.type == "file") {
            modified = (a.sha256 != e.sha256);
        } else if (e.type == "link") {
            modified = (a.target != e.target);
        }
        if (modified) rep.files_modified.push_back(e.name);
    }

    // STEP 2: files_extra — actual config files absent from reference, with
    // package_name "" (unpackaged), not keep-listed, not the syncpoint.
    std::set<std::string> ref_file_names;
    for (const auto& e : scope_els(reference.config_files))
        ref_file_names.insert(e.name);
    for (const auto& a : scope_els(actual.config_files)) {
        if (ref_file_names.find(a.name) != ref_file_names.end()) continue;
        if (!a.package_name.empty()) continue;          // package-managed, not extra
        if (a.name == kSyncpoint) continue;
        if (keep_listed(a.name)) continue;
        rep.files_extra.push_back(a.name);
    }

    // STEP 3: units_divergent — reference services whose actual state differs.
    std::map<std::string, std::string> actual_states;
    for (const auto& u : scope_els(actual.services)) actual_states[u.name] = u.state;
    for (const auto& u : scope_els(reference.services)) {
        auto it = actual_states.find(u.name);
        if (it == actual_states.end() || it->second != u.state)
            rep.units_divergent.push_back(u);
    }

    // STEP 4: packages_divergent — a reference (declared) package is divergent
    // when the actual state contains no package satisfying it. Comparison is on
    // the declarable IDENTITY fields only, and an EMPTY reference field is a
    // wildcard: a desired package carrying name only (version/release/arch "")
    // is satisfied by ANY installed package of that name (TYPES: "" version in
    // a desired manifest = newest from the pinned repo). Each non-empty
    // reference field must match the actual record's corresponding field.
    //
    // The comparison is one-directional: the reference declares which packages
    // must be present; an installed package the reference does not name is not
    // declaration drift in this scope (the full-scan integrity categories cover
    // out-of-band additions). Package drift is only computed when the reference
    // declares the packages scope at all.
    if (reference.packages) {
        auto satisfied_by = [](const PackageRecord& want,
                               const PackageRecord& have) {
            if (want.name != have.name) return false;
            if (!want.version.empty() && want.version != have.version) return false;
            if (!want.release.empty() && want.release != have.release) return false;
            if (!want.arch.empty() && want.arch != have.arch) return false;
            return true;
        };
        for (const auto& want : reference.packages->elements) {
            bool ok = false;
            for (const auto& have : scope_els(actual.packages))
                if (satisfied_by(want, have)) { ok = true; break; }
            if (!ok) rep.packages_divergent.push_back(want);
        }
    }

    // STEP 5: integrity categories (full scan) — presence is drift.
    if (actual.changed_managed_files)
        for (const auto& e : actual.changed_managed_files->elements)
            if (!keep_listed(e.name))
                rep.managed_files_modified.push_back(e.name);
    if (actual.unmanaged_files)
        for (const auto& e : actual.unmanaged_files->elements)
            if (!keep_listed(e.name))
                rep.unmanaged_files_present.push_back(e.name);

    return rep;
}

}  // namespace zd
