# Workspace Feature Requirements and Architecture

## 1. Purpose

This document defines Workspace as a product feature built on Artifact Store
Collections.

It specifies:

- What a Workspace represents.
- How a Workspace is represented as a portable, shareable Collection.
- Which source and artifact behavior it owns.
- Its discovery and adoption policy.
- Its heterogeneous member and relative-reference rules.
- Context and Skill projections.
- Runtime and inference handoff boundaries.
- Future Workspace import and export behavior.
- Current implementation status.
- Ordered work needed to reach the target.

Current root-based code is a stepping stone and not a compatibility constraint.

## 2. Feature intent

A Workspace gives a user one coherent project or contextual scope containing:

- Project instructions and Context.
- Agent Skills.
- Tools.
- MCP server declarations.
- Model presets.
- Agent and Assistant definitions.
- Optional built-in, library, package, or overlay content.
- Future domain Artifact kinds supported by complete Workspace adapters.

A Workspace is intentionally a heterogeneous Collection. Artifact Store and
Workspace policy must not assume that all members have one Artifact kind.

It answers:

- Which sources participate in this Workspace?
- Which artifacts were discovered?
- Which artifacts are valid, missing, disabled, or stale?
- Which local Artifact records represent them?
- Which Context and Skills are selected for a conversation?
- Which source and definition revision is used at runtime?
- Which portable members and assets are required to share the Workspace?
- Which local settings must be omitted from a shared Workspace?
- Whether a shared Workspace can be imported reproducibly?

## 3. Outcomes

Workspace must allow a user to:

- Create a filesystem-backed or empty Workspace.
- Retain stable Workspace identity independently of a filesystem path.
- Attach additional Sources by explicit role.
- Refresh once and receive one coherent catalog.
- Browse valid, invalid, missing, adopted, and unadopted occurrences.
- Preserve local Artifact settings across refresh.
- Select Context and Skills through typed Artifact references.
- Compose bounded Context for normal inference assembly.
- Compose validated load plans for selected supported Artifacts.
- Use eligible filesystem Skills through normal Agent Skills runtime behavior.
- Keep installed and Workspace persistence ownership separate.
- Export one Workspace as a portable manifest or self-contained package.
- Import a shared Workspace from a local file, directory, archive, or
  policy-approved URI.
- Preserve native member formats such as Markdown, JSON, YAML, directories,
  and multi-file Skill packages.

## 4. Dependencies and boundaries

Workspace requires Artifact Store support for:

- Technical Roots.
- Root-scoped Sources.
- Collections and Collection Attachments.
- Collection catalogs and occurrences.
- Stable Artifact Records and `ArtifactRef`.
- Consumer-owned adoption policy.
- Canonical definitions and diagnostics.
- Trusted Source runtime access.
- Portable Collection and Artifact definition contracts.
- Safe relative locator and external URI resolution.
- Planned Collection and Artifact import and export orchestration.

Workspace owns:

- The `workspace.collection` Collection kind.
- Workspace Collection data.
- Attachment role policy.
- Workspace discovery conventions and planning.
- Supported artifact-kind policy.
- Automatic adoption decisions.
- Workspace management views.
- Context composition.
- Skill runtime eligibility and source handoff.
- Workspace reference resolution.
- The portable `workspace.collection` schema.
- Workspace member validation and portable closure enumeration.
- Workspace-native import and export codecs.

Workspace does not own:

- Generic persistence or source transport.
- Agent Skills sessions and tools.
- Inference provider clients.
- Conversation persistence.
- Installed Skill persistence.
- Secret values.
- MCP connections or process execution.
- Generic archive extraction, network acquisition, or package safety.
- Portable schemas for Artifact domains owned by other packages.

## 5. Workspace model

### 5.1 Workspace identity

A Workspace is:

    Collection kind: workspace.collection

Its stable reference is:

    WorkspaceRef{RootID, CollectionID}

The Collection ID is the Workspace aggregate identity. A Root ID is only the
namespace.

Workspace Artifact selections persist:

    ArtifactRef{RootID, ArtifactID}

