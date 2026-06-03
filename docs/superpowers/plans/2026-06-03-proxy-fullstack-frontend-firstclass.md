# Proxy Fullstack — Frontend como cidadão de 1ª classe — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fazer o `.env` do projeto frontend (SPA) ser sincronizado para a origem unificada do proxy fullstack, eliminando o `URL_API` cross-origin hardcoded que quebra Sanctum/CORS.

**Architecture:** Espelhar a máquina já existente do lado API. Um set fixo e bounded de chaves de API-base (`FrontendAPIBaseKeys`) é reescrito *só se já presente* no `.env` apontado por `p.Path`, com a mesma garantia de escopo de `DomainScopedKeys`. O gatilho reusa `syncProxyEnv`/`unbindSitesEnv` em `internal/proxyops`; a reversão grava string vazia (relativa/neutra). Defaults de rota ficam mais ricos (rotas SSO do `unimedvr/core`). `--path` é liberado standalone. Hardening do gateway WSL extrai a seleção de IP para uma função pura testável.

**Tech Stack:** Go 1.x, módulo `github.com/gabriel-sousa99/lerd`. Testes `go test` padrão (sem libs externas). CLI via cobra.

---

## File Structure

| Arquivo | Responsabilidade | Ação |
|---------|------------------|------|
| `internal/envfile/envfile.go` | `FrontendAPIBaseKeys` + `SyncFrontendAPIBase` + `RevertFrontendAPIBase` | Modificar |
| `internal/envfile/frontend_test.go` | Testes do env-sync do frontend | Criar |
| `internal/proxyops/env.go` | Gatilho: `syncProxyEnv` cobre `p.Path`; vars injetáveis frontend | Modificar |
| `internal/proxyops/remove.go` | Reversão do frontend no rm | Modificar |
| `internal/proxyops/update.go` | Reversão do frontend ao limpar/trocar `p.Path` | Modificar |
| `internal/proxyops/env_frontend_test.go` | Testes do gatilho/reversão frontend | Criar |
| `internal/cli/proxy.go` | `defaultAPIPaths` mais ricos; help de `--path`; `fullstackHint` cita frontend | Modificar |
| `internal/cli/proxy_paths_test.go` | Teste dos novos defaults de rota | Criar |
| `internal/podman/hosts.go` | Extrair `pickLANIP` puro (hardening WSL) | Modificar |
| `internal/podman/hosts_lan_test.go` | Testes de seleção de LAN IP | Criar |
| `docs/features/proxy-fullstack-frontend.md` | Doc: HMR snippet, TrustProxies, rotas custom | Criar |
| `CHANGELOG.md` | Entrada oracle.next | Modificar |

---

## Task 1: `SyncFrontendAPIBase` — reescrever API-base do frontend

**Files:**
- Modify: `internal/envfile/envfile.go` (append após `SyncPrimaryDomain`, ~linha 282)
- Test: `internal/envfile/frontend_test.go`

- [ ] **Step 1: Write the failing test**

Criar `internal/envfile/frontend_test.go`:

```go
package envfile

import (
	"os"
	"path/filepath"
	"testing"
)

func writeEnv(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, ".env")
	if err := os.WriteFile(p, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestSyncFrontendAPIBase_RewritesPresentKeysToUnifiedOrigin(t *testing.T) {
	dir := writeEnv(t, "URL_API=http://localhost:8000\nDB_HOST=oracle\n")
	if err := SyncFrontendAPIBase(dir, "gestao-clientes.localhost", true); err != nil {
		t.Fatalf("SyncFrontendAPIBase: %v", err)
	}
	if got := ReadKey(filepath.Join(dir, ".env"), "URL_API"); got != "https://gestao-clientes.localhost" {
		t.Errorf("URL_API = %q, want https://gestao-clientes.localhost", got)
	}
	if got := ReadKey(filepath.Join(dir, ".env"), "DB_HOST"); got != "oracle" {
		t.Errorf("DB_HOST mexido: %q", got)
	}
}

func TestSyncFrontendAPIBase_NoApiPathSuffix(t *testing.T) {
	dir := writeEnv(t, "VITE_API_URL=http://localhost:8000/api\n")
	if err := SyncFrontendAPIBase(dir, "app.localhost", false); err != nil {
		t.Fatal(err)
	}
	if got := ReadKey(filepath.Join(dir, ".env"), "VITE_API_URL"); got != "http://app.localhost" {
		t.Errorf("VITE_API_URL = %q, want http://app.localhost (sem /api)", got)
	}
}

func TestSyncFrontendAPIBase_OnlyTouchesKeysInSet(t *testing.T) {
	dir := writeEnv(t, "VITE_SOMETHING=keep\nURL_API=old\n")
	if err := SyncFrontendAPIBase(dir, "x.localhost", true); err != nil {
		t.Fatal(err)
	}
	if got := ReadKey(filepath.Join(dir, ".env"), "VITE_SOMETHING"); got != "keep" {
		t.Errorf("chave fora do set foi tocada: %q", got)
	}
}

func TestSyncFrontendAPIBase_NoEnvIsNoop(t *testing.T) {
	dir := t.TempDir() // sem .env
	if err := SyncFrontendAPIBase(dir, "x.localhost", true); err != nil {
		t.Errorf("esperado no-op, veio erro: %v", err)
	}
}

func TestSyncFrontendAPIBase_AbsentKeyNotAdded(t *testing.T) {
	dir := writeEnv(t, "DB_HOST=oracle\n")
	if err := SyncFrontendAPIBase(dir, "x.localhost", true); err != nil {
		t.Fatal(err)
	}
	if got := ReadKey(filepath.Join(dir, ".env"), "URL_API"); got != "" {
		t.Errorf("URL_API criado indevidamente: %q", got)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/envfile/ -run TestSyncFrontendAPIBase -v`
