# Proxy Fullstack — Plano 1: Backend Core

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Modelar e gerar o vhost de um proxy fullstack (uma origem, vários paths → upstreams distintos), suportando target = porta (proxy_pass) ou site do lerd (fastcgi), resolvendo o cookie same-origin.

**Architecture:** O `config.Proxy` ganha `Routes []Route` e um `Site` opcional na base. Proxy sem rotas = comportamento atual (template intocado → saída byte-idêntica). Com rotas, `proxyops` resolve cada target (lookup de site → docroot + versão PHP) num `ProxyVhostSpec` e o pacote `nginx` renderiza novos templates fullstack. Nenhuma lógica dependente de versão de PHP: reusa `lerd-php<short>-fpm:9000` como o vhost de site standalone.

**Tech Stack:** Go, `text/template`, `gopkg.in/yaml.v3`, testes `go test`.

**Escopo:** Este plano cobre seções 3, 4, 5 (parcial) e 10 do spec `docs/superpowers/specs/2026-06-01-proxy-fullstack-design.md`. CLI/API (seções 6-7) e UI (seção 8) ficam nos planos 2 e 3. Criar fullstack via `Add`/`Update` fica no plano 2; aqui o foco é modelo + geração + validação, exercitados via `RegenerateProxyVhost`.

---

## File Structure

- `internal/config/proxy.go` (modify) — tipo `Route`, campos `Site`/`Routes` no `Proxy` e no `proxyYAML`, deep-copy, helper `IsFullstack`, validação pura `ValidateProxyRoutes`.
- `internal/config/proxy_routes_test.go` (create) — round-trip YAML + validação pura.
- `internal/nginx/fullstack.go` (create) — tipos `ProxyVhostSpec`/`ProxyTarget`/`ProxyRouteSpec` e `GenerateFullstackProxyVhost`.
- `internal/nginx/templates/vhost-proxy-fullstack.conf.tmpl` (create) — HTTP.
- `internal/nginx/templates/vhost-proxy-fullstack-ssl.conf.tmpl` (create) — HTTPS.
- `internal/nginx/fullstack_test.go` (create) — snapshot do config gerado (base porta + rotas site/porta).
- `internal/proxyops/spec.go` (create) — `resolveProxySpec(p)` (lookup de site via `findSiteFn`, docroot, phpShort, sanitização de named location) + `findSiteFn` injetável.
- `internal/proxyops/spec_test.go` (create) — resolução com site stub.
- `internal/proxyops/vhost.go` (modify) — `RegenerateProxyVhost` ramifica em `IsFullstack`.
- `internal/proxyops/vhost_fullstack_test.go` (create) — regeneração fullstack ponta-a-ponta (com `findSiteFn` stub + HOME temporário).

---

## Task 1: Modelo de dados — `Route`, campos e round-trip YAML

**Files:**
- Modify: `internal/config/proxy.go`
- Test: `internal/config/proxy_routes_test.go`

- [ ] **Step 1: Escrever o teste de round-trip e validação (falhando)**

Create `internal/config/proxy_routes_test.go`:

```go
package config

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestProxyRoutesYAMLRoundTrip(t *testing.T) {
	in := Proxy{
		Name:         "retencao",
		Domains:      []string{"retencao.localhost"},
		UpstreamPort: 9000,
		Secured:      true,
		Routes: []Route{
			{Path: "/api", Site: "retencao-api"},
			{Path: "/sanctum", Site: "retencao-api"},
			{Path: "/legacy", UpstreamPort: 8001, UpstreamHost: "127.0.0.1"},
		},
	}

	raw := proxyRegistryYAML{Proxies: []proxyYAML{in.toYAML()}}
	out, err := yaml.Marshal(raw)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var back proxyRegistryYAML
	if err := yaml.Unmarshal(out, &back); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	got := back.Proxies[0].toProxy()

	if len(got.Routes) != 3 {
		t.Fatalf("routes = %d, want 3\nyaml:\n%s", len(got.Routes), out)
	}
	if got.Routes[0].Path != "/api" || got.Routes[0].Site != "retencao-api" {
		t.Errorf("route[0] = %+v", got.Routes[0])
	}
	if got.Routes[2].UpstreamPort != 8001 || got.Routes[2].UpstreamHost != "127.0.0.1" {
		t.Errorf("route[2] = %+v", got.Routes[2])
	}
	if !got.IsFullstack() {
		t.Error("IsFullstack() = false, want true")
	}
}

func TestSimpleProxyIsNotFullstack(t *testing.T) {
	p := Proxy{Name: "spa", Domains: []string{"spa.localhost"}, UpstreamPort: 5173}
	if p.IsFullstack() {
		t.Error("simple proxy reported as fullstack")
	}
}

func TestValidateProxyRoutes(t *testing.T) {
	cases := []struct {
		name    string
		routes  []Route
		wantErr bool
	}{
		{"ok", []Route{{Path: "/api", Site: "x"}}, false},
		{"no leading slash", []Route{{Path: "api", Site: "x"}}, true},
		{"root path", []Route{{Path: "/", Site: "x"}}, true},
		{"duplicate", []Route{{Path: "/api", Site: "x"}, {Path: "/api", UpstreamPort: 9}}, true},
		{"no target", []Route{{Path: "/api"}}, true},
		{"both targets", []Route{{Path: "/api", Site: "x", UpstreamPort: 9}}, true},
		{"bad port", []Route{{Path: "/api", UpstreamPort: 70000}}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateProxyRoutes(tc.routes)
			if (err != nil) != tc.wantErr {
				t.Errorf("err = %v, wantErr = %v", err, tc.wantErr)
			}
		})
	}
}
```

