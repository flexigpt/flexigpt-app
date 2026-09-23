The verified corrections affect Collection semantics, the completed frontend, import behavior, MCP validation status, and the implementation-oriented sections. A diff would replace most of the document, so this is a full-file replacement.

# Agent Store HLD

Status: Current design and implemented user experience. Source-backed Agent catalog, Agent Collections, managed Agent import and export, protected built-in Agent packages, MCP setup UI, and read-only Agent inspection are available. Managed inline MCP credential-placement validation is partial: known sensitive environment and header positions are validated, while all command and URL credential placements are not comprehensively classified.

Normative foundations:

- [Portable Artifact Declaration Contracts HLD](./artifact_contracts_hld.md)
- [Artifact Store, Resolution, and Ecosystem HLD](./artifacts_hld.md)

This HLD defines Agent-specific Store policy, Agent Collection behavior, managed Agent import and export, built-in Agent integration, and the management user experience.

Portable declaration syntax, Source behavior, Artifact persistence, relationship resolution, fallback, selector behavior, resource verification, enablement, and composition semantics are inherited from the normative HLDs and are not redefined here.

## Table of contents

- [Purpose](#purpose)
- [Scope and boundaries](#scope-and-boundaries)
- [User experience](#user-experience)
- [Architecture and ownership](#architecture-and-ownership)
- [Agent Collections](#agent-collections)
- [Managed Agent authoring](#managed-agent-authoring)
- [Managed dependency lookup](#managed-dependency-lookup)
- [Import preview and commit](#import-preview-and-commit)
- [Reads, resolution, export, and enablement](#reads-resolution-export-and-enablement)
- [Built-in Agents](#built-in-agents)
- [Workspace, Composer, and runtime integration](#workspace-composer-and-runtime-integration)
- [Security and persistence](#security-and-persistence)
- [Implementation mapping](#implementation-mapping)
- [Current implementation status](#current-implementation-status)

## Purpose

The Agent Store is the application-level catalog and lifecycle facade for source-backed `agent` Artifacts.

The primary flow is:

```text
Repository, managed, or protected Source
  -> Agent Definition and Artifact
  -> shared Agent resolver
  -> management UI, Composer projection, or runtime consumer
```

The Agent Store provides:

- Agent catalog and typed resolution views.
- Agent Collections backed by Agent-only Plugin Artifacts.
- File-based managed Agent import.
- Portable Agent export.
- Managed Agent package lifecycle policy.
- Built-in Agent package hydration and validation.
- Agent and Collection enablement through universal Artifact metadata.
- Verified materialization of Agent-owned Text Artifacts.
- MCP setup handoff for resolved MCP Artifacts.

The Agent Store does not add another Agent declaration format, persistence database, membership database, runtime model, or execution engine.

## Scope and boundaries

### Goals

The design must:

- Reuse the Artifact Store, Source model, and typed resolver ecosystem.
- Store Agents as ordinary source-backed `agent` Artifacts.
- Represent user-facing Agent Collections through ordinary `plugin` Artifacts.
- Keep Agent declaration content portable and source-owned.
- Require import and export for managed Agent declaration authoring.
- Preserve partial relationship resolution and diagnostics.
- Keep Collection membership independent from Agent ownership.
- Keep declaration availability separate from runtime readiness.
- Support protected built-in Agent packages without a separate Agent Root or overlay database.
- Let users configure MCP installation-local state without changing Agent YAML.

### In scope

This HLD defines:

- Agent catalog, read, resolution, export, and enablement behavior.
- Agent-only managed Collection policy.
- Baseline Agent Collection provisioning.
- Managed Agent YAML import and deletion.
- Managed Agent immutability after import.
- Managed dependency scope normalization.
- Client-carried signed import preparation.
- Best-effort Collection membership and Agent package publication.
- Built-in Agent package admission and protected Root hydration.
- Agent Text materialization.
- MCP setup handoff after Agent import or inspection.
- Read-only Composer starter projection.
- Workspace and runtime integration boundaries.

### Out of scope

This HLD does not define:

- Agent execution.
- Conversation state.
- Model invocation values.
- Tool user inputs or invocation arguments.
- Skill execution or activation state.
- MCP connection health, discovered capability state, runtime invocation state, or readiness aggregation.
- A visual Agent declaration editor.
- In-place managed Agent declaration editing.
- Managed Agent replacement or upsert.
- Generic import of arbitrary Artifact declaration types.
- Linked-file synchronization or import-file watching.
- Backend-held import sessions, commit-token registries, or replay receipts.
- Cross-package transactions spanning Collection and Agent package mutations.
- Agent release or package version fields.
- URL, Git, package, archive, or command locator materialization.

## User experience

### Browse Agents and Collections

The Agent management experience presents Agent Collections and the currently available Agents selected by each Collection.

A user can:

- Browse user-managed and protected built-in Agent Collections.
- Expand a Collection to view its currently available Agent targets.
- Inspect an Agent's metadata, declaration YAML, resolution state, and diagnostics.
- View whether an Agent or Collection is enabled.
- Export an available Agent.
- Inspect MCP setup requirements for resolved MCP relationships.

The Collection list is an availability-oriented catalog. It shows currently available Agent Artifacts. A Collection capability inspection remains the statusful view for declared relationships that are unavailable or ambiguous.

### Manage Collections

A user can create an ordinary Agent Collection, edit its display metadata, enable or disable it, and delete it when it has no direct declared members.

The baseline Collection is selectable as an import destination, but its logical identity and descriptive metadata are application-provisioned and are not edited through the management UI.

Protected built-in Collections are read-only as declaration content. Their local enabled state remains configurable through the authorized Artifact metadata path.

The management UI does not expose a standalone Agent membership editor, attach operation, or detach operation.

Membership is created or preserved by managed Agent import. This is intentional: Agent declaration authoring is file-based, and Collection management does not become a free-form Agent composition editor.

### Import a managed Agent

The user flow is:

```text
Create or select an editable Agent Collection
  -> choose one YAML file
  -> preview validation, normalized YAML, diagnostics, and conflicts
  -> accept required confirmations
  -> import into the selected Collection
  -> inspect the published Agent and configure MCP setup if needed
```

The preview shows the user:

- The normalized declaration that will be published.
- The resulting Definition digest.
- Projected declaration Artifacts.
- Import-blocking errors and conflicts.
- Informational dependency availability.
- Required confirmations.
- MCP declarations that may need setup after publication.

An unavailable or ambiguous external dependency does not prevent import when the Agent declaration itself is structurally valid.

### Inspect, export, and delete a managed Agent

Managed Agent declarations are immutable after import.

A user changes a managed Agent by:

```text
Export Agent YAML
  -> edit outside the application
  -> import under a new name
```

Or:

```text
Export Agent YAML
  -> delete the managed Agent
  -> edit outside the application
  -> import again with the same name
```

Deleting a managed Agent removes only its managed package. It does not remove Collection relationships. Those relationships become unavailable until an exact matching Agent is restored.

### Configure MCP setup

Import and Agent inspection can identify MCP declarations that require local setup.

For a resolved MCP Artifact, the MCP setup experience can collect or update:

- Text and path installation inputs.
- Secret-backed installation inputs.
- OAuth client credential inputs.
- Local MCP installation data.

For an unresolved named MCP relationship, the user receives setup status and resolution diagnostics but cannot configure a missing Artifact.

MCP setup does not modify Agent YAML. It does not occur during import and does not make import conditional on connection readiness.

### View a Composer starter projection

The Agent details experience can present a read-only Composer starter projection.

The projection may show:

- Resolved mapped or Artifact-backed Model and Tool selections.
- Skill modes.
- Rendered instruction Skill text.
- MCP runtime selections.
- Resolution and projection diagnostics.

This is a read-only consumer projection. It is not an Agent declaration editor, replacement mechanism, or runtime readiness result.

## Architecture and ownership

### Responsibility boundaries

| Concern                                                                      | Owner                   |
| ---------------------------------------------------------------------------- | ----------------------- |
| Portable declaration syntax and validation                                   | Declaration contracts   |
| Roots, Sources, Definitions, Artifacts, resources, and enablement            | Artifact Store          |
| Relationship lookup, fallback, locators, selectors, aliases, and diagnostics | Shared resolvers        |
| Agent catalog policy, Collections, import/export, and managed lifecycle      | Agent Store             |
| Agent package and Collection package publication                             | Managed Source workflow |
| MCP installation values, secret references, OAuth, and profiles              | MCP management          |
| Agent execution and conversation state                                       | Runtime consumers       |

The Agent Store is a domain facade over existing Artifact infrastructure. It does not own a parallel declaration database, resolver, runtime, or membership store.

### Source-backed state

Managed state consists of ordinary Source-backed declaration packages.

The important invariants are:

- Agent Collections are independent Plugin declaration packages.
- Managed Agents are independent Agent declaration packages.
- Artifact Store derives Definitions and Artifacts from those packages.
- Collection membership remains declaration content in the Collection's Plugin Definition.
- No Agent membership rows, reverse-membership rows, ownership records, or Agent-specific sidecar files are created.
- Physical package conventions do not introduce portable Agent release versions.

### Identity and local references

Portable relationships use the forms defined by the declaration contracts:

```text
name
name + scope
name + locator
contained parameters
selector base and filters
```

Local management APIs may use an authorized `ArtifactRef` to address an existing Artifact. An `ArtifactRef` is an operation input and is never persisted in an Agent or Collection declaration.

The Agent Store does not define an Agent release version.

Distinct Agent configurations are represented by:

- Distinct Agent names, which is the preferred managed form.
- Distinct located occurrences when the same semantic name is intentionally used more than once.

## Agent Collections

### Representation

An Agent Collection is an Agent Store view over an Agent-only Plugin Artifact.

```text
Agent Collection
  -> Plugin Artifact
  -> direct named Agent relationships
```

There is no portable `collection` Artifact type and no Agent-specific Collection declaration format.

A managed Agent Collection:

- Uses `type: plugin`.
- Permits only direct `type: agent` members.
- Permits only named external Agent member relationships.
- May use an unlocated name, a located Agent occurrence, or `scope: builtin` where the Plugin contract permits it.
- Does not permit contained Agent declarations.
- Does not permit Agent selectors.
- Does not own its selected Agents.

Managed Agent import always creates or preserves a located external Agent relationship.

A general mixed Plugin remains a Plugin and is not exposed as an Agent Collection.

### Baseline Collection

Each supported user Root has one application-provisioned baseline Agent Collection.

```text
Logical name: agent-baseline
Portable type: plugin
Direct member type: agent
```

The baseline Collection:

- Is an explicit import destination.
- Can receive imported Agent memberships.
- Is non-renamable.
- Has application-provisioned display metadata.
- Is non-deletable.
- Is not automatically selected by a Workspace.
- Is not automatically active.
- Is not an implicit destination for import.

Baseline provisioning is an application Root lifecycle concern, not generic Artifact Store Root behavior.

### Membership behavior

Collection membership follows ordinary Plugin composition semantics.

| Event                                               | Result                                                               |
| --------------------------------------------------- | -------------------------------------------------------------------- |
| Agent is imported into a Collection                 | The Collection receives or preserves one located Agent relationship. |
| Agent package is deleted                            | Declared Collection relationships remain and become unavailable.     |
| Matching Agent package is restored                  | Exact dangling relationships become available again.                 |
| Collection is deleted                               | Independent Agent packages remain available.                         |
| Agent Source updates at the same located occurrence | The relationship continues to identify that occurrence.              |

Collection membership does not:

- Own an Agent package.
- Change Agent metadata.
- Change Agent enablement.
- Configure MCP installation state.
- Configure Agent runtime state.
- Cascade deletion to an Agent.
- Imply Workspace selection or runtime activation.

### Membership maintenance boundary

The current user management experience does not expose arbitrary Agent attach, detach, or member-editing operations.

This means:

- Import is the user-facing path that adds Agent membership.
- Deleting an Agent intentionally leaves a dangling membership.
- A dangling membership still counts as a direct member for Collection deletion.
- An ordinary Collection with declared Agent relationships cannot be deleted until those relationships are removed through an authorized Collection maintenance path.
- The management UI does not currently offer a standalone detach control.

This boundary prevents Collection management from becoming a second Agent declaration editing system.

### Collection deletion

An ordinary managed Agent Collection can be deleted only when it has no direct declared Agent relationships.

The deletion guard includes:

- Available Agent relationships.
- Unavailable Agent relationships.
- Ambiguous Agent relationships.

The backend deletion guard is authoritative even when the Collection catalog currently shows no available Agents.

Baseline and protected built-in Collections cannot be deleted through user-managed APIs.

## Managed Agent authoring

### Import and export are the managed authoring path

Managed Agent declaration authoring is file-based.

The Agent Store does not expose:

- A field-by-field Agent composer.
- Declaration patching.
- Member editing.
- Agent rename.
- Agent replacement.
- Agent upsert.
- In-app YAML editing.

This restriction applies to managed Agent declarations. It does not remove the limited Collection lifecycle described in this HLD.

### Managed import profile

Managed import accepts a strict subset of the portable Agent contract.

| Area                                                           | Managed admission rule                                                       |
| -------------------------------------------------------------- | ---------------------------------------------------------------------------- |
| Root                                                           | Must be one concrete `agent` declaration with a valid name.                  |
| Root locator                                                   | Not allowed.                                                                 |
| Loop or Workflow                                               | Not allowed.                                                                 |
| Text                                                           | Must be contained inline Text.                                               |
| Model                                                          | Must be a named external relationship.                                       |
| Tool                                                           | Must be a named external relationship.                                       |
| Skill                                                          | Must be a named external relationship.                                       |
| MCP                                                            | Must be a named external relationship or a contained inline MCP declaration. |
| MCP policy                                                     | Must be a named external Agent member relationship.                          |
| Selectors                                                      | Not allowed.                                                                 |
| Member locators                                                | Not allowed.                                                                 |
| Nested Agent or Plugin composition                             | Not allowed.                                                                 |
| Contained Model, Tool, Skill, Plugin, Agent, Loop, or Workflow | Not allowed.                                                                 |
| Persisted local IDs or runtime state                           | Not allowed.                                                                 |
| Installation values, secret values, and OAuth tokens           | Not allowed.                                                                 |

An Agent with no members is valid.

A contained Text declaration must:

- Use `instructions` or `user-message` insertion.
- Use inline content.
- Not use a locator.
- Not use source include or exclude patterns.

A contained MCP declaration must:

- Be a concrete MCP declaration.
- Use valid stdio or streamable HTTP transport configuration.
- Declare a command for stdio.
- Declare a URL for streamable HTTP.
- Contain installation-input definitions only, not installation values.
- Contain no source locator or source server selector.
- Contain no secret values, OAuth tokens, or runtime connection state.

Tool `autoExecute: true` requires explicit confirmation.

An inline stdio MCP declaration requires explicit confirmation.

### Validity categories

| Imported declaration state                             | Import result               |
| ------------------------------------------------------ | --------------------------- |
| Empty Agent                                            | Valid and importable.       |
| Structurally valid Agent with unavailable dependencies | Valid and importable.       |
| Structurally valid Agent with ambiguous dependencies   | Valid and importable.       |
| Agent with malformed contained Text or MCP content     | Invalid and not importable. |
| Agent with unsupported managed member form             | Invalid and not importable. |

Imported declarations are never partially salvaged.

The importer must not:

- Drop invalid members.
- Rewrite invalid members into another form.
- Publish only a valid subset.
- Silently remove malformed contained content.

## Managed dependency lookup

### Managed scope normalization

A managed import may accept an omitted scope or `scope: builtin` for an allowed named Agent member dependency.

Before digesting or publishing the Agent, the importer normalizes every allowed named external Agent member dependency to:

```text
scope: builtin
```

The normalized declaration is used for:

- Preview YAML.
- Definition digest calculation.
- Prepared import payloads.
- Published package content.
- Later export.

This prevents imported Agent members from silently resolving to arbitrary user Root Artifacts.

### Managed dependency targets

| Dependency type           | Managed lookup behavior                                                               |
| ------------------------- | ------------------------------------------------------------------------------------- |
| `model`                   | Protected built-in Artifact first, then a registered mapped fallback where supported. |
| `tool`                    | Protected built-in Artifact first, then a registered mapped fallback where supported. |
| `skill`                   | Protected built-in Artifact only.                                                     |
| `mcp`                     | Protected built-in Artifact only.                                                     |
| `mcp.policy` Agent member | Protected built-in Artifact only.                                                     |

The Agent Store does not invent mapped targets.

### Inline MCP policy references

An inline MCP may declare its own portable direct MCP policy reference.

That relation is owned by the MCP contract, not by the Agent member grammar. The portable MCP policy reference does not carry `scope`.

Therefore:

- Agent member scope normalization does not rewrite the MCP-owned policy reference.
- The policy reference is structurally validated during import.
- Preview may report an informational protected-policy observation.
- No policy target snapshot is persisted in the imported Agent.
- Post-publication resolution follows normal MCP policy resolution behavior.

This is intentionally distinct from a direct `mcp.policy` Agent member relationship.

### Informational dependency preflight

Preview resolves normalized Agent member dependencies and reports:

- Declared type and name.
- Normalized scope.
- `available`, `unavailable`, or `ambiguous` status.
- Protected Artifact provenance when available.
- Mapped target provenance when available.
- Relationship diagnostics.

Dependency resolution is informational.

A structurally valid import must not fail solely because:

- A named Model, Tool, Skill, MCP, or MCP policy is absent.
- A named dependency is ambiguous.
- A dependency Source is unavailable.
- An external target declaration has a diagnostic.
- A mapped Model or Tool target is unavailable.

Dependency state is not captured as a commit witness.

## Import preview and commit

### Import input

Managed import accepts one user-selected YAML file.

The file input path is transient. It is not persisted in the managed Agent package or prepared import payload.

The import reader must:

- Read only the selected file.
- Reject directories and unsupported special files.
- Enforce bounded input size.
- Respect cancellation.
- Return valid UTF-8 content.
- Avoid logging declaration content.
- Require one YAML object document.
- Reject duplicate mapping keys and invalid YAML structure.

### Preview

Preview performs the following user-visible workflow:

```text
Read selected YAML
  -> calculate input digest
  -> validate portable Agent declaration
  -> validate managed import profile
  -> normalize managed dependency scopes
  -> validate contained Text and MCP declarations
  -> calculate normalized Definition digest
  -> inspect destination, identity, package, and membership conflicts
  -> inspect dependency status
  -> return preview and signed preparation when importable
```

Preview returns:

- Normalized YAML.
- Input and Definition digests.
- Projected declaration Artifacts.
- Destination Collection information.
- Relationship observations.
- MCP setup descriptors.
- Restored dangling-membership information.
- Import issues and conflicts.
- Required confirmation codes.
- A signed prepared payload when the import can proceed.

### Diagnostics

| Condition                                                | Result                 |
| -------------------------------------------------------- | ---------------------- |
| Invalid YAML or invalid Agent declaration                | Import-blocking error. |
| Invalid managed member form                              | Import-blocking error. |
| Invalid contained Text or MCP declaration                | Import-blocking error. |
| Invalid destination Collection                           | Import-blocking error. |
| Agent identity, package, or Collection-member conflict   | Import-blocking error. |
| Dependency unavailable or ambiguous                      | Warning.               |
| External dependency declaration unavailable or malformed | Warning.               |
| Tool auto-execution                                      | Confirmation required. |
| Inline stdio MCP                                         | Confirmation required. |

### Signed preparation

Preview produces a client-carried HMAC-signed prepared payload.

The payload binds:

- The normalized Agent declaration.
- Input and Definition digests.
- The selected Root, Source, and Collection.
- The expected Collection revision.
- The expected managed Source generation.
- The managed package identity.
- Required confirmation codes.
- Expiration time.

The payload does not contain:

- The original import path.
- Secret values.
- OAuth tokens.
- MCP installation values.
- Backend-held prepared-import state.
- Dependency witnesses.
- Dependency revision snapshots.
- Runtime readiness state.

The prepared payload is authenticated, not encrypted.

The signer is process-local. Application restart invalidates uncommitted prepared payloads.

### Commit

Commit verifies:

- Payload authenticity and fingerprint.
- Payload expiration.
- Required confirmations.
- Collection revision.
- Managed Source generation.
- Normalized declaration validity.
- Current identity and package conflicts.
- Current Collection membership conflicts.

Commit does not:

- Reread the original YAML file.
- Accept replacement declaration content from the client.
- Require dependencies to remain available.
- Require MCP installation completion.
- Attempt MCP connection.

The commit sequence is:

```text
Verify prepared payload
  -> revalidate destination state and conflicts
  -> ensure Collection relationship
  -> publish managed Agent package
  -> reconcile managed Source state
  -> return Agent, Collection, and MCP setup information
```

### Best-effort publication

Collection membership and Agent package publication are separate Source operations.

The combined import operation is intentionally not atomic.

If membership succeeds and package publication fails:

```text
Collection relationship remains declared
  -> relationship is unavailable
  -> dangling membership records user intent
  -> user may correct the problem and preview again
```

No rollback is required.

### Conflicts

Managed Agent identity is Root-scoped:

```text
Root
  + type: agent
  + logical name
```

Import blocks when:

- An Agent with the same identity already exists.
- A conflicting missing or unpurged Agent occurrence exists.
- A protected built-in Agent reserves the requested name.
- The target managed package is occupied.
- A selected Collection already uses the Agent name for another direct relationship.
- A contained Text or MCP declaration conflicts with an existing Root Artifact.
- The package would require replacement.
- The requested name cannot be used as a managed package identity.

Managed import is create-only. It does not replace, merge, rename, or upsert packages.

## Reads, resolution, export, and enablement

### Resolution

Agent reads use the shared typed Agent resolver.

The resolved Agent view preserves:

- Declared relationships.
- Stable occurrence identity.
- `available`, `unavailable`, or `ambiguous` status.
- Artifact-backed targets.
- Mapped Model and Tool targets.
- Relationship behavior such as overrides and Skill use mode.
- Diagnostics.
- Optional Loop or Workflow relationships for general portable Agents.

Partial resolution is valid for inspection, export, and management.

Consumers that require complete resolution use the shared completeness behavior.

Declaration availability does not establish runtime readiness.

### Collection inspection

The Agent catalog lists currently available direct Agent targets for a Collection.

A Collection capability inspection provides the diagnostic view for all declared direct Agent relationships, including unavailable and ambiguous relationships.

This distinction allows the management UI to remain useful while preserving missing and conflicting user intent for inspection and recovery.

### Export

Any available Agent can be exported as canonical portable YAML, including:

- Managed imported Agents.
- Protected built-in Agents.
- Repository-backed Agents.
- Source-selected Agent aliases.
- Explicitly addressed contained Agents.

Export does not require complete dependency resolution.

Export preserves declaration shape:

- Contained declarations remain contained.
- Inline Text remains inline.
- Inline MCP remains inline.
- Managed scope normalization remains visible in managed Agent YAML.
- Repository and protected declarations retain their existing portable shape.

Export excludes local and runtime state, including:

- Collection membership.
- Artifact, Root, and Source IDs.
- Package addresses.
- Artifact enablement.
- Installation values.
- Secret references and values.
- OAuth tokens.
- Selected profiles.
- Connection state.
- Readiness state.

An export may include optional resolution diagnostics and MCP setup descriptors. Those diagnostics do not prevent export.

### Text materialization

Agent Store provides verified Text materialization for Text Artifacts referenced by Agent declarations.

The operation:

- Uses the Artifact Store verified-resource boundary.
- Supports inline and source-backed Text according to the Text contract.
- Returns materialized content with declaration identity and revision context.
- Is not a generic filesystem-read capability.

### Enablement

Agents and Agent Collections use universal `Artifact.Enabled`.

Enablement:

- Does not change declaration resolution.
- Does not change Collection membership.
- Does not change Definition content or digest.
- Does not prevent Agent export.
- Does not prevent authorized declaration reads.
- May be used by consumers for enabled-only filtering.

Disabling a Collection does not disable its Agents.

Disabling an Agent does not remove it from a Collection.

Managed Agent YAML does not contain enablement state.

## Built-in Agents

### Protected package model

Built-in Agents use ordinary Plugin and Agent declarations in the shared protected Root.

A built-in Agent package contains:

- One Agent Collection Plugin declaration.
- Direct named, package-local Agent relationships.
- One matching concrete Agent declaration for every declared Collection member.
- No unreferenced package-local Agent declaration.
- No source-selected Agent aliases at the package boundary.

The Collection and its Agents remain independent Artifacts even when distributed together.

### Built-in package admission

Built-in Agent package admission verifies:

- Package Plugin and Agent declarations are valid.
- The package Collection contains only named Agent relationships.
- Every member locator remains inside the package.
- Member names match their packaged Agent declarations.
- Package Agent identities and declaration occurrences are unique.
- Every packaged Agent declaration is referenced by the package Collection.
- Required relationships resolve completely after hydration.
- Package content contains no local Store identities, secrets, runtime state, or runtime credentials.

Admission validates declaration resolution only. It does not require MCP connection, Tool execution, Skill execution, Model provider readiness, or Agent runtime readiness.

### Protected Root behavior

Built-in Agent packages inherit protected Root behavior from the Artifact ecosystem.

Hydration must:

- Reconcile known package content by package fingerprint.
- Repair changed or incomplete known packages.
- Remove stale known packages.
- Preserve unchanged Artifact references where package content is unchanged.
- Avoid resetting unrelated protected Artifact topology.
- Avoid creating an Agent-specific protected Root or overlay database.
- Avoid inferring ownership between a Collection and its Agents.

Protected Agent package declaration content cannot be edited or deleted through user-managed Agent workflows.

Protected Agent and Collection enablement remains locally configurable through authorized Artifact metadata.

### Current built-in Agent Collections

Current protected Agent package roots include:

- `core-agents`
- `software-development-agents`
- `product-leadership-agents`
- `technical-content-writing-agents`
- `research-analysis-agents`

## Workspace, Composer, and runtime integration

### Workspace integration

Workspace Agent selection remains part of the portable Workspace contract.

```text
Resolve Workspace
  -> resolve explicitly selected Agent relationship
  -> consume the shared Agent capability graph
```

Agent Collections are catalog and organization constructs. They are not automatically selected by a Workspace.

The baseline Collection is not automatically selected.

Protected built-in Agents participate only through explicit relationships and existing fallback behavior.

### Composer integration

The Composer starter projection is a consumer of Agent resolution.

It may project supported resolved Agent capabilities into a read-only starter view. It must:

- Preserve relationship diagnostics.
- Distinguish mapped and Artifact-backed targets.
- Avoid changing Agent declarations.
- Avoid claiming runtime readiness.
- Avoid silently substituting unavailable relationships.

The current Composer projection can render instruction Skill text and selected Model, Tool, Skill, and MCP information.

It does not currently materialize every declared Text Artifact into the Composer preview.

### MCP integration

MCP management owns installation-local configuration and credentials.

After Agent import or inspection:

```text
Resolved MCP Artifact
  -> inspect installation declarations
  -> collect local values or secret references
  -> perform OAuth setup where applicable
  -> select local profile
  -> connect through MCP runtime flows
```

These actions do not modify the Agent declaration.

### Runtime boundary

The Agent Store ends at the resolved Agent declaration graph.

Runtime consumers own:

- Prompt assembly and budgets.
- User messages.
- Model invocation values.
- Tool inputs and invocation arguments.
- MCP runtime context and connection state.
- Skill activation.
- Loop and Workflow execution.
- Scheduling, retries, cancellation, and output handling.
- Conversation and execution state.

## Security and persistence

### Persisted state

The feature persists normal Artifact ecosystem state:

- Managed Agent packages.
- Managed Collection package changes.
- Definitions and Artifacts.
- Source generation and Source reconciliation state.
- Artifact enablement.
- MCP local installation data through MCP management.
- Secret references through MCP and secret-storage APIs.

### Non-persisted state

The feature does not persist:

- Original import paths.
- Backend prepared-import records.
- Commit-token registries.
- Source file content outside the managed package.
- Dependency witnesses.
- Dependency revision snapshots.
- Agent readiness.
- MCP installation values in Agent YAML.
- Secret values in Agent YAML.
- OAuth tokens in Agent YAML.
- Collection membership inside Agent YAML.

### Credential boundary

Portable Agent YAML must not contain secret values.

Current managed inline MCP validation rejects literal credential-like values in recognized sensitive positions, including:

- Root MCP environment variables.
- Root MCP headers.
- Stdio connection-profile environment variables.
- HTTP connection-profile headers.
- Invalid OAuth client-credential input declarations.

The current validation does not provide complete credential detection for:

- Command arguments.
- URLs.
- URL query values.
- URL user information.
- Other transport-specific credential conventions.

Users must not treat managed Agent import as a general secret-scanning mechanism.

## Implementation mapping

| Requirement area                 | Implementation mapping                                                                                                                             |
| -------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------- |
| Portable Agent and Plugin syntax | Existing declaration contracts and schema validation.                                                                                              |
| Managed import restrictions      | A non-portable managed admission profile layered over the portable Agent contract.                                                                 |
| Source-backed persistence        | Existing managed Source publication and Artifact Store reconciliation.                                                                             |
| Import integrity                 | Client-carried HMAC-signed prepared payload with expiry and destination revision checks.                                                           |
| Dependency inspection            | Shared typed resolver with protected built-in scope and mapped fallback support where registered.                                                  |
| Collection policy                | Agent Collection domain restricts managed Collection members to named Agent relationships.                                                         |
| Built-in content                 | Protected package hydration, package verification, and final complete-resolution admission.                                                        |
| Management UI                    | Collection management, import preview and confirmation, Agent inspection/export, enablement, deletion, Composer projection, and MCP setup handoff. |
| MCP local configuration          | Existing MCP management and secret-storage boundaries.                                                                                             |

No Agent-specific declaration database, relationship database, sidecar file, runtime package, or protected overlay Store is required.

## Current implementation status

Status terminology:

- `Available` means the current product path is implemented.
- `Partial` means the feature exists with an intentionally bounded current scope.
- `Pending` means approved verification or coverage work remains.
- `Deferred` means intentionally outside current scope.
- `Not supported` means the behavior is not exposed by the current product.
- Status does not assert completion of all repository-wide builds, tests, static analysis, or end-to-end verification.

### Catalog and Collection capabilities

| Capability                                   | Status        | Notes                                                                                     |
| -------------------------------------------- | ------------- | ----------------------------------------------------------------------------------------- |
| Source-backed Agent catalog                  | Available     | Uses ordinary Agent Artifacts and typed resolver output.                                  |
| Agent resolution with partial diagnostics    | Available     | Available, unavailable, and ambiguous relationships remain observable.                    |
| Agent Collection catalog and management UI   | Available     | Supports browse, create, metadata update, enablement, details, and guarded deletion.      |
| Agent-only managed Collection policy         | Available     | Direct members are named external Agent relationships only.                               |
| Baseline Agent Collection provisioning       | Available     | One selectable, non-deletable baseline Collection is provisioned per supported user Root. |
| Collection capability inspection             | Available     | Preserves direct relationship diagnostics beyond currently available catalog entries.     |
| Direct Agent membership editor               | Not supported | Import is the current user-facing membership creation path.                               |
| Direct Agent attach and detach operations    | Not supported | No standalone management UI or Agent Store surface is exposed.                            |
| Managed Agent deletion preserving membership | Available     | Deletion leaves declared relationships unavailable.                                       |
| Agent and Collection enablement              | Available     | Uses universal `Artifact.Enabled` and does not alter resolution.                          |

### Managed import and export capabilities

| Capability                                                | Status        | Notes                                                                                                  |
| --------------------------------------------------------- | ------------- | ------------------------------------------------------------------------------------------------------ |
| YAML-only managed Agent import                            | Available     | Imports one selected YAML declaration file.                                                            |
| Strict managed Agent admission profile                    | Available     | Restricts managed members to inline Text, named dependencies, and inline MCP where allowed.            |
| Managed scope normalization                               | Available     | Named managed Agent dependencies are persisted with `scope: builtin`.                                  |
| Empty Agent import                                        | Available     | An Agent with no members is valid.                                                                     |
| Preview with normalized YAML and diagnostics              | Available     | Shows errors, warnings, confirmations, conflicts, and projected Artifacts.                             |
| Client-carried signed preparation                         | Available     | Prepared payloads are authenticated, expiring, and process-local.                                      |
| Dependency observations without import gating             | Available     | Missing and ambiguous dependencies remain warnings.                                                    |
| Create-only managed package publication                   | Available     | Replacement, merge, rename, and upsert are not supported.                                              |
| Best-effort Collection membership and package publication | Available     | Dangling membership is preserved on package publication failure.                                       |
| Managed Agent export                                      | Available     | Available Agents export canonical portable YAML without requiring complete resolution.                 |
| Managed Agent Text materialization                        | Available     | Verified backend materialization is available through Agent Store.                                     |
| Managed inline MCP credential-placement validation        | Partial       | Sensitive environment and header positions are validated.                                              |
| Complete URL and command credential classification        | Partial       | Import does not yet classify every credential-bearing URL or command position.                         |
| Protected scope pinning for an inline MCP policy field    | Partial       | The MCP policy field has no portable scope and follows normal MCP policy resolution after publication. |
| Managed Agent visual editor                               | Not supported | Managed declarations are immutable after import.                                                       |
| Generic Artifact import                                   | Not supported | Import is limited to the managed Agent profile.                                                        |
| Cross-package atomic import transaction                   | Not supported | Best-effort publication is intentional.                                                                |
| Linked-file synchronization                               | Not supported | Imported content is published into managed Source packages.                                            |

### Built-in, UI, and runtime capabilities

| Capability                                                      | Status    | Notes                                                                                                                   |
| --------------------------------------------------------------- | --------- | ----------------------------------------------------------------------------------------------------------------------- |
| Built-in Agent package embedding and hydration                  | Available | Uses the shared protected Root and package-scoped reconciliation.                                                       |
| Built-in package structural validation                          | Available | Requires local named package members and complete post-hydration resolution.                                            |
| Stale known built-in package removal                            | Available | Reconciles known package state without sweeping unrelated protected content.                                            |
| Protected Agent and Collection local enablement                 | Available | Uses protected Artifact metadata without an Agent overlay Store.                                                        |
| Managed Collection and import/export frontend                   | Available | Users can manage Collections, preview import, commit import, inspect Agents, export, enable, and delete managed Agents. |
| MCP setup frontend                                              | Available | Resolved MCP Artifacts can be configured through MCP management flows.                                                  |
| Read-only Composer starter projection                           | Available | Shows supported resolved Model, Tool, Skill, MCP, and instruction Skill information.                                    |
| Composer materialization of every Text Artifact                 | Partial   | Text materialization exists in Agent Store but is not fully consumed by the current Composer preview.                   |
| Test and acceptance coverage                                    | Pending   | Unit, integration, hydration recovery, concurrency, and end-to-end verification remain to be completed.                 |
| Agent runtime                                                   | Deferred  | Execution, runtime inputs, state, scheduling, and invocation remain outside Agent Store.                                |
| URL, Git, package, archive, and command locator materialization | Deferred  | Portable locator representation remains available through declaration contracts.                                        |
