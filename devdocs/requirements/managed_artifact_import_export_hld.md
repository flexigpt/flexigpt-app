# Managed Agent File Import and Export HLD

Status: Revised proposed replacement for managed Agent composition authoring and Assistant Preset management.

This design intentionally uses a client-carried HMAC-signed prepared payload. It does not require backend-held prepared import state, projected Artifact overlays, cross-package transactions, Agent readiness aggregation, or an Agent reference catalog.

Normative foundations:

- [Portable Artifact Declaration Contracts HLD](./artifact_contracts_hld.md)
- [Artifact Store, Resolution, and Ecosystem HLD](./artifacts_hld.md)
- [Artifact-backed Agent Store HLD](./agent_store_hld.md)

- [1. Purpose](#1-purpose)
- [2. Scope and non-goals](#2-scope-and-non-goals)
- [3. Core decisions](#3-core-decisions)
- [4. Terminology](#4-terminology)
- [5. Managed Agent invariants](#5-managed-agent-invariants)
- [6. Managed dependency lookup and normalization](#6-managed-dependency-lookup-and-normalization)
- [7. Strict managed Agent profile](#7-strict-managed-agent-profile)
- [8. Import input and validation](#8-import-input-and-validation)
- [9. Preview and signed preparation](#9-preview-and-signed-preparation)
- [10. Best-effort commit behavior](#10-best-effort-commit-behavior)
- [11. Collection membership behavior](#11-collection-membership-behavior)
- [12. MCP handling](#12-mcp-handling)
- [13. Conflict handling](#13-conflict-handling)
- [14. Managed Agent export](#14-managed-agent-export)
- [15. API shape](#15-api-shape)
- [16. UI and runtime responsibilities](#16-ui-and-runtime-responsibilities)
- [17. Security and persistence boundaries](#17-security-and-persistence-boundaries)
- [18. Breaking removals](#18-breaking-removals)
- [19. Implementation requirements](#19-implementation-requirements)
- [20. Current implementation status](#20-current-implementation-status)

## 1. Purpose

This HLD defines file-based import and export for user-managed Agent Artifacts.

The managed Agent workflow is:

```text
Create or select an Agent Collection
  -> select an Agent YAML file
  -> validate and normalize the Agent declaration
  -> inspect current managed dependency resolution
  -> return a signed client-carried prepared payload
  -> commit the prepared Agent package
  -> best-effort add or preserve Collection membership
  -> resolve the published Agent normally
  -> configure MCP installation and runtime state through MCP flows
```

This design replaces field-by-field managed Agent composition.

It does not make Agent import responsible for:

- Connecting MCP servers.
- Populating MCP installation values.
- Saving secrets.
- Obtaining OAuth tokens.
- Determining operational MCP readiness.
- Producing a backend-owned Agent readiness state.
- Building a visual Agent composition editor.
- Maintaining backend-held import sessions.

## 2. Scope and non-goals

### 2.1 In scope

This HLD defines:

- Safe YAML file import for managed Agents.
- Strict managed Agent admission.
- Normalization of managed external dependency scope.
- Client-carried signed prepared payloads.
- Best-effort Agent package publication and Collection membership.
- Conflict detection for Agent identity and managed packages.
- Non-blocking dependency-resolution diagnostics.
- Validation of contained Text and inline MCP declarations.
- Portable managed Agent export.
- Deletion and re-import behavior.
- Removal of structured managed Agent create and replace APIs.

### 2.2 Out of scope

This HLD does not define:

- Backend-held prepared import registries.
- Random commit-token registries.
- One-time commit-token consumption.
- Replay receipts.
- Atomic transactions across Collection and Agent package mutations.
- Projected Artifact Store overlays.
- Agent readiness APIs.
- MCP installation completion APIs in Agent Store.
- MCP connection attempts during import.
- Secret entry during Agent import.
- Reference catalogs for Agent dependency authoring.
- A generic import API that accepts arbitrary declaration types.
- A visual Agent editor.
- In-place Agent declaration editing.
- Managed Agent replacement.
- Agent release versions.
- Linked-file synchronization.
- Watching imported files.
- Assistant Preset compatibility or migration.

## 3. Core decisions

### 3.1 Collections are created separately

Collection creation is independent from Agent import.

The user first creates or selects an existing editable Agent Collection. Import requires that Collection as its destination.

There is no transaction spanning:

```text
Create Collection
  + import Agent
```

### 3.2 Preparation is client-carried and signed

Preview produces an HMAC-signed prepared payload.

The client carries that payload into commit. The backend does not retain prepared import state.

The signed payload prevents the frontend from changing:

- The normalized Agent declaration.
- The managed package identity.
- The selected Collection.
- The expected Collection revision.
- The expected managed Source generation.
- Required confirmation codes.

The signed payload is authenticated, not encrypted. Portable declarations must therefore not contain secret values.

### 3.3 Import is best effort across membership and package publication

Collection membership mutation and Agent package publication are separate managed Source operations.

Each individual operation uses normal Artifact Store guarantees. The application does not require one transaction spanning both operations.

The selected behavior is:

```text
Ensure Collection relationship
  -> publish Agent package
```

If Collection membership succeeds and Agent publication fails:

- The Collection relationship remains declared.
- The relationship is unavailable.
- The dangling relationship represents user intent.
- The user may preview and import again later.

This is intentional behavior.

### 3.4 Resolution is observational, not an import gate

Preview attempts to resolve managed external relationships and reports whether each is:

```text
available
unavailable
ambiguous
```

Resolution result does not make a structurally valid Agent unimportable.

Import must not fail only because:

- A named MCP is absent.
- A named Tool is absent.
- A named Model is absent.
- A named Skill is absent.
- A named MCP Policy is absent.
- A target is ambiguous.
- A target Source is currently unavailable.
- A target declaration has a Source-side diagnostic.

### 3.5 Empty Agents are valid

A managed Agent with no members is valid.

Example:

```yaml
type: agent
name: empty-agent
displayName: Empty Agent
```

This is a valid managed Agent import.

The absence of members is not an error.

### 3.6 Malformed authored content is invalid

Every declaration authored inside the imported file must be valid.

Examples that block import:

- Invalid YAML.
- Invalid root Agent fields.
- Invalid member wire shape.
- Unsupported managed member type.
- Invalid contained Text.
- Invalid inline MCP.
- Invalid MCP transport.
- Invalid inline MCP policy reference syntax.
- Invalid relationship behavior.
- Invalid local secret placement.
- Invalid managed package identity.

The importer does not silently ignore malformed authored content.

### 3.7 Operational readiness belongs outside Agent import

Agent import validates declaration content and publishes Artifacts.

The UI and MCP subsystem own:

- Installation input completion.
- Secret storage.
- Secret references.
- OAuth setup.
- Profile selection.
- Connection establishment.
- Operational MCP readiness.
- User-facing readiness presentation.

## 4. Terminology

| Term                     | Meaning                                                                        |
| ------------------------ | ------------------------------------------------------------------------------ |
| Portable declaration     | A declaration accepted by the standard Artifact declaration contract           |
| Managed Agent profile    | The stricter subset accepted for managed Agent import and export               |
| Imported source file     | The user-selected YAML file read during preview                                |
| Source digest            | Digest of exact bytes read from the selected file                              |
| Normalized declaration   | Canonical Agent declaration after managed import normalization                 |
| Definition digest        | Digest of the normalized portable declaration                                  |
| Prepared payload         | HMAC-signed client-carried import plan                                         |
| Prepared fingerprint     | Digest of the signed prepared payload                                          |
| Managed lookup           | Built-in-first then mapped-fallback lookup used for managed Agent dependencies |
| External relationship    | A named dependency whose target is not contained in the imported Agent         |
| Contained declaration    | Text or MCP declaration physically contained in the imported Agent             |
| Unavailable relationship | A valid relationship whose target cannot currently be resolved                 |
| Malformed declaration    | A declaration whose own syntax, schema, or semantic contract is invalid        |
| Dangling membership      | A declared Collection relationship whose Agent target is unavailable           |

## 5. Managed Agent invariants

### 5.1 Validity categories

Managed Agent import distinguishes three categories.

| Category                                               | Import result              |
| ------------------------------------------------------ | -------------------------- |
| Empty Agent                                            | Valid and importable       |
| Structurally valid Agent with unavailable dependencies | Valid and importable       |
| Agent containing malformed authored declarations       | Invalid and not importable |

Examples:

```yaml
type: agent
name: valid-empty-agent
```

This is valid.

```yaml
type: agent
name: valid-unresolved-agent
members:
  - type: mcp
    name: not-currently-installed
```

This is valid and importable. The MCP relationship is unavailable.

```yaml
type: agent
name: invalid-inline-mcp-agent
members:
  - type: mcp
    name: broken-server
    parameters:
      transport: stdio
```

This is invalid because an inline stdio MCP requires a command.

### 5.2 Imported declarations are never partially salvaged

The imported file is one Agent declaration.

If any authored contained declaration is malformed, preview fails.

The importer must not:

- Drop malformed members.
- Rewrite malformed members into another shape.
- Publish only the valid subset.
- Publish an Agent with a silently removed inline MCP or Text declaration.

External target availability is different. A named relationship remains in the Agent declaration even if its target does not currently exist.

### 5.3 Managed Agents remain immutable after import

Managed Agent declaration content is immutable after import.

The public Agent API must not expose:

- Declaration patching.
- Member add or remove.
- Agent rename.
- Agent replacement.
- Agent upsert.
- New declaration versions.
- An in-app Agent YAML editor.

A user changes a managed Agent by:

- Exporting it.
- Editing it externally.
- Importing it under a new name.

Or:

- Exporting it.
- Deleting the current Agent.
- Editing it externally.
- Importing it again with the same name.

Artifact enablement remains separate local state.

## 6. Managed dependency lookup and normalization

## 6.1 Managed lookup differs from normal lookup

Normal relationship resolution may use:

```text
Current Root
  -> protected built-in Root
  -> mapped fallback
```

Managed Agent import uses:

```text
Protected built-in Root
  -> mapped fallback for supported types
  -> unavailable
```

Managed Agent import must not search the destination user Root for external dependencies.

This prevents a managed imported Agent from silently binding to arbitrary user Root Artifacts.

## 6.2 `scope` is optional in import input

Users may write either form:

```yaml
- type: mcp
  name: github
```

```yaml
- type: mcp
  name: github
  scope: builtin
```

For managed Agent import, these forms have the same meaning.

The importer normalizes all allowed named managed dependencies to:

```yaml
scope: builtin
```

before calculating the managed Definition and publishing the package.

This normalization is required because the ordinary resolver must later preserve managed lookup behavior after publication.

Without normalization, an unscoped relationship would later search the current user Root before built-ins.

## 6.3 Exported managed YAML uses normalized scope

Managed Agent export returns the normalized declaration.

If the input omitted `scope`, exported YAML may contain:

```yaml
scope: builtin
```

This is intentional. It records the actual managed dependency behavior.

The source digest remains the digest of user-selected input bytes. The Definition digest identifies the normalized managed declaration.

## 6.4 Built-in and mapped target behavior

Managed lookup supports:

| Type         | Lookup behavior                                          |
| ------------ | -------------------------------------------------------- |
| `model`      | Built-in Artifact first, then registered mapped fallback |
| `tool`       | Built-in Artifact first, then registered mapped fallback |
| `skill`      | Built-in Artifact only                                   |
| `mcp`        | Built-in Artifact only                                   |
| `mcp.policy` | Built-in Artifact only                                   |

A future type may support mapped fallback only when its resolver explicitly supports mapped targets.

The importer must not invent mapped targets.

## 6.5 Preview resolution is informational

Preview resolves normalized external relationships using managed lookup.

For each relationship, preview reports:

- Declared type.
- Declared name.
- Normalized scope.
- Current status.
- Built-in Artifact provenance when available.
- Mapped target provenance when available.
- Resolution diagnostic when unavailable or ambiguous.

These conditions do not block import:

- No matching built-in Artifact.
- No mapped fallback.
- Ambiguous built-in target.
- Unavailable Source.
- Malformed external target declaration.
- Missing external target package.

## 6.6 No dependency witnesses at commit

External dependency state is not part of commit correctness.

Commit must not fail because:

- A built-in dependency revision changed.
- A mapped target changed.
- A previously resolved target disappeared.
- A previously unavailable target became available.
- A relationship became ambiguous.
- A target Source refreshed.

The imported Agent stores relationships, not snapshots of dependency state.

The normal resolver determines current relationship status after publication.

## 7. Strict managed Agent profile

## 7.1 Root declaration

A managed Agent import root must:

- Use `type: agent`.
- Have a valid portable name.
- Be concrete.
- Have no top-level `locator`.
- Have no `loop`.
- Have no `workflow`.
- Contain only allowed managed member forms.
- Contain no Store identity fields.
- Contain no runtime state fields.

An empty `members` field or absent `members` field is valid.

## 7.2 Allowed member forms

| Member type  | Allowed managed form                                |
| ------------ | --------------------------------------------------- |
| `text`       | Contained inline Text                               |
| `model`      | Named external relationship                         |
| `tool`       | Named external relationship                         |
| `skill`      | Named external relationship                         |
| `mcp`        | Named external relationship or contained inline MCP |
| `mcp.policy` | Named external relationship                         |

Named external managed relationships may omit `scope` or use `scope: builtin`.

The importer normalizes both forms to `scope: builtin`.

## 7.3 Prohibited forms

The managed profile rejects:

- Top-level Agent locator.
- Agent loop.
- Agent workflow.
- Member selectors.
- Member locators.
- User Root named dependencies.
- Contained Model.
- Contained Tool.
- Contained Skill.
- Contained Plugin.
- Contained Agent.
- Contained Loop.
- Contained Workflow.
- Nested Agent composition.
- Persisted Artifact refs.
- Persisted Root IDs.
- Persisted Source IDs.
- Runtime MCP server IDs.
- Agent versions.
- Package versions.
- Local installation values.
- Secret values.
- OAuth tokens.
- Runtime connection state.

## 7.4 Contained Text

Contained Text must:

- Be physically contained in the Agent declaration.
- Use `insert: instructions` or `insert: user-message`.
- Use inline `content`.
- Not use a locator.
- Not use include or exclude patterns.

Example:

```yaml
- type: text
  name: review-instructions
  insert: instructions
  parameters:
    mediaType: text/markdown
    content: |
      Review correctness, security, and maintainability.
```

A malformed contained Text blocks import.

## 7.5 Named Model and Tool relationships

Model and Tool relationships are named external relationships.

Example:

```yaml
- type: model
  name: reasoning
```

```yaml
- type: tool
  name: searchfiles
  overrides:
    autoExecute: true
```

The importer normalizes them to:

```yaml
- type: model
  name: reasoning
  scope: builtin
```

```yaml
- type: tool
  name: searchfiles
  scope: builtin
  overrides:
    autoExecute: true
```

A missing or unavailable target does not block import.

`autoExecute: true` requires explicit user confirmation.

## 7.6 Named Skill, MCP, and MCP Policy relationships

Skill, MCP, and MCP Policy relationships are named external relationships.

Example:

```yaml
- type: skill
  name: reviewing-code
```

```yaml
- type: mcp
  name: github
```

```yaml
- type: mcp.policy
  name: builtin-manual-write
```

The importer normalizes each to `scope: builtin`.

No currently available target is required for import success.

## 7.7 Contained inline MCP

An inline MCP is a contained MCP declaration.

Example:

```yaml
- type: mcp
  name: local-server
  parameters:
    transport: stdio
    command: npx
    args:
      - -y
      - "@example/mcp-server"
```

An inline MCP must:

- Be concrete.
- Have valid MCP transport configuration.
- Have a command for `stdio`.
- Have a URL for `streamableHTTP`.
- Have valid installation input declarations.
- Not contain a declaration locator.
- Not contain a source server selector.
- Not contain runtime connection state.
- Not contain installation input values.
- Not contain secret values.
- Not contain OAuth tokens.

Malformed inline MCP declarations block import.

An inline stdio MCP requires explicit user confirmation.

## 8. Import input and validation

## 8.1 Safe file input

Preview accepts YAML files only.

The production file reader must use the safe filesystem tool adapter, currently represented by `llmtoolsutil.ReadPortableTextFile` or an equivalent safe wrapper.

The file reader must:

- Read only the user-selected file.
- Reject directories and unsupported special files.
- Enforce bounded file size.
- Respect cancellation.
- Prevent unsafe path traversal and symlink escape.
- Avoid logging declaration content.
- Return valid UTF-8 text.
- Not persist the selected path.

## 8.2 YAML requirements

The imported file must contain:

- One YAML document.
- One YAML object root.
- No duplicate mapping keys.
- Bounded YAML aliases and expansion.
- A portable Agent declaration.

Invalid YAML blocks import.

## 8.3 Validation order

Preview performs:

```text
Read selected file
  -> calculate source digest
  -> canonicalize YAML
  -> validate portable Agent schema and raw-input managed profile
  -> normalize managed dependency scopes
  -> validate normalized managed profile and Agent declaration
  -> validate contained Text and inline MCP declarations
  -> calculate managed Definition
  -> inspect destination and identity conflicts
  -> inspect current managed dependency resolution
  -> sign prepared payload
```

`RestoredMemberships` and MCP setup descriptors are signed prepare-time
descriptive snapshots. They are returned from the prepared operation and are
not recomputed or used as commit correctness witnesses.

## 8.4 Import-blocking versus informational diagnostics

| Condition                       | Severity     | Blocks import        |
| ------------------------------- | ------------ | -------------------- |
| Invalid YAML                    | Error        | Yes                  |
| Invalid Agent root              | Error        | Yes                  |
| Invalid contained Text          | Error        | Yes                  |
| Invalid inline MCP              | Error        | Yes                  |
| Unsupported member type or form | Error        | Yes                  |
| Invalid destination Collection  | Error        | Yes                  |
| Name or package conflict        | Error        | Yes                  |
| Invalid prepared payload        | Error        | Yes                  |
| Built-in target unavailable     | Warning      | No                   |
| Mapped target unavailable       | Warning      | No                   |
| Named dependency ambiguous      | Warning      | No                   |
| External target malformed       | Warning      | No                   |
| Tool auto-execute               | Confirmation | No, after acceptance |
| Inline stdio MCP                | Confirmation | No, after acceptance |

## 9. Preview and signed preparation

## 9.1 Preview request

Conceptually:

```text
AgentImportPreviewRequest {
  Path
  Collection
  ExpectedCollectionRevision
  ExpectedSourceDigest?
}
```

The selected Collection identifies:

- Destination Root.
- Managed Agent Source.
- Collection revision.
- Collection package location.

## 9.2 Preview result

Conceptually:

```text
AgentImportPreview {
  Prepared?
  PreparedFingerprint?
  ExpiresAt?

  SourceDigest
  DefinitionDigest
  NormalizedYAML

  Agent
  Destination
  ProjectedArtifacts
  Relationships
  Conflicts
  RestoredMemberships
  Issues

  CanImport
  RequiresConfirmation
  RequiredConfirmationCodes
}
```

`ProjectedArtifacts` describes declaration identities expected from the managed Agent package.

It is not an Artifact Store overlay and does not contain durable `ArtifactRef` values before publication.

`Relationships` reports current managed lookup observations. It does not determine import validity.

## 9.3 Signed prepared payload

The prepared payload contains enough data to deterministically recreate the import operation:

- Normalized canonical Agent declaration.
- Source digest.
- Definition digest.
- Destination Root and Source.
- Selected Collection ref.
- Expected Collection revision.
- Expected Source generation.
- Managed package address.
- Managed Agent document locator.
- Required confirmation codes.
- Expiration time.

The package file list is deterministic from the normalized declaration and managed package layout. It does not need to be separately retained.

The prepared payload does not contain:

- The original import path.
- Secret values.
- OAuth tokens.
- MCP installation values.
- Backend-only prepared state.
- Dependency witnesses.
- Dependency revision snapshots.
- Runtime readiness state.

## 9.4 Stateless preparation behavior

The backend signs the prepared payload using a process-local HMAC key.

Commit requires:

- The signed payload.
- The prepared fingerprint.
- Accepted confirmation codes.

A restart invalidates pending prepared payloads because the signer key is process-local.

Replay behavior is intentionally simple:

- A replay may return a conflict after successful import.
- The application may require a new preview after a failed or uncertain commit.
- No replay receipt is required.

## 10. Best-effort commit behavior

## 10.1 Commit validation

Commit must:

- Verify the HMAC signature.
- Verify the prepared fingerprint.
- Reject expired payloads.
- Verify accepted confirmation codes.
- Revalidate the selected Collection revision.
- Revalidate managed Source generation.
- Revalidate Agent identity conflicts.
- Revalidate managed package conflicts.
- Revalidate Collection membership conflicts.
- Revalidate the normalized managed Agent declaration.

Commit must not:

- Reread the original file.
- Accept replacement YAML from the client.
- Accept replacement package bytes from the client.
- Re-resolve dependencies as an import gate.
- Require dependency witnesses to remain unchanged.
- Require MCP installation completion.
- Attempt MCP connection.

## 10.2 Commit sequence

The intended best-effort sequence is:

```text
Open and verify signed payload
  -> validate destination state and conflicts
  -> ensure desired Collection relationship
  -> publish managed Agent package
  -> refresh or reconcile normal managed Source state
  -> return Agent and Collection state
```

The Collection relationship is written first because it represents the selected user intent.

If Agent publication fails after membership creation:

- The Collection relationship remains declared.
- The relationship is unavailable.
- The user can correct the issue and preview again.
- No rollback is required.

## 10.3 No cross-package transaction requirement

The application does not require a transaction covering:

- Collection package mutation.
- Agent package mutation.
- Source reconciliation.
- Artifact verification.

Each operation must use ordinary Artifact Store consistency guarantees.

The application must not claim that the combined operation is atomic.

## 11. Collection membership behavior

## 11.1 Explicit destination

Every import requires an explicit editable managed Agent Collection.

There is no implicit baseline destination.

The selected Collection must:

- Belong to a non-protected user Root.
- Be an Agent Collection.
- Use the managed Agent Source.
- Be mutable.
- Have the expected revision.
- Use an enabled managed Source.

Collection enabled state does not gate import.

## 11.2 Persisted relationship

Import creates or preserves a located external Agent relationship.

Conceptually:

```yaml
- type: agent
  name: imported-agent
  locator: agent/imported-agent/unversioned/agent.yaml
```

The relationship must not contain an `ArtifactRef`.

## 11.3 Existing relationships

The selected Collection may already contain:

- The exact desired relationship.
- A dangling exact relationship from a deleted Agent.
- An incompatible same-name named relationship.
- A selector.
- A contained Agent declaration.

Behavior:

| Existing Collection state                    | Import behavior                           |
| -------------------------------------------- | ----------------------------------------- |
| Exact desired dangling relationship          | Reuse relationship and publish Agent      |
| Exact desired available relationship         | Conflict because Agent already exists     |
| Same Agent name at another locator           | Conflict                                  |
| Same Agent name as symbolic member           | Conflict                                  |
| Unrelated selector                           | Does not block import                     |
| Unrelated contained Agent                    | Does not block import                     |
| Other Collection exact dangling relationship | Becomes available when Agent is published |

The importer must not reject an import merely because the selected Collection contains an unrelated selector or unrelated contained Agent.

## 11.4 Membership does not own the Agent

Collection membership does not:

- Own the Agent package.
- Delete the Agent when detached.
- Change Agent enablement.
- Change Agent metadata.
- Change MCP installation state.
- Change Agent runtime state.

Deleting an Agent leaves Collection relationships declared and unavailable.

Re-importing the same managed Agent name can restore exact dangling relationships.

## 12. MCP handling

## 12.1 Import validates declaration shape only

Agent import validates:

- Named MCP relationship syntax.
- Inline MCP declaration syntax.
- MCP transport configuration.
- Installation input definitions.
- Authentication declaration shape.
- Policy reference syntax.
- Prohibited local state and secret placement.

Agent import does not validate:

- Whether a named MCP currently exists.
- Whether installation inputs have values.
- Whether secrets exist.
- Whether OAuth setup is complete.
- Whether the MCP can connect.
- Whether the MCP is currently healthy.

## 12.2 Named MCP relationships

A named MCP is valid even when unavailable.

Example:

```yaml
- type: mcp
  name: github
```

Preview normalizes this to:

```yaml
- type: mcp
  name: github
  scope: builtin
```

Preview may report the relationship as unavailable. Import remains allowed.

## 12.3 Inline MCP declarations

Inline MCP declarations are part of the imported Agent and must be valid.

After publication, contained MCP declarations receive normal Artifact occurrences and can be resolved through the Agent capability plan.

The UI may then use the resulting MCP Artifact refs to:

- Show setup requirements.
- Request installation values.
- Save secret references.
- Perform OAuth setup.
- Attempt connection.
- Show connection and runtime state.

## 12.4 Secret handling

Portable Agent YAML must not contain secret values.

The current managed inline MCP validator performs partial declaration-level
credential placement validation for recognized sensitive environment and header
positions, including:

- Sensitive environment variables.
- Sensitive headers.
- Sensitive connection profile headers.
- Sensitive connection profile environment variables.

URL, query-string, and command-argument credential classification remains MCP consumer and installation validation work. The import API must not represent
this partial validation as complete transport-secret detection.

The HMAC-signed prepared payload is not encrypted. This rule prevents secret values from being exposed through it.

## 13. Conflict handling

## 13.1 Root Agent identity

Managed Agent identity is:

```text
Destination Root
  + type: agent
  + logical name
```

The logical name determines the managed package address.

Import blocks when:

- An Agent with the same identity already exists.
- A missing but unpurged conflicting Agent occurrence exists.
- The target managed package address is occupied.
- Managed storage reports a case-folded package collision.
- The name collides with a protected built-in Agent.
- The package would require replacement.
- The name violates managed package naming rules.

Automatic rename is not allowed.

## 13.2 Contained Artifact identity

Contained declarations must not conflict with existing target Root Artifacts.

Text identity includes:

```text
text
  + name
  + insert
```

MCP identity includes:

```text
mcp
  + name
```

Contained Artifact conflicts block import because the managed package cannot safely publish the expected declarations.

## 13.3 Dependency availability is not a conflict

These are not import conflicts:

- Missing built-in Model.
- Missing built-in Tool.
- Missing built-in Skill.
- Missing built-in MCP.
- Missing built-in MCP Policy.
- Missing mapped Model target.
- Missing mapped Tool target.
- Ambiguous external dependency.

They are relationship diagnostics.

## 13.4 Package publication remains create-only

Managed Agent import is create-only.

The importer must not:

- Replace a package.
- Merge package content.
- Upsert a package.
- Overwrite equivalent content from another preview.
- Rename automatically.

A new import preview for an already imported Agent name reports a conflict.

## 14. Managed Agent export

## 14.1 Export eligibility

Managed Agent export is available only for an Agent that is:

- In a non-protected user Root.
- Available.
- Top-level.
- Backed by the managed Agent Source.
- Backed by the managed Agent package kind.
- Stored at the expected managed Agent document locator.
- A concrete Agent, not a source-selected alias.
- Valid against the portable Agent contract.
- Valid against the managed Agent profile.

Dependency resolution is not an export eligibility requirement.

An Agent remains exportable when:

- Named relationships are unavailable.
- Named relationships are ambiguous.
- MCP installation is incomplete.
- Secrets are missing.
- MCP connection is unavailable.

## 14.2 Export content

Export returns normalized portable YAML.

It includes:

- Type.
- Name.
- Source-authored display name.
- Source-authored description.
- Source-authored labels.
- Source-authored annotations.
- Normalized `scope: builtin` relationships.
- Contained Text declarations.
- Contained MCP declarations.
- Portable MCP installation input definitions.

It excludes:

- Collection membership.
- Collection identity.
- Root ID.
- Source ID.
- Artifact ID.
- Package address.
- Artifact revision fields in YAML.
- Definition digest fields in YAML.
- Artifact enablement.
- MCP installation values.
- Secret references.
- Secret values.
- OAuth tokens.
- Selected connection profiles.
- Runtime connection state.
- Agent readiness state.

## 14.3 Export result

Conceptually:

```text
AgentExportResult {
  Type
  Name
  MediaType
  SuggestedFileName
  Content
  ContentDigest
  DefinitionDigest
  ArtifactRevision
}
```

Resolution details may be returned as optional informational data, but export must not fail because relationship resolution is incomplete.

## 15. API shape

## 15.1 Import destinations

```text
ListAgentImportDestinations()
  -> []AgentImportDestination
```

Destinations are existing editable managed Agent Collections.

## 15.2 Preview

```text
PreviewAgentImport(
  AgentImportPreviewRequest
) -> AgentImportPreview
```

Conceptually:

```text
AgentImportPreviewRequest {
  Path
  Collection
  ExpectedCollectionRevision
  ExpectedSourceDigest?
}
```

```text
AgentImportPreview {
  Prepared?
  PreparedFingerprint?
  ExpiresAt?

  SourceDigest
  DefinitionDigest
  NormalizedYAML

  Agent
  Destination
  ProjectedArtifacts
  Relationships
  Conflicts
  RestoredMemberships
  Issues

  CanImport
  RequiresConfirmation
  RequiredConfirmationCodes
}
```

## 15.3 Issues

```text
AgentImportIssue {
  Code
  Severity
  Path?
  Message
}
```

Severity values are:

```text
error
confirmation
warning
information
```

Meaning:

| Severity       | Meaning                                                  |
| -------------- | -------------------------------------------------------- |
| `error`        | Blocks import                                            |
| `confirmation` | Requires explicit user acceptance                        |
| `warning`      | Does not block import, including unresolved dependencies |
| `information`  | Describes normalization or resulting behavior            |

## 15.4 Commit

```text
CommitAgentImport(
  AgentImportCommitRequest
) -> AgentImportCommitResult
```

```text
AgentImportCommitRequest {
  Prepared
  PreparedFingerprint
  AcceptedConfirmationCodes
}
```

```text
AgentImportCommitResult {
  Agent
  Collection
  RestoredMemberships
  PreparedFingerprint
}
```

Commit does not accept:

- A file path.
- YAML content.
- Package bytes.
- A replacement Collection document.
- Dependency witnesses.
- MCP secrets.
- MCP installation values.

## 15.5 Export

```text
ExportManagedAgent(
  AgentExportRequest
) -> AgentExportResult
```

```text
AgentExportRequest {
  Agent
}
```

## 15.6 Post-import resolution

The UI uses existing Agent APIs after commit:

```text
GetAgent
ResolveAgent
ResolveAgentCapabilities
```

The returned capability plan is the authoritative post-publication relationship state.

No Agent readiness API is required.

## 16. UI and runtime responsibilities

## 16.1 Import UI

The import UI owns:

- File selection.
- Destination Collection selection.
- Preview display.
- Confirmation display.
- Commit initiation.
- Display of dangling membership behavior.
- Re-preview after stale or failed commit.

The import UI does not need:

- A reference catalog.
- A field-by-field Agent editor.
- Backend readiness state.
- Backend MCP setup orchestration.

## 16.2 Post-import Agent UI

After successful import, the Agent UI may:

- Resolve the Agent.
- Show available, unavailable, and ambiguous relationships.
- Show normalized YAML.
- Show dangling Collection memberships.
- Export the Agent.
- Enable or disable the Agent.
- Delete the Agent.

## 16.3 MCP UI

After resolving an Agent, the MCP UI may:

- Identify resolved MCP Artifact refs.
- Inspect MCP installation input definitions.
- Collect local installation values.
- Save secret references.
- Perform OAuth setup.
- Select profiles.
- Attempt connection.
- Show runtime state.

These actions do not modify the Agent declaration.

## 17. Security and persistence boundaries

## 17.1 Persisted state

The feature persists normal Artifact ecosystem state:

- Managed Agent package.
- Collection package mutations.
- Definitions.
- Artifacts.
- Source generation.
- Artifact enablement.
- MCP local installation data, through MCP APIs.
- Secret references, through secret and MCP APIs.

## 17.2 Non-persisted state

The feature does not persist:

- Original import path.
- Backend prepared-import records.
- Commit tokens.
- Source file contents outside the managed package.
- Dependency witnesses.
- Dependency revision snapshots.
- Agent readiness.
- MCP installation values in Agent YAML.
- Secret values in Agent YAML.
- OAuth tokens in Agent YAML.
- Collection membership inside Agent YAML.

## 17.3 Signed payload boundary

The signed prepared payload may be visible to the client.

Therefore:

- It must not contain secrets.
- It must not contain private filesystem paths.
- It must not contain local MCP state.
- It must not contain backend-only credentials.
- It must expire.
- It must be HMAC-authenticated before commit.

## 18. Breaking removals

The following legacy capabilities are removed.

### 18.1 Assistant Presets

Removed:

- Assistant Preset APIs.
- Assistant Preset storage.
- Assistant Preset overlays.
- Assistant Preset versions.
- Assistant Preset bundle IDs.
- Assistant Preset migrations.
- Assistant Preset frontend routes.
- Assistant Preset compatibility aliases.
- Assistant Preset dual-read and dual-write behavior.

### 18.2 Structured managed Agent authoring

Removed:

- `ManagedAgentDocument`.
- `ManagedAgentMember`.
- `ManagedAgentCreateRequest`.
- `ManagedAgentCreateResult`.
- `ManagedAgentReplaceRequest`.
- `ManagedAgentReplaceResult`.
- `CreateManagedAgent`.
- `ReplaceManagedAgent`.
- Field-by-field Model selection.
- Field-by-field Tool selection.
- Field-by-field Skill selection.
- Field-by-field MCP selection.
- In-place Agent declaration editing.
- Managed Agent replacement.

### 18.3 Retained operations

These remain:

- Agent import.
- Agent export.
- Agent delete.
- Agent enablement.
- Agent reads.
- Agent resolution.
- Collection create, read, update, enablement, and delete.
- Built-in Agent installation.
- MCP local installation and runtime flows.

## 19. Implementation requirements

## 19.1 Managed scope normalization

The implementation must add a normalization step before Definition creation.

For every allowed named external managed relationship:

```text
scope omitted
  -> normalize to scope: builtin

scope: builtin
  -> retain scope: builtin
```

The normalized declaration must be used for:

- Definition digest.
- Managed package bytes.
- Preview normalized YAML.
- Signed prepared payload.
- Exported YAML.

## 19.2 Non-blocking dependency preflight

`preflightManagedAgentDependencies` must stop producing import-blocking errors for ordinary resolution failures.

Instead:

- Resolve through managed lookup.
- Record `available`, `unavailable`, or `ambiguous`.
- Record built-in or mapped provenance when available.
- Return warnings for resolution failures.
- Do not prevent `CanImport`.
- Do not create dependency witnesses for commit.

Infrastructure failures such as context cancellation, closed Store state, or invalid internal configuration may remain operation errors.

## 19.3 Commit must stop checking dependency witnesses

`preparedDependencyWitness` and `verifyDependencyWitnesses` are not part of this design.

Commit correctness is based on:

- Signed normalized declaration.
- Destination Collection revision.
- Managed Source generation.
- Agent and package conflicts.
- Collection membership conflicts.
- Required confirmations.

It is not based on external dependency state.

## 19.4 Validation errors must become preview issues

Malformed user-authored declarations must be returned as preview `error` issues where possible.

This includes failures from:

- Contained Text decoding.
- Inline MCP decoding.
- Inline MCP semantic validation.
- Managed scope normalization.
- Package layout generation.

They should not normally escape as opaque operation errors.

## 19.5 Secret validation must be tightened

The current sensitive-header and sensitive-environment heuristic is insufficient.

The managed inline MCP validator must reject literal secret-like values in recognized credential-bearing positions, including:

- Command arguments.
- URL query values.
- URL user information.
- Headers.
- Environment variables.
- Connection profile overrides.

Only declared installation-input references may identify secrets.

## 19.6 Export must not require successful resolution

`ExportManagedAgent` must export a valid managed Agent from its immutable Definition even when relationships are unavailable.

Resolution may be returned as optional information, but it must not be a hard export prerequisite.

## 19.7 No implementation required for excluded features

The following do not require implementation:

- Backend-held prepared import registry.
- Random opaque commit token.
- Prepared import replay receipt.
- Projected Artifact overlay.
- Cross-package import transaction.
- Agent readiness aggregate.
- Agent MCP completion aggregate.
- Agent reference catalog.

## 20. Current implementation status

| Capability                                  | Status       | Notes                                                                             |
| ------------------------------------------- | ------------ | --------------------------------------------------------------------------------- |
| Portable Agent schema                       | Available    | Existing `agentv1` contract remains unchanged                                     |
| Managed Agent restrictions schema           | Available    | Raw input permits omitted built-in scope; normalized storage is revalidated       |
| Managed-scope normalization                 | Available    | Normalized `scope: builtin` is used for digest, publication, and export           |
| Conjunctive strict profile validation       | Available    | Existing `allOf(base, restrictions)` implementation                               |
| Safe YAML preview entry point               | Available    | Uses `llmtoolsutil.ReadPortableTextFile`; underlying safety requires verification |
| Explicit Collection destination             | Available    | Existing import destination logic                                                 |
| Client-carried signed prepared payload      | Available    | Existing HMAC signer design matches this HLD                                      |
| Backend prepared-import registry            | Out of scope | Intentionally not required                                                        |
| Projected Artifact overlay                  | Out of scope | Intentionally not required                                                        |
| Cross-package atomic import transaction     | Out of scope | Best-effort membership and publication is intentional                             |
| Managed package create-only publication     | Available    | Existing `AllowPackageReplacement: false` behavior                                |
| Empty Agent import                          | Available    | Existing behavior is retained intentionally                                       |
| Validation of contained Text and inline MCP | Available    | User-authored content failures are returned as structured preview issues          |
| Non-blocking external dependency resolution | Available    | Unavailable and ambiguous dependencies are warnings, not import gates             |
| Built-in-first mapped managed lookup        | Available    | Normalized scope prevents destination-Root dependency lookup                      |
| Dependency witness enforcement              | Removed      | Prepared payloads and commit no longer capture or verify dependency state         |
| Warning severity and relationship reporting | Available    | Preview returns warning issues and statusful relationship observations            |
| Inline MCP literal-secret protection        | Partial      | Sensitive env/header placements are guarded; broader MCP validation is deferred   |
| Collection conflict filtering               | Available    | Only conflicting direct same-name Agent relationships block import                |
| Managed package-address conflict preflight  | Available    | Preview and commit inspect the target managed package directory                   |
| Post-import normal Agent resolution         | Available    | Existing `ResolveAgent` and capability APIs                                       |
| Agent readiness aggregate                   | Out of scope | UI and MCP subsystem responsibility                                               |
| MCP installation completion during import   | Out of scope | UI and MCP subsystem responsibility                                               |
| Agent reference catalog                     | Out of scope | Optional future UI convenience only                                               |
| Managed Agent export                        | Available    | Export reads the immutable Definition; resolution is optional information         |
| Managed Agent delete preserving membership  | Available    | Existing lifecycle behavior                                                       |
| Structured managed Agent create and replace | Removed      | Must remain absent                                                                |
| Assistant Preset backend and compatibility  | Removed      | Must remain absent                                                                |
