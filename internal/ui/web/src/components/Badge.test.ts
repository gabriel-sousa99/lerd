import { render, screen } from '@testing-library/svelte';
import { describe, it, expect, vi } from 'vitest';
import Harness from './Badge.test.svelte';

describe('Badge', () => {
  it('renders span by default', () => {
    render(Harness, { props: { tone: 'running', label: 'running' } });
    const el = screen.getByText('running');
    expect(el.tagName.toLowerCase()).toBe('span');
  });

  it('renders a button when onclick is set', () => {
    const onclick = vi.fn();
    render(Harness, { props: { tone: 'framework', label: 'Laravel', onclick } });
    const el = screen.getByText('Laravel');
    expect(el.tagName.toLowerCase()).toBe('button');
    el.click();
    expect(onclick).toHaveBeenCalledOnce();
  });

  it('shows dot only when dot=true', () => {
    const { container } = render(Harness, { props: { tone: 'running', label: 'up', dot: true } });
    expect(container.querySelector('span.rounded-full')).toBeInTheDocument();
  });

  it('applies tone classes', () => {
    render(Harness, { props: { tone: 'paused', label: 'paused' } });
    expect(screen.getByText('paused').className).toMatch(/text-amber-600|bg-amber-50/);
  });

  // A badge naming something that has a brand colour wears it, so a Laravel
  // pill is Laravel red rather than lerd's own red on every framework alike.
  it('paints from the brand colour instead of the tone when given one', () => {
    render(Harness, { props: { tone: 'framework', label: 'Laravel', brand: '#ff2d20' } });
    const el = screen.getByText('Laravel');
    expect(el.className).toContain('mark-tint');
    expect(el.className).not.toContain('text-lerd-red');
    expect(el.getAttribute('style')).toContain('--mark-tint');
  });

  // A framework declaring no colour, or one the sanitizer would refuse, keeps
  // the tone it always had rather than losing its pill entirely.
  it('keeps the tone when the brand colour is absent or not a plain hex', () => {
    render(Harness, { props: { tone: 'framework', label: 'Slim' } });
    expect(screen.getByText('Slim').className).toContain('text-lerd-red');

    render(Harness, { props: { tone: 'framework', label: 'Bad', brand: 'url(#x)' } });
    const bad = screen.getByText('Bad');
    expect(bad.className).toContain('text-lerd-red');
    expect(bad.getAttribute('style') || '').not.toContain('url(');
  });
});
