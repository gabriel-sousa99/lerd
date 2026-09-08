<script lang="ts">
  import type { Site } from '$stores/sites';
  interface Props {
    site: Site;
    domain?: string;
    scope?: string;
    onSaved?: () => void;
  }
  let { site, domain, scope = 'server', onSaved = () => {} }: Props = $props();
</script>

<div data-testid="nginx-editor-stub" data-scope={scope}>{domain || site.domain}</div>
<button data-testid="stub-saved" onclick={() => onSaved()}>save</button>
