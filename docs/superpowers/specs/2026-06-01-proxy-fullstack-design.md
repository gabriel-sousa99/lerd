# Proxy Fullstack (SPA + API same-origin) — Design

**Data:** 2026-06-01
**Projeto:** lerd (fork Oracle Edition), branch `oracle-oci8-support`
**Status:** aprovado para planejamento

## 1. Problema e causa raiz

A maioria dos projetos é um par **SPA + API**. Hoje cada lado vira um proxy/site
separado (`retencao-spa.localhost` e `retencao-api.localhost`). Como são **origens
distintas** e `.localhost` é tratado como public-suffix pelos navegadores, o cookie
de sessão emitido pela API não é compartilhado com o SPA. Resultado: sessão, CSRF e
Sanctum quebram. Hoje o contorno é `changeOrigin`/CORS no dev server, o que é frágil.

**Solução:** servir SPA e API sob **uma única origem** (`retencao.localhost`),
separados por **path**. O cookie vira first-party same-origin; some o CORS, o
`SameSite=None` e o `changeOrigin`.

Convenção de roteamento:

```
retencao.localhost/            → SPA   (dev server Vite/Quasar)
retencao.localhost/api         → API   (site Laravel servido pelo lerd)
retencao.localhost/sanctum     → API
retencao.localhost/broadcasting→ API
retencao.localhost/storage     → API
```

Sem strip de prefixo: o Laravel já registra rotas sob `/api`, `/sanctum`,
`/broadcasting` e serve `/storage`. O path chega ao backend intacto.

## 2. Objetivos e não-objetivos

**Objetivos**
- Um proxy unificado que roteia paths para upstreams diferentes na mesma origem.
- Cada lado (SPA e cada grupo de paths da API) pode apontar para **um site do lerd**
  (sem porta — lerd resolve FPM/docroot) **ou** para uma **porta** (dev server externo).
- Configuração e visualização claras no dashboard, CLI e API HTTP.
- Retrocompatibilidade total: proxy "simples" (1 domínio → 1 upstream) inalterado.

**Não-objetivos (v1)**
- Reescrita/strip de prefixo de path.
- Incluir customizações `custom.d/*.conf` do site-API dentro do vhost unificado.
- Servir SPA estático buildado como "site" (o caso de dev é porta; um SPA buildado
  deve ser um site normal do lerd, fora deste escopo).
- Migrar/remover automaticamente os proxies/sites separados existentes.

## 3. Modelo de dados (`internal/config/proxy.go`)

Um *target* é **exatamente um** de: `site` (fastcgi para um site do lerd) ou
`upstream_port` (+ `upstream_host` opcional, proxy_pass para host:porta).

```go
type Route struct {
    Path         string `yaml:"path"`                    // ex: "/api"; começa com "/", != "/"
    Site         string `yaml:"site,omitempty"`           // nome de um site do lerd  (target = site)
    UpstreamPort int    `yaml:"upstream_port,omitempty"`  // target = porta
    UpstreamHost string `yaml:"upstream_host,omitempty"`  // default host.containers.internal
}

type Proxy struct {
    // ...campos existentes...
    Site   string  `yaml:"site,omitempty"`   // base (/) como site do lerd (alternativa a UpstreamPort)
    Routes []Route `yaml:"routes,omitempty"` // vazio = proxy simples (comportamento atual)
}
```

- **Proxy simples:** `Routes` vazio, base = `UpstreamPort`/`UpstreamHost`. Idêntico ao atual.
- **Proxy fullstack:** `len(Routes) > 0`. Não há campo `type`; a presença de rotas define o modo.
- **Base (`/`)** = target do próprio `Proxy` (`UpstreamPort`/`UpstreamHost` **ou** `Site`).

Exemplo (`proxies.yaml`):

```yaml
- name: retencao
  domains: [retencao.localhost]
  secured: true
  upstream_port: 9000                 # SPA dev server (porta)
  routes:
    - { path: /api,          site: retencao-api }
    - { path: /sanctum,      site: retencao-api }
    - { path: /broadcasting, site: retencao-api }
    - { path: /storage,      site: retencao-api }
```

## 4. Geração do vhost (`internal/nginx`)

`proxyVhostData` ganha a lista de rotas resolvidas. O gerador resolve cada target:

- **Target = porta** → bloco `proxy_pass` (reusa o template proxy atual, com upgrade de
  websocket e headers `Host $host`, `X-Forwarded-Proto`).
