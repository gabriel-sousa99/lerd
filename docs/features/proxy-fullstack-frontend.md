# Proxy Fullstack — Frontend como cidadão de 1ª classe

Ao registrar um proxy fullstack com `--path <pasta-da-spa>`, o lerd sincroniza a
**API-base** do `.env` da SPA para a origem unificada do proxy, eliminando o
`URL_API` cross-origin que quebra Sanctum/CORS.

## Uso

```bash
lerd proxy add gestao-clientes.localhost \
  --port 9000 \
  --path /caminho/para/gestao-clientes-spa \
  --api-site gestao-clientes-api
```

O que o lerd faz:

- **API (site Laravel):** aponta `APP_URL`, `SESSION_DOMAIN`,
  `SANCTUM_STATEFUL_DOMAINS`, … (`DomainScopedKeys`) para `https://gestao-clientes.localhost`.
- **Frontend (SPA):** aponta `URL_API` / `VITE_API_URL` / `VITE_APP_API_URL`
  (`FrontendAPIBaseKeys`, **só se já presentes**) para `https://gestao-clientes.localhost`
  (raiz da origem, **sem** `/api` — a SPA concatena seus próprios prefixos).

Só chaves já presentes são reescritas; nada fora desses sets é tocado.

## Chaves de API-base reconhecidas

`URL_API` (Quasar), `VITE_API_URL`, `VITE_APP_API_URL`. Projetos com outra
convenção devem adicionar a chave a `FrontendAPIBaseKeys` em
`internal/envfile/envfile.go` ou apontá-la manualmente.

## HMR atrás do proxy (não automatizado)

Quasar/Vite leem config de HMR de `quasar.config.js`/`vite.config`, **não** do
`.env` — fora do alcance do env-sync. Para o websocket de HMR funcionar via
HTTPS no proxy, configure manualmente:

```js
// quasar.config.js
devServer: {
  hmr: {
    clientPort: 443,
    protocol: 'wss',
  },
}
```

## Rotas custom

Os defaults de rota cobrem `/api /sanctum /broadcasting /storage /redirect
/authenticate /login /logout /up`. `/redirect` e `/authenticate` são as rotas
**web no root** de fluxos SSO comuns (`401 → /redirect → provider →
/authenticate → sessão`). Rotas fora desse conjunto exigem `--api-path`
explícito (repetível):

```bash
lerd proxy add app.localhost --port 9000 --path ./spa --api-site app-api \
  --api-path /api --api-path /webhooks --api-path /redirect --api-path /authenticate
```

No dashboard, ative **Editar rotas individualmente** para misturar destinos no
mesmo domínio. Cada path pode apontar para um site lerd ou para seu próprio
host/porta; o target da base `/` continua configurado separadamente no bloco
SPA. O editor preserva essa lista ao reabrir o proxy, sem achatar destinos
diferentes para o primeiro target.

## Operação e diagnóstico no dashboard

O detalhe do proxy consulta seu upstream enquanto estiver aberto. Sem um
`health_path`, a verificação testa a conexão TCP; quando o caminho está definido,
ela faz uma requisição HTTP usando o protocolo do upstream e mostra latência e
status HTTP. O painel também confirma a presença do container Nginx, do vhost e,
para proxies HTTPS, do certificado. O conteúdo do vhost é exibido somente para
leitura e sempre é resolvido a partir do proxy registrado, não de um caminho
fornecido pela requisição.

O tráfego usa o mesmo feed de acesso e agregador de timings dos sites, mas com a
chave `proxy:<nome>` para evitar colisões. Proxies gerenciados também expõem o
journal do unit na aba **Logs**.

> **Atenção (app-side):** `/redirect` precisa **existir** na API. Apps que não
> montam o `core/web.php` referenciam a rota sem a definir → 404. O lerd só
> roteia; a rota é responsabilidade da aplicação.

## TrustProxies / HTTPS no Laravel

O template fastcgi do proxy força `HTTPS on`, `X-Forwarded-Proto` e
`X-Forwarded-Host`, então o Laravel emite cookie `Secure` e gera URLs https —
**o cookie Sanctum same-origin funciona sem TrustProxies.**

Porém, sem TrustProxies, `$request->getClientIp()` retorna o IP do container
nginx, não o do cliente. Se a app faz rate-limit ou log por IP, configure em
`bootstrap/app.php`:

```php
->withMiddleware(function (Middleware $middleware) {
    $middleware->trustProxies(at: '*'); // ou a sub-rede do container
})
```

Verifique também que `SESSION_SECURE_COOKIE` é coerente com o `secured` do proxy.

## Reversão

`lerd proxy rm` ou `lerd proxy edit --path ""` zera as chaves de API-base da SPA
(grava `URL_API=`). O lerd não conhece a URL de dev original; reconfigure
manualmente (ex.: `URL_API=http://localhost:8000`) para rodar a SPA fora do proxy.

## Acesso pela LAN

O subtree `/api/proxies` do dashboard é **loopback-only**: mesmo com
`lan:expose` e `remote-control on` e credenciais válidas, um cliente da LAN
recebe 403. Um proxy em managed mode cria um container que monta um diretório
qualquer do host e roda um comando qualquer nele, ou seja, mais do que o
`/api/sites/link` que já era restrito por esse motivo. Criar e editar proxies
segue funcionando normalmente pela máquina local e pela CLI.

Os valores que chegam ao quadlet e ao vhost (`cmd`, `node_version`, `path`,
`name`, domínios) são validados na gravação **e** de novo na hora de gerar o
unit, porque o `proxies.yaml` é lido sem validação e uma edição manual não pode
injetar diretiva no unit gerado.
