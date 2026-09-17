# Artifact Contract Enhancements HLD

Status: Proposed
Scope: Portable artifact declaration contracts only

This HLD supersedes the contract-related parts of:

- `Shareable and Composable AI Artifact Declarations`
- `Shareable Artifact Declaration and Resolution Improvements`

It intentionally does not define resolution, source refresh, root fallback, Artifact Store persistence, runtime execution, secrets, installation state, or lifecycle behavior. Those belong in separate HLDs.

## 1. Purpose

The artifact declaration platform needs a simpler, stricter, and more portable contract model.

The previous design accumulated several sources of unnecessary complexity:

- Declaration instances could contain both `$schema` and `apiVersion`.
- Collection members used inconsistent shapes.
- Workflow nodes used a `target` wrapper.
- Contained declaration fields were flat beside relationship fields.
- Model declarations allowed an open-ended `parameters` map.
- MCP declaration behavior was hidden in a namespaced metadata extension.
- MCP policy used an unnecessary `body` wrapper.
- Workspace mixed source discovery directives and selected capability roots.
- Collections could only name individual members, not select all matching Artifacts under a directory.

This HLD defines the new declaration contract model.

The primary goals are:

- Keep standalone declarations flat and readable.
- Keep member relationships flat and readable.
- Use `type`, `name`, `locator`, `parameters`, `overrides`, and `use` as sibling member fields.
- Keep target-owned fields separate from relationship-owned fields.
- Remove hidden runtime configuration from `metadata`.
- Support reusable dynamic membership through portable member selectors.
- Treat Workspace as a member-based collection contract.
- Split contracts into base contracts and collection contracts.
- Keep contract validation separate from resolution and runtime behavior.

The intended declaration flow is:

```text
Physical source format
  -> normalized portable declaration contract
  -> separate resolution system
  -> separate runtime consumer
```

This HLD defines only the normalized portable declaration contract.

## 2. Goals

The enhanced contract system must provide:

- One portable standalone declaration shape for every Artifact type.
- One common member relationship shape for all collection-style fields.
- Explicit contained declarations through `parameters`.
- Explicit dynamic membership through member selectors.
- Direct type-specific fields instead of runtime metadata extensions.
- Direct typed Model fields instead of an unbounded parameter map.
- Direct typed MCP fields instead of a namespaced extension object.
- Direct flat MCP policy fields instead of a `body` wrapper.
- A Workspace contract based on `members`, not `declarations` and `roots`.
- Strict JSON Schema validation with no implicit behavior hidden in unknown fields.
- Clear ownership of target configuration, relationship configuration, local installation data, and runtime state.

## 3. Non-goals

This HLD does not define:

- Root-scoped symbolic resolution.
- Built-in Root fallback.
- Tool or Model mapped-name resolution.
- Locator materialization.
- Source registration.
- Source discovery implementation.
- Source refresh behavior.
- Member-selector expansion algorithms.
- Ambiguity handling.
- Cycle detection.
- Capability plans.
- Runtime execution.
- Workflow scheduling.
- Prompt construction.
- MCP connection management.
- MCP installation values.
- Secret storage.
- OAuth token management.
- Artifact enablement behavior.
- Artifact Store ownership, deletion, or persistence behavior.

Those concerns may consume these contracts, but they are not part of this HLD.

## 4. Contract categories

Artifact contracts are organized into two groups.

This grouping is documentation and schema organization only. It is not a new portable declaration field.

```text
Base contracts
  -> independently meaningful capabilities or configuration

Collection contracts
  -> select, group, compose, or structurally organize Artifacts
```

### 4.1 Base contracts

Base contracts are:

```text
text
model
tool
skill
mcp
mcp.policy
```

Base contracts own their direct declaration fields.

Examples:

- A Model owns its provider-qualified model identifier and defaults.
- A Tool owns its implementation and callable properties.
- An MCP declaration owns its transport, authentication requirements, installation input declarations, and connection profiles.
- A Text Artifact owns its content or source-content selection patterns and insertion target.

A base contract may still contain a plural relationship field when that is part of its own declaration semantics.

Example:

```text
skill.allowedTools
```

`skill.allowedTools` uses the common member relationship grammar, but Skill remains a base contract.

### 4.2 Collection contracts

Collection contracts are:

```text
plugin
agent
team
loop
workflow
workspace
```

Collection contracts own relationship and structural fields.

Examples:

- A Plugin owns a reusable selected member set.
- An Agent owns its prompt, member relationships, and optional Loop or Workflow program.
- A Team owns member relationships and optional coordination program.
- A Loop owns its body and completion condition.
- A Workflow owns nodes, edges, joins, and start nodes.
- A Workspace owns selected members for one repository or application experience.

Collection contracts do not own or mutate independently declared member Artifacts.

## 5. Artifact vocabulary

The portable Artifact type vocabulary is:

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

The previous portable type:

```text
collection
```

is retired and replaced by:

```text
plugin
```

A future Artifact type extension must provide:

- A concrete portable `type`.
- A standalone declaration schema.
- A contained declaration payload schema.
- Explicit allowed collection-member positions.
- Explicit member-selector support when it makes sense.
- Explicit relationship `overrides` or `use` semantics when needed.

A new Artifact type must not depend on hidden metadata extensions or arbitrary open-ended properties.

## 6. Schema identity and declaration versioning

Every JSON Schema document retains root schema metadata.

```yaml
# JSON Schema document metadata.
# This is not written into Artifact declaration instances.
$schema: https://json-schema.org/draft/2020-12/schema
$id: https://schemas.flexigpt.site/artifact/mcp/v1.json
```

The `$id` identifies the immutable versioned schema document.

Examples:

```text
https://schemas.flexigpt.site/artifact/text/v1.json
https://schemas.flexigpt.site/artifact/model/v1.json
https://schemas.flexigpt.site/artifact/mcp/v1.json
https://schemas.flexigpt.site/artifact/plugin/v1.json
https://schemas.flexigpt.site/artifact/workspace/v1.json
```

Declaration instances do not contain:

```yaml
$schema: ...
apiVersion: ...
schemaVersion: ...
version: ...
```

Rules:

