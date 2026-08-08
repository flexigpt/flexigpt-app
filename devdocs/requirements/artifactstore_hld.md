# Artifact Store High-Level Design

## Document status and authority

This document defines the current authoritative design baseline for Artifact
Store. It is intended to supersede the prior Artifact Store HLD and its
current-scope replacement after review and adoption.

Artifact Store is the common local control plane for source-backed domain
content. This HLD defines the stable architectural roles, invariants,
requirements, and current product boundaries. It does not prescribe internal
function names, storage layouts, or transport details.

Authority is ordered as follows:

- This HLD governs intended architecture, product decisions, and change gates.
- Implemented code, embedded schemas, and public API contracts are the source
  of truth for current behavior.
- A conflict between this HLD and implemented behavior must be resolved
  explicitly by changing the design or implementation. It must not become an
  undocumented compatibility rule.

This document deliberately does not make import, export, archive, content
closure, package CAS, or direct move a future implementation commitment. Those
features require a new design decision before work begins.

## Purpose

Artifact Store manages the local installation and observation of domain content
without making local application identity part of that content.

It provides common mechanics for features such as Workspace and Skills:

- Root-scoped Source access.
- Local Collection and Attachment lifecycle.
- Bounded discovery and coherent Catalog publication.
- Immutable derived Definition storage.
- Stable local Artifact identity.
- Adoption, pinning, suppression, and local lifecycle operations.
- Managed application-content publication.
- Protected built-in topology hydration.
- Schema registration, validation, and canonicalization for supported
  source-owned Collection documents.

Features own domain meaning, discovery policy, local policy, projections, and
runtime behavior. Artifact Store must not become a second Workspace, Skill, or
runtime domain model.

## Current scope

Artifact Store currently supports three content ownership modes.

| Mode                        | Native-byte authority            | Local storage behavior                                                                                  | Runtime content source                              |
| --------------------------- | -------------------------------- | ------------------------------------------------------------------------------------------------------- | --------------------------------------------------- |
| Linked external content     | User-selected or external Source | Artifact Store indexes source state and stores semantic Definitions, but does not copy the package tree | Original Source after feature-specific verification |
| Managed application content | Application-managed Source       | The application writes complete managed package directories                                             | Managed Source                                      |
| Protected built-in content  | Embedded application package     | Trusted hydration writes the complete package to protected managed storage                              | Protected managed Source                            |

The Source adapter registry also contains an `embedded-directory` adapter.
That is an Artifact Store capability, not automatically a user-facing content
mode. The current application composition does not configure general embedded
providers for ordinary user content.

### Explicit non-goals

The following are intentionally unsupported in the current delivery:

- Artifact import or export.
- Collection import or export.
- Standalone portable package transfer.
- Archive creation or extraction.
- Generic content-closure generation.
- Generic package, blob, or tree content-addressed storage.
- Source-independent snapshots of linked content.
- Offline fallback for unavailable linked native content.
- Generic network acquisition or URI resolution.
- Generic provenance records for imported packages.
- Cross-Root transfer.
- Direct Artifact move between Collections.
- Generic dependency acquisition or resolution.
- Secret storage, secret resolution, or generic runtime process management.

`artifact.Service.Move` is intentionally unsupported.

## Goals

Artifact Store must:

- Keep portable or source-owned semantics separate from local application
  identity and policy.
- Treat Roots as local namespace and trust boundaries.
- Support multiple Source adapters behind one bounded snapshot contract.
- Permit one same-Root Source to be attached to multiple Collections.
- Permit heterogeneous Collections without importing feature semantics into
  Artifact Store.
- Preserve stable Artifact identity while source content changes.
- Keep Catalog Observations separate from locally managed Artifacts.
- Publish one coherent current Catalog per Collection.
- Preserve local Artifact fields during source reconciliation.
- Store immutable canonical semantic Definitions by root-scoped digest.
- Keep native source content source-backed unless application-managed or
  protected built-in hydration explicitly writes it locally.
- Provide feature services with typed lifecycle mechanics while avoiding a
  public raw Collection mutation API.
- Protect built-in topology from ordinary callers.
- Keep runtime handles, sessions, and native paths outside durable Artifact
  identity.

