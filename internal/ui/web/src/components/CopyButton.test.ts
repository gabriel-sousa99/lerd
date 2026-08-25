import { render, screen } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import CopyButton from './CopyButton.svelte';

describe('CopyButton', () => {
  beforeEach(() => {
    vi.useFakeTimers();
  });
  afterEach(() => {
    vi.useRealTimers();
  });

  it('copies a plain string', async () => {
    const writeText = vi.fn(async () => {});
    Object.assign(navigator, { clipboard: { writeText } });
    render(CopyButton, { props: { text: 'app/User.php:12', label: 'Copy path' } });
    screen.getByLabelText('Copy path').click();
    await vi.waitFor(() => expect(writeText).toHaveBeenCalledWith('app/User.php:12'));
  });

  it('defers a thunk until the button is pressed', async () => {
    const writeText = vi.fn(async () => {});
    Object.assign(navigator, { clipboard: { writeText } });
    const build = vi.fn(() => 'select 1');
    render(CopyButton, { props: { text: build, label: 'Copy SQL' } });
    expect(build).not.toHaveBeenCalled();
    screen.getByLabelText('Copy SQL').click();
    await vi.waitFor(() => expect(writeText).toHaveBeenCalledWith('select 1'));
  });

  it('confirms with a checkmark that clears itself', async () => {
    Object.assign(navigator, { clipboard: { writeText: vi.fn(async () => {}) } });
    const { container } = render(CopyButton, { props: { text: 'x', label: 'Copy path' } });
    const marked = () => container.querySelector('.text-emerald-600') !== null;
    expect(marked()).toBe(false);
    screen.getByLabelText('Copy path').click();
    await vi.waitFor(() => expect(marked()).toBe(true));
    await vi.advanceTimersByTimeAsync(1600);
    expect(marked()).toBe(false);
  });

  it('stays quiet when there is no clipboard', async () => {
    Object.assign(navigator, {
      clipboard: {
        writeText: async () => {
          throw new Error('not a secure context');
        }
      }
    });
    const { container } = render(CopyButton, { props: { text: 'x', label: 'Copy path' } });
    screen.getByLabelText('Copy path').click();
    await vi.advanceTimersByTimeAsync(10);
    expect(container.querySelector('.text-emerald-600')).toBeNull();
  });
});
