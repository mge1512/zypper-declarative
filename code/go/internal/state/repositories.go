// generated from spec: zypper-declarative.spec.md sha256:27aee8e374eb3507189bad0b78339109d0116e6a13b55ae4df2ba9a18e769fc4
//
// Repositories actual state, read from the on-disk zypp configuration
// (<root>/etc/zypp/repos.d/*.repo INI files), not from a network refresh or a
// privileged cache. These files are world-readable in the normal case.
package state

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/mge1512/zypper-declarative/internal/manifest"
)

// readRepositories parses every .repo file under <root>/etc/zypp/repos.d.
// Returns (records, unreadable, error). A readable-but-empty directory yields an
// empty slice and unreadable=false (the caller then omits the scope).
func readRepositories(root string) ([]manifest.RepositoryRecord, bool, error) {
	dir := filepath.Join(root, "etc", "zypp", "repos.d")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			// A missing repos.d on a non-zypp root is a genuinely empty source,
			// not an access failure; treat as empty (scope omitted).
			return nil, false, nil
		}
		return nil, true, err
	}
	var recs []manifest.RepositoryRecord
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".repo") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		secs, perr := parseRepoFile(path)
		if perr != nil {
			return nil, true, perr
		}
		recs = append(recs, secs...)
	}
	sort.SliceStable(recs, func(i, j int) bool { return recs[i].Alias < recs[j].Alias })
	return recs, false, nil
}

// parseRepoFile parses an INI .repo file into one or more RepositoryRecords.
func parseRepoFile(path string) ([]manifest.RepositoryRecord, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var recs []manifest.RepositoryRecord
	var cur *manifest.RepositoryRecord
	flush := func() {
		if cur != nil {
			recs = append(recs, *cur)
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
			cur = &manifest.RepositoryRecord{Alias: alias, Type: "rpm-md"}
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
			if n, e := strconv.Atoi(val); e == nil {
				cur.Priority = n
			}
		}
	}
	flush()
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return recs, nil
}

func iniBool(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
