# Workspace

## 1. Document purpose

This HLD defines Workspace as a product feature built on Artifact Store.

It specifies:

- What a Workspace represents.
- Which user problems it solves.
- What must already exist before Workspace can exist.
- How discovery, cataloging, precedence, projection, and runtime handoff behave.
- What Workspace owns and what remains outside it.

Workspace must not become another generic storage platform.

## 2. Problem statement

Users work in a project or contextual scope that may contain:

- Project-specific agents
- Skills
- Tools
- MCP server declarations
- Model preferences
- Instructions
- Context documents
- Built-in resources
- Attached libraries or packages

Today, these resources may be distributed across filesystem conventions and existing application stores. There is no single contextual view that answers:

- Which resources belong to this project?
- Which source supplied each resource?
- Whether a selector is unambiguous when several sources define the same logical artifact?
- Which resources are valid and available?
- Which local settings apply?
- What should be loaded for a given agent or conversation?
- How can project resources coexist with globally installed resources?

Workspace solves this by making a logical root the boundary for contextual artifact discovery and composition.

## 3. Workspace prerequisites

Workspace must not be introduced until the following foundations exist.

### 3.1 Mandatory Artifact Store prerequisites

| Prerequisite                      | Why it is required                                                               |
| --------------------------------- | -------------------------------------------------------------------------------- |
| Logical roots                     | A Workspace needs stable identity and lifecycle independent of a directory path. |
| Source registrations              | Filesystem, embedded, and attached content must use a common source abstraction. |
| Root-to-source attachments        | A Workspace must combine primary, built-in, library, and overlay sources.        |
| Safe bounded discovery            | Workspace discovery operates on untrusted project content.                       |
| Native format adapters            | Workspace files do not all use one portable envelope.                            |
| Canonical definitions             | Workspace resources need a stable portable representation.                       |
| Current root catalog publication  | Workspace must expose one coherent view after refresh.                           |
| Stable local records              | Workspace resources require local identity and enabled state.                    |
| Structured diagnostics            | Invalid project content must be visible and actionable.                          |
| Consumer-owned selector rules     | Workspace must decide whether matching candidates are unambiguous.               |
| Projection boundary               | Definitions must be convertible into existing domain-facing models.              |
| Native filesystem runtime handoff | Filesystem Workspace Skills require a private source-relative path resolver.     |

### 3.2 Mandatory product prerequisites

| Prerequisite                        | Why it is required                                                                             |
| ----------------------------------- | ---------------------------------------------------------------------------------------------- |
| Workspace runtime provider contract | A catalog without a runtime consumption path is not a complete product feature.                |
| Aggregate resource services         | Installed resources and Workspace resources must be presented without duplicating persistence. |
| Trust boundary                      | Selecting a folder must not automatically authorize execution.                                 |
| Secret resolution boundary          | Portable Workspace files must not contain local secret values.                                 |
| User-visible diagnostics            | Refresh and load failures must be understandable.                                              |
| Conversation Skill runtime          | Selected installed and Workspace Skills must share the normal Skill session flow.              |

### 3.3 Optional prerequisites

The following are not required for the first Workspace release:

- Portable package management
- Import and fork
- Historical catalog browsing
- Persistent dependency snapshots
- Generic source materialization
- PostgreSQL metadata support
- Arbitrary third-party source transports

## 4. Desired outcomes

Workspace should allow a user to:

- Select a project directory as a Workspace.
- Create an empty Workspace without a primary directory.
- Discover project resources using known conventions.
- Attach built-ins, libraries, overlays, or packages.
- Refresh the Workspace and see additions, changes, removals, and errors.
- Browse resources by artifact kind.
- Enable or disable local records without editing source files.
- Resolve portable references deterministically.
- Compose a load plan for selected agents or resources.
- Select Context and Skills from a Workspace for a conversation.
- Insert selected Workspace Context into the normal model instruction input.
- Use selected Workspace Skills alongside installed Skills through the same
  Skill runtime, session, prompt, render, resource, and script capabilities.
- Keep runtime credentials and policy local to the application.
- Preserve Workspace origin and source linkage without copying Workspace
  Skills into the installed Skill Store.

## 5. Scope

Workspace owns:

- The `workspace` root meaning.
- Workspace root configuration.
- Workspace source attachment roles.
- Workspace discovery conventions.
- Supported Workspace artifact kinds.
- Workspace-specific validation.
- Record derivation for Workspace resources.
- Workspace selector resolution.
- Workspace catalog views.
- Projection into domain management models.
- Composition of validated runtime inputs.
- Source-relative runtime projection for selected Workspace Skills.
- Conversation-scoped Context and Skill handoff.
- Workspace-specific runtime diagnostics and provenance.
- Workspace lifecycle operations.

## 6. Non-goals

Workspace does not own:

- Generic persistence
- Generic source transport
- Generic canonicalization
- Skill execution
- Tool execution
- MCP connection management
- Model client construction
- Secret storage
- Trust decisions
- Conversation persistence
- Global installed-resource persistence
- Remote source acquisition
- Package distribution infrastructure

## 7. Workspace actors

### 7.1 User

The user:

- Selects or creates a Workspace.
- Attaches optional resource sources.
- Refreshes discovery.
- Reviews diagnostics.
- Chooses resources for use.
- Applies local setup and trust decisions.

### 7.2 Workspace management service

The management service:

- Maintains Workspace root data.
- Plans discovery.
- Produces a Workspace catalog.
- Synchronizes local records.
- Resolves references.
- Produces projections and load plans.

### 7.3 Existing resource providers

Existing Skill, Tool, MCP, Model, and Assistant stores remain providers of installed or global resources.

They are not rewritten by Workspace.

### 7.4 Workspace Skill runtime projection

A Workspace Skill runtime projection:

- Reads Workspace records and definitions.
- Receives selected Workspace record identities from the application flow.
- Resolves local setup.
- Applies trust and policy.
- Resolves an approved filesystem-backed `SKILL.md` occurrence to its existing
  absolute containing directory.
- Registers that directory through the ordinary Agent Skills filesystem
  provider rather than a Workspace-specific provider.
- Uses the same resource indexing, `skills-readresource`, and
  `skills-runscript` behavior and policy as installed filesystem Skills.
- Preserves Workspace origin, record state, source precedence, and diagnostics.
- Does not insert Workspace resources into existing stores.

## 8. Workspace root model

A Workspace is a typed Artifact Root.

Its app-local data includes:

- Workspace mode
- Optional primary source
- Discovery preferences

### 8.1 Filesystem Workspace

A filesystem Workspace has:

- Exactly one primary filesystem source.
- Exactly one enabled primary attachment.
- Optional additional sources.
- A stable root identity independent of the selected path.

The filesystem path belongs to the source registration, not the Workspace’s portable data.

### 8.2 Empty Workspace

An empty Workspace has:

- No primary source.
- Optional attached built-in, library, package, or application-managed sources.
- The same catalog and record semantics as a filesystem Workspace.

## 9. Source attachment roles

Workspace recognizes the following conceptual roles.

### 9.1 Primary

The main project source.

Requirements:

- Exactly one for a filesystem Workspace.
- None for an empty Workspace.
- Enabled while the Workspace is active.

### 9.2 Built-in

Application-provided resources available in the Workspace.

Built-in is a source role, not permission to execute without policy.

### 9.3 Application library

Resources selected from an application-managed library.

### 9.4 Attached package

Resources supplied by an attached distribution or mounted package source.

### 9.5 Overlay

Resources intended to supplement other Workspace sources.

## 10. Supported artifact kinds

Workspace support is explicit. A kind is not supported merely because a source
file can be discovered or parsed.

The current implemented set is:

- Workspace definition, used to load bounded discovery preferences.
- Workspace Context documents.
- Workspace Skills.

Agent, Tool, MCP, Model, Assistant, and other future kinds remain planned.
They must not be presented as current Workspace capability until their
end-to-end consumer path exists.

Every supported kind requires:

- Portable schema identity.
- Native source conventions or another discovery path.
- Semantic validation and stable diagnostics.
- Local record derivation rules.
- A management projection.
- A runtime or product consumer where runtime use is part of the kind.
- Explicit dependency support or an explicit prohibition on dependencies.

Adding a decoder without these responsibilities is incomplete implementation,
not supported Workspace capability.

## 11. Native source conventions

Workspace may recognize conventions such as:

- A Workspace configuration document
- Project-level instruction documents
- Project context documents
- Skill Markdown with frontmatter
- MCP configuration documents
- Structured agent definitions
- Structured model definitions
- Structured tool definitions

### 11.1 Implemented repository conventions

The current source-linked implementation recognizes the following conventions:

| Repository convention                 | Current artifact behavior                                                                                         |
| ------------------------------------- | ----------------------------------------------------------------------------------------------------------------- |
| `.flexigpt/workspace.json`            | Workspace-definition discovery preferences. It is a bootstrap document, not a general runtime artifact.           |
| `AGENTS.md`                           | A Context document with the `agent-instructions` role.                                                            |
| `CLAUDE.md`                           | A Context document with the `assistant-instructions` role.                                                        |
| `README.md`                           | A Context document only when README discovery is enabled.                                                         |
| `<configured-skill-root>/**/SKILL.md` | A Workspace Skill. The default configured root is `.skills`; skill roots are configurable at Workspace open time. |
| Explicit additional Markdown locators | A Context document with the `project-context` role when explicitly targeted by discovery preferences.             |

Context filenames are currently a static code-level convention registry.
Skill roots use a dedicated convention registry and are configuration-driven.
Both registries should remain the extension points for adding conventions
rather than adding path tests throughout discovery code.

### 11.2 Deferred repository conventions

The following convention is a requirement for a future MCP increment, but is
not implemented and must not be described as part of the current MVP:

- `.mcps.json` or `.flexigpt/mcp.json`, with one canonical occurrence
  subresource per declared server.

Every future convention must provide:

- Default or explicitly configured discovery scope.
- A native decoder and canonical definition.
- Semantic validation and structured diagnostics.
- Record derivation and catalog visibility.
- A management projection.
- A runtime or product consumer appropriate to the artifact kind.

