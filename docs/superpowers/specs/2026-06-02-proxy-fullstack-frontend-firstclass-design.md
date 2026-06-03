# Proxy Fullstack — Frontend como cidadão de 1ª classe — Design

**Data:** 2026-06-02
**Projeto:** lerd (fork Oracle Edition), branch `oracle-oci8-support` · v1.21.2-oracle.25
**Status:** aprovado para planejamento (grill-me 2026-06-02)
**Estende:** [`2026-06-01-proxy-fullstack-design.md`](./2026-06-01-proxy-fullstack-design.md)

## 1. Problema e causa raiz

O design original (2026-06-01) resolveu o lado **API**: ao vincular um site Laravel
como API de um proxy fullstack, o `.env` do site é sincronizado para o domínio unificado
(`DomainScopedKeys`: `APP_URL`, `SESSION_DOMAIN`, `SANCTUM_STATEFUL_DOMAINS`, …). Isso
torna o cookie de sessão first-party same-origin no lado servidor.

Mas o **frontend ficou explicitamente fora de escopo** (design 2026-06-01, §2 não-objetivos
e §11.4, que só sincroniza sites de API). Na prática isso quebra o valor do same-origin:

- A SPA é modelada como `--port` (dev server) — **não** é um "site" do lerd, então
  `boundSites()`/`syncProxyEnv()` (`internal/proxyops/env.go`) **nunca tocam** o `.env`
  do projeto frontend.
- A SPA do `gestao-clientes` faz `baseURL: process.env.URL_API` apontando para
  `localhost:8000` / `gestao-clientes-api.localhost` — **cross-origin hardcoded**. O proxy
  entrega same-origin, mas a SPA continua chamando outra origem ⇒ CORS, cookie de terceiros,
  Sanctum quebra.
- O `fullstackHint` (`internal/cli/proxy.go:54-61`) literalmente diz *"ajuste manualmente
  se necessário"* — admitindo que só o lado API é automatizado.

**Causa raiz:** o frontend não tem representação no modelo de dados que permita ao lerd
sincronizar seu `.env`. O caminho do projeto frontend (`p.Path`) **já existe** no struct
`Proxy` (managed.go:39 o usa para o volume do quadlet), mas só é exigido/usado em
`--managed`, e nunca passa por env-sync.

## 2. Decisão (grill-me 2026-06-02)

O frontend vira **cidadão de 1ª classe** do proxy fullstack. Decisões travadas:

| # | Decisão | Valor escolhido |
|---|---------|-----------------|
| D1 | Modelo da SPA | **1ª classe** — `p.Path` representa o projeto frontend e participa do env-sync |
| D2 | Modo de servir em dev | **Dev server (HMR) atrás do proxy** (reusa managed quadlet; HMR no nível nginx já funciona) |
| D3 | Escopo do que o lerd escreve no frontend | **Só a chave de API-base no `.env`** — HMR fica como dica documentada |
| D4 | Valor gravado na API-base | **Origem unificada absoluta** (`https://<domínio-proxy>`), espelhando `APP_URL` |
| D5 | Completude de rotas | **Defaults mais ricos + doc** — incluir rotas de auth do `unimedvr/core` |
| D6 | Mecânica do sync | **Espelhar a máquina dos sites** — set fixo `only-if-present`, dispara no add/edit, reverte no rm/unbind |
| D7 | Escopo extra da rodada | **TrustProxies/HTTPS Laravel** (verificação) + **robustez gateway WSL** (hardening) |

**Não-objetivos (mantidos):** injetar config de HMR automaticamente (Quasar/Vite leem HMR
de `quasar.config.js`/`vite.config`, não do `.env` — fora do alcance do `.env`-sync);
build estático servido como site; reescrita de prefixo de path; limpeza de `ALLOWED_ORIGINS`.

## 3. Núcleo: env-sync do frontend (`internal/envfile` + `internal/proxyops`)

### 3.1 Conjunto de chaves de API-base (novo)

Espelhando `DomainScopedKeys`, um set **fixo e bounded**, **só reescrito se já presente**:

```go
// internal/envfile/envfile.go
// FrontendAPIBaseKeys lista as chaves de .env que apontam a base da API num
// projeto frontend (SPA). Só são reescritas se já existirem — mesma semântica
// e garantia de escopo de DomainScopedKeys. Nada fora deste set é tocado.
var FrontendAPIBaseKeys = []string{
    "URL_API",            // Quasar (gestao-clientes-spa)
    "VITE_API_URL",       // Vite genérico
    "VITE_APP_API_URL",   // Vite/Vue convenção comum
}
```

