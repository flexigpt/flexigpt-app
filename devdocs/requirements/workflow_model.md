# Workflow Declaration and Flow Contract

A Workflow is a typed durable process declaration.

An Agent defines reusable reasoning and interaction behavior. A Workflow defines process behavior, coordination, lifecycle, reliability, and business control flow.

```text
Agent       = reusable reasoning and interaction configuration

Workflow    = typed durable process declaration

Invocation  = one durable execution of a Workflow
```

Workflow v1 is complete only when every valid declaration feature has a typed compiler path, typed runtime behavior, durable persistence semantics, and recovery behavior.

## 1. Scope and version

### 1.1 Language version

Workflow declarations use:

```yaml
apiVersion: v1
type: workflow
```

`apiVersion: v1` identifies the declaration language.

The declaration language version is not the durable runtime payload version.

The current typed runtime payload remains:

```text
artifact.workflow.v1@1
```

The runtime payload can change in place during development. It must not create `artifact.workflow.v1@2`. The declaration language version must remain `v1`.

### 1.2 Workflow responsibilities

A Workflow owns:

- Ordered work.
- Typed callable invocation.
- Runtime target selection.
- Structured control flow.
- State updates.
- Parallel and collection execution.
- State machines.
- Events, channels, ports, waits, and human work.
- Timers and schedules.
- Trigger behavior.
- Retry, reconciliation, compensation, and saga behavior.
- Resource claims and budgets.
- Cancellation and cleanup behavior.

A Workflow does not own:

- Artifact source parsing.
- Artifact storage.
- Credentials.
- Provider configuration.
- Provider clients.
- Native file access.
- MCP client sessions.
- Browser state.
- Frontend state.
- Wails transport behavior.

### 1.3 Agent responsibilities

An Agent owns reusable reasoning and interaction configuration.

An Agent can contribute:

- A Model selection.
- Text and instruction contributions.
- Tool capabilities.
- Skill capabilities.
- MCP capabilities.
- Model system-prompt behavior.
- Agent-specific capability requirements.

A Workflow can invoke an Agent as a typed callable target. The invocation creates a child Agent-loop ProgramFrame. A Workflow input must not become arbitrary chat text.

### 1.4 Invocation responsibilities

An Invocation is one durable Workflow execution.

An Invocation owns:

- Invocation input.
- Selected Workflow definition pin.
- ProgramFrames.
- WorkItems.
- Attempts.
- Waits.
- Timers.
- Trigger occurrences.
- Event and channel records.
- Effect facts.
- Reconciliation facts.
- Compensation facts.
- Resource claims.
- Budgets.
- Execution outcome.

The existing Session v3 document is the only durable root for an Invocation.

## 2. Root declaration

A root Workflow declaration has this shape:

```yaml
apiVersion: v1
type: workflow
name: example

inputSchema:
  type: object

outputSchema:
  type: object

schemas: {}

targets: {}

targetSets: {}

events: {}

channels: {}

ports: {}

triggers: {}

requirements: {}

resources: {}

limits: {}

state:
  schema:
    type: object
  initial: {}

listen: {}

steps: []

output: {}

catch: []

finally:
  steps: []
```

| Field          |                 Required | Meaning                                            |
| -------------- | -----------------------: | -------------------------------------------------- |
| `apiVersion`   | Yes in standalone source | Must be `v1`.                                      |
| `type`         |                      Yes | Must be `workflow`.                                |
| `name`         |                      Yes | Portable Workflow name.                            |
| `inputSchema`  |                      Yes | JSON Schema for invocation input.                  |
| `outputSchema` |                      Yes | JSON Schema for successful root output.            |
| `schemas`      |                       No | Reusable JSON Schema definitions.                  |
| `targets`      |                       No | Named callable targets.                            |
| `targetSets`   |                       No | Static or resolver-backed target collections.      |
| `events`       |                       No | Typed external event contracts.                    |
| `channels`     |                       No | Invocation-local durable message logs.             |
| `ports`        |                       No | Published typed port bindings.                     |
| `triggers`     |                       No | Event and schedule entry points.                   |
| `requirements` |                       No | Named capability requirements.                     |
| `resources`    |                       No | Named claimable resource definitions.              |
| `limits`       |                       No | Invocation limits and cleanup allowance.           |
| `state`        |                       No | Root typed state declaration.                      |
| `listen`       |                       No | Root subscriptions active before root work begins. |
| `steps`        |                      Yes | Ordered root steps.                                |
| `output`       |                      Yes | Explicit successful root output mapping.           |
| `catch`        |                       No | Ordered root failure handlers.                     |
| `finally`      |                       No | Root finalization block.                           |

The root Workflow has no implicit output. The final successful result is the value of `output` or a selected Workflow-level `return`.

## 3. Blocks and step identity

A block contains:

- Optional local state.
- Optional subscriptions.
- Ordered steps.
- Explicit output.
- Optional ordered catch handlers.
- Optional finalizer.

```yaml
steps:
  - id: prepare
    transform:
      value:
        request:
          $expr: input.request

output:
  prepared:
    $expr: steps.prepare.output
```

A block output defaults to `null` only when the block grammar permits omission. The root output is always explicit.

A step identity is unique within its block.

A nested step identity must not shadow an enclosing visible step identity.

A logical operation identity includes:

- Workflow definition identity.
- ProgramFrame path.
- Step identity.
- Repeat iteration identity.
- Foreach item index.
- Target-set selection identity.
- Retry attempt sequence.

A retry does not create a new logical operation identity.

