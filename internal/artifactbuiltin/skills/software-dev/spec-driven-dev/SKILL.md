---
name: spec-driven-dev
description: Implement complete specification-led changes through progressive discovery, coherent implementation, and evidence-based verification. Use for features, fixes, enhancements, and architecture migrations.
---

# Spec-Driven Development

## Completion first

Complete the entire authorized deliverable. Implement every independent requirement and its required integration in this turn.

Treat phases as sequencing checkpoints. Continue through every ready phase, delivering complete contracts, implementations, callers, setup, and tests.

- Make code, integration, and tests the primary output of an implementation turn. Plan enough to select the next edit, then implement it.
- Maximize single-turn delivery: execute independent edits and checks concurrently where supported; continue through dependent waves until the authorized scope is delivered or an exact limit is reached.

## Authority

- Match the requested deliverable: analysis for analysis requests, implementation for implementation requests.
- Use the accepted proposal and explicit requirements as the target; use existing code as implementation evidence.
- Treat an implementation request as authorization for ordinary in-scope decisions. Honor explicit approval gates and seek decisions for consequential changes outside that authorization.
- Choose the smallest design satisfying the full requested scope.
- Reuse contracts and repository abstractions that fit the target boundaries. Implement required replacements and removals.

## Work cycle

- Track requirements, edge cases, affected files, and verification. Reuse supplied requirement IDs.
- Start with supplied source. Inspect the contracts and dependencies needed for the next coherent change.
- Batch independent work; use each result to select the next dependency-ready batch.
- Implement in dependency order across contracts, logic, callers, bindings, persistence, configuration, and tests as applicable.
- Preserve unrelated code and reuse established filesystem, concurrency, validation, and error-handling mechanisms.
- Add required tests with the implementation, including relevant boundary, denial, failure, recovery, and integration cases.
- Run relevant formatters, focused tests, and repository-declared type, lint, and build checks. Expand verification according to the affected contracts.
- Check required removals and dependency boundaries with targeted searches.
- Reconcile every requirement with implementation and verification evidence. Finish remaining independent work before responding.

## Capabilities and blockers

- Repository tools available: apply authorized changes and execute relevant checks.
- Attached source available: deliver grounded implementation and tests; record execution checks as pending.
- Essential contract missing: identify its exact evidenced path, or its unresolved symbol when the path is unknown; isolate dependent requirements and complete the rest.
- Environment or toolchain gap: identify the affected command or operation and continue independent work.
- Failed attempt: use the observed result to choose a materially different next action; repair supported failures and continue independent work.
- Record actual commands, working directories, outcomes, and precise external unblocks. Keep implementation and verification assistant-owned.

## Delivery

- For response artifacts, provide full new or deliberate replacement files and exact-path unified diffs for existing modifications.
- For direct edits, summarize actual changes and verification evidence.
- Update supplied proposal progress from the same evidence.
- Keep authored, applied, and verified states distinct.
- End implementation responses with `## Implementation status` and `## Next steps` task lists, accounting for the full requested scope.
