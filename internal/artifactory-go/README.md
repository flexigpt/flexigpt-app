# Artifactory Go

> Turn changing source material into stable artifacts, a fast catalog, and verified reads.

Artifactory Go is an embeddable artifact storage and lifecycle system for applications that work with files, documents, and managed packages.

It connects content in a Source to immutable Definitions and stable application-facing Artifacts. It keeps source-derived state separate from local preferences, makes synchronization explicit, and provides verified access to the material behind an Artifact.

The central principle is simple:

> Know what was admitted, preserve who it belongs to, and verify what is being used.

Artifactory does not assign product meaning to an artifact. Your application supplies that meaning through document decoders, schemas, and domain behavior.

- [Why this exists](#why-this-exists)
- [Who it is for](#who-it-is-for)
- [What it is not](#what-it-is-not)
- [The mental model](#the-mental-model)
  - [1. Source truth](#1-source-truth)
  - [2. Committed catalog truth](#2-committed-catalog-truth)
  - [3. Local application state](#3-local-application-state)
  - [A concrete example](#a-concrete-example)
- [The entities and what they own](#the-entities-and-what-they-own)
  - [Root: namespace, protection, and retention](#root-namespace-protection-and-retention)
  - [Source: physical origin and discovery](#source-physical-origin-and-discovery)
  - [Definition: immutable admitted meaning](#definition-immutable-admitted-meaning)
  - [Artifact: stable identity across source changes](#artifact-stable-identity-across-source-changes)
  - [Overlay: non-secret local customization](#overlay-non-secret-local-customization)
  - [Secret: bindings are not physical values](#secret-bindings-are-not-physical-values)
- [The workflows](#the-workflows)
  - [1. Register and prepare a Source](#1-register-and-prepare-a-source)
  - [2. Observe, admit, and publish](#2-observe-admit-and-publish)
  - [3. Browse committed state](#3-browse-committed-state)
  - [4. Resolve and read verified material](#4-resolve-and-read-verified-material)
    - [Multi-read verification sessions](#multi-read-verification-sessions)
  - [5. Publish or replace a complete managed package](#5-publish-or-replace-a-complete-managed-package)
  - [6. Remove content and converge on a consistent catalog](#6-remove-content-and-converge-on-a-consistent-catalog)
  - [7. Customize locally and manage secrets](#7-customize-locally-and-manage-secrets)
    - [Atomic overlay and Artifact-local cleanup](#atomic-overlay-and-artifact-local-cleanup)
  - [8. Install and reconcile application-owned content](#8-install-and-reconcile-application-owned-content)
    - [Generated installation evidence](#generated-installation-evidence)
- [Consistency and identity rules](#consistency-and-identity-rules)
  - [Stable identity is separate from content identity](#stable-identity-is-separate-from-content-identity)
  - [Atomicity belongs to explicit operations](#atomicity-belongs-to-explicit-operations)
  - [Concurrency checks stay where races are closed](#concurrency-checks-stay-where-races-are-closed)
  - [Failure must remain observable](#failure-must-remain-observable)
- [Architectural principles](#architectural-principles)
  - [Entities own decisions about their state](#entities-own-decisions-about-their-state)
  - [Workflows own cross-entity sequencing](#workflows-own-cross-entity-sequencing)
  - [Providers execute contracts](#providers-execute-contracts)
  - [Dependencies follow responsibility](#dependencies-follow-responsibility)
  - [Single responsibility is more important than uniform structure](#single-responsibility-is-more-important-than-uniform-structure)
- [Fast, predictable read paths](#fast-predictable-read-paths)
  - [Committed reads do not synchronize](#committed-reads-do-not-synchronize)
  - [Load only what the operation needs](#load-only-what-the-operation-needs)
  - [Admission work is not ordinary read work](#admission-work-is-not-ordinary-read-work)
  - [Caches have limited authority](#caches-have-limited-authority)
- [Validation at the right boundaries](#validation-at-the-right-boundaries)
- [Extension without unrelated rewrites](#extension-without-unrelated-rewrites)
- [The boundary with application domains](#the-boundary-with-application-domains)
- [Trust, safety, and capability boundaries](#trust-safety-and-capability-boundaries)
  - [Portable access comes first](#portable-access-comes-first)
  - [Verification is not a sandbox](#verification-is-not-a-sandbox)
  - [Secret authority is separately supplied](#secret-authority-is-separately-supplied)
  - [Protected content is not approved execution](#protected-content-is-not-approved-execution)
- [Assembly and resource ownership](#assembly-and-resource-ownership)
- [Determinism and compatibility](#determinism-and-compatibility)
- [Putting Artifactory into an application](#putting-artifactory-into-an-application)
- [A checklist for changes and extensions](#a-checklist-for-changes-and-extensions)

## Why this exists

A directory of files is not an application catalog.

Files can change outside the application. Documents can become invalid. A package can be replaced while someone is reading it. Local preferences must survive content updates. Credentials must not become part of portable documents. Built-in content must be repairable without allowing ordinary callers to rewrite it.

Applications often solve these problems independently for every feature. The result is repeated scanning, inconsistent identity rules, slow list operations, and fragile coordination between files and persistent metadata.

Artifactory provides one generic foundation for those responsibilities:

- Stable Artifact identity even when source content changes.
- Immutable, content-identified Definitions.
- Explicit synchronization from Sources into a committed catalog.
- Fast metadata reads that do not trigger ingestion.
- Verified material reads when freshness matters.
- Complete-package publication with explicit retry and recovery semantics.
- Local non-secret customization that is not overwritten by refresh.
- Secret bindings with durable recovery and cleanup.
- Protected application content with coordinated installation and reset.
- Clear ownership boundaries for extensions and alternative providers.

The goal is not to hide every operation behind one large service. It is to make responsibility, authority, and consistency visible.

## Who it is for

Artifactory is useful when an application needs to manage source-backed content rather than merely store arbitrary records.

Examples include:

- File-backed configuration and document catalogs.
- Application extensions and reusable content packages.
- Embedded built-in content alongside user-managed content.
- Workspaces that discover declarations from existing directories.
- Domain systems that need stable references to validated documents.
- Runtimes that must verify source material before using it.

The application may be LLM-related, but Artifactory itself is not LLM-specific.

## What it is not

Artifactory is not:

- A declaration language.
- A dependency solver or graph execution engine.
- An LLM runtime.
- A general-purpose package dependency manager.
- An authorization server.
- A filesystem sandbox.
- A transport or user-interface framework.
- A universal plugin registry.
- A promise that a mutable filesystem remains unchanged after a read.

It does not decide what a Model, Tool, Skill, Plugin, or other domain artifact means. It does not decide whether an admitted artifact is permitted to execute.

Those responsibilities belong to the consuming domain and application.

## The mental model

There are three different kinds of truth.

### 1. Source truth

This is the physical material: files, embedded resources, or content supplied by another Source driver.

A Source may change independently of the catalog.

### 2. Committed catalog truth

This is what Artifactory successfully observed, admitted, and published.

It is suitable for stable references, listings, inspection, and application management. It is not an implicit claim that external files have remained unchanged since publication.

### 3. Local application state

This is what users or the application choose locally: display preferences, enablement, non-secret overlays, and secret bindings.

It must not be overwritten by synchronization.

The main relationship is:

```text
Root
  └── Source
        └── observed declaration
              └── admitted Definition
                    └── source-backed Artifact
```

Additional state attaches locally:

```text
Artifact
  ├── local metadata and enablement
  ├── non-secret overlays
  └── secret bindings
```

A declaration observation is temporary synchronization evidence. It is not another persistent entity.

A catalog entry is a read projection. It is not another owner of Artifact state.

A managed package is a complete physical publication unit. It is not automatically a domain relationship or a dependency graph.

### A concrete example

Suppose a Source contains a declaration at `packages/example/document.json`.

After refresh:

- The Source records the relevant revision and observation evidence.
- A Definition captures the admitted immutable meaning.
- An Artifact provides a stable identity for that typed source origin.
- The catalog exposes lightweight metadata about the Artifact.

If the declaration changes, another Definition may be admitted and the same Artifact updated to point to it.

If the user disabled the Artifact locally, refresh does not re-enable it.

If the file disappears, the Artifact can become missing without immediately deleting its identity or local state.

If a runtime later needs the file, it uses verified resource access. It does not treat the earlier catalog listing as proof that the bytes are still current.

## The entities and what they own

Each entity owns a distinct responsibility. Ownership means deciding its invariants and lifecycle—not necessarily owning a physical backend.

| Entity         | Responsibility                                                                                                              |
| -------------- | --------------------------------------------------------------------------------------------------------------------------- |
| **Root**       | A namespace for Sources and Artifacts, its identity, lifecycle, protection, and retention policy.                           |
| **Source**     | A registered origin of physical content, normalized configuration, discovery configuration, revisions, and snapshot access. |
| **Definition** | Immutable admitted content, canonical representation, digest identity, and Root-local lookup guarantees.                    |
| **Artifact**   | Stable typed source-origin identity, source-derived availability, current Definition linkage, and local metadata.           |
| **Overlay**    | Revisioned non-secret customization attached to a protected Artifact or to the Store.                                       |
| **Secret**     | Binding metadata, trusted plaintext resolution, interrupted-write recovery, and physical-value cleanup.                     |

### Root: namespace, protection, and retention

A Root is a lifecycle and identity boundary. It does not inherently represent a feature, tenant, workspace, or built-in collection. Applications may assign those meanings.

Two policies are intentionally different:

- **Protected:** ordinary callers cannot mutate protected source-backed content.
- **Retained:** the Root itself cannot be retired or purged.

A Root may have either policy, both, or neither.

Trusted installation access may permit protected-content maintenance. It does not override retention.

Protection is not a substitute for application authorization. In-process privilege conventions are not transport credentials or an unforgeable security boundary.

### Source: physical origin and discovery

A Source describes where content comes from and how it can be observed.

It owns:

- Its identity and lifecycle.
- Root-local physical storage identity and reuse.
- Private normalized configuration.
- Public non-secret summaries.
- Discovery scopes and decoder selection hints.
- Revision advancement after relevant changes.
- Driver selection and snapshot access.

Source configuration and declaration discovery are different concerns. A storage driver understands its physical configuration; Artifactory understands how discovery is bounded and coordinated.

Sources may be registered for resource access without declaration discovery. Registering a Source does not require creating Artifacts.

### Definition: immutable admitted meaning

A Definition is admitted content, not a file location and not a runtime object.

Its identity is content-derived and scoped by a Root for lookup and availability. The same digest does not authorize access across Roots.

An existing Definition cannot be silently replaced with different content under the same immutable key.

Definition reads have explicit guarantees:

- Lookup keys are validated.
- Bulk results preserve requested order and cardinality, including repeated keys.
- Missing or unavailable Definitions produce an error rather than a silently shortened result.
- Returned mutable data has clear ownership.
- Caches cannot bypass Root or Definition availability.

Domain-specific typed reconstruction happens outside the generic Store.

### Artifact: stable identity across source changes

An Artifact identifies one typed declaration origin within a Source.

Its identity is not its display name, logical name, or current Definition digest. A single source entry may also emit multiple subresources.

Typed origin distinguishes:

```text
Root + Source + locator + subresource + artifact kind
```

This allows content at an existing physical origin to change kind without silently converting the old Artifact into another kind.

Artifact state distinguishes:

| State            | Meaning                                                                                      |
| ---------------- | -------------------------------------------------------------------------------------------- |
| **Available**    | The observed declaration has a current admitted Definition and source-content evidence.      |
| **Missing**      | The declaration is no longer available through the relevant Source or discovery scope.       |
| **Invalid**      | The observed declaration could not be admitted.                                              |
| **Incompatible** | The origin now emits a different kind from the existing Artifact's immutable typed identity. |

An incompatible Artifact may retain observed Definition evidence for diagnosis. That is not permission to use it as an available Artifact of the old kind.

Refresh owns source-derived fields. Consumers own local fields such as display name, enablement, and non-secret application data.

A refresh command must not contain local fields that it could overwrite accidentally.

Three questions remain separate:

1. Is the Artifact available?
2. Is it enabled?
3. Is the requested use authorized?

None implies the others.

### Overlay: non-secret local customization

Overlays let an application customize protected content without rewriting the source declaration.

The Store owns:

- Namespace registration and applicability.
- Generic payload bounds and object representation.
- Persistence and revisions.
- Protected-Root applicability.
- Atomic deletion behavior.

The feature owns the payload schema, meaning, and migration policy.

Store-scoped overlays cover non-secret local state that has no source-backed Artifact owner.

Overlays must not contain plaintext secrets.

### Secret: bindings are not physical values

A binding identifies which opaque secret value belongs to an Artifact namespace and slot.

The Store owns the binding lifecycle. A physical backend stores only:

```text
opaque reference → secret value
```

The backend does not interpret Artifacts, namespaces, slots, feature payloads, or cleanup policy.

Ordinary binding management exposes safe metadata and replacement or clearing operations. Plaintext resolution is a separate trusted capability.

That separation must exist in the actual implementations, not merely in the static interface through which a broad service happens to be passed.

## The workflows

Entities own state and policy. Workflows coordinate operations that cross those boundaries.

The workflows below describe observable behavior and consistency requirements, independently of a particular backend.

### 1. Register and prepare a Source

An application chooses a Root, a physical Source, and any discovery policy.

Artifactory then:

1. Validates the request and applicable authority.
2. Normalizes physical configuration through the selected driver.
3. Establishes or reuses the Root-local Source identity.
4. Coordinates physical provisioning when the driver requires it.
5. Persists the Source with explicit replay and conflict behavior.

Creation replay must distinguish equivalent intent from a conflicting request.

Provisioning compensation must account for ambiguous persistence outcomes. It must not destroy physical content merely because a caller did not receive a successful commit response.

Discovery preparation is an explicit mutation. It must not be hidden inside a catalog lookup or locator resolution.

### 2. Observe, admit, and publish

Refresh is the explicit transition from source truth to committed catalog truth.

It:

1. Reads the operational Source and relevant revision expectations.
2. Opens one Source snapshot.
3. Discovers candidates within configured bounds.
4. Invokes explicitly registered decoders.
5. Admits Definitions and collects observation evidence.
6. Derives Artifact creates and source-state changes.
7. Confirms and closes the snapshot.
8. Atomically publishes Definitions, Artifact transitions, and refresh state.

A failed publication must not expose half of a new catalog generation.

The observation records what was seen. It does not freeze an independently mutable filesystem after confirmation.

Source lifecycle changes also participate in publication. Disabling or retiring a Source, or removing its final discovery scope, must invalidate the corresponding source-backed state atomically—not through a later callback after the Source change has already committed.

### 3. Browse committed state

Ordinary catalog operations answer:

> What has the Store committed?

They do not answer:

> What would a fresh scan discover right now?

Listings should use lightweight metadata projections. Complete Artifact entities and immutable documents are loaded only when the caller needs them.

Optional document attachment should use bulk lookup rather than a separate read for every row.

A catalog read must not:

- Create or repair a Source.
- Expand discovery.
- Refresh content.
- Open filesystem snapshots.
- Publish a package.
- Recover pending writes.
- Perform durable cleanup.

This separation makes read latency predictable and prevents a management page from unexpectedly mutating the system.

### 4. Resolve and read verified material

When source freshness matters, the application uses verified resource access.

The verification chain is:

```text
Artifact
  → current Definition link
  → Source
  → committed observation evidence
  → current source material
```

Verification checks the evidence required by the operation, including relevant Source revision, generation, and content digests.

Portable entry and tree reads remain separate from trusted native-path acquisition.

A native path is a deployment detail, not portable artifact data. Returning a verified path does not authorize execution, establish a sandbox, or prevent another process from changing the filesystem afterward.

#### Multi-read verification sessions

A coherent operation may need several reads from the same Source.

A verification session:

- Reuses snapshots within the operation.
- Preserves Source revision and generation consistency.
- Confirms retained snapshots before successful batch results are accepted.
- Closes every retained snapshot deterministically.
- Gives nested callers a borrowed session rather than transferring ownership.
- Rejects a session belonging to a different resource service.

A callback producing a value is not sufficient for success. Session completion can still fail, and the caller must reject the batch in that case.

### 5. Publish or replace a complete managed package

The application prepares the package's meaning. Artifactory manages its publication.

Preparation must finish before expected bytes and digests are fixed. This includes family-specific filenames, serialization, line-ending normalization, and semantic validation.

Publication then:

1. Validates intent, authority, and the complete file set.
2. Resolves an eligible Source and the required physical capability.
3. Establishes revision and generation expectations.
4. Publishes complete physical package content.
5. Acknowledges the Source content change.
6. Refreshes when necessary.
7. Verifies the expected Artifact identity and Definition digest.

Equivalent create replay is idempotent.

Replacing different content at an existing address requires replacement authorization and the required generation expectation.

**Physical publication and catalog publication are different commit boundaries.**

A successful physical write followed by a metadata conflict is an explicit recoverable outcome. It must not be reported as if nothing changed, nor hidden behind an unconditional retry that could overwrite unrelated work.

Provider capability determines package support. Business workflows must not infer support from a hard-coded Source-kind string.

### 6. Remove content and converge on a consistent catalog

Package removal:

1. Validates ownership, authority, and expectations.
2. Removes the physical package.
3. Acknowledges the Source change.
4. Prunes explicitly owned discovery state when requested and permitted.
5. Refreshes or publishes the necessary invalidation.
6. Verifies the expected missing state when required.

Retries must converge after interrupted work without ignoring unrelated generation conflicts.

Removing the final discovery scope still has lifecycle consequences. An empty discovery configuration cannot leave earlier source-backed Artifacts incorrectly available.

Removing content, making an Artifact missing, cleaning its attached local state, and purging the Artifact are distinct operations.

### 7. Customize locally and manage secrets

Local changes do not rewrite immutable Definitions.

Non-secret overlays use explicit revision expectations. Universal Artifact enablement remains Artifact state; features must not introduce a competing enablement flag in an overlay.

Secret replacement follows a staged lifecycle:

1. Validate the binding request and expected revisions.
2. Record pending publication.
3. Write a new physical value under a new opaque reference.
4. Atomically publish the binding change and retire the previous reference into durable cleanup.
5. Attempt physical cleanup independently.

An interrupted write remains recoverable. A failed replacement must not silently discard the previously active binding.

Clearing a binding detaches it logically and durably queues cleanup.

Plaintext resolution requires a separately supplied trusted capability and an expected opaque reference. It verifies the physical value against the binding's integrity metadata.

Physical deletion failures remain recorded for retry. Cleanup may complete only when no active binding references the value.

#### Atomic overlay and Artifact-local cleanup

Deleting an Artifact/namespace overlay must atomically:

- Remove the overlay.
- Detach bindings in that same namespace.
- Durably enqueue their physical values for cleanup.

Artifact-wide local cleanup similarly removes attached overlays and bindings in one logical transaction. It does not delete the Artifact itself.

A best-effort physical cleanup pass may follow a successful logical commit. Its failure does not undo that commit.

### 8. Install and reconcile application-owned content

Applications declare which protected Roots, Sources, and packages should exist.

Artifactory coordinates:

- Validation of the complete applicable installation plan.
- Protected topology creation or reconciliation.
- Aggregate and package-level hydration state.
- Shared-Root reset.
- Package publication and verification.
- Successful-installation marker commits.

A reset affecting a shared Root invalidates every participating installer that uses it.

The previous aggregate hydration marker survives reset until replacement installation succeeds. This provides a durable basis for recovery after interruption.

Package batches are not globally atomic. Their publication units and failure boundaries remain explicit. Work for a shared Source must advance Source state and refresh coherently rather than pretending independent per-package refresh loops form one batch.

Reset must queue secret cleanup before deleting metadata that references those values.

Installation and recovery must run before ordinary writers are exposed, or participate in an explicit concurrency protocol that prevents races with them.

#### Generated installation evidence

Generated packages can carry previously admitted declaration evidence to avoid repeating parsing at runtime.

The trusted path remains constrained:

- Build-time compilation uses the ordinary decoding and admission path.
- Reproducibility checks establish equivalence with that path.
- Runtime checks source-content witnesses.
- Installation translates package-relative evidence into Source-relative evidence.
- Hydration markers are committed only after required publication and verification succeeds.

There is no ordinary request flag that turns user-supplied content into trusted compiled evidence.

## Consistency and identity rules

### Stable identity is separate from content identity

Different identifiers answer different questions:

| Identity or evidence                | Question it answers                                       |
| ----------------------------------- | --------------------------------------------------------- |
| Root identity                       | Which namespace owns this state?                          |
| Source identity and storage key     | Which registered physical origin is this?                 |
| Artifact identity                   | Which stable typed source-backed record is this?          |
| Logical name and version            | Which domain-facing declaration identity is described?    |
| Definition digest                   | Which immutable admitted Definition is this?              |
| Source-content digest               | Which physical declaration bytes were observed?           |
| Source generation                   | Which Source view did the operation observe?              |
| Package or registration fingerprint | Which inventory or interpretation configuration was used? |

These values are not interchangeable.

A canonical source document and a canonical Definition may legitimately have different digests.

A logical name is not automatically a unique storage address. A Definition digest is not an Artifact ID.

### Atomicity belongs to explicit operations

The entity or workflow decides the requested transition.

The persistence provider verifies that the transition is still applicable and commits it correctly.

Atomic operations include:

- Definition admission, Artifact synchronization, and refresh-state publication.
- Source lifecycle changes with their required catalog invalidation.
- Overlay deletion with binding detachment and cleanup enqueueing.
- Binding replacement or clearing with durable cleanup bookkeeping.
- Artifact-local overlay and binding cleanup.

A provider may enforce revisions, uniqueness, references, liveness, and immutable-key conflicts. Those safeguards are necessary.

It must not invent lifecycle policy because several tables happen to live in one database.

### Concurrency checks stay where races are closed

Preflight checks improve errors and avoid unnecessary work. They do not replace commit-time checks.

Aggregate publication must cover all relevant concurrent changes. Checking only a Source revision is insufficient when another refresh can create or update Artifacts without advancing that Source revision.

Resource confirmation is also time-dependent. It cannot be replaced with validation performed during registration or an earlier read.

### Failure must remain observable

The system distinguishes:

- Invalid input.
- Unsupported capabilities or interpretation.
- Missing or unavailable state.
- Stale expectations and concurrency conflicts.
- Integrity mismatches.
- Protected operations.
- Closed resources.

Retryable workflows must preserve enough durable evidence to resume safely.

An error after a physical mutation must not be disguised as an operation that had no effect.

## Architectural principles

### Entities own decisions about their state

An entity owns its identity, invariants, direct operations, and applicable persistence contracts.

Its public boundary should make it possible to answer:

- What state does it own?
- What may it change?
- Which decisions does it make?
- What input does it require?
- What must commit atomically?
- Which operations require trusted access?

There is no requirement for every entity to have the same collection of interfaces, factories, or supporting types.

Construction establishes the entity's actual responsibility. It is not another architectural layer.

### Workflows own cross-entity sequencing

A workflow owns ordering, cross-entity preconditions, compensation, retries, and aggregate publication intent.

It does not take ownership of another entity's implementation merely because it coordinates that entity.

There should be one owner for generic refresh, resource verification, package publication, installation, and Artifact-local cleanup behavior—not separate copies in every feature.

### Providers execute contracts

Providers own mechanisms such as:

- Transactions and physical persistence.
- Filesystem or embedded-resource access.
- Schema compilation and execution.
- Encryption and secret-backend integration.
- Provider-specific configuration and resource management.
- Provider-local immutable payload caching.

Providers do not decide:

- Which declaration language is recognized.
- What a Root means to the product.
- Which artifact family a package belongs to.
- Which graph relationships are legal.
- Whether an artifact is permitted to execute.
- Which feature overlay schema applies.
- Whether source changes should produce missing or incompatible Artifacts.

Those are entity, workflow, domain, or application decisions.

### Dependencies follow responsibility

The intended direction is:

```text
shared values and policies
        ↑
entity responsibilities
        ↑
cross-entity workflows
        ↑
application and deployment assembly
```

Providers implement contracts supplied by those responsibilities.

In particular:

- Protection policy does not depend on the installation workflow.
- Ingestion does not depend on installation package-plan representations.
- Resource verification does not import a Source service's private implementation.
- Artifact synchronization consumes explicit observations, not a scanner engine.
- Generic business behavior does not select concrete persistence or filesystem providers.
- Models describe values; they do not conceal service ownership.

Private implementation may live directly inside its owner or behind an owner-local private boundary. Separate composition packages and forwarding aliases are not required for each entity.

### Single responsibility is more important than uniform structure

A smaller public surface means fewer capabilities, not merely fewer files.

Avoid:

- Broad services hidden behind narrow interfaces while retaining privileged methods.
- Universal descriptors that mix drivers, schemas, decoders, and runtime services.
- Generic “state,” “metadata,” or “engine” owners that absorb unrelated responsibilities.
- Constructor-only bridge layers.
- Compatibility aliases that preserve obsolete ownership.
- Helper packages whose names do not reveal the responsibility they serve.

Shared behavior should move to its actual owner. Similar-looking code is not enough reason to combine unrelated policies.

## Fast, predictable read paths

Read performance is an architectural constraint.

### Committed reads do not synchronize

Catalog and immutable lookup operations do not secretly ingest, refresh, repair, publish, or clean up state.

Applications choose explicitly when to refresh and when verified source access is required.

### Load only what the operation needs

- Use catalog projections for listing.
- Load complete Artifacts only when their full state is required.
- Attach Definition documents explicitly.
- Prefer bulk document access over per-row lookups.
- Avoid repeated reconstruction of immutable data.
- Avoid copies that do not establish a new ownership boundary.

### Admission work is not ordinary read work

Immutable reads should not repeatedly:

- Execute the same schema.
- Canonicalize the same admitted document.
- Recalculate the same Definition digest.
- Normalize unchanged configuration.
- Revalidate constructor dependencies.

Necessary relationship, liveness, authorization, and consistency checks remain.

A fast read is not an unchecked read. It is a read that performs only the work justified by its boundary.

### Caches have limited authority

An immutable payload cache may avoid decoding persisted content again.

A domain-owned typed cache may avoid reconstructing a typed declaration again.

Neither cache proves:

- That an Artifact still points to that Definition.
- That the Root or Definition remains available.
- That source bytes are current.
- That the Artifact is enabled.
- That the requested use is authorized.

Root purge or reset must not leave stale cached availability observable after recreation.

Typed cache identity must account for Root-local Definition identity and any interpretation revision that can change during the cache's lifetime.

Transparent private caching is compatible with read-only behavior. Durable repair or mutation disguised as a cache fill is not.

## Validation at the right boundaries

Validation is neither “repeat everything everywhere” nor “validate at startup and trust all future state.”

| Boundary                      | Responsibility                                                                                          |
| ----------------------------- | ------------------------------------------------------------------------------------------------------- |
| Construction                  | Required dependencies, valid configuration, namespace policies, and mandatory capability relationships. |
| Registration                  | Valid identifiers and revisions, duplicate detection, schema bindings, and dispatch consistency.        |
| Public operation entry        | Request structure, identity, bounds, expectations, context, and applicable authority.                   |
| Driver configuration          | Physical configuration validation and normalization.                                                    |
| Decoder and schema output     | Output shape, canonical representation, identity linkage, digest evidence, and diagnostics.             |
| Persistence commit            | Current revisions, liveness, uniqueness, references, and immutable-key conflicts.                       |
| Verified operation completion | Snapshot confirmation and time-dependent Source consistency.                                            |

After successful construction, private methods may assume mandatory dependencies exist.

After request validation, private helpers should receive owned normalized values or an operation plan—not repeatedly reinterpret the same raw request.

Expensive admission steps have one owner:

- Source reading and ingestion establish source-content evidence.
- The schema catalog executes the selected schema.
- Definition admission establishes canonical Definition identity.
- Artifact synchronization derives source-owned Artifact transitions.

If a codec changes the document, validating that changed output is necessary. It is not a duplicate check of the original bytes.

Mutable values must be copied, kept in private immutable representations, or borrowed under an explicit lifetime contract. “Already validated” is meaningless if another caller can mutate the value afterward.

Runtime closure is separate from initialization validity. Removing redundant dependency checks must not remove closed-state behavior.

## Extension without unrelated rewrites

Artifactory is extended through specific contracts.

| Extension                    | What the extension supplies                                                               |
| ---------------------------- | ----------------------------------------------------------------------------------------- |
| A new Source                 | Configuration normalization and bounded snapshot behavior.                                |
| A writable package Source    | Complete-package publication and removal capabilities, with batch support where required. |
| A new inbound format         | Recognition and decoding from bounded candidate bytes and supplied sibling readers.       |
| A new schema catalog         | Expected-key canonicalization and schema execution guarantees.                            |
| A new artifact family        | Schemas, codecs, decoders, and domain-owned interpretation outside the Store.             |
| A new persistence provider   | The named entity and workflow persistence contracts, including atomicity guarantees.      |
| A new secret backend         | Opaque-reference physical value storage.                                                  |
| A new protected installation | Application-owned topology, content inventory, and optional domain lifecycle work.        |

A decoder receives only the inputs required to decode. It must not open native paths, inspect private Source configuration, generate persisted Artifact IDs, publish packages, or mutate the catalog.

A source-aware decoder may read bounded sibling entries. If it emits a declaration from another physical entry, its origin locator and content digest must identify that actual entry.

The generic schema boundary requires an expected schema key. Declaration dispatch—such as interpreting `type`, `apiVersion`, or legacy headers—belongs to the declaration contract.

Multiple complete schema keys may coexist even when a particular declaration language cannot distinguish them. That ambiguity belongs to that language's dispatcher.

Adding a family must not require editing generic Store business logic. Adding a storage provider must not change declaration semantics.

## The boundary with application domains

Artifactory owns generic storage and lifecycle. Domains own meaning.

Domains remain responsible for:

- Declaration vocabulary and semantic validation.
- Header dispatch and source-format adaptation.
- Typed reconstruction from admitted Definitions.
- Logical-name, selector, and version interpretation.
- Dependency graphs, cycles, ambiguity, and graph limits.
- Runtime capability planning.
- Package filenames and text normalization.
- Feature overlay schemas and legal secret slots.
- Built-in product content.
- Execution approval and sandbox policy.

Domain resolution consumes committed or explicitly verified state. It does not silently prepare Sources or publish content.

A Plugin declaration containing membership is still domain content. It does not create a generic Store ownership relationship merely because its members refer to Artifacts.

Likewise, domain graph planning must not automatically fetch plaintext credentials or execute runtime capabilities.

## Trust, safety, and capability boundaries

### Portable access comes first

Ordinary consumers receive portable summaries, metadata, and verified material.

Private Source configuration and native filesystem paths require explicit operational or trusted capabilities.

### Verification is not a sandbox

Portable path validation prevents malformed logical locators. It does not, by itself, prevent filesystem escape through symlinks or establish an operating-system security boundary.

Each Source driver must define traversal, symlink, and snapshot behavior. Applications must choose drivers and runtime isolation appropriate to their trust model.

Reads, scans, and package materialization remain bounded by byte, entry, and traversal limits.

### Secret authority is separately supplied

Binding management must not expose an object that can be asserted into a plaintext-reading service.

Plaintext must never enter:

- Definitions.
- Artifact local data.
- Overlay payloads.
- Generated catalogs.
- Ordinary response objects.
- Diagnostics or logs.

A trusted runtime receives plaintext access only when application composition explicitly supplies it.

### Protected content is not approved execution

Application-installed content may be protected from ordinary mutation while still requiring runtime approval.

A request field cannot create installer privilege. A trusted in-process convention must not be mistaken for transport authorization.

## Assembly and resource ownership

Assembly connects responsibilities; it does not implement their behavior.

There is one coherent Store capability surface. A deployment should not reproduce it through several mirrored handles.

A deployment chooses physical providers, opens resources, supplies application policy and registrations, and arranges shutdown.

The ownership rules are:

1. Business services borrow provider resources.
2. Every provider resource has exactly one lifecycle owner.
3. Caller-supplied resources remain caller-owned when opening fails.
4. Ownership transfers on success only when the opening contract explicitly says so.
5. Locally opened resources are closed on every failed-open path.
6. Successful shutdown closes owned resources once, in a defined order.
7. Consumers and background work stop before borrowed provider resources are closed.
8. Secret services do not independently close a backend owned by deployment.
9. Repositories are not inspected for accidental close support.

Startup recovery and protected installation must complete under the application's concurrency protocol before ordinary writers begin.

Resource lifetime and business capability are different concerns. Receiving permission to read an entity does not make that service the owner of the database or backend.

## Determinism and compatibility

Architectural changes must not silently alter persisted or generated identity.

Changes to package organization, constructors, or internal ownership must preserve the relevant contracts:

- Root and Source identities.
- Artifact typed-origin identity.
- Source-kind strings.
- Storage keys and physical layout.
- Managed package addressing.
- Secret-reference format.
- Definition digest calculation.
- Decoder and schema fingerprint representation.
- Compiled package fingerprints.
- Generated payload representation.
- Deterministic ordering.
- Nil, empty, and omitted representations where they affect canonical identity.
- Publication replay and retry behavior.

A hash is meaningful only within its documented domain. Schema fingerprints, source digests, Definition digests, and package fingerprints must not be substituted for one another.

Once package expectations are established, publication must use those exact bytes. A lower layer must not silently reformat or normalize them.

Storage or generated-format changes require an explicit decision and migration strategy, not an incidental consequence of a refactor.

## Putting Artifactory into an application

A typical integration follows this sequence:

1. **Define ownership.** Decide which Roots are mutable, protected, or retained.
2. **Choose physical providers.** Select persistence, Source drivers, schema execution, and any secret backend.
3. **Register interpretation.** Supply explicit schemas, codecs, and decoders for the content your application understands.
4. **Assemble capabilities.** Give ordinary services only ordinary access; supply plaintext and native-path capabilities separately to trusted runtimes.
5. **Complete startup work.** Recover interrupted operations and reconcile protected content before ordinary writers begin.
6. **Register and prepare Sources explicitly.**
7. **Refresh when synchronization is intended.**
8. **Use the committed catalog for browsing and management.**
9. **Use verified resource access when current source material matters.**
10. **Publish complete prepared packages through the package workflow.**
11. **Keep local customization and secrets outside portable Definitions.**
12. **Stop consumers and close the owning Store handle during shutdown.**

The most important choice at each call site is not which backend to use. It is which responsibility the operation belongs to and which consistency guarantee it needs.

## A checklist for changes and extensions

Before adding behavior, ask:

- Which entity owns the state and the decision?
- Is this a direct entity operation or a cross-entity workflow?
- Is the provider executing an explicit transition or inventing policy?
- Is the operation a committed read, a verified read, or a mutation?
- Could this work unexpectedly make a read slow or stateful?
- Which checks belong at initialization, admission, commit, or completion?
- Are mutable values owned clearly?
- Does a cache bypass a current availability or authority check?
- Are ordinary and trusted capabilities genuinely separate?
- Is a physical mutation being confused with a metadata transaction?
- Can interrupted work be retried without overwriting unrelated changes?
- Does a new abstraction name a real responsibility?
- Can the extension be added without changing unrelated business services?
- Have persisted and generated identities remained unchanged?

Artifactory's architecture is successful when those answers are clear from the owning responsibility and its contract—not from understanding the entire application.