Expected: FAIL — `undefined: SyncFrontendAPIBase` / `FrontendAPIBaseKeys`.

- [ ] **Step 3: Write minimal implementation**

Em `internal/envfile/envfile.go`, ao final do arquivo (após `SyncPrimaryDomain`):

```go
// FrontendAPIBaseKeys lista as chaves de .env que apontam a base da API num
// projeto frontend (SPA). Só são reescritas se já existirem — mesma semântica
// e garantia de escopo de DomainScopedKeys. Nada fora deste set é tocado.
//
// Exportado para que callers (testes, auditorias) possam provar que o escopo
// é bounded. Projetos com outra convenção de chave devem adicioná-la aqui.
var FrontendAPIBaseKeys = []string{
	"URL_API",          // Quasar (gestao-clientes-spa)
	"VITE_API_URL",     // Vite genérico
	"VITE_APP_API_URL", // Vite/Vue convenção comum
}

// SyncFrontendAPIBase reescreve as chaves de FrontendAPIBaseKeys presentes no
// .env do projeto frontend para a origem unificada (scheme://domain, SEM /api —
// a SPA concatena seus próprios prefixos). Só toca chaves existentes,
// idempotente, best-effort se não houver .env.
func SyncFrontendAPIBase(projectPath, domain string, secured bool) error {
	envPath := filepath.Join(projectPath, ".env")
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		return nil
	}

	keys, err := ReadKeys(envPath)
	if err != nil {
		return err
	}
	present := make(map[string]bool, len(keys))
	for _, k := range keys {
		present[k] = true
	}

	scheme := "http"
	if secured {
		scheme = "https"
	}
	url := scheme + "://" + domain

	updates := map[string]string{}
	for _, k := range FrontendAPIBaseKeys {
		if present[k] {
			updates[k] = url
		}
	}
	if len(updates) == 0 {
		return nil
	}
	return ApplyUpdates(envPath, updates)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/envfile/ -run TestSyncFrontendAPIBase -v`
Expected: PASS (5 testes).

- [ ] **Step 5: Commit**

```bash
git add internal/envfile/envfile.go internal/envfile/frontend_test.go
git commit -m "feat(envfile): SyncFrontendAPIBase aponta API-base da SPA para origem unificada"
```

---

## Task 2: `RevertFrontendAPIBase` — reverter API-base no unbind/rm

**Files:**
- Modify: `internal/envfile/envfile.go` (após `SyncFrontendAPIBase`)
- Test: `internal/envfile/frontend_test.go` (append)

> **Decisão (spec §3.2, recomendação):** a reversão grava **string vazia** (`URL_API=`). O lerd não conhece a URL de dev original (`localhost:8000`); um valor relativo/vazio é neutro e não-quebrável, e a doc instrui o dev a reconfigurar para rodar fora do proxy.

- [ ] **Step 1: Write the failing test**

Append em `internal/envfile/frontend_test.go`:

```go
func TestRevertFrontendAPIBase_BlanksPresentKeys(t *testing.T) {
	dir := writeEnv(t, "URL_API=https://app.localhost\nDB_HOST=oracle\n")
	if err := RevertFrontendAPIBase(dir); err != nil {
		t.Fatalf("RevertFrontendAPIBase: %v", err)
	}
	if got := ReadKey(filepath.Join(dir, ".env"), "URL_API"); got != "" {
		t.Errorf("URL_API = %q, want vazio", got)
	}
	if got := ReadKey(filepath.Join(dir, ".env"), "DB_HOST"); got != "oracle" {
		t.Errorf("DB_HOST mexido: %q", got)
	}
}

func TestRevertFrontendAPIBase_NoEnvIsNoop(t *testing.T) {
	dir := t.TempDir()
	if err := RevertFrontendAPIBase(dir); err != nil {
		t.Errorf("esperado no-op, veio erro: %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/envfile/ -run TestRevertFrontendAPIBase -v`
Expected: FAIL — `undefined: RevertFrontendAPIBase`.

- [ ] **Step 3: Write minimal implementation**

Em `internal/envfile/envfile.go`, após `SyncFrontendAPIBase`:

