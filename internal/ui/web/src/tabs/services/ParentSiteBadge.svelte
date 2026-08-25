<script lang="ts">
  import StatusDot from '$components/StatusDot.svelte';
  import { goToTab } from '$stores/route';
  import { sites, siteDotColor } from '$stores/sites';

  interface Props {
    domain: string;
  }
  let { domain }: Props = $props();

  const color = $derived(siteDotColor($sites.find((s) => s.domain === domain)));

  function open() {
    goToTab('sites', domain);
  }
</script>

<button
  onclick={open}
  class="inline-flex items-center gap-1.5 text-xs font-medium bg-gray-100 dark:bg-white/5 hover:bg-gray-200 dark:hover:bg-white/10 border border-gray-200 dark:border-lerd-border text-gray-700 dark:text-gray-300 rounded-full px-2 py-0.5 transition-colors"
>
  <StatusDot {color} size="xs" />
  {domain}
</button>
