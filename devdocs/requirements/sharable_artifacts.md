# Shareable and Composable AI Artifact Declarations

- [Purpose](#purpose)
- [Why this is needed](#why-this-is-needed)
- [Desired user outcome](#desired-user-outcome)
- [Product principles](#product-principles)
  - [One declaration has one semantic type](#one-declaration-has-one-semantic-type)
  - [Composition is explicit](#composition-is-explicit)
  - [Existing file formats remain useful](#existing-file-formats-remain-useful)
  - [Source material remains source-backed](#source-material-remains-source-backed)
  - [Composition is separate from execution](#composition-is-separate-from-execution)
  - [Artifact names are reusable within independent scopes](#artifact-names-are-reusable-within-independent-scopes)
  - [Physical source identity and semantic identity are different](#physical-source-identity-and-semantic-identity-are-different)
- [Artifact vocabulary](#artifact-vocabulary)
- [Common declaration contract](#common-declaration-contract)
- [Progressive declaration forms](#progressive-declaration-forms)
  - [Symbolic reference](#symbolic-reference)
  - [Located declaration](#located-declaration)
  - [Inline declaration](#inline-declaration)
  - [Named inline declaration](#named-inline-declaration)
- [Locator model](#locator-model)
- [Path pattern model](#path-pattern-model)
- [Artifact schemas](#artifact-schemas)
- [Model](#model)
- [Instruction](#instruction)
- [Context](#context)
- [Tool](#tool)
- [Skill](#skill)
- [MCP](#mcp)
- [Collection](#collection)
- [Agent](#agent)
- [Team](#team)
- [Output matching](#output-matching)
- [Loop](#loop)
- [Workflow](#workflow)
- [Workspace](#workspace)
- [Declaration sources](#declaration-sources)
- [Composition behavior](#composition-behavior)
- [Symbolic references](#symbolic-references)
- [Collections](#collections)
- [Agents and Teams](#agents-and-teams)
- [Loops and Workflows](#loops-and-workflows)
- [Supported physical files](#supported-physical-files)
- [User workflows](#user-workflows)
  - [Workspace workflow](#workspace-workflow)
  - [Skill workflow](#skill-workflow)
  - [MCP workflow](#mcp-workflow)
- [Runtime consumer model](#runtime-consumer-model)
  - [Prompt and Context consumer](#prompt-and-context-consumer)
  - [Skill consumer](#skill-consumer)
  - [MCP consumer](#mcp-consumer)
  - [Workflow consumer](#workflow-consumer)
- [Technical architecture](#technical-architecture)
  - [Resolution scope](#resolution-scope)
  - [Sources](#sources)
  - [Definitions](#definitions)
  - [Artifacts](#artifacts)
  - [Resource verification](#resource-verification)
  - [Resolver behavior](#resolver-behavior)
- [Current feature status](#current-feature-status)

## Purpose

The goal is to provide one coherent declaration system for AI/LLM-related artifacts that can be:

- Authored in repositories/files.
- Shared between repositories and applications.
- Discovered from familiar files and directories.
- Composed into larger capabilities.
- Referenced symbolically by name.
- Loaded from local paths.
- Materialized as verified source-backed resources.
- Consumed by prompt, Skill, MCP, Agent, Team, Workflow, and future runtime systems.

The system should allow a repository to describe its AI capabilities without coupling declaration, discovery, composition, and execution into one subsystem.

- A declaration describes what an artifact is and how it relates to other artifacts.
- A resolver turns declarations into a typed graph.
- A consumer decides how to use that graph at runtime.

Repository files and packages
-> Artifact declarations
-> Resolved composition graph
-> Runtime consumers

## Why this is needed

AI capabilities are commonly distributed across many unrelated formats:

- Repository instructions in `AGENTS.md` and `CLAUDE.md`.
- Documentation files that provide useful context.
- Skills stored in `SKILL.md` directories.
- MCP server configuration in `.mcp.json`.
- Agent Markdown files.
- YAML and JSON configuration files.
- Application-managed packages.
- Reusable bundles of prompts, Skills, MCP servers, and workflows.

Without a common declaration model, these formats cannot be composed consistently.

Typical problems include:

- A Skill can be discovered but cannot be referenced from an Agent.
- A Workspace can load files but cannot express a reusable capability set.
- MCP servers and Skills use unrelated identity models.
- A Workflow cannot refer to an Agent using the same reference mechanism as a Collection.
- Repository instructions are treated differently from inline instructions.
- Source-backed files are loaded without a common freshness or integrity model.
- Runtime consumers need to know too much about physical file layouts.
- Reusable AI capability bundles require custom application-specific formats.

The declaration system solves this by separating:

- Declaration:
  - what the artifact means

- Discovery:
  - where declarations are found

- Resolution:
  - how declarations refer to and compose one another

- Resource access:
  - how source material is verified and opened

- Runtime:
  - how the resolved graph is executed or materialized

## Desired user outcome

A user should be able to place declarations and familiar artifact files in a repository, select a Workspace, and receive a resolved graph ready for consumers.

Example repository:

```text
checkout-service/
  AGENTS.md
  README.md
  .mcp.json

  agents/
    reviewer.agent.yaml
    security-reviewer.agent.md

  collections/
    repository-review.yaml

  workflows/
    change-workflow.yaml

  skills/
    code-review/
      SKILL.md
      references/
      scripts/
      assets/

  workspace.yaml
```

The repository can declare:

```text
instruction/repository-rules
context/readme
skill/code-review
mcp/github
agent/reviewer
agent/security-reviewer
collection/repository-review
workflow/change-workflow
workspace/checkout-service
```

A consumer can then ask for:

```text
workspace/checkout-service
  -> roots
  -> resolved agents, collections, Skills, MCP servers, instructions,
     contexts, workflows, and programs
```

## Product principles

### One declaration has one semantic type

Each declaration has one portable `type`.

```yaml
type: skill
name: code-review
```

There is no secondary portable discriminator such as `kind`, `category`, or `role`.

### Composition is explicit

Collections, Agents, Teams, Loops, Workflows, and Workspaces express relationships through typed entries.

```yaml
type: collection
name: repository-review
members:
  - type: instruction
    name: repository-rules
  - type: skill
    name: code-review
  - type: mcp
    name: github
```

### Existing file formats remain useful

Users should not need to rewrite every repository convention into a new custom format.

```text
AGENTS.md
CLAUDE.md
README.md
llms.txt
SKILL.md
.mcp.json
mcp.json
AGENT.md
*.agent.md
canonical JSON
canonical YAML
```

These physical formats normalize into the same artifact vocabulary.

### Source material remains source-backed

A declaration may point to a physical file, directory, MCP configuration, or package resource.

Consumers should receive verified source material rather than unverified file paths.

### Composition is separate from execution

The declaration system can describe:

```text
Agent
Team
Loop
Workflow
```

It does not itself execute them.

Execution belongs to the consumer that owns the applicable runtime.

### Artifact names are reusable within independent scopes

Two unrelated repositories may both define: `skill/code-review` without conflict.

A conflict exists only when multiple available declarations with the same identity participate in the same resolution scope.

### Physical source identity and semantic identity are different

The system preserves both:

- Physical origin:
  - where a declaration came from

- Semantic identity:
  - what declaration name and type it provides

This makes duplicate declarations inspectable rather than silently selecting one.

## Artifact vocabulary

The core artifact vocabulary is:

```text
model
instruction
context
tool
skill
mcp
collection
agent
team
loop
workflow
workspace
```

Additional artifacts supported are:

- `mcp.policy`: It is used by MCP consumers to compose runtime policy.

| Type          | Purpose                                                                       |
| ------------- | ----------------------------------------------------------------------------- |
| `instruction` | Behavioral, task-oriented, or prompt text                                     |
| `context`     | Repository, documentation, or inline information made available to a consumer |
| `tool`        | A named operation or facility                                                 |
| `model`       | A provider-qualified model declaration and parameters                         |
| `skill`       | A path-backed reusable Skill capability                                       |
| `mcp`         | One MCP server capability                                                     |
| `collection`  | An ordered reusable group of declarations                                     |
| `agent`       | A composed AI Agent declaration                                               |
| `team`        | A multi-Agent composition                                                     |
| `loop`        | Repeated application of an entry                                              |
| `workflow`    | An explicit graph of entry applications                                       |
| `workspace`   | A repository declaration manifest and root-selection document                 |
| `mcp.policy`  | MCP runtime policy declaration                                                |

## Common declaration contract

Every canonical declaration has the same conceptual header.

```text
Header {
  $schema?: string
  apiVersion?: string

  type: CoreType
  name: string
  description?: string
  locator?: Locator
  metadata?: map[string, JSONValue]
}
```

Example:

```yaml
$schema: https://schemas.flexigpt.site/artifact/agent/v1.json
apiVersion: v1

type: agent
name: reviewer
description: Reviews repository changes for correctness and security.
```

| Field         | Meaning                                         |
| ------------- | ----------------------------------------------- |
| `type`        | Concrete semantic artifact type                 |
| `name`        | Portable symbolic identity                      |
| `description` | Human-readable discovery information            |
| `locator`     | Declaration-relative or external source address |
| `metadata`    | Namespaced annotation data                      |
| `apiVersion`  | Portable declaration contract version           |
| `$schema`     | Optional JSON Schema location                   |

Every declaration requires `name`, including nested declarations.

A nested object containing exactly `type` and `name` is a symbolic reference.

A nested object containing `type`, `name`, and a locator or type-specific
fields is a named declaration.

The system does not support anonymous declarations.

## Progressive declaration forms

Each artifact type supports three useful declaration forms.

### Symbolic reference

```yaml
type: skill
name: code-review
```

A nested object containing exactly `type` and `name` is a symbolic reference.

### Located declaration

```yaml
type: skill
name: code-review
locator: ./skills/code-review
```

### Inline declaration

```yaml
type: model
name: reasoning
model: anthropic/claude-sonnet
parameters:
  temperature: 0
```

### Named inline declaration

```yaml
type: model
name: reasoning
model: anthropic/claude-sonnet
parameters:
  temperature: 0
```

Named nested declarations become independently addressable Artifacts when they can be materialized independently. Symbolic references remain references.

## Locator model

A portable locator identifies an artifact declaration source, implementation, package, or external resource.

```text
Locator =
  string
  | PathLocator
  | URLLocator
  | GitLocator
  | PackageLocator
  | CommandLocator
```

- Path locator

```yaml
locator: ./skills/code-review
```

```yaml
locator:
  kind: path
  path: ./skills/code-review
```

- URL locator

```yaml
locator:
  kind: url
  url: https://example.com/agents/reviewer.yaml
  integrity: sha256:...
```

- Git locator

```yaml
locator:
  kind: git
  repository: https://github.com/acme/agent-assets.git
  revision: v2.0.0
  path: skills/code-review
```

- Package locator

```yaml
locator:
  kind: package
  manager: go
  package: github.com/acme/agent-tools
  version: v1.2.0
  path: search
```

- Command locator

```yaml
locator:
  kind: command
  command: grep
```

Relative portable locators are interpreted relative to the declaration file containing the locator.

For example:

```text
collections/repository-review.yaml
  locator: ../skills/code-review
```

resolves relative to:

```text
collections/
```

rather than relative to the process working directory.

## Path pattern model

Context selection and declaration discovery use slash-separated patterns.

```text
*       zero or more characters in one path segment
?       one character in one path segment
**      zero or more path segments
[abc]   one character from a character class
```

Patterns are relative to their declared base.

Exclusions are evaluated after inclusions.

```yaml
include:
  - docs/**/*.md
  - src/**/*
exclude:
  - "**/generated/**"
  - vendor/**
```

The same path-matching behavior is used for:

- Source declaration discovery.
- Workspace declaration scans.
- Context resource selection.

## Artifact schemas

## Model

```text
Model {
  Header

  model?: string
  parameters?: map[string, JSONValue]
}
```

Example:

```yaml
type: model
name: reasoning
model: anthropic/claude-sonnet
parameters:
  temperature: 0
  maxTokens: 12000
```

The declaration system validates that `parameters` contains JSON values. The selected model provider defines parameter semantics.

## Instruction

```text
Instruction {
  Header

  content?: string
  mediaType?: string
}
```

Example source-backed instruction:

```yaml
type: instruction
name: repository-rules
locator: ./AGENTS.md
```

Example inline instruction:

```yaml
type: instruction
name: review-rules
mediaType: text/markdown
content: |
  Do not modify generated files.
  Prefer small, reviewable changes.
```

## Context

```text
Context {
  Header

  content?: string
  mediaType?: string

  include?: string[]
  exclude?: string[]
}
```

Example repository context:

```yaml
type: context
name: architecture
locator: ./docs
include:
  - "**/*.md"
exclude:
  - "**/generated/**"
```

Example inline context:

```yaml
type: context
name: review-boundary
mediaType: text/plain
content: |
  The review covers checkout and payment packages.
```

## Tool

```text
Tool {
  Header

  inputSchema?: JSONSchema
  outputSchema?: JSONSchema
}
```

Example:

```yaml
type: tool
name: grep
locator:
  kind: command
  command: grep

inputSchema:
  type: object
  properties:
    pattern:
      type: string
  required:
    - pattern

outputSchema:
  type: string
```

## Skill

```text
Skill {
  Header

  license?: string
  allowedTools?: Tool[]
}
```

Example:

```yaml
type: skill
name: code-review
description: Reviews repository changes
locator: ./skills/code-review

allowedTools:
  - type: tool
    name: repository-search
```

A Skill is an atomic path-backed capability.

```text
skills/code-review/
  SKILL.md
  references/
  scripts/
  assets/
```

The `SKILL.md` source file describes the Skill. The containing directory is the verified Skill resource root.

## MCP

```text
MCP {
  Header

  server?: string

  transport?: stdio | streamable-http | sse

  command?: string
  args?: string[]
  env?: map[string, string]

  url?: string
  headers?: map[string, string]

  include?: {
    tools?: string[]
    resources?: string[]
    prompts?: string[]
  }
}
```

Example stdio MCP:

```yaml
type: mcp
name: filesystem
transport: stdio
command: npx
args:
  - -y
  - "@modelcontextprotocol/server-filesystem"
  - .
```

Example HTTP MCP:

```yaml
type: mcp
name: issue-tracker
transport: streamable-http
url: https://mcp.example.com/issues
headers:
  Authorization: "${ISSUE_TRACKER_TOKEN}"
```

Example source-selected MCP:

```yaml
type: mcp
name: github
locator: ./.mcp.json
server: github
```

An absent `include` selects the complete server capability.

A present `include` selects only the listed tools, resources, or prompts.

## Collection

```text
Collection {
  Header

  version?: string
  members?: Entry[]
}
```

A Collection is an ordered reusable group of declarations.

Example:

```yaml
type: collection
name: repository-review
description: Repository review capabilities

members:
  - type: instruction
    name: repository-rules

  - type: context
    name: architecture

  - type: skill
    name: code-review

  - type: mcp
    name: github

  - type: tool
    name: repository-search

  - type: workflow
    name: review-workflow
```

Collection member order is semantic.

Collections can represent:

- Capability bundles.
- Skill bundles.
- MCP bundles.
- Repository-specific capability groups.
- Reusable package exports.
- Mixed artifact groups.

## Agent

```text
Agent {
  Header

  members?: SetMember[]
  program?: Program
}
```

Example:

```yaml
type: agent
name: reviewer
description: Reviews repository changes

members:
  - type: instruction
    name: reviewer-instructions
    content: |
      Review changes for correctness, security, and maintainability.

  - type: instruction
    name: repository-rules

  - type: context
    name: architecture

  - type: model
    name: reasoning

  - type: skill
    name: code-review

  - type: tool
    name: repository-search

  - type: mcp
    name: github

  - type: collection
    name: repository-review

  - type: agent
    name: security-reviewer

program:
  type: loop
  maxIterations: 16
  until:
    pointer: /complete
    schema:
      const: true
```

A minimal Agent remains valid:

```yaml
type: agent
name: reviewer
```

## Team

```text
Team {
  Header

  members?: SetMember[]
  program?: Program
}
```

Example:

```yaml
type: team
name: change-team

members:
  - type: agent
    name: planner

  - type: agent
    name: implementer

  - type: agent
    name: reviewer

  - type: collection
    name: repository-common

program:
  type: workflow
  name: change-workflow
```

A Team provides multi-Agent composition and optional coordination structure.

## Output matching

```text
OutputMatch {
  pointer?: string
  schema: JSONSchema
}
```

Example:

```yaml
pointer: /decision
schema:
  const: changes-requested
```

`pointer` is an RFC 6901 JSON Pointer into a result value.

If no pointer is present, the schema applies to the complete result.

## Loop

```text
Loop {
  Header

  body?: Entry
  maxIterations?: integer
  until?: OutputMatch
}
```

Example:

```yaml
type: loop
name: reviewer-loop

body:
  type: agent
  name: reviewer

maxIterations: 16

until:
  pointer: /complete
  schema:
    const: true
```

A Loop ends when:

- The result matches `until`.
- `maxIterations` is reached.
- The consumer stops execution for its own runtime reason.

A Loop nested under `Agent.program` or `Team.program` may omit `body`. In that form, the containing Agent or Team is the Loop body.

The Loop still requires a name. A body-less program Loop is a named contextual program node because its body is supplied by its containing Agent or Team.

## Workflow

```text
Workflow {
  Header

  start?: string[]
  nodes?: WorkflowNode[]
  edges?: WorkflowEdge[]
}

WorkflowNode {
  id: string
  target: Entry
  join?: all | any
}

WorkflowEdge {
  from: string
  to: string
  match?: OutputMatch
}
```

Example:

```yaml
type: workflow
name: change-workflow

start:
  - plan

nodes:
  - id: plan
    target:
      type: agent
      name: planner

  - id: implement
    join: any
    target:
      type: agent
      name: implementer

  - id: review
    target:
      type: agent
      name: reviewer

edges:
  - from: plan
    to: implement

  - from: implement
    to: review

  - from: review
    to: implement
    match:
      pointer: /decision
      schema:
        const: changes-requested
```

Workflow rules:

- Node IDs are unique.
- Start IDs identify nodes.
- Edge endpoints identify nodes.
- `join` defaults to `all`.
- `join: all` waits for all applicable incoming edges.
- `join: any` activates after any applicable incoming edge.
- Edges without a matcher are eligible after their source completes.
- Edges with a matcher are eligible only when their source result matches.
- Workflow cycles are valid.
- A node may execute more than once.

## Workspace

```text
Workspace {
  Header

  declarations?: DeclarationSource[]
  roots?: Entry[]
}
```

A Workspace describes:

- Where declarations should be discovered.
- Which declarations are the entry points for a selected repository experience.

Example:

```yaml
type: workspace
name: checkout-service

declarations:
  - ./AGENTS.md
  - ./.mcp.json

  - base: .
    include:
      - agents/**/*.agent.yaml
      - agents/**/*.agent.yml
      - collections/**/*.yaml
      - skills/*/SKILL.md
      - workflows/**/*.yaml
    exclude:
      - vendor/**
      - "**/generated/**"

roots:
  - type: agent
    name: reviewer

  - type: collection
    name: repository-review

  - type: workflow
    name: change-workflow
```

## Declaration sources

```text
DeclarationSource =
  Locator
  | Declaration
  | DeclarationScan

DeclarationScan {
  base: Locator
  include: string[]
  exclude?: string[]
}
```

Exact declaration source:

```yaml
declarations:
  - ./agents/reviewer.agent.yaml
  - ./AGENTS.md
  - ./.mcp.json
```

Scanned declaration source:

```yaml
declarations:
  - base: .
    include:
      - agents/**/*.agent.yaml
      - skills/*/SKILL.md
      - collections/**/*.yaml
      - workflows/**/*.yaml
    exclude:
      - vendor/**
```

Inline declaration source:

```yaml
declarations:
  - type: instruction
    name: review-policy
    content: |
      Do not modify generated files.
```

## Composition behavior

## Symbolic references

A symbolic reference resolves by:

```text
(type, name)
```

Example:

```yaml
type: skill
name: code-review
```

A successful symbolic reference requires exactly one available matching declaration in the active resolution scope.

If none exist:

```text
reference unresolved
```

If multiple exist:

```text
identity conflict
```

The system does not silently select a winner based on source order.

## Collections

Collections expand recursively in declared member order.

```text
collection/repository-review
  -> instruction/repository-rules
  -> context/architecture
  -> skill/code-review
  -> mcp/github
```

Nested Collection cycles are invalid.

```text
collection/a
  -> collection/b

collection/b
  -> collection/a
```

## Agents and Teams

Agents and Teams expand their members in declaration order.

```text
agent/reviewer
  -> instructions
  -> contexts
  -> model
  -> Skills
  -> tools
  -> MCP servers
  -> Collections
  -> delegated Agents
  -> optional program
```

A consumer can decide how to use a resolved Agent or Team graph.

## Loops and Workflows

Loops and Workflows are structural declarations.

The Resolver validates and resolves their targets.

The runtime consumer owns:

- Scheduling.
- Invocation.
- Retries.
- Timeouts.
- Cancellation.
- State persistence.
- Output evaluation.
- Result handling.

## Supported physical files

The declaration system supports the following physical inputs.

| Physical input                  | Produced artifact behavior                              |
| ------------------------------- | ------------------------------------------------------- |
| Canonical JSON                  | One top-level declaration and named nested declarations |
| Canonical YAML                  | One top-level declaration and named nested declarations |
| `AGENTS.md`                     | One `instruction`                                       |
| `CLAUDE.md`                     | One `instruction`                                       |
| `README.md`                     | One `context`                                           |
| `llms.txt`                      | One `context`                                           |
| Selected documentation Markdown | One `context` per selected file                         |
| `SKILL.md`                      | One `skill`                                             |
| `.mcp.json`                     | One `mcp` Artifact per configured server                |
| `mcp.json`                      | One `mcp` Artifact per configured server                |
| `AGENT.md`                      | One `agent` and one generated named body Instruction    |
| `*.agent.md`                    | One `agent` and one generated named body Instruction    |
| Skill Collection manifest       | One `collection` plus source-backed Skills              |
| MCP Collection manifest         | One `collection` plus source-backed MCP declarations    |
| Workspace manifest              | One `workspace`                                         |

A single file may emit multiple Artifacts.

An Agent Markdown body Instruction uses `<agent-name>-instructions` as its
generated logical name.

```text
.mcp.json
  -> mcp/github
  -> mcp/filesystem
  -> mcp/issue-tracker
```

A Skill directory remains one Artifact.

```text
skills/code-review/
  SKILL.md
  references/
  scripts/
  assets/
  -> skill/code-review
```

## User workflows

### Workspace workflow

```text
Select local repository directory or Workspace manifest
  -> register repository source
  -> discover declarations
  -> identify Workspace declaration
  -> load Workspace declaration sources
  -> discover additional declarations
  -> resolve Workspace roots
  -> provide resolved graph to consumers
```

A Workspace can use:

- Existing repository files.
- Canonical JSON and YAML declarations.
- Exact declaration paths.
- Declaration scans.
- Inline declarations.
- Collections.
- Agents.
- Teams.
- Skills.
- MCP servers.
- Workflows.

### Skill workflow

```text
SKILL.md or canonical Skill declaration
  -> Skill declaration
  -> resolved Skill
  -> verified SKILL.md
  -> verified Skill directory
  -> Skill runtime registration
```

Supported Skill entry points include:

- A direct Skill directory path.
- A direct `SKILL.md` path.
- Workspace discovery.
- A canonical Skill declaration with a local path locator.
- A managed Skill package.
- A built-in Skill package.
- A Collection member.
- An Agent or Team member.

### MCP workflow

```text
.mcp.json, mcp.json, or canonical MCP declaration
  -> MCP declaration
  -> resolved MCP server
  -> installation-local configuration
  -> environment and secret materialization
  -> runtime configuration
  -> MCP runtime connection
```

Supported MCP forms include:

- Standard multi-server `.mcp.json`.
- Standard multi-server `mcp.json`.
- Inline stdio declarations.
- Inline HTTP declarations.
- Inline SSE declarations.
- A canonical declaration selecting a server from a source file.
- Collection, Agent, Team, and Workspace membership.
- MCP policy composition.
- Installation-local secret references.
- Runtime include filtering for tools, resources, and prompts.

## Runtime consumer model

### Prompt and Context consumer

```text
resolved Instruction or Context
  -> verified source material
  -> ordered prompt contributions
  -> prompt budget and truncation policy
  -> final prompt
```

Instructions and Contexts can originate from:

- Inline declaration content.
- `AGENTS.md`.
- `CLAUDE.md`.
- `README.md`.
- `llms.txt`.
- Selected documentation.
- A source-relative locator.

### Skill consumer

```text
resolved Skill
  -> verified Skill declaration source
  -> verified Skill directory
  -> Agent Skills runtime registration
  -> Skill prompt, resource, and script operations
```

### MCP consumer

```text
resolved MCP declaration
  -> verified declaration source
  -> optional selected MCP source entry
  -> installation-local configuration
  -> secret and environment substitution
  -> runtime server configuration
  -> MCP connection
```

MCP runtime behavior includes:

- Server identity derived from the Artifact identity.
- Runtime grouping derived from the active resolution scope.
- Source and Definition versioning.
- Connection invalidation.
- Tool discovery.
- Resource discovery.
- Prompt discovery.
- Completion support.
- Policy evaluation.
- Approval handling.
- OAuth and client credential flows.
- Secret redaction in runtime errors and process output.

### Workflow consumer

```text
resolved Workflow
  -> resolved node targets
  -> edge conditions
  -> join behavior
  -> output match schemas
  -> scheduler and executor
```

Workflow execution is intentionally owned by a dedicated runtime consumer.

## Technical architecture

The following architecture supports the product goal.

```text
Root
  -> Source
  -> Entry
  -> Decoder
  -> Definition
  -> Artifact
  -> Resource
```

Typed composition occurs above the generic storage layer.

```text
Artifact + Definition
  -> Artifact Resolver
  -> resolved declaration graph
  -> runtime consumer
```

### Resolution scope

A Root defines the active declaration universe for a repository or application domain.

It owns:

- Registered Sources.
- Source discovery configuration.
- Source refresh state.
- Immutable Definitions.
- Source-backed Artifacts.
- Local Artifact state.
- Symbolic identity lookup.

The semantic lookup identity is:

```text
(type, logicalName)
```

Example:

```text
skill/code-review
agent/reviewer
mcp/github
collection/repository-review
workflow/change-workflow
```

### Sources

A Source is a physical content origin.

Current Source kinds are:

```text
fs-directory
embedded-directory
managed-directory
```

A Source keeps physical configuration separate from declaration discovery configuration.

```text
Source.Config
  filesystem path
  embedded provider and root
  managed source storage

Source.Discovery
  exact files
  directory roots
  include patterns
  exclude patterns
  decoder hints
  allowed decoders
  expected content digests
  scan limits
```

### Definitions

A Definition is the immutable normalized semantic result of decoding a source declaration.

```text
Definition {
  Digest
  Kind
  LogicalName
  LogicalVersion
  DisplayName
  Description
  Labels
  Body
  Dependencies
}
```

The Definition body contains the full canonical declaration structure.

The Definition digest is used for:

- Integrity.
- Equality.
- Change detection.
- Immutable persistence.
- Runtime versioning.
- Managed publication verification.

### Artifacts

An Artifact is the local, addressable record for a named source declaration.

```text
Artifact {
  ID
  RootID
  SourceBinding

  Kind
  LogicalName
  LogicalVersion

  ResolvedDefinition
  SourceContentDigest

  State
  Diagnostics

  DisplayName
  Enabled
  Data
}
```

Artifact local state supports:

- Local enablement.
- Local display name.
- Consumer-specific local data.
- MCP installation settings.
- Workspace runtime disablement.
- Other namespaced consumer settings.

### Resource verification

Resource access verifies:

```text
Artifact
  -> Definition
  -> Source binding
  -> Source refresh state
  -> Source generation
  -> declaration content digest
  -> verified source material
```

Consumers do not need to reopen arbitrary repository paths directly.

### Resolver behavior

The Resolver is responsible for:

- Symbolic lookup.
- Duplicate identity detection.
- Local path locator resolution.
- Collection expansion.
- Agent expansion.
- Team expansion.
- Loop resolution.
- Workflow target resolution.
- Workspace root resolution.
- Collection cycle detection.
- Resolution depth limits.
- Resolution node limits.

The Resolver returns a typed graph. It does not execute that graph.

## Current feature status

| Capability                                         | Status                                                        |
| -------------------------------------------------- | ------------------------------------------------------------- |
| Core artifact type vocabulary                      | Available                                                     |
| Canonical JSON declarations                        | Available                                                     |
| Canonical YAML declarations                        | Available                                                     |
| JSON Schema contract validation                    | Available                                                     |
| `type` and `apiVersion` schema dispatch            | Available                                                     |
| Named nested declaration indexing                  | Available                                                     |
| Named inline declarations                          | Available as source-backed nested Artifacts                   |
| Anonymous declarations                             | Not supported                                                 |
| Root-scoped symbolic lookup                        | Available                                                     |
| Duplicate identity detection                       | Available                                                     |
| Collection resolution                              | Available                                                     |
| Agent resolution                                   | Available                                                     |
| Team resolution                                    | Available                                                     |
| Loop resolution                                    | Available                                                     |
| Workflow structure and target resolution           | Available                                                     |
| Workspace resolution                               | Available                                                     |
| Local path declaration locators                    | Available                                                     |
| Filesystem Sources                                 | Available                                                     |
| Embedded Sources                                   | Available                                                     |
| Managed Sources                                    | Available                                                     |
| Source-backed Definition persistence               | Available                                                     |
| Source-backed Artifact persistence                 | Available                                                     |
| Source refresh state                               | Available                                                     |
| Definition digest verification                     | Available                                                     |
| Resource generation verification                   | Available                                                     |
| `AGENTS.md` support                                | Available                                                     |
| `CLAUDE.md` support                                | Available                                                     |
| `README.md` support                                | Available                                                     |
| `llms.txt` support                                 | Available                                                     |
| Documentation Context discovery                    | Available                                                     |
| `SKILL.md` support                                 | Available                                                     |
| Direct Skill directory registration                | Available                                                     |
| Direct `SKILL.md` registration                     | Available                                                     |
| Managed Skill packages                             | Available                                                     |
| Built-in Skill packages                            | Available                                                     |
| `.mcp.json` support                                | Available                                                     |
| `mcp.json` support                                 | Available                                                     |
| Canonical MCP declarations                         | Available                                                     |
| Source-selected MCP declarations                   | Available                                                     |
| MCP policy declarations                            | Available                                                     |
| MCP installation-local data                        | Available                                                     |
| MCP secret references                              | Available                                                     |
| MCP runtime configuration                          | Available                                                     |
| MCP runtime connection management                  | Available                                                     |
| MCP tool, resource, prompt, and completion support | Available                                                     |
| Workspace prompt planning                          | Available for source-backed Instruction and Context Artifacts |
| Workspace Skill planning                           | Available for source-backed Skill Artifacts                   |
| Workspace MCP planning                             | Available for source-backed MCP Artifacts                     |
| Managed declaration publication                    | Available                                                     |
| Managed package removal                            | Available                                                     |
| Plugin manifests                                   | Support deferred                                              |
| Git locators                                       | Support deferred                                              |
| URL locators                                       | Support deferred                                              |
| Package locators                                   | Support deferred                                              |
| Archive and zip Sources                            | Support deferred                                              |
| Tool execution                                     | Owned by a future Tool consumer                               |
| Model execution                                    | Owned by a future model consumer                              |
| Agent execution                                    | Owned by a future Agent runtime                               |
| Team execution                                     | Owned by a future Team runtime                                |
| Loop execution                                     | Owned by a future execution runtime                           |
| Workflow scheduling and execution                  | Owned by a future Workflow runtime                            |
