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
- Which resource wins when several sources define the same logical artifact?
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
| Consumer-owned precedence         | Workspace must decide which attached source wins.                                |
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
- Workspace source precedence.
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
- Attached-source preferences
- Trust reference
- Capability profile version
- Display preferences

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
- Highest default precedence.

### 9.2 Built-in

Application-provided resources available in the Workspace.

Built-in is a source role, not permission to execute without policy.

### 9.3 Application library

Resources selected from an application-managed library.

### 9.4 Attached package

Resources supplied by an attached distribution or mounted package source.

### 9.5 Overlay

Resources intended to override or supplement lower-priority sources.

### 9.6 Priority semantics

Each attachment has an explicit priority.

Priority determines source precedence for selector-based reference resolution.

Role provides meaning and defaults. Priority provides the actual ordering.

A priority tie between otherwise matching candidates remains ambiguous unless a more specific Workspace rule resolves it.

## 10. Supported artifact kinds

The initial Workspace may support:

- Workspace definition
- Agent definition
- Skill definition
- Tool definition
- MCP server definition
- Model definition
- Instruction document
- Context document

Each artifact kind must define:

- Portable schema identity
- Native source formats
- Validation rules
- Local record defaults
- Projection target
- Runtime consumption boundary
- Whether dependencies are allowed

Adding a kind without a projection or runtime consumer should not be considered complete support.

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

### 11.1 Repository resource MVP conventions

The initial repository-resource MVP must support source-linked discovery and loading of the following primary-source conventions:

| Repository convention           | Intended artifact behavior                                                                                            |
| ------------------------------- | --------------------------------------------------------------------------------------------------------------------- |
| `AGENTS.md`                     | A project instruction or context document.                                                                            |
| `CLAUDE.md`                     | A project instruction or context document.                                                                            |
| `.skills/<skill-name>/SKILL.md` | A Skill definition. The containing Skill directory remains the source-relative location for declared Skill resources. |
| `.mcps.json`                    | An MCP configuration document. One document may emit one or more MCP server definition occurrences.                   |

`README.md` may be included only when enabled by Workspace discovery preferences. It is not a required repository-context convention.

The MVP must provide, for each convention:

- Discovery scope that includes the convention by default for a filesystem Workspace.
- A native decoder and portable canonical definition.
- Semantic validation and structured diagnostics.
- Workspace record derivation and catalog visibility.
- A management projection or context contributor.
- A runtime or product consumer appropriate to the artifact kind.

These resources remain source-linked for the MVP. Discovering a repository resource must not copy it into an installed-resource store. Import, capture, and fork remain optional transfer workflows under `WS-F15`.

These are Workspace conventions. they should be maintained and supported that a new file or pattern or convention semantic can be added very easily later. Eg we should be able to add context file names or skill dir names to scan or mcp json to scan very easily.

Artifact Store only provides candidate traversal and format-adapter invocation.

## 12. Core use cases

### 12.1 Select a filesystem Workspace

The user chooses a directory.

Workspace:

- Creates a source registration.
- Creates a Workspace root.
- Attaches the source as primary.
- Performs discovery when requested.
- Returns the Workspace even if optional immediate discovery fails, together with the failure.

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
- Attachment priority

No candidate means unresolved.

Multiple highest-priority candidates mean ambiguous.

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

The same attachments, priorities, catalog, and selector must produce the same resolution.

### 14.2 Refresh coherence

A Workspace catalog must not combine resources from different incompatible source observations.

### 14.3 Local-state preservation

Refresh must not overwrite:

- User enablement
- Local setup references
- Local trust references
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
- Reapply Workspace trust and runtime policy before the Skill enters a session.
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

Workspace resolves selectors against its current catalog.

Resolution rules are:

1. Filter by artifact kind.
2. Apply logical name when present.
3. Apply version constraint when present.
4. Apply label requirements.
5. Remove disabled or unavailable source candidates.
6. Compare attachment priorities.
7. Select only when exactly one candidate has highest precedence.
8. Return ambiguity otherwise.