Repository resources remain source-linked. Discovery must not copy them into an
installed-resource store. Import, capture, and fork remain optional transfer
workflows under `WS-F15`.

## 12. Core use cases

### 12.1 Select a filesystem Workspace

The user chooses a directory.

Workspace:

- Creates a source registration.
- Creates a Workspace root.
- Attaches the source as primary.
- Does not perform implicit discovery during creation.
- Performs discovery only through an explicit refresh request.

Selecting a directory is not an execution approval.

### 12.2 Create an empty Workspace

The user creates a logical Workspace without selecting a directory.

The Workspace may later receive attached sources or app-local resources.

### 12.3 Attach a source

The user or application attaches:

- Built-in resources
- An application library
- A package source
- An overlay source

Workspace validates role rules but does not take over source transport.

### 12.4 Refresh discovery

Workspace determines the effective discovery plan from:

- Built-in Workspace conventions
- Workspace root preferences
- An optional Workspace definition
- Attachment-specific discovery settings

One user-visible refresh should produce one final published Workspace catalog.

If discovery preferences must first be read from the source, that preliminary observation should not become a separate user-visible catalog state.

### 12.5 Synchronize Workspace records

Valid supported occurrences become local Workspace records.

Synchronization:

- Preserves existing local settings.
- Creates records for newly supported occurrences.
- Updates follow-current records.
- Marks missing or invalid records.
- Does not delete records automatically.
- Does not create duplicate records for the same typed occurrence.

### 12.6 Browse the Workspace catalog

The Workspace catalog combines:

- Workspace root data
- Attached source information
- Current catalog occurrences
- Local records
- Canonical definitions
- Optional grouping
- Diagnostics

It should distinguish:

- Valid synchronized resources
- Valid but not synchronized occurrences
- Records without a current source occurrence
- Invalid resources
- Missing resources
- Stale records

### 12.7 Resolve a reference

A Workspace reference may use:

- Explicit local record identity
- Portable selector

Explicit record identity must belong to the Workspace.

Portable selector resolution uses:

- Artifact kind
- Logical name
- Version constraint
- Labels
- Enabled source attachments

No candidate means unresolved. More than one matching candidate means ambiguous.

Workspace must never silently choose by iteration order, filename order, or registration order.

### 12.8 Compose a load plan

A load plan is the validated handoff from Workspace management to runtime integration.

It contains:

- The selected local records
- Their resolved canonical definitions
- Their domain projections
- Resolved dependencies where required
- Relevant diagnostics
- The catalog publication against which the plan was composed

A load plan does not itself start runtime resources.

### 12.9 Use selected Context and Skills in a conversation

The UI may orchestrate Workspace selection, record selection, and conversation
creation. The backend remains responsible for converting that selection into
normal runtime inputs.

- Selected Context records must be composed into the normal instruction input.
- Selected Workspace Skills must be usable through the same runtime contract as
  installed Skills.
- The UI must not need to resolve Skill resource paths, materialize source
  trees, invoke scripts directly, or emulate Skill session behavior.
- Backend runtime APIs must return structured diagnostics for denied,
  unavailable, stale, invalid, or unsupported selected resources.

## 13. Functional requirements

| ID       | Requirement                                                                           | Priority |
| -------- | ------------------------------------------------------------------------------------- | -------- |
| `WS-F01` | Create filesystem and empty Workspaces.                                               | Core     |
| `WS-F02` | Maintain exactly one primary source for filesystem Workspaces.                        | Core     |
| `WS-F03` | Attach built-in, library, package, and overlay sources.                               | Core     |
| `WS-F04` | Discover supported artifacts using Workspace conventions.                             | Core     |
| `WS-F05` | Allow Workspace configuration to extend discovery scope safely.                       | Core     |
| `WS-F06` | Publish one coherent catalog per user-visible refresh.                                | Core     |
| `WS-F07` | Create and preserve stable local records.                                             | Core     |
| `WS-F08` | Group resources by supported artifact kind for management views.                      | Core     |
| `WS-F09` | Project definitions into existing domain-facing models.                               | Core     |
| `WS-F10` | Resolve explicit and selector-based references deterministically.                     | Core     |
| `WS-F11` | Compose validated load plans.                                                         | Core     |
| `WS-F12` | Expose invalid, missing, stale, and ambiguous states.                                 | Core     |
| `WS-F13` | Keep local setup, trust, and secrets outside portable definitions.                    | Core     |
| `WS-F14` | Integrate with existing installed-resource providers without duplicating persistence. | Core     |
| `WS-F18` | Insert selected Workspace Context through the normal model instruction flow.          | Core     |
| `WS-F19` | Include selected Workspace Skills in the normal Skill runtime and session flow.       | Core     |
| `WS-F20` | Provide selected Workspace Skill resource and script capability parity.               | Core     |
| `WS-F15` | Import, capture, or fork Workspace resources.                                         | Optional |
| `WS-F16` | Materialize source trees for path-only runtime libraries.                             | Optional |
| `WS-F17` | Persist dependency resolution history.                                                | Optional |

## 14. Quality requirements

### 14.1 Predictable precedence

The same attachments, catalog, and selector must produce the same resolution.

### 14.2 Refresh coherence

