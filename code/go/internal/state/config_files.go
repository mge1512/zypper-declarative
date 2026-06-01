// generated from spec: zypper-declarative.spec.md sha256:27aee8e374eb3507189bad0b78339109d0116e6a13b55ae4df2ba9a18e769fc4
//
// config_files actual state: the changed-from-package and unpackaged /etc files
// and symlinks. Per the decisions hints, the METHOD is verdict-parse: let rpm
// do the comparison (rpm -V) rather than building a self-recorded baseline map.
//
//  1. CHANGED config files: the owning packages of /etc config files come from
//     `rpm -qca --queryformat '%{NAME}\n'`; for each package `rpm -V` reports
//     differences (non-zero exit is normal). Keep verify lines whose type char
//     is `c`. Emit the on-disk type: a changed regular file is type "file" with
//     its real sha256; an `L` flag on a package-recorded file is the
//     type-mismatch case, emitted as type "link" with the verbatim on-disk
//     target.
//  2. CONTENT-BEARING GHOSTS: the one case rpm -V skips. Enumerate ghost-flagged
//     /etc paths; emit those with real on-disk content as type "file".
//  3. UNPACKAGED files: /etc paths no package owns, found by walking /etc and
//     subtracting the rpm-owned path set.
//
// Exclusions: keep-list and /etc/etc.syncpoint. Bounded to /etc; content_ref "".
package state

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mge1512/zypper-declarative/internal/manifest"
)

// readConfigFiles returns the config_files records, an unreadable flag, and an
// error. A readable but genuinely-empty /etc (no changed/unpackaged files)
// yields an empty slice with unreadable=false (the caller omits the scope).
func (r *Reader) readConfigFiles(opts Options) ([]manifest.ManagedFileRecord, bool, error) {
	root := opts.Root
	etc := filepath.Join(root, "etc")
	if _, err := os.Stat(etc); err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, true, err
	}

	byPath := map[string]manifest.ManagedFileRecord{}

	// 1. changed config files via rpm -V verdict-parse.
	changed, unread, err := r.changedConfigFiles(root)
	if err != nil || unread {
		return nil, unread, err
	}
	for _, rec := range changed {
		if excluded(rec.Name, opts.KeepList) {
			continue
		}
		byPath[rec.Name] = rec
	}

	// 2. content-bearing ghosts under /etc.
	ghosts, gerr := r.ghostConfigFiles(root)
	if gerr == nil {
		for _, rec := range ghosts {
			if excluded(rec.Name, opts.KeepList) {
				continue
			}
			byPath[rec.Name] = rec
		}
	}

	// 3. unpackaged /etc files (walk and subtract the rpm-owned set).
	unpkg, uerr := r.unpackagedConfigFiles(root, etc, opts.KeepList)
	if uerr == nil {
		for _, rec := range unpkg {
			if _, present := byPath[rec.Name]; present {
				continue
			}
			byPath[rec.Name] = rec
		}
	}

	var out []manifest.ManagedFileRecord
	for _, rec := range byPath {
		out = append(out, rec)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, false, nil
}

// changedConfigFiles drives rpm -qca then rpm -V per owning package.
func (r *Reader) changedConfigFiles(root string) ([]manifest.ManagedFileRecord, bool, error) {
	// Owning packages of config files.
	stdout, stderr, err := r.Runner.Run("rpm", rpmRootArgs(root, "-qca", "--queryformat", "%{NAME}\n"))
	if err != nil && strings.TrimSpace(stdout) == "" && strings.TrimSpace(stderr) != "" {
		return nil, true, errString(stderr)
	}
	if err != nil && strings.TrimSpace(stdout) == "" {
		return nil, true, err
	}
	pkgs := dedupePackages(stdout)

	var recs []manifest.ManagedFileRecord
	for _, pkg := range pkgs {
		vout, verr, verr2 := r.Runner.Run("rpm", rpmRootArgs(root, "-V", "--nodeps", "--noscript", pkg))
		// rpm -V exits non-zero to report differences: that is NORMAL. Treat as a
		// package error only when stdout is empty AND stderr is non-empty.
		if strings.TrimSpace(vout) == "" && strings.TrimSpace(verr) != "" {
			// genuine failure for this package; skip rather than abort the scope.
			_ = verr2
			continue
		}
		recs = append(recs, parseVerifyOutput(root, vout, pkg)...)
	}
	return recs, false, nil
}