A Repeat iteration and a Foreach item do create a new logical operation identity.

## 4. Values, schemas, and expressions

### 4.1 Canonical JSON values

Workflow values can be:

- JSON literals.
- Objects.
- Arrays.
- CEL expressions.
- Explicit literal directives.

```yaml
input:
  request:
    $expr: input.request
  language: en
  flags:
    - concise
    - reviewed
```

A CEL expression uses one field:

```yaml
$expr: input.price * input.quantity
```

A literal directive suppresses expression interpretation:

```yaml
$literal:
  $expr: this-is-data
```

Ordinary strings are literal strings.

### 4.2 Schema boundaries

Workflow uses JSON Schema Draft 2020-12 at these boundaries:

- Invocation input.
- Invocation output.
- State initialization and state updates.
- Target input.
- Target output.
- Target error.
- Target receipt.
- Event data.
- Event correlation.
- Channel message data.
- Human response.
- Foreach item.
- Step output when `outputSchema` is declared.
- Port payloads.
- Trigger input.

Schema validity and business success are separate concepts.

A business decline can be valid output data.

### 4.3 Expression scope

Expressions can use these variables where the controlling construct provides them:

| Variable    | Meaning                                                 |
| ----------- | ------------------------------------------------------- |
| `input`     | Immutable invocation input.                             |
| `state`     | Current state of the nearest state-owning scope.        |
| `steps`     | Completed step outcomes in the visible scope chain.     |
| `item`      | Current Foreach item.                                   |
| `index`     | Current Foreach or Repeat index.                        |
| `result`    | Controlled-block result where defined.                  |
| `event`     | Current matched event, trigger event, or machine event. |
| `messages`  | Authorized channel snapshot at the evaluation boundary. |
| `error`     | Error visible to a catch handler.                       |
| `outcome`   | Scope outcome visible to finalization.                  |
| `completed` | Confirmed operation facts visible to compensation.      |
| `operation` | Original operation facts visible to reconciliation.     |
| `context`   | Invocation metadata and logical evaluation time.        |

Expressions are pure.

Expressions cannot:

- Call external services.
- Read files.
- Resolve Artifacts.
- Resolve credentials.
- Open MCP connections.
- Invoke Tools.
- Invoke Models.
- Invoke Agents.
- Start timers.
- Mutate state.

External behavior must use a typed callable target or a Workflow operation.

### 4.4 Step outcomes

A completed Workflow step has one terminal outcome:

```text
succeeded
failed
skipped
cancelled
uncertain
```

A successful outcome can contain:

- Typed output.
- Selected target identity.
- Receipt facts.
- Effect facts.

A false `when` condition produces `skipped`.

A skipped outcome has no output.

An uncertain outcome indicates that an external effect may have happened but cannot yet be confirmed.

## 5. State

### 5.1 State declaration

```yaml
state:
  schema:
    type: object
    required: [attempts]
    properties:
      attempts:
        type: integer
        minimum: 0
    additionalProperties: false
  initial:
    attempts: 0
```

State is optional.

A state declaration creates a state-owning scope.

A nested block without local state reads and updates the nearest enclosing state-owning scope.

A nested block with local state initializes and owns its own state.

### 5.2 Atomic update

```yaml
- id: record
  set:
    values:
      attempts:
        $expr: state.attempts + 1
    remove:
      - obsoleteField
```

All assignment expressions observe the same prior state snapshot.

The complete resulting state must validate against the owning state schema.

`set` returns the resulting state.

### 5.3 Isolated concurrent state

Parallel branches and Foreach items receive isolated state snapshots.

Their local state cannot race or mutate parent state directly.

A parent state changes only through an explicit merge rule after child results are available.

## 6. Callable targets

### 6.1 Target kinds

```yaml
targets:
  research-agent:
    kind: agent
    ref: researcher

  summarize:
    kind: model
    ref: summary-model

  list-issues:
    kind: tool
    ref: issues.list

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

Supported target kinds are:

| Kind       | Meaning                              |
| ---------- | ------------------------------------ |
| `agent`    | Reusable Agent capability.           |
| `model`    | Direct Model request.                |
| `tool`     | Tool Artifact or MCP Tool operation. |
| `service`  | Registered typed Service binding.    |
| `activity` | Registered typed Activity binding.   |
| `workflow` | Nested Workflow declaration.         |

A Tool target with `server` is an MCP Tool target.

```yaml
targets:
  publish-issue:
    kind: tool
    server:
      name: issue-tracker
    ref: publish_issue
```

### 6.2 Callable interface

A target, target set, or call can declare a callable interface.

```yaml
interface:
  inputSchema:
    type: object
  outputSchema:
    type: object
  errorSchema:
    type: object
  receiptSchema:
    type: object
  modes: [invoke, start]
  context: {}
  ports: {}
  effect:
    kind: keyed
    keyed: true
```

A callable interface defines:

- Input schema.
- Output schema.
- Error schema.
- Receipt schema.
- Supported modes.
- Accepted context forms.
- Typed ports.
- Effect capability.
- Capability limits.

A call-level interface can narrow a target interface. It must not widen the target interface.

### 6.3 Invocation

```yaml
- id: research
  call:
    target: research-agent
    input:
      request:
        $expr: input.request
    context:
      messages:
        $expr: messages.research
