# Artifact Store Protected Overlay and Secret Binding Extension

## Goals

- Preserve the existing `Artifact.Data` behavior for mutable user artifacts.
- Add local, non-secret customization for protected built-in artifacts.
- Store secret values outside SQLite, artifact definitions, source content, overlays, and `Artifact.Data`.
- Reuse the existing MapStore and `keyringencdec` encryption model.
- Store secret references, SHA-256 metadata, ownership, lifecycle state, and cleanup state in Artifact Store SQLite metadata.
- Preserve overlays and secret bindings during routine built-in package refreshes when Artifact identity remains unchanged.
- Remove overlays and bindings during destructive topology resets and permanent Artifact removal.
- Keep Artifact Store source-agnostic and domain-agnostic.
- Support Model, MCP, Tool, Skill, and future protected domains through the same protected overlay mechanism.
- Remove settings-based authentication-key persistence and global provider credentials.
- Keep SQLite as the default persistence adapter while using narrow repository interfaces for future replaceability.

## Requirements

### Scope

This proposal is an additive Artifact Store extension. It does not rewrite Artifact identity, source handling, topology, SQLite composition, normal mutable Artifact metadata, source adapters, or source synchronization.

The core distinction is:

| Artifact category           | Non-secret local customization                          | Secret values                                          |
| --------------------------- | ------------------------------------------------------- | ------------------------------------------------------ |
| Mutable user Artifact       | Existing `Artifact.Data`                                | New Artifact Store secret binding and secret reference |
| Protected built-in Artifact | New protected Artifact overlay stored by Artifact Store | New Artifact Store secret binding and secret reference |

Existing `Artifact.Data` behavior remains correct for mutable user Artifacts:

- It is local metadata.
- It survives source refresh.
- It is not overwritten by `SourceStateUpdate`.
- It is editable through `Artifacts.UpdateData`.
- It is intentionally blocked for protected roots.

Protected built-ins require a separate overlay because they must not be changed through `Artifact.Data`, source files, or source definitions.

### Existing MapStore and Keyring Behavior

The existing settings implementation already uses MapStore and `keyringencdec` correctly.

Its effective storage model is:

```text
OS keyring:
  AES-256-GCM encryption key

settings.json:
  encrypted MapStore value ciphertext
  SHA-256 metadata
  non-empty metadata
  auth-key grouping metadata
```

Plaintext secrets are not stored in `settings.json`.

`EncryptedStringValueEncoderDecoder` does not store each secret directly as a native OS-keychain item. It stores an encryption key in the OS keyring and stores AES-GCM ciphertext in MapStore-backed JSON.

This is an acceptable model and should be reused for Artifact Store secrets.

The new Artifact Store secret extension uses this storage model:

```text
OS keyring:
  Artifact Store secret-encryption key

Artifact Store dedicated MapStore secret files:
  encrypted secret ciphertext only

Artifact Store SQLite metadata:
  secret refs
  SHA-256 values
  ownership and binding metadata
  cleanup state
```

Secret values remain protected through the existing MapStore encryption plugin. SQLite contains neither plaintext secrets nor encrypted secret ciphertext.

### Non-Goals

This change must not:

- Replace `Artifact.Data`.
- Replace source adapters.
- Change source discovery.
- Change definition admission.
- Change Artifact synchronization.
- Change source or package handling.
- Make Artifact Store understand Model or MCP declaration semantics.
- Make SQLite the only possible persistence implementation.
- Store secrets in Artifact definitions, source content, `Artifact.Data`, or overlay payloads.
- Add a generalized Artifact identity or rebinding system.
- Preserve built-in overlays across destructive topology reset unless explicitly added later.

Existing behavior remains unchanged:

- Routine built-in package refresh preserves Artifact IDs when typed source binding is unchanged.
- Routine built-in package updates preserve overlays and secret bindings automatically.
- A destructive protected topology reset removes the root and its local state through the existing `PurgeRoot` lifecycle.

### Additive Architecture

Artifact Store gains three narrow extension areas.

```text
Artifact Store
├── Existing Artifact.Data
│   └── Mutable/user Artifact non-secret local metadata
│
├── Protected Artifact Overlay extension
│   └── Protected built-in Artifact non-secret local metadata
│
└── Secret Binding extension
    ├── SQLite metadata:
    │   ├── secret refs
    │   ├── SHA-256
    │   ├── owner Artifact
    │   ├── namespace and slot
    │   └── garbage-collection state
    │
    └── Pluggable value backend:
        └── MapStore + keyringencdec by default
```

The overlay and secret systems remain distinct:

- Overlay records contain non-secret canonical JSON only.
- Secret bindings contain references and metadata only.
- The secret-value backend contains secret values only.
- Artifact Store SQLite metadata contains references, SHA values, ownership, lifecycle state, and cleanup records.
- No secret value belongs in `Artifact.Data`.
- No secret value belongs in an overlay payload.

### Protected Artifact Overlays

#### Protected Artifact Definition

Artifact Store must not hard-code a built-in root ID.

The existing generic root-protection mechanism remains authoritative:

```go
ProtectionAPI.IsProtectedRoot(rootID)
```

The internal feature name is:

```text
Protected Artifact Overlay
```

Application code may call protected-root overlays built-in overlays where appropriate.

This supports:

- Model built-ins.
- MCP built-ins.
- Tool built-ins.
- Skill built-ins.
- Future protected product topology.

Ordinary mutable Artifacts do not use protected overlays. They continue using `Artifact.Data`.

#### Overlay Identity

A protected overlay is identified by:

```text
Root ID
Artifact ID
Overlay namespace
```

It must not be identified by:

- Logical name.
- Source locator.
- Package path.
- Definition digest.
- Source ID.
- Artifact kind alone.

The exact `ArtifactRef` is required so an overlay cannot transfer to another Artifact with the same logical name.

Example overlay keys:

```text
builtin-root / artifact-id / model.provider.runtime.v1
builtin-root / artifact-id / model.runtime.v1
builtin-root / artifact-id / mcp.runtime.v1
```

#### Overlay Namespace Registration

