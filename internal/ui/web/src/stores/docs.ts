import { writable, get } from 'svelte/store';

// The documentation ships inside the binary, so the dashboard reads it from the
// daemon rather than lerd.sh and a machine with no internet keeps it. The URLs
// live outside /api on purpose: same-origin static-looking GETs are what the
// service worker caches, which is what carries the pages offline.

export interface DocsPageMeta {
  title: string;
  section: string;
  section_label: string;
  slug: string;
  route: string;
}

export interface DocsPage extends DocsPageMeta {
  html: string;
}

export interface DocsSearchResult {
  title: string;
  section: string;
  slug: string;
  route: string;
  snippet: string;
}

export const DEFAULT_DOCS_ROUTE = 'getting-started/requirements';

export const docsIndex = writable<DocsPageMeta[]>([]);
export const docsIndexFailed = writable(false);

export async function loadDocsIndex(): Promise<void> {
  if (get(docsIndex).length > 0) return;
  try {
    const res = await fetch('/docs/index.json');
    if (!res.ok) throw new Error(String(res.status));
    const body = (await res.json()) as { pages?: DocsPageMeta[] };
    docsIndex.set(body.pages ?? []);
    docsIndexFailed.set(false);
  } catch {
    docsIndexFailed.set(true);
  }
}

const rendered = new Map<string, DocsPage>();

// loadDocsPage fetches a page's rendered HTML, keeping it for the session: the
// pages are baked into the running binary and cannot change under us.
export async function loadDocsPage(route: string): Promise<DocsPage | null> {
  const cached = rendered.get(route);
  if (cached) return cached;
  try {
    const res = await fetch('/docs/page/' + route);
    if (!res.ok) return null;
    const page = (await res.json()) as DocsPage;
    rendered.set(route, page);
    return page;
  } catch {
    return null;
  }
}

export async function searchDocs(query: string): Promise<DocsSearchResult[]> {
  if (!query.trim()) return [];
  try {
    const res = await fetch('/docs/search?q=' + encodeURIComponent(query));
    if (!res.ok) return [];
    const body = (await res.json()) as { results?: DocsSearchResult[] };
    return body.results ?? [];
  } catch {
    return [];
  }
}

export interface DocsLocation {
  route: string;
  anchor: string;
}

// parseDocsHash splits '#docs/usage/sites#env-overrides' into the page and the
// heading inside it. Returns null for any hash that isn't the documentation.
export function parseDocsHash(hash: string): DocsLocation | null {
  const h = hash.startsWith('#') ? hash.slice(1) : hash;
  if (h !== 'docs' && !h.startsWith('docs/')) return null;
  const rest = h === 'docs' ? '' : h.slice('docs/'.length);
  const [route, anchor = ''] = rest.split('#');
  return { route: route.replace(/^\/+|\/+$/g, ''), anchor };
}

export const docsLocation = writable<DocsLocation | null>(
  typeof location === 'undefined' ? null : parseDocsHash(location.hash)
);

if (typeof window !== 'undefined') {
  window.addEventListener('hashchange', () => docsLocation.set(parseDocsHash(location.hash)));
}

export function goToDocsPage(route: string, anchor = '') {
  location.hash = 'docs' + (route ? '/' + route : '') + (anchor ? '#' + anchor : '');
}

// docsSiteURL is the same page on lerd.sh, for the overlay's open-in-a-tab link.
export function docsSiteURL(route: string): string {
  return 'https://lerd.sh/' + route;
}
