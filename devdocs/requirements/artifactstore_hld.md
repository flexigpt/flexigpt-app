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

The delivered core is the current platform boundary, not a temporary parallel model. This document separates behavior that exists now from portable transfer and broader domain work that is still planned.

## 2. Feature intent

Artifact Store makes an Artifact the durable item a person can enable, inspect,
select, or use. Roots, Sources, Collections, discovery, and catalogs exist to
give that item a safe place, a current meaning, and explainable provenance.

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

An optional immutable local idempotency key may be stored with a Collection.
It is unique within one Root and Collection kind, is never part of Collection
data or a Portable Collection Definition, and is never returned through a
public projection. It exists only for convergent local provisioning workflows,
such as built-in bundle bootstrap. It does not replace `CollectionID`, does not
become an Artifact identity, and does not authorize cross-Root access.

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

The current Definition envelope carries a portable body and dependency
selectors. It does not yet carry a generic content closure. Relative assets,
package members, and their integrity metadata are transfer work, not current
Artifact Store data.

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
- Optional immutable local idempotency key.
- Revision and timestamps.

Its stable reference is:

    ArtifactRef{RootID, ArtifactID}

Its current address is:

    ArtifactAddress{RootID, CollectionID, ArtifactID, Kind}

`ArtifactAddress` is a current projection and must not replace `ArtifactRef` in
persistent selections.

An Artifact idempotency key is local-only, immutable, and omitted from public
and portable projections. Artifact Store enforces its uniqueness within one
Root, Collection, and Artifact kind. It is appropriate for a feature workflow
such as managed Skill publication, where a caller-supplied operation key must
converge concurrent creation attempts without becoming an `ArtifactID`.

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
- Bounded UTF-8 path segments with no control characters.

Absolute filesystem paths may be supplied to a local import request, but they
become private Source configuration and must not be written into portable
definitions.

Platform-specific filename mapping, case collision handling, permissions, and
symlink behavior belong to the selected Source or MapStore implementation.
Generic Artifact and feature layers must not duplicate those policies.

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
| `AS-F06` | Traverse Source snapshots with locator, count, depth, and byte limits.                 | Core        |
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

`fsdir` currently permits normal operating-system symlink traversal. Symlink containment is deferred Source-adapter policy and is not implemented here.

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

## 9. Product and API boundary

The public split is intentional. Artifact Store is the shared system of
record; feature APIs add only feature meaning. A person should not need to
know how content is stored to work with it.

### 9.1 Artifact API

The Artifact API is the entry point for Root and Source administration:

- Create, inspect, list, update, retire, and purge Roots.
- Create, inspect, list, update, retire, and purge Sources.
- List available Source kinds.
- Read a managed Source's current generation and publish or remove one whole
  managed package.

Source configuration is accepted on create or replacement but is never returned
to a client. The current Artifact API deliberately does not expose raw
Collection editing, generic Artifact purge, refresh, adoption, or pinning endpoints. Those
operations need the policy of the feature that owns the Collection.

### 9.2 Feature APIs

Feature APIs, beginning with Workspace, turn generic storage into user-facing
workflows. They own the meaning of a Collection, attachments, discovery
choices, automatic adoption, and safe Artifact actions. This keeps an
Artifact central without allowing a caller to bypass the feature that
establishes its membership and meaning.

### 9.3 Client guidance

- Treat an Artifact reference as the durable selection. Do not persist a
  source path, a catalog position, a digest, or a runtime handle as a
  substitute for it.
- Treat a catalog occurrence as an observation. It can be valid and visible
  before it has a local Artifact, and an Artifact can later become missing,
  invalid, or incompatible without losing its identity or local settings.
- Send the revision returned by a previous read for every change that asks for
  one. A conflict means the client must reload the affected Root, Source,
  Workspace, or Artifact and let the user retry.
- Use a feature API for an Artifact that belongs to that feature. Generic
  Artifact purge is an internal service capability for trusted feature code,
  not a public transport operation. A feature that has source-side deletion
  semantics must perform that workflow before local metadata is purged.

## 10. Current implementation status

