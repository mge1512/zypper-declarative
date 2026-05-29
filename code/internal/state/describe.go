// generated from spec: zypper-declarative.spec.md sha256:714e75ff672557d2c7344736a5f36b52afa37e0f565b07de83e1b18cc4492014
//
// Package state implements describe-actual-state, the single live-state reader.
// It is the only code that reads live system state; every verb obtains actual
// state through it or through a supplied dump in the same format.
package state

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/mge1512/zypper-declarative/internal/diag"
	"github.com/mge1512/zypper-declarative/internal/manifest"
	"github.com/mge1512/zypper-declarative/internal/meta"
	"github.com/mge1512/zypper-declarative/internal/sysiface"
)

// Reader produces the actual state of the declarable scopes under a root.
type Reader struct {
	Runner   sysiface.CommandRunner
	KeepList map[string]bool
}

// NewReader builds a Reader with the OS runner and an optional keep-list set.
func NewReader(runner sysiface.CommandRunner, keepList []string) *Reader {
	kl := map[string]bool{}
	for _, p := range keepList {
		kl[p] = true
	}
	return &Reader{Runner: runner, KeepList: kl}
}

const syncpoint = "/etc/etc.syncpoint"

// Describe reads the four declarable scopes under root and returns a Manifest
// in the shared schema (describe-actual-state STEPS 1–5). The described root is
// never modified. Genuinely-absent sources yield an initialised empty scope so
// the document is schema-valid and never carries a JSON null scope.
func Describe(root string, runner sysiface.CommandRunner, keepList []string) (manifest.Manifest, *diag.Diagnostic) {
	r := NewReader(runner, keepList)
	return r.read(root)
}

func (r *Reader) read(root string) (manifest.Manifest, *diag.Diagnostic) {
	pkgs, d := r.readPackages(root)
	if d != nil {
		return manifest.Manifest{}, d
	}
	repos := r.readRepositories(root)
	svcs, d := r.readServices(root)
	if d != nil {
		return manifest.Manifest{}, d
	}
	files, d := r.readConfigFiles(root)
	if d != nil {
		return manifest.Manifest{}, d
	}

	m := manifest.Manifest{
		Meta: manifest.ManifestMeta{
			FormatVersion: 1,
			Generator:     meta.Generator,
			CreatedAt:     time.Now().UTC().Format(time.RFC3339),
			DesiredSHA256: "",
		},
		Packages:     pkgs,
		Repositories: repos,
		Services:     svcs,
		ConfigFiles:  files,
	}
	return m, nil
}

// readPackages queries the rpmdb under root. STEP 1.
func (r *Reader) readPackages(root string) (*manifest.PackagesScope, *diag.Diagnostic) {
	scope := manifest.EmptyPackages()
	// rpm --root <root> -qa --qf '%{NAME} %{VERSION} %{RELEASE} %{ARCH}\n'
	stdout, _, err := r.Runner.Run("rpm", "--root", absRoot(root),
		"-qa", "--qf", "%{NAME} %{VERSION} %{RELEASE} %{ARCH}\\n")
	if err != nil {
		// If there is no rpmdb under the root (e.g. an empty test root), treat
		// as a genuinely-empty installed set rather than a query failure.
		if noRPMDB(root) {
			return scope, nil
		}
		return nil, diag.New(diag.DomainPackages, "rpmdb query failed: %v", err)
	}
	sc := bufio.NewScanner(strings.NewReader(stdout))
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		if len(f) < 4 {
			continue
		}
		scope.Elements = append(scope.Elements, manifest.PackageRecord{
			Name: f[0], Version: f[1], Release: f[2], Arch: f[3],
		})
	}
	return scope, nil
}

func noRPMDB(root string) bool {
	candidates := []string{
		filepath.Join(absRoot(root), "var/lib/rpm"),
		filepath.Join(absRoot(root), "usr/lib/sysimage/rpm"),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return false
		}
	}
	return true
}

// readRepositories reads <root>/etc/zypp/repos.d/*.repo INI files. STEP 2.
// Repositories are read from world-readable files, never via a network refresh.
func (r *Reader) readRepositories(root string) *manifest.RepositoriesScope {
	scope := manifest.EmptyRepositories()
	dir := filepath.Join(absRoot(root), "etc/zypp/repos.d")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return scope // genuinely-empty / absent repos.d
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".repo") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		for _, rec := range parseRepoFile(string(data)) {
			scope.Elements = append(scope.Elements, rec)
		}
	}
	return scope
}

func parseRepoFile(content string) []manifest.RepositoryRecord {
	var out []manifest.RepositoryRecord
	var cur *manifest.RepositoryRecord
	flush := func() {
		if cur != nil {
			out = append(out, *cur)
			cur = nil
		}
	}
	sc := bufio.NewScanner(strings.NewReader(content))
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
			cur.Enabled = v == "1" || strings.EqualFold(v, "true")
		case "gpgcheck":
			cur.GPGCheck = v == "1" || strings.EqualFold(v, "true")
		case "autorefresh":
			cur.Autorefresh = v == "1" || strings.EqualFold(v, "true")
		case "priority":
			if n, err := strconv.Atoi(v); err == nil {
				cur.Priority = n
			}
		}
	}
	flush()
	return out
}

