# Workflow Declaration and Flow Contract

A Workflow is a composition of operations, not just a graph with additional restrictions. Structured blocks cover ordinary processes. Explicit state machines cover cyclic and event-driven lifecycles. Typed callable targets provide the extension point for domain-specific work.

## 1. Decision and Scope

### 1.1 Version decision

This is the `v1` contract.

It replaces the existing declaration through a breaking change. There are no current consumers, so the language needs no compatibility layer or legacy execution format.

### 1.2 Core model

```text
Agent       = reusable reasoning and interaction configuration

Workflow    = typed process, coordination, and lifecycle declaration

Invocation  = one durable execution of a Workflow
```

A Workflow owns process behavior:

- What work starts.
- Which inputs each operation receives.
- How work depends on other work.
- How state changes.
- Which events or decisions permit progress.
- How success, failure, cancellation, and compensation affect the process.

An Agent owns its reasoning and internal interaction behavior. Workflows coordinate Agents in the same way that they coordinate other typed capabilities.

### 1.3 Coverage

The language supports:

| Area         | Supported behavior                                                                            |
| ------------ | --------------------------------------------------------------------------------------------- |
| Work         | Agent, Tool, MCP Tool, Model, service, code activity, and nested Workflow calls.              |
| Composition  | Sequences, choices, parallel branches, dependency-based work, and reusable subworkflows.      |
| Dynamic work | Runtime target selection, dynamic collections, and planner-worker processes.                  |
| Iteration    | Counted loops, conditional loops, iterative refinement, and concurrent collection processing. |
| State        | Typed process state, explicit updates, and isolated parallel state.                           |
| Coordination | Messages, signals, callbacks, typed streams, and conversation handoff.                        |
| Human work   | Approvals, forms, assignments, escalation, and multi-person decisions.                        |
| Time         | Delays, deadlines, expiry, recurring schedules, and missed-schedule behavior.                 |
| Lifecycle    | Long-lived state machines, attached children, and independent processes.                      |
| Reliability  | Retry, recovery, cancellation, reconciliation, and compensation.                              |
| Control      | Execution budgets, concurrency limits, rate limits, and keyed resource claims.                |

Domain-specific algorithms and integrations are typed callable work. They do not require new control-flow constructs.

## 2. Document and Block Model

### 2.1 Root document

```yaml
apiVersion: v1
type: workflow
name: example

inputSchema:
  type: object

outputSchema:
  type: object

targets: {}

steps: []

output: {}
```

| Field          | Required | Meaning                                        |
| -------------- | -------: | ---------------------------------------------- |
| `apiVersion`   |      Yes | Language version. Must be `v1`.                |
| `type`         |      Yes | Must be `workflow`.                            |
| `name`         |      Yes | Portable Workflow name.                        |
| `inputSchema`  |      Yes | Accepted invocation input.                     |
| `outputSchema` |      Yes | Successful invocation output.                  |
| `schemas`      |       No | Reusable schema definitions.                   |
| `targets`      |       No | Named callable dependencies.                   |
| `targetSets`   |       No | Allowed target sets for dynamic dispatch.      |
| `events`       |       No | Typed external event contracts.                |
| `channels`     |       No | Workflow-local typed message channels.         |
| `ports`        |       No | Published typed message or stream ports.       |
| `triggers`     |       No | Event and schedule entry points.               |
| `requirements` |       No | Additional named capability requirements.      |
| `resources`    |       No | Named concurrency and rate-control resources.  |
| `limits`       |       No | Invocation budgets and deadlines.              |
| `state`        |       No | Initial typed process state.                   |
| `listen`       |       No | Event subscriptions active for the root scope. |
| `steps`        |      Yes | Ordered root steps.                            |
| `output`       |      Yes | Explicit successful output mapping.            |
| `catch`        |       No | Error handlers.                                |
| `finally`      |       No | Finalization block.                            |

Requirements implied by the declaration are automatic. For example, a call that connects a streaming port requires streaming support. Authors need not repeat that requirement in a feature list.

### 2.2 Blocks

A block is the basic unit of composition.

A block contains:

- An ordered `steps` array.
- An optional `output` mapping.
- Optional local `state`.
- Optional event subscriptions in `listen`.
- Optional `catch` handlers.
- An optional `finally` block.

Steps run in declaration order unless a composition operation introduces concurrency.

A nested block returns its explicit `output`. Its default output is `null`. The root Workflow always declares `output`.

There is no implicit last-step result.

### 2.3 Steps

Each step has an `id` and exactly one operation.

