// generated from spec: zypper-declarative.spec.md sha256:58e1636e2de82ab81a5cd3f81d6b3c9ac6a8976e18f9abb2bbd2b2aba56fe4d4
//
// Package state implements describe-actual-state, the single live-state reader.
// It reads the four declarable scopes under a given root and returns a Manifest
// in the shared schema. No other code reads live system state. Reads are file-
// and-database level (no network refresh, no daemon, no privileged cache).
package state

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mge1512/zypper-declarative/internal/diag"
	"github.com/mge1512/zypper-declarative/internal/manifest"
	"github.com/mge1512/zypper-declarative/internal/system"
)

// OnUnreadable mirrors the config policy for treating an unreadable source.
type OnUnreadable string

const (
	OnUnreadableError OnUnreadable = "error"
	OnUnreadableWarn  OnUnreadable = "warn"
)

// Result is the output of Describe.
type Result struct {
	Manifest    manifest.Manifest
	Diagnostics []*diag.Diagnostic // under warn: one per omitted unreadable scope
}

// Reader reads actual state through a CommandRunner. The runner is injected so
// the package stays the only live reader and is testable with a fake.
type Reader struct {
	Runner   system.CommandRunner
	KeepList map[string]bool
}

// NewReader returns a Reader using the production OS command runner.
func NewReader() *Reader {
	return &Reader{Runner: &system.OSCommandRunner{}, KeepList: map[string]bool{}}
}

const syncpoint = "/etc/etc.syncpoint"

// Describe reads the actual state of the four declarable scopes under root and
// returns a Manifest. Under on_unreadable=error the first unreadable source
// returns an error to the caller; under warn the affected scope is omitted with a
// diagnostic and processing continues. A scope source that cannot be read is
// never represented as an empty scope; a genuinely-empty readable scope is
// omitted.
func (r *Reader) Describe(root string, onUnreadable OnUnreadable) (Result, *diag.Diagnostic) {
	res := Result{Manifest: manifest.Empty()}
	res.Manifest.Meta.Generator = "zypper-declarative"

	// 1. packages: query the rpmdb under root.
	pkgs, perr := r.readPackages(root)
	if perr != nil {
		if onUnreadable == OnUnreadableError {
			return Result{}, diag.Errorf(diag.DomainPackages, "rpmdb unreadable under %s: %v", root, perr)
		}
		res.Diagnostics = append(res.Diagnostics, diag.Warnf(diag.DomainPackages, "rpmdb unreadable under %s, scope omitted: %v", root, perr))
	} else if len(pkgs) > 0 {
		res.Manifest.Packages = &manifest.PackagesScope{
			Attributes: manifest.ScopeAttributes{"package_system": "rpm"},
			Elements:   pkgs,
		}
	}

	// 2. repositories: read on-disk zypp .repo files.
	repos, rerr := r.readRepositories(root)
	if rerr != nil {
		if onUnreadable == OnUnreadableError {
			return Result{}, diag.Errorf(diag.DomainRepositories, "repos.d unreadable under %s: %v", root, rerr)
		}
		res.Diagnostics = append(res.Diagnostics, diag.Warnf(diag.DomainRepositories, "repos.d unreadable under %s, scope omitted: %v", root, rerr))
	} else if len(repos) > 0 {
		res.Manifest.Repositories = &manifest.RepositoriesScope{
			Attributes: manifest.ScopeAttributes{"repository_system": "zypp"},
			Elements:   repos,
		}
	}

	// 3. services: query unit enablement under root.
	svcs, serr := r.readServices(root)
	if serr != nil {
		if onUnreadable == OnUnreadableError {
			return Result{}, diag.Errorf(diag.DomainUnits, "unit enablement unreadable under %s: %v", root, serr)
		}
		res.Diagnostics = append(res.Diagnostics, diag.Warnf(diag.DomainUnits, "unit enablement unreadable under %s, scope omitted: %v", root, serr))
	} else if len(svcs) > 0 {
		res.Manifest.Services = &manifest.ServicesScope{
			Attributes: manifest.ScopeAttributes{"init_system": "systemd"},
			Elements:   svcs,
		}
	}

	// 4. config_files: enumerate /etc under root.
	files, ferr := r.readConfigFiles(root)
	if ferr != nil {
		if onUnreadable == OnUnreadableError {
			return Result{}, diag.Errorf(diag.DomainFiles, "/etc unreadable under %s: %v", root, ferr)
		}
		res.Diagnostics = append(res.Diagnostics, diag.Warnf(diag.DomainFiles, "/etc unreadable under %s, scope omitted: %v", root, ferr))
	} else if len(files) > 0 {
		res.Manifest.ConfigFiles = &manifest.ConfigFilesScope{
			Attributes: nil,
			Elements:   files,
		}
	}

	return res, nil
}