// parseVerifyOutput parses `rpm -V` output, keeping config-file ('c') lines.
// Each line is "<9 flag chars><space><type><space><path>" or "missing ...".
func parseVerifyOutput(root, out, pkg string) []manifest.ManagedFileRecord {
	var recs []manifest.ManagedFileRecord
	sc := bufio.NewScanner(strings.NewReader(out))
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		flags, typ, path, ok := parseVerifyLine(line)
		if !ok {
			continue
		}
		if typ != "c" { // config files only here
			continue
		}
		if path == syncpoint {
			continue
		}
		rec := classifyChanged(root, path, flags, pkg)
		if rec != nil {
			recs = append(recs, *rec)
		}
	}
	return recs
}

// parseVerifyLine extracts (flags, typeChar, path). A "missing" line is reported
// with flags="missing".
func parseVerifyLine(line string) (flags, typ, path string, ok bool) {
	trimmed := strings.TrimLeft(line, " ")
	if strings.HasPrefix(trimmed, "missing") {
		rest := strings.TrimSpace(strings.TrimPrefix(trimmed, "missing"))
		// "missing   c /etc/foo" or "missing /etc/foo"
		fields := strings.Fields(rest)
		if len(fields) == 0 {
			return "", "", "", false
		}
		if len(fields) >= 2 && len(fields[0]) == 1 {
			return "missing", fields[0], strings.Join(fields[1:], " "), true
		}
		return "missing", "", strings.Join(fields, " "), true
	}
	// Standard line: first 9 chars are flags. Then a space, then attr/type, path.
	if len(line) < 11 {
		return "", "", "", false
	}
	flags = line[:9]
	rest := strings.TrimLeft(line[9:], " ")
	// rest may be "c /etc/foo" (config) or "/usr/bin/foo" (no attr marker).
	fields := strings.SplitN(rest, " ", 2)
	if len(fields) == 2 && len(fields[0]) == 1 && isAttrChar(fields[0][0]) {
		return flags, fields[0], strings.TrimSpace(fields[1]), true
	}
	return flags, "", strings.TrimSpace(rest), true
}

func isAttrChar(c byte) bool {
	switch c {
	case 'c', 'd', 'g', 'l', 'r': // config, doc, ghost, license, readme
		return true
	}
	return false
}

// classifyChanged builds a ManagedFileRecord for a changed config path. The `L`
// flag means the link differs from what the package recorded: combined with an
// on-disk symlink, this is the type-mismatch case (emit type "link"). A changed
// regular file is type "file" with its real sha256.
func classifyChanged(root, path, flags, pkg string) *manifest.ManagedFileRecord {
	full := filepath.Join(root, path)
	info, err := os.Lstat(full)
	if err != nil {
		// "missing": the file is deleted on disk. A fresh install would lay it
		// down, so its absence is a change; but we cannot emit on-disk content.
		// Per the model we emit a record only for present paths; skip missing.
		return nil
	}
	mode, user, group := fileOwnerGroupMode(info)
	switch {
	case info.Mode()&os.ModeSymlink != 0:
		target, lerr := os.Readlink(full)
		if lerr != nil {
			return nil
		}
		return &manifest.ManagedFileRecord{
			Name: path, Type: "link", Mode: mode, User: user, Group: group,
			SHA256: "", Target: target, ContentRef: "", PackageName: pkg,
		}
	case info.Mode().IsRegular():
		sum, herr := hashFile(full)
		if herr != nil {
			return nil
		}
		return &manifest.ManagedFileRecord{
			Name: path, Type: "file", Mode: mode, User: user, Group: group,
			SHA256: sum, Target: "", ContentRef: "", PackageName: pkg,
		}
	default:
		return nil // special files are skipped
	}
}

