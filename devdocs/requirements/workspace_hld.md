# Workspace High-Level Design

## Document status and authority

This document defines the current authoritative design baseline for Workspace
on Artifact Store. It is intended to supersede the prior Workspace HLD and its
current-scope replacement after review and adoption.

This HLD defines Workspace-specific architecture, ownership, invariants,
requirements, and current product boundaries. Generic Root, Source,
Collection, Attachment, Catalog, Artifact, Definition, managed-publication,
and protected-topology mechanics are defined by the Artifact Store HLD.

Authority is ordered as follows:

- This HLD governs intended Workspace architecture, product decisions, and
  change gates.
- Implemented code, schemas, and public API contracts are the source of truth
  for current behavior.
- A conflict between this HLD and implementation must be resolved explicitly.
  It must not become an undocumented compatibility rule.

This document does not commit the product to future Workspace transfer work.
Workspace import/export, archives, content closures, package CAS, and Artifact
move remain out of scope until a later HLD amendment is approved.

## Purpose

A Workspace is a local Artifact Store Collection representing a project and its
attached source-backed content.

Workspace provides:

- Empty and filesystem-backed Workspace lifecycle.
- A configured local Workspace namespace.
- Project Source and Attachment role policy.
- Bounded deterministic discovery of Context and Skills.
- Catalog inspection, automatic adoption, pinning, suppression, and local
  Artifact lifecycle.
- Context composition for inference.
- Workspace-owned Skill projection into the shared Agent Skills runtime.
- Local Workspace selection and usage provenance.

Workspace does not own Root administration, generic metadata persistence,
protected built-in content, installed Skill Bundle ownership, generic package
transfer, Agent Skills sessions, provider clients, or secret storage.

## Current scope

A Workspace is a local Collection of kind:

```text
workspace.collection
```

Its durable local reference is:

```text
WorkspaceRef {
  RootID
  CollectionID
}
```

A Workspace may be empty. When it has a primary project Source, that primary
Source is an enabled `fs-directory` Source rooted at a user-selected project
directory.

Workspace content is source-backed:

- Project files remain in their selected Source.
- Artifact Store indexes source state and stores derived semantic Definitions.
- Artifact Store does not copy a user-selected project tree or Workspace Skill
  package into managed package storage.
- Workspace import, export, archive transfer, generic closure transfer,
  package CAS, and Artifact move are not supported.

### Explicit non-goals

The current Workspace delivery does not support:

- Workspace import or export.
- Linked descriptor export.
- Self-contained Workspace package export.
- Archive import or export.
- Generic Content Closure generation.
- Generic package, blob, or tree content-addressed storage.
- Copying a user Workspace tree into Artifact Store managed storage.
- Offline native-content fallback for linked Workspace Skills.
- Imported Workspace provenance or imported Workspace Source roles.
- Cross-Root transfer.
- Direct Artifact move between Workspaces or Collections.
- Generic network acquisition.
- Descriptor-to-local-metadata synchronization.

`artifact.Service.Move` remains unsupported.

## Goals

Workspace must:

- Represent each Workspace as one local `workspace.collection` Collection.
- Keep Workspace identity stable while Sources and discovered content change.
- Use a configured, non-protected Workspace Root rather than caller-selected
  Roots.
- Support empty Workspaces and one-primary filesystem Workspaces.
- Support same-Root secondary Source Attachments with typed local roles.
- Discover Context and shared Agent Skills through bounded deterministic plans.
- Publish one coherent Catalog per refresh.
- Preserve local Artifact state while source-derived state changes.
- Automatically adopt supported content while respecting Suppressions.
- Keep Workspace Context and Workspace Skills as ordinary Artifact records.
- Keep Workspace Skill ownership separate from installed and built-in Skills.
- Keep paths, runtime state, Source configuration, and local policy out of
  source-owned Workspace documents.
- Provide bounded Context composition with local usage provenance.
- Project Workspace Skills only after current source-backed verification.
- Reject protected built-in Root and Source use.

## Workspace Root topology

