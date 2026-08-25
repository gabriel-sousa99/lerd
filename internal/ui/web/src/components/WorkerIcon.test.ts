import { render } from '@testing-library/svelte';
import { describe, it, expect, beforeEach } from 'vitest';
import WorkerIcon from './WorkerIcon.svelte';
import { workerMarks } from '$stores/workerMarks';
import { frameworkMarks } from '$stores/frameworkMarks';
import { serviceIcons } from '$stores/serviceIcons';

const VITE_MARK = '<svg viewBox="0 0 24 24"><path d="M1 1h2v2H1z"/></svg>';
const LARAVEL_MARK = '<svg viewBox="0 0 24 24"><path d="M5 5h2v2H5z"/></svg>';
const STRIPE_MARK = '<svg viewBox="0 0 24 24"><path d="M9 9h2v2H9z"/></svg>';

beforeEach(() => {
  workerMarks.set({
    workers: {
      'laravel/queue': { icon: 'queue', color: '#ff2d20' },
      'laravel/vite': { icon: 'vite', color: '#9135ff' },
      'laravel/horizon': { color: '#ff2d20' },
      'symfony/messenger': { icon: 'queue', color: '#000000' }
    },
    marks: { vite: VITE_MARK }
  });
  frameworkMarks.set({ laravel: { svg: LARAVEL_MARK, color: '#ff2d20' } });
  serviceIcons.set({ 'stripe-mock': STRIPE_MARK });
});

describe('WorkerIcon', () => {
  it('draws the mark the worker names', () => {
    const { container } = render(WorkerIcon, {
      props: { worker: 'vite', framework: 'laravel' }
    });
    expect(container.innerHTML).toContain('M1 1h2v2H1z');
  });

  it('falls back to the glyph the worker names', () => {
    const { container } = render(WorkerIcon, {
      props: { worker: 'queue', framework: 'laravel' }
    });
    expect(container.innerHTML).not.toContain('M1 1h2v2H1z');
    expect(container.querySelector('svg')).toBeTruthy();
  });

  it("borrows the framework's mark when the worker names no icon", () => {
    const { container } = render(WorkerIcon, {
      props: { worker: 'horizon', framework: 'laravel' }
    });
    expect(container.innerHTML).toContain('M5 5h2v2H5z');
  });

  it('inks a worker in the tone its definition declares', () => {
    const { container } = render(WorkerIcon, {
      props: { worker: 'vite', framework: 'laravel' }
    });
    expect(container.querySelector('span')?.getAttribute('style')).toContain('--mark-tint');
  });

  it('draws a worker whose framework is unknown like its namesake elsewhere', () => {
    const { container } = render(WorkerIcon, { props: { worker: 'vite' } });
    expect(container.innerHTML).toContain('M1 1h2v2H1z');
  });

  it('lets a lerd-owned worker borrow a service preset mark', () => {
    const { container } = render(WorkerIcon, {
      props: { worker: 'stripe', preset: 'stripe-mock' }
    });
    expect(container.innerHTML).toContain('M9 9h2v2H9z');
  });

  it('settles for a plain glyph when nothing declares anything', () => {
    workerMarks.set({ workers: {}, marks: {} });
    frameworkMarks.set({});
    const { container } = render(WorkerIcon, { props: { worker: 'unknown' } });
    expect(container.querySelector('svg')).toBeTruthy();
    expect(container.innerHTML).not.toContain('M5 5h2v2H5z');
  });
});
