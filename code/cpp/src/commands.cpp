// generated from spec: zypper-declarative.spec.md sha256:f2cc80627e483a48bb8411d297711bc5f6c6e74c28dbf0dafc8fe7bd8817251e
//
// commands.cpp -- the five verbs. STEPS are implemented in spec order.
#include "commands.hpp"
#include "manifest.hpp"
#include "diff.hpp"
#include "transaction.hpp"

#include <iostream>
#include <sstream>

namespace zd {

static void emit(const Diagnostic& d) {
    std::cerr << (d.severity == Severity::Error ? "error" : "warning")
              << ": " << d.domain << ": " << d.message << "\n";
}

static KeepList keep_of(const Config& cfg) {
    if (cfg.keep_list) return load_keep_list(*cfg.keep_list);
    return KeepList{};
}

static std::string manifest_path_of(const Config& cfg) {
    if (cfg.manifest_path) return *cfg.manifest_path;
    return "/var/lib/zypper-declarative/desired.json"; // delivery-layer staging default
}

// Obtain the actual state: from state_path (offline) or via describe-actual-state.
// Returns exit code != -1 on a hard error (already reported); -1 on success.
static int obtain_actual(const Config& cfg, const CommandRunner& runner,
                         const SystemReader* reader, ScanScope scope,
                         Manifest& actual_out, bool& read_live) {
    (void)runner;
    if (cfg.state_path) {
        read_live = false;
        LoadResult sr = load_state_dump(*cfg.state_path, cfg.explicit_format, cfg);
        if (!sr.ok) { emit(sr.error); return 2; }
        actual_out = sr.manifest;
        return -1;
    }
    read_live = true;
    ActualStateResult as = describe_actual_state(cfg.root.empty() ? "/" : cfg.root,
                                                 OnUnreadable::Error, scope,
                                                 keep_of(cfg), reader);
    if (!as.ok) { emit(as.error); return 1; }
    actual_out = as.manifest;
    return -1;
}

// ------------------------------------------------------------------ diff ----
int cmd_diff(const Config& cfg, const CommandRunner& runner, const SystemReader* reader) {
    // STEP 1: load desired manifest
    LoadResult dr = load_desired_manifest(manifest_path_of(cfg), cfg.explicit_format, cfg);
    if (!dr.ok) { emit(dr.error); return dr.error.domain == "invocation" ? 2 : 1; }

    // STEP 2: applied record (absence -> all empty). In offline state-path mode
    // the reference for the intent diff is still the applied record (default "/")
    // unless not present; the drift portion uses the state file.
    AppliedResult ar = load_applied_record(cfg.applied_root);
    if (!ar.ok) { emit(ar.error); return 1; }

    // STEP 3: intent diff
    Diff d = compute_intent_diff(dr.manifest, ar.record);

    // STEP 4: actual state for drift
    Manifest actual; bool live = true;
    int rc = obtain_actual(cfg, runner, reader, ScanScope::Etc, actual, live);
    if (rc != -1) return rc;
    DriftReport drift = compute_drift(actual, dr.manifest, keep_of(cfg));

    // STEP 5: print plan
    std::ostringstream os;
    os << "packages to install:\n";
    for (auto& p : d.packages_install) os << "  + " << p.name << "\n";
    os << "packages to remove:\n";
    for (auto& p : d.packages_remove) os << "  - " << p.name << "\n";
    os << "repositories to set:\n";
    for (auto& r : d.repos_set) os << "  = " << r.alias << "\n";
    os << "files to write:\n";
    for (auto& f : d.files_write) os << "  > " << f.name << "\n";
    os << "files to delete:\n";
    for (auto& f : d.files_delete) os << "  x " << f << "\n";
    os << "units to change:\n";
    for (auto& u : d.units_change) os << "  ~ " << u.name << " -> " << u.state << "\n";
    os << "drift: " << (drift.empty() ? "clean" : std::to_string(drift.count()) + " item(s)")
       << "\n";
    std::cout << os.str();
    return 0;
}

// ----------------------------------------------------------------- verify ----
int cmd_verify(const Config& cfg, const CommandRunner& runner, const SystemReader* reader) {
    // STEP 1: determine the reference
    AppliedRecord reference;
    if (cfg.manifest_path) {
        LoadResult dr = load_desired_manifest(*cfg.manifest_path, cfg.explicit_format, cfg);
        if (!dr.ok) {
            emit(dr.error);
            return dr.error.domain == "invocation" ? 2 : 2; // read/format -> 2 per spec
        }
        reference = dr.manifest;
    } else {
        AppliedResult ar = load_applied_record(cfg.applied_root);
        if (!ar.ok) { emit(ar.error); return 1; }
        if (!ar.present) {
            std::cerr << "error: invocation: no declaration applied\n";
            return 2;
        }
        reference = ar.record;
    }

    // STEP 2: actual state
    Manifest actual; bool live = true;
    int rc = obtain_actual(cfg, runner, reader, cfg.scope, actual, live);
    if (rc != -1) return rc;

    // STEP 3: drift
    DriftReport drift = compute_drift(actual, reference, keep_of(cfg));

    // STEP 4
    if (drift.empty()) {
        std::cout << "system matches declaration\n";
        return 0;
    }
    for (auto& p : drift.files_modified)
        emit(Diagnostic{Severity::Error, "files", "modified: " + p});
    for (auto& p : drift.files_extra)
        emit(Diagnostic{Severity::Error, "files", "extra: " + p});
    for (auto& u : drift.units_divergent)
        emit(Diagnostic{Severity::Error, "units", "divergent unit: " + u.name});
    for (auto& p : drift.packages_divergent)
        emit(Diagnostic{Severity::Error, "packages", "divergent package: " + p.name});
    for (auto& p : drift.managed_files_modified)
        emit(Diagnostic{Severity::Error, "files", "managed file modified: " + p});
    for (auto& p : drift.unmanaged_files_present)
        emit(Diagnostic{Severity::Error, "files", "unmanaged file present: " + p});
    return 1;
}

// ----------------------------------------------------------------- status ----
int cmd_status(const Config& cfg, const CommandRunner& runner, const SystemReader* reader) {
    // STEP 2: load applied record
    AppliedResult ar = load_applied_record(cfg.applied_root);
    if (!ar.ok) { emit(ar.error); return 1; }
    if (!ar.present) {
        std::cout << "no declaration applied\n";
        return 0;
    }
    // STEP 3
    std::cout << "desired_sha256: " << ar.record.meta.desired_sha256 << "\n";
    std::cout << "format_version: " << ar.record.meta.format_version << "\n";
    std::cout << "generation: " << (ar.record.meta.created_at.empty()
                                    ? "current" : ar.record.meta.created_at) << "\n";
    std::cout << "created_at: " << ar.record.meta.created_at << "\n";
    size_t pkgcount = ar.record.packages ? ar.record.packages->elements.size() : 0;
    std::cout << "packages (resolved lock): " << pkgcount << "\n";

    // STEP 4: drift summary
    Manifest actual; bool live = true;
    int rc = obtain_actual(cfg, runner, reader, ScanScope::Etc, actual, live);
    if (rc != -1) return rc;
    DriftReport drift = compute_drift(actual, ar.record, keep_of(cfg));
    std::cout << "drift: "
              << (drift.empty() ? "clean" : std::to_string(drift.count()) + " drift item(s)")
              << "\n";
    return 0;
}

// --------------------------------------------------------------- describe ----
int cmd_describe(const Config& cfg, const CommandRunner& runner, const SystemReader* reader) {
    (void)runner;
    // STEP 2: obtain actual state with on_unreadable and scope
    ActualStateResult as = describe_actual_state(cfg.root.empty() ? "/" : cfg.root,
                                                 cfg.on_unreadable, cfg.scope,
                                                 keep_of(cfg), reader);
    for (auto& d : as.diagnostics) emit(d);
    if (!as.ok) { emit(as.error); return 1; }

    // STEP 3: resolve output format against out
    ManifestFormat fmt = resolve_format(cfg.explicit_format, cfg.out, cfg.manifest_format);

    // STEP 4 + 5: serialise and write
    if (!write_manifest(as.manifest, fmt, cfg.out)) {
        emit(Diagnostic{Severity::Error, "invocation",
                        "output path unwritable: " + (cfg.out ? *cfg.out : std::string("stdout"))});
        return 2;
    }
    return 0;
}

// ------------------------------------------------------------------ apply ----
int cmd_apply(const Config& cfg, const CommandRunner& runner, const SystemReader* reader) {
    // STEP 1: load desired manifest
    LoadResult dr = load_desired_manifest(manifest_path_of(cfg), cfg.explicit_format, cfg);
    if (!dr.ok) { emit(dr.error); return dr.error.domain == "invocation" ? 2 : 1; }

    // STEP 2: applied record
    AppliedResult ar = load_applied_record(cfg.applied_root);
    if (!ar.ok) { emit(ar.error); return 1; }

    // STEP 3: intent diff
    Diff d = compute_intent_diff(dr.manifest, ar.record);

    // STEP 4: if intent diff empty, check drift; if also empty, nothing to do.
    if (d.empty()) {
        ActualStateResult as = describe_actual_state(cfg.applied_root, OnUnreadable::Error,
                                                     ScanScope::Etc, keep_of(cfg), reader);
        if (!as.ok) { emit(as.error); return 1; }
        DriftReport drift = compute_drift(as.manifest, dr.manifest, keep_of(cfg));
        if (drift.empty()) {
            std::cout << "nothing to do\n";
            return 0;
        }
    }

    // STEP 5: acquire transaction context
    TxnResult tr = acquire_transaction_context(cfg.transaction_mode, runner);
    if (!tr.ok) { emit(tr.error); return 2; }

    // STEP 6: repositories then converge packages
    PackagesConvergeResult pc = converge_packages(tr.ctx, d, runner);
    if (!pc.ok) { emit(pc.error); return 1; }

    // STEP 7: converge files
    std::string content_store = cfg.content_store ? *cfg.content_store : "/";
    ConvergeResult fr = converge_files(tr.ctx, d, keep_of(cfg), content_store, runner);
    if (!fr.ok) { emit(fr.error); return 1; }

    // STEP 8: converge units
    ConvergeResult ur = converge_units(tr.ctx, d, runner);
    if (!ur.ok) { emit(ur.error); return 1; }

    // STEP 9: write applied record
    ConvergeResult wr = write_applied_record(tr.ctx, dr.manifest, dr.desired_sha256,
                                             pc.resolved, runner);
    if (!wr.ok) { emit(wr.error); return 1; }

    // STEP 10: post-converge verification
    ActualStateResult post = describe_actual_state(tr.ctx.root, OnUnreadable::Error,
                                                   ScanScope::Etc, keep_of(cfg), reader);
    if (!post.ok) { emit(post.error); return 1; }
    AppliedRecord newrec = dr.manifest;
    newrec.packages = pc.resolved;
    DriftReport pdrift = compute_drift(post.manifest, newrec, keep_of(cfg));
    if (!pdrift.empty()) {
        emit(Diagnostic{Severity::Error, "files", "post-converge verification found drift"});
        return 1;
    }

    // STEP 11: seal and activate, emit summary
    std::cout << "converged: " << d.packages_install.size() << " package(s), "
              << d.files_write.size() << " file(s) written\n";
    return 0;
}

} // namespace zd
