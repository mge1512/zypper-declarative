// generated from spec: zypper-declarative.spec.md sha256:f2cc80627e483a48bb8411d297711bc5f6c6e74c28dbf0dafc8fe7bd8817251e
//
// Package state implements describe-actual-state, the single live-state reader.
// It is the only code that reads live system state (rpmdb, repos.d, systemd,
// and the /etc tree). Reads are file-and-database level (no network refresh).
package state

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mge1512/zypper-declarative/internal/diag"
	"github.com/mge1512/zypper-declarative/internal/manifest"
)

// OnUnreadable controls how an unreadable source is treated.
type OnUnreadable string

const (
	OnUnreadableError OnUnreadable = "error"
	OnUnreadableWarn  OnUnreadable = "warn"
)

// Scope is the actual-state read scope.
type Scope string

const (
	ScopeEtc  Scope = "etc"
	ScopeFull Scope = "full"
)

// CommandRunner abstracts external command execution so callers can drive rpm,
// systemctl, and zypper through their CLIs (keeping CGO_ENABLED=0).
type CommandRunner interface {
	Run(cmd string, args ...string) (stdout string, stderr string, err error)
}

// OSCommandRunner runs commands via os/exec with a sanitised PATH.
type OSCommandRunner struct{}

// Run executes cmd with args and returns stdout, stderr, and the run error.
func (r *OSCommandRunner) Run(cmd string, args ...string) (string, string, error) {
	oldPath := os.Getenv("PATH")
	_ = os.Setenv("PATH", "/sbin:/bin:/usr/bin:/usr/sbin")
	defer func() { _ = os.Setenv("PATH", oldPath) }()

	c := exec.Command(cmd, args...)
	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr
	err := c.Run()
	return stdout.String(), stderr.String(), err
}

// Options for describe-actual-state.
type Options struct {
	OnUnreadable OnUnreadable
	Scope        Scope
	KeepList     map[string]bool
	Runner       CommandRunner
}

// Result is the actual state and any warn diagnostics.
type Result struct {
	Manifest    *manifest.Manifest
	Diagnostics []*diag.Diagnostic
}

// keepListDefault paths never reported or deleted.
var keepListDefault = map[string]bool{
	"/etc/machine-id": true,
}

// DescribeActualState reads the actual state of the declarable scopes under
// root and returns a Manifest in the shared schema. On an unreadable source
// under on_unreadable=error it returns a *diag.Diagnostic to the caller.
func DescribeActualState(root string, opts Options) (*Result, error) {
	if opts.Runner == nil {
		opts.Runner = &OSCommandRunner{}
	}
	if opts.OnUnreadable == "" {
		opts.OnUnreadable = OnUnreadableError
	}
	if opts.Scope == "" {
		opts.Scope = ScopeEtc
	}
	keep := mergeKeep(opts.KeepList)

	res := &Result{Manifest: &manifest.Manifest{
		Meta: manifest.ManifestMeta{
			FormatVersion: 1,
			Generator:     "zypper-declarative 0.6.2",
			CreatedAt:     time.Now().UTC().Format(time.RFC3339),
		},
	}}

	// 1. packages
	pkgs, err := readPackages(root, opts)
	if d := handleSource(opts, res, "rpmdb", diag.DomainPackages, err); d != nil {
		return nil, d
	} else if err == nil && pkgs != nil && len(pkgs.Elements) > 0 {
		res.Manifest.Packages = pkgs
	}

	// 2. repositories
	repos, err := readRepositories(root)
	if d := handleSource(opts, res, filepath.Join(root, "etc/zypp/repos.d"), diag.DomainRepositories, err); d != nil {
		return nil, d
	} else if err == nil && repos != nil && len(repos.Elements) > 0 {
		res.Manifest.Repositories = repos
	}

	// 3. services
	svcs, err := readServices(root, opts)
	if d := handleSource(opts, res, "unit enablement", diag.DomainServices, err); d != nil {
		return nil, d
	} else if err == nil && svcs != nil && len(svcs.Elements) > 0 {
		res.Manifest.Services = svcs
	}

	// 4. config_files (/etc walk)
	cf, err := readConfigFiles(root, keep, opts)
	if d := handleSource(opts, res, filepath.Join(root, "etc"), diag.DomainFiles, err); d != nil {
		return nil, d
	} else if err == nil && cf != nil && len(cf.Elements) > 0 {
		res.Manifest.ConfigFiles = cf
	}

	// 4a. full-scan integrity
	if opts.Scope == ScopeFull {
		changed, unmanaged, ferr := readFullScan(root, keep, opts)
		if d := handleSource(opts, res, "full scan", diag.DomainFiles, ferr); d != nil {
			return nil, d
		}
		if ferr == nil {
			if changed != nil && len(changed.Elements) > 0 {
				res.Manifest.ChangedManagedFiles = changed
			}
			if unmanaged != nil && len(unmanaged.Elements) > 0 {
				res.Manifest.UnmanagedFiles = unmanaged
			}
		}
	}

	return res, nil
}

