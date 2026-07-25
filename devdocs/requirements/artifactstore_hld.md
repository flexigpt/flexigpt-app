# Artifact Store Feature Requirements and Architecture

## 1. Purpose

This document defines the target Artifact Store feature.

Artifact Store has two equally important goals:

- Provide one local platform for managing domain Collections and Artifacts
  through common identity, Source, discovery, catalog, and local-state models.
- Make domain Collections and Artifacts easy to create, import, export, and
  share across applications, teams, and the internet without leaking local
  metadata.

It is authoritative for:

- The problem Artifact Store solves.
- The durable concepts and invariants it owns.
- The boundary between generic storage and consumer features.
- Required behavior for Collections, discovery, definitions, catalogs, and Artifacts.
- The separation between local entities and portable Collection and Artifact definitions.
- The reference and content-closure rules that keep future import and export clean.
- The current implementation status.
- The ordered work needed to satisfy the requirements.

The current implementation is a useful stepping stone. It is not an alternate
architecture or a compatibility constraint.

## 2. Feature intent

FlexiGPT manages Skills, Context, Tools, MCP declarations, model definitions,
agents, assistants, conversations, and future artifact kinds from local,
embedded, managed, package, remote, and future synchronized sources.

The initial domain model includes:

| Collection kind        | Example Artifact kinds                                                 |
| ---------------------- | ---------------------------------------------------------------------- |
| `skill.bundle`         | `agent.skill`                                                          |
| `mcp.bundle`           | `mcp.server`                                                           |
| `tool.bundle`          | Domain Tool definitions                                                |
| `workspace.collection` | Context, Skills, Tools, MCP, models, agents, and other supported kinds |
| `model.provider`       | Model presets and provider-owned portable definitions                  |
| `conversation.bundle`  | Conversations and domain-approved attachments                          |
| `agent.collection`     | Agent definitions                                                      |
| `assistant.collection` | Assistant definitions                                                  |

A Collection may be homogeneous or heterogeneous. Artifact Store does not
require every Artifact in a Collection to have the same kind.

Artifact Store provides one common substrate for:

- Source registration and safe content access.
- Feature-level grouping.
- Discovery and validation.
- Stable local identity.
- Immutable portable definitions.
- Local enablement and policy metadata.
- Coherent catalog publication.
- Diagnostics and source provenance.
- Consumer-owned runtime handoff.
- Portable Collection and Artifact definitions.
- Native-format import and export.
- Relative member, dependency, and asset references.
- Self-contained package construction.

Artifact Store must prevent each feature from inventing incompatible source,
identity, refresh, and local-state models.

It must also prevent every domain from inventing incompatible rules for local
paths, portable references, package containment, content integrity, and
export safety.

## 3. Outcomes

Artifact Store must make it possible to:

- Use one Root as a technical namespace containing many feature Collections.
- Attach one Source to multiple Collections within the same Root.
- Discover heterogeneous source formats through reusable adapters and decoders.
- Produce deterministic content-addressed definitions.
- Preserve invalid and missing source observations with diagnostics.
- Adopt selected occurrences into stable local Artifacts.
- Preserve local settings while source content changes.
- Pin expected source bindings before content exists.
- Keep Artifact identity independent of Collection-derived keys so direct
  movement can be added later without making movement an initial requirement.
- Keep portable content separate from local metadata and runtime state.
- Let consumers define Collection meaning without importing consumer code into Artifact Store.
- Represent both Collections and Artifacts as first-class portable entities.
- Import and export one Artifact without requiring its entire Collection.
- Import and export a Collection with a deterministic member and asset closure.
- Preserve native JSON, YAML, Markdown, directory, array, and package formats
  through domain codecs.
- Resolve portable relative references without storing machine-local paths in
  shared data.

## 4. Scope and boundaries

Artifact Store owns:

- Technical Roots and Root lifecycle.
- Root-scoped Source registrations.
- Collections and Collection lifecycle.
- Collection-to-Source attachments.
- Bounded source snapshots and discovery orchestration.
- Collection-scoped catalog occurrences.
- Immutable canonical definitions.
- Immutable Portable Collection Definitions.
- Portable member, dependency, asset, and external URI references.
- Stable local Artifact Records.
- Adoption, pinning, suppression, and source-derived reconciliation.
- Structured diagnostics.
- Optimistic concurrency and coherent publication.
- Planned import, export, capture, and package orchestration.
- Optional trusted source capabilities such as native local-path resolution.

