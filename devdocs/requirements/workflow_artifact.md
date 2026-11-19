# Workflow v1 Declaration Model

## 1. Purpose

Workflow v1 is a typed declaration language for process orchestration.

Agent = reusable reasoning and interaction configuration
Workflow = typed process declaration
Invocation = one execution of a Workflow

An Agent owns reusable reasoning and interaction behavior.
A Workflow owns declaration-level process behavior, target-selection rules, and portable orchestration semantics.

This document specifies the Workflow source Artifact family: declaration
admission, canonical Definition construction, read-only plan resolution,
static dependency closure, and source-backed identity facts.

Execution preparation, runtime interpretation, persistence, scheduling, and
frontend behavior are outside this document. Their status belongs in
`workflow_runtime.md`.

This Artifact HLD records source admission, retrieval, static relationships,
and read-only planning only. It does not describe consumer setup,
consumer execution, or consumer-private representations.

## 2. Version contract

Workflow declarations use:

```yaml
apiVersion: v1
type: workflow
```

`apiVersion: v1` identifies the declaration language.
The declaration language version is separate from the durable ProgramFrame payload version.
The current Workflow ProgramFrame payload is: `artifact.workflow.v1@1`.
The payload remains `artifact.workflow.v1@1` during development. Workflow v1 remains `apiVersion: v1`.
A Workflow source document is stored as an Artifact definition. The standalone source field `apiVersion` is admission metadata and is not stored in the canonical definition body.

## 3. Responsibility boundaries

### 3.1 Workflow responsibilities

A Workflow declaration owns orchestration semantics:

- Ordered work.
- Typed callable orchestration declarations.
- Target-selection rules and selection persistence requirements.
- Structured control flow.
- Typed state updates.
- Parallel and collection execution.
- Repeat behavior.
- State machines.
- Events, channels, ports, waits, and human work.
- Timers and schedules.
- Trigger behavior.
- Retry, reconciliation, compensation, and saga behavior.
- Resource claims and execution budgets.
- Cancellation and cleanup behavior.

Workflow Artifact planning exposes admitted static facts and deterministic
source-backed relationships. This document does not specify how a consumer
lowers, binds, schedules, persists, or executes those facts.

A Workflow does not own:

- Artifact source parsing.
- Artifact storage.
- Current execution Artifact pinning.
- Local enablement decisions for execution.
- Credentials.
- Provider configuration.
- Provider clients.
- Native target request grammars.
- Target-domain invocation semantics.
- Native file access.
- MCP client sessions.
- Frontend state.
- Wails transport behavior.
- Browser state.

### 3.2 Workflow Artifact plan boundary

Workflow Artifact planning owns static source facts. It returns an admitted
Workflow document, callable targets, requirements, source inventory, and a
complete static Artifact dependency closure.

The closure contains only source-backed Artifact identity facts and static
relationships. It contains no direct-capability evidence, execution-private
program, binding, timer, provider request, credential, client, callback, or
native handle.

Workflow does not support direct capabilities for Model, Agent, Tool, MCP,
Skill, Service, or Activity. A direct composition result for one of those
families, including Nested Workflow, is unsupported Workflow input. Planning
returns an unsupported planning error before it creates a usable plan. It must
not publish a direct target, fabricate an Artifact, Source, Definition,
revision, pin, or binding.

Consumers can use the plan as a read-only source-fact projection. Consumer
lowering and execution behavior are not specified in this document.

Artifact Store is a passive source-fact provider. It admits and returns
definitions, relationships, identity facts, and static planning facts. It does
not prepare, schedule, bind, dispatch, recover, or otherwise participate in
Workflow execution.

Consumers can read generic Artifact facts when they need current source
identity. The store does not receive consumer-specific execution requests or
expose consumer-specific execution APIs.

### 3.3 Agent responsibilities

An Agent owns reusable reasoning and interaction configuration.

An Agent can contribute:

- A Model selection.
- Instructions and Text contributions.
- Tool capabilities.
- Skill capabilities.
- MCP capabilities.
- Model system prompt behavior.
- Agent-specific capability requirements.

Workflow source can declare an Agent target. Artifact planning records the
declared static relationship, callable interface, and source identity facts.
Current capability readiness and native invocation behavior are outside this
Artifact planning document.

### 3.4 Workflow Artifact API

Workflow is a source-backed Artifact family. Its current Artifact-facing API is
read-only plan resolution through `workflow.Service.ResolvePlan`.

