---
name: docs-audit
description: Read-only audit of docs or README files for structural gaps, stale claims, duplication, missing content, and navigation issues. Produces an audit and rewrite plan; does not rewrite docs.
---

# Docs Audit

Audit existing docs against their audience and purpose. Produce findings and a rewrite plan. Do not rewrite docs or edit files in this skill.

## Use when

- assessing the state of READMEs, guides, or doc sites
- identifying gaps, stale content, duplication, or navigation problems
- producing a rewrite or restructuring plan
- preparing for a docs refresh

## Do not use when

- the user wants docs written or rewritten
- the user wants API reference generated
- the user wants release notes

## Execution model

Workflow phases:

    inventory -> audience and purpose -> structural gaps -> content gaps -> severity -> rewrite plan

Audience and purpose first. Identify who each doc is for and what task it should support. A README that targets new users is a different doc from one targeting contributors. Findings only make sense relative to audience and purpose.

Cite every finding. Each finding names the file or section it refers to. Findings without a citation are noise.

Severity, not adjectives. Use severity (`critical`, `major`, `minor`) tied to user impact, not subjective adjectives ("messy", "boring").

## Hard rules

- Read-only: do not edit, create, delete, or rewrite files.
- Cite files or sections for every finding.
- Do not rewrite docs; produce an audit and a rewrite plan only.
- Tie findings to audience and purpose, not to personal style preference.
- Do not invent claims about content that was not inspected.
- Mark "possibly stale" claims as `unverified` unless source code or release notes confirm.
- Works standalone; does not require any companion prompt.

## Artifact format

Use only sections that matter.

docs-inventory: files reviewed, approximate sizes, last-modified dates if visible.

audience-and-purpose: per major doc, who it serves and what task it supports.

structural-gaps: missing sections, wrong ordering, missing prerequisites, weak onramps.

missing-pages-or-sections: topics absent from the doc set; cite where they would belong.

stale-or-unsupported-claims: claims that look out of date or are unsupported by code/releases; mark `confirmed` or `unverified`.

duplication: overlapping content across pages; consolidation candidates.

navigation-issues: discoverability problems, broken or unclear links, missing index or TOC.

severity: per finding (`critical`|`major`|`minor`) with user-impact reasoning.

recommended-rewrite-plan: ordered set of changes; estimated scope per change; what would be a quick win vs deeper work.

## Response format

End each response with: `scope`, `confidence` (high|medium|low), `status`, key assumptions or what was not inspected, `next-step` (only if useful).
