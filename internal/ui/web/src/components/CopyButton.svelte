<script lang="ts">
  import { onDestroy } from 'svelte';
  import { tooltip } from '$lib/tooltip';
  import Icon from './Icon.svelte';

  interface Props {
    // A thunk defers work that only matters on click, such as inlining bindings
    // into a statement for every row of a long list.
    text: string | (() => string);
    label: string;
    class?: string;
    size?: string;
    // faint keeps the button quiet where it sits beside a control of its own,
    // so it doesn't read as part of that control.
    tone?: 'default' | 'faint';
  }
  let { text, label, class: cls = '', size = 'w-3.5 h-3.5', tone = 'default' }: Props = $props();

  const idle = $derived(
    tone === 'faint'
      ? 'text-gray-300 dark:text-gray-600 hover:text-gray-600 dark:hover:text-gray-200'
      : 'text-gray-400 hover:text-gray-600 dark:hover:text-gray-200'
  );

  let copied = $state(false);
  let timer: ReturnType<typeof setTimeout> | null = null;
  onDestroy(() => {
    if (timer) clearTimeout(timer);
  });

  async function copy() {
    try {
      await navigator.clipboard.writeText(typeof text === 'function' ? text() : text);
      copied = true;
      if (timer) clearTimeout(timer);
      timer = setTimeout(() => (copied = false), 1500);
    } catch {
      /* no clipboard outside a secure context; leave the view untouched */
    }
  }
</script>

<button
  type="button"
  class="shrink-0 flex items-center {copied
    ? 'text-emerald-600 dark:text-emerald-500'
    : idle} {cls}"
  onclick={copy}
  use:tooltip={label}
  aria-label={label}
>
  <Icon name={copied ? 'check' : 'clipboard'} class={size} />
</button>
