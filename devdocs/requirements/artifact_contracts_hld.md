# Portable Artifact Declaration Contracts HLD

- [1. Purpose](#1-purpose)
- [2. Goals and scope](#2-goals-and-scope)
  - [2.1 Goals](#21-goals)
  - [2.2 In scope](#22-in-scope)
  - [2.3 Boundary](#23-boundary)
- [3. Requirements](#3-requirements)
  - [3.1 Declaration requirements](#31-declaration-requirements)
  - [3.2 Identity requirements](#32-identity-requirements)
  - [3.3 Composition requirements](#33-composition-requirements)
  - [3.4 Locator requirements](#34-locator-requirements)
  - [3.5 Validation requirements](#35-validation-requirements)
- [4. Design principles and invariants](#4-design-principles-and-invariants)
  - [4.1 One declaration has one semantic type](#41-one-declaration-has-one-semantic-type)
  - [4.2 Standalone declarations remain flat](#42-standalone-declarations-remain-flat)
  - [4.3 Member relationships remain flat](#43-member-relationships-remain-flat)
  - [4.4 Contained target bodies are separated from relationships](#44-contained-target-bodies-are-separated-from-relationships)
  - [4.5 Containment classification is structural](#45-containment-classification-is-structural)
  - [4.6 Locators are forward-compatible](#46-locators-are-forward-compatible)
  - [4.7 Metadata is annotation only](#47-metadata-is-annotation-only)
  - [4.8 Composition does not imply ownership](#48-composition-does-not-imply-ownership)
- [5. Declaration workflows](#5-declaration-workflows)
  - [5.1 Normalization workflow](#51-normalization-workflow)
  - [5.2 Standalone authoring workflow](#52-standalone-authoring-workflow)
  - [5.3 External relationship workflow](#53-external-relationship-workflow)
  - [5.4 Contained declaration workflow](#54-contained-declaration-workflow)
  - [5.5 Dynamic membership workflow](#55-dynamic-membership-workflow)
- [6. Artifact vocabulary and organization](#6-artifact-vocabulary-and-organization)
  - [6.1 Capability and configuration contracts](#61-capability-and-configuration-contracts)
  - [6.2 Composition and structure contracts](#62-composition-and-structure-contracts)
- [7. Schema identity and declaration versioning](#7-schema-identity-and-declaration-versioning)
- [8. Common declaration model](#8-common-declaration-model)
  - [8.1 Common header](#81-common-header)
  - [8.2 Portable names](#82-portable-names)
  - [8.3 Declaration occurrence](#83-declaration-occurrence)
- [9. Locator contract](#9-locator-contract)
  - [9.1 Locator union](#91-locator-union)
  - [9.2 Scalar locator](#92-scalar-locator)
  - [9.3 Path locator](#93-path-locator)
  - [9.4 URL locator](#94-url-locator)
  - [9.5 Git locator](#95-git-locator)
  - [9.6 Package locator](#96-package-locator)
  - [9.7 Command locator](#97-command-locator)
  - [9.8 Relative locator interpretation](#98-relative-locator-interpretation)
  - [9.9 Context-dependent meaning](#99-context-dependent-meaning)
- [10. Path pattern contract](#10-path-pattern-contract)
- [11. Common member relationship model](#11-common-member-relationship-model)
  - [11.1 Wire shape](#111-wire-shape)
  - [11.2 Named external member](#112-named-external-member)
  - [11.3 Located external member](#113-located-external-member)
  - [11.4 Contained member](#114-contained-member)
  - [11.5 Relationship fields](#115-relationship-fields)
  - [11.6 Text insertion identity](#116-text-insertion-identity)
  - [11.7 MCP source selector](#117-mcp-source-selector)
- [12. Member selectors](#12-member-selectors)
  - [12.1 Selector shape](#121-selector-shape)
  - [12.2 Selector rules](#122-selector-rules)
  - [12.3 Selector base](#123-selector-base)
  - [12.4 Text patterns are not selectors](#124-text-patterns-are-not-selectors)
- [13. Relationship eligibility](#13-relationship-eligibility)
- [14. Capability and configuration contracts](#14-capability-and-configuration-contracts)
  - [14.1 Text](#141-text)
  - [14.2 Model](#142-model)
  - [14.3 Tool](#143-tool)
  - [14.4 Skill](#144-skill)
  - [14.5 MCP](#145-mcp)
  - [14.6 MCP policy](#146-mcp-policy)
- [15. Composition and structure contracts](#15-composition-and-structure-contracts)
  - [15.1 Plugin](#151-plugin)
  - [15.2 Agent](#152-agent)
  - [15.3 Team](#153-team)
  - [15.4 Output matching](#154-output-matching)
  - [15.5 Loop](#155-loop)
  - [15.6 Workflow](#156-workflow)
  - [15.7 Workspace](#157-workspace)
- [16. Validation and canonicalization](#16-validation-and-canonicalization)
  - [16.1 Strict schemas](#161-strict-schemas)
  - [16.2 Member-form validation](#162-member-form-validation)
  - [16.3 Parameters opacity and effective validation](#163-parameters-opacity-and-effective-validation)
  - [16.4 Duplicate and ordering validation](#164-duplicate-and-ordering-validation)
  - [16.5 Metadata validation](#165-metadata-validation)
  - [16.6 Legacy wire forms](#166-legacy-wire-forms)
- [17. Extension model](#17-extension-model)
- [18. Implementation structure](#18-implementation-structure)
- [19. Current contract status](#19-current-contract-status)

## 1. Purpose

This HLD defines the portable declaration model for AI and LLM-related Artifacts.

The contracts provide one normalized vocabulary for Artifacts authored in canonical JSON, canonical YAML, familiar repository files, managed packages, and physical-format adapters.

The declaration flow is:

```text
Physical source format
  -> physical-format decoder
  -> normalized portable declaration
  -> strict contract validation
  -> source-backed Artifact
```

This document defines the normalized portable declaration contract. Resolution, persistence, discovery, managed authoring, and runtime integration are defined by the companion Store and Ecosystem HLD.

## 2. Goals and scope

### 2.1 Goals

The contract system must provide:

- One concrete portable `type` for every Artifact.
- One common standalone declaration header.
- Flat and readable standalone declarations.
- One common member grammar for composition relationships.
- Named external, located external, contained, and selector member forms.
- Explicit separation between target-owned and relationship-owned fields.
- Typed Artifact behavior instead of hidden runtime metadata extensions.
- Strict schemas with predictable extension points.
- Portable locator representation that is not limited to local paths.
- Stable semantic identity independent of source-array position.
- Dynamic membership through member selectors.
- A single Text contract for behavioral and informational source text.
- Workspace composition through ordinary `members`.
- Schema organization that remains independent of Artifact Store persistence.

### 2.2 In scope

This HLD defines:

- Artifact type vocabulary.
- Schema identity and declaration versioning policy.
- Common declaration fields.
- Semantic identity.
- Portable locators.
- Path patterns.
- Member forms and relationship fields.
- Member selectors.
- Type-specific declaration contracts.
- Contract validation and extension rules.
- Current contract implementation structure.

### 2.3 Boundary

These contracts describe declarations and relationships. They do not determine whether a target can be found, installed, materialized, connected, scheduled, or executed.

## 3. Requirements

### 3.1 Declaration requirements

- Every standalone declaration must contain exactly one supported `type`.
- Every standalone declaration must contain a portable `name`.
- Every contained declaration must have an outer `type` and `name`.
- Anonymous declarations are not supported.
- A second portable discriminator such as `kind`, `category`, or `role` must not be required.
- Standalone target-owned fields must remain flat.
- Application behavior must use typed properties rather than reserved metadata extensions.
- Physical-format adapters must normalize into these contracts rather than define alternate portable headers.
- Unknown fields must be rejected except within explicitly open typed maps.
- Portable declarations must not contain local installation values, secrets, enablement, runtime state, or Store identity.

### 3.2 Identity requirements

The default semantic identity is:

```text
(type, name)
```

Text uses the more specific identity:

```text
(text, name, insert)
```

Semantic identity is scoped by the Root that contains the Artifact occurrence.

The contract must preserve the distinction between:

- Semantic identity, which identifies what the declaration represents.
- Declaration occurrence, which identifies where the declaration is physically or structurally written.

Release versions and schema versions are not part of default symbolic identity.

### 3.3 Composition requirements

Plural composition fields must support, where allowed by the containing contract:

- Named external members.
- Located external members.
- Contained declarations.
- Member selectors.

Singular composition fields support one named or contained member and do not support selectors.

The presence of `parameters` determines that a member is contained. This classification must not depend on inspecting the payload inside `parameters`.

### 3.4 Locator requirements

- The shared `Locator` contract must remain capable of representing path, URL, Git, package, command, and scalar locators.
- `locator` must not be globally narrowed to a local path.
- A concrete Artifact contract or resolver may constrain how a locator is interpreted in a particular position.
- Contract acceptance of a locator kind must remain separate from whether an implementation currently has a materializer for that kind.
- Relative path locators must be interpreted relative to the declaration occurrence, not the process working directory.

### 3.5 Validation requirements

- JSON Schema Draft 2020-12 is the portable schema dialect.
- The original document bytes must be schema-validated before decoding into implementation structures.
- Unknown outer member fields must be invalid.
- A member must match exactly one member form.
- Exact duplicate normalized members in one containing field must be invalid.
- Duplicate contained declaration identities in one containing field must be invalid.
- Source-array position must not define relationship or contained declaration identity.

## 4. Design principles and invariants

### 4.1 One declaration has one semantic type

```yaml
type: skill
name: reviewing-code
```

There is no second semantic type field.

### 4.2 Standalone declarations remain flat

A Model, Tool, MCP server, Agent, or other standalone Artifact owns its fields directly.

```yaml
type: model
name: reasoning
model: anthropic/claude-sonnet
temperature: 0.2
```

A standalone declaration does not wrap its body in `parameters`.

### 4.3 Member relationships remain flat

Member identity, target selection, and relationship behavior are siblings.

```yaml
- type: tool
  name: searchfiles
  locator: ./tools/searchfiles.yaml
  overrides:
    autoExecute: true
```

There is no artifact-type-key wrapper and no generic `target` wrapper.

### 4.4 Contained target bodies are separated from relationships

A contained target body is placed under `parameters`.

```yaml
- type: text
  name: repository-rules
  insert: instructions
  parameters:
    mediaType: text/markdown
    content: |
      Keep changes focused.
```

The containing relationship retains `type`, `name`, optional `locator`, relationship `overrides`, and relationship `use`.

### 4.5 Containment classification is structural

The generic member layer classifies a member from its outer fields:

- Outer `base` means selector.
- Outer `parameters` means contained declaration.
- Otherwise, outer `name` means named external relationship.

The generic classifier deliberately does not inspect `parameters` for fields such as `type`, `name`, `locator`, `insert`, or `base`.

When a contained declaration is projected:

- The payload is read from `parameters`.
- Outer `type`, `name`, and `locator` are authoritative.
- For Text, outer `insert` is authoritative.
- Same-named values inside `parameters` do not alter outer identity.
- Other payload fields are validated by the concrete target contract.

There is intentionally no separate generic validation error merely because `parameters` repeats an outer field. This keeps containment and lifecycle classification independent of future target-body evolution.

Authors should still place identity and relationship fields outside `parameters`. The lack of a duplicate-key precheck is not an alternate authoring convention.

### 4.6 Locators are forward-compatible

The shared locator union is broader than the currently available Source adapters.

A path-only implementation must not replace the portable locator contract with a path-only wire shape. Unsupported external locator materialization is an ecosystem capability concern, not a reason to remove the locator kind from the declaration model.

### 4.7 Metadata is annotation only

`metadata` is an opaque user annotation map.

Application behavior must not be hidden in it. In particular, the following retired extension is invalid:

```text
flexigpt.site/mcp-runtime-v1
```

### 4.8 Composition does not imply ownership

Plugin, Agent, Team, Loop, Workflow, Workspace, and Skill relationships describe selection and structure. They do not declare Store ownership, cascading deletion, or target mutation.

## 5. Declaration workflows

### 5.1 Normalization workflow

```text
Repository or managed source
  -> decoder selected from physical format
  -> canonical JSON declaration
  -> concrete type schema
  -> structural validation
  -> normalized declaration entry
```

Schema dispatch is an implementation concern. Declaration instances do not carry schema-dispatch fields.

### 5.2 Standalone authoring workflow

A standalone Artifact:

1. Declares `type` and `name`.
2. Adds optional common descriptive fields.
3. Adds its direct type-specific body.
4. Uses `locator` only according to that type's semantics.
5. Is validated by its concrete versioned schema.

### 5.3 External relationship workflow

A member without `parameters` or `base` identifies an external target.

```yaml
- type: skill
  name: reviewing-code
```

Adding `locator` narrows the expected occurrence.

```yaml
- type: skill
  name: reviewing-code
  locator: ./skills/reviewing-code
```

The locator does not create a copy or a contained target.

### 5.4 Contained declaration workflow

A member with `parameters` declares a contained target.

```text
Outer member identity
  + parameters payload
  -> effective standalone target declaration
  -> concrete target validation
  -> stable contained declaration occurrence
```

An empty object still establishes the contained form:

```yaml
- type: agent
  name: local-agent
  parameters: {}
```

It is valid only if the resulting concrete target contract permits that body.

### 5.5 Dynamic membership workflow

A member with `base` is a selector.

```yaml
- type: skill
  base: ./skills
  include:
    - "**/SKILL.md"
  nameInclude:
    - "review-*"
```

The contract describes the selector. Source inspection and selector expansion are resolver and Source concerns.

## 6. Artifact vocabulary and organization

The portable Artifact vocabulary is:

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

The contracts are organized for documentation and schema reuse as follows.

### 6.1 Capability and configuration contracts

```text
text
model
tool
skill
mcp
mcp.policy
```

These contracts describe independently meaningful content, capabilities, or policy.

A capability contract may still contain a relationship field. For example, `skill.allowedTools` uses the common member grammar.

### 6.2 Composition and structure contracts

```text
plugin
agent
team
loop
workflow
workspace
```

These contracts select, group, or structurally organize other Artifacts.

This organization is not represented by another portable declaration field.

## 7. Schema identity and declaration versioning

Every JSON Schema document contains schema metadata at the schema root.

```yaml
$schema: https://json-schema.org/draft/2020-12/schema
$id: https://schemas.flexigpt.site/artifact/model/v1.json
```

Declaration instances do not contain:

```text
$schema
apiVersion
schemaID
schemaVersion
version
digest
logicalName
logicalVersion
```

Rules:

- `$schema` identifies the dialect of the JSON Schema document.
- `$id` identifies an immutable versioned schema document.
- Schema versioning belongs to schema registration and dispatch.
- No common Artifact release `version` exists in the current contract.
- A future Artifact release version must be distinct from schema version.
- If simultaneous portable schema versions eventually require in-document dispatch, one explicit `schemaVersion` may be introduced at that time.
- A type-specific version field must not be introduced merely for one collection contract.

Representative schema identifiers are:

```text
https://schemas.flexigpt.site/artifact/text/v1.json
https://schemas.flexigpt.site/artifact/model/v1.json
https://schemas.flexigpt.site/artifact/mcp/v1.json
https://schemas.flexigpt.site/artifact/plugin/v1.json
https://schemas.flexigpt.site/artifact/workspace/v1.json
```

## 8. Common declaration model

### 8.1 Common header

Every standalone declaration uses this conceptual header:

```yaml
type: <artifact-type>
name: <portable-name>

displayName: <optional-display-name>
description: <optional-description>

labels:
  <label-key>: <string-value>

locator: <optional-locator>

metadata:
  <annotation-key>: <json-value>
```

Field ownership:

| Field         | Meaning                                                             |
| ------------- | ------------------------------------------------------------------- |
| `type`        | Concrete portable Artifact type                                     |
| `name`        | Portable logical name                                               |
| `displayName` | Source-authored display text                                        |
| `description` | Source-authored descriptive text                                    |
| `labels`      | Source-authored string classifications                              |
| `locator`     | Type-specific source, package, implementation, or selection locator |
| `metadata`    | Opaque user annotation values                                       |

A concrete type may require, forbid, or constrain `locator`.

### 8.2 Portable names

Portable names:

- Are required for every concrete declaration.
- Are stable logical identifiers.
- Are not display labels.
- May be derived deterministically by a physical-format adapter when the physical format has no explicit name.
- Remain Root-scoped during resolution.

### 8.3 Declaration occurrence

A declaration occurrence is identified by:

```text
Root
  -> Source
  -> declaration locator
  -> optional stable structural subresource
```

One Root may contain multiple occurrences with the same semantic identity. That is a resolution ambiguity concern, not a reason to alter the declaration name.

## 9. Locator contract

### 9.1 Locator union

A portable locator is:

```text
Locator =
  scalar string
  | path Locator
  | URL Locator
  | Git Locator
  | package Locator
  | command Locator
```

### 9.2 Scalar locator

A scalar can represent a portable relative path or a non-file URI.

```yaml
locator: ./skills/reviewing-code
```

```yaml
locator: https://example.com/artifacts/reviewer.yaml
```

A `file:` URI is not portable and is invalid.

### 9.3 Path locator

```yaml
locator:
  kind: path
  path: ./skills/reviewing-code
```

### 9.4 URL locator

```yaml
locator:
  kind: url
  url: https://example.com/artifacts/reviewer.yaml
  integrity: sha256:...
```

The URL must be absolute and must not use the `file` scheme.

### 9.5 Git locator

```yaml
locator:
  kind: git
  repository: https://github.com/acme/agent-assets.git
  revision: v2.0.0
  path: skills/reviewing-code
```

`revision` and `path` are optional. A supplied repository URL must be absolute.

### 9.6 Package locator

```yaml
locator:
  kind: package
  manager: npm
  package: "@acme/reviewing-code"
  version: 2.0.0
  path: artifacts/skill
  registry: https://registry.npmjs.org
```

Current package-manager values are:

```text
npm
pypi
cargo
go
maven
nuget
oci
```

### 9.7 Command locator

```yaml
locator:
  kind: command
  command: grep
```

The command locator remains part of the portable locator union.

A type-specific contract may instead require command information in a typed implementation or transport body. That contextual rule does not remove command locators from the common representation.

### 9.8 Relative locator interpretation

Relative path locators are interpreted relative to the declaration occurrence.

For example:

```text
plugins/repository-review.yaml
  locator: ../skills/reviewing-code
```

resolves relative to:

```text
plugins/
```

It does not resolve relative to the process working directory.

### 9.9 Context-dependent meaning

| Context                     | Locator meaning                                                 |
| --------------------------- | --------------------------------------------------------------- |
| Standalone Text             | Source file or source directory                                 |
| Standalone Skill            | Skill package or source                                         |
| Source-selected declaration | Source containing the expected target                           |
| External member             | Constraint selecting the expected declaration occurrence        |
| Contained declaration       | Target-owned locator when that concrete contract permits it     |
| Other standalone types      | Type-specific source, package, alias, or implementation meaning |

Contract support does not imply that every locator kind is currently materializable.

## 10. Path pattern contract

Path patterns use slash-separated portable syntax.

```text
*       zero or more characters in one path segment
?       one character in one path segment
**      zero or more path segments
[abc]   one character from a character class
```

Example:

```yaml
include:
  - "docs/**/*.md"
  - "src/**/*"

exclude:
  - "**/generated/**"
  - "vendor/**"
```

Rules:

- Patterns are relative to the field's declared base.
- Inclusion is evaluated before exclusion.
- Pattern arrays are semantically unordered within one field.
- Text source-resource patterns and member-selector declaration patterns use similar syntax but have different meanings.

## 11. Common member relationship model

### 11.1 Wire shape

A member uses direct sibling fields.

```yaml
- type: <artifact-type>
  name: <portable-name>
  locator: <optional-locator>
  scope: <optional-lookup-scope>
  insert: <text-insertion-identity>
  server: <optional-mcp-source-selector>
  parameters: <optional-contained-target-body>
  overrides: <optional-relationship-behavior>
  use: <optional-container-behavior>
```

Selectors replace `name`, `locator`, `scope`, and `parameters` with `base` and optional filters.

There is no public `CompositionEntry` wrapper and no `target` wrapper.

### 11.2 Named external member

```yaml
- type: skill
  name: reviewing-code
```

This identifies an expected external target by semantic identity.

A named external member may request protected built-in lookup:

```yaml
- type: tool
  name: searchfiles
  scope: builtin
```

`scope`:

- Is valid only for a named external member.
- Currently supports only `builtin`.
- Cannot be combined with `locator`.
- Is not a Root ID or cross-Root import.
- Is not valid for contained members or selectors.

### 11.3 Located external member

```yaml
- type: skill
  name: reviewing-code
  locator: ./skills/reviewing-code
```

A located member:

- Remains an external relationship.
- Selects the expected occurrence at the locator.
- Does not create a copy.
- Does not merge omitted fields from the containing declaration.
- Does not override target configuration through target-body fields.
- Does not fall back to an unlocated target when its selected occurrence is unavailable.

### 11.4 Contained member

```yaml
- type: text
  name: local-rules
  insert: instructions
  locator: ./AGENTS.md
  parameters:
    displayName: Local Rules
    description: Rules declared by the containing Artifact.
    labels:
      scope: local
    mediaType: text/markdown
```

Rules:

- Presence of `parameters` establishes containment.
- `parameters` must be an object.
- `parameters` is not a runtime argument bag.
- `parameters` is not a provider-extension map.
- `parameters` is the target declaration payload.
- `type`, `name`, and optional `locator` remain outside.
- Text `insert` remains outside.
- Relationship `overrides` and `use` remain outside.
- Common target fields such as `displayName`, `description`, `labels`, and `metadata` normally belong inside.
- Type-specific target body fields belong inside.
- Flattened target body fields beside `parameters` are invalid.
- `params` is not an alias for `parameters`.
- `{}` is valid only when the resulting concrete target contract is valid.
- Outer identity fields overwrite same-named values while constructing the effective target.
- The generic member layer does not reject duplicated outer names inside the opaque payload.
- Fields remaining after projection are validated by the target's concrete schema.

A locator-bearing member with `parameters` is contained because `parameters`, not locator kind, determines the form.

### 11.5 Relationship fields

```yaml
- type: tool
  name: searchfiles
  overrides:
    autoExecute: true
```

```yaml
- type: skill
  name: reviewing-code
  use:
    mode: active
```

Ownership is:

| Field         | Owner                            | Meaning                                              |
| ------------- | -------------------------------- | ---------------------------------------------------- |
| `type`        | Relationship                     | Expected target type                                 |
| `name`        | Relationship                     | Expected target name                                 |
| `locator`     | Relationship or contained target | External occurrence selector or target-owned locator |
| `scope`       | Relationship                     | Named lookup scope                                   |
| `parameters`  | Contained target                 | Target declaration payload                           |
| `overrides`   | Relationship                     | Target-type-specific behavior for this relationship  |
| `use`         | Containing contract              | Container-specific use of this relationship          |
| `base`        | Selector                         | Candidate declaration directory                      |
| `include`     | Selector                         | Declaration-source path inclusions                   |
| `exclude`     | Selector                         | Declaration-source path exclusions                   |
| `nameInclude` | Selector                         | Logical-name inclusions                              |
| `nameExclude` | Selector                         | Logical-name exclusions                              |

`overrides` and `use` never mutate:

- Target metadata.
- Target definition.
- Target source content.
- Target installation data.
- Target enablement.
- Target policy.
- Target runtime state.
- Membership in another composition.

### 11.6 Text insertion identity

Text requires one direct insertion target:

```text
instructions
user-message
```

`insert` is required for:

- Standalone Text.
- Named external Text members.
- Located external Text members.
- Contained Text members.

It remains outside contained `parameters`.

A containing contract cannot change the selected Text target through `overrides` or `use`.

### 11.7 MCP source selector

A located MCP relationship may identify one server in a multi-server source.

```yaml
- type: mcp
  name: github
  locator: ./.mcp.json
  server: github
```

`server`:

- Is valid only for `type: mcp`.
- Requires a locator-selected external MCP relationship.
- Selects one server from the source.
- Is not a contained MCP body field.
- Is not valid inside `parameters`.

## 12. Member selectors

### 12.1 Selector shape

```yaml
- type: skill
  base: ./skills

  include:
    - "**/SKILL.md"

  exclude:
    - "**/experimental/**"

  nameInclude:
    - "review-*"
    - "crud-*"

  nameExclude:
    - "legacy-*"
```

### 12.2 Selector rules

- `type` is required.
- `base` is required.
- A selector does not contain outer `name`.
- A selector does not contain outer `locator`.
- A selector does not contain `scope`.
- A selector does not contain `parameters`.
- A selector is valid only in plural member fields.
- `include` and `exclude` match declaration-source paths relative to `base`.
- `nameInclude` and `nameExclude` match decoded logical names.
- Omitted `include` means all candidate declarations of the selected type below `base`.
- `exclude` is applied after `include`.
- `nameExclude` is applied after `nameInclude`.
- A selector may contain relationship behavior allowed by the containing field.
- A zero-match selector is a valid empty dynamic membership set.
- Text and Workspace do not support selectors.
- Selector eligibility remains type-specific.

### 12.3 Selector base

`base` uses the portable Locator representation so its wire shape can evolve consistently. In the current contract it is restricted to:

- A scalar local relative path.
- A structured `kind: path` local relative path.

This current selector restriction must not be generalized into a restriction on ordinary `locator`.

External URL, Git, package, or remote selector bases require corresponding Source and discovery support before they can be enabled.

### 12.4 Text patterns are not selectors

```yaml
- type: text
  name: application-source
  insert: user-message
  locator: ./src
  parameters:
    include:
      - "**/*.go"
```

This selects files as Text content.

```yaml
- type: skill
  base: ./skills
  include:
    - "**/SKILL.md"
```

This selects Skill declarations as members.

The two operations must not be conflated.

## 13. Relationship eligibility

| Containing field     | Allowed target types                                                     | Named | Contained |       Selector | Relationship behavior           |
| -------------------- | ------------------------------------------------------------------------ | ----: | --------: | -------------: | ------------------------------- |
| `plugin.members`     | Mixed supported types                                                    |   Yes |       Yes | Type-dependent | None in v1                      |
| `agent.members`      | `text`, `model`, `skill`, `tool`, `mcp`, `mcp.policy`, `plugin`, `agent` |   Yes |       Yes | Type-dependent | Model, Tool, and Skill behavior |
| `team.members`       | `agent`, `plugin`                                                        |   Yes |       Yes | Type-dependent | None in v1                      |
| `workspace.members`  | Mixed supported types                                                    |   Yes |       Yes | Type-dependent | None in v1                      |
| `skill.allowedTools` | `tool`                                                                   |   Yes |       Yes |            Yes | None in v1                      |
| `loop.body`          | One contract-allowed target                                              |   Yes |       Yes |             No | None in v1                      |
| `workflow.nodes[]`   | One contract-allowed target                                              |   Yes |       Yes |             No | None in v1                      |
| `agent.loop`         | `loop`                                                                   |   Yes |       Yes |             No | None in v1                      |
| `agent.workflow`     | `workflow`                                                               |   Yes |       Yes |             No | None in v1                      |
| `team.loop`          | `loop`                                                                   |   Yes |       Yes |             No | None in v1                      |
| `team.workflow`      | `workflow`                                                               |   Yes |       Yes |             No | None in v1                      |

Current Agent relationship behavior is:

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

Skill use modes are:

```text
available
active
instructions
```

A selector in `agent.members` may apply the permitted behavior to every selected target.

Unknown `overrides` or `use` fields are invalid. Contracts with no defined behavior reject non-empty `overrides` and `use`.

## 14. Capability and configuration contracts

### 14.1 Text

Text represents inline or source-backed text.

Fields beyond the common header are:

```text
insert
content?
mediaType?
include?
exclude?
```

Rules:

- `insert` is required.
- `insert` is `instructions` or `user-message`.
- Exactly one content source is required:
  - Inline content through `content`.
  - Source-backed content through `locator`.
- Inline `content` cannot be combined with `locator`, `include`, or `exclude`.
- A file locator contributes one file.
- A directory locator may use `include` and `exclude`.
- File-versus-directory verification occurs when source resources are inspected.
- Text does not use selector `base`.
- Text identity is `(text, name, insert)`.

Inline example:

```yaml
type: text
name: repository-rules
insert: instructions
displayName: Repository Rules
description: Rules for repository changes.
mediaType: text/markdown
content: |
  Do not modify generated files.
  Keep changes focused.
```

Source-backed example:

```yaml
type: text
name: application-source
insert: user-message
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

Physical-format adapters commonly normalize:

```text
AGENTS.md and CLAUDE.md
  -> Text with insert=instructions

README.md, llms.txt, and selected documentation
  -> Text with insert=user-message
```

### 14.2 Model

Model represents one provider-qualified model and its portable defaults.

Fields are:

```text
model
systemPrompt?
temperature?
maxOutputTokens?
```

Rules:

- `model` is required.
- `temperature`, when present, is between `0` and `2`.
- `maxOutputTokens`, when present, is positive.
- There is no generic provider `parameters`, `options`, or extension map.
- New portable Model settings require explicit contract fields.
- A contained Model still uses the member-level `parameters` field because that field represents the complete contained target body.

```yaml
type: model
name: reasoning
displayName: Reasoning Model
model: anthropic/claude-sonnet

systemPrompt: |
  Be precise, grounded, and concise.

temperature: 0.2
maxOutputTokens: 12000
```

### 14.3 Tool

Tool represents one callable operation.

Fields are:

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

Rules:

- `userCallable`, `llmCallable`, and `autoExecute` are explicit Tool-owned defaults.
- `llmToolType` is currently `function`.
- `inputSchema` is required.
- `userArgSchema` and `outputSchema` are optional.
- Schema-valued fields must contain valid JSON Schema objects or booleans.
- `implementation` is a strict discriminated union.
- A Tool declaration does not use header `locator` for its implementation.
- Command execution is represented by `implementation.kind: command`.
- Relationship `overrides.autoExecute` does not mutate Tool-owned `autoExecute`.

```yaml
type: tool
name: searchfiles
displayName: Search Files
description: Searches repository files.

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

outputSchema:
  type: object
  properties:
    matches:
      type: array
      items:
        type: string

implementation:
  kind: go
  function: github.com/flexigpt/flexigpt/tools.SearchFiles
```

Supported implementations are:

```yaml
implementation:
  kind: go
  function: github.com/flexigpt/flexigpt/tools.SearchFiles
```

```yaml
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
implementation:
  kind: sdk
  provider: example
  operation: search
```

```yaml
implementation:
  kind: command
  command: rg
  args:
    - "--json"
  env:
    NO_COLOR: "1"
```

HTTP response output modes are:

```text
json
text
bytes
```

### 14.4 Skill

Skill represents one reusable package-backed Skill capability.

Fields are:

```text
license?
insert?
allowedTools?
```

Rules:

- A concrete Skill requires `locator`.
- The locator identifies the Skill package or source and retains the full portable Locator shape.
- `insert`, when present, uses a valid Text insertion target.
- `allowedTools` contains only Tool members.
- `allowedTools` supports named, contained, and selector Tool members.
- Skill package scripts, invocation arguments, session state, and runtime registration are not portable Skill fields.

```yaml
type: skill
name: reviewing-code
displayName: Reviewing Code
description: Reviews code changes for correctness and risk.
locator: ./skills/reviewing-code
license: Apache-2.0
insert: instructions

allowedTools:
  - type: tool
    name: searchfiles

  - type: tool
    base: ./tools
    include:
      - "**/*.yaml"
      - "**/*.yml"
      - "**/*.json"
    nameInclude:
      - "repository-*"
```

### 14.5 MCP

MCP represents one MCP server capability.

MCP owns portable declarations for:

- Transport and connection configuration.
- Capability inclusion.
- Timeout.
- Authentication requirements.
- Installation input definitions.
- Available connection profiles.
- Optional default MCP policy reference.

MCP does not own:

- Installation input values.
- Secret values.
- Secret references.
- OAuth tokens.
- Selected connection profile.
- Additional local policy.
- Connection state.
- Generic Artifact enablement.

Fields include:

```text
server?
transport?
command?
args?
env?
url?
headers?
include?
timeoutMS?
auth?
install?
connectionProfiles?
policy?
```

Current transport values are:

```text
stdio
streamableHTTP
```

`sse` is not a current v1 transport value.

Inline stdio example:

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

Inline HTTP example:

```yaml
type: mcp
name: github
displayName: GitHub
description: GitHub MCP server.

transport: streamableHTTP
url: https://api.githubcopilot.com/mcp/

headers:
  X-Client-Mode: default

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

install:
  note: GitHub OAuth client credentials may be configured locally.
  inputs:
    GITHUB_OAUTH_CLIENT:
      kind: oauthClientCredentials
      label: GitHub OAuth Client
      required: false
      clientSecretRequired: false

connectionProfiles:
  local:
    platforms:
      - darwin
      - linux
    http:
      headers:
        X-Client-Mode: local

policy:
  name: builtin-manual-write
  required: true
```

Connection rules:

- Inline stdio requires `transport: stdio` and `command`.
- Inline HTTP requires `transport: streamableHTTP` and `url`.
- Stdio cannot contain HTTP connection fields.
- HTTP cannot contain stdio connection fields.
- Connection fields require an explicit transport.
- A source-selected MCP uses `locator` and optional `server`.
- A source-selected MCP cannot combine locator selection with inline connection or local MCP configuration fields.
- Locator kind remains represented by the common Locator contract. Resolver support determines how a selected locator is materialized.
- Omitted `include` selects the complete server capability.
- A present `include` limits tools, resources, or prompts.

Authentication modes are:

```text
none
apiKey
oauth
clientCredentials
```

Installation input kinds are:

```text
text
secret
path
oauthClientCredentials
```

Installation input declarations may contain:

```text
label
description
note
placeholder
required
default
clientSecretRequired
```

Rules:

- Secret and OAuth-client-credential inputs cannot contain defaults.
- `clientSecretRequired` is valid only for `oauthClientCredentials`.
- `allowEnvironment` declares allowed environment names, not values.

A connection profile:

- May list `linux`, `darwin`, or `windows`.
- Must contain exactly one of `stdio` or `http`.
- May add or remove environment variables or headers.
- Declares available profiles but does not select one locally.

The direct policy relationship is:

```yaml
policy:
  name: builtin-manual-write
  required: true
```

There is no `ref` wrapper.

### 14.6 MCP policy

MCP policy represents portable MCP policy rules.

Fields are:

```text
trustLevel?
defaultPolicy?
toolPolicies?
appsPolicy?
```

Policy fields are direct declaration fields. There is no `body` wrapper.

```yaml
type: mcp.policy
name: builtin-manual-write
displayName: Built-in Manual Write Policy

trustLevel: trusted

defaultPolicy:
  defaultApprovalRule: allow
  defaultExecutionMode: manual
  requireApprovalForUnknownRisk: true
  requireApprovalForWrite: true
  requireApprovalForDestructive: true

toolPolicies:
  delete_issue:
    approvalRule: ask
    executionMode: manual
    allowStaleDigest: false
    expectedDigest: sha256:...

appsPolicy:
  enabled: false
  allowAppInitiatedToolCalls: false
  requireApprovalForOpenLink: true
  requireApprovalForContextUpdates: true
```

Validation rules:

- `trustLevel` is `trusted` or `untrusted`.
- Approval rules are `allow`, `ask`, or `deny`.
- Execution modes are `auto` or `manual`.
- `toolPolicies` keys identify MCP tool names.
- A nested `toolName` is invalid.
- `expectedDigest`, when present, is a valid SHA-256 digest.
- A locator-selected policy cannot combine the locator with local policy fields.
- Policy composition and additional local policy state are consumer concerns.

## 15. Composition and structure contracts

### 15.1 Plugin

Plugin represents a reusable selected Artifact group.

Fields are:

```text
members?
```

```yaml
type: plugin
name: engineering-capabilities
displayName: Engineering Capabilities
description: Shared engineering capabilities.

members:
  - type: text
    name: repository-source
    insert: user-message
    locator: ./src
    parameters:
      mediaType: text/plain
      include:
        - "**/*.go"
        - "**/*.ts"
      exclude:
        - "**/generated/**"

  - type: skill
    base: ./skills
    include:
      - "**/SKILL.md"
    nameInclude:
      - "review-*"

  - type: mcp
    name: github
    locator: ./.mcp.json
    server: github
```

Rules:

- Members may contain mixed supported Artifact types.
- Members may be named, contained, or selectors where the selected type supports selectors.
- Plugin has no `overrides` or `use` behavior in v1.
- Plugin does not own or mutate selected targets.
- `locator` may select another Plugin declaration only when no local `members` body is present.

### 15.2 Agent

Agent represents an AI Agent declaration.

Fields are:

```text
members?
loop?
workflow?
```

At most one of `loop` and `workflow` is allowed.

```yaml
type: agent
name: code-reviewer
displayName: Code Reviewer

members:
  - type: text
    name: reviewer-rules
    insert: instructions
    parameters:
      mediaType: text/markdown
      content: |
        Review correctness, security, and maintainability.

  - type: text
    name: review-request
    insert: user-message
    parameters:
      mediaType: text/markdown
      content: |
        Please review the available repository.

  - type: model
    name: reasoning
    overrides:
      includeSystemPrompt: true

  - type: skill
    base: ./skills
    nameInclude:
      - "review-*"
    use:
      mode: active

  - type: tool
    name: searchfiles
    overrides:
      autoExecute: true

loop:
  name: reviewer-loop
  locator: ./loops/reviewer-loop.yaml
```

Rules:

- `members` accepts the Agent member types in the relationship table.
- Model, Tool, and Skill relationship behavior is explicitly typed.
- `loop` and `workflow` are singular program slots.
- Program slots do not accept selectors.
- A minimal standalone Agent with only `type` and `name` is valid.
- `locator` cannot be combined with local `members`, `loop`, or `workflow`.

### 15.3 Team

Team represents a multi-Agent composition.

Fields are:

```text
members?
loop?
workflow?
```

At most one of `loop` and `workflow` is allowed.

```yaml
type: team
name: change-team
displayName: Change Team

members:
  - type: agent
    base: ./agents
    nameInclude:
      - "change-*"

  - type: agent
    name: reviewer
    locator: ./agents/reviewer.agent.yaml

  - type: plugin
    name: repository-common

workflow:
  name: change-workflow
  locator: ./workflows/change-workflow.yaml
```

Rules:

- Team members are Agents or Plugins.
- Members may be named, contained, or selectors where supported.
- Team has no `overrides` or `use` behavior in v1.
- Program slots are singular and do not accept selectors.
- `locator` cannot be combined with local members or program fields.

### 15.4 Output matching

Loop and Workflow conditions use:

```text
OutputMatch {
  pointer?: string
  schema: JSONSchema
}
```

```yaml
pointer: /decision
schema:
  const: changes-requested
```

Rules:

- `pointer`, when present, is an RFC 6901 JSON Pointer.
- Without `pointer`, the schema applies to the complete result.
- `schema` must be a valid JSON Schema object or boolean.

### 15.5 Loop

Loop represents repeated application of one body target.

Fields are:

```text
body
maxIterations?
until?
```

```yaml
type: loop
name: reviewer-loop

body:
  type: agent
  name: code-reviewer

maxIterations: 12

until:
  pointer: /complete
  schema:
    const: true
```

Rules:

- An inline standalone Loop requires `body`.
- A locator-selected Loop may omit local body fields.
- `body` is a real Loop semantic field, not a generic wrapper.
- `body` contains one flat member.
- `body` accepts a named or contained target.
- `body` does not accept a selector.
- `maxIterations`, when present, is positive.
- `until` uses `OutputMatch`.
- `locator` cannot be combined with local body, iteration, or condition fields.

### 15.6 Workflow

Workflow represents a graph of target applications.

Fields are:

```text
start?
nodes?
edges?
```

A node combines graph fields and flat member fields.

```yaml
type: workflow
name: change-workflow

start:
  - plan

nodes:
  - id: plan
    type: agent
    name: planner

  - id: implement
    join: any
    type: agent
    name: implementer

  - id: review
    type: agent
    name: reviewer

  - id: final-check
    type: tool
    name: validate-change
    parameters:
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

Rules:

- A node contains `id`, optional `join`, and one flat member.
- There is no `target` property.
- Node IDs are unique.
- `start` values identify declared nodes and are unique.
- Edge endpoints identify declared nodes.
- Exact duplicate normalized edges are invalid.
- `join` is `all` or `any`.
- Omitted `join` means `all`.
- A node does not accept a selector.
- Workflow cycles are structurally valid.
- Source-array position does not define execution order.
- `locator` cannot be combined with local graph fields.

### 15.7 Workspace

Workspace represents one explicitly selected Artifact composition for a repository or application experience.

Fields are:

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

members:
  - type: text
    name: repository-rules
    insert: instructions
    locator: ./AGENTS.md
    parameters:
      mediaType: text/markdown

  - type: text
    name: application-source
    insert: user-message
    locator: ./src
    parameters:
      mediaType: text/plain
      include:
        - "**/*.go"
        - "**/*.ts"
      exclude:
        - "**/generated/**"
        - "**/vendor/**"

  - type: skill
    base: ./skills
    include:
      - "**/SKILL.md"
    nameInclude:
      - "checkout-*"
      - "review-*"

  - type: agent
    base: ./agents
    include:
      - "**/*.agent.yaml"
      - "**/*.agent.yml"
      - "**/*.agent.md"

  - type: workflow
    name: change-workflow
    locator: ./workflows/change-workflow.yaml
```

Rules:

- Workspace uses ordinary named, contained, and selector members.
- Text members select source text resources.
- Selectors select declared Artifacts.
- Workspace has no global portable source scanner configuration.
- Workspace has no `overrides` or `use` behavior in v1.
- Workspace does not automatically select every Artifact in its Root.
- Runtime budgets, local disablement, MCP installation data, and active-session state are not Workspace declaration fields.
- `locator` cannot be combined with local `members`.

## 16. Validation and canonicalization

### 16.1 Strict schemas

Concrete contracts use strict schemas.

`additionalProperties: false` applies except to explicitly typed maps such as:

- `labels`
- `metadata`
- JSON Schema values
- Tool or MCP environment maps
- Header maps
- MCP installation input maps
- MCP connection profile maps
- MCP policy Tool maps

Open maps do not permit hiding application contract behavior.

### 16.2 Member-form validation

Valid named member:

```yaml
- type: skill
  name: reviewing-code
```

Valid contained member:

```yaml
- type: text
  name: local-rules
  insert: instructions
  parameters:
    content: |
      Keep changes focused.
```

Valid selector:

```yaml
- type: skill
  base: ./skills
  nameInclude:
    - "review-*"
```

Invalid mixed selector and name:

```yaml
- type: skill
  name: reviewing-code
  base: ./skills
```

Invalid mixed selector and contained body:

```yaml
- type: skill
  base: ./skills
  parameters:
    license: Apache-2.0
```

Invalid flattened contained body:

```yaml
- type: text
  name: local-rules
  insert: instructions
  content: |
    Keep changes focused.
```

### 16.3 Parameters opacity and effective validation

The generic member layer validates that `parameters` is an object but does not use its contents to decide the member form.

The effective target is then validated by its concrete declaration schema.

Consequences:

- A nested `name` does not turn a selector into a named member.
- A nested `base` does not turn a contained member into a selector.
- A nested `insert` does not alter outer Text identity.
- Outer `type`, `name`, `locator`, and Text `insert` are authoritative.
- Unsupported remaining payload fields are rejected by the concrete target schema.
- Redundant nested values may change declaration bytes and digests, but not the authoritative outer identity.

### 16.4 Duplicate and ordering validation

For one containing field:

- Two byte-equivalent canonical members are invalid.
- Two contained declarations with the same type and name are invalid.
- Contained Text additionally distinguishes `insert`.
- Two relationships to the same target may be valid if normalized relationship behavior differs.
- Member-array position is not part of identity.
- Reordering may change source bytes and Definition digest.
- Reordering must not change composition semantics.

Intrinsically ordered values remain ordered, including:

- Tool command arguments.
- MCP command arguments.
- Explicit runtime invocation arrays.

### 16.5 Metadata validation

Invalid:

```yaml
metadata:
  flexigpt.site/mcp-runtime-v1:
    timeoutMS: 60000
```

Valid:

```yaml
timeoutMS: 60000
```

### 16.6 Legacy wire forms

The current contracts do not accept these retired forms:

- Declaration-instance `$schema`.
- Declaration-instance `apiVersion`.
- Collection-only `version`.
- Artifact type `collection`.
- Artifact types `instruction` and `context`.
- Type-key-wrapped members.
- Workflow node `target`.
- Flattened contained target body fields.
- `params` as an alias for `parameters`.
- Generic Model provider `parameters`.
- Tool `autoExecReco`.
- MCP runtime configuration hidden in metadata.
- MCP policy `body`.
- Nested MCP policy `toolName`.
- Workspace `declarations`.
- Workspace `roots`.

The corresponding current forms are:

- `plugin`.
- `text` with explicit `insert`.
- Flat member fields.
- Contained `parameters`.
- Direct typed Model, Tool, MCP, and policy fields.
- Workspace `members`.

## 17. Extension model

A new Artifact type must provide:

- A concrete portable `type`.
- A standalone declaration schema.
- A contained target payload schema.
- A versioned implementation package.
- Explicit locator semantics.
- Explicit allowed member positions.
- Explicit selector eligibility.
- Explicit `overrides` fields, if any.
- Explicit `use` fields, if any.
- Type-specific validation and canonical decoding.

A new type must not rely on:

```yaml
metadata:
  some.vendor/runtime-extension:
    hiddenBehavior: true
```

It must define explicit typed properties instead.

Adding a new type must not require adding a public generic resolver method. Resolver integration is registered per type in the ecosystem layer.

## 18. Implementation structure

Portable contracts are implemented under:

```text
internal/artifactcontract/declaration
```

Shared implementation includes:

- Common headers.
- Locator representation and validation.
- Member classification.
- Relationship projections.
- Member selector validation.
- Output matching.
- Canonical JSON helpers.
- Schema validation.
- Digest helpers.
- Stable contained declaration walking.

Versioned type packages are:

```text
textv1
modelv1
toolv1
skillv1
mcpv1
mcppolicyv1
pluginv1
agentv1
teamv1
loopv1
workflowv1
workspacev1
```

Each package owns:

- One concrete document type.
- One JSON Schema Draft 2020-12 document.
- One internal Artifact Store schema key.
- Strict canonical JSON decoding.
- Type-specific structural validation.

The generic Artifact Store does not import these concrete contract packages. Application composition registers schemas, decoders, and type resolvers.

Contained declarations are emitted at stable semantic subresource paths. Examples include:

```text
members/text/instructions/repository-rules
members/mcp/local-files
allowedTools/tool/searchfiles
body/agent/code-reviewer
nodes/review/agent/reviewer
loop/reviewer-loop
workflow/change-workflow
```

Array indexes are not used in these paths.

No old-wire compatibility decoder or automatic declaration migration is required for the retired development-time forms.

## 19. Current contract status

Status terminology:

- `Available` means the current implementation path exists.
- `Removed` means the legacy wire behavior is intentionally rejected.
- Status does not assert that all repository-wide build, test, static-analysis, migration, or binding verification has completed.

| Contract capability                                                 | Status     |
| ------------------------------------------------------------------- | ---------- |
| Current Artifact type vocabulary                                    | Available  |
| Independent v1 package per type                                     | Available  |
| Draft 2020-12 schemas                                               | Available  |
| Common portable header                                              | Available  |
| Declaration-instance schema fields                                  | Removed    |
| Portable path, URL, Git, package, command, and scalar Locator union | Available  |
| Local path locator validation and resolution helper                 | Available  |
| External locator representation                                     | Available  |
| Flat named members                                                  | Available  |
| Located external members                                            | Available  |
| Contained members through `parameters`                              | Available  |
| Structural containment classification                               | Available  |
| Outer-only contained `locator` ownership enforcement                | Not needed |
| Opaque generic `parameters` classification                          | Available  |
| Stable contained declaration positions                              | Available  |
| Member selectors                                                    | Available  |
| Rejection of `overrides` and `use` where no behavior is defined     | Available  |
| Text insertion identity                                             | Available  |
| Typed Model defaults                                                | Available  |
| Strict Tool implementations                                         | Available  |
| Direct typed MCP fields                                             | Available  |
| Flat MCP policy fields                                              | Available  |
| Plugin contract                                                     | Available  |
| Flat Workflow nodes                                                 | Available  |
| Workspace `members` contract                                        | Available  |
| Exact duplicate member detection                                    | Available  |
| Non-positional member identity                                      | Available  |
| Legacy `collection`, `instruction`, and `context` contracts         | Removed    |
| MCP runtime metadata extension                                      | Removed    |
| Workspace `declarations` and `roots`                                | Removed    |
