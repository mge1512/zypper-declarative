// generated from spec: zypper-declarative.spec.md sha256:aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3
#include "converge.hpp"
#include "package_db.hpp"
#include "manifest_io.hpp"
#include "hashing.hpp"
#include "transaction.hpp"

#include <ctime>
#include <fstream>
#include <filesystem>
#include <system_error>

#include <json/json.h>

namespace fs = std::filesystem;

namespace zd {

static const char* kSyncpoint = "/etc/etc.syncpoint";

// ---------------------------------------------------------------------------
// converge-packages
// ---------------------------------------------------------------------------
Result<PackagesScope> converge_packages(const TransactionContext& ctx, const Diff& diff,
                                        const std::optional<std::string>& /*repo_lock*/,
                                        const CommandRunner& runner) {
    // 1. ensure repositories configured (delegated to zypper within ctx.root).
    //    2/3. install/remove against the pinned repos. We drive zypper offline
    //    against the context root. A non-zero install exit is a packages error.
    for (const auto& p : diff.packages_install) {
        if (p.name.empty()) continue;
        CommandResult cr = runner.run("zypper", {"--root", ctx.root, "--non-interactive",
                                                 "install", "--no-recommends", p.name});
        if (cr.code != 0) {
            return Result<PackagesScope>::fail(
                make_error("packages", "install failed for " + p.name + ": " + cr.err));
        }
    }
    for (const auto& p : diff.packages_remove) {
        if (p.name.empty()) continue;
        CommandResult cr =
            runner.run("zypper", {"--root", ctx.root, "--non-interactive", "remove", p.name});
        if (cr.code != 0) {
            return Result<PackagesScope>::fail(
                make_error("packages", "remove failed for " + p.name + ": " + cr.err));
        }
    }

    // 4. query rpmdb under ctx.root for the resolved installed set (the lock).
    PackageDb db(ctx.root);
    PackagesScope scope;
    scope.attributes["package_system"] = "rpm";
    scope.elements = db.installed_packages();
    return Result<PackagesScope>::success(scope);
}

// ---------------------------------------------------------------------------
// converge-files
// ---------------------------------------------------------------------------
VoidResult converge_files(const TransactionContext& ctx, const Diff& diff,
                          const std::optional<std::string>& content_store,
                          const std::set<std::string>& keep_list) {
    PackageDb db(ctx.root);

    // 1. write declared regular files (symlink convergence is reserved).
    for (const auto& e : diff.files_write) {
        if (e.type != "file") continue;  // symlink/dir convergence reserved for later
        fs::path target = fs::path(ctx.root) / e.name.substr(e.name.find_first_not_of('/'));
        // resolve content via content_ref
        std::string content;
        if (!e.content_ref.empty() && content_store) {
            fs::path blob = fs::path(*content_store) / e.content_ref;
            std::ifstream bf(blob, std::ios::binary);
            if (!bf.is_open())
                return VoidResult::fail(
                    make_error("files", "content resolution failed for " + e.name));
            std::ostringstream ss; ss << bf.rdbuf(); content = ss.str();
        }
        std::error_code ec;
        fs::create_directories(target.parent_path(), ec);
        std::ofstream of(target, std::ios::binary);
        if (!of.is_open())
            return VoidResult::fail(make_error("files", "write failed for " + e.name));
        of.write(content.data(), static_cast<std::streamsize>(content.size()));
        of.close();
        // verify hash
        if (!e.sha256.empty()) {
            auto h = sha256_file(target.string());
            if (!h || *h != e.sha256)
                return VoidResult::fail(
                    make_error("files", "written content hash mismatch for " + e.name));
        }
    }

    // 2. delete files_delete, excluding RPM-owned, keep-listed, syncpoint.
    for (const auto& p : diff.files_delete) {
        if (p == kSyncpoint) continue;
        if (keep_list.count(p)) continue;
        FileBaseline base = db.file_baseline(p);
        if (base.found) continue;  // RPM-owned: never delete
        fs::path target = fs::path(ctx.root) / p.substr(p.find_first_not_of('/'));
        std::error_code ec;
        fs::remove(target, ec);
        if (ec)
            return VoidResult::fail(make_error("files", "delete failed for " + p));
    }
    return VoidResult::success();
}

// ---------------------------------------------------------------------------
// converge-units
// ---------------------------------------------------------------------------
VoidResult converge_units(const TransactionContext& ctx, const Diff& diff,
                          const CommandRunner& runner) {
    for (const auto& u : diff.units_change) {
        std::string action;
        if (u.state == "enabled") action = "enable";
        else if (u.state == "disabled") action = "disable";
        else if (u.state == "masked") action = "mask";
        else continue;
        CommandResult cr = runner.run("systemctl", {"--root", ctx.root, action, u.name});
        if (cr.code != 0)
            return VoidResult::fail(
                make_error("units", "offline enablement failed for " + u.name + ": " + cr.err));
    }
    return VoidResult::success();
}

// ---------------------------------------------------------------------------
// write-applied-record
// ---------------------------------------------------------------------------
static std::string now_rfc3339() {
    std::time_t t = std::time(nullptr);
    std::tm tm{};
    gmtime_r(&t, &tm);
    char buf[32];
    std::strftime(buf, sizeof(buf), "%Y-%m-%dT%H:%M:%SZ", &tm);
    return buf;
}

VoidResult write_applied_record(const TransactionContext& ctx, const Manifest& desired,
                                const std::string& desired_sha256,
                                const PackagesScope& resolved) {
    // 1. construct the AppliedRecord (declarable scopes only).
    Manifest rec;
    rec.meta.format_version = 1;
    rec.meta.generator = desired.meta.generator;
    rec.meta.created_at = now_rfc3339();
    rec.meta.desired_sha256 = desired_sha256;
    rec.repositories = desired.repositories;
    rec.services = desired.services;
    rec.config_files = desired.config_files;
    rec.packages = resolved;  // the lock

    // 2. serialise as canonical JSON (pretty on disk is acceptable; ledger is
    //    always JSON regardless of the desired manifest's input serialisation).
    std::string out = serialize_manifest(rec, ManifestFormat::Json);
    fs::path dir = fs::path(ctx.root) / "usr/lib/zypper-declarative";
    std::error_code ec;
    fs::create_directories(dir, ec);
    if (ec) return VoidResult::fail(make_error("files", "applied record dir create failed"));
    fs::path file = dir / "applied.json";
    std::ofstream of(file, std::ios::binary | std::ios::trunc);
    if (!of.is_open()) return VoidResult::fail(make_error("files", "applied record write failed"));
    of << out;
    of.close();

    // 3. stamp the snapshot userdata.
    VoidResult stamp = stamp_snapshot_userdata(ctx, desired_sha256);
    if (!stamp.ok()) return stamp;
    return VoidResult::success();
}

}  // namespace zd