```yaml
- id: research
  when:
    $expr: input.includeResearch

  timeout: PT2M

  retry:
    maxAttempts: 3
    match:
      categories: [transient, timeout]
    backoff:
      initial: PT1S
      maximum: PT20S
      multiplier: 2

  call:
    target: researcher
    input:
      request:
        $expr: input.request
```

Common step fields are:

| Field             | Meaning                                                               |
| ----------------- | --------------------------------------------------------------------- |
| `id`              | Stable identity within its block.                                     |
| `when`            | Boolean condition evaluated before the operation starts.              |
| `timeout`         | Maximum elapsed duration, including retries and waits.                |
| `retry`           | Retry policy for the logical operation.                               |
| `effect`          | Effect classification, idempotency, reconciliation, and compensation. |
| `claims`          | Required named resource claims.                                       |
| `limits`          | Lower limits for this step and its descendants.                       |
| One operation key | The work or composition performed by the step.                        |

Step IDs are unique within their block. A nested block cannot shadow an enclosing step ID that remains visible.

An operation instance has its own identity for each loop iteration or collection item. Retry attempts retain the same logical operation identity.

## 3. Values, Types, and State

### 3.1 Schemas

Workflow uses JSON Schema Draft 2020-12 for:

- Workflow input and output.
- Callable interfaces.
- Process state.
- Events and messages.
- Human responses.
- Optional intermediate-result constraints.

Schema references resolve from the declaration and its declared schema dependencies.

Validation occurs when a value crosses its declared boundary. Static checks validate what can be established from the declaration. Data-dependent constraints are checked when values become available.

A business rejection can be a valid output. Schema validity and business acceptance are separate decisions.

### 3.2 Value construction

A value is a JSON literal, an object, an array, or an expression.

```yaml
input:
  request:
    $expr: input.request
  documents:
    $expr: input.documents.filter(d, d.enabled)
  language:
    $expr: "has(input.language) ? input.language : 'en'"
  mode: concise
```

An expression uses a single-key object:

```yaml
$expr: input.price * input.quantity
```

Ordinary strings remain literal strings. A literal directive-shaped value can use `$literal`.

Expressions use CEL with a defined, pure standard library for:

- Boolean logic and comparison.
- Arithmetic.
- Conditional values.
- Strings.
- Lists and maps.
- Collection filtering and projection.
- Presence checks.
- Timestamps and durations.

Expressions construct values and evaluate conditions. Custom algorithms and external access use typed callable targets.

All expressions in one binding operation see the same input snapshot.

### 3.3 Expression scopes

| Scope       | Meaning                                                           |
| ----------- | ----------------------------------------------------------------- |
| `input`     | Immutable Workflow invocation input.                              |
| `state`     | Current state of the nearest state-owning scope.                  |
| `steps`     | Completed step outcomes in the current and enclosing blocks.      |
| `item`      | Current collection item.                                          |
| `index`     | Current collection index.                                         |
| `result`    | Current controlled-block result, where the construct defines one. |
| `event`     | Event currently being matched or handled.                         |
| `messages`  | Authorized channel snapshots at the binding boundary.             |
| `error`     | Error being handled.                                              |
| `outcome`   | Block outcome available during finalization.                      |
| `completed` | Confirmed operation data available during compensation.           |
| `operation` | Original operation data available during reconciliation.          |
| `context`   | Defined invocation metadata and logical evaluation time.          |

For example:

```yaml
$expr: steps.research.output.findings
```

A nested block can read completed enclosing results. Parallel branches cannot read each other’s unfinished results.

Missing values produce a binding error unless the expression handles absence. Conditional expressions evaluate only the selected branch.

### 3.4 Step outcomes

A completed step has an outcome:

```text
status
output, when successful
error, when failed or uncertain
metadata
```

Terminal statuses are:

```text
succeeded
failed
skipped
cancelled
uncertain
```

A false `when` condition produces `skipped`, with no output.

Pending, running, and waiting states can be observed during execution, but their unfinished outputs are unavailable to expressions.

### 3.5 Process state

State is optional and explicitly typed.

```yaml
state:
  schema:
    type: object
    required: [attempts, approved]
    properties:
      attempts:
        type: integer
        minimum: 0
      approved:
        type: boolean
    additionalProperties: false

  initial:
    attempts: 0
    approved: false
```

`set` applies one atomic state update:

```yaml
- id: record-review
  set:
    values:
      attempts:
        $expr: state.attempts + 1
      approved:
        $expr: steps.review.output.approved
```

All update expressions read the previous state snapshot. The complete updated state must satisfy its schema.

`set` supports:

- `values`: shallow field assignments.
- `remove`: field names to remove.

It returns the updated state.

A nested state declaration creates local state. A sequential block without local state uses the enclosing state.

Parallel branches and collection items receive isolated state snapshots. Their changes become parent state only through an explicit merge after their results are available.