Artifact Store does not own:

- Workspace, Skill, Tool, MCP, Model, Agent, or Assistant semantics.
- Runtime registration, sessions, prompts, execution, or process lifecycle.
- Secret values or credential acquisition.
- Trust or approval workflows.
- Active conversation sessions, streaming state, or provider execution state.
  A conversation domain may store portable conversation Artifacts.
- Feature-specific precedence.
- Uncontrolled network acquisition. Remote import must use an explicit,
  policy-controlled resolver or Source adapter.

Collection and Artifact import and export are planned platform capabilities.
Historical catalogs, persisted dependency snapshots, generic materialization,
and long-term transfer history remain optional extensions after the portable
contracts are stable.

## 5. Conceptual model and invariants

### 5.0 Local and portable planes

Artifact Store distinguishes local entities from portable entities:

| Local entity           | Portable entity                           |
| ---------------------- | ----------------------------------------- |
| Root-scoped Collection | Portable Collection Definition            |
| Artifact Record        | Portable Artifact Definition              |
| Collection Attachment  | Portable member reference                 |
| Source configuration   | Relative locator or external portable URI |

Local IDs, enablement, Source configuration, credentials, revisions, and
runtime policy are never exported automatically.

Portable definitions contain only shareable domain content, portable
references, and integrity metadata. A local entity may record the digest and
provenance of the portable definition from which it was imported.

### 5.1 Artifact Root

An Artifact Root is a technical namespace and trust boundary.

A Root owns:

- Sources.
- Collections.
- Namespace lifecycle and retention policy.
- Logical reachability of definitions.

A Root is not a Workspace, Skill bundle, provider, or other user-visible feature aggregate.

### 5.2 Source

A Source is a Root-scoped provider of discoverable content.

It contains app-local access configuration and exposes:

- A stable Source ID.
- A Source kind.
- Safe source-relative locators.
- Snapshot generation.
- Bounded stat, directory-read, and content-read operations.
- Optional source-specific trusted capabilities.

The invariant is:

    Source.RootID == Collection.RootID

Cross-Root reuse requires an explicit clone, import, or future sharing capability.

### 5.3 Collection

A Collection is the required feature-level aggregate.

Examples include:

- `workspace.collection`
- `skill.bundle`
- `tool.bundle`
- `model.provider`
- `mcp.bundle`
- `conversation.bundle`
- `agent.collection`
- `assistant.collection`

A Collection kind may allow one Artifact kind or many Artifact kinds.

A Collection owns:

- Display metadata and enablement.
- Consumer-owned local data.
- Source attachments.
- One current catalog.
- Artifact Records.
- Retirement state and revision.

The local Collection is not exported verbatim because it contains local state.
Its shareable representation is a Portable Collection Definition.

### 5.3.1 Portable Collection Definition

A Portable Collection Definition is an immutable, schema-versioned,
content-addressed description of shareable Collection content.

It contains a generic envelope such as:

    PortableCollectionDefinition{
      CollectionKind,
      SchemaID,
      SchemaVersion,
      LogicalName,
      LogicalVersion,
      Labels,
      Body,
      Members
    }

Its domain body is opaque to Artifact Store after registered validation.
Its member references are structurally visible so generic transfer code can
construct and verify a portable closure.

A member may be:

- Embedded in the Collection document.
- A subresource in the same document.
- A relative locator resolved from the Collection document or package root.
- An external portable URI with an optional or required expected digest.

Import creates a new local Collection, Sources, attachments, occurrences, and
Artifact Records. Portable Collection identity and digest do not replace local
Collection or Artifact IDs.

### 5.4 Collection Attachment

A Collection Attachment mounts one Source into one Collection.

It contains:

- Collection ID.
- Source ID.
- Consumer-defined role.
- Enabled state.
- Consumer-owned canonical JSON data.
- Revision and timestamps.

Artifact Store validates structure and containment. The consumer validates role
meaning and attachment data.

### 5.5 Source binding and occurrence

A Source Binding identifies source-relative intent:

    SourceBinding{
      SourceID,
      Locator,
      SubresourceLocator,
      ExpectedKind
    }

An Occurrence is a Collection catalog observation of a Source Binding:

    OccurrenceKey{
      CollectionID,
      SourceID,
      Locator,
      SubresourceLocator
    }

Occurrences may be valid, invalid, or missing. A pin may exist without a current
occurrence.