Dependency graph construction is needed only for artifact kinds that declare dependencies and for product flows that consume the graph.

Persisting every dependency graph is not required for ordinary runtime composition.

## 21. Trust and secrets

Workspace content is untrusted until runtime policy approves its use.

### 21.1 Trust reference

A Workspace may hold an app-local reference to trust state.

That reference:

- Is not portable.
- Does not itself imply approval.
- Is evaluated by a separate trust or policy component.

### 21.2 Secrets

Portable definitions must not contain:

- Secret values
- Local credential IDs
- Local file paths to secrets
- Persisted runtime tokens

A portable definition may declare that a credential is required. Local runtime setup resolves the requirement.

### 21.3 Process and network configuration

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

Consequence:

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
- Source precedence is deterministic and explainable.
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
record reconciliation, cataloging, Context composition, and Workspace-specific
Skill list and load projections.

It does not yet complete the required normal runtime integration. In
particular, selected Workspace Skills do not enter the existing Skill Store
session/runtime APIs, do not participate in normal runtime prompt generation,
and do not support normal Skill resource or script operations. The current
Workspace Skill load projection is not equivalent to a normal runtime Skill.

### 25.1 Mandatory prerequisite mapping

| HLD prerequisite                    | Status            | Current state and notes                                                                                                                                                                        |
| ----------------------------------- | ----------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Logical roots                       | `Implemented`     | Workspace uses typed Artifact Store roots with stable UUIDv7 identities and logical deletion.                                                                                                  |
| Source registrations                | `Implemented`     | Filesystem and embedded source registrations are managed through Artifact Store source services.                                                                                               |
| Root-to-source attachments          | `Implemented`     | Workspace creates and manages primary and additional source attachments with root-local roles, priorities, enablement, and data.                                                               |
| Safe bounded discovery              | `Implemented`     | Workspace plans use bounded Artifact Store discovery with candidate, entry, depth, and byte limits.                                                                                            |
| Native format adapters              | `Partial`         | Workspace has concrete JSON Workspace-definition, Markdown Context-document, and restricted-YAML `SKILL.md` decoders. MCP, Tool, Model, and Agent decoders remain unimplemented.               |
| Canonical definitions               | `Implemented`     | Workspace definitions and any configured decoder output are normalized into Artifact Store canonical definitions.                                                                              |
| Current root catalog publication    | `Implemented`     | Workspace refresh delegates to the atomic Artifact Store current-catalog publisher.                                                                                                            |
| Stable local records                | `Implemented`     | Workspace record policy derives stable local records and refresh reconciliation preserves local record fields.                                                                                 |
| Structured diagnostics              | `Implemented`     | Workspace catalog, Context, Skill, load, and record projections expose catalog, occurrence, record, semantic-projection, policy, exclusion, and truncation diagnostics.                        |
| Consumer-owned precedence           | `Partial`         | Workspace owns attachment-priority resolution and reports ties. Version constraint support is exact-match only and selection explanations are limited.                                         |
| Projection boundary                 | `Partial`         | Context projects into a Workspace composition plan. Skills project into Workspace-specific summaries and Markdown load views, but not into the normal Skill runtime provider/session contract. |
| Workspace runtime provider contract | `Partial`         | Context has a composition API, but no application-level instruction insertion boundary is shown. Workspace Skills have a load projection but no session, runtime, resource, or script handoff. |
| Aggregate resource services         | `Not implemented` | Installed and Workspace Skills are not yet combined into one normal conversation-scoped Skill runtime. A read-only list or render adapter is not sufficient runtime aggregation.               |
| Trust boundary                      | `Partial`         | Workspace metadata can retain app-local trust information, but there is no full Skill runtime handoff at which resource and script policy can be consistently enforced.                        |
| Secret resolution boundary          | `Not implemented` | No local secret-reference resolution service or runtime setup model is implemented. Current Workspace JSON does not model secret values.                                                       |
| User-visible diagnostics            | `Implemented`     | API-safe Wails projections expose resource, occurrence, record, policy, unavailable, truncation, exclusion, and catalog diagnostics. Frontend presentation remains a UI concern.               |