func mergeKeep(extra map[string]bool) map[string]bool {
	m := map[string]bool{}
	for k := range keepListDefault {
		m[k] = true
	}
	for k := range extra {
		m[k] = true
	}
	return m
}

// handleSource maps a source read error to either a returned error (strict) or
// an appended warn diagnostic (warn). A nil err is a no-op.
func handleSource(opts Options, res *Result, source string, domain diag.Domain, err error) *diag.Diagnostic {
	if err == nil {
		return nil
	}
	if opts.OnUnreadable == OnUnreadableWarn {
		res.Diagnostics = append(res.Diagnostics, diag.Warn(domain, "unreadable scope source: %s: %v", source, err))
		return nil
	}
	return diag.New(domain, "unreadable scope source: %s: %v", source, err)
}

// readPackages queries the rpmdb under root for the installed set. An empty or
// absent rpmdb under a synthetic root is treated as "no packages readable here"
// (a genuinely empty scope), not an error.
func readPackages(root string, opts Options) (*manifest.PackagesScope, error) {
	dbPaths := []string{
		filepath.Join(root, "usr/lib/sysimage/rpm"),
		filepath.Join(root, "var/lib/rpm"),
	}
	hasDB := false
	for _, p := range dbPaths {
		if fi, err := os.Stat(p); err == nil && fi.IsDir() {
			hasDB = true
			break
		}
	}
	if !hasDB {
		// No rpmdb under this root: a readable-but-empty packages scope.
		return &manifest.PackagesScope{Attributes: map[string]interface{}{"package_system": manifest.PackageSystemRPM}, Elements: []manifest.PackageRecord{}}, nil
	}
	args := []string{"-qa", "--dbpath", filepath.Join(root, "usr/lib/sysimage/rpm"),
		"--qf", "%{NAME}\t%{VERSION}\t%{RELEASE}\t%{ARCH}\n"}
	stdout, stderr, err := opts.Runner.Run("rpm", args...)
	if err != nil {
		// rpm exits non-zero only on a genuine failure here; report it.
		return nil, &runError{msg: strings.TrimSpace(stderr), err: err}
	}
	scope := &manifest.PackagesScope{
		Attributes: map[string]interface{}{"package_system": manifest.PackageSystemRPM},
		Elements:   []manifest.PackageRecord{},
	}
	sc := bufio.NewScanner(strings.NewReader(stdout))
	for sc.Scan() {
		f := strings.Split(sc.Text(), "\t")
		if len(f) < 4 {
			continue
		}
		scope.Elements = append(scope.Elements, manifest.PackageRecord{
			Name: f[0], Version: f[1], Release: f[2], Arch: f[3],
		})
	}
	return scope, nil
}

// readRepositories reads <root>/etc/zypp/repos.d/*.repo directly (INI sections).
func readRepositories(root string) (*manifest.RepositoriesScope, error) {
	dir := filepath.Join(root, "etc/zypp/repos.d")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			// No repos.d under this root: a readable-but-empty repositories scope.
			return &manifest.RepositoriesScope{Attributes: map[string]interface{}{"repository_system": manifest.RepositorySystemZypp}, Elements: []manifest.RepositoryRecord{}}, nil
		}
		return nil, err
	}
	scope := &manifest.RepositoriesScope{
		Attributes: map[string]interface{}{"repository_system": manifest.RepositorySystemZypp},
		Elements:   []manifest.RepositoryRecord{},
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".repo") {
			continue
		}
		data, rerr := os.ReadFile(filepath.Join(dir, e.Name()))
		if rerr != nil {
			return nil, rerr
		}
		scope.Elements = append(scope.Elements, parseRepoFile(data)...)
	}
	return scope, nil
}

