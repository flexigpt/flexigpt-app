---
name: status-update-authoring
description: Convert project notes, changelogs, meeting notes, or issue summaries into stakeholder-ready status updates. Avoids internal detail unless it affects status, risk, or decisions.
---

# Status Update Authoring

Turn raw project material into a status update a stakeholder can read in two minutes. Drop internal noise. Keep risks and decisions visible.

## Use when

- writing a weekly, sprint, or milestone status update
- summarizing project notes for leadership or cross-team stakeholders
- preparing a status section of a steering document

## Do not use when

- the request is a delivery risk review
- the request is release notes
- the request is a decision record

## Execution model

Workflow phases:

    gather sources -> highlight progress -> identify risks and blockers -> decisions needed -> frame for stakeholders

Audience-aware. Decide who the update is for (exec, peer team, customer-facing) and tune detail accordingly. The same project may have three different updates.

Stakeholder filter. Include internal detail only when it changes status, risk, or a decision the audience must make. Drop the rest.

Stable structure. Use a consistent shape (summary at top, progress, risks, decisions, next steps) so readers can scan. Do not bury the lede.

## Hard rules

- Lead with a one-paragraph summary; do not bury status under a chronology.
- Keep updates stakeholder-ready; avoid internal jargon unless the audience uses it.
- Include risks and blockers honestly; do not soften them out of existence.
- Do not invent progress, dates, or owners; use only source material.
- Decisions-needed must name the decision, the audience, and the deadline if known.
- Works standalone; does not require any companion prompt.

## Artifact format

Use only sections that matter.

summary: one-paragraph plain-language summary; status (on track | at risk | off track | blocked).

progress-since-last-update: concrete outcomes; cite source where useful.

changes-or-shipped-work: releases, milestones reached, scope changes, customer-visible changes.

risks: top risks; impact and trend (new, growing, steady, shrinking).

blockers: current blockers; owner if visible; what they are waiting for.

decisions-needed: decision | audience | deadline if known.

next-steps: focused list for the next period; not a full backlog.

## Response format

End each response with: `scope`, `confidence` (high|medium|low), `status`, key assumptions, `next-step` (only if useful).
