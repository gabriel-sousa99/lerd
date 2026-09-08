<script lang="ts">
  import StatusPill from '$components/StatusPill.svelte';
  import { PRERELEASE_LABEL } from '$stores/phpVersions';
  import { m } from '../../paraglide/messages.js';

  interface Props {
    version: string;
    patch?: string;
    running: boolean;
    isDefault: boolean;
    updateAvailable?: boolean;
    prerelease?: boolean;
    selected: boolean;
    onselect: () => void;
  }
  let {
    version,
    patch,
    running,
    isDefault,
    updateAvailable = false,
    prerelease = false,
    selected,
    onselect
  }: Props = $props();

  // Show the full build when known ("8.5.8"), with the patch tail dimmed so it
  // reads as one number; fall back to the minor until the probe lands. A
  // prerelease build ("8.6.0beta2") is too wide to sit next to the beta mark,
  // so the card keeps the minor and the tooltip carries the full string.
  const display = $derived(patch || version);
  const minor = $derived(display.split('.').slice(0, 2).join('.'));
  const tail = $derived(
    prerelease || display.split('.').length < 3 ? '' : '.' + display.split('.').slice(2).join('.')
  );
</script>

<button
  type="button"
  onclick={onselect}
  title={'PHP ' + display + ' — ' + (running ? m.common_running() : m.common_stopped()) + (isDefault ? ' · ' + m.common_default() : '') + (updateAvailable ? ' · ' + m.system_php_baseUpdateHint() : '') + (prerelease ? ' · ' + PRERELEASE_LABEL : '')}
  class="shrink-0 w-[9.5rem] snap-start text-left flex flex-col gap-2.5 rounded-2xl border p-3 transition-colors {selected
    ? 'border-lerd-red bg-white dark:bg-lerd-card ring-1 ring-lerd-red'
    : 'border-gray-200 dark:border-lerd-border bg-white dark:bg-lerd-card hover:border-gray-300 dark:hover:border-gray-600'}"
>
  <div class="flex items-center justify-between gap-2">
    <span
      class="text-[8px] font-extrabold tracking-wide text-white rounded-md px-1.5 py-1 leading-none"
      style="background: linear-gradient(150deg, #8892bf, #6b74a3);">PHP</span>
    <StatusPill
      tone={running ? 'ok' : 'muted'}
      label={running ? m.common_running() : m.common_stopped()}
    />
  </div>
  <div class="flex items-center gap-1.5 min-w-0">
    <span class="font-mono text-2xl font-semibold tabular-nums tracking-tight leading-none truncate text-gray-900 dark:text-gray-100">
      {minor}<span class="text-[0.8em] font-medium text-gray-400 dark:text-gray-500">{tail}</span>
    </span>
    {#if isDefault}
      <svg
        class="w-3.5 h-3.5 shrink-0 text-lerd-red"
        fill="currentColor"
        viewBox="0 0 20 20"
        aria-label={m.common_default()}
      >
        <path d="M10 1.5l2.6 5.27 5.82.85-4.21 4.1.99 5.78L10 14.77l-5.2 2.73.99-5.78L1.58 7.62l5.82-.85L10 1.5z" />
      </svg>
    {/if}
    {#if prerelease}
      <span
        class="shrink-0 text-[9px] font-bold uppercase tracking-wide leading-none rounded px-1 py-0.5 text-amber-700 bg-amber-100 dark:text-amber-300 dark:bg-amber-400/15"
      >
        {PRERELEASE_LABEL}
      </span>
    {/if}
    {#if updateAvailable}
      <svg
        class="w-3.5 h-3.5 shrink-0 text-yellow-600 dark:text-yellow-400"
        fill="none"
        stroke="currentColor"
        viewBox="0 0 24 24"
        aria-label={m.system_php_baseUpdate()}
      >
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 16V4m0 0L3 8m4-4l4 4m6 0v12m0 0l4-4m-4 4l-4-4" />
      </svg>
    {/if}
  </div>
</button>
