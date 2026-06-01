// generated from spec: zypper-declarative.spec.md sha256:f2cc80627e483a48bb8411d297711bc5f6c6e74c28dbf0dafc8fe7bd8817251e
//
// diff.cpp -- compute-intent-diff and compute-drift. Pure comparisons over two
// in-memory Manifest values: no filesystem, rpmdb, or process I/O.
#include "diff.hpp"

#include <map>
#include <set>
#include <fstream>
#include <algorithm>

namespace zd {

static const char* kSyncpoint = "/etc/etc.syncpoint";

KeepList load_keep_list(const std::string& path) {
    KeepList kl;
    std::ifstream f(path);
    if (!f) return kl;
    std::string line;
    while (std::getline(f, line)) {
        // strip comments and whitespace
        auto hash = line.find('#');
        if (hash != std::string::npos) line = line.substr(0, hash);
        auto b = line.find_first_not_of(" \t\r\n");
        if (b == std::string::npos) continue;
        auto e = line.find_last_not_of(" \t\r\n");
        kl.insert(line.substr(b, e - b + 1));
    }
    return kl;
}

// BEHAVIOR/INTERNAL: compute-intent-diff
Diff compute_intent_diff(const Manifest& desired, const AppliedRecord& applied) {
    Diff d;

    // STEP 1: packages
    if (desired.packages) {
        d.packages_install = desired.packages->elements;
        std::set<std::string> desired_names;
        for (auto& p : desired.packages->elements) desired_names.insert(p.name);
        if (applied.packages) {
            for (auto& p : applied.packages->elements)
                if (desired_names.find(p.name) == desired_names.end())
                    d.packages_remove.push_back(p);
        }
    }
    // STEP 2: repositories
    if (desired.repositories)
        d.repos_set = desired.repositories->elements;

    // STEP 3: config_files
    if (desired.config_files) {
        d.files_write = desired.config_files->elements;
        std::set<std::string> desired_paths;
        for (auto& f : desired.config_files->elements) desired_paths.insert(f.name);
        if (applied.config_files) {
            for (auto& f : applied.config_files->elements)
                if (desired_paths.find(f.name) == desired_paths.end())
                    d.files_delete.push_back(f.name);  // (declared_old - declared_new)
        }
    }
    // STEP 4: services
    if (desired.services) {
        std::map<std::string, std::string> applied_state;
        if (applied.services)
            for (auto& s : applied.services->elements) applied_state[s.name] = s.state;
        for (auto& s : desired.services->elements) {
            auto it = applied_state.find(s.name);
            if (it == applied_state.end() || it->second != s.state)
                d.units_change.push_back(s);
        }
    }
    return d;
}

static bool is_keep(const KeepList& keep, const std::string& path) {
    return keep.find(path) != keep.end();
}

// BEHAVIOR/INTERNAL: compute-drift
DriftReport compute_drift(const Manifest& actual, const AppliedRecord& reference,
                          const KeepList& keep) {
    DriftReport r;

    // index actual config_files by name
    std::map<std::string, const ManagedFileRecord*> actual_files;
    if (actual.config_files)
        for (auto& a : actual.config_files->elements) actual_files[a.name] = &a;

    // STEP 1: files_modified (type is part of identity)
    std::set<std::string> reference_file_names;
    if (reference.config_files) {
        for (auto& e : reference.config_files->elements) {
            reference_file_names.insert(e.name);
            auto it = actual_files.find(e.name);
            if (it == actual_files.end()) continue; // absent in actual = matching
            const ManagedFileRecord& a = *it->second;
            bool modified = false;
            if (a.type != e.type) modified = true;               // type transition
            else if (e.type == "file" && a.sha256 != e.sha256) modified = true;
            else if (e.type == "link" && a.target != e.target) modified = true;
            if (modified) r.files_modified.push_back(e.name);
        }
    }

    // STEP 2: files_extra (unpackaged, undeclared, not keep-listed, not syncpoint)
    if (actual.config_files) {
        for (auto& a : actual.config_files->elements) {
            if (reference_file_names.find(a.name) != reference_file_names.end()) continue;
            if (!a.package_name.empty()) continue;   // package-managed -> not extra
            if (a.name == kSyncpoint) continue;
            if (is_keep(keep, a.name)) continue;
            r.files_extra.push_back(a.name);
        }
    }

    // STEP 3: units_divergent
    if (reference.services) {
        std::map<std::string, std::string> actual_state;
        if (actual.services)
            for (auto& s : actual.services->elements) actual_state[s.name] = s.state;
        for (auto& u : reference.services->elements) {
            auto it = actual_state.find(u.name);
            if (it != actual_state.end() && it->second != u.state)
                r.units_divergent.push_back(u);
        }
    }

    // STEP 4: packages_divergent (identity fields; present in one but not the other)
    {
        auto pkg_key = [](const PackageRecord& p) {
            return p.name + "\x1f" + p.version + "\x1f" + p.release + "\x1f" + p.arch;
        };
        std::set<std::string> ref_keys, act_keys;
        std::map<std::string, PackageRecord> ref_by_key, act_by_key;
        if (reference.packages)
            for (auto& p : reference.packages->elements) {
                ref_keys.insert(pkg_key(p)); ref_by_key[pkg_key(p)] = p;
            }
        if (actual.packages)
            for (auto& p : actual.packages->elements) {
                act_keys.insert(pkg_key(p)); act_by_key[pkg_key(p)] = p;
            }
        // Only compare packages when the reference actually declares the scope.
        if (reference.packages) {
            for (auto& k : ref_keys)
                if (act_keys.find(k) == act_keys.end())
                    r.packages_divergent.push_back(ref_by_key[k]);
            for (auto& k : act_keys)
                if (ref_keys.find(k) == ref_keys.end())
                    r.packages_divergent.push_back(act_by_key[k]);
        }
    }

    // STEP 5: integrity categories (full scan)
    if (actual.changed_managed_files)
        for (auto& e : actual.changed_managed_files->elements)
            if (!is_keep(keep, e.name)) r.managed_files_modified.push_back(e.name);
    if (actual.unmanaged_files)
        for (auto& e : actual.unmanaged_files->elements)
            if (!is_keep(keep, e.name)) r.unmanaged_files_present.push_back(e.name);

    return r;
}

} // namespace zd
