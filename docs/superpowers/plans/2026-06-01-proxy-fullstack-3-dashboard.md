# Proxy Fullstack — Plano 3: Dashboard (Svelte)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Expor o proxy fullstack no dashboard: modo "Fullstack (SPA+API)" no modal de add/edit (layout A com mapa de rotas ao vivo e detecção de `-api`), visualização do roteamento no painel de detalhes, e o bloco "Proxy fullstack" em todos os sites.

**Architecture:** Svelte 5 (runes) + Tailwind 4. O backend (Planos 1-2) já entrega o contrato: `proxyDTO` com `routes`/`site`/`fullstack`; POST/PUT aceitam `routes`/`site`; `GET /api/sites` lista sites para o picker. Frontend: estender tipos da store, extrair um helper puro `buildApiRoutes` (espelha a CLI, testável via vitest), e renderizar a UI validada nos mockups.

**Tech Stack:** Svelte 5, TypeScript, Tailwind 4, Vitest. Build/check: `npm run build` e `npx vitest run` em `internal/ui/web`.

**Decisões herdadas:** §11 do spec; mockups validados (modal layout A, estados do bloco em Sites, painel com rotas estáticas).

---

## File Structure

- `internal/ui/web/src/stores/proxies.ts` (modify) — `Route` interface; `Proxy` ganha `site?`/`routes?`/`fullstack?`; `CreateProxyInput`/`UpdateProxyInput` ganham `site?`/`routes?`.
- `internal/ui/web/src/stores/modals.ts` (modify) — `ModalState.prefill?`; `openProxyAddModal(prefill?)`.
- `internal/ui/web/src/lib/fullstack.ts` (create) — `buildApiRoutes()` puro + `defaultApiPaths()`.
- `internal/ui/web/src/lib/fullstack.test.ts` (create) — testes vitest do helper.
- `internal/ui/web/src/modals/ProxyAddModal.svelte` (modify) — modo Fullstack.
- `internal/ui/web/src/tabs/proxies/ProxyDetailPanel.svelte` (modify) — badge + seção Roteamento.
- `internal/ui/web/src/tabs/sites/SiteDetail.svelte` (modify) — bloco "Proxy fullstack".

Todos os comandos rodam de dentro de `internal/ui/web` (onde está o `package.json`).

---

## Task 1: contrato da store de proxies

**Files:** modify `internal/ui/web/src/stores/proxies.ts`

- [ ] **Step 1: adicionar a interface `Route` e os campos**

No topo, após os imports, antes de `export interface Proxy`:
```ts
export interface Route {
  path: string;
  site?: string;
  upstream_port?: number;
  upstream_host?: string;
}
```
No `export interface Proxy { ... }`, adicionar após `autostart: boolean;`:
```ts
  site?: string;
  routes?: Route[];
  fullstack?: boolean;
```
No `export interface CreateProxyInput { ... }`, adicionar após `autostart?: boolean;`:
```ts
  site?: string;
  routes?: Route[];
```
No `export interface UpdateProxyInput { ... }`, adicionar após `autostart?: boolean;`:
```ts
  site?: string;
  routes?: Route[];
```

- [ ] **Step 2: verificar tipos**

Run (de `internal/ui/web`): `npx tsc --noEmit` (ou `npm run check` se existir no package.json). Expected: sem novos erros.

- [ ] **Step 3: commit**
```bash
git add internal/ui/web/src/stores/proxies.ts
git commit -m "feat(ui/store): tipos de rota/fullstack no Proxy e nos inputs de create/update"
```

---

## Task 2: helper puro `buildApiRoutes` + testes

**Files:** create `internal/ui/web/src/lib/fullstack.ts` and `internal/ui/web/src/lib/fullstack.test.ts`

