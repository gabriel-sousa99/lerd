import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';

// The real editor pulls in Monaco and hits the sites API on mount; swap it
// for a stub so these tests stay focused on the modal shell and close guard.
vi.mock('$tabs/sites/SiteNginxTab.svelte', () => import('./SiteNginxModal.stub.svelte'));

import Harness from './SiteNginxModal.test.svelte';
import { modal } from '$stores/modals';
import type { Site } from '$stores/sites';

const site = { domain: 'acme.test' } as Site;

describe('SiteNginxModal', () => {
  beforeEach(() => modal.set({ kind: null }));

  it('renders nothing when closed', () => {
    render(Harness, { props: { open: false, site, domain: 'acme.test', onclose: () => {} } });
    expect(screen.queryByText(/acme\.test/)).not.toBeInTheDocument();
  });

  it('shows the domain in the title and mounts the editor when open', () => {
    render(Harness, { props: { open: true, site, domain: 'acme.test', onclose: () => {} } });
    expect(screen.getByText(/Nginx config: acme\.test/)).toBeInTheDocument();
    expect(screen.getByTestId('nginx-editor-stub')).toBeInTheDocument();
  });

  it('edits the active worktree domain, not the site primary', () => {
    render(Harness, { props: { open: true, site, domain: 'feat.acme.test', onclose: () => {} } });
    expect(screen.getByText(/Nginx config: feat\.acme\.test/)).toBeInTheDocument();
    expect(screen.getByTestId('nginx-editor-stub')).toHaveTextContent('feat.acme.test');
  });

  it('switches the editor to the location-scope file from the Location tab', async () => {
    render(Harness, { props: { open: true, site, domain: 'acme.test', onclose: () => {} } });
    expect(screen.getByTestId('nginx-editor-stub')).toHaveAttribute('data-scope', 'server');
    await fireEvent.click(screen.getByText('Location'));
    expect(screen.getByTestId('nginx-editor-stub')).toHaveAttribute('data-scope', 'location');
    await fireEvent.click(screen.getByText('Server block'));
    expect(screen.getByTestId('nginx-editor-stub')).toHaveAttribute('data-scope', 'server');
  });

  it('closes the editor after a successful save', async () => {
    const onclose = vi.fn();
    render(Harness, { props: { open: true, site, domain: 'acme.test', onclose } });
    await fireEvent.click(screen.getByTestId('stub-saved'));
    expect(onclose).toHaveBeenCalledOnce();
  });

  it('closes on Escape when no confirm modal is stacked', () => {
    const onclose = vi.fn();
    render(Harness, { props: { open: true, site, domain: 'acme.test', onclose } });
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }));
    expect(onclose).toHaveBeenCalledOnce();
  });

  it('ignores Escape while a stacked confirm modal owns the foreground', () => {
    const onclose = vi.fn();
    render(Harness, { props: { open: true, site, domain: 'acme.test', onclose } });
    modal.set({
      kind: 'nginxSave',
      nginxSave: { domain: 'acme.test', scope: 'server', content: '', original: '', exists: true }
    });
    window.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }));
    expect(onclose).not.toHaveBeenCalled();
  });
});
