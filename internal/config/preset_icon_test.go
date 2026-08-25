package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSanitizeStoreIcon_KeepsTheDrawingAndDropsItsColours(t *testing.T) {
	raw := `<?xml version="1.0"?><svg xmlns="http://www.w3.org/2000/svg" width="128" height="128" viewBox="0 0 24 24" fill="#00758f" class="brand" id="mysql">
	  <g transform="translate(1 1)" style="fill:red">
	    <path d="M4 4h16v16H4z" fill="#ff0000" stroke="blue" fill-rule="evenodd"/>
	    <circle cx="12" cy="12" r="5"/>
	  </g>
	</svg>`
	got, err := SanitizeStoreIcon([]byte(raw))
	if err != nil {
		t.Fatalf("SanitizeStoreIcon: %v", err)
	}
	out := string(got)
	for _, want := range []string{`viewBox="0 0 24 24"`, `d="M4 4h16v16H4z"`, `fill-rule="evenodd"`, `transform="translate(1 1)"`, `<circle`, `r="5"`} {
		if !strings.Contains(out, want) {
			t.Errorf("sanitized icon lost %s: %s", want, out)
		}
	}
	for _, bad := range []string{"fill=\"#", "stroke=", "style=", "class=", "id=", "width=", "height="} {
		if strings.Contains(out, bad) {
			t.Errorf("sanitized icon kept %s: %s", bad, out)
		}
	}
}

func TestSanitizeStoreIcon_DropsScriptExternalRefsAndHandlers(t *testing.T) {
	raw := `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" viewBox="0 0 24 24" onload="alert(1)">
	  <script>alert(2)</script>
	  <foreignObject><div>hi</div></foreignObject>
	  <image xlink:href="https://evil.test/x.png"/>
	  <use href="#x"/>
	  <path d="M1 1h2v2H1z" onclick="alert(3)"/>
	</svg>`
	got, err := SanitizeStoreIcon([]byte(raw))
	if err != nil {
		t.Fatalf("SanitizeStoreIcon: %v", err)
	}
	out := string(got)
	if !strings.Contains(out, `d="M1 1h2v2H1z"`) {
		t.Fatalf("sanitized icon lost its only path: %s", out)
	}
	for _, bad := range []string{"script", "alert", "foreignObject", "href", "evil.test", "onload", "onclick", "<use", "<image", "<div"} {
		if strings.Contains(out, bad) {
			t.Errorf("sanitized icon kept %q: %s", bad, out)
		}
	}
}

func TestSanitizeStoreIcon_DerivesAViewBoxFromWidthAndHeight(t *testing.T) {
	raw := `<svg xmlns="http://www.w3.org/2000/svg" width="48" height="48"><path d="M0 0h1v1H0z"/></svg>`
	got, err := SanitizeStoreIcon([]byte(raw))
	if err != nil {
		t.Fatalf("SanitizeStoreIcon: %v", err)
	}
	if !strings.Contains(string(got), `viewBox="0 0 48 48"`) {
		t.Errorf("width/height should become a viewBox, got %s", got)
	}
}

func TestSanitizeStoreIcon_RejectsWhatIsNotADrawing(t *testing.T) {
	cases := map[string]string{
		"not svg":       `<html><body>hi</body></html>`,
		"no drawing":    `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><script>x</script></svg>`,
		"no view box":   `<svg xmlns="http://www.w3.org/2000/svg"><path d="M0 0h1v1H0z"/></svg>`,
		"not xml":       `just text`,
		"empty":         ``,
		"unknown enity": `<svg viewBox="0 0 24 24"><path d="&xxe;"/></svg>`,
	}
	for name, raw := range cases {
		if _, err := SanitizeStoreIcon([]byte(raw)); err == nil {
			t.Errorf("%s: expected a rejection, got none", name)
		}
	}
}

func TestSanitizeStoreIcon_RejectsAnOversizedIcon(t *testing.T) {
	raw := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path d="` +
		strings.Repeat("M0 0h1v1H0z", maxIconBytes/10) + `"/></svg>`
	if _, err := SanitizeStoreIcon([]byte(raw)); err == nil {
		t.Error("an icon past the size cap should be rejected")
	}
}

func TestNormalizeBrandColor(t *testing.T) {
	ok := map[string]string{
		"#00758F": "#00758f",
		"#abc":    "#abc",
		" #ABC ":  "#abc",
	}
	for in, want := range ok {
		if got := NormalizeBrandColor(in); got != want {
			t.Errorf("NormalizeBrandColor(%q) = %q, want %q", in, got, want)
		}
	}
	for _, in := range []string{"", "red", "rgb(1,2,3)", "#12", "#1234567", "url(#x)", "#00758g", "var(--x)"} {
		if got := NormalizeBrandColor(in); got != "" {
			t.Errorf("NormalizeBrandColor(%q) = %q, want it dropped", in, got)
		}
	}
}

