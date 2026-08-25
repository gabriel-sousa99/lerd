import type { Route } from '$stores/proxies';

// defaultApiPaths espelha defaultAPIPaths() do backend (internal/cli/proxy.go):
// convenções Laravel + rotas de auth comuns (/redirect, /authenticate são as
// rotas web no root do fluxo SSO 401 → /redirect → /authenticate). Manter em
// sincronia com o Go — divergir faz o dashboard criar proxies sem as rotas SSO.
export function defaultApiPaths(): string[] {
  return [
    '/api',
    '/sanctum',
    '/broadcasting',
    '/storage',
    '/redirect',
    '/authenticate',
    '/login',
    '/logout',
    '/up',
  ];
}

export interface ApiTargetInput {
  mode: 'site' | 'port';
  site: string;
  port: number;
  paths: string[];
}

export interface EditableRoute {
  path: string;
  mode: 'site' | 'port';
  site: string;
  port: number;
  host: string;
}

export function editableRoutesFrom(routes: Route[]): EditableRoute[] {
  return routes.map((route) => ({
    path: route.path,
    mode: route.site ? 'site' : 'port',
    site: route.site ?? '',
    port: route.upstream_port ?? 8000,
    host: route.upstream_host ?? ''
  }));
}

export function routesFromEditable(rows: EditableRoute[]): Route[] {
  return rows.map((row) => {
    const path = row.path.trim();
    if (row.mode === 'site') return { path, site: row.site.trim() };
    const host = row.host.trim();
    return {
      path,
      upstream_port: row.port,
      ...(host && host !== 'host.containers.internal' ? { upstream_host: host } : {})
    };
  });
}

// buildApiRoutes turns the modal's fullstack inputs into Route[]. Returns an
// empty array when the API target is incomplete (no site / no port).
export function buildApiRoutes(input: ApiTargetInput): Route[] {
  const hasSite = input.mode === 'site' && input.site.trim() !== '';
  const hasPort = input.mode === 'port' && input.port > 0;
  if (!hasSite && !hasPort) return [];
  const paths = input.paths.length > 0 ? input.paths : defaultApiPaths();
  return paths.map((p) =>
    hasSite ? { path: p, site: input.site.trim() } : { path: p, upstream_port: input.port }
  );
}

// suggestUnifiedDomain maps "retencao-api" → "retencao.localhost".
export function suggestUnifiedDomain(name: string): string {
  const base = name.replace(/\.localhost$/, '').replace(/-api$/, '');
  return `${base}.localhost`;
}
