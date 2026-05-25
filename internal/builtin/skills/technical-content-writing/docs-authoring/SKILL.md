---
name: docs-authoring
description: Create or update READMEs, guides, how-tos, tutorials, conceptual docs, and internal docs from source context. Audience-aware and source-grounded. Includes diagrams only when they clarify the content.
---

# Docs Authoring

Write docs grounded in source material. Identify audience and prerequisites first. Use concrete examples. Include diagrams only when they earn their place.

## Use when

- creating or updating READMEs, user guides, how-tos, tutorials, conceptual docs, or internal docs
- writing onboarding content from code, design docs, or notes
- producing audience-appropriate documentation from source material

## Do not use when

- the user wants an audit of existing docs
- the user wants API reference
- the user wants release notes
- the user wants a troubleshooting article

## Execution model

Workflow phases:

    audience and purpose -> outline -> content -> examples -> diagrams when useful -> review against source

Audience and prerequisites first. State who the doc is for and what they should know before starting. This sets reading depth, terminology, and example complexity.

Outline before prose. Lock the outline before writing prose. A bad outline cannot be saved by good prose.

Concrete examples. Prefer examples derived from real source files or realistic scenarios. Avoid placeholder examples ("foo", "bar") unless the doc is about a generic mechanism.

Diagrams only when they clarify. Use a Mermaid diagram only when it adds real clarity for architecture, flow, lifecycle, state transitions, or sequences. Do not add diagrams for decoration.

Source review. Before finishing, cross-check claims against the inspected source files. Mark anything that is inferred rather than confirmed.

## Hard rules

- Identify audience and prerequisites before writing.
- Provide clear steps and concrete examples; not handwavy generalities.
- Use fenced code blocks with language tags for code; never nest triple backticks.
- Include Mermaid diagrams only when they clarify; never as decoration.
- Keep claims grounded in inspected source files; mark inferred claims.
- Do not invent commands, file paths, environment variables, or behaviors.
- Match the existing docs' tone and structure when extending a doc set.
- Works standalone; does not require any companion prompt.

## Artifact format

Use only sections that apply to the request.

overview: what the doc is, who it serves, what task it supports.

audience-and-prerequisites: explicit audience, assumed knowledge, required setup.

steps-or-sections: ordered steps or conceptual sections; each step has goal, action, expected result.

examples: concrete examples derived from source where possible; cite source paths.

limitations: known limits, gotchas, version constraints.

troubleshooting-notes: short pointers for the most common failures (escalate to `troubleshooting-guide-authoring` for full articles).

open-questions: source ambiguities or missing information that block confident claims.

## Response format

End each response with: `scope`, `confidence` (high|medium|low), `status`, key assumptions or what was not verified against source, `next-step` (only if useful).
