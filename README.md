# Lerd Oracle Edition

> Fork do [`geodro/lerd`](https://github.com/geodro/lerd) com **suporte a
> Oracle Database embutido em todas as imagens PHP** — Oracle Instant
> Client 21.18 (LTS) + `oci8` + memcached + amqp já compilados, prontos
> para PHP 5.6 → 8.5. Drop-in replacement: todo comando `lerd` existente
> funciona igual.

> [!IMPORTANT]
> Este fork mantém o mesmo binário `lerd` (compatibilidade total) e
> aponta o auto-update para **este** repositório. Releases seguem o
> esquema `v1.23.1-oracle.N`.

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/platform-Linux%20%7C%20macOS%20%7C%20WSL2%20(beta)-lightgrey)]()
[![Docs](https://img.shields.io/badge/docs-geodro.github.io%2Flerd-blue)](https://geodro.github.io/lerd/)
[![Reddit](https://img.shields.io/badge/Reddit-r%2Flerd-ff2d20?logo=reddit)](https://reddit.com/r/lerd)
[![Discord](https://img.shields.io/badge/Discord-Join-5865F2?logo=discord&logoColor=white)](https://discord.gg/ej33c5N9s)

[![Fork base](https://img.shields.io/badge/forked%20from-geodro%2Flerd%20v1.23.1-blue)](https://github.com/geodro/lerd)
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

## O que muda em relação ao upstream

> Recursos herdados do upstream (`geodro/lerd`), todos presentes nesta fork:
>
> - 🌐 **Domínios `.test` automáticos** com TLS em um comando, ou [desative o DNS gerenciado pelo lerd](https://geodro.github.io/lerd/features/dns/) e use `*.localhost` (sem dnsmasq, sem mexer no resolver do sistema, sem sudo na parte de DNS); o DNS é ciente de VPN e re-sincroniza os resolvers dos containers em menos de um segundo quando um túnel conecta/desconecta
> - 🐘 **Versão de PHP por projeto** (8.1–8.5, mais uma faixa legacy congelada 7.4 / 8.0 para projetos em stack antigo), troca com um clique
> - ⚡ **Runtime FrankenPHP** por site como alternativa ao PHP-FPM compartilhado, com modo worker do Laravel Octane e Symfony Runtime
> - 📦 **Isolamento de Node.js** por projeto (Node 22, 24)
> - 🖥️ **Web UI embutida** com dashboard raiz, widgets ao vivo, command palette global (Cmd+K), instalar/remover versões de PHP e Node pela página System, e sete idiomas de dashboard (inglês, alemão, espanhol, francês, indonésio, holandês, português)
> - ✏️ **Edição de config no navegador** — nginx por site e global, `php.ini` por versão, arquivos `.env` e tuning de runtime de banco/serviço, cada um validado (`nginx -t` onde se aplica), com backups timestampados e restauração em um clique
> - 🧪 **Aba Tinker** — REPL PHP no navegador por site com autocomplete (models do projeto, helpers do composer, built-ins do PHP), checagem de sintaxe ao vivo (`php -l`) e árvore colapsável para a saída de `dump()`. Funciona em Laravel (`artisan tinker`), Symfony e qualquer projeto PHP baseado em composer
> - 🛰️ **Janela de Debug** que intercepta todo `dump()` / `dd()` e transmite para o dashboard, TUI (tecla `D`), MCP e `lerd dump tail`, escopado por site e por branch de worktree, deixando a resposta original limpa a menos que você ligue o passthrough. A mesma janela captura queries SQL com detecção de N+1 e slow-query, além de e-mails enviados, views renderizadas, eventos disparados, jobs enfileirados e HTTP de saída, tanto em Laravel quanto Symfony, com captura opcional (opt-in) da atividade dos workers de fila
> - 🔥 **Profiler SPX** com liga/desliga em um clique: toda requisição PHP-FPM vira um flame graph visível numa view Profiler same-origin no dashboard. Sem restart do FPM, sem mudar código, e `lerd profile run` perfila um comando artisan ou CLI pontual
> - 💻 **Dashboard no terminal** (`lerd tui`) — TUI estilo btop com status ao vivo, painel de detalhe do site, edição inline de domínio e versão, drop-in de shell, tail de logs e filtro/ordenação — a mesma superfície de operações da web UI, para fluxos com tmux e SSH
> - 🗄️ **Serviços em um clique**: MySQL, PostgreSQL, Redis, Meilisearch, RustFS, Mailpit, Gotenberg, Stripe Mock, Reverb e mais. Todo serviço padrão é um preset YAML que você pode atualizar, migrar, reverter ou reinstalar no lugar, incluindo um reinstall com reset de dados que recria automaticamente os bancos e buckets dos sites vinculados, além de snapshots de banco sob demanda (criar / listar / restaurar / apagar) via CLI, dashboard e MCP
> - 🌳 **Git worktrees de primeira classe** com domínios de branch auto-detectados, versões PHP/Node por worktree, isolamento opcional de banco por worktree (clone do main ou vazio), proxy LAN-share por worktree, templating de `env_overrides` no `.lerd.yaml` para apps multi-tenant, SANs de certificado wildcard automáticos para `*.branch.site.test`, um worker de dev server Vite embutido que roda no host por branch, e um modal no dashboard para adicionar e remover worktrees sem tocar na CLI
> - ⚒️ **Auto-recuperação de workers**: workers de queue, schedule, horizon, reverb e stripe que falharam aparecem em todo lugar (CLI, banner do dashboard, TUI, MCP) e são recuperados com um clique ou `lerd worker heal`, com auto-reload opt-in do Horizon (`horizon:listen`) para o dev pegar mudanças de código sem restart manual (alternável pelo dashboard)
> - 📋 **Logs ao vivo** de PHP-FPM, Queue, Schedule, Reverb, por site
> - 🔒 **Rootless & daemonless** — Podman-native, sem Docker, dual-stack IPv4 + IPv6
> - 🤖 **Servidor MCP** — deixe assistentes de IA (Claude Code, Windsurf, Junie) gerenciarem seu ambiente diretamente
> - 🧩 **Framework store** — definições da comunidade para Laravel, Symfony, WordPress, Drupal, CakePHP, Statamic com auto-detecção versionada
> - ⚡ **Agnóstico de framework**: workers, setup de env e proxy nginx — guiados por definições YAML, não hardcoded
> - 🪟 **Windows via WSL2 (beta)** — `lerd wsl:setup` provisiona systemd, networking mirrored e os pré-requisitos; guia completo em [docs/wsl2.md](docs/wsl2.md)
> - 🧰 **`lerd init` semeia a partir de Herd, DDEV e Lando** e suporta um `.env.lerd_override` pessoal por projeto, para overrides que nunca tocam o `.env` versionado

Esta fork adiciona, por cima de tudo isso:

| Recurso                            | Upstream (geodro/lerd)       | Esta fork (gabriel-sousa99/lerd)                                    |
| ---------------------------------- | ---------------------------- | ------------------------------------------------------------------- |
| Driver Oracle (`oci8`)             | instalar manual              | **compilado em toda imagem** (`oci8` 2.0.12 → 3.4.1 por versão PHP) |
| Oracle Instant Client              | n/a                          | **21.18 LTS** em `/opt/oracle/instantclient`                        |
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
| Versão                             | `1.23.1`                     | `1.23.1-oracle.N`                                                   |

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
lerd about    # confirma "Lerd Oracle Edition" e versão 1.23.1-oracle.N
```

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

No **Database**, além de SQLite/MySQL/PostgreSQL, há **Oracle (externo)** — um
sub-form pede host, porta (1521), service name/SID, usuário e senha. Os valores
vão para um bloco `oracle:` no `.lerd.yaml` e o `.env` recebe
`DB_CONNECTION=oracle` + `DB_HOST`/`DB_PORT`/`DB_DATABASE`/`DB_USERNAME`/`DB_PASSWORD`.
Lerd não sobe container Oracle (é DB externo).

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
lerd php:ext add pdo_dblib 8.4 --apk-deps "freetds-dev"   # extensão por cima
```

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

Há uma aba **Proxies** no dashboard com a mesma funcionalidade. Por baixo:
`nginx proxy_pass` para `host.containers.internal:<porta>`, HTTPS via `mkcert`,
headers `Upgrade`/`Connection` (Vite HMR/WS funcionam).

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
