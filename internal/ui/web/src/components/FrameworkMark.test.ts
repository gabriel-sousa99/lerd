import { render } from '@testing-library/svelte';
import { describe, it, expect, beforeEach } from 'vitest';
import FrameworkMark from './FrameworkMark.svelte';
import { frameworkMarks } from '$stores/frameworkMarks';

const MARK = '<svg viewBox="0 0 24 24"><path d="M3 3h18v18H3z"/></svg>';

beforeEach(() => {
  frameworkMarks.set({});
});

describe('FrameworkMark', () => {
  it('draws the mark the store shipped, tinted by the declared colour', () => {
    frameworkMarks.set({ laravel: { svg: MARK, color: '#e02419' } });
    const { container } = render(FrameworkMark, { props: { name: 'laravel' } });
    expect(container.querySelector('path')?.getAttribute('d')).toBe('M3 3h18v18H3z');
    const el = container.firstElementChild as HTMLElement;
    expect(el.className).toContain('mark-ink');
    expect(el.getAttribute('style')).toContain('--mark-tint: #e02419');
  });

  // The label beside it already names the framework, so a stand-in glyph would
  // only add noise.
  it('draws nothing when the framework has no mark', () => {
    frameworkMarks.set({ laravel: { color: '#e02419' } });
    const { container } = render(FrameworkMark, { props: { name: 'laravel' } });
    expect(container.querySelector('svg')).toBeNull();
  });

  it('draws nothing for a framework the store knows nothing about', () => {
    const { container } = render(FrameworkMark, { props: { name: 'nope' } });
    expect(container.querySelector('svg')).toBeNull();
  });

  it('draws nothing when the site has no framework at all', () => {
    frameworkMarks.set({ laravel: { svg: MARK, color: '#e02419' } });
    const { container } = render(FrameworkMark, { props: { name: undefined } });
    expect(container.querySelector('svg')).toBeNull();
  });

  // Inside a coloured pill the mark takes the pill's colour, so the badge reads
  // as one object rather than a mark sitting next to a badge.
  it('takes the surrounding colour when the caller turns the tint off', () => {
    frameworkMarks.set({ laravel: { svg: MARK, color: '#e02419' } });
    const { container } = render(FrameworkMark, { props: { name: 'laravel', tint: false } });
    const el = container.firstElementChild as HTMLElement;
    expect(el.className).toContain('mark-inherit');
    expect(el.className).not.toContain('mark-ink');
    expect(el.getAttribute('style') || '').not.toContain('--mark-tint');
    expect(container.querySelector('path')).not.toBeNull();
  });

  // A colour out of the store must never reach the page as arbitrary CSS.
  it('ignores a colour that is not a plain hex', () => {
    frameworkMarks.set({ laravel: { svg: MARK, color: 'url(#x)' } });
    const { container } = render(FrameworkMark, { props: { name: 'laravel' } });
    const style = (container.firstElementChild as HTMLElement).getAttribute('style') || '';
    expect(style).not.toContain('url(');
  });
});
