# Proxy Fullstack — Plano 2: CLI + API HTTP + coerência de .env

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Permitir criar/editar/remover um proxy fullstack via CLI e API HTTP, com validação de proxy inteiro e sincronização mínima e coerente do `.env` do site de API para o domínio unificado.

**Architecture:** O Plano 1 já modela e gera o vhost. Aqui: (a) `Proxy.Validate()` estrutural chamado em `SaveProxies`; (b) `proxyops.Add/Update/Remove` ganham rotas + sync de `.env`; (c) CLI opinativa (`--api-site/--api-port/--api-path`); (d) API POST/PUT com `routes`/`site` + DTO; (e) coerência: o `.env` do site de API aponta para o domínio unificado enquanto vinculado, e volta ao próprio ao desvincular, com os pontos de sync do site cientes do vínculo.

**Tech Stack:** Go, cobra (CLI), net/http (API), `internal/envfile`.

**Decisões herdadas:** ver §11 do spec `docs/superpowers/specs/2026-06-01-proxy-fullstack-design.md`.

---

## File Structure

- `internal/config/proxy.go` (modify) — `Proxy.Validate()`; `FindFullstackProxyForSite`.
- `internal/config/proxy.go` SaveProxies (modify) — chama `Validate()` estrutural.
- `internal/proxyops/add.go` (modify) — `AddOptions` ganha `Site`/`Routes`; valida; sync de env pós-add.
- `internal/proxyops/update.go` (modify) — `UpdateOptions` ganha `Routes *[]config.Route` e `Site *string`; replace integral; sync/unsync de env.
- `internal/proxyops/remove.go` (modify) — re-sync do `.env` dos sites desvinculados.
- `internal/proxyops/env.go` (create) — helpers `syncProxyEnv`, `unbindSitesEnv`, `boundSites`.
- `internal/proxyops/env_test.go` (create) — testes dos helpers com `findSiteFn`/`syncEnvFn` stub.
- `internal/siteops/env.go` (create) — `EffectiveEnvDomain(site)` + `SyncSiteEnv(site)` (cientes do fullstack).
- `internal/cli/proxy.go` (modify) — flags `--api-site/--api-port/--api-path` em add/edit + hint advisory.
- `internal/cli/domain.go`, `internal/cli/env.go`, `internal/mcp/server.go`, `internal/siteops/secure.go` (modify) — rotear o sync do site por `siteops.SyncSiteEnv`.
- `internal/ui/proxy_api.go` (modify) — POST/PUT aceitam `routes`/`site`; `proxyDTO` expõe `routes`/`site`/`fullstack`.

---

## Task 1: `Proxy.Validate()` + `FindFullstackProxyForSite` + validação em SaveProxies

**Files:**
- Modify: `internal/config/proxy.go`
- Test: `internal/config/proxy_validate_test.go` (create)

- [ ] **Step 1: Teste falhando — create `internal/config/proxy_validate_test.go`**

```go
package config

import "testing"

func TestProxyValidate(t *testing.T) {
	cases := []struct {
		name    string
		p       Proxy
		wantErr bool
	}{
		{"simple ok", Proxy{Name: "a", Domains: []string{"a.localhost"}, UpstreamPort: 9000}, false},
		{"simple bad port", Proxy{Name: "a", Domains: []string{"a.localhost"}, UpstreamPort: 0}, true},
		{"simple with base site", Proxy{Name: "a", Domains: []string{"a.localhost"}, UpstreamPort: 9000, Site: "x"}, true},
		{"fullstack port base + site route", Proxy{Name: "a", Domains: []string{"a.localhost"}, UpstreamPort: 9000, Routes: []Route{{Path: "/api", Site: "x"}}}, false},
		{"fullstack site base", Proxy{Name: "a", Domains: []string{"a.localhost"}, Site: "spa", Routes: []Route{{Path: "/api", UpstreamPort: 8000}}}, false},
		{"fullstack base both", Proxy{Name: "a", Domains: []string{"a.localhost"}, UpstreamPort: 9000, Site: "spa", Routes: []Route{{Path: "/api", Site: "x"}}}, true},
		{"fullstack base neither", Proxy{Name: "a", Domains: []string{"a.localhost"}, Routes: []Route{{Path: "/api", Site: "x"}}}, true},
		{"fullstack bad route", Proxy{Name: "a", Domains: []string{"a.localhost"}, UpstreamPort: 9000, Routes: []Route{{Path: "api", Site: "x"}}}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.p.Validate(); (err != nil) != tc.wantErr {
				t.Errorf("Validate() err = %v, wantErr = %v", err, tc.wantErr)
			}
		})
	}
}

func TestFindFullstackProxyForSite(t *testing.T) {
	reg := &ProxyRegistry{Proxies: []Proxy{
		{Name: "ret", Domains: []string{"ret.localhost"}, UpstreamPort: 9000, Routes: []Route{{Path: "/api", Site: "ret-api"}}},
		{Name: "blog", Domains: []string{"blog.localhost"}, Site: "blog-site"},
		{Name: "plain", Domains: []string{"plain.localhost"}, UpstreamPort: 3000},
	}}
	if p, ok := findFullstackProxyForSiteIn(reg, "ret-api"); !ok || p.Name != "ret" {
		t.Errorf("route site: got %v ok=%v", p, ok)
	}
	if p, ok := findFullstackProxyForSiteIn(reg, "blog-site"); !ok || p.Name != "blog" {
		t.Errorf("base site: got %v ok=%v", p, ok)
	}
	if _, ok := findFullstackProxyForSiteIn(reg, "nope"); ok {
		t.Error("expected not found")
	}
}
```