All Workspace operations must validate current Collection membership.

### 5.2 Local and portable Workspace representations

A local Workspace Collection contains:

- Local Root and Collection IDs.
- Enablement and revisions.
- Source attachments.
- Local discovery and composition policy.
- Local runtime-disable and credential references.

These values are not exported automatically.

The shareable representation is a Portable Workspace Collection Definition,
initially represented by `.flexigpt/workspace.json`.

Its schema contains only portable data such as:

- Workspace logical name and version.
- Portable labels and descriptive metadata.
- Portable discovery conventions.
- Member references and package-relative roots.
- Optional expected member digests.
- Domain-approved portable policy.

It must not contain:

- Root, Collection, Source, or Artifact IDs.
- Absolute filesystem paths.
- Local enablement.
- Credential or secret references.
- Runtime or conversation state.

A Workspace may be shared as:

- A manifest in a source tree with relative member locators.
- A self-contained archive with the manifest at the package root.
- A linked manifest containing policy-approved external URIs.

The initial native format is JSON. A future YAML codec may represent the same
canonical Portable Workspace Collection Definition without changing the local model.

### 5.3 Workspace Collection data

Workspace Collection data contains local settings such as:

- Discovery preferences.
- Context composition preferences.
- Future local policy references.

The primary Source relationship is represented by the `primary` attachment and
must not be duplicated as an independently authoritative Source ID in Collection data.

Workspace mode is derived:

- No primary attachment means an empty Workspace.
- One enabled primary filesystem attachment means a filesystem Workspace.

### 5.4 Attachment roles

Workspace policy supports:

| Role               | Meaning                        |
| ------------------ | ------------------------------ |
| `primary`          | Main filesystem project Source |
| `built-in`         | Application-provided content   |
| `library`          | Reusable local or team library |
| `attached-package` | Imported or mounted package    |
| `overlay`          | Supplemental content           |

A filesystem Workspace has exactly one enabled primary filesystem attachment.
An empty Workspace has none.

Replacing the primary Source is an explicit Collection operation and does not
change Workspace identity.

### 5.5 Supported Artifact kinds

The initial supported kinds are:

| Kind                | Purpose                                |
| ------------------- | -------------------------------------- |
| `workspace.context` | Bounded Context contribution           |
| `agent.skill`       | Shared portable Agent Skill definition |

`.flexigpt/workspace.json` is the Portable Workspace Collection Definition and
bootstrap control document. It is not required to become an Artifact Record
inside the Workspace that it describes.

Tool, MCP, Model, Agent, and Assistant support is deferred until each has a
complete decoder, schema, validation, adoption, projection, local setup, and
runtime or product consumer.

When those domains are supported, the same Workspace Collection may contain
their Artifacts together. Homogeneous grouping remains available through
derived kind views and does not require separate Workspace Collections.

## 6. Discovery behavior

### 6.1 Conventions

Workspace initially recognizes:

- `.flexigpt/workspace.json` as the portable Workspace descriptor.
- `AGENTS.md`.
- `CLAUDE.md`.
- Optional `README.md`.
- Configured Skill roots containing `SKILL.md`.
- Explicitly targeted Markdown Context files.

### 6.2 Effective plan

The effective Collection discovery plan is derived from:

- Product defaults.
- Workspace Collection discovery preferences.
- Primary-source Portable Workspace Collection Definition preferences.
- Attachment role and attachment-local settings.
- Registered decoder capabilities.

The plan must be deterministic and bounded.

### 6.3 Bootstrap observation

The optional primary Workspace descriptor may be read before final discovery.

That observation:

- Must use the same Source generation expected by final discovery.
- Must not publish an intermediate catalog.
- Must fail the refresh if its Source changes before publication.

### 6.4 Publication and adoption

One user refresh publishes one Collection catalog.

Workspace policy automatically adopts supported Context and Skill conventions,
unless a typed Source Binding is explicitly suppressed.

Other valid occurrences may remain unadopted.

### 6.5 Portable membership and transfer

A Workspace descriptor may identify members through:

- Embedded subresources.
- Relative locators.
- Relative directory roots and include patterns.
- Policy-approved external URIs.

