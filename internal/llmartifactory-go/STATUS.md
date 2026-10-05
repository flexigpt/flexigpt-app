# LLM Artifactory status

## Current boundary

The application now has four explicit layers:

```text
artifactbuiltin
        application-shipped embedded files and generated catalogs
artifactsetup
        application topology, registration selection, installation setup,
        immutable embedded-provider evidence, and startup assembly
llmartifactory-go
        LLM declaration grammar, family contracts, Plugin membership,
        composition, capability planning, and family-local semantics
artifactory-go
        generic Root, Source, Definition, Artifact, Overlay, Secret, and
        cross-entity lifecycle flows
```

`internal/artifactory-go` remains the generic owner of Root, Source,
Definition, Artifact, Overlay, Secret, Refresh, Resource, ManagePackage,
Install, and ArtifactCleanup behavior.

`internal/llmartifactory-go` does not own generic persistence, Source-driver
selection, filesystem layout, secret backend ownership, or protected topology.

## Proposal 1 baseline

Proposal 1 is the completed generic Artifactory foundation.

Completed generic responsibilities include:

- Entity-owned Root, Source, Definition, Artifact, Overlay, and Secret state.
- Explicit Refresh, Resource, ManagePackage, Install, and ArtifactCleanup
  flows.
- Source-owned discovery preparation and lifecycle invalidation publication.
- Definition admission, immutable Definition lookup, and catalog projection
  separation.
- Verified entry, tree, Artifact, and trusted native-path reads.
- Request-scoped verification sessions.
- Observable managed-package publication and removal outcomes.
- Atomic Artifact-local cleanup of selected Data namespaces, overlays,
  bindings, and durable physical-secret cleanup records.
- Secret pending/active/cleanup recovery semantics.
- One `store/compose.Store` aggregate and one `compose/local` deployment
  opening boundary.
- Immutable embedded `iofs` registration evidence.

No local forwarding `Store` wrapper remains.

## Proposal 2 Package A

Package A ownership boundaries are established.

- `llmartifactory-go` is the LLM-domain attachment over one generic Store.
- `artifactbuiltin` owns embedded package files and generated catalog payloads.
- `artifactsetup` owns application Root IDs, Source IDs, storage keys,
  protection, retention, generated catalog installation selection, and
  embedded-provider registration.
- Tool generated catalog membership is supplied from `artifactsetup` through
  `tool/domain.BuiltinCatalog`; Tool family code no longer imports
  `artifactbuiltin`.
- Model consumer construction accepts an explicitly supplied protected
  built-in Root instead of asserting FlexiGPT topology from inside the family.
- Workspace default policy content is assembled by
  `artifactsetup/workspace.DefaultWorkspaceConfig`.
- The application owns the generic Store through `artifactsetup.Handle`;
  `App` no longer mirrors Store ownership with a second long-lived pointer.

The remaining relocation of application-selected package filenames and source
conventions out of `artifactsetup/topology` is Package F work. Those imports
do not reintroduce generic Store ownership into LLM Artifactory, but they
remain an extraction boundary to finish.

## Proposal 2 Package B

The generic features required by LLM Artifactory are integrated.

- Source discovery preparation is Source-owned and explicit.
- Resource sessions cover verified entries, trees, stats, Artifact resolution,
  and trusted native paths.
- ManagePackage reports physical and catalog publication boundaries.
- Artifact cleanup can remove feature-owned Data namespaces atomically.
- Secret recovery checks active binding references before cleanup.
- Local opener ownership transfer occurs only after full success.
- Generated package-set preload and immutable embedded-provider registration
  remain generic Install/provider behavior.
- Application setup now registers the embedded Workspace policy with immutable
  `iofs` evidence rather than using the mutable embedded snapshot path.

## Proposal 2 Package C

Declaration ownership is complete at the declaration boundary.

- Shared grammar, header dispatch, canonical Entry values, member identity,
  locators, and schema dispatch live under `core/declaration`.