- [ ] **Step 2: Rodar → FAIL** (`go test ./internal/config/ -run 'ProxyValidate|FindFullstack' -v`).

- [ ] **Step 3: Implementar em `internal/config/proxy.go`**

Adicionar após `ValidateProxyRoutes`:

```go
// Validate checks structural invariants of a whole proxy (no I/O — site
// existence is verified at the write path, not here). A simple proxy needs a
// valid base port and no base Site. A fullstack proxy (Site set or Routes
// present) needs exactly one base target and valid routes.
func (p Proxy) Validate() error {
	if !p.IsFullstack() {
		if p.Site != "" {
			return fmt.Errorf("proxy simples não pode ter site na base")
		}
		if p.UpstreamPort <= 0 || p.UpstreamPort > 65535 {
			return fmt.Errorf("porta inválida: %d", p.UpstreamPort)
		}
		return nil
	}
	hasSite := p.Site != ""
	hasPort := p.UpstreamPort != 0
	if hasSite == hasPort {
		return fmt.Errorf("base do fullstack precisa de exatamente um target (porta OU site)")
	}
	if hasPort && (p.UpstreamPort <= 0 || p.UpstreamPort > 65535) {
		return fmt.Errorf("porta inválida: %d", p.UpstreamPort)
	}
	return ValidateProxyRoutes(p.Routes)
}

// FindFullstackProxyForSite returns the fullstack proxy whose base Site or any
// route Site equals siteName, if any.
func FindFullstackProxyForSite(siteName string) (*Proxy, bool) {
	reg, err := LoadProxies()
	if err != nil {
		return nil, false
	}
	return findFullstackProxyForSiteIn(reg, siteName)
}

func findFullstackProxyForSiteIn(reg *ProxyRegistry, siteName string) (*Proxy, bool) {
	for i := range reg.Proxies {
		p := &reg.Proxies[i]
		if p.Site == siteName && siteName != "" {
			return p, true
		}
		for _, r := range p.Routes {
			if r.Site == siteName && siteName != "" {
				return p, true
			}
		}
	}
	return nil, false
}
```

- [ ] **Step 4: Chamar `Validate()` em `SaveProxies`**

Em `SaveProxies`, logo no início (antes de montar `raw`), adicionar:

```go
	for _, p := range reg.Proxies {
		if err := p.Validate(); err != nil {
			return fmt.Errorf("proxy %q inválido: %w", p.Name, err)
		}
	}
```

- [ ] **Step 5: Rodar testes do pacote** `go test ./internal/config/` → ok. `gofmt -l` limpo.

- [ ] **Step 6: Commit**
```bash
git add internal/config/proxy.go internal/config/proxy_validate_test.go
git commit -m "feat(config): Proxy.Validate estrutural + FindFullstackProxyForSite + guard em SaveProxies"
```

---

## Task 2: helpers de sync de `.env` em proxyops

**Files:**
- Create: `internal/proxyops/env.go`
- Test: `internal/proxyops/env_test.go`

- [ ] **Step 1: Teste falhando — create `internal/proxyops/env_test.go`**

```go
package proxyops

import (
	"sort"
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

func TestBoundSites(t *testing.T) {
	p := config.Proxy{
		Site: "spa", UpstreamPort: 0,
		Routes: []config.Route{
			{Path: "/api", Site: "api"},
			{Path: "/sanctum", Site: "api"}, // duplicado → distinto
			{Path: "/legacy", UpstreamPort: 8001},
		},
	}
	got := boundSites(p)
	sort.Strings(got)
	want := []string{"api", "spa"}
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("boundSites = %v, want %v", got, want)
	}
}

func TestSyncProxyEnv_PointsSitesToUnifiedDomain(t *testing.T) {
	origFind := findSiteFn
	origSync := syncEnvFn
	defer func() { findSiteFn = origFind; syncEnvFn = origSync }()

	findSiteFn = func(name string) (*config.Site, error) {
		return &config.Site{Name: name, Path: "/srv/" + name}, nil
	}
	type call struct {
		path, domain string
		secured      bool
	}
	var calls []call
	syncEnvFn = func(path, domain string, secured bool) error {
		calls = append(calls, call{path, domain, secured})
		return nil
	}

	p := config.Proxy{
		Domains: []string{"retencao.localhost"}, UpstreamPort: 9000, Secured: true,
		Routes: []config.Route{{Path: "/api", Site: "retencao-api"}},
	}
	if err := syncProxyEnv(p); err != nil {
		t.Fatalf("syncProxyEnv: %v", err)
	}
	if len(calls) != 1 || calls[0].path != "/srv/retencao-api" ||
		calls[0].domain != "retencao.localhost" || !calls[0].secured {
		t.Fatalf("calls = %+v", calls)
	}
}

func TestUnbindSitesEnv_RevertsToOwnDomain(t *testing.T) {
	origFind := findSiteFn
	origSync := syncEnvFn
	defer func() { findSiteFn = origFind; syncEnvFn = origSync }()

	findSiteFn = func(name string) (*config.Site, error) {
		return &config.Site{Name: name, Path: "/srv/" + name, Domains: []string{name + ".localhost"}, Secured: true}, nil
	}
	var gotDomain string
	syncEnvFn = func(path, domain string, secured bool) error { gotDomain = domain; return nil }

	unbindSitesEnv([]string{"retencao-api"})
	if gotDomain != "retencao-api.localhost" {
		t.Errorf("domain = %q, want retencao-api.localhost", gotDomain)
	}
}
```

