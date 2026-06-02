<script lang="ts">
  import type { Proxy } from '$stores/proxies';
  import { deleteProxy, proxyAction } from '$stores/proxies';
  import { goToTab } from '$stores/route';
  import { openProxyEditModal } from '$stores/modals';
  import DetailButton from '$components/DetailButton.svelte';
  import InfoRow from '$components/InfoRow.svelte';

  interface Props {
    proxy: Proxy;
  }
  let { proxy }: Props = $props();

  let busy = $state<string | null>(null);
  let error = $state<string | null>(null);

  const url = $derived(`${proxy.secured ? 'https' : 'http'}://${proxy.domain}`);

  async function run(action: 'secure' | 'unsecure' | 'pause' | 'resume' | 'start' | 'stop') {
    busy = action;
    error = null;
    try {
      await proxyAction(proxy.name, action);
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally {
      busy = null;
    }
  }

  async function remove() {
    if (!confirm(`Remover proxy ${proxy.domain}?`)) return;
    busy = 'delete';
    error = null;
    try {
      await deleteProxy(proxy.name);
      goToTab('proxies');
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally {
      busy = null;
    }
  }
</script>

<div class="flex-1 overflow-y-auto">
  <header class="px-6 py-4 border-b border-gray-100 dark:border-lerd-border flex items-start justify-between gap-4">
    <div class="min-w-0">
      <h1 class="text-lg font-semibold text-gray-900 dark:text-white truncate">
        {proxy.domain}
        {#if proxy.fullstack}
          <span class="ml-2 align-middle text-[10px] uppercase tracking-wide bg-lerd-red/15 text-lerd-red border border-lerd-red/30 rounded px-1.5 py-0.5">fullstack</span>
        {/if}
      </h1>
      <a
        href={url}
        target="_blank"
        rel="noopener noreferrer"
        class="text-xs font-mono text-lerd-red hover:underline truncate block"
      >
        {url}
      </a>
    </div>
    <div class="flex items-center gap-2 shrink-0">
      <DetailButton onclick={() => openProxyEditModal(proxy)} disabled={busy !== null}>
        Edit
      </DetailButton>
      <DetailButton
        onclick={() => run(proxy.secured ? 'unsecure' : 'secure')}
        disabled={busy !== null}
        loading={busy === 'secure' || busy === 'unsecure'}
      >
        {proxy.secured ? 'Unsecure' : 'Secure'}
      </DetailButton>
      <DetailButton
        onclick={() => run(proxy.paused ? 'resume' : 'pause')}
        disabled={busy !== null}
        loading={busy === 'pause' || busy === 'resume'}
      >
        {proxy.paused ? 'Resume' : 'Pause'}
      </DetailButton>
      <DetailButton
        tone="danger"
        onclick={remove}
        disabled={busy !== null}
        loading={busy === 'delete'}
      >
        Delete
      </DetailButton>
    </div>
  </header>

  {#if error}
    <div class="px-6 py-2 text-xs text-red-500">{error}</div>
  {/if}

  <section class="px-6 py-4 space-y-2 border-b border-gray-100 dark:border-lerd-border">
    <h2 class="text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">
      Upstream
    </h2>
    <InfoRow label="Host" value={proxy.upstream_host} />
    <InfoRow label="Porta" value={String(proxy.upstream_port)} />
    {#if proxy.path}
      <InfoRow label="Pasta" value={proxy.path} mono />
    {/if}
    {#if proxy.domains && proxy.domains.length > 1}
      <InfoRow label="Domínios" value={proxy.domains.join(', ')} />
    {/if}
  </section>

  {#if proxy.fullstack}
    <section class="px-6 py-4 space-y-2 border-b border-gray-100 dark:border-lerd-border">
      <h2 class="text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">
        Roteamento (origem única)
      </h2>
      {#each (proxy.routes ?? []) as r (r.path)}
        <div class="flex items-center gap-2 text-xs font-mono mt-1.5">
          <span class="text-sky-500 min-w-[110px]">{r.path}</span>
          <span class="text-gray-400">→</span>
          <span class="text-gray-600 dark:text-gray-300">{r.site ? `site ${r.site}` : `:${r.upstream_port}`}</span>
          <span class="ml-auto text-[10px] uppercase tracking-wide bg-rose-500/10 text-rose-500 rounded px-1.5 py-0.5">API</span>
        </div>
      {/each}
      <div class="flex items-center gap-2 text-xs font-mono mt-1.5">
        <span class="text-emerald-500 min-w-[110px]">/ (resto)</span>
        <span class="text-gray-400">→</span>
        <span class="text-gray-600 dark:text-gray-300">{proxy.site ? `site ${proxy.site}` : `:${proxy.upstream_port}`}</span>
        <span class="ml-auto text-[10px] uppercase tracking-wide bg-emerald-500/10 text-emerald-500 rounded px-1.5 py-0.5">SPA</span>
      </div>
    </section>
  {/if}

  {#if proxy.managed}
    <section class="px-6 py-4 space-y-2">
      <div class="flex items-center justify-between">
        <h2 class="text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">
          Managed dev server
        </h2>
        <div class="flex items-center gap-2">
          <DetailButton
            tone="success"
            onclick={() => run('start')}
            disabled={busy !== null}
            loading={busy === 'start'}
          >
            Start
          </DetailButton>
          <DetailButton
            onclick={() => run('stop')}
            disabled={busy !== null}
            loading={busy === 'stop'}
          >
            Stop
          </DetailButton>
        </div>
      </div>
      {#if proxy.cmd}
        <InfoRow label="Comando" value={proxy.cmd} mono />
      {/if}
      <InfoRow label="Node" value={proxy.node_version || '20 (default)'} />
      <InfoRow label="Autostart" value={proxy.autostart ? 'sim' : 'não'} />
    </section>
  {/if}
</div>
