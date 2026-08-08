# Skills on Artifact Store High-Level Design

## Document status and authority

This document defines the current authoritative design baseline for Skills on
Artifact Store. It is intended to supersede the prior Skill HLD and its
current-scope replacement after review and adoption.

This HLD defines Skill-specific architecture, ownership, invariants,
requirements, and current product boundaries. Generic Root, Source,
Collection, Catalog, Artifact, Definition, managed-publication, and protected
topology mechanics are defined by the Artifact Store HLD.

Authority is ordered as follows:

- This HLD governs intended Skill architecture, product decisions, and change
  gates.
- Implemented code, embedded schemas, and public API contracts are the source
  of truth for current behavior.
- A conflict between this HLD and implemented behavior must be resolved
  explicitly. It must not become an undocumented compatibility rule.

This document does not commit the product to future Skill transfer work.
Standalone transfer, Bundle transfer, archives, closures, and import provenance
remain out of scope until a later HLD amendment is approved.

## Purpose

Skills are source-backed domain content whose semantic authority is `SKILL.md`.

The Skill domain provides:

- Installed Skill Bundles as local `skill.bundle` Collections.
- Installed Skills as local `agent.skill` Artifacts.
- Shared Skill decoding for installed Bundles and Workspaces.
- Linked external and library Skills without package copying.
- Managed application-authored Skill packages.
- Protected built-in Skill package hydration.
- Artifact-backed runtime projection into Agent Skills.
- Artifact-backed durable Skill selection and runtime allow-lists.

The Skill domain does not own Root administration, generic metadata
persistence, generic Source safety, native Source configuration, generic
package transfer, or Agent Skills runtime sessions and execution policy.

## Current scope

Skills currently support four local ownership modes.

| Mode           | Native-byte authority              | Native package location     | Artifact Store copy behavior  |
| -------------- | ---------------------------------- | --------------------------- | ----------------------------- |
| External Skill | User-selected filesystem Source    | Original external directory | No package-tree copy          |
| Library Skill  | Same-Root linked filesystem Source | Original library directory  | No package-tree copy          |
| Managed Skill  | Application-authored package       | Bundle-owned managed Source | Written by the application    |
| Built-in Skill | Embedded application package       | Protected managed Source    | Hydrated by trusted installer |

There is no imported Skill mode in the current delivery.

### Explicit non-goals

The following are intentionally unsupported:

- Standalone Skill import or export.
- Skill Bundle import or export.
- Archive creation or extraction.
- Generic Skill Content Closure generation.
- Portable package provenance.
- Imported Skill Attachment roles.
- Source-independent offline copies of linked Skills.
- Generic package CAS.
- Cross-Root transfer.
- Direct Skill or Artifact move between Bundles or Workspaces.
- General rebinding of an existing Skill to another Source Binding.
- Generic URI acquisition.

`artifact.Service.Move` remains unsupported.

## Architectural statement

A Skill has one semantic authority:

```text
SKILL.md
```

The shared Agent Skill decoder parses `SKILL.md` and derives one immutable
canonical `agent.skill` Definition. The Definition is a semantic projection,
not a portable package archive or byte-for-byte replacement for a linked Skill
directory.

An installed Skill is represented locally as:

```text
ArtifactKind = "agent.skill"
```

An installed Skill Bundle is represented locally as:

```text
CollectionKind = "skill.bundle"
```

A Workspace-discovered Skill is also an `agent.skill` Artifact, but belongs to
`workspace.collection` and remains governed by Workspace policy. Skill Bundle
and Workspace share decoder and Definition semantics but do not share
Collection ownership or lifecycle policy.

## Goals

The Skill domain must:

- Treat `SKILL.md` as the semantic authority for every Skill.
- Use one shared `agent.skill` decoder and Definition contract for Bundle and
  Workspace discovery.
- Keep local Skill identity, local settings, Source configuration, runtime
  state, and native paths outside the Skill Definition.
- Support linked external and library Skills without copying user-selected
  package trees.
- Support application-authored managed Skill creation, same-binding update,
  retry, and deletion.
- Support protected built-in package hydration with static application-owned
  local IDs.
- Preserve local Artifact identity and local settings across compatible source
  updates.
- Use `ArtifactRef` for durable Skill selections.
- Project only current verified Skills into ephemeral runtime objects.
- Fail closed on runtime Skill-name collisions.
- Keep installed Skill Bundle ownership separate from Workspace Skill
  ownership.
- Keep the legacy standalone Skill Store inactive in normal application
  composition.

## Domain model

