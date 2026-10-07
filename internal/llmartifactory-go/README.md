# LLM Artifactory Go

> Portable LLM declaration contracts, artifact-family behavior, and composition over Artifactory Go.

LLM Artifactory Go is FlexiGPT's LLM domain layer. It attaches LLM declaration
semantics, artifact-family behavior, and composition to an already-open generic
[Artifactory Go](../artifactory-go/README.md) deployment.

It is not a second Artifact Store. Root, Source, Definition, Artifact,
refresh, package publication, verified resource access, overlays, secret
lifecycle, installation, persistence, and provider ownership remain generic
Artifactory responsibilities.

- [Purpose and boundary](#purpose-and-boundary)
  - [What it is not](#what-it-is-not)
- [Mental model](#mental-model)
- [Declaration contracts and admission](#declaration-contracts-and-admission)
  - [Portable declaration grammar](#portable-declaration-grammar)
  - [Canonical admission](#canonical-admission)
    - [Canonical JSON and YAML declarations](#canonical-json-and-yaml-declarations)
    - [Family source formats](#family-source-formats)
  - [Typed reconstruction](#typed-reconstruction)
- [Composition and capability planning](#composition-and-capability-planning)
  - [Lookup scope](#lookup-scope)
  - [Capability targets](#capability-targets)
  - [Direct membership versus recursive plans](#direct-membership-versus-recursive-plans)
  - [Explicit composition refresh](#explicit-composition-refresh)
- [Artifact-family ownership](#artifact-family-ownership)
  - [Plugin](#plugin)
  - [Tool implementations](#tool-implementations)
  - [Model, Provider, MCP, and local configuration](#model-provider-mcp-and-local-configuration)
- [Operational flows](#operational-flows)
  - [1. Admit source material](#1-admit-source-material)
  - [2. Read and compose committed declarations](#2-read-and-compose-committed-declarations)
  - [3. Materialize verified source content](#3-materialize-verified-source-content)
  - [4. Author managed family content](#4-author-managed-family-content)
  - [5. Install generated built-in content](#5-install-generated-built-in-content)
  - [6. Execute through runtime adapters](#6-execute-through-runtime-adapters)
- [Local state, secrets, and trusted capabilities](#local-state-secrets-and-trusted-capabilities)
- [Application assembly and built-in content](#application-assembly-and-built-in-content)
- [Extending the layer](#extending-the-layer)
  - [Add a declaration family](#add-a-declaration-family)
  - [Add a source format or filename convention](#add-a-source-format-or-filename-convention)
  - [Add a direct capability](#add-a-direct-capability)
- [Compatibility and change checklist](#compatibility-and-change-checklist)

## Purpose and boundary

The layer sits between generic Artifactory and application-specific content,
runtime, and transport code:

```text
application content, topology, registration, runtime adapters, and transport
                                  ↑
                         LLM Artifactory Go
                                  ↑
                           Artifactory Go
```

| Layer                     | Owns                                                                                                                                                                           |
| ------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| **Artifactory Go**        | Generic Root, Source, Definition, Artifact, Overlay, Secret, refresh, verified resource, managed-package, installation, persistence, and provider behavior.                    |
| **LLM Artifactory Go**    | Portable LLM declaration grammar, family contracts, interpretation, typed reconstruction, composition, Plugin membership, family package semantics, and runtime-neutral plans. |
| **Application setup**     | Selected schemas and decoders, supported filenames, Source profiles, package layouts, built-in topology, embedded content, generated catalogs, and startup ordering.           |
| **Runtime and transport** | Inference execution, Go Tool execution, MCP connections, Skill sessions, plaintext secret use, native-path authority, Wails APIs, cancellation, and process lifecycle.         |

The root `llmartifactory.Artifactory` intentionally exposes only the shared LLM
capabilities:

- `Interpretations()` for registered declaration-family interpretation.
- `Composition()` for the shared read-only composition resolver.

It does not mirror the generic Store, open providers, own a database, expose
all family services, or own application topology. Family services are
constructed separately with the narrow generic APIs and immutable support
values that each family actually needs.

### What it is not

LLM Artifactory Go is not:

- A persistence provider, Source driver, or local Store opener.
- A generic package manager or dependency solver.
- A global built-in content registry.
- A generic plugin execution engine.
- An inference, Tool, MCP, Skill, Team, Loop, or Workflow runtime.
- A Wails or transport package.
- An authorization service or sandbox.
- A plaintext secret service for ordinary family APIs.
- An implicit refresh mechanism for normal reads.
- An external URL, Git, package, archive, or command fetcher.

A declaration locator can express more than a local path, but an external
locator remains unresolved until application assembly explicitly supplies an
appropriate locator resolver. The current application registers the local path
resolver; it does not silently fetch external content.

## Mental model

Generic Artifactory owns the source-backed chain:

```text
Root
  └── Source
        └── observed declaration
              └── admitted Definition
                    └── source-backed Artifact
```

LLM Artifactory starts from the current Artifact and its admitted Definition:

```text
Artifact reference
  → current Definition
  → family-owned declaration entry
  → family relationship facts
  → resolved capability targets or a capability plan
```

The values answer different questions:

| Value                 | Meaning                                                                                                         |
| --------------------- | --------------------------------------------------------------------------------------------------------------- |
| **Declaration**       | Portable LLM content such as an Agent, Skill, Tool, MCP server, Plugin, or Workspace.                           |
| **Definition**        | Immutable admitted meaning of one declaration, owned by generic Artifactory.                                    |
| **Artifact**          | Stable typed source-origin identity with source-derived availability and local state.                           |
| **Plugin**            | An LLM declaration containing membership relationships. It is not a generic Store ownership edge.               |
| **Capability target** | A runtime-neutral result of composition: either an actual Artifact or an explicitly supplied direct capability. |

An LLM logical name and logical version are declaration identities. They do not
replace the stable generic Artifact reference:

```text
Root + Source + locator + subresource + artifact kind
```

The generic Artifact remains the authority for availability, source evidence,
local enablement, and its current Definition link.

## Declaration contracts and admission

### Portable declaration grammar

`core/declaration` owns shared declaration grammar, not the meaning of any
specific declaration type.

The registered portable declaration types are:

```text
text
model
model.provider
tool
skill
mcp
mcp.policy
plugin
agent
team
loop
workflow
workspace
```

Each family owns its versioned contract under its own `contract/v1` package.
That package owns its JSON Schema asset, semantic validation, typed document
value, Definition reconstruction, and relationship extraction.

Shared declaration grammar includes:

- `type`, `name`, `displayName`, `description`, `labels`, `locator`, and
  `metadata`.
- Canonical `Entry` representation.
- Named, contained, and selector member forms.
- Shared relationship fields such as `scope`, `overrides`, and `use`.
- Portable locator syntax.
- Member identity and deterministic ordering.
- Header dispatch from `type` and optional `apiVersion` to a complete schema
  key.

A standalone declaration requires a portable name. A selector is the shared
exception: it identifies a target type and source-relative base rather than a
single named declaration.

| Member form   | Shape                                              | Meaning                                                                                                |
| ------------- | -------------------------------------------------- | ------------------------------------------------------------------------------------------------------ |
| **Named**     | `type` and `name`, optionally `locator` or `scope` | Reference another declaration.                                                                         |
| **Contained** | `type`, `name`, and `parameters`                   | Embed a declaration in the containing document. It is emitted as a source-backed subresource Artifact. |
| **Selector**  | `type`, `base`, include/exclude rules              | Select matching Artifacts from a local Source tree.                                                    |

A contained member keeps target-specific fields inside `parameters`; relationship
fields remain on the outer member. A Plugin member remains declaration content:
removing it does not delete the referenced Artifact, and a missing target leaves
an unavailable relationship rather than creating a cascade delete.

Portable declarations do not contain generic Store identity or local state:

- Artifact IDs or Source IDs.
- Definition digests.
- Artifact revisions, enablement, local `Data`, overlays, or secret bindings.
- Generic Store schema IDs or storage keys.
- Plaintext credentials.

`apiVersion` is accepted as declaration dispatch metadata. The canonical
JSON/YAML declaration path resolves the complete schema key first, then removes
`apiVersion` from the canonical Definition body. An omitted version and the
corresponding explicit supported version therefore do not create divergent
portable Definition bodies merely because one author wrote the dispatch hint.

### Canonical admission

There are two supported admission paths.

#### Canonical JSON and YAML declarations

```text
bounded candidate bytes
  → declaration header dispatch
  → complete expected schema key
  → generic expected-key schema validation
  → family semantic interpretation
  → generic Definition admission
  → generic Artifact refresh synchronization
```

The generic schema provider validates the schema key it was given. It does not
interpret `type`, `apiVersion`, LLM family names, or source filenames.

The declaration dispatcher and family interpretation registry own those
decisions. This keeps generic Artifactory independent of LLM vocabulary while
preventing each family from inventing a separate header-dispatch mechanism.

#### Family source formats

Some source formats are not canonical declaration documents. Their adapters
belong to the relevant family:

| Source format                              | Owning adapter                |
| ------------------------------------------ | ----------------------------- |
| `AGENT.md` and declared `*.agent.md` forms | `agent/sourceformat/markdown` |
| `SKILL.md` packages                        | `skill/sourceformat/markdown` |
| Standard MCP `mcpServers` configuration    | `mcp/sourceformat/config`     |
| Supported Markdown Text files              | `text/sourceformat/markdown`  |

Adapters receive bounded candidates and, where needed, bounded sibling-entry
access from Ingest. They do not open native paths, mutate Sources, publish
packages, generate Artifact IDs, or choose application topology.

The application selects supported document names, patterns, and decoder IDs
through immutable support values. A family must not acquire new filename
assumptions merely because one application currently uses a particular layout.

### Typed reconstruction

A family reconstructs its typed document from an already admitted Definition.
It verifies family identity and semantic invariants, but does not rerun schema
execution or recalculate the Definition digest on ordinary reads.

The ownership chain is:

```text
generic schema catalog     → expected-key schema validation
family contract            → semantic validation and typed reconstruction
generic Definition         → immutable canonical identity
generic Artifact           → current Artifact-to-Definition relationship
family service             → family-facing view, plan, or materialization
```

## Composition and capability planning

`core/composition` owns shared read-only declaration composition. It uses the
single resolver returned by `llmartifactory.Artifactory.Composition()`.

It owns:

- Artifact-to-current-Definition resolution.
- Declaration alias traversal.
- Named relationship lookup.
- Local locator resolution through registered locator handlers.
- Selector expansion.
- Relationship availability and ambiguity results.
- Cycle detection and graph limits.
- Direct Plugin membership resolution.
- Recursive capability-plan projection.
- Runtime-neutral capability target representation.

It does not refresh Sources, prepare discovery, publish content, mutate local
state, execute Tools, connect MCP servers, or resolve plaintext secrets.

### Lookup scope

Application assembly supplies the protected built-in Root through
`composition.ScopeBinding`. The portable `builtin` token is not itself a Root
ID or an authorization grant.

| Relationship form        | Lookup behavior                                                                                                   |
| ------------------------ | ----------------------------------------------------------------------------------------------------------------- |
| Unscoped named reference | Current Root first, then the configured built-in Root when distinct, then an eligible direct capability provider. |
| `scope: builtin`         | Configured built-in Root first, then an eligible direct capability provider.                                      |
| Local path locator       | The declaring Artifact's Source through a registered locator resolver.                                            |
| Workspace composition    | Workspace may explicitly bind a directory Source as the composition Source for same-Root resolution.              |

Composition does not scan arbitrary Roots.

A selector validates its local Source base through the generic verified Source
entry API and resolves matching Artifacts from the committed catalog for the
relevant Source. This is a read-only operation: it does not prepare discovery
or refresh the Source.

### Capability targets

A resolved target is one of:

| Target form  | Meaning                                                                                                           |
| ------------ | ----------------------------------------------------------------------------------------------------------------- |
| **Artifact** | The actual source-backed Artifact selected by ordinary resolution.                                                |
| **Direct**   | An application-supplied runtime-neutral capability with provider identity, provider-local identity, and evidence. |

A direct target never fabricates an Artifact, Source binding, Definition,
revision, local state, or lifecycle record. Direct providers are consulted only
when the applicable source-backed Artifact lookup did not resolve a target.

### Direct membership versus recursive plans

Plugin direct membership and recursive capability planning are intentionally
different APIs.

- `Plugin.ResolveDirectMembers` resolves only immediate Plugin members,
  including selector matches and availability.
- `Plugin.ResolveCapabilities` and the family capability APIs recursively
  expand reachable relationships and return a capability plan.

Listing a Plugin does not recursively resolve an Agent's Tools, a Skill's
allowed Tools, an MCP server's Policy, or a Workflow's nodes.

### Explicit composition refresh

`core/composition/refresh` is the explicit refresh-closure owner. It can:

1. Resolve the current reachable declaration graph.
2. Ask the applicable family/application planner for additional discovery
   requirements.
3. Prepare Source discovery additively.
4. Refresh affected Sources.
5. Repeat within configured depth and pass limits.

Normal composition remains read-only. A caller that needs preparation or
refresh must request it explicitly.

## Artifact-family ownership

The family folders are responsibility owners, not a mechanically uniform
template. A family has only the subpackages justified by its behavior.

| Family                      | Current responsibility                                                                                                                                                   |
| --------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| **Text**                    | Text declaration contract, Markdown adaptation, and verified inline/source-backed materialization.                                                                       |
| **Agent**                   | Agent contract, Agent Markdown adaptation, import/export, managed Agent package behavior, and Agent capability projection.                                               |
| **Skill**                   | Skill contract, `SKILL.md` adaptation, managed Skill package layout, verified materialization, and Skill capability planning.                                            |
| **Plugin**                  | Plugin contract, explicit membership policy, direct and reverse membership views, managed Plugin editing, baseline restrictions, and recursive Plugin plans.             |
| **Tool**                    | Built-in Tool views, Tool Plugin gating, Tool package interpretation, and Tool capability preparation. Both `go` and `sdk` implementations are supported.                |
| **Model**                   | Operational Model and Provider APIs, managed Model/Provider package behavior, settings, Provider credentials, default selection, and resolution inputs.                  |
| **Model Provider contract** | `modelprovider/contract/v1` owns the portable `model.provider` declaration contract. The current operational API remains in `model`.                                     |
| **MCP and MCP Policy**      | MCP declaration/source-format behavior, installation-local state, logical secret selectors, policy composition, managed packages, and runtime-neutral server resolution. |
| **Team, Loop, Workflow**    | Their declaration contracts and runtime-neutral planning projections. They do not execute a team, loop, or workflow.                                                     |
| **Workspace**               | Workspace declaration behavior, directory/default-policy selection, effective Workspace checks, and runtime-neutral prompt/Skill/MCP planning.                           |

### Plugin

A Plugin is a `plugin` Artifact with declaration-owned member relationships.
Its membership policy is stored in Plugin declaration metadata and can be:

- `single-type`;
- `mixed-type`; or
- legacy unconstrained compatibility for existing declarations without explicit
  policy metadata.

Family profiles provide the legal member types and forms. For example:

- Agent Plugins contain Agent relationships.
- Skill Plugins contain Skill relationships.
- MCP Plugins can contain MCP and MCP Policy relationships.
- Tool Plugins are read-only generated built-in Plugin declarations.

Plugin membership is not a generic foreign key or lifecycle ownership edge.

### Tool implementations

Tool declarations support two implementation families:

| Implementation | Runtime owner                                                    |
| -------------- | ---------------------------------------------------------------- |
| `go`           | The local Go Tool runtime invokes the declared Go function.      |
| `sdk`          | The inference integration hydrates a provider-native ToolChoice. |

Both are ordinary source-backed Tool Artifacts. Both participate in generated
catalog content, Tool Plugin membership, catalog listing, enablement, and
composition. The local Go Tool runtime must reject SDK Tool invocation because
SDK Tools belong to inference-provider execution, not because SDK Tool
Artifacts are unsupported.

### Model, Provider, MCP, and local configuration

Model and MCP family settings are family-owned semantics built on generic
local-state facilities:

- Mutable Artifact settings use namespaced Artifact `Data`.
- Protected Artifact settings use registered protected overlays.
- Credentials use generic secret bindings.
- Family packages define legal namespaces, slots, and payload shapes.
- Generic Artifact cleanup removes selected local state atomically and queues
  detached physical secret cleanup.

The LLM layer does not place plaintext credentials in Definitions, Artifacts,
catalog projections, generated catalogs, Plugin documents, diagnostics, or
ordinary family views.

## Operational flows

### 1. Admit source material

```text
Source snapshot
  → selected decoder or source-format adapter
  → family contract / interpretation
  → Ingest observation
  → Definition admission
  → Artifact synchronization
  → committed catalog state
```

Generic Refresh owns snapshot lifecycle, publication atomicity, and Artifact
state synchronization. LLM registrations provide declaration meaning; they do
not alter generic refresh behavior.

### 2. Read and compose committed declarations

```text
ArtifactRef
  → Artifact current Definition
  → family typed reconstruction
  → family relationship facts
  → composition resolver
  → direct membership or capability plan
```

Family list APIs normally use the committed generic catalog. They do not
silently prepare Sources, refresh content, publish packages, or resolve
runtime credentials.

Expected relationship failures become explicit composition results where
appropriate:

- unavailable;
- ambiguous; or
- available.

Unexpected infrastructure, integrity, cancellation, and closed-service
failures remain errors.

### 3. Materialize verified source content

Text, Skill, Workspace, and similar consumers use generic Resource APIs when
they need source material rather than only committed Definition metadata.

```text
Artifact
  → verified Resource resolution
  → Source generation and content evidence
  → bounded Source entry/tree reads
  → family materialization
```

A multi-read operation uses one explicit verification session. Native paths are
available only through the separately supplied trusted native-path capability,
at a narrowly defined materialization boundary. A normal family API does not
gain native-path access merely because a runtime integration needs it.

### 4. Author managed family content

Family authoring prepares final bytes and expected definition identity before
generic publication:

```text
family validation and normalization
  → family package layout and expected Definition
  → explicit Source discovery preparation when needed
  → generic ManagePackage publication
  → Source acknowledgment and refresh
  → expected Artifact verification
```

`Source.PrepareDiscovery` is additive preparation only. It does not hide a
refresh.

Physical package publication and catalog publication remain distinct commit
boundaries. A known physical change followed by a later failure remains a
recoverable generic ManagePackage outcome, not an invisible rollback.

Plugin membership is declaration content and can have its own observable
publication boundary. For example, managed Agent import deliberately preserves
the distinction between membership publication and package publication.

### 5. Install generated built-in content

Application-owned built-in content is compiled through the ordinary generic
Store admission path:

```text
embedded family packages
  → family package preparation
  → generic compilation and admission
  → generated package catalog
  → startup registration and hydration
  → source-content witness verification
```

The generated catalog is trusted application content, not a user-input bypass.
Build-time compilation and reproducibility checks establish its relationship to
ordinary admission. Runtime still verifies the installed source bytes against
the generated witnesses before using compiled evidence.

### 6. Execute through runtime adapters

The LLM layer produces typed views, verified material, capability targets, and
runtime-neutral plans. Application runtime integrations own execution:

```text
family view / plan / target
  → application runtime adapter
  → inference request, Go Tool call, Skill runtime, MCP connection, or workflow runtime
```

Execution approval is separate from Artifact availability and enablement.

## Local state, secrets, and trusted capabilities

Three independent questions remain separate:

1. Is the source-backed Artifact available?
2. Is the Artifact enabled locally?
3. Is this use permitted and runtime-ready?

No answer implies another.

| Concern                                           | Owner                                                                                       |
| ------------------------------------------------- | ------------------------------------------------------------------------------------------- |
| Source-backed availability and current Definition | Generic Artifact and Refresh.                                                               |
| Local enablement                                  | Generic Artifact; family APIs expose family-oriented operations.                            |
| Family non-secret settings                        | Family settings/installation package, stored through generic Data or Overlay facilities.    |
| Secret binding metadata and cleanup               | Generic Secret lifecycle.                                                                   |
| Family secret-slot meaning                        | The owning family, such as Model Provider credentials or MCP logical secret targets.        |
| Plaintext secret resolution                       | Explicit trusted runtime capability supplied by application assembly.                       |
| Native filesystem path resolution                 | Explicit trusted Resource capability supplied only to trusted runtime/materialization code. |
| Runtime execution                                 | Runtime integration and application policy.                                                 |

Protected built-in content may be protected from ordinary source/package
mutation while still allowing explicitly designed local enablement and
feature-owned settings behavior. Protection is not execution approval and is
not a transport authorization mechanism.

Portable locator validation is not a filesystem sandbox. Source drivers define
their traversal, symlink, and snapshot behavior; application runtime isolation
remains an application concern.

## Application assembly and built-in content

`llmartifactory-go` does not import application topology, embedded content,
Wails, or runtime packages.

The FlexiGPT application supplies those concerns through separate packages:

| Package area                 | Application responsibility                                                                                           |
| ---------------------------- | -------------------------------------------------------------------------------------------------------------------- |
| `artifactbuiltin`            | Embedded family packages and generated package catalogs.                                                             |
| `artifactsetup/topology`     | Root and Source identities, protection, retention, and built-in topology.                                            |
| `artifactsetup/registration` | Schema codecs, canonical declaration decoders, source-format decoders, interpretations, and locator factories.       |
| `artifactsetup/llmsupport`   | Immutable family support values: document names/patterns, decoder IDs, package layouts, and managed Source profiles. |
| `artifactsetup/*runtime`     | Application adapters for inference, Tools, Skills, MCP, Workspace, credentials, and runtime identity translation.    |
| `cmd/agentgo`                | Wails transport, cross-Root management aggregation, startup sequencing, cancellation, and shutdown.                  |

The usual assembly sequence is:

1. Application setup selects codecs, decoders, interpretations, locator
   factories, support values, and generic Store policies.
2. Generic local deployment opens Artifactory Go.
3. `llmartifactory.Open` receives named generic capabilities:
   Artifact API, catalog API, ordinary Resource API, interpretations, locator
   factories, scope binding, and optional capability providers/projectors.
4. It constructs the shared locator and composition resolver.
5. Application assembly constructs family services with their narrow
   dependencies and supplied support values.
6. Application installation hydrates protected generated content before normal
   writers are exposed.
7. Runtime adapters receive only the family outputs and trusted capabilities
   required for execution.

The LLM attachment borrows the generic Store. It owns no provider resource and
has no independent provider shutdown. Consumers and runtime work must stop
before the owning generic Store deployment is closed.

## Extending the layer

### Add a declaration family

A new LLM family should normally add:

1. A versioned family contract under `<family>/contract/vN`.
2. A JSON Schema asset, semantic validation, typed reconstruction, and
   interpretation registration.
3. Family relationship extraction if the declaration contains relationships.
4. A family service only where there is real family behavior beyond contract
   interpretation.
5. A source-format adapter only when the family accepts non-canonical input.
6. Package/materialization/settings behavior only where the family owns it.
7. Application registration and support values outside `llmartifactory-go`.
8. A runtime adapter only when execution is required.

It must not require a generic Artifactory type switch, a global built-in
registry, a broad Store facade, or a Wails dependency.

### Add a source format or filename convention

1. Add a family-owned decoder or adapter.
2. Register it explicitly in application setup.
3. Add supported names and patterns to application support/topology values.
4. Add family package validation only if the format changes package semantics.

Do not spread filename checks through unrelated family services.

### Add a direct capability

A direct capability provider must:

1. Be explicitly supplied by application composition.
2. Return a valid direct `CapabilityTarget`.
3. Provide stable provider identity, provider-local identity, and evidence.
4. Be used only after normal source-backed Artifact lookup does not resolve a
   target.
5. Never fabricate generic Artifact identity or lifecycle state.

## Compatibility and change checklist

Changes to this layer must preserve the documented identity domains:

- Declaration type, schema ID, schema version, and canonical declaration body.
- Definition digest calculation.
- Family logical-name and logical-version rules.
- Contained subresource paths.
- Plugin member identity and ordering.
- Managed package address and package-relative file layout.
- Generated package-set and package fingerprint behavior.
- Locator semantics and configured scope behavior.
- Runtime identity formats owned by application adapters.

Before adding or moving behavior, ask:

- Is this generic source-backed storage/lifecycle behavior, LLM declaration
  behavior, application topology/content behavior, or runtime execution?
- Does this family own the state and decision, or is it only coordinating a
  generic flow?
- Is the operation a committed read, an explicit refresh, a verified read, or
  a mutation?
- Is a new package naming a real responsibility rather than preserving an old
  layer?
- Does the change accidentally make a normal read refresh, publish, clean up,
  decrypt, or acquire a native path?
- Are family settings and secret slots kept outside immutable Definitions?
- Does a generated-content change preserve deterministic compilation and
  source-witness verification?
- Can the extension be registered through explicit application support rather
  than modifying generic Store behavior?

The architecture is working when those answers are visible from the owning
family, generic entity, flow, or application assembly boundary—not from a
global registry or knowledge of the entire application.