## Architectural planes and ownership

| Plane                               | Contents                                                                                                                | Owner                                            | Durable role                                     |
| ----------------------------------- | ----------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------ | ------------------------------------------------ |
| Local control plane                 | Roots, Sources, Collections, Attachments, Catalogs, Artifacts, Suppressions, revisions, diagnostics, local feature data | Artifact Store SQLite metadata                   | Local application state                          |
| Derived Definition plane            | Canonical immutable `definition.Definition` values                                                                      | Artifact Store Definition repository             | Semantic projection cache and integrity boundary |
| Native content plane                | Linked files, managed package files, embedded package bytes                                                             | Selected Source adapter or application installer | Native content authority                         |
| Shareable document validation plane | Registered Collection schemas and canonicalized documents                                                               | Registered domain codecs through Artifact Store  | Validation and canonicalization only             |
| Application installation plane      | Built-in registry, static IDs, installation defaults, hydration markers                                                 | Application composition                          | Protected topology declaration                   |
| Runtime plane                       | Runtime registrations, sessions, native paths, prompts, processes, provider state                                       | Feature and runtime services                     | Ephemeral, rebuildable state                     |

### Ownership rules

- Local Root, Source, Collection, Attachment, and Artifact IDs never enter
  source-owned or shareable documents.
- Source configuration is local and private. Normal Source summaries do not
  return it.
- Definitions are immutable semantic projections, not generic native-content
  packages.
- Source-backed native bytes remain authoritative for linked content.
- A feature may consume semantic text stored in a Definition, but Artifact
  Store does not imply that every feature runtime projection has identical
  native-source verification requirements.
- Runtime objects are derived from local records and are never durable
  Artifact identities.

## Core model

### Root

A Root is a local technical namespace and trust boundary. It owns Sources,
Collections, Definition reachability, and local lifecycle boundaries.

A Root is not a Workspace, Skill Bundle, portable package, or user content
container. Cross-Root reuse requires an explicit future design; digest equality
does not authorize cross-Root access.

Roots can be active or retired. Root retirement and purge are guarded by active
child Source and Collection lifecycle checks. Application composition may also
declare a Root retained or protected for narrower application policy reasons.

### Source

A Source is a Root-scoped provider of bytes and snapshots. It owns:

- Source kind and normalized private configuration.
- Enabled and retired state.
- Revision and snapshot generation.
- Bounded stat, directory-read, and content-read operations.
- Optional trusted native local-path resolution.
- Optional complete managed-package publication and removal.

A Source does not own Collection meaning, Artifact identity, feature policy,
portable package identity, or runtime state.

Source configuration is write-only in normal administration APIs. Updating a
Source without replacement configuration preserves its existing normalized
private configuration.

### Collection

A Collection is a local feature aggregate. It has a local kind, display data,
enablement, local feature data, Attachments, one current Catalog, Artifacts,
Suppressions, revisions, and retirement state.

Collections may be homogeneous or heterogeneous. Artifact Store does not
require all Artifacts in a Collection to share a kind.

A local Collection record is not a portable Collection document and is not
exported.

### Attachment

An Attachment is the implementation representation of a Source Mount. It
attaches one same-Root Source to one Collection and contains:

- Source and Collection identity.
- Consumer-defined role.
- Enabled state.
- Local Attachment data.
- Revision and timestamps.

Attachment data is feature-local. It may hold discovery roots, expected source
content digests, local roles, or feature policy. It is not written back into
source-owned documents automatically.

One Source can have Attachments in multiple same-Root Collections. An
Attachment cannot be detached while Artifacts or Suppressions still reference
that Source in the Collection.

### Artifact and Source Binding

An Artifact is a stable local record for adopted or pinned source-backed
content. It has local identity, Collection membership, a typed Source Binding,
local settings, source-derived state, and diagnostics.

Persistent references use:

```text
ArtifactRef {
  RootID
  ArtifactID
}
```

A current placement additionally includes Collection and kind information.
Persistent consumers must use `ArtifactRef`, not a path, digest, runtime handle,
or Catalog position.

A Source Binding contains:

```text
SourceBinding {
  SourceID
  Locator
  SubresourceLocator
  ExpectedKind
}
```

Bindings are local. The current implementation has no `OutputKey`. A decoder
must not emit duplicate outputs for the same Source, locator, and subresource
locator. Supporting a further output-identity dimension requires an explicit
persistence and compatibility design.

Artifact Source Bindings are unique within a Root and Collection by Source,
locator, subresource locator, and expected kind.

### Catalog and Observation

A Catalog is the one coherent current discovery result for a Collection. It
contains Collection, Attachment, and Source revisions; source generations;
plan and decoder fingerprints; Observations; diagnostics; publication revision;
and publication time.

Current Observation states are:

| State     | Meaning                                                                                  |
| --------- | ---------------------------------------------------------------------------------------- |
| `valid`   | A decoder produced a valid Definition and exact source-content digest                    |
| `invalid` | The candidate was read but could not be accepted by discovery or decoding                |
| `missing` | A previously known candidate or subresource is no longer observed in authoritative scope |

There is no persisted `unsupported` Observation state in the current model. An
unrecognized new candidate may have no Observation. A previously known
candidate that is no longer recognized becomes missing.

Artifact source-derived states are:

| State          | Meaning                                                                         |
| -------------- | ------------------------------------------------------------------------------- |
| `available`    | Current valid Observation matches the Artifact kind and Definition              |
| `missing`      | No current matching Observation exists                                          |
| `invalid`      | Current matching Observation is invalid                                         |
| `incompatible` | A valid Observation exists at the same physical occurrence but has another kind |

An Observation is not local user identity. An Artifact remains locally
addressable when its source becomes missing or invalid. A source kind change at
the same physical occurrence does not silently create a replacement Artifact;
it makes the existing Artifact incompatible.

Disabled Sources or Attachments are not scanned during refresh. A subsequent
published Catalog has no current occurrence for their bound Artifacts, so
reconciliation marks those Artifacts missing.

### Definition

A `definition.Definition` is an immutable canonical semantic projection. It is
addressed by a Definition digest and stored in a repository partitioned by Root
reachability.

Definitions may contain parsed semantic content, such as Skill Markdown
semantics or normalized Context text. They are not a byte-for-byte backup of
linked files and do not contain arbitrary package trees, resources, scripts,
local paths, local IDs, enablement, diagnostics, credentials, or runtime
sessions.

Generic Definition selectors may exist as Definition data. Current Artifact
Store does not provide generic dependency acquisition or dependency resolution.

### Suppression

A Suppression records the local decision that a typed Source Binding must not
be automatically adopted. It is Collection-scoped and does not modify source
content.

## Base invariants

- Local identity is application-owned. Root, Source, Collection, and Artifact
  IDs never become source-owned or shareable document identity.
- Artifact Store does not allocate entity IDs. Callers or feature/application
  ID providers supply UUIDv7 values.
- Roots are trust boundaries. Definition digest equality never bypasses Root
  reachability.
- A Source is a byte provider, not a feature aggregate or package identity.
- An Attachment is local feature policy, not portable source metadata.
- Linked source trees are observed and indexed, not copied into managed package
  storage.
- Definitions are semantic projections, not a generic package CAS or offline
  native-content fallback.
- Managed content is written only through managed Source publication.
- Built-in package copying is an explicit application-owned hydration
  exception, not a linked-content fallback.
- Catalog Observations and local Artifact records are distinct.
- Source reconciliation changes only source-derived Artifact fields and
  preserves local identity, enablement, name, and local feature data.
- Every mutable local operation uses optimistic revisions. Source generations
  are snapshot/content concurrency tokens, not operation IDs.
- There is no generic idempotency-key field or acknowledged-operation cache.
- Runtime paths and runtime handles are never persistent identity.
- Artifact Store does not execute discovered content.
- Public generic administration is limited to Root, Source, Source-kind, and
  managed-package mechanics. Features own Collection and Artifact meaning.
- Import, export, closure transfer, archive handling, package CAS, and move
  remain absent until a new HLD amendment defines them.

## Identity, replay, and concurrency