```go
// RevertFrontendAPIBase grava string vazia nas chaves de FrontendAPIBaseKeys
// presentes no .env do frontend, desfazendo o sync para a origem unificada.
// O lerd não conhece a URL de dev original, então um valor vazio (relativo,
// neutro) é o reset seguro. Só toca chaves existentes; no-op sem .env.
func RevertFrontendAPIBase(projectPath string) error {
	envPath := filepath.Join(projectPath, ".env")
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		return nil
	}
	keys, err := ReadKeys(envPath)
	if err != nil {
		return err
	}
	present := make(map[string]bool, len(keys))
	for _, k := range keys {
		present[k] = true
	}
	updates := map[string]string{}
	for _, k := range FrontendAPIBaseKeys {
		if present[k] {
			updates[k] = ""
		}
	}
	if len(updates) == 0 {
		return nil
	}
	return ApplyUpdates(envPath, updates)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/envfile/ -run TestRevertFrontendAPIBase -v`
Expected: PASS (2 testes).

- [ ] **Step 5: Commit**

```bash
git add internal/envfile/envfile.go internal/envfile/frontend_test.go
git commit -m "feat(envfile): RevertFrontendAPIBase zera API-base da SPA no unbind"
```

---

## Task 3: Gatilho — `syncProxyEnv` cobre o frontend (`p.Path`)

**Files:**
- Modify: `internal/proxyops/env.go:9` (vars injetáveis) e `:31-41` (`syncProxyEnv`)
- Test: `internal/proxyops/env_frontend_test.go`

- [ ] **Step 1: Write the failing test**

Criar `internal/proxyops/env_frontend_test.go`:

```go
package proxyops

import (
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

func TestSyncProxyEnv_SyncsFrontendWhenPathSet(t *testing.T) {
	origFind, origSync, origFront := findSiteFn, syncEnvFn, syncFrontendFn
	defer func() { findSiteFn, syncEnvFn, syncFrontendFn = origFind, origSync, origFront }()

	findSiteFn = func(name string) (*config.Site, error) {
		return &config.Site{Name: name, Path: "/srv/" + name}, nil
	}
	syncEnvFn = func(path, domain string, secured bool) error { return nil }

	var frontPath, frontDomain string
	var frontSecured bool
	syncFrontendFn = func(path, domain string, secured bool) error {
		frontPath, frontDomain, frontSecured = path, domain, secured
		return nil
	}

	p := config.Proxy{
		Domains: []string{"gestao-clientes.localhost"}, UpstreamPort: 9000,
		Path: "/srv/spa", Secured: true,
		Routes: []config.Route{{Path: "/api", Site: "gc-api"}},
	}
	if err := syncProxyEnv(p); err != nil {
		t.Fatalf("syncProxyEnv: %v", err)
	}
	if frontPath != "/srv/spa" || frontDomain != "gestao-clientes.localhost" || !frontSecured {
		t.Errorf("frontend sync = (%q, %q, %v)", frontPath, frontDomain, frontSecured)
	}
}

func TestSyncProxyEnv_NoFrontendWhenPathEmpty(t *testing.T) {
	origFind, origSync, origFront := findSiteFn, syncEnvFn, syncFrontendFn
	defer func() { findSiteFn, syncEnvFn, syncFrontendFn = origFind, origSync, origFront }()

	findSiteFn = func(name string) (*config.Site, error) {
		return &config.Site{Name: name, Path: "/srv/" + name}, nil
	}
	syncEnvFn = func(path, domain string, secured bool) error { return nil }

	called := false
	syncFrontendFn = func(path, domain string, secured bool) error { called = true; return nil }

	p := config.Proxy{
		Domains: []string{"x.localhost"}, UpstreamPort: 9000, Secured: true,
		Routes: []config.Route{{Path: "/api", Site: "api"}},
	}
	if err := syncProxyEnv(p); err != nil {
		t.Fatal(err)
	}
	if called {
		t.Error("syncFrontendFn não deveria ser chamado sem p.Path")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/proxyops/ -run TestSyncProxyEnv -v`
Expected: FAIL — `undefined: syncFrontendFn`.

- [ ] **Step 3: Write minimal implementation**

Em `internal/proxyops/env.go`, adicionar a var injetável após a linha 9:

```go
// syncEnvFn is injectable for tests. Production wires to envfile.SyncPrimaryDomain.
var syncEnvFn = envfile.SyncPrimaryDomain

// Frontend env-sync hooks, injectable for tests.
var (
	syncFrontendFn   = envfile.SyncFrontendAPIBase
	revertFrontendFn = envfile.RevertFrontendAPIBase
)
```

Substituir o corpo de `syncProxyEnv` (linhas 31-41):

