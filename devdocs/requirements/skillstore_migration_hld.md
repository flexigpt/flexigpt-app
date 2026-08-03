# Skill Collections on Artifact Store

## 1. Purpose

This document defines the target Skill management feature on Artifact Store.

It specifies:

- How Skill bundles and Skills are represented.
- How Skill bundles and individual Skills are shared as portable entities.
- How managed, built-in, external, and imported Skill sources behave.
- Which metadata is portable and which is local.
- How relative Skill package references and multi-file content closures behave.
- How typed Skill selections reach Agent Skills runtime.
- How native Skill bundle and individual Skill import and export work.
- Current implementation status.
- Ordered work needed to replace the standalone Skill Store.

This document remains a migration target. The current Artifact Store and
Workspace core boundary is complete independently of this migration.

This migration includes the complete backend reference cut:

- `skill.bundle` Collection and `agent.skill` Artifact ownership.
- Artifact-backed Agent Skills runtime resolution and registration.
- Assistant preset Skill selections.
- Conversation Skill selections and Workspace selections.
- Inference Skill-session allow-lists, prompt hydration, and runtime tool use.

The backend source of truth is Artifact Store. New durable Skill selections use
`ArtifactRef`; runtime ownership is derived from the Artifact's current
Collection membership.

Built-ins use one protected static system Root. The Root, managed Source,
Skill Bundle Collections, and Skill Artifacts use committed UUIDv7 values from
app-only embedded built-in metadata. The built-in registry is not a portable
Skill package and is not owned by Artifact Store. Application composition
provides the protected-Root policy and invokes the built-in installer through
narrow Artifact Store and Skill Bundle ports. No user Root receives copied
built-in Sources, Collections, or Artifacts.

The standalone Skill Store remains temporarily in the source tree as a
reference-only implementation for migration verification. It is not active
application state. Normal startup never imports, interprets, migrates,
registers, writes, or resolves legacy Skill Store records. A future one-time
importer, if product requires one, must live outside normal application startup
and create new Artifact Store identities. It must not restore legacy Skill
Store code, identity types, path handling, or a dual-write compatibility path.

## 2. Feature intent

FlexiGPT must support:

- Sharing Skill bundles and individual Skills across teams and the internet.
- Importing and exporting Skills without exposing local application metadata.
- Application-provided built-in Skills.
- User-authored managed Skills.
- User-selected external filesystem Skills.
- Workspace-discovered Skills.
- Future imported and library Skills.
- Normal Agent Skills sessions, rendering, resources, and scripts.

These origins must share portable Skill semantics and runtime behavior without
sharing incompatible durable identity models.

A Skill bundle and an individual Skill are both portable entities. The local
Collection and Artifact Record remain local management entities and are not
exported verbatim.

## 3. Outcomes

The feature must:

- Represent every Skill bundle as a `skill.bundle` Collection.
- Represent every installed or Workspace Skill as an `agent.skill` Artifact.
- Treat `SKILL.md` as the portable Skill authority.
- Use `ArtifactRef` for persistent Skill selection.
- Keep `ArtifactID` independent of Skill Collection identity so direct movement
  can be added later without making it an initial requirement.
- Keep Source paths out of Skill records and portable definitions.
- Use normal Collection and Artifact enablement for built-ins.
- Preserve installed and Workspace provenance without encoded identity branches.
- Project eligible filesystem Skills into ephemeral Agent Skills `SkillDef` values.
- Export one Skill as a native directory or self-contained archive.
- Export one Skill bundle as a portable manifest plus relative Skill packages.
- Import a Skill or Skill bundle from a local path, archive, or policy-approved URI.
- Preserve multi-file Skill resources and scripts through a portable content closure.

## 4. Skill domain model

### 4.1 Skill bundle

A Skill bundle is:

    Collection kind: skill.bundle

Its stable local reference is:

    CollectionRef{RootID, CollectionID}

Collection data may contain:

- Display and organization settings.
- Local provenance.
- Local policy references.
- Presentation ordering.

A display alias or logical key may exist, but it is not the local identity.

The local Collection contains enablement, attachments, revisions, and local
policy. Its shareable representation is a Portable Skill Bundle Definition.

### 4.2 Portable Skill Bundle Definition

A Portable Skill Bundle Definition uses the `skill.bundle` Collection schema.

It contains:

- Logical bundle name and version.
- Portable display metadata.
- Portable labels.
- Ordered or unordered Skill member references.
- Optional expected member digests.
- Domain-approved portable bundle settings.

A Skill member may be:

- Embedded in the bundle document.
- Referenced by a relative Skill package locator.
- Referenced by a policy-approved external URI.