// ghostConfigFiles enumerates ghost-flagged /etc paths and emits those with real
// on-disk content as type "file".
func (r *Reader) ghostConfigFiles(root string) ([]manifest.ManagedFileRecord, error) {
	// FILEFLAGS bit 64 (0x40) marks %ghost. Query all installed packages' file
	// lists with their flags, keep /etc ghost paths.
	out, _, err := r.Runner.Run("rpm", rpmRootArgs(root, "-qa", "--queryformat", "[%{FILENAMES} %{FILEFLAGS}\n]"))
	if err != nil && strings.TrimSpace(out) == "" {
		return nil, err
	}
	var recs []manifest.ManagedFileRecord
	sc := bufio.NewScanner(strings.NewReader(out))
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	seen := map[string]bool{}
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 2 {
			continue
		}
		path := fields[0]
		if !strings.HasPrefix(path, "/etc/") || seen[path] {
			continue
		}
		flags := parseUint(fields[len(fields)-1])
		if flags&0x40 == 0 { // not a ghost
			continue
		}
		seen[path] = true
		full := filepath.Join(root, path)
		info, ierr := os.Lstat(full)
		if ierr != nil || !info.Mode().IsRegular() || info.Size() == 0 {
			continue // empty ghost or absent: suppressed
		}
		sum, herr := hashFile(full)
		if herr != nil {
			continue
		}
		mode, user, group := fileOwnerGroupMode(info)
		owner := r.ownerOf(root, path)
		recs = append(recs, manifest.ManagedFileRecord{
			Name: path, Type: "file", Mode: mode, User: user, Group: group,
			SHA256: sum, Target: "", ContentRef: "", PackageName: owner,
		})
	}
	return recs, nil
}

// ownerOf returns the bare owning package name for a path, or "" if unpackaged.
func (r *Reader) ownerOf(root, path string) string {
	out, _, _ := r.Runner.Run("rpm", rpmRootArgs(root, "-qf", "--queryformat", "%{NAME}", path))
	o := strings.TrimSpace(out)
	if o == "" || strings.Contains(o, "not owned") || strings.HasPrefix(o, "error") {
		return ""
	}
	return o
}

// unpackagedConfigFiles walks /etc and emits paths the rpm-owned set does not
// contain (so a file is never marked unpackaged because a lookup was skipped).
func (r *Reader) unpackagedConfigFiles(root, etc string, keep map[string]bool) ([]manifest.ManagedFileRecord, error) {
	owned := r.ownedPaths(root, "/etc/")
	var recs []manifest.ManagedFileRecord
	walkTree(etc, func(onDisk string, info os.FileInfo) {
		name := modelPath(root, onDisk)
		if owned[name] || excluded(name, keep) {
			return
		}
		rec := recordFor(root, onDisk, info, "")
		if rec != nil {
			recs = append(recs, *rec)
		}
	})
	return recs, nil
}

// ownedPaths returns the set of rpm-owned file paths beginning with prefix.
func (r *Reader) ownedPaths(root, prefix string) map[string]bool {
	owned := map[string]bool{}
	out, _, err := r.Runner.Run("rpm", rpmRootArgs(root, "-qa", "--queryformat", "[%{FILENAMES}\n]"))
	if err != nil && strings.TrimSpace(out) == "" {
		return owned
	}
	sc := bufio.NewScanner(strings.NewReader(out))
	sc.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	for sc.Scan() {
		p := strings.TrimSpace(sc.Text())
		if p != "" && (prefix == "" || strings.HasPrefix(p, prefix)) {
			owned[p] = true
		}
	}
	return owned
}

func excluded(path string, keep map[string]bool) bool {
	if path == syncpoint {
		return true
	}
	return keep != nil && keep[path]
}

func dedupePackages(out string) []string {
	seen := map[string]bool{}
	var pkgs []string
	sc := bufio.NewScanner(strings.NewReader(out))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "(") || strings.HasPrefix(line, "error:") || strings.HasPrefix(line, "warning:") {
			continue
		}
		if !seen[line] {
			seen[line] = true
			pkgs = append(pkgs, line)
		}
	}
	sort.Strings(pkgs)
	return pkgs
}