- [ ] **Step 2: Rodar → FAIL** (`go test ./internal/proxyops/ -run 'BoundSites|SyncProxyEnv|UnbindSitesEnv' -v`).

- [ ] **Step 3: Implementar — create `internal/proxyops/env.go`**

```go
package proxyops

import (
	"github.com/gabriel-sousa99/lerd/internal/config"
	"github.com/gabriel-sousa99/lerd/internal/envfile"
)

// syncEnvFn is injectable for tests. Production wires to envfile.SyncPrimaryDomain.
var syncEnvFn = envfile.SyncPrimaryDomain

// boundSites returns the distinct site names a proxy targets (base + routes).
func boundSites(p config.Proxy) []string {
	seen := map[string]bool{}
	var out []string
	add := func(name string) {
		if name != "" && !seen[name] {
			seen[name] = true
			out = append(out, name)
		}
	}
	add(p.Site)
	for _, r := range p.Routes {
		add(r.Site)
	}
	return out
}

// syncProxyEnv points the .env of every site bound to p at the unified domain
// (p's primary domain), so sessions/cookies are first-party. Best-effort: a
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
	return nil
}

// unbindSitesEnv reverts each named site's .env back to its own primary domain.
func unbindSitesEnv(names []string) {
	for _, name := range names {
		s, err := findSiteFn(name)
		if err != nil {
			continue
		}
		_ = syncEnvFn(s.Path, s.PrimaryDomain(), s.Secured)
	}
}
```

- [ ] **Step 4: Rodar → PASS.** `go test ./internal/proxyops/ -run 'BoundSites|SyncProxyEnv|UnbindSitesEnv' -v`.

- [ ] **Step 5: Commit**
```bash
git add internal/proxyops/env.go internal/proxyops/env_test.go
git commit -m "feat(proxyops): helpers de sync de .env do site para o domínio unificado"
```

---

## Task 3: `AddOptions` aceita fullstack + valida + sync de env

**Files:**
- Modify: `internal/proxyops/add.go`
- Test: `internal/proxyops/add_fullstack_test.go` (create)

- [ ] **Step 1: Teste falhando — create `internal/proxyops/add_fullstack_test.go`**

```go
package proxyops

import (
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

func TestAdd_Fullstack_SyncsEnvAndPersistsRoutes(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	origFind, origSync, origReload, origSecure := findSiteFn, syncEnvFn, nginxReloadFn, secureCertFn
	defer func() {
		findSiteFn, syncEnvFn, nginxReloadFn, secureCertFn = origFind, origSync, origReload, origSecure
	}()
	findSiteFn = func(name string) (*config.Site, error) {
		return &config.Site{Name: name, Path: "/srv/" + name, PublicDir: "public", PHPVersion: "8.2"}, nil
	}
	var syncedDomain string
	syncEnvFn = func(path, domain string, secured bool) error { syncedDomain = domain; return nil }
	nginxReloadFn = func() error { return nil }
	secureCertFn = func(p config.Proxy) error { return nil }

	p, err := Add(AddOptions{
		Domain: "retencao.localhost", Port: 9000,
		Routes: []config.Route{{Path: "/api", Site: "retencao-api"}},
	})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if !p.IsFullstack() || len(p.Routes) != 1 || p.Routes[0].Site != "retencao-api" {
		t.Errorf("proxy = %+v", p)
	}
	if syncedDomain != "retencao.localhost" {
		t.Errorf("env synced to %q, want retencao.localhost", syncedDomain)
	}
	got, err := config.FindProxy(p.Name)
	if err != nil || len(got.Routes) != 1 {
		t.Errorf("persisted = %+v err=%v", got, err)
	}
}
```

- [ ] **Step 2: Rodar → FAIL** (Routes field não existe em AddOptions).

- [ ] **Step 3: Implementar em `internal/proxyops/add.go`**

(a) Em `AddOptions`, adicionar após `AutoStart bool`:
```go
	// Fullstack: Site routes the base (/) to a lerd site; Routes maps path
	// prefixes to their own targets. Empty Routes+Site == simple proxy.
	Site   string
	Routes []config.Route
```

(b) Em `Add`, no literal `p := config.Proxy{...}`, adicionar os campos:
```go
		Site:         opts.Site,
		Routes:       opts.Routes,
```
e como `UpstreamPort: opts.Port` já está, mantenha. Para fullstack com base=site, `opts.Port` pode ser 0 — então **relaxe a checagem de porta** no topo de `Add`: trocar
```go
	if opts.Port <= 0 || opts.Port > 65535 {
		return config.Proxy{}, fmt.Errorf("porta inválida: %d", opts.Port)
	}
```
por
```go
	// Porta da base é obrigatória só quando a base NÃO é um site.
	if opts.Site == "" && (opts.Port <= 0 || opts.Port > 65535) {
		return config.Proxy{}, fmt.Errorf("porta inválida: %d", opts.Port)
	}
```

