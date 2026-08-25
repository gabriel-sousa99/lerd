import { render } from '@testing-library/svelte';
import { describe, it, expect, beforeEach } from 'vitest';
import NavRail from './NavRail.svelte';
import { services } from '$stores/services';
import { serviceIcons } from '$stores/serviceIcons';
import { accessMode } from '$stores/accessMode';
import { dashboardOpen } from '$stores/dashboard';

const MARK = '<svg viewBox="0 0 24 24"><path d="M3 3h18v18H3z"/></svg>';

beforeEach(() => {
  globalThis.fetch = (async () =>
    new Response(JSON.stringify({}), { status: 200 })) as unknown as typeof fetch;
  accessMode.set({ localControl: true, lanExposed: false, checked: true });
  dashboardOpen.set(null);
  serviceIcons.set({ mailpit: MARK });
  services.set([
    {
      name: 'mailpit',
      status: 'active',
      site_count: 0,
      category: 'mail',
      color: '#0f6cbd',
      dashboard: 'http://localhost:8025'
    }
  ]);
});

describe('NavRail', () => {
  it('draws a dashboard launcher with the mark its preset ships', () => {
    const { container } = render(NavRail);
    expect(container.querySelector('.mark-glyph path')?.getAttribute('d')).toBe('M3 3h18v18H3z');
  });

  // The rail is one column of icons that share a hover and active colour; a
  // launcher painting itself in the brand tone would opt out of both.
  it('leaves the mark the rail colour rather than the brand tone', () => {
    const { container } = render(NavRail);
    const mark = container.querySelector('.mark-glyph') as HTMLElement;
    expect(mark.closest('.mark-brand')).toBeNull();
    expect(mark.closest('[style*="--mark-tint"]')).toBeNull();
  });
});