- `$schema` remains at the root of the JSON Schema document.
- `$id` remains at the root of the JSON Schema document.
- `$schema` must not be declared as a property of Artifact declaration instances.
- `apiVersion` is removed from declaration instances.
- Collection-only `version` is removed.
- No common Artifact `version` field is added in this revision.
- If Artifact release versioning is required later, add one optional common `version` field across all Artifact contracts.
- If simultaneous schema versions must coexist later, add one explicit `schemaVersion` field at that time.
- Artifact release version and schema version must remain separate concepts.

## 7. Common standalone declaration header

Every standalone Artifact declaration uses the same flat common header.

```yaml
type: <artifact-type> # Required.
name: <portable-name> # Required.

# Optional source-authored display and discovery fields.
displayName: <display-name>
description: <description>
labels:
  <label-key>: <label-value>

# Optional source or declaration locator.
locator: <locator>

# Optional opaque user annotation map.
# Application behavior must not be hidden here.
metadata:
  <annotation-key>: <json-value>
```

Common header behavior:

- `type` identifies the concrete Artifact contract.
- `name` identifies the Artifact's portable logical name.
- `displayName` is source-authored display text.
- `description` is source-authored descriptive text.
- `labels` are source-authored string classification values.
- `locator` has type-specific meaning.
- `metadata` is opaque user annotation data only.
- Application code must not use `metadata` as a replacement for typed contract fields.
- Reserved application metadata extensions are invalid when a proper typed field exists.

The following MCP metadata extension is specifically retired:

```text
flexigpt.site/mcp-runtime-v1
```

## 8. Common member relationship model

A collection-style field contains member relationships.

Examples:

```text
plugin.members
agent.members
team.members
workspace.members
skill.allowedTools
```

The wire format is flat.

There is no artifact-type key wrapper.

There is no `target` wrapper.

There is no public `CompositionEntry` object.

### 8.1 Named external member

```yaml
- type: skill
  name: reviewing-code
  locator: ./skills/reviewing-code
```

Meaning:

- `type` identifies the expected Artifact type.
- `name` identifies the expected logical Artifact name.
- `locator` optionally narrows the expected declaration occurrence.
- `scope` optionally controls named lookup. When omitted, lookup uses the
  normal current Root then protected built-in Root policy.
  - `scope: builtin` skips current Root lookup and selects only a protected
    built-in Artifact or registered built-in fallback target.
  - `scope` is valid only for a named external member and cannot be combined
    with `locator`, `parameters`, or selector `base`.
- No target declaration body is present.
- This is an external relationship.

### 8.2 Contained member

```yaml
- type: text
  name: local-rules
  insert: instructions
  locator: ./AGENTS.md

  # Presence of `parameters` makes this a contained declaration.
  parameters:
    displayName: Local Rules
    description: Rules defined by the containing declaration.
    labels:
      scope: local
    mediaType: text/markdown
```

A contained member has:

```yaml
type: <artifact-type>
name: <portable-name>
locator: <optional-locator>

parameters:
  # Target-owned common fields, excluding type, name, and locator.
  displayName: <optional-display-name>
  description: <optional-description>
  labels: <optional-label-map>
  metadata: <optional-annotation-map>

  # Target-specific body fields.
```

Rules:

- `text.insert` is required and is a Text identity-level field.
- `text.insert` is outside `parameters`.
- A Text member resolves by `(type, name, insert)`.
- `parameters` is the canonical field name. Do not use `params`.
- `parameters` is not a runtime argument bag.
- `parameters` is not a provider-extension map.
- `parameters` is the target declaration payload for one contained Artifact.
- `parameters` may be `{}` only when the selected Artifact contract permits an otherwise empty contained body.
- Contained target fields must not be placed flat beside `type`, `name`, `locator`, `overrides`, or `use`.
- `type`, `name`, and `locator` remain outer member fields.
- `displayName`, `description`, `labels`, `metadata`, and type-specific target fields belong inside `parameters`.

### 8.3 Relationship fields

A member relationship may contain:

```yaml
- type: tool
  name: searchfiles

  # Optional target-type-specific relationship override.
  overrides:
    autoExecute: true

  # Optional container-owned use behavior.
  use:
    mode: active
```

Ownership rules:

| Field         | Owner                            | Meaning                                                                             |
| ------------- | -------------------------------- | ----------------------------------------------------------------------------------- |
| `type`        | Relationship                     | Expected target Artifact type                                                       |
| `name`        | Relationship                     | Expected target logical name                                                        |
| `locator`     | Relationship or contained target | Target selector for external members, source/resource locator for contained members |
| `parameters`  | Contained target                 | Complete target declaration payload                                                 |
| `overrides`   | Relationship                     | Target-type-specific behavior for one relationship                                  |
| `use`         | Containing contract              | Container-owned behavior for one relationship                                       |
| `base`        | Member selector                  | Directory scope containing zero or more target declarations                         |
| `include`     | Member selector                  | Declaration-source path patterns                                                    |
| `exclude`     | Member selector                  | Declaration-source exclusion patterns                                               |
| `nameInclude` | Member selector                  | Logical Artifact name patterns                                                      |
| `nameExclude` | Member selector                  | Logical Artifact name exclusions                                                    |

`overrides` and `use` never mutate:

- Target Artifact metadata.
- Target Artifact definition.
- Target source content.
- Target installation data.
- Target enablement.
- Target policy.
- Target runtime state.
- Target membership in another Plugin, Agent, Team, or Workspace.

### 8.4 Text insertion identity

`text` replaces the previous `instructions` and `context` Artifact types.

Text requires one direct insertion target:

- `instructions`
- `user-message`

`insert` is part of Text identity and must be present in:

- Standalone Text declarations.
- Named external Text members.
- Located external Text members.
- Contained Text members.

`insert` must not be placed in member `parameters`.

A containing Artifact must not override a selected Text Artifact's insertion
target through `use` or `overrides`.

### 8.5 MCP source selector

MCP supports one additional member field:

```yaml
- type: mcp
  name: github
  locator: ./.mcp.json
  server: github
```

`server` is:

- Valid only for `type: mcp`.
- Valid only for a locator-selected external MCP relationship.
- A source selector for a server in a multi-server MCP configuration.
- Not a contained MCP configuration field.
- Not valid inside `parameters`.

## 9. Common member selector model

A member selector creates dynamic membership.

The recommended term is:

```text
member selector
```