## 4. Callable Work

### 4.1 Targets

Targets have a named reference and a typed interface.

```yaml
targets:
  researcher:
    kind: agent
    ref: researcher

  release-items:
    kind: tool
    ref: issue-tracker.list-release-items

  summarize:
    kind: model
    ref: summary-model

  publish:
    kind: service
    ref: releases.publish

  calculate:
    kind: activity
    ref: pricing.calculate

  review-process:
    kind: workflow
    ref: review-process
```

| Kind       | Meaning                                                             |
| ---------- | ------------------------------------------------------------------- |
| `agent`    | Reusable reasoning and interaction capability.                      |
| `tool`     | Named Tool operation, including an MCP Tool.                        |
| `model`    | Direct Model request with a declared request and response contract. |
| `service`  | Typed local or remote service operation.                            |
| `activity` | Named program, function, script, or worker activity.                |
| `workflow` | Another Workflow.                                                   |

The target interface declares:

- Input, output, and error schemas.
- Supported invocation modes.
- Accepted context forms.
- Typed message and streaming ports.
- Effect and idempotency behavior.
- Applicable capability limits.

A target binding supplies transport, authentication, and deployment details. Local and remote bindings use the same flow contract.

### 4.2 Invoke

`call` defaults to typed call-return behavior.

```yaml
- id: research
  call:
    target: researcher
    input:
      request:
        $expr: input.request
    context:
      messages:
        $expr: messages.research
      attachments:
        $expr: input.attachments
```

The call validates its input, invokes the target, and returns typed output or a typed error.

A nested Workflow receives the mapped input as its own invocation input.

Direct Model calls are part of this contract. Their interfaces define request messages, generation parameters, structured output, and available stream ports.

### 4.3 Explicit context transfer

Only mapped call input, context, and ports cross the target boundary.

Context can include explicit projections of:

- Conversation history.
- Channel messages.
- Attachments.
- Tool results.
- Prior Agent outputs.
- Other permitted context.

Omitted context is empty.

Incoming content retains its declared data or context classification. It does not acquire instruction authority merely because another Agent or Tool produced it.

### 4.4 Dynamic dispatch

A call can select its target at runtime.

```yaml
targetSets:
  specialists:
    members:
      - researcher
      - reviewer
```

```yaml
- id: delegate
  call:
    target:
      $expr: steps.plan.output.assignee
    allowed: specialists
    input:
      $expr: steps.plan.output.task
```

The selected target must belong to the allowed set and satisfy the call’s interface.

A target set can be:

- An explicit member list.
- A typed, constrained set supplied by an authorized resolver.

Resolver work is an ordinary call. This makes discovery failures, timing, and retries explicit.

Heterogeneous targets can use `choose` blocks or typed adapters to map their different interfaces.

The concrete target is fixed once the operation starts. Retry and resumption use that same target.

### 4.5 Asynchronous work

A call can start work and return a handle:

```yaml
- id: start-analysis
  call:
    target: analyzer
    mode: start
    lifetime: attached
    input:
      $expr: input.document

- id: collect-analysis
  wait:
    for: completion
    handle:
      $expr: steps.start-analysis.output.handle
```

`lifetime` is:

| Value         | Meaning                                                                     |
| ------------- | --------------------------------------------------------------------------- |
| `attached`    | Work belongs to the enclosing scope. This is the default.                   |
| `independent` | Work becomes a separately owned process with its own lifecycle and budgets. |

An attached scope settles its children before it completes. Failure or cancellation propagates to attached work.

An independent start returns a reference, not the child’s final result. Its ownership and cancellation relationship are explicit. This supports background business processes without confusing them with ordinary subroutine calls.

Async starts and completion waits can express dependency DAGs that do not fit a simple nested fork-join structure.

### 4.6 Conversation handoff

`mode: handoff` is an Agent call that changes conversation ownership.

It declares:

- The bound conversation.
- The target Agent.
- The transferred input and context.
- Whether completion restores the source owner or retains the target owner.

Ownership changes after target acceptance. The target’s result follows normal success, failure, wait, cancellation, and uncertainty semantics.

`invoke` delegates work. `handoff` transfers conversation ownership. These are separate authoring decisions.

### 4.7 Typed ports and streams

Targets can declare typed input and output ports. A call can connect a port to a local channel:

```yaml
ports:
  progress:
    channel: analysis-progress
    access: write
```

Port direction, schema, and permissions must agree.

Ports support:

- Progress updates.
- Model or Agent output streams.
- Incremental results.
- Live parent-child messaging.

Final call output remains separate from streamed messages.

## 5. Composition and Control Flow

### 5.1 Operation set