A Workspace catalog must not combine resources from different incompatible source observations.

### 14.3 Local-state preservation

Refresh must not overwrite:

- User enablement
- Local setup references
- Explicit pinning
- User organization choices

### 14.4 Source transparency

Users should be able to determine:

- Which source supplied a resource
- Which locator produced it
- Whether it is current
- Why it was selected
- Why another candidate lost or tied

### 14.5 Safe degradation

One invalid resource should not make unrelated valid resources disappear.

A structural failure affecting the entire Workspace may block publication, but candidate-specific validation failures should remain candidate-specific.

### 14.6 Runtime isolation

Loading a Workspace catalog must not automatically:

- Connect to MCP servers
- Start processes
- Invoke tools
- Load model credentials
- Execute Skill content
- Modify installed stores

## 15. Workspace discovery model

### 15.1 Base discovery

Workspace defines a small set of known bootstrap locations and directories.

The purpose of bootstrap discovery is only to locate:

- Workspace configuration
- Standard project artifacts
- Standard attached-source layouts

### 15.2 Expanded discovery

A valid Workspace definition may add:

- Explicit locators
- Additional roots
- Include patterns
- Readme inclusion
- Consumer-approved discovery preferences

Expanded discovery must remain bounded by global safety limits.

### 15.3 Effective plan

The effective plan is the merge of:

- Product defaults
- Root-local preferences
- Workspace definition preferences
- Attachment-local preferences
- Supported format capabilities

Conflicting preferences must have a defined merge rule. They must not depend on map or registration order.

### 15.4 Authoritative scope

A discovery scope is authoritative only for the locations and formats it explicitly owns.

Resources outside that scope must not be marked missing.

## 16. Record model

A Workspace record provides:

- Stable local identity
- Workspace membership
- Artifact kind
- Source occurrence identity
- Current or pinned definition
- Enabled state
- Local data
- State and diagnostics

For the initial release, Workspace needs only two content relationships:

- Follow source
- Pin definition

Manual refresh, capture, fork, embedded overlay, and app-local modes should exist only when a committed user workflow requires them.

## 17. Grouping model

Workspace management screens need resources grouped by artifact kind.

This can be represented as:

- Derived views based on root and artifact kind, or
- Persistent collections when compatibility requires stable bundle identity

Persistent collections should be retained only if an existing external contract genuinely requires stable collection IDs.

A portable package must not automatically be treated as a Workspace collection.

## 18. Projection model

Projection converts:

- Workspace context
- Local record
- Canonical definition

into an existing domain-facing representation.

Examples include:

- Skill management model
- Tool management model
- MCP server setup model
- Model preset model
- Agent preset model
- Instruction contributor
- Context contributor

Projection requirements:

- The definition kind and schema must match the projector.
- App-local IDs are supplied by the record, not read from the portable definition.
- Local enabled and built-in state come from Workspace context.
- Portable definitions must not contain persisted legacy store identities.
- Projection errors remain diagnostics associated with the resource.

## 19. Existing store integration

Existing stores should remain authoritative for globally installed resources.

Workspace should not copy projected resources into those stores.

The target provider model is:

- Installed Skill provider
- Workspace Skill selection resolver
- Installed Tool provider
- Workspace Tool provider
- Installed MCP provider
- Workspace MCP provider
- Installed Model provider
- Workspace Model provider
- Installed Assistant provider
- Workspace Agent provider

A scoped aggregate service merges these providers for a specific user action or conversation.

This prevents:

- Duplicate persistence
- Conflicting identities
- Synchronization loops
- Workspace cleanup mutating global stores
- Source changes being hidden behind copied installed records

A resource occurrence must have one authoritative persistence owner.

### 19.1 Selected Workspace Skill runtime parity

For Workspace Skills, "integration with the existing Skill domain" means
runtime parity, not only management projection or Markdown rendering. A
filesystem-backed Workspace Skill is an ordinary filesystem Skill package at
runtime.

The UI may provide a selected set of installed Skill references and Workspace
record identities. The backend must create or configure a conversation-scoped
Skill runtime in which both origins participate through the same normal Skill
operations:

- Session creation and closure.
- Available and active Skill listing.
- Skill prompt generation.
- Skill activation and unload through the normal Skill tools.
- Skill body rendering with arguments and defaults.
- Source-relative resource indexing and `skills-readresource`.
- Script execution through `skills-runscript`.
- Existing insert behavior, argument behavior, warnings, and resource metadata.

Workspace Skill resource and script behavior must follow the same effective
filesystem-provider policy as installed Skills. This does not authorize
execution during discovery, refresh, catalog browsing, management projection,
listing, or ordinary load-plan composition.

For a filesystem-backed Workspace Skill, runtime handoff must:

- Resolve the Skill root from the Workspace occurrence rather than from a
  portable definition path.
- Preserve record identity, definition digest, source provenance, enablement,
  pinning, and catalog-current checks.
- Apply runtime policy before the Skill enters a session.
- Verify that the selected `SKILL.md` still matches the current catalog
  occurrence before registration.
- Keep source path resolution private and never expose it through Workspace API
  projections.
- Keep filesystem paths, source configuration, credentials, and cache paths
  outside portable definitions and Wails projections.