```go
// syncProxyEnv points the .env of every site bound to p at the unified domain
// (p's primary domain), so sessions/cookies are first-party. When p.Path is
// set (the frontend project folder), its API-base keys are pointed at the same
// unified origin so the SPA stops calling a cross-origin API. Best-effort: a
// missing site or .env is skipped without failing the proxy operation.
func syncProxyEnv(p config.Proxy) error {
	domain := p.PrimaryDomain()
	for _, name := range boundSites(p) {
		s, err := findSiteFn(name)
		if err != nil {
			continue
		}
		_ = syncEnvFn(s.Path, domain, p.Secured)
	}
	if p.Path != "" {
		_ = syncFrontendFn(p.Path, domain, p.Secured)
	}
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/proxyops/ -run TestSyncProxyEnv -v`
Expected: PASS (2 novos + `TestSyncProxyEnv_PointsSitesToUnifiedDomain` existente).

- [ ] **Step 5: Commit**

```bash
git add internal/proxyops/env.go internal/proxyops/env_frontend_test.go
git commit -m "feat(proxyops): syncProxyEnv sincroniza API-base do frontend (p.Path)"
```

---

## Task 4: Reversão no `rm` — `Remove` zera o frontend

**Files:**
- Modify: `internal/proxyops/remove.go:30`
- Test: `internal/proxyops/env_frontend_test.go` (append)

- [ ] **Step 1: Write the failing test**

Append em `internal/proxyops/env_frontend_test.go`:

```go
func TestRemove_RevertsFrontend(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	origFind, origSync, origReload, origUnsec := findSiteFn, syncEnvFn, nginxReloadFn, unsecureCertFn
	origFront, origRevert := syncFrontendFn, revertFrontendFn
	defer func() {
		findSiteFn, syncEnvFn, nginxReloadFn, unsecureCertFn = origFind, origSync, origReload, origUnsec
		syncFrontendFn, revertFrontendFn = origFront, origRevert
	}()
	findSiteFn = func(name string) (*config.Site, error) {
		return &config.Site{Name: name, Path: "/srv/" + name, Domains: []string{name + ".localhost"}}, nil
	}
	syncEnvFn = func(path, domain string, secured bool) error { return nil }
	syncFrontendFn = func(path, domain string, secured bool) error { return nil }
	nginxReloadFn = func() error { return nil }
	unsecureCertFn = func(p config.Proxy) error { return nil }
	secureCertFn = func(p config.Proxy) error { return nil }

	var reverted string
	revertFrontendFn = func(path string) error { reverted = path; return nil }

	p, err := Add(AddOptions{
		Domain: "gc.localhost", Port: 9000, Path: t.TempDir(),
		Routes: []config.Route{{Path: "/api", Site: "gc-api"}},
	})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := Remove(p.Name); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if reverted != p.Path {
		t.Errorf("reverted = %q, want %q", reverted, p.Path)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/proxyops/ -run TestRemove_RevertsFrontend -v`
Expected: FAIL — `reverted = "", want <tmpdir>` (revert não chamado).

- [ ] **Step 3: Write minimal implementation**

Em `internal/proxyops/remove.go`, dentro de `Remove`, após `unbindSitesEnv(sites)` (linha 30):

```go
	_ = nginxReloadFn()
	unbindSitesEnv(sites)
	if p.Path != "" {
		_ = revertFrontendFn(p.Path)
	}
	return nil
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/proxyops/ -run TestRemove_RevertsFrontend -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/proxyops/remove.go internal/proxyops/env_frontend_test.go
git commit -m "feat(proxyops): rm reverte API-base do frontend"
```

---

## Task 5: Reversão no `edit` — `Update` reverte path antigo ao limpar/trocar

**Files:**
- Modify: `internal/proxyops/update.go:94` (capturar `oldPath`) e `:128-134` (revert após sync)
- Test: `internal/proxyops/env_frontend_test.go` (append)

- [ ] **Step 1: Write the failing test**

Append em `internal/proxyops/env_frontend_test.go`:

```go
func TestUpdate_RevertsFrontendOnPathClear(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	origFind, origSync, origReload := findSiteFn, syncEnvFn, nginxReloadFn
	origFront, origRevert := syncFrontendFn, revertFrontendFn
	defer func() {
		findSiteFn, syncEnvFn, nginxReloadFn = origFind, origSync, origReload
		syncFrontendFn, revertFrontendFn = origFront, origRevert
	}()
	findSiteFn = func(name string) (*config.Site, error) {
		return &config.Site{Name: name, Path: "/srv/" + name, Domains: []string{name + ".localhost"}}, nil
	}
	syncEnvFn = func(path, domain string, secured bool) error { return nil }
	syncFrontendFn = func(path, domain string, secured bool) error { return nil }
	nginxReloadFn = func() error { return nil }
	secureCertFn = func(p config.Proxy) error { return nil }

	var reverted string
	revertFrontendFn = func(path string) error { reverted = path; return nil }

	spaDir := t.TempDir()
	p, err := Add(AddOptions{
		Domain: "gc.localhost", Port: 9000, Path: spaDir,
		Routes: []config.Route{{Path: "/api", Site: "gc-api"}},
	})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	empty := ""
	if _, err := Update(p.Name, UpdateOptions{Path: &empty}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if reverted != spaDir {
		t.Errorf("reverted = %q, want %q", reverted, spaDir)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/proxyops/ -run TestUpdate_RevertsFrontendOnPathClear -v`
