import { writable } from 'svelte/store';
import { apiJson, apiFetch } from '$lib/api';
import { wsMessage } from '$lib/ws';

export interface Proxy {
  name: string;
  domain: string;
  domains: string[];
  upstream_port: number;
  upstream_host: string;
  path?: string;
  secured: boolean;
  paused: boolean;
  managed: boolean;
  node_version?: string;
  cmd?: string;
  autostart: boolean;
}

export const proxies = writable<Proxy[]>([]);
export const proxiesLoaded = writable<boolean>(false);

export async function loadProxies(): Promise<void> {
  try {
    const list = await apiJson<Proxy[]>('/api/proxies');
    proxies.set(Array.isArray(list) ? list : []);
    proxiesLoaded.set(true);
  } catch (e) {
    console.error('loadProxies failed', e);
  }
}

export interface CreateProxyInput {
  domain: string;
  port: number;
  path?: string;
  no_secure?: boolean;
  managed?: boolean;
  cmd?: string;
  node_version?: string;
  autostart?: boolean;
}

export async function createProxy(input: CreateProxyInput): Promise<Proxy> {
  const created = await apiJson<Proxy>('/api/proxies', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(input)
  });
  await loadProxies();
  return created;
}

export interface UpdateProxyInput {
  port?: number;
  path?: string;
  cmd?: string;
  node_version?: string;
  upstream_host?: string;
  autostart?: boolean;
}

export async function updateProxy(name: string, input: UpdateProxyInput): Promise<Proxy> {
  const updated = await apiJson<Proxy>(`/api/proxies/${encodeURIComponent(name)}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(input)
  });
  await loadProxies();
  return updated;
}

export async function deleteProxy(name: string): Promise<void> {
  const res = await apiFetch(`/api/proxies/${encodeURIComponent(name)}`, { method: 'DELETE' });
  if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
  await loadProxies();
}

export type ProxyAction = 'secure' | 'unsecure' | 'pause' | 'resume' | 'start' | 'stop';

export async function proxyAction(name: string, action: ProxyAction): Promise<void> {
  await apiJson<Proxy>(`/api/proxies/${encodeURIComponent(name)}/${action}`, { method: 'POST' });
  await loadProxies();
}

// Backend ws_broker emits a frame whenever KindProxies is published. When
// KindProxies is the only kind in the frame, type==="proxies"; when it
// coalesces with other kinds (eventbus debounce 150ms), type==="snapshot"
// and the proxies signal lives in the kinds[] array. Both paths trigger
// a re-fetch via /api/proxies. See internal/ui/ws_broker.go runSnapshotInvalidator.
wsMessage.subscribe((msg) => {
  if (!msg) return;
  if (msg.type === 'proxies' || msg.kinds?.includes?.('proxies')) {
    void loadProxies();
  }
});
