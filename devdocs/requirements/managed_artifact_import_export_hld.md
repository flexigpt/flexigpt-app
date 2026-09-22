# Managed Artifact Import and Export Extension HLD

Status: Proposed breaking replacement for managed Agent composition authoring and Assistant Preset management

Normative foundations:

- [Portable Artifact Declaration Contracts HLD](./artifact_contracts_hld.md)
- [Artifact Store, Resolution, and Ecosystem HLD](./artifacts_hld.md)
- [Artifact-backed Agent Store HLD](./agent_store_hld.md)

This HLD defines strict-profile import and portable export for user-managed Artifacts. Agent is the first supported domain.

This version intentionally omits detailed screen layout and presentation requirements. Backend behavior, validation, transaction, persistence, lifecycle, and API requirements remain normative.

## Purpose and scope

Managed Agents are imported as complete portable YAML declarations rather than assembled through field-by-field management APIs.

```text
Selected Agent YAML file
  -> safe bounded file read
  -> portable and strict-profile validation
  -> projected Artifact resolution and conflict analysis
  -> immutable prepared import
  -> explicit confirmation
  -> atomic managed Source commit
  -> Agent Artifact and Collection membership
```

Export recovers the current managed declaration:

```text
Managed Agent Artifact
  -> current immutable Definition
  -> portable and strict-profile validation
  -> normalized portable YAML
  -> download response
```

For managed Agent authoring, this HLD supersedes:

- Structured `ManagedAgentDocument` and `ManagedAgentMember` authoring.
- Managed Agent creation through member-by-member requests.
- Managed Agent replacement.
- In-place managed Agent declaration editing.
- Any managed Agent composition editor.

The parent HLDs remain authoritative for:

- Portable declaration syntax and schema validation.
- Root, Source, Definition, Artifact, and resource behavior.
- Typed resolver behavior, fallback, alias traversal, locators, selectors, and diagnostics.
- Artifact enablement.
- Agent Collections as Agent-only Plugin Artifacts.
- Protected built-in content.
- Composition non-ownership.
- Runtime execution boundaries.

This HLD does not define:

- A visual Agent composition editor.
- An in-application YAML editor.
- Managed Agent patch, rename, replace, upsert, or version behavior.
- Linked-file synchronization or import-path watching.
- Repository Source registration from the selected import path.
- URL, Git, package, archive, or command import sources.
- MCP connection execution during import.
- Agent execution.
- Assistant Preset compatibility, migration, dual-read, or dual-write behavior.
- A generic public import API that allows callers to select arbitrary Artifact types or bypass domain policy.

No portable schema version, declaration version, package version, or Agent release version is introduced.

## Architecture and ownership

```text
AgentStoreWrapper
  -> Agent import and export API
    -> Agent domain policy
      -> common managed transfer service
        -> safe file reader
        -> strict-profile registry
        -> projected Artifact overlay
        -> prepared import registry
        -> managed Source transaction
      -> Artifact Store and shared typed resolvers
      -> MCP installation and readiness services
```

| Component                  | Responsibility                                                                                                                  |
| -------------------------- | ------------------------------------------------------------------------------------------------------------------------------- |
| Declaration contracts      | Portable declaration parsing, canonicalization, and base-schema validation                                                      |
| Strict-profile registry    | Conjunctive profile registration and initialization-time validation                                                             |
| Safe file reader           | One-file sandboxed import reads through `llmtoolsutil.ReadFile`                                                                 |
| Common transfer service    | Preview lifecycle, prepared imports, fingerprints, tokens, expiration, replay, and portable export helpers                      |
| Projected Artifact overlay | In-memory resolution of planned managed Source changes without persistence                                                      |
| Artifact Store transaction | Staged multi-package managed Source mutation, atomic publication, reconciliation, and recovery                                  |
| Agent domain               | Strict admission, package plan, Collection mutation, conflicts, reference catalog, export eligibility, and readiness projection |
| Shared typed resolvers     | Built-in, current-Root, and mapped fallback relationship resolution                                                             |
| MCP installation subsystem | Local installation values, secret references, authentication completion, and connection-local state                             |
| Agent Store API            | Domain-specific import, export, catalog, and readiness endpoints                                                                |

The common transfer layer owns mechanics only. It must not decide:

- Whether a declaration is admissible for a domain.
- Which Collection can receive a declaration.
- How managed packages are addressed.
- Which relationships are permitted.
- Whether an Agent is ready.
- Which conflicts are blocking.

Agent APIs remain the only public entry point for Agent import and export. Future MCP and Text support must use their own domain APIs and profiles.

## Strict managed profile requirements

The strict managed Agent profile is an admission profile, not a portable schema version.

```text
StrictManagedAgentSchema =
  allOf(
    PortableAgentSchema,
    ManagedAgentRestrictions
  )
```

