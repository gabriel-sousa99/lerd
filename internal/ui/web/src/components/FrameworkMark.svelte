<script lang="ts">
  import { frameworkMarks } from '$stores/frameworkMarks';
  import { brandTintStyle } from '$lib/brandTint';

  interface Props {
    // The framework's store name (laravel, symfony), not its display label.
    name: string | undefined;
    size?: 'sm' | 'md' | 'lg';
    // A mark inside a coloured pill takes the pill's colour rather than the
    // brand tone, so the badge reads as one object instead of two.
    tint?: boolean;
  }
  let { name, size = 'sm', tint = true }: Props = $props();

  // A framework with no mark renders nothing at all rather than a placeholder
  // glyph: the label beside it already says which framework it is, so a stand-in
  // would only add noise to a header that is mostly text.
  const mark = $derived(name ? $frameworkMarks[name] : undefined);
  const style = $derived(tint ? brandTintStyle(mark?.color) : '');
  const box = $derived(size === 'lg' ? 'w-5 h-5' : size === 'md' ? 'w-4 h-4' : 'w-3.5 h-3.5');
  const wrapper = $derived(tint ? 'mark-ink' : 'mark-inherit');
</script>

{#if mark?.svg}
  <span class="{wrapper} {box}" style={style} aria-hidden="true">{@html mark.svg}</span>
{/if}
