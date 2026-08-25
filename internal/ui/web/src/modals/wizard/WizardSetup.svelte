<script lang="ts">
  import Icon from '$components/Icon.svelte';
  import WizardCheckList from './WizardCheckList.svelte';
  import type { SetupStep } from '$stores/wizard';
  import { m } from '../../paraglide/messages.js';

  interface Props {
    steps: SetupStep[];
    selected: string[];
    // Steps that already ran, in the order they ran, with how each ended.
    finished: Array<{ label: string; ok: boolean; error?: string }>;
    current: string;
    onchange: (selected: string[]) => void;
  }
  let { steps, selected, finished, current, onchange }: Props = $props();

  const items = $derived(
    steps.map((s) => ({
      value: s.label,
      label: s.label,
      note: s.optional ? m.siteWizard_stepOptional() : undefined
    }))
  );

  function outcome(label: string) {
    return finished.find((f) => f.label === label);
  }
</script>

<div class="px-5 py-4 space-y-3">
  {#if steps.length === 0}
    <p class="text-xs text-gray-500 dark:text-gray-400">{m.siteWizard_noSteps()}</p>
  {:else if finished.length === 0 && current === ''}
    <p class="text-xs text-gray-500 dark:text-gray-400">{m.siteWizard_setupIntro()}</p>
    <WizardCheckList {items} {selected} {onchange} columns={false} />
  {:else}
    <div class="space-y-1">
      {#each steps.filter((s) => selected.includes(s.label)) as step (step.label)}
        {@const result = outcome(step.label)}
        <div class="flex items-center gap-2 text-sm">
          {#if result?.ok}
            <Icon name="check" class="w-4 h-4 text-emerald-500 shrink-0" />
          {:else if result}
            <Icon name="warn" class="w-4 h-4 text-red-500 shrink-0" />
          {:else if current === step.label}
            <Icon name="spinner" class="w-4 h-4 text-gray-400 shrink-0 animate-spin" />
          {:else}
            <span class="w-4 h-4 shrink-0"></span>
          {/if}
          <span
            class="truncate {result && !result.ok
              ? 'text-red-500'
              : 'text-gray-700 dark:text-gray-300'}"
          >
            {step.label}
          </span>
        </div>
        {#if result && !result.ok && result.error}
          <p class="pl-6 text-xs text-red-500 break-words">{result.error}</p>
        {/if}
      {/each}
    </div>
  {/if}
</div>
