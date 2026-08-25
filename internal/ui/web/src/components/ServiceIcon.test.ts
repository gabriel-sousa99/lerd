import { render } from '@testing-library/svelte';
import { describe, it, expect, beforeEach } from 'vitest';
import ServiceIcon from './ServiceIcon.svelte';
import { presets } from '$stores/presets';
import { services } from '$stores/services';
import { serviceIcons } from '$stores/serviceIcons';

function box(container: HTMLElement): HTMLElement {
  return container.firstElementChild as HTMLElement;
}

const MARK = '<svg viewBox="0 0 24 24"><path d="M3 3h18v18H3z"/></svg>';

beforeEach(() => {
  presets.set([{ name: 'mysql', category: 'databases', icon: 'database' }]);
  services.set([]);
  serviceIcons.set({});
});

describe('ServiceIcon', () => {
  // A caller with only a name resolves the declared metadata through the registry.
  it('tints from the metadata the named preset declares', () => {
    const { container } = render(ServiceIcon, { props: { name: 'mysql' } });
    expect(box(container).className).toContain('indigo');
  });

  it('tints as other when nothing declares the name', () => {
    const { container } = render(ServiceIcon, { props: { name: 'totally-unknown' } });
    expect(box(container).className).toContain('gray');
  });

  it('prefers an explicit category over the registry', () => {
    const { container } = render(ServiceIcon, { props: { name: 'mysql', category: 'cache' } });
    expect(box(container).className).toContain('amber');
    expect(box(container).className).not.toContain('indigo');
  });

  it('draws a w-9 box that scales with its card by default', () => {
    const { container } = render(ServiceIcon, { props: { name: 'redis' } });
    expect(box(container).className).toContain('w-9');
    expect(box(container).className).toContain('group-hover:scale-105');
    expect(container.querySelector('svg')?.getAttribute('class')).toContain('w-5');
  });

  it('draws a w-8 box that holds still when compact', () => {
    const { container } = render(ServiceIcon, { props: { name: 'redis', compact: true } });
    expect(box(container).className).toContain('w-8');
    expect(box(container).className).not.toContain('group-hover:scale-105');
    expect(container.querySelector('svg')?.getAttribute('class')).toContain('w-4');
  });

  it('drops the plate and draws a bigger mark when bare', () => {
    const { container } = render(ServiceIcon, { props: { name: 'mysql', bare: true } });
    expect(box(container).className).not.toContain('rounded-lg');
    expect(box(container).className).not.toContain('bg-indigo');
    expect(box(container).className).toContain('text-indigo');
    expect(container.querySelector('svg')?.getAttribute('class')).toContain('w-7');
  });

  it('takes the brand tone as ink when bare, with no tinted background', () => {
    serviceIcons.set({ mysql: MARK });
    presets.set([{ name: 'mysql', category: 'databases', icon: 'database', color: '#e02419' }]);
    const { container } = render(ServiceIcon, { props: { name: 'mysql', bare: true } });
    expect(box(container).className).toContain('mark-brand');
    expect(box(container).className).not.toContain('mark-tint');
    expect(box(container).getAttribute('style')).toContain('--mark-tint: #e02419');
  });

  // mark-ink sizes and fills whatever svg sits directly inside it, which is the
  // store mark's contract, not the built-in glyph's: a branded preset shipping
  // no mark once drew its outline glyph filled and unbounded.
  it('leaves the built-in glyph its own box and stroke when bare and branded', () => {
    presets.set([{ name: 'rustfs', category: 'storage', icon: 'storage', color: '#0196d0' }]);
    const { container } = render(ServiceIcon, { props: { name: 'rustfs', bare: true } });
    expect(box(container).className).not.toContain('mark-ink');
    const svg = container.querySelector('svg')!;
    expect(svg.getAttribute('class')).toContain('w-7');
    expect(svg.getAttribute('fill')).toBe('none');
  });

  // Beside a line of text the mark answers to the text, not to the icons in a
  // card, so it draws at the size a framework's own inline mark does.
  it('draws a bare mark at text size when inline', () => {
    serviceIcons.set({ mysql: MARK });
    const { container } = render(ServiceIcon, {
      props: { name: 'mysql', bare: true, inline: true }
    });
    expect(container.querySelector('.mark-glyph')?.getAttribute('class')).toContain('w-3.5');
  });

  it('draws a bare mark at icon size when compact', () => {
    serviceIcons.set({ mysql: MARK });
    const { container } = render(ServiceIcon, { props: { name: 'mysql', bare: true, compact: true } });
    expect(container.querySelector('.mark-glyph')?.getAttribute('class')).toContain('w-5');
  });

  // In a chrome strip the mark is one of a row of icons and has to answer to
  // the same hover and active states they do, so it takes no tone of its own.
  it('inherits the surrounding colour when the tint is off', () => {
    serviceIcons.set({ mysql: MARK });
    presets.set([{ name: 'mysql', category: 'databases', icon: 'database', color: '#e02419' }]);
    const { container } = render(ServiceIcon, {
      props: { name: 'mysql', bare: true, compact: true, tint: false }
    });
    expect(container.querySelector('.mark-glyph path')).not.toBeNull();
    expect(box(container).className).not.toContain('mark-brand');
    expect(box(container).className).not.toContain('indigo');
    expect(box(container).getAttribute('style') || '').not.toContain('--mark-tint');
  });

  it('renders the service glyph', () => {
    const { container } = render(ServiceIcon, { props: { name: 'mysql' } });
    expect(container.querySelector('svg')?.innerHTML.length).toBeGreaterThan(0);
  });

  // A preset that ships its own mark and colour draws neither from the binary.
  it('prefers the mark the store shipped, tinted by the declared colour', () => {
    serviceIcons.set({ mysql: MARK });
    presets.set([{ name: 'mysql', category: 'databases', icon: 'database', color: '#e02419' }]);
    const { container } = render(ServiceIcon, { props: { name: 'mysql' } });
    expect(container.querySelector('.mark-glyph path')?.getAttribute('d')).toBe('M3 3h18v18H3z');
    expect(box(container).className).toContain('mark-tint');
    expect(box(container).className).not.toContain('indigo');
    expect(box(container).getAttribute('style')).toContain('--mark-tint: #e02419');
    expect(box(container).getAttribute('style')).toContain('--mark-tint-dark: #e02419');
  });

  // An admin tool is the front end of the engine it administers, so when it
  // ships no mark of its own it draws that engine's, in that engine's colour
  // rather than its own category tone.
  it('draws the mark of what it administers when it ships none itself', () => {
    serviceIcons.set({ postgres: MARK });
    presets.set([
      { name: 'postgres', category: 'databases', icon: 'elephant', color: '#336791' },
      { name: 'pgadmin', category: 'admin', icon: 'elephant', admin_for: ['postgres'] }
    ]);
    const { container } = render(ServiceIcon, { props: { name: 'pgadmin' } });
    expect(container.querySelector('.mark-glyph path')?.getAttribute('d')).toBe('M3 3h18v18H3z');
    expect(box(container).getAttribute('style')).toContain('--mark-tint: #336791');
    expect(box(container).className).not.toContain('emerald');
  });

  // An admin tool with a mark of its own keeps it: phpMyAdmin is not MySQL.
  it('keeps an admin tool the mark it ships itself', () => {
    const OWN = '<svg viewBox="0 0 24 24"><path d="M1 1h2v2H1z"/></svg>';
    serviceIcons.set({ phpmyadmin: OWN, mysql: MARK });
    presets.set([
      { name: 'mysql', category: 'databases', icon: 'database', color: '#00758f' },
      { name: 'phpmyadmin', category: 'admin', icon: 'database', color: '#6c78af', admin_for: ['mysql'] }
    ]);
    const { container } = render(ServiceIcon, { props: { name: 'phpmyadmin' } });
    expect(container.querySelector('.mark-glyph path')?.getAttribute('d')).toBe('M1 1h2v2H1z');
    expect(box(container).getAttribute('style')).toContain('--mark-tint: #6c78af');
  });

  // Nothing administered has a mark either: the declared glyph still draws.
  it('falls back to its own glyph when nothing it administers has a mark', () => {
    serviceIcons.set({});
    presets.set([{ name: 'pgadmin', category: 'admin', icon: 'elephant', admin_for: ['postgres'] }]);
    const { container } = render(ServiceIcon, { props: { name: 'pgadmin' } });
    expect(container.querySelector('.mark-glyph')).toBeNull();
    expect(box(container).className).toContain('emerald');
  });

  // A versioned member carries no mark of its own; it draws its family's.
  it('resolves the mark through the preset a service was installed from', () => {
    serviceIcons.set({ mariadb: MARK });
    services.set([
      { name: 'mariadb-11-8', status: 'active', site_count: 0, preset: 'mariadb', category: 'databases' }
    ]);
    const { container } = render(ServiceIcon, { props: { name: 'mariadb-11-8' } });
    expect(container.querySelector('.mark-glyph path')).not.toBeNull();
  });

  // A bundled preset, or a cache written before icons existed, keeps rendering.
  it('falls back to the declared glyph and category tint with no store mark', () => {
    presets.set([{ name: 'mysql', category: 'databases', icon: 'database' }]);
    const { container } = render(ServiceIcon, { props: { name: 'mysql' } });
    expect(container.querySelector('.mark-glyph')).toBeNull();
    expect(box(container).className).toContain('indigo');
  });

  // A colour that is not a plain hex must never become CSS on the page.
  it('ignores a colour that is not a plain hex', () => {
    presets.set([{ name: 'mysql', category: 'databases', icon: 'database', color: 'url(#x)' }]);
    const { container } = render(ServiceIcon, { props: { name: 'mysql' } });
    expect(box(container).className).toContain('indigo');
    expect(box(container).getAttribute('style') || '').not.toContain('url(');
  });
});