Expected: FAIL — `reverted = "", want <spaDir>`.

- [ ] **Step 3: Write minimal implementation**

Em `internal/proxyops/update.go`, capturar o path antigo junto de `oldSites` (linha 94):

```go
	oldSites := boundSites(*existing)
	oldPath := existing.Path
```

Depois, no bloco de reconciliação de `.env` (atual linhas 121-134), adicionar a reversão do frontend após o `unbindSitesEnv(removed)`:

```go
	newSites := map[string]bool{}
	for _, s := range boundSites(updated) {
		newSites[s] = true
	}
	if updated.IsFullstack() {
		_ = syncProxyEnv(updated)
	}
	var removed []string
	for _, s := range oldSites {
		if !newSites[s] {
			removed = append(removed, s)
		}
	}
	unbindSitesEnv(removed)

	// Frontend: se o path mudou ou foi limpo, reverte a API-base do path
	// antigo. O path novo (se houver e ainda fullstack) já foi sincronizado
	// por syncProxyEnv acima.
	if oldPath != "" && oldPath != updated.Path {
		_ = revertFrontendFn(oldPath)
	}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/proxyops/ -run TestUpdate_RevertsFrontend -v`
Expected: PASS.

- [ ] **Step 5: Run the full proxyops suite (regression)**

Run: `go test ./internal/proxyops/ -v`
Expected: PASS — incluindo `TestAdd_Fullstack_*`, `TestUpdate_*`, `TestRemove_*` existentes.

- [ ] **Step 6: Commit**

```bash
git add internal/proxyops/update.go internal/proxyops/env_frontend_test.go
git commit -m "feat(proxyops): edit reverte API-base do frontend ao limpar/trocar path"
```

---

## Task 6: Defaults de rota mais ricos (rotas SSO do `unimedvr/core`)

**Files:**
- Modify: `internal/cli/proxy.go:15-17` (`defaultAPIPaths`)
- Test: `internal/cli/proxy_paths_test.go`

- [ ] **Step 1: Write the failing test**

Criar `internal/cli/proxy_paths_test.go`:

```go
package cli

import "testing"

func TestDefaultAPIPaths_IncludesSSOAndConventions(t *testing.T) {
	got := defaultAPIPaths()
	want := []string{
		"/api", "/sanctum", "/broadcasting", "/storage",
		"/redirect", "/authenticate", "/login", "/logout", "/up",
	}
	if len(got) != len(want) {
		t.Fatalf("got %d paths %v, want %d %v", len(got), got, len(want), want)
	}
	set := map[string]bool{}
	for _, p := range got {
		set[p] = true
	}
	for _, w := range want {
		if !set[w] {
			t.Errorf("default paths não contém %q", w)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/cli/ -run TestDefaultAPIPaths -v`
Expected: FAIL — `got 4 paths, want 9`.

- [ ] **Step 3: Write minimal implementation**

Substituir `defaultAPIPaths` em `internal/cli/proxy.go`:

```go
func defaultAPIPaths() []string {
	return []string{
		"/api", "/sanctum", "/broadcasting", "/storage",
		"/redirect",     // Core/Routes/web.php GET /redirect/{profile?} → entrypoint SSO (alvo do 401 da SPA)
		"/authenticate", // Core/Routes/web.php GET /authenticate/{profile?} + api.php POST → callback SSO
		"/login", "/logout", "/up", // convenções Laravel/Breeze + healthcheck
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/cli/ -run TestDefaultAPIPaths -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/cli/proxy.go internal/cli/proxy_paths_test.go
git commit -m "feat(cli): defaults de rota cobrem /redirect /authenticate (SSO unimedvr/core)"
```

---

## Task 7: `--path` standalone + `fullstackHint` cita o frontend

**Files:**
- Modify: `internal/cli/proxy.go:128` e `:197` (help de `--path`), `:54-61` (`fullstackHint`)

> **Nota:** a aceitação de `--path` sem `--managed` **já funciona** no nível `proxyops.Add` (só erra quando `Managed && Path==""`, `add.go:50`). Esta task ajusta o texto de ajuda enganoso e enriquece a dica. Sem mudança de lógica.

- [ ] **Step 1: Update do help text de `--path` no add**

Em `internal/cli/proxy.go`, linha 128, trocar:

```go
	c.Flags().StringVar(&path, "path", "", "Pasta do projeto frontend (SPA). Obrigatória com --managed; standalone habilita o sync do .env da SPA")
```

- [ ] **Step 2: Update do help text de `--path` no edit**

Em `internal/cli/proxy.go`, linha 197, trocar:

```go
	c.Flags().StringVar(&path, "path", "", "Nova pasta do projeto frontend (string vazia limpa e reverte o .env da SPA)")
```

- [ ] **Step 3: `fullstackHint` cita o frontend**

Substituir `fullstackHint` (linhas 54-61):

