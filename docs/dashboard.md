# Recursos exclusivos do dashboard

Adições desta fork que vivem no painel web (`http://lerd.localhost`).

## Editor de `.env` (sites + serviços)

- **Site → aba Env:** textarea editável com Save/Descartar/Ctrl+S, dirty badge,
  beforeunload prompt. O backend cria `.env.before_lerd` automático na primeira
  edição (restaurável via `lerd env:restore`).
- **Serviço → aba Env:** editor key=value pro bloco `Environment=` do quadlet.
  Source badge mostra "preset" vs "override do usuário". + adicionar variável /
  remover por linha. Save escreve em `~/.config/lerd/services/<name>.yaml` —
  precisa Restart do serviço pra aplicar (avisado na UI).
  **Loopback-only:** o bloco `Environment=` carrega credenciais
  (`ORACLE_PASSWORD`, `POSTGRES_PASSWORD`, …), então tanto a leitura quanto a
  escrita ficam restritas à máquina local, como já acontece com a aba Env de um
  site. Um cliente da LAN recebe 403 mesmo com credenciais válidas.

> Subir um projeto **preserva o `.env`**: só `APP_URL`/`SESSION_DOMAIN`/
> `SANCTUM_STATEFUL_DOMAINS`/`VITE_REVERB_*`/`REVERB_*` são tocados; `DB_*`,
> `REDIS_*`, `MAIL_*` e credenciais ficam intactos.

## Gerenciamento de extensões PHP

`System → PHP X.Y → Extensões customizadas`: chips de exemplo rápido (`imap`,
`swoole`, `ssh2`, `apcu`, `event`, `pspell`, `tidy`, `pdo_dblib`) que
pré-preenchem nome + `apk-deps`, ou form livre. Build com spinner enquanto
reconstrói. Equivalente a `lerd php:ext add <ext> X.Y --apk-deps "..."`.

## Instalador de versão PHP com logs ao vivo

`System → Instalar versão…`: lista versões disponíveis ainda não instaladas
(5.6 / 7.4 / 8.0 / 8.1 / 8.2 / 8.3 / 8.4 / 8.5) com 1 clique. **Logs SSE ao
vivo** durante o build (apk add / pecl install / COPY layers) com auto-scroll +
"Copiar log". **beforeunload warning** evita fechar a aba no meio do build.

## Comandos artisan customizados (Laravel)

Dropdown **Commands** em cada site Laravel inclui **comandos descobertos
automaticamente** em `app/Console/Commands/*.php` — extraídos via regex do
`$signature` + `$description` (sem rodar PHP). Ícone ▶ distingue dos defaults do
framework.

⛔ **Filtro de destrutivos** em duas camadas:

1. **List filter:** `GET /api/sites/{d}/commands` nunca retorna `migrate:fresh`,
   `db:wipe`, `schema:drop`, `doctrine:fixtures:load`, `queue:flush`,
   `DROP TABLE`, `rm -rf /`, etc.
2. **Runtime block:** `handleCommandRun` retorna HTTP 403 mesmo se o comando
   passar pela lista.

Pra rodar mesmo assim → sempre via CLI: `lerd php artisan migrate:fresh --force`.

## Debug & Troubleshoot

`System → Debug & Troubleshoot`: botões pra rodar diagnósticos contra a
instalação atual (`lerd doctor`, `dns:check`, `podman ps -a`, últimos logs) +
grid de cards pros guias do `docs/debug/*.md`. Botão "Copiar relatório" gera um
bundle pra colar em issue.

## Botão "Abrir no editor" ao lado do terminal

Em cada site: ícone `</>` ao lado do terminal abre o projeto no editor GUI.
Sonda: `$EDITOR_GUI` → `code` / `code-insiders` / `codium` / `cursor` →
JetBrains (`phpstorm` / `webstorm` / `idea` / `goland`) → `subl` / `zed` /
`nova`. macOS: `open -a "Visual Studio Code"` etc.

## Proxies (incl. fullstack)

A aba **Proxies** segue a mesma navegação em lista e detalhe usada por Sites e
Serviços. A lista mostra o estado operacional de cada domínio. No detalhe, as
abas **Visão geral**, **Tráfego**, **Logs** (quando o dev server é gerenciado) e
**Configuração e diagnóstico** reúnem o upstream, as rotas, o estado do Nginx,
o health check, a latência, as rotas lentas e o vhost gerado em modo somente
leitura. Estado e tráfego são atualizados a cada dez segundos enquanto a tela
está aberta, com atualização manual disponível no diagnóstico.

O editor cobre proxies simples e fullstack, aliases, upstream HTTP ou HTTPS,
host e porta, caminho de health check, timeout, TLS, autostart e rotas com
destinos independentes por site ou host/porta. A configuração continua
declarativa em `proxies.yaml`, e cada alteração regenera o vhost. Ver
[Proxy fullstack](features/proxy-fullstack-frontend.md) e
[sites host-proxy](usage/host-proxy.md).

## Serviços novos (presets desta fork)

- **`oracle-xe`** (gvenzl/oracle-xe:21-slim-faststart) — Oracle XE 21c local pra
  dev. Cria automaticamente `LERDPDB` + usuário `lerd_app/lerd`.
  `userns: keep-id:uid=54321,gid=54321` + `chown_data` pra rootless. NLS_LANG pt-BR.
  No wizard do `lerd init` ele aparece como opção de **Database** (escolha única),
  não na multi-seleção de Serviços: é um banco, e oferecê-lo como serviço deixava
  marcar `oracle-xe` com o Database ainda em `sqlite`, gravando dois
  `DB_CONNECTION` diferentes no `.env`.
- **`typesense`** (typesense/typesense:28.0) — search engine open-source,
  alternativa Meilisearch/Algolia. Configura `SCOUT_DRIVER=typesense` no `.env`.
- **`typesense-dashboard`** (bfritscher/typesense-dashboard) — companion web pra
  typesense, segue o padrão do `pgadmin/postgres`.
