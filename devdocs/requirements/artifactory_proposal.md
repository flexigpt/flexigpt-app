# Proposal 1 — Artifactory: source-backed artifact storage and lifecycle

## 1. Purpose

Artifactory is the generic system responsible for:

- Registering and managing Roots and Sources.
- Reading bounded, generation-consistent Source snapshots.
- Decoding Source content through explicitly registered decoders.
- Admitting immutable Definitions.
- Maintaining source-backed Artifact identity and lifecycle.
- Providing a committed Artifact catalog.
- Resolving and reading verified source material.
- Publishing and removing complete managed packages.
- Persisting non-secret local overlays.
- Managing secret bindings and physical-secret lifecycle.
- Installing and reconciling application-declared protected topology.

It is **not** a declaration-language framework, an LLM runtime, or a graph resolver.

Its central relationship is:

Root → Source → observed declaration → admitted Definition → source-backed Artifact

Its additional local state is:

Artifact → local metadata, overlays, and secret bindings

Its operational workflows coordinate those relationships without transferring ownership of the entities involved.

### The architectural objective

A developer should be able to answer all of the following from the owning package and its contracts:

- What state does this responsibility own?
- What decisions does it make?
- What information does it require?
- What may it mutate?
- What must be committed atomically?
- What does the concrete provider merely execute?
- Which capabilities are ordinary consumer access, and which are trusted operational access?

No answer should require understanding a generic `engine`, `localstate`, `metadata`, `kit`, or entity `compose` abstraction.

---

## 2. Assessment of the current baseline

The attached code already establishes several correct boundaries.

### Already in place

| Area                         | Current progress                                                                                 |
| ---------------------------- | ------------------------------------------------------------------------------------------------ |
| Public ownership             | Most APIs and models are entity- or flow-owned.                                                  |
| Provider namespace           | Concrete implementations are under singular `provider`.                                          |
| Source kinds                 | Concrete kind constants belong to `fsdir`, `iofs`, and `managedfs`.                              |
| Source driver contracts      | Driver, Snapshot, bootstrap, and native-path contracts have Source ownership.                    |
| Managed package values       | Physical package addressing, files, and publication values belong to `source/managedpackage`.    |
| Managed publication workflow | Publication/removal orchestration now lives in `flow/managepackage`.                             |
| Installation workflow        | Topology and hydration orchestration now lives in `flow/install`.                                |
| Refresh values               | Refresh state and inspection belong to `flow/refresh/model`.                                     |
| Registration                 | Schema codecs and decoders are explicit registrations rather than a generic Provider descriptor. |
| SQLite dependencies          | SQLite consumes public entity/flow contracts rather than implementation-owned command types.     |
| Package file reading         | Generic `fs.FS` package reading belongs to `source/managedpackage`.                              |

Keep these improvements. Do not repeat their migration merely to match the spelling of an older directory sketch.

In particular, retain the current distinction:

- `source/managedpackage`: physical managed-package concepts.
- `flow/managepackage`: the workflow that manages publication and removal.

### Still incomplete

| Current condition                                                                         | Required end state                                                                                               |
| ----------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------- |
| `*/impl/compat.go` re-exports private implementation                                      | Removed, without replacement aliases.                                                                            |
| Per-flow `compose` packages forward to private constructors                               | Construction belongs to the named flow itself; these are not templates for more bridge packages.                 |
| `secret/internal` owns Overlay, Secret, and ArtifactCleanup behavior                      | Three actual responsibility owners and implementations.                                                          |
| `sqlite.LocalStateRepository` implements unrelated contracts through one public aggregate | Named repository implementations for Overlay, Secret, and ArtifactCleanup.                                       |
| Artifact repository embeds catalog persistence                                            | Artifact persistence and catalog queries are independently supplied contracts.                                   |
| Catalog repository includes `GetMany` of complete Artifacts                               | Complete Artifact loading belongs to Artifact persistence.                                                       |
| Definition reads are exposed directly from the SQLite repository                          | Definition owns the read boundary and its guarantees; provider-specific objects are not the application surface. |
| Ingest observations/results are reachable through `ingest/impl`                           | Explicit staging values under Ingest, with private scanning implementation.                                      |
| Resource imports Source implementation helpers                                            | Source-owned snapshot operations are available through their actual responsibility contracts/helpers.            |
| Ingest caches `install/model.CompiledDocument` values                                     | Ingest owns admitted-document evidence; Install translates installation payloads into that evidence.             |
| Root policy imports the Install flow for privilege interpretation                         | Root protection is a lower-level entity policy, not dependent on a coordinating flow.                            |
| SQLite decides source-disable/retirement consequences                                     | Entity/flow policy constructs explicit transitions; SQLite applies and guards them atomically.                   |
| JSON Schema provider interprets `type` / `apiVersion`                                     | Declaration dispatch belongs to LLM Artifactory.                                                                 |
| `store/spec` contains deployment and provider constants                                   | Those constants move to their actual deployment/provider owners.                                                 |
| Narrow interfaces expose a broad concrete secret service                                  | Actual capability separation, not merely static interface narrowing.                                             |

---

## 3. Requirements

