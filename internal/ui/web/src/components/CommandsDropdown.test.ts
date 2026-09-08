import { render, screen } from '@testing-library/svelte';
import { describe, it, expect, vi, afterEach } from 'vitest';

// Stub the loader and the pin writer while keeping the rest of the store real,
// so the component's own wiring (row buttons, pin toggle, launch) is exercised.
const { loadCommands, setCommandPinned, launchCommand } = vi.hoisted(() => ({
  loadCommands: vi.fn(),
  setCommandPinned: vi.fn(),
  launchCommand: vi.fn()
}));
vi.mock('$stores/commands', async (importOriginal) => {
  const actual = await importOriginal<typeof import('$stores/commands')>();
  return { ...actual, loadCommands, setCommandPinned, launchCommand };
});

import CommandsDropdown from './CommandsDropdown.svelte';

const commands = [
  { name: 'native:run', label: 'Run on device', command: 'php artisan native:run', pinned: true },
  { name: 'migrate', label: 'Run migrations', command: 'php artisan migrate' }
];

afterEach(() => {
  loadCommands.mockReset();
  setCommandPinned.mockReset();
  launchCommand.mockReset();
});

async function settle() {
  await new Promise((r) => setTimeout(r, 0));
}

describe('CommandsDropdown pinned commands', () => {
  it('draws a pinned command as its own button on the row', async () => {
    loadCommands.mockResolvedValue(commands);
    render(CommandsDropdown, { props: { domain: 'acme.test' } });
    await settle();

    const btn = screen.getByRole('button', { name: /Run on device/ });
    btn.click();
    expect(launchCommand).toHaveBeenCalledWith('acme.test', expect.objectContaining({ name: 'native:run' }), {
      branch: ''
    });
    expect(screen.queryByRole('button', { name: /Run migrations/ })).toBeNull();
  });

  it('pins a command from the dropdown and reloads the list', async () => {
    loadCommands.mockResolvedValue([commands[1]]);
    setCommandPinned.mockResolvedValue({ ok: true });
    render(CommandsDropdown, { props: { domain: 'acme.test' } });
    await settle();

    screen.getByRole('button', { name: /Commands/ }).click();
    await settle();
    screen.getByRole('button', { name: /Pin to the row/ }).click();
    await settle();

    expect(setCommandPinned).toHaveBeenCalledWith('acme.test', 'migrate', true);
    expect(loadCommands).toHaveBeenCalledTimes(3); // mount, menu open, after the pin
  });

  it('refuses a third pin so the row cannot overflow', async () => {
    loadCommands.mockResolvedValue([
      { name: 'a', label: 'A', command: 'true', pinned: true },
      { name: 'b', label: 'B', command: 'true', pinned: true },
      { name: 'c', label: 'C', command: 'true' }
    ]);
    render(CommandsDropdown, { props: { domain: 'acme.test' } });
    await settle();

    screen.getByRole('button', { name: /Commands/ }).click();
    await settle();
    const pins = screen.getAllByRole('button', { name: /Pin to the row/ });
    expect(pins).toHaveLength(1);
    expect(pins[0]).toBeDisabled();
  });
});
