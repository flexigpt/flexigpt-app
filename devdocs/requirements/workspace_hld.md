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

The delivered Workspace is a feature-specific view over the shared Artifact
Store. It is not a second project database or a general-purpose Source
administration API. This document distinguishes delivered behavior from
portable transfer and future domain work.

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

A Workspace is a project scope, not the selected item itself. An Artifact is
the durable unit that can be enabled, pinned, suppressed, selected for a
conversation, loaded as Context, or handed to a runtime.

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

## 3. Delivered outcome and planned scope

### 3.1 Available now

A person can:

- Create filesystem-backed and empty Workspaces with stable identity.
- Attach existing Sources by Workspace role.
- Refresh a Workspace and receive one coherent catalog of observations.
- Browse valid, invalid, missing, recorded, and unrecorded observations.
- Automatically adopt supported Context and Skill observations, or explicitly
  adopt, pin, suppress, unsuppress, disable, or remove an Artifact.
- Preserve local Artifact settings while source-derived state changes.
- Persist Context and Skill selections using Artifact references.
- Compose bounded Context with provenance for an inference caller.
- Load eligible local-path-backed Skills through the normal Agent Skills
  runtime path.
- Keep Workspace state separate from legacy installed Skill persistence and
  keep private Source configuration out of Workspace views.
- Use Workspace management for Workspace Collections, attachments, discovery,
  Artifacts, Context, Workspace Skills, suppressions, and diagnostics.
- Use Artifact Store APIs for Root and Source administration while retaining
  Workspace APIs as the only Workspace mutation boundary.
- Keep Workspace Skills out of the installed Skills and Templates management
  route. They are visible only through Workspace management and a selected
  Workspace conversation scope.

### 3.2 Planned, not currently offered

- Import or export a Workspace as a linked manifest or self-contained package.
- Acquire Workspace content from archives or policy-approved external URIs.
- Export portable content closures for Skills and other multi-file Artifacts.
- Add Tool, MCP, Model, Agent, Assistant, and other Artifact domains after
  each has a complete product path.

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

The current `.flexigpt/workspace.json` reader is an optional primary-source
bootstrap input. It can contribute relative discovery targets and expected
content digests to a refresh; it does not create a transferable Workspace or
export package.

The intended shareable representation is a Portable Workspace Collection
Definition, initially represented by `.flexigpt/workspace.json`.

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

Today, only relative descriptor members in the primary Source are accepted as
refresh input. Embedded members and external URIs are deliberately rejected
until import, transfer, and acquisition policy exist.

The initial native format is JSON. A future YAML codec may represent the same
canonical Portable Workspace Collection Definition without changing the local model.

### 5.3 Workspace Collection data

Currently, Workspace Collection data stores discovery preferences and the
revision of the discovery policy that produced the catalog. Prompt-composition
limits are application configuration, not saved Workspace data.

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

### 5.6 Artifact-centred Workspace behavior

Discovery first produces observations. An observation explains what a Source
currently contains; it is not automatically a user selection. A Workspace
Artifact is the durable local record for an item the user wants to manage,
select, enable, or use.

For a client, the distinction is important:

- An unrecorded observation is discoverable content that has no local Artifact.
- An observed Artifact is created automatically only for supported kinds that
  are not explicitly suppressed.
- A pinned Artifact records an intended Source location even before matching
  content is available.
- Removing an Artifact can either prevent automatic recreation through
  suppression or remove only the local record.

The Artifact state explains its source-derived availability:

- Available means the current observation is valid and still matches the
  Artifact's expected kind.
- Missing means the source location or subresource is no longer observed.
- Invalid means the current source content could not be accepted.
- Incompatible means the same source location now produces a different kind.

