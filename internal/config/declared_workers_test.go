package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeProject(t *testing.T, dir, yaml string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, ".lerd.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatalf("writing .lerd.yaml: %v", err)
	}
}

// A worker the framework definition names and a custom worker in .lerd.yaml are
// both declared, and so are the builtins lerd runs outside any definition.
func TestDeclaredWorkerNames_unionOfDefinitionAndProject(t *testing.T) {
	dir := t.TempDir()
	writeProject(t, dir, "framework: laravel\ncustom_workers:\n  mailer:\n    command: php mail\n")
	fw := &Framework{Workers: map[string]FrameworkWorker{"queue": {}, "vite": {}}}

	names, ok := DeclaredWorkerNames(Site{Name: "shop", Path: dir}, fw)
	if !ok {
		t.Fatal("ok = false, want true")
	}
	for _, want := range []string{"queue", "vite", "mailer", StripeWorkerName, HostProxyWorkerName} {
		if !names[want] {
			t.Errorf("names[%q] = false, want true", want)
		}
	}
	if names["jump"] {
		t.Error(`names["jump"] = true for a worker nothing declares`)
	}
}

// A gate that stops matching hides a worker from the list without removing it
// from the definition, so its unit must not be called stale: the next branch
// checkout brings the worker back.
func TestDeclaredWorkerNames_countsGatedWorkers(t *testing.T) {
	dir := t.TempDir()
	rule := FrameworkRule{File: "artisan"}
	fw := &Framework{Workers: map[string]FrameworkWorker{"reverb": {Check: &rule}}}

	names, ok := DeclaredWorkerNames(Site{Name: "shop", Path: dir}, fw)
	if !ok || !names["reverb"] {
		t.Fatalf("names[reverb] = %v (ok=%v), want true", names["reverb"], ok)
	}
}

// With no framework resolved and no custom workers there is nothing to compare
// against, and answering with an empty set would mark every unit on the machine
// stale. The caller has to be told it cannot decide.
func TestDeclaredWorkerNames_unresolvableFramework(t *testing.T) {
	dir := t.TempDir()
	if names, ok := DeclaredWorkerNames(Site{Name: "shop", Path: dir}, nil); ok || names != nil {
		t.Errorf("DeclaredWorkerNames = (%v, %v), want (nil, false)", names, ok)
	}
}
