import { describe, it, expect } from 'vitest';
import { readNDJSON } from './ndjson';

function stream(...chunks: string[]): ReadableStream<Uint8Array> {
  const enc = new TextEncoder();
  return new ReadableStream({
    start(controller) {
      for (const c of chunks) controller.enqueue(enc.encode(c));
      controller.close();
    }
  });
}

describe('readNDJSON', () => {
  it('yields one event per line', async () => {
    const got: unknown[] = [];
    await readNDJSON(stream('{"a":1}\n{"a":2}\n'), (e) => got.push(e));
    expect(got).toEqual([{ a: 1 }, { a: 2 }]);
  });

  // A chunk boundary lands anywhere, so a line split across two reads has to be
  // buffered rather than parsed twice as garbage.
  it('joins a line split across chunks', async () => {
    const got: unknown[] = [];
    await readNDJSON(stream('{"a":', '1}\n'), (e) => got.push(e));
    expect(got).toEqual([{ a: 1 }]);
  });

  it('skips a malformed line and keeps reading', async () => {
    const got: unknown[] = [];
    await readNDJSON(stream('{"a":1}\nnot json\n{"a":2}\n'), (e) => got.push(e));
    expect(got).toEqual([{ a: 1 }, { a: 2 }]);
  });

  // The server flushes per event, so a stream cut before the final newline is
  // normal; the completed lines before it must still have been delivered.
  it('drops a trailing line with no newline', async () => {
    const got: unknown[] = [];
    await readNDJSON(stream('{"a":1}\n{"a":2}'), (e) => got.push(e));
    expect(got).toEqual([{ a: 1 }]);
  });

  it('ignores blank lines', async () => {
    const got: unknown[] = [];
    await readNDJSON(stream('\n{"a":1}\n\n'), (e) => got.push(e));
    expect(got).toEqual([{ a: 1 }]);
  });
});