### 3.1 Functional requirements

The following must continue to work:

1. Root creation, replay, update, retirement, and purge.
2. Protected and retained Root policies.
3. Source creation, ensure/reuse, configuration normalization, bootstrap compensation, update, retirement, discard, and purge.
4. Explicit and directory-based Source discovery.
5. Ordinary and source-aware decoders.
6. Refresh publication with immutable Definition admission and Artifact synchronization.
7. Metadata-only catalog queries and optional bulk document attachment.
8. Verified Artifact resolution and Source entry/tree reads.
9. Shared resource-verification sessions.
10. Managed package create, replace, remove, retry, and discovery pruning.
11. Protected topology installation and reset.
12. Aggregate and package-level hydration markers.
13. Generated compiled package registration and hydration.
14. Protected Artifact overlays and store-scoped overlays.
15. Secret replacement, clearing, plaintext resolution, pending recovery, and garbage draining.
16. Atomic cleanup of state attached to an Artifact.
17. Correct startup failure cleanup and deterministic shutdown.

### 3.2 Extensibility requirements

The following extensions must not require modifying unrelated business services:

- A new Source driver.
- A new managed-package-capable Source driver.
- A new inbound document decoder.
- A new schema catalog implementation.
- A new artifact-family schema codec.
- A new persistence provider implementing the required repository contracts.
- A new physical secret-value backend.
- A new application-owned protected topology declaration.
- A new artifact family supplied by LLM Artifactory or another consumer.

Extensibility must come from **specific contracts**, not from a universal plugin descriptor, capability bag, or generic registry.

---

## 4. Non-negotiable architecture rules

## 4.1 Entities own state and lifecycle policy

The domain entities remain:

- Root
- Source
- Artifact
- Definition
- Overlay
- Secret

Each owns its identity, invariants, direct operations, and applicable persistence contracts.

A package does not need an identical checklist of `API`, `Repository`, `Runtime`, `Component`, and `Factory` types. Add only the contracts justified by that responsibility.

## 4.2 Flows own cross-entity sequencing

The named flows remain:

- Refresh
- Resource
- ManagePackage
- Install
- ArtifactCleanup

A flow owns:

- Operation ordering.
- Cross-entity preconditions.
- Retry and compensation semantics.
- Construction of aggregate publication commands.
- Coordination of capabilities received from the entity owners.

A flow does not acquire ownership of another entity’s implementation merely because it coordinates that entity.

## 4.3 Providers execute contracts; they do not choose domain policy

Concrete providers own:

- SQL and transaction mechanics.
- Filesystem and `fs.FS` operations.
- Schema compilation and execution.
- Encryption/backend integration.
- Provider-specific configuration.
- Provider-local caching and resource management.

They must not decide:

- Which declaration language is recognized.
- Whether an Artifact should become missing or incompatible.
- Which feature a Root represents.
- Which package belongs to an LLM family.
- Which runtime capability a declaration is allowed to execute.
- Which namespace payload schema a feature uses.

Storage enforcement is still required. Revision checks, uniqueness, foreign keys, liveness checks, and immutable-key conflict checks are not forbidden business logic.

The distinction is:

> The entity or flow decides the requested transition.
> The provider verifies that the transition is still applicable and executes it correctly.

## 4.4 No implementation compatibility façades

Remove:

- `*/impl` packages.
- Aliases that export private services or private engine values.
- Constructor variables that simply re-export private constructors.
- Temporary `LocalState` aggregates under a different package name.
- “Component” objects that merely bundle unrelated views of one service.

A migration is not complete when `impl` has been renamed to `compose`.

## 4.5 Construction belongs to the owner

Root construction belongs to Root. Source driver registration belongs to Source. Ingest scanner construction belongs to Ingest. Installation construction belongs to Install.

The construction entry should establish an actual named responsibility, not return a universal component bundle.

### Go package rule

The original requirement is refined to:

> Implementation must remain private and inside its entity or flow directory. A separate `internal` Go package is not mandatory for every service.

For straightforward services, use an unexported implementation in the owning Go package. Public constructors can then establish that service without a forwarding package or an import cycle.

Use an owner-local `internal` subpackage when there is a real private implementation boundary worth enforcing.

Do not introduce aliases or a composition bridge merely to preserve a mechanical `internal/service.go` placement.

This deliberately replaces the earlier literal requirement that every entity implementation be accessed through `<entity>/compose`.

## 4.6 Dependency direction must remain intelligible

The intended directions are:

- Models and specifications do not depend on services or providers.
- Entity behavior depends on its own values, narrowly required contracts, and lower-level entity policies.
- Flows depend on entity capabilities and values.
- Providers depend on the contracts they implement.
- Deployment assembly depends on providers and public construction entries.

In particular:

- Root must not depend on Install to understand Root protection.
- Source/Ingest must not depend on Install’s package plan representation.
- Resource must not depend on Source’s private service.
- Artifact synchronization must not depend on Ingest’s private engine.
- Generic store assembly must not import SQLite, filesystem providers, or the concrete JSON Schema provider.

## 4.7 Minimal public surface means minimal capabilities, not merely fewer files

Export a capability only when it is needed by:

- An ordinary consumer.
- A concrete provider.
- A named cross-entity flow.
- Deployment assembly.

Do not export a service’s entire method set because one flow needs two methods.

Do not retain public APIs solely because an old compatibility package exposed them.

Go-exported persistence values are not automatically transport-facing models. Their purpose must nevertheless be clear from their owner and documentation.

---

## 5. Responsibility ownership

## 5.1 Shared specification — `store/spec`

Own only genuinely shared store vocabulary:

- Store error categories.
- Portable identifiers and locators.
- Logical names and versions.
- Storage keys.
- Shared size/traversal safety limits.
- Portable path validation and matching.
- Shared diagnostics.
- Digest conventions where a store-wide convention is actually required.

It must not own:

- Local database filenames.
- Manifest filenames and deployment layout versions.
- Managed filesystem staging prefixes.
- Default VCS/node_modules traversal exclusions.
- Operating-system keyring identities.
- Application-wide directory modes.

The presence of a constant in several files does not make it a shared domain specification.

Existing lower-level packages such as `cryptoutil`, `jsonutil`, `clockutil`, and `uuidutil` should be reused where appropriate. Do not copy them into Artifactory or wrap them under a new utility namespace just to make the directory tree self-contained.

---

## 5.2 Root — `store/root`

Own:

- Root identity and metadata.
- Create/replay semantics.
- Retirement and purge eligibility.
- Protected Root policy.
- Retained Root policy.
- Interpretation of trusted protected-root mutation access.

Protected and retained remain different:

- **Protected:** ordinary mutation of protected source-backed content is restricted.
- **Retained:** retirement and purge of the Root itself are prohibited.
- Installer access does not bypass retention.

Move policy contracts out of the model package when restructuring the owner. They are behavioral policy, not persisted Root data.

Root protection must not import `flow/install`.

Install is a user of protected-root access, not its foundational owner.

Any context-carried installer grant is a trusted in-process convention—not a transport authorization mechanism or an unforgeable security boundary. Request flags such as `AllowProtected` must never grant authority by themselves.

---

## 5.3 Source — `store/source`

Own:

- Source identity and lifecycle.
- Root-local storage-key reuse.
- Private normalized configuration.
- Public Summary projection.
- Store-owned discovery configuration.
- Source revision advancement.
- Driver registration and dispatch.
- Physical bootstrap coordination.
- Source snapshot access.
- Acknowledgment of successful source-content mutation.

Keep separate:

1. **Ordinary Source operations:** return Summary values.
2. **Operational Source access:** supplies normalized Source state and snapshots to store workflows.
3. **Content-revision acknowledgment:** advances Source metadata after physical mutation.
4. **Native-path access:** explicitly trusted and optional.

Do not put all four behind an ordinary Source API instance with an accidentally broad dynamic method set.

Source owns the meaning of “disabled,” “retired,” and “discovery removed.” The cross-entity publication consequences of those transitions are addressed under Refresh below.

### Source driver contracts — `store/source/driver`

Own:

- Driver configuration normalization.
- Snapshot opening.
- Snapshot operations and lifetime contract.
- Optional physical bootstrap and managed-root removal contracts.
- Optional native-path capability.
- The bounded read primitive for a supplied snapshot entry.

Driver implementations remain in `provider/*`.

The driver contract must specify traversal and symlink behavior rather than implying that lexical locator validation alone provides a filesystem sandbox.

### Source ingest — `store/source/ingest`

Own:

- Decoder contracts.
- Decoder registration.
- Schema-catalog binding.
- Decoder/schema fingerprinting.
- Candidate selection.
- Recognition and decoder selection.
- Source-aware sibling reads.
- Decoded-value admission into transient observations.
- Bounded scan execution.
- Trusted compiled-document evidence used to bypass repeated parsing.

The scanner remains private implementation. Other responsibilities receive a narrow scanning capability and explicit output values.

Transient observations and scan results must no longer be exported through `ingest/impl`.

They are Ingest-owned data, not a new persistent entity and not a catalog projection.

### Managed package concepts — `store/source/managedpackage`

Own:

- Semantic package address shape.
- Package-relative file values.
- Package file-set validation and normalization.
- Physical publication/removal contracts.
- Physical batch-write contract.
- Generic package reading from `fs.FS`.

It does not own:

- Artifact-family filenames.
- LLM declaration parsing.
- Feature-specific package layouts.
- Line-ending normalization policy.
- The sequence “write package → advance Source → refresh → verify Artifact.”

That sequence belongs to `flow/managepackage`.

---

## 5.4 Artifact — `store/artifact`

Own:

- Artifact identity allocation.
- Immutable typed source-origin identity.
- Source-derived Artifact state.
- Local display name, enablement, and Data.
- Artifact relationship reads, including resolving its current Definition link.
- Derivation of creates and source-state updates from admitted observations.
- Missing, invalid, incompatible, and available state rules.
- Purge eligibility.

The following distinction must remain:

- Source refresh owns source-derived fields.
- Consumers own local Artifact fields.
- Source-state commands must not contain local fields that refresh could overwrite.

