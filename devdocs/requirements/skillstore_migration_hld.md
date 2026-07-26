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
Workspace core boundary is complete independently of this migration. The
completion does not authorize a dual write, implicit import, or deletion of
the standalone Skill Store. The standalone package writer, embedded hydration,
bundle metadata, and overlay implementation remain active legacy behavior
until a dedicated one-way migration changes dependent assistant preset and
conversation references in the same release.

Workspace completion does not claim that installed Skills, built-in Skill
bundles, assistant preset Skill selections, or legacy Skill persistence have
already migrated. Until that dedicated migration begins, the standalone Skill
Store remains a separate legacy owner and must not dual-write into Artifact
Store. Its direct filesystem package behavior must not be described as part of
the Artifact Store MapStore-backed managed Source guarantee.

The Artifact Store and Workspace transition does not migrate the standalone
Skill Store. During this phase, installed Skills retain their existing
bundle-and-slug persistence identity while Workspace Skills use typed
`ArtifactRef` identity. No dual write, legacy-directory import, or automatic
user-data migration is permitted.

The existing Skill Store is implementation and migration input only. It is not
a target compatibility contract.

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

Built-in packages must use a normal Source and normal Collection and Artifact state.

The first implementation may hydrate embedded packages into a filesystem Source
because Agent Skills currently requires filesystem packages for full resource
and script behavior.

Each built-in bundle should use the same Portable Skill Bundle Definition used
for normal sharing and import. It contains:

- Stable logical URI or bootstrap key.
- Collection display metadata.
- Relative Skill package member references.
- Seed version.

The bootstrap key finds an existing local Collection but is not its `CollectionID`
and is not a persistent Skill reference.

Portable Skill metadata currently duplicated in `skills.json` must move into
the corresponding `SKILL.md`. A reduced seed descriptor may remain for
an application index of built-in bundle manifests, but it must not duplicate
portable Skill content.

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
| `SK-F10` | Bootstrap built-in Collections idempotently through nonidentity seed keys.          | Core     |
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

## 9. Current implementation status

| Capability                          | Status                        | Current implementation                                                                                                                                                                                         |
| ----------------------------------- | ----------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `SKILL.md` parsing and validation   | Present                       | `agentskills-go` is used by managed creation and Workspace discovery.                                                                                                                                          |
| Managed Skill document creation     | Present as reusable behavior  | `PutSkillArtifact` marshals, parses, and writes a valid package.                                                                                                                                               |
| Managed Artifact Source             | Present at platform layer     | Artifact Store has a MapStore-backed staged writable Source with contained private files and full package-generation coverage; standalone Skill Store adoption remains deferred.                               |
| Skill bundle Collection             | Not present                   | Bundles are MapFileStore records.                                                                                                                                                                              |
| Portable Skill Bundle Definition    | Not present                   | Current `skills.json` is application metadata rather than a portable Collection schema.                                                                                                                        |
| Shared `agent.skill` definition     | Present for Workspace         | Workspace emits and validates the shared `agent.skill` definition; installed Skills remain deferred.                                                                                                           |
| External Source linkage             | Incompatible                  | Absolute location is persisted in `Skill.Location`.                                                                                                                                                            |
| Built-in hydration                  | Present as reusable mechanism | Embedded content is copied to a filesystem directory with a digest marker.                                                                                                                                     |
| Built-in normal Collection state    | Not present                   | Enablement uses bundle and Skill overlay flags.                                                                                                                                                                |
| Agent Skills runtime                | Present                       | Registration, sessions, prompts, rendering, resources, and scripts are implemented.                                                                                                                            |
| Workspace Skill digest verification | Present at handoff            | Definition, Source generation, `SKILL.md`, and package symlink checks occur before runtime registration.                                                                                                       |
| Typed Artifact Skill refs           | Present for Workspace         | Workspace runtime selection uses `ArtifactRef`; installed legacy references remain intentionally.                                                                                                              |
| Same-name policy                    | Partial                       | Aggregate listing marks simultaneously eligible same-name Skills unavailable, and runtime resolution withholds ambiguous durable references. A unified migrated-Collection precedence policy remains deferred. |
| Assistant preset integration        | Incompatible                  | Assistant presets persist the current Skill Store reference type.                                                                                                                                              |
| Individual Skill import and export  | Not present                   | There is no native Skill transfer service or package closure builder.                                                                                                                                          |
| Skill bundle import and export      | Not present                   | There is no portable bundle manifest, relative member resolver, or deterministic archive workflow.                                                                                                             |
| Ordered metadata migrations         | Not present                   | User Skills use a single map-file schema and built-ins use a separate overlay database.                                                                                                                        |

