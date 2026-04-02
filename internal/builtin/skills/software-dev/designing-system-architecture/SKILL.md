---
name: designing-system-architecture
description: Guides system architecture decisions, domain boundary mapping, and event-driven pattern evaluation for new systems or major changes. Use when the user needs to choose or evaluate structure, clarify bounded contexts and integration ownership, or assess whether CQRS, event sourcing, or sagas are justified. Compares options from simplest to most complex and outputs ADR-ready decisions.
---

# Designing System Architecture

## Purpose

Choose the simplest viable architecture that satisfies the constraints, define only the boundaries needed for the decision, and introduce async or event-driven patterns only when they solve a real problem.

## Use when

- The user asks how a system or major change should be structured.
- The user asks `monolith or microservices?`
- The user needs module, bounded context, or ownership boundaries clarified.
- The user asks whether DDD is justified.
- The user asks whether async messaging, outbox, CQRS, event sourcing, or sagas are justified.
- The user wants an architecture recommendation with rationale.
- The user wants an ADR-ready decision.
- The user needs contract ownership, upstream/downstream relationships, or migration direction clarified.

## Do not use when

- The task is only framework syntax or implementation code.
- The task is only repo documentation or current-state documentation.
- The task is only tactical DDD design such as entities, aggregates, repositories, or value objects.
- The task is only infrastructure topology without an architecture decision.
- The user only wants diagrams with no recommendation.
- The task is only low-level code organization inside one already-chosen module.

## Optional references

These are formatting aids, not decision logic. Core reasoning lives in this file.

- `references/adr-template.md`
  - Load only when drafting a full ADR in prose.
- `references/context-mapping-template.md`
  - Load only when producing a detailed subdomain map, bounded-context catalog, glossary, relationship matrix, or contract inventory.

Do not load these for ordinary recommendation work.

## Decision protocol

Follow this sequence.

1. Read this file fully before answering.
2. Determine which scope is in play. Use one or more of:
   - `structure-decision`
   - `boundary-clarification`
   - `event-pattern-evaluation`
3. Check whether minimum evidence exists for that scope.
4. If minimum evidence is missing, do not force a recommendation. Use `clarification mode`.
5. If a missing fact would materially change the option set or recommendation, ask for it before deciding.
6. Ask in one compact batch, usually 3-7 questions.
7. Do not ask questions already answered or questions that would not change the decision.
8. If enough information exists to compare viable options, proceed without further questioning.
9. If the user explicitly allows assumptions, keep them minimal and mark them `[assumed]`.
10. Separate `[observed]` from `[inferred]`.
11. Compare 2-3 viable options when multiple options are genuinely viable.
12. If hard constraints leave only one viable option, say so explicitly and name the eliminated alternatives.
13. Prefer the simplest viable option.
14. Never default to microservices, CQRS, event sourcing, sagas, or full DDD.
15. One bounded context is a valid outcome.
16. Do not invent contexts just to fit DDD vocabulary.
17. Make ownership explicit for modules, data, and contracts where integrations exist.
18. Prefer incremental evolution over rewrite unless hard constraints or migration economics justify rewrite.
19. If a section does not apply, say `not needed` rather than forcing it.
20. If minimum evidence exists but uncertainty remains after one clarification round, provide a provisional recommendation with explicit `confidence`, `open-questions`, and `revisit-triggers`.

## Minimum evidence by scope

### `structure-decision`

Need at least:

- what is being built or changed
- scale or growth trajectory
- ownership or team shape
- top constraints or pain point

### `boundary-clarification`

Need at least:

- main business capabilities
- likely ownership seams
- integrations or contract seams
- overloaded or conflicting terminology if known

### `event-pattern-evaluation`

Need at least:

- consistency requirements
- audit, history, or replay need
- read/write asymmetry if any
- operational readiness

If a requested scope lacks its minimum evidence, ask for it.

## Working definitions

Use these interpretations to reduce drift.

- `layered monolith`: one deployable unit with conventional layers and limited boundary enforcement
- `modular monolith`: one deployable unit with explicit internal module boundaries and contracts
- `bounded context`: a boundary within which terms, rules, and invariants stay internally consistent
- `full DDD`: substantial strategic and tactical modeling investment beyond simple modularization
- `outbox/integration events`: reliable publication of changes to other boundaries while current state remains the source of truth
- `CQRS`: separate read and write models because their needs materially diverge, not merely separate handlers
- `event sourcing`: events are the source of truth and current state is derived from them
- `saga`: coordination of distributed steps with timeout and compensation rules
- `ACL`: a translation layer protecting a consumer model from an upstream model

## Common misreads to avoid

- DDD is not a synonym for microservices.
- Publishing events does not imply event sourcing.
- Separate controllers, handlers, or endpoints do not by themselves justify CQRS.
- A long workflow inside one boundary does not need a saga if one transaction can own it.
- A module boundary is not automatically a bounded context.
- Async messaging does not by itself justify splitting deployment units.