- [ ] **Step 1: escrever o teste — create `internal/ui/web/src/lib/fullstack.test.ts`**
```ts
import { describe, it, expect } from 'vitest';
import { buildApiRoutes, defaultApiPaths } from './fullstack';

describe('buildApiRoutes', () => {
  it('default paths for a site target', () => {
    const routes = buildApiRoutes({ mode: 'site', site: 'retencao-api', port: 0, paths: [] });
    expect(routes.map((r) => r.path)).toEqual(defaultApiPaths());
    expect(routes[0]).toEqual({ path: '/api', site: 'retencao-api' });
  });
  it('port target with custom paths', () => {
    const routes = buildApiRoutes({ mode: 'port', site: '', port: 8000, paths: ['/api'] });
    expect(routes).toEqual([{ path: '/api', upstream_port: 8000 }]);
  });
  it('site mode without a site name yields no routes', () => {
    expect(buildApiRoutes({ mode: 'site', site: '', port: 0, paths: [] })).toEqual([]);
  });
  it('port mode without a port yields no routes', () => {
    expect(buildApiRoutes({ mode: 'port', site: '', port: 0, paths: [] })).toEqual([]);
  });
});
```

- [ ] **Step 2: rodar → FAIL** (de `internal/ui/web`): `npx vitest run src/lib/fullstack.test.ts`

- [ ] **Step 3: implementar — create `internal/ui/web/src/lib/fullstack.ts`**
```ts
import type { Route } from '$stores/proxies';

export function defaultApiPaths(): string[] {
  return ['/api', '/sanctum', '/broadcasting', '/storage'];
}

export interface ApiTargetInput {
  mode: 'site' | 'port';
  site: string;
  port: number;
  paths: string[];
}

// buildApiRoutes turns the modal's fullstack inputs into Route[]. Returns an
// empty array when the API target is incomplete (no site / no port), so the
// caller can treat that as "not fullstack". Paths default to defaultApiPaths().
export function buildApiRoutes(input: ApiTargetInput): Route[] {
  const hasSite = input.mode === 'site' && input.site.trim() !== '';
  const hasPort = input.mode === 'port' && input.port > 0;
  if (!hasSite && !hasPort) return [];
  const paths = input.paths.length > 0 ? input.paths : defaultApiPaths();
  return paths.map((p) =>
    hasSite ? { path: p, site: input.site.trim() } : { path: p, upstream_port: input.port }
  );
}

// stripApiSuffix maps a site/domain name to a suggested unified base domain:
// "retencao-api" → "retencao.localhost". Idempotent-ish for non -api names.
export function suggestUnifiedDomain(name: string): string {
  const base = name.replace(/\.localhost$/, '').replace(/-api$/, '');
  return `${base}.localhost`;
}
```

- [ ] **Step 4: rodar → PASS** `npx vitest run src/lib/fullstack.test.ts`

- [ ] **Step 5: commit**
```bash
git add internal/ui/web/src/lib/fullstack.ts internal/ui/web/src/lib/fullstack.test.ts
git commit -m "feat(ui): helper puro buildApiRoutes/defaultApiPaths/suggestUnifiedDomain + testes"
```

---

## Task 3: modal store — prefill para "criar fullstack a partir do site"

**Files:** modify `internal/ui/web/src/stores/modals.ts`

- [ ] **Step 1: adicionar o tipo de prefill e estender `openProxyAddModal`**

Após `export type LANAction`, adicionar:
```ts
export interface ProxyAddPrefill {
  domain?: string;       // domínio base sugerido (ex: retencao.localhost)
  fullstack?: boolean;   // abre já no modo Fullstack
  apiSite?: string;      // site pré-selecionado como API
}
```
No `export interface ModalState`, adicionar após `proxy?: Proxy;`:
```ts
  prefill?: ProxyAddPrefill;
```
Substituir `openProxyAddModal`:
```ts
export function openProxyAddModal(prefill?: ProxyAddPrefill) {
  modal.set({ kind: 'proxyAdd', prefill });
}
```

- [ ] **Step 2: tipos** `npx tsc --noEmit` (de `internal/ui/web`) → sem novos erros. (chamadas existentes `openProxyAddModal()` continuam válidas — `prefill` é opcional.)

