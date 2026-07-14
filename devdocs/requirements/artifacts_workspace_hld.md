# Artifact Store and Workspace HLD

## 1. Document status

- Status: Proposed architecture
- Scope: Internal Artifact Store foundation and Workspace feature built on it
- Excludes: Conversation configuration, conversation persistence, runtime lifecycle, secret values, policy evaluation, and execution logic
- Primary objective: Establish a durable, minimal artifact lifecycle model that Workspace uses first and other stores may adopt later

## 2. Architectural decisions

| ID      | Decision                                                                                                             |
| ------- | -------------------------------------------------------------------------------------------------------------------- |
| `AS-01` | Artifact Store owns all durable artifact lifecycle state                                                             |
| `AS-02` | Workspace has no separate persistent Workspace Store database                                                        |
| `AS-03` | Workspace is implemented as a typed consumer of Artifact Store                                                       |
| `AS-04` | Current domain stores remain physically unchanged in the Workspace implementation                                    |
| `AS-05` | Current Bundle maps to generic Artifact Collection                                                                   |
| `AS-06` | Current item maps to generic Artifact Record                                                                         |
| `AS-07` | Portable source package and app-local collection are separate concepts                                               |
| `AS-08` | Artifact definitions do not contain app IDs, local paths, secrets, runtime state, or conversation state              |
| `AS-09` | Artifact Store owns source kinds; consumers do not implement source transport behavior                               |
| `AS-10` | Artifact Store initially supports `fs-directory` and `embedded-fs-directory` sources                                 |
| `AS-11` | Artifact revisions are identified by canonical digest, not a generated revision ID                                   |
| `AS-12` | Catalog resources use source locator identity, not a generated resource ID                                           |
| `AS-13` | Artifact Record is the sole generic app-side artifact record                                                         |
| `AS-14` | Artifact Store supports source schemas through registered frontends and does not require a universal source envelope |
| `AS-15` | Runtime projections and execution are outside Artifact Store                                                         |
| `AS-16` | Workspace Root is a typed Artifact Root with kind `flexigpt.workspace`                                               |
| `AS-17` | Workspace artifacts are synchronized into root-local Artifact Records                                                |
| `AS-18` | Imported, captured, linked, and forked artifacts are represented by Artifact Record modes                            |

---

# Part A: Artifact Store

## 3. Purpose

Artifact Store is a generic internal FlexiGPT component that manages portable artifact definitions and app-local artifact records.

It provides the common substrate for:

- Source registration.
- Source scanning.
- Package metadata.
- Artifact discovery.
- Artifact validation.
- Canonical definitions.
- Revision digests.
- Root catalogs.
- Local records.
- Collections.
- Record synchronization.
- Dependency declarations.
- Import.
- Export.
- Forking.
- Capture.
- Diagnostics.
- Provenance.

It does not interpret domain behavior.

It does not know what a Skill, MCP server, Tool, Model Preset, Workspace, Agent, or Assistant Preset means.

## 4. Scope boundary

```text
Artifact Store
  - source and content lifecycle
  - definition lifecycle
  - local record lifecycle
  - collection lifecycle
  - catalog lifecycle

Workspace
  - Workspace root semantics
  - Workspace discovery rules
  - Workspace schemas
  - Workspace resource grouping
  - Workspace composition
  - Workspace runtime projection

Runtime
  - model clients
  - MCP connections
  - Skill sessions
  - Tool execution
  - prompt rendering
  - policy and trust evaluation
```

## 5. Entity model

```text
Artifact Root
  -> Root Source Attachments
  -> Artifact Sources
  -> optional Artifact Packages
  -> Catalog Resources
  -> Canonical Definitions by digest

Artifact Root
  -> Artifact Collections
  -> Artifact Records
  -> Catalog Resources
  -> Canonical Definitions by digest
```

## 6. Entity definitions

## 6.1 Artifact Root

An Artifact Root is an app-owned logical mount.

It groups:

- Attached sources.
- Catalog generations.
- Collections.
- Records.
- Root-local typed configuration.

An Artifact Root is not necessarily a filesystem directory.

Examples:

```text
Backend Workspace
Mobile Workspace
Built-in Resources
App Library
Imported Company Package
```

### Fields

