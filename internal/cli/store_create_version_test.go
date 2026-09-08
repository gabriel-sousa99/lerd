package cli

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"
	"gopkg.in/yaml.v3"
)

// createPackage returns the composer package a create command scaffolds from,
// the last argument naming a vendor/package rather than a flag.
func createPackage(create string) string {
	pkg := ""
	for _, arg := range strings.Fields(create) {
		if strings.HasPrefix(arg, "-") || !strings.Contains(arg, "/") {
			continue
		}
		pkg = arg
	}
	return pkg
}

// lerd new asks which major to scaffold and resolves that major's definition, so
// a create command that leaves its package unconstrained hands composer the
// newest release whatever the answer was and the project on disk ends up a major
// the definition was never written for.
func TestStoreFrameworks_CreateNamesItsMajor(t *testing.T) {
	root := filepath.Join("..", "..", "lerd-frameworks", "frameworks")
	dirs, err := os.ReadDir(root)
	if err != nil {
		t.Skipf("frameworks store checkout not present: %v", err)
	}
	checked := 0
	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}
		files, err := os.ReadDir(filepath.Join(root, d.Name()))
		if err != nil {
			t.Fatalf("read %s: %v", d.Name(), err)
		}
		for _, f := range files {
			if f.IsDir() || filepath.Ext(f.Name()) != ".yaml" {
				continue
			}
			name := filepath.Join(d.Name(), f.Name())
			b, err := os.ReadFile(filepath.Join(root, name))
			if err != nil {
				t.Fatalf("read %s: %v", name, err)
			}
			var fw config.Framework
			if err := yaml.Unmarshal(b, &fw); err != nil {
				t.Fatalf("unmarshal %s: %v", name, err)
			}
			if fw.Create == "" {
				continue
			}
			checked++

			pkg := createPackage(fw.Create)
			vendor, constraint, ok := strings.Cut(pkg, ":")
			if !ok {
				t.Errorf("%s: create scaffolds %s unconstrained, want :^%s.0", name, pkg, fw.Version)
				continue
			}
			major, _, _ := strings.Cut(strings.TrimLeft(constraint, "^~>=v "), ".")
			if major != fw.Version {
				t.Errorf("%s: create scaffolds %s:%s, want major %s", name, vendor, constraint, fw.Version)
			}
			if _, err := strconv.Atoi(fw.Version); err != nil {
				t.Errorf("%s: version %q is not a major", name, fw.Version)
			}
		}
	}
	if checked == 0 {
		t.Skip("no framework definitions in the store checkout")
	}
}