(c) Após montar `p` e ANTES de `secureCertFn`, validar o proxy inteiro:
```go
	if err := p.Validate(); err != nil {
		return config.Proxy{}, err
	}
```

(d) Após `config.AddProxy(p)` e o `nginxReloadFn()`, sincronizar env se fullstack:
```go
	if p.IsFullstack() {
		_ = syncProxyEnv(p)
	}
```

- [ ] **Step 4: Rodar → PASS.** `go test ./internal/proxyops/ -run 'Add_Fullstack' -v`. Depois `go test ./internal/proxyops/` (garantir que os testes simples de Add seguem passando — a checagem de porta relaxada não pode quebrar o caso simples; o teste "porta inválida" simples deve continuar válido pois Site=="" ).

- [ ] **Step 5: Commit**
```bash
git add internal/proxyops/add.go internal/proxyops/add_fullstack_test.go
git commit -m "feat(proxyops): Add aceita fullstack (Site/Routes), valida e sincroniza .env"
```

---

## Task 4: `UpdateOptions` com Routes/Site (replace integral) + sync/unbind

**Files:**
- Modify: `internal/proxyops/update.go`
- Test: `internal/proxyops/update_fullstack_test.go` (create)

- [ ] **Step 1: Teste falhando — create `internal/proxyops/update_fullstack_test.go`**

```go
package proxyops

import (
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

func TestUpdate_ReplacesRoutesAndUnbindsRemoved(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	origFind, origSync, origReload := findSiteFn, syncEnvFn, nginxReloadFn
	defer func() { findSiteFn, syncEnvFn, nginxReloadFn = origFind, origSync, origReload }()
	findSiteFn = func(name string) (*config.Site, error) {
		return &config.Site{Name: name, Path: "/srv/" + name, Domains: []string{name + ".localhost"}, PublicDir: "public", PHPVersion: "8.2", Secured: true}, nil
	}
	var synced []string // "domain" per call
	syncEnvFn = func(path, domain string, secured bool) error { synced = append(synced, domain); return nil }
	nginxReloadFn = func() error { return nil }

	// Seed: fullstack with routes to site "old-api".
	if err := config.AddProxy(config.Proxy{
		Name: "app", Domains: []string{"app.localhost"}, UpstreamPort: 9000, Secured: true,
		Routes: []config.Route{{Path: "/api", Site: "old-api"}},
	}); err != nil {
		t.Fatal(err)
	}

	newRoutes := []config.Route{{Path: "/api", Site: "new-api"}}
	p, err := Update("app", UpdateOptions{Routes: &newRoutes})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if len(p.Routes) != 1 || p.Routes[0].Site != "new-api" {
		t.Errorf("routes = %+v", p.Routes)
	}
	// new-api synced to unified domain; old-api reverted to its own domain.
	var sawUnified, sawRevert bool
	for _, d := range synced {
		if d == "app.localhost" {
			sawUnified = true
		}
		if d == "old-api.localhost" {
			sawRevert = true
		}
	}
	if !sawUnified || !sawRevert {
		t.Errorf("synced = %v (want unified app.localhost + revert old-api.localhost)", synced)
	}
}
```

- [ ] **Step 2: Rodar → FAIL** (Routes/Site não existem em UpdateOptions).

- [ ] **Step 3: Implementar em `internal/proxyops/update.go`**

(a) Em `UpdateOptions`, adicionar:
```go
	// Fullstack edits. Routes is a WHOLE-LIST replacement when non-nil
	// (nil = leave unchanged, &[]Route{} = clear → back to simple). Site
	// sets/clears the base site target.
	Routes *[]config.Route
	Site   *string
```

(b) Em `Update`, antes de `config.AddProxy(updated)`, capturar os sites antigos e aplicar as mudanças de rotas/site:
```go
	oldSites := boundSites(*existing)

	if opts.Site != nil && *opts.Site != updated.Site {
		updated.Site = *opts.Site
		vhostDirty = true
	}
	if opts.Routes != nil {
		updated.Routes = *opts.Routes
		vhostDirty = true
	}
	if err := updated.Validate(); err != nil {
		return nil, err
	}
```

(c) Após o bloco que regenera o vhost (`if vhostDirty && !updated.Paused { ... }`), adicionar a reconciliação de env:
```go
	// Reconcilia o .env: sites ainda vinculados → domínio unificado; sites
	// que saíram → de volta ao próprio domínio.
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
```

NOTE: o `vhostDirty` quando vira simples (Routes limpas) ainda regenera via `RegenerateProxyVhost`, que já ramifica em `IsFullstack()` (Plano 1). Como os nomes de arquivo são os mesmos, não há stale.

- [ ] **Step 4: Rodar → PASS.** `go test ./internal/proxyops/ -run 'Update' -v` (inclui os testes de Update existentes).

- [ ] **Step 5: Commit**
```bash
git add internal/proxyops/update.go internal/proxyops/update_fullstack_test.go
git commit -m "feat(proxyops): Update troca rotas integralmente e reconcilia .env (bind/unbind)"
```

---