Explicit creation uses caller-supplied IDs as replay identity. Repeating a
creation returns the existing entity only when its immutable creation intent
matches current implementation rules.

| Entity     | Current replay intent                                                               |
| ---------- | ----------------------------------------------------------------------------------- |
| Root       | ID, initial display name, initial description                                       |
| Source     | Root, ID, kind, display name, enabled state, normalized private configuration       |
| Collection | Root, ID, and Collection kind; feature services validate additional feature intent  |
| Artifact   | Root, Collection, binding, kind, adoption mode, name, enabled state, and local data |

Mutable changes use expected revisions. Source-side managed publication also
uses source revision and snapshot generation tokens. A source generation is a
concurrency precondition, not a durable operation identity.

## Discovery, Catalog publication, and reconciliation

A feature owns discovery policy and submits a deterministic plan. Artifact
Store owns bounded execution and publication mechanics.

The current refresh contract is:

1. Read the active Collection, Attachments, Sources, and prior Catalog.
2. Open bounded snapshots for enabled Source Attachments.
3. Execute feature-selected discovery plans and registered decoders.
4. Canonicalize and store derived Definitions idempotently by Root and digest.
5. Reconcile existing Artifact source-derived state and apply feature-provided
   automatic-adoption policy.
6. Confirm Source snapshots after potentially slow decoding and policy work.
7. Atomically replace the current Catalog and Artifact source-derived state in
   SQLite.

Catalog publication verifies:

- Collection revision.
- Attachment revisions.
- Source revisions.
- Enabled Source generations.
- Discovery-plan fingerprint.
- Decoder-capability fingerprint.
- Prior Catalog revision.

A failed or stale publication does not replace the prior Catalog. Immutable
Definitions written before a later publication failure may remain reachable in
the content repository; they are not Catalog history.

A current Catalog is current relative to persisted Collection and Source
metadata and its recorded inputs. It is not a permanent guarantee that an
external filesystem cannot change after publication. Features that need native
bytes at runtime must perform their own current source verification through the
trusted Source runtime boundary.

## Adoption, pinning, suppression, and purge

### Adoption

A feature adopts a current valid Observation by supplying an Artifact ID.
Artifact Store verifies current Catalog revision, binding uniqueness, current
Attachment membership, and valid Definition state before creating or replaying
the Artifact.

### Pinning

A feature can pin an expected typed Source Binding before valid content exists.
A pinned Artifact may begin missing, invalid, or incompatible and reconciles to
available content later without changing local identity.

### Suppression

Suppression prevents automatic adoption for one typed Source Binding. Removing
an observed Artifact may create a Suppression. Unsuppression removes only the
local decision; a later refresh applies ordinary feature adoption policy.

### Retirement and purge

- Retirement disables normal use while retaining metadata for Roots, Sources,
  and Collections.
- Unadoption removes only an observed Artifact record and leaves source bytes
  unchanged.
- Generic Artifact purge removes local Artifact metadata only.
- `PurgeAndSuppress` removes local metadata and creates suppression in one
  repository transaction.
- Artifact Store does not recursively delete linked filesystem content,
  immutable Definition files, or arbitrary managed directories after metadata
  purge.

Features own typed source-side deletion and consumer-reference policy.

## Native content, snapshots, and managed publication

### Linked content

Linked Source configuration, including filesystem paths, remains private local
metadata. Artifact Store stores Catalog state, source generation,
source-content digests, diagnostics, and derived Definitions. It does not copy
a linked package tree into managed storage and does not use an old Definition
as a native-content fallback when the Source is unavailable.

### Managed content

Managed Sources publish complete package directories rather than arbitrary
public file edits. Publication validates bounded portable relative paths,
regular-file inventory, duplicate paths, case-insensitive collisions, reserved
names, and size limits.

Managed package publication has two intentionally separate boundaries:

- Source-side staging and atomic publication where the Source adapter supports
  it.
- SQLite source revision acknowledgement and later Catalog refresh.

No API claims one transaction spans Source storage and SQLite. Retried managed
operations reuse local identity and package intent. Failures can require retry,
compensation, or source-owned orphan cleanup.

