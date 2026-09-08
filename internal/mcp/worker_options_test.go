package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

// siteWithTunableQueue seeds a store framework whose queue worker declares a
// tune_command and registers a site for it.
func siteWithTunableQueue(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, "data"))

	storeDir := config.StoreFrameworksDir()
	if err := os.MkdirAll(storeDir, 0o755); err != nil {
		t.Fatal(err)
	}
	def := "name: tinyfw\nlabel: TinyFW\nworkers:\n  queue:\n    label: Queue Worker\n" +
		"    command: php artisan queue:work --queue=default --tries=3\n" +
		"    tune_command: php artisan queue:work --queue={queue} --tries={tries}\n" +
		"  vite:\n    label: Vite\n    command: npm run dev\n"
	if err := os.WriteFile(filepath.Join(storeDir, "tinyfw.yaml"), []byte(def), 0o644); err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".lerd.yaml"), []byte("framework: tinyfw\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := config.SaveSites(&config.SiteRegistry{Sites: []config.Site{
		{Name: "shop", Path: dir, Framework: "tinyfw"},
	}}); err != nil {
		t.Fatal(err)
	}
	return dir
}

var tunableQueue = config.FrameworkWorker{
	Command:     "php artisan queue:work --queue=default --tries=3",
	TuneCommand: "php artisan queue:work --queue={queue} --tries={tries}",
}

func TestWorkerTuneArgs(t *testing.T) {
	t.Run("renders a flag per declared placeholder", func(t *testing.T) {
		got, err := workerTuneArgs(tunableQueue, "queue", []string{"queue=emails", "tries=5"})
		if err != nil {
			t.Fatalf("workerTuneArgs: %v", err)
		}
		if strings.Join(got, " ") != "--queue=emails --tries=5" {
			t.Errorf("got %v", got)
		}
	})

	t.Run("refuses a value the definition does not declare", func(t *testing.T) {
		_, err := workerTuneArgs(tunableQueue, "queue", []string{"timeout=90"})
		if err == nil || !strings.Contains(err.Error(), "queue, tries") {
			t.Fatalf("want an error naming the declared values, got %v", err)
		}
	})

	t.Run("refuses a worker that declares no tuning", func(t *testing.T) {
		plain := config.FrameworkWorker{Command: "npm run dev"}
		if _, err := workerTuneArgs(plain, "vite", []string{"port=5173"}); err == nil {
			t.Fatal("want an error for a worker with no tune_command")
		}
	})

	t.Run("refuses an option that is not name=value", func(t *testing.T) {
		if _, err := workerTuneArgs(tunableQueue, "queue", []string{"emails"}); err == nil {
			t.Fatal("want an error for an option without a name")
		}
	})

	// The value lands in the ExecStart line of the worker's unit, so it is
	// refused here rather than in the process that would run it.
	t.Run("refuses a value that would add an argument", func(t *testing.T) {
		if _, err := workerTuneArgs(tunableQueue, "queue", []string{"queue=default\nExecStartPost=/bin/sh -c evil"}); err == nil {
			t.Fatal("want an error for a value carrying a newline")
		}
	})
}

// An injectable value must be refused before anything is spawned, which is what
// keeps the newline out of the unit the start would write.
func TestWorkerStart_rejectsAnInjectableTuningValue(t *testing.T) {
	siteWithTunableQueue(t)
	res, _ := callToolRaw(t, "worker", map[string]any{
		"action":  "start",
		"site":    "shop",
		"worker":  "queue",
		"options": []any{"queue=default\nExecStartPost=/bin/sh -c evil"},
	})
	if !mcpIsError(res) {
		t.Fatalf("expected an error for a queue value with a newline, got: %s", mcpText(t, res))
	}
}

func TestWorkerStart_rejectsAnUndeclaredOption(t *testing.T) {
	siteWithTunableQueue(t)
	res, _ := callToolRaw(t, "worker", map[string]any{
		"action":  "start",
		"site":    "shop",
		"worker":  "queue",
		"options": []any{"timeout=90"},
	})
	if !mcpIsError(res) {
		t.Fatalf("expected an error for an undeclared option, got: %s", mcpText(t, res))
	}
}

// The schema no longer spells out any worker's knobs, so list is where an
// assistant learns them: the placeholders, the default the definition runs and
// whatever the project committed.
func TestWorkerList_reportsWhatTheDefinitionMakesTunable(t *testing.T) {
	dir := siteWithTunableQueue(t)
	if err := config.SetProjectWorkerOptions(dir, "queue", map[string]string{"queue": "emails"}); err != nil {
		t.Fatal(err)
	}

	res, _ := execWorkerList(map[string]any{"site": "shop"})
	var workers []struct {
		Name    string                    `json:"name"`
		Options []config.WorkerTuneOption `json:"options"`
	}
	if err := json.Unmarshal([]byte(mcpText(t, res)), &workers); err != nil {
		t.Fatalf("decoding worker list: %v", err)
	}

	var queue, vite *struct {
		Name    string                    `json:"name"`
		Options []config.WorkerTuneOption `json:"options"`
	}
	for i := range workers {
		switch workers[i].Name {
		case "queue":
			queue = &workers[i]
		case "vite":
			vite = &workers[i]
		}
	}
	if queue == nil || vite == nil {
		t.Fatalf("want both workers listed, got %s", mcpText(t, res))
	}
	if len(vite.Options) != 0 {
		t.Errorf("a worker without a tune_command should report no options, got %v", vite.Options)
	}
	if len(queue.Options) != 2 {
		t.Fatalf("queue options = %v, want queue and tries", queue.Options)
	}
	if queue.Options[0].Name != "queue" || queue.Options[0].Default != "default" || queue.Options[0].Value != "emails" {
		t.Errorf("queue option = %+v, want the committed value alongside the definition default", queue.Options[0])
	}
	if queue.Options[1].Name != "tries" || queue.Options[1].Default != "3" {
		t.Errorf("tries option = %+v, want the definition default", queue.Options[1])
	}
}
