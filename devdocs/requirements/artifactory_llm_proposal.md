# Proposal 2 — LLM Artifactory: declaration semantics, typed resolution, and runtime planning

## 1. Purpose

LLM Artifactory is the domain layer that gives generic Artifactory content its LLM-related meaning.

It owns:

- Declaration vocabulary.
- Declaration schema identities and semantic validation.
- `type` / `apiVersion` dispatch.
- JSON, YAML, Markdown, and other declaration adapters.
- Projection of valid declarations into generic Definitions.
- Artifact-family package preparation.
- Declaration locator interpretation.
- Typed declaration reconstruction from admitted Definitions.
- Relationship and dependency graph resolution.
- Runtime capability planning.
- Feature-owned overlay payload semantics and secret slot semantics.
- Domain-specific built-in content and installation declarations where the application delegates those responsibilities to it.

It does not own the generic Store lifecycle merely because an LLM operation uses it.

This proposal is a responsibility specification, not a verified file-by-file inventory of the LLM repository. The implementing agent must first map its actual current packages to these responsibilities.

---

## 2. Boundary with Artifactory

| Concern                                         | Artifactory                          | LLM Artifactory                                         |
| ----------------------------------------------- | ------------------------------------ | ------------------------------------------------------- |
| Root/Source persistence                         | Owns                                 | Uses                                                    |
| Source drivers and snapshots                    | Owns                                 | Receives verified reads through store capabilities      |
| Source discovery configuration mechanics        | Owns                                 | Chooses family/workspace discovery policy               |
| Decoder execution and bounded input             | Owns                                 | Supplies declaration decoders                           |
| Declaration recognition                         | Executes registered contract         | Defines behavior                                        |
| JSON Schema execution                           | Generic provider                     | Supplies schemas and semantic codecs                    |
| `type` / `apiVersion` dispatch                  | Does not own                         | Owns                                                    |
| Definition canonical envelope and digest        | Owns                                 | Supplies semantic Definition content                    |
| Artifact identity/lifecycle                     | Owns                                 | Uses                                                    |
| Typed declaration objects                       | Does not own                         | Owns                                                    |
| Catalog projection                              | Owns                                 | Maps into domain-facing lists                           |
| Declaration locator syntax                      | Does not own                         | Owns                                                    |
| Dependency graph expansion                      | Does not own                         | Owns                                                    |
| Generic package write/refresh sequence          | Owns                                 | Supplies validated publication intent                   |
| Package filenames and text normalization        | Does not own                         | Owns by artifact family                                 |
| Binding persistence and physical secret cleanup | Owns                                 | Defines legal slots and when bindings are needed        |
| Plaintext access policy                         | Supplies trusted mechanism           | Application/runtime decides use                         |
| Native-path mechanism                           | Supplies trusted verified capability | Application/runtime decides approval and sandbox policy |
| Generic hydration protocol                      | Owns                                 | Supplies topology/content and domain hooks              |
| Built-in product meaning                        | Does not own                         | LLM layer/application                                   |

No semantic rule should exist in both layers “for safety.”

Independent checks at different trust or concurrency boundaries are allowed. Duplicate ownership of the same semantic decision is not.

---

## 3. Required responsibility areas

Use actual domain ownership rather than a generic contract kit.

## 3.1 Declaration contract

Own:

- Supported declaration types.
- Version vocabulary.
- Header rules.
- Field semantics.
- Cross-field semantic validation.
- Canonical declaration representation.
- Mapping between declaration identity and generic Artifact/Definition identity.

This is where rules about:

- `type`.
- `apiVersion`.
- Legacy headers.
- Mixed header forms.
- Omitted versions.
- Unsupported declaration versions.

must live.

The generic schema provider must not recreate them.

### Dispatch invariant

Dispatch must resolve a document to one unambiguous expected schema key before expected-key schema canonicalization.

An omitted version is permitted only when the declaration contract defines an unambiguous result.

The generic schema catalog may contain multiple full schema keys that an LLM dispatch scheme cannot distinguish. That ambiguity is the LLM dispatcher’s error—not a reason for the generic catalog to know declaration headers.

---

## 3.2 Artifact families

Each supported family owns its specific semantics.

