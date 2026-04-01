---
name: spec-driven-dev
description: "Spec-first development workflow. Use when a request should be clarified into a concrete spec, implemented from that spec, and verified against explicit requirements with persistent artifacts."
---

# Spec-Driven Dev

Define before building, build from the definition, verify against it.

| Situation                                  | Step                       | Artifact        | Purpose                  |
| ------------------------------------------ | -------------------------- | --------------- | ------------------------ |
| Unclear request, new feature, scope choice | Write                      | spec.md         | Request → concrete spec  |
| Spec exists, ready to build                | Implement                  | progress.md     | Plan and build from spec |
| Need proof work is correct                 | Verify                     | verification.md | Check work against spec  |
| Bug or investigation                       | Write → Implement → Verify | -               | Full cycle               |

If unclear which step: ask `Do you want me to Write the spec, Implement from it, or Verify it?`

## Shared principles

- Read before writing. Use exact terms from the user and codebase. Do not invent file paths, commands, or requirements.
- If the starting folder or code repository root is not known, ask explicitly for it rather than doing a deep search.
- Artifacts are durable state; chat context is not. Default artifact path: `specs/[name].md` unless user provides one.
  - All artifact sections present from the start; use `N/A` when not applicable. Edit in place; never append duplicates.
  - If an artifact was last read more than 5 tool calls ago, re-read before the next decision.
- Use parallel tool calls for independent reads, checks, and edits within a batch.
- If you don't get tools to read and write files/text, ask whether you should give the checkpoints in output or can user attach these tools.
- Use questioning gates when things are stuck or unclear.
- User instructions override this skill.

## Step 1 - Write

Active artifact: `spec.md`. Turn a request into an explicit spec.

### Phase 1 - Clarify

Goal: get enough to write a useful spec without asking avoidable questions.

1. Read the request and all provided context.
2. Inspect repo and nearby code to answer what you can.
3. Create a provisional spec file as soon as feature name and path are clear. Fill known sections; put uncertainty in `Assumptions` or `Open Questions`.
4. Ask only for blockers remaining after inspection. If clear enough, continue to Phase 2 in the same response.

Rules:

- Do not ask about facts determinable from repo or context.
- At most 5 questions, ordered by blocking impact. If the user wants speed, ask top 1-2 and proceed provisionally.
- Even when blocked, write the current spec state before replying.

### Phase 2 - Write

1. Re-read the spec draft and any blocker answers.
2. Inspect related code for real paths, naming, interfaces, and patterns.
3. Complete all sections using the format below. Re-read the full spec before presenting.

Rules:

- Every requirement: numbered checkbox, directly testable.
- Uncertain items → `Assumptions` or `Open Questions`, not `Requirements`.
- File paths must come from inspection; mark unverified paths `(unverified)` and add to `Assumptions`.
- Non-obvious design choices: list options briefly, recommend one, ask for confirmation before proceeding.
- Require explicit user confirmation before the spec moves from `provisional` to `confirmed`.

Spec format (all sections required; `N/A` if not applicable):

- **Title**: `Spec: [Name]`
- **Status**: `provisional` | `confirmed`
- **Goal**: what and why, 2–3 sentences
- **Context**: current state, patterns, constraints
- **Requirements**: numbered checkbox list; each concrete and testable
- **Out of Scope**
- **Technical Approach**: chosen approach and rationale
- **Interfaces**: APIs, signatures, contracts, types
- **Data Models**: schemas, config shapes, types
- **File Changes**: table - `File | Action | Description`; paths real or marked `(unverified)`
- **Edge Cases**: table - `Scenario | Handling`
- **Decisions**: key choices with rationale
- **Assumptions**: assumed facts; flag shaky ones
- **Open Questions**

Escape hatches:

- No repo access → work from provided context, mark assumptions.
- Ambiguous requirement → ask, or record interpretation as assumption.
- Open questions remain → proceed provisionally, mark affected requirements.

Output: every response writes the spec artifact exactly once. Final status is `provisional` until user confirms.

## Step 2 - Implement

Active artifact: `progress.md`. Plan and build from an existing spec.

### Phase 1 - Plan

Prerequisites: spec file exists and is fully read. Unresolved `Open Questions`: ask to resolve or waive. Paths in `File Changes` not matching codebase: record the mismatch.

1. Read the full spec.
2. Read each file in `File Changes` plus direct imports and dependents.
3. Order work by dependency: contracts → implementations, shared utilities → consumers, inner → outer.
4. Break work into concrete tasks mapped to spec requirement numbers.
5. Create `progress.md`.

`progress.md` format:

- **Title**: `Progress: [Name]`
- **Status**: `in-progress` | `blocked` | `partial` | `complete`
- **Spec**: path to spec file
- **Assumptions**
- **Plan**: table - `# | Objective | Files | Depends on | Req | Verify by`
- **Task Log**: table - `# | Status | Evidence | Notes`
- **Deviations**

Rules: plan only in this phase; every task traces to a spec requirement; all rows start `pending`.

### Phase 2 - Execute

1. Re-read needed parts of spec and `progress.md`.
2. Complete the next task batch; verify narrowly: test, build, lint, or code review.
3. Write `progress.md` once to record the batch.
4. Continue until complete or blocked.

Rules:

- No work beyond the spec. Do not modify the spec during implementation.
- Do not skip tasks silently.
- Prefer editing files over full rewrites.
- Scope-changing assumptions require user confirmation before proceeding.

3-strike rule: direct fix → change method/input/hypothesis → broaden investigation. After 3 materially different failures: mark `blocked`, record what was tried, ask the user.

Escape hatches:

- Cannot run tests/build → record `code-reviewed only`.
- Spec gap → record under `Deviations`; ask if material.
- Missing dependency → mark task `blocked`, continue unblocked tasks.
- Task larger than planned → split into sub-tasks in Plan, continue.

Output: every response writes `progress.md` exactly once. Final status: `complete`, `partial`, or `blocked`.

## Step 3 - Verify

Active artifact: `verification.md`. Check implementation against the spec.

### Phase 1 - Verify

Prerequisites: spec file exists; implementation exists or is claimed to.

1. Read the full spec and `progress.md` if present.
2. Create `verification.md`.
3. For each requirement and edge case: gather fresh evidence (read code, run tests, run build/lint, compare to requirement text), then record the result.
4. Write verdict, gaps, and follow-ups.

`verification.md` format:

- **Title**: `Verification: [Name]`
- **Status**: `in-progress` → `complete` when done
- **Spec**: path
- **Progress**: path or `N/A`
- **Results**: table - `# | Requirement | Status | Evidence`
- **Edge Case Results**: same table format
- **Verdict**: `verified` | `partially verified` | `not verified` | `failed`
- **Gaps**
- **Follow-ups**

Hard rule: do not claim done, fixed, complete, passing, or ready unless fresh evidence is recorded. Incomplete evidence → verdict stays `not verified` or `partially verified`.

3-strike rule applies here too: after 3 materially different failed verification attempts, mark `blocked`, record what was tried, ask the user.

Escape hatches:

- Cannot run tests/runtime → mark evidence `code review only, not runtime-verified`.
- Ambiguous requirement → mark `not verified: ambiguous requirement`, add to Gaps.
- Partial implementation → mark missing items `not verified: not implemented`.
- No `progress.md` → verify from spec and code, note missing context in Gaps.
- Provisional spec → verify what exists, note limited confidence in Gaps.

Output: every response writes `verification.md` exactly once. Final status: `complete` with verdict.
