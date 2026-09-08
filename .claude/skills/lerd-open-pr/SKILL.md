---
name: lerd-open-pr
description: Draft and open a pull request the lerd way — human prose, none of the banned sections, and the fork's own gh and branch rules. Use when asked to open, prepare, or draft a PR for lerd. Never creates anything on GitHub without explicit per-action approval.
---

# Open a lerd PR

Every write to GitHub (create/edit/close/reopen/merge/comment on an issue or PR)
needs explicit approval **each time**. Draft the text, show it, and wait. Do not
run `gh pr create` or any state-changing `gh` command until the human says go.

## Before the PR

1. Run `/lerd-preflight` — the PR is not ready until the local gate is green.
2. Confirm the branch is off `oracle-oci8-support`, this fork's default branch,
   and not that branch itself, and staged by explicit path (never `git add -A`).
   `git status` first.
3. Pass `--repo gabriel-sousa99/lerd` to every `gh` command, or set it once with
   `gh repo set-default gabriel-sousa99/lerd`. With two remotes and no default,
   `gh` resolves to the upstream parent and `gh pr create` fails with
   `No commits between ... / Head ref must be a branch`, which does not name the
   real cause.

**Issues are disabled on this fork**, so there is no issue to open first and
nothing to link. Upstream's issue-first step and its `Closes #N` / `Refs #N`
linking do not apply here; a PR stands on its own body.

## PR body — write it as a human would

Prose paragraphs, single-line (no column wrapping), explaining what changed and
why.

## Never include

- A Test plan section.
- A Verified / Tested / Manual testing section or trailer.
- A checklist of any kind (`- [ ]` / `- [x]`, "Release checklist"…).
- A "Notes for reviewers" section — we own the project, there is no external reviewer.
- `file:line` citations, em dashes, `Co-Authored-By`, or "Generated with…" footers.
- Prose about tests, TDD, or coverage; and don't mention incidental cleanup.

## PR and issue comments

Casual plain prose. No markdown, no bullets, no hyphens — commas instead. Don't
open with boilerplate like "Pulled it down and put it on my install." Vary it.

## After pushing

Return immediately. Don't sit polling CI; failures get flagged by the human.