Artifact synchronization is an Artifact responsibility. It receives explicit observations and current Artifact values; it must not require an Ingest engine or provider object.

Artifact ID allocation belongs to Artifact. The UUID mechanism can remain in the existing lower-level UUID utility.

Do not preserve an extra ID-provider namespace merely as a naming wrapper if the contract and default implementation can live clearly with Artifact.

### Artifact catalog — `store/artifact/catalog`

Own:

- Committed lightweight read projections.
- Query filtering.
- Metadata-only listing.
- Optional bulk attachment of immutable Definition documents.

Catalog reads must not silently refresh Sources or open filesystem snapshots.

Separate catalog persistence from Artifact persistence.

Specifically:

- `GetMany` of complete Artifact entities belongs to Artifact.
- Catalog repositories return catalog rows.
- The Artifact repository must not embed the entire catalog repository solely because SQLite currently implements both.
- A concrete provider may implement both through its own named repository objects.

The catalog service must not expose local Artifact mutations through its concrete method set.

---

## 5.5 Definition — `store/definition`

Own:

- Definition identity within a Root.
- Canonical Definition representation.
- Definition digest calculation and verification.
- Immutable admission rules.
- Definition lookup and bulk lookup guarantees.
- Definition-body encoding/decoding where the behavior is generic.

A Definition is not an Artifact address and not a typed runtime object.

Definition selectors may remain generic values. Their semantic interpretation, version solving, and graph expansion belong outside Artifactory.

The read boundary must specify:

- Validation of lookup keys.
- Ordering and cardinality for bulk reads.
- Behavior for missing Definitions.
- Ownership of returned mutable data.
- Required liveness/access checks.

Do not expose a concrete SQLite repository as the consumer object merely because its method signatures happen to satisfy `definition.API`.

The entity implementation should own the actual read guarantees. It should not be a forwarding layer with no responsibility.

### Definition schema — `store/definition/schema`

Own:

- Schema keys.
- Parsed canonical document values.
- Codec contract.
- Expected-key canonicalization contract.
- Setup-time catalog access.
- Per-store catalog construction contract, where required by provider-independent assembly.
- Generic passthrough codec behavior.

The generic schema boundary accepts an **expected schema key**.

It must not infer a key from:

- `type`.
- `apiVersion`.
- Legacy declaration headers.
- Artifact-family-specific JSON fields.

The concrete JSON Schema provider validates that expected key’s schema and checks codec output integrity.

LLM declaration dispatch belongs to LLM Artifactory.

---

## 5.6 Overlay — `store/overlay`

Own:

- Protected Artifact overlay operations.
- Store-scoped overlay operations.
- Namespace registration/policy for those operations.
- Generic payload size and canonical-object requirements.
- Overlay revision semantics.
- Protected-root applicability.

Features own the meaning and schema of overlay payloads.

Overlay must not own:

- Plaintext secret reads.
- Pending secret publication.
- Physical secret cleanup.
- Artifact-wide cleanup orchestration.

### Overlay deletion

Deleting an Artifact/namespace overlay retains its current semantic meaning:

- Remove the overlay.
- Detach bindings under the same Artifact/namespace.
- Durably enqueue detached secret values for cleanup.
- Commit those changes atomically.

This atomic cascade is part of the repository operation’s documented contract.

Do not replace it with separately committed Overlay and Secret calls.

After successful detachment, Overlay may request a best-effort cleanup pass through a narrow Secret cleanup capability. Cleanup failure must not undo committed logical deletion.

---

## 5.7 Secret — `store/secret`

Own three distinct responsibilities:

### Binding management

- Public-safe binding metadata.
- Secret replacement requests.
- Secret clearing.
- Expected Artifact/binding revision handling.
- Pending physical-write coordination.

### Trusted plaintext resolution

- Resolve a binding using an expected opaque ref.
- Read the physical value.
- Verify its integrity against binding metadata.
- Return plaintext only through a separately supplied trusted capability.

### Secret lifecycle

- Recover interrupted pending writes.
- Drain the durable cleanup queue.
- Record physical cleanup failures.
- Complete cleanup only when no active binding references the value.

These responsibilities must no longer share the combined Overlay/ArtifactCleanup service.

A binding API must not carry a concrete object that can be asserted into the plaintext runtime API.

Separate private role implementations may share a repository or private state. That is real capability separation, not a compatibility façade.

### Physical secret values — `store/secret/value`

Own only the physical backend contract.

The backend stores:

- Opaque ref → secret value.

It does not store or interpret:

- Artifact identity.
- Namespaces and slots.
- Binding revisions.
- Cleanup policy.
- Feature payloads.

The concrete backend remains `provider/keyringmapstore`.

---

## 6. Flow requirements

## 6.1 Refresh — `store/flow/refresh`

Refresh owns both:

1. Publishing a newly observed Source state.
2. Publishing invalidation caused by relevant Source lifecycle transitions.

### Normal refresh

The workflow is:

1. Read the operational Source.
2. Establish applicable Source and refresh-state revision expectations.
3. Open one Source snapshot.
4. Run bounded Ingest scanning.
5. Obtain admitted Definitions and observations.
6. Ask Artifact synchronization to derive creates and source-state updates.
7. Confirm and close the snapshot.
8. Publish Definitions, Artifact transitions, and refresh state atomically.