- **Target = site** → o lerd faz lookup do site (`config.FindSite`), obtém `path`,
  `public_dir` e `php_version`, e gera um bloco **front-controller** apontando para o
  FPM daquele site.

As `location` da API (prefixos) usam `^~` (prefixo prioritário) e caem num named
location por site. O `location /` (catch-all SPA) vem por último.

```nginx
server {
    listen 443 ssl;
    server_name retencao.localhost;
    ssl_certificate     /etc/nginx/certs/retencao.localhost.crt;
    ssl_certificate_key /etc/nginx/certs/retencao.localhost.key;

    # --- rotas de API (target = site retencao-api, PHP 8.2) ---
    location ^~ /api          { try_files $uri @site_retencao_api; }
    location ^~ /sanctum      { try_files $uri @site_retencao_api; }
    location ^~ /broadcasting { try_files $uri @site_retencao_api; }
    location ^~ /storage      { root /home/user/retencao-api/public; try_files $uri @site_retencao_api; }

    location @site_retencao_api {
        root /home/user/retencao-api/public;
        include fastcgi_params;
        fastcgi_param SCRIPT_FILENAME $document_root/index.php;
        fastcgi_param SCRIPT_NAME      /index.php;
        fastcgi_param HTTP_HOST            $real_forwarded_host;   # = retencao.localhost
        fastcgi_param HTTP_X_FORWARDED_PROTO $real_forwarded_proto;
        fastcgi_pass lerd-php82-fpm:9000;
    }

    # --- SPA catch-all (target = porta 9000, HMR websocket) ---
    location / {
        proxy_pass http://host.containers.internal:9000;
        proxy_set_header Host $host;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        # + headers padrão do template proxy
    }
}
```

**Correção do cookie (o ponto central):** o `fastcgi_param HTTP_HOST $real_forwarded_host`
faz o Laravel enxergar `retencao.localhost` (o host pelo qual foi acessado), então o
cookie de sessão é emitido para o domínio unificado — first-party, same-origin. Não é
preciso `SANCTUM_STATEFUL_DOMAINS` cross-host nem CORS.

**Versão do PHP:** vem do site selecionado (`lerd-php<versão>-fpm`). Targets de API com
sites de versões diferentes geram named locations distintos — suportado.

**Templates:** `vhost-proxy.conf.tmpl` e `vhost-proxy-ssl.conf.tmpl` passam a iterar
`Routes` (porta → bloco proxy; site → named location front-controller) antes do
`location /`. Proxy sem rotas gera saída **idêntica** à atual.

## 5. Validação (`internal/proxyops`)

- Cada `Path` começa com `/` e é `!= "/"`.
- Paths únicos dentro do proxy.
- Cada target (base e rotas) tem **exatamente um** de `site`/`upstream_port`.
- `site` referenciado deve existir (`config.FindSite`); erro claro se não.
- Porta válida (1..65535); avisar se a porta já está em uso por outro serviço do lerd
  (ex.: 9000 ocupada pelo `lerd-rustfs`) — aviso, não bloqueio.
- Ordenar rotas por path decrescente na geração (defensivo; nginx usa longest-prefix).

## 6. CLI (`internal/cli/proxy.go`)

`proxy add` ganha flags fullstack (a presença de `--route`/`--api-site`/`--api-port`
ativa o modo):

```bash
# API como site do lerd (default de paths se nenhum --api-path for passado)
lerd proxy add retencao.localhost --port 9000 \
     --api-site retencao-api \
     --api-path /api --api-path /sanctum --api-path /broadcasting --api-path /storage

# API numa porta externa
lerd proxy add retencao.localhost --port 9000 --api-port 8000
```

- `proxy edit` aceita os mesmos flags para adicionar/substituir rotas.
- `proxy ls` marca `[fullstack]` quando há rotas.
- Defaults de paths quando `--api-site`/`--api-port` presente e nenhum `--api-path`:
  `/api /sanctum /broadcasting /storage`.

## 7. API HTTP (`internal/ui/proxy_api.go`)

- POST/PUT aceitam `routes: [{path, site?, upstream_port?, upstream_host?}]` e `site?`
  para a base. Reusa a infra de PUT esparso já existente.
- `proxyDTO` expõe `routes` e `site`, além de um booleano derivado `fullstack`.
- GET `/api/sites` já existe e alimenta o dropdown de sites no modal.

## 8. UI do dashboard (`internal/ui/web`, Svelte 5 + Tailwind 4)

### 8.1 Modal Add/Edit (`ProxyAddModal.svelte`) — layout aprovado