| Operation   | Purpose                                                   |
| ----------- | --------------------------------------------------------- |
| `call`      | Invoke, start, dispatch, or hand off to a target.         |
| `transform` | Construct a deterministic value.                          |
| `set`       | Update typed process state.                               |
| `scope`     | Create a nested block and lifecycle boundary.             |
| `choose`    | Select one conditional block.                             |
| `parallel`  | Run named branches with a completion policy.              |
| `foreach`   | Process a captured collection.                            |
| `repeat`    | Repeat a process with carried state.                      |
| `machine`   | Execute an explicit state lifecycle.                      |
| `wait`      | Suspend for time, events, completion, or human input.     |
| `emit`      | Append or publish a typed message or event.               |
| `saga`      | Coordinate compensatable effects.                         |
| `cancel`    | Request cancellation of referenced work.                  |
| `return`    | Complete a block or the Workflow with an explicit result. |
| `fail`      | Produce a typed error.                                    |
| `break`     | Exit the nearest Repeat.                                  |
| `continue`  | Complete the current Repeat iteration.                    |

### 5.2 Transform

```yaml
- id: prepare-request
  transform:
    value:
      title:
        $expr: input.title
      enabledDocuments:
        $expr: input.documents.filter(d, d.enabled)
    outputSchema:
      type: object
      required: [title, enabledDocuments]
```

A transform returns `value`. Its optional `outputSchema` constrains the constructed result.

### 5.3 Choice

`choose` evaluates ordered cases and runs the first matching block.

```yaml
- id: decide
  choose:
    cases:
      - when:
          $expr: steps.review.output.approved
        do:
          steps: []
          output:
            status: approved

    default:
      steps: []
      output:
        status: needs-revision
```

Each case is `{when, do}`. `default` is an optional block.

The selected block’s output becomes the choice output. No match and no default produces `routing.no_match`.

Inclusive conditional branching uses guarded parallel branches.

### 5.4 Parallel work

`parallel.branches` is a map from branch name to `{when?, do}`.

All branch conditions evaluate from the same entry snapshot. Only selected branches participate in completion.

```yaml
parallel:
  completion:
    mode: all
  errors: collect
  branches:
    legal:
      do:
        steps: []
        output:
          required: true

    security:
      when:
        $expr: input.requiresSecurityReview
      do:
        steps: []
        output:
          required: true
```

Completion policies are:

| Mode              | Completion condition                                     |
| ----------------- | -------------------------------------------------------- |
| `all`             | Every selected branch succeeds.                          |
| `first-completed` | The first terminal branch outcome determines the result. |
| `first-success`   | The first successful branch wins.                        |
| `quorum`          | A declared positive number of branches succeed.          |

For `all`, `errors` is:

- `fail-fast`: stop remaining work after a failure.
- `collect`: wait for all branches and return their outcomes.

For `first-success` and `quorum`, ordinary branch failures are retained while the required success remains possible. If success becomes impossible, the operation returns an aggregate error.

`remaining` is `cancel` or `wait`, with `cancel` as the default.

The result contains named branch outcomes and, where applicable, winning branch names. Unselected branches are `skipped`.

The operation settles remaining attached work before returning. Uncertain effects remain visible even when another branch satisfies the success condition.

### 5.5 Foreach

```yaml
foreach:
  items:
    $expr: input.files
  itemSchema:
    type: object
  concurrency: 4
  maxItems: 1000
  errors: collect
  do:
    steps: []
    output:
      name:
        $expr: item.name
      position:
        $expr: index
```

Foreach behavior:

- Captures the source collection once.
- Validates each item.
- Creates isolated item scopes.
- Runs within effective concurrency and resource limits.
- Returns item outcomes in source order.
- Returns an empty collection for empty input under `all` completion.

Foreach supports the same completion policies as Parallel. This permits all-item processing, first-match searches, and quorum processing.

Its result contains `items`, an ordered array of outcomes, and winning indices when applicable.

`maxItems` is an optional author limit. Platform and inherited work budgets still apply.

### 5.6 Repeat

Repeat supports:

- `count`: a fixed number of iterations.
- `while`: a condition checked before each iteration.
- `until`: a condition checked after each iteration.
- `lifetime: until-cancelled`: an explicitly ongoing process.

Exactly one of these controls is selected.

A Repeat can declare local carried `state`. Its `do` block runs once per iteration. State initialization occurs once when the Repeat starts.

Optional controls are:

- `maxIterations`.
- `timeout`.
- `onExhausted: fail` or `return`, defaulting to `fail`.

The body can update carried state with `set`. A body-local state declaration is initialized separately for each iteration.

The result is:

```text
state
result       = final body output, or null if no iteration ran
iterations
exhausted
```