The flow owns sequencing and publication intent.

Artifact owns Artifact transition rules.

Definition owns immutable admission rules.

SQLite owns transactional execution and conflict enforcement.

A confirmed snapshot records what was observed. It does not promise that a mutable filesystem will remain unchanged after confirmation. Resource access therefore continues to verify the required generation and content evidence.

### Source lifecycle invalidation

For disabling, retiring, or removing discovery:

- Source determines the Source transition.
- Artifact determines the corresponding source-owned Artifact state change.
- Refresh owns the aggregate invalidation publication.
- The provider applies the Source change, refresh-state invalidation, and Artifact changes atomically.

Do not introduce a new generic maintenance service for this.

Do not update Source first and invalidate Artifacts through a post-commit callback.

Source must not import a Refresh implementation. If Source operations require aggregate publication, inject the specific publication capability they need. The input contract should express a Source transition, not provide arbitrary access to Refresh internals.

If an invalidation plan is computed before the transaction, its preconditions must cover concurrent refresh publication and Artifact changes. Checking only Source revision is insufficient when another refresh can create Artifacts without changing that revision.

---

## 6.2 Resource — `store/flow/resource`

Own:

- Artifact → current Definition → Source → refresh evidence resolution.
- Verified Source entry reads.
- Verified Source tree reads.
- Request-scoped verification sessions.
- Trusted verified native-path resolution.

Separate ordinary verified reads from native-path access.

A filesystem path is not a portable resource projection and must not be part of the ordinary resource capability solely because some trusted runtimes need it.

### Verification sessions

Preserve:

- Snapshot reuse within a coherent operation.
- Source revision and generation consistency.
- Confirmation before successful batch results are accepted.
- Deterministic closure of every retained snapshot.
- Correct nested-session ownership.
- Rejection of a session belonging to another resource service.

Do not remove session behavior as an incidental consequence of moving helpers.

Snapshot bounded reads belong to Source/Driver. Resource owns the multi-entity verification chain and session orchestration.

---

## 6.3 ManagePackage — `store/flow/managepackage`

Own the source-first publication/removal workflow.

### Publication

1. Validate publication intent and authorization.
2. Validate and normalize the complete package.
3. Resolve the operational Source and required physical capability.
4. Establish revision/generation expectations.
5. Publish physical content.
6. Acknowledge changed Source content.
7. Refresh when required.
8. Verify the expected Artifact identity and Definition digest.

### Removal

1. Validate removal intent and authorization.
2. Establish the Source/package expectations.
3. Remove physical content.
4. Acknowledge the Source change.
5. Prune explicitly owned discovery state when requested and permitted.
6. Refresh or perform the required invalidation.
7. Verify that the expected Artifact is missing when the request requires it.

### Required semantics

- Same-content create replay remains idempotent.
- Different content at an existing address requires replacement authorization and the required generation expectation.
- Filesystem and database publication are not one transaction.
- A physical-write success followed by a metadata conflict must remain an explicit recoverable outcome.
- Removal retries must converge without hiding unrelated generation conflicts.
- Discovery pruning is limited to caller-owned authoritative scopes.
- Removing the final discovery scope must still invalidate prior source-backed Artifacts correctly.
- Provider support is determined by capabilities, not hard-coded source-kind strings.

---

## 6.4 Install — `store/flow/install`

Own:

- Protected topology ensure.
- Aggregate hydration preparation and commit.
- Package hydration preparation and commit.
- Protected topology reset.
- Compiled package installation.
- Trusted registration of compiled declaration evidence.
- Coordination of generic bootstrap installers.
- Generic generated package-set serialization/rendering where those helpers exist outside Artifactory today.

Install does not own:

- Built-in feature content.
- Application-specific Root IDs and storage keys.
- Artifact-family schema vocabulary.
- Markdown/YAML declaration conventions.
- Domain-specific package preparation.

### Required installation invariants

- Validate the complete applicable plan before the first physical mutation.
- A shared Root reset invalidates every participating installer using that Root.
- A previous hydration marker survives reset until replacement installation succeeds.
- Package batches are explicitly package-by-package publication, not falsely advertised as globally atomic.
- A Source batch advances Source state and refreshes coherently rather than performing independent per-package refresh loops.
- Hydration markers are committed only after required publication and verification succeeds.
- Reset queues secret cleanup before deleting the metadata that references those secrets.
- Recovery/reset must not race ordinary in-flight writes unless an explicit concurrency protocol permits it.

### Compiled documents

Install owns compiled package plans and installation metadata.

Ingest owns the representation it needs to recognize already-admitted source documents.

Therefore:

- Install translates package-relative compiled declarations into source-relative admitted-document evidence.
- Ingest does not import Install’s package model to maintain its registry.
- Generated payloads remain a trusted registration path, separate from ordinary user-supplied decoder output.
- Runtime verifies source-content witnesses.
- Build-time reproducibility tests establish equivalence with the ordinary decoding/admission path.

Do not expose a general-purpose `SkipValidation` or `AlreadyTrusted` switch on ordinary publication requests.

