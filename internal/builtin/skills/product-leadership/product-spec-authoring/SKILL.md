---
name: product-spec-authoring
description: Turn goals, research notes, constraints, and stakeholder input into a product spec with testable requirements and acceptance criteria. Source-grounded; flags assumptions and missing inputs explicitly.
---

# Product Spec Authoring

Convert goals and source material into a product spec with testable requirements and acceptance criteria. Source-grounded. Mark assumptions explicitly.

## Use when

- producing a product spec from goals, research, or stakeholder input
- turning a rough brief into a structured, reviewable spec
- preparing a spec for engineering or design handoff

## Do not use when

- the request is implementation (move to engineering work)
- the request is a decision record between options
- the request is only feedback analysis

## Execution model

Workflow phases:

    problem framing -> users and goals -> non-goals -> requirements -> acceptance criteria -> risks and open questions -> handoff

Source-grounded. Use attached briefs, research notes, customer notes, support tickets, and pasted context. Cite source names when claims rely on them. Do not invent personas, market data, or behavior.

Requirements vs acceptance criteria. A requirement states what the system must do. Acceptance criteria are observable, testable conditions that confirm a requirement is met. Keep them separate.

Atomic and testable. Each requirement and each acceptance criterion should be small, independently verifiable, and free of ambiguous quantifiers. Replace "fast", "easy", "intuitive" with concrete thresholds, flows, or examples.

Diagrams only when they clarify. Include a Mermaid diagram only when a workflow, state transition, or sequence is hard to express in prose. Do not add diagrams for decoration.

## Hard rules

- Keep requirements testable, atomic, and unambiguous; avoid stacking multiple behaviors in one line.
- Separate requirements (R#) from acceptance criteria (AC#) and link AC# to R#.
- Mark assumptions explicitly; never treat them as facts.
- Do not invent users, market sizes, metrics, or competitor behavior; cite sources when used.
- List non-goals explicitly so scope creep is visible.
- Identify dependencies, risks, and open questions before declaring the spec ready for handoff.
- Works standalone; does not require any companion prompt.

## Artifact format

Use only sections that matter.

problem: what problem is being solved and why now; cite sources.

users-and-personas: who is affected; jobs to be done; relevant constraints.

goals: outcomes the spec is trying to achieve; success signals.

non-goals: explicit out-of-scope items.

functional-requirements: R# | requirement | source tie.

non-functional-requirements: performance, reliability, accessibility, privacy, compliance; quantify where possible.

constraints: technical, organizational, regulatory, or timeline constraints.

acceptance-criteria: AC# | criterion | linked R# | how to verify.

risks: product, user, technical, or delivery risks.

open-questions: explicit unknowns blocking confidence or completeness.

handoff-notes: what engineering or design needs to know to start.

## Response format

End each response with: `scope`, `confidence` (high|medium|low), `status`, key assumptions or missing inputs, `next-step` (only if useful).
