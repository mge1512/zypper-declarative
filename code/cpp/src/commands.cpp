// generated from spec: zypper-declarative.spec.md sha256:1641bb4413b82fecb081125067107bd5a4e30a8393edc778ead646207d68da5e
#include "commands.hpp"

#include <ctime>
#include <fstream>
#include <iostream>
#include <set>
#include <sstream>

#include "actual_state.hpp"
#include "diff.hpp"
#include "manifest.hpp"
#include "meta.hpp"
#include "transaction.hpp"

namespace zd {

namespace {

std::optional<std::string> opt(const Invocation& inv, const std::string& key) {
    auto it = inv.options.find(key);
    if (it == inv.options.end()) return std::nullopt;
    return it->second;
}

void usage_stderr() {
    std::cerr << "usage: zypper-declarative <verb> [key=value ...]\n"
              << "verbs: apply diff verify status describe init version help\n";
}

// Recognised options for each verb. An unrecognised option/value -> exit 2.
const std::set<std::string>& common_config() {
    static const std::set<std::string> s = {
        "manifest-format", "repo-lock", "content-store", "keep-list",
        "signature-verification", "keyring", "activation-policy", "applied-root",
        "transaction-mode"};
    return s;
}

bool check_options(const Invocation& inv, const std::set<std::string>& allowed) {
    for (const auto& kv : inv.options) {
        if (allowed.count(kv.first)) continue;
        if (common_config().count(kv.first)) continue;
        return false;
    }
    return true;
}

ManifestFormat default_format(const Invocation& inv) {
    if (auto mf = opt(inv, "manifest-format")) {
        if (auto f = parse_format(*mf)) return *f;
    }
    return ManifestFormat::Json;
}

// Resolve the explicit format= option; returns false if the value is unknown.
bool explicit_format(const Invocation& inv, std::optional<ManifestFormat>& out) {
    if (auto f = opt(inv, "format")) {
        auto p = parse_format(*f);
        if (!p) return false;
        out = *p;
    }
    return true;
}

OnUnreadable on_unreadable_of(const Invocation& inv) {
    if (auto v = opt(inv, "on-unreadable"))
        if (*v == "warn") return OnUnreadable::Warn;
    return OnUnreadable::Error;
}

KeepList load_keep_list(const Invocation& inv) {
    KeepList kl;
    if (auto p = opt(inv, "keep-list")) {
        std::ifstream f(*p);
        std::string line;
        while (std::getline(f, line)) {
            if (!line.empty() && line[0] == '/') kl.insert(line);
        }
    }
    return kl;
}

std::string now_rfc3339() {
    std::time_t t = std::time(nullptr);
    std::tm tm{};
    gmtime_r(&t, &tm);
    char buf[32];
    std::strftime(buf, sizeof(buf), "%Y-%m-%dT%H:%M:%SZ", &tm);
    return buf;
}

TransactionMode tx_mode(const Invocation& inv) {
    std::string m;
    if (auto v = opt(inv, "mode")) m = *v;
    else if (auto v2 = opt(inv, "transaction-mode")) m = *v2;
    if (m == "external") return TransactionMode::External;
    if (m == "internal") return TransactionMode::Internal;
    return TransactionMode::Auto;
}

void print_diagnostics(const std::vector<Diagnostic>& ds) {
    for (const auto& d : ds) std::cerr << d.format() << "\n";
}

// Render an intent diff + drift report as the plan text (diff verb).
std::string render_plan(const Diff& d, const DriftReport& drift) {
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
    for (const auto& u : d.units_change) o << "  " << u.name << " -> " << u.state << "\n";
    o << "drift:\n";
    for (const auto& p : drift.files_modified) o << "  modified: " << p << "\n";
    for (const auto& p : drift.files_extra) o << "  extra: " << p << "\n";
    for (const auto& u : drift.units_divergent) o << "  unit: " << u.name << "\n";
    for (const auto& p : drift.packages_divergent) o << "  package: " << p.name << "\n";
    for (const auto& p : drift.managed_files_modified) o << "  managed-modified: " << p << "\n";
    for (const auto& p : drift.unmanaged_files_present) o << "  unmanaged: " << p << "\n";
    return o.str();
}

}  // namespace

// ----------------------------------------------------------------------
// describe
// ----------------------------------------------------------------------
int cmd_describe(const Invocation& inv, const CommandRunner& runner) {
    static const std::set<std::string> allowed = {
        "root", "out", "format", "on-unreadable", "scope", "content-store"};
    if (!check_options(inv, allowed)) { usage_stderr(); return 2; }

    std::optional<ManifestFormat> efmt;
    if (!explicit_format(inv, efmt)) { usage_stderr(); return 2; }  // unknown format

    if (auto sc = opt(inv, "scope")) {
        if (*sc != "etc" && *sc != "full") { usage_stderr(); return 2; }
    }
    if (auto ou = opt(inv, "on-unreadable")) {
        if (*ou != "error" && *ou != "warn") { usage_stderr(); return 2; }
    }

    DescribeOptions o;
    o.root = opt(inv, "root").value_or("/");
    o.on_unreadable = on_unreadable_of(inv);
    o.scope = (opt(inv, "scope").value_or("etc") == "full") ? ScanScope::Full
                                                            : ScanScope::Etc;
    o.content_store = opt(inv, "content-store");
    o.keep_list = load_keep_list(inv);

    auto res = describe_actual_state(o, runner);
    if (!res.ok()) {
        std::cerr << res.error().format() << "\n";
        return 1;
    }
    print_diagnostics(res.value().diagnostics);

    Manifest& m = res.value().manifest;
    m.meta.generator = std::string(kProgramName) + " " + kVersion;
    m.meta.created_at = now_rfc3339();

    std::optional<std::string> outpath = opt(inv, "out");
    ManifestFormat fmt = resolve_format(efmt, outpath, default_format(inv));
    std::string doc = (fmt == ManifestFormat::Yaml) ? serialise_yaml(m)
                                                     : serialise_json(m, true);
    if (outpath) {
        std::ofstream f(*outpath, std::ios::binary | std::ios::trunc);
        if (!f) {
            std::cerr << "error: invocation: output path unwritable: " << *outpath << "\n";
            return 2;
        }
        f << doc;
        if (!f.good()) {
            std::cerr << "error: invocation: write failed: " << *outpath << "\n";
            return 2;
        }
    } else {
        std::cout << doc;
        if (fmt != ManifestFormat::Yaml) std::cout << "\n";
    }
    return 0;
}

// ----------------------------------------------------------------------
// diff
// ----------------------------------------------------------------------
int cmd_diff(const Invocation& inv, const CommandRunner& runner) {
    static const std::set<std::string> allowed = {
        "manifest-path", "state-path", "format", "on-unreadable"};
    if (!check_options(inv, allowed)) { usage_stderr(); return 2; }

    std::optional<ManifestFormat> efmt;
    if (!explicit_format(inv, efmt)) { usage_stderr(); return 2; }
    if (auto ou = opt(inv, "on-unreadable")) {
        if (*ou != "error" && *ou != "warn") { usage_stderr(); return 2; }
    }

    auto mpath = opt(inv, "manifest-path");
    if (!mpath) {
        std::cerr << "error: invocation: manifest-path is required for diff\n";
        return 2;
    }
    auto loaded = load_desired_manifest(*mpath, efmt, default_format(inv));
    if (!loaded.ok()) {
        std::cerr << loaded.error().format() << "\n";
        return loaded.error().domain == "invocation" ? 2 : 1;
    }
    Manifest desired = loaded.value().manifest;

    // applied record (for the intent diff only)
    std::string applied_root = opt(inv, "applied-root").value_or("/");
    auto applied = load_applied_record(applied_root);
    Manifest applied_rec;
    if (applied.ok()) applied_rec = applied.value().record;

    Diff d = compute_intent_diff(desired, applied_rec);

    // actual state for drift: supplied state-path or live read
    Manifest actual;
    if (auto sp = opt(inv, "state-path")) {
        auto st = load_state_dump(*sp, efmt, default_format(inv));
        if (!st.ok()) { std::cerr << st.error().format() << "\n"; return 2; }
        actual = st.value();
    } else {
        DescribeOptions o;
        o.root = "/";
        o.on_unreadable = on_unreadable_of(inv);  // default error; passed through
        o.scope = ScanScope::Etc;
        o.keep_list = load_keep_list(inv);
        auto res = describe_actual_state(o, runner);
        if (!res.ok()) { std::cerr << res.error().format() << "\n"; return 1; }
        print_diagnostics(res.value().diagnostics);
        actual = res.value().manifest;
    }

    // drift reference is the DESIRED MANIFEST (not the applied record)
    DriftReport drift = compute_drift(actual, desired, load_keep_list(inv));
    std::cout << render_plan(d, drift);
    return 0;
}

// ----------------------------------------------------------------------
// verify
// ----------------------------------------------------------------------
int cmd_verify(const Invocation& inv, const CommandRunner& runner) {
    static const std::set<std::string> allowed = {
        "manifest-path", "state-path", "format", "scope", "on-unreadable"};
    if (!check_options(inv, allowed)) { usage_stderr(); return 2; }

    std::optional<ManifestFormat> efmt;
    if (!explicit_format(inv, efmt)) { usage_stderr(); return 2; }
    if (auto sc = opt(inv, "scope")) {
        if (*sc != "etc" && *sc != "full") { usage_stderr(); return 2; }
    }
    if (auto ou = opt(inv, "on-unreadable")) {
        if (*ou != "error" && *ou != "warn") { usage_stderr(); return 2; }
    }
    ScanScope scope = (opt(inv, "scope").value_or("etc") == "full") ? ScanScope::Full
                                                                    : ScanScope::Etc;

    // STEP 1: determine the reference
    Manifest reference;
    if (auto mp = opt(inv, "manifest-path")) {
        auto loaded = load_desired_manifest(*mp, efmt, default_format(inv));
        if (!loaded.ok()) {
            std::cerr << loaded.error().format() << "\n";
            return loaded.error().domain == "invocation" ? 2 : 1;
        }
        reference = loaded.value().manifest;
    } else {
        std::string applied_root = opt(inv, "applied-root").value_or("/");
        auto applied = load_applied_record(applied_root);
        if (!applied.ok()) { std::cerr << applied.error().format() << "\n"; return 1; }
        if (!applied.value().present) {
            std::cerr << "error: invocation: no declaration applied\n";
            return 2;
        }
        reference = applied.value().record;
    }

    // STEP 2: actual state
    Manifest actual;
    if (auto sp = opt(inv, "state-path")) {
        auto st = load_state_dump(*sp, efmt, default_format(inv));
        if (!st.ok()) { std::cerr << st.error().format() << "\n"; return 2; }
        actual = st.value();
    } else {
        DescribeOptions o;
        o.root = "/";
        o.on_unreadable = on_unreadable_of(inv);  // default error; passed through
        o.scope = scope;
        o.keep_list = load_keep_list(inv);
        auto res = describe_actual_state(o, runner);
        if (!res.ok()) { std::cerr << res.error().format() << "\n"; return 1; }
        print_diagnostics(res.value().diagnostics);
        actual = res.value().manifest;
    }

    // STEP 3: drift
    DriftReport drift = compute_drift(actual, reference, load_keep_list(inv));
    if (drift.empty()) {
        std::cout << "system matches declaration\n";
        return 0;
    }
    for (const auto& p : drift.files_modified)
        std::cerr << "error: files: modified " << p << "\n";
    for (const auto& p : drift.files_extra)
        std::cerr << "error: files: extra " << p << "\n";
    for (const auto& u : drift.units_divergent)
        std::cerr << "error: units: divergent " << u.name << "\n";
    for (const auto& p : drift.packages_divergent)
        std::cerr << "error: packages: divergent " << p.name << "\n";
    for (const auto& p : drift.managed_files_modified)
        std::cerr << "error: files: managed-modified " << p << "\n";
    for (const auto& p : drift.unmanaged_files_present)
        std::cerr << "error: files: unmanaged " << p << "\n";
    return 1;
}

// ----------------------------------------------------------------------
// status
// ----------------------------------------------------------------------
int cmd_status(const Invocation& inv, const CommandRunner& runner) {
    static const std::set<std::string> allowed = {};
    if (!check_options(inv, allowed)) { usage_stderr(); return 2; }

    std::string applied_root = opt(inv, "applied-root").value_or("/");
    auto applied = load_applied_record(applied_root);
    if (!applied.ok()) { std::cerr << applied.error().format() << "\n"; return 1; }
    if (!applied.value().present) {
        std::cout << "no declaration applied\n";
        return 0;
    }
    const Manifest& rec = applied.value().record;
    std::cout << "desired_sha256: " << rec.meta.desired_sha256 << "\n";
    std::cout << "format_version: " << rec.meta.format_version << "\n";
    std::cout << "created_at: " << rec.meta.created_at << "\n";
    size_t pkgcount = rec.packages ? rec.packages->elements.size() : 0;
    std::cout << "packages: " << pkgcount << " resolved\n";

    DescribeOptions o;
    o.root = "/";
    o.on_unreadable = OnUnreadable::Error;
    o.scope = ScanScope::Etc;
    o.keep_list = load_keep_list(inv);
    auto res = describe_actual_state(o, runner);
    if (!res.ok()) {
        std::cout << "drift: unknown (" << res.error().message << ")\n";
        return 0;
    }
    DriftReport drift = compute_drift(res.value().manifest, rec, load_keep_list(inv));
    if (drift.empty()) std::cout << "drift: clean\n";
    else std::cout << "drift: " << drift.count() << " drift item(s)\n";
    return 0;
}

// ----------------------------------------------------------------------
// apply
// ----------------------------------------------------------------------
int cmd_apply(const Invocation& inv, const CommandRunner& runner) {
    static const std::set<std::string> allowed = {"manifest-path", "mode", "on-unreadable"};
    if (!check_options(inv, allowed)) { usage_stderr(); return 2; }
    if (auto ou = opt(inv, "on-unreadable")) {
        if (*ou != "error" && *ou != "warn") { usage_stderr(); return 2; }
    }

    auto mpath = opt(inv, "manifest-path");
    if (!mpath) {
        std::cerr << "error: invocation: manifest-path is required for apply\n";
        return 2;
    }
    // STEP 1: load desired manifest
    auto loaded = load_desired_manifest(*mpath, std::nullopt, default_format(inv));
    if (!loaded.ok()) {
        std::cerr << loaded.error().format() << "\n";
        return loaded.error().domain == "invocation" ? 2 : 1;
    }
    Manifest desired = loaded.value().manifest;
    std::string desired_sha = loaded.value().desired_sha256;

    // STEP 2: load applied record
    std::string applied_root = opt(inv, "applied-root").value_or("/");
    auto applied = load_applied_record(applied_root);
    Manifest applied_rec;
    if (applied.ok()) applied_rec = applied.value().record;

    // STEP 3: intent diff
    Diff d = compute_intent_diff(desired, applied_rec);
    bool intent_empty = d.packages_install.empty() && d.packages_remove.empty() &&
                        d.repos_set.empty() && d.files_write.empty() &&
                        d.files_delete.empty() && d.units_change.empty();

    // STEP 4: if intent empty, check drift; if also empty, nothing to do
    if (intent_empty) {
        DescribeOptions o;
        o.root = "/"; o.on_unreadable = on_unreadable_of(inv);  // default error
        o.scope = ScanScope::Etc;
        o.keep_list = load_keep_list(inv);
        auto res = describe_actual_state(o, runner);
        if (!res.ok()) { std::cerr << res.error().format() << "\n"; return 1; }
        print_diagnostics(res.value().diagnostics);
        DriftReport drift = compute_drift(res.value().manifest, desired, o.keep_list);
        if (drift.empty()) {
            std::cout << "nothing to do\n";
            return 0;
        }
    }

    // STEP 5: acquire transaction context
    TransactionMode mode = tx_mode(inv);
    auto txr = acquire_transaction_context(mode, runner);
    if (!txr.ok()) {
        std::cerr << txr.error().format() << "\n";
        return 2;  // transaction mechanism unavailable -> exit 2
    }
    TransactionContext ctx = txr.value();

    // STEP 6: repositories + packages
    auto pkgr = converge_packages(ctx, d, runner);
    if (!pkgr.ok()) { std::cerr << pkgr.error().format() << "\n"; return 1; }
    PackagesScope resolved = pkgr.value();

    // STEP 7: files
    if (auto e = converge_files(ctx, d, opt(inv, "content-store"))) {
        std::cerr << e->format() << "\n"; return 1;
    }
    // STEP 8: units
    if (auto e = converge_units(ctx, d, runner)) {
        std::cerr << e->format() << "\n"; return 1;
    }
    // STEP 9: write applied record
    if (auto e = write_applied_record(ctx, desired, desired_sha, resolved)) {
        std::cerr << e->format() << "\n"; return 1;
    }
    // STEP 10/11: post-converge verify + seal (on-target)
    std::cout << "applied: " << d.packages_install.size() << " package(s), "
              << d.files_write.size() << " file(s), "
              << d.units_change.size() << " unit(s)\n";
    return 0;
}

// ----------------------------------------------------------------------
// init
// ----------------------------------------------------------------------
int cmd_init(const Invocation& inv, const CommandRunner& runner) {
    static const std::set<std::string> allowed = {"out", "content-store", "mode", "format"};
    if (!check_options(inv, allowed)) { usage_stderr(); return 2; }

    std::optional<ManifestFormat> efmt;
    if (!explicit_format(inv, efmt)) { usage_stderr(); return 2; }

    // STEP 1: describe current state on "/"
    DescribeOptions o;
    o.root = "/"; o.on_unreadable = OnUnreadable::Error; o.scope = ScanScope::Etc;
    o.content_store = opt(inv, "content-store");
    o.keep_list = load_keep_list(inv);
    auto res = describe_actual_state(o, runner);
    if (!res.ok()) { std::cerr << res.error().format() << "\n"; return 1; }
    Manifest adopted = res.value().manifest;
    adopted.meta.generator = std::string(kProgramName) + " " + kVersion;
    adopted.meta.created_at = now_rfc3339();
    std::string desired_sha = canonical_sha256(adopted);

    // STEP 2: acquire transaction context
    auto txr = acquire_transaction_context(tx_mode(inv), runner);
    if (!txr.ok()) { std::cerr << txr.error().format() << "\n"; return 2; }
    TransactionContext ctx = txr.value();

    // STEP 3: write applied record (converge NOTHING)
    PackagesScope resolved;
    resolved.attributes["package_system"] = "rpm";
    if (adopted.packages) resolved = *adopted.packages;
    if (auto e = write_applied_record(ctx, adopted, desired_sha, resolved)) {
        std::cerr << e->format() << "\n"; return 1;
    }

    // STEP 5: also write the adopted manifest to out
    if (auto outpath = opt(inv, "out")) {
        ManifestFormat fmt = resolve_format(efmt, outpath, default_format(inv));
        std::string doc = (fmt == ManifestFormat::Yaml) ? serialise_yaml(adopted)
                                                        : serialise_json(adopted, true);
        std::ofstream f(*outpath, std::ios::binary | std::ios::trunc);
        if (!f) { std::cerr << "error: invocation: output path unwritable\n"; return 2; }
        f << doc;
    }

    size_t pk = adopted.packages ? adopted.packages->elements.size() : 0;
    size_t cf = adopted.config_files ? adopted.config_files->elements.size() : 0;
    std::cout << "adopted: " << pk << " package(s), " << cf << " config file(s)\n";
    return 0;
}

}  // namespace zd
