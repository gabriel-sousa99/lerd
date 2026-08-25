<script lang="ts">
  import { onMount } from 'svelte';
  import type { Proxy, ProxyGeneratedConfig, ProxyRuntimeStatus, ProxyTrafficStats } from '$stores/proxies';
  import { deleteProxy, loadProxyConfig, loadProxyRuntime, loadProxyStats, proxyAction, startProxyMonitoring } from '$stores/proxies';
  import { accessMode } from '$stores/accessMode';
  import { goToTab } from '$stores/route';
  import { openProxyEditModal } from '$stores/modals';
  import DetailHeader from '$components/DetailHeader.svelte';
  import DetailTabs, { type TabItem } from '$components/DetailTabs.svelte';
  import ButtonMenu, { type ButtonMenuAction } from '$components/ButtonMenu.svelte';
  import StatusPill, { type PillTone } from '$components/StatusPill.svelte';
  import DetailButton from '$components/DetailButton.svelte';
  import InfoRow from '$components/InfoRow.svelte';
  import LogViewer from '$components/LogViewer.svelte';
  import EmptyState from '$components/EmptyState.svelte';
  import ConfirmModal from '$components/ConfirmModal.svelte';
  import { m } from '../../paraglide/messages.js';

  interface Props { proxy: Proxy; }
  let { proxy }: Props = $props();

  type TabId = 'overview' | 'traffic' | 'logs' | 'config';
  let active = $state<TabId>('overview');
  let status = $state<ProxyRuntimeStatus | null>(null);
  let traffic = $state<ProxyTrafficStats | null>(null);
  let generated = $state<ProxyGeneratedConfig | null>(null);
  let busy = $state(false);
  let error = $state<string | null>(null);
  let confirmRemove = $state(false);

  const url = $derived(`${proxy.secured ? 'https' : 'http'}://${proxy.domain}`);
  const tabs = $derived<TabItem<TabId>[]>([
    { id: 'overview', label: m.proxies_tab_overview() },
    { id: 'traffic', label: m.proxies_tab_traffic(), count: traffic?.samples },
    { id: 'logs', label: m.proxies_tab_logs(), hidden: !proxy.managed },
    { id: 'config', label: m.proxies_tab_config() }
  ]);

  function stateLabel(state?: ProxyRuntimeStatus['state']): string {
    if (state === 'healthy') return m.proxies_status_healthy();
    if (state === 'degraded') return m.proxies_status_degraded();
    if (state === 'unreachable') return m.proxies_status_unreachable();
    if (state === 'failed') return m.proxies_status_failed();
    if (state === 'inactive') return m.proxies_status_inactive();
    if (state === 'paused') return m.proxies_status_paused();
    if (state === 'misconfigured') return m.proxies_status_misconfigured();
    return m.proxies_status_checking();
  }

  function stateTone(state?: ProxyRuntimeStatus['state']): PillTone {
    if (state === 'healthy') return 'ok';
    if (state === 'degraded' || state === 'misconfigured') return 'warn';
    if (state === 'unreachable' || state === 'failed') return 'error';
    return 'muted';
  }

  async function refreshOperational() {
    const name = proxy.name;
    const [nextStatus, nextTraffic] = await Promise.allSettled([loadProxyRuntime(name), loadProxyStats(name)]);
    if (name !== proxy.name) return;
    if (nextStatus.status === 'fulfilled') status = nextStatus.value;
    if (nextTraffic.status === 'fulfilled') traffic = nextTraffic.value;
  }

  async function refreshConfig() {
    try { generated = await loadProxyConfig(proxy.name); }
    catch { generated = null; }
  }

  onMount(() => {
    void refreshConfig();
    return startProxyMonitoring(refreshOperational);
  });

  $effect(() => {
    `${proxy.secured}:${proxy.upstream_host}:${proxy.upstream_port}:${proxy.upstream_scheme}:${proxy.timeout_seconds}:${proxy.domains.join(',')}`;
    void refreshConfig();
  });

  async function run(action: 'secure' | 'unsecure' | 'pause' | 'resume' | 'start' | 'stop') {
    busy = true;
    error = null;
    try {
      await proxyAction(proxy.name, action);
      await refreshOperational();
      await refreshConfig();
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally { busy = false; }
  }

  async function remove() {
    busy = true;
    error = null;
    try {
      await deleteProxy(proxy.name);
      confirmRemove = false;
      goToTab('proxies');
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally { busy = false; }
  }

  const actions = $derived.by<ButtonMenuAction[]>(() => {
    const items: ButtonMenuAction[] = [
      { id: 'open', label: m.proxies_open(), href: url, target: '_blank', tone: 'primary' }
    ];
    if (!$accessMode.localControl) return items;
    if (proxy.managed) {
      items.push(
        { id: 'start', label: m.common_start(), onclick: () => run('start') },
        { id: 'stop', label: m.common_stop(), onclick: () => run('stop') }
      );
    }
    items.push(
      { id: 'secure', label: proxy.secured ? m.proxies_unsecure() : m.proxies_secure(), onclick: () => run(proxy.secured ? 'unsecure' : 'secure') },
      { id: 'pause', label: proxy.paused ? m.proxies_resume() : m.proxies_pause(), onclick: () => run(proxy.paused ? 'resume' : 'pause') },
      { id: 'remove', label: m.common_remove(), tone: 'danger', onclick: () => (confirmRemove = true) }
    );
    return items;
  });

  function yesNo(value: boolean): string { return value ? m.common_enabled() : m.common_disabled(); }
  function present(value: boolean): string { return value ? m.proxies_present() : m.proxies_missing(); }
  function fmtMs(value: number): string {
    return value >= 1000 ? `${(value / 1000).toFixed(1)} s` : `${Math.round(value)} ms`;
  }
  function highlight(line: string): string | null {
    if (/error|fatal|failed/i.test(line)) return 'text-red-500';
    if (/warn/i.test(line)) return 'text-yellow-600 dark:text-yellow-400';
    return null;
  }
</script>

{#snippet headerTrailing()}
  <div class="flex items-center gap-2">
    <StatusPill tone={stateTone(status?.state)} label={stateLabel(status?.state)} />
    <ButtonMenu {actions} {busy} onSettings={$accessMode.localControl ? () => openProxyEditModal(proxy) : undefined} settingsTitle={m.common_settings()} />
  </div>
{/snippet}

<DetailHeader title={proxy.domain} trailing={headerTrailing} />
<DetailTabs {tabs} {active} onchange={(id) => (active = id)} />

{#if error}
  <div class="mx-3 mt-3 rounded-lg border border-red-200 dark:border-red-500/30 bg-red-50 dark:bg-red-500/10 px-3 py-2 text-xs text-red-600 dark:text-red-400">{error}</div>
{/if}

{#if active === 'overview'}
  <div class="flex-1 overflow-y-auto p-3 sm:p-5 space-y-5">
    <section class="space-y-2">
      <h2 class="text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">{m.proxies_upstream()}</h2>
      <InfoRow label={m.proxies_protocol()} value={proxy.upstream_scheme.toUpperCase()} />
      <InfoRow label={m.proxies_host()} value={proxy.upstream_host} />
      <InfoRow label={m.proxies_port()} value={String(proxy.upstream_port)} />
      {#if proxy.health_path}<InfoRow label={m.proxies_healthPath()} value={proxy.health_path} />{/if}
      <InfoRow label={m.proxies_timeout()} value={`${proxy.timeout_seconds} s`} />
      {#if proxy.path}<InfoRow label={m.proxies_projectPath()} value={proxy.path} />{/if}
      {#if proxy.domains.length > 1}<InfoRow label={m.proxies_domains()} value={proxy.domains.join(', ')} />{/if}
    </section>

    {#if proxy.fullstack}
      <section class="space-y-2 border-t border-gray-100 dark:border-lerd-border pt-4">
        <h2 class="text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">{m.proxies_routing()}</h2>
        <div class="rounded-lg border border-gray-200 dark:border-lerd-border overflow-hidden">
          {#each proxy.routes ?? [] as route (route.path)}
            <div class="flex items-center gap-3 px-3 py-2 border-b last:border-b-0 border-gray-100 dark:border-lerd-border text-xs font-mono">
              <span class="text-sky-600 dark:text-sky-400 min-w-24">{route.path}</span><span class="text-gray-300 dark:text-gray-600">→</span>
              <span class="truncate text-gray-600 dark:text-gray-300">{route.site ? `site ${route.site}` : `${route.upstream_host ?? proxy.upstream_host}:${route.upstream_port}`}</span>
            </div>
          {/each}
          <div class="flex items-center gap-3 px-3 py-2 text-xs font-mono">
            <span class="text-emerald-600 dark:text-emerald-400 min-w-24">/</span><span class="text-gray-300 dark:text-gray-600">→</span>
            <span class="truncate text-gray-600 dark:text-gray-300">{proxy.site ? `site ${proxy.site}` : `${proxy.upstream_host}:${proxy.upstream_port}`}</span>
          </div>
        </div>
      </section>
    {/if}

    {#if proxy.managed}
      <section class="space-y-2 border-t border-gray-100 dark:border-lerd-border pt-4">
        <h2 class="text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">{m.proxies_runtime()}</h2>
        {#if proxy.cmd}<InfoRow label={m.proxies_command()} value={proxy.cmd} />{/if}
        <InfoRow label={m.proxies_node()} value={proxy.node_version || '20'} />
        <InfoRow label={m.proxies_autostart()} value={yesNo(proxy.autostart)} mono={false} />
        {#if status?.unit_state}<InfoRow label="systemd" value={status.unit_state} />{/if}
      </section>
    {/if}
  </div>
{:else if active === 'traffic'}
  <div class="flex-1 overflow-y-auto p-3 sm:p-5 space-y-5">
    {#if traffic && traffic.samples > 0}
      <div class="grid grid-cols-2 gap-3">
        <div class="rounded-xl border border-gray-200 dark:border-lerd-border p-4"><div class="text-[10px] uppercase tracking-wider text-gray-400">{m.proxies_requests()}</div><div class="mt-1 text-2xl font-semibold tabular-nums text-gray-900 dark:text-white">{traffic.samples}</div></div>
        <div class="rounded-xl border border-gray-200 dark:border-lerd-border p-4"><div class="text-[10px] uppercase tracking-wider text-gray-400">{m.proxies_typicalLatency()}</div><div class="mt-1 text-2xl font-semibold tabular-nums text-gray-900 dark:text-white">{fmtMs(traffic.median_millis)}</div></div>
      </div>
      <section class="space-y-2">
        <h2 class="text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">{m.proxies_slowRoutes()}</h2>
        {#if traffic.slow.length === 0}
          <p class="text-sm text-gray-400 dark:text-gray-500">{m.proxies_noSlowRoutes()}</p>
        {:else}
          <div class="rounded-lg border border-gray-200 dark:border-lerd-border overflow-hidden">
            {#each traffic.slow as route (route.route)}
              <div class="flex items-center gap-3 px-3 py-2.5 border-b last:border-b-0 border-gray-100 dark:border-lerd-border"><code class="flex-1 truncate text-xs text-gray-700 dark:text-gray-300">{route.route}</code><span class="text-xs tabular-nums text-amber-600 dark:text-amber-400">p95 {fmtMs(route.recent_p95_millis ?? route.p95_millis)}</span><span class="text-[10px] tabular-nums text-gray-400">{route.samples}×</span></div>
            {/each}
          </div>
        {/if}
      </section>
    {:else}
      <EmptyState title={m.proxies_noTraffic()} size="sm" />
    {/if}
  </div>
{:else if active === 'logs' && proxy.managed}
  <LogViewer path={`/api/logs/lerd-proxy-${encodeURIComponent(proxy.name)}`} {highlight} />
{:else if active === 'config'}
  <div class="flex-1 overflow-y-auto p-3 sm:p-5 space-y-5">
    <section class="space-y-2">
      <div class="flex items-center justify-between gap-3"><h2 class="text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">{m.proxies_diagnostics()}</h2><DetailButton onclick={refreshOperational}>{m.common_refresh()}</DetailButton></div>
      <InfoRow label={m.proxies_nginx()} value={status?.nginx_running ? m.common_running() : m.common_stopped()} mono={false} />
      <InfoRow label={m.proxies_vhost()} value={present(status?.vhost_present ?? false)} mono={false} />
      {#if proxy.secured}<InfoRow label={m.proxies_certificate()} value={present(status?.certificate_present ?? false)} mono={false} />{/if}
      <InfoRow label={m.proxies_upstreamReachable()} value={status?.upstream_reachable ? m.proxies_reachable() : m.proxies_status_unreachable()} mono={false} />
      {#if status}<InfoRow label={m.proxies_latency()} value={fmtMs(status.latency_ms)} />{/if}
      {#if status?.http_status}<InfoRow label={m.proxies_httpStatus()} value={String(status.http_status)} />{/if}
      {#if status?.checked_at}<InfoRow label={m.proxies_lastCheck()} value={new Date(status.checked_at).toLocaleTimeString()} mono={false} />{/if}
      {#if status?.error}<p class="rounded-lg bg-red-50 dark:bg-red-500/10 px-3 py-2 text-xs font-mono text-red-600 dark:text-red-400 break-all">{status.error}</p>{/if}
    </section>
    <section class="space-y-2 border-t border-gray-100 dark:border-lerd-border pt-4">
      <div><h2 class="text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">{m.proxies_generatedConfig()}</h2><p class="mt-1 text-xs text-gray-400 dark:text-gray-500">{m.proxies_generatedConfigHint()}</p></div>
      {#if generated}
        <div class="rounded-lg border border-gray-200 dark:border-lerd-border overflow-hidden"><div class="px-3 py-2 border-b border-gray-200 dark:border-lerd-border bg-gray-50 dark:bg-white/3 text-[10px] font-mono text-gray-400 truncate">{generated.path}</div><pre class="max-h-96 overflow-auto p-3 bg-gray-950 text-gray-300 text-[11px] leading-relaxed"><code>{generated.content}</code></pre></div>
      {:else}
        <p class="text-sm text-gray-400 dark:text-gray-500">{m.proxies_missing()}</p>
      {/if}
    </section>
  </div>
{/if}

<ConfirmModal open={confirmRemove} title={m.proxies_removeTitle()} body={m.proxies_removeBody({ domain: proxy.domain })} confirmLabel={m.proxies_removeConfirm()} danger loading={busy} onconfirm={remove} onclose={() => (confirmRemove = false)} />