- Return structured diagnostics when resource or script capability is denied or
  unavailable.

Non-filesystem Workspace sources do not gain a synthetic filesystem runtime
path. They remain unavailable for filesystem Skill runtime handoff unless a
future explicit source materialization capability is introduced.

Workspace Skills must not be copied into the installed Skill Store merely to
obtain runtime behavior. A normal Skill runtime participant and an installed
Skill Store record are separate concepts.

## 20. Dependency and reference model

Portable definitions use selectors rather than local references.

Workspace currently resolves selectors against available, recorded resources in
the current catalog. It does not apply attachment-role precedence.

Current resolution rules are:

1. Filter by artifact kind.
2. Apply logical name when present.
3. Apply the currently supported exact version comparison when present.
4. Apply label requirements.
5. Remove disabled, unavailable, stale, unresolved, and projection-invalid
   records.
6. Select only when exactly one candidate remains.
7. Return an unresolved or ambiguous error otherwise.

If role-based precedence is introduced later, its ordered rules, tie behavior,
and diagnostic projection must be defined before it is used for selection.

Dependency graph construction is needed only for artifact kinds that declare dependencies and for product flows that consume the graph.

Persisting every dependency graph is not required for ordinary runtime composition.

## 21. Runtime policy and secrets

Workspace content is untrusted until runtime policy approves its use.

Approval and trust decisions belong to the runtime-policy implementation. They
are not persisted in Workspace root data or portable definitions.

### 21.1 Secrets

Portable definitions must not contain:

- Secret values
- Local credential IDs
- Local file paths to secrets
- Persisted runtime tokens

A portable definition may declare that a credential is required. Local runtime setup resolves the requirement.

### 21.2 Process and network configuration

Commands, URLs, and arguments may be declarative artifact data, but execution remains subject to:

- Trust
- User approval
- Policy
- Environment restrictions
- Secret resolution

## 22. Failure behavior

- An invalid Workspace root cannot be used as a Workspace.
- A filesystem Workspace without exactly one valid primary source is invalid.
- A failed refresh preserves the prior coherent catalog.
- Invalid candidates remain visible with diagnostics.
- Missing candidates leave their records in a missing state.
- An unresolved reference blocks only the dependent load plan.
- An ambiguous reference must not be silently resolved.
- A projection failure prevents that resource from entering a runtime load plan.
- A concurrent catalog change invalidates a load plan being composed.
- A runtime setup failure does not mutate the catalog or portable definition.
- A selected Workspace Skill that cannot enter the normal Skill runtime remains
  visible and returns an unavailable diagnostic rather than silently degrading
  into a text-only Skill.
- Resource or script denial is a runtime-policy result, not a parsing failure.

## 23. Architectural decisions

### 23.1 Workspace is a typed Artifact Store consumer

Reason:

- Workspace needs the generic source, definition, catalog, and record lifecycle.
- A second Workspace database would duplicate identity and synchronization.

### 23.2 Workspace owns conventions, not source access

Reason:

- Workspace knows which paths and formats matter.
- Artifact Store knows how to traverse a registered source safely.

### 23.3 Workspace records are not installed-store records

Reason:

- Workspace resources are contextual and source-linked.
- Installed resources have a different lifecycle and authority.

### 23.4 Runtime providers merge resources at read time

Reason:

- Copying Workspace resources into existing stores creates dual authority.
- Read-time provider composition preserves origin and lifecycle.

### 23.5 Reference precedence is explicit and diagnosable

Reason:

- Multi-source Workspaces need deterministic behavior.
- Silent first-match behavior is unsafe.

### 23.6 One refresh produces one final catalog

Reason:

- Bootstrap and expanded discovery are parts of one user intent.
- Publishing two successive catalog generations for one refresh complicates consistency and user understanding.

### 23.7 Workspace support for a kind requires an end-to-end path

A kind is supported only when all of the following exist:

- Discovery
- Canonical schema
- Validation
- Record derivation
- Management projection
- Runtime or product consumer
- Diagnostics

Parsing a kind without a consumer is incomplete capability, not full support.

### 23.8 Workspace Skills use normal runtime semantics without installed-store copying

Reason:

- Users should not have to understand different Skill behavior based on origin.
- A selected Workspace Skill must support the same normal runtime capabilities
  as a selected installed Skill.
- Copying Workspace Skills into installed persistence would create dual
  authority and synchronization problems.

Consequences:

- The application aggregates installed and Workspace Skill selections at the
  runtime/session boundary.
- Workspace remains authoritative for Workspace record state and provenance.
- Workspace filesystem Skills resolve to ordinary `fs` Agent Skills
  definitions at runtime.
- Runtime identity mapping is ephemeral and consumer-local, not installed-store
  persistence.

## 24. Workspace acceptance outcomes

Workspace is successful when:

- A user selects a project directory and receives a stable Workspace identity.
- Standard project resources appear after one refresh.
- Additional built-in and library sources can be attached.
- Selector ambiguity is deterministic and explainable.
- A changed project resource updates its follow-current record without losing local settings.
- A removed project file leaves a diagnosable missing record.
- Invalid files do not hide unrelated resources.
- Workspace Skills, Tools, MCP servers, Models, and Agents can be projected without being copied into existing stores.
- A runtime-facing service can request a coherent load plan.
- Selected Workspace Context is inserted through the normal instruction flow.
- A selected Workspace Skill has the same user-visible runtime behavior as an
  installed Skill for session activation, prompt generation, rendering,
  resources, and scripts, subject to the same effective policy.
- Secret and trust state remain outside portable definitions.
- Deleting or disabling a Workspace does not delete global installed resources.

## 25. Current implementation status

This mapping assesses the attached implementation, rather than planned architecture.

- `Implemented` means the required behavior has a concrete implementation in the current code.
- `Partial` means that a foundation or subset exists, but the full HLD requirement is not yet met.
- `Not implemented` means no implementation currently provides the requirement.

The current implementation provides Workspace discovery, canonical definitions,
record reconciliation, cataloging, Context composition, Workspace-specific
Skill views, stable Workspace Skill identities, and a private filesystem
handoff path for eligible Skill records.

Normal Workspace Skill runtime participation remains partial. The runtime has
identity-resolution and reconciliation code, but its current synchronization
selection is inverted: it selects runtime-disabled records and the runtime
policy rejects those same records. Consequently, no Workspace Skill reaches
the desired runtime set through normal synchronization.

### 25.1 Mandatory prerequisite mapping

| HLD prerequisite                    | Status            | Current state and notes                                                                                                                                                                                                                          |
| ----------------------------------- | ----------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Logical roots                       | `Implemented`     | Workspace uses typed Artifact Store roots with stable UUIDv7 identities and logical deletion.                                                                                                                                                    |
| Source registrations                | `Implemented`     | Filesystem and embedded source registrations are managed through Artifact Store source services.                                                                                                                                                 |
| Root-to-source attachments          | `Implemented`     | Workspace creates and manages primary and additional source attachments with root-local roles, enablement, and data.                                                                                                                             |
| Safe bounded discovery              | `Implemented`     | Workspace plans use bounded Artifact Store discovery with candidate, entry, depth, and byte limits.                                                                                                                                              |
| Native format adapters              | `Partial`         | Workspace has concrete Workspace-definition JSON, Context Markdown, and `SKILL.md` decoders. MCP, Tool, Model, Agent, and Assistant decoders are not implemented.                                                                                |
| Canonical definitions               | `Implemented`     | Workspace definitions and any configured decoder output are normalized into Artifact Store canonical definitions.                                                                                                                                |
| Current root catalog publication    | `Implemented`     | Workspace refresh delegates to the atomic Artifact Store current-catalog publisher.                                                                                                                                                              |
| Stable local records                | `Implemented`     | Workspace record policy derives stable local records and refresh reconciliation preserves local record fields.                                                                                                                                   |
| Structured diagnostics              | `Implemented`     | Workspace catalog, Context, Skill, load, and record projections expose catalog, occurrence, record, semantic-projection, policy, exclusion, and truncation diagnostics.                                                                          |
| Projection boundary                 | `Partial`         | Context projects into a composition plan. Skills project into Workspace-specific views and can produce a private filesystem runtime location, but no completed general domain projection exists.                                                 |
| Workspace runtime provider contract | `Partial`         | Context composition exists but is not wired into inference instruction assembly. Skill identity and handoff infrastructure exists, but the current synchronization gate prevents normal Workspace Skill registration.                            |
| Aggregate resource services         | `Partial`         | Installed and Workspace Skill providers can be aggregated for management listing and rendering. Normal runtime APIs accept stable Workspace identities, but the current Workspace synchronization defect prevents completed session integration. |
| Runtime policy boundary             | `Partial`         | `RecordRuntimePolicy` enforces Workspace and record enablement plus record-local runtime disablement. It does not yet represent a broader approval or execution-policy service.                                                                  |
| Secret resolution boundary          | `Not implemented` | No local secret-reference resolution service or runtime setup model is implemented. Current Workspace JSON does not model secret values.                                                                                                         |
| User-visible diagnostics            | `Implemented`     | API-safe Wails projections expose resource, occurrence, record, policy, unavailable, truncation, exclusion, and catalog diagnostics. Frontend presentation remains a UI concern.                                                                 |

### 25.2 Functional requirement mapping

