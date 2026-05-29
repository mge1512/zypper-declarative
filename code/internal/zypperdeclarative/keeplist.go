// generated from spec: zypper-declarative.spec.md sha256:714e75ff672557d2c7344736a5f36b52afa37e0f565b07de83e1b18cc4492014
//
// Keep-list handling: the allowlist of persistent-but-undeclared paths that
// describe-actual-state, compute-drift, and converge-files must never report
// or delete.
package zypperdeclarative

import (
	"bufio"
	"os"
	"strings"
)

// KeepList is the set of keep-listed absolute paths plus the always-excluded
// syncpoint.
type KeepList struct {
	paths map[string]bool
}

// LoadKeepList reads the keep-list file at path. An empty path yields a
// keep-list containing only the always-excluded syncpoint. A missing file is
// not an error (the keep-list is optional).
func LoadKeepList(path string) *KeepList {
	kl := &KeepList{paths: map[string]bool{Syncpoint: true}}
	if path == "" {
		return kl
	}
	f, err := os.Open(path)
	if err != nil {
		return kl
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		kl.paths[line] = true
	}
	return kl
}

// Has reports whether p is keep-listed (or the syncpoint).
func (k *KeepList) Has(p string) bool {
	if k == nil {
		return p == Syncpoint
	}
	return k.paths[p]
}