- [ ] **Step 2: Rodar o teste e ver falhar**

Run: `go test ./internal/config/ -run 'ProxyRoutes|SimpleProxy|ValidateProxyRoutes' -v`
Expected: FAIL na compilação (`Route`, `IsFullstack`, `ValidateProxyRoutes` indefinidos).

- [ ] **Step 3: Implementar tipo, campos, conversões, helper e validação**

Em `internal/config/proxy.go`, adicionar o tipo `Route` logo antes de `type Proxy struct`:

```go
// Route is one path-prefixed upstream of a fullstack proxy. Exactly one of
// Site (fastcgi to a lerd PHP site) or UpstreamPort (proxy_pass to host:port)
// must be set.
type Route struct {
	Path         string `yaml:"path"`
	Site         string `yaml:"site,omitempty"`
	UpstreamPort int    `yaml:"upstream_port,omitempty"`
	UpstreamHost string `yaml:"upstream_host,omitempty"`
}
```

No `Proxy` struct, adicionar dois campos após `AutoStart`:

```go
	// Fullstack: Site routes the base (/) to a lerd PHP site instead of a
	// port; Routes maps extra path prefixes to their own upstreams. Empty
	// Routes == a plain single-upstream proxy (unchanged behaviour).
	Site   string  `yaml:"-"`
	Routes []Route `yaml:"-"`
```

No `proxyYAML` struct, adicionar após `AutoStart`:

```go
	Site    string  `yaml:"site,omitempty"`
	Routes  []Route `yaml:"routes,omitempty"`
```

Em `toYAML()`, adicionar ao literal retornado:

```go
		Site:    p.Site,
		Routes:  p.Routes,
```

Em `toProxy()`, adicionar ao literal retornado:

```go
		Site:    py.Site,
		Routes:  py.Routes,
```

Adicionar o helper após `PrimaryDomain()`:

```go
// IsFullstack reports whether this proxy uses path-based routing.
func (p Proxy) IsFullstack() bool { return len(p.Routes) > 0 }

// ValidateProxyRoutes checks route paths and targets. Each path must start
// with "/", differ from "/", be unique, and carry exactly one target
// (Site xor UpstreamPort).
func ValidateProxyRoutes(routes []Route) error {
	seen := make(map[string]bool, len(routes))
	for _, r := range routes {
		if len(r.Path) == 0 || r.Path[0] != '/' {
			return fmt.Errorf("path da rota deve começar com /: %q", r.Path)
		}
		if r.Path == "/" {
			return fmt.Errorf("path da rota não pode ser / (reservado para a base)")
		}
		if seen[r.Path] {
			return fmt.Errorf("path de rota duplicado: %q", r.Path)
		}
		seen[r.Path] = true

		hasSite := r.Site != ""
		hasPort := r.UpstreamPort != 0
		if hasSite == hasPort {
			return fmt.Errorf("rota %q precisa de exatamente um target (site OU upstream_port)", r.Path)
		}
		if hasPort && (r.UpstreamPort <= 0 || r.UpstreamPort > 65535) {
			return fmt.Errorf("rota %q: porta inválida %d", r.Path, r.UpstreamPort)
		}
	}
	return nil
}
```

- [ ] **Step 4: Atualizar deep-copy do cache para incluir Routes**

Em `cloneProxyRegistry`, dentro do loop, após o bloco que copia `Domains`, adicionar:

```go
		if p.Routes != nil {
			cp.Routes = append([]Route(nil), p.Routes...)
		}
```