Relative references resolve against the descriptor directory for a loose
Workspace or against the package root for an archive.

Import must:

- Acquire or open the descriptor through an explicit Source or resolver.
- Stage and validate referenced members.
- Create a local Workspace Collection.
- Create managed, package, or configured Sources and attachments.
- Discover and adopt supported members according to Workspace policy.
- Assign new local IDs.

Export may produce:

- A descriptor-only linked Workspace.
- A source tree containing `.flexigpt/workspace.json`.
- A self-contained archive containing the descriptor and vendored members.

Self-contained export must:

- Snapshot or capture selected members.
- Include domain-declared Artifact assets.
- Rewrite vendored references to package-relative locators.
- Report unsupported, denied, unresolved, or omitted members.
- Exclude local Source paths, IDs, credentials, enablement, and runtime state.

## 7. Context behavior

Workspace Context projection converts:

    ArtifactRef
      -> current Workspace Artifact
      -> workspace.context definition
      -> bounded Context contribution

The projector owns:

- Record and Collection enablement checks.
- Runtime-disable checks.
- Definition validation.
- Convention ordering.
- Prompt and per-document byte budgets.
- Truncation and exclusion.
- Source provenance.
- Structured decisions and diagnostics.

Workspace returns a Context load plan. Application inference composition decides
where to place that plan in the normal model instruction flow.

Workspace does not persist conversations or construct provider-specific requests.

## 8. Skill behavior

Workspace discovers the shared `agent.skill` definition rather than a
Workspace-specific Skill kind.

For an eligible filesystem-backed Skill, runtime handoff is:

    ArtifactRef
      -> Workspace membership validation
      -> current Artifact and occurrence
      -> definition and Source generation checks
      -> trusted local-path resolution
      -> SKILL.md digest verification
      -> containing package directory
      -> ephemeral Agent Skills SkillDef

The handoff must:

- Require an enabled Workspace and Artifact.
- Require available source-derived state.
- Require a current Collection catalog.
- Verify the occurrence definition digest.
- Verify `SKILL.md` content.
- Verify Source generation or an equivalent package content closure.
- Keep native paths outside persistent references and portable definitions.
- Use the normal Agent Skills filesystem provider.

The local desktop management API may display a selected Source root path.
That trusted presentation exception must not become a portable field, runtime
identity, or conversation reference.

## 9. Reference and precedence behavior

Explicit references use `ArtifactRef`.

Portable selectors may match by:

- Kind.
- Logical name.
- Supported version constraint.
- Labels.

Workspace must:

- Exclude disabled, unavailable, stale, and projection-invalid candidates.
- Return unresolved when no candidate remains.
- Return ambiguity when equal-precedence candidates remain.
- Never choose by map order, path order, or registration order.
- Provide explainable candidate diagnostics when precedence is applied.

## 10. Functional requirements