| Skill concept            | Artifact Store representation                                          |
| ------------------------ | ---------------------------------------------------------------------- |
| Installed Skill Bundle   | Local `skill.bundle` Collection                                        |
| Installed Skill          | Local `agent.skill` Artifact                                           |
| Workspace Skill          | `agent.skill` Artifact in `workspace.collection`                       |
| Durable Skill selection  | `ArtifactRef` plus local selection behavior                            |
| Skill package entrypoint | `SKILL.md` at a Source-relative locator                                |
| Runtime Skill            | Ephemeral Agent Skills runtime definition                              |
| Built-in registration    | App-owned static Collection and Artifact IDs mapped to package members |

### Skill Bundle

A Skill Bundle is a local Collection. It contains local identity, display data,
enablement, local Bundle data, Source Attachments, Catalog state, Artifacts,
Suppressions, and retirement state.

Bundle local data includes logical Bundle metadata and, where applicable, the
ID of a Bundle-owned managed Source. It is local state and is not a portable
Bundle manifest.

A local Bundle is never exported in the current delivery.

### Skill Artifact

A Skill Artifact has a durable reference:

```text
ArtifactRef {
  RootID
  ArtifactID
}
```

Its local record includes Collection membership, Source Binding, current
Definition digest when available, adoption mode, enablement, local managed
operation data where applicable, diagnostics, revision, and timestamps.

Runtime path, runtime `SkillDef`, package directory, runtime registration,
session state, and Source configuration are never durable Skill identity.

### Package convention

The current decoder recognizes a `SKILL.md` file only when discovery explicitly
requests the shared Skill decoder. `SKILL.md` must belong to a containing Skill
directory. The expected Skill name is the parent directory name.

A normal Skill package may contain:

```text
<skill-name>/
  SKILL.md
  resources/
  scripts/
```

Resources and scripts are supported as native package files for linked,
managed, and built-in runtime behavior. They are not currently modeled as a
generic portable closure and cannot be transferred through a generic export or
import facility.

## Source Attachment roles

A `skill.bundle` supports exactly these local Attachment roles.

| Role       | Meaning                                          | Current source constraint                                                              |
| ---------- | ------------------------------------------------ | -------------------------------------------------------------------------------------- |
| `managed`  | Bundle-owned application-authored package Source | One managed attachment at most; requires `managed-directory`                           |
| `builtin`  | Protected application-owned package Source       | One built-in attachment at most; requires protected `managed-directory` installer flow |
| `external` | User-selected filesystem Skill Source            | Requires `fs-directory`                                                                |
| `library`  | Reusable same-Root filesystem Skill Source       | Requires `fs-directory`                                                                |

`imported` is not a supported role.

The `managed` role cannot be attached arbitrarily. It is provisioned as the
Bundle-owned managed Source during Bundle creation. The `builtin` role is
reserved for trusted protected-topology installation. Ordinary generic Bundle
attachment APIs may attach only eligible external or library Sources.

Attachment roles, discovery roots, and expected member digests are local
Bundle policy. They are not portable Skill metadata.

## Skill Definition and source authority

The canonical `agent.skill` Definition contains semantic fields derived from
`SKILL.md`, including logical name, display name, description, arguments,
insert behavior, tags, Markdown instructions, and domain-approved frontmatter
projection.

The Definition must not contain:

- Root, Source, Collection, or Artifact IDs.
- Local enablement or aliases.
- Local user tags.
- Absolute paths or package layout paths.
- Runtime registrations or session state.
- Credential values.
- Local diagnostics.

The exact raw `SKILL.md` byte digest belongs to the Catalog Observation and
runtime verification path. Resources and scripts remain native Source content.

For linked Skills, Artifact Store may retain a semantic Definition projection,
including parsed Markdown semantics, but does not retain a full native Skill
package as an offline fallback.

## Current Skill Collection document profile

The current schema-backed Skill Collection document is:

```text
kind:          skill.bundle
schemaID:      skill.bundle.v1
schemaVersion: v1
file name:     collection.json
```

This document profile is currently used for embedded built-in package
hydration and canonicalization. It is not a public Bundle import or export
format.

Current member behavior is intentionally limited:

- Members use generic `ContentRef` forms.
- Built-in hydration requires a package-relative `SKILL.md` locator.
- Built-in member integrity is the exact raw `SKILL.md` digest.
- There are no stable portable member keys.
- There are no expected derived Definition digests.
- There are no closure digests or package profiles.
- There is no generic persisted Collection-document repository.
- URI forms can be structurally valid but are not resolved in current Skill
  workflows.

Static built-in Artifact registration maps an app-owned Artifact ID to a
package-relative `SKILL.md` locator. It does not map to a portable member key.