The Artifact-facing plan contract must provide:

- The admitted Workflow declaration.
- The root Workflow Artifact identity.
- The complete static Artifact dependency closure.
- Callable targets.
- Requirements.
- Source inventory facts.

Current target readiness, runtime Artifact pin conversion, domain binding,
runtime execution, persistence, recovery, and Wails behavior are outside this
Artifact API.

Inline Workflow text remains ordinary declaration data. It is not a direct capability.

Workflow declarations are admitted through ordinary generic Source refresh and
application-selected canonical declaration decoders. Built-in and Workspace
Source configuration are application bootstrap choices. Workflow has no
Workflow-specific source-origin, Source-driver, or package-publication policy.

The completed plan contract returns:

- the admitted Workflow document and current root Artifact identity facts;
- `Inventory.CallableTargets`, containing only declarations that a Workflow
  call can select;
- `Inventory.Requirements`, containing only admission prerequisites;
- source-backed `ResolvedArtifactRef` values for resolved Artifact targets;
- call, context, port, retry, effect, and block planning facts;
- detailed relationship status and issue facts.

Generic Artifact, catalog, Definition, Resource, and enablement APIs provide
the current entity-management operations. No Workflow-specific list, get,
source export, enablement, Plugin, or managed package API is required until a
concrete consumer requires one.

Service and Activity remain typed registered binding prerequisites. They are
not Artifacts and are not direct composition capabilities.

An Artifact-backed requirement receives ordinary composition resolution. A
Service or Activity requirement remains a typed registered-binding
prerequisite. It does not receive a synthetic Artifact resolution.
Requirements never enter `WorkflowProgram.Targets`, target-set selection, or
callable dispatch.

`Plan.Capabilities` remains a recursive composition and diagnostic projection.
Consumers must use the separate callable-target and requirement collections.
They must not rebuild callable target lookup by walking this diagnostic
projection. It is not executable Workflow target data.

`Inventory` source paths are the planning-owned canonical declaration paths for
blocks, calls, ports, retries, and effects. The root block is `root`; a root
step is `root/steps/<step-id>`. The Artifact plan does not define runtime
frame identities or runtime delivery behavior.

Workflow plan resolution is valid for an available Workflow Artifact regardless
of consumer-local enablement state. A consumer can apply its own enablement,
binding, credential, policy, and domain-readiness rules after it receives the
read-only plan. Those rules are not Artifact planning behavior.

A direct composition result for a Workflow-relevant target family is rejected
as unsupported. Workflow planning must not fabricate an Artifact, Source,
Definition, revision, local-state record, lifecycle record, or direct-target
runtime representation for it.

## 4. Architecture

```mermaid
flowchart TD
    A[Workflow source v1] --> B[Workflow contract admission]
    B --> C[Generic Artifact admission]
    C --> D[Read-only Workflow Artifact plan]
    D --> E[Static Artifact closure and source inventory]
    E --> F[External consumer boundary]
```

| Layer                      | Primary package area                               | Responsibilities                                                                                                                           |
| -------------------------- | -------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------ |
| Declaration language       | `internal/llmartifactory-go/workflow/contract/v1/` | Source grammar, source schema, static validation, JSON Schema validity, CEL scope validity, target and requirement declaration validity    |
| Workflow planning          | `internal/llmartifactory-go/workflow/`             | Read-only Artifact-backed relationship resolution, callable-target and requirement inventory, source-backed references, and planning facts |
| Artifact Store             | Generic Artifact Store packages                    | Passive Definition storage, identity, relationships, admitted source facts, and read-only retrieval                                        |
| External consumer boundary | Outside this document                              | Consumer behavior is intentionally not specified here                                                                                      |

The Artifact Store has no execution role. It does not parse consumer intent,
prepare bindings, schedule work, or retain consumer-private state.

The read-only plan and closure are the complete Artifact-facing output. This
document makes no assertion about a consumer's runtime representation.

### 4.1 Artifact interpretation boundary

This document defines only Artifact-side interpretation and read-only planning ownership.