- [ ] **Step 3: commit**
```bash
git add internal/ui/web/src/stores/modals.ts
git commit -m "feat(ui/store): prefill opcional no openProxyAddModal (domínio/fullstack/apiSite)"
```

---

## Task 4: ProxyAddModal — modo Fullstack (layout A)

**Files:** modify `internal/ui/web/src/modals/ProxyAddModal.svelte`

Contexto: componente Svelte 5 (runes `$state`/`$derived`/`$effect`). Hoje tem `domain`, `port`, `path`, `upstreamHost`, `managed`, `cmd`, `nodeVersion`, `autostart`, `noSecure`, e o `submit()` que monta `createProxy`/`updateProxy`. Vamos ADICIONAR o modo fullstack sem quebrar o modo simples.

- [ ] **Step 1: ler o arquivo atual** (`internal/ui/web/src/modals/ProxyAddModal.svelte`) para confirmar os pontos de inserção.

- [ ] **Step 2: script — novos estados e imports**

Adicionar aos imports:
```ts
  import { buildApiRoutes, defaultApiPaths, suggestUnifiedDomain } from '$lib/fullstack';
  import { sites, loadSites } from '$stores/sites';
```
Adicionar aos `$state`:
```ts
  let fullstack = $state(false);
  let spaMode = $state<'port' | 'site'>('port');   // base SPA
  let spaSite = $state('');
  let apiMode = $state<'site' | 'port'>('site');    // API target
  let apiSite = $state('');
  let apiPort = $state<number>(8000);
  let apiPaths = $state<string[]>(defaultApiPaths());
```
Carregar sites ao montar (para os pickers):
```ts
  $effect(() => { void loadSites(); });
```
Pré-preencher a partir do prefill (quando abrir em modo add com prefill):
```ts
  let prefillApplied = $state(false);
  $effect(() => {
    const pf = $modal.prefill;
    if (!editing && pf && !prefillApplied) {
      prefillApplied = true;
      if (pf.domain) domain = pf.domain;
      if (pf.fullstack) fullstack = true;
      if (pf.apiSite) { apiMode = 'site'; apiSite = pf.apiSite; }
    }
  });
```
Pré-preencher a partir de um proxy fullstack existente (edição) — estender o `$effect` que hidrata de `existing` (logo após `noSecure = !p.secured;`):
```ts
      fullstack = (p.routes?.length ?? 0) > 0 || !!p.site;
      if (p.site) { spaMode = 'site'; spaSite = p.site; } else { spaMode = 'port'; }
      const apiRoutes = (p.routes ?? []).filter((r) => r.site || r.upstream_port);
      if (apiRoutes.length) {
        apiPaths = apiRoutes.map((r) => r.path);
        if (apiRoutes[0].site) { apiMode = 'site'; apiSite = apiRoutes[0].site; }
        else { apiMode = 'port'; apiPort = apiRoutes[0].upstream_port ?? 8000; }
      }
```
Detecção de sufixo `-api` (sugestão de domínio base no modo add):
```ts
  $derived.by(() => {});
  const apiHint = $derived(!editing && /-api(\.localhost)?$/.test(domain.trim()));
```
Mapa de origem ao vivo (derivado):
```ts
  const routePreview = $derived(
    fullstack
      ? buildApiRoutes({ mode: apiMode, site: apiSite, port: apiPort, paths: apiPaths })
      : []
  );
```

- [ ] **Step 3: script — montar o payload fullstack no `submit()`**

No ramo de criação (`createProxy({...})`), quando `fullstack`, incluir `site`/`routes` e a base:
```ts
        const routes = buildApiRoutes({ mode: apiMode, site: apiSite, port: apiPort, paths: apiPaths });
        const created = await createProxy({
          domain: domain.trim(),
          port: spaMode === 'port' ? port : 0,
          path: path.trim() || undefined,
          no_secure: noSecure,
          managed: fullstack ? false : managed,
          cmd: !fullstack && managed ? cmd : undefined,
          node_version: !fullstack && managed ? nodeVersion : undefined,
          autostart: !fullstack && managed ? autostart : false,
          site: fullstack && spaMode === 'site' ? spaSite.trim() : undefined,
          routes: fullstack ? routes : undefined
        });
```
No ramo de edição, quando `fullstack` (ou quando havia rotas), enviar `routes` (substituição integral) e `site`:
```ts
        if (fullstack) {
          patch.routes = buildApiRoutes({ mode: apiMode, site: apiSite, port: apiPort, paths: apiPaths });
          patch.site = spaMode === 'site' ? spaSite.trim() : '';
          if (spaMode === 'port' && port !== existing.upstream_port) patch.port = port;
        }
```
(Manter o restante do patch esparso para campos simples.)