## Task 5: `Remove` re-sincroniza `.env` dos sites desvinculados

**Files:**
- Modify: `internal/proxyops/remove.go`
- Test: `internal/proxyops/remove_fullstack_test.go` (create)

- [ ] **Step 1: Ler `internal/proxyops/remove.go`** para conhecer a estrutura atual de `Remove(name string) error` (carrega o proxy, remove vhost, remove do registry, etc.).

- [ ] **Step 2: Teste falhando — create `internal/proxyops/remove_fullstack_test.go`**

```go
package proxyops

import (
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

func TestRemove_Fullstack_RevertsSiteEnv(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	origFind, origSync, origReload := findSiteFn, syncEnvFn, nginxReloadFn
	defer func() { findSiteFn, syncEnvFn, nginxReloadFn = origFind, origSync, origReload }()
	findSiteFn = func(name string) (*config.Site, error) {
		return &config.Site{Name: name, Path: "/srv/" + name, Domains: []string{name + ".localhost"}, Secured: true}, nil
	}
	var reverted []string
	syncEnvFn = func(path, domain string, secured bool) error { reverted = append(reverted, domain); return nil }
	nginxReloadFn = func() error { return nil }

	if err := config.AddProxy(config.Proxy{
		Name: "app", Domains: []string{"app.localhost"}, UpstreamPort: 9000, Secured: true,
		Routes: []config.Route{{Path: "/api", Site: "api"}},
	}); err != nil {
		t.Fatal(err)
	}
	if err := Remove("app"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	found := false
	for _, d := range reverted {
		if d == "api.localhost" {
			found = true
		}
	}
	if !found {
		t.Errorf("env do site não revertido ao próprio domínio; calls=%v", reverted)
	}
}
```

- [ ] **Step 3: Implementar em `internal/proxyops/remove.go`**

No início de `Remove`, antes de remover do registry, capturar os sites vinculados; ao final (após remover com sucesso), revertê-los. Ajuste conforme a estrutura lida no Step 1; o padrão é:

```go
func Remove(name string) error {
	p, err := config.FindProxy(name)
	if err != nil {
		return err
	}
	sites := boundSites(*p)

	// ... lógica existente de remoção (vhost, registry, quadlet) ...

	unbindSitesEnv(sites)
	return nil
}
```

Se `Remove` hoje não carrega o proxy primeiro, adicionar o `FindProxy` no topo (best-effort: se não achar, seguir a remoção como antes). Preservar todo o comportamento existente; só **acrescentar** a captura no início e o `unbindSitesEnv` no fim do caminho de sucesso.

- [ ] **Step 4: Rodar → PASS.** `go test ./internal/proxyops/ -run 'Remove' -v`.

- [ ] **Step 5: Commit**
```bash
git add internal/proxyops/remove.go internal/proxyops/remove_fullstack_test.go
git commit -m "feat(proxyops): Remove reverte .env dos sites desvinculados"
```

---

## Task 6: coerência no sync de domínio do site (fullstack-aware)

**Files:**
- Create: `internal/siteops/env.go`
- Test: `internal/siteops/env_test.go`
- Modify call sites: `internal/cli/domain.go` (2), `internal/cli/env.go` (1), `internal/mcp/server.go` (2), `internal/siteops/secure.go` (1)

- [ ] **Step 1: Teste falhando — create `internal/siteops/env_test.go`**

```go
package siteops

import (
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

func TestEffectiveEnvDomain(t *testing.T) {
	origFind, origSync := findFullstackFn, syncEnvFn
	defer func() { findFullstackFn, syncEnvFn = origFind, origSync }()

	// Site bound to a fullstack → unified domain + proxy.Secured.
	findFullstackFn = func(name string) (*config.Proxy, bool) {
		if name == "api" {
			return &config.Proxy{Domains: []string{"unified.localhost"}, Secured: true}, true
		}
		return nil, false
	}

	d, sec := effectiveEnvDomain(config.Site{Name: "api", Domains: []string{"api.localhost"}, Secured: false})
	if d != "unified.localhost" || !sec {
		t.Errorf("bound: got (%q,%v), want (unified.localhost,true)", d, sec)
	}
	d, sec = effectiveEnvDomain(config.Site{Name: "solo", Domains: []string{"solo.localhost"}, Secured: true})
	if d != "solo.localhost" || !sec {
		t.Errorf("unbound: got (%q,%v), want (solo.localhost,true)", d, sec)
	}
}
```

- [ ] **Step 2: Rodar → FAIL.**

- [ ] **Step 3: Implementar — create `internal/siteops/env.go`**

```go
package siteops

import (
	"github.com/gabriel-sousa99/lerd/internal/config"
	"github.com/gabriel-sousa99/lerd/internal/envfile"
)

// Injectable seams for tests.
var (
	findFullstackFn = config.FindFullstackProxyForSite
	syncEnvFn       = envfile.SyncPrimaryDomain
)

// effectiveEnvDomain returns the domain/TLS that the site's .env should
// reflect: the unified domain of the fullstack proxy it's bound to (if any),
// otherwise its own primary domain.
func effectiveEnvDomain(site config.Site) (string, bool) {
	if p, ok := findFullstackFn(site.Name); ok {
		return p.PrimaryDomain(), p.Secured
	}
	return site.PrimaryDomain(), site.Secured
}

// SyncSiteEnv syncs the site's domain-scoped .env keys to the effective
// domain (fullstack-aware). Drop-in replacement for direct calls to
// envfile.SyncPrimaryDomain(site.Path, site.PrimaryDomain(), site.Secured).
func SyncSiteEnv(site config.Site) error {
	domain, secured := effectiveEnvDomain(site)
	return syncEnvFn(site.Path, domain, secured)
}
```

