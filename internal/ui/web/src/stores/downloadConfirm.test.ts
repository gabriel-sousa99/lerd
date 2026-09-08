import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { get } from 'svelte/store';

function estimate(body: unknown, status = 200) {
  return vi.fn(async () => new Response(JSON.stringify(body), { status })) as unknown as typeof fetch;
}

describe('download confirmation', () => {
  const realFetch = globalThis.fetch;

  beforeEach(() => {
    vi.resetModules();
  });

  afterEach(() => {
    globalThis.fetch = realFetch;
  });

  // The modal is only worth interrupting for when bytes actually move, so an
  // image that is already on the machine goes straight through.
  it('does not prompt when the image is already local', async () => {
    globalThis.fetch = estimate({ image: 'redis:7-alpine', bytes: 0, local: true });
    const { confirmDownload, downloadConfirm } = await import('./downloadConfirm');
    expect(await confirmDownload('redis', { service: 'redis', action: 'update' })).toBe(true);
    expect(get(downloadConfirm).open).toBe(false);
  });

  it('does not prompt when the operation downloads nothing', async () => {
    globalThis.fetch = estimate({ image: '', bytes: 0, local: false });
    const { confirmDownload, downloadConfirm } = await import('./downloadConfirm');
    expect(await confirmDownload('redis', { service: 'redis', action: 'update' })).toBe(true);
    expect(get(downloadConfirm).open).toBe(false);
  });

  // An estimate we cannot fetch is not a reason to block work the user asked for.
  it('proceeds when the estimate cannot be fetched', async () => {
    globalThis.fetch = estimate({ error: 'nope' }, 500);
    const { confirmDownload, downloadConfirm } = await import('./downloadConfirm');
    expect(await confirmDownload('redis', { service: 'redis', action: 'update' })).toBe(true);
    expect(get(downloadConfirm).open).toBe(false);
  });

  it('opens with the size and resolves what the user answered', async () => {
    globalThis.fetch = estimate({ image: 'mysql:8.0', bytes: 233588484, local: false });
    const { confirmDownload, downloadConfirm, answerDownloadConfirm } = await import('./downloadConfirm');

    const pending = confirmDownload('mysql', { service: 'mysql', action: 'update' });
    await vi.waitFor(() => expect(get(downloadConfirm).open).toBe(true));
    expect(get(downloadConfirm).download?.bytes).toBe(233588484);
    expect(get(downloadConfirm).name).toBe('mysql');

    answerDownloadConfirm(false);
    expect(await pending).toBe(false);
    expect(get(downloadConfirm).open).toBe(false);
  });

  it('drops empty query parameters so the backend sees only what was asked', async () => {
    const seen: string[] = [];
    globalThis.fetch = vi.fn(async (url: unknown) => {
      seen.push(String(url));
      return new Response(JSON.stringify({ image: '', bytes: 0, local: false }), { status: 200 });
    }) as unknown as typeof fetch;
    const { confirmDownload } = await import('./downloadConfirm');
    await confirmDownload('mysql', { service: 'mysql', action: 'update', tag: '' });
    expect(seen[0]).toContain('service=mysql');
    expect(seen[0]).toContain('action=update');
    expect(seen[0]).not.toContain('tag=');
  });
});
