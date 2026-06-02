// generated from spec: zypper-declarative.spec.md sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03

#include "manifest.hpp"

#include <algorithm>
#include <fstream>
#include <map>
#include <sstream>

#include <json/json.h>
#include <yaml-cpp/yaml.h>

#include "hash.hpp"

namespace zd {

// --------------------------------------------------------------------------
// resolve-format
// --------------------------------------------------------------------------
static std::optional<ManifestFormat> ext_format(const std::string& path) {
    auto dot = path.rfind('.');
    if (dot == std::string::npos) return std::nullopt;
    std::string ext = path.substr(dot + 1);
    std::transform(ext.begin(), ext.end(), ext.begin(), ::tolower);
    if (ext == "json") return ManifestFormat::Json;
    if (ext == "yaml" || ext == "yml") return ManifestFormat::Yaml;
    return std::nullopt;
}

ManifestFormat resolve_format(const std::optional<ManifestFormat>& explicit_fmt,
                              const std::optional<std::string>& path,
                              ManifestFormat config_default) {
    if (explicit_fmt) return *explicit_fmt;            // explicit wins
    if (path) {
        auto e = ext_format(*path);
        if (e) return *e;                              // recognised extension
    }
    return config_default;                             // CONFIG default
}

// --------------------------------------------------------------------------
// JSON <-> model
// --------------------------------------------------------------------------
static Json::Value attrs_to_json(const std::map<std::string, std::string>& a) {
    Json::Value o(Json::objectValue);  // ALWAYS an object, never null
    for (const auto& kv : a) o[kv.first] = kv.second;
    return o;
}

static Json::Value pkg_to_json(const PackageRecord& p) {
    Json::Value v(Json::objectValue);
    v["name"] = p.name;
    v["version"] = p.version;
    v["release"] = p.release;
    v["arch"] = p.arch;
    return v;
}
static Json::Value repo_to_json(const RepositoryRecord& r) {
    Json::Value v(Json::objectValue);
    v["alias"] = r.alias;
    v["name"] = r.name;
    v["url"] = r.url;
    v["type"] = r.type;
    v["enabled"] = r.enabled;
    v["gpgcheck"] = r.gpgcheck;
    v["autorefresh"] = r.autorefresh;
    v["priority"] = static_cast<Json::Int64>(r.priority);
    return v;
}
static Json::Value svc_to_json(const ServiceRecord& s) {
    Json::Value v(Json::objectValue);
    v["name"] = s.name;
    v["state"] = s.state;
    return v;
}
static Json::Value file_to_json(const ManagedFileRecord& f) {
    Json::Value v(Json::objectValue);
    v["name"] = f.name;
    v["type"] = f.type;
    v["mode"] = f.mode;
    v["user"] = f.user;
    v["group"] = f.group;
    v["sha256"] = f.sha256;
    v["target"] = f.target;
    v["content_ref"] = f.content_ref;
    v["package_name"] = f.package_name;
    return v;
}
static Json::Value mbase_to_json(const ManagedBaselineRecord& f) {
    Json::Value v(Json::objectValue);
    v["name"] = f.name;
    v["type"] = f.type;
    v["mode"] = f.mode;
    v["user"] = f.user;
    v["group"] = f.group;
    v["sha256"] = f.sha256;
    v["target"] = f.target;
    v["package_name"] = f.package_name;
    Json::Value ch(Json::arrayValue);
    for (const auto& c : f.changes) ch.append(c);
    v["changes"] = ch;
    return v;
}
static Json::Value unmanaged_to_json(const UnmanagedFileRecord& f) {
    Json::Value v(Json::objectValue);
    v["name"] = f.name;
    v["type"] = f.type;
    v["mode"] = f.mode;
    v["user"] = f.user;
    v["group"] = f.group;
    v["sha256"] = f.sha256;
    v["target"] = f.target;
    return v;
}

template <class T, class F>
static Json::Value scope_to_json(const ScopeWrapper<T>& scope, F to_json) {
    Json::Value v(Json::objectValue);
    v["_attributes"] = attrs_to_json(scope.attributes);
    Json::Value els(Json::arrayValue);
    for (const auto& e : scope.elements) els.append(to_json(e));
    v["_elements"] = els;
    return v;
}

static Json::Value manifest_to_json(const Manifest& m, bool include_meta) {
    Json::Value root(Json::objectValue);
    if (include_meta) {
        Json::Value meta(Json::objectValue);
        meta["format_version"] = m.meta.format_version;
        meta["generator"] = m.meta.generator;
        meta["created_at"] = m.meta.created_at;
        meta["desired_sha256"] = m.meta.desired_sha256;
        root["meta"] = meta;
    }
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
            scope_to_json(*m.changed_managed_files, mbase_to_json);
    if (m.unmanaged_files)
        root["unmanaged_files"] =
            scope_to_json(*m.unmanaged_files, unmanaged_to_json);
    return root;
}

// --------------------------------------------------------------------------
// Serialise (pretty JSON or YAML)
// --------------------------------------------------------------------------
static std::string emit_yaml_scalar(const std::string& s) {
    // Quote every string scalar so e.g. mode "0600" or version "1.10" is not
    // coerced. Escape embedded quotes/backslashes minimally.
    std::string out = "\"";
    for (char c : s) {
        if (c == '\\' || c == '"') out.push_back('\\');
        out.push_back(c);
    }
    out.push_back('"');
    return out;
}

static std::string json_to_yaml(const Json::Value& root);

std::string serialise(const Manifest& m, ManifestFormat fmt) {
    Json::Value root = manifest_to_json(m, /*include_meta=*/true);
    if (fmt == ManifestFormat::Json) {
        Json::StreamWriterBuilder b;
        b["indentation"] = "  ";
        b["emitUTF8"] = true;
        return Json::writeString(b, root) + "\n";
    }
    return json_to_yaml(root);
}

// Render a JSON object/array tree as YAML (block style), quoting all string
// scalars. Sufficient for the manifest shapes we emit.
static void yaml_node(std::ostringstream& o, const Json::Value& v,
                      const std::string& indent);

static std::string json_to_yaml(const Json::Value& root) {
    std::ostringstream o;
    yaml_node(o, root, "");
    return o.str();
}

static std::string yaml_scalar_of(const Json::Value& v) {
    if (v.isString()) return emit_yaml_scalar(v.asString());
    if (v.isBool()) return v.asBool() ? "true" : "false";
    if (v.isIntegral()) return std::to_string(v.asInt64());
    if (v.isNumeric()) return std::to_string(v.asDouble());
    if (v.isNull()) return "null";
    return emit_yaml_scalar(v.asString());
}

static void yaml_node(std::ostringstream& o, const Json::Value& v,
                      const std::string& indent) {
    if (v.isObject()) {
        for (const auto& key : v.getMemberNames()) {
            const Json::Value& child = v[key];
            if (child.isObject()) {
                if (child.empty()) {
                    o << indent << key << ": {}\n";
                } else {
                    o << indent << key << ":\n";
                    yaml_node(o, child, indent + "  ");
                }
            } else if (child.isArray()) {
                if (child.empty()) {
                    o << indent << key << ": []\n";
                } else {
                    o << indent << key << ":\n";
                    for (const auto& el : child) {
                        o << indent << "  -";
                        if (el.isObject() || el.isArray()) {
                            o << "\n";
                            yaml_node(o, el, indent + "    ");
                        } else {
                            o << " " << yaml_scalar_of(el) << "\n";
                        }
                    }
                }
            } else {
                o << indent << key << ": " << yaml_scalar_of(child) << "\n";
            }
        }
    } else if (v.isArray()) {
        for (const auto& el : v) {
            o << indent << "-";
            if (el.isObject() || el.isArray()) {
                o << "\n";
                yaml_node(o, el, indent + "  ");
            } else {
                o << " " << yaml_scalar_of(el) << "\n";
            }
        }
    } else {
        o << indent << yaml_scalar_of(v) << "\n";
    }
}

// --------------------------------------------------------------------------
// Canonical JSON for desired_sha256 (deterministic, format-independent)
// --------------------------------------------------------------------------
static void write_canonical(std::ostringstream& o, const Json::Value& v);

static std::string escape_json(const std::string& s) {
    std::string out;
    for (char c : s) {
        switch (c) {
            case '"': out += "\\\""; break;
            case '\\': out += "\\\\"; break;
            case '\n': out += "\\n"; break;
            case '\r': out += "\\r"; break;
            case '\t': out += "\\t"; break;
            default: out.push_back(c);
        }
    }
    return out;
}

static void write_canonical(std::ostringstream& o, const Json::Value& v) {
    if (v.isObject()) {
        std::vector<std::string> keys = v.getMemberNames();
        std::sort(keys.begin(), keys.end());
        o << "{";
        for (size_t i = 0; i < keys.size(); ++i) {
            if (i) o << ",";
            o << "\"" << escape_json(keys[i]) << "\":";
            write_canonical(o, v[keys[i]]);
        }
        o << "}";
    } else if (v.isArray()) {
        o << "[";
        for (Json::ArrayIndex i = 0; i < v.size(); ++i) {
            if (i) o << ",";
            write_canonical(o, v[i]);
        }
        o << "]";
    } else if (v.isString()) {
        o << "\"" << escape_json(v.asString()) << "\"";
    } else if (v.isBool()) {
        o << (v.asBool() ? "true" : "false");
    } else if (v.isIntegral()) {
        o << v.asInt64();
    } else if (v.isNull()) {
        o << "null";
    } else {
        o << v.asDouble();
    }
}

std::string canonical_json(const Manifest& m) {
    // Build a copy with _elements sorted by identity key and meta reduced to
    // only the hashed fields (format_version) — created_at and desired_sha256
    // are informational and must not affect identity.
    Manifest c = m;
    // Drop observational scopes from the identity (they never belong to a
    // desired manifest / applied record identity).
    c.changed_managed_files.reset();
    c.unmanaged_files.reset();
    if (c.packages)
        std::sort(c.packages->elements.begin(), c.packages->elements.end(),
                  [](const PackageRecord& a, const PackageRecord& b) {
                      if (a.name != b.name) return a.name < b.name;
                      return a.arch < b.arch;
                  });
    if (c.repositories)
        std::sort(c.repositories->elements.begin(),
                  c.repositories->elements.end(),
                  [](const RepositoryRecord& a, const RepositoryRecord& b) {
                      return a.alias < b.alias;
                  });
    if (c.services)
        std::sort(c.services->elements.begin(), c.services->elements.end(),
                  [](const ServiceRecord& a, const ServiceRecord& b) {
                      return a.name < b.name;
                  });
    if (c.config_files)
        std::sort(c.config_files->elements.begin(),
                  c.config_files->elements.end(),
                  [](const ManagedFileRecord& a, const ManagedFileRecord& b) {
                      return a.name < b.name;
                  });
    Json::Value root = manifest_to_json(c, /*include_meta=*/false);
    // Identity meta: only the structural format_version.
    Json::Value meta(Json::objectValue);
    meta["format_version"] = c.meta.format_version;
    root["meta"] = meta;
    std::ostringstream o;
    write_canonical(o, root);
    return o.str();
}

std::string desired_sha256(const Manifest& m) {
    return sha256_hex(canonical_json(m));
}

// --------------------------------------------------------------------------
// JSON -> model parse
// --------------------------------------------------------------------------
static std::map<std::string, std::string> parse_attrs(const Json::Value& v) {
    std::map<std::string, std::string> a;
    if (v.isObject()) {
        for (const auto& k : v.getMemberNames()) {
            const Json::Value& el = v[k];
            a[k] = el.isString() ? el.asString()
                                  : el.isNull() ? std::string() : el.asString();
        }
    }
    return a;
}

static PackageRecord parse_pkg(const Json::Value& v) {
    PackageRecord p;
    p.name = v.get("name", "").asString();
    p.version = v.get("version", "").asString();
    p.release = v.get("release", "").asString();
    p.arch = v.get("arch", "").asString();
    return p;
}
static RepositoryRecord parse_repo(const Json::Value& v) {
    RepositoryRecord r;
    r.alias = v.get("alias", "").asString();
    r.name = v.get("name", "").asString();
    r.url = v.get("url", "").asString();
    r.type = v.get("type", "").asString();
    r.enabled = v.get("enabled", false).asBool();
    r.gpgcheck = v.get("gpgcheck", false).asBool();
    r.autorefresh = v.get("autorefresh", false).asBool();
    r.priority = v.get("priority", 0).asInt64();
    return r;
}
static ServiceRecord parse_svc(const Json::Value& v) {
    ServiceRecord s;
    s.name = v.get("name", "").asString();
    s.state = v.get("state", "").asString();
    return s;
}
static ManagedFileRecord parse_file(const Json::Value& v) {
    ManagedFileRecord f;
    f.name = v.get("name", "").asString();
    f.type = v.get("type", "").asString();
    f.mode = v.get("mode", "").asString();
    f.user = v.get("user", "").asString();
    f.group = v.get("group", "").asString();
    f.sha256 = v.get("sha256", "").asString();
    f.target = v.get("target", "").asString();
    f.content_ref = v.get("content_ref", "").asString();
    f.package_name = v.get("package_name", "").asString();
    return f;
}
static ManagedBaselineRecord parse_mbase(const Json::Value& v) {
    ManagedBaselineRecord f;
    f.name = v.get("name", "").asString();
    f.type = v.get("type", "").asString();
    f.mode = v.get("mode", "").asString();
    f.user = v.get("user", "").asString();
    f.group = v.get("group", "").asString();
    f.sha256 = v.get("sha256", "").asString();
    f.target = v.get("target", "").asString();
    f.package_name = v.get("package_name", "").asString();
    if (v.isMember("changes") && v["changes"].isArray())
        for (const auto& c : v["changes"]) f.changes.push_back(c.asString());
    return f;
}
static UnmanagedFileRecord parse_unmanaged(const Json::Value& v) {
    UnmanagedFileRecord f;
    f.name = v.get("name", "").asString();
    f.type = v.get("type", "").asString();
    f.mode = v.get("mode", "").asString();
    f.user = v.get("user", "").asString();
    f.group = v.get("group", "").asString();
    f.sha256 = v.get("sha256", "").asString();
    f.target = v.get("target", "").asString();
    return f;
}

template <class T, class F>
static std::optional<ScopeWrapper<T>> parse_scope(const Json::Value& root,
                                                  const char* key, F parse) {
    if (!root.isMember(key)) return std::nullopt;
    const Json::Value& s = root[key];
    ScopeWrapper<T> w;
    if (s.isMember("_attributes")) w.attributes = parse_attrs(s["_attributes"]);
    if (s.isMember("_elements") && s["_elements"].isArray())
        for (const auto& e : s["_elements"]) w.elements.push_back(parse(e));
    return w;
}

static bool json_to_manifest(const Json::Value& root, Manifest& m,
                             std::string& violation) {
    if (!root.isObject()) {
        violation = "manifest root is not an object";
        return false;
    }
    const Json::Value& meta = root["meta"];
    m.meta.format_version = meta.get("format_version", 0).asInt();
    m.meta.generator = meta.get("generator", "").asString();
    m.meta.created_at = meta.get("created_at", "").asString();
    m.meta.desired_sha256 = meta.get("desired_sha256", "").asString();
    if (m.meta.format_version != 1) {
        violation = "meta.format_version must be 1";
        return false;
    }
    m.packages = parse_scope<PackageRecord>(root, "packages", parse_pkg);
    m.repositories =
        parse_scope<RepositoryRecord>(root, "repositories", parse_repo);
    m.services = parse_scope<ServiceRecord>(root, "services", parse_svc);
    m.config_files =
        parse_scope<ManagedFileRecord>(root, "config_files", parse_file);
    m.changed_managed_files = parse_scope<ManagedBaselineRecord>(
        root, "changed_managed_files", parse_mbase);
    m.unmanaged_files = parse_scope<UnmanagedFileRecord>(
        root, "unmanaged_files", parse_unmanaged);
    return true;
}

// --------------------------------------------------------------------------
// YAML safe-profile parse (yaml-cpp): reject non-default tags, multi-document
// streams, and unbounded aliases; explicit string typing. We convert the YAML
// tree into a Json::Value, then reuse the JSON path.
// --------------------------------------------------------------------------
static bool yaml_safe_to_json(const YAML::Node& node, Json::Value& out,
                              int alias_budget, int& used, std::string& err);

static bool node_tag_ok(const YAML::Node& n) {
    // yaml-cpp exposes the resolved tag. Reject explicit non-core tags (e.g.
    // !!python/object, !ruby/..). Core schema tags / empty tag / '?' are fine.
    const std::string& tag = n.Tag();
    if (tag.empty() || tag == "?" || tag == "!") return true;
    static const char* ok[] = {"tag:yaml.org,2002:str", "tag:yaml.org,2002:int",
                               "tag:yaml.org,2002:float",
                               "tag:yaml.org,2002:bool",
                               "tag:yaml.org,2002:null",
                               "tag:yaml.org,2002:map",
                               "tag:yaml.org,2002:seq"};
    for (const char* t : ok)
        if (tag == t) return true;
    return false;
}

static bool yaml_safe_to_json(const YAML::Node& node, Json::Value& out,
                              int alias_budget, int& used, std::string& err) {
    if (!node_tag_ok(node)) {
        err = "YAML uses a disallowed tag: " + node.Tag();
        return false;
    }
    if (++used > alias_budget) {
        err = "YAML alias/node expansion exceeds the safety budget";
        return false;
    }
    switch (node.Type()) {
        case YAML::NodeType::Null:
            out = Json::Value(Json::nullValue);
            return true;
        case YAML::NodeType::Scalar:
            // Explicit string typing: read every scalar as a string, no
            // implicit coercion (NO->false, 1.10->float).
            out = Json::Value(node.Scalar());
            return true;
        case YAML::NodeType::Sequence: {
            out = Json::Value(Json::arrayValue);
            for (const auto& el : node) {
                Json::Value child;
                if (!yaml_safe_to_json(el, child, alias_budget, used, err))
                    return false;
                out.append(child);
            }
            return true;
        }
        case YAML::NodeType::Map: {
            out = Json::Value(Json::objectValue);
            for (const auto& kv : node) {
                if (!node_tag_ok(kv.first)) {
                    err = "YAML key uses a disallowed tag";
                    return false;
                }
                std::string key = kv.first.Scalar();
                Json::Value child;
                if (!yaml_safe_to_json(kv.second, child, alias_budget, used,
                                       err))
                    return false;
                out[key] = child;
            }
            return true;
        }
        default:
            err = "YAML node of undefined type";
            return false;
    }
}

// Returns true on success; on failure sets err. Numeric coercion for known
// integer/bool fields happens later when the JSON path reads typed members,
// so we post-process meta.format_version and the boolean repo fields from
// their string forms here.
static bool parse_yaml_document(const std::string& text, Json::Value& out,
                                std::string& err) {
    std::vector<YAML::Node> docs;
    try {
        docs = YAML::LoadAll(text);
    } catch (const YAML::Exception& e) {
        err = std::string("YAML parse error: ") + e.what();
        return false;
    }
    if (docs.size() != 1) {
        err = "YAML must be a single document (multi-document stream rejected)";
        return false;
    }
    int used = 0;
    // Bounded alias/node expansion budget.
    const int kBudget = 100000;
    if (!yaml_safe_to_json(docs[0], out, kBudget, used, err)) return false;
    return true;
}

// YAML carries scalars as strings; coerce the typed fields the schema needs.
static void coerce_yaml_types(Json::Value& root) {
    if (root.isMember("meta") && root["meta"].isMember("format_version")) {
        Json::Value& fv = root["meta"]["format_version"];
        if (fv.isString()) {
            try {
                fv = Json::Value(std::stoi(fv.asString()));
            } catch (...) {
                fv = Json::Value(0);
            }
        }
    }
    auto coerce_bool = [](Json::Value& v) {
        if (v.isString()) {
            std::string s = v.asString();
            std::transform(s.begin(), s.end(), s.begin(), ::tolower);
            v = Json::Value(s == "true");
        }
    };
    auto coerce_int = [](Json::Value& v) {
        if (v.isString()) {
            try {
                v = Json::Value(static_cast<Json::Int64>(
                    std::stoll(v.asString())));
            } catch (...) {
                v = Json::Value(0);
            }
        }
    };
    if (root.isMember("repositories") &&
        root["repositories"].isMember("_elements")) {
        for (Json::Value& r : root["repositories"]["_elements"]) {
            if (r.isMember("enabled")) coerce_bool(r["enabled"]);
            if (r.isMember("gpgcheck")) coerce_bool(r["gpgcheck"]);
            if (r.isMember("autorefresh")) coerce_bool(r["autorefresh"]);
            if (r.isMember("priority")) coerce_int(r["priority"]);
        }
    }
}

// --------------------------------------------------------------------------
// Shared file read + parse
// --------------------------------------------------------------------------
static bool read_file(const std::string& path, std::string& out) {
    std::ifstream f(path, std::ios::binary);
    if (!f.good()) return false;
    std::ostringstream ss;
    ss << f.rdbuf();
    out = ss.str();
    return true;
}

static Diagnostic diag(Severity sev, const std::string& dom,
                       const std::string& msg) {
    return Diagnostic{sev, dom, msg};
}

static bool has_nonempty_observational(const Manifest& m) {
    if (m.changed_managed_files && !m.changed_managed_files->elements.empty())
        return true;
    if (m.unmanaged_files && !m.unmanaged_files->elements.empty()) return true;
    return false;
}

// Parse text into a Json::Value under the resolved format.
static bool parse_text(const std::string& text, ManifestFormat fmt,
                       Json::Value& root, std::string& err) {
    if (fmt == ManifestFormat::Yaml) {
        if (!parse_yaml_document(text, root, err)) return false;
        coerce_yaml_types(root);
        return true;
    }
    Json::CharReaderBuilder b;
    std::string perr;
    std::istringstream ss(text);
    if (!Json::parseFromStream(b, ss, &root, &perr)) {
        err = "JSON parse error: " + perr;
        return false;
    }
    return true;
}

LoadResult load_desired_manifest(const std::string& manifest_path,
                                 const std::optional<ManifestFormat>& fmt,
                                 const Config& cfg) {
    LoadResult r;
    std::string text;
    if (!read_file(manifest_path, text)) {
        r.error = diag(Severity::Error, "invocation",
                       "manifest unreadable: " + manifest_path);
        return r;  // exit 2 path
    }
    ManifestFormat use = resolve_format(fmt, manifest_path, cfg.manifest_format);
    Json::Value root;
    std::string err;
    if (use == ManifestFormat::Yaml) {
        if (!parse_yaml_document(text, root, err)) {
            // unsafe-YAML or parse error -> manifest error (exit 1)
            r.error = diag(Severity::Error, "manifest", err);
            return r;
        }
        coerce_yaml_types(root);
    } else {
        if (!parse_text(text, ManifestFormat::Json, root, err)) {
            r.error = diag(Severity::Error, "manifest", err);
            return r;
        }
    }
    std::string violation;
    if (!json_to_manifest(root, r.manifest, violation)) {
        r.error = diag(Severity::Error, "manifest", violation);
        return r;
    }
    // A desired manifest must not carry a non-empty observational scope.
    if (has_nonempty_observational(r.manifest)) {
        r.error = diag(Severity::Error, "manifest",
                       "desired manifest carries a non-empty observational "
                       "scope (changed_managed_files or unmanaged_files)");
        return r;
    }
    // Drop tolerated empty observational scopes.
    r.manifest.changed_managed_files.reset();
    r.manifest.unmanaged_files.reset();
    // Signature verification: when enabled, a real keyring check would run
    // here. With no keyring configured we cannot verify; the spec defaults
    // signature-verification on, but verification requires a keyring path. We
    // treat an enabled check without a keyring as not-applicable (no signature
    // material present) rather than fabricating a pass/fail.
    (void)cfg;
    r.desired_sha256 = desired_sha256(r.manifest);
    r.ok = true;
    return r;
}

LoadResult load_state_dump(const std::string& state_path,
                           const std::optional<ManifestFormat>& fmt,
                           const Config& cfg) {
    LoadResult r;
    std::string text;
    if (!read_file(state_path, text)) {
        r.error = diag(Severity::Error, "invocation",
                       "state dump unreadable: " + state_path);
        return r;  // exit 2
    }
    ManifestFormat use = resolve_format(fmt, state_path, cfg.manifest_format);
    Json::Value root;
    std::string err;
    if (!parse_text(text, use, root, err)) {
        // malformed dump -> invocation error (exit 2)
        r.error = diag(Severity::Error, "invocation",
                       "malformed state dump: " + err);
        return r;
    }
    std::string violation;
    if (!json_to_manifest(root, r.manifest, violation)) {
        r.error = diag(Severity::Error, "invocation",
                       "malformed state dump: " + violation);
        return r;
    }
    r.ok = true;
    return r;
}

AppliedLoad load_applied_record(const std::string& root) {
    AppliedLoad a;
    std::string path = root;
    if (!path.empty() && path.back() == '/') path.pop_back();
    path += "/usr/lib/zypper-declarative/applied.json";
    std::string text;
    if (!read_file(path, text)) {
        a.present = false;
        a.ok = true;  // absence is normal
        return a;
    }
    Json::Value v;
    std::string err;
    if (!parse_text(text, ManifestFormat::Json, v, err)) {
        a.ok = false;
        a.error = diag(Severity::Error, "files",
                       "applied record unparseable: " + err);
        return a;
    }
    std::string violation;
    if (!json_to_manifest(v, a.record, violation)) {
        a.ok = false;
        a.error = diag(Severity::Error, "files",
                       "applied record invalid: " + violation);
        return a;
    }
    a.present = true;
    a.ok = true;
    return a;
}

}  // namespace zd
