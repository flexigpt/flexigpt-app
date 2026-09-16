# Artifact contracts

`internal/artifactcontract/declaration` contains portable artifact declaration contracts.

The generic Artifact Store does not import concrete declaration contracts. It
stores generic Definitions, source-backed Artifacts, Source refresh state, and
verified resources.

The `declaration` package contains shared declaration utilities:

- Common `type`, `name`, `description`, `locator`, and `metadata` header.
- Portable locator representation and validation.
- Named nested `Entry` representation.
- Shared output matcher representation.
- Canonical JSON, JSON Schema, cloning, and digest helpers.

Every declaration and every nested `Entry` requires a portable `name`.

Each portable declaration type owns an independent versioned package:

- `instructionv1`
- `contextv1`
- `toolv1`
- `modelv1`
- `skillv1`
- `mcpv1`
- `collectionv1`
- `agentv1`
- `teamv1`
- `loopv1`
- `workflowv1`
- `workspacev1`
- `mcppolicyv1`, a supported MCP policy-domain Artifact contract with portable type `mcp.policy`

Each package owns:

- One concrete Go document type.
- One JSON Schema Draft 2020-12 document.
- One internal Artifact Store schema key.
- Strict canonical JSON decoding.
- Structural validation for that specific declaration type.

Nested heterogeneous declarations use `declaration.Entry`.

`Entry` validates the common header and mandatory name. In a composition
position, `Entry.CompositionForm` distinguishes an external reference from a
contained declaration. A header-only entry with `type`, `name`, an optional
non-command locator, and an optional MCP `server` selector is an external
reference. Header annotations remain membership-local and do not modify the
target Artifact. Type-specific body fields make an entry contained.

Contained declarations are emitted as source-backed subresources. External
references are graph edges only and do not create another Artifact. This keeps
each declaration contract independent and avoids a root-level union schema.

The portable declaration header is:

```yaml
$schema: https://schemas.flexigpt.site/artifact/agent/v1.json
apiVersion: v1
type: agent
name: reviewer
description: Reviews repository changes
locator: ./agents/reviewer.yaml
metadata:
  acme.example/owner: platform
```

The portable document does not contain:

- `schemaID`
- `schemaVersion`
- `digest`
- `logicalName`
- `logicalVersion`
- `displayName`

There are no anonymous declarations.

Physical source-format adapters normalize their inputs into these portable
declarations. They do not introduce alternate portable declaration headers.
