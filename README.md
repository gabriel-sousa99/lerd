# Lerd Oracle Edition

> Fork do [`lerd-env/lerd`](https://github.com/lerd-env/lerd) com **suporte a
> Oracle Database embutido em todas as imagens PHP** — Oracle Instant
> Client 21.18 (LTS) + `oci8` + memcached + amqp já compilados, prontos
> para PHP 5.6 → 8.5. Drop-in replacement: todo comando `lerd` existente
> funciona igual.

> [!IMPORTANT]
> Este fork mantém o mesmo binário `lerd` (compatibilidade total) e
> aponta o auto-update para **este** repositório. Releases seguem o
> esquema `vX.Y.Z-oracle.N`.

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/platform-Linux%20%7C%20macOS%20%7C%20WSL2%20(beta)-lightgrey)]()
[![Docs](https://img.shields.io/badge/docs-lerd.sh-blue)](https://lerd.sh)
[![Reddit](https://img.shields.io/badge/Reddit-r%2Flerd-ff2d20?logo=reddit)](https://reddit.com/r/lerd)
[![Discord](https://img.shields.io/badge/Discord-Join-5865F2?logo=discord&logoColor=white)](https://discord.gg/5JK54s7xCC)

[![Fork base](https://img.shields.io/badge/forked%20from-lerd--env%2Flerd%20v1.33.1-blue)](https://github.com/lerd-env/lerd)
[![Oracle Instant Client](https://img.shields.io/badge/Oracle%20Instant%20Client-21.18-red)]()
[![PHP](https://img.shields.io/badge/PHP-5.6%20%E2%80%93%208.5-777BB4)]()

---

## Sumário

- [O que muda em relação ao upstream](#o-que-muda-em-relação-ao-upstream)
- [Abrangência](#abrangência)
- [Instalação](#instalação) · [WSL2](docs/wsl2.md)
- [Primeiro uso](#primeiro-uso)
- [Trabalhando com Oracle](#trabalhando-com-oracle)
- [Serviços](#serviços) · [Rebuild PHP](#rebuild-de-imagens-php) · [Atualização](#atualização) · [Desinstalação](#desinstalação)
- [Proxies para projetos não-PHP](#proxies-para-projetos-não-php)
- [Diagnóstico](#diagnóstico) · [Dashboard](docs/dashboard.md) · [Comandos](#lista-de-comandos-úteis)

---

### Sites, domains and TLS

- 🌐 **Automatic `.test` domains.** One command gives a project a hostname and TLS that reissues before it expires, with no dnsmasq, no system resolver tweak and no sudo for the DNS bits. You can [opt out of lerd-managed DNS](https://lerd.sh/features/dns) for `*.localhost`, or toggle it later with `dns:enable` / `dns:disable` / `dns:repair`.

- 🔗 **Site groups.** Group related sites so a main site owns a base domain and the rest occupy its subdomains, with a shared or separate database per secondary.

- 🧱 **Host-proxy sites.** Run a Node, Python, Go or any non-PHP dev server on the host and have nginx serve it at a `.test` domain with HTTPS, git worktrees included. A wedged dev server can be bounced from the site header without reaching for a terminal.

- 🌳 **First-class git worktrees.** Auto-detected branch domains, per-worktree PHP and Node versions, optional database isolation, wildcard cert SANs and a per-branch Vite worker. A bare `git worktree add` from any tool is provisioned automatically, and `lerd worktree wait` blocks until the tree is ready.

- 🌍 **Share a site.** On your LAN with a stable port and a QR code, or publicly through ngrok, cloudflared, Expose, Pinggy, serveo or localhost.run. Set a base domain once and every share keeps the same URL between runs, through a tunnel service or the reverse proxy you already run.

- 🎨 **Dev servers on the site's own domain.** A running Vite serves its assets and its hot-reload socket under the site's `.test` hostname instead of advertising `localhost:5173`, so a shared, LAN-opened or worktree page arrives styled. Nothing in the project is edited and nothing is declared per framework.

### PHP, Node and runtimes

- 🐘 **Per-project PHP version.** 8.1 to 8.5, plus a frozen 7.4 / 8.0 legacy tier for projects on the old stack, switched with one click. Custom extensions and Alpine packages are declared once and applied to every image lerd builds.

- ⚡ **FrankenPHP runtime.** Per site, as an alternative to shared PHP-FPM, with Laravel Octane and Symfony Runtime worker mode.

- 📦 **Node.js isolation.** Node 22 or 24 per project, through the bundled fnm or an nvm you already have, switchable from the dashboard. Or **bun** as the JS runtime on the host and, opt-in, inside the container.

- 🪄 **No per-framework setup.** Workers, env values and the nginx vhost are configured for you when you link a project. "Env" means whatever file your framework actually reads: a `.env`, WordPress's `wp-config.php`, Magento's `env.php` or Drupal's `settings.php`, written in place.

- 🧩 **Framework store.** Community definitions for Laravel, Symfony, WordPress, Drupal, Magento, CakePHP, CodeIgniter, Statamic and Tempest, with versioned auto-detection back to the majors still on PHP 7.4. One published tomorrow arrives without a new lerd release.

### Services and databases

- 🗄️ **One-click services.** MySQL, PostgreSQL, Redis, Meilisearch, RustFS, Mailpit, Reverb, OpenSearch and more, the default stack built in and every add-on from a store that updates without a lerd release. Create, drop, snapshot, export and import databases from the service page.

- 🔌 **Host tools that reach the container.** `psql`, `mysql`, `pg_dump` and friends run on your host against lerd's engines with no client installed and no port to remember. Point an IDE's phpstan, php-cs-fixer or phpcs at the same shims and they run in the project's PHP container.

- 🧷 **IDE database wiring** for JetBrains. A project gets one data source pointed at its own lerd database on the host port it actually answers on, written on link and refreshed as the project's database changes, leaving every data source lerd doesn't own untouched.

### Debugging and performance

- 🛰️ **Debug window.** Intercepts every `dump()` / `dd()` and streams it to the dashboard, TUI, MCP and `lerd dump tail`, scoped per site and per worktree branch. The same window captures SQL with N+1 and slow-query detection, plus mail, views, events, queued jobs and outgoing HTTP, on Laravel and Symfony.

- 🔥 **[SPX](https://github.com/NoiseByNorthwest/php-spx) profiler** with one-click on/off. Every PHP-FPM request becomes a flame graph viewable in a same-origin Profiler view in the dashboard, with no FPM restart and no code changes, and `lerd profile run` profiles a one-shot artisan or CLI command.

- 📈 **Request timing analytics.** A durable per-site view of typical and p95 response times, throughput, error rate, and the slowest routes ranked by recent p95 with one-click profiling. Agents get the same signal over MCP with `route_timing` and `optimize_route`.

- 🧪 **Tinker tab.** An in-browser PHP REPL per site with project-aware autocomplete, hover and diagnostics powered by [phpantom_lsp](https://github.com/PHPantom-dev/phpantom_lsp), so your models and Builder chains resolve as you type. Works on Laravel, Symfony and any composer project.

### Interfaces

- 🖥️ **Built-in Web UI.** Sites and services dashboards, live widgets, a global Cmd+K command palette, and install/remove of PHP and Node versions, in fourteen languages. Reachable from another machine behind credentials, with the actions that touch the host staying local until you grant them.

- ✨ **Start a project from the dashboard.** The `+` in Sites scaffolds a project from the framework store or links one you already have, asks what `lerd init` asks, then runs composer and the JS build in the modal. Close the tab mid-install and it picks back up.

- 📚 **The documentation, offline.** Every page ships inside the binary, searchable and rendered in the dashboard, so the one moment you most need the docs, a machine with no internet, is not the moment they stop working. `lerd man` reads the same pages in the terminal.

- 💻 **Terminal dashboard** (`lerd tui`). A btop-style TUI with live status, site detail pane, inline domain and version editing, shell drop-in, log tailing, and filter/sort, the same operations surface as the web UI, for tmux and SSH workflows.

- ✏️ **Edit config in the browser.** Per-site and global nginx, `php.ini` with the version's own file and the shared scope side by side, `.env` files, and database/service runtime tuning, each validated (`nginx -t` where it applies), with timestamped backups and one-click restore.

- 📋 **Live logs** for PHP-FPM, Queue, Schedule and Reverb, per site, rendered in the colour the tool actually emits (artisan, composer, vite, pest) and with a button that hands any log to a real terminal so a long tail survives closing the tab.

- 🔔 **Notifications** for the things worth interrupting you, delivered to open dashboards, to subscribed browsers over Web Push, or to your desktop's native notification daemon. Every one also lands in the dashboard's sidebar bell, which keeps the last 50 with an unread count across reloads.

- 🤖 **MCP server.** Let AI assistants (Claude Code, Cursor, JetBrains Junie, Codex CLI, Gemini CLI, GitHub Copilot, Google Antigravity, Windsurf) manage your environment directly.

### Health and upkeep

- 🧰 **Environment doctor** (`lerd doctor`). Checks the host lerd itself depends on and repairs what it safely can with `--fix`: missing directories, linger, a missing PHP image, the DNS wiring. Anything needing sudo is printed as a command and never run for you, and `--dry-run` shows it first.

- 🩺 **Site doctor.** Framework-agnostic health checks (env drift, application key, composer and node state, security audits, database presence, PHP version range) plus extra checks for your framework, each with a one-click fix, from the web UI, the TUI, `lerd site:doctor` and MCP.

- ⚒️ **Worker self-heal.** Failed queue, schedule, horizon, reverb and stripe workers are surfaced everywhere (CLI, dashboard banner, TUI, MCP) and recovered with one click or `lerd worker heal`. A worker that keeps failing can be stopped from the same banner rather than restarted into the same wall.

- 💾 **Nothing destructive without a way back.** A `service remove --purge` or a `reinstall --reset-data` snapshots every database first, while the data is still where the engine expects it, so recovery is an ordinary `db:restore -A`. Each engine declares how long it gets to shut down.

- 💤 **Idle-suspend.** Activity-driven suspension of a site's workers (queue, schedule, horizon, reverb, stripe, Vite) after a configurable idle timeout, resumed on the next request, CLI command, MCP call or file save, with per-site pinning.

- 📌 **Pinned host tools.** Composer, fnm and mkcert are pinned behind a published manifest rather than whatever `releases/latest` served that day, so an upstream release cannot break a fresh install overnight, and the System page reports each against its pin and applies the update on the card that flagged it.

- 🔒 **Rootless and daemonless.** Podman-native, no Docker required, dual-stack IPv4 + IPv6.

## O que muda em relação ao upstream

> Recursos herdados do upstream (`lerd-env/lerd`), todos presentes nesta fork:
>
> - 🌐 **Domínios `.test` automáticos** com TLS em um comando, ou [desative o DNS gerenciado pelo lerd](https://lerd.sh/features/dns/) e use `*.localhost` (sem dnsmasq, sem mexer no resolver do sistema, sem sudo na parte de DNS); o DNS é ciente de VPN e re-sincroniza os resolvers dos containers em menos de um segundo quando um túnel conecta/desconecta
> - 🐘 **Versão de PHP por projeto** (8.1–8.5, mais uma faixa legacy congelada 7.4 / 8.0 para projetos em stack antigo), troca com um clique
> - ⚡ **Runtime FrankenPHP** por site como alternativa ao PHP-FPM compartilhado, com modo worker do Laravel Octane e Symfony Runtime
> - 📦 **Isolamento de Node.js** por projeto (Node 22, 24) pelo fnm embutido ou por um **nvm** que você já tenha, alternável pelo dashboard, ou **bun** como runtime JS no host e, opt-in, dentro do container
> - 🖥️ **Web UI embutida** com dashboard raiz, widgets ao vivo, command palette global (Cmd+K), instalar/remover versões de PHP e Node pela página System, e quatorze idiomas de dashboard
> - ✏️ **Edição de config no navegador** — nginx por site e global, `php.ini` por versão, arquivos `.env` e tuning de runtime de banco/serviço, cada um validado (`nginx -t` onde se aplica), com backups timestampados e restauração em um clique
> - 🧪 **Aba Tinker** — REPL PHP no navegador por site com autocomplete (models do projeto, helpers do composer, built-ins do PHP), checagem de sintaxe ao vivo (`php -l`) e árvore colapsável para a saída de `dump()`. Funciona em Laravel (`artisan tinker`), Symfony e qualquer projeto PHP baseado em composer
> - 🛰️ **Janela de Debug** que intercepta todo `dump()` / `dd()` e transmite para o dashboard, TUI (tecla `D`), MCP e `lerd dump tail`, escopado por site e por branch de worktree, deixando a resposta original limpa a menos que você ligue o passthrough. A mesma janela captura queries SQL com detecção de N+1 e slow-query, além de e-mails enviados, views renderizadas, eventos disparados, jobs enfileirados e HTTP de saída, tanto em Laravel quanto Symfony, com captura opcional (opt-in) da atividade dos workers de fila
> - 🔥 **Profiler SPX** com liga/desliga em um clique: toda requisição PHP-FPM vira um flame graph visível numa view Profiler same-origin no dashboard. Sem restart do FPM, sem mudar código, e `lerd profile run` perfila um comando artisan ou CLI pontual
> - 💻 **Dashboard no terminal** (`lerd tui`) — TUI estilo btop com status ao vivo, painel de detalhe do site, edição inline de domínio e versão, drop-in de shell, tail de logs e filtro/ordenação — a mesma superfície de operações da web UI, para fluxos com tmux e SSH
> - 🗄️ **Serviços em um clique**: MySQL, PostgreSQL, Redis, Meilisearch, RustFS, Mailpit, Gotenberg, Stripe Mock, Reverb e mais. Todo serviço padrão é um preset YAML que você pode atualizar, migrar, reverter ou reinstalar no lugar, incluindo um reinstall com reset de dados que recria automaticamente os bancos e buckets dos sites vinculados, além de snapshots de banco sob demanda (criar / listar / restaurar / apagar) via CLI, dashboard e MCP
> - 🌳 **Git worktrees de primeira classe** com domínios de branch auto-detectados, versões PHP/Node por worktree, isolamento opcional de banco por worktree (clone do main ou vazio), proxy LAN-share por worktree, templating de `env_overrides` no `.lerd.yaml` para apps multi-tenant, SANs de certificado wildcard automáticos para `*.branch.site.test`, um worker de dev server Vite embutido que roda no host por branch, e um modal no dashboard para adicionar e remover worktrees sem tocar na CLI
> - ⚒️ **Auto-recuperação de workers**: workers de queue, schedule, horizon, reverb e stripe que falharam aparecem em todo lugar (CLI, banner do dashboard, TUI, MCP) e são recuperados com um clique ou `lerd worker heal`, com auto-reload opt-in do Horizon (`horizon:listen`) para o dev pegar mudanças de código sem restart manual (alternável pelo dashboard)
> - 🌍 **Compartilhar um site** na LAN com porta estável e QR code, ou publicamente por ngrok, cloudflared, Expose, serveo ou localhost.run, pela CLI ou pelo menu de share do dashboard. Defina um domínio base no Cloudflare uma vez e todo share é servido em `<site>.<domínio>`, então um webhook ou callback OAuth mantém a URL entre execuções, e o ngrok roda da própria imagem publicada numa máquina que nunca o instalou
> - 🎨 **Dev server no domínio do próprio site**: um Vite rodando serve os assets e o socket de hot-reload sob o hostname do site em vez de anunciar `localhost:5173`, então uma página compartilhada, aberta pela LAN ou de worktree chega estilizada. Nada no projeto é editado e nada é declarado por framework
> - 🧷 **Wiring de banco na IDE** (JetBrains): o projeto ganha um data source apontando para o banco dele no lerd, na porta que realmente responde no host, escrito no link e atualizado conforme o banco muda, sem tocar em nenhum data source que não seja do lerd
> - 🗃️ **Entidades e ações declaradas no preset**: create, drop, export, import e snapshots saíram do Go e viraram declaração YAML, então o MongoDB ganha a aba Databases inteira e o RustFS lista seus buckets na mesma grade — um engine novo no store ganha essas operações sem release do binário
> - 📌 **Host tools fixados** (Composer, fnm, mkcert) atrás de um manifesto publicado em vez do que o `releases/latest` servir no dia, com a versão pendente no card do System como o botão que aplica a atualização
> - 📋 **Logs ao vivo** de PHP-FPM, Queue, Schedule, Reverb, por site
> - 🔒 **Rootless & daemonless** — Podman-native, sem Docker, dual-stack IPv4 + IPv6
> - 🤖 **Servidor MCP** — deixe assistentes de IA (Claude Code, Windsurf, Junie) gerenciarem seu ambiente diretamente
> - 🧩 **Framework store** — definições da comunidade para Laravel, Symfony, WordPress, Drupal, CakePHP, Statamic com auto-detecção versionada
> - ⚡ **Agnóstico de framework**: workers, setup de env e proxy nginx — guiados por definições YAML, não hardcoded
> - 🪟 **Windows via WSL2 (beta)** — `lerd wsl:setup` provisiona systemd, networking mirrored e os pré-requisitos; guia completo em [docs/wsl2.md](docs/wsl2.md)
> - 🧰 **`lerd init` semeia a partir de Herd, DDEV e Lando** e suporta um `.env.lerd_override` pessoal por projeto, para overrides que nunca tocam o `.env` versionado

Esta fork adiciona, por cima de tudo isso:

| Recurso                            | Upstream (lerd-env/lerd)       | Esta fork (gabriel-sousa99/lerd)                                    |
| ---------------------------------- | ---------------------------- | ------------------------------------------------------------------- |
| Driver Oracle (`oci8`)             | instalar manual              | **compilado em toda imagem** (`oci8` 2.0.12 → 3.4.1 por versão PHP) |
| Oracle Instant Client              | n/a                          | **21.18 LTS** em `/opt/oracle/instantclient`                        |
| `tnsnames.ora` / wallet Autonomous | n/a                          | **`$TNS_ADMIN` montado read-only** de `~/.config/lerd/oracle/network/admin` |
| `memcached` / `amqp`               | precisa `lerd php:ext`       | **pré-instaladas**                                                  |
| `openssh-client` no container      | ausente (composer ssh falha) | **instalado** + `$HOME/.ssh` montado em `/root/.ssh`                |
| Suporte PHP                        | 7.4 → 8.5                    | **5.6 → 8.5** (5.6 legacy com libresolv shim)                       |
| `lerd init`/`link` → Database      | sqlite / mysql / postgres    | **+ Oracle (externo)**, bloco `oracle:` no `.lerd.yaml`             |
| DNS padrão                         | `lerd-dns` + `.test` (sudo)  | **off, `.localhost`** (sem sudo, RFC 6761)                          |
| Comandos destrutivos no dashboard  | um clique                    | **filtrados em 2 camadas** (lista + HTTP 403)                       |
| Comandos artisan customizados      | só os do framework           | **auto-discovery de `app/Console/Commands/*.php`**                  |
| Editor de `.env` no dashboard      | só leitura                   | **editável** (Save/Discard/Ctrl+S + backup auto)                    |
| Instalar versão PHP                | só CLI                       | **botão no dashboard + SSE logs ao vivo**                           |
| Service presets adicionais         | mysql/postgres/redis/…       | **+ `oracle-xe` + `typesense` + `typesense-dashboard`**             |
| Xdebug por padrão                  | `start_with_request=yes`     | **`=trigger`** (sem spam em CLI sem IDE)                            |
| Versão                             | `1.33.1`                     | `1.33.1-oracle.1`                                                   |

### Extensões PHP nas imagens

**~32 extensões prontas** — cobre o ecossistema top-10 Laravel (Sanctum,
Horizon, Telescope, spatie/\*, Filament, Socialite, Livewire, Debugbar, Excel,
Dompdf) sem `lerd php:ext add`:

```
oci8        memcached  amqp        redis      imagick    mongodb
igbinary    pcov       xdebug      spx        opcache
curl        gd         intl        zip        pdo_mysql  pdo_pgsql
mysqli      soap       xsl         ldap       pcntl      exif
bcmath      mbstring   gmp         bz2        sysv*      sockets
calendar    dba        shmop       (+ tudo que o php:X.Y-fpm-alpine já traz)
```

Extras: `lerd php:ext add <nome>` (ou pelo dashboard) compila qualquer outra
extensão PECL/PHP por cima da imagem, com deps Alpine (`--apk-deps`).

> ⚠️ **PHP 5.6 (legacy)** vem sem `memcached`/`amqp`/`pcov`/`spx` (PECL atuais
> não compilam em 5.6). Tem oci8 2.0.12 + xdebug 2.5.5 + redis 4.3 + imagick +
> mongodb 1.7 — para apps Laravel 5.x legados que falam com Oracle.

---

## Abrangência

Nada aqui depende de Docker Desktop, banco externo ou pacote do sistema além de
`podman` + `mkcert` (HTTPS opcional) + `git`.

### Versões PHP

| Linha      | Versões         | Notas                                                                       |
| ---------- | --------------- | --------------------------------------------------------------------------- |
| Suportadas | 7.4 → 8.5       | Build próprio FPM (Alpine), com `oci8` específico por versão                |
| Legacy     | 5.6             | Build estendido com `libresolv` shim                                        |
| Sob demanda| qualquer        | `lerd php:install <X.Y>` puxa, builda quadlet, registra no `php:list`       |
| FrankenPHP | 8.2 / 8.3 / 8.4 | Runtime worker, via `.lerd.yaml: runtime: frankenphp`                       |

> Detecção por projeto: `.lerd.yaml: php_version` → `.php-version` →
> `composer.json: require.php` → `php.default_version`. Sempre clampada ao
> range do framework detectado.

### Serviços (21 presets)

Cada preset vira um container systemd user-unit (`lerd-<nome>.service`).

| Categoria        | Presets                                                                                          |
| ---------------- | ------------------------------------------------------------------------------------------------ |
| **Bancos**       | `mysql` (8.4), `mariadb` (11), `postgres` (16 + PostGIS), `mongo`, `oracle-xe` (21c XE, da fork) |
| **Cache / KV**   | `redis`, `memcached`                                                                              |
| **Search**       | `meilisearch`, `typesense`, `elasticsearch`                                                       |
| **Mensageria**   | `rabbitmq`                                                                                        |
| **Object store** | `rustfs` (S3-compatível)                                                                          |
| **Mail / PDF**   | `mailpit` (SMTP catcher + UI), `gotenberg`                                                        |
| **Testes**       | `selenium` (Dusk/Panther), `stripe-mock`                                                          |
| **Admin UI**     | `phpmyadmin`, `pgadmin`, `mongo-express`, `elasticvue`, `typesense-dashboard`                     |

```bash
lerd service list           # disponíveis vs instalados + versões
lerd service status         # estado runtime de todos
lerd service preset <nome>  # instala via wizard com seleção de versão
```

### Frameworks

Laravel e Symfony são built-in (detecção por `composer.json`/estrutura).
WordPress, Drupal, Statamic etc. via `lerd framework add` ou
`~/.config/lerd/frameworks/<nome>.yaml`. Cada framework define range de PHP, env
file, detecção de serviços, workers e comandos artisan.

### Dashboard

Editor de `.env` (sites + serviços), gerenciamento de extensões PHP, instalador
de versão PHP com SSE logs, comandos artisan auto-discovery, filtro de
destrutivos, "Abrir no editor", worktrees por branch e aba de Proxies.
Detalhes em **[docs/dashboard.md](docs/dashboard.md)**.

---

## Comparação

|                      | Lerd | DDEV | Lando | Laravel Herd |
|----------------------|------|------|-------|--------------|
| Podman-native        | ✅   | 🟡   | ❌    | ❌           |
| Rootless             | ✅   | ❌   | ❌    | ✅           |
| Web UI               | ✅   | ❌   | ❌    | ✅           |
| Dashboard no terminal| ✅   | ❌   | ❌    | ❌           |
| Linux                | ✅   | ✅   | ✅    | ❌           |
| macOS                | ✅   | ✅   | ✅    | ✅           |
| Windows (WSL2)       | 🧪   | ✅   | ✅    | ✅           |
| Servidor MCP         | ✅   | ❌   | ❌    | ✅           |
| Livre & open source  | ✅   | ✅   | ✅    | ❌           |

🟡 O DDEV roda sobre Docker por padrão e também pode usar Podman como runtime alternativo; o Lerd é feito exclusivamente para Podman rootless.

---

## Instalação

**Pré-requisitos:** Linux (Arch, Fedora/Nobara, Debian/Ubuntu, openSUSE) ou
macOS · **Podman 4+** (rootless, sem Docker) · `git`, `curl`/`wget`.

### Linux

```bash
curl -fsSL https://raw.githubusercontent.com/gabriel-sousa99/lerd/oracle-oci8-support/install.sh | bash
```

O script verifica `podman`/`git`/`mkcert`, baixa o binário para
`$HOME/.local/bin/lerd` e pergunta se quer DNS gerenciado — **padrão neste fork:
NÃO** → sites em `http://meusite.localhost/` (sem sudo, RFC 6761).

```bash
lerd about    # confirma "Lerd Oracle Edition" e versão 1.33.1-oracle.1
```

### macOS

O mesmo script: ele detecta o sistema e baixa o binário `darwin` da release do
fork, incluindo o `lerd-tray`. Não há tap do Homebrew para este fork — o tap
`lerd-env/lerd` instala o upstream, sem Oracle.

> [!TIP]
> No **WSL2** há premissas extras (systemd, networking mirrored, projetos no
> `$HOME`). Guia completo em **[docs/wsl2.md](docs/wsl2.md)**.

---

## Primeiro uso

```bash
cd ~/meu-projeto-laravel
lerd init     # wizard: PHP, Node, HTTPS, Database, serviços, workers
lerd open     # abre em http://meu-projeto-laravel.localhost
```

---

## Trabalhando com Oracle

```bash
# cliente Laravel
lerd composer require yajra/laravel-oci8
lerd php artisan vendor:publish --provider="Yajra\Oci8\Oci8ServiceProvider" --tag=oracle --force

# confirmar que oci8 carregou
lerd php -r 'var_dump(extension_loaded("oci8"));'   # bool(true)
lerd php --ri oci8 | head -6                         # OCI8 3.4.1 / IC 21.18
```

**Oracle XE 21 local** (validação sem o servidor corporativo) — use o preset
`lerd service preset oracle-xe`, ou manualmente:

```bash
podman run -d --name lerd-oracle-test -p 1521:1521 \
  -e ORACLE_PASSWORD=lerd -e ORACLE_DATABASE=LERDPDB \
  -e APP_USER=lerd_app -e APP_USER_PASSWORD=lerd \
  docker.io/gvenzl/oracle-xe:21-slim-faststart
# espere "DATABASE IS READY" (~30s), aponte o .env e: lerd php artisan migrate
```

**Charset / NLS_LANG:** o bloco `oracle:` aceita `charset:` opcional; quando
definido, `lerd env` escreve `DB_CHARSET` + `NLS_LANG`:

```yaml
oracle:
  host: oracle.example.com
  port: 1521
  service_name: PRODPDB
  username: app_user
  password: ${ORACLE_PASSWORD}   # use placeholder e set no shell
  charset: AL32UTF8              # ou WE8MSWIN1252, WE8ISO8859P15
```

### `tnsnames.ora` e wallet (Autonomous Database)

Banco corporativo raramente é endereçado por `host:port/service`: é endereçado
por **alias** do `tnsnames.ora`, e o Autonomous Database exige um **wallet**
baixado do console. O Instant Client procura os dois em `$TNS_ADMIN`, que nas
imagens do fork é `/opt/oracle/network/admin`, montado **read-only** a partir de:

```
~/.config/lerd/oracle/network/admin/
```

O diretório é criado no primeiro `lerd start` com modo `0700` (é credencial).
Largue os arquivos lá e reinicie o PHP:

```bash
# alias: copie o tnsnames.ora corporativo
cp /caminho/tnsnames.ora ~/.config/lerd/oracle/network/admin/

# Autonomous: descompacte o wallet inteiro (cwallet.sso, sqlnet.ora, tnsnames.ora…)
unzip Wallet_MEUDB.zip -d ~/.config/lerd/oracle/network/admin/

lerd restart
```

Depois use o **alias** onde iria o host, sem porta nem service name:

```yaml
oracle:
  host: MEUDB_high        # alias do tnsnames.ora / wallet
  username: app_user
  password: ${ORACLE_PASSWORD}
```

Conferir de dentro do container:

```bash
lerd php -r 'echo getenv("TNS_ADMIN"), PHP_EOL;'   # /opt/oracle/network/admin
lerd php -r 'var_dump(oci_connect("app_user", getenv("DB_PASSWORD"), "MEUDB_high") !== false);'
```

Nada disso entra na imagem nem no repositório: o wallet fica só no host, montado
somente-leitura, então um container não consegue reescrevê-lo.

---

## Serviços

```bash
lerd service start <nome>      # ex: lerd service start mysql
lerd service stop/restart <nome>
lerd service status            # lista todos com estado
lerd quit                      # para containers + UI + watcher + tray
```

Pausar um único site (mantém serviços): `lerd pause` / `lerd unpause` dentro do
projeto (o nginx vhost vira landing page).

## Rebuild de imagens PHP

```bash
lerd php:rebuild              # todas as versões instaladas
lerd php:rebuild 8.4          # só a 8.4
lerd php:rebuild --local      # do zero, sem pull do ghcr
lerd php:ext add pdo_dblib --apk-deps "freetds-dev"   # extensão por cima
```

> [!NOTE]
> Desde o merge com o upstream v1.30.1, o conjunto de extensões customizadas é
> **global** — vale para toda imagem PHP. `php:ext add` não recebe mais versão;
> a versão só decide qual imagem é reconstruída na hora.

> [!NOTE]
> O template Containerfile deste fork difere do upstream, então o SHA dos pulls
> não bate e o lerd cai no build local automaticamente — é o esperado, garante
> que Instant Client + oci8 + memcached + amqp fiquem na imagem.

## Atualização

```bash
lerd update                # último release em gabriel-sousa99/lerd
lerd update --beta         # pre-release, se houver
lerd update --rollback     # volta à versão anterior (backup automático)
```

Faz download + substituição atômica do binário, reaplica quadlets/DNS/sysctl e,
se a Containerfile mudou, roda `lerd php:rebuild`.

> [!WARNING]
> **Não** use `lerd-installer --update` apontado para o upstream — sobrescreve o
> binário com a versão sem suporte Oracle.

## Desinstalação

```bash
cd my-laravel-project
lerd link
# → https://my-laravel-project.test
lerd open           # the site in your browser
lerd code           # the project in your editor
```

Starting from nothing, `lerd new` asks which framework and which major from the store, scaffolds it, and links the result, so you land on a served site rather than on three commands to type.

`lerd install` already starts everything for you on first run, so you can `lerd link` immediately. Day-to-day:
lerd-installer --uninstall   # remove binários, units systemd e ~/.config|cache|share/lerd
```

> [!CAUTION]
> Os dados dos serviços ficam em `~/.local/share/lerd/data/` e são apagados.
> Faça backup antes (`lerd db:export <site>`). Para limpar imagens podman:
> `podman images --filter "reference=lerd-*" -q | xargs podman rmi -f`.

---

## Proxies para projetos não-PHP

Para SPAs/dev servers (Vue, Quasar, Nuxt, Vite, SvelteKit) que rodam fora do
PHP, o lerd expõe um **proxy manual**: domínio local com HTTPS + WebSocket
fazendo reverse proxy para uma porta do host.

```bash
lerd proxy add app.localhost --port 9000                 # simples
lerd proxy add app.localhost --port 5173 --path ~/proj/app \
  --managed --cmd "npm run dev" --autostart              # lerd levanta o dev server
lerd proxy ls | rm <dom> | unsecure <dom> | start <dom> | logs <dom> -f
```

Há uma aba **Proxies** no dashboard com navegação em lista e detalhe, configuração
de aliases, upstream HTTP/HTTPS, health check, timeout, TLS, autostart e rotas
independentes. As abas de operação mostram alcance e latência do upstream,
tráfego, logs de proxies gerenciados, estado do Nginx, certificado e o vhost
gerado em modo somente leitura. O monitoramento ocorre apenas enquanto o detalhe
está aberto. Por baixo: `nginx proxy_pass` para
`host.containers.internal:<porta>`, HTTPS via `mkcert`, headers
`Upgrade`/`Connection` (Vite HMR/WS funcionam).

### Fullstack (SPA + API na mesma origem)

SPA + API normalmente viram dois domínios — origens diferentes, e como
`.localhost` é public-suffix o cookie de sessão não atravessa os hosts →
**sessão, CSRF e Sanctum quebram**. O modo **fullstack** serve ambos sob **uma
origem**, separados por **path**: cookie first-party, sem CORS.

```bash
# API por um site do lerd (fastcgi, sem porta) — ou --api-port 8000 (dev server externo)
lerd proxy add retencao.localhost --port 9000 --api-site retencao-api

# --path sincroniza a API-base do .env da SPA para a origem unificada
lerd proxy add retencao.localhost --port 9000 --api-site retencao-api \
  --path ~/projetos/retencao-spa
```

Roteamento resultante: `/` → SPA (porta, com HMR); `/api /sanctum /broadcasting
/storage /redirect /authenticate /login /logout /up` → API.

- Cada lado aponta para um **site do lerd** (fastcgi) **ou** uma **porta**.
  Defaults: SPA=porta, API=site.
- **Sync de `.env`** (idempotente, só chaves existentes): a API recebe
  `APP_URL`/`SESSION_DOMAIN`/`SANCTUM_STATEFUL_DOMAINS` no domínio unificado; com
  `--path`, a SPA recebe a API-base (`URL_API`/`VITE_API_URL`/`VITE_APP_API_URL`).
  Desvincular/remover reverte.
- Os defaults de rota cobrem fluxos SSO comuns
  (`401 → /redirect → /authenticate`). Rotas custom via `--api-path` (repetível).

No **dashboard**: toggle **Simples | Fullstack** com pickers, paths editáveis,
mapa de rotas ao vivo e bloco "Proxy fullstack" em cada site.

---

## Diagnóstico

```bash
lerd doctor               # DNS, podman, certs, services, versão
lerd dns:check            # diagnóstico em camadas do resolver
lerd bug-report           # .tar.gz com tudo pra abrir issue
lerd logs <site|service>  # ex: lerd logs mysql
```

Guias por tópico em [`docs/DEBUG.md`](docs/DEBUG.md) — `502`/nginx, DNS,
podman/quadlet, Oracle (`ORA-*`), updates, workers, conflito de porta. Também no
dashboard em **System → Debug & Troubleshoot**.

---

## Lista de comandos úteis

| Comando                               | O que faz                                                               |
| ------------------------------------- | ----------------------------------------------------------------------- |
| `lerd init` / `link` / `unlink`       | Wizard `.lerd.yaml` / registra / remove o site                          |
| `lerd open` / `dashboard` / `tui`     | Navegador / painel web / painel terminal                                |
| `lerd php <args>`                     | Roda php no container (ex: `lerd php artisan tinker`)                    |
| `lerd composer <args>`                | Composer com binários `composer-global` no PATH                         |
| `lerd npm` / `npx <args>`             | Node do projeto via fnm                                                 |
| `lerd db:shell` / `:export` / `:import` | Shell / backup / restore do DB                                        |
| `lerd db:isolate`                     | DB próprio para o worktree atual                                        |
| `lerd horizon` / `queue` / `schedule` / `reverb` `:start`/`:stop` | Workers como serviço systemd               |
| `lerd secure` / `unsecure`            | Liga / desliga HTTPS via mkcert (+ reload do nginx)                     |
| `lerd lan` / `remote-control`         | Expõe sites no LAN / acesso ao dashboard via LAN                        |
| `lerd mcp:enable-global`              | Registra MCP server para Claude / IDE / agentes                         |
| `lerd php:install <ver>`              | **(fork)** Provisiona versão (5.6 → 8.5): build + quadlet + start       |
| `lerd php:rebuild [ver]`              | Reconstrói image FPM                                                    |
| `lerd php:ext add <ext> [ver]`        | Instala extensão via PECL + apk-deps                                    |
| `lerd php:ini <ver>`                  | Edita `98-user.ini` (com validação)                                     |
| `lerd service preset <name>`          | Instala um preset (ex: `oracle-xe`, `typesense`)                        |
| `lerd proxy add/ls/rm`                | Proxy manual para SPAs/dev servers (ver acima)                          |

Lista completa: `lerd --help`.

- [phpantom_lsp](https://github.com/PHPantom-dev/phpantom_lsp) - the PHP language server behind tinker autocomplete, diagnostics and semantic highlighting
- [Monaco](https://github.com/microsoft/monaco-editor) - the editor engine for every in-browser editing surface
- [php-spx](https://github.com/NoiseByNorthwest/php-spx) - the profiler behind the SPX flame graphs
- [mkcert](https://github.com/FiloSottile/mkcert) - the local CA that backs `.test` HTTPS
- [fnm](https://github.com/Schniz/fnm) - the per-project Node version manager
- [Composer](https://getcomposer.org) - fetched on the host for dependency operations
- [Starship](https://starship.rs) - the prompt in the container shell drop-in
- [Simple Icons](https://simpleicons.org) - the service marks in the dashboard, CC0
---

## Compilando do código

```bash
git clone https://github.com/gabriel-sousa99/lerd.git
cd lerd
make build      # binário em build/lerd
make install    # copia para ~/.local/bin/
make test       # go test ./...
```

Requisitos: Go 1.25+, Node 22+, npm 10+.

---

## Créditos

- **Lerd** original — [George Dumitrescu](https://github.com/geodro)
- **Suporte Oracle** (este fork) — [Gabriel Sousa](https://github.com/gabriel-sousa99)

Licença: MIT (herdada do upstream — ver [`LICENSE`](LICENSE)).
