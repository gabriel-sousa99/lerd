import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { get } from 'svelte/store';

describe('docs store', () => {
  const realFetch = globalThis.fetch;

  beforeEach(() => {
    vi.resetModules();
  });

  afterEach(() => {
    globalThis.fetch = realFetch;
  });

  it('parses the documentation hash into a page and an anchor', async () => {
    const { parseDocsHash } = await import('./docs');

    expect(parseDocsHash('#docs')).toEqual({ route: '', anchor: '' });
    expect(parseDocsHash('#docs/usage/sites')).toEqual({ route: 'usage/sites', anchor: '' });
    expect(parseDocsHash('#docs/configuration#per-project')).toEqual({
      route: 'configuration',
      anchor: 'per-project'
    });
    expect(parseDocsHash('#docs/usage/sites/')).toEqual({ route: 'usage/sites', anchor: '' });
  });

  it('ignores hashes that are not the documentation', async () => {
    const { parseDocsHash } = await import('./docs');

    expect(parseDocsHash('#sites/myapp.test')).toBeNull();
    expect(parseDocsHash('#docsomething')).toBeNull();
    expect(parseDocsHash('')).toBeNull();
  });

  it('loads the index once and keeps it', async () => {
    const fetchMock = vi.fn(
      async () =>
        new Response(JSON.stringify({ pages: [{ title: 'Sites', route: 'usage/sites' }] }), {
          status: 200
        })
    );
    globalThis.fetch = fetchMock as unknown as typeof fetch;

    const { loadDocsIndex, docsIndex, docsIndexFailed } = await import('./docs');
    await loadDocsIndex();
    await loadDocsIndex();

    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(fetchMock).toHaveBeenCalledWith('/docs/index.json');
    expect(get(docsIndex)).toHaveLength(1);
    expect(get(docsIndexFailed)).toBe(false);
  });

  it('flags a failed index load', async () => {
    globalThis.fetch = vi.fn(async () => new Response('nope', { status: 500 })) as unknown as typeof fetch;

    const { loadDocsIndex, docsIndexFailed } = await import('./docs');
    await loadDocsIndex();

    expect(get(docsIndexFailed)).toBe(true);
  });

  it('caches a rendered page instead of refetching it', async () => {
    const fetchMock = vi.fn(
      async () =>
        new Response(JSON.stringify({ route: 'usage/sites', title: 'Sites', html: '<h1>Sites</h1>' }), {
          status: 200
        })
    );
    globalThis.fetch = fetchMock as unknown as typeof fetch;

    const { loadDocsPage } = await import('./docs');
    const first = await loadDocsPage('usage/sites');
    const second = await loadDocsPage('usage/sites');

    expect(first?.html).toBe('<h1>Sites</h1>');
    expect(second).toEqual(first);
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it('returns null for a page the daemon does not have', async () => {
    globalThis.fetch = vi.fn(async () => new Response('nope', { status: 404 })) as unknown as typeof fetch;

    const { loadDocsPage } = await import('./docs');
    expect(await loadDocsPage('usage/nope')).toBeNull();
  });

  it('searches only for a non-blank query', async () => {
    const fetchMock = vi.fn(
      async () => new Response(JSON.stringify({ results: [{ route: 'usage/sites' }] }), { status: 200 })
    );
    globalThis.fetch = fetchMock as unknown as typeof fetch;

    const { searchDocs } = await import('./docs');
    expect(await searchDocs('  ')).toEqual([]);
    expect(fetchMock).not.toHaveBeenCalled();

    expect(await searchDocs('site groups')).toHaveLength(1);
    expect(fetchMock).toHaveBeenCalledWith('/docs/search?q=site%20groups');
  });

  it('tracks the hash it is pointed at', async () => {
    const { docsLocation } = await import('./docs');

    location.hash = 'docs/features/mcp';
    window.dispatchEvent(new HashChangeEvent('hashchange'));
    expect(get(docsLocation)).toEqual({ route: 'features/mcp', anchor: '' });

    location.hash = 'sites';
    window.dispatchEvent(new HashChangeEvent('hashchange'));
    expect(get(docsLocation)).toBeNull();
  });
});