The resulting collection behavior is:

```text
dynamic membership
```

A member selector is not an abstract Artifact type and is not a `target` wrapper.

```yaml
- type: skill
  base: ./skills

  # Optional declaration-source path filters, relative to `base`.
  include:
    - "**/SKILL.md"

  exclude:
    - "**/experimental/**"

  # Optional logical Artifact name filters.
  nameInclude:
    - "review-*"
    - "crud-*"

  nameExclude:
    - "legacy-*"
```

Member selector rules:

- `type` is required.
- `base` is required.
- `base` identifies a directory tree containing candidate declarations.
- `include` and `exclude` match declaration-source paths relative to `base`.
- `nameInclude` and `nameExclude` match decoded logical Artifact names.
- `include` omitted means all candidate declarations of the selected type below `base`.
- `exclude` is applied after `include`.
- `nameExclude` is applied after `nameInclude`.
- A selector does not contain `name`.
- A selector does not contain `locator`.
- A selector does not contain `scope`.
- A selector does not contain `parameters`.
- `text` does not support member selectors.
- Text uses `locator`, `include`, and `exclude` to select source text resources.
- A selector may contain allowed `overrides` or `use` fields.
- A selector may match zero Artifacts.
- A zero-match selector is an empty dynamic membership set, not an invalid contract.
- A selector is valid only in a plural member field.
- A selector does not replace a single-target field such as `loop.body` or a Workflow node.

The initial portable selector base is a local relative path:

```yaml
base: ./skills
```

Support for URL, Git, package, archive, or remote selector bases requires separate Source and discovery support.

### 9.1 Text resource patterns versus selector patterns

These fields use similar syntax but have different contract meanings.

```yaml
# Text resource selection.
- type: text
  name: application-source
  insert: user-message
  locator: ./src
  parameters:
    include:
      - "**/*.go"
```

Meaning: Use matching source files as Text content inserted in a user-message.

```yaml
# Artifact member selection.
- type: skill
  base: ./skills
  include:
    - "**/SKILL.md"
```

Meaning:

```text
Select matching Skill declarations as members.
```

A Text resource pattern must not be treated as Artifact discovery.

A member-selector declaration pattern must not be treated as prompt Context content.

## 10. Relationship behavior matrix

The common member grammar exposes `overrides` and `use`, but each containing contract must explicitly define allowed combinations.

Unknown relationship fields are invalid.

| Containing field     | Named member | Contained member | Selector | Allowed relationship behavior            |
| -------------------- | -----------: | ---------------: | -------: | ---------------------------------------- |
| `plugin.members`     |          Yes |              Yes |      Yes | No `overrides` or `use` in v1            |
| `agent.members`      |          Yes |              Yes |      Yes | Tool and Model overrides; Skill use mode |
| `team.members`       |          Yes |              Yes |      Yes | No `overrides` or `use` in v1            |
| `workspace.members`  |          Yes |              Yes |      Yes | No `overrides` or `use` in v1            |
| `skill.allowedTools` |          Yes |              Yes |      Yes | No `overrides` or `use` in v1            |
| `loop.body`          |          Yes |              Yes |       No | No `overrides` or `use` in v1            |
| `workflow.nodes[]`   |          Yes |              Yes |       No | No `overrides` or `use` in v1            |
| `agent.loop`         |          Yes |              Yes |       No | No `overrides` or `use` in v1            |
| `agent.workflow`     |          Yes |              Yes |       No | No `overrides` or `use` in v1            |
| `team.loop`          |          Yes |              Yes |       No | No `overrides` or `use` in v1            |
| `team.workflow`      |          Yes |              Yes |       No | No `overrides` or `use` in v1            |

The currently defined relationship values are:

```yaml
# Agent Tool member.
- type: tool
  name: searchfiles
  overrides:
    autoExecute: true
```

```yaml
# Agent Model member.
- type: model
  name: reasoning
  overrides:
    includeSystemPrompt: true
```

```yaml
# Agent Skill member.
- type: skill
  name: reviewing-code
  use:
    mode: active
```

Allowed Agent Skill modes are:

```text
available
active
instructions
```

## 11. Base contract behavior

All base contracts follow these rules:

- Standalone declaration fields are flat.
- A base contract owns its typed body fields.
- A base contract must not use hidden metadata extensions for behavior.
- A base contract must not contain local installation values, secrets, runtime state, or Artifact Store state.
- A contained base declaration moves its target-owned fields under member `parameters`.
- A named member without `parameters` is an external relationship.
- A selector may select multiple declared base Artifacts where the containing field allows selectors.
- Source-backed content fields remain type-specific.
- Runtime interpretation remains outside this HLD.

## 12. Base contracts

## 12.1 Text

`text` is a source-text Artifact for inline or source-backed text content. It supports:

```text
content?
mediaType?
locator?
include?
exclude?
insert
```

`text` also supports the common Artifact fields, including `name`, `displayName`, `description`, `labels`, and `metadata`.

A concrete `text` declaration must provide exactly one content source:

- Inline text uses `content`.
- Source-backed text uses `locator`, which may identify a file or directory.
- `content` must not be combined with `locator`, `include`, or `exclude`.

For source-backed text:

- A file `locator` contributes that file only.
- A directory `locator` may use `include` and `exclude` to select files.
- `include` and `exclude` are source-text selectors, not Workspace or Plugin
  Artifact discovery fields.

`text` does not use the member-selector `base` field.

The required `insert` field determines how consumers use the text:

- `insert: instructions` identifies behavioral or directive text.
- `insert: user-message` identifies informational or reference text contributed as a user message.

Text identity is `(text, name, insert)`.

This contract is authoritative for all source-text declarations.

### Inline text example

```yaml
type: text
name: repository-rules
insert: instructions
displayName: Repository Rules
description: Rules for repository changes.

labels:
  category: repository
  owner: platform

metadata:
  acme.example/owner: platform

mediaType: text/markdown
content: |
  Do not modify generated files.
  Keep changes focused.
```

### Source-backed text example

```yaml
type: text
name: application-source
insert: user-message
displayName: Application Source
description: Repository source files relevant to the active task.

labels:
  category: source-code
  scope: repository

metadata:
  acme.example/owner: platform

locator: ./src
mediaType: text/plain

include:
  - "**/*.go"
  - "**/*.ts"
  - "**/*.tsx"
  - "**/*.sql"

exclude:
  - "**/generated/**"
  - "**/vendor/**"
  - "**/node_modules/**"
```

