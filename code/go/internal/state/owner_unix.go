// generated from spec: zypper-declarative.spec.md sha256:27aee8e374eb3507189bad0b78339109d0116e6a13b55ae4df2ba9a18e769fc4
//
// Unix ownership extraction for describe-actual-state records.
package state

import (
	"io/fs"
	"os/user"
	"strconv"
	"syscall"
)

// ownerIDs extracts uid/gid from the underlying stat structure.
func ownerIDs(info fs.FileInfo) (uid, gid uint32) {
	if st, ok := info.Sys().(*syscall.Stat_t); ok {
		return st.Uid, st.Gid
	}
	return 0, 0
}

func lookupUser(uid uint32) string {
	if u, err := user.LookupId(strconv.FormatUint(uint64(uid), 10)); err == nil && u.Username != "" {
		return u.Username
	}
	if uid == 0 {
		return "root"
	}
	return strconv.FormatUint(uint64(uid), 10)
}

func lookupGroup(gid uint32) string {
	if g, err := user.LookupGroupId(strconv.FormatUint(uint64(gid), 10)); err == nil && g.Name != "" {
		return g.Name
	}
	if gid == 0 {
		return "root"
	}
	return strconv.FormatUint(uint64(gid), 10)
}
