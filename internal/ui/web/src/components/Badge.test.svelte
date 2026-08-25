<script lang="ts">
  import Badge, { type BadgeTone } from './Badge.svelte';

  interface Props {
    tone: BadgeTone;
    brand?: string;
    dot?: boolean;
    title?: string;
    onclick?: (e: MouseEvent) => void;
    label: string;
  }
  let { tone, brand, dot, title, onclick, label }: Props = $props();
</script>

<Badge {tone} {brand} {dot} {title} {onclick}>{label}</Badge>