// parseRepoFile parses an INI .repo file into RepositoryRecord entries.
func parseRepoFile(data []byte) []manifest.RepositoryRecord {
	var recs []manifest.RepositoryRecord
	var cur *manifest.RepositoryRecord
	flush := func() {
		if cur != nil {
			recs = append(recs, *cur)
			cur = nil
		}
	}
	sc := bufio.NewScanner(bytes.NewReader(data))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			flush()
			alias := strings.TrimSuffix(strings.TrimPrefix(line, "["), "]")
			cur = &manifest.RepositoryRecord{Alias: alias, Type: "rpm-md"}
			continue
		}
		if cur == nil {
			continue
		}
		kv := strings.SplitN(line, "=", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.TrimSpace(kv[0])
		val := strings.TrimSpace(kv[1])
		switch key {
		case "name":
			cur.Name = val
		case "baseurl":
			cur.URL = val
		case "type":
			cur.Type = val
		case "enabled":
			cur.Enabled = toBool(val)
		case "gpgcheck":
			cur.GPGCheck = toBool(val)
		case "autorefresh":
			cur.Autorefresh = toBool(val)
		case "priority":
			if n, err := strconv.Atoi(val); err == nil {
				cur.Priority = n
			}
		}
	}
	flush()
	return recs
}

func toBool(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

// readServices queries unit enablement under root. On a synthetic root with no
// systemd present, it returns a readable-but-empty services scope.
func readServices(root string, opts Options) (*manifest.ServicesScope, error) {
	unitDir := filepath.Join(root, "etc/systemd/system")
	if _, err := os.Stat(unitDir); err != nil {
		// No systemd unit tree under this root: readable-but-empty.
		return &manifest.ServicesScope{Attributes: map[string]interface{}{"init_system": manifest.InitSystemSystemd}, Elements: []manifest.ServiceRecord{}}, nil
	}
	scope := &manifest.ServicesScope{
		Attributes: map[string]interface{}{"init_system": manifest.InitSystemSystemd},
		Elements:   []manifest.ServiceRecord{},
	}
	stdout, _, err := opts.Runner.Run("systemctl", "--root", root, "list-unit-files", "--no-legend", "--no-pager")
	if err != nil {
		return nil, &runError{msg: "unit enablement query failed", err: err}
	}
	sc := bufio.NewScanner(strings.NewReader(stdout))
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 2 {
			continue
		}
		state := fields[1]
		switch state {
		case "enabled", "disabled", "masked":
			scope.Elements = append(scope.Elements, manifest.ServiceRecord{Name: fields[0], State: state})
		default:
			// static and others are observational; omitted.
		}
	}
	return scope, nil
}

// readConfigFiles walks <root>/etc, classifying each entry by its own type
// without following symlinks. It emits changed-or-unpackaged regular files and
// symlinks only; directories are traversed, special files skipped.
func readConfigFiles(root string, keep map[string]bool, opts Options) (*manifest.ConfigFilesScope, error) {
	etc := filepath.Join(root, "etc")
	if _, err := os.Stat(etc); err != nil {
		if os.IsNotExist(err) {
			return &manifest.ConfigFilesScope{Attributes: nil, Elements: []manifest.ManagedFileRecord{}}, nil
		}
		return nil, err
	}
	owners := packageOwnersEtc(root, opts)
	scope := &manifest.ConfigFilesScope{Attributes: nil, Elements: []manifest.ManagedFileRecord{}}

	walkErr := filepath.WalkDir(etc, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		logical := "/" + strings.TrimPrefix(strings.TrimPrefix(p, root), "/")
		if logical == diagSyncpoint || keep[logical] {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		typ := d.Type()
		switch {
		case typ.IsDir():
			return nil // traverse, do not emit
		case typ&fs.ModeSymlink != 0:
			target, lerr := os.Readlink(p)
			if lerr != nil {
				return lerr
			}
			rec := buildRec(p, logical, "link", owners)
			rec.Target = target // verbatim, not resolved
			scope.Elements = append(scope.Elements, rec)
			return nil
		case typ.IsRegular():
			sum, herr := hashFile(p)
			if herr != nil {
				return herr
			}
			rec := buildRec(p, logical, "file", owners)
			rec.SHA256 = sum
			scope.Elements = append(scope.Elements, rec)
			return nil
		default:
			return nil // special file: skip
		}
	})
	if walkErr != nil {
		return nil, walkErr
	}
	return scope, nil
}

const diagSyncpoint = "/etc/etc.syncpoint"

// buildRec constructs a ManagedFileRecord with mode/user/group from lstat.
func buildRec(path, logical, typ string, owners map[string]string) manifest.ManagedFileRecord {
	rec := manifest.ManagedFileRecord{
		Name:        logical,
		Type:        typ,
		User:        "root",
		Group:       "root",
		PackageName: owners[logical],
	}
	if fi, err := os.Lstat(path); err == nil {
		rec.Mode = fmtMode(fi.Mode().Perm())
	} else {
		rec.Mode = "0644"
	}
	return rec
}