A native JSON representation may look conceptually like:

    {
      "kind": "skill.bundle",
      "schemaVersion": "...",
      "name": "...",
      "skills": [
        {
          "locator": "skills/example/SKILL.md",
          "digest": "sha256:..."
        }
      ]
    }

The exact wire format is owned by the Skill domain codec. The canonical
Portable Collection Definition remains format-independent.

The bundle definition must not contain local Collection IDs, Artifact IDs,
absolute paths, local enablement, runtime state, or credential references.

### 4.3 Skill Artifact

A Skill is:

    Artifact kind: agent.skill

Its stable reference is:

    ArtifactRef{RootID, ArtifactID}

Its current placement is:

    ArtifactAddress{
      RootID,
      CollectionID,
      ArtifactID,
      Kind: "agent.skill"
    }

Direct movement is deferred. The model does not derive `ArtifactID` from
`CollectionID`, leaving a future move operation possible.

Until direct movement exists, a caller exports or copies the Skill, adds it to
the target bundle, and unadopts or deletes the original. The new Skill receives
a new local `ArtifactID`, so persistent references and local-only settings must
be updated explicitly.

### 4.4 Portable `agent.skill` definition

The canonical definition derives from `SKILL.md` and contains portable semantics:

- Name.
- Display name.
- Description.
- Insert behavior.
- Arguments.
- Source tags.
- Markdown body.
- Portable raw frontmatter.

It must not contain:

- Root, Collection, Source, or Artifact IDs.
- Absolute paths.
- Local enablement.
- Local user tags.
- Runtime sessions or runtime index state.
- Credential references or secret values.

Parser warnings belong to occurrence diagnostics. Source-content digest belongs
to the occurrence.

An `agent.skill` Artifact is normally a multi-file package:

    <skill-name>/
      SKILL.md
      resources/
      scripts/

`SKILL.md` is the entry point. A portable content closure identifies the
domain-approved files required for export and integrity verification.

An individual Skill export may be:

- The native Skill directory.
- A self-contained Skill archive.
- A generic Artifact package containing the native Skill directory.

### 4.5 Local metadata placement

| Concern                                | Owner                                |
| -------------------------------------- | ------------------------------------ |
| Bundle display metadata and enablement | `skill.bundle` Collection            |
| Skill enablement                       | Artifact Record                      |
| Local display override                 | Artifact data                        |
| Local user tags                        | Artifact data                        |
| Source tags                            | `agent.skill` definition             |
| Filesystem or embedded transport       | Source registration                  |
| Absolute source location               | Private Source configuration         |
| Built-in provenance                    | Source and attachment provenance     |
| Presence and source validity           | Occurrence and Artifact state        |
| Parser warnings                        | Occurrence diagnostics               |
| Runtime resource index                 | Agent Skills runtime                 |
| Script execution policy                | Runtime host                         |
| Credential value                       | Secret system outside Artifact Store |

## 5. Skill Source policy

### 5.1 Attachment roles

`skill.bundle` policy supports:

| Role       | Meaning                                  |
| ---------- | ---------------------------------------- |
| `managed`  | Application-authored Skill packages      |
| `builtin`  | Application-provided Skill packages      |
| `external` | User-selected filesystem packages        |
| `imported` | Imported package content                 |
| `library`  | Reusable local or shared library content |

Artifact Store owns attachment structure. Skill policy owns scope, allowed
roles, adoption, and collision rules.

### 5.2 Managed Skills

Managed authoring must:

- Select or create a Root-scoped managed Source.
- Allocate an `ArtifactID`.
- Stage a package containing a valid `SKILL.md`.
- Validate it with the common `agent.skill` decoder.
- Publish the package atomically within the managed Source where possible.
- Refresh the target Collection.
- Adopt the resulting occurrence using the preallocated Artifact ID.
- Return `ArtifactAddress`.
- Use the caller-supplied `ArtifactID` as the sole durable create replay
  identity.
- Store package SHA-256 only as immutable package-content intent.
- Read Source revision and generation immediately before publication. They are
  request-time concurrency preconditions and are never stored in Artifact data.

A possible internal layout is:

    managed-source/items/<artifact-id>/<skill-name>/SKILL.md

The layout is private and must not appear in portable definitions or persistent selections.

### 5.3 External filesystem Skills

Adding an external Skill must:

- Create or reuse a filesystem Source in the same Root.
- Attach it to the target Skill Collection.
- Discover or pin a relative `SKILL.md` Source Binding.
- Adopt it into an Artifact Record.

The absolute path remains Source configuration. It is not stored in the Skill Artifact.

### 5.4 Built-in Skills

Built-in packages use a normal managed Source and normal Collection and
Artifact state inside the one protected static system Root.

