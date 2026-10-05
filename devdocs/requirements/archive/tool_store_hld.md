# Tool Store HLD

Status: Current design and implementation direction. The Tool Store is Artifact Store-backed, uses the adapted `toolv1` contract, supports built-in Go and SDK Tools, exposes Tool capabilities as mapped targets to Artifact consumers, and preserves Composer, inference, conversation, and management behavior.

Normative foundations:

- [Portable Artifact Declaration Contracts HLD](./artifact_contracts_hld.md)
- [Artifact Store, Resolution, and Ecosystem HLD](./artifacts_hld.md)

This HLD defines Tool-specific behavior. It does not redefine generic Artifact Store persistence, Source behavior, protected topology, Plugin semantics, generic Collection behavior, resolver traversal, or Artifact local enablement.

## Purpose

The Tool Store manages the application-provided Tool catalog.

It supports two Tool implementation families:

| Implementation family | Definition source                                                 | Execution path         |
| --------------------- | ----------------------------------------------------------------- | ---------------------- |
| Go Tool               | `llmtools-go` built-in registry, materialized into Tool Artifacts | Local Tool Runtime     |
| SDK Tool              | Application-authored embedded Tool document                       | Provider inference SDK |

The Tool Store is Artifact Store-backed:

```text
Built-in Tool package
  -> Tool Artifact

Built-in Tool Collection package
  -> Plugin Artifact

Named Tool relationship
  -> normal Artifact resolution
  -> Tool Artifact target mapper
  -> mapped Tool target

Mapped Tool target
  -> Tool Aggregate
  -> Go Tool Runtime or provider inference projection
```

Legacy Tool Bundles are renamed to Tool Collections.

The migration preserves the current product model:

```text
Legacy Bundle
  -> Tool Collection

Legacy Go Tool
  -> generated Tool Artifact

Legacy SDK Tool
  -> embedded Tool Artifact
```

The Tool Store does not support user Tool authoring, user Tool Collections, arbitrary SDK Tool registration, HTTP Tools, command Tools, or runtime-loaded Go plugins.

## Goals and scope

### Goals

The Tool Store must:

- Preserve built-in Go Tool behavior.
- Preserve built-in SDK Tool behavior.
- Preserve provider SDK web-search Tool behavior.
- Preserve Tool argument schemas and SDK user argument schemas.
- Replace legacy Bundles with Tool Collections.
- Preserve current Bundle membership through direct built-in Collection definitions.
- Preserve independent Tool and Collection enablement.
- Store Tool definitions and Collections in Artifact Store managed content.
- Expose Tool capabilities to Artifact consumers as mapped targets.
- Keep Tool catalog storage separate from Go Tool execution.
- Keep SDK Tool execution in provider inference.
- Support Composer Tool selection, Tool options, inference hydration, Tool calls, Tool outputs, retries, and auto-execution.
- Keep MCP Tool behavior separate from Tool Store behavior.
- Continue using `toolv1` while the feature is in development.

### In scope

This HLD defines:

- Built-in `toolv1` Tool Artifacts.
- Read-only Tool Collections represented by Plugin Artifacts.
- Go Tool generation and validation from `llmtools-go`.
- Static SDK Tool package admission.
- Tool and Collection enablement.
- Tool Artifact to mapped-target projection.
- Go Tool invocation through the Tool Aggregate.
- Low-level runtime Tool invocation.
- SDK Tool inference projection.
- Composer Tool selection and option handling.
- Conversation Tool selection persistence.
- Tool management APIs and UI.
- Built-in Tool package hydration.
- Breaking replacement of the legacy Tool Store.

### Out of scope

This HLD does not define:

- User Tool authoring.
- User Tool import, export, replacement, editing, or deletion.
- User-created Tool Collections.
- Tool Collection editing, deletion, or membership mutation.
- Arbitrary Go Tool registration.
- Runtime Go plugin loading.
- Arbitrary SDK Tool registration.
- HTTP Tool declarations.
- Command Tool declarations.
- MCP server setup, connection management, or capability discovery.
- MCP policy and approval semantics.
- Provider credential management.
- Conversation migration from legacy Tool choices.

