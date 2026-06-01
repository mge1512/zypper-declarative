// generated from spec: zypper-declarative.spec.md sha256:f2cc80627e483a48bb8411d297711bc5f6c6e74c28dbf0dafc8fe7bd8817251e
// tests by: claude-opus-4-8
//
// Fixture helpers: write temp manifest files and build synthetic root trees
// for describe tests. Black-box only; no implementation internals.
#ifndef ZD_TEST_FIXTURES_HPP
#define ZD_TEST_FIXTURES_HPP

#include <string>
#include <fstream>
#include <filesystem>
#include <cstdlib>
#include <unistd.h>
#include <sys/stat.h>

namespace zdtest {
namespace fs = std::filesystem;

// Create a unique temporary directory under the system temp dir.
inline fs::path make_tmpdir(const std::string& prefix) {
    fs::path base = fs::temp_directory_path() /
        (prefix + "-" + std::to_string(::getpid()) + "-" +
         std::to_string(reinterpret_cast<uintptr_t>(&prefix) & 0xffffff) + "-" +
         std::to_string(std::rand()));
    std::error_code ec;
    fs::create_directories(base, ec);
    return base;
}

// Write `content` to a file named under tmp, returning its absolute path.
inline std::string write_temp_file(const std::string& name, const std::string& content) {
    static int counter = 0;
    fs::path dir = fs::temp_directory_path() / ("zd-fix-" + std::to_string(::getpid()) +
                                                "-" + std::to_string(counter++));
    std::error_code ec;
    fs::create_directories(dir, ec);
    fs::path p = dir / name;
    std::ofstream f(p);
    f << content;
    f.close();
    return p.string();
}

// A complete, valid desired manifest (JSON) the tests can vary from.
// Declarable scopes only; meta.format_version = 1.
inline std::string valid_desired_json() {
    return R"JSON({
  "meta": {
    "format_version": 1,
    "generator": "test",
    "created_at": "2026-05-29T08:30:00Z",
    "desired_sha256": ""
  },
  "packages": {
    "_attributes": { "package_system": "rpm" },
    "_elements": [
      { "name": "nginx", "version": "", "release": "", "arch": "" }
    ]
  },
  "services": {
    "_attributes": { "init_system": "systemd" },
    "_elements": [
      { "name": "nginx.service", "state": "enabled" }
    ]
  },
  "config_files": {
    "_attributes": null,
    "_elements": [
      {
        "name": "/etc/foo.conf",
        "type": "file",
        "mode": "0644",
        "user": "root",
        "group": "root",
        "sha256": "1111111111111111111111111111111111111111111111111111111111111111",
        "target": "",
        "content_ref": "files/etc/foo.conf",
        "package_name": ""
      }
    ]
  }
})JSON";
}

} // namespace zdtest

#endif // ZD_TEST_FIXTURES_HPP
