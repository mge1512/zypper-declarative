// generated from spec: zypper-declarative.spec.md sha256:714e75ff672557d2c7344736a5f36b52afa37e0f565b07de83e1b18cc4492014
//
// Shared helpers for the CLI verb layer: version/hash accessors, keep-list
// loading, and the actual-state acquisition seam (the single live reader).
package cli

import (
	"bufio"
	"os"
	"strings"

	"github.com/mge1512/zypper-declarative/internal/manifest"
	"github.com/mge1512/zypper-declarative/internal/meta"
	"github.com/mge1512/zypper-declarative/internal/state"
)

func versionString() string { return meta.Version }
func specHash() string      { return meta.SpecSHA256 }

// loadKeepList reads the keep-list allowlist file, one path per line, ignoring
// blanks and # comments. A missing path means an empty keep-list.
func (a *App) loadKeepList(path string) ([]string, map[string]bool) {
	set := map[string]bool{}
	var list []string
	if path == "" {
		return list, set
	}
	f, err := os.Open(path)
	if err != nil {
		return list, set
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		list = append(list, line)
		set[line] = true
	}
	return list, set
}

// actualState obtains the actual state via the single live reader, or returns
// the supplied state dump parsed as a Manifest. This is the only place verbs
// reach live system state.
func (a *App) actualState(cfg Config, root string, keepList []string) (manifest.Manifest, error) {
	m, d := state.Describe(root, a.Runner, keepList)
	if d != nil {
		return manifest.Manifest{}, d
	}
	return m, nil
}
