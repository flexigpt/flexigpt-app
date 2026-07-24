# Artifact Store

## 1. Document purpose

This HLD defines the problem Artifact Store solves, its required capabilities, its boundaries, its conceptual model, and the architectural decisions behind it.

The architecture sections intentionally do not define packages, interfaces,
database tables, SQL statements, file layouts, or implementation algorithms.
The code guide near the end is a repository-navigation aid for the current
implementation. It does not change the architectural ownership rules in this
document.

Artifact Store is a platform capability. Workspace is one consumer of it, but Artifact Store must not contain Workspace-specific meaning.

## 2. Problem statement

FlexiGPT works with several kinds of reusable artifacts:

- Skills
- Tools
- MCP server definitions
- Model presets
- Agent or assistant definitions
- Instructions
- Context documents
- Future artifact types

These artifacts can originate from different locations:

- User-selected directories
- Application-provided built-ins
- Installed libraries
- Imported packages
- Generated application content
- Future remote or synchronized sources

Without a common artifact substrate, every feature must independently solve:

- Source registration
- Discovery
- Parsing
- Validation
- Identity
- Change detection
- Local enablement
- Version tracking
- Diagnostics
- Import and export
- Dependency lookup
- Exposing catalog, record, definition, and source-relative references to
  consumer-owned runtime adapters

That creates duplicated storage models, inconsistent behavior, and unclear ownership.

Artifact Store solves this by establishing a shared lifecycle for discovering portable definitions and relating them to stable, app-local records.

## 3. Desired outcomes

Artifact Store must make it possible to:

- Discover heterogeneous artifact formats through a common source model.
- Preserve portable artifact content independently of application metadata.
- Give the application a stable local identity for an artifact even when its source changes.
- Support multiple logical roots that mount different combinations of sources.
- Detect source additions, changes, removals, and invalid content.
- Produce deterministic catalog views.
- Allow domain features to supply semantics without taking over storage or source access.
- Keep runtime execution, credentials, trust decisions, and conversations outside the store.
- Add advanced capabilities without forcing every consumer to adopt them.

## 4. Actors

### 4.1 Consumer feature

A feature such as Workspace defines:

- Root semantics
- Supported artifact kinds
- Discovery conventions
- Source attachment roles
- Record derivation rules
- Reference-resolution rules
- Projection into domain-facing models

### 4.2 Management application

The management application:

- Creates roots and sources.
- Attaches sources.
- Refreshes catalogs.
- Browses discovered resources.
- Enables, disables, or removes local records.
- Displays diagnostics.

### 4.3 Runtime provider

A runtime provider is outside Artifact Store. Its interaction with Artifact
Store is limited to generic, consumer-selected references:

- Reads current catalog, record, and canonical-definition state through
  Artifact Store ports.
- May receive a consumer-selected source-relative occurrence reference when
  the consumer requires source-backed content.
- May use trusted internal source-runtime access only when the selected source
  adapter explicitly supports the required capability.
- Builds runtime-specific values, applies policy, resolves credentials, and
  manages runtime lifecycle outside Artifact Store.
- Must preserve the consumer's authoritative persistence owner. A runtime
  projection must not imply copying a source-linked record into another store.

Artifact Store does not define runtime input shapes, runtime registration,
sessions, prompt construction, resource access, or execution behavior.

### 4.4 Transfer service

An optional transfer capability:

- Imports portable artifacts.
- Exports portable artifacts.
- Captures source-linked content.
- Forks artifacts into an application-managed destination.

This actor is not required for the core discovery and catalog use case.

## 5. Core use cases

### 5.1 Register a source

A caller registers a source containing artifact candidates.

The source registration holds app-local access configuration. It does not make that configuration portable.

Examples include:

- A local directory
- An application-owned embedded resource tree
- A future verified checkout
- A future unpacked archive

### 5.2 Mount sources into a root

A consumer creates a logical root and attaches one or more sources.