// readPackages queries the rpmdb under root via the rpm CLI. An empty installed
// set (no rpm output, no error) yields an empty slice, not an error.
func (r *Reader) readPackages(root string) ([]manifest.PackageRecord, error) {
	args := []string{}
	if root != "" && root != "/" {
		args = append(args, "--root", root)
	}
	args = append(args, "-qa", "--queryformat", "%{NAME}\t%{VERSION}\t%{RELEASE}\t%{ARCH}\n")
	stdout, _, err := r.Runner.Run("rpm", args)
	if err != nil {
		return nil, err
	}
	var out []manifest.PackageRecord
	sc := bufio.NewScanner(strings.NewReader(stdout))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		f := strings.Split(line, "\t")
		if len(f) < 4 {
			continue
		}
		out = append(out, manifest.PackageRecord{
			Name: f[0], Version: f[1], Release: f[2], Arch: f[3],
		})
	}
	return out, sc.Err()
}

// readRepositories reads the on-disk zypp .repo files under <root>/etc/zypp/repos.d.
// An unreadable directory is an error; a readable directory with no .repo files
// yields an empty slice (genuinely empty).
func (r *Reader) readRepositories(root string) ([]manifest.RepositoryRecord, error) {
	dir := filepath.Join(rootOrSlash(root), "etc", "zypp", "repos.d")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			// A genuinely-absent directory is treated as empty (not an error):
			// there are simply no declared repositories on this root.
			return nil, nil
		}
		return nil, err
	}
	var out []manifest.RepositoryRecord
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".repo") {
			continue
		}
		data, rerr := os.ReadFile(filepath.Join(dir, e.Name()))
		if rerr != nil {
			return nil, rerr
		}
		recs := parseRepoFile(string(data))
		out = append(out, recs...)
	}
	return out, nil
}

// parseRepoFile parses an INI-style .repo file into RepositoryRecords. Each
// section is a repository; the section header is the alias.
func parseRepoFile(content string) []manifest.RepositoryRecord {
	var out []manifest.RepositoryRecord
	var cur *manifest.RepositoryRecord
	flush := func() {
		if cur != nil {
			if cur.Type == "" {
				cur.Type = "rpm-md"
			}
			out = append(out, *cur)
			cur = nil
		}
	}
	sc := bufio.NewScanner(strings.NewReader(content))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			flush()
			alias := strings.TrimSpace(line[1 : len(line)-1])
			cur = &manifest.RepositoryRecord{Alias: alias, Enabled: true, Priority: 99}
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
			cur.Enabled = parseBoolINI(val)
		case "gpgcheck":
			cur.GPGCheck = parseBoolINI(val)
		case "autorefresh":
			cur.Autorefresh = parseBoolINI(val)
		case "priority":
			if p, err := strconv.Atoi(val); err == nil {
				cur.Priority = p
			}
		}
	}
	flush()
	return out
}

