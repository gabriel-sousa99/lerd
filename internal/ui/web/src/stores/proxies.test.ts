import { beforeEach, describe, expect, it, vi } from 'vitest';

const apiJson = vi.fn();
const apiFetch = vi.fn();
vi.mock('$lib/api', () => ({
  apiJson: (...args: unknown[]) => apiJson(...args),
  apiFetch: (...args: unknown[]) => apiFetch(...args)
}));
vi.mock('$lib/ws', () => ({ wsMessage: { subscribe: () => () => undefined } }));

import {
  PROXY_MONITOR_INTERVAL_MS,
  loadProxyConfig,
  loadProxyRuntime,
  loadProxyStats,
  startProxyMonitoring
} from './proxies';

describe('proxy operations API', () => {
  beforeEach(() => {
    apiJson.mockReset().mockResolvedValue({});
    vi.useRealTimers();
  });

  it('loads status, traffic and generated config through scoped endpoints', async () => {
    await loadProxyRuntime('spa app');
    await loadProxyStats('spa app');
    await loadProxyConfig('spa app');

    expect(apiJson).toHaveBeenNthCalledWith(1, '/api/proxies/spa%20app/status');
    expect(apiJson).toHaveBeenNthCalledWith(2, '/api/proxies/spa%20app/stats');
    expect(apiJson).toHaveBeenNthCalledWith(3, '/api/proxies/spa%20app/config');
  });

  it('refreshes operational data every ten seconds while mounted', async () => {
    vi.useFakeTimers();
    const refresh = vi.fn().mockResolvedValue(undefined);
    const stop = startProxyMonitoring(refresh);

    expect(PROXY_MONITOR_INTERVAL_MS).toBe(10_000);
    expect(refresh).toHaveBeenCalledOnce();
    await vi.advanceTimersByTimeAsync(PROXY_MONITOR_INTERVAL_MS * 2);
    expect(refresh).toHaveBeenCalledTimes(3);

    stop();
    await vi.advanceTimersByTimeAsync(PROXY_MONITOR_INTERVAL_MS);
    expect(refresh).toHaveBeenCalledTimes(3);
  });
});
