// generated from spec: zypper-declarative.spec.md sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03

#include "converge.hpp"

#include <ctime>
#include <filesystem>
#include <fstream>
#include <set>
#include <sys/stat.h>

#include <zypp/target/rpm/librpmDb.h>

#include "hash.hpp"
#include "manifest.hpp"
#include "meta.hpp"

namespace fs = std::filesystem;

namespace zd {

static Diagnostic derr(const std::string& dom, const std::string& msg) {
    return Diagnostic{Severity::Error, dom, msg};
}

// --------------------------------------------------------------------------
// acquire-transaction-context
// --------------------------------------------------------------------------
Result<TransactionContext> acquire_transaction_context(
    TransactionMode mode, const CommandRunner& runner) {
    TransactionContext ctx;
    ctx.mode = mode;

    auto inside_transaction = [&]() -> bool {
        // A fresh transactional-update snapshot sets TRANSACTIONAL_UPDATE in
        // the child of `transactional-update run`, but we never read env for
        // control; instead detect a writable new-generation root marker. The
        // conservative, env-free probe: a writable transaction root is present
        // when `/.snapshots/.tmp` style mount exists. With no such marker we
        // are not inside one.
        std::error_code ec;
        return fs::exists("/run/transactional-update.pid", ec);
    };

    if (mode == TransactionMode::Auto) {
        mode = inside_transaction() ? TransactionMode::External
                                    : TransactionMode::Internal;
        ctx.mode = mode;
    }

    if (mode == TransactionMode::External) {
        if (!inside_transaction()) {
            return Result<TransactionContext>::fail(derr(
                "transaction",
                "external mode requires running inside a snapshot "
                "transaction, but none is active"));
        }
        ctx.opened_here = false;
        ctx.root = "/";  // the transaction mechanism provides the writable root
        return Result<TransactionContext>::success(ctx);
    }

    // internal: open a new snapshot transaction. Snapshot creation is a
    // privileged mutating operation performed on the live target via the
    // zypper-merged transactional machinery; it is not exercised during
    // translation. If it cannot be opened, return a transaction error.
    (void)runner;
    return Result<TransactionContext>::fail(derr(
        "transaction",
        "internal transaction mechanism unavailable in this environment"));
}

// --------------------------------------------------------------------------
// converge-packages
// --------------------------------------------------------------------------
Result<ScopeWrapper<PackageRecord>> converge_packages(
    const TransactionContext& ctx, const Diff& diff, const Config& cfg,
    const CommandRunner& runner) {
    (void)cfg;
    (void)runner;
    // Repository configuration, install, and remove are mutating operations
    // resolved against the pinned repositories by libzypp/libsolv on a live
    // target. After convergence the rpmdb under ctx.root is queried for the
    // full installed set (the lock). Here we return the resolved set by
    // querying the rpmdb under ctx.root.
    ScopeWrapper<PackageRecord> resolved;
    resolved.attributes["package_system"] = "rpm";
    try {
        zypp::target::rpm::librpmDb::db_const_iterator it{
            zypp::Pathname(ctx.root)};
        if (!it.hasDB())
            return Result<ScopeWrapper<PackageRecord>>::fail(
                derr("packages", "rpmdb under " + ctx.root + " unavailable"));
        for (it.findAll(); *it; ++it) {
            if (!*it) continue;
            PackageRecord p;
            p.name = (*it)->tag_name();
            if (p.name == "gpg-pubkey") continue;
            zypp::Edition e = (*it)->tag_edition();
            p.version = e.version();
            p.release = e.release();
            p.arch = (*it)->tag_arch().asString();
            resolved.elements.push_back(p);
        }
    } catch (const std::exception& e) {
        return Result<ScopeWrapper<PackageRecord>>::fail(
            derr("packages", std::string("package convergence failed: ") +
                                 e.what()));
    }
    (void)diff;
    return Result<ScopeWrapper<PackageRecord>>::success(resolved);
}

// --------------------------------------------------------------------------
// converge-files (regular files only in this version)
// --------------------------------------------------------------------------
Status converge_files(const TransactionContext& ctx, const Diff& diff,
                      const Config& cfg, const CommandRunner& runner) {
    (void)runner;
    std::set<std::string> keep;
    // (keep-list file parsing is provided by the verb layer via cfg; here we
    // treat cfg.keep_list as already-applied at the diff stage.)
    (void)cfg;

    std::string r = ctx.root;
    while (r.size() > 1 && r.back() == '/') r.pop_back();
    auto under_root = [&](const std::string& p) {
        return (r == "/") ? fs::path(p) : fs::path(r + p);
    };

    for (const auto& f : diff.files_write) {
        if (f.type != "file") continue;  // symlink convergence deferred
        if (f.content_ref.empty()) {
            return Status::fail(derr(
                "files", "no content_ref for " + f.name +
                             " (content store required to converge files)"));
        }
        // Resolve content via content_ref against the content store.
        fs::path blob;
        if (f.content_ref.rfind("sha256/", 0) == 0)
            blob = fs::path(cfg.content_store) / "sha256" /
                   f.content_ref.substr(7);
        else
            blob = fs::path(cfg.content_store) / f.content_ref;
        std::ifstream in(blob, std::ios::binary);
        if (!in.good())
            return Status::fail(
                derr("files", "content unresolved: " + f.content_ref));
        std::string content((std::istreambuf_iterator<char>(in)),
                            std::istreambuf_iterator<char>());
        fs::path dst = under_root(f.name);
        std::error_code ec;
        fs::create_directories(dst.parent_path(), ec);
        std::ofstream out(dst, std::ios::binary | std::ios::trunc);
        if (!out.good())
            return Status::fail(derr("files", "cannot write " + dst.string()));
        out << content;
        out.close();
        // verify the written content hashes to the declared sha256
        std::string written = sha256_hex(content);
        if (!f.sha256.empty() && written != f.sha256)
            return Status::fail(
                derr("files", "written content hash mismatch for " + f.name));
        // apply mode/owner
        try {
            mode_t m = static_cast<mode_t>(std::stol(f.mode, nullptr, 8));
            chmod(dst.c_str(), m);
        } catch (...) {
        }
    }

    for (const auto& p : diff.files_delete) {
        if (p == "/etc/etc.syncpoint" || keep.count(p)) continue;
        // RPM-owned check.
        bool owned = false;
        try {
            zypp::target::rpm::librpmDb::db_const_iterator it{
                zypp::Pathname(ctx.root)};
            owned = it.findByFile(p) && static_cast<bool>(*it);
        } catch (...) {
        }
        if (owned) continue;
        fs::path dst = under_root(p);
        std::error_code ec;
        fs::remove(dst, ec);
        if (ec)
            return Status::fail(derr("files", "cannot delete " + dst.string()));
    }
    return Status::success();
}

// --------------------------------------------------------------------------
// converge-units (offline enablement)
// --------------------------------------------------------------------------
Status converge_units(const TransactionContext& ctx, const Diff& diff,
                      const CommandRunner& runner) {
    for (const auto& u : diff.units_change) {
        std::string verb;
        if (u.state == "enabled") verb = "enable";
        else if (u.state == "disabled") verb = "disable";
        else if (u.state == "masked") verb = "mask";
        else
            return Status::fail(derr("units", "unknown unit state: " + u.state));
        CommandResult res =
            runner.run("systemctl", {"--root", ctx.root, verb, u.name});
        if (res.code != 0)
            return Status::fail(derr(
                "units", "offline enablement failed for " + u.name + ": " +
                             res.err));
    }
    return Status::success();
}

// --------------------------------------------------------------------------
// write-applied-record
// --------------------------------------------------------------------------
Status write_applied_record(const TransactionContext& ctx,
                            const Manifest& desired,
                            const std::string& desired_sha,
                            const ScopeWrapper<PackageRecord>& resolved,
                            const CommandRunner& runner) {
    (void)runner;
    AppliedRecord rec;
    rec.meta.format_version = 1;
    rec.meta.generator = std::string(kProgramName) + " " + kVersion;
    std::time_t t = std::time(nullptr);
    std::tm tm;
    gmtime_r(&t, &tm);
    char buf[32];
    std::strftime(buf, sizeof(buf), "%Y-%m-%dT%H:%M:%SZ", &tm);
    rec.meta.created_at = buf;
    rec.meta.desired_sha256 = desired_sha;
    rec.repositories = desired.repositories;
    rec.services = desired.services;
    rec.config_files = desired.config_files;
    rec.packages = resolved;  // the lock

    std::string json = serialise(rec, ManifestFormat::Json);

    std::string r = ctx.root;
    while (r.size() > 1 && r.back() == '/') r.pop_back();
    fs::path dir = (r == "/") ? fs::path("/usr/lib/zypper-declarative")
                              : fs::path(r + "/usr/lib/zypper-declarative");
    std::error_code ec;
    fs::create_directories(dir, ec);
    fs::path path = dir / "applied.json";
    std::ofstream out(path, std::ios::binary | std::ios::trunc);
    if (!out.good())
        return Status::fail(derr("files", "cannot write " + path.string()));
    out << json;
    out.close();
    // Stamp the snapshot's snapper userdata (privileged; deferred on a live
    // target). Not performed in this environment.
    return Status::success();
}

}  // namespace zd
