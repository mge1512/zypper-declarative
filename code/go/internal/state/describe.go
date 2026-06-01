// generated from spec: zypper-declarative.spec.md sha256:27aee8e374eb3507189bad0b78339109d0116e6a13b55ae4df2ba9a18e769fc4
//
// Package state implements BEHAVIOR/INTERNAL: describe-actual-state, the single
// live-state reader. Every verb that needs actual state obtains it here (or via
// a supplied dump in the same format). No other package reads live system state.
//
// Reads are file-and-database level (no network refresh): rpm for packages and
// the config-file verdict, the on-disk zypp repos.d for repositories, systemctl
// for unit enablement, and a bounded /etc walk for config_files. Repositories
// are read as files, never via exec.
package state

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/mge1512/zypper-declarative/internal/manifest"
	"github.com/mge1512/zypper-declarative/internal/meta"
)

// ScanScope is etc (default, bounded to /etc) or full (adds out-of-/etc integrity).
type ScanScope string

const (
	ScopeEtc  ScanScope = "etc"
	ScopeFull ScanScope = "full"
)

// OnUnreadable selects strict-error vs warn-and-omit handling of an unreadable
// source.
type OnUnreadable string

const (
	OnUnreadableError OnUnreadable = "error"
	OnUnreadableWarn  OnUnreadable = "warn"
)

const syncpoint = "/etc/etc.syncpoint"

// Options controls a describe-actual-state read.
type Options struct {
	Root         string
	OnUnreadable OnUnreadable
	Scope        ScanScope
	KeepList     map[string]bool
	CreatedAt    string // RFC3339; informational
}

// Result is the output of Describe: a Manifest plus any warn diagnostics. On a
// strict unreadable source, Describe returns (nil, diags, error-diagnostic).
type Result struct {
	Manifest    *manifest.Manifest
	Diagnostics []manifest.Diagnostic
}

// Reader reads actual state by driving external tools through a CommandRunner.
type Reader struct {
	Runner CommandRunner
}

// NewReader returns a Reader using the production OSCommandRunner.
func NewReader() *Reader {
	return &Reader{Runner: &OSCommandRunner{}}
}

// Describe implements describe-actual-state. On a strict unreadable source it
// returns a non-nil error Diagnostic; under warn it omits the affected scope and
// records a diagnostic.
func (r *Reader) Describe(opts Options) (*manifest.Manifest, []manifest.Diagnostic, *manifest.Diagnostic) {
	if opts.Root == "" {
		opts.Root = "/"
	}
	if opts.Scope == "" {
		opts.Scope = ScopeEtc
	}
	if opts.OnUnreadable == "" {
		opts.OnUnreadable = OnUnreadableError
	}
	var diags []manifest.Diagnostic
	m := &manifest.Manifest{
		Meta: manifest.ManifestMeta{
			FormatVersion: 1,
			Generator:     meta.Generator(),
			CreatedAt:     opts.CreatedAt,
			DesiredSHA256: "",
		},
	}

	// 1. packages
	pkgs, unread, err := r.readPackages(opts.Root)
	if err != nil || unread {
		if d := r.handleUnreadable(opts, manifest.DomainPackages, "rpm package database", err); d != nil {
			return nil, diags, d
		}
		diags = append(diags, manifest.NewWarning(manifest.DomainPackages, "package database unreadable; scope omitted"))
	} else if len(pkgs) > 0 {
		m.Packages = &manifest.PackagesScope{
			Attributes: map[string]interface{}{"package_system": "rpm"},
			Elements:   pkgs,
		}
	}

	// 2. repositories (read from on-disk repos.d files; no exec)
	repos, unread, err := readRepositories(opts.Root)
	if err != nil || unread {
		if d := r.handleUnreadable(opts, manifest.DomainRepositories, filepath.Join(opts.Root, "etc/zypp/repos.d"), err); d != nil {
			return nil, diags, d
		}
		diags = append(diags, manifest.NewWarning(manifest.DomainRepositories, "repositories source unreadable; scope omitted"))
	} else if len(repos) > 0 {
		m.Repositories = &manifest.RepositoriesScope{
			Attributes: map[string]interface{}{"repository_system": "zypp"},
			Elements:   repos,
		}
	}

	// 3. services
	svcs, unread, err := r.readServices(opts.Root)
	if err != nil || unread {
		if d := r.handleUnreadable(opts, manifest.DomainUnits, "systemd unit enablement", err); d != nil {
			return nil, diags, d
		}
		diags = append(diags, manifest.NewWarning(manifest.DomainUnits, "unit enablement unreadable; scope omitted"))
	} else if len(svcs) > 0 {
		m.Services = &manifest.ServicesScope{
			Attributes: map[string]interface{}{"init_system": "systemd"},
			Elements:   svcs,
		}
	}

	// 4. config_files
	files, unread, err := r.readConfigFiles(opts)
	if err != nil || unread {
		if d := r.handleUnreadable(opts, manifest.DomainFiles, filepath.Join(opts.Root, "etc"), err); d != nil {
			return nil, diags, d
		}
		diags = append(diags, manifest.NewWarning(manifest.DomainFiles, "/etc unreadable; scope omitted"))
	} else if len(files) > 0 {
		m.ConfigFiles = &manifest.ConfigFilesScope{
			Attributes: map[string]interface{}{},
			Elements:   files,
		}
	}

	// 4a. full-scan integrity scopes
	if opts.Scope == ScopeFull {
		changed, unmanaged, ferr := r.readFullScan(opts)
		if ferr != nil {
			if d := r.handleUnreadable(opts, manifest.DomainFiles, "full-scan trees", ferr); d != nil {
				return nil, diags, d
			}
			diags = append(diags, manifest.NewWarning(manifest.DomainFiles, "full scan source unreadable; integrity scopes omitted"))
		} else {
			if len(changed) > 0 {
				m.ChangedManagedFiles = &manifest.ChangedManagedFilesScope{
					Attributes: map[string]interface{}{},
					Elements:   changed,
				}
			}
			if len(unmanaged) > 0 {
				m.UnmanagedFiles = &manifest.UnmanagedFilesScope{
					Attributes: map[string]interface{}{},
					Elements:   unmanaged,
				}
			}
		}
	}

	return m, diags, nil
}