- [ ] **Step 4: Rodar → PASS.** `go test ./internal/siteops/ -run EffectiveEnvDomain -v`.

- [ ] **Step 5: Rotear os call sites do sync por domínio próprio do site**

Em cada arquivo abaixo, substituir a chamada direta a `envfile.SyncPrimaryDomain(site.Path, site.PrimaryDomain(), site.Secured)` (e a variante com `secured`) por `siteops.SyncSiteEnv(site)`. NÃO alterar as chamadas de worktree (`certs/secure.go`, que usam `wt.Path/wt.Domain`) nem `migrate_tld.go` (migração de TLD em massa — fora de escopo).

- `internal/siteops/secure.go:74` — `_ = envfile.SyncPrimaryDomain(site.Path, site.PrimaryDomain(), secured)` → como aqui `secured` é o novo estado sendo aplicado, manter a semântica: trocar por
  ```go
  _ = SyncSiteEnv(config.Site{Name: site.Name, Path: site.Path, Domains: site.Domains, Secured: secured})
  ```
  (mesmo pacote `siteops`, chamar `SyncSiteEnv` direto). Ler o arquivo para confirmar os campos disponíveis de `site` no escopo.
- `internal/cli/domain.go:130` e `:200` — trocar por `siteops.SyncSiteEnv(*site)` (ou `site` conforme for ponteiro/valor; ler o contexto). Importar `internal/siteops` se necessário.
- `internal/cli/env.go:141` — aqui usa `cwd` como path, não `site.Path`. Trocar por `siteops.SyncSiteEnv(config.Site{Name: site.Name, Path: cwd, Domains: site.Domains, Secured: site.Secured})`. Ler o contexto para confirmar as variáveis (`cwd`, `site`).
- `internal/mcp/server.go:3366` e `:3428` — trocar por `siteops.SyncSiteEnv(*site)` conforme o tipo no escopo. Importar `internal/siteops`.

Para cada um: ler ~10 linhas ao redor antes de editar, preservar `//nolint` e tratamento de erro existentes.

- [ ] **Step 6: Build + testes amplos**

`go build ./...` → ok. `go test ./internal/siteops/ ./internal/cli/ ./internal/mcp/ ./internal/proxyops/ ./internal/config/` → ok. `gofmt -l` nos arquivos tocados → vazio.

- [ ] **Step 7: Commit**
```bash
git add internal/siteops/env.go internal/siteops/env_test.go internal/siteops/secure.go internal/cli/domain.go internal/cli/env.go internal/mcp/server.go
git commit -m "feat(siteops): sync de .env do site ciente de fullstack (domínio efetivo)"
```

---

## Task 7: CLI `proxy add`/`edit` — flags fullstack + hint advisory

**Files:**
- Modify: `internal/cli/proxy.go`
- Test: `internal/cli/proxy_fullstack_test.go` (create)

- [ ] **Step 1: Ler `internal/cli/proxy.go`** (já mapeado: `newProxyAddCmd` linhas ~36-77, `newProxyEditCmd` ~79-128). Confirmar os helpers `resolveProxyName`, `upstreamForDisplay`.

- [ ] **Step 2: Teste falhando — create `internal/cli/proxy_fullstack_test.go`**

```go
package cli

import (
	"strings"
	"testing"
)

func TestDefaultAPIPaths(t *testing.T) {
	got := defaultAPIPaths()
	want := []string{"/api", "/sanctum", "/broadcasting", "/storage"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("defaultAPIPaths = %v, want %v", got, want)
	}
}

func TestBuildAPIRoutes_SiteWithDefaults(t *testing.T) {
	routes, err := buildAPIRoutes("retencao-api", 0, nil)
	if err != nil {
		t.Fatalf("buildAPIRoutes: %v", err)
	}
	if len(routes) != 4 || routes[0].Path != "/api" || routes[0].Site != "retencao-api" {
		t.Errorf("routes = %+v", routes)
	}
}

func TestBuildAPIRoutes_PortWithCustomPaths(t *testing.T) {
	routes, err := buildAPIRoutes("", 8000, []string{"/api"})
	if err != nil {
		t.Fatalf("buildAPIRoutes: %v", err)
	}
	if len(routes) != 1 || routes[0].UpstreamPort != 8000 || routes[0].Site != "" {
		t.Errorf("routes = %+v", routes)
	}
}

func TestBuildAPIRoutes_BothTargetsErr(t *testing.T) {
	if _, err := buildAPIRoutes("x", 8000, nil); err == nil {
		t.Error("esperava erro com site e porta juntos")
	}
}

func TestBuildAPIRoutes_NoneIsEmpty(t *testing.T) {
	routes, err := buildAPIRoutes("", 0, nil)
	if err != nil || routes != nil {
		t.Errorf("sem api target deve retornar nil,nil; got %+v err=%v", routes, err)
	}
}
```

