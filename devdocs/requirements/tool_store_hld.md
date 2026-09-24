# Tool Store HLD

Status: Target design. This is a breaking migration from the legacy Tool Store to an Artifact Store-backed Tool Store. It preserves the current built-in Go Tool and SDK Tool capabilities, Collection and Tool enablement, inference Tool-choice hydration, Composer selection, and Go Tool invocation behavior.

Normative foundations:

- [Portable Artifact Declaration Contracts HLD](./artifact_contracts_hld.md)
- [Artifact Store, Resolution, and Ecosystem HLD](./artifacts_hld.md)

This HLD defines Tool-specific catalog storage, Tool Collections, built-in Tool admission, mapped-target exposure, Go and SDK execution routing, inference and conversation integration, management behavior, and migration boundaries.

It does not redefine generic Artifact Store persistence, protected topology, portable Plugin syntax, generic source behavior, or generic Artifact enablement.

## Table of contents <!-- omit from toc -->

- [Purpose](#purpose)
- [Goals and scope](#goals-and-scope)
  - [Goals](#goals)
  - [In scope](#in-scope)
  - [Out of scope](#out-of-scope)
- [Requirements and invariants](#requirements-and-invariants)
  - [Built-in-only Tool admission](#built-in-only-tool-admission)
  - [Tool descriptor compatibility](#tool-descriptor-compatibility)
  - [Bundle to Collection mapping](#bundle-to-collection-mapping)
  - [Artifact Store-backed enablement](#artifact-store-backed-enablement)
  - [Mapped-only external Tool exposure](#mapped-only-external-tool-exposure)
  - [Tool relationship admission](#tool-relationship-admission)
- [Target model](#target-model)
  - [Tool Catalog records](#tool-catalog-records)
  - [Tool Collections](#tool-collections)
  - [Go Tool records](#go-tool-records)
  - [SDK Tool records](#sdk-tool-records)
  - [Tool identity](#tool-identity)
- [Required workflows](#required-workflows)
  - [Built-in Tool Catalog installation](#built-in-tool-catalog-installation)
  - [Browse and manage Collections and Tools](#browse-and-manage-collections-and-tools)
  - [Resolve a Tool for an Artifact consumer](#resolve-a-tool-for-an-artifact-consumer)
  - [Hydrate Tool choices for inference](#hydrate-tool-choices-for-inference)
  - [Execute a Go Tool](#execute-a-go-tool)
  - [Execute an SDK Tool](#execute-an-sdk-tool)
- [Architecture and ownership](#architecture-and-ownership)
  - [Responsibility boundaries](#responsibility-boundaries)
  - [Store, runtime, and aggregate separation](#store-runtime-and-aggregate-separation)
  - [Tool Aggregate responsibilities](#tool-aggregate-responsibilities)
  - [Artifact resolver integration](#artifact-resolver-integration)
- [Management API and UI](#management-api-and-ui)
  - [Management operations](#management-operations)
  - [Management UI](#management-ui)
  - [Composer Tool selection](#composer-tool-selection)
- [Inference and conversation integration](#inference-and-conversation-integration)
  - [Conversation Tool selection](#conversation-tool-selection)
  - [Per-turn persistence](#per-turn-persistence)
  - [Inference bridge](#inference-bridge)
  - [Go Tool calls from inference](#go-tool-calls-from-inference)
  - [SDK Tool calls from inference](#sdk-tool-calls-from-inference)
  - [MCP and Skill Tool coexistence](#mcp-and-skill-tool-coexistence)
- [Built-in content and Artifact Store installation](#built-in-content-and-artifact-store-installation)
  - [Built-in Tool content](#built-in-tool-content)
  - [Package behavior](#package-behavior)
  - [Hydration behavior](#hydration-behavior)
- [Breaking migration considerations](#breaking-migration-considerations)
  - [Migration boundary](#migration-boundary)
  - [Direct Bundle migration](#direct-bundle-migration)
  - [API and identity migration](#api-and-identity-migration)
- [Implementation mapping](#implementation-mapping)
- [Current implementation status](#current-implementation-status)
  - [Current behavior retained by the target](#current-behavior-retained-by-the-target)
  - [Required target work](#required-target-work)
  - [Intentionally unsupported target behavior](#intentionally-unsupported-target-behavior)
- [Difference from the current Tool Store](#difference-from-the-current-tool-store)

## Purpose

The Tool Store manages the application-provided Tool catalog.

It supports two built-in Tool implementation families:

| Tool implementation family | Definition owner                         | Execution owner                               |
| -------------------------- | ---------------------------------------- | --------------------------------------------- |
| Go Tool                    | `llmtools-go` built-in registry          | Local Tool Runtime through `llmtools-go`      |
| SDK Tool                   | Application-authored built-in descriptor | Provider inference SDK through `inference-go` |

The Tool Store does not permit user-authored Tool implementations.

The target model is:

```text
Embedded Go Tool catalog input
  -> llmtools-go built-in registry
  -> Tool Catalog records in Artifact Store

Embedded SDK Tool catalog input
  -> Tool Catalog records in Artifact Store

Embedded legacy Bundle content
  -> Tool Collection Plugin Artifacts in Artifact Store

Tool Collection or Tool selection
  -> Tool Aggregate
  -> mapped Tool target
  -> inference projection or Go Tool Runtime
```

Legacy Tool Bundles are renamed to Tool Collections.

A Tool Collection is an Artifact Store-backed Collection. A Tool is stored as an internal Tool Catalog record in Artifact Store, but is exposed to other domains only through a mapped Tool target.

This preserves the existing boundary:

```text
Agent, Skill, Workspace, Composer, inference
  -> mapped Tool target
  -> Tool Store and Tool Aggregate
  -> backing Tool Catalog record
```

No external consumer receives a raw Tool implementation descriptor, Go function ID, Tool Catalog Artifact reference, legacy Bundle ID, or legacy Tool reference as its executable capability.

## Goals and scope

### Goals

The Tool Store must:

- Preserve the current built-in Go Tool catalog.
- Preserve the current built-in SDK Tool catalog, including provider web-search Tools and their user argument schemas.
- Rename Tool Bundles to Tool Collections.
- Preserve one Collection for every current Bundle.
- Preserve the current Tool membership of every Bundle through its replacement Collection.
- Store Tool Catalog records and Tool Collections in Artifact Store-managed content.
- Use Artifact-level enablement for both Collections and Tools.
- Make Tools available to other Artifact consumers through mapped targets only.
- Keep Go Tool execution separate from Tool catalog storage.
- Route SDK Tools through inference provider execution rather than local Go invocation.
- Preserve Composer Tool selection, Tool configuration, inference hydration, auto-execution behavior, and Tool-call attribution.
- Avoid introducing Tool authoring, arbitrary SDK Tools, HTTP Tools, command Tools, or runtime-loaded Go plugins.

### In scope

This HLD defines:

- Built-in Go Tool Catalog records.
- Built-in SDK Tool Catalog records.
- Tool Collection representation and membership.
- Tool and Collection enablement.
- Tool mapped-target identity and validation.
- Tool fallback and target-projection behavior.
- Go Tool Runtime behavior.
- SDK Tool inference behavior.
- Tool management API and UI behavior.
- Composer and conversation Tool selection behavior.
- Breaking migration from the legacy Tool Store.

### Out of scope

This HLD does not define:

- User Tool authoring.
- User-created Tool Collections.
- Tool or Collection import and export.
- Tool or Collection deletion.
- Tool membership editing.
- Arbitrary Go function registration.
- Runtime Go plugin loading.
- Arbitrary SDK Tool registration.
- HTTP Tool definitions.
- Command Tool definitions.
- Tool implementation fetched from a URL, Git repository, package registry, or local user path.
- MCP server setup, MCP Tool discovery, MCP Tool execution, or MCP connection state.
- Provider credential management.
- Conversation migration from legacy persisted Tool choices.

Existing conversation migration is handled by the independent migration script and is not part of this HLD.

## Requirements and invariants

### Built-in-only Tool admission

The Tool Store admits only application-provided built-in Tools.

A Tool Catalog record must be one of:

- A Go Tool generated from an approved `llmtools-go` built-in registry descriptor.
- An SDK Tool declared by application-owned embedded Tool catalog content.

The Tool Store must reject:

- User-created Tool definitions.
- Tool definitions discovered from arbitrary user Sources.
- Tool definitions with unknown implementation types.
- Tool definitions with an unrecognized Go function ID.
- Tool definitions with an unrecognized SDK type.
- HTTP, command, package, URL, Git, or dynamically loaded implementations.

The supported implementation discriminator is:

```text
go
sdk
```

The legacy `http` implementation type is not supported by the target Tool Store.

### Tool descriptor compatibility

The target Tool Catalog must preserve the existing Tool descriptor behavior, including:

- Tool ID.
- Slug.
- Version.
- Display name.
- Description.
- Tags.
- User-callable flag.
- Model-callable flag.
- Auto-execution recommendation.
- LLM Tool semantic type.
- Argument schema.
- User argument schema.
- Go implementation function identity.
- SDK implementation type.
- Built-in provenance.
- Enabled state.

The Tool Catalog must continue to support current SDK Tool semantic types, including `webSearch`.

The Tool Catalog must not narrow SDK Tools to only function Tools. An SDK Tool's semantic Tool type remains part of its built-in descriptor and is used when creating the provider inference Tool choice.

### Bundle to Collection mapping

Every current legacy Bundle becomes exactly one Tool Collection.

The migration is direct:

```text
Legacy Bundle
  -> one Tool Collection

Legacy Bundle metadata
  -> Tool Collection metadata

Legacy Tool membership in Bundle
  -> Tool membership in Tool Collection
```

For example, a legacy filesystem Bundle becomes one filesystem Tool Collection containing the same filesystem Tools.

This is not inferred dynamically at runtime. It is application-owned built-in content.

Each Tool belongs to one routing Tool Collection in the current product, matching the current one-Tool-to-one-Bundle model.

The routing Collection is used for:

- Management grouping.
- Collection-level enablement.
- Tool list presentation.
- Tool availability validation.

It does not imply that a Collection owns the Tool implementation. The Tool implementation remains independently represented by its Tool Catalog record.

### Artifact Store-backed enablement

The target uses Artifact Store local enablement rather than a legacy Tool overlay database.

| State              | Backing record                     | Meaning                                                       |
| ------------------ | ---------------------------------- | ------------------------------------------------------------- |
| Collection enabled | Tool Collection Artifact `Enabled` | Enables or disables all Tools routed through that Collection. |
| Tool enabled       | Tool Catalog Artifact `Enabled`    | Enables or disables the exact Tool descriptor.                |

A Tool is available only when:

```text
Tool Catalog Artifact is available
  and Tool Catalog Artifact is enabled
  and routing Tool Collection is available
  and routing Tool Collection is enabled
```

Disabling a Collection does not change individual Tool enabled state.

Disabling a Tool does not change Collection enabled state.

Both disabled Collections and disabled Tools remain visible in Tool management views.

### Mapped-only external Tool exposure

Tool Catalog Artifacts are internal Tool Store backing records.

Other domains receive Tools only as mapped targets.

A mapped Tool target must contain:

- `type: tool`.
- The logical Tool name.
- `builtin: true`.
- A Tool Aggregate provider identity.
- A versioned opaque identifier.

The opaque identifier must bind the resolved Tool Catalog record identity and current definition revision or digest.

It may internally identify the backing Tool Catalog Artifact, but that Artifact reference is not a public Tool capability and is not passed directly to consumers.

Mapped targets must be revalidated before use. A mapped target is invalid when:

- Its backing Tool Catalog record no longer exists.
- Its Tool Catalog record is unavailable.
- Its Tool Catalog record is disabled.
- Its routing Collection is unavailable.
- Its routing Collection is disabled.
- Its definition digest no longer matches.
- Its implementation identity no longer matches the admitted Go or SDK descriptor.
- Its logical Tool name no longer matches.

### Tool relationship admission

The Tool Store accepts only named external Tool relationships for executable Tool usage.

Supported form:

```yaml
- type: tool
  name: readfile
  scope: builtin
```

The Tool Store does not accept these as executable Tool Store targets:

- Located Tool relationships.
- Contained Tool declarations.
- Tool selectors.
- Artifact-backed user Tool declarations.
- Tool definitions from a mutable user Root.
- Raw Go function references.
- Raw SDK implementation references.

The shared Artifact resolver may still understand generic portable Tool declaration syntax. Tool Store-owned consumers must route Tool relationship resolution through the Tool Aggregate so that only approved mapped targets are accepted.

## Target model

### Tool Catalog records

A Tool Catalog record is a Tool Store-owned source-backed Artifact Store record.

It represents one built-in Tool descriptor, not a caller-authorable Tool declaration.

Conceptually:

```text
Tool Catalog record
  -> Tool identity
  -> Tool descriptor
  -> Go or SDK implementation discriminator
  -> schemas and callable metadata
  -> routing Tool Collection identity
  -> Artifact availability and enablement
```

The Tool Catalog record preserves the current `Tool` model semantics while moving its persistence to Artifact Store-managed content.

The record is private to the Tool Store. It is not directly returned as an executable Tool target by Artifact resolver consumers.

### Tool Collections

A Tool Collection is a Tool Store view over a protected built-in `plugin` Artifact.

A conceptual Collection declaration is:

```yaml
type: plugin
name: filesystem-tools
displayName: Filesystem Tools
description: Built-in filesystem capabilities.

members:
  - type: tool
    name: readfile
    scope: builtin

  - type: tool
    name: searchfiles
    scope: builtin
```

The Collection declaration contains Tool membership only. It does not contain implementation bodies.

A Tool Collection is:

- Source-backed through protected Artifact Store managed content.
- Read-only as declaration content.
- Enableable and disableable through local Artifact state.
- Visible in management UI.
- Not a user authoring destination.
- Not automatically activated for a Workspace, Agent, Skill, or Composer.

### Go Tool records

A Go Tool record is generated or validated from the linked `llmtools-go` built-in registry.

The Tool Store must verify that:

- The catalog Tool ID matches the registered Tool descriptor.
- The catalog slug and version match the registered Tool descriptor.
- The argument schema matches the registered Tool descriptor.
- The Go function identifier is registered in the approved registry.
- The routing Tool Collection includes the Tool.
- The Tool is not duplicated under another admitted name.

The Go Tool record is not executable by itself. The Tool Aggregate resolves it to a runtime Tool key, and the Tool Runtime invokes the registered `llmtools-go` implementation.

### SDK Tool records

An SDK Tool record is application-authored built-in catalog content.

It contains the current SDK Tool information, including:

- SDK type.
- LLM Tool semantic type.
- Argument schema.
- User argument schema.
- User-callable and model-callable flags.
- Display metadata.
- Routing Tool Collection.
- Any provider-specific built-in metadata required by inference projection.

SDK Tool records include the current provider web-search Tool descriptors.

SDK Tool records are not executed by the local Tool Runtime. They are projected into `inference-go` Tool choices and executed by the configured provider SDK during completion.

### Tool identity

The Tool Store keeps the current Tool identity concepts:

```text
Tool ID
Tool slug
Tool version
```

The Artifact Store Artifact ID is an internal storage identity. It does not replace the Tool's logical identity.

The mapped target identity must bind:

```text
Tool Catalog Artifact identity
  + Tool descriptor digest or revision
  + Tool ID
  + Tool slug
  + Tool version
  + implementation discriminator
```

A Tool slug must identify one admitted active Tool descriptor.

If two active built-in Tool records have the same slug, Tool resolution is ambiguous and must fail. No Collection order, package order, or version ordering may select one implicitly.

## Required workflows

### Built-in Tool Catalog installation

At startup, Tool Store initialization performs:

```text
Load embedded Tool Collection content
  -> load application-authored SDK Tool definitions
  -> inspect approved llmtools-go built-in Tool registry
  -> generate or validate Go Tool definitions
  -> validate Tool descriptors and Collection membership
  -> publish Tool Catalog records to protected Artifact Store content
  -> publish Tool Collection Plugin Artifacts
  -> reconcile current Tool and Collection enablement state
  -> expose Tool Aggregate mapped-target resolution
```

The Tool installer must reject a package when:

- A Collection references a missing Tool Catalog record.
- A Tool Catalog record has no routing Collection.
- A Tool is routed through more than one current Collection.
- A Go Tool does not match the linked `llmtools-go` registry.
- An SDK Tool has an unsupported SDK type or unsupported semantic Tool type.
- A Collection contains a non-Tool member.
- A Collection contains a Tool locator, contained Tool declaration, selector, override, or use block.
- A Tool descriptor is structurally invalid.
- A duplicate Tool slug is admitted.

The Tool installer does not execute Go Tools or SDK Tools.

### Browse and manage Collections and Tools

The management workflow is:

```text
Open Tools
  -> list all Tool Collections, including disabled Collections
  -> expand a Collection
  -> list its Tool Catalog records, including disabled Tools
  -> inspect Go or SDK metadata and schemas
  -> enable or disable a Collection
  -> enable or disable a Tool
  -> reload the management projection
```

The reload action is a read refresh of the management projection. It does not author, republish, or mutate Tool package content.

### Resolve a Tool for an Artifact consumer

An Agent, Skill, Plugin, Team, Workspace, or Composer can reference a built-in Tool by name.

```text
Named Tool relationship
  -> Tool Aggregate target resolver
  -> locate approved Tool Catalog record
  -> validate Tool and Collection availability
  -> emit mapped Tool target
  -> consumer capability plan
```

The Tool Aggregate is responsible for ensuring a Tool Catalog Artifact is never returned as a raw executable Artifact target.

For supported application content, a named Tool relationship resolves only to an approved built-in mapped target.

### Hydrate Tool choices for inference

The inference workflow is:

```text
Conversation Tool selection
  -> mapped Tool target and selection settings
  -> Tool Inference Bridge
  -> resolve and validate Tool Catalog record
  -> validate enabled and model-callable state
  -> project inference-go ToolChoice
  -> append Skill and MCP Tool choices as applicable
  -> provider completion
```

Go Tool hydration:

```text
Mapped Go Tool target
  -> Tool Catalog descriptor
  -> function or custom Tool choice metadata
  -> Tool argument schema
  -> inference-go ToolChoice
```

SDK Tool hydration:

```text
Mapped SDK Tool target
  -> Tool Catalog descriptor
  -> validate user argument instance against UserArgSchema
  -> map SDK semantic Tool type
  -> provider-specific inference-go ToolChoice
  -> provider SDK execution during completion
```

The current SDK web-search behavior remains supported.

A disabled Tool or disabled Collection must not hydrate into an inference Tool choice.

### Execute a Go Tool

A local Go Tool invocation flow is:

```text
Mapped Go Tool target
  -> Tool Aggregate validates target and availability
  -> Tool Catalog resolves runtime Tool key
  -> Tool Runtime calls llmtools-go registry
  -> Tool outputs return to caller
```

The Tool Runtime only executes Go Tools.

The Tool Runtime must reject:

- SDK Tool targets.
- Disabled Tools.
- Tools from disabled Collections.
- Unrecognized runtime Tool keys.
- Raw function IDs supplied by callers.
- Legacy Bundle and Tool references.

SDK Tool execution never goes through the local Tool Runtime.

### Execute an SDK Tool

An SDK Tool is executed as part of provider inference.

```text
Mapped SDK Tool target
  -> Tool Inference Bridge
  -> provider-compatible inference ToolChoice
  -> provider SDK request
  -> provider Tool-call or Tool-output events
  -> conversation outputs
```

The Tool Store does not directly invoke SDK Tools.

## Architecture and ownership

### Responsibility boundaries

| Concern                                           | Owner                                           |
| ------------------------------------------------- | ----------------------------------------------- |
| Go Tool implementation and registry               | `llmtools-go`                                   |
| SDK Tool built-in descriptors                     | Application-owned embedded Tool catalog content |
| Tool Catalog record persistence                   | Artifact Store-backed Tool Store                |
| Tool Collection persistence                       | Artifact Store-backed Tool Store                |
| Collection and Tool enabled state                 | Universal Artifact `Enabled` state              |
| Tool target projection                            | Tool Aggregate                                  |
| Mapped target fallback integration                | Tool Aggregate                                  |
| Go Tool invocation                                | Tool Runtime                                    |
| SDK Tool projection and provider execution        | Tool Inference Bridge and provider inference    |
| Composer Tool selection                           | Composer and conversation UI                    |
| Tool call attribution and auto-execution decision | Conversation and inference consumer             |
| MCP Tool discovery and invocation                 | MCP Store and MCP Runtime                       |

### Store, runtime, and aggregate separation

The target dependency direction is:

```text
Tool Catalog domain types
  <- Tool Store
  <- Tool Runtime
  <- Tool Aggregate
  <- Tool Inference Bridge

Artifact Store
  <- Tool Store
  <- Tool Aggregate

Tool Aggregate
  <- Artifact resolver integration
  <- management facade
  <- Composer and inference consumers
```

The Tool Runtime must not depend on:

- Tool Store types.
- Artifact Store types.
- Tool Collection types.
- Artifact resolver types.
- Wails transport types.
- Legacy `ToolStore`, `ToolBundle`, or `ToolRef` types.

The Tool Store must not invoke the Tool Runtime.

The Tool Aggregate is the only component allowed to combine:

- Tool Catalog record resolution.
- Tool Collection resolution.
- Artifact enabled state.
- Mapped target encoding and decoding.
- Go Tool runtime dispatch.
- SDK Tool inference projection.

This replaces the current direct dependency:

```text
ToolRuntime
  -> ToolStore
```

with:

```text
ToolAggregate
  -> ToolStore
  -> Tool Runtime
  -> Tool Inference Bridge
```

### Tool Aggregate responsibilities

The Tool Aggregate provides four distinct capabilities:

| Capability             | Responsibility                                                                |
| ---------------------- | ----------------------------------------------------------------------------- |
| Tool target resolver   | Converts an approved Tool Catalog record into a mapped Tool target.           |
| Mapped target resolver | Converts a mapped Tool target back into a validated internal Tool descriptor. |
| Tool invocation router | Routes Go Tools to the Tool Runtime and rejects SDK Tool invocation.          |
| Tool inference bridge  | Converts Go and SDK Tool selections into `inference-go` Tool choices.         |

The Tool Aggregate must validate both the Tool and its routed Collection on every mapped target use.

It must not trust cached frontend metadata, legacy IDs, Tool slugs alone, or a caller-provided implementation reference.

### Artifact resolver integration

The existing Tool fallback provider becomes an Artifact Store-backed Tool Aggregate integration.

It must resolve named Tool relationships by consulting the Tool Store's backing Tool Catalog records.

It must return mapped targets, not raw Tool Catalog Artifacts.

This means the Tool resolver path is:

```text
Portable named Tool relationship
  -> Tool Aggregate
  -> Tool Catalog Artifact
  -> mapped target
```

The backing Artifact is internal storage. The mapped target is the external capability.

## Management API and UI

### Management operations

The target Tool Store management surface supports:

| Operation                      | Behavior                                                                            |
| ------------------------------ | ----------------------------------------------------------------------------------- |
| List Tool Collections          | Returns all built-in Tool Collections, including disabled Collections.              |
| Get Tool Collection            | Returns Collection metadata, enabled state, and Tool membership projection.         |
| List Collection Tools          | Returns current Tool descriptors, including disabled Tools.                         |
| Get Tool                       | Returns a read-only Tool descriptor and effective availability.                     |
| Set Collection enabled         | Updates the Tool Collection Artifact `Enabled` state.                               |
| Set Tool enabled               | Updates the Tool Catalog Artifact `Enabled` state.                                  |
| Resolve mapped Tool target     | Internal or aggregate-facing operation that validates and resolves a mapped target. |
| Hydrate inference Tool choices | Converts current Tool selections into `inference-go` Tool choices.                  |
| Invoke mapped Go Tool          | Validates and invokes an enabled mapped Go Tool.                                    |

The management surface does not support:

- Create Tool Collection.
- Delete Tool Collection.
- Edit Tool Collection declaration content.
- Add or remove Collection members.
- Create Tool.
- Delete Tool.
- Edit Tool implementation.
- Import Tool.
- Export Tool.
- Register arbitrary SDK Tools.
- Register arbitrary Go Tools.

### Management UI

The current Tools page becomes a Tool Collections page.

It must preserve the current interaction model:

- List all built-in Collections.
- Expand a Collection to inspect its Tools.
- Show Collection enabled state.
- Show Tool enabled state.
- Toggle Collection enabled state.
- Toggle Tool enabled state.
- Show display name, slug, version, description, tags, callable flags, and schemas.
- Show implementation family as `Go` or `Provider SDK`.
- Show whether a Tool is user-callable or model-callable.
- Show read-only messaging for built-in definitions.
- Surface Collection or Tool load errors without hiding unaffected Collections.

The UI must not add:

- Add Tool controls.
- Delete Tool controls.
- Add Collection controls.
- Delete Collection controls.
- Tool implementation editors.
- HTTP Tool editors.
- Arbitrary SDK Tool editors.
- MCP Tool management inside the Tool page.

### Composer Tool selection

Composer Tool selection must continue to support:

- Selecting enabled Tools.
- Deselecting selected Tools.
- Preserving a selection-specific `choiceID`.
- Configuring `autoExecute`.
- Rendering and validating `UserArgSchema` for SDK Tools.
- Persisting a selection-specific user argument instance.
- Displaying Tool availability and validation errors.

The Composer must identify a selected Tool by its mapped target, not by:

```text
bundleID
toolSlug
toolVersion
```

The UI may retain display metadata as a convenience snapshot, but all execution-relevant fields are reloaded from the Tool Aggregate.

## Inference and conversation integration

### Conversation Tool selection

A current conversation Tool selection should conceptually contain:

```text
ToolSelection
  -> choiceID
  -> mapped Tool target
  -> autoExecute
  -> userArgSchemaInstance
  -> optional display snapshot
```

The Tool semantic type, implementation type, schemas, callable flags, and SDK type are derived from the current validated Tool Catalog descriptor.

They must not be trusted from frontend-provided selection metadata.

### Per-turn persistence

The current behavior of storing both:

- The Tool Store selection used for the turn.
- The hydrated `inference-go` Tool choice used for the request.

remains valid.

The new selection record replaces legacy Bundle and Tool tuple addressing with a mapped target.

The hydrated inference Tool choice remains the request snapshot that was sent to the provider.

### Inference bridge

`ProviderSetAPI` currently depends directly on the legacy `ToolStore` through `buildToolChoices`.

The target `ProviderSetAPI` depends on a Tool Inference Bridge or Tool Aggregate port instead.

Conceptually:

```text
ProviderSetAPI
  -> ToolInferenceBridge.HydrateChoices
  -> ToolAggregate.ResolveMappedTarget
  -> Tool Store catalog and enabled-state validation
  -> Go or SDK inference ToolChoice projection
```

The Tool Inference Bridge must:

- Validate every selected mapped target.
- Reject Tools from disabled Collections.
- Reject disabled Tools.
- Reject non-model-callable Tools.
- Validate SDK user argument instances against the current Tool `UserArgSchema`.
- Preserve current SDK web-search configuration projection.
- Reject an SDK Tool when the selected provider or model does not support its declared SDK behavior.
- Preserve `choiceID` so provider Tool calls can be attributed back to the selected Tool.

### Go Tool calls from inference

When a model calls a Go Tool:

```text
Provider Tool call
  -> choiceID
  -> selected mapped Tool target
  -> Tool Aggregate
  -> Tool Runtime
  -> llmtools-go Tool output
  -> inference Tool output event
```

The `choiceID` maps the provider Tool call to the selected mapped target.

The Tool Runtime never receives a legacy Bundle ID, Tool slug, Tool version, or raw Go function ID from the inference layer.

### SDK Tool calls from inference

When a provider executes an SDK Tool:

```text
Provider completion request
  -> SDK ToolChoice
  -> provider SDK executes server Tool behavior
  -> provider response includes Tool-call or Tool-output events
  -> conversation persists normal inference output
```

No local Tool Runtime call occurs for SDK Tools.

This preserves the current distinction:

| Tool type | Local Tool Runtime | Provider inference SDK        |
| --------- | ------------------ | ----------------------------- |
| Go        | Yes                | Receives Tool definition only |
| SDK       | No                 | Yes                           |

### MCP and Skill Tool coexistence

The Tool Inference Bridge produces only Tool Store-managed Tool choices.

The existing completion pipeline continues to append:

- Skill runtime Tool choices.
- MCP Tool choices.
- MCP Tool mappings.
- MCP context inputs.

MCP Tools are not copied into Tool Collections and are not represented as Tool Catalog records.

## Built-in content and Artifact Store installation

### Built-in Tool content

The target built-in Tool content contains:

- Tool Collection definitions migrated from current Bundle definitions.
- Application-authored SDK Tool definitions.
- Go Tool catalog definitions derived from or validated against `llmtools-go`.
- Collection routing metadata.
- Tool schemas and user argument schemas.
- Tool display metadata.

The legacy standalone tools directory is removed.

The Artifact Store protected managed Source becomes the persistent backing location for Tool Catalog records and Tool Collection packages.

### Package behavior

The Tool installer publishes two conceptual package families:

```text
Tool Collection package
  -> one Plugin declaration
  -> named Tool membership

Tool Catalog package
  -> one Tool Catalog descriptor
  -> Go or SDK implementation metadata
```

The exact physical package filenames are an implementation concern. The package boundary must preserve stable source-backed Artifact identity for unchanged Tool and Collection content.

### Hydration behavior

Ordinary built-in hydration must:

- Create missing Tool Catalog records.
- Create missing Tool Collections.
- Repair changed known Tool packages.
- Repair changed known Collection packages.
- Remove stale known Tool packages.
- Remove stale known Collection packages.
- Preserve unchanged Tool and Collection Artifact references.
- Preserve enabled state for unchanged Tool and Collection Artifacts.
- Revalidate Go Tool descriptors against the linked `llmtools-go` registry.
- Revalidate SDK Tool descriptors against the supported built-in SDK catalog.

Hydration must not:

- Reset enabled state for unchanged Tools.
- Reset enabled state for unchanged Collections.
- Create user-authorable Tool content.
- Expose Tool Catalog Artifacts directly to generic consumers.
- Convert SDK Tools into Go Tools.
- Convert MCP Tools into Tool Catalog records.

## Breaking migration considerations

### Migration boundary

This is a breaking migration because the legacy Tool Store data model and transport identities are removed.

The following legacy storage is removed:

```text
tools_v1 directory
legacy embedded Tool Bundle manifests
legacy Tool JSON catalog files
legacy Tool enablement overlay database
legacy ToolStore-backed runtime lookup
```

The target Artifact Store contains the new Tool Catalog records and Tool Collections.

### Direct Bundle migration

Each current Bundle is migrated directly into a Tool Collection package.

Each current Tool is migrated into one Tool Catalog record:

- Current Go Tools become Go Tool Catalog records.
- Current SDK Tools become SDK Tool Catalog records.
- Existing Tool schemas are preserved.
- Existing SDK `UserArgSchema` values are preserved.
- Existing `LLMToolType` values are preserved.
- Existing Collection membership is preserved.

No legacy Bundle-to-Collection inference is required after packaging. The built-in Collection packages are the source of truth.

### API and identity migration

The following legacy identity form is retired:

```text
bundleID + toolSlug + toolVersion
```

The new executable identity is:

```text
mapped Tool target
```

The following legacy APIs are replaced:

| Legacy behavior                      | Target behavior                                             |
| ------------------------------------ | ----------------------------------------------------------- |
| List Tool Bundles                    | List Tool Collections                                       |
| Patch Tool Bundle                    | Set Tool Collection enabled                                 |
| List Tools by Bundle                 | List Tools by Collection                                    |
| Patch Tool                           | Set Tool enabled                                            |
| Get Tool by Bundle, slug, version    | Resolve Tool descriptor through Tool Store or mapped target |
| Invoke Tool by Bundle, slug, version | Invoke mapped Go Tool through Tool Aggregate                |
| Legacy Tool fallback target          | Artifact Store-backed mapped Tool target                    |
| Direct inference ToolStore lookup    | Tool Inference Bridge hydration                             |

Conversation migration is outside this HLD.

## Implementation mapping

| Current component                 | Target responsibility                                                                            |
| --------------------------------- | ------------------------------------------------------------------------------------------------ |
| `internal/tool/store`             | Artifact Store-backed Tool Store consumer API over Tool Catalog and Tool Collection Artifacts.   |
| `internal/tool/spec.ToolBundle`   | Replaced by Tool Collection Artifact projection.                                                 |
| `internal/tool/spec.Tool`         | Preserved semantically as a Tool Catalog descriptor, backed by Artifact Store source content.    |
| `tools.bundles.json`              | Replaced by embedded Tool Collection package definitions.                                        |
| Existing SDK Tool JSON files      | Migrated into embedded SDK Tool Catalog package definitions.                                     |
| `injectLLMToolsGo`                | Produces or validates Go Tool Catalog records from `llmtools-go`.                                |
| Legacy enablement overlay         | Replaced by Tool and Collection Artifact `Enabled` state.                                        |
| `artifactfallback.Service`        | Replaced by Tool Aggregate mapped-target resolver backed by Artifact Store Tool Catalog records. |
| `ToolRuntime`                     | Go-only runtime with no Tool Store dependency.                                                   |
| `ToolStoreWrapper`                | Management-facing Tool Store facade.                                                             |
| `ToolRuntimeWrapper`              | Aggregate-facing Go invocation facade.                                                           |
| `ProviderSetAPI.buildToolChoices` | Tool Inference Bridge hydration from mapped Tool selections.                                     |
| Tools management page             | Tool Collections management projection retaining current enable or disable controls.             |
| Composer Tool identity helpers    | Mapped-target identity helpers.                                                                  |

## Current implementation status

Status terms:

- `Current` means behavior exists in the supplied implementation.
- `Required` means target-design work is required.
- `Retired` means the legacy behavior is removed by this migration.
- `Not supported` means intentionally outside the target Tool Store.

### Current behavior retained by the target

| Capability                           | Status  | Target disposition                                              |
| ------------------------------------ | ------- | --------------------------------------------------------------- |
| Built-in Go Tools                    | Current | Retained through `llmtools-go`-validated Tool Catalog records.  |
| Built-in SDK Tools                   | Current | Retained through application-authored SDK Tool Catalog records. |
| SDK web-search Tool choices          | Current | Retained through Tool Inference Bridge projection.              |
| Tool argument schemas                | Current | Retained in Tool Catalog records.                               |
| SDK user argument schemas            | Current | Retained in Tool Catalog records and Composer configuration.    |
| Tool Bundle grouping                 | Current | Retained as renamed Tool Collections.                           |
| Bundle enable or disable             | Current | Retained as Tool Collection Artifact enablement.                |
| Tool enable or disable               | Current | Retained as Tool Catalog Artifact enablement.                   |
| Go Tool invocation                   | Current | Retained through a Store-independent Tool Runtime.              |
| Composer Tool selection              | Current | Retained with mapped-target identity.                           |
| Conversation Tool choice persistence | Current | Retained with mapped-target selection records.                  |
| MCP Tool coexistence                 | Current | Retained as a separate MCP subsystem.                           |

### Required target work

| Capability                     | Status   | Required outcome                                                                                            |
| ------------------------------ | -------- | ----------------------------------------------------------------------------------------------------------- |
| Tool Catalog Artifact schema   | Required | Define and register a Tool Store-private source-backed Tool descriptor schema.                              |
| SDK Tool package migration     | Required | Move current SDK Tool descriptors and schemas into protected Artifact Store packages.                       |
| Go Tool package generation     | Required | Generate or validate Go Tool Catalog records from `llmtools-go`.                                            |
| Tool Collection packages       | Required | Convert every current Bundle into one protected Plugin-based Tool Collection.                               |
| Tool installer                 | Required | Hydrate Tool Catalog and Tool Collection package families into Artifact Store.                              |
| Tool Catalog validation        | Required | Enforce Go registry matching, SDK admission, unique slugs, and one Collection route per Tool.               |
| Tool mapped-target aggregate   | Required | Resolve backing Tool Catalog records into validated mapped targets.                                         |
| Resolver integration           | Required | Ensure Tool Store consumers receive mapped targets rather than raw Tool Catalog Artifacts.                  |
| Store-independent Go runtime   | Required | Remove direct dependency on legacy `ToolStore`.                                                             |
| Tool Inference Bridge          | Required | Replace direct legacy Store hydration in `ProviderSetAPI`.                                                  |
| Composer selection migration   | Required | Use mapped targets and current Tool descriptors rather than Bundle tuple identity.                          |
| Management UI migration        | Required | Replace Bundle terminology and legacy APIs with Tool Collection projections.                                |
| Legacy tools directory removal | Required | Remove standalone Tool Store content and overlay persistence after Artifact Store-backed content is active. |

### Intentionally unsupported target behavior

| Capability                                              | Status        |
| ------------------------------------------------------- | ------------- |
| User-created Tool Collection                            | Not supported |
| Tool Collection deletion                                | Not supported |
| Tool Collection membership editing                      | Not supported |
| User-created Go Tool                                    | Not supported |
| User-created SDK Tool                                   | Not supported |
| Arbitrary SDK Tool registration                         | Not supported |
| HTTP Tool definition                                    | Not supported |
| Command Tool definition                                 | Not supported |
| Runtime Go plugin loading                               | Not supported |
| Arbitrary Tool source registration                      | Not supported |
| MCP Tool duplication into Tool Store                    | Not supported |
| Raw Tool Catalog Artifact as external executable target | Not supported |

## Difference from the current Tool Store

The target intentionally preserves current product behavior:

- The same built-in Go Tools remain available.
- The same built-in SDK Tools remain available.
- Current SDK web-search behavior remains available.
- Current Tool argument and user argument schemas remain available.
- Current Bundles become Collections with the same membership.
- Collections and individual Tools can still be enabled or disabled independently.
- Go Tools are still invoked locally.
- SDK Tools are still executed by provider SDK inference.
- Composer Tool selection, user argument configuration, auto-execution handling, Tool-call attribution, and conversation Tool choice persistence remain supported.
- MCP Tools remain separate and continue to coexist with Tool Store Tool choices.

The primary differences are architectural and storage-related:

- Bundles are renamed to Tool Collections.
- Tool definitions and Collections are persisted and hydrated through Artifact Store instead of the legacy `tools_v1` directory and Tool overlay store.
- Tool and Collection enabled state uses Artifact Store `Enabled` metadata instead of a Tool-specific overlay database.
- Tool Catalog records are internal Artifact Store backing records.
- Other domains receive Tools only as mapped targets.
- Go Tool Runtime no longer depends on Tool Store types.
- SDK Tool projection is owned by the Tool Inference Bridge rather than direct legacy Tool Store lookup.
