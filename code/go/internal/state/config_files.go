// generated from spec: zypper-declarative.spec.md sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
//
// config_files reading: the changed-from-package and unpackaged /etc files.
// Method (per the Go decisions hints): let rpm decide via `rpm -V` verdict-parse,
// plus a separate ghost-file pass and an unpackaged-file pass; never build a
// self-maintained recorded-baseline map.
package state

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/mge1512/zypper-declarative/internal/manifest"
)

// flagToChange maps an rpm -V flag character to a change-reason name.
var flagToChange = map[byte]string{
	'S': "size", 'M': "mode", '5': "md5", 'D': "device",
	'L': "link_path", 'U': "user", 'G': "group", 'T': "time", 'P': "caps",
}

// readConfigFiles assembles the config_files scope: the UNION of the rpm -V
// verdict parse, the ghost-content pass, and the unpackaged-file pass, all
// bounded to /etc, minus the keep-list and /etc/etc.syncpoint.
func readConfigFiles(opts Options) (*manifest.ScopeWrapper[manifest.ManagedFileRecord], []Diagnostic, error) {
	etc := filepath.Join(opts.Root, "etc")
	scope := manifest.NewScope[manifest.ManagedFileRecord](map[string]interface{}{})
	var diags []Diagnostic

	if _, err := os.Stat(etc); err != nil {
		if os.IsNotExist(err) {
			return &scope, diags, nil // readable, genuinely empty
		}
		return nil, diags, &Diagnostic{Severity: "Error", Domain: "files", Message: "/etc unreadable: " + err.Error()}
	}

	excluded := func(path string) bool {
		if path == "/etc/etc.syncpoint" {
			return true
		}
		return opts.KeepList[path]
	}

	emitted := map[string]bool{}
	add := func(rec manifest.ManagedFileRecord) {
		if excluded(rec.Name) || emitted[rec.Name] {
			return
		}
		emitted[rec.Name] = true
		scope.Elements = append(scope.Elements, rec)
	}

	// (1) CHANGED config files via rpm -V verdict parse.
	changed, err := readChangedConfigFiles(opts)
	if err != nil {
		return nil, diags, err
	}
	for _, rec := range changed {
		add(rec)
	}

	// (2) GHOST content files: a separate, required pass (rpm -V omits %ghost).
	ghosts, gdiags := readGhostFiles(opts)
	diags = append(diags, gdiags...)
	for _, rec := range ghosts {
		add(rec)
	}

	// (3) UNPACKAGED /etc files: walk /etc, subtract the rpm-owned set.
	unpkg, udiags, err := readUnpackagedEtc(opts, emitted)
	if err != nil {
		return nil, diags, err
	}
	diags = append(diags, udiags...)
	for _, rec := range unpkg {
		add(rec)
	}

	// Content store population for emitted regular-file records.
	if opts.ContentStore != "" {
		cdiags, err := populateContentStore(opts, scope.Elements)
		if err != nil {
			return nil, diags, err
		}
		diags = append(diags, cdiags...)
	}

	return &scope, diags, nil
}

// readChangedConfigFiles runs rpm -V on the config-owning package set and parses
// the verdict. Non-zero rpm exit when differences exist is normal.
func readChangedConfigFiles(opts Options) ([]manifest.ManagedFileRecord, error) {
	// Config-file owning packages.
	qargs := []string{"-qca", "--queryformat", "%{NAME}\n"}
	qargs = append(qargs, dbpath(opts.Root)...)
	stdout, stderr, err := opts.Runner.Run("rpm", qargs)
	if err != nil && strings.TrimSpace(stdout) == "" && strings.TrimSpace(stderr) != "" {
		return nil, &Diagnostic{Severity: "Error", Domain: "files", Message: "rpm -qca failed: " + strings.TrimSpace(stderr)}
	}
	pkgSet := map[string]bool{}
	for _, line := range strings.Split(stdout, "\n") {
		l := strings.TrimSpace(line)
		if l == "" || strings.HasPrefix(l, "(") || strings.HasPrefix(l, "error:") || strings.HasPrefix(l, "warning:") {
			continue
		}
		pkgSet[l] = true
	}

	var recs []manifest.ManagedFileRecord
	for pkg := range pkgSet {
		vargs := []string{"-V", "--nodeps", "--noscript"}
		vargs = append(vargs, dbpath(opts.Root)...)
		vargs = append(vargs, pkg)
		vout, verr, _ := opts.Runner.Run("rpm", vargs)
		// rpm -V exits non-zero when it reports differences: parse regardless.
		// Treat as a package error ONLY when stdout is empty AND stderr non-empty.
		if strings.TrimSpace(vout) == "" && strings.TrimSpace(verr) != "" {
			continue // skip this package; not a fatal /etc read failure
		}
		recs = append(recs, parseVerifyOutput(opts, vout, pkg, true)...)
	}
	return recs, nil
}