Application composition owns one retained, non-protected Workspace Root.

- Workspace creation and listing use this configured Root.
- Transport Root fields are retained only for compatibility and do not let a
  caller select another Workspace Root.
- The Workspace Root is distinct from the protected built-in Root.
- The configured Workspace Root is retained against Root retirement and purge
  by application policy.
- Workspace services reject the protected Root and any Root other than the
  configured Workspace Root.

A Workspace is not a Root and does not own child Collections.

## Domain model

| Workspace concept              | Artifact Store representation                                |
| ------------------------------ | ------------------------------------------------------------ |
| Workspace                      | Local `workspace.collection` Collection                      |
| Workspace reference            | `WorkspaceRef` / `CollectionRef`                             |
| Primary project Source         | One enabled `primary` Attachment to an `fs-directory` Source |
| Secondary content              | `library`, `attached-package`, or `overlay` Attachment       |
| Context document               | `workspace.context` Artifact                                 |
| Workspace Skill                | `agent.skill` Artifact                                       |
| Catalog item                   | Catalog Observation                                          |
| Persistent Workspace selection | `ArtifactRef`                                                |
| Context prompt contribution    | Derived Context projection                                   |
| Runtime Skill                  | Ephemeral Agent Skills projection                            |

### Workspace modes

| Mode       | Meaning                                                                                 |
| ---------- | --------------------------------------------------------------------------------------- |
| Empty      | No primary Attachment exists; secondary Attachments may still exist                     |
| Filesystem | Exactly one enabled primary Attachment exists and uses an enabled `fs-directory` Source |

Changing or clearing the primary Source is an explicit Workspace operation. It
preserves Workspace identity and makes the Catalog stale.

### Source Attachment roles

| Role               | Meaning                                    | Current constraints                                                                                   |
| ------------------ | ------------------------------------------ | ----------------------------------------------------------------------------------------------------- |
| `primary`          | Main linked project Source                 | Exactly zero or one; must be enabled `fs-directory`; changed only through explicit primary operations |
| `library`          | Additional same-Root reusable Source       | Local feature role; attachment overrides allowed                                                      |
| `attached-package` | Additional same-Root package-shaped Source | Local role only; does not mean imported package provenance                                            |
| `overlay`          | Supplemental same-Root Source              | Local feature role; attachment overrides allowed                                                      |

Secondary roles are same-Root Source Attachments. Current generic Workspace
policy does not require every secondary Source to be `fs-directory`, although
filesystem Sources are the normal linked use case. The selected Source adapter
and Workspace discovery policy determine what content is usable.

`attached-package` has no import meaning in the current delivery.

### Local Workspace data

Workspace local data is bounded canonical JSON owned by Workspace:

| Record     | Current local policy data                                                             |
| ---------- | ------------------------------------------------------------------------------------- |
| Collection | Discovery policy revision and local discovery preferences                             |
| Attachment | Optional recursive and authoritative discovery overrides for eligible secondary roles |
| Artifact   | `RuntimeDisabled` local runtime-policy flag                                           |

This data is local-only. It is strictly decoded and validated by Workspace, but
current code does not implement a generic local `schemaID` and `schemaVersion`
envelope. It must not be confused with a portable Workspace descriptor.

### Supported Artifact kinds

Current Workspace supports:

| Artifact kind       | Purpose                       |
| ------------------- | ----------------------------- |
| `workspace.context` | Bounded Context contribution  |
| `agent.skill`       | Shared Agent Skill definition |

A kind is supported only when Workspace has a registered decoder, Definition
validator, adoption policy, and projection behavior. Valid JSON alone does not
make content a supported Workspace Artifact.

## Base invariants

- A Workspace is one local `workspace.collection` Collection in the configured
  Workspace Root.
- Workspace identity is local and remains stable when Sources change.
- The primary project Source remains linked to the selected filesystem Source.
- A user-selected Workspace tree is not copied into managed package storage.
- Secondary Attachment roles are local policy, not portable package provenance.
- Source configuration and native paths are not durable Workspace identity.
- Catalog Observations remain distinct from adopted and pinned Artifacts.
- Local Artifact identity, enablement, runtime-disable state, and local policy
  survive source reconciliation.