The profile registry constructs this conjunction. The restriction schema is never compiled or used independently as import authority.

A strict-profile-valid document is always valid against the base portable Agent schema because the base schema is a required conjunct.

The strict profile must not:

- Modify the portable Agent schema.
- Add fields to portable declaration instances.
- Add a declaration version.
- Add a Store identity field.
- Create another portable schema key.
- Participate in portable declaration dispatch.

Application initialization must fail when:

- The base Agent schema is not registered.
- The profile declaration type is not `agent`.
- The base schema and restrictions cannot compile as Draft 2020-12 schemas.
- Referenced schemas cannot be resolved.
- The resulting profile is not bound to the declared base schema.
- More than one managed Agent profile is registered.

Every managed Agent import and export must pass:

- Portable Agent schema validation.
- Strict managed Agent profile validation.
- Agent semantic admission checks.

The strict schema owns structural restrictions. The Agent domain owns checks that require Artifact Store or resolver context, including:

- Built-in target existence.
- Mapped fallback support.
- Dependency ambiguity.
- MCP policy provenance.
- Destination Collection compatibility.
- Package and identity conflicts.
- Contained Artifact conflicts.
- Credential placement and secret-placeholder rules.

## Managed Agent profile requirements

A managed import root must:

- Use `type: agent`.
- Use a valid portable `name`.
- Be concrete.
- Have no top-level `locator`.
- Have no `loop`.
- Have no `workflow`.
- Contain only allowed managed-profile members.
- Contain no Store identity, runtime identity, release version, or runtime state.

The normal Agent header remains available:

```text
name
displayName
description
labels
metadata
```

`metadata` remains annotation only. It must not store import paths, Collection identity, source digests, fingerprints, readiness, enablement, MCP installation values, secret references, or runtime configuration.

The managed profile allows the following Agent members.

| Member type  | Allowed form                                                 |
| ------------ | ------------------------------------------------------------ |
| `text`       | Contained inline Text                                        |
| `model`      | Named protected built-in or supported mapped fallback target |
| `tool`       | Named protected built-in or supported mapped fallback target |
| `skill`      | Named protected built-in Skill                               |
| `mcp`        | Named protected built-in MCP or contained inline MCP         |
| `mcp.policy` | Named protected built-in MCP policy                          |

Contained Text must:

- Use outer `insert: instructions` or `insert: user-message`.
- Use inline `content`.
- May use `mediaType` and ordinary descriptive fields allowed by the Text contract.
- Not use a locator.
- Not use `include` or `exclude`.
- Not select external source content.

Model relationships may use:

```yaml
overrides:
  includeSystemPrompt: true
```

Tool relationships may use:

```yaml
overrides:
  autoExecute: false
```

A Tool relationship with `autoExecute: true` requires an explicit import confirmation.

Model and Tool relationships may use `scope: builtin` when selecting a protected built-in Artifact or a built-in-scoped mapped fallback target.

An unscoped Model or Tool relationship is valid only when:

- The registered fallback provider accepts the name as a supported mapped target.
- No current-Root or protected built-in Artifact occurrence shadows that name.

Skill relationships must:

- Be named.
- Use `scope: builtin`.
- Resolve to exactly one protected built-in Skill Artifact.
- Use only supported `use.mode` values.

```text
available
active
instructions
```

Named MCP and MCP policy relationships must:

- Be named.
- Use `scope: builtin`.
- Resolve to exactly one protected built-in Artifact.
- Contain no runtime server ID, discovered capability selection, runtime argument value, or local installation value.

A contained inline MCP declaration may:

- Declare supported portable `stdio` or `streamableHTTP` transport.
- Declare command, arguments, URL, headers, environment names, timeout, authentication requirements, installation input definitions, and supported portable MCP fields.
- Refer to an accepted protected built-in MCP policy.
- Use ordinary descriptive fields allowed by the MCP contract.

A contained inline MCP declaration must not contain:

- A declaration locator.
- An MCP source server selector.
- Installation input values.
- Secret values.
- Secret references.
- OAuth tokens.
- Selected connection profile state.
- Active connection state.
- Discovered Tools, resources, prompts, templates, or digests.
- Credentials placed outside the supported MCP installation-input flow.

Executable `stdio` MCP declarations require explicit confirmation. The Agent semantic validator must reject credential placement that bypasses supported MCP installation and secret handling.

The strict profile rejects all other Agent forms, including:

- Member selectors.
- External relationship locators.
- User-Root dependencies.
- Plugin members.
- Nested Agent members.
- Loop or Workflow members.
- Contained Model, Tool, Skill, Plugin, Agent, Loop, or Workflow declarations.
- Persisted `ArtifactRef` values.
- Root IDs or Source IDs.
- Model preset IDs, Tool bundle IDs, or Assistant Preset identifiers.
- Runtime MCP server IDs.
- Declaration, package, or Agent release versions.

