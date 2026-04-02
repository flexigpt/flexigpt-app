---
name: reviewing-code
description: Reviews code, diffs, pull requests, modules, and API/config surfaces for bugs, security flaws, reliability issues, performance/scalability risks, maintainability hazards, and misuse-enabling designs. Use when the user asks to review code, find bugs, audit merge safety, verify a fix, or assess whether an interface is safe to use.
---

# Reviewing Code

## Purpose

Review actual code and produce evidence-based findings. The goal is not to give generic best practices, style feedback, or checklist theater. The goal is to identify real defects, concrete risks, and misuse-enabling designs, then rank them so the most important findings get full treatment.

## Use when

- Reviewing a PR, branch, commit range, or diff
- Reviewing a file, module, snippet, or focused repo area for bugs or production risk
- Assessing whether a change is safe to merge
- Reviewing API signatures, config schemas, defaults, or security-relevant interfaces for footguns
- Verifying whether a fix actually resolves a prior finding
- Auditing prototype or AI-generated code for hidden fragility before wider use

## Do not use when

- The main task is architecture design, boundary selection, or system-structure choice.
- The main task is feature delivery or checkpointed implementation rather than review.
- The user wants a broad rewrite, modernization manifesto, or autonomous cleanup.
- The user wants only lint/style/tool output.
- The user wants runtime penetration testing or exploit development.

## Decision protocol / interaction contract

1. Determine the review mode:
   - `change-review`
   - `code-slice-review`
   - `interface-review`
   - `fix-verification`
2. Check whether minimum evidence exists for that mode.
3. Inspect the code or repo context before asking questions.
4. Ask only for facts that would materially change scope, severity, or verdict.
5. Ask in one compact batch, usually 1-5 questions.
6. If minimum evidence exists, proceed without waiting for perfect certainty.
7. Separate `[observed]`, `[inferred]`, and `unverified`.
8. Separate control gaps from evidence gaps. If a control cannot be verified from the available code, config, tests, or docs, mark it `unverified` rather than assuming it exists or is missing.
9. Never report style-only nits.
10. Maintainability findings must tie to concrete risk such as bug-proneness, hidden coupling, incomplete-fix risk, misleading contracts, or missing safety net on risky code.
11. Never say `looks good`, `LGTM`, or `safe to merge` without showing scope, checklist coverage, and any unverified areas.
12. If no significant findings are found, say so explicitly and still show scope and coverage.
13. Do not make code changes unless the user also asks for implementation.

## Minimum evidence needed

`change-review`:

- a diff, PR, commit range, or changed file list
- enough surrounding context to understand changed behavior

`code-slice-review`:

- the snippet, file, module, or explicit repo slice
- enough local context to understand entry points, dependencies, or callers on risky paths

`interface-review`:

- a signature, schema, config surface, or interface definition
- defaults or examples if available

`fix-verification`:

- the original finding, bug description, or claimed issue
- the change that is supposed to fix it

If minimum evidence is missing, use clarification mode.

## Quality attributes

Every in-scope file or surface must be reviewed against all applicable attributes.

Use these labels only:

- `clean`
- `finding`
- `unverified`
- `n/a`

Attributes:

- `correctness`: normal-path and boundary-path behavior
- `security`: trust boundaries, input handling, auth, authz or tenant enforcement, secrets, unsafe callbacks or redirects, and abuse paths
- `reliability`: failure handling, retries, nulls, idempotency, partial failure
- `availability`: timeout behavior, fallback behavior, dependency failure impact
- `performance`: hot-path efficiency, excessive work, expensive operations
- `scalability`: boundedness under growth, fan-out, batching, N+1, resource growth
- `maintainability`: duplication, hidden coupling, confusing contracts, fragile structure
- `testability`: presence of good seams and meaningful verification on risky behavior
- `operability`: logs, metrics, tracing, diagnosability, rollback visibility
- `compatibility`: callers, migrations, defaults, schema/config/version safety
- `ergonomics`: API/config usability and footguns; applies mainly to interfaces and public surfaces

## Working definitions

- `confirmed`: supported directly by code evidence in scope
- `suspected`: likely issue, but needs missing context, runtime evidence, or framework behavior to confirm
- `unverified`: could not be assessed confidently from available evidence
- `interface footgun`: the easy path or default path leads to insecure or failure-prone use
- `change-risk`: the chance this change introduces bugs, regressions, or incomplete fixes outside the edited lines
- `control gap`: a protective behavior is missing, ineffective, or clearly bypassable
- `evidence gap`: the available material is too thin to confirm whether a control exists or works

## Discovery question bank

Use the smallest relevant subset.