## Base invariants

- `SKILL.md` is the single semantic authority for a Skill.
- Every current Skill is represented as an `agent.skill` Artifact.
- Local Artifact identity never enters `SKILL.md` or `collection.json`.
- Installed Bundle ownership and Workspace ownership remain separate.
- Durable Skill selections use `ArtifactRef`, never Bundle slug, source path,
  runtime `SkillDef`, or legacy Skill identity.
- Linked external and library Skill package trees are not copied into managed
  package storage.
- Managed Skill packages are application-authored local content, not imports.
- Built-ins are application-owned packages hydrated into protected managed
  storage.
- Local aliases, enablement, Bundle display state, managed-operation data, and
  runtime state remain local.
- Source reconciliation preserves local Artifact identity and local fields.
- Runtime registration is derived from current Artifact Store state and is not
  a second durable Skill catalog.
- Same-name runtime ambiguity is fail-closed.
- Scripts are disabled by default unless application composition explicitly
  enables them.
- No current flow imports, exports, archives, snapshots, or creates portable
  closures for Skills.

## Linked external and library Skill behavior

An external or library Skill remains linked to its original filesystem Source.
Artifact Store stores local Source configuration, Attachment policy, Catalog
state, source generation, source-content digest, derived Definition, and local
Artifact state. It does not store a copied package tree, archive, provenance
record, or offline resource fallback.

A direct Skill-directory selection must be normalized to the decoder contract.
For a selected directory such as a Skill package directory, the Source root
must normally be its parent directory and the Attachment discovery root must
select the Skill directory. A Source rooted directly at the Skill directory
does not satisfy the current decoder convention because `SKILL.md` may not be
at the Source root.

External and library Attachments participate in normal Bundle discovery and
can be automatically adopted through feature-provided IDs. Suppression prevents
a removed local Artifact from being automatically recreated on later refresh.

If linked content changes or disappears:

- Refresh reconciles Catalog and Artifact source-derived state.
- Runtime resolution fails closed when source generation or raw `SKILL.md`
  content no longer matches current Catalog state.
- A retained Definition can support diagnostics and display but is not a
  native package fallback.

## Managed Skill lifecycle

Managed Skills are application-authored packages written through a
Bundle-owned `managed-directory` Source.

### Creation and update

A managed create or update operation:

1. Accepts raw `SKILL.md`, a structured Skill document, or a complete package
   file list.
2. Validates and derives the shared `agent.skill` Definition.
3. Establishes or validates a pinned Artifact with caller-supplied Artifact
   identity and stable Source Binding.
4. Records local package-operation intent.
5. Stages and publishes the complete managed package directory.
6. Advances managed Source state when content changed.
7. Refreshes the owning Bundle.
8. Resolves the pinned Artifact only after discovery confirms the expected
   Definition.

Managed update is same-binding replacement. General Artifact rebinding and
move are unsupported.

A managed package digest is a local authoring and replay intent over the
managed package files. It is not a portable package digest, closure digest, or
transport digest.

Source publication and SQLite metadata are separate transactions. A failure
can leave an operation pending; retry uses the same Artifact ID and package
intent.

### Deletion

Managed deletion is typed:

1. Resolve the managed pinned Artifact and its Bundle-owned managed Source.
2. Remove the managed package directory through the managed Source.
3. Refresh the Bundle and confirm the Artifact becomes missing.
4. Purge local Artifact metadata.

External or library Skill removal normally unadopts or purges local metadata
with suppression according to Bundle policy. It does not delete externally
owned files. Built-in Skills are protected and cannot be deleted through
ordinary Skill APIs.

## Runtime projection and selection

Runtime resolution begins with `ArtifactRef` and resolves the owning Collection
before choosing Bundle or Workspace policy.

A runtime Skill projection verifies:

- Artifact and Collection ownership.
- Collection and Artifact enablement.
- Current Catalog and matching current Observation.
- Current `agent.skill` Definition compatibility.
- Source revision and source generation.
- Exact raw `SKILL.md` source-content digest.
- Trusted native local-path capability before package-path handoff.

The result is an ephemeral Agent Skills runtime definition and native package
location. Runtime registration, sessions, prompt rendering, resources, script
execution, and sandboxing are owned by Agent Skills and its runtime provider.

### Collision policy

The Agent Skills runtime has global name semantics. Equal Skill names from
more than one Artifact are fail-closed:

- No candidate wins by map order, source order, path order, or registration
  order.
- Colliding names are removed from desired runtime registration.
- Collection reconciliation reports conflict rather than choosing a winner.
- Artifact-backed allow-lists reject ambiguous requested names.

### Durable selections

