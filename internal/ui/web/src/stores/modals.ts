import { writable } from 'svelte/store';
import type { Site } from './sites';
import type { Proxy } from './proxies';

export type ModalKind =
  | 'domain'
  | 'link'
  | 'preset'
  | 'remoteControl'
  | 'lanProgress'
  | 'worktreeAdd'
  | 'worktreeRemove'
  | 'proxyAdd'
  | 'proxyEdit'
  | null;

export type LANAction = 'expose' | 'unexpose';

export interface ModalState {
  kind: ModalKind;
  site?: Site;
  lanAction?: LANAction;
  onSuccess?: () => void;
  branch?: string;
  proxy?: Proxy;
}

export const modal = writable<ModalState>({ kind: null });

export function openDomainModal(site: Site) {
  modal.set({ kind: 'domain', site });
}

export function openLinkModal() {
  modal.set({ kind: 'link' });
}

export function openPresetModal() {
  modal.set({ kind: 'preset' });
}

export function openRemoteControlModal(onSuccess?: () => void) {
  modal.set({ kind: 'remoteControl', onSuccess });
}

export function openLANProgressModal(lanAction: LANAction) {
  modal.set({ kind: 'lanProgress', lanAction });
}

export function openWorktreeAddModal(site: Site) {
  modal.set({ kind: 'worktreeAdd', site });
}

export function openWorktreeRemoveModal(site: Site, branch: string) {
  modal.set({ kind: 'worktreeRemove', site, branch });
}

export function openProxyAddModal() {
  modal.set({ kind: 'proxyAdd' });
}

export function openProxyEditModal(proxy: Proxy) {
  modal.set({ kind: 'proxyEdit', proxy });
}

export function closeModal() {
  modal.set({ kind: null });
}
