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

// installedMajor is the major of pkg this project has, taken from the lock
// composer resolved and falling back to the constraint the manifest declares,
// which is all a project that has never been installed can offer.
func installedMajor(dir, pkg string) string {
	if v := lockPackages(dir)[pkg]; v != "" {
		if major := extractMajorFromConstraint(v); major != "" {
			return major
		}
	}
	return detectVersionFromComposer(dir, []FrameworkRule{{Composer: pkg}})
}
