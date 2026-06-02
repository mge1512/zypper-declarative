// generated from spec: zypper-declarative.spec.md sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
//
// Package state implements BEHAVIOR/INTERNAL: describe-actual-state, the single
// live-state reader. Every verb that needs actual state obtains it here (or from
// a supplied dump). No other package reads live system state.
package state

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mge1512/zypper-declarative/internal/manifest"
	"github.com/mge1512/zypper-declarative/internal/meta"
	"github.com/mge1512/zypper-declarative/internal/syscmd"
)

// OnUnreadable controls treatment of a source that cannot be read.
type OnUnreadable string

const (
	OnUnreadableError OnUnreadable = "error"
	OnUnreadableWarn  OnUnreadable = "warn"
)

// Scope selects the read scope.
type Scope string

const (
	ScopeEtc  Scope = "etc"
	ScopeFull Scope = "full"
)

// Diagnostic is a domain-tagged diagnostic returned by the reader.
type Diagnostic struct {
	Severity string
	Domain   string
	Message  string
}

func (d *Diagnostic) Error() string { return d.Message }

// Options configures a describe-actual-state read.
type Options struct {
	Root         string
	OnUnreadable OnUnreadable
	Scope        Scope
	ContentStore string // base path for the content store; "" = read-only
	KeepList     map[string]bool
	Runner       syscmd.CommandRunner
}

// Result is the output of Read.
type Result struct {
	Manifest    *manifest.Manifest
	Diagnostics []Diagnostic
}

// Read implements describe-actual-state. It assembles a Manifest of the four
// declarable scopes (plus the two observational scopes under scope=full).
func Read(opts Options) (*Result, error) {
	if opts.Runner == nil {
		opts.Runner = &syscmd.OSCommandRunner{}
	}
	if opts.KeepList == nil {
		opts.KeepList = map[string]bool{}
	}
	res := &Result{Manifest: &manifest.Manifest{
		Meta: manifest.ManifestMeta{FormatVersion: 1, Generator: meta.Generator()},
	}}

	warn := func(domain, msg string) {
		res.Diagnostics = append(res.Diagnostics, Diagnostic{Severity: "Warning", Domain: domain, Message: msg})
	}

	// 1. packages
	pkgScope, err := readPackages(opts)
	if err != nil {
		if opts.OnUnreadable == OnUnreadableError {
			return nil, err
		}
		warn("packages", err.Error())
	} else if pkgScope != nil && len(pkgScope.Elements) > 0 {
		res.Manifest.Packages = pkgScope
	}

	// 2. repositories
	repoScope, err := readRepositories(opts)
	if err != nil {
		if opts.OnUnreadable == OnUnreadableError {
			return nil, err
		}
		warn("repositories", err.Error())
	} else if repoScope != nil && len(repoScope.Elements) > 0 {
		res.Manifest.Repositories = repoScope
	}

	// 3. services
	svcScope, err := readServices(opts)
	if err != nil {
		if opts.OnUnreadable == OnUnreadableError {
			return nil, err
		}
		warn("units", err.Error())
	} else if svcScope != nil && len(svcScope.Elements) > 0 {
		res.Manifest.Services = svcScope
	}

	// 4. config_files (bounded to /etc)
	cfScope, cfDiags, err := readConfigFiles(opts)
	if err != nil {
		if opts.OnUnreadable == OnUnreadableError {
			return nil, err
		}
		warn("files", err.Error())
	} else {
		res.Diagnostics = append(res.Diagnostics, cfDiags...)
		if cfScope != nil && len(cfScope.Elements) > 0 {
			res.Manifest.ConfigFiles = cfScope
		}
	}

	// 4a. full-scan integrity (scope=full only)
	if opts.Scope == ScopeFull {
		cm, um, err := readFullScan(opts)
		if err != nil {
			if opts.OnUnreadable == OnUnreadableError {
				return nil, err
			}
			warn("files", err.Error())
		} else {
			if cm != nil && len(cm.Elements) > 0 {
				res.Manifest.ChangedManagedFiles = cm
			}
			if um != nil && len(um.Elements) > 0 {
				res.Manifest.UnmanagedFiles = um
			}
		}
	}

	return res, nil
}

// readPackages queries the rpmdb under root for the fully-resolved installed set.
func readPackages(opts Options) (*manifest.ScopeWrapper[manifest.PackageRecord], error) {
	args := []string{"-qa", "--queryformat", "%{NAME} %{VERSION} %{RELEASE} %{ARCH}\n"}
	args = append(args, dbpath(opts.Root)...)
	stdout, stderr, err := opts.Runner.Run("rpm", args)
	if err != nil && strings.TrimSpace(stdout) == "" {
		return nil, &Diagnostic{Severity: "Error", Domain: "packages", Message: "rpmdb unreadable: " + strings.TrimSpace(stderr)}
	}
	scope := manifest.NewScope[manifest.PackageRecord](map[string]interface{}{"package_system": "rpm"})
	for _, line := range strings.Split(stdout, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		scope.Elements = append(scope.Elements, manifest.PackageRecord{
			Name: fields[0], Version: fields[1], Release: fields[2], Arch: fields[3],
		})
	}
	return &scope, nil
}