```text
ArtifactRoot
  - RootID
  - Kind
  - DisplayName
  - Description
  - Enabled
  - DataSchemaID
  - Data
  - CreatedAt
  - ModifiedAt
  - SoftDeletedAt
```

`RootID` is generated and stable.

`Kind` is consumer-defined.

Examples:

```text
flexigpt.workspace
flexigpt.app-library
flexigpt.builtin
flexigpt.package-mount
```

`Data` is opaque to Artifact Store.

The owner of the root kind validates and interprets it.

## 6.2 Artifact Source

An Artifact Source is a content provider.

It is responsible only for safely exposing directory-like content to Artifact Store.

Initial supported source kinds:

```text
fs-directory
embedded-fs-directory
memory-directory
```

### Fields

```text
ArtifactSource
  - SourceID
  - Kind
  - DisplayName
  - Enabled
  - ConfigSchemaID
  - Config
  - LastObservedGeneration
  - LastScannedAt
  - CreatedAt
  - ModifiedAt
```

`SourceID` is generated and stable.

### `fs-directory` source

Example configuration:

```json
{
  "rootPath": "/Users/alice/repos/backend",
  "followSymlinks": false,
  "managedByApp": false
}
```

The source driver owns:

- Path normalization.
- Path containment.
- Symlink policy.
- File stat.
- Directory walk.
- File read.
- Source generation calculation.

### `embedded-fs-directory` source

Example configuration:

```json
{
  "providerKey": "flexigpt.builtins",
  "rootLocator": "artifact-bundles"
}
```

At application startup, FlexiGPT registers an embedded filesystem provider under `providerKey`.

Artifact Store can then:

- Walk it.
- Read files.
- Scan it.
- Synchronize records.
- Treat it like any other directory source.

## 6.3 Root Source Attachment

A Root Source Attachment connects an Artifact Source to an Artifact Root.

Its natural key is:

```text
RootID + SourceID
```

No generated attachment ID is required.

### Fields

```text
RootSourceAttachment
  - RootID
  - SourceID
  - Role
  - Priority
  - Enabled
  - DataSchemaID
  - Data
  - CreatedAt
  - ModifiedAt
```

Example roles:

```text
primary
attached-package
built-in
app-library
overlay
```

Artifact Store stores role and priority.

The consumer feature interprets their meaning.

## 6.4 Artifact Package

An Artifact Package is optional portable source metadata.

It is used for:

- Sharing.
- Export.
- Import.
- Package display metadata.
- Source-level grouping.
- Declared package assets.
- Package-level documentation.

It is not equivalent to an Artifact Collection.

Its natural key is:

```text
SourceID + manifest locator
```

### Fields

```text
ArtifactPackage
  - SourceID
  - ManifestLocator
  - Name
  - Version
  - DisplayName
  - Description
  - CurrentManifestDigest
  - State
  - Diagnostics
```

A source may have:

- Zero packages.
- One package.
- Multiple packages.
- Resources outside any package.

## 6.5 Catalog Resource

A Catalog Resource is a discovered source-local artifact occurrence.

It has no generated resource ID.

Its natural identity is:

```text
SourceID
+ Locator
+ SubresourceLocator
```

Examples:

```text
SourceID: backend-fs
Locator: .flexigpt/workspace.json
SubresourceLocator: workspace
```

```text
SourceID: backend-fs
Locator: .flexigpt/models/team-openai.json
SubresourceLocator: models/gpt-5
```

```text
SourceID: backend-fs
Locator: .skills/code-review/SKILL.md
SubresourceLocator: skill
```

### Fields

```text
CatalogResource
  - SourceID
  - Locator
  - SubresourceLocator
  - PackageManifestLocator
  - Kind
  - LogicalName
  - LogicalVersion
  - CurrentDefinitionDigest
  - SourceContentDigest
  - FrontendID
  - State
  - FirstSeenAt
  - LastSeenAt
  - Diagnostics
```

A Catalog Resource represents one source location.

It is not an app-owned item.

## 6.6 Canonical Definition

A Canonical Definition is normalized portable content.

It is identified by canonical SHA-256 digest.

```text
DefinitionDigest = sha256(canonical normalized definition)
```

### Fields

```text
CanonicalDefinition
  - Digest
  - Kind
  - SchemaID
  - SchemaVersion
  - LogicalName
  - LogicalVersion
  - DisplayName
  - Description
  - Labels
  - Extensions
  - DefinitionJSON
  - DependencySelectors
  - AssetManifest
  - CreatedAt
```

