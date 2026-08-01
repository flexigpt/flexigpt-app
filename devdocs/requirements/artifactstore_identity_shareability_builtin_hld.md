# Artifact Store Identity, Shareability, and Built-in Topology Contract

## 1. Authority

This document is authoritative for:

- Caller-supplied Artifact Store IDs.
- Create replay behavior.
- Shareable Skill package boundaries.
- Protected built-in topology.
- Semantic built-in reference resolution.
- Workspace exclusion from built-in ownership.

It applies to backend storage, package installation, runtime composition,
assistant presets, MCP defaults, imports, exports, Wails bindings, and UI work.

## 2. Caller-supplied IDs

- Root, Source, Collection, and Artifact IDs are caller-supplied UUIDv7 values.
- Generic Artifact Store services validate IDs and never allocate them.
- User-facing clients generate UUIDv7 IDs before explicit create requests.
- Feature-owned automatic reconciliation must provide `artifact.Draft.ID`
  before invoking Artifact Store reconciliation.
- Feature-owned automatic-adoption ID providers are composed outside Artifact
  Store. Generic Artifact Store packages expose no ID-provider abstraction and
  have no dependency on `uuidutil.Generator`.
- `uuidutil` remains permitted only behind `basespec` UUIDv7 validation.
- Revisions remain the optimistic-concurrency mechanism for mutable state.

Static built-in IDs are committed UUIDv7 values in application metadata.
Registry loading validates every Root, Source, Collection, and Artifact ID.

## 3. Create replay

Explicit creation follows this contract:

1. Validate the caller-supplied UUIDv7 ID.
2. Attempt insertion.
3. Return the newly created entity when absent.
4. On an ID conflict, read the existing entity with that same ID.
5. Return the existing entity only when creation intent matches.
6. Return `basespec.ErrConflict` when creation intent differs.

Creation intent is intentionally narrower than all mutable fields:

| Entity     | Creation intent                                                         |
| ---------- | ----------------------------------------------------------------------- |
| Root       | Root ID, initial display name, initial description                      |
| Source     | Root ID, Source ID, Source kind, normalized private configuration       |
| Collection | Root ID, Collection ID, Collection kind                                 |
| Artifact   | Root ID, Artifact ID, Collection ID, binding, kind, adoption mode, name |

Feature services may impose additional semantic intent. `skill.bundle`, for
example, validates portable logical bundle metadata. Protected built-in setup
also validates static display metadata, enablement, and required attachment
topology after generic creation succeeds.

Semantic uniqueness remains independent of ID uniqueness:

- Artifact source bindings are unique by Root, Collection, Source, locator,
  subresource locator, and expected kind.
- Different Artifact IDs for the same binding conflict.
- An Artifact ID does not bypass source-binding uniqueness.
- Feature logical-name uniqueness remains feature-owned.

## 4. No idempotency keys

Artifact Store has no `IdempotencyKey` field, request field, index, or schema
column.

The caller-supplied entity ID is the replay identity for explicit creation.
Managed Skill creation uses its caller-supplied `ArtifactID` as the replay
identity and package SHA-256 as immutable package-content intent.

Source revisions and Source generations are request-time concurrency tokens.
They are not durable idempotency keys and must not be stored as managed-Skill
operation state in Artifact local data.

## 5. Shareable Skill package data

Shareable Skill package data may contain only:

- Logical bundle name.
- Optional logical bundle version.
- Logical Skill name.
- Optional logical Skill version.
- Package files, including `SKILL.md`.
- `SKILL.md` metadata and instructions.
- An externally supplied package SHA-256.

It must not contain:

- Root, Source, Collection, or Artifact IDs.
- Revisions, timestamps, enablement, install state, or runtime state.
- SQLite paths, local filesystem paths, Source configuration, or credentials.
- Attachment roles, app-specific metadata, user metadata, or operation keys.

`skillbundle.ShareableSkillPackage` is the portable package form. Its supplied
SHA-256 is verified against canonical package bytes. It is distinct from local
Collection data, Artifact data, source configuration, and the built-in registry.

## 6. Protected built-in topology

The built-in registry is embedded application metadata. It is not a shareable
package manifest.

It declares:

- One protected Root ID.
- One managed Source ID.
- Static Collection IDs for logical bundles.
- Static Artifact IDs for logical Skills.
- Logical names, optional logical versions, package locations, enablement
  defaults, and a registry schema version.

Artifact Store owns only the generic protected topology declaration:

```text
artifactstore/topology.Declaration
  Root
  Sources
```

Artifact Store does not import `artifactbuiltin`, built-in registry JSON,
embedded package files, Skill Bundle schemas, or Agent Skill semantics.

`artifactbuiltin` is an incoming application installer. It receives:

