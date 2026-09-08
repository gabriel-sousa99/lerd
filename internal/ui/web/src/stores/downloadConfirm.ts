import { get, writable } from 'svelte/store';
import { apiJson } from '$lib/api';

// PendingDownload mirrors /api/image-estimate: what an operation would fetch
// before it can run. An empty image means it downloads nothing at all.
export interface PendingDownload {
  image: string;
  bytes: number;
  local: boolean;
}

interface ConfirmState {
  open: boolean;
  /** Display label for whatever is being installed or rebuilt. */
  name: string;
  download: PendingDownload | null;
  resolve: ((ok: boolean) => void) | null;
}

const closed: ConfirmState = { open: false, name: '', download: null, resolve: null };

export const downloadConfirm = writable<ConfirmState>(closed);

// confirmDownload asks before an operation spends bandwidth, and resolves true
// without prompting whenever nothing would actually be downloaded, so the modal
// only ever interrupts a real transfer. An estimate the backend cannot produce
// must not block the operation either.
export async function confirmDownload(
  name: string,
  params: Record<string, string>
): Promise<boolean> {
  const q = new URLSearchParams();
  for (const [k, v] of Object.entries(params)) if (v) q.set(k, v);
  let d: PendingDownload;
  try {
    d = await apiJson<PendingDownload>('/api/image-estimate?' + q.toString());
  } catch {
    return true;
  }
  if (!d || !d.image || d.local) return true;
  return new Promise<boolean>((resolve) =>
    downloadConfirm.set({ open: true, name, download: d, resolve })
  );
}

export function answerDownloadConfirm(ok: boolean) {
  const state = get(downloadConfirm);
  downloadConfirm.set(closed);
  state.resolve?.(ok);
}