## Discovery question bank

This is a question bank, not a mandatory checklist. Select the smallest subset relevant to the active scope. Do not mechanically ask every question.

### Core context

1. What is being built or changed?
2. What is the current architecture and biggest pain point, if any?
3. How fast must it ship, and how reversible should the first version be?
4. How many engineers or teams will own it, and what is their operational maturity?

### Scale and constraints

1. What are the scale, latency, availability, and data-volume needs now and in 12-24 months?
2. What security, compliance, tenancy, audit, or data residency constraints exist?
3. What integrations, external contracts, or legacy constraints exist?

### Domain and boundaries

1. Is the domain mostly CRUD, or rule-heavy with conflicting models or terminology?
2. Where are the likely ownership seams, contract seams, or data-lifecycle differences?

### Consistency and eventing

1. Where is strong consistency required, and where is eventual consistency acceptable?
2. Are read and write needs materially different in shape, scale, or latency?
3. Is historical replay truly required, or would an audit log be enough?
4. What is the rollback path if the first design proves too complex?

## Workflow

1. Summarize `context` using `[observed]`, `[inferred]`, optional `[assumed]`, `missing-inputs`, and `scope`.
2. Classify `complexity` as `low`, `medium`, or `high`.
3. Build a concrete option set, simplest first.
4. Apply the architecture baseline before considering distribution or evented complexity.
5. If boundaries are part of the decision, define only the minimum subdomains, modules, or bounded contexts needed to support the decision.
6. Evaluate async or evented patterns only after simpler options remain insufficient.
7. Reject unnecessary complexity explicitly.
8. Choose the simplest option that satisfies hard constraints.
9. Record `now` vs `later`, rejected or deferred options, risks, and revisit triggers.
10. Produce an ADR-ready recommendation.

## Complexity guide

- `low`: one team, fast delivery, few integrations, simple domain, one transactional model likely fits
- `medium`: several modules or integrations, moderate rule complexity, some scaling or compliance pressure
- `high`: multiple teams, divergent ownership, conflicting models, hard isolation constraints, or strong operational and audit demands

## Architecture baseline

Default path:

1. layered monolith
2. modular monolith
3. modular monolith plus async integration edges
4. separated services only when strong constraints justify them

Rules:

- Put boundaries in code before boundaries in deployment.
- Add async integration edges before splitting deployment units.
- Prefer incremental change over rewrite.
- Keep the option set concrete, not encyclopedic.
- Typical comparisons:
  - `low`: layered monolith vs modular monolith
  - `medium`: modular monolith vs modular monolith plus async edges
  - `high`: modular monolith plus async edges vs separated services
- Choose only options that are genuinely viable under the stated constraints.

### Distribution guardrails

Separated services are viable only when most of these are true:

- boundaries and data ownership are clear
- independent deployment or scaling is a real need
- teams can own services end-to-end
- eventual consistency is acceptable where boundaries exist
- observability, CI/CD, on-call, and incident response are mature

Do not recommend separated services when any of these dominate:

- one team owns everything
- a shared database would remain
- boundaries are still fuzzy
- delivery speed and reversibility matter more than isolation
- operational maturity is weak

Fallback: modular monolith.

### DDD gate

Use full DDD only when at least 2 are true and real domain knowledge is available:

- rules are complex or fast-changing
- multiple teams or modules collide on meaning
- contracts need translation or are unstable
- invariants or auditability are business-critical

If the gate fails, prefer clear modules, explicit contracts, and a small glossary.

## Boundary rules

Use these only to the depth needed for the decision.

- Split boundaries where language, invariants, ownership, release cadence, data lifecycle, or compliance needs diverge.
- Do not split by UI layer, database table shape, or current team chart alone.
- Classify subdomains as `core`, `supporting`, or `generic` only when it helps prioritization.
- One context is valid if the domain does not justify more.
- Shared models are expensive. Use them only with explicit joint governance.
- Reject `Shared Kernel` unless joint ownership is explicit and durable.
- If candidate contexts are unclear, return a provisional map with confidence and open questions instead of pretending certainty.
- For explicit context mapping work, consider each candidate context pair and mark it `relevant` or `irrelevant`.
- For each relevant relationship, state direction, contract owner, translation rule, versioning rule, and likely failure modes.
- Name rejected relationship patterns when the choice materially affects ownership or coupling.
- Consumer-specific translation belongs in the consumer or its ACL unless the provider intentionally owns a published language.
- Additive change first. Breaking change requires a new version and migration window.
- APIs and events must define behavior for timeout, partial data, schema mismatch, and unavailability.
- Async consumers must tolerate duplicates, reordering where applicable, and version skew.

### Relationship patterns

Use these labels only when explicit mapping helps:

- `Partnership`: joint governance and shared cadence
- `Customer-Supplier`: provider owns the contract; consumer can negotiate needs
- `Conformist`: consumer accepts the provider model
- `ACL`: consumer translates to protect its model
- `Open Host Service + Published Language`: provider offers a stable shared contract for several consumers
- `Shared Kernel`: only with explicit joint governance