- What exact scope should be reviewed: diff, file, module, API surface, or fix?
- For change review, what is the base branch or commit range?
- Is the main goal merge safety, bug finding, security review, API/config review, or fix verification?
- Are there known incidents, failing tests, or prior findings related to this area?
- Are there important runtime assumptions: concurrency, scale, public exposure, sensitive data, or external dependencies?
- For interface review, what inputs are caller-controlled and what defaults are security-relevant?

## Workflow

### 1. Lock scope

List:

- files or surfaces in scope
- review mode
- whether full file context was read or only partial snippets were available
- any tests, configs, callers, or docs that must also be checked

If scope is too large for trustworthy full coverage, narrow it or switch to a clearly labeled provisional hotspot review.

### 2. Read surrounding context

Do not review only changed lines or isolated functions. Read enough to understand:

- the containing function, class, or module
- direct callers and callees on risky paths
- data models and invariants
- relevant tests
- relevant config, schemas, defaults, and examples

### 3. Build the review sheet

For every in-scope file or surface, create a review row and mark every applicable quality attribute as:

- `clean`
- `finding`
- `unverified`
- `n/a`

No review is complete until each in-scope file or surface has a row.

### 4. Apply the mandatory checklist

#### A. Correctness

Check for:

- wrong logic, wrong condition, wrong field, wrong variable, wrong caller assumptions
- off-by-one and boundary mistakes
- empty, null, single-item, max-size, and invalid-state handling
- changed signatures with stale callers
- incomplete fixes that address symptom but not root cause

#### B. Security

Check for:

- injection: SQL, command, template, header, path
- XSS or unsafe output handling
- authn/authz and IDOR gaps
- broken tenant binding or missing ownership enforcement
- CSRF on state-changing web actions where relevant
- secrets in code or logs
- weak crypto or unsafe comparison
- unsafe callback, redirect, webhook, or URL-fetch handling
- unsafe deserialization
- information exposure through logs, errors, or debug paths
- denial-of-service via expensive inputs or unbounded work

#### C. Reliability and availability

Check for:

- swallowed errors or misleading success
- missing timeouts
- retries without idempotency
- partial failure without rollback or compensation
- resource leaks
- dependency failure causing avoidable outage
- missing validation of external or deserialized data

#### D. Performance and scalability

Check for:

- N+1 queries or repeated remote calls in loops
- full scans or unbounded queries where limits are expected
- large in-memory buffering where streaming is safer
- blocking work on latency-sensitive paths
- unnecessary recomputation or repeated serialization
- concurrency hazards, race windows, or lock contention
- growth in CPU, memory, or network cost that is not bounded by design

#### E. Maintainability and testability

Report only when tied to real risk. Check for:

- duplicated critical logic likely to drift
- dead or hallucinated code implying behavior that does not exist
- hidden coupling that makes partial fixes likely
- oversized functions or modules obscuring invariants
- weak seams or missing tests on risky behavior
- misleading naming or contracts likely to cause misuse

#### F. Compatibility and operability

Check for:

- changed interfaces with stale consumers
- unsafe defaults or migration behavior
- schema/config changes with missing validation
- poor diagnosability on risky paths
- missing logs, metrics, or traceability where failures would be hard to debug
- rollback ambiguity for high-risk changes

#### G. Interface misuse probes

Use for `interface-review` and for any changed public surface.

For every security-relevant or failure-relevant choice point, probe as applicable:

- `0`
- empty string or empty collection
- `null` / `None`
- negative values
- omitted argument or missing key
- swapped same-typed parameters
- dangerous literals like `"none"`, `"*"`, `"false"`, weak algorithms, broad bind addresses

Check for:

- missing server-side enforcement for tenant binding, ownership, or sensitive action checks
- insecure or failure-prone defaults
- stringly typed security decisions
- algorithm or mode selection footguns
- auth, cookie, token, or CORS defaults that make the easy path unsafe on HTTP-facing or public interfaces
- config cliffs where one setting disables safety
- silent failures or ignored return values
- type confusion between distinct concepts

Every interface finding must include a concrete misuse path.

### 5. Verify each candidate finding

For each issue candidate:

1. Check whether it is already handled elsewhere in the relevant path.
2. Check whether tests cover the scenario.
3. Classify it as `confirmed` or `suspected`.
4. If it cannot be assessed, move it to `unverified-areas` instead of overstating it.
5. Assign severity.

## Severity rules

- `Critical`: exploitable security flaw, data loss/corruption, or common-path crash/failure
- `High`: likely bug or security issue in expected usage; easy misconfiguration breaks safety; merge should normally stop
- `Medium`: real but narrower edge-case risk, incomplete hardening, or likely future regression surface
- `Low`: limited-scope or defensive issue worth noting but not usually merge-blocking

