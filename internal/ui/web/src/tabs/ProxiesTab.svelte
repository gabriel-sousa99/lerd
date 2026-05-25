<script lang="ts">
  import { onMount } from 'svelte';
  import { proxies, proxiesLoaded, loadProxies } from '$stores/proxies';
  import { goToTab, routeRest } from '$stores/route';
  import { openProxyAddModal } from '$stores/modals';
  import LoadingRow from '$components/LoadingRow.svelte';
  import EmptyState from '$components/EmptyState.svelte';
  import ProxyHeader from './proxies/ProxyHeader.svelte';

  const selected = $derived($routeRest);

  onMount(() => {
    if (!$proxiesLoaded) void loadProxies();
  });

  function select(name: string) {
    goToTab('proxies', name);
  }
</script>

<div class="flex flex-col h-full">
  <ProxyHeader onAdd={openProxyAddModal} />

  <div class="flex-1 overflow-y-auto">
    {#if !$proxiesLoaded}
      <LoadingRow />
    {:else if $proxies.length === 0}
      <EmptyState title="Nenhum proxy configurado" size="sm" />
    {:else}
      {#each $proxies as p (p.name)}
        <button
          onclick={() => select(p.name)}
          class="w-full flex items-center gap-2 px-3 py-2.5 text-left transition-colors border-b border-gray-50 dark:border-lerd-border/50 {selected === p.name
            ? 'bg-lerd-red/10 text-lerd-red'
            : 'text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-white/3'}"
        >
          <span class="flex-1 min-w-0">
            <span class="block text-sm truncate font-medium">{p.domain}</span>
            <span class="block text-[11px] text-gray-500 dark:text-gray-400 truncate">
              :{p.upstream_port}
              {#if p.managed}· managed{/if}
              {#if p.paused}· paused{/if}
            </span>
          </span>
          {#if p.secured}
            <svg class="w-3 h-3 shrink-0 text-emerald-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z"/>
            </svg>
          {/if}
          {#if p.paused}
            <svg class="w-3 h-3 shrink-0 text-gray-400" fill="currentColor" viewBox="0 0 24 24">
              <path d="M6 5h4v14H6zM14 5h4v14h-4z"/>
            </svg>
          {/if}
        </button>
      {/each}
    {/if}
  </div>
</div>
