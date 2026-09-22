# Artifact Store, Resolution, and Ecosystem HLD

- [1. Purpose](#1-purpose)
- [2. Goals and scope](#2-goals-and-scope)
  - [2.1 Goals](#21-goals)
  - [2.2 In scope](#22-in-scope)
  - [2.3 Boundary](#23-boundary)
- [3. Requirements](#3-requirements)
  - [3.1 Store requirements](#31-store-requirements)
  - [3.2 Source requirements](#32-source-requirements)
  - [3.3 Resolution requirements](#33-resolution-requirements)
  - [3.4 Composition requirements](#34-composition-requirements)
  - [3.5 Managed authoring requirements](#35-managed-authoring-requirements)
  - [3.6 Consumer requirements](#36-consumer-requirements)
- [4. Design principles and invariants](#4-design-principles-and-invariants)
  - [4.1 Store records occurrences, not composition ownership](#41-store-records-occurrences-not-composition-ownership)
  - [4.2 Resolution is observational](#42-resolution-is-observational)
  - [4.3 Declaration availability is not runtime readiness](#43-declaration-availability-is-not-runtime-readiness)
  - [4.4 Physical and semantic identity remain separate](#44-physical-and-semantic-identity-remain-separate)
  - [4.5 Built-in fallback is selection, not activation](#45-built-in-fallback-is-selection-not-activation)
  - [4.6 Source order has no semantic authority](#46-source-order-has-no-semantic-authority)
  - [4.7 Locator representation and materialization are separate](#47-locator-representation-and-materialization-are-separate)
- [5. Desired user experience](#5-desired-user-experience)
- [6. Primary workflows](#6-primary-workflows)
  - [6.1 Repository onboarding](#61-repository-onboarding)
  - [6.2 Ordinary capability read](#62-ordinary-capability-read)
  - [6.3 Explicit refresh](#63-explicit-refresh)
  - [6.4 Managed Skill or MCP creation](#64-managed-skill-or-mcp-creation)
  - [6.5 Built-in bootstrap](#65-built-in-bootstrap)
- [7. Conceptual architecture](#7-conceptual-architecture)
  - [7.1 Terminology](#71-terminology)
- [8. Root model](#8-root-model)
- [9. Source model](#9-source-model)
  - [9.1 Source responsibilities](#91-source-responsibilities)
  - [9.2 Source registration](#92-source-registration)
  - [9.3 Source refresh](#93-source-refresh)
- [10. Physical-format adapters](#10-physical-format-adapters)
- [11. Definition model](#11-definition-model)
- [12. Artifact model](#12-artifact-model)
- [13. Resource verification](#13-resource-verification)
- [14. Locator resolution architecture](#14-locator-resolution-architecture)
  - [14.1 Local path resolution](#141-local-path-resolution)
  - [14.2 External locator resolution](#142-external-locator-resolution)
  - [14.3 Context-dependent commands](#143-context-dependent-commands)
- [15. Resolver architecture](#15-resolver-architecture)
  - [15.1 Per-type registration](#151-per-type-registration)
  - [15.2 Public typed APIs](#152-public-typed-apis)
  - [15.3 Private shared core](#153-private-shared-core)
- [16. Named lookup and fallback](#16-named-lookup-and-fallback)
  - [16.1 Default named lookup](#161-default-named-lookup)
  - [16.2 Explicit built-in lookup](#162-explicit-built-in-lookup)
  - [16.3 Generic fallback providers](#163-generic-fallback-providers)
  - [16.4 Fallback restrictions](#164-fallback-restrictions)
- [17. Relationship resolution](#17-relationship-resolution)
  - [17.1 Relationship result](#171-relationship-result)
  - [17.2 Named relationships](#172-named-relationships)
  - [17.3 Located relationships](#173-located-relationships)
  - [17.4 Contained relationships](#174-contained-relationships)
  - [17.5 Source-selected aliases](#175-source-selected-aliases)
- [18. Member selectors and discovery](#18-member-selectors-and-discovery)
  - [18.1 Eligibility](#181-eligibility)
  - [18.2 Selector meaning](#182-selector-meaning)
  - [18.3 Selector scope](#183-selector-scope)
  - [18.4 Selector results](#184-selector-results)
  - [18.5 Selector refresh](#185-selector-refresh)
- [19. Composition expansion](#19-composition-expansion)
  - [19.1 Plugin expansion](#191-plugin-expansion)
  - [19.2 Agent expansion](#192-agent-expansion)
  - [19.3 Team expansion](#193-team-expansion)
  - [19.4 Loop expansion](#194-loop-expansion)
  - [19.5 Workflow expansion](#195-workflow-expansion)
  - [19.6 Workspace expansion](#196-workspace-expansion)
- [20. Cycles, limits, and partial validity](#20-cycles-limits-and-partial-validity)
- [21. Ordering and structural identity](#21-ordering-and-structural-identity)
  - [21.1 Non-positional relationships](#211-non-positional-relationships)
  - [21.2 Stable identity](#212-stable-identity)
  - [21.3 Duplicate relationships](#213-duplicate-relationships)
  - [21.4 Source revisions](#214-source-revisions)
- [22. Capability plans and completeness](#22-capability-plans-and-completeness)
- [23. Workspace behavior](#23-workspace-behavior)
  - [23.1 Explicit capability selection](#231-explicit-capability-selection)
  - [23.2 Catalog and capability views](#232-catalog-and-capability-views)
  - [23.3 Completeness](#233-completeness)
  - [23.4 Read-only planning](#234-read-only-planning)
- [24. Consumer integration](#24-consumer-integration)
  - [24.1 Common requirements](#241-common-requirements)
  - [24.2 Text and prompt consumers](#242-text-and-prompt-consumers)
  - [24.3 Tool and Model consumers](#243-tool-and-model-consumers)
  - [24.4 Skill consumer](#244-skill-consumer)
  - [24.5 MCP consumer](#245-mcp-consumer)
  - [24.6 Workflow consumer](#246-workflow-consumer)
- [25. Artifact Store boundaries](#25-artifact-store-boundaries)
  - [25.1 Persisted state](#251-persisted-state)
  - [25.2 Derived state](#252-derived-state)
  - [25.3 Purge behavior](#253-purge-behavior)
- [26. Artifact enablement and local state](#26-artifact-enablement-and-local-state)
  - [26.1 Artifact enablement](#261-artifact-enablement)
  - [26.2 Local state ownership](#262-local-state-ownership)
- [27. Managed Plugin authoring](#27-managed-plugin-authoring)
  - [27.1 Storage model](#271-storage-model)
  - [27.2 Portable and managed domains](#272-portable-and-managed-domains)
  - [27.3 Baseline Plugins](#273-baseline-plugins)
  - [27.4 Membership semantics](#274-membership-semantics)
  - [27.5 Publication and replacement](#275-publication-and-replacement)
  - [27.6 Multi-package creation behavior](#276-multi-package-creation-behavior)
  - [27.7 Supported operations](#277-supported-operations)
  - [27.8 Plugin deletion](#278-plugin-deletion)
- [28. Managed Skill and MCP workflows](#28-managed-skill-and-mcp-workflows)
  - [28.1 Managed Skill](#281-managed-skill)
  - [28.2 Managed MCP server or policy](#282-managed-mcp-server-or-policy)
- [29. Built-in content and protected Roots](#29-built-in-content-and-protected-roots)
  - [29.1 Common declaration behavior](#291-common-declaration-behavior)
  - [29.2 Built-in package composition](#292-built-in-package-composition)
  - [29.3 Protected Root behavior](#293-protected-root-behavior)
  - [29.4 Package hydration](#294-package-hydration)
- [30. Implementation requirements](#30-implementation-requirements)
  - [30.1 Store and Source](#301-store-and-source)
  - [30.2 Resolver](#302-resolver)
  - [30.3 Discovery](#303-discovery)
  - [30.4 Consumers](#304-consumers)
- [31. Current implementation status](#31-current-implementation-status)
  - [31.1 Status interpretation](#311-status-interpretation)
  - [31.2 Sources, persistence, and verification](#312-sources-persistence-and-verification)
  - [31.3 Resolver and discovery](#313-resolver-and-discovery)
  - [31.4 Physical formats](#314-physical-formats)
  - [31.5 Workspace and capability planning](#315-workspace-and-capability-planning)
  - [31.6 Managed Plugins and authoring](#316-managed-plugins-and-authoring)
  - [31.7 Built-in packages](#317-built-in-packages)
  - [31.8 MCP platform and runtime](#318-mcp-platform-and-runtime)
  - [31.9 Pending completion work](#319-pending-completion-work)
  - [31.10 Deferred or unsupported ecosystem capabilities](#3110-deferred-or-unsupported-ecosystem-capabilities)
    - [Source and locator capabilities](#source-and-locator-capabilities)
    - [Authoring and frontend](#authoring-and-frontend)
    - [Execution runtimes](#execution-runtimes)

## 1. Purpose

This HLD defines how portable declarations become source-backed Artifacts and resolved capability graphs.

Portable declaration syntax is defined in [Portable Artifact Declaration Contracts HLD](./artifact_contracts_hld.md).

The complete platform flow is:

```text
Physical Sources
  -> source entries
  -> physical-format decoders
  -> normalized Definitions
  -> source-backed Artifacts
  -> typed resolution graph
  -> consumer capability plan
  -> runtime or management consumer
```

The ecosystem must support repository-authored content, managed packages, protected built-in content, explicit dynamic discovery, partial resolution, and consumer-specific readiness without turning composition into Store ownership.

## 2. Goals and scope

### 2.1 Goals

The ecosystem must provide:

- Root-scoped semantic identity.
- Source-backed immutable Definitions.
- Stable Artifact occurrences.
- Verified source resources.
- Per-type resolver registration.
- Per-type public resolver APIs.
- A shared private resolution core.
- Named, located, contained, and selector relationship resolution.
- Current Root lookup with protected built-in fallback.
- Explicit built-in-only lookup.
- Generic fallback providers.
- Source-selected alias traversal.
- Explicit source refresh.
- Read-only normal resolution.
- Relationship-level availability and diagnostics.
- Stable non-positional relationship identity.
- Partial and strict capability plans.
- Workspace-driven explicit capability selection.
- Managed Plugin authoring without Store ownership.
- Package-scoped built-in hydration.
- Clear separation between declaration availability and runtime readiness.

### 2.2 In scope

This HLD defines:

- Root, Source, Definition, Artifact, and Resource responsibilities.
- Source registration, discovery, refresh, and verification.
- Physical-format adapters.
- Locator-resolution boundaries.
- Resolver architecture and lookup behavior.
- Member-selector discovery and expansion.
- Alias and terminal Artifact behavior.
- Composition expansion.
- Capability plans.
- Workspace behavior.
- Managed Plugin, Skill, MCP, and policy authoring.
- Built-in package behavior.
- Artifact enablement.
- Runtime-consumer boundaries.
- Current implementation and deferred capabilities.

### 2.3 Boundary

This document does not redefine portable wire schemas, and resolution itself does not execute Models, Tools, Agents, Teams, Loops, or Workflows.

## 3. Requirements

### 3.1 Store requirements

Artifact Store must persist:

- Roots.
- Sources.
- Source discovery and refresh state.
- Immutable Definitions.
- Source-backed Artifact occurrences.
- Source bindings.
- Verified resource metadata.
- Artifact-local data.

Artifact Store must not persist generic:

- Plugin membership rows.
- Selector-match rows.
- Parent ownership.
- Reverse composition membership.
- Relationship array positions.
- Composition cascade rules.
- Runtime activation inferred from composition.

### 3.2 Source requirements

Sources must own:

- Physical content access.
- Candidate enumeration.
- Decoder registration and hints.
- Explicit refresh.
- Source integrity and generation.
- Path safety.
- Discovery limits.
- Resource verification.

Ordinary resolver and capability-plan reads must not mutate Source discovery or refresh Sources.

### 3.3 Resolution requirements

The resolver must:

- Return a typed graph without executing it.
- Preserve every declared relationship occurrence.
- Report each relationship as `available`, `unavailable`, or `ambiguous`.
- Never use source order to break ambiguity.
- Resolve located references strictly.
- Resolve contained declarations through stable structural occurrences.
- Expand selectors against indexed Source state.
- Follow source-selected aliases to terminal targets.
- Detect alias and composition cycles.
- Enforce depth and node limits.
- Preserve relationship behavior and provenance.
- Keep unavailable external relationships visible.
- Keep runtime readiness separate from declaration availability.

### 3.4 Composition requirements

- Composition must not mutate target definitions or local state.
- Composition arrays must not define precedence or execution order.
- Exact duplicate normalized relationships must be rejected.
- Distinct relationships to one terminal target must remain distinct in the graph.
- Consumer deduplication must not erase relationship occurrence diagnostics.
- Nested Workspace relationships must remain visible but resolve as unavailable.

### 3.5 Managed authoring requirements

Managed authoring must:

- Store Plugins and independently authored targets as source-backed packages.
- Require explicit compatible Plugin selection for managed Skill, MCP server, and MCP policy creation.
- Create external membership by default.
- Preserve dangling membership when a target is absent.
- Allow one target to belong to multiple Plugins.
- Never delete an independent target merely because membership is removed.
- Protect baseline Plugins through application-level policy.
- Require explicit replacement intent before replacing non-equivalent package content.

### 3.6 Consumer requirements

Consumers must:

- Use typed resolver APIs.
- Preserve diagnostics.
- Distinguish Artifact-backed and mapped fallback targets.
- Apply relationship behavior only where defined.
- Decide runtime readiness independently.
- Materialize verified resources rather than opening arbitrary paths.
- Choose strict or partial completeness explicitly.

## 4. Design principles and invariants

### 4.1 Store records occurrences, not composition ownership

An Artifact represents one declaration occurrence.

A Plugin, Workspace, Agent, or other composition is persisted as an ordinary Definition and Artifact. Its relationships remain declaration content interpreted by resolvers.

### 4.2 Resolution is observational

Normal resolution observes indexed state.

```text
Resolve
  -> read indexed Sources, Definitions, and Artifacts
  -> produce graph
  -> no Source mutation
```

Discovery updates occur only through explicit registration or refresh.

### 4.3 Declaration availability is not runtime readiness

Resolution answers:

```text
Can this relationship identify one target?
```

A runtime answers:

```text
Can this target be used now with the required resources,
credentials, installation values, and runtime support?
```

### 4.4 Physical and semantic identity remain separate

A locator selects a declaration occurrence. It does not alter the selected target's semantic name.

Aliases may provide multiple physical paths to one terminal Artifact.

### 4.5 Built-in fallback is selection, not activation

Protected built-in lookup occurs only for a declared named relationship. It does not automatically add all built-in capabilities to a Workspace or runtime session.

### 4.6 Source order has no semantic authority

Source order cannot:

- Resolve ambiguity.
- Establish member precedence.
- Define execution order.
- Define contained declaration identity.
- Define selector identity.
- Define ownership.

### 4.7 Locator representation and materialization are separate

Portable declarations may retain URL, Git, package, command, and other supported locator kinds even when the current installation has only local path Source adapters.

## 5. Desired user experience

A repository can combine familiar files and canonical declarations.

```text
checkout-service/
  AGENTS.md
  README.md
  .mcp.json

  agents/
    reviewer.agent.yaml
    security-reviewer.agent.md

  plugins/
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

The Source may produce Artifacts such as:

```text
text/repository-rules/instructions
text/readme/user-message
skill/code-review
mcp/github
agent/reviewer
plugin/repository-review
workflow/change-workflow
workspace/checkout-service
```

The expected user flow is:

```text
Select repository or Workspace
  -> register or reuse repository Source
  -> discover Workspace and declaration candidates
  -> explicitly refresh reachable declaration closure
  -> resolve Workspace members
  -> build capability plan
  -> let consumers materialize or execute selected capabilities
```

Unavailable and ambiguous relationships remain visible rather than disappearing from the experience.

## 6. Primary workflows

### 6.1 Repository onboarding

1. The application registers a filesystem Source for the repository.
2. Initial Source discovery locates supported declarations and Workspace candidates.
3. The user or application selects a Workspace.
4. `RefreshWorkspace` computes the reachable local locator and selector closure.
5. Affected Source discovery is updated.
6. Affected Sources are refreshed.
7. The Workspace is resolved from indexed state.
8. Consumer-specific capability plans are produced.

Independent concurrently active declaration universes should use separate Sources.

### 6.2 Ordinary capability read

```text
ResolveWorkspace
  -> load indexed Workspace Artifact
  -> resolve declared members
  -> recursively expand relationships
  -> preserve partial status
  -> produce capability plan
```

This operation does not scan the filesystem or refresh a Source.

The same read-only rule applies to:

- Plugin resolution.
- Agent resolution.
- Team resolution.
- Workflow resolution.
- Prompt planning.
- Skill planning.
- MCP planning.
- Runtime-plan generation.

### 6.3 Explicit refresh

```text
Selected composition root
  -> inspect reachable selectors and located relationships
  -> calculate local discovery closure
  -> update only affected Source discovery
  -> refresh affected Sources
  -> repeat within bounded limits
```

Refresh is explicit because source mutation must not be hidden inside reads.

### 6.4 Managed Skill or MCP creation

```text
Explicitly select compatible editable Plugin
  -> add external membership to Plugin declaration
  -> publish independent target package
  -> refresh affected managed Sources
  -> resolve membership and target
```

If target publication fails after membership publication, the relationship remains declared and unavailable. A retry re-reads the Plugin revision.

### 6.5 Built-in bootstrap

```text
Application startup
  -> inspect declared built-in packages and fingerprints
  -> verify package bytes and expected Artifacts
  -> repair changed or incomplete packages
  -> remove stale known packages
  -> preserve unaffected package ArtifactRefs and local state
```

A hydration marker alone is not sufficient evidence that package content remains present.

## 7. Conceptual architecture

```text
Root
  -> Source
    -> source entry
      -> Decoder
        -> Definition
          -> Artifact
            -> verified Resource

Artifact + Definition
  -> registered type resolver
    -> relationship graph
      -> capability plan
        -> consumer
```

### 7.1 Terminology

| Term                    | Meaning                                                               |
| ----------------------- | --------------------------------------------------------------------- |
| Root                    | Identity, authorization, and default symbolic resolution boundary     |
| Source                  | Registered physical or managed content origin                         |
| Source entry            | Discoverable file, directory, or managed source item                  |
| Decoder                 | Adapter that converts a physical format into portable declarations    |
| Definition              | Immutable normalized semantic content                                 |
| Artifact                | Local addressable record for one declaration occurrence               |
| Resource                | Verified source material associated with an Artifact                  |
| ArtifactRef             | Stable local reference to an Artifact occurrence                      |
| Declaration occurrence  | Physical or structural place where a declaration is written           |
| Relationship occurrence | One declared member, selector, node target, or singular relationship  |
| Terminal Artifact       | Final Artifact reached after following source-selection aliases       |
| Mapped target           | Non-Artifact target returned by a fallback provider                   |
| Capability plan         | Resolved consumer-safe graph with status, provenance, and diagnostics |

## 8. Root model

A Root owns:

- Registered Sources.
- Indexed Definitions and Artifacts.
- Artifact-local state.
- Symbolic lookup scope.
- Source authorization boundary.

The default semantic lookup key is:

```text
(type, logical name)
```

Text additionally uses insertion identity.

Multiple Roots may independently contain the same name.

A Root may also contain several occurrences of the same semantic identity. An unlocated relationship becomes ambiguous unless those occurrences reduce to the same terminal Artifact.

Arbitrary cross-Root imports are not supported.

The protected built-in Root is a special authorized fallback scope. It is not an arbitrary cross-Root import mechanism.

## 9. Source model

### 9.1 Source responsibilities

A Source separates physical configuration from declaration discovery.

```text
Source.Config
  -> filesystem path
  -> embedded provider and root
  -> managed storage location

Source.Discovery
  -> exact entries
  -> directory roots
  -> include and exclude patterns
  -> decoder hints
  -> allowed decoders
  -> expected digests
  -> scan limits
```

Current Source kinds are:

```text
fs-directory
embedded-directory
managed-directory
```

### 9.2 Source registration

Source registration may establish:

- Exact declaration files.
- Scanned directory roots.
- Decoder restrictions.
- Expected content digests.
- Initial Workspace candidate discovery.
- Managed package locations.

Portable Workspace declarations do not own this Source configuration.

### 9.3 Source refresh

A Source refresh:

- Enumerates configured candidates.
- Validates path boundaries.
- Selects physical-format decoders.
- Produces normalized Definitions.
- Reconciles source-backed Artifacts.
- Updates Source generation and diagnostics.
- Preserves ordinary Artifact-local state for continuing occurrences.

A disabled Source affects discovery and resource availability. It is separate from Artifact enablement.

## 10. Physical-format adapters

Supported physical inputs normalize as follows.

| Physical input                  | Produced behavior                                                                            |
| ------------------------------- | -------------------------------------------------------------------------------------------- |
| Canonical JSON                  | One root declaration and any named contained declarations                                    |
| Canonical YAML                  | One root declaration and any named contained declarations                                    |
| `AGENTS.md`                     | Text with `insert=instructions`                                                              |
| `CLAUDE.md`                     | Text with `insert=instructions`                                                              |
| `README.md`                     | Text with `insert=user-message`                                                              |
| `llms.txt`                      | Text with `insert=user-message`                                                              |
| Selected documentation Markdown | Source-backed Text                                                                           |
| `SKILL.md`                      | One independent Skill                                                                        |
| Direct Skill directory          | One Skill rooted at the package                                                              |
| `.mcp.json`                     | One independent MCP Artifact per configured server                                           |
| `mcp.json`                      | One independent MCP Artifact per configured server                                           |
| `AGENT.md`                      | Agent, optional instruction Text for non-empty body, and contained front-matter declarations |
| `*.agent.md`                    | Agent, optional instruction Text for non-empty body, and contained front-matter declarations |
| Canonical Plugin YAML or JSON   | Plugin plus contained declarations                                                           |
| Workspace manifest              | Workspace plus contained declarations                                                        |

One physical file may emit multiple Artifacts.

A locator-bearing external member does not emit a new contained Artifact merely because it has a locator.

A standard `SKILL.md` adapter emits an explicit declaration-relative locator for its source and treats the containing directory as the verified Skill package root.

Physical adapters may derive names deterministically when the source format has no explicit portable name.

## 11. Definition model

A Definition is the immutable normalized result of decoding one declaration occurrence.

Conceptually:

```text
Definition {
  Digest
  Kind
  SchemaID
  SchemaVersion
  LogicalName
  DisplayName
  Description
  Labels
  Body
  Dependencies
}
```

Schema identity in this structure is internal Store registration data. It is not copied into portable declaration instances.

Definition digests support:

- Integrity verification.
- Equality.
- Change detection.
- Immutable persistence.
- Managed package verification.
- Runtime version tracking.

Reordering an unordered source array may change Definition bytes and digest without changing composition behavior.

## 12. Artifact model

An Artifact is a local addressable record for one declaration occurrence.

Conceptually:

```text
Artifact {
  ID
  RootID
  SourceBinding

  Kind
  LogicalName
  ResolvedDefinition
  SourceContentDigest

  State
  Diagnostics

  DisplayName
  Enabled
  Data
}
```

An Artifact provides:

- Stable local addressability through `ArtifactRef`.
- A binding to one Source occurrence.
- Current Definition selection.
- Source and decode diagnostics.
- Generic local enablement.
- Namespaced local consumer data.

Contained declarations receive stable source subresource occurrences. Their containing composition does not become their Store owner.

## 13. Resource verification

Resource access verifies:

```text
Artifact
  -> current Definition
  -> Source binding
  -> Source generation
  -> declaration content digest
  -> expected resource boundary
  -> verified source material
```

Consumers must not reopen arbitrary repository paths directly.

A missing resource may make a target unready for runtime use without invalidating an otherwise resolvable composition relationship.

Bulk verification-session reuse is an optimization and does not change this trust boundary.

## 14. Locator resolution architecture

### 14.1 Local path resolution

A local path locator is resolved relative to the declaration Source entry.

Resolution must:

- Normalize the relative path.
- Remain inside the opened Source.
- Enforce portable path rules.
- Avoid process-working-directory interpretation.
- Avoid arbitrary filesystem access.

### 14.2 External locator resolution

URL, Git, package, command, and future locator kinds are handled by explicit Locator Resolver or Source adapter registrations.

A Locator Resolver may:

- Identify an already registered Source.
- Create an explicitly authorized Source through a refresh workflow.
- Materialize verified content.
- Return an unavailable result when unsupported.

Ordinary resolution must not perform hidden network fetches or Source mutations.

The portable locator shape remains valid even when a corresponding adapter is unavailable.

### 14.3 Context-dependent commands

A command locator remains part of the portable representation. Whether it identifies a declaration source, executable target, or unsupported context is decided by the concrete type resolver and consumer.

Typed Tool and MCP declarations may use their own explicit implementation or transport command fields without narrowing the common locator union.

## 15. Resolver architecture

### 15.1 Per-type registration

The application registers one resolver for each supported Artifact type.

```text
ResolverRegistry
  -> TextResolver
  -> ModelResolver
  -> ToolResolver
  -> SkillResolver
  -> MCPResolver
  -> MCPPolicyResolver
  -> PluginResolver
  -> AgentResolver
  -> TeamResolver
  -> LoopResolver
  -> WorkflowResolver
  -> WorkspaceResolver
```

Each resolver owns:

- Type-specific declaration interpretation.
- Child relationship expansion.
- Selector eligibility.
- Located-source alias behavior.
- Fallback-provider support.
- Consumer projection.

### 15.2 Public typed APIs

Consumers use typed APIs such as:

```text
ResolveText
ResolveModel
ResolveTool
ResolveSkill
ResolveMCP
ResolveMCPPolicy
ResolvePlugin
ResolveAgent
ResolveTeam
ResolveLoop
ResolveWorkflow
ResolveWorkspace
```

Composition-capable resolvers may expose typed refresh APIs:

```text
RefreshPlugin
RefreshAgent
RefreshTeam
RefreshWorkspace
```

Public consumers must not depend on a generic:

```text
ResolveArtifact(type, name)
```

### 15.3 Private shared core

The private core provides:

- Current Root lookup.
- Protected built-in lookup.
- Fallback-provider dispatch.
- Located occurrence selection.
- Contained occurrence selection.
- Selector expansion.
- Alias traversal.
- Cycle detection.
- Limits.
- Diagnostics.
- Stable ordering.
- Capability-plan graph construction.

The shared core must not import Tool, Model, Agent, Workflow, MCP connection, prompt, or secret runtimes.

## 16. Named lookup and fallback

### 16.1 Default named lookup

An unscoped named relationship resolves in this order:

```text
Current Root Artifact
  -> protected built-in Root Artifact
  -> registered fallback provider
  -> unavailable
```

A current Root occurrence takes precedence over a protected built-in occurrence.

A protected built-in Artifact takes precedence over a non-Artifact mapped fallback.

### 16.2 Explicit built-in lookup

A relationship with `scope: builtin` resolves as:

```text
Protected built-in Root Artifact
  -> registered built-in fallback provider
  -> unavailable
```

It does not search the current Root.

The scope is:

- A lookup directive.
- Not a Root ID.
- Not a source locator.
- Not an ArtifactRef.
- Not a generic cross-Root import.

### 16.3 Generic fallback providers

Fallback providers are registered per Artifact type.

Tool and Model are the initial fallback-provider consumers.

A provider may return:

- A protected built-in Artifact-backed target.
- A non-Artifact mapped target.
- No target.

A mapped target has no:

- ArtifactRef.
- Definition.
- Source binding.
- Source resource.
- Artifact enablement.
- Artifact local data.
- Artifact lifecycle.

Mapped target provenance must identify:

- Fallback provider.
- Artifact type.
- Logical name.
- Lookup scope.
- Built-in classification.

### 16.4 Fallback restrictions

Fallback providers apply only to named external relationships.

They are not used for:

- Located relationships.
- Contained declarations.
- Member selectors.
- Source-selected aliases.
- Arbitrary caller-supplied ArtifactRefs.

Current Root ambiguity prevents built-in fallback.

Protected built-in ambiguity prevents mapped fallback.

Artifact enablement and runtime readiness do not affect declaration lookup.

## 17. Relationship resolution

### 17.1 Relationship result

Every relationship occurrence has one status:

```text
available
unavailable
ambiguous
```

The result preserves:

- Relationship occurrence identity.
- Member form.
- Selector occurrence identity.
- Selector-match identity.
- Lookup scope.
- Locator.
- Target provenance.
- Alias provenance.
- Relationship `overrides`.
- Relationship `use`.
- Diagnostics.

The resolver must not silently drop, replace, execute, or mutate a relationship.

### 17.2 Named relationships

An ordinary named target resolves by semantic identity.

Text additionally requires its insertion identity.

Outcomes are:

- No terminal target, `unavailable`.
- One terminal target, `available`.
- More than one distinct terminal target, `ambiguous`.

Several aliases resolving to the same terminal Artifact count as one symbolic target.

### 17.3 Located relationships

A located relationship resolves by:

```text
type
name
locator
optional Text insert
optional MCP server selector
```

It must:

1. Resolve the locator relative to the containing occurrence or through an authorized Locator Resolver.
2. Select the declaration occurrence at that location.
3. Confirm expected type.
4. Confirm expected name.
5. Confirm Text insertion identity when applicable.
6. Confirm MCP server selection when applicable.
7. Follow supported source aliases to a terminal target.

It does not:

- Search the current Root by name.
- Search the protected built-in Root.
- Use fallback providers.
- Merge local target body fields.
- Fall back to another location.

### 17.4 Contained relationships

Contained relationships resolve through stable structural occurrences.

```text
Containing declaration occurrence
  -> stable relationship path
  -> contained Artifact
```

Examples:

```text
members/text/instructions/repository-rules
members/mcp/local-files
allowedTools/tool/searchfiles
body/agent/code-reviewer
nodes/review/agent/reviewer
loop/reviewer-loop
workflow/change-workflow
```

Contained resolution does not perform Root-wide symbolic lookup.

Removing the contained declaration from its source means it is no longer declared. It does not trigger generic composition-owned cascading deletion.

### 17.5 Source-selected aliases

Source-selected declarations may act as aliases to terminal Artifacts.

The resolver must:

- Follow alias chains.
- Detect alias cycles.
- Preserve alias provenance.
- Return terminal ArtifactRefs.
- Treat aliases to one terminal Artifact as one symbolic target.
- Preserve an unavailable alias as unavailable.

Aliases do not merge local connection, policy, capability, or other target fields into the terminal declaration.

Resource claims do not suppress physical declaration targets. Same-Source Skill and MCP targets remain selectable.

## 18. Member selectors and discovery

### 18.1 Eligibility

Selector support is registered by Artifact type and constrained by the containing relationship field.

Text and Workspace do not support member selectors.

Current selector-capable types may include:

```text
model
tool
skill
mcp
mcp.policy
plugin
agent
team
loop
workflow
```

The containing contract may permit only a subset.

### 18.2 Selector meaning

A selector expands conceptually as:

```text
Selector base
  -> candidate declaration enumeration
  -> physical-format decoding
  -> Artifact type filtering
  -> logical-name filtering
  -> selected source-backed occurrences
```

Selectors do not select:

- Arbitrary prompt files.
- Root-wide symbolic names.
- Protected built-in fallback targets.
- Mapped fallback targets.
- Runtime-ready targets.
- Enabled-only targets.
- Metadata query results.

### 18.3 Selector scope

A current selector base is local to the declaring Source.

The implementation must:

1. Resolve `base` relative to the containing declaration occurrence.
2. Stay inside the containing Source.
3. Stay inside the containing Root.
4. Enumerate declaration candidates below the base.
5. Apply source-path inclusion and exclusion.
6. Decode supported physical formats.
7. Filter by selected Artifact type.
8. Apply logical-name inclusion and exclusion.
9. Produce stable match occurrences.

A selector must not:

- Scan an arbitrary filesystem path.
- Scan another mutable Root.
- Scan the protected built-in Root from a mutable Source.
- Fetch remote content without a registered Source adapter.
- Treat Text resource patterns as declaration discovery.

### 18.4 Selector results

A selector can produce:

```text
available with zero matches
available with one or more matches
unavailable selector
```

Zero matches are a valid empty membership set.

A selector is unavailable when its base cannot be inspected, is unsupported, or escapes the permitted Source boundary.

Each selected occurrence retains its own relationship result.

```text
Selector available
  -> Artifact A: available
  -> Artifact B: unavailable
  -> Artifact C: available
```

A decode error for one source entry remains a Source diagnostic and does not invalidate unrelated valid matches.

### 18.5 Selector refresh

Selector matching during resolution is read-only.

Explicit refresh ensures the necessary candidate declarations are indexed:

```text
Composition root
  -> reachable selectors
  -> reachable local located declarations
  -> discovery closure
  -> affected Source updates
  -> Source refresh
```

The closure process must be bounded by depth, node, Source, and iteration limits.

## 19. Composition expansion

Expansion is recursive and typed.

| Artifact type | Expanded relationships        |
| ------------- | ----------------------------- |
| `plugin`      | `members`                     |
| `agent`       | `members`, `loop`, `workflow` |
| `team`        | `members`, `loop`, `workflow` |
| `skill`       | `allowedTools`                |
| `mcp`         | Direct policy relationship    |
| `loop`        | `body`                        |
| `workflow`    | Node targets                  |
| `workspace`   | `members`                     |

The resolver does not expand:

- Text source files into child Artifacts.
- MCP included tools into child Artifacts.
- MCP policy map keys into child Artifacts.
- Runtime Tool calls into graph relationships.

### 19.1 Plugin expansion

Plugin expansion recursively traverses declared members.

Plugin contributes selection and grouping only. It does not apply runtime behavior or own its targets.

### 19.2 Agent expansion

Agent expansion preserves:

- Tool `overrides.autoExecute`.
- Model `overrides.includeSystemPrompt`.
- Skill `use.mode`.

The resolver carries this behavior but does not execute it.

### 19.3 Team expansion

Team expansion traverses Agent and Plugin members and the optional Loop or Workflow program.

Team has no additional relationship behavior in v1.

### 19.4 Loop expansion

Loop expansion resolves one body target.

The resolver validates and resolves the body but does not iterate it.

### 19.5 Workflow expansion

Every node target is resolved independently.

Node identity is:

```text
Workflow occurrence + node ID
```

The graph is determined by node IDs, starts, edges, joins, and matchers. Source-array order is not execution order.

### 19.6 Workspace expansion

Workspace expansion traverses explicitly declared members.

Workspace does not infer capabilities from every Artifact in the Root.

Workspace Text members contribute source text. Workspace selectors contribute declared Artifacts. These remain separate operations.

A nested Workspace member remains represented but resolves as unavailable.

## 20. Cycles, limits, and partial validity

The resolver must detect:

- Alias cycles.
- Plugin composition cycles.
- Agent delegation cycles.
- Other recursive composition cycles.
- Maximum resolution depth.
- Maximum resolved node count.
- Refresh closure limits.

A cycle or exceeded limit affects the applicable relationship path.

It does not:

- Delete declarations.
- Create Store ownership.
- Invalidate unrelated relationship paths.
- Cause fallback to an unintended target.

An unavailable or ambiguous external relationship does not invalidate its containing Plugin, Agent, Team, Loop, Workflow, or Workspace declaration.

## 21. Ordering and structural identity

### 21.1 Non-positional relationships

The following arrays have no caller-defined semantic order:

```text
plugin.members
agent.members
team.members
workspace.members
skill.allowedTools
member-selector matches
workflow.nodes
workflow.edges
workflow.start
```

Moving an entry must not change:

- Relationship identity.
- Contained Artifact identity.
- Selector identity.
- Selector matches.
- Target local state.
- Lookup precedence.
- Store ownership.
- Runtime behavior.
- Cache identity.

### 21.2 Stable identity

Representative identities are:

```text
Named relationship
  -> parent occurrence + field + type + name + locator + scope
     + canonical overrides + canonical use

Contained declaration
  -> parent occurrence + field + type + name
     + Text insert where applicable

Selector
  -> parent occurrence + field + canonical selector fields
     + canonical overrides + canonical use

Workflow node
  -> workflow occurrence + node ID

Workflow edge
  -> workflow occurrence + from + to + canonical match
```

Array indexes are excluded.

### 21.3 Duplicate relationships

Exact normalized duplicates in one parent field are invalid.

Distinct relationships to one target remain valid when their relationship behavior differs.

```yaml
- type: tool
  name: searchfiles
  overrides:
    autoExecute: true

- type: tool
  name: searchfiles
  overrides:
    autoExecute: false
```

A consumer may deduplicate terminal ArtifactRefs in a capability list, but the graph retains both occurrences.

### 21.4 Source revisions

Reordering source arrays may change:

- Raw bytes.
- Source content digest.
- Definition digest.
- Artifact revision.

It must not change semantic composition.

Intrinsically ordered command and invocation argument arrays remain ordered.

## 22. Capability plans and completeness

A capability plan is a consumer-safe projection of the resolved graph.

It preserves:

- Relationship occurrence identity.
- Selector and selector-match identities.
- Artifact-backed, built-in, or mapped provenance.
- Availability status.
- Diagnostics.
- Relationship behavior.
- Lookup scope.
- Terminal target identity.

A consumer may require complete resolution:

```text
RequireComplete = true
```

A strict consumer rejects unavailable or ambiguous required relationships.

A non-strict consumer may use available portions while presenting diagnostics for incomplete paths.

Resolution availability does not assert that a target has:

- Required source resources.
- Installation values.
- Secrets.
- OAuth tokens.
- A selected profile.
- Runtime implementation.
- Active connection.
- Execution permission.

## 23. Workspace behavior

### 23.1 Explicit capability selection

A Workspace capability plan is built from `workspace.members` and their recursive composition.

The Workspace does not automatically activate:

- Every Plugin in the Root.
- Every Skill in the Root.
- Every MCP server in the Root.
- Every protected built-in capability.
- Baseline managed Plugins.

### 23.2 Catalog and capability views

A Workspace catalog may show every discovered Artifact kind.

A Workspace capability view shows only:

- Explicit Workspace members.
- Their expanded relationships.
- Available and unavailable selector matches.
- Relationship diagnostics.
- Consumer-specific readiness.

### 23.3 Completeness

Workspace runtime selection may use:

```text
WorkspaceRuntimeSelection.RequireComplete
```

When enabled, unavailable or ambiguous required relationships reject the selection with relationship-level diagnostics.

### 23.4 Read-only planning

These are read-only:

- Workspace reads.
- Workspace resolution.
- Workspace capability inspection.
- Prompt planning.
- Skill planning.
- MCP planning.
- Runtime-plan generation.

Only explicit refresh updates Source discovery.

## 24. Consumer integration

### 24.1 Common requirements

Consumers must:

- Use typed resolver APIs.
- Respect resolved Root authorization.
- Preserve relationship diagnostics.
- Distinguish Artifact-backed and mapped targets.
- Avoid arbitrary caller-supplied cross-Root ArtifactRefs.
- Apply relationship behavior only when defined.
- Decide runtime readiness separately.
- Avoid positional interpretation of composition arrays.

### 24.2 Text and prompt consumers

Text materialization supports:

- Inline content.
- One verified source file.
- One verified source directory.
- Inclusion and exclusion patterns.
- Deterministic normalized source-path ordering.
- Media type.
- Insertion target.

Routing is:

```text
insert=instructions
  -> behavioral or directive contribution

insert=user-message
  -> informational user-message contribution
```

The prompt consumer owns:

- Prompt budget.
- Truncation.
- Final contribution ordering.
- Request-specific content.
- Model-specific prompt assembly.

### 24.3 Tool and Model consumers

Tool and Model consumers may receive:

- Current Root Artifact-backed targets.
- Protected built-in Artifact-backed targets.
- Mapped fallback targets.

Only a consumer that explicitly supports the mapped target type may accept a mapped result.

A consumer requiring a verified declaration resource must reject a non-Artifact mapped target.

### 24.4 Skill consumer

```text
Resolved Skill
  -> verified SKILL.md
  -> verified Skill package directory
  -> runtime Skill registration
  -> resource and script operations
```

One Skill selected through several Plugins retains one Artifact identity.

### 24.5 MCP consumer

The MCP consumer must:

- Read direct typed declaration fields.
- Resolve the direct policy relationship.
- Use the terminal MCP ArtifactRef for installation-local state.
- Keep input values, secret references, selected profile, OAuth tokens, and connection state outside the portable declaration.
- Avoid interpreting Plugin membership as installation or activation.
- Keep generic Artifact enablement outside MCP Store and runtime internals unless an outer caller explicitly filters selection.

One MCP selected through several compositions retains one terminal installation identity.

### 24.6 Workflow consumer

```text
Resolved Workflow
  -> resolved node targets
  -> starts and edges
  -> join behavior
  -> output matchers
  -> scheduler and executor
```

The runtime owns:

- Scheduling.
- Invocation.
- Retries.
- Timeouts.
- Cancellation.
- State persistence.
- Output evaluation.
- Result handling.

It must not silently substitute another target for an unavailable node.

## 25. Artifact Store boundaries

### 25.1 Persisted state

Artifact Store persists:

```text
Root
Source
Definition
Artifact
Source binding
Artifact local data
Source refresh state
Resource verification state
```

### 25.2 Derived state

Artifact Store does not persist generic:

```text
Plugin membership
selector matches
relationship positions
parent ownership
reverse membership
composition cascade rules
```

A selector is stored only as declaration content in its containing Definition.

Selector matches are derived resolution output. A cache may retain normalized results for efficiency, but the cache must not become ownership or positional state.

### 25.3 Purge behavior

A source-backed Artifact cannot be ordinarily purged while its declaration remains available in its Source.

Purge must follow Source mutation, package removal, or authoritative reconciliation.

Removing a contained declaration from its source removes that declaration occurrence. This remains Source reconciliation, not composition-owned cascade deletion.

## 26. Artifact enablement and local state

### 26.1 Artifact enablement

Every Artifact has one local `Enabled` value.

Rules:

- New Artifacts default to `Enabled=true`.
- The value is retained through ordinary Source refresh.
- The value is retained through ordinary package hydration.
- A disabled Artifact remains readable.
- A disabled Artifact remains addressable by ArtifactRef.
- A disabled Artifact remains editable when otherwise authorized.
- Store list APIs remain exhaustive and expose the value.
- Consumer lists may explicitly request enabled-only filtering.

`Enabled` does not alter:

- Source discovery.
- Definition persistence.
- Graph resolution.
- Ambiguity.
- Resource verification.
- Managed publication.
- Package replacement.
- Package removal.
- Plugin editing.
- Purge eligibility.
- Protected topology authorization.
- MCP policy composition.

MCP has no additional portable or Store-level effective-enabled field.

`Source.Enabled` is separate. Disabling a Source can make its Artifacts missing or resources unavailable.

### 26.2 Local state ownership

| Concern                       | Owner                          |
| ----------------------------- | ------------------------------ |
| Workspace directory enablement | Directory and policy Sources  |
| Generic Artifact enablement   | Artifact record                |
| MCP installation input values | MCP local installation data    |
| MCP secret references         | MCP local installation data    |
| Secret values                 | Secret storage                 |
| OAuth tokens                  | MCP runtime and secret storage |
| Selected MCP profile          | MCP local installation data    |
| Additional local MCP policies | MCP local installation data    |
| Prompt budget                 | Runtime configuration          |
| Tool invocation arguments     | Runtime request                |
| Model invocation values       | Runtime request                |

An intentional protected topology replacement may replace ArtifactRefs. Preserving local state across that operation requires an explicit migration or overlay policy.

## 27. Managed Plugin authoring

### 27.1 Storage model

A managed Plugin is an ordinary source-backed Plugin declaration.

Management code may:

- Read and resolve Plugin members.
- Derive direct membership views.
- Maintain a domain-owned cache.
- Update a managed Plugin document.
- Publish and refresh the managed Source.

It must not make Artifact Store infer ownership or reverse membership.

### 27.2 Portable and managed domains

Portable Plugin declarations may contain mixed Artifact types.

Current user-facing managed surfaces are narrower:

- Skill Plugins accept Skill members.
- MCP Plugins accept MCP servers and MCP policies.
- Mixed Plugin frontend authoring is deferred.

This application restriction does not narrow the portable Plugin contract.

### 27.3 Baseline Plugins

Each supported user Root has:

- One application-provisioned Skill baseline Plugin.
- One application-provisioned MCP baseline Plugin.

Application Root lifecycle, not generic Artifact Store Root creation, provisions these Plugins.

Baseline Plugins have:

- Fixed logical names.
- Fixed managed package locations.
- User-visible selection entries.
- Editable membership.

A baseline Plugin cannot be:

- Renamed.
- Runtime-disabled through application APIs.
- Deleted.

A baseline Plugin is an authoring destination. It is not:

- Automatically active.
- A Workspace default.
- An Agent default.
- A runtime default.
- An implicit API fallback.

Every managed creation request must explicitly identify its destination Plugin.

### 27.4 Membership semantics

Adding an external member does not:

- Copy the target.
- Enable or disable the target.
- Configure the target.
- Install the target.
- Change target metadata.
- Change target policy.

Removing a member does not delete its target.

Deleting an independent target leaves membership declared and unavailable.

Restoring a matching target makes the relationship available without rewriting the Plugin.

One target may be referenced by several Plugins.

A contained declaration exists while it remains present in the containing Plugin document.

Repository discovery never silently attaches a target to an editable Plugin.

### 27.5 Publication and replacement

Managed Plugins, Skills, MCP servers, and MCP policies are published as independent source-backed packages.

Managed discovery is authoritative for these package types.

A named create operation may be idempotent only when:

- Expected package bytes are equivalent.
- Expected Source state is current.
- Expected Artifact state is equivalent.

A non-equivalent package requires:

- Explicit replacement.
- Explicit upsert authorization.
- Application repair.
- Built-in reconciliation.

Equivalent current state should skip unnecessary refresh.

Package removal must reconcile managed discovery and clean stale managed declaration locators.

### 27.6 Multi-package creation behavior

Current managed Skill, MCP server, and policy creation uses two source-side writes:

1. Add external membership to the selected Plugin.
2. Publish the independent target package.

If publication fails:

- Membership remains declared.
- The relationship is visible and unavailable.
- Retry must re-read the Plugin revision.
- Durable transaction or recovery coordination remains pending.

### 27.7 Supported operations

A user may:

- Create an empty compatible Plugin.
- Select a baseline Plugin.
- Select another editable compatible Plugin.
- Add an existing target.
- Add a member before its target exists.
- Add a locator for a specific occurrence.
- Remove a member without deleting the target.
- View unavailable or ambiguous members.
- Attach one target to multiple Plugins.
- View direct compatible Plugin memberships for a target.

### 27.8 Plugin deletion

A managed ordinary Plugin cannot be deleted while it has direct declared members.

Rules:

- Missing members still count.
- Ambiguous members still count.
- Selectors and contained members count.
- Only direct members are checked.
- Nested Plugin members are not recursively counted.
- Baseline Plugins cannot be deleted.
- Deleting an empty Plugin removes only that Plugin package.
- Deleting a Plugin never deletes independent targets.

Deletion flow:

```text
Detach every direct member
  -> re-read expected Plugin revision
  -> verify members remain empty
  -> remove only the Plugin package
```

## 28. Managed Skill and MCP workflows

### 28.1 Managed Skill

```text
Select editable Skill Plugin
  -> declare external Skill membership
  -> publish independent Skill package
  -> refresh managed Sources
  -> display Skill and direct Plugin memberships
```

There is no implicit baseline fallback.

Detaching the Skill leaves the Skill independently available.

Deleting the Skill leaves declared memberships unavailable.

The managed API does not expose standalone Skill creation that omits Plugin selection.

### 28.2 Managed MCP server or policy

```text
Select editable MCP Plugin
  -> declare external MCP or policy membership
  -> publish independent package
  -> configure installation-local values if needed
  -> connect only when selected by an MCP consumer
```

Plugin membership does not:

- Configure secrets.
- Configure OAuth.
- Configure client credentials.
- Select a connection profile.
- Connect the server.
- Interpret generic Artifact enablement.
- Change effective policy.
- Replace installation-local state.

## 29. Built-in content and protected Roots

### 29.1 Common declaration behavior

Built-in and user-owned content use the same:

- Semantic identities.
- Relationship forms.
- Locator semantics.
- Contained declaration semantics.
- Resolver behavior.

They differ only in:

- Edit authority.
- Distribution.
- Protected Root policy.
- Placement of mutable local overlays.

### 29.2 Built-in package composition

A built-in package may distribute independent targets and a Plugin together.

```text
software-dev/
  plugins/
    software-dev.yaml

  skills/
    code-review/
      SKILL.md

    refactoring-code/
      SKILL.md

  mcps/
    github.yaml
```

Physical co-distribution does not establish ownership.

A built-in Plugin may also contain complete MCP or policy declarations. Hydration must not rewrite those contained declarations into generated external packages.

### 29.3 Protected Root behavior

Ordinary Source and declaration mutation is rejected for the protected built-in Root.

Protected Artifact enablement may be mutable through the authorized local metadata path.

Mutable MCP installation data for protected servers is stored in an external local overlay.

Default named lookup may fall back to this Root, but only for explicitly declared relationships.

### 29.4 Package hydration

Hydration tracks:

- Package-scoped desired state.
- Package fingerprints.
- Expected physical bytes.
- Expected Artifact occurrences.

Startup reconciliation:

- Skips equivalent current packages.
- Repairs changed, incomplete, or corrupted known packages.
- Removes stale known declared packages.
- Leaves unrelated packages and Sources unchanged.

Current reconciliation does not sweep manually inserted untracked protected content.

Removing a built-in package purges only overlays and secrets associated with MCP Artifacts removed by that package.

A complete protected topology reset is reserved for:

- Incompatible topology migration.
- Root identity migration.
- Unrecoverable topology corruption.

ArtifactRefs remain stable for unchanged packages during ordinary hydration. Stability is not guaranteed across intentional topology replacement.

Built-in hydration does not, by itself, require MCP runtime connection invalidation in this design.

## 30. Implementation requirements

### 30.1 Store and Source

- Keep the generic Store independent of concrete declaration packages.
- Register concrete schemas and decoders at application composition.
- Preserve immutable Definitions.
- Preserve Source-backed Artifact occurrences.
- Enforce the purge guard.
- Keep generic composition out of Store persistence.
- Verify resources through Source generation and digest state.
- Keep Source refresh explicit.

### 30.2 Resolver

- Register every supported type resolver.
- Expose typed public resolver APIs.
- Keep generic traversal private.
- Preserve every relationship occurrence.
- Implement current and protected built-in lookup.
- Implement generic fallback providers.
- Use Tool and Model as initial mapped fallback consumers.
- Keep fallback out of located, contained, and selector paths.
- Resolve aliases to terminal Artifacts.
- Detect cycles and enforce limits.
- Reject positional identities.

### 30.3 Discovery

- Keep selector eligibility type-specific.
- Resolve selector bases inside the declaring Source.
- Apply declaration path filters before logical-name filters.
- Preserve zero-match selectors.
- Sort matches by stable source-backed identity.
- Update discovery only through explicit refresh.
- Refresh only affected Sources.
- Keep external locator support behind registered adapters.

### 30.4 Consumers

- Update consumers to typed resolver APIs.
- Support partial and strict completeness.
- Support mapped Tool and Model targets where explicitly allowed.
- Use direct typed MCP declaration fields.
- Use Workspace members as the selected capability roots.
- Use one shared Text source-materialization path where practical.
- Keep all runtime execution outside the resolver.

## 31. Current implementation status

### 31.1 Status interpretation

- `Available` means an implementation path exists for the stated scope.
- `Pending` means approved completion or verification remains.
- `Deferred` means intentionally outside current approved implementation scope.
- `Not supported` means no supported behavior currently exists.
- `Removed` means a previous behavior is intentionally no longer exposed.
- Status does not assert successful completion of all builds, tests, migrations, static analysis, or generated bindings.

### 31.2 Sources, persistence, and verification

| Capability                            | Status                                                       |
| ------------------------------------- | ------------------------------------------------------------ |
| Filesystem Sources                    | Available                                                    |
| Embedded Sources                      | Available                                                    |
| Managed Sources                       | Available                                                    |
| Source-backed Definition persistence  | Available                                                    |
| Source-backed Artifact persistence    | Available                                                    |
| Source refresh state                  | Available                                                    |
| Definition digest verification        | Available                                                    |
| Resource generation verification      | Available                                                    |
| Filesystem Source symlink containment | Decision: Not needed and hence not implemented               |
| Universal Artifact enabled metadata   | Available                                                    |
| User-managed Source provisioning      | Available through Workspace APIs                             |
| Explicit managed package replacement  | Available                                                    |
| Equivalent publication refresh skip   | Available                                                    |
| Authoritative managed discovery       | Available for managed Plugin, Skill, MCP, and policy Sources |
| Managed declaration locator cleanup   | Available                                                    |
| Source-backed Artifact purge guard    | Available                                                    |

### 31.3 Resolver and discovery

| Capability                                                           | Status    |
| -------------------------------------------------------------------- | --------- |
| Per-type resolver registry                                           | Available |
| Per-type public resolver APIs                                        | Available |
| Private shared traversal core                                        | Available |
| Root-scoped symbolic lookup                                          | Available |
| Protected built-in fallback                                          | Available |
| Explicit `scope: builtin`                                            | Available |
| Generic fallback-provider infrastructure                             | Available |
| Tool mapped fallback infrastructure                                  | Available |
| Model mapped fallback infrastructure                                 | Available |
| Application-configured Tool mapped fallback                          | Available |
| Application-configured Model mapped fallback                         | Available |
| Duplicate terminal identity detection                                | Available |
| Named external relationships                                         | Available |
| Located external relationships                                       | Available |
| Contained occurrence resolution                                      | Available |
| Member-level partial status                                          | Available |
| Source-selected alias-to-terminal resolution for alias-capable types | Available |
| Supported-alias cycle detection                                      | Available |
| Named-lookup same-terminal alias deduplication                       | Available |
| Selector terminal ArtifactRef projection and deduplication           | Available |
| Alias provenance in public resolver output                           | Pending   |
| Member selectors                                                     | Available |
| Zero-match selectors                                                 | Available |
| Type-specific selector eligibility                                   | Available |
| Explicit selector discovery refresh                                  | Available |
| Application wiring for Plugin, Agent, and Team refresh coordinators  | Pending   |
| Selector base directory verification                                 | Available |
| Portable selector-path matching on Windows                           | Available |
| Refresh-closure Source-count limit                                   | Pending   |
| Stable selector-match identity                                       | Available |
| Non-positional relationship identity                                 | Available |
| Exact duplicate relationship rejection                               | Available |
| Cycle-safe flattened capability plans                                | Available |
| Shared completeness helper                                           | Available |
| Nested Workspace rejection                                           | Available |
| Local path locator resolution                                        | Available |
| URL, Git, and package locator wire representation                    | Available |
| URL, Git, and package materialization adapters                       | Deferred  |

### 31.4 Physical formats

| Capability                          | Status    |
| ----------------------------------- | --------- |
| Canonical JSON                      | Available |
| Canonical YAML                      | Available |
| `AGENTS.md`                         | Available |
| `CLAUDE.md`                         | Available |
| `README.md`                         | Available |
| `llms.txt`                          | Available |
| Documentation Text discovery        | Available |
| `SKILL.md`                          | Available |
| Direct Skill directory registration | Available |
| Direct `SKILL.md` registration      | Available |
| `.mcp.json`                         | Available |
| `mcp.json`                          | Available |
| Canonical MCP declarations          | Available |
| Source-selected MCP declarations    | Available |
| `AGENT.md`                          | Available |
| `*.agent.md`                        | Available |
| Canonical Plugin declarations       | Available |
| Workspace manifests                 | Available |

### 31.5 Workspace and capability planning

| Capability                                    | Status    |
| --------------------------------------------- | --------- |
| Read-only Workspace resolution                | Available |
| Explicit `RefreshWorkspace`                   | Available |
| Source-targeted refresh                       | Available |
| Unrelated Source refresh avoidance            | Available |
| Workspace prompt planning                     | Available |
| Workspace Skill planning                      | Available |
| Workspace MCP planning                        | Available |
| Partial capability occurrence reporting       | Available |
| Strict `RequireComplete` selection            | Available |
| Consistent terminal ArtifactRef deduplication | Available |
| Generic composition inspection plan           | Available |
| Workspace `members` selection                 | Available |
| Workspace `declarations` and `roots` behavior | Removed   |

### 31.6 Managed Plugins and authoring

| Capability                                                 | Status        |
| ---------------------------------------------------------- | ------------- |
| Portable mixed Plugin declarations                         | Available     |
| Domain-specific managed Skill Plugins                      | Available     |
| Domain-specific managed MCP Plugins                        | Available     |
| Direct-member deletion guard                               | Available     |
| Application-provisioned Skill baseline Plugin              | Available     |
| Application-provisioned MCP baseline Plugin                | Available     |
| Baseline runtime-disablement and deletion protection       | Available     |
| Explicit Plugin selection for managed Skill creation       | Available     |
| Explicit Plugin selection for managed MCP creation         | Available     |
| Explicit Plugin selection for managed policy creation      | Available     |
| Attach existing target                                     | Available     |
| Detach without deleting target                             | Available     |
| Direct compatible membership views                         | Available     |
| Managed Plugin publication                                 | Available     |
| Managed Skill publication                                  | Available     |
| Managed MCP server publication                             | Available     |
| Managed MCP policy publication                             | Available     |
| Managed package removal                                    | Available     |
| Explicit replacement protection                            | Available     |
| Standalone managed Skill creation without Plugin selection | Removed       |
| Individual editing of contained Skills                     | Not supported |
| Mixed Plugin frontend authoring                            | Deferred      |

### 31.7 Built-in packages

| Capability                                      | Status                           |
| ----------------------------------------------- | -------------------------------- |
| Managed Skill packages                          | Available                        |
| Built-in Skill packages                         | Available                        |
| Built-in MCP packages                           | Available                        |
| Contained built-in MCP and policy declarations  | Available                        |
| Package-scoped hydration                        | Available                        |
| Known package byte verification and repair      | Available                        |
| Stale known package removal                     | Available                        |
| ArtifactRef stability during ordinary hydration | Available for unchanged packages |
| ArtifactRef stability across topology migration | Not guaranteed                   |

### 31.8 MCP platform and runtime

| Capability                                                  | Status    |
| ----------------------------------------------------------- | --------- |
| MCP policy declarations and resolution                      | Available |
| MCP installation-local data                                 | Available |
| Terminal installation state for source-selected MCP aliases | Available |
| MCP secret references                                       | Available |
| MCP runtime configuration                                   | Available |
| MCP connection management                                   | Available |
| MCP Tool discovery and invocation                           | Available |
| MCP resource support                                        | Available |
| MCP prompt support                                          | Available |
| MCP completion support                                      | Available |
| MCP policy evaluation                                       | Available |
| MCP approval handling                                       | Available |
| MCP OAuth and client credential flows                       | Available |
| MCP secret redaction                                        | Available |

### 31.9 Pending completion work

| Concern                               | Status  | Remaining work                                                                                                |
| ------------------------------------- | ------- | ------------------------------------------------------------------------------------------------------------- |
| Build and acceptance verification     | Pending | Run complete tests, static analysis, migration coverage, and binding generation                               |
| Managed creation durability           | Pending | Membership and independent package publication remain separate Source-side operations                         |
| Protected undeclared-content sweep    | Pending | Known packages are reconciled, but manually inserted untracked protected content is not automatically removed |
| Bulk resource verification efficiency | Pending | Add bounded verification-session reuse for bulk Skill and Workspace materialization                           |

### 31.10 Deferred or unsupported ecosystem capabilities

#### Source and locator capabilities

| Capability                                     | Status        |
| ---------------------------------------------- | ------------- |
| Arbitrary cross-Root Workspace imports         | Not supported |
| Automatic activation of all built-in Artifacts | Not supported |
| Git locator materialization                    | Deferred      |
| Git archive materialization                    | Deferred      |
| URL locator materialization                    | Deferred      |
| Package locator materialization                | Deferred      |
| Archive and zip Sources                        | Deferred      |
| External Plugin package distribution workflow  | Deferred      |

The portable locator contract remains forward-compatible with these capabilities.

#### Authoring and frontend

| Capability                                                               | Status   |
| ------------------------------------------------------------------------ | -------- |
| Mixed Plugin frontend authoring                                          | Deferred |
| Managed authoring for Agent, Team, Loop, Workflow, Tool, Model, and Text | Deferred |
| Standalone management UI for Agent, Team, Loop, and Workflow plans       | Deferred |

#### Execution runtimes

| Capability                        | Status                   |
| --------------------------------- | ------------------------ |
| Tool execution                    | Future Tool consumer     |
| Model execution                   | Future Model consumer    |
| Agent execution                   | Future Agent runtime     |
| Team execution                    | Future Team runtime      |
| Loop execution                    | Future execution runtime |
| Workflow scheduling and execution | Future Workflow runtime  |
