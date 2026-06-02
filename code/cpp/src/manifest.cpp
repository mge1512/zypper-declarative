// generated from spec: zypper-declarative.spec.md sha256:aafbb3158415b5c82fe459a26d0d21cbd39a077f689d5fdfb998bf5f947350a3
#include "manifest.hpp"

#include <openssl/evp.h>

#include <json/json.h>
#include <yaml-cpp/yaml.h>

#include <algorithm>
#include <cstdio>
#include <fstream>
#include <sstream>

#include "meta.hpp"

namespace zd {

namespace {

// ----------------------------------------------------------------------
// resolve-format helpers
// ----------------------------------------------------------------------
std::string lower_ext(const std::string& path) {
    auto dot = path.find_last_of('.');
    if (dot == std::string::npos) return "";
    std::string ext = path.substr(dot + 1);
    std::transform(ext.begin(), ext.end(), ext.begin(),
                   [](unsigned char c) { return std::tolower(c); });
    return ext;
}

// ----------------------------------------------------------------------
// JSON <-> data model (jsoncpp)
// ----------------------------------------------------------------------
Json::Value attrs_to_json(const std::map<std::string, std::string>& a) {
    Json::Value o(Json::objectValue);  // ALWAYS an object, never null
    for (const auto& kv : a) o[kv.first] = kv.second;
    return o;
}

Json::Value pkg_to_json(const PackageRecord& r) {
    Json::Value v(Json::objectValue);
    v["name"] = r.name; v["version"] = r.version;
    v["release"] = r.release; v["arch"] = r.arch;
    return v;
}
Json::Value repo_to_json(const RepositoryRecord& r) {
    Json::Value v(Json::objectValue);
    v["alias"] = r.alias; v["name"] = r.name; v["url"] = r.url; v["type"] = r.type;
    v["enabled"] = r.enabled; v["gpgcheck"] = r.gpgcheck;
    v["autorefresh"] = r.autorefresh;
    v["priority"] = static_cast<Json::Int64>(r.priority);
    return v;
}
Json::Value svc_to_json(const ServiceRecord& r) {
    Json::Value v(Json::objectValue);
    v["name"] = r.name; v["state"] = r.state;
    return v;
}
Json::Value file_to_json(const ManagedFileRecord& r) {
    Json::Value v(Json::objectValue);
    v["name"] = r.name; v["type"] = r.type; v["mode"] = r.mode;
    v["user"] = r.user; v["group"] = r.group; v["sha256"] = r.sha256;
    v["target"] = r.target; v["content_ref"] = r.content_ref;
    v["package_name"] = r.package_name;
    return v;
}
Json::Value mbase_to_json(const ManagedBaselineRecord& r) {
    Json::Value v(Json::objectValue);
    v["name"] = r.name; v["type"] = r.type; v["mode"] = r.mode;
    v["user"] = r.user; v["group"] = r.group; v["sha256"] = r.sha256;
    v["target"] = r.target; v["package_name"] = r.package_name;
    Json::Value ch(Json::arrayValue);
    for (const auto& c : r.changes) ch.append(c);
    v["changes"] = ch;
    return v;
}
Json::Value unmanaged_to_json(const UnmanagedFileRecord& r) {
    Json::Value v(Json::objectValue);
    v["name"] = r.name; v["type"] = r.type; v["mode"] = r.mode;
    v["user"] = r.user; v["group"] = r.group; v["sha256"] = r.sha256;
    v["target"] = r.target;
    return v;
}

template <class T, class F>
Json::Value scope_to_json(const ScopeWrapper<T>& s, F to_json) {
    Json::Value v(Json::objectValue);
    v["_attributes"] = attrs_to_json(s.attributes);
    Json::Value els(Json::arrayValue);
    for (const auto& e : s.elements) els.append(to_json(e));
    v["_elements"] = els;
    return v;
}

Json::Value manifest_to_json(const Manifest& m) {
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
            scope_to_json(*m.changed_managed_files, mbase_to_json);
    if (m.unmanaged_files)
        root["unmanaged_files"] = scope_to_json(*m.unmanaged_files, unmanaged_to_json);
    return root;
}

// Canonical JSON: _elements sorted by identity key, attributes already sorted
// (std::map), compact separators. Produces a stable byte stream for hashing.
Manifest sort_for_canonical(Manifest m) {
    if (m.packages)
        std::sort(m.packages->elements.begin(), m.packages->elements.end(),
                  [](const PackageRecord& a, const PackageRecord& b) {
                      if (a.name != b.name) return a.name < b.name;
                      return a.arch < b.arch;
                  });
    if (m.repositories)
        std::sort(m.repositories->elements.begin(), m.repositories->elements.end(),
                  [](const RepositoryRecord& a, const RepositoryRecord& b) {
                      return a.alias < b.alias;
                  });
    if (m.services)
        std::sort(m.services->elements.begin(), m.services->elements.end(),
                  [](const ServiceRecord& a, const ServiceRecord& b) {
                      return a.name < b.name;
                  });
    if (m.config_files)
        std::sort(m.config_files->elements.begin(), m.config_files->elements.end(),
                  [](const ManagedFileRecord& a, const ManagedFileRecord& b) {
                      return a.name < b.name;
                  });
    return m;
}

std::string json_to_string(const Json::Value& v, bool pretty) {
    Json::StreamWriterBuilder b;
    if (pretty) {
        b["indentation"] = "  ";
    } else {
        b["indentation"] = "";
    }
    b["commentStyle"] = "None";
    b["sortKeys"] = true;  // stable key order for canonicalisation
    return Json::writeString(b, v);
}

// ----------------------------------------------------------------------
// JSON parsing -> data model
// ----------------------------------------------------------------------
std::map<std::string, std::string> json_attrs(const Json::Value& v) {
    std::map<std::string, std::string> a;
    if (v.isObject()) {
        for (const auto& k : v.getMemberNames()) {
            const Json::Value& e = v[k];
            if (e.isString()) a[k] = e.asString();
            else a[k] = e.asString();
        }
    }
    return a;
}

PackageRecord json_pkg(const Json::Value& v) {
    PackageRecord r;
    r.name = v.get("name", "").asString();
    r.version = v.get("version", "").asString();
    r.release = v.get("release", "").asString();
    r.arch = v.get("arch", "").asString();
    return r;
}
RepositoryRecord json_repo(const Json::Value& v) {
    RepositoryRecord r;
    r.alias = v.get("alias", "").asString();
    r.name = v.get("name", "").asString();
    r.url = v.get("url", "").asString();
    r.type = v.get("type", "").asString();
    r.enabled = v.get("enabled", true).asBool();
    r.gpgcheck = v.get("gpgcheck", true).asBool();
    r.autorefresh = v.get("autorefresh", false).asBool();
    r.priority = v.get("priority", 99).asInt64();
    return r;
}
ServiceRecord json_svc(const Json::Value& v) {
    ServiceRecord r;
    r.name = v.get("name", "").asString();
    r.state = v.get("state", "").asString();
    return r;
}
ManagedFileRecord json_file(const Json::Value& v) {
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
ManagedBaselineRecord json_mbase(const Json::Value& v) {
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
UnmanagedFileRecord json_unmanaged(const Json::Value& v) {
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

template <class T, class F>
ScopeWrapper<T> json_scope(const Json::Value& v, F from_json) {
    ScopeWrapper<T> s;
    if (v.isMember("_attributes")) s.attributes = json_attrs(v["_attributes"]);
    if (v.isMember("_elements") && v["_elements"].isArray())
        for (const auto& e : v["_elements"]) s.elements.push_back(from_json(e));
    return s;
}

Manifest json_to_manifest(const Json::Value& root) {
    Manifest m;
    if (root.isMember("meta")) {
        const Json::Value& meta = root["meta"];
        m.meta.format_version = meta.get("format_version", 1).asInt();
        m.meta.generator = meta.get("generator", "").asString();
        m.meta.created_at = meta.get("created_at", "").asString();
        m.meta.desired_sha256 = meta.get("desired_sha256", "").asString();
    }
    if (root.isMember("packages"))
        m.packages = json_scope<PackageRecord>(root["packages"], json_pkg);
    if (root.isMember("repositories"))
        m.repositories = json_scope<RepositoryRecord>(root["repositories"], json_repo);
    if (root.isMember("services"))
        m.services = json_scope<ServiceRecord>(root["services"], json_svc);
    if (root.isMember("config_files"))
        m.config_files = json_scope<ManagedFileRecord>(root["config_files"], json_file);
    if (root.isMember("changed_managed_files"))
        m.changed_managed_files =
            json_scope<ManagedBaselineRecord>(root["changed_managed_files"], json_mbase);
    if (root.isMember("unmanaged_files"))
        m.unmanaged_files =
            json_scope<UnmanagedFileRecord>(root["unmanaged_files"], json_unmanaged);
    return m;
}

// ----------------------------------------------------------------------
// YAML safe-profile parsing -> JSON value
//
// Safe profile (load-desired-manifest STEP 3): non-executing loader (yaml-cpp's
// Node API never executes), reject non-default tags, single document only,
// bounded alias expansion, explicit typing. We convert the YAML tree to a
// Json::Value with explicit string typing so YAML implicit coercion (NO->false,
// 1.10->float) does not apply: every scalar is read as a string and the JSON
// layer interprets it per the schema.
// ----------------------------------------------------------------------
struct YamlUnsafe {
    std::string reason;
};

void check_node_tags(const YAML::Node& n, int depth) {
    if (depth > 100) throw YamlUnsafe{"alias/recursion depth exceeded"};
    const std::string& tag = n.Tag();
    // Default/implicit tags only: "?", "!", or the canonical core-schema tags.
    if (!(tag.empty() || tag == "?" || tag == "!" ||
          tag.rfind("tag:yaml.org,2002:", 0) == 0)) {
        throw YamlUnsafe{"non-default YAML tag '" + tag + "' is not permitted"};
    }
    if (n.IsSequence())
        for (const auto& c : n) check_node_tags(c, depth + 1);
    else if (n.IsMap())
        for (const auto& kv : n) check_node_tags(kv.second, depth + 1);
}

Json::Value yaml_scalar_to_json(const YAML::Node& n, const std::string& key) {
    // Explicit typing per schema: booleans/integers for the fields the schema
    // types as such; everything else as a string.
    std::string s = n.as<std::string>();
    if (key == "format_version" || key == "priority") {
        try { return Json::Value(static_cast<Json::Int64>(std::stoll(s))); }
        catch (...) { return Json::Value(s); }
    }
    if (key == "enabled" || key == "gpgcheck" || key == "autorefresh") {
        if (s == "true") return Json::Value(true);
        if (s == "false") return Json::Value(false);
        return Json::Value(s);
    }
    return Json::Value(s);
}

Json::Value yaml_to_json(const YAML::Node& n, const std::string& key, int depth) {
    if (depth > 100) throw YamlUnsafe{"alias/recursion depth exceeded"};
    if (n.IsScalar()) return yaml_scalar_to_json(n, key);
    if (n.IsSequence()) {
        Json::Value a(Json::arrayValue);
        for (const auto& c : n) a.append(yaml_to_json(c, key, depth + 1));
        return a;
    }
    if (n.IsMap()) {
        Json::Value o(Json::objectValue);
        for (const auto& kv : n) {
            std::string k = kv.first.as<std::string>();
            o[k] = yaml_to_json(kv.second, k, depth + 1);
        }
        return o;
    }
    if (n.IsNull()) return Json::Value(Json::nullValue);
    return Json::Value(Json::nullValue);
}

// Parse text to Json::Value, format-specific. Throws YamlUnsafe / sets errmsg.
bool parse_to_json(const std::string& text, ManifestFormat fmt,
                   Json::Value& out, std::string& errmsg) {
    if (fmt == ManifestFormat::Json) {
        Json::CharReaderBuilder b;
        std::string errs;
        std::istringstream is(text);
        if (!Json::parseFromStream(b, is, &out, &errs)) {
            errmsg = errs;
            return false;
        }
        return true;
    }
    // YAML safe profile
    try {
        std::vector<YAML::Node> docs = YAML::LoadAll(text);
        if (docs.empty()) { errmsg = "empty YAML document"; return false; }
        if (docs.size() > 1) {
            errmsg = "multi-document YAML streams are not permitted";
            return false;
        }
        const YAML::Node& root = docs.front();
        check_node_tags(root, 0);
        out = yaml_to_json(root, "", 0);
        return true;
    } catch (const YamlUnsafe& u) {
        errmsg = std::string("unsafe YAML: ") + u.reason;
        return false;
    } catch (const std::exception& e) {
        errmsg = std::string("YAML parse error: ") + e.what();
        return false;
    }
}

bool read_file(const std::string& path, std::string& out) {
    std::ifstream f(path, std::ios::binary);
    if (!f) return false;
    std::ostringstream ss;
    ss << f.rdbuf();
    out = ss.str();
    return true;
}

// schema-validate: format_version must be 1; declarable scopes conform; an
// observational scope present with non-empty _elements is rejected.
std::optional<Diagnostic> validate_desired(const Manifest& m) {
    if (m.meta.format_version != 1)
        return err("manifest", "meta.format_version must be 1");
    if (m.changed_managed_files && !m.changed_managed_files->elements.empty())
        return err("manifest",
                   "desired manifest must not carry a non-empty changed_managed_files scope");
    if (m.unmanaged_files && !m.unmanaged_files->elements.empty())
        return err("manifest",
                   "desired manifest must not carry a non-empty unmanaged_files scope");
    return std::nullopt;
}

}  // namespace

// ----------------------------------------------------------------------
// Public API
// ----------------------------------------------------------------------

std::optional<ManifestFormat> parse_format(const std::string& s) {
    if (s == "json") return ManifestFormat::Json;
    if (s == "yaml") return ManifestFormat::Yaml;
    return std::nullopt;
}

ManifestFormat resolve_format(const std::optional<ManifestFormat>& explicit_fmt,
                              const std::optional<std::string>& path,
                              ManifestFormat default_fmt) {
    if (explicit_fmt) return *explicit_fmt;  // explicit always wins
    if (path) {
        std::string ext = lower_ext(*path);
        if (ext == "json") return ManifestFormat::Json;
        if (ext == "yaml" || ext == "yml") return ManifestFormat::Yaml;
    }
    return default_fmt;
}

std::string sha256_hex(const std::string& bytes) {
    unsigned char digest[EVP_MAX_MD_SIZE];
    unsigned int len = 0;
    EVP_MD_CTX* ctx = EVP_MD_CTX_new();
    EVP_DigestInit_ex(ctx, EVP_sha256(), nullptr);
    EVP_DigestUpdate(ctx, bytes.data(), bytes.size());
    EVP_DigestFinal_ex(ctx, digest, &len);
    EVP_MD_CTX_free(ctx);
    static const char* hexd = "0123456789abcdef";
    std::string out;
    out.reserve(len * 2);
    for (unsigned int i = 0; i < len; ++i) {
        out.push_back(hexd[digest[i] >> 4]);
        out.push_back(hexd[digest[i] & 0xf]);
    }
    return out;
}

std::string serialise_json(const Manifest& m, bool pretty) {
    return json_to_string(manifest_to_json(m), pretty);
}

std::string canonical_sha256(const Manifest& m) {
    // Hash is over the canonical model: meta.desired_sha256 and created_at are
    // excluded from identity (they are informational / self-referential).
    Manifest c = sort_for_canonical(m);
    c.meta.desired_sha256 = "";
    c.meta.created_at = "";
    std::string canon = json_to_string(manifest_to_json(c), /*pretty=*/false);
    return sha256_hex(canon);
}

namespace {
std::string yaml_quote(const std::string& s) {
    std::string out = "\"";
    for (char ch : s) {
        if (ch == '\\' || ch == '"') out.push_back('\\');
        out.push_back(ch);
    }
    out.push_back('"');
    return out;
}
void emit_attrs(std::ostringstream& o, const std::string& indent,
                const std::map<std::string, std::string>& a) {
    o << indent << "_attributes:";
    if (a.empty()) { o << " {}\n"; return; }
    o << "\n";
    for (const auto& kv : a)
        o << indent << "  " << kv.first << ": " << yaml_quote(kv.second) << "\n";
}
}  // namespace

std::string serialise_yaml(const Manifest& m) {
    // Hand-emitted YAML restricted to the schema, with string scalars quoted so
    // values such as mode "0600" are not coerced. Same data model as JSON.
    std::ostringstream o;
    o << "meta:\n";
    o << "  format_version: " << m.meta.format_version << "\n";
    o << "  generator: " << yaml_quote(m.meta.generator) << "\n";
    o << "  created_at: " << yaml_quote(m.meta.created_at) << "\n";
    o << "  desired_sha256: " << yaml_quote(m.meta.desired_sha256) << "\n";
    if (m.packages) {
        o << "packages:\n";
        emit_attrs(o, "  ", m.packages->attributes);
        o << "  _elements:";
        if (m.packages->elements.empty()) o << " []\n";
        else {
            o << "\n";
            for (const auto& p : m.packages->elements) {
                o << "    - name: " << yaml_quote(p.name) << "\n";
                o << "      version: " << yaml_quote(p.version) << "\n";
                o << "      release: " << yaml_quote(p.release) << "\n";
                o << "      arch: " << yaml_quote(p.arch) << "\n";
            }
        }
    }
    if (m.repositories) {
        o << "repositories:\n";
        emit_attrs(o, "  ", m.repositories->attributes);
        o << "  _elements:";
        if (m.repositories->elements.empty()) o << " []\n";
        else {
            o << "\n";
            for (const auto& r : m.repositories->elements) {
                o << "    - alias: " << yaml_quote(r.alias) << "\n";
                o << "      name: " << yaml_quote(r.name) << "\n";
                o << "      url: " << yaml_quote(r.url) << "\n";
                o << "      type: " << yaml_quote(r.type) << "\n";
                o << "      enabled: " << (r.enabled ? "true" : "false") << "\n";
                o << "      gpgcheck: " << (r.gpgcheck ? "true" : "false") << "\n";
                o << "      autorefresh: " << (r.autorefresh ? "true" : "false") << "\n";
                o << "      priority: " << r.priority << "\n";
            }
        }
    }
    if (m.services) {
        o << "services:\n";
        emit_attrs(o, "  ", m.services->attributes);
        o << "  _elements:";
        if (m.services->elements.empty()) o << " []\n";
        else {
            o << "\n";
            for (const auto& s : m.services->elements) {
                o << "    - name: " << yaml_quote(s.name) << "\n";
                o << "      state: " << yaml_quote(s.state) << "\n";
            }
        }
    }
    if (m.config_files) {
        o << "config_files:\n";
        emit_attrs(o, "  ", m.config_files->attributes);
        o << "  _elements:";
        if (m.config_files->elements.empty()) o << " []\n";
        else {
            o << "\n";
            for (const auto& f : m.config_files->elements) {
                o << "    - name: " << yaml_quote(f.name) << "\n";
                o << "      type: " << yaml_quote(f.type) << "\n";
                o << "      mode: " << yaml_quote(f.mode) << "\n";
                o << "      user: " << yaml_quote(f.user) << "\n";
                o << "      group: " << yaml_quote(f.group) << "\n";
                o << "      sha256: " << yaml_quote(f.sha256) << "\n";
                o << "      target: " << yaml_quote(f.target) << "\n";
                o << "      content_ref: " << yaml_quote(f.content_ref) << "\n";
                o << "      package_name: " << yaml_quote(f.package_name) << "\n";
            }
        }
    }
    return o.str();
}

Result<LoadedManifest> load_desired_manifest(
    const std::string& manifest_path,
    const std::optional<ManifestFormat>& explicit_fmt,
    ManifestFormat default_fmt) {
    std::string text;
    if (!read_file(manifest_path, text))
        return err("invocation", "manifest unreadable: " + manifest_path);
    ManifestFormat fmt = resolve_format(explicit_fmt,
                                        std::optional<std::string>(manifest_path),
                                        default_fmt);
    Json::Value root;
    std::string emsg;
    if (!parse_to_json(text, fmt, root, emsg)) {
        // Unsafe YAML and schema parse errors are manifest errors (exit 1).
        return err("manifest", "manifest parse failed: " + emsg);
    }
    if (!root.isObject())
        return err("manifest", "manifest root is not an object");
    Manifest m = json_to_manifest(root);
    if (auto v = validate_desired(m)) return *v;
    // Drop tolerated empty observational scopes.
    m.changed_managed_files.reset();
    m.unmanaged_files.reset();
    LoadedManifest lm;
    lm.manifest = m;
    lm.desired_sha256 = canonical_sha256(m);
    return lm;
}

Result<Manifest> load_state_dump(const std::string& state_path,
                                 const std::optional<ManifestFormat>& explicit_fmt,
                                 ManifestFormat default_fmt) {
    std::string text;
    if (!read_file(state_path, text))
        return err("invocation", "state dump unreadable: " + state_path);
    ManifestFormat fmt = resolve_format(explicit_fmt,
                                        std::optional<std::string>(state_path),
                                        default_fmt);
    Json::Value root;
    std::string emsg;
    if (!parse_to_json(text, fmt, root, emsg))
        return err("invocation", "malformed state dump: " + emsg);
    if (!root.isObject())
        return err("invocation", "malformed state dump: root is not an object");
    Manifest m = json_to_manifest(root);
    if (m.meta.format_version != 1)
        return err("invocation", "malformed state dump: format_version must be 1");
    return m;
}

Result<AppliedLoad> load_applied_record(const std::string& root) {
    std::string base = root;
    if (!base.empty() && base.back() == '/') base.pop_back();
    std::string path = base + "/usr/lib/zypper-declarative/applied.json";
    std::string text;
    if (!read_file(path, text)) {
        AppliedLoad a;  // all scopes empty, present=false
        a.present = false;
        return a;
    }
    Json::Value v;
    std::string emsg;
    if (!parse_to_json(text, ManifestFormat::Json, v, emsg))
        return err("files", "applied record unparseable: " + emsg);
    AppliedLoad a;
    a.record = json_to_manifest(v);
    a.present = true;
    return a;
}

}  // namespace zd