The termination condition is checked before exhaustion is declared.

Retry reattempts the same logical operation. Repeat performs a new process iteration with current state and new operation instances.

### 5.7 Early completion

`return` contains a value and a destination:

```yaml
return:
  to: workflow
  value:
    status: already-complete
```

`to` is `block` or `workflow`, defaulting to `block`.

A Workflow-level return cancels remaining attached work, runs finalization, and validates the selected result against `outputSchema`.

A first accepted Workflow-level return determines the selected result when concurrent returns race. Outstanding uncertain effects remain part of the invocation outcome.

`break` and `continue` apply to Repeat. Per-item early completion uses block return. Early collection completion uses Foreach completion policies.

## 6. State Machines and Long-Lived Processes

A state machine represents a business lifecycle with named states and explicit transitions.

It is appropriate when the process can revisit states, wait for different events, or remain active for a long period.

A machine declares:

- `initial`: initial state name.
- Optional carried `state`.
- Optional `listen` subscriptions.
- Named `states`.

Each state can declare:

- `entry`: a block run on entry.
- `exit`: a block run on exit.
- Ordered `transitions`.
- `final: true` and an `output`.

A transition contains:

- An event subscription, an `after` timer, or an immediate condition.
- Optional `when`.
- Optional `do`.
- A destination `to`.

```yaml
machine:
  initial: open

  listen:
    updates:
      event: case-updated
      correlation:
        caseId:
          $expr: input.caseId

  states:
    open:
      transitions:
        - event: updates
          when:
            $expr: event.data.action == 'close'
          to: closed

        - event: updates
          to: open

        - after: P30D
          to: expired

    closed:
      final: true
      output:
        status: closed

    expired:
      final: true
      output:
        status: expired
```

In this example, each update re-enters `open` and restarts its inactivity timer.

Transition semantics are:

- Entry actions finish before transition selection.
- Immediate transitions are checked first.
- Event and timer transitions then wait for an eligible occurrence.
- For the same occurrence, the first matching transition wins.
- The machine exits the current state, runs transition actions, and enters the destination.
- A state timer starts on entry and ends on exit.
- A final state returns its explicit output.

Event handling is serialized within one machine. Parallel machines express independent lifecycles.

Machines can contain ordinary Workflow blocks, including nested machines. Repeat remains the simpler form for iterative computation.

## 7. Events, Messages, Human Work, and Time

### 7.1 External events

An event has a typed envelope:

```text
id
type
source
data
correlation
time
```

An event declaration defines its data schema and source contract.

A subscription in `listen` specifies:

- Event contract.
- Correlation values.
- Optional filtering.
- Retention and capacity constraints.

Subscriptions become active when their scope starts, before its steps execute.

This makes request-and-callback processes safe when a callback arrives before the corresponding wait.

```yaml
- id: payment
  scope:
    listen:
      confirmation:
        event: payment-confirmed
        correlation:
          orderId:
            $expr: input.orderId

    steps:
      - id: request-payment
        call:
          target: payments-request
          input:
            orderId:
              $expr: input.orderId

      - id: receive-confirmation
        timeout: PT15M
        wait:
          for: event
          subscription: confirmation

    output:
      $expr: steps.receive-confirmation.output.data
```

Matching events remain available within subscription retention until consumed.

An event wait can consume an event or observe it. Consumption affects that subscription, not every other subscriber.

Duplicate identity is defined by source and event ID. Accepted order is defined per subscription; unrelated sources have no implied global order.

### 7.2 Local channels

A channel is a typed append-only log within an invocation.

```yaml
channels:
  review:
    schema:
      type: object
      required: [notes]
      properties:
        notes:
          type: string
      additionalProperties: false
    maxMessages: 50
    maxBytes: 262144
```

Channels can restrict writers and readers by declared step path.

`emit` appends a local message or publishes an external event.

```yaml
- id: publish-draft
  emit:
    to:
      channel: review
    kind: draft
    data:
      notes:
        $expr: steps.write.output.notes
    deduplicationKey:
      $expr: input.requestId
```

An emit returns a receipt.

Channel reads are non-destructive. A channel wait can use a cursor to wait for new messages.

Nested targets receive copied message values unless an explicit typed port grants live access.

External publication follows the destination’s delivery contract. A publication receipt confirms acceptance, not completion of the receiving business process.

### 7.3 Wait

`wait` suspends until its declared condition resolves.

| `for`        | Required declaration                     | Successful output           |
| ------------ | ---------------------------------------- | --------------------------- |
| `timer`      | `after` or absolute `at`                 | Firing information.         |
| `event`      | Subscription and optional match          | Event envelope.             |
| `channel`    | Channel and optional cursor or match     | Message and cursor.         |
| `completion` | Started-operation handle                 | Target’s successful output. |
| `human`      | Assignment, request, and response schema | Validated human response.   |

