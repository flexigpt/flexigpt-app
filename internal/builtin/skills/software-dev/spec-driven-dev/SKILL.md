---
name: spec-driven-dev
description: Discovers, scopes, implements, and verifies bounded software changes with artifacts as checkpoints. Use when delivering a feature, bug fix, enhancement, refactor, or investigation from workspace access, attached files, or pasted code that should be clarified into explicit requirements, implemented against them, and verified with recorded evidence.
---

# Spec-Driven Development

## Purpose

Deliver real software changes in a disciplined way.

Clarify enough to act, implement only within written scope, and verify with fresh evidence. Artifacts are checkpoints that improve decision quality across turns; they are not the end goal.

## Use when

- the user wants a feature, bug fix, enhancement, refactor, or investigation completed
- the user wants `here is the context, just make the change`
- the user wants `inspect first, then change`
- the user wants discovery only: where behavior lives, how a flow works, or which files or surfaces are involved
- the task benefits from explicit requirements, implementation tracking, or verification evidence
- the available context comes from a workspace or repo, attached files, pasted code, or a mix of these

## Do not use when

- the user only wants a quick explanation, brainstorm, or one-off answer with no durable working state
- the task is only generic documentation with no implementation or verification component
- the main task is architecture direction rather than software delivery
- the main task is only code review or audit with no implementation
- the task is trivial and the user explicitly does not want a structured workflow

For small changes, keep the workflow minimal and proportional. Do not add ceremony for its own sake.

## Context sources and delivery surface

This skill can work from any combination of:

- a workspace or repo accessible through tools
- attached files
- code pasted directly in chat
- explicit behavior descriptions, error messages, tests, or diffs provided by the user

Rules:

- start from the richest trustworthy context already available
- do not assume a writable workspace exists
- if a writable workspace is available and the target area is identified, apply changes directly
- if direct editing is unavailable but the context is sufficient, produce ready-to-apply code or diffs in markdown and let the user apply them
- if tools exist but the workspace or repo location is unknown, ask for the repo root or relevant starting path instead of searching blindly
- if the available context is too thin to locate the right edit surface or preserve behavior safely, ask for the missing context before proceeding
- be explicit about what was seen directly versus what remains inferred or unverified

## Phases

Determine the current phase from the request and current state:

- `Discover`
- `Write`
- `Implement`
- `Verify`

Typical starts:

- start at `Discover` when the relevant files, current behavior, or blast radius are not yet clear
- start at `Write` when the work needs explicit requirements before editing
- start at `Implement` when the requested behavior, scope, and likely edit surface are already clear enough to act safely
- start at `Verify` when the change already exists and the user wants confidence or evidence

If multiple phases are needed, run only the next necessary phases in order. Do not ask the user which phase to use unless the request is genuinely ambiguous. It is ok to complete multiple phases in one go if possible.

## Working artifacts

This skill always generates artifact checkpoints. Use the smallest artifact that fits the current phase:

- `discovery`: what was inspected, current behavior, likely edit points, blockers
- `spec`: scope, requirements, edge cases, intended change targets
- `progress`: implementation plan, task state, evidence, deviations
- `verification`: requirement-by-requirement evidence and verdict

Rules:

- keep artifacts in the conversation
- do not repeat or fork artifact sections unnecessarily across turns
- for small, clear changes, keep the artifact minimal and continue in the same turn

## Decision protocol

Follow this sequence.

1. Read this file fully before answering.
2. Determine the requested outcome and current phase.
3. Choose the smallest set of phases needed to complete the user's actual request.
4. Check whether minimum evidence exists for the needed phases.
5. Inspect the available context thoroughly before asking questions, whether it comes from a workspace, attachments, pasted code, or prior outputs.
6. Ask only blocker questions that would materially change scope, behavior, interfaces, or verification.
7. Ask in one compact batch, usually 1-3 questions, up to 5 only when the spec would otherwise be unusable.
8. If minimum evidence exists, proceed. Do not wait for perfect certainty.
9. If the request spans multiple phases, run them in sequence.
10. If the change is already clear enough to implement safely, do not force broad discovery first.
11. If discovery is needed before editing, do one focused discovery pass, then ask at most one compact confirmation only if the answer would materially change the edit.
12. Prefer one bounded implementation pass over many alternating discovery and edit loops.
13. Record non-blocking uncertainty under `Assumptions`, `Open Questions`, or `Deviations`.
14. Re-read the active artifact after external edits or after more than 5 tool actions since the last read.

## Shared operating rules

- read before writing. Reuse exact terms from the user and codebase.
- never invent file paths, commands, interfaces, requirements, or test results.
- start from the context already provided, whether workspace files, attachments, pasted code, diffs, logs, or explicit requirements.
- prefer inspecting available context over asking the user to restate it.
- if there is no attached context and no known workspace root or starting area, ask for either:
  - the relevant workspace or repo path, or
  - the relevant files, snippets, diffs, or error output