Examples from the current ecosystem include Model, MCP, Tool, Skill, Collection, Agent, Context, and Workspace declarations. The actual family inventory must come from the repository.

A family owns, where applicable:

- Its published schema.
- Semantic codec behavior.
- Declaration recognition/adaptation.
- Typed Definition reconstruction.
- Package document conventions.
- Legal relationships and dependency expectations.
- Feature overlay schemas.
- Secret namespace/slot semantics.
- Runtime requirements.
- Built-in content declarations.

Do not centralize unrelated family decisions in a new generic `artifactcontract` replacement.

Shared declaration-language behavior can remain shared when it is genuinely part of one declaration contract.

---

## 3.3 Declaration decoding and adaptation

Own:

- JSON/YAML declaration parsing.
- Markdown adapters.
- Family-specific manifest interpretation.
- Projection of source bytes into canonical declaration documents.
- Conversion into generic Definition content.
- Domain diagnostics.

Artifactory supplies:

- Candidate bytes.
- Source identity.
- Requested decoder information.
- Bounded sibling reads when permitted.
- Schema-catalog access through the documented binding contract.

Decoders must not:

- Open native paths.
- Read Source configuration.
- Write package files.
- Persist Definitions.
- Generate Store Artifact IDs.
- Refresh Sources.
- Mutate catalog state.

### Source-aware formats

A source-aware decoder may emit declarations whose physical origin differs from its initial candidate.

The emitted locator/digest pair must describe the actual source entry used for that declaration. It must not attach one file’s bytes to another file’s origin.

Reading referenced siblings still goes through the supplied bounded Source reader.

---

## 3.4 Declaration locators

Own:

- Portable declaration locator syntax.
- Family-specific locator validation.
- Interpretation of expected kind and logical identity.
- Subresource conventions.
- Read-only binding to catalog/runtime lookup capabilities.
- Locator resolution limits and diagnostics.

The old generic locator-resolver contracts should be relocated here where their meaning can be stated directly.

Do not preserve JSON-shaped “generic locator input” solely to avoid importing the declaration contract into a package that should never have owned that interpretation.

### Read-only resolution

Locator resolution must not secretly:

- Create a Source.
- Expand discovery.
- Refresh a Source.
- Publish a package.

A workspace/source preparation operation may explicitly perform those steps before resolution. The ownership and ordering must be visible.

Resolution itself consumes committed or explicitly verified state.

---

## 3.5 Typed Definition access and caching

Own:

- Reconstruction of domain types from admitted Definitions.
- Verification that a Definition is appropriate for the requested family/schema.
- Immutable typed projections.
- Typed document caches.

A generic `DocumentCache[T]` used only for declaration projections belongs here, not in Artifactory’s local composition.

Cache identity must include the information required to distinguish:

- Root-local Definition identity.
- Relevant interpretation/contract revision when projections can change within the cache lifetime.

Returned mutable values must be owned or clearly documented as borrowed immutable values.

Do not let a cache bypass:

- Artifact relationship validation.
- Current source verification when execution requires it.
- Authorization or enablement decisions.
- Root/Definition availability.

A typed projection cache is not proof that the underlying Source is still current.

---

## 3.6 Relationship resolution and capability planning

Own:

- Typed relationships.
- Dependency expansion.
- Version/selector interpretation.
- Cycle detection.
- Graph limits.
- Ambiguity handling.
- Runtime capability requirements.
- Domain-level compatibility checks.
- Plans that describe what execution needs.

Artifactory supplies entities, admitted Definitions, catalog projections, and verified material. It does not infer an LLM execution graph.

Graph planning should not automatically execute capabilities or fetch plaintext secrets.

Separate:

1. Resolving the declaration graph.
2. Validating that the graph is permitted and coherent.
3. Constructing the runtime plan.
4. Acquiring approved runtime material.

The actual application/runtime retains authority over execution.

---

## 3.7 Artifact-family package preparation

Own:

- Primary document names.
- Package-internal resource conventions.
- Declaration serialization.
- Required file selection.
- Text and line-ending normalization.
- Family-specific package validation.
- Expected Artifact identities and Definition digests.

Use Artifactory for:

- Generic file-set safety.
- Semantic managed-package address shape.
- Physical publication.
- Source revision acknowledgment.
- Refresh.
- Expected Artifact verification.

### Publication invariant

All family-owned normalization must happen **before** the expected bytes and digests are fixed.

A package must not be fingerprinted and then silently rewritten during publication.

Do not reproduce the Store’s source-first workflow in every family service.

A family should prepare valid publication intent and invoke the ManagePackage flow.

---

## 3.8 Local feature state and secrets

Own:

- Overlay payload schema and migration policy.
- Legal namespace usage.
- Legal secret slots for each declaration.
- Domain-specific decisions about clearing or preserving bindings.
- How a runtime uses a resolved secret.

Artifactory owns:

- Overlay persistence and revisions.
- Binding metadata.
- Pending/active/cleanup lifecycle.
- Physical secret backend access.
- Durable detachment and garbage collection.

Requirements:

- Plaintext must not enter Definition bodies, Artifact Data, overlay payloads, generated catalogs, ordinary DTOs, diagnostics, or logs.
- Feature transport-facing services receive binding metadata/write capability, not plaintext runtime capability.
- Universal Artifact enablement remains universal local Artifact state. Do not recreate it as a competing feature overlay.
- Protected built-in content is not automatically authorized for execution.
- A request field must not manufacture installer privilege.

---

## 3.9 Built-in content and installation

The LLM layer/application owns:

- Which built-ins exist.
- Their source bytes.
- Their family meanings.
- Their Root/Source declarations.
- Their desired package inventory.
- Family-specific pre/post installation work.

Artifactory Install owns:

- Generic bootstrap sequencing.
- Hydration markers.
- Package batching.
- Protected reset.
- Source revision publication.
- Refresh.
- Generic compiled expectation verification.

Generic bootstrap and compiled-install support currently residing in an artifact-contract layer should move to Artifactory only if it can operate exclusively on generic store values and capabilities.

Domain-specific package preparation and built-in content selection remain here.

### Generated-content invariant

Generated compiled data must be equivalent to the ordinary path:

source bytes → declaration adapter → expected-schema validation → semantic projection → Definition admission

Tests must establish that equivalence.

Runtime compiled registration is not a substitute for build-time domain validation.

---

## 4. Validation rules

The same boundary discipline applies here.

### Initialization

Validate once:

- Family registrations.
- Dispatch maps.
- Duplicate or ambiguous schema/type/version mappings.
- Required decoder/schema bindings.
- Registered locator resolver slots.
- Required runtime dependencies.

Services may then assume those relationships are present.

### Declaration entry

Validate:

- Raw input bounds and parseability at the appropriate parser boundary.
- Declaration header rules.
- Expected schema selection.
- Schema and semantic correctness.
- Package-specific meaning when preparing a package.

Do not execute the same schema again inside a projection helper whose documented input has already passed that schema.

### Graph entry and resolution

Some semantic checks cannot occur during standalone document admission.

These include:

- Whether a referenced Artifact exists.
- Whether relationships resolve unambiguously.
- Whether the complete graph contains a cycle.
- Whether the requested runtime has the required capabilities.
- Whether a selected Artifact is enabled and permitted for execution.

Perform these at graph/planning/runtime boundaries. They are not redundant admission checks.

### Runtime acquisition

Recheck time-sensitive facts through the Store’s verified APIs and trusted runtime capabilities.

An admitted declaration is not proof of current source bytes or current secret binding state.

---

## 5. Concrete invariants

1. **Store identity is Store-owned.**
   Families do not invent persisted Artifact IDs or directly mutate source-derived Artifact state.

2. **Declaration identity is contract-owned.**
   Families define declaration kind, logical identity, schema, and semantic content.

3. **Digest domains are distinct.**
   Source bytes, canonical declaration documents, Definitions, package inventories, and registry fingerprints must not be used interchangeably.

4. **Dispatch is explicit.**
   The declaration layer selects an expected schema key; generic schema validation does not infer LLM headers.

5. **Resolution is not hidden ingestion.**
   Locator and graph reads do not silently create, refresh, or publish store content.

6. **Packages are prepared before publication.**
   Family normalization finishes before expected digests and publication expectations are established.

