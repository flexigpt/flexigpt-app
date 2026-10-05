# Proposal B — LLM Artifactory Go

Artifact-family ownership, shared declaration core, supplied content, and runtime-independent planning.

- [Status and purpose](#status-and-purpose)
- [Goals](#goals)
  - [Artifact-family ownership](#artifact-family-ownership)
  - [A limited shared LLM core](#a-limited-shared-llm-core)
  - [Application-supplied content and capabilities](#application-supplied-content-and-capabilities)
  - [Minimal consumer surface](#minimal-consumer-surface)
  - [Preserved generic boundaries](#preserved-generic-boundaries)
- [Non-goals](#non-goals)
- [Destination layout](#destination-layout)
  - [LLM Artifactory root](#llm-artifactory-root)
  - [Shared LLM core](#shared-llm-core)
  - [Artifact-family folders](#artifact-family-folders)
- [Internal structure of an artifact-family folder](#internal-structure-of-an-artifact-family-folder)
- [Detailed artifact-family layout and responsibility assignments](#detailed-artifact-family-layout-and-responsibility-assignments)
  - [Text](#text)
    - [Responsibility moves](#responsibility-moves)
  - [Agent](#agent)
    - [Responsibility moves](#responsibility-moves-1)
  - [Skill](#skill)
    - [Responsibility moves](#responsibility-moves-2)
  - [Tool](#tool)
    - [Tool implementation scope](#tool-implementation-scope)
    - [Responsibility moves](#responsibility-moves-3)
  - [Model](#model)
    - [Responsibility moves](#responsibility-moves-4)
  - [Model Provider](#model-provider)
    - [Responsibility moves](#responsibility-moves-5)
  - [MCP](#mcp)
    - [Responsibility moves](#responsibility-moves-6)
  - [MCP Policy](#mcp-policy)
    - [Responsibility moves](#responsibility-moves-7)
  - [Plugin](#plugin)
    - [Plugin membership policy](#plugin-membership-policy)
    - [Family Plugin profiles](#family-plugin-profiles)
    - [Responsibility moves](#responsibility-moves-8)
  - [Team, Loop, and Workflow](#team-loop-and-workflow)
    - [Responsibility moves](#responsibility-moves-9)
  - [Workspace](#workspace)
    - [Responsibility moves](#responsibility-moves-10)
- [Shared declaration core](#shared-declaration-core)
  - [Declaration grammar ownership](#declaration-grammar-ownership)
  - [Family interpretation contract](#family-interpretation-contract)
  - [Canonical declaration decoding](#canonical-declaration-decoding)
  - [Typed Definition reconstruction](#typed-definition-reconstruction)
- [Composition and capability planning](#composition-and-capability-planning)
  - [Composition ownership](#composition-ownership)
  - [Direct membership and recursive plans](#direct-membership-and-recursive-plans)
  - [Lookup scope binding](#lookup-scope-binding)
  - [Capability targets](#capability-targets)
  - [Direct capability inputs](#direct-capability-inputs)
  - [Explicit composition refresh](#explicit-composition-refresh)
- [Plugin replacement of Collection](#plugin-replacement-of-collection)
  - [Member identity](#member-identity)
  - [Relationship ownership](#relationship-ownership)
- [Built-in content, generated catalogs, and preload](#built-in-content-generated-catalogs-and-preload)
  - [Application content layout](#application-content-layout)
  - [Generated content path](#generated-content-path)
  - [Preload and immutable content](#preload-and-immutable-content)
- [Generic Artifactory additions included in this proposal](#generic-artifactory-additions-included-in-this-proposal)
  - [Source-owned declaration discovery preparation](#source-owned-declaration-discovery-preparation)
  - [Complete Resource verification-session participation](#complete-resource-verification-session-participation)
  - [Explicit package-publication outcome](#explicit-package-publication-outcome)
  - [Atomic Artifact local-state cleanup](#atomic-artifact-local-state-cleanup)
  - [Secret recovery after ambiguous publication](#secret-recovery-after-ambiguous-publication)
  - [Local opener ownership transfer](#local-opener-ownership-transfer)
  - [Generic generated-content preload](#generic-generated-content-preload)
  - [Utility placement](#utility-placement)
- [Registration and assembly](#registration-and-assembly)
  - [No universal registration descriptor](#no-universal-registration-descriptor)
  - [LLM aggregate construction](#llm-aggregate-construction)
  - [Local deployment](#local-deployment)
- [Validation boundaries](#validation-boundaries)
- [Work packages](#work-packages)
  - [Package A: Establish the LLM library, content, and application-setup boundaries](#package-a-establish-the-llm-library-content-and-application-setup-boundaries)
    - [Purpose](#purpose)
    - [Scope](#scope)
    - [Destination folders and file groups](#destination-folders-and-file-groups)
    - [Responsibility movements completed in this package](#responsibility-movements-completed-in-this-package)
    - [Required input set](#required-input-set)
    - [Completion result](#completion-result)
  - [Package B: Complete generic Artifactory responsibilities required by LLM Artifactory](#package-b-complete-generic-artifactory-responsibilities-required-by-llm-artifactory)
    - [Purpose](#purpose-1)
    - [Scope](#scope-1)
    - [Destination folders and file groups](#destination-folders-and-file-groups-1)
    - [Required input set](#required-input-set-1)
    - [Completion result](#completion-result-1)
  - [Package C: Move declaration semantics into the core and artifact families](#package-c-move-declaration-semantics-into-the-core-and-artifact-families)
    - [Purpose](#purpose-2)
    - [Scope](#scope-2)
    - [Destination folders and file groups](#destination-folders-and-file-groups-2)
    - [Required input set](#required-input-set-2)
    - [Completion result](#completion-result-2)
  - [Package D: Establish composition and replace Collection with Plugin](#package-d-establish-composition-and-replace-collection-with-plugin)
    - [Purpose](#purpose-3)
    - [Scope](#scope-3)
    - [Destination folders and file groups](#destination-folders-and-file-groups-3)
    - [Required input set](#required-input-set-3)
    - [Completion result](#completion-result-3)
  - [Package E: Consolidate artifact-family services, local state, and runtime-neutral capability preparation](#package-e-consolidate-artifact-family-services-local-state-and-runtime-neutral-capability-preparation)
    - [Purpose](#purpose-4)
    - [Scope](#scope-4)
    - [Required input sets](#required-input-sets)
    - [Completion result](#completion-result-4)
  - [Package F: Bind supplied content, direct capabilities, application setup, runtime adapters, and wrappers](#package-f-bind-supplied-content-direct-capabilities-application-setup-runtime-adapters-and-wrappers)
    - [Purpose](#purpose-5)
    - [Scope](#scope-5)
    - [Destination folders and file groups](#destination-folders-and-file-groups-4)
    - [Required input set](#required-input-set-4)
    - [Completion result](#completion-result-5)
- [Alignment with the prior Artifactory proposal and retained principles](#alignment-with-the-prior-artifactory-proposal-and-retained-principles)
  - [Retained Artifactory principles](#retained-artifactory-principles)
  - [Earlier work-package status](#earlier-work-package-status)
  - [Architectural violations removed by this proposal](#architectural-violations-removed-by-this-proposal)
  - [Intentional architectural exceptions](#intentional-architectural-exceptions)
  - [Generic Artifactory placement after this proposal](#generic-artifactory-placement-after-this-proposal)

## Status and purpose

This proposal replaces the previous LLM Artifactory proposal.

Proposal A established the generic Artifactory core:

```text
Root
  └── Source
        └── observed declaration
              └── admitted Definition
                    └── source-backed Artifact
```

It also established the associated local state model:

```text
Artifact
  ├── local metadata and enablement
  ├── non-secret local state
  └── secret bindings
```

The generic implementation is now substantially complete. The remaining work is not another generic storage redesign. It is the extraction and cleanup of LLM-specific declaration semantics, artifact-family behavior, composition behavior, built-in content handling, and runtime-independent capability preparation.

I propose creating `internal/llmartifactory-go` as the LLM artifact domain layer above `internal/artifactory-go`.

The resulting boundary is:

| Layer                                   | Responsibility                                                                                                                                                                                                                             |
| --------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `internal/artifactory-go`               | Generic source-backed artifact storage, lifecycle, package publication, verification, local state, secret lifecycle, and protected installation.                                                                                           |
| `internal/llmartifactory-go`            | LLM declaration language, artifact-family semantics, typed Definition reconstruction, Plugin membership, composition, capability planning, family package preparation, family local-state meaning, and runtime-neutral target preparation. |
| `internal/artifactbuiltin`              | Application-shipped embedded content and generated content catalogs.                                                                                                                                                                       |
| `internal/artifactsetup`                | Application topology, Root and Source identities, registration selection, installation planning, baseline provisioning, and startup sequencing.                                                                                            |
| Existing runtime and aggregate packages | Execution, inference adaptation, Go Tool invocation, MCP connections, plaintext secret use, Skill sessions, Workflow execution, conversation behavior, and transport concerns.                                                             |

This is an architectural split with functional preservation. The only deliberate functional narrowing is Tool support: Tool declarations and preparation retain Go implementations only. Existing SDK Tool branches are removed rather than retained as inactive extensibility.

## Goals

### Artifact-family ownership

Each LLM artifact type owns its own:

- Declaration schema and semantic validation.
- Source-format adaptation.
- Typed Definition reconstruction.
- Artifact-specific identity qualifiers.
- Relationship interpretation.
- Package layout and package preparation.
- Local-state schema and legal secret slots where applicable.
- Runtime-neutral capability preparation.
- Built-in content preparation rules where applicable.

### A limited shared LLM core

The shared core owns only behavior that is genuinely cross-family:

- Declaration grammar.
- Canonical declaration entry representation.
- Header dispatch mechanics.
- Generic member representation.
- Generic declaration locator grammar.
- Family interpretation registration.
- Generic relationship traversal mechanics.
- Read-only composition resolution.
- Explicit composition refresh closure.
- Generic capability-target representation.

It does not own concrete Agent, Skill, Model, MCP, Plugin, Tool, Workspace, Workflow, or Team semantics.

### Application-supplied content and capabilities

The reusable LLM library does not know:

- Which Agents, Skills, MCP servers, Tools, Models, or Workspace policies ship with FlexiGPT.
- Which Root is the protected built-in Root.
- Which Root is the user Root.
- Which embedded files are compiled into the binary.
- Which Go Tool registry is used.
- Which Model adapters are installed.
- Which inference runtime is present.
- Which secret backend or keyring identity is used.

Those facts enter through explicit application assembly.

### Minimal consumer surface

Ordinary consumers receive family-oriented APIs:

- Agent API.
- Skill API.
- Tool API.
- Model API.
- Model Provider API.
- MCP API.
- MCP Policy API.
- Plugin API.
- Text API.
- Workspace API.
- Composition API where direct cross-family planning is required.

Consumers do not construct separate locator registries, graph resolvers, Collection services, compiled-catalog readers, or generic local-state adapters.

### Preserved generic boundaries

The proposal preserves Proposal A’s central rules:

- Entities own state and lifecycle policy.
- Flows own cross-entity sequencing.
- Providers execute contracts rather than choose policy.
- Catalog reads remain committed reads.
- Source freshness requires verified access.
- Package publication remains source-first.
- Secret lifecycle remains generic and durable.
- Protected installation remains generic and explicit.
- Runtime execution remains outside Artifactory and LLM Artifactory.

## Non-goals

This proposal does not introduce:

- A second generic Store.
- A universal LLM provider descriptor combining schemas, decoders, locators, content, drivers, and runtime services.
- A new persistence provider.
- A new Source driver.
- A dependency solver.
- A general plugin execution engine.
- A workflow runtime.
- A generic authorization server.
- A generic frontend transport layer.
- Backward-compatible package aliases.
- Wrapper packages that only redirect to moved APIs.
- A global built-in registry inside the reusable LLM library.
- Automatic URL, Git, package, archive, or command Locator fetching.

Tests and documentation are intentionally deferred until the structural and ownership boundaries have settled.

## Destination layout

### LLM Artifactory root

The root package has one narrow responsibility: assembling LLM artifact services over an already assembled generic Artifactory Store.

| Path                                         | Responsibility                                                                                                                                  |
| -------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------- |
| `internal/llmartifactory-go/api.go`          | Aggregate public surface and named family capabilities.                                                                                         |
| `internal/llmartifactory-go/config.go`       | Explicit aggregate configuration.                                                                                                               |
| `internal/llmartifactory-go/open.go`         | Construction from the generic Artifactory aggregate and selected registrations.                                                                 |
| `internal/llmartifactory-go/registration.go` | Registration validation and assembly of separately supplied schemas, decoders, family interpreters, locator handlers, and capability providers. |
| `internal/llmartifactory-go/close.go`        | Release of LLM-owned in-memory resources only. It does not close the generic Store.                                                             |
| `internal/llmartifactory-go/artifactory.go`  | The assembled LLM Artifactory handle.                                                                                                           |

The root package contains no declaration parsing, family rules, runtime behavior, content inventory, or application topology.

The LLM aggregate borrows the generic `store/compose.Store` capability surface. It does not reproduce Root, Source, Artifact, Definition, Resource, Secret, Overlay, or Install fields through another mirrored aggregate.

The current package remains inside `internal/` while the application and runtime are still in one repository. Its exported surface is designed as the future public surface for extraction into a standalone repository. No temporary public forwarding package is added.

### Shared LLM core

The following folder is organizational only:

| Path                               | Responsibility                                                    |
| ---------------------------------- | ----------------------------------------------------------------- |
| `internal/llmartifactory-go/core/` | Directory only. It contains no Go package and no direct Go files. |

Its child packages have specific responsibilities.

| Path                               | Responsibility                                                                                                                                                                                                                           |
| ---------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `core/declaration/`                | Shared declaration grammar: type token shape, header syntax, canonical entry representation, generic member form, generic relationship envelope, generic locator grammar, canonical member identity, and declaration dispatch mechanics. |
| `core/declaration/decoder/`        | Canonical JSON and YAML declaration decoding mechanics. It dispatches to a complete expected schema key and delegates family semantics to registered family interpreters.                                                                |
| `core/declaration/interpretation/` | Family interpretation contracts and the immutable registry used by declaration decoding and composition. It contains no runtime services, Source drivers, content inventory, or persistence behavior.                                    |
| `core/composition/`                | Read-only relationship resolution, capability-plan values, scope binding, target values, cycle detection, graph limits, ambiguity handling, and direct-versus-recursive planning.                                                        |
| `core/composition/locator/`        | Locator-resolution contracts and registered locator handlers. It resolves committed state only.                                                                                                                                          |
| `core/composition/refresh/`        | Explicit reachable-declaration discovery preparation and refresh closure.                                                                                                                                                                |
| `core/composition/internal/`       | Private traversal, lookup, plan assembly, and resolver state.                                                                                                                                                                            |

`core` is not a substitute for `utils`, `kit`, `manager`, or `base`. It has no catch-all package. A new concern enters this folder only when it is shared across artifact families and cannot be assigned to an existing named owner.

Concrete declaration types do not live under `core/declaration`. For example:

- Agent declarations live in `agent/contract/v1`.
- MCP declarations live in `mcp/contract/v1`.
- Plugin declarations live in `plugin/contract/v1`.
- Tool declarations live in `tool/contract/v1`.

The core defines the grammar through which a declaration type participates. It does not define what a declaration type means.

### Artifact-family folders

The top-level folders under `internal/llmartifactory-go` are artifact-family owners.

| Top-level folder | Artifact responsibility                                                                                                                       |
| ---------------- | --------------------------------------------------------------------------------------------------------------------------------------------- |
| `agent/`         | Agent declaration, import/export, capability projection, Agent Markdown adaptation, and managed Agent packages.                               |
| `skill/`         | Skill declaration, `SKILL.md` adaptation, package layout, verified Skill materialization, and Skill capability preparation.                   |
| `tool/`          | Go Tool declaration, registered Go Tool metadata, Tool capability preparation, and Tool package conventions.                                  |
| `model/`         | Model declaration, Model local settings, Provider linkage, Model target preparation, and Model package conventions.                           |
| `modelprovider/` | Model Provider declaration, Provider local settings, credential binding semantics, default Model selection, and Provider package conventions. |
| `mcp/`           | MCP declaration, MCP source format, installation data, secret target semantics, and MCP package conventions.                                  |
| `mcppolicy/`     | MCP Policy declaration, policy normalization, policy composition, and policy package conventions.                                             |
| `plugin/`        | Plugin declaration, Plugin membership policy, direct membership resolution, Plugin editing, and Plugin package conventions.                   |
| `text/`          | Text declaration, Text Markdown adaptation, verified Text materialization, and Text content conventions.                                      |
| `team/`          | Team declaration and Team relationship semantics.                                                                                             |
| `loop/`          | Loop declaration and Loop planning semantics.                                                                                                 |
| `workflow/`      | Workflow declaration, node and edge semantics, and Workflow planning semantics.                                                               |
| `workspace/`     | Workspace declaration, effective Workspace selection, directory-associated artifact behavior, and Workspace capability selection.             |

No `collection/`, `consumerapi/`, `providerapi/`, `builtin/`, or generic `domain/` umbrella exists at the LLM Artifactory root.

## Internal structure of an artifact-family folder

Each artifact family uses the same ownership pattern where the responsibility exists. The pattern is not mechanically mandatory; empty folders are not created merely for uniformity.

| Family subfolder or file group | Responsibility                                                                                                                                                                           |
| ------------------------------ | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `<family>/api.go`              | Family public API contract.                                                                                                                                                              |
| `<family>/service.go`          | Family service behavior and construction.                                                                                                                                                |
| `<family>/domain/`             | Family-owned domain values, request and result values, views, plan values, and family policy values.                                                                                     |
| `<family>/contract/v1/`        | Portable declaration document, JSON Schema asset, schema codec, semantic validation, Definition reconstruction, relationship extraction, and declaration identity rules for the version. |
| `<family>/sourceformat/`       | Source-format adapters that transform non-canonical source material into the family’s canonical declaration form.                                                                        |
| `<family>/package/`            | Package address conventions, package-relative files, normalization, expected artifacts, and preparation.                                                                                 |
| `<family>/settings/`           | Non-secret local-state semantics and local configuration behavior.                                                                                                                       |
| `<family>/secret/`             | Family-specific secret namespace, legal slot, logical-secret selector, and binding behavior. It contains no plaintext resolution.                                                        |
| `<family>/installation/`       | Family-specific installation state and lifecycle decisions.                                                                                                                              |
| `<family>/materialize/`        | Verified source-material reconstruction for a trusted runtime consumer.                                                                                                                  |
| `<family>/capability/`         | Runtime-neutral direct-capability metadata and target preparation.                                                                                                                       |
| `<family>/local/`              | Explicit local deployment integration using concrete filesystem or embedded Source adapters.                                                                                             |
| `<family>/internal/`           | Private implementation only when a real family-local implementation boundary exists.                                                                                                     |

The artifact-family root remains the owner of the family service. The design does not create a forwarding `<family>/compose`, `<family>/impl`, or `<family>/manager` package.

## Detailed artifact-family layout and responsibility assignments

### Text

| Path                          | Responsibility                                                                                                                |
| ----------------------------- | ----------------------------------------------------------------------------------------------------------------------------- |
| `text/contract/v1/`           | Text declaration, insertion target, schema, codec, Definition reconstruction, inline-content rules, and source-locator rules. |
| `text/sourceformat/markdown/` | Generic Markdown-to-Text declaration adaptation.                                                                              |
| `text/materialize/`           | Verified Text materialization from inline content or verified Source entries and trees.                                       |
| `text/domain/`                | Materialized Text values and Text-facing result projections.                                                                  |
| `text/`                       | Text read and materialization API.                                                                                            |

#### Responsibility moves

| Current responsibility                              | Destination                                                 |
| --------------------------------------------------- | ----------------------------------------------------------- |
| `artifactcontract/declaration/textv1`               | `llmartifactory-go/text/contract/v1`                        |
| `artifactcontract/providermarkdown/text_decoder.go` | `llmartifactory-go/text/sourceformat/markdown`              |
| `artifactcontract/materializetext`                  | `llmartifactory-go/text/materialize`                        |
| Agent-owned Text materialization entrypoint         | Removed. Agent and Workspace consume the Text API directly. |

Text owns source-backed content selection. Workspace runtime continues to own prompt budgeting, truncation, insertion-channel rendering, and conversation behavior.

### Agent

| Path                           | Responsibility                                                                                                                                            |
| ------------------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `agent/contract/v1/`           | Agent declaration, schema, codec, relationship legality, contained identity rules, and managed-import profile restrictions.                               |
| `agent/sourceformat/markdown/` | `AGENT.md` and `*.agent.md` adaptation.                                                                                                                   |
| `agent/domain/`                | Agent views, capability projections, import-preview values, import conflicts, import confirmations, and export values.                                    |
| `agent/import/`                | Managed Agent normalization, preview, prepared import envelopes, commit validation, package expectation preparation, and membership restoration analysis. |
| `agent/package/`               | Managed Agent package layout and publication expectations.                                                                                                |
| `agent/`                       | Agent reads, enablement, import/export, and managed Agent lifecycle.                                                                                      |

#### Responsibility moves

| Current responsibility                                        | Destination                                                                                    |
| ------------------------------------------------------------- | ---------------------------------------------------------------------------------------------- |
| `artifactcontract/declaration/agentv1`                        | `agent/contract/v1`                                                                            |
| `artifactcontract/providermarkdown/agent_markdown_decoder.go` | `agent/sourceformat/markdown`                                                                  |
| `agent/store/domain/managed_import.go`                        | `agent/import`                                                                                 |
| `agent/store/domain/package_layout.go`                        | `agent/package`                                                                                |
| `agent/store/consumerapi/import*.go`                          | `agent/import` and `agent/domain`                                                              |
| `agent/store/consumerapi/export.go`                           | `agent`                                                                                        |
| `artifactcontract/signer`                                     | `agent/import`, because prepared Agent import envelopes are currently Agent-specific behavior. |

The generic authenticated-envelope mechanism is not extracted into Artifactory during this proposal because Agent import is its only current owner. A later second independent consumer would justify a responsibility-specific package under `artifactory-go/contractutil/envelopeutil`.

Agent import remains an explicit multi-step workflow:

1. Parse and canonicalize input.
2. Apply managed Agent profile restrictions.
3. Normalize managed references.
4. Validate Agent-specific managed content policy.
5. Prepare expected Definition and package identity.
6. Inspect existing Artifact, package, membership, and dependency state.
7. Produce an expiring prepared-import envelope.
8. Revalidate the prepared content and current witnesses during commit.
9. Publish Plugin membership and Agent package as separate observable commits.

The existing behavior that a membership write can succeed before package publication remains visible rather than being represented as one false transaction.

### Skill

| Path                          | Responsibility                                                                                      |
| ----------------------------- | --------------------------------------------------------------------------------------------------- |
| `skill/contract/v1/`          | Portable Skill declaration, schema, codec, Definition reconstruction, and Skill relationship rules. |
| `skill/sourceformat/skillmd/` | `SKILL.md` parsing, warnings, and projection to a portable Skill declaration.                       |
| `skill/domain/`               | Skill views, parsed Skill document projections, warnings, and package identity values.              |
| `skill/package/`              | Managed Skill layout, Skill directory naming rule, file normalization, and package preparation.     |
| `skill/materialize/`          | Verified Skill package materialization and trusted native-path acquisition.                         |
| `skill/local/`                | Native filesystem Skill registration and local Source preparation.                                  |
| `skill/`                      | Skill artifact reads, enablement, managed lifecycle, and capability preparation.                    |

#### Responsibility moves

| Current responsibility                               | Destination                                     |
| ---------------------------------------------------- | ----------------------------------------------- |
| `artifactcontract/declaration/skillv1`               | `skill/contract/v1`                             |
| `skill/store/domain/enc_dec.go`                      | `skill/sourceformat/skillmd` and `skill/domain` |
| `skill/store/domain/managed_files.go`                | `skill/package`                                 |
| `skill/store/domain/package_layout.go`               | `skill/package`                                 |
| `skill/store/domain/runtime_package.go`              | `skill/materialize`                             |
| `skill/store/materialize`                            | `skill/materialize`                             |
| `skill/store/providerapi`                            | `skill/sourceformat/skillmd`                    |
| Skill artifact portions of `skill/store/consumerapi` | `skill` and `skill/local`                       |

The outer Skill runtime retains:

- Agent Skills runtime catalog synchronization.
- Session creation and closure.
- Prompt rendering.
- Script execution.
- Runtime-owned Skill identities.
- Runtime registration and invalidation.

The LLM artifact layer exposes verified Skill material and a runtime-neutral capability plan. It does not become the Agent Skills runtime.

### Tool

| Path                | Responsibility                                                                                                                           |
| ------------------- | ---------------------------------------------------------------------------------------------------------------------------------------- |
| `tool/contract/v1/` | Go Tool declaration, schema, codec, Tool Definition reconstruction, input and output schema semantics, and Go implementation validation. |
| `tool/domain/`      | Tool descriptor, Tool view, Tool package identity, and Tool collection membership policy values.                                         |
| `tool/package/`     | Tool package and Tool Plugin package preparation.                                                                                        |
| `tool/capability/`  | Registered Go Tool metadata lookup, artifact-to-capability target preparation, target evidence, and Tool availability checks.            |
| `tool/`             | Tool reads, enablement, Plugin gate checks, and capability planning.                                                                     |

#### Tool implementation scope

The resulting Tool contract supports:

- `implementation.kind = go`.
- Registered Go function identity.
- Registered Go Tool metadata.
- Input and output JSON Schema.
- Version and Tool capability metadata.
- Go Tool target preparation.

The resulting Tool contract does not support:

- SDK Tool implementation declarations.
- SDK Tool type declarations.
- SDK Tool package preparation.
- SDK Tool target mapping.
- SDK Tool inference hydration.
- SDK Tool runtime execution.

#### Responsibility moves

| Current responsibility                                  | Destination                                                                                                       |
| ------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------- |
| `artifactcontract/declaration/toolv1`                   | `tool/contract/v1`                                                                                                |
| `tool/store/domain`                                     | `tool/domain` and `tool/package`                                                                                  |
| Tool artifact behavior in `tool/store/consumerapi`      | `tool` and `tool/capability`                                                                                      |
| Artifact-to-mapped-target portion of `tool/aggregate`   | `tool/capability`                                                                                                 |
| Go Tool metadata extraction from `tool/llmtoolsadapter` | Remains in an application/runtime adapter and is exposed to `tool/capability` through a metadata-only capability. |
| Go Tool invocation from `tool/llmtoolsadapter`          | Remains outside LLM Artifactory in the Tool runtime integration.                                                  |

The metadata capability and invocation capability are separate concrete objects. The Tool family receives only the metadata capability. Tool invocation remains available only to the Tool runtime.

The generated Tool inventory contains only Go Tools after this split. Existing SDK Tool declarations and generated SDK Tool entries are intentionally removed.

### Model

| Path                 | Responsibility                                                                                                                                      |
| -------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------- |
| `model/contract/v1/` | Model declaration, schema, codec, Definition reconstruction, and Provider reference semantics.                                                      |
| `model/parameters/`  | Portable Model defaults, capability-patch, output, reasoning, cache, and adapter-parameter vocabulary shared by Model and Model Provider contracts. |
| `model/domain/`      | Model projections, Model settings values, Provider-reference values, and Model configuration evidence.                                              |
| `model/settings/`    | Mutable and protected Model local-state semantics.                                                                                                  |
| `model/package/`     | Model package identity and package preparation.                                                                                                     |
| `model/capability/`  | Runtime-neutral Model target input and configuration evidence.                                                                                      |
| `model/`             | Model reads, managed Model lifecycle, settings, and target preparation.                                                                             |

#### Responsibility moves

| Current responsibility                                   | Destination                                                   |
| -------------------------------------------------------- | ------------------------------------------------------------- |
| `artifactcontract/declaration/modelv1`                   | `model/contract/v1`                                           |
| Model portions of `model/store/domain`                   | `model/domain`, `model/parameters`, and `model/package`       |
| Model overlay behavior from `model/store/overlay`        | `model/settings`                                              |
| Model artifact behavior from `model/store/consumerapi`   | `model`                                                       |
| Artifact target payload from `model/aggregate/target.go` | `model/capability`                                            |
| Inference SDK conversion and request execution           | Remains in the Model runtime aggregate and inference adapter. |

The Model family retains source-backed Model identity, Provider reference interpretation, Model local settings, and runtime-neutral configuration evidence.

The Model runtime retains:

- Inference adapter selection.
- Capability derivation.
- Request-level merge behavior.
- Inference client construction.
- Plaintext credential use.
- Provider publication into the long-lived inference runtime.
- Runtime fallback selection behavior.

### Model Provider

| Path                         | Responsibility                                                                                                                  |
| ---------------------------- | ------------------------------------------------------------------------------------------------------------------------------- |
| `modelprovider/contract/v1/` | Model Provider declaration, schema, codec, connection and authentication declaration validation, and Definition reconstruction. |
| `modelprovider/domain/`      | Provider projection, Provider settings values, adapter identity, and default Model reference values.                            |
| `modelprovider/settings/`    | Mutable and protected Provider overlay semantics.                                                                               |
| `modelprovider/credential/`  | Provider credential slot semantics and binding-management operations. It does not expose plaintext reads.                       |
| `modelprovider/preference/`  | Default-Provider preference persistence and selection rules.                                                                    |
| `modelprovider/package/`     | Provider package identity and package preparation.                                                                              |
| `modelprovider/`             | Provider reads, managed lifecycle, settings, credential binding, and default Model selection.                                   |

#### Responsibility moves

| Current responsibility                                    | Destination                                                              |
| --------------------------------------------------------- | ------------------------------------------------------------------------ |
| `artifactcontract/declaration/modelproviderv1`            | `modelprovider/contract/v1`                                              |
| Provider portions of `model/store/domain`                 | `modelprovider/domain`                                                   |
| Provider overlay behavior from `model/store/overlay`      | `modelprovider/settings`                                                 |
| Provider credential behavior from `model/store/overlay`   | `modelprovider/credential`                                               |
| `cmd/agentgo/wrapper_model_preferences.go`                | `modelprovider/preference`                                               |
| Provider artifact behavior from `model/store/consumerapi` | `modelprovider`                                                          |
| Provider target preparation from `model/aggregate`        | `modelprovider` and `model/capability` where Model evidence is required. |

The current stable managed Model Source ID assumption requires correction. SQLite currently treats Source IDs as globally unique, while the existing comment assumes the same Source ID can exist in every Root.

The new Model managed-source path uses:

- The stable Root-local managed Source storage key.
- `Source.Ensure`.
- A newly allocated candidate Source ID for creation intent.
- The Source ID returned by the existing or newly created Source.

Existing persisted Sources retain their existing IDs. No database identity migration is introduced solely to preserve an incorrect Root-local ID assumption.

### MCP

| Path                       | Responsibility                                                                                                                      |
| -------------------------- | ----------------------------------------------------------------------------------------------------------------------------------- |
| `mcp/contract/v1/`         | MCP declaration, schema, codec, transport declaration semantics, input declaration semantics, and Definition reconstruction.        |
| `mcp/sourceformat/config/` | Standard `mcpServers` source configuration adaptation.                                                                              |
| `mcp/domain/`              | Runtime-independent MCP connection declaration projection and configuration values.                                                 |
| `mcp/installation/`        | Non-secret installation data, connection-profile selection, protected overlay handling, and installation validation.                |
| `mcp/secret/`              | Logical MCP secret selector syntax, legal target validation, Artifact Store binding-slot mapping, and binding-management semantics. |
| `mcp/package/`             | Managed MCP package identity and preparation.                                                                                       |
| `mcp/`                     | MCP reads, managed MCP lifecycle, installation planning, and runtime-neutral server preparation.                                    |

#### Responsibility moves

| Current responsibility                                                                                   | Destination                                   |
| -------------------------------------------------------------------------------------------------------- | --------------------------------------------- |
| `artifactcontract/declaration/mcpv1`                                                                     | `mcp/contract/v1`                             |
| `mcp/store/providerapi`                                                                                  | `mcp/sourceformat/config`                     |
| Runtime-independent portions of `mcp/store/domain/server`                                                | `mcp/domain` and `mcp/installation`           |
| `mcp/store/domain/secret`                                                                                | `mcp/secret`                                  |
| `mcp/store/overlay`                                                                                      | `mcp/installation`                            |
| Artifact-facing MCP behavior from `mcp/store/consumerapi`                                                | `mcp`                                         |
| MCP logical-secret translation from `cmd/agentgo/wrapper_mcp_artifact_secrets.go`                        | `mcp/secret`                                  |
| Plaintext secret use, OAuth token use, connection establishment, runtime state, approval, and invocation | Remain in MCP runtime and aggregate packages. |

`mcp/secret` receives generic secret binding management capability. It does not receive `secret.RuntimeAPI`. The runtime adapter receives plaintext access through a separate explicitly supplied trusted capability.

The current `ServerData` local-state model remains structurally compatible:

- Mutable MCP artifacts retain namespaced Artifact Data.
- Protected MCP artifacts retain protected overlay records.
- Secret values remain generic Artifact Store secret bindings.
- MCP-specific logical secret references remain family-owned values.

### MCP Policy

| Path                     | Responsibility                                                            |
| ------------------------ | ------------------------------------------------------------------------- |
| `mcppolicy/contract/v1/` | MCP Policy declaration, schema, codec, and Definition reconstruction.     |
| `mcppolicy/domain/`      | Runtime-independent policy values, normalization, and policy composition. |
| `mcppolicy/package/`     | Managed MCP Policy package identity and preparation.                      |
| `mcppolicy/`             | Policy reads, authoring, managed lifecycle, and policy-plan preparation.  |

#### Responsibility moves

| Current responsibility                         | Destination                                          |
| ---------------------------------------------- | ---------------------------------------------------- |
| `artifactcontract/declaration/mcppolicyv1`     | `mcppolicy/contract/v1`                              |
| `mcp/store/domain/policy`                      | `mcppolicy/domain`                                   |
| MCP Policy behavior in `mcp/store/consumerapi` | `mcppolicy`                                          |
| Runtime approval evaluation                    | Remains in MCP runtime policy and connection layers. |

MCP Policy composition remains runtime-independent. Runtime policy enforcement remains outside LLM Artifactory.

### Plugin

Plugin replaces the current Collection entity and API.

A Plugin is an Artifact of declaration type `plugin`. Collection is not retained as a second artifact-family abstraction.

| Path                  | Responsibility                                                                                                           |
| --------------------- | ------------------------------------------------------------------------------------------------------------------------ |
| `plugin/contract/v1/` | Plugin declaration, schema, codec, Definition reconstruction, and Plugin declaration metadata semantics.                 |
| `plugin/domain/`      | Plugin views, member values, membership policy, authoring profile, baseline policy, and direct membership result values. |
| `plugin/membership/`  | Direct membership inspection, reverse membership inspection, membership mutation, and ordered membership identity.       |
| `plugin/package/`     | Managed Plugin package identity, document preparation, and expected artifact preparation.                                |
| `plugin/`             | Plugin reads, list operations, authoring, deletion eligibility, enablement delegation, and package publication.          |

#### Plugin membership policy

A Plugin membership policy is explicit and independent from the currently observed member list.

| Membership mode      | Meaning                                                                                                                                                 |
| -------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `single-type`        | Every direct Plugin member declares one specified artifact type.                                                                                        |
| `mixed-type`         | Every direct Plugin member belongs to an explicit allowed set containing two or more artifact types.                                                    |
| legacy unconstrained | Existing Plugins without the explicit membership policy metadata remain readable and preserve their existing behavior. They are not silently rewritten. |

The policy is stored in one Plugin-owned metadata entry in the existing declaration metadata object.

The policy value contains:

- Membership mode.
- Allowed direct artifact types.
- Allowed direct member forms.
- Relationship-behavior restrictions when required by the owning feature profile.

The policy distinguishes an empty Agent Plugin from an unconstrained empty Plugin. An empty single-type Plugin can remain Agent-only even before its first member is added.

#### Family Plugin profiles

The owner of a feature-specific Plugin supplies its profile:

| Feature                 | Plugin membership profile                                                     |
| ----------------------- | ----------------------------------------------------------------------------- |
| Agent grouping          | Single-type `agent`, named direct members.                                    |
| Skill grouping          | Single-type `skill`, named direct members.                                    |
| Tool grouping           | Single-type `tool`, named direct members, read-only content policy.           |
| MCP grouping            | Mixed-type `mcp` and `mcp.policy`, with the currently supported member forms. |
| Generic authored Plugin | Explicit single-type or mixed-type policy provided by the caller.             |

Baseline identity and application provisioning are not generic Plugin state. Application setup supplies baseline declarations. Plugin owns the rule that a configured baseline cannot be deleted.

#### Responsibility moves

| Current responsibility                  | Destination                                                                                               |
| --------------------------------------- | --------------------------------------------------------------------------------------------------------- |
| `internal/collection`                   | `llmartifactory-go/plugin`                                                                                |
| `artifactcontract/declaration/pluginv1` | `plugin/contract/v1`                                                                                      |
| `collection.DomainPolicy`               | Split between `plugin/domain` membership policy and the owning Agent, Skill, MCP, or Tool family profile. |
| Agent Collection API                    | Removed; Agent uses Plugin API with Agent membership profile.                                             |
| Skill Collection API                    | Removed; Skill uses Plugin API with Skill membership profile.                                             |
| MCP Collection API                      | Removed; MCP uses Plugin API with mixed MCP/MCP Policy profile.                                           |
| Tool Collection API                     | Removed; Tool uses Plugin API with Tool read-only profile.                                                |
| `collection.DocumentCache`              | `plugin` as a Plugin Definition projection cache.                                                         |

Plugin membership remains declaration content. It does not create generic Artifact ownership, foreign keys, lifecycle cascades, or target deletion rights.

The current distinction remains explicit:

- Removing a Plugin member does not delete the referenced Artifact.
- Removing an Artifact can leave an unavailable Plugin member.
- Removing a managed Plugin package is distinct from removing its direct members.
- Contained declarations remain source-emitted Artifacts and are not editable managed Plugin members.

### Team, Loop, and Workflow

| Family      | Internal layout                       | Responsibility                                                                                                                         |
| ----------- | ------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------- |
| `team/`     | `contract/v1`, `domain`, root service | Team declaration, Team membership legality, direct Loop and Workflow relationship positions, and Team plan values.                     |
| `loop/`     | `contract/v1`, `domain`, root service | Loop declaration, body relationship semantics, output-match validation, and Loop plan values.                                          |
| `workflow/` | `contract/v1`, `domain`, root service | Workflow declaration, stable node identity, node-member legality, starts, edges, joins, output-match values, and Workflow plan values. |

#### Responsibility moves

| Current responsibility                                           | Destination                                               |
| ---------------------------------------------------------------- | --------------------------------------------------------- |
| `artifactcontract/declaration/teamv1`                            | `team/contract/v1`                                        |
| `artifactcontract/declaration/loopv1`                            | `loop/contract/v1`                                        |
| `artifactcontract/declaration/workflowv1`                        | `workflow/contract/v1`                                    |
| Family branches in `artifactcontract/decoder/tree_validation.go` | Respective family contract packages.                      |
| Family branches in `artifactcontract/declaration/walk.go`        | Respective family relationship extractors.                |
| Family branches in `artifactcontract/resolve/structure.go`       | Respective family composition facts and plan projections. |

Loop execution and Workflow scheduling remain outside LLM Artifactory.

### Workspace

| Path                     | Responsibility                                                                                                             |
| ------------------------ | -------------------------------------------------------------------------------------------------------------------------- |
| `workspace/contract/v1/` | Workspace declaration, schema, codec, Definition reconstruction, and Workspace membership semantics.                       |
| `workspace/domain/`      | Workspace view, effective Workspace values, directory-associated Source identity, and composition-source context.          |
| `workspace/selection/`   | Effective Workspace selection, manifest/default-policy precedence, capability selection, and Workspace artifact selection. |
| `workspace/local/`       | Explicit native-directory and embedded-policy Source integration. It receives policy content as an input.                  |
| `workspace/`             | Workspace artifact reads, directory registration behavior, effective Workspace resolution, and capability-plan generation. |

#### Responsibility moves

| Current responsibility                                              | Destination                                                                    |
| ------------------------------------------------------------------- | ------------------------------------------------------------------------------ |
| `artifactcontract/declaration/workspacev1`                          | `workspace/contract/v1`                                                        |
| Artifact-facing Workspace behavior in `workspace/store/consumerapi` | `workspace`, `workspace/domain`, `workspace/selection`, and `workspace/local`. |
| `workspace/defaultpolicy` content loading                           | `artifactbuiltin/workspace` and `artifactsetup/workspace`.                     |
| Workspace prompt adapter                                            | Remains in outer Workspace runtime integration.                                |
| Workspace Skill adapter                                             | Remains in outer Workspace runtime integration.                                |
| Workspace MCP adapter                                               | Remains in outer Workspace runtime integration.                                |
| Conversation selection and usage                                    | Remain in conversation/runtime packages.                                       |

The existing Workspace behavior remains preserved:

- A physical Workspace manifest suppresses the fallback default Workspace, including when that manifest is invalid.
- The default Workspace policy remains Root-local and is not automatically promoted into the protected built-in Root.
- The policy Source and directory Source remain distinct.
- Workspace composition may use the directory Source as its composition Source.
- Explicit empty artifact selection remains distinct from omitted selection.
- Workspace removal retires Artifact Store state without deleting the external filesystem directory.

`workspace/local` is an explicit local deployment adapter. It may use filesystem and embedded Source drivers. The core Workspace package does not select provider kinds or import application embedded content.

## Shared declaration core

### Declaration grammar ownership

`core/declaration` owns only the shared declaration grammar:

- `type` token form.
- Optional `apiVersion`.
- Header shape.
- Canonical Entry representation.
- Named member, contained member, and selector member form.
- Generic relationship envelope.
- Generic declaration Locator syntax.
- Portable reference and path rules.
- Canonical member identity and ordering.
- Dispatch rules from header to complete expected schema key.

It does not contain:

- An enumeration of Agent, MCP, Tool, Model, Skill, Plugin, Team, Loop, Workflow, or Workspace.
- A central schema registry listing all family schemas.
- Family-specific fields.
- Family-specific relationship validation.
- Family package filenames.
- Application Root IDs.
- Built-in content references.

### Family interpretation contract

`core/declaration/interpretation` owns the narrow contract through which a family participates.

A family interpretation provides:

- The declaration type or types it owns.
- Supported complete schema keys.
- Semantic projection after expected-key schema canonicalization.
- Definition reconstruction checks.
- Relationship facts extracted from an admitted declaration.
- Contained-declaration identity fragments.
- Declaration locator role.
- Selector eligibility.
- Capability-target eligibility.
- Family-specific diagnostics and identity validation.

The interpretation contract contains no:

- Source driver access.
- Filesystem path access.
- Store persistence access.
- Runtime execution access.
- Content inventory.
- Secret plaintext access.
- Application topology access.

### Canonical declaration decoding

The canonical JSON and YAML decoders remain shared mechanics.

Their sequence is:

1. Receive bounded candidate bytes from Ingest.
2. Read declaration header syntax.
3. Resolve a complete expected schema key through the assembled declaration interpretation registry.
4. Invoke generic expected-key schema canonicalization.
5. Invoke the owning family semantic interpretation.
6. Produce decoded values with actual source origin evidence.
7. Pass Definition values to generic Definition admission through Ingest.

The generic JSON Schema provider remains unaware of `type`, `apiVersion`, legacy headers, omitted-version behavior, or family declaration vocabulary.

### Typed Definition reconstruction

Every family with typed reads follows the same ownership split:

| Boundary                                       | Owner                                              |
| ---------------------------------------------- | -------------------------------------------------- |
| Schema execution and canonical source document | Generic schema catalog plus family semantic codec. |
| Immutable Definition digest admission          | Generic Definition.                                |
| Typed reconstruction from admitted Definition  | Owning family contract package.                    |
| Artifact-to-current-Definition linkage         | Generic Artifact.                                  |
| Runtime material preparation                   | Owning family or outer runtime integration.        |

A typed reconstruction validates:

- Artifact kind.
- Schema identity.
- Logical identity.
- Definition linkage.
- Family-specific immutable invariants.

It does not execute the schema again or recalculate the Definition digest.

## Composition and capability planning

### Composition ownership

`core/composition` owns:

- Named relationship lookup.
- Configured lookup-scope order.
- Located relationship resolution.
- Selector expansion.
- Direct-member resolution.
- Recursive capability planning.
- Alias traversal.
- Ambiguity handling.
- Cycle detection.
- Graph limits.
- Partial relationship results.
- Target identity and evidence.
- Capability-plan projection.

Family packages supply the relationship facts and semantic rules.

### Direct membership and recursive plans

Plugin direct-member inspection and full capability planning are distinct operations.

| Operation                 | Behavior                                                                                                                                              |
| ------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------- |
| Plugin direct membership  | Resolves one Plugin’s immediate member relationships, including named references, selectors, contained members, aliases, availability, and ambiguity. |
| Recursive capability plan | Recursively expands the selected artifact graph according to family relationship facts and graph limits.                                              |

Plugin direct-membership reads do not recursively resolve an Agent’s Tools, a Skill’s Tools, an MCP’s Policy, or a Workflow’s nodes merely to list Plugin members.

### Lookup scope binding

The declaration language retains the existing portable `builtin` scope token.

The token is not a Root ID, authorization grant, or protected-content privilege.

Application assembly supplies the actual scope binding:

| Scope                   | Application-supplied meaning                                                  |
| ----------------------- | ----------------------------------------------------------------------------- |
| Current Root            | The Root in which the originating Artifact exists.                            |
| Composition Source      | The explicitly selected Source context used by Workspace composition.         |
| Built-in scope          | The configured protected application artifact namespace.                      |
| Direct capability scope | A configured capability provider eligible for the requested family and scope. |

The composition layer does not scan arbitrary Roots.

### Capability targets

The current `MappedTarget.Builtin` flag is replaced by explicit target provenance and configured scope binding.

A capability target records:

- Target form: source-backed Artifact or supplied direct capability.
- Artifact reference where an Artifact exists.
- Family declaration type.
- Capability provider identity where applicable.
- Provider-local identifier.
- Logical name.
- Configuration or revision evidence.
- Scope provenance.
- Availability diagnostics.

A direct capability never fabricates:

- Artifact IDs.
- Source bindings.
- Definition bodies.
- Artifact local data.
- Generic Artifact lifecycle state.

### Direct capability inputs

The initial direct-capability inputs are:

| Family         | Direct capability input                                                                  |
| -------------- | ---------------------------------------------------------------------------------------- |
| Tool           | Registered Go Tool metadata.                                                             |
| Model          | Runtime-neutral Model target provider supplied by application inference integration.     |
| Model Provider | Installed adapter descriptor supplied by application inference integration.              |
| MCP            | Optional runtime-neutral server-preparation provider where supplied.                     |
| Other families | No direct capability provider until a real independently supplied implementation exists. |

The direct capability path remains available for later artifact families without requiring a generic built-in registry or universal provider descriptor.

### Explicit composition refresh

`core/composition/refresh` owns reachable declaration preparation and refresh.

Its sequence is:

1. Resolve the declaration graph using committed state.
2. Identify selector bases and local declaration locators requiring Source discovery preparation.
3. Ask the relevant family for its discovery requirement.
4. Apply Source-owned discovery mutation.
5. Refresh the affected Sources.
6. Repeat within the configured graph-depth and refresh-closure bounds.
7. Return explicit refresh results and diagnostics.

Normal composition resolution has no refresh capability.

## Plugin replacement of Collection

The current `internal/collection` package is removed after migration.

The new Plugin API provides:

- Plugin read.
- Plugin list.
- Plugin create.
- Plugin update.
- Plugin delete.
- Plugin enablement.
- Direct member add.
- Direct member remove.
- Direct member list.
- Reverse membership inspection.
- Managed Plugin package publication.
- Baseline restrictions.
- Membership policy validation.

The current family APIs that expose Collection operations are removed.

The application or frontend identifies a Plugin’s role through the family profile that created or owns it. It does not receive a separate Collection artifact type.

### Member identity

Plugin owns one member identity model used by:

- Display ordering.
- Duplicate detection.
- Add-or-ensure behavior.
- Removal selection.
- Reverse membership reporting.
- Generated package preparation.

The current mismatch between raw declaration ordering and normalized relationship ordering is removed.

### Relationship ownership

A Plugin member is a declaration relationship, not an ownership edge.

The following behavior remains:

| Event                       | Result                                                                          |
| --------------------------- | ------------------------------------------------------------------------------- |
| Plugin member removal       | The relationship is removed; the target Artifact remains.                       |
| Target Artifact removal     | The Plugin relationship remains and resolves unavailable.                       |
| Target Artifact replacement | The relationship resolves according to normal identity and locator rules.       |
| Plugin package deletion     | Plugin Artifact lifecycle occurs independently from target Artifacts.           |
| Contained declaration       | Remains a source-emitted Artifact and is not managed as a direct Plugin member. |

## Built-in content, generated catalogs, and preload

### Application content layout

Built-in content moves outside the reusable LLM Artifactory library.

| Path                                   | Responsibility                                                                                                               |
| -------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------- |
| `internal/artifactbuiltin/agent/`      | Embedded Agent package files and generated Agent catalog payload.                                                            |
| `internal/artifactbuiltin/skill/`      | Embedded Skill package files and generated Skill catalog payload.                                                            |
| `internal/artifactbuiltin/mcp/`        | Embedded MCP package files and generated MCP catalog payload.                                                                |
| `internal/artifactbuiltin/tool/`       | Generated Go Tool package inventory and Tool Plugin inventory.                                                               |
| `internal/artifactbuiltin/model/`      | Generated Model and Model Provider package inventory.                                                                        |
| `internal/artifactbuiltin/workspace/`  | Embedded default Workspace policy content.                                                                                   |
| `internal/artifactsetup/registration/` | Application-selected codecs, decoders, family interpretations, locator handlers, and direct capability providers.            |
| `internal/artifactsetup/topology/`     | Application Root IDs, Source IDs, Source storage keys, protection policy, retention policy, and baseline identity.           |
| `internal/artifactsetup/install/`      | Protected installation plans, generated package registrations, startup ordering, and family installation lifecycle bindings. |
| `internal/artifactsetup/workspace/`    | Workspace local policy Source preparation and policy-content injection.                                                      |

No package under `llmartifactory-go` imports `artifactbuiltin`.

### Generated content path

The generated-content sequence remains:

1. Family package preparation validates final package bytes.
2. Generic Install compilation uses the ordinary Store path.
3. Generic Source refresh admits Definitions and synchronizes Artifacts.
4. Expected Artifact identity and Definition digests are verified.
5. Generic compiled package evidence is generated.
6. Application content packages embed that evidence.
7. Generic Install registers the evidence at startup.
8. Runtime verifies source-content witnesses before using generated evidence.

The generated content path remains equivalent to ordinary admission:

```text
source bytes
  → family source adapter
  → expected schema validation
  → family semantic projection
  → Definition admission
  → Artifact synchronization
```

### Preload and immutable content

The generic Install layer owns the compiled package wire format and can own:

- One-time generated-payload decoding.
- One-time generated-payload fingerprint calculation.
- Explicit generated-payload preload.
- Immutable ownership of decoded generated package values.

Artifact families own typed Definition projection caches where those projections materially improve their normal read paths.

The generic embedded filesystem provider receives an explicit immutable-content registration mode.

That mode records:

- Provider identity.
- Content evidence.
- Immutable revision.
- Registration lifetime.

It allows embedded immutable content to reuse verified evidence rather than recomputing a full tree fingerprint for every snapshot open and confirmation.

Arbitrary `fs.FS` values remain mutable by default and retain normal fingerprint and confirmation behavior.

## Generic Artifactory additions included in this proposal

### Source-owned declaration discovery preparation

The generic Source entity already owns discovery configuration. The remaining generic operation is an explicit Source-owned discovery preparation command.

It accepts:

- Root ID.
- Source ID.
- Expected Source revision.
- Required declaration locator or directory scope.
- Required decoder hint.
- Authoritative-scope requirement.
- The caller’s ownership intent for discovery mutation.

It performs:

- Request validation.
- Current Source read.
- Discovery merge or explicit replacement.
- Revision-checked Source update.
- Preservation of unrelated Source discovery configuration.

It does not refresh the Source. Refresh remains explicit.

This removes generic discovery mutation from the current Collection package and makes the operation reusable by Agent, Skill, MCP, Model, Plugin, and Workspace behavior.

Target placement:

| Path                                 | Responsibility                                                   |
| ------------------------------------ | ---------------------------------------------------------------- |
| `artifactory-go/store/source/`       | Public Source-owned discovery preparation contract and behavior. |
| `artifactory-go/store/source/model/` | Generic discovery requirement value.                             |
| `artifactory-go/provider/sqlite/`    | Existing Source revision enforcement only; no family policy.     |

### Complete Resource verification-session participation

The current verification session participates in Artifact resolution and entry reads, but not consistently in Source stat and tree reads.

The Resource flow is extended so one session can retain and confirm the same Source snapshot for:

- Artifact resolution.
- Source entry reads.
- Source entry stat.
- Source tree reads.
- Trusted native-path resolution.

The session distinguishes:

| Read mode                     | Required evidence                                                                                                       |
| ----------------------------- | ----------------------------------------------------------------------------------------------------------------------- |
| Resource-only Source read     | Source revision, Source generation, and snapshot confirmation.                                                          |
| Artifact-backed verified read | Artifact state, Definition link, Source revision, refresh generation, source-content digest, and snapshot confirmation. |

A Source that is valid for resource-only access does not require a declaration refresh record.

Target placement:

| Path                                           | Responsibility                                                               |
| ---------------------------------------------- | ---------------------------------------------------------------------------- |
| `artifactory-go/store/flow/resource/`          | Public session and verified read behavior.                                   |
| `artifactory-go/store/flow/resource/internal/` | Shared snapshot retention, stat/tree execution, and completion confirmation. |
| `artifactory-go/store/source/`                 | Bounded snapshot read primitives only.                                       |

### Explicit package-publication outcome

The generic ManagePackage flow currently has a physical publication boundary followed by Source acknowledgment and catalog refresh. Those boundaries remain distinct.

A typed generic outcome records:

- Physical package state.
- Observed Source generation.
- Whether Source metadata acknowledgment completed.
- Whether refresh completed.
- Whether the expected Artifact was verified.
- Whether recovery is required after a completed physical mutation.

This replaces ambiguous plain errors after known physical mutation.

Target placement:

| Path                                             | Responsibility                                |
| ------------------------------------------------ | --------------------------------------------- |
| `artifactory-go/store/flow/managepackage/model/` | Publication and removal outcome values.       |
| `artifactory-go/store/flow/managepackage/`       | Outcome construction and workflow sequencing. |
| `artifactory-go/provider/managedfs/`             | Physical write evidence only.                 |

Family package services consume the generic outcome rather than adding family-specific retry engines.

### Atomic Artifact local-state cleanup

The current Model cleanup path can remove generic overlays and bindings in one operation, then separately remove a mutable Artifact Data namespace.

The generic Store gains explicit support for:

- Revision-checked owned Artifact Data field replacement.
- Revision-checked owned Artifact Data field deletion.
- Atomic cleanup plans covering:
  - selected Artifact Data fields,
  - selected protected overlay namespaces,
  - related secret bindings,
  - durable physical secret cleanup queue insertion.
- Preservation of unrelated Artifact Data and unrelated feature namespaces.

Target placement:

| Path                                         | Responsibility                                                       |
| -------------------------------------------- | -------------------------------------------------------------------- |
| `artifactory-go/store/artifact/`             | Generic namespaced local-data mutation.                              |
| `artifactory-go/store/flow/artifactcleanup/` | Atomic multi-owner local cleanup coordination.                       |
| `artifactory-go/provider/sqlite/`            | Transactional execution of explicit cleanup and local-data commands. |

Model Provider, Model, and MCP select their own namespaces. Generic Artifactory does not interpret their payloads.

### Secret recovery after ambiguous publication

The existing secret replacement sequence can receive an error after an uncertain binding publication. A cleanup request must not convert a still-active secret record into cleanup state merely because the caller received an error.

The generic Secret lifecycle is tightened to distinguish:

| State   | Meaning                                                      |
| ------- | ------------------------------------------------------------ |
| Pending | Physical value was staged but has not been confirmed active. |
| Active  | A current binding references the value.                      |
| Cleanup | The value is detached and eligible for physical deletion.    |

The recovery path verifies binding references before queueing cleanup. Physical deletion occurs only after durable cleanup eligibility is established.

Target placement:

| Path                                            | Responsibility                                 |
| ----------------------------------------------- | ---------------------------------------------- |
| `artifactory-go/store/secret/`                  | Binding and lifecycle behavior.                |
| `artifactory-go/provider/sqlite/local_state.go` | Reference-aware record transition enforcement. |
| `artifactory-go/store/secret/value/`            | Physical opaque value storage only.            |

### Local opener ownership transfer

`compose/local.Open` transfers caller-supplied secret-backend ownership only after the complete opening operation succeeds, including retained Root initialization.

A failed local opening leaves caller-supplied secret storage caller-owned.

Target placement:

| Path                                        | Responsibility                                          |
| ------------------------------------------- | ------------------------------------------------------- |
| `artifactory-go/compose/local/open.go`      | Successful ownership transfer boundary.                 |
| `artifactory-go/compose/local/resources.go` | Local resource cleanup and exactly-once close behavior. |

### Generic generated-content preload

The generic generated package-set wire representation is already owned by Install. The preload mechanism remains there rather than being duplicated in each family built-in catalog package.

Target placement:

| Path                                             | Responsibility                                                                           |
| ------------------------------------------------ | ---------------------------------------------------------------------------------------- |
| `artifactory-go/store/flow/install/generated.go` | Decode, canonicalization, fingerprinting, and preload of generic generated package sets. |
| `artifactory-go/store/flow/install/model/`       | Immutable generated-content holder where needed.                                         |

### Utility placement

No new generic utility package is created merely to reserve a folder.

The following existing packages remain where they are:

- `internal/jsonutil`
- `internal/yamlutil`
- `internal/clockutil`
- `internal/uuidutil`
- `internal/cryptoutil`

If a genuinely generic helper later has two independent Artifactory owners, it belongs beneath:

| Path                                                | Rule                                                                 |
| --------------------------------------------------- | -------------------------------------------------------------------- |
| `artifactory-go/contractutil/`                      | Directory only. It contains no Go package and no direct files.       |
| `artifactory-go/contractutil/<responsibility>util/` | A responsibility-specific package with a narrow documented contract. |

Current examples that remain family-owned rather than being prematurely generalized:

| Current behavior                       | Owner after split          |
| -------------------------------------- | -------------------------- |
| Prepared Agent import envelope signing | `agent/import`             |
| Plugin typed Definition cache          | `plugin`                   |
| Family typed Definition caches         | Owning family              |
| MCP logical secret-reference mapping   | `mcp/secret`               |
| Model preference payload               | `modelprovider/preference` |

## Registration and assembly

### No universal registration descriptor

The aggregate receives named registrations:

- Schema codecs.
- Source decoders.
- Family interpretation registrations.
- Locator handlers.
- Direct capability providers.
- Family local-state namespace registrations.
- Family-specific service configuration.

A universal object containing all of these is not introduced.

For example:

| Concern                   | Registration owner                |
| ------------------------- | --------------------------------- |
| Agent schema codec        | `agent/contract/v1`               |
| Agent Markdown decoder    | `agent/sourceformat/markdown`     |
| Skill `SKILL.md` decoder  | `skill/sourceformat/skillmd`      |
| MCP config decoder        | `mcp/sourceformat/config`         |
| Tool Go metadata provider | Application Tool integration      |
| Model target provider     | Application inference integration |
| Built-in package sets     | Application content setup         |
| Protected Root mapping    | Application topology setup        |

### LLM aggregate construction

The LLM aggregate construction sequence is:

1. Receive an assembled generic Store.
2. Validate all selected schema codecs.
3. Validate all selected Source decoders.
4. Build the generic schema catalog through Artifactory.
5. Bind expected-key schema canonicalization to relevant declaration decoders.
6. Validate family interpretation registrations.
7. Build the composition registry.
8. Bind locator handlers.
9. Construct family services with narrow generic capabilities.
10. Construct the aggregate family surface.
11. Retain only LLM-owned in-memory state and close behavior.

The generic Store remains owned by the deployment that opened it.

### Local deployment

No second LLM-specific local opener is introduced.

The existing `artifactory-go/compose/local` package remains the owner of:

- Local layout.
- SQLite opening.
- Source driver construction.
- Embedded filesystem provider construction.
- JSON Schema provider selection.
- Secret backend ownership.
- Generic Store shutdown.

`artifactsetup` opens the generic local Store, attaches LLM Artifactory, prepares protected installation, and then starts runtime integrations.

This preserves Proposal A’s single generic aggregate and single local deployment opening boundary.

## Validation boundaries

| Boundary                                       | Owner                                                     |
| ---------------------------------------------- | --------------------------------------------------------- |
| Family registration                            | Root LLM aggregate construction.                          |
| Schema key and decoder compatibility           | Family contract registration plus generic schema binding. |
| Raw declaration parsing                        | Source format adapter or canonical declaration decoder.   |
| Header dispatch                                | Shared declaration core.                                  |
| Expected-key schema validation                 | Generic schema catalog.                                   |
| Family semantic validation                     | Owning family contract.                                   |
| Definition digest admission                    | Generic Definition.                                       |
| Artifact source state                          | Generic Artifact synchronization.                         |
| Relationship legality                          | Owning containing family.                                 |
| Graph existence, ambiguity, cycles, and limits | Shared composition.                                       |
| Source freshness                               | Generic Resource verified access.                         |
| Package byte normalization                     | Owning family package preparation.                        |
| Physical publication                           | Generic ManagePackage.                                    |
| Local overlay semantics                        | Owning family settings or installation package.           |
| Secret value lifecycle                         | Generic Secret.                                           |
| Plaintext secret use                           | Trusted runtime integration.                              |
| Runtime execution permission                   | Runtime aggregate and application policy.                 |

The same semantic rule does not appear in both generic Artifactory and LLM Artifactory.

Independent checks remain where the boundary changes:

- Admission validates declaration meaning.
- Composition validates graph meaning.
- Package publication validates physical and catalog expectations.
- Resource verification validates time-sensitive Source evidence.
- Runtime validates execution-specific capability requirements.

## Work packages

### Package A: Establish the LLM library, content, and application-setup boundaries

#### Purpose

This package establishes the destination ownership boundary without introducing compatibility layers or changing generic Store behavior.

It is an enabler package. It moves code near its final owner, updates internal callers directly, and removes old import ownership.

#### Scope

- Create `internal/llmartifactory-go`.
- Create the `core` directory structure.
- Create top-level artifact-family folders.
- Move existing LLM artifact behavior from `artifactcontract`, `collection`, and family Store packages into destination ownership areas.
- Create `internal/artifactbuiltin`.
- Create `internal/artifactsetup`.
- Move application content and topology ownership out of reusable artifact packages.
- Add the root LLM aggregate attachment boundary over the generic Store.
- Remove old package imports from wrappers and application initialization.
- Remove obsolete old package ownership rather than preserving aliases.

#### Destination folders and file groups

| Destination                           | Required file groups                                                            |
| ------------------------------------- | ------------------------------------------------------------------------------- |
| `llmartifactory-go/`                  | Aggregate API, configuration, construction, registration validation, lifecycle. |
| `llmartifactory-go/core/`             | Empty organizational folder with no direct Go package.                          |
| `llmartifactory-go/core/declaration/` | Initially moved shared declaration grammar files.                               |
| `llmartifactory-go/core/composition/` | Initially moved shared resolver files.                                          |
| `llmartifactory-go/<artifact-type>/`  | Initially moved family files in their nearest destination ownership area.       |
| `artifactbuiltin/<family>/`           | Embedded assets and generated catalog payloads.                                 |
| `artifactsetup/`                      | Application topology, registration, installation, and startup composition.      |

#### Responsibility movements completed in this package

| Current owner                      | New owner                                                   |
| ---------------------------------- | ----------------------------------------------------------- |
| `internal/artifactcontract`        | LLM core and artifact-family folders.                       |
| `internal/collection`              | Plugin artifact family.                                     |
| Family `store/builtin` packages    | Application content packages.                               |
| `artifactcontract/topology`        | Family package conventions plus application topology setup. |
| `artifactcontract/builtin`         | Application content and setup.                              |
| Application wrapper business logic | LLM family services or application setup.                   |

#### Required input set

The implementation input set for this package is limited to:

- `internal/artifactcontract/**`
- `internal/collection/**`
- `internal/agent/store/**`
- `internal/skill/store/**`
- `internal/tool/store/**`
- `internal/model/store/**`
- `internal/mcp/store/**`
- `internal/workspace/store/**`
- `internal/workspace/defaultpolicy/**`
- `cmd/agentgo/app.go`
- `cmd/agentgo/wrapper_artifactstore.go`
- `cmd/agentgo/wrapper_builtin_topology.go`
- `cmd/agentgo/wrapper_agent_store.go`
- `cmd/agentgo/wrapper_skill_store.go`
- `cmd/agentgo/wrapper_tool_store.go`
- `cmd/agentgo/wrapper_model_store.go`
- `cmd/agentgo/wrapper_mcp_bootstrap.go`
- `cmd/agentgo/wrapper_workspace_store.go`
- `artifactory-go/store/compose/**`
- `artifactory-go/compose/local/**`

The generic provider implementations, SQLite implementation, runtime execution packages, frontend files, and generated schema contents are not required for this package except where a moved source file imports one of their public contracts.

#### Completion result

At completion:

- The LLM artifact library exists as a distinct repository unit.
- Product content has a separate ownership location.
- Application topology has a separate ownership location.
- The old `artifactcontract` and `collection` locations no longer own implementation.
- No compatibility aliases or import-forwarding packages remain.
- The generic local Store opener remains the only local deployment opener.

### Package B: Complete generic Artifactory responsibilities required by LLM Artifactory

#### Purpose

This package completes generic responsibilities that currently appear in feature packages or whose current behavior is insufficient for the split.

#### Scope

- Add Source-owned declaration discovery preparation.
- Complete Resource verification-session support for stat and tree reads.
- Add typed ManagePackage physical/catalog outcome values.
- Add atomic namespaced Artifact Data cleanup behavior.
- Correct secret recovery after ambiguous binding publication.
- Correct local opener resource-ownership transfer.
- Add generic generated package-set preload behavior.
- Add immutable embedded filesystem evidence registration.

#### Destination folders and file groups

| Destination                                      | Required file groups                                     |
| ------------------------------------------------ | -------------------------------------------------------- |
| `artifactory-go/store/source/`                   | Discovery preparation contract and Source behavior.      |
| `artifactory-go/store/source/model/`             | Generic discovery requirement values.                    |
| `artifactory-go/store/flow/resource/`            | Session API and verified stat/tree behavior.             |
| `artifactory-go/store/flow/resource/internal/`   | Shared retained snapshot behavior.                       |
| `artifactory-go/store/flow/managepackage/`       | Publication/removal outcome behavior.                    |
| `artifactory-go/store/flow/managepackage/model/` | Outcome values and recoverable-state values.             |
| `artifactory-go/store/artifact/`                 | Namespaced local-data mutation.                          |
| `artifactory-go/store/flow/artifactcleanup/`     | Atomic cleanup-plan coordination.                        |
| `artifactory-go/store/secret/`                   | Pending/active/cleanup recovery behavior.                |
| `artifactory-go/provider/sqlite/`                | Transactional persistence for explicit generic commands. |
| `artifactory-go/store/flow/install/`             | Generated-content preload.                               |
| `artifactory-go/provider/iofs/`                  | Immutable content evidence support.                      |
| `artifactory-go/compose/local/`                  | Ownership-transfer correction.                           |

#### Required input set

The implementation input set for this package is limited to:

- `internal/artifactory-go/store/source/**`
- `internal/artifactory-go/store/flow/resource/**`
- `internal/artifactory-go/store/flow/managepackage/**`
- `internal/artifactory-go/store/source/managedpackage/**`
- `internal/artifactory-go/store/artifact/**`
- `internal/artifactory-go/store/flow/artifactcleanup/**`
- `internal/artifactory-go/store/secret/**`
- `internal/artifactory-go/store/overlay/**`
- `internal/artifactory-go/store/flow/install/**`
- `internal/artifactory-go/store/compose/**`
- `internal/artifactory-go/compose/local/**`
- `internal/artifactory-go/provider/sqlite/**`
- `internal/artifactory-go/provider/iofs/**`
- `internal/artifactory-go/provider/managedfs/**`

No LLM family source, application wrapper source, frontend source, or runtime source is required except for later caller migration.

#### Completion result

At completion:

- Family services no longer need a Collection-owned discovery helper.
- Resource sessions can provide coherent verified tree and stat operations.
- Family publication code receives observable generic publication outcomes.
- Model and MCP cleanup can use one generic transactional cleanup boundary.
- Secret lifecycle remains safe after ambiguous persistence outcomes.
- Embedded immutable content has a generic fast path without weakening mutable `fs.FS` verification.

### Package C: Move declaration semantics into the core and artifact families

#### Purpose

This package removes the current central declaration-type switches while preserving one declaration grammar and one canonical expected-key schema path.

#### Scope

- Move shared declaration grammar into `core/declaration`.
- Move family declarations and schemas into corresponding `<family>/contract/v1` folders.
- Replace the central schema-codec list with explicit family codec registration at aggregate construction.
- Replace central tree validation with family interpretation.
- Replace central walk behavior with family relationship extraction and family identity fragments.
- Keep generic traversal mechanics private to the core.
- Move Agent managed profile behavior into Agent contract ownership.
- Move Model reference behavior into Model and Model Provider ownership.
- Move canonical JSON/YAML decoder mechanics into the declaration core.
- Preserve declaration dispatch outside the generic JSON Schema provider.

#### Destination folders and file groups

| Destination                                 | Required file groups                                                                              |
| ------------------------------------------- | ------------------------------------------------------------------------------------------------- |
| `core/declaration/`                         | Header, Entry, generic member, generic locator, dispatch, ordering, generic validation.           |
| `core/declaration/decoder/`                 | Canonical JSON/YAML declaration decoding.                                                         |
| `core/declaration/interpretation/`          | Family interpretation contract and registry.                                                      |
| `<family>/contract/v1/`                     | Schema assets, document values, codecs, semantic checks, reconstruction, relationship extraction. |
| `agent/contract/v1/`                        | Managed Agent profile restrictions.                                                               |
| `model/domain/` and `modelprovider/domain/` | Model and Provider reference values.                                                              |

#### Required input set

The implementation input set for this package is limited to:

- `internal/llmartifactory-go/core/**` created in the library-boundary work package.
- The moved equivalents of:
  - `internal/artifactcontract/declaration/**`
  - `internal/artifactcontract/codec/**`
  - `internal/artifactcontract/decoder/**`
  - `internal/artifactcontract/providercanonical/**`
  - `internal/artifactcontract/providermarkdown/**`
- All declaration schema files currently under:
  - `artifactcontract/declaration/agentv1`
  - `artifactcontract/declaration/loopv1`
  - `artifactcontract/declaration/mcppolicyv1`
  - `artifactcontract/declaration/mcpv1`
  - `artifactcontract/declaration/modelproviderv1`
  - `artifactcontract/declaration/modelv1`
  - `artifactcontract/declaration/pluginv1`
  - `artifactcontract/declaration/skillv1`
  - `artifactcontract/declaration/teamv1`
  - `artifactcontract/declaration/textv1`
  - `artifactcontract/declaration/toolv1`
  - `artifactcontract/declaration/workflowv1`
  - `artifactcontract/declaration/workspacev1`
- Public generic contracts:
  - `artifactory-go/store/definition/**`
  - `artifactory-go/store/definition/schema/**`
  - `artifactory-go/store/source/ingest/**`

#### Completion result

At completion:

- The generic JSON Schema provider still accepts only complete expected schema keys.
- Header dispatch is owned by the LLM declaration core.
- A new artifact-family declaration no longer requires editing a central family switch.
- Family typed reconstruction no longer re-executes admission work.
- Existing canonical bytes, Definition digests, schema IDs, and dispatch behavior remain stable.

### Package D: Establish composition and replace Collection with Plugin

#### Purpose

This package creates one shared composition owner and one Plugin owner.

#### Scope

- Move generic resolver mechanics into `core/composition`.
- Move locator resolver contracts into `core/composition/locator`.
- Move explicit refresh closure into `core/composition/refresh`.
- Replace `MappedTarget.Builtin` with explicit target provenance and scope binding.
- Add direct-member and recursive-plan distinction.
- Move Collection behavior into Plugin.
- Add Plugin single-type, mixed-type, and legacy-unconstrained membership policy.
- Move direct member ordering, reverse membership inspection, and membership mutation under Plugin.
- Move feature-specific Plugin profiles into Agent, Skill, MCP, and Tool owners.
- Remove family Collection forwarding APIs.

#### Destination folders and file groups

| Destination                  | Required file groups                                                                 |
| ---------------------------- | ------------------------------------------------------------------------------------ |
| `core/composition/`          | Public composition plans, scope values, target values, and read-only resolution API. |
| `core/composition/locator/`  | Locator contracts, registered handlers, and path locator behavior.                   |
| `core/composition/refresh/`  | Explicit source-preparation and refresh closure.                                     |
| `core/composition/internal/` | Private graph traversal, cycle detection, selector processing, and lookup mechanics. |
| `plugin/`                    | Plugin service and public Plugin API.                                                |
| `plugin/domain/`             | Membership policy, Plugin views, baseline constraints, and direct member results.    |
| `plugin/membership/`         | Direct/reverse membership behavior.                                                  |
| `plugin/package/`            | Plugin package behavior.                                                             |

#### Required input set

The implementation input set for this package is limited to:

- `internal/llmartifactory-go/core/declaration/**`
- `internal/llmartifactory-go/core/composition/**`
- Moved equivalents of:
  - `internal/artifactcontract/locator/**`
  - `internal/artifactcontract/resolve/**`
  - `internal/collection/**`
- Family relationship code from:
  - Agent contract and import behavior.
  - Skill contract.
  - MCP contract and Policy references.
  - Plugin contract.
  - Team, Loop, Workflow, and Workspace contracts.
  - Tool and Model direct-target behavior.
- Public generic contracts:
  - `artifactory-go/store/artifact/**`
  - `artifactory-go/store/artifact/catalog/**`
  - `artifactory-go/store/flow/resource/**`
  - `artifactory-go/store/flow/refresh/**`
  - `artifactory-go/store/source/**`

#### Completion result

At completion:

- Plugin is the only managed membership artifact service.
- Family-specific Collection APIs no longer exist.
- Direct Plugin membership does not recursively expand all downstream capabilities.
- Recursive capability plans remain explicit.
- Scope resolution is supplied by application configuration rather than hard-coded built-in Root identity.
- Normal resolution remains read-only.
- Refresh closure remains explicit and separate.

### Package E: Consolidate artifact-family services, local state, and runtime-neutral capability preparation

#### Purpose

This package completes ownership moves from feature Store packages into their artifact-family owners.

#### Scope

- Move Text materialization to Text.
- Move Agent import/export and Agent package operations to Agent.
- Move Skill source adaptation, package handling, and materialization to Skill.
- Move Go Tool metadata-based capability preparation to Tool.
- Remove SDK Tool support.
- Split Model and Model Provider services.
- Move Model and Provider settings, credentials, and preferences to their owners.
- Move MCP installation and secret semantics to MCP.
- Move MCP Policy semantics to MCP Policy.
- Move Team, Loop, and Workflow plans to their owners.
- Move Workspace artifact behavior and effective Workspace selection to Workspace.
- Remove family-owned `consumerapi`, `providerapi`, `domain`, and `builtin` layering where the layers only split one owner.

#### Required input sets

This package can be implemented through independently reviewable family bundles. Each bundle has a bounded required source set.

| Family bundle            | Required input folders and files                                                                                                                                                                                                                                          |
| ------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Text, Agent, Skill       | `llmartifactory-go/text/**`, `agent/**`, `skill/**`; moved equivalents of `artifactcontract/materializetext`, `agent/store/domain`, `agent/store/consumerapi`, `skill/store/domain`, `skill/store/materialize`, `skill/store/providerapi`, and `skill/store/consumerapi`. |
| Tool                     | `llmartifactory-go/tool/**`; moved equivalents of `tool/store/domain`, `tool/store/consumerapi`, `tool/aggregate/target.go`, `tool/aggregate/service.go`, and `tool/llmtoolsadapter`.                                                                                     |
| Model and Model Provider | `llmartifactory-go/model/**`, `modelprovider/**`; moved equivalents of `model/store/domain`, `model/store/overlay`, `model/store/consumerapi`, `model/aggregate/target.go`, `model/aggregate/default_provider.go`, and model preference wrapper behavior.                 |
| MCP and MCP Policy       | `llmartifactory-go/mcp/**`, `mcppolicy/**`; moved equivalents of `mcp/store/domain`, `mcp/store/domain/server`, `mcp/store/domain/secret`, `mcp/store/domain/policy`, `mcp/store/overlay`, `mcp/store/providerapi`, and artifact-facing MCP consumer behavior.            |
| Team, Loop, Workflow     | `llmartifactory-go/team/**`, `loop/**`, `workflow/**`; moved declaration and resolver behavior only.                                                                                                                                                                      |
| Workspace                | `llmartifactory-go/workspace/**`; moved equivalents of artifact-facing portions of `workspace/store/consumerapi`, `workspace/store/domain`, and `workspace/defaultpolicy` contract use.                                                                                   |

#### Completion result

At completion:

- Each artifact type owns its declaration, domain values, family service, source adaptation, package rules, and local-state semantics.
- Family Store wrappers no longer contain domain behavior.
- Generic Source, Artifact, Definition, Resource, Secret, Overlay, ManagePackage, and Install contracts remain the only generic dependencies.
- Runtime execution packages receive runtime-neutral family outputs rather than reconstructing artifact meaning themselves.
- Tool behavior supports Go implementations only.

### Package F: Bind supplied content, direct capabilities, application setup, runtime adapters, and wrappers

#### Purpose

This package completes the runtime-independent library boundary and moves application-specific assembly out of the library.

#### Scope

- Move embedded assets and generated catalogs into `artifactbuiltin`.
- Move application Root, Source, retention, protection, baseline, and installation setup into `artifactsetup`.
- Bind generic generated-content preload to application-supplied content registrations.
- Bind immutable embedded filesystem registrations.
- Bind Go Tool metadata provider.
- Bind Model target and adapter providers.
- Bind MCP runtime adapters and trusted plaintext secret access.
- Bind Skill runtime adapters and native-path materialization.
- Bind Workspace prompt, Skill, and MCP runtime adapters.
- Reduce Wails wrappers to transport, recovery, and request/response handling.
- Preserve startup ordering and shutdown ordering.
- Remove obsolete built-in installer wrappers that only forward to generic Install.

#### Destination folders and file groups

| Destination                              | Required file groups                                                                                                 |
| ---------------------------------------- | -------------------------------------------------------------------------------------------------------------------- |
| `internal/artifactbuiltin/**`            | Embedded content, generated catalogs, generated inventory indexes.                                                   |
| `internal/artifactsetup/registration/**` | Explicit application schema, decoder, interpreter, locator, and direct-capability binding.                           |
| `internal/artifactsetup/topology/**`     | Root and Source declarations, retained and protected policies, baseline identities.                                  |
| `internal/artifactsetup/install/**`      | Protected installation plans, compiled registration selection, preload, baseline provisioning, and startup ordering. |
| Existing runtime aggregate packages      | Runtime adapters consuming family outputs.                                                                           |
| `cmd/agentgo/wrapper_*.go`               | Thin transport wrappers only.                                                                                        |

#### Required input set

The implementation input set for this package is limited to:

- `internal/artifactbuiltin/**` after the library-boundary work package.
- `internal/artifactsetup/**` after the library-boundary work package.
- Existing family built-in folders before migration:
  - `internal/agent/store/builtin/**`
  - `internal/skill/store/builtin/**`
  - `internal/mcp/store/builtin/**`
  - `internal/model/store/builtin/**`
  - `internal/tool/store/builtin/**`
- Runtime and aggregate adapters:
  - `internal/agent/**`
  - `internal/skill/aggregate/**`
  - `internal/skill/runtime/**`
  - `internal/tool/aggregate/**`
  - `internal/tool/runtime/**`
  - `internal/model/aggregate/**`
  - `internal/model/inferenceadapter/**`
  - `internal/mcp/aggregate/**`
  - `internal/mcp/runtime/**`
  - `internal/workspace/runtime/**`
  - `internal/workspace/conversation/**`
  - `internal/workspace/store/adapter/**`
- Application assembly and wrappers:
  - `cmd/agentgo/app.go`
  - `cmd/agentgo/wrapper_artifactstore.go`
  - `cmd/agentgo/wrapper_builtin_topology.go`
  - `cmd/agentgo/wrapper_agent_store.go`
  - `cmd/agentgo/wrapper_skill_store.go`
  - `cmd/agentgo/wrapper_tool_store.go`
  - `cmd/agentgo/wrapper_tool_aggregate.go`
  - `cmd/agentgo/wrapper_model_store.go`
  - `cmd/agentgo/wrapper_model_aggregate.go`
  - `cmd/agentgo/wrapper_model_preferences.go`
  - `cmd/agentgo/wrapper_model_secret.go`
  - `cmd/agentgo/wrapper_mcp_bootstrap.go`
  - `cmd/agentgo/wrapper_mcp_store.go`
  - `cmd/agentgo/wrapper_mcp_aggregate.go`
  - `cmd/agentgo/wrapper_mcp_artifact_secrets.go`
  - `cmd/agentgo/wrapper_workspace_store.go`
  - `cmd/agentgo/wrapper_workspace_runtime.go`

#### Completion result

At completion:

- LLM Artifactory imports no application built-in content.
- Application setup owns product topology and startup ordering.
- Runtime integrations own execution and trusted runtime authority.
- Wrappers no longer contain artifact business workflows.
- The reusable LLM library can be extracted without importing FlexiGPT application topology, runtime packages, content packages, or Wails wrappers.

## Alignment with the prior Artifactory proposal and retained principles

### Retained Artifactory principles

| Proposal 1 principle                                                    | Treatment in this proposal                                                                                                                                               |
| ----------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| Root, Source, Artifact, Definition, Overlay, and Secret own their state | Retained. LLM Artifactory consumes these entities and does not absorb their lifecycle.                                                                                   |
| Flows own cross-entity sequencing                                       | Retained. Generic Refresh, Resource, ManagePackage, Install, and ArtifactCleanup remain generic. LLM composition refresh owns only LLM reachable-declaration sequencing. |
| Providers execute contracts                                             | Retained. LLM families do not select SQLite, filesystem drivers, or secret backends.                                                                                     |
| Declaration dispatch stays outside generic JSON Schema provider         | Retained. Existing dispatch behavior moves from `artifactcontract` to `core/declaration`; no generic provider behavior is added.                                         |
| No universal provider descriptor                                        | Retained. Aggregate construction receives separate named registrations.                                                                                                  |
| No broad ordinary capability with recoverable privileged methods        | Retained. Secret binding, plaintext resolution, native paths, Go Tool metadata, Go Tool invocation, and runtime connections remain separate capabilities.                |
| Catalog reads remain committed reads                                    | Retained. Composition normal resolution uses committed state. Explicit refresh closure is separate.                                                                      |
| Verified reads remain explicit                                          | Retained. Text and Skill materialization use Resource verification sessions.                                                                                             |
| Family package preparation happens before generic publication           | Retained. Families prepare bytes and expected Definition identity; ManagePackage publishes and verifies.                                                                 |
| Generic installation remains generic                                    | Retained. Generated content and application topology remain outside Install; Install keeps generic hydration and compilation behavior.                                   |
| No compatibility façades                                                | Retained. Old `artifactcontract`, `collection`, family Store forwarding APIs, and wrapper-owned business behavior are removed rather than aliased.                       |
| Construction belongs to the owner                                       | Retained. Family root packages own family construction. `store/compose` remains the single justified generic Store assembly boundary.                                    |
| No vague manager, metadata, localstate, or utility owner                | Retained. `core` is folder-only; child packages have named responsibilities. `contractutil` remains empty unless an actual generic subpackage is justified.              |

### Earlier work-package status

| Proposal 1 work                                                                       | Status in this proposal                                                                                                                                                                         |
| ------------------------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Work Package A: owner-service split and capability segregation                        | Treated as completed baseline. This proposal does not rename completed owners into new LLM abstractions.                                                                                        |
| Work Package B: generic aggregate assembly and local opening                          | Treated as completed baseline. No LLM-specific duplicate local opener or mirrored aggregate is introduced.                                                                                      |
| Work Package C: Source lifecycle publication, Definition admission, Resource sessions | Treated as completed baseline with targeted completion of stat/tree session participation and local-state cleanup gaps.                                                                         |
| Work Package D: declaration dispatch separation                                       | The supplied implementation already keeps dispatch outside the generic JSON Schema provider. This proposal relocates that responsibility from `artifactcontract` into the LLM declaration core. |

### Architectural violations removed by this proposal

| Current condition                                                                                                                             | Resulting correction                                                                                                                  |
| --------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------- |
| `artifactcontract` centrally owns family declarations, schemas, decoder dispatch, resolver structure, locators, topology, and content helpers | Shared grammar moves to the core; family semantics move to family folders; application topology and content move outside the library. |
| `collection` owns Plugin behavior plus Agent, Skill, MCP, and Tool policies                                                                   | Plugin owns Plugin behavior; each family owns its profile; application owns baselines.                                                |
| Feature Store packages mix artifact behavior, runtime behavior, content, and transport-facing APIs                                            | Artifact behavior moves to LLM family owners; runtime and transport remain outside.                                                   |
| Built-in content is imported through application-specific global helpers                                                                      | Content becomes an explicit application-supplied installation input.                                                                  |
| Wrapper code owns preferences, baseline routing, secret-reference conversion, and other artifact behavior                                     | Those responsibilities move to the owning family or application setup package.                                                        |
| MCP Store imports runtime-oriented policy and materialization behavior                                                                        | Runtime-independent policy and installation semantics move into LLM artifact owners; runtime materialization remains external.        |
| Tool contract retains SDK implementation paths despite the requested Go-only scope                                                            | SDK Tool declaration, package, target, and runtime paths are removed.                                                                 |
| Model managed Source identity assumes Root-local Source IDs despite global SQLite Source ID uniqueness                                        | Managed Source provisioning uses Source storage-key reuse and the actual returned Source identity.                                    |
| Resource sessions do not cover every verified Resource operation                                                                              | Stat and tree reads join the same session model.                                                                                      |
| Generic Artifact cleanup does not include mutable Artifact Data namespaces atomically                                                         | Generic Artifact and ArtifactCleanup commands gain explicit namespaced data cleanup.                                                  |

### Intentional architectural exceptions

The following structures remain because they represent real responsibilities rather than compatibility layers.

| Structure                       | Reason                                                                                                                               |
| ------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------ |
| `artifactory-go/store/compose`  | It is the single generic Store aggregate assembler defined by Proposal 1.                                                            |
| `artifactory-go/compose/local`  | It owns local deployment resources and local provider lifetime.                                                                      |
| `llmartifactory-go/core/`       | It is a directory-only grouping with no generic package. Its child packages own declaration grammar and composition mechanics.       |
| `<family>/local` packages       | They are explicit deployment adapters for concrete local Source kinds. Core family services remain provider-independent.             |
| `<family>/materialize` packages | They expose verified material preparation where a family genuinely needs it. Trusted native-path access remains separately supplied. |
| Runtime aggregate packages      | They own execution, runtime invalidation, connection state, inference SDK integration, and plaintext secret use.                     |
| `artifactbuiltin`               | It is application content, not reusable library behavior.                                                                            |
| `artifactsetup`                 | It is application topology and startup assembly, not generic Store or LLM family behavior.                                           |

### Generic Artifactory placement after this proposal

The following behavior remains or becomes part of generic Artifactory:

| Behavior                                                                  | Generic Artifactory location                 |
| ------------------------------------------------------------------------- | -------------------------------------------- |
| Root, Source, Artifact, Definition, Overlay, Secret lifecycle             | Existing entity packages.                    |
| Source discovery configuration and declaration-discovery preparation      | `store/source`.                              |
| Decoder execution and bounded scanning                                    | `store/source/ingest`.                       |
| Expected-key schema canonicalization                                      | `store/definition/schema`.                   |
| Artifact synchronization and lifecycle invalidation                       | `store/artifact` and `flow/refresh`.         |
| Verified reads and sessions                                               | `flow/resource`.                             |
| Managed package physical/catalog sequencing                               | `flow/managepackage`.                        |
| Protected topology, generated package wire format, preload, and hydration | `flow/install`.                              |
| Artifact local-data mutation and atomic cleanup                           | `store/artifact` and `flow/artifactcleanup`. |
| Secret binding lifecycle and durable cleanup                              | `store/secret`.                              |
| Embedded immutable filesystem evidence                                    | `provider/iofs`.                             |
| Generic local deployment opening                                          | `compose/local`.                             |

The following behavior remains outside generic Artifactory:

| Behavior                                                 | Owner after split                                 |
| -------------------------------------------------------- | ------------------------------------------------- |
| Declaration type meaning                                 | LLM family contract.                              |
| Header dispatch semantics                                | LLM declaration core.                             |
| Locator interpretation                                   | LLM composition locator package and family facts. |
| Graph resolution                                         | LLM composition.                                  |
| Plugin membership                                        | Plugin family.                                    |
| Package document filenames and family byte normalization | Artifact family package owner.                    |
| Agent import semantics                                   | Agent family.                                     |
| Skill package semantics                                  | Skill family.                                     |
| Tool implementation semantics                            | Tool family.                                      |
| Model and Provider settings semantics                    | Model and Model Provider families.                |
| MCP installation semantics                               | MCP family.                                       |
| Built-in content choice                                  | Application content and setup.                    |
| Runtime execution                                        | Existing runtime and aggregate packages.          |

This preserves Proposal A’s entity-oriented and responsibility-oriented design while giving LLM artifacts the same ownership clarity that generic Artifactory now has.