### 25.2 Functional requirement mapping

| ID       | Requirement                                                                           | Status                                  | Current state and notes                                                                                                                                                                                                                                       |
| -------- | ------------------------------------------------------------------------------------- | --------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `WS-F01` | Create filesystem and empty Workspaces.                                               | `Implemented`                           | `Service.CreateFilesystem`, `Service.CreateEmpty`, and the filesystem provisioner create the required root and source relationships.                                                                                                                          |
| `WS-F02` | Maintain exactly one primary source for filesystem Workspaces.                        | `Implemented`                           | Filesystem Workspace validation requires one enabled primary filesystem attachment matching `PrimarySourceID`. Primary attachments cannot be attached, detached, or replaced through Workspace attachment operations.                                         |
| `WS-F03` | Attach built-in, library, package, and overlay sources.                               | `Implemented`                           | Workspace validates and persists the `built-in`, `library`, `attached-package`, and `overlay` roles. Package transport and package distribution itself remain unimplemented.                                                                                  |
| `WS-F04` | Discover supported artifacts using Workspace conventions.                             | `Partial`                               | `AGENTS.md`, `CLAUDE.md`, optional `README.md`, and `.skills/**/SKILL.md` have concrete decoders and records. Context and Skill convention matching still needs centralized declarative configuration, and MCP plus the remaining artifact kinds are pending. |
| `WS-F05` | Allow Workspace configuration to extend discovery scope safely.                       | `Implemented`                           | `.flexigpt/workspace.json` can add locators, roots, patterns, and README inclusion. Preferences are validated, merged deterministically, and passed through bounded discovery. Workspace YAML is not supported.                                               |
| `WS-F06` | Publish one coherent catalog per user-visible refresh.                                | `Implemented`                           | The preliminary Workspace-definition read does not publish a catalog. The subsequent Artifact Store refresh produces one final publication.                                                                                                                   |
| `WS-F07` | Create and preserve stable local records.                                             | `Implemented`                           | Record reconciliation creates one local record per supported typed occurrence and preserves local state across source-derived updates.                                                                                                                        |
| `WS-F08` | Group resources by supported artifact kind for management views.                      | `Implemented`                           | Public catalog responses expose artifact-kind groups, recorded resources, and detailed unrecorded occurrences.                                                                                                                                                |
| `WS-F09` | Project definitions into existing domain-facing models.                               | `Partial`                               | Context projects into Workspace contributions. Skills project into Workspace-specific summaries and Markdown load views, but not into the normal Skill provider and session runtime model.                                                                    |
| `WS-F10` | Resolve explicit and selector-based references deterministically.                     | `Partial`                               | Explicit record resolution, logical-name matching, labels, attachment priorities, and ambiguity errors exist. Version constraints only support exact equality and there is no detailed resolution explanation.                                                |
| `WS-F11` | Compose validated load plans.                                                         | `Partial`                               | Context and Skill load views validate Workspace records and definitions, but selected Context is not yet shown entering the normal model instruction flow and selected Skills do not enter the normal Skill runtime/session flow.                             |
| `WS-F12` | Expose invalid, missing, stale, and ambiguous states.                                 | `Implemented`                           | Public catalog APIs expose valid, invalid, missing, unrecorded, unresolved, stale, projection-invalid, denied, unavailable, and precedence-ambiguous states with diagnostics.                                                                                 |
| `WS-F13` | Keep local setup, trust, and secrets outside portable definitions.                    | `Implemented` for Context and Skill MVP | Runtime approval and trust references are app-local record/root data. Context and Skill portable definitions contain no credentials, secret values, local paths, or runtime state.                                                                            |
| `WS-F14` | Integrate with existing installed-resource providers without duplicating persistence. | `Implemented` for Skills                | Installed and filesystem-backed Workspace Skills share one Agent Skills runtime while retaining distinct installed and Workspace persistence ownership.                                                                                                       |
| `WS-F16` | Materialize source trees for path-only runtime libraries.                             | `Not implemented`                       | Workspace can provide source metadata in a load plan but cannot materialize a safe source tree or resource closure.                                                                                                                                           |
| `WS-F17` | Persist dependency resolution history.                                                | `Not implemented`                       | No dependency graph, resolution history, or historical catalog integration exists.                                                                                                                                                                            |
| `WS-F18` | Insert selected Workspace Context through the normal model instruction flow.          | `Partial`                               | `ComposeWorkspaceContext` can return a Workspace prompt contribution, but the backend does not yet show a normal conversation instruction assembly integration boundary.                                                                                      |
| `WS-F19` | Include selected Workspace Skills in the normal Skill runtime and session flow.       | `Not implemented`                       | `CreateSkillSession`, `GetSkillsPrompt`, `ListRuntimeSkills`, `RenderSkill`, and `InvokeSkillTool` currently operate through the installed Skill Store only.                                                                                                  |
| `WS-F19` | Include selected Workspace Skills in the normal Skill runtime and session flow.       | `Implemented` for filesystem sources    | Stable Workspace identities resolve to approved filesystem Skill directories and enter the ordinary Agent Skills runtime as `fs` definitions.                                                                                                                 |
| `WS-F20` | Provide selected Workspace Skill resource and script capability parity.               | `Implemented` for filesystem sources    | The shared `fsskillprovider` indexes resources and handles `skills-readresource` and `skills-runscript` under the same policy as installed filesystem Skills.                                                                                                 |