The first implementation hydrates embedded packages into a managed Source
because Agent Skills currently requires a trusted native filesystem package
path for full resource and script behavior.

The app-only registry contains static local IDs, local installation defaults,
portable Collection payload references, and mappings from static Artifact IDs
to portable Collection members. It must not contain raw package bytes,
portable Collection metadata, duplicated Skill metadata, or absolute paths.

Each embedded portable Skill Collection package contains `collection.json` and
relative Skill package directories. The shared managed Source materializes
those packages under their portable package roots. Each local Skill Collection
mounts the shared Source through a Collection-owned attachment scope so one
Collection discovers only its own portable package subtree.

Shareable package data contains only logical bundle and Skill names, package
files, `SKILL.md`, optional portable versions, and externally supplied package
SHA-256 values. It contains no app UUIDs, source locations, revisions,
timestamps, enablement, idempotency keys, or installation metadata.

Built-in installer updates are explicit trusted installation operations. Normal
Skill Bundle creation, refresh, mutation, retirement, and purge APIs cannot
mutate the protected built-in topology.

Portable Skill metadata currently duplicated in `skills.json` must move into
the corresponding `SKILL.md`. A reduced seed descriptor may remain for
an application index of built-in bundle manifests, but it must not duplicate
portable Skill content.

Ordinary managed-package publication and removal also reject the protected Root.
The built-in installer receives a separate protected managed-package callback
from application composition. That callback requires installer capability and
the configured protected Root. A future package upgrade workflow must use the
same protected callback and must remain separate from ordinary managed Skill
creation, update, and deletion flows.

### 5.5 Imported Skills

A future import operation creates:

- A local package or managed Source.
- One or more local Skill Collections.
- Newly assigned Artifact IDs.

Portable package identifiers never replace local Artifact identity.

Import may target:

- One native Skill directory or archive.
- One Portable Skill Bundle Definition.
- One self-contained Skill bundle package.
- One policy-approved external URI.

Local filesystem paths supplied to import become private Source configuration.
They are not copied into portable Skill or bundle definitions.

### 5.6 Skill export layout

A self-contained Skill bundle export may use:

    skill-bundle.json
    skills/
      <skill-name>/
        SKILL.md
        resources/
        scripts/

The exporter must:

- Snapshot or capture every selected Skill package.
- Include the domain-approved resource and script closure.
- Rewrite vendored references to relative package locators.
- Preserve portable Skill names, frontmatter, arguments, tags, and Markdown.
- Exclude local IDs, enablement, user tags, Source paths, runtime warnings,
  runtime indexes, and secret data.
- Report Skills that cannot be exported because their Source is unavailable,
  stale, denied, or unsupported.

## 6. Agent Skills runtime boundary

Agent Skills runtime continues to own:

- Provider registration.
- Runtime Skill registration and removal.
- Sessions and active Skill state.
- Skill prompts.
- Rendering and argument application.
- Resource indexing and reading.
- Script execution.

Artifact Store and Skill policy own durable identity and eligibility, not runtime lifecycle.

### 6.1 Runtime projection

For a selected filesystem-backed Skill:

    ArtifactRef
      -> Artifact Record
      -> current Collection membership
      -> Collection and Artifact enablement
      -> current valid occurrence
      -> canonical definition validation
      -> Source generation or package closure verification
      -> trusted local-path resolution
      -> SKILL.md source-content digest verification
      -> containing package directory
      -> ephemeral Agent Skills SkillDef

`SkillDef{Type, Name, Location}` is a process-local runtime handle. It must not
be persisted in Artifact Store, assistant presets, or conversations.

### 6.2 Runtime selections

Persistent selection uses:

    SkillSelection{
      Artifact: ArtifactRef,
      PreLoadAsActive,
      UseAsInstructions
    }

If a transport needs a string, it may encode a temporary value such as:

    artifact-skill/<root-id>/<artifact-id>

The string must be decoded at the boundary and never become the durable model.

### 6.3 Name collisions

Skill selection must apply an explicit scope or precedence policy.

It must:

- Reject equal-precedence same-name collisions with diagnostics.
- Apply documented Collection precedence only where a product requires it.
- Never choose by map order, path order, origin string, or legacy bundle identity.

## 7. Functional requirements