Managed package storage is application-owned tree storage. It is not generic
content-addressed package storage and does not create import provenance.

### Source adapter policy

Source adapters own direct filesystem behavior, including symlink policy,
containment behavior, and native local-path resolution. The current filesystem
adapter uses normal operating-system traversal semantics, including ordinary
symlink traversal. Artifact Store and feature services must not add a second,
conflicting generic filesystem policy above a selected adapter.

## Definitions, digests, and shareable schema documents

### Current digest vocabulary

| Value                        | Current meaning                                                                        |
| ---------------------------- | -------------------------------------------------------------------------------------- |
| Definition digest            | Canonical semantic `definition.Definition` payload                                     |
| Source-content digest        | Exact bytes of one discovered source candidate                                         |
| Source generation            | Adapter-owned identity for a Source snapshot state                                     |
| Shareable-document digest    | Canonical supported `collection.json` or `workspace.json` document                     |
| Managed Skill package digest | Local managed-authoring intent over package files                                      |
| Hydration fingerprint        | Desired protected topology, registrations, canonical documents, and package-file state |

These values are intentionally distinct. Current Artifact Store has no generic
closure digest, package digest, archive digest, or transport digest.

### Shareable schema documents

Artifact Store hosts a registry of supported shareable Collection codecs. The
registry:

- Selects a codec by entity type, Collection kind, schema ID, and schema
  version.
- Validates the JSON Schema.
- Strictly decodes and semantically validates the domain model.
- Canonicalizes JSON.
- Calculates an omitted document digest or verifies a supplied one.

Canonicalization is not persistence, import, export, acquisition, or package
transfer. Current code supports Collection documents, not a generic shareable
Artifact-document repository. There is no generic SQLite relationship between
a local Collection and a shareable document digest.

A URI may be schema-valid while still operationally unsupported. Current
features must reject any reference form for which they have no explicit
resolver or materializer.

## Runtime boundary

Runtime requests begin with `ArtifactRef`. Artifact Store and the owning
feature resolve current Artifact membership, enablement, Catalog state,
Definition compatibility, and any required Source state.

Artifact Store provides trusted source primitives for features that need native
content:

- Open and confirm a Source snapshot.
- Verify source generation.
- Verify exact source-content digest.
- Resolve a native local path only through a Source adapter that explicitly
  supports that capability.

Features decide whether their runtime projection requires native-byte
verification or can safely use an already validated semantic Definition. This
is intentionally feature-specific. Runtime sessions, registrations, native
paths, and process handles remain non-durable.

## Protected built-in topology

Built-ins are application-owned content, not linked user content.

The application-owned built-in registry declares protected local topology:

- One static protected Root ID.
- One static shared managed Source ID.
- Static Collection and Artifact IDs owned by feature registries.
- Embedded package locations.
- Static Artifact-to-package-member mappings.
- Installation defaults.

Portable built-in `collection.json` documents own Collection semantics. The
application registry owns local installation identity and topology only.

Protected hydration:

1. Validates registry declarations and embedded package documents.
2. Canonicalizes the Collection document and calculates member content digests.
3. Builds a desired hydration fingerprint covering topology, registrations,
   canonical documents, member digests, and package files.
4. Resets stale protected topology before installation when the fingerprint
   differs.
5. Ensures protected Root and Source topology.
6. Publishes complete package directories into the protected managed Source.
7. Ensures static Collections and pinned static Artifacts.
8. Refreshes scoped Catalogs and commits hydration state after convergence.

One protected managed Source can contain several package directories. Each
protected Collection Attachment scopes discovery to its package directory.

A stale hydration reset removes protected metadata, protected managed payloads,
and protected Root-scoped Definitions before rebuilding them. Hydration markers
survive until a new successful convergence commits replacement state.

Undeclared active Collections or Artifacts in canonical protected topology are
operational conflicts. They are not automatically adopted, copied, renamed, or
migrated.

Ordinary transport callers do not receive trusted installer capability. Trusted
installer context currently bypasses generic protected-Root mutation checks,
so application composition and installer code are the security boundary. This
HLD does not claim a narrower capability than the current implementation
provides.