The same source may be attached to multiple roots. Each root may interpret source roles and precedence differently.

### 5.3 Discover artifacts

A consumer supplies discovery scope and supported formats.

Artifact Store:

- Traverses the selected source scope safely.
- Detects candidate files.
- Delegates source-format interpretation to the appropriate format adapter.
- Produces normalized definitions.
- Records valid, invalid, and missing source occurrences.
- Publishes a coherent catalog view.

### 5.4 Maintain stable local records

A user may need local settings that are not part of the portable definition:

- Enabled state
- Local display or organization settings
- Collection placement
- Tracking preference
- Local setup references
- User-specific policy references

A stable app-local record preserves these values while the underlying source definition changes.

### 5.5 Diagnose source and definition problems

Invalid files must remain visible as diagnosable catalog occurrences where possible.

A malformed artifact must not silently disappear or replace the last known valid application state without explanation.

### 5.6 Resolve portable references

A definition may refer to another artifact using portable attributes such as:

- Kind
- Logical name
- Version constraint
- Labels

Portable definitions must not refer to local root IDs, source IDs, record IDs, filesystem paths, or secrets.

Candidate discovery is generic. Final precedence belongs to the consumer feature.

## 6. Optional use cases

The following are valid extensions but are not prerequisites for the core store:

- Historical catalog retention
- Dependency graph persistence
- Package manifest management
- Import, export, capture, and fork
- Asset closure construction
- Materialization into a real directory
- Provenance history
- Remote acquisition
- Multiple metadata backends

These capabilities should remain separate from the core catalog lifecycle unless a committed product scenario requires them.

## 7. Scope

Artifact Store owns:

- Logical roots
- Source registrations
- Root-to-source attachments
- Safe discovery orchestration
- Source occurrence metadata
- Canonical portable definitions
- Stable local artifact records
- Current catalog publication
- Structured diagnostics
- Catalog, record, and definition read references for consumer-owned handoff
- Generic candidate matching
- Optimistic conflict detection
- Trusted internal source-runtime access for consumers that have an explicit
  source-backed need

Artifact Store may expose an optional trusted-internal native local-path
resolver for source adapters that already represent local directories. This is
not public source metadata, not portable definition data, and not generic
materialization.

Artifact Store may support, as optional extensions:

- Collections
- Packages
- Definition history
- Dependency snapshots
- Transfer provenance
- Materialization

## 8. Non-goals

Artifact Store does not own:

- Skill execution
- Tool execution
- MCP connections
- Model clients
- Agent sessions
- Conversation state
- Secret values
- Credential acquisition
- Trust decisions
- Approval decisions
- Runtime policy evaluation
- Remote Git or network acquisition
- Feature-specific reference precedence
- Domain-specific user interfaces

It also does not require every native source format to use one generic artifact file envelope.

## 9. Conceptual model

### 9.1 Artifact Root

An Artifact Root is an app-local logical scope.

It answers:

- Which sources participate in this scope?
- Which local artifact records belong to it?
- Which consumer semantics apply?
- Which current catalog publication is valid for it?

A root is not necessarily a directory.

### 9.2 Artifact Source

An Artifact Source is a registered provider of discoverable content.

It owns no feature semantics. Its responsibilities are limited to:

- Access configuration
- Safe traversal
- Content reads
- Change observation
- Source-relative identity

Source configuration is app-local and may contain local paths or provider registrations.

### 9.3 Root Source Attachment

An attachment makes a source available within a root.

It carries root-local properties such as:

- Role
- Enabled state
- Consumer-specific attachment data

The source itself remains independently registered.

### 9.4 Catalog Occurrence

A Catalog Occurrence represents an artifact found at a particular source-relative location.

Its identity is derived from:

- Source
- Locator
- Optional subresource locator

This is important because one file may emit multiple logical artifacts.

A catalog occurrence may be:

- Valid
- Invalid
- Missing

### 9.5 Canonical Definition

A Canonical Definition is immutable portable content.

It contains:

- Artifact kind
- Schema identity and version
- Logical name and version
- Portable labels
- Portable artifact data
- Portable dependency selectors

Portable asset declarations may be added by a future asset-capability
extension. They are not part of the current core definition model.

Its digest is its revision identity.

It excludes:

- Root IDs
- Source IDs
- Record IDs
- Local filesystem paths
- Secret values
- Runtime state
- User-specific configuration
- Diagnostics
- Timestamps

### 9.6 Artifact Record

An Artifact Record is the application’s stable local representation of an artifact.

It references:

- A root
- A source occurrence
- App-local settings

The record survives source refreshes and can preserve user state when source content changes or disappears.

### 9.7 Artifact Collection

A collection is an optional app-local grouping of records.

Collections are useful when the product needs:

- Bundle compatibility
- User-visible grouping
- Per-group settings
- Stable group identity

Collections are not portable packages and must not be treated as package manifests.

If grouping can be derived from root and artifact kind, persistent collections are optional.

### 9.8 Catalog Publication

A Catalog Publication is the coherent current view of a root after discovery.

It identifies:

- The root configuration used.
- The attached source observations used.
- The resulting catalog content.
- Diagnostics produced during discovery.

The core requirement is one coherent current publication. Retaining every historical publication is optional.

## 10. Key distinctions

### 10.1 Source occurrence versus definition

A source occurrence answers, “Where was this artifact observed?”

A definition answers, “What portable content does this artifact represent?”

The same definition may occur in multiple sources.

### 10.2 Definition versus record

A definition is portable and immutable.

A record is local and mutable.

Changing a record’s enabled state must not change the definition digest.

### 10.3 Collection versus package

A collection is local organization.

A package is portable distribution.

They may be related by a consumer, but they are not interchangeable.

### 10.4 Discovery versus runtime loading

Discovery validates and catalogs content.

Runtime loading creates executable or operational resources.

Artifact Store stops before runtime creation.

## 11. Functional requirements

| ID       | Requirement                                                                 | Priority                      |
| -------- | --------------------------------------------------------------------------- | ----------------------------- |
| `AS-F01` | Register and manage app-local sources.                                      | Core                          |
| `AS-F02` | Create logical roots and attach multiple sources.                           | Core                          |
| `AS-F03` | Allow consumers to define source roles and attachment-local metadata.       | Core                          |
| `AS-F04` | Discover bounded source candidates safely.                                  | Core                          |
| `AS-F05` | Support multiple native source formats through format adapters.             | Core                          |
| `AS-F06` | Allow one candidate to produce zero, one, or multiple artifact occurrences. | Core                          |
| `AS-F07` | Normalize valid artifacts into immutable portable definitions.              | Core                          |
| `AS-F08` | Identify definitions by deterministic content digest.                       | Core                          |
| `AS-F09` | Publish one coherent current catalog for a root.                            | Core                          |
| `AS-F10` | Preserve invalid and missing occurrence states with diagnostics.            | Core                          |
| `AS-F11` | Maintain stable app-local records for artifacts requiring local state.      | Core                          |
| `AS-F12` | Preserve local record settings during source refresh.                       | Core                          |
| `AS-F13` | Refresh source-linked records to their current source definition.           | Core                          |
| `AS-F14` | Resolve candidate definitions using portable selectors.                     | Core when references are used |
| `AS-F15` | Let consumers decide ambiguity and selection rules.                         | Core when references are used |
| `AS-F16` | Export or import portable definitions and assets.                           | Optional                      |
| `AS-F17` | Retain historical definition occurrence revisions.                          | Optional                      |
| `AS-F18` | Persist dependency resolution snapshots.                                    | Optional                      |
| `AS-F19` | Materialize source content for path-oriented runtimes.                      | Optional                      |
| `AS-F20` | Record transfer provenance.                                                 | Optional                      |

## 12. Quality requirements

### 12.1 Determinism

Given the same source content, format adapters, and discovery scope, the resulting definitions and catalog must be equivalent.