The digest is the revision identity.

There is no generated `RevisionID`.

Definitions may be deduplicated by digest even when found in different sources.

## 6.7 Artifact Record

An Artifact Record is the local app item.

It is the generic equivalent of:

```text
Skill
Tool
Assistant Preset
Model Preset entry
MCP server registration
```

It has a generated stable `RecordID`.

### Fields

```text
ArtifactRecord
  - RecordID
  - RootID
  - CollectionID
  - Kind
  - Name
  - Version
  - SourceID
  - Locator
  - SubresourceLocator
  - RecordMode
  - TrackingMode
  - PinnedDefinitionDigest
  - LastResolvedDefinitionDigest
  - Enabled
  - DataSchemaID
  - Data
  - State
  - CreatedAt
  - ModifiedAt
```

`RecordMode` values:

```text
linked
captured
forked
app-local
embedded-overlay
```

`TrackingMode` values:

```text
follow-source
pin-digest
manual-refresh
```

Artifact Record is the only generic app-side artifact entity.

There is no separate generic binding entity.

## 6.8 Artifact Collection

An Artifact Collection is an app-local grouping of Artifact Records.

It maps directly to current Bundle semantics.

### Fields

```text
ArtifactCollection
  - CollectionID
  - RootID
  - Kind
  - Slug
  - DisplayName
  - Description
  - Enabled
  - DataSchemaID
  - Data
  - CreatedAt
  - ModifiedAt
  - SoftDeletedAt
```

`CollectionID` is generated and stable.

## 6.9 Root Catalog Generation

A Root Catalog Generation is a durable scan publication record.

It does not require a random ID.

Its identity is:

```text
RootID + Generation
```

### Fields

```text
RootCatalogGeneration
  - RootID
  - Generation
  - SourceGenerations
  - ScanPlanDigest
  - CatalogDigest
  - CreatedAt
  - Diagnostics
```

---

# 7. Bundle, package, collection, and item mapping

## 7.1 Current model

```text
Bundle
  -> Item
```

## 7.2 Generic Artifact Store model

```text
ArtifactCollection
  -> ArtifactRecord
    -> CatalogResource
      -> CanonicalDefinition digest
```

## 7.3 Portable package model

```text
ArtifactPackage
  -> CatalogResources
```

## 7.4 Full relationship

```text
ArtifactPackage
  -> CatalogResource
  -> CanonicalDefinition

ArtifactCollection
  -> ArtifactRecord
  -> CatalogResource
  -> CanonicalDefinition
```

An Artifact Package may cause a derived Artifact Collection to be created, but they remain separate objects.

## 7.5 Mapping table

| Current concept       | Generic artifact equivalent |
| --------------------- | --------------------------- |
| Bundle ID             | Collection ID               |
| Bundle slug           | Collection slug             |
| Bundle display name   | Collection display name     |
| Bundle enabled state  | Collection enabled state    |
| Bundle soft deletion  | Collection soft deletion    |
| Item ID               | Record ID                   |
| Item slug             | Record name                 |
| Item version          | Record version              |
| Item enabled state    | Record enabled state        |
| Item source location  | Record source locator       |
| Item content          | Canonical Definition        |
| Item revision         | Definition digest           |
| Built-in item overlay | Embedded overlay Record     |
| User-created item     | App-local Record            |
| Imported item         | Captured or linked Record   |
| Forked item           | Forked Record               |
| Missing source item   | Record state `missing`      |

---

# 8. Artifact Store responsibilities

## 8.1 Root lifecycle

Artifact Store provides:

- Create root.
- Read root.
- List roots.
- Update root.
- Enable or disable root.
- Soft delete root.
- Attach source.
- Detach source.
- List root attachments.
- Validate root typed data through registered root kind hook.

## 8.2 Source lifecycle

Artifact Store provides:

- Create source.
- Read source.
- List sources.
- Update source.
- Enable or disable source.
- Delete source.
- Validate source config through source driver.
- Observe source generation.
- Store source diagnostics.
- Scan source content.

## 8.3 Package lifecycle

Artifact Store provides:

- Discover package manifests.
- Store package metadata.
- Track package manifest digest.
- Associate catalog resources with package locator.
- Store package diagnostics.
- Export package metadata.