```

A call evaluates input and context once.

The selected concrete target, prepared input, selected target-set membership, effect key, retry identity, and deadline remain stable for that logical operation.

### 6.4 Modes

| Mode      | Meaning                                            |
| --------- | -------------------------------------------------- |
| `invoke`  | Typed call-return behavior.                        |
| `start`   | Starts child work and returns a completion handle. |
| `handoff` | Transfers conversation ownership to an Agent.      |

`invoke` is the default.

`start` supports:

| Lifetime      | Meaning                                                          |
| ------------- | ---------------------------------------------------------------- |
| `attached`    | Child belongs to the parent scope.                               |
| `independent` | Child becomes a separate Execution in the same Session document. |

An independent child has:

- Parent Execution identity.
- Root Execution identity.
- Ownership facts.
- Cancellation relationship.
- Child budget.
- Completion handle.

`handoff` is valid only for an Agent target.

A handoff declares:

- Conversation identity.
- Target Agent.
- Ownership.
- Completion behavior.

Completion behavior is:

| Value     | Meaning                                                             |
| --------- | ------------------------------------------------------------------- |
| `restore` | Restore the previous conversation owner after child completion.     |
| `retain`  | Keep the target Agent as conversation owner after child completion. |

### 6.5 Target sets

A target set limits dynamic dispatch.

```yaml
targetSets:
  specialists:
    members:
      - analyst
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

A static target set has explicit members.

A resolver-backed target set has:

- A resolver value.
- A callable interface.
- Optional constraints.

The runtime creates a typed resolver WorkItem at the first call boundary.

The resolver WorkItem persists normalized allowed membership before target selection.

The selected target identity becomes part of the durable step outcome.

A retry, wait, recovery operation, or resumed operation uses the persisted membership and selected target identity.

### 6.6 Provider-managed Tool Artifacts

A provider SDK Tool Artifact is a provider-managed Model capability.

It can appear only in a typed Model execution plan.

It must not become:

- A local Tool WorkItem.
- A synthetic local result.
- A fabricated local success.
- A local retry target.

A Workflow call to a provider-managed Tool Artifact requires a typed Model plan that owns the provider Tool choice. If such a plan is unavailable, admission or binding fails before unrelated external work begins.

## 7. Context, ports, and channels

### 7.1 Explicit context transfer

Only mapped input, context, ports, and copied message values cross a target boundary.

A target does not implicitly receive:

- Whole Session history.
- Workspace files.
- Tool outputs.
- Attachments.
- Agent outputs.
- Channel logs.
- Credentials.
- Native paths.
- App state.

### 7.2 Ports

A port is a typed named boundary between a caller and a target.

A callable port declares:

```yaml
ports:
  progress:
    direction: output
    stream: true
    schema:
      type: object
```

A call binds a target port to a Workflow channel:

```yaml
call:
  target: analyzer
  ports:
    progress:
      channel: analysis-progress
      access: write
```

Port validation checks:

- Port name.
- Port direction.
- Stream capability.
- Channel existence.
- Channel read or write authorization.
- Source schema.
- Destination schema.
- Parent-child ownership.
- Target interface compatibility.

Input ports transfer authorized retained channel values or explicitly mapped values.

Output ports transfer final target values.

Streamed ports transfer ordered incremental values.

A final call output remains separate from a streamed port.

### 7.3 Channels

A channel is an invocation-local append-only durable message log.

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
    readers:
      - review
    writers:
      - draft
```

A channel message has:

- Message identity.
- Sequence.
- Channel identity.
- Kind.
- Data.
- Correlation.
- Deduplication key.
- Creation time.
- Consumption facts.

Channel reads are non-destructive unless a wait explicitly consumes a message.

A consume operation is scoped to a wait or subscription identity. It does not globally remove the message.

### 7.4 Emit

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

An emit can target:

- A local channel.
- A Workflow event.

An emit returns a durable receipt.

A duplicate deduplication key for the same destination returns the existing receipt and does not append another message.

## 8. Structured operations

### 8.1 Operation set

| Operation   | Purpose                                                 |
| ----------- | ------------------------------------------------------- |
| `call`      | Invoke, start, dispatch, or hand off to a typed target. |
| `transform` | Construct a deterministic value.                        |
| `set`       | Atomically update typed state.                          |
| `scope`     | Create a nested block and lifecycle boundary.           |
| `choose`    | Select one conditional block.                           |
| `parallel`  | Run named branches.                                     |
| `foreach`   | Process a captured collection.                          |
| `repeat`    | Repeat a block with carried state.                      |
| `machine`   | Run an explicit state lifecycle.                        |
| `wait`      | Suspend for a typed condition.                          |
| `emit`      | Publish a local channel message or event.               |
| `saga`      | Define a compensation boundary.                         |
| `cancel`    | Request cancellation of child work.                     |
| `return`    | Complete a block or Workflow with an explicit value.    |
| `fail`      | Produce a typed failure.                                |
| `break`     | Exit the nearest Repeat.                                |
| `continue`  | Complete the current Repeat iteration.                  |

### 8.2 Transform

```yaml
- id: prepare
  transform:
    value:
      title:
        $expr: input.title
      enabled:
        $expr: input.documents.filter(d, d.enabled)
    outputSchema:
      type: object
```

A transform is deterministic.

A transform cannot create an external effect.

### 8.3 Scope

```yaml
- id: protected
  scope:
    state:
      schema:
        type: object
      initial: {}
    steps: []
    output: null
```

A scope creates a lifecycle boundary.

A scope can own:

- Local state.
- Subscriptions.
- Catch handlers.
- Finalization.
- Attached children.
- Resource claims.
- Cleanup work.

### 8.4 Choose

```yaml
- id: route
  choose:
    cases:
      - when:
          $expr: input.priority == 'high'
        do:
          steps: []
          output:
            queue: urgent
    default:
      steps: []
      output:
        queue: normal
