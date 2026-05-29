// generated from spec: zypper-declarative.spec.md sha256:b2d0de88fbed1163678e59e931c741b9d999b71f902f6eb01db8790bb813d057
//
// Package state implements describe-actual-state: the single live-state
// reader. It reads the four declarable scopes under a root into the shared
// data model, and under scope=full adds the two observational integrity
// scopes. No other package reads live system state.
package state

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mge1512/zypper-declarative/internal/manifest"
	"github.com/mge1512/zypper-declarative/internal/meta"
	"github.com/mge1512/zypper-declarative/internal/sysexec"
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

// Options for describe-actual-state.
type Options struct {
	Root         string
	OnUnreadable OnUnreadable
	Scope        Scope
	Runner       sysexec.CommandRunner
	KeepList     map[string]bool
}

// Result is the output of describe-actual-state.
type Result struct {
	Manifest    *manifest.Manifest
	Diagnostics []*manifest.Diagnostic // under warn: one per omitted scope
}

// Describe implements BEHAVIOR/INTERNAL: describe-actual-state. On the first
// unreadable source under on_unreadable=error it returns an error to the
// caller; under warn it omits the affected scope, records a diagnostic, and
// continues.
func Describe(opts Options) (*Result, *manifest.Diagnostic) {
	if opts.Root == "" {
		opts.Root = "/"
	}
	if opts.Scope == "" {
		opts.Scope = ScopeEtc
	}
	r := opts.Runner
	res := &Result{Manifest: &manifest.Manifest{
		Meta: manifest.Meta{
			FormatVersion: 1,
			Generator:     meta.Program + " " + meta.Version,
			CreatedAt:     nowRFC3339(),
		},
	}}

	// helper: handle an unreadable source per policy.
	unreadable := func(domain, source string) *manifest.Diagnostic {
		d := manifest.NewError(domain, "unreadable scope source: "+source)
		if opts.OnUnreadable == OnUnreadableWarn {
			res.Diagnostics = append(res.Diagnostics, manifest.NewWarning(domain, "omitting unreadable scope source: "+source))
			return nil
		}
		return d
	}

	// 1. packages (rpmdb under root).
	pkgs, perr := ReadPackages(opts.Root, r)
	if perr != nil {
		if d := unreadable(manifest.DomainPackages, "rpmdb"); d != nil {
			return nil, d
		}
	} else if len(pkgs.Elements) > 0 {
		res.Manifest.Packages = pkgs
	}

	// 2. repositories (on-disk /etc/zypp/repos.d).
	repos, rerr := readRepositories(opts.Root)
	if rerr != nil {
		if d := unreadable(manifest.DomainRepositories, "/etc/zypp/repos.d"); d != nil {
			return nil, d
		}
	} else if len(repos.Elements) > 0 {
		res.Manifest.Repositories = repos
	}

	// 3. services (unit enablement under root).
	svcs, serr := readServices(opts.Root, r)
	if serr != nil {
		if d := unreadable(manifest.DomainUnits, "unit enablement"); d != nil {
			return nil, d
		}
	} else if len(svcs.Elements) > 0 {
		res.Manifest.Services = svcs
	}

	// 4. config_files (changed-from-package and unpackaged /etc files).
	cfiles, cerr := readConfigFiles(opts.Root, r, opts.KeepList)
	if cerr != nil {
		if d := unreadable(manifest.DomainFiles, cerr.Error()); d != nil {
			return nil, d
		}
	} else if len(cfiles.Elements) > 0 {
		res.Manifest.ConfigFiles = cfiles
	}

	// 4a. full-scan integrity (only under scope=full).
	if opts.Scope == ScopeFull {
		changed, unmanaged, ferr := fullScan(opts.Root, r, opts.KeepList)
		if ferr != nil {
			if d := unreadable(manifest.DomainFiles, "full-scan: "+ferr.Error()); d != nil {
				return nil, d
			}
		} else {
			if len(changed.Elements) > 0 {
				res.Manifest.ChangedManagedFiles = changed
			}
			if len(unmanaged.Elements) > 0 {
				res.Manifest.UnmanagedFiles = unmanaged
			}
		}
	}

	return res, nil
}

