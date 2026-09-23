# MCP Store HLD

Status: Desired-state design. This HLD defines the target MCP Store declaration-management experience. Current implementation behavior, migration work, and delivery gaps are described only after the desired-state sections.

Normative foundations:

- [Portable Artifact Declaration Contracts HLD](./artifact_contracts_hld.md)
- [Artifact Store, Resolution, and Ecosystem HLD](./artifacts_hld.md)

Related user-experience reference:

- [Agent Store HLD](./agent_store_hld.md)

This HLD defines MCP-specific Store policy, MCP Collection behavior, MCP Server and MCP Policy import and export, managed package lifecycle, and the declaration-management UI.

Portable declaration syntax, Source behavior, Definitions, Artifact persistence, relationship resolution, locators, fallback, enablement, resource verification, protected Root behavior, and runtime behavior are inherited from the normative HLDs and are not redefined here.

## Table of contents <!-- omit from toc -->

- [Purpose](#purpose)
- [Scope and boundaries](#scope-and-boundaries)
  - [Goals](#goals)
  - [In scope](#in-scope)
  - [Out of scope](#out-of-scope)
- [Desired user experience](#desired-user-experience)
  - [Browse MCP Collections](#browse-mcp-collections)
  - [Create an MCP Collection shell](#create-an-mcp-collection-shell)
  - [Import MCP declarations](#import-mcp-declarations)
  - [Export and revise a declaration](#export-and-revise-a-declaration)
  - [MCP setup handoff](#mcp-setup-handoff)
- [MCP Store concepts](#mcp-store-concepts)
- [MCP Collections](#mcp-collections)
  - [Representation](#representation)
  - [Baseline Collection](#baseline-collection)
  - [Membership behavior](#membership-behavior)
  - [Collection deletion boundary](#collection-deletion-boundary)
- [Managed package forms](#managed-package-forms)
- [Interchange format and package format](#interchange-format-and-package-format)
- [Import formats and admission profiles](#import-formats-and-admission-profiles)
  - [General format rules](#general-format-rules)
  - [Accepted root declarations](#accepted-root-declarations)
  - [Standalone MCP Server admission profile](#standalone-mcp-server-admission-profile)
  - [Standalone MCP Policy admission profile](#standalone-mcp-policy-admission-profile)
  - [MCP Plugin admission profile](#mcp-plugin-admission-profile)
  - [Relationship preservation](#relationship-preservation)
  - [Secret-placement admission boundary](#secret-placement-admission-boundary)
  - [Required confirmations](#required-confirmations)
- [Import preview and commit](#import-preview-and-commit)
  - [Preview flow](#preview-flow)
  - [Preview diagnostics](#preview-diagnostics)
  - [Conflict rules](#conflict-rules)
  - [Signed preparation](#signed-preparation)
  - [Commit behavior](#commit-behavior)
  - [Publication sequences](#publication-sequences)
- [Reads, export, deletion, and enablement](#reads-export-deletion-and-enablement)
  - [Reads and resolution](#reads-and-resolution)
  - [Export](#export)
  - [Deletion and revision](#deletion-and-revision)
  - [Enablement](#enablement)
- [UI and API boundaries](#ui-and-api-boundaries)
  - [Allowed MCP Store UI operations](#allowed-mcp-store-ui-operations)
  - [Prohibited MCP Store UI operations](#prohibited-mcp-store-ui-operations)
  - [Desired UI-facing service surface](#desired-ui-facing-service-surface)
- [Security and local-state boundaries](#security-and-local-state-boundaries)
- [Built-in MCP content](#built-in-mcp-content)
- [Migration from legacy user-driven MCP authoring](#migration-from-legacy-user-driven-mcp-authoring)
  - [Migration principles](#migration-principles)
  - [Migration phases](#migration-phases)
  - [Legacy data disposition](#legacy-data-disposition)
  - [User migration paths](#user-migration-paths)
    - [Legacy standalone MCP Server](#legacy-standalone-mcp-server)
    - [Legacy MCP Policy](#legacy-mcp-policy)
    - [Legacy MCP Bundle or Collection](#legacy-mcp-bundle-or-collection)
  - [Legacy API retirement](#legacy-api-retirement)
- [Current status, desired status, and gaps](#current-status-desired-status-and-gaps)
  - [Declaration, package, and import status](#declaration-package-and-import-status)
  - [UI, lifecycle, and migration status](#ui-lifecycle-and-migration-status)

## Purpose

The MCP Store is the application lifecycle facade for source-backed:

- `mcp` Server Artifacts.
- `mcp.policy` Policy Artifacts.
- MCP-domain `plugin` Artifacts presented as MCP Collections.

The declaration-management flow is:

```text
JSON or YAML declaration file
  -> MCP import preview
  -> managed Source package publication
  -> Artifact Store reconciliation
  -> shared declaration resolution
  -> MCP Collection catalog, inspection, export, and setup handoff
```

The MCP Store provides:

- MCP Collection catalog and capability inspection.
- Import of one MCP Server, one MCP Policy, or one MCP Plugin declaration package.
- JSON and YAML interchange.
- Canonical declaration export as JSON or YAML.
- Create-only managed package publication.
- File-based declaration revision through export, external editing, deletion, and re-import.
- MCP Collection membership created by import rather than by a member editor.
- Import diagnostics, conflict detection, confirmations, and signed prepared imports.
- MCP setup handoff after a Server is imported or inspected.
- Protected built-in MCP package inspection and export.

The MCP Store does not provide a second MCP declaration database, a field-by-field MCP declaration editor, a generic runtime-control panel, or a replacement execution system.

## Scope and boundaries

### Goals

The design must:

- Reuse ordinary source-backed `mcp`, `mcp.policy`, and `plugin` Artifacts.
- Preserve portable declaration ownership in Source content.
- Support JSON and YAML as import and export interchange formats.
- Make import and export the only user-facing way to author or revise managed MCP declarations.
- Support both:
  - One standalone MCP Server declaration.
  - One MCP-domain Plugin declaration containing MCP Server and Policy relationships.
- Keep MCP Collection membership separate from target ownership.
- Preserve unresolved and ambiguous relationship intent.
- Keep declaration availability separate from installation setup and runtime readiness.
- Keep installation-local values outside portable declarations and exports.
- Permit protected built-in MCP declarations to remain read-only while retaining authorized local state behavior.

### In scope

This HLD defines:

- MCP Collection behavior and MCP-domain Plugin restrictions.
- Import and export of `mcp`, `mcp.policy`, and MCP-domain `plugin` declarations.
- JSON and YAML interchange behavior.
- Managed package naming and publication policy.
- File-import preview, commit, confirmation, conflict, and deletion behavior.
- MCP Collection shell lifecycle.
- Membership creation through standalone Server or Policy import.
- MCP Policy import and export behavior.
- MCP setup handoff boundaries.
- Protected built-in MCP package behavior relevant to catalog and import policy.
- Migration away from user-driven MCP declaration forms.

### Out of scope

This HLD does not define:

- Portable MCP, MCP Policy, Plugin, or member wire syntax.
- Source registration, resource verification, or resolver implementation.
- MCP connection management.
- MCP connection health.
- OAuth browser flow behavior.
- Secret-store implementation.
- MCP discovery snapshots.
- Tool, resource, prompt, completion, or App invocation.
- Approval decisions and execution policy enforcement.
- Runtime connection profiles or selected profile behavior.
- Runtime readiness aggregation.
- Archive, Git, URL, package, or command locator materialization.
- Import of standard multi-server `.mcp.json` configuration files as managed UI input.
- Generic import of arbitrary Artifact types.
- Automatic transfer of secrets or installation-local values to a new Artifact occurrence.
- A visual MCP Server, MCP Policy, or Plugin declaration editor.
- In-app YAML or JSON declaration editing.

## Desired user experience

### Browse MCP Collections

The MCP management page presents MCP Collections and their declared MCP Server and MCP Policy relationships.

An MCP Collection is shown as a collection-oriented view. It is not a new Artifact type.

A user can:

- Browse user-managed and protected MCP Collections.
- Expand a Collection to view currently available direct MCP Server targets.
- Inspect a Collection capability view that includes available, unavailable, and ambiguous direct relationships.
- Inspect available MCP Server and MCP Policy declarations.
- View declaration diagnostics and package provenance.
- Export an available declaration as JSON or YAML.
- Enable or disable a Collection or available target through universal Artifact metadata.
- Open a separate setup handoff for a resolved MCP Server.

The MCP Store page does not display runtime connection controls, runtime health, discovered tools, OAuth progress, or invocation controls.

### Create an MCP Collection shell

A user can create an empty ordinary MCP Collection.

The collection shell exists only to provide an explicit destination for later standalone MCP Server or MCP Policy imports.

The limited Collection lifecycle permits:

- Create with logical name, display name, and description.
- Edit display metadata.
- Enable or disable through universal Artifact metadata.
- Import a standalone MCP Server or Policy into the Collection.
- Delete an empty Collection through the ordinary guarded Collection lifecycle.

The MCP Store does not expose a generic Plugin editor. In particular, it does not expose arbitrary member insertion, removal, attachment, detachment, selector authoring, or raw declaration editing.

### Import MCP declarations

The import page accepts one selected `.json`, `.yaml`, or `.yml` file.

The root declaration determines the import mode:

```text
type: mcp
  -> import one standalone MCP Server into a selected MCP Collection

type: mcp.policy
  -> import one standalone MCP Policy into a selected MCP Collection

type: plugin
  -> import one MCP Plugin package as an MCP Collection in a selected user Root
```

The user flow is:

```text
Select user Root
  -> optionally select an editable MCP Collection
  -> select one JSON or YAML declaration file
  -> preview normalized declaration, projected Artifacts, diagnostics, and conflicts
  -> accept required confirmations
  -> commit import
  -> inspect declaration and perform separate MCP setup if required
```

A selected MCP Collection is required for `mcp` and `mcp.policy` imports.

A selected MCP Collection is invalid for a root `plugin` import. A Plugin import creates its own MCP Collection Artifact from the imported Plugin declaration.

### Export and revise a declaration

Managed declaration revision is file-based.

A user changes an independently managed MCP Server or Policy by:

```text
Export declaration
  -> edit JSON or YAML outside the application
  -> delete the managed package
  -> import the revised declaration
```

A user changes an imported MCP Plugin package by:

```text
Export Plugin declaration
  -> edit JSON or YAML outside the application
  -> import under a new Plugin name

or

Export Plugin declaration
  -> remove the imported Plugin package through the package lifecycle
  -> import again with the same Plugin name
```

The package lifecycle is explicit because Plugin package removal is source mutation. It is not inferred target ownership or a generic composition cascade.

### MCP setup handoff

Import preview and declaration inspection can identify MCP Servers that declare installation inputs.

The MCP Store may provide a setup handoff for a resolved Server Artifact.

The handoff can direct a user to configure installation-local values through MCP setup APIs, but it must not:

- Modify MCP declaration JSON or YAML.
- Write secret values into declaration files.
- Change a declaration Definition digest.
- Make import contingent on connection or runtime readiness.
- Require a connection attempt during preview or commit.

## MCP Store concepts

| Concept                   | Representation                                                           | MCP Store meaning                                                                              |
| ------------------------- | ------------------------------------------------------------------------ | ---------------------------------------------------------------------------------------------- |
| MCP Server                | `mcp` Artifact                                                           | A source-backed portable MCP Server declaration.                                               |
| MCP Policy                | `mcp.policy` Artifact                                                    | A source-backed portable MCP Policy declaration.                                               |
| MCP Collection            | `plugin` Artifact                                                        | An MCP-domain view over a Plugin whose direct members are MCP Servers or MCP Policies.         |
| MCP Plugin package        | Managed package containing one root `plugin` declaration                 | A file-imported MCP Collection declaration. It may contain MCP Server and Policy declarations. |
| Collection shell          | Managed package containing one initially empty root `plugin` declaration | An application-created destination for standalone imports.                                     |
| Standalone Server package | Managed package containing one root `mcp` declaration                    | A separately imported MCP Server.                                                              |
| Standalone Policy package | Managed package containing one root `mcp.policy` declaration             | A separately imported MCP Policy.                                                              |
| Setup descriptor          | Derived inspection output                                                | A non-secret description of MCP installation inputs requiring local configuration.             |

The desired vocabulary uses `MCP Collection`.

`MCP Bundle` is a legacy UI term. It must not become a new portable Artifact type, a second persistence model, or a separate ownership construct.

The central representation is:

```text
MCP Collection
  -> Plugin Artifact
  -> direct mcp and mcp.policy relationships
```

The Collection, Servers, and Policies remain ordinary Artifacts with independent declaration occurrences.

## MCP Collections

### Representation

An MCP Collection is an MCP Store view over a `plugin` Artifact.

The managed MCP Collection domain restricts direct Plugin members to:

```text
mcp
mcp.policy
```

The managed Collection domain does not permit:

- `text`, `model`, `tool`, `skill`, `agent`, `team`, `loop`, `workflow`, or `workspace` members.
- Nested Plugin members.
- Member selectors.
- Relationship `overrides`.
- Relationship `use`.

A permitted direct MCP Collection member may be:

- A named external relationship.
- A located external relationship.
- A contained `mcp` declaration.
- A contained `mcp.policy` declaration.

A direct relationship remains a relationship. It does not copy, own, enable, configure, install, or delete its target.

### Baseline Collection

Each supported user Root has one application-provisioned baseline MCP Collection.

The baseline Collection:

- Is an explicit destination for standalone MCP Server and Policy import.
- Has an application-owned logical identity and package location.
- Is non-renamable.
- Is non-deletable through ordinary Collection APIs.
- Is not automatically selected by a Workspace.
- Is not automatically active.
- Is not an implicit import destination.
- Uses ordinary Artifact enablement behavior.

Baseline provisioning remains an application Root lifecycle responsibility.

### Membership behavior

Standalone Server and Policy import creates or preserves a located external relationship in the selected Collection.

```text
Standalone Server or Policy import
  -> ensure one stable located relationship in Collection Plugin
  -> publish independent target package
  -> reconcile managed Source
```

The relationship locator must be deterministic for the managed package identity. Importing a replacement after deletion restores the same relationship even when the newly selected interchange file uses another supported format.

Collection membership behavior is:

| Event                             | Result                                                                                        |
| --------------------------------- | --------------------------------------------------------------------------------------------- |
| Standalone MCP Server import      | Creates or preserves one direct located `mcp` relationship.                                   |
| Standalone MCP Policy import      | Creates or preserves one direct located `mcp.policy` relationship.                            |
| Server or Policy package deletion | Relationship remains declared and becomes unavailable.                                        |
| Exact package restoration         | Matching dangling relationship becomes available again.                                       |
| Collection shell deletion         | Independent Server and Policy packages remain available.                                      |
| Imported Plugin package deletion  | Removes only declarations sourced by that package and preserves independent external targets. |

The MCP Store does not expose a standalone attach, detach, or arbitrary membership-editor UI.

### Collection deletion boundary

The ordinary Collection deletion guard remains authoritative for Collection shells.

It includes all direct declared relationships, including:

- Available relationships.
- Unavailable relationships.
- Ambiguous relationships.
- Located relationships to deleted standalone packages.

An imported MCP Plugin package has a separate package-removal lifecycle operation. That operation removes source content within the selected managed package. It is not a generic Collection deletion operation and does not delete independent external target packages.

## Managed package forms

Managed packages remain Source-backed packages. Package kinds are storage conventions and are not portable declaration versions.

| Package purpose                      | Managed package kind | Root declaration | Managed document                       |
| ------------------------------------ | -------------------- | ---------------- | -------------------------------------- |
| Standalone Server import             | `mcp`                | `mcp`            | `mcp.json` or `mcp.yaml`               |
| Standalone Policy import             | `mcp-policy`         | `mcp.policy`     | `mcp-policy.json` or `mcp-policy.yaml` |
| Application-created Collection shell | `mcp-collection`     | `plugin`         | `plugin.json` or `plugin.yaml`         |
| Imported MCP Plugin                  | `mcp-plugin`         | `plugin`         | `plugin.json` or `plugin.yaml`         |

`unversioned` may remain part of the managed package address convention. It is not a portable release field and must not be emitted into declarations.

## Interchange format and package format

JSON and YAML are supported interchange formats.

The import format is determined from the selected file extension:

| Extension | Interchange format | Published package document                                 |
| --------- | ------------------ | ---------------------------------------------------------- |
| `.json`   | JSON               | Canonical JSON document                                    |
| `.yaml`   | YAML               | Canonical YAML document                                    |
| `.yml`    | YAML               | Canonical YAML document using the `.yaml` managed filename |

The importer canonicalizes both formats to one normalized semantic declaration before validation and digesting.

The selected input file's:

- Absolute path.
- Original filename.
- Comments.
- Whitespace.
- YAML anchors.
- YAML aliases.
- Source key ordering.

are not preserved as managed package state.

For a fresh package, the import format determines the managed package document format.

For restoration of a deleted standalone Server or Policy package, an existing exact dangling Collection relationship takes precedence. The importer publishes the prior managed document format and locator so that it restores the existing relationship instead of creating a second relationship.

## Import formats and admission profiles

### General format rules

Import accepts exactly one object declaration document.

The importer must:

- Accept JSON from `.json`.
- Accept YAML from `.yaml` and `.yml`.
- Reject unsupported extensions.
- Reject multiple documents.
- Reject duplicate mapping keys.
- Reject malformed objects.
- Canonicalize through the portable declaration contract path.
- Validate the original normalized declaration against the portable contract before applying managed restrictions.
- Use canonical semantic JSON as the Definition digest input.
- Render canonical JSON or canonical YAML only after successful validation.

The importer must not infer declaration type from filename conventions. Root `type` determines the import mode.

### Accepted root declarations

| Root type      | Import mode              | Destination             | Result                                                           |
| -------------- | ------------------------ | ----------------------- | ---------------------------------------------------------------- |
| `mcp`          | Standalone Server import | Editable MCP Collection | Independent Server package plus located Collection relationship. |
| `mcp.policy`   | Standalone Policy import | Editable MCP Collection | Independent Policy package plus located Collection relationship. |
| `plugin`       | MCP Plugin import        | Mutable user Root       | One imported MCP Plugin package and one MCP Collection Artifact. |
| Any other type | Rejected                 | None                    | The MCP Store is not a generic Artifact importer.                |

A standard `.mcp.json` or `mcp.json` file using the multi-server `mcpServers` source format is not a managed import input.

It remains a repository Source adapter concern. A future bulk-import workflow may explicitly transform it into a Plugin package, but that is not part of this HLD.

### Standalone MCP Server admission profile

A standalone `mcp` import must be one concrete MCP Server declaration.

The managed Server profile requires:

- Root `type: mcp`.
- Valid portable name.
- No declaration `locator`.
- No `server` source selector.
- Concrete `stdio` or `streamableHTTP` transport.
- A valid stdio command for `stdio`.
- A valid URL for `streamableHTTP`.
- Portable installation input definitions only.
- No installation values, secret references, secret values, OAuth tokens, selected profile, connection state, or runtime state.

The declaration may contain a direct portable MCP Policy reference.

That policy reference remains subject to ordinary portable resolution. The importer must not invent `scope`, Artifact references, revision snapshots, or hidden policy bindings for it.

### Standalone MCP Policy admission profile

A standalone `mcp.policy` import must be one concrete MCP Policy declaration.

The managed Policy profile requires:

- Root `type: mcp.policy`.
- Valid portable name.
- No declaration `locator`.
- No local runtime state.
- No secret values.
- No Artifact IDs, Root IDs, or installation-local references.

Standalone Policy import exists so file-based MCP Policy authoring remains possible after removal of manual policy forms.

### MCP Plugin admission profile

An imported MCP Plugin must be a concrete root `plugin` declaration.

The managed MCP Plugin profile requires:

- Root `type: plugin`.
- Valid portable name.
- No root declaration `locator`.
- Direct member types limited to `mcp` and `mcp.policy`.
- No member selectors.
- No nested Plugin members.
- No relationship `overrides` or `use`.
- No contained declaration types other than `mcp` and `mcp.policy`.

A direct MCP Collection relationship may remain named or located. It remains an external relationship and does not cause the importer to copy or materialize another declaration file.

A contained MCP Server or Policy declaration must be concrete:

- A contained `mcp` cannot use a declaration `locator` or MCP `server` selector.
- A contained `mcp.policy` cannot use a declaration `locator`.
- A contained MCP Server may reference a Policy by portable name.
- A contained MCP Policy may be used by one or more contained Servers in the same Plugin package.

The preview recognizes a Policy contained in the same imported Plugin package as a planned package-local target. It must not report that Policy as unavailable solely because the package has not yet been published.

### Relationship preservation

Unlike managed Agent import, MCP import does not normalize named Plugin members to `scope: builtin`.

Reasons:

- An MCP Collection may intentionally organize independently managed Servers and Policies in the current Root.
- Located MCP relationships are meaningful Collection membership declarations.
- The direct MCP `policy` field has no portable `scope`.
- Import must not silently change a Plugin's portable relationship meaning.

The importer preserves valid relationship fields exactly after canonicalization.

Unavailable and ambiguous external Server or Policy dependencies are observations, not import blockers, unless they create an identity or package conflict.

### Secret-placement admission boundary

Portable MCP declarations must not contain installation values or secret values.

The importer relies on the MCP declaration contract to require that declared secret inputs are represented as installation input definitions and permitted placeholder targets.

The importer must:

- Reject invalid secret-input placement defined by the MCP contract.
- Reject secret defaults where the portable contract forbids them.
- Reject invalid URL user information through normal MCP URL validation.
- Exclude all local secret references and values from normalized declaration output.
- Return setup descriptors without returning secret values.

The importer is not a general-purpose secret scanner.

It must not claim comprehensive detection of credentials embedded in arbitrary command arguments, URLs, query parameters, or ordinary string values.

### Required confirmations

Preview requires explicit confirmation for declarations that need deliberate review.

Initial required confirmation classes are:

| Condition                                            | Confirmation                                                                         |
| ---------------------------------------------------- | ------------------------------------------------------------------------------------ |
| Any imported `stdio` MCP Server                      | The declaration can later start a local command when selected by a runtime consumer. |
| A Policy that enables automatic execution by default | The imported declaration changes default policy posture.                             |
| A Policy that allows app-initiated tool calls        | The imported declaration changes application interaction posture.                    |

Confirmation accepts declaration risk. It does not execute, connect, install, enable, or otherwise activate a Server.

## Import preview and commit

### Preview flow

```text
Read selected file
  -> detect JSON or YAML format
  -> canonicalize and validate portable declaration
  -> apply MCP managed admission profile
  -> determine import mode and destination
  -> derive managed package and document locator
  -> inspect identity, package, and Collection conflicts
  -> inspect Server and Policy relationship status
  -> calculate projected Artifact digests
  -> return preview and signed prepared import
```

Preview returns:

- Root declaration type and import mode.
- Normalized JSON or YAML rendering for review.
- Input digest and root Definition digest.
- Projected Artifacts and their Definition digests.
- Planned package address and managed document locator.
- Selected Collection information for standalone imports.
- Existing dangling-membership restoration information where applicable.
- Relationship observations and diagnostics.
- MCP setup descriptors.
- Blocking issues and conflicts.
- Required confirmation codes.
- Expiration time.
- A signed prepared payload when import can proceed.

### Preview diagnostics

| Condition                                          | Result                                    |
| -------------------------------------------------- | ----------------------------------------- |
| Invalid JSON or YAML                               | Import-blocking error.                    |
| Invalid portable declaration                       | Import-blocking error.                    |
| Unsupported root type                              | Import-blocking error.                    |
| Invalid managed MCP profile                        | Import-blocking error.                    |
| Invalid destination Collection                     | Import-blocking error.                    |
| Protected Root destination                         | Import-blocking error.                    |
| Identity, package, or membership conflict          | Import-blocking error.                    |
| Invalid contained MCP Server or Policy             | Import-blocking error.                    |
| Unavailable external Server or Policy relationship | Warning.                                  |
| Ambiguous external Server or Policy relationship   | Warning.                                  |
| Package-local contained Policy reference           | Informational planned-target observation. |
| Stdio Server or policy-risk confirmation           | Confirmation required.                    |

The importer must not partially salvage an invalid document.

It must not:

- Drop invalid members.
- Publish valid subsets of a Plugin.
- Convert unsupported members into another type.
- Rewrite unavailable relationships into a different target.
- Remove a malformed Policy reference.
- Replace an existing package.

### Conflict rules

Managed import is create-only.

The importer blocks when:

- A Server, Policy, or Plugin Artifact with the requested Root-scoped identity already exists.
- A conflicting missing or unpurged Artifact occurrence reserves the identity.
- A protected built-in Artifact reserves the requested identity.
- The target package address is occupied by non-equivalent content.
- A selected Collection already has a conflicting direct relationship for the imported standalone target.
- A contained declaration conflicts with an existing Root Artifact.
- A previous dangling relationship requires one managed document locator but the requested import would publish another incompatible locator.
- Import would require replacement, merge, rename, or upsert behavior.

Equivalent current package content may be recognized as an idempotent no-op only when the expected package, Artifact identity, Collection relationship, and Source state are equivalent.

### Signed preparation

The MCP importer uses the same client-carried signed preparation pattern as managed Agent import.

The prepared payload binds:

- Normalized declaration content.
- Input format and publish format.
- Input and Definition digests.
- Projected declaration Artifacts.
- Selected Root.
- Selected Collection when applicable.
- Expected Collection revision when applicable.
- Expected managed Source generation.
- Managed package identity and document locator.
- Required confirmation codes.
- Expiration time.

The payload does not contain:

- Original file path.
- File system handles.
- Secret values.
- Secret references.
- Installation values.
- OAuth tokens.
- Selected profiles.
- Runtime state.
- Runtime readiness.
- Dependency witnesses.
- Dependency revision snapshots.
- Backend-held import-session state.

The signer is process-local. Restart invalidates uncommitted prepared imports.

### Commit behavior

Commit verifies:

- Prepared payload authenticity and fingerprint.
- Payload expiration.
- Required confirmations.
- Destination Root and Collection authorization.
- Expected Collection revision when applicable.
- Expected managed Source generation.
- Current package, identity, and membership conflicts.
- Current normalized declaration validity.

Commit does not:

- Reread the selected file.
- Accept replacement declaration content from the client.
- Require external dependencies to remain available.
- Collect installation values.
- Resolve secrets.
- Attempt connection or discovery.
- require runtime readiness.

### Publication sequences

Standalone Server and Policy imports use two Source-side operations:

```text
Ensure Collection relationship
  -> publish independent target package
  -> reconcile managed Source
```

The operation is intentionally not atomic.

If Collection membership succeeds and target package publication fails:

```text
Collection relationship remains declared
  -> relationship is unavailable
  -> dangling relationship records import intent
  -> user can correct input and preview again
```

Plugin import publishes one Plugin package:

```text
Publish Plugin package
  -> reconcile managed Source
  -> root Plugin and contained Artifacts become available
```

A Plugin package may contain external relationships. Publication does not materialize external target packages.

## Reads, export, deletion, and enablement

### Reads and resolution

MCP Store reads use the shared typed resolver and preserve:

- Declared relationship occurrences.
- Available, unavailable, and ambiguous status.
- Artifact-backed target provenance.
- Mapped target provenance where supported by shared resolver behavior.
- Relationship diagnostics.
- Contained declaration identity.
- Source-selected alias behavior where portable contracts allow it.

The Collection catalog is availability-oriented.

A Collection capability inspection is diagnostic-oriented and includes unavailable and ambiguous direct relationships.

Resolution availability does not establish installation completion, connection capability, runtime readiness, or execution permission.

### Export

Available declarations can be exported in either canonical JSON or canonical YAML.

| Export target  | Exported root declaration |
| -------------- | ------------------------- |
| MCP Server     | `mcp`                     |
| MCP Policy     | `mcp.policy`              |
| MCP Collection | `plugin`                  |

Export preserves declaration shape.

In particular:

- Contained MCP Servers and Policies remain contained.
- Named and located external relationships remain external.
- A direct MCP Policy reference remains a direct Policy reference.
- No target is inlined merely because it resolves.
- No unavailable relationship is silently removed.
- Existing source-selected declaration shape is retained.

Export excludes all local and runtime state, including:

- Artifact, Root, and Source IDs.
- Artifact enablement.
- Collection membership outside the exported Plugin declaration.
- Package addresses.
- Installation values.
- Secret references.
- Secret values.
- OAuth tokens.
- Selected connection profiles.
- Additional local Policies.
- Connection state.
- Discovery data.
- Runtime readiness.

Export of a Plugin is declaration export, not an archive export.

A Plugin that references external standalone Server or Policy packages remains non-self-contained after export. To recreate that complete arrangement, a user exports and imports each independent declaration package separately.

### Deletion and revision

| Target                           | Allowed user action           | Result                                                                                   |
| -------------------------------- | ----------------------------- | ---------------------------------------------------------------------------------------- |
| Independently managed MCP Server | Delete managed Server package | Removes only the Server package. Direct Collection relationships remain unavailable.     |
| Independently managed MCP Policy | Delete managed Policy package | Removes only the Policy package. Direct Collection relationships remain unavailable.     |
| Contained Server or Policy       | No individual deletion        | Export and revise or remove the owning Plugin package.                                   |
| Empty Collection shell           | Guarded Collection deletion   | Removes only the Collection shell package.                                               |
| Imported MCP Plugin package      | Explicit package removal      | Removes declarations sourced by that package and preserves independent external targets. |
| Protected built-in declaration   | No user-managed deletion      | Protected package policy applies.                                                        |

Managed revision does not support:

- Patch.
- Merge.
- Rename.
- Replace.
- Upsert.
- Field-level declaration updates.
- In-app JSON or YAML editing.

### Enablement

MCP Servers, Policies, and Collections use universal `Artifact.Enabled`.

Enablement:

- Does not alter declaration content.
- Does not alter Definition digest.
- Does not alter relationship resolution.
- Does not alter package identity.
- Does not alter import or export.
- Does not transfer declaration ownership.
- May be used by a caller for enabled-only catalog filtering.

Enablement is not a substitute for declaration import, setup, or runtime behavior.

## UI and API boundaries

### Allowed MCP Store UI operations

| Area                      | Allowed operations                                                                                  |
| ------------------------- | --------------------------------------------------------------------------------------------------- |
| MCP Collections           | Browse, inspect, create shell, edit display metadata, enable, disable, guarded delete.              |
| Import                    | Select JSON or YAML file, choose destination where required, preview, confirm, commit.              |
| Export                    | Export available Server, Policy, or Plugin declaration as JSON or YAML.                             |
| Managed package lifecycle | Delete independently managed package or imported Plugin package through explicit lifecycle actions. |
| Setup handoff             | Open a separate setup flow for a resolved MCP Server.                                               |

### Prohibited MCP Store UI operations

The MCP Store UI must not expose:

- Add Server form.
- Edit Server form.
- Edit Policy form.
- Policy upsert form.
- Raw JSON editor.
- Raw YAML editor.
- Manual field patching.
- Manual transport editor.
- Manual header editor.
- Manual environment editor.
- Manual Policy attachment editor.
- Generic member attach or detach controls.
- Generic Plugin member editor.
- Replace or upsert controls.
- Connection, discovery, OAuth, invocation, approval, or runtime controls.

### Desired UI-facing service surface

| Capability                      | Desired service behavior                                                                   |
| ------------------------------- | ------------------------------------------------------------------------------------------ |
| List Collections                | Returns MCP Collection views and declaration-level status.                                 |
| Inspect Collection              | Returns a capability view for all direct relationships.                                    |
| Create Collection shell         | Creates an empty MCP-domain Plugin package.                                                |
| Update Collection metadata      | Updates limited display metadata only.                                                     |
| Preview MCP import              | Validates JSON or YAML, computes package plan, returns diagnostics and signed preparation. |
| Commit MCP import               | Publishes only the signed prepared declaration plan.                                       |
| Export MCP declaration          | Returns canonical JSON or YAML for one available declaration.                              |
| Delete managed Server or Policy | Removes one independent package with optimistic concurrency.                               |
| Delete imported Plugin package  | Removes one imported Plugin package with explicit package authorization.                   |
| Set enabled                     | Uses universal Artifact enablement metadata.                                               |
| Setup handoff                   | Returns no-secret setup descriptors or delegates to separate setup APIs.                   |

The frontend must not call low-level declaration creation, replacement, policy-upsert, or membership mutation APIs directly.

Collection membership APIs may remain available to the importer and authorized maintenance code, but they are not user-facing MCP Store authoring APIs.

## Security and local-state boundaries

Portable MCP declaration files must never contain:

- Secret values.
- Secret references.
- OAuth access tokens.
- OAuth refresh tokens.
- Installation values.
- Selected connection profile.
- Local additional Policies.
- Connection state.
- Runtime state.
- Artifact IDs.
- Root IDs.
- Source IDs.
- Package addresses.

The MCP Store must:

- Avoid logging imported declaration bodies and secret-like setup values.
- Return setup descriptors without secret values.
- Keep file paths transient.
- Require explicit confirmation for declaration risk classes.
- Preserve package and declaration provenance for audit and conflict reporting.
- Keep secret storage and local installation mutation outside import and export.

A new Artifact occurrence does not automatically receive local setup state from another Artifact occurrence.

This includes a Server recreated after export, deletion, and re-import. Any installation-local value migration requires a separate explicit migration mechanism and must never copy raw secret values through declaration import.

## Built-in MCP content

Protected built-in MCP packages continue to use ordinary portable declarations and protected Root behavior inherited from the Artifact ecosystem.

The MCP Store must:

- Allow read-only inspection and export of available protected declarations.
- Allow authorized local enablement behavior through universal Artifact metadata.
- Keep protected declaration content outside user-managed import mutation.
- Reject import into the protected Root.
- Reject attempts to reserve protected Server, Policy, or Plugin identities in a user Root where package policy reserves those identities.
- Keep local setup state outside protected declaration content.

Built-in package hydration, package verification, stale-package cleanup, and protected overlay behavior remain governed by the Artifact ecosystem and MCP local-state design.

## Migration from legacy user-driven MCP authoring

The existing MCP implementation exposes user-driven creation and editing of small MCP declaration objects.

This includes:

- Frontend Add Server and Edit Server forms.
- Form-driven transport, environment, header, policy, and installation declaration editing.
- Managed Server create and replace operations.
- Managed MCP Policy upsert operations.
- Direct Collection membership mutation APIs.
- MCP Bundle terminology in the current frontend.
- Runtime controls colocated with declaration management.

The target MCP Store replaces declaration authoring with import and export. Existing declarations remain readable during migration.

### Migration principles

- Do not delete existing user-managed MCP packages automatically.
- Do not copy raw secrets into imported declarations.
- Do not change existing runtime behavior as part of this HLD.
- Preserve local state only while an existing Artifact occurrence remains the same.
- Do not silently convert a multi-server `.mcp.json` configuration into a managed Plugin package.
- Do not use package replacement as a hidden import implementation.
- Keep migration reversible until legacy UI and APIs are formally retired.

### Migration phases

| Phase                      | Change                                                                                                                        | Result                                                            |
| -------------------------- | ----------------------------------------------------------------------------------------------------------------------------- | ----------------------------------------------------------------- |
| 1. Import foundation       | Add MCP JSON and YAML import parser, managed profiles, preview, signed preparation, package publisher, and exporter.          | New flow can coexist with legacy packages.                        |
| 2. Package format support  | Add managed YAML document recognition for `mcp`, `mcp.policy`, and Plugin package documents.                                  | JSON and YAML imports can produce source-backed managed packages. |
| 3. Read compatibility      | Treat legacy managed Servers, Policies, and Collections as readable and exportable.                                           | Existing users can inspect and export before changing anything.   |
| 4. UI cutover              | Replace Add/Edit Server and Policy authoring controls with Import and Export actions.                                         | New declaration changes become file-based.                        |
| 5. API restriction         | Remove frontend access to create, replace, upsert, attach, detach, and member-patching APIs.                                  | Low-level APIs become importer-only, migration-only, or internal. |
| 6. Legacy package adoption | Permit explicit adoption only for byte-equivalent or semantically equivalent package content under a reviewed migration path. | Existing compatible packages can retain identity where safe.      |
| 7. Retirement              | Remove legacy form drafts, manual declaration mutation endpoints, and obsolete Bundle terminology.                            | MCP Store reaches the desired import/export-only declaration UX.  |

### Legacy data disposition

| Existing state                               | Migration treatment                                                                                                      |
| -------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------ |
| Existing managed MCP Server package          | Remains readable and exportable. It may remain in place until explicitly migrated or replaced through the file workflow. |
| Existing managed MCP Policy package          | Remains readable and exportable. New Policy authoring uses Policy import or Plugin import.                               |
| Existing MCP Collection Plugin               | Remains readable. Its direct relationships remain declaration content and are not converted into membership rows.        |
| Existing direct Collection membership        | Remains valid. It is no longer edited from the MCP Store UI.                                                             |
| Existing Artifact enablement                 | Remains Artifact-local metadata. It is not exported.                                                                     |
| Existing local installation data             | Remains associated with the existing Artifact occurrence only.                                                           |
| Existing secret references and secret values | Remain outside export. They are not copied during re-import.                                                             |
| Existing protected built-in package          | Remains under protected package hydration. No user migration is performed.                                               |
| Existing runtime session or discovery state  | Not handled by this migration HLD.                                                                                       |

### User migration paths

#### Legacy standalone MCP Server

```text
Export current MCP Server declaration
  -> review or edit JSON or YAML outside the application
  -> select an MCP Collection
  -> import the declaration
  -> configure local setup for the new Artifact occurrence if required
```

If the old Server package remains present with the same Root-scoped identity, import blocks. The user must either:

- Keep the legacy package unchanged.
- Use a new logical name.
- Delete the old managed package after exporting it.
- Use an explicit reviewed package-adoption or migration path.

#### Legacy MCP Policy

```text
Export current MCP Policy declaration
  -> edit JSON or YAML outside the application
  -> import into an MCP Collection

or

Export Policy declaration
  -> place it as a contained mcp.policy declaration in a Plugin file
  -> import the Plugin package
```

#### Legacy MCP Bundle or Collection

```text
Export the Collection Plugin declaration
  -> export any independent Server and Policy packages it references
  -> edit declaration files outside the application
  -> import a new MCP Plugin package or rebuild through standalone imports
```

A Plugin export is not an archive. Independently managed external targets must be exported separately.

### Legacy API retirement

The following APIs must no longer be reachable from the MCP Store frontend after cutover:

- Create managed MCP Server.
- Replace managed MCP Server.
- Upsert managed MCP Policy.
- Direct Collection member add.
- Direct Collection member attach.
- Direct Collection member remove.
- Arbitrary declaration patch.
- Arbitrary Plugin update beyond limited Collection display metadata.

These APIs may temporarily remain available to:

- The importer.
- Explicit package migration tooling.
- Protected package hydration.
- Authorized repair and administrative workflows.

They must not remain hidden frontend backdoors for manual declaration editing.

## Current status, desired status, and gaps

Status terminology:

- `Available` means an implementation path exists in the attached code.
- `Legacy` means the behavior exists but conflicts with this desired HLD.
- `Partial` means foundational support exists but the desired MCP Store flow is incomplete.
- `Pending` means required design or implementation work remains.
- `Excluded` means intentionally outside this HLD.

### Declaration, package, and import status

| Capability                                           | Current status                | Desired status                        | Gap or required change                                                                    |
| ---------------------------------------------------- | ----------------------------- | ------------------------------------- | ----------------------------------------------------------------------------------------- |
| Portable `mcp`, `mcp.policy`, and `plugin` contracts | Available                     | Retained                              | No new portable MCP type is needed.                                                       |
| Canonical JSON declaration decoding                  | Available                     | Retained                              | Reuse for MCP import and export.                                                          |
| Canonical YAML declaration decoding                  | Available                     | Retained                              | Wire managed MCP package discovery and publication to YAML forms.                         |
| Standard `.mcp.json` repository decoding             | Available                     | Retained as repository ingestion only | Do not expose it as managed import until an explicit bulk-import design exists.           |
| Managed MCP Server package publication               | Available                     | Retained behind importer only         | Remove direct user-facing create and replace paths.                                       |
| Managed MCP Policy package publication               | Available                     | Retained behind importer only         | Replace upsert semantics with create-only Policy import.                                  |
| MCP Collection package publication                   | Available                     | Retained                              | Add imported Plugin package form and package provenance.                                  |
| `mcp-plugin` package kind                            | Not supported                 | Required                              | Add package address, publisher, remover, discovery, and migration support.                |
| Standalone MCP Server JSON import preview and commit | Not supported                 | Required                              | Implement managed profile, preview, signed preparation, conflict checks, and publication. |
| Standalone MCP Server YAML import preview and commit | Not supported                 | Required                              | Add YAML format routing and deterministic managed filename behavior.                      |
| Standalone MCP Policy import                         | Not supported                 | Required                              | Implement file-based Policy authoring to replace manual upsert.                           |
| MCP Plugin JSON import                               | Not supported                 | Required                              | Implement Plugin profile, contained declaration projection, and package publication.      |
| MCP Plugin YAML import                               | Not supported                 | Required                              | Reuse canonical YAML decoder and Plugin package writer.                                   |
| MCP declaration export as JSON                       | Not supported as MCP Store UI | Required                              | Add canonical declaration exporter.                                                       |
| MCP declaration export as YAML                       | Not supported as MCP Store UI | Required                              | Add canonical YAML renderer and format selection.                                         |
| Relationship diagnostics during import               | Partial                       | Required                              | Reuse shared resolver inspection and add package-local Policy planning.                   |
| Signed prepared import infrastructure                | Available generically         | Required for MCP importer             | Bind MCP-specific package and Collection state to prepared payloads.                      |
| Create-only import behavior                          | Partial                       | Required                              | Existing replacement and upsert paths must not be used by normal import.                  |

### UI, lifecycle, and migration status

| Capability                                             | Current status          | Desired status                                | Gap or required change                                                                 |
| ------------------------------------------------------ | ----------------------- | --------------------------------------------- | -------------------------------------------------------------------------------------- |
| MCP Bundle catalog UI                                  | Available               | Rename and evolve into MCP Collection catalog | Remove Bundle-only conceptual model.                                                   |
| Add MCP Server form                                    | Legacy                  | Removed                                       | Replace with standalone Server import.                                                 |
| Edit MCP Server form                                   | Legacy                  | Removed                                       | Use export, external edit, delete, and re-import.                                      |
| Form-driven Policy editing                             | Legacy backend behavior | Removed                                       | Use Policy import or Plugin import.                                                    |
| Manual transport and header editing                    | Legacy                  | Removed                                       | Declaration content is file-authored only.                                             |
| Manual Collection membership APIs                      | Available               | Internal only                                 | Importer owns membership creation and preservation.                                    |
| MCP Collection shell creation                          | Available               | Retained                                      | Limit UI to Collection lifecycle metadata and import destination selection.            |
| Collection capability inspection                       | Available               | Retained                                      | Present unavailable and ambiguous relationships clearly.                               |
| Collection deletion guard                              | Available               | Retained for Collection shells                | Add separate imported Plugin package-removal lifecycle.                                |
| Independent MCP Server deletion                        | Available               | Retained                                      | Preserve dangling Collection intent.                                                   |
| Contained MCP Server deletion                          | Not supported           | Intentionally not supported                   | Revise owning Plugin through file workflow.                                            |
| Artifact enablement                                    | Available               | Retained                                      | Keep separate from declaration editing.                                                |
| MCP setup and secret APIs                              | Available               | Retained as separate concern                  | Ensure import/export never carries local values or secret references.                  |
| Built-in MCP package hydration                         | Available               | Retained                                      | Do not route protected content through user import.                                    |
| Runtime controls on MCP Store page                     | Available in current UI | Removed from MCP Store scope                  | Move to a separate runtime-oriented surface or service.                                |
| Import, migration, and package lifecycle test coverage | Pending                 | Required                                      | Add unit, integration, concurrency, package recovery, format, and end-to-end coverage. |