## 8.4 Catalog lifecycle

Artifact Store provides:

- Execute scan plan.
- Discover candidates.
- Select artifact frontend.
- Normalize zero, one, or many definitions from candidate.
- Validate definitions.
- Store catalog resources.
- Store canonical definitions by digest.
- Store diagnostics.
- Publish root catalog generation.

## 8.5 Record synchronization

Artifact Store provides:

- Find record by root plus source locator.
- Create derived linked record.
- Update record source state.
- Update last resolved digest.
- Preserve record local data.
- Mark records stale.
- Mark records missing.
- Avoid automatic record deletion.
- Support pin and detach behavior.

## 8.6 Collection lifecycle

Artifact Store provides:

- Create collection.
- Ensure base collection.
- Read collection.
- List collections.
- Update collection.
- Enable or disable collection.
- Soft delete collection.
- Validate collection existence.
- Reject deletion when records remain.
- Move records between collections.
- Sweep empty soft-deleted collections.

## 8.7 Transfer lifecycle

Artifact Store provides:

- Export canonical definition.
- Export package metadata.
- Export frontend-declared assets.
- Import definition into app-local source.
- Create linked record.
- Create captured record.
- Create forked record.
- Preserve provenance.
- Store captured assets in optional blob store.

## 8.8 Dependency lifecycle

Artifact Store provides:

- Store dependency selectors.
- Query candidate catalog resources.
- Build dependency graph.
- Detect cycles.
- Report missing dependencies.
- Report ambiguous candidates.
- Return diagnostics.

Artifact Store does not choose consumer precedence rules.

---

# 9. Artifact Store validation model

| Validation layer              | Owner                               | Examples                                           |
| ----------------------------- | ----------------------------------- | -------------------------------------------------- |
| Root validation               | Artifact Store plus root hook       | root kind, root data schema                        |
| Source validation             | Artifact Store plus source driver   | filesystem config, embedded provider key           |
| Source safety                 | Artifact Store and source driver    | path traversal, containment, file limits           |
| Generic definition validation | Artifact Store                      | locator, digest, metadata fields                   |
| Structural validation         | Artifact frontend                   | JSON schema, source document shape                 |
| Semantic validation           | Artifact frontend callback          | Workspace discovery declarations, MCP schema rules |
| Collection validation         | Artifact Store plus collection hook | slug, collection state, placement rules            |
| Record validation             | Artifact Store plus frontend        | target locator, local data schema                  |
| Runtime validation            | Outside Artifact Store              | Skill indexing, MCP connection, model readiness    |
| Policy validation             | Outside Artifact Store              | trust, approval, execution policy                  |

---

# 10. Artifact Store source model

## 10.1 Source kind ownership

Artifact Store owns source kind implementations.

Domain features may not create ad hoc transport implementations.

| Source kind             | Owner          | Initial status  |
| ----------------------- | -------------- | --------------- |
| `fs-directory`          | Artifact Store | Required        |
| `embedded-fs-directory` | Artifact Store | Required        |
| `memory-directory`      | Artifact Store | Test-only       |
| `git-checkout`          | Artifact Store | Future optional |
| `zip-directory`         | Artifact Store | Future optional |
| `cas-directory`         | Artifact Store | Future optional |

## 10.2 Materialization rule

A domain-specific remote acquisition flow should usually materialize to an Artifact Store-supported directory source.

Examples:

```text
MCP-managed HTTP download
-> verified local directory cache
-> fs-directory source
```

```text
Git sync worker
-> checkout directory
-> fs-directory source
```

```text
ZIP importer
-> extraction directory
-> fs-directory source
```

This avoids embedding network, Git, archive, or credential behavior in Workspace, Skills, MCP, or Assistant packages.

## 10.3 Source driver contract

```go
type SourceDriver interface {
  Kind() string

  ValidateConfig(
    ctx context.Context,
    config json.RawMessage,
  ) []Diagnostic

  Snapshot(
    ctx context.Context,
    source ArtifactSource,
  ) (SourceGeneration, error)

  Open(
    ctx context.Context,
    source ArtifactSource,
    locator string,
  ) (io.ReadCloser, error)

  Stat(
    ctx context.Context,
    source ArtifactSource,
    locator string,
  ) (SourceEntry, error)

  ReadDir(
    ctx context.Context,
    source ArtifactSource,
    locator string,
  ) ([]SourceEntry, error)

  Walk(
    ctx context.Context,
    source ArtifactSource,
    root string,
    fn WalkFunc,
  ) error
}
```

