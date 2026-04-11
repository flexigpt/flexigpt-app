---
name: designing-system-architecture
description: Guides system architecture decisions, domain boundary mapping, and event-driven pattern evaluation. Use when the user needs to choose or evaluate structure, clarify bounded contexts and integration ownership, or assess whether CQRS, event sourcing, or sagas are justified. Compares options simplest to most complex and outputs ADR-ready decisions.
---

# Designing System Architecture

Choose the simplest viable architecture that satisfies constraints. Define only the boundaries needed for the decision. Introduce async or event-driven patterns only when they solve a real problem.

## Use when

- choosing how a system or major change should be structured
- clarifying module, bounded context, or ownership boundaries
- evaluating whether DDD, async messaging, CQRS, event sourcing, or sagas are justified
- producing an architecture recommendation with rationale or ADR

## Do not use when

- the task is only framework syntax or implementation code
- the task is only tactical DDD (entities, aggregates, repositories, value objects)
- the task is only infrastructure topology without an architecture decision
- the task is only diagrams with no recommendation

## Execution model

**Pick scope(s)** from the request:

- `structure-decision`: how to structure the system or change
- `boundary-clarification`: domain boundaries, ownership, contracts
- `event-pattern-evaluation`: whether async or evented patterns are justified

**Workflow:**

1. Summarize context: `[observed]`, `[inferred]`, `[assumed]`, `missing-inputs`, `scope`, `complexity` (low|medium|high).
2. If minimum evidence for the scope is missing, use clarification mode.
3. Build concrete option set, simplest first. Apply architecture baseline before considering distribution or evented complexity.
4. If boundaries are part of the decision, define only the minimum needed.
5. Evaluate async or evented patterns only after simpler options remain insufficient.
6. Reject unnecessary complexity explicitly.
7. Choose simplest option satisfying hard constraints. Record now vs later, rejected options, risks, revisit triggers.
8. Produce ADR-ready recommendation.

**Complexity:**

- `low`: one team, few integrations, simple domain, one transactional model likely fits
- `medium`: several modules or integrations, moderate rules, some scaling or compliance pressure
- `high`: multiple teams, divergent ownership, conflicting models, hard isolation, strong audit demands

**Context priority.** Attached files are current state. Then pasted context and stated requirements. Then named workspace paths. Then adjacent files. Then targeted search. Then questions. Do not skip earlier layers while they have unused evidence.

**Batch everything.** Identify all reads before calling tools. Read together. Ask 3-7 blocker questions in one batch when needed.

**When enough information exists** to compare viable options, proceed. If minimum evidence exists but uncertainty remains, give a provisional recommendation with explicit `confidence`, `open-questions`, and `revisit-triggers`.

**Decision boundary.** This skill ends at recommendation and ADR-ready output. Ask questions only when minimum evidence is missing or the answer would materially change the recommendation. If the user later wants implementation, carry the chosen option and constraints into an implementation skill.

## Hard rules

- Prefer the simplest viable option. Never default to microservices, CQRS, event sourcing, sagas, or full DDD.
- One bounded context is a valid outcome. Do not invent contexts to fit DDD vocabulary.
- Put boundaries in code before boundaries in deployment.
- Add async integration edges before splitting deployment units.
- Prefer incremental evolution over rewrite unless hard constraints justify it.
- Make ownership explicit for modules, data, and contracts where integrations exist.
- Compare 2-3 viable options when genuinely viable. If constraints leave one option, name what eliminated the others.
- Separate `[observed]` from `[inferred]` from `[assumed]`.
- Do not ask questions already answered or that would not change the decision.
- If a section does not apply, say `not needed` rather than forcing it.
- User instructions override this skill.

**Misreads to prevent:**

- DDD is not a synonym for microservices.
- Publishing events does not imply event sourcing.
- Separate controllers or handlers do not by themselves justify CQRS.
- A long workflow inside one boundary does not need a saga if one transaction can own it.
- A module boundary is not automatically a bounded context.
- Async messaging does not by itself justify splitting deployment units.

## Minimum evidence by scope

**structure-decision**: what is being built or changed, scale or growth trajectory, ownership or team shape, top constraints or pain point.

**boundary-clarification**: main business capabilities, likely ownership seams, integrations or contract seams, overloaded terminology if known.

**event-pattern-evaluation**: consistency requirements, audit or history or replay need, read/write asymmetry if any, operational readiness.

## Architecture baseline

Default path, increasing complexity:

1. Layered monolith
2. Modular monolith
3. Modular monolith + async integration edges
4. Separated services (only when strong constraints justify)

Typical comparisons: `low` = 1 vs 2. `medium` = 2 vs 3. `high` = 3 vs 4.

### Distribution guardrails

Separated services viable only when most hold: clear boundaries and data ownership, real independent deployment or scaling need, teams own services end-to-end, eventual consistency acceptable at boundaries, mature observability/CI-CD/on-call/incident response.

Do not recommend when: one team owns everything, shared database would remain, boundaries are fuzzy, delivery speed and reversibility matter more than isolation, operational maturity is weak. Fallback: modular monolith.