7. **Typed caches are not freshness evidence.**
   Verified source material still requires the Resource flow where appropriate.

8. **Runtime authority is separately supplied.**
   Plaintext and native-path access are not recoverable by asserting an ordinary API into a broader concrete service.

9. **Source material remains portable until runtime requires otherwise.**
   Native paths are a trusted deployment detail, not declaration data.

10. **Generic lifecycle behavior is not duplicated.**
    Source revision sequencing, refresh publication, hydration bookkeeping, and secret garbage collection remain in Artifactory.

11. **New families do not modify the generic store.**
    They supply schemas, decoders, domain resolution behavior, and package preparation.

12. **New storage providers do not modify declaration semantics.**
    Provider extension is independent of family extension.

---

## 6. What should not exist in the final LLM layer

Remove or avoid:

- A generic contract kit containing unrelated helpers.
- A universal provider descriptor passed into Artifactory.
- Direct imports of Artifactory private implementations.
- Direct SQL access for Store entities.
- Direct filesystem reads that bypass verified Source access.
- Family-specific copies of managed publication sequencing.
- Family-specific secret cleanup engines.
- Multiple schema dispatch implementations with slightly different header rules.
- A generic registry that owns both declarations, source drivers, locators, and runtime services.
- Compatibility aliases preserved indefinitely after responsibility migration.

Convenience registration may exist at an actual family or LLM-module boundary, but it must unpack into explicit Artifactory registrations. It does not become a new generic Store abstraction.

---

## 7. Acceptance criteria

The LLM implementation must demonstrate:

- All declaration header dispatch is outside Artifactory’s JSON Schema provider.
- Decoder adapters use bounded supplied inputs and readers.
- Generic Definition and Artifact lifecycle is not duplicated.
- Locator resolution and source preparation are visibly separate operations.
- Typed graph rules are owned by the appropriate declaration/family layer.
- Feature list operations use catalog projections unless complete Artifacts are genuinely required.
- Typed caches preserve immutability and do not bypass runtime verification.
- Plaintext/native-path capabilities do not leak into ordinary APIs.
- Generated built-ins reproduce ordinary admission results.
- Adding a family requires no generic Artifactory business-code changes.
- Generic bootstrap/package helpers have been transferred to their actual Store owners rather than copied.

### LLM Artifactory implementation-agent prompt

> Inspect the current LLM Artifactory and artifact-contract code before changing it. Use the two proposals above as the responsibility specification; do not assume the package names or file inventory from an earlier response are current.
>
> Map every current responsibility to declaration contracts, artifact families, declaration decoding, declaration locators, typed Definition access, graph/capability planning, package preparation, local feature state, or built-in content.
>
> Keep generic Root/Source/Artifact/Definition persistence and lifecycle, managed publication sequencing, hydration bookkeeping, resource verification, and secret cleanup in Artifactory.
>
> Move `type` / `apiVersion` dispatch and declaration-header validation out of the generic JSON Schema provider into the declaration layer. Resolve an expected schema key before invoking Artifactory’s expected-key canonicalization capability. Preserve ambiguity and compatibility behavior deliberately and test it.
>
> Relocate declaration locator contracts to their semantic owner. Keep resolver operations read-only; make Source preparation and refresh explicit outside resolution.
>
> Transfer genuinely generic package-reading, bootstrap, compiled-install, and generated-package helpers to their Artifactory owners where not already done. Do not transfer family filenames, line-ending normalization, built-in content, typed graph rules, or runtime policy.
>
> Use explicit schema/decoder registrations and narrow Store capabilities. Do not introduce a replacement generic provider descriptor, contract kit, or compatibility façade.
>
> Validate registrations at initialization, declaration inputs at admission, graph semantics at resolution, and time-sensitive material through verified runtime access. Do not repeatedly execute schema validation in already-admitted projection helpers.
>
> Separate ordinary metadata/binding operations from trusted plaintext and native-path acquisition. Preserve digest-domain distinctions, package-byte determinism, typed-cache immutability, and generated-path equivalence.
>
> Deliver a responsibility inventory, complete coherent changes, exact caller migration instructions, regression tests, and a report separating verified behavior from any remaining coordinated Artifactory work.