// readRepositories reads the on-disk zypp repository configuration directly.
func readRepositories(opts Options) (*manifest.ScopeWrapper[manifest.RepositoryRecord], error) {
	dir := filepath.Join(opts.Root, "etc", "zypp", "repos.d")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			// Genuinely absent directory: an empty, readable repositories source.
			scope := manifest.NewScope[manifest.RepositoryRecord](map[string]interface{}{"repository_system": "zypp"})
			return &scope, nil
		}
		return nil, &Diagnostic{Severity: "Error", Domain: "repositories", Message: "repos.d unreadable (" + dir + "): " + err.Error()}
	}
	scope := manifest.NewScope[manifest.RepositoryRecord](map[string]interface{}{"repository_system": "zypp"})
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".repo") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, &Diagnostic{Severity: "Error", Domain: "repositories", Message: "repo file unreadable (" + e.Name() + "): " + err.Error()}
		}
		scope.Elements = append(scope.Elements, parseRepoFile(string(data))...)
	}
	return &scope, nil
}

// parseRepoFile parses INI sections into RepositoryRecords (baseurl -> url).
func parseRepoFile(content string) []manifest.RepositoryRecord {
	var recs []manifest.RepositoryRecord
	var cur *manifest.RepositoryRecord
	flush := func() {
		if cur != nil {
			recs = append(recs, *cur)
			cur = nil
		}
	}
	for _, raw := range strings.Split(content, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			flush()
			alias := strings.TrimSuffix(strings.TrimPrefix(line, "["), "]")
			cur = &manifest.RepositoryRecord{Alias: alias, Type: "rpm-md", Enabled: true, GPGCheck: true}
			continue
		}
		if cur == nil {
			continue
		}
		eq := strings.IndexByte(line, '=')
		if eq < 0 {
			continue
		}
		key := strings.TrimSpace(line[:eq])
		val := strings.TrimSpace(line[eq+1:])
		switch key {
		case "name":
			cur.Name = val
		case "baseurl":
			cur.URL = val
		case "type":
			cur.Type = val
		case "enabled":
			cur.Enabled = iniBool(val)
		case "gpgcheck":
			cur.GPGCheck = iniBool(val)
		case "autorefresh":
			cur.Autorefresh = iniBool(val)
		case "priority":
			if p, err := strconv.Atoi(val); err == nil {
				cur.Priority = p
			}
		}
	}
	flush()
	return recs
}

func iniBool(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

// readServices queries unit enablement and normalises to enabled/disabled/masked.
func readServices(opts Options) (*manifest.ScopeWrapper[manifest.ServiceRecord], error) {
	args := []string{"list-unit-files", "--no-legend", "--no-pager"}
	if opts.Root != "" && opts.Root != "/" {
		args = append([]string{"--root", opts.Root}, args...)
	}
	stdout, stderr, err := opts.Runner.Run("systemctl", args)
	if err != nil && strings.TrimSpace(stdout) == "" {
		return nil, &Diagnostic{Severity: "Error", Domain: "units", Message: "unit enablement unreadable: " + strings.TrimSpace(stderr)}
	}
	scope := manifest.NewScope[manifest.ServiceRecord](map[string]interface{}{"init_system": "systemd"})
	for _, line := range strings.Split(stdout, "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) < 2 {
			continue
		}
		name, st := fields[0], fields[1]
		if !isDeclarableUnitName(name) {
			continue
		}
		switch st {
		case "enabled", "enabled-runtime":
			scope.Elements = append(scope.Elements, manifest.ServiceRecord{Name: name, State: "enabled"})
		case "disabled":
			scope.Elements = append(scope.Elements, manifest.ServiceRecord{Name: name, State: "disabled"})
		case "masked", "masked-runtime":
			scope.Elements = append(scope.Elements, manifest.ServiceRecord{Name: name, State: "masked"})
		}
		// static, generated, indirect, etc. are observational and omitted.
	}
	return &scope, nil
}

func isDeclarableUnitName(name string) bool {
	for _, s := range []string{".service", ".timer", ".socket", ".target", ".path", ".mount"} {
		if strings.HasSuffix(name, s) {
			return true
		}
	}
	return false
}

// dbpath returns rpm --dbpath args when reading a non-default root.
func dbpath(root string) []string {
	if root == "" || root == "/" {
		return nil
	}
	return []string{"--root", root}
}

// hashFile returns the SHA256 hex digest of a file's bytes.
func hashFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

// fileMeta returns mode/user/group strings for a path (best-effort).
func fileMeta(info fs.FileInfo) (mode, user, group string) {
	mode = fmt.Sprintf("%04o", info.Mode().Perm())
	user, group = "root", "root"
	return
}
