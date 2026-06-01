// generated from spec: zypper-declarative.spec.md sha256:f2cc80627e483a48bb8411d297711bc5f6c6e74c28dbf0dafc8fe7bd8817251e
//
// describe.hpp -- describe-actual-state, the single live-state reader, and the
// system-integration seam it uses (SystemReader) so tests can supply a double.
#ifndef ZD_DESCRIBE_HPP
#define ZD_DESCRIBE_HPP

#include "types.hpp"
#include "config.hpp"
#include "diff.hpp"
#include <string>
#include <vector>
#include <optional>

namespace zd {

// Result of describe-actual-state.
struct ActualStateResult {
    bool ok = true;                       // false under on_unreadable=error
    Manifest manifest;
    std::vector<Diagnostic> diagnostics;  // warn diagnostics
    Diagnostic error;                     // set when ok=false
};

// The system-integration seam: package ownership and rpmdb query, unit
// enablement. The on-disk repos.d read and the /etc walk are filesystem reads
// done directly by describe-actual-state. A SystemReader supplies the parts
// that need libzypp / systemd; the production reader links those libraries, a
// test double can stand in. nullptr = no system integration available (synthetic
// root mode), in which case packages and services scopes are read as empty and
// every /etc regular file/symlink is treated as unpackaged.
class SystemReader {
public:
    virtual ~SystemReader() = default;
    // Query the installed package set under root; ok=false if rpmdb unreadable.
    virtual PackagesScope query_packages(const std::string& root, bool& ok) const = 0;
    // Query unit enablement under root; ok=false if unreadable.
    virtual ServicesScope query_services(const std::string& root, bool& ok) const = 0;
    // Owning package of a path under root ("" if unpackaged).
    virtual std::string owning_package(const std::string& root,
                                       const std::string& path) const = 0;
    // Does this regular file differ from its package baseline content? For an
    // unpackaged file returns true (it has no baseline to match).
    virtual bool file_differs_from_baseline(const std::string& root,
                                            const std::string& path,
                                            const std::string& actual_sha256) const = 0;
    // Does this symlink differ from its package-recorded target?
    virtual bool link_differs_from_baseline(const std::string& root,
                                            const std::string& path,
                                            const std::string& actual_target) const = 0;
};

// BEHAVIOR/INTERNAL: describe-actual-state
ActualStateResult describe_actual_state(const std::string& root,
                                        OnUnreadable on_unreadable,
                                        ScanScope scope,
                                        const KeepList& keep,
                                        const SystemReader* reader);

} // namespace zd

#endif // ZD_DESCRIBE_HPP
