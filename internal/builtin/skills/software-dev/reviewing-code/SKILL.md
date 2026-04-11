---
name: reviewing-code
description: Reviews code, diffs, pull requests, modules, and API/config surfaces for bugs, security flaws, reliability issues, performance/scalability risks, maintainability hazards, and misuse-enabling designs. Use when the user asks to review code, find bugs, audit merge safety, verify a fix, or assess whether an interface is safe to use.
---

# Reviewing Code

Find real defects, concrete risks, and misuse-enabling designs in actual code. Rank findings so the most important get full treatment. No generic best practices, style feedback, or checklist theater.

## Use when

- reviewing a PR, diff, commit range, or branch
- reviewing a file, module, snippet, or focused repo area for bugs or production risk
- assessing whether a change is safe to merge
- reviewing API signatures, config schemas, defaults, or security-relevant interfaces for footguns
- verifying whether a fix resolves a prior finding
- auditing prototype or AI-generated code before wider use

## Do not use when

- the main task is architecture design or system-structure choice
- the main task is feature delivery or implementation
- the user wants a broad rewrite or autonomous cleanup
- the user wants only lint/style/tool output
- the user wants runtime penetration testing

## Execution model

**Pick the mode** from the request:

- `change-review`: diff, PR, commit range
- `code-slice-review`: file, module, snippet, repo area
- `interface-review`: API signature, config schema, defaults
- `fix-verification`: checking if a fix resolves a prior finding

**Workflow** (all modes):

1. Lock scope: files/surfaces in review, mode, full vs partial context available.
2. Read surrounding context: containing functions/classes, callers/callees on risky paths, data models, relevant tests/config/docs. Never review only changed lines or isolated functions.
3. Review every in-scope file/surface against the checklist. Mark each attribute `clean|finding|unverified|n/a`.
4. Verify each candidate finding: check if handled elsewhere, check if tests cover it, classify `confirmed|suspected`, or move to `unverified-areas`.
5. Score and rank findings. Full write-ups for prioritized findings.
6. Output the artifact.

**Context priority.** Attached files are current state. Then pasted code, diffs, logs, stated requirements. Then named workspace paths. Then adjacent workspace files. Then targeted search for a specific unknown. Then questions. Do not skip earlier layers while they have unused evidence.

**Batch everything.** Before any tool call, identify all files to read. Read them together. Do not interleave single reads with single questions.

**Maximize progress per turn.** Complete the full review in one response when possible. If scope is too large for trustworthy full coverage, either narrow it or perform a clearly labeled `provisional hotspot review`.

**When blocked.** Exhaust all available context first. Then ask 1-5 true blocker questions in one batch.

**Decision boundary.** This skill ends at review output. Ask questions only when missing context or scope ambiguity would materially change the review. If the user asks to fix findings, convert them into explicit requirements in an implementation skill rather than silently mixing review with edits.

## Hard rules

- Read full file context, not just changed lines or isolated snippets.
- Classify evidence: `confirmed` (supported by code in scope), `suspected` (likely but needs missing context or runtime to confirm), `unverified` (could not assess from available evidence).
- Separate control gaps (protective behavior missing or bypassable) from evidence gaps (material too thin to confirm whether a control exists).
- Never report style-only nits.
- Maintainability findings must tie to concrete risk: bug-proneness, hidden coupling, incomplete-fix risk, misleading contracts, or missing safety net.
- Never say `looks good`, `LGTM`, or `safe to merge` without showing scope, checklist coverage, and unverified areas.
- If no significant findings, say so explicitly and still show scope, coverage, and unverified areas.
- If tests or execution evidence are unavailable, say `code review only, not runtime-verified`.
- Do not overclaim beyond what was seen. Do not generalize beyond the reviewed slice.
- Do not make code changes unless the user also asks for implementation.
- Do not re-read files already provided as attachments unless comparing versions.
- Do not search while direct evidence remains unused.
- User instructions override this skill.

## Mode reference

### change-review

Entry: a diff, PR, commit range, or changed file list with enough surrounding context.

- Resolve the full diff before concluding.
- List every changed file in scope.
- Read surrounding context for each changed file.
- If diff output truncates, inspect files individually until all changed lines are covered.

### code-slice-review

Entry: the snippet, file, module, or explicit repo slice with enough local context.

- Be explicit about what was and was not seen.

### interface-review

Entry: a signature, schema, config surface, or interface definition with defaults or examples if available.

- Map every relevant parameter, return contract, enum, default, config knob, and caller choice point.
- For HTTP, auth, or config surfaces, identify where authn, authz, tenant/ownership checks, validation, and dangerous defaults are actually enforced.
- Every finding must include: surface, misuse path, current behavior, safer contract or fix.
- Apply interface misuse probes from the checklist.

### fix-verification

Entry: the original finding or bug description and the change supposed to fix it.

For each original finding:

- Locate the supposed fix.
- Judge root cause vs symptom fix.
- Search for sibling patterns nearby.
- Note regressions or newly introduced issues.
- Classify: `fixed | partially fixed | not fixed | new issue introduced`.

## Review checklist

