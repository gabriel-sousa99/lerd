package config

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// The framework store's half of the mark handling; the sanitiser and the cache
// read and write it sits on are in store_icon.go, shared with the service store.
//
// A framework's mark is per family, not per version: Laravel 11 and Laravel 12
// are the same logo, so one <name>.svg sits beside the <name>@<version>.yaml
// files rather than inside them. Nothing is embedded here the way the default
// service stack is, because no framework definition ships in the binary at all.

// FrameworkMark is a framework's identity as the dashboard needs it: the
// silhouette and the colour to paint it in. Either half may be empty, and a
// framework with neither keeps rendering as its label alone.
type FrameworkMark struct {
	SVG   string `json:"svg,omitempty"`
	Color string `json:"color,omitempty"`
}

// SaveStoreFrameworkIcon caches an icon fetched from the framework store beside
// the definition's YAML.
func SaveStoreFrameworkIcon(name string, data []byte) error {
	return saveStoreIcon(StoreFrameworksDir(), name, data)
}

// FrameworkIcon returns a framework's mark, if the store shipped one.
func FrameworkIcon(name string) (string, bool) {
	return readStoreIcon(StoreFrameworksDir(), name)
}

// FrameworkMarks returns the mark and brand colour of every framework this
// install has a cached definition for, keyed by framework name, so lerd-ui can
// hand the dashboard the whole set in one response and the browser never
// reaches the store origin itself. A framework whose definition declares a
// colour but ships no mark still appears, since the colour alone tints its
// label.
func FrameworkMarks() map[string]FrameworkMark {
	out := make(map[string]FrameworkMark)
	for _, name := range frameworkNamesInCache() {
		mark := FrameworkMark{Color: NormalizeBrandColor(cachedFrameworkColor(name))}
		if svg, ok := FrameworkIcon(name); ok {
			mark.SVG = svg
		}
		if mark.SVG != "" || mark.Color != "" {
			out[name] = mark
		}
	}
	return out
}

// frameworkNamesInCache lists the framework names the store cache holds, from
// both the versioned <name>@<version>.yaml files and any unversioned <name>.yaml,
// plus any name that has a mark but whose definition has since been pruned.
func frameworkNamesInCache() []string {
	entries, err := os.ReadDir(StoreFrameworksDir())
	if err != nil {
		return nil
	}
	seen := map[string]bool{}
	var out []string
	add := func(name string) {
		if name == "" || seen[name] || !validPresetName(name) {
			return
		}
		seen[name] = true
		out = append(out, name)
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		switch {
		case strings.HasSuffix(e.Name(), ".yaml"):
			name := strings.TrimSuffix(e.Name(), ".yaml")
			add(strings.SplitN(name, "@", 2)[0])
		case strings.HasSuffix(e.Name(), ".svg"):
			add(strings.TrimSuffix(e.Name(), ".svg"))
		}
	}
	sort.Strings(out)
	return out
}

// cachedFrameworkColor reads the brand colour off any cached definition for a
// framework. The colour is a property of the family rather than of one release,
// so every version file carries the same value and the first one found answers.
func cachedFrameworkColor(name string) string {
	paths, _ := filepath.Glob(filepath.Join(StoreFrameworksDir(), name+"@*.yaml"))
	paths = append(paths, filepath.Join(StoreFrameworksDir(), name+".yaml"))
	sort.Strings(paths)
	for _, p := range paths {
		if fw := loadFrameworkYAML(p); fw != nil && fw.Color != "" {
			return fw.Color
		}
	}
	return ""
}