| Artifact concern                                                            | Owner                 | Artifact-facing result                                            |
| --------------------------------------------------------------------------- | --------------------- | ----------------------------------------------------------------- |
| Artifact envelope, Definition identity, and persistence identity            | Artifact Store        | Admitted immutable source facts                                   |
| Workflow grammar, source schema, static CEL scope, and source relationships | Workflow contract v1  | Typed admitted Workflow declaration                               |
| Artifact-backed target and requirement resolution                           | Workflow planning     | Read-only plan, inventory, closure, and unsupported-direct issues |
| Consumer execution, binding, persistence, transport, and scheduling         | Outside this document | No Artifact HLD behavior is specified                             |

Artifact planning ends when it returns admitted source facts. Consumer-local
representations and consumer behavior are not part of Artifact planning.

LLM declaration-family registration is also split by ownership:

- `internal/llmartifactory-go/registration` assembles family interpretations,
  schema codecs, canonical declaration decoders, source-format decoders, and
  locator factories.
- `internal/artifactsetup/registration` supplies application topology choices
  such as candidate filenames and package document layouts.

Generic schema type-overlap checks are domain-neutral JSON utility behavior.
They have one implementation in `internal/jsonutil`, used by declaration and
consumer boundaries without giving Artifact Store an execution role.

## 5. Root declaration

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

| Field          |                 Required | Meaning                                             |
| -------------- | -----------------------: | --------------------------------------------------- |
| `apiVersion`   | Yes in standalone source | Must be `v1`.                                       |
| `type`         |                      Yes | Must be `workflow`.                                 |
| `name`         |                      Yes | Portable Workflow name.                             |
| `inputSchema`  |                      Yes | JSON Schema for invocation input.                   |
| `outputSchema` |                      Yes | JSON Schema for successful root output.             |
| `schemas`      |                       No | Reusable JSON Schema definitions.                   |
| `targets`      |                       No | Named callable targets.                             |
| `targetSets`   |                       No | Static or resolver-backed target collections.       |
| `events`       |                       No | Typed external event contracts.                     |
| `channels`     |                       No | Invocation-local durable message logs.              |
| `ports`        |                       No | Published typed port bindings.                      |
| `triggers`     |                       No | Event and schedule entry points.                    |
| `requirements` |                       No | Named Artifact or registered-binding prerequisites. |
| `resources`    |                       No | Named claimable resource definitions.               |
| `limits`       |                       No | Invocation limits and cleanup allowance.            |
| `state`        |                       No | Root typed state declaration.                       |
| `listen`       |                       No | Root subscriptions active before root work begins.  |
| `steps`        |                      Yes | Ordered root steps.                                 |
| `output`       |                      Yes | Explicit successful root output mapping.            |
| `catch`        |                       No | Ordered root failure handlers.                      |
| `finally`      |                       No | Root finalization block.                            |

The root Workflow has no implicit output. The final successful value is the root `output` value or an explicit Workflow-level `return`.

## 6. Blocks, identities, and outcomes

### 6.1 Blocks

A block contains:

- Optional local state.
- Optional subscriptions.
- Ordered steps.
- Explicit output.
- Optional ordered catch handlers.
- Optional finalization.

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

A nested block can omit `output` only when its grammar permits omission. Its default output is `null`.

The root output is always explicit.

### 6.2 Step identity

A step identity is unique within its block.

A nested step identity cannot shadow an enclosing visible step identity.

A logical operation identity includes:

- Workflow definition identity.
- ProgramFrame path.
- Step identity.
- Repeat iteration identity.
- Foreach item index.
- Target-set selection identity.
- Retry attempt sequence.

A retry retains one logical operation identity.

A Repeat iteration and a Foreach item create new logical operation identities.

### 6.3 Terminal step outcomes

A Workflow step has one durable terminal outcome:

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

An uncertain outcome means an external effect can have happened but the runtime has no durable confirmation.

## 7. Values, schemas, and expressions

### 7.1 Canonical JSON values

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

A literal directive preserves expression-looking data:

```yaml
$literal:
  $expr: this-is-data
```

Ordinary strings are literal strings.

### 7.2 Schema boundaries

Workflow uses JSON Schema Draft 2020-12 at these boundaries:

- Invocation input.
- Invocation output.
- State initialization.
- State updates.
- Target input.
- Target output.
- Target error.
- Target receipt.
- Event data.
- Event correlation.
- Channel message data.
- Human response.
- Foreach item.
- Step output when `outputSchema` exists.
- Port payloads.
- Trigger input.

Schema validity and business success are separate concepts.

A business decline can be valid output data.

### 7.3 Expression scope

Expressions can use these variables when the controlling construct provides them:

| Variable    | Meaning                                                       |
| ----------- | ------------------------------------------------------------- |
| `input`     | Immutable invocation input.                                   |
| `state`     | Current state of the nearest state-owning scope.              |
| `steps`     | Completed step outcomes visible in the current scope chain.   |
| `item`      | Current Foreach item.                                         |
| `index`     | Current Foreach or Repeat index.                              |
| `result`    | Controlled-block result where defined.                        |
| `event`     | Current matched event, trigger event, or machine event.       |
| `messages`  | Authorized channel snapshot at the evaluation boundary.       |
| `error`     | Error visible to a catch handler.                             |
| `outcome`   | Scope outcome visible to finalization.                        |
| `completed` | Confirmed operation facts visible to compensation.            |
| `operation` | Original operation facts visible to reconciliation.           |
| `context`   | Invocation metadata and logical evaluation time.              |
| `ports`     | Authorized port input values for the current target boundary. |

Expressions are pure.

Expressions do not:

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

External behavior uses a typed callable target or a Workflow operation.

### 7.4 Durable evaluation facts

The runtime persists evaluation facts that can affect a resumed expression:

- Logical `context.time`.
- Authorized retained channel snapshots.
- Current invocation input.
- Current scope state.
- Visible completed step outcomes.
- Active subscription facts.
- Current event, result, error, operation, completed-operation, and outcome values where applicable.
- Port input snapshots and cursors.

The runtime does not persist compiled CEL programs or compiled JSON Schema programs.

## 8. State

### 8.1 State declaration

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

A state declaration creates a state-owning scope.

A nested block with local `state` initializes and owns its own state.

A nested block without local `state` reads and updates the closest enclosing state-owning scope.

### 8.2 Atomic update

```yaml
- id: record
  set:
    values:
      attempts:
        $expr: state.attempts + 1
    remove:
      - obsoleteField
```

All assignment expressions observe the same state snapshot.

The complete result validates against the owning state schema.

`set` returns the resulting state.

### 8.3 Concurrent state isolation

Parallel branches and Foreach items receive isolated state snapshots.

Their state cannot race with parent state.

A parent state changes only through an explicit deterministic merge after child outcomes are available.

The merge operation is represented by a typed StateUpdate WorkItem. It uses the captured parent state, selected child outcomes, the declared state schema, and a durable merge result.

## 9. Callable targets

### 9.1 Target kinds

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

Agent and Model target declarations use typed Workflow adapters. Admission
returns a typed pre-dispatch failure when an adapter is unavailable, the
selected Artifact is disabled or stale, required domain readiness is absent,
or the declaration requests unsupported transport.

An MCP Tool target declares a server:

```yaml
targets:
  publish-issue:
    kind: tool
    server:
      name: issue-tracker
    ref: publish_issue
```

### 9.2 Callable interface

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
  limits: {}
  capabilities: []
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
- Limits.

A call-level interface can narrow a target interface. It cannot widen the target interface.

### 9.3 Invocation

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

A call declaration identifies input, context, allowed target set, effect,
retry, deadline, and port intent. Planning exposes their static forms through
the admitted document and Inventory. Execution preparation captures current
target and binding facts. Durable call handling is specified in
`workflow_runtime.md`.

### 9.4 Call modes

| Mode      | Meaning                                            |
| --------- | -------------------------------------------------- |
| `invoke`  | Typed call-return behavior.                        |
| `start`   | Starts child work and returns a completion handle. |
| `handoff` | Transfers conversation ownership to an Agent.      |

`invoke` is the default.

`start` supports:

| Lifetime      | Meaning                                                 |
| ------------- | ------------------------------------------------------- |
| `attached`    | The child belongs to the parent scope.                  |
| `independent` | The child is an Execution in the same Session document. |

An independent child stores:

- Parent Execution identity.
- Root Execution identity.
- Parent ProgramFrame identity.
- Ownership facts.
- Cancellation relationship.
- Authorized child budget.
- Completion handle.

`handoff` is valid only for an Agent target.

A handoff stores:

- Conversation identity.
- Target Agent pin.
- Ownership facts.
- Saved conversation owner.
- Completion behavior.

Completion behavior is:

| Value     | Meaning                                                                 |
| --------- | ----------------------------------------------------------------------- |
| `restore` | Restore the saved owner after child completion.                         |
| `retain`  | Keep the target Agent as the conversation owner after child completion. |