---

# 11. Artifact Store extension model

## 11.1 Root kind hook

```go
type RootKindHook interface {
  Kind() string

  ValidateRootData(
    ctx context.Context,
    root ArtifactRoot,
  ) []Diagnostic

  ValidateSourceAttachment(
    ctx context.Context,
    root ArtifactRoot,
    attachment RootSourceAttachment,
  ) []Diagnostic
}
```

## 11.2 Artifact frontend

```go
type ArtifactFrontend interface {
  ID() string

  Recognizes(
    ctx context.Context,
    candidate ArtifactCandidate,
  ) Recognition

  Decode(
    ctx context.Context,
    candidate ArtifactCandidate,
  ) ([]CanonicalDefinition, []Diagnostic)

  ValidateStructure(
    ctx context.Context,
    definition CanonicalDefinition,
  ) []Diagnostic

  ValidateSemantic(
    ctx context.Context,
    definition CanonicalDefinition,
  ) []Diagnostic

  ExtractDependencies(
    ctx context.Context,
    definition CanonicalDefinition,
  ) ([]ArtifactSelector, []Diagnostic)

  ValidateRecordData(
    ctx context.Context,
    definition CanonicalDefinition,
    record ArtifactRecordDraft,
  ) []Diagnostic

  DescribeExportClosure(
    ctx context.Context,
    definition CanonicalDefinition,
  ) (ExportClosure, []Diagnostic)
}
```

## 11.3 Collection kind hook

```go
type CollectionKindHook interface {
  Kind() string

  ValidateCollectionData(
    ctx context.Context,
    collection ArtifactCollection,
  ) []Diagnostic

  ValidateRecordPlacement(
    ctx context.Context,
    collection ArtifactCollection,
    record ArtifactRecord,
  ) []Diagnostic
}
```

---

# 12. Artifact Store scanning workflow

```text
Artifact Root
-> attached Source
-> source generation
-> caller-supplied scan plan
-> candidates
-> frontend recognition
-> canonical definitions
-> generic validation
-> structural validation
-> semantic validation
-> catalog resources
-> definition digests
-> root catalog generation
-> record synchronization
```

A scan plan may contain:

- Explicit source locators.
- Directory roots.
- Glob patterns.
- Recursion settings.
- Candidate priorities.
- Allowed frontend IDs.
- Maximum file sizes.
- Maximum traversal depth.
- Package manifest patterns.

Artifact Store executes plans.

Consumers define plans.

---

# 13. Artifact Store record synchronization workflow

```text
Root catalog generation
-> valid Catalog Resource
-> locate Artifact Record by:
   RootID + SourceID + Locator + SubresourceLocator + Kind

if record exists:
  update source state
  update last resolved digest
  preserve local record data

if record does not exist:
  create derived linked Artifact Record
  assign RecordID
  optionally place in derived Collection
```

When source content changes:

```text
Catalog Resource current definition digest changes
-> linked Record follows new digest
or
-> pinned Record remains on pinned digest
or
-> manual-refresh Record becomes stale
```

When source content disappears:

```text
Catalog Resource state becomes missing
-> Artifact Record remains
-> Artifact Record state becomes missing
```

---

# 14. Artifact Store API surface

```text
Roots
  - CreateRoot
  - GetRoot
  - ListRoots
  - UpdateRoot
  - DeleteRoot
  - AttachSource
  - DetachSource
  - ListRootSources

Sources
  - CreateSource
  - GetSource
  - ListSources
  - UpdateSource
  - DeleteSource
  - RefreshSource

Catalog
  - ScanSource
  - ScanRoot
  - GetRootCatalogGeneration
  - ListCatalogResources
  - GetCatalogResource
  - GetDefinitionByDigest
  - ListDefinitionHistory

Records
  - GetRecord
  - ListRecords
  - CreateRecord
  - UpdateRecord
  - RefreshRecord
  - PinRecord
  - DetachRecord
  - DeleteRecord

Collections
  - EnsureBaseCollection
  - CreateCollection
  - UpdateCollection
  - DeleteCollection
  - ListCollections
  - AddRecordToCollection
  - RemoveRecordFromCollection

Dependencies
  - GetDependencies
  - FindCandidates
  - BuildDependencyGraph
  - ExplainDependencyResolution

Transfer
  - ExportRecord
  - ImportDefinition
  - CaptureRecord
  - ForkRecord
```