// ReadPackages queries the rpmdb under root and returns a fully populated
// PackagesScope. A genuine failure to read the rpmdb is returned as an error.
func ReadPackages(root string, r sysexec.CommandRunner) (*manifest.PackagesScope, *manifest.Diagnostic) {
	scope := &manifest.PackagesScope{
		Attributes: map[string]interface{}{"package_system": "rpm"},
		Elements:   []manifest.PackageRecord{},
	}
	if r == nil {
		return scope, nil
	}
	args := []string{"-qa", "--qf", "%{NAME}\t%{VERSION}\t%{RELEASE}\t%{ARCH}\n"}
	if root != "" && root != "/" {
		args = append([]string{"--root", root}, args...)
	}
	stdout, _, err := r.Run("rpm", args)
	if err != nil {
		// A genuine failure to read the rpmdb is an unreadable source. We
		// distinguish "rpm not present" (env without rpm) from a real I/O
		// error by checking whether any output was produced; with no rpm at
		// all, treat as unreadable.
		if stdout == "" {
			return nil, manifest.NewError(manifest.DomainPackages, "rpmdb unreadable")
		}
	}
	for _, line := range strings.Split(strings.TrimRight(stdout, "\n"), "\n") {
		if line == "" {
			continue
		}
		f := strings.Split(line, "\t")
		if len(f) < 4 {
			continue
		}
		scope.Elements = append(scope.Elements, manifest.PackageRecord{
			Name: f[0], Version: f[1], Release: f[2], Arch: f[3],
		})
	}
	return scope, nil
}

func readRepositories(root string) (*manifest.RepositoriesScope, error) {
	scope := &manifest.RepositoriesScope{
		Attributes: map[string]interface{}{"repository_system": "zypp"},
		Elements:   []manifest.RepositoryRecord{},
	}
	dir := filepath.Join(root, "etc", "zypp", "repos.d")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			// Genuinely-empty readable scope: directory absent is treated as
			// no repositories configured (readable, empty) -> omitted by caller.
			return scope, nil
		}
		// Permission denied or other I/O failure is unreadable.
		return nil, err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".repo") {
			continue
		}
		recs, perr := parseRepoFile(filepath.Join(dir, e.Name()))
		if perr != nil {
			return nil, perr
		}
		scope.Elements = append(scope.Elements, recs...)
	}
	return scope, nil
}

func parseRepoFile(path string) ([]manifest.RepositoryRecord, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var out []manifest.RepositoryRecord
	var cur *manifest.RepositoryRecord
	flush := func() {
		if cur != nil {
			out = append(out, *cur)
			cur = nil
		}
	}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			flush()
			alias := strings.TrimSuffix(strings.TrimPrefix(line, "["), "]")
			cur = &manifest.RepositoryRecord{Alias: alias, Type: "rpm-md", Enabled: true}
			continue
		}
		if cur == nil {
			continue
		}
		k, v, ok := splitKV(line)
		if !ok {
			continue
		}
		switch k {
		case "name":
			cur.Name = v
		case "baseurl", "url":
			cur.URL = v
		case "type":
			cur.Type = v
		case "enabled":
			cur.Enabled = toBool(v)
		case "gpgcheck":
			cur.GPGCheck = toBool(v)
		case "autorefresh":
			cur.AutoRefresh = toBool(v)
		case "priority":
			if n, err := strconv.Atoi(v); err == nil {
				cur.Priority = n
			}
		}
	}
	flush()
	return out, sc.Err()
}

