---
name: release-notes-authoring
description: Create release notes and changelogs from diffs, commit notes, PR descriptions, issue summaries, or version notes. Distinguishes user-facing changes from internal changes; never invents changes.
---

# Release Notes Authoring

Turn change material into release notes a user can actually use. Group by impact. Call out breaking changes loudly.

## Use when

- producing release notes or a changelog entry from a diff, PR list, commit log, or issue summary
- consolidating multiple PRs into a single release entry
- summarizing what changed for end users or operators

## Do not use when

- the user wants a status update
- the user wants conceptual docs
- the user wants troubleshooting content

## Execution model

Workflow phases:

    gather changes -> categorize -> identify breaking changes -> migration notes -> finalize

User-facing vs internal. Most readers care about behavior changes that affect them. Demote internal refactors and CI changes unless they affect downstream users.

Breaking changes are loud. A breaking change section that is short or hidden under "Other" is a release-notes failure. Lead with breaking changes when present.

Migration notes. For breaking changes, include concrete steps: what to change, what to test, what to roll back to if needed.

Draft mode. If the source material is incomplete (missing PRs, ambiguous commits), mark the release notes as `draft` and list what is missing.

## Hard rules

- Do not invent changes; every entry must tie to a source (PR, commit, issue, diff).
- If source material is incomplete, mark notes as `draft` and list missing items.
- Separate user-facing changes from internal changes.
- Lead with breaking changes when present.
- Include migration notes for every breaking change.
- Keep language concrete: "added X for Y" beats "improved experience".
- Note known issues when source indicates them; do not hide regressions.
- Works standalone; does not require any companion prompt.

## Artifact format

Use only sections that matter.

highlights: one-paragraph summary of what matters in this release.

added: new capabilities, endpoints, options.

changed: behavior changes that affect users; defaults updated.

fixed: bugs fixed; cite issue or PR.

breaking-changes: each with what changes, who is affected, and source tie.

migration-notes: steps per breaking change; before/after examples where useful.

known-issues: regressions, open bugs noted in source.

## Response format

End each response with: `scope`, `confidence` (high|medium|low), `status` (`draft` if material is incomplete), key assumptions or missing source, `next-step` (only if useful).
