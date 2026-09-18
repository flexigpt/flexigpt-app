# Artifact-backed Agent Store HLD

Status: Backend implementation and static built-in Assistant Preset conversion available; verification and consumer migration pending

Normative foundations:

- [Portable Artifact Declaration Contracts HLD](./artifact_contracts_hld.md)
- [Artifact Store, Resolution, and Ecosystem HLD](./artifacts_hld.md)

This HLD defines only the Agent-specific Store and managed-authoring behavior. Declaration syntax, Source behavior, Artifact persistence, relationship resolution, fallback, enablement, and composition semantics are inherited from the two normative HLDs and are not redefined here.

- [1. Purpose](#1-purpose)
- [2. Goals and scope](#2-goals-and-scope)
  - [2.1 Goals](#21-goals)
  - [2.2 In scope](#22-in-scope)
  - [2.3 Out of scope](#23-out-of-scope)
- [3. Requirements](#3-requirements)
  - [3.1 Declaration requirements](#31-declaration-requirements)
  - [3.2 Storage requirements](#32-storage-requirements)
  - [3.3 Agent Collection requirements](#33-agent-collection-requirements)
  - [3.4 Resolution requirements](#34-resolution-requirements)
  - [3.5 Breaking-change requirement](#35-breaking-change-requirement)
- [4. Architecture and ownership](#4-architecture-and-ownership)
  - [4.1 Component responsibilities](#41-component-responsibilities)
  - [4.2 Persisted state](#42-persisted-state)
  - [4.3 References](#43-references)
- [5. Agent declaration usage](#5-agent-declaration-usage)
  - [5.1 Agent example](#51-agent-example)
  - [5.2 Agent-specific declaration rules](#52-agent-specific-declaration-rules)
- [6. Agent Collections](#6-agent-collections)
  - [6.1 Collection representation](#61-collection-representation)
  - [6.2 Collection declaration](#62-collection-declaration)
  - [6.3 Baseline Agent Collection](#63-baseline-agent-collection)
  - [6.4 Membership behavior](#64-membership-behavior)
  - [6.5 Collection deletion](#65-collection-deletion)
- [7. Managed Agent authoring](#7-managed-agent-authoring)
  - [7.1 Managed Source layout](#71-managed-source-layout)
  - [7.2 Supported operations](#72-supported-operations)
  - [7.3 Create Agent](#73-create-agent)
  - [7.4 Attach an existing Agent](#74-attach-an-existing-agent)
  - [7.5 Replace Agent](#75-replace-agent)
  - [7.6 Detach and delete](#76-detach-and-delete)
  - [7.7 Agent variants and versions](#77-agent-variants-and-versions)
- [8. Agent reads and resolution](#8-agent-reads-and-resolution)
  - [8.1 Read APIs](#81-read-apis)
  - [8.2 Resolution result](#82-resolution-result)
  - [8.3 Validation boundary](#83-validation-boundary)
  - [8.4 Enablement](#84-enablement)
- [9. Built-in Agents](#9-built-in-agents)
  - [9.1 Package structure](#91-package-structure)
  - [9.2 Package validation](#92-package-validation)
  - [9.3 Protected Root behavior](#93-protected-root-behavior)
- [10. Workspace and consumer integration](#10-workspace-and-consumer-integration)
- [11. Implementation areas](#11-implementation-areas)
- [12. Functional parity with Assistant Presets](#12-functional-parity-with-assistant-presets)
- [13. Differences with respect to Assistant Presets](#13-differences-with-respect-to-assistant-presets)
- [14. Runtime boundary](#14-runtime-boundary)
- [15. Implementation status](#15-implementation-status)

## 1. Purpose

The Agent Store provides application-level management of source-backed Agent Artifacts.

The primary flow is:

```text
Agent declaration
  -> managed or repository Source
  -> Agent Definition and Artifact
  -> shared Agent resolver
  -> declaration consumer
```

The Agent Store adds:

- Managed Agent authoring.
- Agent catalog APIs.
- Agent-only collections backed by Plugins.
- Built-in Agent packages.
- Agent-specific management policy.

It does not add another declaration format or another persistence system.

## 2. Goals and scope

### 2.1 Goals

The design must:

- Reuse the current Artifact Store and resolver ecosystem.
- Store Agents as ordinary source-backed `agent` Artifacts.
- Represent Agent collections using the existing `plugin` contract.
- Replace Assistant Preset use cases using current Artifact concepts.
- Use named, scoped, located, contained, and selector relationships as intended.
- Keep Agent Store concerns separate from Agent runtime concerns.
- Avoid Agent-specific copies of functionality already provided by Artifact Store.

### 2.2 In scope

This HLD defines:

- Agent Store consumer APIs.
- Managed Agent package authoring.
- Agent-only Plugin management.
- Agent catalog and collection views.
- Agent enablement management.
- Built-in Agent package integration.
- Use of shared Agent and Plugin resolution.
- Functional correspondence with Assistant Preset use cases.

### 2.3 Out of scope

This HLD does not define:

- Agent execution.
- Runtime launch inputs.
- Runtime projections.
- Conversation state.
- Tool invocation arguments.
- Skill execution or activation state.
- MCP discovery, connection, prompts, resources, or conversation context.
- Model provider readiness.
- Workflow or Loop execution.
- Assistant Preset compatibility APIs.
- Assistant Preset wire aliases.
- Automatic migration behavior.
- Dual-read or dual-write behavior.
- Agent release or version fields.
- Additional sidecar files or storage Artifacts.

## 3. Requirements

### 3.1 Declaration requirements

Agent Store declarations must use the existing portable contracts.

An Agent declaration:

- Uses `type: agent`.
- Uses the current flat member grammar.
- Uses Text for instruction and user-message content.
- Uses Model, Tool, Skill, MCP, Plugin, and Agent relationships as defined by `agentv1`.
- Uses `loop` or `workflow` only through the existing Agent program fields.
- Does not contain Store IDs or runtime identities.
- Does not contain declaration-instance schema or version fields.

Agent relationships must not persist:

- Model preset references.
- Tool Store references.
- Skill `ArtifactRef` values.
- MCP runtime server IDs.
- Root IDs.
- Source IDs.

### 3.2 Storage requirements

The Agent Store must use Artifact Store for:

- Roots.
- Sources.
- Definitions.
- Agent Artifacts.
- Plugin Artifacts.
- Source bindings.
- Verified resources.
- Artifact enablement.
- Artifact-local data where independently required by a consumer.

The Agent Store must not create:

- A separate Agent declaration database.
- Agent membership rows.
- Reverse membership records as authoritative state.
- Agent-specific built-in overlay storage.
- Runtime configuration files.
- Agent publication descriptors.
- Agent release metadata.

Managed Agent and Plugin declaration files are the only Agent Store source content required by this design.

### 3.3 Agent Collection requirements

A user-facing Agent Collection must:

- Be backed by a `plugin` Artifact.
- Contain only Agent member relationships.
- Support the Agent member forms allowed by the Plugin contract.
- Use external located Agent membership for ordinary managed Agent creation.
- Preserve unavailable and ambiguous memberships.
- Permit one Agent to belong to multiple collections.
- Keep collection enablement separate from Agent enablement.
- Avoid ownership and cascading deletion.

Managed Agent creation must explicitly identify its destination Agent Collection.

### 3.4 Resolution requirements

The Agent Store must use the shared typed resolvers.

It must not implement separate lookup logic for:

- Models.
- Tools.
- Skills.
- MCP servers.
- Plugins.
- Nested Agents.
- Loops.
- Workflows.

Agent reads may return the existing resolver result so consumers can observe:

- Available relationships.
- Unavailable relationships.
- Ambiguous relationships.
- Artifact-backed targets.
- Mapped Tool or Model targets.
- Relationship diagnostics.
- Relationship `overrides`.
- Relationship `use`.

The Agent Store must not interpret declaration availability as runtime readiness.

### 3.5 Breaking-change requirement

The Agent Store is a new design, not a compatibility layer.

It must not:

- Expose Assistant Preset request or response types.
- Accept Assistant Preset JSON.
- Retain bundle, preset, slug-version, or legacy reference aliases.
- Store compatibility-only fields.
- Introduce fields solely to preserve old wire behavior.
- Translate runtime state into Agent Store declaration state.
- Add intermediate bridge records.

Functional parity means that the former user use cases can be expressed through the new declaration, Store, resolver, and runtime boundaries. It does not mean API, code, identifier, storage, or wire parity.

## 4. Architecture and ownership

### 4.1 Component responsibilities

```text
Agent Store consumer API
  -> Agent Store domain policy
    -> managed Source publisher
    -> Artifact Store
    -> Agent and Plugin resolvers
```

| Component             | Responsibility                                                                                 |
| --------------------- | ---------------------------------------------------------------------------------------------- |
| Artifact Store        | Source-backed Definitions, Artifacts, resources, enablement, and local state                   |
| Declaration contracts | Portable Agent, Plugin, Text, Model, Tool, Skill, MCP, Loop, and Workflow syntax               |
| Shared resolvers      | Relationship lookup, locator handling, fallback, selectors, cycles, and diagnostics            |
| Agent Store           | Managed Agent policy, Agent Collections, catalog APIs, and built-in Agent package registration |
| Agent runtime         | Execution and all run-specific values                                                          |

The Agent Store is therefore a domain facade over existing Artifact infrastructure.

### 4.2 Persisted state

Agent Store persistence consists of ordinary Source content:

```text
Managed Source
  -> Plugin declaration files
  -> Agent declaration files
```

Artifact Store derives and persists the corresponding Definitions and Artifacts.

Agent Collection membership remains inside the Plugin Definition. It is not copied into another database.

Agent metadata such as `displayName`, `description`, `labels`, and `metadata` remains in the portable declaration.

Enablement remains in `Artifact.Enabled`.

### 4.3 References

Portable declaration relationships use:

```text
name
name + scope
name + locator
contained parameters
selector base and filters
```

Local Agent Store APIs may use an authorized `ArtifactRef` to address an existing local Artifact.

When an existing Artifact is attached to an Agent or Agent Collection, the persisted relationship must be converted to the appropriate portable relationship form. The local `ArtifactRef` is not written into the declaration.

## 5. Agent declaration usage

### 5.1 Agent example

```yaml
type: agent
name: bug-investigator
displayName: Bug Investigator
description: Investigates failures using repository evidence.

members:
  - type: text
    name: investigation-rules
    insert: instructions
    parameters:
      mediaType: text/markdown
      content: |
        Separate observations from assumptions.
        Prefer the smallest safe change.

  - type: text
    name: investigation-request
    insert: user-message
    parameters:
      mediaType: text/markdown
      content: |
        Investigate the current failure using available repository evidence.

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

  - type: mcp
    name: github
```

The Agent Store does not give special meaning to an initial user message. It is an ordinary Text member with `insert: user-message`.

### 5.2 Agent-specific declaration rules

The Agent Store relies on the existing Agent contract for member eligibility and validation.

Agent-specific management must preserve:

- Text `insert` identity.
- Model `overrides.includeSystemPrompt`.
- Tool `overrides.autoExecute`.
- Skill `use.mode`.
- MCP membership as a relationship to an MCP Artifact.
- Optional Plugin and nested Agent composition.
- Optional Loop or Workflow relationship.

Named Model and Tool relationships may resolve through registered fallback providers.

`scope: builtin` requests protected built-in lookup.

A locator selects an exact declaration occurrence and does not fall back to another occurrence.

No MCP-specific selection or discovery data belongs in the Agent declaration beyond the MCP contract itself.

## 6. Agent Collections

### 6.1 Collection representation

An Agent Collection is an Agent Store domain view over a Plugin.

```text
Agent Collection
  -> Plugin Artifact
  -> Agent-only direct members
```

There is no `collection` Artifact type and no Agent-specific collection wire format.

A Plugin qualifies as a managed Agent Collection when:

- It is provisioned in the managed Agent collection domain.
- Every direct member has `type: agent`.

A general mixed Plugin remains a Plugin and is not exposed as a managed Agent Collection.

### 6.2 Collection declaration

```yaml
type: plugin
name: engineering-agents
displayName: Engineering Agents
description: Agents used for engineering work.

members:
  - type: agent
    name: bug-investigator
    locator: ../../agent/bug-investigator/agent.yaml

  - type: agent
    name: code-reviewer
    locator: ../../agent/code-reviewer/agent.yaml
```

The Plugin contract remains authoritative for named, located, contained, and selector member behavior.

Managed `CreateAgent` uses a located external relationship by default so the Agent remains independently managed and the collection selects the intended occurrence.

### 6.3 Baseline Agent Collection

Each supported user Root has one application-provisioned baseline Agent Collection:

```text
Logical name: agent-baseline
Portable type: plugin
Allowed member type: agent
```

The baseline Agent Collection is:

- Editable.
- Non-renamable.
- Non-deletable.
- Explicitly selectable.
- Not an implicit destination for Agent creation.
- Not automatically selected by a Workspace.
- Not automatically active.

Provisioning the baseline collection is an application Root lifecycle concern, not generic Artifact Store Root behavior.

### 6.4 Membership behavior

Agent Collection relationships follow ordinary Plugin semantics.

```text
Member removed
  -> Agent remains available

Agent replaced at the same located occurrence
  -> relationship continues to identify that occurrence

Agent deleted
  -> relationship remains declared
  -> relationship becomes unavailable

Collection deleted
  -> independent Agents remain available

Agent added to another collection
  -> both relationships select the same Agent occurrence
```

Collection enablement does not alter Agent enablement or resolution.

### 6.5 Collection deletion

An ordinary managed Agent Collection can be deleted only when it has no direct declared members.

The existing managed Plugin deletion rules apply:

- Unavailable members count.
- Ambiguous members count.
- Selectors count.
- Contained members count.
- Only direct members are checked.
- Deleting the collection never deletes independent Agents.
- The baseline collection cannot be deleted.
- Built-in collection declarations cannot be deleted.

## 7. Managed Agent authoring

### 7.1 Managed Source layout

A managed Agent Source follows the existing managed package model.

A representative layout is:

```text
user-agents/
  plugin/
    agent-baseline/
      unversioned/
        plugin.yaml

    engineering-agents/
      unversioned/
        plugin.yaml

  agent/
    bug-investigator/
      unversioned/
        agent.yaml

    code-reviewer/
      unversioned/
        agent.yaml
```

The exact storage keys are managed Source concerns.

The Plugin relationship locator is relative to the Plugin declaration occurrence and must remain inside the authorized Source boundary.

No additional Agent Store file is required beside the Agent declaration.

### 7.2 Supported operations

The Agent Store exposes domain operations for:

- Creating an empty Agent Collection.
- Updating Agent Collection descriptive fields.
- Listing Agent Collections.
- Setting Agent Collection Artifact enablement.
- Deleting an empty Agent Collection.
- Creating a managed Agent in an explicitly selected collection.
- Attaching an existing Agent to a collection.
- Detaching an Agent from a collection.
- Replacing a managed Agent package with explicit replacement intent.
- Listing and reading Agents.
- Setting Agent Artifact enablement.
- Deleting a managed Agent package.
- Listing direct Agent Collection memberships.

These operations publish and reconcile Source content. They do not write composition state directly into Artifact Store records.

### 7.3 Create Agent

Managed Agent creation receives:

```text
Explicit Agent Collection ArtifactRef
Expected Collection revision
Portable Agent declaration
Initial Artifact.Enabled value
```

The flow is:

```text
Validate selected Agent Collection
  -> calculate managed Agent package location
  -> add located external Agent membership
  -> publish Agent declaration package
  -> refresh affected managed Source
  -> set initial Artifact.Enabled value
  -> return Agent and Collection views
```

The selected Agent Collection is mandatory.

There is no implicit baseline fallback.

The publication behavior follows the existing managed Plugin, Skill, and MCP authoring model:

- Equivalent retry is idempotent.
- Non-equivalent existing package content requires explicit replacement.
- A collection relationship may temporarily remain unavailable if target publication fails.
- Retry uses the current collection revision.

Repository-authored Agents may exist without belonging to a managed Agent Collection. The explicit collection requirement applies only to the managed Agent creation API.

### 7.4 Attach an existing Agent

The attach operation accepts:

```text
Editable Agent Collection
Expected Collection revision
Authorized existing Agent ArtifactRef
```

The Agent Store derives the persisted relationship:

- Use a relative locator when the exact occurrence is representable from the collection Source.
- Use `scope: builtin` for a protected built-in Agent.
- Use a named relationship when Root-scoped symbolic lookup is the intended behavior.
- Reject an attachment that would require an unsupported cross-Root reference.

The supplied `ArtifactRef` is an operation input only.

### 7.5 Replace Agent

A managed Agent package may be replaced only with:

- Explicit replacement intent.
- The expected current package or Artifact revision.
- A complete valid Agent declaration.

Replacement publishes a new immutable Definition and updates the existing source-backed Agent occurrence.

Changing `Artifact.Enabled` is not replacement and does not modify the Agent Definition.

Renaming a managed Agent is not an in-place operation. A differently named Agent is created and collection relationships are updated explicitly.

### 7.6 Detach and delete

Detaching and deleting remain independent.

```text
Detach
  -> remove collection relationship
  -> preserve Agent package
```

```text
Delete Agent
  -> remove Agent package
  -> preserve declared relationships
  -> affected relationships become unavailable
```

The Agent Store must not silently detach an Agent from every collection as part of deletion.

Built-in Agent packages cannot be replaced or deleted through user-managed APIs.

### 7.7 Agent variants and versions

The current portable contract has no Agent release version.

The Agent Store does not add one.

Several selectable Agent configurations are represented by:

- Distinct Agent names, which is the preferred managed-authoring form.
- Distinct located occurrences when the same semantic name is intentionally used more than once.

Named lookup becomes ambiguous when several distinct occurrences with the same name are visible in one Root. Located relationships remain exact.

Artifact Definitions are immutable, while explicit package replacement selects a new Definition for an existing Artifact occurrence.

## 8. Agent reads and resolution

### 8.1 Read APIs

The Agent Store provides:

- `GetAgent`.
- `ListAgents`.
- `ResolveAgent`.
- `GetAgentCollection`.
- `ListAgentCollections`.
- `ListAgentCollectionMembers`.
- `ListDirectAgentMemberships`.

The APIs use Artifact Store identity, paging, provenance, and enablement instead of defining Assistant Preset IDs or page-token contracts.

Agent lists may filter by:

- Authorized Root.
- Agent `ArtifactRef`.
- Logical name.
- Agent Collection.
- Built-in provenance.
- Artifact enablement.

Collection membership lists retain unavailable and ambiguous relationships.

### 8.2 Resolution result

`ResolveAgent` delegates to the existing typed Agent resolver.

The returned view contains the Agent Artifact and the shared resolution graph, including:

- Declared members.
- Relationship form and identity.
- `available`, `unavailable`, or `ambiguous` status.
- Artifact-backed target references.
- Mapped Tool or Model targets.
- Alias provenance when provided by the resolver.
- Relationship `overrides`.
- Relationship `use`.
- Diagnostics.
- Optional Loop or Workflow relationship.

No Agent Store-specific runtime plan is introduced.

A consumer may request complete resolution using the shared completeness behavior. Partial resolution remains valid for inspection and editing.

### 8.3 Validation boundary

The Agent Store validates:

- Agent declarations through the registered portable schema.
- Agent Collection declarations through the Plugin schema.
- The Agent-only managed collection restriction.
- Managed package and locator safety.
- Expected revisions for managed writes.

The shared resolver determines whether declared targets exist and are unambiguous.

The Agent Store does not validate:

- Model provider availability.
- Tool execution readiness.
- Tool user arguments.
- Skill runtime arguments or resources.
- MCP connection state.
- MCP discovered Tools, resources, prompts, or digests.
- MCP authentication.
- Workflow executability.

A successful managed write may return an incomplete resolution result. The declaration remains valid and the missing relationship remains visible.

Built-in package admission may require complete declaration resolution because shipped built-in content is expected to be internally consistent.

### 8.4 Enablement

Agents and Agent Collection Plugins use universal `Artifact.Enabled`.

Enablement:

- Does not change resolution.
- Does not change collection membership.
- Does not change the Definition digest.
- Does not prevent declaration reads or edits.
- May be used by catalog consumers for explicit enabled-only filtering.

Disabling a collection does not disable its Agents.

Disabling an Agent does not remove it from collections.

Built-in Agent and Plugin enablement uses the authorized protected Artifact metadata path. No Agent-specific overlay database is introduced.

## 9. Built-in Agents

### 9.1 Package structure

Built-in Agent packages use ordinary Plugin and Agent declarations.

```text
internal/artifactcontract/builtin/agents/
  core-agents/
    plugin.yaml

    base/
      agent.yaml

    local-reader/
      agent.yaml

  software-development-agents/
    plugin.yaml

    bug-investigator/
      agent.yaml
```

A built-in collection is a Plugin with Agent-only members:

```yaml
type: plugin
name: software-development-agents
displayName: Software Development Agents

members:
  - type: agent
    name: bug-investigator
    locator: ./bug-investigator/agent.yaml

  - type: agent
    name: code-reviewer
    locator: ./code-reviewer/agent.yaml
```

The Plugin and Agents are independent Artifacts even when distributed in the same package.

### 9.2 Package validation

The Agent built-in package validator verifies:

- Plugin and Agent declarations pass their registered schemas.
- The package Plugin contains only Agent members.
- Located Agent members remain within the package Source.
- Located members identify the expected Agent names.
- Required relationships resolve in the protected Root or through registered built-in fallback providers.
- Duplicate managed Agent identities and relationships are rejected.
- The package contains no Store IDs, runtime IDs, secrets, or runtime state.

The validator uses Artifact resolution only. It does not perform MCP discovery, Skill execution checks, Tool execution checks, or Model provider checks.

### 9.3 Protected Root behavior

Built-in Agent packages join the existing shared protected Root.

They inherit package-scoped hydration, protected mutation policy, and local enablement behavior from the Artifact ecosystem HLD.

Agent package hydration must not:

- Create a separate Agent Root.
- Reset existing protected Skill or MCP topology.
- Replace unchanged package ArtifactRefs.
- Add a separate built-in Agent overlay Store.
- Infer ownership between a built-in collection and its Agents.

## 10. Workspace and consumer integration

Workspace Agent selection already exists in the portable Workspace contract and shared resolver.

```yaml
type: workspace
name: checkout-service

members:
  - type: agent
    name: bug-investigator
    locator: ./agents/bug-investigator.agent.yaml
```

No Workspace schema extension is required by the Agent Store.

A Workspace or another consumer may:

```text
Resolve Workspace
  -> obtain Agent relationship
  -> resolve Agent
  -> consume the shared Agent relationship graph
```

Agent Collections are not automatically selected by a Workspace.

The baseline Agent Collection is not automatically selected.

Built-in Agents participate only through explicit named, scoped, or located relationships and the existing fallback rules.

## 11. Implementation areas

Existing infrastructure reused without Agent-specific duplication:

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
  root_fallback.go
  model.go
  tool.go
```

Agent-specific implementation belongs under:

```text
internal/agent/store/
  builtin/
  consumerapi/
  domain/

cmd/agentgo/
  wrapper_agentstore.go
```

The backend implementation provides:

- Agent Store consumer requests, responses, catalog reads, and shared resolver access.
- Agent-only managed Plugin Collection policy.
- Baseline Agent Collection provisioning for user Roots.
- Managed Agent package creation, explicit replacement, and removal.
- Portable named, located, contained, selector, and `scope: builtin` collection relationships.
- Built-in Agent package embedding, validation, package-scoped hydration, stale-package removal, and final resolution validation.
- Agent and Agent Collection enablement through universal `Artifact.Enabled`.
- Application composition and backend wrapper initialization.

No new portable declaration type, Agent runtime package, publication descriptor, or Agent-specific persistence database is required.

## 12. Functional parity with Assistant Presets

Functional parity means that the user intent previously served by Assistant Presets remains expressible in the new architecture. It does not imply old API, wire, identifier, or storage behavior.

| Former use case                                  | New representation or owner                                |
| ------------------------------------------------ | ---------------------------------------------------------- |
| Group assistant configurations                   | Agent Collection backed by a Plugin                        |
| Create and edit a group                          | Managed Plugin authoring                                   |
| Require an explicit destination group            | Explicit Agent Collection in managed `CreateAgent`         |
| Enable or disable a group                        | Plugin Artifact `Enabled`                                  |
| Prevent deletion of a non-empty group            | Managed Plugin direct-member deletion guard                |
| Create a reusable assistant configuration        | Portable Agent declaration                                 |
| Display a name and description                   | Agent `displayName` and `description`                      |
| Enable or disable one configuration              | Agent Artifact `Enabled`                                   |
| List built-in and user configurations            | Agent Artifact catalog                                     |
| Read built-in configuration as read-only         | Agent in the protected built-in Root                       |
| Locally enable or disable built-in content       | Protected Agent or Plugin Artifact `Enabled`               |
| Supply an initial user message                   | Text member with `insert: user-message`                    |
| Supply reusable instructions                     | Text members with `insert: instructions`                   |
| Select a Model                                   | Model member                                               |
| Include the selected Model system prompt         | Model `overrides.includeSystemPrompt`                      |
| Select Tools                                     | Tool members                                               |
| Express Tool auto-execution intent               | Tool `overrides.autoExecute`                               |
| Select Skills                                    | Skill members                                              |
| Make a Skill available                           | Skill `use.mode: available`                                |
| Preload a Skill as active                        | Skill `use.mode: active`                                   |
| Use a Skill as instructions                      | Skill `use.mode: instructions`                             |
| Include an MCP server capability                 | MCP member                                                 |
| Select a specific source-backed occurrence       | Located relationship                                       |
| Select protected built-in content                | `scope: builtin`                                           |
| Select existing Tool or Model Store content      | Registered named fallback provider                         |
| Verify that selected declarations exist          | Shared Agent relationship resolution                       |
| Report missing selections                        | Unavailable relationship diagnostics                       |
| Report conflicting selections                    | Ambiguous relationship diagnostics                         |
| Keep several selectable configurations           | Distinct Agent names or distinct located Agent occurrences |
| Preserve immutable declaration content           | Immutable Artifact Definitions                             |
| Update a managed configuration explicitly        | Managed package replacement with expected revision         |
| Attach one configuration to several groups       | One Agent referenced by several Plugins                    |
| Remove from a group without deleting             | Detach Plugin relationship                                 |
| Delete an Agent without composition ownership    | Remove Agent package and leave relationships unavailable   |
| Supply Tool user values                          | Agent runtime input, not Agent Store state                 |
| Supply MCP selected Tools, resources, or prompts | Agent runtime input, not Agent Store state                 |
| Preserve runtime ordering where required         | Agent runtime request, not declaration member order        |

The former Store mixed declaration and runtime concerns. The new design satisfies those use cases through their intended owners rather than preserving that mixture.

## 13. Differences with respect to Assistant Presets

The Agent Store is intentionally different from the Assistant Preset Store:

- An Agent is a portable Artifact declaration, not a custom preset JSON record.
- An Agent Collection is a Plugin, not a bundle record or `collection` Artifact.
- There are no Assistant Preset IDs, slugs, versions, aliases, or compatibility endpoints.
- There is no Agent Store `initialText` field. Initial user content is Text with `insert: user-message`.
- There is no stored `startingMCPContext`. MCP runtime context belongs to the runtime.
- There are no persisted model preset, Tool, Skill Artifact, or MCP runtime references in Agent declarations.
- Tool and Model Store integration uses registered fallback providers.
- Exact source-backed selection uses locators.
- Protected built-in selection uses `scope: builtin`.
- Agent member arrays are not runtime-ordered selections.
- Missing relationships remain visible through partial resolution instead of being hidden.
- Artifact enablement does not alter declaration resolution.
- Collection membership does not own or delete Agents.
- Managed Agent replacement uses Artifact Source publication and immutable Definitions.
- No additional sidecar file, publication record, release record, or Agent-specific database is introduced.
- No compatibility, migration, or dual-storage design is part of this HLD.

## 14. Runtime boundary

The Agent Store ends at the resolved Agent declaration graph.

A future Agent runtime may define its own request types for:

- User messages.
- Tool user inputs.
- Tool invocation arguments.
- MCP conversation context.
- Model invocation values.
- Skill activation state.
- Workflow or Loop execution.
- Conversation and execution state.

Those values must not be persisted as Agent Store declaration state.

The runtime consumes the shared Agent resolver result, including relationship availability, target provenance, `overrides`, `use`, and diagnostics. It is responsible for deciding whether the resolved declarations are executable in the current runtime environment.

## 15. Implementation status

Status terminology:

- `Available` means a backend implementation path exists.
- `Pending` means verification, migration, or an approved follow-up remains.
- `Deferred` means intentionally outside the current Agent Store boundary.
- Status does not assert that all repository-wide builds, tests, generated bindings, static analysis, or migration verification has completed.

| Capability                                              | Status     | Notes                                                                                                         |
| ------------------------------------------------------- | ---------- | ------------------------------------------------------------------------------------------------------------- |
| Source-backed Agent Artifact catalog                    | Available  | Reads use the existing Artifact Store and Agent contract.                                                     |
| Shared Agent resolution and capability plans            | Available  | Uses the existing typed resolver and fallback registrations.                                                  |
| Agent-only Plugin Collection domain                     | Available  | Managed Collections permit named, contained, and selector Agent members.                                      |
| Agent Collection baseline provisioning                  | Available  | One editable non-deletable `agent-baseline` Plugin is provisioned per user Root.                              |
| Managed Agent creation                                  | Available  | Requires explicit Agent Collection selection and creates a located external membership.                       |
| Managed Agent replacement                               | Available  | Requires expected Artifact revision and source generation.                                                    |
| Managed Agent deletion                                  | Available  | Removes the package, preserves relationships, and purges the removed root Agent Artifact.                     |
| Agent attach and detach                                 | Available  | Same-Root, protected built-in, and unsupported cross-Root cases are handled explicitly.                       |
| Agent and Collection enablement                         | Available  | Uses universal `Artifact.Enabled`; enablement does not alter resolution.                                      |
| Built-in Agent package hydration                        | Available  | Uses the shared protected Root and package hydration markers.                                                 |
| Built-in package resolution admission                   | Available  | Final hydration requires complete Plugin capability resolution.                                               |
| Built-in Agent local enablement                         | Available  | Uses protected Artifact metadata mutation, without an Agent overlay Store.                                    |
| Built-in Assistant Preset conversion                    | Available  | Built-in presets are represented as protected Plugin and Agent packages using `plugin.yaml` and `agent.yaml`. |
| Built-in Agent package layout                           | Available  | Each package uses `plugin.yaml` and direct `<agent-name>/agent.yaml` children.                                |
| Legacy Tool selection conversion                        | Available  | Legacy Tool slugs become named Tool relationships with `overrides.autoExecute`.                               |
| Legacy starter text conversion                          | Available  | Starter text becomes contained Text with `insert: user-message`.                                              |
| Legacy core Skill conversion                            | Available  | `markdown-output`, `use-explicit-tools-batched`, and `grounded-local-work` map to `use.mode: instructions`.   |
| Legacy active Skill conversion                          | Available  | Every supplied specialist Skill ArtifactRef maps to a protected named Skill with `use.mode: active`.          |
| Legacy Model system-prompt preference                   | Not needed | Supplied presets had no selected Model reference, so the boolean had no portable target relationship.         |
| Assistant Preset compatibility endpoint                 | Removed    | The Agent Store exposes no Assistant Preset request or response type.                                         |
| Agent-specific persistence database                     | Removed    | Only ordinary Artifact Store metadata and managed Source packages are used.                                   |
| Agent runtime                                           | Deferred   | Execution, runtime inputs, state, scheduling, and invocation remain outside Agent Store.                      |
| URL, Git, package, archive, and command materialization | Deferred   | Portable locators remain supported declaration data.                                                          |
| Test and acceptance coverage                            | Pending    | Unit, integration, hydration-recovery, concurrency, and end-to-end coverage remain to be added.               |
| Frontend and Wails binding migration                    | Pending    | The backend wrapper exists; frontend exposure and binding generation must be completed.                       |
| Legacy Assistant Preset retirement                      | Pending    | Existing callers must migrate before legacy Store removal.                                                    |
| Atomic membership and package publication               | Pending    | Current approved behavior preserves an unavailable relationship if independent package publication fails.     |

Built-in Agent package roots currently include:

```text
core-agents
software-development-agents
product-leadership-agents
technical-content-writing-agents
research-analysis-agents
```

Every former built-in Assistant Preset is represented by one protected Agent Artifact. Legacy Artifact IDs, bundle IDs, versions, timestamps, and Assistant Preset compatibility fields are not persisted in Agent declarations.
