import { describe, it, expect } from "vitest";
import { findDuplicates, keepOnly, keepOnlyEach } from "./envDuplicates";

describe("env duplicates", () => {
  const file = [
    "# DATABASE_URL=mysql://commented",
    "DATABASE_URL=postgresql://postgres@lerd-postgres:5432/app",
    "APP_ENV=dev",
    "",
    "# lerd debug test",
    'DATABASE_URL="sqlite:///var/app.db"',
  ].join("\n");

  it("reports a key set twice, ignoring a commented value", () => {
    const dupes = findDuplicates(file);
    expect(dupes).toHaveLength(1);
    expect(dupes[0].key).toBe("DATABASE_URL");
    expect(dupes[0].occurrences.map((o) => o.value)).toEqual([
      "postgresql://postgres@lerd-postgres:5432/app",
      "sqlite:///var/app.db",
    ]);
  });

  it("says nothing about a file that sets each key once", () => {
    expect(findDuplicates("APP_ENV=dev\nDATABASE_URL=x\n")).toEqual([]);
  });

  it("keeps the chosen occurrence and drops the other, leaving the rest alone", () => {
    const dupes = findDuplicates(file);
    const keepLast = dupes[0].occurrences[1].line;
    const out = keepOnly(file, "DATABASE_URL", keepLast);

    expect(out).toContain('DATABASE_URL="sqlite:///var/app.db"');
    expect(out).not.toContain("postgresql://");
    // everything else survives, comments included
    expect(out).toContain("APP_ENV=dev");
    expect(out).toContain("# DATABASE_URL=mysql://commented");
    expect(out).toContain("# lerd debug test");
    expect(findDuplicates(out)).toEqual([]);
  });

  it("can keep the first instead", () => {
    const dupes = findDuplicates(file);
    const out = keepOnly(file, "DATABASE_URL", dupes[0].occurrences[0].line);
    expect(out).toContain("postgresql://");
    expect(out).not.toContain("sqlite:///var/app.db");
  });

  // Resolving one key drops lines anywhere in the file, above the kept line as
  // well as below it, so every line number taken from the same reading of the
  // buffer has to be resolved against that reading. Applying them one after
  // another renumbers the file under the choices still to come, and the file
  // silently loses keys the user never chose to drop.
  it("resolves several keys against the file the user was looking at", () => {
    const many = [
      "DB_HOST=old",
      "DB_DATABASE=old",
      "APP_ENV=local",
      "DB_HOST=lerd-mysql",
      "DB_DATABASE=acme",
    ].join("\n");
    const dupes = findDuplicates(many);
    const keepLast = dupes.map((d) => ({
      key: d.key,
      line: d.occurrences[d.occurrences.length - 1].line,
    }));

    const out = keepOnlyEach(many, keepLast);
    expect(out.split("\n")).toEqual(["APP_ENV=local", "DB_HOST=lerd-mysql", "DB_DATABASE=acme"]);
    expect(findDuplicates(out)).toEqual([]);
  });

  it("resolves a mix of first and last choices in one pass", () => {
    const many = ["A=1", "B=1", "A=2", "B=2", "A=3"].join("\n");
    const out = keepOnlyEach(many, [
      { key: "A", line: 0 },
      { key: "B", line: 3 },
    ]);
    expect(out.split("\n")).toEqual(["A=1", "B=2"]);
  });

  it("leaves a file alone when nothing was chosen", () => {
    expect(keepOnlyEach(file, [])).toBe(file);
  });
});
