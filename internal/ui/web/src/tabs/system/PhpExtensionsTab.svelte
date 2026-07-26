<script lang="ts">
  import Badge from '$components/Badge.svelte';
  import DetailButton from '$components/DetailButton.svelte';
  import EmptyState from '$components/EmptyState.svelte';
  import { fetchPhpExtensions, type PhpExtensionsReport, type PhpSetState } from '$stores/phpVersions';
  import { addPhpExtension, removePhpExtension } from '$stores/phpExtensions';
  import { m } from '../../paraglide/messages.js';

  interface Props {
    version: string;
  }
  let { version }: Props = $props();

  let report = $state<PhpExtensionsReport | null>(null);
  let loading = $state(true);
  let error = $state('');

  // Reading php -m starts a container, so this loads on open rather than on
  // every status broadcast. The backend caches it against the image ID.
  async function load(v: string) {
    loading = true;
    error = '';
    const res = await fetchPhpExtensions(v);
    if (v !== version) return; // a newer version won the race
    report = res.report ?? null;
    error = res.error ?? res.modules_error ?? '';
    loading = false;
  }

  $effect(() => {
    load(version);
  });

  const declaredCount = $derived(
    (report?.extensions.declared?.length ?? 0) + (report?.packages.declared?.length ?? 0)
  );
  const modules = $derived(report?.modules ?? []);

  // A declared entry is only shown as present when the image really has it, so
  // nothing here advertises what the image did not load.
  function entries(set: PhpSetState): { name: string; has: boolean }[] {
    return (set.declared ?? []).map((name) => ({ name, has: (set.has ?? []).includes(name) }));
  }

  // ── Fork addition: add/remove custom extensions from the dashboard ─────────
  // The backend runs `podman build` + an FPM unit restart on every add/remove
  // (1–3 minutes), so the form blocks for the whole operation and the report is
  // refetched afterwards — the rebuilt image is the source of truth, not the
  // request we just sent.

  let extAdding = $state(false);
  let extError = $state('');
  let extName = $state('');
  let extApkDeps = $state('');
  let removingExt = $state(''); // name of the ext being removed, for a per-row spinner

  // Quick-add chips for the most commonly-requested extensions the base image
  // doesn't ship. Selecting one pre-fills the form so the user can review before
  // adding. The `apk` strings are the same Alpine packages the CLI's built-in
  // extApkDeps map uses, or what the extension's pecl docs recommend.
  interface ExtPreset {
    name: string;
    apk: string;
    why: string;
  }
  const extPresets: ExtPreset[] = [
    { name: 'imap', apk: 'imap-dev krb5-dev openssl-dev c-client', why: 'Caixa de e-mail IMAP/POP3 (Laravel Mail, Symfony Mailer com transport IMAP).' },
    { name: 'swoole', apk: 'linux-headers openssl-dev curl-dev pcre-dev', why: 'Laravel Octane (alternativa ao FrankenPHP), coroutines.' },
    { name: 'ssh2', apk: 'libssh2-dev', why: 'phpseclib auxiliar, deploy hooks, transferências SFTP nativas.' },
    { name: 'apcu', apk: '', why: 'Cache em memória userland (sessões PHP, Doctrine cache).' },
    { name: 'event', apk: 'libevent-dev openssl-dev', why: 'Event-loop nativo (workers de socket persistente).' },
    { name: 'pspell', apk: 'aspell-dev', why: 'Correção ortográfica server-side.' },
    { name: 'tidy', apk: 'tidyhtml-dev', why: 'Sanitização e validação de HTML (laravel-dompdf input cleanup).' },
    { name: 'pdo_dblib', apk: 'freetds-dev', why: 'SQL Server / Sybase via FreeTDS — alternativa ao sqlsrv da MS.' }
  ];

  function pickPreset(p: ExtPreset) {
    extName = p.name;
    extApkDeps = p.apk;
    extError = '';
  }

  async function onAddExtension() {
    const name = extName.trim();
    if (!name) {
      extError = 'Informe o nome da extensão (ex: imap, swoole, sqlsrv)';
      return;
    }
    if (!/^[a-zA-Z0-9_-]+$/.test(name)) {
      extError = 'Nome inválido — use apenas letras, dígitos, hífens e sublinhados';
      return;
    }
    extAdding = true;
    extError = '';
    const deps = extApkDeps
      .split(/[\s,]+/)
      .map((d) => d.trim())
      .filter(Boolean);
    try {
      const res = await addPhpExtension(version, name, deps);
      if (res.ok) {
        extName = '';
        extApkDeps = '';
        // Soft warning: installed, but the FPM restart failed.
        if (res.error) extError = res.error;
        await load(version);
      } else {
        extError = res.error || 'Falha ao instalar a extensão';
      }
    } finally {
      extAdding = false;
    }
  }

  async function onRemoveExtension(name: string) {
    if (!confirm(`Remover a extensão "${name}"? A imagem do PHP ${version} será reconstruída.`)) {
      return;
    }
    removingExt = name;
    extError = '';
    try {
      const res = await removePhpExtension(version, name);
      if (res.ok) {
        await load(version);
      } else {
        extError = res.error || 'Falha ao remover a extensão';
      }
    } finally {
      removingExt = '';
    }
  }

  const extBusy = $derived(extAdding || removingExt !== '');