## 10. Next steps

The Artifact Store and Workspace completion does not change standalone Skill
Store durable ownership. Its direct package implementation remains active
legacy behavior, not dead code and not an Artifact Store managed Source.

Accordingly, this completion pass intentionally performs no deletion or
dual-write conversion in `internal/skillstore`. Its direct package creation,
embedded hydration, MapFileStore metadata, and built-in overlay files are not
dead code. They remain required until the dedicated one-way migration updates
assistant preset and conversation references in the same release.

Accordingly, `internal/skillstore` filesystem package creation, embedded
hydration, MapFileStore metadata, built-in overlays, and installed Skill
runtime references must remain until the one-way migration updates assistant
preset and conversation references in the same release. They must not be
dual-written or deleted as cleanup in the Artifact Store and Workspace pass.

### 10.1 Satisfied Artifact Store prerequisites

- Implement Collections, Portable Collection Definitions, Artifact refs,
  managed Sources, adoption, pinning, and suppression.
- Establish one application-owned Artifact Store.
- Keep direct movement deferred unless a committed workflow requires it.

These prerequisites are now available. This migration HLD intentionally retains
the list as a boundary contract, not as an instruction to reopen Artifact Store
or Workspace persistence during the standalone Skill migration.

### 10.2 Extract the common Skill artifact package

- Move `SKILL.md` decoding and validation out of Workspace-specific ownership.
- Define `agent.skill` schema and decoder revision.
- Emit parser warnings as occurrence diagnostics.
- Keep local metadata out of canonical definitions.
- Add Skill content-closure enumeration for `SKILL.md`, resources, and scripts.
- Add native Skill directory and archive exporters.

### 10.3 Implement `skill.bundle` policy

- Define Collection data and attachment roles.
- Define allowed Source scopes.
- Define automatic versus explicit adoption.
- Define local Artifact data.
- Define collision policy.
- Define the portable `skill.bundle` schema, member references, validator,
  importer, and exporter.

### 10.4 Implement managed authoring

- Add staged managed Source writes.
- Allocate Artifact IDs before source publication.
- Add idempotency and orphan cleanup.
- Refresh and adopt through normal Collection behavior.

### 10.5 Convert built-ins

- Move portable metadata into each `SKILL.md`.
- Replace the current built-in metadata file with Portable Skill Bundle
  Definitions plus a minimal application bootstrap index.
- Register the hydrated tree as a normal Source.
- Bootstrap Collections and Artifacts idempotently.
- Replace overlay flags with normal Collection and Artifact enablement.

### 10.6 Port runtime selection

- Replace current Skill reference unions with `ArtifactRef`.
- Generalize Workspace source verification to all filesystem-backed `agent.skill` Artifacts.
- Verify Source generation or package closure before registration.
- Keep runtime state rebuildable and nontransactional.

### 10.7 Port consumers

- Update aggregate Skill listing and collision handling.
- Update Workspace Skill projection.
- Update assistant preset selection and validation.
- Update conversation selection persistence.
- Regenerate Wails bindings and frontend models.
- Change startup to stop opening the standalone Skill Store.

### 10.8 Handle existing data separately

- Do not introduce dual writes or permanent legacy identity fields.
- If existing data is retained, use a one-way importer that assigns new Root,
  Collection, Source, and Artifact IDs.
- Reset or migrate dependent assistant preset and conversation references in
  the same application transition.

### 10.9 Implement Skill transfer after the core switch

- Import individual Skill directories and archives into managed or package Sources.
- Export individual Skills in native directory or archive form.
- Import Portable Skill Bundle Definitions with embedded, relative, or
  policy-approved URI members.
- Export deterministic self-contained Skill bundle archives.
- Verify member and package digests during import.
- Add transfer tests for resources, scripts, parser warnings, unavailable
  Sources, archive traversal, duplicate paths, and local metadata exclusion.

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
