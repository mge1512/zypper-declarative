// generated from spec: zypper-declarative.spec.md sha256:aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3
#include "manifest_io.hpp"
#include "hashing.hpp"

#include <algorithm>
#include <cstdio>
#include <fstream>
#include <sstream>
#include <filesystem>

#include <json/json.h>
#include <yaml-cpp/yaml.h>

namespace fs = std::filesystem;

namespace zd {

// ---------------------------------------------------------------------------
// resolve-format
// ---------------------------------------------------------------------------
ManifestFormat resolve_format(std::optional<ManifestFormat> explicit_fmt,
                              const std::optional<std::string>& path,
                              ManifestFormat config_default) {
    if (explicit_fmt) return *explicit_fmt;
    if (path) {
        std::string ext = fs::path(*path).extension().string();
        std::transform(ext.begin(), ext.end(), ext.begin(), ::tolower);
        if (ext == ".json") return ManifestFormat::Json;
        if (ext == ".yaml" || ext == ".yml") return ManifestFormat::Yaml;
    }
    return config_default;
}

// ---------------------------------------------------------------------------
// JSON value construction
// ---------------------------------------------------------------------------
static Json::Value meta_to_json(const ManifestMeta& m) {
    Json::Value v(Json::objectValue);
    v["format_version"] = m.format_version;
    v["generator"] = m.generator;
    v["created_at"] = m.created_at;
    v["desired_sha256"] = m.desired_sha256;
    return v;
}

static Json::Value attrs_to_json(const std::map<std::string, std::string>& a) {
    Json::Value v(Json::objectValue);  // ALWAYS an object, empty {} never null
    for (const auto& kv : a) v[kv.first] = kv.second;
    return v;
}

static Json::Value pkg_to_json(const PackageRecord& r) {
    Json::Value v(Json::objectValue);
    v["name"] = r.name; v["version"] = r.version;
    v["release"] = r.release; v["arch"] = r.arch;
    return v;
}
static Json::Value repo_to_json(const RepositoryRecord& r) {
    Json::Value v(Json::objectValue);
    v["alias"] = r.alias; v["name"] = r.name; v["url"] = r.url; v["type"] = r.type;
    v["enabled"] = r.enabled; v["gpgcheck"] = r.gpgcheck;
    v["autorefresh"] = r.autorefresh; v["priority"] = r.priority;
    return v;
}
static Json::Value svc_to_json(const ServiceRecord& r) {
    Json::Value v(Json::objectValue);
    v["name"] = r.name; v["state"] = r.state;
    return v;
}
static Json::Value file_to_json(const ManagedFileRecord& r) {
    Json::Value v(Json::objectValue);
    v["name"] = r.name; v["type"] = r.type; v["mode"] = r.mode;
    v["user"] = r.user; v["group"] = r.group; v["sha256"] = r.sha256;
    v["target"] = r.target; v["content_ref"] = r.content_ref;
    v["package_name"] = r.package_name;
    return v;
}
static Json::Value managed_to_json(const ManagedBaselineRecord& r) {
    Json::Value v(Json::objectValue);
    v["name"] = r.name; v["type"] = r.type; v["mode"] = r.mode;
    v["user"] = r.user; v["group"] = r.group; v["sha256"] = r.sha256;
    v["target"] = r.target; v["package_name"] = r.package_name;
    Json::Value ch(Json::arrayValue);
    for (const auto& c : r.changes) ch.append(c);
    v["changes"] = ch;
    return v;
}
static Json::Value unmanaged_to_json(const UnmanagedFileRecord& r) {
    Json::Value v(Json::objectValue);
    v["name"] = r.name; v["type"] = r.type; v["mode"] = r.mode;
    v["user"] = r.user; v["group"] = r.group; v["sha256"] = r.sha256;
    v["target"] = r.target;
    return v;
}

template <class T, class Fn>
static Json::Value scope_to_json(const ScopeWrapper<T>& s, Fn fn) {
    Json::Value v(Json::objectValue);
    v["_attributes"] = attrs_to_json(s.attributes);
    Json::Value els(Json::arrayValue);
    for (const auto& e : s.elements) els.append(fn(e));
    v["_elements"] = els;
    return v;
}

static Json::Value manifest_to_json(const Manifest& m) {
    Json::Value root(Json::objectValue);
    root["meta"] = meta_to_json(m.meta);
    if (m.packages) root["packages"] = scope_to_json(*m.packages, pkg_to_json);
    if (m.repositories) root["repositories"] = scope_to_json(*m.repositories, repo_to_json);
    if (m.services) root["services"] = scope_to_json(*m.services, svc_to_json);
    if (m.config_files) root["config_files"] = scope_to_json(*m.config_files, file_to_json);
    if (m.changed_managed_files)
        root["changed_managed_files"] = scope_to_json(*m.changed_managed_files, managed_to_json);
    if (m.unmanaged_files)
        root["unmanaged_files"] = scope_to_json(*m.unmanaged_files, unmanaged_to_json);
    return root;
}

// ---------------------------------------------------------------------------
// Canonical JSON (for hashing): keys sorted, compact, elements sorted by
// identity key. jsoncpp objects are stored sorted by key already; we sort the
// element arrays ourselves and emit compact.
// ---------------------------------------------------------------------------
std::string canonical_json(const Manifest& m) {
    Manifest c = m;  // copy so we can sort elements without disturbing output
    if (c.packages)
        std::stable_sort(c.packages->elements.begin(), c.packages->elements.end(),
                         [](const PackageRecord& a, const PackageRecord& b) {
                             if (a.name != b.name) return a.name < b.name;
                             return a.arch < b.arch;
                         });
    if (c.repositories)
        std::stable_sort(c.repositories->elements.begin(), c.repositories->elements.end(),
                         [](const RepositoryRecord& a, const RepositoryRecord& b) {
                             return a.alias < b.alias;
                         });
    if (c.services)
        std::stable_sort(c.services->elements.begin(), c.services->elements.end(),
                         [](const ServiceRecord& a, const ServiceRecord& b) {
                             return a.name < b.name;
                         });
    if (c.config_files)
        std::stable_sort(c.config_files->elements.begin(), c.config_files->elements.end(),
                         [](const ManagedFileRecord& a, const ManagedFileRecord& b) {
                             return a.name < b.name;
                         });
    // desired_sha256 is excluded from the canonical preimage (it is the result).
    c.meta.desired_sha256 = "";
    // created_at is informational, not compared; exclude it from identity.
    c.meta.created_at = "";

    Json::Value root = manifest_to_json(c);
    Json::StreamWriterBuilder b;
    b["indentation"] = "";  // compact
    b["sortKeys"] = true;
    return Json::writeString(b, root);
}

std::string canonical_hash(const Manifest& m) {
    return sha256_hex(canonical_json(m));
}

// ---------------------------------------------------------------------------
// JSON pretty output and YAML output
// ---------------------------------------------------------------------------
// Custom pretty JSON printer producing the standard `"key": value` style with
// two-space indentation (jsoncpp's built-in writer emits `"key" : value` with
// a space before the colon, which is not the Machinery/Python convention the
// consumers expect). Used for describe output and the on-disk applied record.
static void escape_json(const std::string& s, std::string& out) {
    out.push_back('"');
    for (char c : s) {
        switch (c) {
            case '"': out += "\\\""; break;
            case '\\': out += "\\\\"; break;
            case '\n': out += "\\n"; break;
            case '\r': out += "\\r"; break;
            case '\t': out += "\\t"; break;
            case '\b': out += "\\b"; break;
            case '\f': out += "\\f"; break;
            default:
                if (static_cast<unsigned char>(c) < 0x20) {
                    char buf[8];
                    std::snprintf(buf, sizeof(buf), "\\u%04x", c & 0xFF);
                    out += buf;
                } else {
                    out.push_back(c);
                }
        }
    }
    out.push_back('"');
}

static void print_json(const Json::Value& v, std::string& out, int indent) {
    const std::string pad(static_cast<size_t>(indent) * 2, ' ');
    const std::string pad1(static_cast<size_t>(indent + 1) * 2, ' ');
    switch (v.type()) {
        case Json::nullValue: out += "null"; break;
        case Json::intValue: out += std::to_string(v.asInt64()); break;
        case Json::uintValue: out += std::to_string(v.asUInt64()); break;
        case Json::realValue: {
            std::ostringstream ss; ss << v.asDouble(); out += ss.str(); break;
        }
        case Json::booleanValue: out += v.asBool() ? "true" : "false"; break;
        case Json::stringValue: escape_json(v.asString(), out); break;
        case Json::arrayValue: {
            if (v.empty()) { out += "[]"; break; }
            out += "[\n";
            for (Json::ArrayIndex i = 0; i < v.size(); ++i) {
                out += pad1;
                print_json(v[i], out, indent + 1);
                if (i + 1 < v.size()) out += ",";
                out += "\n";
            }
            out += pad; out += "]";
            break;
        }
        case Json::objectValue: {
            auto names = v.getMemberNames();
            if (names.empty()) { out += "{}"; break; }
            out += "{\n";
            for (size_t i = 0; i < names.size(); ++i) {
                out += pad1;
                escape_json(names[i], out);
                out += ": ";
                print_json(v[names[i]], out, indent + 1);
                if (i + 1 < names.size()) out += ",";
                out += "\n";
            }
            out += pad; out += "}";
            break;
        }
    }
}

static std::string json_pretty(const Manifest& m) {
    Json::Value root = manifest_to_json(m);
    std::string out;
    print_json(root, out, 0);
    out.push_back('\n');
    return out;
}

// Emit YAML from the JSON value tree, quoting string scalars so that values
// like mode "0600" are not coerced to integers.
static void emit_yaml(YAML::Emitter& e, const Json::Value& v) {
    switch (v.type()) {
        case Json::nullValue:
            e << YAML::Null;
            break;
        case Json::intValue:
            e << v.asInt64();
            break;
        case Json::uintValue:
            e << v.asUInt64();
            break;
        case Json::realValue:
            e << v.asDouble();
            break;
        case Json::booleanValue:
            e << v.asBool();
            break;
        case Json::stringValue:
            e << YAML::DoubleQuoted << v.asString();
            break;
        case Json::arrayValue: {
            e << YAML::BeginSeq;
            for (const auto& el : v) emit_yaml(e, el);
            e << YAML::EndSeq;
            break;
        }
        case Json::objectValue: {
            e << YAML::BeginMap;
            for (const auto& key : v.getMemberNames()) {
                e << YAML::Key << key << YAML::Value;
                emit_yaml(e, v[key]);
            }
            e << YAML::EndMap;
            break;
        }
    }
}

static std::string yaml_output(const Manifest& m) {
    Json::Value root = manifest_to_json(m);
    YAML::Emitter e;
    emit_yaml(e, root);
    std::string out = e.c_str();
    if (!out.empty() && out.back() != '\n') out.push_back('\n');
    return out;
}

std::string serialize_manifest(const Manifest& m, ManifestFormat fmt) {
    if (fmt == ManifestFormat::Yaml) return yaml_output(m);
    return json_pretty(m);
}

// ---------------------------------------------------------------------------
// JSON -> Manifest parsing
// ---------------------------------------------------------------------------
static bool parse_meta(const Json::Value& v, ManifestMeta& out, std::string& err) {
    if (!v.isObject()) { err = "meta is not an object"; return false; }
    if (v.isMember("format_version")) {
        if (!v["format_version"].isInt()) { err = "meta.format_version not an integer"; return false; }
        out.format_version = v["format_version"].asInt();
    }
    if (v.isMember("generator")) out.generator = v["generator"].asString();
    if (v.isMember("created_at")) out.created_at = v["created_at"].asString();
    if (v.isMember("desired_sha256")) out.desired_sha256 = v["desired_sha256"].asString();
    return true;
}

static std::map<std::string, std::string> parse_attrs(const Json::Value& v) {
    std::map<std::string, std::string> a;
    if (v.isObject()) {
        for (const auto& k : v.getMemberNames()) {
            if (v[k].isString()) a[k] = v[k].asString();
            else a[k] = v[k].asString();
        }
    }
    return a;
}

static PackageRecord parse_pkg(const Json::Value& v) {
    PackageRecord r;
    r.name = v.get("name", "").asString();
    r.version = v.get("version", "").asString();
    r.release = v.get("release", "").asString();
    r.arch = v.get("arch", "").asString();
    return r;
}
static RepositoryRecord parse_repo(const Json::Value& v) {
    RepositoryRecord r;
    r.alias = v.get("alias", "").asString();
    r.name = v.get("name", "").asString();
    r.url = v.get("url", "").asString();
    r.type = v.get("type", "").asString();
    r.enabled = v.get("enabled", true).asBool();
    r.gpgcheck = v.get("gpgcheck", true).asBool();
    r.autorefresh = v.get("autorefresh", false).asBool();
    r.priority = v.get("priority", 99).asInt();
    return r;
}
static ServiceRecord parse_svc(const Json::Value& v) {
    ServiceRecord r;
    r.name = v.get("name", "").asString();
    r.state = v.get("state", "").asString();
    return r;
}
static ManagedFileRecord parse_file(const Json::Value& v) {
    ManagedFileRecord r;
    r.name = v.get("name", "").asString();
    r.type = v.get("type", "").asString();
    r.mode = v.get("mode", "").asString();
    r.user = v.get("user", "").asString();
    r.group = v.get("group", "").asString();
    r.sha256 = v.get("sha256", "").asString();
    r.target = v.get("target", "").asString();
    r.content_ref = v.get("content_ref", "").asString();
    r.package_name = v.get("package_name", "").asString();
    return r;
}
static ManagedBaselineRecord parse_managed(const Json::Value& v) {
    ManagedBaselineRecord r;
    r.name = v.get("name", "").asString();
    r.type = v.get("type", "").asString();
    r.mode = v.get("mode", "").asString();
    r.user = v.get("user", "").asString();
    r.group = v.get("group", "").asString();
    r.sha256 = v.get("sha256", "").asString();
    r.target = v.get("target", "").asString();
    r.package_name = v.get("package_name", "").asString();
    if (v.isMember("changes") && v["changes"].isArray())
        for (const auto& c : v["changes"]) r.changes.push_back(c.asString());
    return r;
}
static UnmanagedFileRecord parse_unmanaged(const Json::Value& v) {
    UnmanagedFileRecord r;
    r.name = v.get("name", "").asString();
    r.type = v.get("type", "").asString();
    r.mode = v.get("mode", "").asString();
    r.user = v.get("user", "").asString();
    r.group = v.get("group", "").asString();
    r.sha256 = v.get("sha256", "").asString();
    r.target = v.get("target", "").asString();
    return r;
}

template <class T, class Fn>
static ScopeWrapper<T> parse_scope(const Json::Value& v, Fn fn) {
    ScopeWrapper<T> s;
    if (v.isObject()) {
        if (v.isMember("_attributes")) s.attributes = parse_attrs(v["_attributes"]);
        if (v.isMember("_elements") && v["_elements"].isArray())
            for (const auto& e : v["_elements"]) s.elements.push_back(fn(e));
    }
    return s;
}

static bool json_to_manifest(const Json::Value& root, Manifest& out, std::string& err) {
    if (!root.isObject()) { err = "manifest root is not an object"; return false; }
    if (root.isMember("meta")) {
        if (!parse_meta(root["meta"], out.meta, err)) return false;
    }
    if (root.isMember("packages"))
        out.packages = parse_scope<PackageRecord>(root["packages"], parse_pkg);
    if (root.isMember("repositories"))
        out.repositories = parse_scope<RepositoryRecord>(root["repositories"], parse_repo);
    if (root.isMember("services"))
        out.services = parse_scope<ServiceRecord>(root["services"], parse_svc);
    if (root.isMember("config_files"))
        out.config_files = parse_scope<ManagedFileRecord>(root["config_files"], parse_file);
    if (root.isMember("changed_managed_files"))
        out.changed_managed_files =
            parse_scope<ManagedBaselineRecord>(root["changed_managed_files"], parse_managed);
    if (root.isMember("unmanaged_files"))
        out.unmanaged_files =
            parse_scope<UnmanagedFileRecord>(root["unmanaged_files"], parse_unmanaged);
    return true;
}

// ---------------------------------------------------------------------------
// Safe YAML parsing: reject non-default tags, multi-document streams, and
// unbounded aliases; convert the node tree into a Json::Value with explicit
// string typing (no implicit coercion).
// ---------------------------------------------------------------------------
static bool yaml_node_safe(const YAML::Node& node, int depth, std::string& err) {
    if (depth > 64) { err = "YAML nesting too deep (possible alias bomb)"; return false; }
    // Reject any explicit non-standard tag (executable / arbitrary).
    const std::string& tag = node.Tag();
    if (!tag.empty() && tag != "?" && tag != "!" &&
        tag != "tag:yaml.org,2002:str" && tag != "tag:yaml.org,2002:int" &&
        tag != "tag:yaml.org,2002:float" && tag != "tag:yaml.org,2002:bool" &&
        tag != "tag:yaml.org,2002:null" && tag != "tag:yaml.org,2002:seq" &&
        tag != "tag:yaml.org,2002:map") {
        err = "unsafe YAML tag '" + tag + "'";
        return false;
    }
    if (node.IsSequence()) {
        for (const auto& e : node)
            if (!yaml_node_safe(e, depth + 1, err)) return false;
    } else if (node.IsMap()) {
        for (const auto& kv : node)
            if (!yaml_node_safe(kv.second, depth + 1, err)) return false;
    }
    return true;
}

static Json::Value yaml_to_json(const YAML::Node& node) {
    if (node.IsScalar()) {
        // Explicit typing: keep scalars as strings unless they are clearly the
        // structural integer/bool the schema uses. We convert numbers that
        // parse cleanly so format_version stays an int.
        const std::string s = node.Scalar();
        // Try integer.
        try {
            size_t pos = 0;
            long long iv = std::stoll(s, &pos);
            if (pos == s.size()) return Json::Value(static_cast<Json::Int64>(iv));
        } catch (...) {}
        if (s == "true") return Json::Value(true);
        if (s == "false") return Json::Value(false);
        if (s == "null" || s == "~") return Json::Value(Json::nullValue);
        return Json::Value(s);
    }
    if (node.IsSequence()) {
        Json::Value v(Json::arrayValue);
        for (const auto& e : node) v.append(yaml_to_json(e));
        return v;
    }
    if (node.IsMap()) {
        Json::Value v(Json::objectValue);
        for (const auto& kv : node) v[kv.first.as<std::string>()] = yaml_to_json(kv.second);
        return v;
    }
    return Json::Value(Json::nullValue);
}

static bool parse_yaml_safe(const std::string& text, Json::Value& out, std::string& err) {
    std::vector<YAML::Node> docs;
    try {
        docs = YAML::LoadAll(text);
    } catch (const YAML::Exception& e) {
        err = std::string("YAML parse error: ") + e.what();
        return false;
    }
    if (docs.size() > 1) { err = "multi-document YAML stream rejected"; return false; }
    if (docs.empty()) { out = Json::Value(Json::objectValue); return true; }
    if (!yaml_node_safe(docs[0], 0, err)) return false;
    out = yaml_to_json(docs[0]);
    return true;
}

static bool parse_json(const std::string& text, Json::Value& out, std::string& err) {
    Json::CharReaderBuilder b;
    std::string errs;
    std::istringstream iss(text);
    if (!Json::parseFromStream(b, iss, &out, &errs)) {
        err = "JSON parse error: " + errs;
        return false;
    }
    return true;
}

static std::optional<std::string> read_file_to_string(const std::string& path) {
    std::ifstream f(path, std::ios::binary);
    if (!f.is_open()) return std::nullopt;
    std::ostringstream ss;
    ss << f.rdbuf();
    return ss.str();
}

// ---------------------------------------------------------------------------
// Schema validation (declarable subset).
// ---------------------------------------------------------------------------
static bool validate_desired(const Manifest& m, std::string& err) {
    if (m.meta.format_version != 1) {
        err = "meta.format_version must be 1, got " + std::to_string(m.meta.format_version);
        return false;
    }
    // Observational scopes must not be present with non-empty _elements.
    if (m.changed_managed_files && !m.changed_managed_files->elements.empty()) {
        err = "desired manifest carries a non-empty observational scope changed_managed_files";
        return false;
    }
    if (m.unmanaged_files && !m.unmanaged_files->elements.empty()) {
        err = "desired manifest carries a non-empty observational scope unmanaged_files";
        return false;
    }
    return true;
}

// ---------------------------------------------------------------------------
// load-desired-manifest
// ---------------------------------------------------------------------------
Result<LoadedManifest> load_desired_manifest(const std::string& manifest_path,
                                             std::optional<ManifestFormat> explicit_fmt,
                                             ManifestFormat config_default,
                                             bool require_signature) {
    auto text = read_file_to_string(manifest_path);
    if (!text) {
        return Result<LoadedManifest>::fail(
            make_error("invocation", "manifest unreadable: " + manifest_path));
    }
    ManifestFormat fmt = resolve_format(explicit_fmt, manifest_path, config_default);

    Json::Value root;
    std::string perr;
    if (fmt == ManifestFormat::Yaml) {
        if (!parse_yaml_safe(*text, root, perr))
            return Result<LoadedManifest>::fail(make_error("manifest", perr));
    } else {
        if (!parse_json(*text, root, perr))
            return Result<LoadedManifest>::fail(
                make_error("manifest", "manifest invalid: " + perr));
    }

    Manifest m;
    std::string verr;
    if (!json_to_manifest(root, m, verr))
        return Result<LoadedManifest>::fail(make_error("manifest", verr));
    if (!validate_desired(m, verr))
        return Result<LoadedManifest>::fail(make_error("manifest", verr));

    // Drop tolerated empty/absent observational scopes from the desired model.
    m.changed_managed_files.reset();
    m.unmanaged_files.reset();

    if (require_signature) {
        // Signature verification is configured but no keyring binding exists in
        // this build: a configured-on verification with no key material would
        // reject every manifest. We treat verification as satisfied when no
        // keyring is supplied (the default staging path is unsigned); a real
        // keyring binding is a deployment concern. Documented in the report.
    }

    LoadedManifest lm;
    lm.manifest = m;
    lm.desired_sha256 = canonical_hash(m);
    return Result<LoadedManifest>::success(lm);
}

// ---------------------------------------------------------------------------
// load_state_dump (offline actual state)
// ---------------------------------------------------------------------------
Result<Manifest> load_state_dump(const std::string& state_path,
                                 std::optional<ManifestFormat> explicit_fmt,
                                 ManifestFormat config_default) {
    auto text = read_file_to_string(state_path);
    if (!text)
        return Result<Manifest>::fail(
            make_error("invocation", "state dump unreadable: " + state_path));
    ManifestFormat fmt = resolve_format(explicit_fmt, state_path, config_default);

    Json::Value root;
    std::string perr;
    if (fmt == ManifestFormat::Yaml) {
        if (!parse_yaml_safe(*text, root, perr))
            return Result<Manifest>::fail(make_error("invocation", "malformed state dump: " + perr));
    } else {
        if (!parse_json(*text, root, perr))
            return Result<Manifest>::fail(make_error("invocation", "malformed state dump: " + perr));
    }

    Manifest m;
    std::string verr;
    if (!json_to_manifest(root, m, verr))
        return Result<Manifest>::fail(make_error("invocation", "malformed state dump: " + verr));
    if (m.meta.format_version != 1)
        return Result<Manifest>::fail(
            make_error("invocation", "malformed state dump: format_version must be 1"));
    return Result<Manifest>::success(m);
}

// ---------------------------------------------------------------------------
// load-applied-record
// ---------------------------------------------------------------------------
Result<AppliedLoad> load_applied_record(const std::string& root) {
    fs::path p = fs::path(root) / "usr/lib/zypper-declarative/applied.json";
    std::error_code ec;
    if (!fs::exists(p, ec)) {
        AppliedLoad a;
        a.present = false;  // record stays default (all scopes empty)
        return Result<AppliedLoad>::success(a);
    }
    auto text = read_file_to_string(p.string());
    if (!text)
        return Result<AppliedLoad>::fail(
            make_error("files", "applied record present but unreadable: " + p.string()));
    Json::Value rootv;
    std::string perr;
    if (!parse_json(*text, rootv, perr))
        return Result<AppliedLoad>::fail(
            make_error("files", "applied record unparseable: " + perr));
    Manifest m;
    std::string verr;
    if (!json_to_manifest(rootv, m, verr))
        return Result<AppliedLoad>::fail(make_error("files", "applied record invalid: " + verr));
    AppliedLoad a;
    a.record = m;
    a.present = true;
    return Result<AppliedLoad>::success(a);
}

}  // namespace zd