- Workspace Context and Workspace Skills may coexist in one heterogeneous
  Collection.
- Installed and built-in Skills remain outside Workspace ownership.
- Persistent Workspace item selection uses `ArtifactRef` and validates current
  Collection membership.
- Source-owned descriptor metadata influences discovery only. It does not
  silently replace local Workspace metadata or policy.
- Runtime sessions, paths, registrations, provider state, and conversation
  state are not portable Workspace state.
- Workspace cannot use the protected built-in Root or mount a protected
  built-in Source.
- No current Workspace flow imports, exports, archives, snapshots, or creates
  portable closures.

## Source-owned Workspace descriptor

The optional native descriptor is located at:

```text
.flexigpt/workspace.json
```

The current descriptor schema is:

```text
kind:          workspace.collection
schemaID:      workspace.collection.v1
schemaVersion: v1
```

The descriptor is a source-owned, schema-validated discovery document. It is
not a local Collection record, imported document, persisted provenance record,
or exported package manifest.

### Descriptor rules

- A missing descriptor is valid.
- A present valid descriptor is read from the primary Source only.
- An invalid descriptor fails Workspace refresh. It does not become a local
  fallback document.
- Schema canonicalization computes an omitted digest or verifies a supplied
  digest, but does not persist the document as a generic shareable record.
- The descriptor is not itself an Artifact in the Workspace Catalog.
- Descriptor logical metadata does not replace local Workspace display name,
  description, enablement, Attachment roles, runtime settings, or local
  discovery policy revision.

### Descriptor discovery behavior

A valid descriptor may contribute:

- Additional discovery locators.
- Additional discovery roots.
- Include-README preference.
- Explicit member locators.
- Optional expected raw source-content digests for member locators.

The descriptor is read from one confirmed primary Source snapshot. Its source
generation is made a precondition of the subsequent discovery plan. If that
Source changes before Catalog publication, refresh fails rather than publishing
an intermediate Catalog.

Descriptor-relative locators are resolved relative to the descriptor directory,
`.flexigpt/`, not the project root. A descriptor-relative root of `.` therefore
means `.flexigpt/`.

Current active descriptor behavior supports package-relative locator members
only. URI members are structurally valid schema forms but are rejected because
there is no resolver. Embedded members are not implemented. Member media type
and role metadata do not create transfer or materialization behavior by
themselves.

## Discovery and Catalog behavior

Workspace builds a deterministic effective discovery plan from:

- Product discovery conventions.
- Primary and secondary Attachment roles.
- Local Workspace discovery preferences.
- Local Attachment overrides where the role permits them.
- Valid primary-source descriptor hints.
- Registered decoder capabilities.

Current default conventions include:

- `AGENTS.md` and `CLAUDE.md` in the primary project Source.
- Optional `README.md` when local or descriptor discovery preferences request
  it.
- Workspace Skill roots, with `.flexigpt/skills` as the default Skill root.
- Explicitly requested Markdown Context files and roots.
- Secondary-source Markdown and Skill discovery according to the attached
  discovery profile.

Discovery is bounded by Source and plan limits. It does not execute discovered
content.

### Refresh contract

A Workspace refresh:

1. Reads current Workspace and Attachment metadata.
2. Reads and confirms the optional primary descriptor snapshot.
3. Builds a deterministic plan using local and descriptor policy.
4. Delegates bounded Source discovery and Definition storage to Artifact
   Store.
5. Automatically adopts supported Context and Skill Observations unless
   suppressed.
6. Confirms snapshots and atomically publishes one Catalog with reconciled
   Artifact source-derived state.
7. Does not mutate runtime registrations directly.

Publication fails when relevant Collection, Attachment, Source, source
generation, plan, decoder, or descriptor preconditions change. The prior
Catalog remains current when publication fails.

