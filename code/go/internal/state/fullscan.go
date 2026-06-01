// generated from spec: zypper-declarative.spec.md sha256:b2d0de88fbed1163678e59e931c741b9d999b71f902f6eb01db8790bb813d057
//
// Full-scan integrity (scope=full): scans the package-managed OS trees outside
// /etc for changed packaged files and unpackaged additions, emitting the two
// observational scopes. Expensive and opt-in.
package state

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/mge1512/zypper-declarative/internal/manifest"
	"github.com/mge1512/zypper-declarative/internal/sysexec"
)

// scannedTrees are the package-managed trees scanned under scope=full:
// /usr, the usr-merge compatibility roots, and /boot. /etc, /opt, and the
// virtual/runtime/mutable-data trees are excluded.
var scannedTrees = []string{"/usr", "/bin", "/sbin", "/lib", "/lib64", "/boot"}

// fullScan returns the changed_managed_files and unmanaged_files scopes. A
// required path that genuinely cannot be read is returned as an error.
func fullScan(root string, r sysexec.CommandRunner, keep map[string]bool) (*manifest.ChangedManagedFilesScope, *manifest.UnmanagedFilesScope, error) {
	changed := &manifest.ChangedManagedFilesScope{
		Attributes: map[string]interface{}{},
		Elements:   []manifest.ManagedBaselineRecord{},
	}
	unmanaged := &manifest.UnmanagedFilesScope{
		Attributes: map[string]interface{}{},
		Elements:   []manifest.UnmanagedFileRecord{},
	}

	owned := ownedPathSet(root, r)

	for _, tree := range scannedTrees {
		base := filepath.Join(root, strings.TrimPrefix(tree, "/"))
		info, err := os.Lstat(base)
		if err != nil {
			if os.IsNotExist(err) {
				continue // tree absent on this system; skip
			}
			return nil, nil, err
		}
		// Do not descend into separate filesystem mounts other than the named
		// ones: we only walk the named trees themselves.
		rootDev := deviceOf(info)

		werr := filepath.WalkDir(base, func(path string, d fs.DirEntry, e error) error {
			if e != nil {
				return e
			}
			if d.IsDir() {
				if di, ierr := d.Info(); ierr == nil && deviceOf(di) != rootDev && path != base {
					return filepath.SkipDir // crossed into another mount
				}
				return nil
			}
			rel := "/" + strings.TrimPrefix(strings.TrimPrefix(path, root), "/")
			if keep[rel] {
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
			sum := ""
			if ftype == "file" {
				if h, herr := hashFile(path); herr == nil {
					sum = h
				}
			}
			if owned[rel] {
				if pkg, ch := packagedOutsideEtcChanged(root, rel, r); ch {
					changed.Elements = append(changed.Elements, manifest.ManagedBaselineRecord{
						Name:        rel,
						Type:        ftype,
						Mode:        modeString(fi.Mode()),
						User:        "root",
						Group:       "root",
						SHA256:      sum,
						PackageName: pkg,
						Changes:     []string{"sha256"},
					})
				}
			} else {
				unmanaged.Elements = append(unmanaged.Elements, manifest.UnmanagedFileRecord{
					Name:   rel,
					Type:   ftype,
					Mode:   modeString(fi.Mode()),
					User:   "root",
					Group:  "root",
					SHA256: sum,
				})
			}
			return nil
		})
		if werr != nil {
			return nil, nil, werr
		}
	}
	return changed, unmanaged, nil
}

// ownedPathSet returns the set of file paths owned by any installed package.
func ownedPathSet(root string, r sysexec.CommandRunner) map[string]bool {
	out := map[string]bool{}
	if r == nil {
		return out
	}
	args := []string{"-qal"}
	if root != "" && root != "/" {
		args = append([]string{"--root", root}, args...)
	}
	stdout, _, _ := r.Run("rpm", args)
	for _, line := range strings.Split(stdout, "\n") {
		p := strings.TrimSpace(line)
		if p != "" {
			out[p] = true
		}
	}
	return out
}

func packagedOutsideEtcChanged(root, rel string, r sysexec.CommandRunner) (string, bool) {
	pkg := owningPackage(root, rel, r)
	if pkg == "" {
		return "", false
	}
	return pkg, packagedFileChanged(root, rel, r)
}
