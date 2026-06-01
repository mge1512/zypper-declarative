// generated from spec: zypper-declarative.spec.md sha256:f2cc80627e483a48bb8411d297711bc5f6c6e74c28dbf0dafc8fe7bd8817251e
//
// manifest.cpp -- resolve-format, JSON/YAML serialisation and parsing, schema
// validation, the YAML safe profile, the canonical-model hash, and
// load-desired-manifest / load-applied-record / load-state-dump.
#include "manifest.hpp"
#include "hashing.hpp"

#include <json/json.h>
#include <yaml-cpp/yaml.h>

#include <algorithm>
#include <functional>
#include <fstream>
#include <sstream>
#include <filesystem>

namespace fs = std::filesystem;

namespace zd {

// --------------------------------------------------------------------------
// resolve-format
// --------------------------------------------------------------------------
static std::optional<ManifestFormat> ext_format(const std::string& path) {
    auto pos = path.find_last_of('.');
    if (pos == std::string::npos) return std::nullopt;
    std::string ext = path.substr(pos);
    std::transform(ext.begin(), ext.end(), ext.begin(), ::tolower);
    if (ext == ".json") return ManifestFormat::Json;
    if (ext == ".yaml" || ext == ".yml") return ManifestFormat::Yaml;
    return std::nullopt;
}

ManifestFormat resolve_format(const std::optional<ManifestFormat>& explicit_fmt,
                              const std::optional<std::string>& path,
                              ManifestFormat default_fmt) {
    if (explicit_fmt.has_value()) return *explicit_fmt;          // explicit wins
    if (path.has_value()) {
        auto e = ext_format(*path);
        if (e.has_value()) return *e;                            // recognised extension
    }
    return default_fmt;                                          // CONFIG default
}

// --------------------------------------------------------------------------
// JSON serialisation (Machinery shape)
// --------------------------------------------------------------------------
static Json::Value pkg_to_json(const PackageRecord& p) {
    Json::Value v(Json::objectValue);
    v["name"] = p.name; v["version"] = p.version;
    v["release"] = p.release; v["arch"] = p.arch;
    return v;
}
static Json::Value repo_to_json(const RepositoryRecord& r) {
    Json::Value v(Json::objectValue);
    v["alias"] = r.alias; v["name"] = r.name; v["url"] = r.url;
    v["type"] = r.type; v["enabled"] = r.enabled; v["gpgcheck"] = r.gpgcheck;
    v["autorefresh"] = r.autorefresh; v["priority"] = r.priority;
    return v;
}
static Json::Value svc_to_json(const ServiceRecord& s) {
    Json::Value v(Json::objectValue);
    v["name"] = s.name; v["state"] = s.state;
    return v;
}
static Json::Value file_to_json(const ManagedFileRecord& f) {
    Json::Value v(Json::objectValue);
    v["name"] = f.name; v["type"] = f.type; v["mode"] = f.mode;
    v["user"] = f.user; v["group"] = f.group; v["sha256"] = f.sha256;
    v["target"] = f.target; v["content_ref"] = f.content_ref;
    v["package_name"] = f.package_name;
    return v;
}
static Json::Value baseline_to_json(const ManagedBaselineRecord& b) {
    Json::Value v(Json::objectValue);
    v["name"] = b.name; v["type"] = b.type; v["mode"] = b.mode;
    v["user"] = b.user; v["group"] = b.group; v["sha256"] = b.sha256;
    v["target"] = b.target; v["package_name"] = b.package_name;
    Json::Value ch(Json::arrayValue);
    for (auto& c : b.changes) ch.append(c);
    v["changes"] = ch;
    return v;
}
static Json::Value unmanaged_to_json(const UnmanagedFileRecord& u) {
    Json::Value v(Json::objectValue);
    v["name"] = u.name; v["type"] = u.type; v["mode"] = u.mode;
    v["user"] = u.user; v["group"] = u.group; v["sha256"] = u.sha256;
    v["target"] = u.target;
    return v;
}

template <class T, class F>
static Json::Value scope_to_json(const ScopeWrapper<T>& s, F elem_fn) {
    Json::Value v(Json::objectValue);
    if (s.has_attributes) {
        Json::Value attrs(Json::objectValue);
        for (auto& kv : s.attributes) attrs[kv.first] = kv.second;
        v["_attributes"] = attrs;
    } else {
        v["_attributes"] = Json::Value(Json::nullValue);
    }
    Json::Value elems(Json::arrayValue);
    for (auto& e : s.elements) elems.append(elem_fn(e));
    v["_elements"] = elems;
    return v;
}

static Json::Value manifest_to_json(const Manifest& m) {
    Json::Value root(Json::objectValue);
    Json::Value meta(Json::objectValue);
    meta["format_version"] = m.meta.format_version;
    meta["generator"] = m.meta.generator;
    meta["created_at"] = m.meta.created_at;
    meta["desired_sha256"] = m.meta.desired_sha256;
    root["meta"] = meta;
    if (m.packages)
        root["packages"] = scope_to_json(*m.packages, pkg_to_json);
    if (m.repositories)
        root["repositories"] = scope_to_json(*m.repositories, repo_to_json);
    if (m.services)
        root["services"] = scope_to_json(*m.services, svc_to_json);
    if (m.config_files)
        root["config_files"] = scope_to_json(*m.config_files, file_to_json);
    if (m.changed_managed_files)
        root["changed_managed_files"] =
            scope_to_json(*m.changed_managed_files, baseline_to_json);
    if (m.unmanaged_files)
        root["unmanaged_files"] = scope_to_json(*m.unmanaged_files, unmanaged_to_json);
    return root;
}

std::string serialize_manifest(const Manifest& m, ManifestFormat fmt, bool pretty) {
    Json::Value root = manifest_to_json(m);
    if (fmt == ManifestFormat::Json) {
        Json::StreamWriterBuilder b;
        if (pretty) {
            b["indentation"] = "  ";
        } else {
            b["indentation"] = "";
        }
        b["enableYAMLCompatibility"] = false;
        std::string s = Json::writeString(b, root);
        if (pretty && (s.empty() || s.back() != '\n')) s.push_back('\n');
        return s;
    }
    // YAML: render the same data model. We emit via yaml-cpp from the JSON tree.
    YAML::Emitter em;
    std::function<void(const Json::Value&)> emit = [&](const Json::Value& v) {
        switch (v.type()) {
            case Json::nullValue: em << YAML::Null; break;
            case Json::intValue: em << v.asInt64(); break;
            case Json::uintValue: em << v.asUInt64(); break;
            case Json::realValue: em << v.asDouble(); break;
            case Json::booleanValue: em << v.asBool(); break;
            case Json::stringValue: em << v.asString(); break;
            case Json::arrayValue: {
                em << YAML::BeginSeq;
                for (auto& e : v) emit(e);
                em << YAML::EndSeq;
                break;
            }
            case Json::objectValue: {
                em << YAML::BeginMap;
                for (auto it = v.begin(); it != v.end(); ++it) {
                    em << YAML::Key << it.name() << YAML::Value;
                    emit(*it);
                }
                em << YAML::EndMap;
                break;
            }
        }
    };
    emit(root);
    std::string s = em.c_str();
    if (s.empty() || s.back() != '\n') s.push_back('\n');
    return s;
}

// --------------------------------------------------------------------------
// Canonical JSON for hashing: keys sorted, _elements sorted by identity, compact.
// --------------------------------------------------------------------------
static Json::Value sort_value(const Json::Value& v); // fwd

static Json::Value sort_array_by_identity(const std::string& scope_key,
                                          const Json::Value& arr) {
    std::vector<Json::Value> items;
    for (auto& e : arr) items.push_back(sort_value(e));
    auto key_of = [&](const Json::Value& e) -> std::string {
        if (scope_key == "packages")
            return e.get("name", "").asString() + "\x1f" + e.get("arch", "").asString();
        if (scope_key == "repositories")
            return e.get("alias", "").asString();
        if (scope_key == "services")
            return e.get("name", "").asString();
        if (scope_key == "config_files")
            return e.get("name", "").asString();
        return e.get("name", "").asString();
    };
    std::stable_sort(items.begin(), items.end(),
        [&](const Json::Value& a, const Json::Value& b) {
            return key_of(a) < key_of(b);
        });
    Json::Value out(Json::arrayValue);
    for (auto& e : items) out.append(e);
    return out;
}

static Json::Value sort_value(const Json::Value& v) {
    if (v.isObject()) {
        Json::Value out(Json::objectValue);
        std::vector<std::string> keys = v.getMemberNames();
        std::sort(keys.begin(), keys.end());
        for (auto& k : keys) out[k] = sort_value(v[k]);
        return out;
    }
    if (v.isArray()) {
        Json::Value out(Json::arrayValue);
        for (auto& e : v) out.append(sort_value(e));
        return out;
    }
    return v;
}

std::string canonical_json(const Manifest& m) {
    // Build a model copy with desired_sha256 cleared so the hash is over the
    // intent, not over a previously-stamped value.
    Manifest c = m;
    c.meta.desired_sha256 = "";
    c.meta.created_at = "";        // informational only, not compared/hashed
    c.meta.generator = "";         // informational, exclude from identity
    Json::Value root = manifest_to_json(c);
    // Re-sort _elements per scope by identity, then sort all keys.
    for (const char* sk : {"packages", "repositories", "services", "config_files"}) {
        if (root.isMember(sk) && root[sk].isMember("_elements")) {
            root[sk]["_elements"] = sort_array_by_identity(sk, root[sk]["_elements"]);
        }
    }
    Json::Value sorted = sort_value(root);
    Json::StreamWriterBuilder b;
    b["indentation"] = "";
    b["enableYAMLCompatibility"] = false;
    return Json::writeString(b, sorted);
}

Sha256 canonical_hash(const Manifest& m) {
    return sha256_hex(canonical_json(m));
}

// --------------------------------------------------------------------------
// Parsing: JSON via jsoncpp; YAML via yaml-cpp under the safe profile.
// --------------------------------------------------------------------------
static Diagnostic manifest_err(const std::string& msg) {
    return Diagnostic{Severity::Error, "manifest", msg};
}

// Convert a validated YAML node tree to a Json::Value, enforcing the safe
// profile: reject any node carrying a non-default tag; explicit typing only.
static bool yaml_to_json(const YAML::Node& node, Json::Value& out, std::string& err);

static bool yaml_tag_is_safe(const std::string& tag) {
    // Default/implicit tags from yaml-cpp are "?" (non-specific) or the standard
    // core schema tags. Reject anything that looks like an application/executable
    // tag (e.g. "!!python/...", "!foo", "tag:...").
    if (tag.empty() || tag == "?" || tag == "!") return true;
    if (tag == "tag:yaml.org,2002:str" || tag == "tag:yaml.org,2002:int" ||
        tag == "tag:yaml.org,2002:float" || tag == "tag:yaml.org,2002:bool" ||
        tag == "tag:yaml.org,2002:null" || tag == "tag:yaml.org,2002:map" ||
        tag == "tag:yaml.org,2002:seq") return true;
    return false;
}

static bool yaml_to_json(const YAML::Node& node, Json::Value& out, std::string& err) {
    if (!yaml_tag_is_safe(node.Tag())) {
        err = "unsafe YAML tag '" + node.Tag() + "'";
        return false;
    }
    switch (node.Type()) {
        case YAML::NodeType::Null:
            out = Json::Value(Json::nullValue);
            return true;
        case YAML::NodeType::Scalar:
            // explicit typing: keep scalars as strings; numeric coercion is done
            // explicitly by the schema reader below, not by implicit YAML typing.
            out = node.Scalar();
            return true;
        case YAML::NodeType::Sequence: {
            out = Json::Value(Json::arrayValue);
            for (std::size_t i = 0; i < node.size(); ++i) {
                Json::Value child;
                if (!yaml_to_json(node[i], child, err)) return false;
                out.append(child);
            }
            return true;
        }
        case YAML::NodeType::Map: {
            out = Json::Value(Json::objectValue);
            for (auto it = node.begin(); it != node.end(); ++it) {
                Json::Value child;
                if (!yaml_to_json(it->second, child, err)) return false;
                out[it->first.as<std::string>()] = child;
            }
            return true;
        }
        default:
            err = "undefined YAML node";
            return false;
    }
}

// --- schema validation over a Json::Value tree -----------------------------
static bool as_int(const Json::Value& v, int& out) {
    if (v.isInt()) { out = v.asInt(); return true; }
    if (v.isString()) {
        try { out = std::stoi(v.asString()); return true; } catch (...) { return false; }
    }
    return false;
}
static bool as_bool(const Json::Value& v, bool& out) {
    if (v.isBool()) { out = v.asBool(); return true; }
    if (v.isString()) {
        std::string s = v.asString();
        if (s == "true") { out = true; return true; }
        if (s == "false") { out = false; return true; }
    }
    return false;
}
static std::string as_str(const Json::Value& v) {
    if (v.isString()) return v.asString();
    if (v.isNull()) return "";
    return v.asString();
}

template <class T, class F>
static bool parse_scope(const Json::Value& sv, ScopeWrapper<T>& out, F elem_fn,
                        std::string& err) {
    if (!sv.isObject()) { err = "scope is not an object"; return false; }
    if (sv.isMember("_attributes")) {
        const Json::Value& a = sv["_attributes"];
        if (a.isNull()) { out.has_attributes = false; }
        else if (a.isObject()) {
            out.has_attributes = true;
            for (auto it = a.begin(); it != a.end(); ++it)
                out.attributes[it.name()] = as_str(*it);
        } else { err = "_attributes must be object or null"; return false; }
    }
    if (sv.isMember("_elements")) {
        const Json::Value& es = sv["_elements"];
        if (!es.isArray()) { err = "_elements must be an array"; return false; }
        for (auto& e : es) {
            T rec;
            if (!elem_fn(e, rec, err)) return false;
            out.elements.push_back(rec);
        }
    }
    return true;
}

static bool elem_pkg(const Json::Value& e, PackageRecord& r, std::string& err) {
    r.name = as_str(e["name"]); r.version = as_str(e["version"]);
    r.release = as_str(e["release"]); r.arch = as_str(e["arch"]);
    if (r.name.empty()) { err = "package name must be non-empty"; return false; }
    return true;
}
static bool elem_repo(const Json::Value& e, RepositoryRecord& r, std::string& err) {
    r.alias = as_str(e["alias"]); r.name = as_str(e["name"]); r.url = as_str(e["url"]);
    r.type = as_str(e["type"]);
    as_bool(e["enabled"], r.enabled); as_bool(e["gpgcheck"], r.gpgcheck);
    as_bool(e["autorefresh"], r.autorefresh); as_int(e["priority"], r.priority);
    if (r.alias.empty()) { err = "repository alias must be non-empty"; return false; }
    if (r.url.empty()) { err = "repository url must be non-empty"; return false; }
    return true;
}
static bool elem_svc(const Json::Value& e, ServiceRecord& r, std::string& err) {
    r.name = as_str(e["name"]); r.state = as_str(e["state"]);
    if (r.name.empty()) { err = "service name must be non-empty"; return false; }
    if (r.state != "enabled" && r.state != "disabled" && r.state != "masked") {
        err = "service state must be enabled|disabled|masked"; return false;
    }
    return true;
}
static bool elem_file(const Json::Value& e, ManagedFileRecord& r, std::string& err) {
    r.name = as_str(e["name"]); r.type = as_str(e["type"]); r.mode = as_str(e["mode"]);
    r.user = as_str(e["user"]); r.group = as_str(e["group"]);
    r.sha256 = as_str(e["sha256"]); r.target = as_str(e["target"]);
    r.content_ref = as_str(e["content_ref"]); r.package_name = as_str(e["package_name"]);
    if (r.type != "file" && r.type != "link" && r.type != "dir") {
        err = "config file type must be file|link|dir"; return false;
    }
    return true;
}
static bool elem_baseline(const Json::Value& e, ManagedBaselineRecord& r, std::string&) {
    r.name = as_str(e["name"]); r.type = as_str(e["type"]); r.mode = as_str(e["mode"]);
    r.user = as_str(e["user"]); r.group = as_str(e["group"]);
    r.sha256 = as_str(e["sha256"]); r.target = as_str(e["target"]);
    r.package_name = as_str(e["package_name"]);
    if (e.isMember("changes") && e["changes"].isArray())
        for (auto& c : e["changes"]) r.changes.push_back(as_str(c));
    return true;
}
static bool elem_unmanaged(const Json::Value& e, UnmanagedFileRecord& r, std::string&) {
    r.name = as_str(e["name"]); r.type = as_str(e["type"]); r.mode = as_str(e["mode"]);
    r.user = as_str(e["user"]); r.group = as_str(e["group"]);
    r.sha256 = as_str(e["sha256"]); r.target = as_str(e["target"]);
    return true;
}

// Validate a Json::Value tree as a Manifest. is_desired controls observational-
// scope rejection (a desired manifest must not carry a non-empty observational
// scope; empty/absent is tolerated and dropped).
static bool validate_manifest(const Json::Value& root, bool is_desired,
                              Manifest& out, Diagnostic& err) {
    if (!root.isObject()) { err = manifest_err("manifest root is not an object"); return false; }
    if (!root.isMember("meta") || !root["meta"].isObject()) {
        err = manifest_err("missing meta object"); return false;
    }
    const Json::Value& meta = root["meta"];
    int fv = 0;
    if (!as_int(meta["format_version"], fv) || fv != 1) {
        err = manifest_err("meta.format_version must be 1"); return false;
    }
    out.meta.format_version = 1;
    out.meta.generator = as_str(meta["generator"]);
    out.meta.created_at = as_str(meta["created_at"]);
    out.meta.desired_sha256 = as_str(meta["desired_sha256"]);

    std::string e;
    if (root.isMember("packages")) {
        PackagesScope s;
        if (!parse_scope(root["packages"], s, elem_pkg, e)) { err = manifest_err(e); return false; }
        out.packages = s;
    }
    if (root.isMember("repositories")) {
        RepositoriesScope s;
        if (!parse_scope(root["repositories"], s, elem_repo, e)) { err = manifest_err(e); return false; }
        out.repositories = s;
    }
    if (root.isMember("services")) {
        ServicesScope s;
        if (!parse_scope(root["services"], s, elem_svc, e)) { err = manifest_err(e); return false; }
        out.services = s;
    }
    if (root.isMember("config_files")) {
        ConfigFilesScope s;
        if (!parse_scope(root["config_files"], s, elem_file, e)) { err = manifest_err(e); return false; }
        out.config_files = s;
    }
    // Observational scopes
    if (root.isMember("changed_managed_files")) {
        ChangedManagedFilesScope s;
        if (!parse_scope(root["changed_managed_files"], s, elem_baseline, e)) {
            err = manifest_err(e); return false;
        }
        if (is_desired) {
            if (!s.elements.empty()) {
                err = manifest_err("desired manifest carries a non-empty "
                                   "changed_managed_files observational scope");
                return false;
            }
            // empty observational scope tolerated and dropped
        } else {
            out.changed_managed_files = s;
        }
    }
    if (root.isMember("unmanaged_files")) {
        UnmanagedFilesScope s;
        if (!parse_scope(root["unmanaged_files"], s, elem_unmanaged, e)) {
            err = manifest_err(e); return false;
        }
        if (is_desired) {
            if (!s.elements.empty()) {
                err = manifest_err("desired manifest carries a non-empty "
                                   "unmanaged_files observational scope");
                return false;
            }
        } else {
            out.unmanaged_files = s;
        }
    }
    return true;
}

LoadResult parse_manifest(const std::string& text, ManifestFormat fmt, bool is_desired) {
    LoadResult lr;
    Json::Value root;
    if (fmt == ManifestFormat::Json) {
        Json::CharReaderBuilder b;
        std::string errs;
        std::istringstream is(text);
        if (!Json::parseFromStream(b, is, &root, &errs)) {
            lr.ok = false; lr.error = manifest_err("JSON parse error: " + errs);
            return lr;
        }
    } else {
        // YAML safe profile.
        std::vector<YAML::Node> docs;
        try {
            docs = YAML::LoadAll(text);
        } catch (const std::exception& ex) {
            lr.ok = false; lr.error = manifest_err(std::string("YAML parse error: ") + ex.what());
            return lr;
        }
        if (docs.size() != 1) {
            lr.ok = false;
            lr.error = manifest_err("YAML safe profile rejects multi-document streams");
            return lr;
        }
        std::string yerr;
        if (!yaml_to_json(docs[0], root, yerr)) {
            lr.ok = false; lr.error = manifest_err("YAML safe profile: " + yerr);
            return lr;
        }
    }
    Manifest m;
    Diagnostic err;
    if (!validate_manifest(root, is_desired, m, err)) {
        lr.ok = false; lr.error = err; return lr;
    }
    lr.ok = true; lr.manifest = m;
    lr.desired_sha256 = canonical_hash(m);
    return lr;
}

static bool read_file(const std::string& path, std::string& out) {
    std::ifstream f(path, std::ios::binary);
    if (!f) return false;
    std::ostringstream ss; ss << f.rdbuf();
    out = ss.str();
    return true;
}

LoadResult load_desired_manifest(const std::string& manifest_path,
                                 const std::optional<ManifestFormat>& explicit_fmt,
                                 const Config& cfg) {
    LoadResult lr;
    std::string text;
    if (!read_file(manifest_path, text)) {
        lr.ok = false;
        lr.error = Diagnostic{Severity::Error, "invocation",
                              "manifest unreadable: " + manifest_path};
        return lr;
    }
    ManifestFormat fmt = resolve_format(explicit_fmt, manifest_path, cfg.manifest_format);
    lr = parse_manifest(text, fmt, /*is_desired=*/true);
    // signature verification is abstracted; default-on but no keyring path here
    // means it is a no-op for offline parsing (documented in the report).
    return lr;
}

LoadResult load_state_dump(const std::string& state_path,
                           const std::optional<ManifestFormat>& explicit_fmt,
                           const Config& cfg) {
    LoadResult lr;
    std::string text;
    if (!read_file(state_path, text)) {
        lr.ok = false;
        lr.error = Diagnostic{Severity::Error, "invocation",
                              "state dump unreadable: " + state_path};
        return lr;
    }
    ManifestFormat fmt = resolve_format(explicit_fmt, state_path, cfg.manifest_format);
    LoadResult parsed = parse_manifest(text, fmt, /*is_desired=*/false);
    if (!parsed.ok) {
        // a malformed state dump is an invocation error (exit 2), not a manifest
        // error: re-stamp the domain.
        parsed.error.domain = "invocation";
    }
    return parsed;
}

AppliedResult load_applied_record(const std::string& root) {
    AppliedResult ar;
    fs::path p = fs::path(root) / "usr" / "lib" / "zypper-declarative" / "applied.json";
    std::error_code ec;
    if (!fs::exists(p, ec)) {
        ar.present = false; ar.ok = true; // all scopes empty
        return ar;
    }
    std::string text;
    if (!read_file(p.string(), text)) {
        ar.ok = false;
        ar.error = Diagnostic{Severity::Error, "files",
                              "applied record present but unreadable: " + p.string()};
        return ar;
    }
    LoadResult lr = parse_manifest(text, ManifestFormat::Json, /*is_desired=*/false);
    if (!lr.ok) {
        ar.ok = false;
        ar.error = Diagnostic{Severity::Error, "files",
                              "applied record present but unparseable"};
        return ar;
    }
    ar.ok = true; ar.present = true; ar.record = lr.manifest;
    return ar;
}

bool write_manifest(const Manifest& m, ManifestFormat fmt,
                    const std::optional<std::string>& out_path) {
    std::string doc = serialize_manifest(m, fmt, /*pretty=*/true);
    if (!out_path.has_value()) {
        std::fwrite(doc.data(), 1, doc.size(), stdout);
        return true;
    }
    std::ofstream f(*out_path, std::ios::binary | std::ios::trunc);
    if (!f) return false;
    f << doc;
    f.flush();
    return static_cast<bool>(f);
}

} // namespace zd
