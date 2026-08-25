<script lang="ts">
  import { onMount } from 'svelte';
  import ListPanel from '$components/ListPanel.svelte';
  import ListRow from '$components/ListRow.svelte';
  import StatusDot, { type StatusColor } from '$components/StatusDot.svelte';
  import ActionButton from '$components/ActionButton.svelte';
  import Icon from '$components/Icon.svelte';
  import LoadingRow from '$components/LoadingRow.svelte';
  import EmptyState from '$components/EmptyState.svelte';
  import {
    proxies,
    proxiesLoaded,
    loadProxies,
    loadProxyRuntime,
    startProxyMonitoring,
    type ProxyRuntimeStatus
  } from '$stores/proxies';
  import { accessMode } from '$stores/accessMode';
  import { goToTab, routeRest } from '$stores/route';
  import { openProxyAddModal } from '$stores/modals';
  import { m } from '../paraglide/messages.js';

  const selected = $derived($routeRest);
  let runtime = $state<Record<string, ProxyRuntimeStatus>>({});

  async function refreshRuntime() {
    const entries = await Promise.all(
      $proxies.map(async (proxy) => {
        try {
          return [proxy.name, await loadProxyRuntime(proxy.name)] as const;
        } catch {
          return [proxy.name, undefined] as const;
        }
      })
    );
    runtime = Object.fromEntries(entries.filter((entry) => entry[1] !== undefined)) as Record<string, ProxyRuntimeStatus>;
  }

  function statusColor(proxyName: string, paused: boolean): StatusColor {
    if (paused) return 'gray';
    const state = runtime[proxyName]?.state;
    if (state === 'healthy') return 'green';
    if (state === 'degraded' || state === 'misconfigured') return 'amber';
    if (state === 'failed' || state === 'unreachable') return 'red';
    return 'gray';
  }

  function select(name: string) {
    goToTab('proxies', name);
  }

  onMount(() => {
    void loadProxies().then(refreshRuntime);
    return startProxyMonitoring(refreshRuntime);
  });
</script>

{#snippet actions()}
  {#if $accessMode.localControl}
    <ActionButton title={m.proxies_add()} tone="accent" onclick={() => openProxyAddModal()}>
      <Icon name="plus" class="w-3.5 h-3.5" />
    </ActionButton>
  {/if}
{/snippet}

<ListPanel title={m.proxies_title()} {actions}>
  {#if !$proxiesLoaded}
    <LoadingRow />
  {:else if $proxies.length === 0}
    <EmptyState title={m.proxies_empty()} size="sm" />
  {:else}
    {#each $proxies as proxy (proxy.name)}
      {#snippet leading()}
        <StatusDot color={statusColor(proxy.name, proxy.paused)} pulse={runtime[proxy.name]?.state === 'healthy'} />
      {/snippet}
      {#snippet trailing()}
        <span class="text-[10px] tabular-nums text-gray-400 dark:text-gray-600">:{proxy.upstream_port}</span>
      {/snippet}
      <ListRow active={selected === proxy.name} onclick={() => select(proxy.name)} {leading} {trailing}>
        <span class="min-w-0">
          <span class="block truncate">{proxy.domain}</span>
          {#if proxy.managed || proxy.paused}
            <span class="block text-[10px] font-normal text-gray-400 dark:text-gray-500">
              {proxy.managed ? m.proxies_managed() : ''}{proxy.managed && proxy.paused ? ' · ' : ''}{proxy.paused ? m.proxies_paused() : ''}
            </span>
          {/if}
        </span>
      </ListRow>
    {/each}
  {/if}
</ListPanel>
