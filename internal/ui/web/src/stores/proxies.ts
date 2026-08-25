import { writable } from 'svelte/store';
import { apiJson, apiFetch } from '$lib/api';
import { wsMessage } from '$lib/ws';

export interface Route {
  path: string;
  site?: string;
  upstream_port?: number;
  upstream_host?: string;
}

export interface Proxy {
  name: string;
  domain: string;
  domains: string[];
  upstream_port: number;
  upstream_host: string;
  upstream_scheme: 'http' | 'https';
  health_path?: string;
  timeout_seconds: number;
  path?: string;
  secured: boolean;
  paused: boolean;
  managed: boolean;
  node_version?: string;
  cmd?: string;
  autostart: boolean;
  site?: string;
  routes?: Route[];
  fullstack?: boolean;
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
  aliases?: string[];
  port: number;
  upstream_host?: string;
  upstream_scheme?: 'http' | 'https';
  health_path?: string;
  timeout_seconds?: number;
  path?: string;
  no_secure?: boolean;
  managed?: boolean;
  cmd?: string;
  node_version?: string;
  autostart?: boolean;
  site?: string;
  routes?: Route[];
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
  aliases?: string[];
  port?: number;
  path?: string;
  cmd?: string;
  node_version?: string;
  upstream_host?: string;
  upstream_scheme?: 'http' | 'https';
  health_path?: string;
  timeout_seconds?: number;
  autostart?: boolean;
  site?: string;
  routes?: Route[];
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

export interface ProxyRuntimeStatus {
  state: 'healthy' | 'degraded' | 'unreachable' | 'failed' | 'inactive' | 'paused' | 'misconfigured';
  upstream_reachable: boolean;
  latency_ms: number;
  http_status?: number;
  checked_at: string;
  nginx_running: boolean;
  vhost_present: boolean;
  certificate_present: boolean;
  unit_state?: string;
  error?: string;
}

export interface ProxyRouteStat {
  route: string;
  method: string;
  example: string;
  p50_millis: number;
  p95_millis: number;
  recent_p95_millis?: number;
  multiplier: number;
  samples: number;
}

export interface ProxyTrafficStats {
  site: string;
  median_millis: number;
  samples: number;
  slow: ProxyRouteStat[];
  updated_at?: string;
}

export interface ProxyGeneratedConfig {
  path: string;
  content: string;
}

export const PROXY_MONITOR_INTERVAL_MS = 10_000;

function proxyEndpoint(name: string, resource: string): string {
  return `/api/proxies/${encodeURIComponent(name)}/${resource}`;
}

export function loadProxyRuntime(name: string): Promise<ProxyRuntimeStatus> {
  return apiJson<ProxyRuntimeStatus>(proxyEndpoint(name, 'status'));
}

export function loadProxyStats(name: string): Promise<ProxyTrafficStats> {
  return apiJson<ProxyTrafficStats>(proxyEndpoint(name, 'stats'));
}

export function loadProxyConfig(name: string): Promise<ProxyGeneratedConfig> {
  return apiJson<ProxyGeneratedConfig>(proxyEndpoint(name, 'config'));
}

export function startProxyMonitoring(refresh: () => void | Promise<void>): () => void {
  void refresh();
  const timer = window.setInterval(() => void refresh(), PROXY_MONITOR_INTERVAL_MS);
  return () => window.clearInterval(timer);
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