Conversation migration is owned by the separate migration script and is not part of this HLD.

## Tool model

### Tool Artifact

A Tool is a source-backed Artifact with:

```text
type: tool
schema: toolv1
root: protected built-in Root
source: built-in managed package Source
```

A Tool Artifact contains:

- Logical Tool name.
- Tool version.
- Display name.
- Description.
- Tags.
- Auto-execution default.
- Input schema.
- Optional user argument schema.
- Optional output schema.
- Go or SDK implementation declaration.

The Tool Artifact is the authoritative backing record for:

- Tool source provenance.
- Tool definition digest.
- Tool Artifact revision.
- Tool enabled state.
- Tool package identity.
- Tool Collection membership validation.
- Mapped-target validation.

### Tool contract version

The current feature uses the adapted `toolv1` contract in place. The active Tool implementation kinds are:

```text
go
sdk
```

The current Tool contract intentionally does not contain:

```text
userCallable
llmCallable
http implementation
command implementation
locator
```

A selected Tool is advertised to inference according to its implementation and current selection context.

The Composer determines whether a Tool call is locally runnable:

| Tool implementation | Local Composer execution | Provider inference execution            |
| ------------------- | ------------------------ | --------------------------------------- |
| Go                  | Yes                      | Advertised as a function Tool choice    |
| SDK                 | No                       | Provider SDK executes the Tool behavior |

### Go Tool declaration

A Go Tool declaration identifies one application-linked `llmtools-go` registration:

```yaml
type: tool
name: readfile
version: v1
displayName: Read file
autoExecute: true

inputSchema:
  type: object

implementation:
  kind: go
  function: github.com/flexigpt/llmtools-go/fstool/readfile.ReadFile
```

The Tool Store validates a Go Tool against the linked `llmtools-go` registry.

### SDK Tool declaration

An SDK Tool declaration identifies an application-authored provider capability:

```yaml
type: tool
name: openaiwebsearch
version: v1.0.0
displayName: OpenAI Responses SDK Web Search
autoExecute: false

inputSchema:
  type: object
  additionalProperties: false

userArgSchema:
  type: object
  properties:
    searchContextSize:
      type: string

implementation:
  kind: sdk
  sdkType: providerSDKTypeOpenAIResponses
  sdkToolType: webSearch
```

The current SDK Tool semantic types are:

```text
function
custom
webSearch
```

The current embedded SDK Tool catalog uses provider web-search Tools.

### Tool Collection

A Tool Collection is a read-only Tool Store view over a protected `plugin` Artifact.

Example:

```yaml
type: plugin
name: fs
displayName: File-system utilities
description: Common file-system helpers.

members:
  - type: tool
    name: readfile
    scope: builtin

  - type: tool
    name: searchfiles
    scope: builtin
```

A Tool Collection must:

- Be a concrete Plugin declaration.
- Not be a locator alias.
- Contain only `type: tool` members.
- Use named external Tool members.
- Use `scope: builtin`.
- Not use Tool locators.
- Not use contained Tool declarations.
- Not use Tool selectors.
- Not use Tool overrides.
- Not use Tool `use` behavior.
- Be read-only.
- Be built-in package content.

Each admitted Tool belongs to exactly one Tool Collection in the current product.

This preserves the existing Bundle model:

```text
One Bundle
  -> one Tool Collection

One Tool
  -> one Tool Collection
```

The Collection groups and gates Tool availability. It does not own or execute the Tool implementation.

## Availability and enablement

A Tool is available for mapped-target use only when:

```text
Tool Artifact is available
  and Tool Artifact is enabled
  and Tool Collection Artifact is available
  and Tool Collection Artifact is enabled
```

Tool enablement uses the Tool Artifact's universal `Enabled` field.

Collection enablement uses the Tool Collection Plugin Artifact's universal `Enabled` field.

Disabling a Tool Collection:

- Makes all routed Tools unavailable for mapped-target resolution.
- Does not mutate individual Tool enabled state.
- Does not delete Tool Artifacts.
- Does not change Tool definitions.
- Does not prevent Tool management inspection.

Disabling a Tool:

- Makes that Tool unavailable for mapped-target resolution.
- Does not mutate its Tool Collection enabled state.
- Does not delete the Tool Artifact.
- Does not change its definition.

Management views include disabled Tools and disabled Collections.

## Mapped Tool targets

### External capability form

Artifact consumers receive a Tool capability as:

```text
resolve.MappedTarget
```

The current Tool mapped-target provider is:

```text
flexigpt.tool.aggregate.v1
```

The mapped target payload contains:

```text
ToolArtifact
DefinitionDigest
Name
Version
Implementation
```

The target is versioned through the mapped identifier format:

```text
v1.<base64url-canonical-json>
```

The payload is an encoded protocol identity. It is not intended to be secret or opaque for security purposes.

### Why the mapped target contains Artifact identity

The Tool Artifact reference and definition digest allow the Tool Aggregate to verify that the selected Tool still:

- Exists.
- Belongs to the protected built-in Tool Store.
- Has not changed definition.
- Has the same logical name.
- Has the same version.
- Has the same implementation kind.
- Is enabled.
- Belongs to an enabled Tool Collection.

This makes a stale mapped target unavailable rather than silently invoking a changed Tool definition.

### Artifact target mapper

Tool mapping uses `resolve.ArtifactTargetMapper`.

The flow is:

```text
Named Tool relationship
  -> normal Artifact lookup
  -> terminal Tool Artifact
  -> Tool Aggregate MapArtifactTarget
  -> ResolveEnabledTool
  -> mapped Tool target
```

This differs from a fallback-only Tool design.

Tool Artifacts exist in the protected built-in Root, so normal Artifact lookup succeeds. The target mapper converts that resolved Artifact into the mapped Tool target before a consumer receives it.

### Target mapper requirements

The Tool target mapper must:

- Handle only `type: tool`.
- Require a protected built-in Tool Artifact.
- Require the configured built-in Tool Source.
- Require an available Tool Artifact.
- Require an enabled Tool Artifact.
- Resolve exactly one Tool Collection for the Tool.
- Require an available Tool Collection.
- Require an enabled Tool Collection.
- Verify Tool definition digest.
- Verify Tool name.
- Verify Tool version.
- Verify Tool implementation kind.
- Return a mapped Tool target when handled.
- Return unavailable status when the Tool or Collection is disabled or unavailable.

The Tool target mapper is the minimum requirement for any consumer that resolves Tool relationships.

### Resolver composition rule

Every consumer-facing resolver that can resolve Tool relationships must receive the Tool target mapper.

This includes resolver construction used by:

- Agent Store.
- Skill Store.
- MCP Store where Tool relationships can occur.
- Workspace Store.
- Any future Artifact consumer that resolves `type: tool`.

Tool Collection management does not need a graph resolver because it validates direct Collection membership through the Tool Store.

### Capability plan behavior

For normal Tool relationships, capability plans expose:

```text
mapped Tool target
```

The backing Tool Artifact is storage and management identity.

Management APIs may expose backing Artifact references because they need:

- Tool enablement.
- Collection enablement.
- Revision-based updates.
- Inspection.
- Refresh and retry behavior.

Mapped selector behavior is acceptable in the current scope. Tool Collections themselves do not allow selectors.

## Built-in Tool installation

### Package families

The Tool installer publishes two built-in package families:

```text
tool/<tool-name>/<tool-version>
  -> tool declaration

tool-collection/<collection-name>/unversioned
  -> plugin declaration
```

A Tool package contains exactly its Tool declaration document.

A Tool Collection package contains exactly its Plugin declaration document.

### Go Tool package preparation

For every Tool Collection member not backed by a static SDK Tool document:

```text
Tool Collection member name
  -> llmtools-go registry lookup by name
  -> Go Tool descriptor
  -> generated toolv1 Tool document
  -> Tool package
```

The generated Go Tool document includes:

- Name.
- Version.
- Display metadata.
- Tags.
- Auto-execution default.
- Input schema.
- Go function identifier.

The Tool Store validates the hydrated Tool Artifact against the registry before using it.

### SDK Tool package preparation

SDK Tool documents are embedded under their Tool Collection directory.

For every static SDK Tool:

- The Tool directory must contain only its Tool document.
- The Tool must be declared by the containing Tool Collection.
- The Tool implementation kind must be `sdk`.
- The Tool name must be unique across all Tool Collections.
- The Tool must be source-backed by the protected built-in Tool Source.

### Go Tool admission

A generated or hydrated Go Tool must match the `llmtools-go` descriptor for:

```text
name
version
function
input schema
```

A Go Tool that does not match the linked registry is unavailable.

A registry Tool not named by an embedded Tool Collection is not admitted to the application Tool catalog.

### Auto-execution policy

Go Tool auto-execution policy is application-controlled.

The `llmtools-go` adapter defines the policy for registered Go functions.

Current behavior is:

```text
Known mutating or dangerous functions
  -> autoExecute: false

Other admitted Go functions
  -> autoExecute: true
```

This is intentional because the application controls both:

- The embedded Tool Collection membership.
- The `llmtools-go` registry version.
- The non-auto-executable function list.

Auto-execution remains a Tool default and a Composer selection behavior. It is not general authorization.

### Hydration behavior

The Tool installer participates in protected built-in package hydration.

It must:

- Publish missing Tool packages.
- Publish missing Tool Collection packages.
- Replace changed known Tool packages.
- Replace changed known Tool Collection packages.
- Remove stale known Tool packages.
- Remove stale known Tool Collection packages.
- Preserve unchanged Tool Artifact references.
- Preserve unchanged Collection Artifact references.
- Preserve Tool enabled state for unchanged Tool Artifacts.
- Preserve Collection enabled state for unchanged Tool Collection Artifacts.
- Revalidate Go Tool registry compatibility during Tool reads and mapped-target resolution.

The legacy standalone Tool directory and Tool overlay store are not used by the target implementation.

## Tool Store API

### Management API

The Tool Store management API supports:

| Operation                      | Behavior                                                                                 |
| ------------------------------ | ---------------------------------------------------------------------------------------- |
| List Tool Collections          | Lists built-in read-only Tool Collections.                                               |
| Get Tool Collection            | Returns Collection metadata, members, revision, and enabled state.                       |
| List Collection Tools          | Resolves Tool members into Tool views.                                                   |
| Get Tool                       | Returns a Tool view from a protected built-in Tool Artifact.                             |
| Set Tool Collection enabled    | Updates Collection Artifact `Enabled` state.                                             |
| Set Tool enabled               | Updates Tool Artifact `Enabled` state.                                                   |
| Resolve enabled Tool           | Resolves a Tool together with its routed Collection and verifies effective availability. |
| Install built-in Tool package  | Privileged installer operation.                                                          |
| Remove built-in Tool package   | Privileged installer operation.                                                          |
| Ensure built-in Source current | Privileged installer operation.                                                          |

The Tool Store management API does not support:

- Tool creation.
- Tool deletion.
- Tool editing.
- Tool import.
- Tool export.
- Tool Collection creation.
- Tool Collection deletion.
- Tool Collection metadata editing.
- Tool Collection membership editing.

### Tool view

A Tool management view contains:

- Tool Artifact metadata.
- Definition digest.
- Name.
- Version.
- Display name.
- Description.
- Tags.
- Auto-execution default.
- Input schema.
- User argument schema.
- Output schema.
- Implementation kind.
- SDK type and SDK Tool type when applicable.

The current Tool view may expose raw Go function identity. This is acceptable for the current application scope and developer inspection.

The function identity is not the normal execution capability identity for Tool Store consumers. Mapped targets remain the Tool Store capability form.

## Tool Aggregate

### Responsibilities

The Tool Aggregate owns the connection between:

- Tool Store.
- Tool Artifact target mapper.
- Mapped Tool target encoding.
- Mapped Tool target decoding.
- Go Tool Runtime.
- Inference Tool choice hydration.