- if only partial context is available, keep claims proportional to what was actually seen and mark unseen dependencies or callers as assumptions or unverified.
- if work originates from review findings, convert those findings into explicit requirements and edge cases before editing; do not implement directly from finding text alone.
- when the user wants speed, ask only the top blockers and proceed provisionally when safe.
- if the task touches auth, authz, session, public API, template, callback, file-handling, or multi-tenant behavior, capture denied or misuse cases explicitly.
- always batch independent reads, checks, and edits when the tools allow.
- if direct editing is not possible, produce copy-paste-ready outputs with clear target labels and application notes.
- do not claim a file or workspace was modified unless the tool-accessible workspace was actually edited.
- Do not delete any code comments/debugging statements from the part of the code which is not under consideration. Where ever the source code has changes, update the comments/debugging as per new logic, never delete them.
- user instructions override this skill.

## Minimum evidence by phase

`Discover`:

- the investigation goal or question
- at least one anchor:
  - attached context
  - pasted code or diff
  - file path
  - symbol
  - module
  - feature name
  - error or symptom
  - interface or behavior area

`Write`:

- the goal is understood
- the affected area is known
- evidence exists in a workspace, attachments, pasted code, or other supplied context, or explicit assumptions can be recorded safely

`Implement`:

- the full spec is available, or the request and inspected context already define a safe bounded change
- the implementation surface is identified:
  - a writable workspace or repo, or
  - attached file(s), or
  - pasted code or diff
- if direct editing is unavailable, enough surrounding context exists to produce trustworthy ready-to-apply output
- no unresolved question remains that would change scope, public behavior, or required interfaces

`Verify`:

- the full spec or explicit expected behavior is available
- the implementation is accessible in a workspace, attachments, pasted code, or generated output from the current conversation
- there is at least one fresh evidence path for each requirement or edge case

If minimum evidence is missing, use clarification mode.

## Clarification mode

Use this only when minimum evidence for the current phase is missing.

Return:

- `phase`
- `known-so-far`
- `decision-blockers`
- `questions-for-user`
- `recommended-next-step`

Keep questions short, prioritized, and limited to true blockers.

## Discover

Goal: understand the relevant code area enough to answer the question or prepare a bounded edit.

Workflow:

1. inspect the provided context or the smallest useful accessible workspace area
2. identify current behavior, entry points, dependencies, and likely ownership points
3. map the likely edit surface or decision surface
4. stop once the next step is clear:
   - `implement now`
   - `confirm one decision`
   - `need clarification`
   - `discovery complete`

Rules:

- do not turn discovery into a full repo or workspace tour
- ask questions only when available context cannot answer a blocker
- when the user wants discovery only, do not drift into implementation
- if only partial files or snippets are available, do not overclaim caller, callee, or test coverage beyond what was seen

`discovery` artifact format:

- `request-goal`
- `[observed]`
- `[inferred]`
- `sources-read`
- `candidate-targets`
- `current-behavior`
- `missing-inputs`
- `recommended-next-step`
- `confidence`: `high | medium | low`

## Write

Goal: produce an implementable spec that bounds the work.

Workflow:

1. read the request and provided context
2. inspect the relevant workspace area, attached files, or pasted code to answer what you can yourself
3. generate the `spec` artifact as soon as the work scope is clear enough to name
4. fill known sections immediately
5. put uncertainty in `Assumptions` or `Open Questions`, not in `Requirements`
6. ask only the remaining blocker questions
7. stop clarifying once requirements are concrete enough to implement safely
8. re-read the full spec before using it for `Implement` or `Verify`

Rules:

- do not ask for facts already available from the workspace or provided context
- requirements must be concrete, testable, and stable
- use requirement IDs: `[ ] R1. ...`, `[ ] R2. ...`
- use edge-case IDs: `E1`, `E2`, ...
- if the change touches public or security-sensitive behavior, include at least one denied or misuse case where relevant
- use real file paths or target names from inspection
- if a path or target is not confirmed, mark it `(unverified)` and record that under `Assumptions`
- if multiple materially different approaches exist, note the options briefly, recommend one, and ask for confirmation before implementation
- `Status` remains `provisional` until the user confirms
- if the user explicitly asks to continue without confirmation, implementation may proceed provisionally
- for narrow, clear changes, write a minimal but complete spec and continue in the same turn if implementation is also requested

`spec` artifact format:

- `Title`: `Spec: [Name]`
- `Status`: `provisional | confirmed`
- `Goal`
- `Context`
- `Requirements`
- `Out of Scope`
- `Technical Approach`
- `Interfaces`
- `Data Models`
- `Change Targets`: table `Target | Action | Description`
- `Edge Cases`: table `ID | Scenario | Handling`
- `Decisions`
- `Assumptions`
- `Open Questions`