- Segmented control **Simples | Fullstack (SPA + API)**.
- Em Fullstack:
  - **Domínio base**.
  - **SPA** (catch-all `/`): toggle **Site | Porta** (default **Porta**) → input de porta
    *ou* dropdown de site.
  - **API**: toggle **Site | Porta** (default **Site**) → dropdown de sites do lerd
    (mostra `nome · PHP x.y`) *ou* input de porta; + chips editáveis de paths
    pré-preenchidos com `/api /sanctum /broadcasting /storage`.
  - **Mapa de origem ao vivo**: lista cada `path → target` (`/api → site retencao-api`,
    `/ → porta :9000`) atualizando conforme os campos mudam.
- **Detecção de sufixo `-api`:** se o nome/domínio digitado termina em `-api`, mostra um
  hint sugerindo modo Fullstack e pré-preenche o domínio base sem o sufixo
  (`retencao-api → retencao.localhost`).
- Edição: pré-preenche do proxy; domínio bloqueado (como hoje).

### 8.2 Seção Sites (`tabs/sites/SiteDetail.svelte`) — afford de proxy

Bloco **"Proxy fullstack"** perto do link e da versão PHP, **em todos os sites**:

- **Sem proxy que use este site:** texto curto + botão **+ Criar proxy fullstack**, que
  abre o `ProxyAddModal` em modo Fullstack **pré-preenchido com este site como API** e
  domínio base = nome do site sem o sufixo `-api` (editável).
- **Site já é a API de um proxy:** mostra o domínio unificado (`↗ retencao.localhost`) +
  os paths, com botão **Editar** que abre o modal de edição daquele proxy.

A store de proxies ganha um helper para achar o proxy cujo `routes[].site == <site>`.

### 8.3 Painel de detalhes do proxy (`proxies/ProxyDetailPanel.svelte`)

- Badge `fullstack` no header quando há rotas.
- Seção **Roteamento**: uma linha por rota e pela base (`path → target`), com tag
  SPA/API e um ponto de status (verde = porta/upstream respondendo; cinza = sem
  resposta) via polling leve.
- Linha "Cookie: first-party (same-origin)" nos detalhes.

### 8.4 Store/tipos (`stores/proxies.ts`)

`Proxy` ganha `routes?: Route[]` e `site?: string`; `Route = { path; site?; upstream_port?; upstream_host? }`.
`createProxy`/`updateProxy` serializam `routes`.

## 9. Plano de testes

**Go**
- `config`: round-trip YAML com `routes` (site e porta); proxy sem rotas inalterado.
- `proxyops`: validação (path inválido, duplicado, target ambíguo/ausente, site
  inexistente); geração de vhost — snapshot de múltiplos `location` (porta + site).
- `cli`: `proxy add --api-site` e `--api-port` criam rotas; defaults de paths.
- `api`: POST/PUT com `routes`; `proxyDTO.fullstack`.
- **regressão:** proxy sem `routes` gera vhost byte-idêntico ao atual.

**Frontend (vitest)**
- Modal: toggle Site/Porta, chips de paths, mapa de origem, detecção `-api`.
- Sites: estados 1/2 do bloco de proxy.

**E2E manual (fixtures já criadas)**
- `retencao-api` (Laravel 13 + Sanctum + **sqlite**, site do lerd) e `retencao-spa`
  (Quasar, dev server em porta livre — `:9000` pode estar ocupada pelo `lerd-rustfs`,
  usar ex. `:9100`).
- Criar proxy fullstack `retencao.localhost` (SPA=porta, API=site `retencao-api`).
- Verificar: `GET /api/ping` via `retencao.localhost`; fluxo `/sanctum/csrf-cookie` →
  cookie `XSRF-TOKEN`/`laravel-session` em `retencao.localhost`; `GET /api/user`
  autenticado funcionando **sem** `changeOrigin`/CORS.

## 10. Considerações e limitações

- **HTTPS + cookie Secure:** com o proxy `secured`, o `X-Forwarded-Proto=https` deixa o
  Laravel emitir cookie `Secure`. O app deve usar `SESSION_SECURE_COOKIE` coerente; é
  config do app, fora do lerd.
- **`custom.d` do site-API** não é incluído no vhost unificado (v1).
- **Conflito de porta** do SPA: validar/avisar; não fixar `9000`.
- **Coexistência:** o site `retencao-api.localhost` standalone continua funcionando para
  testar a API diretamente; o fullstack só adiciona o ponto de entrada unificado.
```