- [ ] **Step 3: Implementar os helpers + flags em `internal/cli/proxy.go`**

(a) Adicionar os helpers (perto do topo do arquivo, após os imports):
```go
func defaultAPIPaths() []string {
	return []string{"/api", "/sanctum", "/broadcasting", "/storage"}
}

// buildAPIRoutes turns the opinionated CLI flags into config.Route entries.
// Exactly one of apiSite/apiPort may be set. When neither is set it returns
// (nil, nil) — a simple proxy. Paths default to defaultAPIPaths().
func buildAPIRoutes(apiSite string, apiPort int, apiPaths []string) ([]config.Route, error) {
	hasSite := apiSite != ""
	hasPort := apiPort != 0
	if !hasSite && !hasPort {
		if len(apiPaths) > 0 {
			return nil, fmt.Errorf("--api-path requer --api-site ou --api-port")
		}
		return nil, nil
	}
	if hasSite && hasPort {
		return nil, fmt.Errorf("use --api-site OU --api-port, não os dois")
	}
	paths := apiPaths
	if len(paths) == 0 {
		paths = defaultAPIPaths()
	}
	var routes []config.Route
	for _, p := range paths {
		r := config.Route{Path: p}
		if hasSite {
			r.Site = apiSite
		} else {
			r.UpstreamPort = apiPort
		}
		routes = append(routes, r)
	}
	return routes, nil
}

// fullstackHint returns an advisory message to print after creating/editing a
// fullstack proxy whose API is a site, so the user knows the .env was pointed
// at the unified domain.
func fullstackHint(domain string, secured bool) string {
	scheme := "http"
	if secured {
		scheme = "https"
	}
	return fmt.Sprintf("dica: o .env do site de API foi apontado para %s://%s "+
		"(APP_URL / SANCTUM_STATEFUL_DOMAINS, se presentes). Ajuste manualmente se necessário.", scheme, domain)
}
```

(b) Em `newProxyAddCmd`, adicionar as flags e passar as rotas:
```go
	var apiSite string
	var apiPort int
	var apiPaths []string
```
Dentro do `RunE`, antes de `proxyops.Add(...)`:
```go
		routes, rerr := buildAPIRoutes(apiSite, apiPort, apiPaths)
		if rerr != nil {
			return rerr
		}
```
No `proxyops.AddOptions{...}`, adicionar `Routes: routes,`. Após o `fmt.Printf` de sucesso, se `len(routes) > 0`:
```go
		if len(routes) > 0 {
			fmt.Println(fullstackHint(p.PrimaryDomain(), p.Secured))
		}
```
Registrar as flags:
```go
	c.Flags().StringVar(&apiSite, "api-site", "", "Site do lerd que serve a API (fullstack)")
	c.Flags().IntVar(&apiPort, "api-port", 0, "Porta da API em dev server externo (fullstack)")
	c.Flags().StringArrayVar(&apiPaths, "api-path", nil, "Path roteado para a API (repetível; default: /api /sanctum /broadcasting /storage)")
```
Remover/relaxar `_ = c.MarkFlagRequired("port")` → a porta da base deixa de ser obrigatória quando há `--api-site` na base? Não: a base SPA continua sendo `--port`. Mantenha `--port` obrigatório (o caso comum é SPA em porta). Deixe `MarkFlagRequired("port")` como está.

(c) Em `newProxyEditCmd`, adicionar as mesmas flags `--api-site/--api-port/--api-path` e, quando QUALQUER uma for `Changed`, construir as rotas e setar `opts.Routes`:
```go
	if cmd.Flags().Changed("api-site") || cmd.Flags().Changed("api-port") || cmd.Flags().Changed("api-path") {
		routes, rerr := buildAPIRoutes(apiSite, apiPort, apiPaths)
		if rerr != nil {
			return rerr
		}
		opts.Routes = &routes
	}
```
(declare `var apiSite string; var apiPort int; var apiPaths []string` no topo de `newProxyEditCmd` e registre as flags como no add).

- [ ] **Step 4: Rodar → PASS.** `go test ./internal/cli/ -run 'APIPaths|APIRoutes' -v`. Depois `go test ./internal/cli/` (regressão das flags existentes).

- [ ] **Step 5: Commit**
```bash
git add internal/cli/proxy.go internal/cli/proxy_fullstack_test.go
git commit -m "feat(cli): flags fullstack em proxy add/edit (--api-site/--api-port/--api-path) + hint"
```

---

## Task 8: API HTTP — POST/PUT aceitam routes/site; DTO expõe fullstack

**Files:**
- Modify: `internal/ui/proxy_api.go`
- Test: `internal/ui/proxy_api_fullstack_test.go` (create)

- [ ] **Step 1: Teste falhando — create `internal/ui/proxy_api_fullstack_test.go`**

Modelar no estilo dos testes existentes do pacote `ui` (ler `internal/ui/proxy_api_test.go` para o padrão de request/recorder). O teste deve: POST `/api/proxies` com body `{"domain":"r.localhost","port":9000,"routes":[{"path":"/api","site":"r-api"}]}` e verificar que o `proxyDTO` retornado tem `Routes` com 1 item e `Fullstack=true`. (Stubar `proxyops` se o pacote já tiver hooks; caso contrário, usar `XDG_DATA_HOME` temporário + `findSiteFn`/`syncEnvFn`/`nginxReloadFn` stub via funções exportadas de teste — verificar o que `proxyops` expõe para testes do pacote `ui`. Se não houver seam, validar só a (de)serialização do DTO com `toProxyDTO`.)