This distinction prevents Artifact identity from being derived from Collection
placement and leaves direct movement possible in the future.

Direct movement is not required for the initial implementation. When it is not
supported, a caller may export or copy portable content, add it to the target
Collection, and unadopt or delete the original. The new local Artifact receives
a new `ArtifactID`, so persistent references and local-only settings must be
updated explicitly.

### 5.6 Canonical definition

A Definition is immutable portable content identified by a deterministic digest.

It may contain:

- Artifact kind.
- Schema ID and schema version.
- Logical name and logical version.
- Portable labels.
- Portable body data.
- Portable dependency selectors.

An Artifact Definition may also declare a portable content closure containing
relative assets or package members. The closure describes portable content,
not Source configuration or a native runtime path.

A single definition may originate from one file, one subresource in an array,
or a multi-file package.

It must not contain:

- Root, Collection, Source, or Artifact IDs.
- Absolute paths.
- Local credential references or secret values.
- Local enablement.
- Diagnostics or timestamps.
- Runtime state.

Definition reachability and authorization are Root-scoped. Physical digest
deduplication may be shared only inside one trust boundary.

### 5.7 Artifact Record

An Artifact Record is the stable local representation of one adopted or pinned artifact.

It contains:

- Stable `ArtifactID`.
- Required Root and Collection membership.
- Artifact kind.
- Source Binding.
- Current resolved definition digest, when available.
- Local enablement and consumer-owned data.
- Source-derived state and diagnostics.
- Revision and timestamps.

Its stable reference is:

    ArtifactRef{RootID, ArtifactID}

Its current address is:

    ArtifactAddress{RootID, CollectionID, ArtifactID, Kind}

`ArtifactAddress` is a current projection and must not replace `ArtifactRef` in
persistent selections.

### 5.8 Collection catalog

Each Collection owns one coherent current catalog publication.

A publication records:

- Collection revision.
- Attachment revisions.
- Source registration revisions.
- Source snapshot generations.
- Discovery-plan fingerprint.
- Decoder capability fingerprint.
- Occurrences and diagnostics.
- Catalog revision and publication time.

A failed refresh leaves the prior publication unchanged.

### 5.9 Portable references and content shapes

Portable references use explicit forms rather than one ambiguous `location`
string:

    PortableContentRef{
      Locator,
      URI,
      SubresourceLocator,
      Digest,
      MediaType,
      Role
    }

Exactly one primary reference form is used:

- `Locator` is relative to the containing document or package root.
- `URI` is an external portable reference.
- Embedded content is represented by the containing schema and a subresource.

Portable locator rules include:

- Slash-separated relative paths.
- No absolute paths.
- No `.` or `..` segments.
- No archive escape.
- Deterministic case and normalization rules defined by the package format.

Absolute filesystem paths may be supplied to a local import request, but they
become private Source configuration and must not be written into portable
definitions.

An HTTP or HTTPS URI is acquisition input, not permission for discovery to
perform arbitrary network access. A resolver must enforce scheme, redirect,
address, size, timeout, media-type, and digest policy.

Supported content shapes include:

- One file producing one Artifact.
- One file or array producing several Artifacts through subresource locators.
- One directory or package producing one Artifact plus assets.
- One Collection document embedding several Artifacts.
- One Collection document referencing separate Artifact files or packages.

### 5.10 Domain extension contracts

Each domain supplies versioned implementations for the schemas it owns:

- Collection decoder and validator.
- Artifact decoder and validator.
- Native Collection and Artifact exporter.
- Portable member and content-closure enumerator.
- Optional reference and dependency extractor.
- Collection policy and Artifact projection.

Artifact Store owns:

- Generic envelopes.
- Canonicalization and digests.
- Locator and URI safety.
- Source and Collection containment.
- Publication, diagnostics, and conflict handling.
- Generic package assembly and extraction safety.

Artifact Store treats domain bodies as opaque data, but it must not mark a
definition valid until the registered domain validator succeeds. An unknown
schema may remain visible as an unsupported or invalid occurrence.

## 6. Functional requirements