| ID       | Requirement                                                                         | Priority |
| -------- | ----------------------------------------------------------------------------------- | -------- |
| `SK-F01` | Create and manage `skill.bundle` Collections.                                       | Core     |
| `SK-F02` | Decode and validate a shared `agent.skill` definition from `SKILL.md`.              | Core     |
| `SK-F03` | Create managed Skills through a managed Source.                                     | Core     |
| `SK-F04` | Add external Skills through Source registration and relative occurrence linkage.    | Core     |
| `SK-F05` | Adopt valid Skill occurrences explicitly.                                           | Core     |
| `SK-F06` | Pin expected external Skill Source Bindings.                                        | Core     |
| `SK-F07` | Preserve local Skill metadata during source refresh.                                | Core     |
| `SK-F08` | Avoid Collection-derived Skill identity so future direct movement remains possible. | Deferred |
| `SK-F09` | Use normal Collection and Artifact enablement for built-ins.                        | Core     |
| `SK-F10` | Bootstrap one protected built-in topology through static registry UUIDv7 IDs.       | Core     |
| `SK-F11` | Keep local user tags separate from Source tags.                                     | Core     |
| `SK-F12` | Persist Skill selections as `ArtifactRef`.                                          | Core     |
| `SK-F13` | Project eligible filesystem Skills into ephemeral Agent Skills definitions.         | Core     |
| `SK-F14` | Retain session, prompt, render, resource, and script behavior.                      | Core     |
| `SK-F15` | Verify current definition, Source, and `SKILL.md` content before runtime use.       | Core     |
| `SK-F16` | Return structured diagnostics for stale, unavailable, denied, or ambiguous Skills.  | Core     |
| `SK-F17` | Keep Workspace and installed persistence ownership separate.                        | Core     |
| `SK-F18` | Update assistant presets and conversations to use Artifact-based selections.        | Core     |
| `SK-F19` | Keep runtime synchronization rebuildable and separate from metadata transactions.   | Core     |
| `SK-F20` | Define and validate a portable `skill.bundle` Collection schema.                    | Core     |
| `SK-F21` | Import and export individual native Skill packages.                                 | Planned  |
| `SK-F22` | Import and export Skill bundles with relative package closure.                      | Planned  |
| `SK-F23` | Support embedded, relative, and policy-approved URI Skill members.                  | Planned  |
| `SK-F24` | Preserve multi-file resource and script content in portable exports.                | Planned  |

## 8. Skill API direction

Collection operations include:

    CreateSkillBundle(rootID, request) -> CollectionRef
    GetSkillBundle(collectionRef)
    ListSkillBundles(rootID)
    UpdateSkillBundle(collectionRef, request)
    RetireSkillBundle(collectionRef)

Source operations include:

    AttachSkillSource(collectionRef, request)
    UpdateSkillSourceAttachment(collectionRef, sourceID, request)
    DetachSkillSource(collectionRef, sourceID, request)

Skill operations include:

    CreateSkill(collectionRef, request) -> ArtifactAddress
    AddExternalSkill(collectionRef, request) -> ArtifactAddress
    AdoptSkillOccurrence(collectionRef, occurrenceKey) -> ArtifactAddress
    PinSkillSourceBinding(collectionRef, request) -> ArtifactAddress
    GetSkill(artifactRef)
    ListSkills(collectionRef)
    UpdateSkillLocalData(artifactRef, request)
    SetSkillEnabled(artifactRef, request)
    UnadoptSkill(artifactRef, request)

Planned transfer operations include:

    ImportSkill(collectionRef, input) -> ArtifactAddress
    ExportSkill(artifactRef, options) -> PortableOutput
    ImportSkillBundle(rootID, input) -> CollectionRef
    ExportSkillBundle(collectionRef, options) -> PortableOutput

Runtime operations accept typed Artifact-based Skill selections.

Direct `MoveSkill` is deferred. Until implemented, callers add or import the
portable Skill into the target Collection and then unadopt or delete the
original.

Bundle IDs, Skill slugs, and provider identity strings may remain display or
transport values only where explicitly required. They are not durable identity.

### 8.1 Backend consumer completion

The backend migration includes every durable backend consumer of a Skill
selection:

| Consumer         | Durable representation after migration                                                                |
| ---------------- | ----------------------------------------------------------------------------------------------------- |
| Skill runtime    | `artifact.ArtifactRef`                                                                                |
| Assistant preset | `ArtifactSkillSelection{Artifact: ArtifactRef}`                                                       |
| Conversation     | `EnabledSkillRefs []ArtifactRef`, `ActiveSkillRefs []ArtifactRef`, and Workspace selection references |
| Inference        | Artifact-backed session allow-list and Artifact-backed Skill Runtime requests                         |
| Workspace        | `ArtifactRef` for Workspace Skills and Context                                                        |

The Agent Skills runtime remains process-local. It receives a verified
ephemeral `SkillDef` only after the Artifact router resolves the Artifact and
its owning Collection feature adapter.