Protected Collection and Artifact preference mutation has no ordinary public
path in the current delivery.

## API boundary

Artifact administration APIs provide:

- Root lifecycle.
- Source lifecycle and Source-kind discovery.
- Managed Source state inspection.
- Managed package publication and removal.

They do not expose private Source configuration through normal summary reads.

Feature APIs own:

- Collection creation and update.
- Attachment roles and local Attachment data.
- Discovery planning and refresh policy.
- Artifact adoption, pinning, suppression, and local settings.
- Typed lifecycle and source-side cleanup.
- Domain runtime projection.

There is no public raw Collection mutation API and no generic public
Artifact import, export, or move API.

## Security requirements

Artifact Store must:

- Treat discovered content as untrusted and never execute it during discovery.
- Bound traversal depth, entry count, candidate bytes, and total scan bytes.
- Keep Source configuration private in normal Source APIs.
- Validate managed package paths and reject unsafe path forms and collisions.
- Validate portable locators in supported schema documents.
- Verify declared source-content digests where a feature supplies them.
- Keep runtime state outside durable Artifact Store identity.
- Delegate direct filesystem link and path policy to the selected Source
  adapter.
- Avoid lexical scanning for arbitrary secret-looking user content.
- Reject archive and network acquisition work because those facilities are not
  implemented.

Artifact Store does not provide a generic secret store or resolver. Opaque
adapter configuration remains private, and application composition owns any
credential policy.

## Current-scope requirements

| ID       | Requirement                                                                                                           | Status                               |
| -------- | --------------------------------------------------------------------------------------------------------------------- | ------------------------------------ |
| `AS-C01` | Separate local control state, derived Definitions, native source content, installation metadata, and runtime state    | Implemented                          |
| `AS-C02` | Manage Root-scoped Sources with private configuration and bounded snapshots                                           | Implemented                          |
| `AS-C03` | Manage same-Root Collections and Attachments, including reuse of one Source by multiple Collections                   | Implemented                          |
| `AS-C04` | Use caller- or feature-supplied UUIDv7 identity and revision-based concurrency                                        | Implemented                          |
| `AS-C05` | Preserve stable Artifact identity through typed Source Bindings and source reconciliation                             | Implemented                          |
| `AS-C06` | Publish one coherent current Catalog per Collection                                                                   | Implemented                          |
| `AS-C07` | Preserve valid, invalid, and missing Observations separately from Artifacts                                           | Implemented                          |
| `AS-C08` | Store root-scoped immutable canonical derived Definitions                                                             | Implemented                          |
| `AS-C09` | Preserve local Artifact fields when source-derived state changes                                                      | Implemented                          |
| `AS-C10` | Support adoption, pinning, suppression, unadoption, purge, and typed lifecycle guards                                 | Implemented                          |
| `AS-C11` | Support staged managed package publication with revision and generation checks                                        | Implemented                          |
| `AS-C12` | Keep domain semantics and feature policy in registered feature services                                               | Implemented                          |
| `AS-C13` | Validate and canonicalize supported shareable Collection documents without implying persistence or transfer           | Implemented                          |
| `AS-C14` | Provide feature-controlled verified runtime handoff primitives                                                        | Implemented                          |
| `AS-C15` | Hydrate protected built-ins with app-owned static local topology                                                      | Implemented                          |
| `AS-C16` | Keep generic public mutation behind Root and Source administration while features own Collection and Artifact meaning | Implemented                          |
| `AS-C17` | Enforce current source, locator, managed-package, and discovery safety boundaries                                     | Implemented for current source modes |
| `AS-C18` | Treat the metadata layout as unreleased fresh v1 with no historical migration obligation                              | Implemented policy                   |
| `AS-C19` | Do not provide import, export, closures, archives, package CAS, provenance, or network acquisition                    | Intentional current-scope exclusion  |
| `AS-C20` | Do not support direct Artifact move                                                                                   | Intentional current-scope exclusion  |

## Current implementation status

### Implemented

- Root, Source, Collection, Attachment, Artifact, and Suppression lifecycle.
- Filesystem, managed, and embedded Source adapter infrastructure.
- Bounded discovery, decoder selection, Catalog currentness, and atomic
  Catalog publication.