### 12.2 Safety

Source traversal must be bounded and reject unsafe path behavior.

Required controls include:

- Relative locators
- Root containment
- Symlink policy
- Candidate count limits
- Traversal depth limits
- Per-file limits
- Total-read limits
- Asset limits

### 12.3 Coherent publication

Readers must not observe a catalog assembled from incompatible source observations.

A refresh either publishes a coherent current view or leaves the prior view in place.

### 12.4 Conflict detection

If a root, attachment, source registration, source observation, or local record changes during an operation, the operation must fail rather than silently overwrite the newer state.

### 12.5 Portability

Portable definitions and packages must not contain machine-local or user-local information.

### 12.6 Diagnosability

Failures should identify:

- The affected root or source
- The affected source-relative locator
- The validation phase
- A stable diagnostic code
- Whether the resource is invalid, missing, ambiguous, or unsupported

### 12.7 Graceful capability failure

An operation requiring an unavailable optional capability must fail explicitly. It must not partially emulate the capability through an unsafe path.

### 12.8 Extensibility

New artifact kinds and native formats should not require changes to generic source traversal or metadata concepts.

New source transports should not require changes to artifact format adapters.

## 13. Principal lifecycle

### 13.1 Root lifecycle

A root moves through:

- Created
- Enabled
- Disabled
- Retired

Disabling a root prevents active discovery and runtime use but preserves local metadata.

Deletion should be logical unless irreversible deletion is explicitly required.

### 13.2 Source lifecycle

A source moves through:

- Registered
- Enabled
- Observed
- Changed
- Disabled
- Removed

Changing access configuration invalidates prior source observations.

Removing a source registration must be prevented while active relationships still require it.

### 13.3 Catalog occurrence lifecycle

An occurrence moves through:

- Newly discovered
- Valid
- Invalid
- Missing
- Valid again

A missing occurrence remains useful because linked records need to explain why their source is unavailable.

### 13.4 Record lifecycle

A record moves through:

- Created from a valid occurrence
- Available
- Missing
- Invalid
- Incompatible
- Removed

The record should not be automatically deleted because a source file disappears.

## 14. Architectural decisions

### 14.1 Separate portable content from app-local metadata

Reason:

- Portable content must be shareable and deduplicable.
- Local metadata contains identities, paths, preferences, and operational state.
- Mixing them makes exports unsafe and revisions unstable.

Consequence:

- Definitions are immutable.
- Records and source registrations remain mutable.
- Portable export can exclude app-local state by construction.

### 14.2 Use content digest as definition revision

Reason:

- Equivalent definitions should have equivalent revision identity.
- A generated revision ID does not establish content equivalence.
- Digest identity supports equivalence checks and deduplication.

Consequence:

- Canonicalization rules become part of the public contract.
- Definition changes create a new digest rather than mutating old content.

### 14.3 Keep source occurrence identity separate from record identity

Reason:

- Source location can disappear or change.
- The application still needs stable user settings and references.
- Multiple roots may use the same source occurrence differently.

Consequence:

- Occurrences describe discovery.
- Records describe local adoption.

### 14.4 Consumers own domain semantics

Reason:

- Generic storage cannot know what a Workspace primary source, built-in artifact, Skill, Tool, or Agent means.
- Putting those meanings in the generic layer would make Artifact Store a monolith.

Consequence:

- Consumers define typed root rules, artifact semantics, local record derivation, and reference precedence.

### 14.5 Source transport and artifact parsing are independent

Reason:

- Filesystem, embedded content, archives, and future remote snapshots are transport concerns.
- JSON, YAML, Markdown, and domain-specific formats are interpretation concerns.

Consequence:

- A source transport can support many formats.
- A format adapter can operate over many source transports.

### 14.5a Native local path resolution is optional and source-specific

Reason:

- Some runtime libraries already operate safely on an existing local directory.
- Copying an already materialized source tree adds unnecessary state and drift.

Consequence:

- A filesystem source adapter may expose a trusted internal locator-to-path capability.
- Non-filesystem sources return unsupported rather than receiving an implicit copied filesystem view.
- The consumer retains a source ID and validated locator reference until it
  explicitly requests this capability.
- Native paths remain adapter-derived implementation details rather than
  catalog fields, definition data, or public source metadata.
- Consumers remain responsible for policy, execution, and public API boundaries.

### 14.6 Publish a current root catalog atomically from the reader’s perspective

Reason:

- Runtime and management consumers need a coherent view.
- Live joins over independently changing sources can produce inconsistent results.

Consequence:

- Refresh has a publication boundary.
- A source change detected during refresh causes conflict or retry.
- Historical publication retention remains optional.

### 14.7 Keep runtime construction outside Artifact Store

Reason:

- Runtime resources depend on credentials, trust, local policy, installed libraries, and process lifecycle.
- Persisting runtime objects would couple discovery to execution.

Consequence:

- Artifact Store exposes validated definitions and local records.
- Runtime providers perform the final projection.

## 15. Extension model

The core should expose only capabilities that have clear ownership.

### 15.1 Source connector

Adds a source transport and safe content access model.

### 15.2 Format adapter

Recognizes a source candidate and produces canonical definitions and diagnostics.

### 15.3 Root policy

Defines typed root data and attachment constraints for a consumer.

### 15.4 Record policy

Determines which valid occurrences become local records and which local defaults are applied.

### 15.5 Reference resolver

Applies consumer-specific precedence to generic candidate matches.

### 15.6 Runtime projector

Converts a record and definition into a domain management or runtime input model.

Runtime projectors belong to the consumer or runtime integration, not Artifact Store.

### 15.7 Transfer capability

Provides import, export, capture, fork, asset closure, compensation, and provenance.

This should be an optional architectural capability rather than an obligatory part of catalog discovery.

## 16. Consistency model

### 16.1 External sources

Artifact Store cannot make an external directory and local metadata part of one transaction.

It can guarantee:

- The source was observed before reading.
- It was checked again before publication.
- Publication fails if the source is known to have changed.
- Readers receive a coherent published view.

It cannot guarantee that an uncontrolled source never changes in the final instant around publication.

### 16.2 Portable definitions

Definitions are immutable after publication.

A definition retrieved by digest must verify against that digest.

### 16.3 Local metadata

Local mutations use optimistic conflict detection.

A caller acting on stale local state receives a conflict rather than overwriting a concurrent change.

## 17. Security and trust boundary

Artifact Store treats discovered content as untrusted data.

It must not:

- Execute source content.
- Expand arbitrary executable configuration.
- Resolve secret values.
- Follow unsafe filesystem references.
- Treat source attachment as a trust grant.
- Interpret a successful parse as permission to execute.

Trust and execution authorization occur after discovery.

Portable definitions may contain secret references only if those references are portable identifiers with no secret value. Machine-local credential identifiers should normally remain in local record or runtime configuration.

## 18. Failure behavior

- An invalid candidate produces diagnostics and an invalid occurrence where identity is known.
- A removed candidate becomes missing after an authoritative refresh.
- An unknown candidate remains undiscovered.
- A format ownership tie fails explicitly.
- A selector with no candidates is unresolved.
- A selector with unresolved precedence is ambiguous.
- A changed source during refresh prevents publication.
- A missing optional capability returns an unsupported-capability result.
- A failed refresh does not replace the last coherent catalog.

## 19. Core acceptance outcomes

Artifact Store is successful when:

- Two roots can mount different combinations of the same registered sources.
- Refreshing one root produces a coherent catalog for that root.
- A valid native artifact becomes a deterministic canonical definition.
- Invalid content remains diagnosable.
- A changed artifact receives a new definition digest.
- A source-linked local record follows the updated definition without losing local settings.
- A missing source artifact does not silently delete its local record.
- Portable exports contain no app-local identities, paths, or secret values.
- Runtime code can consume definitions without depending on metadata persistence details.