## Implement

Goal: turn the spec or already-clear request into working changes in an accessible workspace, attached files, or ready-to-apply output.

Workflow:

1. read the full spec if one exists
2. if no usable spec exists but the change is already clear, generate a minimal `spec` artifact first and continue
3. if a spec is `provisional`, ask for confirmation unless the user explicitly authorized provisional implementation or the remaining uncertainty is non-blocking
4. read each target to be changed fully when accessible, plus direct dependencies and dependents needed to avoid blind edits
5. if only snippets or partial files are available, inspect enough surrounding context to make the change safely; if that is not possible, ask for the missing context before editing
6. generate or update the `progress` artifact with a plan mapped to requirement IDs before editing
7. execute the next task batch in dependency order:
   - contracts
   - shared utilities
   - implementations
   - callers
   - tests and docs
8. after the batch, verify narrowly and update the `progress` artifact once
9. continue until `complete`, `partial`, or `blocked`

Rules:

- do not change the spec silently during implementation
- do no work beyond the spec or clearly stated request
- if a writable workspace exists and the target area is identified, apply changes directly
- if direct editing is unavailable but the context is sufficient, return ready-to-apply output as fenced code blocks or diffs labeled by target
- do not claim a target was edited when only a proposed change was produced in the conversation
- when only partial context is available, keep scope bounded to that visible surface and mark unseen dependencies, callers, or tests as assumptions or blockers
- prefer targeted edits over rewrites, but do not get stuck on brittle edit mechanics
- use targeted replacement only for simple exact substitutions
- if targeted replacement fails, anchors drift, or the change spans multiple nearby regions, switch to whole-file rewriting instead of retrying brittle replacements repeatedly
- keep the edit set bounded
- every task must trace to one or more `R#` IDs when a spec exists
- if implementation reveals a scope, behavior, or interface change, stop and update the spec before continuing
- if a likely follow-up is useful but outside scope, record it as a follow-up instead of silently expanding the change
- ask before proceeding on anything that changes scope, public behavior, or required interfaces
- if tests, builds, or runtime checks are unavailable, record `code-reviewed only`
- if the change was produced only as output and not applied in a workspace, record that explicitly
- 3-strike rule:
  - try a direct fix
  - then a materially different method or hypothesis
  - then broaden the investigation
    After 3 failed approaches, mark the task `blocked`, record what was tried, and ask the user.
- in a discovery-then-edit flow, prefer one bounded implementation pass once the edit surface is clear

`progress` artifact format:

- `Title`: `Progress: [Name]`
- `Status`: `in-progress | partial | blocked | complete`
- `Assumptions`
- `Plan`: table `# | Objective | Targets | Depends on | Req | Verify by`
- `Task Log`: table `# | Status | Evidence | Notes`
- `Deviations`

Plan rules:

- create the plan before editing code or drafting final patch output
- start all plan rows as `pending`
- keep task numbering stable once work begins

## Verify

Goal: determine whether the implementation satisfies the spec or explicitly stated expected behavior.

Workflow:

1. read the full spec and `progress` artifact if present
2. generate or update the `verification` artifact
3. for every requirement and edge case, gather fresh evidence and record the result
4. compare implementation and evidence to the exact requirement text, not intent alone
5. finish with a verdict, gaps, and follow-ups

Rules:

- fresh evidence means evidence gathered during the current verification pass: code inspection, attached-file inspection, pasted-code inspection, test output, build or lint output, command output, or an explicit comparison between code and requirement text
- if the implementation exists only as generated output in the conversation, verify against that output and note reduced confidence
- never claim `done`, `fixed`, `complete`, `passing`, or `ready` without fresh evidence
- if evidence is incomplete, the verdict must stay `not verified` or `partially verified`
- if runtime checks are unavailable, say `code review only, not runtime-verified`
- if the change was not actually applied in a workspace, say `proposed change only, not applied`
- if a requirement is ambiguous, record `not verified: ambiguous requirement`
- if work is missing, record `not verified: not implemented`
- for public or security-sensitive behavior, verify both allowed and denied or invalid behavior when relevant
- if the spec is provisional, verify what exists and note reduced confidence under `Gaps`
- 3-strike rule applies here too. After 3 materially different failed verification attempts, mark `Status: blocked`, record what was tried, and ask the user

`verification` artifact format:

- `Title`: `Verification: [Name]`
- `Status`: `in-progress | blocked | complete`
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

## Response summary

After each phase, give a short summary with:

- `current-phase`
- `artifact`
- `delivery-mode`: `direct edit | ready-to-apply output | analysis only`
- `status` or `verdict`
- key assumptions, deviations, or blockers
- next action for the user, if any

If one response covers multiple phases, summarize the handoff clearly and keep each artifact consolidated.
