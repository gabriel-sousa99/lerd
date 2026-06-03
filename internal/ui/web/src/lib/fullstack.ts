import type { Route } from '$stores/proxies';

// defaultApiPaths espelha defaultAPIPaths() do backend (internal/cli/proxy.go):
// convenções Laravel + rotas de auth do unimedvr/core (/redirect, /authenticate
// são as rotas web no root do fluxo SSO 401 → /redirect → /authenticate). Manter
// em sincronia com o Go — divergir faz o dashboard criar proxies sem as rotas SSO.
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