---

# 15. Artifact Store persistence

Artifact Store should use one metadata repository.

Recommended initial implementation:

```text
SQLite metadata database
+ filesystem blob and asset store
```

The metadata repository persists:

```text
artifact_roots
artifact_sources
root_source_attachments
artifact_packages
catalog_resources
canonical_definitions
artifact_records
artifact_collections
root_catalog_generations
artifact_dependencies
artifact_transfer_provenance
```

No runtime state is persisted in these tables.

---

# Part B: Workspace Feature

## 16. Workspace feature purpose

Workspace is a typed Artifact Store consumer.

Workspace provides:

- Workspace Root semantics.
- Workspace discovery plans.
- Workspace artifact frontends.
- Workspace resource grouping.
- Workspace catalog views.
- Workspace resource projection.
- Workspace reference resolution.
- Workspace composition.
- Runtime projector inputs.

Workspace does not own a separate persistence database.

## 17. Workspace root model

A Workspace Root is:

```text
ArtifactRoot
  Kind = flexigpt.workspace
```

### Workspace root data

```text
WorkspaceRootData
  - Mode: filesystem or empty
  - PrimarySourceID, optional
  - RootTrustReference
  - DiscoveryPreferences
  - AttachedPackagePreferences
  - CapabilityProfileVersion
  - Display preferences
```

Filesystem path is stored in the attached `fs-directory` source config.

Empty Workspace Root has no primary filesystem source.

## 18. Workspace source attachments

A Filesystem Workspace Root has:

```text
RootSourceAttachment
  RootID: workspace root
  SourceID: filesystem source
  Role: primary
```

An attached package has:

```text
RootSourceAttachment
  RootID: workspace root
  SourceID: package source
  Role: attached-package
```

Built-in Workspace definitions may come from:

```text
RootSourceAttachment
  RootID: built-in root or workspace root
  SourceID: embedded-fs-directory source
  Role: built-in
```

## 19. Workspace definition kinds

Workspace registers artifact frontends for canonical kinds such as:

```text
flexigpt.workspace.definition
flexigpt.agent.definition
flexigpt.skill.definition
flexigpt.model.definition
flexigpt.mcp.server.definition
flexigpt.tool.definition
flexigpt.instruction.document
flexigpt.context.document
```

These canonical kinds may be emitted from source formats such as:

```text
workspace.json
workspace.yaml
SKILL.md
.mcp.json
model preset JSON
AGENTS.md
README.md
legacy assistant JSON
```

No source file needs to use a common artifact envelope.

## 20. Workspace artifact records

Workspace scans produce Catalog Resources and Canonical Definitions.

Artifact Store synchronizes them into root-local Artifact Records.

Example:

```text
Filesystem source:
  /repo/.skills/code-review/SKILL.md

Catalog Resource:
  SourceID: backend-fs
  Locator: .skills/code-review/SKILL.md
  Subresource: skill
  Kind: flexigpt.skill.definition

Canonical Definition:
  Digest: sha256:abc...

Artifact Record:
  RecordID: 019f...
  RootID: backend-workspace
  Kind: flexigpt.skill.definition
  RecordMode: linked
  TrackingMode: follow-source
```

The Artifact Record is the Workspace-local app item.

No current `SkillStore` item is created.

## 21. Workspace derived collections

Workspace may create derived Artifact Collections to support current bundle-oriented APIs.

Examples:

```text
Backend Workspace Skills
Backend Workspace Models
Backend Workspace MCP Servers
Backend Workspace Agents
```

These collections are local derived grouping.

They may be derived from:

```text
RootID
+ source package manifest locator
+ artifact kind
```

The portable source package remains separate.

The derived collection allows Workspace providers to project:

```text
ArtifactCollection.CollectionID
-> existing BundleID

ArtifactRecord.RecordID
-> existing ItemID
```

## 22. Workspace components