### 9.5 Target sets

A target set restricts dynamic dispatch.

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

Planning records a resolver-backed target set as deferred selection work. It
does not evaluate membership, select a target, or create runtime work.
Durable resolver execution is specified in `workflow_runtime.md`.

### 9.6 Provider-managed Tool Artifacts

A provider SDK Tool Artifact is a provider-managed Model capability.

It appears only in a typed Model execution plan.

It does not become:

- A local Tool WorkItem.
- A synthetic local result.
- A fabricated local success.
- A local retry target.

Artifact planning records the provider-managed target relationship and callable
interface. It does not choose a Model plan, choose a provider Tool, or create
local Tool work.

## 10. Declaration transport contract

Workflow source declares:

- Explicit call input and context forms.
- Callable interface ports and schemas.
- Channel binding and requested access.
- Published Workflow ports.
- Stream intent.

Contract admission validates static declaration compatibility. Planning exposes
immutable transport facts in Inventory. Current target capability, native
request adaptation, port staging, port delivery, stream ordering, and target
settlement are specified in `workflow_runtime.md`.

## 11. Context, ports, and channels

### 11.1 Explicit context transfer

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
- Application state.

### 11.2 Ports

A port is a typed named boundary between caller and target.

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
- Published root Workflow port compatibility for nested Workflow targets.

Input ports transfer authorized retained channel values or explicit mapped values.

Output ports transfer final target values.

A port declaration distinguishes input, final-output, and streamed-output
contracts. Runtime staging, cursor handling, delivery, retention, and stream
semantics are specified in `workflow_runtime.md`.

### 11.3 Channels

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

A channel message contains:

- Message identity.
- Sequence.
- Channel identity.
- Kind.
- Data.
- Correlation.
- Deduplication key.
- Creation time.
- Consumption facts.

Channel reads are non-destructive unless a wait consumes a message.

Consumption is scoped to a wait or subscription identity. It does not globally remove a message.

### 11.4 Emit

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

A duplicate deduplication key for the same destination returns the stored receipt and does not append another message.

## 12. Structured operations

### 12.1 Operation set

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

### 12.2 Transform

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

A transform does not create an external effect.

### 12.3 Scope

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

### 12.4 Choose

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

### 12.5 Parallel

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

| Mode              | Completion condition                             |
| ----------------- | ------------------------------------------------ |
| `all`             | Every selected branch succeeds.                  |
| `first-completed` | The first terminal branch determines the result. |
| `first-success`   | The first successful branch wins.                |
| `quorum`          | A declared positive number of branches succeed.  |

`remaining` behavior:

| Value    | Meaning                                              |
| -------- | ---------------------------------------------------- |
| `cancel` | Request cancellation of remaining selected branches. |
| `wait`   | Wait for remaining selected branches to settle.      |

For `all`, `errors` behavior:

| Value       | Meaning                                                       |
| ----------- | ------------------------------------------------------------- |
| `fail-fast` | Request cancellation after a branch failure.                  |
| `collect`   | Wait for all selected branches and return aggregate outcomes. |

Branch outcomes remain durable after another branch satisfies completion policy.

An uncertain branch remains visible. It is not converted to success, skipped, or cancelled because another branch completes.

### 12.6 Foreach

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
- Preserves source-order outcomes.
- Applies effective concurrency and resource limits.
- Supports Parallel completion modes.
- Supports cancellation.
- Supports explicit parent-state merge.
- Returns an empty ordered item list for an empty `all` collection.

A Foreach item identity includes its captured source index.

### 12.7 Repeat

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

`continue` completes the current Repeat iteration.

### 12.8 Durable Fork and Join

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
- Child ProgramFrame identities.
- Join policy.
- Child terminal facts.

A parent interpreter receives child settlement only after the Join WorkItem confirms the child terminal state.

## 13. State machines

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

State entry completes before transition selection.

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

## 14. Events, waits, and human work

### 14.1 Event contracts

```yaml
events:
  payment-confirmed:
    source: payments
    schema:
      type: object
    correlationSchema:
      type: object
```

An event envelope contains:

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

### 14.2 Subscriptions

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

### 14.3 Wait

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

A completion wait preserves success, failure, cancellation, and uncertainty semantics.

### 14.4 Human wait

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

A response is accepted only when the response actor satisfies persisted assignment facts.

A business rejection remains response data unless the Workflow explicitly fails.

### 14.5 Timer behavior

