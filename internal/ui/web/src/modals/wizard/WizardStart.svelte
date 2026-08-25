<script lang="ts">
  import Icon, { type IconName } from '$components/Icon.svelte';
  import { m } from '../../paraglide/messages.js';

  interface Props {
    onchoose: (mode: 'link' | 'create') => void;
  }
  let { onchoose }: Props = $props();

  const choices = [
    {
      mode: 'link' as const,
      icon: 'sites' as IconName,
      title: m.siteWizard_startLinkTitle(),
      description: m.siteWizard_startLinkDesc()
    },
    {
      mode: 'create' as const,
      icon: 'plus' as IconName,
      title: m.siteWizard_startCreateTitle(),
      description: m.siteWizard_startCreateDesc()
    }
  ];
</script>

<div class="px-5 py-4 space-y-2">
  {#each choices as choice (choice.mode)}
    <button
      onclick={() => onchoose(choice.mode)}
      class="w-full flex items-start gap-3 p-3 text-left rounded-lg border border-gray-200 dark:border-lerd-border hover:border-lerd-red/50 hover:bg-gray-50 dark:hover:bg-white/5 transition-colors"
    >
      <Icon name={choice.icon} class="w-5 h-5 mt-0.5 shrink-0 text-gray-400" />
      <span class="min-w-0">
        <span class="block text-sm font-medium text-gray-900 dark:text-white">{choice.title}</span>
        <span class="block text-xs text-gray-500 dark:text-gray-400">{choice.description}</span>
      </span>
    </button>
  {/each}
</div>