Novo helper, simétrico a `SyncPrimaryDomain`:

```go
// SyncFrontendAPIBase reescreve as chaves de FrontendAPIBaseKeys presentes no
// .env do projeto frontend para a origem unificada (scheme://domain, SEM /api —
// a SPA concatena seus próprios prefixos). Só toca chaves existentes, idempotente,
// best-effort se não houver .env.
func SyncFrontendAPIBase(projectPath, domain string, secured bool) error
```

> **D4:** valor = `scheme://domain` (raiz da origem). No `gestao-clientes`, a instância
> `api` faz `URL_API + '/api'` e o redirect de 401 navega para `${URL_API}/redirect` —
> ambos resolvem na origem unificada quando `URL_API = https://gestao-clientes.localhost`.

### 3.2 Gatilho e reversão (`internal/proxyops/env.go`)

Reusa a estrutura de `syncProxyEnv`/`unbindSitesEnv`. Hoje `boundSites()` só coleta
sites PHP; adicionar o caminho do frontend:

```go
// syncProxyEnv: além de sincronizar o .env de cada site de API (atual),
// se p.Path != "" sincroniza a API-base do projeto frontend:
func syncProxyEnv(p config.Proxy) error {
    domain := p.PrimaryDomain()
    for _, name := range boundSites(p) {           // sites de API (inalterado)
        if s, err := findSiteFn(name); err == nil {
            _ = syncEnvFn(s.Path, domain, p.Secured)
        }
    }
    if p.Path != "" {                              // NOVO: frontend
        _ = syncFrontendFn(p.Path, domain, p.Secured)
    }
    return nil
}
```

- **Add/Edit fullstack:** dispara junto com o sync dos sites de API.
- **Rm / edit que limpa `p.Path`:** reverter a API-base do frontend ao valor de dev
  isolado. Decisão de reversão: como o lerd não conhece a URL "original" de dev
  (`localhost:8000`), a reversão grava **a string vazia** (`URL_API=`) — relativo, neutro,
  não-quebrável — ou restaura de um backup `.env.lerd.bak` se existir. *(A decidir no Plano:
  vazio vs. backup. Recomendação: vazio + hint.)*

### 3.3 Liberar `--path` sem `--managed`

Hoje `--path` é "obrigatória se `--managed`" (`cli/proxy.go:128`). Para o env-sync funcionar
mesmo quando o dev roda `quasar dev` manualmente, **permitir `--path` standalone**:

- `--path` válido sem `--managed` → lerd não gera quadlet, mas **sincroniza** o `.env` do
  frontend e exibe o caminho no `proxy ls`.
- `--managed` sem `--path` → erro (inalterado).

## 4. Defaults de rota mais ricos (D5) — `internal/cli/proxy.go`

`defaultAPIPaths()` passa a cobrir as convenções Laravel + as rotas de auth do pacote
compartilhado `vendor/unimedvr/core`:

```go
func defaultAPIPaths() []string {
    return []string{
        "/api", "/sanctum", "/broadcasting", "/storage",
        "/redirect",      // Core/Routes/web.php  GET /redirect/{profile?}  → SSO entrypoint (alvo do 401 da SPA)
        "/authenticate",  // Core/Routes/web.php  GET /authenticate/{profile?} + api.php POST → SSO callback
        "/login", "/logout", "/up",  // convenções Laravel/Breeze/healthcheck
    }
}
```

> **Por que `/redirect` e `/authenticate` juntos:** o fluxo SSO do `unimedvr/core` é
> `401 → GET /redirect → provider → GET /authenticate → sessão`. Ambas são **rotas web no
> root** (fora de `/api`); sem estarem no route-set, caem na base (dev server SPA) → 404.
> Como o `unimedvr/core` é usado por todos os apps Laravel da org, baquear no default é
> justificado para este fork.

- Rotas custom continuam exigindo `--api-path` explícito (documentar).
- `/redirect` precisa **existir** na API (a análise de auth achou referência sem rota em
  apps que não montam o `core/web.php`) — isso é app-side, mas o doc do lerd deve alertar.

## 5. Verificações de escopo extra (D7)

### 5.1 TrustProxies / HTTPS Laravel — **verificação + doc**