Artifact Store receives registered overlay namespaces at composition time.

```go
package overlay

type Namespace string

func (v Namespace) Validate() error {
  return basespec.ValidateIdentifier(
    "protected overlay namespace",
    string(v),
    basespec.MaxKindBytes,
  )
}
```

Application-owned namespaces may include:

```go
const (
  ModelProviderRuntimeNamespace overlay.Namespace = "model.provider.runtime"
  ModelRuntimeNamespace         overlay.Namespace = "model.runtime"
  MCPRuntimeNamespace           overlay.Namespace = "mcp.runtime"
)
```

Composition configuration includes registered namespaces and a secret-value backend.

```go
type Config struct {
  BaseDirectory string

  EmbeddedProviders map[string]fs.FS
  Providers         []providerapi.Provider

  ProtectedRootIDs []root.RootID
  RetainedRoots    []root.RootDraft

  ProtectedOverlayNamespaces []overlay.Namespace
  SecretValues               secretapi.ValueStore
}
```

Namespace rules:

- Duplicate namespaces fail composition.
- Unregistered namespaces are rejected.
- Artifact Store validates namespace syntax and payload canonicalization.
- Model, MCP, Tool, Skill, and future domains own payload schema validation.
- Artifact Store must not import Model or MCP packages.
- New modules register their own namespace during application composition.

#### Overlay Payload Ownership

Artifact Store owns:

- Persistence.
- Namespace registration.
- Revision checks.
- Protected-root checks.
- Artifact existence checks.
- Canonical JSON checks.
- Lifecycle cleanup.

Domain packages own:

- Payload fields.
- Schema-version meaning.
- Semantic validation.
- Runtime interpretation.
- Compatibility rules.

For example, the Model package validates a Provider runtime payload before sending it to Artifact Store. Artifact Store stores canonical JSON under the Model namespace.

This matches the ownership model already used by generic `Artifact.Data`.

#### Overlay API

`compositionapi.Store` gains a narrow protected overlay API.

```go
type ProtectedOverlayAPI interface {
  Get(
    ctx context.Context,
    ref artifact.ArtifactRef,
    namespace overlay.Namespace,
  ) (overlay.Record, bool, error)

  Put(
    ctx context.Context,
    request overlay.PutRequest,
  ) (overlay.Record, error)

  Delete(
    ctx context.Context,
    ref artifact.ArtifactRef,
    namespace overlay.Namespace,
    expectedArtifactRevision uint64,
    expectedOverlayRevision uint64,
  ) error
}
```

Suggested overlay types:

```go
package overlay

type Record struct {
  Artifact      artifact.ArtifactRef
  Namespace     Namespace
  SchemaVersion string
  Payload       json.RawMessage
  Revision      uint64
  CreatedAt     time.Time
  ModifiedAt    time.Time
}

type PutRequest struct {
  Artifact artifact.ArtifactRef

  Namespace     Namespace
  SchemaVersion string
  Payload       json.RawMessage

  ExpectedArtifactRevision uint64
  ExpectedOverlayRevision  uint64
}
```

Overlay rules:

- `ExpectedArtifactRevision` must match the current Artifact revision.
- `ExpectedOverlayRevision == 0` means create only.
- A non-zero `ExpectedOverlayRevision` must match the existing overlay revision.
- New overlays begin at revision `1`.
- Overlay updates increment a normal `uint64` revision.
- `Payload` must be a canonical JSON object.
- `Payload` is bounded by `basespec.MaxLocalDataBytes`.
- The target Artifact must belong to a protected root.
- The target Artifact must be available for public read and write operations.
- `Delete` removes both the non-secret overlay payload and all secret bindings in that namespace.

#### Public and Maintenance Operations

Public overlay operations require:

- An existing Artifact.
- A protected root.
- An available Artifact state.
- A matching expected Artifact revision.
- A matching expected overlay revision.

Artifact Store also requires a trusted lifecycle-only maintenance API.

```go
type LocalStateMaintenanceAPI interface {
  PurgeArtifactLocalState(
    ctx context.Context,
    ref artifact.ArtifactRef,
  ) error
}
```

This API is used when:

- A built-in managed package is explicitly removed.
- A stale compiled package disappears during hydration.
- A protected topology root is reset.
- An Artifact is permanently purged.

It must operate even when the Artifact is `missing`, because cleanup may happen after source refresh marks the Artifact unavailable.

It must not be exposed through frontend or Wails bindings.

#### Overlay SQLite Storage

Artifact Store adds the following logical SQLite table.

```sql
CREATE TABLE artifact_protected_overlays (
    root_id TEXT NOT NULL,
    artifact_id TEXT NOT NULL,
    namespace TEXT NOT NULL,

    schema_version TEXT NOT NULL,
    payload_json BLOB NOT NULL,

    revision INTEGER NOT NULL CHECK (revision > 0),
    created_at INTEGER NOT NULL,
    modified_at INTEGER NOT NULL,

    PRIMARY KEY (root_id, artifact_id, namespace)
);
```

Required index:

```sql
CREATE INDEX idx_artifact_protected_overlays_artifact
    ON artifact_protected_overlays(root_id, artifact_id);
```

Both the repository and the domain service must validate:

```text
overlay.root_id == artifact.root_id
overlay.artifact_id exists in that root
```

Overlay fields must not be added to:

- `artifact.Artifact`.
- `catalog.Entry`.
- `definition.Definition`.
- `source.Source`.
- `source.RefreshState`.

An overlay is local state adjacent to an Artifact, not source-derived Artifact state.

### Secret Bindings

#### Secret Storage Rule

The secret-value backend stores only secret values.

SQLite stores:

- Secret references.
- SHA-256 values.
- Binding ownership.
- Artifact references.
- Namespace and slot names.
- Revisions.
- Pending writes.
- Cleanup state.

The following must never contain secret values:

- Artifact definitions.
- Source content.
- Overlay payloads.
- `Artifact.Data`.
- SQLite metadata.
- Runtime configuration persistence.

#### Secret Reference

Secret references are typed opaque values.

```go
package secret

type Ref string
```

