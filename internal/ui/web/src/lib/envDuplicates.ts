// A dotenv file setting a key twice means different things to different
// runtimes: Symfony's dotenv keeps the last assignment, Laravel's phpdotenv
// keeps the first, and lerd reads the first. Nothing can be right for all of
// them, so the editor shows the conflict and the user says which value they
// meant.

export interface EnvDuplicate {
  key: string;
  /** Every live occurrence, in file order. */
  occurrences: { line: number; value: string }[];
}

/** Keys the buffer sets more than once, ignoring commented-out lines. */
export function findDuplicates(text: string): EnvDuplicate[] {
  const seen = new Map<string, { line: number; value: string }[]>();
  text.split("\n").forEach((raw, i) => {
    const line = raw.trim();
    if (!line || line.startsWith("#")) return;
    const eq = line.indexOf("=");
    if (eq <= 0) return;
    const key = line.slice(0, eq).trim();
    if (!key) return;
    const value = line
      .slice(eq + 1)
      .trim()
      .replace(/^["']|["']$/g, "");
    const at = seen.get(key) ?? [];
    at.push({ line: i, value });
    seen.set(key, at);
  });
  return [...seen.entries()]
    .filter(([, occ]) => occ.length > 1)
    .map(([key, occurrences]) => ({ key, occurrences }))
    .sort((a, b) => a.key.localeCompare(b.key));
}

/**
 * Drops every occurrence of key except the one on keepLine, leaving the rest of
 * the file untouched so the result is an ordinary unsaved edit the user reviews
 * and saves like any other.
 */
export function keepOnly(text: string, key: string, keepLine: number): string {
  return keepOnlyEach(text, [{ key, line: keepLine }]);
}

/**
 * Resolves several keys at once against the buffer the line numbers were read
 * from. One pass, because dropping an occurrence renumbers every line under it:
 * resolving the keys one after another would leave the later choices pointing at
 * lines that have moved, and the file would lose keys nobody chose to drop.
 */
export function keepOnlyEach(text: string, choices: { key: string; line: number }[]): string {
  if (choices.length === 0) return text;
  const keep = new Map(choices.map((c) => [c.key, c.line]));
  const out = text.split("\n").filter((raw, i) => {
    const line = raw.trim();
    if (!line || line.startsWith("#")) return true;
    const eq = line.indexOf("=");
    if (eq <= 0) return true;
    const keepLine = keep.get(line.slice(0, eq).trim());
    return keepLine === undefined || keepLine === i;
  });
  return out.join("\n");
}