Catalog freshness means that recorded Collection and Source metadata, plan, and
decoder inputs still match. It is not continuous filesystem watching. Direct
external filesystem changes become visible on refresh and are additionally
checked by the Workspace Skill runtime handoff.

### Adoption, pinning, and suppression

Workspace automatically adopts supported valid Context and Skill Observations
through a feature-provided Artifact ID source. Users may also manually adopt or
pin supported typed Source Bindings.

- Pinning permits an expected Context or Skill before valid source content
  exists.
- Suppression prevents automatic re-adoption of a typed Source Binding.
- Unsuppression removes the local decision; later refresh applies normal
  adoption policy.
- Unsupported kinds are not adopted merely because a document is valid JSON.

Artifact Store owns generic Observation state and reconciliation. Workspace
owns which supported kinds are eligible for discovery, adoption, and
projection.

## Workspace Context behavior

A Workspace Context is an Artifact of kind:

```text
workspace.context
```

The Context decoder validates supported Markdown content and derives a
canonical Definition containing normalized semantic text, Context role, media
type, and display information.

### Context projection boundary

Current Context behavior is deliberately distinct from Skill native-package
handoff:

- Context native files remain source-owned and are not copied as package trees.
- The derived Context Definition retains normalized semantic text.
- Context composition reads that validated Definition after Catalog-current and
  Artifact eligibility checks.
- Context composition does not reopen the Source or verify raw source-content
  digest and source generation immediately before prompt construction.

Therefore, Context is not an offline Workspace package, but it is a persisted
semantic text projection. A current Catalog is required for composition, yet
it is not a live filesystem-byte assertion after external files change.

If product policy later requires every Context use to verify current source
bytes immediately before inference, that requires an explicit implementation
and HLD change.

### Context composition

Workspace Context projection owns:

- Definition validation.
- Context convention ordering.
- Per-document and aggregate prompt byte budgets.
- Truncation or exclusion policy.
- Runtime-disable checks.
- Structured diagnostics and composition decisions.
- Local usage provenance.

Persistent Context selections use `ArtifactRef`. Conversation and inference
usage may record selected and used Definition digests, Artifact revisions,
locators, availability, inclusion, exclusion, and truncation decisions. Those
are local consumer records, not descriptor content.

## Workspace Skill behavior

Workspace Skills use the shared `agent.skill` Definition and the shared Agent
Skill decoder.

A Workspace Skill runtime handoff verifies:

- Workspace and Artifact ownership.
- Workspace and Artifact enablement.
- Current Catalog and matching current Observation.
- Current shared Skill Definition compatibility.
- Source revision and source generation.
- Exact raw `SKILL.md` source-content digest.
- Trusted native local-path capability.

Only after those checks does Workspace hand the original Skill package directory
to the Agent Skills runtime. Workspace does not copy Workspace Skill resources
or scripts into managed Skill storage and does not persist a runtime path or
runtime `SkillDef`.

Agent Skills runtime owns registration, sessions, prompt construction, resource
access, script execution policy, and sandboxing. Workspace policy participates
in Artifact eligibility and runtime-disable decisions.

Installed and built-in Skills remain in their own Collections. The application
or conversation layer may combine selected Workspace and installed Skills, but
Workspace does not mount, adopt, or own protected built-in content.

## Runtime, selection, and path presentation

### Persistent selection

Persistent Workspace selections use `ArtifactRef`. Operations verify Root
equality, current Workspace membership, supported kind, and current
eligibility. Runtime and inference paths revalidate current state rather than
relying only on selection-time validation.

### Local source-path presentation

Normal Source summaries do not expose private Source configuration. A trusted
local Workspace management view may display the root path of an Attachment when
the selected Source adapter supports trusted local-path resolution.

That presentation exception is local-only:

- It does not expose Source configuration through generic Source APIs.
- It does not become portable descriptor content.
- It does not become Artifact identity, runtime identity, or conversation
  identity.

### Runtime state

Workspace does not persist native paths, runtime registrations, sessions, or
provider state. Runtime behavior is derived from Artifact Store state and may
be rebuilt.

## Lifecycle and retention

