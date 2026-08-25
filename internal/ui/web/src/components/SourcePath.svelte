<script lang="ts">
  import { openInEditor } from '$lib/editor';
  import { tooltip } from '$lib/tooltip';
  import CopyButton from './CopyButton.svelte';
  import { m } from '../paraglide/messages.js';

  interface Props {
    file: string;
    line?: number;
    // Vendor frames and header paths sit behind the app's own code.
    muted?: boolean;
    // Keep only the tail of a long path where the row has no width to spare.
    short?: boolean;
  }
  let { file, line, muted = false, short = false }: Props = $props();

  const reference = $derived(line ? `${file}:${line}` : file);
  const shown = $derived(short ? shorten(file) : file);

  function shorten(path: string): string {
    const parts = path.split('/');
    return parts.length <= 3 ? path : '…/' + parts.slice(-3).join('/');
  }
</script>

<span
  class="inline-flex items-stretch max-w-full rounded-sm border border-gray-200 dark:border-lerd-border overflow-hidden align-middle"
>
  <button
    type="button"
    class="px-1.5 py-0.5 font-mono text-left hover:underline {muted
      ? 'hover:text-gray-600 dark:hover:text-gray-300'
      : 'text-lerd-red'} {short ? 'min-w-0 truncate' : 'break-all'}"
    onclick={() => openInEditor(file, line ?? 1)}
    use:tooltip={short ? `${m.queries_openInEditor()} — ${reference}` : m.queries_openInEditor()}
  >{shown}{#if line}:{line}{/if}</button>
  <CopyButton
    text={reference}
    label={m.queries_copyPath()}
    tone="faint"
    size="w-3.5 h-3.5"
    class="px-1.5 border-l border-gray-200 dark:border-lerd-border transition-colors hover:bg-gray-100 dark:hover:bg-white/10"
  />
</span>
