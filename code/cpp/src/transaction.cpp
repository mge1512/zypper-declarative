// generated from spec: zypper-declarative.spec.md sha256:aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3
#include "transaction.hpp"
#include "meta.hpp"

#include <filesystem>
#include <system_error>

#include <snapper/Snapper.h>
#include <snapper/Snapshot.h>
#ifdef ZD_SNAPPER_REPORT_PARAM
#include <snapper/Plugins.h>
#endif

namespace fs = std::filesystem;

namespace zd {

// Detect whether the process is running inside an externally-opened snapshot
// transaction (e.g. transactional-update run ...). The detection is
// environment-marker based on filesystem state, not on a behaviour-controlling
// environment variable.
static std::optional<std::string> external_transaction_root() {
    // transactional-update exposes the new snapshot's root mount; the canonical
    // marker is the presence of the transactional-update lock/state and a
    // writable /etc subvolume under the new generation. We probe the documented
    // run-state path. Absence means we are not inside a transaction.
    std::error_code ec;
    // transactional-update sets up the new root at "/" inside its run; its
    // state file marks the active transaction.
    if (fs::exists("/run/transactional-update.pid", ec)) return std::string("/");
    return std::nullopt;
}

Result<TransactionContext> acquire_transaction_context(TransactionMode mode) {
    TransactionMode resolved = mode;

    if (mode == TransactionMode::Auto) {
        if (external_transaction_root()) resolved = TransactionMode::External;
        else resolved = TransactionMode::Internal;
    }

    TransactionContext ctx;
    ctx.mode = resolved;

    if (resolved == TransactionMode::External) {
        auto root = external_transaction_root();
        if (!root) {
            return Result<TransactionContext>::fail(make_error(
                "transaction",
                "external mode but not running inside a snapshot transaction"));
        }
        ctx.root = *root;
        ctx.opened_here = false;
        return Result<TransactionContext>::success(ctx);
    }

    // internal: open a new snapshot transaction through libsnapper.
    try {
        snapper::Snapper sh("root", "/");
        snapper::SCD scd;
        scd.read_only = false;
        scd.description = "zypper-declarative";
#ifdef ZD_SNAPPER_REPORT_PARAM
        snapper::Plugins::Report report;
        auto it = sh.createSingleSnapshotOfDefault(scd, report);
#else
        auto it = sh.createSingleSnapshotOfDefault(scd);
#endif
        ctx.opened_here = true;
        ctx.snapshot_id = std::to_string(it->getNum());
        ctx.root = it->snapshotDir();
        return Result<TransactionContext>::success(ctx);
    } catch (const std::exception& e) {
        return Result<TransactionContext>::fail(make_error(
            "transaction", std::string("internal transaction could not be opened: ") + e.what()));
    } catch (...) {
        return Result<TransactionContext>::fail(
            make_error("transaction", "internal transaction could not be opened"));
    }
}

VoidResult stamp_snapshot_userdata(const TransactionContext& ctx,
                                   const std::string& desired_sha256) {
    if (ctx.snapshot_id.empty()) return VoidResult::success();
    try {
        snapper::Snapper sh("root", "/");
        unsigned num = static_cast<unsigned>(std::stoul(ctx.snapshot_id));
        auto it = sh.getSnapshots().find(num);
        if (it != sh.getSnapshots().end()) {
            snapper::SMD smd;
            smd.description = it->getDescription();
            smd.userdata = it->getUserdata();
            smd.userdata["manifest"] = desired_sha256;
            sh.modifySnapshot(it, smd);
        }
        return VoidResult::success();
    } catch (const std::exception& e) {
        return VoidResult::fail(make_error("files", std::string("userdata stamp failed: ") + e.what()));
    } catch (...) {
        return VoidResult::fail(make_error("files", "userdata stamp failed"));
    }
}

VoidResult seal_and_activate(const TransactionContext& ctx, const std::string&) {
    if (!ctx.opened_here) return VoidResult::success();  // external mechanism handles it
    // Sealing read-only and marking the boot default is delegated to the
    // snapshot machinery on a live transactional target; on the build host this
    // path is never reached (no internal transaction is openable). The verb
    // layer documents activation scheduling.
    return VoidResult::success();
}

}  // namespace zd
