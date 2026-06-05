<script lang="ts">
  import DetailPanel from '$components/DetailPanel.svelte';
  import SiteHeader from './SiteHeader.svelte';
  import SiteControls from './SiteControls.svelte';
  import SiteLogs from './SiteLogs.svelte';
  import SiteTinkerTab from './SiteTinkerTab.svelte';
  import SiteEnvTab from './SiteEnvTab.svelte';
  import SiteNginxModal from '../../modals/SiteNginxModal.svelte';
  import DumpsTab from '$tabs/DumpsTab.svelte';
  import DetailButton from '$components/DetailButton.svelte';
  import { resumeSite, loadSites, activeWorktreeDomain, type Site } from '$stores/sites';
  import { proxies } from '$stores/proxies';
  import { openProxyAddModal, openProxyEditModal } from '$stores/modals';
  import { suggestUnifiedDomain } from '$lib/fullstack';
  import { routeRest } from '$stores/route';
  import { m } from '../../paraglide/messages.js';

  let resumeBusy = $state(false);
  async function onResume() {
    resumeBusy = true;
    try {
      await resumeSite(site.domain);
      await loadSites();
    } finally {
      resumeBusy = false;
    }
  }

  interface Props {
    site: Site;
  }
  let { site }: Props = $props();

  type TabId = 'overview' | 'tinker' | 'env' | 'dumps';
  const TAB_STORAGE_KEY = 'lerd:siteDetailTab';

  function readStoredTab(): TabId {
    if (typeof localStorage === 'undefined') return 'overview';
    const v = localStorage.getItem(TAB_STORAGE_KEY);
    if (v === 'tinker' || v === 'env' || v === 'dumps') return v;
    return 'overview';
  }

  let active = $state<TabId>(readStoredTab());
  let activeWorktreeBranch = $state<string>('');
  let nginxOpen = $state(false);
  const canTinker = $derived(Boolean(site.uses_php));
  const canDumps = $derived(Boolean(site.uses_php));
  const canEnv = $derived(Boolean(site.has_env));
  // A lone Overview tab can't be switched to anything, so don't render the tab
  // row at all when no other tab is available (e.g. static sites).
  const hasExtraTabs = $derived(canEnv || canTinker || canDumps);

  const siteName = $derived(site.name ?? site.domain.replace(/\.localhost$/, ''));
  const boundProxy = $derived(
    $proxies.find((p) => p.site === siteName || (p.routes ?? []).some((r) => r.site === siteName))
  );

  // The route can deep-link a sub-tab (e.g. dump notifications go to
  // #sites/<domain>/dumps). When the second segment names a tab, honour it
  // and overwrite the stored selection.
  $effect(() => {
    const seg = $routeRest.split('/')[1] ?? '';
    if (seg === 'nginx') {
      nginxOpen = true;
    } else if (seg === 'tinker' || seg === 'env' || seg === 'dumps' || seg === 'overview') {
      active = seg;
    }
  });

  $effect(() => {
    if (active === 'tinker' && !canTinker) active = 'overview';
    if (active === 'env' && !canEnv) active = 'overview';
    if (active === 'dumps' && !canDumps) active = 'overview';
  });

  $effect(() => {
    if (typeof localStorage !== 'undefined') {
      localStorage.setItem(TAB_STORAGE_KEY, active);
    }
  });

  // Reset selection when the site changes or the chosen branch disappears.
  $effect(() => {
    if (!activeWorktreeBranch) return;
    const exists = (site.worktrees || []).some((w) => w.branch === activeWorktreeBranch);
    if (!exists) activeWorktreeBranch = '';
  });

  const tabBtn = (tab: TabId, isActive: boolean) =>
    'pb-1 text-xs font-medium border-b-2 transition-colors ' +
    (isActive
      ? 'border-lerd-red text-lerd-red'
      : 'border-transparent text-gray-500 hover:text-gray-700 dark:hover:text-gray-300');
</script>

{#snippet tabs()}
  <button class={tabBtn('overview', active === 'overview')} onclick={() => (active = 'overview')}>{m.sites_tabs_overview()}</button>
  {#if canEnv}
    <button class={tabBtn('env', active === 'env')} onclick={() => (active = 'env')}>{m.sites_tabs_env()}</button>
  {/if}
  {#if canTinker}
    <button class={tabBtn('tinker', active === 'tinker')} onclick={() => (active = 'tinker')}>{m.sites_tabs_tinker()}</button>
  {/if}
  {#if canDumps}
    <button class={tabBtn('dumps', active === 'dumps')} onclick={() => (active = 'dumps')}>{m.nav_dumps()}</button>
  {/if}
{/snippet}

<DetailPanel>
  <SiteHeader
    {site}
    tabs={site.paused || !hasExtraTabs ? undefined : tabs}
    {activeWorktreeBranch}
    onWorktreeChange={(b) => (activeWorktreeBranch = b)}
    onOpenNginx={() => (nginxOpen = true)}
  />
  {#if site.paused}
    <div class="flex-1 flex items-center justify-center px-6">
      <div class="flex flex-col items-center gap-3 max-w-md text-center">
        <svg class="w-10 h-10 text-gray-400 dark:text-gray-600" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
          <rect x="6" y="5" width="4" height="14" rx="1" />
          <rect x="14" y="5" width="4" height="14" rx="1" />
        </svg>
        <h2 class="text-base font-semibold text-gray-700 dark:text-gray-200">{m.sites_pausedDetail_title()}</h2>
        <p class="text-xs text-gray-500 dark:text-gray-400 leading-relaxed">
          {m.sites_pausedDetail_hint({ domain: site.domain })}
        </p>
        <button
          type="button"
          onclick={onResume}
          disabled={resumeBusy}
          class="mt-1 inline-flex items-center gap-1.5 px-3 py-1.5 rounded-md text-xs font-medium bg-lerd-red hover:bg-lerd-redhov text-white disabled:opacity-50 transition-colors"
        >
          {resumeBusy ? m.sites_pausedDetail_busy() : m.sites_pausedDetail_action()}
        </button>
      </div>
    </div>
  {:else if active === 'overview'}
    <SiteControls {site} {activeWorktreeBranch} />
    <section class="px-6 py-4 space-y-2 border-b border-gray-100 dark:border-lerd-border">
      <h2 class="text-xs font-semibold uppercase tracking-wider text-gray-500 dark:text-gray-400">Proxy fullstack</h2>
      {#if boundProxy}
        <div class="flex items-center justify-between gap-2">
          <span class="text-xs text-gray-600 dark:text-gray-300">
            API em
            <a href={`${boundProxy.secured ? 'https' : 'http'}://${boundProxy.domain}`} target="_blank" rel="noopener" class="font-mono text-lerd-red hover:underline">↗ {boundProxy.domain}</a>
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
    <SiteLogs {site} {activeWorktreeBranch} />
  {:else if active === 'env'}
    {#key site.domain + '@' + activeWorktreeBranch}
      <SiteEnvTab {site} branch={activeWorktreeBranch} />
    {/key}
  {:else if active === 'tinker'}
    {#key site.domain + '@' + activeWorktreeBranch}
      <SiteTinkerTab {site} branch={activeWorktreeBranch} />
    {/key}
  {:else if active === 'dumps'}
    <DumpsTab siteScope={site.name} />
  {/if}
</DetailPanel>

<SiteNginxModal
  {site}
  domain={activeWorktreeDomain(site, activeWorktreeBranch)}
  open={nginxOpen}
  onclose={() => (nginxOpen = false)}
/>