Ajustar `canSubmit` para o modo fullstack (a base precisa de porta OU site; a API precisa de site OU porta):
```ts
  const canSubmit = $derived.by(() => {
    if (!editing && domain.trim().length === 0) return false;
    if (fullstack) {
      const baseOk = spaMode === 'port' ? port > 0 && port <= 65535 : spaSite.trim() !== '';
      const apiOk = apiMode === 'site' ? apiSite.trim() !== '' : apiPort > 0 && apiPort <= 65535;
      return baseOk && apiOk;
    }
    return port > 0 && port <= 65535;
  });
```

- [ ] **Step 4: markup — toggle Simples|Fullstack + campos**

Logo após o campo de Domínio (e antes do campo Porta), adicionar o segmented control e, quando `fullstack`, esconder os campos simples (porta/managed) e mostrar os fullstack. Use as classes Tailwind já presentes no projeto (`lerd-red`, `lerd-border`, etc.). Estrutura:
- segmented `Simples | Fullstack (SPA + API)` ligando `fullstack` (desabilitado em edição se quiser travar — NÃO trave; permitir conversão);
- hint `-api` (quando `apiHint`): um aviso clicável "Detectado -api — usar domínio base `{suggestUnifiedDomain(domain)}`" que seta `domain = suggestUnifiedDomain(domain)` e `fullstack = true`;
- bloco **SPA**: toggle `Site|Porta` (liga `spaMode`); se porta → reusa o input `port`; se site → `<select bind:value={spaSite}>` populado de `$sites` (`{#each $sites as s}<option value={s.domain ?? s.name}>`… use o identificador que o backend espera, que é o **nome** do site; ver nota abaixo);
- bloco **API**: toggle `Site|Porta` (liga `apiMode`); se site → `<select bind:value={apiSite}>` de `$sites`; se porta → input `apiPort`; depois os **chips** de `apiPaths` (editáveis: remover por chip, adicionar via input). Para v1, um input de texto que aceita paths separados por espaço sincronizando `apiPaths` é suficiente (chips são incremento visual);
- **mapa de origem**: lista `routePreview` (`{#each routePreview as r}` → `r.path → r.site ? 'site '+r.site : ':'+r.upstream_port`) + a linha base (`/ → ` porta/site).

NOTA sobre identificador do site: o backend resolve `route.site`/`proxy.site` por **nome de site** (`config.FindSite(name)`). O objeto `Site` no frontend tem `name?` e `domain`. Use `s.name ?? s.domain` como value do option e rotule com `s.domain` + (se houver) versão PHP. Confirme no `sites` store qual campo é o nome canônico antes de finalizar.

Quando `fullstack`, ocultar o checkbox `managed` e o fieldset managed (fullstack é ortogonal a managed; no v1 fullstack não usa managed).

- [ ] **Step 5: build + check**

De `internal/ui/web`: `npx vitest run` (helper tests verdes), `npx tsc --noEmit` (sem erros novos), `npm run build` (compila). Se o projeto tiver `npm run check` (svelte-check), rodar também.

- [ ] **Step 6: commit**
```bash
git add internal/ui/web/src/modals/ProxyAddModal.svelte
git commit -m "feat(ui): modo Fullstack no ProxyAddModal (toggle, pickers de site, paths, mapa de rotas, detecção -api)"
```

---

## Task 5: ProxyDetailPanel — badge + seção Roteamento (estática)

**Files:** modify `internal/ui/web/src/tabs/proxies/ProxyDetailPanel.svelte`

