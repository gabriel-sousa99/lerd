package sitedoctor

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

// drupalish declares a primary env file lerd writes and a fallback published so
// an already-installed project can be read, which is Drupal's shape.
func drupalish() *config.Framework {
	return &config.Framework{
		Name:  "drupalish",
		Label: "Drupalish",
		Env: config.FrameworkEnvConf{
			File:           ".env",
			Format:         "dotenv",
			FallbackFile:   "web/sites/default/settings.php",
			FallbackFormat: "php-const",
			Services: map[string]config.FrameworkServiceDef{
				"mysql": {Vars: []string{"DB_HOST=lerd-mysql", "DB_NAME={{site}}"}},
			},
		},
	}
}

// A project read through its fallback still has no env file of its own, and
// reporting the fallback as present hides the one lerd and the framework's own
// install command both need.
func TestCheckEnvPresent_reportsTheMissingPrimaryBehindAReadableFallback(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "web", "sites", "default"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "web/sites/default/settings.php"), []byte("<?php\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	resp := Run(context.Background(), dir, drupalish())

	for _, c := range resp.Checks {
		if c.Name != "env_present" {
			continue
		}
		if c.Status != StatusFail || !strings.Contains(c.Detail, ".env") {
			t.Errorf("env_present = %+v, want a failure naming .env", c)
		}
		return
	}
	t.Fatal("no env_present check in the report")
}

func TestCheckEnvPresent_passesOnceThePrimaryExists(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("DB_HOST=lerd-mysql\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	resp := Run(context.Background(), dir, drupalish())

	for _, c := range resp.Checks {
		if c.Name == "env_present" && c.Status != StatusOK {
			t.Errorf("env_present = %+v, want ok", c)
		}
	}
}

// The finding has to name the remedy. Leaving it at "it is missing" starts the
// debugging session lerd exists to save.
func TestEnvMissingDetail_namesTheCommandThatCreatesIt(t *testing.T) {
	if d := envMissingDetail(".env", "", false); !strings.Contains(d, "lerd env") {
		t.Errorf("detail = %q, want the command that creates it", d)
	}
	d := envMissingDetail(".env", ".env.example", false)
	if !strings.Contains(d, "lerd env") || !strings.Contains(d, ".env.example") {
		t.Errorf("detail = %q, want the command and the example it copies", d)
	}
}

// A file the application writes during its own install must not be recommended
// into existence: an empty one reads to the framework as "already configured"
// and breaks a project that was only waiting to be installed (#1563).
func TestEnvMissingDetail_doesNotRecommendCreatingAnAppWrittenFile(t *testing.T) {
	d := envMissingDetail("config/system/settings.php", "", true)
	if strings.Contains(d, "lerd env") {
		t.Errorf("detail = %q, but running that command is what breaks the project", d)
	}
	if !strings.Contains(d, "install") {
		t.Errorf("detail = %q, want it to point at the framework's own install", d)
	}
}

// The same check keeps its ordinary advice for a dotenv file, which is lerd's
// to create.
func TestCheckEnvPresent_keepsTheCommandForALerdOwnedFile(t *testing.T) {
	c, _ := checkEnvPresent(t.TempDir(), ".env", ".env.example", false)
	if c.Status != StatusFail || !strings.Contains(c.Detail, "lerd env") {
		t.Errorf("check = %+v, want a failure naming lerd env", c)
	}
}

// An app file lerd can seed from an example is still lerd's to create, which is
// WordPress copying wp-config-sample.php, so the example wins over the installer.
func TestEnvMissingDetail_prefersTheExampleOverTheInstaller(t *testing.T) {
	d := envMissingDetail("wp-config.php", "wp-config-sample.php", true)
	if !strings.Contains(d, "lerd env") || !strings.Contains(d, "wp-config-sample.php") {
		t.Errorf("detail = %q, want the command and the sample it copies", d)
	}
}