func TestSaveStorePresetIcon_SanitizesOnTheWayInAndTheSeamServesIt(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	raw := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="#f00"><path d="M2 2h4v4H2z"/><script>bad()</script></svg>`
	if err := SaveStorePresetIcon("demo", []byte(raw)); err != nil {
		t.Fatalf("SaveStorePresetIcon: %v", err)
	}
	got, ok := PresetIcon("demo")
	if !ok {
		t.Fatal("the seam did not serve the saved icon")
	}
	if strings.Contains(got, "bad()") || strings.Contains(got, `fill="#f00"`) {
		t.Errorf("the cached icon was not sanitized: %s", got)
	}
	if !strings.Contains(got, `d="M2 2h4v4H2z"`) {
		t.Errorf("the cached icon lost its path: %s", got)
	}
	if icons := PresetIcons(); icons["demo"] != got {
		t.Errorf("StorePresetIcons should list demo, got %v", icons)
	}
}

func TestSaveStorePresetIcon_RejectsAnUnusableIconRatherThanCachingIt(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	if err := SaveStorePresetIcon("demo", []byte(`<svg onload="x"/>`)); err == nil {
		t.Fatal("expected an unusable icon to be rejected")
	}
	if _, ok := PresetIcon("demo"); ok {
		t.Error("a rejected icon must not land in the cache")
	}
}

func TestRemoveStorePreset_TakesTheIconWithIt(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	if err := SaveStorePresetIcon("demo", []byte(`<svg viewBox="0 0 24 24"><path d="M0 0h1v1H0z"/></svg>`)); err != nil {
		t.Fatalf("SaveStorePresetIcon: %v", err)
	}
	if err := RemoveStorePreset("demo"); err != nil {
		t.Fatalf("RemoveStorePreset: %v", err)
	}
	if _, ok := PresetIcon("demo"); ok {
		t.Error("removing a store preset should drop its icon too")
	}
	if _, err := os.Stat(filepath.Join(StorePresetsDir(), "demo.svg")); !os.IsNotExist(err) {
		t.Errorf("the icon file survived the removal: %v", err)
	}
}

// The default stack is never fetched from the store, so its marks have to come
// out of the binary or it could not carry one at all.
func TestPresetIcon_ServesTheDefaultStackFromTheEmbeddedBundle(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	icons := PresetIcons()
	for _, name := range []string{"mysql", "postgres", "redis", "meilisearch"} {
		svg, ok := PresetIcon(name)
		if !ok {
			t.Errorf("%s ships no mark", name)
			continue
		}
		if icons[name] != svg {
			t.Errorf("%s is served by PresetIcon but missing from PresetIcons", name)
		}
	}
}

// Every mark that ships has to already be in the form the sanitizer produces,
// so what the binary serves is never markup this build would have refused.
func TestPresetIcon_ShippedMarksAreAlreadySanitized(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	for name, svg := range PresetIcons() {
		clean, err := SanitizeStoreIcon([]byte(svg))
		if err != nil {
			t.Errorf("%s ships a mark the sanitizer rejects: %v", name, err)
			continue
		}
		if string(clean) != strings.TrimSpace(svg) {
			t.Errorf("%s ships a mark the sanitizer would change:\n have %s\n want %s", name, svg, clean)
		}
	}
}

// A published mark supersedes the shipped one, the same way a store preset
// supersedes the built-in YAML of the same name.
func TestPresetIcon_StoreCacheSupersedesTheEmbeddedMark(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	before, ok := PresetIcon("mysql")
	if !ok {
		t.Fatal("mysql ships no mark to supersede")
	}
	if err := SaveStorePresetIcon("mysql", []byte(`<svg viewBox="0 0 24 24"><path d="M0 0h9v9H0z"/></svg>`)); err != nil {
		t.Fatalf("SaveStorePresetIcon: %v", err)
	}
	after, _ := PresetIcon("mysql")
	if after == before || !strings.Contains(after, `d="M0 0h9v9H0z"`) {
		t.Errorf("the cached mark should win, got %s", after)
	}
	if PresetIcons()["mysql"] != after {
		t.Error("PresetIcons should list the cached mark, not the embedded one")
	}
}

// A preset that declares a colour must declare a plain hex, or the dashboard
// silently drops it and the tint reverts to the category.
func TestBundledPresets_DeclareUsableBrandColours(t *testing.T) {
	metas, err := ListPresets()
	if err != nil {
		t.Fatalf("ListPresets: %v", err)
	}
	var withColor int
	for _, m := range metas {
		p, err := LoadPreset(m.Name)
		if err != nil {
			continue
		}
		if p.Color == "" {
			continue
		}
		withColor++
		if NormalizeBrandColor(p.Color) == "" {
			t.Errorf("%s declares %q, which is not a plain hex", m.Name, p.Color)
		}
	}
	if withColor == 0 {
		t.Error("no bundled preset declares a brand colour")
	}
}

func TestStorePresetIcon_RejectsANameThatWouldEscapeTheCacheDir(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	if _, ok := PresetIcon("../../etc/passwd"); ok {
		t.Error("a traversing name must not resolve to an icon")
	}
	if err := SaveStorePresetIcon("../evil", []byte(`<svg viewBox="0 0 1 1"><path d="M0 0h1v1H0z"/></svg>`)); err == nil {
		t.Error("a traversing name must be rejected")
	}
}
