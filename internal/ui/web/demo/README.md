# Backend-free UI demo

Builds the real lerd web UI (`../src`) as a standalone bundle that runs with
**no daemon**, for embedding in the docs landing page (`docs/index.md` iframes
`/demo/`).

How it works:

- `stubs.ts` patches `window.fetch` and `window.WebSocket` before the app
  loads, so every store reads from the JSON in `fixtures/` instead of the
  daemon. It is imported first in `main.ts`.
- `fixtures/` are real `/api/*` responses captured from a running daemon, then
  sanitized (site names, domains, paths and app names swapped for demo values).
  Regenerate the same way: snapshot the endpoints, scrub identifying fields.
- The theme follows the system preference, so a capture of the demo has to pin
  the browser to dark rather than assume it.
- Service, framework and worker marks come from `fixtures/*-marks.json` and
  `fixtures/service-icons.json`, captured off a running daemon the same way
  every other fixture is. Recapture them when the stores publish new artwork.

Build it:

```
npm run build:demo   # → docs/public/demo/
```

Rebuild whenever the UI components change so the demo stays in sync. The output
is static; the docs site just serves it from `public/demo/`.