| ID       | Requirement                                                                           | Status            | Current state and notes                                                                                                                                                                                                                                                                      |
| -------- | ------------------------------------------------------------------------------------- | ----------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `WS-F01` | Create filesystem and empty Workspaces.                                               | `Implemented`     | `Service.CreateFilesystem`, `Service.CreateEmpty`, and the filesystem provisioner create the required root and source relationships.                                                                                                                                                         |
| `WS-F02` | Maintain exactly one primary source for filesystem Workspaces.                        | `Implemented`     | Filesystem Workspace validation requires one enabled primary filesystem attachment matching `PrimarySourceID`. Primary attachments cannot be attached, detached, or replaced through Workspace attachment operations.                                                                        |
| `WS-F03` | Attach built-in, library, package, and overlay sources.                               | `Partial`         | Role validation and attachment persistence are implemented. The public Workspace API can attach an existing source, but it does not yet provide source provisioning for built-in, library, package, or overlay content. Package transport and distribution are absent.                       |
| `WS-F04` | Discover supported artifacts using Workspace conventions.                             | `Partial`         | Workspace definition JSON, `AGENTS.md`, `CLAUDE.md`, optional `README.md`, configured Skill roots, and explicitly targeted Markdown Context files are implemented. MCP and all other planned artifact kinds are absent.                                                                      |
| `WS-F05` | Allow Workspace configuration to extend discovery scope safely.                       | `Implemented`     | `.flexigpt/workspace.json` can add locators, roots, patterns, and README inclusion. Preferences are validated, merged deterministically, and passed through bounded discovery. Workspace YAML is not supported.                                                                              |
| `WS-F06` | Publish one coherent catalog per user-visible refresh.                                | `Implemented`     | The preliminary Workspace-definition read does not publish a catalog. The subsequent Artifact Store refresh produces one final publication.                                                                                                                                                  |
| `WS-F07` | Create and preserve stable local records.                                             | `Implemented`     | Record reconciliation creates one local record per supported typed occurrence and preserves local state across source-derived updates.                                                                                                                                                       |
| `WS-F08` | Group resources by supported artifact kind for management views.                      | `Implemented`     | Public catalog responses expose artifact-kind groups, recorded resources, and detailed unrecorded occurrences.                                                                                                                                                                               |
| `WS-F09` | Project definitions into existing domain-facing models.                               | `Partial`         | Context projects into Workspace contributions. Skills project into Workspace-specific summaries and load views. The remaining planned kinds have no projections.                                                                                                                             |
| `WS-F10` | Resolve explicit and selector-based references deterministically.                     | `Partial`         | Explicit record resolution, logical-name matching, labels, and ambiguity errors exist. Version constraints only support exact equality, resolution considers recorded resources only, and there is no explainable candidate result.                                                          |
| `WS-F11` | Compose validated load plans.                                                         | `Partial`         | Context and Skill load views validate Workspace records and definitions, but selected Context is not yet shown entering the normal model instruction flow and selected Skills do not enter the normal Skill runtime/session flow.                                                            |
| `WS-F12` | Expose invalid, missing, stale, and ambiguous states.                                 | `Partial`         | Catalog and record views expose occurrence state, record state, diagnostics, catalog currentness, unrecorded occurrences, and unresolved records. Selector ambiguity is returned as an engine error, not an explainable public resolution result.                                            |
| `WS-F13` | Keep local setup, approval, and secrets outside portable definitions.                 | `Partial`         | Record-local runtime disablement is separate from definitions, and implemented Context and Skill schemas define no credential fields. There is no general secret-reference service, approval model, or enforcement that arbitrary Markdown and frontmatter cannot contain sensitive content. |
| `WS-F14` | Integrate with existing installed-resource providers without duplicating persistence. | `Partial`         | Installed and Workspace Skills retain separate persistence ownership and have stable aggregate-provider identities. The runtime registration defect means the normal shared session path is not complete.                                                                                    |
| `WS-F15` | Import, capture, or fork Workspace resources.                                         | `Not implemented` | No import, capture, fork, source-copy, package, or transfer-provenance workflow exists.                                                                                                                                                                                                      |
| `WS-F16` | Materialize source trees for path-only runtime libraries.                             | `Not implemented` | Workspace can provide source metadata in a load plan but cannot materialize a safe source tree or resource closure.                                                                                                                                                                          |
| `WS-F17` | Persist dependency resolution history.                                                | `Not implemented` | No dependency graph, resolution history, or historical catalog integration exists.                                                                                                                                                                                                           |
| `WS-F18` | Insert selected Workspace Context through the normal model instruction flow.          | `Partial`         | `ComposeWorkspaceContext` can return a Workspace prompt contribution, but the backend does not yet show a normal conversation instruction assembly integration boundary.                                                                                                                     |
| `WS-F19` | Include selected Workspace Skills in the normal Skill runtime and session flow.       | `Partial`         | Stable Workspace identities, runtime lookup, and filesystem handoff infrastructure exist. The current synchronization path selects `RuntimeDisabled` records, and the runtime policy rejects those records, leaving the desired Workspace runtime set empty.                                 |
| `WS-F20` | Provide selected Workspace Skill resource and script capability parity.               | `Partial`         | A valid selected filesystem Skill can be projected to the shared filesystem provider path, which is the intended parity mechanism. Because `WS-F19` registration is currently blocked, normal session resource and script parity is not complete.                                            |

### 25.3 Quality requirement mapping