### Contained Text bodies

When a `text` body is declared within another member, place the body fields under
that member's `parameters`.

### Changes from the previous contract

- `context` is replaced by `text`.
- Every Text Artifact requires a direct `insert: instructions` or `insert: user-message` field.
- `insert` is no longer placed in member `parameters`.
- Text identity now includes `insert`.
- Contained Text body fields are placed under member `parameters`.
- Directive and informational content share the same source-selection behavior
  through `locator`, `include`, and `exclude`.

## 12.2 Model

Model represents one provider-qualified model and its supported portable defaults.

Contract fields:

```text
model
systemPrompt?
temperature?
maxOutputTokens?
```

A concrete Model requires `model`.

```yaml
type: model
name: reasoning
displayName: Reasoning Model
description: General-purpose reasoning model.

labels:
  category: reasoning
  provider: anthropic

metadata:
  acme.example/owner: platform

# Required provider-qualified model identifier.
model: anthropic/claude-sonnet

# Optional target-owned Model defaults.
systemPrompt: |
  Be precise, grounded, and concise.

temperature: 0.2
maxOutputTokens: 12000

# There is intentionally no generic `parameters`, `options`,
# or provider-extension map.
```

Changed from the previous contract:

- The generic Model `parameters` map is removed.
- Supported portable Model properties are named direct fields.
- Unknown Model properties are invalid.
- New portable Model settings require explicit schema additions.
- `systemPrompt` is a direct Model-owned field.
- A contained Model uses member `parameters` for its complete Model payload.
- Agent `overrides.includeSystemPrompt` remains relationship-owned and does not mutate the Model declaration.

Example contained Model:

```yaml
- type: model
  name: local-reasoning
  parameters:
    displayName: Local Reasoning Model
    description: Conservative local Model settings.
    model: anthropic/claude-sonnet
    systemPrompt: |
      Prefer conservative conclusions.
    temperature: 0
    maxOutputTokens: 8000
```

## 12.3 Tool

Tool represents one named callable operation.

Contract fields:

```text
userCallable
llmCallable
autoExecute
llmToolType
inputSchema
userArgSchema?
outputSchema?
implementation
```

```yaml
type: tool
name: searchfiles
displayName: Search Files
description: Searches repository files.

labels:
  category: repository
  owner: developer-tools

metadata:
  acme.example/owner: developer-tools

# Tool-owned invocation defaults.
userCallable: false
llmCallable: true
autoExecute: false
llmToolType: function

inputSchema:
  type: object
  required:
    - query
  properties:
    query:
      type: string

userArgSchema:
  type: object
  properties:
    maxResults:
      type: integer
      minimum: 1

outputSchema:
  type: object
  properties:
    matches:
      type: array
      items:
        type: string

# Tool implementation is a strict discriminated union.
implementation:
  kind: go
  function: github.com/flexigpt/flexigpt/tools.SearchFiles
```

Supported implementation kinds must each have strict schemas.

```yaml
# Go Tool.
implementation:
  kind: go
  function: github.com/flexigpt/flexigpt/tools.SearchFiles
```

```yaml
# HTTP Tool.
implementation:
  kind: http
  request:
    method: GET
    urlTemplate: https://example.com/search?q=${query}
  response:
    successCodes:
      - 200
    bodyOutputMode: json
```

```yaml
# SDK Tool.
implementation:
  kind: sdk
  provider: example
  operation: search
```

```yaml
# Command Tool.
implementation:
  kind: command
  command: rg
  args:
    - "--json"
  env:
    NO_COLOR: "1"
```

Changed from the previous contract:

- The minimal Tool contract is expanded with callable, execution, schema, and implementation fields.
- `autoExecReco` is replaced by `autoExecute`.
- Tool command execution belongs in `implementation.kind: command`.
- Command locators are not used for new Tool declarations.
- `autoExecute` on the Tool is the target-owned default.
- `overrides.autoExecute` on an Agent member is relationship-owned.
- Tool enablement, runtime state, and Artifact Store IDs are not Tool declaration fields.

## 12.4 Skill

Skill represents one reusable package-backed Skill capability.

Contract fields:

```text
license?
insert?
allowedTools?
```

A standard concrete Skill requires a package `locator`.

```yaml
type: skill
name: reviewing-code
displayName: Reviewing Code
description: Reviews code changes for correctness and risk.

labels:
  category: software-development
  owner: platform

metadata:
  acme.example/owner: platform

locator: ./skills/reviewing-code
license: Apache-2.0

# Skill-owned insertion behavior.
insert: instructions

allowedTools:
  # Exact Tool member.
  - type: tool
    name: searchfiles

  # Dynamic Tool membership.
  - type: tool
    base: ./tools
    include:
      - "**/*.yaml"
      - "**/*.yml"
      - "**/*.json"
    nameInclude:
      - "repository-*"
```

Changed from the previous contract:

- `insert` becomes a first-class Skill field.
- `allowedTools` uses the common member grammar.
- `allowedTools` supports Tool member selectors.
- `allowedTools` accepts only `type: tool`.
- Standard Skills remain package-backed.
- Skill package resources, scripts, arguments, and session state remain outside the portable declaration contract.

## 12.5 MCP

MCP represents one MCP server capability.

MCP owns:

- Transport and connection declaration.
- Capability include rules.
- Timeout declaration.
- Authentication requirements.
- Installation input declarations.
- Available connection profiles.
- Optional default MCP policy reference.

MCP does not own:

- Input values.
- Secret values.
- Secret references.
- OAuth tokens.
- Selected connection profile.
- Additional installation-local policies.
- Connection state.
- Generic Artifact enablement.
- Protected Root overlays.
- Runtime extension metadata.

