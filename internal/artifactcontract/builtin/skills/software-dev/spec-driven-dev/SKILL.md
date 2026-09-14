---
name: spec-driven-dev
description: Plans, scopes, implements, and verifies bounded software changes from workspace files, attached files, pasted code, diffs, or logs. Use for features, bug fixes, refactors, enhancements, or investigations that benefit from explicit requirements, tracked implementation, and evidence-based verification.
---

# Spec-Driven Development

Deliver bounded software changes through strict phases, batched work, and evidence-based verification. Keep the workflow proportional to the change. No ceremony for its own sake.

## Execution model

- Phases flow in strict order. Start at the earliest incomplete phase.

  Discover → Write → user confirms spec → Implement → Verify

**Context priority.** Attached files are current state. Then pasted code, diffs, logs, stated requirements. Then named workspace paths. Then adjacent workspace files. Then targeted search for a specific unknown. Then questions. Do not skip earlier layers while they have unused evidence.

**Discovery before reads.** Before reading workspace files, first do a breadth-first discovery pass. Use provided context plus cheap discovery tools such as listings, filename matches, symbol/reference lookup, and targeted content search to identify likely files, neighbors, tests, interfaces, and specific regions worth reading. Let discovery build the read-batch. Prefer discovery outputs that return paths, symbols, matches, or small snippets over opening files one by one. Do not make behavioral claims from discovery alone.

**Batch everything.** Before any tool call, identify the full discovery-batch, read-batch, write-batch, and verify-batch. Execute together. Do not interleave single reads with single questions or edit one file at a time when multiple targets are clear.

**When blocked.** Exhaust all available context first. Then ask 1-3 true blocker questions in one batch. Include any partial progress before asking. When the user wants speed, proceed provisionally on non-blocking uncertainty, but do not bypass spec confirmation.

**Re-read the active artifact** after external edits or after 5+ tool actions since the last read.

**Maximize progress per turn.** If some work is unblocked and some is blocked, do the unblocked work now and ask only the remaining blockers. If the next phase is already unblocked, enter it immediately unless that would cross the spec confirmation gate. Discover and Write may be completed in the same response. So can Implement and Verify.

**Human in loop artifact confirmation** applies only to the written spec artifact. Present the spec artifact and ask the user to confirm or modify it. If the user requests changes, update the spec and ask again. Proceed to Implement and Verify only after the user confirms. If new evidence later changes scope, interfaces, or public behavior, update the spec and ask for confirmation again before continuing.

## Hard rules

- Read before writing. Reuse the user's and codebase's exact terms.
- Never invent paths, symbols, requirements, commands, or test results.
- Keep claims proportional to what was actually seen. Mark unseen callers, dependencies, or tests as assumptions.
- Do not re-read files already provided as attachments unless comparing versions or checking drift.
- Do not search while direct evidence from attachments, pasted context, or named paths remains unused.
- Do not do work beyond the spec or stated request. Record useful follow-ups instead of expanding scope.
- Do not change the spec silently. If new evidence changes scope or interfaces, update the spec first.
- Do not claim files were modified unless actually modified in an accessible workspace.
- Do not delete unrelated comments or debugging statements; update only where logic changed.
- If work originates from review findings, convert findings into explicit requirements and edge cases before editing.
- If the task touches auth, authz, sessions, public APIs, templates, callbacks, file-handling, or multi-tenant behavior, include denied or misuse edge cases.
- If no writable workspace exists, produce ready-to-apply output for all targets together.
- Be explicit about what was seen directly versus what is inferred or unverified.
- User instructions override this skill.

## 3-strike rule

Applies in Implement and Verify. On failure:

1. Try a direct fix.
2. Try a materially different method or hypothesis.
3. Broaden the investigation.

After 3 failed approaches, mark the task `blocked`, record what was tried, and ask the user.

## Phase reference

### Discover

Entry: the user goal exists and at least one anchor (file, symbol, path, error, behavior area).