| ID       | Requirement                                                                            | Priority    |
| -------- | -------------------------------------------------------------------------------------- | ----------- |
| `AS-F01` | Create, read, list, update, retire, and purge technical Roots.                         | Core        |
| `AS-F02` | Register and manage Root-scoped Sources.                                               | Core        |
| `AS-F03` | Create and manage typed Collections within a Root.                                     | Core        |
| `AS-F04` | Attach one Source to multiple Collections in the same Root.                            | Core        |
| `AS-F05` | Let consumers validate Collection kinds, data, roles, and attachment data.             | Core        |
| `AS-F06` | Traverse Source snapshots with locator, symlink, count, depth, and byte limits.        | Core        |
| `AS-F07` | Support independent Source adapters and format decoders.                               | Core        |
| `AS-F08` | Allow one candidate to emit zero, one, or multiple occurrences.                        | Core        |
| `AS-F09` | Canonicalize portable definitions and identify them by digest.                         | Core        |
| `AS-F10` | Publish one coherent current catalog per Collection.                                   | Core        |
| `AS-F11` | Preserve valid, invalid, and missing occurrences with diagnostics.                     | Core        |
| `AS-F12` | Adopt a current valid occurrence into a stable Artifact Record.                        | Core        |
| `AS-F13` | Pin an expected Source Binding before content exists.                                  | Core        |
| `AS-F14` | Support consumer-selected automatic adoption.                                          | Core        |
| `AS-F15` | Preserve local Artifact fields during source reconciliation.                           | Core        |
| `AS-F16` | Support suppression so auto-adopted Artifacts are not unintentionally recreated.       | Core        |
| `AS-F17` | Avoid Collection-derived Artifact identity so future direct movement remains possible. | Deferred    |
| `AS-F18` | Resolve Artifacts by typed `ArtifactRef` independently of Collection placement.        | Core        |
| `AS-F19` | Expose `ArtifactAddress` when current placement is required.                           | Core        |
| `AS-F20` | Provide an application-managed writable Source capability.                             | Core        |
| `AS-F21` | Keep public projections free of raw Source configuration.                              | Core        |
| `AS-F22` | Use optimistic revisions for all local mutations.                                      | Core        |
| `AS-F23` | Distinguish retirement, unadoption, suppression, deletion, and destructive purge.      | Core        |
| `AS-F24` | Match portable selectors without owning final consumer precedence.                     | Conditional |
| `AS-F25` | Normalize domain-defined Portable Collection Definitions.                              | Core        |
| `AS-F26` | Support single-file, multi-output, embedded-array, directory, and package Artifacts.   | Core        |
| `AS-F27` | Import and export individual Artifacts through domain codecs.                          | Planned     |
| `AS-F28` | Import and export Collections with deterministic member and asset closure.             | Planned     |
| `AS-F29` | Resolve relative locators and policy-approved external URIs during import.             | Planned     |
| `AS-F30` | Support history, dependency snapshots, provenance history, and materialization.        | Optional    |

## 7. Quality and security requirements

- Discovery must be deterministic for equivalent inputs.
- Catalog publication must fail on stale Collection, attachment, Source, or catalog revisions.
- Decoder and plan changes must make previous publications stale or require refresh.
- Untrusted source content must never execute during discovery.
- Definitions retrieved by digest must be revalidated against that digest.
- Invalid candidates must not hide unrelated valid candidates.
- Native local paths must remain trusted internal values.
- A digest must not by itself authorize cross-Root definition access.
- Managed Source writes must be staged, idempotent, and recoverable.
- Runtime and credential failures must not mutate portable definitions.
- Export must never include local IDs, absolute paths, Source configuration,
  secret values, local enablement, or runtime state unless a domain explicitly
  defines a safe portable equivalent.
- Archive import must reject path traversal, links, duplicate normalized paths,
  decompression bombs, and configured size or entry-limit violations.
- Remote acquisition must be explicit, bounded, and protected against unsafe
  schemes, redirects, and network destinations.
- A self-contained export must be assembled from one confirmed snapshot or
  captured immutable closure.
- Equivalent portable input and export options must produce equivalent canonical content.

## 8. Required architecture behavior

### 8.1 Refresh

A Collection refresh must:

- Read the Collection, attachments, Sources, and expected revisions.
- Open snapshots for enabled attachments.
- Build or receive a consumer-owned deterministic discovery plan.
- Discover and validate candidates.
- Store canonical definitions idempotently.
- Reconcile source-derived state for existing Artifacts.
- Apply consumer adoption decisions.
- Confirm Source snapshots.
- Atomically publish catalog and Artifact metadata changes.

### 8.2 Adoption and suppression

Artifact Store must distinguish:

- A valid but unadopted occurrence.
- An adopted Artifact.
- A pinned Source Binding.
- A suppressed typed Source Binding.

Consumer policy chooses defaults. Artifact Store enforces uniqueness and
preserves explicit user decisions.

### 8.3 Future movement compatibility

The initial implementation is not required to provide `MoveArtifact`.

The persistence design must only ensure that:

- `ArtifactID` is not derived from `CollectionID`.
- `ArtifactRef` does not contain `CollectionID`.
- Source Binding is distinct from Collection occurrence.
- Collection membership can be changed by a future operation.

Until direct movement is implemented, callers use export or copy, add, and
unadopt or delete. Artifact Store returns an explicit unsupported result for a
move request.

### 8.4 Managed Source writes

Managed authoring must use:

- A staging location.
- Format validation before publication.
- Atomic source-side rename or equivalent publication where supported.
- Refresh and adoption after source visibility.
- Compensation or orphan cleanup when metadata publication fails.

No API may claim one transaction spans an external filesystem and metadata database.

### 8.5 Import and export

Import must:

- Recognize a domain Collection or Artifact format.
- Resolve relative references against a well-defined manifest or package base.
- Acquire external URIs only through explicit policy-controlled resolvers.
- Verify declared and observed digests.
- Stage and validate all content before local publication.
- Store acquired content in a managed, package, or configured Source.
- Assign new local Root, Collection, Source, and Artifact IDs as applicable.
- Record portable definition digests and acquisition provenance locally.

Exporting one Artifact must:

- Ask the domain exporter for its native representation and content closure.
- Snapshot or capture source-linked content.
- Exclude local-only metadata.
- Emit either a native single file, native directory, or self-contained archive.

Exporting one Collection must:

- Ask the domain exporter for a Portable Collection Definition.
- Enumerate selected members and their content closures.
- Write a deterministic manifest and relative member layout.
- Rewrite vendored external references to package-relative locators.
- Report omitted, unresolved, denied, or nonportable members.

A valid self-contained archive may use a layout such as:

    collection.json
    artifacts/<member-relative-content>

The exact manifest name and member layout are domain-format decisions. Generic
package code enforces containment, limits, integrity, and deterministic output.

## 9. Generic API direction

Root and Source operations include:

    CreateRoot
    CreateSource
    UpdateSource
    RetireSource

Collection operations include:

    CreateCollection
    UpdateCollection
    RetireCollection
    AttachCollectionSource
    RefreshCollection
    GetCollectionCatalog

Artifact operations include:

    AdoptOccurrence
    PinSourceBinding
    SuppressSourceBinding
    GetArtifact
    ListCollectionArtifacts
    SetArtifactEnabled
    UpdateArtifactData
    UnadoptArtifact
    PurgeArtifact

Planned transfer operations include:

    ImportArtifact
    ExportArtifact
    ImportCollection
    ExportCollection
    CapturePortableClosure

`MoveArtifact` is a deferred capability. Until it exists, the service returns
`ErrUnsupported` and callers use add or import followed by unadoption or deletion.

Consumer APIs wrap these operations with typed feature semantics.

## 10. Current implementation status

| Capability                                  | Status                         | Current implementation                                                                           |
| ------------------------------------------- | ------------------------------ | ------------------------------------------------------------------------------------------------ |
| Safe Source adapters and snapshots          | Present                        | Filesystem and embedded adapters provide bounded, source-relative access.                        |
| Decoder registry and multi-output discovery | Present                        | Recognition, ambiguity handling, diagnostics, and subresources are implemented.                  |
| Canonical definitions and digest repository | Present                        | Canonical JSON, SHA-256 identity, integrity checks, and filesystem storage exist.                |
| Coherent publication                        | Present as reusable foundation | SQLite atomically publishes root catalogs and record reconciliation.                             |
| Stable local metadata record                | Partial                        | `record.Record` provides the mechanics but uses root-scoped `RecordID`.                          |
| Consumer adoption policy                    | Partial                        | `record.Policy.Derive` chooses automatic creation, but explicit adoption decisions are absent.   |
| Technical namespace Root                    | Incompatible                   | Current Root is the feature aggregate.                                                           |
| Root-scoped Source containment              | Not present                    | Sources are globally registered within one store.                                                |
| Collection aggregate                        | Not present                    | There is no Collection model or persistence.                                                     |
| Collection catalog and occurrence           | Not present                    | Catalogs and occurrences are Root-scoped.                                                        |
| Artifact refs and addresses                 | Not present                    | Public identity is `RecordID`.                                                                   |
| Pinning and suppression                     | Not present                    | No corresponding model or service operations exist.                                              |
| Future movement-compatible identity         | Partial                        | `RecordID` is independent, but records and occurrences are root-scoped and no Collection exists. |
| Managed writable Source                     | Not present                    | Artifact Store Source adapters are read-only.                                                    |
| Portable Collection Definition              | Not present                    | Current definitions represent Artifacts only.                                                    |
| Multi-output source documents               | Present                        | One decoder candidate can emit multiple subresource occurrences.                                 |
| Multi-file portable content closure         | Not present                    | Definitions do not enumerate package assets or export closure.                                   |
| Collection and Artifact import/export       | Not present                    | No transfer, package assembly, URI resolver, or native exporter contract exists.                 |
| Ordered schema migrations                   | Not present                    | SQLite uses one fixed schema fingerprint.                                                        |
| Shared application composition              | Not present                    | Workspace owns a private Artifact Store and Skill Store uses separate persistence.               |
| Trusted local-path handoff                  | Present                        | Filesystem Sources optionally implement `LocalPathResolver`.                                     |