Completion waits propagate target failure, cancellation, and uncertainty.

Human waits declare authorized users or roles, the information presented, and the accepted response schema.

```yaml
wait:
  for: human
  assignees:
    roles: [release-manager]
  request:
    notes:
      $expr: steps.write.output.notes
  responseSchema:
    type: object
    required: [approved]
    properties:
      approved:
        type: boolean
      comment:
        type: string
    additionalProperties: false
```

A human rejection is response data. The Workflow decides whether it means revision, termination, escalation, or another business outcome.

Multi-person approvals compose human waits with Foreach or Parallel. Timers, choices, and completion policies express reminders, escalation, first-response decisions, and quorum decisions.

Configuration, authentication, and recovery decisions use the same typed human or event waits.

### 7.4 Time

Durations use ISO 8601. Absolute times use RFC 3339.

A timeout measures elapsed time, including durable waiting and retry backoff.

A wait preserves its prepared input, correlation, assignments, and deadline across suspension. Resumption does not silently create a new wait.

A timeout produces a typed error. If an external effect may already have occurred, its effect state is uncertain.

### 7.5 Triggers

Triggers describe entry into the Workflow. They are separate from the body’s internal coordination.

Supported trigger forms are:

| Form                | Behavior                                                             |
| ------------------- | -------------------------------------------------------------------- |
| Explicit invocation | Starts with supplied input. Available without a trigger declaration. |
| Event               | Maps a typed event into invocation input.                            |
| Schedule            | Starts at declared calendar or interval occurrences.                 |

An event trigger declares:

- Event contract and filter.
- Input mapping.
- An instance key when deduplication or instance addressing is needed.
- `start`, `signal`, or `start-or-signal` behavior.

A schedule declares:

- Calendar expression or interval.
- Time zone for calendar schedules.
- Overlap policy: `allow`, `skip`, `queue`, or `replace`.
- Missed-occurrence policy: `skip`, `latest`, or bounded `catch-up`.

Calendar gaps are skipped. Repeated local times produce one occurrence for that scheduled local time. UTC schedules avoid local-time ambiguity.

A trigger starts or addresses an invocation. A subscription receives events within an existing invocation.

## 8. Failure, Effects, and Lifecycle

### 8.1 Typed errors

Errors contain:

```text
code
category
message
retryable
step path
effect state
sanitized details
causes
```

Categories include validation, routing, execution, transient availability, timeout, authorization, limit, cancellation, and uncertainty.

Targets can add domain-specific error codes and typed details.

Business decisions remain output data unless the Workflow explicitly treats them as process failures.

### 8.2 Catch and finally

A block can declare ordered handlers:

```yaml
catch:
  - match:
      categories: [timeout, transient]
    do:
      steps:
        - id: notify-delay
          call:
            target: notify-operator
            input:
              code:
                $expr: error.code
      output:
        status: deferred

finally:
  steps:
    - id: release-session
      call:
        target: release-session
        input: {}
```

A matcher can constrain category, code, or other defined error fields. Specified fields combine with AND. Values within one field are alternatives.

The first matching handler runs.

A handler can:

- Return a recovery result.
- Perform alternate work.
- Request human resolution.
- Raise another error.

An unhandled error propagates to the enclosing scope.

Finalization runs after success, failure, timeout, or cancellation. Its output is ignored. It receives the scope’s `outcome`.

A finalization failure becomes the primary error if the scope had succeeded. Otherwise, it is attached to the original error.

Cancellation retains its cancellation status through ordinary error handling. Finalization provides its cleanup path.

### 8.3 Retry

Retry policy declares:

- `maxAttempts`, including the first attempt.
- Error matching.
- Backoff.
- Optional per-attempt timeout.

Retry reuses prepared input, the selected target, and the logical operation key.

A composite retry restarts its local control and state from the step’s entry snapshot. Completed external effects remain subject to the same idempotency and reconciliation rules.

Automatic retry requires both:

- A matching retry policy.
- A safe effect contract.

A retryable error flag alone is insufficient.

### 8.4 Effect contracts

An effect declaration states retry safety and optional recovery behavior.

```yaml
effect:
  kind: keyed
  key:
    $expr: input.requestId + ':reserve'
  compensate:
    steps:
      - id: release-reservation
        call:
          target: release-reservation
          input:
            reservationId:
              $expr: completed.output.reservationId
```

Effect kinds are:

