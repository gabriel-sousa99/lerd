package envfile

import (
	"path/filepath"
	"testing"
)

func TestSyncFrontendAPIBase_RewritesPresentKeysToUnifiedOrigin(t *testing.T) {
	envPath := writeEnv(t, "URL_API=http://localhost:8000\nDB_HOST=oracle\n")
	dir := filepath.Dir(envPath)
	if err := SyncFrontendAPIBase(dir, "gestao-clientes.localhost", true); err != nil {
		t.Fatalf("SyncFrontendAPIBase: %v", err)
	}
	if got := ReadKey(envPath, "URL_API"); got != "https://gestao-clientes.localhost" {
		t.Errorf("URL_API = %q, want https://gestao-clientes.localhost", got)
	}
	if got := ReadKey(envPath, "DB_HOST"); got != "oracle" {
		t.Errorf("DB_HOST mexido: %q", got)
	}
}

func TestSyncFrontendAPIBase_NoApiPathSuffix(t *testing.T) {
	envPath := writeEnv(t, "VITE_API_URL=http://localhost:8000/api\n")
	dir := filepath.Dir(envPath)
	if err := SyncFrontendAPIBase(dir, "app.localhost", false); err != nil {
		t.Fatal(err)
	}
	if got := ReadKey(envPath, "VITE_API_URL"); got != "http://app.localhost" {
		t.Errorf("VITE_API_URL = %q, want http://app.localhost (sem /api)", got)
	}
}

func TestSyncFrontendAPIBase_OnlyTouchesKeysInSet(t *testing.T) {
	envPath := writeEnv(t, "VITE_SOMETHING=keep\nURL_API=old\n")
	dir := filepath.Dir(envPath)
	if err := SyncFrontendAPIBase(dir, "x.localhost", true); err != nil {
		t.Fatal(err)
	}
	if got := ReadKey(envPath, "VITE_SOMETHING"); got != "keep" {
		t.Errorf("chave fora do set foi tocada: %q", got)
	}
}

func TestSyncFrontendAPIBase_RewritesAllKeysInSet(t *testing.T) {
	envPath := writeEnv(t, "URL_API=a\nVITE_API_URL=b\nVITE_APP_API_URL=c\n")
	dir := filepath.Dir(envPath)
	if err := SyncFrontendAPIBase(dir, "app.localhost", true); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"URL_API", "VITE_API_URL", "VITE_APP_API_URL"} {
		if got := ReadKey(envPath, k); got != "https://app.localhost" {
			t.Errorf("%s = %q, want https://app.localhost", k, got)
		}
	}
}

func TestSyncFrontendAPIBase_NoEnvIsNoop(t *testing.T) {
	dir := t.TempDir() // sem .env
	if err := SyncFrontendAPIBase(dir, "x.localhost", true); err != nil {
		t.Errorf("esperado no-op, veio erro: %v", err)
	}
}

func TestSyncFrontendAPIBase_AbsentKeyNotAdded(t *testing.T) {
	envPath := writeEnv(t, "DB_HOST=oracle\n")
	dir := filepath.Dir(envPath)
	if err := SyncFrontendAPIBase(dir, "x.localhost", true); err != nil {
		t.Fatal(err)
	}
	if got := ReadKey(envPath, "URL_API"); got != "" {
		t.Errorf("URL_API criado indevidamente: %q", got)
	}
}