| ID       | Requirement                                                                     | Priority |
| -------- | ------------------------------------------------------------------------------- | -------- |
| `WS-F01` | Create filesystem and empty Workspaces as `workspace.collection` Collections.   | Core     |
| `WS-F02` | Address Workspaces through `WorkspaceRef`.                                      | Core     |
| `WS-F03` | Enforce zero or one primary attachment according to Workspace mode.             | Core     |
| `WS-F04` | Attach built-in, library, package, and overlay Sources.                         | Core     |
| `WS-F05` | Replace a primary Source without replacing Workspace identity.                  | Core     |
| `WS-F06` | Build bounded deterministic discovery plans.                                    | Core     |
| `WS-F07` | Publish one coherent catalog for one refresh request.                           | Core     |
| `WS-F08` | Automatically adopt supported Context and Skill occurrences.                    | Core     |
| `WS-F09` | Respect explicit Source Binding suppression.                                    | Core     |
| `WS-F10` | Preserve local Artifact state across refresh.                                   | Core     |
| `WS-F11` | Expose resources, occurrences, diagnostics, and currentness by kind.            | Core     |
| `WS-F12` | Resolve explicit references and selectors deterministically.                    | Core     |
| `WS-F13` | Compose validated generic load plans for selected supported Artifacts.          | Core     |
| `WS-F14` | Compose bounded Context load plans.                                             | Core     |
| `WS-F15` | Integrate selected Context with normal inference assembly.                      | Core     |
| `WS-F16` | Load selected `agent.skill` Artifacts through normal Agent Skills runtime.      | Core     |
| `WS-F17` | Preserve resource and script parity for eligible filesystem Skills.             | Core     |
| `WS-F18` | Keep installed and Workspace persistence ownership separate.                    | Core     |
| `WS-F19` | Keep local paths, credentials, and runtime handles out of portable definitions. | Core     |
| `WS-F20` | Return diagnostics for unavailable, denied, stale, or ambiguous selections.     | Core     |
| `WS-F21` | Allow heterogeneous Artifact kinds according to Workspace policy.               | Core     |
| `WS-F22` | Define a portable, versioned `workspace.collection` schema.                     | Core     |
| `WS-F23` | Import and export linked or self-contained Workspaces.                          | Planned  |
| `WS-F24` | Add future Artifact kinds only with an end-to-end consumer path.                | Core     |
| `WS-F25` | Support capture, fork, dependency history, and generic materialization.         | Optional |

## 11. API direction

Workspace operations include:

    CreateFilesystemWorkspace(rootID, request) -> WorkspaceRef
    CreateEmptyWorkspace(rootID, request) -> WorkspaceRef
    GetWorkspace(workspaceRef)
    ListWorkspaces(rootID)
    UpdateWorkspace(workspaceRef, request)
    RetireWorkspace(workspaceRef)
    PurgeWorkspace(workspaceRef)
    ReplaceWorkspacePrimarySource(workspaceRef, request)
    SetWorkspacePrimarySource(workspaceRef, request)

Attachment and catalog operations include:

    AttachWorkspaceSource(workspaceRef, request)
    UpdateWorkspaceAttachment(workspaceRef, sourceID, request)
    DetachWorkspaceSource(workspaceRef, sourceID, request)
    RefreshWorkspace(workspaceRef)
    GetWorkspaceCatalog(workspaceRef)

Artifact and projection operations include:

    GetWorkspaceArtifact(workspaceRef, artifactRef)
    ListWorkspaceArtifacts(workspaceRef)
    AdoptWorkspaceOccurrence(workspaceRef, occurrenceRef)
    PinWorkspaceArtifact(workspaceRef, sourceBinding)
    ListWorkspaceSuppressions(workspaceRef)
    SuppressWorkspaceBinding(workspaceRef, sourceBinding)
    UnsuppressWorkspaceBinding(workspaceRef, sourceBinding)
    SetWorkspaceArtifactEnabled(workspaceRef, artifactRef, request)
    SetWorkspaceArtifactRuntimeDisabled(workspaceRef, artifactRef, request)
    ComposeWorkspaceLoadPlan(workspaceRef, artifactRefs)
    ComposeWorkspaceContext(workspaceRef, artifactRefs)
    LoadWorkspaceSkills(workspaceRef, artifactRefs)

Planned transfer operations include:

    ImportWorkspace(rootID, input) -> WorkspaceRef
    ExportWorkspace(workspaceRef, options) -> PortableOutput
    ExportWorkspaceArtifact(workspaceRef, artifactRef, options) -> PortableOutput

No API infers durable identity from an encoded runtime string.

## 12. Current implementation status