Catalog currentness is separate from Artifact state. A visible Artifact can be
available while its catalog needs refresh because Workspace, Source, decoder,
or discovery-policy inputs changed.

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
| `WS-F15` | Return selected Context with provenance and bounded composition for inference.  | Core     |
| `WS-F16` | Load selected `agent.skill` Artifacts through normal Agent Skills runtime.      | Core     |
| `WS-F17` | Preserve resource and script parity for eligible filesystem Skills.             | Core     |
| `WS-F18` | Keep installed and Workspace persistence ownership separate.                    | Core     |
| `WS-F19` | Keep local paths, credentials, and runtime handles out of portable definitions. | Core     |
| `WS-F20` | Return diagnostics for unavailable, denied, stale, or ambiguous selections.     | Core     |
| `WS-F21` | Allow heterogeneous Artifact kinds according to Workspace policy.               | Core     |
| `WS-F22` | Read a versioned Workspace descriptor safely as an optional refresh input.      | Core     |
| `WS-F23` | Import and export linked or self-contained Workspaces.                          | Planned  |
| `WS-F24` | Add future Artifact kinds only with an end-to-end consumer path.                | Core     |
| `WS-F25` | Support capture, fork, dependency history, and generic materialization.         | Optional |

## 11. API ownership and client journey

This is the only implementation-oriented section of this document. It explains
which boundary a client should use without requiring the client to understand
storage details.

### 11.1 Use the Artifact API for shared setup

Use the Artifact API to manage Roots, Sources, supported Source kinds, and
application-managed package contents. A Source's private configuration is
write-only and must never be treated as Workspace or portable data.

### 11.2 Use the Workspace API for Workspace meaning

Use the Workspace API to create and manage Workspaces, attach Sources, refresh
discovery, inspect catalogs, and perform Workspace-scoped Artifact actions.
A filesystem Workspace creation flow can create its primary filesystem Source
as a convenience; otherwise, create or select Sources through the Artifact API.
There is intentionally no raw public Collection editor for Workspace state.

### 11.3 Typical client flow

1. Create or select a Root and Sources through the Artifact API.
2. Create an empty Workspace or a filesystem Workspace.
3. Attach eligible existing Sources where the user wants additional content.
4. Refresh and present the catalog before asking the user to select content.
5. Present observations separately from local Artifacts so a user can see what
   is discovered, adopted, suppressed, missing, or invalid.
6. Persist user selections with Artifact references, then request Context or
   Skill projections from the Workspace API.

### 11.4 Revisions, freshness, and destructive actions

Every mutation uses the revision returned by an earlier read. A conflict means
the displayed state changed and must be reloaded before retrying. Use catalog
currentness and diagnostics to decide whether a refresh is required; do not
treat catalog position, source path, definition digest, or runtime location as
durable identity.

When a Workspace owns an Artifact, use the Workspace API for removal so it can
verify membership. The generic Artifact purge operation is reserved for callers
where the necessary feature-level ownership checks have already happened.

## 12. Current implementation status

| Capability                         | Status                  | Current implementation                                                                                                                                                         |
| ---------------------------------- | ----------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Workspace identity and lifecycle   | Present                 | A Workspace is a `workspace.collection` addressed by `WorkspaceRef`. Filesystem and empty creation, update, primary Source changes, retirement, and typed purge are available. |
| Artifact-first API split           | Present                 | Artifact API owns Root and Source administration. Workspace API owns Workspace policy, catalog views, and Workspace-scoped Artifact actions.                                   |
| Source attachments and roles       | Present                 | One enabled filesystem primary Source is optional. Built-in, library, package, and overlay attachments can add content under typed Workspace rules.                            |
| Discovery and refresh              | Present                 | Refresh uses bounded deterministic plans, configured decoders, attachment settings, Workspace preferences, and primary descriptor observations.                                |
| Descriptor bootstrap               | Present but limited     | `.flexigpt/workspace.json` can add relative discovery targets and expected digests from the primary Source. It is not yet an import/export format.                             |
| Catalog freshness                  | Present                 | Workspace reports stale catalog metadata, decoder changes, and discovery-policy changes. A refresh produces a coherent replacement catalog.                                    |
| Artifact lifecycle                 | Present                 | Supported observations can be automatically adopted or manually adopted, pinned, suppressed, unsuppressed, enabled, runtime-disabled, unadopted, or purged.                    |
| Artifact-centred catalog views     | Present                 | Catalog responses distinguish observations from recorded Artifacts and expose available, missing, invalid, incompatible, unresolved, and unrecorded states.                    |
| Context projection and composition | Present                 | Context definitions are validated, ordered, bounded, truncated or excluded by policy, and returned with diagnostics and provenance for callers.                                |
| Shared Agent Skill projection      | Present                 | Workspace uses the shared `agent.skill` definition and validates `SKILL.md` before runtime use.                                                                                |
| Workspace frontend plumbing        | Present                 | Workspace UI uses Artifact Store APIs for Root and Source setup, Workspace APIs for Collection and Artifact behavior, and keeps Workspace Skills outside installed Skills UI.  |
| Skill runtime handoff              | Present                 | Eligible Sources with trusted local-path capability verify Source generation, occurrence definition, and `SKILL.md` content before exposing a package directory to runtime.    |
| Conversation selection provenance  | Present                 | Conversation resolution records selected and used Artifact revisions and definition digests, including partial or unavailable outcomes.                                        |
| Derived runtime reconciliation     | Present and nonblocking | Workspace Skill runtime state is rebuildable. Durable Workspace mutations must not be rolled back because a later runtime refresh is delayed or fails.                         |
| Source configuration privacy       | Present                 | Workspace views expose Source summaries and optional trusted local presentation paths, never raw Source configuration.                                                         |
| Workspace transfer                 | Not present             | There is no Workspace importer, exporter, archive workflow, URI resolver, or portable content-closure builder.                                                                 |
| Future Artifact domains            | Not present             | MCP, Tool, Model, Agent, Assistant, and other domains remain absent until each has a complete decoder, validator, projection, and product consumer.                            |

