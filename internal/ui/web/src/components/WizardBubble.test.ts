import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { get } from 'svelte/store';
import WizardBubble from './WizardBubble.svelte';
import { modal } from '$stores/modals';
import { activeRun } from '$stores/wizard';
import { clearWizardState, saveWizardState, wizardState } from '$lib/wizardState';

describe('WizardBubble', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    clearWizardState();
    activeRun.set(null);
    modal.set({ kind: null });
  });

  it('stays out of the way when no flow was parked', () => {
    render(WizardBubble);
    expect(screen.queryByText('Waiting to continue')).toBeNull();
  });

  // Work sent to the background needs somewhere to come back to, and it says
  // which project it belongs to rather than just that something is happening.
  it('shows the parked project while its run goes on', async () => {
    saveWizardState({ step: 'setup', dir: '/home/u/shop', runId: 'r1', runKind: 'setup' });
    activeRun.set({
      id: 'r1',
      kind: 'setup',
      dir: '/home/u/shop',
      label: 'composer install',
      status: 'running',
      started: 0
    });

    render(WizardBubble);

    expect(await screen.findByText('shop')).toBeTruthy();
    expect(screen.getByText('composer install')).toBeTruthy();
  });

  // A run that ended leaves the bubble behind saying the flow can be picked up,
  // which is the feedback the + button on its own never gave.
  it('says the flow is waiting once the run ends', async () => {
    saveWizardState({ step: 'setup', dir: '/home/u/shop' });

    render(WizardBubble);

    expect(await screen.findByText('Waiting to continue')).toBeTruthy();
  });

  it('reopens the wizard when clicked', async () => {
    saveWizardState({ step: 'setup', dir: '/home/u/shop' });
    render(WizardBubble);

    await fireEvent.click(await screen.findByText('shop'));

    expect(get(modal).kind).toBe('link');
  });

  // The wizard is already on screen with the same state; a bubble on top of it
  // would only repeat itself.
  it('hides while the wizard is open', async () => {
    saveWizardState({ step: 'setup', dir: '/home/u/shop' });
    modal.set({ kind: 'link' });

    render(WizardBubble);

    await waitFor(() => expect(screen.queryByText('shop')).toBeNull());
  });

  it('can be dismissed, which drops the parked note', async () => {
    saveWizardState({ step: 'setup', dir: '/home/u/shop' });
    render(WizardBubble);

    await fireEvent.click(screen.getByLabelText('Close'));

    await waitFor(() => expect(get(wizardState)).toBeNull());
  });
});
