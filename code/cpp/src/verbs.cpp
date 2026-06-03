// generated from spec: zypper-declarative.spec.md sha256:aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3
#include "verbs.hpp"
#include "manifest_io.hpp"
#include "describe.hpp"
#include "diffdrift.hpp"
#include "transaction.hpp"
#include "converge.hpp"
#include "meta.hpp"

#include <fstream>
#include <set>
#include <sstream>
#include <filesystem>

namespace fs = std::filesystem;

namespace zd {

static const char* kDefaultManifest = "/var/lib/zypper-declarative/desired.json";

static void emit_diag(std::ostream& err, const Diagnostic& d) {
    std::string sev = (d.severity == Severity::Error) ? "error" : "warning";
    err << sev << ": domain=" << d.domain << ": " << d.message << "\n";
}

static std::set<std::string> load_keep_list(const Config& cfg) {
    std::set<std::string> kl;
    if (!cfg.keep_list) return kl;
    std::ifstream f(*cfg.keep_list);
    std::string line;
    while (std::getline(f, line)) {
        auto s = line.find_first_not_of(" \t\r");
        if (s == std::string::npos) continue;
        line = line.substr(s);
        if (line.empty() || line[0] == '#') continue;
        auto e = line.find_last_not_of(" \t\r");
        kl.insert(line.substr(0, e + 1));
    }
    return kl;
}

// Resolve the actual state for diff/verify: either a supplied dump (offline) or
// a live describe read. Returns false (and sets exit) on error.
static bool resolve_actual_state(const Config& cfg, const CommandRunner& runner,
                                 ScanScope scope, OnUnreadable on_unreadable,
                                 std::ostream& err, Manifest& out, int& exit_code) {
    if (cfg.state_path) {
        auto r = load_state_dump(*cfg.state_path, cfg.explicit_format, cfg.manifest_format);
        if (!r.ok()) {
            emit_diag(err, *r.error);
            exit_code = 2;  // malformed dump -> invocation error
            return false;
        }
        out = *r.value;
        return true;
    }
    auto dr = describe_actual_state(cfg.root, on_unreadable, scope, std::nullopt,
                                    load_keep_list(cfg), ZD_GENERATOR, runner);
    if (!dr.ok()) {
        emit_diag(err, *dr.error);
        exit_code = 1;
        return false;
    }
    for (const auto& d : dr.diagnostics) emit_diag(err, d);
    out = dr.manifest;
    return true;
}

// ---------------------------------------------------------------------------
// describe
// ---------------------------------------------------------------------------
int verb_describe(const Config& cfg, const CommandRunner& runner, std::ostream& out,
                  std::ostream& err) {
    auto dr = describe_actual_state(cfg.root, cfg.on_unreadable, cfg.scope, cfg.content_store,
                                    load_keep_list(cfg), ZD_GENERATOR, runner);
    if (!dr.ok()) {
        emit_diag(err, *dr.error);
        return 1;
    }
    for (const auto& d : dr.diagnostics) emit_diag(err, d);

    ManifestFormat fmt = resolve_format(cfg.explicit_format, cfg.out, cfg.manifest_format);
    std::string doc = serialize_manifest(dr.manifest, fmt);

    if (cfg.out) {
        std::error_code ec;
        fs::path p(*cfg.out);
        if (p.has_parent_path() && !fs::exists(p.parent_path(), ec)) {
            err << "error: domain=invocation: output path unwritable: " << *cfg.out << "\n";
            return 2;
        }
        std::ofstream of(*cfg.out, std::ios::binary | std::ios::trunc);
        if (!of.is_open()) {
            err << "error: domain=invocation: output path unwritable: " << *cfg.out << "\n";
            return 2;
        }
        of << doc;
        if (!of) {
            err << "error: domain=invocation: output path unwritable: " << *cfg.out << "\n";
            return 2;
        }
    } else {
        out << doc;
        if (!doc.empty() && doc.back() != '\n') out << "\n";
    }
    return 0;
}

// ---------------------------------------------------------------------------
// diff
// ---------------------------------------------------------------------------
int verb_diff(const Config& cfg, const CommandRunner& runner, std::ostream& out,
              std::ostream& err) {
    std::string mpath = cfg.manifest_path.value_or(kDefaultManifest);
    auto lm = load_desired_manifest(mpath, cfg.explicit_format, cfg.manifest_format,
                                    cfg.signature_verification);
    if (!lm.ok()) {
        emit_diag(err, *lm.error);
        return lm.error->domain == "invocation" ? 2 : 1;
    }
    const Manifest& desired = lm.value->manifest;

    auto applied = load_applied_record(cfg.applied_root);
    if (!applied.ok()) {
        emit_diag(err, *applied.error);
        return 1;
    }

    Diff d = compute_intent_diff(desired, applied.value->record);

    // Actual state for the drift portion, compared against the DESIRED MANIFEST.
    Manifest actual;
    int exit_code = 0;
    if (!resolve_actual_state(cfg, runner, ScanScope::Etc, cfg.on_unreadable, err, actual,
                              exit_code))
        return exit_code;
    DriftReport drift = compute_drift(actual, desired, load_keep_list(cfg));

    // Print the combined plan.
    out << "packages to install:\n";
    for (const auto& p : d.packages_install) out << "  " << p.name << "\n";
    out << "packages to remove:\n";
    for (const auto& p : d.packages_remove) out << "  " << p.name << "\n";
    out << "repositories to set:\n";
    for (const auto& r : d.repos_set) out << "  " << r.alias << "\n";
    out << "files to write:\n";
    for (const auto& f : d.files_write) out << "  " << f.name << "\n";
    out << "files to delete:\n";
    for (const auto& f : d.files_delete) out << "  " << f << "\n";
    out << "units to change:\n";
    for (const auto& u : d.units_change) out << "  " << u.name << " " << u.state << "\n";

    if (!drift.empty()) {
        out << "drift:\n";
        for (const auto& p : drift.files_modified) out << "  modified " << p << "\n";
        for (const auto& p : drift.files_extra) out << "  extra " << p << "\n";
        for (const auto& u : drift.units_divergent) out << "  unit " << u.name << "\n";
        for (const auto& p : drift.packages_divergent) out << "  package " << p.name << "\n";
        for (const auto& p : drift.managed_files_modified) out << "  managed-modified " << p << "\n";
        for (const auto& p : drift.unmanaged_files_present) out << "  unmanaged " << p << "\n";
    }
    return 0;
}

// ---------------------------------------------------------------------------
// verify
// ---------------------------------------------------------------------------
int verb_verify(const Config& cfg, const CommandRunner& runner, std::ostream& out,
                std::ostream& err) {
    // 1. determine reference.
    Manifest reference;
    if (cfg.manifest_path) {
        auto lm = load_desired_manifest(*cfg.manifest_path, cfg.explicit_format,
                                        cfg.manifest_format, cfg.signature_verification);
        if (!lm.ok()) {
            emit_diag(err, *lm.error);
            return lm.error->domain == "invocation" ? 2 : 2;  // verify: read/format -> 2
        }
        reference = lm.value->manifest;
    } else {
        auto applied = load_applied_record(cfg.applied_root);
        if (!applied.ok()) {
            emit_diag(err, *applied.error);
            return 1;
        }
        if (!applied.value->present) {
            err << "error: domain=invocation: no declaration applied\n";
            return 2;
        }
        reference = applied.value->record;
    }

    // 2. actual state.
    Manifest actual;
    int exit_code = 0;
    if (!resolve_actual_state(cfg, runner, cfg.scope, cfg.on_unreadable, err, actual, exit_code))
        return exit_code;

    // Observational scopes are only meaningful under scope=full; under scope=etc
    // a supplied dump's observational scopes are ignored.
    if (cfg.scope == ScanScope::Etc) {
        actual.changed_managed_files.reset();
        actual.unmanaged_files.reset();
    }

    // 3. drift.
    DriftReport drift = compute_drift(actual, reference, load_keep_list(cfg));

    if (drift.empty()) {
        out << "system matches declaration\n";
        return 0;
    }
    for (const auto& p : drift.files_modified)
        err << "error: domain=files: " << p << " differs from declaration\n";
    for (const auto& p : drift.files_extra)
        err << "error: domain=files: " << p << " is an undeclared extra file\n";
    for (const auto& u : drift.units_divergent)
        err << "error: domain=services: " << u.name << " state differs from declaration\n";
    for (const auto& p : drift.packages_divergent)
        err << "error: domain=packages: " << p.name << " differs from declaration\n";
    for (const auto& p : drift.managed_files_modified)
        err << "error: domain=files: " << p << " changed from package baseline\n";
    for (const auto& p : drift.unmanaged_files_present)
        err << "error: domain=files: " << p << " is an unmanaged addition\n";
    return 1;
}

// ---------------------------------------------------------------------------
// status
// ---------------------------------------------------------------------------
int verb_status(const Config& cfg, const CommandRunner& runner, std::ostream& out,
                std::ostream& err) {
    auto applied = load_applied_record(cfg.applied_root);
    if (!applied.ok()) {
        emit_diag(err, *applied.error);
        return 1;
    }
    if (!applied.value->present) {
        out << "no declaration applied\n";
        return 0;
    }
    const Manifest& rec = applied.value->record;
    out << "desired_sha256: " << rec.meta.desired_sha256 << "\n";
    out << "format_version: " << rec.meta.format_version << "\n";
    out << "generation: " << (rec.meta.desired_sha256.empty() ? "unknown" : rec.meta.desired_sha256.substr(0, 12)) << "\n";
    out << "created_at: " << rec.meta.created_at << "\n";
    size_t pkgcount = rec.packages ? rec.packages->elements.size() : 0;
    out << "packages: " << pkgcount << " resolved\n";

    // drift summary: read actual state and compare against the applied record.
    auto dr = describe_actual_state(cfg.root, cfg.on_unreadable, ScanScope::Etc, std::nullopt,
                                    load_keep_list(cfg), ZD_GENERATOR, runner);
    if (!dr.ok()) {
        // status is read-only and fast; a read error is surfaced but does not
        // change the invocation-valid exit. Report it and still exit 0.
        emit_diag(err, *dr.error);
        out << "drift: unknown\n";
        return 0;
    }
    for (const auto& d : dr.diagnostics) emit_diag(err, d);
    DriftReport drift = compute_drift(dr.manifest, rec, load_keep_list(cfg));
    size_t n = drift.files_modified.size() + drift.files_extra.size() +
               drift.units_divergent.size() + drift.packages_divergent.size();
    if (n == 0) out << "drift: clean\n";
    else out << "drift: " << n << " drift item(s)\n";
    return 0;
}

// ---------------------------------------------------------------------------
// apply
// ---------------------------------------------------------------------------
int verb_apply(const Config& cfg, const CommandRunner& runner, std::ostream& out,
               std::ostream& err) {
    // PRECONDITION: the transaction mechanism selected by mode must be
    // available. For mode=external we verify availability up front: if the
    // process is not running inside a snapshot transaction, this is a
    // transaction error (exit 2) regardless of the manifest, matching the
    // apply_transaction_unavailable example.
    if (cfg.transaction_mode == TransactionMode::External) {
        auto ctxr0 = acquire_transaction_context(cfg.transaction_mode);
        if (!ctxr0.ok()) {
            emit_diag(err, *ctxr0.error);
            return 2;
        }
    }

    std::string mpath = cfg.manifest_path.value_or(kDefaultManifest);
    auto lm = load_desired_manifest(mpath, cfg.explicit_format, cfg.manifest_format,
                                    cfg.signature_verification);
    if (!lm.ok()) {
        emit_diag(err, *lm.error);
        return lm.error->domain == "invocation" ? 2 : 1;
    }
    const Manifest& desired = lm.value->manifest;
    const std::string& desired_sha = lm.value->desired_sha256;

    auto applied = load_applied_record(cfg.applied_root);
    if (!applied.ok()) {
        emit_diag(err, *applied.error);
        return 1;
    }

    Diff d = compute_intent_diff(desired, applied.value->record);

    bool intent_empty = d.packages_install.empty() && d.packages_remove.empty() &&
                        d.repos_set.empty() && d.files_write.empty() &&
                        d.files_delete.empty() && d.units_change.empty();

    if (intent_empty) {
        auto dr = describe_actual_state(cfg.root, cfg.on_unreadable, ScanScope::Etc, std::nullopt,
                                        load_keep_list(cfg), ZD_GENERATOR, runner);
        if (!dr.ok()) {
            emit_diag(err, *dr.error);
            return 1;
        }
        for (const auto& diag : dr.diagnostics) emit_diag(err, diag);
        DriftReport drift = compute_drift(dr.manifest, desired, load_keep_list(cfg));
        if (drift.empty()) {
            out << "nothing to do\n";
            return 0;
        }
    }

    // Acquire transaction.
    auto ctxr = acquire_transaction_context(cfg.transaction_mode);
    if (!ctxr.ok()) {
        emit_diag(err, *ctxr.error);
        return 2;
    }
    const TransactionContext& ctx = *ctxr.value;

    // Converge packages.
    auto cp = converge_packages(ctx, d, cfg.repo_lock, runner);
    if (!cp.ok()) { emit_diag(err, *cp.error); return 1; }
    // Converge files.
    auto cf = converge_files(ctx, d, cfg.content_store, load_keep_list(cfg));
    if (!cf.ok()) { emit_diag(err, *cf.error); return 1; }
    // Converge units.
    auto cu = converge_units(ctx, d, runner);
    if (!cu.ok()) { emit_diag(err, *cu.error); return 1; }
    // Write applied record.
    auto wr = write_applied_record(ctx, desired, desired_sha, *cp.value);
    if (!wr.ok()) { emit_diag(err, *wr.error); return 1; }

    // Post-converge verification.
    auto vr = describe_actual_state(ctx.root, cfg.on_unreadable, ScanScope::Etc, std::nullopt,
                                    load_keep_list(cfg), ZD_GENERATOR, runner);
    if (!vr.ok()) { emit_diag(err, *vr.error); return 1; }
    Manifest newrec = desired;
    newrec.packages = *cp.value;
    DriftReport pd = compute_drift(vr.manifest, newrec, load_keep_list(cfg));
    if (!pd.empty()) {
        err << "error: domain=files: post-converge verification found drift\n";
        return 1;
    }

    auto seal = seal_and_activate(ctx, cfg.activation_policy);
    if (!seal.ok()) { emit_diag(err, *seal.error); return 1; }

    out << "applied: " << d.packages_install.size() << " package(s), "
        << d.files_write.size() << " file(s), " << d.units_change.size()
        << " unit(s); snapshot " << (ctx.snapshot_id.empty() ? "external" : ctx.snapshot_id)
        << "\n";
    return 0;
}

// ---------------------------------------------------------------------------
// init
// ---------------------------------------------------------------------------
int verb_init(const Config& cfg, const CommandRunner& runner, std::ostream& out,
              std::ostream& err) {
    // init FORCES on_unreadable=warn for its live read (overrides the default).
    auto dr = describe_actual_state("/", OnUnreadable::Warn, ScanScope::Etc, cfg.content_store,
                                    load_keep_list(cfg), ZD_GENERATOR, runner);
    if (!dr.ok()) {
        emit_diag(err, *dr.error);
        return 1;  // hard state-collection failure
    }
    for (const auto& d : dr.diagnostics) emit_diag(err, d);

    Manifest adopted = dr.manifest;
    std::string adopted_sha = canonical_hash(adopted);

    // Acquire transaction (takes a snapshot).
    auto ctxr = acquire_transaction_context(cfg.transaction_mode);
    if (!ctxr.ok()) {
        emit_diag(err, *ctxr.error);
        return 2;
    }
    const TransactionContext& ctx = *ctxr.value;

    // Write the applied record (adopted state is its own desired). Resolve the
    // packages scope as the adopted packages (already fully resolved by the
    // describe read).
    PackagesScope resolved;
    resolved.attributes["package_system"] = "rpm";
    if (adopted.packages) resolved = *adopted.packages;
    auto wr = write_applied_record(ctx, adopted, adopted_sha, resolved);
    if (!wr.ok()) { emit_diag(err, *wr.error); return 1; }

    // Converge NOTHING.

    // Also write the adopted manifest to `out` (operator-facing).
    std::string outpath = cfg.out.value_or("/var/lib/zypper-declarative/manifest.json");
    ManifestFormat fmt = resolve_format(cfg.explicit_format, outpath, cfg.manifest_format);
    {
        std::error_code ec;
        fs::path p(outpath);
        if (p.has_parent_path()) fs::create_directories(p.parent_path(), ec);
        std::ofstream of(outpath, std::ios::binary | std::ios::trunc);
        if (!of.is_open()) {
            err << "error: domain=invocation: output path unwritable: " << outpath << "\n";
            return 2;
        }
        of << serialize_manifest(adopted, fmt);
    }

    auto seal = seal_and_activate(ctx, cfg.activation_policy);
    if (!seal.ok()) { emit_diag(err, *seal.error); return 1; }

    size_t pkgs = adopted.packages ? adopted.packages->elements.size() : 0;
    size_t files = adopted.config_files ? adopted.config_files->elements.size() : 0;
    out << "adopted: " << pkgs << " package(s), " << files << " config file(s); snapshot "
        << (ctx.snapshot_id.empty() ? "external" : ctx.snapshot_id) << "\n";
    return 0;
}

}  // namespace zd