| Capability                              | Status                    | Current implementation                                                                                                          |
| --------------------------------------- | ------------------------- | ------------------------------------------------------------------------------------------------------------------------------- |
| Filesystem and empty Workspace creation | Present                   | Provisioning creates Sources and typed Collections with compensation.                                                           |
| Workspace Collection identity           | Present                   | A Workspace is a `workspace.collection` addressed by `WorkspaceRef`.                                                            |
| Shared application Artifact Store       | Present                   | Application startup opens one Artifact Store and injects its services into Workspace.                                           |
| Attachment role policy                  | Present as reusable logic | Primary, built-in, library, package, and overlay roles exist for Collection attachments.                                        |
| Workspace discovery planning            | Present                   | Defaults, profile scopes, explicit decoder hints, preferences, bootstrap definition, and bounds are implemented.                |
| Portable Workspace descriptor           | Present foundation        | `.flexigpt/workspace.json` is read as a portable Collection descriptor and remains outside Workspace Artifact Records.          |
| Context decoding and composition        | Present                   | Context validation, ordering, budgets, truncation, and diagnostics exist.                                                       |
| Context inference integration           | Present                   | The Workspace inference bridge resolves Context, injects bounded current-turn input, and returns usage provenance.              |
| Skill Markdown parsing                  | Present                   | `SKILL.md` parsing and validation are implemented in the Workspace Skill adapter.                                               |
| Shared `agent.skill` definition         | Present                   | Workspace uses the shared `agent.skill` definition and decoder.                                                                 |
| Skill filesystem runtime handoff        | Present                   | Any Source adapter explicitly advertising trusted local paths can perform the verified `SKILL.md` runtime handoff.              |
| Full package freshness                  | Point-in-time verified    | Source generation is verified before and after trusted package handoff; portable closure export remains planned.                |
| Heterogeneous Collection support        | Present foundation        | Workspace uses one collection-scoped catalog and Artifact lifecycle for all registered supported kinds.                         |
| Workspace import and export             | Not present               | There is no portable Collection codec, URI resolver, closure builder, or archive workflow.                                      |
| Typed Workspace and Artifact refs       | Present                   | Workspace uses `WorkspaceRef` and `ArtifactRef`; runtime handles remain process-local.                                          |
| Runtime reconciliation                  | Present and derived       | Wails wrappers enqueue coalesced runtime reconciliation after durable Workspace mutations; metadata commits do not wait for it. |
| Generic Source mutation reconciliation  | Present and derived       | Artifact Store wrapper mutations schedule root-scoped Workspace runtime reconciliation without coupling Artifact Store to runtime. |
| Workspace Artifact purge                | Present                   | Workspace exposes membership-checked destructive Artifact purge and schedules derived runtime reconciliation.                    |
| Primary Source lifecycle                | Present                   | Empty Workspaces can gain, replace, or explicitly clear one primary filesystem Source through a dedicated operation.            |
| Attached Source provisioning            | Present for current scope | Artifact Store exposes Source lifecycle APIs and Workspace attaches existing Sources through typed attachment roles.            |
| Workspace retirement and purge          | Present                   | Retirement is reversible only through retained metadata; typed purge verifies retired `workspace.collection` kind before delete. |
| Future artifact kinds                   | Not present               | MCP, Tool, Model, Agent, and Assistant paths are absent.                                                                        |

## 13. Next steps

### 13.0 Current core completion boundary

The current Workspace scope is filesystem and empty Workspace lifecycle,
existing-Source attachment, Context, shared Agent Skills, catalog inspection,
runtime handoff, inference hydration, and derived Skill Runtime reconciliation.
Current-catalog pins immediately reflect valid, invalid, missing, or
incompatible observations when a current catalog exists. Conversation usage
records selected and actually used definition and Artifact revisions.

Discovery profiles now request registered Workspace decoders for their own
explicit locators and directory roots. This is required for attached-library
Markdown: scanning a candidate is not sufficient when a decoder intentionally
recognizes non-conventional files only after an explicit hint.

Transfer formats, URI acquisition, archive handling, portable closure export,
and additional Artifact kinds remain deliberate omissions. Workspace does not
perform generic Collection validation through raw Artifact Store APIs; typed
Workspace APIs remain the public mutation boundary for Workspace state. The
historical implementation items in sections 13.1 through 13.8 are complete for
the current Workspace scope. Only transfer, portable closure work, and future
artifact domains remain deferred.

### 13.1 Complete Artifact Store prerequisites

- Implement Collections, Collection catalogs, Artifact refs, adoption, pinning, suppression, and managed Sources.
- Do not build a second Workspace-specific persistence model.

### 13.2 Change composition ownership