## Event-pattern gates

Do not evaluate evented patterns unless simpler structural options have been considered.

Compare, in order:

1. `CRUD`
2. `CRUD + audit log`
3. `CRUD + outbox/integration events`
4. `CQRS`
5. `event sourcing`

Rules:

- Prefer `CRUD` when one transactional model is enough.
- Prefer `CRUD + audit log` when history or compliance is needed but current state remains the source of truth.
- Prefer `CRUD + outbox/integration events` when reliable async integration is needed without changing the core write model.
- Use `CQRS` only when read and write needs materially diverge in model, scale, or latency.
- Use `event sourcing` only when events must be the source of truth, replay is valuable, and the team can operate it.
- Projections are part of CQRS or event sourcing, not a standalone choice.
- Sagas are workflow control for distributed steps, not a justification by themselves to split services.
- Use sagas only for cross-boundary workflows that cannot stay in one transaction and have real compensation, timeout, or human-wait requirements.
- If operational readiness is weak, reject event-heavy designs.

### Strong triggers

- temporal queries or replay are product requirements
- many read models must be built from the same change stream
- read and write models differ materially
- reliable async integration across boundaries is required
- long-running compensated workflows are unavoidable

### Anti-triggers

- one transaction is enough
- mostly CRUD plus ordinary reporting
- low operational maturity
- unclear rollback path

### Minimum operational readiness for async or evented designs

Require most of these before recommending async or event-heavy patterns:

- tracing or correlation IDs
- retries with bounded backoff
- idempotent handlers
- DLQ or poison-message handling
- schema versioning policy
- replay or backfill procedure
- clear ownership and on-call

Additional requirements for event sourcing:

- event store backup and recovery
- projection rebuild plan and time budget
- snapshot policy if needed
- upcasting or version migration strategy
- practiced replay outside production

### Workflow control

- Prefer `single-transaction` when the workflow can stay within one boundary.
- Prefer `orchestration` when ordering, visibility, timeouts, human waits, or manual intervention matter.
- Prefer `choreography` when steps are loosely coupled and local autonomy matters more than central flow control.
- Every non-atomic distributed step needs an idempotent compensation or an explicit statement that it is not reversible.
- Compensate business effects, not history.

## Output

Use these sections in this order when giving a recommendation. If a section does not apply, say `not needed`.

### `context`

- `[observed]`
- `[inferred]`
- `[assumed]` if any
- `missing-inputs`
- `scope`
- `complexity`
- `confidence`: `high` | `medium` | `low`

### `decision-options`

For each viable option:

- summary
- fit
- main benefits
- main costs
- why it is viable here

If only one option is viable, say which constraints eliminated the others.

### `chosen-architecture`

- chosen option
- why it wins over the others
- what is decided now
- what is intentionally deferred

### `boundaries-and-contracts`

Include only what the decision needs:

- subdomains or modules if useful
- bounded contexts or module boundaries
- glossary or overloaded terms if naming matters
- upstreams/downstreams
- contract ownership
- translation and versioning rules
- change approval rules if cross-team contracts exist

### `eventing-decision`

Either:

- `not needed` with reason

Or:

- chosen pattern and why
- event or stream catalog with producer, consumers, key fields, ordering need, idempotency key, and versioning note
- workflow model: `single-transaction`, `orchestration`, or `choreography`
- compensation, timeout, retry, and rollback expectations

### `rejected-options`

- each rejected or deferred option
- why not now
- what would change for it to become viable

### `trade-offs`

- benefits gained
- costs accepted
- complexity intentionally avoided

### `failure-modes`

List the important failure modes for the chosen design, such as:

- semantic mismatch
- timeout or unavailability
- stale contract version
- duplicate or out-of-order delivery
- partial failure across boundaries
- manual recovery path

### `operational-guardrails`

- required guardrails for the chosen design
- ownership and observability expectations
- migration or rollback path

### `risks-and-mitigations`

- key risks
- mitigations or follow-up checks

### `revisit-triggers`

Concrete thresholds or conditions that justify reopening the decision.

### `open-questions`

Include only when confidence is not `high` or the recommendation is provisional.

### `adr-draft`

ADR-ready text. Use the structure from `references/adr-template.md` when full wording is requested.

## Clarification mode

Use this instead of the full recommendation output only when minimum evidence for the active scope is missing.

Return:

- `known-so-far`
- `decision-blockers`
- `questions-for-user`

Keep `questions-for-user` compact and prioritized.

## If information is missing

- Do not guess.
- Ask only for missing facts that would change the option set or recommendation.
- If minimum evidence is missing, use `clarification mode`.
- If minimum evidence exists but uncertainty remains, give a provisional recommendation using the full output schema with explicit `confidence`, `open-questions`, and `revisit-triggers`.