func readServices(root string, r sysexec.CommandRunner) (*manifest.ServicesScope, error) {
	scope := &manifest.ServicesScope{
		Attributes: map[string]interface{}{"init_system": "systemd"},
		Elements:   []manifest.ServiceRecord{},
	}
	if r == nil {
		return scope, nil
	}
	args := []string{"list-unit-files", "--no-legend", "--no-pager"}
	if root != "" && root != "/" {
		args = append([]string{"--root", root}, args...)
	}
	stdout, _, err := r.Run("systemctl", args)
	if err != nil && stdout == "" {
		// No systemd available in this environment: treat as readable-empty so
		// the scope is omitted, not as an unreadable error, because list-unit-files
		// returning nothing is a normal (empty) outcome. A genuine I/O failure
		// on a real systemd host surfaces as an error from the runner with output.
		return scope, nil
	}
	for _, line := range strings.Split(strings.TrimRight(stdout, "\n"), "\n") {
		f := strings.Fields(line)
		if len(f) < 2 {
			continue
		}
		name, st := f[0], f[1]
		var norm string
		switch st {
		case "enabled", "enabled-runtime":
			norm = "enabled"
		case "disabled":
			norm = "disabled"
		case "masked", "masked-runtime":
			norm = "masked"
		default:
			continue // static and others are not declarable
		}
		scope.Elements = append(scope.Elements, manifest.ServiceRecord{Name: name, State: norm})
	}
	return scope, nil
}