Suggested physical form:

```text
secret.v1/<uuid-v7>
```

Examples:

```text
secret.v1/0197d810-6e9b-7000-8000-000000000001
secret.v1/0197d810-6e9b-7000-8000-000000000002
```

Secret-reference rules:

- Artifact Store generates references.
- Callers never provide a reference for a newly written secret.
- References are not derived from provider names, Artifact names, paths, or secret values.
- Secret values are never overwritten in place.
- Replacing a credential creates a new secret reference.
- The old reference is detached and queued for cleanup only after metadata publication succeeds.

This is required because SQLite and MapStore cannot share one transaction.

#### Secret Binding Identity

A secret binding is identified by:

```text
Artifact reference
Namespace
Slot
```

Example Model Provider API-key binding:

```text
ArtifactRef:
  builtin-root / openai-provider-artifact

Namespace:
  model.provider.runtime

Slot:
  apiKey
```

Example MCP binding:

```text
ArtifactRef:
  mcp-server-artifact

Namespace:
  mcp.runtime

Slot:
  header.authorization
```

The slot is not an arbitrary UI-provided environment variable or raw header name. The relevant domain owns and validates stable slot names.

#### Secret Binding API

Management metadata operations remain separate from runtime secret reads.

```go
type SecretBindingAPI interface {
  GetBinding(
    ctx context.Context,
    key secret.BindingKey,
  ) (secret.Binding, bool, error)

  ReplaceBinding(
    ctx context.Context,
    request secret.ReplaceBindingRequest,
  ) (secret.Binding, error)

  ClearBinding(
    ctx context.Context,
    request secret.ClearBindingRequest,
  ) error
}
```

Suggested types:

```go
package secret

type BindingKey struct {
  Artifact  artifact.ArtifactRef
  Namespace overlay.Namespace
  Slot      string
}

type Binding struct {
  Key      BindingKey
  Ref      Ref
  SHA256   string
  Revision uint64
  Active   bool
}

type ReplaceBindingRequest struct {
  Key BindingKey

  ExpectedArtifactRevision uint64
  ExpectedBindingRevision  uint64

  Value string
}

type ClearBindingRequest struct {
  Key BindingKey

  ExpectedArtifactRevision uint64
  ExpectedBindingRevision  uint64
}
```

Binding rules:

- Binding revision is a normal `uint64` revision.
- `ExpectedBindingRevision == 0` means create only.
- The secret service validates the current Artifact reference and Artifact revision.
- Normal user operations reject unavailable Artifacts.
- Trusted maintenance operations may detach bindings from missing Artifacts.
- Empty input means clear the binding at the typed Model or MCP command layer.
- The low-level secret store must not persist an empty secret as a credential.
- SHA-256 is calculated from plaintext before encryption and stored in SQLite.
- SHA-256 preserves useful existing metadata behavior.
- The initial implementation must not attach the same active secret reference to multiple bindings.

#### Runtime Secret Read API

A generic `GetSecret(ref)` API must not be exposed through frontend wrappers.

Trusted runtime code uses a narrower API.

```go
type SecretRuntimeAPI interface {
  ReadBinding(
    ctx context.Context,
    key secret.BindingKey,
    expectedRef secret.Ref,
  ) (string, secret.Binding, error)
}
```

Runtime read behavior:

- Verify that the binding currently exists.
- Verify that the binding is active.
- Verify that the current reference matches `expectedRef`.
- Read the secret from the configured value backend.
- Return plaintext only to trusted Model or MCP runtime code.
- Never expose this API through Wails bindings.
- Never pass this API to decoders, providers, source adapters, or schemas.

The Model runtime resolves a Provider Artifact, resolves the active `apiKey` binding, and reads that binding only when creating inference runtime configuration.

The MCP runtime resolves only the secret slots required by the selected MCP server.

### Secret Metadata Storage

#### Secret Records

```sql
CREATE TABLE artifact_secret_records (
    ref TEXT PRIMARY KEY,
    store_name TEXT NOT NULL,

    sha256 TEXT NOT NULL,

    state TEXT NOT NULL CHECK (
        state IN ('pending', 'active', 'cleanup')
    ),

    created_at INTEGER NOT NULL,
    modified_at INTEGER NOT NULL
);
```

Record state meanings:

- `pending` means SQLite knows the intended reference but it is not attached to a live Artifact binding.
- `active` means the secret reference is attached to an active binding.
- `cleanup` means the secret is detached and must be deleted from the physical secret backend.

No plaintext or ciphertext appears in this table.

#### Secret Bindings

```sql
CREATE TABLE artifact_secret_bindings (
    root_id TEXT NOT NULL,
    artifact_id TEXT NOT NULL,

    namespace TEXT NOT NULL,
    slot TEXT NOT NULL,

    secret_ref TEXT,
    revision INTEGER NOT NULL CHECK (revision > 0),

    created_at INTEGER NOT NULL,
    modified_at INTEGER NOT NULL,

    PRIMARY KEY (root_id, artifact_id, namespace, slot),

    FOREIGN KEY (secret_ref)
        REFERENCES artifact_secret_records(ref)
);
```

Binding rules:

- A live binding has a non-null `secret_ref`.
- A cleared binding has a null `secret_ref`.
- Binding revision increments after replacement or clearing.
- The same active secret reference must not be attached to multiple bindings in the initial implementation.
- This keeps ownership and cleanup unambiguous.

#### Secret Cleanup Queue

```sql
CREATE TABLE artifact_secret_cleanup (
    secret_ref TEXT PRIMARY KEY,
    store_name TEXT NOT NULL,

    attempts INTEGER NOT NULL DEFAULT 0,
    last_error TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL,
    modified_at INTEGER NOT NULL
);
```

Cleanup queue rules:

- It must not have foreign keys to Root, Artifact, Source, or Overlay rows.
- It must survive Artifact and root deletion.
- It contains only references and backend identifiers.
- It is the durable queue for physical secret deletion.

### Secret Value Backend

#### Backend Interface

Artifact Store depends on a small secret-value port.

