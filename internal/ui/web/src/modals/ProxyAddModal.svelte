<script lang="ts">
  import Modal from '$components/Modal.svelte';
  import DetailButton from '$components/DetailButton.svelte';
  import { closeModal } from '$stores/modals';
  import { createProxy } from '$stores/proxies';
  import { goToTab } from '$stores/route';

  let domain = $state('');
  let port = $state<number>(9000);
  let path = $state('');
  let managed = $state(false);
  let cmd = $state('npm run dev');
  let nodeVersion = $state('20');
  let autostart = $state(false);
  let noSecure = $state(false);
  let error = $state<string | null>(null);
  let saving = $state(false);

  const canSubmit = $derived(domain.trim().length > 0 && port > 0 && port <= 65535);

  async function submit(e: SubmitEvent) {
    e.preventDefault();
    if (!canSubmit || saving) return;
    error = null;
    saving = true;
    try {
      const created = await createProxy({
        domain: domain.trim(),
        port,
        path: path.trim() || undefined,
        no_secure: noSecure,
        managed,
        cmd: managed ? cmd : undefined,
        node_version: managed ? nodeVersion : undefined,
        autostart: managed ? autostart : false
      });
      closeModal();
      if (created?.name) {
        goToTab('proxies', created.name);
      }
    } catch (e) {
      error = e instanceof Error ? e.message : String(e);
    } finally {
      saving = false;
    }
  }
</script>

<Modal open title="Adicionar proxy" onclose={closeModal} size="md">
  <form id="proxy-add-form" class="px-5 py-4 space-y-4" onsubmit={submit}>
    <label class="block space-y-1">
      <span class="text-xs font-medium text-gray-600 dark:text-gray-300">Domínio (sem esquema)</span>
      <input
        type="text"
        bind:value={domain}
        placeholder="gestao-clientes.localhost"
        required
        class="w-full bg-white dark:bg-lerd-bg border border-gray-200 dark:border-lerd-border rounded-md px-3 py-1.5 text-sm font-mono focus:outline-none focus:border-lerd-red"
      />
    </label>

    <label class="block space-y-1">
      <span class="text-xs font-medium text-gray-600 dark:text-gray-300">Porta do dev server</span>
      <input
        type="number"
        bind:value={port}
        min="1"
        max="65535"
        required
        class="w-full bg-white dark:bg-lerd-bg border border-gray-200 dark:border-lerd-border rounded-md px-3 py-1.5 text-sm focus:outline-none focus:border-lerd-red"
      />
    </label>

    <label class="block space-y-1">
      <span class="text-xs font-medium text-gray-600 dark:text-gray-300">
        Pasta do projeto <span class="text-gray-400">(opcional, obrigatória se managed)</span>
      </span>
      <input
        type="text"
        bind:value={path}
        placeholder="/home/u/projetos/gestao-clientes-spa"
        class="w-full bg-white dark:bg-lerd-bg border border-gray-200 dark:border-lerd-border rounded-md px-3 py-1.5 text-sm font-mono focus:outline-none focus:border-lerd-red"
      />
    </label>

    <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
      <input type="checkbox" bind:checked={noSecure} class="accent-lerd-red" />
      <span>HTTP apenas (sem mkcert)</span>
    </label>

    <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
      <input type="checkbox" bind:checked={managed} class="accent-lerd-red" />
      <span>Managed (lerd inicia o dev server)</span>
    </label>

    {#if managed}
      <fieldset class="space-y-3 border border-gray-200 dark:border-lerd-border rounded-md p-3">
        <label class="block space-y-1">
          <span class="text-xs font-medium text-gray-600 dark:text-gray-300">Comando</span>
          <input
            type="text"
            bind:value={cmd}
            class="w-full bg-white dark:bg-lerd-bg border border-gray-200 dark:border-lerd-border rounded-md px-3 py-1.5 text-sm font-mono focus:outline-none focus:border-lerd-red"
          />
        </label>

        <label class="block space-y-1">
          <span class="text-xs font-medium text-gray-600 dark:text-gray-300">Node major version</span>
          <input
            type="text"
            bind:value={nodeVersion}
            class="w-full bg-white dark:bg-lerd-bg border border-gray-200 dark:border-lerd-border rounded-md px-3 py-1.5 text-sm focus:outline-none focus:border-lerd-red"
          />
        </label>

        <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
          <input type="checkbox" bind:checked={autostart} class="accent-lerd-red" />
          <span>Iniciar com <code class="font-mono text-xs">lerd start</code></span>
        </label>
      </fieldset>
    {/if}

    {#if error}
      <p class="text-xs text-red-500">{error}</p>
    {/if}
  </form>

  {#snippet footer()}
    <DetailButton onclick={closeModal} disabled={saving}>Cancelar</DetailButton>
    <DetailButton
      tone="primary"
      onclick={() => {
        const form = document.getElementById('proxy-add-form') as HTMLFormElement | null;
        form?.requestSubmit();
      }}
      disabled={!canSubmit || saving}
      loading={saving}
    >
      Criar
    </DetailButton>
  {/snippet}
</Modal>