1. Start from provided context.
2. Before reading additional workspace files, do a breadth-first discovery pass to collect likely targets, direct neighbors, tests, interfaces, and specific regions worth reading.
3. Read as much as possible (including direct neighbors like callers, callees, tests, interfaces) in one batch.
4. Prefer another discovery pass before another serial file-read pass. Go deeper only if specific blockers remains.
5. Stop when the edit surface is clear or the question is answered.

Do not turn discovery into a repo tour. Do not overclaim about unseen callers or coverage. When the user wants discovery only, do not drift into implementation.

Produce `discovery` artifact.

### Write

Entry: goal understood, affected area known.

1. Convert facts and intent into the smallest implementable spec.
2. Fill known sections. Put unknowns in `Assumptions` or `Open Questions`, not in `Requirements`.
3. Requirements must be concrete and testable. Use IDs: `R1`, `R2`, `E1`, `E2`.
4. Use real paths from inspection. If target coverage is still incomplete, do one more discovery pass before reading more files or asking questions. Mark unconfirmed paths `(unverified)` under `Assumptions`.
5. If multiple materially different approaches exist, recommend one and ask before implementing.
6. If implementation is unblocked, do not start it yet. Present the spec artifact and ask the user to confirm or modify it before starting the next phases.

Produce `spec` artifact. `Status` stays `provisional` until the user confirms. Once the user confirms, start the next phases.

### Implement

Entry: bounded change, identified targets, spec confirmed, no blocker that would change scope, interfaces, or public behavior.

1. If no spec exists but the change is clear, create a minimal spec first.
2. If the spec is `provisional`, ask for confirmation or modification and stop.
3. Create `progress` plan mapped to R# before editing.
4. Read all write-batch targets plus required adjacent context together.
5. Edit in dependency order: contracts → utilities → implementations → callers → tests/docs.
6. One bounded pass. Prefer whole-file rewriting when changes span multiple nearby regions. Use targeted replacement only for simple exact substitutions.
7. After the batch, verify narrowly and update `progress` once.

If runtime checks are unavailable, record `code-reviewed only`. If the change was produced as output only, record `proposed change only`.

Produce `progress` artifact.

### Verify

Entry: expected behavior or spec available, implementation visible.

1. Gather fresh evidence for every R# and E# in one batch.
2. Compare implementation to the exact requirement text, not intent alone.
3. Record status and evidence for each requirement and edge case.
4. State verdict.

Fresh evidence means: code inspection, test output, build or lint output, command output, or explicit comparison between code and requirement text gathered during this pass.

Never claim done, fixed, complete, or passing without fresh evidence. If evidence is incomplete, verdict stays `partially verified` or `not verified`. Say `code review only` if no runtime checks. Say `proposed change only` if not applied to workspace. If the spec is provisional, note reduced confidence under `Gaps`.

Produce `verification` artifact.

## Artifact formats

Artifacts live in the conversation. Keep them minimal. Update instead of rewriting.

**discovery**: `request-goal`, `sources-read`, `observed`, `inferred`, `candidate-targets`, `current-behavior`, `blockers`, `next-step`, `confidence` (high|medium|low).

**spec**: `Title` (Spec: name), `Status` (confirmed|provisional), `Goal`, `Context`, `Requirements` ([ ] R1...), `Out of Scope`, `Technical Approach`, `Interfaces`, `Data Models`, `Change Targets`, `Edge Cases` (E1...), `Decisions`, `Assumptions`, `Open Questions`. Use only fields that matter for the change.

**progress**: `Title` (Progress: name), `Status` (in-progress|partial|blocked|complete), `Assumptions`, `Plan` (# | pending/done/blocked | Objective | Targets | Depends on | Req | Verify by), `Task Log` (# | Evidence | Notes), `Deviations`. Create plan before editing. Keep numbering stable.

**verification**: `Title` (Verification: name), `Status`, `Results` (R# | pass/partial/fail/not verified | Evidence), `Edge Case Results` (E# | same), `Verdict` (verified|partially verified|not verified|failed), `Gaps`, `Follow-ups`.

## Response format

End each response with: `completed-phases`, `current-phase`, `delivery-mode` (direct edit|ready-to-apply output|analysis only), `status`, key assumptions or blockers, `next-step`.