```

Cases evaluate in declaration order.

The first matching case executes.

No matching case without `default` produces `routing.no_match`.

### 8.5 Parallel

```yaml
parallel:
  completion:
    mode: all
  errors: collect
  remaining: wait
  branches:
    legal:
      do:
        steps: []
        output:
          approved: true
    security:
      when:
        $expr: input.requiresSecurityReview
      do:
        steps: []
        output:
          approved: true
```

All branch conditions evaluate from the same entry snapshot.

Only selected branches participate in completion.

Completion modes:

| Mode              | Completion condition                            |
| ----------------- | ----------------------------------------------- |
| `all`             | Every selected branch succeeds.                 |
| `first-completed` | First terminal branch determines the result.    |
| `first-success`   | First successful branch wins.                   |
| `quorum`          | A declared positive number of branches succeed. |

`remaining` is:

| Value    | Meaning                                              |
| -------- | ---------------------------------------------------- |
| `cancel` | Request cancellation of remaining selected branches. |
| `wait`   | Wait for remaining selected branches to settle.      |

For `all`, `errors` is:

| Value       | Meaning                                                       |
| ----------- | ------------------------------------------------------------- |
| `fail-fast` | Request cancellation after a branch failure.                  |
| `collect`   | Wait for all selected branches and return aggregate outcomes. |

Branch outcomes remain durable and visible even after another branch wins.

An uncertain branch remains visible. It is not converted to success, skipped, or cancellation merely because another branch completes the policy.

### 8.6 Foreach

```yaml
foreach:
  items:
    $expr: input.files
  itemSchema:
    type: object
  concurrency: 4
  maxItems: 1000
  completion:
    mode: all
  errors: collect
  remaining: wait
  do:
    steps: []
    output:
      name:
        $expr: item.name
      index:
        $expr: index
```

Foreach:

- Captures the source collection once.
- Validates each item.
- Creates isolated item state.
- Preserves source-order outputs.
- Applies effective concurrency and resource limits.
- Supports all Parallel completion modes.
- Supports cancellation.
- Supports explicit parent-state merge.
- Returns an empty ordered item list for an empty `all` collection.

A Foreach item identity includes its captured source index.

### 8.7 Repeat

```yaml
repeat:
  state:
    schema:
      type: object
    initial:
      attempts: 0
  until:
    $expr: state.approved
  maxIterations: 3
  onExhausted: fail
  timeout: PT10M
  do:
    steps: []
    output: null
```

Exactly one repeat control is selected:

| Control                     | Meaning                          |
| --------------------------- | -------------------------------- |
| `count`                     | Fixed number of iterations.      |
| `while`                     | Condition before each iteration. |
| `until`                     | Condition after each iteration.  |
| `lifetime: until-cancelled` | Ongoing lifecycle.               |

Repeat returns:

```text
state
result
iterations
exhausted
```

A body-local state initializes once per iteration.

A carried Repeat state initializes once for the Repeat.

`break` exits the nearest Repeat.

`continue` completes the current iteration of the nearest Repeat.

### 8.8 Durable Fork and Join

A child scheduling boundary uses a typed Fork WorkItem.

A child settlement aggregation boundary uses a typed Join WorkItem.

A Fork WorkItem persists:

- Parent ProgramFrame identity.
- Child ProgramFrame intents.
- Child identities.
- Child lifetimes.
- Child ownership facts.
- Child deadlines.
- Child program snapshots.

A Join WorkItem persists:

- Parent ProgramFrame identity.
- Child ProgramFrame identity or identities.
- Join policy.
- Child terminal facts.

A parent interpreter receives child settlement only after the Join WorkItem confirms the child terminal state.

## 9. State machines

A state machine models cyclic and event-driven lifecycle behavior.

```yaml
machine:
  initial: open
  state:
    schema:
      type: object
    initial:
      version: 1
  listen:
    updates:
      event: case-updated
      correlation:
        caseId:
          $expr: input.caseId
  states:
    open:
      entry:
        steps: []
        output: null
      transitions:
        - event: updates
          when:
            $expr: event.data.action == 'close'
          to: closed
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

A machine persists:

- Current state.
- Carried state.
- Local state ownership.
- Entry time.
- Transition generation.
- Active subscriptions.
- Active event waits.
- Active timer waits.
- Selected transition.
- Current event.
- Child block state.

Machine behavior is serialized within the owning ProgramFrame.

State entry behavior completes before transition selection.

Transition selection order is:

1. Immediate transitions.
2. Event transitions.
3. Timer transitions.

For one accepted event, the first matching transition wins.

A transition performs:

1. Current state exit block.
2. Transition block.
3. Target state entry block.
4. Target state transition selection.

A state timer begins on entry and ends on exit.

A final state returns its explicit output.

## 10. Events, waits, and human work

### 10.1 Event contracts

```yaml
events:
  payment-confirmed:
    source: payments
    schema:
      type: object
    correlationSchema:
      type: object
```

An event envelope has:

```text
id
type
source
data
correlation
time
```

Duplicate event identity is `(source, id)`.

An event contract controls:

- Source.
- Data schema.
- Correlation schema.

### 10.2 Subscriptions

```yaml
listen:
  confirmation:
    event: payment-confirmed
    correlation:
      orderId:
        $expr: input.orderId
    filter:
      $expr: event.data.status == 'confirmed'
    retention: P1D
    capacity: 100
    maxBytes: 262144
```

