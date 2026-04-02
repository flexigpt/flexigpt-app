---
name: spec-driven-dev
description: Plans, implements, and verifies software changes through a spec-driven workflow with durable artifacts. Use when delivering a feature, bug fix, enhancement, refactor, or investigation that should be clarified into explicit requirements, implemented against them, and verified with recorded evidence.
---

# Spec-Driven Development

## Purpose

Deliver real software changes in a disciplined way.

Clarify enough to act, implement only within written scope, and verify with fresh evidence. Artifacts are checkpoints and durable working state that improve decision quality across turns; they are not the end goal.

## Use when

- the user wants a feature, bug fix, enhancement, refactor, or investigation completed
- the work should be clarified into explicit requirements before or during implementation
- the change should be implemented against written scope rather than ad hoc chat memory
- the user wants checkpointed progress and explicit verification
- the task spans planning, coding, and validation

## Do not use when

- the user only wants a quick explanation, brainstorm, or one-off answer with no durable work product
- the task is only generic documentation with no implementation or verification component
- the task is trivial and the user explicitly does not want a spec/checkpointed workflow

For small changes, keep the workflow minimal and proportional. Do not add ceremony for its own sake.

## Decision protocol

Follow this sequence.

1. Read this file fully before answering.
2. Determine the requested outcome and active mode:
   - `Write`
   - `Implement`
   - `Verify`
3. Choose the smallest set of modes needed to complete the user's actual request.
4. Check whether minimum evidence exists for the active mode.
5. Inspect the repo and provided context before asking questions.
6. Ask only for blocker facts that would materially change scope, behavior, interfaces, or verification.
7. Ask in one compact batch, usually 1-3 questions.
8. Once minimum evidence exists, proceed. Do not wait for perfect certainty.
9. If the request spans multiple modes, run them in order: `Write -> Implement -> Verify`.
10. After each mode, update the active artifact once and give a brief status summary.

Do not ask `Should I write the spec, implement from it, or verify it?` unless routing is genuinely unclear after reading the request and context.

## Artifacts

Default artifact directory: `specs/[slug]/`

Use user-provided paths when given. Otherwise keep all work for the same change together:

- `specs/[slug]/spec.md`
  - requirements, scope, decisions, and intended file changes
- `specs/[slug]/progress.md`
  - implementation plan, task state, evidence, and deviations
- `specs/[slug]/verification.md`
  - requirement-by-requirement verification evidence and verdict

Create the active artifact as soon as the work name and path are clear.

## Shared operating rules

- Read before writing. Reuse exact terms from the user and codebase.
- Never invent file paths, commands, interfaces, requirements, or test results.
- If the repo root or starting folder is unknown, ask for it instead of doing a deep search.
- Keep all required artifact sections present from the start. Use `N/A` when not applicable.
- Edit artifacts in place. Never append duplicate sections.
- If work originates from review findings, convert those findings into explicit requirements and edge cases before editing; do not implement directly from finding text alone.
- For auth, authz, session, public API, template, callback, file-handling, or multi-tenant changes, capture misuse or denied-path behavior explicitly in `Requirements` or `Edge Cases`, not only happy-path behavior.
- Re-read the active artifact before a new decision if it may be stale:
  - after external edits
  - after more than 5 tool actions since the last read
- Batch independent reads, checks, and edits when tools allow.
- If repo or file tools are unavailable, work from provided context, mark unverified facts as assumptions, and still produce the artifact once in chat.
- User instructions override this skill.

## Human-in-the-loop rules

Ask only when the answer cannot be determined from the request or repo inspection and would materially affect:

- scope
- external behavior
- required interfaces
- risk
- verification outcome

Question limits:

- normally ask up to 3 questions in one turn
- ask up to 5 only in `Write` when the spec would otherwise be unusable
- if the user wants speed, ask the top 1-2 blockers and proceed provisionally when safe

Minimum evidence to proceed:

- `Write`
  - the goal is understood
  - the affected area is known
  - and either repo evidence exists or explicit assumptions are recorded
- `Implement`
  - the full spec has been read
  - the repo or workspace is located
  - no unresolved question remains that would change scope, public behavior, or required interfaces
- `Verify`
  - the full spec has been read
  - the implementation is accessible
  - there is at least one fresh evidence path for each requirement or edge case

If minimum evidence is missing, use clarification mode instead of forcing progress.

If uncertainty is non-blocking, proceed provisionally and record it in `Assumptions` or `Deviations`.

Fresh evidence means evidence gathered during the current verification pass: code inspection, test output, build or lint output, command output, or an explicit comparison between code and requirement text.

## Clarification mode

Use this when minimum evidence for the active mode is missing.

Return:

- `mode`
- `known-so-far`
- `decision-blockers`
- `questions-for-user`
- `recommended-next-step`

Keep `questions-for-user` short, prioritized, and limited to true blockers.

## Write

Active artifact: `spec.md`

Goal: produce an implementable spec that bounds the work. The purpose is to enable good implementation and verification, not to write exhaustive documentation.

Workflow:

1. Read the request and all provided context.
2. Inspect the relevant repo area or files to answer what you can yourself.
3. Create or update `spec.md` as soon as the work name and path are clear.
4. Fill known sections immediately.
5. Put uncertainty in `Assumptions` or `Open Questions`, not in `Requirements`.
6. Ask only the remaining blocker questions.
7. Stop clarifying once requirements are concrete enough to implement safely.
8. Re-read the full spec before presenting it or using it for the next mode.

Rules:

- Do not ask for facts already available from the repo or provided context.
- Requirements must be concrete, testable, and stable.
- Use requirement IDs: `[ ] R1. ...`, `[ ] R2. ...`
- Use edge-case IDs: `E1`, `E2`, ...
- If the change touches public or security-sensitive behavior, include at least one negative or misuse case where relevant, such as denied access, invalid input, stale token/session, missing ownership, unsafe callback input, or bad file metadata.
- Use real file paths from inspection.
- If a path is not confirmed, mark it `(unverified)` and record that under `Assumptions`.
- If multiple materially different approaches exist, note the options briefly, recommend one, and ask for confirmation before implementation.
- `Status` remains `provisional` until the user confirms.
- If the user explicitly asks to continue without confirmation, implementation may proceed provisionally, but the spec stays `provisional`.
- For narrow, clear changes, write a minimal but complete spec and continue in the same turn if the user has asked you to proceed.

`spec.md` format:

- `Title`: `Spec: [Name]`
- `Status`: `provisional | confirmed`
- `Goal`: 2-3 sentences
- `Context`
- `Requirements`
- `Out of Scope`
- `Technical Approach`
- `Interfaces`
- `Data Models`
- `File Changes`: table `File | Action | Description`
- `Edge Cases`: table `ID | Scenario | Handling`
- `Decisions`
- `Assumptions`
- `Open Questions`

Output discipline:

- Write or update `spec.md` once, even when blocked.
- In chat, briefly report:
  - status
  - main assumptions or blockers
  - whether implementation can start
  - what user decision is needed, if any

## Implement

Active artifact: `progress.md`

Goal: turn the spec into working changes in code or files.

Workflow:

1. Read the full spec.
2. If no usable spec exists, switch to `Write`.
3. If the spec is `provisional`, ask the user to confirm it or explicitly authorize provisional implementation.
4. Read each file listed in `File Changes`, plus direct dependencies and dependents needed to avoid blind edits.
5. Create or update `progress.md` with a plan mapped to requirement IDs before editing.
6. Execute the next task batch in dependency order:
   - contracts
   - shared utilities
   - implementations
   - callers
   - tests and docs
7. After the batch, verify narrowly and update `progress.md` once.
8. Continue until `complete`, `partial`, or `blocked`.

Rules:

- Do not change the spec during implementation.
- Do no work beyond the spec.
- Do not keep polishing the plan once it is good enough to execute.
- Prefer targeted edits over rewrites.
- Every task must trace to one or more `R#` IDs.
- If implementation reveals that a reviewed risk or misuse case needs a different requirement or interface decision, stop and update the spec first instead of silently drifting scope.
- If a path in `File Changes` does not match the codebase, record the mismatch under `Deviations` before editing.
- If you discover useful work outside the spec, record it as a follow-up instead of silently expanding scope.
- Ask for confirmation before proceeding on anything that changes scope, public behavior, or required interfaces.
- Do not skip tasks silently. If a task splits or order changes, update `Plan`.
- If a dependency or prerequisite is missing, mark that task `blocked` and continue unblocked tasks.
- If tests, builds, or runtime checks are unavailable, record `code-reviewed only`.
- 3-strike rule:
  - try a direct fix
  - then a materially different method or hypothesis
  - then broaden the investigation
    After 3 failed approaches, mark the task `blocked`, record what was tried, and ask the user.

`progress.md` format:

- `Title`: `Progress: [Name]`
- `Status`: `in-progress | partial | blocked | complete`
- `Spec`: path to `spec.md`
- `Assumptions`
- `Plan`: table `# | Objective | Files | Depends on | Req | Verify by`
- `Task Log`: table `# | Status | Evidence | Notes`
- `Deviations`

Plan rules:

- Create the plan before editing code.
- Start all plan rows as `pending`.
- Keep task numbering stable once work begins.

Output discipline:

- Write or update `progress.md` once per response, even when blocked.
- In chat, briefly report:
  - what changed
  - what evidence was gathered
  - whether the change is complete, partial, or blocked
  - what confirmation or next step is needed, if any

## Verify

Active artifact: `verification.md`

Goal: determine whether the implementation actually satisfies the spec.

Workflow:

1. Read the full spec and `progress.md` if present.
2. Create or update `verification.md`.
3. For every requirement and edge case, gather fresh evidence and record the result.
4. Compare implementation and evidence to the exact requirement text, not to intent alone.
5. Finish with a verdict, gaps, and follow-ups.

Rules:

- Never claim `done`, `fixed`, `complete`, `passing`, or `ready` without fresh evidence recorded in the artifact.
- If evidence is incomplete, the verdict must stay `not verified` or `partially verified`.
- If runtime checks are unavailable, say `code review only, not runtime-verified`.
- If a requirement is ambiguous, record `not verified: ambiguous requirement`.
- If work is missing, record `not verified: not implemented`.
- For public or security-sensitive behavior, verify both allowed behavior and denied/invalid/misuse behavior when relevant.
- If `progress.md` is absent, verify from the spec and code and note the missing implementation record under `Gaps`.
- If the spec is provisional, verify what exists and note reduced confidence under `Gaps`.
- 3-strike rule applies here too. After 3 materially different failed verification attempts, mark `Status: blocked`, record what was tried, and ask the user.

`verification.md` format:

- `Title`: `Verification: [Name]`
- `Status`: `in-progress | blocked | complete`
- `Spec`: path to `spec.md`
- `Progress`: path to `progress.md` or `N/A`
- `Results`: table `Req | Requirement | Status | Evidence`
- `Edge Case Results`: table `Edge | Scenario | Status | Evidence`
- `Verdict`: `verified | partially verified | not verified | failed`
- `Gaps`
- `Follow-ups`

Use row statuses:

- `pass`
- `partial`
- `fail`
- `not verified`

Output discipline:

- Write or update `verification.md` once, even when blocked.
- In chat, briefly report:
  - verdict
  - strongest evidence
  - remaining gaps
  - required follow-up

## Response summary

After writing any artifact, give a short chat summary with:

- current mode
- artifact path
- status or verdict
- key assumptions, deviations, or blockers
- next action for the user, if any

If one response covers multiple modes, write each active artifact once and clearly summarize the handoff between modes.