func parseBoolINI(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// readServices queries unit enablement under root via systemctl. Purely-static
// units are omitted. An empty result (no enabled/disabled/masked units) yields
// an empty slice.
func (r *Reader) readServices(root string) ([]manifest.ServiceRecord, error) {
	args := []string{}
	if root != "" && root != "/" {
		args = append(args, "--root", root)
	}
	args = append(args, "list-unit-files", "--no-legend", "--no-pager")
	stdout, _, err := r.Runner.Run("systemctl", args)
	if err != nil {
		return nil, err
	}
	var out []manifest.ServiceRecord
	sc := bufio.NewScanner(strings.NewReader(stdout))
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 2 {
			continue
		}
		name := fields[0]
		state := normaliseUnitState(fields[1])
		if state == "" {
			continue // static / generated / not declarable
		}
		out = append(out, manifest.ServiceRecord{Name: name, State: state})
	}
	return out, sc.Err()
}

func normaliseUnitState(s string) string {
	switch s {
	case "enabled", "enabled-runtime":
		return "enabled"
	case "disabled":
		return "disabled"
	case "masked", "masked-runtime":
		return "masked"
	default:
		return ""
	}
}

// readConfigFiles enumerates /etc under root and, for each file that is changed
// from its package default or unpackaged, builds a ManagedFileRecord with the
// actual sha256, mode, user, group, type, and package_name. Package-pristine
// files, the keep-list, and the syncpoint are skipped. content_ref is "".
//
// Determining "changed from package default" precisely requires querying rpm
// verification per file. To keep this deterministic and to avoid emitting an
// empty scope on a read failure, this implementation reports unpackaged /etc
// files (package_name == "") and files rpm reports as modified. A read failure
// on the /etc tree itself is an error.
func (r *Reader) readConfigFiles(root string) ([]manifest.ManagedFileRecord, error) {
	etc := filepath.Join(rootOrSlash(root), "etc")
	info, err := os.Stat(etc)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // no /etc on this root: genuinely empty
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, nil
	}

	var out []manifest.ManagedFileRecord
	walkErr := filepath.WalkDir(etc, func(path string, d os.DirEntry, werr error) error {
		if werr != nil {
			// A subtree we cannot read: surface as an error to honour the
			// never-emit-empty rule (caller decides error vs warn).
			return werr
		}
		if d.IsDir() {
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		rel := "/" + filepath.ToSlash(mustRel(rootOrSlash(root), path))
		if rel == syncpoint || r.KeepList[rel] {
			return nil
		}
		owner := r.ownerPackage(root, rel)
		// Only report unpackaged or changed files; package-pristine files are
		// absent. Without per-file rpm verification we treat owned files as
		// pristine unless rpm reports them modified.
		modified := owner != "" && r.fileModified(root, rel)
		if owner != "" && !modified {
			return nil // package-pristine: absent from the scope
		}
		sum, herr := fileSHA256(path)
		if herr != nil {
			return herr
		}
		mode, uid, gid := fileStat(path)
		out = append(out, manifest.ManagedFileRecord{
			Name:        rel,
			Type:        "file",
			Mode:        mode,
			User:        uid,
			Group:       gid,
			SHA256:      sum,
			ContentRef:  "",
			PackageName: owner,
		})
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}
	return out, nil
}

// ownerPackage returns the owning package name for a path, or "" if unpackaged.
func (r *Reader) ownerPackage(root, path string) string {
	args := []string{}
	if root != "" && root != "/" {
		args = append(args, "--root", root)
	}
	args = append(args, "-qf", path)
	stdout, _, err := r.Runner.Run("rpm", args)
	if err != nil {
		return "" // not owned by any package
	}
	line := strings.TrimSpace(stdout)
	if line == "" || strings.Contains(line, "not owned") {
		return ""
	}
	return line
}

// fileModified reports whether rpm verification flags this path as changed.
func (r *Reader) fileModified(root, path string) bool {
	args := []string{}
	if root != "" && root != "/" {
		args = append(args, "--root", root)
	}
	args = append(args, "-Vf", path)
	stdout, _, _ := r.Runner.Run("rpm", args)
	// rpm -V prints a line with verification flags when something differs; empty
	// output means pristine.
	return strings.TrimSpace(stdout) != ""
}

func fileSHA256(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func rootOrSlash(root string) string {
	if root == "" {
		return "/"
	}
	return root
}

func mustRel(base, target string) string {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return strings.TrimPrefix(target, base)
	}
	return rel
}