</script>

<div class="flex flex-col h-full">
  <div class="flex-1 overflow-y-auto p-3 sm:p-5 space-y-5">
    {#if loading}
      <p class="text-sm text-gray-400">…</p>
    {:else if !report?.built}
      <EmptyState title={m.system_php_ext_notBuilt({ version })} />
    {:else}
      {#if report.needs_rebuild}
        <p
          class="text-xs text-amber-700 dark:text-amber-400 bg-amber-50 dark:bg-amber-900/20 border border-amber-200 dark:border-amber-500/30 rounded-lg px-3 py-2"
        >
          {m.system_php_ext_needsRebuild({ version })}
        </p>
      {/if}

      {#if declaredCount === 0}
        <p class="text-xs text-gray-500 dark:text-gray-400">{m.system_php_ext_none()}</p>
      {:else}
        {#each [{ label: m.system_php_ext_declared(), set: report.extensions, removable: true }, { label: m.system_php_ext_packages(), set: report.packages, removable: false }] as group (group.label)}
          {#if (group.set.declared?.length ?? 0) > 0}
            <div class="space-y-2">
              <span class="text-sm font-medium text-gray-800 dark:text-gray-200">{group.label}</span>
              <div class="flex flex-wrap gap-1.5">
                {#each entries(group.set) as e (e.name)}
                  {#if group.removable}
                    <span class="inline-flex items-center gap-1">
                      <Badge
                        tone={e.has ? 'running' : 'stopped'}
                        dot={!report.needs_rebuild}
                        title={e.has ? undefined : m.system_php_ext_cannot({ version })}
                      >
                        {e.name}
                      </Badge>
                      <button
                        type="button"
                        onclick={() => onRemoveExtension(e.name)}
                        disabled={extBusy}
                        class="text-[11px] text-red-600 dark:text-red-400 hover:underline disabled:opacity-50 disabled:no-underline"
                        title="Remover extensão (reconstrói a imagem)"
                      >
                        {removingExt === e.name ? 'removendo…' : '×'}
                      </button>
                    </span>
                  {:else}
                    <Badge
                      tone={e.has ? 'running' : 'stopped'}
                      dot={!report.needs_rebuild}
                      title={e.has ? undefined : m.system_php_ext_cannot({ version })}
                    >
                      {e.name}
                    </Badge>
                  {/if}
                {/each}
              </div>
            </div>
          {/if}
        {/each}
        <p class="text-xs text-gray-500 dark:text-gray-400 -mt-2">{m.system_php_ext_manageHelp()}</p>
      {/if}

      <div class="space-y-2">
        <span class="text-sm font-medium text-gray-800 dark:text-gray-200">
          {m.system_php_ext_modules({ count: modules.length })}
        </span>
        <p class="text-xs text-gray-500 dark:text-gray-400">{m.system_php_ext_modulesHelp({ version })}</p>
        <div class="flex flex-wrap gap-1.5">
          {#each modules as mod (mod)}
            <Badge tone="neutral">{mod}</Badge>
          {/each}
        </div>
      </div>

      {#if error}
        <p class="text-xs text-red-500">{error}</p>
      {/if}
    {/if}

    <!--
      Fork addition: the dashboard equivalent of `lerd php:ext add/remove`. The
      declared set is global (it reaches every image); this rebuilds and bounces
      the version whose tab you are on.
    -->
    <div class="space-y-2 border-t border-gray-100 dark:border-lerd-border pt-4">
      <span class="text-sm font-medium text-gray-800 dark:text-gray-200">Adicionar extensão</span>

      <div>
        <p class="text-[10px] font-semibold text-gray-400 uppercase tracking-wider mb-1.5">Exemplos rápidos</p>
        <div class="flex flex-wrap gap-1.5">
          {#each extPresets as p (p.name)}
            <button
              type="button"
              onclick={() => pickPreset(p)}
              disabled={extBusy}
              title={p.why + (p.apk ? '  ·  apk: ' + p.apk : '  ·  sem deps Alpine extras')}
              class="font-mono text-[11px] px-2 py-0.5 rounded-full border border-gray-200 dark:border-lerd-border bg-white dark:bg-white/5 text-gray-700 dark:text-gray-200 hover:bg-emerald-50 hover:border-emerald-300 dark:hover:bg-emerald-500/10 dark:hover:border-emerald-500/50 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              + {p.name}
            </button>
          {/each}
        </div>
      </div>

      <div class="grid grid-cols-1 sm:grid-cols-[180px_1fr_auto] gap-2">
        <div class="flex flex-col gap-1">
          <label for="ext-name-{version}" class="text-[10px] font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider">Nome da extensão</label>
          <input
            id="ext-name-{version}"
            type="text"
            placeholder="imap, swoole, ssh2…"
            bind:value={extName}
            disabled={extBusy}
            class="font-mono text-xs px-2.5 py-1.5 bg-white dark:bg-lerd-dark-2 border border-gray-200 dark:border-lerd-border rounded text-gray-900 dark:text-gray-100 placeholder:text-gray-400 dark:placeholder:text-gray-500 focus:outline-none focus:ring-1 focus:ring-emerald-500 focus:border-emerald-500 disabled:opacity-50"
          />
        </div>
        <div class="flex flex-col gap-1">
          <label for="ext-apk-{version}" class="text-[10px] font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider">
            Pacotes Alpine (opcional) <span class="font-normal normal-case text-gray-400">— separe por espaço ou vírgula</span>
          </label>
          <input
            id="ext-apk-{version}"
            type="text"
            placeholder="imap-dev krb5-dev openssl-dev"
            bind:value={extApkDeps}
            disabled={extBusy}
            class="font-mono text-xs px-2.5 py-1.5 bg-white dark:bg-lerd-dark-2 border border-gray-200 dark:border-lerd-border rounded text-gray-900 dark:text-gray-100 placeholder:text-gray-400 dark:placeholder:text-gray-500 focus:outline-none focus:ring-1 focus:ring-emerald-500 focus:border-emerald-500 disabled:opacity-50"
          />
        </div>
        <div class="flex flex-col gap-1">
          <span class="text-[10px] font-semibold text-transparent uppercase tracking-wider select-none">·</span>
          <DetailButton
            tone="success"
            onclick={onAddExtension}
            disabled={extBusy || !extName.trim()}
            loading={extAdding}
            title="Instala via pecl/docker-php-ext-install, reconstrói a imagem e reinicia o FPM (1–3min)"
          >{extAdding ? 'Reconstruindo…' : 'Adicionar'}</DetailButton>
        </div>
      </div>

      {#if extAdding}
        <p class="text-[10px] text-emerald-600 dark:text-emerald-400 leading-relaxed flex items-center gap-1.5">
          <span class="inline-block w-1.5 h-1.5 bg-emerald-500 rounded-full animate-pulse"></span>
          Reconstruindo imagem <span class="font-mono">lerd-php{version.replace('.', '')}-fpm:local</span> com a nova extensão — pode levar 1 a 3 minutos…
        </p>
      {:else}
        <p class="text-[10px] text-gray-400 leading-relaxed">
          Equivalente a <code class="font-mono text-gray-500 dark:text-gray-400">lerd php:ext add &lt;ext&gt; --apk-deps "&lt;pacotes&gt;"</code>.
          A imagem é reconstruída e o container reinicia ao final.
        </p>
      {/if}

      {#if extError}
        <div class="text-xs font-medium text-red-600 dark:text-red-400 bg-red-50 dark:bg-red-500/10 rounded-lg px-3 py-2 break-words">
          {extError}
        </div>
      {/if}
    </div>
  </div>
</div>
