// generated from spec: zypper-declarative.spec.md sha256:27aee8e374eb3507189bad0b78339109d0116e6a13b55ae4df2ba9a18e769fc4
//
// Full-scan integrity (scope=full): scan the package-managed OS trees OUTSIDE
// /etc and emit two observational scopes:
//   - changed_managed_files: packaged files differing from the package baseline,
//     found via `rpm -Va` verdict-parse keeping NON-config lines.
//   - unmanaged_files: files no package owns, found by walking the trees and
//     subtracting the rpm-owned path set.
//
// Trees: /usr and the usr-merge roots (/bin /sbin /lib /lib64) and /boot.
// Excluded: /etc, /opt, and virtual/runtime/mutable-data trees. The keep-list is
// honoured. This step is expensive and runs only under scope=full.
package state

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mge1512/zypper-declarative/internal/manifest"
)

var fullScanTrees = []string{"/usr", "/bin", "/sbin", "/lib", "/lib64", "/boot"}

// readFullScan returns the changed_managed_files and unmanaged_files element sets.
func (r *Reader) readFullScan(opts Options) ([]manifest.ManagedBaselineRecord, []manifest.UnmanagedFileRecord, error) {
	root := opts.Root

	// changed_managed_files via rpm -Va, keeping non-config lines outside /etc.
	changed, err := r.changedManagedFiles(root, opts.KeepList)
	if err != nil {
		return nil, nil, err
	}

	// unmanaged_files: walk the trees, subtract rpm-owned paths.
	owned := r.ownedPaths(root, "") // all owned paths
	var unmanaged []manifest.UnmanagedFileRecord
	for _, tree := range fullScanTrees {
		onDiskTree := filepath.Join(root, tree)
		if _, serr := os.Stat(onDiskTree); serr != nil {
			continue
		}
		walkTree(onDiskTree, func(onDisk string, info os.FileInfo) {
			name := modelPath(root, onDisk)
			if owned[name] || excluded(name, opts.KeepList) {
				return
			}
			rec := recordFor(root, onDisk, info, "")
			if rec == nil {
				return
			}
			unmanaged = append(unmanaged, manifest.UnmanagedFileRecord{
				Name: rec.Name, Type: rec.Type, Mode: rec.Mode,
				User: rec.User, Group: rec.Group, SHA256: rec.SHA256, Target: rec.Target,
			})
		})
	}
	sort.SliceStable(unmanaged, func(i, j int) bool { return unmanaged[i].Name < unmanaged[j].Name })
	return changed, unmanaged, nil
}

// changedManagedFiles parses `rpm -Va` keeping non-config lines whose path is in
// the scanned trees (outside /etc).
func (r *Reader) changedManagedFiles(root string, keep map[string]bool) ([]manifest.ManagedBaselineRecord, error) {
	out, stderr, err := r.Runner.Run("rpm", rpmRootArgs(root, "-Va", "--nodeps", "--noscript"))
	if err != nil && strings.TrimSpace(out) == "" && strings.TrimSpace(stderr) != "" {
		return nil, errString(stderr)
	}
	var recs []manifest.ManagedBaselineRecord
	sc := bufio.NewScanner(strings.NewReader(out))
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		flags, typ, path, ok := parseVerifyLine(line)
		if !ok || typ == "c" { // config files belong to /etc handling
			continue
		}
		if !inFullScanTrees(path) || excluded(path, keep) {
			continue
		}
		rec := r.baselineRecord(root, path, flags)
		if rec != nil {
			recs = append(recs, *rec)
		}
	}
	sort.SliceStable(recs, func(i, j int) bool { return recs[i].Name < recs[j].Name })
	return recs, nil
}

// baselineRecord builds a ManagedBaselineRecord for a changed packaged path.
func (r *Reader) baselineRecord(root, path, flags string) *manifest.ManagedBaselineRecord {
	full := filepath.Join(root, path)
	info, err := os.Lstat(full)
	if err != nil {
		return nil
	}
	owner := r.ownerOf(root, path)
	mode, user, group := fileOwnerGroupMode(info)
	changes := decodeVerifyFlags(flags)
	switch {
	case info.Mode()&os.ModeSymlink != 0:
		target, lerr := os.Readlink(full)
		if lerr != nil {
			return nil
		}
		return &manifest.ManagedBaselineRecord{
			Name: path, Type: "link", Mode: mode, User: user, Group: group,
			SHA256: "", Target: target, PackageName: owner, Changes: changes,
		}
	case info.Mode().IsRegular():
		sum, herr := hashFile(full)
		if herr != nil {
			return nil
		}
		return &manifest.ManagedBaselineRecord{
			Name: path, Type: "file", Mode: mode, User: user, Group: group,
			SHA256: sum, Target: "", PackageName: owner, Changes: changes,
		}
	default:
		return nil
	}
}

// decodeVerifyFlags maps the 9-char rpm verify flag string (SM5DLUGTP) to a
// changes list naming what differs.
func decodeVerifyFlags(flags string) []string {
	if flags == "missing" {
		return []string{"missing"}
	}
	names := []string{"size", "mode", "sha256", "device", "target", "user", "group", "time", "caps"}
	var changes []string
	for i := 0; i < len(flags) && i < len(names); i++ {
		c := flags[i]
		if c != '.' && c != '?' && c != ' ' {
			changes = append(changes, names[i])
		}
	}
	return changes
}

func inFullScanTrees(path string) bool {
	for _, tree := range fullScanTrees {
		if path == tree || strings.HasPrefix(path, tree+"/") {
			return true
		}
	}
	return false
}

// modelPath converts an on-disk absolute path back to its model path (rooted at
// "/") for a non-/ root.
func modelPath(root, onDisk string) string {
	if root == "" || root == "/" {
		return onDisk
	}
	if rel, err := filepath.Rel(root, onDisk); err == nil {
		return "/" + filepath.ToSlash(rel)
	}
	return onDisk
}