```yaml
type: mcp
name: github
displayName: GitHub
description: GitHub MCP server.

labels:
  category: source-control
  owner: platform

# Opaque user annotations only.
# Runtime configuration must not be hidden here.
metadata:
  acme.example/owner: platform

# Inline executable MCP connection.
transport: streamableHTTP
url: https://api.githubcopilot.com/mcp/

headers:
  X-Client-Mode: default

# Optional server capability selection.
include:
  tools:
    - create_issue
    - get_issue
  resources:
    - repository://issues
  prompts:
    - issue-summary

timeoutMS: 60000

auth:
  mode: oauth
  clientCredentialsInput: GITHUB_OAUTH_CLIENT
  clientIDMetadataDocumentURL: https://app.example.com/mcp-client.json

# Installation declarations describe available inputs.
# They do not contain input values or secret values.
install:
  note: GitHub OAuth client credentials may be configured locally.

  inputs:
    GITHUB_OAUTH_CLIENT:
      kind: oauthClientCredentials
      label: GitHub OAuth Client
      required: false
      clientSecretRequired: false

# Portable available profiles.
# The locally selected profile is not part of this declaration.
connectionProfiles:
  local:
    platforms:
      - darwin
      - linux

    http:
      headers:
        X-Client-Mode: local

# `policy` directly identifies an mcp.policy relationship.
# There is no `ref` wrapper.
policy:
  name: builtin-manual-write
  required: true
```

MCP authentication contract:

```yaml
auth:
  mode: none | apiKey | oauth | clientCredentials

  # Valid for OAuth and client credentials modes where applicable.
  clientCredentialsInput: <installation-input-name>

  # Optional HTTPS OAuth client metadata document.
  clientIDMetadataDocumentURL: <https-url>
```

MCP installation input contract:

```yaml
install:
  note: <optional-description>

  inputs:
    <input-name>:
      kind: text | secret | path | oauthClientCredentials
      label: <optional-display-name>
      description: <optional-description>
      note: <optional-description>
      placeholder: <optional-placeholder>
      required: <boolean>
      default: <optional-non-secret-default>
      clientSecretRequired: <boolean>

  allowEnvironment:
    - <environment-variable-name>
```

MCP connection profile contract:

```yaml
connectionProfiles:
  <profile-name>:
    platforms:
      - linux
      - darwin
      - windows

    # Exactly one of `stdio` or `http` is allowed.
    stdio:
      command: <optional-command>
      args:
        - <optional-argument>
      env:
        <environment-name>: <value-template>
      removeEnv:
        - <environment-name>

    http:
      url: <optional-url-template>
      headers:
        <header-name>: <value-template>
      removeHeaders:
        - <header-name>
```

MCP connection validation rules:

- Inline `stdio` MCP requires `transport: stdio` and `command`.
- Inline HTTP MCP requires `transport: streamableHTTP` or `sse` and `url`.
- Inline stdio MCP cannot contain HTTP connection fields.
- Inline HTTP MCP cannot contain stdio connection fields.
- Command execution uses direct `command`, `args`, and `env` fields.
- New MCP declarations do not use command locators.
- A source-selected MCP uses a non-command `locator` and optional `server`.
- A source-selected MCP cannot combine locator selection with inline connection fields.
- A source-selected MCP cannot define local `include`, `auth`, `install`, `connectionProfiles`, `policy`, or timeout overrides.
- Secret values are never valid in portable MCP declarations.
- Secret references are never valid in portable MCP declarations.
- Installation input declarations may identify placeholders but may not provide secret values.

Changed from the previous contract:

- The `flexigpt.site/mcp-runtime-v1` metadata extension is removed.
- `displayName`, `description`, and `labels` move to the common header.
- `timeoutMS` moves to a direct MCP field.
- `auth` moves to a direct MCP field.
- `install` moves to a direct MCP field.
- `connectionProfiles` moves to a direct MCP field.
- `policy.ref` becomes `policy.name`.
- MCP logical version is removed from the extension.
- Local installation data remains outside the declaration.
- `metadata` may not be used as an MCP runtime extension mechanism.

## 12.6 MCP policy

MCP policy represents portable MCP policy rules.

Contract fields:

```text
trustLevel?
defaultPolicy?
toolPolicies?
appsPolicy?
```

```yaml
type: mcp.policy
name: builtin-manual-write
displayName: Built-in Manual Write Policy
description: Trusted write policy requiring manual execution.

labels:
  category: mcp-policy
  owner: platform

metadata:
  acme.example/owner: platform

# Policy fields are flat.
# There is no `body` wrapper.
trustLevel: trusted

defaultPolicy:
  defaultApprovalRule: allow
  defaultExecutionMode: manual
  requireApprovalForUnknownRisk: true
  requireApprovalForWrite: true
  requireApprovalForDestructive: true

# Map keys identify MCP tool names.
toolPolicies:
  delete_issue:
    approvalRule: ask
    executionMode: manual
    allowStaleDigest: false

appsPolicy:
  enabled: false
  allowAppInitiatedToolCalls: false
  requireApprovalForOpenLink: true
  requireApprovalForContextUpdates: true
```

MCP policy validation rules:

- `trustLevel` is `trusted` or `untrusted`.
- Approval rules are `allow`, `ask`, or `deny`.
- Execution modes are `auto` or `manual`.
- `toolPolicies` map keys identify MCP tool names.
- Nested `toolName` is invalid because it duplicates the map key.
- `expectedDigest`, when present, is a SHA-256 digest.
- `body` is invalid.
- Policy runtime composition and local additional policy state are outside this HLD.

Changed from the previous contract:

- The abstract `body` wrapper is removed.
- Policy fields are direct declaration fields.
- Redundant nested `toolName` is removed.
- Declaration-instance `$schema` and `apiVersion` are removed.
- `displayName` and `labels` are common header fields.

## 13. Collection contract behavior

All collection contracts follow these rules:

- Collection contracts select or structurally compose Artifacts.
- Collection contracts use the common member grammar in plural member fields.
- A named member can refer to one expected Artifact.
- A contained member can declare one Artifact in the containing declaration.
- A member selector can select multiple Artifacts of one concrete type.
- Collection contracts do not create hidden Artifact ownership.
- Collection contracts do not mutate independently declared target configuration.
- Relationship-specific behavior belongs in `overrides` or `use`, only when explicitly defined.
- Member arrays have no positional semantic identity, execution order, or precedence behavior.
- Reordering a member array must not change relationship identity, contained Artifact identity, selector identity, target local state, or cache identity.
- Exact duplicate normalized members and selectors in one containing field are invalid. Distinct relationships to the same target remain valid only when their normalized relationship fields differ.
- Source reordering may change document bytes, content digests, and Artifact revisions, but must not change composition behavior.
- Workflow arrays define graph structure, not source-array execution order.
- Single-target fields do not accept member selectors.
- Source discovery mechanics for member selectors belong to a separate Discovery and Resolution HLD.