```go
package secretapi

type ValueStore interface {
  Name() string

  Put(
    ctx context.Context,
    ref secret.Ref,
    value string,
  ) error

  Get(
    ctx context.Context,
    ref secret.Ref,
  ) (string, error)

  Delete(
    ctx context.Context,
    ref secret.Ref,
  ) error

  Close() error
}
```

The value store receives:

- Opaque secret references.
- Secret values.

The value store must not receive:

- Artifact references.
- Root IDs.
- Logical names.
- Overlay namespaces.
- SHA values.
- Source locators.
- Definition data.

This keeps the physical secret backend secret-only.

#### Default Keyring-Backed MapStore Backend

Add a package such as:

```text
internal/artifactstore/secretstore/keyringmapstore
```

It uses:

- `mapstore-go`.
- `mapstore-go/jsonencdec`.
- `mapstore-go/keyringencdec`.

Suggested physical layout:

```text
<artifact-store-base>/
  secrets/
    v1/
      <opaque-secret-id>.json
```

Each file represents exactly one secret reference.

Conceptually, each file contains:

```json
{
  "value": "AES-GCM ciphertext produced by keyringencdec"
}
```

Secret files contain:

- One encrypted secret value.
- No Artifact metadata.
- No SHA value.
- No namespace.
- No slot.
- No source path.
- No logical name.

The filename derives only from the opaque secret-reference ID.

A separate MapStore file is required for every secret reference because:

- Reading one secret must not decode every secret.
- A global MapStore would load all secret values into memory through `GetAll`.
- Per-secret deletion is straightforward.
- Per-secret cleanup does not require rewriting a global secret file.

#### Dedicated Keyring Codec

The backend creates a dedicated codec.

```go
codec, err := keyringencdec.NewEncryptedStringValueEncoderDecoder(
  "FlexiGPTArtifactSecretStore",
  "artifactstore-v1",
)
```

It must not reuse the existing settings codec service:

```text
FlexiGPTKeyRingEncDec / user
```

The separate service name ensures:

- Removing settings auth keys cannot affect Artifact Store secrets.
- Secret-store lifecycle remains independent from settings lifecycle.
- Future secret-backend migration remains isolated.

The MapStore `ValueEncDecGetter` returns the codec only for the secret leaf value path.

```go
func secretValueEncoder(path []string) mapstore.IOEncoderDecoder {
  if len(path) == 1 && path[0] == "value" {
    return codec
  }
  return nil
}
```

MapStore must write and read the `value` field. Secret plaintext must never be written using `os.WriteFile`.

#### File Permissions

The backend must create:

```text
secrets directory: 0700
secret files:      0600
```

Ciphertext is encrypted, but the directory and files must remain private.

The secret directory is owned by the secret backend. It is not source content and is not part of the managed package tree.

#### Create-Only Secret References

`ValueStore.Put` must fail when a physical secret reference already exists.

It must never overwrite an existing reference.

Artifact Store creates a new reference for every replacement. A duplicate reference indicates corruption or an unexpected retry path.

This prevents an incorrect replay from corrupting a currently active credential.

#### Strict Lost-Key Handling

The current `EncryptedStringValueEncoderDecoder` behavior is acceptable for initial settings setup but unsafe for a durable dedicated secret store.

Current behavior:

```go
case errors.Is(err, ErrNotFound):
  // Generate a new AES key.
```

If the keyring entry is deleted while encrypted secret files remain, generating a new key makes existing ciphertext permanently unreadable.

Add a MapStore option:

```go
func WithCreateKeyIfMissing(enabled bool) Option
```

Required behavior:

- Existing defaults remain `true` for backward compatibility.
- Artifact Store secret reads use strict mode.
- Strict mode returns an error when the keyring key is missing.
- Strict mode must not silently generate a replacement key while encrypted secret records exist.
- A first-ever secret write may create the key.
- Later reads and writes require the existing key.
- Missing keys return a typed secret-store-unavailable error.
- Recovery requires restoring the keychain item or explicitly resetting the secret store.

This is a small MapStore and keyring extension, not an Artifact Store rewrite.

#### Keychain Unavailable Behavior

Artifact Store must still open when the operating-system keychain is unavailable.

`keyringencdec.NewEncryptedStringValueEncoderDecoder` does not access the keychain until encryption or decryption is attempted.

| Operation                                 | Keychain unavailable behavior |
| ----------------------------------------- | ----------------------------- |
| Artifact listing                          | Works                         |
| Source refresh                            | Works                         |
| Built-in non-secret overlay read or write | Works                         |
| Settings read or write                    | Works                         |
| Secret metadata read                      | Works                         |
| Secret binding clear                      | Works logically               |
| Secret replacement                        | Fails                         |
| Runtime secret resolution                 | Fails                         |
| Physical secret cleanup                   | Remains queued                |

No plaintext fallback is allowed.

## Workflows

### Secret Binding Replacement

SQLite and MapStore cannot share one transaction.

A currently bound secret reference must never be overwritten.

Given:

```text
ArtifactRef
Namespace
Slot
Expected Artifact Revision
Expected Binding Revision
New plaintext secret
```

The replacement workflow is:

- Validate the request and target Artifact.
- Generate a new opaque `secret.Ref`.
- Calculate SHA-256 from the new plaintext secret.
- Insert an `artifact_secret_records` row in SQLite with state `pending`.
- Write plaintext to the secret-value backend under the new reference.
- In one SQLite transaction:
  - Re-read and validate the Artifact revision.
  - Re-read and validate the binding revision.
  - Attach the new reference to the binding.
  - Increment the binding revision.
  - Mark the new secret record `active`.
  - Mark the previous secret record `cleanup`.
  - Insert the previous reference into `artifact_secret_cleanup`.
- Run cleanup on a best-effort basis after the SQLite transaction commits.
- Return the new binding metadata.

If the physical secret write fails:

- The existing binding remains unchanged.
- The pending record is marked `cleanup`.
- Cleanup removes any partially created physical value if one exists.

If the final SQLite transaction conflicts:

- The existing binding remains unchanged.
- The new pending reference is marked `cleanup`.
- Cleanup removes the unused new physical secret.

