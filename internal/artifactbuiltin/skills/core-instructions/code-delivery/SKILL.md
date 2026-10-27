---
name: code-delivery
description: Deliver complete implementation artifacts, coding-task checklists, and proposal-status updates.
insert: instructions
---

# Code delivery

Use for implementation tasks and requested code patches.

## Completion first

Complete the full requested change: contracts, implementation, callers, wiring, tests, and required removals. Finish every feasible requirement in this turn; defer only work with an evidenced blocker.

Own all implementation, integration, and verification. Run available checks; delegate only checks that require user access.

Complete every affected caller, migration, and integration handoff. Missing build or test access blocks only the checks that require it; deliver all source changes and tests supported by inspected contracts and conventions.

Use the available output budget for complete artifacts. Reduce explanation and repetition before implementation scope; a preferred response length is not a completion limit.

## Response artifacts

- New or deliberate replacement file: full contents and exact path.
- Existing modified file: unified diff against the exact repository path and identified source version.
- Include complete executable bodies, imports, fixtures, configuration, and required setup.
- Resolve existing APIs against inspected declarations; deliver required new contracts with their implementation and callers. When the relevant source is available, a missing API is implementation work, not a blocker.
- Name essential missing dependencies by exact evidenced path. For an unknown path, name the unresolved contract. Complete independent artifacts.
- Describe file renames and removals in plain sentences with exact paths.
- Use tool schemas for tool calls; keep response tracking outside implementation file contents.

## Progress

- Track authored, applied, and verified states separately; support each claim with observed evidence.
- Provide a unified diff for supplied proposal or status tracking document if provided.
- When a tracking source is unavailable, use the closing lists as the progress record.

## Coding-task closure

- End each implementation or code-patch response with exactly `## Implementation status`, then `## Next steps`, both containing concise Markdown task lists.
- Use `- [x]` for evidenced completed statements and `- [ ]` for pending requirements or verification.
- Account for every requirement with delivered artifacts or an evidenced blocker. Finish all unblocked work before closing; keep blocked implementation, integration, and verification assistant-owned.
- Limit user actions to review, patch application, authorization, exact external inputs, and checks that require user access.
- Put user actions and related verification caveats only under `### Expected user actions` within `## Next steps`; include check steps or commands and pass criteria. Report verification status in `## Implementation status`.
- For finished scope, include a checked completion item in `Next steps`.
