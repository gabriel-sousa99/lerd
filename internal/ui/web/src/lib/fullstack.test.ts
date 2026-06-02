import { describe, it, expect } from 'vitest';
import { buildApiRoutes, defaultApiPaths, suggestUnifiedDomain } from './fullstack';

describe('buildApiRoutes', () => {
  it('default paths for a site target', () => {
    const routes = buildApiRoutes({ mode: 'site', site: 'retencao-api', port: 0, paths: [] });
    expect(routes.map((r) => r.path)).toEqual(defaultApiPaths());
    expect(routes[0]).toEqual({ path: '/api', site: 'retencao-api' });
  });
  it('port target with custom paths', () => {
    const routes = buildApiRoutes({ mode: 'port', site: '', port: 8000, paths: ['/api'] });
    expect(routes).toEqual([{ path: '/api', upstream_port: 8000 }]);
  });
  it('site mode without a site name yields no routes', () => {
    expect(buildApiRoutes({ mode: 'site', site: '', port: 0, paths: [] })).toEqual([]);
  });
  it('port mode without a port yields no routes', () => {
    expect(buildApiRoutes({ mode: 'port', site: '', port: 0, paths: [] })).toEqual([]);
  });
});

describe('suggestUnifiedDomain', () => {
  it('strips -api suffix and appends .localhost', () => {
    expect(suggestUnifiedDomain('retencao-api')).toBe('retencao.localhost');
  });
  it('appends .localhost to a plain name', () => {
    expect(suggestUnifiedDomain('foo')).toBe('foo.localhost');
  });
  it('handles a full -api.localhost domain', () => {
    expect(suggestUnifiedDomain('retencao-api.localhost')).toBe('retencao.localhost');
  });
});
