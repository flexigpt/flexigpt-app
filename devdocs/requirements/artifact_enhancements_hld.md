# Artifact Resolution, Discovery, and Consumer Enhancements HLD

Status: Proposed

This HLD is the discovery, resolution, source-boundary, and consumer-integration companion to `Artifact Contract Enhancements HLD`.

The Contract Enhancements HLD owns:

- Artifact declaration syntax.
- Base and collection contract definitions.
- Flat member fields.
- Member selectors.
- Contained declaration `parameters`.
- Common headers.
- Typed MCP fields.
- Flat MCP policy fields.
- Workspace `members`.
- Plugin naming.
- Text Artifact identity, insertion, and source-content contract fields.

This HLD owns:

- Resolver registration and architecture.
- Source discovery behavior.
- Member-selector expansion.
- Root and built-in lookup.
- Generic fallback providers.
- Explicit built-in lookup.
- Alias resolution.
- Composition expansion.
- Ordering and identity invariants.
- Capability plans.
- Consumer integration.

## 1. Purpose

The Artifact platform must resolve validated portable declarations into source-backed, typed, consumer-safe capability graphs.

The resolver must support:

- Named external members.
- Located external members.
- Contained members.
- Dynamic member selectors.
- Current Root lookup.
- Protected built-in Root fallback.
- Explicit built-in-only lookup.
- Generic fallback providers.
- Source-selected terminal aliases.
- Partial composition graphs.
- Typed consumer capability plans.
- Explicit source refresh.
- Deterministic output without positional member behavior.

The platform flow is:

```text
Source-backed declaration
  -> contract validation
  -> indexed Artifact
  -> typed resolver
  -> resolved composition graph
  -> capability plan
  -> consumer
```

The resolver identifies declarations and relationships.

The resolver does not execute, materialize, connect, schedule, or invoke Artifacts.

## 2. Goals

This HLD must provide:

- One private shared resolution core.
- One registered resolver implementation per Artifact type.
- One public typed resolver API per Artifact type.
- Generic fallback-provider infrastructure.
- Tool and Model fallback providers as initial registrations.
- Explicit built-in-only relationship lookup.
- Strict located-reference behavior.
- Explicit source refresh for selectors and local locators.
- Typed member-selector expansion.
- Shared Text source behavior.
- No Text member selectors.
- Stable relationship identity independent of member position.
- No generic Artifact Store membership or selector-result entity.
- Consumer-safe partial capability plans.

## 3. Non-goals

This HLD does not define:

- Declaration JSON Schema.
- Contract field syntax beyond the required resolution-facing additions.
- Tool execution.
- Model execution.
- Agent execution.
- Team execution.
- Loop execution.
- Workflow scheduling.
- Prompt rendering policy.
- Prompt budget policy.
- MCP connection lifecycle.
- MCP secret values.
- MCP installation values.
- OAuth token handling.
- Artifact enablement policy.
- Managed package lifecycle.
- User-facing authoring UI.
- Generic Artifact Store schema changes.

## 4. Resolver architecture

## 4.1 Per-type resolver registration

The application must register a resolver for every supported Artifact type.

```text
text
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
workspace
```

The resolver registry is an application-composition and bootstrap concern.

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

Each type resolver owns:

- Type-specific declaration interpretation.
- Type-specific child relationship expansion.
- Type-specific selector eligibility.
- Type-specific direct relationship behavior.
- Type-specific fallback-provider registration.
- Type-specific consumer result projection.

## 4.2 Public typed resolver APIs

Consumers must use typed public resolver APIs.

Examples:

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

Composition-capable type resolvers may also expose typed refresh APIs.

Examples:

```text
RefreshPlugin
RefreshAgent
RefreshTeam
RefreshWorkspace
```

The public consumer API must not require callers to invoke a generic method such as:

```text
ResolveArtifact(type, name)
```

A generic resolver remains internal implementation infrastructure.

## 4.3 Private shared resolution core

The shared resolution core may implement common behavior for every type resolver.

