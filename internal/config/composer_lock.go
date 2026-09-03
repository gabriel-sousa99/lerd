package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// lockedPackage is the part of a composer.lock entry a check cares about: what
// the package is called, which version resolved, and the names it stands in for.
type lockedPackage struct {
	Name    string            `json:"name"`
	Version string            `json:"version"`
	Replace map[string]string `json:"replace"`
	Provide map[string]string `json:"provide"`
}

type composerLockEntry struct {
	modTime  time.Time
	size     int64
	packages map[string]string
}

var (
	composerLockMu    sync.Mutex
	composerLockCache = map[string]composerLockEntry{}
)

// lockPackages returns what composer installed for a project, package name to
// the version that resolved, and nil for a project with no lock. A name another
// package replaces or provides is listed too, at the version of the package
// standing in for it: that is the only place a name like tempest/database
// appears, since composer writes one entry for the framework replacing it.
func lockPackages(dir string) map[string]string {
	path := filepath.Join(dir, "composer.lock")
	st, err := os.Stat(path)
	if err != nil {
		return nil
	}

	composerLockMu.Lock()
	defer composerLockMu.Unlock()
	if e, ok := composerLockCache[path]; ok && e.modTime.Equal(st.ModTime()) && e.size == st.Size() {
		return e.packages
	}

	data, err := composerReadFile(path)
	if err != nil {
		return nil
	}
	var lock struct {
		Packages    []lockedPackage `json:"packages"`
		PackagesDev []lockedPackage `json:"packages-dev"`
	}
	if json.Unmarshal(data, &lock) != nil {
		return nil
	}

	pkgs := make(map[string]string, len(lock.Packages)+len(lock.PackagesDev))
	for _, entry := range append(lock.Packages, lock.PackagesDev...) {
		if entry.Name == "" {
			continue
		}
		pkgs[entry.Name] = entry.Version
	}
	// Second pass, so a package that is really installed keeps its own version
	// rather than the one a later entry claims to replace it at.
	for _, entry := range append(lock.Packages, lock.PackagesDev...) {
		for _, stood := range []map[string]string{entry.Replace, entry.Provide} {
			for name, version := range stood {
				if _, real := pkgs[name]; real {
					continue
				}
				if version == "self.version" {
					version = entry.Version
				}
				pkgs[name] = version
			}
		}
	}

	composerLockCache[path] = composerLockEntry{modTime: st.ModTime(), size: st.Size(), packages: pkgs}
	return pkgs
}

// ComposerHasInstalled reports whether a project has pkg available to run: named
// in its composer.json, or installed underneath it. The manifest alone answers
// what the project asked for, which misses a dependency it never named itself
// and one that arrives under the package replacing it, and what a check is
// really asking is whether the code is there.
func ComposerHasInstalled(dir, pkg string, extraSections ...string) bool {
	if ComposerHasPackage(dir, pkg, extraSections...) {
		return true
	}
	_, ok := lockPackages(dir)[pkg]
	return ok
}

// DetectPackageMajor is the major of a composer package a project has: the
// version composer resolved for it, and failing that the constraint the manifest
// declares, which is all a project that has never been installed can offer. A
// constraint is a claim about what would be installed, and a poor one where it
// spans two majors, so the lock answers first wherever there is one.
//
// versionKey and extraSections belong to the manifest fallback: a dot-path to
// read when the constraint carries no digits, and the sections to look in beyond
// require and require-dev.
func DetectPackageMajor(dir, pkg, versionKey string, extraSections ...string) string {
	if major := lockedMajor(dir, pkg); major != "" {
		return major
	}
	return detectVersionFromComposer(dir, []FrameworkRule{{
		Composer:         pkg,
		VersionKey:       versionKey,
		ComposerSections: extraSections,
	}})
}

// lockedMajor is the major composer resolved for pkg, empty when the project has
// no lock or the lock does not carry the package.
func lockedMajor(dir, pkg string) string {
	if v := lockPackages(dir)[pkg]; v != "" {
		return extractMajorFromConstraint(v)
	}
	return ""
}

// installedMajor is DetectPackageMajor for a package named plainly, which is how
// the package layer's own files are named.
func installedMajor(dir, pkg string) string {
	return DetectPackageMajor(dir, pkg, "")
}
