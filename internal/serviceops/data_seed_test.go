package serviceops

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

func seedTestService() *config.CustomService {
	return &config.CustomService{Name: "seed-db", Image: "example.test/db:1", DataDir: "/data", DataSeed: []string{"db/control", "db/system"}}
}

func writeSeedFiles(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, "db"), 0755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"control", "system"} {
		if err := os.WriteFile(filepath.Join(dir, "db", name), []byte("seed"), 0600); err != nil {
			t.Fatal(err)
		}
	}
}

func TestPrepareServiceDataPublishesCompleteSeedOnlyOnce(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	svc := seedTestService()
	calls := 0
	copyData := func(_ *config.CustomService, dest string) error { calls++; writeSeedFiles(t, dest); return nil }
	if err := prepareServiceData(svc, copyData); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(config.DataSubDir(svc.Name), "db/control")
	if err := os.WriteFile(path, []byte("user database"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := prepareServiceData(svc, copyData); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("copied image %d times, want once", calls)
	}
	data, _ := os.ReadFile(path)
	if string(data) != "user database" {
		t.Fatal("existing database overwritten")
	}
}

func TestPrepareServiceDataRefusesPartialDatabaseWithoutChangingIt(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	svc := seedTestService()
	dir := config.DataSubDir(svc.Name)
	if err := os.MkdirAll(filepath.Join(dir, "dbconfig"), 0755); err != nil {
		t.Fatal(err)
	}
	called := false
	err := prepareServiceData(svc, func(_ *config.CustomService, _ string) error { called = true; return nil })
	if err == nil || !strings.Contains(err.Error(), "incomplete") {
		t.Fatalf("want incomplete-data error, got %v", err)
	}
	if called {
		t.Fatal("tried to replace an existing data directory")
	}
	if _, err := os.Stat(filepath.Join(dir, "dbconfig")); err != nil {
		t.Fatal("existing files changed")
	}
}

func TestPrepareServiceDataFailedCopyLeavesTargetEmptyAndCanRetry(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	svc := seedTestService()
	err := prepareServiceData(svc, func(_ *config.CustomService, dest string) error {
		writeSeedFiles(t, dest)
		return errors.New("copy interrupted")
	})
	if err == nil {
		t.Fatal("copy failure was ignored")
	}
	entries, readErr := os.ReadDir(config.DataSubDir(svc.Name))
	if readErr != nil && !os.IsNotExist(readErr) {
		t.Fatal(readErr)
	}
	if len(entries) != 0 {
		t.Fatal("partial copy leaked into live data")
	}
	if err := prepareServiceData(svc, func(_ *config.CustomService, dest string) error { writeSeedFiles(t, dest); return nil }); err != nil {
		t.Fatal(err)
	}
}

func TestPrepareServiceDataRejectsIncompleteImageSeed(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	err := prepareServiceData(seedTestService(), func(_ *config.CustomService, _ string) error { return nil })
	if err == nil {
		t.Fatal("published a seed that lacks required files")
	}
}
