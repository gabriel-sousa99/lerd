<script lang="ts">
  import Icon from '$components/Icon.svelte';
  import { fly } from 'svelte/transition';
  import { openLinkModal } from '$stores/modals';
  import { wizardBubble } from '$stores/wizard';
  import { clearWizardState } from '$lib/wizardState';
  import { m } from '../paraglide/messages.js';

  // A flow sent to the background needs somewhere to come back to. The bubble is
  // that place: it says what is happening and reopens the wizard where it was.
  function resume() {
    openLinkModal();
  }
</script>

{#if $wizardBubble}
  <div
    class="fixed bottom-3 right-3 z-50 flex items-center gap-2 rounded-full border border-gray-200 dark:border-lerd-border bg-white dark:bg-lerd-card shadow-lg pl-3 pr-1.5 py-1.5"
    transition:fly={{ y: 12, duration: 200 }}
  >
    <button class="flex items-center gap-2 min-w-0 text-left" onclick={resume}>
      {#if $wizardBubble.running}
        <Icon name="spinner" class="w-4 h-4 shrink-0 text-lerd-red animate-spin" />
      {:else}
        <Icon name="check" class="w-4 h-4 shrink-0 text-emerald-500" />
      {/if}
      <span class="min-w-0">
        <span class="block text-xs font-medium text-gray-900 dark:text-white truncate">
          {$wizardBubble.project || m.siteWizard_title()}
        </span>
        <span class="block text-[11px] text-gray-500 dark:text-gray-400 truncate">
          {$wizardBubble.running
            ? $wizardBubble.label || m.siteWizard_bubbleRunning()
            : m.siteWizard_bubbleWaiting()}
        </span>
      </span>
    </button>

    <!-- Dismissing only drops the wizard's note; a run already on the host is
         the daemon's to finish either way. -->
    <button
      class="shrink-0 w-6 h-6 flex items-center justify-center rounded-full text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 hover:bg-gray-100 dark:hover:bg-white/5 transition-colors"
      aria-label={m.common_close()}
      title={m.common_close()}
      onclick={clearWizardState}
    >
      <Icon name="close" class="w-3.5 h-3.5" />
    </button>
  </div>
{/if}
