import { apiFetch } from '$lib/api';

interface ActionResponse {
  ok: boolean;
  error?: string;
}

// Reading the declared/loaded extension set is fetchPhpExtensions' job (see
// $stores/phpVersions): it reports what the image actually carries, not just
// what config declares. These two only mutate; callers refetch that report
// afterwards so the list reflects the rebuilt image rather than the request.
//
// The declared set is global — it applies to every PHP image — so `version`
// selects which image is rebuilt and bounced now, not which set is edited.

export async function addPhpExtension(
  version: string,
  ext: string,
  apkDeps: string[] = []
): Promise<ActionResponse> {
  try {
    const res = await apiFetch('/api/php-versions/' + encodeURIComponent(version) + '/extensions', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ extension: ext, apk_deps: apkDeps })
    });
    return (await res.json()) as ActionResponse;
  } catch (e) {
    return { ok: false, error: e instanceof Error ? e.message : String(e) };
  }
}

export async function removePhpExtension(version: string, ext: string): Promise<ActionResponse> {
  try {
    const res = await apiFetch(
      '/api/php-versions/' +
        encodeURIComponent(version) +
        '/extensions/' +
        encodeURIComponent(ext),
      { method: 'DELETE' }
    );
    return (await res.json()) as ActionResponse;
  } catch (e) {
    return { ok: false, error: e instanceof Error ? e.message : String(e) };
  }
}
