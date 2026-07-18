# Artifact Store and Workspace High-Level Design

- [Document Status and Scope](#document-status-and-scope)
  - [Reading Guide](#reading-guide)
- [Architectural Decisions](#architectural-decisions)
- [Artifact Store Architecture](#artifact-store-architecture)
  - [Purpose](#purpose)
  - [Architecture and Ownership](#architecture-and-ownership)
  - [Domain Model](#domain-model)
  - [Entity Definitions](#entity-definitions)
    - [Artifact Root](#artifact-root)
    - [Artifact Source](#artifact-source)
    - [Root Source Attachment](#root-source-attachment)
    - [Artifact Package](#artifact-package)
    - [Catalog Resource](#catalog-resource)
    - [Canonical Definition](#canonical-definition)
    - [Artifact Record](#artifact-record)
    - [Artifact Collection](#artifact-collection)
    - [Root Catalog Generation](#root-catalog-generation)
  - [Workspace Compatibility Mapping](#workspace-compatibility-mapping)
    - [Current Model](#current-model)
    - [Generic Artifact Store Model](#generic-artifact-store-model)
    - [Portable Package Model](#portable-package-model)
    - [Full Relationship](#full-relationship)
    - [Concept Mapping](#concept-mapping)
  - [Responsibilities](#responsibilities)
    - [Root Lifecycle](#root-lifecycle)
    - [Source Lifecycle](#source-lifecycle)
    - [Package Lifecycle](#package-lifecycle)
    - [Catalog Lifecycle](#catalog-lifecycle)
    - [Record Synchronization](#record-synchronization)
    - [Collection Lifecycle](#collection-lifecycle)
    - [Transfer Lifecycle](#transfer-lifecycle)
    - [Dependency Lifecycle](#dependency-lifecycle)
    - [Source Materialization](#source-materialization)
  - [Validation Model](#validation-model)
  - [Source Model](#source-model)
    - [Source Kind Ownership](#source-kind-ownership)
    - [Materialization Rule](#materialization-rule)
    - [Future Acquisition Integration](#future-acquisition-integration)
    - [Source Driver Contract](#source-driver-contract)
  - [Extension Points](#extension-points)
    - [Root Kind Hook](#root-kind-hook)
    - [Artifact Frontend](#artifact-frontend)
    - [Collection Kind Hook](#collection-kind-hook)
    - [Dependency Resolver](#dependency-resolver)
  - [Scanning Workflow](#scanning-workflow)
  - [Record Synchronization Workflow](#record-synchronization-workflow)
  - [Public API](#public-api)
  - [Persistence](#persistence)
- [Workspace Integration](#workspace-integration)
  - [Workspace Purpose](#workspace-purpose)
  - [Workspace Root Model](#workspace-root-model)
  - [Workspace root data](#workspace-root-data)
  - [Workspace Source Attachments](#workspace-source-attachments)
  - [Workspace Definition Kinds](#workspace-definition-kinds)
  - [Workspace Artifact Records](#workspace-artifact-records)
  - [Workspace Derived Collections](#workspace-derived-collections)
  - [Implemented Components](#implemented-components)
  - [Discovery Workflow](#discovery-workflow)
    - [Select a Filesystem Workspace Root](#select-a-filesystem-workspace-root)
    - [Bootstrap Discovery](#bootstrap-discovery)
    - [Expanded Discovery](#expanded-discovery)
  - [Resource Projection](#resource-projection)
  - [Existing Store Integration](#existing-store-integration)
  - [Runtime Boundary](#runtime-boundary)
  - [Refresh Workflow](#refresh-workflow)
  - [Import and Fork Workflow](#import-and-fork-workflow)
  - [Ownership Summary](#ownership-summary)
  - [Architecture Summary](#architecture-summary)
  - [Current state record](#current-state-record)
  - [Remaining risks and next steps](#remaining-risks-and-next-steps)

## Document Status and Scope

| Attribute  | Details                                                                                                               |
| ---------- | --------------------------------------------------------------------------------------------------------------------- |
| Status     | Artifact Store and Workspace are implementation baselines.                                                            |
| Scope      | The internal Artifact Store foundation and the Workspace feature built on it.                                         |
| Exclusions | Conversation configuration and persistence, runtime lifecycle, secret values, policy evaluation, and execution logic. |
| Objective  | Establish a durable, minimal artifact lifecycle model that Workspace uses first and other stores may adopt later.     |

### Reading Guide

- `Artifact Store Architecture` describes the implemented `internal/artifactstore` architecture.
- `Workspace Integration` describes the implemented typed Workspace consumer.
- `implemented` describes code currently present in Artifact Store. `planned` describes intended future behavior.

## Architectural Decisions

| ID      | Status                  | Decision                                                                                                                                                          |
| ------- | ----------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `AS-01` | Implemented             | Artifact Store owns app-local artifact lifecycle metadata and coordinates portable content through repository ports.                                              |
| `AS-02` | Implemented             | Workspace does not introduce a separate persistent Workspace Store database.                                                                                      |
| `AS-03` | Implemented             | Workspace is a typed consumer of Artifact Store.                                                                                                                  |
| `AS-04` | Implemented             | Existing domain stores remain physically unchanged during Workspace adoption.                                                                                     |
| `AS-05` | Implemented mapping     | A current Bundle maps to generic `ArtifactCollection`.                                                                                                            |
| `AS-06` | Implemented mapping     | A current item maps to generic `ArtifactRecord`.                                                                                                                  |
| `AS-07` | Implemented             | Portable package content and app-local collections are separate concepts and storage domains.                                                                     |
| `AS-08` | Implemented             | Portable definitions exclude app IDs, local paths, secrets, runtime state, and conversation state.                                                                |
| `AS-09` | Implemented             | Source transport is an Artifact Store extension port registered during composition, not behavior implemented by frontends or consumer features.                   |
| `AS-10` | Implemented             | `NewStore` installs `fs-directory` and `embedded-fs-directory`. `memory-directory` is a contract and test-only injection point, not a default driver.             |
| `AS-11` | Implemented             | Artifact revisions are canonical definition digests, not generated revision IDs.                                                                                  |
| `AS-12` | Implemented             | Catalog resources use source occurrence identity, not generated resource IDs.                                                                                     |
| `AS-13` | Implemented             | `ArtifactRecord` is the sole generic app-side artifact item.                                                                                                      |
| `AS-14` | Implemented             | Registered frontends own source-format recognition and decoding. The portable-definition frontend is registered by default.                                       |
| `AS-15` | Implemented             | Runtime projections and execution remain outside Artifact Store.                                                                                                  |
| `AS-16` | Implemented             | A Workspace Root is `ArtifactRoot(kind=workspace.root)`.                                                                                                          |
| `AS-17` | Implemented             | Workspace discovery synchronizes selected catalog resources into root-local records.                                                                              |
| `AS-18` | Implemented generically | Linked, captured, forked, and app-local record modes exist. Their Workspace-specific use is planned.                                                              |
| `AS-19` | Implemented             | Root scans atomically publish source observations and immutable root catalog snapshots in metadata storage.                                                       |
| `AS-20` | Implemented             | Read-only source projection and source-kind-specific transfer writes use separate optional materializer ports.                                                    |
| `AS-21` | Implemented             | Frontend ownership is selected against the complete Store registry. Scan allowlists limit publication but cannot cause a different frontend to claim a candidate. |
| `AS-22` | Implemented             | Frontends may declaratively request bounded source asset roots. Artifact Store owns traversal, reads, digests, and portable-content persistence.                  |
| `AS-23` | Implemented             | Root-kind dependency resolvers choose consumer precedence while Artifact Store owns graph construction, cycle detection, and snapshot persistence.                |
| `AS-24` | Implemented             | Workspace-derived record names include a stable source-occurrence suffix and do not depend on mutable definition content.                                         |
| `AS-25` | Implemented             | App-managed sidecar and compensation directories are reserved source paths and are excluded by the filesystem source driver.                                      |

## Artifact Store Architecture

### Purpose

Artifact Store is a generic internal FlexiGPT component that manages portable artifact definitions and app-local artifact records.

The implemented public façade is `artifactstore.Store`. Its business methods
operate through injected ports rather than directly through SQLite, MapStore,
or operating-system filesystem data access.

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

### Architecture and Ownership

The implementation separates domain orchestration from persistence, portable
content, source I/O, and runtime projection. `artifactstore.Store` is the
business façade and does not embed adapter-specific persistence or source
transport behavior.

| Layer                    | Implemented location                                     | Owns                                                                                                    | Does not own                                                        |
| ------------------------ | -------------------------------------------------------- | ------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------- |
| Contracts                | `internal/artifactstore/spec`                            | Portable and app-local entities, ports, errors, and operation payloads                                  | SQLite, MapStore, source I/O, Workspace semantics                   |
| Canonical codec          | `internal/artifactstore/baseutils`                       | Canonical JSON encoding, canonical definition encoding, SHA-256 digests                                 | Persistence and source traversal                                    |
| Business layer           | `internal/artifactstore`                                 | Lifecycle rules, validation orchestration, scanning, synchronization, transfer coordination, registries | SQL statements, MapStore calls, direct filesystem reads             |
| Metadata adapter         | `internal/artifactstore/metadatastore`                   | SQLite app-local metadata, repository transactions, optimistic checks, relational constraints           | Canonical definition bodies, assets, frontend parsing, source reads |
| Portable content adapter | `internal/artifactstore/contentstore`                    | MapStore-backed definitions, assets, and portable package manifests                                     | Roots, sources, records, collections, catalog publication           |
| Source adapters          | `internal/artifactstore/sourcedriver`                    | Source configuration, traversal safety, reads, snapshots, source generations                            | Catalog persistence, frontend selection, domain interpretation      |
| Materialization adapters | `internal/artifactstore/materializer` and injected ports | Stable source projection and source-kind-specific transfer writes                                       | Root metadata and runtime execution                                 |
| Typed consumer           | Workspace or another feature                             | Typed semantics, frontend semantics, scan plans, record derivation, projections                         | Generic storage adapters and source transport                       |

`NewStore(baseDir)` opens SQLite metadata at `artifactstore.sqlite`, opens a
MapStore-backed portable content repository under `artifact-content`, registers
the portable-definition frontend, and installs the `fs-directory` and
`embedded-fs-directory` drivers. It does not configure a source materializer,
definition materializer, typed hook, version matcher, or consumer frontend.

`NewStoreWithMetadataRepository` permits a different metadata implementation.
Portable-content-dependent operations remain unavailable until a
`PortableContentRepository` is supplied during composition.

### Domain Model

```text
ArtifactSource
  -> mutable source-local Catalog Resources
  -> retained Catalog Resource Revisions
  -> Canonical Definition digests

Artifact Root
  -> RootSourceAttachments -> enabled ArtifactSources
  -> RootCatalogGeneration -> immutable root catalog resource snapshots
  -> ArtifactCollections -> ArtifactRecords
       -> source occurrence and resolved or pinned definition digest
```

A source catalog is shared by every root that attaches the source. A published
root catalog is a durable snapshot of the active source catalog, rather than a
live query over current source metadata.

Records retain their local identity and data independently of a source
occurrence. A captured, forked, or detached record can remain resolvable even
when its original source occurrence is no longer present in a root catalog.

### Entity Definitions

#### Artifact Root

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

##### Fields

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
workspace.root
app-library
builtin
package-mount
```

`Data` is opaque to Artifact Store.

Generic validation requires `Data` to be a bounded JSON object and requires a
schema ID when the object is not empty. A registered `RootKindHook`, when one
exists for the root kind, performs typed root-data and source-attachment
validation. Interpretation remains the responsibility of the consumer.

#### Artifact Source

An Artifact Source is a content provider.

It is responsible only for safely exposing directory-like content to Artifact Store.

The default Store installs the following source kinds:

```text
fs-directory
embedded-fs-directory
```

##### Fields

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
  - ObservationRevision
  - Diagnostics
  - CreatedAt
  - ModifiedAt
```

`SourceID` is generated and stable.

##### `fs-directory` Source

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

The `fs-directory` driver delegates local file reads, stats, and directory
listing to `LLMTools` `FSTool`. Artifact Store business logic does not perform
direct operating-system filesystem I/O.

##### `embedded-fs-directory` Source

Example configuration:

```json
{
  "providerKey": "builtins",
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

#### Root Source Attachment

A Root Source Attachment connects an Artifact Source to an Artifact Root.

Its natural key is:

```text
RootID + SourceID
```

No generated attachment ID is required.

Detaching removes only this app-local relationship. It does not delete the
source registration, source content, source catalog history, records, or
portable definitions.

##### Fields

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

#### Artifact Package

An Artifact Package is app-local metadata for an observed portable
package-manifest occurrence. The portable `PortablePackageManifest` body belongs
in `PortableContentRepository`; SQLite stores observation metadata and an
optional manifest digest.

The implemented Store exposes `PublishArtifactPackage`, `GetArtifactPackage`,
and `ListArtifactPackagesForSource`. A caller discovers and parses a manifest
before publishing its metadata. `ScanRoot` does not perform generic package
manifest discovery or automatically derive package-to-resource membership.

It is not equivalent to an Artifact Collection.

Its natural key is:

```text
SourceID + manifest locator
```

##### Fields

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

#### Catalog Resource

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

##### Fields

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

The mutable current resource is source-local. During a root scan, the active
source resources are copied into the newly published root catalog snapshot.
Root catalog readers use that immutable snapshot only after verifying that the
root's attachments and observed source generations are still current.

#### Canonical Definition

A Canonical Definition is normalized portable content.

It is identified by canonical SHA-256 digest.

```text
DefinitionDigest = sha256(canonical normalized definition)
```

##### Fields

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
```

The digest is the revision identity.

There is no generated `RevisionID`.

Definitions may be deduplicated by digest even when found in different sources.
Definition bodies are stored through `PortableContentRepository`, not in SQLite.

#### Artifact Record

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

##### Fields

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
  - Diagnostics
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

Captured, forked, and app-local records always use `pin-digest`. For every
pin-digest record, `PinnedDefinitionDigest` and
`LastResolvedDefinitionDigest` are both required and equal. This keeps detached
record resolution independent of mutable source state.

Artifact Record is the only generic app-side artifact entity.

There is no separate generic binding entity.

#### Artifact Collection

An Artifact Collection is an app-local grouping of Artifact Records.

It maps directly to current Bundle semantics.

##### Fields

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

#### Root Catalog Generation

A Root Catalog Generation is a durable scan publication record.

It does not require a random ID.

Its identity is:

```text
RootID + Generation
```

##### Fields

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

Each generation also has a durable resource snapshot in metadata. The
`SourceGenerations` map is the freshness boundary for that snapshot, not a
subscription to source changes. A changed attachment, disabled source, changed
source configuration, or missing/current-generation mismatch makes
root-catalog reads conflict until another `ScanRoot` publication succeeds.

---

### Workspace Compatibility Mapping

#### Current Model

```text
Bundle
  -> Item
```

This is a target Workspace compatibility mapping. Workspace itself is not
implemented yet, and Artifact Store has not changed any existing Bundle or item
persistence.

#### Generic Artifact Store Model

```text
ArtifactCollection
  -> ArtifactRecord -> source occurrence fields
                    -> resolved or pinned CanonicalDefinition digest

ArtifactSource
  -> CatalogResource
    -> CanonicalDefinition digest
```

#### Portable Package Model

```text
ArtifactPackage
  -> observed portable package manifest metadata
```

#### Full Relationship

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

#### Concept Mapping

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

### Responsibilities

#### Root Lifecycle

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

#### Source Lifecycle

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

#### Package Lifecycle

Artifact Store provides app-local package-observation persistence:

- Publish parsed package metadata.
- Read package metadata by source and manifest locator.
- List package metadata for a source.
- Validate package metadata and package diagnostics.

Package parsing, package discovery, portable manifest persistence, package
archive creation, and package-to-resource grouping are not generic Store
workflows. Portable package-manifest storage is available through
`PortableContentRepository`, not through a separate Store-level package export
operation.

#### Catalog Lifecycle

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
- Persist an immutable catalog-resource snapshot for that root generation.

The public scan entry point is `ScanRoot`. Source scanning is an internal part
of a root scan and is not exposed as a standalone `ScanSource` API.

#### Record Synchronization

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
- Persist synchronization updates only when the published catalog generation is
  still current.

`RecordSyncPolicy` is consumer-owned. It decides whether a valid catalog
resource creates a linked record and supplies only local derivation values.

#### Collection Lifecycle

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
- Change or clear record placement through `UpdateRecord`.

#### Transfer Lifecycle

Artifact Store provides:

- Return an `ExportedRecord` containing the portable definition envelope and a
  frontend-declared export closure.
- Persist imported definitions and assets through `PortableContentRepository`.
- Materialize import, capture, and fork payloads through an injected
  source-kind-specific `DefinitionMaterializer`.
- Create captured or forked destination records and transfer provenance.
- Invalidate the destination source observation after transfer publication.

`ExportRecord` does not create an archive or an `ArtifactPackage`. Import,
capture, and fork require an enabled destination source, an enabled root/source
attachment, and a registered `DefinitionMaterializer` for the destination
source kind. Source writes happen before the SQLite publication; failed metadata
publication triggers best-effort compensation through `DiscardDefinition`.

Portable content is immutable and may be persisted before the metadata
transaction. It can therefore remain content-addressed storage even if a later
metadata publication conflicts.

#### Dependency Lifecycle

Artifact Store provides:

- Read dependency selectors from canonical definitions.
- Query candidate catalog resources.
- Build dependency graph.
- Detect cycles.
- Report missing dependencies.
- Report ambiguous candidates.
- Persist dependency-resolution snapshots against a root catalog generation.

When a root-kind dependency resolver is registered, it may select one candidate
from the complete candidate set. The explanation retains every candidate while
the persisted resolved snapshot contains the selected candidate.

Artifact Store does not choose consumer precedence rules.

#### Source Materialization

`MaterializeSource` produces an app-local, stable real-directory projection for
consumers that require a directory path. It reads through the source driver and
delegates atomic directory publication to an injected `SourceMaterializer`.

This is separate from `DefinitionMaterializer`, which owns writes required by
import, capture, and fork. Neither materializer is installed by default.

---

### Validation Model

| Validation layer              | Owner                                        | Examples                                                              |
| ----------------------------- | -------------------------------------------- | --------------------------------------------------------------------- |
| Root validation               | Artifact Store plus optional root hook       | root metadata, typed root data, attachment rules                      |
| Source validation             | Artifact Store plus source driver            | source config normalization, filesystem config, embedded provider key |
| Source safety                 | Source driver and scan orchestration         | path traversal, containment, traversal and file limits                |
| Generic definition validation | Artifact Store and canonical codec           | locator, digest, canonical JSON, metadata fields                      |
| Structural validation         | Artifact frontend                            | JSON schema, source document shape                                    |
| Semantic validation           | Artifact frontend callback                   | consumer-specific source semantics                                    |
| Collection validation         | Artifact Store plus optional collection hook | slug, collection state, placement rules                               |
| Record validation             | Artifact Store, collection hook, frontend    | target occurrence, placement, local data                              |
| Publication consistency       | Metadata repository                          | optimistic timestamps, scan expectations, SQLite constraints          |
| Runtime validation            | Outside Artifact Store                       | Skill indexing, MCP connection, model readiness                       |
| Policy validation             | Outside Artifact Store                       | trust, approval, execution policy                                     |

---

### Source Model

#### Source Kind Ownership

Source kinds are registered with `artifactstore.Store` during application
composition. Typed consumers use those drivers through Artifact Store; they do
not implement source transport inside a frontend, root hook, or Workspace.

The public driver registry allows an application to add an approved source kind
without changing Store business logic.

| Source kind             | Owner          | Initial status                                      |
| ----------------------- | -------------- | --------------------------------------------------- |
| `fs-directory`          | Artifact Store | Required                                            |
| `embedded-fs-directory` | Artifact Store | Required                                            |
| `memory-directory`      | Artifact Store | Test-only injected driver, not installed by default |
| `git-checkout`          | Artifact Store | Future optional                                     |
| `zip-directory`         | Artifact Store | Future optional                                     |
| `cas-directory`         | Artifact Store | Future optional                                     |

#### Materialization Rule

Artifact Store has two intentionally separate materialization boundaries:

- `SourceMaterializer` projects an enabled source-driver snapshot into an
  application-owned directory publication for a file-oriented consumer.
- `DefinitionMaterializer` performs source-kind-specific destination writes for
  import, capture, and fork before metadata publication.

The Store does not provide a default publisher or definition writer. Runtime
code and typed consumers must not bypass these ports by reading source paths or
writing source content directly.

#### Future Acquisition Integration

A remote acquisition flow may materialize its verified result into an Artifact
Store-supported directory source. Network, Git, archive, credential, and trust
behavior remain outside Artifact Store and Workspace.

```text
Git sync worker
-> verified checkout directory
-> fs-directory source
```

```text
ZIP importer
-> verified extraction directory
-> fs-directory source
```

#### Source Driver Contract

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

`SourceConfigNormalizer` is an optional companion port. When a driver provides
it, Artifact Store normalizes source configuration before generic validation and
persistence. Drivers also own source-generation calculation; the Store only
uses that generation to protect scans and root catalog freshness.

---

### Extension Points

#### Root Kind Hook

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

#### Artifact Frontend

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
  ) ([]DecodedArtifact, []Diagnostic)

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

`DecodedArtifact` may include declarative source asset roots:

```text
SourceAssetRoot
  - Root
  - PortablePrefix
  - IncludePatterns
  - Recursive
```

Frontends do not receive source readers. Artifact Store executes these requests
through the source driver under scan entry, depth, file-count, per-definition
asset-count, and total-byte limits.

`DecodedArtifact` carries a source-local `SubresourceLocator` as well as the
definition emitted by a frontend. Artifact Store canonicalizes the definition,
calculates its digest, and persists portable content only after frontend
diagnostics pass generic validation.

#### Collection Kind Hook

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

#### Dependency Resolver

```go
type DependencyResolver interface {
  RootKind() RootKind

  ResolveDependency(
    ctx context.Context,
    root ArtifactRoot,
    attachments []RootSourceAttachment,
    selector ArtifactSelector,
    candidates []DependencyCandidate,
  ) (*DependencyCandidate, []Diagnostic)
}
```

The resolver does not query source content, build graphs, or persist snapshots.

### Scanning Workflow

`ScanRoot` is the public scan operation. It serializes scan publication within
one Store instance and performs the following work:

1. Reads the active root, all root/source attachments, and their source state.
2. Builds source plans for enabled attachments and snapshots each source before scanning.
3. Collects bounded candidates through the source driver.
4. Selects one registered frontend or reports a recognition tie.
5. Decodes, canonicalizes, validates, and stores portable definitions.
6. Confirms every scanned source generation has not changed.
7. Atomically publishes source observations, current source catalogs, the root
   catalog generation, and that generation's immutable resource snapshot.

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
-> separate `SyncRecords` call when the consumer chooses to synchronize records
```

A scan plan may contain:

- Explicit source locators.
- Directory roots and `path.Match` include patterns.
- Recursion settings.
- Allowed frontend IDs.
- Maximum file sizes.
- Maximum total source-read bytes.
- Maximum candidate count.
- Maximum traversal entries.
- Maximum traversal depth.
- Authoritative or partial source-catalog publication behavior.

Artifact Store executes plans.

Consumers define plans.

Frontend recognition always runs against the complete registered frontend set.
An allowlist is checked only after the globally winning frontend is known. This
prevents two roots from assigning different frontend ownership to the same
source occurrence.

With no source plans, `ScanRoot` performs an authoritative recursive scan of
every enabled attachment. With explicit plans, every planned source must be an
enabled attachment. An enabled source that has never been observed must be
planned; an already observed unplanned source remains part of the root catalog.

---

### Record Synchronization Workflow

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

`DetachRecord` changes the existing record to a captured, pinned record using
its already resolved definition. It deliberately does not copy content or
create a new source occurrence. `CaptureRecord` and `ForkRecord` are separate
transfer operations that materialize content into a destination source.

Synchronization never deletes records and does not create collections itself.
The consumer must ensure a collection exists before a `RecordSyncPolicy`
returns its ID as the placement for a newly derived record.

---

### Public API

```text
Roots and attachments
  - CreateRoot, GetRoot, GetRootIncludingDeleted, ListRoots
  - UpdateRoot, DeleteRoot
  - AttachSource, GetRootSourceAttachment, ListRootSources
  - UpdateRootSourceAttachment, DetachSource

Sources
  - CreateSource, GetSource, ListSources, UpdateSource, DeleteSource

Scanning and catalog
  - ScanRoot
  - GetCatalogResource, ListCatalogResourcesForSource
  - ListCatalogResourcesForRoot, GetRootCatalogGeneration
  - GetDefinitionByDigest, ListDefinitionHistory

Records
  - CreateRecord, GetRecord, ListRecords, UpdateRecord, RefreshRecord, SyncRecords
  - PinRecord, DetachRecord, DeleteRecord, ExportRecord

Collections
  - EnsureBaseCollection, CreateCollection
  - GetCollection, GetCollectionIncludingDeleted, ListCollections
  - UpdateCollection, DeleteCollection

Dependencies
  - GetDependencies, FindCandidates
  - BuildDependencyGraph, ExplainDependencyResolution
  - ListDependencySnapshots
  - ListTransferProvenance

Packages
  - PublishArtifactPackage
  - GetArtifactPackage, ListArtifactPackagesForSource

Transfer
  - ImportDefinition, CaptureRecord, ForkRecord

Runtime-facing source projection
  - MaterializeSource
```

### Persistence

The default Store uses two durable repositories with different ownership
boundaries:

```text
SQLite metadata
  - roots, sources, attachments, collections, records
  - package observations, source catalog resources, revisions
  - root catalog generations and immutable resource snapshots
  - dependency snapshots and transfer provenance

MapStore portable content
  - canonical definition files by digest
  - immutable asset files by digest
  - portable package manifests
```

SQLite metadata does not store authoritative canonical definition bodies, assets,
or portable package-manifest bodies. Catalog and record rows retain digests and
app-local occurrence metadata only.

The metadata adapter owns transactions for root-scan publication, record
synchronization, transfer metadata publication, and dependency snapshot
replacement. Portable-content writes are content-addressed and occur outside
those SQLite transactions.

No runtime state, live connection, secret value, policy decision, or execution
state is persisted by either Artifact Store repository.

## Workspace Integration

### Workspace Purpose

Workspace is implemented as a typed Artifact Store consumer in
`internal/workspace`. It provides a Workspace root hook, collection hook,
native source frontend, internal YAML decoder, discovery planner, record
synchronization policy, catalog service, default projectors, reference
resolution, and load-plan composition.

Workspace owns no persistence database. Artifact Store remains the durable
owner of roots, sources, catalog generations, records, and collections.

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

### Workspace Root Model

A Workspace Root is:

```text
ArtifactRoot
  Kind = workspace.root
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

### Workspace Source Attachments

A planned filesystem Workspace Root will have:

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

### Workspace Definition Kinds

When implemented, Workspace may register artifact frontends for canonical kinds such as:

```text
workspace.definition
agent.definition
skill.definition
model.definition
mcp.server.definition
tool.definition
instruction.document
context.document
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

### Workspace Artifact Records

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
  Kind: skill.definition

Canonical Definition:
  Digest: sha256:abc...

Artifact Record:
  RecordID: 019f...
  RootID: backend-workspace
  Kind: skill.definition
  RecordMode: linked
  TrackingMode: follow-source
```

The Artifact Record is the Workspace-local app item.

No current `SkillStore` item is created.

### Workspace Derived Collections

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

### Implemented Components

Workspace is implemented as the typed Artifact Store consumer described above.
The implementation keeps domain persistence and runtime construction outside
Workspace while providing typed discovery, validation, synchronization, catalog,
reference, load-plan, and projection boundaries.

| Component                    | Current responsibility                                                 |
| ---------------------------- | ---------------------------------------------------------------------- |
| `WorkspaceService`           | Typed façade over Artifact Store                                       |
| `WorkspaceRootKindHook`      | Validation of `workspace.root` data                                    |
| `WorkspaceDiscoveryPlanner`  | Bootstrap and expanded scan planning                                   |
| `WorkspaceArtifactFrontends` | Native JSON, restricted YAML, Markdown, and declared Skill asset roots |
| `WorkspaceCatalogService`    | Root catalog and record view                                           |
| `WorkspaceCollectionPolicy`  | Derived collection synchronization and transfer defaults               |
| `WorkspaceResourceProjector` | Default and injectable management projectors                           |
| `WorkspaceReferenceResolver` | Attachment-priority dependency resolver                                |
| `WorkspaceLoadComposer`      | Load-plan composition through Artifact Store dependency graphs         |
| `WorkspaceRuntimeProjectors` | Consumer-specific extension point                                      |

### Discovery Workflow

#### Select a Filesystem Workspace Root

```text
Frontend
-> planned WorkspaceService.SelectFilesystemRoot(path)

WorkspaceService
-> artifactstore.Store.CreateRoot(
     kind=workspace.root,
   )

-> artifactstore.Store.CreateSource(
     kind=fs-directory,
     config={rootPath:path},
   )

-> artifactstore.Store.AttachSource(
     rootID,
     sourceID,
     role=primary,
   )

-> WorkspaceDiscoveryPlanner.BuildBootstrapPlan

-> artifactstore.Store.ScanRoot(
     rootID,
     bootstrapPlan,
   )

-> artifactstore.Store.SyncRecords(
     rootID,
     workspaceRecordPolicy,
   )

-> WorkspaceCatalogService.Query(rootID)
```

#### Bootstrap Discovery

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

#### Expanded Discovery

After a Workspace Definition is identified:

```text
Workspace Definition
-> WorkspaceDiscoveryPlanner.BuildExpandedPlan
-> artifactstore.Store.ScanRoot
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

### Resource Projection

Workspace must project Artifact Records into existing domain resource shapes.

```text
Artifact Record
+ Canonical Definition
+ Workspace Root Data
= existing domain response and runtime projector input
```

| Artifact kind           | Existing domain shape                             |
| ----------------------- | ------------------------------------------------- |
| `skill.definition`      | `Skill` and `SkillRef`                            |
| `tool.definition`       | `Tool` and `ToolRef`                              |
| `mcp.server.definition` | `MCPServerConfig` and MCP refs                    |
| `model.definition`      | `ProviderPreset`, `ModelPreset`, `ModelPresetRef` |
| `agent.definition`      | Assistant or Agent option                         |
| `instruction.document`  | instruction contributor                           |
| `context.document`      | context contributor                               |

The mapping is performed by Workspace projectors.

The frontend does not need to understand Artifact Store keys or source locators.

Projected domain values are management projections. They are not inserted into
the existing stores. File-oriented runtime adapters must resolve
`SourceID + Locator`, or materialize canonical definition assets, before
constructing runtime objects. In particular, a projected Skill's source-relative
location is not an installed `SkillStore` filesystem registration.

### Existing Store Integration

This is a planned compatibility integration. Existing stores remain installed-resource providers.

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
Workspace providers are not persisted through the existing stores.
They project Workspace records and canonical definitions directly, preserving
the Artifact Store boundary and allowing each runtime consumer to opt in.

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

### Runtime Boundary

Artifact Store does not construct runtime objects, and a future Workspace
implementation must not add runtime behavior to Artifact Store.

Workspace runtime projectors consume Artifact Records and Canonical Definitions.

```text
Artifact Record
+ Canonical Definition
+ Workspace Root Data
+ local setup references
+ trust and policy state
= runtime projection
```

If a runtime library requires a directory rather than `fs.FS`-style source
access, Workspace may request `MaterializeSource` through the configured
`SourceMaterializer`. The resulting `RootPath` remains app-local runtime data.

No default `SourceMaterializer` or directory publisher is installed by the app
composition in this baseline. Filesystem definition transfer uses atomic
individual file writes and does not require rename support.

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

### Refresh Workflow

```text
Planned Workspace refresh
-> WorkspaceDiscoveryPlanner.BuildCurrentPlan
-> artifactstore.Store.ScanRoot
-> Artifact Store updates Catalog Resources
-> new Canonical Definition digests where content changed
-> Artifact Store publishes new root catalog generation
-> Artifact Store synchronizes Artifact Records
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

### Import and Fork Workflow

This is a planned use of the implemented generic transfer workflow.

```text
Workspace Artifact Record
-> Canonical Definition digest
-> artifactstore.Store.ImportDefinition, CaptureRecord, or ForkRecord
-> enabled attached destination source and DefinitionMaterializer
-> source-kind-specific DefinitionMaterializer
-> new Catalog Resource
-> new Artifact Record
-> target Artifact Collection
```

Transfer publication invalidates the destination source observation. Workspace
must rescan the destination root before relying on a current root catalog.

When a Workspace transfer omits `CollectionID`, Workspace ensures the derived
collection for the definition kind. Missing local name and version values are
derived from source occurrence identity and portable definition metadata.

The original Workspace Artifact Record remains linked to its source.

The imported or forked Artifact Record becomes independent.

### Ownership Summary

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

### Architecture Summary

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
  is ArtifactRoot(workspace.root).

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

### Current state record

- Workspace scan ownership is deterministic across roots.
- Scans and declarative asset acquisition are bounded.
- Skill resources become portable assets and participate in definition digests.
- Managed transfer internals cannot reappear as catalog artifacts.
- Dependency precedence is consumer-owned without duplicating graph logic.
- Dependency snapshots are persisted for Workspace load plans.
- Pinned and detached records retain frontend validation.
- Export uses historical frontend ownership.
- Workspace-derived local identities are collision-resistant and stable across content changes.
- Workspace transfers create projectable records by default.
- Initialization and attachment sagas have safer compensation.
- The local Workspace API exposes normalized source registrations.
- The HLD reflects the implemented architecture and current runtime boundary.

### Remaining risks and next steps

- Filesystem generation currently relies on entry metadata such as size and modification time. A same-size content rewrite with a restored timestamp may evade generation comparison. A hardened content-generation strategy needs either:
  - content hashing by the source driver, or
  - a trusted filesystem generation/watch provider.

- External source reads and SQLite publication cannot be one atomic transaction. The existing pre-publication generation confirmation minimizes, but cannot eliminate, the final filesystem-to-database race.

- Filesystem transfer is a bounded multi-file saga. Individual writes are atomic, but process termination between files can leave unreferenced files. Production crash recovery should add a durable transfer journal or generation marker.

- Workspace management projections do not yet constitute complete runtime providers. In particular:
  - canonical Skill assets need a runtime materializer for pinned and captured Skills,
  - embedded source Skills need an app-local real-directory projection when used by `agentskills-go`,
  - MCP relative commands and working directories need a runtime source resolver,
  - Workspace records are intentionally not inserted into existing stores.

- `NewService` performs several independent registry mutations. The wrapper closes the Store on failure, but an injected shared Store can remain partially composed. A future atomic `RegisterConsumerComponents` composition API would remove this edge case.

- Workspace catalog loading combines an immutable source catalog generation with separately read local collections and records. Concurrent local record mutations can produce an eventually consistent view. A repository-level Workspace read snapshot would be required for a fully atomic combined view.

- Tests still need to cover:
  - two roots scanning one source with different frontend allowlists,
  - source asset traversal limits and path containment,
  - transfer compensation directories remaining undiscoverable,
  - historical frontend validation,
  - dependency resolver ties and snapshot persistence,
  - record-mode invariants,
  - occurrence-derived name collisions,
  - attachment compensation,
  - transfer auto-collection,
  - Windows path and SQLite locking behavior.

Additional files useful for the next pass:

- `go.mod`, `go.sum`, and `go.work`, if present, to validate Go and dependency versions.
- Existing Artifact Store and Workspace tests or test fixtures.
- The built-in `skills`, `tools`, `mcp`, and `assistantpresets` asset trees, not only their loaders.
- Wails binding-generation configuration and expected frontend JSON contracts.
- Any intended production `SourceMaterializer` or directory publisher implementation.
- Runtime provider interfaces that will merge installed and Workspace resources.
