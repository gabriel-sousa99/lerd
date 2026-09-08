package sitedoctor

import (
	"errors"
	"strings"
	"testing"
)

// A schema that does not exist is not something migrations can create: the
// migrate command fails against a database the engine does not have, so the
// finding used to name a remedy that could not work and sent the user to the
// CLI. The fix creates the database, and the migrate button comes back on the
// re-check that follows.
func TestCheckServerDatabase_offersToCreateTheDatabase(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("XDG_DATA_HOME", tmp)

	dir := t.TempDir()
	writeEnv(t, dir, ".env", "DB_CONNECTION=mysql\nDB_HOST=lerd-mysql\nDB_DATABASE=shop\n")
	restore := stubDatabaseLister(func(string) ([]string, error) { return []string{"other"}, nil })
	defer restore()

	c, ok := checkServerDatabase(dir)
	if !ok || c.Status != StatusFail {
		t.Fatalf("check = %+v (ok=%v), want a failure for the missing schema", c, ok)
	}
	if c.Fix != FixCreateDatabase {
		t.Errorf("fix = %q, want %q", c.Fix, FixCreateDatabase)
	}
	if strings.Contains(c.Detail, "lerd db:create") {
		t.Errorf("detail = %q, should not send the user to the CLI now the fix creates it", c.Detail)
	}
}

// The fix works the set out again rather than trusting the client, so it can
// only ever create what the check would have reported.
func TestMissingDatabases(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("XDG_DATA_HOME", tmp)

	dir := t.TempDir()
	writeEnv(t, dir, ".env", "DB_CONNECTION=mysql\nDB_HOST=lerd-mysql\nDB_DATABASE=shop\n")

	restore := stubDatabaseLister(func(string) ([]string, error) { return []string{"other"}, nil })
	missing := MissingDatabases(dir)
	restore()
	if len(missing) != 1 || missing[0].Database != "shop" || missing[0].Service != "mysql" {
		t.Fatalf("missing = %+v, want shop on mysql", missing)
	}

	restore = stubDatabaseLister(func(string) ([]string, error) { return []string{"shop"}, nil })
	if got := MissingDatabases(dir); len(got) != 0 {
		t.Errorf("missing = %+v, want none once the database exists", got)
	}
	restore()

	// An engine that cannot be reached holds no missing database as far as the
	// fix is concerned, or pressing it would create a schema that already exists.
	restore = stubDatabaseLister(func(string) ([]string, error) { return nil, errors.New("engine down") })
	defer restore()
	if got := MissingDatabases(dir); len(got) != 0 {
		t.Errorf("missing = %+v, want none while the engine is unreachable", got)
	}
}
