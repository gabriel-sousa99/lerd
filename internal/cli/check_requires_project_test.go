package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The alias has to keep failing where the old command did. Run one directory too
// high, "all checks pass" reads as a healthy project rather than as the wrong
// directory, which is the answer the user actually needed.
func TestRequireProjectConfig_failsWithoutTheFile(t *testing.T) {
	err := requireProjectConfig(t.TempDir())
	if err == nil {
		t.Fatal("expected a failure in a directory with no .lerd.yaml")
	}
	if !strings.Contains(err.Error(), ".lerd.yaml") || !strings.Contains(err.Error(), "lerd init") {
		t.Errorf("error = %q, want it to name the file and the command that creates it", err)
	}
}

func TestRequireProjectConfig_passesWithTheFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".lerd.yaml"), []byte("php_version: \"8.5\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := requireProjectConfig(dir); err != nil {
		t.Errorf("requireProjectConfig = %v, want nil", err)
	}
}