## 14. Collection contracts

## 14.1 Plugin

Plugin replaces the previous `collection` contract.

Plugin represents a reusable selected Artifact group.

Contract fields:

```text
members?
```

```yaml
type: plugin
name: crud-harness
displayName: CRUD Harness
description: Shared CRUD implementation and review capabilities.

labels:
  category: harness
  owner: platform

metadata:
  acme.example/owner: platform

members:
  # Contained Context.
  - type: text
    name: crud-source
    insert: user-message
    locator: ./src
    parameters:
      displayName: CRUD Source
      description: Application source relevant to CRUD behavior.
      mediaType: text/plain
      include:
        - "**/*.go"
        - "**/*.ts"
        - "**/*.tsx"
        - "**/*.sql"
      exclude:
        - "**/generated/**"
        - "**/vendor/**"
        - "**/node_modules/**"

  # Dynamic Skill membership.
  - type: skill
    base: ./skills
    include:
      - "**/SKILL.md"
    exclude:
      - "**/experimental/**"
    nameInclude:
      - "crud-*"

  # Dynamic Agent membership.
  - type: agent
    base: ./agents
    include:
      - "**/*.agent.yaml"
      - "**/*.agent.yml"
      - "**/*.agent.md"
    nameInclude:
      - "crud-*"

  # Exact external MCP member.
  - type: mcp
    name: github
    locator: ./.mcp.json
    server: github
```

Plugin behavior:

- Plugin supports named members, contained members, and member selectors.
- Plugin may contain mixed supported Artifact types.
- Plugin has no Plugin-specific `overrides` or `use` behavior in v1.
- `overrides` and `use` are invalid in `plugin.members` until a concrete Plugin consumer behavior is defined.
- Plugin member selection does not make Plugin the owner of targets.

Changed from the previous contract:

- `collection` becomes `plugin`.
- Collection-specific `version` is removed.
- Members use flat `type`, `name`, `locator`, `parameters`, `overrides`, and `use` fields.
- Contained declaration fields move into `parameters`.
- Dynamic member selectors are added.
- There is no artifact-type-key nesting.
- There is no generic `target` wrapper.

## 14.2 Agent

Agent represents an AI Agent declaration.

Contract fields:

```text
prompt?
members?
loop?
workflow?
```

At most one of `loop` and `workflow` is valid.

```yaml
type: agent
name: code-reviewer
displayName: Code Reviewer
description: Reviews requested code changes.

labels:
  category: engineering
  owner: platform

metadata:
  acme.example/owner: platform

# Optional Agent-owned prompt.
prompt:
  mediaType: text/markdown
  content: |
    Review the requested change for defects, regressions, and risk.

members:
  # Contained Instruction member.
  - type: text
    name: reviewer-rules
    insert: instructions
    parameters:
      displayName: Reviewer Rules
      mediaType: text/markdown
      content: |
        Review correctness, security, and maintainability.

  # Exact Model relationship.
  - type: model
    name: reasoning
    overrides:
      includeSystemPrompt: true

  # Dynamic Skill membership.
  - type: skill
    base: ./skills
    include:
      - "**/SKILL.md"
    nameInclude:
      - "review-*"
      - "security-*"
    use:
      mode: active

  # Exact Tool relationship.
  - type: tool
    name: searchfiles
    overrides:
      autoExecute: true

  # Exact MCP relationship.
  - type: mcp
    name: github
    locator: ./.mcp.json
    server: github

  # Exact Plugin relationship.
  - type: plugin
    name: engineering-capabilities
    locator: ./plugins/engineering-capabilities.yaml

# Optional direct program slot.
# `loop` and `workflow` cannot both be present.
loop:
  name: reviewer-loop
  locator: ./loops/reviewer-loop.yaml
```

Agent behavior:

- Agent supports named, contained, and selector members.
- Agent supports direct optional `loop` or `workflow` program slots.
- Agent Tool members may define `overrides.autoExecute`.
- Agent Model members may define `overrides.includeSystemPrompt`.
- Agent Skill members may define `use.mode`.
- An Agent member selector may apply permitted Agent relationship behavior to every selected member.
- Agent `loop` and `workflow` fields are single-target fields and do not allow selectors.

Changed from the previous contract:

- Flat generic members remain flat, but contained target fields move under `parameters`.
- The artifact-type-key nested member form is not used.
- `program` is replaced by direct `loop` or `workflow`.
- `collection` references become `plugin` references.
- `prompt` is a first-class Agent field.
- Agent relationship ownership is explicit through `overrides` and `use`.

## 14.3 Team

Team represents a multi-Agent composition.

Contract fields:

```text
members?
loop?
workflow?
```

At most one of `loop` and `workflow` is valid.

```yaml
type: team
name: change-team
displayName: Change Team
description: Plans, implements, and reviews repository changes.

labels:
  category: engineering
  owner: platform

metadata:
  acme.example/owner: platform

members:
  # Dynamic Agent membership.
  - type: agent
    base: ./agents
    include:
      - "**/*.agent.yaml"
      - "**/*.agent.yml"
      - "**/*.agent.md"
    nameInclude:
      - "change-*"
    nameExclude:
      - "experimental-*"

  # Exact Agent member.
  - type: agent
    name: reviewer
    locator: ./agents/reviewer.agent.yaml

  # Exact shared Plugin member.
  - type: plugin
    name: repository-common
    locator: ./plugins/repository-common.yaml

# Optional direct program slot.
workflow:
  name: change-workflow
  locator: ./workflows/change-workflow.yaml
```

Team behavior:

- Team supports named, contained, and selector members.
- Team supports direct optional `loop` or `workflow`.
- Team has no Team-specific `overrides` or `use` behavior in v1.
- `overrides` and `use` are invalid in `team.members` until explicitly defined.
- Team program fields are single-target fields and do not accept selectors.

Changed from the previous contract:

- `program` is replaced by direct `loop` or `workflow`.
- `collection` references become `plugin` references.
- Team members support dynamic member selectors.
- Contained Team member declaration fields move under `parameters`.