---

## 6.5 ArtifactCleanup — `store/flow/artifactcleanup`

Own:

- Authorization of Artifact-local state purge.
- Coordination of Artifact existence/lifecycle requirements.
- Atomic removal of overlays and binding references attached to that Artifact.
- Durable enqueueing of detached secret cleanup.
- Optional best-effort physical-secret cleanup after commit.

It does not:

- Delete the Artifact itself.
- Implement Secret garbage collection.
- Own store-scoped overlay deletion.
- Replace Install’s whole-topology reset.

Its repository operation must preserve the transaction boundary across Overlay and Secret binding tables.

---

## 7. Validation and admission policy

The objective is **validation at the correct boundary**, not repeated defensive validation at every function call.

It is also not “validate once at startup and trust all future state.”

## 7.1 Boundary rules

| Boundary                          | Validate here                                                                                             | Do not repeat downstream                                                    |
| --------------------------------- | --------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------- |
| Service construction              | Required dependencies, supported configuration, registered namespaces, mandatory capability relationships | Repeated dependency-nil checks in every method                              |
| Registration                      | Decoder IDs/revisions, schema keys, duplicate registrations, required schema bindings                     | Registry completeness checks during every candidate decode                  |
| Public operation entry            | Request structure, IDs, bounds, required expectations, context contract, authorization                    | Identical request validation in each private helper                         |
| Driver configuration entry        | Provider-specific configuration and normalization                                                         | Re-normalizing unchanged configuration during unrelated metadata operations |
| Decoder/schema output boundary    | Output shape, identity linkage, canonical representation, digest evidence, diagnostics                    | Rehashing the same admitted Definition at every staging step                |
| Snapshot/provider result boundary | Required provider-result invariants and source evidence                                                   | Unrelated domain validation of already established values                   |
| Transaction commit                | Current revisions, liveness, uniqueness, references, immutable-key conflicts                              | Business decisions already expressed in the command                         |
| Resource/session completion       | Source consistency and snapshot confirmation                                                              | Nothing: this is a new time-dependent check, not redundant validation       |

### Construction guarantees

After successful construction:

- Mandatory dependencies are present.
- Configuration is valid.
- Namespace policies are established.
- Registrations are internally consistent.

Private methods may assume those facts.

Optional capabilities are represented and handled intentionally. They are not treated as incompletely initialized mandatory dependencies.

### Request guarantees

Validate an external request before any side effect.

Private helpers called as part of that operation should receive normalized values or an operation plan whose structural invariants have already been established.

Do not thread raw, mutable request objects through many helpers and ask each helper to validate them again.

### Admission guarantees

Define one owner for each expensive admission step:

- Source-content digest calculation: Source/Ingest read boundary.
- Schema canonicalization/execution: schema catalog boundary.
- Definition canonicalization and digest: Definition admission.
- Artifact state derivation: Artifact synchronization.

A canonical source document and a canonical Definition are different values and may legitimately have different digests.

If a codec changes a canonical document, validating the changed output is not a duplicate check of the original input.

### Persistence guarantees

Optimistic concurrency checks remain inside the transaction.

A preflight revision check does not replace a commit-time revision check.

Similarly:

- Source liveness may change.
- Bindings may be replaced.
- An Artifact may be updated.
- A snapshot may become stale.

Those checks must remain where the race is actually closed.

### Mutable values

“Already validated” is only meaningful if ownership prevents later mutation.

Use:

- Owned copies.
- Immutable private representations.
- Carefully scoped borrowed values.

Do not rely on callers leaving exported maps, slices, and pointer-bearing structs unchanged.

### Shutdown

Removing repetitive nil checks must not remove the actual closed-state contract.

Initialization validity and runtime closure are different concerns.

---

## 8. Placement of utilities and former artifact-contract helpers

These are ownership decisions, not a request for a new helper framework.

