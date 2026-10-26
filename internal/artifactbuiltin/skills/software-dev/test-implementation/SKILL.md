---
name: test-implementation
description: Deliver complete unit, integration, or regression tests from workspace or attached code, using supported repository conventions and evidence-based verification.
---

# Test Implementation

Complete the requested test coverage and all required fixtures and helpers in this turn.

## Scope

- Implement the requested tests and supporting test artifacts.
- Apply production-code changes when explicitly included in the authorized task.
- Preserve unrelated tests, fixtures, and snapshots.

## Execution

- Map every requested behavior and edge case to a test and its expected outcome.
- Inspect available neighboring tests, contracts, and dependency declarations.
- Match the existing framework and style. When examples are absent, use a declared framework or a suitable supported standard-library facility.
- Implement complete tests, fixtures, and helpers together; identify newly introduced support code explicitly.
- Prefer the smallest test level that exercises the required behavior.
- Run focused tests first, then affected suites and build checks.
- Record actual commands, working directories, outcomes, and relevant failure details.
- Repair failures caused by the test change. Isolate exact contract or environment blockers and complete independent cases.
- With attachment-only delivery, source-check the tests and record execution as pending.

## Delivery

- Provide full new or deliberate replacement files and exact-path unified diffs for existing modifications.
- Report covered cases, remaining requirements, and verification evidence.
- Distinguish authored tests, applied changes, and executed checks.
- End implementation responses with `## Implementation status` and `## Next steps` task lists.
- Keep remaining test implementation, integration, and verification assistant-owned.
