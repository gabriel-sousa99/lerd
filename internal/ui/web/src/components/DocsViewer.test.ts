import { render, screen, waitFor, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import DocsViewer from './DocsViewer.svelte';

// The store reads the hash at import time and follows hashchange after that, so
// tests point the location and announce it rather than reloading the module: a
// reset would give the component a second copy of the Svelte runtime.
function goTo(hash: string) {
  location.hash = hash;
  window.dispatchEvent(new HashChangeEvent('hashchange'));
}

const INDEX = {
  pages: [
    {
      title: 'Requirements',
      section: 'getting-started',
      section_label: 'Getting Started',
      slug: 'requirements',
      route: 'getting-started/requirements'
    },
    {
      title: 'Site Management',
      section: 'usage',
      section_label: 'Usage',
      slug: 'sites',
      route: 'usage/sites'
    }
  ]
};

const PAGE = {
  title: 'Site Management',
  section: 'usage',
  section_label: 'Usage',
  slug: 'sites',
  route: 'usage/sites',
  html: '<h1 id="site-management">Site Management</h1><p>Link a project.</p>'
};

function stubFetch() {
  return vi.fn(async (url: string) => {
    if (url === '/docs/index.json') return new Response(JSON.stringify(INDEX), { status: 200 });
    if (url.startsWith('/docs/page/')) return new Response(JSON.stringify(PAGE), { status: 200 });
    if (url.startsWith('/docs/search'))
      return new Response(
        JSON.stringify({ results: [{ ...PAGE, snippet: 'Link a project.' }] }),
        { status: 200 }
      );
    return new Response('nope', { status: 404 });
  });
}

describe('DocsViewer', () => {
  const realFetch = globalThis.fetch;

  beforeEach(() => {
    globalThis.fetch = stubFetch() as unknown as typeof fetch;
  });

  afterEach(() => {
    globalThis.fetch = realFetch;
    goTo('');
  });

  it('lists the pages grouped by section', async () => {
    goTo('docs');
    render(DocsViewer);

    expect(await screen.findByText('Getting Started')).toBeInTheDocument();
    expect(await screen.findByText('Usage')).toBeInTheDocument();
    expect(await screen.findByText('Requirements')).toBeInTheDocument();
  });

  it('renders the page the hash points at', async () => {
    goTo('docs/usage/sites');
    const { container } = render(DocsViewer);

    await waitFor(() => {
      expect(container.querySelector('.docs-prose')?.innerHTML).toContain('Link a project.');
    });
    expect(globalThis.fetch).toHaveBeenCalledWith('/docs/page/usage/sites');
  });

  it('navigates by rewriting the hash', async () => {
    goTo('docs');
    render(DocsViewer);

    await fireEvent.click(await screen.findByText('Site Management'));
    expect(location.hash).toBe('#docs/usage/sites');
  });

  it('replaces the list with search results', async () => {
    goTo('docs');
    render(DocsViewer);

    await screen.findByText('Requirements');
    await fireEvent.input(screen.getByRole('searchbox'), { target: { value: 'link' } });

    expect(await screen.findByText('Link a project.')).toBeInTheDocument();
    await waitFor(() => expect(screen.queryByText('Getting Started')).not.toBeInTheDocument());
  });

  it('reports a page it cannot load', async () => {
    globalThis.fetch = vi.fn(async (url: string) =>
      url === '/docs/index.json'
        ? new Response(JSON.stringify(INDEX), { status: 200 })
        : new Response('nope', { status: 404 })
    ) as unknown as typeof fetch;

    goTo('docs/usage/gone');
    render(DocsViewer);

    expect(await screen.findByText('This page could not be loaded.')).toBeInTheDocument();
  });
});