- Root-scoped immutable Definition repository.
- Artifact reconciliation, adoption, pinning, suppression, local updates, and
  purge.
- Managed complete-package publication and removal.
- Registered Skill and Workspace Collection-document schema canonicalization.
- Protected built-in topology hydration and stale-state reset.
- Wails-facing Root and Source administration bindings.

### Intentionally absent

- Transfer, archive, closure, provenance, package-CAS, and move facilities.
- Generic persisted shareable-document repository.
- Generic external URI resolver.
- Catalog history retention.
- Generic physical cleanup after metadata purge.

### Fresh-schema policy

The current SQLite metadata schema is an unreleased fresh-v1 schema. It has no
supported migration from earlier development layouts. A database that does not
match the current schema must be recreated until a released compatibility
policy is explicitly adopted.

## Implementation map

| Responsibility                                           | High-level implementation area                                        |
| -------------------------------------------------------- | --------------------------------------------------------------------- |
| Root lifecycle                                           | `internal/artifactstore/root`                                         |
| Source lifecycle and adapter registry                    | `internal/artifactstore/source`                                       |
| Filesystem, managed, and embedded adapters               | `internal/artifactstore/source/fsdir`, `managed`, and `embedded`      |
| Collection and Attachment lifecycle                      | `internal/artifactstore/collection`                                   |
| Artifact lifecycle and reconciliation                    | `internal/artifactstore/artifact`                                     |
| Discovery plans, decoder registry, and bounded execution | `internal/artifactstore/discovery`                                    |
| Catalog model and currentness reads                      | `internal/artifactstore/catalog`                                      |
| Refresh orchestration                                    | `internal/artifactstore/refresh`                                      |
| Derived Definition model and repository                  | `internal/artifactstore/definition` and `definition/maprepo`          |
| Shareable document registry                              | `internal/artifactstore/shareable`                                    |
| Managed Artifact publication orchestration               | `internal/artifactstore/managedartifact`                              |
| Local metadata persistence and atomic publication        | `internal/artifactstore/sqlite`                                       |
| Protected topology and hydration coordination            | `internal/artifactstore/topology` and `internal/artifactstore/system` |
| Transport-independent administration API                 | `internal/artifactstore/api.go`                                       |

The implementation map is a reading guide, not a substitute for the
invariants in this HLD.

## Change gates

This HLD must be amended before adding any of the following:

- Artifact or Collection import/export.
- Content closure or deterministic archive support.
- Generic package CAS or linked-source snapshotting.
- Offline fallback for linked native content.
- Generic URI or network acquisition.
- Imported Source roles or generic provenance.
- Cross-Root transfer.
- Artifact move.
- A generic shareable Artifact-document repository.
- A stronger protected-installer capability model.
- Released-schema migration commitments.

A future transfer design must define ownership, local identity allocation,
package materialization, provenance, safety policy, rollback behavior, and
runtime behavior before implementation begins.

## Prior-requirement transition

The prior Artifact Store requirement family is superseded as follows when this
document is adopted:

| Prior requirement family  | Disposition                                                                                                |
| ------------------------- | ---------------------------------------------------------------------------------------------------------- |
| `AS-R01` through `AS-R05` | Retained by `AS-C01` through `AS-C07` with current state terminology                                       |
| `AS-R06`                  | Retained only for derived Artifact Definitions; no generic persisted Collection-document repository exists |
| `AS-R07` through `AS-R09` | Retained by `AS-C09` through `AS-C11`                                                                      |
| `AS-R10` through `AS-R12` | Retired from current scope because transfer and closures are unsupported                                   |
| `AS-R13` through `AS-R17` | Retained by `AS-C12`, `AS-C14`, `AS-C15`, and `AS-C16`                                                     |
| `AS-R18`                  | Retired from current scope because import provenance is unsupported                                        |
| `AS-R19`                  | Replaced by `AS-C13` and `AS-C18`                                                                          |
| `AS-R20`                  | Retained only for current source, locator, discovery, and managed-package safety in `AS-C17`               |

No prior transfer design or completion criterion remains an active requirement
under this HLD.