## 11. Next steps

### 11.1 Resolve model invariants

- Finalize `SourceBinding`, occurrence, pin, and suppression semantics.
- Define Portable Collection Definition, Portable Artifact Definition,
  member-reference, asset-reference, and content-closure envelopes.
- Define relative locator and external URI resolution rules.
- Define domain codec, validator, reference-enumerator, and exporter contracts.
- Decide Root authorization and definition deduplication boundaries.
- Define retirement versus purge.
- Define decoder and discovery-plan fingerprints.
- Keep direct movement deferred unless a committed workflow requires it.

### 11.2 Establish evolvable persistence

- Add ordered schema migrations or start a clean development database with a migration ledger.
- Add Root containment to Sources.
- Introduce `CollectionID`, `CollectionKind`, `ArtifactID`, `ArtifactRef`, and `ArtifactAddress`.

### 11.3 Implement Collections

- Add Collection lifecycle and Collection Attachments.
- Add consumer validation ports without importing consumer packages.
- Preserve optimistic revision checks.

### 11.4 Move catalog ownership

- Port current discovery and publication mechanics to Collection scope.
- Include Collection, attachment, Source, plan, and decoder inputs in freshness checks.

### 11.5 Implement Artifact lifecycle

- Port stable record mechanics to Artifact Records.
- Add adoption, pinning, suppression, unadoption, and purge.
- Preserve local data during source-derived reconciliation.

### 11.6 Add managed Sources

- Introduce staged writable Source capabilities.
- Define idempotency, compensation, and orphan cleanup.

### 11.7 Change application composition

- Open one Artifact Store at application startup.
- Inject Collection-oriented services into Workspace and Skill consumers.
- Keep store lifecycle ownership in the application.

### 11.8 Add transfer after the core model is stable

- Implement Artifact import and export using the portable contracts defined
  before persistence work.
- Implement Collection import and export with deterministic package closure.
- Add filesystem and HTTP acquisition resolvers with explicit policy.
- Keep direct movement, historical catalogs, dependency snapshots,
  provenance history, and generic materialization deferred until committed
  workflows require them.

## 12. Acceptance outcomes

Artifact Store satisfies this document when:

- Root is only a technical namespace.
- Collections own attachments, catalogs, and Artifacts.
- Every Source, Collection, and Artifact is contained by one Root.
- One Source can participate in multiple Collections in that Root.
- Artifact identity is not derived from Collection identity, without requiring
  direct movement in the initial implementation.
- Collection occurrences are coherent and independently diagnosable.
- Automatic adoption respects explicit suppression.
- Managed content follows the same Source, occurrence, definition, and Artifact lifecycle as external content.
- Collections and Artifacts both have portable, schema-versioned representations.
- A Collection may contain one or many Artifact kinds according to domain policy.
- Artifacts can originate from one file, a subresource array, or a multi-file package.
- Individual Artifacts and complete Collections can be exported without local IDs,
  paths, credentials, or runtime state.
- Relative and remote references can be imported safely into local Sources.
- Definitions remain free of local identity, paths, credentials, and runtime state.
- Domain packages own semantic schemas, validation, native codecs, and projection.
- Runtime packages consume verified handoffs without becoming Artifact Store dependencies.