Implementação mínima garantida (sem depender de seams de proxyops): testar `toProxyDTO` com um `config.Proxy` fullstack:
```go
package ui

import "testing"
import "github.com/gabriel-sousa99/lerd/internal/config"

func TestToProxyDTO_Fullstack(t *testing.T) {
	p := config.Proxy{
		Name: "r", Domains: []string{"r.localhost"}, UpstreamPort: 9000,
		Routes: []config.Route{{Path: "/api", Site: "r-api"}},
	}
	dto := toProxyDTO(p)
	if !dto.Fullstack || len(dto.Routes) != 1 || dto.Routes[0].Site != "r-api" {
		t.Errorf("dto = %+v", dto)
	}
}
```

- [ ] **Step 2: Rodar → FAIL** (campos Routes/Fullstack não existem no DTO).

- [ ] **Step 3: Implementar em `internal/ui/proxy_api.go`**

(a) No `proxyDTO`, adicionar:
```go
	Site      string         `json:"site,omitempty"`
	Routes    []config.Route `json:"routes,omitempty"`
	Fullstack bool           `json:"fullstack"`
```
(b) Em `toProxyDTO`, no literal de retorno, adicionar:
```go
		Site:      p.Site,
		Routes:    p.Routes,
		Fullstack: p.IsFullstack(),
```
(c) No handler POST (`handleProxies`, case `MethodPost`), estender o `body` struct e a chamada `Add`:
```go
		var body struct {
			Domain      string         `json:"domain"`
			Port        int            `json:"port"`
			Path        string         `json:"path"`
			NoSecure    bool           `json:"no_secure"`
			Managed     bool           `json:"managed"`
			Command     string         `json:"cmd"`
			NodeVersion string         `json:"node_version"`
			AutoStart   bool           `json:"autostart"`
			Site        string         `json:"site"`
			Routes      []config.Route `json:"routes"`
		}
```
e em `proxyops.AddOptions{...}` adicionar `Site: body.Site,` e `Routes: body.Routes,`.
(d) No handler PUT (`handleProxyAction`, bloco `MethodPut`), estender o `body` e o `UpdateOptions`:
```go
			var body struct {
				Port         *int            `json:"port"`
				Path         *string         `json:"path"`
				Command      *string         `json:"cmd"`
				NodeVersion  *string         `json:"node_version"`
				UpstreamHost *string         `json:"upstream_host"`
				AutoStart    *bool           `json:"autostart"`
				Routes       *[]config.Route `json:"routes"`
				Site         *string         `json:"site"`
			}
```
e em `proxyops.Update(name, proxyops.UpdateOptions{...})` adicionar `Routes: body.Routes,` e `Site: body.Site,`.

- [ ] **Step 4: Rodar → PASS.** `go test ./internal/ui/ -run 'ProxyDTO_Fullstack' -v`. Depois `go test ./internal/ui/`.

- [ ] **Step 5: Commit**
```bash
git add internal/ui/proxy_api.go internal/ui/proxy_api_fullstack_test.go
git commit -m "feat(api): POST/PUT aceitam routes/site; proxyDTO expõe routes/site/fullstack"
```

---

## Self-Review (resultado)

- **Cobertura das decisões do grill (§11 do spec):** Validação inteira + guard em SaveProxies → Task 1; sync de env (bind/unbind, itera sites distintos) → Tasks 2-5; coerência fullstack-aware no sync do site → Task 6; CLI opinativa + hint → Task 7; API POST/PUT + DTO → Task 8.
- **Placeholders:** as NOTES das Tasks 5 e 6 (ler o arquivo antes de editar call sites de uma linha) são instruções de execução concretas — os trechos a substituir são identificados exatamente (`envfile.SyncPrimaryDomain(site.Path, site.PrimaryDomain(), site.Secured)`), não lacunas de design.
- **Consistência de tipos:** `config.Route`, `config.Proxy.{Site,Routes}`, `Proxy.Validate()`, `Proxy.IsFullstack()`, `FindFullstackProxyForSite`; `proxyops.{boundSites,syncProxyEnv,unbindSitesEnv,syncEnvFn,findSiteFn}`; `siteops.{SyncSiteEnv,effectiveEnvDomain,findFullstackFn,syncEnvFn}`; `AddOptions.{Site,Routes}`, `UpdateOptions.{Routes,Site}`; DTO `{Site,Routes,Fullstack}` — usados de forma idêntica entre tasks.
- **Ordem:** Task 1 (config) → 2 (helpers) → 3/4/5 (Add/Update/Remove usam os helpers) → 6 (siteops, usa FindFullstackProxyForSite da Task 1) → 7 (CLI usa AddOptions/UpdateOptions das Tasks 3/4) → 8 (API idem). Dependências respeitadas.
- **Risco conhecido:** o sync de env é best-effort (não falha a operação de proxy) — coerente com o comportamento de `SyncPrimaryDomain` (no-op sem `.env`). `migrate_tld.go` e worktrees ficam fora da coerência fullstack-aware (documentado).
```