A missing or ambiguous strict-profile dependency is a blocking import issue. Missing local MCP completion is not a declaration error and must be reported as setup required.

## Import preview requirements

Import requires an explicit editable Agent Collection.

The selected Collection determines the target Root and managed Agent Source. Callers must not supply a divergent Root or arbitrary Source.

The selected Collection must:

- Be an Agent-only managed Plugin.
- Belong to a user Root.
- Be editable under Agent domain policy.
- Use the managed Agent Source expected by the Agent domain.
- Be supplied with the expected Collection Artifact revision.

There is no implicit baseline Collection fallback.

Collection enablement does not gate import. Collection enablement and Agent enablement remain independent.

The preview request conceptually contains:

```text
Path
Collection
ExpectedCollectionRevision
ExpectedSourceDigest?
```

The import reader must use a narrow port:

```text
ReadPortableArtifactFile(
  context,
  selectedPath,
  maximumBytes
) -> bytes and normalized path metadata
```

The production adapter must use `llmtoolsutil.ReadFile`.

The reader must:

- Authorize only the explicitly selected file.
- Use a sandboxed and symlink-safe filesystem boundary.
- Apply cross-platform path handling through the filesystem tool.
- Reject directories, devices, and unsupported special files.
- Enforce a bounded read.
- Avoid exposing a generic file-read capability to the frontend.
- Avoid logging declaration content or secret-like values.
- Return file and cancellation errors without exposing unrelated filesystem content.

An absolute pasted path may be handled through a one-file sandbox rooted at the selected file's parent and basename. The safe filesystem implementation remains responsible for containment and symlink handling.

Managed Agent import initially accepts YAML only. The input must contain:

- Exactly one YAML document.
- One object at the YAML root.
- No duplicate mapping keys.
- Bounded alias and expansion behavior.

The selected file is read exactly once for one preview.

The backend calculates the authoritative source digest. A caller-supplied expected source digest is only an optional revalidation guard.

The selected path is preview input only. It must not be:

- Written into the portable declaration.
- Stored as an Agent locator.
- Persisted as Agent state.
- Required during commit.
- Required after import completes.

Preview is read-only and must:

- Validate the destination Collection.
- Safely read and parse the YAML document.
- Validate the portable Agent schema.
- Validate the strict managed profile.
- Run Agent semantic admission.
- Calculate canonical portable declaration bytes and Definition digest.
- Build the managed Agent package plan and exact package files.
- Build the selected Collection member mutation when required.
- Project the root Agent and every contained Artifact.
- Resolve the projected Agent against the target Root, protected built-in Root, and registered fallback providers.
- Inspect Root, package, Source, Collection, and contained identity conflicts.
- Identify exact dangling Collection relationships that would be restored.
- Identify MCP installation, secret, and authentication completion requirements.
- Derive predicted readiness.
- Return normalized portable YAML and structured issues.
- Produce no persistent mutation.

The projected overlay contains:

- The planned managed Agent package Source entry.
- The projected root Agent Artifact.
- Every projected contained Text and MCP Artifact.
- The planned Collection document when changed.
- Current Artifact Store state outside the planned mutation.

The existing typed resolver must run against this overlay. Preview must not implement a separate Agent, Model, Tool, Skill, MCP, Plugin, or fallback resolver.

Projected Artifacts have preview-only identities. Preview responses address them by stable declaration occurrence path and must not expose them as durable `ArtifactRef` values.

A failed preview must not create:

- Managed Source content.
- Definition or Artifact records.
- Collection membership.
- Local MCP installation data.
- Secret data.
- Persistent preview records.

A blocking preview issue returns no committable token.

## Prepared import requirements

A successful importable preview creates an immutable backend-held prepared import.

The prepared import contains:

- Canonical portable Agent declaration bytes.
- Normalized exportable YAML.
- Source digest.
- Definition digest.
- Prepared managed package files.
- Root and contained Artifact expectations.
- Selected Root, Source, and Collection.
- Expected Collection revision.
- Expected managed Source generation.
- Planned Collection document mutation when needed.
- Agent package address.
- Dependency witnesses.
- Conflict analysis.
- Required confirmation issue codes.
- Predicted readiness and MCP setup requirements.
- Creation and expiration times.

The prepared import must not contain:

- Secret values.
- OAuth tokens.
- Resolved secret contents.
- Arbitrary frontend-supplied declaration changes.
- A requirement to reread the external import file.

The preview exposes three distinct digests:

```text
SourceDigest
  -> exact external file bytes read during preview

DefinitionDigest
  -> canonical portable Agent declaration

PreparedFingerprint
  -> complete prepared import plan
```

The prepared fingerprint binds at least:

- Strict profile identity.
- Canonical declaration bytes.
- Managed package files.
- Root and Source identity.
- Selected Collection.
- Expected Collection revision.
- Expected Source generation.
- Package address.
- Planned Collection relationship.
- Projected Artifact expectations.
- Dependency witnesses.
- Required confirmation codes.

A digest is identity, not frontend authority. The frontend must not be able to replace prepared declaration bytes, package content, destination mutation, or projected Artifacts during commit.

The commit token must be:

- Cryptographically random.
- Opaque to the frontend.
- Bound to one prepared import.
- Bound to one destination.
- Bound to one prepared fingerprint.
- Bound to one domain operation.
- Time-limited.

The prepared import registry must:

- Be process-local.
- Be bounded by entry count and total byte size.
- Expire unused entries.
- Remove abandoned entries.
- Retain bounded success receipts for committed-token replay.
- Reject replay with a different fingerprint or confirmation set.
- Require a new preview after application restart.

Commit must not reread the external file.

Commit must revalidate mutable environmental witnesses, including:

- Collection Artifact revision.
- Managed Source generation.
- Relevant Root and Source existence.
- Expected package absence.
- Existing exact dangling memberships.
- Protected built-in Artifact revisions or Definition digests.
- Mapped fallback provider identity.
- Mapped target identifier.
- Fallback catalog generation when exposed by the provider.

A changed witness makes the prepared import stale and requires a new preview.

## Identity, conflict, and Collection requirements

The managed root Agent identity is:

```text
Target Root
  + type: agent
  + logical name
```

The logical name also determines the managed package address.

Managed import is create-only.

The following are prohibited:

- Automatic renaming.
- Package merge.
- Package overwrite.
- Package replacement.
- Equivalent-content no-op import from a new preview.
- Automatic package relocation.

A new preview for an already imported name must report a conflict even when the new declaration is byte-equivalent.

A blocking root Agent conflict includes:

- An available Agent with the same logical name in the target Root.
- A missing but unpurged conflicting Agent occurrence.
- A managed package already occupying the calculated package address.
- A case-folded physical package collision on a case-insensitive platform.
- A name reserved by Agent domain policy.
- A protected built-in Agent with the same logical name.

Every contained Artifact emitted by import must be checked for target-Root conflict.

Contained identity checks include:

```text
Text
  -> type + name + insert

MCP
  -> type + name
```

The Agent domain may enforce an Agent-prefixed naming convention for contained Text and MCP declarations through semantic validation.

The selected Collection membership is a located external Agent relationship.

It must persist:

```text
type: agent
name: <agent-name>
locator: <managed-package-relative-locator>
```

It must not persist an `ArtifactRef`.

Collection relationship handling is:

| Existing Collection member                                 | Import behavior                           |
| ---------------------------------------------------------- | ----------------------------------------- |
| No matching relationship                                   | Add the required located relationship     |
| Exact unavailable located relationship and missing package | Reuse it as a restoration relationship    |
| Exact available located relationship                       | Conflict because the Agent already exists |
| Same name with another locator                             | Blocking conflict                         |
| Same name through symbolic lookup                          | Blocking conflict                         |
| Selector overlapping the imported Agent                    | Blocking conflict                         |
| Contained Agent with the same identity                     | Blocking conflict                         |

Unavailable exact relationships in other Agent Collections are restoration candidates. They do not require rewriting. When the Agent package is restored at the same declaration occurrence, those relationships become available again.

Preview must report affected restored memberships.

Import must not remove or rewrite unrelated Collection members.

Collection membership does not own the Agent:

- One Agent may belong to several Collections.
- Detaching a Collection member does not delete the Agent.
- Deleting an Agent package leaves declared Collection relationships unavailable.
- Deleting a Collection does not delete independent Agents.
- Re-importing the same deleted Agent package may restore all exact dangling relationships to that package occurrence.

Collection deletion, baseline protection, Collection enablement, direct-member deletion guards, and other Collection lifecycle behavior remain governed by the Agent Store HLD.

## Atomic commit and recovery requirements

Commit accepts only:

```text
CommitToken
PreparedFingerprint
AcceptedConfirmationCodes
```

Commit must reject:

- Unknown, expired, or invalid tokens.
- Fingerprint mismatches.
- Missing required confirmations.
- Changed confirmation sets on replay.
- Stale Collection revisions.
- Stale Source generations.
- Stale environmental witnesses.
- Identity or package conflicts.
- Replacement declaration bytes.
- Replacement package files.
- Replacement Collection documents.
- Any import path supplied during commit.

A successful commit publishes exactly the package plan prepared during preview.

One Agent import transaction may include:

- Creation of one managed Agent package.
- Replacement of the selected managed Collection package when a relationship must be added.
- No Collection package change when the exact required relationship already exists.
- Reconciliation of the resulting Source generation.
- Verification of the expected root and contained Artifacts.

