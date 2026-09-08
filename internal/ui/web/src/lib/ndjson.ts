// readNDJSON feeds each JSON line of a streaming response body to onEvent as it
// arrives. A line that does not parse is skipped rather than aborting the read:
// a stream cut mid-line should not throw away the events that came before it.
export async function readNDJSON<T>(
  body: ReadableStream<Uint8Array>,
  onEvent: (evt: T) => void
): Promise<void> {
  const reader = body.getReader();
  const decoder = new TextDecoder();
  let buf = '';
  for (;;) {
    const { done, value } = await reader.read();
    if (done) break;
    buf += decoder.decode(value, { stream: true });
    let nl: number;
    while ((nl = buf.indexOf('\n')) >= 0) {
      const line = buf.slice(0, nl).trim();
      buf = buf.slice(nl + 1);
      if (!line) continue;
      try {
        onEvent(JSON.parse(line) as T);
      } catch {
        /* partial or malformed line */
      }
    }
  }
}