| Capability                              | Status                           | Current implementation                                                                                                                                                                                                                                     |
| --------------------------------------- | -------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Root and Source lifecycle               | Present                          | Roots are technical namespaces. Sources are Root-owned, can be attached to multiple same-Root Collections, and support revision-checked lifecycle changes.                                                                                                 |
| Public Artifact API boundary            | Present and scoped               | The public API manages Roots, Sources, Source kinds, and managed package publication. Feature APIs own Artifact deletion workflows where membership or source-side cleanup matters.                                                                        |
| Source adapters and snapshots           | Present                          | Filesystem, embedded, managed, and additional adapters can provide bounded source-relative discovery access.                                                                                                                                               |
| Collections and attachments             | Present                          | Collections, attachment roles, enablement, local data, retirement, and optimistic revisions are durable platform capabilities used by feature services.                                                                                                    |
| Discovery, catalog, and reconciliation  | Present                          | Refresh publishes one Collection-scoped catalog, preserves valid, invalid, and missing observations, and reconciles source-derived Artifact state.                                                                                                         |
| Canonical definitions                   | Present                          | Immutable definitions use canonical JSON and digest identity, are Root-reachable, and are revalidated on read.                                                                                                                                             |
| Artifact lifecycle                      | Present                          | Stable Artifact references, observed adoption, pinning, suppression, local enablement, local data, unadoption, and purge are implemented for feature consumers.                                                                                            |
| Managed package authoring               | Present                          | Managed Sources publish complete staged packages, use source generations as optimistic tokens, and advance Source revision after visible content changes.                                                                                                  |
| Catalog currentness and conflicts       | Present                          | Publication checks Collection, attachment, Source, catalog, plan, decoder, and snapshot inputs. Consumers can identify stale metadata and require refresh.                                                                                                 |
| Persistence and relationship invariants | Present                          | Artifact Store uses one fresh `v1` schema in the `artifacts_v1` namespace. There is no migration ledger or legacy schema compatibility path. Database and service checks preserve active Root, Source, Collection, attachment, and Artifact relationships. |
| Source privacy and local paths          | Present                          | Public Source responses omit configuration. Trusted native paths remain an internal capability for adapters that explicitly support them.                                                                                                                  |
| Feature integration                     | Present for Workspace and Skills | Workspace and `skill.bundle` consume injected Artifact Store capabilities. Agent Skills runtime, assistant preset, conversation, and inference use Artifact-backed Skill references.                                                                       |
| Portable Collection Definition          | Linked Skill manifest present    | The canonical portable Collection envelope exists. `skill.bundle.v1` can build canonical integrity-pinned linked JSON from current Artifact Store state. Package capture, import, URI acquisition, and multi-source closure export remain unavailable.     |
| Portable content closure                | Not present                      | Definitions do not yet enumerate the assets, package members, or integrity closure required for portable export.                                                                                                                                           |
| Artifact and Collection transfer        | Planned                          | Import, export, archive assembly, acquisition, provenance capture, and native domain exporters are not implemented.                                                                                                                                        |
| Direct Artifact movement                | Deferred                         | Artifact identity is independent of Collection identity, but moving an Artifact between Collections remains unsupported.                                                                                                                                   |
| Historical catalogs and materialization | Optional                         | Historical publication retention, dependency snapshots, and generic materialization remain future extensions.                                                                                                                                              |

## 11. Current boundary and remaining work

### 11.0 Delivered core boundary

The delivered core covers Root, Source, Collection, attachment, discovery,
catalog publication, durable Artifacts, canonical definitions, and managed
package publication. Revision-checked mutations ensure that a changed Source
or Collection makes an older catalog currentness claim invalid.

The public Artifact API intentionally owns shared Root and Source management.
Collections and Artifact actions are used through feature APIs when their
meaning depends on feature policy, membership, or local settings. The current
Skill migration includes `skill.bundle`, Artifact-backed Agent Skills runtime
resolution, assistant preset references, conversation references, and inference
allow-lists.

Managed Source snapshot generations are source-owned facts. Artifact Store
persists Source revisions and catalog publication generations, but does not
store a second acknowledged-generation cache. A retry derives current state
from a confirmed Source snapshot and the caller's revision and generation
preconditions. A pending feature Artifact may hold immutable source revision
and generation preconditions only until its requested source-derived state
becomes available. Completion must remove those preconditions rather than
retaining a second stale copy of Source state.

For a non-transactional feature workflow such as managed Skill publication,
the pending Artifact may retain the immutable source revision and generation
that were used to begin that operation. Those values are operation
preconditions, not current Source state and not an acknowledged-generation
cache. A retry reuses them only while the Artifact remains unresolved. This
allows a completed source-side package write to advance Source metadata exactly
once after an interrupted acknowledgement.

