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

Context priority. Use supplied evidence first, then named paths, relevant neighbors, and targeted searches for unresolved mapping questions.

Progressive discovery. Read known relevant files directly. Use listings and symbol searches to locate the next required entry point, contract, dependency, or test.

Dependency-ready batches. Batch independent searches and reads. Use each result to select the next batch and extend the requested map.

Stop at the map. When the map answers the user's question, stop. Do not start reviews, designs, or edits even if they feel obvious.

## Hard rules

- Build successive dependency-ready batches from supplied evidence and newly discovered relationships.
- Do not recommend architecture or design changes unless explicitly asked.
- Do not claim uninspected areas are safe, correct, or risky.
- Prefer maps and concise explanations over exhaustive file-by-file summaries.
- Separate `observed` (seen in code) from `inferred` (likely from naming or context) from `assumed`.
- Do not invent files, symbols, dependencies, or behaviors.
- Complete the full requested map across successive batches; attach each remaining area to its specific missing evidence or access dependency.
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
