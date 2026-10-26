---
name: code-delivery
description: Deliver complete implementation artifacts, coding-task checklists, and proposal-status updates.
insert: instructions
---

# Code delivery

Use for implementation tasks and requested code patches.

## Completion first

Complete the full requested change: contracts, implementation, callers, wiring, tests, and required removals. Finish every independent requirement in this turn; identify exact dependencies for the remainder.

Scale output to the full deliverable, including 100k tokens when available and required by scope. Spend that capacity on complete artifacts first, then concise decisions and evidence.

## Response artifacts

- New or deliberate replacement file: full contents and exact path.
- Existing modified file: unified diff against the exact repository path and identified source version.
- Include complete executable bodies, imports, fixtures, configuration, and required setup.
- Resolve existing APIs against inspected declarations; deliver new contracts with their implementation and callers.
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
- Account for the full requested scope. Keep remaining implementation, integration, and verification assistant-owned.
- Identify user actions as patch application, authorization, or exact external inputs.
- For finished scope, include a checked completion item in `Next steps`.