- Family contracts own schemas, semantic validation, typed reconstruction,
  relationship extraction, and family identity qualifiers.
- Generic JSON Schema execution accepts complete expected schema keys only.
- `apiVersion` is dispatch grammar rather than Definition content:
  omitted v1 and explicit `apiVersion: v1` retain canonical Definition bytes.
- Composition reconstructs family relationships through
  `AdmittedRelationships`; it does not execute already-completed family JSON
  Schema validation after Definition admission.
- Typed family reconstruction validates admitted Definition linkage and
  family-owned immutable invariants without recomputing Definition digests.

## Proposal 2 Package D

Plugin and composition ownership are complete.

- Plugin is the only managed membership Artifact family.
- Legacy `collection.yaml`, `collection.yml`, and `collection.json` documents
  are no longer accepted as Plugin documents.
- Family and Wails transport vocabulary uses `Plugin`, not `Collection`.
- Plugin membership policy is declaration metadata with explicit
  `single-type`, `mixed-type`, and read-only legacy-unconstrained semantics.
- Agent, Skill, MCP, and Tool profiles own their membership restrictions.
- Plugin direct membership and recursive capability planning are separate:
  - `ResolveDirectMembers` resolves immediate members and selector matches.
  - `ResolveCapabilities` recursively projects reachable capabilities.
- Plugin display order, duplicate detection, add-or-ensure behavior, removal,
  and reverse-membership reporting share one declaration-owned normalized
  member ordering.
- Profile-owned baselines are identified consistently during reads, editing,
  listing, and deletion eligibility checks.
- Composition owns lookup order, aliases, selectors, scope binding,
  ambiguity handling, cycle limits, direct capability targets, and capability
  plan projection.
- Normal composition resolution remains read-only.
- Explicit reachable discovery preparation and refresh closure remain under
  `core/composition/refresh`.

## Remaining Proposal 2 Package E work

Package E remains deliberately separate from the completed A-D boundary.

- Replace remaining `consumerapi` ownership with family-root services and
  family-owned API packages.
- Complete Text service extraction from Agent materialization entrypoints.
- Complete Skill source-format, package, and materialization consolidation.
- Split Model and Model Provider services into named settings, credential,
  preference, package, and capability owners.
- Move MCP installation and logical-secret behavior from aggregate-era
  wrappers into `mcp/installation` and `mcp/secret`.
- Add Team, Loop, and Workflow family plan projections.
- Complete Go-only Tool narrowing by removing SDK Tool declaration, generated
  inventory, target, and runtime branches.
- Move remaining family-local package naming helpers into their family
  package owners.

## Remaining Proposal 2 Package F work

Package F is application binding and runtime integration work.

- Bind Go Tool metadata direct-capability providers.
- Bind inference adapter and Model Provider direct-capability providers.
- Bind runtime-neutral MCP preparation providers where needed.
- Move remaining application-selected filename and source-format selection out
  of `artifactsetup/topology`.
- Move wrapper-owned MCP secret translation, MCP settings, and Model
  preference persistence into their family owners.
- Reduce Wails wrappers to transport/recovery concerns after runtime adapter
  construction moves into application setup.
- Migrate outer runtime aggregate interfaces from their former Collection
  terminology to Plugin terminology.

## Position relative to the destination

Proposal 1 is complete.

Proposal 2 Packages A through D are complete at their intended declaration,
composition, membership, and application-content boundary. The project is now
at the intended intermediate architecture:

```text
application content and topology
        ↓ explicit setup registrations
LLM declaration grammar + family contracts + Plugin + composition
        ↓ generic entity and flow capabilities
Artifactory Root / Source / Definition / Artifact / local state
```

Packages E and F remain family-service consolidation and application/runtime
binding work. They no longer require another generic Store redesign, another
declaration dispatcher, another membership abstraction, or another graph
resolver.