## 20. Current implementation status

This mapping assesses the attached implementation, rather than planned architecture.

- `Implemented` means the required behavior has a concrete implementation in the current code.
- `Partial` means that a foundation or subset exists, but the full HLD requirement is not yet met.
- `Not implemented` means no implementation currently provides the requirement.

These statuses describe only Artifact Store packages and their generic
capabilities. Consumer behavior, consumer runtime integration, and product
completion are assessed by the owning consumer HLDs.

Artifact Store provides generic roots, source access, canonical definitions,
catalog publication, records, diagnostics, occurrence references, and trusted
internal source-runtime access. Consumers may use these generic references to
construct their own runtime inputs, but Artifact Store does not define,
register, execute, or manage those inputs.

### 20.1 Functional requirement mapping

| ID       | Requirement                                                                 | Status            | Current state and notes                                                                                                                                                                                                            |
| -------- | --------------------------------------------------------------------------- | ----------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `AS-F01` | Register and manage app-local sources.                                      | `Implemented`     | `source.Service` supports create, read, list, update, and delete operations. SQLite persists source metadata, while source adapters normalize app-local configuration.                                                             |
| `AS-F02` | Create logical roots and attach multiple sources.                           | `Implemented`     | `root.Service` manages roots and root-to-source attachments. The SQLite store persists attachments independently from sources and roots.                                                                                           |
| `AS-F03` | Allow consumers to define source roles and attachment-local metadata.       | `Implemented`     | Attachments provide consumer-owned `Role`, `Enabled`, and canonical local `Data` fields. Artifact Store does not interpret consumer-specific attachment data.                                                                      |
| `AS-F04` | Discover bounded source candidates safely.                                  | `Implemented`     | Discovery limits candidate count, entry count, depth, per-candidate bytes, and total bytes. Filesystem and embedded adapters validate relative locators and do not follow symbolic links; discovery ignores symlink candidates.    |
| `AS-F05` | Support multiple native source formats through format adapters.             | `Implemented`     | The decoder registry, recognition process, ownership rules, candidate diagnostics, and multi-output decode contract are implemented. Native formats remain consumer-provided extensions.                                           |
| `AS-F06` | Allow one candidate to produce zero, one, or multiple artifact occurrences. | `Implemented`     | A decoder returns a slice of decoded subresources. Discovery validates subresource locators, prevents duplicate emitted occurrence keys, and handles decoders that emit no resources.                                              |
| `AS-F07` | Normalize valid artifacts into immutable portable definitions.              | `Implemented`     | Decoders produce `definition.Definition` values, which are canonicalized and stored by immutable digest-addressed content files.                                                                                                   |
| `AS-F08` | Identify definitions by deterministic content digest.                       | `Implemented`     | Canonical JSON and SHA-256 digests are implemented. Definitions verify supplied digests and reject digest mismatches.                                                                                                              |
| `AS-F09` | Publish one coherent current catalog for a root.                            | `Implemented`     | Refresh confirms source snapshots, checks root and source revisions, then atomically publishes catalog occurrences and record reconciliation changes in one SQLite transaction.                                                    |
| `AS-F10` | Preserve invalid and missing occurrence states with diagnostics.            | `Implemented`     | Current catalogs persist valid, invalid, and missing occurrences with structured diagnostics. Authoritative discovery marks prior in-scope occurrences missing rather than deleting their records.                                 |
| `AS-F11` | Maintain stable app-local records for artifacts requiring local state.      | `Implemented`     | Records have stable UUIDv7 identities and retain root, occurrence, local data, enablement, state, diagnostics, and revision independently of definitions.                                                                          |
| `AS-F12` | Preserve local record settings during source refresh.                       | `Implemented`     | Reconciliation updates source-derived definition, state, and diagnostics while preserving local record name, enablement, and local data.                                                                                           |
| `AS-F13` | Refresh source-linked records to their current source definition.           | `Implemented`     | Reconciliation updates valid records to the definition emitted by their current source occurrence and marks missing, invalid, or incompatible records without deleting them.                                                       |
| `AS-F14` | Resolve candidate definitions using portable selectors.                     | `Partial`         | Definitions can declare and validate portable selectors. Artifact Store does not yet provide generic current-catalog candidate matching, complete version constraint support, dependency resolution, or dependency graph handling. |
| `AS-F15` | Let consumers decide ambiguity and selection rules.                         | `Implemented`     | Generic storage retains attachments, occurrences, and records without applying any consumer precedence or winner-selection policy.                                                                                                 |
| `AS-F16` | Export or import portable definitions and assets.                           | `Not implemented` | There is no package manifest, asset closure, import, export, capture, or fork capability.                                                                                                                                          |
| `AS-F17` | Retain historical definition occurrence revisions.                          | `Not implemented` | SQLite retains only `artifact_current_catalogs` and current occurrences. Historical catalog generations and occurrence revisions are not stored.                                                                                   |
| `AS-F18` | Persist dependency resolution snapshots.                                    | `Not implemented` | Definitions can declare selectors, but dependency graph construction, resolution snapshots, and reproducibility reports do not exist.                                                                                              |
| `AS-F19` | Materialize source content for path-oriented runtimes.                      | `Not implemented` | Generic materialization remains absent. Filesystem sources may expose a trusted native locator-to-path capability, which is not materialization and does not create a copied source tree.                                          |
| `AS-F20` | Record transfer provenance.                                                 | `Not implemented` | Transfer workflows and provenance metadata are not implemented.                                                                                                                                                                    |

