import { writable } from 'svelte/store';

// The site wizard's place in the flow, kept outside the component so closing
// the modal or reloading the page does not lose a scaffold that is still
// running on the host. The run itself lives in lerd-ui; this is only the note
// telling the wizard which one to reattach to.
export interface WizardState {
  step: string;
  dir?: string;
  parent?: string;
  name?: string;
  framework?: string;
  frameworkVersion?: string;
  runId?: string;
  runKind?: string;
  // Setup steps still to run, head first, so a parked setup resumes with the
  // rest of its list rather than an empty one.
  queue?: string[];
}

const KEY = 'lerd.siteWizard';

export function loadWizardState(): WizardState | null {
  try {
    const raw = localStorage.getItem(KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as WizardState;
    return parsed && typeof parsed.step === 'string' ? parsed : null;
  } catch {
    return null;
  }
}

// wizardState is the parked flow as the rest of the dashboard sees it, so the
// bubble offering to resume appears and disappears with the note itself.
export const wizardState = writable<WizardState | null>(loadWizardState());

export function saveWizardState(state: WizardState): void {
  wizardState.set(state);
  try {
    localStorage.setItem(KEY, JSON.stringify(state));
  } catch {
    // A browser refusing storage costs the resume across reloads, not the flow.
  }
}

export function clearWizardState(): void {
  wizardState.set(null);
  try {
    localStorage.removeItem(KEY);
  } catch {
    // As above: nothing to clear if nothing could be stored.
  }
}