// handleUnreadable returns a non-nil error Diagnostic under strict mode, or nil
// under warn (the caller then omits the scope and records a warning).
func (r *Reader) handleUnreadable(opts Options, domain, source string, err error) *manifest.Diagnostic {
	if opts.OnUnreadable == OnUnreadableWarn {
		return nil
	}
	msg := fmt.Sprintf("unreadable source: %s", source)
	if err != nil {
		msg = fmt.Sprintf("unreadable source: %s: %v", source, err)
	}
	d := manifest.NewError(domain, msg)
	return &d
}

// ---------------------------------------------------------------------------
// packages
// ---------------------------------------------------------------------------

// readPackages queries the rpmdb under root for the installed set with name,
// version, release, and arch populated. Returns (records, unreadable, error).
func (r *Reader) readPackages(root string) ([]manifest.PackageRecord, bool, error) {
	args := rpmRootArgs(root, "-qa", "--queryformat", "%{NAME} %{VERSION} %{RELEASE} %{ARCH}\n")
	stdout, stderr, err := r.Runner.Run("rpm", args)
	if err != nil && strings.TrimSpace(stdout) == "" && strings.TrimSpace(stderr) != "" {
		return nil, true, fmt.Errorf("%s", strings.TrimSpace(stderr))
	}
	if err != nil && strings.TrimSpace(stdout) == "" {
		// rpm not present at all: treat as unreadable source.
		return nil, true, err
	}
	var recs []manifest.PackageRecord
	sc := bufio.NewScanner(strings.NewReader(stdout))
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "(") || strings.HasPrefix(line, "error:") || strings.HasPrefix(line, "warning:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		recs = append(recs, manifest.PackageRecord{
			Name:    fields[0],
			Version: fields[1],
			Release: fields[2],
			Arch:    fields[3],
		})
	}
	sort.SliceStable(recs, func(i, j int) bool { return recs[i].Name < recs[j].Name })
	return recs, false, nil
}

// ---------------------------------------------------------------------------
// services
// ---------------------------------------------------------------------------

// readServices queries unit enablement under root and normalises to enabled,
// disabled, or masked. Purely-static units are omitted.
func (r *Reader) readServices(root string) ([]manifest.ServiceRecord, bool, error) {
	args := []string{"list-unit-files", "--no-legend", "--no-pager", "--plain"}
	if root != "" && root != "/" {
		args = append([]string{"--root", root}, args...)
	}
	stdout, stderr, err := r.Runner.Run("systemctl", args)
	if err != nil && strings.TrimSpace(stdout) == "" && strings.TrimSpace(stderr) != "" {
		return nil, true, fmt.Errorf("%s", strings.TrimSpace(stderr))
	}
	if err != nil && strings.TrimSpace(stdout) == "" {
		return nil, true, err
	}
	var recs []manifest.ServiceRecord
	sc := bufio.NewScanner(strings.NewReader(stdout))
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 2 {
			continue
		}
		name := fields[0]
		state := fields[1]
		if !isDeclarableUnitName(name) {
			continue
		}
		switch state {
		case "enabled", "enabled-runtime":
			recs = append(recs, manifest.ServiceRecord{Name: name, State: "enabled"})
		case "disabled":
			recs = append(recs, manifest.ServiceRecord{Name: name, State: "disabled"})
		case "masked", "masked-runtime":
			recs = append(recs, manifest.ServiceRecord{Name: name, State: "masked"})
		default:
			// static, generated, transient, indirect, etc.: not declarable; omit.
		}
	}
	sort.SliceStable(recs, func(i, j int) bool { return recs[i].Name < recs[j].Name })
	return recs, false, nil
}

func isDeclarableUnitName(name string) bool {
	for _, suf := range []string{".service", ".timer", ".socket", ".target", ".path", ".mount"} {
		if strings.HasSuffix(name, suf) {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// rpmRootArgs prepends --root for a non-/ root.
func rpmRootArgs(root string, rest ...string) []string {
	if root != "" && root != "/" {
		return append([]string{"--root", root}, rest...)
	}
	return rest
}

// hashFile returns the hex SHA256 of a regular file's content.
func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// fileOwnerGroupMode returns mode (octal string), user, group for a path using
// lstat info. Owner/group are numeric ids rendered as strings if name lookup is
// not available; for the declarable model the spec requires non-empty user and
// group, so we fall back to "root" only when the id is 0 and otherwise the
// numeric id string. This keeps the field non-empty as the schema requires.
func fileOwnerGroupMode(info fs.FileInfo) (mode, user, group string) {
	mode = "0" + strconv.FormatUint(uint64(info.Mode().Perm()), 8)
	if len(mode) == 2 { // e.g. "00"
		mode = "0000"
	}
	uid, gid := ownerIDs(info)
	user = lookupUser(uid)
	group = lookupGroup(gid)
	return mode, user, group
}
