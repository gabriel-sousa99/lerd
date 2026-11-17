package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"
	"github.com/gabriel-sousa99/lerd/internal/envfile"
)

func TestEnvWiresLocalOracleWhenFrameworkDoesNotMapThePreset(t *testing.T) {
	for _, failStart := range []bool{false, true} {
		t.Run(map[bool]string{false: "ready", true: "startup fails"}[failStart], func(t *testing.T) {
			t.Setenv("XDG_CONFIG_HOME", t.TempDir())
			t.Setenv("XDG_DATA_HOME", t.TempDir())
			dir := t.TempDir()
			t.Chdir(dir)
			fw := &config.Framework{Name: "seed-app", Label: "Seed App", Env: config.FrameworkEnvConf{File: ".env", Format: "dotenv", SQLite: &config.FrameworkServiceDef{Detect: []config.FrameworkServiceDetect{{Key: "DB_CONNECTION", ValuePrefix: "sqlite"}}, Vars: []string{"DB_CONNECTION=sqlite"}}, Services: map[string]config.FrameworkServiceDef{"mysql": {Detect: []config.FrameworkServiceDetect{{Key: "DB_CONNECTION", ValuePrefix: "mysql"}}, Vars: []string{"DB_CONNECTION=mysql"}}}}}
			if err := config.SaveFramework(fw); err != nil {
				t.Fatal(err)
			}
			if err := config.SaveSites(&config.SiteRegistry{Sites: []config.Site{{Name: "seed-app", Domains: []string{"seed-app.localhost"}, Path: dir, Framework: fw.Name, PHPVersion: "8.4"}}}); err != nil {
				t.Fatal(err)
			}
			if err := config.SaveProjectConfig(dir, &config.ProjectConfig{Framework: fw.Name, Services: []config.ProjectService{{Name: "oracle-xe"}}}); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("DB_CONNECTION=sqlite\nDB_DATABASE=database/database.sqlite\n"), 0600); err != nil {
				t.Fatal(err)
			}
			original := ensureServiceRunning
			t.Cleanup(func() { ensureServiceRunning = original })
			var started []string
			ensureServiceRunning = func(name string) error {
				started = append(started, name)
				if failStart {
					return errors.New("incomplete seed data")
				}
				return nil
			}
			err := runEnv(nil, nil)
			if (err != nil) != failStart {
				t.Fatalf("startup failure=%v, env result=%v", failStart, err)
			}
			if failStart && !strings.Contains(err.Error(), "incomplete seed data") {
				t.Fatalf("startup error lost: %v", err)
			}
			values := envfile.ReadValues(filepath.Join(dir, ".env"))
			for key, want := range map[string]string{"DB_CONNECTION": "oracle", "DB_HOST": "lerd-oracle-xe", "DB_DATABASE": "LERDPDB", "DB_USERNAME": "lerd_app"} {
				if values[key] != want {
					t.Errorf("%s=%s, want %s", key, values[key], want)
				}
			}
			if len(started) != 1 || started[0] != "oracle-xe" {
				t.Fatalf("started=%v, want only oracle-xe", started)
			}
		})
	}
}
