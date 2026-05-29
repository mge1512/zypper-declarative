// generated from spec: zypper-declarative.spec.md sha256:b2d0de88fbed1163678e59e931c741b9d999b71f902f6eb01db8790bb813d057
//
// Platform helpers for the live-state reader. Linux only (PLATFORM: Linux).
package state

import (
	"os"
	"syscall"
	"time"
)

func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// deviceOf returns the device id for a FileInfo, used to avoid crossing into
// separate filesystem mounts during the full scan.
func deviceOf(fi os.FileInfo) uint64 {
	if st, ok := fi.Sys().(*syscall.Stat_t); ok {
		return uint64(st.Dev)
	}
	return 0
}
