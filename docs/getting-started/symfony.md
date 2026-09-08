# Symfony walkthrough

End-to-end: from `lerd install` to a Symfony app running on `https://myapp.test` with Doctrine, MySQL, and a Messenger worker.

::: info Prerequisites
You've already run `lerd install` once on this machine. If not, see [Installation](installation.md).
:::

::: tip Drive it from your AI assistant
Run `lerd mcp:enable-global` once and your AI assistant (Claude Code, Cursor, Junie, Codex, Gemini, Copilot, Antigravity, Windsurf) can call every command below through the grouped MCP tools: `framework` `action: "project_new"`, `site` `action: "link"`, `env` `action: "setup"`, `framework` `action: "setup"`, `db` `action: "create"`, `site` `action: "tls_enable"`, `worker`, etc. See [AI Integration](../features/mcp.md).
:::

---

## 1. Create the project

::: code-group

```bash [lerd new]
cd ~/Lerd
lerd new myapp --framework=symfony
# runs: composer create-project symfony/skeleton:^8.0 ./myapp
```

```bash [composer]
cd ~/Lerd
composer create-project symfony/skeleton myapp
```

```bash [existing repo]
cd ~/Lerd
git clone git@github.com:you/myapp.git
```

:::

---

## 2. Register the site

```bash
cd myapp
lerd link
```

`lerd link` detects Symfony (via `symfony.lock` or the composer package), assigns `http://myapp.test`, and sets the document root to `public/`. Detection also resolves the major version from `composer.lock` and pulls the matching definition, `symfony@8` for a current skeleton, from the [framework store](../usage/frameworks.md#framework-store). There is nothing to register by hand.

---

## 3. Configure PHP, Node, database, services

```bash
lerd init
```

```
? PHP version: 8.5
? Node version (clear to follow the lerd default instead of pinning): 22
? Enable HTTPS? Yes
? Database: mysql
? Services: [mailpit]
? Workers to auto-start: [messenger]
Saved .lerd.yaml
```

The wizard discovers `messenger` as an available worker because the definition declares it and the `check: composer: symfony/messenger` rule matches your project. A project that also pulls in `symfony/scheduler` gets a `scheduler` worker beside it on the same terms.

---

## 4. Bootstrap the project

```bash
lerd setup
```

```
? Select setup steps to run:
  ◉ composer install
  ◉ npm ci                       # only if package.json exists
  ◉ lerd env                     # injects DATABASE_URL, MAILER_DSN, DEFAULT_URI
  ◉ Run migrations               # from the definition's setup block
  ◯ Load fixtures
  ◉ Clear cache
  ◉ Install assets
  ◉ lerd secure                  # mkcert TLS for myapp.test
  ◉ messenger:start
  ◉ lerd open
```

"Run migrations", "Load fixtures", "Clear cache" and "Install assets" come from the `setup:` block in `symfony@8`. Lerd surfaces them automatically and respects the `check:` rules, so migrations only appear with `doctrine/doctrine-migrations-bundle` installed and fixtures only with `doctrine/doctrine-fixtures-bundle`.

When it finishes, `https://myapp.test` opens in your browser and `lerd-messenger-myapp` is running as a systemd user service.

---

## 5. Verify

```bash
lerd status
```

```bash
# Tail messenger logs
journalctl --user -u lerd-messenger-myapp -f
```

App logs show up in the [Web UI](../features/web-ui.md) **App Logs** tab. The definition already points at `var/log/*.log` and parses them as Monolog, so a standard project needs nothing; a [user overlay](../usage/framework-definitions.md) is where you would change the paths or the parser.

---

## What just happened

| Command | What it did |
|---|---|
| `lerd link` | Detected Symfony, fetched `symfony@8` from the store, assigned `myapp.test`, set document root to `public/` |
| `lerd init` | Wrote `.lerd.yaml` with PHP, Node, MySQL, Mailpit, messenger |
| `lerd env` (via setup) | Wrote `DATABASE_URL=mysql://root:lerd@lerd-mysql:3306/myapp` and `MAILER_DSN=smtp://lerd-mailpit:1025` into `.env.local`, seeded from the committed `.env` |
| `lerd secure` (via setup) | Issued mkcert cert, set `DEFAULT_URI=https://myapp.test` |
| Doctrine migrations + cache:clear | Ran via the framework's `setup:` block |
| `lerd worker start messenger` (via setup) | Launched `lerd-messenger-myapp` |

---

## FrankenPHP / Symfony Runtime (optional)

By default your site runs on the shared PHP-FPM stack. To run it on a per-site FrankenPHP container instead (useful for testing under the long-running worker model Symfony Runtime provides):

```bash
lerd runtime frankenphp            # classic mode
lerd runtime frankenphp --worker   # Symfony Runtime, keeps the kernel in memory
lerd runtime fpm                   # back to shared PHP-FPM
```

Worker mode needs `composer require runtime/frankenphp-symfony`. Lerd starts the FrankenPHP container with `--watch` so edits to controllers and config reload within a second or two without restarting the worker manually. See the [FrankenPHP runtime](../features/frankenphp.md) page for limitations.

---

## Next steps

- [Frameworks & Workers](../usage/frameworks.md): add custom workers, customise log paths, define more setup steps
- [Database](../usage/database.md): `lerd db:import`, `lerd db:shell`, switching to Postgres
- [Services](../usage/services.md): start Meilisearch, RustFS (S3), custom services
- [HTTPS](../features/https.md): how `lerd secure` works under the hood
- [AI Integration (MCP)](../features/mcp.md): drive lerd from Claude Code, Cursor, Junie, etc.