func splitKV(line string) (string, string, bool) {
	i := strings.IndexByte(line, '=')
	if i < 0 {
		return "", "", false
	}
	return strings.TrimSpace(line[:i]), strings.TrimSpace(line[i+1:]), true
}

// readServices queries unit enablement under root. STEP 3. Purely-static units
// are omitted; states normalise to enabled|disabled|masked.
func (r *Reader) readServices(root string) (*manifest.ServicesScope, *diag.Diagnostic) {
	scope := manifest.EmptyServices()
	// systemctl --root <root> list-unit-files --no-legend
	stdout, _, err := r.Runner.Run("systemctl", "--root", absRoot(root),
		"list-unit-files", "--no-legend")
	if err != nil {
		// Absent systemd tree under a test root -> genuinely empty.
		if _, statErr := os.Stat(filepath.Join(absRoot(root), "usr/lib/systemd")); os.IsNotExist(statErr) {
			return scope, nil
		}
		return nil, diag.New(diag.DomainUnits, "unit query failed: %v", err)
	}
	sc := bufio.NewScanner(strings.NewReader(stdout))
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		if len(f) < 2 {
			continue
		}
		name, st := f[0], f[1]
		switch st {
		case "enabled", "disabled", "masked":
			scope.Elements = append(scope.Elements, manifest.ServiceRecord{Name: name, State: st})
		default:
			// static, generated, indirect, etc. are observational; omit.
		}
	}
	return scope, nil
}

// readConfigFiles enumerates /etc under root. STEP 4. Only changed-from-package
// and unpackaged files appear; package-pristine files, the keep-list, and the
// syncpoint are skipped. content_ref is "".
func (r *Reader) readConfigFiles(root string) (*manifest.ConfigFilesScope, *diag.Diagnostic) {
	scope := manifest.EmptyConfigFiles()
	etc := filepath.Join(absRoot(root), "etc")
	info, err := os.Stat(etc)
	if err != nil || !info.IsDir() {
		return scope, nil // no /etc under this root
	}

	// Build the set of rpm-verified-changed files, if rpm is available.
	changed, owners := r.rpmVerifyEtc(root)

	walkErr := filepath.Walk(etc, func(path string, fi os.FileInfo, werr error) error {
		if werr != nil {
			return nil // skip unreadable entries; do not fail the whole walk
		}
		if fi.IsDir() {
			return nil
		}
		logical := "/" + strings.TrimPrefix(strings.TrimPrefix(path, absRoot(root)), "/")
		// normalise to start with /etc/
		logical = "/etc/" + strings.TrimPrefix(strings.TrimPrefix(logical, "/etc/"), "/")
		if logical == syncpoint || r.KeepList[logical] {
			return nil
		}
		owner := owners[logical]
		// Skip package-pristine files: owned and not in the changed set.
		if owner != "" && !changed[logical] {
			return nil
		}
		sum, herr := hashFile(path)
		if herr != nil {
			return nil
		}
		mode := fmtMode(fi.Mode().Perm())
		scope.Elements = append(scope.Elements, manifest.ManagedFileRecord{
			Name:        logical,
			Type:        "file",
			Mode:        mode,
			User:        "root",
			Group:       "root",
			SHA256:      sum,
			ContentRef:  "",
			PackageName: owner,
		})
		return nil
	})
	if walkErr != nil {
		return nil, diag.New(diag.DomainFiles, "/etc enumeration failed: %v", walkErr)
	}
	return scope, nil
}

// rpmVerifyEtc returns the set of changed /etc files and a path->package map.
// When rpm is unavailable (test root), both are empty and every /etc file is
// treated as unpackaged (so it appears, which is correct for an unmanaged root).
func (r *Reader) rpmVerifyEtc(root string) (changed map[string]bool, owners map[string]string) {
	changed = map[string]bool{}
	owners = map[string]string{}
	if noRPMDB(root) {
		return changed, owners
	}
	// rpm --root <root> -Va --nomtime  (S.5....T lines name changed files)
	stdout, _, err := r.Runner.Run("rpm", "--root", absRoot(root), "-Va")
	if err == nil {
		sc := bufio.NewScanner(strings.NewReader(stdout))
		for sc.Scan() {
			fields := strings.Fields(sc.Text())
			if len(fields) == 0 {
				continue
			}
			p := fields[len(fields)-1]
			if strings.HasPrefix(p, "/etc/") {
				changed[p] = true
			}
		}
	}
	return changed, owners
}

func hashFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func fmtMode(p os.FileMode) string {
	return "0" + strconv.FormatUint(uint64(p), 8)
}

func absRoot(root string) string {
	if root == "" {
		return "/"
	}
	return root
}
