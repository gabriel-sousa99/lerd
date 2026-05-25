<script lang="ts">
  import ActionButton from '$components/ActionButton.svelte';
  import Icon from '$components/Icon.svelte';

  interface Props {
    onAdd: () => void;
  }
  let { onAdd }: Props = $props();
</script>

<div
  class="flex items-center justify-between px-4 py-3 border-b border-gray-100 dark:border-lerd-border shrink-0 min-h-[48px]"
>
  <span class="text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider">
    Proxies
  </span>
  <ActionButton title="Adicionar proxy" tone="accent" onclick={onAdd}>
    <Icon name="plus" class="w-3.5 h-3.5" />
  </ActionButton>
</div>
