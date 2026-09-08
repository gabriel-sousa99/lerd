package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProjectWorkerOptions(t *testing.T) {
	dir := setupProjectConfig(t, &ProjectConfig{WorkerOptions: map[string]map[string]string{
		"queue": {"queue": "high,default,low", "tries": "5"},
	}})

	got := ProjectWorkerOptions(dir, "queue")
	if got["queue"] != "high,default,low" || got["tries"] != "5" {
		t.Errorf("got %v, want the persisted queue options", got)
	}
	if got := ProjectWorkerOptions(dir, "schedule"); len(got) != 0 {
		t.Errorf("worker with no options: got %v, want none", got)
	}
	if got := ProjectWorkerOptions(t.TempDir(), "queue"); len(got) != 0 {
		t.Errorf("project with no .lerd.yaml: got %v, want none", got)
	}
}

func TestSetProjectWorkerOptions(t *testing.T) {
	t.Run("creates .lerd.yaml so the options survive", func(t *testing.T) {
		dir := t.TempDir()
		if err := SetProjectWorkerOptions(dir, "queue", map[string]string{"queue": "emails"}); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(filepath.Join(dir, ".lerd.yaml")); err != nil {
			t.Fatalf("expected .lerd.yaml to be created: %v", err)
		}
		if got := ProjectWorkerOptions(dir, "queue"); got["queue"] != "emails" {
			t.Errorf("got %v, want queue=emails", got)
		}
	})

	t.Run("replaces the worker's options and leaves other workers alone", func(t *testing.T) {
		dir := setupProjectConfig(t, &ProjectConfig{WorkerOptions: map[string]map[string]string{
			"queue":   {"queue": "emails", "tries": "5"},
			"horizon": {"tries": "1"},
		}})
		if err := SetProjectWorkerOptions(dir, "queue", map[string]string{"queue": "high,low"}); err != nil {
			t.Fatal(err)
		}
		got := ProjectWorkerOptions(dir, "queue")
		if got["queue"] != "high,low" || len(got) != 1 {
			t.Errorf("got %v, want only queue=high,low", got)
		}
		if got := ProjectWorkerOptions(dir, "horizon"); got["tries"] != "1" {
			t.Errorf("other worker: got %v, want tries=1", got)
		}
	})

	t.Run("empty values drop the worker, and the block with the last one", func(t *testing.T) {
		dir := setupProjectConfig(t, &ProjectConfig{WorkerOptions: map[string]map[string]string{
			"queue": {"queue": "emails"},
		}})
		if err := SetProjectWorkerOptions(dir, "queue", nil); err != nil {
			t.Fatal(err)
		}
		cfg := loadConfig(t, dir)
		if cfg.WorkerOptions != nil {
			t.Errorf("got %v, want the block gone", cfg.WorkerOptions)
		}
	})

	t.Run("clearing a project with no .lerd.yaml writes nothing", func(t *testing.T) {
		dir := t.TempDir()
		if err := SetProjectWorkerOptions(dir, "queue", nil); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(filepath.Join(dir, ".lerd.yaml")); !os.IsNotExist(err) {
			t.Errorf("expected no .lerd.yaml, got %v", err)
		}
	})
}
