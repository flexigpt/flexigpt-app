---
name: delivery-risk-review
description: Review plans, specs, milestones, issue lists, or status notes for delivery risks, dependency risks, blockers, unclear owners, and mitigations. Actionable, not theoretical.
---

# Delivery Risk Review

Find risks that could actually delay or derail delivery. Distinguish current blockers from future risks. Identify owners only when visible in source context.

## Use when

- reviewing a plan, spec, milestone list, or status note for risk
- preparing a pre-mortem before committing to a date
- assessing why delivery has slipped or might slip
- producing an actionable risk list for a steering meeting

## Do not use when

- the request is a single decision record
- the request is prioritization of new work
- the request is a status update for stakeholders

## Execution model

Workflow phases:

    scope -> risk inventory -> dependency map -> blockers -> mitigations -> decisions needed

Use source context only. Identify owners, dates, and dependencies only when they appear in the provided plan, issues, or notes. Do not invent owners.

Current blockers vs future risks. Blockers are things stopping work today. Risks are conditions that may cause delay or quality problems if not mitigated. Mix them and the artifact loses urgency signal.

Actionable mitigations. A mitigation must name the action, owner (if visible), and condition for being effective. Vague mitigations ("monitor closely") are not mitigations.

## Hard rules

- Focus on actionable risk; do not list generic risks that apply to any project.
- Separate current blockers from future risks explicitly.
- Identify owners only when they appear in source context; otherwise mark `owner unclear`.
- Tie each risk to evidence in the source material (plan, ticket, note).
- Distinguish dependency risk from execution risk from external risk.
- Do not invent dates, sequence, or scope.
- Works standalone; does not require any companion prompt.

## Artifact format

Use only sections that matter.

scope-reviewed: documents, tickets, milestones, or notes inspected; date range.

delivery-risks: risk | severity | likelihood | source tie | impact if it occurs.

dependency-risks: dependency | provider | consumer | criticality | latest status.

blockers: current blockers (work stopped today); owner if visible; oldest blocker first.

unclear-owners: items, decisions, or risks without an identifiable owner in source.

schedule-or-sequencing-concerns: parts of the plan whose order or timing increases risk.

mitigations: per risk or blocker: action | owner (if visible) | when it takes effect.

decisions-needed: explicit decisions required to unblock work or reduce risk.

## Response format

End each response with: `scope`, `confidence` (high|medium|low), `status`, key assumptions, `next-step` (only if useful).
