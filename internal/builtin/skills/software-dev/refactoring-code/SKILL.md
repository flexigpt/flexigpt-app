---
name: refactoring-code
description: Selects and performs bounded, behavior-preserving refactors or legacy-containment slices that reduce complexity, duplication, coupling, and change risk without turning into a rewrite. Use when the user asks to refactor code, clean up a risky module, modernize a legacy area safely, or decide what technical debt slice to tackle next from workspace access, attached files, or pasted code.
---

# Refactoring Code

## Purpose

Choose the smallest safe refactor that materially improves code structure or change safety while preserving intended behavior, including caller-visible failure and control behavior where relevant.

This skill is for bounded cleanup and legacy containment, not for broad rewrites or generic modernization manifestos.

## Use when

- the user asks to refactor a file, module, or subsystem safely
- the user wants to reduce duplication, split a god class, simplify a complex workflow, or improve testability
- the user wants to modernize a legacy area incrementally without breaking behavior
- the user asks what technical-debt slice should be tackled first
- the user wants a safe cleanup plan before implementation
- the available code context comes from a workspace or repo, attached files, pasted code, or a mix of these

## Do not use when

- the user wants a new feature implemented from scratch
- the main question is system architecture direction
- the request is only style cleanup, formatter output, or tool-driven lint fixes
- the request is a broad rewrite, full-repo cleanup, or `optimize everything`
- public behavior is expected to change substantially and that change has not been agreed

## Context sources and delivery surface

This skill can work from any combination of:

- a workspace or repo accessible through tools
- attached files
- pasted code, snippets, or diffs in chat
- explicit bug reports, failing tests, or user descriptions of the pain point

Rules:

- start from the richest trustworthy context already available
- do not assume a writable workspace exists
- if a writable workspace is available and the refactor slice is clear, apply changes directly
- if direct editing is unavailable but the context is sufficient, produce ready-to-apply code or diffs in markdown and let the user apply them
- if tools exist but the workspace or repo location is unknown, ask for the repo root or relevant starting path instead of searching blindly
- if the available context is too thin to identify seams, preserved behavior, or likely blast radius safely, ask for the missing context before proceeding
- be explicit about what was seen directly versus what remains inferred or unverified
- do not claim a file or workspace was modified unless it was actually edited through available tools

## Decision protocol / interaction contract

1. Determine the mode:
   - `slice-selection`
   - `refactor-plan`
   - `refactor-apply`
2. Inspect the available code evidence before asking questions, whether it comes from a workspace, attachments, pasted code, or diffs.
3. Ask only for facts that materially affect preserved behavior, public contract, risk, or slice boundary.
4. Ask in one compact batch, usually 1-4 questions.
5. If the bounded slice and preserved behavior are already clear from the request and available context, do not force extra discovery before `refactor-apply`.
6. Proceed once the smallest safe slice is clear.
7. Prefer one bounded slice over a broad roadmap unless the user explicitly asks for a roadmap.
8. Preserve external behavior unless the user explicitly approves a change.
9. If the code is too risky to refactor directly, make the first slice a safety-net or seam-creation slice.
10. If a refactor is too large for a trustworthy pass, split it further.
11. Never recommend a rewrite when a smaller seam, adapter, extraction, or containment move would work.
12. Do not delete comments or debugging statements outside the chosen slice. Where the refactor changes behavior or structure inside the slice, update nearby comments or debugging to match the new logic rather than leaving them stale.

## Minimum evidence needed

`slice-selection`:

- the repo, module, file, snippet, or code area under discussion
- the pain point: bugs, duplication, complexity, coupling, upgrade friction, or legacy drag
- some evidence of where the pain concentrates

`refactor-plan`:

- the target file/module/slice or visible code surface
- the behavior that must be preserved
- enough code context to identify seams, dependencies, and risk

`refactor-apply`:

- a bounded slice
- preserved behavior or acceptable change clearly stated
- enough editable or referenceable context to make a trustworthy change:
  - writable workspace, or
  - attached file(s), or
  - pasted code with sufficient surrounding context
- a safety net:
  - existing tests, or
  - characterization tests, or
  - explicit manual verification steps
- if the current baseline already fails, the failing baseline must be recorded before editing so verification claims stay honest

If minimum evidence is missing, use clarification mode.

## Working definitions

- `behavior-preserving`: external behavior, contracts, and expected outputs remain the same for intended use
- `characterization test`: a test that captures current behavior before changing structure
- `seam`: a point where behavior can be isolated, wrapped, substituted, or verified safely
- `legacy containment`: isolating risky or outdated code behind a stable facade instead of rewriting everything
- `bounded slice`: a refactor small enough to review and verify safely in one pass

