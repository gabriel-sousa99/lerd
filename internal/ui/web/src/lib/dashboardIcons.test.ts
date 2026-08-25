import { describe, it, expect } from 'vitest';
import { dashboardIconSvg } from './dashboardIcons';

describe('dashboardIconSvg', () => {
  it('draws the glyph a preset names', () => {
    expect(dashboardIconSvg('gotenberg', 'document')).toContain('<path');
    expect(dashboardIconSvg('gotenberg', 'document')).not.toBe(dashboardIconSvg('gotenberg', 'docs'));
  });

  // A store preset can name a glyph added after the running binary shipped, so
  // an unknown name has to degrade rather than draw nothing.
  it('falls back to a generic glyph for a name this build does not have', () => {
    expect(dashboardIconSvg('whatever', 'not-a-glyph-yet')).toBe(dashboardIconSvg('whatever'));
    expect(dashboardIconSvg('whatever')).toContain('<path');
  });

  it('keeps the dashboards that are not services on their own glyphs', () => {
    expect(dashboardIconSvg('docs')).toBe(dashboardIconSvg('x', 'docs'));
    expect(dashboardIconSvg('profiler')).toBe(dashboardIconSvg('x', 'flame'));
  });
});