| Kind             | Meaning                                                        |
| ---------------- | -------------------------------------------------------------- |
| `read`           | No externally visible business mutation.                       |
| `idempotent`     | Repeating the operation has the same business effect.          |
| `keyed`          | The target deduplicates repeated requests by the supplied key. |
| `non-idempotent` | Repetition can create another effect.                          |

The target interface supplies the base effect contract. A call can add safeguards or stronger constraints.

The same logical operation uses the same key across retries. New Repeat iterations and collection items are distinct operations.

### 8.5 Reconciliation and uncertainty

Uncertainty means that an effect may have happened, but its outcome is not known.

An operation can declare an `effect.reconcile` block. It receives the original input, key, and error through `operation`.

Reconciliation reports:

- `applied`, with a confirmed outcome or typed effect receipt.
- `not-applied`.
- `unknown`.

The block can call a status service, await a callback, or request human confirmation.

Confirmed success follows normal output validation. Confirmed non-application can permit retry. An unknown outcome remains uncertain.

A callable that can leave partial effects exposes typed effect receipts, including on failure. Compensation eligibility follows confirmed effects, not merely whole-step success.

### 8.6 Sagas and compensation

A saga defines a business compensation boundary.

```yaml
saga:
  compensateOn: [failed, cancelled]
  compensationErrors: continue
  do:
    steps: []
    output: {}
```

The saga:

- Tracks confirmed effects with declared compensation.
- Compensates in reverse dependency order.
- Can compensate independent effects concurrently within limits.
- Preserves the original failure and records compensation outcomes.
- Supports `continue` or `stop` after a compensation failure.

A handled error inside the body does not fail the saga. An outer handler can recover after the saga has compensated.

Nested Workflow compensation is defined by its callable compensation contract. The parent does not depend on the child’s private steps.

Unknown effects require reconciliation before automatic compensation.

Compensation is a business action. It can reverse or offset an effect; it does not erase history or promise atomic rollback.

### 8.7 Cancellation and child lifetime

`cancel` requests cancellation of referenced work.

Cancellation:

- Stops queued normal work.
- Propagates to attached children.
- Resolves active waits as cancelled.
- Requests cancellation from active targets.
- Runs finalization and applicable compensation.

Cleanup uses a separate bounded cleanup allowance so that an expired normal-work deadline does not prevent cleanup from starting.

Independent processes follow their declared ownership relationship.

A cancellation acknowledgment confirms cancellation handling. It does not prove that an external effect never occurred.

## 9. Limits, Resources, and Consistency

### 9.1 Limits

Limits are optional author constraints, combined with inherited and platform limits.

Standard limits include:

- Total duration.
- Step executions.
- Call attempts.
- Nested Workflow depth.
- Concurrency.
- Model tokens.
- Monetary cost.
- Message and event storage.
- Cleanup duration.

Attached work shares the parent’s aggregate budgets. Child scopes can lower limits.

Independent processes require separate authorized budgets. Starting them still consumes the caller’s applicable work budget.

A long-lived process can omit a declaration deadline. Waiting time and active-work budgets are separate concerns.

### 9.2 Resource claims

Named resource claims provide:

- Concurrency control.
- Rate limits.
- Per-key serialization.
- Shared service capacity.

A step can claim units from a resource, optionally using a key such as customer, tenant, or account.

A scope can hold a claim across several operations. Claims end when their owning scope settles.

This expresses per-customer ordering or shared API throttling without using process state as a lock.

### 9.3 Definition and binding consistency

An invocation uses a stable Workflow definition.

Once an operation starts, its prepared values, concrete target, logical key, and deadlines remain stable through retry, wait, and resumption.

New discovery or target selection is explicit work. It can affect later operations without changing operations already in progress.

### 9.4 Declaration validity

A valid declaration has:

- Known fields and well-formed operations.
- Unique, visible step identities.
- Valid expression scopes and references.
- Compatible callable, channel, and port interfaces.
- Valid machine states and transition destinations.
- Valid loop, completion, and concurrency parameters.
- Explicit root output mapping.
- Satisfiable declared capabilities and resource requirements.

Known declaration and capability errors are rejected before business work starts.

Late availability changes and data-dependent validation failures become typed operational errors before the affected target performs work.

All control constructs in this proposal belong to `v1`. Integration capabilities are declared by targets rather than left as unfinished language operations.

## 10. Complete Example

This example drafts release notes, performs bounded review and revision, requests human approval, and publishes with a stable idempotency key.

The referenced target interfaces provide:

- Writer: returns `notes`.
- Reviewer: returns `approved` and `feedback`.
- Publisher: accepts release notes and supports keyed invocation.