| Component                    | Responsibility                                        |
| ---------------------------- | ----------------------------------------------------- |
| `WorkspaceService`           | Typed façade over Artifact Store                      |
| `WorkspaceRootKindHook`      | Validates `flexigpt.workspace` root data              |
| `WorkspaceDiscoveryPlanner`  | Builds bootstrap and expanded scan plans              |
| `WorkspaceArtifactFrontends` | Parses Workspace-specific source formats              |
| `WorkspaceCatalogService`    | Groups root catalog records into Workspace categories |
| `WorkspaceCollectionPolicy`  | Creates derived collections where required            |
| `WorkspaceResourceProjector` | Maps Artifact Records to existing domain shapes       |
| `WorkspaceReferenceResolver` | Resolves Workspace artifact references                |
| `WorkspaceLoadComposer`      | Produces Workspace Load Plans                         |
| `WorkspaceRuntimeProjectors` | Builds runtime-ready domain inputs                    |

---

# 23. Workspace discovery workflow

## 23.1 Select Filesystem Workspace Root

```text
Frontend
-> WorkspaceService.SelectFilesystemRoot(path)

WorkspaceService
-> ArtifactService.CreateRoot(
     kind=flexigpt.workspace,
   )

-> ArtifactService.CreateSource(
     kind=fs-directory,
     config={rootPath:path},
   )

-> ArtifactService.AttachSource(
     rootID,
     sourceID,
     role=primary,
   )

-> WorkspaceDiscoveryPlanner.BuildBootstrapPlan

-> ArtifactService.ScanRoot(
     rootID,
     bootstrapPlan,
   )

-> ArtifactService.SyncRecords(
     rootID,
     workspaceRecordPolicy,
   )

-> WorkspaceCatalogService.Query(rootID)
```

## 23.2 Bootstrap discovery

Workspace bootstrap plan may inspect:

```text
.flexigpt/workspace.json
.flexigpt/workspace.yaml
.mcp.json
.mcps.json
mcp.json
mcps.json
AGENTS.md
README.md
.flexigpt/agents/
.flexigpt/models/
.skills/
```

Workspace owns these conventions.

Artifact Store only executes the plan.

## 23.3 Expanded discovery

After a Workspace Definition is identified:

```text
Workspace Definition
-> WorkspaceDiscoveryPlanner.BuildExpandedPlan
-> ArtifactService.ScanRoot
-> ArtifactService.SyncRecords
-> new root catalog generation
-> WorkspaceCatalogService.Query
```

The Workspace Definition may add paths for:

- Skills.
- Agents.
- Models.
- MCP servers.
- Tools.
- Instructions.
- Context.
- Packages.

---

# 24. Workspace resource projection

Workspace must project Artifact Records into existing domain resource shapes.

```text
Artifact Record
+ Canonical Definition
+ Workspace Root Data
= existing domain response and runtime projector input
```

| Artifact kind                    | Existing domain shape                             |
| -------------------------------- | ------------------------------------------------- |
| `flexigpt.skill.definition`      | `Skill` and `SkillRef`                            |
| `flexigpt.tool.definition`       | `Tool` and `ToolRef`                              |
| `flexigpt.mcp.server.definition` | `MCPServerConfig` and MCP refs                    |
| `flexigpt.model.definition`      | `ProviderPreset`, `ModelPreset`, `ModelPresetRef` |
| `flexigpt.agent.definition`      | Assistant or Agent option                         |
| `flexigpt.instruction.document`  | instruction contributor                           |
| `flexigpt.context.document`      | context contributor                               |

The mapping is performed by Workspace projectors.

The frontend does not need to understand Artifact Store keys or source locators.

---

# 25. Current store integration

Current stores remain installed-resource providers.

```text
Current SkillStore
  -> InstalledSkillProvider

Current ToolStore
  -> InstalledToolProvider

Current MCP Store
  -> InstalledMCPProvider

Current ModelPresetStore
  -> InstalledModelProvider

Current AssistantPresetStore
  -> InstalledAssistantProvider
```

Workspace creates additional providers:

```text
WorkspaceSkillProvider
WorkspaceToolProvider
WorkspaceMCPProvider
WorkspaceModelProvider
WorkspaceAssistantProvider
WorkspaceContextProvider
```

A future scoped facade can merge them:

```text
ScopedSkillService
  - Installed Skill provider
  - Built-in Skill provider
  - Workspace Skill provider
```

