package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeCachedFramework(t *testing.T, filename, body string) {
	t.Helper()
	dir := StoreFrameworksDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, filename), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestSaveStoreFrameworkIcon_SanitizesOnTheWayInAndTheSeamServesIt(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	raw := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="#ff2d20"><path d="M2 2h8v8H2z"/><script>bad()</script></svg>`
	if err := SaveStoreFrameworkIcon("laravel", []byte(raw)); err != nil {
		t.Fatalf("SaveStoreFrameworkIcon: %v", err)
	}
	svg, ok := FrameworkIcon("laravel")
	if !ok {
		t.Fatal("the seam did not serve the saved mark")
	}
	if strings.Contains(svg, "bad()") || strings.Contains(svg, "#ff2d20") {
		t.Errorf("the cached mark was not sanitized: %s", svg)
	}
	if !strings.Contains(svg, `d="M2 2h8v8H2z"`) {
		t.Errorf("the cached mark lost its drawing: %s", svg)
	}
}

// One mark serves the whole family, so it sits beside the versioned files
// rather than inside them and every version resolves to the same file.
func TestFrameworkMarks_OneMarkServesEveryVersion(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	writeCachedFramework(t, "laravel@11.yaml", "name: laravel\nlabel: Laravel\nversion: \"11\"\ncolor: \"#FF2D20\"\n")
	writeCachedFramework(t, "laravel@12.yaml", "name: laravel\nlabel: Laravel\nversion: \"12\"\ncolor: \"#FF2D20\"\n")
	if err := SaveStoreFrameworkIcon("laravel", []byte(`<svg viewBox="0 0 24 24"><path d="M2 2h8v8H2z"/></svg>`)); err != nil {
		t.Fatalf("SaveStoreFrameworkIcon: %v", err)
	}

	marks := FrameworkMarks()
	got, ok := marks["laravel"]
	if !ok {
		t.Fatalf("laravel missing from the marks: %v", marks)
	}
	if !strings.Contains(got.SVG, "<path") {
		t.Errorf("laravel mark = %q", got.SVG)
	}
	if got.Color != "#ff2d20" {
		t.Errorf("laravel colour = %q, want the normalized hex", got.Color)
	}
	if len(marks) != 1 {
		t.Errorf("two version files must not become two entries, got %v", marks)
	}
}

// A colour with no mark still tints the label, so it must not be dropped for
// want of artwork.
func TestFrameworkMarks_ColourAloneStillCounts(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	writeCachedFramework(t, "symfony@7.yaml", "name: symfony\nlabel: Symfony\nversion: \"7\"\ncolor: \"#000000\"\n")
	writeCachedFramework(t, "tempest@1.yaml", "name: tempest\nlabel: Tempest\nversion: \"1\"\n")

	marks := FrameworkMarks()
	if marks["symfony"].Color != "#000000" {
		t.Errorf("symfony should carry its colour, got %+v", marks["symfony"])
	}
	if _, ok := marks["tempest"]; ok {
		t.Errorf("a framework with neither mark nor colour should be absent, got %+v", marks["tempest"])
	}
}

// A colour that is not a plain hex must never reach the page as CSS.
func TestFrameworkMarks_DropsAColourThatIsNotPlainHex(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	writeCachedFramework(t, "evil@1.yaml", "name: evil\nlabel: Evil\nversion: \"1\"\ncolor: \"url(https://evil.test/x)\"\n")
	if _, ok := FrameworkMarks()["evil"]; ok {
		t.Error("a non-hex colour must be dropped, leaving nothing to serve")
	}
}

// The cache holds <name>@<version>.yaml, so the family name has to come off the
// left of the @ rather than the whole stem.
func TestFrameworkMarks_KeysByFamilyNotByVersionedFilename(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	writeCachedFramework(t, "wordpress.yaml", "name: wordpress\nlabel: WordPress\ncolor: \"#21759B\"\n")
	marks := FrameworkMarks()
	if _, ok := marks["wordpress"]; !ok {
		t.Errorf("an unversioned definition should still key by name, got %v", marks)
	}
}