## 14.4 Loop

Loop represents repeated application of one body target.

Contract fields:

```text
body?
maxIterations?
until?
```

```yaml
type: loop
name: reviewer-loop
displayName: Reviewer Loop
description: Repeats review until the result is complete.

labels:
  category: review
  owner: platform

metadata:
  acme.example/owner: platform

# `body` is a real Loop field.
# It contains one direct flat member relationship.
body:
  type: agent
  name: code-reviewer
  locator: ./agents/code-reviewer.yaml

maxIterations: 12

until:
  pointer: /complete
  schema:
    const: true
```

Loop behavior:

- `body` remains a semantic Loop field.
- `body` is not a generic `target` wrapper.
- `body` uses flat member fields: `type`, `name`, `locator`, `parameters`, `overrides`, and `use`.
- `body` accepts one named or contained target.
- `body` does not accept a member selector because a Loop has one body target.
- `maxIterations` and `until` remain direct Loop fields.
- `until.schema` is a JSON Schema.
- `until.pointer`, when present, is an RFC 6901 JSON Pointer.

Changed from the previous contract:

- The Loop body no longer uses an artifact-type-key nested object.
- The Loop body remains nested only because `body` is an actual Loop semantic field.
- The nested member itself is flat.

## 14.5 Workflow

Workflow represents a structural graph of target applications.

Contract fields:

```text
start?
nodes?
edges?
```

```yaml
type: workflow
name: change-workflow
displayName: Change Workflow
description: Plans, implements, and reviews a repository change.

labels:
  category: engineering
  owner: platform

metadata:
  acme.example/owner: platform

start:
  - plan

nodes:
  # Workflow node fields and member fields are siblings.
  # There is no `target` wrapper.
  - id: plan
    type: agent
    name: planner
    locator: ./agents/planner.agent.yaml

  - id: implement
    join: any
    type: agent
    name: implementer
    locator: ./agents/implementer.agent.yaml

  - id: review
    type: agent
    name: reviewer
    locator: ./agents/reviewer.agent.yaml

  # A contained node target uses `parameters`.
  - id: final-check
    type: tool
    name: validate-change
    parameters:
      displayName: Validate Change
      description: Validates the final repository state.
      userCallable: false
      llmCallable: true
      autoExecute: false
      llmToolType: function
      inputSchema:
        type: object
      implementation:
        kind: go
        function: github.com/flexigpt/flexigpt/tools.ValidateChange

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

  - from: review
    to: final-check
    match:
      pointer: /decision
      schema:
        const: approved
```

Workflow behavior:

- A Workflow node contains Workflow structural fields and one flat member target.
- `id`, `join`, `type`, `name`, `locator`, `parameters`, `overrides`, and `use` are sibling node fields.
- A Workflow node does not contain `target`.
- A Workflow node does not accept a member selector because one node has one target.
- Node IDs must be unique.
- `start` values must identify declared node IDs.
- Edge `from` and `to` values must identify declared node IDs.
- `join` is `all` or `any`.
- `join` defaults to `all`.
- Workflow source array position does not define execution order.

Changed from the previous contract:

- The `target` wrapper is removed.
- `type` and `name` move directly onto each Workflow node.
- Contained Workflow node target fields move under node `parameters`.
- Node target behavior is no longer hidden behind a second nested object.

## 14.6 Workspace

Workspace represents one selected Artifact composition for a repository or application experience.

Contract fields:

```text
members?
```

Workspace does not contain:

```text
declarations
roots
```

```yaml
type: workspace
name: checkout-service
displayName: Checkout Service
description: Workspace declaration for the checkout service repository.

labels:
  category: repository
  owner: platform

metadata:
  acme.example/owner: platform

# Workspace contains selected members only.
members:
  # Contained source-backed Instruction.
  - type: instructions
    name: repository-rules
    locator: ./AGENTS.md
    parameters:
      displayName: Repository Rules
      description: Rules for changes in this repository.
      mediaType: text/markdown

  # Contained source-backed Context.
  #
  # This selects source files as Context content.
  # It does not select Artifact declarations.
  - type: test
    name: application-source
    insert: user-message
    locator: ./src
    parameters:
      displayName: Application Source
      description: Checkout service implementation source.
      mediaType: text/plain
      include:
        - "**/*.go"
        - "**/*.ts"
        - "**/*.tsx"
        - "**/*.sql"
      exclude:
        - "**/generated/**"
        - "**/vendor/**"
        - "**/node_modules/**"

  # Dynamic Skill membership.
  - type: skill
    base: ./skills
    include:
      - "**/SKILL.md"
    exclude:
      - "**/experimental/**"
    nameInclude:
      - "checkout-*"
      - "review-*"

  # Dynamic Agent membership.
  - type: agent
    base: ./agents
    include:
      - "**/*.agent.yaml"
      - "**/*.agent.yml"
      - "**/*.agent.md"
    nameExclude:
      - "experimental-*"

  # Dynamic Plugin membership.
  - type: plugin
    base: ./plugins
    include:
      - "**/*.yaml"
      - "**/*.yml"

  # Dynamic MCP membership.
  - type: mcp
    base: .
    include:
      - ".mcp.json"
      - "mcp.json"

  # Exact Workflow member.
  - type: workflow
    name: change-workflow
    locator: ./workflows/change-workflow.yaml
```

Workspace behavior:

- Workspace supports named, contained, and selector members.
- Workspace members replace the previous `declarations` and `roots` fields.
- Contained Instructions and Contexts are expected normal Workspace members.
- A Context member selects source content.
- A member selector selects declared Artifacts.
- Workspace has no global `include` or `exclude` fields.
- Workspace has no portable source scanner configuration field.
- Workspace has no Workspace-specific `overrides` or `use` behavior in v1.
- `overrides` and `use` are invalid in `workspace.members` until explicitly defined.
- Runtime budgets, selected runtime Artifacts, local disablement, and MCP installation data remain outside the Workspace declaration.

Changed from the previous contract:

- `declarations` is removed.
- `roots` is removed.
- `members` becomes the one Workspace composition field.
- Inline declaration sources become contained Workspace members.
- Source discovery directives are replaced by member selectors where the intent is dynamic Artifact membership.
- Context source-content patterns remain on Context.
- Workspace source-discovery mechanics remain outside the portable Workspace contract.