### DDD gate

Full DDD only when at least 2 hold and real domain knowledge is available: rules are complex or fast-changing, multiple teams collide on meaning, contracts need translation or are unstable, invariants or auditability are business-critical.

If the gate fails: clear modules, explicit contracts, small glossary.

## Boundary rules

Use only to the depth the decision needs.

- Split where language, invariants, ownership, release cadence, data lifecycle, or compliance diverge.
- Do not split by UI layer, table shape, or current team chart alone.
- Classify subdomains as core/supporting/generic only when it helps prioritization.
- Shared models are expensive. Use only with explicit joint governance.
- Reject Shared Kernel unless joint ownership is explicit and durable.
- If candidate contexts are unclear, return a provisional map with confidence and open questions.

**Relationship patterns** (when explicit mapping helps): Partnership, Customer-Supplier, Conformist, ACL, Open Host Service + Published Language, Shared Kernel.

**Contract rules:** consumer-specific translation belongs in the consumer or its ACL. Additive change first; breaking change requires new version and migration window. APIs and events must define behavior for timeout, partial data, schema mismatch, unavailability. Async consumers must tolerate duplicates, reordering, and version skew.

**Context mapping** (when detailed mapping requested): for each candidate context pair mark relevant or irrelevant. For relevant pairs: direction, contract owner, translation rule, versioning rule, likely failure modes.

## Event-pattern gates

Compare in order:

1. CRUD
2. CRUD + audit log
3. CRUD + outbox/integration events
4. CQRS
5. Event sourcing

- CRUD when one transactional model is enough.
- CRUD + audit log when history or compliance needed but current state remains source of truth.
- CRUD + outbox when reliable async integration needed without changing the write model.
- CQRS only when read and write needs materially diverge in model, scale, or latency.
- Event sourcing only when events must be source of truth, replay is valuable, and team can operate it.
- Projections are part of CQRS or event sourcing, not a standalone choice.
- Sagas only for cross-boundary workflows that cannot stay in one transaction and have real compensation, timeout, or human-wait requirements.
- If operational readiness is weak, reject event-heavy designs.

**Strong triggers:** temporal queries or replay as product requirements, many read models from same change stream, materially different read/write models, reliable async across boundaries, unavoidable long-running compensated workflows.

**Anti-triggers:** one transaction is enough, mostly CRUD plus reporting, low operational maturity, unclear rollback path.

**Operational readiness for async/evented** (require most): tracing/correlation IDs, retries with bounded backoff, idempotent handlers, DLQ/poison-message handling, schema versioning policy, replay/backfill procedure, clear ownership and on-call.

**Additional for event sourcing:** event store backup/recovery, projection rebuild plan, snapshot policy if needed, upcasting/version migration, practiced replay outside production.

**Workflow control:** single-transaction when workflow stays in one boundary. Orchestration when ordering, visibility, timeouts, or intervention matter. Choreography when steps are loosely coupled and local autonomy matters. Every non-atomic distributed step needs idempotent compensation or explicit statement that it is not reversible. Compensate business effects, not history.

## Optional references

Load only when needed, not for ordinary recommendations:

- `references/adr-template.md`: when drafting a full ADR in prose
- `references/context-mapping-template.md`: when producing a detailed subdomain map, context catalog, glossary, or contract inventory

## Artifact format

Use sections in order. Say `not needed` for sections that do not apply.

**context**: `observed`, `inferred`, `assumed`, `missing-inputs`, `scope`, `complexity`, `confidence` (high|medium|low).

**decision-options**: for each viable option: summary, fit, benefits, costs, why viable. If one option: what eliminated the others.

**chosen-architecture**: chosen option, why it wins, what is decided now, what is deferred.

**boundaries-and-contracts** (if needed): subdomains/modules, bounded contexts, glossary, upstreams/downstreams, contract ownership, translation/versioning rules.

**eventing-decision**: `not needed` with reason, or: chosen pattern and why, event/stream catalog (producer, consumers, key fields, ordering, idempotency key, versioning), workflow model, compensation/timeout/retry/rollback expectations.

**rejected-options**: each rejected or deferred option, why not now, what would make it viable.

**trade-offs**: benefits gained, costs accepted, complexity avoided.

**failure-modes**: semantic mismatch, timeout/unavailability, stale contracts, duplicate/out-of-order delivery, partial failure, manual recovery.

**operational-guardrails**: required guardrails, ownership/observability expectations, migration/rollback path.

**risks-and-mitigations**: key risks and mitigations.

**revisit-triggers**: concrete thresholds or conditions to reopen the decision.

**open-questions**: only when confidence is not high or recommendation is provisional.

**adr-draft**: ADR-ready text using `references/adr-template.md` structure when full wording requested.

## Clarification mode

Use only when minimum evidence for the active scope is missing after exhausting available context.

Return: `scope`, `known-so-far`, `decision-blockers`, `questions-for-user`.

## Response format

End each response with: `scope`, `complexity`, `confidence`, `status`, key assumptions or open questions, `next-step` (only if the user must act).
