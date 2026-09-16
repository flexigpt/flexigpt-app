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
  - [Located reference](#located-reference)
  - [Contained declaration](#contained-declaration)
  - [Identity and uniqueness](#identity-and-uniqueness)
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
- [External references](#external-references)
- [Collections](#collections)
- [Agents and Teams](#agents-and-teams)
- [Loops and Workflows](#loops-and-workflows)
- [Workspaces](#workspaces)
- [Supported physical files](#supported-physical-files)
- [Built-in and user-owned Artifacts](#built-in-and-user-owned-artifacts)
- [User workflows](#user-workflows)
  - [Workspace workflow](#workspace-workflow)
  - [Collection workflow](#collection-workflow)
  - [Skill workflow](#skill-workflow)
  - [MCP workflow](#mcp-workflow)
- [Requirements and implementation impact](#requirements-and-implementation-impact)
  - [Requirements](#requirements)
  - [Implementation impact](#implementation-impact)
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

The goal is to provide one coherent declaration system for AI and LLM-related artifacts that can be:

- Authored in repositories and application-managed storage.
- Shared between repositories and applications.
- Discovered from familiar files and directories.
- Composed into larger capabilities.
- Referenced by name or by a specific location.
- Materialized as verified source-backed resources.
- Consumed by prompt, Skill, MCP, Agent, Team, Workflow, and future runtime systems.

The system separates distinct concerns:

- A declaration describes a thing.
- A composition document selects, groups, or declares things.
- A resolver identifies the intended things.
- A runtime decides how to use the resolved things.

```text
Repository files and packages
  -> Artifact declarations
  -> resolved composition graph
  -> runtime consumers
```

Collections, Agents, Teams, Loops, Workflows, and Workspaces are ordinary Artifacts. They are not alternate stores, ownership databases, or lifecycle parents.

Artifact Store persists a Collection declaration as an ordinary source-backed
Artifact, but it does not persist Collection membership as a generic Store
relationship. Membership is declaration content interpreted by the artifact
contract resolver. There must be no collection foreign key, membership table,
cascading deletion rule, or collection-specific Artifact lifecycle behavior in
Artifact Store.

User-facing collection editing is an application-domain concern above Artifact
Store. It edits a managed Collection declaration document.

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
- A Collection can accidentally become a hidden lifecycle layer rather than a reusable group.

The declaration system separates:

- Declaration:
  - what an artifact means.

- Discovery:
  - where declarations are found.

- Resolution:
  - how declarations identify and compose other declarations.

- Resource access:
  - how source material is verified and opened.

- Runtime:
  - how the resolved graph is executed or materialized.

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

A Collection can group independently declared things:

```yaml
type: collection
name: repository-review
description: Repository review capabilities

members:
  - type: instruction
    name: repository-rules

  - type: skill
    name: code-review

  - type: mcp
    name: github
    locator: ../.mcp.json
    server: github
```

From the user's perspective:

```text
Use repository-rules and code-review from the active repository scope.

Use github from this MCP configuration.
```

The locator narrows where the intended thing is found. It does not create another copy of the Skill or MCP.

A composition document can also contain a complete declaration when no separate declaration exists:

```yaml
type: collection
name: local-development

members:
  - type: mcp
    name: local-files
    transport: stdio
    command: npx
    args:
      - -y
      - "@modelcontextprotocol/server-filesystem"
      - .
```

Here, `local-files` is declared in the Collection document itself. There is no separate MCP declaration to find.

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

Every composition-entry position uses the same user-facing choices:

- Use a thing found by identity in the active scope.
- Use a thing found at a stated location.
- Declare a thing completely in the containing document.

A locator does not change a member from a reference into a contained declaration.

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

A declaration may point to a physical file, directory, MCP configuration, package resource, command, or external source.

Consumers receive verified source material rather than arbitrary unverified file paths.

### Composition is separate from execution

The declaration system can describe:

```text
Collection
Agent
Team
Loop
Workflow
Workspace
```

It does not itself execute them.

Execution belongs to the consumer that owns the applicable runtime.

### Artifact names are reusable within independent scopes

Two unrelated repositories may both define:

```text
skill/code-review
mcp/github
collection/repository-review
```

without conflict.

A conflict exists only when multiple available declarations with the same identity participate in the same resolution scope.

### Physical source identity and semantic identity are different

The system preserves both:

- Physical origin:
  - where a declaration occurrence came from.

- Semantic identity:
  - what declaration name and type it provides.

This makes duplicate declarations inspectable rather than silently selecting one.

A locator selects a physical declaration occurrence. It does not change the semantic identity of the selected thing.

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

A composition entry can be an external reference or a contained declaration. The containing field determines that it is a composition entry. No separate `ref` wrapper is required.

## Progressive declaration forms

Composition entries support three forms.

### Symbolic reference

```yaml
type: skill
name: code-review
```

This means:

```text
Use skill/code-review.
Find one available matching declaration in the active Root.
```

The entry identifies a thing by semantic identity only.

### Located reference

```yaml
type: skill
name: code-review
locator: ./skills/code-review
```

This means:

```text
Use skill/code-review.
Find that Skill at ./skills/code-review.
```

A located reference is the same kind of relationship as a symbolic reference. The locator narrows where the intended declaration is found.

The selected target must confirm the requested `type` and `name`.

A located reference does not:

- Create another declaration.
- Create a Collection-specific copy of the target.
- Override the target's configuration.
- Fall back to another declaration found elsewhere when the stated location is unavailable.

A type-specific selector may be needed when the location contains more than one declaration.

For example:

```yaml
type: mcp
name: github
locator: ./.mcp.json
server: github
```

The `server` selector identifies one independently declared MCP server in a multi-server configuration. It does not make the Collection declare another `mcp/github`.

### Contained declaration

A contained declaration is complete in the containing document.

```yaml
type: mcp
name: local-files
transport: stdio
command: npx
args:
  - -y
  - "@modelcontextprotocol/server-filesystem"
  - .
```

The containing document is the declaration source for this MCP. No other MCP declaration must be found.

A contained declaration:

- Has a required `type` and `name`.
- Contains the type-specific fields required to describe the thing.
- May refer to ordinary resource material when its type permits that.
- Is distinct from an entry that has only identity and an optional locator.
- Remains associated with the document in which it is declared.

A locator alone is not sufficient to make a composition entry a contained declaration.

This avoids an ambiguous interpretation:

```yaml
type: skill
name: code-review
locator: ./skills/code-review
```

This is a located external Skill reference. A standard Skill package at that location declares the Skill itself.

An inline declaration must include actual type-specific declaration data. For example, an inline MCP includes connection fields, and an inline Instruction includes content.

### Identity and uniqueness

The system has both logical identity and declaration-occurrence identity.

The logical identity of a named declaration is:

```text
(type, name)
```

Examples:

```text
skill/code-review
mcp/github
agent/reviewer
collection/repository-review
workflow/change-workflow
```

Logical identity is Root-scoped.

A declaration occurrence identifies where a particular declaration is defined:

```text
Root
  -> Source
    -> declaration location
    -> optional named structural position
```

A contained declaration has a declaration occurrence within its containing document. Its named structural position distinguishes it from another contained declaration with the same logical identity.

The following rules apply:

- Multiple Roots may contain the same `(type, name)` without conflict.
- A Root may contain multiple declaration occurrences with the same `(type, name)`.
- An unlocated symbolic reference requires exactly one available matching target.
- Multiple available matching targets make an unlocated reference ambiguous.
- A located reference selects the declaration occurrence at the stated location.
- The target at that location must confirm the requested type and name.
- Source order is never a tie-breaker.
- A contained declaration is selected directly through its containing document and named structural position.
- A contained declaration can have the same logical identity as another declaration occurrence, but unlocated Root-wide selection remains ambiguous when more than one available target exists.

A version may describe release or compatibility information, but it is not part of the default symbolic lookup identity.

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

- Path locator:

```yaml
locator: ./skills/code-review
```

```yaml
locator:
  kind: path
  path: ./skills/code-review
```

- URL locator:

```yaml
locator:
  kind: url
  url: https://example.com/agents/reviewer.yaml
  integrity: sha256:...
```

- Git locator:

```yaml
locator:
  kind: git
  repository: https://github.com/acme/agent-assets.git
  revision: v2.0.0
  path: skills/code-review
```

- Package locator:

```yaml
locator:
  kind: package
  manager: go
  package: github.com/acme/agent-tools
  version: v1.2.0
  path: search
```

- Command locator:

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

A locator has a context-specific role:

| Context                                       | Meaning                                                                                                                  |
| --------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------ |
| Independent declaration                       | Identifies source material, a package, an implementation, or another declaration source as defined by that artifact type |
| Composition entry with no complete local body | Identifies where the expected external member must be found                                                              |
| Complete contained declaration                | Does not obtain omitted declaration data from another declaration merely because it is present                           |

A locator used in a composition entry is a selection constraint. If the declared location does not provide the expected thing, the member is unavailable.

A command locator is connection or execution information for a Tool or MCP declaration. It is not a location for finding another independently declared Tool or MCP.

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

Example source-backed Instruction:

```yaml
type: instruction
name: repository-rules
locator: ./AGENTS.md
```

Example contained Instruction:

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

Example repository Context:

```yaml
type: context
name: architecture
locator: ./docs
include:
  - "**/*.md"
exclude:
  - "**/generated/**"
```

Example contained Context:

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

Example independent Skill declaration:

```yaml
type: skill
name: code-review
description: Reviews repository changes
locator: ./skills/code-review

allowedTools:
  - type: tool
    name: repository-search
```

A concrete independent Skill requires a Skill package locator. The `SKILL.md` physical-format adapter emits an explicit `./SKILL.md` locator relative to its own declaration occurrence.

A standard Skill is an atomic path-backed capability.

```text
skills/code-review/
  SKILL.md
  references/
  scripts/
  assets/
```

The `SKILL.md` source file declares the Skill. The containing directory is the verified Skill resource root.

When a Skill appears in a composition-entry position with only `type`, `name`, and an optional locator, it is an external reference to that independently declared Skill.

```yaml
- type: skill
  name: code-review
  locator: ./skills/code-review
```

This does not create another `skill/code-review` declaration in the containing Collection, Agent, Team, Workflow, or Workspace.

A contained Skill is possible only when the containing document includes a complete self-contained Skill declaration supported by the Skill contract. A locator pointing to a normal `SKILL.md` package is an external reference.

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

Example MCP selected from a multi-server configuration:

```yaml
type: mcp
name: github
locator: ./.mcp.json
server: github
```

When this form appears as an independent declaration, it identifies connection data for that MCP declaration.

When this form appears in a composition-entry position, it means:

```text
Use the independently declared mcp/github server from .mcp.json.
```

The `server` selector narrows a multi-server configuration. It does not make the containing document declare another `mcp/github`.

An absent `include` selects the complete server capability.

A present `include` selects only the listed tools, resources, or prompts.

An MCP declaration must use exactly one connection-source model:

- An inline executable connection.
- A command locator, which implies `stdio`.
- A non-command locator selecting connection data from another source entry.

A complete inline connection cannot also contain a non-command source locator.

A composition entry cannot override the connection details, include rules, authentication declaration, policy, installation configuration, or runtime state of an independently declared MCP.

Distinct MCP configurations should use distinct logical names.

```text
mcp/github-read
mcp/github-write
```

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
    locator: ./.mcp.json
    server: github

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

Collection membership does not create an ownership hierarchy.

A collection member is either an external edge or a genuinely contained
declaration. A locator alone does not make a member contained. In a composition
position, `type`, `name`, an optional non-command locator, and an optional MCP
`server` selector identify an external target. Common header annotations may
be retained as membership-local presentation data, but they never override the
selected target's metadata, runtime configuration, policy, local state, or
enablement.

A member becomes contained only when it has type-specific declaration body
fields. A command locator is executable Tool or MCP declaration data and is
therefore contained rather than an external source selection.

An external member is not emitted as a source subresource Artifact. A true
contained declaration is emitted at a stable named structural position. This
distinction prevents a Collection from becoming an implicit Artifact owner.

The member forms are:

| Member form                        | Meaning                                                                  |
| ---------------------------------- | ------------------------------------------------------------------------ |
| Exact `type` and `name`            | Use one independently declared target found by Root-wide identity lookup |
| `type`, `name`, and locator        | Use the same expected target at the stated location                      |
| Complete type-specific declaration | Declare a contained Artifact in the Collection document                  |

An external member may be unavailable without invalidating the Collection.

```text
collection/repository-review
  -> skill/code-review: available
  -> mcp/github: unavailable
  -> workflow/review-workflow: available
```

The Collection remains valid if its own declaration remains valid.

Collection membership has these lifecycle rules:

- Adding an external member does not create, copy, enable, disable, or configure the target.
- Removing an external member does not delete the target.
- Deleting an independently declared target does not remove the member from the Collection.
- Restoring a target makes the existing member available again.
- Editing an independently declared target affects every Collection that includes it.
- A contained declaration exists while it remains in the containing Collection document.
- A Collection does not become invalid because an external member is missing, disabled, invalid, or ambiguous.

- A user-managed Collection cannot be deleted while its direct `members` array
  contains any declared member. A missing or ambiguous target still counts as
  a declared member.
- The deletion guard checks only direct members. It does not recursively count
  members of nested Collections.
- Deleting an empty Collection removes only the Collection declaration. It
  never deletes independently declared targets.

Portable source documents may contain true contained declarations. The managed
user Collection API should create external membership entries only, unless a
future specialized authoring flow explicitly requires contained content.

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

An Agent supports the same composition-entry forms as a Collection:

- An unlocated external member.
- A located external member.
- A complete contained declaration.

A minimal Agent remains valid:

```yaml
type: agent
name: reviewer
```

A minimal `type` and `name` object in a composition-entry position remains an external reference. A contained Agent must include its actual composition or program data.

An unavailable Agent member does not invalidate the Agent declaration. It affects readiness for a consumer that intends to use that member.

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

Team members follow the same unlocated-reference, located-reference, and contained-declaration semantics as Collection and Agent members.

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

A Loop body can be an unlocated external target, a located external target, or a contained declaration.

An unavailable body does not make the Loop declaration structurally invalid. It makes that Loop unavailable for execution until a suitable body is available.

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

Workflow node targets follow the same composition-entry semantics as Collection members.

A Workflow remains structurally valid when a target is unavailable. The affected node is unavailable for execution until its target can be resolved.

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

When `declarations` is omitted, the Workspace uses the declaration universe already configured on its containing Source.

When `declarations` is present, including an empty array, it defines the selected Workspace's declaration scope for that Source.

Initial broad scanning used to identify a Workspace is not retained as an implicit source of additional declarations.

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

Workspace roots follow the same composition-entry semantics as Collection members.

A Workspace root can:

- Find an independent declaration by type and name.
- Find an independent declaration at a stated location.
- Contain a complete declaration in the Workspace document.

A Workspace remains valid when a selected root or a nested member is unavailable. The resolved capability plan reports the affected target as unavailable rather than treating the Workspace declaration itself as invalid.

All previously discovered Workspace declaration candidate locations remain in Source discovery when one Workspace installs an explicit declaration scope. This permits switching between Workspaces in one Source.

Only the selected Workspace's non-Workspace declaration universe is active. Concurrently active, independent declaration universes require separate Sources.

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

A Workspace declaration source adds independent declarations to the Workspace declaration universe.

A contained declaration in `Workspace.roots` is different. It is part of Workspace composition rather than a declaration source list.

## Composition behavior

Composition behavior applies consistently to:

- Collection members.
- Agent and Team members.
- Agent and Team programs.
- Loop bodies.
- Workflow node targets.
- Workspace roots.
- Skill `allowedTools`.
- Other fields that accept named declarations.

## External references

An unlocated external reference resolves by:

```text
(type, name)
```

Example:

```yaml
type: skill
name: code-review
```

A successful unlocated reference requires exactly one available matching declaration in the active resolution scope.

If none exist:

```text
member unavailable
```

If multiple exist:

```text
member ambiguous
```

The system does not silently select a winner based on source order.

A located external reference resolves by:

```text
(type, name, locator)
```

Example:

```yaml
type: skill
name: code-review
locator: ./skills/code-review
```

The target at the location must resolve as `skill/code-review`.

If the target is absent, disabled, invalid, incompatible, or has another type or name:

```text
member unavailable
```

A location is not an advisory fallback. A located reference does not silently use another matching declaration from elsewhere in the Root.

Resource absence is distinct from declaration absence:

- Removing an independent declaration makes that Artifact unavailable.
- Removing a package resource needed by an otherwise valid declaration can make that Artifact not ready for runtime use.
- Removing a contained declaration from its containing document means it is no longer declared.
- Missing members do not invalidate the containing Collection, Agent, Team, Loop, Workflow, or Workspace declaration.

## Collections

Collections expand recursively in declared member order.

```text
collection/repository-review
  -> instruction/repository-rules
  -> context/architecture
  -> skill/code-review
  -> mcp/github
```

Collection expansion preserves every declared member, including unavailable and ambiguous members.

A Collection display or capability plan reports member-level status:

```text
collection/repository-review
  -> instruction/repository-rules: available
  -> skill/code-review: available
  -> mcp/github: unavailable
```

A Collection can therefore be used as a best-effort bucket:

- A user can add a member before its target exists.
- A user can remove an unavailable member.
- A target can disappear without invalidating the Collection.
- A restored target becomes available without rewriting the Collection.
- A member can be detached without deleting its target.
- A target can be deleted without removing its declared membership.

Nested Collection cycles are reported during resolution.

```text
collection/a
  -> collection/b

collection/b
  -> collection/a
```

The Collection declarations remain valid. The cyclic expansion is unavailable for the affected member path.

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

Agents and Teams remain first-class composition types. They retain their own member ordering, program structure, and runtime responsibilities.

An unavailable member does not invalidate the Agent or Team declaration. It is reported in the resolved capability plan.

A runtime that requires a complete Agent or Team capability set may reject use of that plan. A catalog, management view, or partial-capability runtime may use the available members while reporting the unavailable ones.

## Loops and Workflows

Loops and Workflows are structural declarations.

The resolver validates and resolves their targets.

The runtime consumer owns:

- Scheduling.
- Invocation.
- Retries.
- Timeouts.
- Cancellation.
- State persistence.
- Output evaluation.
- Result handling.

A Loop or Workflow remains structurally valid when a referenced target is unavailable. The affected body or node is unavailable for execution.

A runtime that requires all Workflow nodes to be executable rejects the execution with target-level explanations. It does not silently substitute another target.

## Workspaces

A Workspace resolves its roots in declaration order.

A Workspace root can expand a Collection, Agent, Team, Loop, Workflow, or another supported target.

The Workspace capability plan preserves:

- The selected roots.
- Resolved available capabilities.
- Unavailable and ambiguous root or nested-member statuses.
- Consumer-specific readiness information.

A Workspace does not automatically activate every Collection, Skill, or MCP in its Root. Its roots and their declared composition determine the selected capability set.

## Supported physical files

The declaration system supports the following physical inputs.

| Physical input                  | Produced artifact behavior                                 |
| ------------------------------- | ---------------------------------------------------------- |
| Canonical JSON                  | One top-level declaration and named contained declarations |
| Canonical YAML                  | One top-level declaration and named contained declarations |
| `AGENTS.md`                     | One `instruction`                                          |
| `CLAUDE.md`                     | One `instruction`                                          |
| `README.md`                     | One `context`                                              |
| `llms.txt`                      | One `context`                                              |
| Selected documentation Markdown | One `context` per selected file                            |
| `SKILL.md`                      | One independent `skill`                                    |
| `.mcp.json`                     | One independent `mcp` Artifact per configured server       |
| `mcp.json`                      | One independent `mcp` Artifact per configured server       |
| `AGENT.md`                      | One `agent` and one generated named body Instruction       |
| `*.agent.md`                    | One `agent` and one generated named body Instruction       |
| Canonical Collection YAML       | One `collection` plus any complete contained declarations  |
| Workspace manifest              | One `workspace`                                            |

A Collection file does not create another Skill or MCP merely because an external member supplies a locator.

A single file may emit multiple Artifacts when it contains complete named declarations.

An Agent Markdown body Instruction uses `<agent-name>-instructions` as its generated logical name.

Contained declarations use stable type and name structural positions, for example:

```text
members/mcp/local-files
members/instruction/reviewer-rules
nodes/review/target/agent/reviewer
```

Reordering an array changes declaration content and Artifact revision, but does not replace the intended identity of a named contained declaration.

## Built-in and user-owned Artifacts

Application built-ins and user-managed content use the same declaration and composition semantics.

The difference is edit authority and distribution:

| Aspect                    | Built-in content       | User-managed content               |
| ------------------------- | ---------------------- | ---------------------------------- |
| Declaration semantics     | Same                   | Same                               |
| Unlocated external member | Same                   | Same                               |
| Located external member   | Same                   | Same                               |
| Contained declaration     | Same                   | Same                               |
| Editing                   | Application-controlled | User-controlled                    |
| Distribution              | Application release    | Repository or managed user storage |

A built-in release can distribute related files together:

```text
software-dev/
  collections/
    software-dev.yaml

  skills/
    code-review/
      SKILL.md
    refactoring-code/
      SKILL.md

  mcps/
    github.yaml
```

The built-in Collection groups independently declared things:

```yaml
type: collection
name: software-dev

members:
  - type: skill
    name: code-review
    locator: ../skills/code-review

  - type: skill
    name: refactoring-code
    locator: ../skills/refactoring-code

  - type: mcp
    name: github
    locator: ../mcps/github.yaml
```

Physical co-distribution does not make the Collection the owner of the Skills or MCP servers. It makes package-relative locations portable.

A built-in Collection can contain a complete declaration when that declaration exists only in the Collection document. This has the same meaning as a contained declaration in a user-managed Collection.

Ordinary Source and Artifact mutation is rejected for the protected built-in Root.

MCP installation state for protected servers is stored in an external local overlay.

User-owned direct and managed Artifacts live in user Roots and retain ordinary local Artifact state.

Symbolic resolution is Root-scoped. A Workspace in a user Root does not automatically import Artifacts from the protected built-in Root. Explicit cross-Root Workspace imports are not currently supported.

Built-in Artifact IDs are not guaranteed to survive a hydration-changing application upgrade because stale protected topology is reset and rebuilt.

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
  -> provide a capability plan to consumers
```

A Workspace can use:

- Existing repository files.
- Canonical JSON and YAML declarations.
- Exact declaration paths.
- Declaration scans.
- Contained declarations.
- Collections.
- Agents.
- Teams.
- Skills.
- MCP servers.
- Workflows.

A Workspace capability plan reports unavailable and ambiguous selected members. It does not remove them from the declared composition.

### Collection workflow

Skill and MCP authoring may provision an editable default Collection for a
Root, but a default is only a domain-specific creation fallback. It is not an
automatically active capability set and is not a Workspace default.

Every user-facing managed Skill or MCP creation command resolves an actual
editable Collection before publishing the independent Artifact. If the caller
does not select one, the API may explicitly resolve a configured domain default
first. If no default is configured, the API must require collection selection
or collection creation.

Workspace does not provision, select, or activate a default Collection.

A mixed Collection remains valid where a user wants to group several artifact types together.

The default Collections are authoring destinations. They are not automatically active in every Workspace, Agent, Team, or runtime session.

A user can:

- Create an empty Collection.
- Add an existing independently declared Skill, MCP, Policy, Instruction, Context, Agent, Team, Loop, Workflow, or other supported Artifact.
- Add a member before its target is available.
- Add a location when a specific declaration occurrence is intended.
- Remove a member without deleting the target.
- See unavailable and ambiguous members without losing the Collection declaration.

The collection deletion workflow is:

- Detach every direct member.
- Re-read the Collection at the expected revision.
- Reject deletion if any direct membership remains.
- Clear or replace a default designation before deleting a designated default.
- Remove only the Collection declaration package after the guard succeeds.

Repository discovery does not silently alter an editable Collection. A discovered Skill or MCP can be attached to a selected Collection or the applicable default Collection.

### Skill workflow

The user-facing Skill creation flow is Collection-oriented.

```text
Select a Skill Collection
  -> use the default editable Skill Collection when none is selected
  -> create an independent Skill declaration and Skill package
  -> add the Skill to the selected Collection
  -> show the Skill and its Collection memberships
```

The user sees one independently declared Skill and one or more Collection memberships.

```text
Detach Skill from Collection
  -> Skill remains independently available

Delete Skill
  -> Collection membership remains declared
  -> Collection reports the member as unavailable
```

The user-facing API does not provide a standalone Skill creation action.

This does not make a newly created Skill Collection-owned. The Skill can later be included in additional Collections.

A Skill discovered from an existing repository remains independently declared. The user may attach it to a selected or default Collection.

### MCP workflow

The user-facing MCP creation flow follows the same model.

```text
Select an MCP Collection
  -> use the default editable MCP Collection when none is selected
  -> create an independent MCP declaration
  -> add the MCP server to the selected Collection
  -> configure installation-local inputs when required
  -> enable and connect only when the user chooses
```

Collection membership does not:

- Configure secrets.
- Configure OAuth.
- Enable runtime use.
- Connect the server.
- Change the effective MCP policy.

Those remain properties of the independently declared MCP server and its user-local configuration.

If an MCP server is deleted, its Collection membership remains visible and unavailable.

## Requirements and implementation impact

### Requirements

The system must support:

- The same composition-entry semantics in Collection, Agent, Team, Loop, Workflow, Workspace, and other declaration-composition fields.
- Unlocated external references by `(type, name)`.
- Located external references by `(type, name, locator)`.
- Complete contained declarations in a composition document.
- A locator that narrows target selection without changing the target's identity.
- A locator-bearing external Skill or MCP member that does not create a Collection-specific copy.
- Explicit ambiguity when multiple unlocated targets match the same logical identity.
- Explicit unavailability when a located target is absent or does not match the expected type and name.
- Member-level availability and readiness reporting.
- Composition declarations that remain valid when external members are unavailable.
- Strict consumer behavior when a runtime requires a complete capability set.
- Partial capability plans when a consumer permits available members to be used.
- Editable user Collections.
- At least one default editable Skill Collection and MCP Collection per user Root.
- User-facing Skill creation that attaches the new Skill to a selected or default Collection.
- User-facing MCP creation that attaches the new MCP server to a selected or default Collection.
- Detach behavior that changes only Collection membership.
- Delete behavior that leaves declared membership in place and changes member availability.
- Identical composition semantics for built-in and user-managed content.

### Implementation impact

The target behavior changes the interpretation of locator-bearing composition entries.

The implementation must therefore provide:

- A clear distinction between external references and complete contained declarations.
- Located external resolution for all supported artifact types, including Skills and MCP servers.
- Verification that a location provides the requested type and name.
- Member-level availability results rather than failure of the containing declaration on the first unavailable member.
- Explicit completeness policy for consumers that require every selected member.
- Independent declaration discovery for Skills, MCP servers, and Policies referenced by built-in Collections.
- User-managed Collection authoring and membership updates.
- Default editable Collection provisioning for user Roots.
- Collection-oriented user creation flows for Skills and MCP servers.
- Migration of existing locator-bearing Collection members that currently create Collection-specific child declarations.
- Preservation of user-local MCP configuration only for the same independently declared MCP identity.
- Continued support for complete contained declarations where a declaration genuinely exists only in its containing document.

The resolver must preserve every declared composition occurrence with a
member-level status. A missing, ambiguous, disabled, invalid, cyclic, or
locator-unresolvable target affects that relationship only. It does not make
the containing Collection declaration invalid.

The existing source-backed declaration model remains appropriate.

No ownership relationship, parent lifecycle relationship, cascading deletion rule, or Collection-specific duplicate Artifact identity is required for external members.

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

A Skill included by several Collections remains one Skill identity. Collection membership determines whether it is selected for a particular experience. It does not create another Skill runtime identity.

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

An MCP included by several Collections retains one MCP identity and one corresponding installation configuration.

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

The runtime decides whether all target nodes are required before execution can begin. It does not silently substitute another target when one node is unavailable.

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

A Root can contain multiple declaration occurrences with the same semantic identity.

An unlocated external reference is valid only when one available target can be selected.

A located external reference uses source location to select the intended declaration occurrence.

A contained declaration has an additional stable structural occurrence within its containing document. That occurrence identifies where the declaration is defined. It is not an ownership edge from the containing Artifact.

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

A contained declaration has a source occurrence within its containing document. Its source occurrence may use a stable named structural position. This identifies where that declaration is written. It does not make the containing Collection, Agent, Team, Workflow, or Workspace its owner.

Protected built-in Artifacts are an exception to ordinary local mutation. Their Source-owned state is application-managed. Consumer settings requiring user mutation use explicit external overlays, as MCP installation settings do.

For user-managed Collections, source-backed collection editing changes only the
Collection declaration body. The generic Artifact Store never infers reverse
membership, deletes a target because it was detached, or deletes memberships
because a target became unavailable. A management view may derive membership by
reading Collection declarations or maintaining a domain-owned cache.

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

Consumers do not reopen arbitrary repository paths directly.

A missing resource affects declaration readiness and runtime materialization. It does not invalidate an otherwise valid containing composition declaration.

### Resolver behavior

The Resolver is responsible for:

- Symbolic lookup.
- Located declaration selection.
- Type and name confirmation for located targets.
- Duplicate identity detection.
- Collection expansion.
- Agent expansion.
- Team expansion.
- Loop resolution.
- Workflow target resolution.
- Workspace root resolution.
- Cycle detection.
- Resolution depth limits.
- Resolution node limits.
- Preservation of ordered member-level availability, unavailability, and ambiguity results.

The Resolver returns a typed graph and member-status information. It does not execute that graph.

## Current feature status

| Capability                                                        | Status                                                                                    |
| ----------------------------------------------------------------- | ----------------------------------------------------------------------------------------- |
| Core artifact type vocabulary                                     | Available                                                                                 |
| Canonical JSON declarations                                       | Available                                                                                 |
| Canonical YAML declarations                                       | Available                                                                                 |
| JSON Schema contract validation                                   | Available                                                                                 |
| `type` and `apiVersion` schema dispatch                           | Available                                                                                 |
| Named nested declaration indexing                                 | Available                                                                                 |
| Named contained declarations                                      | Available as source-backed nested Artifacts                                               |
| Stable named contained-declaration positions                      | Available                                                                                 |
| Unlocated symbolic references                                     | Available                                                                                 |
| Located composite entries as reference edges                      | Available                                                                                 |
| Located Skill external references                                 | Available through composition-entry classification and path resolution                    |
| Located MCP external references                                   | Available through composition-entry classification and path resolution                    |
| Uniform locator semantics across composition types                | Available for all supported declaration kinds                                             |
| Same-Source Skill and MCP resource claim suppression              | Available as current compatibility behavior; not part of target external-member semantics |
| Anonymous declarations                                            | Not supported                                                                             |
| Root-scoped symbolic lookup                                       | Available                                                                                 |
| Duplicate identity detection                                      | Available                                                                                 |
| Member-level unavailable and ambiguous status results             | Available in resolver and Workspace capability plans                                      |
| Best-effort Collection display and expansion                      | Available                                                                                 |
| Editable managed Collections                                      | Available through Skill and MCP collection APIs                                           |
| Collection deletion guard                                         | Available for editable Collections with direct-member protection                          |
| Managed Skill creation in a Collection                            | Available; explicit editable Collection is required                                       |
| Managed MCP creation in a Collection                              | Available; explicit editable Collection is required                                       |
| Managed MCP policy creation in a Collection                       | Available; explicit editable Collection is required                                       |
| Collection resolution                                             | Available with partial relationship behavior                                              |
| Agent resolution                                                  | Available with strict behavior                                                            |
| Team resolution                                                   | Available with strict behavior                                                            |
| Loop resolution                                                   | Available with strict behavior                                                            |
| Workflow structure and target resolution                          | Available with strict behavior                                                            |
| Workspace resolution                                              | Available with strict behavior                                                            |
| Local path declaration locators                                   | Available for currently supported locator targets                                         |
| Filesystem Sources                                                | Available                                                                                 |
| Embedded Sources                                                  | Available                                                                                 |
| Managed Sources                                                   | Available                                                                                 |
| Source-backed Definition persistence                              | Available                                                                                 |
| Source-backed Artifact persistence                                | Available                                                                                 |
| Source refresh state                                              | Available                                                                                 |
| Definition digest verification                                    | Available                                                                                 |
| Resource generation verification                                  | Available                                                                                 |
| `AGENTS.md` support                                               | Available                                                                                 |
| `CLAUDE.md` support                                               | Available                                                                                 |
| `README.md` support                                               | Available                                                                                 |
| `llms.txt` support                                                | Available                                                                                 |
| Documentation Context discovery                                   | Available                                                                                 |
| `SKILL.md` support                                                | Available                                                                                 |
| Direct Skill directory registration                               | Available                                                                                 |
| Direct `SKILL.md` registration                                    | Available                                                                                 |
| Managed Skill packages                                            | Available                                                                                 |
| Built-in Skill packages                                           | Available with independently discovered Skill Artifacts                                   |
| Built-in MCP packages                                             | Available with independently published MCP and policy Artifacts                           |
| Canonical Skill Collection declarations                           | Available as external grouping declarations                                               |
| Individual editing of contained Skills                            | Not supported; managed Skill APIs create independent package Artifacts                    |
| User-managed editable Collections                                 | Available                                                                                 |
| Default editable Skill Collection per user Root                   | Not provided by design; explicit selection is required                                    |
| Default editable MCP Collection per user Root                     | Not provided by design; explicit selection is required                                    |
| Create Skill in Collection                                        | Available                                                                                 |
| Create MCP in Collection                                          | Available                                                                                 |
| Attach an existing Skill or MCP to a Collection through user APIs | Available                                                                                 |
| Detach a member without deletion through user APIs                | Available                                                                                 |
| User-facing standalone Skill creation                             | Removed from managed authoring flow                                                       |
| `.mcp.json` support                                               | Available                                                                                 |
| `mcp.json` support                                                | Available                                                                                 |
| Canonical MCP declarations                                        | Available                                                                                 |
| Source-selected MCP declarations                                  | Available                                                                                 |
| MCP policy declarations                                           | Available                                                                                 |
| MCP installation-local data                                       | Available                                                                                 |
| MCP secret references                                             | Available                                                                                 |
| MCP runtime configuration                                         | Available                                                                                 |
| MCP runtime connection management                                 | Available                                                                                 |
| MCP tool, resource, prompt, and completion support                | Available                                                                                 |
| Workspace prompt planning                                         | Available with partial capability occurrence reporting                                    |
| Workspace Skill planning                                          | Available with partial capability occurrence reporting                                    |
| Workspace MCP planning                                            | Available with partial capability occurrence reporting                                    |
| User managed Source provisioning                                  | Available through the Workspace consumer API                                              |
| Managed declaration publication                                   | Available for managed Collections, Skills, MCPs, and MCP policies                         |
| Managed MCP server publication                                    | Available                                                                                 |
| Managed package removal                                           | Available                                                                                 |
| Cross-Root Workspace imports                                      | Not supported                                                                             |
| Automatic built-in visibility in user Workspaces                  | Not supported                                                                             |
| Built-in ArtifactRef stability across unchanged hydration         | Available                                                                                 |
| Built-in ArtifactRef stability across hydration reset             | Not guaranteed; refs are intentionally replaced                                           |
| Plugin manifests                                                  | Support deferred                                                                          |
| Git locators                                                      | Support deferred                                                                          |
| Git archive materialization                                       | Support deferred                                                                          |
| URL locators                                                      | Support deferred                                                                          |
| Package locators                                                  | Support deferred                                                                          |
| Archive and zip Sources                                           | Support deferred                                                                          |
| Tool execution                                                    | Owned by a future Tool consumer                                                           |
| Model execution                                                   | Owned by a future model consumer                                                          |
| Agent execution                                                   | Owned by a future Agent runtime                                                           |
| Team execution                                                    | Owned by a future Team runtime                                                            |
| Loop execution                                                    | Owned by a future execution runtime                                                       |
| Workflow scheduling and execution                                 | Owned by a future Workflow runtime                                                        |