```text
Shared resolution core
  -> current Root lookup
  -> protected built-in Root lookup
  -> fallback-provider dispatch
  -> located declaration lookup
  -> contained declaration lookup
  -> selector expansion
  -> alias traversal
  -> cycle detection
  -> limits
  -> diagnostics
  -> normalized ordering
```

The shared core must not import:

```text
Tool runtime
Model runtime
Skill runtime
MCP runtime
Agent runtime
Team runtime
Workflow runtime
Prompt runtime
Secret storage
MCP installation storage
```

Type resolvers and consumer adapters provide type-specific integration.

## 4.4 Resolver extension points

A type resolver may register:

- A fallback provider.
- Selector support.
- Contained declaration support.
- Located-source alias support.
- Type-specific child relationship expansion.
- Type-specific capability projection.

A new type must not require modifying a public generic resolver API.

It must register its own type resolver and explicit capabilities.

## 5. Lookup scopes and fallback providers

## 5.1 Default named lookup

A named external member without explicit scope resolves in this order:

```text
Current Root Artifact
  -> protected built-in Root Artifact
  -> registered fallback provider
  -> unavailable
```

Example:

```yaml
- type: tool
  name: searchfiles
```

A current Root Artifact takes precedence over a protected built-in Artifact.

A protected built-in Artifact takes precedence over a non-Artifact fallback target.

## 5.2 Explicit built-in lookup

A named external member may explicitly request built-in lookup.

```yaml
- type: tool
  name: searchfiles
  scope: builtin
```

This resolves in this order:

```text
Protected built-in Root Artifact
  -> registered built-in fallback provider
  -> unavailable
```

It must not search the current Root.

This supports cases where a user explicitly intends the application-provided Tool, Model, or future built-in capability even when a current Root Artifact has the same type and name.

`scope: builtin` is:

- A relationship lookup directive.
- Not a Root ID.
- Not a cross-Root import mechanism.
- Not a source locator.
- Not an ArtifactRef.
- Not valid for contained members.
- Not valid for located members.
- Not valid for member selectors.

## 5.3 Generic fallback provider model

Fallback providers are generic infrastructure.

They are registered per Artifact type.

```text
Artifact type resolver
  -> optional fallback provider
  -> fallback result or unavailable
```

Tool and Model are the initial fallback-provider consumers.

```text
ToolResolver
  -> Tool fallback provider

ModelResolver
  -> Model fallback provider
```

A fallback provider may return:

- An Artifact-backed protected built-in target.
- A non-Artifact application-provided target.
- No target.

A non-Artifact fallback target is a mapped target.

Mapped targets have no:

```text
ArtifactRef
Definition
Source binding
Source resource
Artifact enablement
Artifact lifecycle
Artifact local data
```

Mapped target provenance must identify:

```text
fallback provider
artifact type
logical name
lookup scope
built-in classification
```

A future Artifact type may register a fallback provider only when its type resolver and consumer explicitly support mapped targets.

## 5.4 Fallback restrictions

Fallback providers are valid only for named external relationships.

They are not used for:

- Located external relationships.
- Contained members.
- Member selectors.
- Source-selected MCP aliases.
- Source-selected MCP policy aliases.
- Arbitrary caller-provided ArtifactRefs.

Current Root ambiguity prevents protected built-in and fallback-provider lookup.

Protected built-in ambiguity prevents fallback-provider lookup.

Artifact enablement does not affect declaration lookup.

Runtime readiness does not affect declaration lookup.

## 6. Text Artifact behavior

## 6.1 Text source model

Text is the one source-text Artifact type.

```text
Inline text
  -> content + insert

Single source file
  -> locator + insert

Source directory
  -> locator + insert + include + exclude
```

Text may represent:

- Inline content.
- One source file.
- One source directory.
- A deterministic selected set of text files.

Text support:

```text
insert
content
locator
mediaType
include
exclude
```

Text identity is:

```text
(text, name, insert)
```

`insert: instructions` identifies behavioral or directive text.
`insert: user` identifies informational or reference text inserted as a
user-message contribution.

The physical-format decoder emits:

```text
AGENTS.md and CLAUDE.md
  -> text with insert=instructions
```