Both packages must belong to the same authorized managed Agent Source. Otherwise import is rejected.

The transaction must not modify unrelated packages.

The durable transaction boundary is the managed Source package set.

Commit sequence is:

```text
Load prepared import
  -> verify token, fingerprint, and confirmations
  -> acquire managed Source transaction lock
  -> verify Source generation and Collection revision
  -> verify package absence, identity state, and dependency witnesses
  -> stage Agent package
  -> stage Collection package mutation when needed
  -> validate staged Source projection
  -> atomically publish staged package set
  -> reconcile Source once
  -> verify expected Artifacts and Collection
  -> store success receipt
  -> return result
```

The staged Source projection must be equivalent to the projection used during preview.

The transaction must not expose durable state where:

- The selected Collection relationship exists without the prepared Agent package.
- The prepared Agent package exists without the required selected Collection relationship.
- Only part of the prepared Agent package is published.
- A conflicting package was replaced.
- Unrelated packages were changed.

After durable success, the token is consumed for new mutation attempts. A duplicate request using the same token, fingerprint, and confirmations returns the original success result from the replay receipt.

If interruption occurs after Source publication and before the response:

- The transaction receipt or Source generation identifies the committed mutation.
- Source reconciliation is rerun idempotently.
- Expected Artifacts and Collection membership are reverified.
- Replay returns the recovered original result.
- No second Agent package or duplicate Collection member is created.

No Agent-specific transaction database or sidecar file may be introduced. Generic Artifact Store staging and recovery metadata may be extended for this purpose.

New imported Agents default to `Artifact.Enabled=true`.

## Export and managed Agent lifecycle requirements

Only a user-managed top-level Agent is eligible for managed export.

The requested Artifact must:

- Be an Agent.
- Be available.
- Belong to a non-protected user Root.
- Be top-level rather than contained.
- Be backed by the managed Agent Source.
- Use the managed Agent package kind.
- Use the expected concrete Agent document location.
- Not be a source-selected alias.
- Pass portable and strict managed profile validation.

Built-in Agents, repository-authored Agents, and contained child Agents are not eligible for managed Agent export.

Export reads the current immutable Definition selected by the Agent Artifact. It validates the Definition against the portable Agent schema and strict managed profile, then returns normalized portable YAML.

Current relationship availability and readiness do not gate export eligibility. They are returned separately for inspection.

The export result includes:

```text
Type
Name
MediaType
SuggestedFileName
Content
ContentDigest
DefinitionDigest
ArtifactRevision
Resolution
Readiness
```

For Agent export:

```text
MediaType: application/yaml
SuggestedFileName: <agent-name>.agent.yaml
```

Managed export is declaration recovery, not Store backup.

Managed Agent declaration content is immutable after import. The Agent management API must not expose:

- Declaration patch.
- Member add or remove.
- Agent rename.
- Agent replacement.
- Agent upsert.
- New Agent version creation.
- In-application declaration editing.

A changed declaration requires one of these workflows:

```text
Export
  -> modify externally
  -> use a new Agent name
  -> import as another Agent
```

```text
Export
  -> delete current managed Agent
  -> modify externally
  -> import using the original Agent name
```

Deleting a managed Agent removes its package. It does not detach the Agent from Collections. Declared relationships remain and become unavailable until an exact package occurrence is restored.

Artifact enablement and MCP local completion are local-state operations. They do not edit the Agent Definition.

## Readiness and reference catalog requirements

Agent management exposes separate readiness dimensions:

| Field                  | Meaning                                                                                    |
| ---------------------- | ------------------------------------------------------------------------------------------ |
| `DeclarationValid`     | Current Definition passes portable and strict managed validation                           |
| `ResolutionComplete`   | Every required relationship is available and unambiguous                                   |
| `InstallationComplete` | Required MCP installation inputs, secret references, and authentication setup are complete |
| `Ready`                | `DeclarationValid`, `ResolutionComplete`, and `InstallationComplete` are true              |

`Ready` means configuration readiness only.

It does not assert:

- Active MCP connection.
- Network reachability.
- Model provider availability.
- Tool execution permission.
- Runtime scheduling availability.
- Successful Agent execution.

Missing MCP completion does not invalidate an otherwise valid declaration. It prevents readiness.

Preview must identify inline and selected MCP requirements by stable declaration occurrence path, for example:

```text
members/mcp/example-server
```

After commit, contained MCP declarations have terminal Artifact identities. Existing MCP installation and secret APIs own completion after the Artifact exists.

MCP completion may include:

- Installation input values.
- Secret references.
- Authentication setup.
- Local connection profile selection where supported.
- Local policy or runtime configuration owned by the MCP subsystem.

MCP completion must not modify the Agent Definition.

