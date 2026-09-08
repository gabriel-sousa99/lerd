package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A .lerd.yaml people get wrong most often is a top-level sequence: the leading
// dash on `- domains` makes the document a list, which never unmarshals into
// ProjectConfig. The error has to name the file, since the caller reporting it
// only knows the directory.
func TestLoadProjectConfig_unparseableNamesTheFile(t *testing.T) {
	dir := t.TempDir()
	broken := "- domains\n    - main\n    - subone\n"
	if err := os.WriteFile(filepath.Join(dir, ".lerd.yaml"), []byte(broken), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadProjectConfig(dir)
	if err == nil {
		t.Fatal("a top-level sequence must not parse as a project config")
	}
	if !strings.Contains(err.Error(), ".lerd.yaml") {
		t.Errorf("error = %q, want the file named in it", err)
	}
	// Roughly thirty call sites read this with `proj, _ :=`, and two of them
	// dereference the result without a nil check. A failed load returns an
	// empty config so those keep working instead of panicking.
	if cfg == nil {
		t.Fatal("a failed load must still return an empty config, not nil")
	}
	if !cfg.IsEmpty() {
		t.Errorf("config from a failed load = %+v, want empty", cfg)
	}
}