Durations use ISO 8601.

Absolute times use RFC 3339.

A timer preserves:

- Scheduling time.
- Intended fire time.
- Actual fire time.
- Enclosing step deadline.

A timer cannot be manually resolved.

A timeout remains separate from a successful timer firing.

## 15. Triggers

Triggers create or address Workflow Executions.

A trigger is not a subscription.

A subscription receives an event inside an existing Invocation.

A trigger creates or addresses an Invocation from outside the Workflow body.

### 15.1 Event trigger

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

### 15.2 Schedule trigger

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

Schedule overlap policy:

| Policy    | Meaning                                                   |
| --------- | --------------------------------------------------------- |
| `allow`   | Start another Execution.                                  |
| `skip`    | Record and skip while an instance is active.              |
| `queue`   | Persist the occurrence until an active instance settles.  |
| `replace` | Cancel matching active work and start the new occurrence. |

Missed-occurrence policy:

| Policy     | Meaning                                             |
| ---------- | --------------------------------------------------- |
| `skip`     | Skip missed occurrences.                            |
| `latest`   | Accept only the latest missed occurrence.           |
| `catch-up` | Accept at most `maxOccurrences` missed occurrences. |

Calendar gaps are skipped.

Repeated local times produce one occurrence for one scheduled local time.

UTC schedules avoid local-time ambiguity.

## 16. Effects, retries, reconciliation, compensation, and sagas

### 16.1 Effect contract

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

### 16.2 Effect state

Effect state is distinct from WorkItem state.

| Effect state          | Meaning                                 |
| --------------------- | --------------------------------------- |
| `notApplicable`       | Read-only work.                         |
| `notStarted`          | No external invocation started.         |
| `dispatched`          | External invocation can be in progress. |
| `confirmedApplied`    | Effect is confirmed.                    |
| `confirmedNotApplied` | Non-application is confirmed.           |
| `uncertain`           | Effect can have happened.               |
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

### 16.3 Retry

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

Read, idempotent, and keyed work can retry under the declared policy.

A non-idempotent WorkItem does not retry after uncertainty.

Confirmed non-application from reconciliation can enable retry for a non-idempotent WorkItem.

### 16.4 Reconciliation

A reconciliation block receives typed `operation` facts.

`operation` contains:

- Original WorkItem identity.
- Original Attempt identity.
- Original input.
- Selected target facts.
- Idempotency key.
- Receipt.
- Uncertainty error.

A reconciliation result distinguishes:

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
| `unknown`    | Keep the original WorkItem uncertain and open recovery review.    |

A reconciliation starts through a typed Reconciliation WorkItem.

The Reconciliation WorkItem persists original operation facts before the reconciliation block starts.

### 16.5 Saga

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

An uncertain effect enters automatic compensation only after reconciliation confirms application.

### 16.6 Compensation

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

A compensation failure does not replace primary failure or cancellation.

Primary failure remains primary.

Compensation failures attach as typed causes.

### 16.7 Cleanup allowance

Cleanup allowance is separate from normal work duration.

A normal deadline can stop ordinary work while allowing bounded:

- Finalization.
- Reconciliation.
- Compensation.
- Target cancellation acknowledgement.
- Resource release.

Cleanup work uses declared cleanup duration.

## 17. Cancellation

Cancellation:

- Stops queued normal work.
- Requests cancellation from active targets.
- Resolves ordinary waits as cancelled.
- Propagates to attached children.
- Propagates to independent children when `cancelWithParent` is true.
- Runs finalization.
- Starts eligible reconciliation and compensation.
- Retains uncertainty when cancellation races with external mutation.

Cancellation does not prove that a dispatched mutating target did not apply its effect.

Cancellation remains cancellation through catch and finally behavior.

A catch handler can return an explicit allowed result only after cancellation cleanup.

## 18. Resources and limits

### 18.1 Resources

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

### 18.2 Claims

```yaml
claims:
  - resource: customer-write
    units: 1
    key:
      $expr: input.customerId
```

A claim is acquired before owning WorkItem dispatch.

A claim is released when its owner settles.

A claim can be retained by a scope.

### 18.3 Limits

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

Attached children share parent aggregate budget.

Independent children persist an authorized child budget.

Cost budgets are deferred.

No cost meter, pricing model, currency model, or cost settlement behavior is to be implemented in this cycle.

## 19. Declaration validity

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

