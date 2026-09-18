# Artifact-backed Agent Store HLD

Status: Proposed

Normative foundations:

- [Portable Artifact Declaration Contracts HLD](./artifact_contracts_hld.md)
- [Artifact Store, Resolution, and Ecosystem HLD](./artifacts_hld.md)

If this HLD conflicts with either normative foundation, the Artifact declaration and ecosystem HLDs take precedence.

- [1. Purpose](#1-purpose)
- [2. Goals and scope](#2-goals-and-scope)
  - [2.1 Goals](#21-goals)
  - [2.2 In scope](#22-in-scope)
  - [2.3 Boundary](#23-boundary)
- [3. Requirements](#3-requirements)
  - [3.1 Artifact foundation requirements](#31-artifact-foundation-requirements)
  - [3.2 Agent Collection requirements](#32-agent-collection-requirements)
  - [3.3 Managed Agent publication requirements](#33-managed-agent-publication-requirements)
  - [3.4 Resolution and planning requirements](#34-resolution-and-planning-requirements)
  - [3.5 Assistant Preset replacement requirements](#35-assistant-preset-replacement-requirements)
- [4. Design principles and invariants](#4-design-principles-and-invariants)
  - [4.1 Artifact contracts remain authoritative](#41-artifact-contracts-remain-authoritative)
  - [4.2 Agent Store is a domain facade](#42-agent-store-is-a-domain-facade)
  - [4.3 Portable behavior and launch defaults remain separate](#43-portable-behavior-and-launch-defaults-remain-separate)
  - [4.4 Agent Collections are backed by Plugins](#44-agent-collections-are-backed-by-plugins)
  - [4.5 Relationships do not persist direct Store references](#45-relationships-do-not-persist-direct-store-references)
  - [4.6 Release identity does not alter Artifact identity](#46-release-identity-does-not-alter-artifact-identity)
  - [4.7 Resolution, enablement, and readiness remain separate](#47-resolution-enablement-and-readiness-remain-separate)
  - [4.8 Collection membership does not establish ownership](#48-collection-membership-does-not-establish-ownership)
  - [4.9 Composition order has no semantic authority](#49-composition-order-has-no-semantic-authority)
- [5. Desired user experience](#5-desired-user-experience)
- [6. Conceptual architecture](#6-conceptual-architecture)
  - [6.1 Terminology](#61-terminology)
- [7. Portable Agent declaration usage](#7-portable-agent-declaration-usage)
  - [7.1 Canonical Agent example](#71-canonical-agent-example)
  - [7.2 Agent field ownership](#72-agent-field-ownership)
  - [7.3 Member behavior](#73-member-behavior)
  - [7.4 Agent Markdown behavior](#74-agent-markdown-behavior)
- [8. Agent Store data model and ownership](#8-agent-store-data-model-and-ownership)
  - [8.1 Source-backed state](#81-source-backed-state)
  - [8.2 Agent publication descriptor](#82-agent-publication-descriptor)
  - [8.3 Derived state](#83-derived-state)
  - [8.4 Local references and portable references](#84-local-references-and-portable-references)
- [9. Agent publication identity, releases, and locators](#9-agent-publication-identity-releases-and-locators)
  - [9.1 Portable Agent identity](#91-portable-agent-identity)
  - [9.2 Managed publication identity](#92-managed-publication-identity)
  - [9.3 Release labels](#93-release-labels)
  - [9.4 Managed Source layout](#94-managed-source-layout)
  - [9.5 Relationship locator rules](#95-relationship-locator-rules)
  - [9.6 Immutable publication behavior](#96-immutable-publication-behavior)
- [10. Agent Collections](#10-agent-collections)
  - [10.1 Agent Collection domain](#101-agent-collection-domain)
  - [10.2 Agent Collection declaration](#102-agent-collection-declaration)
  - [10.3 Managed Collection restrictions](#103-managed-collection-restrictions)
  - [10.4 Baseline Agent Collection](#104-baseline-agent-collection)
  - [10.5 Membership semantics](#105-membership-semantics)
  - [10.6 Collection deletion](#106-collection-deletion)
- [11. Primary workflows](#11-primary-workflows)
  - [11.1 Create an Agent Collection](#111-create-an-agent-collection)
  - [11.2 Create a managed Agent publication](#112-create-a-managed-agent-publication)
  - [11.3 Attach an existing Agent](#113-attach-an-existing-agent)
  - [11.4 Read and plan an Agent](#114-read-and-plan-an-agent)
  - [11.5 Refresh an Agent closure](#115-refresh-an-agent-closure)
  - [11.6 Replace or publish a new release](#116-replace-or-publish-a-new-release)
  - [11.7 Detach or delete](#117-detach-or-delete)
- [12. Agent resolution and planning](#12-agent-resolution-and-planning)
  - [12.1 Plan inputs](#121-plan-inputs)
  - [12.2 Resolution behavior](#122-resolution-behavior)
  - [12.3 Agent plan](#123-agent-plan)
  - [12.4 Partial and strict plans](#124-partial-and-strict-plans)
  - [12.5 Execution boundary](#125-execution-boundary)
- [13. Launch defaults and conversation compatibility](#13-launch-defaults-and-conversation-compatibility)
  - [13.1 Why launch defaults are separate](#131-why-launch-defaults-are-separate)
  - [13.2 Launch default model](#132-launch-default-model)
  - [13.3 Stable member keys](#133-stable-member-keys)
  - [13.4 Launch input precedence](#134-launch-input-precedence)
  - [13.5 Legacy-compatible conversation projection](#135-legacy-compatible-conversation-projection)
  - [13.6 MCP launch context](#136-mcp-launch-context)
- [14. Validation and runtime readiness](#14-validation-and-runtime-readiness)
  - [14.1 Validation layers](#141-validation-layers)
  - [14.2 Managed publication validation](#142-managed-publication-validation)
  - [14.3 Skill readiness](#143-skill-readiness)
  - [14.4 MCP validation](#144-mcp-validation)
  - [14.5 Enablement policy](#145-enablement-policy)
- [15. Catalog, listing, and enablement](#15-catalog-listing-and-enablement)
  - [15.1 Collection views](#151-collection-views)
  - [15.2 Agent views](#152-agent-views)
  - [15.3 Listing and pagination](#153-listing-and-pagination)
  - [15.4 Effective selection](#154-effective-selection)
- [16. Built-in Agent packages](#16-built-in-agent-packages)
  - [16.1 Package layout](#161-package-layout)
  - [16.2 Built-in validation](#162-built-in-validation)
  - [16.3 Protected Root behavior](#163-protected-root-behavior)
  - [16.4 Package hydration](#164-package-hydration)
- [17. Workspace and consumer integration](#17-workspace-and-consumer-integration)
  - [17.1 Workspace selection](#171-workspace-selection)
  - [17.2 Prompt and capability consumers](#172-prompt-and-capability-consumers)
  - [17.3 Runtime boundary](#173-runtime-boundary)
- [18. Assistant Preset migration and removal](#18-assistant-preset-migration-and-removal)
  - [18.1 Migration mapping](#181-migration-mapping)
  - [18.2 Reference conversion](#182-reference-conversion)
  - [18.3 Migration phases](#183-migration-phases)
    - [Phase 1: Build Agent Store](#phase-1-build-agent-store)
    - [Phase 2: Convert built-in content](#phase-2-convert-built-in-content)
    - [Phase 3: Import user content](#phase-3-import-user-content)
    - [Phase 4: Switch consumers](#phase-4-switch-consumers)
    - [Phase 5: Remove the legacy Store](#phase-5-remove-the-legacy-store)
  - [18.4 Removal gate](#184-removal-gate)
- [19. Implementation structure](#19-implementation-structure)
  - [19.1 Components](#191-components)
  - [19.2 Concurrency and recovery](#192-concurrency-and-recovery)
  - [19.3 Test requirements](#193-test-requirements)
- [20. Assistant Preset functionality parity](#20-assistant-preset-functionality-parity)
- [21. Differences with respect to Assistant Presets](#21-differences-with-respect-to-assistant-presets)
- [22. Current and proposed status](#22-current-and-proposed-status)

## 1. Purpose

This HLD defines an Agent Store domain service built on the existing Artifact Store, portable Agent contract, typed resolver, managed Source publishing, and protected built-in Root.

The complete flow is:

```text
Agent declaration or managed Agent package
  -> Source decoder
  -> immutable Definition
  -> source-backed Agent Artifact
  -> typed Agent resolution graph
  -> Agent plan
  -> conversation adapter or future Agent runtime
```

The Agent Store replaces the legacy Assistant Preset Store as the application surface for:

- User-managed and built-in Agents.
- Agent-only collections.
- Immutable managed Agent publications.
- Agent catalog and enablement.
- Model, Tool, Skill, MCP, Text, Plugin, and nested Agent selection.
- Legacy-compatible conversation launch defaults.
- Workspace Agent planning.

The Agent Store does not create a second persistence system for portable Agent declarations. Portable declarations remain authoritative in Artifact Store Sources.

## 2. Goals and scope

### 2.1 Goals

The Agent Store must provide:

- One Agent management surface over source-backed Agent Artifacts.
- Agent-only collections without reintroducing the removed `collection` Artifact type.
- Managed Agent creation with explicit destination collection selection.
- User and built-in Agent catalog views.
- Immutable publication behavior comparable to Assistant Preset versions.
- Exact occurrence selection through locators.
- Named and protected built-in lookup where locators are not appropriate.
- Typed Agent plans with relationship-level diagnostics.
- Universal Artifact enablement for Agent and collection state.
- A clean boundary between portable Agent behavior and runtime launch defaults.
- A migration path that preserves Assistant Preset behavior without persisting legacy direct references.
- Package-scoped built-in hydration in the shared protected Root.
- No Agent execution dependency in the management service.

### 2.2 In scope

This HLD defines:

- The Agent Store domain facade.
- Agent publication and catalog concepts.
- Agent-only collections backed by `plugin` Artifacts.
- Managed Source layout and publication behavior.
- Agent release labels outside portable declarations.
- Agent resolution and plan projections.
- Launch-default storage required to replace Assistant Presets.
- Built-in Agent package composition and hydration.
- Workspace and conversation-consumer integration.
- Assistant Preset migration and removal.
- Current implementation boundaries and pending work.

### 2.3 Boundary

This HLD does not:

- Redefine the portable `agent` contract.
- Add `apiVersion`, `version`, or schema fields to Agent declarations.
- Reintroduce `collection`, `instruction`, or `context` Artifact types.
- Persist `ArtifactRef`, model preset references, Tool references, or MCP runtime server IDs in portable declarations.
- Make Plugin membership Store ownership.
- Make Artifact enablement part of resolution.
- Execute an Agent, Model, Tool, Skill, MCP server, Loop, or Workflow.
- Store secrets, OAuth tokens, MCP connection state, or execution history.
- Make baseline or built-in Agent Collections automatically active.
- Support arbitrary cross-Root imports.

## 3. Requirements

### 3.1 Artifact foundation requirements

The Agent Store must:

- Use existing `agentv1`, `pluginv1`, `textv1`, `modelv1`, `toolv1`, `skillv1`, `mcpv1`, `loopv1`, and `workflowv1` contracts.
- Use Artifact Store Roots, Sources, Definitions, Artifacts, resources, and local state.
- Use `ResolveAgent` and `ResolvePlugin` instead of implementing another symbolic resolver.
- Preserve named, located, contained, and selector relationship behavior.
- Keep ordinary reads and plans free of Source mutation.
- Use explicit refresh for locator or selector discovery changes.
- Return local `ArtifactRef` values only as authorized application references, never as portable declaration relationships.
- Preserve relationship occurrence identity and diagnostics.

### 3.2 Agent Collection requirements

The Agent Store must:

- Represent a user-facing Agent Collection with a source-backed `plugin` Artifact.
- Restrict managed Agent Collection members to Agents.
- Use external Agent relationships for independently managed Agent publications.
- Use located external relationships by default.
- Permit one Agent publication to appear in multiple Agent Collections.
- Permit a relationship to remain declared when its Agent is unavailable.
- Keep collection enablement separate from Agent enablement.
- Require an explicit editable Agent Collection when creating a managed Agent.
- Provide an application-provisioned baseline Agent Collection.
- Keep the baseline collection explicit, editable, non-deletable, non-renamable, and non-automatic.
- Prevent deletion of a collection with direct declared members.

### 3.3 Managed Agent publication requirements

A managed Agent publication must:

- Contain one portable Agent declaration.
- Be published as an independent package.
- Have a stable package occurrence and locator.
- Optionally contain a typed Agent Store publication descriptor.
- Support an Agent Store release label without adding that label to the portable Agent declaration.
- Be immutable by default after successful creation.
- Require explicit replacement for non-equivalent bytes at the same package occurrence.
- Support side-by-side releases through distinct package occurrences and locators.
- Use universal `Artifact.Enabled` state.
- Preserve `displayName`, `description`, built-in provenance, creation metadata, and modification metadata in the Agent Store projection.
- Avoid direct references to legacy model, Tool, Skill, or MCP Store identities.

### 3.4 Resolution and planning requirements

Agent planning must:

- Begin from one authorized Agent occurrence.
- Use the Root of that occurrence as the default lookup Root.
- Apply current Root, protected built-in Root, and registered fallback behavior from the Artifact ecosystem HLD.
- Resolve located members strictly without name fallback.
- Preserve Tool `overrides.autoExecute`.
- Preserve Model `overrides.includeSystemPrompt`.
- Preserve Skill `use.mode`.
- Preserve Agent `prompt`, Text members, and optional program relationships.
- Preserve unavailable and ambiguous members.
- Detect cycles and enforce resolver limits.
- Support partial and strict completeness.
- Keep declaration availability, Artifact enablement, resource readiness, and runtime readiness distinct.
- Never invoke a selected capability.

### 3.5 Assistant Preset replacement requirements

The replacement must carry forward, in principle:

- Bundle creation, update, enablement, listing, filtering, and deletion.
- Agent-only bundle membership.
- Immutable named and released items.
- Preset enablement.
- Built-in read-only content with mutable local enablement.
- Display name and description.
- Starting composer text.
- Model selection and model system prompt preference.
- Tool selection, auto-execution preference, and Tool user-input defaults.
- Skill selection and preload mode.
- MCP server, Tool, resource, resource-template, prompt, digest, argument, and policy selections.
- Deterministic listing and pagination.
- Validation of duplicate selections and malformed launch defaults.
- Strict reference and readiness validation where required by a consumer.
- Existing disabled-resource behavior through an explicit compatibility policy.

Runtime-only starter values must not be forced into the portable Agent declaration. They are represented by the Agent publication launch descriptor and copied into one-run launch input.

## 4. Design principles and invariants

### 4.1 Artifact contracts remain authoritative

The Agent Store consumes the existing portable contracts. It does not define an alternate Agent wire format.

A valid Agent declaration therefore begins with:

```yaml
type: agent
name: bug-investigator
```

It does not contain:

```text
apiVersion
schemaVersion
version
collection
instruction
context
target
```

Agent Store application policy may restrict managed authoring more narrowly than the portable contract. It must not change the meaning of a valid portable declaration.

### 4.2 Agent Store is a domain facade

The Agent Store coordinates existing components:

```text
Agent Store
  -> managed Source publisher
  -> Artifact Store
  -> Agent and Plugin resolvers
  -> verified resource reader
  -> Agent catalog projection
  -> launch-plan adapter
```

It is not a parallel declaration database.

The Agent Store may own:

- Managed package policy.
- Agent Collection restrictions.
- Publication release metadata.
- A typed launch-default sidecar.
- Rebuildable catalog indexes.
- Consumer-specific readiness projections.

It does not own duplicate copies of Agent declaration bodies.

### 4.3 Portable behavior and launch defaults remain separate

Portable Agent behavior includes:

```text
Agent prompt
Text members
Model relationships
Tool relationships
Skill relationships and modes
MCP relationships
Nested Plugin or Agent relationships
Loop or Workflow relationship
```

Launch defaults include starter values copied into a future run:

```text
Editable initial text
Preferred model relationship when several models exist
Tool user argument schema instances
Runtime presentation order
MCP selected capabilities and arguments
Compatibility preference for the caller's current model
```

Launch defaults are not embedded in `metadata` and are not added to `agentv1`. They use a separate, explicit Agent Store package contract.

### 4.4 Agent Collections are backed by Plugins

An Agent Collection is a domain view over a `plugin` Artifact whose managed direct members are Agents.

```text
Agent Collection
  -> Plugin Artifact
  -> direct Agent relationships
```

There is no portable `type: collection`.

The term Agent Collection is retained because it is the user-facing replacement for an Assistant Preset bundle. The backing Artifact remains a Plugin everywhere in the declaration and resolver layers.

### 4.5 Relationships do not persist direct Store references

Portable Agent and Plugin declarations use:

- `name` for symbolic relationships.
- `scope: builtin` for protected built-in-only lookup.
- `locator` for an exact source occurrence.
- `parameters` for a contained declaration.
- `base` and filters for selectors.

They do not persist:

- `artifact.ArtifactRef`.
- `modelpreset.ModelPresetRef`.
- `tool.ToolRef`.
- `mcpServer.ServerID`.
- Root IDs.
- Source IDs.

A local API may accept an authorized `ArtifactRef` to identify an existing target. Before publication, the Agent Store converts it into a valid named, scoped, or located relationship.

### 4.6 Release identity does not alter Artifact identity

Portable Agent identity remains:

```text
(agent, name)
```

An Agent Store release label is management metadata associated with one managed package occurrence.

It is not:

- A portable Agent field.
- A schema version.
- Part of symbolic Agent identity.
- Used by the generic resolver.
- A substitute for a locator.

Two releases of the same Agent can coexist because they have distinct declaration occurrences and locators. An unlocated symbolic lookup may become ambiguous when both are in one Root.

### 4.7 Resolution, enablement, and readiness remain separate

The Agent Store distinguishes:

```text
Resolution availability
  -> can one declaration target be identified?

Artifact enablement
  -> has the user locally enabled this Artifact?

Resource readiness
  -> can required verified source material be opened?

Runtime readiness
  -> can the current consumer use the target now?
```

A disabled Agent or member still resolves.

A catalog or runtime consumer may reject disabled Artifacts, but the resolver must not hide or replace them.

### 4.8 Collection membership does not establish ownership

Adding an Agent to a collection does not:

- Copy the Agent.
- Enable the Agent.
- Configure the Agent.
- Change the Agent declaration.
- Transfer package ownership.
- Prevent membership in another collection.

Removing a relationship does not delete its Agent.

Deleting an Agent publication does not silently remove relationships from collections.

### 4.9 Composition order has no semantic authority

`agent.members` and `plugin.members` are semantically unordered.

Source-array position must not define:

- Relationship identity.
- Target precedence.
- Runtime execution order.
- Collection ownership.
- Release selection.
- Cache identity.

A launch descriptor may preserve a presentation or request-array order by stable member key for Assistant Preset compatibility. That order does not affect Artifact resolution.

## 5. Desired user experience

A repository can author Agents directly:

```text
checkout-service/
  AGENTS.md

  agents/
    reviewer.agent.yaml
    security-reviewer.agent.md

  plugins/
    engineering-agents.yaml

  skills/
    reviewing-code/
      SKILL.md

  workflows/
    change-workflow.yaml

  workspace.yaml
```

A managed Agent Source uses independent collection and Agent packages:

```text
managed-agent-source/
  agent-plugins/
    <collection-key>/
      plugin.yaml

  agents/
    <publication-key>/
      agent.yaml
      publication.json
```

The default user flow is:

```text
Create or select an Agent Collection
  -> author an Agent declaration
  -> optionally provide launch defaults and a release label
  -> publish a located Agent relationship to the collection
  -> publish the independent Agent package
  -> refresh the affected managed Source
  -> resolve the Agent
  -> return Agent, collection, and plan views
```

The read flow is:

```text
Select Agent from catalog or Workspace
  -> resolve the exact Agent occurrence
  -> expand declared relationships
  -> apply enablement and readiness policy
  -> build an Agent plan
  -> optionally project a conversation starter
```

Unavailable relationships remain visible with diagnostics.

## 6. Conceptual architecture

```text
Repository or managed package
  -> Source entry
  -> physical-format decoder
  -> Agent or Plugin Definition
  -> source-backed Artifact
  -> Agent or Plugin resolver
  -> relationship graph
  -> Agent plan
  -> conversation adapter or future Agent runtime
```

Managed publication adds one application-side package resource:

```text
Agent package
  -> agent.yaml
     -> portable Agent Artifact

  -> publication.json
     -> Agent Store release and launch defaults
     -> not a portable Artifact
```

### 6.1 Terminology

| Term                 | Meaning                                                                      |
| -------------------- | ---------------------------------------------------------------------------- |
| Agent Artifact       | One source-backed occurrence of a portable `agent` declaration               |
| Agent publication    | One managed Agent package occurrence                                         |
| Publication key      | Local managed package identity used to address package storage               |
| Release label        | Optional Agent Store catalog label attached to one publication               |
| Agent Collection     | User-facing Agent-only collection backed by a `plugin` Artifact              |
| Collection member    | One direct Agent relationship in the backing Plugin                          |
| Member key           | Stable non-positional key for one direct Agent member relationship           |
| Agent plan           | Read-only projection of the resolved Agent graph                             |
| Launch defaults      | Package values copied into one-run launch input                              |
| Conversation starter | Compatibility projection for the existing conversation UI and inference path |
| Runtime readiness    | Consumer decision about whether a resolved target can currently be used      |

## 7. Portable Agent declaration usage

### 7.1 Canonical Agent example

```yaml
type: agent
name: bug-investigator
displayName: Bug Investigator
description: Investigates failures from logs, tests, code, and configuration.

labels:
  domain: engineering

prompt:
  mediaType: text/markdown
  content: |
    Investigate the supplied issue using repository evidence.
    Separate observations from assumptions and propose safe verification.

members:
  - type: text
    name: investigation-rules
    insert: instructions
    parameters:
      mediaType: text/markdown
      content: |
        Prefer the smallest safe change.
        Do not modify generated files.

  - type: text
    name: repository-architecture
    insert: user-message
    locator: ../docs/architecture.md
    parameters:
      mediaType: text/markdown

  - type: model
    name: reasoning
    overrides:
      includeSystemPrompt: true

  - type: tool
    name: searchfiles
    scope: builtin
    overrides:
      autoExecute: true

  - type: tool
    name: readfile
    scope: builtin
    overrides:
      autoExecute: true

  - type: skill
    name: bug-investigation
    locator: ../skills/bug-investigation/SKILL.md
    use:
      mode: active

  - type: skill
    name: grounded-local-work
    use:
      mode: instructions

  - type: mcp
    name: github

workflow:
  name: investigation-workflow
  locator: ../workflows/investigation-workflow.yaml
```

This declaration contains no Store identity and no Agent Store release label.

### 7.2 Agent field ownership

| Field                  | Owner                             | Meaning                                        |
| ---------------------- | --------------------------------- | ---------------------------------------------- |
| `type`                 | Agent declaration                 | Portable Artifact type                         |
| `name`                 | Agent declaration                 | Root-scoped logical name                       |
| `displayName`          | Agent declaration                 | User-facing name                               |
| `description`          | Agent declaration                 | User-facing description                        |
| `labels`               | Agent declaration                 | Source-authored classifications                |
| `prompt`               | Agent declaration                 | Agent-owned base prompt                        |
| `members`              | Agent declaration                 | Capability and composition relationships       |
| `loop`                 | Agent declaration                 | Optional Loop program relationship             |
| `workflow`             | Agent declaration                 | Optional Workflow program relationship         |
| `locator`              | Agent declaration or relationship | Source selection according to contract context |
| `overrides`            | Relationship                      | Model or Tool behavior for that relationship   |
| `use`                  | Relationship                      | Skill use mode                                 |
| `Enabled`              | Artifact record                   | Local generic enablement                       |
| Release label          | Agent Store publication           | Managed catalog label                          |
| Initial editor text    | Launch defaults                   | Editable starter text                          |
| MCP selected resources | Launch defaults or launch input   | One-run context template                       |

`prompt` must not be used to store legacy `StartingText`. The two fields have different meanings.

### 7.3 Member behavior

Agent members support the types authorized by the portable contract:

```text
text
model
tool
skill
mcp
mcp.policy
plugin
agent
```

Loop and Workflow are represented by the singular `loop` and `workflow` program slots, not as ordinary Agent members.

The Agent Store preserves:

```yaml
- type: tool
  name: searchfiles
  overrides:
    autoExecute: true
```

```yaml
- type: model
  name: reasoning
  overrides:
    includeSystemPrompt: true
```

```yaml
- type: skill
  name: reviewing-code
  use:
    mode: active
```

Skill use mode mapping is:

| Mode           | Meaning                                                        |
| -------------- | -------------------------------------------------------------- |
| `available`    | Skill is available without initial activation                  |
| `active`       | Agent runtime intends to preload the Skill                     |
| `instructions` | Agent runtime intends to materialize the Skill as instructions |

Instruction and context concepts use Text:

```text
Legacy instruction
  -> type=text, insert=instructions

Legacy informational context
  -> type=text, insert=user-message
```

Unknown `overrides` or `use` fields remain invalid.

### 7.4 Agent Markdown behavior

`AGENT.md` and `*.agent.md` use ordinary Agent front matter.

```markdown
---
type: agent
name: bug-investigator
displayName: Bug Investigator

prompt:
  mediaType: text/markdown
  content: |
    Investigate the requested failure using evidence.

members:
  - type: model
    name: reasoning
---

Use repository files, logs, and tests as evidence.
```

The physical-format adapter emits:

```text
Agent.prompt
  -> Investigate the requested failure using evidence.

Markdown body
  -> Text Artifact with insert=instructions
```

The Markdown body is not reclassified as `prompt`.

## 8. Agent Store data model and ownership

### 8.1 Source-backed state

The following state is owned by the Artifact Store:

```text
Root
Source
Definition
Agent Artifact
Plugin Artifact
Source binding
Verified resources
Artifact.Enabled
Artifact-local data
Source generation and diagnostics
```

The portable Agent and Plugin bodies are stored only as Definitions and Source content.

The Agent Store must not persist separate copies of:

- Agent members.
- Plugin members.
- Resolver results.
- Reverse collection memberships.
- Selector matches.
- Parent ownership.
- Runtime capability state.

### 8.2 Agent publication descriptor

A managed Agent package may include a strict Agent Store descriptor:

```text
AgentPublicationDescriptor {
  SchemaVersion

  ReleaseLabel?
  ImportedCreatedAt?
  ImportedModifiedAt?

  LaunchDefaults?
}
```

The descriptor:

- Is application-specific.
- Is not a portable Artifact declaration.
- Is not decoded as an Artifact.
- Is versioned independently from `agentv1`.
- Is included in the managed package fingerprint.
- Is read through verified package resources.
- Must not contain secrets or runtime connection state.
- Must not contain target `ArtifactRef` values.
- May be absent for repository-authored Agents.

A managed publication is therefore one package unit:

```text
Agent Definition digest
  + publication descriptor digest
  + verified package resources
  -> Agent publication fingerprint
```

The Agent Definition digest remains independent of the descriptor digest.

### 8.3 Derived state

The Agent Store derives:

- Agent Collection membership views.
- Reverse direct collection membership views.
- Agent publication summaries.
- Built-in provenance.
- Release labels.
- Relationship availability.
- Effective catalog selectability.
- Runtime readiness.
- Stable member keys.
- Pagination cursors.
- Conversation starter projections.

A cache may retain this data for performance. The cache is rebuildable and must not become the authority for declaration membership.

### 8.4 Local references and portable references

The following distinction is required:

| Reference                    | Allowed use                                                             |
| ---------------------------- | ----------------------------------------------------------------------- |
| `ArtifactRef`                | Authorized local API request or response                                |
| `name`                       | Portable symbolic relationship                                          |
| `name` plus `scope: builtin` | Portable protected built-in lookup                                      |
| `name` plus `locator`        | Portable exact occurrence relationship                                  |
| Member key                   | Agent publication descriptor reference to one direct Agent relationship |
| MCP runtime `ServerID`       | Transient runtime context only                                          |

It is valid for `GetAgent` or `PlanAgent` to receive an `ArtifactRef` returned by the local catalog.

It is invalid to serialize that `ArtifactRef` into an Agent or Plugin declaration.

Every caller-supplied `ArtifactRef` must be checked against the authorized current Root or an allowed protected built-in result.

## 9. Agent publication identity, releases, and locators

### 9.1 Portable Agent identity

The portable semantic identity is:

```text
(agent, name)
```

Identity is scoped by Root.

One Root may contain several occurrences of the same Agent identity. A named relationship is ambiguous when those occurrences resolve to different terminal Artifacts.

### 9.2 Managed publication identity

A managed publication has:

```text
Publication key
Agent logical name
Agent declaration occurrence
Optional release label
Exact locator
ArtifactRef after Source refresh
```

The publication key is a managed storage identity. It is not written into the Agent declaration.

A client normally identifies a managed publication by:

- Its returned `ArtifactRef`.
- Its publication key.
- A collection relationship occurrence.
- A compatibility key consisting of collection, Agent name, and release label.

The Agent Store must not assume that Agent name and release label are globally unique. Imported Assistant Presets from different bundles may have the same slug and version but different content.

### 9.3 Release labels

A release label preserves Assistant Preset version behavior without changing the portable contract.

```text
Assistant Preset version
  -> Agent Store publication release label
  -> distinct managed package occurrence
  -> distinct locator
```

Rules:

- A release label is optional for general repository Agents.
- A release label is required by the Assistant Preset compatibility importer.
- A release label is immutable for one publication.
- The label is displayed and filterable by the Agent catalog.
- The resolver does not inspect it.
- Selecting a release means selecting its exact occurrence, normally through a collection member locator.
- Publishing a new release creates a new package occurrence.
- Reusing an existing release label in another collection is permitted when the collection relationships identify exact occurrences.
- Duplicate Agent name and release combinations in one collection are rejected unless an explicit migration disambiguation policy is used.

### 9.4 Managed Source layout

The default managed layout is:

```text
managed-agent-source/
  agent-plugins/
    <collection-key>/
      plugin.yaml

  agents/
    <publication-key>/
      agent.yaml
      publication.json
```

All managed Agent collection and publication packages for one user Root are registered beneath a common managed Source boundary.

This permits safe relative locators such as:

```yaml
- type: agent
  name: bug-investigator
  locator: ../../agents/<publication-key>/agent.yaml
```

An implementation may shard managed Sources only when it can still represent and materialize collection relationships through supported locator adapters.

### 9.5 Relationship locator rules

When converting an existing target into a declaration relationship, the Agent Store chooses the relationship form as follows:

```text
Exact target in the same authorized Source boundary
  -> name + relative locator

Exact protected built-in target
  -> name + scope=builtin

Unique target in the current Root with no portable exact locator
  -> named relationship

Tool or Model supplied by a registered fallback provider
  -> named relationship

External URL, Git, or package target with an installed adapter
  -> name + corresponding portable locator

Target that cannot be represented safely
  -> reject attachment or retain an unavailable migration diagnostic
```

A direct `ArtifactRef` is never used as a locator substitute.

Located relationships:

- Are resolved relative to the declaration occurrence.
- Must stay within the permitted Source or registered locator adapter.
- Do not fall back to name lookup.
- Do not search the protected built-in Root.
- Do not use mapped fallback providers.

### 9.6 Immutable publication behavior

Managed Agent publication uses create-if-absent semantics.

```text
No package at publication key
  -> publish

Equivalent package already present
  -> idempotent success

Non-equivalent package already present
  -> conflict unless explicit Replace or repair is authorized
```

A successful publication is immutable by default.

Changing any of the following normally creates a new publication:

- Agent prompt.
- Agent members.
- Relationship behavior.
- Agent program.
- Display name or description when immutable release history is desired.
- Packaged launch defaults.
- Release label.

Explicit replacement is reserved for:

- User-confirmed replacement.
- Administrative repair.
- Built-in package reconciliation.
- Migration correction.

Repository-authored Agents remain governed by their Source and may update at the same occurrence through ordinary Source refresh.

## 10. Agent Collections

### 10.1 Agent Collection domain

The Agent Collection domain is:

```text
User-facing concept:       Agent Collection
Portable backing type:     plugin
Allowed managed member:    agent
Baseline logical name:     agent-baseline
Default membership form:   located external relationship
```

No new portable subtype or discriminator is added to Plugin.

A Plugin is recognized as a managed Agent Collection because:

- It is registered in the Agent Store managed package domain.
- It is declared by a built-in Agent package manifest.
- It is the application-provisioned Agent baseline package.

An arbitrary repository Plugin may be inspected as an Agent-only Plugin, but it is not automatically made editable by the managed Agent Collection API.

### 10.2 Agent Collection declaration

```yaml
type: plugin
name: engineering-agents
displayName: Engineering Agents
description: Managed Agents for engineering work.

members:
  - type: agent
    name: bug-investigator
    locator: ../../agents/01-publication/agent.yaml

  - type: agent
    name: code-reviewer
    locator: ../../agents/02-publication/agent.yaml
```

Two releases of the same Agent may be represented by different locators:

```yaml
type: plugin
name: investigation-agents
displayName: Investigation Agents

members:
  - type: agent
    name: bug-investigator
    locator: ../../agents/bug-investigator-v1/agent.yaml

  - type: agent
    name: bug-investigator
    locator: ../../agents/bug-investigator-v2/agent.yaml
```

The Agent Store reads the release labels from the corresponding publication descriptors.

### 10.3 Managed Collection restrictions

The portable Plugin contract supports mixed types, contained declarations, and selectors. The managed Agent Collection API is intentionally narrower.

Managed Agent Collections in the initial version permit:

- Empty collections.
- Named external Agent relationships.
- Located external Agent relationships.
- Dangling Agent relationships.
- Several relationships to different occurrences with the same Agent name.
- One Agent publication in several collections.

Managed Agent Collection APIs reject creation of:

- Non-Agent members.
- Contained Agents.
- Agent selectors.
- Nested Plugins.
- Relationship `overrides`.
- Relationship `use`.

These restrictions keep managed Agent publications independently addressable and provide bundle-like lifecycle behavior.

Repository-authored Plugins retain the complete portable Plugin contract.

### 10.4 Baseline Agent Collection

Each supported user Root receives one application-provisioned baseline Agent Collection:

```text
Logical name: agent-baseline
Display name: Agent Baseline
```

The baseline collection is:

- Backed by a managed Plugin package.
- Editable.
- Explicitly selectable.
- Non-renamable.
- Non-deletable.
- Not runtime-disableable through application APIs.
- Not automatically selected.
- Not automatically added to a Workspace.
- Not automatically expanded into a conversation.
- Not an implicit destination for Agent creation.

Every managed Agent create request must explicitly identify its destination collection, including when that destination is `agent-baseline`.

Ordinary user-created and built-in Agent Collections may be locally disabled.

### 10.5 Membership semantics

```text
Collection member removed
  -> Agent publication remains available

Agent publication updated by explicit replacement
  -> relationship locator remains valid when the occurrence is unchanged

Agent publication deleted
  -> collection relationship remains declared
  -> relationship becomes unavailable

Collection deleted
  -> independent Agent publications remain available

Agent attached to another collection
  -> same Agent Artifact may be selected through both relationships
```

Collection membership does not alter Agent enablement or runtime readiness.

A named collection member may change from available to ambiguous when another matching Agent occurrence appears. A located relationship remains exact.

### 10.6 Collection deletion

A managed ordinary Agent Collection can be deleted only when it has no direct declared members.

Rules:

- Unavailable members count.
- Ambiguous named members count.
- Every direct relationship must be removed first.
- Only direct Plugin members are checked.
- Agent packages are never deleted as part of collection deletion.
- The baseline collection cannot be deleted.
- Built-in collection declarations cannot be deleted.
- Deletion removes only the Plugin package and reconciles its Source occurrence.

Deletion does not require a generic composition cascade.

## 11. Primary workflows

### 11.1 Create an Agent Collection

```text
Authorized user Root
  + collection name and descriptive fields
  + initial Enabled value
  -> verify name policy
  -> allocate collection package key
  -> publish empty plugin.yaml
  -> refresh managed Agent Source
  -> set Plugin Artifact.Enabled
  -> return Agent Collection view
```

The collection `name` is immutable after creation. Display name and description may be changed through explicit Plugin package replacement with an expected revision.

### 11.2 Create a managed Agent publication

The request contains:

```text
Exact destination Agent Collection
Expected Collection revision
Portable Agent declaration
Optional publication release label
Optional launch defaults
Initial Agent.Enabled value
```

The flow is:

```text
Validate destination Collection
  -> require editable Agent Collection
  -> require Collection enabled for compatibility authoring
  -> validate Agent declaration
  -> validate publication descriptor
  -> allocate publication key and target locator
  -> add located external Agent relationship to Collection
  -> publish Agent package
  -> refresh affected managed Source
  -> locate resulting Agent Artifact
  -> set requested Artifact.Enabled
  -> resolve Agent and Collection
  -> return views and diagnostics
```

The destination collection is mandatory. There is no implicit baseline fallback.

Collection membership and Agent package publication remain two coordinated Source-side changes.

If collection publication succeeds and Agent publication fails:

- The relationship remains declared.
- The relationship is visible as unavailable.
- Retry re-reads the current collection revision.
- Retry reuses the allocated publication key where safe.
- No hidden rollback deletes unrelated collection edits.

### 11.3 Attach an existing Agent

The request may identify an existing Agent using an authorized local `ArtifactRef`.

The Agent Store then:

1. Loads the Agent occurrence.
2. Verifies Root authorization.
3. Derives its portable name.
4. Derives a safe locator or lookup scope.
5. Constructs an external Agent relationship.
6. Publishes the updated Plugin with an expected revision.
7. Returns the new relationship occurrence.

The local `ArtifactRef` is not persisted in the Plugin.

When an exact portable locator cannot be represented:

- `scope: builtin` is used for an exact protected built-in symbolic target.
- A named relationship may be used for one current Root target.
- The caller must acknowledge that an unlocated relationship can later become ambiguous.
- Attachment is rejected if safe representation is impossible.

### 11.4 Read and plan an Agent

```text
Exact Agent selection
  -> read indexed Agent Artifact and publication descriptor
  -> ResolveAgent
  -> preserve relationship statuses
  -> apply optional completeness policy
  -> apply catalog enablement policy
  -> inspect consumer runtime readiness
  -> return Agent plan
```

The read does not refresh Sources or connect runtime services.

### 11.5 Refresh an Agent closure

Explicit refresh is used when repository Agent locators or selectors may not yet be indexed.

```text
Selected Agent or Plugin
  -> inspect reachable local locators and selectors
  -> calculate bounded discovery closure
  -> update affected Source discovery
  -> refresh affected Sources
  -> resolve from new indexed state
```

Managed package creation already knows the affected managed Source and refreshes it explicitly.

### 11.6 Replace or publish a new release

The preferred update flow is:

```text
Existing Agent publication
  -> create new publication key
  -> publish new release
  -> add new located relationship
  -> optionally detach old relationship
```

Explicit replacement at the same publication occurrence requires:

- Existing Artifact or package identity.
- Expected package or Source revision.
- Explicit replacement intent.
- Full declaration and descriptor validation.

Changing only `Artifact.Enabled` does not replace package content or change the Agent Definition digest.

### 11.7 Detach or delete

Detachment and deletion are separate operations.

```text
Detach Agent
  -> remove one direct Collection relationship
  -> preserve Agent package
```

```text
Delete Agent publication
  -> remove only that Agent package
  -> reconcile managed Source
  -> leave relationships in every Collection declared and unavailable
```

A convenience operation may detach from one selected collection and then delete the Agent publication, but it must:

- Check direct memberships.
- Require explicit deletion intent.
- Never silently remove other collection memberships.
- Never cascade through nested compositions.

Built-in Agent publications cannot be deleted through user APIs.

## 12. Agent resolution and planning

### 12.1 Plan inputs

An Agent plan receives:

```text
Authorized Root context
Exact Agent ArtifactRef, publication key, or Collection member occurrence
Optional Workspace context
Optional RequireComplete
Optional RequireEnabled
Optional consumer readiness policy
Optional one-run launch input
```

The Agent declaration itself never stores a Root ID.

When an `ArtifactRef` is used, the Agent Store verifies that it belongs to:

- The authorized current Root.
- The protected built-in Root through an authorized catalog or resolution result.

### 12.2 Resolution behavior

Named relationships use the shared lookup order:

```text
Current Root Artifact
  -> protected built-in Root Artifact
  -> registered fallback provider
  -> unavailable
```

Relationships with `scope: builtin` use:

```text
Protected built-in Root Artifact
  -> registered built-in fallback provider
  -> unavailable
```

Located relationships use only the selected occurrence:

```text
Resolve locator
  -> verify expected type and name
  -> follow supported aliases
  -> terminal target or diagnostic
```

Contained relationships resolve through stable structural occurrences.

Selectors expand from indexed Source state and preserve individual match results.

The Agent Store must not implement a special Tool or Model lookup path. Tool and Model fallback is shared resolver infrastructure.

### 12.3 Agent plan

Conceptually:

```text
AgentPlan {
  AgentTarget
  AgentDefinitionDigest
  PublicationDescriptorDigest?

  Enabled
  BuiltIn
  ReleaseLabel?

  Prompt?
  TextContributions[]

  Members[] {
    MemberKey
    RelationshipResult
    ArtifactRef?
    MappedTarget?
    Overrides?
    Use?
    Enabled?
    Readiness?
    Diagnostics[]
  }

  Loop?
  Workflow?

  LaunchDefaults?
  Complete
  Ready
  Diagnostics[]
}
```

The plan preserves all declared relationship occurrences, even when several relationships reach the same terminal Artifact.

A flattened capability list may deduplicate terminal targets for a consumer, but the relationship graph remains complete.

### 12.4 Partial and strict plans

A partial plan:

- Returns the Agent declaration.
- Returns every available, unavailable, and ambiguous relationship.
- Includes applicable launch-default diagnostics.
- Allows a consumer to use supported available portions.

A strict plan with `RequireComplete=true` fails selection when a required relationship is unavailable or ambiguous.

A compatibility conversation consumer may additionally require:

- Agent enabled.
- Selected collection enabled.
- Exactly one usable preferred Model when a model change is requested.
- All selected Tools ready.
- All active or instruction Skills materializable.
- All required MCP server mappings available.
- No stale required MCP capability selection.

### 12.5 Execution boundary

Agent planning does not:

- Invoke a Model.
- Invoke a Tool.
- Execute a Skill script.
- Connect an MCP server.
- Fetch an MCP resource.
- Materialize an MCP prompt.
- Run a Loop.
- Schedule a Workflow.
- Invoke a nested Agent.
- Persist conversation state.
- Persist execution history.

## 13. Launch defaults and conversation compatibility

### 13.1 Why launch defaults are separate

Assistant Presets stored a mixture of:

- Durable capability selection.
- Agent-like behavior.
- Editable initial UI text.
- Runtime Tool input defaults.
- Dynamic MCP conversation selections.

The portable Agent contract intentionally contains only durable declaration behavior.

The replacement therefore splits one legacy object into:

```text
Portable Agent declaration
  -> durable capability and behavior selection

Agent publication launch defaults
  -> optional starter values copied to a run

Artifact local enablement
  -> local on or off state
```

This split avoids treating editable initial text as an Agent system prompt and avoids persisting MCP runtime server IDs in a portable declaration.

### 13.2 Launch default model

Conceptually:

```text
AgentLaunchDefaults {
  InitialText?

  PreferredModelMemberKey?
  IncludeCurrentModelSystemPrompt?

  OrderedToolSelections[] {
    MemberKey
    UserArgSchemaInstance?
  }

  MemberPresentationOrder[]?

  MCPContext?
}
```

`MCPContext` is an Agent-specific template:

```text
AgentMCPContextTemplate {
  Servers[] {
    MCPMemberKey
    ToolExposure
    SelectedTools[]
  }

  Resources[] {
    MCPMemberKey
    URI
    Digest?
  }

  ResourceTemplates[] {
    MCPMemberKey
    URITemplate
    Digest?
    ArgumentValues
  }

  Prompts[] {
    MCPMemberKey
    PromptName
    Digest?
    ArgumentValues
  }
}
```

Selected MCP Tool data may retain:

- Tool name.
- Provider Tool name.
- Choice identity.
- Digest.
- Approval rule override.
- Execution mode override.

Rules:

- `InitialText` is valid UTF-8.
- `InitialText` remains limited to 16 KiB for Assistant Preset compatibility.
- Initial text is stored verbatim.
- Tool user argument schema instances are starter values, not Tool invocation arguments.
- The descriptor must not contain credentials, secret values, OAuth tokens, or selected connection profiles.
- `IncludeCurrentModelSystemPrompt` exists only for compatibility with a legacy preset that expressed a model-system-prompt preference without selecting a Model.
- A modern Agent should express the preference through the selected Model relationship's `overrides.includeSystemPrompt`.
- MCP runtime server IDs are forbidden.
- Runtime conversation state is forbidden.

### 13.3 Stable member keys

Launch defaults refer to direct Agent members using stable member keys.

Conceptually:

```text
MemberKey =
  canonical hash of:
    containing field
    + type
    + name
    + locator
    + scope
    + Text insert
    + canonical overrides
    + canonical use
```

Member keys:

- Exclude source-array position.
- Survive member reordering.
- Change when relationship identity or behavior changes.
- Are local to the containing Agent publication.
- Are validated against the Agent declaration during publication.
- Are projected to full relationship occurrence identities during planning.
- Are not target Artifact references.

Launch defaults do not refer directly to selector matches in the initial version. Assistant Preset migration emits explicit direct members.

### 13.4 Launch input precedence

The effective one-run launch input is:

```text
Packaged launch defaults
  -> caller edits or request overrides
  -> validated one-run AgentLaunchInput
```

A request may:

- Replace the initial editable text.
- Select one declared Model relationship.
- Add or replace Tool user-input instances.
- Replace the MCP context template.
- Omit packaged defaults.

A request may not use launch defaults to smuggle in undeclared Tools, Skills, Models, or MCP servers.

The launch input remains one-run state and is not written back to the Agent declaration.

### 13.5 Legacy-compatible conversation projection

The compatibility adapter produces:

```text
ConversationStarter {
  InitialText

  ModelTarget?
  IncludeModelSystemPrompt?

  ToolTargets[]
  ToolUserInputs[]

  SkillTargets[]
  SkillModes[]

  MCPConversationContext?

  AgentPrompt?
  InstructionText[]
  UserMessageText[]

  Diagnostics[]
}
```

Projection behavior is:

```text
Agent prompt
  -> Agent base prompt contribution

Text insert=instructions
  -> instruction contribution

Text insert=user-message
  -> user-message context contribution

Model member
  -> Artifact-backed or mapped runtime Model target

Tool member
  -> Artifact-backed or mapped runtime Tool target

Skill member
  -> verified Skill target and declared use mode

MCP member
  -> terminal MCP installation target
```

The adapter may internally return legacy runtime handles to an existing inference implementation. Those handles are transient consumer values and are not persisted in Agent or Plugin declarations.

When no Model is declared and the launch defaults express no preferred Model, selecting the Agent leaves the caller's current Model unchanged. This preserves legacy optional model behavior.

When several Model members exist:

- `PreferredModelMemberKey` selects one for the compatibility adapter.
- A one-run launch input may select one.
- Otherwise the compatibility adapter returns an ambiguity diagnostic.
- A future full Agent runtime may define broader multi-Model behavior.

### 13.6 MCP launch context

At launch time:

```text
MCP member key
  -> resolved terminal MCP Artifact
  -> installation-local MCP state
  -> transient MCP runtime ServerID
  -> MCPConversationContext
```

The runtime `ServerID` exists only after the MCP consumer binds the resolved Artifact to its local installation.

The Agent Store preserves legacy MCP behavior by validating:

- Every referenced MCP member.
- Selected Tool names.
- Tool digests.
- Provider Tool names.
- Choice identities.
- Tool enabled state.
- Approval-rule strengthening.
- Execution-mode strengthening.
- Selected resource URIs and digests.
- Resource-template URIs, digests, and required arguments.
- Prompt names, digests, and required arguments.

MCP discovery during publication is best-effort when the runtime is unavailable or authentication is required.

Launch-time validation may be strict because current capability discovery can differ from the saved template.

## 14. Validation and runtime readiness

### 14.1 Validation layers

Agent Store validation has four layers.

```text
Portable contract validation
  -> is the Agent or Plugin declaration structurally valid?

Managed package validation
  -> is this a valid Agent Store collection or publication package?

Resolution validation
  -> do relationships identify zero, one, or several targets?

Runtime readiness
  -> can the selected consumer use those targets now?
```

These layers must not be collapsed into one Store validity bit.

### 14.2 Managed publication validation

Publication validation verifies:

- Agent document passes the registered `agentv1` schema.
- Original document bytes contain no unknown fields.
- Agent name matches the expected publication request.
- Publication descriptor passes its strict internal schema.
- Release label satisfies Agent Store package naming policy.
- Every member key in launch defaults identifies one direct Agent member.
- Preferred Model member identifies a Model relationship.
- Tool launch entries identify Tool relationships.
- MCP template entries identify MCP relationships.
- Duplicate launch entries are rejected.
- Initial text is valid UTF-8 and within the compatibility limit.
- No secret-bearing fields are present.
- Package paths remain inside the managed Source.
- Collection locator identifies the expected Agent type and name after refresh.

An unavailable external relationship does not make the portable Agent declaration invalid.

Managed authoring may request stricter admission through:

```text
RequireResolvable
RequireEnabled
RequireLaunchReady
```

The Assistant Preset compatibility create path uses strict admission where the legacy Store previously required enabled and resolvable references.

### 14.3 Skill readiness

The portable contract records Skill use intent. Skill package inspection determines whether that mode is currently usable.

For Assistant Preset-compatible behavior:

| Skill mode     | Compatibility readiness rule                                                                                                                      |
| -------------- | ------------------------------------------------------------------------------------------------------------------------------------------------- |
| `available`    | Skill resolves and can be registered                                                                                                              |
| `active`       | Skill uses instruction insertion and does not require launch arguments                                                                            |
| `instructions` | Skill uses instruction insertion, requires no launch arguments, and has no additional resources that cannot be represented as direct instructions |

A readiness failure:

- Does not invalidate the Agent declaration.
- Appears on the Skill relationship.
- Fails a strict compatibility launch.
- Remains visible in a partial Agent plan.

The Agent Store does not add package arguments or resources to the portable Skill contract.

### 14.4 MCP validation

MCP validation is divided into:

- Portable MCP declaration and relationship resolution.
- Installation-local state validation.
- Dynamic discovery validation.
- Runtime policy validation.

Publication-time dynamic discovery treats these conditions as temporarily unavailable rather than structurally invalid:

- MCP runtime not ready.
- MCP authentication required.

When discovery succeeds, mismatched or weakened selections are errors.

A selected Tool override must not:

- Weaken an approval rule.
- Change manual execution to automatic when policy requires manual execution.
- Accept a stale digest when strict freshness is required.

No MCP credentials or secret values are stored in the Agent publication descriptor.

### 14.5 Enablement policy

Artifact enablement does not affect relationship resolution.

A compatibility admission or launch policy may require:

- Destination collection enabled.
- Agent Artifact enabled.
- Selected Artifact-backed Model enabled.
- Selected Artifact-backed Tool enabled.
- Selected Skill enabled.
- Selected MCP server permitted by the MCP consumer.

Mapped Model or Tool targets use their provider-specific readiness rather than `Artifact.Enabled`.

Disabling a dependency after Agent publication makes the Agent unready for a strict consumer but does not rewrite or invalidate its Definition.

## 15. Catalog, listing, and enablement

### 15.1 Collection views

Conceptually:

```text
AgentCollectionView {
  PluginArtifactRef
  CollectionKey

  Name
  DisplayName
  Description

  Enabled
  BuiltIn
  Baseline

  DirectMembers[]
  CreatedAt?
  ModifiedAt?
  Diagnostics[]
}
```

`BuiltIn` is derived from protected Root and package provenance. It is not a portable declaration field.

`Baseline` is derived from application package policy.

### 15.2 Agent views

Conceptually:

```text
AgentPublicationView {
  AgentArtifactRef
  PublicationKey?

  Name
  ReleaseLabel?
  DisplayName
  Description

  Locator
  DefinitionDigest
  PublicationDescriptorDigest?

  Enabled
  BuiltIn

  DirectCollectionMemberships[]
  CreatedAt?
  ModifiedAt?

  ResolutionStatus?
  RuntimeReadiness?
  Diagnostics[]
}
```

Repository Agents may have:

- No publication key.
- No release label.
- No publication descriptor.

A collection member view additionally contains:

- Collection relationship occurrence identity.
- Member form.
- Member key.
- Relationship locator or scope.
- Availability status.
- Target `ArtifactRef`, when available.
- Target release label, when managed.
- Diagnostics.

### 15.3 Listing and pagination

The Agent Store provides:

- `ListAgentCollections`.
- `ListAgents`.
- `GetAgentCollection`.
- `GetAgent`.
- `ListAgentCollectionMembers`.
- `ListDirectAgentMemberships`.

Listing supports:

- Exact collection keys or local refs.
- Agent names.
- Release labels.
- Built-in or user provenance.
- `IncludeDisabled`.
- Availability filtering.
- Runtime-readiness filtering when explicitly requested.
- Opaque page tokens.

For compatibility, the initial limits remain:

```text
Default page size: 25
Maximum page size: 256
```

A page token binds:

- The original filters.
- Include-disabled behavior.
- Page size.
- Stable cursor identity.
- Relevant Source or catalog generation when snapshot consistency is required.

Default ordering is deterministic:

```text
Modified time descending
  -> stable local occurrence identity
```

Built-in and user items participate in one deterministic result set. Their ordering must not imply lookup precedence or activation.

### 15.4 Effective selection

Catalog selectability is a consumer projection.

For direct Agent selection:

```text
Selectable =
  Agent occurrence available
  and Agent.Enabled
```

For selection through an Agent Collection:

```text
Selectable =
  Collection.Enabled
  and relationship available
  and Agent.Enabled
```

Strict launch readiness additionally requires:

```text
Selectable
  and required relationship completeness
  and required resource readiness
  and runtime consumer readiness
```

A disabled collection does not disable its member Agents outside that collection.

A disabled Agent remains readable and appears when `IncludeDisabled=true`.

Enablement changes:

- Update only Artifact-local state.
- Preserve the Definition digest.
- Preserve collection membership.
- Work for protected built-in Artifacts through the authorized metadata path.
- Do not require a separate Agent overlay database.

## 16. Built-in Agent packages

### 16.1 Package layout

Built-in Agent content joins the existing protected built-in Root.

Conceptual package layout:

```text
internal/artifactcontract/builtin/agents/
  software-dev/
    plugin.yaml

    agents/
      bug-investigator/
        agent.yaml
        publication.json

      code-reviewer/
        agent.yaml
        publication.json

  product-leadership/
    plugin.yaml
    agents/
      product-reviewer/
        agent.yaml
        publication.json
```

A built-in Plugin uses located Agent relationships:

```yaml
type: plugin
name: agent-software-dev
displayName: Software Development Agents
description: Built-in Agents for software development.

members:
  - type: agent
    name: bug-investigator
    locator: ./agents/bug-investigator/agent.yaml

  - type: agent
    name: code-reviewer
    locator: ./agents/code-reviewer/agent.yaml
```

Physical co-distribution does not establish ownership.

### 16.2 Built-in validation

The built-in Agent package validator verifies:

- Package manifest and directory identity.
- Plugin declaration validity.
- Agent declaration validity.
- Publication descriptor validity.
- Direct Plugin members are Agents.
- Located members stay within the authorized package Source.
- Every located member resolves to the expected Agent name.
- Every launch-default member key is valid.
- Duplicate collection relationships are rejected.
- Duplicate publication identities are reported.
- Required built-in Tool and Model names resolve through protected built-in Artifact or fallback providers.
- Required Skill and MCP targets resolve in the protected Root.
- Strict compatibility launch rules hold for shipped Assistant Preset replacements.
- Built-in Agent Collections intended for display are non-empty.
- No package contains secret or installation-local values.

### 16.3 Protected Root behavior

Built-in Agent and Plugin declarations are read-only.

The following remain locally mutable through authorized Artifact metadata:

- Agent `Enabled`.
- Agent Collection Plugin `Enabled`.

The following remain read-only:

- Agent declaration.
- Plugin declaration.
- Publication release label.
- Packaged launch defaults.
- Package locators.

The protected Root is shared with built-in Skill, MCP, Text, Tool, Model, Plugin, and other Artifact packages.

No separate Agent Root is introduced.

### 16.4 Package hydration

Agent packages participate in package-scoped hydration:

```text
Existing protected Root
  -> reconcile Skill packages
  -> reconcile MCP packages
  -> reconcile Agent packages
  -> preserve unaffected package Artifacts and local state
```

Hydration must:

- Track Agent package fingerprints.
- Verify expected declaration and descriptor bytes.
- Repair incomplete or changed known packages.
- Remove stale known Agent packages.
- Preserve unchanged Agent and Plugin `ArtifactRef` values.
- Preserve local enablement for unchanged occurrences.
- Avoid resetting the protected Root when the Agent installer marker is absent.
- Avoid rewriting Skill or MCP package ownership.
- Avoid sweeping unrelated manually registered protected content unless a separate topology migration authorizes it.

The legacy Assistant Preset built-in overlay database is not reused.

## 17. Workspace and consumer integration

### 17.1 Workspace selection

Workspace Agent relationships continue to use the portable Workspace contract.

```yaml
type: workspace
name: checkout-service

members:
  - type: agent
    name: bug-investigator
    locator: ./agents/bug-investigator.agent.yaml
```

Workspace Agent planning is:

```text
Workspace
  -> resolved Agent relationship
  -> terminal Agent Artifact
  -> Agent plan
```

The Workspace does not automatically select:

- Every Agent in its Root.
- Every Agent Collection.
- The baseline Agent Collection.
- Every protected built-in Agent.

Protected built-in fallback occurs only for a declared named relationship.

A Workspace may explicitly select an Agent Collection by selecting its backing Plugin, but that is ordinary Plugin composition, not implicit collection activation.

### 17.2 Prompt and capability consumers

Agent consumers use resolved capability data:

```text
Agent prompt
  -> prompt consumer

Text members
  -> verified Text materializer

Model and Tool members
  -> Artifact-backed or mapped consumers

Skill members
  -> Skill materializer and runtime registration

MCP members
  -> MCP installation and runtime adapters

Loop and Workflow
  -> future execution consumers
```

Consumers must preserve relationship diagnostics and must not substitute another target for an unavailable located relationship.

### 17.3 Runtime boundary

The initial Agent Store provides:

- Catalog management.
- Managed package publication.
- Agent and collection reads.
- Typed Agent resolution.
- Launch-default validation.
- Conversation-starter projection.
- Workspace Agent planning.

It does not provide:

- An Agent execution engine.
- Delegation execution.
- Loop execution.
- Workflow scheduling.
- Tool invocation.
- Model invocation.
- Conversation persistence.
- Retry, timeout, or cancellation policy.

The existing conversation and inference path may consume a `ConversationStarter` and perform execution under its existing runtime contracts.

## 18. Assistant Preset migration and removal

### 18.1 Migration mapping

| Legacy Assistant Preset concept    | Agent Store representation                                          |
| ---------------------------------- | ------------------------------------------------------------------- |
| `AssistantPresetBundle`            | Agent Collection backed by a Plugin Artifact                        |
| Bundle ID                          | Managed collection key and temporary migration alias                |
| Bundle slug                        | Plugin `name`                                                       |
| Bundle display name                | Plugin `displayName`                                                |
| Bundle description                 | Plugin `description`                                                |
| Bundle enabled                     | Plugin Artifact `Enabled`                                           |
| Bundle built-in flag               | Protected Root provenance                                           |
| Assistant Preset                   | Managed Agent publication                                           |
| Preset ID                          | Managed publication key and temporary migration alias               |
| Preset slug                        | Agent `name`                                                        |
| Preset version                     | Publication release label and exact package locator                 |
| Preset display name                | Agent `displayName`                                                 |
| Preset description                 | Agent `description`                                                 |
| Preset enabled                     | Agent Artifact `Enabled`                                            |
| Preset built-in flag               | Protected Root provenance                                           |
| `StartingText`                     | Publication launch default `InitialText`                            |
| `StartingModelPresetRef`           | Model member using locator, built-in scope, or named fallback       |
| `StartingIncludeModelSystemPrompt` | Model relationship `overrides.includeSystemPrompt`                  |
| Tool selections                    | Tool members plus launch-default Tool inputs and presentation order |
| Skill selections                   | Skill members with `use.mode`                                       |
| `StartingMCPContext`               | MCP members plus Agent MCP launch template                          |
| Created and modified times         | Imported Agent Store publication and catalog metadata               |

`StartingText` must not be migrated into Agent `prompt`.

### 18.2 Reference conversion

Legacy direct references are converted before publishing an Agent declaration.

The conversion procedure is:

1. Resolve the legacy reference through its owning Store or runtime adapter.
2. Identify the portable type and logical name.
3. Identify the terminal source-backed Artifact when one exists.
4. Derive a source-relative locator when it is valid from the Agent declaration occurrence.
5. Use `scope: builtin` for a protected built-in target.
6. Use a named relationship for a unique current Root target.
7. Use a registered Tool or Model fallback name when the target is not Artifact-backed.
8. Record dynamic starter values in the publication launch descriptor.
9. Never serialize the old direct reference.

Specific mappings are:

```text
ModelPresetRef
  -> source-backed Model locator
  or named Model fallback target

ToolRef
  -> source-backed Tool locator
  or named Tool fallback target

ArtifactSkillSelection.Artifact
  -> Skill name + locator
  or Skill name + scope=builtin

MCP runtime ServerID
  -> terminal MCP Artifact
  -> MCP name + locator or scope
  -> member key in launch template
```

Every MCP server referenced by any legacy server, resource, resource template, or prompt selection becomes an Agent MCP member.

If an exact reference cannot be represented safely:

- The migration does not insert an `ArtifactRef` into portable content.
- The imported relationship remains explicitly unavailable with a diagnostic, or the migration blocks switching that preset.
- The preset is never silently dropped.

A temporary migration ledger may map legacy bundle and preset IDs to new local collection and publication references. The ledger is not portable content and is deleted after migration verification.

### 18.3 Migration phases

The replacement proceeds in phases.

#### Phase 1: Build Agent Store

- Implement Agent catalog and planning.
- Implement Agent Collections backed by Plugins.
- Implement managed Agent publication.
- Implement publication launch descriptors.
- Implement built-in Agent packages.
- Implement conversation compatibility projection.

#### Phase 2: Convert built-in content

- Convert each built-in Assistant Preset bundle to a built-in Agent Collection.
- Convert each built-in preset to an Agent publication.
- Convert direct references to portable relationships.
- Move starter-only values to publication descriptors.
- Transfer built-in Agent and collection enablement overlays.
- Validate every built-in conversation starter.

#### Phase 3: Import user content

- Read every non-deleted user Assistant Preset bundle.
- Publish an Agent Collection using the legacy bundle ID as a migration package key where practical.
- Publish each preset as an independent Agent package.
- Add a located Agent relationship to the collection.
- Preserve release, enablement, display metadata, description, and timestamps.
- Preserve invalid or missing dependencies as diagnostics instead of silently omitting the Agent.
- Import idempotently.

#### Phase 4: Switch consumers

- Replace Assistant Preset list and get usage with Agent catalog APIs.
- Replace Assistant Preset apply behavior with `BuildConversationStarter`.
- Replace bundle management UI with Agent Collection management.
- Replace preset enablement with Agent Artifact enablement.
- Regenerate frontend bindings.
- Keep the legacy Store read-only for rollback during verification.

#### Phase 5: Remove the legacy Store

- Stop initializing `AssistantPresetStore`.
- Remove Assistant Preset wrappers.
- Remove legacy lookup adapters.
- Remove legacy built-in bundle files.
- Remove the Assistant Preset overlay database after successful state transfer.
- Remove migration-only compatibility code after the rollback period.

No long-term dual-write design is required.

### 18.4 Removal gate

The Assistant Preset Store may be removed only after:

- Every built-in bundle has a corresponding Agent Collection.
- Every built-in preset has a corresponding Agent publication.
- Every user bundle and preset is migrated or has an explicit blocking diagnostic.
- Enablement flags are verified.
- Release lookups are verified.
- Starting text is preserved byte-for-byte.
- Model, Tool, Skill, and MCP selections are behaviorally compared.
- MCP policy-strengthening rules are verified.
- Pagination and disabled-item behavior are covered by replacement tests.
- All application and frontend call sites use Agent Store APIs.
- No portable Agent or Plugin declaration contains a legacy direct reference.
- Rollback and recovery procedures have been tested.

## 19. Implementation structure

### 19.1 Components

The existing declaration and resolver packages remain authoritative:

```text
internal/artifactcontract/declaration/
  agentv1/
  pluginv1/
  textv1/
  modelv1/
  toolv1/
  skillv1/
  mcpv1/
  loopv1/
  workflowv1/

internal/artifactcontract/resolve/
  agent.go
  plugin.go
  model.go
  tool.go
  root_fallback.go
  composition_entry.go
```

The Agent Store is organized conceptually as:

```text
internal/agent/store/
  consumerapi/
  domain/
  catalog/
  managed/
  publicationv1/
  builtin/
  launch/
  migration/

internal/agent/runtime/spec/
  launch.go
  plan.go

internal/agent/consumer/conversation/
  adapter.go

internal/workspace/store/adapter/agent/
  adapter.go

cmd/agentgo/
  wrapper_agentstore.go
```

Responsibilities are:

| Component            | Responsibility                                      |
| -------------------- | --------------------------------------------------- |
| `consumerapi`        | Stable Agent Store requests and responses           |
| `domain`             | Agent Collection and publication policies           |
| `catalog`            | Read, list, filter, pagination, and projections     |
| `managed`            | Source package publication and replacement          |
| `publicationv1`      | Strict non-portable publication descriptor          |
| `builtin`            | Built-in package declarations and hydration         |
| `launch`             | Launch-default validation and Agent plan enrichment |
| `migration`          | Temporary Assistant Preset import                   |
| conversation adapter | Legacy-compatible conversation starter              |
| Workspace adapter    | Agent plan projection from Workspace resolution     |

The Agent Store depends on interfaces for:

- Artifact Store reads and local state.
- Source publication and refresh.
- Agent and Plugin resolution.
- Verified resource access.
- Tool and Model consumer mappings.
- Skill inspection.
- MCP terminal resolution and optional discovery.

It does not depend on the Assistant Preset Store after migration.

### 19.2 Concurrency and recovery

Managed operations use optimistic concurrency:

```text
Expected Plugin Definition or package revision
  + requested mutation
  -> compare
  -> publish or conflict
```

Publication creation uses package create-if-absent behavior.

The design does not require process-local slug locks for correctness.

Collection update and Agent package publication are separate Source-side operations. Recovery rules are:

- A published collection relationship may temporarily be unavailable.
- Retrying re-reads the current collection revision.
- Equivalent existing target packages are idempotent.
- Non-equivalent packages require explicit replacement.
- Package cleanup must not delete an Agent referenced by another collection.
- A failed initial `Enabled=false` update is returned as a recoverable partial operation and must be retried before reporting success.
- Durable multi-package transaction coordination may be added without changing declaration semantics.

### 19.3 Test requirements

Tests must cover:

- Agent declarations without legacy instance schema fields.
- Text replacement for instruction and context.
- Agent Collection publication as Plugin.
- Rejection of non-Agent managed collection members.
- Located member resolution.
- Named and `scope: builtin` resolution.
- Tool and Model fallback targets.
- Side-by-side Agent releases with the same Agent name.
- Named ambiguity when several releases exist.
- Stable member keys across reordering.
- Member-key changes when relationship behavior changes.
- Launch-default validation.
- Starting text byte preservation and size limits.
- Model system prompt tri-state behavior.
- Tool auto-execution and user-input projection.
- Skill mode migration and readiness.
- MCP runtime ID removal and transient rebinding.
- MCP digest and policy validation.
- Agent and collection enablement.
- Built-in local enablement preservation.
- Collection deletion guards.
- Detach without delete.
- Delete without cascade.
- Dangling membership visibility.
- Pagination across built-in and user content.
- Idempotent migration.
- Package hydration without protected Root reset.
- No Source mutation during plan reads.
- No Agent execution from the Store or resolver.

## 20. Assistant Preset functionality parity

The following table verifies the intended replacement coverage.

| Assistant Preset functionality                | Agent Store replacement                                         | Parity                                 |
| --------------------------------------------- | --------------------------------------------------------------- | -------------------------------------- |
| Create bundle                                 | Create Agent Collection Plugin package                          | Carried                                |
| Replace bundle metadata                       | Explicit Plugin package replacement                             | Carried                                |
| Immutable bundle slug                         | Immutable Plugin `name` for managed collections                 | Carried                                |
| Enable or disable bundle                      | Plugin Artifact `Enabled`                                       | Carried                                |
| List bundles                                  | List Agent Collections                                          | Carried                                |
| Filter bundles by ID                          | Filter by collection key or local ref                           | Carried                                |
| Include disabled bundles                      | `IncludeDisabled` collection listing                            | Carried                                |
| Paginate bundles                              | Opaque Agent catalog page token                                 | Carried                                |
| Reject deletion of non-empty bundle           | Direct-member collection deletion guard                         | Carried                                |
| Built-in bundle read-only content             | Protected built-in Plugin package                               | Carried                                |
| Mutable built-in bundle enabled flag          | Protected Plugin Artifact `Enabled`                             | Carried                                |
| Create preset only in selected bundle         | Managed Agent create requires explicit collection               | Carried                                |
| Reject create into disabled bundle            | Compatibility authoring requires enabled destination collection | Carried                                |
| Preset slug                                   | Agent `name`                                                    | Carried                                |
| Preset version                                | Publication release label plus exact locator                    | Carried outside portable declaration   |
| Several preset versions                       | Several Agent package occurrences                               | Carried                                |
| Immutable preset version                      | Immutable managed Agent publication                             | Carried                                |
| Explicit conflict on duplicate version        | Package and collection publication conflict policy              | Carried                                |
| Get exact preset                              | Resolve exact publication or collection relationship            | Carried                                |
| List presets by bundle                        | List collection member occurrences                              | Carried                                |
| Include disabled presets                      | `IncludeDisabled` Agent listing                                 | Carried                                |
| Paginate presets                              | Opaque Agent catalog page token                                 | Carried                                |
| Enable or disable preset                      | Agent Artifact `Enabled`                                        | Carried                                |
| Delete user preset                            | Delete managed Agent publication                                | Carried without cascade                |
| Built-in preset read-only content             | Protected built-in Agent package                                | Carried                                |
| Mutable built-in preset enabled flag          | Protected Agent Artifact `Enabled`                              | Carried                                |
| Display name and description                  | Agent and Plugin common fields                                  | Carried                                |
| Built-in flag                                 | Derived protected Root provenance                               | Carried                                |
| Created and modified timestamps               | Agent Store package and local-state projection                  | Carried                                |
| Starting text                                 | Publication `InitialText` launch default                        | Carried without misusing Agent prompt  |
| Optional model selection                      | Model relationship or no preference                             | Carried                                |
| Model system prompt tri-state                 | Model override or compatibility launch default                  | Carried                                |
| Tool selection                                | Tool relationships                                              | Carried                                |
| Tool auto-execution preference                | `overrides.autoExecute`                                         | Carried                                |
| Tool user argument instance                   | Launch default keyed by Tool member                             | Carried outside declaration            |
| Ordered Tool selections                       | Ordered launch projection using stable member keys              | Carried outside composition semantics  |
| Skill selection                               | Skill relationships                                             | Carried                                |
| Preload active Skill                          | `use.mode: active`                                              | Carried                                |
| Use Skill as instructions                     | `use.mode: instructions`                                        | Carried                                |
| Available Skill                               | `use.mode: available`                                           | Carried                                |
| Skill argument and resource restrictions      | Compatibility runtime-readiness validation                      | Carried outside portable schema        |
| MCP server selection                          | MCP relationships                                               | Carried                                |
| MCP selected Tools                            | MCP launch template                                             | Carried outside declaration            |
| MCP selected resources                        | MCP launch template                                             | Carried outside declaration            |
| MCP resource templates and arguments          | MCP launch template                                             | Carried outside declaration            |
| MCP prompts and arguments                     | MCP launch template                                             | Carried outside declaration            |
| MCP discovery digests                         | MCP launch template with launch-time revalidation               | Carried                                |
| MCP policy-strengthening validation           | Launch-default and runtime readiness validation                 | Carried                                |
| Direct model, Tool, Skill, and MCP references | Named, scoped, or located relationships                         | Replaced intentionally                 |
| Reference enabled checks on create            | Strict compatibility admission policy                           | Carried                                |
| Duplicate Tool and Skill selection checks     | Contract duplicate validation plus launch member-key validation | Carried                                |
| Built-in snapshot rebuilding                  | Artifact package hydration and catalog projection               | Replaced by shared infrastructure      |
| Per-slug process lock                         | Expected revisions and managed package conflicts                | Replaced by stronger concurrency model |
| Soft-deleted empty bundle cleanup             | Authoritative managed package removal                           | Equivalent user-visible deletion       |
| Store close lifecycle                         | Agent Store and owned publisher/resource lifecycle              | Carried                                |

All user-visible Assistant Preset capabilities have a destination in the proposed design.

The implementation mechanisms are intentionally not preserved when Artifact Store already supplies a stronger shared mechanism.

## 21. Differences with respect to Assistant Presets

| Area                     | Assistant Preset Store                                      | Agent Store                                                                                  |
| ------------------------ | ----------------------------------------------------------- | -------------------------------------------------------------------------------------------- |
| Primary record           | Custom JSON preset file                                     | Source-backed portable Agent Artifact                                                        |
| Grouping                 | Custom Assistant Preset bundle                              | Agent-only Plugin projected as Agent Collection                                              |
| Portable collection type | Custom bundle model                                         | No `collection` type                                                                         |
| Instructions and context | Implicit starter behavior                                   | Text with explicit `insert`                                                                  |
| Version                  | Field in preset JSON                                        | Agent Store release label outside Agent declaration                                          |
| Exact release selection  | Bundle, slug, and version lookup                            | Located Agent relationship or exact publication                                              |
| Identity                 | Store ID, slug, and version                                 | Root-scoped `(agent, name)` plus occurrence                                                  |
| Direct references        | Model preset ref, Tool ref, Skill ArtifactRef, MCP ServerID | Named, scoped, or located Artifact relationships                                             |
| Initial editor text      | `StartingText` in preset                                    | `InitialText` in publication launch defaults                                                 |
| Agent prompt             | No direct equivalent                                        | First-class Agent `prompt`                                                                   |
| Model behavior           | Preset-level fields                                         | Model member and relationship override                                                       |
| Tool behavior            | Ordered Tool selections                                     | Tool members plus runtime launch defaults                                                    |
| Skill behavior           | Two legacy booleans                                         | One typed Skill `use.mode`                                                                   |
| MCP server identity      | Persisted runtime ServerID                                  | MCP relationship plus transient runtime binding                                              |
| MCP selected context     | Stored directly in preset                                   | Stored in non-portable launch template                                                       |
| Ordering                 | Tool and Skill arrays treated as ordered                    | Composition unordered; compatibility presentation order stored separately                    |
| Missing target           | Preset creation usually rejected                            | Declaration can remain partially resolvable; strict consumer may reject launch               |
| Disabled dependency      | Rejected by legacy reference validation                     | Still resolves; readiness or compatibility policy rejects use                                |
| Enablement               | Bundle and preset fields or built-in overlay flags          | Universal Plugin and Agent Artifact `Enabled`                                                |
| Built-in storage         | Dedicated embedded bundle loader and overlay database       | Shared protected Root and package hydration                                                  |
| Membership ownership     | Preset file physically inside bundle directory              | Plugin relationship does not own Agent package                                               |
| Detach                   | Not independent from preset file placement                  | Removes relationship and preserves Agent                                                     |
| Agent delete             | Removes preset from its bundle directory                    | Removes independent Agent package and leaves dangling relationships                          |
| Bundle delete            | Soft-delete implementation with delayed cleanup             | Remove empty Plugin package through Source reconciliation                                    |
| Resolution               | Store-specific lookup adapters                              | Shared typed Artifact resolver                                                               |
| Runtime readiness        | Mixed into Store validation                                 | Explicitly separated from declaration resolution                                             |
| Workspace support        | Indirect application behavior                               | Ordinary Workspace Agent relationships and Agent plans                                       |
| Agent programs           | Unsupported                                                 | Optional Loop or Workflow relationship                                                       |
| Nested composition       | Unsupported                                                 | Plugin and Agent relationships supported by the portable contract                            |
| Execution                | Preset applied to existing conversation                     | Agent plan plus compatibility conversation projection; full Agent execution remains deferred |

The most important semantic differences are:

- `StartingText` is not converted into Agent `prompt`.
- Assistant Preset versions remain available as Agent Store release labels and located occurrences, but no `version` field is added to the portable Agent declaration.
- Agent Collections are Plugins, not a new portable collection type.
- Direct Store references are removed from declaration content.
- Dynamic MCP selections remain launch data rather than durable Agent behavior.
- Artifact resolution remains observable even when a target is disabled or not runtime-ready.
- Agent and collection lifecycle no longer imply ownership or cascading deletion.
- Agent member order cannot be used as lookup or execution precedence.

## 22. Current and proposed status

Status terminology:

- `Available` means the supporting Artifact infrastructure exists.
- `Proposed` means this HLD defines the approved target design but implementation has not yet been completed.
- `Pending` means implementation or verification work remains.
- `Deferred` means intentionally outside the initial Agent Store scope.
- `Removed after migration` means legacy behavior remains only until replacement gates pass.

| Capability                                            | Status                  |
| ----------------------------------------------------- | ----------------------- |
| Portable `agentv1` contract                           | Available               |
| Portable `pluginv1` contract                          | Available               |
| Text replacement for instruction and context          | Available               |
| Agent physical-format adapters                        | Available               |
| Typed Agent resolver                                  | Available               |
| Typed Plugin resolver                                 | Available               |
| Protected built-in fallback                           | Available               |
| Tool and Model fallback-provider infrastructure       | Available               |
| Universal Artifact enablement                         | Available               |
| Managed Source package publication foundation         | Available               |
| Agent Store domain facade                             | Proposed                |
| Agent Collection projection over Plugin               | Proposed                |
| Application-provisioned Agent baseline Plugin         | Proposed                |
| Managed Agent publication                             | Proposed                |
| Agent publication descriptor                          | Proposed                |
| Agent launch-default validation                       | Proposed                |
| Agent catalog and pagination                          | Proposed                |
| Conversation-starter compatibility adapter            | Proposed                |
| Built-in Agent packages                               | Proposed                |
| Agent package hydration registration                  | Proposed                |
| Workspace Agent Store adapter                         | Proposed                |
| Assistant Preset migration importer                   | Proposed                |
| User-data migration verification                      | Pending                 |
| Frontend Agent Store bindings                         | Pending                 |
| Replacement of Assistant Preset call sites            | Pending                 |
| Removal of legacy direct reference lookup adapters    | Pending                 |
| Removal of Assistant Preset Store                     | Removed after migration |
| Removal of Assistant Preset built-in overlay database | Removed after migration |
| Full Agent execution runtime                          | Deferred                |
| Loop execution                                        | Deferred                |
| Workflow scheduling and execution                     | Deferred                |

The Artifact ecosystem HLD currently marks managed Agent authoring as deferred. Adoption of this HLD promotes that specific capability from deferred ecosystem scope to proposed Agent Store implementation work. Until implementation and status updates are complete, the current Artifact ecosystem status remains authoritative.
