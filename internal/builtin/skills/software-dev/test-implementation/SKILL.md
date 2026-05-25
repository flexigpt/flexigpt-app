---
name: test-implementation
description: Write or update unit, integration, or regression tests, then verify with focused shell commands. Use when the user wants new tests written, a bug fix confirmed by tests, or backfilled coverage. Matches the existing test style.
---

# Test Implementation

Write or update tests, then verify with focused commands. Match the codebase's existing test style. Do not rewrite unrelated tests.

## Use when

- writing or updating unit, integration, or regression tests
- confirming a fix with new test coverage
- backfilling tests for known behavior
- closing a coverage gap identified earlier

## Do not use when

- the user only wants a test plan
- the user wants production code written
- the user wants a code review
- the user wants bug diagnosis

## Execution model

Workflow phases:

    discover test style -> choose targets -> plan cases -> implement -> focused verification -> report evidence

Discover existing style first. Before writing, inspect at least one existing test file in the same area to learn: framework, assertion style, naming convention, fixture strategy, test doubles approach, file location, and how integration vs unit tests are separated. If no existing tests exist, ask for the preferred framework before writing.

Plan before writing. List the cases to implement and their level (unit, integration, regression) before adding code. Cases should map to a requirement, bug, or behavior under test.

Implement narrowly. Add or modify only the tests required for the requested behavior. Do not refactor unrelated tests. Do not change production code unless explicitly asked.

Verify focused. Run only the new or modified tests where possible. Capture command output. If running tests is unavailable, mark verification `proposed only` and explain.

## Hard rules

- Match the existing test style and framework exactly; do not introduce a new framework unless asked.
- Prefer narrow, focused tests for the requested behavior.
- Do not rewrite unrelated tests, snapshots, or fixtures.
- Do not modify production code unless explicitly asked.
- Do not claim tests passed without command output captured in the artifact.
- Use shell only for focused test or build commands; explain briefly why each command is needed.
- If tests cannot be executed, say `proposed only` and do not claim verification.
- Do not invent helpers, fixtures, or test data that do not exist in the codebase without naming them as new additions.
- Works standalone; does not require any companion prompt.

## Artifact format

Use only sections that matter.

test-scope: what behavior is being covered, source tie (requirement, bug, code path).

existing-test-style: framework, naming, assertion, fixture, and structure observed; cite the example file.

planned-tests: ordered list of (case | level | target file | source tie).

files-changed: list of test files added or modified; brief reason per file.

verification-commands: exact commands to run only the new or modified tests.

verification-results: command output (success/failure summary); if unavailable, mark `proposed only`.

gaps: cases planned but not implemented, or behaviors not covered confidently, with reasons.

status: `complete`, `partial`, `blocked`, or `proposed only`.

## Response format

End each response with: `completed-phases`, `current-phase`, `delivery-mode` (direct edit | ready-to-apply output | analysis only), `status`, key assumptions or blockers, `next-step` (only if useful).
