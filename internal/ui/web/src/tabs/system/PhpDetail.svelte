<script lang="ts">
  import StatusPill from '$components/StatusPill.svelte';
  import ButtonMenu, { type ButtonMenuAction } from '$components/ButtonMenu.svelte';
  import DetailTabs, { type TabItem } from '$components/DetailTabs.svelte';
  import LogViewer from '$components/LogViewer.svelte';
  import DetailButton from '$components/DetailButton.svelte';
  import InfoRow from '$components/InfoRow.svelte';
  import PhpIniTab from './PhpIniTab.svelte';
  import { status, loadStatus } from '$stores/status';
  import { setDefaultPhp, startPhp, stopPhp } from '$stores/phpVersions';
  import {
    phpExtensions,
    loadPhpExtensions,
    addPhpExtension,
    removePhpExtension
  } from '$stores/phpExtensions';
  import { sites, sitesByPhp } from '$stores/sites';
  import { xdebugOn, xdebugOff, XDEBUG_MODES, type XdebugMode } from '$stores/xdebug';
  import { goToTab } from '$stores/route';
  import { openPhpRemoveModal } from '$stores/modals';
  import { m } from '../../paraglide/messages.js';

  interface Props {
    version: string;
  }
  let { version }: Props = $props();

  // Reload custom extensions whenever the PHP tab swaps version.
  $effect(() => {
    loadPhpExtensions(version);
  });

  const customExts = $derived($phpExtensions[version] ?? []);
  // (continues — extension management helpers below)

  let extAdding = $state(false);
  let extError = $state('');
  let extName = $state('');
  let extApkDeps = $state('');
  let removingExt = $state(''); // name of ext currently being removed (for per-row spinner)

  // Quick-add chips for the most commonly-requested PHP extensions that the
  // base lerd image doesn't ship. Selecting one pre-fills the form so the
  // user can review/edit before hitting Adicionar. The `apk` strings are
  // the same Alpine packages the CLI's built-in extApkDeps map uses (or what
  // pecl docs recommend for the extension), validated against alpine v3.x
  // package names.
  interface ExtPreset {
    name: string;
    apk: string;
    why: string; // short tooltip explaining where this is typically needed
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
        if (res.error) {
          // Soft warning (installed but FPM restart failed, etc.)
          extError = res.error;
        }
      } else {
        extError = res.error || 'Falha ao instalar a extensão';
      }
    } finally {
      extAdding = false;
    }
  }

  async function onRemoveExtension(name: string) {
    if (!confirm(`Remover a extensão "${name}" do PHP ${version}? A imagem será reconstruída.`)) {
      return;
    }
    removingExt = name;
    extError = '';
    try {
      const res = await removePhpExtension(version, name);
      if (!res.ok) {
        extError = res.error || 'Falha ao remover a extensão';
      }
    } finally {
      removingExt = '';
    }
  }

  const isDefault = $derived($status.php_default === version);
  const siteCount = $derived($sitesByPhp.get(version) ?? 0);
  const fpm = $derived($status.php_fpms.find((f) => f.version === version));
  const running = $derived(Boolean(fpm?.running));
  const xdebugEnabled = $derived(Boolean(fpm?.xdebug_enabled));
  const xdebugMode = $derived<XdebugMode>((fpm?.xdebug_mode as XdebugMode) || 'debug');
  const container = $derived('lerd-php' + version.replace('.', '') + '-fpm');
  const sitesUsing = $derived($sites.filter((s) => s.php_version === version));

  let defaultBusy = $state(false);
  let fpmBusy = $state(false);
  let xdebugBusy = $state(false);
  let xdebugMenuOpen = $state(false);
  let xdebugRootEl: HTMLDivElement | undefined = $state();

  // The parent (PhpPage) no longer wraps us in {#key version}; reset
  // per-version transient state when the version prop changes so a stale
  // open xdebug menu doesn't leak across tabs.
  $effect(() => {
    version;
    xdebugMenuOpen = false;
  });

  function closeXdebugMenu() {
    xdebugMenuOpen = false;
  }

  function onXdebugDocClick(e: MouseEvent) {
    if (!xdebugRootEl) return;
    if (!xdebugRootEl.contains(e.target as Node)) closeXdebugMenu();
  }

  function onXdebugDocKey(e: KeyboardEvent) {
    if (e.key === 'Escape') closeXdebugMenu();
  }

  $effect(() => {
    if (!xdebugMenuOpen) return;
    document.addEventListener('mousedown', onXdebugDocClick);
    document.addEventListener('keydown', onXdebugDocKey);
    return () => {
      document.removeEventListener('mousedown', onXdebugDocClick);
      document.removeEventListener('keydown', onXdebugDocKey);
    };
  });

  type TabId = 'logs' | 'sites' | 'config';
  let active = $state<TabId>('logs');
  const tabs = $derived<TabItem<TabId>[]>([
    { id: 'logs', label: m.services_tabs_logs(), hidden: !running },
    { id: 'sites', label: m.system_php_sites() },
    { id: 'config', label: m.system_php_iniTab() }
  ]);

  $effect(() => {
    if (active === 'logs' && !running) active = 'sites';
  });

  // Log section tabs at the bottom of the panel.
  type LogTab = 'all' | 'errors';
  let logTab = $state<LogTab>('all');
  const logTabs: TabItem<LogTab>[] = [
    { id: 'all', label: m.system_php_logsAll() },
    { id: 'errors', label: m.system_php_logsErrors() }
  ];

  // Errors filter: any line that looks like a PHP runtime error / FPM
  // diagnostic. Matches both user-code errors emitted by the engine
  // (Fatal/Parse/Warning/Notice/Deprecated) and FPM's own diagnostics
  // (ERROR/ALERT/WARNING in square brackets, plus the "exception" keyword
  // Laravel and Monolog drop when forwarding handled exceptions to stderr).
  const errorLineRegex = /(PHP (Fatal|Parse|Warning|Notice|Deprecated|Recoverable)|\[(?:ERROR|ALERT|EMERGENCY|CRITICAL|WARNING)\]|Uncaught |Stack trace:|exception\s*"|"level":"(?:error|critical|alert|emergency)")/i;
  function isErrorLine(line: string): boolean {
    return errorLineRegex.test(line);
  }
  function highlightLogLine(line: string): string | null {
    if (/PHP (Fatal|Parse) error|Uncaught |\[(?:EMERGENCY|ALERT|CRITICAL)\]|"level":"(?:critical|alert|emergency)"/i.test(line)) {
      return 'text-red-600 dark:text-red-400';
    }
    if (/PHP (Warning|Recoverable)|\[ERROR\]|"level":"error"/i.test(line)) {
      return 'text-orange-600 dark:text-orange-400';
    }
    if (/PHP (Notice|Deprecated)|\[WARNING\]/i.test(line)) {
      return 'text-amber-600 dark:text-amber-400';
    }
    return null;
  }

  async function onSetDefault() {
    defaultBusy = true;
    try {
      await setDefaultPhp(version);
      await loadStatus();
    } finally {
      defaultBusy = false;
    }
  }

  // fpmAction = '' idle, 'starting' | 'stopping' while in flight. Used to
  // pick the toast message + the inline status badge under the toggle so
  // the user knows the request is being processed (the systemd unit may
  // take 1-2s to flip state on slow containers and the click feels dead
  // without feedback).
  let fpmAction = $state<'' | 'starting' | 'stopping'>('');
  let fpmActionError = $state('');

  async function onToggleFpm() {
    fpmBusy = true;
    fpmAction = running ? 'stopping' : 'starting';
    fpmActionError = '';
    try {
      const ok = await (running ? stopPhp(version) : startPhp(version));
      // loadStatus polls the snapshot endpoint until the running flag
      // actually changes, so the user sees the dot/pill flip after the
      // server confirms the unit transitioned.
      await loadStatus();
      if (!ok) {
        fpmActionError = fpmAction === 'starting'
          ? `Falha ao iniciar PHP ${version}-fpm (veja journalctl --user -u lerd-php${version.replace('.', '')}-fpm)`
          : `Falha ao parar PHP ${version}-fpm`;
      }
    } finally {
      fpmBusy = false;
      fpmAction = '';
    }
  }

  async function onToggleXdebug() {
    xdebugBusy = true;
    try {
      if (xdebugEnabled) {
        await xdebugOff(version);
      } else {
        await xdebugOn(version, xdebugMode);
      }
      await loadStatus();
    } finally {
      xdebugBusy = false;
    }
  }

  async function onSetXdebugMode(e: Event) {
    const mode = (e.target as HTMLSelectElement).value as XdebugMode;
    if (mode === xdebugMode) return;
    xdebugBusy = true;
    try {
      await xdebugOn(version, mode);
      await loadStatus();
    } finally {
      xdebugBusy = false;
    }
  }

  const headerBusy = $derived(fpmBusy || defaultBusy);

  const headerActions = $derived.by<ButtonMenuAction[]>(() => {
    if (isDefault) return [];
    const acts: ButtonMenuAction[] = [];
    if (running) {
      acts.push({
        id: 'stop',
        icon: stopIcon,
        label: m.common_stop(),
        title: siteCount > 0 ? m.system_php_stopWarn({ count: siteCount }) : m.system_php_stopTitle(),
        disabled: fpmBusy,
        onclick: onToggleFpm
      });
    } else {
      acts.push({
        id: 'start',
        tone: 'success',
        icon: startIcon,
        label: m.common_start(),
        title: m.system_php_startTitle(),
        disabled: fpmBusy,
        onclick: onToggleFpm
      });
    }
    acts.push({
      id: 'set-default',
      icon: starIcon,
      label: m.system_php_setDefault(),
      disabled: defaultBusy,
      onclick: onSetDefault
    });
    acts.push({
      id: 'remove',
      tone: 'danger',
      icon: trashIcon,
      label: m.common_remove(),
      title: siteCount > 0 ? m.system_php_removeWarn({ count: siteCount }) : m.system_php_removeTitle(),
      onclick: () => openPhpRemoveModal({ version, siteCount })
    });
    return acts;
  });
</script>

{#snippet startIcon()}
  <svg class="w-3.5 h-3.5" fill="currentColor" viewBox="0 0 24 24"><path d="M8 5v14l11-7z"/></svg>
{/snippet}
{#snippet stopIcon()}
  <svg class="w-3.5 h-3.5" fill="currentColor" viewBox="0 0 24 24"><rect x="6" y="6" width="12" height="12" rx="1"/></svg>
{/snippet}
{#snippet starIcon()}
  <svg class="w-3.5 h-3.5" fill="currentColor" viewBox="0 0 20 20"><path d="M10 1.5l2.6 5.27 5.82.85-4.21 4.1.99 5.78L10 14.77l-5.2 2.73.99-5.78L1.58 7.62l5.82-.85L10 1.5z"/></svg>
{/snippet}
{#snippet trashIcon()}
  <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"/></svg>
{/snippet}

<div
  class="flex flex-wrap items-center justify-between gap-y-2 px-3 py-3 border-b border-gray-100 dark:border-lerd-border shrink-0"
>
  <div class="flex items-center gap-3 flex-wrap">
    <span class="font-semibold text-gray-900 dark:text-white text-base">PHP {version}</span>
    <StatusPill tone={running ? 'ok' : 'muted'} label={running ? m.common_running() : m.common_stopped()} />
    {#if siteCount > 0}
      <span class="text-xs text-gray-400 dark:text-gray-500">
        {siteCount} {siteCount === 1 ? m.common_site() : m.common_sites()}
      </span>
    {/if}
  </div>
  <div class="flex items-center gap-2 flex-wrap">
    <div bind:this={xdebugRootEl} class="relative inline-flex">
      <button
        type="button"
        onclick={onToggleXdebug}
        disabled={xdebugBusy}
        aria-pressed={xdebugEnabled}
        title={xdebugEnabled ? 'Disable Xdebug' : 'Enable Xdebug'}
        class="inline-flex items-center gap-1.5 px-3 py-1.5 border border-gray-200 dark:border-lerd-border transition-colors text-xs font-medium text-gray-700 dark:text-gray-200 disabled:opacity-50 {xdebugEnabled
          ? 'rounded-l-lg border-r-0 bg-emerald-50/60 dark:bg-emerald-900/15 hover:bg-emerald-50 dark:hover:bg-emerald-900/25'
          : 'rounded-lg bg-white dark:bg-lerd-card hover:bg-gray-50 dark:hover:bg-white/5'}"
      >
        {#if xdebugBusy}
          <svg class="w-2.5 h-2.5 animate-spin text-amber-500" fill="none" viewBox="0 0 24 24">
            <circle class="opacity-30" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
            <path class="opacity-90" fill="currentColor" d="M4 12a8 8 0 018-8v8H4z" />
          </svg>
        {:else}
          <span class="shrink-0 w-2 h-2 rounded-full {xdebugEnabled ? 'bg-emerald-500' : 'border border-gray-300 dark:border-gray-600 bg-transparent'}"></span>
        {/if}
        <span>{m.system_php_xdebug()}</span>
      </button>
      {#if xdebugEnabled}
        <button
          type="button"
          onclick={() => (xdebugMenuOpen = !xdebugMenuOpen)}
          disabled={xdebugBusy}
          aria-haspopup="menu"
          aria-expanded={xdebugMenuOpen}
          title={m.system_php_xdebugModeTitle()}
          class="inline-flex items-center gap-1 px-3 py-1.5 rounded-r-lg border border-gray-200 dark:border-lerd-border transition-colors text-xs font-medium text-gray-700 dark:text-gray-200 bg-emerald-50/60 dark:bg-emerald-900/15 hover:bg-emerald-50 dark:hover:bg-emerald-900/25 disabled:opacity-50"
        >
          <span class="font-mono">{xdebugMode}</span>
          <svg class="w-3 h-3 transition-transform {xdebugMenuOpen ? 'rotate-180' : ''}" fill="none" stroke="currentColor" stroke-width="2.5" viewBox="0 0 24 24" stroke-linecap="round" stroke-linejoin="round">
            <polyline points="6 9 12 15 18 9"/>
          </svg>
        </button>
        {#if xdebugMenuOpen}
          <div
            role="menu"
            class="absolute right-0 top-full mt-1 z-50 min-w-40 rounded-xl bg-white dark:bg-lerd-card border border-gray-200 dark:border-lerd-border shadow-xl py-1"
          >
            {#each XDEBUG_MODES as mode (mode)}
              {@const selected = mode === xdebugMode}
              <button
                type="button"
                role="menuitem"
                onclick={() => {
                  xdebugMenuOpen = false;
                  onSetXdebugMode({ target: { value: mode } } as unknown as Event);
                }}
                class="w-full text-left px-3 py-1.5 text-xs font-mono hover:bg-gray-50 dark:hover:bg-white/5 transition-colors {selected ? 'text-lerd-red font-semibold' : 'text-gray-700 dark:text-gray-200'}"
              >
                {mode}
              </button>
            {/each}
          </div>
        {/if}
      {/if}
    </div>
    <ButtonMenu actions={headerActions} busy={headerBusy} />
  </div>
</div>

<DetailTabs {tabs} {active} onchange={(id) => (active = id)} />
{#if active === 'logs' && running}
  <DetailTabs tabs={logTabs} active={logTab} onchange={(id) => (logTab = id)} />
  <LogViewer
    path={'/api/logs/' + container}
    highlight={highlightLogLine}
    filter={logTab === 'errors' ? isErrorLine : undefined}
    emptyLabel={logTab === 'errors' ? m.system_php_logsErrorsEmpty() : undefined}
  />
{:else if active === 'sites'}
  <div class="px-3 sm:px-5 py-3 shrink-0">
    {#if sitesUsing.length === 0}
      <p class="text-sm text-gray-400">{m.system_noSitesUsingPhp({ version })}</p>
    {:else}
      <div class="flex flex-wrap gap-2">
        {#each sitesUsing as s (s.domain)}
          <button
            onclick={() => goToTab('sites', s.domain)}
            class="inline-flex items-center gap-1.5 text-xs font-medium bg-gray-100 dark:bg-white/5 hover:bg-gray-200 dark:hover:bg-white/10 border border-gray-200 dark:border-lerd-border text-gray-700 dark:text-gray-300 rounded-full px-2.5 py-1 transition-colors"
          >
            <span class="w-1.5 h-1.5 rounded-full shrink-0 {s.fpm_running ? 'bg-emerald-500' : 'bg-gray-400'}"></span>
            {s.domain}
          </button>
        {/each}
      </div>
    {/if}
  </div>
{:else if active === 'config'}
  <div class="px-3 sm:px-5 py-3 space-y-4 overflow-y-auto shrink-0">
    <PhpIniTab {version} />

    <InfoRow label={m.system_container()} value={container} />

    {#if fpmAction}
      <div class="text-[11px] text-emerald-600 dark:text-emerald-400 bg-emerald-50 dark:bg-emerald-500/10 rounded-lg px-3 py-2 flex items-center gap-2">
        <span class="inline-block w-1.5 h-1.5 bg-emerald-500 rounded-full animate-pulse"></span>
        {fpmAction === 'starting'
          ? `Iniciando container ${container}… (systemctl --user start)`
          : `Parando container ${container}… (systemctl --user stop)`}
      </div>
    {/if}

    {#if fpmActionError}
      <div class="text-[11px] text-red-600 dark:text-red-400 bg-red-50 dark:bg-red-500/10 rounded-lg px-3 py-2 break-words">
        {fpmActionError}
      </div>
    {/if}

    <!--
      Extensões customizadas — gerencia o equivalente de `lerd php:ext add/remove`
      pelo dashboard. O backend roda `podman build` + restart do FPM unit a cada
      add/remove (1–3 minutos), por isso o spinner fica de pé durante toda a
      operação e a UI bloqueia novos cliques.
    -->
    <div>
      <div class="flex items-center justify-between mb-2">
        <p class="text-xs font-semibold text-gray-400 uppercase tracking-wider">Extensões customizadas</p>
        <span class="text-[10px] text-gray-400">{customExts.length} configurada{customExts.length === 1 ? '' : 's'}</span>
      </div>

      {#if customExts.length === 0}
        <div class="text-xs text-gray-500 dark:text-gray-400 mb-3 leading-relaxed">
          Nenhuma extensão extra além das <strong class="text-gray-700 dark:text-gray-300">31 já compiladas</strong> na imagem
          (<span class="font-mono text-[10px]">oci8, redis, imagick, mongodb, memcached, amqp, igbinary, xdebug, gd, intl, zip, pdo_*, soap, xsl, ldap, pcntl, exif, bcmath, mbstring, gmp, bz2, sysv*, sockets, …</span>).
          Use o formulário abaixo para instalar uma extensão extra via PECL ou <span class="font-mono">docker-php-ext-install</span>.
        </div>
      {:else}
        <div class="space-y-1.5 mb-3">
          {#each customExts as ext (ext.name)}
            <div class="flex items-center justify-between gap-2 px-2.5 py-1.5 bg-gray-50 dark:bg-white/5 border border-gray-200 dark:border-lerd-border rounded text-xs">
              <div class="flex items-center gap-2 min-w-0">
                <span class="font-mono font-medium text-gray-700 dark:text-gray-200 shrink-0">{ext.name}</span>
                {#if ext.apk_deps && ext.apk_deps.length > 0}
                  <span class="text-gray-400 truncate" title="Pacotes Alpine">apk: {ext.apk_deps.join(' ')}</span>
                {/if}
              </div>
              <button
                onclick={() => onRemoveExtension(ext.name)}
                disabled={removingExt === ext.name || extAdding}
                class="text-red-600 dark:text-red-400 hover:underline disabled:opacity-50 disabled:no-underline shrink-0"
                title="Remover extensão (reconstrói a imagem)"
              >
                {removingExt === ext.name ? 'removendo…' : 'remover'}
              </button>
            </div>
          {/each}
        </div>
      {/if}

      <!--
        Chips de exemplo — 1-clique pré-preenche os campos com nome + apk deps
        prontos. Cobre as extensões mais comuns que faltam na base do lerd.
      -->
      <div class="mb-3">
        <p class="text-[10px] font-semibold text-gray-400 uppercase tracking-wider mb-1.5">Exemplos rápidos</p>
        <div class="flex flex-wrap gap-1.5">
          {#each extPresets as p (p.name)}
            <button
              type="button"
              onclick={() => pickPreset(p)}
              disabled={extAdding || removingExt !== ''}
              title={p.why + (p.apk ? '  ·  apk: ' + p.apk : '  ·  sem deps Alpine extras')}
              class="font-mono text-[11px] px-2 py-0.5 rounded-full border border-gray-200 dark:border-lerd-border bg-white dark:bg-white/5 text-gray-700 dark:text-gray-200 hover:bg-emerald-50 hover:border-emerald-300 dark:hover:bg-emerald-500/10 dark:hover:border-emerald-500/50 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              + {p.name}
            </button>
          {/each}
        </div>
      </div>

      <div class="space-y-2">
        <div class="grid grid-cols-1 sm:grid-cols-[180px_1fr_auto] gap-2">
          <div class="flex flex-col gap-1">
            <label for="ext-name-{version}" class="text-[10px] font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider">Nome da extensão</label>
            <input
              id="ext-name-{version}"
              type="text"
              placeholder="imap, swoole, ssh2…"
              bind:value={extName}
              disabled={extAdding || removingExt !== ''}
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
              disabled={extAdding || removingExt !== ''}
              class="font-mono text-xs px-2.5 py-1.5 bg-white dark:bg-lerd-dark-2 border border-gray-200 dark:border-lerd-border rounded text-gray-900 dark:text-gray-100 placeholder:text-gray-400 dark:placeholder:text-gray-500 focus:outline-none focus:ring-1 focus:ring-emerald-500 focus:border-emerald-500 disabled:opacity-50"
            />
          </div>
          <div class="flex flex-col gap-1">
            <span class="text-[10px] font-semibold text-transparent uppercase tracking-wider select-none">·</span>
            <DetailButton
              tone="success"
              onclick={onAddExtension}
              disabled={extAdding || removingExt !== '' || !extName.trim()}
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
            Equivalente a <code class="font-mono text-gray-500 dark:text-gray-400">lerd php:ext add &lt;ext&gt; {version} --apk-deps "&lt;pacotes&gt;"</code>.
            A imagem é reconstruída e o container reinicia ao final.
          </p>
        {/if}
      </div>

      {#if extError}
        <div class="mt-2 text-xs font-medium text-red-600 dark:text-red-400 bg-red-50 dark:bg-red-500/10 rounded-lg px-3 py-2 break-words">
          {extError}
        </div>
      {/if}
    </div>
  </div>
{/if}
