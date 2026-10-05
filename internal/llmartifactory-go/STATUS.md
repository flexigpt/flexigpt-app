# LLM Artifactory status

## Architectural boundary

`internal/artifactory-go` remains the generic owner of Root, Source,
Definition, Artifact, Overlay, Secret, Refresh, Resource, ManagePackage,
Install, and ArtifactCleanup behavior.

`internal/llmartifactory-go` owns LLM declaration language behavior,
family interpretation, Plugin membership, composition, capability planning,
and family-specific package and local-state meaning.

`internal/artifactbuiltin` owns application-shipped embedded content and
generated catalog payloads.

`internal/artifactsetup` owns application registration selection, protected
topology, Root and Source identities, startup installation, and local
deployment assembly.

## Proposal 1 / generic Artifactory baseline

Completed generic responsibilities include:

- Entity-owned Root, Source, Definition, Artifact, Overlay, and Secret state.
- Explicit Refresh, Resource, ManagePackage, Install, and ArtifactCleanup
  workflows.
- Source discovery preparation and lifecycle invalidation publication.
- Immutable Definition admission and Root-local immutable Definition reads.
- Catalog projections separated from complete Artifact reads.
- Verified source entry, tree, artifact, and native-path access.
- Explicit verification sessions.
- Managed package source-first publication outcomes.
- Generic protected overlay, secret binding, cleanup, and recovery behavior.
- Generic generated-package compilation, installation, and preload behavior.
- Generic `store/compose.Store` assembly and local deployment ownership.

## Proposal 2 package A and B

Completed boundary work includes:

- `llmartifactory-go` exists as the LLM-domain attachment over one generic
  `store/compose.Store`.
- Application embedded content is isolated under `artifactbuiltin`.
- Application topology and registration selection are isolated under
  `artifactsetup`.
- Generic local opening remains the only local Store opening path.
- Application-owned protected Root, Source, retention, and generated catalog
  declarations remain outside the reusable LLM domain layer.

## Proposal 2 package C status

Declaration semantics are now owned by shared grammar plus family
interpretations:

- `core/declaration` owns only shared declaration grammar, canonical Entry
  representation, member forms, locators, ordering, and dispatch mechanics.
- `core/declaration/decoder` performs canonical JSON/YAML adaptation and
  complete expected-schema-key dispatch.
- Generic JSON Schema execution remains expected-key-only.
- `core/declaration/interpretation` owns immutable family registration,
  Definition reconstruction, typed reconstruction validation, relationship
  extraction, selector eligibility, and declaration alias eligibility.
- Family contracts own their schemas and semantic validation under
  `<family>/contract/v1`.
- Canonical JSON and YAML decoders declare the exact registered schema keys
  they require.
- Agent Markdown now emits admitted definitions through the supplied family
  interpretation registry rather than a central declaration-definition switch.
- Text and Plugin expose family-owned Definition reconstruction boundaries.
- Tool interpretation is restored as the Tool family interpretation rather
  than the accidental copied Skill implementation.

The remaining Package C stabilization work is caller migration rather than a
second declaration-language implementation:

- Generated catalog preparation receives an explicit immutable interpretation
  registry instead of referring to a package-global `interpretations` value.
- Application registration selection is under `artifactsetup/registration`;
  canonical decoders, source-format decoders, schema codecs, and locator
  factories remain separately selected registrations.

## Proposal 2 package D status

Composition and Plugin ownership are now explicit:

- `core/composition` owns lookup order, alias traversal, selector expansion,
  ambiguity handling, cycle limits, direct Plugin membership resolution, and
  recursive capability-plan projection.
- Capability targets use explicit form and provenance:
  - source-backed `artifact`,
  - application-supplied `direct`.
- `MappedTarget`, fallback-provider behavior, and caller-owned built-in flags
  are removed from the composition contract.
- A direct capability never fabricates Artifact, Source, Definition, local
  state, or lifecycle identity.
- Artifact capability projectors may preserve an Artifact target or produce a
  direct target with provider-owned identity and evidence.
- Explicit refresh remains separate from normal read-only resolution.
- Plugin is the only LLM membership artifact. Plugin membership remains
  declaration content rather than Artifact ownership.
- Plugin profile terminology is now explicit. Agent, Skill, MCP, and Tool
  provide family-owned `plugin.Profile` values instead of a generic
  Collection-domain policy.
- `MappedTarget`, fallback-provider behavior, and caller-owned built-in flags
  are not part of composition. Runtime consumers receive
  `composition.CapabilityTarget`.
- Source-backed runtime consumers validate `TargetFormArtifact` and use the
  referenced Artifact. Application-supplied direct targets remain explicit
  `TargetFormDirect` values and are not fabricated as Artifacts.
- Application topology now owns the Plugin document vocabulary:
  - generic Plugin document use: `plugin`,
  - managed Plugin document use: `managedPlugin`,
  - Agent managed Plugin document use: `agentManagedPlugin`,
  - Tool Plugin document use: `toolPlugin`.
- Built-in topology validates that the built-in Root remains protected and
  that its managed package Source remains enabled and authoritative.
- Generic Plugin document selection no longer accepts `collection.yaml`,
  `collection.yml`, or `collection.json`.

## Current backend migration state

The completed caller migration in this structural stabilization includes:

- Agent managed import receives the explicit declaration interpretation
  registry required for contained-definition admission.
- Generated Agent, Skill, MCP, Tool, and Model catalog preparation receives
  the explicit registry required to reconstruct expected Definitions.
- Generated catalog tests construct application-selected registrations rather
  than importing removed `core/registration`, `providerapi`, or compatibility
  packages.
- Model and Tool aggregate capability inputs use `CapabilityTarget`; obsolete
  mapped-target encoding and decoding owners are removed.
- Direct capability test setup uses `DirectCapabilityProvider` with explicit
  provider identity, provider-local identity, provenance, and evidence.

## Remaining backend migration

- Remaining runtime adapters that still persist, emit, or decode
  `MappedTarget` values must migrate to `CapabilityTarget`.
- Existing Collection-named transport methods in family consumer APIs and
  Wails wrappers remain a Package E/F caller migration item; Plugin remains
  the sole declaration and membership owner.
- Direct Model and Tool capability runtime adapters must be bound through
  `artifactsetup` before application-supplied direct targets are executable.

## Remaining package E work

- Move remaining family behavior out of `consumerapi`, `providerapi`, and
  aggregate-era folders into the actual family owners.
- Complete Text API extraction from Agent-owned materialization entrypoints.
- Complete Skill source-format, package, and materialization consolidation.
- Complete Model and Model Provider separation into named settings,
  credential, preference, package, and capability owners.
- Complete MCP installation and secret ownership consolidation.
- Finish Team, Loop, and Workflow family plan projections.
- Remove remaining family-local Collection terminology from internal names.

## Remaining package F work

- Bind application Go Tool metadata, inference-adapter direct capabilities,
  and Artifact capability projectors through `artifactsetup`.
- Migrate Wails wrappers and aggregate adapters to the new target and Plugin
  contracts.
- Remove wrapper-owned target mapping, fallback construction, and legacy
  Collection projections.
- Move remaining application-selected filenames and embedded package rules out
  of legacy topology helpers and into family package/setup owners.
- Finish runtime adapter binding and startup ordering through
  `artifactsetup`.