This prevents a secret value changing before an overlay compare-and-swap succeeds.

### Secret Binding Clear

Clearing a binding must not delete the physical secret first.

The clear workflow is:

- Verify the Artifact revision.
- Verify the binding revision.
- Set `secret_ref = NULL`.
- Increment the binding revision.
- Mark the previous secret record `cleanup`.
- Insert the previous reference into the cleanup queue.
- Commit the SQLite transaction.
- Run cleanup on a best-effort basis.

If cleanup fails:

- The credential is logically detached.
- Runtime cannot resolve it through Artifact Store.
- The encrypted physical secret remains queued for deletion.
- Startup or a later mutation retries cleanup.

### Batch Secret Updates

MCP may need multiple secret values to change together.

Add an internal batch operation.

```go
ReplaceBindings(
  ctx context.Context,
  request secret.ReplaceBindingsRequest,
) ([]secret.Binding, error)
```

The batch workflow must:

- Reserve all pending secret references in SQLite.
- Write all physical secret values.
- Attach all bindings in a single SQLite transaction.
- Attach none if any physical value write fails.
- Queue every staged reference for cleanup if the operation fails.

This prevents partially updated logical MCP configuration when multiple header or environment secrets change together.

### Combined Protected Overlay and Secret Update

Protected built-ins require a combined internal operation.

```go
ApplyProtectedOverlay(
  ctx context.Context,
  request protectedoverlay.ApplyRequest,
) (protectedoverlay.Result, error)
```

The request may include:

- A replacement non-secret overlay payload.
- Secret-slot replacements.
- Secret-slot clears.

The workflow is:

- Validate the protected Artifact, namespace, payload, revisions, and secret slots.
- Write new physical secret values before publishing metadata.
- Commit the following in one final SQLite transaction:
  - Overlay payload.
  - Overlay revision.
  - Secret-binding changes.
  - Secret cleanup queue entries.
- Run cleanup after the transaction commits.

This supports MCP configuration edits where public connection metadata and associated secret slots must change together.

Model Provider API-key changes normally use the simpler `ReplaceBinding` operation because they typically update only the `apiKey` slot.

### Normal Source Refresh

No source-refresh behavior changes are required.

Existing behavior intentionally excludes local metadata from `SourceStateUpdate`:

```go
// SourceStateUpdate intentionally excludes:
// DisplayName
// Enabled
// Data
```

Protected overlays and secret-binding tables must also remain untouched by source refresh.

As a result:

- Source refresh does not overwrite overlays.
- Source refresh does not remove secret bindings.
- Source refresh does not contact MapStore or the keychain.

### Routine Built-In Package Update

Existing synchronization behavior remains useful.

Artifacts are matched by:

```text
root_id + source_id + locator + subresource_locator + kind
```

When a built-in package changes content but retains the same typed source binding:

- The existing Artifact row remains.
- The existing Artifact ID remains.
- The protected overlay remains.
- The secret binding remains.
- The new source-derived Definition replaces only source-owned state.

This is the expected routine built-in update behavior.

### Explicit Built-In Package Removal

When a compiled built-in package is removed:

- Source refresh marks the previous Artifact `missing`.
- The Artifact row may remain for diagnostics.
- Protected overlays and secret bindings must be removed explicitly.
- Old secret references must be queued for cleanup.

Local state must not be deleted merely because an external source is temporarily unavailable.

Cleanup occurs only for known managed deletion or known stale built-in package removal.

The existing Model built-in lifecycle may remain, but it must call Artifact Store local-state maintenance instead of an external settings overlay repository.

Previous behavior:

```go
if !plan.TopologyCurrent {
  l.overlays.PurgeRoot(...)
}
```

Required behavior:

```text
Remove this call. Artifact Store topology purge owns root-local overlay and secret cleanup.
```

Stale package cleanup workflow:

- Capture only prior available Artifact references.
- Hydrate or refresh current packages.
- Compare prior available Artifact references with currently available Artifact references.
- Call the following only when a previous Artifact is no longer available:

```go
LocalStateMaintenanceAPI.PurgeArtifactLocalState(ctx, ref)
```

`CaptureBuiltInPackageArtifacts` must require `StateAvailable`.

Without this correction, a removed package may still have an Artifact row in `StateMissing`, causing reconciliation to incorrectly consider it current.

### Protected Topology Reset

`sqlite.PurgeTopologyRoot` currently deletes root metadata directly.

It must be extended to:

- Select all secret references bound to Artifacts in the root.
- Insert those references into `artifact_secret_cleanup`.
- Delete protected overlays for the root.
- Delete secret bindings for the root.
- Delete ordinary Artifact and root metadata as it does today.
- Commit the metadata transaction.
- Run physical secret cleanup after the transaction.
- Retain cleanup queue rows if physical cleanup fails.

The physical MapStore secret backend must not be called inside the SQLite transaction.

This replaces the existing intended behavior of `SettingsOverlayRepository.PurgeRoot` with Artifact Store-owned lifecycle management.

### Permanent Artifact Purge

A permanent Artifact purge must queue secret cleanup before deleting binding metadata.

A pure `ON DELETE CASCADE` approach is insufficient because it would remove the binding row and lose the secret reference before cleanup is scheduled.

Use one or both of:

- A repository helper used by every Artifact purge path.
- A SQLite `BEFORE DELETE` trigger that inserts references into `artifact_secret_cleanup`.

Cleanup insertion should use `INSERT OR IGNORE`.

The cleanup queue must survive Artifact and root deletion.

### Model Integration

#### Model Local State

Replace the settings-backed persistence role of:

```text
internal/model/store/overlay/settings_overlay_repository.go
```

with a Model local-state adapter that branches by root policy.

```text
Protected Model Provider:
  non-secret payload -> Protected Artifact Overlay

Mutable Model Provider:
  non-secret payload -> Artifact.Data

Protected Model:
  non-secret payload -> Protected Artifact Overlay

Mutable Model:
  non-secret payload -> Artifact.Data

All Model credentials:
  secret binding -> SQLite
  secret value -> keyring-backed MapStore
```