- [ ] **Step 5: Rodar os testes e ver passar**

Run: `go test ./internal/config/ -run 'ProxyRoutes|SimpleProxy|ValidateProxyRoutes' -v`
Expected: PASS (3 testes / subtests).

- [ ] **Step 6: Garantir que nada quebrou no pacote config**

Run: `go test ./internal/config/`
Expected: ok.

- [ ] **Step 7: Commit**

```bash
git add internal/config/proxy.go internal/config/proxy_routes_test.go
git commit -m "feat(config): modelar rotas de proxy fullstack (Route + validação)"
```

---

## Task 2: Resolver `ProxyVhostSpec` em proxyops (lookup de site → docroot/PHP)

**Files:**
- Create: `internal/proxyops/spec.go`
- Test: `internal/proxyops/spec_test.go`

- [ ] **Step 1: Escrever o teste de resolução (falhando)**

Create `internal/proxyops/spec_test.go`:

```go
package proxyops

import (
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

func TestResolveProxySpec_PortBaseAndSiteRoute(t *testing.T) {
	orig := findSiteFn
	findSiteFn = func(name string) (*config.Site, error) {
		return &config.Site{
			Name:       "retencao-api",
			Domains:    []string{"retencao-api.localhost"},
			Path:       "/home/user/retencao-api",
			PublicDir:  "public",
			PHPVersion: "8.2",
		}, nil
	}
	defer func() { findSiteFn = orig }()

	p := config.Proxy{
		Name:         "retencao",
		Domains:      []string{"retencao.localhost"},
		UpstreamPort: 9000,
		Secured:      true,
		Routes: []config.Route{
			{Path: "/api", Site: "retencao-api"},
			{Path: "/sanctum", Site: "retencao-api"},
			{Path: "/legacy", UpstreamPort: 8001},
		},
	}

	spec, err := resolveProxySpec(p)
	if err != nil {
		t.Fatalf("resolveProxySpec: %v", err)
	}
	if spec.Base.IsSite || spec.Base.UpstreamPort != 9000 || spec.Base.UpstreamHost != "host.containers.internal" {
		t.Errorf("base = %+v", spec.Base)
	}
	if len(spec.Routes) != 3 {
		t.Fatalf("routes = %d, want 3", len(spec.Routes))
	}
	// Rotas para o mesmo site compartilham o mesmo named location.
	if spec.Routes[0].Target.LocationName != spec.Routes[1].Target.LocationName {
		t.Errorf("rotas do mesmo site deveriam compartilhar location: %q vs %q",
			spec.Routes[0].Target.LocationName, spec.Routes[1].Target.LocationName)
	}
	r0 := spec.Routes[0].Target
	if !r0.IsSite || r0.DocRoot != "/home/user/retencao-api/public" || r0.PHPShort != "82" {
		t.Errorf("route[0].target = %+v", r0)
	}
	if spec.Routes[0].Target.LocationName != "site_retencao_api" {
		t.Errorf("location name = %q, want site_retencao_api", spec.Routes[0].Target.LocationName)
	}
	r2 := spec.Routes[2].Target
	if r2.IsSite || r2.UpstreamPort != 8001 || r2.UpstreamHost != "host.containers.internal" {
		t.Errorf("route[2].target = %+v", r2)
	}
}

func TestResolveProxySpec_SiteNotFound(t *testing.T) {
	orig := findSiteFn
	findSiteFn = func(name string) (*config.Site, error) {
		return nil, config.ErrSiteNotFoundForTest
	}
	defer func() { findSiteFn = orig }()

	p := config.Proxy{
		Name: "x", Domains: []string{"x.localhost"}, UpstreamPort: 9000,
		Routes: []config.Route{{Path: "/api", Site: "missing"}},
	}
	if _, err := resolveProxySpec(p); err == nil {
		t.Fatal("esperava erro para site inexistente")
	}
}
```

NOTE: `config.ErrSiteNotFoundForTest` não existe — substituir por um erro real. Trocar o corpo do stub por `return nil, fmt.Errorf("not found")` e importar `fmt` no teste. (Mantido aqui para evidenciar a intenção; usar `fmt.Errorf`.)

- [ ] **Step 2: Rodar e ver falhar**

Run: `go test ./internal/proxyops/ -run ResolveProxySpec -v`
Expected: FAIL na compilação (`findSiteFn`, `resolveProxySpec`, tipos indefinidos).

- [ ] **Step 3: Implementar a resolução**

Create `internal/proxyops/spec.go`:

```go
package proxyops

import (
	"strings"

	"github.com/gabriel-sousa99/lerd/internal/config"
	"github.com/gabriel-sousa99/lerd/internal/nginx"
)

// findSiteFn is injectable so tests can resolve sites without a real
// sites.yaml. Production wires to config.FindSite.
var findSiteFn = config.FindSite

// resolveProxySpec turns a fullstack config.Proxy into the fully-resolved
// data the nginx package needs: every site target carries an absolute
// docroot and PHP short version; routes that hit the same site share one
// named location so the fastcgi block is emitted once.
func resolveProxySpec(p config.Proxy) (nginx.ProxyVhostSpec, error) {
	if err := config.ValidateProxyRoutes(p.Routes); err != nil {
		return nginx.ProxyVhostSpec{}, err
	}

	base, err := resolveTarget(p.Site, p.UpstreamPort, p.UpstreamHost)
	if err != nil {
		return nginx.ProxyVhostSpec{}, err
	}

	spec := nginx.ProxyVhostSpec{
		Domain:  p.PrimaryDomain(),
		Secured: p.Secured,
		Base:    base,
	}
	for _, r := range p.Routes {
		t, err := resolveTarget(r.Site, r.UpstreamPort, r.UpstreamHost)
		if err != nil {
			return nginx.ProxyVhostSpec{}, err
		}
		spec.Routes = append(spec.Routes, nginx.ProxyRouteSpec{Path: r.Path, Target: t})
	}
	return spec, nil
}

// resolveTarget builds a nginx.ProxyTarget from a (site | port) pair. site
// wins when set; otherwise it's a port target defaulting the host.
func resolveTarget(site string, port int, host string) (nginx.ProxyTarget, error) {
	if site != "" {
		s, err := findSiteFn(site)
		if err != nil {
			return nginx.ProxyTarget{}, err
		}
		public := s.PublicDir
		if public == "" {
			public = "public"
		}
		return nginx.ProxyTarget{
			IsSite:       true,
			SiteName:     s.Name,
			DocRoot:      strings.TrimRight(s.Path, "/") + "/" + public,
			PHPShort:     strings.ReplaceAll(s.PHPVersion, ".", ""),
			LocationName: sanitizeLocation(s.Name),
		}, nil
	}
	h := host
	if h == "" {
		h = defaultUpstreamHost
	}
	return nginx.ProxyTarget{UpstreamPort: port, UpstreamHost: h}, nil
}

// sanitizeLocation maps a site name to a safe nginx named-location label,
// e.g. "retencao-api" → "site_retencao_api".
func sanitizeLocation(name string) string {
	var b strings.Builder
	b.WriteString("site_")
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	return b.String()
}
```

NOTE: este passo não compila sozinho — depende dos tipos `nginx.ProxyVhostSpec`/`ProxyTarget`/`ProxyRouteSpec` criados na Task 3. Implementar a Task 3 antes de rodar os testes (a ordem de escrita é spec→nginx, mas a ordem de compilação exige nginx primeiro). Para manter TDD verde, **execute Step 5 só após a Task 3**.

- [ ] **Step 4: Ajustar o teste para usar `fmt.Errorf`**

Em `spec_test.go`, no `TestResolveProxySpec_SiteNotFound`, trocar `config.ErrSiteNotFoundForTest` por `fmt.Errorf("not found")` e adicionar `"fmt"` aos imports.

- [ ] **Step 5: (após Task 3) Rodar e ver passar**

Run: `go test ./internal/proxyops/ -run ResolveProxySpec -v`
Expected: PASS.

- [ ] **Step 6: Commit (após Task 3 compilar)**

```bash
git add internal/proxyops/spec.go internal/proxyops/spec_test.go
git commit -m "feat(proxyops): resolver ProxyVhostSpec (lookup de site, docroot, named location)"
```

---

## Task 3: Templates fullstack + `GenerateFullstackProxyVhost`

**Files:**
- Create: `internal/nginx/fullstack.go`
- Create: `internal/nginx/templates/vhost-proxy-fullstack.conf.tmpl`
- Create: `internal/nginx/templates/vhost-proxy-fullstack-ssl.conf.tmpl`
- Test: `internal/nginx/fullstack_test.go`

- [ ] **Step 1: Criar os tipos e o gerador**

Create `internal/nginx/fullstack.go`:

```go
package nginx

import (
	"bytes"
	"os"
	"path/filepath"
	"text/template"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

// ProxyTarget is a resolved upstream: either a lerd PHP site (fastcgi) or a
// host:port (proxy_pass). proxyops fills this from config.
type ProxyTarget struct {
	IsSite       bool
	UpstreamHost string // port mode
	UpstreamPort int    // port mode
	DocRoot      string // site mode: absolute docroot (path/publicdir)
	PHPShort     string // site mode: e.g. "82"
	SiteName     string // site mode
	LocationName string // site mode: named location label (e.g. site_retencao_api)
}

// ProxyRouteSpec binds a path prefix to a resolved target.
type ProxyRouteSpec struct {
	Path   string
	Target ProxyTarget
}

// ProxyVhostSpec is the fully-resolved input for a fullstack proxy vhost.
type ProxyVhostSpec struct {
	Domain  string
	Secured bool
	Base    ProxyTarget      // catch-all "/"
	Routes  []ProxyRouteSpec // path-prefixed routes
}

// siteLocations dedups the named locations needed for site targets so the
// fastcgi block for a given site is emitted exactly once.
func (s ProxyVhostSpec) siteLocations() []ProxyTarget {
	seen := map[string]bool{}
	var out []ProxyTarget
	consider := func(t ProxyTarget) {
		if t.IsSite && !seen[t.LocationName] {
			seen[t.LocationName] = true
			out = append(out, t)
		}
	}
	consider(s.Base)
	for _, r := range s.Routes {
		consider(r.Target)
	}
	return out
}

// GenerateFullstackProxyVhost renders the fullstack template for spec and
// writes conf.d/<domain>.conf (HTTP) or <domain>-ssl.conf (HTTPS),
// removing the stale counterpart like GenerateProxyVhost does.
func GenerateFullstackProxyVhost(spec ProxyVhostSpec) error {
	tmplName := "vhost-proxy-fullstack.conf.tmpl"
	confName := spec.Domain + ".conf"
	stale := spec.Domain + "-ssl.conf"
	if spec.Secured {
		tmplName = "vhost-proxy-fullstack-ssl.conf.tmpl"
		confName = spec.Domain + "-ssl.conf"
		stale = spec.Domain + ".conf"
	}

	tmplData, err := GetTemplate(tmplName)
	if err != nil {
		return err
	}
	tmpl, err := template.New(tmplName).Funcs(template.FuncMap{
		"siteLocations": spec.siteLocations,
	}).Parse(string(tmplData))
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, spec); err != nil {
		return err
	}
	if err := os.MkdirAll(config.NginxConfD(), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(config.NginxConfD(), confName), buf.Bytes(), 0644); err != nil {
		return err
	}
	_ = os.Remove(filepath.Join(config.NginxConfD(), stale))
	return nil
}

// renderFullstackForTest renders the SSL template to a string without
// touching the filesystem (used by tests).
func renderFullstackForTest(spec ProxyVhostSpec, name string) (string, error) {
	tmplData, err := GetTemplate(name)
	if err != nil {
		return "", err
	}
	tmpl, err := template.New(name).Funcs(template.FuncMap{
		"siteLocations": spec.siteLocations,
	}).Parse(string(tmplData))
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, spec); err != nil {
		return "", err
	}
	return buf.String(), nil
}
```

- [ ] **Step 2: Criar o template HTTP**

Create `internal/nginx/templates/vhost-proxy-fullstack.conf.tmpl`:

```
server {
    listen 80;
    listen [::]:80;
    server_name {{.Domain}};
{{if .Base.IsSite}}
    root {{.Base.DocRoot}};
    index index.php index.html;
{{end}}
{{range .Routes}}
    location ^~ {{.Path}} {
{{if .Target.IsSite}}        root {{.Target.DocRoot}};
        try_files $uri @{{.Target.LocationName}};
{{else}}        proxy_pass http://{{.Target.UpstreamHost}}:{{.Target.UpstreamPort}};
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $connection_upgrade;
        proxy_read_timeout 86400;
        proxy_buffering off;
{{end}}    }
{{end}}
{{range siteLocations}}
    location @{{.LocationName}} {
        root {{.DocRoot}};
        set $fpm "lerd-php{{.PHPShort}}-fpm";
        fastcgi_pass $fpm:9000;
        fastcgi_index index.php;
        fastcgi_param SCRIPT_FILENAME $document_root/index.php;
        fastcgi_param SCRIPT_NAME /index.php;
        fastcgi_buffers 16 16k;
        fastcgi_buffer_size 32k;
        include fastcgi_params;
        fastcgi_param HTTP_HOST $real_forwarded_host;
        fastcgi_param SERVER_NAME $real_forwarded_host;
        fastcgi_param HTTP_X_FORWARDED_HOST $real_forwarded_host;
        fastcgi_param HTTP_X_FORWARDED_PROTO $real_forwarded_proto;
        fastcgi_param HTTP_X_FORWARDED_PORT $real_forwarded_port;
        fastcgi_param HTTP_X_REAL_IP $remote_addr;
        fastcgi_param HTTP_X_FORWARDED_FOR $remote_addr;
        fastcgi_param LERD_SITE {{.SiteName}};
    }
{{end}}
    location / {
{{if .Base.IsSite}}        try_files $uri $uri/ /index.php?$query_string;
{{else}}        proxy_pass http://{{.Base.UpstreamHost}}:{{.Base.UpstreamPort}};
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $connection_upgrade;
        proxy_read_timeout 86400;
        proxy_connect_timeout 5s;
        proxy_buffering off;
{{end}}    }
{{if .Base.IsSite}}
    location ~ \.php$ {
        set $fpm "lerd-php{{.Base.PHPShort}}-fpm";
        fastcgi_pass $fpm:9000;
        fastcgi_index index.php;
        fastcgi_param SCRIPT_FILENAME $realpath_root$fastcgi_script_name;
        include fastcgi_params;
        fastcgi_param HTTP_HOST $real_forwarded_host;
        fastcgi_param SERVER_NAME $real_forwarded_host;
        fastcgi_param HTTP_X_FORWARDED_PROTO $real_forwarded_proto;
    }
{{end}}
}
```

- [ ] **Step 3: Criar o template HTTPS**

Create `internal/nginx/templates/vhost-proxy-fullstack-ssl.conf.tmpl` (igual ao HTTP, com redirect 80→443 e bloco TLS; o miolo de locations é idêntico):

```
server {
    listen 80;
    listen [::]:80;
    server_name {{.Domain}};
    return 302 https://$host$request_uri;
}

server {
    listen 443 ssl;
    listen [::]:443 ssl;
    server_name {{.Domain}};

    ssl_certificate /etc/nginx/certs/{{.Domain}}.crt;
    ssl_certificate_key /etc/nginx/certs/{{.Domain}}.key;
{{if .Base.IsSite}}
    root {{.Base.DocRoot}};
    index index.php index.html;
{{end}}
{{range .Routes}}
    location ^~ {{.Path}} {
{{if .Target.IsSite}}        root {{.Target.DocRoot}};
        try_files $uri @{{.Target.LocationName}};
{{else}}        proxy_pass http://{{.Target.UpstreamHost}}:{{.Target.UpstreamPort}};
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $connection_upgrade;
        proxy_read_timeout 86400;
        proxy_buffering off;
{{end}}    }
{{end}}
{{range siteLocations}}
    location @{{.LocationName}} {
        root {{.DocRoot}};
        set $fpm "lerd-php{{.PHPShort}}-fpm";
        fastcgi_pass $fpm:9000;
        fastcgi_index index.php;
        fastcgi_param SCRIPT_FILENAME $document_root/index.php;
        fastcgi_param SCRIPT_NAME /index.php;
        fastcgi_param HTTPS on;
        fastcgi_buffers 16 16k;
        fastcgi_buffer_size 32k;
        include fastcgi_params;
        fastcgi_param HTTP_HOST $real_forwarded_host;
        fastcgi_param SERVER_NAME $real_forwarded_host;
        fastcgi_param HTTP_X_FORWARDED_HOST $real_forwarded_host;
        fastcgi_param HTTP_X_FORWARDED_PROTO $real_forwarded_proto;
        fastcgi_param HTTP_X_FORWARDED_PORT $real_forwarded_port;
        fastcgi_param HTTP_X_REAL_IP $remote_addr;
        fastcgi_param HTTP_X_FORWARDED_FOR $remote_addr;
        fastcgi_param LERD_SITE {{.SiteName}};
    }
{{end}}
    location / {
{{if .Base.IsSite}}        try_files $uri $uri/ /index.php?$query_string;
{{else}}        proxy_pass http://{{.Base.UpstreamHost}}:{{.Base.UpstreamPort}};
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $connection_upgrade;
        proxy_read_timeout 86400;
        proxy_connect_timeout 5s;
        proxy_buffering off;
{{end}}    }
{{if .Base.IsSite}}
    location ~ \.php$ {
        set $fpm "lerd-php{{.Base.PHPShort}}-fpm";
        fastcgi_pass $fpm:9000;
        fastcgi_index index.php;
        fastcgi_param SCRIPT_FILENAME $realpath_root$fastcgi_script_name;
        fastcgi_param HTTPS on;
        include fastcgi_params;
        fastcgi_param HTTP_HOST $real_forwarded_host;
        fastcgi_param SERVER_NAME $real_forwarded_host;
        fastcgi_param HTTP_X_FORWARDED_PROTO $real_forwarded_proto;
    }
{{end}}
}
```

