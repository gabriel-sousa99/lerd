# Instalando no WSL2 (Ubuntu/Debian)

O lerd roda no WSL2, mas algumas premissas que ele assume em Linux nativo
(systemd ativo, NetworkManager/resolved gerenciando DNS, browser local)
não valem por padrão no WSL. Esta página descreve a configuração mínima
para que tudo funcione.

> [!IMPORTANT]
> **Distros suportadas:** Ubuntu 22.04+ ou Debian 12+ rodando em WSL ≥
> 0.67.6 (que trouxe suporte oficial a systemd). Versões mais antigas
> precisam de workarounds tipo `genie`/`subsystemctl` que não são testados.

## 1. Habilitar systemd no WSL2

Toda a arquitetura do lerd (Quadlet, watcher, UI, FPM) roda como user
units do systemd. Sem systemd, **nada inicia**.

Dentro da distro WSL, crie/edite `/etc/wsl.conf`:

```ini
[boot]
systemd=true

[user]
default=seu-usuario
```

Depois, no **PowerShell do Windows**: `wsl --shutdown`. Reabra o terminal WSL e confirme:

```bash
ps -p 1 -o comm=        # deve imprimir: systemd
systemctl --user status # deve responder sem erro
```

## 2. Configurar `networkingMode=mirrored` (recomendado)

Sem isso, `http://meusite.localhost` só responde **dentro** do WSL — o
Chrome do Windows não acha. Com o modo espelhado, a interface de rede
do WSL2 reflete a do Windows e `localhost` funciona dos dois lados.

No Windows, edite/crie `%USERPROFILE%\.wslconfig`:

```ini
[wsl2]
networkingMode=mirrored
dnsTunneling=true
firewall=true
autoProxy=true
```

Depois `wsl --shutdown` de novo.

> [!NOTE]
> Requer WSL ≥ 2.0.0 e Windows 11 22H2+. Sem mirrored networking você
> ainda consegue acessar pelos IPs da VM (`ip addr show eth0`) ou usar
> o browser dentro do WSL com `wslview`/`xdg-open`.

## 3. Instalar pacotes base

```bash
sudo apt update
sudo apt install -y \
  podman uidmap fuse-overlayfs slirp4netns \
  curl git unzip libnss3-tools
```

- `podman` + `uidmap` + `slirp4netns` → rootless containers
- `fuse-overlayfs` → fallback se o overlayfs nativo do kernel WSL falhar
- `libnss3-tools` → fornece `certutil` para o `mkcert`

> [!TIP]
> Não use `docker.io`/`docker-ce` no WSL2 com lerd — ele é construído
> exclusivamente para Podman rootless + Quadlet.

## 4. Habilitar linger

Garante que os containers continuam rodando depois que você fecha o terminal:

```bash
sudo loginctl enable-linger $USER
```

Faça **logout e login de novo** (ou `wsl --shutdown` + reabrir) para a
sessão pegar a mudança.

## 5. Rodar o instalador em modo `.localhost`

```bash
curl -fsSL https://raw.githubusercontent.com/gabriel-sousa99/lerd/oracle-oci8-support/install.sh | bash
```

Quando perguntar **"Enable lerd-managed DNS for \*.test domains?"** responda
**N** (não). O fork já vem com `*.localhost` como padrão, que resolve
para loopback automaticamente (RFC 6761) sem precisar de
`systemd-resolved`/`NetworkManager` — coisas que normalmente não estão
rodando no WSL2.

> [!WARNING]
> **Não escolha o modo `.test`** no WSL2. O setup de DNS em
> `internal/dns/setup.go` precisa de NetworkManager ou systemd-resolved
> ativos para instalar dispatcher/drop-in, e nenhum dos dois roda por
> padrão no WSL2 Ubuntu/Debian. Vai falhar feio.

## 6. Mantenha projetos em `$HOME` (NÃO em `/mnt/c/...`)