- Open Artifact Store once in application startup.
- Inject Root, Source, Collection, catalog, Artifact, and refresh services into Workspace.
- Make Workspace close only its own derived components.

### 13.3 Port Workspace policy

- Convert Root data into `workspace.collection` data.
- Make the primary attachment authoritative.
- Port role validation, discovery planning, adoption, and catalog queries to Collection scope.
- Convert `.flexigpt/workspace.json` from an Artifact definition into the
  Portable Workspace Collection Definition.

### 13.4 Define the portable Workspace format

- Define the versioned `workspace.collection` canonical schema.
- Define JSON decoding and export first, with room for equivalent YAML codecs.
- Define embedded, relative, and external member references.
- Define which discovery preferences are portable.
- Define Workspace content-closure enumeration across heterogeneous Artifact kinds.
- Define linked and self-contained export profiles.
- Reserve import and export service boundaries without blocking the initial
  Collection persistence work.

### 13.5 Port Context

- Replace `RootID` and `RecordID` contracts with `WorkspaceRef` and `ArtifactRef`.
- Preserve ordering, provenance, budgets, and diagnostics.
- Add an end-to-end test proving selected Context reaches the provider request.

### 13.6 Port Skills with the Skill Store replacement

- Extract shared `agent.skill` decoding and validation.
- Replace encoded Workspace Skill identities with typed Artifact selections.
- Preserve private filesystem handoff and Agent Skills runtime parity.
- Add Source-generation or package-closure freshness verification.

### 13.7 Decouple runtime reconciliation

- Treat runtime state as rebuildable derived state.
- Do not report an unqualified mutation failure after durable Workspace state has already committed.
- Return synchronization status or retry asynchronously.
- Batch resolution for multiple Artifact refs from one Workspace.

### 13.8 Complete Source management

- Add application flows to create or select built-in, library, package, and overlay Sources.
- Keep Source transport outside Workspace role semantics.

### 13.9 Add transfer and future kinds in bounded increments

- Implement Workspace import and export after the Collection and portable
  definition contracts are stable.
- Add MCP only after defining credential references and connection policy.
- Add Tool, Model, Agent, Assistant, and conversation-related kinds only with
  complete domain codecs, projectors, and product consumers.

### 13.10 Core verification suite

The Workspace verification suite must cover:

- Filesystem and empty Workspace creation.
- Existing Source attachment for each currently supported role.
- Attached arbitrary Markdown Context through profile decoder hints.
- Context and Skill discovery, automatic adoption, catalog inspection, and load.
- Current and stale catalog behavior after Collection, attachment, and Source revisions.
- Explicit adoption, pinning, suppression, unsuppression, unadoption, and purge.
- Context prompt composition and inference hydration provenance.
- Skill Source generation and `SKILL.md` digest verification.
- Retirement and typed Workspace purge.

## 14. Acceptance outcomes

Workspace satisfies this document when:

- A Workspace is a `workspace.collection`.
- `WorkspaceRef` is its aggregate identity.
- Attachments, catalog, occurrences, and Artifacts are Collection-scoped.
- Primary Source state has one authoritative representation.
- Context and Skill selections persist `ArtifactRef`.
- Artifact identity is not derived from Workspace identity, without requiring
  direct move support.
- Selected Context reaches normal inference assembly with provenance and limits.
- Workspace Skills use shared `agent.skill` definitions.
- Eligible filesystem Skills use normal Agent Skills sessions, rendering, resources, and scripts.
- A Workspace can contain different supported Artifact kinds in one Collection.
- `.flexigpt/workspace.json` is a portable Collection descriptor without local IDs or paths.
- A Workspace can be exported as a linked manifest or self-contained package.
- A shared Workspace can be imported from a local Source or policy-approved URI.
- Relative member references remain valid after a self-contained package is moved.
- Native member formats are preserved through domain codecs.
- Workspace and installed resources retain separate persistence owners.
- No root-as-Workspace, `RecordID`, or encoded Workspace Skill identity is required by new persistence.
- A retired Workspace can be purged through a typed Workspace lifecycle API
  without allowing that API to purge another Collection kind.