It provides:

| Operation                      | Behavior                                                      |
| ------------------------------ | ------------------------------------------------------------- |
| Map Tool target                | Maps an enabled Tool Artifact to a mapped target.             |
| Map Artifact target            | Implements `resolve.ArtifactTargetMapper` for Tool Artifacts. |
| Resolve mapped Tool            | Decodes and revalidates a mapped Tool target.                 |
| Invoke mapped Tool             | Routes a mapped Go Tool to local runtime.                     |
| Hydrate inference Tool choice  | Converts a Tool selection into an inference Tool choice.      |
| Hydrate inference Tool choices | Converts a selection list into inference Tool choices.        |

### Mapped Tool resolution

Mapped Tool resolution performs:

```text
Decode mapped target
  -> load protected Tool Artifact
  -> verify Tool definition digest
  -> verify Tool name
  -> verify Tool version
  -> verify implementation kind
  -> verify Tool enabled state
  -> resolve Tool Collection
  -> verify Collection enabled state
  -> return resolved Tool view
```

A Tool target does not remain valid simply because its encoded identifier is syntactically valid.

### Mapped Go Tool invocation

Mapped Go Tool invocation performs:

```text
Mapped target
  -> Tool Aggregate ResolveMappedTool
  -> verify implementation kind is go
  -> resolve internal function identifier
  -> Tool Runtime Invoke
```

Mapped SDK Tool invocation is not routed to local Tool Runtime.

SDK Tools execute through provider inference after Tool choice hydration.

## Tool Runtime

### Low-level runtime role

The Tool Runtime is a low-level execution service.

It accepts:

```text
function
arguments
timeout
```

It validates:

- Runtime service availability.
- Context.
- Function text.
- JSON argument syntax.
- Timeout range.
- Registry membership through the configured Go Tool caller.

The Tool Runtime deliberately does not depend on:

- Tool Store.
- Tool Collection.
- Artifact Store.
- Tool Artifact state.
- Mapped targets.
- SDK Tool descriptors.
- Composer state.
- Conversation state.

### Raw runtime invocation

The raw runtime wrapper is intentionally available.

It is useful for application-level direct execution of registered `llmtools-go` functions.

It is not a Tool Store capability-resolution API.

Consequently:

```text
Tool Aggregate invocation
  -> respects Tool Artifact and Collection enablement

Raw runtime invocation
  -> executes an admitted registry function directly
```

The raw runtime path is intentionally separate from Tool Store enablement behavior.

### MCP runtime behavior

MCP Tool invocation is also a runtime concern.

MCP execution may be exposed through runtime-facing application APIs, but it remains governed by MCP-specific behavior:

- Server identity.
- Connection readiness.
- Tool discovery snapshot.
- Tool digest.
- Approval policy.
- Execution mode.
- Invocation arguments.
- MCP response content.

MCP Tools are not Tool Store Tool Artifacts and are not Tool Collection members.

The Tool Runtime and MCP Runtime may remain separate internal services while sharing an application-level runtime-facing transport layer.

## SDK Tool inference behavior

### SDK execution boundary

SDK Tools are not locally executed by Tool Runtime.

They are projected into provider inference choices:

```text
ToolSelection
  -> Tool Aggregate ResolveMappedTool
  -> SDK Tool descriptor
  -> inference ToolChoice
  -> provider SDK request
  -> provider Tool-call or Tool-output events
```

### SDK Tool choice hydration

For Go Tools, inference hydration produces:

```text
type: function
name: Tool name
description: Tool description
arguments: Tool input schema
```

For SDK Tools, inference hydration uses the Tool's declared SDK semantic type.

Current behavior:

| SDK Tool type | Inference projection                                             |
| ------------- | ---------------------------------------------------------------- |
| `webSearch`   | `ToolTypeWebSearch` with `WebSearchToolChoiceItem` configuration |
| `function`    | Function-style Tool choice with input schema                     |
| `custom`      | Custom-style Tool choice with input schema                       |