// parseVerifyOutput parses `rpm -V` output lines. configOnly keeps only lines
// whose type char is 'c' (a config file).
func parseVerifyOutput(opts Options, out, pkg string, configOnly bool) []manifest.ManagedFileRecord {
	var recs []manifest.ManagedFileRecord
	for _, raw := range strings.Split(out, "\n") {
		line := strings.TrimRight(raw, "\r")
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "missing") {
			// A deleted file. Format: "missing     c /etc/foo"
			fields := strings.Fields(line)
			if len(fields) < 2 {
				continue
			}
			path := fields[len(fields)-1]
			typeChar := ""
			if len(fields) >= 3 {
				typeChar = fields[len(fields)-2]
			}
			if configOnly && typeChar != "c" {
				continue
			}
			recs = append(recs, deletedRecord(opts, path, pkg))
			continue
		}
		// Standard line: <9 flag chars><space><optional type char><space><path>
		if len(line) < 11 {
			continue
		}
		flags := line[:9]
		rest := strings.TrimSpace(line[9:])
		typeChar := ""
		path := rest
		parts := strings.Fields(rest)
		if len(parts) >= 2 && len(parts[0]) == 1 {
			typeChar = parts[0]
			path = strings.Join(parts[1:], " ")
		}
		if configOnly && typeChar != "c" {
			continue
		}
		rec := buildChangedRecord(opts, path, pkg, flags)
		if rec != nil {
			recs = append(recs, *rec)
		}
	}
	return recs
}

// buildChangedRecord builds a config_files record from an rpm -V flag string,
// emitting the on-disk type (regular file or, for an L mismatch, a link).
func buildChangedRecord(opts Options, path, pkg, flags string) *manifest.ManagedFileRecord {
	abs := filepath.Join(opts.Root, path)
	info, err := os.Lstat(abs)
	if err != nil {
		// File reported changed but not stat-able now; skip (transient).
		return nil
	}
	changes := changesFromFlags(flags)
	if info.Mode()&os.ModeSymlink != 0 {
		// On-disk is a symlink (the type-mismatch / link case).
		target, _ := os.Readlink(abs)
		return &manifest.ManagedFileRecord{
			Name: path, Type: "link", Mode: "0777", User: "root", Group: "root",
			SHA256: "", Target: target, ContentRef: "", PackageName: pkg,
			Status: "changed", Changes: ensureChanges(changes, "link_path"),
		}
	}
	if info.Mode().IsDir() {
		return nil // a directory verify line is not a config file we emit
	}
	if !info.Mode().IsRegular() {
		return nil // special file
	}
	sum, err := hashFile(abs)
	if err != nil {
		return nil
	}
	mode, user, group := fileMeta(info)
	return &manifest.ManagedFileRecord{
		Name: path, Type: "file", Mode: mode, User: user, Group: group,
		SHA256: sum, Target: "", ContentRef: "", PackageName: pkg,
		Status: "changed", Changes: ensureChanges(changes, "md5"),
	}
}

func deletedRecord(opts Options, path, pkg string) manifest.ManagedFileRecord {
	return manifest.ManagedFileRecord{
		Name: path, Type: "file", Mode: "0000", User: "root", Group: "root",
		SHA256: "", Target: "", ContentRef: "", PackageName: pkg,
		Status: "changed", Changes: []string{"deleted"},
	}
}

// changesFromFlags converts the 9-char flag string into a change-reason list.
func changesFromFlags(flags string) []string {
	var out []string
	for i := 0; i < len(flags); i++ {
		c := flags[i]
		if c == '.' || c == '?' {
			continue
		}
		if name, ok := flagToChange[c]; ok {
			out = append(out, name)
		}
	}
	return out
}

func ensureChanges(changes []string, fallback string) []string {
	if len(changes) == 0 {
		return []string{fallback}
	}
	return changes
}

