---
name: test-case-design
description: Design test scenarios from specs, requirements, bug reports, or code paths. Produces a traceable test plan and case list, not test code. Use when the user wants to plan tests before implementation, map coverage gaps, or design regression cases for a confirmed bug.
---

# Test Case Design

Design a test plan and case list traceable to requirements, bugs, behaviors, or code paths. Plan only; no code, no edits.

## Use when

- planning tests before implementation
- mapping coverage gaps in existing tests
- designing regression cases for a confirmed bug
- agreeing on what to test before writing it

## Do not use when

- the user wants test code written
- the user wants a quality review of existing code
- the user wants a fix or implementation

## Execution model

Workflow phases:

    objective -> behavior under test -> matrix -> case enumeration -> coverage gaps -> fixture and data design

Inputs. Prefer spec text, requirements, bug reports, attached code, or pasted snippets. Identify the behavior under test before enumerating cases. If only code is available, derive intended behavior from public contracts, names, and call sites; mark inferred behavior explicitly.

Behavioral matrix. For each behavior under test, enumerate inputs and modes that change outcome: input shape, size, type, validity, boundary, state preconditions, external dependencies, concurrency, configuration, and feature flags. The matrix drives case generation.

Traceability. Tie each case to a requirement, bug, behavior, or code path where possible. Cases without a tied source are exploratory; mark them as such.

Level placement. Each case must be a candidate for unit, integration, or end-to-end. Prefer the smallest level that exercises the behavior.

## Hard rules

- Do not modify files.
- Do not write actual test code.
- Tie each case to a requirement, bug, behavior, or code path where possible; otherwise mark exploratory.
- Separate happy-path, edge, negative, and regression cases.
- Mark each case as a unit, integration, or end-to-end candidate.
- Do not invent requirements or behaviors not supported by source material.
- Call out coverage gaps explicitly rather than silently skipping cases.
- Works standalone; does not require any companion prompt or shell access.

## Artifact format

Use only sections that matter.

test-objective: what the plan is for and what success looks like.

behavior-under-test: contract, expected outputs, side effects, invariants.

test-matrix: dimensions (inputs, states, configs) and the values per dimension that change behavior.

happy-path-cases: normal valid usage. ID | input/preconditions | expected | level | source tie.

edge-and-boundary-cases: empty, single, max, min, off-by-one, locale, time, encoding. Same columns.

negative-and-error-cases: invalid inputs, missing permissions, dependency failure, timeouts, schema violations.

regression-cases: cases tied to a known bug or past incident.

unit-integration-e2e-candidates: explicit assignment per case or per group; rationale for each placement.

coverage-gaps: behaviors that cannot be tested confidently with current evidence, and why.

data-and-fixture-ideas: representative inputs, test doubles, fixtures, fakes, or recorded payloads needed.

## Response format

End each response with: `scope`, `confidence` (high|medium|low), `status`, key assumptions, `next-step` (only if useful).
