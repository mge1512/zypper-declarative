// generated from spec: zypper-declarative.spec.md sha256:714e75ff672557d2c7344736a5f36b52afa37e0f565b07de83e1b18cc4492014
//
// Human-readable rendering of the intent diff, drift report, and the apply
// summary, written to stdout. Field labels follow the spec Diff/DriftReport
// type names so the plan is unambiguous.
package cli

import (
	"fmt"

	"github.com/mge1512/zypper-declarative/internal/diff"
)

// printPlan renders the combined plan for diff (intent diff + drift report).
func (a *App) printPlan(intent diff.Diff, drift diff.DriftReport) {
	w := a.Stdout
	fmt.Fprintln(w, "Plan:")

	fmt.Fprintln(w, "  packages_install:")
	for _, p := range intent.PackagesInstall {
		fmt.Fprintf(w, "    - %s\n", pkgLabel(p.Name, p.Version, p.Release, p.Arch))
	}
	fmt.Fprintln(w, "  packages_remove:")
	for _, p := range intent.PackagesRemove {
		fmt.Fprintf(w, "    - %s\n", pkgLabel(p.Name, p.Version, p.Release, p.Arch))
	}
	fmt.Fprintln(w, "  repos_set:")
	for _, r := range intent.ReposSet {
		fmt.Fprintf(w, "    - %s (%s)\n", r.Alias, r.URL)
	}
	fmt.Fprintln(w, "  files to write:")
	for _, f := range intent.FilesWrite {
		fmt.Fprintf(w, "    - %s\n", f.Name)
	}
	fmt.Fprintln(w, "  files to delete:")
	for _, p := range intent.FilesDelete {
		fmt.Fprintf(w, "    - %s\n", p)
	}
	fmt.Fprintln(w, "  units_change:")
	for _, u := range intent.UnitsChange {
		fmt.Fprintf(w, "    - %s -> %s\n", u.Name, u.State)
	}

	fmt.Fprintln(w, "Drift:")
	if drift.Empty() {
		fmt.Fprintln(w, "  clean")
		return
	}
	for _, p := range drift.FilesModified {
		fmt.Fprintf(w, "  files_modified: %s\n", p)
	}
	for _, p := range drift.FilesExtra {
		fmt.Fprintf(w, "  files_extra: %s\n", p)
	}
	for _, u := range drift.UnitsDivergent {
		fmt.Fprintf(w, "  units_divergent: %s (%s)\n", u.Name, u.State)
	}
	for _, p := range drift.PackagesDivergent {
		fmt.Fprintf(w, "  packages_divergent: %s\n", p.Name)
	}
}

// printIntentSummary renders the change summary after a successful apply.
func (a *App) printIntentSummary(intent diff.Diff) {
	w := a.Stdout
	fmt.Fprintf(w, "convergence complete: %d package(s) installed, %d removed, %d file(s) written, %d deleted, %d unit(s) changed\n",
		len(intent.PackagesInstall), len(intent.PackagesRemove),
		len(intent.FilesWrite), len(intent.FilesDelete), len(intent.UnitsChange))
}

// emitDrift writes one diagnostic per drift item to stderr (verify).
func (a *App) emitDrift(drift diff.DriftReport) {
	w := a.Stderr
	for _, p := range drift.FilesModified {
		fmt.Fprintf(w, "Error [files] %s modified\n", p)
	}
	for _, p := range drift.FilesExtra {
		fmt.Fprintf(w, "Error [files] %s is extra (unpackaged, undeclared)\n", p)
	}
	for _, u := range drift.UnitsDivergent {
		fmt.Fprintf(w, "Error [units] %s diverges (declared %s)\n", u.Name, u.State)
	}
	for _, p := range drift.PackagesDivergent {
		fmt.Fprintf(w, "Error [packages] %s diverges\n", p.Name)
	}
}

func pkgLabel(name, version, release, arch string) string {
	if version == "" {
		return name
	}
	return fmt.Sprintf("%s-%s-%s.%s", name, version, release, arch)
}
