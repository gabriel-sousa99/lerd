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

// viteFromVersion is the first major of each framework whose own scaffolding or
// standard integration builds assets with vite. Older majors shipped webpack
// (Mix, Encore) and never grow a dev server, so they declare no vite worker.
var viteFromVersion = map[string]int{
	"laravel":  9,
	"statamic": 4,
	"symfony":  5,
	"cakephp":  4,
	"tempest":  3,
}

// Serving a dev server under the site's domain is a property of the tool, not of
// the framework running it, so every definition that can run vite declares the
// worker in one shape. The check gates it on vite being installed, which is what
// keeps it invisible to a project that never added a dev server.
func TestStoreFrameworks_DeclareViteWorker(t *testing.T) {
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
			major, err := strconv.Atoi(strings.TrimSuffix(f.Name(), ".yaml"))
			if err != nil {
				t.Fatalf("%s: version is not a major", name)
			}
			from, wantVite := viteFromVersion[d.Name()]
			wantVite = wantVite && major >= from

			v, ok := fw.Workers["vite"]
			if !ok {
				if wantVite {
					t.Errorf("%s: no vite worker declared", name)
				}
				continue
			}
			checked++
			if !v.Host {
				t.Errorf("%s: vite worker must run on the host", name)
			}
			if v.PerWorktree == nil || !*v.PerWorktree {
				t.Errorf("%s: vite worker must be per_worktree", name)
			}
			if !v.ReplacesBuild {
				t.Errorf("%s: vite worker must set replaces_build", name)
			}
			if v.Restart != "on-failure" {
				t.Errorf("%s: vite worker restart = %q, want on-failure", name, v.Restart)
			}
			if v.Command != "npm run dev" {
				t.Errorf("%s: vite worker command = %q, want npm run dev", name, v.Command)
			}
			if v.Check == nil || v.Check.File != "node_modules/vite" {
				t.Errorf("%s: vite worker must check for node_modules/vite", name)
			}
		}
	}
	if checked == 0 {
		t.Skip("no framework definitions in the store checkout")
	}
}
