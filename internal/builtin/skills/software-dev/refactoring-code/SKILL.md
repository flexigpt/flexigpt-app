---
name: refactoring-code
description: Selects and performs bounded, behavior-preserving refactors that reduce complexity, duplication, coupling, and change risk without turning into a rewrite. Use when the user asks to refactor code, clean up a risky module, modernize a legacy area safely, or decide what technical debt slice to tackle next from workspace access, attached files, or pasted code.
---

# Refactoring Code

Choose the smallest safe refactor that materially improves structure or change safety while preserving behavior. Bounded cleanup and legacy containment, not rewrites.

## Use when

- refactoring a file, module, or subsystem safely
- reducing duplication, splitting a god class, simplifying complexity, improving testability
- modernizing a legacy area incrementally without breaking behavior
- deciding which technical-debt slice to tackle first

## Do not use when

- implementing a new feature from scratch
- the question is system architecture direction
- the request is only style, lint, or formatter cleanup
- the request is a broad rewrite or full-repo cleanup
- public behavior is expected to change substantially without agreement

## Execution model

Modes flow in order. Start at the earliest incomplete mode.

    slice-selection → refactor-plan → user confirms plan → refactor-apply

**Context priority.** Attached files are current state. Then pasted code, diffs, logs, stated requirements. Then named workspace paths. Then adjacent workspace files. Then targeted search for a specific unknown. Then questions. Do not skip earlier layers while they have unused evidence.

**Batch everything.** Before any tool call, identify the full read-batch, write-batch, and verify-batch. Execute each batch together. Do not interleave single reads with single questions or edit one file at a time when multiple targets are clear.

**Maximize progress per turn.** If some work is unblocked and some is blocked, do the unblocked work now and ask only the remaining blockers. If the next mode is already unblocked, enter it immediately unless that would cross the plan confirmation gate. `slice-selection` and `refactor-plan` may be completed in the same response.

**When blocked.** Exhaust all available context first. Then ask 1-4 true blocker questions in one batch. Include any partial progress before asking.

**Re-read the active artifact** after external edits or after 5+ tool actions since the last read.

**Human in loop plan confirmation** applies only before `refactor-apply`. Present the chosen slice, preserved behavior, safety net, strategy, and refactor plan. Ask the user to confirm or modify. If the user requests changes, update the plan and ask again. Proceed to `refactor-apply` only after the user confirms. If the slice, preserved behavior, public interface impact, or strategy changes materially later, update the plan and ask again before continuing.

## Hard rules

- Read before writing. Reuse the user's and codebase's exact terms.
- Never invent paths, symbols, commands, or test results.
- Keep claims proportional to what was actually seen. Mark unseen callers, dependencies, or tests as assumptions.
- Do not re-read files already provided as attachments unless comparing versions or checking drift.
- Do not search while direct evidence remains unused.
- Preserve external behavior unless the user explicitly approves a change.
- For security-sensitive code (auth, authz, sessions, validation, multi-tenant), preserve control behavior and negative-path outcomes, not just happy-path outputs.
- If tests are missing for behavior that must be preserved, the first slice is the safety net.
- If the boundary is unclear, the first slice is seam creation or a facade, not deep internals.
- If public behavior or interfaces would change, ask before proceeding.
- If the slice requires many unrelated files, split it.
- If the refactor would become a rewrite, stop and re-scope.
- If the real issue is architectural, say so rather than forcing a local refactor.
- If no meaningful gain is likely, recommend leaving it alone.
- Prefer containment over replacement for risky legacy dependencies.
- Prefer extraction over redesign when pain is localized.
- Prefer one completed slice over a large partial cleanup.
- Never recommend a rewrite when a smaller seam, adapter, extraction, or containment move would work.
- If the baseline already has failing tests, record the failing set before editing.
- Do not fabricate ROI, saved hours, or debt scores.
- Do not claim files were modified unless actually modified in an accessible workspace.
- Do not delete comments or debugging statements outside the chosen slice; update them inside the slice where logic changed.
- User instructions override this skill.

## 3-strike rule

On failure during refactor-apply:

1. Try a direct fix.
2. Try a materially different approach.
3. Broaden the investigation.

After 3 failed approaches, mark `blocked`, record what was tried, and ask the user.

## Mode reference

### slice-selection

Entry: a code area or pain point is identified with at least some visible code context.