### 20.2 Quality requirement mapping

| HLD section | Requirement                 | Status        | Current state and notes                                                                                                                                                                                                                                                                                                                      |
| ----------- | --------------------------- | ------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `12.1`      | Determinism                 | `Implemented` | Candidate traversal and occurrences are sorted, decoder ties fail explicitly, and canonical definition digests are deterministic for equivalent decoded content. Publication and observation timestamps intentionally vary between refreshes.                                                                                                |
| `12.2`      | Safety                      | `Partial`     | Locator validation, containment checks, symlink rejection, and discovery bounds are implemented. Portable asset declarations, asset traversal, and asset-specific limits are not implemented because asset support is not yet present.                                                                                                       |
| `12.3`      | Coherent publication        | `Implemented` | A refresh publishes one current catalog transactionally or leaves the prior catalog in place. Source snapshots are confirmed before publication.                                                                                                                                                                                             |
| `12.4`      | Conflict detection          | `Partial`     | Root, source, attachment-through-root, and record revisions use optimistic conflict checks. Filesystem snapshot fingerprints include regular-file content and confirmation detects observed source changes. As with any uncontrolled external source, a final change after confirmation cannot be made transactional with local publication. |
| `12.5`      | Portability                 | `Partial`     | The canonical definition envelope excludes app-local metadata by design. Definition bodies remain schema-owned opaque JSON, so each artifact decoder must still enforce that its body contains no local paths, local identities, or secret values.                                                                                           |
| `12.6`      | Diagnosability              | `Partial`     | Structured diagnostics include severity, stable code, message, and source-relative locations. Invalid and missing occurrences are persisted. Some structural refresh failures are still returned only as operation errors rather than persisted root diagnostics.                                                                            |
| `12.7`      | Graceful capability failure | `Partial`     | Explicit unsupported and unavailable errors exist, but unavailable optional capabilities do not yet expose dedicated capability endpoints or consistent unsupported-operation responses.                                                                                                                                                     |
| `12.8`      | Extensibility               | `Implemented` | Source adapters, decoder registries, root-local attachment data, record policies, and consumer-owned resolution allow new transports and artifact formats without changing generic traversal or persistence concepts.                                                                                                                        |

## 21. Code guide

