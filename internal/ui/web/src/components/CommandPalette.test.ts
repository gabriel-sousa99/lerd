import { render, screen } from '@testing-library/svelte';
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import CommandPalette from './CommandPalette.svelte';
import { paletteOpen } from '$stores/commandPalette';
import { presets } from '$stores/presets';
import { services } from '$stores/services';
import { serviceIcons } from '$stores/serviceIcons';
import { sites } from '$stores/sites';

const MARK = '<svg viewBox="0 0 24 24"><path d="M3 3h18v18H3z"/></svg>';

describe('CommandPalette service marks', () => {
  const realFetch = globalThis.fetch;

  beforeEach(() => {
    globalThis.fetch = vi.fn(async () => {
      throw new Error('offline');
    }) as unknown as typeof fetch;
    sites.set([]);
    presets.set([]);
    serviceIcons.set({});
    paletteOpen.set(true);
  });

  afterEach(() => {
    globalThis.fetch = realFetch;
    paletteOpen.set(false);
    services.set([]);
    serviceIcons.set({});
  });

  // A service row names the service and its version, the way a site row names
  // its domain and framework, so the mark rides beside the version too.
  it('draws the mark beside a service row', () => {
    serviceIcons.set({ redis: MARK });
    services.set([
      { name: 'redis', status: 'active', site_count: 0, version: '7.4', category: 'cache', color: '#ff4438' }
    ]);
    const { container } = render(CommandPalette);
    expect(screen.getByText('7.4')).toBeInTheDocument();
    expect(container.querySelector('.mark-glyph path')?.getAttribute('d')).toBe('M3 3h18v18H3z');
  });

  // The row it sits in is a line of text, so the mark matches the framework
  // mark a site row carries rather than the icon a card draws.
  it('sizes the mark to the text beside it', () => {
    serviceIcons.set({ redis: MARK });
    services.set([{ name: 'redis', status: 'active', site_count: 0, version: '7.4' }]);
    const { container } = render(CommandPalette);
    expect(container.querySelector('.mark-glyph')?.getAttribute('class')).toContain('w-3.5');
  });
});
