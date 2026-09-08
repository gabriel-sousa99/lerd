<script lang="ts">
  import Modal from '$components/Modal.svelte';
  import DetailButton from '$components/DetailButton.svelte';
  import { closeModal, modal } from '$stores/modals';
  import { createProxy, updateProxy, type Proxy } from '$stores/proxies';
  import { goToTab } from '$stores/route';
  import {
    buildApiRoutes,
    defaultApiPaths,
    editableRoutesFrom,
    routesFromEditable,
    suggestUnifiedDomain,
    type EditableRoute
  } from '$lib/fullstack';
  import { sites, loadSites } from '$stores/sites';
  import { m } from '../paraglide/messages.js';

  // Edit mode: when the modal store carries `kind: 'proxyEdit'`, we
  // pre-fill from the existing proxy and submit a partial PUT instead
  // of a POST. Domain and the managed toggle are locked in edit mode —
  // changing those requires rm+add (see proxyops.Update).
  const editing = $derived($modal.kind === 'proxyEdit' && !!$modal.proxy);
  const existing = $derived(editing ? ($modal.proxy as Proxy) : undefined);

  let domain = $state('');
  let aliasesText = $state('');
  let port = $state<number>(9000);
  let path = $state('');
  let upstreamHost = $state('');
  let upstreamScheme = $state<'http' | 'https'>('http');
  let healthPath = $state('');
  let timeoutSeconds = $state<number>(86400);
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
  let individualRoutes = $state(false);
  let editableRoutes = $state<EditableRoute[]>([]);
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
      aliasesText = p.domains.slice(1).join('\n');
      port = p.upstream_port;
      path = p.path ?? '';
      upstreamHost = p.upstream_host && p.upstream_host !== 'host.containers.internal' ? p.upstream_host : '';
      upstreamScheme = p.upstream_scheme ?? 'http';
      healthPath = p.health_path ?? '';
      timeoutSeconds = p.timeout_seconds || 86400;
      managed = p.managed;
      cmd = p.cmd ?? 'npm run dev';
      nodeVersion = p.node_version ?? '20';
      autostart = p.autostart;
      noSecure = !p.secured;
      fullstack = (p.routes?.length ?? 0) > 0 || !!p.site;
      if (p.site) { spaMode = 'site'; spaSite = p.site; } else { spaMode = 'port'; }
      const apiRoutes = (p.routes ?? []).filter((r) => r.site || r.upstream_port);
      if (apiRoutes.length) {
        individualRoutes = true;
        editableRoutes = editableRoutesFrom(apiRoutes);
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
  const aliases = $derived(aliasesText.split(/[\s,]+/).map((s) => s.trim()).filter(Boolean));
  const routePreview = $derived(
    fullstack
      ? individualRoutes
        ? routesFromEditable(editableRoutes)
        : buildApiRoutes({ mode: apiMode, site: apiSite, port: apiPort, paths: apiPaths })
      : []
  );
  const apiHint = $derived(!editing && /-api(\.localhost)?$/.test(domain.trim()));

  const canSubmit = $derived.by(() => {
    if (!editing && domain.trim().length === 0) return false;
    if (fullstack) {
      const baseOk = spaMode === 'port' ? port > 0 && port <= 65535 : spaSite.trim() !== '';
      const apiOk = individualRoutes
        ? editableRoutes.length > 0 && editableRoutes.every((route) =>
            route.path.trim().startsWith('/') && route.path.trim() !== '/' &&
            (route.mode === 'site' ? route.site.trim() !== '' : route.port > 0 && route.port <= 65535)
          )
        : apiMode === 'site' ? apiSite.trim() !== '' : apiPort > 0 && apiPort <= 65535;
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
        const existingAliases = existing.domains.slice(1);
        if (aliases.join('\n') !== existingAliases.join('\n')) patch.aliases = aliases;
        if (port !== existing.upstream_port) patch.port = port;
        const newPath = path.trim();
        if (newPath !== (existing.path ?? '')) patch.path = newPath;
        const existingHost = existing.upstream_host && existing.upstream_host !== 'host.containers.internal'
          ? existing.upstream_host
          : '';
        const newHost = upstreamHost.trim();
        if (newHost !== existingHost) patch.upstream_host = newHost;
        if (upstreamScheme !== existing.upstream_scheme) patch.upstream_scheme = upstreamScheme;
        if (healthPath.trim() !== (existing.health_path ?? '')) patch.health_path = healthPath.trim();
        if (timeoutSeconds !== existing.timeout_seconds) patch.timeout_seconds = timeoutSeconds;
        if (existing.managed) {
          if (cmd !== (existing.cmd ?? '')) patch.cmd = cmd;
          if (nodeVersion !== (existing.node_version ?? '')) patch.node_version = nodeVersion;
          if (autostart !== existing.autostart) patch.autostart = autostart;
        }
        if (fullstack) {
          patch.routes = routePreview;
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
        const routes = routePreview;
        const created = await createProxy({
          domain: domain.trim(),
          aliases,
          port: !fullstack || spaMode === 'port' ? port : 0,
          upstream_host: upstreamHost.trim() || undefined,
          upstream_scheme: upstreamScheme,
          health_path: healthPath.trim() || undefined,
          timeout_seconds: timeoutSeconds,
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

  function enableIndividualRoutes() {
    if (editableRoutes.length === 0) editableRoutes = editableRoutesFrom(routePreview);
    individualRoutes = true;
  }

  function addEditableRoute() {
    editableRoutes = [...editableRoutes, { path: '/api', mode: 'port', site: '', port: 8000, host: '' }];
  }

  function removeEditableRoute(index: number) {
    editableRoutes = editableRoutes.filter((_, i) => i !== index);
  }
</script>

<Modal open title={editing ? m.proxies_modal_editTitle() : m.proxies_modal_addTitle()} onclose={closeModal} size="lg">
  <form id="proxy-add-form" class="@container px-5 py-4 space-y-4 max-h-[60vh] overflow-y-auto" onsubmit={submit}>
    <label class="block space-y-1">
      <span class="text-xs font-medium text-gray-600 dark:text-gray-300">{m.proxies_modal_domain()}</span>
      <input
        type="text"
        bind:value={domain}
        placeholder="gestao-clientes.localhost"
        required
        disabled={editing}
        class="w-full bg-white dark:bg-lerd-bg border border-gray-200 dark:border-lerd-border rounded-md px-3 py-1.5 text-sm font-mono focus:outline-none focus:border-lerd-red disabled:opacity-60 disabled:cursor-not-allowed"
      />
      {#if editing}
        <span class="text-[10px] text-gray-400">{m.proxies_modal_domainLocked()}</span>
      {/if}
    </label>

    <label class="block space-y-1">
      <span class="text-xs font-medium text-gray-600 dark:text-gray-300">{m.proxies_modal_aliases()} <span class="text-gray-400">({m.proxies_modal_onePerLine()})</span></span>
      <textarea
        bind:value={aliasesText}
        rows="2"
        placeholder="admin.gestao-clientes.localhost"
        class="w-full resize-y bg-white dark:bg-lerd-bg border border-gray-200 dark:border-lerd-border rounded-md px-3 py-1.5 text-sm font-mono focus:outline-none focus:border-lerd-red"
      ></textarea>
    </label>

    <!-- @lg, not sm: the panel is a fixed max-w-lg, so a viewport breakpoint
         put two columns in 464px of form. The container query asks the form. -->
    <fieldset class="grid grid-cols-1 @lg:grid-cols-2 gap-3 border border-gray-200 dark:border-lerd-border rounded-md p-3">
      <legend class="px-1 text-xs font-medium text-gray-600 dark:text-gray-300">{m.proxies_upstream()}</legend>
      <label class="block space-y-1">
        <span class="text-xs font-medium text-gray-600 dark:text-gray-300">{m.proxies_protocol()}</span>
        <select bind:value={upstreamScheme} class="w-full bg-white dark:bg-lerd-bg border border-gray-200 dark:border-lerd-border rounded-md px-3 py-1.5 text-sm font-mono focus:outline-none focus:border-lerd-red">
          <option value="http">HTTP</option>
          <option value="https">HTTPS</option>
        </select>
      </label>
      <label class="block space-y-1">
        <span class="text-xs font-medium text-gray-600 dark:text-gray-300">{m.proxies_host()} <span class="text-gray-400">({m.proxies_modal_hostHint()})</span></span>
        <input type="text" bind:value={upstreamHost} placeholder="host.containers.internal" class="w-full bg-white dark:bg-lerd-bg border border-gray-200 dark:border-lerd-border rounded-md px-3 py-1.5 text-sm font-mono focus:outline-none focus:border-lerd-red" />
      </label>
      <label class="block space-y-1">
        <span class="text-xs font-medium text-gray-600 dark:text-gray-300">{m.proxies_healthPath()} <span class="text-gray-400">({m.proxies_modal_optional()})</span></span>
        <input type="text" bind:value={healthPath} placeholder="/health" class="w-full bg-white dark:bg-lerd-bg border border-gray-200 dark:border-lerd-border rounded-md px-3 py-1.5 text-sm font-mono focus:outline-none focus:border-lerd-red" />
      </label>
      <label class="block space-y-1">
        <span class="text-xs font-medium text-gray-600 dark:text-gray-300">{m.proxies_modal_timeoutSeconds()}</span>
        <input type="number" bind:value={timeoutSeconds} min="1" max="86400" class="w-full bg-white dark:bg-lerd-bg border border-gray-200 dark:border-lerd-border rounded-md px-3 py-1.5 text-sm focus:outline-none focus:border-lerd-red" />
      </label>
    </fieldset>

    <div class="space-y-1">
      <span class="text-xs font-medium text-gray-600 dark:text-gray-300">{m.proxies_modal_mode()}</span>
      <div class="inline-flex rounded-md border border-gray-200 dark:border-lerd-border overflow-hidden">
        <button
          type="button"
          onclick={() => (fullstack = false)}
          class="px-3 py-1.5 text-xs font-medium transition-colors {!fullstack
            ? 'bg-lerd-red text-white'
            : 'bg-white dark:bg-lerd-bg text-gray-600 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-lerd-border/40'}"
        >
          {m.proxies_modal_simple()}
        </button>
        <button
          type="button"
          onclick={() => (fullstack = true)}
          class="px-3 py-1.5 text-xs font-medium transition-colors border-l border-gray-200 dark:border-lerd-border {fullstack
            ? 'bg-lerd-red text-white'
            : 'bg-white dark:bg-lerd-bg text-gray-600 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-lerd-border/40'}"
        >
          {m.proxies_modal_fullstack()}
        </button>
      </div>
    </div>

    {#if apiHint}
      <button
        type="button"
        onclick={() => { domain = suggestUnifiedDomain(domain); fullstack = true; }}
        class="block w-full text-left text-[11px] text-lerd-red hover:underline"
      >
        {m.proxies_modal_apiDetected({ domain: suggestUnifiedDomain(domain) })}
      </button>
    {/if}

    {#if !fullstack}
      <label class="block space-y-1">
        <span class="text-xs font-medium text-gray-600 dark:text-gray-300">{m.proxies_modal_devPort()}</span>
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
          {m.proxies_projectPath()} <span class="text-gray-400">({m.proxies_modal_projectPathManaged()})</span>
        </span>
        <input
          type="text"
          bind:value={path}
          placeholder="/home/u/projetos/gestao-clientes-spa"
          class="w-full bg-white dark:bg-lerd-bg border border-gray-200 dark:border-lerd-border rounded-md px-3 py-1.5 text-sm font-mono focus:outline-none focus:border-lerd-red"
        />
      </label>

      {#if !editing}
        <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
          <input type="checkbox" bind:checked={noSecure} class="accent-lerd-red" />
          <span>{m.proxies_modal_httpOnly()}</span>
        </label>

        <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
          <input type="checkbox" bind:checked={managed} class="accent-lerd-red" />
          <span>{m.proxies_modal_managed()}</span>
        </label>
      {/if}

      {#if (editing && existing?.managed) || (!editing && managed)}
        <fieldset class="space-y-3 border border-gray-200 dark:border-lerd-border rounded-md p-3">
          <label class="block space-y-1">
            <span class="text-xs font-medium text-gray-600 dark:text-gray-300">{m.proxies_command()}</span>
            <input
              type="text"
              bind:value={cmd}
              class="w-full bg-white dark:bg-lerd-bg border border-gray-200 dark:border-lerd-border rounded-md px-3 py-1.5 text-sm font-mono focus:outline-none focus:border-lerd-red"
            />
          </label>

          <label class="block space-y-1">
            <span class="text-xs font-medium text-gray-600 dark:text-gray-300">{m.proxies_modal_nodeVersion()}</span>
            <input
              type="text"
              bind:value={nodeVersion}
              class="w-full bg-white dark:bg-lerd-bg border border-gray-200 dark:border-lerd-border rounded-md px-3 py-1.5 text-sm focus:outline-none focus:border-lerd-red"
            />
          </label>

          <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
            <input type="checkbox" bind:checked={autostart} class="accent-lerd-red" />
            <span>{m.proxies_modal_autostartHint()}</span>
          </label>
        </fieldset>
      {/if}
    {:else}
      <!-- Fullstack mode: SPA base + API routes -->
      <fieldset class="space-y-3 border border-gray-200 dark:border-lerd-border rounded-md p-3">
        <legend class="px-1 text-xs font-medium text-gray-600 dark:text-gray-300">{m.proxies_modal_base()}</legend>

        <div class="inline-flex rounded-md border border-gray-200 dark:border-lerd-border overflow-hidden">
          <button
            type="button"
            onclick={() => (spaMode = 'port')}
            class="px-3 py-1 text-xs font-medium transition-colors {spaMode === 'port'
              ? 'bg-lerd-red text-white'
              : 'bg-white dark:bg-lerd-bg text-gray-600 dark:text-gray-300'}"
          >
            {m.proxies_port()}
          </button>
          <button
            type="button"
            onclick={() => (spaMode = 'site')}
            class="px-3 py-1 text-xs font-medium transition-colors border-l border-gray-200 dark:border-lerd-border {spaMode === 'site'
              ? 'bg-lerd-red text-white'
              : 'bg-white dark:bg-lerd-bg text-gray-600 dark:text-gray-300'}"
          >
            {m.proxies_modal_site()}
          </button>
        </div>

        {#if spaMode === 'port'}
          <label class="block space-y-1">
            <span class="text-xs font-medium text-gray-600 dark:text-gray-300">{m.proxies_modal_devPort()}</span>
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
            <span class="text-xs font-medium text-gray-600 dark:text-gray-300">{m.proxies_modal_site()}</span>
            <select
              bind:value={spaSite}
              class="w-full bg-white dark:bg-lerd-bg border border-gray-200 dark:border-lerd-border rounded-md px-3 py-1.5 text-sm font-mono focus:outline-none focus:border-lerd-red"
            >
              <option value="">— {m.proxies_modal_chooseSite()} —</option>
              {#each $sites as s (s.domain)}
                <option value={siteValue(s)}>{s.domain}</option>
              {/each}
            </select>
          </label>
        {/if}
      </fieldset>

      <fieldset class="space-y-3 border border-gray-200 dark:border-lerd-border rounded-md p-3">
        <legend class="px-1 text-xs font-medium text-gray-600 dark:text-gray-300">API</legend>

        <div class="flex items-center justify-between gap-3">
          <span class="text-[11px] text-gray-400">{m.proxies_modal_commonTargetHint()}</span>
          <button type="button" onclick={() => individualRoutes ? (individualRoutes = false) : enableIndividualRoutes()} class="text-[11px] font-medium text-lerd-red hover:underline">
            {individualRoutes ? m.proxies_modal_useCommonTarget() : m.proxies_modal_editRoutes()}
          </button>
        </div>

        {#if individualRoutes}
          <div class="space-y-2">
            {#each editableRoutes as route, index (index)}
              <div class="rounded-md border border-gray-200 dark:border-lerd-border p-2.5 space-y-2">
                <div class="grid grid-cols-[1fr_auto_auto] gap-2">
                  <input type="text" bind:value={route.path} placeholder="/api" aria-label={m.proxies_modal_routePath()} class="min-w-0 bg-white dark:bg-lerd-bg border border-gray-200 dark:border-lerd-border rounded-md px-2 py-1.5 text-xs font-mono focus:outline-none focus:border-lerd-red" />
                  <select bind:value={route.mode} aria-label={m.proxies_modal_targetType()} class="bg-white dark:bg-lerd-bg border border-gray-200 dark:border-lerd-border rounded-md px-2 py-1.5 text-xs focus:outline-none focus:border-lerd-red">
                    <option value="site">{m.proxies_modal_site()}</option>
                    <option value="port">{m.proxies_modal_hostPort()}</option>
                  </select>
                  <button type="button" onclick={() => removeEditableRoute(index)} aria-label={m.proxies_modal_removeRoute()} class="px-2 text-gray-400 hover:text-red-500">×</button>
                </div>
                {#if route.mode === 'site'}
                  <select bind:value={route.site} aria-label={m.proxies_modal_routeSite()} class="w-full bg-white dark:bg-lerd-bg border border-gray-200 dark:border-lerd-border rounded-md px-2 py-1.5 text-xs font-mono focus:outline-none focus:border-lerd-red">
                    <option value="">— {m.proxies_modal_chooseSite()} —</option>
                    {#each $sites as s (s.domain)}<option value={siteValue(s)}>{s.domain}</option>{/each}
                  </select>
                {:else}
                  <div class="grid grid-cols-[1fr_7rem] gap-2">
                    <input type="text" bind:value={route.host} placeholder="host.containers.internal" aria-label={m.proxies_modal_routeHost()} class="min-w-0 bg-white dark:bg-lerd-bg border border-gray-200 dark:border-lerd-border rounded-md px-2 py-1.5 text-xs font-mono focus:outline-none focus:border-lerd-red" />
                    <input type="number" bind:value={route.port} min="1" max="65535" aria-label={m.proxies_modal_routePort()} class="bg-white dark:bg-lerd-bg border border-gray-200 dark:border-lerd-border rounded-md px-2 py-1.5 text-xs focus:outline-none focus:border-lerd-red" />
                  </div>
                {/if}
              </div>
            {/each}
            <button type="button" onclick={addEditableRoute} class="text-xs font-medium text-lerd-red hover:underline">+ {m.proxies_modal_addRoute()}</button>
          </div>
        {:else}

        <div class="inline-flex rounded-md border border-gray-200 dark:border-lerd-border overflow-hidden">
          <button
            type="button"
            onclick={() => (apiMode = 'site')}
            class="px-3 py-1 text-xs font-medium transition-colors {apiMode === 'site'
              ? 'bg-lerd-red text-white'
              : 'bg-white dark:bg-lerd-bg text-gray-600 dark:text-gray-300'}"
          >
            {m.proxies_modal_site()}
          </button>
          <button
            type="button"
            onclick={() => (apiMode = 'port')}
            class="px-3 py-1 text-xs font-medium transition-colors border-l border-gray-200 dark:border-lerd-border {apiMode === 'port'
              ? 'bg-lerd-red text-white'
              : 'bg-white dark:bg-lerd-bg text-gray-600 dark:text-gray-300'}"
          >
            {m.proxies_port()}
          </button>
        </div>

        {#if apiMode === 'site'}
          <label class="block space-y-1">
            <span class="text-xs font-medium text-gray-600 dark:text-gray-300">{m.proxies_modal_apiSite()}</span>
            <select
              bind:value={apiSite}
              class="w-full bg-white dark:bg-lerd-bg border border-gray-200 dark:border-lerd-border rounded-md px-3 py-1.5 text-sm font-mono focus:outline-none focus:border-lerd-red"
            >
              <option value="">— {m.proxies_modal_chooseSite()} —</option>
              {#each $sites as s (s.domain)}
                <option value={siteValue(s)}>{s.domain}</option>
              {/each}
            </select>
          </label>
        {:else}
          <label class="block space-y-1">
            <span class="text-xs font-medium text-gray-600 dark:text-gray-300">{m.proxies_modal_apiPort()}</span>
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
          <span class="text-xs font-medium text-gray-600 dark:text-gray-300">{m.proxies_modal_apiPaths()}</span>
          <input
            type="text"
            bind:value={apiPathsText}
            placeholder="/api /sanctum /broadcasting /storage"
            class="w-full bg-white dark:bg-lerd-bg border border-gray-200 dark:border-lerd-border rounded-md px-3 py-1.5 text-sm font-mono focus:outline-none focus:border-lerd-red"
          />
        </label>
        {/if}
      </fieldset>

      <label class="block space-y-1">
        <span class="text-xs font-medium text-gray-600 dark:text-gray-300">
          {m.proxies_projectPath()} <span class="text-gray-400">({m.proxies_modal_optional()})</span>
        </span>
        <input
          type="text"
          bind:value={path}
          placeholder="/home/u/projetos/gestao-clientes-spa"
          class="w-full bg-white dark:bg-lerd-bg border border-gray-200 dark:border-lerd-border rounded-md px-3 py-1.5 text-sm font-mono focus:outline-none focus:border-lerd-red"
        />
      </label>

      {#if !editing}
        <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
          <input type="checkbox" bind:checked={noSecure} class="accent-lerd-red" />
          <span>{m.proxies_modal_httpOnly()}</span>
        </label>
      {/if}

      <div class="space-y-1">
        <span class="text-xs font-medium text-gray-600 dark:text-gray-300">{m.proxies_modal_originMap()}</span>
        <ul class="rounded-md border border-gray-200 dark:border-lerd-border p-2 text-xs font-mono space-y-0.5">
          {#each routePreview as r}
            <li>{r.path} → {r.site ? 'site ' + r.site : (r.upstream_host ? r.upstream_host : upstreamHost || 'host.containers.internal') + ':' + r.upstream_port}</li>
          {/each}
          <li>/ ({m.proxies_modal_rest()}) → {spaMode === 'site' ? 'site ' + spaSite : ':' + port}</li>
        </ul>
      </div>
    {/if}

    {#if error}
      <p class="text-xs text-red-500">{error}</p>
    {/if}
  </form>

  {#snippet footer()}
    <DetailButton onclick={closeModal} disabled={saving}>{m.common_cancel()}</DetailButton>
    <DetailButton
      tone="primary"
      onclick={() => {
        const form = document.getElementById('proxy-add-form') as HTMLFormElement | null;
        form?.requestSubmit();
      }}
      disabled={!canSubmit || saving}
      loading={saving}
    >
      {editing ? m.common_save() : m.proxies_modal_create()}
    </DetailButton>
  {/snippet}
</Modal>
