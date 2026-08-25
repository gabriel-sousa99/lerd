<script lang="ts">
  import Popover from '$components/Popover.svelte';
  import { buttonMenuBaseClass, buttonMenuToneClass } from '$components/ButtonMenu.svelte';
  import Icon from '$components/Icon.svelte';
  import FrameworkMark from '$components/FrameworkMark.svelte';
  import StatusDot from '$components/StatusDot.svelte';
  import { sites, siteDotColor } from '$stores/sites';
  import { goToTab } from '$stores/route';
  import { m } from '../paraglide/messages.js';

  interface Props {
    domains: string[];
  }
  let { domains }: Props = $props();

  // The state comes from the sites store, so a row says what the site is
  // actually doing instead of carrying a decorative dot.
  const rows = $derived(
    domains.map((d) => {
      const site = $sites.find((s) => s.domain === d);
      return { domain: d, color: siteDotColor(site), framework: site?.framework };
    })
  );
  const label = $derived(m.sites_count({ count: domains.length }));

  function open(close: () => void, domain: string) {
    close();
    goToTab('sites', domain);
  }
</script>

{#if domains.length > 0}
  <Popover {label} align="right" width={260}>
    {#snippet triggerButton(toggle: () => void, isOpen: boolean)}
      <button
        type="button"
        onclick={toggle}
        title={label}
        aria-label={label}
        aria-expanded={isOpen}
        class="{buttonMenuBaseClass} rounded-lg {buttonMenuToneClass.secondary}"
      >
        <Icon name="sites" class="w-3.5 h-3.5" />
        <span class="tabular-nums">{domains.length}</span>
        <svg class="w-3 h-3" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
          <polyline points="6 9 12 15 18 9"/>
        </svg>
      </button>
    {/snippet}
    {#snippet children(close: () => void)}
      <ul class="py-1 max-h-72 overflow-y-auto">
        {#each rows as row (row.domain)}
          <li>
            <button
              type="button"
              onclick={() => open(close, row.domain)}
              class="w-full flex items-center gap-2 px-3 py-1.5 text-xs text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-white/5 transition-colors"
            >
              <StatusDot color={row.color} size="xs" />
              <span class="flex-1 truncate text-left">{row.domain}</span>
              <FrameworkMark name={row.framework} />
            </button>
          </li>
        {/each}
      </ul>
    {/snippet}
  </Popover>
{/if}
