package store

import (
	"os"
	"testing"
	"time"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

// RefreshIndex must persist the fetched index to config.StoreIndexFile() so the
// offline detection and listing paths can read the full catalogue without a
// network round trip. loadCachedIndex reads that same file back.
func TestRefreshIndex_CachesToDisk(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	srv := testServer(t)
	defer srv.Close()
	c := testClient(t, srv)

	if _, err := c.RefreshIndex(); err != nil {
		t.Fatalf("RefreshIndex: %v", err)
	}
	if _, err := os.Stat(config.StoreIndexFile()); err != nil {
		t.Fatalf("expected cached index at %s: %v", config.StoreIndexFile(), err)
	}

	idx, err := loadCachedIndex()
	if err != nil {
		t.Fatalf("loadCachedIndex: %v", err)
	}
	if len(idx.Frameworks) != 2 || idx.Frameworks[1].Name != "symfony" {
		t.Fatalf("unexpected cached index: %+v", idx)
	}
}

// A refresh that finds the catalogue unchanged still has to move the cache's
// mtime forward. That mtime is how the resolvers decide whether what the index
// does not list counts as evidence, and a store that publishes nothing new for a
// day would otherwise look like a store nobody has reached in one.
func TestRefreshIndex_TouchesAnUnchangedCache(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	srv := testServer(t)
	defer srv.Close()
	c := testClient(t, srv)

	if _, err := c.RefreshIndex(); err != nil {
		t.Fatalf("RefreshIndex: %v", err)
	}
	backdated := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(config.StoreIndexFile(), backdated, backdated); err != nil {
		t.Fatalf("Chtimes: %v", err)
	}

	if _, err := c.RefreshIndex(); err != nil {
		t.Fatalf("RefreshIndex (second): %v", err)
	}
	info, err := os.Stat(config.StoreIndexFile())
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if time.Since(info.ModTime()) > time.Minute {
		t.Errorf("cached index still dated %v, want it touched by the refresh", info.ModTime())
	}
}

// Fetching a framework without naming a version reads the index to resolve the
// latest, and that copy has to be kept. It is the fetch a machine that has never
// reached the store makes, and throwing the index away there leaves detection
// seeing only the built-ins until the watcher's first refresh.
func TestFetchFramework_LatestCachesTheIndex(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	srv := testServer(t)
	defer srv.Close()
	c := testClient(t, srv)

	if _, err := c.FetchFramework("laravel", ""); err != nil {
		t.Fatalf("FetchFramework(laravel, latest): %v", err)
	}
	idx, err := loadCachedIndex()
	if err != nil {
		t.Fatalf("loadCachedIndex: %v", err)
	}
	if len(idx.Frameworks) != 2 {
		t.Fatalf("cached index has %d frameworks, want 2", len(idx.Frameworks))
	}
}
