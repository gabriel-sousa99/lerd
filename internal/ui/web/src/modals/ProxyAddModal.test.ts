import { render } from '@testing-library/svelte';
import { describe, it, expect, vi } from 'vitest';
import ProxyAddModal from './ProxyAddModal.svelte';

vi.mock('$stores/sites', async (importOriginal) => {
  const actual = await importOriginal<typeof import('$stores/sites')>();
  return { ...actual, loadSites: vi.fn() };
});

// This form is the longest in the dashboard, and Modal.svelte deliberately
// leaves the scroll to each body, so without these two classes the fields past
// the fold were simply unreachable.
describe('ProxyAddModal layout', () => {
  it('scrolls its own body', () => {
    render(ProxyAddModal);
    const form = document.getElementById('proxy-add-form');
    expect(form).not.toBeNull();
    expect(form!.className).toContain('overflow-y-auto');
    expect(form!.className).toContain('max-h-[60vh]');
  });

  // The panel is a fixed max-w, so a viewport breakpoint answered a question
  // nobody asked and put two columns in 464px. The columns follow the form.
  it('sizes the upstream columns off the form, not the window', () => {
    render(ProxyAddModal);
    const form = document.getElementById('proxy-add-form');
    expect(form!.className).toContain('@container');

    const fieldset = form!.querySelector('fieldset');
    expect(fieldset).not.toBeNull();
    expect(fieldset!.className).toContain('@lg:grid-cols-2');
    expect(fieldset!.className).not.toContain('sm:grid-cols-2');
  });
});