A subscription becomes active when its scope begins, before its steps run.

A subscription stores:

- Event contract identity.
- Correlation facts.
- Filter facts.
- Retention settings.
- Capacity settings.
- Byte settings.
- Accepted event identities.
- Consumption facts.

### 10.3 Wait

```yaml
wait:
  for: event
  subscription: confirmation
  consume: true
```

Supported wait forms:

| `for`        | Required fields                     | Successful output           |
| ------------ | ----------------------------------- | --------------------------- |
| `timer`      | `after` or `at`                     | Timer firing facts.         |
| `event`      | `subscription`                      | Event envelope.             |
| `channel`    | `channel`                           | Channel message and cursor. |
| `completion` | `handle`                            | Child completion output.    |
| `human`      | assignees, request, response schema | Validated human response.   |

A completion handle can identify one:

- WorkItem.
- ProgramFrame.
- Execution.

A completion wait propagates success, failure, cancellation, and uncertainty.

### 10.4 Human wait

```yaml
wait:
  for: human
  assignees:
    users:
      - release-manager-1
    roles:
      - release-manager
  request:
    release:
      $expr: input.release
  responseSchema:
    type: object
    required: [approved]
```

A human wait persists:

- User assignments.
- Role assignments.
- Request value.
- Response schema.
- Response authorization requirements.
- Deadline.
- Resolution identity.
- Typed response.

A response is accepted only when the response actor satisfies the persisted assignment facts.

A business rejection remains response data unless the Workflow explicitly fails.

### 10.5 Timer behavior

Durations use ISO 8601.

Absolute times use RFC 3339.

A timer preserves:

- Scheduling time.
- Intended fire time.
- Actual fire time.
- Enclosing step deadline.

A timer cannot be manually resolved.

A timeout remains separate from successful timer firing.

## 11. Triggers

Triggers start or address Workflow Executions.

A trigger is not a subscription.

A subscription receives an event inside an existing Invocation.

A trigger creates or addresses an Invocation from outside the body.

### 11.1 Event trigger

```yaml
triggers:
  intake:
    event:
      event: request-received
      filter:
        $expr: event.data.enabled
      input:
        request:
          $expr: event.data
      instanceKey:
        $expr: event.correlation.requestId
      behavior: start-or-signal
```

Event trigger behavior:

| Behavior          | Meaning                                                            |
| ----------------- | ------------------------------------------------------------------ |
| `start`           | Start a new Execution.                                             |
| `signal`          | Signal an active matching Execution.                               |
| `start-or-signal` | Signal an active matching Execution or start one when none exists. |

A trigger occurrence persists:

- Trigger identity.
- Event source.
- Event identity.
- Occurrence time.
- Accepted time.
- Mapped invocation input.
- Instance key.
- Selected Execution.
- Queue position.
- State.
- Failure facts.

### 11.2 Schedule trigger

```yaml
triggers:
  hourly:
    schedule:
      calendar: "0 * * * *"
      timeZone: America/New_York
      input:
        reportDate:
          $expr: context.time
      overlap: queue
      missed:
        mode: catch-up
        maxOccurrences: 3
```

A schedule has exactly one of:

| Field      | Meaning                             |
| ---------- | ----------------------------------- |
| `calendar` | Five-field numeric cron expression. |
| `interval` | ISO 8601 interval.                  |

A calendar schedule requires an IANA time zone.

A schedule overlap policy is:

| Policy    | Meaning                                                   |
| --------- | --------------------------------------------------------- |
| `allow`   | Start another Execution.                                  |
| `skip`    | Record and skip while an instance is active.              |
| `queue`   | Persist the occurrence until an active instance settles.  |
| `replace` | Cancel matching active work and start the new occurrence. |

A missed-occurrence policy is:

| Policy     | Meaning                                             |
| ---------- | --------------------------------------------------- |
| `skip`     | Skip missed occurrences.                            |
| `latest`   | Accept only the latest missed occurrence.           |
| `catch-up` | Accept at most `maxOccurrences` missed occurrences. |

Calendar gaps are skipped.

Repeated local times produce one occurrence for that scheduled local time.

UTC schedules avoid local-time ambiguity.

## 12. Effects, retries, reconciliation, and compensation

### 12.1 Effect contract

```yaml
effect:
  kind: keyed
  key:
    $expr: input.requestId + ':publish'
  reconcile:
    steps: []
    output:
      decision: unknown
  compensate:
    steps: []
    output: null
```

Effect kinds:

| Kind             | Meaning                                                    |
| ---------------- | ---------------------------------------------------------- |
| `read`           | No externally visible business mutation.                   |
| `idempotent`     | Repeating has the same business effect.                    |
| `keyed`          | Target deduplicates repeated requests with one stable key. |
| `non-idempotent` | Repeating can create another business effect.              |

A keyed operation has one stable idempotency key per logical WorkItem.

The key remains unchanged through:

- Attempts.
- Retry backoff.
- Durable waits.
- Recovery.
- Reconciliation.
- Resumption.

A new Repeat iteration or Foreach item receives a new logical WorkItem identity and can have a new key.

### 12.2 Effect state

Effect state is distinct from WorkItem state.

| Effect state          | Meaning                                 |
| --------------------- | --------------------------------------- |
| `notApplicable`       | Read-only work.                         |
| `notStarted`          | No external invocation started.         |
| `dispatched`          | External invocation may be in progress. |
| `confirmedApplied`    | Effect is confirmed.                    |
| `confirmedNotApplied` | Non-application is confirmed.           |
| `uncertain`           | Effect may have happened.               |
| `compensating`        | Compensation is in progress.            |
| `compensated`         | Compensation is confirmed.              |
| `compensationFailed`  | Compensation failed or is uncertain.    |