Artifact planning resolves static declaration relationships and source-backed
Artifact identity facts represented by the admitted plan and closure.

Current execution pinning, binding availability, provider ownership, policy,
and transport behavior are outside this document.

## 20. Artifact consumer boundary

```text
Workflow source v1
        |
        v
Declaration validation
        |
        v
Generic Artifact admission
        |
        v
Read-only Workflow Artifact plan
        |
        v
External consumer boundary
```

The declaration layer owns:

- Source syntax.
- Schema validity.
- Static CEL scope.
- Static target relationships.
- Portable declaration validity.

Workflow planning owns:

- Read-only target and requirement resolution.
- Callable-target and requirement inventory projection.
- Direct source-backed Artifact reference projection.

Canonical JSON values are used only at arbitrary JSON boundaries.

External consumer representations, semantic bindings, and execution state are outside this document.

## 21. Workflow Artifact completion criteria

Workflow Artifact completion requires every valid Workflow v1 source field to
satisfy one result:

- The field is represented in the typed declaration model and static plan.
- The field is rejected by Workflow v1 language validation.
- The field is represented as an explicit external-consumer handoff requirement.

Workflow Artifact completion includes:

- Typed declaration decoding.
- Canonical definition construction.
- Static CEL scope validation.
- Static target and requirement planning.
- Deterministic source inventory construction.
- Complete static source-backed Artifact closure construction.
- Explicit rejection of unsupported direct composition results.
- One canonical declaration-path model in contract admission and planning.
- No runtime interpretation or runtime-private data.
- No Artifact Store execution role.

## 22. Workflow Artifact implementation status

| Artifact planning task                                    | Status      | Evidence or required result                                                                                                                                              |
| --------------------------------------------------------- | ----------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Decode and validate Workflow v1 declarations              | Implemented | `internal/llmartifactory-go/workflow/contract/v1/`                                                                                                                       |
| Build callable-target and requirement inventory           | Implemented | `internal/llmartifactory-go/workflow/inventory.go`                                                                                                                       |
| Reject direct capability composition for Workflow targets | Implemented | `internal/llmartifactory-go/workflow/closure.go` rejects direct Model, Agent, Tool, MCP, Skill, Service, Activity, and Nested Workflow composition before a plan exists. |
| Publish a deterministic complete static Artifact closure  | Implemented | `internal/llmartifactory-go/workflow/closure.go` publishes sorted nodes, edges, paths, and refresh checks                                                                |
| Include nested Workflow plan nodes in the closure         | Implemented | `ArtifactClosureNode.Workflow` contains admitted nested Workflow document, inventory, and composition facts                                                              |
| Publish one canonical declaration-path model              | Implemented | Contract admission, Inventory, and closure paths retain the canonical `root/...` declaration identity                                                                    |
| Keep consumer execution interpretation outside planning   | Implemented | `internal/llmartifactory-go/workflow/` contains source planning facts and has no consumer execution dependency                                                           |

## 23. Direct-capability and generic-callable-capability cleanup

| Cleanup task                                                                     | Status      | Required result                                                                                                                                                                                                      |
| -------------------------------------------------------------------------------- | ----------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Reject direct composition results for Workflow callable targets and requirements | Implemented | `internal/llmartifactory-go/workflow/closure.go` returns `UnsupportedDirectCapabilityError` for Model, Agent, Tool, MCP, Skill, Service, Activity, and Nested Workflow direct composition.                           |
| Remove Workflow direct-capability closure data                                   | Implemented | `internal/llmartifactory-go/workflow/closure.go` has no `DirectCapabilityEvidence`, direct sorting, direct fingerprint input, or direct closure field. Generic `composition.CapabilityPlan` remains diagnostic only. |
| Preserve Service and Activity binding targets                                    | Implemented | `internal/llmartifactory-go/workflow/contract/v1/validation.go` keeps them as typed non-Artifact declarations and prevents synthetic Artifact composition.                                                           |
| Preserve inline text as declaration data                                         | Implemented | `internal/llmartifactory-go/workflow/contract/v1/workflow-v1.schema.json` retains ordinary `value` grammar. It adds no direct target form.                                                                           |
| Remove generic callable capability runtime semantics                             | Implemented | `internal/llmartifactory-go/workflow/contract/v1/types.go` and `workflow-v1.schema.json` remove `CallableInterface.capabilities`. The closed schema rejects the former field at admission.                           |