The migration does not preserve legacy `BundleID`, `SkillSlug`, `SkillID`, or
`Location` as a parallel runtime identity. Those values can exist only in
legacy reference data used by an explicit offline migration or diagnostic
tool.

## 9. Current implementation status

| Capability                         | Status                              | Current implementation                                                                                                                                                                                                                                                      |
| ---------------------------------- | ----------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `SKILL.md` parsing and validation  | Present                             | `agentskills-go` is used by managed creation and Workspace discovery.                                                                                                                                                                                                       |
| Managed Skill document creation    | Present                             | `skillbundle.CreateManagedSkill` uses the shared `skillartifact` decoder, pins, publishes through the managed Source capability, refreshes, and resolves an Artifact. Managed deletion preflights revision state and reports retry-required source-side partial completion. |
| Structured managed authoring       | Present                             | Callers may supply an Agent Skills `SkillDocument`; serialization is delegated to `agentskills-go`. Raw `SKILL.md` and complete package inputs remain supported.                                                                                                            |
| Managed Artifact Source            | Present at platform layer           | Artifact Store owns MapStore-backed staged writable Source publication and removal. Skill Bundle code must delegate to this capability rather than create package files itself.                                                                                             |
| Skill bundle Collection            | Present                             | `skill.bundle` uses normal Artifact Store Collections, attachments, catalog publication, Artifacts, enablement, and revisions.                                                                                                                                              |
| Portable Skill Bundle Definition   | Present, linked-manifest codec      | `skill.bundle.v1` provides canonical shareable JSON over `CollectionDefinition` with integrity-pinned raw `SKILL.md` entrypoints. It is not yet a self-contained package export, persisted transfer record, importer, or closure assembler.                                 |
| Shared `agent.skill` definition    | Present                             | `skillartifact` owns shared `SKILL.md` parsing and canonical definition validation for Workspace and Skill Bundle discovery.                                                                                                                                                |
| External Source linkage            | Present                             | External filesystem content is attached as an Artifact Store Source and represented by a relative Source Binding.                                                                                                                                                           |
| Built-in hydration                 | Partial                             | Built-ins currently bootstrap one managed Source, Collections, and Artifacts, but portable Collection package loading, attachment-scoped discovery, and non-duplicated registry metadata remain required.                                                                   |
| Built-in package upgrades          | Deferred                            | Bootstrap is idempotent add-missing setup, not an in-place package upgrade protocol. Changed built-in package content requires an explicit versioned upgrade workflow.                                                                                                      |
| Built-in startup convergence       | Partial                             | Startup convergence exists for the current limited registry, but a shared Source requires Collection-scoped discovery and a final stale-catalog convergence pass after all package publication.                                                                             |
| Built-in normal Collection state   | Present                             | Built-in enablement uses normal Collection and Artifact enablement.                                                                                                                                                                                                         |
| Agent Skills runtime               | Present                             | Artifact router resolution, registration, sessions, prompts, rendering, resources, and scripts are Artifact-backed.                                                                                                                                                         |
| Managed operation recovery         | Present                             | Managed Skill retries use the caller-supplied Artifact ID and immutable package SHA-256. Source revision and generation are read immediately before publication and remain request-time concurrency preconditions only.                                                     |
| Runtime source verification        | Present                             | Runtime handoff verifies current catalog occurrence, Source generation, and raw `SKILL.md` content digest through Source snapshots, then delegates native package parsing, resource access, sandboxing, and script execution to `agentskills-go`.                           |
| Script execution policy            | Present and explicit                | Artifact-backed runtime defaults to scripts disabled. Explicit application composition may enable scripts, while `agentskills-go` and `llmtools-go` retain all execution and sandbox policy.                                                                                |
| Artifact-backed backend references | Present                             | Assistant presets, conversations, inference, Workspace, and Skill Runtime use Artifact-backed Skill references.                                                                                                                                                             |
| Runtime synchronization state      | Present and derived only            | Artifact Store is durable authority. Before reconciliation, `SkillRuntime` re-reads every process-local Collection partition and drops unavailable partitions. Its maps are only the inventory required to remove and reindex Agent Skills provider registrations.          |
| Runtime synchronization trigger    | Present and demand-driven           | A selected Artifact is resolved from Artifact Store and reconciled into Agent Skills runtime only when runtime work needs it. No observer or feature-owned membership cache exists.                                                                                         |
| External Skill attachment route    | Present                             | A filesystem Source is created through Artifact Store administration, then attached through the typed Skill Bundle boundary and refreshed into `agent.skill` Artifacts.                                                                                                     |
| Managed create replay              | Present                             | Managed creation uses the caller-supplied Artifact ID as replay identity and package SHA-256 as immutable package-content intent. There is no hidden idempotency key or durable source-operation state.                                                                     |
| Legacy Skill Store                 | Reference-only, inactive            | `internal/skillstore` remains temporarily for source comparison only. `cmd/agentgo` does not initialize it, and no normal runtime, selection, inference, or persistence path may call it.                                                                                   |
| Same-name policy                   | Partial                             | Aggregate listing marks simultaneously eligible same-name Skills unavailable, and runtime resolution withholds ambiguous durable references. A unified migrated-Collection precedence policy remains deferred.                                                              |
| Assistant preset integration       | Present                             | Assistant preset validation resolves `ArtifactSkillSelection` through the Artifact-backed Skill Runtime.                                                                                                                                                                    |
| Conversation integration           | Present                             | Conversations persist Artifact-backed Skill references and Workspace selections.                                                                                                                                                                                            |
| Inference integration              | Present                             | Inference resolves Artifact-backed Skill allow-lists, Workspace usage, sessions, prompts, and Skill tools through Skill Runtime.                                                                                                                                            |
| Legacy-data migration              | Reset gate present, importer absent | Assistant presets and conversations use Artifact-reference-specific schema versions. Existing standalone Skill Store and older conversation data still require an explicit reset or offline importer.                                                                       |
| Individual Skill import and export | Not present                         | There is no native Skill transfer service or package closure builder.                                                                                                                                                                                                       |
| Skill bundle import and export     | Not present                         | Canonical linked `skill.bundle.v1` manifest JSON exists, but there is no package-closure builder, member acquisition resolver, importer, exporter, or deterministic archive workflow.                                                                                       |