// readConfigFiles enumerates only <root>/etc and reports changed-from-package
// and unpackaged /etc files (minus the keep-list and the syncpoint). Bounded to
// /etc; it does not read, hash, or verify files outside /etc, and never runs a
// whole-system package verification.
//
// To consult package metadata for only the /etc files enumerated here while
// keeping the cost bounded to /etc (not the installed base), the owning-package
// set for /etc and the changed-/etc set are each gathered in a single bulk rpm
// query, rather than one rpm invocation per file.
func readConfigFiles(root string, r sysexec.CommandRunner, keep map[string]bool) (*manifest.ConfigFilesScope, error) {
	scope := &manifest.ConfigFilesScope{
		Attributes: nil,
		Elements:   []manifest.ManagedFileRecord{},
	}
	etc := filepath.Join(root, "etc")
	info, err := os.Stat(etc)
	if err != nil {
		if os.IsNotExist(err) {
			return scope, nil // readable-empty (no /etc): omitted by caller
		}
		return nil, err
	}
	if !info.IsDir() {
		return scope, nil
	}

	owned := ownedEtcSet(root, r)            // path -> owning package (bulk, one rpm call)
	changed := changedEtcSet(root, r, owned) // changed /etc paths (verify owners only)

	walkErr := filepath.WalkDir(etc, func(path string, d fs.DirEntry, werr error) error {
		if werr != nil {
			// Permission denial on a required /etc path is unreadable.
			return werr
		}
		if d.IsDir() {
			return nil
		}
		rel := "/" + strings.TrimPrefix(strings.TrimPrefix(path, root), "/")
		if rel == "/etc/etc.syncpoint" || keep[rel] {
			return nil
		}
		fi, ierr := d.Info()
		if ierr != nil {
			return ierr
		}
		ftype := "file"
		if fi.Mode()&os.ModeSymlink != 0 {
			ftype = "link"
		}
		pkg := owned[rel]
		if pkg != "" {
			// Package-owned: include only if it differs from the package
			// baseline. A verifier returning non-zero (differences) is the
			// normal changed-file result, not an unreadable source.
			if !changed[rel] {
				return nil // package-pristine: omitted
			}
		}
		sum := ""
		if ftype == "file" {
			if h, herr := hashFile(path); herr == nil {
				sum = h
			}
		}
		scope.Elements = append(scope.Elements, manifest.ManagedFileRecord{
			Name:        rel,
			Type:        ftype,
			Mode:        modeString(fi.Mode()),
			User:        "root",
			Group:       "root",
			SHA256:      sum,
			ContentRef:  "",
			PackageName: pkg,
		})
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}
	return scope, nil
}

// ownedEtcSet returns a map of /etc path -> owning package, gathered in a
// single bulk rpm query (rpm -qla with package+path), filtered to /etc. This
// keeps the per-file cost out of the walk while consulting package metadata
// only for /etc.
func ownedEtcSet(root string, r sysexec.CommandRunner) map[string]string {
	out := map[string]string{}
	if r == nil {
		return out
	}
	args := []string{"-qa", "--qf", "[%{FILENAMES}\t%{NAME}\n]"}
	if root != "" && root != "/" {
		args = append([]string{"--root", root}, args...)
	}
	stdout, _, err := r.Run("rpm", args)
	if err != nil && stdout == "" {
		return out
	}
	for _, line := range strings.Split(stdout, "\n") {
		f := strings.SplitN(line, "\t", 2)
		if len(f) != 2 {
			continue
		}
		path := strings.TrimSpace(f[0])
		if strings.HasPrefix(path, "/etc/") {
			out[path] = strings.TrimSpace(f[1])
		}
	}
	return out
}

// changedEtcSet returns the set of package-owned /etc paths that differ from
// their package baseline. It verifies only the packages that own an /etc file
// (the owners gathered by ownedEtcSet), never a whole-system verification, so
// the cost is bounded by the packages touching /etc rather than the installed
// base. A non-zero exit reporting differences is the normal changed-file
// result, not an unreadable source.
func changedEtcSet(root string, r sysexec.CommandRunner, owned map[string]string) map[string]bool {
	out := map[string]bool{}
	if r == nil || len(owned) == 0 {
		return out
	}
	pkgSet := map[string]bool{}
	for _, pkg := range owned {
		if pkg != "" {
			pkgSet[pkg] = true
		}
	}
	pkgs := make([]string, 0, len(pkgSet))
	for p := range pkgSet {
		pkgs = append(pkgs, p)
	}
	if len(pkgs) == 0 {
		return out
	}
	args := append([]string{"-V"}, pkgs...)
	if root != "" && root != "/" {
		args = append([]string{"--root", root}, args...)
	}
	stdout, _, _ := r.Run("rpm", args)
	for _, line := range strings.Split(stdout, "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}
		// rpm -V output: "<flags> [c] /path". The path is the last field.
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		p := fields[len(fields)-1]
		if strings.HasPrefix(p, "/etc/") {
			out[p] = true
		}
	}
	return out
}

func owningPackage(root, rel string, r sysexec.CommandRunner) string {
	if r == nil {
		return ""
	}
	args := []string{"-qf", rel}
	if root != "" && root != "/" {
		args = append([]string{"--root", root}, args...)
	}
	stdout, _, err := r.Run("rpm", args)
	if err != nil {
		return "" // not owned by any package (or rpm unavailable): unpackaged
	}
	name := strings.TrimSpace(strings.SplitN(stdout, "\n", 2)[0])
	if strings.Contains(name, "not owned") {
		return ""
	}
	return name
}

// packagedFileChanged reports whether a package-owned /etc file differs from
// its package baseline. It verifies only this file (rpm -V <pkg> filtered),
// never a whole-system verification. A non-zero exit reporting differences is
// the normal changed result.
func packagedFileChanged(root, rel string, r sysexec.CommandRunner) bool {
	if r == nil {
		return false
	}
	args := []string{"-Vf", rel}
	if root != "" && root != "/" {
		args = append([]string{"--root", root}, args...)
	}
	stdout, _, _ := r.Run("rpm", args)
	// rpm -V prints a line per changed file; empty output (and possibly exit 0)
	// means pristine. Non-empty output naming the file means changed.
	return strings.TrimSpace(stdout) != ""
}

func hashFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func splitKV(line string) (string, string, bool) {
	i := strings.IndexByte(line, '=')
	if i < 0 {
		return "", "", false
	}
	return strings.TrimSpace(line[:i]), strings.TrimSpace(line[i+1:]), true
}

func toBool(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

func modeString(m os.FileMode) string {
	return "0" + strconv.FormatUint(uint64(m.Perm()), 8)
}