Workspace supports:

- Empty Workspace creation.
- Filesystem Workspace provisioning.
- Explicit primary Source set, replace, and clear operations.
- Same-Root secondary Source Attachment lifecycle.
- Refresh, automatic and manual adoption, pinning, suppression, unsuppression,
  Artifact enablement, local runtime disablement, unadoption, and typed purge.
- Workspace retirement and purge.

### Retirement and purge behavior

Workspace retirement retires and disables the underlying Collection.

Typed Workspace purge verifies that the target is a retired
`workspace.collection` with the expected revision. Current SQLite persistence
then removes Collection-scoped metadata through the Collection lifecycle,
including Catalog records, Attachments, Artifacts, and Suppressions.

Workspace purge does not:

- Delete linked project files.
- Delete arbitrary external Source content.
- Remove unrelated Source records.
- Perform a generic consumer-reference check in the current Workspace purge
  path.

Any stronger source-side deletion or consumer-reference policy requires an
explicit future feature design.

## Protected built-in exclusion

Workspace does not own protected built-ins.

- Workspace rejects the protected Root.
- Workspace cannot mount a protected built-in Source.
- Workspace cannot adopt protected built-in Artifacts.
- Workspace Artifact references remain same-Root.
- Built-in and installed Skills are combined with Workspace selections only at
  the application or conversation layer.

## Security boundaries

Workspace must:

- Reject the protected Root and cross-Root Artifact references.
- Keep normal Source configuration private.
- Use bounded Source discovery.
- Require current Catalog state before Context or Skill projection.
- Use trusted Source capabilities for native Skill package paths.
- Keep runtime paths and runtime registrations out of persistent Workspace
  references.
- Keep local Attachment roles, enablement, local discovery policy, runtime
  flags, conversation state, and diagnostics out of source-owned descriptor
  content.
- Never execute content during discovery.
- Delegate Skill script execution and sandbox policy to the Agent Skills
  runtime.

## Current-scope requirements

| ID       | Requirement                                                                                               | Status                              |
| -------- | --------------------------------------------------------------------------------------------------------- | ----------------------------------- |
| `WS-C01` | Represent each Workspace as a local `workspace.collection` Collection                                     | Implemented                         |
| `WS-C02` | Use `WorkspaceRef` for durable Workspace identity                                                         | Implemented                         |
| `WS-C03` | Use one configured retained non-protected Workspace Root                                                  | Implemented                         |
| `WS-C04` | Support empty and filesystem Workspace modes                                                              | Implemented                         |
| `WS-C05` | Enforce zero or one enabled primary filesystem Source Attachment                                          | Implemented                         |
| `WS-C06` | Support same-Root library, attached-package, and overlay Attachments as local roles                       | Implemented                         |
| `WS-C07` | Build bounded deterministic discovery plans                                                               | Implemented                         |
| `WS-C08` | Publish one coherent Catalog per refresh                                                                  | Implemented                         |
| `WS-C09` | Support `workspace.context` and shared `agent.skill` Artifacts                                            | Implemented                         |
| `WS-C10` | Automatically adopt supported content while respecting Suppressions                                       | Implemented                         |
| `WS-C11` | Preserve local Artifact state during source reconciliation                                                | Implemented                         |
| `WS-C12` | Compose bounded Context projections with diagnostics and local provenance                                 | Implemented                         |
| `WS-C13` | Resolve Workspace Skills through verified source-backed runtime handoff                                   | Implemented                         |
| `WS-C14` | Keep Workspace Skill ownership separate from installed and built-in Skills                                | Implemented                         |
| `WS-C15` | Reject protected Root, protected Source, and cross-Root Workspace ownership                               | Implemented                         |
| `WS-C16` | Validate `.flexigpt/workspace.json` through the versioned schema registry for discovery only              | Implemented                         |
| `WS-C17` | Keep linked Workspace trees and Workspace Skill package trees out of managed package storage              | Implemented                         |
| `WS-C18` | Expose Workspace mutation only through typed Workspace APIs                                               | Implemented                         |
| `WS-C19` | Do not provide Workspace transfer, closures, archives, package CAS, imported provenance, or Artifact move | Intentional current-scope exclusion |