1. Lock the target: exact slice, pain being reduced, behavior that must stay unchanged, what is out of scope.
2. Gather hotspot evidence from available context: bug locations, high churn, complexity, duplication, dependency friction.
3. Choose the smallest safe slice: high pain, locally concentrated, boundary-clear, verifiable. Heuristic: ~3-5 core files, ~500 lines of change, completable in 1-5 focused days. Not hard limits.

Exit: bounded slice chosen with clear scope and preserved behavior stated.

### refactor-plan

Entry: bounded slice chosen, preserved behavior known.

1. Establish the safety net: existing tests, characterization tests needed, or manual verification steps. Do not start structural changes when preservation cannot be checked.
2. Choose strategy from the catalog below.
3. Plan the slice: exact targets, sequence, verification after each substep, rollback path, follow-on slices if useful.
4. Present the chosen slice, preserved behavior, safety net, strategy, and refactor plan. Ask the user to confirm or modify before `refactor-apply`.

Exit: concrete plan with strategy, safety net, and verification steps.

### refactor-apply

Entry: plan exists, plan confirmed, safety net established or explicitly accepted as absent, user requested code changes.

1. Read all write-batch targets plus required adjacent context together.
2. Apply changes in small steps, verify preserved behavior after each meaningful substep.
3. One bounded pass. Prefer whole-file rewriting when changes span multiple nearby regions.
4. If writable workspace is available, apply there. Otherwise return ready-to-apply output for all targets together.
5. Stop if the slice grows beyond its original boundary.

Exit: refactor complete, or remaining gaps marked `partial`/`blocked`.

## Strategy catalog

**simplify-in-place**: Complexity is local, boundaries acceptable. Split long functions, replace nested branching with guards, isolate pure calculations from side effects.

**extract-function-or-module**: Duplication exists or a coherent responsibility can be isolated while callers stay stable. Shared validation, repeated parsing, reusable policy evaluation.

**split-by-responsibility**: Mixed responsibilities with visible internal seams and stable public contract. Separate persistence from business rules, orchestration from transformation.

**introduce-facade-or-adapter**: Legacy dependency leaks through many callers or upgrade blocked by direct coupling. Wrap legacy clients, isolate framework code, translate unstable external contracts.

**strangler-slice**: Only when subsystem boundary is clear, routing seam exists, and old/new coexistence is feasible. Not the default modernization answer.

**leave-alone**: Pain is low, churn is low, risk of refactor exceeds benefit, or the real issue is elsewhere.

## Refactor drivers

Use to identify whether refactoring is justified:

- duplicated critical logic
- long/deeply nested functions obscuring invariants
- god classes with mixed responsibilities
- hidden coupling causing ripple edits
- fragile flag-based branching
- side effects mixed into pure logic
- legacy dependency spread through many callers
- inability to test risky logic without large setup
- recurring bugs clustering in one area
- dependency upgrade blocked by missing abstraction

If none are meaningfully present, consider `leave-alone`.

## Artifact format

Artifacts live in the conversation. Keep them minimal. Update instead of rewriting. Use only sections that matter.

**refactor-context**: `request-goal`, `observed`, `inferred`, `missing-inputs`, `mode`, `delivery-mode` (direct edit|ready-to-apply output|analysis only), `confidence` (high|medium|low).

**problem-hotspots**: files/surfaces considered, why each is a candidate, evidence.

**chosen-slice**: exact scope, why this slice first, what is out of scope.

**behavior-to-preserve**: public behavior, internal invariants, caller/contract constraints.

**safety-net**: existing tests, characterization tests needed, manual verification steps.

**strategy**: chosen strategy, why it fits, rejected alternatives (brief).

**refactor-plan**: ordered list of (step | targets | purpose | verification).

**risks-and-rollback**: key risks, stop conditions, rollback path.

**follow-on-slices**: next 2-4 slices, one line each.

**implementation-status** (only if code changes made): status (not started|in progress|partial|complete), applied|proposed only, targets changed, verification evidence, deviations.

## Clarification mode

Use only when the current mode lacks minimum evidence after exhausting available context.

Return: `mode`, `known-so-far`, `decision-blockers`, `questions-for-user`, `next-step`.

## Response format

End each response with: `completed-modes`, `current-mode`, `delivery-mode` (direct edit|ready-to-apply output|analysis only), `status`, key assumptions or blockers, `next-step`.