Effect state persists in:

- WorkItems.
- Attempts.
- Receipts.
- Errors.
- Effect records.
- Reconciliation facts.
- Compensation facts.

### 12.3 Retry

A retry policy contains:

```yaml
retry:
  maxAttempts: 3
  match:
    categories: [transient, timeout]
  backoff:
    initial: PT1S
    maximum: PT30S
    multiplier: 2
    jitter: 0.2
  perAttemptTimeout: PT10S
```

Automatic retry requires:

1. A matching retry policy.
2. A retry-safe effect contract.

Read, idempotent, and keyed work can retry according to the policy.

Non-idempotent work must not retry after uncertainty.

Confirmed non-application from reconciliation can enable a retry for a non-idempotent operation.

### 12.4 Reconciliation

A reconciliation block receives typed `operation` facts.

`operation` contains:

- Original WorkItem identity.
- Original attempt identity.
- Original input.
- Selected target facts.
- Idempotency key.
- Receipt.
- Uncertainty error.

A reconciliation result must distinguish:

```json
{
  "decision": "applied",
  "output": {},
  "receipt": {}
}
```

```json
{
  "decision": "notApplied"
}
```

```json
{
  "decision": "unknown",
  "error": {
    "code": "status_unavailable",
    "message": "status service did not confirm the effect"
  }
}
```

Reconciliation behavior:

| Decision     | Runtime behavior                                                  |
| ------------ | ----------------------------------------------------------------- |
| `applied`    | Confirm the original WorkItem and continue with confirmed output. |
| `notApplied` | Mark confirmed non-application and evaluate retry eligibility.    |
| `unknown`    | Keep the original WorkItem uncertain and require recovery review. |

A reconciliation starts through a typed Reconciliation WorkItem.

The Reconciliation WorkItem persists original operation facts before the reconciliation block starts.

### 12.5 Saga

```yaml
saga:
  compensateOn: [failed, cancelled]
  compensationErrors: continue
  concurrency: 1
  do:
    steps: []
    output: {}
```

A Saga defines a business compensation boundary.

A Saga records confirmed compensatable effects in dependency and confirmation order.

A Saga does not compensate:

- Read effects.
- Confirmed non-application.
- Unknown effects.
- Effects without a compensation block.
- Effects outside the Saga boundary.

A Saga must reconcile an uncertain effect before automatic compensation.

### 12.6 Compensation

Compensation runs in reverse effect order while preserving dependency constraints.

A compensation operation receives typed `completed` facts.

`completed` contains:

- Original operation facts.
- Confirmed output.
- Receipt.
- Effect identity.
- Compensation identity.

A Compensation WorkItem persists compensation intent before a compensation child ProgramFrame starts.

Compensation policy:

| Policy     | Meaning                                            |
| ---------- | -------------------------------------------------- |
| `continue` | Record failure and continue eligible compensation. |
| `stop`     | Record failure and stop later compensation.        |

A compensation failure does not replace the primary failure or cancellation.

The primary failure remains primary.

Compensation failures attach as typed causes.

### 12.7 Cleanup allowance

Cleanup allowance is separate from normal work duration.

A normal deadline can stop ordinary work while still allowing bounded:

- Finalization.
- Reconciliation.
- Compensation.
- Target cancellation acknowledgment.
- Resource release.

Cleanup work uses the declared cleanup duration where present.

## 13. Cancellation

Cancellation:

- Stops queued normal work.
- Requests cancellation from active targets.
- Resolves ordinary waits as cancelled.
- Propagates to attached children.
- Propagates to independent children only when `cancelWithParent` is true.
- Runs finalization.
- Starts eligible reconciliation and compensation.
- Retains uncertainty when cancellation races with an external mutation.

Cancellation does not prove that a dispatched mutating target did not apply its effect.

Cancellation remains cancellation through catch and finally behavior.

A catch handler cannot convert cancellation into ordinary success unless the Workflow explicitly returns an allowed result after cancellation cleanup.

## 14. Resources and limits

### 14.1 Resources

```yaml
resources:
  customer-write:
    concurrency: 1
    keyed: true
  vendor-api:
    rate:
      units: 100
      per: PT1M
      burst: 10
```

A resource can define:

- Concurrency.
- Rate.
- Burst.
- Per-key serialization.

### 14.2 Claims

```yaml
claims:
  - resource: customer-write
    units: 1
    key:
      $expr: input.customerId
```

A claim is acquired before its owning WorkItem dispatches.

A claim is released when its owner settles.

A claim can be held by a scope.

### 14.3 Limits

```yaml
limits:
  duration: P1D
  cleanupDuration: PT10M
  stepExecutions: 1000
  callAttempts: 100
  workflowDepth: 16
  concurrency: 10
  modelTokens: 200000
  messageCount: 10000
  messageBytes: 10485760
  eventCount: 10000
  eventBytes: 10485760
  cost:
    amount: 100
    currency: USD
```

A child limit can only narrow an inherited limit.

Attached children share the parent aggregate budget.

Independent children persist an authorized child budget.

## 15. Declaration validity

A valid Workflow declaration has:

- Known fields.
- Valid operation forms.
- Unique visible step identities.
- Valid CEL scope references.
- Valid JSON Schemas.
- Valid target relationships.
- Valid target-set membership.
- Valid callable interfaces.
- Valid port bindings.
- Valid channel permissions.
- Valid machine state names and transitions.
- Valid Repeat and Foreach controls.
- Valid trigger schedules.
- Valid event and correlation contracts.
- Valid resource claims.
- Valid limit inheritance.
- Explicit root output mapping.

Admission rejects:

- Invalid syntax.
- Invalid language semantics.
- Invalid static interface relationships.
- Unsatisfied required Artifact pins.
- Missing registered Service or Activity bindings.
- Missing required credentials.
- Missing current policy approval.
- Missing current external binding availability.

Admission must not accept a valid operation and later ignore it.

## 16. Compiler and runtime boundary

```text
Workflow source v1
        |
        v
Declaration validation
        |
        v
Artifact admission and pinning
        |
        v
Typed runtime-neutral Workflow program
        |
        v
Session Runtime v3 ProgramFrame interpreter
        |
        v
Setup-owned domain binders
```

The declaration layer owns:

- Source syntax.
- Schema validity.
- Static CEL scope.
- Static target relationships.
- Portable declaration validity.

Runtime setup owns:

- Artifact pinning.
- Current dependency verification.
- Provider planning.
- Tool binding.
- MCP discovery.
- Skill resolution.
- Workspace planning.
- Attachment materialization.
- Service binding.
- Activity binding.
- External domain adaptation.

Session Runtime v3 owns:

- ProgramFrames.
- WorkItems.
- Attempts.
- Waits.
- Timers.
- Recovery.
- Idempotency.
- Cancellation.
- Durable effect state.
- Trigger state.
- Control-flow state.
- Persistence-safe semantics.

Canonical JSON values are allowed only at arbitrary JSON boundaries.

Typed structures own:

- IDs.
- Pins.
- Kinds.
- Limits.
- Effects.
- Attempts.
- Waits.
- Receipts.
- Errors.
- Budgets.
- Trigger state.
- Port state.
- Resource claims.
- Control-flow state.

## 17. Functional completion

Workflow Runtime v3 functional completion requires every valid Workflow v1 field to satisfy one of these conditions:

1. It compiles into typed runtime behavior with durable recovery semantics.
2. It is invalid by Workflow v1 language semantics.
3. It requires a currently unavailable required Artifact pin, binding, credential, policy, or external domain capability and fails before unrelated external work begins.

Functional completion does not require:

- Frontend Workflow adoption.
- Session v2 deletion.
- Session Store v2 deletion.
- Offline migration from v2 to v3.
- Removal of legacy code paths.
- Wails transport redesign.

Those are aggregate-stage work after functional backend completion.

## 18. Implementation status

### Phase 1: durable runtime foundation

- [x] Session v3 is the durable root for Executions, ProgramFrames, WorkItems, Attempts, Waits, effects, triggers, and control-flow state.
- [x] Session Store v3 is the only authoritative V3 persistence store.
- [x] Derived index and FTS data are rebuildable.
- [x] Tombstones are durable deletion barriers.
- [x] Command idempotency records persist in the Session document.
- [x] Restore preserves command idempotency records.
- [x] Runtime close cancels tracked timers, dispatch contexts, and owned asynchronous work.

### Phase 2: callable targets and planning

- [x] Tool Artifact targets use typed Tool WorkItems.
- [x] MCP Tool targets use current MCP discovery and persisted MCP binding facts.
- [x] Service targets use registered typed Service bindings.
- [x] Activity targets use registered typed Activity bindings.
- [x] Nested Workflow targets use child Workflow ProgramFrames.
- [x] Agent targets use child Agent-loop ProgramFrames.
- [x] Model targets use child single-model ProgramFrames.
- [x] Static target sets enforce member identity.
- [x] Resolver-backed target sets use typed resolver WorkItems and persist membership.
- [x] Selected target identity persists with the logical Workflow step.
- [x] Provider SDK Tool Artifacts remain provider-managed Model choices.
- [x] Callable interface limits propagate to direct WorkItems and child ProgramFrame boundaries.
- [x] Full typed Agent, Workspace, Model, instruction, Text, Tool, Skill, MCP, attachment, message, channel, and port plans are complete before every Agent-loop and single-model dispatch.
- [x] Full typed Service and Activity execution-plan propagation is complete for every binding kind.

### Phase 3: ports, channels, and parent-child messaging

- [x] Local channel emit, retained channel records, cursors, consume facts, observe behavior, and emit deduplication are typed runtime concepts.
- [x] Completion handles support WorkItems, ProgramFrames, and Executions.
- [x] Parent-child completion waits preserve terminal outcome semantics.
- [ ] Input port plans are compiled and delivered for all callable target kinds.
- [ ] Output port plans are compiled and delivered for all callable target kinds.
- [ ] Streamed port plans are compiled and delivered for all callable target kinds.
- [ ] Port ownership, channel authorization, and schema enforcement are complete at every target boundary.
- [ ] Durable parent-child live port messaging is complete.

### Phase 4: structured Workflow behavior

- [x] Transform, Set, Scope, Choose, Return, Fail, Break, and Continue are typed Workflow operations.
- [x] Parallel supports `all`, `first-completed`, `first-success`, `quorum`, `cancel`, `wait`, `fail-fast`, and `collect`.
- [x] Foreach captures source order, validates items, preserves ordered outcomes, isolates item state, and supports completion policies.
- [x] Repeat supports count, while, until, timeout, carried state, local state, exhaustion, break, continue, and until-cancelled lifecycle.
- [x] Machine initial state, entry, exit, immediate transitions, event transitions, timer transitions, transition blocks, final output, serialized handling, cancellation, and recovery are typed runtime behavior.
- [x] Fork WorkItems persist child scheduling boundaries.
- [x] Join WorkItems persist child aggregation boundaries.
- [ ] Structured control-flow budget interaction is complete.
- [ ] Structured control-flow resource claim interaction is complete.

