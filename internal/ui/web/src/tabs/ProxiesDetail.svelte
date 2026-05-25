<script lang="ts">
  import { proxies } from '$stores/proxies';
  import { routeRest } from '$stores/route';
  import DetailPanel from '$components/DetailPanel.svelte';
  import EmptyState from '$components/EmptyState.svelte';
  import ProxyDetailPanel from './proxies/ProxyDetailPanel.svelte';

  const current = $derived($proxies.find((p) => p.name === $routeRest));
</script>

<DetailPanel>
  {#if !$routeRest}
    <div class="flex-1 flex items-center justify-center">
      <EmptyState title="Selecione um proxy para ver os detalhes" />
    </div>
  {:else if !current}
    <div class="flex-1 flex items-center justify-center">
      <EmptyState title="Proxy não encontrado" />
    </div>
  {:else}
    {#key current.name}
      <ProxyDetailPanel proxy={current} />
    {/key}
  {/if}
</DetailPanel>
