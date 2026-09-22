Proposed file: `devdocs/requirements/managed_artifact_import_export_hld.md`

# Managed Artifact Import and Export Extension HLD

Status: Proposed breaking replacement for managed Agent composition authoring and Assistant Preset management

Normative foundations:

- [Portable Artifact Declaration Contracts HLD](./artifact_contracts_hld.md)
- [Artifact Store, Resolution, and Ecosystem HLD](./artifacts_hld.md)
- [Artifact-backed Agent Store HLD](./agent_store_hld.md)

This HLD extends the existing Artifact ecosystem with previewed, strict-profile import and portable export for user-managed Artifacts.

The first supported domain is Agent.

The common infrastructure must remain reusable for MCP and Text, but each domain remains the public entry point and owns its admission, destination, lifecycle, conflict, and readiness policies.

- [1. Purpose](#1-purpose)
- [2. Goals and scope](#2-goals-and-scope)
  - [2.1 Goals](#21-goals)
  - [2.2 In scope](#22-in-scope)
  - [2.3 Out of scope](#23-out-of-scope)
- [3. Normative relationship to existing HLDs](#3-normative-relationship-to-existing-hlds)
- [4. Terminology](#4-terminology)
- [5. Requirements](#5-requirements)
  - [5.1 Domain ownership requirements](#51-domain-ownership-requirements)
  - [5.2 File input requirements](#52-file-input-requirements)
  - [5.3 Strict profile schema requirements](#53-strict-profile-schema-requirements)
  - [5.4 Preview requirements](#54-preview-requirements)
  - [5.5 Prepared import requirements](#55-prepared-import-requirements)
  - [5.6 Commit requirements](#56-commit-requirements)
  - [5.7 Atomicity requirements](#57-atomicity-requirements)
  - [5.8 Export requirements](#58-export-requirements)
  - [5.9 Managed declaration immutability requirements](#59-managed-declaration-immutability-requirements)
  - [5.10 Collection requirements](#510-collection-requirements)
  - [5.11 Identity and conflict requirements](#511-identity-and-conflict-requirements)
  - [5.12 Reference catalog requirements](#512-reference-catalog-requirements)
  - [5.13 MCP completion and Agent readiness requirements](#513-mcp-completion-and-agent-readiness-requirements)
  - [5.14 Breaking-change requirements](#514-breaking-change-requirements)
- [6. Design principles and invariants](#6-design-principles-and-invariants)
  - [6.1 Preview is authoritative preparation](#61-preview-is-authoritative-preparation)
  - [6.2 Commit does not reread the import source](#62-commit-does-not-reread-the-import-source)
  - [6.3 A digest is identity, not frontend authority](#63-a-digest-is-identity-not-frontend-authority)
  - [6.4 Strict profile is not a portable schema version](#64-strict-profile-is-not-a-portable-schema-version)
  - [6.5 Importability and readiness are separate](#65-importability-and-readiness-are-separate)
  - [6.6 Export is declaration recovery, not Store backup](#66-export-is-declaration-recovery-not-store-backup)
  - [6.7 Collection membership does not own the Agent](#67-collection-membership-does-not-own-the-agent)
  - [6.8 Domains drive common infrastructure](#68-domains-drive-common-infrastructure)
- [7. Architecture and ownership](#7-architecture-and-ownership)
  - [7.1 Conceptual architecture](#71-conceptual-architecture)
  - [7.2 Component responsibilities](#72-component-responsibilities)
  - [7.3 Common infrastructure and domain drivers](#73-common-infrastructure-and-domain-drivers)
- [8. Strict managed profile schemas](#8-strict-managed-profile-schemas)
  - [8.1 Schema purpose](#81-schema-purpose)
  - [8.2 Subset by construction](#82-subset-by-construction)
  - [8.3 Initialization-time validation](#83-initialization-time-validation)
  - [8.4 Schema and semantic admission](#84-schema-and-semantic-admission)
  - [8.5 Schema placement](#85-schema-placement)
- [9. Agent managed import profile](#9-agent-managed-import-profile)
  - [9.1 Agent root declaration](#91-agent-root-declaration)
  - [9.2 Allowed Agent members](#92-allowed-agent-members)
  - [9.3 Prohibited Agent forms](#93-prohibited-agent-forms)
  - [9.4 Contained Text](#94-contained-text)
  - [9.5 Model and Tool relationships](#95-model-and-tool-relationships)
  - [9.6 Skill relationships](#96-skill-relationships)
  - [9.7 MCP relationships](#97-mcp-relationships)
  - [9.8 Inline MCP declarations](#98-inline-mcp-declarations)
  - [9.9 Metadata and local state](#99-metadata-and-local-state)
- [10. Safe file input](#10-safe-file-input)
- [11. Agent import preview flow](#11-agent-import-preview-flow)
  - [11.1 Preview request](#111-preview-request)
  - [11.2 Preview processing](#112-preview-processing)
  - [11.3 Projected Artifact resolution](#113-projected-artifact-resolution)
  - [11.4 Preview result](#114-preview-result)
  - [11.5 Revalidation](#115-revalidation)
- [12. Prepared import trust model](#12-prepared-import-trust-model)
  - [12.1 Backend-held prepared state](#121-backend-held-prepared-state)
  - [12.2 Digests and fingerprint](#122-digests-and-fingerprint)
  - [12.3 Commit token](#123-commit-token)
  - [12.4 Expiration and replay](#124-expiration-and-replay)
  - [12.5 Environmental witnesses](#125-environmental-witnesses)
- [13. Identity and conflict handling](#13-identity-and-conflict-handling)
  - [13.1 Root Agent identity](#131-root-agent-identity)
  - [13.2 Contained Artifact identities](#132-contained-artifact-identities)
  - [13.3 Package conflicts](#133-package-conflicts)
  - [13.4 Collection relationship conflicts](#134-collection-relationship-conflicts)
  - [13.5 Built-in conflicts](#135-built-in-conflicts)
  - [13.6 Delete and re-import](#136-delete-and-re-import)
- [14. Atomic import commit](#14-atomic-import-commit)
  - [14.1 Transaction contents](#141-transaction-contents)
  - [14.2 Commit sequence](#142-commit-sequence)
  - [14.3 Durable result and recovery](#143-durable-result-and-recovery)
- [15. Portable managed export](#15-portable-managed-export)
  - [15.1 Export eligibility](#151-export-eligibility)
  - [15.2 Export content](#152-export-content)
  - [15.3 Export exclusions](#153-export-exclusions)
  - [15.4 Export and re-import](#154-export-and-re-import)
- [16. Agent Collections and lifecycle](#16-agent-collections-and-lifecycle)
- [17. Agent readiness and MCP completion](#17-agent-readiness-and-mcp-completion)
  - [17.1 Readiness dimensions](#171-readiness-dimensions)
  - [17.2 Inline MCP completion](#172-inline-mcp-completion)
  - [17.3 Secret handling](#173-secret-handling)
  - [17.4 Readiness derivation](#174-readiness-derivation)
- [18. User interface flow](#18-user-interface-flow)
  - [18.1 Agent management page](#181-agent-management-page)
  - [18.2 Import dialog](#182-import-dialog)
  - [18.3 Preview presentation](#183-preview-presentation)
  - [18.4 Confirmation and commit](#184-confirmation-and-commit)
  - [18.5 Export flow](#185-export-flow)
  - [18.6 Delete and re-import flow](#186-delete-and-re-import-flow)
- [19. Consumer API shape](#19-consumer-api-shape)
  - [19.1 Agent import destination](#191-agent-import-destination)
  - [19.2 Agent import preview](#192-agent-import-preview)
  - [19.3 Agent import commit](#193-agent-import-commit)
  - [19.4 Agent export](#194-agent-export)
  - [19.5 Agent reference catalog](#195-agent-reference-catalog)
  - [19.6 Agent readiness](#196-agent-readiness)
- [20. Persistence and security boundaries](#20-persistence-and-security-boundaries)
- [21. Breaking removals](#21-breaking-removals)
  - [21.1 Assistant Preset removal](#211-assistant-preset-removal)
  - [21.2 Managed Agent authoring removal](#212-managed-agent-authoring-removal)
  - [21.3 Behaviors intentionally not retained](#213-behaviors-intentionally-not-retained)
- [22. Implementation areas](#22-implementation-areas)
  - [22.1 Contract profile infrastructure](#221-contract-profile-infrastructure)
  - [22.2 Common managed transfer infrastructure](#222-common-managed-transfer-infrastructure)
  - [22.3 Artifact Store transaction support](#223-artifact-store-transaction-support)
  - [22.4 Agent domain implementation](#224-agent-domain-implementation)
  - [22.5 Application composition and wrappers](#225-application-composition-and-wrappers)
  - [22.6 Future MCP and Text reuse](#226-future-mcp-and-text-reuse)
- [23. Implementation status](#23-implementation-status)

## 1. Purpose

This HLD defines a managed Artifact import and export workflow that replaces field-by-field managed Agent composition.

The primary Agent import flow is:

```text
User-selected Agent YAML path
  -> safe bounded file read
  -> portable Agent validation
  -> strict managed Agent profile validation
  -> projected managed package and Artifact preparation
  -> relationship resolution and conflict analysis
  -> immutable backend-held preview
  -> explicit user confirmation
  -> atomic managed Source commit
  -> Agent Artifact and Collection membership
```

The primary Agent export flow is:

```text
Managed Agent Artifact
  -> current immutable Definition
  -> strict managed profile validation
  -> normalized portable YAML
  -> user-selected download destination
```

The workflow is based on portable declarations. It does not preserve Assistant Preset APIs, storage, identifiers, versions, or behaviors.

## 2. Goals and scope

### 2.1 Goals

The design must:

- Replace managed Agent composition forms with import of complete portable Agent files.
- Present the complete Agent Artifact instead of asking users to select members one at a time.
- Require an explicit user Agent Collection destination.
- Validate the selected file without modifying Artifact Store.
- Return a complete preview and clear import instructions before confirmation.
- Prepare the exact managed package and Artifact projection during preview.
- Commit the previously prepared content without reading the external file again.
- Prevent frontend modification of prepared content from changing the committed Artifact.
- Commit Agent package creation and Collection membership atomically.
- Export the current portable Definition of a user-managed Agent.
- Disallow in-place Agent declaration editing and replacement.
- Support delete-and-re-import or import-under-a-new-name workflows.
- Provide a strict managed Agent schema that is a subset of the portable Agent schema.
- Validate strict-profile registration at application initialization.
- Expose the built-in and mapped names accepted by the managed Agent profile.
- Permit inline MCP declarations while keeping secrets and local installation values outside portable content.
- Derive Agent readiness from relationship resolution and required local MCP completion.
- Provide common infrastructure reusable by MCP and Text import/export.
- Keep Agent, MCP, and Text domains as their own public entry points and policy owners.
- Avoid declaration or package version changes.

### 2.2 In scope

This HLD defines:

- Common strict managed profile registration.
- Safe import file reading.
- Agent import preview.
- Backend-held prepared imports.
- Source, Definition, and prepared-plan fingerprints.
- Agent import confirmation.
- Atomic managed package and Collection mutation.
- Agent-specific strict admission rules.
- Agent name and package conflict behavior.
- Managed Agent portable export.
- Managed Agent declaration immutability.
- Built-in and mapped reference catalogs.
- Inline MCP setup requirements.
- Derived Agent readiness.
- Breaking removal of Assistant Presets.
- Removal of structured managed Agent creation and replacement.

### 2.3 Out of scope

This HLD does not define:

- A visual Agent composition editor.
- An in-application YAML editor.
- In-place managed Agent declaration editing.
- Managed Agent replacement.
- Managed Agent release versions.
- Linked-file synchronization.
- Watching the imported source path.
- Repository Source registration from the import path.
- Import from URL, Git, package, command, or archive locators.
- Assistant Preset compatibility.
- Assistant Preset migration.
- Assistant Preset dual-read or dual-write.
- Export of built-in, repository-authored, or contained child Artifacts as independent managed files.
- Export of Agent Collection membership as part of an Agent declaration.
- Agent execution.
- MCP connection execution during import.
- Persistence of secrets in the preview, portable declaration, or export.
- A generic frontend import endpoint that bypasses domain policy.

## 3. Normative relationship to existing HLDs

This HLD preserves the portable declaration and Artifact Store models defined by the parent HLDs.

It changes the managed Agent authoring behavior defined by the Agent Store HLD.

The following Agent Store behaviors are superseded:

- Structured managed Agent creation through `ManagedAgentDocument`.
- Field-by-field member construction through `ManagedAgentMember`.
- Managed Agent replacement.
- In-place update semantics for a managed Agent declaration.
- Any frontend composition editor for managed Agents.

The following Agent Store behaviors remain:

- Agent Collections are Agent-only Plugin Artifacts.
- Import requires an explicit Agent Collection.
- Agent and Collection enablement remain independent.
- An Agent can belong to several Collections.
- Collection membership does not own the Agent.
- Deleting an Agent does not implicitly detach Collection relationships.
- Deleting a Collection does not delete Agents.
- Built-in Agents remain protected.
- Local management operations may use authorized `ArtifactRef` values.
- Portable declarations must not persist Store identity.
- Repository-authored Agents may use the complete portable Agent contract.

No existing portable schema version, declaration version, package version, or Agent release version is changed by this extension.

The strict managed profile is an additional admission schema. It is not a new portable Agent schema version and does not participate in portable declaration dispatch.

## 4. Terminology

| Term                       | Meaning                                                                                       |
| -------------------------- | --------------------------------------------------------------------------------------------- |
| Portable declaration       | A declaration accepted by the normative Artifact contract                                     |
| Strict managed profile     | A domain-owned subset of a portable declaration accepted for managed import and export        |
| Import source              | The external user-selected file read during preview                                           |
| Source digest              | Digest of the exact bytes read from the external file                                         |
| Portable Definition digest | Digest of the canonical portable declaration                                                  |
| Prepared import            | Immutable backend-held import state produced by successful preview                            |
| Prepared fingerprint       | Digest binding prepared bytes, destination, mutations, dependencies, and expected state       |
| Commit token               | Opaque capability identifying one prepared import                                             |
| Projected Artifact         | An in-memory Artifact occurrence expected from the prepared managed Source mutation           |
| Environmental witness      | Revision, digest, generation, or mapped-target identity that must remain current until commit |
| Importable                 | The prepared import has no blocking admission or conflict issue                               |
| Resolution complete        | Every required declaration relationship resolves uniquely                                     |
| Installation complete      | Every required local MCP installation input and secret reference is populated                 |
| Agent ready                | Resolution is complete and all required local completion checks pass                          |
| Managed export             | Portable Definition bytes produced from an eligible user-managed root Artifact                |

## 5. Requirements

### 5.1 Domain ownership requirements

- Agent APIs must remain the public entry point for Agent import and export.
- MCP APIs must remain the future public entry point for MCP import and export.
- Text APIs must remain the future public entry point for Text import and export.
- Common infrastructure must not decide domain admission rules.
- Common infrastructure must not decide domain destination rules.
- Common infrastructure must not decide Collection membership rules.
- Common infrastructure must not decide domain readiness.
- The Agent domain must provide the Agent strict profile, destination policy, conflict policy, package plan, reference catalog, and readiness projection.
- No generic public API may allow a caller to select an arbitrary declaration type and bypass its domain.

### 5.2 File input requirements

- Agent import preview must read the file through the safe filesystem tool adapter based on `llmtoolsutil.ReadFile`.
- Import must not use unrestricted `os.ReadFile`.
- The read must be symlink-safe, sandboxed, and cross-platform.
- The read capability must be limited to the user-selected file.
- The input must be a regular bounded text file.
- Agent import initially accepts YAML files only.
- The file must contain exactly one YAML document.
- The YAML root must be an object.
- Duplicate YAML mapping keys must be rejected.
- YAML alias or expansion behavior must remain bounded.
- The read file path is preview input only.
- The path must not be written into the portable declaration.
- The path must not be persisted as an Agent locator.
- The path must not be required during commit.
- The path must not be required after import.
- An optional caller-supplied expected source digest may be used during revalidation.
- The backend-calculated source digest is authoritative.

### 5.3 Strict profile schema requirements

- Each managed importable domain must register one strict profile against one portable base schema.
- The Agent strict profile must be registered against the current Agent schema.
- The strict profile must be a subset of the portable base schema by construction.
- A strict profile must not replace or modify the base schema.
- A strict profile must not add fields to portable declaration instances.
- A strict profile must not add a declaration version.
- A strict profile must not add a Store identity field.
- A strict profile must be compiled using JSON Schema Draft 2020-12.
- Application initialization must fail if a registered strict profile cannot be compiled.
- Application initialization must fail if the strict profile is not conjunctively bound to its declared base schema.
- Every imported document must pass both the base portable schema and the strict profile.
- Every exported document must pass the base portable schema and the strict profile.
- Semantic admission checks not expressible in JSON Schema must run after schema validation.

### 5.4 Preview requirements

Preview must:

- Require an explicit editable Agent Collection.
- Derive the target Root and managed Source from the selected Collection.
- Read the external file exactly once for that preview.
- Calculate the source digest.
- Decode and validate the portable Agent declaration.
- Validate the strict Agent profile.
- Validate Agent-specific semantic restrictions.
- Calculate the canonical portable Definition.
- Calculate the managed package address and source files.
- Calculate the exact Collection member relationship.
- Project every root and contained Artifact expected from the declaration.
- Resolve the projected Agent against the target Root, built-in Root, and registered fallback providers.
- Inspect name, package, Source, Collection, and contained identity conflicts.
- Identify existing dangling Collection relationships that the import would restore.
- Identify MCP installation and secret completion requirements.
- Determine whether the import is allowed.
- Determine whether user confirmation is required.
- Determine whether the resulting Agent would be ready immediately after import.
- Produce normalized portable YAML for display.
- Produce structured issues with severity, code, location, message, and remediation.
- Produce no persistent mutation.

A failed preview must not create:

- Source content.
- Artifact records.
- Collection membership.
- local MCP installation data.
- secret data.
- persistent preview records.

### 5.5 Prepared import requirements

A successful importable preview must create an immutable backend-held prepared import.

The prepared import must contain:

- The canonical portable Agent declaration.
- The normalized exportable YAML.
- The source digest.
- The portable Definition digest.
- The prepared managed package files.
- The projected root and contained Artifact expectations.
- The selected Root, Source, and Collection.
- The expected Collection revision.
- The expected managed Source generation.
- The planned Collection document mutation, if one is required.
- The Agent package address.
- Dependency witnesses.
- Conflict analysis.
- Required confirmation issue codes.
- Predicted readiness and setup requirements.
- A creation time and expiration time.

The prepared import must not contain:

- MCP secret values.
- OAuth tokens.
- resolved secret contents.
- arbitrary frontend-supplied declaration changes.
- a requirement to reread the external file.

### 5.6 Commit requirements

Commit must:

- Accept only an opaque commit token, prepared fingerprint, and explicit confirmations.
- Resolve the token to backend-held prepared state.
- Reject an unknown, expired, consumed, or mismatched token.
- Reject a fingerprint mismatch.
- Reject missing required confirmations.
- Revalidate Collection revision, Source generation, identity conflicts, and dependency witnesses.
- Not reread the external file.
- Not accept replacement declaration bytes from the frontend.
- Not accept a modified managed package from the frontend.
- Commit exactly the declaration and package prepared during preview.
- Return the imported Agent, Collection result, restored memberships, and current readiness.
- Be idempotent for replay of the same successfully committed token.
- Consume the token after a durable success.
- Reject a new import preview for an already present conflicting Agent name.

New imported Agents default to `Artifact.Enabled=true`.

Agent enablement may be changed later through the universal Artifact enablement API. It is not part of portable import or export content.

### 5.7 Atomicity requirements

The durable managed Source mutation must be atomic.

One Agent import transaction may include:

- Creation of the managed Agent package.
- Replacement of the selected managed Collection package with a document containing the new located Agent relationship.
- No Collection package change when the exact required relationship already exists.
- Reconciliation of the resulting Source generation.
- Verification of the expected Agent and contained Artifacts.

The transaction must not expose a durable state in which:

- A newly added selected Collection relationship exists without its prepared Agent package.
- The prepared Agent package exists but the required selected Collection membership was omitted.
- Only part of the prepared package files were published.
- A conflicting package was replaced.

A committed Source mutation followed by interrupted derived-state reconciliation must be recoverable and idempotently verifiable.

No Agent-specific transaction database or sidecar file may be introduced.

### 5.8 Export requirements

- Only a user-managed, top-level, concrete Agent may be exported through the managed Agent export API.
- Built-in Agents must not be exported through this API.
- Repository-authored Agents must not be exported through this API.
- Contained Agent child Artifacts must not be exported independently through this API.
- Export must read the current immutable Definition selected by the Agent Artifact.
- Export must validate the Definition against the portable Agent schema.
- Export must validate the Definition against the strict managed Agent profile.
- Export must produce normalized portable YAML.
- Export must provide a suggested portable filename.
- Export must return the Artifact revision and Definition digest used.
- Export must remain available when dependencies are currently unavailable.
- Export must report current resolution and readiness issues separately from export eligibility.
- Export must not include Agent Collection membership.
- Export must not include local enablement.
- Export must not include Root, Source, Artifact, package, or runtime identity.
- Export must not include MCP installation values, secret references, OAuth tokens, connection state, or discovery state.

### 5.9 Managed declaration immutability requirements

Managed Agent declaration content must not be edited in place.

The Agent management API must not expose:

- Agent declaration patch.
- Member add or remove.
- Agent rename.
- Agent replacement.
- Agent upsert.
- New Agent version creation.
- In-app declaration editor.

A user who wants changed content must:

- Export the current managed Agent.
- Modify the exported file externally.
- Give it a new Agent name and import it as another Agent.

Alternatively, the user may:

- Export the current managed Agent.
- Delete the current Agent.
- Modify the file.
- Import the file again using the same name.

Artifact enablement and MCP local completion are local-state operations and are not declaration edits.

### 5.10 Collection requirements

- Import must identify an explicit editable Agent Collection.
- There is no implicit baseline destination.
- The selected Collection must belong to a user Root.
- The selected Collection must use the managed Agent Source expected by the Agent domain.
- The selected Collection must be an Agent-only managed Plugin.
- Collection enablement must not gate import.
- Agent enablement must not follow Collection enablement.
- Import must add a located external Agent relationship.
- The persisted relationship must not contain an `ArtifactRef`.
- Existing exact dangling membership may be reused.
- Existing incompatible membership must be reported as a conflict.
- Importing one Agent must not remove or rewrite unrelated Collection members.
- Export must not infer ownership from Collection membership.
- Deleting an Agent must not automatically detach it from Collections.
- Re-importing the same deleted Agent package may restore all exact dangling relationships to that package occurrence.

### 5.11 Identity and conflict requirements

- The root Agent logical name is the managed package identity.
- Automatic renaming is prohibited.
- Silent replacement is prohibited.
- An existing available Agent with the same semantic identity in the target Root is a blocking conflict.
- An existing managed package at the calculated Agent package address is a blocking conflict.
- A case-folded physical package collision on a case-insensitive platform is a blocking conflict.
- An existing protected built-in Agent with the same logical name is a blocking conflict for strict managed import.
- Every contained Artifact identity emitted by the import must be checked for target-Root conflict.
- Text identity checks must include `insert`.
- A missing package with an exact dangling managed Collection locator is a restoration case, not a duplicate.
- A same-name Collection member that points elsewhere is a blocking conflict.
- A selector or contained Collection member that would overlap the imported name is a blocking conflict.
- New preview after successful import must report a name conflict.
- Replay of the same commit token must return the original result rather than create another Artifact.

### 5.12 Reference catalog requirements

The Agent domain must expose the names accepted by the strict Agent profile.

The catalog must include, where supported:

- Protected built-in Models.
- Protected built-in Tools.
- Protected built-in Skills.
- Protected built-in MCP servers.
- Protected built-in MCP policies.
- Mapped Model targets accepted by the current fallback provider.
- Mapped Tool targets accepted by the current fallback provider.

Each catalog item must expose:

- Artifact type.
- Logical name.
- Display name.
- Description where available.
- Built-in or mapped provenance.
- Supported Agent relationship behavior.
- Current declaration availability.
- A copyable YAML relationship snippet.

The catalog is informational.

It must not:

- Add a relationship to an Agent.
- Become a field-by-field composition editor.
- Persist target IDs.
- Treat runtime readiness as declaration availability.

Fallback providers that want to expose mapped targets in the catalog must implement a bounded catalog port in addition to exact-name fallback resolution.

### 5.13 MCP completion and Agent readiness requirements

- A strict managed Agent may contain an inline MCP declaration.
- MCP installation input definitions may be portable declaration content.
- MCP installation input values must not be portable declaration content.
- Secret values must not be portable declaration content.
- Secret values must not be included in export.
- Secret values must not be retained in prepared import state.
- Preview must identify required MCP installation inputs and secret requirements.
- Missing MCP completion must not invalidate an otherwise valid portable Agent declaration.
- Missing MCP completion must make the Agent not ready.
- An imported Agent may therefore be importable but not ready.
- Agent readiness must be derived, not persisted in the Agent Definition.
- Agent readiness must require complete required relationship resolution.
- Agent readiness must require all required MCP installation inputs to be populated.
- Agent readiness must require all required MCP secret references to resolve.
- Agent readiness must require applicable MCP authentication setup to be complete.
- Current active MCP connection state is not required for configuration readiness.
- Agent execution remains responsible for operational readiness.

### 5.14 Breaking-change requirements

The change is development-time breaking.

The implementation must not retain:

- Assistant Preset APIs.
- Assistant Preset wire types.
- Assistant Preset storage.
- Assistant Preset built-in overlays.
- Assistant Preset versions.
- Assistant Preset bundle IDs.
- Assistant Preset migration.
- Assistant Preset compatibility aliases.
- Assistant Preset frontend routes.
- Structured managed Agent authoring.
- Managed Agent replacement.
- Legacy dual-read or dual-write behavior.

No version bump, compatibility decoder, or migration layer is required.

## 6. Design principles and invariants

### 6.1 Preview is authoritative preparation

Preview is not only validation.

A successful preview prepares the exact managed mutation that commit will use.

```text
Previewed bytes
  -> canonical declaration
  -> package files
  -> Collection relationship
  -> projected Artifacts
  -> prepared import
```

Commit does not reconstruct this plan from frontend values.

### 6.2 Commit does not reread the import source

The external file is relevant only while producing a preview.

```text
Preview
  -> read file
  -> prepare immutable backend state

Commit
  -> consume prepared state
  -> no file read
```

Changing or deleting the original file after preview does not alter the prepared import.

The user must explicitly revalidate to produce a preview from changed file content.

### 6.3 A digest is identity, not frontend authority

A plain digest supplied by the frontend is not proof that frontend content was not modified.

The backend therefore retains the prepared bytes and plan. The frontend receives only an opaque commit token and fingerprint.

The frontend cannot replace the prepared declaration during commit.

### 6.4 Strict profile is not a portable schema version

The strict profile determines what this application is willing to manage through import and export.

It does not change what a valid portable Agent declaration is.

Repository-authored Agents continue to use the complete Agent contract.

### 6.5 Importability and readiness are separate

```text
Portable and strict-profile valid
  -> importable

Resolution complete
  + required local MCP setup complete
  -> ready
```

An Agent can be imported before it is ready.

### 6.6 Export is declaration recovery, not Store backup

Export returns portable declaration semantics.

It does not return:

- Artifact Store records.
- local state.
- Collection membership.
- secret state.
- runtime state.
- source package metadata.

### 6.7 Collection membership does not own the Agent

The selected Collection is the required import destination.

It does not become the Agent owner.

### 6.8 Domains drive common infrastructure

The common layer owns mechanics.

The Agent domain owns Agent meaning.

Future MCP and Text support must register their own domain profiles rather than adding type conditionals to a generic frontend API.

## 7. Architecture and ownership

### 7.1 Conceptual architecture

```text
AgentStoreWrapper
  -> Agent import/export consumer API
    -> Agent managed profile and domain policy
      -> common managed transfer service
        -> safe file reader
        -> strict profile registry
        -> canonical declaration decoder
        -> projected Artifact resolver
        -> prepared import registry
        -> atomic managed Source transaction API
      -> Artifact Store
      -> Agent Collection domain
      -> shared declaration resolver
      -> MCP installation/readiness aggregate
```

Future domains use the same lower-level mechanics:

```text
MCPStoreWrapper
  -> MCP import/export consumer API
    -> MCP managed profile
      -> common managed transfer service
```

```text
TextStoreWrapper
  -> Text import/export consumer API
    -> Text managed profile
      -> common managed transfer service
```

### 7.2 Component responsibilities

| Component                      | Responsibility                                                       |
| ------------------------------ | -------------------------------------------------------------------- |
| Safe file reader               | One-file sandboxed read through `llmtoolsutil`                       |
| Portable decoder               | YAML normalization and portable declaration validation               |
| Strict profile registry        | Base-schema binding, profile compilation, fail-fast initialization   |
| Prepared import registry       | Backend-held immutable preview state and commit tokens               |
| Projected Artifact reader      | In-memory overlay of prepared Source and current Store state         |
| Common transfer service        | Preview lifecycle, fingerprinting, token handling, export mechanics  |
| Atomic managed transaction API | Multi-package Source mutation, expected generation, reconciliation   |
| Agent domain                   | Strict Agent rules, package layout, Collection membership, conflicts |
| Shared resolver                | Built-in and mapped relationship resolution                          |
| MCP aggregate                  | Installation requirements, secret completion, readiness              |
| Agent management UI            | Preview, confirmation, inspection, export, deletion                  |

### 7.3 Common infrastructure and domain drivers

A common managed profile descriptor conceptually provides:

```text
ManagedProfile {
  DeclarationType
  BaseSchema
  RestrictionSchema
  Decode
  ValidateSemantics
  BuildPackagePlan
  BuildDestinationMutation
  InspectConflicts
  BuildReferenceCatalog
  ProjectReadiness
  Export
}
```

The common service must not expose this descriptor directly to the frontend.

The Agent consumer API selects the Agent profile internally.

## 8. Strict managed profile schemas

### 8.1 Schema purpose

The strict managed Agent schema constrains portable Agent declarations to forms the application can safely:

- Import as one managed package.
- Resolve without user-root external dependencies.
- Export and re-import.
- Inspect without a composition editor.
- Complete through supported local MCP setup flows.

The strict schema remains narrower than the portable Agent schema.

### 8.2 Subset by construction

A generic JSON Schema implication solver is not required.

The strict profile is compiled as:

```text
StrictManagedAgentSchema =
  allOf(
    PortableAgentSchema,
    ManagedAgentRestrictionSchema
  )
```

The common profile registry constructs the `allOf` wrapper itself.

The domain supplies only:

- The registered portable base schema key.
- The restriction schema.
- The declaration type.
- Semantic admission hooks.

Because the portable schema is always a required conjunct, every strict-profile-valid document must also be portable-schema-valid.

The restriction schema must not be compiled or used independently as the import authority.

### 8.3 Initialization-time validation

During application composition, the strict profile registry must:

- Confirm that the base Agent schema is registered.
- Confirm that the profile declaration type is `agent`.
- Compile the portable base and restriction schema as one Draft 2020-12 schema.
- Bind all referenced schema resources.
- Confirm that the resulting profile reports the declared base schema key.
- Confirm that exactly one profile is registered for the Agent managed import domain.
- Fail application initialization on any inconsistency.

The existing Agent schema key and schema version remain unchanged.

### 8.4 Schema and semantic admission

The strict schema owns structural restrictions such as:

- Prohibited top-level locator.
- Prohibited Agent program fields.
- Allowed member types.
- Allowed member forms.
- Inline Text requirement.
- Inline MCP shape.
- Prohibited selectors.
- Prohibited located relationships.

The Agent semantic validator owns contextual restrictions such as:

- Built-in target existence.
- Mapped target support.
- Dependency ambiguity.
- MCP policy provenance.
- Name conflicts.
- contained identity conflicts.
- package conflicts.
- secret placeholder rules.
- destination Collection compatibility.

### 8.5 Schema placement

Representative layout:

```text
internal/artifactcontract/declaration/
  agentv1/
    agent-v1.schema.json
    agent-managed-restrictions.schema.json
    document.go
    managed_profile.go
```

The restrictions schema is stored beside the Agent declaration schema because it constrains the same portable declaration type.

It is not added to generic declaration dispatch and does not create another Artifact `SchemaKey`.

Future domains follow the same pattern:

```text
internal/artifactcontract/declaration/
  mcpv1/
    mcp-v1.schema.json
    mcp-managed-restrictions.schema.json

  textv1/
    text-v1.schema.json
    text-managed-restrictions.schema.json
```

## 9. Agent managed import profile

### 9.1 Agent root declaration

A managed import root must:

- Use `type: agent`.
- Have a valid portable `name`.
- Be concrete.
- Have no top-level `locator`.
- Have no `loop`.
- Have no `workflow`.
- Use only strict-profile members.
- Contain no Store or runtime identity.

The common Agent header fields remain allowed:

- `name`.
- `displayName`.
- `description`.
- `labels`.
- `metadata`.

### 9.2 Allowed Agent members

The initial strict profile permits:

| Member type  | Allowed form                                        |
| ------------ | --------------------------------------------------- |
| `text`       | Contained inline Text                               |
| `model`      | Named protected built-in or supported mapped target |
| `tool`       | Named protected built-in or supported mapped target |
| `skill`      | Named protected built-in                            |
| `mcp`        | Named protected built-in or contained inline MCP    |
| `mcp.policy` | Named protected built-in                            |

The profile does not initially permit:

- Plugin members.
- Nested Agent members.
- Loop members.
- Workflow members.
- Member selectors.
- User-root external members.
- Located external members.

### 9.3 Prohibited Agent forms

The strict profile rejects:

- Any selector `base`.
- Any external member `locator`.
- Any top-level Agent `locator`.
- Any user-root named dependency.
- Any persisted `ArtifactRef`.
- Any Root ID.
- Any Source ID.
- Any runtime MCP server ID.
- Any Tool bundle ID.
- Any Model preset ID.
- Any declaration version.
- Any Agent release version.
- Any program `loop` or `workflow`.
- Any contained Tool.
- Any contained Skill.
- Any contained Model.
- Any contained Plugin.
- Any contained Agent.
- Any contained Loop.
- Any contained Workflow.

These restrictions apply only to managed Agent import and export.

### 9.4 Contained Text

Contained Text may use:

- `insert: instructions`.
- `insert: user-message`.
- Inline `content`.
- Optional `mediaType`.
- Portable Text descriptive fields.

Contained Text must not use:

- A file or directory locator.
- Include patterns.
- Exclude patterns.
- External source content.

Example:

```yaml
- type: text
  name: reviewer-instructions
  insert: instructions
  parameters:
    mediaType: text/markdown
    content: |
      Review correctness and security.
```

### 9.5 Model and Tool relationships

Model and Tool relationships may use:

- `scope: builtin` when selecting a protected built-in or built-in mapped target.
- An unscoped name only when the registered fallback provider resolves it as an accepted mapped target and no Artifact occurrence shadows it.

Model may use:

```yaml
overrides:
  includeSystemPrompt: true
```

Tool may use:

```yaml
overrides:
  autoExecute: false
```

A Tool relationship with `autoExecute: true` requires explicit preview confirmation.

The reference catalog must show the exact accepted names and copyable snippets.

### 9.6 Skill relationships

Skill relationships must:

- Be named.
- Use `scope: builtin`.
- Resolve to one protected built-in Skill Artifact.
- Use only a supported `use.mode`.

Supported modes remain:

```text
available
active
instructions
```

A managed Agent import must not refer to a Skill by `ArtifactRef`.

A managed Agent import must not contain a Skill because a concrete Skill requires an external package resource.

### 9.7 MCP relationships

A named MCP relationship must:

- Use `scope: builtin`.
- Resolve to one protected built-in MCP Artifact.
- Contain no runtime server ID.
- Contain no selected discovered Tool, resource, template, or prompt.
- Contain no runtime argument value.

A named MCP policy relationship must resolve to one protected built-in MCP Policy Artifact.

### 9.8 Inline MCP declarations

An inline MCP is represented as a contained MCP member:

```yaml
- type: mcp
  name: example-server
  parameters:
    transport: streamableHTTP
    url: https://example.com/mcp
    auth:
      mode: apiKey
    install:
      inputs:
        EXAMPLE_API_KEY:
          kind: secret
          label: Example API key
          required: true
```

An inline MCP:

- Must be concrete.
- Must not contain a declaration locator.
- Must not contain a source server selector.
- May declare stdio or streamable HTTP transport supported by the portable MCP contract.
- May declare installation input definitions.
- May declare authentication requirements.
- May refer to an accepted built-in MCP policy.
- Must not contain installation values.
- Must not contain secret defaults.
- Must not contain OAuth tokens.
- Must not contain selected local connection profile state.
- Must not contain active connection state.

Credential-bearing inline values must use the MCP installation-input mechanism supported by the MCP consumer.

The semantic admission validator must reject credential placement that bypasses the supported installation-input and secret-reference flow.

Preview must prominently show:

- Commands.
- URLs.
- Environment variable names.
- Header names.
- Authentication mode.
- Required installation inputs.
- Required secrets.
- Auto-execution or approval-relevant policy.

Executable stdio MCP declarations require explicit confirmation.

### 9.9 Metadata and local state

Agent `metadata` remains annotation only.

The strict profile must not use metadata to store:

- import source path.
- collection identity.
- source digest.
- prepared fingerprint.
- Agent readiness.
- MCP installation values.
- secret references.
- local enablement.
- runtime configuration.

All such values remain application or Store state.

## 10. Safe file input

The application must define a narrow file-reader port:

```text
ReadPortableArtifactFile(
  context,
  selectedPath,
  maximumBytes
) -> bytes and normalized path metadata
```

The production adapter uses `llmtoolsutil.ReadFile`.

The adapter must:

- Authorize only the explicitly selected file.
- Use the safe filesystem tool's sandbox boundary.
- Prevent symlink escape.
- Handle platform path rules through the filesystem tool.
- Reject directories.
- Reject devices and unsupported special files.
- Enforce a bounded read.
- Return cancellation and file errors without exposing unrelated file content.
- Avoid logging declaration content or secret-like values.
- Avoid exposing a generic file-reading method to the frontend.

For an absolute pasted path, the adapter may derive a one-file sandbox from the selected path's parent and basename. The safe filesystem implementation remains responsible for containment and symlink handling.

## 11. Agent import preview flow

### 11.1 Preview request

Conceptually:

```text
AgentImportPreviewRequest {
  Path
  Collection
  ExpectedCollectionRevision
  ExpectedSourceDigest?
}
```

The Collection identifies the Root and managed Source.

A separate caller-supplied Root is not required.

### 11.2 Preview processing

```text
Validate Collection destination
  -> read selected file through safe reader
  -> calculate source digest
  -> check optional expected source digest
  -> parse one YAML object
  -> validate portable Agent schema
  -> validate strict Agent profile
  -> run Agent semantic admission
  -> build canonical Definition
  -> build managed Agent package
  -> build Collection relationship mutation
  -> project Source and Artifacts
  -> resolve projected Agent
  -> inspect conflicts
  -> inspect MCP setup requirements
  -> calculate readiness
  -> retain prepared import
  -> return preview
```

If any structural or semantic error blocks import, preview returns issues but no committable token.

### 11.3 Projected Artifact resolution

Preview must validate the declaration as it would exist after import without committing it.

The common transfer infrastructure therefore provides an in-memory projected Artifact overlay containing:

- The prepared Agent package Source entry.
- The root Agent Artifact.
- Every contained Text and MCP Artifact.
- The planned Collection document when changed.
- Current Artifact Store records outside the prepared mutation.

The existing typed resolver runs against the overlay plus current Store state.

Projected Artifact references are preview-only identities. They must not be exposed as durable `ArtifactRef` values.

The preview response addresses projected declarations by stable occurrence paths.

### 11.4 Preview result

Conceptually:

```text
AgentImportPreview {
  CommitToken?
  PreparedFingerprint?
  ExpiresAt?

  SourceDigest
  DefinitionDigest
  NormalizedYAML

  AgentSummary
  Destination
  ProjectedArtifacts
  Relationships
  Conflicts
  RestoredMemberships
  MCPSetupRequirements
  Readiness

  CanImport
  RequiresConfirmation
  ReadyAfterCommit

  Issues
}
```

Issue severities are:

```text
error
confirmation
setup
information
```

Meanings:

- `error` blocks commit.
- `confirmation` requires explicit user acceptance.
- `setup` does not block import but prevents readiness.
- `information` describes resulting behavior.

Every issue should contain:

```text
code
severity
path
message
instruction
```

Source line and column may be included when available from YAML parsing.

### 11.5 Revalidation

The UI may rerun preview at any time.

Revalidation:

- Reads the external file again.
- Calculates a new source digest.
- Creates a new prepared import.
- Returns a new commit token and fingerprint.
- Does not mutate the previous prepared import.
- Does not alter persisted Store state.

Changing any of these requires revalidation:

- File content.
- Destination Collection.
- Destination Collection revision.
- Confirmation-relevant profile behavior.

## 12. Prepared import trust model

### 12.1 Backend-held prepared state

Prepared import state is held by the backend in a bounded process-local registry.

The frontend does not send the prepared YAML or package back during commit.

This prevents a modified frontend object from changing committed content.

Prepared state is not durable across application restart.

A restart requires a new preview.

### 12.2 Digests and fingerprint

The preview exposes three separate digests:

```text
SourceDigest
  -> exact external file bytes read during preview

DefinitionDigest
  -> canonical portable Agent declaration

PreparedFingerprint
  -> complete prepared import plan
```

The prepared fingerprint binds at least:

- Strict profile identity.
- Canonical declaration bytes.
- managed package files.
- Root and Source.
- selected Collection.
- expected Collection revision.
- expected Source generation.
- package address.
- planned Collection member.
- projected Artifact expectations.
- dependency witnesses.
- required confirmation codes.

A fingerprint alone is not accepted without its commit token.

### 12.3 Commit token

The commit token must be:

- Cryptographically random.
- Opaque to the frontend.
- Bound to one prepared import.
- Bound to one destination.
- Bound to one prepared fingerprint.
- Time-limited.
- Unusable for another domain operation.

Commit accepts no alternate path or declaration content.

### 12.4 Expiration and replay

The prepared registry must:

- Enforce a bounded number of entries.
- Enforce a bounded total byte size.
- Expire unused entries.
- Remove abandoned entries.
- Retain a bounded success receipt for committed-token replay.
- Return the original result for a duplicate commit request using the same successfully committed token.
- Reject token reuse with a different fingerprint or confirmation set.

### 12.5 Environmental witnesses

The external file is not reread at commit, but mutable Store dependencies must remain current.

Prepared state records witnesses for:

- Collection Artifact revision.
- managed Source generation.
- existing exact dangling memberships.
- protected built-in Artifact revisions or Definition digests.
- mapped target provider identity.
- mapped target identifier.
- any fallback catalog generation exposed by the provider.
- relevant Root and Source existence.

Commit checks these witnesses.

A changed witness makes the prepared import stale and requires revalidation.

## 13. Identity and conflict handling

### 13.1 Root Agent identity

The root managed Agent identity is:

```text
target Root
  + type: agent
  + logical name
```

The logical name also determines the managed package address.

Preview blocks import when:

- An available Agent with that name exists in the target Root.
- A missing but unpurged conflicting Agent occurrence exists.
- Another managed package occupies the calculated package address.
- Physical package normalization collides on the current platform.
- The name is reserved by Agent domain policy.

### 13.2 Contained Artifact identities

The decoder derives contained Artifact occurrences using stable semantic subresource paths.

Preview must inspect conflicts for every emitted identity.

For Text:

```text
text
  + name
  + insert
```

For MCP:

```text
mcp
  + name
```

The strict profile should encourage Agent-prefixed contained names, for example:

```text
my-agent-instructions
my-agent-request
my-agent-mcp
```

A dynamic name-prefix rule that cannot be expressed in JSON Schema may be enforced by the Agent semantic validator.

### 13.3 Package conflicts

Managed import is create-only.

The following are prohibited:

- Package replacement.
- Package merge.
- Package overwrite.
- Equivalent-content no-op import from a new preview.
- Automatic package rename.

A new preview for an already imported name returns a conflict even when the declaration is byte-equivalent.

Idempotency is limited to replay of the same commit token.

### 13.4 Collection relationship conflicts

The desired selected Collection relationship is a located external Agent member identifying the managed package occurrence.

Possible outcomes:

- No matching relationship:
  - Stage addition.
- Exact matching available relationship:
  - Conflict because the Agent already exists.
- Exact matching unavailable relationship and package absent:
  - Reuse as restoration.
- Same name with another locator:
  - Conflict.
- Same name as a symbolic member:
  - Conflict.
- Same name selected by a selector:
  - Conflict.
- Same name as a contained member:
  - Conflict.

Unavailable exact relationships in other Agent Collections may also be restored by re-import. Preview must list those affected Collections.

### 13.5 Built-in conflicts

A strict managed Agent name must not equal a protected built-in Agent name.

This avoids implicit Root shadowing in the managed import workflow.

Users must choose a distinct name when deriving a custom Agent from a built-in declaration.

Other member references may intentionally select built-ins through `scope: builtin`.

### 13.6 Delete and re-import

Delete remains package deletion, not Collection ownership mutation.

```text
Delete managed Agent
  -> remove Agent package
  -> preserve Collection relationships
  -> relationships become unavailable
```

Re-importing the same name:

```text
Preview same managed package address
  -> detect package absent
  -> detect exact dangling memberships
  -> prepare package restoration
  -> add selected membership only when missing
  -> commit
  -> exact dangling memberships become available
```

If the user keeps the existing Agent, changed content must use a new Agent name.

## 14. Atomic import commit

### 14.1 Transaction contents

The common Artifact Store transaction API must support one managed Source transaction containing multiple package mutations.

For Agent import this includes:

- One Agent package creation.
- Zero or one Agent Collection package replacement.
- Expected managed Source generation.
- Expected Collection Artifact revision.
- Expected package absence.
- Expected resulting root and contained Definitions.
- No unrelated package changes.

Both packages must belong to the same authorized managed Agent Source.

Otherwise import is rejected.

### 14.2 Commit sequence

```text
Load prepared import
  -> verify token and fingerprint
  -> verify confirmations
  -> acquire managed Source transaction lock
  -> verify Source generation
  -> verify Collection revision
  -> verify identity and package absence
  -> verify dependency witnesses
  -> stage Agent package
  -> stage Collection package mutation if required
  -> validate staged Source projection
  -> atomically publish staged package set
  -> reconcile Source once
  -> verify expected Artifacts and Collection
  -> store commit receipt
  -> return result
```

The staged Source projection must be equivalent to the projection used during preview.

### 14.3 Durable result and recovery

The durable transaction boundary is the managed Source package set.

Artifact and Definition records are derived from the committed Source.

If interruption occurs after Source publication but before response:

- The transaction receipt or Source generation identifies the committed mutation.
- Reconciliation is rerun idempotently.
- Replaying the same commit token returns the recovered result.
- No second package is created.
- No duplicate Collection member is added.

Generic Artifact Store staging and metadata may be extended for this purpose.

No Agent-specific transaction sidecar is introduced.

## 15. Portable managed export

### 15.1 Export eligibility

The Agent domain verifies that the requested Artifact:

- Is an Agent.
- Is in a non-protected user Root.
- Is available.
- Is top-level.
- Is backed by the managed Agent Source.
- Uses the managed Agent package kind.
- Uses the expected concrete Agent document location.
- Is not a source-selected alias.
- Passes the strict managed Agent structural profile.

Current dependency resolution is not required for export eligibility.

### 15.2 Export content

The export result contains:

```text
PortableArtifactExport {
  Type
  Name
  MediaType
  SuggestedFileName
  Content
  ContentDigest
  DefinitionDigest
  ArtifactRevision
  Resolution
  Readiness
}
```

For Agent export:

```text
MediaType: application/yaml
SuggestedFileName: <agent-name>.agent.yaml
```

`Content` is normalized portable YAML generated from the current immutable Definition.

### 15.3 Export exclusions

Export excludes:

- Collection membership.
- Collection identity.
- Root ID.
- Source ID.
- Artifact ID.
- package address.
- Artifact revision fields inside YAML.
- Definition digest fields inside YAML.
- local Agent enablement.
- local MCP installation input values.
- secret references and values.
- OAuth state.
- MCP connection profile selection.
- MCP runtime connection state.
- discovered MCP Tools, resources, prompts, or digests.
- readiness state.

Portable MCP installation input definitions remain exportable.

### 15.4 Export and re-import

An exported Agent file can be:

- Imported into another valid destination after changing its root Agent name and any conflicting contained names.
- Re-imported under its original name after deleting the existing managed Agent.
- Retained as a user-authored source file outside the application.

Export does not establish a synchronization link.

## 16. Agent Collections and lifecycle

The primary Agent management view is Artifact-oriented.

Collections are used for:

- Import destination.
- Filtering.
- Membership display.
- Empty Collection management.

The UI must not imply that a Collection owns its Agents.

Managed Collection operations remain:

- Create empty Agent Collection.
- Update display name and description.
- Enable or disable.
- Delete when empty.
- Inspect direct member status.
- Detach a direct member.
- Attach an existing eligible Agent where separately exposed.

Agent declaration operations become:

- Import.
- Inspect.
- Export.
- Enable or disable.
- Delete.

Agent declaration operations do not include:

- Create through a form.
- Edit.
- Replace.
- Version.
- Rename.

## 17. Agent readiness and MCP completion

### 17.1 Readiness dimensions

Agent management exposes separate readiness dimensions:

```text
DeclarationValid
ResolutionComplete
InstallationComplete
Ready
```

Definitions:

- `DeclarationValid` means the current Definition passes the portable and strict schemas.
- `ResolutionComplete` means all required relationships are available and unambiguous.
- `InstallationComplete` means all required local installation inputs, secret references, and authentication setup are complete.
- `Ready` means `DeclarationValid`, `ResolutionComplete`, and `InstallationComplete`.

`Ready` is configuration readiness.

It does not assert:

- Active MCP connection.
- Current network availability.
- Model provider availability.
- Tool execution permission.
- Successful Agent execution.

### 17.2 Inline MCP completion

Preview identifies inline MCP requirements by stable occurrence path, for example:

```text
members/mcp/example-server
```

After import, the contained MCP has a real terminal `ArtifactRef`.

The existing MCP installation and secret APIs then own completion.

The Agent detail view may launch the MCP completion UI for each incomplete MCP occurrence.

MCP completion does not modify the Agent Definition.

### 17.3 Secret handling

Preview returns only secret descriptors:

- Input name.
- Input kind.
- Label.
- Description.
- Required flag.
- Client-secret requirement.
- Occurrence path.

Preview never returns or retains secret values.

Secret values are submitted only to the MCP installation or secret subsystem after the MCP Artifact exists.

Export never reads secret values.

### 17.4 Readiness derivation

The Agent Store remains responsible for declaration and resolution.

An Agent management aggregate derives readiness by combining:

```text
Resolved Agent capability plan
  + terminal MCP Artifact identities
  + MCP installation completion
  + secret-reference health
  + authentication setup health
```

Readiness is calculated on demand.

It must not be persisted into:

- Agent YAML.
- Agent Definition.
- generic Artifact state.
- Agent metadata.
- Collection membership.

## 18. User interface flow

### 18.1 Agent management page

The page displays Agents rather than Assistant Preset bundles.

Primary controls:

- User Root or active Workspace selection.
- Agent Collection filter.
- Search.
- provenance filter.
- readiness filter.
- `Import Agent`.
- Collection management.

The Agent list shows:

- Display name.
- Logical name.
- managed or built-in provenance.
- enabled state.
- resolution state.
- setup/readiness state.
- Collection memberships.

Selecting an Agent shows:

- Summary.
- Normalized portable YAML.
- relationships and diagnostics.
- contained MCP setup state.
- Collection memberships.
- export.
- enablement.
- deletion for eligible managed Agents.

No declaration editor is shown.

### 18.2 Import dialog

The dialog contains:

- Destination Collection.
- File path.
- Native browse action.
- `Validate and Preview`.
- Strict profile explanation.
- Copyable valid Agent example.
- Available built-in and mapped reference catalog.
- Copyable reference snippets.

Changing the destination or file requires another preview.

### 18.3 Preview presentation

The preview shows:

- Source digest.
- Definition digest.
- Agent identity.
- Normalized YAML.
- planned Collection.
- projected contained Artifacts.
- protected built-in references.
- mapped references.
- inline MCP commands and URLs.
- resolution status.
- conflict analysis.
- restored dangling memberships.
- setup requirements.
- whether the Agent will be ready after commit.
- blocking errors.
- confirmation warnings.
- remediation instructions.

The dialog must clearly distinguish:

```text
Can import: yes or no
Ready after import: yes or no
Setup required: yes or no
```

### 18.4 Confirmation and commit

The user confirms the prepared preview.

The frontend submits:

- Commit token.
- Prepared fingerprint.
- Accepted confirmation issue codes.

The frontend does not submit:

- YAML.
- path.
- package bytes.
- Collection replacement content.
- projected Artifact data.

On success:

- The new Agent becomes selected.
- The Agent detail view opens.
- If setup is incomplete, the MCP completion flow is offered.
- The Agent remains visibly not ready until completion succeeds.

### 18.5 Export flow

For an eligible managed Agent:

- User selects `Export`.
- Backend returns the current portable export.
- The existing download flow writes the returned content.
- The UI explains that membership, enablement, and secrets are not exported.
- The UI explains that importing while the current Agent exists requires a new name.

### 18.6 Delete and re-import flow

The deletion dialog explains:

- The managed Agent package will be deleted.
- Collection relationships will remain declared and become unavailable.
- Re-importing the same Agent name can restore those relationships.
- Export should be used first if the user wants to preserve or modify the declaration.

After deletion:

- The unavailable Collection relationships remain inspectable.
- The user may modify the exported YAML.
- The user previews and imports again.
- Preview reports which dangling memberships will be restored.

## 19. Consumer API shape

The names below are conceptual. Concrete Go naming follows repository conventions.

### 19.1 Agent import destination

```text
ListAgentImportDestinations()
  -> []AgentImportDestination
```

```text
AgentImportDestination {
  RootID
  RootDisplayName

  Collection
  CollectionRevision
  CollectionName
  CollectionDisplayName
  Baseline
  Enabled
}
```

Only eligible editable managed Agent Collections are returned.

### 19.2 Agent import preview

```text
PreviewAgentImport(
  AgentImportPreviewRequest
) -> AgentImportPreview
```

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
  CommitToken?
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
  MCPSetupRequirements
  Readiness
  Issues

  CanImport
  RequiresConfirmation
  ReadyAfterCommit
}
```

Validation and policy failures are represented as preview issues.

Unexpected infrastructure failures remain operation errors.

### 19.3 Agent import commit

```text
CommitAgentImport(
  AgentImportCommitRequest
) -> AgentImportCommitResult
```

```text
AgentImportCommitRequest {
  CommitToken
  PreparedFingerprint
  AcceptedConfirmationCodes
}
```

```text
AgentImportCommitResult {
  Agent
  Collection
  RestoredMemberships
  Readiness
  CommitFingerprint
}
```

No declaration content is accepted by commit.

### 19.4 Agent export

```text
ExportManagedAgent(
  AgentExportRequest
) -> PortableArtifactExport
```

```text
AgentExportRequest {
  Agent
}
```

The backend returns portable bytes. The download destination remains an application UI concern.

### 19.5 Agent reference catalog

```text
ListAgentImportReferenceCatalog(
  AgentReferenceCatalogRequest
) -> AgentReferenceCatalog
```

The request may include the selected destination Collection so that accepted mapped and built-in names are checked in the correct Root context.

### 19.6 Agent readiness

```text
GetAgentReadiness(
  ArtifactRef
) -> AgentReadiness
```

```text
AgentReadiness {
  DeclarationValid
  ResolutionComplete
  InstallationComplete
  Ready
  Issues
  MCPRequirements
}
```

The Agent management detail response may include this projection to avoid a separate call.

## 20. Persistence and security boundaries

The feature persists only normal Artifact ecosystem state:

- Managed Agent package.
- Managed Collection package update.
- Definitions.
- Artifacts.
- Source generation.
- Artifact enablement.
- MCP local installation data after separate completion.
- Secret references in the MCP installation subsystem.
- Secret values in secret storage.

The feature does not persist:

- External import path.
- prepared preview after expiration.
- source file contents outside the managed package.
- commit token in the Agent declaration.
- source digest in the Agent declaration.
- prepared fingerprint in the Agent declaration.
- readiness in the Agent declaration.
- Collection membership inside the Agent declaration.
- secret values in prepared state.
- Agent-specific transaction sidecars.

Security-sensitive preview information must identify behavior without exposing secrets.

Preview must call out:

- Stdio commands.
- Command arguments.
- HTTP URLs.
- Header names.
- Environment variable names.
- Authentication modes.
- auto-execute overrides.
- required secret inputs.
- approval-relevant MCP policies.

Artifact enablement must not be presented as a security sandbox.

## 21. Breaking removals

### 21.1 Assistant Preset removal

The following backend areas are removed:

```text
cmd/agentgo/
  wrapper_assistantpreset_artifact.go

internal/assistantpreset/
  lookupimpl/
  spec/
  store/
```

Legacy Assistant Preset embedded data, overlay storage, storage constants, and application storage declarations are removed.

The following frontend area is removed:

```text
frontend/app/assistantpresets/
```

Assistant Preset routes, API adapters, generated models, composer integrations, and management labels are removed.

No migration or compatibility path is retained.

### 21.2 Managed Agent authoring removal

The following Agent APIs and types are removed:

- `ManagedAgentDocument`.
- `ManagedAgentMember`.
- `ManagedAgentCreateRequest`.
- `ManagedAgentCreateResult`.
- `ManagedAgentReplaceRequest`.
- `ManagedAgentReplaceResult`.
- `CreateManagedAgent`.
- `ReplaceManagedAgent`.

The internal package-publication logic may be refactored and retained behind import commit.

The following remain:

- Managed Agent delete.
- Agent enablement.
- Agent reads.
- Agent resolution.
- Collection reads and management.
- Collection membership internals.
- Built-in Agent installation.

### 21.3 Behaviors intentionally not retained

The new design does not retain:

- Assistant Preset versions.
- New-version creation.
- bundle enablement gating.
- ordered Tool selection.
- ordered Skill selection.
- persisted Tool user arguments.
- persisted MCP conversation context.
- persisted discovered MCP capabilities.
- field-by-field Model selection.
- field-by-field Tool selection.
- field-by-field Skill selection.
- field-by-field MCP selection.
- copying another preset through a form.
- editing a managed Agent declaration.
- replacing a managed Agent declaration.

## 22. Implementation areas

### 22.1 Contract profile infrastructure

Representative common implementation:

```text
internal/artifactcontract/managedprofile/
  descriptor.go
  registry.go
  schema.go
  validation.go
```

Responsibilities:

- Bind one restriction schema to one portable base schema.
- Construct the conjunctive strict schema.
- Compile registered profiles.
- Reject duplicate or invalid profile registration.
- Expose strict validation to domain services.
- Keep profile schemas out of portable declaration dispatch.

Agent-specific additions:

```text
internal/artifactcontract/declaration/agentv1/
  agent-managed-restrictions.schema.json
  managed_profile.go
```

No existing Agent schema key or version changes.

### 22.2 Common managed transfer infrastructure

Representative implementation:

```text
internal/managedartifact/transfer/
  file_reader.go
  preview.go
  prepared.go
  registry.go
  fingerprint.go
  export.go
  types.go
```

Responsibilities:

- Safe file-reader port.
- Source digest.
- normalized portable content.
- prepared import registry.
- commit tokens.
- fingerprints.
- expiration and replay.
- common preview issue model.
- common portable export response.
- no domain type switching in public APIs.

The production file-reader adapter uses:

```text
internal/llmtoolsutil.ReadFile
```

### 22.3 Artifact Store transaction support

Extend the managed Artifact boundary with a transaction or batch mutation API capable of:

- Expected Source generation.
- Multiple managed package mutations in one Source.
- Package create and package replace operations.
- Staged projection.
- Atomic package-set publication.
- One Source reconciliation.
- Expected Artifact verification.
- Durable idempotent recovery.

Representative area:

```text
internal/artifactstore/
  managedtransaction/
```

or an extension of:

```text
internal/artifactstore/compositionapi/
internal/artifactstore/providerapi/
```

The transaction implementation must reuse existing managed Source staging and metadata facilities where possible.

### 22.4 Agent domain implementation

Representative Agent additions:

```text
internal/agent/store/
  consumerapi/
    import.go
    export.go
    reference_catalog.go
    readiness.go
    import_types.go

  domain/
    managed_profile.go
    import_policy.go
    import_conflicts.go
    import_package.go
```

Responsibilities:

- Agent strict semantic admission.
- Agent destination validation.
- Agent package planning.
- Collection member planning.
- root and contained identity conflict checks.
- dangling membership restoration.
- built-in and mapped reference validation.
- managed Agent export eligibility.
- Agent readiness projection.

Existing `publishManagedAgent` logic should be refactored into the atomic transaction planner rather than exposed as structured authoring.

### 22.5 Application composition and wrappers

`AgentStoreWrapper` gains methods for:

- Listing import destinations.
- Listing accepted reference names.
- Previewing an Agent import path.
- Committing a prepared Agent import.
- Exporting a managed Agent.
- Reading Agent management detail and readiness.

The wrapper composes:

- Agent domain import/export service.
- common prepared import registry.
- strict profile registry.
- safe file reader.
- managed transaction API.
- MCP readiness dependencies.

No Assistant Preset wrapper remains.

### 22.6 Future MCP and Text reuse

Future MCP import/export adds:

- MCP restriction schema beside `mcpv1`.
- MCP destination and Collection policy.
- MCP package planning.
- MCP conflict policy.
- MCP export eligibility.
- MCP installation completion projection.

Future Text import/export adds:

- Text restriction schema beside `textv1`.
- Text destination policy.
- inline or managed resource policy.
- Text package planning.
- Text export eligibility.

Neither future domain changes the Agent profile.

The common transfer service remains unaware of Agent, MCP, and Text semantics.

## 23. Implementation status

Status terminology:

- `Available` means an existing implementation path or dependency is present.
- `Planned` means required by this HLD but not yet implemented.
- `Removed` means the capability must not exist after this breaking change.
- `Deferred` means intentionally reserved for a later domain profile.

| Capability                                      | Status    | Notes                                              |
| ----------------------------------------------- | --------- | -------------------------------------------------- |
| Portable Agent schema                           | Available | Existing `agentv1` contract remains unchanged      |
| Canonical Agent YAML decoding                   | Available | Existing canonical YAML provider and Agent decoder |
| Managed Agent Source and package layout         | Available | Existing Agent managed Source infrastructure       |
| Agent Collections                               | Available | Existing Agent-only Plugin domain                  |
| Built-in Agent, Skill, Tool, and MCP resolution | Available | Existing protected Root and typed resolver         |
| Tool and Model mapped fallback                  | Available | Current registered fallback types                  |
| Safe cross-platform file operations             | Available | `llmtoolsutil` and `llmtools-go/fstool`            |
| Strict managed profile registry                 | Planned   | Common conjunctive schema registration             |
| Agent managed restriction schema                | Planned   | Additional schema beside `agentv1`                 |
| Initialization-time profile validation          | Planned   | Fail-fast application composition                  |
| Safe Agent YAML preview reader                  | Planned   | Uses the safe file adapter                         |
| Projected managed Artifact overlay              | Planned   | Required for mutation-free dry run                 |
| Agent semantic import admission                 | Planned   | Built-in, mapped, inline MCP, and conflict policy  |
| Backend-held prepared import registry           | Planned   | Bounded token and fingerprint state                |
| Source, Definition, and prepared fingerprints   | Planned   | Separate digests with distinct purposes            |
| Environmental dependency witnesses              | Planned   | Collection, Source, built-in, and mapped state     |
| Atomic multi-package managed Source transaction | Planned   | Agent package plus Collection package              |
| Commit without source-file reread               | Planned   | Commit consumes backend-held prepared state        |
| Idempotent committed-token replay               | Planned   | Returns original result                            |
| Managed Agent portable YAML export              | Planned   | User-managed top-level Agents only                 |
| Agent accepted-reference catalog                | Planned   | Built-in and mapped names with snippets            |
| Inline MCP import profile                       | Planned   | Declaration permitted without local values         |
| MCP completion requirements in preview          | Planned   | Input descriptors only                             |
| Derived Agent readiness                         | Planned   | Resolution plus MCP local completion               |
| In-place managed Agent editing                  | Removed   | Export, rename externally, and import instead      |
| Managed Agent replacement                       | Removed   | Delete and re-import or use a new name             |
| Structured `ManagedAgentDocument` authoring     | Removed   | Replaced by strict file import                     |
| Assistant Preset backend                        | Removed   | No compatibility or migration                      |
| Assistant Preset frontend                       | Removed   | Replaced by Agent Artifact management              |
| Assistant Preset storage and overlays           | Removed   | No retained legacy behavior                        |
| Agent declaration versions                      | Removed   | No Agent release version is introduced             |
| MCP domain import/export profile                | Deferred  | Reuses common transfer infrastructure              |
| Text domain import/export profile               | Deferred  | Reuses common transfer infrastructure              |