### Phase 5: effects, reconciliation, and compensation

- [x] Read, idempotent, keyed, and non-idempotent effect contracts are typed.
- [x] Stable keyed idempotency keys persist in WorkItems and Attempts.
- [x] Effect state persists in WorkItems, Attempts, receipts, and errors.
- [x] Uncertain non-idempotent work does not automatically replay.
- [x] Reconciliation WorkItem intent and typed reconciliation result states exist.
- [x] Saga scope identity and confirmed compensatable effect ledger facts exist.
- [x] Compensation WorkItem intent and cleanup state exist.
- [x] Reconciliation child-block execution is complete for all effect-bearing target kinds.
- [x] Applied reconciliation output validation is complete against original target and call contracts.
- [x] Confirmed non-application retry behavior is complete for all retry policies.
- [x] Unknown reconciliation outcomes always route to typed recovery review.
- [x] Reverse dependency-aware compensation scheduling is complete.
- [x] Compensation concurrency limits are complete.
- [x] Compensation `continue` and `stop` policies are complete.
- [x] Primary outcome cause projection includes typed compensation causes.
- [x] Cleanup timer expiration and bounded cleanup cancellation behavior are complete.

### Phase 6: coordination and triggers

- [x] Event waits, channel waits, timer waits, completion waits, and human waits are typed WorkItem behavior.
- [x] Signals use source and identity deduplication.
- [x] Trigger registrations persist inside the existing Session v3 document.
- [x] Event triggers persist accepted occurrence identity, mapped input, instance key, and selected Execution facts.
- [x] Schedule triggers support interval and five-field cron scheduling with IANA time zones.
- [x] Trigger overlap policies are typed.
- [x] Trigger missed-occurrence policies are typed.
- [x] Event retention limits are complete per subscription identity.
- [x] Subscription capacity and byte limits are complete.
- [x] Channel retention and byte limits are complete across all paths.
- [x] Human response actor authorization is complete.
- [x] Trigger registration without an initial explicit Workflow start is complete.
- [x] Trigger occurrence failure projection and operational management commands are complete.

### Phase 7: resources, budgets, and limits

- [x] Child Execution budget facts are typed and durable.
- [x] Child cleanup duration is typed and durable.
- [x] Root Workflow limits compile to typed Execution budgets.
- [x] Step limits compile to typed nested budgets.
- [x] Resource declarations compile to typed resource definitions.
- [x] Resource claims compile to typed WorkItem claim plans.
- [x] Concurrency claims are enforced.
- [x] Rate claims are enforced.
- [x] Keyed serialization claims are enforced.
- [ ] Model token, message, event, cost, and step budget accounting are enforced.
- [ ] Budget exhaustion propagates typed limit errors.

### Phase 8: Session Runtime v3 normal capability

- [x] Direct Model ProgramFrames exist.
- [x] Agent-loop ProgramFrames exist.
- [x] Provider streaming, usage, protocol output, debugger projection, Tool continuation, cancellation, retries, waits, recovery, uncertainty, and command idempotency exist for current V3 paths.
- [x] Agent, Workspace, Model, Text, Tool, Skill, and MCP setup paths have typed ownership boundaries.
- [x] Go Tool, provider SDK Tool, MCP Tool, and Skill Tool have typed domain routes.
- [ ] Go Tool, MCP Tool, and Skill Tool input and streamed port adapters are unavailable in their current narrow domain contracts.
- [x] MCP resources, prompts, Apps, approvals, App Tool calls, App resource reads, and App context staging have typed domain routes.
- [x] Skill sessions and active Skill lifecycle have typed domain routes.
- [x] Attachment admission uses a narrow Attachment Runtime domain port.
- [x] Local file snapshotting is complete for V3 attachment admission.
- [x] Image snapshotting is complete for V3 attachment admission.
- [x] Directory expansion is complete for V3 attachment admission.
- [x] URL normalization is complete for V3 attachment admission.
- [x] Replay-safe attachment materialization is complete for V3 attachment admission.
- [x] Historical attachment modification handling is complete for V3 attachment admission.
- [x] Provider attachment adaptation is complete for V3 attachment admission.
- [ ] Service and Activity runtime binding registration is complete in application composition.
- [ ] V3 transport-neutral command and projection APIs cover all normal Session v2 backend behavior.

### Phase 9: functional completion gate

- [ ] Every valid Workflow v1 source field is audited against typed compilation, language invalidity, or pre-dispatch unavailable-binding failure.
- [ ] No valid Workflow v1 operation remains a permanent unsupported runtime branch.
- [ ] No valid Workflow v1 field is silently ignored.
- [ ] No external mutating WorkItem can replay automatically after uncertainty without confirmed non-application.
- [ ] All normal Session v2 backend capabilities have typed V3 equivalents.
- [ ] Functional backend completion is reached.

### Phase 10: aggregate stage after functional completion

- [ ] Frontend V3 adoption.
- [ ] Frontend transport migration.
- [ ] Offline Session v2 to V3 migration tooling.
- [ ] Session v2 deprecation decision.
- [ ] Session Store v2 deprecation decision.
- [ ] Legacy runtime path removal.
- [ ] Final aggregate cleanup.