```yaml
apiVersion: v1
type: workflow
name: prepare-release-notes

inputSchema:
  type: object
  required: [requestId, release]
  properties:
    requestId:
      type: string
    release:
      type: string
  additionalProperties: false

outputSchema:
  type: object
  required: [status, notes]
  properties:
    status:
      enum: [published, declined]
    notes:
      type: string
  additionalProperties: false

targets:
  writer:
    kind: agent
    ref: release-note-writer

  reviewer:
    kind: agent
    ref: release-note-reviewer

  publisher:
    kind: tool
    ref: releases.publish-notes

limits:
  concurrency: 2
  duration: P2D

steps:
  - id: draft
    call:
      target: writer
      input:
        release:
          $expr: input.release

  - id: refine
    repeat:
      state:
        schema:
          type: object
          required: [notes, approved]
          properties:
            notes:
              type: string
            approved:
              type: boolean
          additionalProperties: false

        initial:
          notes:
            $expr: steps.draft.output.notes
          approved: false

      until:
        $expr: state.approved

      maxIterations: 3
      onExhausted: fail

      do:
        steps:
          - id: review
            call:
              target: reviewer
              input:
                release:
                  $expr: input.release
                notes:
                  $expr: state.notes

          - id: revise
            when:
              $expr: "!steps.review.output.approved"
            call:
              target: writer
              input:
                release:
                  $expr: input.release
                notes:
                  $expr: state.notes
                feedback:
                  $expr: steps.review.output.feedback

          - id: update-review-state
            set:
              values:
                approved:
                  $expr: steps.review.output.approved
                notes:
                  $expr: >-
                    steps.review.output.approved
                    ? state.notes
                    : steps.revise.output.notes

  - id: approval
    timeout: P1D
    wait:
      for: human
      assignees:
        roles: [release-manager]
      request:
        release:
          $expr: input.release
        notes:
          $expr: steps.refine.output.state.notes
      responseSchema:
        type: object
        required: [approved]
        properties:
          approved:
            type: boolean
          comment:
            type: string
        additionalProperties: false

  - id: publish
    when:
      $expr: steps.approval.output.approved

    retry:
      maxAttempts: 3
      match:
        categories: [transient, timeout]
      backoff:
        initial: PT1S
        maximum: PT10S
        multiplier: 2

    effect:
      kind: keyed
      key:
        $expr: input.requestId + ':publish-notes'

    call:
      target: publisher
      input:
        requestId:
          $expr: input.requestId
        release:
          $expr: input.release
        notes:
          $expr: steps.refine.output.state.notes

output:
  status:
    $expr: >-
      steps.approval.output.approved
      ? 'published'
      : 'declined'
  notes:
    $expr: steps.refine.output.state.notes
```

The example distinguishes three different decisions:

- Review and revision are process iteration.
- Publishing retry is another attempt at the same logical effect.
- Human rejection is a valid business result.

## 11. Comparative Design Decisions

| Decision                                            | Alternative considered                               | Reason                                                                                      |
| --------------------------------------------------- | ---------------------------------------------------- | ------------------------------------------------------------------------------------------- |
| Structured blocks plus explicit state machines      | One general edge graph                               | Sequences, joins, loop state, and lifecycle transitions have direct semantics.              |
| Pure standard expressions plus typed activities     | Tiny mapping operators or embedded unrestricted code | Authors get useful data processing without mixing external effects into value construction. |
| One typed call model                                | Separate process rules for each target kind          | Agent, Model, Tool, service, code, and Workflow work compose consistently.                  |
| Separate invoke, start, and handoff modes           | One ambiguous delegation concept                     | Return values, child lifetime, and conversation ownership are different concerns.           |
| Explicit parallel completion policies               | A single join rule                                   | All-results, first-result, first-success, and quorum are distinct practical requirements.   |
| Scoped state and explicit result merging            | Shared mutable branch state                          | Parallel behavior remains understandable and independent of completion timing.              |
| Events, subscriptions, and durable waits            | Polling as the main coordination mechanism           | Callbacks, human decisions, and long-lived processes become direct declarations.            |
| Business decisions as typed output                  | Treating every rejection as failure                  | Declines, revisions, and negative findings can follow normal process logic.                 |
| Separate work budgets and process lifetime          | Mandatory small bounds on every process              | Finite jobs and long-lived business processes use the same language.                        |
| Explicit effect, reconciliation, and saga contracts | General rollback or exactly-once promises            | The declaration reflects what external systems can actually guarantee.                      |
| Protocol-neutral target bindings                    | Local-only control-flow semantics                    | Remote integrations can use the same language without a separate workflow model.            |

The result is one process language for pipelines, dynamic Agent work, human decisions, asynchronous integrations, and event-driven business lifecycles. Practical breadth comes from these composable contracts, with typed callable work as the extension point.