Assistant, conversation, inference, and Workspace integrations must persist
Artifact-backed selections. A persistent Skill selection may include local
selection behavior, such as active or instruction use, but must not persist a
native path, runtime provider registration ID, legacy Skill ID, or Bundle slug
as local identity.

## Protected built-in Skills

Built-ins are application-owned embedded packages. The application registry
owns:

- Protected Root and Source IDs.
- Static Skill Bundle Collection IDs.
- Static Skill Artifact IDs.
- Embedded package locations.
- Artifact-to-package-member locator registration.
- Installation defaults.

Embedded `collection.json` owns portable Collection semantics and member
references. The registry does not duplicate Skill instructions, descriptions,
frontmatter, or native package bytes.

Trusted startup hydration:

1. Validates static IDs, registry declarations, package layout, and Collection
   documents.
2. Calculates raw member `SKILL.md` digests from embedded package bytes.
3. Canonicalizes the Collection document.
4. Builds a desired hydration fingerprint covering topology, registrations,
   canonical documents, member digests, and every package file.
5. Resets stale protected topology before rebuilding it.
6. Publishes complete package directories into the shared protected managed
   Source.
7. Ensures static Bundle topology and pinned static Artifacts.
8. Refreshes scoped built-in Catalogs and commits hydration state on success.

One shared protected Source can hold multiple built-in Bundle package roots.
Each Bundle Attachment scopes discovery to its own root.

A current hydration fingerprint avoids unnecessary package publication and
Catalog replacement. A stale fingerprint causes protected topology reset and
rebuild rather than portable migration. There is no ordinary user preference
mutation path for protected Bundle or Artifact state.

Undeclared active Bundles in the protected Root, or undeclared Artifacts in a
canonical built-in Bundle, are explicit conflicts. They are not silently
adopted, renamed, or migrated.

### Built-in semantic references

Portable application-level references to built-ins use semantic Bundle and
Skill names rather than local Root IDs, Artifact IDs, paths, or registry
metadata. Local application state may resolve such a reference to an
`ArtifactRef`, but that local resolution is not portable identity.

## Legacy Skill Store cutover

Artifact Store is the active Skill authority.

Current policy is reset and recreate, not migration:

- `internal/skillstore` is reference-only code.
- Normal application startup does not initialize it as active state.
- No runtime fallback, dual write, or legacy-ID resolver exists.
- No offline importer is implemented.
- No automatic startup migration is supported.
- New durable selections use Artifact-backed references.

A released compatibility migration, if ever needed, requires a separate design
and must remain outside normal application startup.

## Security and API boundaries

Skill Bundle APIs own:

- Bundle lifecycle.
- Attachment role validation.
- Discovery roots and expected member-digest policy.
- External, library, managed, and built-in policy.
- Managed Skill create, update, and typed deletion.
- Typed Skill adoption, pinning, suppression, and purge.
- Runtime projection for `skill.bundle` ownership.

Artifact administration owns Source lifecycle and managed package mechanics.
It does not expose private Source configuration through normal Source summary
APIs.

Skill runtime handoff uses trusted Source capabilities only after current
Artifact and Catalog validation. Skill discovery does not execute scripts or
other discovered content. Script execution remains disabled unless explicitly
configured at runtime composition.

## Current-scope requirements

| ID       | Requirement                                                                                                                                        | Status                              |
| -------- | -------------------------------------------------------------------------------------------------------------------------------------------------- | ----------------------------------- |
| `SK-C01` | Represent installed Skill Bundles as local `skill.bundle` Collections                                                                              | Implemented                         |
| `SK-C02` | Represent every current Skill as an `agent.skill` Artifact                                                                                         | Implemented                         |
| `SK-C03` | Treat `SKILL.md` as Skill semantic authority and derive one shared Definition                                                                      | Implemented                         |
| `SK-C04` | Use the shared Skill decoder for Bundle and Workspace discovery                                                                                    | Implemented                         |
| `SK-C05` | Support linked external and library filesystem Skills without package-tree copying                                                                 | Implemented                         |
| `SK-C06` | Support Bundle-owned managed Skill creation, same-binding update, retry, and deletion                                                              | Implemented                         |
| `SK-C07` | Preserve local Skill state during refresh and compatible source update                                                                             | Implemented                         |
| `SK-C08` | Use `ArtifactRef` for durable Skill selection                                                                                                      | Implemented                         |
| `SK-C09` | Project current verified Skills into ephemeral Agent Skills runtime definitions                                                                    | Implemented                         |
| `SK-C10` | Fail closed on same-name runtime collisions and ambiguous allow-lists                                                                              | Implemented                         |
| `SK-C11` | Keep installed Bundle ownership separate from Workspace Skill ownership                                                                            | Implemented                         |
| `SK-C12` | Hydrate built-in packages into protected managed storage with static local topology                                                                | Implemented                         |
| `SK-C13` | Validate built-in `collection.json` through the versioned schema registry                                                                          | Implemented                         |
| `SK-C14` | Keep local IDs, paths, enablement, Source configuration, runtime state, and local operation data out of Skill Definitions and Collection documents | Implemented                         |
| `SK-C15` | Reject imported Skill Attachment roles and all current transfer behavior                                                                           | Intentional current-scope exclusion |
| `SK-C16` | Keep the legacy standalone Skill Store inactive in normal application composition                                                                  | Implemented policy                  |
| `SK-C17` | Keep scripts disabled by default unless application composition explicitly enables them                                                            | Implemented policy                  |
| `SK-C18` | Do not support direct Artifact move or general Source Binding rebinding                                                                            | Intentional current-scope exclusion |