This guide describes module responsibilities for code analysis. It is not a
public API or ownership contract. Read from the generic lifecycle outward;
consumer semantics and runtime behavior deliberately live outside this tree.

- `internal/artifactstore`
  - Shared identifiers, invariants, diagnostics, errors, digest utilities,
    clocks, and generic validation.
  - Defines generic invariants without defining consumer artifact semantics.
- `internal/artifactstore/jsoncanon`
  - Canonical JSON encoding and equality rules used by definitions and
    app-local JSON fields.
- `internal/artifactstore/source`
  - Source registration lifecycle, adapter registry, operational snapshots,
    and trusted internal source-runtime access.
  - `source/fsdir` provides filesystem traversal and optional local-path
    resolution; `source/embedded` provides read-only embedded trees.
- `internal/artifactstore/root`
  - Generic logical-root and root-to-source attachment lifecycle.
- `internal/artifactstore/discovery`
  - Bounded candidate traversal, decoder registration and selection,
    occurrence state generation, and source-plan processing.
- `internal/artifactstore/definition`
  - Canonical immutable definition rules and definition repository ports.
  - `definition/fsrepo` is the current digest-addressed definition backend.
- `internal/artifactstore/catalog`
  - Current catalog snapshot and occurrence read models.
- `internal/artifactstore/record`
  - Stable local records, source-derived reconciliation, and local record
    mutations such as enablement, local data updates.
- `internal/artifactstore/refresh`
  - Generic refresh orchestration and atomic publication contract.
- `internal/artifactstore/sqlite`
  - SQLite metadata persistence, catalog reads, and transactional publisher.
- `internal/artifactstore/system`
  - Internal composition of metadata storage, definition storage, source
    adapters, decoders, and generic services.
  - Exposes only the generic service and trusted-reference capabilities needed
    by internal consumer composition.

Consumer dependency direction is deliberate:

- Consumer packages such as `internal/workspace` depend on Artifact Store
  roots, sources, catalogs, records, definitions, and refresh services.
- Consumer-owned adapters transform selected catalog and source-relative
  references into their own management or runtime inputs.
- Runtime packages consume those consumer-owned inputs and do not become
  Artifact Store subpackages.
- `cmd/agentgo` performs application composition and Wails exposure.
- Artifact Store must not import consumer or runtime packages. Consumer
  convention, precedence, projection, policy, prompt, session, resource, and
  execution behavior remain outside this tree.

## 22. Next steps

- Keep consumer handoff explicit without expanding Artifact Store into a
  runtime framework.
  - Continue to expose generic catalog, record, definition, occurrence, and
    source-relative references rather than consumer-specific runtime values.
  - Keep native local-path access trusted, internal, optional, and
    source-adapter-specific.
  - Do not add runtime registration, prompt construction, session, trust,
    credential, resource, or execution semantics to Artifact Store.

- Close the remaining generic selector and diagnostic gaps.
  - Add current-catalog candidate matching for portable selectors while
    retaining final precedence and ambiguity decisions in consumer features.
  - Define complete version-constraint behavior before selectors are used for
    dependency resolution.
  - Surface structural discovery and publication failures through consistent
    root-scoped diagnostics where persistence is appropriate.

- Preserve the generic source and runtime boundary.
  - Keep native local-path resolution trusted, internal, source-specific, and
    optional.
  - Do not add consumer format conventions, prompt composition, runtime
    registration, resource access, script execution, trust, or session
    semantics to Artifact Store.
  - Keep generic materialization optional. A future implementation requires a
    separate capability with explicit ownership and safety rules.

- Keep optional capabilities deferred until a committed workflow requires them.
  - Packages, import, export, capture, fork, transfer provenance, generic
    asset closure, and generic materialization address `AS-F16`, `AS-F19`,
    and `AS-F20`.
  - Dependency graph construction, version matcher registries, persisted
    resolution snapshots, and reproducibility reporting address `AS-F18`.
  - Historical catalogs, occurrence revision history, historical source
    observations, and catalog comparison address `AS-F17`.