Achado (gestao-clientes-api): **não há** `trustProxies()` em `bootstrap/app.php`. Porém o
template fastcgi (`vhost-proxy-fullstack-ssl.conf.tmpl:44,78`) força
`fastcgi_param HTTPS on` + `HTTP_X_FORWARDED_PROTO $real_forwarded_proto` +
`HTTP_HOST $real_forwarded_host`. Consequências:

- ✅ Laravel vê `$_SERVER['HTTPS']='on'` ⇒ emite cookie `Secure`, gera URLs https. **Cookie
  Sanctum same-origin funciona sem TrustProxies.**
- ⚠️ Sem TrustProxies, `$request->getClientIp()` retorna o IP do container nginx (não o
  cliente real). Impacto baixo em dev; relevante se a app fizer rate-limit/log por IP.

**Ação:** sem mudança de código no lerd. Documentar em `docs/features/` a recomendação de
`->trustProxies(at: '*')` (ou a sub-rede do container) para apps que dependem de IP do
cliente. Verificar `SESSION_SECURE_COOKIE` coerente com o proxy `secured`.

### 5.2 Robustez do gateway WSL — **hardening review**

Os fixes #S2082/#S2084 (eth0 vs `10.255.255.254`, filtro de loopback) são recentes.
Revisar `internal/podman/hosts.go` `DetectHostGatewayIP()` / `primaryLANIP()`:

- Confirmar que o fallback `169.254.1.2` / `10.0.2.2` não mascara falha silenciosa.
- Garantir que o IP detectado é estável entre reboots de WSL (eth0 muda?).
- Cobrir com teste: iface só-loopback, múltiplas ifaces, eth0 ausente.
- Verificar se `host.containers.internal` resolve consistentemente no container nginx para
  o `proxy_pass` da base SPA (o dev server roda no host).

## 6. Plano de testes

**Go**
- `envfile`: `SyncFrontendAPIBase` reescreve só chaves presentes; idempotente; no-op sem
  `.env`; valor = `scheme://domain` sem `/api`; prova de escopo (não toca chaves fora do set).
- `proxyops`: `syncProxyEnv` sincroniza frontend quando `p.Path != ""`; reversão no rm;
  fullstack sem `p.Path` inalterado.
- `cli`: `--path` aceito sem `--managed`; `--managed` sem `--path` erra; novos defaults de path.
- **regressão:** proxy simples e fullstack-só-API inalterados.

**E2E manual (gestao-clientes)**
- Criar proxy fullstack `gestao-clientes.localhost` (SPA=`--port 9000 --path <spa>`,
  API=`--api-site gestao-clientes-api`).
- Verificar: `URL_API` da SPA reescrito para `https://gestao-clientes.localhost`;
  `/sanctum/csrf-cookie` → cookie em `gestao-clientes.localhost`; login SSO via
  `/redirect`→`/authenticate`; `GET /api/user` autenticado **sem** CORS/changeOrigin.
- HMR: confirmar necessidade do snippet `devServer.hmr.clientPort=443/protocol=wss` no
  `quasar.config.js` (dica documentada, não automatizada — D3).

## 7. Arquivos a tocar

| Arquivo | Mudança |
|---------|---------|
| `internal/envfile/envfile.go` | `FrontendAPIBaseKeys` + `SyncFrontendAPIBase()` |
| `internal/proxyops/env.go` | `syncProxyEnv`/`unbindSitesEnv` cobrem `p.Path` (frontend) |
| `internal/cli/proxy.go` | `defaultAPIPaths()` mais ricos; liberar `--path` sem `--managed`; ajustar `fullstackHint` |
| `internal/podman/hosts.go` | hardening + testes do gateway WSL (§5.2) |
| `docs/features/*.md` | doc HMR snippet, TrustProxies, rotas custom |
| `CHANGELOG.md` | entrada oracle.next |

## 8. Limitações conhecidas

- HMR não é automatizado (D3): Quasar/Vite leem HMR fora do `.env`.
- Reversão da API-base do frontend não restaura a URL de dev original (grava vazio/backup).
- `/redirect` deve existir na API (app-side); o lerd só roteia.
- Sets de chave de API-base (`FrontendAPIBaseKeys`) e de rotas default são opinativos para
  o stack Unimed (Quasar + `unimedvr/core`); projetos com outras convenções usam `--api-path`
  e devem ter sua chave de env adicionada ao set.