## 10. Breaking migration plan

### 10.1 Cutover contract

Artifact Store replaces the standalone Skill Store as the active backend
source of truth. The standalone Skill Store is retained temporarily only as a
legacy reference store and possible offline migration input.

- A standalone installed Skill bundle becomes a `skill.bundle` Collection.
- A standalone installed Skill becomes an `agent.skill` Artifact.
- A caller-supplied Artifact ID is managed-Skill create replay identity.
- No hidden local Artifact idempotency key or Source-generation cache exists.
- Workspace-discovered Skills remain `agent.skill` Artifacts in
  `workspace.collection`; they are not moved into Skill bundle Collections.
- All new backend persistence uses `ArtifactRef`.
- Assistant preset, conversation, inference, and runtime Skill references are
  Artifact-backed as part of this migration.
- Legacy `BundleID`, `SkillSlug`, `SkillID`, `SkillType`, and `Location` are
  not accepted as durable runtime identity after cutover.
- Normal application startup must not initialize `internal/skillstore` as an
  active store, register its Skills, resolve runtime selections from it, or
  write to it.
- A dedicated offline migration or diagnostic command may read legacy
  Skill Store data without changing it. Such a command must be explicit,
  read-only for legacy input, and must not run during normal startup.
- No dual write or active legacy-reference adapter is permitted.
- Runtime origin and Collection ownership are projections resolved from the
  Artifact, not encoded in a durable Skill reference.

This is an API and persistence breaking change. Product must choose one data
policy before release:

- Version or reset legacy Skill-dependent stores; or
- Run a one-time offline importer that creates new local Collections, Sources,
  and Artifacts, rewrites dependent assistant preset and conversation
  references, and is removed from the normal application path afterward.

### 10.1.1 Legacy reference-store policy

The legacy Skill Store is permitted only under these constraints:

- It is not opened by normal application startup.
- It is never mutated after Artifact Store cutover.
- It does not register runtime Skills.
- It does not resolve assistant preset, conversation, inference, or Workspace
  Skill references.
- It is read only by an explicit migration, diagnostics, or test utility.
- The migration utility creates new Artifact Store identities. It does not
  preserve legacy IDs as Artifact IDs or use legacy locations as portable data.

### 10.2 Required target boundaries

Artifact Store remains the owner of Roots, Sources, Collections, Artifacts,
catalogs, managed content publication, and generic lifecycle invariants.

The new Skill bundle feature owns:

- `skill.bundle` Collection validation and views.
- Skill-specific Collection data and attachment-role policy.
- Skill discovery planning, refresh, adoption, pinning, suppression, and local
  Artifact data.
- Built-in bundle bootstrap.
- Managed and external Skill authoring workflows.
- Skill bundle listing and management APIs.

`skillartifact` remains the shared owner of `agent.skill` portable definition
decoding and validation. Workspace and Skill bundle features must use the same
decoder and definition contract.

Agent Skills runtime remains process-local. It receives only a verified,
ephemeral filesystem `SkillDef`; it must not persist runtime locations or use
legacy Skill Store identity.

### 10.3 Required migration decisions

Before implementation, define:

- The legacy data disposition policy.
- The initial same-name runtime collision policy. The default is fail-closed:
  equal-priority same-name Skills are not registered.
- The built-in Source strategy. Built-ins must become normal Artifact Store
  Sources, Collections, and Artifacts with normal enablement.
- Managed package update semantics for content edits and built-in upgrades.
- Executable-file semantics for managed Skill scripts.
- External package root rules and the expected-name behavior for `SKILL.md`.

### 10.4 Ordered implementation checklist

- [x] Register the shared `agent.skill` decoder once in Artifact Store
      composition.
- [x] Add the typed `skill.bundle` feature service, API, planner, policy,
      bootstrap service, managed Source workflow, and Artifact-backed runtime
      resolver.
- [x] Use normal Collection and Artifact enablement for built-in Skill bundles.
- [x] Resolve Skill Runtime ownership from Artifact and Collection membership.
- [x] Replace backend assistant preset Skill selections with
      `ArtifactSkillSelection`.
- [x] Replace backend conversation, inference, and runtime Skill references
      with `ArtifactRef`.
- [x] Keep runtime synchronization demand-driven from Artifact Store-backed
      Artifact resolution. No Root mutation observer is retained.
- [x] Reject legacy assistant-preset and conversation schemas rather than
      silently interpreting standalone Skill Store identities.
- [x] Add retry-safe managed Skill creation using the caller-supplied
      `ArtifactID` as the sole replay identity.
- [x] Remove managed package content before purging a managed Skill Artifact.
- [x] Move Source generation confirmation and runtime package handoff out of
      Workspace and Skill Bundle adapters.
- [x] Define canonical portable `skill.bundle.v1` JSON manifests.

The following items remain required before the migration can be declared
release-complete:

- [x] Keep `internal/skillstore` reference-only. Normal startup does not open,
      mutate, register, or resolve it.
- [x] Keep Skill Bundle listing read-only. Built-ins are installed only in one
      global protected Root; existing and newly created user Roots receive no
      copied built-in Source, Collection, or Artifact.
- [x] Revalidate discovery-plan and decoder fingerprints before Skill adoption,
      portable linked-manifest generation, or runtime handoff.
- [x] Use `v1` for the inactive reference Skill Store schema label so all clean
      persistence contracts use one version vocabulary.
- [ ] Remove the reference-only standalone Skill Store package after migration
      verification is complete. Active application state already uses the clean
      `*_v1` namespaces as the breaking persistence boundary.
- [x] Verify the registered runtime version after Artifact resolution before
      treating a `SkillDef` as available.
- [x] Remove public generic Artifact purge transport. Typed feature workflows
      own source-side cleanup before using the internal Artifact purge service.
- [x] Keep Workspace and Skill Bundle synchronizers feature-owned. A Workspace
      reconciliation must never remove a `skill.bundle` runtime partition.
- [x] Use static registry Collection and Artifact IDs for built-in topology
      convergence rather than a bootstrap key or opaque Collection data.
- [x] Remove observer-driven runtime synchronization. Runtime registration is
      demand-driven from Artifact Store-backed Artifact resolution.
- [x] Fail closed for unavailable or same-name-colliding Artifact Skill
      allow-list entries rather than silently dropping them.
- [x] Use caller-supplied Artifact IDs for managed create replay and retain
      only package SHA-256 as immutable package-content intent.
- [x] Expose typed Skill Bundle Source attachment while retaining Source
      configuration ownership in Artifact Store.
- [ ] Add deterministic Skill and bundle import/export only after the portable
      Collection Definition and content-closure contracts exist.
- [x] Use a clean `*_v1` application namespace and intentionally do not carry
      standalone Skill Store, assistant-preset, or conversation data forward.

### 10.5 Deferred work

Portable Skill and Skill bundle import, export, content closure assembly,
archive handling, URI acquisition, and direct Artifact movement remain separate
work after the breaking core migration. The canonical linked manifest is a
shareable JSON descriptor, not a self-contained export. Their generic
requirements belong to Artifact Store. This document retains only the
Skill-specific package, manifest, and closure rules.

Two parity items remain before the old package can be deleted:

- Add a typed installed-Skill management projection with portable document
  metadata, local display metadata, local user tags, diagnostics, filtering,
  and pagination. Returning only raw Artifact Records does not replace the old
  management list and get behavior.
- Add an Agent Skills session allow-list capability upstream. Prompt filtering
  currently limits what is advertised, but the attached `agentskills-go`
  session API does not persist an allowed-Skill set for `skills-load`. This
  must be fixed in `agentskills-go`, not recreated as an application-side
  sandbox or parallel session catalog.

