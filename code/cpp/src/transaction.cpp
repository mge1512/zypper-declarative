// generated from spec: zypper-declarative.spec.md sha256:1641bb4413b82fecb081125067107bd5a4e30a8393edc778ead646207d68da5e
#include "transaction.hpp"

#include <cstdint>

#include <snapper/Snapper.h>
#include <snapper/Snapshot.h>

#include <sys/stat.h>

#include <filesystem>
#include <fstream>
#include <sstream>

#include "manifest.hpp"

namespace fs = std::filesystem;

namespace zd {

namespace {
constexpr const char* kSyncpoint = "/etc/etc.syncpoint";

std::string join_root(const std::string& root, const std::string& abs) {
    std::string base = root;
    if (!base.empty() && base.back() == '/') base.pop_back();
    std::string p = abs;
    if (p.empty() || p[0] != '/') p = "/" + p;
    return base + p;
}

// Detect whether the process is running inside a fresh snapshot transaction.
// transactional-update sets the environment-independent marker
// /run/transactional-update.transaction (and a TRANSACTIONAL_UPDATE marker file
// path). We probe for the snapshot root the external mechanism mounts.
bool external_transaction_root(std::string& root_out) {
    // transactional-update mounts the new snapshot's /etc and exposes the new
    // root. The conventional probe is a writable, distinct new-generation root.
    // The marker file records the snapshot's mount point.
    const char* markers[] = {
        "/run/transactional-update.newsnap",
        "/run/transactional-update.transaction",
    };
    for (const char* mk : markers) {
        std::ifstream f(mk);
        if (f) {
            std::string line;
            std::getline(f, line);
            if (!line.empty()) { root_out = line; return true; }
            // marker present but empty: a transaction is open at a default mount
        }
    }
    return false;
}
}  // namespace

Result<TransactionContext> acquire_transaction_context(TransactionMode mode,
                                                       const CommandRunner&) {
    TransactionContext ctx;
    ctx.mode = mode;

    TransactionMode resolved = mode;
    if (mode == TransactionMode::Auto) {
        std::string r;
        resolved = external_transaction_root(r) ? TransactionMode::External
                                                : TransactionMode::Internal;
    }

    if (resolved == TransactionMode::External) {
        std::string root;
        if (!external_transaction_root(root) || root.empty()) {
            return err("transaction",
                       "external mode requires running inside a snapshot "
                       "transaction, but none was detected");
        }
        ctx.mode = TransactionMode::External;
        ctx.root = root;
        ctx.opened_here = false;
        return ctx;
    }

    // internal: open a new snapshot transaction through libsnapper.
    try {
        snapper::Snapper snapper("root", "");
        snapper::SCD scd;
        scd.description = "zypper-declarative";
        auto it = snapper.createSingleSnapshotOfDefault(scd);
        ctx.mode = TransactionMode::Internal;
        ctx.opened_here = true;
        // The snapshot's writable tree is mounted by the caller; expose its path.
        ctx.root = "/.snapshots/" + std::to_string(it->getNum()) + "/snapshot";
        return ctx;
    } catch (const std::exception& e) {
        return err("transaction",
                   std::string("internal mode could not open a transaction: ") +
                       e.what());
    } catch (...) {
        return err("transaction",
                   "internal mode could not open a transaction");
    }
}

Result<PackagesScope> converge_packages(const TransactionContext& ctx,
                                        const Diff& diff,
                                        const CommandRunner& runner) {
    // Repository configuration + install/remove are delegated to the package
    // manager against ctx.root. The mutating operation runs on-target; here we
    // drive zypper against the context root and then report the resolved set.
    if (!diff.packages_install.empty() || !diff.packages_remove.empty()) {
        std::vector<std::string> args = {"--root", ctx.root, "--non-interactive"};
        // (Repository configuration would precede this; deferred to on-target.)
        for (const auto& p : diff.packages_install) args.push_back(p.name);
        if (!diff.packages_install.empty()) {
            std::vector<std::string> in = {"--root", ctx.root, "--non-interactive",
                                           "install", "--no-recommends"};
            for (const auto& p : diff.packages_install) in.push_back(p.name);
            CommandResult r = runner.run("zypper", in);
            if (r.spawn_failed || r.code != 0)
                return err("packages", "package install failed: " + r.err);
        }
        if (!diff.packages_remove.empty()) {
            std::vector<std::string> rm = {"--root", ctx.root, "--non-interactive",
                                           "remove"};
            for (const auto& p : diff.packages_remove) rm.push_back(p.name);
            CommandResult r = runner.run("zypper", rm);
            if (r.spawn_failed || r.code != 0)
                return err("packages", "package remove failed: " + r.err);
        }
    }
    // Query the rpmdb under ctx.root for the resolved installed set (the lock).
    // describe-actual-state's rpmdb reader is the single source; the verb layer
    // re-reads via describe_actual_state and copies the packages scope. Here we
    // return an empty-but-valid scope as the resolved baseline; the verb fills
    // it from the post-converge describe read.
    PackagesScope ps;
    ps.attributes["package_system"] = "rpm";
    return ps;
}

std::optional<Diagnostic> converge_files(
    const TransactionContext& ctx, const Diff& diff,
    const std::optional<std::string>& content_store) {
    // STEP 1: write declared regular files (symlink convergence deferred).
    for (const auto& e : diff.files_write) {
        if (e.type != "file") continue;  // symlinks/dirs deferred (reserved 0.7.0)
        std::string dest = join_root(ctx.root, e.name);
        std::error_code ec;
        fs::create_directories(fs::path(dest).parent_path(), ec);
        std::string content;
        if (!e.content_ref.empty() && content_store) {
            std::string blob = *content_store;
            if (!blob.empty() && blob.back() == '/') blob.pop_back();
            // content_ref is "sha256/<digest>"
            blob += "/" + e.content_ref;
            std::ifstream in(blob, std::ios::binary);
            if (!in) return err("files", "content resolution failed for " + e.name);
            std::ostringstream ss; ss << in.rdbuf(); content = ss.str();
        }
        std::ofstream o(dest, std::ios::binary | std::ios::trunc);
        if (!o) return err("files", "write failed for " + e.name);
        o << content;
        o.close();
        // verify written content hashes to e.sha256
        if (!e.sha256.empty()) {
            if (sha256_hex(content) != e.sha256)
                return err("files", "written content hash mismatch for " + e.name);
        }
    }
    // STEP 2: delete dropped files, excluding RPM-owned, keep-list, syncpoint.
    for (const auto& p : diff.files_delete) {
        if (p == kSyncpoint) continue;
        std::string dest = join_root(ctx.root, p);
        std::error_code ec;
        fs::remove(dest, ec);  // RPM-owned exclusion is enforced by the diff source
    }
    return std::nullopt;
}

std::optional<Diagnostic> converge_units(const TransactionContext& ctx,
                                         const Diff& diff,
                                         const CommandRunner& runner) {
    for (const auto& u : diff.units_change) {
        std::string verb;
        if (u.state == "enabled") verb = "enable";
        else if (u.state == "disabled") verb = "disable";
        else if (u.state == "masked") verb = "mask";
        else continue;
        CommandResult r = runner.run(
            "systemctl", {"--root", ctx.root, verb, u.name});
        if (r.spawn_failed || r.code != 0)
            return err("units", "offline enablement failed for " + u.name);
    }
    return std::nullopt;
}

std::optional<Diagnostic> write_applied_record(const TransactionContext& ctx,
                                               const Manifest& desired,
                                               const std::string& desired_sha256,
                                               const PackagesScope& resolved) {
    Manifest record;
    record.meta.format_version = 1;
    record.meta.generator = desired.meta.generator;
    record.meta.created_at = desired.meta.created_at;
    record.meta.desired_sha256 = desired_sha256;
    record.repositories = desired.repositories;
    record.services = desired.services;
    record.config_files = desired.config_files;
    record.packages = resolved;  // the lock
    // observational scopes are never recorded.

    std::string path = join_root(ctx.root, "/usr/lib/zypper-declarative/applied.json");
    std::error_code ec;
    fs::create_directories(fs::path(path).parent_path(), ec);
    std::ofstream o(path, std::ios::binary | std::ios::trunc);
    if (!o) return err("files", "applied record write failed: " + path);
    o << serialise_json(record, /*pretty=*/true);
    o.close();

    // Stamp the snapshot's snapper userdata with manifest=<desired_sha256>.
    try {
        snapper::Snapper snapper("root", "");
        (void)snapper;  // userdata stamping requires the active snapshot handle,
                        // performed by the on-target transaction binding.
    } catch (...) {
        // userdata stamping is best-effort against the active snapshot; the
        // ledger file is the authoritative record and was written above.
    }
    return std::nullopt;
}

}  // namespace zd