## 6.2 Text does not support member selectors

Text must not support selector `base`.

Invalid:

```yaml
- type: text
  base: ./instructions
```

Invalid:

```yaml
- type: text
  base: ./docs
```

A text Artifact does not discover multiple child Artifacts.

It directly contributes one or more text resources.

Valid contained source-backed Instruction:

```yaml
- type: text
  name: repository-rules
  insert: instructions
  locator: ./instructions
  parameters:
    include:
      - "**/*.md"
    exclude:
      - "**/drafts/**"
```

Valid contained source-backed Text:

```yaml
- type: text
  name: repository-docs
  insert: user-message
  locator: ./docs
  parameters:
    include:
      - "**/*.md"
    exclude:
      - "**/generated/**"
```

## 6.3 Text resource selection

For Text:

- A file locator contributes that file.
- A directory locator contributes files selected by `include` and `exclude`.
- `include` and `exclude` match source resource paths relative to the directory locator.
- Matched files are materialized in deterministic normalized path order.
- Source array order does not determine text-resource ordering.
- Inline `content` cannot be combined with `locator`, `include`, or `exclude`.
- File and directory validation occurs when verified source resources are inspected.
- A file locator with directory-only selection patterns is invalid at source-resource validation time.

Text `include` and `exclude` must never be interpreted as Artifact declaration discovery rules.

## 6.4 Text Artifact relationship behavior

A named external Text member must include the expected insertion identity:

```yaml
- type: text
  name: repository-rules
  insert: instructions
  locator: ./instructions/repository-rules.yaml
```

A contained Text member may select source text:

```yaml
- type: text
  name: repository-rules
  insert: instructions
  locator: ./AGENTS.md
  parameters:
    mediaType: text/markdown
```

The presence of `parameters` distinguishes a contained Text declaration from an external located Text relationship. `insert` remains outside `parameters`.

## 7. Member selectors and discovery

## 7.1 Selector eligibility

Member selector support is registered per Artifact type resolver.

A type resolver must explicitly declare whether it supports selectors.

```text
Type resolver
  -> supports selector
  -> does not support selector
```

Instruction and Context explicitly do not support selectors.

Member selectors are intended for Artifact types where a directory commonly contains multiple independently declared capabilities.

Typical selector-capable types include:

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

Selector capability remains type-specific and must be explicitly registered.

## 7.2 Selector meaning

A member selector selects source-backed Artifact occurrences.

```yaml
- type: skill
  base: ./skills
  include:
    - "**/SKILL.md"
  nameInclude:
    - "review-*"
```

Conceptually:

```text
Selector base
  -> declaration candidate enumeration
  -> physical-format decoding
  -> Artifact type filtering
  -> logical-name filtering
  -> selected Artifact occurrences
```

Selectors do not:

- Select arbitrary files as prompt text.
- Select arbitrary ArtifactRefs supplied by callers.
- Select current Root Artifacts by symbolic lookup.
- Use protected built-in fallback.
- Use fallback providers.
- Use Artifact enablement.
- Use runtime readiness.
- Use local installation state.
- Use arbitrary metadata queries.

## 7.3 Selector refresh behavior

Normal resolution is read-only.

Selectors may require explicit source refresh before matching Artifacts are indexed.

The platform must provide typed refresh entry points backed by shared internal refresh logic.

```text
RefreshWorkspace
RefreshPlugin
RefreshAgent
RefreshTeam
```

The shared refresh process must:

```text
Start from a selected composition root
  -> inspect reachable selectors and located declaration relationships
  -> calculate local discovery closure
  -> update only affected Source discovery configuration
  -> refresh affected Sources
  -> repeat within bounded limits
```

Normal reads must not mutate discovery:

```text
ResolveWorkspace
ResolvePlugin
ResolveAgent
ResolveWorkflow
Build capability plan
Build prompt plan
Build Skill plan
Build MCP plan
```

## 7.4 Selector scope

A selector is scoped to the Source containing the declaring Artifact occurrence.

```yaml
- type: skill
  base: ./skills
```

The implementation must:

1. Resolve `base` relative to the containing declaration source.
2. Stay within the containing Source.
3. Stay within the containing Root.
4. Enumerate candidate declaration files below the resolved base.
5. Apply source-path include and exclude patterns.
6. Decode supported physical declaration formats.
7. Filter decoded Artifacts by selector type.
8. Filter decoded Artifact names by name patterns.
9. Produce stable selector match occurrences.

Selectors must not:

- Scan arbitrary filesystem paths.
- Scan another Root.
- Scan a protected built-in Root from a mutable user Root.
- Fetch remote content without an explicit Source adapter.
- Treat Context or Instruction resource patterns as declaration scans.

## 7.5 Selector results

A selector produces:

```text
Available with zero matches
Available with one or more matches
Unavailable selector
```

A zero-match selector is valid.

It is an available empty membership set.

A selector is unavailable when its declared base cannot be inspected or is outside permitted source scope.

Individual selected Artifact occurrences retain their own availability result.

```text
Selector available
  -> matching Artifact A: available
  -> matching Artifact B: unavailable
  -> matching Artifact C: available
```

Source decode errors remain source diagnostics.

They do not invalidate valid selector matches.

## 8. Relationship resolution

## 8.1 Resolution statuses

Every relationship occurrence has one status:

```text
available
unavailable
ambiguous
```

The resolver must preserve:

- Named member occurrences.
- Located member occurrences.
- Contained member occurrences.
- Selector occurrences.
- Selector match occurrences.
- Relationship-local `overrides`.
- Relationship-local `use`.
- Lookup scope.
- Target provenance.
- Diagnostics.

The resolver must not:

- Silently drop unavailable relationships.
- Select one ambiguous target by source order.
- Replace unavailable relationships with another target.
- Mutate target configuration.
- Execute Artifacts.
- Materialize resources as part of declaration resolution.

## 8.2 Named external relationships

An unlocated named relationship normally resolves by: (type, name, optional scope)

Text resolves by: (text, name, insert, optional scope)

Default scope:

```text
Current Root
  -> protected built-in Root
  -> registered fallback provider
  -> unavailable
```

Built-in scope:

```text
Protected built-in Root
  -> registered built-in fallback provider
  -> unavailable
```

Multiple aliases resolving to one terminal Artifact count as one symbolic target.

Multiple distinct terminal Artifacts with the same type and name are ambiguous.

## 8.3 Located relationships

A located relationship resolves by:

```text
(type, name, locator, optional MCP server selector)
```

It must:

1. Resolve the locator relative to the containing declaration occurrence.
2. Select the declaration occurrence at that location.
3. Confirm expected type.
4. Confirm expected name.
5. Confirm expected Text insert target when `type: text`.
6. Confirm MCP server selector when applicable.
7. Follow source-selection aliases to a terminal Artifact.

A located relationship does not:

- Search the current Root by name.
- Search the protected built-in Root.
- Use fallback providers.
- Use `scope: builtin`.
- Merge local body fields into its target.

## 8.4 Contained relationships

A contained relationship resolves through its stable structural declaration occurrence.

```text
Containing declaration
  -> structural relationship path
  -> contained Artifact
```

Contained relationships do not use Root-wide symbolic lookup.

Examples of stable contained paths:

```text
members/text/instructions/repository-rules
members/mcp/local-files
allowedTools/tool/searchfiles
body/agent/code-reviewer
nodes/review/agent/reviewer
loop/reviewer-loop
workflow/change-workflow
```

Contained structural paths must not use source array indexes.

## 8.5 Source-selected aliases

Source-selected MCP and MCP policy declarations resolve to terminal Artifacts.

The resolver must:

- Follow alias chains.
- Detect alias cycles.
- Preserve alias provenance.
- Return terminal ArtifactRefs to consumers.
- Treat aliases to the same terminal Artifact as one symbolic target.
- Preserve unavailable alias relationships as unavailable.

The resolver must not merge local configuration into source-selected aliases.

## 9. Composition expansion

## 9.1 Common expansion rules

Composition expansion is recursive and typed.

The resolver must preserve all declared relationship occurrences even when a consumer later deduplicates terminal ArtifactRefs.

