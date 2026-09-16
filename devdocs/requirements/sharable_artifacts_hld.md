# Shareable and Composable AI Artifact Declarations

- [Executive summary](#executive-summary)
- [Motivation and desired user outcome](#motivation-and-desired-user-outcome)
  - [Why this is needed](#why-this-is-needed)
  - [Desired repository experience](#desired-repository-experience)
- [Goals, scope, and non-goals](#goals-scope-and-non-goals)
  - [Goals](#goals)
  - [In scope](#in-scope)
  - [Non-goals](#non-goals)
- [Requirements](#requirements)
  - [Declaration requirements](#declaration-requirements)
  - [Composition requirements](#composition-requirements)
  - [Resolution requirements](#resolution-requirements)
  - [Ordering requirements](#ordering-requirements)
  - [Discovery and Workspace requirements](#discovery-and-workspace-requirements)
  - [Collection and authoring requirements](#collection-and-authoring-requirements)
  - [Built-in content requirements](#built-in-content-requirements)
  - [Runtime and resource requirements](#runtime-and-resource-requirements)
- [Design principles and invariants](#design-principles-and-invariants)
  - [One declaration has one semantic type](#one-declaration-has-one-semantic-type)
  - [Composition is explicit and uniform](#composition-is-explicit-and-uniform)
  - [Existing file formats remain useful](#existing-file-formats-remain-useful)
  - [Source material remains source-backed](#source-material-remains-source-backed)
  - [Composition is separate from execution](#composition-is-separate-from-execution)
  - [Names are reusable in independent scopes](#names-are-reusable-in-independent-scopes)
  - [Physical and semantic identity are different](#physical-and-semantic-identity-are-different)
  - [Collections are groups, not owners](#collections-are-groups-not-owners)
  - [Resolution is observational](#resolution-is-observational)
  - [Declaration status and runtime readiness are separate](#declaration-status-and-runtime-readiness-are-separate)
- [Conceptual model](#conceptual-model)
  - [Concern boundaries](#concern-boundaries)
  - [Terminology](#terminology)
  - [Artifact vocabulary](#artifact-vocabulary)
- [Portable declaration model](#portable-declaration-model)
  - [Common declaration header](#common-declaration-header)
  - [Semantic identity and declaration occurrence](#semantic-identity-and-declaration-occurrence)
  - [Composition-entry forms](#composition-entry-forms)
    - [Symbolic reference](#symbolic-reference)
    - [Located reference](#located-reference)
    - [Contained declaration](#contained-declaration)
    - [Entry classification](#entry-classification)
  - [Ordering and occurrence semantics](#ordering-and-occurrence-semantics)
  - [Locator model](#locator-model)
    - [Path locator](#path-locator)
    - [URL locator](#url-locator)
    - [Git locator](#git-locator)
    - [Package locator](#package-locator)
    - [Command locator](#command-locator)
  - [Path pattern model](#path-pattern-model)
- [Artifact contracts](#artifact-contracts)
  - [Model](#model)
  - [Instruction](#instruction)
  - [Context](#context)
  - [Tool](#tool)
  - [Skill](#skill)
  - [MCP](#mcp)
  - [MCP policy](#mcp-policy)
  - [Collection](#collection)
  - [Agent](#agent)
  - [Team](#team)
  - [Output matching](#output-matching)
  - [Loop](#loop)
  - [Workflow](#workflow)
  - [Workspace](#workspace)
- [Declaration discovery and physical formats](#declaration-discovery-and-physical-formats)
  - [Declaration sources](#declaration-sources)
  - [Supported physical inputs](#supported-physical-inputs)
- [Resolution and capability plans](#resolution-and-capability-plans)
  - [Resolution scope](#resolution-scope)
  - [Unlocated external resolution](#unlocated-external-resolution)
  - [Located external resolution](#located-external-resolution)
  - [Aliases and terminal Artifacts](#aliases-and-terminal-artifacts)
  - [Contained declarations](#contained-declarations)
  - [Relationship status and declaration validity](#relationship-status-and-declaration-validity)
  - [Consumer completeness policy](#consumer-completeness-policy)
  - [Collection expansion](#collection-expansion)
  - [Agent and Team expansion](#agent-and-team-expansion)
  - [Loop and Workflow resolution](#loop-and-workflow-resolution)
  - [Resolver limits](#resolver-limits)
- [Workspace behavior](#workspace-behavior)
  - [Workspace declaration universe](#workspace-declaration-universe)
  - [Workspace roots and capabilities](#workspace-roots-and-capabilities)
  - [Read-only operations and explicit refresh](#read-only-operations-and-explicit-refresh)
  - [Catalog and capability views](#catalog-and-capability-views)
  - [Workspace user flow](#workspace-user-flow)
- [Managed Collection authoring and lifecycle](#managed-collection-authoring-and-lifecycle)
  - [Storage boundary](#storage-boundary)
  - [Portable and managed Collection domains](#portable-and-managed-collection-domains)
  - [Baseline Collections](#baseline-collections)
  - [Managed membership semantics](#managed-membership-semantics)
  - [Managed publication and replacement](#managed-publication-and-replacement)
  - [Supported Collection operations](#supported-collection-operations)
  - [Collection deletion](#collection-deletion)
  - [Skill creation flow](#skill-creation-flow)
  - [MCP server and policy creation flow](#mcp-server-and-policy-creation-flow)
- [Built-in and user-owned artifacts](#built-in-and-user-owned-artifacts)
  - [Built-in package composition](#built-in-package-composition)
  - [Protected Root behavior](#protected-root-behavior)
  - [Package hydration](#package-hydration)
  - [Skill package identity](#skill-package-identity)
- [Runtime consumer model](#runtime-consumer-model)
  - [Prompt and Context consumer](#prompt-and-context-consumer)
  - [Skill consumer](#skill-consumer)
  - [MCP consumer](#mcp-consumer)
  - [Workflow consumer](#workflow-consumer)
  - [Runtime implementation boundaries](#runtime-implementation-boundaries)
- [Technical architecture](#technical-architecture)
  - [Root](#root)
  - [Source](#source)
  - [Definition](#definition)
  - [Artifact](#artifact)
  - [Artifact Store boundary](#artifact-store-boundary)
  - [Resource verification](#resource-verification)
  - [Resolver](#resolver)
  - [Workspace refresh architecture](#workspace-refresh-architecture)
- [Implementation consequences and finalized decisions](#implementation-consequences-and-finalized-decisions)
  - [External references versus contained declarations](#external-references-versus-contained-declarations)
  - [Member-level resolution](#member-level-resolution)
  - [Managed authoring](#managed-authoring)
  - [MCP identity and local configuration](#mcp-identity-and-local-configuration)
  - [Built-in packages](#built-in-packages)
  - [Finalized semantic decisions](#finalized-semantic-decisions)
- [Current implementation status](#current-implementation-status)
  - [Status interpretation](#status-interpretation)
  - [Declaration platform and resolution](#declaration-platform-and-resolution)
  - [Sources, persistence, and resource verification](#sources-persistence-and-resource-verification)
  - [Physical format support](#physical-format-support)
  - [Workspace capability planning](#workspace-capability-planning)
  - [Collections and managed authoring](#collections-and-managed-authoring)
  - [Skill and built-in package support](#skill-and-built-in-package-support)
  - [MCP platform and runtime](#mcp-platform-and-runtime)
  - [Pending completion work](#pending-completion-work)
  - [Deferred and unsupported capabilities](#deferred-and-unsupported-capabilities)
    - [Source, locator, and cross-Root capabilities](#source-locator-and-cross-root-capabilities)
    - [Authoring and frontend scope](#authoring-and-frontend-scope)
    - [Execution runtimes](#execution-runtimes)

## Executive summary

This design provides one declaration system for AI and LLM-related artifacts.

The system allows artifacts to be:

- Authored in repositories or application-managed storage.
- Discovered from familiar files and directories.
- Shared between repositories and applications.
- Referred to by semantic identity or by a specific location.
- Composed into larger capabilities.
- Materialized from verified source-backed resources.
- Consumed by prompt, Skill, MCP, Agent, Team, Workflow, and future runtime systems.

The declaration system separates four responsibilities:

- A declaration describes an artifact.
- A composition declaration selects, groups, or contains artifacts.
- A resolver produces a typed graph and relationship-level status.
- A runtime consumer decides how to materialize or execute that graph.

```text
Repository files and managed packages
  -> normalized artifact declarations
  -> resolved composition graph
  -> runtime-specific capability plan
  -> runtime consumers
```

Composition artifacts such as Collections, Agents, Teams, Loops, Workflows, and Workspaces are ordinary artifacts. They are not alternate stores, ownership databases, or lifecycle parents.

In particular, a Collection is a declaration containing membership relationships. Artifact Store persists the Collection declaration as an ordinary source-backed Artifact. It does not persist Collection membership as a generic Store relationship.

There must be no generic:

- Collection foreign key on an Artifact.
- Collection membership table in Artifact Store.
- Cascading deletion from Collection to member.
- Collection-specific Artifact identity.
- Collection-specific member lifecycle behavior in Artifact Store.

User-facing Collection editing is an application-domain operation above Artifact Store. It updates a managed Collection declaration document.

## Motivation and desired user outcome

### Why this is needed

AI capabilities are commonly distributed across unrelated formats:

- Repository instructions in `AGENTS.md` and `CLAUDE.md`.
- Documentation files that provide useful Context.
- Skills stored in `SKILL.md` directories.
- MCP server configuration in `.mcp.json`.
- Agent Markdown files.
- Canonical YAML and JSON configuration files.
- Application-managed packages.
- Reusable bundles of prompts, Skills, MCP servers, and Workflows.

Without a common declaration model, these formats cannot be composed consistently.

Typical problems include:

- A Skill can be discovered but cannot be referenced uniformly from an Agent.
- A Workspace can load files but cannot express a reusable capability set.
- MCP servers and Skills use unrelated identity models.
- A Workflow cannot refer to an Agent using the same mechanism as a Collection.
- Repository instructions are treated differently from inline instructions.
- Source-backed files are loaded without a common freshness or integrity model.
- Runtime consumers need to understand physical repository layouts.
- A Collection can accidentally become a hidden ownership and lifecycle layer.
- Missing or ambiguous members can invalidate an entire composition instead of being reported at the affected relationship.

The design addresses these problems by separating:

- Declaration:
  - What an artifact means.

- Discovery:
  - Where declarations are found.

- Resolution:
  - How declarations identify and compose other declarations.

- Resource access:
  - How source material is verified and opened.

- Runtime:
  - How a resolved graph is materialized or executed.

### Desired repository experience

A user should be able to place declarations and familiar artifact files in a repository, select a Workspace, and receive a resolved capability graph.

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

The repository can provide declarations with identities such as:

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

A Collection can group independently declared artifacts:

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

From the user's perspective, this means:

```text
Use instruction/repository-rules from the active Root.
Use skill/code-review from the active Root.
Use mcp/github from the stated MCP configuration.
```

The locator narrows where the intended MCP declaration is found. It does not create a Collection-specific copy of the MCP server.

A composition document can also contain a complete declaration when there is no separate declaration to find:

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

Here, `mcp/local-files` is declared inside the Collection document. The Collection document is its declaration source.

## Goals, scope, and non-goals

### Goals

The system must provide:

- One portable declaration vocabulary for supported AI artifacts.
- Uniform identity and reference semantics across composition fields.
- Support for familiar repository formats without requiring complete conversion to canonical YAML or JSON.
- Source-backed resource verification.
- Explicit resolution of missing and ambiguous relationships.
- Reusable, Root-scoped names.
- Composition across Collections, Agents, Teams, Loops, Workflows, and Workspaces.
- Partial capability plans that preserve unavailable relationships.
- Strict runtime behavior when a consumer requires a complete capability set.
- Editable managed Collections without introducing Store-level ownership.
- Equivalent declaration semantics for built-in and user-owned content.
- A clear boundary between declaration resolution and runtime execution.

### In scope

This HLD defines:

- Artifact vocabulary and common declaration fields.
- Composition-entry forms.
- Identity, locator, and occurrence semantics.
- Type-specific declaration contracts.
- Discovery and physical-format normalization.
- Root-scoped resolution.
- Workspace declaration scope.
- Managed Collection authoring and lifecycle.
- Built-in package behavior.
- Runtime consumer boundaries.
- Source-backed storage and resource verification.
- Current implementation status.

### Non-goals

The declaration layer does not itself provide:

- Model execution.
- Tool execution.
- Agent or Team execution.
- Loop execution.
- Workflow scheduling or execution.
- Runtime retries, cancellation, timeouts, or state persistence.
- MCP secret, OAuth, enablement, connection, or policy changes through Collection membership.
- Implicit cross-Root imports.
- Automatic activation of all artifacts discovered in a Root.
- Automatic activation of baseline Collections.
- Source mutation during ordinary resolution or capability reads.
- Generic Artifact Store ownership or lifecycle behavior for composition relationships.

Git, URL, package, archive, and plugin support are represented where relevant in the model, but some remain deferred in the current implementation.

## Requirements

The terms `must`, `should`, and `may` in this section describe target behavior. Current implementation status is documented separately.

### Declaration requirements

- Every canonical declaration must have one semantic `type`.
- Every declaration, including a contained declaration, must have a `name`.
- Portable declarations must not require a second discriminator such as `kind`, `category`, or `role`.
- The default semantic identity must be `(type, name)`.
- Semantic identity must be scoped to a Root.
- Release or compatibility version information must not be part of default symbolic identity.
- The system must preserve both semantic identity and physical declaration-occurrence identity.
- Familiar physical formats must normalize into the same declaration vocabulary.
- Source-backed artifacts must retain verifiable links to their source material.

### Composition requirements

The same entry model must apply to:

- Collection members.
- Agent and Team members.
- Agent and Team programs.
- Loop bodies.
- Workflow node targets.
- Workspace roots.
- Skill `allowedTools`.
- Other declaration fields that accept named artifacts.

Every composition position must support:

- An unlocated external reference by `(type, name)`.
- A located external reference by `(type, name, locator)`.
- A complete contained declaration.

A locator-bearing external entry must:

- Select an existing declaration occurrence.
- Preserve the target's semantic identity.
- Not create a copy of the target.
- Not override target configuration, policy, state, or metadata.
- Not fall back to another location when its selected location is unavailable.

An `mcp` or `mcp.policy` declaration selected through a non-command locator is
a pure alias to its selected terminal Artifact. It must not introduce local
capability selection or policy fields that alter the selected target.

A complete contained declaration must:

- Include the artifact's required type-specific declaration body.
- Produce a declaration occurrence within its containing document.
- Remain distinct from an external reference with only identity, locator, and selectors.

### Resolution requirements

The resolver must:

- Require exactly one available target for an unlocated symbolic reference.
- Report ambiguity when multiple matching declaration occurrences are available.
- Verify the expected type and name for a located reference.
- Report a located reference as unavailable if the selected location does not provide the expected target.
- Never use source order as an ambiguity tie-breaker.
- Detect composition cycles.
- Apply resolution-depth and resolution-node limits.
- Preserve every declared relationship occurrence.
- Report relationship-level `available`, `unavailable`, or `ambiguous` status.
- Keep the containing composition declaration valid when an external relationship is unavailable or ambiguous.
- Return a typed graph without executing it.

A consumer must be able to:

- Reject a capability plan when it requires all selected relationships.
- Use available portions of a capability plan when partial operation is supported.
- Report unavailable or ambiguous relationships without silently dropping them.

### Ordering requirements

The following arrays must not have caller-defined execution or precedence semantics:

- `Collection.members`.
- `Agent.members`.
- `Team.members`.
- `Skill.allowedTools`.
- `Workspace.roots`.

Workflow node, edge, and start arrays describe graph structure rather than source-array execution order.

Resolution and flattened capability output must use deterministic normalized ordering.

Reordering source arrays may still change:

- Raw source bytes.
- Source content digest.
- Artifact revision.

It must not, by itself, change composition behavior.

Explicit runtime request ordering is separate from declaration composition ordering.

### Discovery and Workspace requirements

The system must support:

- Exact declaration sources.
- Scanned declaration sources.
- Inline declaration sources.
- Slash-separated include and exclude patterns.
- Workspace-selected declaration universes.
- Explicit refresh of local locator closure.
- Read-only resolution and capability operations.
- Discovery of Workspace candidates independently of the selected Workspace's non-Workspace declaration scope.
- Separate Sources when concurrently active independent declaration universes are required.

When a Workspace provides `declarations`, including an empty array, that field must define the selected Workspace's non-Workspace declaration universe for its Source.

Nested Workspace references must remain declared but resolve as unavailable.

### Collection and authoring requirements

The system must support:

- Editable user-managed Collections.
- External members that remain declared when their target is missing, invalid, disabled, deleted, or ambiguous.
- Detaching a member without deleting its target.
- Deleting a target without silently deleting Collection membership.
- Restoring a target without rewriting the Collection.
- Complete contained declarations in portable Collection documents.
- Managed Collection APIs that create external membership entries by default.
- Deletion protection for managed Collections with direct members.
- No recursive ownership or deletion semantics.

Each supported user Root must have:

- One application-provisioned Skill baseline Collection.
- One application-provisioned MCP baseline Collection.

Baseline Collections must:

- Have fixed logical names.
- Have fixed managed package locations.
- Be editable.
- Not be renamed.
- Not be disabled.
- Not be runtime-disabled.
- Not be deleted.
- Be explicit selectable authoring destinations.
- Never be implicit API defaults or fallbacks.
- Never be automatically active in a Workspace or runtime session.

Application Root lifecycle, rather than generic Artifact Store Root creation,
must establish these baselines when a user Root is created and reconcile
existing user Roots during application startup. Every application path that
creates a user Root must invoke this application-level provisioner.

Managed Skill, MCP server, and MCP policy creation must receive an explicitly selected compatible editable Collection.

Managed named create operations may be idempotent only for equivalent expected
package content and Artifact state. They must not replace a different existing
package unless the caller uses explicit replacement or upsert, or the operation
is authorized application repair or built-in reconciliation.

### Built-in content requirements

Built-in and user-owned declarations must use the same:

- Identity semantics.
- Composition-entry semantics.
- Locator semantics.
- Contained-declaration semantics.
- Resolution behavior.

Differences may exist only in:

- Edit authority.
- Distribution.
- Protected Root behavior.
- Placement of mutable local overlays.

Package hydration must reconcile only affected built-in packages unless a topology or Root migration requires a complete reset.

### Runtime and resource requirements

Consumers must receive verified resources rather than arbitrary unverified filesystem paths.

Declaration resolution must remain separate from:

- Runtime readiness.
- Resource materialization.
- Installation-local configuration.
- Secret substitution.
- Connection management.
- Scheduling and execution.

## Design principles and invariants

### One declaration has one semantic type

Each declaration has one portable `type`.

```yaml
type: skill
name: code-review
```

There is no secondary portable discriminator such as `kind`, `category`, or `role`.

### Composition is explicit and uniform

Composition relationships are represented by typed entries.

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

The same symbolic, located, and contained forms apply across composition fields.

### Existing file formats remain useful

Users should not need to rewrite every repository convention into a new format.

Supported inputs include:

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

Physical-format adapters normalize these inputs into the common artifact vocabulary.

### Source material remains source-backed

A declaration may refer to:

- A physical file.
- A directory.
- An MCP configuration entry.
- A package resource.
- A command.
- An external source.

Consumers receive verified source material rather than arbitrary paths.

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

It does not execute them. Execution belongs to the applicable runtime consumer.

### Names are reusable in independent scopes

Unrelated Roots may each define:

```text
skill/code-review
mcp/github
collection/repository-review
```

without conflict.

A conflict occurs only when more than one available target with the same semantic identity participates in one resolution scope and cannot be reduced to the same terminal Artifact.

### Physical and semantic identity are different

The system preserves:

- Physical origin:
  - Where a declaration occurrence came from.

- Semantic identity:
  - Which artifact type and logical name it declares.

A locator selects a physical declaration occurrence. It does not change the selected artifact's semantic identity.

### Collections are groups, not owners

Collection membership is declaration content interpreted by the resolver.

Adding or removing a Collection member must not create generic ownership, parentage, or lifecycle coupling in Artifact Store.

### Resolution is observational

Ordinary resolution and capability-plan reads do not change Source discovery or refresh Sources.

Discovery changes are performed through explicit registration or refresh operations.

### Declaration status and runtime readiness are separate

The resolver reports whether a declaration relationship can be identified.

A runtime consumer determines whether the resolved target has all resources, configuration, credentials, or runtime support needed for use.

## Conceptual model

### Concern boundaries

The complete flow is:

```text
Physical Sources
  -> source entries
  -> physical-format decoders
  -> normalized Definitions
  -> source-backed Artifacts
  -> typed declaration resolution
  -> capability plans
  -> runtime consumers
```

Storage and typed composition have separate responsibilities:

```text
Root
  -> Source
  -> source entry
  -> Decoder
  -> Definition
  -> Artifact
  -> Resource

Artifact + Definition
  -> Artifact Resolver
  -> resolved declaration graph
  -> runtime consumer
```

### Terminology

| Term                   | Meaning                                                                                       |
| ---------------------- | --------------------------------------------------------------------------------------------- |
| Root                   | The identity and resolution boundary for a repository or application domain                   |
| Source                 | A registered physical content origin                                                          |
| Source entry           | One discoverable file, directory, or managed source item                                      |
| Definition             | Immutable normalized semantic content decoded from a source declaration                       |
| Artifact               | A local addressable record for one declaration occurrence                                     |
| Resource               | Verified source material associated with an Artifact                                          |
| Declaration            | The semantic description of an artifact                                                       |
| Declaration occurrence | The physical or structural place where a declaration is written                               |
| Semantic identity      | The Root-scoped pair `(type, name)`                                                           |
| Composition entry      | A reference to or contained declaration of another artifact                                   |
| Composition occurrence | One relationship position in a containing declaration                                         |
| Capability plan        | A resolved graph with relationship-level status and consumer-specific readiness               |
| Terminal Artifact      | The final independently addressable Artifact reached after resolving source-selection aliases |
| ArtifactRef            | The stable local reference used to identify an Artifact occurrence                            |

### Artifact vocabulary

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

The implementation also supports:

```text
mcp.policy
```

| Type          | Purpose                                                                 |
| ------------- | ----------------------------------------------------------------------- |
| `instruction` | Behavioral, task-oriented, or prompt text                               |
| `context`     | Repository, documentation, or inline information supplied to a consumer |
| `tool`        | A named operation or facility                                           |
| `model`       | A provider-qualified model declaration and parameters                   |
| `skill`       | A path-backed reusable Skill capability                                 |
| `mcp`         | One MCP server capability                                               |
| `mcp.policy`  | MCP runtime policy composed by MCP consumers                            |
| `collection`  | A reusable group of declarations                                        |
| `agent`       | A composed AI Agent declaration                                         |
| `team`        | A multi-Agent composition                                               |
| `loop`        | Repeated application of an entry                                        |
| `workflow`    | An explicit graph of entry applications                                 |
| `workspace`   | A repository declaration manifest and root-selection document           |

## Portable declaration model

### Common declaration header

Every canonical declaration has the same conceptual header.

```text
Header {
  $schema?: string
  apiVersion?: string

  type: ArtifactType
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

Every declaration requires `name`, including nested and contained declarations.

### Semantic identity and declaration occurrence

The logical identity of a declaration is:

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

A declaration occurrence identifies where one declaration is defined:

```text
Root
  -> Source
    -> declaration location
      -> optional named structural position
```

The following rules apply:

- Multiple Roots may contain the same `(type, name)` without conflict.
- One Root may contain multiple declaration occurrences with the same `(type, name)`.
- An unlocated symbolic reference requires exactly one available terminal target.
- Multiple available terminal targets make an unlocated reference ambiguous.
- Multiple aliases resolving to the same terminal Artifact count as one symbolic target.
- A located reference selects the occurrence at the stated location.
- The selected target must confirm the requested type and name.
- Source order is never an ambiguity tie-breaker.
- A contained declaration is selected through its containing document and named structural position.
- A contained declaration may have the same semantic identity as another declaration occurrence.
- Duplicate contained sibling declarations with the same type and name are invalid.
- A version may describe release or compatibility information, but is not part of default symbolic lookup identity.

### Composition-entry forms

A composition field accepts a `CompositionEntry`.

```text
CompositionEntry =
  SymbolicReference
  | LocatedReference
  | ContainedDeclaration
```

The containing field establishes that the object is a composition entry. A separate `ref` wrapper is not required.

#### Symbolic reference

```yaml
type: skill
name: code-review
```

Meaning:

```text
Use skill/code-review.
Find exactly one available matching target in the active Root scope.
```

The entry identifies a target only by semantic identity.

#### Located reference

```yaml
type: skill
name: code-review
locator: ./skills/code-review
```

Meaning:

```text
Use skill/code-review.
Find the expected declaration at ./skills/code-review.
```

A located reference is still an external relationship. The locator narrows target selection.

The target must confirm the requested `type` and `name`.

A located reference does not:

- Create another declaration.
- Create a composition-specific copy.
- Override the selected target.
- Fall back to an unlocated match.
- Change the target's semantic identity.

A type-specific selector may be used when one location contains multiple declarations.

```yaml
type: mcp
name: github
locator: ./.mcp.json
server: github
```

The `server` field selects one server from a multi-server MCP configuration. It does not make the containing document declare another `mcp/github`.

#### Contained declaration

A contained declaration is complete within the containing document.

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

The containing document is the declaration source for `mcp/local-files`.

A contained declaration:

- Has a required `type` and `name`.
- Includes the type-specific body needed to declare the artifact.
- May refer to ordinary resources where its artifact contract permits that.
- Produces a stable declaration occurrence in the containing document.
- Does not obtain omitted declaration data from another declaration merely because it has a locator.

A locator alone does not make an entry contained.

```yaml
type: skill
name: code-review
locator: ./skills/code-review
```

This is a located reference to a standard Skill package. It is not a contained Skill declaration.

An inline Instruction, by contrast, includes declaration content:

```yaml
type: instruction
name: review-rules
mediaType: text/markdown
content: |
  Do not modify generated files.
  Prefer small, reviewable changes.
```

#### Entry classification

An entry with only the following information is external:

- `type`.
- `name`.
- An optional non-command locator.
- An optional type-specific source selector such as MCP `server`.
- Optional relationship-local presentation annotations.

Relationship-local annotations do not override the selected target's:

- Metadata.
- Runtime configuration.
- Policy.
- Local state.
- Enablement.
- Installation data.

Metadata differences may still distinguish separate external composition occurrences.

An entry becomes contained when it includes a complete type-specific declaration body according to its artifact contract.

A command locator is executable Tool or MCP declaration data. It is not a source-selection locator and therefore contributes to a contained or independent concrete declaration.

An external entry does not emit a source subresource Artifact. A contained declaration does.

A declaration that is valid with only `type` and `name`, such as a minimal Agent, is interpreted as an external reference when it appears in a composition-entry position. A contained form must include actual local composition or program data.

### Ordering and occurrence semantics

Composition arrays do not provide caller-defined precedence or execution order.

The resolver must:

- Preserve every declared composition occurrence.
- Preserve relationship-local metadata.
- Produce deterministic normalized output ordering.
- Preserve status for unavailable and ambiguous occurrences.

Flattened prompt, Skill, and MCP capability lists use consistent ArtifactRef deduplication. The relationship graph still retains each declaration occurrence that led to the target.

Workflow arrays represent graph records. Workflow behavior is determined by IDs, edges, matchers, and joins rather than array position.

Runtime consumers may define explicit runtime request ordering independently of source-array order.

### Locator model

A portable locator identifies a declaration source, implementation, package, command, or external resource.

```text
Locator =
  string
  | PathLocator
  | URLLocator
  | GitLocator
  | PackageLocator
  | CommandLocator
```

#### Path locator

```yaml
locator: ./skills/code-review
```

Equivalent structured form:

```yaml
locator:
  kind: path
  path: ./skills/code-review
```

#### URL locator

```yaml
locator:
  kind: url
  url: https://example.com/agents/reviewer.yaml
  integrity: sha256:...
```

#### Git locator

```yaml
locator:
  kind: git
  repository: https://github.com/acme/agent-assets.git
  revision: v2.0.0
  path: skills/code-review
```

#### Package locator

```yaml
locator:
  kind: package
  manager: go
  package: github.com/acme/agent-tools
  version: v1.2.0
  path: search
```

#### Command locator

```yaml
locator:
  kind: command
  command: grep
```

Relative locators are interpreted relative to the declaration file containing the locator.

For example:

```text
collections/repository-review.yaml
  locator: ../skills/code-review
```

is resolved relative to:

```text
collections/
```

It is not resolved relative to the process working directory.

A locator has a context-specific role:

| Context                        | Meaning                                                                                                                      |
| ------------------------------ | ---------------------------------------------------------------------------------------------------------------------------- |
| Independent declaration        | Identifies source material, a package, an implementation, or another source as defined by that artifact contract             |
| Incomplete composition entry   | Selects where the expected external target must be found                                                                     |
| Complete contained declaration | Supplies artifact data only where the artifact contract defines it; it does not implicitly import omitted declaration fields |

A locator in an external composition entry is a selection constraint. If the location does not provide the expected target, that relationship is unavailable.

A command locator is connection or execution information for a Tool or MCP declaration. It is not a location used to find a separate Tool or MCP declaration.

### Path pattern model

Declaration discovery and Context selection use slash-separated patterns.

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

The same matching behavior applies to:

- Source declaration discovery.
- Workspace declaration scans.
- Context resource selection.

## Artifact contracts

### Model

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

### Instruction

```text
Instruction {
  Header

  content?: string
  mediaType?: string
}
```

Source-backed Instruction:

```yaml
type: instruction
name: repository-rules
locator: ./AGENTS.md
```

Contained Instruction:

```yaml
type: instruction
name: review-rules
mediaType: text/markdown
content: |
  Do not modify generated files.
  Prefer small, reviewable changes.
```

### Context

```text
Context {
  Header

  content?: string
  mediaType?: string

  include?: string[]
  exclude?: string[]
}
```

Repository Context:

```yaml
type: context
name: architecture
locator: ./docs
include:
  - "**/*.md"
exclude:
  - "**/generated/**"
```

Contained Context:

```yaml
type: context
name: review-boundary
mediaType: text/plain
content: |
  The review covers checkout and payment packages.
```

### Tool

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

### Skill

```text
Skill {
  Header

  license?: string
  allowedTools?: CompositionEntry[]
}
```

Independent Skill declaration:

```yaml
type: skill
name: code-review
description: Reviews repository changes
locator: ./skills/code-review

allowedTools:
  - type: tool
    name: repository-search
```

A concrete independent Skill requires a Skill package locator.

A standard Skill is an atomic path-backed capability:

```text
skills/code-review/
  SKILL.md
  references/
  scripts/
  assets/
```

The `SKILL.md` file declares the Skill. Its containing directory is the verified Skill resource root.

The `SKILL.md` physical-format adapter emits an explicit `./SKILL.md` locator relative to its own declaration occurrence.

When a Skill appears in a composition position with only `type`, `name`, and an optional locator, it is an external reference:

```yaml
- type: skill
  name: code-review
  locator: ./skills/code-review
```

This does not create another `skill/code-review` declaration in the containing Collection, Agent, Team, Workflow, or Workspace.

A contained Skill is allowed only when the containing document includes a complete self-contained Skill declaration supported by the Skill contract. A locator pointing to a normal `SKILL.md` package remains an external reference.

`allowedTools` uses the common composition-entry model and has no caller-defined ordering semantics.

### MCP

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

Stdio MCP:

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

HTTP MCP:

```yaml
type: mcp
name: issue-tracker
transport: streamable-http
url: https://mcp.example.com/issues
headers:
  Authorization: "${ISSUE_TRACKER_TOKEN}"
```

MCP selected from a multi-server configuration:

```yaml
type: mcp
name: github
locator: ./.mcp.json
server: github
```

In an independent declaration, this form selects connection data for `mcp/github`.

In a composition position, it means:

```text
Use the independently declared mcp/github selected from .mcp.json.
```

The `server` selector participates in source target selection. It does not create a contained MCP declaration.

For a concrete MCP declaration, an absent `include` selects the complete server
capability.

A present `include` limits the declared server capability to the listed tools,
resources, or prompts.

A non-command source-selected MCP is a pure alias to a terminal Artifact and
cannot define local `include` rules.

An MCP declaration must use exactly one connection-source model:

- An inline executable connection.
- A command locator, which implies `stdio`.
- A non-command locator selecting connection data from another source entry as a pure alias.

A complete inline connection cannot also contain a non-command source locator.

An external MCP composition entry cannot override:

- Connection details.
- Include rules.
- Authentication declarations.
- Runtime policy.
- Installation configuration.
- Runtime state.
- Enablement.

Distinct configurations of the same logical server should use distinct names:

```text
mcp/github-read
mcp/github-write
```

### MCP policy

`mcp.policy` is a supported declaration type used by MCP consumers to compose
runtime policy.

Concrete MCP policies use a required `body` with the following optional fields:

| Field                 | Meaning                                                   |
| --------------------- | --------------------------------------------------------- |
| `trustLevel`          | Trust classification with `trusted` or `untrusted` values |
| `defaultPolicy`       | Default approval and execution rules                      |
| `toolPolicies`        | Per-tool policy overrides                                 |
| `expectedToolDigests` | Optional expected digests for discovered tools            |
| `staleDigestPolicy`   | Policy applied when an expected tool digest is stale      |
| `appsPolicy`          | MCP Apps enablement and approval behavior                 |

Approval rules use `allow`, `ask`, or `deny`. Execution modes use `auto` or
`manual`.

A concrete MCP policy requires `body`. A non-command source-selected policy may
use a locator without `body`; it is a pure alias to the selected terminal
policy Artifact and cannot merge or override the selected policy body.

`mcp.policy` follows the common header, identity, composition, Source,
Definition, Artifact, and resolution contracts.

### Collection

```text
Collection {
  Header

  version?: string
  members?: CompositionEntry[]
}
```

A Collection is a reusable group of declarations.

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
    locator: ../.mcp.json
    server: github

  - type: tool
    name: repository-search

  - type: workflow
    name: review-workflow
```

Collections can represent:

- Capability bundles.
- Skill bundles.
- MCP bundles.
- Repository-specific groups.
- Reusable package exports.
- Mixed artifact groups.

Portable Collection declarations may contain mixed artifact types. User-facing managed Collection APIs may impose narrower domain rules as described later.

Collection membership does not create an ownership hierarchy.

The member forms are:

| Form                               | Meaning                                                                  |
| ---------------------------------- | ------------------------------------------------------------------------ |
| Exact `type` and `name`            | Use one independently declared target found by Root-wide identity lookup |
| `type`, `name`, and locator        | Use the expected external target at the stated location                  |
| Complete type-specific declaration | Declare a contained Artifact in the Collection document                  |

A Collection remains valid when an external member is unavailable or ambiguous.

```text
collection/repository-review
  -> skill/code-review: available
  -> mcp/github: unavailable
  -> workflow/review-workflow: available
```

Collection member arrays have no caller-defined ordering semantics.

### Agent

```text
Agent {
  Header

  members?: CompositionEntry[]
  program?: ProgramEntry
}

ProgramEntry =
  CompositionEntry constrained to loop or workflow
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
  name: reviewer-program
  maxIterations: 16
  until:
    pointer: /complete
    schema:
      const: true
```

An Agent supports all common composition-entry forms.

A minimal independent Agent is valid:

```yaml
type: agent
name: reviewer
```

The same object in a composition position is an external Agent reference. A contained Agent must include actual local composition or program data.

An unavailable member does not invalidate the Agent declaration. It affects readiness for consumers that intend to use that member.

Agent member arrays have no caller-defined ordering semantics.

### Team

```text
Team {
  Header

  members?: CompositionEntry[]
  program?: ProgramEntry
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

Team members and programs follow the common symbolic, located, and contained entry semantics.

Team member arrays have no caller-defined ordering semantics.

### Output matching

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

When `pointer` is absent, the schema applies to the complete result.

### Loop

```text
Loop {
  Header

  body?: CompositionEntry
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
- The consumer stops execution for a runtime-specific reason.

A Loop nested under `Agent.program` or `Team.program` may omit `body`. The containing Agent or Team is then the Loop body.

The Loop still requires a name. A body-less program Loop is a named contextual program node whose body is supplied by its containing Agent or Team.

A Loop body may be:

- An unlocated external target.
- A located external target.
- A contained declaration.

An unavailable body does not make the Loop structurally invalid. It makes the Loop unavailable for execution until a suitable body is available.

### Workflow

```text
Workflow {
  Header

  start?: string[]
  nodes?: WorkflowNode[]
  edges?: WorkflowEdge[]
}

WorkflowNode {
  id: string
  target: CompositionEntry
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
- An edge without a matcher is eligible after its source completes.
- An edge with a matcher is eligible only when its source result matches.
- Workflow cycles are valid.
- A node may execute more than once.
- Array position does not define execution order.

Workflow node targets use the common composition-entry model.

A Workflow remains structurally valid when a target is unavailable. The affected node remains unavailable for execution until its target resolves.

### Workspace

```text
Workspace {
  Header

  declarations?: DeclarationSource[]
  roots?: CompositionEntry[]
}
```

A Workspace describes:

- Where declarations should be discovered for a selected repository experience.
- Which declarations are the roots of that experience.

When `declarations` is omitted, the Workspace uses the declaration universe already configured on its containing Source.

When `declarations` is present, including an empty array, it defines the selected Workspace's non-Workspace declaration scope for that Source.

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

Workspace roots use the common composition-entry model.

A root may:

- Select an independent declaration by type and name.
- Select an independent declaration at a location.
- Contain a complete declaration.

A Workspace remains valid when a root or nested relationship is unavailable. Its capability plan reports the affected relationship.

Workspace root arrays have no caller-defined ordering semantics.

A Workspace reference nested inside another Workspace remains declared but resolves as unavailable.

## Declaration discovery and physical formats

### Declaration sources

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

Exact sources:

```yaml
declarations:
  - ./agents/reviewer.agent.yaml
  - ./AGENTS.md
  - ./.mcp.json
```

Scanned sources:

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

Inline source:

```yaml
declarations:
  - type: instruction
    name: review-policy
    content: |
      Do not modify generated files.
```

A Workspace declaration source adds independent declarations to the Workspace declaration universe.

A contained declaration in `Workspace.roots` is different. It belongs to Workspace composition rather than the independent declaration-source list.

### Supported physical inputs

| Physical input                  | Produced artifact behavior                                                                                       |
| ------------------------------- | ---------------------------------------------------------------------------------------------------------------- |
| Canonical JSON                  | One top-level declaration and named contained declarations                                                       |
| Canonical YAML                  | One top-level declaration and named contained declarations                                                       |
| `AGENTS.md`                     | One `instruction`                                                                                                |
| `CLAUDE.md`                     | One `instruction`                                                                                                |
| `README.md`                     | One `context`                                                                                                    |
| `llms.txt`                      | One `context`                                                                                                    |
| Selected documentation Markdown | One `context` per selected file                                                                                  |
| `SKILL.md`                      | One independent `skill`                                                                                          |
| `.mcp.json`                     | One independent `mcp` Artifact per configured server                                                             |
| `mcp.json`                      | One independent `mcp` Artifact per configured server                                                             |
| `AGENT.md`                      | One `agent`, an Instruction for a non-empty Markdown body, and complete contained declarations from front matter |
| `*.agent.md`                    | One `agent`, an Instruction for a non-empty Markdown body, and complete contained declarations from front matter |
| Canonical Collection YAML       | One `collection` plus complete contained declarations                                                            |
| Workspace manifest              | One `workspace` plus complete contained root declarations                                                        |

A Collection file does not create another Skill or MCP merely because an external member supplies a locator.

One physical file may emit multiple Artifacts when it contains complete named declarations.

For Agent Markdown, a generated `<agent-name>-instructions` Instruction is emitted
only when the Markdown body is non-empty. Complete declarations in front matter
are emitted at their stable contained-declaration positions.

A Workspace manifest emits contained Artifacts for complete declarations in its
roots. External Workspace roots remain composition relationship edges.

Contained declarations use stable type and name structural positions, for example:

```text
members/mcp/local-files
members/instruction/reviewer-rules
nodes/review/target/agent/reviewer
```

Reordering an array may change declaration bytes, content digest, and Artifact revision. It does not replace the intended identity of a named contained declaration or introduce composition-order semantics.

Resolution selects a contained Artifact by this exact structural position,
rather than by logical identity and equal body alone.

## Resolution and capability plans

### Resolution scope

A Root defines the default semantic identity scope.

A selected Workspace may further define which non-Workspace declarations from its containing Source participate in the active declaration universe.

Symbolic identity lookup uses:

```text
(type, name)
```

A Root can contain multiple declaration occurrences with that identity.

### Unlocated external resolution

An unlocated reference resolves by:

```text
(type, name)
```

Example:

```yaml
type: skill
name: code-review
```

Resolution outcomes are:

- No available target:
  - The relationship is `unavailable`.

- Exactly one available terminal target:
  - The relationship is `available`.

- Multiple available terminal targets:
  - The relationship is `ambiguous`.

The resolver never silently selects a winner based on source order.

### Located external resolution

A located reference resolves by:

```text
(type, name, locator, optional selector)
```

Example:

```yaml
type: skill
name: code-review
locator: ./skills/code-review
```

The selected location must resolve to `skill/code-review`.

If the selected declaration is absent, disabled, invalid, incompatible, or has a different type or name, the relationship is unavailable.

A located reference never falls back to another matching declaration elsewhere in the Root.

For MCP declarations, `server` participates in selecting the target from a multi-server source.

### Aliases and terminal Artifacts

Canonical source-selected declarations may act as aliases of physical-format declarations.

For example, a canonical declaration may select an MCP server from `.mcp.json`.

Resolution follows the source-selection relationship to a terminal Artifact.

The following rules apply:

- Multiple aliases resolving to one terminal Artifact count as one symbolic target.
- Resource claims do not suppress physical declaration targets.
- Same-Source Skill and MCP targets remain selectable.
- A source-selected alias cannot merge local capability or policy fields into its terminal Artifact.
- External composition does not create another Artifact identity.
- Prompt, Skill, and MCP capability lists use consistent ArtifactRef deduplication.

### Contained declarations

A contained declaration resolves directly through:

```text
containing declaration
  -> stable named structural position
  -> contained Artifact
```

It does not require Root-wide symbolic lookup for that composition occurrence.

Removing a contained declaration from its containing document means it is no longer declared.

A contained declaration's source occurrence does not imply ownership or cascading lifecycle behavior in Artifact Store.

### Relationship status and declaration validity

Declaration resolution uses:

```text
available
unavailable
ambiguous
```

A relationship may be unavailable because its target is:

- Missing.
- Disabled.
- Invalid.
- Incompatible.
- Cyclic on the affected expansion path.
- Not resolvable at its stated locator.
- A nested Workspace.
- Otherwise excluded from the active scope.

The resolver preserves every declared composition occurrence and its status.

A missing or ambiguous external target does not invalidate the containing:

- Collection.
- Agent.
- Team.
- Loop.
- Workflow.
- Workspace.

Resource absence is distinct from declaration absence:

- Removing an independent declaration makes its relationships unavailable.
- Removing a required resource from an otherwise resolvable declaration may make the Artifact not ready for runtime use.
- Removing a contained declaration means it is no longer declared.
- Resource failure does not invalidate an otherwise valid containing composition declaration.

### Consumer completeness policy

A capability plan contains resolved targets and relationship-level diagnostics.

A consumer that requires a complete capability set may reject the plan.

A catalog, management view, or partial-capability runtime may use available targets while reporting unavailable or ambiguous relationships.

The resolver itself does not silently remove, replace, or execute relationships.

### Collection expansion

Collections expand recursively using deterministic normalized ordering.

Expansion preserves:

- Every declared member occurrence.
- Relationship-local annotations.
- Available targets.
- Unavailable targets.
- Ambiguous targets.

Example:

```text
collection/repository-review
  -> instruction/repository-rules: available
  -> context/architecture: available
  -> skill/code-review: available
  -> mcp/github: unavailable
```

This supports best-effort groups:

- A member may be added before its target exists.
- An unavailable member may be removed.
- A target may disappear without invalidating the Collection.
- A restored target becomes available without rewriting the Collection.
- A member may be detached without deleting its target.
- A target may be deleted without deleting its declared memberships.

Nested Collection cycles are reported on the affected expansion path.

```text
collection/a
  -> collection/b

collection/b
  -> collection/a
```

The Collection declarations remain valid. The cyclic relationship is unavailable for that path.

### Agent and Team expansion

Agents and Teams may compose:

- Instructions.
- Contexts.
- Models.
- Skills.
- Tools.
- MCP servers.
- Collections.
- Delegated Agents.
- Optional Loop or Workflow programs.

Their member arrays have no caller-defined ordering semantics. Resolution produces deterministic normalized output.

An unavailable member is reported in the capability plan without invalidating the Agent or Team declaration.

A strict runtime may reject an Agent or Team plan that is not complete.

### Loop and Workflow resolution

Loops and Workflows are structural declarations.

The resolver validates and resolves:

- Loop bodies.
- Workflow node IDs.
- Start node references.
- Edge endpoints.
- Node targets.
- Nested composition relationships.
- Cycles and limits.

The runtime consumer owns:

- Scheduling.
- Invocation.
- Retries.
- Timeouts.
- Cancellation.
- State persistence.
- Output evaluation.
- Result handling.

A Loop or Workflow remains structurally valid when a target is unavailable. Execution readiness is reported at the affected body or node.

### Resolver limits

The resolver must enforce:

- Cycle detection.
- Maximum resolution depth.
- Maximum resolution node count.

Exceeding a limit affects the applicable relationship path. It does not create an ownership or deletion relationship.

## Workspace behavior

### Workspace declaration universe

A Workspace controls two related concerns:

- `declarations` defines the selected non-Workspace declaration universe for its Source.
- `roots` defines the selected capability entry points.

When `declarations` is omitted, existing Source discovery configuration is used.

When `declarations` is present, including an empty array, it replaces the selected Workspace's non-Workspace declaration scope for that Source.

Initial broad scanning used to locate Workspace declarations is not retained as an implicit source of unrelated declarations.

However, previously discovered Workspace candidate locations remain in Source discovery. This permits switching between Workspaces in one Source.

Only the selected Workspace's non-Workspace declaration universe is active for that Source.

Concurrently active independent declaration universes require separate Sources.

### Workspace roots and capabilities

Workspace roots are resolved using deterministic normalized ordering.

A root can expand a:

- Collection.
- Agent.
- Team.
- Loop.
- Workflow.
- Other supported non-Workspace target.

A Workspace does not automatically activate every Collection, Skill, MCP server, or other Artifact in its Root.

The selected capability set is determined by:

- Workspace roots.
- Their resolved composition.
- Consumer-specific filtering and readiness rules.

The Workspace capability plan preserves:

- Selected root occurrences.
- Available capabilities.
- Unavailable and ambiguous roots.
- Unavailable and ambiguous nested relationships.
- Consumer-specific readiness information.

For Workspace runtime plans, `WorkspaceRuntimeSelection.RequireComplete`
selects strict completeness behavior. When enabled, unavailable and ambiguous
required relationships reject the runtime selection with target-level diagnostics.

Nested Workspace references remain visible as declarations but resolve as unavailable.

### Read-only operations and explicit refresh

Located-reference resolution is read-only.

The following operations must not mutate Sources:

- Workspace reads.
- Workspace capability reads.
- Prompt planning.
- Skill planning.
- MCP planning.
- Runtime-plan generation.

`RefreshWorkspace` explicitly:

- Computes the reachable local locator closure.
- Updates Source discovery.
- Refreshes affected declaration state.

Ordinary resolution does not perform this refresh implicitly.

### Catalog and capability views

Workspace catalog views show all discovered Artifact kinds.

Workspace capability views show only the capabilities selected through roots and their composition.

Prompt, Skill, and MCP capability lists use the same ArtifactRef deduplication behavior.

### Workspace user flow

```text
Select a local repository directory or Workspace manifest
  -> register the repository Source
  -> discover declaration and Workspace candidates
  -> identify the selected Workspace
  -> compute and refresh its local declaration closure
  -> apply the Workspace declaration scope
  -> resolve Workspace roots
  -> provide a capability plan to consumers
```

A Workspace may use:

- Existing repository files.
- Canonical JSON and YAML declarations.
- Exact declaration paths.
- Declaration scans.
- Inline declaration sources.
- Contained root declarations.
- Collections.
- Agents.
- Teams.
- Skills.
- MCP servers.
- Loops.
- Workflows.

Unavailable and ambiguous relationships remain visible in the capability plan.

## Managed Collection authoring and lifecycle

### Storage boundary

A managed Collection is stored as a source-backed Collection declaration.

Artifact Store does not maintain generic Collection relationships.

User-facing management code may:

- Read Collection declarations to derive membership.
- Maintain a domain-owned membership cache.
- Update a managed Collection document.
- Re-resolve the declaration after publication.

It must not make Artifact Store infer ownership, reverse membership, or cascading lifecycle behavior.

### Portable and managed Collection domains

Portable Collection declarations may contain mixed artifact types.

The currently exposed frontend Collection surfaces are domain-specific:

- User-created Skill Collections accept only Skill members.
- User-created MCP Collections accept MCP server and MCP policy members.
- Workspace, Skill, and MCP APIs are the exposed frontend Collection surfaces.
- Mixed-Collection frontend authoring is deferred.

A mixed Collection remains valid in canonical source documents and for future or non-frontend use cases.

### Baseline Collections

Each user Root has:

- One application-provisioned Skill baseline Collection.
- One application-provisioned MCP baseline Collection.

Generic Artifact Store Root creation remains domain-neutral and does not infer
Skill or MCP Collection policy. Application Root lifecycle establishes these
baselines during user Root creation and reconciles existing user Roots during
startup.

Baseline Collections have:

- Fixed logical names.
- Fixed managed package locations.
- User-visible selection entries.
- Editable membership.

A baseline Collection cannot be:

- Renamed.
- Disabled.
- Runtime-disabled.
- Deleted.

A baseline Collection is an authoring destination. It is not:

- An automatically active capability set.
- A Workspace default.
- An Agent or Team default.
- A runtime-session default.
- An API fallback.

Every managed Skill, MCP server, or MCP policy creation request must explicitly provide the selected compatible editable Collection.

### Managed membership semantics

Managed authoring APIs create external membership entries unless a future specialized flow explicitly supports contained content.

Adding an external member does not:

- Create or copy the target.
- Enable or disable the target.
- Configure the target.
- Modify target metadata.
- Install the target.
- Change runtime policy.

Removing an external member does not delete its target.

Editing an independently declared target affects every Collection that refers to that target.

Deleting an independently declared target leaves membership declarations intact. Those relationships become unavailable.

Restoring a matching target makes existing relationships available again.

A contained declaration exists while it remains in the containing Collection document.

### Managed publication and replacement

Managed Collections, Skills, MCP servers, and MCP policies are published as
independent source-backed packages. Managed discovery is authoritative for
these package types.

A named create operation may be idempotent only when expected package content
and expected Artifact state are equivalent. It must not silently replace a
different existing package. Explicit replacement or `upsert`, application
repair, and built-in reconciliation may authorize replacement.

Equivalent current Source and expected Artifact state skips unnecessary Source
refresh. Package removal reconciles managed discovery and cleans up stale
managed declaration locators.

Current managed Skill, MCP server, and MCP policy creation uses two source-side
writes:

- Record the selected Collection's external membership.
- Publish the independently declared package.

If publication fails, the declared member remains valid, visible, and
unavailable. A retry must re-read the Collection revision. Durable recovery of
the multi-package operation remains pending.

### Supported Collection operations

A user can:

- Create an empty compatible Collection.
- Select a baseline Collection.
- Select another editable compatible Collection.
- Create a new Collection and then explicitly select it.
- Add an existing independently declared Artifact.
- Add a member before its target is available.
- Add a locator when a specific declaration occurrence is intended.
- Remove a member without deleting the target.
- View unavailable or ambiguous members.
- Attach one independent target to multiple Collections.
- View direct compatible Collection membership declarations for a selected
  Artifact, including available, unavailable, and ambiguous direct relationship
  status.

Portable mixed Collections may include:

- Skills.
- MCP servers.
- MCP policies.
- Instructions.
- Contexts.
- Agents.
- Teams.
- Loops.
- Workflows.
- Other supported artifacts.

Repository discovery does not silently alter an editable Collection. A discovered Skill or MCP server may be attached only through an explicit authoring operation.

### Collection deletion

A managed user Collection cannot be deleted while its direct `members` array contains any declared member.

The deletion guard follows these rules:

- Missing or ambiguous targets still count as declared members.
- Only direct members are checked.
- Members of nested Collections are not counted recursively.
- Baseline Collections can never be deleted.
- Deleting an empty ordinary Collection removes only the Collection declaration package.
- Deleting a Collection never deletes independently declared targets.

Deletion flow for an ordinary managed Collection:

```text
Detach every direct member
  -> re-read the Collection at the expected revision
  -> reject deletion if any direct member remains
  -> remove only the Collection declaration package
```

### Skill creation flow

The user-facing managed Skill creation flow is Collection-oriented.

```text
Explicitly select an editable Skill Collection
  -> record an external Skill membership in the selected Collection
  -> publish the independent Skill declaration and package
  -> show the Skill and all Collection memberships
```

Current failure and retry behavior is defined in [Managed publication and replacement](#managed-publication-and-replacement).

There is no implicit baseline fallback.

The user-facing managed API does not expose standalone Skill creation that omits Collection selection.

The result is still one independently declared Skill.

```text
Detach Skill from Collection
  -> Skill remains independently available

Delete Skill
  -> Collection membership remains declared
  -> Collection reports the Skill relationship as unavailable
```

Collection-oriented creation does not make the Skill Collection-owned. The Skill may later be attached to additional Collections.

A Skill discovered from a repository remains independently declared. It may be attached to an explicitly selected editable Collection.

### MCP server and policy creation flow

Managed MCP creation follows the same identity and membership model.

```text
Explicitly select an editable MCP Collection
  -> record an external member in the selected Collection
  -> publish the independent MCP server or MCP policy declaration
  -> configure installation-local inputs where required
  -> enable and connect only when the user chooses
```

There is no implicit baseline fallback.

Collection membership does not:

- Configure secrets.
- Configure OAuth.
- Configure client credentials.
- Enable runtime use.
- Connect the server.
- Change effective MCP policy.
- Replace installation-local configuration.

These remain properties of the independently declared MCP Artifact and its user-local configuration.

If an MCP server or policy is deleted, its Collection membership remains declared and unavailable.

## Built-in and user-owned artifacts

Built-in and user-owned artifacts use the same declaration and composition semantics.

| Aspect                     | Built-in content       | User-owned content            |
| -------------------------- | ---------------------- | ----------------------------- |
| Declaration semantics      | Same                   | Same                          |
| Unlocated external members | Same                   | Same                          |
| Located external members   | Same                   | Same                          |
| Contained declarations     | Same                   | Same                          |
| Resolution behavior        | Same                   | Same                          |
| Editing                    | Application-controlled | User-controlled               |
| Distribution               | Application release    | Repository or managed storage |

### Built-in package composition

A built-in release may distribute related files together:

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

A built-in Collection may group independently declared artifacts:

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

Physical co-distribution does not make the Collection the owner of these Artifacts. It makes package-relative locations portable.

A built-in Collection may also contain a complete MCP or MCP policy declaration when that declaration is authored inside the Collection document.

Built-in package installation must not rewrite contained MCP or MCP policy declarations into generated external declaration files.

### Protected Root behavior

Ordinary Source and Artifact mutation is rejected for the protected built-in Root.

Protected built-in Artifact state is application-managed.

Mutable MCP installation state for protected servers is stored in an external local overlay.

User-owned direct and managed Artifacts live in user Roots and retain ordinary local Artifact state.

Symbolic lookup remains Root-scoped.

A Workspace in a user Root does not automatically import Artifacts from the protected built-in Root.

Explicit cross-Root Workspace imports are not currently supported.

### Package hydration

Hydration maintains package-scoped desired state and package fingerprints.

During bootstrap, each declared built-in package is checked idempotently
against expected package content and expected Artifacts. A matching hydration
marker alone is not proof that physical package content still exists.

Equivalent packages with current Source and Artifact state do not trigger an
unnecessary Source refresh. Changed, incomplete, or corrupted known packages
are repaired and refreshed. Stale declared packages are removed through
package hydration reconciliation.

Current reconciliation is authoritative for known declared packages. It does
not yet sweep manually inserted untracked content from protected Sources.

Artifacts, overlays, secrets, and settings outside a changed or removed package remain stable.

Removing a package purges only overlays and secrets associated with MCP Artifacts removed by that package.

A complete protected-topology reset and ArtifactRef replacement is reserved for:

- Incompatible topology migration.
- Root identity migration.
- Unrecoverable topology corruption.

ArtifactRefs remain stable for unchanged packages during ordinary hydration.

ArtifactRef stability is not guaranteed across an intentional topology migration.

### Skill package identity

A Skill directory remains one Artifact:

```text
skills/code-review/
  SKILL.md
  references/
  scripts/
  assets/
  -> skill/code-review
```

Collection membership does not create additional Skill identities.

## Runtime consumer model

### Prompt and Context consumer

```text
resolved Instructions and Contexts
  -> verified source material
  -> deterministic or consumer-defined contribution ordering
  -> prompt budget and truncation policy
  -> final prompt
```

Instructions and Contexts may originate from:

- Inline content.
- `AGENTS.md`.
- `CLAUDE.md`.
- `README.md`.
- `llms.txt`.
- Selected documentation.
- A source-relative locator.

The consumer owns prompt budgeting, truncation, and final ordering policy.

### Skill consumer

```text
resolved Skill
  -> verified Skill declaration source
  -> verified Skill directory
  -> Agent Skills runtime registration
  -> Skill prompt, resource, and script operations
```

A Skill included by several Collections retains one Skill identity.

Collection membership determines whether the Skill is selected for a particular experience. It does not create another runtime identity.

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

- Server identity derived from Artifact identity.
- Runtime grouping derived from active resolution scope.
- Source and Definition versioning.
- Connection invalidation.
- Tool discovery.
- Resource discovery.
- Prompt discovery.
- Completion support.
- Policy evaluation.
- Approval handling.
- OAuth flows.
- Client credential flows.
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

Workflow execution is owned by a dedicated runtime consumer.

The runtime decides whether all target nodes must be ready before execution begins.

It does not silently substitute another target when a node target is unavailable.

### Runtime implementation boundaries

Declaration resolution remains limited to:

```text
available
unavailable
ambiguous
```

Runtime readiness and resource materialization remain consumer concerns.

Explicit runtime request ordering is separate from declaration composition ordering.

Built-in package hydration does not require MCP runtime connection invalidation within the scope of this design.

## Technical architecture

### Root

A Root is the identity and resolution boundary for a repository or application domain.

It owns:

- Registered Sources.
- Source discovery configuration.
- Source refresh state.
- Immutable Definitions.
- Source-backed Artifacts.
- Local Artifact state.
- Symbolic identity lookup.

Generic Artifact Store Root creation remains domain-neutral. Application Root
lifecycle owns user-Root baseline establishment and startup reconciliation
because baseline Collections are application-domain policy.

The semantic lookup key is:

```text
(type, logicalName)
```

Examples:

```text
skill/code-review
agent/reviewer
mcp/github
collection/repository-review
workflow/change-workflow
```

A Root may contain multiple declaration occurrences with the same semantic identity.

### Source

A Source is a physical content origin.

Current Source kinds are:

```text
fs-directory
embedded-directory
managed-directory
```

A Source keeps physical configuration separate from declaration discovery.

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

### Definition

A Definition is the immutable normalized semantic result of decoding a declaration.

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

The Definition body contains the complete canonical declaration structure.

Definition digests support:

- Integrity.
- Equality.
- Change detection.
- Immutable persistence.
- Runtime versioning.
- Managed publication verification.

### Artifact

An Artifact is the local addressable record for a declaration occurrence.

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

A contained declaration has a source occurrence within its containing document. That occurrence may use a stable named structural position.

The structural occurrence identifies where the declaration is written. It does not make its containing Collection, Agent, Team, Workflow, or Workspace an owner.

Protected built-in Artifacts are exceptions to ordinary local mutation. Mutable consumer settings use explicit external overlays.

### Artifact Store boundary

Artifact Store persists:

- Sources.
- Definitions.
- Artifacts.
- Source bindings.
- Source refresh and integrity state.
- Artifact-local state.

Artifact Store does not infer:

- Collection ownership.
- Reverse membership.
- Cascading deletion.
- Generic composition lifecycle.
- Runtime activation from Collection membership.

A source-backed Artifact must not be purged while its declaration remains
available in a Source. Purge must be coupled to source mutation or reconciliation.

For user-managed Collections, editing changes only the source-backed Collection declaration body.

A management view may derive membership by:

- Reading Collection declarations.
- Resolving their composition entries.
- Maintaining a domain-owned cache.

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

A missing resource affects Artifact readiness and runtime materialization. It does not invalidate an otherwise valid containing composition declaration.

### Resolver

The Resolver is responsible for:

- Root-scoped symbolic lookup.
- Located declaration selection.
- Type and name confirmation.
- MCP source-entry selection.
- Alias-to-terminal resolution.
- Duplicate identity detection.
- Collection expansion.
- Agent expansion.
- Team expansion.
- Loop resolution.
- Workflow target resolution.
- Workspace root resolution.
- Nested Workspace rejection.
- Cycle detection.
- Resolution-depth limits.
- Resolution-node limits.
- Deterministic normalized ordering.
- Preservation of relationship-level status.

The Resolver returns:

- A typed graph.
- Composition occurrences.
- Target ArtifactRefs.
- Availability or ambiguity status.
- Diagnostics needed by capability consumers.
- A cycle-safe flattened capability plan suitable for frontend transport.
- A common completeness helper for strict consumers.

It does not execute the graph.

### Workspace refresh architecture

Resolution operates against indexed declaration state and remains read-only.

`RefreshWorkspace` is the explicit operation that:

- Computes reachable local locators.
- Extends or updates applicable Source discovery.
- Refreshes only the Workspace Source and Sources whose reachable local locator closure changed.
- Rebuilds affected Definitions and Artifacts.
- Leaves unrelated Source state unchanged.

Workspace read and planning operations consume this indexed state without mutating it.

## Implementation consequences and finalized decisions

### External references versus contained declarations

The implementation must classify composition entries before emitting nested Artifacts.

It must ensure that:

- Locator-bearing external Skill and MCP entries remain reference edges.
- External entries do not become source subresource Artifacts.
- Complete contained declarations receive stable named structural positions.
- Located targets confirm the requested type and name.
- Command locators remain executable declaration data.
- Common relationship annotations do not override target state.

Existing locator-bearing Collection members that were previously interpreted as Collection-specific child declarations require migration to external reference semantics.

### Member-level resolution

The resolver must preserve every declared composition occurrence.

A missing, ambiguous, disabled, invalid, cyclic, or locator-unresolvable target affects only that relationship path.

The implementation must not fail the entire containing Collection when the first unavailable relationship is encountered.

Consumers must apply an explicit completeness policy.

### Managed authoring

Managed authoring must provide:

- Source-backed editable Collections.
- Direct membership updates.
- Revision-aware deletion guards.
- Explicit Collection selection.
- Skill baseline provisioning.
- MCP baseline provisioning.
- Managed Skill publication.
- Managed MCP server publication.
- Managed MCP policy publication.
- Attach and detach operations.
- Package removal that does not alter unrelated targets.

Managed APIs should create external members unless a specialized contained-authoring flow is introduced.

### MCP identity and local configuration

Migrating a reference must preserve MCP installation-local configuration only when it still identifies the same independently declared MCP identity.

Distinct MCP configurations must use distinct logical names rather than membership-specific overrides.

### Built-in packages

Built-in implementation must support both:

- Independently declared package artifacts grouped by Collections.
- Complete contained MCP and MCP policy declarations authored inside built-in Collections.

Hydration must not rewrite contained declarations merely to fit an external-reference implementation model.

### Finalized semantic decisions

The following decisions are authoritative:

- Collection, Agent, Team, Skill tool, and Workspace arrays have no caller-defined ordering semantics.
- Capability output uses deterministic normalized ordering.
- Runtime request order is independent of declaration array order.
- Managed package replacement requires explicit replacement intent.
- Managed Skill and MCP authoring always receives an explicit Collection.
- Baseline Collections are selectable destinations, not defaults or fallbacks.
- Application Root lifecycle, rather than generic Artifact Store Root creation,
  provisions and reconciles baseline Collections.
- Nested Workspace references resolve as unavailable.
- Located resolution is read-only.
- `RefreshWorkspace` owns reachable-local-locator discovery updates.
- MCP `server` participates in source target selection.
- A non-command source-selected MCP is a pure terminal alias and cannot define
  local `include` rules.
- Resource claims do not suppress physical declaration targets.
- Source-selected aliases resolve to terminal Artifacts.
- Multiple aliases to one terminal Artifact count as one symbolic target.
- Built-in contained MCP and MCP policy declarations remain contained.
- Package hydration affects only changed packages unless topology recovery requires a reset.
- Runtime readiness remains outside declaration resolution.
- Artifact Store does not own generic composition relationships.
- Ordinary Artifact purge requires the source-backed Artifact to be missing.
- Workspace refresh does not refresh unrelated Root Sources.

## Current implementation status

### Status interpretation

- `Available` means an implementation path exists for the stated scope.
- `Pending` means approved completion, verification, or API-consolidation work remains.
- `Deferred` means the capability is intentionally outside current approved scope.
- `Not supported` means no supported behavior currently exists.
- `Removed` means the behavior is intentionally no longer exposed.
- Status does not assert successful build, test, migration, static-analysis, or Wails binding verification.

### Declaration platform and resolution

| Capability                                           | Status                                                                                |
| ---------------------------------------------------- | ------------------------------------------------------------------------------------- |
| Core artifact type vocabulary                        | Available                                                                             |
| Canonical JSON declarations                          | Available                                                                             |
| Canonical YAML declarations                          | Available                                                                             |
| JSON Schema contract validation                      | Available                                                                             |
| `type` and `apiVersion` schema dispatch              | Available                                                                             |
| Named nested declaration indexing                    | Available                                                                             |
| Named contained declarations                         | Available as source-backed nested Artifacts                                           |
| Stable named contained-declaration positions         | Available                                                                             |
| Anonymous declarations                               | Not supported                                                                         |
| Root-scoped symbolic lookup                          | Available                                                                             |
| Duplicate semantic identity detection                | Available                                                                             |
| Unlocated symbolic references                        | Available                                                                             |
| Located composition entries as reference edges       | Available                                                                             |
| Located Skill external references                    | Available through composition-entry classification and path resolution                |
| Located MCP external references                      | Available through composition-entry classification and path resolution                |
| Uniform locator semantics across composition types   | Available through read-only indexed resolution and explicit refresh-closure discovery |
| Same-Source Skill and MCP resource-claim suppression | Removed; source-selected declarations use terminal alias behavior                     |
| Member-level unavailable and ambiguous results       | Available in resolver, Collection capability plans, and Workspace capability plans    |
| Best-effort Collection display and expansion         | Available                                                                             |
| Collection resolution                                | Available with partial relationship behavior                                          |
| Agent resolution                                     | Available with relationship-level partial results; execution runtime is deferred      |
| Team resolution                                      | Available with relationship-level partial results; execution runtime is deferred      |
| Loop resolution                                      | Available with relationship-level partial results; execution runtime is deferred      |
| Workflow structure and target resolution             | Available with relationship-level partial results; execution runtime is deferred      |
| Workspace resolution                                 | Available with partial capability reporting and explicit strict runtime selection     |
| Workspace completeness policy                        | Available through `WorkspaceRuntimeSelection.RequireComplete`                         |
| Generic direct composition capability plans          | Available through cycle-safe flattened resolver plans                                 |
| Generic completeness policy                          | Available through the shared resolver completeness helper                             |
| Source-selected MCP aliases                          | Available as pure aliases; non-command aliases cannot define local `include` rules    |
| MCP policy declaration contract                      | Available, including concrete policy body validation                                  |
| Local path declaration locators                      | Available for supported locator targets                                               |
| Exact contained occurrence resolution                | Available through stable structural subresource selection                             |

### Sources, persistence, and resource verification

| Capability                           | Status                                                           |
| ------------------------------------ | ---------------------------------------------------------------- |
| Filesystem Sources                   | Available                                                        |
| Embedded Sources                     | Available                                                        |
| Managed Sources                      | Available                                                        |
| Source-backed Definition persistence | Available                                                        |
| Source-backed Artifact persistence   | Available                                                        |
| Source refresh state                 | Available                                                        |
| Definition digest verification       | Available                                                        |
| Resource generation verification     | Available                                                        |
| User-managed Source provisioning     | Available through the Workspace consumer API                     |
| Explicit managed package replacement | Available                                                        |
| Equivalent publication refresh skip  | Available when Source and expected Artifact state are current    |
| Authoritative managed discovery      | Available for managed Collection, Skill, MCP, and policy Sources |
| Managed declaration locator cleanup  | Available after managed package removal                          |
| Source-backed Artifact purge guard   | Available; ordinary purge requires a missing Artifact            |

### Physical format support

| Capability                          | Status    |
| ----------------------------------- | --------- |
| `AGENTS.md` support                 | Available |
| `CLAUDE.md` support                 | Available |
| `README.md` support                 | Available |
| `llms.txt` support                  | Available |
| Documentation Context discovery     | Available |
| `SKILL.md` support                  | Available |
| Direct Skill directory registration | Available |
| Direct `SKILL.md` registration      | Available |
| `.mcp.json` support                 | Available |
| `mcp.json` support                  | Available |
| Canonical MCP declarations          | Available |
| Source-selected MCP declarations    | Available |
| `AGENT.md` support                  | Available |
| `*.agent.md` support                | Available |

### Workspace capability planning

| Capability                                    | Status                                                   |
| --------------------------------------------- | -------------------------------------------------------- |
| Read-only Workspace resolution and planning   | Available; Source mutation remains in `RefreshWorkspace` |
| Workspace prompt planning                     | Available with partial capability-occurrence reporting   |
| Workspace Skill planning                      | Available with partial capability-occurrence reporting   |
| Workspace MCP planning                        | Available with partial capability-occurrence reporting   |
| Explicit Workspace capability completeness    | Available through `RequireComplete`                      |
| Consistent explicit ArtifactRef deduplication | Available for prompt, Skill, and MCP selections          |
| Generic Artifact composition inspection       | Available through a frontend-safe flattened plan         |
| Source-targeted Workspace refresh             | Available; unrelated Root Sources are not refreshed      |

### Collections and managed authoring

| Capability                                                         | Status                                                                     |
| ------------------------------------------------------------------ | -------------------------------------------------------------------------- |
| Portable mixed Collection declarations                             | Available in canonical source documents                                    |
| Domain-specific editable managed Collections                       | Available through Skill and MCP Collection APIs                            |
| Collection deletion guard                                          | Available with direct-member protection                                    |
| Canonical Skill Collection declarations                            | Available as external grouping declarations                                |
| Application-provisioned Skill baseline Collection per user Root    | Available through application Root provisioning and startup reconciliation |
| Application-provisioned MCP baseline Collection per user Root      | Available through application Root provisioning and startup reconciliation |
| Baseline rename, disable, runtime-disable, and deletion protection | Available through application-facing Collection and Workspace APIs         |
| Explicit Collection selection for managed Skill authoring          | Available                                                                  |
| Explicit Collection selection for managed MCP authoring            | Available                                                                  |
| Managed Skill creation in a Collection                             | Available; an explicit editable Collection is required                     |
| Managed MCP creation in a Collection                               | Available; an explicit editable Collection is required                     |
| Managed MCP policy creation in a Collection                        | Available; an explicit editable Collection is required                     |
| Attach an existing Skill or MCP to a Collection through user APIs  | Available                                                                  |
| Detach a member without deleting its target through user APIs      | Available                                                                  |
| Direct compatible Artifact membership views                        | Available with direct-member relationship status                           |
| Explicit replacement protection for managed create flows           | Available                                                                  |
| User-facing standalone Skill creation                              | Removed from the managed authoring flow                                    |
| Individual editing of contained Skills                             | Not supported; managed Skill APIs create independent package Artifacts     |
| Managed declaration publication                                    | Available for Collections, Skills, MCP servers, and MCP policies           |
| Managed MCP server publication                                     | Available                                                                  |
| Managed package removal                                            | Available                                                                  |
| Mixed-Collection frontend authoring                                | Deferred                                                                   |

### Skill and built-in package support

| Capability                                                       | Status                                                            |
| ---------------------------------------------------------------- | ----------------------------------------------------------------- |
| Managed Skill packages                                           | Available                                                         |
| Built-in Skill packages                                          | Available with independently discovered Skill Artifacts           |
| Built-in MCP packages                                            | Available with contained MCP and policy declaration Artifacts     |
| Built-in package-scoped hydration                                | Available                                                         |
| Known built-in package byte verification and repair              | Available                                                         |
| Stale known built-in package removal                             | Available                                                         |
| Built-in ArtifactRef stability across ordinary package hydration | Available for unchanged packages                                  |
| Built-in ArtifactRef stability across topology migration         | Not guaranteed; affected references may be intentionally replaced |

### MCP platform and runtime

| Capability                                | Status    |
| ----------------------------------------- | --------- |
| MCP policy declarations                   | Available |
| MCP installation-local data               | Available |
| MCP secret references                     | Available |
| MCP runtime configuration                 | Available |
| MCP runtime connection management         | Available |
| MCP tool discovery and invocation support | Available |
| MCP resource support                      | Available |
| MCP prompt support                        | Available |
| MCP completion support                    | Available |
| MCP policy evaluation                     | Available |
| MCP approval handling                     | Available |
| MCP OAuth and client credential flows     | Available |
| MCP secret redaction                      | Available |

### Pending completion work

| Capability or concern                     | Status  | Remaining work                                                                                                |
| ----------------------------------------- | ------- | ------------------------------------------------------------------------------------------------------------- |
| Build and acceptance verification         | Pending | Run full tests, static analysis, migration coverage, and Wails binding generation                             |
| Managed create operation durability       | Pending | Skill, MCP, and policy membership and package publication remain separate source-side operations              |
| Protected Source undeclared content sweep | Pending | Known packages are repaired, but manually inserted untracked protected packages are not automatically removed |
| Bulk resource verification efficiency     | Pending | Expose bounded verification-session reuse for bulk Skill and Workspace materialization                        |

### Deferred and unsupported capabilities

#### Source, locator, and cross-Root capabilities

| Capability                                                   | Status           |
| ------------------------------------------------------------ | ---------------- |
| Cross-Root Workspace imports                                 | Not supported    |
| Automatic built-in visibility in user Workspaces             | Not supported    |
| Plugin manifests and external package distribution workflows | Deferred         |
| Git locators                                                 | Support deferred |
| Git archive materialization                                  | Support deferred |
| URL locators                                                 | Support deferred |
| Package locators                                             | Support deferred |
| Archive and zip Sources                                      | Support deferred |

#### Authoring and frontend scope

| Capability                                                                                    | Status   |
| --------------------------------------------------------------------------------------------- | -------- |
| Mixed-Collection frontend authoring                                                           | Deferred |
| Managed authoring APIs for Agent, Team, Loop, Workflow, Tool, Model, Instruction, and Context | Deferred |
| Direct standalone management UI for Agent, Team, Loop, and Workflow composition plans         | Deferred |

#### Execution runtimes

| Capability                        | Status                              |
| --------------------------------- | ----------------------------------- |
| Tool execution                    | Owned by a future Tool consumer     |
| Model execution                   | Owned by a future Model consumer    |
| Agent execution                   | Owned by a future Agent runtime     |
| Team execution                    | Owned by a future Team runtime      |
| Loop execution                    | Owned by a future execution runtime |
| Workflow scheduling and execution | Owned by a future Workflow runtime  |