Esta é a otimização **mais importante** no WSL2. Bind mounts de
`/mnt/c/...` para dentro de containers passam pelo 9P (filesystem
remoto), que é **ordens de magnitude mais lento** que `ext4` nativo do WSL2.

```bash
# ❌ EVITAR (Composer/npm install vão demorar 10x)
cd /mnt/c/Users/seu-user/projetos/meu-projeto && lerd init

# ✅ FAZER
mkdir -p ~/projetos && cd ~/projetos
git clone git@github.com:org/meu-projeto.git
cd meu-projeto && lerd init
```

Para editar com o VS Code do Windows, use a extensão **Remote – WSL** e
abra a pasta direto do WSL (`code .` dentro do WSL).

## 7. HTTPS / mkcert no WSL2

O `mkcert -install` que o lerd dispara instala a CA na trust store do
**WSL**, não do Windows:

- Browser dentro do WSL (Firefox/Chromium via apt): confia ✓
- Browser do Windows (Chrome/Edge/Firefox): **não confia** ✗

Opções:

- **Dev local:** use HTTP (`http://meusite.localhost`). Suficiente pra 90% dos casos.
- **HTTPS sério:** exporte a CA do mkcert e importe no Windows manualmente:
  ```bash
  cp "$(mkcert -CAROOT)/rootCA.pem" /mnt/c/Users/$USER/Desktop/lerd-rootCA.crt
  # no Windows: duplo-clique → Instalar → "Autoridades de certificação raiz confiáveis"
  ```

## 8. Tray icon não funciona

O `lerd-tray` precisa de uma bandeja gráfica (`StatusNotifierItem` /
`AppIndicator`), que o WSL2 não fornece nativamente. Sem prejuízo
funcional — pilote tudo via CLI ou pelo dashboard web em `http://lerd.localhost`.
Para desativar e não ver erros de log:

```bash
systemctl --user mask lerd-tray.service
```

## 9. Auto-start do Oracle XE de teste

Se for usar o preset `oracle-xe` (Oracle XE 21c local), confira que a
imagem `gvenzl/oracle-xe:21-slim-faststart` baixa por completo (~2.5 GB)
**antes** de `lerd service start oracle-xe`. WSL2 tem disco dinâmico — se
acabar o espaço no `.vhdx`, o serviço falha silencioso. Verifique:

```bash
df -h ~/.local/share/containers
```

## Checklist final

Antes do `lerd init` do primeiro projeto, valide tudo:

```bash
ps -p 1 -o comm=                                   # systemd
systemctl --user is-active default.target          # active
loginctl show-user $USER --property=Linger         # Linger=yes
podman info --format '{{.Host.Security.Rootless}}' # true
podman info --format '{{.Store.GraphDriverName}}'  # overlay
systemctl --user is-active lerd-ui                 # active
getent hosts teste.localhost                       # ::1 / 127.0.0.1
```

Se tudo verde, `cd ~/projetos/meu-projeto && lerd init` funciona como em
Linux nativo.

## Troubleshooting WSL2

| Sintoma                                           | Solução                                                         |
| ------------------------------------------------- | --------------------------------------------------------------- |
| `System has not been booted with systemd as init` | `[boot] systemd=true` em `/etc/wsl.conf` + `wsl --shutdown`     |
| `Failed to connect to bus` em `systemctl --user`  | falta `loginctl enable-linger` ou faltou relogar                |
| `http://...localhost` não abre no Chrome Windows  | falta `networkingMode=mirrored` no `.wslconfig`                 |
| Composer / npm install lentíssimo                 | projeto está em `/mnt/c/...` — mova para `~/projetos/`          |
| `podman build` falha com overlay                  | `sudo apt install fuse-overlayfs` + `podman system reset`       |
| `lerd doctor` reclama de DNS                      | você optou pelo modo `.test` — reinstale e escolha `.localhost` |
| Erro `cannot allocate memory` em builds grandes   | aumente RAM da VM: `[wsl2] memory=8GB` no `.wslconfig`          |