| Behavior                                                        | Owner                                       | Boundary                                                   |
| --------------------------------------------------------------- | ------------------------------------------- | ---------------------------------------------------------- |
| Portable identifier/locator validation                          | `store/spec`                                | Shared generic value rules                                 |
| Path-pattern matching                                           | `store/spec`                                | Portable matching only                                     |
| Diagnostic cloning, bounds, and equality                        | `store/spec/diagnostic`                     | Shared diagnostic value behavior                           |
| Discovery scope merging                                         | `store/source`                              | Source discovery configuration                             |
| Ensure/reconcile/refresh helper                                 | `store/flow/refresh`                        | Cross-entity lifecycle sequence                            |
| Bounded read of a supplied snapshot entry                       | `store/source/driver`                       | Snapshot contract operation                                |
| Generation/digest verification of Source content                | `store/source`                              | Source evidence operation                                  |
| Shared Artifact verification session                            | `store/flow/resource`                       | Multi-entity read consistency                              |
| Native filesystem root normalization and storage-key derivation | `provider/fsdir`                            | Concrete filesystem Source support                         |
| Default traversal exclusions and Git-submodule handling         | `provider/fsdir`                            | Provider traversal policy                                  |
| Managed staging names and physical file modes                   | `provider/managedfs`                        | Concrete managed storage layout                            |
| Store manifest and local deployment paths/modes                 | `compose/local`                             | Deployment layout                                          |
| Application-wide directory policy                               | Application composition                     | Not a store domain rule                                    |
| Package address/file-set validation                             | `store/source/managedpackage` and its model | Generic complete-package rules                             |
| Reading complete packages from `fs.FS`                          | `store/source/managedpackage`               | Generic package materialization                            |
| Selecting one of caller-specified package document locators     | `store/source/managedpackage`               | Generic selection, no built-in filenames                   |
| Definition body encoding/decoding                               | `store/definition` and its model            | Generic Definition representation                          |
| Generic passthrough codec                                       | `store/definition/schema`                   | Operates under the catalog’s documented admission contract |
| Generic bootstrap coordination                                  | `store/flow/install`                        | Installer sequencing and hydration                         |
| Generic compiled package installer                              | `store/flow/install`                        | Installation, not declaration semantics                    |
| Generic generated package-set decode/render                     | `store/flow/install`                        | Stable generic generated representation                    |
| Artifact ID allocation                                          | `store/artifact`                            | Artifact identity ownership                                |
| Physical secret encryption/keyring names                        | `provider/keyringmapstore`                  | Concrete secret backend                                    |
| Provider-shared MapStore raw I/O                                | `provider/internal/mapstoreio`              | Provider-private implementation reuse                      |
| Typed document projection cache                                 | LLM declaration/family owner                | Not a generic store helper                                 |
| Declaration locator interpretation                              | LLM declaration/locator owner               | Not a generic store helper                                 |
| Package line-ending normalization                               | LLM artifact-family package preparation     | Must occur before expected bytes/digests are fixed         |

The generic helper transfers from an external artifact-contract layer should be made based on actual source inspection. Do not assume that every file with “package,” “schema,” or “install” in its name is generic.

---

## 9. Assembly and resource ownership

Some assembly is necessary. It belongs in one explicit place rather than inside every entity.

### Generic aggregate assembly — `store/compose`

This is the one justified generic assembly boundary.

Its responsibility is to:

- Construct a coherent set of entity and flow services.
- Supply the same relevant policies and registrations consistently.
- Bind decoders to the schema catalog.
- Connect named persistence capabilities.
- Distribute ordinary and trusted capabilities separately.
- Establish the store’s shutdown path.

It must not implement entity or flow behavior.

It receives explicit contracts such as:

- Root persistence.
- Source persistence.
- Artifact persistence.
- Catalog persistence.
- Definition persistence.
- Overlay persistence.
- Secret persistence.
- Refresh publication.
- Installation persistence.
- Artifact cleanup persistence.
- Source drivers.
- Schema factory and codecs.
- Decoders.
- Secret-value storage.
- Clock and Artifact ID allocation.
- Root and namespace policies.

There is no generic `Metadata` or `LocalState` dependency.

### Local deployment — `compose/local`

Own:

- Deployment path validation.
- Manifest/layout handling.
- SQLite opening.
- Local driver construction.
- Concrete JSON Schema provider selection.
- Local secret backend setup when provided by this deployment.
- Cleanup of resources opened by the deployment.
- Calling generic aggregate assembly.

There is no `compose/local/internal/assembly` service graph beneath it.

### Avoid duplicated aggregate surfaces

There should be one aggregate capability surface, not three mirrored sets of fields across:

- `local.Store`.
- `assembly.Components`.
- `storecompose.Owner`.

The local opener should return the generic assembled store handle, with its deployment shutdown arranged explicitly. It should not forward every API through another layer.

### Ownership contract

Adopt one documented rule:

- Business services borrow provider resources.
- A provider resource has exactly one lifecycle owner.
- Caller-supplied resources remain caller-owned if opening fails.
- Ownership transfers on successful opening only when the opening contract explicitly says so.
- Locally opened resources are closed on every failure path.
- Successful shutdown closes them once in a defined order.
- Secret binding/lifecycle services do not independently close a backend owned by deployment.
- Repository interfaces are not inspected for accidental `Close` support.

This replaces the current ambiguous combination of backend closure in the secret service and deployment-level failure cleanup.

---

## 10. Persistence and compatibility constraints

Structural cleanup must preserve:

- Existing database format.
- Table and column names unless a separately justified migration is approved.
- Source-kind strings.
- Root and Source identities.
- Source storage layout.
- Managed package directory addressing.
- Secret ref format.
- Definition digest calculation.
- Decoder/schema fingerprint format.
- Compiled package fingerprint format.
- Generated payload representation.
- Artifact typed-origin identity.
- Protected and retained Root behavior.
- Existing publication and retry semantics.

Changing a Go package name must not change persisted or generated identity.

In particular, preserve deterministic ordering and nil/empty representation where those affect canonical fingerprints.

### Caches

Cache ownership must be explicit:

- SQLite may cache immutable persisted Definition payloads.
- A typed declaration cache belongs to its declaration/family owner.
- Resource verification sessions are operation-scoped, not general caches.
- Ingest compiled-document registration is owner-scoped trusted state.

An immutable payload cache must not bypass mutable existence or liveness rules.