Preview returns only MCP setup descriptors, including:

- Input name.
- Input kind.
- Label.
- Description.
- Required status.
- Client-secret requirement.
- Stable occurrence path.

Preview must never return or retain secret values.

The Agent readiness aggregate derives readiness on demand from:

```text
Resolved Agent capability plan
  + terminal MCP Artifact identities
  + MCP installation completion
  + secret-reference health
  + authentication setup health
```

Readiness must not be persisted in:

- Agent YAML.
- Agent Definition.
- Generic Artifact state.
- Agent metadata.
- Collection membership.

The Agent domain must expose a bounded reference catalog for names accepted by the strict managed profile.

The catalog includes supported:

- Protected built-in Models.
- Protected built-in Tools.
- Protected built-in Skills.
- Protected built-in MCP servers.
- Protected built-in MCP policies.
- Mapped Model targets accepted by fallback providers.
- Mapped Tool targets accepted by fallback providers.

Each catalog item includes:

- Artifact type.
- Logical name.
- Display name.
- Description where available.
- Built-in or mapped provenance.
- Supported Agent relationship behavior.
- Current declaration availability.
- Copyable YAML relationship snippet.

The catalog is informational only. It must not:

- Add a relationship to an Agent.
- Persist target IDs.
- Become a composition editor.
- Treat runtime readiness as declaration availability.

A fallback provider that exposes mapped targets in the catalog must provide a bounded catalog port in addition to exact-name resolution.

## Backend flow

Preview flow:

```text
Selected Collection + YAML path
  -> validate Collection and expected revision
  -> safely read selected file once
  -> parse one bounded YAML object
  -> validate portable Agent schema
  -> validate strict managed profile
  -> run semantic admission
  -> build managed package and Collection mutation
  -> project Artifacts and resolve relationships
  -> inspect conflicts and MCP completion requirements
  -> calculate readiness
  -> retain prepared import
  -> return preview
```

Commit flow:

```text
Commit token + prepared fingerprint + confirmations
  -> load backend-held prepared import
  -> revalidate mutable witnesses
  -> stage package set
  -> atomically publish managed Source mutation
  -> reconcile and verify Artifacts
  -> persist generic success receipt
  -> return Agent, Collection, restored memberships, and readiness
```

Revalidation is a new preview:

```text
Changed file, destination, Collection revision, or confirmation-relevant policy
  -> new safe file read
  -> new prepared import
  -> new token and fingerprint
```

Revalidation does not mutate the previous prepared import or persistent Artifact state.

Export flow:

```text
Managed Agent ArtifactRef
  -> verify export eligibility
  -> load current immutable Definition
  -> validate portable and strict profile
  -> normalize portable YAML
  -> return export response
```

## Backend API contract

Concrete Go naming may follow repository conventions, but the backend API surface must provide equivalent operations.

```text
ListAgentImportDestinations()
PreviewAgentImport(...)
CommitAgentImport(...)
ExportManagedAgent(...)
ListAgentImportReferenceCatalog(...)
GetAgentReadiness(...)
```

`ListAgentImportDestinations` returns only eligible editable managed Agent Collections.

Each destination includes:

```text
RootID
RootDisplayName
Collection
CollectionRevision
CollectionName
CollectionDisplayName
Baseline
Enabled
```

`PreviewAgentImport` accepts:

```text
AgentImportPreviewRequest {
  Path
  Collection
  ExpectedCollectionRevision
  ExpectedSourceDigest?
}
```

`PreviewAgentImport` returns:

```text
AgentImportPreview {
  CommitToken?
  PreparedFingerprint?
  ExpiresAt?

  SourceDigest
  DefinitionDigest
  NormalizedYAML

  Agent
  Destination
  ProjectedArtifacts
  Relationships
  Conflicts
  RestoredMemberships
  MCPSetupRequirements
  Readiness

  CanImport
  RequiresConfirmation
  ReadyAfterCommit

  Issues
}
```

Preview issues use these severities:

```text
error
confirmation
setup
information
```

Each issue contains:

```text
code
severity
path
message
instruction
```

Source line and column may be included when YAML parsing makes them available.

Validation, policy, conflict, and readiness findings are returned as preview issues. Unexpected infrastructure failures remain operation errors.

`CommitAgentImport` accepts:

```text
AgentImportCommitRequest {
  CommitToken
  PreparedFingerprint
  AcceptedConfirmationCodes
}
```

`CommitAgentImport` returns:

```text
AgentImportCommitResult {
  Agent
  Collection
  RestoredMemberships
  Readiness
  CommitFingerprint
}
```

No declaration content, import path, package bytes, projected Artifact data, or Collection replacement content is accepted by commit.

`ExportManagedAgent` accepts:

```text
AgentExportRequest {
  Agent
}
```

