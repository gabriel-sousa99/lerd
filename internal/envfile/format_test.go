package envfile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The store reaches every install within a day, whatever binary it runs, so a
// definition naming a format added after a given release lands on machines that
// cannot honour it. Writing such a file as dotenv appended key=value lines into
// it; a PHP settings file so treated stops parsing and the site is down.
func TestApplyUpdatesIn_refusesAFormatItCannotWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.php")
	original := "<?php\n$databases['default']['default'] = ['driver' => 'sqlite'];\n"
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	err := ApplyUpdatesIn(path, "php-something-new", map[string]string{"databases.default.default.host": "lerd-mysql"})
	if err == nil {
		t.Fatal("an unknown format was written rather than refused")
	}
	if !strings.Contains(err.Error(), "update lerd") {
		t.Errorf("error = %q, want it to say what the user should do", err)
	}
	body, _ := os.ReadFile(path)
	if string(body) != original {
		t.Errorf("the file was touched by a refused write:\n%s", body)
	}
}

func TestApplyUpdatesIn_writesEveryFormatItKnows(t *testing.T) {
	dir := t.TempDir()
	cases := []struct{ name, format, seed, want string }{
		{".env", "dotenv", "DB_HOST=old\n", "DB_HOST=lerd-mysql"},
		{".env2", "", "DB_HOST=old\n", "DB_HOST=lerd-mysql"},
		{"wp-config.php", "php-const", "<?php\ndefine( 'DB_HOST', 'old' );\n", "lerd-mysql"},
	}
	for _, c := range cases {
		path := filepath.Join(dir, c.name)
		if err := os.WriteFile(path, []byte(c.seed), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := ApplyUpdatesIn(path, c.format, map[string]string{"DB_HOST": "lerd-mysql"}); err != nil {
			t.Fatalf("%s: %v", c.format, err)
		}
		body, _ := os.ReadFile(path)
		if !strings.Contains(string(body), c.want) {
			t.Errorf("%s not written:\n%s", c.format, body)
		}
	}
}

// Reading an unknown format must not invent keys out of whatever the file
// happens to contain, which is what parsing PHP as dotenv did.
func TestValues_returnsNothingForAnUnknownFormat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.php")
	if err := os.WriteFile(path, []byte("<?php\n$databases['default']['default'] = ['driver' => 'sqlite'];\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := Values(path, "php-something-new"); len(got) != 0 {
		t.Errorf("Values = %v, want nothing for a format this binary does not know", got)
	}
}

func TestKnownFormat(t *testing.T) {
	for _, f := range []string{"", "dotenv", "php-const", "php-array", "php-vars"} {
		if !KnownFormat(f) {
			t.Errorf("KnownFormat(%q) = false, want true", f)
		}
	}
	if KnownFormat("php-something-new") {
		t.Error("an unknown format reported as known")
	}
}

// The dotenv writer is reachable directly, and a caller that resolved a path
// without carrying its format alongside can hand it a PHP settings file. Writing
// `KEY=value` after the closing tag leaves a file that no longer parses, so the
// site is down: refuse instead, the same way an unknown format is refused.
func TestApplyUpdates_refusesAFileThatIsNotDotenv(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"wp-config.php", "settings.php"} {
		path := filepath.Join(dir, name)
		original := "<?php\ndefine( 'DB_NAME', 'blog' );\n"
		if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
			t.Fatal(err)
		}

		err := ApplyUpdates(path, map[string]string{"APP_KEY": "base64:abc"})
		if err == nil {
			t.Fatalf("%s was written as dotenv rather than refused", name)
		}
		if body, _ := os.ReadFile(path); string(body) != original {
			t.Errorf("%s was modified by a refused write:\n%s", name, body)
		}
	}

	// A real dotenv file is still written, leading blank lines and all.
	env := filepath.Join(dir, ".env")
	if err := os.WriteFile(env, []byte("\nAPP_ENV=local\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ApplyUpdates(env, map[string]string{"APP_KEY": "base64:abc"}); err != nil {
		t.Fatalf("a dotenv file must still be writable: %v", err)
	}
	if body, _ := os.ReadFile(env); !strings.Contains(string(body), "APP_KEY=base64:abc") {
		t.Errorf("key not written:\n%s", body)
	}
}
