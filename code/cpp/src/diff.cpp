// generated from spec: zypper-declarative.spec.md sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03

#include "diff.hpp"

#include <algorithm>
#include <map>

namespace zd {

Diff compute_intent_diff(const Manifest& desired,
                         const AppliedRecord& applied) {
    Diff d;
    // Step 1: packages
    if (desired.packages) {
        d.packages_install = desired.packages->elements;
        std::set<std::string> desired_names;
        for (const auto& p : desired.packages->elements)
            desired_names.insert(p.name);
        if (applied.packages) {
            for (const auto& p : applied.packages->elements)
                if (!desired_names.count(p.name)) d.packages_remove.push_back(p);
        }
    }
    // Step 2: repositories
    if (desired.repositories) d.repos_set = desired.repositories->elements;
    // Step 3: config_files
    if (desired.config_files) {
        d.files_write = desired.config_files->elements;
        std::set<std::string> desired_paths;
        for (const auto& f : desired.config_files->elements)
            desired_paths.insert(f.name);
        if (applied.config_files) {
            for (const auto& f : applied.config_files->elements)
                if (!desired_paths.count(f.name)) d.files_delete.push_back(f.name);
        }
    }
    // Step 4: services
    if (desired.services) {
        std::map<std::string, std::string> applied_state;
        if (applied.services)
            for (const auto& s : applied.services->elements)
                applied_state[s.name] = s.state;
        for (const auto& s : desired.services->elements) {
            auto it = applied_state.find(s.name);
            if (it == applied_state.end() || it->second != s.state)
                d.units_change.push_back(s);
        }
    }
    return d;
}

DriftReport compute_drift(const Manifest& actual,
                          const AppliedRecord& reference,
                          const std::set<std::string>& keep_list) {
    DriftReport r;
    auto is_kept = [&](const std::string& path) {
        return path == "/etc/etc.syncpoint" || keep_list.count(path) > 0;
    };

    // Index actual config_files by name.
    std::map<std::string, ManagedFileRecord> actual_files;
    if (actual.config_files)
        for (const auto& f : actual.config_files->elements)
            actual_files[f.name] = f;

    // Step 1: files_modified — declared entries whose actual diverges.
    std::set<std::string> reference_file_names;
    if (reference.config_files) {
        for (const auto& e : reference.config_files->elements) {
            reference_file_names.insert(e.name);
            auto it = actual_files.find(e.name);
            if (it == actual_files.end())
                continue;  // absent from actual -> treated as matching
            const ManagedFileRecord& a = it->second;
            bool modified = false;
            if (a.type != e.type)
                modified = true;  // type is part of identity
            else if (e.type == "file" && a.sha256 != e.sha256)
                modified = true;
            else if (e.type == "link" && a.target != e.target)
                modified = true;
            if (modified) r.files_modified.push_back(e.name);
        }
    }

    // Step 2: files_extra — unpackaged, undeclared, not keep-listed.
    if (actual.config_files) {
        for (const auto& a : actual.config_files->elements) {
            if (reference_file_names.count(a.name)) continue;
            if (!a.package_name.empty()) continue;  // package-managed, not extra
            if (is_kept(a.name)) continue;
            r.files_extra.push_back(a.name);
        }
    }

    // Step 3: units_divergent.
    if (reference.services) {
        std::map<std::string, std::string> actual_state;
        if (actual.services)
            for (const auto& s : actual.services->elements)
                actual_state[s.name] = s.state;
        for (const auto& u : reference.services->elements) {
            auto it = actual_state.find(u.name);
            if (it != actual_state.end() && it->second != u.state)
                r.units_divergent.push_back(u);
        }
    }

    // Step 4: packages_divergent — present in one but not the other (identity).
    {
        auto pkg_key = [](const PackageRecord& p) {
            return p.name + "\x1f" + p.version + "\x1f" + p.release + "\x1f" +
                   p.arch;
        };
        std::map<std::string, PackageRecord> ref_pkgs, act_pkgs;
        if (reference.packages)
            for (const auto& p : reference.packages->elements)
                ref_pkgs[pkg_key(p)] = p;
        if (actual.packages)
            for (const auto& p : actual.packages->elements)
                act_pkgs[pkg_key(p)] = p;
        if (reference.packages) {
            for (const auto& kv : ref_pkgs)
                if (!act_pkgs.count(kv.first))
                    r.packages_divergent.push_back(kv.second);
            for (const auto& kv : act_pkgs)
                if (!ref_pkgs.count(kv.first))
                    r.packages_divergent.push_back(kv.second);
        }
    }

    // Step 5: integrity categories (full scan) — presence is itself drift.
    if (actual.changed_managed_files)
        for (const auto& e : actual.changed_managed_files->elements)
            if (!is_kept(e.name)) r.managed_files_modified.push_back(e.name);
    if (actual.unmanaged_files)
        for (const auto& e : actual.unmanaged_files->elements)
            if (!is_kept(e.name)) r.unmanaged_files_present.push_back(e.name);

    return r;
}

}  // namespace zd