For each in-scope file/surface, mark each attribute `clean|finding|unverified|n/a`.

**correctness**: wrong logic/conditions/fields/variables/caller assumptions. Off-by-one and boundary mistakes. Empty/null/single-item/max-size/invalid-state handling. Changed signatures with stale callers. Incomplete fixes addressing symptom not root cause.

**security**: injection (SQL, command, template, header, path). XSS or unsafe output. Authn/authz gaps, IDOR, broken tenant binding, missing ownership enforcement. CSRF on state-changing actions. Secrets in code or logs. Weak crypto or unsafe comparison. Unsafe callback/redirect/webhook/URL-fetch. Unsafe deserialization. Information exposure through logs/errors/debug. DoS via expensive inputs or unbounded work.

**reliability and availability**: swallowed errors or misleading success. Missing timeouts. Retries without idempotency. Partial failure without rollback. Resource leaks. Dependency failure causing avoidable outage. Missing validation of external or deserialized data.

**performance and scalability**: N+1 queries or repeated remote calls in loops. Unbounded queries or full scans. Large in-memory buffering where streaming is safer. Blocking on latency-sensitive paths. Unnecessary recomputation. Concurrency hazards, race windows, lock contention. Unbounded growth in CPU/memory/network.

**maintainability and testability** (only when tied to real risk): duplicated critical logic likely to drift. Dead or hallucinated code implying nonexistent behavior. Hidden coupling making partial fixes likely. Oversized functions/modules obscuring invariants. Weak seams or missing tests on risky behavior. Misleading naming or contracts.

**compatibility and operability**: changed interfaces with stale consumers. Unsafe defaults or migration behavior. Schema/config changes with missing validation. Poor diagnosability on risky paths. Missing logs/metrics/traceability. Rollback ambiguity for high-risk changes.

**interface misuse probes** (interface-review and any changed public surface): probe with `0`, empty, `null`, negative, omitted, swapped same-typed params, dangerous literals (`"none"`, `"*"`, `"false"`, weak algos, broad binds). Check: missing server-side enforcement for tenant/ownership/sensitive actions. Insecure or failure-prone defaults. Stringly typed security decisions. Algorithm/mode selection footguns. Auth/cookie/token/CORS defaults making the easy path unsafe. Config cliffs where one setting disables safety. Silent failures or ignored return values. Type confusion between distinct concepts. Every finding must include a concrete misuse path.

## Severity and scoring

**Severity**:

- `Critical`: exploitable security flaw, data loss/corruption, or common-path crash
- `High`: likely bug or security issue in expected usage; easy misconfiguration breaks safety; normally merge-blocking
- `Medium`: narrower edge-case risk, incomplete hardening, or future regression surface
- `Low`: limited-scope defensive issue; not usually merge-blocking

**Scoring** (whole numbers, confirmed and suspected findings only):

- `impact`: 0 negligible, 1 local, 2 user-visible/meaningful security impact, 3 security compromise/data loss/outage
- `likelihood`: 0 weakly supported, 1 rare edge case, 2 plausible in normal use, 3 default/common path
- `exposure`: 0 narrow internal, 1 limited callers, 2 public/many callers/dangerous default
- `recovery`: 0 easy to detect/undo, 1 moderate cleanup, 2 hard to detect/hard to repair

`finding-score = impact + likelihood + exposure + recovery` (range 0-10).

**Ranking**: 0-2 findings: fully explain all. 3+ findings: compute average score; full write-ups for all Critical/High, all score >= 7, all score > batch average. Remaining findings listed briefly under `secondary-findings`. Never use scoring to hide a real Critical/High issue. Never inflate speculative concerns without evidence.

## Artifact format

Artifacts live in the conversation. Keep them minimal. Use only sections that matter.

**review-context**: `review-mode`, `request-goal`, `observed`, `inferred`, `missing-inputs`, `confidence` (high|medium|low).

**scope-and-coverage**: files/surfaces reviewed, full vs partial context, tests/configs/docs/callers examined, anything out of scope.

**quality-attribute-coverage**: one row per in-scope file/surface. Columns: file/surface | correctness | security | reliability | availability | performance | scalability | maintainability | testability | operability | compatibility | ergonomics | notes.

**priority-findings** (full write-up each): `Title`, `Location`, `Certainty` (confirmed|suspected), `Severity`, `Finding score`, `Category`, `Problem`, `Evidence`, `Why it matters`, `Suggested fix`.

**secondary-findings** (brief bullets): title, location, certainty, severity, score, one-line reason, one-line fix.

**fix-verification** (fix-verification mode only): per-finding status.

**unverified-areas**: what could not be reviewed confidently and why.

**overall-assessment**: 2-5 sentences: aggregate risk, merge/usage recommendation, strongest issues, fix priority, code-only or test-supported.

## Clarification mode

Use only when minimum evidence is missing after exhausting available context.

Return: `review-mode`, `known-so-far`, `decision-blockers`, `questions-for-user`.

## Response format

End each response with: `review-mode`, `confidence`, `status`, key assumptions or unverified areas, `next-step` (only if the user must act).
