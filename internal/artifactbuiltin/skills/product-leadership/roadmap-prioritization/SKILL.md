---
name: roadmap-prioritization
description: Prioritize ideas, requests, bugs, and opportunities against goals using impact, effort, risk, and dependencies. Honest about missing inputs and uses relative ranking when evidence is qualitative.
---

# Roadmap Prioritization

Order items against goals. Be honest about evidence. Use relative ranking when numeric inputs are missing.

## Use when

- prioritizing a backlog, opportunity list, or bug set against goals
- producing a recommended sequence for a quarter, sprint, or release
- comparing items with material differences in impact, effort, risk, and dependencies

## Do not use when

- the request is a single decision record
- the request is a product spec
- the request is delivery risk review across a plan

## Execution model

Workflow phases:

    items -> criteria -> evidence -> scoring or ranking -> dependencies -> recommended order

Define criteria first. Anchor criteria to stated goals (e.g., "reduce time to first value", "increase activation", "reduce support load"). Do not start scoring before criteria and goals are explicit.

Evidence-aware scoring. Use numeric impact and effort only when supported by data; otherwise use relative ranking (`high`/`medium`/`low`, or ordinal pairs). Do not fabricate point estimates.

Dependencies and sequencing. A high-impact item is not first if it depends on something else. Surface dependencies and let them constrain ordering.

Missing inputs as first-class output. When critical inputs are missing (effort estimate, impact data, owner), say so in the artifact rather than guessing.

## Hard rules

- Do not invent impact, effort, or risk numbers.
- Use relative ranking when evidence is qualitative; do not force ICE/RICE-style numbers without data.
- Call out missing inputs explicitly and list which items would change priority once known.
- Tie each criterion to a stated goal.
- Respect dependencies in the recommended order.
- Distinguish "should do soon" from "should do first"; not every important item is the first item.
- Works standalone; does not require any companion prompt.

## Artifact format

Use only sections that matter.

items-considered: item | one-line summary | source.

prioritization-criteria: criterion | linked goal | how it is measured (numeric or relative).

impact: per item; mark `data-backed` or `inferred`.

effort: per item; mark `data-backed` or `inferred`.

risk: per item; types (technical, organizational, user, market) and severity.

dependencies: prerequisite items, external dependencies, sequencing constraints.

recommended-order: ordered list with one-line rationale per item.

what-would-change-priority: events, metrics, or learnings that would re-order the list.

## Response format

End each response with: `scope`, `confidence` (high|medium|low), `status`, key assumptions or missing inputs, `next-step` (only if useful).
