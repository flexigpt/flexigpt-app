# Artifact contracts

`internal/artifactcontract/declaration` contains portable artifact declaration contracts.

The generic Artifact Store does not import concrete declaration contracts. It
stores generic Definitions, source-backed Artifacts, Source refresh state, and
verified resources.

The `declaration` package contains shared declaration utilities:

- Common `type`, `name`, `displayName`, `description`, `labels`, `locator`, and
  `metadata` header.
- Portable locator representation and validation.
- Flat named, contained, and selector member representations.
- Shared output matcher representation.
- Canonical JSON, JSON Schema, cloning, and digest helpers.

Standalone declarations require a portable `name`. Member selectors require
`type` and `base` and intentionally do not contain `name`.

Each portable declaration type owns an independent versioned package:

- `textv1`
- `modelv1`
- `toolv1`
- `skillv1`
- `mcpv1`
- `mcppolicyv1`
- `pluginv1`
- `agentv1`
- `teamv1`
- `loopv1`
- `workflowv1`
- `workspacev1`

Each package owns:

- One concrete Go document type.
- One JSON Schema Draft 2020-12 document.
- One internal Artifact Store schema key.
- Strict canonical JSON decoding.
- Structural validation for that specific declaration type.

Heterogeneous relationship fields use `declaration.Entry`. The wire format is
flat:

```yaml
- type: skill
  name: reviewing-code
  locator: ./skills/reviewing-code

- type: text
  name: repository-rules
  insert: instructions
  parameters:
    mediaType: text/markdown
    content: Keep changes focused.

- type: skill
  base: ./skills
  include:
    - "**/SKILL.md"
```

Presence of `parameters` identifies a contained declaration. Target-owned
common and type-specific fields belong inside `parameters`. Relationship-owned
`type`, `name`, `locator`, `overrides`, and `use` remain outside it.

Contained declarations are emitted as source-backed subresources. External
references and selectors remain graph relationships.

The portable declaration header is:

```yaml
type: agent
name: reviewer
displayName: Repository Reviewer
description: Reviews repository changes
locator: ./agents/reviewer.yaml
labels:
  category: engineering
metadata:
  acme.example/owner: platform
```

The portable document does not contain:

- `$schema`
- `apiVersion`
- `schemaID`
- `schemaVersion`
- `digest`
- `logicalName`
- `logicalVersion`

There are no anonymous declarations.

Physical source-format adapters normalize their inputs into these portable
declarations. They do not introduce alternate portable declaration headers.