| HLD section | Requirement              | Status        | Current state and notes                                                                                                                                                                                                                                                                                    |
| ----------- | ------------------------ | ------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `14.1`      | Predictable precedence   | `Partial`     | Selector resolution is deterministic because multiple matching resources are ambiguous. There is no attachment-role precedence policy, complete version matching, or explainable winner and loser projection.                                                                                              |
| `14.2`      | Refresh coherence        | `Implemented` | Artifact Store publishes one atomic final catalog. Bootstrap Workspace-definition preferences are bound to the primary-source generation expected by final discovery.                                                                                                                                      |
| `14.3`      | Local-state preservation | `Implemented` | Refresh reconciliation preserves enablement, local data, pins, names, and local record identity while updating only source-derived state.                                                                                                                                                                  |
| `14.4`      | Source transparency      | `Partial`     | Catalog views expose source IDs, locators, occurrences, record state, currentness, and diagnostics. They do not expose selector candidate evaluation, precedence, or rejection explanations.                                                                                                               |
| `14.5`      | Safe degradation         | `Implemented` | Candidate-specific decode and definition failures produce invalid occurrences while unrelated valid candidates continue. Structural failures prevent publication and preserve the prior catalog.                                                                                                           |
| `14.6`      | Runtime isolation        | `Implemented` | Discovery, catalog reads, and Context composition do not execute content, connect external services, load credentials, or modify installed persistence. Lifecycle-triggered runtime synchronization may inspect and register selected runtime definitions, but does not create sessions or execute Skills. |

## 26. Code guide

This guide describes module-level responsibilities for code analysis. It is not
an API contract and intentionally avoids describing individual structs.

- `internal/workspace`
  - Public Workspace feature boundary, API-safe request and response
    projections, configuration, and feature composition.
  - This is the boundary used by HTTP, CLI, and Wails callers.
- `internal/workspace/engine`
  - Workspace root semantics, attachment-role policy, discovery planning,
    bootstrap definition loading, record policy, query resolution, catalog
    projection, and runtime-policy decisions.
  - Owns Workspace meaning; it does not own generic storage or source access.
- `internal/workspace/contextadapter`
  - Context conventions, Markdown decoding, semantic validation, inspection,
    policy-aware prompt composition, ordering, and prompt-budget handling.
- `internal/workspace/skilladapter`
  - Skill conventions, `SKILL.md` decoding, semantic validation, management
    projection, selected Skill loading, and private filesystem handoff.
  - It does not own Agent Skills runtime registration, sessions, or execution.
- `internal/workspace/provision`
  - Filesystem Workspace provisioning and compensation when root creation
    fails after source creation.
- `internal/artifactstore`
  - Upstream generic dependency providing roots, sources, discovery,
    definitions, catalogs, records, and refresh publication.
- `internal/skillruntime`
  - Downstream runtime consumer for installed and Workspace Skill identities,
    runtime reconciliation, sessions, prompts, rendering, and tool calls.
- `cmd/agentgo`
  - Application composition and Wails wrappers.
  - `wrapper_workspace.go` invokes Workspace lifecycle APIs and synchronizes
    Workspace runtime state. `wrapper_skill.go` exposes installed, aggregate,
    and normal runtime APIs.

Dependency direction is intentional:

- `workspace` depends on `artifactstore`.
- `skilladapter` is the Workspace-owned bridge to selected source-linked
  Skills.
- `skillruntime` consumes the Workspace Skill adapter through application
  composition.
- Artifact Store must not import Workspace or Skill runtime packages.

- Complete selected Workspace Context integration.
  - Define the application instruction assembly boundary that accepts selected
    Context record IDs and inserts `ComposeWorkspaceContext` output into the
    normal model instruction input.
  - Preserve Context ordering, provenance, budgets, truncation, exclusion, and
    runtime-policy diagnostics through the conversation request.

- Repair and verify Workspace Skill runtime integration before expanding Skill
  capability.
  - Correct runtime synchronization to select enabled, available,
    catalog-current records whose `RuntimeDisabled` value is `false`.
  - Add tests proving that a stable Workspace identity appears in
    `CreateSkillSession`, `GetSkillsPrompt`, `ListRuntimeSkills`,
    `RenderSkill`, and `InvokeSkillTool`.
  - Verify that refreshes, record enablement changes, pin and follow changes,
    attachment changes, and Workspace deletion reconcile the normal runtime
    deterministically.
  - Keep runtime locations private. Wails and frontend code must persist only
    stable installed or Workspace identities.

- Complete attached-source provisioning.
  - Add an application-level way to register or select sources intended for
    built-in, library, package, and overlay attachment roles.
  - Keep source transport ownership outside Workspace role validation.

- Implement MCP only after Context integration and full Skill runtime parity.
  - Add decoders for `.flexigpt/mcp.json` and `.mcps.json`.
  - Emit one canonical definition and occurrence subresource per server.
  - Add semantic validation that excludes secret values and machine-local credential IDs from portable definitions.
  - Add app-local MCP setup records and a read-only Workspace MCP provider.
  - Do not start or connect MCP servers during discovery, catalog browsing, or load-plan composition.

- Close remaining cross-kind resolution gaps.
  - Add complete version-constraint semantics.
  - Return explainable reference resolution results containing selected, rejected, unavailable, and tied candidates.
  - Surface structural refresh errors as root-scoped diagnostics where persistence is appropriate.
  - Add local secret resolution only when an artifact kind such as MCP requires it.

- Keep migration and optional capabilities deferred.
  - Do not migrate or remove the existing Skill Store. Runtime aggregation must
    coexist with it without making it the persistence owner of Workspace Skills.
  - Defer packages, import, capture, fork, generic source materialization, dependency history, and historical catalogs until concrete workflows require them.