#### Model Provider Payload

The existing persisted Provider overlay contains:

```go
type ProviderOverlay struct {
  SchemaVersion string
  Revision      uint64

  CredentialRef string

  Connection        json.RawMessage
  Defaults          json.RawMessage
  Capabilities      json.RawMessage
  DefaultModel      *declaration.ArtifactNameReference
  AdapterParameters json.RawMessage
}
```

Replace it with a non-secret payload:

```go
type ProviderOverlayPayload struct {
  Connection        json.RawMessage
  Defaults          json.RawMessage
  Capabilities      json.RawMessage
  DefaultModel      *declaration.ArtifactNameReference
  AdapterParameters json.RawMessage
}
```

The following become persistence metadata rather than payload fields:

- Overlay schema version.
- Overlay revision.
- Credential reference.
- Credential SHA.
- Credential binding revision.

Credential references exist only in Artifact Store SQLite secret-binding metadata.

#### Model Runtime Payload

Use the following non-secret Model payload:

```go
type ModelOverlayPayload struct {
  Defaults          json.RawMessage
  Capabilities      json.RawMessage
  AdapterParameters json.RawMessage
}
```

For protected Models, persist this through `ProtectedOverlayAPI`.

For mutable Models, persist it in a namespaced `Artifact.Data` field:

```text
flexigpt.site/model-runtime-v1
```

Use existing helpers:

```go
artifact.DecodeDataObject
artifact.EncodeDataObject
```

This preserves `Artifact.Data` fields owned by other consumers.

#### Mutable Provider Default Model

The current mutable Provider default Model uses a separate `Artifact.Data` namespace:

```text
flexigpt.site/model-provider-default-model-v1
```

As an active-development breaking change, fold this value into the mutable Provider runtime payload.

Provider runtime state becomes:

| Provider type      | Local Provider runtime state                               |
| ------------------ | ---------------------------------------------------------- |
| Mutable Provider   | `Artifact.Data["flexigpt.site/model-provider-runtime-v1"]` |
| Protected Provider | `artifact_protected_overlays` row                          |

The resolution order becomes:

```text
local Provider runtime default Model
Provider declaration default Model
first enabled linked Model fallback
```

This removes the previous protected-versus-mutable split:

```text
protected Provider default -> external settings overlay
mutable Provider default -> Artifact.Data
```

#### Provider API Key

Model Provider API keys use one binding slot:

```text
Namespace: model.provider.runtime
Slot:      apiKey
```

Provider API keys must no longer be stored in:

- Settings auth-key values.
- `ProviderOverlay.CredentialRef`.
- `Artifact.Data`.
- Model source definitions.
- Runtime overlay JSON.

Provider runtime views become conceptually:

```go
type ProviderRuntimeOverlayView struct {
  OverlayRevision uint64

  Connection        json.RawMessage
  Defaults          json.RawMessage
  Capabilities      json.RawMessage
  DefaultModel      *declaration.ArtifactNameReference
  AdapterParameters json.RawMessage

  CredentialConfigured bool
  CredentialSHA256     string
  CredentialRevision   uint64
}
```

Exposing `CredentialSHA256` is permitted under the existing metadata policy. It may remain internal if the frontend does not need it.

#### Model Runtime Credential Resolution

Replace the settings-backed resolver:

```go
type CredentialResolver interface {
  ResolveModelCredential(
    ctx context.Context,
    ref string,
  ) (Credential, error)
}
```

with a binding-aware resolver:

```go
type CredentialResolver interface {
  ResolveModelCredential(
    ctx context.Context,
    binding secret.Binding,
  ) (Credential, error)
}
```

The resolver:

- Receives the active Provider API-key binding.
- Calls `ArtifactStore.SecretRuntimeAPI.ReadBinding`.
- Returns:

```go
type Credential struct {
  APIKey  string
  Version string
}
```

Use the stored SHA-256 as `Credential.Version` to preserve inference fingerprint behavior.

Runtime fingerprints may continue to include:

- Model Artifact revision.
- Model Definition digest.
- Provider Artifact revision.
- Provider Definition digest.
- Overlay revisions.
- Credential SHA-256.
- Adapter version.

Runtime fingerprints must never include plaintext secret values.

#### Model Default Provider Preference

`modelSettingsAdapter` currently stores default-provider preference through encrypted auth-key storage.

This preference is not secret and must become a normal setting.

```go
type ModelPreferences struct {
  DefaultProvider *artifact.ArtifactRef `json:"defaultProvider,omitempty"`
}
```

Create a small adapter implementing:

```go
aggregate.DefaultProviderPreferences
```

using standard `SettingStore` methods.

The old encrypted auth-key storage path must not remain for default-provider preference.

#### Provider Credential Update

Replace the existing `ModelStoreWrapper` sequence:

```text
write settings secret
then update settings overlay
```

with:

```text
read current Provider and secret binding metadata
validate Provider and authentication mode
replace secret binding through Artifact Store
return binding metadata and Provider runtime view
```

When the secret is empty:

```text
clear binding
queue old secret for cleanup
```

Empty encrypted secrets must not be used as placeholders.

### MCP Integration

#### MCP Non-Secret Runtime State

Protected MCP Artifacts use:

```text
artifact_protected_overlays
namespace: mcp.runtime
```

Mutable MCP Artifacts use:

```text
Artifact.Data
namespace: flexigpt.site/mcp-runtime-v1
```

#### MCP Secret Slots

MCP defines stable logical slots, such as:

```text
transport.authorization
header.authorization
header.x-api-key
env.token
env.client-secret
```

The MCP domain owns:

- Legal slots.
- Payload fields that refer to slots.
- Header normalization.
- Environment variable mapping.
- Runtime injection rules.

Artifact Store owns:

- Bindings.
- Secret references.
- SHA metadata.
- Physical secret storage.
- Cleanup.

#### MCP Runtime Secret Resolution

The MCP runtime must:

- Resolve the source-backed MCP Artifact.
- Resolve local non-secret configuration.
- Resolve active secret bindings for declared slots.
- Read secret values immediately before creating a runtime connection or process environment.
- Inject secrets into the selected MCP transport.
- Never serialize secret values into persisted runtime configuration.

For stdio MCP:

- Inject secrets only into the launched child-process environment.
- Do not call `os.Setenv`.
- Do not mutate application-global environment state.

For HTTP MCP:

- Build secret headers only for the selected connection.
- Do not persist those headers into overlays or settings.

### Settings Authentication-Key Removal

Remove the following from `internal/setting/spec`:

- `AuthKeyType`.
- `AuthKeyName`.
- `AuthKey`.
- `AuthKeysSchema`.
- `AuthKeyMeta`.
- `GetAuthKeyRequest`.
- `SetAuthKeyRequest`.
- `DeleteAuthKeyRequest`.
- Auth-key responses.
- Auth-key errors.

Remove this field from `SettingsSchema`:

```go
AuthKeys AuthKeysSchema
```

Remove the following from `SettingStore`:

- `encEncrypt`.
- `valueEncDecGetter`.
- `SetAuthKey`.
- `GetAuthKey`.
- `DeleteAuthKey`.
- `BuiltInAuthKeys`.
- Auth-key migration.
- Auth-key defaults.
- `computeSHA`.

Remove this method from `SettingStoreWrapper`:

```go
GetAuthKey
```

Remove these methods from `CompletionWrapper`:

```go
SetAuthKey
DeleteAuthKey
```

Remove the legacy global-provider credential path:

```go
w.providersetAPI.SetProviderAPIKey(...)
```

Model runtime credentials must be Artifact-addressed and resolved through the Model Aggregate rather than globally mutated by provider name.

#### Breaking Migration Policy

This is an active-development breaking change.

The migration policy is:

- Delete the old top-level `authKeys` field from settings.
- Do not retain a fallback reader.
- Do not transparently migrate encrypted settings credentials into Artifact Store.
- Require re-entry of Model and MCP secrets.
- Keep the old OS-keychain encryption key if it already exists.
- Use a new Artifact Store keyring service name for new secret encryption.

This avoids guessing which previous settings key belongs to which current `ArtifactRef`.

## Implementation Items

### Generic Extension Types

Add the following packages:

```text
basespec/overlay
basespec/secret
secretapi
```

Implement:

- Namespace validation.
- Secret-reference validation.
- Binding-key validation.
- Metadata-only binding types.
- Typed errors.

Suggested errors:

```go
ErrOverlayNotFound
ErrSecretBindingNotFound
ErrSecretUnavailable
ErrSecretValueNotFound
ErrSecretCleanupPending
```

### SQLite Schema and Repository Extensions

Add:

- Protected overlay table.
- Secret record table.
- Secret binding table.
- Secret cleanup queue.
- Required indexes.
- Root and Artifact purge cleanup handling.
- Additive existing-schema migration in `initializeSchema`.

Do not alter existing Source, Definition, Artifact, or Root semantics.

### Protected Overlay Service

Implement:

```text
internal/artifactstore/internal/protectedoverlay
```

Dependencies:

- Artifact repository reader.
- Root policy.
- Overlay repository.
- Clock.
- Registered namespace set.

Responsibilities:

- Protected-root validation.
- Artifact-state validation.
- Revision compare-and-swap.
- Canonical JSON validation.
- Overlay persistence.
- Namespace registration enforcement.
- Lifecycle deletion support.

### Secret Service

Implement:

```text
internal/artifactstore/internal/secret
```

Dependencies:

- Artifact repository reader.
- Secret metadata repository.
- `secretapi.ValueStore`.
- Clock.

Responsibilities:

- Generate opaque secret references.
- Compute SHA-256.
- Stage pending metadata.
- Write encrypted physical values.
- Attach and detach bindings.
- Queue cleanup.
- Resolve runtime binding values.
- Retry cleanup at startup and after mutations.
- Preserve logical correctness when physical cleanup is unavailable.

### Keyring-Backed MapStore Secret Backend

Implement:

```text
internal/artifactstore/secretstore/keyringmapstore
```

Requirements:

- One encrypted MapStore file per secret reference.
- Dedicated keyring service and user.
- No global secret map.
- Strict lost-key behavior.
- Create-only secret references.
- Private directory and file permissions.
- No secret metadata in secret files.
- No fallback storage.
- No direct plaintext filesystem writes.

### Narrow Repository Ports

Do not rewrite all Artifact Store persistence behind a large generalized backend abstraction.

Follow the existing DDD pattern with narrow repository ports.

```go
type ProtectedOverlayRepository interface {
  Get(...)
  Put(...)
  Delete(...)
  DeleteByArtifact(...)
  DeleteByRoot(...)
}

type SecretMetadataRepository interface {
  CreatePending(...)
  ActivateBinding(...)
  ClearBinding(...)
  ListCleanup(...)
  CompleteCleanup(...)
}
```

Architecture requirements:

- `internal/artifactstore/internal/protectedoverlay.Service` depends on repository interfaces.
- `internal/artifactstore/internal/secret.Service` depends on metadata repository interfaces and `secretapi.ValueStore`.
- `internal/artifactstore/internal/sqlite` implements the interfaces.
- `system.Components` wires the SQLite implementations.
- `compositionapi.Store` exposes narrow APIs.

This is consistent with existing patterns:

- `artifactimpl.Repository`.
- `sourceimpl.Repository`.
- `rootimpl.Repository`.
- `refreshimpl.Publisher`.

### Composition Changes

Add these fields to `compositionapi.Config`:

```go
type Config struct {
  BaseDirectory string

  EmbeddedProviders map[string]fs.FS
  Providers         []providerapi.Provider

  ProtectedRootIDs []root.RootID
  RetainedRoots    []root.RootDraft

  ProtectedOverlayNamespaces []overlay.Namespace
  SecretValues               secretapi.ValueStore
}
```

Add these fields to `compositionapi.Store`:

```go
type Store struct {
  Roots            RootAPI
  Sources          SourceAPI
  Discovery        DiscoveryAPI
  Artifacts        ArtifactAPI
  Resources        ResourceAPI
  Schemas          SchemaAPI
  ManagedArtifacts ManagedArtifactAPI
  Protection       ProtectionAPI
  Topology         installerapi.API

  ProtectedOverlays ProtectedOverlayAPI
  SecretBindings    SecretBindingAPI
  SecretRuntime     SecretRuntimeAPI
  LocalState        LocalStateMaintenanceAPI

  LocatorResolvers []providerapi.LocatorResolverFactory
}
```

`SecretRuntime` is a trusted internal composition capability only.

It must not be exposed through Wails wrappers.

### Application Composition

`composeArtifactStore` creates the keyring-backed MapStore secret backend and passes it into Artifact Store composition.

```go
secretValues, err := keyringmapstore.New(
  filepath.Join(baseDirectory, "secrets"),
  keyringmapstore.Config{
    KeyringService: "FlexiGPTArtifactSecretStore",
    KeyringUser:    "artifactstore-v1",
  },
)
if err != nil {
  return nil, err
}

return compositionapi.Open(
  ctx,
  compositionapi.Config{
    BaseDirectory: baseDirectory,
    Providers:     providers,

    ProtectedRootIDs: documentTopology.ProtectedRootIDs(),
    RetainedRoots:    documentTopology.RetainedRootDrafts(),

    ProtectedOverlayNamespaces: []overlay.Namespace{
      modelOverlay.ProviderRuntimeNamespace,
      modelOverlay.ModelRuntimeNamespace,
      mcpOverlay.RuntimeNamespace,
    },
    SecretValues: secretValues,
  },
)
```

Ownership rules:

- On successful `compositionapi.Open`, Artifact Store owns `SecretValues`.
- If `compositionapi.Open` fails after receiving `SecretValues`, Artifact Store closes it.
- `Store.Close` closes the secret backend after cleanup work has stopped.
- Pending secret cleanup runs after opening the extension.
- Artifact Store startup must not fail solely because the keychain is unavailable.

### Model Migration

Replace:

```text
modelSettingsAdapter
SettingsOverlayRepository
modelCredentialResolver
```

with:

```text
Artifact-backed Model local-state adapter
Artifact Store secret-binding adapter
Artifact Store runtime secret reader
Normal settings-backed default-provider preference adapter
```

Model migration requirements:

- Protected Provider runtime state uses protected overlays.
- Mutable Provider runtime state uses namespaced `Artifact.Data`.
- Protected Model runtime state uses protected overlays.
- Mutable Model runtime state uses namespaced `Artifact.Data`.
- Provider API keys use `model.provider.runtime / apiKey` secret bindings.
- Default-provider preference becomes an ordinary setting.
- Runtime credentials resolve through `SecretRuntimeAPI`.
- Inference fingerprints use credential SHA-256 rather than plaintext credentials.

### MCP Migration

Apply the same split:

```text
Protected MCP non-secret configuration -> protected overlay
Mutable MCP non-secret configuration -> Artifact.Data
MCP secret values -> secret bindings + secret backend
```

MCP migration requirements:

- Define stable typed secret slots.
- Validate slots in MCP domain code.
- Use batch replacement for multi-secret configuration changes.
- Use combined protected overlay and secret updates when non-secret and secret MCP configuration must change together.
- Inject stdio secrets only into child-process environments.
- Inject HTTP secrets only into the active connection request or client configuration.

### Built-In Lifecycle Changes

Remove this dependency from Model built-in installer construction:

```go
Overlays modelOverlay.RootPurger
```

Remove this lifecycle behavior:

```go
if !plan.TopologyCurrent {
  l.overlays.PurgeRoot(...)
}
```

Protected topology reset now deletes overlays and queues secret cleanup inside Artifact Store.

Retain Model lifecycle responsibilities that remain domain-specific:

- Identify Model package scopes.
- Apply initial enablement labels.
- Identify previously available package Artifacts that are no longer available after hydration.

Replace direct settings-overlay deletion with:

```go
artifactStore.LocalState.PurgeArtifactLocalState(ctx, ref)
```

Only purge prior Artifacts when they are no longer currently available.

### Source and Provider Extensibility

The extension remains independent of Artifact origin.

It works for Artifacts from:

- Filesystem directories.
- Managed package directories.
- Embedded filesystems.
- Archive source adapters.
- Git source adapters.
- Registry source adapters.
- Future database source adapters.

Overlay and secret-binding keys use only:

```text
ArtifactRef
Namespace
Slot
```

They never use:

- Native paths.
- Source configuration.
- Locators.
- Filesystem roots.
- Package layouts.
- Decoder internals.

The secret backend is not:

- A source adapter.
- An Artifact Provider.
- A schema codec.
- A source persistence mechanism.

It is a separately configured secret-value extension.

## Expected Outcome

| Requirement                   | Result                                                         |
| ----------------------------- | -------------------------------------------------------------- |
| User Artifact overrides       | Continue using `Artifact.Data`                                 |
| Built-in non-secret overrides | Stored in Artifact Store protected overlay rows                |
| Built-in source definitions   | Remain immutable and untouched                                 |
| Secrets                       | Stored only in dedicated encrypted MapStore secret backend     |
| Encryption key                | Stored in OS keyring through `keyringencdec`                   |
| Secret refs and SHA           | Stored in Artifact Store SQLite metadata                       |
| Model credentials             | Bound to Provider Artifacts through secret bindings            |
| MCP secrets                   | Bound to MCP Artifacts through secret bindings                 |
| Settings auth keys            | Removed                                                        |
| Routine built-in update       | Overlay survives when Artifact identity survives               |
| Destructive topology reset    | Overlays and bindings are deleted and secret cleanup is queued |
| SQLite                        | Remains the default adapter with narrow new repository ports   |
| Sources                       | Remain unchanged and source-agnostic                           |
| MapStore                      | Used only for encrypted secret values                          |

This design gives Artifact Store ownership of protected built-in local state and secret references without replacing the existing Artifact Store architecture or duplicating the existing mutable `Artifact.Data` mechanism.
