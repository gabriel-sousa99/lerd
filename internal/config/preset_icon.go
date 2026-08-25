package config

import (
	"os"
)

// The service store's half of the mark handling; the sanitiser and the cache
// read and write it sits on are in store_icon.go, shared with the framework
// store.

// SaveStorePresetIcon caches an icon fetched from the service store beside the
// preset's YAML.
func SaveStorePresetIcon(name string, data []byte) error {
	return saveStoreIcon(StorePresetsDir(), name, data)
}

// PresetIcon returns a preset's own mark, if it has one. It layers the same way
// the YAML seam does: the store cache first, so a published mark supersedes the
// shipped one, then the embedded bundle. The embed matters for the default
// stack, which is never fetched from the store and so would otherwise have no
// way to carry a mark at all. A preset with none falls back to the built-in
// glyph its YAML names.
func PresetIcon(name string) (string, bool) {
	if svg, ok := readStoreIcon(StorePresetsDir(), name); ok {
		return svg, true
	}
	if !validPresetName(name) {
		return "", false
	}
	data, err := presetFS.ReadFile("presets/" + name + ".svg")
	if err != nil {
		return "", false
	}
	return string(data), true
}

// PresetIcons returns every mark either layer can serve, keyed by preset name,
// so lerd-ui can hand the dashboard the whole set in one response and the
// browser never reaches the store origin itself.
func PresetIcons() map[string]string {
	out := make(map[string]string)
	for _, name := range iconNames(presetFS, "presets") {
		if svg, ok := PresetIcon(name); ok {
			out[name] = svg
		}
	}
	for _, name := range iconNames(os.DirFS(StorePresetsDir()), ".") {
		if svg, ok := PresetIcon(name); ok {
			out[name] = svg
		}
	}
	return out
}

// removeStorePresetIcon drops a preset's cached icon, called wherever the cached
// YAML goes so the two never outlive each other.
func removeStorePresetIcon(name string) error {
	return removeStoreIcon(StorePresetsDir(), name)
}
