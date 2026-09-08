package config

import (
	"os"
	"path/filepath"
	"testing"
)

// A project may write its domains with the TLD, so the sync back into .lerd.yaml
// has to recognise that main.test and main are the same domain. Comparing the
// raw strings left both spellings in the file, one per link.
func TestSyncProjectDomains_tldSpellingIsNotADuplicate(t *testing.T) {
	dir := t.TempDir()
	body := "domains:\n  - main.test\n  - subone.test\n  - subtwo\n"
	if err := os.WriteFile(filepath.Join(dir, ".lerd.yaml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	registered := []string{"main.test", "subone.test", "subtwo.test"}
	if err := SyncProjectDomains(dir, registered, "test"); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadProjectConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"main", "subone", "subtwo"}
	if len(cfg.Domains) != len(want) {
		t.Fatalf("domains = %v, want %v", cfg.Domains, want)
	}
	for i, w := range want {
		if cfg.Domains[i] != w {
			t.Errorf("domains[%d] = %q, want %q", i, cfg.Domains[i], w)
		}
	}
}

// A domain the user wrote with the TLD still has to come out of the file when
// it is removed, otherwise the next link re-registers it.
func TestRemoveProjectDomain_matchesTheTLDSpelling(t *testing.T) {
	dir := t.TempDir()
	body := "domains:\n  - main\n  - subone.test\n"
	if err := os.WriteFile(filepath.Join(dir, ".lerd.yaml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := ReplaceProjectDomain(dir, []string{"main.test"}, "subone.test", "test"); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadProjectConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Domains) != 1 || cfg.Domains[0] != "main" {
		t.Errorf("domains = %v, want [main]", cfg.Domains)
	}
}
