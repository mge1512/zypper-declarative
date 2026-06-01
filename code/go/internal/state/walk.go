// generated from spec: zypper-declarative.spec.md sha256:27aee8e374eb3507189bad0b78339109d0116e6a13b55ae4df2ba9a18e769fc4
//
// Filesystem walk helpers shared by the /etc walk and the full-scan walk. Each
// entry is classified by its own type via lstat (symlinks are NOT followed):
// regular files are hashed, symlinks record their verbatim target, directories
// are traversed but not emitted, special files are skipped. A directory,
// symlink, or special file is never read as a file and never an error.
package state

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mge1512/zypper-declarative/internal/manifest"
)

func parseUint(s string) uint64 {
	n, _ := strconv.ParseUint(strings.TrimSpace(s), 10, 64)
	return n
}

func errString(s string) error { return fmt.Errorf("%s", strings.TrimSpace(s)) }

// walkTree recursively visits dir (an absolute on-disk path). For every regular
// file and symlink it invokes visit with the path AS IT SHOULD APPEAR in the
// model (the on-disk path; callers pass dir already rooted) and the lstat info.
// Directories are descended; special files are skipped. Errors reading a
// directory listing are silently skipped here (callers treat genuine access
// failures separately).
func walkTree(dir string, visit func(path string, info os.FileInfo)) {
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// Skip unreadable entries here; describe-actual-state's strict mode is
			// applied at the scope level by the caller.
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil // traverse, do not emit
		}
		info, lerr := os.Lstat(path)
		if lerr != nil {
			return nil
		}
		mode := info.Mode()
		switch {
		case mode&os.ModeSymlink != 0, mode.IsRegular():
			visit(path, info)
		default:
			// device, fifo, socket: skip
		}
		return nil
	})
}

// recordFor builds a ManagedFileRecord for an /etc path from its lstat info.
// path is the on-disk absolute path; the emitted name is the same path made
// relative to root only if root != "/". For the /etc walk the on-disk path IS
// the model path, so callers pass the absolute path already and we keep it.
func recordFor(root, onDisk string, info os.FileInfo, pkg string) *manifest.ManagedFileRecord {
	name := onDisk
	if root != "" && root != "/" {
		if rel, err := filepath.Rel(root, onDisk); err == nil {
			name = "/" + filepath.ToSlash(rel)
		}
	}
	mode, user, group := fileOwnerGroupMode(info)
	switch {
	case info.Mode()&os.ModeSymlink != 0:
		target, err := os.Readlink(onDisk)
		if err != nil {
			return nil
		}
		return &manifest.ManagedFileRecord{
			Name: name, Type: "link", Mode: mode, User: user, Group: group,
			SHA256: "", Target: target, ContentRef: "", PackageName: pkg,
		}
	case info.Mode().IsRegular():
		sum, err := hashFile(onDisk)
		if err != nil {
			return nil
		}
		return &manifest.ManagedFileRecord{
			Name: name, Type: "file", Mode: mode, User: user, Group: group,
			SHA256: sum, Target: "", ContentRef: "", PackageName: pkg,
		}
	default:
		return nil
	}
}
