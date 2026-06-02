// generated from spec: zypper-declarative.spec.md sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
//
// full-scan integrity (scope=full): scan the package-managed OS trees outside
// /etc and emit the changed_managed_files and unmanaged_files observational scopes.
package state

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/mge1512/zypper-declarative/internal/manifest"
)

// fullScanTrees are the package-managed trees scanned under scope=full.
var fullScanTrees = []string{"usr", "bin", "sbin", "lib", "lib64", "boot"}

// readFullScan emits changed_managed_files (packaged files differing from
// baseline) and unmanaged_files (files no package owns) for the scanned trees.
func readFullScan(opts Options) (*manifest.ScopeWrapper[manifest.ManagedBaselineRecord], *manifest.ScopeWrapper[manifest.UnmanagedFileRecord], error) {
	cm := manifest.NewScope[manifest.ManagedBaselineRecord](map[string]interface{}{})
	um := manifest.NewScope[manifest.UnmanagedFileRecord](map[string]interface{}{})

	// changed_managed_files: rpm -Va verdict parse, keeping NON-config lines.
	changed, err := readChangedManaged(opts)
	if err != nil {
		return nil, nil, err
	}
	cm.Elements = append(cm.Elements, changed...)

	// unmanaged_files: walk the trees, subtract the rpm-owned set.
	owned := rpmOwnedAll(opts)
	for _, tree := range fullScanTrees {
		base := filepath.Join(opts.Root, tree)
		info, err := os.Lstat(base)
		if err != nil || info.Mode()&os.ModeSymlink != 0 {
			continue // absent or a usr-merge compat symlink: skip
		}
		err = filepath.WalkDir(base, func(p string, d os.DirEntry, werr error) error {
			if werr != nil {
				if opts.OnUnreadable == OnUnreadableError {
					return &Diagnostic{Severity: "Error", Domain: "files", Message: "full-scan walk failed at " + p}
				}
				return nil
			}
			rel := "/" + strings.TrimPrefix(strings.TrimPrefix(p, opts.Root), "/")
			if d.IsDir() {
				return nil
			}
			fi, ierr := d.Info()
			if ierr != nil {
				return nil
			}
			if owned[rel] || opts.KeepList[rel] {
				return nil
			}
			if fi.Mode()&os.ModeSymlink != 0 {
				target, _ := os.Readlink(p)
				um.Elements = append(um.Elements, manifest.UnmanagedFileRecord{
					Name: rel, Type: "link", Mode: "0777", User: "root", Group: "root", SHA256: "", Target: target,
				})
				return nil
			}
			if !fi.Mode().IsRegular() {
				return nil
			}
			sum, herr := hashFile(p)
			if herr != nil {
				return nil
			}
			mode, user, group := fileMeta(fi)
			um.Elements = append(um.Elements, manifest.UnmanagedFileRecord{
				Name: rel, Type: "file", Mode: mode, User: user, Group: group, SHA256: sum, Target: "",
			})
			return nil
		})
		if err != nil {
			if d, ok := err.(*Diagnostic); ok {
				return nil, nil, d
			}
		}
	}

	return &cm, &um, nil
}

// readChangedManaged runs rpm -Va and parses the NON-config lines into
// ManagedBaselineRecords for the scanned trees.
func readChangedManaged(opts Options) ([]manifest.ManagedBaselineRecord, error) {
	args := []string{"-Va", "--nodeps", "--noscript"}
	args = append(dbpath(opts.Root), args...)
	out, errOut, _ := opts.Runner.Run("rpm", args)
	if strings.TrimSpace(out) == "" && strings.TrimSpace(errOut) != "" {
		// rpm -Va producing only stderr with no verdict is treated as a soft
		// failure of the integrity scan source.
		if opts.OnUnreadable == OnUnreadableError {
			return nil, &Diagnostic{Severity: "Error", Domain: "files", Message: "rpm -Va failed: " + strings.TrimSpace(errOut)}
		}
		return nil, nil
	}
	var recs []manifest.ManagedBaselineRecord
	for _, raw := range strings.Split(out, "\n") {
		line := strings.TrimRight(raw, "\r")
		if line == "" || len(line) < 11 {
			continue
		}
		flags := line[:9]
		rest := strings.TrimSpace(line[9:])
		parts := strings.Fields(rest)
		typeChar := ""
		path := rest
		if len(parts) >= 2 && len(parts[0]) == 1 {
			typeChar = parts[0]
			path = strings.Join(parts[1:], " ")
		}
		if typeChar == "c" {
			continue // config files belong to config_files, not here
		}
		if !inFullScanTrees(path) {
			continue
		}
		if opts.KeepList[path] {
			continue
		}
		abs := filepath.Join(opts.Root, path)
		info, err := os.Lstat(abs)
		if err != nil {
			continue
		}
		pkg := ownerOf(opts, path)
		changes := changesFromFlags(flags)
		if info.Mode()&os.ModeSymlink != 0 {
			target, _ := os.Readlink(abs)
			recs = append(recs, manifest.ManagedBaselineRecord{
				Name: path, Type: "link", Mode: "0777", User: "root", Group: "root",
				SHA256: "", Target: target, PackageName: pkg, Changes: ensureChanges(changes, "link_path"),
			})
			continue
		}
		if !info.Mode().IsRegular() {
			continue
		}
		sum, herr := hashFile(abs)
		if herr != nil {
			continue
		}
		mode, user, group := fileMeta(info)
		recs = append(recs, manifest.ManagedBaselineRecord{
			Name: path, Type: "file", Mode: mode, User: user, Group: group,
			SHA256: sum, Target: "", PackageName: pkg, Changes: ensureChanges(changes, "md5"),
		})
	}
	return recs, nil
}

func inFullScanTrees(path string) bool {
	for _, t := range fullScanTrees {
		if strings.HasPrefix(path, "/"+t+"/") || path == "/"+t {
			return true
		}
	}
	return false
}

// rpmOwnedAll returns the set of all paths owned by an installed package.
func rpmOwnedAll(opts Options) map[string]bool {
	owned := map[string]bool{}
	args := []string{"-qal"}
	args = append(dbpath(opts.Root), args...)
	out, _, _ := opts.Runner.Run("rpm", args)
	for _, line := range strings.Split(out, "\n") {
		l := strings.TrimSpace(line)
		if strings.HasPrefix(l, "/") {
			owned[l] = true
		}
	}
	return owned
}