## 15. Contract validation requirements

All enhanced contracts must use strict validation.

### 15.1 General validation

- Every standalone declaration requires `type` and `name`.
- Every named or contained member requires `type` and `name`.
- Every member selector requires `type` and `base`.
- A member must match exactly one form:
  - Named external member.
  - Contained member.
  - Member selector.
- Unknown fields are invalid.
- `additionalProperties: false` is required except for explicitly typed maps such as:
  - `labels`
  - `metadata`
  - JSON Schema objects
  - MCP environment maps
  - MCP header maps
  - MCP installation input maps
  - MCP connection profile maps
  - MCP policy tool policy maps

### 15.2 Member form validation

Named external member:

```yaml
- type: skill
  name: reviewing-code
  locator: ./skills/reviewing-code
```

Contained member:

```yaml
- type: text
  name: local-rules
  insert: instructions
  parameters:
    content: |
      Keep changes focused.
```

Member selector:

```yaml
- type: skill
  base: ./skills
  nameInclude:
    - "review-*"
```

Invalid mixed form:

```yaml
# Invalid because a selector cannot also identify one name.
- type: skill
  name: reviewing-code
  base: ./skills
```

Invalid mixed form:

```yaml
# Invalid because a selector cannot contain a declaration body.
- type: skill
  base: ./skills
  parameters:
    license: Apache-2.0
```

Invalid flattened contained body:

```yaml
# Invalid because contained Instruction fields must be under `parameters`.
- type: text
  name: local-rules
  insert: instructions
  content: |
    Keep changes focused.
```

### 15.3 Metadata validation

`metadata` remains an opaque JSON annotation map.

It must not contain application-defined contract behavior.

Invalid pattern:

```yaml
metadata:
  flexigpt.site/mcp-runtime-v1:
    timeoutMS: 60000
```

Correct pattern:

```yaml
timeoutMS: 60000
```

### 15.4 Extension validation

A future Artifact type extension must not rely on:

```yaml
metadata:
  some.vendor/runtime-extension:
    # Contract behavior hidden here.
```

Instead, it must define:

```yaml
type: some.vendor.artifact

# Explicit named schema properties.
```

A future extension type must also explicitly declare:

- Which plural member fields accept it.
- Whether it supports contained declarations.
- Whether it supports member selectors.
- Which `overrides` fields are valid.
- Which `use` fields are valid.

## 16. Consolidated change commentary

| Previous contract area            | New contract                 | Motivation                                                                                             |
| --------------------------------- | ---------------------------- | ------------------------------------------------------------------------------------------------------ | --- |
| Declaration `$schema` property    | Removed from instances       | `$schema` belongs to the JSON Schema document, not the declaration instance                            |
| `apiVersion`                      | Removed                      | Current schema identity is already represented by immutable schema `$id`                               |
| Collection `version`              | Removed                      | No current artifact-release requirement justifies a Collection-only version field                      |
| `collection` type                 | Renamed to `plugin`          | Plugin better describes a reusable selected capability set                                             |
| Type-key nested members           | Removed                      | It introduced unnecessary nesting                                                                      |
| Generic member `type` field       | Retained as direct sibling   | Flat `type`, `name`, `locator`, `parameters`, `overrides`, and `use` are easier to author and validate |
| `target` wrapper                  | Removed                      | Workflow nodes can place target fields directly beside node fields                                     |
| Flat contained body fields        | Moved under `parameters`     | Separates target declaration body from relationship behavior                                           |
| `CompositionEntry` public concept | Removed                      | The wire contract uses direct member fields; no wrapper type is needed                                 |
| Model `parameters` map            | Removed                      | Supported portable Model settings should be named schema properties                                    |
| Tool `autoExecReco`               | Renamed to `autoExecute`     | One clear field name for target default behavior                                                       |
| MCP metadata runtime extension    | Flattened into MCP fields    | Runtime-required declaration configuration must be explicit and typed                                  |
| MCP policy `body`                 | Removed                      | Policy fields are direct properties of the policy Artifact                                             |
| MCP policy `toolName`             | Removed                      | The `toolPolicies` map key already identifies the Tool                                                 |
| Workspace `declarations`          | Removed                      | Source discovery does not belong as a separate portable Workspace declaration concern                  |
| Workspace `roots`                 | Replaced by `members`        | Workspace becomes a regular member-based collection contract                                           |
| `instructions` and `context`      | Replaced by `text`           | One text contract with explicit insertion identity replaces two near-duplicate contracts               |
| Directory-based Artifact groups   | Member selectors             | Supports reusable dynamic membership without naming every Artifact                                     |
| Source file text selection        | Text `include` and `exclude` | Keeps source content selection distinct from Artifact member selection                                 |     |

## 17. Implementation contract requirements

Implementation must update the portable contract layer to support this HLD.

Required contract-layer changes:

- Remove declaration-instance `$schema` from all schemas and headers.
- Remove declaration-instance `apiVersion` from all schemas and headers.
- Remove Collection-only `version`.
- Add common `displayName` and `labels`.
- Rename `collection` contract and schema to `plugin`.
- Replace legacy Collection references with Plugin references.
- Implement common member validation for:
  - Named external members.
  - Contained members.
  - Member selectors.
- Remove `instructions` and `context` contract types.
- Add `text` contract type with required direct `insert`.
- Add Text identity indexing by `(text, name, insert)`.
- Require contained declaration body fields under `parameters`.
- Reject flattened contained body fields in member positions.
- Add selector validation to all permitted plural member fields.
- Reject selectors in singular member positions.
- Replace Workflow node `target` wrappers with flat node member fields.
- Flatten MCP runtime declaration fields from metadata into direct MCP fields.
- Reject the retired MCP runtime metadata extension key.
- Flatten MCP policy fields and remove `body`.
- Remove generic Model `parameters`.
- Add named Model properties.
- Rename Tool `autoExecReco` to `autoExecute`.
- Represent Tool command execution through strict Tool implementation variants.
- Remove Workspace `declarations`.
- Remove Workspace `roots`.
- Add Workspace `members`.

Breaking contract changes are accepted while the feature remains under development.

No compatibility decoder, old-wire fallback, or automatic document migration is required by this HLD.