For example, a protected Root purge/reset must not leave cached Root-local Definition availability incorrectly observable after the Root is recreated.

---

## 11. Implementation sequence

Use these work packages rather than the old phase numbers.

### Work package A — Finish actual owner services

Deliver:

- Separate Overlay, Secret, and ArtifactCleanup implementations.
- Named SQLite repository implementations for those owners.
- Independent Artifact and catalog persistence contracts.
- Definition-owned read behavior.
- Public Ingest staging values.
- Removal of implementation aliases and compatibility packages.
- Owner construction entries instead of entity/flow composition façades.
- Actual ordinary/trusted capability separation.
- Root protection independent of Install.
- Ingest independent of Install package models.

This must not end with a temporary `secret/compose.LocalStateRepository` or an equivalent aggregate under another name.

### Work package B — Collapse assembly duplication

Deliver:

- One generic aggregate assembler.
- A thin local deployment opener.
- Deletion of `compose/local/internal/assembly`.
- One aggregate handle and explicit shutdown ownership.
- No concrete provider imports from generic business packages.

This becomes straightforward once work package A establishes the real owner services.

### Work package C — Complete policy and admission boundaries

Deliver:

- Explicit Source lifecycle invalidation commands/publication.
- Removal of hidden lifecycle decisions from SQLite.
- Consolidated Definition admission.
- Removal of repeated initialization checks and repeated expensive admission work.
- Preservation of transactional and snapshot-time checks.
- Resource session and capability-boundary verification.

### Work package D — Complete the LLM/generic contract separation

Coordinate with the second proposal:

- Remove declaration dispatch from the generic JSON Schema provider.
- Transfer remaining generic bootstrap/compiled-install helpers into Install.
- Keep declaration locators, typed decoding, package preparation, and graph behavior in LLM Artifactory.
- Remove obsolete APIs rather than preserving aliases.

Each package should finish at its intended ownership boundary. Do not add a temporary layer whose only purpose is to survive until the next work package.

---

## 12. Completion criteria

The implementing agent must demonstrate:

### Architecture

- No `*/impl` compatibility packages.
- No entity-specific composition bridge packages.
- No generic `localstate`, `metadata`, `maintenance`, `kit`, or global engine owner.
- No provider dependency from entity or flow business code.
- No cross-entity private implementation imports.
- No service interfaces hidden in data-model packages without a specific reason.
- No broad concrete service exposed through an ordinary secret or resource capability.

### Behavior

Tests must cover at least:

- Protected versus retained Root behavior.
- Source creation replay and ambiguous commit handling.
- Source disable/retire/discovery-removal invalidation.
- Refresh publication conflicts.
- Preservation of local Artifact fields across refresh.
- Catalog metadata-only reads and bulk document attachment.
- Resource session confirmation and closure failures.
- Managed publication/removal retry behavior.
- Empty-discovery pruning.
- Shared-Root hydration reset.
- Compiled-path equivalence with ordinary admission.
- Overlay deletion and durable secret detachment.
- Secret replacement failure, recovery, and cleanup retries.
- Failed-open resource ownership.
- Exactly-once close behavior.
- Cache behavior across topology reset.

### Delivery

- Concrete changes against the current source—not an older proposed layout.
- No unimplemented “replace this with the corrected block below” fragments.
- An exact external caller migration inventory.
- Clear identification of tests actually run and any remaining work.

### Artifactory implementation-agent prompt

> Use the last attached Artifactory tree as the baseline. The previously proposed Phase 3 patch was not applied and must not be used.
>
> Implement the Artifactory proposal above as the authoritative target. Preserve entity-versus-flow ownership and all documented storage, digest, fingerprint, secret, and package-publication invariants.
>
> Begin by finishing the actual owner services: split Overlay, Secret, and ArtifactCleanup; split their SQLite repositories; separate Artifact persistence from catalog queries; establish Definition read guarantees; expose only the required Ingest staging values; and remove implementation compatibility façades.
>
> Do not replace `impl` packages with entity `compose` wrappers, exported aliases, generic components, or a renamed LocalState aggregate. Public construction belongs to the actual owner. Use unexported owner-local implementations where that eliminates artificial Go package cycles.
>
> Remove dependency inversions: Root protection must not depend on Install, and Ingest must not depend on Install package-plan models. Separate ordinary binding/resource capabilities from plaintext/native-path capabilities through actual implementation method sets.
>
> Then consolidate aggregate assembly and local deployment according to the proposal, including explicit resource ownership and failure cleanup. Do not duplicate the aggregate API fields across multiple handles.
>
> Validate mandatory dependencies and configuration at initialization, requests at entry, and plugin outputs at their trust boundary. Retain transaction-time concurrency checks and resource-time consistency checks. Do not introduce caller-controlled validation bypass flags.
>
> Inspect external usages before contracting APIs. Keep the implementation scoped to Artifactory and its tests unless broader changes are authorized, and provide exact caller migration instructions.
>
> Deliver coherent complete patches, architecture checks, regression tests, and an honest test report. If policy extraction or coordinated LLM changes require a subsequent delivery, identify the exact remaining work without adding a temporary forwarding layer.