`Agent` is an authorized Agent `ArtifactRef`.

`ListAgentImportReferenceCatalog` may accept destination Collection context so built-in and mapped names are evaluated in the correct Root context.

`GetAgentReadiness` returns:

```text
AgentReadiness {
  DeclarationValid
  ResolutionComplete
  InstallationComplete
  Ready
  Issues
  MCPRequirements
}
```

No generic frontend API may accept an arbitrary declaration type and route it through common import infrastructure without domain policy.

## Persistence and security requirements

The feature persists only normal Artifact ecosystem state:

| Persisted through existing systems                 | Not persisted by this feature                                      |
| -------------------------------------------------- | ------------------------------------------------------------------ |
| Managed Agent package                              | External import path                                               |
| Selected Collection package update                 | External source file outside the managed package                   |
| Definitions and Artifacts                          | Commit token in the declaration                                    |
| Managed Source generation and reconciliation state | Source digest in the declaration                                   |
| Artifact enablement                                | Prepared fingerprint in the declaration                            |
| MCP installation data after separate completion    | Readiness in the declaration                                       |
| Secret references in MCP installation state        | Collection membership in the Agent declaration                     |
| Secret values in secret storage                    | Secret values in preview, prepared state, export, or Agent package |

Portable managed export must not include:

- Collection membership or Collection identity.
- Root ID, Source ID, Artifact ID, or package address.
- Artifact revision, Definition digest, source digest, or fingerprint fields inside YAML.
- Local Agent enablement.
- MCP installation input values.
- Secret references or secret values.
- OAuth state.
- Selected MCP connection profile.
- MCP connection state.
- Discovered MCP Tools, resources, prompts, templates, or digests.
- Runtime state.
- Readiness state.

Portable MCP installation input definitions remain exportable because they are declaration content.

Prepared imports are process-local and temporary. They do not survive expiration or application restart.

Security-sensitive preview data must identify behavior without exposing secrets. Preview must call out:

- Stdio commands.
- Command arguments.
- HTTP URLs.
- Header names.
- Environment variable names.
- Authentication modes.
- Auto-execute overrides.
- Required secret inputs.
- Approval-relevant MCP policies.

Artifact enablement must not be represented as a security sandbox.

## Breaking removals

The change is development-time breaking.

The implementation must remove:

- Assistant Preset APIs.
- Assistant Preset wire types.
- Assistant Preset storage.
- Assistant Preset embedded data and overlays.
- Assistant Preset versions and bundle IDs.
- Assistant Preset compatibility aliases.
- Assistant Preset migration.
- Assistant Preset dual-read and dual-write behavior.
- Assistant Preset frontend routes, generated bindings, and management labels.
- Structured managed Agent creation.
- Structured managed Agent replacement.
- Field-by-field managed Agent composition APIs.

The following managed Agent APIs and types are removed:

- `ManagedAgentDocument`.
- `ManagedAgentMember`.
- `ManagedAgentCreateRequest`.
- `ManagedAgentCreateResult`.
- `ManagedAgentReplaceRequest`.
- `ManagedAgentReplaceResult`.
- `CreateManagedAgent`.
- `ReplaceManagedAgent`.

The following legacy backend areas are removed:

```text
cmd/agentgo/
  wrapper_assistantpreset_artifact.go

internal/assistantpreset/
  lookupimpl/
  spec/
  store/
```

Legacy Assistant Preset embedded data, overlays, storage constants, and application storage declarations are removed.

The new design does not retain:

- Assistant Preset versions or new-version creation.
- Bundle enablement gating.
- Ordered Tool selection.
- Ordered Skill selection.
- Persisted Tool user arguments.
- Persisted MCP conversation context.
- Persisted discovered MCP capabilities.
- Field-by-field Model, Tool, Skill, or MCP selection.
- Copying another preset through a form.
- Managed Agent declaration editing.
- Managed Agent declaration replacement.

No compatibility decoder, migration layer, or version bump is required.

## Implementation requirements

Strict-profile infrastructure belongs under:

```text
internal/artifactcontract/managedprofile/
  descriptor.go
  registry.go
  schema.go
  validation.go
```

It must:

- Bind one restrictions schema to one portable base schema.
- Construct the `allOf` profile schema.
- Compile and validate all registered profiles at initialization.
- Reject duplicate or invalid registration.
- Expose strict validation to domain services.
- Keep profiles out of portable declaration dispatch.

Agent profile additions belong under:

```text
internal/artifactcontract/declaration/agentv1/
  agent-managed-restrictions.schema.json
  managed_profile.go
```

The existing Agent schema key and portable schema version must not change.

Common transfer infrastructure belongs under:

```text
internal/managedartifact/transfer/
  file_reader.go
  preview.go
  prepared.go
  registry.go
  fingerprint.go
  export.go
  types.go
```

