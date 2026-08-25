import { render } from '@testing-library/svelte';
import { describe, it, expect } from 'vitest';
import SiteLogs from './SiteLogs.svelte';
import type { Site } from '$stores/sites';

function site(over: Partial<Site> = {}): Site {
  return { domain: 'app.test', name: 'app', uses_php: true, ...over } as Site;
}

describe('SiteLogs tabs', () => {
  it('keeps a stopped worker tab so its logs stay readable', () => {
    const { getByText } = render(SiteLogs, {
      props: {
        site: site({
          has_queue_worker: true,
          queue_running: false,
          framework_workers: [{ name: 'vite', label: 'Vite', running: false }]
        })
      }
    });
    expect(getByText('Queue')).toBeTruthy();
    expect(getByText('Vite')).toBeTruthy();
  });

  it('marks a failing worker so it stands out from a stopped one', () => {
    const { getByText } = render(SiteLogs, {
      props: {
        site: site({
          has_queue_worker: true,
          queue_failing: true,
          framework_workers: [{ name: 'vite', label: 'Vite', running: false }]
        })
      }
    });
    expect(getByText('Queue !')).toBeTruthy();
  });

  it('offers no worker tab for a site that has none', () => {
    const { queryByText } = render(SiteLogs, { props: { site: site() } });
    expect(queryByText('Queue')).toBeNull();
    expect(queryByText('Vite')).toBeNull();
  });
});