Managed package generations cover every published payload directory. The
managed adapter excludes only its private staging directory. MapStore remains
the storage implementation boundary for immutable definitions and managed
package file payloads. MapStore owns file-level output behavior, including
its platform-specific path, permission, and durability handling. The managed
Source adapter owns only staged package assembly, package equivalence, and the atomic package-directory publication
boundary. Artifact Store does not layer another private-directory, permission,
symlink, or platform-specific durability abstraction around MapStore-managed
definition and package payload files. Feature code must not inspect
MapStore-managed paths or recreate MapStore containment rules.
Portable locator validation therefore remains platform-neutral. Storage
adapters reject or map platform-specific collisions when materializing content.
This avoids inheriting the broad project traversal exclusions used by external filesystem
Sources. This preserves generation-based runtime freshness for normal package
resources and scripts.

MapStore remains the sole Artifact Store implementation for immutable
definition files and application-managed package payload files. External
filesystem Sources remain direct filesystem adapters by design.

Public Source updates treat omitted configuration as "preserve current private
configuration". Callers can therefore change display name or enablement without
receiving or resending filesystem paths or other opaque Source configuration.
Managed package clients obtain their current optimistic generation through a
dedicated confirmed-state API.

Import, export, archive handling, URI acquisition, portable content closure
assembly, and direct Artifact movement remain deliberate omissions rather than
partially implemented features. Generic Collection persistence remains behind
consumer services; public domain APIs must not bypass consumer validation of
Collection kind, attachment role, and opaque local data.

## 11. Completion record and deferred work

Sections 11.1 through 11.7 are completed implementation history. They are not
remaining work items for the current Artifact Store boundary.

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

- Start a clean `artifacts_v1` namespace with one schema and reject older Artifact Store metadata rather than migrating it.
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

### 11.8 Remaining priorities

Portable transfer is the next platform-level capability. It must build on the
existing local lifecycle rather than introduce another persistence model.

- Define domain-owned native import and export contracts for Collections and
  Artifacts.
- Add portable content-closure enumeration and deterministic package assembly.
- Add explicit, policy-controlled local and remote acquisition rather than
  allowing discovery to fetch arbitrary URIs.
- Keep public mutations feature-aware when Collection meaning matters.
- Treat direct movement as a separate product decision, not an implied side
  effect of export or import.
- Keep Agent Skills registration maps process-local and derived. Every
  reconciliation must re-read Artifact Store before changing registrations;
  cached Collection membership must never decide durable eligibility.
- Keep native path validation in Source adapters. Generic Source Runtime,
  Workspace, Skill Bundle, and Artifact projection layers must not clean,
  normalize, resolve symlinks, or apply platform-specific path rules a second
  time.
- Define an explicit managed-Source physical-retention policy for Source purge.
  Current metadata purge intentionally does not add an unsafe independent
  recursive filesystem delete after a database transaction. Any future
  physical cleanup must be a Source-adapter-owned, recoverable workflow.
- Defer historical catalogs, dependency snapshots, and generic materialization
  until a committed workflow needs them.

## 12. Acceptance outcomes

The following is the full target acceptance list. The local lifecycle,
discovery, catalog, and Artifact outcomes are delivered now. Statements that
require portable closure, import/export, or relative and remote transfer remain
future acceptance criteria.

- Root is only a technical namespace.
- Collections own attachments, catalogs, and Artifacts.
- Every Source, Collection, and Artifact is contained by one Root.
- One Source can participate in multiple Collections in that Root.
- Artifact identity is not derived from Collection identity, without requiring
  direct movement in the initial implementation.
- Collection occurrences are coherent and independently diagnosable.
- Automatic adoption respects explicit suppression.
- Managed content follows the same Source, occurrence, definition, and Artifact lifecycle as external content.
- A managed source-side mutation cannot silently leave an older catalog
  current after its metadata acknowledgement is interrupted.
- Retiring a Collection does not prevent its no-longer-active Sources from
  retiring; destructive independent Source purge remains ordered after
  purging retired Collection attachment history.
- Collections and Artifacts both have portable, schema-versioned representations.
- A Collection may contain one or many Artifact kinds according to domain policy.
- Artifacts can originate from one file, a subresource array, or a multi-file package.
- Individual Artifacts and complete Collections can be exported without local IDs,
  paths, credentials, or runtime state.
- Relative and remote references can be imported safely into local Sources.
- Definitions remain free of local identity, paths, credentials, and runtime state.
- Domain packages own semantic schemas, validation, native codecs, and projection.
- Runtime packages consume verified handoffs without becoming Artifact Store dependencies.
