package sitedoctor

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

// A check that only speaks about an optional package (NativePHP's desktop
// runtime) has nothing to say on a project without it, and a permanently green
// row is clutter. The gate is the same FrameworkRule workers and commands use.
func TestFrameworkChecks_DropsCheckWhoseGateFails(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "composer.json"), []byte(`{"require":{"laravel/framework":"^12.0"}}`), 0644); err != nil {
		t.Fatal(err)
	}
	fw := &config.Framework{Doctor: &config.FrameworkDoctor{Checks: []config.DoctorCheck{
		{Name: "always", Type: "command", Command: "true"},
		{Name: "gated_out", Type: "command", Command: "true", Check: &config.FrameworkRule{Composer: "nativephp/electron"}},
		{Name: "gated_in", Type: "command", Command: "true", Check: &config.FrameworkRule{Composer: "laravel/framework"}},
	}}}

	var names []string
	for _, c := range frameworkChecks(fw, dir) {
		names = append(names, c.Name)
	}
	want := []string{"always", "gated_in"}
	if len(names) != len(want) {
		t.Fatalf("checks = %v, want %v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("checks = %v, want %v", names, want)
		}
	}
}