## Discovery question bank

Use the smallest relevant subset.

- What exact pain are we trying to reduce: bugs, complexity, coupling, duplication, test gaps, upgrade drag, or performance hotspots?
- What behavior must stay unchanged?
- Is there an existing failing area, repeated bug pattern, or change hotspot?
- Are tests present? If not, can we add characterization tests or define manual checks first?
- Is the target publicly consumed or internally scoped?
- Is the user asking for a plan only, or also for actual code changes?
- Is this auth/authz/session/validation or other security-sensitive code, and if so what control behavior must remain intact?
- Do we have writable workspace access, or should the output be returned as ready-to-apply code or diff?

## Refactor drivers

Use these to identify whether refactoring is justified.

- duplicated critical logic
- long or deeply nested functions obscuring invariants
- god classes or modules with mixed responsibilities
- hidden coupling causing ripple edits
- fragile branching based on booleans, flags, or mode strings
- side effects mixed into pure logic
- legacy dependency or API usage spread through many callers
- inability to test risky logic without large setup
- recurring bugs clustering in one area
- dependency upgrade blocked by missing abstraction

If none of these are present in a meaningful way, consider `leave alone`.

## Workflow

### 1. Lock the target and preserved behavior

State:

- exact slice in scope
- pain being reduced
- behavior that must remain unchanged
- security controls, failure behavior, and caller-visible error semantics that must remain unchanged when relevant
- what is out of scope

If preserved behavior is unclear and a change could affect users or callers, ask before proceeding.

### 2. Gather hotspot evidence

Use what is actually available from the visible context:

- recent bug locations
- repeated review findings
- files or code surfaces with high churn
- obvious complexity
- repeated logic
- known incidents
- difficult test setup
- dependency friction
- pasted diffs or failing examples

Do not fabricate ROI, saved hours, or fake debt scores.

### 3. Choose the smallest safe slice

Prefer a slice that is:

- high pain
- locally concentrated
- boundary-clear
- verifiable
- completable in 1-5 focused days
- usually 3-5 core files or one similarly bounded visible surface
- usually around 500 lines of change max

These are heuristics, not hard limits. If a slightly larger slice is still boundary-clear and verifiable, prefer that over an artificially split change that increases risk. If the problem is larger, pick the first enabling slice, not the whole campaign.

### 4. Establish the safety net first

If tests are missing for behavior that must be preserved:

- first slice = characterization tests, a verification harness, or explicit manual verification steps

Do not start structural refactoring first when behavior preservation cannot be checked.

If the baseline already has failing tests or known broken flows:

- record the current failing set before editing
- avoid claiming unrelated verification as new proof of safety
- prefer a smaller seam-creation or containment slice if the baseline is too unstable

### 5. Choose the strategy

Use the simplest strategy that matches the problem.

`simplify-in-place`:

- Use when:
  - complexity is local
  - boundaries are already acceptable
  - behavior can be preserved directly

- Examples:
  - split long function
  - replace nested branching with guard clauses
  - isolate pure calculations from side effects

`extract-function-or-module`:

- Use when:
  - duplication exists
  - a coherent responsibility can be isolated
  - callers can stay stable

- Examples:
  - shared validation logic
  - repeated mapping or parsing logic
  - reusable policy or rule evaluation

`split-by-responsibility`:

- Use when:
  - a class or module has clearly mixed responsibilities
  - internal seams are visible
  - public contract can stay stable during the split

- Examples:
  - separate persistence from business rules
  - separate orchestration from transformation
  - separate read formatting from write logic

`introduce-facade-or-adapter`:

- Use when:
  - a legacy dependency leaks through many callers
  - an upgrade is blocked by direct coupling
  - an external API or SDK should be contained

- Examples:
  - wrap legacy service client
  - isolate framework-specific code
  - create translator around unstable external contracts

`strangler-slice`:

- Use only when:
  - a subsystem boundary is clear
  - routing or delegation seam exists
  - coexistence between old and new is feasible
  - this is truly subsystem replacement, not local cleanup

- Do not use this as the default modernization answer.

`leave-alone`:

- Use when:
  - pain is low
  - churn is low
  - risk of refactor exceeds likely benefit
  - the real issue is elsewhere

### 6. Plan the slice

For the chosen slice, define:

- exact targets in scope
- sequence of changes
- verification step after each meaningful substep
- rollback path
- follow-on slices, if any

Use file paths when known. If only snippets or visible surfaces are available, name those precisely and mark unseen dependencies as assumptions or blockers.

### 7. Apply only if requested

If the user asked for actual code changes:

- make changes in small steps
- keep behavior stable
- re-check preserved behavior after each step
- stop if the slice grows beyond its original safety boundary
- record any deviation before continuing
- if a writable workspace is available, apply the refactor there
- if direct editing is unavailable but the context is sufficient, return ready-to-apply output with clear target labels
- if only partial context is available and a trustworthy refactor cannot be completed safely, stop and ask for the missing context

### 8. Edit execution

When `refactor-apply` is selected:

- read the full target file before editing it when the full file is accessible
- if only partial code is available, do not refactor beyond the visible safe boundary without asking for more context
- keep the slice bounded to the chosen targets
- use targeted replacement only for simple exact edits
- if targeted replacement fails, anchors drift, or the refactor touches multiple nearby regions, switch to whole-file rewriting instead of retrying brittle replacements
- batch independent reads and writes when tools allow
- verify the changed behavior or safety net immediately after editing
- if the output is only proposed in chat and not applied in a workspace, state that explicitly

## Decision rules

1. If tests are missing, the first slice is the safety net.
2. If the boundary is unclear, the first slice is seam creation or a facade, not deep internals.
3. If public behavior or interfaces would change, ask before proceeding.
4. If the slice requires many unrelated files, split it.
5. If the refactor would become a rewrite, stop and re-scope.
6. Prefer containment over replacement for risky legacy dependencies.
7. Prefer extraction over redesign when the pain is localized.
8. Prefer one completed slice over a large partial cleanup.
9. If the real issue is architectural, say so and route upward rather than forcing a local refactor answer.
10. If no meaningful gain is likely, recommend leaving it alone.
11. For security-sensitive or public-contract code, preserve control behavior and negative-path outcomes, not just happy-path outputs.
12. If only snippets are available and surrounding context could materially change the refactor, ask for more context before applying.

## Human-in-the-loop behavior

Ask questions when the answer would change:

- what behavior must be preserved
- whether a public contract may change
- whether breaking changes are allowed
- whether the user wants plan-only or actual refactor
- whether missing tests can be added first
- whether auth, session, tenant, validation, or other security-relevant behavior may change
- whether direct workspace edits are possible or the result should be returned as ready-to-apply output

Question limits:

- usually 1-4 questions
- ask in one batch
- if the user wants speed, ask only the top blockers and proceed with explicit assumptions where safe

Proceed without more questions when:

- scope is clear
- preserved behavior is clear
- strategy choice is clear
- verification path exists

Use clarification mode only when those are not true.

## Output structure

Use these sections in this order.

`refactor-context`:

- request goal
- `[observed]`
- `[inferred]`
- `missing-inputs`
- `mode`
- `delivery-mode`: `direct edit | ready-to-apply output | analysis only`
- `confidence`: `high | medium | low`

`problem-hotspots`:

- exact files, modules, or code surfaces considered
- why each is a candidate
- evidence of pain concentration

`chosen-slice`:

- exact scope
- why this slice first
- what is intentionally out of scope

`behavior-to-preserve`:

- public behavior
- internal invariants worth preserving
- caller or contract constraints

`strategy`:

- chosen strategy
- why it fits better than alternatives
- rejected strategies and why not now

`safety-net`:

- existing tests or checks
- characterization tests needed
- manual verification if automated checks are absent

`refactor-plan`: Use a small ordered list -

1. step
2. targets
3. purpose
4. verification after the step

`risks-and-rollback`:

- key risks
- stop conditions
- rollback path

`follow-on-slices`:

- next 2-4 slices only if useful
- one line each

`implementation-status`: Include only if actual refactoring was requested.

- `not started | in progress | partial | complete`
- `applied | proposed only`
- targets changed
- verification evidence gathered
- deviations from plan

## Clarification mode

Use this instead of the full output only when minimum evidence is missing.

Return:

- `mode`
- `known-so-far`
- `decision-blockers`
- `questions-for-user`
- `recommended-next-step`

## Fallback modes

### Scope too broad

If the user asks for broad cleanup across the whole repo:

- identify the top candidate slice
- explain why that slice should go first
- do not produce a rewrite manifesto

### No safety net

If there is no credible way to preserve behavior:

- do not recommend deep refactoring yet
- make the first slice test creation, seam creation, or verification harness setup

### Limited context

If only partial files, snippets, or pasted code are available:

- constrain the recommendation to the visible surface
- mark unseen callers, dependencies, and tests as assumptions or blockers
- do not claim full-slice safety if surrounding context was not available

### Architecture mismatch

If the underlying problem is architectural rather than local structure:

- say the refactor is constrained by unresolved architecture
- recommend finalizing architecture first