- The validated application registry.
- An injected embedded or external `fs.FS`.
- `topology.Ensurer`.
- Narrow Skill Bundle installation ports.

The installer validates package locations before mutation, ensures the one
protected Root and Source, then creates static feature Collections and
Artifacts through Skill Bundle APIs.

Ordinary APIs cannot create, update, retire, purge, attach to, refresh, publish
to, or remove packages from the protected Root. Only a trusted installer or
update path carrying the installer capability may mutate it.

`root.Service.EnsureSystem` additionally requires that the requested Root ID
matches the application-configured protected-root policy. A privileged context
does not grant permission to create arbitrary Roots through the protected
topology path.

### 6.1 Protected managed package operations

Artifact Store exposes ordinary and protected managed-package operations.

- `PublishManagedPackage` and `RemoveManagedPackage` reject a protected Root,
  including when an accidental privileged context is supplied.
- `PublishProtectedManagedPackage` and
  `RemoveProtectedManagedPackage` require both a declared protected Root and
  the trusted installer capability.
- Application composition injects protected operations only into the
  appropriate feature installer or update path.
- The public Artifact Store transport uses ordinary operations only and cannot
  mutate the protected Source.

This is a generic protected-topology mechanism. Artifact Store does not infer
that a protected package is a Skill, Assistant, MCP server, or any other
application feature.

### 6.2 Installation convergence and dynamic-record fence

Application startup must invoke the application-owned built-in installer. It
must not install built-ins from a Root-created callback and must never iterate
user Roots as installation targets.

The trusted installer sequence is:

1. Load and validate embedded application registry metadata.
2. Validate every registry UUIDv7 value and every package directory against
   the injected embedded or external `fs.FS`.
3. Ensure the one declared protected Root and managed Source.
4. Reject an active `skill.bundle` in the protected Root that uses a declared
   built-in logical bundle name with a non-registry Collection ID.
5. Ensure each static built-in Collection and attachment topology.
6. Reject an undeclared active `agent.skill` Artifact in a canonical built-in
   Skill Bundle.
7. Ensure each static Artifact and its managed package through the protected
   package publication path.
8. Refresh only when the catalog is unavailable or stale. A current catalog
   must not be republished merely because the application restarted.

The installer never adopts, copies, renames, or migrates dynamic built-in
records into canonical topology. Legacy Artifact Store metadata is rejected by
the clean `artifacts_v1` namespace gate. A manually contaminated `artifacts_v1`
installation fails with conflict and requires an explicit offline repair tool.

## 7. Built-in semantic references

Shareable built-in references are semantic:

```json
{
  "bundle": "core-instructions",
  "skill": "markdown-output"
}
```

They must not serialize local `ArtifactRef`, Root IDs, Artifact IDs, package
paths, or installation metadata.

`builtin/metadata.ResolveBuiltInSkill` resolves a semantic reference through
only protected application metadata and returns the static local
`artifact.ArtifactRef`.

Installed app-local presets may store the resolved Artifact reference in local
SQLite metadata. Export must convert that local reference back to a semantic
built-in reference or reject export as non-portable.

## 8. Workspace exclusion

Workspace has no built-in library feature.

- Workspace may not resolve protected built-in Artifact references.
- Workspace may not attach the protected built-in Source.
- Workspace may not store `workspace_library_references`.
- Workspace Root equality remains strict.
- Workspace runtime registration contains only Workspace-owned Artifacts.
- Workspace feature services reject the configured protected Root before
  creating, reading, listing, refreshing, or mutating a Workspace.

The existing `library` Workspace attachment role remains a same-Root local
Source role. It is not a built-in or cross-Root reference mechanism.

Built-in Skills are installed in protected `skill.bundle` Collections and are
used by app-owned assistant, MCP, and runtime-default integrations through
semantic built-in references.

## 9. Storage migration boundary

- There is no schema migration ledger.
- Artifact Store does not inspect, classify, migrate, or adapt legacy Artifact
  Store metadata. Deployment chooses a fresh directory instead.
- Dynamic legacy built-ins are never imported into canonical built-in topology.
- Fresh installation creates exactly one protected Root and one protected
  managed Source after the trusted installer runs.

## 10. Required acceptance tests

- Same-ID create with matching intent returns the existing record.
- Same-ID create with changed creation intent conflicts.
- Different Artifact IDs for the same source binding conflict.
- Every static registry ID passes UUIDv7 validation.
- Fresh installation creates exactly one protected Root and Source.
- Built-in Collections and Artifacts use registry IDs.
- Creating user Roots creates no built-in copies.
- Ordinary callers cannot mutate or purge the protected Root.
- Registry package locations resolve to regular embedded package directories.
- Shareable packages contain no local identity or app metadata fields.
- Semantic built-in references resolve to static protected Artifact references.
- Workspace rejects all protected-Root Artifact references.