The resolver must recursively expand:

| Artifact type | Relationship fields           |
| ------------- | ----------------------------- |
| `plugin`      | `members`                     |
| `agent`       | `members`, `loop`, `workflow` |
| `team`        | `members`, `loop`, `workflow` |
| `skill`       | `allowedTools`                |
| `mcp`         | Direct `policy` reference     |
| `loop`        | `body`                        |
| `workflow`    | Node targets                  |
| `workspace`   | `members`                     |

The resolver does not expand:

```text
Text source files into child Artifacts
MCP include tools into child Artifacts
MCP policy tool policy keys into child Artifacts
```

## 9.2 Plugin expansion

Plugin expansion traverses:

```text
plugin.members
```

Plugins may contain named members, contained members, and selector members.

Plugin member selectors may select all supported selector-capable Artifact types.

Plugin itself does not apply runtime behavior.

## 9.3 Agent expansion

Agent expansion traverses:

```text
agent.members
agent.loop
agent.workflow
```

The resolver preserves Agent relationship behavior for consumers.

```text
Tool member
  -> overrides.autoExecute

Model member
  -> overrides.includeSystemPrompt

Skill member
  -> use.mode
```

The resolver does not execute those behaviors.

## 9.4 Team expansion

Team expansion traverses:

```text
team.members
team.loop
team.workflow
```

Team member selectors are supported where the selected type supports selectors.

Team has no Team-specific relationship behavior in the current contract.

## 9.5 Loop expansion

Loop expansion resolves:

```text
loop.body
```

A Loop body is one target relationship.

It does not accept selectors.

The resolver validates the target relationship but does not execute a loop.

## 9.6 Workflow expansion

Workflow expansion resolves every node target.

Workflow node identity is based on:

```text
workflow declaration
node id
```

Workflow node target fields are direct node siblings:

```text
id
join
type
name
locator
parameters
overrides
use
scope
```

A Workflow node does not accept a selector because it represents one target application.

Workflow arrays describe graph records only.

They do not define execution sequence.

## 9.7 Workspace expansion

Workspace expansion traverses:

```text
workspace.members
```

Workspace does not use:

```text
workspace.declarations
workspace.roots
```

A Workspace member may be:

- A named external Artifact.
- A located external Artifact.
- A contained Text Artifact.
- A contained Artifact of another supported type.
- A selector member.

Workspace Context and Instruction members directly supply text resources.

Workspace selectors discover and select Artifact declarations.

These are separate concerns.

```text
Workspace Context
  -> source text files

Workspace selector
  -> declared Artifact occurrences
```

## 10. Capability plans and completeness

A capability plan is a consumer-safe view of the resolved graph.

It must preserve:

- Relationship occurrence identity.
- Selector occurrence identity.
- Selector match identity.
- Artifact target provenance.
- Built-in target provenance.
- Mapped target provenance.
- Relationship status.
- Diagnostics.
- Relationship `overrides`.
- Relationship `use`.
- Lookup scope.

A consumer may request strict completeness.

```text
RequireComplete = true
```

A strict consumer rejects unavailable or ambiguous required relationships.

A non-strict consumer may use available portions of a capability plan while reporting unavailable and ambiguous relationships.

The resolver does not decide runtime readiness.

## 11. Ordering and structural identity

## 11.1 No positional member semantics

Member order must not create semantic behavior.

This applies to:

```text
plugin.members
agent.members
team.members
workspace.members
skill.allowedTools
member-selector results
workflow.nodes
workflow.edges
workflow.start
```

Moving a member from one array position to another must not:

- Change relationship identity.
- Change contained Artifact identity.
- Change local Artifact state.
- Change selector identity.
- Change selector matches.
- Change lookup precedence.
- Change capability precedence.
- Change runtime behavior.
- Change store ownership.
- Change cache keys.
- Cause a target to be treated as deleted and recreated.

## 11.2 Stable identity rules

Relationship identity must use normalized semantic fields.

Examples:

```text
Named member
  -> parent occurrence + field + type + name + locator + scope + use + overrides

Contained member
  -> parent occurrence + field + type + name

Member selector
  -> parent occurrence + field + canonical selector fields

Workflow node
  -> workflow occurrence + node id

Workflow edge
  -> workflow occurrence + from + to + canonical match
```

Contained declaration paths must not use array position.

Selector identities must not use array position.

Workflow nodes use `id`, not node array position.

## 11.3 Duplicate normalized relationships

Exact duplicate normalized relationships in one parent field are invalid.

Examples:

```text
Two identical named members
Two identical selectors
Two identical Workflow edges
Two contained declarations with the same type and name
```

Distinct relationships to the same target remain valid when their normalized relationship fields differ.

Examples:

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

A consumer may later deduplicate terminal ArtifactRefs for capability output, but the graph preserves distinct relationship occurrences.

## 11.4 Source bytes and revisions

Reordering source arrays may change:

```text
Raw source bytes
Definition digest
Source content digest
Artifact revision
```

That is acceptable.

It must not change semantic composition behavior.

## 11.5 Intrinsically ordered values

This rule applies to composition and graph membership arrays.

It does not remove order from values whose order is intrinsically meaningful.

Examples include:

```text
Tool command arguments
MCP command arguments
Ordered prompt text supplied by a runtime request
Runtime invocation argument arrays
```

`include` and `exclude` pattern arrays are unordered within each field.

Their cross-field rule remains:

```text
include first
exclude second
```

## 12. Consumer integration

## 12.1 Common consumer requirements

Consumers must:

- Use typed public resolver APIs.
- Accept current Root and protected built-in ArtifactRefs when authorized by resolution.
- Preserve relationship diagnostics.
- Distinguish Artifact-backed and mapped targets.
- Avoid arbitrary caller-supplied cross-Root ArtifactRefs.
- Apply `overrides` and `use` only where their type and container contracts define behavior.
- Decide runtime readiness separately from resolution availability.
- Avoid positional interpretation of member arrays.

## 12.2 Text consumers

Text consumers materialize:

```text
Inline text
Single verified source file
Verified directory file set
Include and exclude patterns
Media type
Insert target
```

The consumer routes Text using `insert`.

```text
insert=instructions
  -> behavioral contribution

insert=user-message
  -> user-message contribution
```

A Workspace prompt consumer may use one shared Text source-materialization adapter.

## 12.3 Tool and Model consumers

Tool and Model consumers must support:

```text
Artifact-backed current Root targets
Artifact-backed protected built-in targets
Mapped fallback targets
```

Mapped targets are accepted only by consumers that explicitly support the mapped target type.

A consumer requiring verified source resources must reject non-Artifact mapped targets when source resources are required.

## 12.4 MCP consumer

The MCP consumer must:

- Read direct MCP declaration fields.
- Stop reading runtime configuration from metadata extensions.
- Resolve direct MCP policy references.
- Use terminal MCP ArtifactRefs for installation-local state.
- Keep input values, secret references, selected profile, OAuth tokens, and connection state outside portable declarations.

## 12.5 Workspace consumer

The Workspace consumer must:

- Resolve `workspace.members`.
- Support contained Instruction and Context members.
- Support selector-expanded Artifact members.
- Treat Workspace selectors as Artifact selection.
- Treat Workspace Context and Instruction locators as text-resource selection.
- Avoid inferring capabilities from every Artifact in the Root.
- Use explicit capability selection and completeness policy.

## 13. Source and Artifact Store boundaries

## 13.1 Source boundary

Sources own:

- Physical content access.
- Candidate enumeration.
- Decoder registration.
- Decoder hints.
- Source refresh state.
- Integrity validation.
- Source path safety.
- Discovery limits.

The resolver requests source refresh through typed refresh APIs.

The resolver must not directly open arbitrary filesystem paths.

## 13.2 Artifact Store boundary

Artifact Store persists:

```text
Root
Source
Definition
Artifact
Source binding
Artifact local data
Refresh state
```

Artifact Store must not persist:

```text
Generic Plugin membership rows
Generic selector-match rows
Relationship position fields
Member array indexes
Generic parent ownership
Generic reverse membership
Cascade lifecycle rules
```

