import { render } from '@testing-library/svelte';
import { describe, it, expect, beforeEach } from 'vitest';
import SitesWidget from './SitesWidget.svelte';
import { sites, sitesLoaded, type Site } from '$stores/sites';
import { accessMode } from '$stores/accessMode';

function site(over: Partial<Site> = {}): Site {
  return { domain: 'app.test', name: 'app', fpm_running: true, ...over } as Site;
}

// The tile's own title, told apart from the card title by its darker ink.
function tileTitles(container: HTMLElement): string[] {
  return [...container.querySelectorAll('.font-semibold.text-gray-900')].map(
    (n) => n.textContent?.trim() ?? ''
  );
}

describe('SitesWidget', () => {
  beforeEach(() => {
    sites.set([]);
    sitesLoaded.set(true);
    accessMode.set({ localControl: true, lanExposed: false, checked: true });
  });

  it('points at lerd park when no site is linked', () => {
    const { getByText } = render(SitesWidget);
    expect(getByText('lerd park')).toBeTruthy();
  });

  // The widget owns the selection, not the markup: every row is the same tile
  // the Sites tab draws.
  it('renders one compact site card per site', () => {
    sites.set([site(), site({ domain: 'blog.test', name: 'blog' })]);
    const { container } = render(SitesWidget);
    expect(container.querySelectorAll('.rounded-lg.p-2\\.5')).toHaveLength(2);
    expect(tileTitles(container)).toEqual(['blog.test', 'app.test']);
  });

  it('carries the app name and the framework subline the tile builds', () => {
    sites.set([site({ app_name: 'My Shop', framework_label: 'Laravel', php_version: '8.4' })]);
    const { getByText } = render(SitesWidget);
    expect(getByText('My Shop')).toBeTruthy();
    expect(getByText('app.test · Laravel · PHP 8.4')).toBeTruthy();
  });

  // Registry order is oldest first, so the newest linked project leads.
  it('reverses registry order and drops paused sites', () => {
    sites.set([
      site({ domain: 'one.test', name: 'one' }),
      site({ domain: 'two.test', name: 'two', paused: true }),
      site({ domain: 'three.test', name: 'three' })
    ]);
    const { container } = render(SitesWidget);
    expect(tileTitles(container)).toEqual(['three.test', 'one.test']);
  });

  it('leaves the store order alone while reversing its own view', () => {
    const rows = [site({ domain: 'one.test' }), site({ domain: 'two.test' })];
    sites.set(rows);
    render(SitesWidget);
    expect(rows.map((s) => s.domain)).toEqual(['one.test', 'two.test']);
  });

  it('keeps the failing worker dot the hand rolled row used to draw', () => {
    sites.set([site({ queue_failing: true })]);
    const { container } = render(SitesWidget);
    expect(container.querySelector('.bg-red-500')).toBeTruthy();
  });

  it('turns the card critical while a worker is failing', () => {
    sites.set([site({ queue_failing: true })]);
    const { container } = render(SitesWidget);
    expect((container.firstElementChild as HTMLElement).className).toContain('border-l-red-500');
  });

  it('counts running sites in the summary pill', () => {
    sites.set([site(), site({ domain: 'blog.test', fpm_running: false })]);
    const { getByText } = render(SitesWidget);
    expect(getByText('1/2 running')).toBeTruthy();
  });

  it('hides the link button without local control', () => {
    accessMode.set({ localControl: false, lanExposed: false, checked: true });
    sites.set([site()]);
    const { queryByText, getByText } = render(SitesWidget);
    expect(queryByText('Link site')).toBeNull();
    expect(getByText('Open sites →')).toBeTruthy();
  });
});