- [ ] **Step 1: ler o arquivo atual** para ver o header (`{proxy.domain}`, link `url`, `DetailButton`s) e a seção "Proxy"/InfoRow.

- [ ] **Step 2: badge fullstack no header**

Ao lado do `<h1>{proxy.domain}</h1>`, quando `proxy.fullstack`, renderizar um badge:
```svelte
{#if proxy.fullstack}
  <span class="ml-2 align-middle text-[10px] uppercase tracking-wide bg-lerd-red/15 text-lerd-red border border-lerd-red/30 rounded px-1.5 py-0.5">fullstack</span>
{/if}
```

- [ ] **Step 3: seção Roteamento (estática, sem polling)**

Quando `proxy.fullstack`, adicionar uma `<section>` (mesmo padrão das demais — `px-6 py-4 ... border-b`), título `Roteamento`, listando a base e cada rota:
```svelte
{#if proxy.fullstack}
  <section class="px-6 py-4 space-y-1.5 border-b border-gray-100 dark:border-lerd-border">
    <h2 class="text-xs font-semibold uppercase tracking-wider text-gray-500">Roteamento (origem única)</h2>
    {#each (proxy.routes ?? []) as r}
      <div class="flex items-center gap-2 text-xs font-mono">
        <span class="text-sky-500 min-w-[110px]">{r.path}</span>
        <span class="text-gray-400">→</span>
        <span class="text-gray-600 dark:text-gray-300">{r.site ? `site ${r.site}` : `:${r.upstream_port}`}</span>
        <span class="ml-auto text-[10px] uppercase tracking-wide bg-rose-500/10 text-rose-500 rounded px-1.5 py-0.5">API</span>
      </div>
    {/each}
    <div class="flex items-center gap-2 text-xs font-mono">
      <span class="text-emerald-500 min-w-[110px]">/ (resto)</span>
      <span class="text-gray-400">→</span>
      <span class="text-gray-600 dark:text-gray-300">{proxy.site ? `site ${proxy.site}` : `:${proxy.upstream_port}`}</span>
      <span class="ml-auto text-[10px] uppercase tracking-wide bg-emerald-500/10 text-emerald-500 rounded px-1.5 py-0.5">SPA</span>
    </div>
  </section>
{/if}
```
Ajustar nomes de classe/estrutura ao que o arquivo já usa (ler primeiro). Sem polling de status (decisão do grill).

- [ ] **Step 4: build/check** (de `internal/ui/web`): `npx tsc --noEmit`, `npm run build`.

- [ ] **Step 5: commit**
```bash
git add internal/ui/web/src/tabs/proxies/ProxyDetailPanel.svelte
git commit -m "feat(ui): painel de proxy mostra badge fullstack e roteamento (estático)"
```

---

## Task 6: SiteDetail — bloco "Proxy fullstack" (todos os sites)

**Files:** modify `internal/ui/web/src/tabs/sites/SiteDetail.svelte`

- [ ] **Step 1: ler o arquivo atual** para achar onde o link e a versão do PHP são mostrados (próximo deles entra o bloco). Confirmar como o componente acessa o objeto `site` e se já importa stores.

- [ ] **Step 2: derivar o proxy fullstack vinculado a este site**

No script, importar a store de proxies e os modais:
```ts
  import { proxies } from '$stores/proxies';
  import { openProxyAddModal, openProxyEditModal } from '$stores/modals';
  import { suggestUnifiedDomain } from '$lib/fullstack';
```
Derivar (o nome canônico do site é `site.name ?? site.domain` — confirmar no store de sites):
```ts
  const siteName = $derived(site.name ?? site.domain);
  const boundProxy = $derived(
    $proxies.find((p) => p.site === siteName || (p.routes ?? []).some((r) => r.site === siteName))
  );
```

- [ ] **Step 3: markup — bloco perto do link/versão PHP**

