# Omarchy bar widget

[lerd Glance](https://github.com/lerd-env/lerd-omarchy-glance) is a plugin for the
[Omarchy](https://omarchy.org) Quattro shell that puts the state of your environment in the
desktop bar, so the answer to "is anything broken" is already on screen.

It is a separate, optional install. lerd does not need it and it does not need anything
from lerd beyond the dashboard already running. If you are setting lerd up on Omarchy for
the first time, start with [PHP development on Omarchy](../getting-started/omarchy.md).

```bash
omarchy plugin add https://github.com/lerd-env/lerd-omarchy-glance.git --enable
```

---

## What it shows

The bar carries lerd's mark on its own. A coloured dot appears on it only when something
needs attention, yellow for a warning and red when nginx is down, so a healthy environment
stays quiet. Hovering gives you the counts and the list of problems without opening
anything.

Clicking opens a panel:

```
lerd                        1.33.1   ← version, with a dot when an update is out
CPU        0.34%   Memory   824 MB   ← totals across every lerd container
9 containers      14.4% of 5.6 GB
─────────────────────────────────
Sites                          1/1
Services                       3/3
 Queue                         0/1   ← one row per worker type, with its glyph
 Schedule                      0/1
nginx                            🟢
.test resolution                 🟢
watcher                          🟢
PHP                    🟢 8.4 🟢 8.5  ← the default in bold
─────────────────────────────────
🟢 mysql            v8.4.11    3306
🟢 phpmyadmin        latest     8080
🟢 typesense          v30.2     8108
─────────────────────────────────
Reclaimable                  57 MB   ← only when there is space to reclaim
[ Clean up ]
[ Open dashboard ]
```

A "Needs attention" section appears above the buttons when something is wrong, listing it
in plain words: a failed service, a worker that died, DNS that stopped resolving.

---

## What it reads

The widget polls `/api/status`, `/api/sites`, `/api/services`, `/api/workers/health`,
`/api/stats`, `/api/version` and `/api/disk` on `127.0.0.1:7073`, every 30 seconds, and
every 5 seconds while the panel is open. Clicking the widget forces a refresh.

Because it is a loopback client it needs no credentials, the same way the dashboard in your
browser does not. Nothing is sent anywhere else.

Workers are read from lerd's own health verdict rather than inferred from a site's
`has_queue_worker` style flags, so a queue worker you simply never started is not reported
as broken. Paused sites are left out of the counts.

---

## What it can change

Two buttons, and nothing happens without pressing one.

**Open dashboard** runs `xdg-open http://lerd.localhost`.

**Clean up** appears only when lerd reports reclaimable disk space, shows how much, and
posts to `/api/disk`, which is the same cleanup [`lerd cleanup`](../reference/commands.md)
performs. lerd re-inspects the host and applies its own fresh plan, so the button never
asks podman to remove an image that has since become live.

---

## Requirements

Omarchy Quattro, and lerd running on the same machine. There is no configuration. If you
serve the dashboard on another port, change the `endpoint` property at the top of
`BarWidget.qml`.

To remove it:

```bash
omarchy plugin remove sh.lerd.glance
```