The Tool Aggregate performs the implementation-specific projection. The inference wrapper owns final provider request composition and global Tool choice uniqueness.

### User argument configuration

SDK Tool user argument configuration is selection-specific.

The selection stores:

```text
choiceID
target
autoExecute
userArgSchemaInstance
```

Current behavior is intentionally lightweight:

- Frontend validates JSON shape and required top-level fields for user experience.
- Web-search configuration is decoded into the inference web-search configuration structure.
- SDK `function` and `custom` configurations are accepted structurally when present.
- The system does not add a general backend JSON Schema validation layer.
- The system does not add a separate SDK vendor registry or validation surface.

This is intentional for the current built-in-only scope.

### Provider compatibility

The frontend filters SDK Tools by the current provider SDK type.

This is the current product behavior.

The Tool Store remains provider-neutral. It stores SDK type metadata and Tool semantic type metadata, while inference and provider integration decide whether a projected Tool choice is accepted by the selected provider.

## Composer and conversation integration

### Tool selection identity

A Tool selection uses:

```text
ToolSelection
  -> choiceID
  -> mapped target
  -> autoExecute
  -> userArgSchemaInstance
```

The mapped target replaces the legacy identity:

```text
bundleID
toolSlug
toolVersion
```

Frontend-enriched Tool choices may additionally contain:

- Tool display name.
- Description.
- Tool version.
- Collection reference.
- Collection name.
- Implementation kind.
- SDK type.
- SDK Tool semantic type.

That metadata is for display and Composer behavior. The mapped target remains the durable Tool selection identity.

### Conversation persistence

Conversation messages persist:

```text
ToolSelections
ToolChoices
```

`ToolSelections` are the current Tool Store selection records.

`ToolChoices` are hydrated inference request snapshots.

For a new completion:

```text
CompletionRequest.ToolSelections
  -> Tool Aggregate hydration
  -> current inference ToolChoices
```

The inference wrapper does not infer current Tool choices from historical `ToolChoices`.

### Composer Tool categories

The Composer supports:

| Category                    | Behavior                                                             |
| --------------------------- | -------------------------------------------------------------------- |
| Per-message Go Tool         | Attached for one message and may support auto-execution.             |
| Conversation Tool           | Remains available across subsequent turns until disabled or removed. |
| SDK web-search Tool         | Managed as a provider-specific web-search template.                  |
| SDK function or custom Tool | Projected according to SDK Tool type and current provider behavior.  |
| Skill Tool                  | Managed by Skill Runtime.                                            |
| MCP Tool                    | Managed by MCP Runtime and MCP policy.                               |

### Composer Tool hydration

The Composer hydrates persisted Tool selections through the Tool Aggregate.

```text
Mapped target
  -> ResolveMappedTool
  -> Tool view
  -> frontend-enriched Tool choice
```

If a target is unavailable:

- The original selection remains inspectable.
- The UI reports a Tool selection issue.
- The Tool cannot be used for a new completion until it is resolvable again or removed.
- A historical Go Tool call can still produce a submitted Tool error result when needed to complete a model Tool-call sequence.

### User argument UI

The Composer Tool options UI uses the Tool `UserArgSchema`.

It supports:

- JSON object editing.
- Required-key display.
- Schema display.
- Example generation.
- Invalid JSON feedback.
- Required-key validation.
- Persisted user argument instances.
- Separate Tool option state for attached, conversation, and web-search selections.

### Auto-execution

Auto-execution is a Composer behavior.

For Tool Store Tools:

- Go Tools may be auto-executed.
- SDK Tools are not locally runnable by Composer.
- Web-search SDK behavior is handled by provider inference.
- Tool auto-execution can be overridden per selection.
- Tool default auto-execution originates from the Tool Artifact.

MCP and Skill auto-execution remain owned by their respective runtime domains.

### Tool choice uniqueness

The Tool Aggregate ensures Tool selection identity and Tool choice ID correctness for Tool Store choices.

The inference wrapper owns global provider request uniqueness after combining:

```text
Tool Store choices
  + Skill Runtime choices
  + MCP choices
```

The inference wrapper is responsible for ensuring provider-facing Tool names and choice IDs are valid for one request.

## Tool management UI

### Tool Collections page

The Tool management page presents built-in Tool Collections.

A user can:

- Browse Tool Collections.
- Expand a Tool Collection.
- Inspect Tool metadata.
- Inspect Tool schemas.
- Inspect Tool version.
- Inspect Tool implementation kind.
- Inspect SDK type when applicable.
- Inspect effective enabled state.
- Enable or disable a Tool Collection.
- Enable or disable a Tool.
- Reload the management projection.

The page does not provide:

- Create Tool Collection.
- Delete Tool Collection.
- Edit Tool Collection.
- Add Tool.
- Delete Tool.
- Edit Tool.
- Import Tool.
- Export Tool.
- Register SDK Tool.
- Register Go Tool.
- Manage MCP Tools.

### Effective Tool state

A Tool row distinguishes:

```text
Enabled
Disabled
Collection disabled
```

The effective enabled state is:

```text
Tool Artifact enabled
  and Tool Collection Artifact enabled
```

The UI retains the individual Tool toggle even when the Collection is disabled so the user can configure Tool state before re-enabling the Collection.

### Implementation display

The UI displays:

| Tool kind | Implementation label | Execution label    |
| --------- | -------------------- | ------------------ |
| Go        | Go                   | Local runtime      |
| SDK       | Provider API         | Provider inference |

For SDK Tools, the UI displays the SDK type.

For Go Tools, the UI displays the Tool auto-execution default.

## Security and boundary rules

The Tool Store prevents arbitrary Tool authoring by requiring:

- Protected built-in Root provenance.
- Configured built-in Tool Source provenance.
- Managed Tool package origin.
- Tool package identity validation.
- Tool Collection membership validation.
- Go Tool registry validation.
- Tool and Collection enabled-state validation for mapped targets.

The Tool Store does not attempt to validate every possible provider SDK vendor surface.

The Tool Store does not attempt to apply a generic backend JSON Schema validator to SDK user configuration.

The Tool Store does not own MCP approval or policy behavior.

The Tool Aggregate and Tool Store protect catalog-managed Tool use. The low-level runtime wrapper intentionally remains capable of direct registered runtime invocation.

## Breaking migration

This is a breaking migration.

The legacy Tool Store is removed:

```text
tools_v1 directory
legacy Tool Bundle manifests
legacy Tool JSON catalog
legacy Tool overlay database
legacy ToolStore lookup APIs
legacy Bundle and Tool tuple identity
legacy Tool Runtime Store dependency
```

The replacement is:

```text
Artifact Store Tool Artifacts
Artifact Store Tool Collection Plugin Artifacts
Artifact Enabled state
Tool Artifact target mapping
mapped Tool targets
Tool Aggregate
Store-independent Tool Runtime
Tool Aggregate inference hydration
```

The migration is direct:

```text
Legacy Bundle
  -> embedded Tool Collection package

Legacy Go Tool
  -> generated Tool Artifact package

Legacy SDK Tool
  -> embedded Tool Artifact package
```

No Tool v2 contract is introduced. The active Tool contract remains `toolv1`.

## Implementation mapping

