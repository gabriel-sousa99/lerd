# Backlog de ideias (pré-brainstorm)

Itens levantados mas ainda não brainstormados. Cada um terá seu próprio
ciclo brainstorm → spec → plano quando priorizado.

## `lerd link` — auto-detecção de env, pular wizard (2026-06-01)

**Ideia:** ao rodar `lerd link` via CLI, se o projeto já tem variáveis de
ambiente configuradas para os recursos que o wizard normalmente pergunta
(banco de dados, fila, storage, etc.), **detectar automaticamente** e
**pular o wizard**, exibindo um resumo "tudo já configurado" — porém com uma
**opção para editar** caso o usuário queira ajustar.

**A explorar no brainstorm:**
- Quais recursos/variáveis o wizard atual do `lerd link` pergunta hoje
  (mapear o fluxo em `internal/cli` / o comando link e o setup de presets).
- Critério de "já configurado" por recurso (presença e validade da var? ex.
  `DB_CONNECTION` + `DB_HOST`/sqlite; `QUEUE_CONNECTION`; `FILESYSTEM_DISK`).
- UX do "pular com resumo + editar" (flag não-interativa? prompt único de
  confirmação? `--reconfigure` para forçar o wizard?).
- Interação com presets do lerd (oracle-xe, mysql, redis, etc.) e com o
  `.env` existente (não sobrescrever config válida do usuário).

**Restrição herdada (org):** qualquer toque em banco respeita as regras
Oracle — nada de DDL destrutivo; default sqlite em testes.
