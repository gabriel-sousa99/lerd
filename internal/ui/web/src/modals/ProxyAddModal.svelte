<script lang="ts">
  import Modal from '$components/Modal.svelte';
  import DetailButton from '$components/DetailButton.svelte';
  import { closeModal, modal } from '$stores/modals';
  import { createProxy, updateProxy, type Proxy } from '$stores/proxies';
  import { goToTab } from '$stores/route';
  import { buildApiRoutes, defaultApiPaths, suggestUnifiedDomain } from '$lib/fullstack';
  import { sites, loadSites } from '$stores/sites';

  // Edit mode: when the modal store carries `kind: 'proxyEdit'`, we
  // pre-fill from the existing proxy and submit a partial PUT instead
  // of a POST. Domain and the managed toggle are locked in edit mode —
  // changing those requires rm+add (see proxyops.Update).
  const editing = $derived($modal.kind === 'proxyEdit' && !!$modal.proxy);
  const existing = $derived(editing ? ($modal.proxy as Proxy) : undefined);

  let domain = $state('');
  let port = $state<number>(9000);
  let path = $state('');
  let upstreamHost = $state('');
  let managed = $state(false);
  let cmd = $state('npm run dev');
  let nodeVersion = $state('20');
  let autostart = $state(false);
  let noSecure = $state(false);
  let error = $state<string | null>(null);
  let saving = $state(false);

  // Fullstack (SPA + API) mode state.
  let fullstack = $state(false);
  let spaMode = $state<'port' | 'site'>('port');
  let spaSite = $state('');
  let apiMode = $state<'site' | 'port'>('site');
  let apiSite = $state('');
  let apiPort = $state<number>(8000);
  let apiPathsText = $state(defaultApiPaths().join(' '));
  let prefillApplied = $state(false);

  // Snapshot the proxy on open so we can detect which fields actually
  // changed and only send those in the PUT payload.
  let snapshotName = $state<string | null>(null);

  $effect(() => {
    void loadSites();
  });

  // The canonical identifier the backend resolves via config.FindSite is the
  // site NAME (e.g. "retencao-api"), not the domain ("retencao-api.localhost").
  // `name` may be absent on the frontend Site, so fall back to stripping
  // `.localhost` off the domain.
  function siteValue(s: { name?: string; domain: string }): string {
    return s.name ?? s.domain.replace(/\.localhost$/, '');
  }

  $effect(() => {
    const p = existing;
    if (p && snapshotName !== p.name) {
      snapshotName = p.name;
      domain = p.domain;
      port = p.upstream_port;
      path = p.path ?? '';
      upstreamHost = p.upstream_host && p.upstream_host !== 'host.containers.internal' ? p.upstream_host : '';
      managed = p.managed;
      cmd = p.cmd ?? 'npm run dev';
      nodeVersion = p.node_version ?? '20';
      autostart = p.autostart;
      noSecure = !p.secured;
      fullstack = (p.routes?.length ?? 0) > 0 || !!p.site;
      if (p.site) { spaMode = 'site'; spaSite = p.site; } else { spaMode = 'port'; }
      const apiRoutes = (p.routes ?? []).filter((r) => r.site || r.upstream_port);
      if (apiRoutes.length) {
        apiPathsText = apiRoutes.map((r) => r.path).join(' ');
        if (apiRoutes[0].site) { apiMode = 'site'; apiSite = apiRoutes[0].site; }
        else { apiMode = 'port'; apiPort = apiRoutes[0].upstream_port ?? 8000; }
      }
      error = null;
    }
  });

  // Apply prefill from the modal store (add mode only).
  $effect(() => {
    const pf = $modal.prefill;
    if (!editing && pf && !prefillApplied) {
      prefillApplied = true;
      if (pf.domain) domain = pf.domain;
      if (pf.fullstack) fullstack = true;
      if (pf.apiSite) { apiMode = 'site'; apiSite = pf.apiSite; }
    }
  });

  const apiPaths = $derived(apiPathsText.split(/\s+/).map((s) => s.trim()).filter(Boolean));
  const routePreview = $derived(
    fullstack ? buildApiRoutes({ mode: apiMode, site: apiSite, port: apiPort, paths: apiPaths }) : []
  );
  const apiHint = $derived(!editing && /-api(\.localhost)?$/.test(domain.trim()));

  const canSubmit = $derived.by(() => {
    if (!editing && domain.trim().length === 0) return false;
    if (fullstack) {
      const baseOk = spaMode === 'port' ? port > 0 && port <= 65535 : spaSite.trim() !== '';
      const apiOk = apiMode === 'site' ? apiSite.trim() !== '' : apiPort > 0 && apiPort <= 65535;
      return baseOk && apiOk;
    }
    return port > 0 && port <= 65535;
  });

  async function submit(e: SubmitEvent) {
    e.preventDefault();
    if (!canSubmit || saving) return;
    error = null;
    saving = true;
    try {
      if (editing && existing) {
        // Build a sparse payload — only the fields that actually changed.
        const patch: Record<string, unknown> = {};
        if (port !== existing.upstream_port) patch.port = port;
        const newPath = path.trim();
        if (newPath !== (existing.path ?? '')) patch.path = newPath;
        const existingHost = existing.upstream_host && existing.upstream_host !== 'host.containers.internal'
          ? existing.upstream_host
          : '';
        const newHost = upstreamHost.trim();
        if (newHost !== existingHost) patch.upstream_host = newHost;
        if (existing.managed) {
          if (cmd !== (existing.cmd ?? '')) patch.cmd = cmd;
          if (nodeVersion !== (existing.node_version ?? '')) patch.node_version = nodeVersion;
          if (autostart !== existing.autostart) patch.autostart = autostart;
        }
        if (fullstack) {
          patch.routes = buildApiRoutes({ mode: apiMode, site: apiSite, port: apiPort, paths: apiPaths });
          patch.site = spaMode === 'site' ? spaSite.trim() : '';
          if (spaMode === 'port' && port !== existing.upstream_port) patch.port = port;
        } else if ((existing.routes?.length ?? 0) > 0 || existing.site) {
          // converting fullstack → simple: clear routes/site
          patch.routes = [];
          patch.site = '';
        }
        await updateProxy(existing.name, patch);
        closeModal();
        goToTab('proxies', existing.name);
      } else {
        const routes = buildApiRoutes({ mode: apiMode, site: apiSite, port: apiPort, paths: apiPaths });
        const created = await createProxy({
          domain: domain.trim(),
          port: !fullstack || spaMode === 'port' ? port : 0,
          path: path.trim() || undefined,
          no_secure: noSecure,
          managed: !fullstack && managed,
          cmd: !fullstack && managed ? cmd : undefined,
          node_version: !fullstack && managed ? nodeVersion : undefined,
          autostart: !fullstack && managed ? autostart : false,
          site: fullstack && spaMode === 'site' ? spaSite.trim() : undefined,
          routes: fullstack ? routes : undefined
        });
        closeModal();
        if (created?.name) {
          goToTab('proxies', created.name);
        }
      }
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally {
      saving = false;
    }
  }
</script>

<Modal open title={editing ? 'Editar proxy' : 'Adicionar proxy'} onclose={closeModal} size="md">
  <form id="proxy-add-form" class="px-5 py-4 space-y-4" onsubmit={submit}>
    <label class="block space-y-1">
      <span class="text-xs font-medium text-gray-600 dark:text-gray-300">Domínio (sem esquema)</span>
      <input
        type="text"
        bind:value={domain}
        placeholder="gestao-clientes.localhost"
        required
        disabled={editing}
        class="w-full bg-white dark:bg-lerd-bg border border-gray-200 dark:border-lerd-border rounded-md px-3 py-1.5 text-sm font-mono focus:outline-none focus:border-lerd-red disabled:opacity-60 disabled:cursor-not-allowed"
      />
      {#if editing}
        <span class="text-[10px] text-gray-400">Para trocar o domínio, remova e crie de novo.</span>
      {/if}
    </label>

    <div class="space-y-1">
      <span class="text-xs font-medium text-gray-600 dark:text-gray-300">Modo</span>
      <div class="inline-flex rounded-md border border-gray-200 dark:border-lerd-border overflow-hidden">
        <button
          type="button"
          onclick={() => (fullstack = false)}
          class="px-3 py-1.5 text-xs font-medium transition-colors {!fullstack
            ? 'bg-lerd-red text-white'
            : 'bg-white dark:bg-lerd-bg text-gray-600 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-lerd-border/40'}"
        >
          Simples
        </button>
        <button
          type="button"
          onclick={() => (fullstack = true)}
          class="px-3 py-1.5 text-xs font-medium transition-colors border-l border-gray-200 dark:border-lerd-border {fullstack
            ? 'bg-lerd-red text-white'
            : 'bg-white dark:bg-lerd-bg text-gray-600 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-lerd-border/40'}"
        >
          Fullstack (SPA + API)
        </button>
      </div>
    </div>

    {#if apiHint}
      <button
        type="button"
        onclick={() => { domain = suggestUnifiedDomain(domain); fullstack = true; }}
        class="block w-full text-left text-[11px] text-lerd-red hover:underline"
      >
        Detectado <code class="font-mono">-api</code> — usar <code class="font-mono">{suggestUnifiedDomain(domain)}</code>
      </button>
    {/if}

    {#if !fullstack}
      <label class="block space-y-1">
        <span class="text-xs font-medium text-gray-600 dark:text-gray-300">Porta do dev server</span>
        <input
          type="number"
          bind:value={port}
          min="1"
          max="65535"
          required
          class="w-full bg-white dark:bg-lerd-bg border border-gray-200 dark:border-lerd-border rounded-md px-3 py-1.5 text-sm focus:outline-none focus:border-lerd-red"
        />
      </label>

      <label class="block space-y-1">
        <span class="text-xs font-medium text-gray-600 dark:text-gray-300">
          Pasta do projeto <span class="text-gray-400">(opcional, obrigatória se managed)</span>
        </span>
        <input
          type="text"
          bind:value={path}
          placeholder="/home/u/projetos/gestao-clientes-spa"
          class="w-full bg-white dark:bg-lerd-bg border border-gray-200 dark:border-lerd-border rounded-md px-3 py-1.5 text-sm font-mono focus:outline-none focus:border-lerd-red"
        />
      </label>

      {#if editing}
        <label class="block space-y-1">
          <span class="text-xs font-medium text-gray-600 dark:text-gray-300">
            Upstream host <span class="text-gray-400">(vazio = host.containers.internal)</span>
          </span>
          <input
            type="text"
            bind:value={upstreamHost}
            placeholder="host.containers.internal"
            class="w-full bg-white dark:bg-lerd-bg border border-gray-200 dark:border-lerd-border rounded-md px-3 py-1.5 text-sm font-mono focus:outline-none focus:border-lerd-red"
          />
        </label>
      {:else}
        <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
          <input type="checkbox" bind:checked={noSecure} class="accent-lerd-red" />
          <span>HTTP apenas (sem mkcert)</span>
        </label>

        <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
          <input type="checkbox" bind:checked={managed} class="accent-lerd-red" />
          <span>Managed (lerd inicia o dev server)</span>
        </label>
      {/if}

      {#if (editing && existing?.managed) || (!editing && managed)}
        <fieldset class="space-y-3 border border-gray-200 dark:border-lerd-border rounded-md p-3">
          <label class="block space-y-1">
            <span class="text-xs font-medium text-gray-600 dark:text-gray-300">Comando</span>
            <input
              type="text"
              bind:value={cmd}
              class="w-full bg-white dark:bg-lerd-bg border border-gray-200 dark:border-lerd-border rounded-md px-3 py-1.5 text-sm font-mono focus:outline-none focus:border-lerd-red"
            />
          </label>

          <label class="block space-y-1">
            <span class="text-xs font-medium text-gray-600 dark:text-gray-300">Node major version</span>
            <input
              type="text"
              bind:value={nodeVersion}
              class="w-full bg-white dark:bg-lerd-bg border border-gray-200 dark:border-lerd-border rounded-md px-3 py-1.5 text-sm focus:outline-none focus:border-lerd-red"
            />
          </label>

          <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
            <input type="checkbox" bind:checked={autostart} class="accent-lerd-red" />
            <span>Iniciar com <code class="font-mono text-xs">lerd start</code></span>
          </label>
        </fieldset>
      {/if}
    {:else}
      <!-- Fullstack mode: SPA base + API routes -->
      <fieldset class="space-y-3 border border-gray-200 dark:border-lerd-border rounded-md p-3">
        <legend class="px-1 text-xs font-medium text-gray-600 dark:text-gray-300">SPA (base /)</legend>

        <div class="inline-flex rounded-md border border-gray-200 dark:border-lerd-border overflow-hidden">
          <button
            type="button"
            onclick={() => (spaMode = 'port')}
            class="px-3 py-1 text-xs font-medium transition-colors {spaMode === 'port'
              ? 'bg-lerd-red text-white'
              : 'bg-white dark:bg-lerd-bg text-gray-600 dark:text-gray-300'}"
          >
            Porta
          </button>
          <button
            type="button"
            onclick={() => (spaMode = 'site')}
            class="px-3 py-1 text-xs font-medium transition-colors border-l border-gray-200 dark:border-lerd-border {spaMode === 'site'
              ? 'bg-lerd-red text-white'
              : 'bg-white dark:bg-lerd-bg text-gray-600 dark:text-gray-300'}"
          >
            Site
          </button>
        </div>

        {#if spaMode === 'port'}
          <label class="block space-y-1">
            <span class="text-xs font-medium text-gray-600 dark:text-gray-300">Porta do dev server</span>
            <input
              type="number"
              bind:value={port}
              min="1"
              max="65535"
              class="w-full bg-white dark:bg-lerd-bg border border-gray-200 dark:border-lerd-border rounded-md px-3 py-1.5 text-sm focus:outline-none focus:border-lerd-red"
            />
          </label>
        {:else}
          <label class="block space-y-1">
            <span class="text-xs font-medium text-gray-600 dark:text-gray-300">Site</span>
            <select
              bind:value={spaSite}
              class="w-full bg-white dark:bg-lerd-bg border border-gray-200 dark:border-lerd-border rounded-md px-3 py-1.5 text-sm font-mono focus:outline-none focus:border-lerd-red"
            >
              <option value="">— escolha um site —</option>
              {#each $sites as s (s.domain)}
                <option value={siteValue(s)}>{s.domain}</option>
              {/each}
            </select>
          </label>
        {/if}
      </fieldset>

      <fieldset class="space-y-3 border border-gray-200 dark:border-lerd-border rounded-md p-3">
        <legend class="px-1 text-xs font-medium text-gray-600 dark:text-gray-300">API</legend>

        <div class="inline-flex rounded-md border border-gray-200 dark:border-lerd-border overflow-hidden">
          <button
            type="button"
            onclick={() => (apiMode = 'site')}
            class="px-3 py-1 text-xs font-medium transition-colors {apiMode === 'site'
              ? 'bg-lerd-red text-white'
              : 'bg-white dark:bg-lerd-bg text-gray-600 dark:text-gray-300'}"
          >
            Site
          </button>
          <button
            type="button"
            onclick={() => (apiMode = 'port')}
            class="px-3 py-1 text-xs font-medium transition-colors border-l border-gray-200 dark:border-lerd-border {apiMode === 'port'
              ? 'bg-lerd-red text-white'
              : 'bg-white dark:bg-lerd-bg text-gray-600 dark:text-gray-300'}"
          >
            Porta
          </button>
        </div>

        {#if apiMode === 'site'}
          <label class="block space-y-1">
            <span class="text-xs font-medium text-gray-600 dark:text-gray-300">Site da API</span>
            <select
              bind:value={apiSite}
              class="w-full bg-white dark:bg-lerd-bg border border-gray-200 dark:border-lerd-border rounded-md px-3 py-1.5 text-sm font-mono focus:outline-none focus:border-lerd-red"
            >
              <option value="">— escolha um site —</option>
              {#each $sites as s (s.domain)}
                <option value={siteValue(s)}>{s.domain}</option>
              {/each}
            </select>
          </label>
        {:else}
          <label class="block space-y-1">
            <span class="text-xs font-medium text-gray-600 dark:text-gray-300">Porta da API</span>
            <input
              type="number"
              bind:value={apiPort}
              min="1"
              max="65535"
              class="w-full bg-white dark:bg-lerd-bg border border-gray-200 dark:border-lerd-border rounded-md px-3 py-1.5 text-sm focus:outline-none focus:border-lerd-red"
            />
          </label>
        {/if}

        <label class="block space-y-1">
          <span class="text-xs font-medium text-gray-600 dark:text-gray-300">Paths da API (separados por espaço)</span>
          <input
            type="text"
            bind:value={apiPathsText}
            placeholder="/api /sanctum /broadcasting /storage"
            class="w-full bg-white dark:bg-lerd-bg border border-gray-200 dark:border-lerd-border rounded-md px-3 py-1.5 text-sm font-mono focus:outline-none focus:border-lerd-red"
          />
        </label>
      </fieldset>

      <label class="block space-y-1">
        <span class="text-xs font-medium text-gray-600 dark:text-gray-300">
          Pasta do projeto <span class="text-gray-400">(opcional)</span>
        </span>
        <input
          type="text"
          bind:value={path}
          placeholder="/home/u/projetos/gestao-clientes-spa"
          class="w-full bg-white dark:bg-lerd-bg border border-gray-200 dark:border-lerd-border rounded-md px-3 py-1.5 text-sm font-mono focus:outline-none focus:border-lerd-red"
        />
      </label>

      <div class="space-y-1">
        <span class="text-xs font-medium text-gray-600 dark:text-gray-300">Mapa de origem</span>
        <ul class="rounded-md border border-gray-200 dark:border-lerd-border p-2 text-xs font-mono space-y-0.5">
          {#each routePreview as r}
            <li>{r.path} → {r.site ? 'site ' + r.site : ':' + r.upstream_port}</li>
          {/each}
          <li>/ (resto) → {spaMode === 'site' ? 'site ' + spaSite : ':' + port}</li>
        </ul>
      </div>
    {/if}

    {#if error}
      <p class="text-xs text-red-500">{error}</p>
    {/if}
  </form>

  {#snippet footer()}
    <DetailButton onclick={closeModal} disabled={saving}>Cancelar</DetailButton>
    <DetailButton
      tone="primary"
      onclick={() => {
        const form = document.getElementById('proxy-add-form') as HTMLFormElement | null;
        form?.requestSubmit();
      }}
      disabled={!canSubmit || saving}
      loading={saving}
    >
      {editing ? 'Salvar' : 'Criar'}
    </DetailButton>
  {/snippet}
</Modal>