The existing store persistence formats do not change.

---

# 26. Workspace runtime boundary

Artifact Store does not construct runtime objects.

Workspace runtime projectors consume Artifact Records and Canonical Definitions.

```text
Artifact Record
+ Canonical Definition
+ Workspace Root Data
+ local setup references
+ trust and policy state
= runtime projection
```

Examples:

```text
Workspace Skill Record
-> Skill projector
-> agentskills-go SkillDef

Workspace MCP Record
-> MCP projector
-> MCP runtime server config

Workspace Model Record
-> Model projector
-> model connector configuration

Workspace Tool Record
-> Tool projector
-> Tool Gateway configuration
```

---

# 27. Workspace refresh workflow

```text
Workspace refresh
-> WorkspaceDiscoveryPlanner.BuildCurrentPlan
-> ArtifactService.ScanRoot
-> Artifact Store updates Catalog Resources
-> new Canonical Definition digests where content changed
-> Artifact Store synchronizes Artifact Records
-> Artifact Store publishes new root catalog generation
-> WorkspaceCatalogService reloads
```

Outcomes:

| Source state    | Catalog Resource       | Artifact Record                                        |
| --------------- | ---------------------- | ------------------------------------------------------ |
| Unchanged       | Same digest            | Same resolved digest                                   |
| Content changed | New digest             | Follows, pins, or becomes stale based on tracking mode |
| File removed    | Missing                | Record remains and becomes missing                     |
| New file        | New catalog resource   | New derived linked record when policy allows           |
| Invalid file    | Invalid resource state | Existing record remains diagnosable                    |

---

# 28. Workspace import and fork workflow

```text
Workspace Artifact Record
-> Canonical Definition digest
-> Artifact Transfer Service
-> app-local fs-directory source
-> new Catalog Resource
-> new Artifact Record
-> target Artifact Collection
```

The original Workspace Artifact Record remains linked to source.

The imported or forked Artifact Record becomes independent.

---

# 29. Artifact Store and Workspace ownership summary

| Concern                         | Artifact Store |                    Workspace | Runtime |
| ------------------------------- | -------------: | ---------------------------: | ------: |
| Root persistence                |            Yes |              Typed root data |      No |
| Source persistence              |            Yes |    Creates Workspace sources |      No |
| Source driver behavior          |            Yes |                           No |      No |
| Package metadata                |            Yes | Interprets package relevance |      No |
| Catalog scan                    |            Yes |           Supplies scan plan |      No |
| Canonical definitions           |            Yes |           Supplies frontends |   Reads |
| Definition digest               |            Yes |                         Uses |   Reads |
| Artifact records                |            Yes |       Supplies record policy |   Reads |
| Collections                     |            Yes |  Creates derived collections |      No |
| Workspace discovery conventions |             No |                          Yes |      No |
| Workspace resource grouping     |             No |                          Yes |      No |
| Runtime projections             |             No |      Builds projector inputs |     Yes |
| Skill sessions                  |             No |                           No |     Yes |
| MCP connections                 |             No |                           No |     Yes |
| Model clients                   |             No |                           No |     Yes |
| Tool execution                  |             No |                           No |     Yes |

---

# 30. Final architecture statement

```text
Artifact Collection
  is the generic equivalent of the current Bundle.

Artifact Record
  is the generic equivalent of the current Item.

Artifact Source
  is where source content comes from.

Artifact Package
  is optional portable package metadata and is not a Bundle.

Catalog Resource
  is a source-local discovered definition occurrence.

Canonical Definition
  is portable normalized content identified by digest.

Artifact Root
  is the app-local mount that groups sources, records, collections, and catalog generations.

Workspace Root
  is ArtifactRoot(kind=flexigpt.workspace).

Workspace
  is a typed Artifact Store consumer.

Runtime
  remains outside Artifact Store.
```

The essential relationship is:

```text
ArtifactRoot
  -> ArtifactCollection
    -> ArtifactRecord
      -> CatalogResource
        -> CanonicalDefinition digest

ArtifactRoot
  -> ArtifactSource
    -> optional ArtifactPackage
      -> CatalogResource
```

This provides one durable artifact lifecycle system while preserving a clean distinction between:

```text
portable package
local collection
local item record
source definition
runtime behavior
```
