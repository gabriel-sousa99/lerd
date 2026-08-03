package cli

import (
	"os"
	"path/filepath"
	"testing"
)

// The Oracle baseline exists to seed the key shape, not to own the values: the
// wizard tells the user to leave host/user/password blank and fill them in .env
// directly, so a second `lerd env` must not reset them to the empty defaults.
func TestSeedEnvDefaults_keepsHandFilledValues(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	os.WriteFile(envPath, []byte(
		"DB_CONNECTION=oracle\n"+
			"DB_HOST=oracle.corp.example.com\n"+
			"DB_PORT=1600\n"+
			"DB_DATABASE=PROD1\n"+
			"DB_USERNAME=app_user\n"+
			"DB_PASSWORD=s3cret\n"), 0644)

	updates := map[string]string{}
	seedEnvDefaults(envPath, oracleEnvVarsDefault, updates)

	// DB_CONNECTION is the one value the database choice owns.
	if updates["DB_CONNECTION"] != "oracle" {
		t.Errorf("DB_CONNECTION = %q; want oracle (the selection must be enforced)", updates["DB_CONNECTION"])
	}
	// Everything the user filled in must be left alone entirely.
	for _, k := range []string{"DB_HOST", "DB_PORT", "DB_DATABASE", "DB_USERNAME", "DB_PASSWORD"} {
		if v, ok := updates[k]; ok {
			t.Errorf("%s would be rewritten to %q; want it left as the user wrote it", k, v)
		}
	}
}

// On a project whose .env has none of the keys, the baseline still seeds all of
// them so the file shape is right.
func TestSeedEnvDefaults_seedsMissingKeys(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	os.WriteFile(envPath, []byte("APP_NAME=Laravel\n"), 0644)

	updates := map[string]string{}
	seedEnvDefaults(envPath, oracleEnvVarsDefault, updates)

	want := map[string]string{
		"DB_CONNECTION": "oracle",
		"DB_HOST":       "",
		"DB_PORT":       "1521",
		"DB_DATABASE":   "XEPDB1",
		"DB_USERNAME":   "",
		"DB_PASSWORD":   "",
	}
	for k, v := range want {
		got, ok := updates[k]
		if !ok {
			t.Errorf("%s missing from updates; want it seeded as %q", k, v)
			continue
		}
		if got != v {
			t.Errorf("%s = %q; want %q", k, got, v)
		}
	}
}

// A key present but empty is a blank the baseline is allowed to fill.
func TestSeedEnvDefaults_fillsPresentButEmptyKeys(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	os.WriteFile(envPath, []byte("DB_PORT=\nDB_DATABASE=   \n"), 0644)

	updates := map[string]string{}
	seedEnvDefaults(envPath, oracleEnvVarsDefault, updates)

	if updates["DB_PORT"] != "1521" {
		t.Errorf("DB_PORT = %q; want 1521 (an empty value is a blank to fill)", updates["DB_PORT"])
	}
	if updates["DB_DATABASE"] != "XEPDB1" {
		t.Errorf("DB_DATABASE = %q; want XEPDB1 (whitespace-only counts as blank)", updates["DB_DATABASE"])
	}
}

// A missing .env must not panic; every default is simply seeded.
func TestSeedEnvDefaults_missingEnvFile(t *testing.T) {
	updates := map[string]string{}
	seedEnvDefaults(filepath.Join(t.TempDir(), "absent", ".env"), oracleEnvVarsDefault, updates)
	if updates["DB_PORT"] != "1521" {
		t.Errorf("DB_PORT = %q; want 1521 when there is no .env to read", updates["DB_PORT"])
	}
}
