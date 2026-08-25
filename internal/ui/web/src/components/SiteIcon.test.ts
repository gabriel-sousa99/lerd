import { render } from '@testing-library/svelte';
import { describe, it, expect, beforeEach } from 'vitest';
import SiteIcon from './SiteIcon.svelte';
import { frameworkMarks } from '$stores/frameworkMarks';
import type { Site } from '$stores/sites';

const MARK = '<svg viewBox="0 0 24 24"><path d="M3 3h18v18H3z"/></svg>';

function site(extra: Partial<Site> = {}): Site {
  return { domain: 'example.test', fpm_running: true, ...extra } as Site;
}

beforeEach(() => {
  frameworkMarks.set({ laravel: { svg: MARK, color: '#ff2d20' } });
});

describe('SiteIcon', () => {
  // A site with nothing of its own to show used to be a bare status dot, which
  // said only running or not; its framework is the more useful identity, and
  // the row's own indicators already carry the state.
  it('draws the framework mark when the site has no favicon', () => {
    const { container } = render(SiteIcon, { props: { site: site({ framework: 'laravel' }) } });
    expect(container.querySelector('.mark-ink path')?.getAttribute('d')).toBe('M3 3h18v18H3z');
    expect(container.querySelector('.mark-ink')?.getAttribute('style')).toContain('--mark-tint');
  });

  // The site's own icon still wins: it is more specific than its framework's.
  it('prefers the favicon over the framework mark', () => {
    const { container } = render(SiteIcon, {
      props: { site: site({ framework: 'laravel', has_favicon: true }) }
    });
    expect(container.querySelector('img')).not.toBeNull();
    expect(container.querySelector('.mark-ink')).toBeNull();
  });

  // A framework the store ships no mark for, or no framework at all, keeps the
  // dot rather than an empty slot.
  it('falls back to the status dot with no favicon and no mark', () => {
    const { container } = render(SiteIcon, { props: { site: site({ framework: 'slim' }) } });
    expect(container.querySelector('.mark-ink')).toBeNull();
    expect(container.querySelector('span[class*="bg-"]')).not.toBeNull();
  });

  it('keeps the container glyph for a custom-container site', () => {
    const { container } = render(SiteIcon, {
      props: { site: site({ framework: 'laravel', custom_container: true }) }
    });
    expect(container.querySelector('.mark-ink')).toBeNull();
    expect(container.querySelector('svg')).not.toBeNull();
  });
});
