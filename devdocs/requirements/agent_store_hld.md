# Artifact-backed Agent Store

Status: Proposed
Declaration version: `v1`
Agent declaration: enhanced `agentv1`
Agent Store runtime: deferred
Existing legacy preset store: remains unchanged until replacement is complete

- [1. Agent user declaration](#1-agent-user-declaration)
  - [User-facing purpose](#user-facing-purpose)
  - [Full Agent YAML example](#full-agent-yaml-example)
  - [Agent base prompt](#agent-base-prompt)
  - [Agent Markdown behavior](#agent-markdown-behavior)
  - [Prompt validation](#prompt-validation)
- [2. Agent member behavior](#2-agent-member-behavior)
  - [Tool member](#tool-member)
  - [Model member](#model-member)
  - [Skill member](#skill-member)
  - [MCP member](#mcp-member)
  - [Instruction and Context members](#instruction-and-context-members)
  - [Collection, Agent, Team, Loop, and Workflow members](#collection-agent-team-loop-and-workflow-members)
- [3. Agent program structure](#3-agent-program-structure)
  - [Direct Loop program](#direct-loop-program)
  - [Direct Workflow program](#direct-workflow-program)
- [4. Agent resolution](#4-agent-resolution)
  - [Resolution inputs](#resolution-inputs)
  - [Artifact-backed members](#artifact-backed-members)
  - [Tool and Model members](#tool-and-model-members)
    - [Agent result shape](#agent-result-shape)
- [5. Agent Collections](#5-agent-collections)
  - [Agent Collection domain](#agent-collection-domain)
  - [Managed source layout](#managed-source-layout)
  - [User Collection declaration](#user-collection-declaration)
  - [Managed Agent create flow](#managed-agent-create-flow)
  - [Agent Collection behavior](#agent-collection-behavior)
- [6. Built-in Agents](#6-built-in-agents)
  - [Built-in package location](#built-in-package-location)
  - [Built-in Agent Collection](#built-in-agent-collection)
  - [Built-in package validation](#built-in-package-validation)
  - [Hydration admission](#hydration-admission)
- [7. Workspace support](#7-workspace-support)
  - [Workspace capability output](#workspace-capability-output)
  - [Workspace Agent flow](#workspace-agent-flow)
  - [Built-in fallback in Workspace](#built-in-fallback-in-workspace)
  - [Workspace runtime boundary](#workspace-runtime-boundary)
- [8. Agent runtime schema boundary](#8-agent-runtime-schema-boundary)
- [9. Implementation areas](#9-implementation-areas)
- [10. Final Agent decisions](#10-final-agent-decisions)

## 1. Agent user declaration

### User-facing purpose

An Agent declaration defines:

- One Agent-owned base prompt.
- Instructions and context.
- Model selection.
- Tool selection.
- Tool auto-execution intent.
- Skill selection and intended Skill use.
- MCP server membership.
- Collections and nested Agents.
- Loop or Workflow structure.

It does not define:

- Tool invocation arguments.
- Tool user argument instances.
- MCP conversation state.
- MCP selected resources or prompt arguments.
- MCP installation configuration.
- Secrets.
- Execution history.
- Scheduling.
- Retry behavior.
- Cancellation behavior.

### Full Agent YAML example

```yaml
apiVersion: v1
type: agent
name: bug-investigator
displayName: Bug Investigator
version: v1
description: Investigates failures from logs, tests, code, and configuration.

prompt:
  mediaType: text/markdown
  content: |
    Investigate the supplied issue using evidence from repository files,
    Skills, Tools, and MCP servers.

    Identify likely root cause, missing evidence, safe fix direction,
    and verification steps.

members:
  - type: instruction
    name: bug-investigator-rules
    mediaType: text/markdown
    content: |
      Separate observations from assumptions.
      Prefer the smallest safe change.
      Do not modify generated files.

  - type: instruction
    name: repository-rules

  - type: context
    name: repository-architecture

  - type: model
    name: reasoning
    overrides:
      includeSystemPrompt: true

  - type: tool
    name: statpath
    overrides:
      autoExecute: true

  - type: tool
    name: listdirectory
    overrides:
      autoExecute: true

  - type: tool
    name: searchfiles
    overrides:
      autoExecute: true

  - type: tool
    name: readfile
    overrides:
      autoExecute: true

  - type: skill
    name: grounded-local-work
    use:
      mode: instructions

  - type: skill
    name: bug-investigation
    use:
      mode: active

  - type: mcp
    name: github

loop:
  name: bug-investigation-loop
  maxIterations: 12
```

### Agent base prompt

`prompt` is an additional Agent-owned declaration field.

```yaml
prompt:
  mediaType: text/markdown
  content: |
    Review the requested change for correctness, security, and maintainability.
```

It is separate from Instructions.

```text
Agent.prompt
  -> Agent-owned base prompt

Instruction members
  -> composable instruction Artifacts

Agent Markdown body
  -> generated Instruction Artifact
```

The Agent base prompt does not replace current Agent Markdown body behavior.

### Agent Markdown behavior

Current behavior remains:

```text
AGENT.md or *.agent.md Markdown body
  -> generated <agent-name>-instructions Instruction Artifact
  -> included in Agent members
```

The Agent Markdown front matter may additionally declare `prompt`.

```markdown
---
apiVersion: v1
type: agent
name: bug-investigator

prompt:
  mediaType: text/markdown
  content: |
    Investigate failures using evidence.

members:
  - type: model
    name: reasoning
---

Use repository files, logs, and tests as evidence.
```

The result contains both:

```text
Agent.prompt
  -> Investigate failures using evidence.

Generated Instruction
  -> Use repository files, logs, and tests as evidence.
```

No Markdown body content is reclassified as an Agent prompt.

### Prompt validation

```text
AgentPrompt {
  content: string
  mediaType?: string
}
```

| Rule                       | Behavior                             |
| -------------------------- | ------------------------------------ |
| `prompt` omitted           | Agent has no base prompt             |
| `prompt.content` present   | Required non-empty UTF-8 content     |
| `prompt.mediaType` omitted | Consumer default media type behavior |
| `prompt` source locator    | Not supported in v1                  |
| External reusable prompt   | Use an Instruction member            |
| Prompt changes             | Change Agent Definition digest       |
| Prompt Artifact            | No separate Artifact is created      |

## 2. Agent member behavior

### Tool member

```yaml
- type: tool
  name: searchfiles
  overrides:
    autoExecute: true
```

Meaning:

```text
Resolve tool/searchfiles.
For this Agent, auto-execute it when the Tool runtime otherwise permits it.
```

The Tool's own declaration remains unchanged.

```text
Tool.AutoExecReco
  -> Tool recommendation

Agent member overrides.autoExecute
  -> Agent-specific declared override
```

No Agent member field changes:

```text
Tool Artifact.Enabled
Tool Store Tool.IsEnabled
Tool Store Tool.AutoExecReco
Tool implementation
Tool version
Tool bundle membership
```

### Model member

```yaml
- type: model
  name: reasoning
  overrides:
    includeSystemPrompt: true
```

Meaning:

```text
Resolve model/reasoning.
For this Agent, include the resolved model's system prompt.
```

The Agent does not yet declare Model parameter overrides.

```yaml
# Not part of Agent v1.
- type: model
  name: reasoning
  overrides:
    parameters:
      temperature: 0.2
```

### Skill member

```yaml
- type: skill
  name: bug-investigation
  use:
    mode: active
```

```yaml
- type: skill
  name: grounded-local-work
  use:
    mode: instructions
```

| Mode           | Meaning                                                    |
| -------------- | ---------------------------------------------------------- |
| `available`    | Skill is available without an initial activation directive |
| `active`       | Future Agent runtime preloads the Skill as active          |
| `instructions` | Future Agent runtime uses the Skill as instructions        |

The Skill itself remains authoritative for:

```text
insert
arguments
resources
scripts
assets
allowedTools
```

The Agent relation expresses intended use only.

### MCP member

```yaml
- type: mcp
  name: github
```

An Agent MCP member means:

```text
This Agent includes the resolved MCP server capability.
```

It does not mean:

```text
Configure MCP credentials.
Select MCP connection profile.
Modify MCP policy.
Select MCP resources.
Select MCP prompts.
Select MCP tools.
Connect the MCP server.
```

Those values remain MCP installation or runtime values.

### Instruction and Context members

```yaml
- type: instruction
  name: repository-rules
```

```yaml
- type: context
  name: repository-architecture
```

These are ordinary Artifact relationships.

An Agent may contain local instruction or context declarations.

```yaml
- type: instruction
  name: local-rules
  content: |
    Do not modify generated files.
```

The contained Instruction becomes a source-backed subresource Artifact.

### Collection, Agent, Team, Loop, and Workflow members

The Agent may include existing composition Artifacts normally.

```yaml
- type: collection
  name: engineering-capabilities
```

```yaml
- type: agent
  name: security-reviewer
```

```yaml
- type: workflow
  name: security-review-workflow
```

No Agent-specific override is defined for these member types in v1.

## 3. Agent program structure

### Direct Loop program

```yaml
loop:
  name: bug-investigation-loop
  maxIterations: 12
```

The `loop` key identifies the contained or referenced program type.

No `program` wrapper exists.

A loop with only `name` is a reference:

```yaml
loop:
  name: shared-review-loop
```

A Loop with body configuration is a contained Loop declaration.

```yaml
loop:
  name: bug-investigation-loop
  maxIterations: 12
  until:
    pointer: /complete
    schema:
      const: true
```

A Loop directly inside an Agent may omit `body`.

Its body is the containing Agent, matching existing Agent program semantics.

### Direct Workflow program

```yaml
workflow:
  name: change-workflow
```

A contained Workflow program uses direct Workflow fields.

```yaml
workflow:
  name: review-workflow
  start:
    - analyze

  nodes:
    - id: analyze
      type: agent
      name: analyzer

    - id: review
      type: agent
      name: reviewer

  edges:
    - from: analyze
      to: review
```

At most one of these fields is valid:

```text
loop
workflow
```

## 4. Agent resolution

### Resolution inputs

An Agent plan receives:

```text
Agent ArtifactRef
Resolution Root
Optional Workspace context
```

The Agent declaration never stores a Root ID.

The resolved Agent Artifact determines the default Root.

### Artifact-backed members

These member types use the shared Artifact resolver:

```text
instruction
context
skill
mcp
mcp.policy
collection
agent
team
loop
workflow
workspace
```

Resolution order for a mutable user Root:

```text
current Root
  -> protected built-in Root
  -> unavailable
```

Resolution order for a protected built-in Root:

```text
protected built-in Root
  -> unavailable
```

### Tool and Model members

Tool and Model members use the shared named resolution pipeline.

```text
current Root Tool or Model Artifact
  -> protected built-in Root Tool or Model Artifact
  -> user map
  -> unavailable
```

#### Agent result shape

```text
AgentMemberResult {
  Declared member

  Status
  Diagnostic

  ArtifactRef?
  MappedTarget?

  Overrides?
  Use?
}
```

Examples:

```text
type=tool, name=searchfiles
  -> mapped Tool target
  -> overrides.autoExecute=true

type=model, name=reasoning
  -> mapped Model target
  -> overrides.includeSystemPrompt=true

type=skill, name=bug-investigation
  -> built-in Skill ArtifactRef
  -> use.mode=active
```

The Agent Store plan does not invoke:

- Models.
- Tools.
- Skills.
- MCP servers.
- Workflows.
- Loops.
- Other Agents.

## 5. Agent Collections

### Agent Collection domain

```text
Domain name:                 agent
Managed Source storage key:  user-agents
Managed Source display name: User-managed Agents
Baseline Collection name:    agent-baseline
Allowed member type:         agent
```

### Managed source layout

```text
user-agents/
  collection/
    agent-baseline/
      unversioned/
        collection.json

  agent/
    bug-investigator/
      unversioned/
        agent.json
```

### User Collection declaration

```yaml
apiVersion: v1
type: collection
name: engineering-agents
displayName: Engineering Agents
description: User-managed engineering Agent collection.

members:
  - type: agent
    name: bug-investigator
    locator: ../../../agent/bug-investigator/unversioned/agent.json

  - type: agent
    name: code-reviewer
    locator: ../../../agent/code-reviewer/unversioned/agent.json
```

### Managed Agent create flow

```text
Explicit Agent Collection ArtifactRef
  + expected Collection revision
  + Agent v1 declaration
  + initial Artifact.Enabled value
  -> validate selected Agent Collection
  -> add external Agent membership
  -> publish agent.json package
  -> refresh managed Source
  -> verify Agent Artifact
  -> return Agent and Collection views
```

The selected Collection is mandatory.

No fallback Collection exists.

### Agent Collection behavior

```text
Collection member removed
  -> Agent remains available

Agent updated
  -> Collection relationship remains valid

Agent deleted
  -> Collection member remains declared
  -> relationship becomes unavailable

Collection deleted
  -> Agent Artifacts remain available
```

The baseline Collection is:

```text
agent-baseline
```

It is:

- Editable.
- Non-deletable.
- Explicitly selectable.
- Not automatically selected.
- Not automatically active.

## 6. Built-in Agents

### Built-in package location

```text
internal/artifactcontract/builtin/agents/
  software-dev/
    collection.yaml

  product-leadership/
    collection.yaml

  research-analysis/
    collection.yaml
```

Published managed package location:

```text
agent-collection/<collection-name>/unversioned/collection.yaml
```

### Built-in Agent Collection

```yaml
apiVersion: v1
type: collection
name: agent-software-dev
displayName: Software Development Agents
description: Built-in Agents for software development tasks.

members:
  - type: agent
    name: bug-investigator
    displayName: Bug Investigator
    description: Investigates failures using repository evidence.

    prompt:
      mediaType: text/markdown
      content: |
        Investigate the supplied failure using evidence.

    members:
      - type: model
        name: reasoning
        overrides:
          includeSystemPrompt: true

      - type: tool
        name: searchfiles
        overrides:
          autoExecute: true

      - type: skill
        name: bug-investigation
        use:
          mode: active

      - type: skill
        name: grounded-local-work
        use:
          mode: instructions
```

The built-in Collection and Agent reside in the existing shared protected built-in Root.

No separate Agent Root is introduced.

### Built-in package validation

The Agent built-in package validator verifies:

- Collection package directory matches Collection logical name.
- Direct Collection members are contained Agent declarations.
- Contained Agents satisfy enhanced `agentv1` validation.
- Tool names resolve through the built-in Tool resolver.
- Model names resolve through the built-in Model resolver.
- Skill and MCP names resolve in the protected built-in Root.
- Duplicate Agent identities are rejected.
- Agent package changes do not modify Skill or MCP package ownership.

### Hydration admission

The Agent installer joins the existing shared protected Root.

```text
Existing protected Root
  -> Skill installer
  -> MCP installer
  -> Agent installer
```

A missing Agent installer hydration marker must not reset a current Skill and MCP built-in topology.

```text
Current Skill and MCP built-ins
  + Agent installer not present
  -> publish Agent packages
  -> preserve protected Root
  -> preserve Skill and MCP ArtifactRefs
```

## 7. Workspace support

### Workspace capability output

Workspace capability planning adds:

```text
AgentArtifacts
```

```text
WorkspaceCapabilityPlan {
  PromptArtifacts
  SkillArtifacts
  MCPArtifacts
  AgentArtifacts
  Occurrences
  Complete
}
```

### Workspace Agent flow

```text
Workspace
  -> resolved Agent Artifact
  -> shared Root and built-in fallback resolution
  -> shared Tool and Model resolution
  -> Agent plan
```

The Workspace Agent adapter belongs under:

```text
internal/workspace/store/adapter/agent
```

### Built-in fallback in Workspace

A user Workspace may resolve built-in content only through shared resolver fallback.

```text
Workspace user Root
  -> current Root Agent, Skill, MCP, Context, Tool, or Model
  -> protected built-in Root fallback
```

The Workspace does not accept arbitrary external Root references.

A returned built-in ArtifactRef is valid only when produced by capability resolution.

### Workspace runtime boundary

Workspace Agent support initially provides planning only.

It does not:

- Execute an Agent.
- Invoke a model.
- Invoke a Tool.
- Start a Skill session.
- Connect an MCP server.
- Start a Workflow.
- Run a Loop.
- Persist Agent run state.

## 8. Agent runtime schema boundary

The Agent declaration contains durable behavior.

```text
Agent prompt
Tool auto-execution override
Model system prompt inclusion
Skill mode
Artifact membership
```

The runtime contains one-run inputs.

```text
Initial user text
Tool user argument instances
Tool invocation arguments
MCP selected resources
MCP selected prompts
MCP prompt arguments
MCP discovery digests
Conversation state
Execution state
```

When frontend runtime types are needed before a backend Agent runtime exists, they belong in:

```text
internal/agent/runtime/spec
```

Example runtime-only values:

```go
type AgentLaunchInput struct {
  InitialText string `json:"initialText,omitempty"`

  ToolInputs []AgentToolRuntimeInput `json:"toolInputs,omitempty"`

  MCPContext *mcpConversation.MCPConversationContext `json:"mcpContext,omitempty"`
}

type AgentToolRuntimeInput struct {
  ToolRef               toolSpec.ToolRef
  UserArgSchemaInstance jsonutil.JSONRawString
}
```

This package is not:

- An Artifact Store declaration package.
- A source decoder.
- A source-backed Artifact.
- An Agent Store persistence model.
- An Agent runtime service.

## 9. Implementation areas

```text
internal/artifactcontract/declaration/
  declaration.go
  entry.go
  composition.go
  walk.go
  agentv1/
  teamv1/
  workflowv1/
  modelv1/
  toolv1/
  skillv1/
  mcpv1/
  collectionv1/
  workspacev1/

internal/artifactcontract/resolve/
  root_fallback.go
  composition_entry.go
  mapped_reference.go
  model.go
  tool.go

internal/agent/store/
  builtin/
  consumerapi/
  domain/

internal/agent/runtime/spec/
  types.go

internal/workspace/store/adapter/agent/
  adapter.go
```

No initial Agent runtime engine, aggregate, scheduler, or executor is included.

## 10. Final Agent decisions

- Agent declarations remain `apiVersion: v1`.
- Agent member wire format uses direct `type` and `name`.
- Agent member wire format does not use `target`.
- Agent prompt is a first-class Agent field.
- Agent Markdown body remains generated Instruction content.
- Agent prompt is additional to Agent Markdown body and Instruction members.
- Tool `autoExecute` is an Agent member override.
- Model `includeSystemPrompt` is an Agent member override.
- Skill mode is an Agent member `use` property.
- Model parameter overrides are deferred.
- Tool user argument instances are runtime-only.
- MCP runtime context is runtime-only.
- Agent Collections do not override Agent behavior.
- Generic Tool and Model resolution is shared infrastructure.
- User Root resolution falls back to built-in Root.
- Built-in Root resolution does not fall back to user Roots.
- Agent execution remains deferred.
