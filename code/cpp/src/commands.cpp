// generated from spec: zypper-declarative.spec.md sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03

#include "commands.hpp"

#include <fstream>
#include <iostream>
#include <set>
#include <sstream>

#include "converge.hpp"
#include "describe.hpp"
#include "diff.hpp"
#include "manifest.hpp"

namespace zd {

namespace {

// Map an internal Diagnostic domain to the process exit code per the spec.
int exit_for(const Diagnostic& d) {
    if (d.domain == "invocation" || d.domain == "transaction") return 2;
    return 1;  // manifest, packages, files, units, repositories -> logical fail
}

void emit(const Diagnostic& d) {
    std::cerr << (d.severity == Severity::Error ? "error" : "warning") << ": ["
              << d.domain << "] " << d.message << "\n";
}

std::set<std::string> load_keep_list(const std::string& path) {
    std::set<std::string> out;
    if (path.empty()) return out;
    std::ifstream f(path);
    std::string line;
    while (std::getline(f, line)) {
        size_t a = line.find_first_not_of(" \t\r\n");
        if (a == std::string::npos) continue;
        if (line[a] == '#') continue;
        size_t b = line.find_last_not_of(" \t\r\n");
        out.insert(line.substr(a, b - a + 1));
    }
    return out;
}

// Obtain the actual state for the drift portion: from a supplied state dump
// (offline) or via describe-actual-state on "/".
bool obtain_actual(const Config& cfg, const CommandRunner& runner,
                   ScanScope scope, Manifest& actual, int& errcode) {
    if (cfg.state_path_given && cfg.state_path) {
        LoadResult lr = load_state_dump(*cfg.state_path, cfg.explicit_format, cfg);
        if (!lr.ok) {
            emit(lr.error);
            errcode = exit_for(lr.error);
            return false;
        }
        actual = lr.manifest;
        return true;
    }
    std::set<std::string> keep = load_keep_list(cfg.keep_list);
    DescribeResult dr = describe_actual_state(cfg.root, /*strict=*/true, scope,
                                              keep, cfg.content_store, runner);
    if (!dr.ok) {
        emit(dr.error);
        errcode = exit_for(dr.error);
        return false;
    }
    actual = dr.manifest;
    return true;
}

}  // namespace

// --------------------------------------------------------------------------
// describe
// --------------------------------------------------------------------------
int cmd_describe(const Config& cfg, const CommandRunner& runner) {
    std::set<std::string> keep = load_keep_list(cfg.keep_list);
    DescribeResult dr = describe_actual_state(cfg.root, cfg.on_unreadable_error,
                                              cfg.scope, keep, cfg.content_store,
                                              runner);
    for (const auto& d : dr.diagnostics) emit(d);
    if (!dr.ok) {
        emit(dr.error);
        return exit_for(dr.error);
    }
    ManifestFormat fmt = resolve_format(
        cfg.explicit_format,
        cfg.out.empty() ? std::optional<std::string>() : cfg.out,
        cfg.manifest_format);
    std::string doc = serialise(dr.manifest, fmt);
    if (cfg.out.empty()) {
        std::cout << doc;
        return 0;
    }
    std::ofstream out(cfg.out, std::ios::binary | std::ios::trunc);
    if (!out.good()) {
        emit({Severity::Error, "invocation",
              "output path unwritable: " + cfg.out});
        return 2;
    }
    out << doc;
    return 0;
}

// --------------------------------------------------------------------------
// diff
// --------------------------------------------------------------------------
int cmd_diff(const Config& cfg, const CommandRunner& runner) {
    LoadResult lr =
        load_desired_manifest(cfg.manifest_path, cfg.explicit_format, cfg);
    if (!lr.ok) {
        emit(lr.error);
        return exit_for(lr.error);
    }
    AppliedLoad al = load_applied_record(cfg.applied_root);
    if (!al.ok) {
        emit(al.error);
        return exit_for(al.error);
    }
    Diff d = compute_intent_diff(lr.manifest, al.record);
    Manifest actual;
    int errcode = 1;
    if (!obtain_actual(cfg, runner, ScanScope::Etc, actual, errcode))
        return errcode;
    std::set<std::string> keep = load_keep_list(cfg.keep_list);
    DriftReport drift = compute_drift(actual, al.record, keep);

    std::ostringstream o;
    o << "packages to install:\n";
    for (const auto& p : d.packages_install) o << "  " << p.name << "\n";
    o << "packages to remove:\n";
    for (const auto& p : d.packages_remove) o << "  " << p.name << "\n";
    o << "repositories to set:\n";
    for (const auto& r : d.repos_set) o << "  " << r.alias << "\n";
    o << "files to write:\n";
    for (const auto& f : d.files_write) o << "  " << f.name << "\n";
    o << "files to delete:\n";
    for (const auto& p : d.files_delete) o << "  " << p << "\n";
    o << "units to change:\n";
    for (const auto& u : d.units_change) o << "  " << u.name << " -> " << u.state
                                           << "\n";
    o << "drift:\n";
    for (const auto& p : drift.files_modified) o << "  modified " << p << "\n";
    for (const auto& p : drift.files_extra) o << "  extra " << p << "\n";
    for (const auto& u : drift.units_divergent)
        o << "  unit " << u.name << "\n";
    std::cout << o.str();
    return 0;
}

// --------------------------------------------------------------------------
// verify
// --------------------------------------------------------------------------
int cmd_verify(const Config& cfg, const CommandRunner& runner) {
    AppliedRecord reference;
    bool have_reference = false;
    if (cfg.manifest_path_given && !cfg.manifest_path.empty()) {
        LoadResult lr =
            load_desired_manifest(cfg.manifest_path, cfg.explicit_format, cfg);
        if (!lr.ok) {
            emit(lr.error);
            // read/format -> 2, else 1
            return lr.error.domain == "invocation" ? 2 : 1;
        }
        reference = lr.manifest;
        have_reference = true;
    } else {
        AppliedLoad al = load_applied_record(cfg.applied_root);
        if (!al.ok) {
            emit(al.error);
            return exit_for(al.error);
        }
        if (!al.present) {
            emit({Severity::Error, "invocation", "no declaration applied"});
            return 2;
        }
        reference = al.record;
        have_reference = true;
    }
    (void)have_reference;

    Manifest actual;
    int errcode = 1;
    if (!obtain_actual(cfg, runner, cfg.scope, actual, errcode)) return errcode;

    std::set<std::string> keep = load_keep_list(cfg.keep_list);
    DriftReport drift = compute_drift(actual, reference, keep);
    if (drift.empty()) {
        std::cout << "system matches declaration\n";
        return 0;
    }
    for (const auto& p : drift.files_modified)
        emit({Severity::Error, "files", "drift: modified " + p});
    for (const auto& p : drift.files_extra)
        emit({Severity::Error, "files", "drift: extra " + p});
    for (const auto& u : drift.units_divergent)
        emit({Severity::Error, "units", "drift: unit " + u.name});
    for (const auto& p : drift.packages_divergent)
        emit({Severity::Error, "packages", "drift: package " + p.name});
    for (const auto& p : drift.managed_files_modified)
        emit({Severity::Error, "files", "integrity: modified " + p});
    for (const auto& p : drift.unmanaged_files_present)
        emit({Severity::Error, "files", "integrity: unmanaged " + p});
    return 1;
}

// --------------------------------------------------------------------------
// status
// --------------------------------------------------------------------------
int cmd_status(const Config& cfg, const CommandRunner& runner) {
    AppliedLoad al = load_applied_record(cfg.applied_root);
    if (!al.ok) {
        emit(al.error);
        return exit_for(al.error);
    }
    if (!al.present) {
        std::cout << "no declaration applied\n";
        return 0;
    }
    size_t pkgcount =
        al.record.packages ? al.record.packages->elements.size() : 0;
    std::cout << "desired_sha256: " << al.record.meta.desired_sha256 << "\n";
    std::cout << "format_version: " << al.record.meta.format_version << "\n";
    std::cout << "generation: current\n";
    std::cout << "created_at: " << al.record.meta.created_at << "\n";
    std::cout << "packages: " << pkgcount << "\n";

    std::set<std::string> keep = load_keep_list(cfg.keep_list);
    DescribeResult dr = describe_actual_state(cfg.applied_root, true,
                                              ScanScope::Etc, keep, "", runner);
    if (dr.ok) {
        DriftReport drift = compute_drift(dr.manifest, al.record, keep);
        size_t n = drift.files_modified.size() + drift.files_extra.size() +
                   drift.units_divergent.size() +
                   drift.packages_divergent.size();
        if (n == 0)
            std::cout << "clean\n";
        else
            std::cout << n << " drift item(s)\n";
    } else {
        std::cout << "clean\n";
    }
    return 0;
}

// --------------------------------------------------------------------------
// apply
// --------------------------------------------------------------------------
int cmd_apply(const Config& cfg, const CommandRunner& runner) {
    LoadResult lr =
        load_desired_manifest(cfg.manifest_path, cfg.explicit_format, cfg);
    if (!lr.ok) {
        emit(lr.error);
        return exit_for(lr.error);
    }
    AppliedLoad al = load_applied_record(cfg.applied_root);
    if (!al.ok) {
        emit(al.error);
        return exit_for(al.error);
    }
    Diff d = compute_intent_diff(lr.manifest, al.record);
    std::set<std::string> keep = load_keep_list(cfg.keep_list);

    bool intent_empty = d.packages_install.empty() && d.packages_remove.empty() &&
                        d.repos_set.empty() && d.files_write.empty() &&
                        d.files_delete.empty() && d.units_change.empty();
    if (intent_empty) {
        DescribeResult dr = describe_actual_state(cfg.root, true, ScanScope::Etc,
                                                  keep, "", runner);
        if (dr.ok) {
            DriftReport drift = compute_drift(dr.manifest, al.record, keep);
            if (drift.empty()) {
                std::cout << "nothing to do\n";
                return 0;
            }
        }
        // If we cannot read the live state here (unprivileged/no rpmdb access),
        // we still report nothing-to-do only when the intent is empty AND drift
        // could be computed empty; otherwise fall through to the transaction.
    }

    // Acquire the transaction context (mutating; deferred on a live target).
    Result<TransactionContext> ctxr =
        acquire_transaction_context(cfg.transaction_mode, runner);
    if (!ctxr.ok) {
        emit(ctxr.error);
        return exit_for(ctxr.error);
    }
    const TransactionContext& ctx = ctxr.value;

    Result<ScopeWrapper<PackageRecord>> pkgs =
        converge_packages(ctx, d, cfg, runner);
    if (!pkgs.ok) {
        emit(pkgs.error);
        return exit_for(pkgs.error);
    }
    Status fs_st = converge_files(ctx, d, cfg, runner);
    if (!fs_st.ok) {
        emit(fs_st.error);
        return exit_for(fs_st.error);
    }
    Status u_st = converge_units(ctx, d, runner);
    if (!u_st.ok) {
        emit(u_st.error);
        return exit_for(u_st.error);
    }
    Status w_st =
        write_applied_record(ctx, lr.manifest, lr.desired_sha256, pkgs.value,
                             runner);
    if (!w_st.ok) {
        emit(w_st.error);
        return exit_for(w_st.error);
    }
    // Post-converge verification.
    DescribeResult post = describe_actual_state(ctx.root, true, ScanScope::Etc,
                                                keep, "", runner);
    if (post.ok) {
        DriftReport drift = compute_drift(post.manifest, lr.manifest, keep);
        if (!drift.empty()) {
            emit({Severity::Error, "files",
                  "post-converge verification found drift; discarded"});
            return 1;
        }
    }
    std::cout << "applied: converged to declaration\n";
    return 0;
}

}  // namespace zd
