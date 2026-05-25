---
name: decision-record-authoring
description: Write a decision record from context, options, evidence, and trade-offs. Produces a clear recommendation with explicit revisit triggers, not a generic pros-and-cons list.
---

# Decision Record Authoring

Write a decision record that takes a position. Recommendation, trade-offs accepted, and what would change the decision.

## Use when

- documenting a product, technical, or organizational decision
- choosing between options with material trade-offs
- recording a decision so future teams understand why
- producing an ADR-style artifact

## Do not use when

- the request is a product spec
- the request is an architecture decision needing system-level option comparison (use a designing system architecture skill/flow first, then this skill for the ADR)
- the request is feedback analysis

## Execution model

Workflow phases:

    context -> options -> evidence -> trade-offs -> recommendation -> revisit triggers

Take a position. A decision record without a recommendation is just a comparison. State the chosen option, why it wins, and what is being accepted in exchange.

Evidence vs assumption. Distinguish evidence (data, prior incidents, quoted constraints, cited sources) from assumption (reasonable belief without direct support).

Revisit triggers. State the conditions that would reopen the decision: metric thresholds, scale points, organizational changes, dependency shifts, or external events. Without revisit triggers, the record will silently rot.

## Hard rules

- Always state a clear recommendation; do not punt with "depends" or "team's choice".
- Compare at least two viable options when more than one exists; if only one is viable, name what eliminated the others.
- Separate evidence from assumptions.
- Do not invent data or quotes; cite sources.
- Include explicit revisit triggers.
- Trade-offs section must state what is being given up, not only what is gained.
- Works standalone; does not require any companion prompt.

## Artifact format

Use only sections that matter.

context: what is being decided, who is affected, constraints, status quo.

decision-needed: the specific question this record answers.

options-considered: each option with one-sentence summary, fit, evidence, and primary cost.

decision: the chosen option, in one paragraph.

rationale: why this option wins; how it satisfies hard constraints.

trade-offs: what is accepted as a cost or limitation by choosing this option.

risks: residual risks and the proposed mitigations.

revisit-triggers: concrete conditions that should reopen this decision.

next-steps: actions, owners, and rough timing implied by the decision.

## Response format

End each response with: `scope`, `confidence` (high|medium|low), `status`, key assumptions, `next-step` (only if useful).
