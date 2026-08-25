<script lang="ts">
  import { glyphOr } from '$lib/dashboardIcons';
  import { workerMarks, workerMarkFor } from '$stores/workerMarks';
  import { frameworkMarks } from '$stores/frameworkMarks';
  import { serviceIcons } from '$stores/serviceIcons';
  import { brandTintStyle } from '$lib/brandTint';

  interface Props {
    // The worker's store name (queue, vite), not its display label.
    worker: string;
    // The framework of the site running it, which decides both the tone and,
    // where two frameworks name different icons for the same worker, which one.
    framework?: string;
    // A worker that is lerd's own rather than a framework's (the Stripe
    // listener) borrows the mark of the service preset that already ships it.
    preset?: string;
    compact?: boolean;
    // The plate a group heading carries, small enough to sit on a line of
    // uppercase label text.
    heading?: boolean;
    // A heading standing over rows from more than one framework speaks for all
    // of them, so it drops the brand tone rather than claiming one of theirs.
    tint?: boolean;
  }
  let { worker, framework, preset, compact = false, heading = false, tint = true }: Props = $props();

  const declared = $derived(workerMarkFor($workerMarks, worker, framework));
  const storeMark = $derived(declared?.icon ? $workerMarks.marks[declared.icon] : undefined);
  const presetMark = $derived(preset ? $serviceIcons[preset] : undefined);
  const fwMark = $derived(framework ? $frameworkMarks[framework]?.svg : undefined);

  // First hit wins: the mark the worker names, then the glyph it names inked in
  // its tone, then the mark of the framework running it, and a plain gear for a
  // worker whose definition says nothing at all.
  const mark = $derived(presetMark ?? storeMark ?? (declared?.icon ? undefined : fwMark));
  const glyph = $derived(mark ? '' : glyphOr(declared?.icon, 'gear'));
  const tone = $derived(
    tint
      ? brandTintStyle(declared?.color ?? (framework ? $frameworkMarks[framework]?.color : undefined))
      : ''
  );

  const box = $derived(
    heading
      ? 'w-5 h-5 rounded-md'
      : compact
        ? 'w-8 h-8 rounded-lg'
        : 'w-9 h-9 rounded-lg transition-transform group-hover:scale-105'
  );
  const size = $derived(heading ? 'w-3 h-3' : compact ? 'w-4 h-4' : 'w-5 h-5');
</script>

<span
  class="shrink-0 inline-flex items-center justify-center {tone
    ? 'mark-tint'
    : 'bg-gray-100 dark:bg-white/5 text-gray-500 dark:text-gray-400'} {box}"
  style={tone}
>
  {#if mark}
    <span class="mark-glyph {size}">{@html mark}</span>
  {:else}
    <svg class={size} fill="none" stroke="currentColor" viewBox="0 0 24 24">{@html glyph}</svg>
  {/if}
</span>