```go
// fullstackHint returns an advisory message printed after creating/editing a
// fullstack proxy so the user knows the API site's .env (and, when --path is
// set, the SPA's API-base key) was pointed at the unified domain.
func fullstackHint(domain string, secured bool) string {
	scheme := "http"
	if secured {
		scheme = "https"
	}
	return fmt.Sprintf("dica: os .env do site de API (APP_URL/SANCTUM_STATEFUL_DOMAINS) e da SPA "+
		"(URL_API/VITE_API_URL, se presentes) foram apontados para %s://%s. "+
		"Para HMR atrás do proxy, ajuste devServer.hmr no quasar.config.js/vite.config (veja docs).", scheme, domain)
}
```

- [ ] **Step 4: Build + verify**

Run: `go build ./... && go test ./internal/cli/ -v`
Expected: build OK, testes do pacote `cli` PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/cli/proxy.go
git commit -m "feat(cli): help de --path standalone e fullstackHint citando a SPA"
```

---

## Task 8: Hardening WSL — extrair `pickLANIP` puro e testar

**Files:**
- Modify: `internal/podman/hosts.go:281-305` (`primaryLANIP`)
- Test: `internal/podman/hosts_lan_test.go`

> **Spec §5.2:** cobrir com teste iface só-loopback, múltiplas ifaces, eth0 ausente. `net.Interfaces()`/`net.Dial` não são testáveis diretamente; extraímos a seleção para uma função pura `pickLANIP` que recebe a lista de ifaces já materializada.

- [ ] **Step 1: Write the failing test**

Criar `internal/podman/hosts_lan_test.go`:

```go
package podman

import (
	"net"
	"testing"
)

func cidr(t *testing.T, s string) net.Addr {
	t.Helper()
	ip, ipnet, err := net.ParseCIDR(s)
	if err != nil {
		t.Fatal(err)
	}
	ipnet.IP = ip
	return ipnet
}

func TestPickLANIP_SkipsLoopbackAndDown(t *testing.T) {
	ifaces := []lanIface{
		{up: true, loopback: true, addrs: []net.Addr{cidr(t, "127.0.0.1/8")}},
		{up: false, loopback: false, addrs: []net.Addr{cidr(t, "10.0.0.5/24")}}, // down
		{up: true, loopback: false, addrs: []net.Addr{cidr(t, "172.20.0.3/20")}}, // eth0 WSL
	}
	if got := pickLANIP(ifaces); got != "172.20.0.3" {
		t.Errorf("pickLANIP = %q, want 172.20.0.3", got)
	}
}

func TestPickLANIP_LoopbackOnlyReturnsEmpty(t *testing.T) {
	ifaces := []lanIface{
		{up: true, loopback: true, addrs: []net.Addr{cidr(t, "127.0.0.1/8")}},
	}
	if got := pickLANIP(ifaces); got != "" {
		t.Errorf("pickLANIP = %q, want vazio", got)
	}
}

func TestPickLANIP_NoInterfacesReturnsEmpty(t *testing.T) {
	if got := pickLANIP(nil); got != "" {
		t.Errorf("pickLANIP = %q, want vazio", got)
	}
}