## 13. Current boundary and remaining work

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

No raw generic Collection mutation endpoint is exposed as a Workspace API.
Workspace mutations remain typed so Collection kind, attachment role, and
opaque local data are validated by Workspace policy. Generic Artifact Store
public APIs intentionally stop at Root, Source, managed-package, and scoped
destructive Artifact operations.

The frontend follows the same boundary:

- Workspace management uses typed Workspace APIs for Workspace lifecycle,
  attachment changes, catalog reads, Artifact adoption, suppression, and
  Artifact mutation.
- Source registration, Source availability, and Source summaries use the
  Artifact Store API. Private Source configuration is write-only and is not
  copied into Workspace or conversation state.
- Installed Skills remain in the Skills UI. Workspace Skills are projected from
  selected Workspace Artifacts and are never added to installed Skill bundles.

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

### 13.9 Remaining product work

- Keep new frontend and application flows on the Artifact API for Root and
  Source administration and on typed Workspace APIs for Workspace meaning.
  Workspace management and composer selection now follow this rule.
- Add a dedicated cross-feature Artifact Store Source administration surface
  only if users need Source management outside a Workspace. The current
  Workspace Source flow is intentionally a scoped Artifact Store client.
- Implement portable Workspace import and export only after defining content
  closure, archive, acquisition, provenance, and omission policy.
- Add explicit source-provisioning flows for built-in, library, package, and
  overlay content without placing source transport details in Workspace data.
- Keep inference assembly and Skill runtime synchronization at the application
  edge, with visible diagnostics and retry rather than rollback of committed
  Workspace metadata.
- Add future Artifact kinds only with their full decoder, schema validation,
  adoption policy, projection, runtime or product consumer, and transfer plan.

### 13.10 Verification that must stay current

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
- Workspace UI rejection of stale Workspace revisions after delayed catalog
  responses.
- Composer Workspace menu refresh behavior without recursive reloads.
- Installed Skills UI isolation from Workspace-provided Skills.
- Accessible Workspace search, radio groups, discovery-path fields, and
  source-binding controls.

The implemented suite also verifies that an empty Workspace can attach a
filesystem library Source, discover arbitrary Markdown through explicit
decoder hints, suppress an automatically adopted occurrence, and re-adopt it
only after explicit unsuppression and refresh. Transfer, archive, URI, and
future-domain verification remain deferred with their corresponding features.

## 14. Acceptance outcomes

The local Artifact-centred Workspace criteria below describe the delivered
core. The transfer and portable-package criteria remain target outcomes until
their importer, exporter, acquisition, and closure work is implemented.

The full Workspace target is complete when:

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