// readGhostFiles enumerates ghost-flagged /etc paths and emits content-bearing
// ghost regular files (rpm -V never reports %ghost). An empty ghost is suppressed.
func readGhostFiles(opts Options) ([]manifest.ManagedFileRecord, []Diagnostic) {
	var recs []manifest.ManagedFileRecord
	var diags []Diagnostic
	// Enumerate ghost paths under /etc: query all installed packages' file lists
	// with their flags; FILEFLAGS bit 64 (0x40) is the GHOST marker.
	qargs := []string{"-qa", "--queryformat", "[%{FILENAMES} %{FILEFLAGS}\n]"}
	qargs = append(qargs, dbpath(opts.Root)...)
	stdout, _, _ := opts.Runner.Run("rpm", qargs)
	seen := map[string]bool{}
	for _, line := range strings.Split(stdout, "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) < 2 {
			continue
		}
		path := fields[0]
		if !strings.HasPrefix(path, "/etc/") {
			continue
		}
		flagsVal := fields[len(fields)-1]
		if !ghostBit(flagsVal) {
			continue
		}
		if seen[path] {
			continue
		}
		seen[path] = true
		abs := filepath.Join(opts.Root, path)
		info, err := os.Lstat(abs)
		if err != nil {
			continue // ghost not present on disk -> nothing to capture
		}
		if info.Mode()&os.ModeSymlink != 0 {
			// Ghost symlinks (alternatives) handled by their own rule; the
			// common content-bearing ghost case is regular files.
			rec, ok := ghostSymlinkRecord(opts, path, abs)
			if ok {
				recs = append(recs, rec)
			}
			continue
		}
		if !info.Mode().IsRegular() || info.Size() == 0 {
			continue // empty ghost suppressed; non-regular skipped
		}
		sum, err := hashFile(abs)
		if err != nil {
			if opts.OnUnreadable == OnUnreadableError {
				diags = append(diags, Diagnostic{Severity: "Error", Domain: "files", Message: "ghost file unreadable: " + path})
			} else {
				diags = append(diags, Diagnostic{Severity: "Warning", Domain: "files", Message: "ghost file content unreadable: " + path})
			}
			continue
		}
		mode, user, group := fileMeta(info)
		pkg := ownerOf(opts, path)
		recs = append(recs, manifest.ManagedFileRecord{
			Name: path, Type: "file", Mode: mode, User: user, Group: group,
			SHA256: sum, Target: "", ContentRef: "", PackageName: pkg,
			Status: "changed", Changes: []string{"ghost_content"},
		})
	}
	return recs, diags
}

// ghostSymlinkRecord judges a ghost symlink (alternatives): suppress when the
// on-disk target equals the auto/best target; emit otherwise.
func ghostSymlinkRecord(opts Options, path, abs string) (manifest.ManagedFileRecord, bool) {
	target, err := os.Readlink(abs)
	if err != nil {
		return manifest.ManagedFileRecord{}, false
	}
	name := filepath.Base(path)
	best := autoBestAlternative(opts, name)
	if best != "" && best == target {
		return manifest.ManagedFileRecord{}, false // pristine, suppress
	}
	if best == "" {
		// Cannot determine the reproducible target: emit (its target is not
		// known to be reproducible), per the spec's conservative rule.
	}
	pkg := ownerOf(opts, path)
	return manifest.ManagedFileRecord{
		Name: path, Type: "link", Mode: "0777", User: "root", Group: "root",
		SHA256: "", Target: target, ContentRef: "", PackageName: pkg,
		Status: "changed", Changes: []string{"link_path"},
	}, true
}

// autoBestAlternative queries the alternatives DB for the auto/best target.
func autoBestAlternative(opts Options, name string) string {
	out, _, err := opts.Runner.Run("update-alternatives", []string{"--query", name})
	if err != nil && strings.TrimSpace(out) == "" {
		return ""
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "Best:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "Best:"))
		}
		if strings.HasPrefix(line, "Value:") {
			// fall through; Best preferred
		}
	}
	return ""
}

// ghostBit reports whether an rpm FILEFLAGS value has the GHOST bit (0x40).
func ghostBit(v string) bool {
	n := atoiSafe(v)
	return n&0x40 != 0
}

func atoiSafe(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return n
		}
		n = n*10 + int(c-'0')
	}
	return n
}

// ownerOf returns the bare owning package name for a path ("" if unpackaged).
func ownerOf(opts Options, path string) string {
	args := []string{"-qf", "--queryformat", "%{NAME}\n", path}
	args = append(dbpath(opts.Root), args...)
	out, _, err := opts.Runner.Run("rpm", args)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(out, "\n") {
		l := strings.TrimSpace(line)
		if l == "" || strings.Contains(l, "not owned") || strings.HasPrefix(l, "error:") {
			return ""
		}
		return l
	}
	return ""
}

