<script lang="ts">
  import SiteControls from './SiteControls.svelte';
  import SiteServiceCard from './SiteServiceCard.svelte';
  import SiteRequestTiming from './SiteRequestTiming.svelte';
  import DetailButton from '$components/DetailButton.svelte';
  import type { Site } from '$stores/sites';
  import { proxies } from '$stores/proxies';
  import { openProxyAddModal, openProxyEditModal } from '$stores/modals';
  import { suggestUnifiedDomain } from '$lib/fullstack';
  import { m } from '../../paraglide/messages.js';

  interface Props {
    site: Site;
    activeWorktreeBranch?: string;
  }
  let { site, activeWorktreeBranch = '' }: Props = $props();

  const svcNames = $derived(site.services || []);

  // Fork addition: the fullstack proxy this site is the API of, if any. Lives
  // on the overview beside the runtime controls, which is where it sat before
  // upstream split SiteDetail's overview out into this component.
  const siteName = $derived(site.name ?? site.domain.replace(/\.localhost$/, ''));
  const boundProxy = $derived(
    $proxies.find((p) => p.site === siteName || (p.routes ?? []).some((r) => r.site === siteName))
  );
</script>

{#snippet sectionTitle(title: string)}
  <h3 class="mb-2.5 text-[11px] font-semibold uppercase tracking-wider text-gray-400 dark:text-gray-500">
    {title}
  </h3>
{/snippet}

<div class="flex-1 min-h-0 overflow-y-auto p-4 space-y-4">
  <section>
    {@render sectionTitle(m.sites_overview_runtimeWorkers())}
    <SiteControls {site} {activeWorktreeBranch} />
  </section>

  <section>
    {@render sectionTitle('Proxy fullstack')}
    {#if boundProxy}
      <div class="flex items-center justify-between gap-2">
        <span class="text-xs text-gray-600 dark:text-gray-300">
          API em
          <a
            href={`${boundProxy.secured ? 'https' : 'http'}://${boundProxy.domain}`}
            target="_blank"
            rel="noopener"
            class="font-mono text-lerd-red hover:underline">↗ {boundProxy.domain}</a
          >
          <span class="text-gray-400">· {(boundProxy.routes ?? []).map((r) => r.path).join(' ')}</span>
        </span>
        <DetailButton onclick={() => openProxyEditModal(boundProxy)}>Editar</DetailButton>
      </div>
    {:else}
      <div class="flex items-center justify-between gap-2">
        <span class="text-xs text-gray-500">Servir este site como API sob um domínio único com seu SPA.</span>
        <DetailButton
          tone="primary"
          onclick={() =>
            openProxyAddModal({ fullstack: true, apiSite: siteName, domain: suggestUnifiedDomain(siteName) })}
          >+ Criar proxy fullstack</DetailButton
        >
      </div>
    {/if}
  </section>

  {#if svcNames.length > 0}
    <section>
      {@render sectionTitle(m.services_title())}
      <div class="grid grid-cols-2 sm:grid-cols-3 xl:grid-cols-4 gap-3">
        {#each svcNames as name (name)}
          <SiteServiceCard {name} database={site.db_database} />
        {/each}
      </div>
    </section>
  {/if}

  <SiteRequestTiming {site} {activeWorktreeBranch} />
</div>
