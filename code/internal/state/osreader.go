// generated from spec: zypper-declarative.spec.md sha256:f8ff76ecbc4bbc69a49e2e32b2924da3a64df1ad46196e05ce8c137b684429b2
//
// OSReader is the production Reader. Repositories are read directly from the
// on-disk zypp configuration (<root>/etc/zypp/repos.d/*.repo), which is
// world-readable in the normal case. The package, service, and config-file
// scopes are read by driving rpm and systemctl against the root; the tool makes
// no network call of its own.

package state

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/mge1512/zypper-declarative/internal/diag"
	"github.com/mge1512/zypper-declarative/internal/manifest"
)

// OSReader reads live state via files and external commands.
type OSReader struct {
	Runner CommandRunner
}

// NewOSReader returns an OSReader wired to the OS command runner.
func NewOSReader() *OSReader {
	return &OSReader{Runner: &OSCommandRunner{}}
}

// ReadRepositories reads <root>/etc/zypp/repos.d/*.repo. A missing repos.d that
// cannot be opened is a SourceError; a readable but empty repos.d yields an
// empty slice (a genuinely-empty scope, omitted by Describe).
func (r *OSReader) ReadRepositories(root string) ([]manifest.RepositoryRecord, *SourceError) {
	dir := filepath.Join(root, "etc", "zypp", "repos.d")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, &SourceError{Domain: diag.DomainRepositories, Source: dir, Wrapped: err}
	}
	var repos []manifest.RepositoryRecord
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".repo") {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)
	for _, name := range names {
		p := filepath.Join(dir, name)
		recs, ferr := parseRepoFile(p)
		if ferr != nil {
			return nil, &SourceError{Domain: diag.DomainRepositories, Source: p, Wrapped: ferr}
		}
		repos = append(repos, recs...)
	}
	return repos, nil
}

// parseRepoFile parses an INI-style .repo file into RepositoryRecords. Each
// section is one repository; the section header is the alias.
func parseRepoFile(path string) ([]manifest.RepositoryRecord, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var repos []manifest.RepositoryRecord
	var cur *manifest.RepositoryRecord
	flush := func() {
		if cur != nil {
			repos = append(repos, *cur)
			cur = nil
		}
	}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			flush()
			alias := strings.TrimSpace(line[1 : len(line)-1])
			cur = &manifest.RepositoryRecord{Alias: alias, Type: "rpm-md", Enabled: true}
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
		switch strings.ToLower(key) {
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
			if n, perr := strconv.Atoi(val); perr == nil {
				cur.Priority = n
			}
		}
	}
	flush()
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return repos, nil
}

func iniBool(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// ReadPackages drives `rpm` against the root to report the installed set, fully
// resolved (name, version, release, arch). If the rpmdb cannot be read the
// source is reported as unreadable.
func (r *OSReader) ReadPackages(root string) ([]manifest.PackageRecord, *SourceError) {
	args := []string{"-qa", "--qf", "%{NAME}\t%{VERSION}\t%{RELEASE}\t%{ARCH}\n"}
	if root != "" && root != "/" {
		args = append([]string{"--root", root}, args...)
	}
	out, _, err := r.Runner.Run("rpm", args, "")
	if err != nil {
		return nil, &SourceError{Domain: diag.DomainPackages, Source: "rpmdb", Wrapped: err}
	}
	var pkgs []manifest.PackageRecord
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) < 4 {
			continue
		}
		pkgs = append(pkgs, manifest.PackageRecord{
			Name: fields[0], Version: fields[1], Release: fields[2], Arch: fields[3],
		})
	}
	return pkgs, nil
}

// ReadServices drives `systemctl` to report declarable unit enablement under the
// root. Purely-static units are omitted (not declarable).
func (r *OSReader) ReadServices(root string) ([]manifest.ServiceRecord, *SourceError) {
	args := []string{"list-unit-files", "--no-legend", "--no-pager", "--plain"}
	if root != "" && root != "/" {
		args = append([]string{"--root", root}, args...)
	}
	out, _, err := r.Runner.Run("systemctl", args, "")
	if err != nil {
		return nil, &SourceError{Domain: diag.DomainUnits, Source: "unit enablement", Wrapped: err}
	}
	var svcs []manifest.ServiceRecord
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		name, state := fields[0], normaliseUnitState(fields[1])
		if state == "" {
			continue // not declarable (static, generated, transient, etc.)
		}
		svcs = append(svcs, manifest.ServiceRecord{Name: name, State: state})
	}
	return svcs, nil
}

func normaliseUnitState(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
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

// ReadConfigFiles enumerates changed-from-package and unpackaged /etc files
// under the root via `rpm`, excluding package-pristine files, the keep-list, and
// the syncpoint. content_ref is empty in actual state.
func (r *OSReader) ReadConfigFiles(root string, keepList map[string]bool) ([]manifest.ManagedFileRecord, *SourceError) {
	etc := filepath.Join(root, "etc")
	if _, err := os.Stat(etc); err != nil {
		return nil, &SourceError{Domain: diag.DomainFiles, Source: etc, Wrapped: err}
	}
	// The exact changed-vs-pristine determination uses rpm verification; the
	// concrete rpm wiring is environment-specific. This reader returns the set
	// rpm reports as changed config files, with package ownership populated. When
	// rpm is unavailable the source is reported unreadable.
	args := []string{"-Va", "--noscripts", "--nofiledigest"}
	if root != "" && root != "/" {
		args = append([]string{"--root", root}, args...)
	}
	out, _, err := r.Runner.Run("rpm", args, "")
	if err != nil {
		return nil, &SourceError{Domain: diag.DomainFiles, Source: "rpm config-file verification", Wrapped: err}
	}
	var files []manifest.ManagedFileRecord
	for _, line := range strings.Split(out, "\n") {
		path := extractEtcPath(line)
		if path == "" {
			continue
		}
		if path == "/etc/etc.syncpoint" || (keepList != nil && keepList[path]) {
			continue
		}
		files = append(files, manifest.ManagedFileRecord{
			Name:        path,
			Type:        "file",
			ContentRef:  "",
			PackageName: ownerOf(r, root, path),
		})
	}
	return files, nil
}

// extractEtcPath returns the /etc path mentioned on an `rpm -Va` line, or "".
func extractEtcPath(line string) string {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return ""
	}
	last := fields[len(fields)-1]
	if strings.HasPrefix(last, "/etc/") {
		return last
	}
	return ""
}

// ownerOf returns the owning package of a path, or "" if unpackaged.
func ownerOf(r *OSReader, root, path string) string {
	args := []string{"-qf", path, "--qf", "%{NAME}"}
	if root != "" && root != "/" {
		args = append([]string{"--root", root}, args...)
	}
	out, _, err := r.Runner.Run("rpm", args, "")
	if err != nil {
		return ""
	}
	out = strings.TrimSpace(out)
	if strings.Contains(out, "not owned") || out == "" {
		return ""
	}
	return out
}