| Responsibility                    | Implementation mapping                                            |
| --------------------------------- | ----------------------------------------------------------------- |
| Tool declaration contract         | `internal/artifactcontract/declaration/toolv1`                    |
| Tool Artifact domain              | `internal/tool/store/domain`                                      |
| Tool Collection domain            | `internal/tool/store/consumerapi` and shared `collection` package |
| Go Tool registry adapter          | `internal/tool/llmtoolsadapter`                                   |
| Tool installer                    | `internal/tool/store/builtin`                                     |
| Tool Store API                    | `internal/tool/store/consumerapi.API`                             |
| Go Tool Runtime                   | `internal/tool/runtime.Service`                                   |
| Tool Aggregate                    | `internal/tool/aggregate.Service`                                 |
| Mapped Tool target                | `internal/tool/aggregate.TargetV1`                                |
| Tool Artifact target mapper       | `resolve.ArtifactTargetMapper` implemented by Tool Aggregate      |
| Tool management wrapper           | `ToolStoreWrapper`                                                |
| Tool Aggregate wrapper            | `ToolAggregateWrapper`                                            |
| Tool Runtime wrapper              | `ToolRuntimeWrapper`                                              |
| Inference Tool hydration          | `ToolAggregate.HydrateInferenceToolChoices`                       |
| Inference Tool choice assembly    | `ProviderSetAPI`                                                  |
| Composer Tool selection hydration | `hydrateToolSelectionsForUI`                                      |
| Composer Go Tool execution        | `invokeMappedTool` through Tool Aggregate                         |
| Composer MCP Tool execution       | MCP runtime and MCP management APIs                               |
| Tool Collections page             | `frontend/app/tools`                                              |

## Current implementation status

### Available

| Capability                                                    | Status    |
| ------------------------------------------------------------- | --------- |
| Adapted `toolv1` Go implementation kind                       | Available |
| Adapted `toolv1` SDK implementation kind                      | Available |
| SDK Tool semantic types `function`, `custom`, and `webSearch` | Available |
| Built-in Tool Artifacts                                       | Available |
| Built-in Tool Collections                                     | Available |
| Read-only Tool Collection domain                              | Available |
| One Tool Collection route per Tool                            | Available |
| Tool Artifact enablement                                      | Available |
| Tool Collection enablement                                    | Available |
| Go Tool registry adapter                                      | Available |
| Go Tool document generation                                   | Available |
| Static SDK Tool documents                                     | Available |
| Protected Tool package hydration                              | Available |
| Tool Artifact target mapper                                   | Available |
| Mapped Tool target encoding                                   | Available |
| Mapped Tool target revalidation                               | Available |
| Tool Aggregate                                                | Available |
| Store-independent Go Tool Runtime                             | Available |
| Raw low-level Go Tool runtime invocation                      | Available |
| SDK web-search inference projection                           | Available |
| Tool selections using mapped targets                          | Available |
| Conversation ToolSelections persistence shape                 | Available |
| Composer Tool selection hydration                             | Available |
| Composer Tool options UI                                      | Available |
| Tool Collection management UI                                 | Available |
| Go and SDK effective-state display                            | Available |
| MCP Tool execution as a separate runtime concern              | Available |

### Current scope decisions

| Decision                                                                                  | Status      |
| ----------------------------------------------------------------------------------------- | ----------- |
| No Tool v2 contract                                                                       | Intentional |
| No `userCallable` field                                                                   | Intentional |
| No `llmCallable` field                                                                    | Intentional |
| No generic backend JSON Schema validation for SDK user arguments                          | Intentional |
| No separate SDK vendor registry                                                           | Intentional |
| No Tool output schema enforcement                                                         | Intentional |
| No explicit legacy Tool ID field                                                          | Intentional |
| Raw Go function identifiers may be exposed to application management and runtime surfaces | Intentional |
| Raw runtime invocation may bypass Tool Store enablement                                   | Intentional |
| Go auto-execution policy is controlled by the application adapter                         | Intentional |
| Inference wrapper owns global Tool choice uniqueness                                      | Intentional |
| Mapped targets may encode backing Artifact identity                                       | Intentional |

### Not supported

| Capability                                    | Status        |
| --------------------------------------------- | ------------- |
| User-created Tool                             | Not supported |
| User-created Tool Collection                  | Not supported |
| Tool Collection editing                       | Not supported |
| Tool Collection membership editing            | Not supported |
| Arbitrary Go Tool registration                | Not supported |
| Runtime Go plugin loading                     | Not supported |
| Arbitrary SDK Tool registration               | Not supported |
| HTTP Tool declaration                         | Not supported |
| Command Tool declaration                      | Not supported |
| Local SDK Tool execution through Tool Runtime | Not supported |
| MCP Tool duplication into Tool Collections    | Not supported |
| Tool import or export                         | Not supported |
