import { render, screen } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import WorkerHealthBanner from './WorkerHealthBanner.svelte';
import { unhealthyWorkers } from '$stores/workerHealth';

const calls: string[] = [];

beforeEach(() => {
  calls.length = 0;
  unhealthyWorkers.set([{ site: 'app', worker: 'queue', unit: 'lerd-queue-app', state: 'failed' }]);
  vi.stubGlobal(
    'fetch',
    vi.fn(async (url: string) => {
      calls.push(String(url));
      return new Response(JSON.stringify({ ok: true, stopped: 1, unhealthy: [] }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' }
      });
    })
  );
});

afterEach(() => {
  unhealthyWorkers.set([]);
  vi.unstubAllGlobals();
});

describe('WorkerHealthBanner', () => {
  it('offers stopping the workers as well as healing them', () => {
    render(WorkerHealthBanner);
    expect(screen.getByRole('button', { name: 'Heal' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Stop them' })).toBeInTheDocument();
  });

  it('asks the server to stop the reported workers', async () => {
    render(WorkerHealthBanner);
    screen.getByRole('button', { name: 'Stop them' }).click();
    await vi.waitFor(() => expect(calls.some((u) => u.includes('/api/workers/stop'))).toBe(true));
  });

  it('re-reads the health snapshot once the stop is done', async () => {
    render(WorkerHealthBanner);
    screen.getByRole('button', { name: 'Stop them' }).click();
    await vi.waitFor(() => expect(calls.some((u) => u.includes('/api/workers/health'))).toBe(true));
  });

  it('draws nothing while every worker is healthy', () => {
    unhealthyWorkers.set([]);
    const { container } = render(WorkerHealthBanner);
    expect(container.querySelectorAll('button')).toHaveLength(0);
  });
});