A selector declaration is part of the containing Artifact definition.

Selector matches are derived resolution output.

A cache may store normalized selector results for efficiency, but it must not become generic ownership or positional membership state.

## 13.3 Local state boundary

Portable declarations do not contain local runtime state.

Examples:

| Concern                       | Owner                          |
| ----------------------------- | ------------------------------ |
| Artifact enabled state        | Generic Artifact metadata      |
| Workspace runtime disablement | Workspace local Artifact data  |
| MCP installation input values | MCP local installation data    |
| MCP secret references         | MCP local installation data    |
| Secret values                 | Secret storage                 |
| OAuth tokens                  | MCP runtime and secret storage |
| Selected MCP profile          | MCP local installation data    |
| Additional local MCP policies | MCP local installation data    |
| Prompt budget                 | Runtime configuration          |
| Tool arguments                | Runtime request                |
| Model invocation values       | Runtime request                |

## 14. Implementation requirements

### 14.1 Resolver registry

- Add a registry of per-type resolvers.
- Register one resolver for each supported Artifact type.
- Expose per-type public resolver APIs.
- Keep generic traversal and graph logic internal.
- Remove public reliance on a generic Artifact type resolver endpoint.

### 14.2 Fallback providers

- Replace Tool and Model-specific map logic with generic fallback-provider infrastructure.
- Register Tool and Model fallback providers first.
- Add built-in fallback provenance.
- Support explicit `scope: builtin`.
- Prevent `scope: builtin` from searching the current Root.
- Prevent mapped fallback for located, contained, and selector relationships.

### 14.3 Text Artifact support

- Remove Instruction and Context type-specific resolution paths.
- Add one Text resolver with required insertion identity.
- Ensure Text supports files and directories.
- Reject `base` member selectors for Text.
- Reuse one source-text materialization path where practical.
- Route consumer behavior through `text.insert`.

### 14.4 Selector support

- Add selector eligibility to type resolver registration.
- Implement selector discovery closure through explicit refresh only.
- Match selector path patterns against declaration-source paths.
- Match selector name patterns against decoded logical names.
- Preserve selector and selector-match occurrences.
- Sort selector results by stable source-backed identity.
- Do not use selector array index as identity.

### 14.5 Ordering support

- Remove positional relationship identity from resolvers, caches, APIs, and persistence.
- Use stable contained paths based on semantic fields.
- Use canonical selector identity.
- Use Workflow node IDs for node identity.
- Reject exact duplicate normalized relationships.
- Preserve source-byte digest behavior independently from semantic ordering.

### 14.6 Consumer updates

- Update all consumers to use per-type public resolvers.
- Update Tool and Model consumers for mapped fallback targets.
- Update MCP consumer for direct MCP fields.
- Update Workspace consumer for `members` only.
- Update prompt consumers for shared Instruction and Context source behavior.
- Update Plugin, Agent, Team, Skill, Loop, and Workflow consumers for non-positional relationship identity.

## 15. Final decisions

- Current Root lookup precedes protected built-in Root lookup.
- Fallback providers are generic infrastructure.
- Tool and Model are the first fallback-provider consumers.
- Explicit `scope: builtin` skips current Root lookup.
- Built-in fallback providers may return non-Artifact mapped targets.
- Resolver registration is per Artifact type.
- Public resolution APIs are per Artifact type.
- Shared generic resolution remains internal.
- Text replaces Instruction and Context.
- Text requires identity-level `insert: instructions | user`.
- Text supports `locator`, `include`, and `exclude`, but not selector `base`.
- Member selectors are type-specific and explicitly registered.
- Member selectors are expanded only through explicit refresh.
- Normal resolution is read-only.
- Located relationships never use fallback.
- Contained relationships resolve by stable structural occurrence.
- Selector results are direct source-backed Artifact occurrence selections.
- Workspace resolves `members`, not `declarations` or `roots`.
- Member order has no semantic identity or runtime behavior.
- Source reordering may change document bytes and digests but not composition behavior.
- Artifact Store does not persist generic membership, selector matches, or relationship positions.