## Current implementation status

### Implemented

- `skill.bundle` Collections and `agent.skill` Artifacts.
- Shared `SKILL.md` parsing and Definition validation.
- External and library filesystem Skill discovery.
- Managed Skill creation from raw Markdown, structured documents, or complete
  package files.
- Same-binding managed replacement, refresh, retry signaling, and typed
  deletion.
- Artifact-backed runtime routing for Bundle and Workspace ownership.
- Raw `SKILL.md` digest and Source-generation verification before native
  package-path handoff.
- Fail-closed runtime collision and allow-list behavior.
- Protected built-in hydration, static IDs, scoped package roots, and stale
  topology reset.
- Artifact-backed consumer integration and inactive legacy Skill Store.

### Intentionally absent

- Standalone and Bundle transfer.
- Archive and closure support.
- Imported role and provenance.
- Public generic Skill import/export APIs.
- Legacy data migration.

## Implementation map

| Responsibility                                                               | High-level implementation area    |
| ---------------------------------------------------------------------------- | --------------------------------- |
| Shared Skill decoder, Definition projection, and native package verification | `internal/skill/artifact`         |
| Skill Bundle lifecycle, role policy, discovery, and managed authoring        | `internal/skill/bundle`           |
| Workspace-owned Skill projection                                             | `internal/skill/workspaceadapter` |
| Artifact-backed runtime router, collision handling, sessions, and tools      | `internal/skill/runtime`          |
| Built-in Skill registry hydration and installer policy                       | `internal/skill/artifactbuiltin`  |
| Embedded built-in package resources and topology registry                    | `internal/builtin`                |
| Skill Collection schema and canonical model                                  | `internal/builtin/schema`         |
| Application composition and Wails-facing Skill boundary                      | `cmd/agentgo`                     |

The implementation map is a reading guide. It does not replace the ownership
and invariant rules in this HLD.

## Change gates

This HLD must be amended before adding:

- Standalone Skill import or export.
- Skill Bundle import or export.
- Closure generation, package manifests, archive support, or package CAS.
- Imported Attachment roles or portable provenance.
- Offline fallback for linked Skills.
- Cross-Root Skill transfer.
- Direct Artifact move or general Source Binding rebinding.
- A user preference mutation surface for protected built-ins.
- Legacy migration or compatibility adapters.
- Runtime name precedence in place of fail-closed collision behavior.

A future transfer design must define package ownership, closure contents,
identity allocation, materialization, provenance, safety policy, and runtime
semantics before implementation begins.

## Prior-requirement transition

The prior Skill requirement family is superseded as follows when this document
is adopted:

| Prior requirement family  | Disposition                                                                                                        |
| ------------------------- | ------------------------------------------------------------------------------------------------------------------ |
| `SK-R01` through `SK-R09` | Retained by `SK-C01` through `SK-C09`                                                                              |
| `SK-R10`                  | Narrowed to current schema-backed built-in Collection document validation in `SK-C13`; it is not a transfer format |
| `SK-R11` through `SK-R13` | Retired from current scope because transfer and closures are unsupported                                           |
| `SK-R14` and `SK-R15`     | Retained by `SK-C12` and `SK-C13`                                                                                  |
| `SK-R16`                  | Retained by `SK-C10`                                                                                               |
| `SK-R17` through `SK-R19` | Retained by `SK-C07`, `SK-C09`, `SK-C11`, and `SK-C14`                                                             |
| `SK-R20`                  | Retained by `SK-C16`                                                                                               |

The old IDs must not be reused with changed meanings. No prior transfer plan or
legacy migration option remains an active requirement under this HLD.
