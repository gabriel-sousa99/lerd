package cli

import (
	"strings"
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

func TestLinkAsksForDatabaseWhenOnlyOtherServicesAreConfigured(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	for _, tc := range []struct {
		name string
		proj *config.ProjectConfig
		want bool
	}{
		{"new", nil, true},
		{"redis only", &config.ProjectConfig{Services: []config.ProjectService{{Name: "redis"}}}, true},
		{"local oracle", &config.ProjectConfig{Services: []config.ProjectService{{Name: "oracle-xe"}}}, false},
		{"external oracle", &config.ProjectConfig{Oracle: &config.ProjectOracleConfig{Host: "db.example"}}, false},
		{"mysql", &config.ProjectConfig{Services: []config.ProjectService{{Name: "mysql"}}}, false},
		{"explicit db", &config.ProjectConfig{DB: config.ProjectDB{Service: "postgres"}}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := linkNeedsDatabaseChoice(tc.proj); got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestDatabaseOptionsDistinguishLocalOracleFromExternal(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	options, _ := buildDatabaseOptions(nil)
	labels := map[string]string{}
	for _, o := range options {
		labels[o.Value] = o.Key
	}
	if !strings.Contains(labels["oracle-xe"], "Oracle XE") || !strings.Contains(labels["oracle-xe"], "local") {
		t.Fatalf("unclear local Oracle option: %q", labels["oracle-xe"])
	}
	if !strings.Contains(strings.ToLower(labels["oracle"]), "external") {
		t.Fatalf("unclear external Oracle option: %q", labels["oracle"])
	}
}

func TestOracleLocationRequiresAnExplicitLocalChoice(t *testing.T) {
	for _, tc := range []struct {
		choice string
		yes    bool
		want   string
		calls  int
	}{
		{"oracle", true, "oracle-xe", 1}, {"oracle", false, "oracle", 1},
		{"oracle-xe", false, "oracle-xe", 0}, {"mysql", true, "mysql", 0},
	} {
		calls := 0
		got := promptOracleLocation(tc.choice, func(_ string, defaultYes bool) bool {
			calls++
			if defaultYes {
				t.Fatal("must preserve the existing external choice by default")
			}
			return tc.yes
		})
		if got != tc.want || calls != tc.calls {
			t.Fatalf("%s yes=%v: got %s (%d prompts), want %s (%d prompts)", tc.choice, tc.yes, got, calls, tc.want, tc.calls)
		}
	}
}

func TestWizardSwitchingToLocalOracleDropsExternalConnection(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	defaults := &config.ProjectConfig{Oracle: &config.ProjectOracleConfig{Host: "corporate.example", Username: "remote-user"}}
	for _, choice := range []string{"oracle", "oracle-xe"} {
		cfg, err := projectConfigFromAnswers(t.TempDir(), defaults, ProjectAnswers{Kind: ProjectKindPHP, Database: choice, PHPVersion: "8.4"}, false)
		if err != nil {
			t.Fatal(err)
		}
		if (cfg.Oracle != nil) != (choice == "oracle") {
			t.Fatalf("choice=%s, unexpected external Oracle config: %+v", choice, cfg.Oracle)
		}
		if len(cfg.Services) != 1 || cfg.Services[0].Name != choice {
			t.Fatalf("wrong services for %s: %+v", choice, cfg.Services)
		}
	}
}

func TestLocalOracleChoiceOverridesDetectedSQLite(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	chosen := map[string]bool{"oracle-xe": true}
	if !userPickedDBFromYAML(chosen) {
		t.Fatal("local Oracle must count as an explicit database choice")
	}
	fw := &config.Framework{}
	fw.Env.SQLite = &config.FrameworkServiceDef{Detect: []config.FrameworkServiceDetect{{Key: "DB_CONNECTION", ValuePrefix: "sqlite"}}}
	if projectUsesSQLite(chosen, map[string]string{"DB_CONNECTION": "sqlite"}, fw, false) {
		t.Fatal("detected SQLite overrides selected Oracle XE")
	}
}