It must provide:

- Safe file-reader integration.
- Source digest calculation.
- Canonical portable content handling.
- Prepared import storage.
- Commit tokens.
- Prepared fingerprints.
- Expiration and replay handling.
- Shared preview issue types.
- Shared portable export response types.
- No public domain-bypassing import API.

The production file-reader adapter uses:

```text
internal/llmtoolsutil.ReadFile
```

Artifact Store transaction support must provide a generic managed Source transaction or equivalent batch mutation API with:

- Expected Source generation.
- Expected package state.
- Multiple package mutations in one Source.
- Package creation and replacement operations.
- Staged projection validation.
- Atomic package-set publication.
- One Source reconciliation.
- Expected Artifact verification.
- Durable idempotent recovery.

Representative implementation location:

```text
internal/artifactstore/managedtransaction/
```

or an equivalent extension of existing managed Source infrastructure.

Agent domain implementation belongs under:

```text
internal/agent/store/
  consumerapi/
    import.go
    export.go
    reference_catalog.go
    readiness.go
    import_types.go

  domain/
    managed_profile.go
    import_policy.go
    import_conflicts.go
    import_package.go
```

It must implement:

- Agent strict semantic admission.
- Destination validation.
- Managed package planning.
- Collection membership planning.
- Root and contained identity conflict checks.
- Dangling-membership restoration analysis.
- Built-in and mapped reference validation.
- Export eligibility.
- Reference catalog projection.
- Readiness aggregation.

Existing managed Agent publication code may be refactored into the atomic transaction planner, but it must not remain exposed as structured authoring.

`AgentStoreWrapper` must expose:

- Import destination listing.
- Reference catalog listing.
- Import preview.
- Prepared import commit.
- Managed Agent export.
- Agent readiness and management detail.

Future MCP and Text import/export must add their own:

- Restriction schemas.
- Domain profiles.
- Destination policies.
- Package plans.
- Conflict policies.
- Export eligibility.
- Readiness projections.

The common transfer service must not gain Agent-, MCP-, or Text-specific conditionals.

Verification must cover:

- Strict-profile registration and base-schema binding.
- Safe YAML reading and duplicate-key rejection.
- Preview non-mutation.
- Projected resolution and diagnostics.
- Built-in and mapped fallback admission.
- Inline MCP credential restrictions.
- Identity and package conflicts.
- Dangling-membership restoration.
- Stale witness rejection.
- Token expiration and replay behavior.
- Atomic publication and interrupted reconciliation recovery.
- Secret redaction.
- Export eligibility and exclusion behavior.
- Removal of structured managed Agent and Assistant Preset APIs.

## Implementation status

| Capability                                        | Status    | Notes                                                           |
| ------------------------------------------------- | --------- | --------------------------------------------------------------- |
| Portable Agent schema and canonical YAML decoding | Available | Existing `agentv1` contract remains unchanged                   |
| Managed Agent Source and package layout           | Available | Existing managed Agent Source infrastructure is reused          |
| Agent Collections                                 | Available | Existing Agent-only Plugin domain is reused                     |
| Built-in Agent and capability resolution          | Available | Protected Root and typed resolver infrastructure exist          |
| Tool and Model mapped fallback                    | Available | Existing fallback-provider infrastructure is reused             |
| Safe cross-platform file operations               | Available | `llmtoolsutil` and filesystem tooling are available             |
| Strict managed profile registry                   | Planned   | Common conjunctive schema registration                          |
| Agent managed restrictions schema                 | Planned   | Additional admission schema beside `agentv1`                    |
| Initialization-time profile validation            | Planned   | Fail-fast application composition                               |
| Safe Agent YAML preview reader                    | Planned   | Uses the narrow safe file-reader port                           |
| Projected Artifact overlay                        | Planned   | Required for mutation-free preview                              |
| Prepared import registry                          | Planned   | Token, fingerprint, expiration, and replay state                |
| Environmental witness validation                  | Planned   | Collection, Source, built-in, mapped target, and provider state |
| Atomic multi-package managed Source transaction   | Planned   | Agent package plus Collection package mutation                  |
| Managed Agent import and export APIs              | Planned   | Agent-specific consumer API and wrapper integration             |
| Managed Agent reference catalog                   | Planned   | Built-in and mapped names with snippets                         |
| Derived Agent readiness                           | Planned   | Resolution plus MCP local completion                            |
| Structured managed Agent authoring                | Removed   | Replaced by strict file import                                  |
| Managed Agent replacement                         | Removed   | Delete and re-import or use a new name                          |
| Assistant Preset backend and frontend             | Removed   | No compatibility or migration path                              |
| MCP import and export profile                     | Deferred  | Reuses common transfer infrastructure                           |
| Text import and export profile                    | Deferred  | Reuses common transfer infrastructure                           |
