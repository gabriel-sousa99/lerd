package config

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// A worker's identity on the dashboard, like a framework's in framework_icon.go,
// comes out of the store rather than out of Go. The difference is that a mark
// belongs to the worker rather than to the framework running it: Laravel and
// Tempest both run Vite, and one drawing serves both, so the marks are cached
// under workers/<icon>.svg and keyed by the icon name each definition declares.

// StoreWorkerIconsDir is where worker marks are cached, beside the framework
// definitions that name them.
func StoreWorkerIconsDir() string {
	return filepath.Join(StoreFrameworksDir(), "workers")
}

// WorkerMark is how one worker of one framework asks to be drawn: the glyph or
// mark name it declares, and the tone to ink it in.
type WorkerMark struct {
	Icon  string `json:"icon,omitempty"`
	Color string `json:"color,omitempty"`
}

// WorkerMarkSet is the whole answer for the dashboard: what each worker asks
// for, keyed "<framework>/<worker>", and the drawings themselves, keyed by icon
// name so one mark serves every framework that names it.
type WorkerMarkSet struct {
	Workers map[string]WorkerMark `json:"workers"`
	Marks   map[string]string     `json:"marks"`
}

// SaveStoreWorkerIcon caches a worker mark fetched from the framework store.
func SaveStoreWorkerIcon(icon string, data []byte) error {
	return saveStoreIcon(StoreWorkerIconsDir(), icon, data)
}

// WorkerIcon returns a cached worker mark, if the store shipped one.
func WorkerIcon(icon string) (string, bool) {
	return readStoreIcon(StoreWorkerIconsDir(), icon)
}

// WorkerMarks reads every cached framework definition and reports how its
// workers want to be drawn, so lerd-ui can hand the dashboard the whole set in
// one response and the browser never reaches the store itself. A worker that
// declares neither an icon nor a colour, and whose framework declares no colour
// either, is absent: there is nothing to say about it that a plain glyph does
// not already say.
func WorkerMarks() WorkerMarkSet {
	out := WorkerMarkSet{Workers: map[string]WorkerMark{}, Marks: map[string]string{}}
	for _, name := range frameworkNamesInCache() {
		fwColor := NormalizeBrandColor(cachedFrameworkColor(name))
		for worker, w := range cachedFrameworkWorkers(name) {
			color := NormalizeBrandColor(w.Color)
			if color == "" {
				color = fwColor
			}
			if w.Icon == "" && color == "" {
				continue
			}
			out.Workers[name+"/"+worker] = WorkerMark{Icon: w.Icon, Color: color}
		}
	}
	for _, icon := range cachedWorkerIconNames() {
		if svg, ok := WorkerIcon(icon); ok {
			out.Marks[icon] = svg
		}
	}
	return out
}

// cachedFrameworkWorkers reads the worker block off any cached definition for a
// framework. Workers are declared per version, and the newest file that has
// them answers, so a framework whose latest release added one still reports it.
func cachedFrameworkWorkers(name string) map[string]FrameworkWorker {
	paths := append(versionedFrameworkPaths(name), filepath.Join(StoreFrameworksDir(), name+".yaml"))
	for _, p := range paths {
		if fw := loadFrameworkYAML(p); fw != nil && len(fw.Workers) > 0 {
			return fw.Workers
		}
	}
	return nil
}

// cachedWorkerIconNames lists the marks the store has handed this install.
func cachedWorkerIconNames() []string {
	entries, err := os.ReadDir(StoreWorkerIconsDir())
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".svg") {
			continue
		}
		if n := strings.TrimSuffix(e.Name(), ".svg"); validPresetName(n) {
			out = append(out, n)
		}
	}
	sort.Strings(out)
	return out
}
