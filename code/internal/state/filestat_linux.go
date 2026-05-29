// generated from spec: zypper-declarative.spec.md sha256:58e1636e2de82ab81a5cd3f81d6b3c9ac6a8976e18f9abb2bbd2b2aba56fe4d4
//
// Platform file-stat helper for describe-actual-state (Linux). Returns the
// file's octal mode and the owning user/group as strings.
package state

import (
	"fmt"
	"os"
	"os/user"
	"strconv"
	"syscall"
)

// fileStat returns the octal mode (e.g. "0644") and the owning user and group of
// the file at path. On any failure it returns conservative defaults.
func fileStat(path string) (mode, owner, group string) {
	mode, owner, group = "0644", "root", "root"
	info, err := os.Lstat(path)
	if err != nil {
		return
	}
	mode = fmt.Sprintf("0%o", info.Mode().Perm())
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return
	}
	owner = lookupUser(st.Uid)
	group = lookupGroup(st.Gid)
	return
}

func lookupUser(uid uint32) string {
	if u, err := user.LookupId(strconv.FormatUint(uint64(uid), 10)); err == nil && u.Username != "" {
		return u.Username
	}
	return strconv.FormatUint(uint64(uid), 10)
}

func lookupGroup(gid uint32) string {
	if g, err := user.LookupGroupId(strconv.FormatUint(uint64(gid), 10)); err == nil && g.Name != "" {
		return g.Name
	}
	return strconv.FormatUint(uint64(gid), 10)
}
