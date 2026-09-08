package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

// A worker whose command names a program the project has not installed yet dies
// on "No such file or directory" and is restarted every few seconds, so the
// reason scrolls past in a log nobody is reading while the unit flaps.
func TestMissingWorkerProgramMsg_namesTheProgramTheProjectLacks(t *testing.T) {
	dir := t.TempDir()
	w := config.FrameworkWorker{Command: "vendor/nativephp/php-bin/bin/host/php artisan native:jump"}

	msg := missingWorkerProgramMsg("jump", dir, w)
	if msg == "" {
		t.Fatal("expected a refusal for a program that is not there")
	}
	if !strings.Contains(msg, "vendor/nativephp/php-bin/bin/host/php") {
		t.Errorf("message = %q, want it to name the missing program", msg)
	}
}

func TestMissingWorkerProgramMsg_silentOnceItExists(t *testing.T) {
	dir := t.TempDir()
	program := filepath.Join(dir, "vendor", "bin", "runner")
	if err := os.MkdirAll(filepath.Dir(program), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(program, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	w := config.FrameworkWorker{Command: "vendor/bin/runner --flag"}
	if msg := missingWorkerProgramMsg("runner", dir, w); msg != "" {
		t.Errorf("msg = %q, want none once the program is installed", msg)
	}
}

// A bare name is resolved through PATH and an absolute path belongs to the host,
// so neither is lerd's to second-guess: php artisan queue:work must never be
// refused because there is no ./php in the project.
func TestMissingWorkerProgramMsg_leavesPathAndAbsoluteAlone(t *testing.T) {
	dir := t.TempDir()
	for _, cmd := range []string{"php artisan queue:work", "/usr/bin/node server.js", "npm run dev"} {
		if msg := missingWorkerProgramMsg("w", dir, config.FrameworkWorker{Command: cmd}); msg != "" {
			t.Errorf("command %q was refused: %s", cmd, msg)
		}
	}
	if msg := missingWorkerProgramMsg("w", dir, config.FrameworkWorker{}); msg != "" {
		t.Errorf("empty command was refused: %s", msg)
	}
}