func TestHostCandidates_OrderAndDedup(t *testing.T) {
	got := hostCandidates("172.20.0.1", "172.20.0.1") // getent == lan: dedup
	want := []string{"172.20.0.1", "10.0.2.2"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("hostCandidates = %v, want %v", got, want)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/podman/ -run 'TestPickLANIP|TestHostCandidates' -v`
Expected: FAIL — `undefined: lanIface` / `pickLANIP`.

- [ ] **Step 3: Write minimal implementation**

Em `internal/podman/hosts.go`, substituir `primaryLANIP` (linhas 281-305) por:

```go
// lanIface is a testable projection of a network interface: the flags pickLANIP
// cares about plus its addresses. Built from net.Interface in primaryLANIP.
type lanIface struct {
	up       bool
	loopback bool
	addrs    []net.Addr
}

// pickLANIP returns the first non-loopback IPv4 address belonging to an
// interface that is up and not loopback. Pure (no syscalls) so the loopback /
// down / absent-eth0 cases are unit-testable. Returns "" when nothing matches.
func pickLANIP(ifaces []lanIface) string {
	for _, iface := range ifaces {
		if !iface.up || iface.loopback {
			continue
		}
		for _, addr := range iface.addrs {
			if ipnet, ok := addr.(*net.IPNet); ok {
				if v4 := ipnet.IP.To4(); v4 != nil && !v4.IsLoopback() {
					return v4.String()
				}
			}
		}
	}
	return ""
}

// primaryLANIP returns the local IPv4 address that the kernel would use to
// reach a public destination. Duplicates internal/dns/setup_common.go's
// helper because importing dns from podman would create a cycle. Falls back to
// scanning interfaces (via pickLANIP) when the UDP dial trick is unavailable.
func primaryLANIP() string {
	conn, err := net.Dial("udp4", "1.1.1.1:80")
	if err == nil {
		defer conn.Close()
		return conn.LocalAddr().(*net.UDPAddr).IP.String()
	}
	ifaces, ifErr := net.Interfaces()
	if ifErr != nil {
		return ""
	}
	projected := make([]lanIface, 0, len(ifaces))
	for _, iface := range ifaces {
		addrs, _ := iface.Addrs()
		projected = append(projected, lanIface{
			up:       iface.Flags&net.FlagUp != 0,
			loopback: iface.Flags&net.FlagLoopback != 0,
			addrs:    addrs,
		})
	}
	return pickLANIP(projected)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/podman/ -run 'TestPickLANIP|TestHostCandidates' -v`
Expected: PASS (4 testes).

- [ ] **Step 5: Run the podman suite (regression)**

Run: `go test ./internal/podman/`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/podman/hosts.go internal/podman/hosts_lan_test.go
git commit -m "refactor(podman): extrai pickLANIP puro + testes (hardening gateway WSL)"
```

---

## Task 9: Documentação — HMR, TrustProxies, rotas custom

**Files:**
- Create: `docs/features/proxy-fullstack-frontend.md`

- [ ] **Step 1: Escrever a doc**

Criar `docs/features/proxy-fullstack-frontend.md`:

````markdown
# Proxy Fullstack — Frontend como cidadão de 1ª classe

Ao registrar um proxy fullstack com `--path <pasta-da-spa>`, o lerd sincroniza a
**API-base** do `.env` da SPA para a origem unificada do proxy, eliminando o
`URL_API` cross-origin que quebra Sanctum/CORS.

## Uso

```bash
lerd proxy add gestao-clientes.localhost \
  --port 9000 \
  --path /caminho/para/gestao-clientes-spa \
  --api-site gestao-clientes-api
```

O que o lerd faz:

- **API (site Laravel):** aponta `APP_URL`, `SESSION_DOMAIN`,
  `SANCTUM_STATEFUL_DOMAINS`, … (`DomainScopedKeys`) para `https://gestao-clientes.localhost`.
- **Frontend (SPA):** aponta `URL_API` / `VITE_API_URL` / `VITE_APP_API_URL`
  (`FrontendAPIBaseKeys`, **só se já presentes**) para `https://gestao-clientes.localhost`
  (raiz da origem, **sem** `/api` — a SPA concatena seus próprios prefixos).

Só chaves já presentes são reescritas; nada fora desses sets é tocado.

## Chaves de API-base reconhecidas

`URL_API` (Quasar), `VITE_API_URL`, `VITE_APP_API_URL`. Projetos com outra
convenção devem adicionar a chave a `FrontendAPIBaseKeys` em
`internal/envfile/envfile.go` ou apontá-la manualmente.

## HMR atrás do proxy (não automatizado)

Quasar/Vite leem config de HMR de `quasar.config.js`/`vite.config`, **não** do
`.env` — fora do alcance do env-sync. Para o websocket de HMR funcionar via
HTTPS no proxy, configure manualmente:

```js
// quasar.config.js
devServer: {
  hmr: {
    clientPort: 443,
    protocol: 'wss',
  },
}
```

## Rotas custom

Os defaults de rota cobrem `/api /sanctum /broadcasting /storage /redirect
/authenticate /login /logout /up`. `/redirect` e `/authenticate` são as rotas
**web no root** do fluxo SSO do `unimedvr/core` (`401 → /redirect → provider →
/authenticate → sessão`). Rotas fora desse conjunto exigem `--api-path`
explícito (repetível):

```bash
lerd proxy add app.localhost --port 9000 --path ./spa --api-site app-api \
  --api-path /api --api-path /webhooks --api-path /redirect --api-path /authenticate
```

> **Atenção (app-side):** `/redirect` precisa **existir** na API. Apps que não
> montam o `core/web.php` referenciam a rota sem a definir → 404. O lerd só
> roteia; a rota é responsabilidade da aplicação.

## TrustProxies / HTTPS no Laravel

O template fastcgi do proxy força `HTTPS on`, `X-Forwarded-Proto` e
`X-Forwarded-Host`, então o Laravel emite cookie `Secure` e gera URLs https —
**o cookie Sanctum same-origin funciona sem TrustProxies.**

Porém, sem TrustProxies, `$request->getClientIp()` retorna o IP do container
nginx, não o do cliente. Se a app faz rate-limit ou log por IP, configure em
`bootstrap/app.php`:

```php
->withMiddleware(function (Middleware $middleware) {
    $middleware->trustProxies(at: '*'); // ou a sub-rede do container
})
```

Verifique também que `SESSION_SECURE_COOKIE` é coerente com o `secured` do proxy.

## Reversão

`lerd proxy rm` ou `lerd proxy edit --path ""` zera as chaves de API-base da SPA
(grava `URL_API=`). O lerd não conhece a URL de dev original; reconfigure
manualmente (ex.: `URL_API=http://localhost:8000`) para rodar a SPA fora do proxy.
````

- [ ] **Step 2: Commit**

```bash
git add docs/features/proxy-fullstack-frontend.md
git commit -m "docs: proxy fullstack frontend (HMR, TrustProxies, rotas SSO, reversão)"
```

---

## Task 10: CHANGELOG

**Files:**
- Modify: `CHANGELOG.md` (topo)

- [ ] **Step 1: Ler o topo do CHANGELOG para casar o formato**

Run: `head -40 CHANGELOG.md`
Verificar o padrão da última entrada (`oracle.25`) e o estilo de bullets.

- [ ] **Step 2: Adicionar a entrada `oracle.next`**

Inserir no topo da seção de versões, seguindo o formato existente (ajuste o número/data ao padrão do arquivo):

```markdown
## v1.21.2-oracle.next

### Added
- Proxy fullstack: frontend como cidadão de 1ª classe. `--path` sincroniza a
  API-base do `.env` da SPA (`URL_API`/`VITE_API_URL`/`VITE_APP_API_URL`, só se
  presentes) para a origem unificada, eliminando o cross-origin que quebrava
  Sanctum/CORS.
- Defaults de rota mais ricos: `/redirect`, `/authenticate`, `/login`,
  `/logout`, `/up` — cobre o fluxo SSO do `unimedvr/core`.
- `--path` aceito standalone (sem `--managed`) para habilitar o env-sync da SPA
  quando o dev roda `quasar dev` manualmente.
- Doc `docs/features/proxy-fullstack-frontend.md` (HMR, TrustProxies, rotas custom).

### Changed
- `fullstackHint` agora cita o `.env` da SPA e o snippet de HMR.

### Fixed
- Hardening da detecção do gateway WSL: seleção de LAN IP extraída para
  `pickLANIP` (testável; cobre iface só-loopback, ifaces down e eth0 ausente).

### Reverted
- `lerd proxy rm`/`edit --path ""` zera a API-base da SPA (valor neutro vazio).
```

- [ ] **Step 3: Commit**

```bash
git add CHANGELOG.md
git commit -m "docs: changelog oracle.next (proxy fullstack frontend first-class)"
```

---

## Verificação final (build + suíte completa)

- [ ] **Build e testes do projeto inteiro**

Run: `go build ./... && go test ./...`
Expected: build OK; todos os pacotes PASS (sem regressão em `envfile`, `proxyops`, `cli`, `podman`, `config`).

- [ ] **Lint/vet**

Run: `go vet ./...`
Expected: sem warnings novos.

---

## E2E manual (gestao-clientes) — checklist pós-merge

> Requer ambiente real (Podman + WSL + projeto gestao-clientes). Não automatizado.

- [ ] Criar proxy: `lerd proxy add gestao-clientes.localhost --port 9000 --path <spa> --api-site gestao-clientes-api`
- [ ] Conferir `.env` da SPA: `URL_API=https://gestao-clientes.localhost` (sem `/api`).
- [ ] Conferir `.env` da API: `APP_URL`/`SANCTUM_STATEFUL_DOMAINS` apontando ao domínio unificado.
- [ ] `GET /sanctum/csrf-cookie` → cookie setado em `gestao-clientes.localhost`.
- [ ] Login SSO: `401 → /redirect → provider → /authenticate → sessão`.
- [ ] `GET /api/user` autenticado **sem** CORS e **sem** `changeOrigin`.
- [ ] HMR: aplicar `devServer.hmr.clientPort=443/protocol=wss` no `quasar.config.js`; confirmar websocket conecta via `wss://gestao-clientes.localhost`.
- [ ] WSL: `host.containers.internal` resolve consistentemente no container nginx para o `proxy_pass` da base SPA; estável após reboot do WSL.
- [ ] `lerd proxy rm gestao-clientes.localhost` → `URL_API` da SPA volta a vazio.

---

## Self-Review (preenchido durante a escrita)

**Cobertura da spec:**
- §3.1 `FrontendAPIBaseKeys` + `SyncFrontendAPIBase` → Task 1 ✓
- §3.2 gatilho `syncProxyEnv` cobre `p.Path` → Task 3 ✓; reversão rm → Task 4 ✓; reversão edit → Task 5 ✓; decisão vazio vs backup → resolvida (vazio) Task 2 ✓
- §3.3 `--path` standalone → Task 7 (já funciona em proxyops; help ajustado) ✓
- §4 defaults de rota ricos → Task 6 ✓
- §5.1 TrustProxies (doc-only) → Task 9 ✓
- §5.2 hardening WSL + testes → Task 8 ✓
- §6 plano de testes Go → Tasks 1-8; E2E manual → checklist ✓
- §7 arquivos a tocar → todos mapeados em File Structure ✓

**Consistência de tipos:** `SyncFrontendAPIBase(projectPath, domain string, secured bool) error` e `RevertFrontendAPIBase(projectPath string) error` usados consistentemente em envfile, env.go (vars `syncFrontendFn`/`revertFrontendFn`), Remove, Update. `pickLANIP([]lanIface) string` consistente entre hosts.go e teste.

**Placeholders:** nenhum — todo passo tem código/comando concreto.
