import { describe, it, expect } from 'vitest';
import { brandTint, brandTintStyle, luminance } from './brandTint';

function rgb(hex: string): [number, number, number] {
  return [
    parseInt(hex.slice(1, 3), 16),
    parseInt(hex.slice(3, 5), 16),
    parseInt(hex.slice(5, 7), 16)
  ];
}

describe('brandTint', () => {
  it('keeps a mid-tone brand colour as declared in both themes', () => {
    const t = brandTint('#e02419');
    expect(t).not.toBeNull();
    expect(t!.light).toBe('#e02419');
    expect(t!.dark).toBe('#e02419');
  });

  it('expands a three digit hex', () => {
    expect(brandTint('#abc')).toEqual(brandTint('#aabbcc'));
  });

  // The case the issue calls out: a near-black brand mark on the dark card.
  it('lifts a near-black brand colour so it separates from the dark card', () => {
    const t = brandTint('#000000')!;
    expect(luminance(rgb(t.dark))).toBeGreaterThan(0.15);
    expect(t.light).toBe('#000000');
  });

  it('deepens a near-white brand colour so it separates from the light card', () => {
    const t = brandTint('#ffffff')!;
    expect(luminance(rgb(t.light))).toBeLessThan(0.55);
    expect(t.dark).toBe('#ffffff');
  });

  it('rejects anything that is not a plain hex', () => {
    for (const bad of ['', 'red', 'rgb(1,2,3)', '#12', 'url(#x)', 'var(--x)', '#00758g', undefined, null]) {
      expect(brandTint(bad)).toBeNull();
    }
  });
});

describe('brandTintStyle', () => {
  it('declares both tones as custom properties', () => {
    expect(brandTintStyle('#e02419')).toBe('--mark-tint:#e02419;--mark-tint-dark:#e02419');
  });

  it('is empty when there is no usable colour, so the category tint stands', () => {
    expect(brandTintStyle('javascript:alert(1)')).toBe('');
    expect(brandTintStyle(undefined)).toBe('');
  });
});