func fmtMode(p os.FileMode) string {
	return "0" + strconv.FormatUint(uint64(p), 8)
}

// packageOwnersEtc returns a map logical-path -> owning package for /etc paths,
// derived from the rpmdb. Empty when no rpmdb is present under root.
func packageOwnersEtc(root string, opts Options) map[string]string {
	owners := map[string]string{}
	dbpath := filepath.Join(root, "usr/lib/sysimage/rpm")
	if fi, err := os.Stat(dbpath); err != nil || !fi.IsDir() {
		return owners
	}
	stdout, _, err := opts.Runner.Run("rpm", "-qa", "--dbpath", dbpath, "--qf", "[%{FILENAMES}\t%{NAME}\n]")
	if err != nil {
		return owners
	}
	sc := bufio.NewScanner(strings.NewReader(stdout))
	for sc.Scan() {
		f := strings.SplitN(sc.Text(), "\t", 2)
		if len(f) == 2 && strings.HasPrefix(f[0], "/etc/") {
			owners[f[0]] = f[1]
		}
	}
	return owners
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	buf := make([]byte, 64*1024)
	for {
		n, rerr := f.Read(buf)
		if n > 0 {
			h.Write(buf[:n])
		}
		if rerr != nil {
			break
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// readFullScan scans the package-managed trees outside /etc under scope=full.
func readFullScan(root string, keep map[string]bool, opts Options) (*manifest.ChangedManagedFilesScope, *manifest.UnmanagedFilesScope, error) {
	trees := []string{"usr", "bin", "sbin", "lib", "lib64", "boot"}
	changed := &manifest.ChangedManagedFilesScope{Attributes: nil, Elements: []manifest.ManagedBaselineRecord{}}
	unmanaged := &manifest.UnmanagedFilesScope{Attributes: nil, Elements: []manifest.UnmanagedFileRecord{}}
	owned := ownedPathSet(root, opts)
	for _, t := range trees {
		base := filepath.Join(root, t)
		fi, err := os.Lstat(base)
		if err != nil || !fi.IsDir() {
			continue
		}
		werr := filepath.WalkDir(base, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			logical := "/" + strings.TrimPrefix(strings.TrimPrefix(p, root), "/")
			if keep[logical] {
				if d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			typ := d.Type()
			if typ.IsDir() {
				return nil
			}
			if !typ.IsRegular() && typ&fs.ModeSymlink == 0 {
				return nil // special file
			}
			if _, ok := owned[logical]; ok {
				// Owned: would compare to baseline; baseline comparison via rpm -V
				// is reserved for the live milestone. Skip pristine here.
				return nil
			}
			// Unmanaged addition.
			rec := manifest.UnmanagedFileRecord{Name: logical, User: "root", Group: "root"}
			if typ&fs.ModeSymlink != 0 {
				if target, lerr := os.Readlink(p); lerr == nil {
					rec.Type = "link"
					rec.Target = target
				}
			} else {
				rec.Type = "file"
				if sum, herr := hashFile(p); herr == nil {
					rec.SHA256 = sum
				}
			}
			if lfi, lerr := os.Lstat(p); lerr == nil {
				rec.Mode = fmtMode(lfi.Mode().Perm())
			}
			unmanaged.Elements = append(unmanaged.Elements, rec)
			return nil
		})
		if werr != nil {
			return nil, nil, werr
		}
	}
	return changed, unmanaged, nil
}

// ownedPathSet returns the set of package-owned paths from the rpmdb under root.
func ownedPathSet(root string, opts Options) map[string]bool {
	owned := map[string]bool{}
	dbpath := filepath.Join(root, "usr/lib/sysimage/rpm")
	if fi, err := os.Stat(dbpath); err != nil || !fi.IsDir() {
		return owned
	}
	stdout, _, err := opts.Runner.Run("rpm", "-qa", "--dbpath", dbpath, "--qf", "[%{FILENAMES}\n]")
	if err != nil {
		return owned
	}
	sc := bufio.NewScanner(strings.NewReader(stdout))
	for sc.Scan() {
		owned[strings.TrimSpace(sc.Text())] = true
	}
	return owned
}

// runError wraps an external command failure as a source-read error.
type runError struct {
	msg string
	err error
}

func (e *runError) Error() string {
	if e.msg != "" {
		return e.msg + ": " + e.err.Error()
	}
	return e.err.Error()
}