Typed attachment update and detach workflows remain the next local-management
work item. They must define role-specific cleanup behavior before exposing
detachment of a managed or built-in Source. In particular, a managed package
must not become unreachable while a pinned managed Skill Artifact still claims
source-side deletion responsibility.

Built-in package upgrades also require an explicit versioned publication and
Artifact reconciliation policy. Bootstrap must not be treated as an upgrader.

### 10.6 Next-round verification gate

The next implementation review must verify all of the following before marking
the backend migration clean:

- `cmd/agentgo` initializes Artifact Store, Workspace, Skill Bundle, and
  Artifact-backed Skill Runtime without initializing the standalone Skill Store.
- No normal runtime, assistant preset lookup, conversation resolver, or
  inference path imports or calls `internal/skillstore`.
- Assistant preset persistence uses `ArtifactSkillSelection` and validates it
  through the Artifact-backed Skill Runtime.
- Conversation persistence uses Artifact-backed enabled and active Skill refs,
  including Workspace Skill selections.
- Inference accepts only Artifact-backed Skill allow-lists and resolves them
  through the Artifact router.
- A managed Skill publication can recover deterministically after every
  failure boundary: pin, source publication, Source metadata acknowledgement,
  refresh, and final Artifact read.
- Deleting a managed Skill removes source package content and local metadata
  without allowing refresh to recreate the deleted Skill.
- Runtime handoff delegates package parsing and runtime behavior to
  `agentskills-go`; Source snapshots verify the catalogued raw `SKILL.md`
  digest once immediately before native-path handoff. Feature code must not introduce custom
  parsing, sandboxing, executable-file, or cross-platform path policy.
- MapStore owns link behavior for MapStore-managed payloads. Workspace, Skill
  Bundle, and runtime layers must not add a second link policy. Direct
  filesystem Source behavior must be defined by the selected source adapter;
  MapStore cannot protect paths it does not own.
- A Root created after startup receives no copied built-in Source, Collection,
  or Artifact. Built-in installation targets only the static protected Root,
  and listing a bundle must not mutate state.
- A decoder or discovery-plan revision change makes Skill adoption, runtime
  projection, and linked portable JSON generation return catalog stale until
  refresh succeeds.
- A pending managed Skill can rebase after another package advances the same
  managed Source and cannot remain permanently stuck on its first revision.
- Retiring a built-in Collection must not cause bootstrap, listing, or startup
  to recreate it or fail because its durable bootstrap key remains reserved.
- MapStore-managed definition files are accessed only through MapStore. Feature
  code must not recreate MapStore path, symlink, permission, or durability
  handling.
- No Root mutation observer or feature-owned Root membership cache exists.
  Runtime eligibility is resolved from Artifact Store at demand time.
- Feature synchronization tracks only Collections owned by that feature and
  does not maintain a second persistent or in-memory membership authority.
- The runtime rejects a stale registration when the same `SkillDef` has a
  different resolved Artifact version, including a changed local Artifact
  revision.
- Legacy Skill Store data can only be read through the chosen explicit
  migration or diagnostic path.
- Portable `skill.bundle` JSON remains a canonical linked manifest until
  deterministic package closure, import, export, archive, and acquisition
  contracts are implemented.

## 11. Acceptance outcomes

The Skill feature satisfies this document when:

- Creating a Skill bundle creates a `skill.bundle` Collection.
- Creating or adopting a Skill returns `ArtifactAddress`.
- Persistent selection uses `ArtifactRef`.
- Skill identity is not derived from bundle identity, without requiring direct move support.
- `SKILL.md` is portable Skill authority.
- Source kinds and paths are absent from canonical Skill definitions.
- External Skills are represented by Sources and Source Bindings.
- Built-in enablement uses normal Collection and Artifact state.
- Built-in initialization is idempotent without using seed keys as local IDs.
- Managed Skill storage is source-backed and independent of bundle placement.
- A Skill bundle has a portable, versioned Collection Definition.
- One Skill can be exported independently as a native package.
- One Skill bundle can be exported with relative Skill package references.
- Export preserves `SKILL.md`, resources, and scripts while excluding local metadata.
- Skill and bundle imports assign new local IDs and preserve portable digests and provenance.
- Runtime receives only an ephemeral, verified Agent Skills `SkillDef`.
- Sessions, prompts, rendering, resources, and scripts retain normal Agent Skills behavior.
- No MapFileStore bundle identity, overlay key, or encoded installed or Workspace identity is required by normal operation.
- Assistant presets, conversations, inference, and Skill Runtime persist and
  resolve only Artifact-backed Skill references.
- The standalone Skill Store is either removed after migration or retained only
  as an explicit read-only legacy reference store outside normal application
  execution.
