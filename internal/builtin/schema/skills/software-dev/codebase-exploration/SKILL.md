---
name: codebase-exploration
description: Map an unfamiliar repo or module by identifying entry points, modules, flows, configs, tests, dependencies, and risks. Use when the user wants a map of code before deciding what to do next. Discovery-first; no architecture decisions, edits, or reviews.
---

# Codebase Exploration

Build a map of an unfamiliar code area from discovery and targeted reading. Stop at the map. Do not drift into review, refactor, or design.

## Use when

- a repo, module, service, or directory is unfamiliar
- the user wants a map before deciding what to do next
- onboarding into a new code area
- preparing for a later review, refactor, design, or change

## Do not use when

- the user wants design or architecture recommendations
- the user wants edits
- the user wants a code review
- the user wants bug diagnosis
- the user wants tests

## Execution model

Workflow phases:

    scope -> discovery -> read-batch -> map -> risk surfaces -> next steps

Context priority. Attached files are current state. Then pasted context and stated requirements. Then named workspace paths. Then adjacent files. Then targeted search. Then questions.

Discovery before reads. Before opening files, run a breadth-first discovery pass. Use listings, filename patterns, symbol and reference lookup, and targeted content search to find entry points, modules, handlers, data models, integrations, configs, and tests. Discovery produces the read-batch. Prefer outputs that return paths, matches, or small snippets over opening files one by one.

Batch everything. Identify the full discovery-batch and read-batch before calling tools. Execute together. Do not interleave single reads with single questions.

Stop at the map. When the map answers the user's question, stop. Do not start reviews, designs, or edits even if they feel obvious.

## Hard rules

- Discovery-first; build a single read batch before opening files.
- Do not recommend architecture or design changes unless explicitly asked.
- Do not claim uninspected areas are safe, correct, or risky.
- Prefer maps and concise explanations over exhaustive file-by-file summaries.
- Separate `observed` (seen in code) from `inferred` (likely from naming or context) from `assumed`.
- Do not invent files, symbols, dependencies, or behaviors.
- If the area is too large for one pass, return a partial map with what was not inspected and the recommended next slice.
- Works standalone; does not require any companion prompt or shell access.

## Artifact format

Use only sections that matter.

scope-inspected: what was actually opened or searched, language/stack observed, repo or module boundary.

entry-points: executables, services, route or handler registrations, CLI commands, build targets.

main-modules: logical groupings, what each module owns, dependencies between modules.

data-and-control-flows: how a typical request, job, or message flows; key state transitions; high-traffic paths.

configuration-and-environment-surfaces: config files, env vars, flags, feature toggles, secrets-handling surfaces.

tests-and-verification-surfaces: where tests live, test style observed, coverage signal.

external-dependencies-and-integrations: external services, libraries with significant influence on design, integration contracts.

risky-or-surprising-areas: unusual patterns, suspected dead code, sharp edges, untyped or untested zones.

what-to-inspect-next: short ordered list for the next exploration or work pass.

## Response format

End each response with: `scope`, `confidence` (high|medium|low), `status`, key assumptions or what was not inspected, `next-step` (only if useful).
