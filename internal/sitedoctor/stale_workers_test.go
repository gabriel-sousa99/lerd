package sitedoctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

func writeUnit(t *testing.T, name string) {
	t.Helper()
	dir := config.SystemdUserDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte("[Service]\n"), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

// A definition that retires a worker takes it out of the list and off the
// dashboard, but nothing takes its unit off disk, where it stays armed for boot.
// The doctor is the one surface that asks whether a unit still answers to
// something, so it has to name the ones that do not.
func TestCheckStaleWorkers_reportsARetiredWorker(t *testing.T) {
	site := registerSite(t, config.Site{Name: "shop", Domains: []string{"shop.test"}})
	writeUnit(t, "lerd-queue-shop.service")
	writeUnit(t, "lerd-jump-shop.service")

	fw := &config.Framework{Workers: map[string]config.FrameworkWorker{"queue": {}}}
	c, ok := checkStaleWorkers(site.Path, fw)
	if !ok {
		t.Fatal("checkStaleWorkers did not run")
	}
	if c.Status != StatusWarn {
		t.Fatalf("status = %q, want %q (detail: %s)", c.Status, StatusWarn, c.Detail)
	}
	if !strings.Contains(c.Detail, "jump") || strings.Contains(c.Detail, "queue") {
		t.Errorf("detail = %q, want it to name jump and not queue", c.Detail)
	}
	if c.Fix != FixStaleWorkers {
		t.Errorf("fix = %q, want %q", c.Fix, FixStaleWorkers)
	}
}

// Every unit answering to a declared worker is the ordinary state of an install,
// and a check that warns there would warn on every site forever.
func TestCheckStaleWorkers_quietWhenEveryUnitIsDeclared(t *testing.T) {
	site := registerSite(t, config.Site{Name: "shop", Domains: []string{"shop.test"}})
	writeUnit(t, "lerd-queue-shop.service")
	writeUnit(t, "lerd-nginx.service")

	fw := &config.Framework{Workers: map[string]config.FrameworkWorker{"queue": {}}}
	c, ok := checkStaleWorkers(site.Path, fw)
	if !ok || c.Status != StatusOK {
		t.Errorf("check = %+v (ok=%v), want an OK status", c, ok)
	}
}

// With no framework resolved and no custom workers there is nothing to compare
// against, so the check has to stand down rather than call every unit stale.
func TestCheckStaleWorkers_skippedWithoutADefinition(t *testing.T) {
	site := registerSite(t, config.Site{Name: "shop", Domains: []string{"shop.test"}})
	writeUnit(t, "lerd-queue-shop.service")

	if c, ok := checkStaleWorkers(site.Path, nil); ok {
		t.Errorf("checkStaleWorkers ran and returned %+v, want it skipped", c)
	}
}