## Finding scoring and ranking

Score only `confirmed` and `suspected` findings. Do not score `unverified-areas`.

Score each finding with whole numbers:

- `impact`:
  - `0` negligible
  - `1` local incorrectness or minor degradation
  - `2` user-visible failure, partial data risk, or meaningful security/reliability impact
  - `3` security compromise, data loss/corruption, outage risk, or common-path failure
- `likelihood`:
  - `0` weakly supported or highly contingent
  - `1` rare edge case
  - `2` plausible in normal use or common misuse
  - `3` default, obvious, or common path
- `exposure`:
  - `0` narrow internal path
  - `1` limited caller set or limited blast radius
  - `2` public/common path, many callers, or dangerous default
- `recovery`:
  - `0` easy to detect and undo
  - `1` moderate cleanup or rollback difficulty
  - `2` hard to detect, hard to repair, or risky rollback

`finding-score = impact + likelihood + exposure + recovery` with range `0-10`.

### Ranking rules

- If there are 0-2 findings total, fully explain all findings.
- If there are 3 or more findings, compute the average finding score across scored findings.
- Give full write-ups to:
  - every `Critical` or `High` finding
  - every finding with score `>= 7`
  - every finding with score strictly greater than the batch average
- Remaining `Medium` and `Low` findings may be listed briefly under `secondary-findings`.
- Never use scoring to hide a real `Critical` or `High` issue.
- Never turn speculative concerns into high-scored findings without evidence.

## Mode-specific rules

`change-review`:

- Resolve the full diff before concluding.
- List every changed file in scope.
- Read surrounding context for each changed file.
- If the diff output truncates, inspect files individually until all changed lines are covered.

`code-slice-review`:

- Be explicit about what was and was not seen.
- Do not generalize beyond the reviewed slice.

`interface-review`:

- Map every relevant parameter, return contract, enum, default, config knob, and caller choice point.
- For HTTP, auth, or config surfaces, also identify where authn, authz, tenant/ownership checks, validation, and dangerous defaults are actually enforced.
- Every finding must include:
  - surface
  - misuse path
  - current behavior
  - safer contract or fix

`fix-verification`:

For each original finding:

- locate the supposed fix
- judge whether it addresses root cause or only symptom
- search for sibling patterns nearby
- note regressions or newly introduced issues
- classify as:
  - `fixed`
  - `partially fixed`
  - `not fixed`
  - `new issue introduced`

## Output structure

Use these sections in this order.

`review-context`:

- review mode
- request goal
- `[observed]`
- `[inferred]`
- `missing-inputs`
- `confidence`: `high | medium | low`

`scope-and-coverage`:

- files or surfaces reviewed
- full-context vs partial-context review
- tests/configs/docs/callers examined
- anything intentionally out of scope

`quality-attribute-coverage`: Provide one row per in-scope file or surface. Columns:

- file or surface
- correctness
- security
- reliability
- availability
- performance
- scalability
- maintainability
- testability
- operability
- compatibility
- ergonomics
- notes

`priority-findings`: Fully explain the prioritized findings only. For each:

- `Title`
- `Location`
- `Certainty`: `confirmed | suspected`
- `Severity`
- `Finding score`
- `Category`
- `Problem`
- `Evidence`
- `Why it matters`
- `Suggested fix`

`secondary-findings`: Brief bullets for lower-scored real findings. For each bullet include:

- title
- location
- certainty
- severity
- finding score
- one-line reason it matters
- one-line suggested next step

`fix-verification`: Include only in `fix-verification` mode

`unverified-areas`: List anything that could not be reviewed confidently and why

`overall-assessment`: 2-5 sentences covering -

- aggregate risk
- merge or usage recommendation
- strongest issues
- what should be fixed first
- whether the review is code-only or also supported by tests/evidence

## Clarification mode

Use this instead of the full output only when minimum evidence is missing.

Return:

- `review-mode`
- `known-so-far`
- `decision-blockers`
- `questions-for-user`

## Fallback modes

### Scope too large

If the user asks for a whole-repo review and the scope is too large for trustworthy full coverage:

- ask for a narrower target, or
- perform a `provisional hotspot review`

If doing the latter, label it exactly as `provisional hotspot review`.

### No significant findings

If no significant findings are found:

- say `no significant findings found`
- still include `scope-and-coverage`
- still include `quality-attribute-coverage`
- still list `unverified-areas` if any

### Missing runtime evidence

If tests or execution evidence are unavailable:

- do not overclaim
- say `code review only, not runtime-verified`
