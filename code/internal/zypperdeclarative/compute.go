// generated from spec: zypper-declarative.spec.md sha256:714e75ff672557d2c7344736a5f36b52afa37e0f565b07de83e1b18cc4492014
//
// BEHAVIOR/INTERNAL: compute-intent-diff and compute-drift. Both are pure
// comparisons that perform no I/O.
package zypperdeclarative

// ComputeIntentDiff computes the changes from the previously applied
// declaration to the desired declaration, scope by scope. It does not consult
// the filesystem. A scope absent in desired produces no change for that scope.
func ComputeIntentDiff(desired *Manifest, applied *AppliedRecord) *Diff {
	d := &Diff{
		PackagesInstall: []PackageRecord{},
		PackagesRemove:  []PackageRecord{},
		ReposSet:        []RepositoryRecord{},
		FilesWrite:      []ManagedFileRecord{},
		FilesDelete:     []string{},
		UnitsChange:     []ServiceRecord{},
	}

	// Step 1: packages.
	if desired.Packages != nil {
		d.PackagesInstall = append(d.PackagesInstall, desired.Packages.Elements...)
		desiredNames := map[string]bool{}
		for _, p := range desired.Packages.Elements {
			desiredNames[p.Name] = true
		}
		if applied != nil && applied.Packages != nil {
			for _, p := range applied.Packages.Elements {
				if !desiredNames[p.Name] {
					d.PackagesRemove = append(d.PackagesRemove, p)
				}
			}
		}
	}

	// Step 2: repositories.
	if desired.Repositories != nil {
		d.ReposSet = append(d.ReposSet, desired.Repositories.Elements...)
	}

	// Step 3: config_files.
	if desired.ConfigFiles != nil {
		d.FilesWrite = append(d.FilesWrite, desired.ConfigFiles.Elements...)
		desiredFileNames := map[string]bool{}
		for _, f := range desired.ConfigFiles.Elements {
			desiredFileNames[f.Name] = true
		}
		if applied != nil && applied.ConfigFiles != nil {
			for _, f := range applied.ConfigFiles.Elements {
				if !desiredFileNames[f.Name] {
					d.FilesDelete = append(d.FilesDelete, f.Name)
				}
			}
		}
	}

	// Step 4: services.
	if desired.Services != nil {
		appliedStates := map[string]string{}
		if applied != nil && applied.Services != nil {
			for _, s := range applied.Services.Elements {
				appliedStates[s.Name] = s.State
			}
		}
		for _, s := range desired.Services.Elements {
			if prev, ok := appliedStates[s.Name]; !ok || prev != s.State {
				d.UnitsChange = append(d.UnitsChange, s)
			}
		}
	}

	return d
}

// ComputeDrift compares an actual-state Manifest against a declaration, scope
// by scope on identity fields, and reports divergence. Performs no I/O. The
// keep-list (and the syncpoint, which the keep-list always contains) is
// excluded from files_extra.
func ComputeDrift(actual *Manifest, reference *AppliedRecord, keep *KeepList) *DriftReport {
	r := &DriftReport{
		FilesModified:     []string{},
		FilesExtra:        []string{},
		UnitsDivergent:    []ServiceRecord{},
		PackagesDivergent: []PackageRecord{},
	}

	// Step 1: files_modified.
	actualFiles := map[string]ManagedFileRecord{}
	if actual != nil && actual.ConfigFiles != nil {
		for _, f := range actual.ConfigFiles.Elements {
			actualFiles[f.Name] = f
		}
	}
	refFiles := map[string]bool{}
	if reference != nil && reference.ConfigFiles != nil {
		for _, e := range reference.ConfigFiles.Elements {
			refFiles[e.Name] = true
			if af, ok := actualFiles[e.Name]; ok {
				if af.SHA256 != e.SHA256 {
					r.FilesModified = append(r.FilesModified, e.Name)
				}
			}
			// A declared file absent from actual is treated as matching.
		}
	}

	// Step 2: files_extra. Only unpackaged, undeclared, non-keep-listed,
	// non-syncpoint /etc files.
	for _, a := range orderedActualFiles(actual) {
		if refFiles[a.Name] {
			continue
		}
		if a.PackageName != "" {
			continue // package-owned files are not "extra"
		}
		if keep.Has(a.Name) || a.Name == Syncpoint {
			continue
		}
		r.FilesExtra = append(r.FilesExtra, a.Name)
	}

	// Step 3: units_divergent.
	actualUnits := map[string]string{}
	if actual != nil && actual.Services != nil {
		for _, s := range actual.Services.Elements {
			actualUnits[s.Name] = s.State
		}
	}
	if reference != nil && reference.Services != nil {
		for _, u := range reference.Services.Elements {
			if state, ok := actualUnits[u.Name]; ok && state != u.State {
				r.UnitsDivergent = append(r.UnitsDivergent, u)
			} else if !ok {
				// reference declares a unit the actual state does not report:
				// the declared state differs from the (absent) actual state.
				r.UnitsDivergent = append(r.UnitsDivergent, u)
			}
		}
	}

	// Step 4: packages_divergent. Any package present in one but not the other
	// (identity comparison on name + version + release + arch).
	actualPkgs := map[string]PackageRecord{}
	if actual != nil && actual.Packages != nil {
		for _, p := range actual.Packages.Elements {
			actualPkgs[pkgKey(p)] = p
		}
	}
	refPkgs := map[string]PackageRecord{}
	if reference != nil && reference.Packages != nil {
		for _, p := range reference.Packages.Elements {
			refPkgs[pkgKey(p)] = p
		}
	}
	for k, p := range refPkgs {
		if _, ok := actualPkgs[k]; !ok {
			r.PackagesDivergent = append(r.PackagesDivergent, p)
		}
	}
	for k, p := range actualPkgs {
		if _, ok := refPkgs[k]; !ok {
			r.PackagesDivergent = append(r.PackagesDivergent, p)
		}
	}

	return r
}

func orderedActualFiles(actual *Manifest) []ManagedFileRecord {
	if actual == nil || actual.ConfigFiles == nil {
		return nil
	}
	return actual.ConfigFiles.Elements
}

func pkgKey(p PackageRecord) string {
	return p.Name + "\x00" + p.Version + "\x00" + p.Release + "\x00" + p.Arch
}