// readUnpackagedEtc walks /etc and emits files that no package owns.
func readUnpackagedEtc(opts Options, already map[string]bool) ([]manifest.ManagedFileRecord, []Diagnostic, error) {
	owned := rpmOwnedSet(opts)
	var recs []manifest.ManagedFileRecord
	var diags []Diagnostic
	etc := filepath.Join(opts.Root, "etc")

	err := filepath.WalkDir(etc, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			if opts.OnUnreadable == OnUnreadableError {
				return &Diagnostic{Severity: "Error", Domain: "files", Message: "/etc walk failed at " + p + ": " + err.Error()}
			}
			diags = append(diags, Diagnostic{Severity: "Warning", Domain: "files", Message: "skipping unreadable: " + p})
			return nil
		}
		rel := "/" + strings.TrimPrefix(strings.TrimPrefix(p, opts.Root), "/")
		if d.IsDir() {
			return nil // traverse, emit nothing
		}
		info, ierr := d.Info()
		if ierr != nil {
			return nil
		}
		// Classify by own type (lstat semantics via DirEntry).
		if info.Mode()&os.ModeSymlink != 0 {
			if owned[rel] || already[rel] {
				return nil
			}
			if rel == "/etc/etc.syncpoint" || opts.KeepList[rel] {
				return nil
			}
			target, _ := os.Readlink(p)
			recs = append(recs, manifest.ManagedFileRecord{
				Name: rel, Type: "link", Mode: "0777", User: "root", Group: "root",
				SHA256: "", Target: target, ContentRef: "", PackageName: "",
			})
			return nil
		}
		if !info.Mode().IsRegular() {
			return nil // special file: skip
		}
		if owned[rel] || already[rel] {
			return nil
		}
		if rel == "/etc/etc.syncpoint" || opts.KeepList[rel] {
			return nil
		}
		sum, herr := hashFile(p)
		if herr != nil {
			if opts.OnUnreadable == OnUnreadableError {
				return &Diagnostic{Severity: "Error", Domain: "files", Message: "/etc file unreadable: " + rel}
			}
			diags = append(diags, Diagnostic{Severity: "Warning", Domain: "files", Message: "/etc file unreadable: " + rel})
			return nil
		}
		mode, user, group := fileMeta(info)
		recs = append(recs, manifest.ManagedFileRecord{
			Name: rel, Type: "file", Mode: mode, User: user, Group: group,
			SHA256: sum, Target: "", ContentRef: "", PackageName: "",
		})
		return nil
	})
	if err != nil {
		if d, ok := err.(*Diagnostic); ok {
			return nil, diags, d
		}
		return nil, diags, &Diagnostic{Severity: "Error", Domain: "files", Message: err.Error()}
	}
	return recs, diags, nil
}

// rpmOwnedSet returns the set of /etc paths owned by some installed package.
func rpmOwnedSet(opts Options) map[string]bool {
	owned := map[string]bool{}
	args := []string{"-qal"}
	args = append(dbpath(opts.Root), args...)
	out, _, _ := opts.Runner.Run("rpm", args)
	for _, line := range strings.Split(out, "\n") {
		l := strings.TrimSpace(line)
		if strings.HasPrefix(l, "/etc/") {
			owned[l] = true
		}
	}
	return owned
}

// populateContentStore writes emitted regular-file bytes into the content store,
// content-addressed by sha256, and sets each record's content_ref.
func populateContentStore(opts Options, elems []manifest.ManagedFileRecord) ([]Diagnostic, error) {
	var diags []Diagnostic
	for i := range elems {
		e := &elems[i]
		if e.Type != "file" || e.SHA256 == "" {
			continue
		}
		blobDir := filepath.Join(opts.ContentStore, "sha256")
		if err := os.MkdirAll(blobDir, 0o755); err != nil {
			return diags, &Diagnostic{Severity: "Error", Domain: "files", Message: "content store unwritable: " + err.Error()}
		}
		blob := filepath.Join(blobDir, e.SHA256)
		if _, err := os.Stat(blob); err == nil {
			e.ContentRef = "sha256/" + e.SHA256 // dedup: already present
			continue
		}
		abs := filepath.Join(opts.Root, e.Name)
		data, err := os.ReadFile(abs)
		if err != nil {
			if opts.OnUnreadable == OnUnreadableError {
				return diags, &Diagnostic{Severity: "Error", Domain: "files", Message: "content unreadable: " + e.Name}
			}
			diags = append(diags, Diagnostic{Severity: "Warning", Domain: "files", Message: "content unreadable, content_ref left empty: " + e.Name})
			continue
		}
		if err := os.WriteFile(blob, data, 0o644); err != nil {
			return diags, &Diagnostic{Severity: "Error", Domain: "files", Message: "content store write failed: " + err.Error()}
		}
		e.ContentRef = "sha256/" + e.SHA256
	}
	return diags, nil
}