- [ ] **Step 4: Escrever o teste de snapshot do gerador**

Create `internal/nginx/fullstack_test.go`:

```go
package nginx

import (
	"strings"
	"testing"
)

func TestRenderFullstack_PortBaseSiteRoutes(t *testing.T) {
	spec := ProxyVhostSpec{
		Domain:  "retencao.localhost",
		Secured: true,
		Base:    ProxyTarget{UpstreamHost: "host.containers.internal", UpstreamPort: 9000},
		Routes: []ProxyRouteSpec{
			{Path: "/api", Target: siteTarget("retencao-api", "/home/u/retencao-api/public", "82")},
			{Path: "/sanctum", Target: siteTarget("retencao-api", "/home/u/retencao-api/public", "82")},
		},
	}
	out, err := renderFullstackForTest(spec, "vhost-proxy-fullstack-ssl.conf.tmpl")
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	mustContain := []string{
		"server_name retencao.localhost;",
		"return 302 https://$host$request_uri;",
		"ssl_certificate /etc/nginx/certs/retencao.localhost.crt;",
		"location ^~ /api {",
		"location ^~ /sanctum {",
		"try_files $uri @site_retencao_api;",
		"location @site_retencao_api {",
		"fastcgi_pass $fpm:9000;",
		"lerd-php82-fpm",
		"fastcgi_param HTTP_HOST $real_forwarded_host;",
		"proxy_pass http://host.containers.internal:9000;",
	}
	for _, s := range mustContain {
		if !strings.Contains(out, s) {
			t.Errorf("output não contém %q\n---\n%s", s, out)
		}
	}
	// O bloco fastcgi do site deve aparecer UMA vez (dedup por named location).
	if n := strings.Count(out, "location @site_retencao_api {"); n != 1 {
		t.Errorf("named location aparece %d vezes, want 1", n)
	}
}

func siteTarget(name, docroot, phpShort string) ProxyTarget {
	return ProxyTarget{
		IsSite: true, SiteName: name, DocRoot: docroot,
		PHPShort: phpShort, LocationName: "site_" + strings.ReplaceAll(name, "-", "_"),
	}
}
```

- [ ] **Step 5: Rodar e ver passar**

Run: `go test ./internal/nginx/ -run RenderFullstack -v`
Expected: PASS.

- [ ] **Step 6: Garantir embed dos novos templates**

Os templates são lidos via `GetTemplate`. Confirmar que o diretório `internal/nginx/templates/` é incluído por um `//go:embed`. Rodar:

Run: `go test ./internal/nginx/`
Expected: ok (se `GetTemplate` falhar com "template não encontrado", localizar a diretiva `//go:embed templates/*` em `internal/nginx/*.go` e confirmar que cobre os novos arquivos — `*.tmpl` já casa).

- [ ] **Step 7: Commit**

```bash
git add internal/nginx/fullstack.go internal/nginx/templates/vhost-proxy-fullstack.conf.tmpl internal/nginx/templates/vhost-proxy-fullstack-ssl.conf.tmpl internal/nginx/fullstack_test.go
git commit -m "feat(nginx): templates e gerador de vhost fullstack (porta + site/fastcgi)"
```

---

## Task 4: Ramificar `RegenerateProxyVhost` e fechar o ciclo proxyops

**Files:**
- Modify: `internal/proxyops/vhost.go`
- Test: `internal/proxyops/vhost_fullstack_test.go`

- [ ] **Step 1: Escrever o teste de regeneração fullstack (falhando)**

Create `internal/proxyops/vhost_fullstack_test.go`:

```go
package proxyops

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gabriel-sousa99/lerd/internal/config"
)

func TestRegenerateProxyVhost_Fullstack(t *testing.T) {
	// Isola DataDir/NginxConfD num HOME temporário.
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("XDG_DATA_HOME", filepath.Join(tmp, "share"))

	orig := findSiteFn
	findSiteFn = func(name string) (*config.Site, error) {
		return &config.Site{
			Name: "retencao-api", Domains: []string{"retencao-api.localhost"},
			Path: "/home/u/retencao-api", PublicDir: "public", PHPVersion: "8.2",
		}, nil
	}
	defer func() { findSiteFn = orig }()

	p := config.Proxy{
		Name: "retencao", Domains: []string{"retencao.localhost"},
		UpstreamPort: 9000, Secured: true,
		Routes: []config.Route{{Path: "/api", Site: "retencao-api"}},
	}
	if err := RegenerateProxyVhost(p); err != nil {
		t.Fatalf("RegenerateProxyVhost: %v", err)
	}

	confPath := filepath.Join(config.NginxConfD(), "retencao.localhost-ssl.conf")
	data, err := os.ReadFile(confPath)
	if err != nil {
		t.Fatalf("lendo vhost gerado: %v", err)
	}
	for _, want := range []string{"location ^~ /api {", "location @site_retencao_api {", "lerd-php82-fpm"} {
		if !strings.Contains(string(data), want) {
			t.Errorf("vhost não contém %q", want)
		}
	}
}
```

NOTE: se `config.NginxConfD()` não derivar de `XDG_DATA_HOME`/`HOME`, inspecionar `internal/config/paths.go` e ajustar os `t.Setenv` para as variáveis corretas que o lerd usa para o data dir.

- [ ] **Step 2: Rodar e ver falhar**

Run: `go test ./internal/proxyops/ -run RegenerateProxyVhost_Fullstack -v`
Expected: FAIL — o vhost gerado é o simples (sem `location ^~ /api`).

- [ ] **Step 3: Ramificar `RegenerateProxyVhost`**

Em `internal/proxyops/vhost.go`, substituir a função `RegenerateProxyVhost`:

```go
// RegenerateProxyVhost writes the nginx config for p based on Secured. For
// fullstack proxies (p.IsFullstack()) it resolves the route spec and renders
// the fullstack template; otherwise it keeps the simple single-upstream path
// (byte-identical to before).
func RegenerateProxyVhost(p config.Proxy) error {
	if p.IsFullstack() {
		spec, err := resolveProxySpec(p)
		if err != nil {
			return err
		}
		return nginx.GenerateFullstackProxyVhost(spec)
	}
	return nginx.GenerateProxyVhost(p.PrimaryDomain(), upstreamHost(p), p.UpstreamPort, p.Secured)
}
```

- [ ] **Step 4: Rodar e ver passar (inclui os testes da Task 2)**

Run: `go test ./internal/proxyops/ -run 'RegenerateProxyVhost_Fullstack|ResolveProxySpec' -v`
Expected: PASS.

- [ ] **Step 5: Suite completa dos pacotes tocados**

Run: `go test ./internal/config/ ./internal/nginx/ ./internal/proxyops/`
Expected: ok em todos.

- [ ] **Step 6: Commit**

```bash
git add internal/proxyops/vhost.go internal/proxyops/vhost_fullstack_test.go
git commit -m "feat(proxyops): RegenerateProxyVhost gera fullstack quando há rotas"
```

---

## Self-Review (resultado)

- **Cobertura do spec (este plano):** §3 modelo de dados → Task 1; §4 geração de vhost (porta + site/fastcgi, dedup de named location, HTTP_HOST same-origin) → Tasks 2-4; §5 validação (path, duplicado, target único, porta, site inexistente) → Tasks 1-2; §10 versão de PHP agnóstica (reusa `lerd-php<short>-fpm`) → Task 3. Retrocompat (proxy simples byte-idêntico) garantida por usar o template antigo intocado quando `len(Routes)==0`.
- **Fora deste plano (planos 2-3):** criar fullstack via `Add`/`Update`, flags de CLI, `routes` no POST/PUT, DTO, e toda a UI.
- **Placeholders:** nenhum. As duas NOTES (erro de teste em Task 2 Step 4; variáveis de env do data dir em Task 4 Step 1) são instruções concretas de ajuste, não lacunas.
- **Consistência de tipos:** `nginx.ProxyVhostSpec{Domain,Secured,Base,Routes}`, `nginx.ProxyTarget{IsSite,UpstreamHost,UpstreamPort,DocRoot,PHPShort,SiteName,LocationName}`, `nginx.ProxyRouteSpec{Path,Target}` usados de forma idêntica em `spec.go`, `fullstack.go`, templates e testes. `findSiteFn`, `resolveProxySpec`, `sanitizeLocation`, `config.Route`, `config.ValidateProxyRoutes`, `Proxy.IsFullstack` consistentes entre tasks.
- **Ordem de compilação:** Task 2 (spec.go) depende dos tipos da Task 3 (nginx). Documentado: escrever na ordem 1→2→3→4, mas só rodar/commitar os testes da Task 2 após a Task 3 compilar (Steps 5-6 da Task 2).
```