Adicionar uma seção (mesmo padrão visual das outras do SiteDetail):
```svelte
<section class="...mesmo padrão...">
  <h2 class="text-xs font-semibold uppercase tracking-wider text-gray-500">Proxy fullstack</h2>
  {#if boundProxy}
    <div class="flex items-center justify-between gap-2">
      <span class="text-xs">
        API em
        <a href={`${boundProxy.secured ? 'https' : 'http'}://${boundProxy.domain}`} target="_blank" class="font-mono text-lerd-red hover:underline">↗ {boundProxy.domain}</a>
        <span class="text-gray-400">· {(boundProxy.routes ?? []).map((r) => r.path).join(' ')}</span>
      </span>
      <DetailButton onclick={() => openProxyEditModal(boundProxy)}>Editar</DetailButton>
    </div>
  {:else}
    <div class="flex items-center justify-between gap-2">
      <span class="text-xs text-gray-500">Servir este site como API sob um domínio único com seu SPA.</span>
      <DetailButton tone="primary" onclick={() => openProxyAddModal({ fullstack: true, apiSite: siteName, domain: suggestUnifiedDomain(siteName) })}>+ Criar proxy fullstack</DetailButton>
    </div>
  {/if}
</section>
```
Usar o `DetailButton` (já usado no projeto) e ajustar classes ao padrão do arquivo. O bloco aparece em **todos** os sites (decisão do grill).

- [ ] **Step 4: build/check** (de `internal/ui/web`): `npx tsc --noEmit`, `npm run build`.

- [ ] **Step 5: commit**
```bash
git add internal/ui/web/src/tabs/sites/SiteDetail.svelte
git commit -m "feat(ui): bloco 'Proxy fullstack' no SiteDetail (criar/editar a partir do site)"
```

---

## Task 7: rebuild do bundle embutido + verificação final

**Files:** assets embutidos (ver Makefile/embed)

- [ ] **Step 1:** descobrir como o frontend é embutido no binário Go (procurar `//go:embed` em `internal/ui/*.go` apontando para `web/dist` ou similar, e o alvo no `Makefile`). Reportar o comando de build do front (provavelmente `npm run build` em `internal/ui/web` gerando `dist/` embutido).

- [ ] **Step 2:** rodar o build do front e depois `go build ./...` para garantir que o binário embute o bundle novo sem erro.

- [ ] **Step 3:** `npx vitest run` (de `internal/ui/web`) verde; `go test ./internal/ui/` verde.

- [ ] **Step 4: commit** (se o bundle embutido for versionado; se `dist/` estiver em `.gitignore`, pular)
```bash
git add -A internal/ui/web
git commit -m "chore(ui): rebuild do bundle do dashboard com o modo fullstack"
```

---

## Self-Review (resultado)

- **Cobertura (spec §8 + mockups):** modal fullstack layout A (toggle, pickers site/porta, paths, mapa ao vivo, detecção -api) → Task 4; painel com badge + roteamento estático → Task 5; bloco "Proxy fullstack" em todos os sites (estados criar/editar) → Task 6; tipos/contrato → Tasks 1-3; helper puro testado → Task 2.
- **Placeholders:** as NOTES das Tasks 4-6 (ler o componente antes de inserir; confirmar o campo de nome canônico do site no `sites` store) são instruções concretas — o identificador de site (`name` vs `domain`) DEVE ser confirmado contra `stores/sites.ts` no início de cada task de componente, pois o backend resolve por **nome de site**.
- **Consistência:** `Route {path, site?, upstream_port?, upstream_host?}` igual ao DTO Go; `buildApiRoutes` espelha a lógica da CLI (`internal/cli/proxy.go`); prefill usa `openProxyAddModal({domain, fullstack, apiSite})`.
- **Decisões do grill respeitadas:** sem polling de status (rotas estáticas); bloco em todos os sites; domínio sugerido tira `-api`.
- **Risco conhecido:** o identificador de site no picker/binding precisa casar com o que o backend espera (nome do site). Confirmar cedo evita rotas que não resolvem (`resolveProxySpec` → site não encontrado → erro na criação). Tasks de componente começam lendo o arquivo + o `sites` store por isso.
```