### 25.3 Quality requirement mapping

| HLD section | Requirement              | Status        | Current state and notes                                                                                                                                                                                                                                                       |
| ----------- | ------------------------ | ------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `14.1`      | Predictable precedence   | `Partial`     | Attachment priorities provide deterministic selection and ties are rejected. Complete version constraint semantics and explainable winner and loser reporting are still missing.                                                                                              |
| `14.2`      | Refresh coherence        | `Implemented` | Artifact Store publishes one atomic final catalog. Bootstrap Workspace-definition preferences are bound to the expected primary-source generation used by final discovery.                                                                                                    |
| `14.3`      | Local-state preservation | `Implemented` | Refresh reconciliation preserves enablement, local data, pins, names, and local record identity while updating only source-derived state.                                                                                                                                     |
| `14.4`      | Source transparency      | `Partial`     | Internal resources retain record, source, occurrence, current-catalog status, definition, and diagnostics. Public Workspace catalog responses need detailed occurrence and diagnostic projections, plus explainable selector winner, loser, unavailable, and tied candidates. |
| `14.5`      | Safe degradation         | `Implemented` | Candidate-specific decode and definition failures produce invalid occurrences while unrelated valid candidates continue. Structural failures still prevent publication, as required.                                                                                          |
| `14.6`      | Runtime isolation        | `Implemented` | Current Workspace loading only composes data. It does not connect to MCP servers, start processes, invoke tools, load credentials, execute Skills, or modify installed stores.                                                                                                |

## 26. Next steps

- Complete frontend Workspace Skill orchestration.
  - Refresh a Workspace before selecting discovered Skills.
  - Require explicit `RuntimeAllowed` approval before session selection.
  - Persist stable installed or Workspace identities, never runtime locations.
  - Recreate or update a Skill session after Workspace refresh, record
    enablement changes, runtime approval changes, source attachment changes, or
    Workspace deletion.
  - Render `insert=user-message` Skills through `RenderProvidedSkill`.
  - Use selected instruction Skills through normal session prompt generation.
  - Keep resource paths in `SKILL.md` body documentation rather than injecting
    resource inventories into prompts.

- Complete selected Workspace Context integration.
  - Define the application instruction assembly boundary that accepts selected
    Context record IDs and inserts `ComposeWorkspaceContext` output into the
    normal model instruction input.
  - Preserve Context ordering, provenance, budgets, truncation, exclusion, and
    runtime-policy diagnostics through the conversation request.

- Implement MCP only after Context and full Skill runtime parity are complete.
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