## Current implementation status

### Implemented

- Configured retained Workspace Root and typed Workspace lifecycle.
- Empty and filesystem Workspace creation.
- Explicit primary Source lifecycle and secondary Attachment roles.
- Context and shared Skill discovery.
- Descriptor validation, canonicalization, descriptor-relative discovery hints,
  and expected member source-content digest checks.
- Catalog inspection, automatic adoption, manual adoption, pinning,
  suppression, Artifact enablement, runtime-disable state, unadoption, and
  typed purge.
- Context composition with bounded budgets, decisions, diagnostics, and local
  provenance.
- Workspace Skill runtime projection with Source generation and raw `SKILL.md`
  verification.
- Workspace API-safe projections, including local Source-path presentation for
  trusted desktop management.
- Conversation and inference integration through Artifact-backed Workspace
  selections.

### Intentionally absent

- Workspace transfer and portable package materialization.
- Archive and closure support.
- Imported Workspace role or provenance.
- Generic source-independent native-content fallback.
- Descriptor-driven local metadata synchronization.
- Cross-Root transfer and Artifact move.

## Implementation map

| Responsibility                                                 | High-level implementation area                           |
| -------------------------------------------------------------- | -------------------------------------------------------- |
| Workspace API and API-safe projections                         | `internal/workspace`                                     |
| Workspace Collection and Attachment policy                     | `internal/workspace/artifactadapter`                     |
| Workspace local Collection and Attachment data                 | `internal/workspace/collectiondata` and `attachmentdata` |
| Workspace Context decoding and composition                     | `internal/workspace/contextadapter`                      |
| Descriptor loading, plan construction, and refresh preparation | `internal/workspace/discovery`                           |
| Filesystem Workspace provisioning                              | `internal/workspace/provision`                           |
| Workspace selection and conversation usage                     | `internal/workspace/selection`                           |
| Shared Workspace Skill projection                              | `internal/skill/workspaceadapter`                        |
| Workspace descriptor schema and canonical model                | `internal/builtin/schema`                                |
| Artifact Store mechanics and source runtime                    | `internal/artifactstore`                                 |
| Application composition and Wails-facing Workspace boundary    | `cmd/agentgo`                                            |

The implementation map is a reading guide. It does not replace the invariants
and ownership rules in this HLD.

## Change gates

This HLD must be amended before adding:

- Workspace import or export.
- Content closure generation.
- Workspace package CAS or archive support.
- Snapshotting a linked Workspace tree for offline runtime fallback.
- Imported Workspace Attachment roles or provenance.
- Descriptor-to-local-metadata synchronization.
- Cross-Root Workspace transfer.
- Artifact move.
- A requirement that Context composition reverify source bytes immediately
  before inference.
- Stronger Workspace purge cleanup or consumer-reference policy.

A future transfer design must define package ownership, local identity
allocation, materialization, provenance, security policy, rollback behavior,
and Context and Skill runtime semantics before implementation begins.

## Prior-requirement transition

The prior Workspace requirement family is superseded as follows when this
document is adopted:

| Prior requirement family  | Disposition                                                                                                  |
| ------------------------- | ------------------------------------------------------------------------------------------------------------ |
| `WS-R01` through `WS-R14` | Retained by `WS-C01` through `WS-C15` with configured-Root and current runtime clarifications                |
| `WS-R15`                  | Narrowed to descriptor schema validation for source-owned discovery in `WS-C16`; it is not a transfer format |
| `WS-R16` through `WS-R18` | Retired from current scope because export, import, and closures are unsupported                              |
| `WS-R19`                  | Retained by `WS-C16` and `WS-C17` with source-document and local-state separation                            |
| `WS-R20`                  | Retained by `WS-C18`                                                                                         |

The old IDs must not be reused with changed meanings. No prior transfer plan,
portable package profile, or transfer completion criterion remains an active
requirement under this HLD.
