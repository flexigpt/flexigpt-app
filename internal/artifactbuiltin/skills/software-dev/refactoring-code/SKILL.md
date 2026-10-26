---
name: refactoring-code
description: Complete bounded, behavior-preserving refactors that improve structure, testability, and change safety. Use for requested cleanup, extraction, simplification, or legacy containment.
---

# Refactoring Code

## Completion first

Complete the full authorized refactor, including characterization tests and every affected caller. Use coherent slices as checkpoints and continue through every ready slice.

## Scope and authority

- Preserve public behavior, contracts, security controls, denial paths, and error outcomes within the agreed scope.
- Treat an explicit refactoring request as authorization for ordinary in-scope planning and edits.
- Honor explicit approval gates; seek decisions for consequential changes outside the authorized scope.
- Use `spec-driven-dev` for behavior-changing architecture migrations.
- For analysis-only requests, deliver the requested slice selection or refactor recommendation.

## Work cycle

- Identify the requested improvement, preserved behavior, affected contracts, and completion criteria.
- Inspect supplied source first; discover the next required caller, seam, or test as work progresses.
- Establish the safety net and create required characterization tests in the same implementation pass.
- Choose a strategy that satisfies the requested boundaries and reduces concrete change risk.
- Implement dependency-coherent changes across affected code, callers, and tests.
- Preserve unrelated code, comments, and diagnostics.
- Run available formatters and relevant tests; compare outcomes with preserved behavior.
- Use observed failures to choose the next correction. Isolate exact missing dependencies and continue independent work.
- Account for the full requested refactor before returning.

## Strategies

- Simplify in place: clarify local control flow and invariants.
- Extract: isolate cohesive logic or shared policy.
- Split responsibilities: separate distinct ownership behind stable contracts.
- Introduce a facade or adapter: contain a dependency within the accepted boundaries.
- Strangler slice: use an authorized coexistence and routing boundary.
- Retain structure: preserve the existing design when it best satisfies the requested goal and safety constraints.

## Evidence and delivery

- Ground existing APIs and behavior claims in inspected declarations, callers, and tests.
- Record baseline failures when observed and distinguish them from failures introduced by the change.
- With repository tools, apply and verify. With attached source, deliver grounded patches and tests and track execution as pending.
- Provide full new or deliberate replacement files and exact-path unified diffs for existing modifications.
- Report the preserved behavior, actual delivery state, verification evidence, and precise remaining dependencies.
- End implementation responses with `## Implementation status` and `## Next steps` task lists. Keep remaining implementation and verification assistant-owned.
